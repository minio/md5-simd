package md5simd

import (
	"encoding/hex"
	_ "fmt"
	"strings"
	"testing"
	"unsafe"
)

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func TestBlock16(t *testing.T) {

	input := [16][]byte{}

	gld, i := golden[8:], 0
	// fill initial test vectors from golden test vectors with length >= 64
	for g := range gld {
		if len(gld[g].in) >= 64 {
			input[i] = []byte(gld[g].in[:64])
			i++
			if i >= 8 /*len(input)*/ {
				break
			}
		}
	}
	// fill upper 8 test vectors with the reverse strings of lower
	for ; i < len(input); i++ {
		input[i] = []byte(Reverse(string(input[i-8])))
	}

	var s digest16
	for i := 0; i < 16; i++ {
		s.v0[i], s.v1[i], s.v2[i], s.v3[i] = init0, init1, init2, init3
	}

	bufs := [16]int32{4, 4 + MaxBlockSize, 4 + MaxBlockSize*2, 4 + MaxBlockSize*3, 4 + MaxBlockSize*4, 4 + MaxBlockSize*5, 4 + MaxBlockSize*6, 4 + MaxBlockSize*7,
		4 + MaxBlockSize*8, 4 + MaxBlockSize*9, 4 + MaxBlockSize*10, 4 + MaxBlockSize*11, 4 + MaxBlockSize*12, 4 + MaxBlockSize*13, 4 + MaxBlockSize*14, 4 + MaxBlockSize*15}

	base := make([]byte, 4+16*MaxBlockSize)

	for i := 0; i < len(input); i++ {
		copy(base[bufs[i]:], input[i])
	}

	zreg := [64 * 4]byte{}

	block16(&s.v0[0], uintptr(unsafe.Pointer(&(base[0]))), &bufs[0], 64, &zreg)

	want :=
		`00000000  82 3c 09 52 b9 77 11 2a  65 ee 4c 82 f9 ad 4d 28  |.<.R.w.*e.L...M(|
00000010  82 53 aa b9 4d 9c 94 07  93 c7 ce 70 9b 18 c9 a7  |.S..M......p....|
00000020  4a 95 90 aa 8f 8f f6 29  59 b3 95 9f 5f 3c b0 08  |J......)Y..._<..|
00000030  56 1d 88 66 26 d7 12 cc  e4 41 2f 07 1b 7c 1a 4d  |V..f&....A/..|.M|
00000040  ab 71 57 fc 43 2d ee a3  b5 a8 11 9a 3d e2 33 84  |.qW.C-......=.3.|
00000050  41 b0 a7 71 38 3e 16 e6  8c 23 80 fa f2 18 45 c3  |A..q8>...#....E.|
00000060  72 08 7e 17 a6 52 b7 a9  24 38 d1 44 f1 12 ec a2  |r.~..R..$8.D....|
00000070  bb 0a 2c c5 7a cc a2 49  bf 44 a2 1b 0f fe 08 49  |..,.z..I.D.....I|
00000080  9f 5d 41 c2 1b 45 75 aa  36 3a 05 f9 36 a9 14 18  |.]A..Eu.6:..6...|
00000090  e1 1c f8 67 52 f4 59 c8  de 2e c6 c1 24 f3 fd 82  |...gR.Y.....$...|
000000a0  7c 0d c0 7d 2a 1e f4 9e  60 f9 0e 11 b9 fd a5 79  ||..}*..........y|
000000b0  57 9d 20 80 cc f3 da 4e  ec 7b 5d 2b 71 86 1d e0  |W. ....N.{]+q...|
000000c0  06 db 9c fa 5d fa 1f 90  fc 1f f4 61 cc 2c 8e 3a  |....]......a.,.:|
000000d0  87 84 9f 50 39 78 ec 5b  01 a8 be fa 0a 0b 5f 9d  |...P9x.[......_.|
000000e0  75 e1 ce 30 97 4c 9e 87  6d b4 1c e8 ae 59 0f cd  |u..0.L..m....Y..|
000000f0  7e 4d a1 cf 85 2d 33 1d  4a a7 0f 36 26 9e fd 37  |~M...-3.J..6&..7|
`

	got := hex.Dump(zreg[:])
	got = strings.ReplaceAll(got, "`", ".")
	if got != want {
		t.Fatalf("got %s\n                    want %s", got, want)
	}
}
