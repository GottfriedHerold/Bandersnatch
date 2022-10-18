package pointserializer

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type ParameterAware interface {
	RecognizedParameters() []string
	HasParameter(paramName string) bool
	GetParameter(paramName string) any
	// WithParameter(paramName string, newParam any) AssignableToReceiver -- Note: The return type may be either an interface or the concrete receiver type. The lack of covariance is why this is not part of the actual interface, but enforced at runtime using reflection.
	// We ask that the *dynamic* type returned is the same as the receiver.
}

// TODO: Might not be exported?

var anyType = utils.TypeOfType[any]()
var stringType = utils.TypeOfType[string]()

// WithParameter is used to create a modified copy of t, with the parameter given by parameterName changed to newParam.
// The given t and the returned value of type T are supposed to be pointers.
//
// Typically, t is a (de-)serializer, but we also use it internally on sub-units of serializers.
//
// NOTE: This free function works by assuming that t satsfies the ParameterAware interface.
// We require (not expressible via Go interfaces without generics, which would not work here) as part of the interface that
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
