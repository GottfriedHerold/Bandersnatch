package utils

import (
	"fmt"
	"reflect"
)

// This file contains some helper functions to outsource some boilerplate checks when calling methods via reflection.

// DoesMethodExist checks whether receiverType has a method of name methodName with inputs and outputs of (approximately) the given type.
// Note that input and output types only need to match up to assignability. On failure, gives a reason string explaining the failure.
// On success, reason == ""
//
// Note: If receiverType is a pointer type, this also finds methods with value receivers.
func DoesMethodExist(receiverType reflect.Type, methodName string, inputs []reflect.Type, outputs []reflect.Type) (good bool, reason string) {
	var typeName string = GetReflectName(receiverType)

	// Commented out to allow methods on non-structs. Not sure if we need this case, though.

	/*
		var elemType reflect.Type
		// var ptrReceiver bool
		if receiverType.Kind() == reflect.Ptr {
			elemType = receiverType.Elem()
			// ptrReceiver = true
		} else {
			elemType = receiverType
			// ptrReceiver = false
		}

			if elemType.Kind() != reflect.Struct {
				return false, fmt.Sprintf("receiver type %v is no struct or pointer-to-struct", typeName)
			}
	*/

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
