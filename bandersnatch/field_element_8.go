package bandersnatch

import (
	"fmt"
	"math/big"
	"math/rand"
)

// naive implementation. Field elements are represented as (unsigned) big-endian byte slices representing an integer in [0, BaseFieldSize)
// (because big.Int is convertible to/from it, although big.Int's internal representation is little-endian word slices + sign bits)
// Note that we decide not to embed a big.Int, to avoid pointer indirection: This makes assigment, equality and zero-initialization actually work
// at the expense of speed, but this is only really used to test other implementations against for correctness anyway.
type bsFieldElement_8 struct {
	v [32]byte
}

var bsFieldElement_8_zero bsFieldElement_8

var bsFieldElement_8_one bsFieldElement_8 = bsFieldElement_8{v: [32]byte{31: 1}}

func (z *bsFieldElement_8) Add(x, y *bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	var yInt *big.Int = big.NewInt(0).SetBytes(y.v[:])
	xInt.Add(xInt, yInt)
	xInt.Mod(xInt, BaseFieldSize)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) Sub(x, y *bsFieldElement_8) {
	var xInt *big.Int = big.NewInt(0).SetBytes(x.v[:])
	var yInt *big.Int = big.NewInt(0).SetBytes(y.v[:])
	xInt.Sub(xInt, yInt)
	xInt.Mod(xInt, BaseFieldSize) // Note that Int.Mod returns elements in [0, BaseFieldSize), even if xInt is negative.
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
	xInt.Mod(xInt, BaseFieldSize)
	xInt.FillBytes(z.v[:])
}

func (z *bsFieldElement_8) ToInt() *big.Int {
	var xInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	return xInt
}

func (z *bsFieldElement_8) SetInt(v *big.Int) {
	var xInt *big.Int = big.NewInt(0)
	xInt.Mod(v, BaseFieldSize)
	xInt.FillBytes(z.v[:])
}

// generates a random field element. Non crypto-grade randomness. Used for testing only.
func (z *bsFieldElement_8) setRandomUnsafe(rnd *rand.Rand) {
	var xInt *big.Int = big.NewInt(0).Rand(rnd, BaseFieldSize)
	xInt.FillBytes(z.v[:])
}

// useful for debugging
func (z *bsFieldElement_8) Format(s fmt.State, ch rune) {
	var xInt *big.Int = big.NewInt(0).SetBytes(z.v[:])
	xInt.Format(s, ch)
}

// multiplicative inverse
func (z *bsFieldElement_8) Inv(x *bsFieldElement_8) {
	var xInt *big.Int = new(big.Int).SetBytes(x.v[:])
	xInt.ModInverse(xInt, BaseFieldSize)
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
