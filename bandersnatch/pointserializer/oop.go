package pointserializer

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
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
	normalizeParameter("BitHeader"):         {getter: "GetBitHeader", setter: "SetBitHeader", vartype: utils.TypeOfType[common.BitHeader]()},
	normalizeParameter("BitHeader2"):        {getter: "GetBitHeader2", setter: "SetBitHeader2", vartype: utils.TypeOfType[common.BitHeader]()},
	normalizeParameter("SubgroupOnly"):      {getter: "IsSubgroupOnly", setter: "SetSubgroupRestriction", vartype: utils.TypeOfType[bool]()},
	normalizeParameter("GlobalSliceHeader"): {getter: "GetGlobalSliceHeader", setter: "SetGlobalSliceHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("GlobalSliceFooter"): {getter: "GetGlobalSliceFooter", setter: "SetGlobalSliceFooter", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("PerPointHeader"):    {getter: "GetPerPointHeader", setter: "SetPerPointHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("PerPointFooter"):    {getter: "GetPerPointFooter", setter: "SetPerPointFooter", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("SinglePointHeader"): {getter: "GetSinglePointHeader", setter: "SetSinglePointHeader", vartype: utils.TypeOfType[[]byte]()},
	normalizeParameter("SinglePointFooter"): {getter: "GetSinglePointFooter", setter: "SetSinglePointFooter", vartype: utils.TypeOfType[[]byte]()},
}

type ParameterAware interface {
	RecognizedParameters() []string // returns a slice of strings of all parameters that are recognized by this particular serializer.
}

// normalizeParameter is called on all parameter name arguments to make them case-insensitive.
func normalizeParameter(arg string) string {
	return strings.ToLower(arg)
}

func concatParameterList(list1 []string, list2 []string) []string {
	return utils.ConcatenateListsWithoutDuplicates(list1, list2, normalizeParameter)
}

// hasParameter(serializer, parameterName) checks whether the (dynamic) type of serializer has setter and getter methods for the given parameter.
// The name of these getter and setter methods is looked up via the serializerParams map.
// parameterName is case-insensitive
//
// Note that the serializer argument is only used to derive the generic parameters and may be a nil pointer of the appropriate type.
//
// Note: This internal function does NOT look at RecognizedParameters()
func hasParameter(serializer any, parameterName string) bool {
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

// TODO: We might not need this if we just make Validate() a hard requirement for all components of our serializer parts.

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
	ParameterAware
},
](serializer SerializerPtr, parameterName string, newParam any) SerializerType {

	// Obtain string representations of parameter type. This is only used for better error messages.
	var typeName string = testutils.GetReflectName(utils.TypeOfType[SerializerType]())

	// Retrieve method name from parameterName
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive. The map keys are all normalized
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		panic(ErrorPrefix + "makeCopyWithParams called with unrecognized parameter name")
	}

	// check whether parameterName is in the serializer.RecognizedParameters() list (modulo normalizeParameters)
	parameterName = normalizeParameter(parameterName)
	paramList := serializer.RecognizedParameters()
	if !utils.ElementInList(parameterName, paramList, normalizeParameter) {
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

	// Get setter method (as a reflect.Value of function Kind) and make some basic checks.
	// Note that users can actually trigger these panics when trying to modify a parameter for a serializer that does not have it.
	setterMethod := clonePtrValue.MethodByName(paramInfo.setter)
	if !setterMethod.IsValid() {
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with type %v lacking a setter method %v for the requested parameter %v", typeName, paramInfo.setter, parameterName))
	}
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
		panic(fmt.Errorf(ErrorPrefix+"makeCopyWithParams called with wrong type of argument %v. Expected argument type was %v", testutils.GetReflectName(newParamType), testutils.GetReflectName(paramInfo.vartype)))
	}

	// Call Setter on clone with new value. This may fail if, e.g. parameter
	setterMethod.Call([]reflect.Value{newParamValue})

	// Any setter called above should, of course, validate whether the input is valid;
	// however in some cases, there are constraints that cannot be handled by the setter
	// (such as a setter from a struct-embedded type, where there are constraints imposed via the embedding type)
	clonePtr.Validate()

	return *clonePtr
}

// getSerializerParam takes a serializer and returns the parameter stored under the key parameterName.
// The type of the return value depends on parameterName.
// parameterName is case-insensitive.
//
// Note that we should pass a pointer to this function, since we reflect-call a function with it as receiver.
// We also accept pointer-to-interface, in which case, we dereference once.
func getSerializerParam(serializer ParameterAware, parameterName string) interface{} {

	// receiverName := utils.NameOfType[ValueType]() // used for diagnostics.
	receiverType := reflect.TypeOf(serializer)
	receiverName := testutils.GetReflectName(receiverType)

	// check whether parameterName is in the serializer.RecognizedParameters() list (modulo normalizeParameters)
	if !utils.ElementInList(parameterName, serializer.RecognizedParameters(), normalizeParameter) {
		panic(fmt.Errorf(ErrorPrefix+"getSerializerParam called on %v with parameter name %v that is not among the list of recognized parameters for this type", receiverName, parameterName))
	}

	// Normalize parameterName
	parameterName = normalizeParameter(parameterName)

	// Obtain getter method string
	paramInfo, ok := serializerParams[parameterName]
	if !ok {
		panic(fmt.Errorf(ErrorPrefix+"getSerializerParam called on %v with unrecognized parameter name %v (normalized)", receiverName, parameterName))
	}

	getterName := paramInfo.getter
	expectedReturnType := paramInfo.vartype // Note: If this an interface type, it's OK if the getter returns a realization.

	// Obtain getter and check parameters
	/*
		var serializerValue reflect.Value
		if utils.TypeOfType[ValueType]().Kind() == reflect.Interface {
			serializerValue = reflect.ValueOf(*serializer)
		} else {
			serializerValue = reflect.ValueOf(serializer)
		}
	*/
	serializerValue := reflect.ValueOf(serializer)

	getterMethod := serializerValue.MethodByName(getterName)
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
	// The latter is hard to get right (due to nil etc), so we just don't; this is only for better diagnostics anyway.
	if outType := getterType.Out(0); !outType.AssignableTo(expectedReturnType) && outType.Kind() != reflect.Interface {
		panic(fmt.Errorf(ErrorPrefix+"Getter Method %v called via getSerializeParam on %v returns value of type %v, which is not assignable to %v", getterName, receiverName, outType, expectedReturnType))
	}

	retValue := getterMethod.Call([]reflect.Value{})[0] // Note: If the getter method returns an interface, retValue.Type() is actually this static interface type.
	return retValue.Interface()
}
