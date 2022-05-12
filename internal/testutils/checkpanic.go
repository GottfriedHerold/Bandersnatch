package testutils

import (
	"fmt"
	"reflect"
	"strings"
)

// CheckPanic runs fun(args...), captures all panics() and returns whether a panic occurred.
// It does not re-raise or return the actual panic argument (unless the panic argument is a string starting with "reflect" -- likely an error from the reflect package)
//
// This function is only used in testing.
func CheckPanic(fun interface{}, args ...interface{}) (didPanic bool) {
	didPanic = true
	fun_reflected := reflect.ValueOf(fun)
	if fun_reflected.Kind() != reflect.Func {
		panic("checkPanic's first argument must be a function")
	}
	function_arguments := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		function_arguments[i] = reflect.ValueOf(args[i])
	}
	// we catch all panics. Unfortunately, this also catches panics *inside* the reflect package from malformed calls to
	// reflect.Value.Call due to using checkPanic wrongly (e.g. by using a wrong number/type of args).
	// HACK: 	we check whether the panic's error starts with reflect (by convention, such errors start with the package name --
	// 			indeed, checking my used standard library source code for Go1.17 confirms this; )
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
