package pointserializer

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// The functions defined in this file serve to enable OOP-like programming styles.
// Notably, we have the issue that our serializers contain user-configurable parameter settings (such as endianness or header choices).
//
// The actual values of these parameters may have some validity constraints and also require some non-trivial code to correctly (deep!-)copy
// (thank you, Go, for not allowing to overload assignment).
// Now, we have actually several serializer types that share parameters, so the "Go way" of doing this forces us to basically Copy&Paste all that parameter assigment
// for every combination. This is doubly bad, because that boilerplate is actually non-trivial due to error handling code and a nightmare to maintain consistently if spread around the code base.
// The problem is not solvable by struct embedding the configurable parameters, because the user-facing API contains methods/functions with semantics
//  "Create a copy of this serializer with parameter bar set to foo". This cannot be defined on the parameter,
// because methods called on embedded structs by design (The Go language designers bascially do not trust their programmers to know when/how to use OOP, so they forbid it outright)
// do not know anything about the embedding struct.
//
// We "solve" this problem by just screwing with idiomatic Go and use reflection:
// We bascially define a makeCopyWithParams function defined for the embedding struct (and is hence aware of it) that just calls the appropriate
// functions on the embedded struct via reflection.

// serializerParams is a global constant map that is used to lookup the names of setter and getter methods and their expected/returned types (which are called via reflection)
var serializerParams = map[string]struct {
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

// parameterAware is the interface satisfied by all (parts of) serializers that work with makeCopyWithParameters
type parameterAware interface {
	RecognizedParameters() []string // returns a slice of strings of all parameters that are recognized by this particular serializer.
	HasParameter(paramName string) bool
}

// NOTE: Our convention is to use the same, possibly non-normalized name for parameters everywhere.

// normalizeParameter is called on all parameter name arguments to make them case-insensitive.
func normalizeParameter(arg string) string {
	return strings.ToLower(arg)
}

// concatParameterList concatenates two lists of parameters, removing duplicates modulo normalizeParameters.
// The intended use is to implement RecognizedParameters() by merging component lists of accepted parameters.
func concatParameterList(list1 []string, list2 []string) []string {
	return utils.ConcatenateListsWithoutDuplicates(list1, list2, normalizeParameter)
}

// TOOD: This does not check argument types for getters and setters!

// hasSetterAndGetterForParameter(serializer, parameterName) checks whether the (dynamic) type of serializer has setter and getter methods for the given parameter.
// The name of these getter and setter methods is looked up via the serializerParams map.
// parameterName is case-insensitive
//
// Note that the serializer argument is only used to derive the generic parameters and may be a nil pointer of the appropriate type.
//
// Note: This internal function does NOT look at RecognizedParameters().
// It instead uses reflection to check the presence of methods.
// It panics if called on invalid parameter strings not in the serializerParams map.
//
// DEPRECATED
func hasSetterAndGetterForParameter(serializer any, parameterName string) bool {
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive
	paramInfo, ok := serializerParams[parameterName]

	// Technically, we could just meaningfully return false if parameterName is not found in serializerParams.
	// However, this is an internal function and we never intend to call hasParameter on anything but a fixed string which is supposed to be a key of the serializerParams map.
	// Hence, this can only occur due to a bug (e.g. a typo in the parameterName string).
	if !ok {
		panic(ErrorPrefix + "hasParameter called with unrecognized parameter name")
	}

	serializerType := reflect.TypeOf(serializer)
	_, ok = serializerType.MethodByName(paramInfo.getter)
	if !ok {
		return false
	}
	_, ok = serializerType.MethodByName(paramInfo.setter)
	return ok
}

// these 2 might go to testing and use testutils.DoesMethodExist

func validateSetter(serializer parameterAware, parameterName string) {
	if serializer == nil {
		panic(ErrorPrefix + "validateSetter called with nil serializer")
	}
	serializerType := reflect.TypeOf(serializer)
	var typeName string = utils.GetReflectName(serializerType) // used for better error messages

	// Retrieve method name from parameterName
	parameterNameNormalized := normalizeParameter(parameterName) // case-insensitive. The map keys are all normalized
	paramInfo, ok := serializerParams[parameterNameNormalized]
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"validateSetter called with unrecognized parameter name %v. This is unrecognized no matter the serializer type", parameterName))
	}

	// reflect.TypeOf reaches into serializer. This is just to make absolutely sure, because the specification of MethodByName differs if serializerType is an interface
	if serializerType.Kind() == reflect.Interface {
		panic("Should be unreachable")
	}

	setterMethod, ok := serializerType.MethodByName(paramInfo.setter)
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"validateSetter could not retrieve setter methods named %v for type %v", parameterNameNormalized, typeName))
	}
	var setterType reflect.Type = setterMethod.Type // type of method. Note that setterType describes a function with the first argument being the receiver.
	if setterType.Kind() != reflect.Func {
		panic("Should be unreachable")
	}

	// We refuse to consider setters with >0 return values, since we would silently discarding the output.
	if numOutputs := setterType.NumOut(); numOutputs != 0 {
		panic(fmt.Errorf(ErrorPrefix+"validateSetter called with type %v whose parameter setter %v returns a non-zero number %v of return values", typeName, paramInfo.setter, numOutputs))
	}

	// Check the number of input arguments. Note that the first input is the receiver (since we called MethodByName on a reflect.Type rather than a reflect.Value).
	if numInputs := setterType.NumIn(); numInputs != 2 {
		panic(fmt.Errorf(ErrorPrefix+"validateSetter called with type %v whose parameter setter %v takes %v rather than 1 input argument ", typeName, paramInfo.setter, numInputs-1))
	}

	inputArgType := setterType.In(1) // declared argument type for the setter function.
	if !paramInfo.vartype.AssignableTo(inputArgType) {
		panic(fmt.Errorf(ErrorPrefix+"validateSetter detected setter %v for %v with invalid signature: We expected a type %v, but got %v instead",
			paramInfo.setter, typeName, utils.GetReflectName(paramInfo.vartype), utils.GetReflectName(inputArgType)))
	}
}

