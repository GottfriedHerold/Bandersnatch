package bandersnatch

import (
	"math/rand"
	"testing"
)

// test specific to Point_xtw go here. Note that most tests are contained in generic tests from curve_point_test_*_test.go files

// Old test, somewhat redundant with general test for Validate().
// We just keep it around.

func TestPointsOnCurve(t *testing.T) {
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

// Test whether Point_xtw.Add(*Point_xtw, *Point_xtw) is consistent with addNaive_ttt.
// Consistency of addition routines for various point types against each other are tested by the general framework.
func TestCompareAddAgainstNaive(t *testing.T) {
	var checkfun_addnaive = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		flags := s.AnyFlags()
		if flags.CheckFlag(PointFlagNAP) {
			return true, "skipped"
		}
		var point0 Point_xtw_subgroup
		point0.SetFrom(s.Points[0])
		var point1 Point_xtw_subgroup
		point1.SetFrom(s.Points[1])
		var result1, result2 Point_xtw_subgroup
		result1.Add(&point0, &point1)
		result2.addNaive_ttt(&point0.point_xtw_base, &point1.point_xtw_base)
		if !result1.IsEqual(&result2) {
			return false, "Addition differs from naive defininition"
		}
		return true, ""
	}
	make_samples2_and_run_tests(t, checkfun_addnaive, "Addition inconsistent with naive definition", pointTypeXTWSubgroup, pointTypeXTWSubgroup, 20, 0)
}
