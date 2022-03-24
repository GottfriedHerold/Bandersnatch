package bandersnatch

import (
	"fmt"
	"math/big"
	"reflect"
)

// initFieldElementFromString initializes a field element from a given string. The given string can be a decimal or hex representation, but needs to be prefixed if hex.
// Since we only use it internally to initialize package-level variables (intendend to be constant) from (compile-time!) constant strings, panic() on error is appropriate.
func initFieldElementFromString(input string) (output bsFieldElement_64) {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	if !success {
		panic("String used to initialize field element not recognized as a valid number")
	}
	output.SetBigInt(t)
	return
}

// initIntFromString initializes a big.Int from a given string similar to initFieldElementFromString. The given string can be decimal or hex, but needs to be prefixed if hex.
// This essentially is equivalent to big.Int's SetString method, except that it panics on error (which is appropriate for initialization from compile-time constant strings).
func initIntFromString(input string) *big.Int {
	var t *big.Int = big.NewInt(0)
	var success bool
	t, success = t.SetString(input, 0)
	// Note: panic is the appropriate error handling here. Also, since this code is only run during package import, there is actually no way to catch it.
	if !success {
		panic("String used to initialized big.Int not recognized as a valid number")
	}
	return t
}

// move to internal/utils package:

// assert(condition) panics if condition is false; assert(condition, error) panics if condition is false with panic(error).
func assert(condition bool, err ...interface{}) {
	if len(err) > 1 {
		panic("assert can only handle 1 extra error argument")
	}
	if !condition {
		if len(err) == 0 {
			panic("This is not supposed to be possible")
		} else {
			panic(err[0])
		}
	}
}

// GetReflectName obtain a string representation of the given type using the reflection package. It cover more cases that plain c.Name() does.
func GetReflectName(c reflect.Type) (ret string) {
	// reflect.Type's  Name() only works for defined types, which
	// e.g. *Point_xtw_full is not. (Only Point_xtw_full is a defined type)
	ret = c.Name()
	if ret != "" {
		return
	}

	switch c.Kind() {
	case reflect.Ptr:
		return "*" + GetReflectName(c.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%v]%v", c.Len(), GetReflectName(c.Elem()))
	case reflect.Slice:
		return fmt.Sprintf("[]%v", GetReflectName(c.Elem()))
	default:
		return "<<type with unknown name>>"
	}
}

// doesMethodExist checks whether receiverType has a method of name methodName with inputs and outputs of (approximately) the given type.
// Note that input and output types only need to match up to assignability. On failure, gives a reason string explaining the failure.
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
