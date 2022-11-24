package fieldElements

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// This file contains an implementation of the FieldElementInterface that
// just forwards everything to [*big.Int]. This will be horrendously inefficient;
// the sole purpose of this is to enable differential tests and get a baseline for efficiency.

// NOTE: We cannot really store a *big.Int inside our field elements (or even a big.Int), because that would cause aliasing problems:
// Part of our contract is that plain assignment x = y of field elements works and does not cause any aliasing relationship between x and y, so posterior changes to x will not affect y.
// As a consequence, every single operation has to convert back-and-forth.

type bsFieldElement_BigInt struct {
	value Uint256 // required to be in [0, BaseFieldSize) -- we have a unique representation
}

func (z *bsFieldElement_BigInt) ToBigInt() *big.Int {
	return z.value.ToBigInt()
}

func (z *bsFieldElement_BigInt) SetBigInt(x *big.Int) {
	xReduced := new(big.Int).Mod(x, baseFieldSize_Int)
	z.value.SetBigInt(xReduced)
}

func (z *bsFieldElement_BigInt) IsZero() bool {
	return z.value.IsZero()
}

func (z *bsFieldElement_BigInt) IsOne() bool {
	return z.value.IsOne()
}

func (z *bsFieldElement_BigInt) SetZero() {
	z.value.SetZero()
}

func (z *bsFieldElement_BigInt) SetOne() {
	z.value.SetOne()
}

func (z *bsFieldElement_BigInt) Sign() int {
	if z.IsZero() {
		return 0
	} else if z.value.Cmp(&oneHalfModBaseField_uint256) < 0 {
		return +1
	} else {
		return -1
	}
}

func (z *bsFieldElement_BigInt) Jacobi() int {
	zInt := z.ToBigInt()
	return big.Jacobi(zInt, baseFieldSize_Int)
}

func (z *bsFieldElement_BigInt) Add(x, y *bsFieldElement_BigInt) {
	xInt := x.ToBigInt()
	yInt := y.ToBigInt()
	zInt := new(big.Int).Add(xInt, yInt)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) Sub(x, y *bsFieldElement_BigInt) {
	xInt := x.ToBigInt()
	yInt := y.ToBigInt()
	zInt := new(big.Int).Sub(xInt, yInt)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) Mul(x, y *bsFieldElement_BigInt) {
	xInt := x.ToBigInt()
	yInt := y.ToBigInt()
	zInt := new(big.Int).Mul(xInt, yInt)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) Neg(x *bsFieldElement_BigInt) {
	xInt := x.ToBigInt()
	zInt := new(big.Int).Neg(xInt)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) Inv(x *bsFieldElement_BigInt) {
	if x.IsZero() {
		panic(ErrDivisionByZero)
	}
	xInt := x.ToBigInt()
	zInt := new(big.Int).ModInverse(xInt, baseFieldSize_Int)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) SquareEq() {
	z.Mul(z, z)
}

func (z *bsFieldElement_BigInt) NegEq() {
	z.Neg(z)
}

func (z *bsFieldElement_BigInt) InvEq() {
	z.Inv(z)
}

func (z *bsFieldElement_BigInt) SetUint256(x *Uint256) {
	z.value = *x
	z.value.reduceBarret_fa()
}

func (z *bsFieldElement_BigInt) ToUint256(x *Uint256) {
	*x = z.value
}

func (z *bsFieldElement_BigInt) SetUint64(x uint64) {
	z.value.SetUint64(x)
}

func (z *bsFieldElement_BigInt) SetInt64(x int64) {
	if x >= 0 {
		z.value.SetUint64(uint64(x))
	} else {
		z.value.SetUint64(uint64(-x))
		z.NegEq()
	}
}

func (z *bsFieldElement_BigInt) ToUint64() (uint64, error) {
	zInt := z.ToBigInt()
	if !zInt.IsUint64() {
		return 0, errorsWithData.NewErrorWithParameters(ErrCannotRepresentFieldElement, ErrorPrefix+"the field Element %v{FieldElement} cannot be represented as an uint64", "FieldElement", *z)
	}
	return zInt.Uint64(), nil
}

func (z *bsFieldElement_BigInt) ToInt64() (int64, error) {
	var zInt *big.Int
	if z.Sign() >= 0 {
		zInt = z.ToBigInt()
	} else {
		// make zInt negative representation of z
		var zNeg bsFieldElement_BigInt
		zNeg.Neg(z)
		zInt = zNeg.ToBigInt()
		zInt.Neg(zInt)
	}
	if !zInt.IsInt64() {
		return 0, errorsWithData.NewErrorWithParameters(ErrCannotRepresentFieldElement, ErrorPrefix+"the field Element %v{FieldElement} cannot be represented as an int64", "FieldElement", *z)
	} else {
		return zInt.Int64(), nil
	}
}

func (z *bsFieldElement_BigInt) MulFive(x *bsFieldElement_BigInt) {
	xInt := x.ToBigInt()
	Int5 := big.NewInt(5)
	zInt := new(big.Int).Mul(xInt, Int5)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) MulEqFive() {
	z.MulFive(z)
}

func (z *bsFieldElement_BigInt) DoubleEq() {
	z.Add(z, z)
}

