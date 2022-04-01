package pointserializer

/*
var _ pointSerializerInterface = &pointSerializerXY{}
var _ pointSerializerInterface = &pointSerializerXAndSignY{}
var _ pointSerializerInterface = &pointSerializerYAndSignX{}
*/

// var oneBitHeader bitHeader = bitHeader{prefixBits: 0b1, prefixLen: 1}

/*
func TestSerializersHasWithEndianness(t *testing.T) {
	isWithEndiannessMethod := func(fun reflect.Value, targetType reflect.Type) (good bool, reason string) {
		CloneMethodType := fun.Type()
		if CloneMethodType.NumIn() != 1 {
			return false, "supposed WithEndianness method does not take exactly 1 argument"
		}
		if CloneMethodType.NumOut() != 1 {
			return false, "supposed WithEndianness method does not return exactly 1 argument"
		}
		ArgumentType := CloneMethodType.In(0)
		var e binary.ByteOrder
		EndiannessType := reflect.TypeOf(&e).Elem()
		if !EndiannessType.AssignableTo(ArgumentType) {
			return false, "supposed WithEndianness does not take binary.ByteOrder as argument"
		}
		ReturnedType := CloneMethodType.Out(0)
		if !ReturnedType.AssignableTo(targetType) {
			return false, "supposed WithEndianness has wrong return type"
		}
		return true, ""
	}

	for _, basicSerializer := range allPointSerializersBasicHavingWithEndianness {
		serializer := reflect.ValueOf(basicSerializer)
		serializerTypeDirect := serializer.Type().Elem()
		// serializerType := reflect.TypeOf(basicSerializer)
		name := serializerTypeDirect.Name()
		WithEndiannessMethod := serializer.MethodByName(basicSerializerNewEndiannessFun)
		if !WithEndiannessMethod.IsValid() {
			t.Fatalf("Basic point serializer %v does not have valid %v method", name, basicSerializerNewEndiannessFun)
		}
		if ok, reason := isWithEndiannessMethod(WithEndiannessMethod, serializerTypeDirect); !ok {
			t.Fatalf("Basic Point serializer %v does not have valid %v method: Reason is %v", name, basicSerializerNewEndiannessFun, reason)
		}
	}
}

func TestSerializersHasWithSubgroupOnly(t *testing.T) {
	isWithSubgroupOnlyMethod := func(fun reflect.Value, targetType reflect.Type) (good bool, reason string) {
		CloneMethodType := fun.Type()
		if CloneMethodType.NumIn() != 1 {
			return false, "supposed WithSubgroupOnly method does not take exactly 1 argument"
		}
		if CloneMethodType.NumOut() != 1 {
			return false, "supposed WithSubgroupOnly method does not return exactly 1 argument"
		}
		ArgumentType := CloneMethodType.In(0)
		var b bool
		BoolType := reflect.TypeOf(b)
		if !BoolType.AssignableTo(ArgumentType) {
			return false, "supposed WithSubgroupOnly does not take bool as argument"
		}
		ReturnedType := CloneMethodType.Out(0)
		if !ReturnedType.AssignableTo(targetType) {
			return false, "supposed WithSubgroupOnly has wrong return type"
		}
		return true, ""
	}

	for _, basicSerializer := range allPointSerializersBasicHavingWithEndianness {
		serializer := reflect.ValueOf(basicSerializer)
		serializerTypeDirect := serializer.Type().Elem()
		// serializerType := reflect.TypeOf(basicSerializer)
		name := serializerTypeDirect.Name()
		withSubgroupOnlyMethod := serializer.MethodByName(basicSerializerNewSubgroupRestrict)
		if !withSubgroupOnlyMethod.IsValid() {
			t.Fatalf("Basic point serializer %v does not have valid %v method", name, basicSerializerNewSubgroupRestrict)
		}
		if ok, reason := isWithSubgroupOnlyMethod(withSubgroupOnlyMethod, serializerTypeDirect); !ok {
			t.Fatalf("Basic Point serializer %v does not have valid %v method: Reason is %v", name, basicSerializerNewSubgroupRestrict, reason)
		}
	}
}

*/

// var _ TestSample

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
