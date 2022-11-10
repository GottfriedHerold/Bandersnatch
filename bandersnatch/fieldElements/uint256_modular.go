package fieldElements

import (
	"math/big"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains methods on uint256 that interpret the elements as residues modulo BaseFieldSize and perform appropriate operations (such as addition, multiplication of residues)
// For reasons of efficiency, neither the input nor the output of these functions have to be the smallest representative of the residue class, i.e.
// elements are not neccessarily in [0, BaseFieldSize)
// Instead, we need to keep track of "reduction quality", i.e. in what range the uint256 is guaranteed to be.
// The relevant upper bounds (with strict inequality) are BaseFieldSize, 2**256-BaseFieldSize, 2*BaseFieldSize, 2**256.

// Bounds for reducedness-quality

var (
	twoTo256_Int            *big.Int = common.TwoTo256_Int                                                                                       // 2**256 == 115792089237316195423570985008687907853269984665640564039457584007913129639936
	doubleBaseFieldSize_Int *big.Int = utils.InitIntFromString("104871750350252380958895481016371931675381105001055275645207317399877162369026") // 2 * BaseFieldSize
	montgomeryRepBound_Int  *big.Int = utils.InitIntFromString("63356214062190004944123244500501942015579432165112926216853925307974548455423")  // 2**256 - BaseFieldSize
	// BaseFieldSize_Int *big.Int (already defined elsewhere) // 52435875175126190479447740508185965837690552500527637822603658699938581184513
)

// NOTE: Suffixes for reduction follows the following convention:
//   - a means arbitrary, i.e. the number is in [0,2^256)
//   - b means double-range, i.e. the number is in [0, 2*BaseFieldSize). Every residue class has exactly 2 possible representations
//   - c means carry-avoiding, i.e. the number is in [0, 2^256-BaseFieldSize). Note that 2^256/BaseFieldSize is between 2 and 3, so this is stricter than b.
//   - f means fully-reduced, i.e. in [0, BaseFieldSize)
//
// Methods have suffixes ReduceWeak_b to mean they operate with double-range reduced numbers.
// If the input constraints and output promises differ, we write _bc to have output b and input c
// If we have multiple reduction quality statements depending on the input, we write multiple clauses: foo_b_c means that if the input is b-reduced, so is the output and if the input is c-reduced, the output is c-reduced.

// toUint64 returns an array with the canonical representative of the residue class.
//
// DEPRECATED
func (z *uint256) toUint64() [4]uint64 {
	z.reduceBarret_fa() // Reduce to canonical residue modulo m
	return [4]uint64{z[0], z[1], z[2], z[3]}
}

// AddAndReduce_b_c sets z, such that z==x+y mod BaseFieldSize holds. Assumes x and y to be weakly reduced.
//
// z might not be the smallest such representations. More precisely:
//   - If x and y are both in [0, UINT256MAX-BaseFieldSize), then so is z.
//   - If x and y are both in [0, 2*BaseFieldSize), then so is z.
func (z *uint256) AddAndReduce_b_c(x, y *uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)

	// If carry == 1, then z.maybe_reduce_once() actually commutes with the -=mdoubled here: it won't do anything either before or after it.

	// On overflow, subtract an appropriate multiple of BaseFieldSize.
	// The preconditions guarantee that subtracting 2*BaseFieldSize always remedies the overflow.
	if carry != 0 {
		z[0], carry = bits.Sub64(z[0], baseFieldSizeDoubled_64_0, 0)
		z[1], carry = bits.Sub64(z[1], baseFieldSizeDoubled_64_1, carry)
		z[2], carry = bits.Sub64(z[2], baseFieldSizeDoubled_64_2, carry)
		z[3], _ = bits.Sub64(z[3], baseFieldSizeDoubled_64_3, carry)
	}
	// NOTE: We could do an else if here! This works for both cases of preconditions.
	if z[3] > baseFieldSize_3 {
		z[0], carry = bits.Sub64(z[0], baseFieldSize_0, 0)
		z[1], carry = bits.Sub64(z[1], baseFieldSize_1, carry)
		z[2], carry = bits.Sub64(z[2], baseFieldSize_2, carry)
		z[3], _ = bits.Sub64(z[3], baseFieldSize_3, carry) // _ is guaranteed to be 0
	}
}

