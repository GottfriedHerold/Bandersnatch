package bandersnatch

import "testing"

func checkfun_torsionA(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	t := s.Points[0].Clone().(CurvePointPtrInterface)
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
	if !t.(Validateable).Validate() {
		return false, "torsionAddA does not result in valid point"
	}
	t2 := s.Points[0].Clone().(CurvePointPtrInterface)
	var A Point_xtw_full
	A.SetAffineTwoTorsion()
	t2.AddEq(&A)
	return t2.IsEqual(t), "torsionAddA does not matching Addition of point"
}

func checkfun_torsionE1(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(Case_zero_moduloA) {
		return true, "skipped"
	}
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	t := s.Points[0].Clone().(CurvePointPtrInterface)
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
	if !t.(Validateable).Validate() {
		return false, "torsionAddE1 does not result in valid point"
	}
	t2 := s.Points[0].Clone().(CurvePointPtrInterface)
	var E1 Point_xtw_full
	E1.SetE1()
	t2.AddEq(&E1)
	return t2.IsEqual(t), "torsionAddE1 does not matching Addition of point"
}

func checkfun_torsionE2(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(Case_zero_moduloA) {
		return true, "skipped"
	}
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	t := s.Points[0].Clone().(CurvePointPtrInterface)
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
	if !t.(Validateable).Validate() {
		return false, "torsionAddE2 does not result in valid point"
	}
	t2 := s.Points[0].Clone().(CurvePointPtrInterface)
	var E2 Point_xtw_full
	E2.SetE2()
	t2.AddEq(&E2)
	return t2.IsEqual(t), "torsionAddE1 does not matching Addition of point"
}

func checkfun_torsion_group(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.Points[0].CanOnlyRepresentSubgroup() {
		panic("Do not call this test function on types that can only represent subgroup elements")
	}
	if s.AnyFlags().CheckFlag(Case_singular) {
		return true, "skipped"
	}
	if !s.Points[0].CanRepresentInfinity() && s.AnyFlags().CheckFlag(Case_2torsion) {
		return true, "skipped"
	}
	t1 := s.Points[0].Clone().(CurvePointPtrInterface)
	t2 := s.Points[0].Clone().(CurvePointPtrInterface)
	t3 := s.Points[0].Clone().(CurvePointPtrInterface)
	t4 := s.Points[0].Clone().(CurvePointPtrInterface)
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

func TestTorsionAddProperties(t *testing.T) {
	for _, pointType := range allFullCurveTestPointTypes {
		pointstring := PointTypeToString(pointType)
		make_samples1_and_run_tests(t, checkfun_torsionA, "torsionAddA did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsionE1, "torsionAddE1 did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsionE2, "torsionAddE2 did not work as expected "+pointstring, pointType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_torsion_group, "torsionAdd_foo do not form a Z/2 x Z/2 group "+pointstring, pointType, 50, excludeNoPoints)
	}
}
