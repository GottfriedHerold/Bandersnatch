package bandersnatch

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

var _ pointSerializerInterface = &pointSerializerXY{}
var _ pointSerializerInterface = &pointSerializerXAndSignY{}
var _ pointSerializerInterface = &pointSerializerYAndSignX{}

func TestRecoverYFromXAffine(t *testing.T) {
	var checkfun_recover_y checkfunction = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		assert(getPointType(s.Points[0]) == pointTypeAXTWFull)
		if s.AnyFlags().CheckFlag(Case_singular) {
			return true, "skipped"
		}
		x, y := s.Points[0].XY_affine()
		var p_flip Point_axtw_full
		p_flip.Add(s.Points[0], &AffineOrderTwoPoint_axtw)
		good_subgroup := s.Points[0].IsInSubgroup() || p_flip.IsInSubgroup()

		yRecChecked, errChecked := recoverYFromXAffine(&x, true)
		yRecUnchecked, errUnchecked := recoverYFromXAffine(&x, false)
		if errUnchecked != nil {
			return false, "RecoverYFromAffineX reported unexpected error (without Legendre check)"
		}
		if !good_subgroup {
			if errChecked != ErrXNotInSubgroup {
				return false, "RecoverYFromXAffineX did not report expected error for X not in subgroup"
			}
		} else {
			if errChecked != nil {
				return false, "RecoverYFromAffineX reported unexpected error (with Legendre check)"
			}
		}
		if ok, _ := yRecUnchecked.CmpAbs(&y); !ok {
			return false, "RecoverYFromAffineX did not reproduce Y (up to sign)"
		}
		if ok, _ := yRecChecked.CmpAbs(&y); good_subgroup && !ok {
			return false, "RecoverYFromAffineX did not reproduce Y up to sign (with Legendre check)"
		}
		return true, "ok"
	}
	make_samples1_and_run_tests(t, checkfun_recover_y, "RecoverYFromXAffine not working for good x-coos", pointTypeAXTWFull, 500, Case_singular)
	var rng *rand.Rand = rand.New(rand.NewSource(500))
	var num_good, num_notOnCurve, num_notOnSubgroup int
	const iterations = 1000
	var temp FieldElement
	for i := 0; i < iterations; i++ {
		temp.setRandomUnsafe(rng)
		_, err := recoverYFromXAffine(&temp, true)
		switch err {
		case ErrXNotInSubgroup:
			num_notOnSubgroup++
		case ErrXNotOnCurve:
			num_notOnCurve++
		case nil:
			num_good++
		default:
			err = fmt.Errorf("unexpected error returned by RecoverYFromXAffine: %w", err)
			t.Fatal(err)
		}
	}
	assert(iterations >= 1000)
	const delta int = iterations / 10
	if (num_notOnCurve > iterations/2+delta) || (num_notOnCurve < iterations/2-delta) {
		t.Fatal("Unexpected ratio of x coordinates not on curve")
	}
	if (num_notOnSubgroup > iterations/4+delta) || (num_notOnSubgroup < iterations/4-delta) {
		t.Fatal("Unexpected ratio of x coordinates on curve, but not in subgroup")
	}
	if (num_good > iterations/4+delta) || (num_good < iterations/4-delta) {
		t.Fatal("Unexpected ratio of x coordinates that are good")
	}
}

