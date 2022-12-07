package fieldElements

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file is part of the implementation of the Uint256 (and also a similar Uint512) data type.
// Uint256 is a 256-bit unsigned integer data type used (mostly internally) to implement our field element types.
//
// Note that a Uint256 is an integer, not a residue, so arithmetic is as for usual uints, i.e. modulo 2^256.
// Funtions and Methods that operate on Uint256's and perform modular arithmetic explicitly say so in their description and function name.
// These are contained in different files.

// NOTE: The set of exported functions and methods that perform *modular* arithmetic is not particularly stable;
// we export it mostly to enable certain advanced optimizations outside the package (mixed Montgomery multiplication, for instance) or for users who want to perform extensive computations in the base field.

// Note that the code is split into 3 parts:
//   uint256.go (integer arithmetic / arithmetic modulo 2^256) -- this file
//   uint256_modular.go (arithmetic that works modulo BaseFieldSize)
//   uint256_montgomery.go (Montgomery arithmetic)
//

// Uint256 is a 256-bit (unsigned) integer.
//
// We provide methods for elementary arithmetic and for arithmetic modulo BaseFieldSize (the latter explicitly say they perform moular reduction)
// This type is based on [4]uint64 with low-endian convention as part of the API, so x[i] will retrieve the i'th (low-endian) uint64.
//
// Note that this type is mostly for internal and cross-package usage; we do not guarantee that the exported methods (and their names) are stable.
type Uint256 [4]uint64 // low-endian

// Uint512 is a 512-bit (unsigned) integer.
//
// This type works mostly like [Uint256], but we provide relatively little functionaliy, as this type only matters for intermediate results.
type Uint512 [8]uint64 // low-endian

// ToBigInt converts the given uint256 to a [*big.Int]
func (z *Uint256) ToBigInt() *big.Int {
	// convert uint256 to big-endian (because big.Int's SetBytes takes a big-endian byte slice)
	var big_endian_byte_slice [32]byte
	binary.BigEndian.PutUint64(big_endian_byte_slice[0:8], z[3])
	binary.BigEndian.PutUint64(big_endian_byte_slice[8:16], z[2])
	binary.BigEndian.PutUint64(big_endian_byte_slice[16:24], z[1])
	binary.BigEndian.PutUint64(big_endian_byte_slice[24:32], z[0])

	// convert to *big.Int
	return new(big.Int).SetBytes(big_endian_byte_slice[:])
}

// ToBigInt converts the given uint512 to a [*big.Int]
func (z *Uint512) ToBigInt() *big.Int {
	// convert uint256 to big-endian (because big.Int's SetBytes takes a big-endian byte slice)
	var big_endian_byte_slice [64]byte
	binary.BigEndian.PutUint64(big_endian_byte_slice[0:8], z[7])
	binary.BigEndian.PutUint64(big_endian_byte_slice[8:16], z[6])
	binary.BigEndian.PutUint64(big_endian_byte_slice[16:24], z[5])
	binary.BigEndian.PutUint64(big_endian_byte_slice[24:32], z[4])
	binary.BigEndian.PutUint64(big_endian_byte_slice[32:40], z[3])
	binary.BigEndian.PutUint64(big_endian_byte_slice[40:48], z[2])
	binary.BigEndian.PutUint64(big_endian_byte_slice[48:56], z[1])
	binary.BigEndian.PutUint64(big_endian_byte_slice[56:64], z[0])

	// convert to *big.Int
	return new(big.Int).SetBytes(big_endian_byte_slice[:])
}

// NOTE: fmt's printing routines will always pick Format over String, even for "%v" or fmt.Sprint -- as such the String() method will rarely be called.

// String is provided to satisfy the [fmt.Stringer] interface. Note that this is defined on value receivers.
func (z Uint256) String() string {
	return z.ToBigInt().String()
}

// String is provided to satisfy the [fmt.Stringer] interface. Note that this is defined on value receivers.
func (z Uint512) String() string {
	return z.ToBigInt().String()
}

// Format is provided to satisfy the [fmt.Formatter] interface. Note that this is defined on value receivers.
//
// We internally convert to [*big.Int] and hence support the same formats as [big.Int].
func (z Uint256) Format(s fmt.State, ch rune) {
	z.ToBigInt().Format(s, ch)
}

