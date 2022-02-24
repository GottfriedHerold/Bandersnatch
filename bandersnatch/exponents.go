package bandersnatch

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/bits"
)

// This file contains implementations for the type used to store exponents used in exponentiation algorithms.
// When we are working in the prime-order subgroup, these exponents are naturally viewed as elements in GF(p253), where
// p253 == 0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1 is the size of the subgroup.
// Outside of the prime-order-subgroup, we can view them as elements modulo 2*p253 (the cofactor is 4, but the group structure is Z/2 x Z/2 x Z/p253), which is not prime.
//
// In order not to complicate things (and since doing this in Go is really a pain), we do not want to distinguish the p253 and the 2*p253 cases by having separate types.
// We work modulo 2*p253 for the most part; note, however, that the GLV decomposition needs to work modulo p253.
// For exponentiation algorithms when we do not know whether P is in the subgroup, we reduce to the subgroup case anyway by computing (2n)*P = n*(2P) resp. (2n+1)*P = P + n*(2P).

// Note: The implementation for Exponents is quite different from the implementation FieldElement of the field of definition GF(BaseFieldSize) of the curve.
// For FieldElement, we internally use Montgomery representation to speed up multiplication. For Exponents, we do not multiply often, So we use a "plain" representation.
// If needed, we can implement a "mixed" multiplication -- the main thing we need to do inside the library is compute A * n for exponents n and constant A, so
// we could use Montgomery multiplication anyway.

