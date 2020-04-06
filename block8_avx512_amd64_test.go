package md5simd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	"unsafe"
)

func TestBlockAvx512(t *testing.T) {

	input := [16][]byte{}

	for i := range input {
		input[i] = bytes.Repeat([]byte{0x61 + byte(i*1)}, 64)
	}

	var s digest16

	// TODO: See if we can eliminate cache altogether (with 32 ZMM registers)
	var cache cache16 // stack storage for block16 tmp state

	bufs := [16]int32{4, 4 + MaxBlockSize, 4 + MaxBlockSize*2, 4 + MaxBlockSize*3, 4 + MaxBlockSize*4, 4 + MaxBlockSize*5, 4 + MaxBlockSize*6, 4 + MaxBlockSize*7,
		4 + MaxBlockSize*8, 4 + MaxBlockSize*9, 4 + MaxBlockSize*10, 4 + MaxBlockSize*11, 4 + MaxBlockSize*12, 4 + MaxBlockSize*13, 4 + MaxBlockSize*14, 4 + MaxBlockSize*15}

	base := make([]byte, 4+16*MaxBlockSize)

	for i := 0; i < len(input); i++ {
		copy(base[bufs[i]:], input[i])
	}

	zreg := [64]byte{}

	block8Avx512(&s.v0[0], uintptr(unsafe.Pointer(&(base[0]))), &bufs[0], &cache[0], 64, &zreg)

	fmt.Println(hex.Dump(zreg[:]))
}

