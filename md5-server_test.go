package md5simd

import (
	"fmt"
	"hash"
	"testing"
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
