package md5simd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"testing"
)

func TestGolden16(t *testing.T) {

	const megabyte = 1

	server := NewMd5Server16()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = NewMd5_x16(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, megabyte*1024*1024)
	}

	for i := range h16 {
		h16[i].Write(input[i])
	}

	for i := range h16 {
		digest := h16[i].Sum([]byte{})
		got := fmt.Sprintf("%x\n", digest)

		h := md5.New()
		h.Write(input[i])
		want := fmt.Sprintf("%x\n", h.Sum(nil))

		if got != want {
			t.Errorf("TestGolden16[%d], got %v, want %v", i, got, want)
		}
	}
}

