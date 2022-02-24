package bandersnatch

import "math/big"

// This file collects the various constants that we use throughout the Bandersnatch implementations.
//
// Note: We might want to move FieldElement (field of definition) and/or Exponents (scalar fields) into
// separate packages at some point. For that reason, the file is separated into "sections" according to that split.
//

// BaseFieldSize_untyped is the prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int.
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use BaseFieldSize of type big.Int instead
const (
	BaseFieldSize_untyped = 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	BaseFieldSize_string  = "0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

var BaseFieldSize_Int = initIntFromString(BaseFieldSize_string)

/*
	These are used as constants in the multiplication algorithm.
	Since there are no compile-time const-arrays in go, we need to define individual constants and manually
	unroll loops to make the compiler aware these are constants.
	(Or initialize a local array with these)
*/

// 64-bit sized words of the modulus
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

// Curve parameters

// GroupOrder is the order of the p253-subgroup of the Bandersnatch curve.
// This is a 253-bit prime number.
const (
	GroupOrder        = 0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1
	GroupOrder_string = "0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1"
)

// Cofactor is the cofactor of the Bandersnatch group, i.e. the size of the quotient of the group of rational curve points by the prime-order subgroup.
// The structure of this group is Z/2 x Z/2
const (
	Cofactor        = 4
	Cofactor_string = "4"
)

// CurveOrder denotes the non-prime size of the group of rational points of the Bandersnatch curve.
const (
	CurveOrder        = 52435875175126190479447740508185965837236623573762281007145613226918750691204 // = Cofactor * GroupOrder
	CurveOrder_string = "52435875175126190479447740508185965837236623573762281007145613226918750691204"
)

// CurveExponent is the exponent of the group of rational points of the Bandersnatch curve, i.e. we have CurveExponent*P = Neutral Element for all rational P.
// This is 2*GroupOrder rather than 4*GroupOrder, because the cofactor group has structure Z/2 x Z/2.
// When computing with (general) exponents, we work modulo this number.
const (
	CurveExponent        = 2 * GroupOrder
	CurveExponent_string = "26217937587563095239723870254092982918618311786881140503572806613459375345602"
)

// GroupOrder_Int is the order of the relevant prime order subgroup of the Bandersnatch curve as a *big.Int
var GroupOrder_Int *big.Int = initIntFromString(GroupOrder_string)

// Cofactor_Int is the cofactor of the Bandersnatch group as a *big.Int
var Cofactor_Int *big.Int = big.NewInt(Cofactor)

// CurveOrder_Int is the (non-prime) order of the group of rational points of the Bandersnatch curve as a *big.Int
var CurveOrder_Int *big.Int = new(big.Int).Mul(GroupOrder_Int, Cofactor_Int)

// CurveExponent_Int is the exponent of the group of rational points of the Bandersnatch curve as a *big.Int. This is 2*p253, where p253 is the size of the prime-order subgroup.
var CurveExponent_Int *big.Int = initIntFromString(CurveExponent_string)

// EndomorphismEigenvalue is a number, such that the efficient degree-2 endomorphism acts as multiplication by this constant on the p253-subgroup.
// This is a square root of -2 modulo GroupOrder
const (
	EndomorphismEivenvalue        = 0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05
	EndomorphismEigenvalue_string = "0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05"
)

const endomorphismEigenvalueIsOdd = true // we chose an odd representative above. This info is needed to get some test right.

// EndomorphismEigenvalue_Int is a *big.Int, such that the the efficient degree-2 endomorphism of the Bandersnatch curve acts as multiplication by this constant on the p253-subgroup.
var EndomorphismEigenvalue_Int *big.Int = initIntFromString(EndomorphismEigenvalue_string)

// (p253-1)/2. We can represent Z/p253 by numbers from -halfGroupOrder, ... , + halfGroupOrder. This is used in the GLV decomposition algorithm.
const (
	halfGroupOrder        = (GroupOrder - 1) / 2
	halfGroupOrder_string = "6554484396890773809930967563523245729654577946720285125893201653364843836400"
)

var halfGroupOrder_Int = initIntFromString(halfGroupOrder_string)

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
	CurveParameterD_Int *big.Int     = initIntFromString(CurveParameterD_string)
	CurveParameterD_fe  FieldElement = initFieldElementFromString(CurveParameterD_string)
	CurveParameterA_Int *big.Int     = initIntFromString(CurveParameterA_string)
	CurveParameterA_fe  FieldElement = initFieldElementFromString(CurveParameterA_string)
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
	// squareRootDbyA_Int *big.Int     = initIntFromString(squareRootDByA_string) // TODO: Do we need this?
	squareRootDbyA_fe FieldElement = initFieldElementFromString(squareRootDByA_string)
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
	// endo_a1_fe       FieldElement = initFieldElementFromString(endo_a1_string)
	// endo_a2_fe       FieldElement = initFieldElementFromString(endo_a2_string)
	endo_b_fe        FieldElement = initFieldElementFromString(endo_b_string)
	endo_c_fe        FieldElement = initFieldElementFromString(endo_c_string)
	endo_binverse_fe FieldElement = initFieldElementFromString(endo_binverse_string) // Note == SqrtDDivA_fe
	endo_bcd_fe      FieldElement = initFieldElementFromString(endo_bcd_string)
)

// utility constants

var (
	one_Int      = initIntFromString("1")
	two_Int      = initIntFromString("2")
	twoTo32_Int  = initIntFromString("0x1_00000000")
	twoTo64_Int  = initIntFromString("0x1_00000000_00000000")
	twoTo128_Int = initIntFromString("0x1_00000000_00000000_00000000_00000000")
	twoTo256_Int = initIntFromString("0x1_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")
	minusOne_Int = initIntFromString("-1")
)

// The point here is to force users to write Deserialize(..., TrustedInput, ...) rather than Deserialize(..., true, ...)
// in order to have better understandable semantics
// Golang does not have enum types, sadly, so we need to use structs: declaring a "type InPointTrusted bool" would cause Deserialze(..., true, ...)  to actually work due to implicit conversion.

// IsPointTrusted is a struct encapsulating a bool controlling whether some input is trusted or not.
// This is used to enforce better readable semantics in arguments.
// Users should use the predefined values TrustedInput and UntrustedInput of this type.
type IsPointTrusted struct {
	v bool
}

func (b IsPointTrusted) Bool() bool { return b.v }

// TrustedInput and UntrustedInput are used as arguments to Deserialization routines and in ToSubgroup.
var (
	TrustedInput   IsPointTrusted = IsPointTrusted{v: true}
	UntrustedInput IsPointTrusted = IsPointTrusted{v: false}
)
