package fieldElements

import (
	"fmt"
	"math/big"
	"math/rand"
)

// TODO: DOC

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
