package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// We test uint256.go by running differential tests against big.Int's capabilities.
// Note that, for now, we don't test "special values" nearly enough: -> TODO:
// Use PrecomputedCache's capabilities to pre-seed with special elements.

func TestBigIntToUint256Roundtrip(t *testing.T) {
	prepareTestFieldElements(t)
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
		var z Uint256
		z.FromBigInt(bigInt)
		testutils.FatalUnless(t, z == Uint256{uint64(i), 0, 0, 0}, "")
	}
	minusOne := big.NewInt(-1)
	tooLarge := new(big.Int).Add(twoTo256_Int, big.NewInt(1))

	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUInt256(minusOne) }), "BigIntToUint256 did not panic on negative inputs")
	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUInt256(tooLarge) }), "BigIntToUint256 did not panic on too large inputs")
}

func TestBigIntToUint512Roundtrip(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 1000

	BigSamples := CachedBigInt.GetElements(SeedAndRange{2, twoTo512_Int}, num)
	for _, bigSample := range BigSamples {
		var z Uint512
		z.FromBigInt(bigSample)
		backToBig := z.ToBigInt()
		testutils.FatalUnless(t, backToBig != bigSample, "Aliasing detected") // Note: Comparison is between pointers
		testutils.FatalUnless(t, backToBig.Cmp(bigSample) == 0, "Roundtrip failure")
	}
	for i := int64(0); i < 10; i++ {
		bigInt := big.NewInt(i)
		var z Uint512
		z.FromBigInt(bigInt)
		testutils.FatalUnless(t, z == Uint512{uint64(i), 0, 0, 0, 0, 0, 0, 0}, "")
	}
	minusOne := big.NewInt(-1)
	tooLarge := new(big.Int).Add(twoTo512_Int, big.NewInt(1))

	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUint512(minusOne) }), "BigIntToUint512 did not panic on negative inputs")
	testutils.FatalUnless(t, testutils.CheckPanic(func() { BigIntToUint512(tooLarge) }), "BigIntToUint512 did not panic on too large inputs")
}

func TestUint256Add(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint256
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
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint256
	for _, x := range xs {
		for _, y := range ys {
			carry1 := z1.AddAndReturnCarry(&x, &y) == 1
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
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint256
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
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint256
	for _, x := range xs {
		for _, y := range ys {
			borrow1 := z1.SubAndReturnBorrow(&x, &y) == 1
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
	prepareTestFieldElements(t)

	const num = 1000
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	for _, x := range xs {
		res1 := x.IsZero()
		xInt := x.ToBigInt()
		res2 := xInt.Sign() == 0
		testutils.FatalUnless(t, res1 == res2, "Is Zero differs from big.Int")
	}

	for _, x := range []Uint256{{0, 0, 0, 0}, {1, 1, 1, 1}, {0, 0, 0, 1}, {1, 0, 0, 0}} {
		res0 := (x == Uint256{0, 0, 0, 0})
		res1 := x.IsZero()
		xInt := x.ToBigInt()
		res2 := xInt.Sign() == 0
		testutils.FatalUnless(t, res1 == res2, "IsZero differs from big.Int")
		testutils.FatalUnless(t, res1 == res0, "IsZero wrong")
	}
}

func TestLongMul256By64(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000
	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint64.GetElements(1, num)

	for _, x := range xs {
		for _, y := range ys {
			xInt := x.ToBigInt()
			yInt := new(big.Int).SetUint64(y)
			zInt := new(big.Int).Mul(xInt, yInt)
			var z [5]uint64
			LongMul256By64(&z, &x, y)

			// convert z to resInt of type *big.Int. This is somewhat involved.
			var zLow Uint256 = *(*[4]uint64)(z[0:4]) // lower-order words of z
			zLowInt := zLow.ToBigInt()
			resInt := new(big.Int).SetUint64(z[4]) // high-order word of z
			resInt.Mul(resInt, twoTo256_Int)
			resInt.Add(resInt, zLowInt) // add up high- and low-order words

			testutils.FatalUnless(t, zInt.Cmp(resInt) == 0, "256bit x 64bit -> 320bit multiplication is wrong")

			// compare with deprecated function
			low2, high2 := mul_four_one_64(&x, y)
			testutils.FatalUnless(t, low2 == z[0], "")
			testutils.FatalUnless(t, utils.CompareSlices(high2[:], z[1:5]), "")
		}
	}
}

func TestUint256LongMul(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint512
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
	prepareTestFieldElements(t)

	const num = 2048
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	var z1, z2 Uint512
	for _, x := range xs {
		z1.LongSquare(&x)
		xInt := x.ToBigInt()
		zInt := new(big.Int).Mul(xInt, xInt)
		// zInt.Mod(zInt, twoTo512_Int) -- No modular reduction here.
		z2.FromBigInt(zInt)
		testutils.FatalUnless(t, z1 == z2, "LongSquare result differs from big.Int")
	}

}

func TestUint256Cmp(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 256
	xs := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	ys := CachedUint256.GetElements(SeedAndRange{seed: 1, allowedRange: twoTo256_Int}, num)
	for _, x := range xs {
		for _, y := range ys {
			xInt := x.ToBigInt()
			yInt := y.ToBigInt()
			testutils.FatalUnless(t, xInt.Cmp(yInt) == x.Cmp(&y), "Cmp differs from big.Int")
		}
	}
}
