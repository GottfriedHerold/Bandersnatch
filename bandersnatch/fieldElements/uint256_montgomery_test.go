package fieldElements

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func TestMontgomeryDifferential(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 256
	xs := CachedUint256.GetElements(pc_uint256_c, num)
	ys := CachedUint256.GetElements(pc_uint256_c, num)
	for _, x := range xs {
		for _, y := range ys {
			var z1, z2 Uint256
			z1.mulMontgomerySlow_c(&x, &y)
			z2.mulMontgomery_Unrolled_c(&x, &y)
			// z1.reduceBarret_fa()
			// z2.reduceBarret_fa()
			testutils.FatalUnless(t, z1 == z2, "MulMontgomery and MulMontgomeryV2 differ")
		}
	}
}

func TestToNonMontgomery_fc(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 10000

	xs := CachedUint256.GetElements(pc_uint256_c, num)
	for _, x := range xs {
		xCopy := x
		xInt := x.ToBigInt()
		z := x.ToNonMontgomery_fc()
		testutils.FatalUnless(t, xCopy == x, "ToNonMontgomery changes argument")
		testutils.FatalUnless(t, z.IsReduced_f(), "ToNonMontgomery_fc does not properly reduce")
		zInt := z.ToBigInt()
		zInt.Mul(zInt, twoTo256_Int)
		zInt.Mod(zInt, baseFieldSize_Int)
		xInt.Mod(xInt, baseFieldSize_Int)
		testutils.FatalUnless(t, xInt.Cmp(zInt) == 0, "ToNonMontgomery does not Multiply by 2**-256 mod BaseFieldSize")

		var z2 Uint256
		z2.FromMontgomeryRepresentation_fc(&xCopy)
		testutils.FatalUnless(t, z2 == z, "FromMontgomery differs from ToNonMontgomery")

	}
}

func TestAddMulShift64(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 500
	xs := CachedUint256.GetElements(pc_uint256_a, num)
	ys := CachedUint64.GetElements(1, num)
	targets := CachedUint256.GetElements(pc_uint256_a, num)
	for i, x := range xs {
		y := ys[i]
		for _, target := range targets {
			targetCopy := target
			xCopy := x
			yCopy := y
			xInt := x.ToBigInt()
			yInt := new(big.Int).SetUint64(y)
			tInt := target.ToBigInt()

			tempInt := new(big.Int).Mul(xInt, yInt)
			tempInt.Add(tempInt, tInt)
			expectedLow := new(big.Int).Mod(tempInt, common.TwoTo64_Int)
			expectedHigh := new(big.Int).Rsh(tempInt, 64)

			low := add_mul_shift_64(&targetCopy, &x, y)

			testutils.FatalUnless(t, x == xCopy, "")
			testutils.FatalUnless(t, y == yCopy, "")
			testutils.FatalUnless(t, expectedLow.Uint64() == low, "output word of add_mul_shift_64 is wrong")
			resInt := targetCopy.ToBigInt()
			testutils.FatalUnless(t, resInt.Cmp(expectedHigh) == 0, "target words of add_mul_shift_64 differ from big.Int computation")
		}
	}
}

func TestMontgomeryStep(t *testing.T) {
	prepareTestFieldElements(t)

	const num = 5000

	xs := CachedUint256.GetElements(pc_uint256_c, num)
	qs := CachedUint64.GetElements(1, num)

	for i, xCopy := range xs {
		q := qs[i]
		x := xCopy
		xInt := x.ToBigInt()
		qInt := new(big.Int).SetUint64(q)

		expectedResult := new(big.Int).Mul(qInt, baseFieldSize_Int)
		expectedResult.Rsh(expectedResult, 64)
		expectedResult.Add(expectedResult, common.One_Int)
		expectedResult.Add(expectedResult, xInt) // ((q*baseField) >> 64) + x + 1

		montgomery_step_64(&x, q)

		result := x.ToBigInt()
		// Note: This also implies that no overflow happened, since otherwise expectedResult >= 2**256 and the test cannot succeed.
		testutils.FatalUnless(t, result.Cmp(expectedResult) == 0, "montgomery_step_64 does not match specification")

	}
}

