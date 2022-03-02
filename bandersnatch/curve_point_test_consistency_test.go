package bandersnatch

import (
	"reflect"
	"testing"
)

/*
	This file contains tests that ensure that different implementations of the CurvePointPtrInterface interface agree with each other.

	For now, this is only checked for IsNaP, IsAtInfinity, IsInSubgroup, IsE1, IsE2 and affine coos (which are unique)
	Note: We check consistency of affine coos with projective etc. in other tests.

	TODO: Check consistency also for binary operations?
*/

func TestConsistency(t *testing.T) {
	for _, referenceType := range allTestPointTypes {
		refTypeString := pointTypeToString(referenceType)
		for _, sampleType1 := range allTestPointTypes {
			make_samples1_and_run_tests(t, make_checkfun_consistency_queries(referenceType), "consistency checks for type "+refTypeString, sampleType1, 30, excludeNoPoints)
		}
	}
}

// check whether IsNaP, IsAtInfinity, IsInSubgroup, IsE1, IsE2 and affine coordinates
// remain consistent when converting to reference type. Note that depending on the sample,
// not all checks are done, since converting points at infinity / not in the subgroup might not be possible.
func make_checkfun_consistency_queries(referenceType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(PointFlagNAP)
		subgroup := !singular && s.Points[0].IsInSubgroup()
		infinite := !singular && s.AnyFlags().CheckFlag(PointFlag_infinite)
		sampleType := getPointType(s.Points[0])
		if infinite && !typeCanRepresentInfinity(referenceType) {
			return true, "skipped"
		}
		if typeCanOnlyRepresentSubgroup(referenceType) && !subgroup {
			return true, "skipped"
		}
		conv := makeCurvePointPtrInterface(referenceType)
		if singular {
			conv.SetFrom(s.Points[0])
			return conv.IsNaP(), "NaP-ness not preserved under conversion"
		}
		if typeCanOnlyRepresentSubgroup(referenceType) && !typeCanOnlyRepresentSubgroup(sampleType) {
			ok := conv.SetFromSubgroupPoint(s.Points[0].Clone(), UntrustedInput)
			if !ok {
				panic("Conversion to subgroup failed unexpectedly")
			}
		} else {
			conv.SetFrom(s.Points[0].Clone())
		}

		var ans1, ans2 interface{}
		funsByName := []string{"IsAtInfinity", "IsNeutralElement", "IsInSubgroup"}
		if !infinite {
			funsByName = append(funsByName, "X_affine", "Y_affine")
			_, ok1 := s.Points[0].(CurvePointPtrInterfaceCooReadExtended)
			_, ok2 := conv.(CurvePointPtrInterfaceCooReadExtended)
			if ok1 && ok2 {
				funsByName = append(funsByName, "T_affine")
			}
		}
		_, ok1 := s.Points[0].(CurvePointPtrInterfaceDistinguishInfinity)
		_, ok2 := conv.(CurvePointPtrInterfaceDistinguishInfinity)
		if ok1 && ok2 && typeCanRepresentInfinity(sampleType) && typeCanRepresentInfinity(referenceType) {
			funsByName = append(funsByName, "IsE1", "IsE2")
		}
		for _, funname := range funsByName {
			clone := s.Points[0].Clone()
			convClone := conv.Clone()
			cloneReflected := reflect.ValueOf(clone)
			convReflected := reflect.ValueOf(convClone)
			method1 := cloneReflected.MethodByName(funname)
			method2 := convReflected.MethodByName(funname)
			if !method1.IsValid() || !method2.IsValid() {
				panic("Invalid function name " + funname + ". This test uses reflect.MethodByName, so refactoring of method names needs to be adjusted here.")
			}
			var emptyargs []reflect.Value = make([]reflect.Value, 0)
			ans1 = method1.Call(emptyargs)[0].Interface()
			ans2 = method2.Call(emptyargs)[0].Interface()
			var good bool
			switch ans1 := ans1.(type) {
			case FieldElement:
				ans2conv := ans2.(FieldElement)
				good = ans1.IsEqual(&ans2conv)
			default:
				good = (ans1 == ans2)
			}
			if !good {
				s.Log(ans1, ans2)
				return false, "Method inconstinent for method " + funname
			}
		}
		return true, ""
	}
	return returned_function
}

