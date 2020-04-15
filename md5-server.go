//+build !noasm,!appengine

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"encoding/binary"
	"hash"
	"sync/atomic"
	"time"
)

const BlockSize = 64
const Size = 16
const chunk = BlockSize
const MaxBlockSize = 1024 * 1024 * 2

// Estimated sleep time for a chunk of MaxBlockSize based
// on 800 MB/sec hashing performance
const WriteSleepMs = 1000 * MaxBlockSize / (800 * 1024 * 1024)

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

// NewMd5 - initialize instance for Md5 implementation.
func NewMd5(md5srv *Md5Server) hash.Hash {
	uid := atomic.AddUint64(&uidCounter, 1)
	return &Md5Digest{uid: uid, md5srv: md5srv}
}

// Md5Server - Type to implement parallel handling of MD5 invocations
type Md5Server struct {
	blocksCh chan blockInput       // Input channel
	totalIn  int                   // Total number of inputs waiting to be processed
	lanes    [16]Md5LaneInfo       // Array with info per lane
	digests  map[uint64][Size]byte // Map of uids to (interim) digest results
	bases    [2][]byte             // base memory (only for non-AVX512 mode)
}

// NewMd5Server - Create new object for parallel processing handling
func NewMd5Server() *Md5Server {
	md5srv := &Md5Server{}
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
func (md5srv *Md5Server) Process() {
	for {
		select {
		case block := <-md5srv.blocksCh:

			// If reset message, reset and continue
			if block.reset {
				md5srv.reset(block.uid)
				continue
			}

			// Get slot
			index := block.uid % uint64(len(md5srv.lanes))

			if md5srv.lanes[index].block != nil {
				// If slot is already filled, process all inputs,
				// including most probably previous block for same hash
				md5srv.blocks()
			}
			md5srv.totalIn++
			md5srv.lanes[index] = Md5LaneInfo{uid: block.uid, block: block.msg}
			if block.final {
				md5srv.lanes[index].outputCh = block.sumCh
			}
			if md5srv.totalIn == len(md5srv.lanes) {
				// if all lanes are filled, process all lanes
				md5srv.blocks()
			}

		case <-time.After(10 * time.Microsecond):
			for _, lane := range md5srv.lanes {
				if lane.block != nil { // check if there is any input to process
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

func (md5srv *Md5Server) getDigests() (s digest16) {
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