// AddEqAndReduce_a computes z+=x mod BaseFieldSize, where the result may not be fully reduced.
// Both the inputs and the resulting z maybe anywhere in [0, 2^256).
func (z *uint256) AddEqAndReduce_a(x *uint256) {
	t0, c := bits.Add64(z[0], x[0], 0)
	t1, c := bits.Add64(z[1], x[1], c)
	t2, c := bits.Add64(z[2], x[2], c)
	t3, c := bits.Add64(z[3], x[3], c)

	if c == 0 {
		z[3], z[2], z[1], z[0] = t3, t2, t1, t0
		return
	}

	u0, b := bits.Sub64(t0, mmu1_0, 0)
	u1, b := bits.Sub64(t1, mmu1_1, b)
	u2, b := bits.Sub64(t2, mmu1_2, b)
	u3, _ := bits.Sub64(t3, mmu1_3, b)

	t0, b = bits.Sub64(t0, mmu0_0, 0)
	t1, b = bits.Sub64(t1, mmu0_1, b)
	t2, b = bits.Sub64(t2, mmu0_2, b)
	t3, b = bits.Sub64(t3, mmu0_3, b)

	// Subtract the larger multiple of m if necessary

	if b == 0 {
		t3, t2, t1, t0 = u3, u2, u1, u0
	}

	z[3], z[2], z[1], z[0] = t3, t2, t1, t0
}

// SubAndReduce_c sets z, such that z==x-y mod BaseFieldSize holds.
// Assumes x and y to be weakly reduced in [0, UINT256MAX-BaseFieldSize).
//
// It guarantees the same for z.
func (z *uint256) SubAndReduce_c(x, y *uint256) {
	var borrow uint64 // only takes values 0,1

	// Set z := x - y mod 2^256
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)

	// If we do not underflow, the result is correct: It is at least as reduces as x was.
	// Otherwise, we need to add an appropriate multiple of BaseFieldSize
	if borrow != 0 {
		// NOTE: mentally rename borrow -> carry

		// If z[3] > 0xFFFFFFFF_FFFFFFFF-baseFieldSize_3, then adding 1*BaseFieldSize is guaranteed to overflow
		// Consequently, adding 1*BaseFieldSize is already enough (and the resulting z is actually fully reduced)
		if z[3] > 0xFFFFFFFF_FFFFFFFF-baseFieldSize_3 {
			z[0], borrow = bits.Add64(z[0], baseFieldSize_0, 0)
			z[1], borrow = bits.Add64(z[1], baseFieldSize_1, borrow)
			z[2], borrow = bits.Add64(z[2], baseFieldSize_2, borrow)
			z[3], _ = bits.Add64(z[3], baseFieldSize_3, borrow) // _ is guaranteed to be 1
		} else {
			// Due to constraints to y, adding 2*BaseFieldSize is guaranteed to create an overflow, so we end up with some z' == x-y mod BaseFieldSize.
			// Note that the result is in the correct range.
			// This is because the only case where we choose +=2*BaseFieldSize even though += 1* BaseFieldSize would suffice is
			// when z[3] == 0xFFFFFFFF_FFFFFFFF-baseFieldSize_3 in the condition above (and we would need to look at the other words to decide)
			// In this case, after adding +=2*BaseFieldSize, the resulting z' has
			// z'[3] == baseFieldSize_3 or z'[3]== baseFieldSize_3+1. Each case guarantees that z is in [0, UINT256MAX-BaseFieldSize)
			z[0], borrow = bits.Add64(z[0], baseFieldSizeDoubled_64_0, 0)
			z[1], borrow = bits.Add64(z[1], baseFieldSizeDoubled_64_1, borrow)
			z[2], borrow = bits.Add64(z[2], baseFieldSizeDoubled_64_2, borrow)
			z[3], _ = bits.Add64(z[3], baseFieldSizeDoubled_64_3, borrow) // _ is guaranteed to be 1
		}
	}

}

