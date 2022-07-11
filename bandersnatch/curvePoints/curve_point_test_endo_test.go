package curvePoints

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// This file contains the test for the efficient degree-2 endomorphism Endo.
// We test the following properties:
//
// Endo(P) == Endo(P+A)
// Endo acts as intended on 2-torsion points
// Endo(P) or Endo(P)+A is in the subgroup.
// Endo() is non-trivial
// Endo is a group homomorphism
// Endo(P) = EV*P for P in the prime-order subgroup and EV a square root of -2 mod p253

func TestAllEndomorphismProperties(t *testing.T) {
	for _, receiverType := range allTestPointTypes {
		test_endomorphism_properties(t, receiverType, excludeNoPoints)
	}
}

func test_endomorphism_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := pointTypeToString(receiverType)
	// var type1, type2 PointType

	make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, receiverType, 50, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue
		}
		if typeCanOnlyRepresentSubgroup(receiverType) && !typeCanOnlyRepresentSubgroup(type1) {
			continue
		}
		make_samples1_and_run_tests(t, make_checkfun_endo_sane(receiverType), "Endomorphism did not pass sanity checks"+point_string, type1, 50, excludedFlags)
		make_samples1_and_run_tests(t, make_checkfun_endo_nontrivial(receiverType), "Nontriviality of Endomorphism failed for "+point_string, type1, 50, excludeNoPoints)
	}

	make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, receiverType, receiverType, 50, excludedFlags|PointFlagNAP)
	for _, type1 := range allTestPointTypes {
		for _, type2 := range allTestPointTypes {
			if typeCanOnlyRepresentSubgroup(receiverType) && (!typeCanOnlyRepresentSubgroup(type1) || !typeCanOnlyRepresentSubgroup(type2)) {
				continue
			}
			make_samples2_and_run_tests(t, make_checkfun_endo_homomorphic(receiverType), "Endomorphism is not homomorphic"+point_string, type1, type2, 50, excludedFlags|PointFlagNAP)
		}
	}
	make_samples1_and_run_tests(t, checkfun_endo_action, "Endomorphism does not act as intended "+point_string, receiverType, 50, excludedFlags)

}

// This function checks whether the endomorphism factors through P vs P+A.
// Also checks that the result of the endomorphism is in the correct subgroup (p253+{N,A}) and that it acts correctly on 2-torsion.
func make_checkfun_endo_sane(receiverType PointType) checkfunction {
	return func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		sampleType := getPointType(s.Points[0])
		var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
		var infinite bool = s.AnyFlags().CheckFlag(PointFlag_infinite)
		var result = makeCurvePointPtrInterface(receiverType)
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

		resultValidateable, ok := result.(validateable)
		if ok {
			if !resultValidateable.Validate() {
				return false, "Endo(P) is not on curve"
			}
		}

		resultClone := result.Clone()
		var X, Z FieldElement
		X = resultClone.X_projective()
		Z = resultClone.Z_projective()
		if !legendreCheckA_projectiveXZ(X, Z) {
			return false, "Result of applying endomorphism is not in 2*p253-subgroup"
		}

		if infinite {
			if !result.IsEqual(&AffineOrderTwoPoint_xtw) {
				return false, "Endo(infinite point) != affine two-torsion"
			}
		} else if s.AnyFlags().CheckFlag(PointFlag_zeroExact) {
			if !result.IsNeutralElement() {
				return false, "Endo(N) != N"
			}
		} else if s.AnyFlags().CheckFlag(PointFlag_A) {
			if !result.IsNeutralElement() {
				return false, "Endo(A) != N"
			}
		}

		if !typeCanOnlyRepresentSubgroup(sampleType) {
			inputClone := s.Points[0].Clone()
			inputClone.AddEq(&AffineOrderTwoPoint_xtw)
			resultClone.Endo(inputClone)
			if !result.IsEqual(resultClone) {
				return false, "Endo(P) != Endo(P+A)"
			}
		}

		return true, ""
	}
}

// checks that Endo is non-trivial
func make_checkfun_endo_nontrivial(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		if s.AnyFlags().CheckFlag(PointFlagNAP) {
			return true, "skipped"
		}
		neutralResultExpected := s.AnyFlags().CheckFlag(PointFlag_zeroModuloA)
		receiver := makeCurvePointPtrInterface(receiverType)
		receiver.Endo(s.Points[0])
		if !neutralResultExpected && receiver.IsNeutralElement() {
			return false, "Endo unexpectedly resulted in neutral element"
		}
		if neutralResultExpected && !receiver.IsNeutralElement() {
			return false, "Endo unexpectedly did not result in neutral element"
		}
		return true, ""
	}
	return returned_function
}

// checks whether Endo(P) + Endo(Q) == Endo(P+Q)
func make_checkfun_endo_homomorphic(receiverType PointType) (returned_function checkfunction) {
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(2)
		// This should be ruled out at the call site
		if s.AnyFlags().CheckFlag(PointFlagNAP) {
			panic("Should not call checkfun_endo_homomorphic on NaP test samples")
		}

		if !typeCanRepresentInfinity(receiverType) {
			testutils.Assert(typeCanRepresentInfinity(pointTypeXTWFull))
			var P1, P2, sumInput, sumEndo Point_xtw_full
			P1.Endo(s.Points[0])
			P2.Endo(s.Points[1])
			sumInput.Add(s.Points[0], s.Points[1])
			sumEndo.Endo(&sumInput)
			if P1.IsAtInfinity() || P2.IsAtInfinity() || sumInput.IsAtInfinity() || sumEndo.IsAtInfinity() {
				return true, "Skipped"
			}
		}

		endo1 := makeCurvePointPtrInterface(receiverType)
		endo2 := makeCurvePointPtrInterface(receiverType)
		sum := makeCurvePointPtrInterface(receiverType)
		result1 := makeCurvePointPtrInterface(receiverType)
		result2 := makeCurvePointPtrInterface(receiverType)
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
	var singular bool = s.AnyFlags().CheckFlag(PointFlagNAP)
	var p253 bool = !(s.AnyFlags().CheckFlag(PointFlag_outsideP253 | PointFlag_random))
	var good_subgroup = !(s.AnyFlags().CheckFlag(PointFlag_outsideGoodgroup | PointFlag_random))
	var random = s.AnyFlags().CheckFlag(PointFlag_random)
	// var good_subgroup = !(s.AnyFlags().CheckFlag(Case_outside_goodgroup) || s.AnyFlags().CheckFlag(Case_infinite))
	pointType := getPointType(s.Points[0])
	result1 := makeCurvePointPtrInterface(pointType)
	result1.Endo(s.Points[0])
	if result1.IsNaP() != singular {
		return false, "Running Endo_FullCurve resulted in different NaP-status than the argument"
	}
	if singular {
		// skip further tests. The relevant properties are verified by endo_sane
		return true, ""
	}
	if s.AnyFlags().CheckFlag(PointFlag_infinite) {
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

	if common.EndomorphismEigenvalue%2 == 0 {
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
		if !difference.IsEqual(&AffineOrderTwoPoint_xtw) {
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
