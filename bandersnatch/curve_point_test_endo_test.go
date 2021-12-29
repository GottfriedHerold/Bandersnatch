package bandersnatch

import "testing"

// This function checks whether the endomorphism factors through P=P+A and Endo and Endo_FullCurve agree
func make_checkfun_endo_sane(receiverType PointType) checkfunction {
	return func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		var infinite bool = s.AnyFlags().CheckFlag(Case_infinite)
		var result1 = MakeCurvePointPtrInterfaceFromType(receiverType)
		var result2 = MakeCurvePointPtrInterfaceFromType(receiverType)
		result1.Endo_FullCurve(s.Points[0])

		if singular {
			if !result1.IsNaP() {
				return false, "Endo_FullCurve(NaP) did not result in NaP"
			}
			result2.Endo(s.Points[0])
			if !result2.IsNaP() {
				return false, "Endo(NaP) did not result in NaP"
			}
			// No further checks
			return true, ""
		}

		if result1.IsNaP() {
			return false, "Endo_FullCurve(P) resulted in NaP for non-NaP P"
		}

		var result_xtw Point_xtw = result1.ExtendedTwistedEdwards()
		if !result_xtw.isPointOnCurve() {
			return false, "Endo result is not on curve"
		}
		if !result_xtw.legendre_check_point() {
			return false, "Endo result not in subgroup"
		}

		if !infinite { // Endo may not work on points at infinity
			result2.Endo(s.Points[0])
			if !result1.IsEqual_FullCurve(result2) {
				return false, "Endo(P) and Endo_exact(P) differ"
			}
		} else {
			// input was point at infinity. Output should be affine order-2 point
			if !result1.IsEqual_FullCurve(&orderTwoPoint_xtw) {
				return false, "Endo_FullCurve(infinite point) != affine order-2 point"
			}
		}

		if s.Points[0].IsNeutralElement() != result1.IsNeutralElement_FullCurve() {
			return false, "Endo_FullCurve act as expected wrt neutral elements"
		}
		if !infinite { // On infinite points, AddEq(&orderTwoPoint) might not work
			var point_copy = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)
			point_copy.AddEq(&orderTwoPoint_xtw)
			result2.Endo_FullCurve(point_copy)
			if !result1.IsEqual_FullCurve(result2) {
				return false, "Endo_FullCurve(P) != Endo_FullCurve(P+A)"
			}
		}
		return true, ""
	}
}

// checks whether Endo(P) + Endo(Q) == Endo(P+Q)
func make_checkfun_endo_homomorphic(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		// This should be ruled out at the call site
		if s.AnyFlags().CheckFlag(Case_singular) {
			panic("Should not call checkfun_endo_homomorphic on NaP test samples")
		}
		if s.AnyFlags().CheckFlag(Case_differenceInfinite) {
			return true, "" // need to skip test, because computing P+Q will fail.
		}
		endo1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		endo2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		sum := MakeCurvePointPtrInterfaceFromType(receiverType)
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType)
		endo1.Endo_FullCurve(s.Points[0])
		endo2.Endo_FullCurve(s.Points[1])
		sum.Add(s.Points[0], s.Points[1])
		result1.Add(endo1, endo2)
		result2.Endo_FullCurve(sum)
		if result1.IsNaP() {
			return false, "Endo(P) + Endo(Q) resulted in NaP" // cannot trigger exceptional cases of addition, because the range of Endo is the good subgroup (verified by endo_sane).
		}
		if result2.IsNaP() {
			return false, "Endo(P+Q) resulted in unexpected NaP"
		}
		if !result1.IsEqual_FullCurve(result2) {
			return false, "Endo(P+Q) != Endo(P) + Endo(Q)"
		}
		return true, ""
	}
	return
}

// checks whether the Endomorphism acts as exponentiation by sqrt(2)
func checkfun_endo_action(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var p253 bool = !(s.AnyFlags().CheckFlag(Case_outside_goodgroup) || s.AnyFlags().CheckFlag(Case_outside_p253) || s.AnyFlags().CheckFlag(Case_infinite))
	var good_subgroup = !(s.AnyFlags().CheckFlag(Case_outside_goodgroup) || s.AnyFlags().CheckFlag(Case_infinite))
	pointType := GetPointType(s.Points[0])
	result1 := MakeCurvePointPtrInterfaceFromType(pointType)
	result1.Endo_FullCurve(s.Points[0])
	if result1.IsNaP() != singular {
		return false, "Running Endo_FullCurve resulted in different NaP-status than the argument"
	}
	if singular {
		// skip further tests. The relevant properties are verified by endo_sane
		return true, ""
	}
	var input_copy Point_xtw = s.Points[0].ExtendedTwistedEdwards()
	var result2 Point_xtw
	result2.exp_naive_xx(&input_copy, EndomorphismEigenvalue_Int)

	// We (should) have Endo(P) == Endoeigenvalue * P exactly on the p253 subgroup.
	// Furthermore, we have Endo(A) = N, Endo(infinity) = A and Endo is homomorphic.

	if p253 {
		if !result1.IsEqual_FullCurve(&result2) {
			return false, "Running Endo_FullCurve did not (exactly) match exponentiation operation"
		}
	} else if good_subgroup {
		// Outside p253, but still in the good subgroup, we still have Endo(P) = Eigenvalue*P + A (in fact, this can be used to distinguish p253 from G')
		if !result1.IsEqual(&result2) {
			return false, "Running Endo_FullCurve did not match exponentiation operation"
		}
	} else {
		result1.DoubleEq()
		result2.DoubleEq()
		if result2.IsNaP() {
			if !s.AnyFlags().CheckFlag(Case_infinite) {
				panic("exp_naive failed in point where this should not be possible")
			}
			return true, "" // Skip in this borderline case. This can only happen if exp_naive failed, which should only
		}
		if !result1.IsEqual_FullCurve(&result2) {
			return false, "Running Endo_FullCurve did not match exponentiation up to 2-torsion"
		}
	}
	return true, ""
}

func test_endomorphism_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	var type1, type2 PointType

	make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, receiverType, 10, excludedFlags)
	for _, type1 = range allTestPointTypes {
		if type1 == receiverType {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, type1, 10, excludedFlags)
	}

	make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, receiverType, receiverType, 10, excludedFlags|Case_singular)
	for _, type1 = range allTestPointTypes {
		for _, type2 = range allTestPointTypes {
			make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, type1, type2, 10, excludedFlags|Case_singular)
		}
	}

	make_samples1_and_run_tests(t, checkfun_endo_action, "Endomorphism does not act as intended "+point_string, receiverType, 10, excludedFlags)

}

func TestEndomorphismPropertiesForXTW(t *testing.T) {
	test_endomorphism_properties(t, pointTypeXTW, 0)
}

func TestEndomorphismPropertiesForAXTW(t *testing.T) {
	test_endomorphism_properties(t, pointTypeAXTW, 0)
}

func TestEndomorphismPropertiesForEFGH(t *testing.T) {
	test_endomorphism_properties(t, pointTypeEFGH, 0)
}