func TestRecoverXFromYAffine(t *testing.T) {
	var checkfun_recover_x checkfunction = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		assert(getPointType(s.Points[0]) == pointTypeAXTWFull)
		if s.AnyFlags().CheckFlag(Case_singular) {
			return true, "skipped"
		}
		x, y := s.Points[0].XY_affine()

		xRec, err := recoverXFromYAffine(&y)
		if err != nil {
			return false, "RecoverXFromAffineY reported error on valid y coordiate "
		}
		if ok, _ := xRec.CmpAbs(&x); !ok {
			return false, "RecoverXFromAffineY did not reproduce X up to sign"
		}
		return true, "ok"
	}
	make_samples1_and_run_tests(t, checkfun_recover_x, "RecoverXFromYAffine not working for good y-coos", pointTypeAXTWFull, 500, Case_singular)
	var yAtInfinity FieldElement = squareRootDbyA_fe
	yAtInfinity.InvEq() // sqrt(a/d)

	_, err := recoverXFromYAffine(&yAtInfinity)
	if err == nil {
		t.Fatal("recoverXFromYAffine does not produce error for Y=sqrt(d/a)")
	}
	_, err = recoverXFromYAffine(&FieldElementZero)
	if err == nil {
		t.Fatal("recoverXFromYAffine does not produce error for Y=0")
	}
	x, err := recoverXFromYAffine(&FieldElementOne)
	if err != nil {
		t.Fatal("recoverXFromYAffine reports error for y=1")
	}
	if !x.IsZero() {
		t.Fatal("recoverXFromYAffine does not return 0 for y=1")
	}
	var rng *rand.Rand = rand.New(rand.NewSource(500))
	var num_good, num_notOnCurve int
	const iterations = 1000
	var temp FieldElement
	for i := 0; i < iterations; i++ {
		temp.setRandomUnsafe(rng)
		_, err := recoverXFromYAffine(&temp)
		switch err {
		case ErrYNotOnCurve:
			num_notOnCurve++
		case nil:
			num_good++
		default:
			err = fmt.Errorf("unexpected error returned by RecoverXFromYAffine: %w", err)
			t.Fatal(err)
		}
	}
	assert(iterations >= 1000)
	const delta int = iterations / 10
	if (num_notOnCurve > iterations/2+delta) || (num_notOnCurve < iterations/2-delta) {
		t.Fatal("Unexpected ratio of x coordinates not on curve")
	}
	if (num_good > iterations/2+delta) || (num_good < iterations/2-delta) {
		t.Fatal("Unexpected ratio of x coordinates that are good")
	}
}

func TestMapToFieldElement(t *testing.T) {
	var checkfun_MapToFieldElement checkfunction = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		infinite := s.AnyFlags().CheckFlag(Case_infinite)
		zeroModA := s.AnyFlags().CheckFlag(Case_zero_moduloA)
		var result FieldElement
		didPanic := checkPanic(func(arg CurvePointPtrInterfaceRead) { result = MapToFieldElement(arg) }, s.Points[0])
		if singular && !didPanic {
			return false, "MapToFieldElement did not panic for NaP inputs"
		}
		if infinite && !didPanic {
			return false, "MapToFieldElement did not panic for infinite inputs"
		}
		if infinite || singular {
			return true, ""
		}
		if didPanic {
			return false, "MapToFieldElement unexpectedly panicked"
		}
		if !result.IsZero() == zeroModA {
			return false, "MapToFieldElement should return 0 exactly for N, A"
		}
		clone := s.Points[0].Clone()
		if !clone.CanOnlyRepresentSubgroup() {
			clone.AddEq(&AffineOrderTwoPoint_axtw)
			result2 := MapToFieldElement(clone)
			if !result.IsEqual(&result2) {
				return false, "MapToFieldElement not invariant under torsionAddA"
			}
		}

		return true, ""
	}
	for _, pointType := range allTestPointTypes {
		make_samples1_and_run_tests(t, checkfun_MapToFieldElement, "Test for MapToFieldElement failed for receiver type "+pointTypeToString(pointType), pointType, 500, excludeNoPoints)
	}
}

