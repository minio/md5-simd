// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"sort"
)

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

func generateMaskAndRounds8(input [8][]byte) (mr []maskRounds) {

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
