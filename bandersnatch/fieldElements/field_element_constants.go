package fieldElements

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains all important constants used in the field element implementation.
// This includes exported constants and internal pre-computed constants.
//
// NOTES:
//   - We often defined multipe versions of a given constant that differ in type. This is mostly for convenience.
//     Our convention therefore is to suffix the constant with a tag for the type:
//        - _Int for big.Int
//        - _untyped for untyped constants -- Note that the language does not let one do ANYTHING with >64-bit constants other than define other constants.
//        - _string for a string representation (either hex/decimal/binary; if non-decimal, it is prefixed. It may contain "_"-separators for readability.
//           We require it to be formatted in a way that [*big.Int]'s SetString method and [fmt]'s Scanning functions understand it.)
//        - _64, 32, 16, 8 for low-endian uint64/uint32/uint16/uint8 arrays
//        - _uint256 for uint256. This is essentially the same as _64, since uint256 is based on [4]uint64 with low-endian convention.
//   - Since Go lacks const arrays, we define some large 256-bit constants both as untyped 256-bit constants and separately as constants for every individual word.
//     The convention is that these are prefixed as constant_64_0, constant_64_1 for the (low-endian) 0th, 1st etc. uint64-word.
//     Algorithms with unrolled loops (and unfortunately with unrolled loops only) can then make use of the individual constant words efficiently.
//     Unfortunately, Go is particularly ill-suited for these kind of optimizations.
//   - Since Go lacks constness for anything but simple types, "constant" arrays, structs etc. are defined as variables; these could theoretically be accidentially modified.
//     We intentionally expose *copies* of certain unexported variables here to prevent users from accidentially modifying the unexported ones.
//     The issue is that these constants may be used internally in the actual implementation of the arithmetic.
//     Accidental modifications can then lead to errors that are very hard to debug.
//     The other benefit is to give the compiler at least a chance to observe that these are never modified.
//     For the above reason, internal code should not use the exported variables.
//   - Some constants are copied from other packages. This is due to cross-package usage and to avoid dependency cycles.

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
//
// NOTE: We probably won't internally use this anyway outside of testing.
var baseFieldSize_Int *big.Int = new(big.Int).Set(BaseFieldSize_Int)

// baseFieldSize_uint256 is the size of the field of definition of the Bandersnatch curve as an uint256
var baseFieldSize_uint256 Uint256 = BaseFieldSize_64

// baseFieldSize_i_j are (typed) constants derived from BaseFieldSize_untyped at the end of this file. These give the j'th (low-endian) i-bit words.

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

// An implementation of the base field might actually use more bytes; we don't.

// BaseFieldBitLength is the bitlength of BaseFieldSize
const BaseFieldBitLength = common.BaseFieldBitLength // == 255

// BaseFieldByteLength is the length of BaseFieldSize in bytes.
const BaseFieldByteLength = common.BaseFieldByteLength // == 32

/***************************
 	uint 256 - related constants
*****************************/

// not writing 1<<256 - 1 due to portability (intermediate result has 257 bits)

// uint256Max_untyped is 2**256 - 1
const uint256Max_untyped = 2*((1<<255)-1) + 1 // == (1 << 256) - 1 == 0xFFFFFFFF_FFFFFFFF_FFFFFFFF_FFFFFFFF_FFFFFFFF_FFFFFFFF_FFFFFFFF_FFFFFFFF

var zero_uint256 Uint256 = [4]uint64{0, 0, 0, 0}
var one_uint256 Uint256 = [4]uint64{1, 0, 0, 0}
var two_uint256 Uint256 = [4]uint64{2, 0, 0, 0}
var uint256Max_uint256 Uint256 = [4]uint64{0xFFFFFFFF_FFFFFFFF, 0xFFFFFFFF_FFFFFFFF, 0xFFFFFFFF_FFFFFFFF, 0xFFFFFFFF_FFFFFFFF}

// Note: We copy pointers here. This is fine.
var (
	twoTo256_Int *big.Int = common.TwoTo256_Int // 2**256 == 115792089237316195423570985008687907853269984665640564039457584007913129639936
	twoTo512_Int *big.Int = common.TwoTo512_Int // 2**512
)

/*******
	Numbers derived from BaseFieldSize.
	These are relevant as constants inside our field element implementation(s)
********/

// const MontgomeryMultiplier = 1<<256 // This is not defined explicitly, because
//  a) It is a 257-bit constant, which is not portable according to the Go spec.
//  b) Code relies deeply on this exact value. Changing it would break anyway.

// FullMontgomeryMultiplier == 2^256 is the Montgomery multiplier.

// 2 * BaseFieldSize

const twiceBaseFieldSize_untyped = 2 * BaseFieldSize_untyped