func TestRoundTripDeserializeFromFieldElements(t *testing.T) {
	getArgsXYAffine := func(arg CurvePointPtrInterfaceRead) (x, y *FieldElement) {
		x = new(FieldElement)
		y = new(FieldElement)
		*x, *y = arg.XY_affine()
		return
	}

	getArgsXAndSignY := func(arg CurvePointPtrInterfaceRead) (x *FieldElement, signY int) {
		x_val, y_val := arg.XY_affine()
		x = new(FieldElement)
		*x = x_val
		signY = y_val.Sign()
		return
	}

	getArgsYAndSignX := func(arg CurvePointPtrInterfaceRead) (y *FieldElement, signX int) {
		x_val, y_val := arg.XY_affine()
		y = new(FieldElement)
		*y = y_val
		signX = x_val.Sign()
		return
	}

	getArgsXTimesSignY := func(arg CurvePointPtrInterfaceRead) (xSignY *FieldElement) {
		x_val, y_val := arg.XY_affine()
		xSignY = new(FieldElement)
		*xSignY = x_val
		signY := y_val.Sign()
		if signY == -1 {
			xSignY.NegEq()
		} else {
			assert(signY == 1)
		}
		return
	}

	getArgsXYTimesSignY := func(arg CurvePointPtrInterfaceRead) (xSignY, ySignY *FieldElement) {
		x_val, y_val := arg.XY_affine()
		xSignY = new(FieldElement)
		ySignY = new(FieldElement)
		signY := y_val.Sign()
		*xSignY = x_val
		*ySignY = y_val
		if signY == -1 {
			xSignY.NegEq()
			ySignY.NegEq()
		} else {
			assert(signY == +1)
		}
		return
	}

	checkfun_FullCurvePointFromXYAffine := make_checkfun_recoverPoint(CurvePointFromXYAffine_full, "FullCurvePointFromXYAffine", false, getArgsXYAffine, false)
	checkfun_SubgroupCurvePointFromXYAffine := make_checkfun_recoverPoint(CurvePointFromXYAffine_subgroup, "SubgroupCurvePointFromXYAffine", true, getArgsXYAffine, false)
	checkfun_FullCurvePointFromXAndSignY := make_checkfun_recoverPoint(CurvePointFromXAndSignY_full, "FullCurvePointFromXAndSignY", false, getArgsXAndSignY, false)
	checkfun_SubgroupCurvePointFromXAndSignY := make_checkfun_recoverPoint(CurvePointFromXAndSignY_subgroup, "SubgroupCurvePointFromXAndSignY", true, getArgsXAndSignY, false)
	checkfun_FullCurvePointFromYAndSignX := make_checkfun_recoverPoint(CurvePointFromYAndSignX_full, "FullCurvePointFromYAndSignX", false, getArgsYAndSignX, false)
	checkfun_SubgroupCurvePointFromYAndSignX := make_checkfun_recoverPoint(CurvePointFromYAndSignX_subgroup, "SubgroupCurvePointFromYAndSignX", true, getArgsYAndSignX, false)
	checkfun_SubgroupCurvePointFromXTimesSignY := make_checkfun_recoverPoint(CurvePointFromXTimesSignY_subgroup, "SubgroupCurvePointFromXTimesSignY", true, getArgsXTimesSignY, true)
	checkfun_SubgroupCurvePointFromXYTimesSignY := make_checkfun_recoverPoint(CurvePointFromXYTimesSignY_subgroup, "SubgroupCurvePointFromXYTimesSignY", true, getArgsXYTimesSignY, true)

	for _, pointType := range allTestPointTypes {
		pointString := pointTypeToString(pointType)
		make_samples1_and_run_tests(t, checkfun_recoverFromXYAffine, "Failure to recover point from XYAffine for "+pointString, pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_FullCurvePointFromXYAffine, "Failure to recover point from FullCurveFromXYAffine for "+pointString, pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_SubgroupCurvePointFromXYAffine, "Failure to recover point from SubgroupCurveFromXYAffine for "+pointString, pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_FullCurvePointFromXAndSignY, "Failure to recover point from FullCurvePointFromXAndSignY for "+pointString, pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_SubgroupCurvePointFromXAndSignY, "Failure to recover point from SubgroupCurvePointFromXAndSignY for "+pointString, pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_FullCurvePointFromYAndSignX, "Failure to recover point from FullCurvePointFromYAndSignX", pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_SubgroupCurvePointFromYAndSignX, "Failure to recover point from SubgroupCurvePointFromYAndSignX", pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_SubgroupCurvePointFromXTimesSignY, "Failure to recover point from SubgroupCurvePointFromXTimesSignY", pointType, 200, excludeNoPoints)
		make_samples1_and_run_tests(t, checkfun_SubgroupCurvePointFromXYTimesSignY, "Failure to recover point from SubgroupCurvePointFromXYTimesSignY", pointType, 200, excludeNoPoints)
	}
}

func checkfun_recoverFromXYAffine(s *TestSample) (bool, string) {
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
	point, err := CurvePointFromXYAffine_full(&x, &y, TrustedInput)
	if err != nil {
		return false, "FullCurvePointFromXYAffine reported unexpected error (TrustedInput)"
	}
	if !point.IsEqual(s.Points[0]) {
		return false, "FullCurvePointFromXYAffine did not recover point (TrustedInput)"
	}
	point, err = CurvePointFromXYAffine_full(&x, &y, UntrustedInput)
	if err != nil {
		return false, "FullCurvePointFromXYAffine reported unexpected error (UntrustedInput)"
	}
	if !point.IsEqual(s.Points[0]) {
		return false, "FullCurvePointFromXYAffine did not recover point (UntrustedInput)"
	}
	point_subgroup, err := CurvePointFromXYAffine_subgroup(&x, &y, UntrustedInput)
	if !subgroup {
		if err == nil {
			return false, "SubgroupCurvePointFromXYAffine did not report subgroup error"
		}
	} else {
		if err != nil {
			return false, "SubgroupCurvePointFromXYAffine reported unexpected error"
		}
		if !point_subgroup.IsEqual(s.Points[0]) {
			return false, "SubgroupCurvePointFromXYAffine did not recover point (UntrustedInput)"
		}
	}
	if subgroup {
		point_subgroup, err = CurvePointFromXYAffine_subgroup(&x, &y, TrustedInput)
		if err != nil {
			return false, "SubgroupCurvePointFromXYAffine reported unexpected error (TrustedInput)"
		}
		if !point_subgroup.IsEqual(s.Points[0]) {
			return false, "SubgroupCurvePointFromXYAffine did not recover point (TrustedInput)"
		}
	}
	return true, ""
}

