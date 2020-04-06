// +build amd64

package md5simd

//go:noescape
func block8(state *uint32, base uintptr, bufs *int32, cache *byte, n int)

//go:noescape
func block8Avx512(state *uint32, base uintptr, bufs *int32, cache *byte, n int, zreg *[64]byte)