// Format is provided to satisfy the [fmt.Formatter] interface. Note that this is defined on value receivers.
//
// We internally convert to [*big.Int] and hence support the same formats as [big.Int].
func (z Uint512) Format(s fmt.State, ch rune) {
	z.ToBigInt().Format(s, ch)
}

// TODO: Move BigIntToUIntArray into this package.
// (Currently not done this way because of overlapping use in the exponents package -- this will change)

// BigIntToUInt256 converts a [*big.Int] to a uint256.
//
// This function panics if x is not between 0 and 2^256 - 1.
func BigIntToUInt256(x *big.Int) (result Uint256) {
	return utils.BigIntToUIntArray(x)
}

// BigIntToUint512 converts a [*big.Int] to a uint512
//
// This function panics if x is not between 0 and 2^512 - 1
func BigIntToUint512(x *big.Int) (result Uint512) {
	if x.Sign() < 0 {
		panic(ErrorPrefix + "Uint512.FromBigInt: Trying to convert negative big.Int")
	}
	if x.BitLen() > 512 {
		panic(ErrorPrefix + "Uint512.FromBigInt: big int too large to fit into uint512")
	}
	var bigEndianByteSlice [64]byte
	x.FillBytes(bigEndianByteSlice[:])
	result[0] = binary.BigEndian.Uint64(bigEndianByteSlice[56:64])
	result[1] = binary.BigEndian.Uint64(bigEndianByteSlice[48:56])
	result[2] = binary.BigEndian.Uint64(bigEndianByteSlice[40:48])
	result[3] = binary.BigEndian.Uint64(bigEndianByteSlice[32:40])
	result[4] = binary.BigEndian.Uint64(bigEndianByteSlice[24:32])
	result[5] = binary.BigEndian.Uint64(bigEndianByteSlice[16:24])
	result[6] = binary.BigEndian.Uint64(bigEndianByteSlice[8:16])
	result[7] = binary.BigEndian.Uint64(bigEndianByteSlice[0:8])
	return
}

// SetBigInt sets z:=x, where x is a [*big.Int].
//
// We assume that 0 <= x < 2**256, else we panic.
func (z *Uint256) SetBigInt(x *big.Int) {
	*z = utils.BigIntToUIntArray(x)
}

// SetBigInt sets z:=x, where x is a [*big.Int].
//
// We assume that 0 <= x < 2**512, else we panic.
func (z *Uint512) SetBigInt(x *big.Int) {
	*z = BigIntToUint512(x)
}

// SetUint64 converts a uint64 to a Uint256.
//
// Usage: z.SetUint64(x) sets z := x, where x is a uint64
func (z *Uint256) SetUint64(x uint64) {
	*z = Uint256{x, 0, 0, 0}
}

// SetUint64 converts a uint64 to a Uint512.
//
// Usage: z.SetUint64(x) sets z := x, where x is a uint64
func (z *Uint512) SetUint64(x uint64) {
	*z = Uint512{x, 0, 0, 0, 0, 0, 0, 0}
}

// ToUint256 performs *x := *z. There is not really any need to call this; it is provided to satisfy the [ToUint256Convertible] interface.
func (z *Uint256) ToUint256(x *Uint256) {
	*x = *z
}

// SetUint256 sets z := x. Note that plain assignment works just fine. This method solely exists to unify interfaces.
func (z *Uint256) SetUint256(x *Uint256) {
	*z = *x
}

// SetUint256 sets z := x, converting a Uint256 to a Uint512 (with highest words set to 0)
func (z *Uint512) SetUint256(x *Uint256) {
	z[0] = x[0]
	z[1] = x[1]
	z[2] = x[2]
	z[3] = x[3]
	z[4] = 0
	z[5] = 0
	z[6] = 0
	z[7] = 0
}

// ToUint256 converts a (sufficiently small) Uint512 to Uint256, setting x := z. The API is chosen to satisfy the [ToUint256Convertible] interface.
//
// If z does not fit into a Uint256, this method panics
func (z *Uint512) ToUint256(x *Uint256) {
	if z[4]|z[5]|z[6]|z[7] != 0 {
		panic(ErrorPrefix + "called ToUint256 on Uint512 that did not fit into a Uint256")
	}
	x[0] = z[0]
	x[1] = z[1]
	x[2] = z[2]
	x[3] = z[3]
}

// Q: Should we have a signed int64 -> Uint256 conversion
// Q: Should we have a uint256 -> uint64 conversion (same API as in FieldElementInterface)

