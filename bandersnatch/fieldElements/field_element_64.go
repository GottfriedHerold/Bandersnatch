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
	words uint256
}

// Note: We export *copies* of these variables. Internal functions should use the original.
// This way the compiler has a chance to determine that these value never change and optimize for it.

// representation of zero. This is supposedly a constant.
var bsFieldElement_64_zero bsFieldElement_64

// alternative representation of zero. Note that we must never call Normalize() on it, which e.g. IsEqual may do.
var bsFieldElement_64_zero_alt bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}}

// The field element 1.
var bsFieldElement_64_one bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{twoTo256ModBaseField_64_0, twoTo256ModBaseField_64_1, twoTo256ModBaseField_64_2, twoTo256ModBaseField_64_3}}

// The field element -1
var bsFieldElement_64_minusone bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{minus2To256ModBaseField_64_0, minus2To256ModBaseField_64_1, minus2To256ModBaseField_64_2, minus2To256ModBaseField_64_3}}

// The number 2^256 in Montgomery form.
var bsFieldElement_64_r bsFieldElement_64 = bsFieldElement_64{words: [4]uint64{0: twoTo512ModBaseField_64_0, 1: twoTo512ModBaseField_64_1, 2: twoTo512ModBaseField_64_2, 3: twoTo512ModBaseField_64_3}}

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

// isNormalized checks whether the internal representaion is in 0<= . < BaseFieldSize.
// This function is only used internally. Users should just call Normalize if in doubt.
func (z *bsFieldElement_64) isNormalized() bool {
	return z.words.is_fully_reduced()
}

