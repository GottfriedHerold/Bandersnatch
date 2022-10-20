package pointserializer

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// TODO: Add list of all ParameterAware types and run check for DoesMethodExist

type dummyParameterAware_Invalid struct {
}

var dummyRecognizedParams = []string{"Param1", "Param2", "Param3"}

func (*dummyParameterAware_Invalid) RecognizedParameters() []string {
	return dummyRecognizedParams
}

func (*dummyParameterAware_Invalid) HasParameter(paramName string) bool {
	return utils.ElementInList(paramName, dummyRecognizedParams, normalizeParameter)
}

func (*dummyParameterAware_Invalid) GetParameter(paramName string) any {
	switch normalizeParameter(paramName) {
	case normalizeParameter("Param1"):
		return "Value1"
	case normalizeParameter("Param2"):
		return "Value2"
	case normalizeParameter("Param3"):
		return 3
	default:
		panic("Invalid")
	}
}

type dummyParameterAware struct {
	dummyParameterAware_Invalid
	value1 string
	value2 string
	value3 int
}

func (d *dummyParameterAware) GetParameter(paramName string) any {
	switch normalizeParameter(paramName) {
	case normalizeParameter("Param1"):
		return d.value1
	case normalizeParameter("Param2"):
		return d.value2
	case normalizeParameter("Param3"):
		return d.value3
	default:
		panic("Invalid")
	}
}

func (d *dummyParameterAware) WithParameter(paramName string, newParam any) *dummyParameterAware {
	d2 := *d
	switch normalizeParameter(paramName) {
	case normalizeParameter("Param1"):
		d2.value1 = newParam.(string)
	case normalizeParameter("Param2"):
		d2.value2 = newParam.(string)
	case normalizeParameter("Param3"):
		d2.value3 = newParam.(int)
	default:
		panic("Invalid")
	}
	return &d2
}

type dummyParameterAware2 struct {
	dummyParameterAware
}

func (d *dummyParameterAware2) WithParameter(paramName string, newParam any) ParameterAware {
	super := d.dummyParameterAware.WithParameter(paramName, newParam)
	ret := dummyParameterAware2{dummyParameterAware: *super}
	return &ret
}

var _ ParameterAware = &dummyParameterAware_Invalid{}
var _ ParameterAware = &dummyParameterAware{}
var _ ParameterAware = &dummyParameterAware2{}

func TestWithParameterFreeFunction(t *testing.T) {
	var dInvalid dummyParameterAware_Invalid
	var d1 dummyParameterAware
	var d2 dummyParameterAware2
	d1.value3 = 1
	d2.value3 = 1

	DidPanic := testutils.CheckPanic(WithParameter[*dummyParameterAware_Invalid], &dInvalid, "Param1", "foo")
	testutils.FatalUnless(t, DidPanic, "WithParameter did not fail for missing ParameterAware functionality")

	DidPanic = testutils.CheckPanic(WithParameter[*dummyParameterAware], &d1, "InvalidParam", "foo")
	testutils.FatalUnless(t, DidPanic, "WithParameter did not fail for invalid parameter")

	d1_modified := WithParameter(&d1, "Param3", 5)
	d2_modified := WithParameter(&d2, "Param3", 7)
	testutils.FatalUnless(t, d1.value3 == 1, "WithParameters modified value3")
	testutils.FatalUnless(t, d2.value3 == 1, "WithParameters modified value3")
	testutils.FatalUnless(t, d1_modified.value3 == 5, "WithParameters did not modify value3")
	testutils.FatalUnless(t, d2_modified.value3 == 7, "WithParameters did not modify value3")

}

// ensureDefaultSettersAndGettersWorkForSerializer makes sure that our default_* functions work for the given type.
// Note that serializerType may be either a value or a pointer type
func ensureDefaultSettersAndGettersWorkForSerializer(serializerType reflect.Type, t *testing.T) {
	if serializerType.Kind() != reflect.Pointer {
		serializerType = reflect.PtrTo(serializerType)
	}
	nilSerializer := reflect.Zero(serializerType).Interface().(ParameterAware)

	if nilSerializer.HasParameter("InvalidParameter") {
		t.Fatalf("Serializer of type %v does recognize an invalid paramter as valid", utils.GetReflectName(serializerType))
	}

	ok, reason := utils.DoesMethodExist(serializerType, "Clone", []reflect.Type{}, []reflect.Type{anyType})
	testutils.FatalUnless(t, ok, reason)

	params := nilSerializer.RecognizedParameters()
	for _, param := range params {
		if !nilSerializer.HasParameter(param) {
			t.Fatalf("Serializer of type %v does not have parameter %v recognized by HasParameter", utils.GetReflectName(serializerType), param)
		}
		validate_DefaultGetterForParam(t, serializerType, param)
		validate_DefaultSetterForParam(t, serializerType, param)
	}
}

// these 2 might go to testing and use testutils.DoesMethodExist

func validate_DefaultSetterForParam(t *testing.T, serializerType reflect.Type, parameterName string) {

	var typeName string = utils.GetReflectName(serializerType) // used for better error messages

	// Retrieve method name from parameterName
	parameterNameNormalized := normalizeParameter(parameterName) // case-insensitive. The map keys are all normalized
	paramInfo, ok := default_serializerParamFuns[parameterNameNormalized]
	testutils.FatalUnless(t, ok, "validate_DefaultSetterForParam called with unrecognized parameter name %v. This is unrecognized no matter the serializer type", parameterName)

	// The specification of MethodByName differs if serializerType is an interface; calling it with an interface is a bug in the test.
	if serializerType.Kind() == reflect.Interface {
		panic(fmt.Errorf("validate_DefaultSetterForParam called with an interface type %v", typeName))
	}

	ok, reason := utils.DoesMethodExist(serializerType, paramInfo.setter, []reflect.Type{paramInfo.vartype}, []reflect.Type{})
	testutils.FatalUnless(t, ok, reason)
}

func validate_DefaultGetterForParam(t *testing.T, serializerType reflect.Type, parameterName string) {
	var typeName string = utils.GetReflectName(serializerType) // used for better error messages

	// Retrieve method name from parameterName
	parameterNameNormalized := normalizeParameter(parameterName) // case-insensitive. The map keys are all normalized
	paramInfo, ok := default_serializerParamFuns[parameterNameNormalized]
	testutils.FatalUnless(t, ok, "validate_DefaultGetterForParam called with unrecognized parameter name %v. This is unrecognized no matter the serializer type", parameterName)

	// The specification of MethodByName differs if serializerType is an interface; calling it with an interface is a bug in the test.
	if serializerType.Kind() == reflect.Interface {
		panic(fmt.Errorf("validate_DefaultGetterForParam called with an interface type %v", typeName))
	}

	ok, reason := utils.DoesMethodExist(serializerType, paramInfo.getter, []reflect.Type{}, []reflect.Type{paramInfo.vartype})
	testutils.FatalUnless(t, ok, reason)
}
