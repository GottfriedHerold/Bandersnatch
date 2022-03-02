package bandersnatch

import "testing"

// This test file checks whether the various coordinate retrieval functions are consistent with each other and allow reconstructing the point.
// We check:
//
// Reconstructing the point for projective / affine coos gives back point that is considered equal.
// consistency of affine and projective coos (projective are multiple of affine)
// affine coos and affine_decaf coos are consistent (i.e. either give the same point or differ by A) when applicable
// projective coos and projective_decaf coos are consistent (i.e. either give the same point or differ by A) when applicable
// extended coos are consistent (i.e. T=X*Y/Z)
// Multi-coo functions like XY_affine etc. are consistent with X_affine and Y_affine
//

func TestCoordinateFunctions(t *testing.T) {
	for _, type1 := range allTestPointTypes {
		type1String := pointTypeToString(type1)
		make_samples1_and_run_tests(t, checkfun_consistency_affine_projective, "constistency of affine and projective coordinated failed for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_extended_coordinates, "extended coordinate interface is giving wrong results for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_coordinates, "coordinates do not allow to reconstruct point for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_decaf_affine, "affine coordinates do not match with decaf_affine coordinates for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_decaf_projective, "projective coordinates do not match with decaf_projective coordinates for "+type1String, type1, 50, excludeNoPoints)
	}
}

// check whether querying for projective XYZ resp. affine XY-coordinates allows to reconstruct point that is considered equal
func checkfun_consistency_coordinates(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	infinite := s.AnyFlags().CheckFlag(PointFlag_infinite)
	singular := s.AnyFlags().CheckFlag(PointFlagNAP)
	if singular {
		return true, "skipped" // for now
	}
	clone := s.Points[0].Clone()

	if !infinite {
		var Pp Point_xtw_full
		Pp.x, Pp.y, Pp.z = clone.XYZ_projective()
		Pp.t.Mul(&Pp.x, &Pp.y)
		Pp.x.MulEq(&Pp.z)
		Pp.y.MulEq(&Pp.z)
		Pp.z.SquareEq()
		if !Pp.Validate() {
			return false, "reconstructed projective point did not Validate"
		}
		if !Pp.IsEqual(s.Points[0]) {
			return false, "projective coordinates do not give back point"
		}
		var Pa Point_axtw_full
		clone = s.Points[0].Clone()
		Pa.x, Pa.y = clone.XY_affine()
		Pa.t.Mul(&Pa.x, &Pa.y)
		if !Pa.Validate() {
			return false, "reconstructed affine point did not Validate"
		}
		if !Pa.IsEqual(s.Points[0]) {
			return false, "affine coordinates do not give back point"
		}
	} else {
		var coo FieldElement
		coo = s.Points[0].Y_projective()
		if !coo.IsZero() {
			return false, "Y coordinate non-zero on infinite curve point"
		}
		coo = s.Points[0].Z_projective()
		if !coo.IsZero() {
			return false, "Z coordinate non-zero on infinite curve point"
		}
	}
	return true, ""
}

// check whether X_affine, Y_affine, XY_affine, X_projective, Y_projective, Z_projective, XYZ_projective are consistent
func checkfun_consistency_affine_projective(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(PointFlagNAP | PointFlag_infinite) {
		return true, "skipped"
	}
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone()
	clone3 := s.Points[0].Clone()
	clone4 := s.Points[0].Clone()
	x_proj := clone1.X_projective()
	y_proj := clone1.Y_projective()
	z_proj := clone1.Z_projective()
	x_proj2, y_proj2, z_proj2 := clone4.XYZ_projective()
	if z_proj.IsZero() {
		return false, "finite point has Z-coordinate 0"
	}
	x_affine := clone2.X_affine()
	y_affine := clone2.Y_affine()
	x_affine2, y_affine2 := clone3.XY_affine()
	if !x_affine.IsEqual(&x_affine2) {
		return false, "X_affine and XY_affine disagree"
	}
	if !y_affine.IsEqual(&y_affine2) {
		return false, "Y_affine and XY_affine disagree"
	}
	x_affine.MulEq(&z_proj)
	y_affine.MulEq(&z_proj)
	if !x_affine.IsEqual(&x_proj) {
		return false, "X_projective and X_affine do not match"
	}
	if !y_affine.IsEqual(&y_proj) {
		return false, "Y_projective and Y_affine do not match"
	}
	x_proj.MulEq(&z_proj2)
	y_proj.MulEq(&z_proj2)
	x_proj2.MulEq(&z_proj)
	y_proj2.MulEq(&z_proj)
	if !x_proj.IsEqual(&x_proj2) {
		return false, "X_projective and XYZ_projective are inconsistent"
	}
	if !y_proj.IsEqual(&y_proj2) {
		return false, "Y_projective and XYZ_projective are inconsistent"
	}
	return true, ""
}