// BitLen returns the length of z in bits. This means that we return the smallest i>=0, s.t. z < 2^i.
// The bitlength of 0 is 0.
func (z *Uint256) BitLen() int {
	l := bits.Len64(z[3])
	if l != 0 {
		return l + 3*64
	}
	l = bits.Len64(z[2])
	if l != 0 {
		return l + 2*64
	}
	l = bits.Len64(z[1])
	if l != 0 {
		return l + 64
	}
	return bits.Len64(z[0])
}

// Add computes an addition z := x + y.
// The addition is carried out modulo 2^256
func (z *Uint256) Add(x, y *Uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], _ = bits.Add64(x[3], y[3], carry)
}

// Note: We don't have a variant that take a carry as input, as we don't need it.
// carries are stored in an uint64 (even though it's 0/1-valued). This is consistent with bits.Add64 etc.

// AddAndReturnCarry computes an addition z := x + y.
// The addition is carried out modulo 2^256
// Returns the carry value
func (z *Uint256) AddAndReturnCarry(x, y *Uint256) (carry uint64) {
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)
	return
}

// Sub computes subtraction z := x - y, modulo 2^256
func (z *Uint256) Sub(x, y *Uint256) {
	var borrow uint64 // only takes values 0,1
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], _ = bits.Sub64(x[3], y[3], borrow)
}

// SubAndReturnBorrow computes subtraction z := x - y, modulo 2^256
// returns borrow bit
func (z *Uint256) SubAndReturnBorrow(x, y *Uint256) (borrow uint64) {
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)
	return

}

// Mul computes z := x * y (modulo 2^256)
func (z *Uint256) Mul(x, y *Uint256) {
	var res0, res1, res2, res3 uint64 // result. z might alias x or y, so we store in res.
	var high, low, carry uint64
	res1, res0 = bits.Mul64(x[0], y[0])

	res2, low = bits.Mul64(x[0], y[1])
	res1, carry = bits.Add64(res1, low, 0) // carry -> res2

	res3, low = bits.Mul64(x[0], y[2])
	res2, carry = bits.Add64(res2, low, carry) // carry -> res3

	_, low = bits.Mul64(x[0], y[3])
	res3, _ = bits.Add64(res3, low, carry)

	high, low = bits.Mul64(x[1], y[0]) // low -> res1, high -> res2
	res1, carry = bits.Add64(res1, low, 0)
	res2, carry = bits.Add64(res2, high, carry) // carry -> res3

	_, low = bits.Mul64(x[1], y[2])
	res3, _ = bits.Add64(res3, low, carry)

	high, low = bits.Mul64(x[1], y[1]) // low -> res2, high -> res3
	res2, carry = bits.Add64(res2, low, 0)
	res3, _ = bits.Add64(res3, high, carry)

	high, low = bits.Mul64(x[2], y[0]) // low -> res2, high -> res3
	res2, carry = bits.Add64(res2, low, 0)
	res3, _ = bits.Add64(res3, high, carry)

	_, low = bits.Mul64(x[2], y[1])
	res3 += low

	_, low = bits.Mul64(x[3], y[0])
	res3 += low

	z[0] = res0
	z[1] = res1
	z[2] = res2
	z[3] = res3
}

// Increments computes z = x + 1 (modulo 2^256)
func (z *Uint256) Increment(x *Uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], 1, 0)
	z[1], carry = bits.Add64(x[1], 0, carry)
	z[2], carry = bits.Add64(x[2], 0, carry)
	z[3], _ = bits.Add64(x[3], 0, carry)
}

// IncrementEq computes z += 1 (modulo 2^256)
func (z *Uint256) IncrementEq() {
	var carry uint64
	z[0], carry = bits.Add64(z[0], 1, 0)
	z[1], carry = bits.Add64(z[1], 0, carry)
	z[2], carry = bits.Add64(z[2], 0, carry)
	z[3], _ = bits.Add64(z[3], 0, carry)
}

// Decrement computes z := x - 1 modulo 2^256
func (z *Uint256) Decrement(x *Uint256) {
	var borrow uint64
	z[0], borrow = bits.Sub64(x[0], 1, 0)
	z[1], borrow = bits.Sub64(x[1], 0, borrow)
	z[2], borrow = bits.Sub64(x[2], 0, borrow)
	z[3], _ = bits.Sub64(x[3], 0, borrow)
}