func validateGetter(serializer parameterAware, parameterName string) {
	if serializer == nil {
		panic(ErrorPrefix + "validateGetter called with nil serializer")
	}
	serializerType := reflect.TypeOf(serializer)
	var typeName string = utils.GetReflectName(serializerType) // used for better error messages

	// Retrieve method name from parameterName
	parameterNameNormalized := normalizeParameter(parameterName) // case-insensitive. The map keys are all normalized
	paramInfo, ok := serializerParams[parameterNameNormalized]
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"validateGetter called with unrecognized parameter name %v. This is unrecognized no matter the serializer type", parameterName))
	}

	exptectedArgType := paramInfo.vartype

	// reflect.TypeOf reaches into serializer. This is just to make absolutely sure, because the specification of MethodByName differs if serializerType is an interface
	if serializerType.Kind() == reflect.Interface {
		panic("Should be unreachable")
	}

	getterMethod, ok := serializerType.MethodByName(paramInfo.getter)
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"validateGetter could not retrieve getter methods named %v for type %v", parameterNameNormalized, typeName))
	}
	var getterType reflect.Type = getterMethod.Type // type of method. Note that getterType describes a function with the first argument being the receiver.
	if getterType.Kind() != reflect.Func {
		panic("Should be unreachable")
	}

	// We refuse to consider setters with !=1 return values
	if numOutputs := getterType.NumOut(); numOutputs != 1 {
		panic(fmt.Errorf(ErrorPrefix+"validateGetter called with type %v whose parameter getter %v returns a number %v != 1 of return values", typeName, paramInfo.getter, numOutputs))
	}

	// Check the number of input arguments. Note that the first input is the receiver (since we called MethodByName on a reflect.Type rather than a reflect.Value).
	if numInputs := getterType.NumIn(); numInputs != 1 {
		panic(fmt.Errorf(ErrorPrefix+"validateGetter called with type %v whose parameter getter %v takes %v rather than 0 input argument ", typeName, paramInfo.getter, numInputs-1))
	}

	returnedType := getterType.Out(0) // declared return type for the getter function.
	if !returnedType.AssignableTo(exptectedArgType) {
		panic(fmt.Errorf(ErrorPrefix+"validateGetter detected getter %v for %v with invalid signature: We expected a return type %v, but got %v instead",
			paramInfo.setter, typeName, utils.GetReflectName(exptectedArgType), utils.GetReflectName(returnedType)))
	}
}