// Checks consistency of the extended coordinate interface (i.e. having T coordinates) with the other coos when applicable
func checkfun_consistency_extended_coordinates(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(PointFlagNAP) {
		return true, "skipped"
	}
	infinite := s.AnyFlags().CheckFlag(PointFlag_infinite)
	if _, ok := s.Points[0].(CurvePointPtrInterfaceCooReadExtended); !ok {
		return true, "skipped"
	}
	clone := s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
	t_proj := clone.T_projective()
	clone = s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
	x_proj, y_proj, z_proj := clone.XYZ_projective()
	clone = s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
	x_proj2, y_proj2, t_proj2, z_proj2 := clone.XYTZ_projective()
	if !infinite {
		clone1 := s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
		clone2 := s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
		clone3 := s.Points[0].Clone().(CurvePointPtrInterfaceCooReadExtended)
		x_affine, y_affine := clone1.XY_affine()
		t_affine := clone2.T_affine()
		x_affine2, y_affine2, t_affine2 := clone3.XYT_affine()
		if !x_affine.IsEqual(&x_affine2) {
			return false, "X_affine and XYT_affine do not match"
		}
		if !y_affine.IsEqual(&y_affine2) {
			return false, "Y_affine and XYT_affine do not match"
		}
		if !t_affine.IsEqual(&t_affine2) {
			return false, "T_affine and XYT_affine do not match"
		}
		var temp FieldElement
		temp.Mul(&x_affine, &y_affine)
		if !temp.IsEqual(&t_affine) {
			return false, "T_affine != X_affine * Y_affine"
		}
		t_affine.MulEq(&z_proj)
		if !t_affine.IsEqual(&t_proj) {
			return false, "T_affine and T_projective do not match"
		}
	}
	var temp1, temp2 FieldElement
	temp1.Mul(&x_proj, &y_proj)
	temp2.Mul(&t_proj, &z_proj)
	if !temp1.IsEqual(&temp2) {
		return false, "X*Y == T*Z not satisfied"
	}
	t_proj.MulEq(&z_proj2)
	x_proj.MulEq(&z_proj2)
	y_proj.MulEq(&z_proj2)
	t_proj2.MulEq(&z_proj)
	x_proj2.MulEq(&z_proj)
	y_proj2.MulEq(&z_proj)
	if !t_proj.IsEqual(&t_proj2) {
		return false, "T_projective and XYTZ_projective do not match"
	}
	if !x_proj.IsEqual(&x_proj2) {
		return false, "X_projective and XYTZ_projective do not match"
	}
	if !y_proj.IsEqual(&y_proj2) {
		return false, "Y_projecitve and XYTZ_projective do not match"
	}
	return true, ""
}

