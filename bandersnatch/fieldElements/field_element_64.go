package fieldElements

import (
	"fmt"
	"math/big"
	"math/bits"
	"math/rand"

	"github.com/GottfriedHerold/Bandersnatch/internal/callcounters"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file gives an implementation for field elements (meaning the field of
// definition of the Bandersnatch elliptic curve) using a low-endian Montgomery
// representation without uniqueness of internal representation
// (i.e. a given field element can have multiple representations).

/*
	WARNING :
	The correctness of this implementation subtly relies on
	- the fact that BaseFieldSize/2^256 is between 1/3 and 1/2
	(The >1/3 condition is due to the non-unique representations, where code relies on their exact possible relations.)
	(Be aware that most computations actually *do* result in the smallest possible representation -- so this might not show in test)
	- certain bit-patterns of BaseFieldSize and terms derived from it.

	Adapting this code to other moduli is, hence, extremely error-prone and is recommended against!
*/

// r == 2^256 is the Montgomery multiplier.

// Note: Since Go lacks const arrays,
// we define large 256-bit constants both as untyped 256-bit constants and
// separately as constants for every 64-bit word.
// (Note that the language does not let one do ANYTHING with 256-bit constants other than define other constants)

// baseFieldSizeDoubled_64_i denotes the i'th 64-bit word of 2 * BaseFieldSize
const (
	baseFieldSizeDoubled_64_0 = (2 * BaseFieldSize_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	baseFieldSizeDoubled_64_1
	baseFieldSizeDoubled_64_2
	baseFieldSizeDoubled_64_3
)

// minusOneHalf_64 equals 1/2 * (BaseFieldSize-1) as untyped int
const (
	minusOneHalf_64 = (BaseFieldSize_untyped - 1) / 2
)

// mhalved_64_i denotes the i'th 64-bit word of 1/2 * (BaseFieldSize-1)
const (
	minusOneHalf_64_0 = (minusOneHalf_64 >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	minusOneHalf_64_1
	minusOneHalf_64_2
	minusOneHalf_64_3
)

// rsquared_untyped is 2^512 mod BaseFieldSize. This is useful for converting to/from montgomery form.
const (
	rsquared_untyped = 0x748d9d99f59ff1105d314967254398f2b6cedcb87925c23c999e990f3f29c6d
)

// rsquared_64_i is the i'th 64-bit word of 2^512 mod BaseFieldSize.
const (
	rsquared_64_0 = (rsquared_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	rsquared_64_1
	rsquared_64_2
	rsquared_64_3
)

// 2^256 - 2*BaseFieldSize == 2^256 mod BaseFieldSize. This is also the (unique) Montgomery representation of 1.
// Note: Value is 0x1824b159acc5056f_998c4fefecbc4ff5_5884b7fa00034802_00000001fffffffe
// The weird computation is to avoid 1 << 256, which is not portable according to the go spec (intermediate results are too large even for untyped computations)

// rModBaseField_untyped is 2^256 mod BaseFieldSize. This is also the Montgomery representation of 1.
const rModBaseField_untyped = 2 * ((1 << 255) - BaseFieldSize_untyped)

func init() {
	if rModBaseField_untyped != 0x1824b159acc5056f_998c4fefecbc4ff5_5884b7fa00034802_00000001fffffffe {
		panic(0)
	}
}

// rModBaseField_64_i is the i'th 64-bit word of (2^256 mod BaseFieldSize). Note that this corresponds to the Montgomery representation of 1.
const (
	rModBaseField_64_0 uint64 = (rModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	rModBaseField_64_1
	rModBaseField_64_2
	rModBaseField_64_3
)

// montgomeryNegOne_untyped is the negative of rModBaseField modulo BaseFieldSize.
// This is the Montgomery representation of -1.
const montgomeryNegOne_untyped = BaseFieldSize_untyped - rModBaseField_untyped

// montgomeryNegOne_i is the i'th 64-bit word of the negative of rModBaseField modulo BaseFieldSize.
// This is the Montgomery representation of -1.
const (
	montgomeryNegOne_0 uint64 = (montgomeryNegOne_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	montgomeryNegOne_1
	montgomeryNegOne_2
	montgomeryNegOne_3
)

type bsFieldElement_64 struct {
	// field elements stored in low-endian 64-bit uints in Montgomery form, i.e.
	// a bsFieldElement_64 encodes a field element x if
	// words - x * 2^256 == 0 (mod BaseFieldSize), where words is interpreted in LowEndiant as a 256-bit number.

	// Note that the representation of x is actually NOT unique.
	// The invariant that we maintain to get efficient field operations is that
	//
	// ********************************************
	// *                                          *
	// *   0 <= words < (2^256) - BaseFieldSize  *
	// *                                          *
	// ********************************************
	//
	// i.e. adding BaseFieldSize does not overflow.
	// Of course, this invariant concerns the Montgomery representation, interpreting words directly as a 256-bit integer.
	// Since BaseFieldSize is between 1/3*2^256 and 1/2*2^256, a given field element x might have either 1 or 2 possible representations as
	// a bsFieldElement_64, both of which are equally valid as far as this implementation is concerned.
	words [4]uint64
}

// Note: We export *copies* of these variables. Internal functions should use the original.
// This way the compiler has a chance to determine that these value never change and optimize for it.

// representation of zero. This is supposedly a constant.
var bsFieldElement_64_zero bsFieldElement_64

// alternative representation of zero. Note that we must never call Normalize() on it, which e.g. IsEqual may do.
var bsFieldElement_64_zero_alt bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}}

// The field element 1.
var bsFieldElement_64_one bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{rModBaseField_64_0, rModBaseField_64_1, rModBaseField_64_2, rModBaseField_64_3}}

// The field element -1
var bsFieldElement_64_minusone bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{montgomeryNegOne_0, montgomeryNegOne_1, montgomeryNegOne_2, montgomeryNegOne_3}}

// The number 2^256 in Montgomery form.
var bsFieldElement_64_r bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{0: rsquared_64_0, 1: rsquared_64_1, 2: rsquared_64_2, 3: rsquared_64_3}}

// Benchmarking only:

var _ = callcounters.CreateHierarchicalCallCounter("FieldOps", "Field Operations", "")
var _ = callcounters.CreateHierarchicalCallCounter("AddSubFe", "Additions and Subtractions", "FieldOps")
var _ = callcounters.CreateHierarchicalCallCounter("Multiplications", "", "FieldOps")
var _ = callcounters.CreateHierarchicalCallCounter("Divisions", "", "FieldOps")
var _ = callcounters.CreateHierarchicalCallCounter("OtherFe", "Others", "FieldOps")
var _ = callcounters.CreateHierarchicalCallCounter("AddFe", "Additions", "AddSubFe")
var _ = callcounters.CreateHierarchicalCallCounter("SubFe", "Subtractions", "AddSubFe")
var _ = callcounters.CreateHierarchicalCallCounter("Jacobi", "Jacobi symbols", "OtherFe")
var _ = callcounters.CreateHierarchicalCallCounter("NegFe", "Negations", "OtherFe")
var _ = callcounters.CreateHierarchicalCallCounter("MulFe", "generic Multiplications", "Multiplications")
var _ = callcounters.CreateHierarchicalCallCounter("MulByFive", "Multiplications by 5", "Multiplications")
var _ = callcounters.CreateHierarchicalCallCounter("Squarings", "", "Multiplications")
var _ = callcounters.CreateHierarchicalCallCounter("SqrtFe", "Square roots", "OtherFe")
var _ = callcounters.CreateHierarchicalCallCounter("InvFe", "Inversions", "Divisions")
var _ = callcounters.CreateHierarchicalCallCounter("DivideFe", "generic Divisions", "Divisions")

// maybe_reduce_once changes the representation of z to restore the invariant that z.words + BaseFieldSize must not overflow.
// This is done by subtracting BaseFieldSize if the number is too large.
func (z *bsFieldElement_64) maybe_reduce_once() {
	var borrow uint64
	// Note: if z.words[3] == m_64_3, we may or may not be able to reduce, depending on the other words.
	// At any rate, we do not really need to, so we don't check.
	if z.words[3] > baseFieldSize_3 {
		z.words[0], borrow = bits.Sub64(z.words[0], baseFieldSize_0, 0)
		z.words[1], borrow = bits.Sub64(z.words[1], baseFieldSize_1, borrow)
		z.words[2], borrow = bits.Sub64(z.words[2], baseFieldSize_2, borrow)
		z.words[3], _ = bits.Sub64(z.words[3], baseFieldSize_3, borrow) // _ is guaranteed to be 0
	}
}

// isNormalized checks whether the internal representaion is in 0<= . < BaseFieldSize.
// This function is only used internally. Users should just call Normalize if in doubt.
func (z *bsFieldElement_64) isNormalized() bool {
	// Workaround for Go's lack of const-arrays. Hoping for smart-ish compiler.
	// Note that the RHS is const and the left-hand-side is local and never written to after initialization.
	var baseFieldSize_copy [4]uint64 = [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}
	for i := int(3); i >= 0; i-- {
		if z.words[i] < baseFieldSize_copy[i] {
			return true
		} else if z.words[i] > baseFieldSize_copy[i] {
			return false
		}
	}
	// if we get here, z.words == BaseFieldSize
	return false
}

// Normalize() changes the internal representation of z to a unique number in 0 <= . < BaseFieldSize
// After a call to z.Normalize(), we are guaranteed that read-operations on z will no longer potentially
// change the internal representation until we write to z again.
//
// After a call to Normalize on both operands, the default == operator does the right thing.
// This is mostly an internal function, but it might be needed for compatibility with other libraries that scan the internal byte representation (for hashing, say)
// or when using bsFieldElement_64 as keys for a map or when sharing a field element between multiple goroutines.
func (z *bsFieldElement_64) Normalize() {
	if z.isNormalized() {
		return
	}
	var borrow uint64
	z.words[0], borrow = bits.Sub64(z.words[0], baseFieldSize_0, 0)
	z.words[1], borrow = bits.Sub64(z.words[1], baseFieldSize_1, borrow)
	z.words[2], borrow = bits.Sub64(z.words[2], baseFieldSize_2, borrow)
	z.words[3], borrow = bits.Sub64(z.words[3], baseFieldSize_3, borrow)
	if borrow != 0 {
		panic(ErrorPrefix + "Underflow in normalization. This was supposed to be impossible to happen.")
	}
}

// Sign outputs the "sign" of the field element.
// More precisely, consider the integer representation z of minimal absolute value (i.e between -BaseField/2 < . < BaseField/2) and take its sign.
// The return value is in {-1,0,+1}.
// This is not compatible with addition or multiplication. It has the property that Sign(z) == -Sign(-z), which is the main thing we need.
// We also might use the fact that positive-sign field elements start with 00 in their high-endian serializiation.
func (z *bsFieldElement_64) Sign() int {
	if z.IsZero() {
		return 0
	}
	// we take the sign of the non-Montgomery form.
	// Of course, the property that Sign(z) == -Sign(-z) would hold either way (and not switching would actually be more efficient).
	// However, Sign() enters into (De)Serialization routines for curve points. This choice is probably more portable.
	var low_endian_words [4]uint64 = z.undoMontgomery()

	// Go's lack of const-arrays is annoying.
	var mhalf_copy [4]uint64 = [4]uint64{minusOneHalf_64_0, minusOneHalf_64_1, minusOneHalf_64_2, minusOneHalf_64_3}

	for i := int(3); i >= 0; i-- {
		if low_endian_words[i] > mhalf_copy[i] {
			return -1
		} else if low_endian_words[i] < mhalf_copy[i] {
			return 1
		}
	}
	// If we get here, z is equal to mhalf, which we defined as (BaseFieldSize-1)/2. Due to rounding, this corresponds to +1
	return 1
}

// TODO: Make MUCH more efficient. The standard library's implementation's performance appears to be quite bad.
// (For a start, it allocates like crazy, for which there is absolutely no reason)
// Furthermore, the standard library does the Euclid-like algorithm with computing
// denominator modulo numerator, but chooses the representative in 0 <= . < denominator
// (Rather than minimal absolute value, with is better)

// Jacobi computes the Legendre symbol of the received elements z.
// This means that z.Jacobi() is +1 if z is a non-zero square and -1 if z is a non-square. z.Jacobi() == 0 iff z.IsZero()
func (z *bsFieldElement_64) Jacobi() int {
	IncrementCallCounter("Jacobi")
	tempInt := z.ToBigInt()
	return big.Jacobi(tempInt, BaseFieldSize_Int)
}

// Add is used to perform addition.
//
// Use z.Add(&x, &y) to add x + y and store the result in z.
func (z *bsFieldElement_64) Add(x, y *bsFieldElement_64) {
	IncrementCallCounter("AddFe")

	var carry uint64
	z.words[0], carry = bits.Add64(x.words[0], y.words[0], 0)
	z.words[1], carry = bits.Add64(x.words[1], y.words[1], carry)
	z.words[2], carry = bits.Add64(x.words[2], y.words[2], carry)
	z.words[3], carry = bits.Add64(x.words[3], y.words[3], carry)
	// carry == 1 basically only happens here if you do it on purpose (add up *lots* of non-normalized numbers).
	// NOTE: If carry == 1, then z.maybe_reduce_once() actually commutes with the -=mdoubled here: it won't do anything either before or after it.

	if carry != 0 {
		z.words[0], carry = bits.Sub64(z.words[0], baseFieldSizeDoubled_64_0, 0)
		z.words[1], carry = bits.Sub64(z.words[1], baseFieldSizeDoubled_64_1, carry)
		z.words[2], carry = bits.Sub64(z.words[2], baseFieldSizeDoubled_64_2, carry)
		z.words[3], _ = bits.Sub64(z.words[3], baseFieldSizeDoubled_64_3, carry)
	}

	// else?
	z.maybe_reduce_once()
}

func (z *bsFieldElement_64) Add2(x, y *bsFieldElement_64) {
	xUint256 := (*uint256)(&x.words)
	yUint256 := (*uint256)(&y.words)
	zUint256 := (*uint256)(&z.words)
	zUint256.Add_ReduceNonUnique(xUint256, yUint256)
}

// Sub is used to perform subtraction.
//
// Use z.Sub(&x, &y) to compute x - y and store the result in z.
func (z *bsFieldElement_64) Sub(x, y *bsFieldElement_64) {
	IncrementCallCounter("SubFe")
	var borrow uint64 // only takes values 0,1
	z.words[0], borrow = bits.Sub64(x.words[0], y.words[0], 0)
	z.words[1], borrow = bits.Sub64(x.words[1], y.words[1], borrow)
	z.words[2], borrow = bits.Sub64(x.words[2], y.words[2], borrow)
	z.words[3], borrow = bits.Sub64(x.words[3], y.words[3], borrow)
	if borrow != 0 {
		// mentally rename borrow -> carry
		if z.words[3] > 0xFFFFFFFF_FFFFFFFF-baseFieldSize_3 {
			z.words[0], borrow = bits.Add64(z.words[0], baseFieldSize_0, 0)
			z.words[1], borrow = bits.Add64(z.words[1], baseFieldSize_1, borrow)
			z.words[2], borrow = bits.Add64(z.words[2], baseFieldSize_2, borrow)
			z.words[3], _ = bits.Add64(z.words[3], baseFieldSize_3, borrow) // _ is one
		} else {
			z.words[0], borrow = bits.Add64(z.words[0], baseFieldSizeDoubled_64_0, 0)
			z.words[1], borrow = bits.Add64(z.words[1], baseFieldSizeDoubled_64_1, borrow)
			z.words[2], borrow = bits.Add64(z.words[2], baseFieldSizeDoubled_64_2, borrow)
			z.words[3], _ = bits.Add64(z.words[3], baseFieldSizeDoubled_64_3, borrow) // _ is one
			// Note: z might be > BaseFieldSize, but not by much. This is fine.
		}
	}
}

var _ = callcounters.CreateAttachedCallCounter("SubFromNeg", "Subtractions called by Neg", "SubFe").
	AddToThisFromSource("NegFe", 1).
	AddThisToTarget("FieldOps", -1)

// Neg computes the additive inverse (i.e. -x)
//
// Use z.Neg(&x) to set z = -x
func (z *bsFieldElement_64) Neg(x *bsFieldElement_64) {
	IncrementCallCounter("NegFe")
	// IncrementCallCounter("SubFromNeg") -- done automatically
	z.Sub(&bsFieldElement_64_zero_alt, x) // using alt here makes the if borrow!=0 in Sub unlikely.
}

// mul_four_one_64 multiplies a 4x64 bit number by a 1x64 bit number. The result is 5x64 bits, split as 1x64 (low) + 4x64 (high), everything low-endian.
func mul_four_one_64(x *[4]uint64, y uint64) (low uint64, high [4]uint64) {
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

// add_mul_shift_64 computes (target + x * y) >> 64, stores the result in target and return the uint64 shifted out (everything low-endian)
func add_mul_shift_64(target *[4]uint64, x *[4]uint64, y uint64) (low uint64) {

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
func montgomery_step_64(t *[4]uint64, q uint64) {
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

	// Cannot happen:
	if carry1 != 0 {
		panic("Overflow in montgomery step")
	}

}

// Mul computes multiplication in the field.
//
// Use z.Mul(&x, &y) to set z = x * y
func (z *bsFieldElement_64) Mul(x, y *bsFieldElement_64) {
	IncrementCallCounter("MulFe")
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
	var temp [4]uint64

	// -1/Modulus mod r.
	const negativeInverseModulus = (0xFFFFFFFF_FFFFFFFF * 0x00000001_00000001) % (1 << 64)
	const negativeInverseModulus_uint uint64 = negativeInverseModulus

	var reducer uint64

	reducer, temp = mul_four_one_64(&x.words, y.words[0]) // NOTE: temp <= B - floor(B/r) - 1  <= B + floor(M/r), see overflow analysis below

	// If reducer == 0, then temp == x*y[0]/r.
	// Otherwise, we need to compute temp = ([temp, reducer] + BaseFieldSize * (reducer * negativeInverseModulus mod r)) / r
	// Note that we know exactly what happens in the least significant uint64 in the addition (result is 0, carry is 1).
	// Be aware that carry 1 relies on reducer != 0, hence the if reducer!=0 condition
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, &x.words, y.words[1])
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, &x.words, y.words[2])
	if reducer != 0 {
		montgomery_step_64(&temp, reducer*negativeInverseModulus_uint)
	}

	reducer = add_mul_shift_64(&temp, &x.words, y.words[3])
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

	z.words = temp
	z.maybe_reduce_once()
}

// IsZero checks whether the field element is zero
func (z *bsFieldElement_64) IsZero() bool {
	return (z.words[0]|z.words[1]|z.words[2]|z.words[3] == 0) || (*z == bsFieldElement_64_zero_alt)
}

// IsOne checks whether the field element is 1
func (z *bsFieldElement_64) IsOne() bool {
	// Note: The representation of 1 is unique:
	// bsFieldElement_64_one.words corresponds to 2^256 - 2*BaseField, so the other (potential) Montgomery representation
	// would be 2^256-1*BaseFieldSize, which (barely) violates our invariant that addition of BaseFieldSize does not overflow.
	return *z == bsFieldElement_64_one
}

// SetOne sets the field element to 1.
func (z *bsFieldElement_64) SetOne() {
	z.words = bsFieldElement_64_one.words
}

// SetZero sets the field element to 0.
func (z *bsFieldElement_64) SetZero() {
	z.words = bsFieldElement_64_zero.words
}

// shift_once shifts the internal uint64 array once (equivalent to division by 2^64) and returns the shifted-out uint64
func (z *bsFieldElement_64) shift_once() (result uint64) {
	result = z.words[0]
	z.words[0] = z.words[1]
	z.words[1] = z.words[2]
	z.words[2] = z.words[3]
	z.words[3] = 0
	return
}

// undoMontgomery gives a low-endian representation of the underlying number, undoing the Montgomery form.
func (z *bsFieldElement_64) undoMontgomery() [4]uint64 {

	// What we need to do here is equivalent to
	// temp.Mul(z, [1,0,0,0])  // where the [1,0,0,0] is the Montgomery representation of the number 1/r.
	// temp.Normalize()
	// return temp.words

	// -1/Modulus mod r.
	const negativeInverseModulus = (0xFFFFFFFF_FFFFFFFF * 0x00000001_00000001) % (1 << 64)
	const negativeInverseModulus_uint uint64 = negativeInverseModulus

	var reducer uint64 = z.words[0]
	var temp bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{0: z.words[1], 1: z.words[2], 2: z.words[3], 3: 0}}

	if reducer != 0 {
		montgomery_step_64(&temp.words, reducer*negativeInverseModulus_uint)
	}
	reducer = temp.shift_once()
	if reducer != 0 {
		montgomery_step_64(&temp.words, reducer*negativeInverseModulus_uint)
	}

	reducer = temp.shift_once()
	if reducer != 0 {
		montgomery_step_64(&temp.words, reducer*negativeInverseModulus_uint)
	}

	reducer = temp.shift_once()
	if reducer != 0 {
		montgomery_step_64(&temp.words, reducer*negativeInverseModulus_uint)
	}

	temp.maybe_reduce_once()
	temp.Normalize()
	return temp.words
}

var _ = callcounters.CreateAttachedCallCounter("MulEqFromMontgomery", "", "MulEqFe")

// restoreMontgomery restores the internal Montgomery representation, assuming the current internal representation is *NOT* in Montgomery form.
// This must only be used internally.
func (z *bsFieldElement_64) restoreMontgomery() {
	IncrementCallCounter("MulEqFromMontgomery")
	z.MulEq(&bsFieldElement_64_r)
}

// ToBigInt returns a *big.Int that stores a representation of (a copy of) the given field element.
func (z *bsFieldElement_64) ToBigInt() *big.Int {
	temp := z.undoMontgomery()
	return utils.UIntarrayToInt(&temp)
}

// SetBigInt converts from *big.Int to a field element. The input need not be reduced modulo the field size.
func (z *bsFieldElement_64) SetBigInt(v *big.Int) {
	sign := v.Sign()
	w := new(big.Int).Set(v)
	w.Abs(w)

	// Can be done much more efficiently if desired, but we do not convert often.
	w.Lsh(w, 256) // To account for Montgomery form
	w.Mod(w, BaseFieldSize_Int)
	if sign < 0 {
		w.Sub(BaseFieldSize_Int, w)
	}
	z.words = utils.BigIntToUIntArray(w)
	// Note z is Normalized.
}

// ToUInt64 returns z with err==nil if z can be represented by a uint64.
//
// If z cannot be represented by a uint64, returns <something, should not be used>, ErrCannotRepresentAsUInt64
func (z *bsFieldElement_64) ToUInt64() (result uint64, err error) {
	temp := z.undoMontgomery()
	result = temp[0]
	if (temp[1] | temp[2] | temp[3]) != 0 {
		err = ErrCannotRepresentAsUInt64
	}
	return
}

// SetUInt64 sets z to the given value.
func (z *bsFieldElement_64) SetUInt64(value uint64) {
	// Sets z.words to the correct value (not in Montgomery form)
	z.words[0] = value
	z.words[1] = 0
	z.words[2] = 0
	z.words[3] = 0
	// put in Montgomery form
	z.restoreMontgomery()
}

// temporarily exported. Needs some restructing to unexport.

// SetRandomUnsafe generates a uniformly random field element.
// Note that this is not crypto-grade randomness. This is used in unit-testing only.
// We do NOT guarantee that the distribution is even close to uniform.
func (z *bsFieldElement_64) SetRandomUnsafe(rnd *rand.Rand) {
	// Not the most efficient way (transformation to Montgomery form is obviously not needed), but for testing purposes we want the _64 and _8 variants to have the same output for given random seed.
	var xInt *big.Int = new(big.Int).Rand(rnd, BaseFieldSize_Int)
	z.SetBigInt(xInt)
}

// SetRandomUnsafeNonZero generates uniformly random non-zero field elements.
// Note that this is not crypto-grade randomness. This is used in unit-testing only.
// We do NOT guarantee that the distribution is even close to uniform.
func (z *bsFieldElement_64) SetRandomUnsafeNonZero(rnd *rand.Rand) {
	for {
		var xInt *big.Int = new(big.Int).Rand(rnd, BaseFieldSize_Int)
		if xInt.Sign() != 0 {
			z.SetBigInt(xInt)
			return
		}
		// We only get here with negligible probability, but we prefer to be precise if we can.
		// (in particular, because rnd could be crafted)
	}
}

// Multiply_by_five computes z *= 5.
// This is useful, because the coefficient of a in the twisted Edwards representation of Bandersnatch is a=-5
func (z *bsFieldElement_64) Multiply_by_five() {
	IncrementCallCounter("MulByFive")

	// We multiply by five by multiplying each individual word by 5 and then correcting the overflows later.

	var overflow1, overflow2, overflow3, overflow4 uint64 // overflow_i *contributes to* the i-th uint64 (i.e. comes from the i-1'th)
	var carry uint64
	// could do this with bit fiddling as well, but that's more complicated and probably slower (depends on compiler)
	overflow1, z.words[0] = bits.Mul64(z.words[0], 5)
	overflow2, z.words[1] = bits.Mul64(z.words[1], 5)
	overflow3, z.words[2] = bits.Mul64(z.words[2], 5)
	overflow4, z.words[3] = bits.Mul64(z.words[3], 5)

	// Note that due to the size restrictions on z and the particular value of BaseFieldSize, 0 <= overflow4 <= 2
	// overflow4 contributes overflow4 * 2^256 == overflow4 * rModBaseField (mod BaseField) to the total result.
	// splitting this into words gives the following contributions

	// contributions due to overflows:
	overflow1 += overflow4 * rModBaseField_64_1 // This computations itself cannot overflow, because 2*rModBaseField_1 + 2 is not large enough
	overflow2 += overflow4 * rModBaseField_64_2 // this overflows itself iff overflow4 == 2
	overflow3 += overflow4*rModBaseField_64_3 + (overflow4 / 2)

	// Read this as overflow0 := overflow4 * rModBaseField_64_0
	// and mentally rename overflow4 -> overflow0 from here on
	overflow4 *= rModBaseField_64_0

	z.words[0], carry = bits.Add64(z.words[0], overflow4, 0)
	z.words[1], carry = bits.Add64(z.words[1], overflow1, carry)
	z.words[2], carry = bits.Add64(z.words[2], overflow2, carry)
	z.words[3], carry = bits.Add64(z.words[3], overflow3, carry)

	// if carry == 1, we need to add 1<<256 mod BaseField == rModBaseField.
	// We do this via bit-masking rather than with an if

	overflow4 = -carry // == carry * 0xFFFFFFFF_FFFFFFFF

	z.words[0], carry = bits.Add64(z.words[0], overflow4&rModBaseField_64_0, 0)
	z.words[1], carry = bits.Add64(z.words[1], overflow4&rModBaseField_64_1, carry)
	z.words[2], carry = bits.Add64(z.words[2], overflow4&rModBaseField_64_2, carry)
	z.words[3], _ = bits.Add64(z.words[3], overflow4&rModBaseField_64_3, carry) // _ == 0 is guaranteed

	z.maybe_reduce_once()
}

// Inv computes the multiplicative Inverse:
//
// z.Inv(x) performs z:= 1/x. If x is 0, the behaviour is undefined (possibly panic)
func (z *bsFieldElement_64) Inv(x *bsFieldElement_64) {
	IncrementCallCounter("InvFe")
	// Slow, but rarely used anyway (due to working in projective coordinates)
	t := x.ToBigInt()
	if t.ModInverse(t, BaseFieldSize_Int) == nil {
		panic(ErrDivisionByZero)
	}
	z.SetBigInt(t)
}

var _ = callcounters.CreateAttachedCallCounter("InvFromDivide", "Inversion in Divide", "InvFe").
	AddToThisFromSource("DivideFe", +1).
	AddThisToTarget("Divisions", -1)

var _ = callcounters.CreateAttachedCallCounter("MulFromDivide", "Multiplications called by Divide", "MulFe").
	AddToThisFromSource("DivideFe", +1).
	AddThisToTarget("FieldOps", -1)

// TODO: Specify behaviour for denom == 0?
// Note that the reason for the ambiguity is the behaviour of big.Int (and consequently the _8 comparison implementation)

// Divide performs division: z.Divide(num, denom) means z = num/denom
//
// Division by zero causes undefined behaviour (possibly panic, possibly returns 0)
func (z *bsFieldElement_64) Divide(num *bsFieldElement_64, denom *bsFieldElement_64) {
	IncrementCallCounter("DivideFe")

	var temp bsFieldElement_64 // needed, because some of z, num, denom might alias
	temp.Inv(denom)
	z.Mul(num, &temp)
}

// IsEqual compares two field elements for equality, i.e. it checks whether z == x (mod BaseFieldSize)
func (z *bsFieldElement_64) IsEqual(x *bsFieldElement_64) bool {
	// There are at most 2 possible representations per field element and they differ by exactly BaseFieldSize.
	// So it is enough to reduce the larger one, provided it is much larger.

	switch {
	case z.words[3] == x.words[3]:
		return *z == *x
	case z.words[3] > x.words[3]:
		// Note that RHS cannot overflow due to invariant
		if z.words[3] > x.words[3]+(baseFieldSize_3-1) {
			z.Normalize()
			return *z == *x
		} else {
			return false
		}
	case z.words[3] < x.words[3]:
		// Note that RHS cannot overflow due to invariant
		if x.words[3] > z.words[3]+(baseFieldSize_3-1) {
			x.Normalize()
			return *z == *x
		} else {
			return false
		}
	// needed to make golang not complain about missing return in all branches. The cases above are obviously exhaustive.
	default:
		panic(ErrorPrefix + "This cannot happen")
	}
}

// TODO: error or bool? Specify what happens with z on error?

// SquareRoot computes a SquareRoot in the field.
//
// Use ok := z.SquareRoot(&x).
//  The return value tells whether the operation was successful.
// If x is not a square, the return value is false and z is untouched.
func (z *bsFieldElement_64) SquareRoot(x *bsFieldElement_64) (ok bool) {
	IncrementCallCounter("SqrtFe")
	xInt := x.ToBigInt()
	if xInt.ModSqrt(xInt, BaseFieldSize_Int) == nil {
		return false
	}
	z.SetBigInt(xInt)
	return true
}

// Format is provided to satisfy the fmt.Formatter interface. Note that this is defined on value receivers.
// We internally convert to big.Int and hence support the same formats as big.Int.
func (z bsFieldElement_64) Format(s fmt.State, ch rune) {
	z.ToBigInt().Format(s, ch)
}

// String is provided to satisfy the fmt.Stringer interface. Note that this is defined on a *value* receiver.
func (z bsFieldElement_64) String() string {
	z.Normalize()
	return z.ToBigInt().String()
}

var _ = callcounters.CreateAttachedCallCounter("AddEqFe", "", "AddFe")

// var _ = callcounters.CreateHierarchicalCallCounter("AddEqFe", "", "AddSubFe")

// AddEq implements += for field elements.
//
// z.AddEq(&x) is equvalent to z.Add(&z, &x)
func (z *bsFieldElement_64) AddEq(y *bsFieldElement_64) {
	IncrementCallCounter("AddEqFe")

	// z.Add(z,x) is strangely slow (x2.5 compared to z.Add(x,y) for z!=x,y)
	// I have no idea why, probably the writes to z stall the reads from z (even though they shouldn't).
	z.Add(z, y)

	// This should work as well, but adds complexity and error-proneness while
	// being only slightly faster, so we use the simple approach for now.
	// Proably, we would need to write it in assembly anyway.
	/*
		var carry uint64
		var too_large bool
		var overflow uint64
		var temp0, temp1, temp2, temp3 uint64
		temp3, overflow = bits.Add64(z.words[3], y.words[3], 0)
		too_large = temp3 > m_64_3
		way_too_large := temp3 > mdoubled_64_3
		temp0, carry = bits.Add64(z.words[0], y.words[0], 0)
		temp1, carry = bits.Add64(z.words[1], y.words[1], carry)
		temp2, carry = bits.Add64(z.words[2], y.words[2], carry)

		temp3 += carry // this might overflow, but that's fine, because then way_too_large is true

		// overflow == true basically only happens here if you do it on purpose (add up *lots* of non-normalized numbers).
		// Also, overflow and too_large are exclusive due to the size constraints on the input (x+BaseField, z+Basefield do not overflow)

		var borrow uint64
		// Note: if z.words[3] == m_64_3, we may or may not be able to reduce, depending on the other words. At any rate, we do not really need to.
		if too_large {
			z.words[0], borrow = bits.Sub64(temp0, m_64_0, 0)
			z.words[1], borrow = bits.Sub64(temp1, m_64_1, borrow)
			z.words[2], borrow = bits.Sub64(temp2, m_64_2, borrow)
			z.words[3], _ = bits.Sub64(temp3+carry, m_64_3, borrow) // _ is guaranteed to be 0
		} else if way_too_large || overflow != 0 {
			z.words[0], borrow = bits.Sub64(temp0, mdoubled_64_0, 0)
			z.words[1], borrow = bits.Sub64(temp1, mdoubled_64_1, borrow)
			z.words[2], borrow = bits.Sub64(temp2, mdoubled_64_2, borrow)
			z.words[3], _ = bits.Sub64(temp3+carry, mdoubled_64_3, borrow)
		} else {
			z.words[0] = temp0
			z.words[1] = temp1
			z.words[2] = temp2
			z.words[3] = temp3 + carry
		}
	*/
}

var _ = callcounters.CreateAttachedCallCounter("SubEqFe", "", "SubFe")

// SubEq implements the -= operation.
//
// z.SubEq(&x) is equivalent to z.Add(&z, &x)
func (z *bsFieldElement_64) SubEq(x *bsFieldElement_64) {
	IncrementCallCounter("SubEqFe")
	z.Sub(z, x)
}

var _ = callcounters.CreateAttachedCallCounter("MulEqFe", "", "MulFe")

// MulEq implements the *= operation.
//
// z.MulEq(&x) is equivalent to z.Mul(&z, &x)
func (z *bsFieldElement_64) MulEq(x *bsFieldElement_64) {
	IncrementCallCounter("MulEqFe")
	z.Mul(z, x)
}

var _ = callcounters.CreateAttachedCallCounter("MulFromSquare", "as part of non-optimized Squaring", "MulFe").
	AddToThisFromSource("Squarings", +1).
	AddThisToTarget("Multiplications", -1)

// Square squares the field element, computing z = x * x
//
// z.Square(&x) is equivalent to z.Mul(&x, &x)
func (z *bsFieldElement_64) Square(x *bsFieldElement_64) {
	IncrementCallCounter("Squarings")
	z.Mul(x, x)
}

// SquareEq replaces the field element by its squared value.
//
// z.SquareEq() is equivalent to z.Square(&z)
func (z *bsFieldElement_64) SquareEq() {
	IncrementCallCounter("Squarings")
	z.Mul(z, z) // or z.Square(z), but it's the same for now (except for the need to adjust call counters)
}

// Double computes the double of a point, i.e. z = 2*x == x + x
//
// z.Double(&x) is equivalent to z.Add(&x, &x)
func (z *bsFieldElement_64) Double(x *bsFieldElement_64) {
	z.Add(x, x)
}

// DoubleEq replaces the provided field element by its doubled value, i.e. computes z *= 2
//
// z.DoubleEq() is equivalent to z.Double(&z)
func (z *bsFieldElement_64) DoubleEq() {
	z.Add(z, z)
}

var _ = callcounters.CreateAttachedCallCounter("NegEqFe", "", "NegFe")

// NegEq replaces the provided field element by its negative.
//
// z.NegEq() is equivalent to z.Neg(&z)
func (z *bsFieldElement_64) NegEq() {
	IncrementCallCounter("NegEqFe")
	z.Neg(z)
}

var _ = callcounters.CreateAttachedCallCounter("InvEqFe", "", "InvFe")

// TODO: Consider specifying what happens at 0.

// InvEq replaces the provided field element by its multiplicative inverse (in the field, i.e. modulo BaseFieldSize).
// The behaviour is unspecified (potentially panic) if z is zero.
//
// z.InvEq is equivalent to z.Inv(&z)
func (z *bsFieldElement_64) InvEq() {
	z.Inv(z)
}

var _ = callcounters.CreateAttachedCallCounter("DivideEqFe", "", "DivideFe")

// DivideEq performs a z /= x operation
// The behaviour is undefined (potentially panic) if x is zero.
//
// z.DivideEq(&x) is equivalent to z.Divide(&z, &x) for non-zero x
func (z *bsFieldElement_64) DivideEq(denom *bsFieldElement_64) {
	z.Divide(z, denom)
}

// This is essentially a helper function that we need in several places.

// CmpAbs compares the absolute values of two field elements and whether the signs match:
//
// absEqual is true iff x == +/- z
// exactly equal is true iff x == z
func (z *bsFieldElement_64) CmpAbs(x *bsFieldElement_64) (absEqual bool, exactlyEqual bool) {
	if z.IsEqual(x) {
		return true, true
	}
	var tmp bsFieldElement_64
	tmp.Neg(x)
	if tmp.IsEqual(z) {
		return true, false
	}
	return false, false
}
