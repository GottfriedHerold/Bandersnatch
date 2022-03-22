package curveserialize

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// import "github.com/GottfriedHerold/Bandersnatch/bandersnatch"

/*
var _ pointSerializerInterface = &pointSerializerXY{}
var _ pointSerializerInterface = &pointSerializerXAndSignY{}
var _ pointSerializerInterface = &pointSerializerYAndSignX{}
*/

var _ curvePointDeserializer_basic = &pointSerializerXY{}
var _ curvePointDeserializer_basic = &pointSerializerXAndSignY{}
var _ curvePointDeserializer_basic = &pointSerializerYAndSignX{}
var _ curvePointDeserializer_basic = &pointSerializerXTimesSignY{}
var _ curvePointDeserializer_basic = &pointSerializerXYTimesSignY{}

var _ curvePointSerializer_basic = &pointSerializerXY{}
var _ curvePointSerializer_basic = &pointSerializerXAndSignY{}
var _ curvePointSerializer_basic = &pointSerializerYAndSignX{}
var _ curvePointSerializer_basic = &pointSerializerXTimesSignY{}
var _ curvePointSerializer_basic = &pointSerializerXYTimesSignY{}

type subgroupRestrictionInterface interface {
	SetSubgroupRestriction(bool)
	IsSubgroupOnly() bool
}

var _ subgroupRestrictionInterface = &subgroupRestriction{}
var _ subgroupRestrictionInterface = &subgroupOnly{}

var allValuesSerializers []interface{} = []interface{}{&valuesSerializerFeFe{}, &valuesSerializerHeaderFeHeaderFe{}, &valuesSerializerFe{}, &valuesSerializerFeCompressedBit{}, &valuesSerializerHeaderFe{}}

var allCurvePointSerializers_basic []curvePointDeserializer_basic = []curvePointDeserializer_basic{&pointSerializerXY{}, &pointSerializerXAndSignY{}, &pointSerializerYAndSignX{}, &pointSerializerXTimesSignY{}}
var allClonablePointSerializers_basic = allCurvePointSerializers_basic
var allClonablePointDeserializers_basic = allClonablePointSerializers_basic
var allPointSerializersBasicHavingWithEndianness = allClonablePointSerializers_basic
var allPointSerializersBasicHavingWithSubgroupOnly = allClonablePointDeserializers_basic

func isCloneMethod(fun reflect.Value, targetType reflect.Type) (good bool, reason string) {
	CloneMethodType := fun.Type()
	if CloneMethodType.NumIn() != 0 {
		return false, "supposed Clone method takes >0 arguments"
	}
	if CloneMethodType.NumOut() != 1 {
		return false, "supposed Clone method returns != 1 argments"
	}
	ReturnedType := CloneMethodType.Out(0)
	if !ReturnedType.AssignableTo(targetType) {
		return false, "supposed Clone function's returned type is not assignable to the given targetType"
	}
	return true, ""
}

func TestSerializersHasClonable(t *testing.T) {
	for _, basicSerializer := range allClonablePointSerializers_basic {
		serializer := reflect.ValueOf(basicSerializer)
		serializerType := reflect.TypeOf(basicSerializer)
		name := serializerType.Elem().Name()
		CloneMethod := serializer.MethodByName(basicSerializerCloneFun)
		if !CloneMethod.IsValid() {
			t.Fatalf("Basic Point serializer %v does not have Clone method", name)
		}
		if ok, reason := isCloneMethod(CloneMethod, serializerType); !ok {
			t.Fatalf("Basic Point serializer %v not clonable with reason %v", name, reason)
		}
	}
}

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
