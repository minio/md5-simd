//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"encoding/binary"
	"time"

	"github.com/klauspost/cpuid"
)

// Estimated sleep time for a chunk of MaxBlockSize based
// on 800 MB/sec hashing performance
const writeSleepMs = 1000 * MaxBlockSize / (800 * 1024 * 1024)

// MD5 initialization constants
const (
	init0 = 0x67452301
	init1 = 0xefcdab89
	init2 = 0x98badcfe
	init3 = 0x10325476
)

// md5ServerUID - Does not start at 0 but next multiple of 16 so as to be able to
// differentiate with default initialisation value of 0
const md5ServerUID = 16

// Message to send across input channel
type blockInput struct {
	uid   uint64
	msg   []byte
	reset bool
	final bool
	sumCh chan [Size]byte
}

// md5LaneInfo - Info for each lane
type md5LaneInfo struct {
	uid      uint64          // unique identification for this MD5 processing
	block    []byte          // input block to be processed
	outputCh chan [Size]byte // channel for output result
}

// md5Server - Type to implement parallel handling of MD5 invocations
type md5Server struct {
	uidCounter uint64
	blocksCh   chan blockInput       // Input channel
	totalIn    int                   // Total number of inputs waiting to be processed
	lanes      [16]md5LaneInfo       // Array with info per lane
	digests    map[uint64][Size]byte // Map of uids to (interim) digest results
	bases      [2][]byte             // base memory (only for non-AVX512 mode)
}

// NewServer - Create new object for parallel processing handling
func NewServer() Server {
	if !cpuid.CPU.AVX2() {
		return &fallbackServer{}
	}
	md5srv := &md5Server{}
	md5srv.digests = make(map[uint64][Size]byte)
	md5srv.blocksCh = make(chan blockInput)
	md5srv.uidCounter = md5ServerUID - 1
	if !hasAVX512 {
		// only reserve memory when not on AVX512
		md5srv.bases[0] = make([]byte, 4+8*MaxBlockSize)
		md5srv.bases[1] = make([]byte, 4+8*MaxBlockSize)
	}

	// Start a single thread for reading from the input channel
	go md5srv.process(md5srv.blocksCh)
	return md5srv
}

// process - Sole handler for reading from the input channel
func (s *md5Server) process(blocksCh chan blockInput) {
	processBlock := func(block blockInput) {
		// If reset message, reset and we're done
		if block.reset {
			s.reset(block.uid)
			return
		}

		// Get slot
		index := block.uid % uint64(len(s.lanes))

		if s.lanes[index].block != nil {
			// If slot is already filled, process all inputs,
			// including most probably previous block for same hash
			s.blocks()
		}

		// Intercept final messages that are small and process synchronously
		if block.final && len(block.msg) <= 128 {

			var dig digest
			d, ok := s.digests[block.uid]
			if ok {
				dig.s[0] = binary.LittleEndian.Uint32(d[0:4])
				dig.s[1] = binary.LittleEndian.Uint32(d[4:8])
				dig.s[2] = binary.LittleEndian.Uint32(d[8:12])
				dig.s[3] = binary.LittleEndian.Uint32(d[12:16])
			} else {
				dig.s[0], dig.s[1], dig.s[2], dig.s[3] = init0, init1, init2, init3
			}

			blockGeneric(&dig, block.msg)

			sum := [Size]byte{}
			binary.LittleEndian.PutUint32(sum[0:], dig.s[0])
			binary.LittleEndian.PutUint32(sum[4:], dig.s[1])
			binary.LittleEndian.PutUint32(sum[8:], dig.s[2])
			binary.LittleEndian.PutUint32(sum[12:], dig.s[3])

			block.sumCh <- sum
			return
		}

		s.totalIn++
		s.lanes[index] = md5LaneInfo{uid: block.uid, block: block.msg}
		if block.final {
			s.lanes[index].outputCh = block.sumCh
		}
		if s.totalIn == len(s.lanes) {
			// if all lanes are filled, process all lanes
			s.blocks()
		}
	}

	for {
		select {
		case block, ok := <-blocksCh:
			if !ok {
				return
			}
			processBlock(block)
		}

		for busy := true; busy; {
			select {
			case block, ok := <-blocksCh:
				if !ok {
					return
				}
				processBlock(block)

			case <-time.After(10 * time.Microsecond):
				l, lane := 0, md5LaneInfo{}
				for l, lane = range s.lanes {
					if lane.block != nil { // check if there is any input to process
						s.blocks()
						break // we are done
					}
				}
				if l == len(s.lanes) { // no work to do, so exit this loop and go back to single select
					busy = false
				}
			}
		}
	}
}

func (s *md5Server) Close() {
	if s.blocksCh != nil {
		close(s.blocksCh)
		s.blocksCh = nil
	}
}

// Do a reset for this calculation
func (s *md5Server) reset(uid uint64) {
	// Check if there is a message still waiting to be processed (and remove if so)
	for i, lane := range s.lanes {
		if lane.uid == uid {
			if lane.block != nil {
				s.lanes[i] = md5LaneInfo{} // clear message
				s.totalIn--
			}
		}
	}

	// Delete entry from hash map
	delete(s.digests, uid)
}

// Invoke assembly and send results back
func (s *md5Server) blocks() {

	inputs := [16][]byte{}
	for i := range inputs {
		inputs[i] = s.lanes[i].block
	}

	state := s.getDigests()
	blockMd5_x16(&state, inputs, s.bases)

	s.totalIn = 0
	for i := 0; i < len(s.lanes); i++ {
		uid, outputCh := s.lanes[i].uid, s.lanes[i].outputCh
		digest := [Size]byte{}
		binary.LittleEndian.PutUint32(digest[0:], state.v0[i])
		binary.LittleEndian.PutUint32(digest[4:], state.v1[i])
		binary.LittleEndian.PutUint32(digest[8:], state.v2[i])
		binary.LittleEndian.PutUint32(digest[12:], state.v3[i])

		if outputCh == nil {
			s.digests[uid] = digest // save updated digest for next iteration
		} else {
			outputCh <- digest // send back result of padded trailer (and keep previous state for subsequent writes)
		}
		s.lanes[i] = md5LaneInfo{}
	}
}

func (s *md5Server) getDigests() (d digest16) {
	for i, lane := range s.lanes {
		a, ok := s.digests[lane.uid]
		if ok {
			d.v0[i] = binary.LittleEndian.Uint32(a[0:4])
			d.v1[i] = binary.LittleEndian.Uint32(a[4:8])
			d.v2[i] = binary.LittleEndian.Uint32(a[8:12])
			d.v3[i] = binary.LittleEndian.Uint32(a[12:16])
		} else {
			d.v0[i] = init0
			d.v1[i] = init1
			d.v2[i] = init2
			d.v3[i] = init3
		}
	}
	return
}

/*
func (s *fallbackServer) write(uid uint64, p []byte) (nn int, err error) {
	s.blocksCh <- blockInput{uid: uid, msg: p}
	return len(p), nil
}

// sum - return sha256 sum in bytes for a given sum id.
func (s *fallbackServer) sum(uid uint64, p []byte) [16]byte {
	sumCh := make(chan [16]byte)
	s.blocksCh <- blockInput{uid: uid, msg: p, final: true, sumCh: sumCh}
	return <-sumCh
}

*/
