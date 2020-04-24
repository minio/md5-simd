//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"reflect"
	"testing"
)

type maskTest struct {
	in  [8]int
	out []maskRounds
}

var goldenMask = []maskTest{
	{[8]int{0, 0, 0, 0, 0, 0, 0, 0}, []maskRounds{}},
	{[8]int{64, 0, 64, 0, 64, 0, 64, 0}, []maskRounds{{0x55, 1}}},
	{[8]int{0, 64, 0, 64, 0, 64, 0, 64}, []maskRounds{{0xaa, 1}}},
	{[8]int{64, 64, 64, 64, 64, 64, 64, 64}, []maskRounds{{0xff, 1}}},
	{[8]int{128, 128, 128, 128, 128, 128, 128, 128}, []maskRounds{{0xff, 2}}},
	{[8]int{64, 128, 64, 128, 64, 128, 64, 128}, []maskRounds{{0xff, 1}, {0xaa, 1}}},
	{[8]int{128, 64, 128, 64, 128, 64, 128, 64}, []maskRounds{{0xff, 1}, {0x55, 1}}},
	{[8]int{64, 192, 64, 192, 64, 192, 64, 192}, []maskRounds{{0xff, 1}, {0xaa, 2}}},
	{[8]int{0, 64, 128, 0, 64, 128, 0, 64}, []maskRounds{{0xb6, 1}, {0x24, 1}}},
	{[8]int{1 * 64, 2 * 64, 3 * 64, 4 * 64, 5 * 64, 6 * 64, 7 * 64, 8 * 64},
		[]maskRounds{{0xff, 1}, {0xfe, 1}, {0xfc, 1}, {0xf8, 1}, {0xf0, 1}, {0xe0, 1}, {0xc0, 1}, {0x80, 1}}},
	{[8]int{2 * 64, 1 * 64, 3 * 64, 4 * 64, 5 * 64, 6 * 64, 7 * 64, 8 * 64},
		[]maskRounds{{0xff, 1}, {0xfd, 1}, {0xfc, 1}, {0xf8, 1}, {0xf0, 1}, {0xe0, 1}, {0xc0, 1}, {0x80, 1}}},
	{[8]int{10 * 64, 20 * 64, 30 * 64, 40 * 64, 50 * 64, 60 * 64, 70 * 64, 80 * 64},
		[]maskRounds{{0xff, 10}, {0xfe, 10}, {0xfc, 10}, {0xf8, 10}, {0xf0, 10}, {0xe0, 10}, {0xc0, 10}, {0x80, 10}}},
	{[8]int{10 * 64, 19 * 64, 27 * 64, 34 * 64, 40 * 64, 45 * 64, 49 * 64, 52 * 64},
		[]maskRounds{{0xff, 10}, {0xfe, 9}, {0xfc, 8}, {0xf8, 7}, {0xf0, 6}, {0xe0, 5}, {0xc0, 4}, {0x80, 3}}},
}

func TestGenerateMaskAndRounds(t *testing.T) {
	input := [8][]byte{}
	maskRound := [8]maskRounds{}
	for gcase, g := range goldenMask {
		for i, l := range g.in {
			buf := make([]byte, l)
			input[i] = buf[:]
		}

		rounds := generateMaskAndRounds8(input, &maskRound)

		mr := make([]maskRounds, 0, 8)
		for r := 0; r < rounds; r++ {
			mr = append(mr, maskRound[r])
		}

		if !reflect.DeepEqual(mr, g.out) {
			t.Fatalf("case %d: got %04x\n                    want %04x", gcase, mr, g.out)
		}
	}
}
