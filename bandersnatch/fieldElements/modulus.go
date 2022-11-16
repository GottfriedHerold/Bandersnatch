package fieldElements

// DEPRECATED FILE

import (
	"math/bits"
)

// m is an alias for BaseFiledSize
// re is the reciprocral of the modulus
// mmu0 is the largest multiple of the modulus such that  mmu0 < 2^256
// mmu1 is the smallest multiple of the modulus such that mm1 >= 2^256

const (
	// modulo    = common.BaseFieldSize_untyped //0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	reciprocal = 0x0000000000000002_355094edfede377c_38b5dcb707e08ed3_65043eb4be4bad71_42737a020c0d6393 // [5]uint64{0x42737a020c0d6393 0x65043eb4be4bad71 0x38b5dcb707e08ed3 0x355094edfede377c 0x2}
	mmu0       = 0xe7db4ea6533afa90_6673b0101343b00a_a77b4805fffcb7fd_fffffffe00000002                  // [4]uint64{0xfffffffe00000002 0xa77b4805fffcb7fd 0x6673b0101343b00a 0xe7db4ea6533afa90}
	mmu1       = 0x5bc8f5f97cd877d8_99ad88181ce5880f_fb38ec08fffb13fc_fffffffd00000003                  // [4]uint64{0xfffffffd00000003 0xfb38ec08fffb13fc 0x99ad88181ce5880f 0x5bc8f5f97cd877d8}
)

// 64-bit sized words of the modulus. The index is the position of the word. Shorthand for the baseFieldSize
const (
	m_0 = baseFieldSize_0
	m_1 = baseFieldSize_1
	m_2 = baseFieldSize_2
	m_3 = baseFieldSize_3
)

