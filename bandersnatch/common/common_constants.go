package common

import "math/big"

// This file contains common constants related to the Bandersnatch curve such
// as the size of its defining field.
// Due to cyclic import issues (the package for field elements imports this file),
// constants of type FieldElement are not defined in this package.
//
// NOTE: Other packages typically define their own copies of these constants in order to shorten import paths.

// BaseFieldSize_untyped is the prime modulus (i.e. size) of the field of definition of Bandersnatch as untyped int.
// Due to overflowing all standard types, this is only useful in constant expressions.
// In most case, you want to use BaseFieldSize_Int of type big.Int instead
const (
	BaseFieldSize_untyped = 0x73eda753_299d7d48_3339d808_09a1d805_53bda402_fffe5bfe_ffffffff_00000001
	BaseFieldSize_string  = "0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"
)

var BaseFieldSize_Int = InitIntFromString(BaseFieldSize_string)

// BaseFieldBitLength is the bitlength of BaseFieldSize
const BaseFieldBitLength = 255

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
var GroupOrder_Int *big.Int = InitIntFromString(GroupOrder_string)

// Cofactor_Int is the cofactor of the Bandersnatch group as a *big.Int
var Cofactor_Int *big.Int = big.NewInt(Cofactor)

// CurveOrder_Int is the (non-prime) order of the group of rational points of the Bandersnatch curve as a *big.Int
var CurveOrder_Int *big.Int = new(big.Int).Mul(GroupOrder_Int, Cofactor_Int)

// CurveExponent_Int is the exponent of the group of rational points of the Bandersnatch curve as a *big.Int. This is 2*p253, where p253 is the size of the prime-order subgroup.
var CurveExponent_Int *big.Int = InitIntFromString(CurveExponent_string)

// EndomorphismEigenvalue is a number, such that the efficient degree-2 endomorphism acts as multiplication by this constant on the p253-subgroup.
// This is a square root of -2 modulo GroupOrder
const (
	EndomorphismEigenvalue        = 0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05
	EndomorphismEigenvalue_string = "0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05"
)

// EndomorphismEigenvalue_Int is a *big.Int, such that the the efficient degree-2 endomorphism of the Bandersnatch curve acts as multiplication by this constant on the p253-subgroup.
var EndomorphismEigenvalue_Int *big.Int = InitIntFromString(EndomorphismEigenvalue_string)

// TODO: Do we want to export this at all?

// InitIntFromString initializes a big.Int from a given string similar to InitFieldElementFromString.
// This internally uses big.Int's SetString and understands exactly those string formats.
// This implies that the given string can be decimal, hex, octal or binary, but needs to be prefixed if not decimal.
//
// This essentially is equivalent to big.Int's SetString method, except that it panics on error (which is appropriate for initialization globals from constant strings literal).
func InitIntFromString(input string) *big.Int {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	// Note: panic is the appropriate error handling here. Also, since this code is only run during package import, there is actually no way to catch it.
	if !success {
		panic("String used to initialized big.Int not recognized as a valid number")
	}
	return t
}

// TODO: Rename to InputTrusted?

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
