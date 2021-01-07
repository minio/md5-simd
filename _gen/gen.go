package main

//go:generate go run gen.go -out ../md5block_amd64.s -stubs ../md5block_amd64.go -pkg=md5simd

import (
	x "github.com/mmcloughlin/avo/build"
	"github.com/mmcloughlin/avo/buildtags"
	o "github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// AMD:
// 2025 BMI2               :RORX r32, r32, r32                    L:   0.29ns=  1.0c  T:   0.15ns=  0.50c
// 271 X86                 :ROL r32, imm8                         L:   0.29ns=  1.0c  T:   0.15ns=  0.50c
//
// INTEL:
// 271 X86             :ROL r32, imm8                         L:   0.27ns=  1.0c  T:   0.14ns=  0.50c
// 2025 BMI2            :RORX r32, r32, r32                    L:   0.27ns=  1.0c  T:   0.14ns=  0.50c

// Neither appear to have any gains
// Don't bother with BMI2
const useROLX = false

func ROLL(imm int, gpr reg.GPVirtual) {
	if useROLX {
		x.RORXL(o.U8(32-imm), gpr, gpr)
	} else {
		x.ROLL(o.U8(imm), gpr)
	}
}

// AMD:
// 154 X86                 :XOR r32, r32           L:   0.06ns=  0.2c  T:   0.06ns=  0.25c
// 166 X86                 :NOT r32                L:   0.26ns=  1.0c  T:   0.11ns=  0.43c
//
// INTEL:
// Inst  166 X86   : NOT r32                       L:   0.45ns=  1.0c  T:   0.11ns=  0.25c
// Inst  154 X86   : XOR r32, r32                  L:   0.11ns=  0.2c  T:   0.11ns=  0.25c
func NOTL(gpr, ones reg.GPVirtual) {
	// Use XOR
	if false {
		x.NOTL(gpr)
	} else {
		x.XORL(ones, gpr)
	}
}

func main() {
	x.Constraint(buildtags.Not("appengine").ToConstraint())
	x.Constraint(buildtags.Not("noasm").ToConstraint())
	x.Constraint(buildtags.Term("gc").ToConstraint())
	x.TEXT("blockScalar", 0, "func(dig *[4]uint32, p []byte)")
	x.Doc("Encode p to digest")
	x.Pragma("noescape")

	srcLen := x.Load(x.Param("p").Len(), x.GP64())
	digest := x.Load(x.Param("dig"), x.GP64())
	src := x.Load(x.Param("p").Base(), x.GP64())

	x.SHRQ(o.U8(6), srcLen)
	x.SHLQ(o.U8(6), srcLen)
	end := x.GP64()
	x.LEAQ(o.Mem{Base: src, Index: srcLen, Scale: 1}, end)
	x.CMPQ(src, end)
	x.JEQ(o.LabelRef("end"))
	var dig [4]reg.GPVirtual
	for i := range dig {
		dig[i] = x.GP32()
		x.MOVL(o.Mem{Base: digest, Disp: i * 4}, dig[i])
	}
	AX, BX, CX, DX := dig[0], dig[1], dig[2], dig[3]

	// Keep ones in a register
	ones := x.GP32()
	x.MOVL(o.U32(0xffffffff), ones)

	x.Label("loop")
	var block [4]reg.VecVirtual
	R8, R9 := x.GP32(), x.GP32()
	// load source. Skipped if idx < 0
	var loadSrc func(idx int, dst reg.GPVirtual)
	// Appears slower.
	const useXMM = false
	if useXMM {
		for i := range block {
			block[i] = x.XMM()
			x.MOVUPS(o.Mem{Base: src, Disp: 16 * i}, block[i])
		}
		// load source. Skipped if idx < 0
		loadSrc = func(idx int, dst reg.GPVirtual) {
			if idx < 0 {
				return
			}

			// 4 per block
			xmm := block[idx/4]
			x.PEXTRD(o.U8(idx&3), xmm, dst)
		}
	} else {
		loadSrc = func(idx int, dst reg.GPVirtual) {
			if idx < 0 {
				return
			}
			x.MOVL(o.Mem{Base: src, Disp: idx * 4}, dst)
		}
	}
	const useLEA = false
	loadSrc(0, R8)
	x.MOVL(DX, R9)

	// Copy digest
	R12, R13, R14, R15 := x.GP32(), x.GP32(), x.GP32(), x.GP32()
	x.MOVL(AX, R12)
	x.MOVL(BX, R13)
	x.MOVL(CX, R14)
	x.MOVL(DX, R15)

	// ROUND 1:
	x.Comment("ROUND1")
	ROUND1 := func(a, b, c, d reg.GPVirtual, index, con, shift int) {
		x.XORL(c, R9)
		if useLEA {
			x.LEAL(o.Mem{Base: a, Disp: con, Index: R8, Scale: 1}, a)
		} else {
			x.ADDL(o.U32(con), a)
			x.ADDL(R8, a)
		}
		x.ANDL(b, R9)
		x.XORL(d, R9)
		loadSrc(index, R8)
		x.ADDL(R9, a)
		ROLL(shift, a)
		x.MOVL(c, R9)
		x.ADDL(b, a)
	}

	ROUND1(AX, BX, CX, DX, 1, 0xd76aa478, 7)
	ROUND1(DX, AX, BX, CX, 2, 0xe8c7b756, 12)
	ROUND1(CX, DX, AX, BX, 3, 0x242070db, 17)
	ROUND1(BX, CX, DX, AX, 4, 0xc1bdceee, 22)
	ROUND1(AX, BX, CX, DX, 5, 0xf57c0faf, 7)
	ROUND1(DX, AX, BX, CX, 6, 0x4787c62a, 12)
	ROUND1(CX, DX, AX, BX, 7, 0xa8304613, 17)
	ROUND1(BX, CX, DX, AX, 8, 0xfd469501, 22)
	ROUND1(AX, BX, CX, DX, 9, 0x698098d8, 7)
	ROUND1(DX, AX, BX, CX, 10, 0x8b44f7af, 12)
	ROUND1(CX, DX, AX, BX, 11, 0xffff5bb1, 17)
	ROUND1(BX, CX, DX, AX, 12, 0x895cd7be, 22)
	ROUND1(AX, BX, CX, DX, 13, 0x6b901122, 7)
	ROUND1(DX, AX, BX, CX, 14, 0xfd987193, 12)
	ROUND1(CX, DX, AX, BX, 15, 0xa679438e, 17)
	// adjusted to load index 1
	ROUND1(BX, CX, DX, AX, 1, 0x49b40821, 22)

	x.Comment("ROUND2")
	x.MOVL(DX, R9)
	R10 := x.GP32()
	x.MOVL(DX, R10)

	ROUND2 := func(a, b, c, d reg.GPVirtual, index, con, shift int) {
		NOTL(R9, ones)
		if useLEA {
			x.LEAL(o.Mem{Base: a, Disp: con, Index: R8, Scale: 1}, a)
		} else {
			x.ADDL(o.U32(con), a)
			x.ADDL(R8, a)
		}

		x.ANDL(b, R10)
		x.ANDL(c, R9)
		loadSrc(index, R8)
		x.ORL(R9, R10)
		x.MOVL(c, R9)
		x.ADDL(R10, a)
		x.MOVL(c, R10)
		ROLL(shift, a)
		x.ADDL(b, a)
	}
	ROUND2(AX, BX, CX, DX, 6, 0xf61e2562, 5)
	ROUND2(DX, AX, BX, CX, 11, 0xc040b340, 9)
	ROUND2(CX, DX, AX, BX, 0, 0x265e5a51, 14)
	ROUND2(BX, CX, DX, AX, 5, 0xe9b6c7aa, 20)
	ROUND2(AX, BX, CX, DX, 10, 0xd62f105d, 5)
	ROUND2(DX, AX, BX, CX, 15, 0x2441453, 9)
	ROUND2(CX, DX, AX, BX, 4, 0xd8a1e681, 14)
	ROUND2(BX, CX, DX, AX, 9, 0xe7d3fbc8, 20)
	ROUND2(AX, BX, CX, DX, 14, 0x21e1cde6, 5)
	ROUND2(DX, AX, BX, CX, 3, 0xc33707d6, 9)
	ROUND2(CX, DX, AX, BX, 8, 0xf4d50d87, 14)
	ROUND2(BX, CX, DX, AX, 13, 0x455a14ed, 20)
	ROUND2(AX, BX, CX, DX, 2, 0xa9e3e905, 5)
	ROUND2(DX, AX, BX, CX, 7, 0xfcefa3f8, 9)
	ROUND2(CX, DX, AX, BX, 12, 0x676f02d9, 14)
	// Adjusted to load index 5
	ROUND2(BX, CX, DX, AX, 5, 0x8d2a4c8a, 20)

	x.Comment("ROUND3")
	x.MOVL(CX, R9)
	ROUND3 := func(a, b, c, d reg.GPVirtual, index, con, shift int) {
		// LEAL const(a)(R8*1), a; \
		if useLEA {
			x.LEAL(o.Mem{Base: a, Disp: con, Index: R8, Scale: 1}, a)
		} else {
			x.ADDL(o.U32(con), a)
			x.ADDL(R8, a)
		}
		loadSrc(index, R8)

		x.XORL(d, R9)
		x.XORL(b, R9)
		x.ADDL(R9, a)
		ROLL(shift, a)
		x.MOVL(b, R9)
		x.ADDL(b, a)
	}

	ROUND3(AX, BX, CX, DX, 8, 0xfffa3942, 4)
	ROUND3(DX, AX, BX, CX, 11, 0x8771f681, 11)
	ROUND3(CX, DX, AX, BX, 14, 0x6d9d6122, 16)
	ROUND3(BX, CX, DX, AX, 1, 0xfde5380c, 23)
	ROUND3(AX, BX, CX, DX, 4, 0xa4beea44, 4)
	ROUND3(DX, AX, BX, CX, 7, 0x4bdecfa9, 11)
	ROUND3(CX, DX, AX, BX, 10, 0xf6bb4b60, 16)
	ROUND3(BX, CX, DX, AX, 13, 0xbebfbc70, 23)
	ROUND3(AX, BX, CX, DX, 0, 0x289b7ec6, 4)
	ROUND3(DX, AX, BX, CX, 3, 0xeaa127fa, 11)
	ROUND3(CX, DX, AX, BX, 6, 0xd4ef3085, 16)
	ROUND3(BX, CX, DX, AX, 9, 0x4881d05, 23)
	ROUND3(AX, BX, CX, DX, 12, 0xd9d4d039, 4)
	ROUND3(DX, AX, BX, CX, 15, 0xe6db99e5, 11)
	ROUND3(CX, DX, AX, BX, 2, 0x1fa27cf8, 16)
	ROUND3(BX, CX, DX, AX, 0, 0xc4ac5665, 23)

	// Use extra reg for constant
	x.Comment("ROUND4")
	x.MOVL(ones, R9)
	x.XORL(DX, R9)
	ROUND4 := func(a, b, c, d reg.GPVirtual, index, con, shift int) {
		// LEAL const(a)(R8*1), a; \
		if useLEA {
			x.LEAL(o.Mem{Base: a, Disp: con, Index: R8, Scale: 1}, a)
		} else {
			x.ADDL(o.U32(con), a)
			x.ADDL(R8, a)
		}
		x.ORL(b, R9)
		x.XORL(c, R9)
		x.ADDL(R9, a)
		loadSrc(index, R8)
		if index >= 0 {
			x.MOVL(ones, R9)
		}
		ROLL(shift, a)
		if index >= 0 {
			x.XORL(c, R9)
		}
		x.ADDL(b, a)
	}

	ROUND4(AX, BX, CX, DX, 7, 0xf4292244, 6)
	ROUND4(DX, AX, BX, CX, 14, 0x432aff97, 10)
	ROUND4(CX, DX, AX, BX, 5, 0xab9423a7, 15)
	ROUND4(BX, CX, DX, AX, 12, 0xfc93a039, 21)
	ROUND4(AX, BX, CX, DX, 3, 0x655b59c3, 6)
	ROUND4(DX, AX, BX, CX, 10, 0x8f0ccc92, 10)
	ROUND4(CX, DX, AX, BX, 1, 0xffeff47d, 15)
	ROUND4(BX, CX, DX, AX, 8, 0x85845dd1, 21)
	ROUND4(AX, BX, CX, DX, 15, 0x6fa87e4f, 6)
	ROUND4(DX, AX, BX, CX, 6, 0xfe2ce6e0, 10)
	ROUND4(CX, DX, AX, BX, 13, 0xa3014314, 15)
	ROUND4(BX, CX, DX, AX, 4, 0x4e0811a1, 21)
	ROUND4(AX, BX, CX, DX, 11, 0xf7537e82, 6)
	ROUND4(DX, AX, BX, CX, 2, 0xbd3af235, 10)
	ROUND4(CX, DX, AX, BX, 9, 0x2ad7d2bb, 15)
	ROUND4(BX, CX, DX, AX, -1, 0xeb86d391, 21)

	x.ADDL(R12, AX)
	x.ADDL(R13, BX)
	x.ADDL(R14, CX)
	x.ADDL(R15, DX)

	// NEXT LOOP
	x.Comment("Prepare next loop")
	x.ADDQ(o.U8(64), src)
	x.CMPQ(src, end)
	x.JB(o.LabelRef("loop"))

	// Write...
	x.Comment("Write output")
	digest = x.Load(x.Param("dig"), x.GP64())
	for i := range dig {
		x.MOVL(dig[i], o.Mem{Base: digest, Disp: i * 4})
	}

	x.Label("end")
	x.RET()

	x.Generate()
}
