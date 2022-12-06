package fieldElements

import (
	"math/bits"
)

// This file is part of the implementation of the Uint256 (and also a similar Uint512) data type.
// Uint256 is a 256-bit unsigned integer data type used (mostly internally) to implement our field element types.
//
// Note that a Uint256 is an integer, not a residue, so arithmetic is as for usual uints, i.e. modulo 2^256.
// Funtions and Methods that operate on Uint256's and perform modular arithmetic explicitly say so in their description and function name.
//
// The set of exported functions and methods for these is not particularly stable;
// we export it mostly to enable certain advanced optimizations outside the package (mixed Montgomery multiplication being the main one) for users who want to perform extensive computations in the base field.

// Note that the code is split into 3 parts:
//   uint256.go (integer arithmetic / arithmetic modulo 2^256)
//   uint256_modular.go (arithmetic that works modulo BaseFieldSize)
//   uint256_montgomery.go (Montgomery arithmetic)
//

// montgomery_iteration performs t := (t / 2**64) + x * y weakMod BaseFieldSize.
// Note that the division by 2**64 is done modulo BaseFieldSize.
//
// The reduction quaility is such that if for the input, (t>>64) and x are c-reduced, then for the output (t>>64) is c-reduced.
//
// For the input, this means
//
//	t>>64 + BaseFieldSize < 2**256 and
//	x + BaseFieldSize < 2**256,
func montgomery_iteration(t *[5]uint64, x *Uint256, y uint64) {
	var low, high, carry1, carry2, carry3, carry4 uint64

	// Change t to an equivalent representation modulo BaseFieldSize, s.t. t[0] == 0

	// If t[0] == 0, we don't need to do anything (and the algorithm below would actually be wrong)
	if t[0] != 0 {
		q := t[0] * negativeInverseModulus_uint64 // computation will overflow, so this is performed modulo 2**64. This is exactly as desired.
		// q is chosen, s.t. t + q*BaseFieldSize == 0 mod 2**64.
		// We now add q*BaseFieldSize to t.

		high, _ = bits.Mul64(q, baseFieldSize_0)
		// t[0], carry = bits.Add64(t[0], _, 0) for _ from the line above gives t[0] == 0, carry==1 by construction; we can omit this.
		// t[0] = 0 is omitted, because we will later write to t[0] anyway.
		t[1], carry1 = bits.Add64(t[1], high, 1) // After this, carry1 needs to go in t[2]

		high, low = bits.Mul64(q, baseFieldSize_1)
		t[1], carry2 = bits.Add64(t[1], low, 0)       // After this, carry2 needs to go in t[2]
		t[2], carry2 = bits.Add64(t[2], high, carry2) // After this, carry2 needs to go in t[3]

		high, low = bits.Mul64(q, baseFieldSize_2)
		t[2], carry1 = bits.Add64(t[2], low, carry1)  // After this, carry1 needs to go in t[3]
		t[3], carry1 = bits.Add64(t[3], high, carry1) // After this, carry1 needs to go in t[4]

		high, low = bits.Mul64(q, baseFieldSize_3)
		t[3], carry2 = bits.Add64(t[3], low, carry2)    // After this, carry2 needs to go in t[4]
		t[4], _ = bits.Add64(t[4], high+carry1, carry2) // _ == 0.
		// The last carry is_ = 0 here:
		// In fact, we know for the input q < 2**64  and t>>64 + BaseFieldSize < 2**256, so we get:
		// t < 2**320 - 2**64 * BaseFieldSize
		// => (t + q*BaseFieldSize) < 2**320 + BaseFieldSize * (-2**64 + q) <= 2**320 - BaseFieldSize.
	}
	// Mentally apply t[0] = 0. (We omit this, as t[0] will be overwritten in the next operation, but it helps to understand)
	// After this, t now stores an equivalent representation (i.e. differing by a multiple of BaseFieldSize) of the values that was given for t as input.

	// Now compute (t >> 64) + x * y from the current value of t. We do this in one go, as the >>64 just means reading from a higher index.
	// Bounds analysis:
	//   t >> 64 < 2**256 (because t has 320 bits)
	//   x*y <= (2**256 - BaseFieldSize - 1) * (2**64 - 1)
	//   => t + x*y < 2**256 + 2**320 - 2**64 BaseFieldSize - 2**64 - 2**256 + BaseFieldSize + 1
	//   => t + x*y < (2**320 - 2**64 BaseFieldSize) - 2**64 + BaseFieldSize + 1 < 2**320 - 2**64 BaseFieldSize.
	//   => (t + x*y) >> 64 < 2**256 - BaseFieldSize (Note that normally, a < b only implies a>>1 <= b>>1, but since the rhs above was divisible by 2**64, we actually get <)
	// This means (t+ x * y) >> 64 is c-reduced
	carry1, t[0] = bits.Mul64(x[0], y)       // Large carry1 -> t[1]
	t[0], carry2 = bits.Add64(t[0], t[1], 0) // t[0] finished writing, t[1] finished reading, binary carry2 -> t[1]

	carry3, t[1] = bits.Mul64(x[1], y)              // large carry3 -> t[2]
	t[1], carry2 = bits.Add64(t[1], carry1, carry2) // binary carry2 -> t[2]
	t[1], carry1 = bits.Add64(t[1], t[2], 0)        // binary carry1 -> t[2], t[1] finished writing, t[2] finished reading

	carry4, t[2] = bits.Mul64(x[2], y)              // large carry4 -> t[3]
	t[2], carry2 = bits.Add64(t[2], carry3, carry2) // binary carry2 -> t[3]
	t[2], carry1 = bits.Add64(t[2], t[3], carry1)   // binary carry1 -> t[3], t[2] finished writing, t[3] finished reading

	carry3, t[3] = bits.Mul64(x[3], y)              // large carry3 -> t[4]
	t[3], carry2 = bits.Add64(t[3], carry4, carry2) // binary carry2 -> t[4]
	t[3], carry1 = bits.Add64(t[3], t[4], carry1)   // bianry carry1 -> t[4]

	t[4] = carry3 + carry1 + carry2 // cannot overflow by the above analysis. (in fact, this is true unconditionally even without bound on x or the input t. We need a stronger bound than no-overflow, though)
}

