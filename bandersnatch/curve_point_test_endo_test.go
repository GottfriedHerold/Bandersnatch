package bandersnatch

import "testing"

// This function checks whether the endomorphism factors through P vs P+A.
func make_checkfun_endo_sane(receiverType PointType) checkfunction {
	return func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		sampleType := GetPointType(s.Points[0])
		var singular bool = s.AnyFlags().CheckFlag(Case_singular)
		var infinite bool = s.AnyFlags().CheckFlag(Case_infinite)
		var result = MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		// var result2 = MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		result.Endo(s.Points[0])

		if singular {
			if !result.IsNaP() {
				return false, "Endo_FullCurve(NaP) did not result in NaP"
			}
			// No further checks
			return true, ""
		}

		if result.IsNaP() {
			return false, "Endo(P) resulted in NaP for non-NaP P"
		}

		if !result.(validateable).Validate() {
			return false, "Endo(P) is not on curve"
		}

		resultClone := result.Clone().(CurvePointPtrInterface)
		var X, Z FieldElement
		X = resultClone.X_projective()
		Z = resultClone.Z_projective()
		if !legendreCheckA_projectiveXZ(X, Z) {
			return false, "Result of applying endomorphism is not in 2*p253-subgroup"
		}

		if infinite {
			if !result.IsEqual(&orderTwoPoint_xtwfull) {
				return false, "Endo(infinite point) != affine two-torsion"
			}
		} else if s.AnyFlags().CheckFlag(Case_zero_exact) {
			if !result.IsNeutralElement() {
				return false, "Endo(N) != N"
			}
		} else if s.AnyFlags().CheckFlag(Case_A) {
			if !result.IsNeutralElement() {
				return false, "Endo(A) != N"
			}
		}

		if !typeCanOnlyRepresentSubgroup(sampleType) {
			inputClone := s.Points[0].Clone().(CurvePointPtrInterface)
			inputClone.AddEq(&orderTwoPoint_xtwfull)
			resultClone.Endo(inputClone)
			if !result.IsEqual(resultClone) {
				return false, "Endo(P) != Endo(P+A)"
			}
		}

		return true, ""
	}
}

// checks whether Endo(P) + Endo(Q) == Endo(P+Q)
func make_checkfun_endo_homomorphic(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		// This should be ruled out at the call site
		if s.AnyFlags().CheckFlag(Case_singular) {
			panic("Should not call checkfun_endo_homomorphic on NaP test samples")
		}

		if !typeCanRepresentInfinity(receiverType) {
			assert(typeCanRepresentInfinity(pointTypeXTWFull))
			var P1, P2, sumInput, sumEndo Point_xtw_full
			P1.Endo(s.Points[0])
			P2.Endo(s.Points[1])
			sumInput.Add(s.Points[0], s.Points[1])
			sumEndo.Endo(&sumInput)
			if P1.IsAtInfinity() || P2.IsAtInfinity() || sumInput.IsAtInfinity() || sumEndo.IsAtInfinity() {
				return true, "Skipped"
			}
		}

		endo1 := MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		endo2 := MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		sum := MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		result1 := MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		result2 := MakeCurvePointPtrInterfaceFromType(receiverType).(CurvePointPtrInterface)
		endo1.Endo(s.Points[0])
		endo2.Endo(s.Points[1])
		sum.Add(s.Points[0], s.Points[1])
		result1.Add(endo1, endo2)
		result2.Endo(sum)
		if result1.IsNaP() {
			return false, "Endo(P) + Endo(Q) resulted in NaP"
		}
		if result2.IsNaP() {
			return false, "Endo(P+Q) resulted in unexpected NaP"
		}
		if !result1.IsEqual(result2) {
			return false, "Endo(P+Q) != Endo(P) + Endo(Q)"
		}
		return true, ""
	}
	return
}

