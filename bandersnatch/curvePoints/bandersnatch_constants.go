package curvePoints

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/fieldElements"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file collects the various constants that we use throughout the Bandersnatch implementations.
//
// This file is responsible for constant useds in the package for the curve points.
//
// Constants usually have a _suffix, which has some meaning; we often provide constant in several types.

// forward typedefs, to simplify things.

const ErrorPrefix = "bandersnatch / curve points: "

type FieldElement = fieldElements.FieldElement
type IsInputTrusted = common.IsInputTrusted

// These are COPIES and unexported by design.
var fieldElementZero = fieldElements.FieldElementZero
var fieldElementOne = fieldElements.FieldElementOne
var fieldElementMinusOne = fieldElements.FieldElementMinusOne
var fieldElementTwo = fieldElements.FieldElementTwo
var untrustedInput = common.UntrustedInput
var trustedInput = common.TrustedInput

const Cofactor = common.Cofactor
const CurveOrder = common.CurveOrder

var EndomorphismEigenvalue_Int = common.EndomorphismEigenvalue_Int // copies pointer
var GroupOrder_Int = common.GroupOrder_Int                         // copies pointer

// The affine y/z coordinate actually extends to the two points at infinity:
// While both y==z==0 at the points at infinity, we can use y/z == xy/xz == tz/xz == t/x to get a meaningful result
// that is neither 0 nor infinity.
//
// NOTE: Currently unused outside testing.
var yAtInfinity_E1 FieldElement
var yAtInfinity_E2 FieldElement

func init() {
	yAtInfinity_E1.Inv(&squareRootDbyA_fe)
	yAtInfinity_E2.Neg(&yAtInfinity_E1)
}