// check consistency of _decaf_affine with _affine
func checkfun_consistency_decaf_affine(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(PointFlagNAP | PointFlag_infinite) {
		return true, "Skipped"
	}
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone()
	Xd := clone1.X_decaf_affine()
	Yd := clone1.Y_decaf_affine()
	Td := clone1.T_decaf_affine()
	X, Y := clone2.XY_affine()
	var T FieldElement
	T.Mul(&X, &Y)
	// check Y first to get sign, because it cannot be zero.
	var correctsign, ok bool
	ok, correctsign = Y.CmpAbs(&Yd)
	if !ok {
		return false, "Y_affine and Y_decaf_affine do not match up to sign"
	}
	// We cannot use CmpAbs on X directly due to the 0 case.
	// So we flip the sign and expect an exact match (we use CmpAbs for better errors)
	if !correctsign {
		X.NegEq()
	}
	ok, correctsign = X.CmpAbs(&Xd)
	if !ok {
		return false, "X_affine and X_decaf_affine do not match"
	}
	if !correctsign {
		return false, "Both X_affine and X_decaf_affine and Y_affine and Y_decaf_affine match individually, but they differ in relative sign."
	}
	// T needs to match exactly in any case.
	if !T.IsEqual(&Td) {
		return false, "T_affine and T_decaf_affine do not match"
	}
	return true, ""
}

// check consistency of _decaf_projective and _projective
func checkfun_consistency_decaf_projective(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	if s.AnyFlags().CheckFlag(PointFlagNAP) {
		return true, "skipped"
	}
	infinite := s.AnyFlags().CheckFlag(PointFlag_infinite)
	// We treat the case of points at infinity completely separately
	if infinite {
		Xd := s.Points[0].X_decaf_projective()
		Yd := s.Points[0].Y_decaf_projective()
		Td := s.Points[0].T_decaf_projective()
		Zd := s.Points[0].Z_decaf_projective()
		if !Zd.IsZero() {
			return false, "Z_decaf_projective != 0 for point at infinity"
		}
		if !Yd.IsZero() {
			return false, "Y_decaf_projective != 0 for point at infinity"
		}
		if Xd.IsZero() {
			return false, "X_decaf_projective == 0 for point at infinity"
		}
		if Td.IsZero() {
			return false, "T_decaf_projective == 0 for point at infinity"
		}
		Td.MulEq(&squareRootDbyA_fe)
		ok, _ := Td.CmpAbs(&Xd)
		if !ok {
			return false, "X_decaf_projective / T_decaf_projective != sqrt(d/a) for point at infinity"
		}
		return true, ""
	}
	assert(!infinite) // We treated the infinite case above
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone()
	X, Y, Z := clone1.XYZ_projective()
	var T FieldElement
	T.Mul(&X, &Y)
	X.MulEq(&Z)
	Y.MulEq(&Z)
	Z.SquareEq()
	Xd := clone2.X_decaf_projective()
	Yd := clone2.Y_decaf_projective()
	Td := clone2.T_decaf_projective()
	Zd := clone2.Z_decaf_projective()
	if Z.IsZero() {
		return false, "Z_projective returned 0 for non-infinite point"
	}
	if Zd.IsZero() {
		return false, "Z_decaf_projective returned 0 for non-infinite point"
	}
	// We check X/Z = +/- Xd/Zd etc. Clearing denominators (which are non-zero)
	Xd.MulEq(&Z)
	Yd.MulEq(&Z)
	Td.MulEq(&Z)
	X.MulEq(&Zd)
	Y.MulEq(&Zd)
	T.MulEq(&Zd)
	// check Y first to get sign (Y cannot be zero):
	if Yd.IsZero() {
		return false, "Y_decaf_projective returned 0 for non-infinite point"
	}
	ok, correctsign := Yd.CmpAbs(&Y)
	if !ok {
		return false, "Y_decaf_projective and Y_projective do not match"
	}
	if !correctsign {
		Xd.NegEq()
	}
	ok, correctsign = Xd.CmpAbs(&X)
	if !ok {
		return false, "X_decaf_projective and X_projective do not match"
	}
	if !correctsign {
		return false, "<foo>_decaf_projective and <foo>_projecive for <foo> in X,Y,Z make inconsistent choices wrt P vs. P+A"
	}
	if !T.IsEqual(&Td) {
		return false, "T_decaf_projective and T_projective do not match"
	}
	return true, ""
}
