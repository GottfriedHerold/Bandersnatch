package bandersnatch

import (
	"strconv"
	"testing"
)

// check whether P+Q == Q+P
func make_checkfun_addition_commutes(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		var sumInfinite = s.AnyFlags().CheckFlag(Case_sumInfinite)
		if sumInfinite && !typeCanRepresentInfinity(receiverType) {
			return true, "" // skip test in that case
		}
		result1 := makeCurvePointPtrInterface(receiverType)
		result2 := makeCurvePointPtrInterface(receiverType)
		result1.Add(s.Points[0], s.Points[1])
		result2.Add(s.Points[1], s.Points[0])

		if sumInfinite && !result1.IsAtInfinity() {
			return false, "P+Q did not result in a point at infinity, even though the sample says it should"
		}
		if sumInfinite && !result2.IsAtInfinity() {
			return false, "Q+P did not result in a point at infinity, even though the sample says it should"
		}

		var expected, got bool
		expected = !singular
		if wasInvalidPointEncountered(func() { got = result1.IsEqual(result2) }) != singular {
			return false, "comparison of P+Q =? Q+P with NaPs involved did not trigger error handler"
		}
		return expected == got, "Curve point addition not commutative"
	}
	return
}

// ensure that P + neutral element == P
func make_checkfun_addition_of_zero(receiverType PointType, zeroType PointType) (returned_function checkfunction) {
	var zero CurvePointPtrInterface = makeCurvePointPtrInterface(zeroType)
	zero.SetNeutral()
	returned_function = func(s *TestSample) (bool, string) {
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		infinite := s.AnyFlags().CheckFlag(Case_infinite)
		if infinite && !typeCanRepresentInfinity(receiverType) {
			return true, "" // This case is skipped.
		}
		result := makeCurvePointPtrInterface(receiverType)
		result.Add(s.Points[0], zero)

		if infinite && !result.IsAtInfinity() {
			return false, "Point at infinity + Neutral Element is not at infinity"
		}

		var expected, got bool
		expected = !singular
		if wasInvalidPointEncountered(func() { got = result.IsEqual(s.Points[0]) }) != singular {
			return false, "comparison of P + neutral element =? P with NaP P did not trigger error handler"
		}
		return expected == got, "P + Neutral Element != P"
	}
	return
}

// Checks that Neg results in an additive inverse
func make_checkfun_negative(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		if !typeCanRepresentInfinity(receiverType) && s.AnyFlags().CheckFlag(Case_infinite) {
			return true, ""
		}
		negative_of_point := makeCurvePointPtrInterface(receiverType)
		sum := makeCurvePointPtrInterface(receiverType)
		negative_of_point.Neg(s.Points[0])
		if singular != negative_of_point.IsNaP() {
			return false, "Taking negative of NaP did not result in same NaP status. Was expecting " + strconv.FormatBool(singular) + " but got " + strconv.FormatBool(negative_of_point.IsNaP())
		}
		sum.Add(s.Points[0], negative_of_point)
		expected := !singular
		var got bool
		if wasInvalidPointEncountered(func() { got = sum.IsNeutralElement() }) != singular {
			return false, "comparing P + (-P) =? neutral with P NaP did not trigger error handler"
		}
		return expected == got, "P + (-P) != neutral"
	}
	return
}

// Checks that subtraction is compatible with addition, i.e. (P-Q)+Q == P and P-Q == P + (-Q)
func make_checkfun_subtraction(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)

		// If the points in the sample are outside of the subgroup, we might hit the singular cases of addition/subtraction.
		// For addition, we have a Case - flag, but not for subtraction. We just skip the test.
		// NOT ANYMORE
		/*
			if s.AnyFlags().CheckFlag(Case_outside_goodgroup) {
				return true, ""
			}
		*/
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		var differenceInfinite bool = s.AnyFlags().CheckFlag(Case_differenceInfinite)
		result_of_subtraction := makeCurvePointPtrInterface(receiverType)
		negative_of_point := makeCurvePointPtrInterface(receiverType)
		result1 := makeCurvePointPtrInterface(receiverType)
		result2 := makeCurvePointPtrInterface(receiverType)

		if differenceInfinite && !typeCanRepresentInfinity(receiverType) {
			return true, ""
		}

		result_of_subtraction.Sub(s.Points[0], s.Points[1])

		if !singular && differenceInfinite && !result_of_subtraction.IsAtInfinity() {
			return false, "Expected result of subtraction to be infinity, but did not get it"
		}
		if !singular && !differenceInfinite && result_of_subtraction.IsAtInfinity() {
			return false, "Subtraction resulted in unexpected infinity"
		}

		if !typeCanRepresentInfinity(receiverType) && s.Flags[0].CheckFlag(Case_infinite) {
			result1 = makeCurvePointPtrInterface(pointTypeXTWFull)
		}

		result1.Add(result_of_subtraction, s.Points[1])
		var got bool
		var expected bool = !singular
		if wasInvalidPointEncountered(func() { got = result1.IsEqual(s.Points[0]) }) != singular {
			return false, "Wrong NaP behaviour when checking (P-Q) + Q ?= P"
		}
		if got != expected {
			return false, "(P-Q) + Q != P"
		}

		if !typeCanRepresentInfinity(receiverType) && s.Flags[1].CheckFlag(Case_infinite) {
			return true, ""
		}

		// Check that P - Q == P + (-Q)
		negative_of_point.Neg(s.Points[1])
		result2.Add(s.Points[0], negative_of_point)
		if wasInvalidPointEncountered(func() { got = result2.IsEqual(result_of_subtraction) }) != singular {
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
	return func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		result1 := makeCurvePointPtrInterface(receiverType)
		result1.Double(s.Points[0])
		if result1.IsNaP() != singular {
			return false, "Point doubling resulted in different NaP status"
		}
		if singular {
			return true, ""
		}

		// The distinction is needed, because addition does NOT automatically go from full curve type -> subgroup type
		// whereas doubling does.
		var result2 CurvePointPtrInterface
		if typeCanOnlyRepresentSubgroup(receiverType) && !s.Points[0].CanOnlyRepresentSubgroup() {
			result2 = makeCurvePointPtrInterface(pointTypeXTWFull)
		} else {
			result2 = makeCurvePointPtrInterface(receiverType)
		}
		result2.Add(s.Points[0], s.Points[0])

		if !result1.IsEqual(result2) {
			return false, "Point doubling differs from computing P + P"
		}
		return true, ""
	}
}

