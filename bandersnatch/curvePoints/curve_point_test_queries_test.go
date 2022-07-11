package curvePoints

import "testing"

// This file contains generic tests for curve points that ensure that certain query functions work as intended, namely:
// IsAtInfinity, IsNaP, Validate

func TestQueriesForAllPointTypes(t *testing.T) {
	for _, pointType := range allTestPointTypes {
		test_queries(t, pointType, excludeNoPoints)
	}
}

func test_queries(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := pointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_recognize_infinity, "Did not recognize infinite points "+point_string, receiverType, 50, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_recognize_NaP, "Did not recognize invalid points arising from singularities "+point_string, receiverType, 50, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_IsAtInfinity_consistent, "IsAtInfinity inconsistent "+point_string, receiverType, 50, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_IsNaP_consistentAXTW, "IsNaP inconsistent with conversion to axtw "+point_string, receiverType, 50, excludedFlags|PointFlag_infinite)
	make_samples1_and_run_tests(t, checkfun_IsNaP_consistentXTW, "IsAtInfinity inconsistent with conversion to xtw "+point_string, receiverType, 50, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_validate, "Validation failure for "+point_string, receiverType, 50, excludedFlags)
}

func checkfun_validate(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(PointFlagNAP)
	valid := s.Points[0].Validate()
	if singular && valid {
		return false, "NaP passed as valid"
	}
	if !singular && !valid {
		return false, "non-NaP was not recognized as valid"
	}
	return true, ""
}

// checks whether IsAtInfinity correctly recognizes points at infinity.
func checkfun_recognize_infinity(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(PointFlagNAP)
	var expected bool = s.Flags[0].CheckFlag(PointFlag_infinite)
	return guardForInvalidPoints(expected, singular, "Infinite point was not recognized", s.Points[0].IsAtInfinity)
}

// checks whether IsAtInfinity commutes with conversion to xtw

func checkfun_IsAtInfinity_consistent(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var point_copy Point_xtw_full
	point_copy.SetFrom(s.Points[0])
	return point_copy.IsAtInfinity() == s.Points[0].IsAtInfinity(), "IsAtInfinity does not commute with conversion to xtw."
}

// checks whether IsNaP() correctly recognizes NaPs
func checkfun_recognize_NaP(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var expected bool = s.Flags[0].CheckFlag(PointFlagNAP)
	var got = s.Points[0].IsNaP()
	return expected == got, "Test sample marked as singular, but IsNaP() does not agree"
}

// somewhat redundant with tests for conversion

func checkfun_IsNaP_consistentXTW(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var point_copy Point_xtw_full
	point_copy.SetFrom(s.Points[0])
	return point_copy.IsNaP() == s.Points[0].IsNaP(), "IsNaP does not commute with conversion to xtw"
}

func checkfun_IsNaP_consistentAXTW(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(PointFlag_infinite) {
		panic("Do not call checkfun_IsNaP_consistentAXTW with points at infinity")
	}
	var point_copy Point_axtw_full
	point_copy.SetFrom(s.Points[0])
	return point_copy.IsNaP() == s.Points[0].IsNaP(), "IsNaP does not commute with conversion to axtw"
}
