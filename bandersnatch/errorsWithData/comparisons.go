package errorsWithData

// NOTE: functions here might go to utils

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// EqualityComparisonFunction is a type alias to func(any,any) bool.
// Functions of this type are used by this package to compare arbitrary values for equality (with true meaning "the values are equal").
//
// Such functions are used when [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] flags are used.
//
// We provide [Comparison_IsEqual] as an actual non-trivial comparison method suitable for most of our purposes.
//
// Note that this package (by default, unless overriden by flags) treats a panic'ing EqualityComparisonFunction as an "inequal" result,
// but additionally takes the value provided in the panic as a reason for inequality.
type EqualityComparisonFunction = func(any, any) (result bool)

// withPanicResults takes an EqualityComparisonFunction f as input and returns a new function that calls f, but catches panics in f.
// If f panics, we return result == false, didPanic == true and panicValue is the panic value.
//
// Otherwise, the result is whatever f outputs, didPanic == false, panicValue == nil
func withPanicResults(f EqualityComparisonFunction) func(any, any) (result bool, didPanic bool, panicValue any) {
	return func(x, y any) (result bool, didPanic bool, panicValue any) {
		defer func() {
			if panicValue = recover(); panicValue != nil {
				didPanic = true
				result = false
			}
		}()
		result = f(x, y)
		return
	}
}

// comparison_handleNils compares x and y for equality with the following quirks:
//
//   - if either x or y are the nil interface, then the comparison result is true iff the other argument is either a nil interface or a nil of concrete type.
//     (this behaviour is appropriate for usage with the [errorsWithData] package)
//   - if both x and y have the same incomparable (dynamic) type, the function panics (the normal behaviour of x==y)
//   - otherwise, we check whether x==y holds
func comparison_handleNils(x, y any) (isEqual bool) {
	if x == nil {
		if y == nil { // Note that yReflected:=reflect.ValueOf(nil) panics on yReflected.IsNil(), so we have to special-case y==nil
			return true
		}
		yReflected := reflect.ValueOf(y)
		if !utils.IsNilable(yReflected.Type()) {
			return false
		}
		return yReflected.IsNil()
	}
	if y == nil {
		// x==nil was handled above, so x != nil is guaranteed
		xReflected := reflect.ValueOf(x)
		if !utils.IsNilable(xReflected.Type()) {
			return false
		}
		return xReflected.IsNil()
	}
	// x != nil, y != nil is guaranteed
	return x == y // NOTE: may panic
}

