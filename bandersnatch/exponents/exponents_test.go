package exponents

import (
	"math/big"
	"math/rand"
	"testing"
)

func (z *Exponent) add_naive(x *Exponent, y *Exponent) {
	xInt := x.ToBigInt_Full()
	yInt := y.ToBigInt_Full()
	zInt := big.NewInt(0)
	zInt.Add(xInt, yInt)
	z.SetBigInt(zInt)
}

func (z *Exponent) sub_naive(x *Exponent, y *Exponent) {
	xInt := x.ToBigInt_Full()
	yInt := y.ToBigInt_Full()
	zInt := big.NewInt(0)
	zInt.Sub(xInt, yInt)
	z.SetBigInt(zInt)
}

func (z *Exponent) mul_naive(x *Exponent, y *Exponent) {
	xInt := x.ToBigInt_Full()
	yInt := y.ToBigInt_Full()
	zInt := big.NewInt(0)
	zInt.Mul(xInt, yInt)
	z.SetBigInt(zInt)
}

func TestAddSubMulScalarField(t *testing.T) {
	const iterations = 10000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var xInt = big.NewInt(0)
	var yInt = big.NewInt(0)
	var x, y, z1, z2 Exponent
	for i := 0; i < iterations; i++ {
		xInt.Rand(drng, GroupOrder_Int) // intentionally larger
		yInt.Rand(drng, GroupOrder_Int)
		x.SetBigInt(xInt)
		y.SetBigInt(yInt)

		z1.Add(&x, &y)
		z2.add_naive(&x, &y)
		if z1 != z2 {
			t.Fatal("Result of Add and add_naive do not match for Exponent")
		}

		z1.Sub(&x, &y)
		z2.sub_naive(&x, &y)
		if z1 != z2 {
			t.Fatal("Result of Sub and sub_naive do not match for Exponent")
		}

		z1.Mul(&x, &y)
		z2.mul_naive(&x, &y)
		if z1 != z2 {
			t.Fatal("Result of Mul and mul_naive do not match for Exponent")
		}
	}
}

func TestUint128Bit(t *testing.T) {
	const iterations = 1000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var num *big.Int = big.NewInt(0)
	for i := 0; i < iterations; i++ {
		num.Rand(drng, twoTo128_Int)
		if i < 100 {
			num.SetInt64(int64(i))
		}
		var num128 uint128
		num128.SetBigInt(num)
		for j := 0; j < 128; j++ {
			if num.Bit(j) != num128.Bit(uint(j)) {
				t.Fatal("num128.Bit does not match big.Int's Bit")
			}
		}
	}
}

func TestGLVBit(t *testing.T) {
	const iterations = 1000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var num *big.Int = big.NewInt(0)
	for i := 0; i < iterations; i++ {
		num.Rand(drng, twoTo128_Int)
		if i < 100 {
			num.SetInt64(int64(i))
		}
		sign := (drng.Intn(2) == 0)
		if sign {
			num.Neg(num)
		}
		var numGLV glvExponent
		numGLV.SetBigInt(num)
		if num.Sign() != 0 && (numGLV.Sign() != num.Sign()) {
			t.Fatal("glvExponent's Sign does not match input via SetBigInt.")
		}
		num.Abs(num) // glvExponent.Bit takes the absolute value (which is appropriate for out application), but big.Int.Bit does not.
		for j := 0; j < 128; j++ {
			if num.Bit(j) != numGLV.Bit(uint(j)) {
				t.Fatal("glvExponent's Bit does not match big.Int's")
			}
		}
	}
}

func TestRoundTripBigIntScalarField(t *testing.T) {
	const iterations = 10000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var xInt = big.NewInt(0)
	var yInt = big.NewInt(1)
	var x, y Exponent
	for i := 0; i < iterations; i++ {
		xInt.Rand(drng, CurveExponent_Int)
		x.SetBigInt(xInt)
		yInt = x.ToBigInt_Full()
		if xInt.Cmp(yInt) != 0 {
			t.Fatal("big.Int conversion roundtrip failure for Scalar field")
		}
		y.SetBigInt(yInt)
		if x != y {
			t.Fatal("big.Int conversion roundtrip failure for Scalar field (2)")
		}
	}

}

func TestAdd128(t *testing.T) {
	const iterations = 10000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var xInt = big.NewInt(0)
	var yInt = big.NewInt(0)
	var zInt = big.NewInt(0)
	var x, y, z1, z2 uint128
	for i := 0; i < iterations; i++ {
		xInt.Rand(drng, twoTo128_Int)
		yInt.Rand(drng, twoTo128_Int)
		zInt.Add(xInt, yInt)
		zInt.Mod(zInt, twoTo128_Int)
		x.SetBigInt(xInt)
		y.SetBigInt(yInt)
		z1.SetBigInt(zInt)
		z2 = add128(x, y)
		if z1 != z2 {
			t.Fatal("add128 does not match Addition of math.big")
		}
	}
}

func TestMul128(t *testing.T) {
	const iterations = 10000
	var drng *rand.Rand = rand.New(rand.NewSource(1000))
	var xInt = big.NewInt(0)
	var yInt = big.NewInt(0)
	var zInt = big.NewInt(0)
	var x, y, z1, z2 uint128
	for i := 0; i < iterations; i++ {
		xInt.Rand(drng, twoTo128_Int)
		yInt.Rand(drng, twoTo128_Int)
		zInt.Mul(xInt, yInt)
		zInt.Mod(zInt, twoTo128_Int)
		x.SetBigInt(xInt)
		y.SetBigInt(yInt)
		z1.SetBigInt(zInt)
		z2 = mul128(x, y)
		if z1 != z2 {
			t.Fatal("add128 does not match Addition of math.big")
		}
	}
}
