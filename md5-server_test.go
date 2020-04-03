package md5simd

import (
	"fmt"
	"hash"
	"testing"
	"bytes"
	"reflect"
	"crypto/md5"
	"encoding/binary"
)

type md5Test struct {
	in  string
	want string
}

var golden = []md5Test{
	{ "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "014842d480b571495a4a0363793f7367" },
	{ "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "0b649bcb5a82868817fec9a6e709d233" },
	{ "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "bcd5708ed79b18f0f0aaa27fd0056d86" },
	{ "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "e987c862fbd2f2f0ca859cb8d7806bf3" },
	{ "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "982731671f0cd82cafce8d96a98e7a48" },
	{ "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "baf13e8b16d8c06324d7c9ab32cb7ff0" },
	{ "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", "8ea3109cbd951bba1ace2f401a784ae4" },
	{ "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh", "d141045bfb385cad357e7c39c60e5da0" },
}

func TestGolden(t *testing.T) {

	server := NewMd5Server()
	h8 := [8]hash.Hash{}
	for i := range h8 {
		h8[i] = NewMd5(server)
	}

	for i := range h8 {
		h8[i].Write([]byte(golden[i].in))
	}

	for i := range h8 {
		digest := h8[i].Sum([]byte{})
		if fmt.Sprintf("%x", digest) != golden[i].want {
			t.Errorf("TestGolden[%d], got %v, want %v", i, fmt.Sprintf("%x", digest), golden[i].want)
		}
	}
}

func testGolden(t *testing.T, megabyte int) {

	server := NewMd5Server()
	h8 := [8]hash.Hash{}
	input := [8][]byte{}
	for i := range h8 {
		h8[i] = NewMd5(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, megabyte*1024*1024)
	}

	for i := range h8 {
		h8[i].Write(input[i])
	}

	for i := range h8 {
		digest := h8[i].Sum([]byte{})
		got := fmt.Sprintf("%x\n", digest)

		h := md5.New()
		h.Write(input[i])
		want := fmt.Sprintf("%x\n", h.Sum(nil))

		if got != want {
			t.Errorf("TestGolden[%d], got %v, want %v", i, got, want)
		}
	}
}

func TestGolden(t *testing.T) {
	t.Run("1MB", func(t *testing.T) {
		testGolden(t, 1)
	})
	t.Run("2MB", func(t *testing.T) {
		testGolden(t, 2)
	})
}

func benchmarkGolden(b *testing.B, blockSize int) {

	server := NewMd5Server()
	h8 := [8]hash.Hash{}
	input := [8][]byte{}
	for i := range h8 {
		h8[i] = NewMd5(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	b.SetBytes(int64(blockSize*8))
	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for i := range h8 {
			h8[i].Write(input[i])
		}
	}
}

func BenchmarkGolden(b *testing.B) {
	b.Run("32KB", func(b *testing.B) {
		benchmarkGolden(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkGolden(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkGolden(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkGolden(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkGolden(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkGolden(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkGolden(b, 2*1024*1024)
	})
}

type maskTest struct {
	in  [8]int
	out [8]maskRounds
}

var goldenMask = []maskTest{
	{[8]int{0, 0, 0, 0, 0, 0, 0, 0}, [8]maskRounds{}},
	{[8]int{64, 0, 64, 0, 64, 0, 64, 0}, [8]maskRounds{{0x55, 1}}},
	{[8]int{0, 64, 0, 64, 0, 64, 0, 64}, [8]maskRounds{{0xaa, 1}}},
	{[8]int{64, 64, 64, 64, 64, 64, 64, 64}, [8]maskRounds{{0xff, 1}}},
	{[8]int{128, 128, 128, 128, 128, 128, 128, 128}, [8]maskRounds{{0xff, 2}}},
	{[8]int{64, 128, 64, 128, 64, 128, 64, 128}, [8]maskRounds{{0xff, 1}, {0xaa, 1}}},
	{[8]int{128, 64, 128, 64, 128, 64, 128, 64}, [8]maskRounds{{0xff, 1}, {0x55, 1}}},
	{[8]int{64, 192, 64, 192, 64, 192, 64, 192}, [8]maskRounds{{0xff, 1}, {0xaa, 2}}},
	{[8]int{0, 64, 128, 0, 64, 128, 0, 64}, [8]maskRounds{{0xb6, 1}, {0x24, 1}}},
	{[8]int{1 * 64, 2 * 64, 3 * 64, 4 * 64, 5 * 64, 6 * 64, 7 * 64, 8 * 64},
		[8]maskRounds{{0xff, 1}, {0xfe, 1}, {0xfc, 1}, {0xf8, 1}, {0xf0, 1}, {0xe0, 1}, {0xc0, 1}, {0x80, 1}}},
	{[8]int{2 * 64, 1 * 64, 3 * 64, 4 * 64, 5 * 64, 6 * 64, 7 * 64, 8 * 64},
		[8]maskRounds{{0xff, 1}, {0xfd, 1}, {0xfc, 1}, {0xf8, 1}, {0xf0, 1}, {0xe0, 1}, {0xc0, 1}, {0x80, 1}}},
	{[8]int{10 * 64, 20 * 64, 30 * 64, 40 * 64, 50 * 64, 60 * 64, 70 * 64, 80 * 64},
		[8]maskRounds{{0xff, 10}, {0xfe, 10}, {0xfc, 10}, {0xf8, 10}, {0xf0, 10}, {0xe0, 10}, {0xc0, 10}, {0x80, 10}}},
	{[8]int{10 * 64, 19 * 64, 27 * 64, 34 * 64, 40 * 64, 45 * 64, 49 * 64, 52 * 64},
		[8]maskRounds{{0xff, 10}, {0xfe, 9}, {0xfc, 8}, {0xf8, 7}, {0xf0, 6}, {0xe0, 5}, {0xc0, 4}, {0x80, 3}}},
}

func TestGenerateMaskAndRounds(t *testing.T) {
	input := [8][]byte{}
	for gcase, g := range goldenMask {
		for i, l := range g.in {
			buf := make([]byte, l)
			input[i] = buf[:]
		}

		mr := generateMaskAndRounds(input)

		if !reflect.DeepEqual(mr, g.out) {
			t.Fatalf("case %d: got %04x\n                    want %04x", gcase, mr, g.out)
		}
	}
}

func TestBlocks(t *testing.T) {

	inputs := [8][]byte{}
	want := [8]string{}
	for i := range inputs {
		inputs[i] = bytes.Repeat([]byte{0x61 + byte(i)}, (i+1)*64)

		{
			var d digest
			d.s[0], d.s[1], d.s[2], d.s[3] = init0, init1, init2, init3

			blockGeneric(&d, inputs[i])

			var digest [Size]byte
			binary.LittleEndian.PutUint32(digest[0:], d.s[0])
			binary.LittleEndian.PutUint32(digest[4:], d.s[1])
			binary.LittleEndian.PutUint32(digest[8:], d.s[2])
			binary.LittleEndian.PutUint32(digest[12:], d.s[3])

			want[i] = fmt.Sprintf("%x", digest)
			//fmt.Println(want[i])
		}
	}

	var s digest8

	for i := 0; i < 8; i++ {
		s.v0[i], s.v1[i], s.v2[i], s.v3[i] = init0, init1, init2, init3
	}

	base := make([]byte, 4+8*MaxBlockSize)

	blockMd5(&s, inputs,base)

	for i := 0; i < 8; i++ {
		var digest [Size]byte
		binary.LittleEndian.PutUint32(digest[0:], s.v0[i])
		binary.LittleEndian.PutUint32(digest[4:], s.v1[i])
		binary.LittleEndian.PutUint32(digest[8:], s.v2[i])
		binary.LittleEndian.PutUint32(digest[12:], s.v3[i])

		got := fmt.Sprintf("%x", digest)
		if got != want[i] {
			t.Errorf("TestBlocks[%d], got %v, want %v", i, got, want[i])
		}
	}
}