// SubAndReduce_b sets z, such that z==x-y mod BaseFieldSize holds.
// Assumes x and y to be weakly reduced in [0, 2*BaseFieldSize).
//
// It guarantees the same for z.
func (z *uint256) SubAndReduce_b(x, y *uint256) {
	var borrow uint64 // only takes values 0,1

	// Set z := x - y mod 2^256
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)

	// If we do not underflow, the result is correct: It is at least as reduces as x was.
	// Otherwise, we need to add an appropriate multiple of BaseFieldSize
	if borrow != 0 {
		// Due to constraints to y, adding 2*BaseFieldSize will guaranteed to create an overflow, so we end up with some z' == x-y mod BaseFieldSize.
		// Note that the result is in the correct range.

		// Mentally rename borrow -> carry
		z[0], borrow = bits.Add64(z[0], baseFieldSizeDoubled_64_0, 0)
		z[1], borrow = bits.Add64(z[1], baseFieldSizeDoubled_64_1, borrow)
		z[2], borrow = bits.Add64(z[2], baseFieldSizeDoubled_64_2, borrow)
		z[3], _ = bits.Add64(z[3], baseFieldSizeDoubled_64_3, borrow) // _ is one
	}

}

// SubEqAndReduce_a computes the difference z -= x mod BaseFieldSize, where z is not guaranteed to be fully reduced.
// Both inputs and outputs are only guaranteed to be in the interval [0..2**256).
func (z *uint256) SubEqAndReduce_a(x *uint256) {
	t0, b := bits.Sub64(z[0], x[0], 0)
	t1, b := bits.Sub64(z[1], x[1], b)
	t2, b := bits.Sub64(z[2], x[2], b)
	t3, b := bits.Sub64(z[3], x[3], b)

	u0, c := bits.Add64(t0, mmu1_0, 0)
	u1, c := bits.Add64(t1, mmu1_1, c)
	u2, c := bits.Add64(t2, mmu1_2, c)
	u3, _ := bits.Add64(t3, mmu1_3, c)

	v0, c := bits.Add64(t0, mmu0_0, 0)
	v1, c := bits.Add64(t1, mmu0_1, c)
	v2, c := bits.Add64(t2, mmu0_2, c)
	v3, c := bits.Add64(t3, mmu0_3, c)

	// Add the larger multiple of m if necessary

	if c == 0 {
		v3, v2, v1, v0 = u3, u2, u1, u0
	}

	// Add if underflow

	if b != 0 {
		t3, t2, t1, t0 = v3, v2, v1, v0
	}

	z[3], z[2], z[1], z[0] = t3, t2, t1, t0

}

// This function is rather slow.

// Written by Luan, comment by Gotti (to best understanding)
// Removed computation of b and d. This is completely unneeded for odd modulus.

