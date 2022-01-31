package bandersnatch

import (
	"testing"
)

/*
	Tests whether p.AddEq(x) and p.Add(x,x) are the same (and similar for SubEq, DoubleEq, NegEq, EndoEq)
*/

func checkfun_AddEq(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	singular := s.AnyFlags().CheckFlag(Case_singular)

	receiverType := getPointType(s.Points[0])

	result1 := s.Points[0].Clone().(CurvePointPtrInterface)
	result2 := makeCurvePointPtrInterface(receiverType)

	if !typeCanRepresentInfinity(receiverType) && !singular {
		var temp Point_xtw_full
		temp.Add(s.Points[0], s.Points[1])
		if temp.IsAtInfinity() {
			return true, ""
		}
	}

	result1.AddEq(s.Points[1])
	result2.Add(s.Points[0], s.Points[1])

	expected := !singular
	expected_error := singular

	if singular {
		if !result1.IsNaP() {
			return false, "AddEq did not result in NaP, even though one of the arguments was a NaP"
		}
		if !result2.IsNaP() {
			return false, "Addition did not result in NaP, even though one the arguments was a NaP"
		}
	}
	return guardForInvalidPoints(expected, expected_error, "AddEq did not match result of Add", result1.IsEqual, result2)
}

func checkfun_SubEq(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	differenceInfinite := s.AnyFlags().CheckFlag(Case_differenceInfinite)

	receiverType := getPointType(s.Points[0])

	result1 := s.Points[0].Clone().(CurvePointPtrInterface)
	result2 := makeCurvePointPtrInterface(receiverType)

	if !typeCanRepresentInfinity(receiverType) && !singular {
		var temp Point_xtw_full
		temp.Sub(s.Points[0], s.Points[1])
		if temp.IsAtInfinity() != differenceInfinite {
			return false, "Sample marked as having difference at infinity, but result of subtraction does not match"
		}
		if differenceInfinite {
			return true, ""
		}
	}

	result1.SubEq(s.Points[1])
	result2.Sub(s.Points[0], s.Points[1])

	// fmt.Println(result1.String(), result2.String())

	expected := !singular
	expected_error := singular

	if singular {
		if !result1.IsNaP() {
			return false, "SubEq did not result in NaP, even though one of the arguments was a NaP"
		}
		if !result2.IsNaP() {
			return false, "Subtraction did not result in NaP, even though one the arguements was a NaP"
		}
	}
	return guardForInvalidPoints(expected, expected_error, "SubEq did not match result of Sub", result1.IsEqual, result2)
}

func checkfun_DoubleEq(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	receiverType := getPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface)
	result2 := makeCurvePointPtrInterface(receiverType)

	result1.DoubleEq()
	result2.Double(s.Points[0])

	if singular {
		if !result1.IsNaP() || !result2.IsNaP() {
			return false, "Doubling a NaP resulted in NoN-NaP"
		}
	} else {
		if result1.IsNaP() || result2.IsNaP() {
			return false, "Doubling a Non-NaP resulted in a NaP"
		}
	}
	expected := !singular
	return guardForInvalidPoints(expected, singular, "Double and DoubleEq do not match", result1.IsEqual, result2)
}

func checkfun_NegEq(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	receiverType := getPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface)
	result2 := makeCurvePointPtrInterface(receiverType)

	result1.NegEq()
	result2.Neg(s.Points[0])

	if singular {
		if !result1.IsNaP() || !result2.IsNaP() {
			return false, "Negating a NaP resulted in NoN-NaP"
		}
	} else {
		if result1.IsNaP() || result2.IsNaP() {
			return false, "Negating a Non-NaP resulted in a NaP"
		}
	}
	expected := !singular
	return guardForInvalidPoints(expected, singular, "Neg and NegEq do not match", result1.IsEqual, result2)
}

func checkfun_EndoEq(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	receiverType := getPointType(s.Points[0])
	result1 := s.Points[0].Clone().(CurvePointPtrInterface)
	result2 := makeCurvePointPtrInterface(receiverType)

	result1.EndoEq()
	result2.Endo(s.Points[0])

	if singular {
		if !result1.IsNaP() || !result2.IsNaP() {
			return false, "Endo(NaP) resulted in NoN-NaP"
		}
	} else {
		if result1.IsNaP() || result2.IsNaP() {
			return false, "Endo(Non-NaP) resulted in a NaP"
		}
	}
	expected := !singular
	return guardForInvalidPoints(expected, singular, "Endo and EndoEq do not match", result1.IsEqual, result2)
}

func test_CurvePointPtrInterface_EqVariants(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := pointTypeToString(receiverType)
	make_samples2_and_run_tests(t, checkfun_AddEq, "AddEq did not behave as expected "+point_string, receiverType, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_SubEq, "SubEq did not behave as expected "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type2 := range allTestPointTypes {
		if type2 == receiverType {
			continue
		}
		if typeCanOnlyRepresentSubgroup(receiverType) && !typeCanOnlyRepresentSubgroup(type2) {
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
	test_CurvePointPtrInterface_EqVariants(t, pointTypeXTWSubgroup, excludeNoPoints)
	test_CurvePointPtrInterface_EqVariants(t, pointTypeXTWFull, excludeNoPoints)
}

func TestEqVariantsForAXTW(t *testing.T) {
	test_CurvePointPtrInterface_EqVariants(t, pointTypeAXTWSubgroup, excludeNoPoints)
	test_CurvePointPtrInterface_EqVariants(t, pointTypeAXTWFull, excludeNoPoints)
}

func TestEqVariantsForEFGH(t *testing.T) {
	test_CurvePointPtrInterface_EqVariants(t, pointTypeEFGHSubgroup, excludeNoPoints)
	test_CurvePointPtrInterface_EqVariants(t, pointTypeEFGHFull, excludeNoPoints)
}
