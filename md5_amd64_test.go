//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"bytes"
	"hash"
	"testing"

	"github.com/klauspost/cpuid"
)

const benchmarkWithSum = true

func benchmarkAvx512(b *testing.B, blockSize int) {

	server := NewServer()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = server.NewHash()
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()
	var tmp [Size]byte

	for j := 0; j < b.N; j++ {
		for i := range h16 {
			h16[i].Write(input[i])
		}
		if benchmarkWithSum {
			for i := range h16 {
				_ = h16[i].Sum(tmp[:0])
				// FIXME(fwessels): Broken, since Sum closes the stream.
				// Once fixed this can be removed.
				h16[i].Reset()
			}
		}
	}
}

func BenchmarkAvx512(b *testing.B) {

	if !hasAVX512 {
		b.SkipNow()
	}

	b.Run("32KB", func(b *testing.B) {
		benchmarkAvx512(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkAvx512(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkAvx512(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkAvx512(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkAvx512(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkAvx512(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkAvx512(b, 2*1024*1024)
	})
}

func benchmarkAvx512P(b *testing.B, blockSize int) {
	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		input := bytes.Repeat([]byte{0x61}, blockSize)
		server := NewServer()
		defer server.Close()
		h16 := [16]hash.Hash{}
		for i := range h16 {
			h16[i] = server.NewHash()
		}
		var tmp [Size]byte
		for pb.Next() {
			for i := range h16 {
				h16[i].Write(input)
			}
			if benchmarkWithSum {
				for i := range h16 {
					_ = h16[i].Sum(tmp[:0])
					// FIXME(fwessels): Broken, since Sum closes the stream.
					// Once fixed this can be removed.
					h16[i].Reset()
				}
			}
		}
	})
}

func BenchmarkAvx512Parallel(b *testing.B) {

	if !hasAVX512 {
		b.SkipNow()
	}

	b.Run("32KB", func(b *testing.B) {
		benchmarkAvx512P(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkAvx512P(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkAvx512P(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkAvx512P(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkAvx512P(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkAvx512P(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkAvx512P(b, 2*1024*1024)
	})
}

func benchmarkAvx2(b *testing.B, blockSize int) {
	server := NewServer()
	defer server.Close()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = server.NewHash()
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}

	const cores = 2 // AVX2 runs on two cores, so split effective performance in half
	b.SetBytes(int64(blockSize * 16 / cores))
	b.ReportAllocs()
	b.ResetTimer()
	var tmp [Size]byte

	for j := 0; j < b.N; j++ {
		for i := range h16 {
			h16[i].Write(input[i])
		}
		if benchmarkWithSum {
			for i := range h16 {
				_ = h16[i].Sum(tmp[:0])
				// FIXME(fwessels): Broken, since Sum closes the stream.
				// Once fixed this can be removed.
				h16[i].Reset()
			}
		}
	}
}

func benchmarkAvx2P(b *testing.B, blockSize int) {
	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		input := bytes.Repeat([]byte{0x61}, blockSize)
		server := NewServer()
		defer server.Close()
		h16 := [16]hash.Hash{}
		for i := range h16 {
			h16[i] = server.NewHash()
		}
		var tmp [Size]byte
		for pb.Next() {
			for i := range h16 {
				h16[i].Write(input)
			}
			if benchmarkWithSum {
				for i := range h16 {
					_ = h16[i].Sum(tmp[:0])
					// FIXME(fwessels): Broken, since Sum closes the stream.
					// Once fixed this can be removed.
					h16[i].Reset()
				}
			}
		}
	})
}

// Runs with a single
func benchmarkAvx2PSingle(b *testing.B, blockSize int) {
	b.SetBytes(int64(blockSize))
	b.ReportAllocs()
	b.ResetTimer()
	server := NewServer()
	defer server.Close()
	b.RunParallel(func(pb *testing.PB) {
		input := bytes.Repeat([]byte{0x61}, blockSize)
		hasher := server.NewHash()
		var tmp [Size]byte
		for pb.Next() {
			hasher.Write(input)
			if benchmarkWithSum {
				_ = hasher.Sum(tmp[:0])
				// FIXME(fwessels): Broken, since Sum closes the stream.
				// Once fixed this can be removed.
				hasher.Reset()
			}
		}
	})
}

func BenchmarkAvx2(b *testing.B) {

	restore := hasAVX512

	// Make sure AVX512 is disabled
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkAvx2(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkAvx2(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkAvx2(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkAvx2(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkAvx2(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkAvx2(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkAvx2(b, 2*1024*1024)
	})

	hasAVX512 = restore
}

func BenchmarkAvx2Parallel(b *testing.B) {
	if !cpuid.CPU.AVX2() {
		b.SkipNow()
	}
	restore := hasAVX512

	// Make sure AVX512 is disabled
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkAvx2P(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkAvx2P(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkAvx2P(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkAvx2P(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkAvx2P(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkAvx2P(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkAvx2P(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkAvx2P(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkAvx2P(b, 8*1024*1024)
	})
	hasAVX512 = restore
}

func benchmarkAvx2ParallelSingle(b *testing.B) {
	if !cpuid.CPU.AVX2() {
		b.SkipNow()
	}
	restore := hasAVX512

	// Make sure AVX512 is disabled
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkAvx2PSingle(b, 8*1024*1024)
	})
	hasAVX512 = restore
}