// ModularInverse_a_NAIVEHAC computes the multiplicative Inverse of a residue, if it exists.
// z := x^-1 (mod m)
// in the case no multiplicative inverse exists, returns false, true otherwise
// Input and output values are weakly reduced to the interval [0..2**256)
func (z *uint256) ModularInverse_a_NAIVEHAC(x *uint256) bool {

	// Removed in favor or correct check:
	/*
		// check if inverse exists
		if (x[3]|x[2]|x[1]|x[0]) == 0 || // x == 0
			(m_3|m_2|m_1|m_0) == 0 || // modulus == 0
			(x[0]|m_0)&1 == 0 { // 2|gcd(u,v)

			//panic("Panic! value has no multiplicative inverse")
			return false
		}
	*/

	x.reduceBarret_fa()
	if x.IsZero() {
		return false
	}

	var (
		b, c, // Borrow & carry
		a4, a3, a2, a1, a0,
		// b4, b3, b2, b1, b0,
		c4, c3, c2, c1, c0 uint64
		// d4, d3, d2, d1, d0 uint64
	)

	u3, u2, u1, u0 := x[3], x[2], x[1], x[0]
	var v3, v2, v1, v0 uint64 = m_3, m_2, m_1, m_0 //cant use :=, go will infer as signed type, m_0 will overlflow

	a4, a3, a2, a1, a0 = 0, 0, 0, 0, 1
	// b4, b3, b2, b1, b0 = 0, 0, 0, 0, 0
	c4, c3, c2, c1, c0 = 0, 0, 0, 0, 0
	// d4, d3, d2, d1, d0 = 0, 0, 0, 0, 1

	// invariants:
	// u = a * x + b * m
	// v = c * x + d * m
	// Note that a,b,c,d can become negative, so represented using 2s complement by 5 uint64 - words  -- Gotti: Are 5 words always enough?
	// At least one of u,v odd

	done := false

	for !done {

		// invariant holds

		// If u is even, divide u, a, b by 2
		for u0&1 == 0 {

			// u /= 2
			u0 = (u0 >> 1) | (u1 << 63)
			u1 = (u1 >> 1) | (u2 << 63)
			u2 = (u2 >> 1) | (u3 << 63)
			u3 = (u3 >> 1)

			// If a or b are odd, we can adjust the representation u = a*x + b*m to make both a,b even. (u refers to the value before the above halving operation)

			//if (a0|b0)&1 == 1 {
			if a0&1 == 1 {

				a0, c = bits.Add64(a0, m_0, 0)
				a1, c = bits.Add64(a1, m_1, c)
				a2, c = bits.Add64(a2, m_2, c)
				a3, c = bits.Add64(a3, m_3, c)
				a4, _ = bits.Add64(a4, 0, c)

				// b0, b = bits.Sub64(b0, x[0], 0)
				// b1, b = bits.Sub64(b1, x[1], b)
				// b2, b = bits.Sub64(b2, x[2], b)
				// b3, b = bits.Sub64(b3, x[3], b)
				// b4, _ = bits.Sub64(b4, 0, b)
			}

			// a/=2, b/=2.
			a0 = (a0 >> 1) | (a1 << 63)
			a1 = (a1 >> 1) | (a2 << 63)
			a2 = (a2 >> 1) | (a3 << 63)
			a3 = (a3 >> 1) | (a4 << 63)
			a4 = uint64(int64(a4) >> 1)

			//b0 = (b0 >> 1) | (b1 << 63)
			//b1 = (b1 >> 1) | (b2 << 63)
			//b2 = (b2 >> 1) | (b3 << 63)
			//b3 = (b3 >> 1) | (b4 << 63)
			//b4 = uint64(int64(b4) >> 1)
			// invariant restored
		}

		// If v is even, divide v, c, d by 2
		for v0&1 == 0 {

			// v /=2
			v0 = (v0 >> 1) | (v1 << 63)
			v1 = (v1 >> 1) | (v2 << 63)
			v2 = (v2 >> 1) | (v3 << 63)
			v3 = (v3 >> 1)

			// If c or d are odd, we can adjust the representation v = c * x + d *m to make both c,d even. (v refers to the value before the above halving operation)
			//if (c0|d0)&1 == 1 {
			if c0&1 == 1 {

				c0, c = bits.Add64(c0, m_0, 0)
				c1, c = bits.Add64(c1, m_1, c)
				c2, c = bits.Add64(c2, m_2, c)
				c3, c = bits.Add64(c3, m_3, c)
				c4, _ = bits.Add64(c4, 0, c)

				// d0, b = bits.Sub64(d0, x[0], 0)
				// d1, b = bits.Sub64(d1, x[1], b)
				// d2, b = bits.Sub64(d2, x[2], b)
				// d3, b = bits.Sub64(d3, x[3], b)
				// d4, _ = bits.Sub64(d4, 0, b)
			}

			c0 = (c0 >> 1) | (c1 << 63)
			c1 = (c1 >> 1) | (c2 << 63)
			c2 = (c2 >> 1) | (c3 << 63)
			c3 = (c3 >> 1) | (c4 << 63)
			c4 = uint64(int64(c4) >> 1)

			// d0 = (d0 >> 1) | (d1 << 63)
			// d1 = (d1 >> 1) | (d2 << 63)
			// d2 = (d2 >> 1) | (d3 << 63)
			// d3 = (d3 >> 1) | (d4 << 63)
			// d4 = uint64(int64(d4) >> 1)

			// invariant restored
		}

		// Now u,v are both odd (or zero)
		// Replace u by u-v or v by v-u (depending on which one is larger)

		t0, b := bits.Sub64(u0, v0, 0)
		t1, b := bits.Sub64(u1, v1, b)
		t2, b := bits.Sub64(u2, v2, b)
		t3, b := bits.Sub64(u3, v3, b)

		if b == 0 { // u >= v

			u3, u2, u1, u0 = t3, t2, t1, t0

			a0, b = bits.Sub64(a0, c0, 0)
			a1, b = bits.Sub64(a1, c1, b)
			a2, b = bits.Sub64(a2, c2, b)
			a3, b = bits.Sub64(a3, c3, b)
			a4, _ = bits.Sub64(a4, c4, b)

			// b0, b = bits.Sub64(b0, d0, 0)
			// b1, b = bits.Sub64(b1, d1, b)
			// b2, b = bits.Sub64(b2, d2, b)
			// b3, b = bits.Sub64(b3, d3, b)
			// b4, _ = bits.Sub64(b4, d4, b)

		} else { // v > u

			v0, b = bits.Sub64(v0, u0, 0)
			v1, b = bits.Sub64(v1, u1, b)
			v2, b = bits.Sub64(v2, u2, b)
			v3, _ = bits.Sub64(v3, u3, b)

			c0, b = bits.Sub64(c0, a0, 0)
			c1, b = bits.Sub64(c1, a1, b)
			c2, b = bits.Sub64(c2, a2, b)
			c3, b = bits.Sub64(c3, a3, b)
			c4, _ = bits.Sub64(c4, a4, b)

			// d0, b = bits.Sub64(d0, b0, 0)
			// d1, b = bits.Sub64(d1, b1, b)
			// d2, b = bits.Sub64(d2, b2, b)
			// d3, b = bits.Sub64(d3, b3, b)
			// d4, _ = bits.Sub64(d4, b4, b)
		}

		if (u3 | u2 | u1 | u0) == 0 {
			done = true
		}
	}

	// Cannot happen for m prime, we already checked for zero, so removed.

	/*
		if (v3 | v2 | v1 | (v0 - 1)) != 0 { // gcd(z,m) != 1
			//z[3], z[2], z[1], z[0] = 0, 0, 0, 0
			//panic("Panic! value has no multiplicative inverse")
			return false
		}
	*/

	// We effectively reduce the (signed!) 5-word c modulu m to a 256-bit number.
	// This assumes that int(c4) is small in absolute value. Why is that so? -- Gotti

	// Add or subtract modulus to find 256-bit inverse (<= 2 iterations expected)

	for (c4 >> 63) != 0 {
		c0, c = bits.Add64(c0, m_0, 0)
		c1, c = bits.Add64(c1, m_1, c)
		c2, c = bits.Add64(c2, m_2, c)
		c3, c = bits.Add64(c3, m_3, c)
		c4, _ = bits.Add64(c4, 0, c)
	}

	for c4 != 0 {
		c0, b = bits.Sub64(c0, m_0, 0)
		c1, b = bits.Sub64(c1, m_1, b)
		c2, b = bits.Sub64(c2, m_2, b)
		c3, b = bits.Sub64(c3, m_3, b)
		c4, _ = bits.Sub64(c4, 0, b)
	}

	z[3], z[2], z[1], z[0] = c3, c2, c1, c0
	return true
}

