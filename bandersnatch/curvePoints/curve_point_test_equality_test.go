package bandersnatch

import (
	"fmt"
	"math/rand"
	"testing"
)

// This test file runs checks whether the neutral element is correctly recognized and equality is correctly recognized.
// Note that our sampling framework marks samples according to what we expect here.
//
// We also additionally check whether clones / SetFrom / SetFromSubgroup makes things compare equal.

// checks whether IsNeutralElement correctly recognized neutral elements
func checkfun_recognize_neutral(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(PointFlagNAP)
	var expected bool = s.Flags[0].CheckFlag(PointFlag_zeroExact) && !singular
	return guardForInvalidPoints(expected, singular, "Neutral point not recognized", s.Points[0].IsNeutralElement)
}

// checks whether IsEqual correctly recognizes pairs of equal points
func checkfun_recognize_equality(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
	var expected bool = s.AnyFlags().CheckFlag(PointFlag_equalExact) && !singular
	result1, result2 := guardForInvalidPoints(expected, singular, "equality testing failed", s.Points[0].IsEqual, s.Points[1])
	if !result1 {
		fmt.Println(expected)
	}
	return result1, result2
}

// partially redundant with checkfun_alias_IsEqual

// check whether IsEqual correctly recognizes clones of points
func checkfun_equality_for_clones(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(PointFlagNAP)
	if singular {
		return true, "skipped"
	}
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceTestSample)

	if !clone1.IsEqual(s.Points[0]) {
		return false, "clone not equal to original"
	}
	var rng *rand.Rand = rand.New(rand.NewSource(1000))
	for i := 0; i < 10; i++ {
		clone3 := s.Points[0].Clone().(CurvePointPtrInterfaceTestSample)
		clone3.rerandomizeRepresentation(rng)
		if !clone2.IsEqual(clone3) {
			return false, "Clones are not considered equal after rerandomization"
		}
	}
	return true, ""
}

// check whether IsEqual correctly recognize points copied via SetFrom
func make_checkfun_equality_SetFrom(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(PointFlagNAP)
		receiverSubgroupOnly := typeCanOnlyRepresentSubgroup(receiverType)
		sourceSubgroupOnly := s.Points[0].CanOnlyRepresentSubgroup()
		receiverCanStoreInfinity := typeCanRepresentInfinity(receiverType)
		if receiverSubgroupOnly && !sourceSubgroupOnly {
			return true, "skipped"
		}
		if singular {
			return true, "skipped"
		}
		if !receiverCanStoreInfinity && s.AnyFlags().CheckFlag(PointFlag_infinite) {
			return true, "skipped"
		}
		pointCopy := makeCurvePointPtrInterface(receiverType)
		pointCopy.SetFrom(s.Points[0].Clone())
		if !pointCopy.IsEqual(s.Points[0]) {
			return false, "SetFrom does not result in point that is considered equal"
		}
		return true, ""
	}
	return
}

// check whether IsEqual correctly recognized points copied via SetFromSubgroup
func make_checkfun_equality_SetFromSubgroup(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(PointFlagNAP)
		// receiverSubgroupOnly := typeCanOnlyRepresentSubgroup(receiverType)
		sourceSubgroupOnly := s.Points[0].CanOnlyRepresentSubgroup()
		receiverCanStoreInfinity := typeCanRepresentInfinity(receiverType)
		pointCopy := makeCurvePointPtrInterface(receiverType)
		if singular {
			return true, "skipped"
		}
		if !receiverCanStoreInfinity && s.AnyFlags().CheckFlag(PointFlag_infinite) {
			return true, "skipped"
		}
		ok := pointCopy.SetFromSubgroupPoint(s.Points[0].Clone(), untrustedInput)
		if !ok {
			if sourceSubgroupOnly {
				return false, "SetFromSubgroupPoint returned ok=false for source point type that can only store subgroup points"
			} else {
				return true, ""
			}
		}
		if !pointCopy.IsEqual(s.Points[0]) {
			return false, "SetFromSubgroupPoint does not result in point that is considered equal to source"
		}
		pointCopy2 := makeCurvePointPtrInterface(receiverType)
		ok2 := pointCopy2.SetFromSubgroupPoint(s.Points[0].Clone(), trustedInput)
		if !ok2 {
			return false, "SetFromSubgroupPoint with trusted input returned false when untrusted returned true"
		}
		if !pointCopy2.IsEqual(s.Points[0]) {
			return false, "SetFromSubgroupPoint with trusted input did not result in point that is considered equal to source"
		}
		return true, ""
	}
	return
}

func TestEqualityProperties(t *testing.T) {
	for _, receiverType := range allTestPointTypes {
		point_string := pointTypeToString(receiverType)
		make_samples1_and_run_tests(t, checkfun_recognize_neutral, "Did not recognize neutral element for "+point_string, receiverType, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_equality_for_clones, "Clone not recognized as equal for "+point_string, receiverType, 100, excludeNoPoints)
		make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality "+point_string, receiverType, receiverType, 50, excludeNoPoints)
		for _, type1 := range allTestPointTypes {
			if type1 == receiverType {
				continue // already checked
			}
			other_string := pointTypeToString(type1)
			make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality for "+point_string+" and "+other_string, receiverType, type1, 50, excludeNoPoints)
			make_samples1_and_run_tests(t, make_checkfun_equality_SetFrom(receiverType), "SetFrom not compatible with equality for "+point_string+" and "+other_string, type1, 50, excludeNoPoints)
			make_samples1_and_run_tests(t, make_checkfun_equality_SetFromSubgroup(receiverType), "SetFromSubgroup not compatible with equality for "+point_string+" and "+other_string, type1, 50, excludeNoPoints)
		}
	}
}