// DecrementEq computes z -= 1 modulo 2^256
func (z *Uint256) DecrementEq() {
	var borrow uint64
	z[0], borrow = bits.Sub64(z[0], 1, 0)
	z[1], borrow = bits.Sub64(z[1], 0, borrow)
	z[2], borrow = bits.Sub64(z[2], 0, borrow)
	z[3], _ = bits.Sub64(z[3], 0, borrow)
}

// SetZero sets the Uint256 to zero.
func (z *Uint256) SetZero() {
	*z = Uint256{}
}

// SetOne sets the Uint256 to one
func (z *Uint256) SetOne() {
	z[0] = 1
	z[1] = 0
	z[2] = 0
	z[3] = 0
}

// IsZero checks whether the uint256 is (exactly) zero.
func (z *Uint256) IsZero() bool {
	return z[0]|z[1]|z[2]|z[3] == 0
}

// IsOne checks whether the Uint256 is (exactly) one.
func (z *Uint256) IsOne() bool {
	return (z[0]-1)|z[1]|z[2]|z[3] == 0
}

// ShiftRightEq_64 right-shifts the internal uint64 array by 64 bit (equivalent to truncated-towards-minus-infinity division by 2^64) and returns the shifted-out uint64
func (z *Uint256) ShiftRightEq_64() (ShiftOut uint64) {
	ShiftOut = z[0]
	z[0] = z[1]
	z[1] = z[2]
	z[2] = z[3]
	z[3] = 0
	return
}

// ShiftRightEq shifts the internal uint64 array by the given amount, with 0 <= amount <= 64
//
// If amount > 64, the behaviour is unspecified
func (z *Uint256) ShiftRightEq(amount uint) {
	z[0] = (z[0] >> amount) | (z[1] << (64 - amount))
	z[1] = (z[1] >> amount) | (z[2] << (64 - amount))
	z[2] = (z[2] >> amount) | (z[3] << (64 - amount))
	z[3] = (z[3] >> amount)
}

// ShiftLeftEq_64 left-shifts the internal uint64 array by 64bit (equivalent to multiplication by 2^64) and returns the shifted-out uint64
func (z *Uint256) ShiftLeftEq_64() (ShiftOut uint64) {
	ShiftOut = z[3]
	z[3] = z[2]
	z[2] = z[1]
	z[1] = z[0]
	z[0] = 0
	return
}

