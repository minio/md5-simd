package md5simd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"reflect"
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
	{"", "d41d8cd98f00b204e9800998ecf8427e" },
	{"a","0cc175b9c0f1b6a831c399e269772661" },
	{"ab", "187ef4436122d1cc2f40dc2b92f0eba0" },
	{"abc", "900150983cd24fb0d6963f7d28e17f72" },
	{"abcd", "e2fc714c4727ee9395f324cd2e7f331f"},
	{"abcde", "ab56b4d92b40713acc5af89985d4b786" },
	{"abcdef", "e80b5017098950fc58aad83c8c14978e"},
	{"abcdefg", "7ac66c0f148de9519b8bd264312c4d64" },
	{"abcdefgh","e8dc4081b13434b45189a720b77b6818"},
	{"abcdefghi","8aa99b1f439ff71293e95357bac6fd94"},
	{"abcdefghij","a925576942e94b2ef57a066101b48876"},
	{"Discard medicine more than two years old.","d747fc1719c7eacb84058196cfe56d57"},
	{"He who has a shady past knows that nice guys finish last.", "bff2dcb37ef3a44ba43ab144768ca837"},
	{"I wouldn't marry him with a ten foot pole.", "0441015ecb54a7342d017ed1bcfdbea5"},
	{"Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave", "9e3cac8e9e9757a60c3ea391130d3689"},
	{"The days of the digital watch are numbered.  -Tom Stoppard", "a0f04459b031f916a59a35cc482dc039"},
	{"Nepal premier won't resign.", "e7a48e0fe884faf31475d2a04b1362cc"},
	{"For every action there is an equal and opposite government program.", "637d2fe925c07c113800509964fb0e06"},
	{"His money is twice tainted: 'taint yours and 'taint mine.", "834a8d18d5c6562119cf4c7f5086cb71" },
	{"There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977", "de3a4d2fd6c73ec2db2abad23b444281" },
	{"It's a tiny change to the code and not completely disgusting. - Bob Manchek", "acf203f997e2cf74ea3aff86985aefaf" },
	{"size:  a.out:  bad magic", "e1c1384cb4d2221dfdd7c795a4222c9a" },
	{"The major problem is with sendmail.  -Mark Horton","c90f3ddecc54f34228c063d7525bf644" },
	{"Give me a rock, paper and scissors and I will move the world.  CCFestoon", "cdf7ab6c1fd49bd9933c43f3ea5af185" },
	{"If the enemy is within range, then so are you.", "83bc85234942fc883c063cbd7f0ad5d0"},
	{"It's well we cannot hear the screams/That we create in others' dreams.", "277cbe255686b48dd7e8f389394d9299"},
	{"You remind me of a TV show, but that's all right: I watch it anyway.", "fd3fb0a7ffb8af16603f3d3af98f8e1f"},
	{"C is as portable as Stonehedge!!", "469b13a78ebf297ecda64d4723655154"},
	{"Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley", "63eb3a2f466410104731c4b037600110"},
	{"The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule", "72c2ed7592debca1c90fc0100f931a2f"},
	{"How can you write a big system without C++?  -Paul Glick", "132f7619d33b523b1d9e5bd8e0928355"},
	{"", "d41d8cd98f00b204e9800998ecf8427e"},
}

func TestGolangGolden(t *testing.T) {

	server := NewMd5Server()
	h8 := [8]hash.Hash{}
	for i := range h8 {
		h8[i] = NewMd5(server)
	}

	for tc := 0; tc < len(golden); tc += 8 {
		for i := range h8 {
			h8[i].Reset()
			h8[i].Write([]byte(golden[tc+i].in))
		}

		for i := range h8 {
			digest := h8[i].Sum([]byte{})
			if fmt.Sprintf("%x", digest) != golden[tc+i].want {
				t.Errorf("TestGolangGolden[%d], got %v, want %v", tc+i, fmt.Sprintf("%x", digest), golden[tc+i].want)
			}
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