func make_checkfun_recoverPoint(recoveryFun interface{}, name string, subgroupOnly bool, argGetter interface{}, roundTripModuloA bool) (returned_function checkfunction) {
	recoveryFun_r := reflect.ValueOf(recoveryFun)
	argGetter_r := reflect.ValueOf(argGetter)
	assert(recoveryFun_r.Kind() == reflect.Func)
	assert(argGetter_r.Kind() == reflect.Func)
	returned_function = func(s *TestSample) (bool, string) {
		s.AssertNumberOfPoints(1)
		singular := s.AnyFlags().CheckFlag(Case_singular)
		infinite := s.AnyFlags().CheckFlag(Case_infinite)
		subgroup := s.Points[0].IsInSubgroup()
		var pointPlusA Point_xtw_full // only used if roundTripModuloA is true
		if roundTripModuloA {
			assert(subgroupOnly)
			pointPlusA.SetFrom(s.Points[0])
			pointPlusA.AddEq(&AffineOrderTwoPoint_axtw)
			subgroup = subgroup || pointPlusA.IsInSubgroup()
		}
		if infinite {
			return true, "skipped" // affine X,Y coos make no sense.
		}
		if singular {
			return true, "skipped" // We can't reliably get coos from the point
		}
		point_r := reflect.ValueOf(s.Points[0])
		var argGetterInput [1]reflect.Value = [1]reflect.Value{point_r}
		args_r := argGetter_r.Call(argGetterInput[:])
		Trusted_r := reflect.ValueOf(TrustedInput)
		Untrusted_r := reflect.ValueOf(UntrustedInput)
		args_r = append(args_r, Untrusted_r)
		res_r := recoveryFun_r.Call(args_r)
		// Voodoo to take the adress of the return value of a reflect.Call. We need a new variable of pointer type, allocate memory and copy.
		pointPtr_r := reflect.New(res_r[0].Type())
		pointPtr_r.Elem().Set(res_r[0])
		point := pointPtr_r.Interface().(CurvePointPtrInterfaceRead)
		var err error
		res_r1 := res_r[1].Interface()
		if res_r1 == nil {
			err = nil
		} else {
			err = res_r1.(error)
		}
		if subgroupOnly {
			if subgroup && err != nil {
				return false, "Unexpected error encountered when recovering point with " + name + " (UntrustedInput)"
			}
			if !subgroup && err == nil {
				return false, "Untrusted deserialization of non-subgroup input to subgroup point did not result in error for " + name
			}
		} else if err != nil {
			return false, "Unexpected error reported when recovering point with " + name + " (UntrustedInput)"
		}

		if !roundTripModuloA && err == nil && !point.IsEqual(s.Points[0]) {
			return false, "Untrusted deserialization did not reproduce the original for " + name
		}
		if roundTripModuloA && err == nil {
			if !point.IsEqual(s.Points[0]) && !point.IsEqual(&pointPlusA) {
				return false, "Untrusted deserialization did not reproduce the original modulo A for " + name
			}
		}
		if subgroupOnly && !subgroup {
			return true, ""
		}
		args_r[len(args_r)-1] = Trusted_r
		res_r = recoveryFun_r.Call(args_r)
		pointPtr_r = reflect.New(res_r[0].Type())
		pointPtr_r.Elem().Set(res_r[0])
		point = pointPtr_r.Interface().(CurvePointPtrInterfaceRead)
		res_r1 = res_r[1].Interface()
		if res_r1 == nil {
			err = nil
		} else {
			err = res_r1.(error)
		}
		if err != nil {
			return false, "Unexpected error reported when recovering point with " + name + " (TrustedInput)"
		}
		if !roundTripModuloA && !point.IsEqual(s.Points[0]) {
			return false, "TrustedInput deserialization did not reproduce the original for " + name
		}
		if roundTripModuloA && !point.IsEqual(s.Points[0]) && !point.IsEqual(&pointPlusA) {
			return false, "TrustedInput deserialization did not reproduce the original modulo A for " + name
		}
		return true, ""

	}
	return
}

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
