package fieldElements

import (
	"encoding/binary"
	"math/big"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type uint256 [4]uint64 // low-endian
type uint512 [8]uint64 // low-endian

// ToBigInt converts the given uint256 to a [*big.Int]
func (z *uint256) ToBigInt() *big.Int {
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
func (z *uint512) ToBigInt() *big.Int {
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

// TODO: Move BigIntToUIntArray into this package.
// (Currently not done this way because of overlapping use in the exponents package -- this will change)

// BigIntToUInt256 converts a [*big.Int] to a uint256.
//
// This function panics if x is not between 0 and 2^256 - 1.
func BigIntToUInt256(x *big.Int) (result uint256) {
	return utils.BigIntToUIntArray(x)
}

func BigIntToUint512(x *big.Int) (result uint512) {
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

func (z *uint256) FromBigInt(x *big.Int) {
	*z = utils.BigIntToUIntArray(x)
}

func (z *uint512) FromBigInt(x *big.Int) {
	*z = BigIntToUint512(x)
}

/*
// BigIntToUIntArray converts a big.Int to a low-endian [4]uint64 array without Montgomery conversions.
// We assume 0 <= x < 2^256
func BigIntToUIntArray(x *big.Int) (result [4]uint64) {
	// As this is an internal function, panic is OK for error handling.
	if x.Sign() < 0 {
		panic(ErrorPrefix + "bigIntToUIntArray: Trying to convert negative big Int")
	}
	if x.BitLen() > 256 {
		panic(ErrorPrefix + "bigIntToUIntArray: big Int too large to fit into 32 bytes.")
	}
	var big_endian_byte_slice [32]byte
	x.FillBytes(big_endian_byte_slice[:])
	result[0] = binary.BigEndian.Uint64(big_endian_byte_slice[24:32])
	result[1] = binary.BigEndian.Uint64(big_endian_byte_slice[16:24])
	result[2] = binary.BigEndian.Uint64(big_endian_byte_slice[8:16])
	result[3] = binary.BigEndian.Uint64(big_endian_byte_slice[0:8])
	return
}
*/

// Add computes an addition z := x + y.
// The addition is carried out modulo 2^256
func (z *uint256) Add(x, y *uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], _ = bits.Add64(x[3], y[3], carry)
}

// AddWithCarry computes an addition z := x + y.
// The addition is carried out modulo 2^256
// Returns the carry value
func (z *uint256) AddWithCarry(x, y *uint256) (carry uint64) {
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)
	return
}

// Sub computes subtraction z := x - y, modulo 2^256
func (z *uint256) Sub(x, y *uint256) {
	var borrow uint64 // only takes values 0,1
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], _ = bits.Sub64(x[3], y[3], borrow)
}

// SubWithBorrow computes subtraction z := x - y, modulo 2^256
// returns borrow bit
func (z *uint256) SubWithBorrow(x, y *uint256) (borrow uint64) {
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)
	return
}

// IsZero checks whether the uint256 is (exactly) zero.
func (z *uint256) IsZero() bool {
	return z[0]|z[1]|z[2]|z[3] == 0

}

// NOTE: Kept in uint256.go, because it is essentially a 256-bit -> 512-bit multiplication.
// We will separate those parts anyway (i.e. make this a method of uint512 and have the field element class call the reduction).

