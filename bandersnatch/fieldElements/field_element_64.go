package fieldElements

import (
	"encoding/binary"
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

type bsFieldElement_MontgomeryNonUnique struct {
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
	words Uint256
}

// Note: We export *copies* of these variables. Internal functions should use the original.
// This way the compiler has a chance to determine that these value never change and optimize for it.

// representation of zero. This is supposedly a constant.
var bsFieldElement_64_zero bsFieldElement_MontgomeryNonUnique

// alternative representation of zero. Note that we must never call Normalize() on it, which e.g. IsEqual may do.
var bsFieldElement_64_zero_alt bsFieldElement_MontgomeryNonUnique = bsFieldElement_MontgomeryNonUnique{words: [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}}

// The field element 1.
var bsFieldElement_64_one bsFieldElement_MontgomeryNonUnique = bsFieldElement_MontgomeryNonUnique{words: [4]uint64{twoTo256ModBaseField_64_0, twoTo256ModBaseField_64_1, twoTo256ModBaseField_64_2, twoTo256ModBaseField_64_3}}

// The field element -1
var bsFieldElement_64_minusone bsFieldElement_MontgomeryNonUnique = bsFieldElement_MontgomeryNonUnique{words: [4]uint64{minus2To256ModBaseField_64_0, minus2To256ModBaseField_64_1, minus2To256ModBaseField_64_2, minus2To256ModBaseField_64_3}}

// The number 2^256 in Montgomery form.
var bsFieldElement_64_r bsFieldElement_MontgomeryNonUnique = bsFieldElement_MontgomeryNonUnique{words: [4]uint64{0: twoTo512ModBaseField_64_0, 1: twoTo512ModBaseField_64_1, 2: twoTo512ModBaseField_64_2, 3: twoTo512ModBaseField_64_3}}

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
func (z *bsFieldElement_MontgomeryNonUnique) isNormalized() bool {
	return z.words.is_fully_reduced()
}

// Normalize() changes the internal representation of z to a unique number in 0 <= . < BaseFieldSize
// After a call to z.Normalize(), we are guaranteed that read-operations on z will no longer potentially
// change the internal representation until we write to z again.
//
// After a call to Normalize on both operands, the default == operator does the right thing.
// This is mostly an internal function, but it might be needed for compatibility with other libraries that scan the internal byte representation (for hashing, say)
// or when using bsFieldElement_64 as keys for a map or when sharing a field element between multiple goroutines.
func (z *bsFieldElement_MontgomeryNonUnique) Normalize() {
	z.words.Reduce_fb()
}

// Sign outputs the "sign" of the field element.
// More precisely, consider the integer representation z of minimal absolute value (i.e between -BaseField/2 < . < BaseField/2) and take its sign.
// The return value is in {-1,0,+1}.
// This is not compatible with addition or multiplication. It has the property that Sign(z) == -Sign(-z), which is the main thing we need.
// We also might use the fact that positive-sign field elements start with 00 in their high-endian serializiation.
func (z *bsFieldElement_MontgomeryNonUnique) Sign() int {
	if z.IsZero() {
		return 0
	}
	// we take the sign of the non-Montgomery form.
	// Of course, the property that Sign(z) == -Sign(-z) would hold either way (and not switching would actually be more efficient).
	// However, Sign() enters into (De)Serialization routines for curve points. This choice is probably more portable.
	var nonMontgomery Uint256 = z.words.ToNonMontgomery_fc()

	var mhalf_copy Uint256 = [4]uint64{minusOneHalfModBaseField_64_0, minusOneHalfModBaseField_64_1, minusOneHalfModBaseField_64_2, minusOneHalfModBaseField_64_3}

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
func (z *bsFieldElement_MontgomeryNonUnique) Jacobi() int {
	IncrementCallCounter("Jacobi")
	tempInt := z.ToBigInt()
	return big.Jacobi(tempInt, baseFieldSize_Int)
}

// Add is used to perform addition.
//
// Use z.Add(&x, &y) to add x + y and store the result in z.
func (z *bsFieldElement_MontgomeryNonUnique) Add(x, y *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("AddFe")

	z.words.addAndReduce_b_c(&x.words, &y.words)
}

// Sub is used to perform subtraction.
//
// Use z.Sub(&x, &y) to compute x - y and store the result in z.
func (z *bsFieldElement_MontgomeryNonUnique) Sub(x, y *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("SubFe")
	z.words.SubAndReduce_c(&x.words, &y.words)
}

var _ = callcounters.CreateAttachedCallCounter("SubFromNeg", "Subtractions called by Neg", "SubFe").
	AddToThisFromSource("NegFe", 1).
	AddThisToTarget("FieldOps", -1)

// Neg computes the additive inverse (i.e. -x)
//
// Use z.Neg(&x) to set z = -x
func (z *bsFieldElement_MontgomeryNonUnique) Neg(x *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("NegFe")
	// IncrementCallCounter("SubFromNeg") -- done automatically
	z.Sub(&bsFieldElement_64_zero_alt, x) // using alt here makes the if borrow!=0 in Sub unlikely.

}

// Mul computes multiplication in the field.
//
// Use z.Mul(&x, &y) to set z = x * y
func (z *bsFieldElement_MontgomeryNonUnique) Mul(x, y *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("MulFe")

	z.words.MulMontgomerySlow_c(&x.words, &y.words)
}

// IsZero checks whether the field element is zero
func (z *bsFieldElement_MontgomeryNonUnique) IsZero() bool {
	// return (z.words[0]|z.words[1]|z.words[2]|z.words[3] == 0) || (*z == bsFieldElement_64_zero_alt)
	return (z.words[0]|z.words[1]|z.words[2]|z.words[3] == 0) || (z.words == [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3})
}

// IsOne checks whether the field element is 1
func (z *bsFieldElement_MontgomeryNonUnique) IsOne() bool {
	// Note: The representation of 1 is unique:
	// bsFieldElement_64_one.words corresponds to 2^256 - 2*BaseField, so the other (potential) Montgomery representation
	// would be 2^256-1*BaseFieldSize, which (barely) violates our invariant that addition of BaseFieldSize does not overflow.
	return *z == bsFieldElement_64_one
}

// SetOne sets the field element to 1.
func (z *bsFieldElement_MontgomeryNonUnique) SetOne() {
	z.words = bsFieldElement_64_one.words
}

// SetZero sets the field element to 0.
func (z *bsFieldElement_MontgomeryNonUnique) SetZero() {
	z.words = Uint256{}
}

var _ = callcounters.CreateAttachedCallCounter("MulEqFromMontgomery", "", "MulEqFe")

// restoreMontgomery restores the internal Montgomery representation, assuming the current internal representation is *NOT* in Montgomery form.
// This must only be used internally.
func (z *bsFieldElement_MontgomeryNonUnique) restoreMontgomery() {
	IncrementCallCounter("MulEqFromMontgomery")
	z.MulEq(&bsFieldElement_64_r)
}

// ToBigInt returns a *big.Int that stores a representation of (a copy of) the given field element.
func (z *bsFieldElement_MontgomeryNonUnique) ToBigInt() *big.Int {
	temp := z.words.ToNonMontgomery_fc()
	return temp.ToBigInt()
}

// SetBigInt converts from *big.Int to a field element. The input need not be reduced modulo the field size.
func (z *bsFieldElement_MontgomeryNonUnique) SetBigInt(v *big.Int) {
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

// TODO: Replace SetBigInt by this

// _fromBigInt converts from a [*big.Int] to a field element.
//
// The input may be arbitrariliy large or negative and will automatically be reduced modulo BaseFieldSize.
func (z *bsFieldElement_MontgomeryNonUnique) _fromBigInt(v *big.Int) {
	sign := v.Sign()
	w := new(big.Int).Abs(v)
	w.Mod(w, baseFieldSize_Int)
	z.words.FromBigInt(w)
	z.words.FromMontgomeryRepresentation_fc(&z.words)
	if sign < 0 {
		z.words.Sub(&baseFieldSize_uint256, &z.words)
	}
	// Note: z is fully normalized
}

// ToUint64 returns z with err==nil if z can be represented by a uint64.
//
// If z cannot be represented by a uint64, returns <something, should not be used>, ErrCannotRepresentAsUInt64
func (z *bsFieldElement_MontgomeryNonUnique) ToUint64() (result uint64, err error) {
	temp := z.words.ToNonMontgomery_fc()
	result = temp[0]
	if (temp[1] | temp[2] | temp[3]) != 0 {
		err = ErrCannotRepresentAsUint64
	}
	return
}

// SetUint64 sets z to the given value.
func (z *bsFieldElement_MontgomeryNonUnique) SetUint64(value uint64) {
	// Sets z.words to the correct value (not in Montgomery form)
	z.words[0] = value
	z.words[1] = 0
	z.words[2] = 0
	z.words[3] = 0
	// put in Montgomery form
	z.words.ConvertToMontgomeryRepresentation_c(&z.words)
}

// TODO: Make more efficient

func (z *bsFieldElement_MontgomeryNonUnique) SetInt64(value int64) {
	if value >= 0 {
		z.words[0] = uint64(value)
		z.words[1] = 0
		z.words[2] = 0
		z.words[3] = 0

	} else {
		z.words[0] = uint64(-value)
		z.words[1] = 0
		z.words[2] = 0
		z.words[3] = 0
		z.NegEq()
	}
	z.words.ConvertToMontgomeryRepresentation_c(&z.words)
}

func (z *bsFieldElement_MontgomeryNonUnique) ToInt64() (result int64, err error) {
	var temp Uint256
	temp.FromMontgomeryRepresentation_fc(&z.words)
	if (temp[1]|temp[2]|temp[3] == 0) && (temp[0]>>63 == 0) {
		result = int64(temp[0])
		return
	}
	temp.Sub(&baseFieldSize_uint256, &temp) // No modular reduction here
	// Note: need to get -2^63 right (negative range is larger than positive range)
	if (temp[1]|temp[2]|temp[3] == 0) && (temp[0] <= 1<<63) {
		result = int64(-temp[0])
		return
	}
	err = ErrCannotRepresentAsInt64
	return
}

// temporarily exported. Needs some restructing to unexport.

// SetRandomUnsafe generates a uniformly random field element.
// Note that this is not crypto-grade randomness. This is used in unit-testing only.
// We do NOT guarantee that the distribution is even close to uniform.
func (z *bsFieldElement_MontgomeryNonUnique) SetRandomUnsafe(rnd *rand.Rand) {
	// Not the most efficient way (transformation to Montgomery form is obviously not needed), but for testing purposes we want the _64 and _8 variants to have the same output for given random seed.
	var xInt *big.Int = new(big.Int).Rand(rnd, baseFieldSize_Int)
	z.SetBigInt(xInt)
}

// NOTE: The bsFieldElement_MontgomeryNonUnique implementation actually works for x c-reduced,
// but we don't want to promise that.

// SetUint256 sets the field element z from the Uint256 x. Note that we do not ask for x to be in the [0, BaseFieldSize) range; we reduce as needed.
func (z *bsFieldElement_MontgomeryNonUnique) SetUint256(x *Uint256) {
	z.words = *x
	z.words.Reduce_ca()
	z.words.ConvertToMontgomeryRepresentation_c(&z.words)
}

// ToUint256 converts a field element z to a Uint256 in the range [0, BaseFieldSize)
//
// Use z.ToUint256(&x) for x := z.
//
// NOTE: Having the caller provide a pointer to x (rather than returning a Uint256) is done for effiency during internal usage.
func (z *bsFieldElement_MontgomeryNonUnique) ToUint256(x *Uint256) {
	if z == nil {
		x.FromMontgomeryRepresentation_fc(nil)
	} else {
		x.FromMontgomeryRepresentation_fc(&z.words)
	}
}

// SetRandomUnsafeNonZero generates uniformly random non-zero field elements.
// Note that this is not crypto-grade randomness. This is used in unit-testing only.
// We do NOT guarantee that the distribution is even close to uniform.
//
// DEPRECATED
func (z *bsFieldElement_MontgomeryNonUnique) SetRandomUnsafeNonZero(rnd *rand.Rand) {
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

// TODO: Move to uint256_modular.go

// MulEqFive computes z *= 5.
// This is useful, because the coefficient of a in the twisted Edwards representation of Bandersnatch is a=-5
func (z *bsFieldElement_MontgomeryNonUnique) MulEqFive() {
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
	// For the other overflows, we have 0 <= overflow1,2,3 <= 4
	// overflow4 contributes overflow4 * 2^256 == overflow4 * twoTo256ModBaseField (mod BaseField) to the total result.
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

	z.words.Reduce_ca()
}

// MulFive computes multiplication by five
//
// Use z.MulFive(&x) to compute z = x*5.
// This is useful, because the coefficient of a in the twisted Edwards representation of Bandersnatch is a=-5
func (z *bsFieldElement_MontgomeryNonUnique) MulFive(x *bsFieldElement_MontgomeryNonUnique) {
	*z = *x
	z.MulEqFive()
}

// TODO: Specify the behaviour for 0

// Inv computes the multiplicative Inverse:
//
// z.Inv(x) performs z:= 1/x. If x is 0, the behaviour is undefined (possibly panic)
func (z *bsFieldElement_MontgomeryNonUnique) Inv(x *bsFieldElement_MontgomeryNonUnique) {
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
func (z *bsFieldElement_MontgomeryNonUnique) Divide(num *bsFieldElement_MontgomeryNonUnique, denom *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("DivideFe")

	var temp bsFieldElement_MontgomeryNonUnique // needed, because some of z, num, denom might alias
	temp.Inv(denom)
	z.Mul(num, &temp)
}

// IsEqual compares two field elements for equality, i.e. it checks whether z == x (mod BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) IsEqual(x *bsFieldElement_MontgomeryNonUnique) bool {
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
// NOTE: For non-zero squares x, there are two possible square roots.
// We do not guarantee that the choice is deterministic. Multiple calls with the same x can give different z's
func (z *bsFieldElement_MontgomeryNonUnique) SquareRoot(x *bsFieldElement_MontgomeryNonUnique) (ok bool) {
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
func (z bsFieldElement_MontgomeryNonUnique) Format(s fmt.State, ch rune) {
	z.ToBigInt().Format(s, ch)
}

// String is provided to satisfy the fmt.Stringer interface. Note that this is defined on a *value* receiver.
func (z bsFieldElement_MontgomeryNonUnique) String() string {
	z.Normalize()
	return z.ToBigInt().String()
}

var _ = callcounters.CreateAttachedCallCounter("AddEqFe", "", "AddFe")

// var _ = callcounters.CreateHierarchicalCallCounter("AddEqFe", "", "AddSubFe")

// AddEq implements += for field elements.
//
// z.AddEq(&x) is equvalent to z.Add(&z, &x)
func (z *bsFieldElement_MontgomeryNonUnique) AddEq(y *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("AddEqFe")

	z.Add(z, y)
}

var _ = callcounters.CreateAttachedCallCounter("SubEqFe", "", "SubFe")

// SubEq implements the -= operation.
//
// z.SubEq(&x) is equivalent to z.Add(&z, &x)
func (z *bsFieldElement_MontgomeryNonUnique) SubEq(x *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("SubEqFe")
	z.Sub(z, x)
}

var _ = callcounters.CreateAttachedCallCounter("MulEqFe", "", "MulFe")

// MulEq implements the *= operation.
//
// z.MulEq(&x) is equivalent to z.Mul(&z, &x)
func (z *bsFieldElement_MontgomeryNonUnique) MulEq(x *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("MulEqFe")
	z.Mul(z, x)
}

var _ = callcounters.CreateAttachedCallCounter("MulFromSquare", "as part of non-optimized Squaring", "MulFe").
	AddToThisFromSource("Squarings", +1).
	AddThisToTarget("Multiplications", -1)

// Square squares the field element, computing z = x * x
//
// z.Square(&x) is equivalent to z.Mul(&x, &x)
func (z *bsFieldElement_MontgomeryNonUnique) Square(x *bsFieldElement_MontgomeryNonUnique) {
	IncrementCallCounter("Squarings")
	z.Mul(x, x)
}

// SquareEq replaces the field element by its squared value.
//
// z.SquareEq() is equivalent to z.Square(&z)
func (z *bsFieldElement_MontgomeryNonUnique) SquareEq() {
	IncrementCallCounter("Squarings")
	z.Mul(z, z) // or z.Square(z), but it's the same for now (except for the need to adjust call counters)
}

// Double computes the double of a point, i.e. z = 2*x == x + x
//
// z.Double(&x) is equivalent to z.Add(&x, &x)
func (z *bsFieldElement_MontgomeryNonUnique) Double(x *bsFieldElement_MontgomeryNonUnique) {
	z.Add(x, x)
}

// DoubleEq replaces the provided field element by its doubled value, i.e. computes z *= 2
//
// z.DoubleEq() is equivalent to z.Double(&z)
func (z *bsFieldElement_MontgomeryNonUnique) DoubleEq() {
	z.Add(z, z)
}

var _ = callcounters.CreateAttachedCallCounter("NegEqFe", "", "NegFe")

// NegEq replaces the provided field element by its negative.
//
// z.NegEq() is equivalent to z.Neg(&z)
func (z *bsFieldElement_MontgomeryNonUnique) NegEq() {
	IncrementCallCounter("NegEqFe")
	z.Neg(z)
}

var _ = callcounters.CreateAttachedCallCounter("InvEqFe", "", "InvFe")

// TODO: Consider specifying what happens at 0.

// InvEq replaces the provided field element by its multiplicative inverse (in the field, i.e. modulo BaseFieldSize).
// The behaviour is unspecified (potentially panic) if z is zero.
//
// z.InvEq is equivalent to z.Inv(&z)
func (z *bsFieldElement_MontgomeryNonUnique) InvEq() {
	z.Inv(z)
}

var _ = callcounters.CreateAttachedCallCounter("DivideEqFe", "", "DivideFe")

// DivideEq performs a z /= x operation
// The behaviour is undefined (potentially panic) if x is zero.
//
// z.DivideEq(&x) is equivalent to z.Divide(&z, &x) for non-zero x
func (z *bsFieldElement_MontgomeryNonUnique) DivideEq(denom *bsFieldElement_MontgomeryNonUnique) {
	z.Divide(z, denom)
}

// This is essentially a helper function that we need in several places.

// CmpAbs compares the absolute values of two field elements and whether the signs match:
//
// absEqual is true iff x == +/- z
// exactly equal is true iff x == z
func (z *bsFieldElement_MontgomeryNonUnique) CmpAbs(x *bsFieldElement_MontgomeryNonUnique) (absEqual bool, exactlyEqual bool) {
	if z.IsEqual(x) {
		return true, true
	}
	var tmp bsFieldElement_MontgomeryNonUnique
	tmp.Neg(x)
	if tmp.IsEqual(z) {
		return true, false
	}
	return false, false
}

// RerandomizeRepresentation changes the internal representation to an equivalent one, taking randomness from the uniform seed.
//
// NOTES: This is an internal function that is used to make differential tests generic.
// We cannot take an *rand.rng here, but rather assume that seed fresh uniform randomness. Taking bits of seed directly is fine.
//
// We ask that if z1.IsEqual(&z2), then after
//
//	z1.RerandomizeRepresentation(s)
//	z2.RerandomizeRepresentation(s)
//
// we have z1 == z2 (equal internal representation)
func (z *bsFieldElement_MontgomeryNonUnique) RerandomizeRepresentation(seed uint64) {
	z.words.Reduce_fb()
	if seed&1 == 1 {
		z.words.Add(&z.words, &baseFieldSize_uint256)
		if !z.words.IsReduced_c() {
			z.words.Sub(&z.words, &baseFieldSize_uint256)
		}
	}
}

// ToBytes writes an unspecified internal representation of itself to buf[0:LEN], where LEN==32 can be obtained from [BytesLength]
//
// ToBytes and [SetBytes] perform raw serialization, therefore exposing the internal representation of field elements.
// This representation may be non-unique, non-obvious and is NOT part of the API.
// As a consequence of the latter, the output format is not stable and unsuited for serializing to disk.
// We only guarantee that we can read back the bytes using SetBytes for the same library version on the same architecture with the same data type.
// Doing so guarantees that the internal representation is preserved.
//
// It is up to the caller to ensure the buffer is large enough.
func (z *bsFieldElement_MontgomeryNonUnique) ToBytes(buf []byte) {
	binary.LittleEndian.PutUint64(buf[0:8], z.words[0])
	binary.LittleEndian.PutUint64(buf[8:16], z.words[1])
	binary.LittleEndian.PutUint64(buf[16:24], z.words[2])
	binary.LittleEndian.PutUint64(buf[24:32], z.words[3])
}

// SetBytes sets a field element from buf[0:LEN] that was written by [ToBytes], where LEN==32 can be obtained from [BytesLength]
//
// ToBytes and SetBytes perform raw (de)serialization, therefore exposing the internal representation of field elements.
// This representation may be non-unique, non-obvious and is NOT part of the API.
// As a consequence of the latter, the output format is not stable and unsuited for serializing to disk.
// We only guarantee that we can read back the bytes using SetBytes for the same library version on the same architecture with the same data type.
// Doing so guarantees that the internal representation is preserved.
//
// It is up to the caller to ensure the buffer is large enough.
// SetBytes performs no validity checks and using it with aribtrary byte slices can silently violate internal consistency conditions.
func (z *bsFieldElement_MontgomeryNonUnique) SetBytes(buf []byte) {
	z.words[0] = binary.LittleEndian.Uint64(buf[0:8])
	z.words[1] = binary.LittleEndian.Uint64(buf[8:16])
	z.words[2] = binary.LittleEndian.Uint64(buf[16:24])
	z.words[3] = binary.LittleEndian.Uint64(buf[24:32])
}

func (z *bsFieldElement_MontgomeryNonUnique) BytesLength() int { return 32 }

// IsEqualAsBigInt converts the argument and itself to [*big.Int]s and checks for equality.
// This function is not very efficient and should only be used in testing.
func (z *bsFieldElement_MontgomeryNonUnique) IsEqualAsBigInt(x interface{ ToBigInt() *big.Int }) bool {
	xInt := x.ToBigInt()
	zInt := z.ToBigInt()
	return xInt.Cmp(zInt) == 0
}

// AddInt64 performs addition of a field element and an int64.
//
// More precisely, z.AddInt64(&x, y) sets z := x+y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) AddInt64(x *bsFieldElement_MontgomeryNonUnique, y int64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetInt64(y)
	z.Add(x, &yFE)
}

// SubInt64 performs subtraction of a field element minus an int64.
//
// More precisely, z.SubInt64(&x, y) sets z := x-y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) SubInt64(x *bsFieldElement_MontgomeryNonUnique, y int64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetInt64(y)
	z.Sub(x, &yFE)
}

// AddInt64 performs multiplication of a field element and an int64.
//
// More precisely, z.MulInt64(&x, y) sets z := x*y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) MulInt64(x *bsFieldElement_MontgomeryNonUnique, y int64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetInt64(y)
	z.Mul(x, &yFE)
}

// DivideInt64 performs division of a field element by an int64.
//
// More precisely, z.DivideInt64(&x, y) sets z := x+y (modulo BaseFieldSize).
// If y == 0, this function panics.
func (z *bsFieldElement_MontgomeryNonUnique) DivideInt64(x *bsFieldElement_MontgomeryNonUnique, y int64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetInt64(y)
	z.Divide(x, &yFE)
}

// AddUint64 performs addition of a field element and an uint64.
//
// More precisely, z.AddUint64(&x, y) sets z := x+y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) AddUint64(x *bsFieldElement_MontgomeryNonUnique, y uint64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetUint64(y)
	z.Add(x, &yFE)
}

// SubUint64 performs subtraction of a field element minus an uint64.
//
// More precisely, z.SubUint64(&x, y) sets z := x-y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) SubUint64(x *bsFieldElement_MontgomeryNonUnique, y uint64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetUint64(y)
	z.Sub(x, &yFE)
}

// MulUint64 performs multiplication of a field element by an uint64.
//
// More precisely, z.MulUint64(&x, y) sets z := x*y (modulo BaseFieldSize)
func (z *bsFieldElement_MontgomeryNonUnique) MulUint64(x *bsFieldElement_MontgomeryNonUnique, y uint64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetUint64(y)
	z.Mul(x, &yFE)
}

// DivideUint64 performs division of a field element by an uint64.
//
// More precisely, z.DivideUint64(&x, y) sets z := x/y (modulo BaseFieldSize)
// if y == 0, this function panics.
func (z *bsFieldElement_MontgomeryNonUnique) DivideUint64(x *bsFieldElement_MontgomeryNonUnique, y uint64) {
	var yFE bsFieldElement_MontgomeryNonUnique
	yFE.SetUint64(y)
	z.Divide(x, &yFE)
}
