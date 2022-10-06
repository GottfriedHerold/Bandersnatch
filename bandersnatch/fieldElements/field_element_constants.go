package fieldElements

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
)

// FieldElement is an element of the field of definition of the Bandersnatch curve.
//
// The size of this field matches (by design) the size of the prime-order subgroup of the BLS12-381 curve.
type FieldElement = bsFieldElement_64

// BaseFieldSize_untyped is the prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int.
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use BaseFieldSize_Int of type big.Int instead
const (
	BaseFieldSize_untyped = common.BaseFieldSize_untyped // == 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	BaseFieldSize_string  = common.BaseFieldSize_string  // == "0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

var BaseFieldSize_Int *big.Int = common.BaseFieldSize_Int

// BaseFieldSize_64 is the size of the field of definition of the Bandersnatch curve as little-endian uint64 array
var BaseFieldSize_64 [4]uint64 = [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}

// BaseFieldSize_32 is the size of the field of definition of the Bandersnatch curve as little-endian uint32 array
var BaseFieldSize_32 [8]uint32 = [8]uint32{baseFieldSize_32_0, baseFieldSize_32_1, baseFieldSize_32_2, baseFieldSize_32_3, baseFieldSize_32_4, baseFieldSize_32_5, baseFieldSize_32_6, baseFieldSize_32_7}

// BaseFieldBitLength is the bitlength of BaseFieldSize
const BaseFieldBitLength = common.BaseFieldBitLength // == 255

// BaseFieldByteLength is the length of BaseFieldSize in bytes.
const BaseFieldByteLength = common.BaseFieldByteLength // == 32

// An implementation of the base field might actually use more bytes; we don't.

// NOTE: We intentionally expose *copies* of unexported variables here to prevent users from modifying bsFieldElement_64_one etc.
// The issue is that these constants may be used internally in the actual implementation of the arithmetic; accidental modifications can then lead to errors that are very hard to debug.
// The other benefit is to give the compiler at least a chance to observe that these are never modified.
// For the above reason, internal code should not use the exported variables.

var (
	// Important constants of type FieldElement
	FieldElementOne      FieldElement = bsFieldElement_64_one
	FieldElementZero     FieldElement = bsFieldElement_64_zero
	FieldElementMinusOne FieldElement = bsFieldElement_64_minusone
	FieldElementTwo      FieldElement = InitFieldElementFromString("2")
)

/*
	These are used as constants in the multiplication algorithm.
	Since there are no compile-time const-arrays in go,
	we define individual constants and manually	unroll loops to make the compiler aware these are constants.
	(Or initialize a local array with these)
*/

// 64-bit sized words of the modulus. The index is the position of the word
const (
	baseFieldSize_0 = (BaseFieldSize_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	baseFieldSize_1
	baseFieldSize_2
	baseFieldSize_3
)

// 32-bit sized words of the modulus
const (
	baseFieldSize_32_0 = (BaseFieldSize_untyped >> (iota * 32)) & 0xFFFFFFFF
	baseFieldSize_32_1
	baseFieldSize_32_2
	baseFieldSize_32_3
	baseFieldSize_32_4
	baseFieldSize_32_5
	baseFieldSize_32_6
	baseFieldSize_32_7
)