// TODO: We may remove this after making Validate() a hard requirement for all components of our serializer parts.

// validater is an interface used for type-asserting/checking whether some type has a Validate() method
type validater interface {
	Validate() // panics on failure, so no return value
}

// makeCopyWithParameters(serializer, parameterName, newParam) takes a serializer (anything with a Clone-method, really) and returns an
// independent copy (create via Clone() with the parameter given by parameterName replaced by newParam.
//
// The serializer argument is a pointer, but the returned value is not.
// parameterName is looked up in the global serializerParams map to obtain getter/setter method names.
// There must be a Clone() - Method defined on SerializerPtr returning either a SerializerType or SerializerPtr
// The function panics on failure.
func makeCopyWithParameters[SerializerType any, SerializerPtr interface {
	*SerializerType
	// utils.Clonable[SerializerPtr] OR utils.Clonable[SerializerType]
	Validate()
	parameterAware
},
](serializer SerializerPtr, parameterName string, newParam any) SerializerType {

	// Obtain string representations of parameter type. This is only used for better error messages.
	var typeName string = utils.GetReflectName(utils.TypeOfType[SerializerType]())

	// Retrieve method name from parameterName
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive. The map keys are all normalized
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		panic(ErrorPrefix + "makeCopyWithParams called with unrecognized parameter name")
	}

	// check whether parameterName is accepted by this serializer
	if !serializer.HasParameter(parameterName) {
		panic(fmt.Errorf(ErrorPrefix+"getSerializerParam called on %v with parameter name %v (normalized) that is not among the list of recognized parameters for this type", typeName, parameterName))
	}

	// Make a copy of the serializer. Note that Clone() returns a copy.

	var clonePtr SerializerPtr
	switch clonable := any(serializer).(type) {
	case utils.Clonable[SerializerPtr]:
		clonePtr = clonable.Clone()
	case utils.Clonable[SerializerType]:
		nonPointerClone := clonable.Clone()
		clonePtr = &nonPointerClone
	default:
		panic(ErrorPrefix + " called makeCopyWithParameters on type that has no appropriate Clone() method")
	}

	clonePtrValue := reflect.ValueOf(clonePtr)

	// Get setter method (as a reflect.Value with .Kind() == reflect.Function) and make some basic checks.
	setterMethod := clonePtrValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with type %v lacking a setter method %v for the requested parameter %v", typeName, paramInfo.setter, parameterName))
	}

	// This subsumes the test below, anyway
	validateSetter(serializer, parameterName)

	// We refuse to call setters with >0 return values rather than silently discarding them.
	if numOutputs := setterMethod.Type().NumOut(); numOutputs != 0 {
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with type %v whose parameter setter %v returns a non-zero number %v of return values", typeName, paramInfo.setter, numOutputs))
	}

	// Not really needed, since this would cause a panic from setterMethod.Call later, but we prefer a more meaningful error message.
	if numInputs := setterMethod.Type().NumIn(); numInputs != 1 {
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with type %v whose parameter setter %v takes %v rather than 1 input argument ", typeName, paramInfo.setter, numInputs))
	}

	// Wrap newParam that was input as new value and make some basic check. Note that the AssignableTo-Check is not really needed and we could just let Call() panic.
	// However, we catch that particular error case for the sake of better error reporting.
	newParamValue := reflect.ValueOf(newParam)
	newParamType := newParamValue.Type()
	if !newParamType.AssignableTo(paramInfo.vartype) {
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with wrong type of argument %v. Expected argument type was %v", utils.GetReflectName(newParamType), utils.GetReflectName(paramInfo.vartype)))
	}

	// Call Setter on clone with new value. This may fail for various reasons (such as a Validate() call from the setter panicking)
	setterMethod.Call([]reflect.Value{newParamValue})

	// Any setter called above should, of course, validate whether the input is valid;
	// however in some cases, there are constraints that cannot be handled by the setter
	// (such as a setter from a struct-embedded type, where there are constraints imposed via the embedding type)
	clonePtr.Validate()

	return *clonePtr
}