// BaseFieldSize_untyped is the prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int.
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use BaseFieldSize_Int of type big.Int instead
const (
	BaseFieldSize_untyped = 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	BaseFieldSize_string  = "0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

var BaseFieldSize_Int = utils.InitIntFromString(BaseFieldSize_string)

/*
	These are used as constants in the multiplication algorithm.
	Since there are no compile-time const-arrays in go (JUST WHY???),
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

// BaseFieldSize_64 is the size of the field of definition of the Bandersnatch curve as little-endian uint64 array
var BaseFieldSize_64 [4]uint64 = [4]uint64{baseFieldSize_0, baseFieldSize_1, baseFieldSize_2, baseFieldSize_3}

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

// BaseFieldSize_32 is the size of the field of definition of the Bandersnatch curve as little-endian uint32 array
var BaseFieldSize_32 [8]uint32 = [8]uint32{m_32_0, m_32_1, m_32_2, m_32_3, m_32_4, m_32_5, m_32_6, m_32_7}

// BaseFieldBitLength is the bitlength of BaseFieldSize
const BaseFieldBitLength = 255

// An implementation of the base field might actually use more bytes; we don't.

// BaseFieldByteLength is the number of bytes of BaseFieldSize == (mimimum) number of bytes needed to store individual field elements.
const BaseFieldByteLength = (BaseFieldBitLength + 7) / 8 // == 32

const GroupOrder = common.GroupOrder

// (p253-1)/2. We can represent Z/p253 by numbers from -halfGroupOrder, ... , + halfGroupOrder. This is used in the GLV decomposition algorithm.
const (
	halfGroupOrder        = (GroupOrder - 1) / 2
	halfGroupOrder_string = "6554484396890773809930967563523245729654577946720285125893201653364843836400"
)

var halfGroupOrder_Int = utils.InitIntFromString(halfGroupOrder_string)

// parameters a, d in twisted Edwards form ax^2 + y^2 = 1 + dx^2y^2

// Note: both a and d are non-squares

// CurveParameterA denotes the constant a in the twisted Edwards representation ax^2+y^2 = 1+dx^2y^2 of the Bandersnatch curve
const (
	CurveParameterA        = -5
	CurveParameterA_string = "-5"
)

// CurveParameterD denotes the constant d in the twisted Edwards representation ax^2+y^2 = 1+dx^2y^2 of the Bandersnatch curve.
// Note that d == -15 - 10\sqrt{2}
const (
	CurveParameterD        = 0x6389c12633c267cbc66e3bf86be3b6d8cb66677177e54f92b369f2f5188d58e7
	CurveParameterD_string = "0x6389c12633c267cbc66e3bf86be3b6d8cb66677177e54f92b369f2f5188d58e7"
)

// CurveParameters as *big.Int's or FieldElements
var (
	CurveParameterD_Int *big.Int     = utils.InitIntFromString(CurveParameterD_string)
	CurveParameterD_fe  FieldElement = fieldElements.InitFieldElementFromString[FieldElement](CurveParameterD_string)
	CurveParameterA_Int *big.Int     = utils.InitIntFromString(CurveParameterA_string)
	CurveParameterA_fe  FieldElement = fieldElements.InitFieldElementFromString[FieldElement](CurveParameterA_string)
)

// squareRootDByA is a square root of d/a. Due to the way the bandersnatch curve was constructed, we have (sqrt(d/a) + 1)^2 == 2.
// This number appears in coordinates of the order-2 points at inifinity and in the formulae for the endomorphism.
// Note that there are two square roots of d/a; be sure to make consistent choices.
const (
	squareRootDByA        = 37446463827641770816307242315180085052603635617490163568005256780843403514038
	squareRootDByA_string = "37446463827641770816307242315180085052603635617490163568005256780843403514038"
)

// const, really
var (
	// squareRootDbyA_Int *big.Int     = common.InitIntFromString(squareRootDByA_string) // TODO: Do we need this?
	squareRootDbyA_fe FieldElement = fieldElements.InitFieldElementFromString[FieldElement](squareRootDByA_string)
)

// These parameters appear in the formulae for the endomorphism.
const (
	// endo_a1              = 0x23c58c92306dbb95960f739827ac195334fcd8fa17df036c692f7ddaa306c7d4
	// endo_a1_string       = "0x23c58c92306dbb95960f739827ac195334fcd8fa17df036c692f7ddaa306c7d4"
	// endo_a2              = 0x23c58c92306dbb96b0b30d3513b222f50d02d8ff03e5036c69317ddaa306c7d4
	// endo_a2_string       = "0x23c58c92306dbb96b0b30d3513b222f50d02d8ff03e5036c69317ddaa306c7d4"
	endo_b               = 0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b4 // == sqrt(2) - 1 == sqrt(a/d)
	endo_b_string        = "0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b4"
	endo_binverse        = 0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b6 // =1/endo_b == endo_b + 2 == sqrt(d/a). Equals sqrtDByA
	endo_binverse_string = "0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b6"
	endo_bcd_string      = "36255886417209629651405037489028103282266637240540121152239675547668312569901" // == endo_b * endo_c * CurveParameterD
	endo_c               = 0x6cc624cf865457c3a97c6efd6c17d1078456abcfff36f4e9515c806cdf650b3d
	endo_c_string        = "0x6cc624cf865457c3a97c6efd6c17d1078456abcfff36f4e9515c806cdf650b3d"
	// endo_c1 == - endo_b
	//c1 = 0x2123b4c7a71956a2d149cacda650bd7d2516918bf263672811f0feb1e8daef4d
)

var (
	// endo_a1_fe       FieldElement = fieldElements.InitFieldElementFromString(endo_a1_string)
	// endo_a2_fe       FieldElement = fieldElements.InitFieldElementFromString(endo_a2_string)
	endo_b_fe        FieldElement = fieldElements.InitFieldElementFromString[FieldElement](endo_b_string)
	endo_c_fe        FieldElement = fieldElements.InitFieldElementFromString[FieldElement](endo_c_string)
	endo_binverse_fe FieldElement = fieldElements.InitFieldElementFromString[FieldElement](endo_binverse_string) // Note == SqrtDDivA_fe
	endo_bcd_fe      FieldElement = fieldElements.InitFieldElementFromString[FieldElement](endo_bcd_string)
)
