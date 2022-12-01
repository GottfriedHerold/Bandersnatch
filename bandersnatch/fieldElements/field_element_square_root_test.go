package fieldElements

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestSqrtHelperFunctions(t *testing.T) {
	t.Run("Validity of Constant", testSqrt_Constants)
	t.Run("Special-cased exponentiation algorithms", testSqrt_computeCertainPowers)
	t.Run("Dlog in small-order subgroup of roots of unity", testSqrt_NegDlogInSmallSubgroup)
	t.Run("Inverse square root for 2^32th roots of unity", testSqrt_InvSqrtEq)
}

func testSqrt_Constants(t *testing.T) {
	var exponent Uint256
	exponent.SetOne()
	for i := 0; i <= BaseField2Adicity; i++ { // Note <= here
		testutils.FatalUnless(t, sqrtAlg_OrderAsDyadicRootOfUnity(&sqrtPrecomp_PrimitiveDyadicRoots[i]) == BaseField2Adicity-i, "")
		testutils.FatalUnless(t, sqrtPrecomp_PrimitiveDyadicRoots[i].isNormalized(), "precomputed constant not normalized")
		var target feType_SquareRoot
		target.Exp(&DyadicRootOfUnity_fe, &exponent)
		testutils.FatalUnless(t, target.IsEqual(&sqrtPrecomp_PrimitiveDyadicRoots[i]), "precomputed roots of unity are wrong")
		exponent.Add(&exponent, &exponent)
	}
}

func testSqrt_computeCertainPowers(t *testing.T) {
	const num = 100
	var xs []feType_SquareRoot = GetPrecomputedFieldElements[feType_SquareRoot](10001, num)
	for i, x := range xs {
		if i == 0 {
			x.SetZero()
		}
		if i == 1 {
			x.SetOne()
		}
		var sqrtCand1, sqrtCand2, rootOfUnity1, rootOfUnity2, rootOfUnity3, xCopy feType_SquareRoot
		xCopy = x
		x.sqrtAlg_ComputeRelevantPowers(&sqrtCand1, &rootOfUnity1)
		rootOfUnity2.sqrtAlg_ExpOddOrder(&x)
		testutils.FatalUnless(t, x.IsEqual(&xCopy), "square root helper exponentiations modify receiver")
		testutils.FatalUnless(t, rootOfUnity1.IsEqual(&rootOfUnity2), "squre root helper exponentations disagree")
		sqrtCand2.Exp(&x, &tonelliShanksExponent_uint256)
		rootOfUnity3.Exp(&x, &BaseFieldMultiplicateOddOrder_uint256)
		testutils.FatalUnless(t, rootOfUnity1.IsEqual(&rootOfUnity3), "square root helper exponentiation did not exponentiate into dyadic roots of unity correctly")
		testutils.FatalUnless(t, sqrtCand1.IsEqual(&sqrtCand2), "square root helper exponentiation did not exponentiate candidate square root mod 2^32th root of unity correctly")
	}
}

func testSqrt_NegDlogInSmallSubgroup(t *testing.T) {
	const numRoots = 1 << sqrtParam_BlockSize
	var target feType_SquareRoot
	var j Uint256
	for i := uint(0); i < numRoots; i++ {
		j.SetUint64(uint64(i))
		target.Exp(&sqrtPrecomp_ReconstructionDyadicRoot, &j)
		negLogRoot := sqrtAlg_NegDlogInSmallDyadicSubgroup(&target)
		testutils.FatalUnless(t, negLogRoot < 1<<sqrtParam_BlockSize, "dlog for dyadic roots of unity gives anwers in the wrong range")
		testutils.FatalUnless(t, (negLogRoot+i)%(1<<sqrtParam_BlockSize) == 0, "dlog for dyadic roots of unity does not work as expected")
	}
}

func testSqrt_GetPrecomputedBlock(t *testing.T) {
	for realOrder := 0; realOrder < 32; realOrder++ {
		if realOrder%sqrtParam_BlockSize != 0 {
			continue
		}
		order := realOrder / sqrtParam_BlockSize
		for multiplier := 0; multiplier < (1 << sqrtParam_BlockSize); multiplier++ {
			var target1, target2 feType_SquareRoot
			sqrtAlg_GetPrecomputedRootOfUnity(&target1, multiplier, uint(order))
			realExponent := uint64(multiplier << realOrder)
			var exponentUint Uint256
			exponentUint.SetUint64(realExponent)
			target2.Exp(&dyadicRootOfUnity_fe, &exponentUint)
			testutils.FatalUnless(t, target1.IsEqual(&target2), "sqrtAlg_getPrecomputedRootOfUnity differs from Exp")
		}
	}
}

func testSqrt_InvSqrtEq(t *testing.T) {
	const num = 1000
	var exponents []uint64 = CachedUint64.GetElements(10001, num)
	var exponent Uint256
	var testFE feType_SquareRoot
	for _, exp64 := range exponents {
		exp64 &= (1<<32 - 1)
		exponent.SetUint64(exp64)
		testFE.Exp(&DyadicRootOfUnity_fe, &exponent)
		testFE_copy := testFE
		isSquare := testFE.invSqrtEqDyadic()
		testutils.FatalUnless(t, isSquare == (exp64&1 == 0), "inverse sqaure root algorithm does not correctly recognize squares")
		if !isSquare {
			testutils.FatalUnless(t, testFE.IsEqual(&testFE_copy), "inverse dyandic square root modifies argument on failure")
			continue
		}
		testFE.SquareEq()
		testFE.MulEq(&testFE_copy)
		testutils.FatalUnless(t, testFE.IsOne(), "inverse square root for dyadic roots does not give inverse square root")
	}
}
