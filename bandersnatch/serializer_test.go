package bandersnatch

var _ pointSerializerInterface = &pointSerializerXY{}
var _ pointSerializerInterface = &pointSerializerXAndSignY{}
var _ pointSerializerInterface = &pointSerializerYAndSignX{}

/*
func checkfun_recoverFromXAndSignY(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.AnyFlags().CheckFlag(Case_singular)
	infinite := s.AnyFlags().CheckFlag(Case_infinite)
	subgroup := s.Points[0].IsInSubgroup()
	if infinite {
		return true, "skipped" // affine X,Y coos make no sense.
	}
	if singular {
		return true, "skipped" // We can't reliably get coos from the point
	}
	x, y := s.Points[0].XY_affine()
	signY := y.Sign()
	point, err := FullCurvePointFromXAndSignY(&x, signY, TrustedInput)
	if err != nil {
		return false, "FullCurvePointFromXAndSignY reported unexpected error (TrustedInput)"
	}
	if !point.IsEqual(s.Points[0]) {
		return false, "FullCurvePointFromXAndSignY did not recover point (TrustedInput)"
	}
	point, err = FullCurvePointFromXAndSignY(&x, signY, UntrustedInput)
	if err != nil {
		return false, "FullCurvePointFromXAndSignY reported unexpected error (UntrustedInput)"
	}
	if !point.IsEqual(s.Points[0]) {
		return false, "FullCurvePointFromXAndSignY did not recover point (UntrustedInput)"
	}
	point_subgroup, err := FullCurvePointFromXAndSignY(&x, signY, UntrustedInput)
	if !subgroup {
		if err == nil {
			return false, "FullCurvePointFromXAndSignY did not report subgroup error"
		}
	} else {
		if err != nil {
			return false, "FullCurvePointFromXAndSignY reported unexpected error"
		}
		if !point_subgroup.IsEqual(s.Points[0]) {
			return false, "SubgroupCurvePointFromXYAffine did not recover point (UntrustedInput)"
		}
	}
	if subgroup {
		point_subgroup, err = SubgroupCurvePointFromXYAffine(&x, &y, TrustedInput)
		if err != nil {
			return false, "SubgroupCurvePointFromXYAffine reported unexpected error (TrustedInput)"
		}
		if !point_subgroup.IsEqual(s.Points[0]) {
			return false, "SubgroupCurvePointFromXYAffine did not recover point (TrustedInput)"
		}
	}
	return true, ""
}
*/