const (
	curveExponent_0 = (CurveExponent >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	curveExponent_1
	curveExponent_2
	curveExponent_3
)

const (
	curveOrder_0 = (CurveOrder >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	curveOrder_1
	curveOrder_2
	curveOrder_3
)

const (
	groupOrder_0 = (GroupOrder >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	groupOrder_1
	groupOrder_2
	groupOrder_3
)

// Exponent stores an integer value used as an exponent for exponentiation algorithms.
type Exponent struct {
	value [4]uint64 // low-endian, between 0 and curveExponent-1
}

type ScalarField = Exponent

// p253Exponent is the value p253 of type Exponent (which works modulo 2*p253)
var p253Exponent Exponent = Exponent{value: [4]uint64{groupOrder_0, groupOrder_1, groupOrder_2, groupOrder_3}}

// ToBigInt_Full converts the exponent into a *big.Int, working modulo 2*p253
func (z *Exponent) ToBigInt_Full() (result *big.Int) {
	result = uintarrayToInt(&z.value)
	return
}

// ToBigInt_Full converts the exponent into a *big.Int, working modulo p253
func (z *Exponent) ToBigInt_Subgroup() (result *big.Int) {
	temp := z.ModuloP253()
	return temp.ToBigInt_Full()
}

// ModuloP253 reduces the exponent modulo p253 (we usually work internally modulo 2*p253). We do not modify the receiver and return a fresh exponent.
//
// NOTE: This reduction is done automatically for exponentiation algorithms.
// The purpose of this function is mostly for formatted printing via fmt.Printf("...%v...", exponent.ModuloP253())
func (z *Exponent) ModuloP253() (ret Exponent) {
	if z.isNormalized_Subgroup() {
		ret = *z
	} else {
		ret.Sub(&p253Exponent, z)
	}
	return
}

// isNormalized_Subgroup checks whether the exponent value is in the range 0<=. < GroupOrder.
// Note that Exponent works modulo 2*GroupOrder
func (z *Exponent) isNormalized_Subgroup() bool {
	if z.value[3] > groupOrder_3 {
		return false
	} else if z.value[3] < groupOrder_3 {
		return true
	}

	if z.value[2] > groupOrder_2 {
		return false
	} else if z.value[2] < groupOrder_2 {
		return true
	}

	if z.value[1] > groupOrder_1 {
		return false
	} else if z.value[1] < groupOrder_1 {
		return true
	}

	return z.value[0] < groupOrder_0
}

// SetBigInt sets z to the value given by x of type big.Int
func (z *Exponent) SetBigInt(x *big.Int) {
	var xReduced *big.Int = big.NewInt(0)
	xReduced.Mod(x, CurveExponent_Int) // is in 0 <= . < CurveExponent_Int, even if input is negative
	z.value = bigIntToUIntArray(xReduced)
}

// SetUInt sets the value of z to the given unsigned integer.
func (z *Exponent) SetUInt(x uint64) {
	z.value[0] = x
	z.value[1] = 0
	z.value[2] = 0
	z.value[3] = 0
}

// SetInt sets the value of z to the given (signed) integer.
func (z *Exponent) SetInt(x int64) {
	if x >= 0 {
		z.SetUInt(uint64(x))
	} else {
		var borrow uint64
		// Reinterpret x as uint64(this implicitly adds 2^64) and add it to curveExponent-2^64
		z.value[0], borrow = bits.Add64(curveExponent_0, uint64(x), 0)
		z.value[1] = curveExponent_1 - 1 + borrow
		z.value[2] = curveExponent_2
		z.value[3] = curveExponent_3
	}
}

// normalize_once reduces z once by curveExponent, provided the value stores is >= curveExponent.
func (z *Exponent) normalize_once() {
	if !z.isNormalized() {
		var borrow uint64
		z.value[0], borrow = bits.Sub64(z.value[0], curveExponent_0, 0)
		z.value[1], borrow = bits.Sub64(z.value[1], curveExponent_1, borrow)
		z.value[2], borrow = bits.Sub64(z.value[2], curveExponent_2, borrow)
		z.value[3], _ = bits.Sub64(z.value[3], curveExponent_3, borrow)
	}
}

// String is provided to satisfy the fmt.Stringer interface. Note that this is defined on value receivers for convenience.
//
// NOTE: Since Exponent internally works modulo 2*p253 to accomodate for exponentiation outside the p253-subgroup without a separate Exponent type,
// users that work only with subgroups should use z.ModuloP253().String()
func (z Exponent) String() string {
	return z.ToBigInt_Full().String()
}

// Format is provided to satisfy the fmt.Formatter interface. Note that this is defined on value receivers for convenience.
//
// NOTE: Since Exponent internally works modulo 2*p253 to accomodate for exponentiation outside the p253-subgroup without a separate Exponent type,
// users that work only with subgroups should use z.ModuloP253().String()
func (z Exponent) Format(s fmt.State, ch rune) {
	z.ToBigInt_Full().Format(s, ch)
}

/*
func (z *Exponent) maybe_reduce_once() {
	if z.value[3] > curveExponent_3 {
		var borrow uint64
		z.value[0], borrow = bits.Sub64(z.value[0], curveExponent_0, 0)
		z.value[1], borrow = bits.Sub64(z.value[1], curveExponent_1, borrow)
		z.value[2], borrow = bits.Sub64(z.value[2], curveExponent_2, borrow)
		z.value[3], _ = bits.Sub64(z.value[3], curveExponent_3, borrow)
	}
}
*/

// isNormalized checks whether the internally stored value is in 0 <= . < CurveExponent
func (z *Exponent) isNormalized() bool {
	if z.value[3] > curveExponent_3 {
		return false
	} else if z.value[3] < curveExponent_3 {
		return true
	}

	if z.value[2] > curveExponent_2 {
		return false
	} else if z.value[2] < curveExponent_2 {
		return true
	}

	if z.value[1] > curveExponent_1 {
		return false
	} else if z.value[1] < curveExponent_1 {
		return true
	}

	return z.value[0] < curveExponent_0
}

// Add performs addition of exponents.
//
// Use z.Add(&x, &y) to compute z = x+y (modulo 2*p253)
func (z *Exponent) Add(x *Exponent, y *Exponent) {
	var carry uint64
	z.value[0], carry = bits.Add64(x.value[0], y.value[0], 0)
	z.value[1], carry = bits.Add64(x.value[1], y.value[1], carry)
	z.value[2], carry = bits.Add64(x.value[2], y.value[2], carry)
	z.value[3], _ = bits.Add64(x.value[3], y.value[3], carry)
	z.normalize_once()
}

// Sub performs subtraction of exponents.
//
// Use z.Sub(&x, &y) to compute z = x - y (modulo 2*p253)
func (z *Exponent) Sub(x *Exponent, y *Exponent) {
	var borrow uint64
	z.value[0], borrow = bits.Sub64(x.value[0], y.value[0], 0)
	z.value[1], borrow = bits.Sub64(x.value[1], y.value[1], borrow)
	z.value[2], borrow = bits.Sub64(x.value[2], y.value[2], borrow)
	z.value[3], borrow = bits.Sub64(x.value[3], y.value[3], borrow)
	if borrow != 0 {
		z.value[0], borrow = bits.Add64(z.value[0], curveExponent_0, 0)
		z.value[1], borrow = bits.Add64(z.value[1], curveExponent_1, borrow)
		z.value[2], borrow = bits.Add64(z.value[2], curveExponent_2, borrow)
		z.value[3], _ = bits.Add64(z.value[3], curveExponent_3, borrow)
	}
}

// Mul performs multiplication of exponents
//
// Use z.Mul(&x, &y) to compute z = x * y (modulo 2*p253)
func (z *Exponent) Mul(x *Exponent, y *Exponent) {
	xInt := x.ToBigInt_Full()
	yInt := y.ToBigInt_Full()
	zInt := big.NewInt(0)
	zInt.Mul(xInt, yInt)
	z.SetBigInt(zInt)
}

// Neg performs negation of exponents
//
// Use z.Neg(&x) to compute z = -x (modulo 2*p253)
func (z *Exponent) Neg(x *Exponent) {
	// special case of zero, because we want to ensure the result is in 0 <= . < curveExponent
	if x.IsZero_Full() {
		z.SetZero()
		return
	}
	var borrow uint64
	z.value[0], borrow = bits.Sub64(curveExponent_0, z.value[0], 0)
	z.value[1], borrow = bits.Sub64(curveExponent_1, z.value[1], borrow)
	z.value[2], borrow = bits.Sub64(curveExponent_2, z.value[2], borrow)
	z.value[3], _ = bits.Sub64(curveExponent_3, z.value[3], borrow)
}

// SetZero sets the exponent value to 0
func (z *Exponent) SetZero() {
	z.value = [4]uint64{}
}

// SetOne sets the exponent value to 1
func (z *Exponent) SetOne() {
	z.value = [4]uint64{1, 0, 0, 0}
}

// IsZero_Full checks whether the exponent is 0 modulo 2*p253.
//
// Note: Use IsZero_Subgroup to check for zero-ness modulo p253
func (z *Exponent) IsZero_Full() bool {
	return *z == Exponent{}
}

// IsZero_Subgroup checks whether the exponent is 0 modulo p253.
func (z *Exponent) IsZero_Subgroup() bool {
	return *z == Exponent{} || *z == p253Exponent
}

// IsOne_Full checks whether the exponent is 1 modulo 2*p253
func (z *Exponent) IsOne_Full() bool {
	return z.value == [4]uint64{1, 0, 0, 0}
}

// IsOne_Subgroup checks whether the exponent is 1 modulo p253
func (z *Exponent) IsOne_Subgroup() bool {
	return z.value == [4]uint64{1, 0, 0, 0} || z.value == [4]uint64{groupOrder_0 + 1, groupOrder_1, groupOrder_2, groupOrder_3}
}

// glvExponent is an exponent of at most 128 bit (usually 126bit). This is usually the result of a GLV-decomposition.
type glvExponent struct {
	value uint128 // Note that we do NOT compute modulo anything here.
	sign  int     // sign of the number. If value is all-zero, sign can be any value.
}

type uint128 [2]uint64 // low-endian

func add128(x, y uint128) (res uint128) {
	var carry uint64
	res[0], carry = bits.Add64(x[0], y[0], 0)
	res[1], _ = bits.Add64(x[1], y[1], carry)
	return
}

func mul128(x, y uint128) (res uint128) {
	res[1], res[0] = bits.Mul64(x[0], y[0])
	res[1] += x[0]*y[1] + x[1]*y[0]
	return
}

func (z *uint128) SetBigInt(x *big.Int) {
	var temp *big.Int = new(big.Int)
	temp.Mod(x, twoTo128_Int)
	var bigEndianSlice [16]byte
	temp.FillBytes(bigEndianSlice[:])
	z[0] = binary.BigEndian.Uint64(bigEndianSlice[8:16])
	z[1] = binary.BigEndian.Uint64(bigEndianSlice[0:8])
}

func (z uint128) ToBigInt() (x *big.Int) {
	var bigEndianSlice [16]byte
	binary.BigEndian.PutUint64(bigEndianSlice[8:16], z[0])
	binary.BigEndian.PutUint64(bigEndianSlice[0:8], z[1])
	return new(big.Int).SetBytes(bigEndianSlice[:])
}

func (z uint128) Bit(i uint) uint {
	if i < 64 {
		return uint((z[0] >> i) % 2)
	} else {
		return uint((z[1] >> (i - 64)) % 2)
	}
}

// Bit returns the value of the i'th bit of |z|
func (z *glvExponent) Bit(i uint) uint {
	if i < 64 {
		return uint((z.value[0] >> i) % 2)
	} else {
		return uint((z.value[1] >> (i - 64)) % 2)
	}
}

func (z *glvExponent) Sign() int {
	if (z.value == uint128{}) {
		return 0
	} else {
		return z.sign
	}
}

func (z *glvExponent) ToBigInt() (ret *big.Int) {
	ret = z.value.ToBigInt()
	if z.sign < 0 {
		ret.Neg(ret)
	}
	return
}

func (z *glvExponent) SetBigInt(input *big.Int) {
	if input.BitLen() > 128 {
		panic("bandersnatch / exponents: glvExponent can only take values of at most 128bit")
	}
	z.sign = input.Sign()
	var bigEndianByteSlice [16]byte
	input.FillBytes(bigEndianByteSlice[:])
	z.value[0] = binary.BigEndian.Uint64(bigEndianByteSlice[8:16])
	z.value[1] = binary.BigEndian.Uint64(bigEndianByteSlice[0:8])
}

// IsNegative returns whether the exponent is negative. For z==0, the answer is arbitrary.
func (z *glvExponent) IsNegative() bool {
	return z.sign < 0
}

type glvExponents struct {
	U, V glvExponent
}
