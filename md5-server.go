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
	"encoding/binary"
	"errors"
	"hash"
	"math/bits"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"
	"fmt"
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

// NewMd5 - initialize parallel Md5 implementation.
func NewMd5(md5srv *Md5Server) hash.Hash {
	uid := atomic.AddUint64(&uidCounter, 1)
	return &Md5Digest{uid: uid, md5srv: md5srv}
}

// Md5Digest - Type for computing MD5 using AVX2
type Md5Digest struct {
	uid     uint64
	md5srv  *Md5Server
	x       [chunk]byte
	nx      int
	len     uint64
	final   bool
	result  [Size]byte
}

// Size - Return size of checksum
func (d *Md5Digest) Size() int { return Size }

// BlockSize - Return blocksize of checksum
func (d Md5Digest) BlockSize() int { return BlockSize }

// Reset - reset digest to its initial values
func (d *Md5Digest) Reset() {
	d.md5srv.blocksCh <- blockInput{uid: d.uid, reset: true}
	d.nx = 0
	d.len = 0
	d.final = false
}

// Write to digest
func (d *Md5Digest) Write(p []byte) (nn int, err error) {
	// break input into chunks of maximum MaxBlockSize size
	for {
		l := len(p)
		if l > MaxBlockSize {
			l = MaxBlockSize
		}
		nnn, err := d.write(p[:l])
		if err != nil {
			return nn, err
		}
		nn += nnn
		p = p[l:]

		if len(p) == 0 {
			break
		}

		time.Sleep(WriteSleepMs * time.Millisecond)
	}
	return
}

func (d *Md5Digest) write(p []byte) (nn int, err error) {

	if d.final {
		return 0, errors.New("Md5Digest already finalized. Reset first before writing again")
	}

	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == chunk {
			d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: d.x[:]}
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= chunk {
		n := len(p) &^ (chunk - 1)
		d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: p[:n]}
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

// Sum - Return MD5 sum in bytes
func (d *Md5Digest) Sum(in []byte) (result []byte) {

	if d.final {
		return append(in, d.result[:]...)
	}

	trail := make([]byte, 0, 128)
	trail = append(trail, d.x[:d.nx]...)

	len := d.len
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		trail = append(trail, tmp[0:56-len%64]...)
	} else {
		trail = append(trail, tmp[0:64+56-len%64]...)
	}
	d.nx = 0

	// Length in bits.
	len <<= 3
	binary.LittleEndian.PutUint64(tmp[:], len) // append length in bits
	trail = append(trail, tmp[0:8]...)

	sumCh := make(chan [Size]byte)
	d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: trail, final: true, sumCh: sumCh}
	d.result = <-sumCh
	d.final = true
	return append(in, d.result[:]...)
}

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

// Md5Server - Type to implement parallel handling of MD5 invocations
type Md5Server struct {
	blocksCh chan blockInput       // Input channel
	totalIn  int                   // Total number of inputs waiting to be processed
	lanes    [8]Md5LaneInfo        // Array with info per lane (out of 8)
	digests  map[uint64][Size]byte // Map of uids to (interim) digest results
	base     []byte				   // Buffer for merging different streams into
}

// Md5LaneInfo - Info for each lane
type Md5LaneInfo struct {
	uid      uint64          // unique identification for this MD5 processing
	block    []byte          // input block to be processed
	outputCh chan [Size]byte // channel for output result
}

// NewMd5Server - Create new object for parallel processing handling
func NewMd5Server() *Md5Server {
	md5srv := &Md5Server{}
	md5srv.digests = make(map[uint64][Size]byte)
	md5srv.blocksCh = make(chan blockInput)
	md5srv.base = make([]byte, 4+8*MaxBlockSize)

	// Start a single thread for reading from the input channel
	go md5srv.Process()
	return md5srv
}