func (z bsFieldElement_BigInt) Format(s fmt.State, ch rune) {
	z.ToBigInt().Format(s, ch)
}

func (z bsFieldElement_BigInt) String() string {
	return z.ToBigInt().String()
}

func (z *bsFieldElement_BigInt) ToBytes(buf []byte) {
	binary.LittleEndian.PutUint64(buf[0:8], z.value[0])
	binary.LittleEndian.PutUint64(buf[8:16], z.value[1])
	binary.LittleEndian.PutUint64(buf[16:24], z.value[2])
	binary.LittleEndian.PutUint64(buf[24:32], z.value[3])
}

func (z *bsFieldElement_BigInt) SetBytes(buf []byte) {
	z.value[0] = binary.LittleEndian.Uint64(buf[0:8])
	z.value[1] = binary.LittleEndian.Uint64(buf[8:16])
	z.value[2] = binary.LittleEndian.Uint64(buf[16:24])
	z.value[3] = binary.LittleEndian.Uint64(buf[24:32])
	// Check validity. This is not required by contract, but we have no reason not to.
	zInt := z.ToBigInt()
	if zInt.Cmp(baseFieldSize_Int) >= 0 {
		panic("SetBytes called on bsFieldElement_BigInt with byte slice that is not a valid internal representation")
	}
}

func (z *bsFieldElement_BigInt) BytesLength() int { return 32 }

func (z *bsFieldElement_BigInt) Normalize() {
	// do nothing
}

func (z *bsFieldElement_BigInt) RerandomizeRepresentation(seed uint64) {
	// do nothing
}

// DEPRECATED
func (z *bsFieldElement_BigInt) SetRandomUnsafe(rnd *rand.Rand) {
	zInt := new(big.Int)
	zInt.Rand(rnd, baseFieldSize_Int)
	z.SetBigInt(zInt)
}

func (z *bsFieldElement_BigInt) Divide(x, y *bsFieldElement_BigInt) {
	var yInv bsFieldElement_BigInt
	yInv.Inv(y)
	z.Mul(x, &yInv)
}

func (z *bsFieldElement_BigInt) Double(x *bsFieldElement_BigInt) {
	z.Add(x, x)
}

func (z *bsFieldElement_BigInt) Square(x *bsFieldElement_BigInt) {
	z.Mul(x, x)
}

func (z *bsFieldElement_BigInt) AddEq(y *bsFieldElement_BigInt) {
	z.Add(z, y)
}

func (z *bsFieldElement_BigInt) SubEq(y *bsFieldElement_BigInt) {
	z.Sub(z, y)
}

func (z *bsFieldElement_BigInt) MulEq(y *bsFieldElement_BigInt) {
	z.Mul(z, y)
}

func (z *bsFieldElement_BigInt) DivideEq(y *bsFieldElement_BigInt) {
	z.Divide(z, y)
}

func (z *bsFieldElement_BigInt) IsEqual(y *bsFieldElement_BigInt) bool {
	return *z == *y
}

func (z *bsFieldElement_BigInt) CmpAbs(y *bsFieldElement_BigInt) (absValsEqual bool, exactlyEqual bool) {
	exactlyEqual = (*y == *z)
	var yNeg bsFieldElement_BigInt
	yNeg.Neg(y)
	absValsEqual = exactlyEqual || (yNeg == *z)
	return
}

func (z *bsFieldElement_BigInt) AddInt64(x *bsFieldElement_BigInt, y int64) {
	var yFE bsFieldElement_BigInt
	yFE.SetInt64(y)
	z.Add(x, &yFE)
}

func (z *bsFieldElement_BigInt) AddUint64(x *bsFieldElement_BigInt, y uint64) {
	var yFE bsFieldElement_BigInt
	yFE.SetUint64(y)
	z.Add(x, &yFE)
}

func (z *bsFieldElement_BigInt) SubInt64(x *bsFieldElement_BigInt, y int64) {
	var yFE bsFieldElement_BigInt
	yFE.SetInt64(y)
	z.Sub(x, &yFE)
}

func (z *bsFieldElement_BigInt) SubUint64(x *bsFieldElement_BigInt, y uint64) {
	var yFE bsFieldElement_BigInt
	yFE.SetUint64(y)
	z.Sub(x, &yFE)
}

func (z *bsFieldElement_BigInt) MulInt64(x *bsFieldElement_BigInt, y int64) {
	var yFE bsFieldElement_BigInt
	yFE.SetInt64(y)
	z.Mul(x, &yFE)
}

func (z *bsFieldElement_BigInt) MulUint64(x *bsFieldElement_BigInt, y uint64) {
	var yFE bsFieldElement_BigInt
	yFE.SetUint64(y)
	z.Mul(x, &yFE)
}

func (z *bsFieldElement_BigInt) DivideInt64(x *bsFieldElement_BigInt, y int64) {
	var yFE bsFieldElement_BigInt
	yFE.SetInt64(y)
	z.Divide(x, &yFE)
}

func (z *bsFieldElement_BigInt) DivideUint64(x *bsFieldElement_BigInt, y uint64) {
	var yFE bsFieldElement_BigInt
	yFE.SetUint64(y)
	z.Divide(x, &yFE)
}