// MulMontgomery_c performs Montgomery multiplication, i.e. z := x * y / 2**256 mod BaseFieldSize.
//
// We assume that x and y are c-reduced, i.e. x,y < 2**256 - BaseFieldSize and we guaranteed the same for z.
func (z *Uint256) MulMontgomery_c(x, y *Uint256) {
	z.mulMontgomery_Unrolled_c(x, y)
}

// unrolled Montgomery multiplication:

// mulMontgomery_Unrolled_c performs Montgomery multiplication, i.e. z := x * y / 2**256 mod BaseFieldSize.
//
// We assume that x and y are c-reduced, i.e. x,y < 2**256 - BaseFieldSize and we guaranteed the same for z.
// This implements MulMontgomery_c. (The indirection is because we have a slightly different version for comparison (that only differs in unrolling and variable naming/reuse)
func (z *Uint256) mulMontgomery_Unrolled_c(x, y *Uint256) {
	var temp [5]uint64

	// compute z as x*y / r^4 bmod BaseFieldSize with r==2^64
	// To do so, note that x*y == x*(y[0] + ry[1]+r^2y[2]+r^3y[3]), so
	// x*y / r^4 == 1/r^4 x*y[0] + 1/r^3 x*y[1] + 1/r^2 x*y[2] + 1/r x*y[3],
	// which can be computed as ((((x*y[0]/r + x*y[1]) /r + x*y[1]) / r + x*y[2]) /r) + x*y[3]) /r

	LongMulUint64(&temp, x, y[0]) // temp == x*y[0]
	// NOTE: (temp >> 64) < x, so (temp>>64) is c-reduced. and montgomery_iteration will preserve this.
	montgomery_iteration(&temp, x, y[1]) // temp == (x*y[0] / r) + x*y[1]
	montgomery_iteration(&temp, x, y[2]) // temp == ((x*y[0] / r) + x*y[1])/r + x*y[2]
	montgomery_iteration(&temp, x, y[3]) // temp == (((x*y[0] / r) + x*y[1])/r + x*y[2])/r + x*y[3]
	// We need to divide by r mod BaseFieldSize. This is just another montgomery_iteration, but with y == 0 and we can write directly to z (so the second part is done by just writing to the correct z[i]).
	if temp[0] == 0 {
		z[0] = temp[1]
		z[1] = temp[2]
		z[2] = temp[3]
		z[3] = temp[4]
	} else {
		var carry1, carry2, high, low uint64
		temp[0] *= negativeInverseModulus_uint64

		high, _ = bits.Mul64(temp[0], baseFieldSize_0)
		z[0], carry1 = bits.Add64(temp[1], high, 1)

		high, low = bits.Mul64(temp[0], baseFieldSize_1)
		z[0], carry2 = bits.Add64(z[0], low, 0)
		z[1], carry2 = bits.Add64(temp[2], high, carry2)

		high, low = bits.Mul64(temp[0], baseFieldSize_2)
		z[1], carry1 = bits.Add64(z[1], low, carry1)
		z[2], carry1 = bits.Add64(temp[3], high, carry1)

		high, low = bits.Mul64(temp[0], baseFieldSize_3)
		z[2], carry2 = bits.Add64(z[2], low, carry2)
		z[3], _ = bits.Add64(temp[4], high+carry1, carry2) // _ == 0 for the same
	}
	z.Reduce_ca()
}

