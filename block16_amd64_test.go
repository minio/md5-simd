package md5simd

import (
	"bytes"
	"encoding/hex"
	_ "fmt"
	"strings"
	"testing"
	"unsafe"
)

func TestBlock16(t *testing.T) {

	input := [16][]byte{}

	for i := range input {
		input[i] = bytes.Repeat([]byte{0x61 + byte(i*1)}, 64)
	}

	var s digest16
	for i := 0; i < 16; i++ {
		s.v0[i], s.v1[i], s.v2[i], s.v3[i] = init0, init1, init2, init3
	}

	// TODO: See if we can eliminate cache altogether (with 32 ZMM registers)
	var cache cache16 // stack storage for block16 tmp state

	bufs := [16]int32{4, 4 + MaxBlockSize, 4 + MaxBlockSize*2, 4 + MaxBlockSize*3, 4 + MaxBlockSize*4, 4 + MaxBlockSize*5, 4 + MaxBlockSize*6, 4 + MaxBlockSize*7,
		4 + MaxBlockSize*8, 4 + MaxBlockSize*9, 4 + MaxBlockSize*10, 4 + MaxBlockSize*11, 4 + MaxBlockSize*12, 4 + MaxBlockSize*13, 4 + MaxBlockSize*14, 4 + MaxBlockSize*15}

	base := make([]byte, 4+16*MaxBlockSize)

	for i := 0; i < len(input); i++ {
		copy(base[bufs[i]:], input[i])
	}

	zreg := [64 * 4]byte{}

	block16(&s.v0[0], uintptr(unsafe.Pointer(&(base[0]))), &bufs[0], &cache[0], 64, &zreg)

	want :=
		`00000000  56 ff d4 89 9a 41 30 f2  71 99 67 b6 9c e5 d0 d2  |V....A0.q.g.....|
00000010  f1 8e 1e 44 71 dc 3f 4a  d3 84 88 69 3e b0 a1 53  |...Dq.?J...i>..S|
00000020  d0 fb 78 5b 61 29 05 29  04 cb ba 5d 81 e9 a6 78  |..x[a).)...]...x|
00000030  08 04 f9 19 b8 44 5c 67  0c 5a 7c 0b 1f c0 85 bf  |.....D\g.Z|.....|
00000040  62 d9 5c 12 4e fe 09 50  5a 70 67 57 d8 a3 1a 6f  |b.\.N..PZpgW...o|
00000050  56 8e eb af bb d0 45 66  ad a7 5b dc 23 3e d5 66  |V.....Ef..[.#>.f|
00000060  71 1d 01 63 93 a0 63 e2  3a 6b 69 e7 8a b7 2c 94  |q..c..c.:ki...,.|
00000070  1e 79 57 c8 61 89 60 92  11 5b ab 90 b5 17 ca 3e  |.yW.a....[.....>|
00000080  33 de ca 69 2f 85 c6 ba  c1 6e 29 16 88 df 8b 8b  |3..i/....n).....|
00000090  ae d8 00 6d a6 e6 d4 84  59 c7 f7 eb 26 61 dc af  |...m....Y...&a..|
000000a0  9a 79 e2 e3 1a 50 e4 1f  a2 54 e3 ce 0c d7 7f ca  |.y...P...T......|
000000b0  7d 1a 71 ac 94 14 25 dc  6a 6d 50 b6 2d 66 7b f4  |}.q...%.jmP.-f{.|
000000c0  25 e3 33 00 2f cc 31 e6  f2 a2 56 25 12 69 4c 9f  |%.3./.1...V%.iL.|
000000d0  84 17 92 91 44 6f ea d6  db b0 08 42 dd 4f 9c b3  |....Do.....B.O..|
000000e0  69 e1 de 49 d7 9d c1 22  21 8a b5 f0 4a 71 74 9c  |i..I..."!...Jqt.|
000000f0  06 d4 de f2 7e ac 6b ae  0b 2c 1d 18 1e 5f 03 67  |....~.k..,..._.g|
`

	got := hex.Dump(zreg[:])
	got = strings.ReplaceAll(got, "`", ".")
	if got != want {
		t.Fatalf("got %s\n                    want %s", got, want)
	}
}
