package md5simd

import (
	"encoding/binary"
	"fmt"
	"hash"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// NewMd5_x16 - initialize parallel Md5 implementation.
func NewMd5_x16(md5srv *Md5Server16) hash.Hash {
	uid := atomic.AddUint64(&uidCounter, 1)
	return &Md5Digest16{uid: uid, md5srv: md5srv}
}

// Interface function to assembly code
func blockMd5_x16(s *digest16, input [16][]byte, bases [2][]byte) {
	if hasAVX512 {
		blockMd5_x16_internal(s, input)
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
		go func() { blockMd5(&s8a, i8[0], bases[0]); wg.Done() }()
		go func() { blockMd5(&s8b, i8[1], bases[1]); wg.Done() }()
		wg.Wait()

		for i := range s8a.v0 {
			j := i + 8
			s.v0[i], s.v1[i], s.v2[i], s.v3[i] = s8a.v0[i], s8a.v1[i], s8a.v2[i], s8a.v3[i]
			s.v0[j], s.v1[j], s.v2[j], s.v3[j] = s8b.v0[i], s8b.v1[i], s8b.v2[i], s8b.v3[i]
		}
	}
}

// Interface function to assembly code
func blockMd5_x16_internal(s *digest16, input [16][]byte) {

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

// Md5Server16 - Type to implement parallel handling of MD5 invocations
type Md5Server16 struct {
	blocksCh chan blockInput       // Input channel
	totalIn  int                   // Total number of inputs waiting to be processed
	lanes    [16]Md5LaneInfo       // Array with info per lane
	digests  map[uint64][Size]byte // Map of uids to (interim) digest results
	bases    [2][]byte			   // base memory (only for non-AVX512 mode)
}

// NewMd5Server16 - Create new object for parallel processing handling
func NewMd5Server16() *Md5Server16 {
	md5srv := &Md5Server16{}
	md5srv.digests = make(map[uint64][Size]byte)
	md5srv.blocksCh = make(chan blockInput)
	if !hasAVX512 {
		// only reserve memory when not on AVX512
		md5srv.bases[0] = make([]byte, 4+8*MaxBlockSize)
		md5srv.bases[1] = make([]byte, 4+8*MaxBlockSize)
	}

	// Start a single thread for reading from the input channel
	go md5srv.Process()
	return md5srv
}

// Process - Sole handler for reading from the input channel
func (md5srv *Md5Server16) Process() {
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
func (md5srv *Md5Server16) reset(uid uint64) {

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
func (md5srv *Md5Server16) blocks() {

	inputs := [16][]byte{}
	for i := range inputs {
		inputs[i] = md5srv.lanes[i].block
	}

	state := md5srv.getDigests()
	blockMd5_x16(&state, inputs, md5srv.bases)

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

func (md5srv *Md5Server16) Write(uid uint64, p []byte) (nn int, err error) {
	md5srv.blocksCh <- blockInput{uid: uid, msg: p}
	return len(p), nil
}

// Sum - return sha256 sum in bytes for a given sum id.
func (md5srv *Md5Server16) Sum(uid uint64, p []byte) [16]byte {
	sumCh := make(chan [16]byte)
	md5srv.blocksCh <- blockInput{uid: uid, msg: p, final: true, sumCh: sumCh}
	return <-sumCh
}

func (md5srv *Md5Server16) getDigests() (s digest16) {
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

func generateMaskAndRounds16(input [16][]byte) (mr []maskRounds) {

	// Sort on blocks length small to large
	var sorted [16]lane
	for c, inpt := range input {
		sorted[c] = lane{uint(len(inpt)), uint(c)}
	}
	sort.Sort(lanes(sorted[:]))

	// Create mask array including 'rounds' (of processing blocks of 64 bytes) between masks
	m, round := uint64(0xffff), uint64(0)
	mr = make([]maskRounds, 0, 16)
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