// SquareMontgomery_c performs Montgomery squaring, i.e. z = x^2 / 2^256 mod BaseFieldSize
//
// We assume that x is c-reduced, i.e. x < 2^256 - BaseFieldSize, and we guarantee the same for z.
func (z *Uint256) SquareMontgomery_c(x *Uint256) {
	z.mulMontgomery_Unrolled_c(x, x)
}

// FromMontgomeryRepresentation undoes Montgomery representation.
//
// This means that z.FromMontgomeryRepresentation(x) computes z := x / 2**256 mod BaseFieldSize
func (z *Uint256) FromMontgomeryRepresentation_fc(x *Uint256) {
	// equivalent to
	// z.MulMontgomery_c(x, [1,0,0,0])
	// z.Reduce_f()
	var temp [4]uint64 = *x
	var q, low, carry1, carry2 uint64
	for i := 0; i < 4; i++ {
		// divide by 2**64 mod BaseFieldSize:
		// This works essentially the same as montgomery_step, but we don't need 5 uint words.

		// invariant here:
		// temp == x * 2**(-64*i) mod BaseFieldSize
		// temp is c-reduced, i.e. temp < 2**256 - BaseFieldSize

		q = temp[0] * negativeInverseModulus_uint64
		if q == 0 {
			temp[0], temp[1], temp[2], temp[3] = temp[1], temp[2], temp[3], 0
		} else {
			// compute temp = (temp + q * baseFieldSize) / 2**64. This division is exact by construction.
			temp[0], _ = bits.Mul64(q, baseFieldSize_0)
			temp[0], carry1 = bits.Add64(temp[0], temp[1], 1) //carry1 -> temp[1]

			temp[1], low = bits.Mul64(q, baseFieldSize_1)
			temp[0], carry2 = bits.Add64(temp[0], low, 0)
			temp[1], carry2 = bits.Add64(temp[1], temp[2], carry2) // carry2 -> temp[2]

			temp[2], low = bits.Mul64(q, baseFieldSize_2)
			temp[1], carry1 = bits.Add64(temp[1], low, carry1)
			temp[2], carry1 = bits.Add64(temp[2], temp[3], carry1) // carry1 -> temp[3]

			temp[3], low = bits.Mul64(q, baseFieldSize_3)
			temp[2], carry2 = bits.Add64(temp[2], low, carry2)
			temp[3] += carry2 + carry1

			// Note: temp <= (2**256 - BaseFieldSize - 1 + (2**64 - 1) * BaseFieldSize) / 2**64 < 2**192 + BaseFieldSize < 2**256 - BaseFieldSize
		}

		// invariant here:
		// temp == x * 2**(-64*(i+1)) mod BaseFieldSize
		// temp is c-reduced

	}
	*z = temp // we could write directly to z by unrolling the above loop and write to z in the last iteration
	z.Reduce_fb()
}

// ConvertToMontgomeryRepresentation sets z := x * 2**256 mod BaseFieldSize.
func (z *Uint256) ConvertToMontgomeryRepresentation_c(x *Uint256) {
	// TODO: unroll
	z.mulMontgomery_Unrolled_c(x, &twoTo512ModBaseField_uint256)
}

const uint256MontgomeryExponentiationAlgUsed = "Square and Multiply" // displayed in benchmark log

// ModularExponentiationMontgomery_fa sets z := base^exponent modulo BaseFieldSize, where z and base are both in Montgomery form.
//
// By convention, 0^0 is 1 here.
func (z *Uint256) ModularExponentiationMontgomery_fa(base *Uint256, exponent *Uint256) {
	// NOTE: Square and multiply takes around 256 (or 255) * 1.5 approx. 380 multiplications/squarings
	// Sliding window takes around 310
	// Still, Square and multiply is faster according to my benchmarks.
	base.modularExponentiationSquareAndMultiplyMontgomery_fa(base, exponent)
}

