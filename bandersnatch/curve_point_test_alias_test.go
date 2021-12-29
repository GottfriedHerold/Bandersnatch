package bandersnatch

import (
	"testing"
)

/*
	These tests verify whether our functions work even if receiver and arguments (which are usually pointers) alias.
*/

func checkfun_alias_IsEqual(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	clone := s.Points[0].Clone()
	return guardForInvalidPoints(expected, singular, "Querying P == P (modulo A) failed for identical arguments", clone.IsEqual, clone)
}

func checkfun_alias_IsEqualExact(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	clone := s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	return guardForInvalidPoints(expected, singular, "Querying P == P exactly failed for identical arguments", clone.IsEqual_FullCurve, clone)
}

func make_checkfun_alias_Add(receiverType PointType) checkfunction {
	return func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		var clone1, clone2, clone3, clone4 CurvePointPtrInterface_FullCurve
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

		clone1 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone2 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone3 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone4 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone1.Add(clone1, clone1)
		result1.Add(clone2, clone2)
		result2.Add(clone3, clone4)

		if singular {
			return clone1.IsNaP() && result1.IsNaP() && result2.IsNaP(), "Alias test for Add did not get NaP when expected"
		}
		if !(clone1.IsEqual_FullCurve(result1) && clone1.IsEqual_FullCurve(result2)) {
			return false, "Addition gives inconsistent results when arguments alias"
		}
		return true, ""
	}
}

func make_checkfun_alias_Sub(receiverType PointType) checkfunction {
	return func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		var clone1, clone2, clone3, clone4 CurvePointPtrInterface_FullCurve
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)

		clone1 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone2 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone3 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone4 = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
		clone1.Sub(clone1, clone1)
		result1.Sub(clone2, clone2)
		result2.Sub(clone3, clone4)

		if singular {
			return clone1.IsNaP() && result1.IsNaP() && result2.IsNaP(), "Alias test for Sub did not get NaP when expected"
		}
		if !(clone1.IsEqual_FullCurve(result1) && clone1.IsEqual_FullCurve(result2)) {
			return false, "Subtraction gives inconsistent results when arguments alias"
		}
		return true, ""
	}
}

func checkfun_alias_Double(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone()
	result := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result.Double(clone2)
	clone1.Double(clone1)
	return guardForInvalidPoints(expected, singular, "Computing Double failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_Neg(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone()
	result := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result.Neg(clone2)
	clone1.Neg(clone1)
	return guardForInvalidPoints(expected, singular, "Computing negative failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_Endo(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	infinite := s.AnyFlags().CheckFlag(Case_infinite) // Endo is allowed to fail for points at infinity
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone()
	result := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result.Endo(clone2)
	clone1.Endo(clone1)
	if result.IsNaP() != clone1.IsNaP() {
		return false, "computing endo with receiver == argument leads to inconsistent NaP status"
	}
	if infinite && result.IsNaP() {
		expected = false
		singular = true
	}
	return guardForInvalidPoints(expected, singular, "Computing endomorphism failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_EndoFullCurve(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	result := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result.Endo_FullCurve(clone2)
	clone1.Endo_FullCurve(clone1)
	return guardForInvalidPoints(expected, singular, "Computing endomorphism (full version) failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_AddEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	result := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result.AddEq(clone2)
	clone1.AddEq(clone1)
	return guardForInvalidPoints(expected, singular, "Computing AddEq failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_SubEq(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	result := s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	result.SubEq(clone2)
	clone1.SubEq(clone1)
	return guardForInvalidPoints(expected, singular, "Computing SubEq failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func checkfun_alias_SetFrom(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	expected := !singular
	var clone1 CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
	clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	result := MakeCurvePointPtrInterfaceFromType(GetPointType(s.Points[0]))
	result.SetFrom(clone2)
	clone1.SetFrom(clone1)
	return guardForInvalidPoints(expected, singular, "SetFrom failed when receiver aliases argument", clone1.IsEqual_FullCurve, result)
}

func test_aliasing_CurvePointPtrInterface(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_alias_IsEqual, "Alias testing for IsEqual failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_IsEqualExact, "Alias testing for IsEqual_FullCurve failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, make_checkfun_alias_Add(receiverType), "Alias testing for Add failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, make_checkfun_alias_Sub(receiverType), "Alias testing for Sub failed "+point_string, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_alias_Add(receiverType), "Alias testing for Add failed "+point_string, type1, 10, excludedFlags)
		make_samples1_and_run_tests(t, make_checkfun_alias_Sub(receiverType), "Alias testing for Sub failed "+point_string, type1, 10, excludedFlags)
	}
	make_samples1_and_run_tests(t, checkfun_alias_Double, "Alias testing for Double failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_Neg, "Alias testing for Neg failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_Endo, "Alias testing for Endo failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_EndoFullCurve, "Alias testing for Endo_FullCurve failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_AddEq, "Alias testing for AddEq failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_SubEq, "Alias testing for SubEq failed "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_alias_SetFrom, "Alias testing for SetFrom failed "+point_string, receiverType, 10, excludedFlags)
}

func TestAliasingForXTW(t *testing.T) {
	test_aliasing_CurvePointPtrInterface(t, pointTypeXTW, 0)
}

func TestAliasingForAXTW(t *testing.T) {
	test_aliasing_CurvePointPtrInterface(t, pointTypeAXTW, 0)
}

func TestAliasingForEFGH(t *testing.T) {
	test_aliasing_CurvePointPtrInterface(t, pointTypeEFGH, 0)
}
