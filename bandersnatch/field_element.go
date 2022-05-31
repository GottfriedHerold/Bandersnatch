package bandersnatch

// Code for FieldElement (meaning the field of definition of the bandersnatch curve)
// is in field_element_64.go and field_element_8.go
// Only field_element_64.go is used; field_element_8.go serves as a compariso
// implementation that is only used in testing.

// NOTE: The _8 comparison implementation does not have everything implemented.

// TODO: Interface here is not complete. Refer to field_element_64.go for what we actually provide.

/*
	This is the intended interface of Field Elements.
	Of course, this cannot be made an actual interface without possibly sacrificing efficiency
	since Go lacks generics:
	(The arguments to Mul etc. in the interface and the concret type need to match, so
	so the actual types' Mul(), Add() etc. implementation would need to accept an
	interface type and start by making a type assertion.)

	As as Go1.18, we have generics, but this does not change things :(
	(without making unacceptable sacrifices in efficiency, at least)

type BSFieldElement_Interface interface {
	IsZero() bool
	IsOne() bool
	SetOne()
	SetZero()
	Mul(x, y *BSFieldElement_Interface)
	Add(x, y *BSFieldElement_Interface)
	Sub(x, y *BSFieldElement_Interface)
	Square(x *BSFieldElement_Interface)
	Neg(x *BSFieldElement_Interface)
	Inv(x *BSFieldElement_Interface)
	Divide(x, y *BSFieldElement_Interface)
	ToBigInt() *big.Int
	ToUInt64() (uint64, err )
	SetBigInt(x *big.Int)
	Normalize()
	IsEqual(other *BSFieldElement_Interface) bool
	Sign() int
	Jacobi() int
	AddEq(y *BSFieldElement_Interface)
	SubEq(y *BSFieldElement_Interface)
	SquareEq()
	DivideEq(y *BSFieldElement_Interface)
	NegEq()

}
*/

// FieldElement is an element of the field of definition of the Bandersnatch curve.
//
// The size of this field matches (by design) the size of the prime-order subgroup of the BLS12-381 curve.
type FieldElement = bsFieldElement_64

// NOTE: We intentionally expose copies of unexported variables here to prevent users from modifying bsFieldElement_64_one etc. and
// to give the compiler at least a chance to observe that these are never modified.
// Internal code should not use the exported variables.

var (
	// Important constants of type FieldElement
	FieldElementOne      FieldElement = bsFieldElement_64_one
	FieldElementZero     FieldElement = bsFieldElement_64_zero
	FieldElementMinusOne FieldElement = bsFieldElement_64_minusone
	FieldElementTwo      FieldElement = initFieldElementFromString("2")
)
