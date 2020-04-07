
#define prepmask \
	VXORPS   mask, mask, mask \
	VPCMPGTD mask, off, kmask

#define prep(index) \
	KMOVQ      kmask, ktmp                      \
	VPGATHERDD index*4(base)(off*1), ktmp, mem

#define load(index) \
	VMOVAPD index*64(cache), mem

#define store(index) \
	VMOVAPD mem, index*64(cache)

#define roll(shift, a) \
	VPSLLD $shift, a, rtmp1 \
	VPSRLD $32-shift, a, a  \
	VORPS  rtmp1, a, a

#define ROUND1(a, b, c, d, index, const, shift) \
	VXORPS  c, tmp, tmp            \
	VPADDD  64*const(consts), a, a \
	VPADDD  mem, a, a              \
	VANDPS  b, tmp, tmp            \
	VXORPS  d, tmp, tmp            \
	prep(index)                    \
	VPADDD  tmp, a, a              \
	roll(shift,a)                  \
	VMOVAPD c, tmp                 \
	VPADDD  b, a, a

#define ROUND1load(a, b, c, d, index, const, shift) \
	VXORPS  c, tmp, tmp            \
	VPADDD  64*const(consts), a, a \
	VPADDD  mem, a, a              \
	VANDPS  b, tmp, tmp            \
	VXORPS  d, tmp, tmp            \
	load(index)                    \
	VPADDD  tmp, a, a              \
	roll(shift,a)                  \
	VMOVAPD c, tmp                 \
	VPADDD  b, a, a


TEXT ·block16(SB),4,$0-48

    MOVQ state+0(FP),BX
    MOVQ base+8(FP),SI
    MOVQ bufs+16(FP),AX
    MOVQ cache+24(FP),CX
    MOVQ n+32(FP),DX
    MOVQ ·avx512md5consts+0(SB),DI

    // Align cache (which is stack allocated by the compiler)
    // to a 512 bit boundary (ymm register alignment)
    // The cache16 type is deliberately oversized to permit this.
    ADDQ $63,CX
    ANDB $-64,CL

#define a Z0
#define b Z1
#define c Z2
#define d Z3

#define sa Z4
#define sb Z5
#define sc Z6
#define sd Z7

#define tmp  Z8
#define xtmp X8

#define kmask K1
#define ktmp  K2
#define mask Z10
#define off  Z11

#define ones Z12

#define rtmp1  Z13
#define xrtmp1 X13
#define rtmp2  Z14
#define xrtmp2 X14

#define mem   Z15
#define xmem  X15

#define dig    BX
#define cache  CX
#define count  DX
#define base   SI
#define consts DI

	// load digest into state registers
	VMOVUPD (dig), a
	VMOVUPD 0x40(dig), b
	VMOVUPD 0x80(dig), c
	VMOVUPD 0xc0(dig), d

	// load source buffer offsets
	VMOVUPD (AX), off

	prepmask
//	VPCMPEQD ones, ones, ones

loop:
	VMOVAPD a, sa
	VMOVAPD b, sb
	VMOVAPD c, sc
	VMOVAPD d, sd

	prep(0)
	VMOVAPD d, tmp
	store(0)

	ROUND1(a,b,c,d, 1,0x00, 7)
	store(1)
	ROUND1(d,a,b,c, 2,0x01,12)
	store(2)
	ROUND1(c,d,a,b, 3,0x02,17)
	store(3)
	ROUND1(b,c,d,a, 4,0x03,22)
	store(4)
	ROUND1(a,b,c,d, 5,0x04, 7)
	store(5)
	ROUND1(d,a,b,c, 6,0x05,12)
	store(6)
	ROUND1(c,d,a,b, 7,0x06,17)
	store(7)
	ROUND1(b,c,d,a, 8,0x07,22)
	store(8)
	ROUND1(a,b,c,d, 9,0x08, 7)
	store(9)
	ROUND1(d,a,b,c,10,0x09,12)
	store(10)
	ROUND1(c,d,a,b,11,0x0a,17)
	store(11)
	ROUND1(b,c,d,a,12,0x0b,22)
	store(12)
	ROUND1(a,b,c,d,13,0x0c, 7)
	store(13)
	ROUND1(d,a,b,c,14,0x0d,12)
	store(14)
	ROUND1(c,d,a,b,15,0x0e,17)
	store(15)

	ROUND1load(b,c,d,a, 0,0x0f,22)

    MOVQ zreg+40(FP),AX
    VMOVDQU32 a, (AX)
    VMOVDQU32 b, 0x40(AX)
    VMOVDQU32 c, 0x80(AX)
    VMOVDQU32 d, 0xc0(AX)

	RET