// modularExponentiationSquareAndMultiplyMontgomery_fa implements ModularExponentiationMontgomery_fa using naive square&multiply
func (z *Uint256) modularExponentiationSquareAndMultiplyMontgomery_fa(base *Uint256, exponent *Uint256) {
	// simple sliding window exponentiation
	if exponent.IsZero() {
		*z = twoTo256ModBaseField_uint256 // montgomery representation of 1
		return
	}

	var acc Uint256 = *base
	// We need to reduce here, because SquareMontgomery_c and MulMontgomery_c require their inputs c-reduced.
	acc.Reduce_ca()

	// simple square-and-multiply. This is not really optimized (no sliding window etc)
	L := exponent.BitLen()

	for i := int(L - 2); i >= 0; i-- {
		// acc == base^(exponent >> (i+1))
		acc.SquareMontgomery_c(&acc)

		if exponent[i/64]&(1<<(i%64)) != 0 {
			acc.mulMontgomery_Unrolled_c(&acc, base)

		}
		// acc == base^(exponent >> i)
	}
	acc.Reduce_fb()
	*z = acc
}

// modularExponentiationSquareAndMultiplyMontgomery_fa implements ModularExponentiationMontgomery_fa using a sliding window exponentiation
func (z *Uint256) modularExponentiationSlidingWindowMontgomery_fa(base *Uint256, exponent *Uint256) {

	// sliding window exponentiation:

	// we use a window size depending on the bitlength of the exponent. 4 is the default.
	// Note that this takes care of the exponent == 0 special case as well.

	var windowSize uint8 = 4
	L := exponent.BitLen()
	if L <= 80 { // for short exponents, we modify the window size.
		if L == 0 {
			*z = twoTo256ModBaseField_uint256 // montgomery representation of 1
			return
		} else if L == 1 {
			*z = *base
			z.Reduce_ca()
			z.Reduce_fb()
			return // window size 1 is better than 2 for very small window sizes.
		} else if L <= 6 {
			windowSize = 1
		} else if L <= 24 {
			windowSize = 2
		} else { // 24 < L <= 80
			windowSize = 3
		}
	}

	var acc Uint256 = *base
	// We need to reduce here, because SquareMontgomery_c and MulMontgomery_c require their inputs c-reduced.
	acc.Reduce_ca()
	decomp := exponent.SlidingWindowDecomposition(windowSize)

	// precompute small odd powers of base
	precomputeSize := 1 << (windowSize - 1)
	var precomputes [8]Uint256 // = make([]Uint256, precomputeSize) // precomputes[i] holds base^(2*i+1)

	precomputes[0] = acc
	acc.SquareMontgomery_c(&acc) // acc = base^2 (will be overwritten later)

	for i := 1; i < precomputeSize; i++ {
		precomputes[i].mulMontgomery_Unrolled_c(&precomputes[i-1], &acc)

	}

	// decomp gives a decomposition exponent == sum_i decomp[i].exp << decomp[i].pos (with decomp[i].pos a strictly decreasing sequence)
	// len(decomp) > 0
	exponentRemaining := decomp[0].pos
	acc = precomputes[decomp[0].exp>>1]

	// The following invariant holds before and after each iteration:
	// acc^(1<<exponentRemaining) == base^(sum_{i=0}^{sth.} decomp[i].exp << decomp[i].pos), where
	// the sum ranges over those entries of decomp that are already processed. (the index-0 one before we start the loop, hence the range decomp[1:] )

	for _, decompositionEntry := range decomp[1:] { // j = _ + 1
		for i := uint(0); i < exponentRemaining-decompositionEntry.pos; i++ { // note the difference is guaranteed to be > 0
			acc.SquareMontgomery_c(&acc)

		}
		exponentRemaining = decompositionEntry.pos
		acc.mulMontgomery_Unrolled_c(&acc, &precomputes[decompositionEntry.exp>>1])

	}

	// acc^(1<<exponentRemaining) = base^(sum_i decomp[i].exp << decomp[i].pos) = base^exponent now

	// square exponentRemainingg many times
	for i := uint(0); i < exponentRemaining; i++ {
		acc.SquareMontgomery_c(&acc)

	}

	// reduce fully
	acc.Reduce_fb()
	*z = acc

}

