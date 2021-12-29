package bandersnatch

import "testing"

/*
	Tests whether p.AddEq(x) and p.Add(x,x) are the same (and similar for SubEq, DoubleEq, NegEq, EndoEq)
*/

func checkfun_AddEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	singular_sum := s.AnyFlags().CheckFlag(Case_differenceInfinite)
	if singular_sum && singular {
		panic("TestSample makes no sense")
	}

	receiverType := GetPointType(s.Points[0])

	result1 := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

	if !canRepresentInfinity(receiverType) {
		var temp Point_xtw
		temp.Add(s.Points[0], s.Points[1])
		if temp.IsAtInfinity() {
			return true, ""
		}
	}

	result1.AddEq(s.Points[1])
	result2.Add(s.Points[0], s.Points[1])

	expected := !singular
	expected_error := singular

	if singular_sum {
		if result1.IsNaP() || result2.IsNaP() {
			expected = false
			expected_error = true
		}
	}
	if singular {
		if !result1.IsNaP() {
			return false, "AddEq did not result in NaP, even though one of the arguments was a NaP"
		}
		if !result2.IsNaP() {
			return false, "Addition did not result in NaP, even though one the arguements was a NaP"
		}
	}
	return guardForInvalidPoints(expected, expected_error, "AddEq did not match result of Add", result1.IsEqual_FullCurve, result2)
}

func checkfun_SubEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	singular := s.AnyFlags().CheckFlag(Case_singular)

	allowed_error := s.AnyFlags().CheckFlag(Case_outside_goodgroup)

	receiverType := GetPointType(s.Points[0])

	result1 := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

	if !canRepresentInfinity(receiverType) {
		var temp Point_xtw
		temp.Sub(s.Points[0], s.Points[1])
		if temp.IsAtInfinity() {
			return true, ""
		}
	}

	result1.SubEq(s.Points[1])
	result2.Sub(s.Points[0], s.Points[1])

	expected := !singular
	expected_error := singular

	if allowed_error {
		if result1.IsNaP() || result2.IsNaP() {
			expected = false
			expected_error = true
		}
	}
	if singular {
		if !result1.IsNaP() {
			return false, "AddEq did not result in NaP, even though one of the arguments was a NaP"
		}
		if !result2.IsNaP() {
			return false, "Addition did not result in NaP, even though one the arguements was a NaP"
		}
	}
	return guardForInvalidPoints(expected, expected_error, "AddEq did not match result of Add", result1.IsEqual_FullCurve, result2)
}

func checkfun_DoubleEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	receiverType := GetPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

	result1.DoubleEq()
	result2.Double(s.Points[0])

	if !(singular == result1.IsNaP() && singular == result2.IsNaP()) {
		return false, "Doubling resulted in unexpected NaP"
	}

	expected := !singular
	return guardForInvalidPoints(expected, singular, "Double and DoubleEq do not match", result1.IsEqual_FullCurve, result2)
}

func checkfun_NegEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	receiverType := GetPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

	result1.NegEq()
	result2.Neg(s.Points[0])

	if !(singular == result1.IsNaP() && singular == result2.IsNaP()) {
		return false, "Negation resulted in unexpected NaP"
	}

	expected := !singular
	return guardForInvalidPoints(expected, singular, "Neg and NegEq do not match", result1.IsEqual_FullCurve, result2)
}

func checkfun_EndoEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	infinite := s.AnyFlags().CheckFlag(Case_infinite)
	receiverType := GetPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

	result1.EndoEq()
	result2.Endo(s.Points[0])

	// Endo may fail at points at infinity. We know that the correct answer is A, so we just check for that directly.
	if infinite {
		if !result1.IsNaP() && !result1.IsEqual_FullCurve(&orderTwoPoint_xtw) {
			return false, "EndoEq on infinite point gave wrong result"
		}
		if !result2.IsNaP() && !result2.IsEqual_FullCurve(&orderTwoPoint_xtw) {
			return false, "Endo on infinite point gave wrong result"
		}
		return true, ""
	}

	if !(singular == result1.IsNaP() && singular == result2.IsNaP()) {
		return false, "Doubling resulted in unexpected NaP"
	}

	expected := !singular

	return guardForInvalidPoints(expected, singular, "Double and DoubleEq do not match", result1.IsEqual_FullCurve, result2)
}

func test_CurvePointPtrInterface_EqVariants(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples2_and_run_tests(t, checkfun_AddEq, "AddEq did not behave as expected "+point_string, receiverType, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_SubEq, "SubEq did not behave as expected "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type2 := range allTestPointTypes {
		if type2 == receiverType {
			continue
		}
		make_samples2_and_run_tests(t, checkfun_AddEq, "AddEq did not behave as expected "+point_string, receiverType, type2, 10, excludedFlags)
		make_samples2_and_run_tests(t, checkfun_SubEq, "SubEq did not behave as expected "+point_string, receiverType, type2, 10, excludedFlags)
	}
	make_samples1_and_run_tests(t, checkfun_DoubleEq, "DoubleEq did not behave as expected "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_NegEq, "NegEq did not behave as expected "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_EndoEq, "EndoEq did not behave as expected "+point_string, receiverType, 10, excludedFlags)
}

func TestEqVariantsForXTW(t *testing.T) {
	test_CurvePointPtrInterface_EqVariants(t, pointTypeXTW, 0)
}

func TestEqVariantsForAXTW(t *testing.T) {
	test_CurvePointPtrInterface_EqVariants(t, pointTypeAXTW, 0)
}

func TestEqVariantsForEFGH(t *testing.T) {
	test_CurvePointPtrInterface_EqVariants(t, pointTypeEFGH, 0)
}
