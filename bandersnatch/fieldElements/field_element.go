package fieldElements

import (
	"fmt"
	"math/big"
	"math/rand"
)

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

type FieldElementInterface_common interface {
	IsZero() bool
	IsOne() bool
	SetOne()
	SetZero()

	Normalize()
	RerandomizeRepresentation(seed uint64) // rerandomize internal representation. Only required for tests and benchmarks.

	Sign() int
	Jacobi() int
	SquareEq()
	NegEq()
	InvEq()
	SetBigInt(x *big.Int)
	SetUint256(x *Uint256)
	SetUint64(x uint64)
	SetInt64(x int64)
	ToBigInt() *big.Int
	ToUint256(x *Uint256)
	ToUint64() (uint64, error)
	ToInt64() (int64, error)

	MulEqFive()

	DoubleEq()

	SetRandomUnsafe(rnd *rand.Rand) // DEPRECATED

	fmt.Formatter
	fmt.Stringer
	ToBytes(buf []byte)
	SetBytes(buf []byte)
	BytesLength() int
	IsEqualAsBigInt(interface{ ToBigInt() *big.Int }) bool
}

type FieldElementInterface[SelfRead any] interface {
	FieldElementInterface_common

	Add(x, y SelfRead)
	Sub(x, y SelfRead)
	Mul(x, y SelfRead)
	Divide(x, y SelfRead)
	Double(x SelfRead)
	Square(x SelfRead)

	MulFive(x SelfRead)
	Neg(x SelfRead)
	Inv(x SelfRead)

	AddEq(y SelfRead)
	SubEq(y SelfRead)
	MulEq(y SelfRead)
	DivideEq(y SelfRead)

	IsEqual(other SelfRead) bool
	CmpAbs(other SelfRead) (absValuesEqual bool, exactlyEqual bool)

	AddInt64(x SelfRead, y int64)
	AddUint64(x SelfRead, y uint64)
	SubInt64(x SelfRead, y int64)
	SubUint64(x SelfRead, y uint64)
	MulInt64(x SelfRead, y int64)
	MulUint64(x SelfRead, y uint64)
	DivideInt64(x SelfRead, y int64)
	DivideUint64(x SelfRead, y uint64)
}
