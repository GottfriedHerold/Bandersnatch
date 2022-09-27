package testutils

import (
	"fmt"
	"reflect"
	"strings"
)

// CheckPanic runs fun(args...), captures all panics() and returns whether a panic occurred.
// It does not re-raise or return the actual panic argument
// (unless the panic argument is a string starting with "reflect" -- likely an error from the reflect package)
//
// This function is only used in testing. It does not work with untyped nil arguments to variadic functions fun due to limitations of Go1.18.
func CheckPanic(fun interface{}, args ...interface{}) (didPanic bool) {
	didPanic = true
	fun_reflected := reflect.ValueOf(fun)
	if fun_reflected.Kind() != reflect.Func {
		panic("checkPanic's first argument must be a function")
	}
	fun_type := fun_reflected.Type()
	if !fun_type.IsVariadic() {
		if fun_type.NumIn() != len(args) {
			panic("checkPanic called with wrong number of arguments")
		}
	}

	function_arguments := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		// reflect.ValueOf(nil) gives the zero value of reflect.Value,
		// which is NOT a valid reflection of an actual nil interface, but rather "no reflect.Value"
		// (untyped nil / nil interface simply HAS no appropriate reflect.Value)
		// There is actually a proposal to change that for future Go versions.
		// As of Go1.18, we need to specifically catch that case,
		// analyze the type of argument that fun_reflected expects and create a nil value
		// of appropriate interface/chan/map/func/pointer/slice type

		// checking for untyped nil
		if args[i] == nil {
			// Not worth the hassle to get that right.
			if fun_type.IsVariadic() {
				panic("bandersnatch / testing framework: CheckPanic does not work with untyped nil arguments for variadic functions. Sorry about that.")
			}
			nilArgType := fun_reflected.Type().In(i)
			switch nilArgType.Kind() {
			case reflect.Interface, reflect.Chan, reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func:
				function_arguments[i] = reflect.Zero(nilArgType)
			default:
				panic(fmt.Errorf("bandersnatch / testing framework: CheckPanic called with function and untyped nil as %v'th argument, but the function does not take an interface/channel/map/slice/func/pointer that could be nil for that argument", i))
			}
		} else {
			function_arguments[i] = reflect.ValueOf(args[i])
		}
	}
	// we catch all panics. Unfortunately, this also catches panics *inside* the reflect package from malformed calls to
	// reflect.Value.Call due to using checkPanic wrongly (e.g. by using a wrong number/type of args).
	// HACK: 	we check whether the panic's error starts with reflect (by convention, such errors start with the package name --
	// 			indeed, checking my used standard library source code for Go1.17 confirms this)
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		var errstring string
		switch err := err.(type) {
		case string:
			errstring = err
		case error:
			errstring = err.Error()
		case fmt.Stringer:
			errstring = err.String()
		default:
			return
		}
		if strings.HasPrefix(errstring, "reflect") {
			panic(err)
		}
	}()
	fun_reflected.Call(function_arguments)
	didPanic = false
	return
}