// LongMulUint64 multiplies a 256bit by a 64 bit uint, resulting in a 320-bit uint (stored as low-endian [5]uint64)
//
// Usage is LongMulUint64(&z, &x, y) to compute z := x * y. (The data types will only allow one order anyway)
func LongMulUint64(target *[5]uint64, x *Uint256, y uint64) {
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

// LongMul computes a 256-bit x 256-bit -> 512-bit multiplication, without any modular reduction. z:=x*y
func (z *Uint512) LongMul(x, y *Uint256) {
	var c, t0, t1, q0, q1, q2, q3, q4, q5, q6, q7 uint64

	q2, q1 = bits.Mul64(y[0], x[1])
	q4, q3 = bits.Mul64(y[0], x[3])

	t1, q0 = bits.Mul64(y[0], x[0])
	q1, c = bits.Add64(q1, t1, 0)
	t1, t0 = bits.Mul64(y[0], x[2])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q4, _ = bits.Add64(q4, 0, c)

	t1, t0 = bits.Mul64(y[1], x[1])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	q5, t0 = bits.Mul64(y[1], x[3])
	q4, c = bits.Add64(q4, t0, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(y[1], x[0])
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	t1, t0 = bits.Mul64(y[1], x[2])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(y[2], x[1])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	q6, t0 = bits.Mul64(y[2], x[3])
	q5, c = bits.Add64(q5, t0, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(y[2], x[0])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(y[2], x[2])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(y[3], x[1])
	q4, c = bits.Add64(q4, t0, 0)
	q5, c = bits.Add64(q5, t1, c)
	q7, t0 = bits.Mul64(y[3], x[3])
	q6, c = bits.Add64(q6, t0, c)
	q7, _ = bits.Add64(q7, 0, c)

	t1, t0 = bits.Mul64(y[3], x[0])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	t1, t0 = bits.Mul64(y[3], x[2])
	q5, c = bits.Add64(q5, t0, c)
	q6, c = bits.Add64(q6, t1, c)
	q7, _ = bits.Add64(q7, 0, c)
	z[0] = q0
	z[1] = q1
	z[2] = q2
	z[3] = q3
	z[4] = q4
	z[5] = q5
	z[6] = q6
	z[7] = q7
}

// LongSquare computes a 256-bit to 512-bit squaring operation without modular reduction. z := x*x
func (z *Uint512) LongSquare(x *Uint256) {
	var c, t0, t1, q0, q1, q2, q3, q4, q5, q6, q7 uint64

	q4, q3 = bits.Mul64(x[0], x[3])

	t1, q2 = bits.Mul64(x[0], x[2])
	q3, c = bits.Add64(q3, t1, 0)
	q5, t0 = bits.Mul64(x[1], x[3])
	q4, c = bits.Add64(q4, t0, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, q1 = bits.Mul64(x[0], x[1])
	q2, c = bits.Add64(q2, t1, 0)
	t1, t0 = bits.Mul64(x[1], x[2])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q6, t0 = bits.Mul64(x[2], x[3])
	q5, c = bits.Add64(q5, t0, c)
	q6, _ = bits.Add64(q6, 0, c)

	q1, c = bits.Add64(q1, q1, 0)
	q2, c = bits.Add64(q2, q2, c)
	q3, c = bits.Add64(q3, q3, c)
	q4, c = bits.Add64(q4, q4, c)
	q5, c = bits.Add64(q5, q5, c)
	q6, c = bits.Add64(q6, q6, c)
	q7, _ = bits.Add64(0, 0, c)

	t1, q0 = bits.Mul64(x[0], x[0])
	q1, c = bits.Add64(q1, t1, 0)
	t1, t0 = bits.Mul64(x[1], x[1])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(x[2], x[2])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	t1, t0 = bits.Mul64(x[3], x[3])
	q6, c = bits.Add64(q6, t0, c)
	q7, _ = bits.Add64(q7, t1, c)

	z[0] = q0
	z[1] = q1
	z[2] = q2
	z[3] = q3
	z[4] = q4
	z[5] = q5
	z[6] = q6
	z[7] = q7
}

// Cmp compares x and z. More precisely, z.Cmp(&x) returns
//
//	-1 if z < x
//	 0 if z ==x
//	+1 if z > x
//
// Note that the returned value matches [*big.Int]'s Cmp method
func (z *Uint256) Cmp(x *Uint256) int {
	for i := int(3); i >= 0; i-- {
		if z[i] < x[i] {
			return -1
		}
		if z[i] > x[i] {
			return +1
		}
	}
	return 0
}

// IsLessThan compares two uin256's.
//
// The behaviour is as the name suggests: z.IsLessThan(x) is true iff z < x.
func (z *Uint256) IsLessThan(x *Uint256) bool {
	return z.Cmp(x) == -1
}

type ToUint256Convertible interface {
	ToUint256(*Uint256)
}

// IsEqualAsUint256 converts arg1 and arg2 to Uint256 and compares those for equality.
func IsEqualAsUint256(arg1 ToUint256Convertible, arg2 ToUint256Convertible) bool {
	var u1, u2 Uint256
	arg1.ToUint256(&u1)
	arg2.ToUint256(&u2)
	return u1 == u2
}

// unsignedExponentDecompositionEntry models entries of a decomposition of a Uint256 x into a sum x = exp_0 << pos_0 + exp_1 << pos_1 + ...
//
// such decompositions arise in sliding windows algorithms for exponentiation.
// Note that the actual exponentiation algorithms are in uint256_modular.go and/or uint256_montgomery.go,
// since we only care about modular exponentiation. The decomposition algorithm for sliding window is contained here, though.
type unsignedExponentDecompositionEntry struct {
	exp uint
	pos uint
}

// SlidingWindowDecomposition performs a sliding window decomposition of z with the given window size, which must be strictly between 0 and 64.
//
// The returned decomposition has the following properties:
//
//   - z == sum_i decomposition[i].exp << decomposition[i].pos
//   - The decomposition[i].pos are strictly decreasing
//   - Each decomposition[i].exp is odd
//   - decomposition[i].exp < (1<<windowSize)
//
// For the current implementation, the difference decomposition[i].pos - decomposition[i+1].pos is >= windowSize, but we don't guarantee this.
func (z *Uint256) SlidingWindowDecomposition(windowsize uint8) (decomposition []unsignedExponentDecompositionEntry) {
	if windowsize == 0 {
		panic(ErrorPrefix + "trying a sliding window decomposition with window size 0")
	}
	if windowsize >= 64 {
		panic(ErrorPrefix + "trying a sliding window decomposition with window size >=64")
	}
	var MAXSIZE = (int(windowsize) + 255) / int(windowsize)             // maximal number of entries in the resulting decomposition
	decomposition = make([]unsignedExponentDecompositionEntry, MAXSIZE) // resulting decomposition

	// MAXSIZE - 1 - actualSize is the current number of decomposition entries we have already created.
	// actualSize counts from above towards 0 for efficiency reasons (because we want the result in reverse order)
	var actualSize int = int(MAXSIZE - 1)
	var currentPos int = 0 // index of the bit that we process next, all lower-order bits have already been processed.
	var maxL = int(64 - windowsize)

	var mask_selectLow uint64 = (1 << uint64(windowsize)) - 1    // mask to select the windowSize many least significant bits
	var mask_selectHigh uint64 = math.MaxUint64 - mask_selectLow // mask to select the remaining bits.

	zCopy := *z   // make a working copy of z that we can modify. We generally read off the least significant bits and right-shift zCopy.
	var z0 uint64 // working copy of z[0]

	// for efficiency reasons, we do not operate of zCopy, but on z0 (a copy of zCopy[0]) without chaning zCopy itself. Only when we have processed sufficiently many bits, we actually perfom the accumulated changes to zCopy.
	for {

		// Let zLow be the currentPos many least significant bits of z. We have at this point:
		//   - zCopy == z >> currentPos
		//   - sum_i decomposition[i].pos << decomposition[i].pos == zLow

		z0 = zCopy[0]                 // working copy of z0
		L := bits.TrailingZeros64(z0) // number of trailing bits of zCopy, capped at 64
		if L == 64 {
			// zCopy has >= 64 trailing zeros.
			if zCopy.IsZero() {
				// if zCopy == 0, we are done
				break
			} else {
				// last word of zCopy is 0, we shift everything by 64 bits
				currentPos += 64
				zCopy.ShiftRightEq_64()
				continue
			}
		}

		// We generally delay right-shifting zCopy and only right-shift z0, performing the operation on zCopy only when worthwhile.
		// Hereby, L counts the number of right-shifts we need to perform on zCopy afterwards
		currentPos += L

		// By the above, the invariants transform to
		//   - zCopy >> L == z >> currentPos
		//   - sum_i decomposition[i].pos << decomposition[i].pos == zLow
		// This still holds, because the trailing L bits of zCopy we 0.

		// if L > 64-windowSize we actually commit to zCopy right away and restart the loop.

		z0 >>= L
		for L <= maxL { // maxL == 64 - windowSize
			// invariant: z0 == zCopy[0] >> L, which equals zCopy >> 0 in at least the lowest windowSize bits
			// invariant: z0 is odd

			// actually read off the next entry of the decomposition:
			decomposition[actualSize].pos = uint(currentPos)
			decomposition[actualSize].exp = uint(z0 & mask_selectLow)
			actualSize -= 1

			// Now we remove the entry from z0.
			// We could do
			//   z0 >>= windowSize
			//   L += windowSize
			//   currentPos += windowSize
			// to maintain the sum_i decomposition[i].pos << decomposition[i].pos == zLow and zCopy >> L == z >> currentPos invariants
			// However, this would be followed by furtherZeroBits := bits.TrailingZeros64(z0) and then processing those as well.
			// We instead clear (rather than shift-out) the bits only call TrailingZeros64 once, so zeroBits includes windowSize AND also the next zero bits.

			z0 &= mask_selectHigh
			zeroBits := bits.TrailingZeros64(z0) // guaranteed to be >= windowSize. Let furtherZeroBits := zeroBits - windowSize

			if zeroBits+L > 64 { // we have exhausted z0 and need to consider bits that are only in zCopy.

				// Revert to the above strategy and do NOT include furtherZeroBits.
				// Mentally perform z0 >>= windowSize, but this is unneeded.
				L += int(windowsize) // Note: guaranteed to be <=64
				currentPos += int(windowsize)
				break
			}
			// We process as described above, but also include furtherZeroBits
			L += zeroBits
			z0 >>= zeroBits
			currentPos += zeroBits

			// if L > 64 - windowSize now, we break the loop
		}

		zCopy.ShiftRightEq(uint(L))
	}
	decomposition = decomposition[actualSize+1:]
	return
}
