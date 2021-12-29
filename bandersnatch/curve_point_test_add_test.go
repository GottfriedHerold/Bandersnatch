package bandersnatch

import (
	"strconv"
	"testing"
)

// check whether P+Q == Q+P
func make_checkfun_addition_commutes(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result1.Add(s.Points[0], s.Points[1])
		result2.Add(s.Points[1], s.Points[0])

		var expected, got1, got2 bool
		expected = !singular
		if wasInvalidPointEncountered(func() { got1 = result1.IsEqual(result2) }) != singular {
			return false, "comparison of P+Q =? Q+P with NaPs involved did not trigger error handler"
		}
		if wasInvalidPointEncountered(func() { got2 = result1.IsEqual_FullCurve(result2) }) != singular {
			return false, "exact Comparison of P+Q =? Q+P with NaPs involved did not trigger error handler"
		}
		return expected == got1 && expected == got2, "Curve point addition not commutative"
	}
	return
}

// ensure that P + neutral element == P for P finite curve point (for infinite P, the addition law does not work for P + neutral element)
func make_checkfun_addition_of_zero(receiverType PointType, zeroType PointType) (returned_function checkfunction) {
	var zero CurvePointPtrInterface = MakeCurvePointPtrInterfaceFromType(zeroType)
	zero.SetNeutral()
	returned_function = func(s TestSample) (bool, string) {
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		if s.AnyFlags().CheckFlag(Case_infinite) {
			return true, "" // This case is skipped.
		}
		result := MakeCurvePointPtrInterfaceFromType(receiverType)
		result.Add(s.Points[0], zero)
		var expected, got1, got2 bool
		expected = !singular
		if wasInvalidPointEncountered(func() { got1 = result.IsEqual(s.Points[0]) }) != singular {
			return false, "comparison of P + neutral element =? P with NaP P did not trigger error handler"
		}
		if wasInvalidPointEncountered(func() { got2 = result.IsEqual_FullCurve(s.Points[0]) }) != singular {
			return false, "exact omparison of P + neutral element =? P with NaP P did not trigger error handler"
		}
		return expected == got1 && expected == got2, "P + 0 != P"
	}
	return
}

// Checks that Neg results in an additive inverse
func make_checkfun_negative(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		negative_of_point := MakeCurvePointPtrInterfaceFromType(receiverType)
		sum := MakeCurvePointPtrInterfaceFromType(receiverType)
		negative_of_point.Neg(s.Points[0])
		if singular != negative_of_point.IsNaP() {
			return false, "Taking negative of NaP did not result in same NaP status. Was expecting " + strconv.FormatBool(singular) + " but got " + strconv.FormatBool(negative_of_point.IsNaP())
		}
		sum.Add(s.Points[0], negative_of_point)
		expected := !singular
		var got bool
		if wasInvalidPointEncountered(func() { got = sum.IsNeutralElement_FullCurve() }) != singular {
			return false, "comparing P + (-P) =? neutral with P NaP did not trigger error handler"
		}
		return expected == got, "P + (-P) != neutral"
	}
	return
}

// Checks that subtraction is compatible with addition, i.e. (P-Q)+Q == P and P-Q == P + (-Q)
func make_checkfun_subtraction(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)

		// If the points in the sample are outside of the subgroup, we might hit the singular cases of addition/subtraction.
		// For addition, we have a Case - flag, but not for subtraction. We just skip the test.
		if s.AnyFlags().CheckFlag(Case_outside_goodgroup) {
			return true, ""
		}

		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		result_of_subtraction := MakeCurvePointPtrInterfaceFromType(receiverType)
		negative_of_point := MakeCurvePointPtrInterfaceFromType(receiverType)
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

		result_of_subtraction.Sub(s.Points[0], s.Points[1])
		result1.Add(result_of_subtraction, s.Points[1])
		var got bool
		var expected bool = !singular
		if wasInvalidPointEncountered(func() { got = result1.IsEqual_FullCurve(s.Points[0]) }) != singular {
			return false, "Wrong NaP behaviour when checking (P-Q) + Q ?= P"
		}
		if got != expected {
			return false, "(P-Q) + Q != P"
		}

		// Check that P - Q == P + (-Q)
		negative_of_point.Neg(s.Points[1])
		result2.Add(s.Points[0], negative_of_point)
		if wasInvalidPointEncountered(func() { got = result2.IsEqual_FullCurve(result_of_subtraction) }) != singular {
			return false, "Wrong NaP behaviour when checking P - Q ?= P + (-Q)"
		}
		if got != expected {
			return false, "P - Q != P + (-Q)"
		}
		return true, ""
	}
	return
}

