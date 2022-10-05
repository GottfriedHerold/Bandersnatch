package fieldElements

import "math/bits"

type uint256 [4]uint64

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

func (z *uint256) Add_ReduceNonUnique(x, y *uint256) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)
	// carry == 1 basically only happens here if you do it on purpose (add up *lots* of non-normalized numbers).
	// NOTE: If carry == 1, then z.maybe_reduce_once() actually commutes with the -=mdoubled here: it won't do anything either before or after it.

	if carry != 0 {
		z[0], carry = bits.Sub64(z[0], baseFieldSizeDoubled_64_0, 0)
		z[1], carry = bits.Sub64(z[1], baseFieldSizeDoubled_64_1, carry)
		z[2], carry = bits.Sub64(z[2], baseFieldSizeDoubled_64_2, carry)
		z[3], _ = bits.Sub64(z[3], baseFieldSizeDoubled_64_3, carry)
	}

	if z[3] > baseFieldSize_3 {
		z[0], carry = bits.Sub64(z[0], baseFieldSize_0, 0)
		z[1], carry = bits.Sub64(z[1], baseFieldSize_1, carry)
		z[2], carry = bits.Sub64(z[2], baseFieldSize_2, carry)
		z[3], _ = bits.Sub64(z[3], baseFieldSize_3, carry) // _ is guaranteed to be 0
	}

	// z.maybe_reduce_once()
}

func (z *uint256) maybe_reduce_once() {
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
