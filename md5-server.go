//+build !noasm,!appengine

/*
 * Minio Cloud Storage, (C) 2020 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package md5simd

import (
	"fmt"
	"math/bits"
	"sort"
	"sync/atomic"
	"unsafe"
)

const BlockSize = 64
const Size = 16
const chunk = BlockSize
const MaxBlockSize = 1024*1024*2

// Estimated sleep time for a chunk of MaxBlockSize based
// on 800 MB/sec hashing performance
const WriteSleepMs = 1000 * MaxBlockSize / (800*1024*1024)

// MD5 initialization constants
const (
	init0 = 0x67452301
	init1 = 0xefcdab89
	init2 = 0x98badcfe
	init3 = 0x10325476
)

// Md5ServerUID - Do not start at 0 but next multiple of 8 so as to be able to
// differentiate with default initialisation value of 0
const Md5ServerUID = 8
var uidCounter uint64 = 8 - 1

var used_8 = uint64(0)
var unused_8 = uint64(0)
var capacity_8 = uint64(0)

// Interface function to assembly code
func blockMd5(s *digest8, input [8][]byte, base []byte) {

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

	maskRounds := generateMaskAndRounds(input)

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

// Message to send across input channel
type blockInput struct {
	uid   uint64
	msg   []byte
	reset bool
	final bool
	sumCh chan [Size]byte
}

// Md5LaneInfo - Info for each lane
type Md5LaneInfo struct {
	uid      uint64          // unique identification for this MD5 processing
	block    []byte          // input block to be processed
	outputCh chan [Size]byte // channel for output result
}

// Helper struct for sorting blocks based on length
type lane struct {
	len uint
	pos uint
}

type lanes []lane

func (lns lanes) Len() int           { return len(lns) }
func (lns lanes) Swap(i, j int)      { lns[i], lns[j] = lns[j], lns[i] }
func (lns lanes) Less(i, j int) bool { return lns[i].len < lns[j].len }

// Helper struct for
type maskRounds struct {
	mask   uint64
	rounds uint64
}

func generateMaskAndRounds(input [8][]byte) (mr []maskRounds) {

	// Sort on blocks length small to large
	var sorted [8]lane
	for c, inpt := range input {
		sorted[c] = lane{uint(len(inpt)), uint(c)}
	}
	sort.Sort(lanes(sorted[:]))

	// Create mask array including 'rounds' (of processing blocks of 64 bytes) between masks
	m, round := uint64(0xff), uint64(0)
	mr = make([]maskRounds, 0, 8)
	for _, s := range sorted {
		if s.len > 0 {
			if uint64(s.len)>>6 > round {
				mr = append(mr, maskRounds{m, (uint64(s.len) >> 6) - round})
			}
			round = uint64(s.len) >> 6
		}
		m = m & ^(1 << uint(s.pos))
	}

	return
}