func checkfun_associative_law(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(3)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	// We not do use a separate receiver type (and a make_checkfun...) due to speed.
	receiverType := getPointType(s.Points[0])
	result1 := makeCurvePointPtrInterface(receiverType)
	result2 := makeCurvePointPtrInterface(receiverType)

	if !typeCanRepresentInfinity(receiverType) {
		var result_xtw Point_xtw_full // we compute everything in Point_xtw_full - coordinates first to ensure no infinities occur
		s0 := s.Points[0].Clone().(CurvePointPtrInterfaceRead)
		s1 := s.Points[1].Clone().(CurvePointPtrInterfaceRead)
		s2 := s.Points[2].Clone().(CurvePointPtrInterfaceRead)
		result_xtw.Add(s0, s1)
		if result_xtw.IsAtInfinity() {
			return true, "skipped"
		}
		result_xtw.AddEq(s2)
		if result_xtw.IsAtInfinity() {
			return true, "skipped"
		}
		result_xtw.Add(s1, s2)
		if result_xtw.IsAtInfinity() {
			return true, "skipped"
		}
	}

	result1.Add(s.Points[0], s.Points[1])
	result1.Add(result1, s.Points[2])
	result2.Add(s.Points[1], s.Points[2])
	result2.Add(s.Points[0], result2)

	var expected bool = !singular
	return guardForInvalidPoints(expected, singular, "Test for associative law failed", result1.IsEqual, result2)
}

func test_addition_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := pointTypeToString(receiverType)
	// var type1, type2 PointType
	make_samples2_and_run_tests(t, make_checkfun_addition_commutes(receiverType), "Addition did not commute for "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type1) || !typeCanOnlyRepresentSubgroup(type2)) {
				continue
			}
			make_samples2_and_run_tests(t, make_checkfun_addition_commutes(receiverType), "Addition did not commute for "+point_string, type1, type2, 10, excludedFlags)
		}
	}

	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type1) || !typeCanOnlyRepresentSubgroup(type2)) {
				continue
			}
			make_samples1_and_run_tests(t, make_checkfun_addition_of_zero(receiverType, type1), "Addition of neutral changes point for"+point_string, type2, 10, excludedFlags)
		}
	}

	make_samples1_and_run_tests(t, make_checkfun_negative(receiverType), "Negating points did not work as expected"+point_string, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if typeCanOnlyRepresentSubgroup(receiverType) && !typeCanOnlyRepresentSubgroup(type1) {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_negative(receiverType), "Negating points did not work as expected"+point_string, type1, 10, excludedFlags)
	}

	make_samples2_and_run_tests(t, make_checkfun_subtraction(receiverType), "Subtraction did not work as expected"+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type1) || !typeCanOnlyRepresentSubgroup(type2)) {
				continue
			}
			make_samples2_and_run_tests(t, make_checkfun_subtraction(receiverType), "Subtraction did not work as expected"+point_string, type1, type2, 10, excludedFlags)
		}
	}

	make_samples1_and_run_tests(t, make_checkfun_doubling(receiverType), "Doubling does not work as intended "+point_string, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		make_samples1_and_run_tests(t, make_checkfun_doubling(receiverType), "Doubling does not work as intended "+point_string, type1, 10, excludedFlags)
	}

	// type1 == receiverType for checkfun_associative_law
	for _, type2 := range allTestPointTypes {
		for _, type3 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type2) || !typeCanOnlyRepresentSubgroup(type3)) {
				continue
			}
			make_samples3_and_run_tests(t, checkfun_associative_law, "Associative law does not hold "+point_string+" "+pointTypeToString(type2)+" "+pointTypeToString(type3), receiverType, type2, type3, 10, excludeNoPoints)
			// run_tests_on_samples(checkfun_associative_law, t, samples, "Associative law does not hold "+point_string+" "+PointTypeToString(type1)+" "+PointTypeToString(type2))
		}
	}

}

func TestAdditionProperties(t *testing.T) {
	for _, pointType := range allTestPointTypes {
		test_addition_properties(t, pointType, excludeNoPoints)
	}
}