// checks whether the Endomorphism acts as exponentiation by sqrt(2)
func checkfun_endo_action(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var p253 bool = !(s.AnyFlags().CheckFlag(Case_outside_p253 | Case_random))
	var good_subgroup = !(s.AnyFlags().CheckFlag(Case_outside_goodgroup | Case_random))
	var random = s.AnyFlags().CheckFlag(Case_random)
	// var good_subgroup = !(s.AnyFlags().CheckFlag(Case_outside_goodgroup) || s.AnyFlags().CheckFlag(Case_infinite))
	pointType := GetPointType(s.Points[0])
	result1 := MakeCurvePointPtrInterfaceFromType(pointType).(CurvePointPtrInterface)
	result1.Endo(s.Points[0])
	if result1.IsNaP() != singular {
		return false, "Running Endo_FullCurve resulted in different NaP-status than the argument"
	}
	if singular {
		// skip further tests. The relevant properties are verified by endo_sane
		return true, ""
	}
	if s.AnyFlags().CheckFlag(Case_infinite) {
		// exp_naive might not work in this case.
		// Note that endo_sane covers this case anyway.
		return true, "skipped"
	}
	var input_copy Point_xtw_full // = s.Points[0].ExtendedTwistedEdwards()
	input_copy.SetFrom(s.Points[0])
	var result2 Point_xtw_full
	result2.exp_naive_xx(&input_copy.point_xtw_base, EndomorphismEigenvalue_Int)

	// We (should) have Endo(P) == Endoeigenvalue * P exactly on the p253 subgroup.
	// Furthermore, we have Endo(A) = N, Endo(infinity) = A and Endo is homomorphic.

	if !endomorphismEigenvalueIsOdd {
		// not really needed, but the testing code depends on this.
		// We need to adjust things if that changes.
		// (Notably, Endo(P) = Eigenvalue * P would hold for all rational points if Eigenvalue was even.)
		panic("Endomorphism eigenvalue is even")
	}

	if p253 {
		if !result1.IsEqual(&result2) {
			return false, "Running Endo did not match exponentiation operation"
		}
	} else if good_subgroup {
		// Endo(P) and Eigenvalue*P differ exactly by +A in that case.
		var difference Point_xtw_full
		difference.Sub(result1, &result2)
		if !difference.IsEqual(&orderTwoPoint_xtwfull) {
			return false, "Running Endo did not match exponentiation operation up to A in 2*p253-subgroup"
		}
	} else if random {
		var difference Point_xtw_full
		difference.Sub(result1, &result2)
		difference.DoubleEq()
		if !difference.IsNeutralElement() {
			return false, "Endo(P) and exponentiation do not differ by two-torsion."
		}
	} else {
		var difference Point_xtw_full
		difference.Sub(result1, &result2)
		if !difference.IsAtInfinity() {
			return false, "Running Endo did not match exponentiation operation up to infinity outside of 2*p253 subgroup"
		}
	}
	return true, ""
}

func test_endomorphism_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	// var type1, type2 PointType

	make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue
		}
		if typeCanOnlyRepresentSubgroup(receiverType) && !typeCanOnlyRepresentSubgroup(type1) {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, type1, 10, excludedFlags)
	}

	make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, receiverType, receiverType, 10, excludedFlags|Case_singular)
	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type1) || !typeCanOnlyRepresentSubgroup(type2)) {
				continue
			}
			make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, type1, type2, 10, excludedFlags|Case_singular)
		}
	}
	make_samples1_and_run_tests(t, checkfun_endo_action, "Endomorphism does not act as intended "+point_string, receiverType, 10, excludedFlags)

}

func TestEndomorphismPropertiesForXTW(t *testing.T) {
	test_endomorphism_properties(t, pointTypeXTWFull, excludeNoPoints)
	test_endomorphism_properties(t, pointTypeXTWSubgroup, excludeNoPoints)
}

func TestEndomorphismPropertiesForAXTW(t *testing.T) {
	test_endomorphism_properties(t, pointTypeAXTWFull, excludeNoPoints)
	test_endomorphism_properties(t, pointTypeAXTWSubgroup, excludeNoPoints)
}

func TestEndomorphismPropertiesForEFGH(t *testing.T) {
	test_endomorphism_properties(t, pointTypeEFGHFull, excludeNoPoints)
	test_endomorphism_properties(t, pointTypeEFGHSubgroup, excludeNoPoints)
}