// reduce_ca replaces z by some number z' with z' == z mod BaseFieldSize.
//
// z' is guaranteed to be in [0, UINT256MAX-BaseFieldSize)
func (z *uint256) reduce_ca() {
	var borrow uint64
	// Note: if z.words[3] == m_64_3, we may or may not be able to reduce, depending on the other words.
	// At any rate, we do not really need to, so we don't check.
	if z[3] > baseFieldSize_3 {
		z[0], borrow = bits.Sub64(z[0], baseFieldSize_0, 0)
		z[1], borrow = bits.Sub64(z[1], baseFieldSize_1, borrow)
		z[2], borrow = bits.Sub64(z[2], baseFieldSize_2, borrow)
		z[3], _ = bits.Sub64(z[3], baseFieldSize_3, borrow) // _ is guaranteed to be 0
	}
}

// reduce_fb replaces z by some number z' with z' == z mod BaseFieldSize. Assumes z is already weakly reduced.
//
// More precisely, z' is guaranteed to be in [0, BaseFieldSize), provided z is in [0, 2*BaseFieldSize)
func (z *uint256) reduce_fb() {
	if !z.is_fully_reduced() {

		var borrow uint64
		z[0], borrow = bits.Sub64(z[0], baseFieldSize_0, 0)
		z[1], borrow = bits.Sub64(z[1], baseFieldSize_1, borrow)
		z[2], borrow = bits.Sub64(z[2], baseFieldSize_2, borrow)
		z[3], _ = bits.Sub64(z[3], baseFieldSize_3, borrow)
	}

}

