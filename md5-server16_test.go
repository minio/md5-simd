// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/remeh/sizedwaitgroup"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func testMd5Simulator(t *testing.T, server *Md5Server) {

	rand.Seed(time.Now().UnixNano())
	verifier := make(map[string]string)

	mu := sync.Mutex{}

	swg := sizedwaitgroup.New(24)
	for _i := 0; _i < 1000; _i++ {
		swg.Add()
		go func(i int) {
			defer swg.Done()
			h := NewMd5(server)
			mbs := 10 + rand.Intn(100)
			h.Write(bytes.Repeat([]byte{0x61 + byte(i)}, mbs*1024*1024))
			digest := fmt.Sprintf("%x", h.Sum([]byte{}))
			mu.Lock()
			verifier[fmt.Sprintf("%d-%d", i, mbs)] = digest
			mu.Unlock()
		}(_i)
	}

	swg.Wait()

	fmt.Printf("Verifying %d entries...\n", len(verifier))

	swg = sizedwaitgroup.New(runtime.NumCPU())

	for _input, _digest := range verifier {

		swg.Add()
		go func(input, digest string) {
			defer swg.Done()

			p := strings.Split(input, "-")
			i, _ := strconv.Atoi(p[0])
			mbs, _ := strconv.Atoi(p[1])

			h := md5.New()
			h.Write(bytes.Repeat([]byte{0x61 + byte(i)}, mbs*1024*1024))
			d := fmt.Sprintf("%x", h.Sum([]byte{}))

			if digest != d {
				t.Errorf("testMd5Simulator[%s], got %s, want %s", input, digest, d)
			}
		}(_input, _digest)
	}
	swg.Wait()

}

func TestMd5Simulator(t *testing.T) {

	if testing.Short() {
		t.SkipNow()
	}

	server := NewMd5Server()

	t.Run("", func(t *testing.T) {
		testMd5Simulator(t, server)
	})
}
