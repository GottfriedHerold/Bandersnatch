package curveserialize

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch"
)

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

var allValuesSerializers []interface{} = []interface{}{
	&valuesSerializerFeFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFeHeaderFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFe{fieldElementEndianness: defaultEndianness},
	&valuesSerializerFeCompressedBit{fieldElementEndianness: defaultEndianness},
	&valuesSerializerHeaderFe{fieldElementEndianness: defaultEndianness},
}

var oneBitHeader bitHeader = bitHeader{prefixBits: 0b1, prefixLen: 1}

var allCurvePointSerializers_basic []curvePointDeserializer_basic = []curvePointDeserializer_basic{&pointSerializerXY{}, &pointSerializerXAndSignY{}, &pointSerializerYAndSignX{}, &pointSerializerXTimesSignY{}}
var allClonablePointSerializers_basic = allCurvePointSerializers_basic
var allClonablePointDeserializers_basic = allClonablePointSerializers_basic
var allPointSerializersBasicHavingWithEndianness = allClonablePointSerializers_basic
var allPointSerializersBasicHavingWithSubgroupOnly = allClonablePointDeserializers_basic

func TestSerializersHasClonable(t *testing.T) {
	for _, basicSerializer := range allClonablePointSerializers_basic {
		serializerType := reflect.TypeOf(basicSerializer)
		ok, reason := bandersnatch.DoesMethodExist(serializerType, "Clone", []reflect.Type{}, []reflect.Type{serializerType})
		if !ok {
			t.Error(reason)
		}
	}
}

func TestValuesSerializers(t *testing.T) {
	var drng *rand.Rand = rand.New(rand.NewSource(1024))
	for _, valuesSerializer := range allValuesSerializers {
		serializerV := reflect.ValueOf(valuesSerializer)
		serializerT := reflect.TypeOf(valuesSerializer)
		var typeName string = bandersnatch.GetReflectName(serializerT)
		if serializerT.Kind() != reflect.Ptr {
			t.Fatal("values serializer is not a pointer receiver")
		}
		valueSerializerFun := serializerV.MethodByName("SerializeValues")
		valueDeserializerFun := serializerV.MethodByName("DeserializeValues")
		if !valueSerializerFun.IsValid() {
			t.Fatalf("%v has no SerializeValues method", typeName)
		}
		if !valueDeserializerFun.IsValid() {
			t.Fatalf("%v has no DeserializeValues method", typeName)
		}
		var err error
		var bytesWritten int
		valuesSerializerFunType := valueSerializerFun.Type()
		numInputs := valuesSerializerFunType.NumIn()
		var inputs []reflect.Value = make([]reflect.Value, numInputs)
		var inputTypes []reflect.Type = make([]reflect.Type, numInputs)
		for j := 0; j < numInputs; j++ {
			inputTypes[j] = valuesSerializerFunType.In(j)
		}
		feType := reflect.TypeOf(&bandersnatch.FieldElement{})
		boolType := reflect.TypeOf(bool(true))
		const iterations = 20
		var buf bytes.Buffer
		inputs[0] = reflect.ValueOf(&buf)
		for i := 0; i < iterations; i++ {
			expectedLen := 0
			for j := 1; j < numInputs; j++ {
				switch inputTypes[j] {
				case feType:
					var fe bandersnatch.FieldElement
					fe.SetRandomUnsafe(drng)
					inputs[j] = reflect.ValueOf(&fe)
					expectedLen += 32

				case boolType:
					var bit bool = (drng.Intn(2) == 0)
					inputs[j] = reflect.ValueOf(bit)
				default:
					// not really an error; it's just that the test cannot accomodate this and needs to be extended.
					panic("unrecognized type to be serialized")
				}

			}
			buf.Reset()
			outputs := valueSerializerFun.Call(inputs)
			if len(outputs) != 2 {
				t.Fatalf("%v's SerializeValues method does not return 2 values", typeName)
			}
			bytesWritten = outputs[0].Interface().(int)
			if outputs[1].Interface() == nil {
				err = nil
			} else {
				err = outputs[1].Interface().(error)
			}
			if err != nil {
				t.Fatalf("%v's SerializeValues method returned err == %v", typeName, err)
			}
			if bytesWritten != expectedLen {
				t.Fatalf("%v's SerializeValues method returned %v for bytesWritten, expected %v", typeName, bytesWritten, expectedLen)
			}
			// bytes.Buffer has seperate read and write positions.
			outputs = valueDeserializerFun.Call([]reflect.Value{reflect.ValueOf(&buf)})
			if len(outputs) < 2 {
				t.Fatalf("%v's DeserializeValues methods returns <2 values", typeName)
			}

			if outputs[1].Interface() != nil {
				err = outputs[1].Interface().(error)
				t.Fatalf("For %v, Writing via SerializeValues and reading back via DeserializeValues gave error %v", typeName, err)
			}
			bytesRead := outputs[0].Interface().(int)
			if bytesRead != bytesWritten {
				t.Fatalf("%v's DeserializeValues and SerializeValues method differ in #bytes read/written", typeName)
			}
			if len(outputs)-1 != numInputs {
				t.Fatalf("For %v, DeserializeValues and SerializeValues do not have matching number of relevant arguments", typeName)
			}
			for j := 1; j < numInputs; j++ {
				switch inputTypes[j] {
				case feType:
					var feIn *bandersnatch.FieldElement = inputs[j].Interface().(*bandersnatch.FieldElement)
					var feGot bandersnatch.FieldElement = outputs[j+1].Interface().(bandersnatch.FieldElement)
					if !feIn.IsEqual(&feGot) {
						t.Fatalf("For %v, did not get back %v'th value (starting at 1) of type FieldElement via DeserializeValues that were serialized vie SerializeValues", typeName, j)
					}
				case boolType:
					var bitIn bool = inputs[j].Interface().(bool)
					var bitOut bool = outputs[j+1].Interface().(bool)
					if bitIn != bitOut {
						t.Fatalf("For %v, did not get back %v'th value (starting at 1) of type bool via DeserializeValues that were serialized vie SerializeValues", typeName, j)
					}
				default:
					// not really an error; it's just that the test cannot accomodate this and needs to be extended.
					panic("unrecognized type to be deserialized")
				}
			}
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
