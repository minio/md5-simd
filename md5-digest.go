// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"encoding/binary"
	"errors"
	"time"
)

// Md5Digest - Type for computing MD5 using either AVX2 or AVX512
type Md5Digest struct {
	uid    uint64
	md5srv *Md5Server
	x      [BlockSize]byte
	nx     int
	len    uint64
	final  bool
	result [Size]byte
}

// Size - Return size of checksum
func (d *Md5Digest) Size() int { return Size }

// BlockSize - Return blocksize of checksum
func (d Md5Digest) BlockSize() int { return BlockSize }

// Reset - reset digest to its initial values
func (d *Md5Digest) Reset() {
	d.md5srv.blocksCh <- blockInput{uid: d.uid, reset: true}
	d.nx = 0
	d.len = 0
	d.final = false
}

// Write to digest
func (d *Md5Digest) Write(p []byte) (nn int, err error) {
	// break input into chunks of maximum MaxBlockSize size
	for {
		l := len(p)
		if l > MaxBlockSize {
			l = MaxBlockSize
		}
		nnn, err := d.write(p[:l])
		if err != nil {
			return nn, err
		}
		nn += nnn
		p = p[l:]

		if len(p) == 0 {
			break
		}

		time.Sleep(WriteSleepMs * time.Millisecond)
	}
	return
}

func (d *Md5Digest) write(p []byte) (nn int, err error) {

	if d.final {
		return 0, errors.New("Md5Digest already finalized. Reset first before writing again")
	}

	nn = len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == BlockSize {
			d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: d.x[:]}
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: p[:n]}
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

// Sum - Return MD5 sum in bytes
func (d *Md5Digest) Sum(in []byte) (result []byte) {

	if d.final {
		return append(in, d.result[:]...)
	}

	trail := make([]byte, 0, 128)
	trail = append(trail, d.x[:d.nx]...)

	len := d.len
	// Padding.  Add a 1 bit and 0 bits until 56 bytes mod 64.
	var tmp [64]byte
	tmp[0] = 0x80
	if len%64 < 56 {
		trail = append(trail, tmp[0:56-len%64]...)
	} else {
		trail = append(trail, tmp[0:64+56-len%64]...)
	}
	d.nx = 0

	// Length in bits.
	len <<= 3
	binary.LittleEndian.PutUint64(tmp[:], len) // append length in bits
	trail = append(trail, tmp[0:8]...)

	sumCh := make(chan [Size]byte)
	d.md5srv.blocksCh <- blockInput{uid: d.uid, msg: trail, final: true, sumCh: sumCh}
	d.result = <-sumCh
	d.final = true
	return append(in, d.result[:]...)
}
