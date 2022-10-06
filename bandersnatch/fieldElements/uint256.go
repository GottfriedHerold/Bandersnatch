package fieldElements

import (
	"encoding/binary"
	"math/big"
	"math/bits"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type uint256 [4]uint64 // low-endian

func (z *uint256) ToBigInt() *big.Int {
	var big_endian_byte_slice [32]byte
	binary.BigEndian.PutUint64(big_endian_byte_slice[0:8], z[3])
	binary.BigEndian.PutUint64(big_endian_byte_slice[8:16], z[2])
	binary.BigEndian.PutUint64(big_endian_byte_slice[16:24], z[1])
	binary.BigEndian.PutUint64(big_endian_byte_slice[24:32], z[0])
	return new(big.Int).SetBytes(big_endian_byte_slice[:])
}

func BigIntToUInt256(x *big.Int) (result uint256) {
	return utils.BigIntToUIntArray(x)
}

// Add computes an addition z := x + y.
// The addition is carried out modulo 2^256
func (z *uint256) Add(x, y *uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], _ = bits.Add64(x[3], y[3], carry)
}

func (z *uint256) AddWithCarry(x, y *uint256) (carry uint64) {
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)
	return
}

func (z *uint256) Sub(x, y *uint256) {
	var borrow uint64 // only takes values 0,1
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], _ = bits.Sub64(x[3], y[3], borrow)
}

func (z *uint256) SubWithBorrow(x, y *uint256) (borrow uint64) {
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)
	return
}

// AddAndReduce_Weak sets z, such that z==x+y mod BaseFieldSize holds. Assumes x and y to be weakly reduced.
//
// z might not be the smallest such representations. More precisely:
//  - If x and y are both in [0, UINT256MAX-BaseFieldSize), then so is z.
//  - If x and y are both in [0, 2*BaseFieldSize), then so is z.
func (z *uint256) AddAndReduce_Weak(x, y *uint256) {
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

// SubAndReduce_Weak1 sets z, such that z==x-y mod BaseFieldSize holds.
// Assumes x and y to be weakly reduced in [0, UINT256MAX-BaseFieldSize).
//
// It guarantees the same for z.
func (z *uint256) SubAndReduce_Weak1(x, y *uint256) {
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
			z[3], _ = bits.Add64(z[3], baseFieldSize_3, borrow) // _ is one
		} else {
			// Due to constraints to y, adding 2*BaseFieldSize is guaranteed to create an overflow, so we end up with some z' == x-y mod BaseFieldSize.
			// Note that the result is in the correct range.
			// This is because the only case where we choose +=2*BaseFieldSize even though += 1* BaseFieldSize would suffice is
			// when z[3] == 0xFFFFFFFF_FFFFFFFF-baseFieldSize_3 in the condition above (and we would need to look at the other words to decide)
			// In this case, after adding +=2*BaseFieldSize, the resulting z' has
			// z'[3] == baseFieldSize_3 or z'[3]== baseFieldSize_3+1. Both guaranteed z in [0, UINT256MAX-BaseFieldSize)
			z[0], borrow = bits.Add64(z[0], baseFieldSizeDoubled_64_0, 0)
			z[1], borrow = bits.Add64(z[1], baseFieldSizeDoubled_64_1, borrow)
			z[2], borrow = bits.Add64(z[2], baseFieldSizeDoubled_64_2, borrow)
			z[3], _ = bits.Add64(z[3], baseFieldSizeDoubled_64_3, borrow) // _ is one
		}
	}
}

// SubAndReduce_Weak1 sets z, such that z==x-y mod BaseFieldSize holds.
// Assumes x and y to be weakly reduced in [0, 2*BaseFieldSize).
//
// It guarantees the same for z.
func (z *uint256) SubAndReduce_Weak2(x, y *uint256) {
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

// IsZero checks whether the uint256 is (exactly) zero.
func (z *uint256) IsZero() bool {
	return z[0]|z[1]|z[2]|z[3] == 0
}

// reduce_weakly replaces z by some number z' with z' == z mod BaseFieldSize.
//
// z' is guaranteed to be in [0, UINT256MAX-BaseFieldSize)
func (z *uint256) reduce_weakly() {
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

// reduce_weak_to_full replaces z by some number z' with z' == z mod BaseFieldSize. Assumes z is already weakly reduced.
//
// More precisely, z' is guaranteed to be in [0, BaseFieldSize), provided z is in [0, 2*BaseFieldSize)
func (z *uint256) reduce_weak_to_full() {
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
