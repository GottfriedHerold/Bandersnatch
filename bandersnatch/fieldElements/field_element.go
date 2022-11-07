package fieldElements

import "math/big"

// Code for FieldElement (meaning the field of definition of the bandersnatch curve)
// is in field_element_64.go and field_element_8.go
// Only field_element_64.go is used; field_element_8.go serves as a comparison
// implementation that is only used in testing.

// NOTE: The _8 comparison implementation does not have everything implemented.

// TODO: Interface here is not complete.
// Refer to field_element_64.go for what we actually provide.

/*
	This is the intended interface of Field Elements.
	Of course, this cannot be made an actual interface without possibly sacrificing efficiency
	since Go lacks generics:
	(The arguments to Mul etc. in the interface and the concrete type need to match, so
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

type FieldElementInterface[SelfRead any] interface {
	IsZero() bool
	IsOne() bool
	SetOne() bool
	SetZero() bool
	Mul(x, y SelfRead)
	Add(x, y SelfRead)
	Sub(x, y SelfRead)
	Square(x SelfRead)
	Neg(x SelfRead)
	Inv(x SelfRead)
	Divide(x, y SelfRead)
	ToBigInt() *big.Int
	SetBigInt(x *big.Int)
	ToUInt64() (uint64, error)
	SetUInt64(x uint64)
	Normalize()
	IsEqual(other SelfRead) bool
	Sign() int
	Jacobi() int
	AddEq(y SelfRead)
	SubEq(y SelfRead)
	SquareEq()
	DivideEq(y SelfRead)
	NegEq()
}