// 64-bit sized words of the reciprocal, with the index being the position of the word.
const (
	re_0 = (reciprocal >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	re_1
	re_2
	re_3
	re_4
)

// 64-bit sized words of the largest multiple of the modulus such that  mmu0 < 2^256, with the index being the position of the word.
const (
	mmu0_0 = (mmu0 >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	mmu0_1
	mmu0_2
	mmu0_3
)

// 64-bit sized words of the smallest multiple of the modulus such that mmu1 >= 2^256, with the index being the position of the word.
const (
	mmu1_0 = (mmu1 >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	mmu1_1
	mmu1_2
	mmu1_3
)

/*

The following functions can be used to calculate the reciprical, mmu0 and mmu1 of an arbritrary modulus m. For usage in the uint256 lib,
the constants above are used instead.

*/

// modulus contains a modulus `m` as well as derived values that help speed up computations.
// The allowed range for `m` is `2^192` to `2^256-1`.
// mmu0 is the largest multiple of the modulus such that  mmu0 < 2^256
// mmu1 is the smallest multiple of the modulus such that mmu1 >= 2^256
type modulus struct {
	m    [4]uint64 // modulus
	re   [5]uint64 // reciprocal
	mmu0 [4]uint64 // m*(mu/2^256 + 0) (with / being integer division)
	mmu1 [4]uint64 // m*(mu/2^256 + 1) % 2^256 (with / being integer division)
}

// Generates a modulus from a uint256 (equivalent to [4]uint64 )
func (z *modulus) FromUint256(m Uint256) {

	if m[3] == 0 {
		panic("Modulus < 2^192")
	}
	// Store the modulus itself
	*z = modulus{m: m}

	// Compute reciprocal of m
	z.reciprocal()

	// Compute mmu0, mmu1
	z.mmu()
}

// ToUint256 returns an array with the modulus.
func (z *modulus) ToUint256() Uint256 {
	return z.m
}

// reciprocal computes a 320-bit value re representing 2^512/m
// (or equivalently 1/m in a 0.320-bit fixed point representation)
//
// Notes:
// - starts with a 32-bit division, refines with newton-raphson iterations
// - re * m < 2^512
// - re * m + m >= 2^512
// - re * m + m == 2^512 iff m is a power of 2
func (z *modulus) reciprocal() {
	// Note: specialized for m[3] != 0
	s := bits.LeadingZeros64(z.m[3])
	p := 63 - s

	// 0 or a power of 2?

	// Check if at least one bit is set in m[2], m[1] or m[0],
	// or at least two bits in m[3]
	// If not, m is 0 or a power of 2
	if z.m[0]|z.m[1]|z.m[2]|(z.m[3]&(z.m[3]-1)) == 0 {
		z.re[4] = ^uint64(0) >> uint(p&63)
		z.re[3] = ^uint64(0)
		z.re[2] = ^uint64(0)
		z.re[1] = ^uint64(0)
		z.re[0] = ^uint64(0)

		return
	}

	// Maximise division precision by left-aligning divisor
	var (
		y  [4]uint64 // left-aligned copy of m
		r0 uint32    // estimate of 2^31/y
	)

	y = shiftleft256(z.m, uint(s)) // 1/2 < y < 1

	// Extract most significant 32 bits

	yh := uint32(y[3] >> 32)

	if yh == 0x80000000 { // Avoid overflow in division
		r0 = 0xffffffff
	} else {
		r0, _ = bits.Div32(0x80000000, 0, yh)
	}

	// First iteration: 32 -> 64

	t1 := uint64(r0)             // 2^31/y
	t1 *= t1                     // 2^62/y^2
	t1, _ = bits.Mul64(t1, y[3]) // 2^62/y^2 * 2^64/y / 2^64 = 2^62/y

	r1 := uint64(r0) << 32 // 2^63/y
	r1 -= t1               // 2^63/y - 2^62/y = 2^62/y
	r1 *= 2                // 2^63/y

	if (r1 | (y[3] << 1)) == 0 {
		r1 = ^uint64(0)
	}

	// Second iteration: 64 -> 128

	// square: 2^126/y^2
	a2h, a2l := bits.Mul64(r1, r1)

	// multiply by y: e2h:e2l:b2h = 2^126/y^2 * 2^128/y / 2^128 = 2^126/y
	b2h, _ := bits.Mul64(a2l, y[2])
	c2h, c2l := bits.Mul64(a2l, y[3])
	d2h, d2l := bits.Mul64(a2h, y[2])
	e2h, e2l := bits.Mul64(a2h, y[3])

	b2h, c := bits.Add64(b2h, c2l, 0)
	e2l, c = bits.Add64(e2l, c2h, c)
	e2h, _ = bits.Add64(e2h, 0, c)

	_, c = bits.Add64(b2h, d2l, 0)
	e2l, c = bits.Add64(e2l, d2h, c)
	e2h, _ = bits.Add64(e2h, 0, c)

	// subtract: t2h:t2l = 2^127/y - 2^126/y = 2^126/y
	t2l, b := bits.Sub64(0, e2l, 0)
	t2h, _ := bits.Sub64(r1, e2h, b)

	// double: r2h:r2l = 2^127/y
	r2l, c := bits.Add64(t2l, t2l, 0)
	r2h, _ := bits.Add64(t2h, t2h, c)

	if (r2h | r2l | (y[3] << 1)) == 0 {
		r2h = ^uint64(0)
		r2l = ^uint64(0)
	}

	// Third iteration: 128 -> 192

	// square r2 (keep 256 bits): 2^190/y^2
	a3h, a3l := bits.Mul64(r2l, r2l)
	b3h, b3l := bits.Mul64(r2l, r2h)
	c3h, c3l := bits.Mul64(r2h, r2h)

	a3h, c = bits.Add64(a3h, b3l, 0)
	c3l, c = bits.Add64(c3l, b3h, c)
	c3h, _ = bits.Add64(c3h, 0, c)

	a3h, c = bits.Add64(a3h, b3l, 0)
	c3l, c = bits.Add64(c3l, b3h, c)
	c3h, _ = bits.Add64(c3h, 0, c)

	// multiply by y: q = 2^190/y^2 * 2^192/y / 2^192 = 2^190/y

	x0 := a3l
	x1 := a3h
	x2 := c3l
	x3 := c3h

	var q0, q1, q2, q3, q4, t0 uint64

	q0, _ = bits.Mul64(x2, y[0])
	q1, t0 = bits.Mul64(x3, y[0])
	q0, c = bits.Add64(q0, t0, 0)
	q1, _ = bits.Add64(q1, 0, c)

	t1, _ = bits.Mul64(x1, y[1])
	q0, c = bits.Add64(q0, t1, 0)
	q2, t0 = bits.Mul64(x3, y[1])
	q1, c = bits.Add64(q1, t0, c)
	q2, _ = bits.Add64(q2, 0, c)

	t1, t0 = bits.Mul64(x2, y[1])
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	q2, _ = bits.Add64(q2, 0, c)

	t1, t0 = bits.Mul64(x1, y[2])
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	q3, t0 = bits.Mul64(x3, y[2])
	q2, c = bits.Add64(q2, t0, c)
	q3, _ = bits.Add64(q3, 0, c)

	t1, _ = bits.Mul64(x0, y[2])
	q0, c = bits.Add64(q0, t1, 0)
	t1, t0 = bits.Mul64(x2, y[2])
	q1, c = bits.Add64(q1, t0, c)
	q2, c = bits.Add64(q2, t1, c)
	q3, _ = bits.Add64(q3, 0, c)

	t1, t0 = bits.Mul64(x1, y[3])
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	q4, t0 = bits.Mul64(x3, y[3])
	q3, c = bits.Add64(q3, t0, c)
	q4, _ = bits.Add64(q4, 0, c)

	t1, t0 = bits.Mul64(x0, y[3])
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	t1, t0 = bits.Mul64(x2, y[3])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q4, _ = bits.Add64(q4, 0, c)

	// subtract: t3 = 2^191/y - 2^190/y = 2^190/y
	_, b = bits.Sub64(0, q0, 0)
	_, b = bits.Sub64(0, q1, b)
	t3l, b := bits.Sub64(0, q2, b)
	t3m, b := bits.Sub64(r2l, q3, b)
	t3h, _ := bits.Sub64(r2h, q4, b)

	// double: r3 = 2^191/y
	r3l, c := bits.Add64(t3l, t3l, 0)
	r3m, c := bits.Add64(t3m, t3m, c)
	r3h, _ := bits.Add64(t3h, t3h, c)

	// Fourth iteration: 192 -> 320

	// square r3

	a4h, a4l := bits.Mul64(r3l, r3l)
	b4h, b4l := bits.Mul64(r3l, r3m)
	c4h, c4l := bits.Mul64(r3l, r3h)
	d4h, d4l := bits.Mul64(r3m, r3m)
	e4h, e4l := bits.Mul64(r3m, r3h)
	f4h, f4l := bits.Mul64(r3h, r3h)

	b4h, c = bits.Add64(b4h, c4l, 0)
	e4l, c = bits.Add64(e4l, c4h, c)
	e4h, _ = bits.Add64(e4h, 0, c)

	a4h, c = bits.Add64(a4h, b4l, 0)
	d4l, c = bits.Add64(d4l, b4h, c)
	d4h, c = bits.Add64(d4h, e4l, c)
	f4l, c = bits.Add64(f4l, e4h, c)
	f4h, _ = bits.Add64(f4h, 0, c)

	a4h, c = bits.Add64(a4h, b4l, 0)
	d4l, c = bits.Add64(d4l, b4h, c)
	d4h, c = bits.Add64(d4h, e4l, c)
	f4l, c = bits.Add64(f4l, e4h, c)
	f4h, _ = bits.Add64(f4h, 0, c)

	// multiply by y

	x1, x0 = bits.Mul64(d4h, y[0])
	x3, x2 = bits.Mul64(f4h, y[0])
	t1, t0 = bits.Mul64(f4l, y[0])
	x1, c = bits.Add64(x1, t0, 0)
	x2, c = bits.Add64(x2, t1, c)
	x3, _ = bits.Add64(x3, 0, c)

	t1, t0 = bits.Mul64(d4h, y[1])
	x1, c = bits.Add64(x1, t0, 0)
	x2, c = bits.Add64(x2, t1, c)
	x4, t0 := bits.Mul64(f4h, y[1])
	x3, c = bits.Add64(x3, t0, c)
	x4, _ = bits.Add64(x4, 0, c)
	t1, t0 = bits.Mul64(d4l, y[1])
	x0, c = bits.Add64(x0, t0, 0)
	x1, c = bits.Add64(x1, t1, c)
	t1, t0 = bits.Mul64(f4l, y[1])
	x2, c = bits.Add64(x2, t0, c)
	x3, c = bits.Add64(x3, t1, c)
	x4, _ = bits.Add64(x4, 0, c)

	t1, t0 = bits.Mul64(a4h, y[2])
	x0, c = bits.Add64(x0, t0, 0)
	x1, c = bits.Add64(x1, t1, c)
	t1, t0 = bits.Mul64(d4h, y[2])
	x2, c = bits.Add64(x2, t0, c)
	x3, c = bits.Add64(x3, t1, c)
	x5, t0 := bits.Mul64(f4h, y[2])
	x4, c = bits.Add64(x4, t0, c)
	x5, _ = bits.Add64(x5, 0, c)
	t1, t0 = bits.Mul64(d4l, y[2])
	x1, c = bits.Add64(x1, t0, 0)
	x2, c = bits.Add64(x2, t1, c)
	t1, t0 = bits.Mul64(f4l, y[2])
	x3, c = bits.Add64(x3, t0, c)
	x4, c = bits.Add64(x4, t1, c)
	x5, _ = bits.Add64(x5, 0, c)

	t1, t0 = bits.Mul64(a4h, y[3])
	x1, c = bits.Add64(x1, t0, 0)
	x2, c = bits.Add64(x2, t1, c)
	t1, t0 = bits.Mul64(d4h, y[3])
	x3, c = bits.Add64(x3, t0, c)
	x4, c = bits.Add64(x4, t1, c)
	x6, t0 := bits.Mul64(f4h, y[3])
	x5, c = bits.Add64(x5, t0, c)
	x6, _ = bits.Add64(x6, 0, c)
	t1, t0 = bits.Mul64(a4l, y[3])
	x0, c = bits.Add64(x0, t0, 0)
	x1, c = bits.Add64(x1, t1, c)
	t1, t0 = bits.Mul64(d4l, y[3])
	x2, c = bits.Add64(x2, t0, c)
	x3, c = bits.Add64(x3, t1, c)
	t1, t0 = bits.Mul64(f4l, y[3])
	x4, c = bits.Add64(x4, t0, c)
	x5, c = bits.Add64(x5, t1, c)
	x6, _ = bits.Add64(x6, 0, c)

	// subtract
	_, b = bits.Sub64(0, x0, 0)
	_, b = bits.Sub64(0, x1, b)
	r4l, b := bits.Sub64(0, x2, b)
	r4k, b := bits.Sub64(0, x3, b)
	r4j, b := bits.Sub64(r3l, x4, b)
	r4i, b := bits.Sub64(r3m, x5, b)
	r4h, _ := bits.Sub64(r3h, x6, b)

	// Multiply candidate for 1/4y by y, with full precision

	x0 = r4l
	x1 = r4k
	x2 = r4j
	x3 = r4i
	x4 = r4h

	q1, q0 = bits.Mul64(x0, y[0])
	q3, q2 = bits.Mul64(x2, y[0])
	q5, q4 := bits.Mul64(x4, y[0])

	t1, t0 = bits.Mul64(x1, y[0])
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	t1, t0 = bits.Mul64(x3, y[0])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(x0, y[1])
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	t1, t0 = bits.Mul64(x2, y[1])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q6, t0 := bits.Mul64(x4, y[1])
	q5, c = bits.Add64(q5, t0, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(x1, y[1])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(x3, y[1])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(x0, y[2])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(x2, y[2])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	q7, t0 := bits.Mul64(x4, y[2])
	q6, c = bits.Add64(q6, t0, c)
	q7, _ = bits.Add64(q7, 0, c)

	t1, t0 = bits.Mul64(x1, y[2])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	t1, t0 = bits.Mul64(x3, y[2])
	q5, c = bits.Add64(q5, t0, c)
	q6, c = bits.Add64(q6, t1, c)
	q7, _ = bits.Add64(q7, 0, c)

	t1, t0 = bits.Mul64(x0, y[3])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	t1, t0 = bits.Mul64(x2, y[3])
	q5, c = bits.Add64(q5, t0, c)
	q6, c = bits.Add64(q6, t1, c)
	q8, t0 := bits.Mul64(x4, y[3])
	q7, c = bits.Add64(q7, t0, c)
	q8, _ = bits.Add64(q8, 0, c)

	t1, t0 = bits.Mul64(x1, y[3])
	q4, c = bits.Add64(q4, t0, 0)
	q5, c = bits.Add64(q5, t1, c)
	t1, t0 = bits.Mul64(x3, y[3])
	q6, c = bits.Add64(q6, t0, c)
	q7, c = bits.Add64(q7, t1, c)
	q8, _ = bits.Add64(q8, 0, c)

	// Final adjustments: increment/decrement the result to get the correct reciprocal

	// subtract q from 1/4
	q0, b = bits.Sub64(0, q0, 0)
	q1, b = bits.Sub64(0, q1, b)
	q2, b = bits.Sub64(0, q2, b)
	q3, b = bits.Sub64(0, q3, b)
	q4, b = bits.Sub64(0, q4, b)
	q5, b = bits.Sub64(0, q5, b)
	q6, b = bits.Sub64(0, q6, b)
	q7, b = bits.Sub64(0, q7, b)
	q8, b = bits.Sub64(uint64(1)<<62, q8, b)

	// decrement the result
	x0, t := bits.Sub64(r4l, 1, 0)
	x1, t = bits.Sub64(r4k, 0, t)
	x2, t = bits.Sub64(r4j, 0, t)
	x3, t = bits.Sub64(r4i, 0, t)
	x4, _ = bits.Sub64(r4h, 0, t)

	// commit the decrement if the subtraction underflowed (reciprocal was too large)
	if b != 0 {
		r4h, r4i, r4j, r4k, r4l = x4, x3, x2, x1, x0
	}

	// subtract y from q
	/*q0*/
	_, b = bits.Sub64(q0, y[0], 0)
	/*q1*/ _, b = bits.Sub64(q1, y[1], b)
	/*q2*/ _, b = bits.Sub64(q2, y[2], b)
	/*q3*/ _, b = bits.Sub64(q3, y[3], b)
	/*q4*/ _, b = bits.Sub64(q4, 0, b)
	/*q5*/ _, b = bits.Sub64(q5, 0, b)
	/*q6*/ _, b = bits.Sub64(q6, 0, b)
	/*q7*/ _, b = bits.Sub64(q7, 0, b)
	/*q8*/ _, b = bits.Sub64(q8, 0, b)

	// increment the result
	x0, t = bits.Add64(r4l, 1, 0)
	x1, t = bits.Add64(r4k, 0, t)
	x2, t = bits.Add64(r4j, 0, t)
	x3, t = bits.Add64(r4i, 0, t)
	x4, _ = bits.Add64(r4h, 0, t)

	// commit the increment if the subtraction did not underflow (reciprocal was too small)
	if b == 0 {
		r4h, r4i, r4j, r4k, r4l = x4, x3, x2, x1, x0
	}

	// Shift to correct bit alignment, truncating excess bits

	p = p - 1

	x0, c = bits.Add64(r4l, r4l, 0)
	x1, c = bits.Add64(r4k, r4k, c)
	x2, c = bits.Add64(r4j, r4j, c)
	x3, c = bits.Add64(r4i, r4i, c)
	x4, _ = bits.Add64(r4h, r4h, c)

	if p < 0 {
		r4h, r4i, r4j, r4k, r4l = x4, x3, x2, x1, x0
		p = 0 // avoid negative shift below
	}

	// Shift right 0-62 bits
	{
		r := uint(p)      // right shift
		l := uint(64 - r) // left shift

		x0 = (r4l >> r) | (r4k << l)
		x1 = (r4k >> r) | (r4j << l)
		x2 = (r4j >> r) | (r4i << l)
		x3 = (r4i >> r) | (r4h << l)
		x4 = (r4h >> r)
	}

	if p > 0 {
		r4h, r4i, r4j, r4k, r4l = x4, x3, x2, x1, x0
	}

	z.re[0] = r4l
	z.re[1] = r4k
	z.re[2] = r4j
	z.re[3] = r4i
	z.re[4] = r4h
}

// precomputer the modulo multiples used in speed optmization
func (z *modulus) mmu() {
	var c, t0, t1, q0, q1, q2, q3, q4 uint64

	q2, q1 = bits.Mul64(z.m[1], z.re[4])
	q4, q3 = bits.Mul64(z.m[3], z.re[4])

	t1, q0 = bits.Mul64(z.m[0], z.re[4])
	q1, c = bits.Add64(q1, t1, 0)
	t1, t0 = bits.Mul64(z.m[2], z.re[4])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q4, _ = bits.Add64(q4, 0, c)

	if q4 != 0 {
		panic("Error preparing mmu0")
	}

	z.mmu0 = [4]uint64{q0, q1, q2, q3}

	z.mmu1[0], c = bits.Add64(z.mmu0[0], z.m[0], 0)
	z.mmu1[1], c = bits.Add64(z.mmu0[1], z.m[1], c)
	z.mmu1[2], c = bits.Add64(z.mmu0[2], z.m[2], c)
	z.mmu1[3], c = bits.Add64(z.mmu0[3], z.m[3], c)

	// => there must be a carry out from the addition above
	if c != 1 {
		panic("Error preparing mmu1")
	}
}

// shiftleft256 shifts the 256-bit value in a little-endian array left by 0-63 bits.
// This function is used in the calculation of the reciprocal to produce a left aligned copy of the modulus.
// For simplicity, it considers that m >= 2^192, hence there will never be more than 63 leading zeros.
//
// TODO: Alternativelly, this function could be inlined to reciprocal.
func shiftleft256(x Uint256, s uint) (z Uint256) {
	l := s % 64 // left shift
	r := 64 - l // right shift

	z[0] = (x[0] << l)
	z[1] = (x[1] << l) | (x[0] >> r)
	z[2] = (x[2] << l) | (x[1] >> r)
	z[3] = (x[3] << l) | (x[2] >> r)

	return z
}

//The bandersnatch values are