// This tests whether doubling works as intended
func make_checkfun_doubling(receiverType PointType) checkfunction {
	return func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result1.Double(s.Points[0])
		if result1.IsNaP() != singular {
			return false, "Point doubling resulted in different NaP status"
		}
		if singular {
			return true, ""
		}
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2.Add(s.Points[0], s.Points[0])
		if !result1.IsEqual_FullCurve(result2) {
			return false, "Point doubling differs from computing P + P"
		}
		return true, ""
	}
}

// This function checks the associative law on point addition.
// Note that we assume that our testsamples do not contain triples where exceptional cases for the addition laws occur.
// (This is why the generator for testsamples that contain triples only produces random output)
func checkfun_associative_law(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(3)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	result1 := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result2 := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))

	result1.Add(s.Points[0], s.Points[1])
	result1.Add(result1, s.Points[2])
	result2.Add(s.Points[1], s.Points[2])
	result2.Add(s.Points[0], result2)

	var expected bool = !singular
	return guardForInvalidPoints(expected, singular, "Test for associative law failed", result1.IsEqual_FullCurve, result2)
}

func test_addition_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	var type1, type2 PointType
	make_samples2_and_run_tests(t, make_checkfun_addition_commutes(receiverType), "Addition did not commute for "+point_string, receiverType, receiverType, 10, excludedFlags|Case_differenceInfinite)
	for _, type1 = range allTestPointTypes {
		for _, type2 = range allTestPointTypes {
			make_samples2_and_run_tests(t, make_checkfun_addition_commutes(receiverType), "Addition did not commute for "+point_string, type1, type2, 10, excludedFlags|Case_differenceInfinite|Case_infinite|Case_outside_goodgroup)
		}
	}

	for _, type1 = range allTestPointTypes {
		for _, type2 = range allTestPointTypes {
			make_samples1_and_run_tests(t, make_checkfun_addition_of_zero(receiverType, type1), "Addition of neutral changes point for"+point_string, type2, 10, excludedFlags|Case_infinite) // Infinite + Neutral will cause a NaP
		}
	}

	make_samples1_and_run_tests(t, make_checkfun_negative(receiverType), "Negating points did not work as expected"+point_string, receiverType, 10, excludedFlags)
	for _, type1 = range allTestPointTypes {
		make_samples1_and_run_tests(t, make_checkfun_negative(receiverType), "Negating points did not work as expected"+point_string, type1, 10, excludedFlags|Case_infinite)
	}

	make_samples2_and_run_tests(t, make_checkfun_subtraction(receiverType), "Subtraction did not work as expected"+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type1 = range allTestPointTypes {
		for _, type2 = range allTestPointTypes {
			make_samples2_and_run_tests(t, make_checkfun_subtraction(receiverType), "Subtraction did not work as expected"+point_string, type1, type2, 10, excludedFlags|Case_infinite)
		}
	}

	make_samples1_and_run_tests(t, make_checkfun_doubling(receiverType), "Doubling does not work as intended "+point_string, receiverType, 10, excludedFlags)
	for _, type1 = range allTestPointTypes {
		make_samples1_and_run_tests(t, make_checkfun_doubling(receiverType), "Doubling does not work as intended "+point_string, type1, 10, excludedFlags|Case_infinite)
	}

	for _, type1 = range allTestPointTypes {
		for _, type2 = range allTestPointTypes {
			samples := MakeTestSamples3(5, receiverType, type1, type2, excludedFlags|Case_outside_goodgroup)
			run_tests_on_samples(checkfun_associative_law, t, samples, "Associative law does not hold "+point_string+" "+PointTypeToString(type1)+" "+PointTypeToString(type2))
		}
	}
}

func TestAdditionPropertiesForXTW(t *testing.T) {
	test_addition_properties(t, pointTypeXTW, 0)
}

func TestAdditionPropertiesForAXTW(t *testing.T) {
	test_addition_properties(t, pointTypeAXTW, Case_infinite)
}

func TestAdditionPropertiesForEFGH(t *testing.T) {
	test_addition_properties(t, pointTypeEFGH, 0)
}