// is_fully_reduced checks whether z is in [0, BaseFieldSize)
func (z *uint256) is_fully_reduced() bool {
	// Workaround for Go's lack of const-arrays. Hoping for smart-ish compiler.
	// Note that the RHS is const and the left-hand-side is local and never written to after initialization.
	var baseFieldSize_copy uint256 = [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}
	for i := int(3); i >= 0; i-- {
		if z[i] < baseFieldSize_copy[i] {
			return true
		} else if z[i] > baseFieldSize_copy[i] {
			return false
		}
	}
	// if we get here, z.words == BaseFieldSize
	return false
}

// Barrett reduction from the Handbook of Applied Cryptography.
// Used in the MulEq and SquareEq to execute a weak reduction to the inverval [0..2**256]
func (z *uint256) ReduceUint512ToUint256_a(x uint512) {
	// q1 = x/2^192
	x0 := x[3]
	x1 := x[4]
	x2 := x[5]
	x3 := x[6]
	x4 := x[7]

	// q2 = q1 * mu; q3 = q2 / 2^320

	var q0, q1, q2, q3, q4, q5, t0, t1, c uint64

	q0, _ = bits.Mul64(x3, re_0)
	q1, t0 = bits.Mul64(x4, re_0)
	q0, c = bits.Add64(q0, t0, 0)
	q1, _ = bits.Add64(q1, 0, c)

	t1, _ = bits.Mul64(x2, re_1)
	q0, c = bits.Add64(q0, t1, 0)
	q2, t0 = bits.Mul64(x4, re_1)
	q1, c = bits.Add64(q1, t0, c)
	q2, _ = bits.Add64(q2, 0, c)

	t1, t0 = bits.Mul64(x3, re_1)
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	q2, _ = bits.Add64(q2, 0, c)

	t1, t0 = bits.Mul64(x2, re_2)
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	q3, t0 = bits.Mul64(x4, re_2)
	q2, c = bits.Add64(q2, t0, c)
	q3, _ = bits.Add64(q3, 0, c)

	t1, _ = bits.Mul64(x1, re_2)
	q0, c = bits.Add64(q0, t1, 0)
	t1, t0 = bits.Mul64(x3, re_2)
	q1, c = bits.Add64(q1, t0, c)
	q2, c = bits.Add64(q2, t1, c)
	q3, _ = bits.Add64(q3, 0, c)

	t1, _ = bits.Mul64(x0, re_3)
	q0, c = bits.Add64(q0, t1, 0)
	t1, t0 = bits.Mul64(x2, re_3)
	q1, c = bits.Add64(q1, t0, c)
	q2, c = bits.Add64(q2, t1, c)
	q4, t0 = bits.Mul64(x4, re_3)
	q3, c = bits.Add64(q3, t0, c)
	q4, _ = bits.Add64(q4, 0, c)

	t1, t0 = bits.Mul64(x1, re_3)
	q0, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	t1, t0 = bits.Mul64(x3, re_3)
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q4, _ = bits.Add64(q4, 0, c)

	t1, t0 = bits.Mul64(x0, re_4)
	_, c = bits.Add64(q0, t0, 0)
	q1, c = bits.Add64(q1, t1, c)
	t1, t0 = bits.Mul64(x2, re_4)
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q5, t0 = bits.Mul64(x4, re_4)
	q4, c = bits.Add64(q4, t0, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(x1, re_4)
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	t1, t0 = bits.Mul64(x3, re_4)
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q5, _ = bits.Add64(q5, 0, c)

	// Drop the fractional part of q3

	q0 = q1
	q1 = q2
	q2 = q3
	q3 = q4
	q4 = q5

	// r1 = x mod 2^320

	x0 = x[0]
	x1 = x[1]
	x2 = x[2]
	x3 = x[3]
	x4 = x[4]

	// r2 = q3 * m mod 2^320

	var r0, r1, r2, r3, r4 uint64

	r4, r3 = bits.Mul64(q0, m_3)
	_, t0 = bits.Mul64(q1, m_3)
	r4, _ = bits.Add64(r4, t0, 0)

	t1, r2 = bits.Mul64(q0, m_2)
	r3, c = bits.Add64(r3, t1, 0)
	_, t0 = bits.Mul64(q2, m_2)
	r4, _ = bits.Add64(r4, t0, c)

	t1, t0 = bits.Mul64(q1, m_2)
	r3, c = bits.Add64(r3, t0, 0)
	r4, _ = bits.Add64(r4, t1, c)

	t1, r1 = bits.Mul64(q0, m_1)
	r2, c = bits.Add64(r2, t1, 0)
	t1, t0 = bits.Mul64(q2, m_1)
	r3, c = bits.Add64(r3, t0, c)
	r4, _ = bits.Add64(r4, t1, c)

	t1, t0 = bits.Mul64(q1, m_1)
	r2, c = bits.Add64(r2, t0, 0)
	r3, c = bits.Add64(r3, t1, c)
	_, t0 = bits.Mul64(q3, m_1)
	r4, _ = bits.Add64(r4, t0, c)

	t1, r0 = bits.Mul64(q0, m_0)
	r1, c = bits.Add64(r1, t1, 0)
	t1, t0 = bits.Mul64(q2, m_0)
	r2, c = bits.Add64(r2, t0, c)
	r3, c = bits.Add64(r3, t1, c)
	_, t0 = bits.Mul64(q4, m_0)
	r4, _ = bits.Add64(r4, t0, c)

	t1, t0 = bits.Mul64(q1, m_0)
	r1, c = bits.Add64(r1, t0, 0)
	r2, c = bits.Add64(r2, t1, c)
	t1, t0 = bits.Mul64(q3, m_0)
	r3, c = bits.Add64(r3, t0, c)
	r4, _ = bits.Add64(r4, t1, c)

	// r = r1 - r2

	var b uint64

	r0, b = bits.Sub64(x0, r0, 0)
	r1, b = bits.Sub64(x1, r1, b)
	r2, b = bits.Sub64(x2, r2, b)
	r3, b = bits.Sub64(x3, r3, b)
	r4, b = bits.Sub64(x4, r4, b)

	// if r<0 then r+=m

	x0, c = bits.Add64(r0, m_0, 0)
	x1, c = bits.Add64(r1, m_1, c)
	x2, c = bits.Add64(r2, m_2, c)
	x3, c = bits.Add64(r3, m_3, c)
	x4, _ = bits.Add64(r4, 0, c)

	// commit if borrow
	if b != 0 {
		r4, r3, r2, r1, r0 = x4, x3, x2, x1, x0
	}

	// incomplete reduction is possible if m < 2^256/3
	if m_3 < 0x5555555555555555 {
		z[3], z[2], z[1], z[0] = r3, r2, r1, r0
		return
	}

	// q = r - m
	x0, b = bits.Sub64(r0, m_0, 0)
	x1, b = bits.Sub64(r1, m_1, b)
	x2, b = bits.Sub64(r2, m_2, b)
	x3, b = bits.Sub64(r3, m_3, b)
	x4, b = bits.Sub64(r4, 0, b)

	// commit if no borrow
	if b == 0 {
		r4, r3, r2, r1, r0 = x4, x3, x2, x1, x0
	}

	// q = r - m
	x0, b = bits.Sub64(r0, m_0, 0)
	x1, b = bits.Sub64(r1, m_1, b)
	x2, b = bits.Sub64(r2, m_2, b)
	x3, b = bits.Sub64(r3, m_3, b)
	_, b = bits.Sub64(r4, 0, b)

	// commit if no borrow
	if b == 0 {
		r3, r2, r1, r0 = x3, x2, x1, x0
	}

	z[3], z[2], z[1], z[0] = r3, r2, r1, r0

}

// reduceBarret_fa computes computes the canonical form z mod m, storing back in z
func (z *uint256) reduceBarret_fa() {

	// NB: Most variable names in the comments match the pseudocode for
	// 	Barrett reduction in the Handbook of Applied Cryptography.

	var x0, x1, x2, x3, x4, r0, r1, r2, r3, r4, q3, t0, t1, c uint64

	// q1 = x/2^192
	// q2 = q1 * mu; q3 = q2 / 2^320

	q3, _ = bits.Mul64(z[3], re_4)

	// r1 = x mod 2^320 = x

	x0 = z[0]
	x1 = z[1]
	x2 = z[2]
	x3 = z[3]
	x4 = 0

	// r2 = q3 * m mod 2^320

	r2, r1 = bits.Mul64(q3, m_1)
	r4, r3 = bits.Mul64(q3, m_3)

	t1, r0 = bits.Mul64(q3, m_0)
	r1, c = bits.Add64(r1, t1, 0)
	t1, t0 = bits.Mul64(q3, m_2)
	r2, c = bits.Add64(r2, t0, c)
	r3, c = bits.Add64(r3, t1, c)
	r4, _ = bits.Add64(r4, 0, c)

	// r = r1 - r2 = x - r2

	// Note: x < 2^256
	//    => q3 <= x/m
	//    => q3*m <= x
	//    => r2 <= x
	//    => r >= 0

	var b uint64

	r0, b = bits.Sub64(x0, r0, 0)
	r1, b = bits.Sub64(x1, r1, b)
	r2, b = bits.Sub64(x2, r2, b)
	r3, b = bits.Sub64(x3, r3, b)
	r4, _ = bits.Sub64(x4, r4, b)

	for {
		// if r>=m then r-=m

		x0, b = bits.Sub64(r0, m_0, 0)
		x1, b = bits.Sub64(r1, m_1, b)
		x2, b = bits.Sub64(r2, m_2, b)
		x3, b = bits.Sub64(r3, m_3, b)
		x4, b = bits.Sub64(r4, 0, b)

		if b != 0 {
			break
		}

		// commit if no borrow (r1 >= r2 + m)
		r4, r3, r2, r1, r0 = x4, x3, x2, x1, x0
	}
	z[3], z[2], z[1], z[0] = r3, r2, r1, r0
}

// NOTE: Inconsistent syntax, since it returns a value. Therefore deprecated

// ComputeModularNegative_Weak_f computes the negation (additive inverse) of a number modulo m.
// input values don't need to be fully reduced.
//
// DEPRECATED
func (z *uint256) ComputeModularNegative_Weak_f() (r uint256) {
	t0, b := bits.Sub64(mmu0_0, z[0], 0)
	t1, b := bits.Sub64(mmu0_1, z[1], b)
	t2, b := bits.Sub64(mmu0_2, z[2], b)
	t3, b := bits.Sub64(mmu0_3, z[3], b)

	if b == 0 {
		r[3], r[2], r[1], r[0] = t3, t2, t1, t0
		return r
	}

	t0, b = bits.Sub64(mmu1_0, z[0], 0)
	t1, b = bits.Sub64(mmu1_1, z[1], b)
	t2, b = bits.Sub64(mmu1_2, z[2], b)
	t3, _ = bits.Sub64(mmu1_3, z[3], b)

	r[3], r[2], r[1], r[0] = t3, t2, t1, t0
	return

}

// DoubleEqAndReduce_a doubles a number modulo m, weakly reduced reduce to to the interval [0..2**256)
// input values don't need to be fully reduced.
func (z *uint256) DoubleEqAndReduce_a() {

	t0, c := bits.Add64(z[0], z[0], 0)
	t1, c := bits.Add64(z[1], z[1], c)
	t2, c := bits.Add64(z[2], z[2], c)
	t3, c := bits.Add64(z[3], z[3], c)

	u0, b := bits.Sub64(t0, mmu1_0, 0)
	u1, b := bits.Sub64(t1, mmu1_1, b)
	u2, b := bits.Sub64(t2, mmu1_2, b)
	u3, _ := bits.Sub64(t3, mmu1_3, b)

	v0, b := bits.Sub64(t0, mmu0_0, 0)
	v1, b := bits.Sub64(t1, mmu0_1, b)
	v2, b := bits.Sub64(t2, mmu0_2, b)
	v3, b := bits.Sub64(t3, mmu0_3, b)

	// Subtract the larger multiple of m if necessary
	if b == 0 {
		v3, v2, v1, v0 = u3, u2, u1, u0
	}

	// Subtract if overflow
	if c != 0 {
		t3, t2, t1, t0 = v3, v2, v1, v0
	}

	z[3], z[2], z[1], z[0] = t3, t2, t1, t0
}
