// +build amd64

package md5simd

import (
	"fmt"
	"unsafe"
	"sync"
	"sync/atomic"
	"math/bits"
)

//go:noescape
func block8(state *uint32, base uintptr, bufs *int32, cache *byte, n int)

//go:noescape
func block16(state *uint32, ptrs *int64, mask uint64, n int)

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
			if len(input[i])> MaxBlockSize {
				panic(fmt.Sprintf("Sanity check fails for lane %d: maximum input length cannot exceed MaxBlockSize", i))
			}
		}
	}

	bufs := [8]int32{4, 4+MaxBlockSize, 4+MaxBlockSize*2, 4+MaxBlockSize*3, 4+MaxBlockSize*4, 4+MaxBlockSize*5, 4+MaxBlockSize*6, 4+MaxBlockSize*7}
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
			bufs[j] += int32(64*m.rounds) // update pointers for next round
			if m.mask & (1 << j) != 0 {	  // update digest if still masked as active
				(*s).v0[j], (*s).v1[j], (*s).v2[j], (*s).v3[j] = sdup.v0[j], sdup.v1[j], sdup.v2[j], sdup.v3[j]
			}
		}
	}
}
