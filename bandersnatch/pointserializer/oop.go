package pointserializer

import (
	"fmt"
	"reflect"
	"strings"

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

// parameterAware is the interface satisfied by all (parts of) serializers that work with makeCopyWithParameters
//
// DEPRECATED
type parameterAware interface {
	RecognizedParameters() []string // returns a slice of strings of all parameters that are recognized by this particular serializer.
	HasParameter(paramName string) bool
}

// NOTE: Our convention is to use the same, possibly non-normalized name for parameters everywhere.

// normalizeParameter is called on all parameter name arguments to make them case-insensitive.
func normalizeParameter(arg string) string {
	return strings.ToLower(arg)
}

// concatenateParameterList concatenates two lists of parameters, removing duplicates modulo normalizeParameters.
// The intended use is to implement RecognizedParameters() by merging component lists of accepted parameters.
//
// DEPRECATED
func concatenateParameterList(list1 []string, list2 []string) []string {
	return utils.ConcatenateListsWithoutDuplicates(list1, list2, normalizeParameter)
}

// makeCopyWithParameters(serializer, parameterName, newParam) takes a serializer (anything with a Clone-method, really) and returns an
// independent copy (create via Clone() with the parameter given by parameterName replaced by newParam.
//
// The serializer argument is a pointer, but the returned value is not.
// parameterName is looked up in the global serializerParams map to obtain getter/setter method names.
// There must be a Clone() - Method defined on SerializerPtr returning either a SerializerType or SerializerPtr
// The function panics on failure.
//
// DEPRECATED
func makeCopyWithParameters[SerializerType any, SerializerPtr interface {
	*SerializerType
	// utils.Clonable[SerializerPtr] OR utils.Clonable[SerializerType]
	// Validate()
	parameterAware
},
](serializer SerializerPtr, parameterName string, newParam any) SerializerType {

	// Obtain string representations of parameter type. This is only used for better error messages.
	var typeName string = utils.GetReflectName(utils.TypeOfType[SerializerType]())

	// Retrieve method name from parameterName
	parameterName = normalizeParameter(parameterName) // make parameterName case-insensitive. The map keys are all normalized
	paramInfo, ok := default_serializerParamFuns[parameterName]
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

	// MISSING

	// validate_DefaultSetterForParam(serializer, parameterName)

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
	// clonePtr.Validate()

	return *clonePtr
}
