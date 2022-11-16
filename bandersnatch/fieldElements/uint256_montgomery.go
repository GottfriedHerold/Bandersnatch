package fieldElements

import (
	"math/bits"
)

func (z *uint256) ToNonMontgomery_fc() uint256 {

	// What we need to do here is equivalent to
	// temp.Mul(z, [1,0,0,0])  // where the [1,0,0,0] is the Montgomery representation of the number 1/r.
	// temp.Normalize()
	// return temp.words

	var reducer uint64 = z[0]
	var temp uint256 = [4]uint64{0: z[1], 1: z[2], 2: z[3], 3: 0}

	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}
	reducer = temp.ShiftRight_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	reducer = temp.ShiftRight_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	reducer = temp.ShiftRight_64()
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint64)
	}

	temp.reduce_ca()
	temp.reduce_fb()
	return temp
}

// mul_four_one_64 multiplies a 4x64 bit number by a 1x64 bit number. The result is 5x64 bits, split as 1x64 (low) + 4x64 (high), everything low-endian.
//
// DEPRECATED in favor of version taking a pointer argument to store the result.
func mul_four_one_64(x *uint256, y uint64) (low uint64, high uint256) {
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

// TODO: Move to uint256.go

func mul_uint256_by_uint64(target *[5]uint64, x *uint256, y uint64) {
	var carry, mul_low uint64
	target[1], target[0] = bits.Mul64(x[0], y)

	target[2], mul_low = bits.Mul64(x[1], y)
	target[1], carry = bits.Add64(target[1], mul_low, 0)

	target[3], mul_low = bits.Mul64(x[2], y)
	target[2], carry = bits.Add64(target[2], mul_low, carry)

	target[4], mul_low = bits.Mul64(x[3], y)
	target[3], carry = bits.Add64(target[3], mul_low, carry)

	target[4] += carry
}

// add_mul_shift_64 computes (target + x * y) >> 64, stores the result in target and return the uint64 shifted out (everything low-endian)
func add_mul_shift_64(target *uint256, x *uint256, y uint64) (low uint64) {

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
func montgomery_step_64(t *uint256, q uint64) {
	var low, high, carry1, carry2 uint64

	high, _ = bits.Mul64(q, baseFieldSize_0)
	t[0], carry1 = bits.Add64(t[0], high, 1) // After this, carry1 needs to go in t[1]

	high, low = bits.Mul64(q, baseFieldSize_1)
	t[0], carry2 = bits.Add64(t[0], low, 0)       // After this, carry2 needs to go in t[1]
	t[1], carry2 = bits.Add64(t[1], high, carry2) // After this, carry2 needs to go in t[2]

	high, low = bits.Mul64(q, baseFieldSize_2)
	t[1], carry1 = bits.Add64(t[1], low, carry1)  // After this, carry1 needs to go in t[2]
	t[2], carry1 = bits.Add64(t[2], high, carry1) // After this, carry1 needs to go in t[3]

	high, low = bits.Mul64(q, baseFieldSize_3)
	t[2], carry2 = bits.Add64(t[2], low, carry2) // After this, carry2 needs to go in t[3]
	t[3], carry1 = bits.Add64(t[3], high+carry1, carry2)

	_ = carry1
	/*
		// Cannot happen (outside benchmark):
		if carry1 != 0 {
			panic("Overflow in montgomery step")
		}
	*/

}

func (z *uint256) MulMontgomery_c(x, y *uint256) {

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
	var temp uint256

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

	temp.reduce_ca()
	*z = temp
}

func mon2(t *[5]uint64, x *uint256, y uint64) {
	var low, high, carry1, carry2, carry3, carry4 uint64
	q := t[0]
	if q != 0 {
		q *= negativeInverseModulus_uint64

		high, _ = bits.Mul64(q, baseFieldSize_0)
		t[1], carry1 = bits.Add64(t[1], high, 1) // After this, carry1 needs to go in t[2]

		high, low = bits.Mul64(q, baseFieldSize_1)
		t[1], carry2 = bits.Add64(t[1], low, 0)       // After this, carry2 needs to go in t[2]
		t[2], carry2 = bits.Add64(t[2], high, carry2) // After this, carry2 needs to go in t[3]

		high, low = bits.Mul64(q, baseFieldSize_2)
		t[2], carry1 = bits.Add64(t[2], low, carry1)  // After this, carry1 needs to go in t[3]
		t[3], carry1 = bits.Add64(t[3], high, carry1) // After this, carry1 needs to go in t[4]

		high, low = bits.Mul64(q, baseFieldSize_3)
		t[3], carry2 = bits.Add64(t[3], low, carry2) // After this, carry2 needs to go in t[4]
		t[4], carry2 = bits.Add64(t[4], high+carry1, carry2)
		// testutils.Assert(carry2 == 0)
		_ = carry2
	}
	// pretend that we write t[0] = 0

	carry1, t[0] = bits.Mul64(x[0], y)       // Large carry1 -> t[1]
	t[0], carry2 = bits.Add64(t[0], t[1], 0) // t[0] finished writing, t[1] finished reading, binary carry2 -> t[1]

	carry3, t[1] = bits.Mul64(x[1], y)              // large carry3 -> t[2]
	t[1], carry2 = bits.Add64(t[1], carry1, carry2) // binary carry2 -> t[2]
	t[1], carry1 = bits.Add64(t[1], t[2], 0)        // binary carry1 -> t[2], t[1] finished writing, t[2] finished reading

	carry4, t[2] = bits.Mul64(x[2], y)              // large carry4 -> t[3]
	t[2], carry2 = bits.Add64(t[2], carry3, carry2) // binary carry2 -> t[3]
	t[2], carry1 = bits.Add64(t[2], t[3], carry1)   // binary carry1 -> t[3], t[2] finished writing, t[3] finished reading

	carry3, t[3] = bits.Mul64(x[3], y) // large carry3 -> t[4]
	t[3], carry2 = bits.Add64(t[3], carry4, carry2)
	t[3], carry1 = bits.Add64(t[3], t[4], carry1)

	t[4] = carry3 + carry1 + carry2
}

// unrolled Montgomery multiplication:

func (z *uint256) MulMontgomeryV2_c(x, y *uint256) {
	var temp [5]uint64

	mul_uint256_by_uint64(&temp, x, y[0])
	mon2(&temp, x, y[1])
	mon2(&temp, x, y[2])
	mon2(&temp, x, y[3])
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
		z[3], _ = bits.Add64(temp[4], high+carry1, carry2)
	}
	z.reduce_ca()
}

func (z *uint256) FromMontgomeryRepresentation_fc(x *uint256) {
	// equivalent to
	// z.MulMontgomery_c(x, [1,0,0,0])
	// z.Reduce_f()
	var temp [4]uint64 = *x
	var q, low, carry1, carry2 uint64
	for i := 0; i < 4; i++ {
		// divide by 2**64 mod BaseFieldSize:
		q = temp[0] * negativeInverseModulus_uint64
		if q == 0 {
			temp[0], temp[1], temp[2], temp[3] = temp[1], temp[2], temp[3], 0
		} else {
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
		}
	}
	*z = temp
	// z.reduce_ca()
	z.reduce_fb()
}