/*

// Only check affine
// This test is somewhat redundant with checkfun_projective_coordinate_queries and checkfun_affine_coordinate_queries anyway.

func checkfun_consistency_coos(s TestSample) (bool, string) {

	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(Case_infinite) {
		panic("Do not run this test on infinte points")
	}
	if s.AnyFlags().CheckFlag(Case_singular) {
		panic("Do not run this check on NaPs")
	}
	var point_copy CurvePointPtrInterfaceRead = s.Points[0].Clone()
	var point_copy_xtw Point_xtw = s.Points[0].ExtendedTwistedEdwards()
	X1 := point_copy.X_affine()
	Y1 := point_copy.Y_affine()
	X2 := point_copy_xtw.X_affine()
	Y2 := point_copy_xtw.Y_affine()
	if !X1.IsEqual(&X2) {
		return false, "X/Z coordinate not consistent with conversion to ExtendedTwistedEdwards"
	}
	if !Y1.IsEqual(&Y2) {
		return false, "Y/Z coordinate not consistent with conversion to ExtendedTwistedEdwards"
	}
	return true, ""
}

func checkfun_consistency_IsNeutralExact(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	point_xtw := s.Points[0].ExtendedTwistedEdwards()

	// expected := !singular
	return guardForInvalidPoints(point_xtw.IsNeutralElement_FullCurve(), singular, "IsNeutralElement_FullCurve not compatible with conversion to xtw", s.Points[0].IsNeutralElement_FullCurve)
}

func checkfun_consistency_IsNeutralElement(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	point_xtw := s.Points[0].ExtendedTwistedEdwards()

	// expected := !singular
	return guardForInvalidPoints(point_xtw.IsNeutralElement(), singular, "IsNeutralElement not compatible with conversion to xtw", s.Points[0].IsNeutralElement)
}

func checkfun_consistency_IsAtInfinity(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	point_xtw := s.Points[0].ExtendedTwistedEdwards()

	// expected := !singular
	return guardForInvalidPoints(point_xtw.IsAtInfinity(), singular, "IsAtInfinity not compatible with conversion to xtw", s.Points[0].IsAtInfinity)
}

func checkfun_consistency_IsNaP(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	point_copy := s.Points[0].ExtendedTwistedEdwards()

	return point_copy.IsNaP() == s.Points[0].IsNaP(), "IsNaP does not commute with conversion to xtw"
}

func checkfun_consistency_IsEqual_FullCurve(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	// singular := s.AnyFlags().CheckFlag(Case_singular)
	point_xtw0 := s.Points[0].ExtendedTwistedEdwards()
	poiny_xtw1 := s.Points[1].ExtendedTwistedEdwards()

	return point_xtw0.IsEqual_FullCurve(&poiny_xtw1) == s.Points[0].IsEqual_FullCurve(s.Points[1]), "Exact equality comparison does not commute with converstion to xtw"
}

func checkfun_consistency_IsEqual(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	// singular := s.AnyFlags().CheckFlag(Case_singular)
	point_xtw0 := s.Points[0].ExtendedTwistedEdwards()
	poiny_xtw1 := s.Points[1].ExtendedTwistedEdwards()

	return point_xtw0.IsEqual(&poiny_xtw1) == s.Points[0].IsEqual(s.Points[1]), "Exact equality comparison does not commute with converstion to xtw"
}

func make_checkfun_consistency_Add(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(2)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		if s.AnyFlags().CheckFlag(Case_differenceInfinite) {
			return true, "" // skip test
		}
		point_xtw0 := s.Points[0].ExtendedTwistedEdwards()
		poiny_xtw1 := s.Points[1].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver1.Add(&point_xtw0, &poiny_xtw1)
		receiver2.Add(s.Points[0], s.Points[1])
		var receiver3, receiver4 Point_xtw
		receiver3.Add(s.Points[0], s.Points[1])
		receiver4.Add(&point_xtw0, &poiny_xtw1)
		expected := !singular
		ok, err = guardForInvalidPoints(expected, singular, "Addition does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Addition does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Addition does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func make_checkfun_consistency_Sub(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(2)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		if s.AnyFlags().CheckFlag(Case_outside_goodgroup) {
			return true, "" // skip test
		}
		point_xtw0 := s.Points[0].ExtendedTwistedEdwards()
		poiny_xtw1 := s.Points[1].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver1.Sub(&point_xtw0, &poiny_xtw1)
		receiver2.Sub(s.Points[0], s.Points[1])
		var receiver3, receiver4 Point_xtw
		receiver3.Sub(s.Points[0], s.Points[1])
		receiver4.Sub(&point_xtw0, &poiny_xtw1)
		expected := !singular
		ok, err = guardForInvalidPoints(expected, singular, "Subtraction does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Subtraction does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Subtraction does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func make_checkfun_consistency_Double(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)

		point_xtw := s.Points[0].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		var receiver3, receiver4 Point_xtw

		receiver1.Double(&point_xtw)
		receiver2.Double(s.Points[0])
		receiver3.Double(&point_xtw)
		receiver4.Double(s.Points[0])

		expected := !singular

		ok, err = guardForInvalidPoints(expected, singular, "Doubling does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Doubling does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Doubling does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func make_checkfun_consistency_Endo(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		if s.AnyFlags().CheckFlag(Case_infinite) {
			panic("Do not call with infinite points")
		}

		point_xtw := s.Points[0].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		var receiver3, receiver4 Point_xtw

		receiver1.Endo(&point_xtw)
		receiver2.Endo(s.Points[0])
		receiver3.Endo(&point_xtw)
		receiver4.Endo(s.Points[0])

		expected := !singular

		ok, err = guardForInvalidPoints(expected, singular, "Endo does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Endo does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Endo does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func make_checkfun_consistency_Endo_safe(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)

		point_xtw := s.Points[0].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		var receiver3, receiver4 Point_xtw

		receiver1.Endo_FullCurve(&point_xtw)
		receiver2.Endo_FullCurve(s.Points[0])
		receiver3.Endo_FullCurve(&point_xtw)
		receiver4.Endo_FullCurve(s.Points[0])

		expected := !singular

		ok, err = guardForInvalidPoints(expected, singular, "Endo_FullCurve does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Endo_FullCurve does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Endo_FullCurve does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func make_checkfun_consistency_Neg(receiverType PointType) checkfunction {
	return func(s TestSample) (ok bool, err string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)

		if !canRepresentInfinity(receiverType) && s.AnyFlags().CheckFlag(Case_infinite) {
			return true, "" // skip test
		}

		point_xtw := s.Points[0].ExtendedTwistedEdwards()
		receiver1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		receiver2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		var receiver3, receiver4 Point_xtw

		receiver1.Neg(&point_xtw)
		receiver2.Neg(s.Points[0])
		receiver3.Neg(&point_xtw)
		receiver4.Neg(s.Points[0])

		expected := !singular

		ok, err = guardForInvalidPoints(expected, singular, "Neg does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, receiver2)
		if !ok {
			return
		}
		ok, err = guardForInvalidPoints(expected, singular, "Neg does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver3)
		if !ok {
			return
		}
		return guardForInvalidPoints(expected, singular, "Neg does not commute with conversion to xtw", receiver1.IsEqual_FullCurve, &receiver4)
	}
}

func test_consistency_with_xtw(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_consistency_IsNeutralExact, "exact Neutral Element test inconstent with conversion "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_consistency_IsNeutralElement, "Neutral Element test inconstent with conversion "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_consistency_coos, "affine coordinate functions inconstent with conversion "+point_string, receiverType, 10, excludedFlags|Case_infinite|Case_singular)
	make_samples1_and_run_tests(t, checkfun_consistency_IsNaP, "IsNaP inconstent with conversion "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_consistency_IsAtInfinity, "IsAtInfinity inconstent with conversion "+point_string, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_consistency_IsEqual, "Equality test inconstent with conversion "+point_string, receiverType, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_consistency_IsEqual_FullCurve, "exact equality test inconstent with conversion "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type2 := range allTestPointTypes {
		if type2 == receiverType {
			continue
		}
		make_samples2_and_run_tests(t, checkfun_consistency_IsEqual, "Equality test inconstent with conversion "+point_string, receiverType, type2, 10, excludedFlags)
		make_samples2_and_run_tests(t, checkfun_consistency_IsEqual_FullCurve, "exact equality test inconstent with conversion "+point_string, receiverType, type2, 10, excludedFlags)
	}
	make_samples2_and_run_tests(t, make_checkfun_consistency_Add(receiverType), "Addition inconsistent with conversion to xtw "+point_string, receiverType, receiverType, 10, excludedFlags|Case_differenceInfinite)
	make_samples2_and_run_tests(t, make_checkfun_consistency_Sub(receiverType), "Subtraction inconsistent with conversion to xtw "+point_string, receiverType, receiverType, 10, excludedFlags|Case_outside_goodgroup)
	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			make_samples2_and_run_tests(t, make_checkfun_consistency_Add(receiverType), "Addition inconsistent with conversion to xtw "+point_string, type1, type2, 10, excludedFlags|Case_differenceInfinite)
			make_samples2_and_run_tests(t, make_checkfun_consistency_Sub(receiverType), "Subtraction inconsistent with conversion to xtw "+point_string, type1, type2, 10, excludedFlags|Case_outside_goodgroup)
		}
	}
	make_samples1_and_run_tests(t, make_checkfun_consistency_Double(receiverType), "Doubling inconsistent with conversion to xtw "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, make_checkfun_consistency_Neg(receiverType), "Negation inconsistent with conversion to xtw "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, make_checkfun_consistency_Endo(receiverType), "Endo inconsistent with conversion to xtw "+point_string, receiverType, 10, excludedFlags|Case_infinite)
	make_samples1_and_run_tests(t, make_checkfun_consistency_Endo_safe(receiverType), "Endo_FullCurve inconsistent with conversion to xtw "+point_string, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_consistency_Double(receiverType), "Doubling inconsistent with conversion to xtw "+point_string, type1, 10, excludedFlags)
		make_samples1_and_run_tests(t, make_checkfun_consistency_Neg(receiverType), "Negation inconsistent with conversion to xtw "+point_string, type1, 10, excludedFlags)
		make_samples1_and_run_tests(t, make_checkfun_consistency_Endo(receiverType), "Endo inconsistent with conversion to xtw "+point_string, type1, 10, excludedFlags|Case_infinite)
		make_samples1_and_run_tests(t, make_checkfun_consistency_Endo_safe(receiverType), "Endo_FullCurve inconsistent with conversion to xtw "+point_string, type1, 10, excludedFlags)
	}

}

func TestConsistencyXTWForXTW(t *testing.T) {
	// consistency with itself. -- more of a sanity check...
	test_consistency_with_xtw(t, pointTypeXTW, 0)
}

func TestConsistencyXTWForAXTW(t *testing.T) {
	test_consistency_with_xtw(t, pointTypeAXTW, 0)
}

func TestConsistencyXTWForEFGH(t *testing.T) {
	test_consistency_with_xtw(t, pointTypeEFGH, 0)
}

*/
