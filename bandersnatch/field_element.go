// Note: Name is bsfield_element, because there is already a field_element in bls12381

package bandersnatch

/*
	Trying out various implementations here for field elements of GF(BaseFieldSize)
	Notes: Internal representations are not guaranteed to be stable, may contain pointers or non-unique representations.
	In particular, neither assigment nor comparison operators are guaranteed to work as expected.
*/

// NOTE: The _8 comparison implementation does not have everything implemented.

/*
	This is the intended interface of Field Elements.
	Of course, this cannot be made an actual interface without possibly sacrificing efficiency
	since Go lacks generics:
	(The arguments to Mul etc. in the interface and the concret type need to match, so
	so the actual types' Mul(), Add() etc. implementation would need to accept an
	interface type and start by making a type assertion.)
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

type FieldElement = bsFieldElement_64

var (
	FieldElementOne  = bsFieldElement_64_one
	FieldElementZero = bsFieldElement_64_zero

	// We do not expose FieldElementZero_alt, because users doing IsEqual(&FieldElementZero_alt, .) might call Normalize() on it, which would make
	// IsZero() subsequently fail.
	// FieldElementZero_alt = bsFieldElement_64_zero_alt

	FieldElementMinusOne              = bsFieldElement_64_minusone
	FieldElementTwo      FieldElement = initFieldElementFromString("2")
)