// Computes z*=x (mod m) weakly reduced to the interval [0..2**256)
// input values don't need to be fully reduced.
func (z *uint256) MulEqAndReduce_a(x *uint256) {
	var c, t0, t1, q0, q1, q2, q3, q4, q5, q6, q7 uint64

	q2, q1 = bits.Mul64(z[0], x[1])
	q4, q3 = bits.Mul64(z[0], x[3])

	t1, q0 = bits.Mul64(z[0], x[0])
	q1, c = bits.Add64(q1, t1, 0)
	t1, t0 = bits.Mul64(z[0], x[2])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	q4, _ = bits.Add64(q4, 0, c)

	t1, t0 = bits.Mul64(z[1], x[1])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	q5, t0 = bits.Mul64(z[1], x[3])
	q4, c = bits.Add64(q4, t0, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(z[1], x[0])
	q1, c = bits.Add64(q1, t0, 0)
	q2, c = bits.Add64(q2, t1, c)
	t1, t0 = bits.Mul64(z[1], x[2])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, t0 = bits.Mul64(z[2], x[1])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	q6, t0 = bits.Mul64(z[2], x[3])
	q5, c = bits.Add64(q5, t0, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(z[2], x[0])
	q2, c = bits.Add64(q2, t0, 0)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(z[2], x[2])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	q6, _ = bits.Add64(q6, 0, c)

	t1, t0 = bits.Mul64(z[3], x[1])
	q4, c = bits.Add64(q4, t0, 0)
	q5, c = bits.Add64(q5, t1, c)
	q7, t0 = bits.Mul64(z[3], x[3])
	q6, c = bits.Add64(q6, t0, c)
	q7, _ = bits.Add64(q7, 0, c)

	t1, t0 = bits.Mul64(z[3], x[0])
	q3, c = bits.Add64(q3, t0, 0)
	q4, c = bits.Add64(q4, t1, c)
	t1, t0 = bits.Mul64(z[3], x[2])
	q5, c = bits.Add64(q5, t0, c)
	q6, c = bits.Add64(q6, t1, c)
	q7, _ = bits.Add64(q7, 0, c)

	// Reduce back into uint256
	z.ReduceUint512ToUint256_a(uint512{q0, q1, q2, q3, q4, q5, q6, q7})

}

// Computes z = x * x (mod m) weakly reduced reduce to to the interval [0..2**256)
// input values don't need to be fully reduced.
func (z *uint256) SquareEqAndReduce_a() {
	var c, t0, t1, q0, q1, q2, q3, q4, q5, q6, q7 uint64

	q4, q3 = bits.Mul64(z[0], z[3])

	t1, q2 = bits.Mul64(z[0], z[2])
	q3, c = bits.Add64(q3, t1, 0)
	q5, t0 = bits.Mul64(z[1], z[3])
	q4, c = bits.Add64(q4, t0, c)
	q5, _ = bits.Add64(q5, 0, c)

	t1, q1 = bits.Mul64(z[0], z[1])
	q2, c = bits.Add64(q2, t1, 0)
	t1, t0 = bits.Mul64(z[1], z[2])
	q3, c = bits.Add64(q3, t0, c)
	q4, c = bits.Add64(q4, t1, c)
	q6, t0 = bits.Mul64(z[2], z[3])
	q5, c = bits.Add64(q5, t0, c)
	q6, _ = bits.Add64(q6, 0, c)

	q1, c = bits.Add64(q1, q1, 0)
	q2, c = bits.Add64(q2, q2, c)
	q3, c = bits.Add64(q3, q3, c)
	q4, c = bits.Add64(q4, q4, c)
	q5, c = bits.Add64(q5, q5, c)
	q6, c = bits.Add64(q6, q6, c)
	q7, _ = bits.Add64(0, 0, c)

	t1, q0 = bits.Mul64(z[0], z[0])
	q1, c = bits.Add64(q1, t1, 0)
	t1, t0 = bits.Mul64(z[1], z[1])
	q2, c = bits.Add64(q2, t0, c)
	q3, c = bits.Add64(q3, t1, c)
	t1, t0 = bits.Mul64(z[2], z[2])
	q4, c = bits.Add64(q4, t0, c)
	q5, c = bits.Add64(q5, t1, c)
	t1, t0 = bits.Mul64(z[3], z[3])
	q6, c = bits.Add64(q6, t0, c)
	q7, _ = bits.Add64(q7, t1, c)

	// Reduce back into uint256
	z.ReduceUint512ToUint256_a(uint512{q0, q1, q2, q3, q4, q5, q6, q7})
}

// LongMul computes a 256 bits -> 512 multiplication, without any modular reduction. z:=x*y
func (z *uint512) LongMul(x, y *uint256) {
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
func (z *uint512) LongSquare(x *uint256) {
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
