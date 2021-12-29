package bandersnatch

import "testing"

// checks whether IsNeutralElement correctly recognized neutral elements
func checkfun_recognize_neutral(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.Flags[0].CheckFlag(Case_zero) && !singular
	return guardForInvalidPoints(expected, singular, "Neutral point not recognized", s.Points[0].IsNeutralElement)
}

// checks whether IsNeutralElement_FullCurve correctly recognizes neutral elements
func checkfun_recognize_neutral_exact(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.Flags[0].CheckFlag(Case_zero_exact) && !singular
	return guardForInvalidPoints(expected, singular, "exact testing for neutral element did not work.", s.Points[0].IsNeutralElement_FullCurve)
}

// checks whether IsEqual correctly recognizes pairs of equal points (modulo P = P+A)
func checkfun_recognize_equality(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.AnyFlags().CheckFlag(Case_equal) && !singular
	return guardForInvalidPoints(expected, singular, "equality testing failed", s.Points[0].IsEqual, s.Points[1])
}

// checks whether IsEqual_FullCurve correctly recognizes pairs of exactly equal points.
func checkfun_recognize_equality_exact(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.AnyFlags().CheckFlag(Case_equal_exact) && !singular
	return guardForInvalidPoints(expected, singular, "equality testing failed", s.Points[0].IsEqual_FullCurve, s.Points[1])
}

func test_equality_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_recognize_neutral, "Did not recognize neutral element for "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_recognize_neutral_exact, "Did not recognize exact neutral element for "+point_string, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality "+point_string, receiverType, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_recognize_equality_exact, "Did not recognize exact equality "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue // already checked
		}
		make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality "+point_string, receiverType, type1, 10, excludedFlags)
		make_samples2_and_run_tests(t, checkfun_recognize_equality_exact, "Did not recognize exact equality "+point_string, receiverType, type1, 10, excludedFlags)
	}
}

func TestEqualityForXTW(t *testing.T) {
	test_equality_properties(t, pointTypeXTW, 0)
}

func TestEqualityForAXTW(t *testing.T) {
	test_equality_properties(t, pointTypeAXTW, 0)
}

func TestEqualityForEFGH(t *testing.T) {
	test_equality_properties(t, pointTypeEFGH, 0)
}
