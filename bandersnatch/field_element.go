// Note: Name is bsfield_element, because there is already a field_element in bls12381

package bandersnatch

import (
	"math/big"
)

// prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use BaseFieldSize of type big.Int instead
const BaseFieldSize_untyped = 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001

/*
	These are used as constants in the multiplication algorithm.
	Since there are no compile-time const-arrays in go, we need to define individual constants and manually
	unroll loops to make the compiler aware these are constants.
	(Or initialize a local array with these)
*/

// 64-bit sized words of the modulus
const (
	m_64_0 = (BaseFieldSize_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	m_64_1
	m_64_2
	m_64_3
)

// BaseFieldSize in little-endian uint64 array
var BaseFieldSize_64 [4]uint64 = [4]uint64{m_64_0, m_64_1, m_64_2, m_64_3}

// 32-bit sized words of the modulus
const (
	m_32_0 = (BaseFieldSize_untyped >> (iota * 32)) & 0xFFFFFFFF
	m_32_1
	m_32_2
	m_32_3
	m_32_4
	m_32_5
	m_32_6
	m_32_7
)

// BaseFieldSize in little-endian uint32 array
var BaseFieldSize_32 [8]uint32 = [8]uint32{m_32_0, m_32_1, m_32_2, m_32_3, m_32_4, m_32_5, m_32_6, m_32_7}

// bitlength of BaseFieldSize
const BaseFieldBitLength = 255

// number of bytes of BaseFieldSize == number of bytes needed to store individual field elements.
// We might actually use more bytes
const BaseFieldByteLength = (BaseFieldBitLength + 7) / 8

// Modulus of the Base field as a big-endian byte array (big.Int is easier with big-endian)
var BaseFieldSize_BigEndianBytes [BaseFieldByteLength]byte = func() (ret [BaseFieldByteLength]byte) {
	for i := 0; i < 8; i++ {
		ret[0*8+i] = byte((uint64(m_64_3) >> ((7 - i) * 8)) & 0xFF)
		ret[1*8+i] = byte((uint64(m_64_2) >> ((7 - i) * 8)) & 0xFF)
		ret[2*8+i] = byte((uint64(m_64_1) >> ((7 - i) * 8)) & 0xFF)
		ret[3*8+i] = byte((uint64(m_64_0) >> ((7 - i) * 8)) & 0xFF)
	}
	return
}()

var BaseFieldSize *big.Int = big.NewInt(0).SetBytes(BaseFieldSize_BigEndianBytes[:])

/*
	Trying out various implementations here for field elements of GF(BaseFieldSize)
	Notes: Internal representations are not guaranteed to be stable, may contain pointers or non-unique representations.
	In particular, neither assigment nor comparison operators are guaranteed to work as expected.
*/

/*
	This is the intended interface of Field Elements.
	Of course, this cannot be made an actual interface without possibly sacrificing efficiency
	since Go lacks generics:
	(The arguments to Mul etc. in the interface and the concret type need to match, so
	so the actual types' Mul(), Add() etc. implementation would need to accept an
	interface type and start by making a type assertion.)
type BSFieldElement_Interface interface {
	IsZero() bool
	IsOne() bool
	SetOne()
	SetZero()
	Mul(x, y *BSFieldElement_Interface)
	Add(x, y *BSFieldElement_Interface)
	Sub(x, y *BSFieldElement_Interface)
	Inv(x *BSFieldElement_Interface)
	SetInt(x *big.Int)
	ToInt() *big.Int
	Normalize()
	IsEqual(other *BSFieldElement_Interface) bool
}
*/

type FieldElement = bsFieldElement_64

var (
	FieldElementOne  = bsFieldElement_64_one
	FieldElementZero = bsFieldElement_64_zero

	// We do not expose FieldElementZero_alt, because users doing IsEqual(&FieldElementZero_alt, .) might call Normalize() on it, which would make
	// IsZero() subsequently fail.
	// FieldElementZero_alt = bsFieldElement_64_zero_alt

	FieldElementMinusOne = bsFieldElement_64_minusone
)
