package bandersnatch

import (
	"math/rand"
	"testing"
)

func TestPointsOnCurve(t *testing.T) {
	var f checkfunction = func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.Flags[0].CheckFlag(Case_singular)
		// isPointOnCurve is only provided for pointTypeXTW
		var point Point_xtw = *(s.Points[0].(*Point_xtw))
		return singular != point.isPointOnCurve(), ""
	}
	make_samples1_and_run_tests(t, f, "Did not recognize points as being on the curve", pointTypeXTW, 10, 0)
	point := example_generator_xtw
	if !point.isPointOnCurve() {
		t.Fatal("Example point is not on curve")
	}
	drng := rand.New(rand.NewSource(202))

	// Modifiy each coo and check whether it is still on the curve.
	point.x.setRandomUnsafe(drng)
	if point.isPointOnCurve() {
		t.Fatal("modified example point with wrong x-coo is still on curve")
	}
	point.x.SetZero()
	if point.isPointOnCurve() {
		t.Fatal("modified example point with zeroed x-coo is still on curve")
	}

	point = example_generator_xtw
	point.y.setRandomUnsafe(drng)
	if point.isPointOnCurve() {
		t.Fatal("modified example point wth wrong y-coo is still on curve")
	}
	point.y.SetZero()
	if point.isPointOnCurve() {
		t.Fatal("modified example point with zeroed y-coo is still on curve")
	}

	point = example_generator_xtw
	point.t.setRandomUnsafe(drng)
	if point.isPointOnCurve() {
		t.Fatal("modified example point with wrong t-coo is still on curve")
	}
	point.t.SetZero()
	if point.isPointOnCurve() {
		t.Fatal("modified example point with zeroed t-coo is still on curve")
	}

	point = example_generator_xtw
	point.z.setRandomUnsafe(drng)
	if point.isPointOnCurve() {
		t.Fatal("modified example point with wrong z-coo is still on curve")
	}
	point.z.SetZero()
	if point.isPointOnCurve() {
		t.Fatal("modified example point with zeroed z-coo is still on curve")
	}
}

// Test whether *Point_xtw::Add(*Point_xtw, *Point_xtw) is consistent with addNaive_ttt. Note that *Point_xtw::Add(*Point_xtw, *Point_xtw) is checked by other tests
// for consistency with all our other variants.
func TestCompareAddAgainstNaive(t *testing.T) {
	make_samples2_and_run_tests(t, checkfun_addnaive, "Addition inconsistent with naive definition", pointTypeXTW, pointTypeXTW, 20, 0)
}

func checkfun_addnaive(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	flags := s.AnyFlags()
	if flags.CheckFlag(Case_singular) || flags.CheckFlag(Case_outside_goodgroup) {
		return true, ""
	}
	var point0 Point_xtw = *(s.Points[0].(*Point_xtw))
	var point1 Point_xtw = *(s.Points[0].(*Point_xtw))
	var result1, result2 Point_xtw
	result1.Add(&point0, &point1)
	result2.addNaive_ttt(&point0, &point1)
	if !result1.IsEqual_FullCurve(&result2) {
		return false, "Addition differs from naive defininition"
	}
	return true, ""
}

