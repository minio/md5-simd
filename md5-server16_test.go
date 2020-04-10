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

func TestGolangGolden16(t *testing.T) {

	server := NewMd5Server16()
	h16 := [16]hash.Hash{}
	for i := range h16 {
		h16[i] = NewMd5_x16(server)
	}

	// Skip first 8, so we even 2 rounds of 16 test vectors
	golden16 := golden[8:]

	for tc := 0; tc < len(golden16); tc += 16 {
		for i := range h16 {
			h16[i].Reset()
			h16[i].Write([]byte(golden16[tc+i].in))
		}

		for i := range h16 {
			digest := h16[i].Sum([]byte{})
			if fmt.Sprintf("%x", digest) != golden16[tc+i].want {
				t.Errorf("TestGolangGolden[%d], got %v, want %v", tc+i, fmt.Sprintf("%x", digest), golden16[tc+i].want)
			}
		}
	}
}

func benchmarkGolden16(b *testing.B, blockSize int) {

	server := NewMd5Server16()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = NewMd5_x16(server)
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		for i := range h16 {
			h16[i].Write(input[i])
		}
	}
}

func BenchmarkGolden16(b *testing.B) {
	b.Run("32KB", func(b *testing.B) {
		benchmarkGolden16(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkGolden16(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkGolden16(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkGolden16(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkGolden16(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkGolden16(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkGolden16(b, 2*1024*1024)
	})
}

func benchmarkCryptoMd5(b *testing.B, blockSize int) {

	input := bytes.Repeat([]byte{0x61}, blockSize)

	b.SetBytes(int64(blockSize))
	b.ReportAllocs()
	b.ResetTimer()

	h := md5.New()

	for j := 0; j < b.N; j++ {
		h.Write(input)
	}
}

func BenchmarkCryptoMd5(b *testing.B) {
	b.Run("32KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkCryptoMd5(b, 2*1024*1024)
	})
}