/********************
// DEPRECATED FUNCTIONS:
// These are less efficient, but more readable variants.
**********************/

// mul_four_one_64 multiplies a 4x64 bit number by a 1x64 bit number. The result is 5x64 bits, split as 1x64 (low) + 4x64 (high), everything low-endian.
//
// DEPRECATED in favor of version taking a pointer argument to store the result.
func mul_four_one_64(x *Uint256, y uint64) (low uint64, high Uint256) {
	var carry, mul_result_low uint64

	high[0], low = bits.Mul64(x[0], y)

	high[1], mul_result_low = bits.Mul64(x[1], y)
	high[0], carry = bits.Add64(high[0], mul_result_low, 0)

	high[2], mul_result_low = bits.Mul64(x[2], y)
	high[1], carry = bits.Add64(high[1], mul_result_low, carry)

	high[3], mul_result_low = bits.Mul64(x[3], y)
	high[2], carry = bits.Add64(high[2], mul_result_low, carry)

	high[3] += carry
	return
}

// DEPRECATED:

// ToNonMontgomery_fc undoes Montgomery representation.
//
// More precisely, z.ToNonMontgomery_fc() returns z*(1/2**256) mod BaseFieldSize
//
// DEPRECATED / MOVE_TO_TESTING: Performance loss from returning an uint256 is worse (probably due to allocation) than writing to receiver. FromMontgomery is better.
func (z *Uint256) ToNonMontgomery_fc() Uint256 {
	// What we need to do here is equivalent to
	// temp.Mul(z, [1,0,0,0])  // where the [1,0,0,0] is the Montgomery representation of the number 1/r.
	// temp.Normalize()
	// return temp.words

	var reducer uint64 = z[0]
	var temp Uint256 = [4]uint64{0: z[1], 1: z[2], 2: z[3], 3: 0}

	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}
	reducer = temp.ShiftRightEq_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	reducer = temp.ShiftRightEq_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	reducer = temp.ShiftRightEq_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	temp.Reduce_ca()
	temp.Reduce_fb()
	return temp
}

// DEPRECATED in favor of (more efficient) version above.

// mulMontgomerySlow_c performs z := x * y / 2**256 WeakMod BaseFieldSize. (Division is modulo as well)
//
// If all inputs are c-reduced (i.e. x+BaseFieldSize < 2**256 etc), so is the output
// This is a slower implementation of MulMontgomery_c than the one given above.
func (z *Uint256) mulMontgomerySlow_c(x, y *Uint256) {

	/*
		We perform Montgomery multiplication, i.e. we need to find x*y / r^4 bmod BaseFieldSize with r==2^64
		To do so, note that x*y == x*(y[0] + ry[1]+r^2y[2]+r^3y[3]), so
		x*y / r^4 == 1/r^4 x*y[0] + 1/r^3 x*y[1] + 1/r^2 x*y[2] + 1/r x*y[3],
		which can be computed as ((((x*y[0]/r + x*y[1]) /r + x*y[1]) / r + x*y[2]) /r) + x*y[3]) /r
		i.e by interleaving adding x*y[i] and dividing by r (everything is mod BaseFieldSize).
		We store the intermediate results in temp

		Dividing by r modulo BaseFieldSize is done by adding a suitable multiple of BaseFieldSize
		(which we can always do mod BaseFieldSize) s.t. the result is divisible by r and just dividing by r.
		This has the effect of reducing the size of number, thereby performing a (partial) modular reduction (Montgomery's trick)
	*/

	// temp holds the result of computation so far. We only write into z at the end, because z might alias x or y.
	var temp Uint256

	// -1/Modulus mod r.
	const negativeInverseModulus = (0xFFFFFFFF_FFFFFFFF * 0x00000001_00000001) % (1 << 64)
	const negativeInverseModulus_uint uint64 = negativeInverseModulus

	var reducer uint64

	reducer, temp = mul_four_one_64(x, y[0]) // NOTE: temp <= B - floor(B/r) - 1  <= B + floor(M/r), see overflow analysis below

	// If reducer == 0, then temp == x*y[0]/r.
	// Otherwise, we need to compute temp = ([temp, reducer] + BaseFieldSize * (reducer * negativeInverseModulus mod r)) / r
	// Note that we know exactly what happens in the least significant uint64 in the addition (result is 0, carry is 1).
	// Be aware that carry 1 relies on reducer != 0, hence the if reducer!=0 condition
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, x, y[1])
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, x, y[2])
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, x, y[3])
	if reducer != 0 {
		// TODO: Store directly into z?
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	/*
		Overflow analysis:
		Let B:= 2^256 - BaseFieldSize - 1. We know that 0<= x,y <= B and need to ensure that 0<=z<=B to maintain our invariants:

		(1) If temp <= B + M (which is 2^256 - 1, so this condition is somewhat vacuous) and x <= B, then after applying add_mul_shift_64(&temp, x, y), we have
		temp <= (B + M + B * (r-1)) / r <= B + floor(M/r)

		(2) If temp <= B + floor(M/r) is satisfied and we compute montgomery_step_64(&temp, something), we afterwards obtain
		temp <= B + floor(M/r) + floor(M*(r-1)/r) + 1 == B + M  (this implies there is no overflow inside montgomery_step_64)

		Since the end result might be bigger than B, we may need to reduce by M, but once is enough.
	*/

	temp.Reduce_ca()
	*z = temp
}

