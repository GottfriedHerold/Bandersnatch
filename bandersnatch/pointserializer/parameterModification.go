package pointserializer

import (
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// TODO: Might not be exported?

type ParameterAware interface {
	RecognizedParameters() []string
	HasParameter(paramName string) bool
	GetParameter(paramName string) any
	// WithParameter(paramName string, newParam any) AssignableToReceiver -- Note: The return type may be either an interface or the concrete receiver type. The lack of covariance is why this is not part of the actual interface, but enforced at runtime using reflection.
	// We ask that the *dynamic* type returned is the same as the receiver.
}

var anyType = utils.TypeOfType[any]()
var stringType = utils.TypeOfType[string]()

// WithParameter is used to create a modified copy of t, with the parameter given by parameterName changed to newParam.
// The given t and the returned value of type T are supposed to be pointers.
//
// Typically, t is a (de-)serializer, but we also use it internally on sub-units of serializers.
//
// NOTE: This free function works by assuming that t satsfies the ParameterAware interface.
// We require (not expressible via Go interfaces without generics, and even generics would not work here) as part of that interface that
// T (or potentially a value receiver) has a method WithParameter(string, any) DynamicallyAssignableTo<T>
// which returns something whose *dynamic* type is assignable to T.
func WithParameter[T ParameterAware](t T, paramName string, newParam any) T {
	var TType = utils.TypeOfType[T]()

	// We check this before calling, in order to get better error messages.
	// NOTE: We do not impose any restriction on the return type here. The reason is that this function is concerned with the dynamic type that is returned.
	if ok, errMsg := utils.DoesMethodExist(TType, "WithParameter", []reflect.Type{stringType, anyType}, []reflect.Type{anyType}); !ok {
		panic(ErrorPrefix + "WithParameter free function failed, because receiver lacks an appropriate WithParameter method:\n" + errMsg)
	}

	if !t.HasParameter(paramName) {
		panic(fmt.Errorf(ErrorPrefix+"WithParameter free function called with parameterName %v that is invalid for the given argument %v", paramName, t))
	}

	receiverValue := reflect.ValueOf(t)
	methodValue := receiverValue.MethodByName("WithParameter")
	// The DoesMethodExist check before should have taken care of that (and given a more elaborate error message).
	if !methodValue.IsValid() {
		panic(ErrorPrefix + "WithParameter method not found. this should be impossible")
	}

	// "Just" call methodValue with paramName and newParam.
	// Unfortunately, it's not as simple, because reflect.ValueOf() handles nil in a way that requires special treatment.
	// We use common.CallFunction_FixNil
	paramNameValue := reflect.ValueOf(paramName)
	newParamValue := reflect.ValueOf(newParam)
	out := common.CallFunction_FixNil(methodValue, []reflect.Value{paramNameValue, newParamValue})[0]

	// should we panic with a more verbose error message if that type-assertion fails?
	return out.Interface().(T)
}

func default_HasParameter[T interface{ RecognizedParameters() []string }](t T, paramName string) bool {
	return utils.ElementInList(paramName, t.RecognizedParameters(), normalizeParameter)
}

// default_serializerParamFuns is a global constant map that is used to lookup the names of setter and getter methods and their expected/returned types (which are called via reflection)
var default_serializerParamFuns = map[string]struct {
	getter  string
	setter  string
	vartype reflect.Type
}{
	// Note: We use utils.TypeOfType rather than reflect.TypeOf, since this also works with interface types such as binary.ByteOrder.
	normalizeParameter("Endianness"):        {getter: "GetEndianness", setter: "SetEndianness", vartype: utils.TypeOfType[binary.ByteOrder]()},
	normalizeParameter("BitHeader"):         {getter: "GetBitHeader", setter: "SetBitHeaderFromBitHeader", vartype: utils.TypeOfType[common.BitHeader]()},
	normalizeParameter("BitHeader2"):        {getter: "GetBitHeader2", setter: "SetBitHeader2", vartype: utils.TypeOfType[common.BitHeader]()},
	normalizeParameter("SubgroupOnly"):      {getter: "IsSubgroupOnly", setter: "SetSubgroupRestriction", vartype: utils.TypeOfType[bool]()},
	normalizeParameter("GlobalSliceHeader"): {getter: "GetGlobalSliceHeader", setter: "SetGlobalSliceHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("GlobalSliceFooter"): {getter: "GetGlobalSliceFooter", setter: "SetGlobalSliceFooter", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("PerPointHeader"):    {getter: "GetPerPointHeader", setter: "SetPerPointHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("PerPointFooter"):    {getter: "GetPerPointFooter", setter: "SetPerPointFooter", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("SinglePointHeader"): {getter: "GetSinglePointHeader", setter: "SetSinglePointHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("SinglePointFooter"): {getter: "GetSinglePointFooter", setter: "SetSinglePointFooter", vartype: utils.TypeOfType[[]byte]()},
}

// default_GetParameter is a default implementation for GetParameter. The latter It takes a serializer and returns the parameter stored under the key parameterName.
// The type of the return value depends on parameterName.
// parameterName is case-insensitive.
//
// NOTE: default_GetParameter relies on the fact that all parameterNames are in the global default_serializerParamFuns map and the type has appropriate Getters/Setters.
// The constraint on T is unable to express that.
// Add a test using ensureDefaultSettersAndGettersWorkForSerializer to validate this for each type you use this on.
//
// Note that we should pass a pointer to this function, since we reflect-call a function with it as receiver.
func default_GetParameter[T interface{ HasParameter(string) bool }](serializer T, parameterName string) interface{} {

	// used for diagnostics.
	receiverType := reflect.TypeOf(serializer)
	receiverName := utils.GetReflectName(receiverType)

	// check whether parameterName is recognized by the serializer
	if !serializer.HasParameter(parameterName) {
		panic(fmt.Errorf(ErrorPrefix+"GetParameter called on type %v with parameter name %v that is not among the list of recognized parameters for this type", receiverName, parameterName))
	}

	// Normalize parameterName
	parameterName = normalizeParameter(parameterName)

	// Obtain getter method string
	paramInfo, ok := default_serializerParamFuns[parameterName]
	if !ok {
		// If we get here, a parameter accepted by parameterName is not in the default_serializerParamFuns map.
		// In this case, we must not use default_getParameter.
		panic(fmt.Errorf(ErrorPrefix+"default_getParameter called on %v with unrecognized parameter name %v (normalized). This is not supposed to be possible", receiverName, parameterName))
	}

	getterName := paramInfo.getter
	expectedReturnType := paramInfo.vartype // Note: If this an interface type, it's OK if the getter returns a realization.

	// Obtain getter Method
	serializerValue := reflect.ValueOf(serializer)
	getterMethod := serializerValue.MethodByName(getterName)

	// Check some validity contraints on the getter
	// If any of these happen, we (as opposed to the user of the library) screwed up.
	if !getterMethod.IsValid() {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: default_getParameter called on %v with parameter %v, but that type does not have a %v method", receiverName, parameterName, getterName))
	}
	getterType := getterMethod.Type()
	if numIn := getterType.NumIn(); numIn != 0 {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: Getter Method %v called via default_getParameter on %v takes %v > 0 arguments", getterName, receiverName, numIn))
	}
	if numOut := getterType.NumOut(); numOut != 1 {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: Getter Method %v called via default_getParameter on %v returns %v != 1 arguments", getterName, receiverName, numOut))
	}

	// Check type returned by getter method. Note: If the getter method returns an interface (such as any), we should actually check the dynamic type.
	// The latter is hard to get right (due to nil etc) and can only be done AFTER the actual call, so we just don't; this is only for better diagnostics anyway.
	if outType := getterType.Out(0); !outType.AssignableTo(expectedReturnType) && outType.Kind() != reflect.Interface {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: Getter Method %v called via default_getParameter on %v returns value of type %v, which is not assignable to %v", getterName, receiverName, outType, expectedReturnType))
	}

	// Note: If the getter method returns an interface, retValue.Type() is actually this static interface type (rather than the dynamic type).
	// In particular, if the getter returns any(nil), this does actually work without needing special treatment.
	retValue := getterMethod.Call([]reflect.Value{})[0]
	return retValue.Interface()
}

// default_WithParamter provides a default implementation of WithParameter, (soft-)required for the ParameterAware interface.
//
// NOTE: default_WithParameter relies on the fact that all parameterNames are in the global default_serializerParamFuns map and the type has appropriate Getters/Setters.
// Furthermore, the type needs an appropriate Clone method.
// The constraint on T is unable to express that.
//
// Add a test using ensureDefaultSettersAndGettersWorkForSerializer to validate this for each type you use this on.
func default_WithParameter[T any, Ptr interface {
	*T
	HasParameter(string) bool
	Validate()
}](serializer Ptr, parameterName string, newParam any) Ptr {

	// Obtain string representations of parameter type. This is only used for better error messages.
	var typeName string = utils.GetReflectName(utils.TypeOfType[T]())

	// check whether parameterName is accepted by this serializer
	if !serializer.HasParameter(parameterName) {
		panic(fmt.Errorf(ErrorPrefix+"WithParameter called on %v with parameter name %v (normalized) that is not among the list of recognized parameters for this type", typeName, parameterName))
	}

	// Retrieve method name from parameterName
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive. The map keys are all normalized
	paramInfo, ok := default_serializerParamFuns[parameterName]
	// If this happens, we (and not the library user) screwed up.
	if !ok {
		panic(ErrorPrefix + "Internal error: default_withParameter called with unrecognized parameter name")
	}

	// If this panics, we screwed up.
	clonePtr := common.Clone(serializer)

	clonePtrValue := reflect.ValueOf(clonePtr)

	// Get setter method (as a reflect.Value with .Kind() == reflect.Function) and make some basic checks.
	setterMethod := clonePtrValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: default_withParameter called with type %v lacking a setter method %v for the requested parameter %v", typeName, paramInfo.setter, parameterName))
	}

	ok, errMsg := utils.DoesMethodExist(clonePtrValue.Type(), paramInfo.setter, []reflect.Type{paramInfo.vartype}, []reflect.Type{})
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"Internal error: WithParameter called on type %v for parameter %v, whose implementation uses default_withParameter. However, the argument types of the internal setters do not match what default_withParameter expects:\n%v", typeName, parameterName, errMsg))
	}

	newParamValue := reflect.ValueOf(newParam)

	common.CallFunction_FixNil(setterMethod, []reflect.Value{newParamValue})
	clonePtr.Validate()

	return clonePtr
}
