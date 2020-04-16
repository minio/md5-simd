// +build !amd64 appengine !gc noasm
	
// Copyright (c) 2020 MinIO Inc. All rights reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file.

package md5simd

import (
	"crypto/md5"
	"hash"
)

var hasAVX2 bool
var hasAVX512 bool

type digest16 struct {
	v0, v1, v2, v3 [16]uint32
}

// NewMd5
func NewMd5(md5srv *Md5Server) hash.Hash {
	return md5.New()
}

func blockMd5_x16(s *digest16, input [16][]byte, bases [2][]byte) {
}