func TestUint256_MontgomeryModularExponentiation(t *testing.T) {
	prepareTestFieldElements(t)
	const num = 1000

	bases := CachedUint256.GetElements(SeedAndRange{seed: 10003, allowedRange: twoTo256_Int}, num)
	exponents := CachedUint256.GetElements(SeedAndRange{seed: 10004, allowedRange: twoTo256_Int}, num)

	var target Uint256
	target.modularExponentiationSlidingWindowMontgomery_fa(&zero_uint256, &zero_uint256)
	testutils.FatalUnless(t, target == twoTo256ModBaseField_uint256, "0^0 != 1")

	for i, basis := range bases {
		exponent := exponents[i]
		exponentCopy := exponent
		basisCopy := basis
		target.modularExponentiationSlidingWindowMontgomery_fa(&basis, &exponent)
		testutils.FatalUnless(t, target.IsReduced_f(), "ModularExponentiationMontgomery_fa does not fully reduce")

		basisInt := basis.ToBigInt()
		invserseMultiplier := new(big.Int).ModInverse(twoTo256_Int, baseFieldSize_Int)
		basisInt.Mul(basisInt, invserseMultiplier)
		exponentInt := exponent.ToBigInt()
		targetInt := new(big.Int).Exp(basisInt, exponentInt, baseFieldSize_Int)
		targetInt.Mul(targetInt, twoTo256_Int)
		targetInt.Mod(targetInt, baseFieldSize_Int)
		testutils.FatalUnless(t, target.ToBigInt().Cmp(targetInt) == 0, "ModularExponentiaionMontgomery_fa does not match big.Int")

		target = basis
		target.modularExponentiationSlidingWindowMontgomery_fa(&target, &exponent)
		testutils.FatalUnless(t, target.ToBigInt().Cmp(targetInt) == 0, "ModularExponentiaionMontgomery_fa does not work for aliasing args")

		dummy1 := basisCopy
		dummy2 := basisCopy
		target.modularExponentiationSlidingWindowMontgomery_fa(&dummy1, &dummy2)
		dummy1.modularExponentiationSlidingWindowMontgomery_fa(&dummy1, &dummy1)
		testutils.FatalUnless(t, dummy1 == target, "ModularExponentiationMontgomery_fa does not work for target, basis, exponent all aliasing")

		target.modularExponentiationSlidingWindowMontgomery_fa(&basis, &one_uint256)
		dummy1 = basis
		dummy1.reduceBarret_fa()
		testutils.FatalUnless(t, target == dummy1, "x^1 != x modulo BaseFieldSize")
		target.modularExponentiationSlidingWindowMontgomery_fa(&basis, &zero_uint256)
		testutils.FatalUnless(t, target == twoTo256ModBaseField_uint256, "x^0 != 1 (in Montgomery Form)")

		testutils.FatalUnless(t, exponent == exponentCopy, "Argument was modified during ModularExponentiation")
		testutils.FatalUnless(t, basis == basisCopy, "Argument was modified during ModularExponentiation")

	}

}

/****************
OLD TESTS
somewhat outdated, incompatible in style and redundant, but the tests themselves are valid, so there is no harm in keeping them for now.
*************/

func TestMulHelpers(testing_instance *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(11141))
	const iterations = 1000
	bound := big.NewInt(1)
	bound.Lsh(bound, 256) // bound = 2^256

	R := big.NewInt(1)
	R.Lsh(R, 64) // R = 2^64

	oneInt := big.NewInt(1)

	// Test mul_four_one_64 by comparing to big.Int computation on random inputs x, y
	for i := 1; i < iterations; i++ {
		xInt := new(big.Int).Rand(drng, bound)
		var x Uint256 = utils.BigIntToUIntArray(xInt)

		var y uint64 = drng.Uint64()
		yInt := new(big.Int).SetUint64(y)

		// x*y as computed via big.Int.Mul
		resultInt := new(big.Int).Mul(xInt, yInt)

		low, high := mul_four_one_64(&x, y)
		lowInt := new(big.Int).SetUint64(low)
		highInt := utils.UIntarrayToInt((*[4]uint64)(&high))
		resultInt2 := new(big.Int).Mul(highInt, R)

		// x*y as computed using mul_four_one
		resultInt2.Add(resultInt2, lowInt)

		if resultInt.Cmp(resultInt2) != 0 {
			testing_instance.Error("mul_four_one is incorrect")
			break
		}
	}

	// Test montgomery_step_64
	for i := 1; i < iterations; i++ {
		tInt := new(big.Int).Rand(drng, bound)
		var t Uint256 = utils.BigIntToUIntArray(tInt)

		var q uint64 = drng.Uint64()
		qInt := new(big.Int).SetUint64(q)

		qInt.Mul(qInt, baseFieldSize_Int)
		qInt.Div(qInt, R)
		tInt.Add(tInt, qInt)
		tInt.Add(tInt, oneInt)
		if tInt.BitLen() > 256 {
			// In case of overflow, we do not guarantee anything anyway.
			continue
		}
		montgomery_step_64(&t, q)
		tInt2 := utils.UIntarrayToInt((*[4]uint64)(&t))
		if tInt.Cmp(tInt2) != 0 {
			testing_instance.Error("montgomery_step_64 is incorrect", *tInt, *tInt2)
			break
		}

	}

	// Test add_mul_shift_64
	for i := 1; i < iterations; i++ {
		targetInt := new(big.Int).Rand(drng, bound)
		var target Uint256 = utils.BigIntToUIntArray(targetInt)

		xInt := new(big.Int).Rand(drng, bound)
		var x Uint256 = utils.BigIntToUIntArray(xInt)

		var y uint64 = drng.Uint64()
		yInt := new(big.Int).SetUint64(y)

		// compute using big.Int (result_low1, result_target2 return value/new target)
		resultInt := new(big.Int)
		resultInt.Mul(xInt, yInt)
		resultInt.Add(resultInt, targetInt)
		resultlowInt := new(big.Int).Mod(resultInt, R)
		var result_low1 uint64 = resultlowInt.Uint64()
		resultInt.Rsh(resultInt, 64)
		result_target1 := utils.BigIntToUIntArray(resultInt)

		result_low2 := add_mul_shift_64(&target, &x, y)
		if target != result_target1 {
			testing_instance.Error("add_mul_shift_64 is wrong (target)")
			break
		}
		if result_low1 != result_low2 {
			testing_instance.Error("add_mul_shift_64 is wrong (low)", result_low1, result_low2)
			break
		}
	}

}
