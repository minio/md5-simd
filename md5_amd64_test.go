//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"bytes"
	"hash"
	"runtime"
	"sync"
	"testing"

	"github.com/klauspost/cpuid/v2"
)

const benchmarkWithSum = true

func BenchmarkAvx512(b *testing.B) {

	if !hasAVX512 {
		b.SkipNow()
	}

	b.Run("32KB", func(b *testing.B) {
		benchmarkSingle(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkSingle(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkSingle(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkSingle(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkSingle(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkSingle(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkSingle(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkSingle(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkSingle(b, 8*1024*1024)
	})
}

func BenchmarkAvx512Parallel(b *testing.B) {

	if !hasAVX512 {
		b.SkipNow()
	}

	b.Run("32KB", func(b *testing.B) {
		benchmarkParallel(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkParallel(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkParallel(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkParallel(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkParallel(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkParallel(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkParallel(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkParallel(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkParallel(b, 8*1024*1024)
	})
}

func benchmarkSingle(b *testing.B, blockSize int) {
	server := NewServer()
	defer server.Close()
	h16 := [16]hash.Hash{}
	input := [16][]byte{}
	for i := range h16 {
		h16[i] = server.NewHash()
		input[i] = bytes.Repeat([]byte{0x61 + byte(i)}, blockSize)
	}
	// Technically this uses up to 2 cores, but it is the throughput of a single server.
	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()
	var tmp [Size]byte

	for j := 0; j < b.N; j++ {
		var wg sync.WaitGroup
		wg.Add(16)
		for i := range h16 {
			go func(i int) {
				// write to all concurrently
				defer wg.Done()
				h16[i].Reset()
				h16[i].Write(input[i])
				if benchmarkWithSum {
					_ = h16[i].Sum(tmp[:0])
				}
			}(i)
		}
		wg.Wait()
	}
}

func benchmarkSingleWriter(b *testing.B, blockSize int) {
	server := NewServer()
	defer server.Close()
	h := server.NewHash()
	input := bytes.Repeat([]byte{0x61}, blockSize)

	b.SetBytes(int64(blockSize))
	b.ReportAllocs()
	b.ResetTimer()
	var tmp [Size]byte

	for j := 0; j < b.N; j++ {
		h.Write(input)
		if benchmarkWithSum {
			_ = h.Sum(tmp[:0])
		}
	}
}

func benchmarkParallel(b *testing.B, blockSize int) {
	// We write input 16x per loop.
	// We have to alloc per parallel
	b.SetBytes(int64(blockSize * 16))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		input := bytes.Repeat([]byte{0x61}, blockSize)
		server := NewServer()
		defer server.Close()
		var h16 [16]Hasher
		for i := range h16 {
			h16[i] = server.NewHash()
			defer h16[i].Close()
		}
		var tmp [Size]byte
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(16)
			for i := range h16 {
				// Write to all concurrently
				go func(i int) {
					defer wg.Done()
					h16[i].Reset()
					h16[i].Write(input)
					if benchmarkWithSum {
						_ = h16[i].Sum(tmp[:0])
					}
				}(i)
			}
			wg.Wait()
		}
	})
}

func BenchmarkAvx2(b *testing.B) {
	// Make sure AVX512 is disabled
	restore := hasAVX512
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkSingle(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkSingle(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkSingle(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkSingle(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkSingle(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkSingle(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkSingle(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkSingle(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkSingle(b, 8*1024*1024)
	})

	hasAVX512 = restore
}

func BenchmarkAvx2Parallel(b *testing.B) {
	if !cpuid.CPU.Supports(cpuid.AVX2) {
		b.SkipNow()
	}
	restore := hasAVX512

	// Make sure AVX512 is disabled
	hasAVX512 = false
	b.SetParallelism((runtime.GOMAXPROCS(0) + 1) / 2)

	b.Run("32KB", func(b *testing.B) {
		benchmarkParallel(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkParallel(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkParallel(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkParallel(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkParallel(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkParallel(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkParallel(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkParallel(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkParallel(b, 8*1024*1024)
	})
	hasAVX512 = restore
}

// BenchmarkAvx2SingleWriter will benchmark the speed having only a single writer
// writing blocks with the specified size.
// This is pretty much the worst case scenario.
func BenchmarkAvx2SingleWriter(b *testing.B) {
	// Make sure AVX512 is disabled
	restore := hasAVX512
	hasAVX512 = false

	b.Run("32KB", func(b *testing.B) {
		benchmarkSingleWriter(b, 32*1024)
	})
	b.Run("64KB", func(b *testing.B) {
		benchmarkSingleWriter(b, 64*1024)
	})
	b.Run("128KB", func(b *testing.B) {
		benchmarkSingleWriter(b, 128*1024)
	})
	b.Run("256KB", func(b *testing.B) {
		benchmarkSingleWriter(b, 256*1024)
	})
	b.Run("512KB", func(b *testing.B) {
		benchmarkSingleWriter(b, 512*1024)
	})
	b.Run("1MB", func(b *testing.B) {
		benchmarkSingleWriter(b, 1024*1024)
	})
	b.Run("2MB", func(b *testing.B) {
		benchmarkSingleWriter(b, 2*1024*1024)
	})
	b.Run("4MB", func(b *testing.B) {
		benchmarkSingleWriter(b, 4*1024*1024)
	})
	b.Run("8MB", func(b *testing.B) {
		benchmarkSingleWriter(b, 8*1024*1024)
	})

	hasAVX512 = restore
}
