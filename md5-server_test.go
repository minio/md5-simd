package md5simd

import (
	"fmt"
	"testing"
)

type md5Test struct {
	out [16]byte
	in  string
}

var golden = []md5Test{
	{ [...]byte{0,1,2,3,4,5,6,7,8,9,0xa,0xb,0xc,0xd,0xe,0xf}, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
}

func TestGolden(t *testing.T) {

	server := NewMd5Server()
	h512_0 := NewMd5(server)
	h512_1 := NewMd5(server)
	h512_2 := NewMd5(server)
	h512_3 := NewMd5(server)
	h512_4 := NewMd5(server)
	h512_5 := NewMd5(server)
	h512_6 := NewMd5(server)
	h512_7 := NewMd5(server)

	h512_0.Write([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	h512_1.Write([]byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	h512_2.Write([]byte("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"))
	h512_3.Write([]byte("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"))
	h512_4.Write([]byte("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"))
	h512_5.Write([]byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	h512_6.Write([]byte("gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"))
	h512_7.Write([]byte("hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"))

	digest_0 := h512_0.Sum([]byte{})

	fmt.Println(digest_0)

	//for _, g := range golden {
	//	h512.Reset()
	//	// h512.Write([]byte(g.in))
	//	digest := h512.Sum([]byte(g.in))
	//	s := fmt.Sprintf("%x", digest)
	//	fmt.Println("md5 =", s)
	//	//if !reflect.DeepEqual(digest, g.out[:]) {
	//	//	t.Fatalf("Sum256 function: sha256(%s) = %s want %s", g.in, s, hex.EncodeToString(g.out[:]))
	//	//}
	//}
}
