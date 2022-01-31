package bandersnatch

import "testing"

func TestCoordinateFunctions(t *testing.T) {
	for _, type1 := range allTestPointTypes {
		type1String := pointTypeToString(type1)
		make_samples1_and_run_tests(t, checkfun_consistency_affine_projective, "constistency of affine and projective coordinated failed for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_extended_coordinates, "extended coordinate interface is giving wrong results for "+type1String, type1, 50, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_consistency_coordinates, "coordinates do not allow to reconstruct point for "+type1String, type1, 50, excludeNoPoints)
	}
}

// check whether querying for projective XYZ resp. affine XY-coordinates allows to reconstruct point that is considered equal
func checkfun_consistency_coordinates(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	infinite := s.AnyFlags().CheckFlag(Case_infinite)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	if singular {
		return true, "skipped" // for now
	}
	clone := s.Points[0].Clone().(CurvePointPtrInterface)

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
		clone = s.Points[0].Clone().(CurvePointPtrInterface)
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
	if s.AnyFlags().CheckFlag(Case_singular | Case_infinite) {
		return true, "skipped"
	}
	clone1 := s.Points[0].Clone().(CurvePointPtrInterface)
	clone2 := s.Points[0].Clone().(CurvePointPtrInterface)
	clone3 := s.Points[0].Clone().(CurvePointPtrInterface)
	clone4 := s.Points[0].Clone().(CurvePointPtrInterface)
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
	if s.AnyFlags().CheckFlag(Case_singular) {
		return true, "skipped"
	}
	infinite := s.AnyFlags().CheckFlag(Case_infinite)
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