// add_mul_shift_64 computes (target + x * y) >> 64, stores the result in target and return the uint64 shifted out (everything low-endian)
//
// DEPRECATED: helper function for slow variant
func add_mul_shift_64(target *Uint256, x *Uint256, y uint64) (low uint64) {

	// carry_mul_even resp. carry_mul_odd end up in target[even] resp. target[odd]
	// Could do with fewer carries, but that's more error-prone (and also this is more pipeline-friendly, not that it mattered much)

	var carry_mul_even uint64
	var carry_mul_odd uint64
	var carry_add_1 uint64
	var carry_add_2 uint64

	carry_mul_even, low = bits.Mul64(x[0], y)
	low, carry_add_2 = bits.Add64(low, target[0], 0)

	carry_mul_odd, target[0] = bits.Mul64(x[1], y)
	target[0], carry_add_1 = bits.Add64(target[0], carry_mul_even, 0)
	target[0], carry_add_2 = bits.Add64(target[0], target[1], carry_add_2)

	carry_mul_even, target[1] = bits.Mul64(x[2], y)
	target[1], carry_add_1 = bits.Add64(target[1], carry_mul_odd, carry_add_1)
	target[1], carry_add_2 = bits.Add64(target[1], target[2], carry_add_2)

	carry_mul_odd, target[2] = bits.Mul64(x[3], y)
	target[2], carry_add_1 = bits.Add64(target[2], carry_mul_even, carry_add_1)
	target[2], carry_add_2 = bits.Add64(target[2], target[3], carry_add_2)

	target[3] = carry_mul_odd + carry_add_1 + carry_add_2
	return
}

// montgomery_step_64(&t, q) performs t+= (q*BaseFieldSize)/2^64 + 1, assuming no overflow (which needs to be guaranteed by the caller).
//
// DEPRECATED: Helper function for slow variant
func montgomery_step_64(t *Uint256, q uint64) {
	var low, high, carry1, carry2 uint64

	high, _ = bits.Mul64(q, baseFieldSize_0) // throw away least significant uint64 of q*BaseFieldSize.
	t[0], carry1 = bits.Add64(t[0], high, 1) // After this, carry1 needs to go in t[1]

	high, low = bits.Mul64(q, baseFieldSize_1)
	t[0], carry2 = bits.Add64(t[0], low, 0)       // After this, carry2 needs to go in t[1]
	t[1], carry2 = bits.Add64(t[1], high, carry2) // After this, carry2 needs to go in t[2]

	high, low = bits.Mul64(q, baseFieldSize_2)
	t[1], carry1 = bits.Add64(t[1], low, carry1)  // After this, carry1 needs to go in t[2]
	t[2], carry1 = bits.Add64(t[2], high, carry1) // After this, carry1 needs to go in t[3]

	high, low = bits.Mul64(q, baseFieldSize_3)
	t[2], carry2 = bits.Add64(t[2], low, carry2)    // After this, carry2 needs to go in t[3]
	t[3], _ = bits.Add64(t[3], high+carry1, carry2) // _ == 0 guaranteed by caller.
	// If we had an overflow, we would just compute mod 2**256, which is sane behaviour.

	/*
		// Cannot happen (outside benchmark):
		if carry1 != 0 {
			panic("Overflow in montgomery step")
		}
	*/

}