// CustomComparisonMethod returns a comparison function f := func(x,y any) bool{...}
// that is used to compare two values x and y for equality with the following quirks:
//
//   - if either x or y are the nil interface, then the comparison result f(x,y) is true iff the other argument is either a nil interface or a nil of concrete type
//   - otherwise, if either x or y are pointer types, then the comparison will directly compare the pointers.
//   - otherwise, we try each methodname in the list of methodnames in order:
//     if x has a method (on either pointer or value receiver) named methodname, we will call x.methodname(y) resp. x.methodname(&y)
//     -- Whether the method is called with &y or y is deduced from the method's signature. If both options are valid, we match the way x is passed.
//     -- The method must return a bool (possibly inside an interface) as its first return value; further return values are discarded.
//     -- If the method has a wrong signature, we panic.
//     -- Note that when calling with a pointer receiver or argument, we actuall pass a pointer to a copy of x or y.
//   - otherwise, if there is no method from the list of methodnames defined on x's type, we resort to plain == (which may panic for incomparable types)
//
// NOTE: Plain comparison takes precendence over methods with pointer receiver from the list if either x or y are pointers.
// Using pointer arguments / receivers is often just done to avoid copying,
// not necessarily because the pointers are the objects where we want to have custom equality semantics.
// Unfortunately, the Go language has no way to either express or differentiate these concepts.
// For the intended use case in the [errorsWithData] package, our choice is appropriate, as data accompanying errors should be the actual data rather than
// a (possibly shared) pointer to it; this may seem less efficient (due to issues with escape analysis and Go interfaces, it often is not), but guarantees immutability, which is more important for diagnostics anyway.
func CustomComparisonMethod(methodnames ...string) EqualityComparisonFunction {
	// Mostly Copy&Pasted from specialized function for (single) methodname  == "IsEqual" and adapted.
	// TODO: Check in-code comments
	return func(x, y any) (isEqual bool) {
		// Note: if x or y are nil, these are invalid objects. We special case these below.
		// Because of that, we cannot use xType = xValue.Type(), because that would panic for x==nil
		xValue := reflect.ValueOf(x)
		xType := reflect.TypeOf(x)
		yValue := reflect.ValueOf(y)
		yType := reflect.TypeOf(y)

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

		for _, methodname := range methodnames {

			// MethodByName follows Go's conventions and will find methods defined with value receivers if xType is a value type
			// If xType is a pointer type, it will find methods defined with BOTH value and pointer receivers.
			// So we need to check for non-pointer xType first.
			if EqualMethod, found := xType.MethodByName(methodname); found {
				// method found with value receiver x
				t := EqualMethod.Type // t.Kind is reflect.Function.

				if t.NumIn() != 2 {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an %v method with wrong number of arguments", x, methodname))
				}

				// The first argument is the receiver, i.e. xValue. We need to figure out if the second one should be yValue or a reflect.Value corresponding to a pointer.
				secondArgType := t.In(1)
				var worksWithValueArg bool = yType.AssignableTo(secondArgType)
				var worksWithPtrArg bool = reflect.PtrTo(yType).AssignableTo(secondArgType)
				if !worksWithPtrArg && !worksWithValueArg {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an %v method, but it cannot take an argument of type %T", x, methodname, y))
				}

				var callResults []reflect.Value
				if worksWithValueArg {
					callResults = EqualMethod.Func.Call([]reflect.Value{xValue, yValue})
				} else {
					// Need to call methodname on "&y", more precisely on a pointer to the value held inside the interface y.
					// This is not possible without copying y, since Go cannot take addresses of values inside an interface.
					callResults = EqualMethod.Func.Call([]reflect.Value{xValue, utils.PointerToCopy(yValue)})
				}

				if len(callResults) == 0 {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an %v method, but it this method does not return anything", x, methodname))
				}
				var result any = callResults[0].Interface()
				// Note: We type-assert result.(bool) rather than checking EqualMethod.Out(0) == reflect.TypeOfType[bool]() before the call.
				// This has the advantage of directly checking the *dynamic* type of the returned value.
				resultBool, ok := result.(bool)
				if !ok {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparsion, type %T had an %v method, but its first return value is not bool", x, methodname))
				}
				return resultBool
			}

			// Check for pointer receiver:
			xPointerType := reflect.PointerTo(xType)
			if EqualMethod, found := xPointerType.MethodByName(methodname); found {
				// method found with pointer receiver x
				t := EqualMethod.Type // t.Kind is reflect.Function.

				if t.NumIn() != 2 {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an %v method with wrong number of arguments", x, methodname))
				}

				// first argument is the receiver. As above, we need to make a copy (this time of x)
				xCopyPointer := utils.PointerToCopy(xValue)

				secondArgType := t.In(1)
				var worksWithValueArg bool = yType.AssignableTo(secondArgType)
				var worksWithPtrArg bool = reflect.PtrTo(yType).AssignableTo(secondArgType)
				if !worksWithPtrArg && !worksWithValueArg {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an %v method, but it cannot take an argument of type %T", x, methodname, y))
				}

				var callResults []reflect.Value
				// This time, we check the pointer case first:
				if worksWithPtrArg {
					callResults = EqualMethod.Func.Call([]reflect.Value{xCopyPointer, utils.PointerToCopy(yValue)})
				} else {
					callResults = EqualMethod.Func.Call([]reflect.Value{xCopyPointer, yValue})
				}

				if len(callResults) == 0 {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an %v method, but it this method does not return anything", x, methodname))
				}
				var result any = callResults[0].Interface()
				// Note: We type-assert result.(bool) rather than checking EqualMethod.Out(0) == reflect.TypeOfType[bool]() before the call.
				// This has the advantage of directly checking the *dynamic* type of the returned value.
				resultBool, ok := result.(bool)
				if !ok {
					panic(fmt.Errorf(ErrorPrefix+"during equality comparsion, type %T had an %v method, but its first return value is not bool", x, methodname))
				}
				return resultBool
			}
		}

		return x == y // may panic

	}
}

