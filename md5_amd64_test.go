//+build !noasm,!appengine,gc

// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"bytes"
	"crypto/md5"
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

// TestCloseRemovesDigest checks that the server forgets a hasher when it is
// closed, also when Sum was called and Reset was not.
func TestCloseRemovesDigest(t *testing.T) {
	server := NewServer()
	t.Cleanup(server.Close)
	s, ok := server.(*md5Server)
	if !ok {
		t.Skip("fallback server")
	}
	input := bytes.Repeat([]byte{0x61}, 3*BlockSize+7)
	want := md5.Sum(input)
	const n = 1000

	// Sum, then Close.
	for i := 0; i < n; i++ {
		h := server.NewHash()
		h.Write(input)
		if got := h.Sum(nil); !bytes.Equal(got, want[:]) {
			t.Fatalf("got %x, want %x", got, want)
		}
		h.Close()
	}
	// Close with a block still queued.
	for i := 0; i < n; i++ {
		h := server.NewHash()
		h.Write(input)
		h.Close()
	}
	// The server answers a Sum only after a processing pass, and every pass
	// either drops all closed hashers it still knows (it scans its clients
	// until it has no more clients than lanes) or makes progress on at
	// least 16 of them. The 1000 rounds give it far more passes than the
	// 2000 it could need, so when the probe's Sum returns the server knows
	// only the probe and touches the map no more: it can be read without
	// a race.
	for i := 0; i < n; i++ {
		h := server.NewHash()
		h.Write(input)
		if got := h.Sum(nil); !bytes.Equal(got, want[:]) {
			t.Fatalf("got %x, want %x", got, want)
		}
		h.Close()
	}
	probe := server.NewHash()
	defer probe.Close()
	probe.Write(input)
	probe.Sum(nil)
	if got := len(s.digests); got != 1 {
		t.Errorf("server keeps %d digests after closing %d hashers, want 1 (the open hasher)", got, 3*n)
	}
}

// TestCloseWithFullCycle checks that an idle server forgets a hasher that was
// closed when the cycle channel was full, so that Close could not send its uid.
func TestCloseWithFullCycle(t *testing.T) {
	if !cpuid.CPU.Supports(cpuid.AVX2) {
		t.Skip("fallback server")
	}
	s := newMd5Server(ServerOptions{UseAVX512: true})
	done := make(chan struct{})
	go func() {
		s.process(s.newInput)
		close(done)
	}()
	waitCycleEmpty := func() {
		for len(s.cycle) > 0 {
			runtime.Gosched()
		}
	}

	h := s.NewHash()
	h.Write(bytes.Repeat([]byte{0x61}, 3*BlockSize+7))
	h.Sum(nil)
	waitCycleEmpty() // no uid of h is queued any more

	// A channel that nobody reads is always full.
	h.(*md5Digest).cycleServer = make(chan uint64)
	h.Close()
	// The server reads only a stale uid, which does not fill a lane.
	s.cycle <- 0
	waitCycleEmpty()

	// The server returns only from a wait, after it has handled the stale
	// uid, so after done the map can be read without a race.
	s.Close()
	<-done
	if got := len(s.digests); got != 0 {
		t.Errorf("server keeps %d digests of closed hashers, want 0", got)
	}
}
