package testutils

import (
	"fmt"
	"reflect"
)

// This file contains some helper functions to outsource some boilerplate checks when calling methods via reflection.

// DoesMethodExist checks whether receiverType has a method of name methodName with inputs and outputs of (approximately) the given type.
// Note that input and output types only need to match up to assignability. On failure, gives a reason string explaining the failure.
// On success, reason == ""
func DoesMethodExist(receiverType reflect.Type, methodName string, inputs []reflect.Type, outputs []reflect.Type) (good bool, reason string) {
	var elemType reflect.Type
	// var ptrReceiver bool
	if receiverType.Kind() == reflect.Ptr {
		elemType = receiverType.Elem()
		// ptrReceiver = true
	} else {
		elemType = receiverType
		// ptrReceiver = false
	}

	var typeName string = GetReflectName(receiverType)

	if elemType.Kind() != reflect.Struct {
		return false, fmt.Sprintf("receiver type %v is no struct or pointer-to-struct", typeName)
	}

	var method reflect.Method
	method, ok := receiverType.MethodByName(methodName)
	if !ok {
		return false, fmt.Sprintf("type %v has no method named %v", typeName, methodName)
	}
	methodType := method.Type
	// +1 comes from the receiver. Since we called MethodByName on a reflect.Type (rather than a reflect.Value), the method is unbound and has the receiver as explicit first input.
	if methodType.NumIn() != len(inputs)+1 {
		return false, fmt.Sprintf("method %v.%v has an unexpected number of input arguments. We expected %v, got %v", typeName, methodName, len(inputs), methodType.NumIn()-1)
	}
	if methodType.NumOut() != len(outputs) {
		return false, fmt.Sprintf("method %v.%v has an unexpected number of output arguments. We expected %v, got %v", typeName, methodName, len(outputs), methodType.NumOut())
	}

	for i, t := range inputs {
		if !t.AssignableTo(methodType.In(i + 1)) {
			return false, fmt.Sprintf("method %v.%v's %v'th parameter type %v is not assignable to the expected parameter %v", typeName, methodName, i+1, GetReflectName(t), GetReflectName(methodType.In(i+1)))
		}
	}

	for i, t := range outputs {
		if !methodType.Out(i).AssignableTo(t) {
			return false, fmt.Sprintf("method %v.%v's %v'th output of type %v is not assignable to the expected type %v", typeName, methodName, i+1, GetReflectName(methodType.Out(i)), GetReflectName(t))
		}
	}
	return true, ""
}

// CallMethodByName(receiver, methodName, args...) calls receiver.method(args...) using reflection.
// It takes care about some part of the boilerplate and hides reflect.Value / reflect.Type at the expense of some
// issues with settability.
func CallMethodByName(receiver interface{}, methodName string, inputs ...interface{}) (outputs []interface{}) {
	var receiverValue reflect.Value = reflect.ValueOf(receiver)
	var receiverType reflect.Type = receiverValue.Type()
	var receiverTypeName string = GetReflectName(receiverType)
	methodValue := receiverValue.MethodByName(methodName)

	// Not sure about this, we better bail out
	if methodValue.Type().IsVariadic() {
		panic("Don't use CallMethodByName for variadic functions")
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