// getSerializerParameter takes a serializer and returns the parameter stored under the key parameterName.
// The type of the return value depends on parameterName.
// parameterName is case-insensitive.
//
// Note that we should pass a pointer to this function, since we reflect-call a function with it as receiver.
func getSerializerParameter(serializer parameterAware, parameterName string) interface{} {

	// used for diagnostics.
	receiverType := reflect.TypeOf(serializer)
	receiverName := utils.GetReflectName(receiverType)

	// check whether parameterName is recognized by the serializer
	if !serializer.HasParameter(parameterName) {
		panic(fmt.Errorf(ErrorPrefix+"getSerializerParam called on %v with parameter name %v that is not among the list of recognized parameters for this type", receiverName, parameterName))
	}

	// Normalize parameterName
	parameterName = normalizeParameter(parameterName)

	// Obtain getter method string
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		// If we get here, something output by RecognizedParameters() is not in the global serializerParams map.
		// This is not supposed to happen even if we call getSerializerParameter with bad input.
		panic(fmt.Errorf(ErrorPrefix+"getSerializerParam called on %v with unrecognized parameter name %v (normalized). This is not supposed to be possible", receiverName, parameterName))
	}

	getterName := paramInfo.getter
	expectedReturnType := paramInfo.vartype // Note: If this an interface type, it's OK if the getter returns a realization.

	// Obtain getter Method
	serializerValue := reflect.ValueOf(serializer)
	getterMethod := serializerValue.MethodByName(getterName)

	// Check some validity contraints on the getter
	if !getterMethod.IsValid() {
		panic(fmt.Errorf("bandersnatch / serialization: getSerializerParam called on %v with parameter %v, but that type does not have a %v method", receiverName, parameterName, getterName))
	}
	getterType := getterMethod.Type()
	if numIn := getterType.NumIn(); numIn != 0 {
		panic(fmt.Errorf(ErrorPrefix+"Getter Method %v called via getSerializeParam on %v takes %v > 0 arguments", getterName, receiverName, numIn))
	}
	if numOut := getterType.NumOut(); numOut != 1 {
		panic(fmt.Errorf(ErrorPrefix+"Getter Method %v called via getSerializeParam on %v returns %v != 1 arguments", getterName, receiverName, numOut))
	}

	// Check type returned by getter method. Note: If the getter method returns an interface (such as any), we should actually check the dynamic type.
	// The latter is hard to get right (due to nil etc) and can only be done AFTER the actual call, so we just don't; this is only for better diagnostics anyway.
	if outType := getterType.Out(0); !outType.AssignableTo(expectedReturnType) && outType.Kind() != reflect.Interface {
		panic(fmt.Errorf(ErrorPrefix+"Getter Method %v called via getSerializeParam on %v returns value of type %v, which is not assignable to %v", getterName, receiverName, outType, expectedReturnType))
	}

	retValue := getterMethod.Call([]reflect.Value{})[0] // Note: If the getter method returns an interface, retValue.Type() is actually this static interface type.
	return retValue.Interface()
}