var (
	twiceBaseFieldSize_Int     *big.Int  = utils.InitIntFromString("104871750350252380958895481016371931675381105001055275645207317399877162369026") // 2 * BaseFieldSize
	twiceBaseFieldSize_uint256 Uint256   = BigIntToUInt256(twiceBaseFieldSize_Int)
	twiceBaseFieldSize_64      [4]uint64 = [4]uint64{twiceBaseFieldSize_64_0, twiceBaseFieldSize_64_1, twiceBaseFieldSize_64_2, twiceBaseFieldSize_64_3}
)

// not portable, because >256 bits
// const thriceBaseFieldSize_untyped = 3 * BaseFieldSize_untyped // 157307625525378571438343221524557897513071657501582913467810976099815743553539
// const thriceBaseFieldSizeMod2To256_untyped = (3 * BaseFieldSize_untyped) % (1 << 256)

const thriceBaseFieldSizeMod2To256_untyped = 41515536288062376014772236515869989659801672835942349428353392091902613913603 // 3*BaseFieldSize - (1<<256)

var (
	thriceBaseFieldSize_Int              *big.Int  = utils.InitIntFromString("157307625525378571438343221524557897513071657501582913467810976099815743553539")
	thriceBaseFieldSizeMod2To256_Int     *big.Int  = utils.InitIntFromString("41515536288062376014772236515869989659801672835942349428353392091902613913603")
	thriceBaseFieldSizeMod2To256_uint256 Uint256   = BigIntToUInt256(thriceBaseFieldSizeMod2To256_Int)
	thriceBaseFieldSize_64               [5]uint64 = [5]uint64{thriceBaseFieldSize_64_0, thriceBaseFieldSize_64_1, thriceBaseFieldSize_64_2, thriceBaseFieldSize_64_3, thriceBaseFieldSize_64_4}
)

// The weird computation here is to avoid 1 << 256, which is not portable according to the go spec (intermediate results are too large even for untyped computations)

// twoTo256ModBaseField_untyped is 2^256 mod BaseFieldSize. This is also the Montgomery representation of 1 (for the canonical Montgomery multiplier 2^256).
const twoTo256ModBaseField_untyped = 2 * ((1 << 255) - BaseFieldSize_untyped) // 0x1824b159acc5056f_998c4fefecbc4ff5_5884b7fa00034802_00000001fffffffe == 10920338887063814464675503992315976177888879664585288394250266608035967270910

var (
	twoTo256ModBaseField_Int     *big.Int = utils.InitIntFromString("10920338887063814464675503992315976177888879664585288394250266608035967270910")
	twoTo256ModBaseField_uint256 Uint256  = Uint256{twoTo256ModBaseField_64_0, twoTo256ModBaseField_64_1, twoTo256ModBaseField_64_2, twoTo256ModBaseField_64_3}
)

// twoTo512ModBaseField_untyped is 2^512 mod BaseFieldSize. This is useful for converting to/from montgomery form.
const (
	twoTo512ModBaseField_untyped = 0x748d9d99f59ff1105d314967254398f2b6cedcb87925c23c999e990f3f29c6d
)

var (
	twoTo512ModBaseField_Int     *big.Int = utils.InitIntFromString("0x748d9d99f59ff1105d314967254398f2b6cedcb87925c23c999e990f3f29c6d")
	twoTo512ModBaseField_uint256 Uint256  = Uint256{twoTo512ModBaseField_64_0, twoTo512ModBaseField_64_1, twoTo512ModBaseField_64_2, twoTo512ModBaseField_64_3}
)

// We do not neccessarily assume that all representatives of field elements are fully reduced. In the current bsFieldElement_64 implementation, we assume that
// all representations are (supposed to be) in the range [0, montgomeryRepBound) in order for our algorithms to work correctly.

var (
	montgomeryBound_Int     *big.Int = utils.InitIntFromString("63356214062190004944123244500501942015579432165112926216853925307974548455423") // 2**256 - BaseFieldSize
	montgomeryBound_uint256 Uint256  = BigIntToUInt256(montgomeryBound_Int)
)

// minusOneHalfModBaseField_untyped equals 1/2 * (BaseFieldSize-1) as untyped int. This equals -1/2 mod BaseFieldSize
const minusOneHalfModBaseField_untyped = (BaseFieldSize_untyped - 1) / 2 // 26217937587563095239723870254092982918845276250263818911301829349969290592256
var (
	minusOneHalfModBaseField_Int     *big.Int = utils.InitIntFromString("26217937587563095239723870254092982918845276250263818911301829349969290592256")
	minusOneHalfModBaseField_uint256 Uint256  = Uint256{minusOneHalfModBaseField_64_0, minusOneHalfModBaseField_64_1, minusOneHalfModBaseField_64_2, minusOneHalfModBaseField_64_3}
)

// oneHalfModBaseField_untyped equals 1/2 * (BaseFieldSize+1) as untyped int. This equals +1/2 mod BaseFieldSize
const oneHalfModBaseField_untyped = (BaseFieldSize_untyped + 1) / 2 // 26217937587563095239723870254092982918845276250263818911301829349969290592257, one more than the above

