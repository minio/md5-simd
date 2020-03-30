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
	h512 := NewMd5(server)

	for _, g := range golden {
		h512.Reset()
		// h512.Write([]byte(g.in))
		digest := h512.Sum([]byte(g.in))
		s := fmt.Sprintf("%x", digest)
		fmt.Println("md5 =", s)
		//if !reflect.DeepEqual(digest, g.out[:]) {
		//	t.Fatalf("Sum256 function: sha256(%s) = %s want %s", g.in, s, hex.EncodeToString(g.out[:]))
		//}
	}
}