// comparison_IsEqual compares x and y for equality with the following quirks:
//
//   - if either x or y are the nil interface, then the comparison result is true iff the other argument is either a nil interface or a nil of concrete type
//   - otherwise, if either x or y are pointer types, then the comparison will directly compare the pointers.
//   - otherwise, if x has an IsEqual method (on either pointer or value receiver), we will call x.IsEqual(y) resp. x.IsEqual(&y)
//     -- Whether IsEqual is called with &y or y is deducted from the function signature. If both options are valid, we match the way x is passed.
//     -- The IsEqual method must return a bool (possibly inside an interface) as its first return value; further return values are discarded.
//     -- If the IsEqual method has the wrong signature, we panic
//     -- Note that when calling with a pointer receiver or argument, we actuall pass a pointer to a copy of x or y.
//   - otherwise, if there is no IsEqual method, we resort to plain == (which may panic)
//
// NOTE: Plain comparison takes precendence over an IsEqual method if either x or y are pointers.
//
// This means that if an IsEqual method is defined on a pointer receiver (say, for efficiency of passing arguments),
// you need to pass the non-pointer value to Comparison_IsEqual if you want to use the custom method.
// Essentially, we assume that pointer receivers are chosen *solely* for argument passing efficiency and the custom equality semantics
// are conceptually on the value-types, not the pointer type (notwithstanding that Go does not think of types that way to start with).
// Since data stored in ErrorsWithData should be values, not pointers unless you are really interested in the memory address, this is the appropriate behaviour.
//
// Comparison_IsEqual is functionally equivalent to [CustomComparisonMethod]("IsEqual")
func Comparison_IsEqual(x, y any) (isEqual bool) {
	// Note: if x or y are nil, these are invalid objects. We special case these below.
	// Because of that, we cannot use xType = xValue.Type(), because that would panic for x==nil
	xValue := reflect.ValueOf(x)
	xType := reflect.TypeOf(x)
	yValue := reflect.ValueOf(y)
	yType := reflect.TypeOf(y)

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

	// MethodByName follows Go's conventions and will find methods defined with value receivers if xType is a value type
	// If xType is a pointer type, it will find methods defined with BOTH value and pointer receivers.
	// So we need to check for non-pointer xType first.
	if EqualMethod, found := xType.MethodByName("IsEqual"); found {
		// method found with value receiver x
		t := EqualMethod.Type // t.Kind is reflect.Function.

		if t.NumIn() != 2 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method with wrong number of arguments", x))
		}

		// The first argument is the receiver, i.e. xValue. We need to figure out if the second one should be yValue or or reflect.Value corresponding to a pointer.

		secondArgType := t.In(1)
		var worksWithValueArg bool = yType.AssignableTo(secondArgType)
		var worksWithPtrArg bool = reflect.PtrTo(yType).AssignableTo(secondArgType)
		if !worksWithPtrArg && !worksWithValueArg {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method, but it cannot take an argument of type %T", x, y))
		}

		// call EqualMethod:
		var callResults []reflect.Value
		if worksWithValueArg {
			callResults = EqualMethod.Func.Call([]reflect.Value{xValue, yValue})
		} else {
			// Need to call EqualMethod on "&y", more precisely on a pointer to the value held inside the interface y.
			// This requires making a copy of y.
			callResults = EqualMethod.Func.Call([]reflect.Value{xValue, utils.PointerToCopy(yValue)})
		}

		if len(callResults) == 0 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, %T had an IsEqual method, but it this method does not return anything", x))
		}
		var result any = callResults[0].Interface()
		// Note: We type-assert result.(bool) rather than checking EqualMethod.Out(0) == reflect.TypeOfType[bool]() before the call.
		// This has the advantage of directly checking the *dynamic* type of the returned value.
		resultBool, ok := result.(bool)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparsion, type %T had an IsEqual method, but its first return value is not bool", x))
		}
		return resultBool
	}

	// Check for pointer receiver, i.e. whether &x has an IsEqual method:
	xPointerType := reflect.PointerTo(xType)
	if EqualMethod, found := xPointerType.MethodByName("IsEqual"); found {
		// method found with pointer receiver x
		t := EqualMethod.Type // t.Kind is reflect.Function.

		if t.NumIn() != 2 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an IsEqual method with wrong number of arguments", x))
		}

		// first argument is the receiver. As above, we need to make a copy (this time of x)
		xCopyPointer := utils.PointerToCopy(xValue)

		secondArgType := t.In(1)
		var worksWithValueArg bool = yType.AssignableTo(secondArgType)
		var worksWithPtrArg bool = reflect.PtrTo(yType).AssignableTo(secondArgType)
		if !worksWithPtrArg && !worksWithValueArg {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an IsEqual method, but it cannot take an argument of type %T", x, y))
		}

		// This time, we check the pointer case first:
		var callResults []reflect.Value
		if worksWithPtrArg {
			callResults = EqualMethod.Func.Call([]reflect.Value{xCopyPointer, utils.PointerToCopy(yValue)})
		} else {
			callResults = EqualMethod.Func.Call([]reflect.Value{xCopyPointer, yValue})
		}

		if len(callResults) == 0 {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparison, *%T had an IsEqual method, but it this method does not return anything", x))
		}
		var result any = callResults[0].Interface()
		// Note: We type-assert result.(bool) rather than checking EqualMethod.Out(0) == reflect.TypeOfType[bool]() before the call.
		// This has the advantage of directly checking the *dynamic* type of the returned value.
		resultBool, ok := result.(bool)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"during equality comparsion, type %T had an IsEqual method, but its first return value is not bool", x))
		}
		return resultBool
	}

	return x == y // may panic

}
