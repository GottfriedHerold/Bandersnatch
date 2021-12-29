package bandersnatch

import "testing"

// checks whether IsAtInfinity correctly recognizes points at infinity.
func checkfun_recognize_infinity(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.Flags[0].CheckFlag(Case_infinite)
	return guardForInvalidPoints(expected, singular, "Infinite point was not recognized", s.Points[0].IsAtInfinity)
}

// checks whether IsAtInfinity commutes with conversion to xtw

func checkfun_IsAtInfinity_consistent(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	point_copy := s.Points[0].ExtendedTwistedEdwards()
	return point_copy.IsAtInfinity() == s.Points[0].IsAtInfinity(), "IsAtInfinity does not commute with conversion to xtw."
}

// checks whether IsNaP() correctly recognizes NaPs
func checkfun_recognize_NaP(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var expected bool = s.Flags[0].CheckFlag(Case_singular)
	var got = s.Points[0].IsNaP()
	return expected == got, "Test sample marked as singular, but IsNaP() does not agree"
}

// somewhat redundant with tests for conversion

func checkfun_IsNaP_consistentXTW(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	point_copy := s.Points[0].ExtendedTwistedEdwards()
	return point_copy.IsNaP() == s.Points[0].IsNaP(), "IsNaP does not commute with conversion to xtw"
}

func checkfun_IsNaP_consistentAXTW(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(Case_infinite) {
		panic("Do not call checkfun_IsNaP_consistentAXTW with points at infinity")
	}
	point_copy := s.Points[0].AffineExtended()
	return point_copy.IsNaP() == s.Points[0].IsNaP(), "IsNaP does not commute with conversion to axtw"
}

func test_queries(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_recognize_infinity, "Did not recognize infinite points "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_recognize_NaP, "Did not recognize invalid points arising from singularities "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_IsAtInfinity_consistent, "IsAtInfinity inconsistent "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_IsNaP_consistentAXTW, "IsNaP inconsistent with conversion to axtw "+point_string, receiverType, 10, excludedFlags|Case_infinite)
	make_samples1_and_run_tests(t, checkfun_IsNaP_consistentXTW, "IsAtInfinity inconsistent with conversion to xtw "+point_string, receiverType, 10, excludedFlags)
}

func TestQueriesForXTW(t *testing.T) {
	test_queries(t, pointTypeXTW, 0)
}

func TestQueriesForAXTW(t *testing.T) {
	test_queries(t, pointTypeAXTW, 0)
}

func TestQueriesForEFGH(t *testing.T) {
	test_queries(t, pointTypeEFGH, 0)
}
