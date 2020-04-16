// +build amd64

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"fmt"
	"github.com/klauspost/cpuid"
	"hash"
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

var hasAVX2 bool
var hasAVX512 bool

//go:noescape
func block8(state *uint32, base uintptr, bufs *int32, cache *byte, n int)

//go:noescape
func block16(state *uint32, ptrs *int64, mask uint64, n int)

// NewMd5 - initialize instance for Md5 implementation.
func NewMd5(md5srv *Md5Server) hash.Hash {
	uid := atomic.AddUint64(&uidCounter, 1)
	return &Md5Digest{uid: uid, md5srv: md5srv}
}

// 8-way 4x uint32 digests in 4 ymm registers
// (ymm0, ymm1, ymm2, ymm3)
type digest8 struct {
	v0, v1, v2, v3 [8]uint32
}

// Stack cache for 8x64 byte md5.BlockSize bytes.
// Must be 32-byte aligned, so allocate 512+32 and
// align upwards at runtime.
type cache8 [512 + 32]byte

// MD5 magic numbers for one lane of hashing; inflated
// 8x below at init time.
var md5consts = [64]uint32{
	0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee,
	0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501,
	0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
	0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821,
	0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa,
	0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
	0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed,
	0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a,
	0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
	0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70,
	0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05,
	0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
	0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039,
	0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1,
	0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
	0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
}

// inflate the consts 8-way for 8x md5 (256 bit ymm registers)
var avx256md5consts = func(c []uint32) []uint32 {
	inf := make([]uint32, 8*len(c))
	for i := range c {
		for j := 0; j < 8; j++ {
			inf[(i*8)+j] = c[i]
		}
	}
	return inf
}(md5consts[:])

// 16-way 4x uint32 digests in 4 zmm registers
type digest16 struct {
	v0, v1, v2, v3 [16]uint32
}

// inflate the consts 16-way for 16x md5 (512 bit zmm registers)
var avx512md5consts = func(c []uint32) []uint32 {
	inf := make([]uint32, 16*len(c))
	for i := range c {
		for j := 0; j < 16; j++ {
			inf[(i*16)+j] = c[i]
		}
	}
	return inf
}(md5consts[:])

func init() {
	hasAVX2 = cpuid.CPU.AVX2()
	hasAVX512 = cpuid.CPU.AVX512F()
}

// Interface function to assembly code
func blockMd5_x16(s *digest16, input [16][]byte, bases [2][]byte) {
	if hasAVX512 {
		blockMd5_avx512(s, input)
	} else {
		s8a, s8b := digest8{}, digest8{}
		for i := range s8a.v0 {
			j := i + 8
			s8a.v0[i], s8a.v1[i], s8a.v2[i], s8a.v3[i] = s.v0[i], s.v1[i], s.v2[i], s.v3[i]
			s8b.v0[i], s8b.v1[i], s8b.v2[i], s8b.v3[i] = s.v0[j], s.v1[j], s.v2[j], s.v3[j]
		}

		i8 := [2][8][]byte{}
		for i := range i8[0] {
			i8[0][i], i8[1][i] = input[i], input[8+i]
		}

		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() { blockMd5_avx2(&s8a, i8[0], bases[0]); wg.Done() }()
		go func() { blockMd5_avx2(&s8b, i8[1], bases[1]); wg.Done() }()
		wg.Wait()

		for i := range s8a.v0 {
			j := i + 8
			s.v0[i], s.v1[i], s.v2[i], s.v3[i] = s8a.v0[i], s8a.v1[i], s8a.v2[i], s8a.v3[i]
			s.v0[j], s.v1[j], s.v2[j], s.v3[j] = s8b.v0[i], s8b.v1[i], s8b.v2[i], s8b.v3[i]
		}
	}
}

// Interface function to AVX512 assembly code
func blockMd5_avx512(s *digest16, input [16][]byte) {

	// Sanity check to make sure we're not passing in more data than MaxBlockSize
	{
		for i := 1; i < len(input); i++ {
			if len(input[i]) > MaxBlockSize {
				panic(fmt.Sprintf("Sanity check fails for lane %d: maximum input length cannot exceed MaxBlockSize", i))
			}
		}
	}

	ptrs := [16]int64{}

	for i := range ptrs {
		if input[i] != nil {
			ptrs[i] = int64(uintptr(unsafe.Pointer(&(input[i][0]))))
		}
	}

	sdup := *s // create copy of initial states to receive intermediate updates

	maskRounds := generateMaskAndRounds16(input)

	for _, m := range maskRounds {

		block16(&sdup.v0[0], &ptrs[0], m.mask, int(64*m.rounds))

		for j := 0; j < len(ptrs); j++ {
			ptrs[j] += int64(64 * m.rounds) // update pointers for next round
			if m.mask&(1<<j) != 0 {         // update digest if still masked as active
				(*s).v0[j], (*s).v1[j], (*s).v2[j], (*s).v3[j] = sdup.v0[j], sdup.v1[j], sdup.v2[j], sdup.v3[j]
			}
		}
	}
}

// Interface function to AVX2 assembly code
func blockMd5_avx2(s *digest8, input [8][]byte, base []byte) {

	// Sanity check to make sure we're not passing in more data than MaxBlockSize
	{
		for i := 1; i < len(input); i++ {
			if len(input[i]) > MaxBlockSize {
				panic(fmt.Sprintf("Sanity check fails for lane %d: maximum input length cannot exceed MaxBlockSize", i))
			}
		}
	}

	bufs := [8]int32{4, 4 + MaxBlockSize, 4 + MaxBlockSize*2, 4 + MaxBlockSize*3, 4 + MaxBlockSize*4, 4 + MaxBlockSize*5, 4 + MaxBlockSize*6, 4 + MaxBlockSize*7}
	for i := 0; i < len(input); i++ {
		copy(base[bufs[i]:], input[i])
	}

	sdup := *s // create copy of initial states to receive intermediate updates

	maskRounds := generateMaskAndRounds8(input)

	for _, m := range maskRounds {
		var cache cache8 // stack storage for block8 tmp state
		block8(&sdup.v0[0], uintptr(unsafe.Pointer(&(base[0]))), &bufs[0], &cache[0], int(64*m.rounds))

		atomic.AddUint64(&used_8, uint64(bits.OnesCount(uint(m.mask)))*64*m.rounds)
		atomic.AddUint64(&unused_8, (8-uint64(bits.OnesCount(uint(m.mask))))*64*m.rounds)
		atomic.AddUint64(&capacity_8, 8*64*m.rounds)

		for j := 0; j < len(bufs); j++ {
			bufs[j] += int32(64 * m.rounds) // update pointers for next round
			if m.mask&(1<<j) != 0 {         // update digest if still masked as active
				(*s).v0[j], (*s).v1[j], (*s).v2[j], (*s).v3[j] = sdup.v0[j], sdup.v1[j], sdup.v2[j], sdup.v3[j]
			}
		}
	}
}
