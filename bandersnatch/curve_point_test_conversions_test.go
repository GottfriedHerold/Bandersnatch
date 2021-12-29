package bandersnatch

import "testing"

// checks whether AffineExtended gives a point that is considered equal to the original.
func checkfun_conversion_to_affine(s TestSample) (ok bool, error_reason string) {
	s.AssertNumberOfPoints(1)
	var singular bool = s.Flags[0].CheckFlag(Case_singular)
	var infinite bool = s.Flags[0].CheckFlag(Case_infinite)
	var affine_point Point_axtw
	if singular {
		affine_point = s.Points[0].AffineExtended()
		return affine_point.IsNaP(), "conversion to affine of NaP does not result in NaP"
	}
	if infinite {
		// return true, "" // FIXME
		ok = true // return value in case of a recover()'ed panic
		defer func() { recover() }()
		affine_point = s.Points[0].AffineExtended()
		return affine_point.IsNaP(), "conversion to affine of ininite point neither panics nor results in NaP"
	}
	affine_point = s.Points[0].AffineExtended()
	return affine_point.IsEqual_FullCurve(s.Points[0]), "conversion to affine point does not equal the original"
}

// checks whether ExtendedTwistedEdwards gives a point that is considered equal to the original
func checkfun_conversion_to_xtw(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular bool = s.Flags[0].CheckFlag(Case_singular)
	var point_xtw Point_xtw
	if singular {
		point_xtw = s.Points[0].ExtendedTwistedEdwards()
		return point_xtw.IsNaP(), "conversion of NaP to xtw point did not result in NaP"
	}
	point_xtw = s.Points[0].ExtendedTwistedEdwards()
	return point_xtw.IsEqual_FullCurve(s.Points[0]), "conversion to xtw did not result in point that was considered equal"
}

// checks whether Clone() gives a point that is considered equal to the original.
func checkfun_clone(s TestSample) (ok bool, err string) {
	s.AssertNumberOfPoints(1)
	var singular bool = s.Flags[0].CheckFlag(Case_singular)

	var point_copy CurvePointPtrInterface_FullCurve = s.Points[0].Clone().(CurvePointPtrInterface_FullCurve)

	if singular != point_copy.IsNaP() {
		return false, "cloning did not result in the same NaP status as the original"
	}

	if singular {
		ok, err = guardForInvalidPoints(false, true, "error when comparing clone of NaP to original", point_copy.IsEqual_FullCurve, s.Points[0])
	} else {
		ok, err = guardForInvalidPoints(true, false, "error when comparing clone of non-NaP to original", point_copy.IsEqual_FullCurve, s.Points[0])
	}
	if !ok {
		return
	}

	// modify point_copy and try again to make sure clone is not tied to the original (Note that CurvePointPtrInterface's concrete values are pointers)
	point_copy.AddEq(&example_generator_xtw) // shouldn't change NaP - status as example_generator + E1 is not among our sample set.
	expected := false

	return guardForInvalidPoints(expected, singular, "Clone of point was equal, even after copy was modified", point_copy.IsEqual_FullCurve, s.Points[0])
}

// checks whether projective coo functions work as expected
func checkfun_projective_coordinate_queries(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	p := s.Points[0].Clone() // <foo>_projective might change the point.
	var X FieldElement = p.X_projective()
	var Y FieldElement = p.Y_projective()
	var Z FieldElement = p.Z_projective()

	if s.Flags[0].CheckFlag(Case_infinite) {
		if !Z.IsZero() {
			return false, "Z_projective did not return 0 for infinite point"
		}
		// points at infinity (happen to) have Y-coo 0.
		if !Y.IsZero() {
			return false, "Y_projective did not return 0 for infinite point"
		}
		// Skip further tests: X,Y,Z are insufficient to reconstruct points at infinity anyway, unless we identify P with P+A.
		return true, ""
	}

	var T FieldElement
	T.Mul(&X, &Y) // want to set T to X*Y / Z
	// To divide by Z, rescale instead
	X.MulEq(&Z)
	Y.MulEq(&Z)
	Z.SquareEq()
	var point_copy Point_xtw = Point_xtw{x: X, y: Y, t: T, z: Z}
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = !singular

	return guardForInvalidPoints(expected, singular, "Reconstruction of point via <foo>_projective coos failed", point_copy.IsEqual_FullCurve, s.Points[0])
}

// checks whether affine coo function work as expected
func checkfun_affine_coordinate_queries(s TestSample) (ok bool, err string) {
	s.AssertNumberOfPoints(1)
	p := s.Points[0].Clone() // we operate on a copy, because querying affine coos might modify the point.
	if s.Flags[0].CheckFlag(Case_infinite) {
		// We expect a panic
		ok = true // return value in case of a recover()'ed panic
		err = ""
		defer func() { recover() }()
		_ = p.X_affine()
		_ = p.Y_affine()
		return false, "Calling X_affine and Y_affine on infinite point did not panic"
	}

	var X FieldElement = p.X_affine()
	var Y FieldElement = p.Y_affine()
	var T FieldElement
	T.Mul(&X, &Y) // want to set T to X*Y (since Z==1 on Point_axtw)
	var point_copy Point_axtw = Point_axtw{x: X, y: Y, t: T}
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = !singular
	return guardForInvalidPoints(expected, singular, "Reconstruction of point via <foo>_projective coos failed", point_copy.IsEqual_FullCurve, s.Points[0])
}

func test_conversion_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)

	make_samples1_and_run_tests(t, checkfun_conversion_to_affine, "Conversion to affine did not work "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_conversion_to_xtw, "Conversion to xtw did not work "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_projective_coordinate_queries, "Projective coordinate queries did not work"+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_affine_coordinate_queries, "Affine coordinate queries did not work"+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_clone, "cloning did not work"+point_string, receiverType, 10, excludedFlags)
}

func TestConversionForXTW(t *testing.T) {
	test_conversion_properties(t, pointTypeXTW, 0)
}

func TestConversionForAXTW(t *testing.T) {
	test_conversion_properties(t, pointTypeAXTW, 0)
}

func TestConverstionForEFGH(t *testing.T) {
	test_conversion_properties(t, pointTypeEFGH, 0)
}
