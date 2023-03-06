package testutils

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// CallMethodByName(receiver, methodName, args...) calls receiver.method(args...) using reflection.
// It takes care about some part of the boilerplate and hides reflect.Value / reflect.Type at the expense of some
// issues with settability.
//
// Currently, it won't work with variadic methods and panics for those.
func CallMethodByName(receiver interface{}, methodName string, inputs ...interface{}) (outputs []interface{}) {
	var receiverValue reflect.Value = reflect.ValueOf(receiver)
	var receiverType reflect.Type = receiverValue.Type()
	var receiverTypeName string = utils.GetReflectName(receiverType)
	methodValue := receiverValue.MethodByName(methodName)

	// Not sure about this, we better bail out
	if methodValue.Type().IsVariadic() {
		panic("Don't use CallMethodByName for variadic methods")
	}

	if !methodValue.IsValid() {
		panic(fmt.Errorf("%v has no method called %v", receiverTypeName, methodName))
	}
	var inputValues []reflect.Value = make([]reflect.Value, len(inputs))
	outputs = make([]interface{}, methodValue.Type().NumOut())
	for i, input := range inputs {
		inputValues[i] = reflect.ValueOf(input)
	}

	outputValues := methodValue.Call(inputValues)
	for i, output := range outputValues {
		outputs[i] = output.Interface()
	}
	return
}