var (
	oneHalfModBaseField_Int     *big.Int = utils.InitIntFromString("26217937587563095239723870254092982918845276250263818911301829349969290592257")
	oneHalfModBaseField_uint256          = Uint256{oneHalfModBaseField_64_0, oneHalfModBaseField_64_1, oneHalfModBaseField_64_2, oneHalfModBaseField_64_3}
)

/*
var oneHalfModBaseField_Int *big.Int = func() (ret *big.Int) {
	ret = big.NewInt(1)
	var twoInt *big.Int = big.NewInt(2)
	ret.Add(ret, baseFieldSize_Int)
	ret.Div(ret, twoInt)
	return
}()
*/

// const oneHalfModBaseField_uint256 Uint256 = Uint256{oneH}

// minus2To256ModBaseField_untyped is -(2**256) modulo BaseFieldSize.
// This is the Montgomery representation of -1.
const minus2To256ModBaseField_untyped = BaseFieldSize_untyped - twoTo256ModBaseField_untyped // 41515536288062376014772236515869989659801672835942349428353392091902613913603

var (
	minus2To256ModBaseField_uint256 = Uint256{minus2To256ModBaseField_64_0, minus2To256ModBaseField_64_1, minus2To256ModBaseField_64_2, minus2To256ModBaseField_64_3}
)

// neativeInverseModulus_uint64 is -1/BaseFieldSize mod 2**64.
// This constant is used during montgomery multiplication.
const negativeInverseModulus_uint64 = 18446744069414584319 // == (0xFFFFFFFF_FFFFFFFF * 0x00000001_00000001) % (1<<64)

/****************************
	Exported Field Elements
****************************/

// NOTE2: The FieldElementOne etc. here are explicitly types as FieldElement, which is a type alias to the actually used field element implementation.
// So if we change FieldElement to alias a different type, these definitions will fail and need to be changed. This is intentional.

// FieldElement is an element of the field of definition of the Bandersnatch curve.
//
// The size of this field matches (by design) the size of the prime-order subgroup of the BLS12-381 curve.
type FieldElement = bsFieldElement_MontgomeryNonUnique

var (
	// Important constants of type FieldElement
	FieldElementOne      FieldElement = bsFieldElement_64_one
	FieldElementZero     FieldElement = bsFieldElement_64_zero
	FieldElementMinusOne FieldElement = bsFieldElement_64_minusone
	FieldElementTwo      FieldElement = InitFieldElementFromString("2")
)

/***************************
	Individual words of untyped constants

	These are used as constants in the algorithm.
	Since there are no compile-time const-arrays in go,
	we define individual constants and manually	unroll loops to make the compiler aware these are constants.
	Or we initialize a arrays using these constants.
************************/

// baseFieldSizeDoubled_64_i denotes the i'th 64-bit word of 2 * BaseFieldSize
const (
	twiceBaseFieldSize_64_0 = (twiceBaseFieldSize_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	twiceBaseFieldSize_64_1
	twiceBaseFieldSize_64_2
	twiceBaseFieldSize_64_3
)

// mhalved_64_i denotes the i'th 64-bit word of 1/2 * (BaseFieldSize-1)
const (
	minusOneHalfModBaseField_64_0 = (minusOneHalfModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	minusOneHalfModBaseField_64_1
	minusOneHalfModBaseField_64_2
	minusOneHalfModBaseField_64_3
)

// rsquared_64_i is the i'th 64-bit word of 2^512 mod BaseFieldSize.
const (
	twoTo512ModBaseField_64_0 = (twoTo512ModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	twoTo512ModBaseField_64_1
	twoTo512ModBaseField_64_2
	twoTo512ModBaseField_64_3
)

// rModBaseField_64_i is the i'th 64-bit word of (2^256 mod BaseFieldSize). Note that this corresponds to the Montgomery representation of 1.
const (
	twoTo256ModBaseField_64_0 uint64 = (twoTo256ModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	twoTo256ModBaseField_64_1
	twoTo256ModBaseField_64_2
	twoTo256ModBaseField_64_3
)

// montgomeryNegOne_i is the i'th 64-bit word of the negative of rModBaseField modulo BaseFieldSize.
// This is the Montgomery representation of -1.
const (
	minus2To256ModBaseField_64_0 uint64 = (minus2To256ModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	minus2To256ModBaseField_64_1
	minus2To256ModBaseField_64_2
	minus2To256ModBaseField_64_3
)

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

const (
	thriceBaseFieldSize_64_0 = (thriceBaseFieldSizeMod2To256_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	thriceBaseFieldSize_64_1
	thriceBaseFieldSize_64_2
	thriceBaseFieldSize_64_3
	thriceBaseFieldSize_64_4 = 1
)

const (
	oneHalfModBaseField_64_0 = (oneHalfModBaseField_untyped >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	oneHalfModBaseField_64_1
	oneHalfModBaseField_64_2
	oneHalfModBaseField_64_3
)
