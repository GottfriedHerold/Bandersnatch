package bandersnatch

import (
	"reflect"
	"sync"
)

/*
	This file contains error handling code that get called whenever a NaP (Not-A-Point) is encountered.
	Apart from bugs within the library, such points can occur due to using uninitialized variables and continuing to work with them.
	Also, certain exceptional cases for some group operations can result in this (we take care they cannot appear while working on the good subgroup)
	To protect the user from mistakes (the naive implementation would give true for comparisons involving NaPs -- this is potentially a disaster if used in any kind of cryptographic verification),
	we check for NaP-ness whenever we "exit" the library, i.e. on Equality or Zero-Ness comparisons.
	If a NaP is found, we call an error handler that the user can replace by whatever he wishes.
	Panic is appropriate here IMO, but there is a school of thought that strongly dislikes panic, so the user can choose.
*/

// NaPErrorHandler is the function type for error handler that can be installed for dealing with NaPs.
// When such a handler is called, reason is a descriptive error message, comparison tells whether the NaP was detected during a comparison (in which case the return value of the function should be the comparison result),
// points is a number of relevant points that were involved.
type NaPErrorHandler func(reason string, comparison bool, points ...CurvePointPtrInterfaceRead) bool

// empty error handler.
func trivial_error_handler(string, bool, ...CurvePointPtrInterfaceRead) bool {
	return false
}

// error handler that just panics.
func panic_error_handler(reason string, _ bool, _ ...CurvePointPtrInterfaceRead) bool {
	// Should be also log / output the points?
	panic(reason)
}

// currently installed NaP error handler. Being a mutable global object, this is protected by mutex.
// Note that the mutex is not held when the error handler is *called* -- Thread-safety of the error handlers themselves is the duty of the user-provided handlers.
// The mutex only protects the variable itself.
var current_error_handler NaPErrorHandler = trivial_error_handler
var error_handler_mutex sync.Mutex

// SetNaPErrorHandler exchanges the currently installed handler that is called when a NaP is encountered with the one provided and returns the previously installed one.
func SetNaPErrorHandler(new_handler NaPErrorHandler) (old_handler NaPErrorHandler) {
	error_handler_mutex.Lock()
	defer error_handler_mutex.Unlock()
	old_handler = current_error_handler
	current_error_handler = new_handler
	return
}

// GetNaPErrorHandler gets the currently installed error handler for dealing with NaPs.
func GetNaPErrorHandler() NaPErrorHandler {
	error_handler_mutex.Lock()
	defer error_handler_mutex.Unlock()
	f := current_error_handler
	return f
}

// This function is called whenever we hit a NaP. reason is an error string, comparison tells whether this was in an (equality or zeroness) comparison, points are relevant points that may be useful for debugging.
// It calls the user-provided error handler with reason, comparison, points. The output is taken as the comparison result (if the error handler does not panic, false is the reasonable choice) if comparison is true.
func napEncountered(reason string, comparison bool, points ...CurvePointPtrInterfaceRead) bool {
	// Note that this essentially locks the mutex, gets a copy of the handler and releases the mutex before actually calling f.
	f := GetNaPErrorHandler()
	return f(reason, comparison, points...)
}

// was InvalidPointEncountered(f) calls f() and returns whether the napEncountered - error handler was triggered during execution of f.
// This function is only used in testing.
func wasInvalidPointEncountered(fun func()) bool {
	var old_handler_ptr *NaPErrorHandler // indirection to avoid the need for manually locking and circumventing the SetNaPErrorHandler interface.
	var error_bit bool = false
	new_handler := func(reason string, comparison bool, points ...CurvePointPtrInterfaceRead) bool {
		error_bit = true
		return (*old_handler_ptr)(reason, comparison, points...)
	}
	old_handler := SetNaPErrorHandler(new_handler)
	old_handler_ptr = &old_handler
	defer SetNaPErrorHandler(*old_handler_ptr)
	fun()
	return error_bit
}

// guardForInvalidPoints checks whether fun(args...)==expected, while monitoring for NaP-handling. expect_singular determines whether we expect a NaP.
// This function is only used in testing.
func guardForInvalidPoints(expected interface{}, expect_singular bool, error_message string, fun interface{}, args ...interface{}) (ok bool, err string) {
	var old_handler_ptr *NaPErrorHandler
	var error_bit bool = false
	new_handler := func(reason string, comparison bool, points ...CurvePointPtrInterfaceRead) bool {
		error_bit = true
		return (*old_handler_ptr)(reason, comparison, points...)
	}
	old_handler := SetNaPErrorHandler(new_handler)
	old_handler_ptr = &old_handler
	defer SetNaPErrorHandler(*old_handler_ptr)
	var got interface{}
	fun_reflected := reflect.ValueOf(fun)
	// To get better error messages
	if fun_reflected.Kind() != reflect.Func {
		panic("guardForInvalidPoints must be called with a function as fourth argument")
	}
	function_arguments := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		function_arguments[i] = reflect.ValueOf(args[i])
	}
	got = fun_reflected.Call(function_arguments)[0].Interface()
	if expect_singular != error_bit {
		if error_bit {
			return false, "When performing check with intended error message " + error_message + ", an unexpected NaP was encountered"
		}
		return false, "When performing check with intended error message " + error_message + ", NaP handler was not called, but was expected to."
	}
	if expected != got {
		return false, error_message // outputing expected and got would be nice, but that is messy (because we cannot ask expected and got to be fmt.Stringers, as bools would not satisfy that)
	}
	return true, ""
}