// Normalize() changes the internal representation of z to a unique number in 0 <= . < BaseFieldSize
// After a call to z.Normalize(), we are guaranteed that read-operations on z will no longer potentially
// change the internal representation until we write to z again.
//
// After a call to Normalize on both operands, the default == operator does the right thing.
// This is mostly an internal function, but it might be needed for compatibility with other libraries that scan the internal byte representation (for hashing, say)
// or when using bsFieldElement_64 as keys for a map or when sharing a field element between multiple goroutines.
func (z *bsFieldElement_64) Normalize() {
	z.words.reduce_fb()
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
	var nonMontgomery uint256 = z.words.ToNonMontgomery_fc()

	var mhalf_copy uint256 = [4]uint64{minusOneHalfModBaseField_64_0, minusOneHalfModBaseField_64_1, minusOneHalfModBaseField_64_2, minusOneHalfModBaseField_64_3}

	for i := int(3); i >= 0; i-- {
		if nonMontgomery[i] > mhalf_copy[i] {
			return -1
		} else if nonMontgomery[i] < mhalf_copy[i] {
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
// Or use a binary-gcd-like variant.

// Jacobi computes the Legendre symbol of the received elements z.
// This means that z.Jacobi() is +1 if z is a non-zero square and -1 if z is a non-square. z.Jacobi() == 0 iff z.IsZero()
func (z *bsFieldElement_64) Jacobi() int {
	IncrementCallCounter("Jacobi")
	tempInt := z.ToBigInt()
	return big.Jacobi(tempInt, baseFieldSize_Int)
}

// Add is used to perform addition.
//
// Use z.Add(&x, &y) to add x + y and store the result in z.
func (z *bsFieldElement_64) Add(x, y *bsFieldElement_64) {
	IncrementCallCounter("AddFe")

	z.words.AddAndReduce_b_c(&x.words, &y.words)
}

// Sub is used to perform subtraction.
//
// Use z.Sub(&x, &y) to compute x - y and store the result in z.
func (z *bsFieldElement_64) Sub(x, y *bsFieldElement_64) {
	IncrementCallCounter("SubFe")
	z.words.SubAndReduce_c(&x.words, &y.words)
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

// Mul computes multiplication in the field.
//
// Use z.Mul(&x, &y) to set z = x * y
func (z *bsFieldElement_64) Mul(x, y *bsFieldElement_64) {
	IncrementCallCounter("MulFe")

	z.words.MulMontgomery_c(&x.words, &y.words)
}

// IsZero checks whether the field element is zero
func (z *bsFieldElement_64) IsZero() bool {
	// return (z.words[0]|z.words[1]|z.words[2]|z.words[3] == 0) || (*z == bsFieldElement_64_zero_alt)
	return (z.words[0]|z.words[1]|z.words[2]|z.words[3] == 0) || (z.words == [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3})
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

var _ = callcounters.CreateAttachedCallCounter("MulEqFromMontgomery", "", "MulEqFe")

// restoreMontgomery restores the internal Montgomery representation, assuming the current internal representation is *NOT* in Montgomery form.
// This must only be used internally.
func (z *bsFieldElement_64) restoreMontgomery() {
	IncrementCallCounter("MulEqFromMontgomery")
	z.MulEq(&bsFieldElement_64_r)
}

// ToBigInt returns a *big.Int that stores a representation of (a copy of) the given field element.
func (z *bsFieldElement_64) ToBigInt() *big.Int {
	temp := z.words.ToNonMontgomery_fc()
	return temp.ToBigInt()
}

// SetBigInt converts from *big.Int to a field element. The input need not be reduced modulo the field size.
func (z *bsFieldElement_64) SetBigInt(v *big.Int) {
	sign := v.Sign()
	w := new(big.Int).Set(v)
	w.Abs(w)

	// Can be done much more efficiently if desired, but we do not convert often.
	w.Lsh(w, 256) // To account for Montgomery form
	w.Mod(w, baseFieldSize_Int)
	if sign < 0 {
		w.Sub(baseFieldSize_Int, w)
	}
	z.words = utils.BigIntToUIntArray(w)
	// Note z is Normalized.
}

// ToUInt64 returns z with err==nil if z can be represented by a uint64.
//
// If z cannot be represented by a uint64, returns <something, should not be used>, ErrCannotRepresentAsUInt64
func (z *bsFieldElement_64) ToUInt64() (result uint64, err error) {
	temp := z.words.ToNonMontgomery_fc()
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
	var xInt *big.Int = new(big.Int).Rand(rnd, baseFieldSize_Int)
	z.SetBigInt(xInt)
}

// SetRandomUnsafeNonZero generates uniformly random non-zero field elements.
// Note that this is not crypto-grade randomness. This is used in unit-testing only.
// We do NOT guarantee that the distribution is even close to uniform.
func (z *bsFieldElement_64) SetRandomUnsafeNonZero(rnd *rand.Rand) {
	for {
		var xInt *big.Int = new(big.Int).Rand(rnd, baseFieldSize_Int)
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
	overflow1 += overflow4 * twoTo256ModBaseField_64_1 // This computations itself cannot overflow, because 2*rModBaseField_1 + 2 is not large enough
	overflow2 += overflow4 * twoTo256ModBaseField_64_2 // this overflows itself iff overflow4 == 2
	overflow3 += overflow4*twoTo256ModBaseField_64_3 + (overflow4 / 2)

	// Read this as overflow0 := overflow4 * rModBaseField_64_0
	// and mentally rename overflow4 -> overflow0 from here on
	overflow4 *= twoTo256ModBaseField_64_0

	z.words[0], carry = bits.Add64(z.words[0], overflow4, 0)
	z.words[1], carry = bits.Add64(z.words[1], overflow1, carry)
	z.words[2], carry = bits.Add64(z.words[2], overflow2, carry)
	z.words[3], carry = bits.Add64(z.words[3], overflow3, carry)

	// if carry == 1, we need to add 1<<256 mod BaseField == rModBaseField.
	// We do this via bit-masking rather than with an if

	overflow4 = -carry // == carry * 0xFFFFFFFF_FFFFFFFF

	z.words[0], carry = bits.Add64(z.words[0], overflow4&twoTo256ModBaseField_64_0, 0)
	z.words[1], carry = bits.Add64(z.words[1], overflow4&twoTo256ModBaseField_64_1, carry)
	z.words[2], carry = bits.Add64(z.words[2], overflow4&twoTo256ModBaseField_64_2, carry)
	z.words[3], _ = bits.Add64(z.words[3], overflow4&twoTo256ModBaseField_64_3, carry) // _ == 0 is guaranteed

	z.words.reduce_ca()
}

// Inv computes the multiplicative Inverse:
//
// z.Inv(x) performs z:= 1/x. If x is 0, the behaviour is undefined (possibly panic)
func (z *bsFieldElement_64) Inv(x *bsFieldElement_64) {
	IncrementCallCounter("InvFe")
	// Slow, but rarely used anyway (due to working in projective coordinates)
	t := x.ToBigInt()
	if t.ModInverse(t, baseFieldSize_Int) == nil {
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
//
//	The return value tells whether the operation was successful.
//
// If x is not a square, the return value is false and z is untouched.
func (z *bsFieldElement_64) SquareRoot(x *bsFieldElement_64) (ok bool) {
	IncrementCallCounter("SqrtFe")
	xInt := x.ToBigInt()
	if xInt.ModSqrt(xInt, baseFieldSize_Int) == nil {
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

	z.Add(z, y)
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
