package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestBigIntToUint256Roundtrip(t *testing.T) {
	const num = 1000

	BigSamples := CachedBigInt.GetElements(SeedAndRange{1, twoTo256_Int}, num)
	for _, bigSample := range BigSamples {
		z := BigIntToUInt256(bigSample)
		backToBig := z.ToBigInt()
		testutils.FatalUnless(t, backToBig != bigSample, "Aliasing detected") // Note: Comparison is between pointers
		testutils.FatalUnless(t, backToBig.Cmp(bigSample) == 0, "Roundtrip failure")
	}
	for i := int64(0); i < 10; i++ {
		bigInt := big.NewInt(i)
		var z uint256
		z.FromBigInt(bigInt)
		testutils.FatalUnless(t, z == uint256{uint64(i), 0, 0, 0}, "")
	}
	minusOne := big.NewInt(-1)
	tooLarge := new(big.Int).Add(twoTo256_Int, big.NewInt(1))

	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUInt256(minusOne) }), "BigIntToUint256 did not panic on negative inputs")
	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUInt256(tooLarge) }), "BigIntToUint256 did not panic on too large inputs")
}

func TestBigIntToUint512Roundtrip(t *testing.T) {
	const num = 1000

	BigSamples := CachedBigInt.GetElements(SeedAndRange{2, twoTo512_Int}, num)
	for _, bigSample := range BigSamples {
		var z uint512
		z.FromBigInt(bigSample)
		backToBig := z.ToBigInt()
		testutils.FatalUnless(t, backToBig != bigSample, "Aliasing detected") // Note: Comparison is between pointers
		testutils.FatalUnless(t, backToBig.Cmp(bigSample) == 0, "Roundtrip failure")
	}
	for i := int64(0); i < 10; i++ {
		bigInt := big.NewInt(i)
		var z uint512
		z.FromBigInt(bigInt)
		testutils.FatalUnless(t, z == uint512{uint64(i), 0, 0, 0, 0, 0, 0, 0}, "")
	}
	minusOne := big.NewInt(-1)
	tooLarge := new(big.Int).Add(twoTo512_Int, big.NewInt(1))

	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUint512(minusOne) }), "BigIntToUint512 did not panic on negative inputs")
	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUint512(tooLarge) }), "BigIntToUint512 did not panic on too large inputs")
}

func TestUint256Add(t *testing.T) {
	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint256
	for _, x := range xs {
		for _, y := range ys {
			z1.Add(&x, &y)
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			zInt := new(big.Int).Add(xInt, yInt)
			zInt.Mod(zInt, twoTo256_Int)
			z2.FromBigInt(zInt)
			testutils.FatalUnless(t, z1 == z2, "Addition result differs from big.Int addition")
		}
	}
}

func TestUint256AddWithCarry(t *testing.T) {
	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint256
	for _, x := range xs {
		for _, y := range ys {
			carry1 := z1.AddWithCarry(&x, &y) == 1
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			zInt := new(big.Int).Add(xInt, yInt)
			carry2 := zInt.Cmp(twoTo256_Int) >= 0
			zInt.Mod(zInt, twoTo256_Int)
			z2.FromBigInt(zInt)
			testutils.FatalUnless(t, z1 == z2, "Addition result differs from big.Int addition")
			testutils.FatalUnless(t, carry1 == carry2, "Addition result differs from big.Int addition")
		}
	}
}

func TestUint256Sub(t *testing.T) {
	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint256
	for _, x := range xs {
		for _, y := range ys {
			z1.Sub(&x, &y)
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			zInt := new(big.Int).Sub(xInt, yInt)
			zInt.Mod(zInt, twoTo256_Int)
			z2.FromBigInt(zInt)
			testutils.FatalUnless(t, z1 == z2, "Subtraction result differs from big.Int addition")
		}
	}
}

func TestUint256SubWithBorrow(t *testing.T) {
	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint256
	for _, x := range xs {
		for _, y := range ys {
			borrow1 := z1.SubWithBorrow(&x, &y) == 1
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			zInt := new(big.Int).Sub(xInt, yInt)
			borrow2 := zInt.Sign() < 0
			zInt.Mod(zInt, twoTo256_Int)
			z2.FromBigInt(zInt)
			testutils.FatalUnless(t, z1 == z2, "Subtraction result differs from big.Int addition")
			testutils.FatalUnless(t, borrow1 == borrow2, "Subtraction result differs from big.Int addition")
		}
	}
}

func TestUint256IsZero(t *testing.T) {
	const num = 1000
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	for _, x := range xs {
		res1 := x.IsZero()
		xInt := x.ToBigInt()
		res2 := xInt.Sign() == 0
		testutils.FatalUnless(t, res1 == res2, "Is Zero differs from big.Int")
	}

	for _, x := range []uint256{{0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 1}, {1, 0, 0, 0}} {
		res0 := (x == uint256{0, 0, 0, 0})
		res1 := x.IsZero()
		xInt := x.ToBigInt()
		res2 := xInt.Sign() == 0
		testutils.FatalUnless(t, res1 == res2, "IsZero differs from big.Int")
		testutils.FatalUnless(t, res1 == res0, "IsZero wrong")
	}
}

func TestUint256LongMul(t *testing.T) {
	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint512
	for _, x := range xs {
		for _, y := range ys {
			z1.LongMul(&x, &y)
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			zInt := new(big.Int).Mul(xInt, yInt)
			// zInt.Mod(zInt, twoTo512_Int) -- No modular reduction here.
			z2.FromBigInt(zInt)
			testutils.FatalUnless(t, z1 == z2, "Long-Mul result differs from big.Int")
		}
	}
}

func TestUint256Square(t *testing.T) {
	const num = 2048
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 uint512
	for _, x := range xs {
		z1.LongSquare(&x)
		xInt := x.ToBigInt()
		zInt := new(big.Int).Mul(xInt, xInt)
		// zInt.Mod(zInt, twoTo512_Int) -- No modular reduction here.
		z2.FromBigInt(zInt)
		testutils.FatalUnless(t, z1 == z2, "LongSquare result differs from big.Int")
	}

}
