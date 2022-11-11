package fieldElements

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
)

// This file contains all important constants used in the field element implementation.
// This includes exported constants and internal pre-computed constants.
//
// NOTES:
//   - We often defined multipe versions of a given constant that differ in type. This is mostly for convenience.
//     Our convention therefore is to suffix the constant with a tag for the type:
//        - _Int for big.Int
//        - _untyped for untyped constants
//        - _string for a string representation (either hex/decimal/binary; if non-decimal, it is prefixed. It may contain "_"-separators for readability.
//           We require it to be formatted in a way that [*big.Int]'s SetString method and [fmt]'s Scanning functions understand it.)
//        - _64, 32, 16, 8 for uint64/uint32/uint16/uint8 arrays
//   - Since Go lacks const arrays, we define some large 256-bit constants both as untyped 256-bit constants and separately as constants for every individual word.
//     The convention is that these are prefixed as constant_64_0, constant_64_1 for the (low-endian) 0th, 1st etc. uint64-word.
//
//     Note that the language does not let one do ANYTHING with >64-bit constants other than define other constants.

// NOTE:

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

// BaseFieldSize_Int is the prime modulus (i.e. size) of the field of definition of the Bandersnatch curve as a [*big.Int]
var BaseFieldSize_Int *big.Int = common.BaseFieldSize_Int

// baseFieldSize_Int is an internal unexported deep-copy of BaseFieldSize_int.
// This is not exported to prevent accidential modifications.
// Internal code should use this one for more robust debugging.
// NOTE: We probably won't internally use this anyway outside of testing, so this is not very imporant, actually.
var baseFieldSize_Int *big.Int = new(big.Int).Set(BaseFieldSize_Int)

// baseFieldSize_i_j are (typed) constants derived from BaseFieldSize_untyped at the end of this file.

// BaseFieldSize_64 is the size of the field of definition of the Bandersnatch curve as little-endian uint64 array
var BaseFieldSize_64 [4]uint64 = [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}

// BaseFieldSize_32 is the size of the field of definition of the Bandersnatch curve as little-endian uint32 array
var BaseFieldSize_32 [8]uint32 = [8]uint32{baseFieldSize_32_0, baseFieldSize_32_1, baseFieldSize_32_2, baseFieldSize_32_3, baseFieldSize_32_4, baseFieldSize_32_5, baseFieldSize_32_6, baseFieldSize_32_7}

// BaseFieldSize_16 is the size of the field of definition of the Bandersnatch curve as little-endian uint16 array
var BaseFieldSize_16 [16]uint16 = [16]uint16{
	baseFieldSize_16_0, baseFieldSize_16_1, baseFieldSize_16_2, baseFieldSize_16_3, baseFieldSize_16_4, baseFieldSize_16_5, baseFieldSize_16_6, baseFieldSize_16_7,
	baseFieldSize_16_8, baseFieldSize_16_9, baseFieldSize_16_10, baseFieldSize_16_11, baseFieldSize_16_12, baseFieldSize_16_13, baseFieldSize_16_14, baseFieldSize_16_15,
}

// BaseFieldSize_8 is the size of the field of definition of the Bandersnatch curve as little-endian uint8 array
var BaseFieldSize_8 [32]uint8 = [32]uint8{
	baseFieldSize_8_0, baseFieldSize_8_1, baseFieldSize_8_2, baseFieldSize_8_3, baseFieldSize_8_4, baseFieldSize_8_5, baseFieldSize_8_6, baseFieldSize_8_7,
	baseFieldSize_8_8, baseFieldSize_8_9, baseFieldSize_8_10, baseFieldSize_8_11, baseFieldSize_8_12, baseFieldSize_8_13, baseFieldSize_8_14, baseFieldSize_8_15,
	baseFieldSize_8_16, baseFieldSize_8_17, baseFieldSize_8_18, baseFieldSize_8_19, baseFieldSize_8_20, baseFieldSize_8_21, baseFieldSize_8_22, baseFieldSize_8_23,
	baseFieldSize_8_24, baseFieldSize_8_25, baseFieldSize_8_26, baseFieldSize_8_27, baseFieldSize_8_28, baseFieldSize_8_29, baseFieldSize_8_30, baseFieldSize_8_31,
}

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

// const r=MontgomeryMultiplier = 1<<256 // This is not defined explicitly, because
//  a) It is a 257-bit constant, which is not portable according to the Go spec.
//  b) Code relies deeply on this exact value. Changing it would break everything.

// r == 2^256 is the Montgomery multiplier.

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
	baseFieldSize_32_0 uint32 = (BaseFieldSize_untyped >> (iota * 32)) & 0xFFFFFFFF
	baseFieldSize_32_1
	baseFieldSize_32_2
	baseFieldSize_32_3
	baseFieldSize_32_4
	baseFieldSize_32_5
	baseFieldSize_32_6
	baseFieldSize_32_7
)

// 16-bit sized words of the modulus
const (
	baseFieldSize_16_0 uint16 = (BaseFieldSize_untyped >> (iota * 16)) & 0xFFFF
	baseFieldSize_16_1
	baseFieldSize_16_2
	baseFieldSize_16_3
	baseFieldSize_16_4
	baseFieldSize_16_5
	baseFieldSize_16_6
	baseFieldSize_16_7
	baseFieldSize_16_8
	baseFieldSize_16_9
	baseFieldSize_16_10
	baseFieldSize_16_11
	baseFieldSize_16_12
	baseFieldSize_16_13
	baseFieldSize_16_14
	baseFieldSize_16_15
)

// 8-bit sized bytes of the modulus
const (
	baseFieldSize_8_0 uint8 = (BaseFieldSize_untyped >> (iota * 8)) & 0xFF
	baseFieldSize_8_1
	baseFieldSize_8_2
	baseFieldSize_8_3
	baseFieldSize_8_4
	baseFieldSize_8_5
	baseFieldSize_8_6
	baseFieldSize_8_7
	baseFieldSize_8_8
	baseFieldSize_8_9
	baseFieldSize_8_10
	baseFieldSize_8_11
	baseFieldSize_8_12
	baseFieldSize_8_13
	baseFieldSize_8_14
	baseFieldSize_8_15
	baseFieldSize_8_16
	baseFieldSize_8_17
	baseFieldSize_8_18
	baseFieldSize_8_19
	baseFieldSize_8_20
	baseFieldSize_8_21
	baseFieldSize_8_22
	baseFieldSize_8_23
	baseFieldSize_8_24
	baseFieldSize_8_25
	baseFieldSize_8_26
	baseFieldSize_8_27
	baseFieldSize_8_28
	baseFieldSize_8_29
	baseFieldSize_8_30
	baseFieldSize_8_31
)
