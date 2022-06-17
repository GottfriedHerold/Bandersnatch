package bandersnatch

import "testing"

// This file contains test for the torsionAddA, torstionAddE1, torsionAddE2 methods of curve points.
// (i.e. for points satisfying the torsionAdder interface)

// checkfun_torsionA ensures compatibility of torsionAddA with Add(., A)
func checkfun_torsionA(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
	t := s.Points[0].Clone()
	t.(torsionAdder).torsionAddA()
	if singular {
		if !t.IsNaP() {
			return false, "torsionAddA does not preserve NaPs"
		} else {
			return true, ""
		}
	}
	if t.IsNaP() {
		return false, "torsionAddA of non-NaP resulted in NaP"
	}
	if !t.(validateable).Validate() {
		return false, "torsionAddA does not result in valid point"
	}
	t2 := s.Points[0].Clone()
	var A Point_xtw_full
	A.SetAffineTwoTorsion()
	t2.AddEq(&A)
	return t2.IsEqual(t), "torsionAddA does not matching Addition of point"
}

// checkfun_torsionE1 ensures compatibility of torsionAddE1 with Add(., E1)
func checkfun_torsionE1(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(PointFlag_zeroModuloA) {
		return true, "skipped"
	}
	var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
	t := s.Points[0].Clone()
	t.(torsionAdder).torsionAddE1()
	if singular {
		if !t.IsNaP() {
			return false, "torsionAddE1 does not preserve NaPs"
		} else {
			return true, ""
		}
	}
	if t.IsNaP() {
		return false, "torsionAddE1 of non-NaP resulted in NaP"
	}
	if !t.(validateable).Validate() {
		return false, "torsionAddE1 does not result in valid point"
	}
	t2 := s.Points[0].Clone()
	var E1 Point_xtw_full
	E1.SetE1()
	t2.AddEq(&E1)
	return t2.IsEqual(t), "torsionAddE1 does not matching Addition of point"
}

// checkfun_torsionE2 ensures compatibility of torsionAddE2 with Add(., E2)
func checkfun_torsionE2(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(PointFlag_zeroModuloA) {
		return true, "skipped"
	}
	var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
	t := s.Points[0].Clone()
	t.(torsionAdder).torsionAddE2()
	if singular {
		if !t.IsNaP() {
			return false, "torsionAddE2 does not preserve NaPs"
		} else {
			return true, ""
		}
	}
	if t.IsNaP() {
		return false, "torsionAddE2 of non-NaP resulted in NaP"
	}
	if !t.(validateable).Validate() {
		return false, "torsionAddE2 does not result in valid point"
	}
	t2 := s.Points[0].Clone()
	var E2 Point_xtw_full
	E2.SetE2()
	t2.AddEq(&E2)
	return t2.IsEqual(t), "torsionAddE1 does not matching Addition of point"
}

// checkfun_torsion_group ensures that the action of torsionAddA, torsionAddE1, torsionAddE2 acts like the action of elements from Z/2 x Z/2
func checkfun_torsion_group(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if s.AnyFlags().CheckFlag(PointFlagNAP) {
		return true, "skipped"
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(PointFlag_2torsion) {
		return true, "skipped"
	}
	t1 := s.Points[0].Clone()
	t2 := s.Points[0].Clone()
	t3 := s.Points[0].Clone()
	t4 := s.Points[0].Clone()
	t1.(torsionAdder).torsionAddA()
	t1.(torsionAdder).torsionAddA()
	t2.(torsionAdder).torsionAddE1()
	t2.(torsionAdder).torsionAddE1()
	t3.(torsionAdder).torsionAddE2()
	t3.(torsionAdder).torsionAddE2()
	t4.(torsionAdder).torsionAddA()
	t4.(torsionAdder).torsionAddE1()
	t4.(torsionAdder).torsionAddE2()
	if !t1.IsEqual(s.Points[0]) {
		return false, "torsionAddA is no involution"
	}
	if !t2.IsEqual(s.Points[0]) {
		return false, "torsionAddE1 is no involution"
	}
	if !t3.IsEqual(s.Points[0]) {
		return false, "torsionAddE2 is no involution"
	}
	// Sanity check:
	if !t4.IsEqual(s.Points[0]) {
		return false, "torsionAddA, E1, E2 do not compose to identity"
	}
	return true, "Ok"
}

// TestTorsionAddProperties verifies properties of the torsionAdder interface for all curve point types from allFullCurveTestPointTypes.
// This assumes that all such point types satisfy the torsionAdder interface (this is checked in curve_point_test.go).
func TestTorsionAddProperties(t *testing.T) {
	for _, pointType := range allFullCurveTestPointTypes {
		pointstring := pointTypeToString(pointType)
		make_samples1_and_run_tests(t, checkfun_torsionA, "torsionAddA did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsionE1, "torsionAddE1 did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsionE2, "torsionAddE2 did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsion_group, "torsionAdd_foo do not form a Z/2 x Z/2 group "+pointstring, pointType, 50, excludeNoPoints)
	}
}
