package fieldElements

import (
	"fmt"
	"math/big"
	"math/rand"
)

// naive implementation. Field elements are represented as (unsigned) big-endian byte slices representing an integer in [0, BaseFieldSize).
// Different from most other code, we use a big-endian representation here, because big.Int is easily convertible to/from it,
// although big.Int's internal representation is little-endian word slices + sign bits)
// Note that we decide not to embed a big.Int, to avoid pointer indirection: This makes assigment, equality and zero-initialization actually work
// at the expense of speed, but this is only really used to test other implementations against for correctness anyway.
type bsFieldElement_8 struct {
	v [32]byte
}

var bsFieldElement_8_zero bsFieldElement_8

var bsFieldElement_8_one bsFieldElement_8 = bsFieldElement_8{v: [32]byte{31: 1}}

var bsFieldElement_8_oneHalf bsFieldElement_8 = func() (ret bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(1)
	var twoInt *big.Int = big.NewInt(2)
	xInt.Add(xInt, baseFieldSize_Int)
	xInt.Div(xInt, twoInt)
	xInt.FillBytes(ret.v[:])
	return
}()

var oneHalfModBaseField_Int *big.Int = func() (ret *big.Int) {
	ret = big.NewInt(1)
	var twoInt *big.Int = big.NewInt(2)
	ret.Add(ret, baseFieldSize_Int)
	ret.Div(ret, twoInt)
	return
}()

func (z *bsFieldElement_8) Add(x, y *bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	var yInt *big.Int = big.NewInt(0).SetBytes(y.v[:])
	xInt.Add(xInt, yInt)
	xInt.Mod(xInt, baseFieldSize_Int)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) Sub(x, y *bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	var yInt *big.Int = big.NewInt(0).SetBytes(y.v[:])
	xInt.Sub(xInt, yInt)
	xInt.Mod(xInt, baseFieldSize_Int) // Note that Int.Mod returns elements in [0, BaseFieldSize), even if xInt is negative.
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) IsZero() bool {
	return z.v == bsFieldElement_8_zero.v
}

func (z *bsFieldElement_8) IsOne() bool {
	return z.v == bsFieldElement_8_one.v
}

func (z *bsFieldElement_8) SetOne() {
	z.v = bsFieldElement_8_one.v
}

func (z *bsFieldElement_8) SetZero() {
	z.v = bsFieldElement_8_zero.v
}

func (z *bsFieldElement_8) Mul(x, y *bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	var yInt *big.Int = big.NewInt(0).SetBytes(y.v[:])
	xInt.Mul(xInt, yInt)
	xInt.Mod(xInt, baseFieldSize_Int)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) ToBigInt() *big.Int {
	var xInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	return xInt
}

func (z *bsFieldElement_8) SetBigInt(v *big.Int) {
	var xInt *big.Int = big.NewInt(0)
	xInt.Mod(v, baseFieldSize_Int)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) SetUInt64(value uint64) {
	var xInt *big.Int = big.NewInt(0)
	xInt.SetUint64(value)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) ToUint64() (result uint64, err bool) {
	var zInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	err = !z.ToBigInt().IsUint64()
	if !err {
		result = zInt.Uint64()
	}
	return
}

// TODO: Move to *_test.go file? It's only used in testing

// generates a random field element. Non crypto-grade randomness. Used for testing/benchmarking only.
func (z *bsFieldElement_8) setRandomUnsafe(rnd *rand.Rand) {
	var xInt *big.Int = big.NewInt(0).Rand(rnd, baseFieldSize_Int)
	xInt.FillBytes(z.v[:])
}

// useful for debugging
func (z bsFieldElement_8) Format(s fmt.State, ch rune) {
	var xInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	xInt.Format(s, ch)
}

func (z bsFieldElement_8) String() string {
	var zInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	return zInt.String()
}

// multiplicative inverse
func (z *bsFieldElement_8) Inv(x *bsFieldElement_8) {
	var xInt *big.Int = new(big.Int).SetBytes(x.v[:])
	xInt.ModInverse(xInt, baseFieldSize_Int)
	xInt.FillBytes(z.v[:])
}

// Changes to a unique internal representation
func (z *bsFieldElement_8) Normalize() {
	// no-op
}

// Comparison
func (z *bsFieldElement_8) IsEqual(x *bsFieldElement_8) bool {
	return *z == *x
}

func (z *bsFieldElement_8) Neg(x *bsFieldElement_8) {
	if x.IsZero() {
		return
	}
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	xInt.Sub(baseFieldSize_Int, xInt)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) AddEq(x *bsFieldElement_8) {
	z.Add(z, x)
}

func (z *bsFieldElement_8) SubEq(x *bsFieldElement_8) {
	z.Sub(z, x)
}

func (z *bsFieldElement_8) MulEq(x *bsFieldElement_8) {
	z.Mul(z, x)
}

func (z *bsFieldElement_8) Square(x *bsFieldElement_8) {
	z.Mul(x, x)
}

func (z *bsFieldElement_8) SquareEq() {
	z.Mul(z, z)
}

func (z *bsFieldElement_8) InvEq() {
	z.Inv(z)
}

func (z *bsFieldElement_8) NegEq() {
	z.Neg(z)
}

func (z *bsFieldElement_8) Divide(num *bsFieldElement_8, denom *bsFieldElement_8) {
	var temp bsFieldElement_8
	temp.Inv(denom)
	z.Mul(&temp, num)
}

func (z *bsFieldElement_8) DivideEq(denom *bsFieldElement_8) {
	z.Divide(z, denom)
}

func (z *bsFieldElement_8) Sign() int {
	if z.IsZero() {
		return 0
	}
	var zInt *big.Int = new(big.Int).SetBytes(z.v[:])
	// oneHalfModBaseField_Int = (BaseFieldSize+1)/2. The case of == here gives -1
	if oneHalfModBaseField_Int.CmpAbs(zInt) == 1 {
		return 1
	} else {
		return -1
	}
}

func (z *bsFieldElement_8) Jacobi() int {
	var zInt *big.Int = new(big.Int).SetBytes(z.v[:])
	return big.Jacobi(zInt, baseFieldSize_Int)
}

func (z *bsFieldElement_8) SquareRoot(x *bsFieldElement_8) (ok bool) {
	var xInt *big.Int = new(big.Int).SetBytes(x.v[:])
	if xInt.ModSqrt(xInt, baseFieldSize_Int) == nil {
		return false
	}
	xInt.FillBytes(z.v[:])
	return true
}
