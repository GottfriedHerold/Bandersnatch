// common is the subpackage of the bandersnatch module that collects some basic utility routines that are used by both the field element and the curve point parts of the module.
//
// Note that most of the exported symbols are redefined in multiple packages as aliases.
//
// TODO: This package is quite unstable, as we might move things around / split this into sub-packages.
// E.g. we would want the FieldElementEndianness routines to work with our uint256 data type rather than [4]uint64.
// However, this involves exporting uint256 and reoganizing/splitting packages to avoid dependency cycles.
//
// TODO: Explain with more examples in detail once this package is finalized.
package common

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains common constants related to the Bandersnatch curve such
// as the size of its defining field.
// Due to cyclic import issues (namely, the package for field elements imports this file),
// constants of type FieldElement related to the Bandersnatch curve are not defined in this package.
//
// NOTE: Other packages typically define their own copies of these constants in order to shorten import paths.

// BaseFieldSize is the prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int.
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use [BaseFieldSize_Int] of type [*big.Int] instead
const (
	BaseFieldSize         = 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	BaseFieldSize_untyped = BaseFieldSize // the _untyped is just for emphasis and helps with internal code documentation.
	BaseFieldSize_string  = "0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

// BaseFieldSize_Int is the prime modulus (i.e. size) of the field of definition of Bandersnatch as a [*big.Int].
var BaseFieldSize_Int = utils.InitIntFromString(BaseFieldSize_string) // 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001

// BaseFieldBitLength is the bitlength of [BaseFieldSize]
const BaseFieldBitLength = 255

// BaseFieldByteLength is the number of bytes of [BaseFieldSize] == (mimimum) number of bytes needed to store individual field elements.
const BaseFieldByteLength = (BaseFieldBitLength + 7) / 8 // == 32

// Curve parameters

// GroupOrder is the order of the p253-subgroup of the Bandersnatch curve.
// This is a 253-bit prime number. We also provide this as [GroupOrder_Int] of type [*big.Int]
const (
	GroupOrder        = 0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1
	GroupOrder_string = "0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1"
)

// GroupOrder_Int is the order of the relevant prime order subgroup of the Bandersnatch curve as a [*big.Int]
var GroupOrder_Int *big.Int = utils.InitIntFromString(GroupOrder_string)

// Cofactor is the cofactor of the Bandersnatch group, i.e. the size of the quotient of the group of rational curve points by the prime-order subgroup.
// The structure of this group is Z/2 x Z/2. As with all constants, it is also provided as [Cofactor_Int] of type [*big.Int].
const (
	Cofactor        = 4
	Cofactor_string = "4"
)

// Cofactor_Int is the cofactor of the Bandersnatch group as a [*big.Int]
var Cofactor_Int *big.Int = big.NewInt(Cofactor) // value: 4

// CurveOrder denotes the non-prime size of the group of rational points of the Bandersnatch curve.
// As with all constant, we also export a [CurveOrder_Int] of type [*big.Int]
const (
	CurveOrder        = 52435875175126190479447740508185965837236623573762281007145613226918750691204 // = Cofactor * GroupOrder
	CurveOrder_string = "52435875175126190479447740508185965837236623573762281007145613226918750691204"
)

// CurveOrder_Int is the (non-prime) order of the group of rational points of the Bandersnatch curve as a [*big.Int]
var CurveOrder_Int *big.Int = new(big.Int).Mul(GroupOrder_Int, Cofactor_Int) // decimal value is 52435875175126190479447740508185965837236623573762281007145613226918750691204

// CurveExponent is the exponent of the group of rational points of the Bandersnatch curve, i.e. we have CurveExponent*P = Neutral Element for all rational curve points P.
//
// This value is 2 * [GroupOrder] rather than 4 * [GroupOrder], because the cofactor group has structure Z/2 x Z/2 rather than Z/4.
// When computing expressions like x*P for points P that might not be in the prime-order subgroup, we work modulo this number for x.
//
// This is also provided as [CurveExponent_Int] of type [*big.Int]
//
// Note: While we generally use additive notation for the group, this constant is still called exponent, because it matches the general definition of the exponent of a group, often written exp(G) for arbitrary groups G.
const (
	CurveExponent        = 2 * GroupOrder
	CurveExponent_string = "26217937587563095239723870254092982918618311786881140503572806613459375345602"
)

// CurveExponent_Int is the exponent of the group of rational points of the Bandersnatch curve as a [*big.Int]. This is is equal 2*p253, where p253 (exported as [GroupOrder]) is the size of the prime-order subgroup.
var CurveExponent_Int *big.Int = utils.InitIntFromString(CurveExponent_string) // value: 26217937587563095239723870254092982918618311786881140503572806613459375345602

// TODO: Link to paper

// EndomorphismEigenvalue is a number, such that the efficient degree-2 endomorphism acts as multiplication by this constant on the prime order subgroup.
// This is a square root of -2 modulo [GroupOrder]
//
// NOTE: There are actually two efficient degree-2 endomorphisms, which are dual to each other. The other one acts by -EndomorphismEigenvalue (i.e. the other square root of -2).
// The choice here is compatible with the methods provided for computing the endomorphism in the [curvePoints] package and with the explicit formula given in the bandersnatch paper.
const (
	EndomorphismEigenvalue        = 0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05
	EndomorphismEigenvalue_string = "0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05"
)

// EndomorphismEigenvalue_Int is a *big.Int, such that the the efficient degree-2 endomorphism of the Bandersnatch curve acts as multiplication by this constant on the p253-subgroup.
// This is a square root of -2 modulo GroupOrder
//
// NOTE: There are actually two efficient degree-2 endomorphisms, which are dual to each other.
// The other one acts on the prime-order subgroup by multiplications with -EndomorphismEigenvalue (i.e. the other square root of -2).
// The choice here is compatible with the methods provided for computing the endomorphism in the [curvePoints] package and with the explicit formula given in the bandersnatch paper.
var EndomorphismEigenvalue_Int *big.Int = utils.InitIntFromString(EndomorphismEigenvalue_string)

// The point here is to force users to write Deserialize(..., TrustedInput, ...) rather than Deserialize(..., true, ...)
// in order to have better understandable semantics
// Golang does not have enum types, sadly, so we need to use structs: declaring a "type InPointTrusted bool" would cause Deserialize(..., true, ...)  to actually work due to implicit conversion.

// TODO: Refer to actual example with correct deserialization names

// IsInputTrusted is a struct encapsulating a bool controlling whether some input is trusted or not.
// Users should use the predefined values TrustedInput and UntrustedInput of this type.
//
// Deserialization routines and methods where we need to know/ensure whether some curve point is in the prime-order subgroup take an input argument of this type.
// If we know that we can trust the input, the library can omit certain (in some cases very expensive) checks.
//
// We prefer to use a separate type rather than bool for this to force users to write foo.Deserialize(bar, TrustedInput, bar2) rather than foo.Deserialize(bar, true, bar2) in order to have self-documenting syntax.
type IsInputTrusted struct {
	v bool
}

// Bool turns a trust level of type [IsInputTrusted] into a bool, with true indicating that the input is trusted.
func (b IsInputTrusted) Bool() bool { return b.v }

// TrustedInput and UntrustedInput are used as arguments to Deserialization routines and in ToSubgroup.
var (
	TrustedInput   IsInputTrusted = IsInputTrusted{v: true}
	UntrustedInput IsInputTrusted = IsInputTrusted{v: false}
)

// TrustLevelFromBool wraps a bool into an [IsInputTrusted] with true indicating that the input is trustworthy.
func TrustLevelFromBool(v bool) IsInputTrusted {
	return IsInputTrusted{v: v}
}

// utility constants

/*
var (
	One_Int      = InitIntFromString("1")
	Two_Int      = InitIntFromString("2")
	TwoTo32_Int  = InitIntFromString("0x1_00000000")
	TwoTo64_Int  = InitIntFromString("0x1_00000000_00000000")
	TwoTo128_Int = InitIntFromString("0x1_00000000_00000000_00000000_00000000")
	TwoTo256_Int = InitIntFromString("0x1_00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000")
	MinusOne_Int = InitIntFromString("-1")
)

*/
