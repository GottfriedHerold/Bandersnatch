package errorsWithData

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type EqualityComparisonFunction func(any, any) (result bool)

func comparison_very_naive_old(x, y any) (equal bool, reason error) {
	return x == y, nil
}

// comparison_handleNils compares x and y for equality with the following quirks:
//
//   - if either x or y are the nil interface, then the comparison result is true iff the other argument is either a nil interface or a nil of concrete type.
//     (this behaviour is appropriate for usage with [errorsWithData] )
//   - if both x and y have the same incomparable (dynamic) type, the function panics
//   - otherwise, we check whether x==y holds
func comparison_handleNils(x, y any) (isEqual bool) {
	if x == nil {
		if y == nil { // Note that yReflected:=reflect.ValueOf(nil) panics on yReflected.IsNil(), so we have to special-case this
			return true
		}
		yReflected := reflect.ValueOf(y)
		if !utils.IsNilable(yReflected.Type()) {
			return false
		}
		return yReflected.IsNil()
	}
	if y == nil {
		// x != nil is guaranteed
		xReflected := reflect.ValueOf(x)
		if !utils.IsNilable(xReflected.Type()) {
			return false
		}
		return xReflected.IsNil()
	}
	// x != nil, y != nil is guaranteed
	return x == y // NOTE: may panic
}

// comparison_IsEqual compares x and y for equality with the following quirks:
//
// - if either x or y are the nil interface, then the comparison result is true iff the other argument is either a nil interface or a nil of concrete type

func comparison_IsEqual(x, y any) (isEqual bool) {
	xValue := reflect.ValueOf(x)
	xType := xValue.Type()
	yValue := reflect.ValueOf(y)
	yType := yValue.Type()

	if x == nil {
		if y == nil { // Note that yValue:=reflect.ValueOf(nil) panics on yReflected.IsNil(), so we have to special-case this
			return true
		}
		if !utils.IsNilable(yType) {
			return false
		}
		return yValue.IsNil()
	}
	if y == nil {
		// x != nil is guaranteed

		if !utils.IsNilable(xType) {
			return false
		}
		return xValue.IsNil()
	}

	if xType.Kind() == reflect.Pointer {
		return x == y
	}

	if yType.Kind() == reflect.Pointer {
		return false // x == y is always false, because x's kind is not pointer
	}

	if EqualMethod, found := xType.MethodByName("IsEqual"); found {
		// method found with value receiver x
		t := EqualMethod.Type // t.Kind is reflect.Function
		if t.NumIn() != 2 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method with wrong number of arguments", x))
		}
		secondArgType := t.Out(1)
		var worksWithValueArg bool = yType.AssignableTo(secondArgType)
		var worksWithPtrArg bool = reflect.PtrTo(yType).AssignableTo(secondArgType)
		if !worksWithPtrArg && !worksWithValueArg {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method, but it cannot take an argument of type %T", x, y))
		}
		var argVal reflect.Value
		if worksWithValueArg {
			argVal = reflect.ValueOf(y)
		} else {
			// Need to call EqualMethod on &y. This is not possible without copying y, since Go cannot take addresses of values inside an interface.
			argVal = reflect.New(yType)
			argValDeref := argVal.Elem()
			argValDeref.Set(reflect.ValueOf(y))
		}
		var callResults []reflect.Value = EqualMethod.Func.Call([]reflect.Value{xValue, argVal})
		if len(callResults) == 0 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method, but it returned 0 results", x))
		}
		var result any = callResults[0].Interface()
		resultBool, ok := result.(bool)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparsion, type %T had an IsEqual method, but its first return value is not bool", x))
		}
		return resultBool
	}

	return x == y

}
