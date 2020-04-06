package md5simd

var hasAVX2 bool

// 8-way 4x uint32 digests in 4 ymm registers
// (ymm0, ymm1, ymm2, ymm3)
type digest8 struct {
	v0, v1, v2, v3 [8]uint32
}

// Stack cache for 8x64 byte md5.BlockSize bytes.
// Must be 32-byte aligned, so allocate 512+32 and
// align upwards at runtime.
type cache8 [512 + 32]byte

// MD5 magic numbers for one lane of hashing; inflated
// 8x below at init time.
var md5consts = [64]uint32{
	0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee,
	0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501,
	0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
	0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821,
	0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa,
	0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
	0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed,
	0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a,
	0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
	0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70,
	0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05,
	0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
	0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039,
	0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1,
	0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
	0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
}

// inflate the consts 8-way for 8x md5 (256 bit ymm registers)
var avx256md5consts = func(c []uint32) []uint32 {
	inf := make([]uint32, 8*len(c))
	for i := range c {
		for j := 0; j < 8; j++ {
			inf[(i*8)+j] = c[i]
		}
	}
	return inf
}(md5consts[:])

// 16-way 4x uint32 digests in 4 zmm registers
type digest16 struct {
	v0, v1, v2, v3 [16]uint32
}

// Stack cache for 16x64 byte md5.BlockSize bytes.
// Must be 32-byte aligned, so allocate 1024+64 and
// align upwards at runtime.
type cache16 [1024 + 64]byte

// inflate the consts 16-way for 16x md5 (512 bit zmm registers)
var avx512md5consts = func(c []uint32) []uint32 {
	inf := make([]uint32, 16*len(c))
	for i := range c {
		for j := 0; j < 16; j++ {
			inf[(i*16)+j] = c[i]
		}
	}
	return inf
}(md5consts[:])