// Process - Sole handler for reading from the input channel
func (md5srv *Md5Server) Process() {
	for {
		select {
		case block := <-md5srv.blocksCh:
			if block.reset {
				md5srv.reset(block.uid)
				continue
			}
			index := block.uid % uint64(len(md5srv.lanes))
			// fmt.Println("Adding message:", block.uid, index)

			if md5srv.lanes[index].block != nil { // If slot is already filled, process all inputs
				//fmt.Println("Invoking Blocks()")
				md5srv.blocks()
			}
			md5srv.totalIn++
			md5srv.lanes[index] = Md5LaneInfo{uid: block.uid, block: block.msg}
			if block.final {
				md5srv.lanes[index].outputCh = block.sumCh
			}
			if md5srv.totalIn == len(md5srv.lanes) {
				// fmt.Println("Invoking Blocks() while FULL: ")
				md5srv.blocks()
			}

			// TODO: test with larger timeout
		case <-time.After(1 * time.Microsecond):
			for _, lane := range md5srv.lanes {
				if lane.block != nil { // check if there is any input to process
					// fmt.Println("Invoking Blocks() on TIMEOUT: ")
					md5srv.blocks()
					break // we are done
				}
			}
		}
	}
}

// Do a reset for this calculation
func (md5srv *Md5Server) reset(uid uint64) {

	// Check if there is a message still waiting to be processed (and remove if so)
	for i, lane := range md5srv.lanes {
		if lane.uid == uid {
			if lane.block != nil {
				md5srv.lanes[i] = Md5LaneInfo{} // clear message
				md5srv.totalIn--
			}
		}
	}

	// Delete entry from hash map
	delete(md5srv.digests, uid)
}

// Invoke assembly and send results back
func (md5srv *Md5Server) blocks() {

	inputs := [8][]byte{}
	for i := range inputs {
		inputs[i] = md5srv.lanes[i].block
	}

	state := md5srv.getDigests()
	blockMd5(&state, inputs, md5srv.base)

	md5srv.totalIn = 0
	for i := 0; i < len(md5srv.lanes); i++ {
		uid, outputCh := md5srv.lanes[i].uid, md5srv.lanes[i].outputCh
		digest := [Size]byte{}
		binary.LittleEndian.PutUint32(digest[0:], state.v0[i])
		binary.LittleEndian.PutUint32(digest[4:], state.v1[i])
		binary.LittleEndian.PutUint32(digest[8:], state.v2[i])
		binary.LittleEndian.PutUint32(digest[12:], state.v3[i])
		md5srv.digests[uid] = digest
		md5srv.lanes[i] = Md5LaneInfo{}

		if outputCh != nil {
			// Send back result
			outputCh <- digest
			delete(md5srv.digests, uid) // Delete entry from hashmap
		}
	}
}

func (md5srv *Md5Server) Write(uid uint64, p []byte) (nn int, err error) {
	md5srv.blocksCh <- blockInput{uid: uid, msg: p}
	return len(p), nil
}

// Sum - return sha256 sum in bytes for a given sum id.
func (md5srv *Md5Server) Sum(uid uint64, p []byte) [16]byte {
	sumCh := make(chan [16]byte)
	md5srv.blocksCh <- blockInput{uid: uid, msg: p, final: true, sumCh: sumCh}
	return <-sumCh
}

func (md5srv *Md5Server) getDigests() (s digest8) {
	for i, lane := range md5srv.lanes {
		a, ok := md5srv.digests[lane.uid]
		if ok {
			s.v0[i] = binary.LittleEndian.Uint32(a[0:4])
			s.v1[i] = binary.LittleEndian.Uint32(a[4:8])
			s.v2[i] = binary.LittleEndian.Uint32(a[8:12])
			s.v3[i] = binary.LittleEndian.Uint32(a[12:16])
		} else {
			s.v0[i] = init0
			s.v1[i] = init1
			s.v2[i] = init2
			s.v3[i] = init3
		}
	}
	return
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