/*
OLD TESTS -- They work, but we want to have coverage displayed for the new tests only

func TestExampleIsGenerator(t *testing.T) {
	if !NeutralElement_xtw.isPointOnCurve() {
		t.Fatal("Neutral element not on curve")
	}
	if !example_generator_xtw.isPointOnCurve() {
		t.Fatal("Example point is not on curve")
	}
}

func TestRandomSampling(t *testing.T) {
	const iterations = 1000
	var drng *rand.Rand = rand.New(rand.NewSource(666))
	for i := 0; i < iterations; i++ {
		p := makeRandomPointOnCurve_t(drng)
		if !p.isPointOnCurve() {
			t.Fatal("Randomly generated curve point is actually not on curve", i)
		}
	}
}

func Test_add_xxx(t *testing.T) {
	const iterations = 1000
	var drng *rand.Rand = rand.New(rand.NewSource(666))
	var p1, p2, res1, res2 Point_xtw

	res1.add_ttt(&NeutralElement_xtw, &NeutralElement_xtw)
	if !res1.isPointOnCurve() {
		t.Fatal("0+0 not on curve for add_xxx")
	}
	if !res1.is_equal_exact_tt(&NeutralElement_xtw) {
		t.Fatal("0 + 0 != 0 on curve for add_xxx")
	}

	for i := 0; i < iterations; i++ {

		p1 = makeRandomPointOnCurve_t(drng)
		p2.add_ttt(&p1, &NeutralElement_xtw)
		if !p2.isPointOnCurve() {
			t.Fatal("x + 0 is not on curve for random x on curve in add_xxx")
		}
		if !p1.is_equal_exact_tt(&p2) {
			t.Fatal("x + 0 != x for random x in add_xxx")
		}
		p2.add_ttt(&NeutralElement_xtw, &p1)
		if !p2.isPointOnCurve() {
			t.Fatal("0 + x is not on curve for random x on curve in add_xxx")
		}
		if !p1.is_equal_exact_tt(&p2) {
			t.Fatal("0 + x != x for random x in add_xxx")
		}

		p2 = makeRandomPointOnCurve_t(drng)
		_ = p2.isPointOnCurve()
		_ = p1.isPointOnCurve()
		res1.add_ttt(&p1, &p2)
		res2.addNaive_ttt(&p1, &p2)
		if !res1.isPointOnCurve() {
			t.Fatal("Result of curve addition not on curve for add_xxx")
		}
		if !res1.is_equal_exact_tt(&res2) {
			t.Fatal("Results of curve addition do not match for add_xxx and add_xxx_naive")
		}
		res2.add_ttt(&p2, &p1)
		if !res1.is_equal_exact_tt(&res2) {
			t.Fatal("x + y != y + x for random x, y with add_xxx")
		}
	}
}

func Test_sub_xxx(t *testing.T) {
	const iterations = 10
	var drng *rand.Rand = rand.New(rand.NewSource(66354))
	var p1, p2, res1, res2 Point_xtw
	for i := 0; i < iterations; i++ {
		p1 = makeRandomPointOnCurve_t(drng)
		p2 = makeRandomPointOnCurve_t(drng)
		res1.sub_ttt(&p1, &p2)
		res2.add_ttt(&res1, &p2)
		if !res2.is_equal_exact_tt(&p1) {
			t.Fatal("(x-y)+y != x for random x,y in sub_ttt")
		}
	}
}

func Test_neg_ttt(t *testing.T) {
	const iterations = 25
	var drng *rand.Rand = rand.New(rand.NewSource(112412))
	var p1, p2, result Point_xtw
	for i := 0; i < iterations; i++ {
		switch i {
		case 0:
			p1 = NeutralElement_xtw
		case 1:
			p1 = orderTwoPoint_xtw
		case 2:
			p1 = exceptionalPoint_1_xtw
		case 3:
			p1 = exceptionalPoint_2_xtw
		default:
			p1 = makeRandomPointOnCurve_t(drng)
		}
		p2.neg_tt(&p1)
		result.add_ttt(&p1, &p2)
		if !result.is_equal_exact_tt(&NeutralElement_xtw) {
			t.Fatal("(-x) + x != 0 for random x")
		}
	}
}

func TestSingularAddition(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(666))

	var temp1 Point_xtw = makeRandomPointOnCurve_t(drng)
	var temp2, temp3, temp4, temp5 Point_xtw
	temp2.add_ttt(&temp1, &exceptionalPoint_1_xtw)
	temp3.add_ttt(&temp1, &temp2)
	temp4.add_ttt(&temp1, &temp1)
	temp5.add_ttt(&temp4, &exceptionalPoint_1_xtw)
	if temp1.IsNaP() || temp2.IsNaP() || temp4.IsNaP() || temp5.IsNaP() {
		t.Fatal("Singular point after Point addition")
	}
	if !temp3.IsNaP() {
		t.Error("Addition where singularity was expected did not result in singularity.")
	}
}

func TestPsi(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(6666))
	var temp1, temp2, temp3, result1, result2, result3 Point_xtw

	const iterations = 10

	for i := 0; i < iterations; i++ {
		temp1 = makeRandomPointOnCurve_t(drng)
		result1.computeEndomorphism_tt(&temp1)
		if !result1.isPointOnCurve() {
			t.Fatal("Psi(random point) is not on curve")
		}

		temp2 = makeRandomPointOnCurve_t(drng)
		temp3.add_ttt(&temp1, &temp2)
		result2.computeEndomorphism_tt(&temp2)
		result1.add_ttt(&result1, &result2)
		result3.computeEndomorphism_tt(&temp3)
		if !result1.is_equal_exact_tt(&result3) {
			t.Fatal("Psi is not homomorphic")
		}

		temp1 = makeRandomPointOnCurve_t(drng)
		temp2.neg_tt(&temp1)
		result1.computeEndomorphism_tt(&temp1)
		result2.computeEndomorphism_tt(&temp2)
		result2.neg_tt(&result2)
		if !result1.is_equal_exact_tt(&result2) {
			t.Fatal("Psi is not compatible with negation")
		}

		temp1.SetNeutral()
		result1.computeEndomorphism_tt(&temp1)
		if !result1.IsNeutralElement_FullCurve() {
			t.Fatal("Psi(Neutral) != Neutral")
		}

		temp1 = orderTwoPoint_xtw
		result1.computeEndomorphism_tt(&temp1)
		if !result1.IsNeutralElement_FullCurve() {
			t.Fatal("Psi(affine order-2 point) != Neutral")
		}

		temp2 = makeRandomPointOnCurve_t(drng)
		temp1.sub_ttt(&orderTwoPoint_xtw, &temp2)
		result1.computeEndomorphism_tt(&temp1)
		result2.computeEndomorphism_tt(&temp2)
		result3.add_ttt(&result1, &result2)
		if !result3.IsNeutralElement_FullCurve() {
			t.Fatal("Psi is not homomorphic for sum = affine-order-2")
		}

		result1.Endo_FullCurve(&exceptionalPoint_1_xtw)
		if !result1.is_equal_exact_tt(&orderTwoPoint_xtw) {
			t.Fatal("Psi(E1) != affine-order-2")
		}
		temp2 = makeRandomPointOnCurve_t(drng)
		temp1.sub_ttt(&exceptionalPoint_1_xtw, &temp2)
		if temp1.IsNaP() {
			t.Fatal("Unexpected singularity encountered")
		}
		result1.Endo_FullCurve(&temp1)
		result2.Endo_FullCurve(&temp2)
		temp3.add_ttt(&temp1, &temp2)
		if result1.IsNaP() || result2.IsNaP() || temp3.IsNaP() {
			t.Fatal("Unexpected singularity encountered")
		}
		if !temp3.is_equal_exact_tt(&exceptionalPoint_1_xtw) {
			t.Fatal("Associative Law fails when sum is E1")
		}
		result1.add_ttt(&result1, &result2) // requires add_xxx to be safe enough, which it is.
		if !result1.is_equal_exact_tt(&orderTwoPoint_xtw) {
			t.Fatal("Homomorphic properties of Psi unsatisfied when sum is E1")
		}
		result1.Endo_FullCurve(&exceptionalPoint_2_xtw)
		if !result1.is_equal_exact_tt(&orderTwoPoint_xtw) {
			t.Fatal("Psi(E2) != affine-order-2 point")
		}

		temp1 = makeRandomPointInSubgroup_t(drng)
		result1.computeEndomorphism_tt(&temp1)
		result2.exp_naive_xx(&temp1, EndoEigenvalue_Int)
		if !result1.is_equal_exact_tt(&result2) {
			t.Fatal("Psi does not act as multiplication by EndoEigenvalue on random point in subgroup")
		}
	}
}
*/
