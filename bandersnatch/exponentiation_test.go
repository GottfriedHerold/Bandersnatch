//go:build ignore

package bandersnatch

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func TestSimpleExponentiation(t *testing.T) {
	const iterations = 10
	var temp1, temp2, temp3, temp4 Point_xtw_subgroup
	temp1.exp_naive_xx(&example_generator_xtw, GroupOrder_Int)
	if !temp1.IsNeutralElement() {
		t.Fatal("Either naive exponentiation is wrong or example point not in subgroup")
	}
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	var exp1 = big.NewInt(0)
	var exp2 = big.NewInt(1)
	var exp3 = big.NewInt(-1)

	temp1.sampleRandomUnsafe(drng)
	temp2.exp_naive_xx(&temp1.point_xtw_base, exp2) // exponent is 1
	if !temp2.IsEqual(&temp1) {
		t.Error("1 * P != P for naive exponentiation")
	}
	temp2.exp_naive_xx(&temp1.point_xtw_base, exp1) // exponent is 0
	if !temp2.IsNeutralElement() {
		t.Error("0 * P != Neutral element for naive exponentiation")
	}
	temp2.exp_naive_xx(&temp1.point_xtw_base, exp3)
	temp1.NegEq()
	if !temp1.IsEqual(&temp2) {
		t.Error("-1 * P != -P for naive exponentiation")
	}

	var p1, p2, p3 Point_xtw_subgroup
	for i := 0; i < iterations; i++ {
		p1.sampleRandomUnsafe(drng)
		p2.sampleRandomUnsafe(drng)
		p3.Add(&p1, &p2)
		exp1.Rand(drng, CurveOrder_Int)
		exp2.Rand(drng, CurveOrder_Int)
		exp3.Add(exp1, exp2)
		temp1.exp_naive_xx(&p1.point_xtw_base, exp1)
		temp2.exp_naive_xx(&p2.point_xtw_base, exp1)
		temp3.exp_naive_xx(&p3.point_xtw_base, exp1)
		temp4.Add(&temp1, &temp2)
		if !temp3.IsEqual(&temp4) {
			t.Error("a * (P+Q) != a*P + a*Q for naive exponentiation")
		}
		temp2.exp_naive_xx(&p1.point_xtw_base, exp2)
		temp3.exp_naive_xx(&p1.point_xtw_base, exp3)
		temp4.Add(&temp1, &temp2)
		if !temp3.IsEqual(&temp4) {
			t.Error("(a+b) * P != a*P + b*P for naive exponentiation")
		}
	}
}

func TestSlidingWindowExponentiation(t *testing.T) {
	// check equality with naive implementation

	var checkfun_equal_naive checkfunction = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(PointFlagNAP)
		if singular {
			return true, "skipped"
		}
		testutils.Assert(s.Points[0].CanOnlyRepresentSubgroup())

		const iterations = 500
		var drng *rand.Rand = rand.New(rand.NewSource(1024))
		var exponent *big.Int = big.NewInt(0)
		var EVPlusOne *big.Int = big.NewInt(1)
		EVPlusOne.Add(EVPlusOne, EndomorphismEigenvalue_Int)

		var P1, P2 Point_xtw_subgroup
		P1.SetFrom(s.Points[0])
		P2.SetFrom(s.Points[0])
		for i := 0; i < iterations; i++ {
			// These numbers are "special" in the GLV - decomposition only has one component
			switch {
			case i < 128:
				exponent.SetInt64(int64(i - 64))
			case i < 256:
				exponent.SetInt64(int64(i - 64 - 128))
				exponent.Mul(exponent, EndomorphismEigenvalue_Int)
			case i < 384:
				exponent.SetInt64(int64(i - 64 - 256))
				exponent.Mul(exponent, EVPlusOne)
			default:
				exponent.Rand(drng, CurveOrder_Int)
			}
			var resultNaive Point_xtw_subgroup
			var resultSlidingWindow Point_efgh_subgroup
			resultNaive.exp_naive_xx(&P1.point_xtw_base, exponent)
			var exponent_ScalarField Exponent
			exponent_ScalarField.SetBigInt(exponent)
			resultSlidingWindow = exponentiate_slidingWindow(&P2, &exponent_ScalarField)
			if !resultNaive.IsEqual(&resultSlidingWindow) {
				return false, "expnaive and sliding window results differ"
			}
		}
		return true, ""
	}
	make_samples1_and_run_tests(t, checkfun_equal_naive, "comparison of square-and-multiply and sliding window", pointTypeXTWSubgroup, 30, PointFlagNAP)
}
