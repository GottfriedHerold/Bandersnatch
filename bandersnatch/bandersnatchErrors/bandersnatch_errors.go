package bandersnatchErrors

import (
	"errors"
	"fmt"
	"strings"
)

// NOTE: We might not export this at all?

// BandersnatchError is an error (wrapper) that wraps an arbitrary error, but extends the error interface by
// allowing to embed arbitrary data.
// Data is embedded as a strings -> interface{} map with case-insensitive strings.
//
// Note that a similar functionality that is more likely to fit users' needs is also available by free functions
// GetDataFromError, AddDataToError, HasData.
// These free functions work on plain errors (by type-asserting internally)
// and for wrapped errors actually traverse the error chain.
// The methods with BandersnatchError receivers do NOT traverse the error chain.
type BandersnatchError interface {
	error
	Unwrap() error
	// GetData obtains the parameter stored under the key parameterName. Returns the value and whether it was present.
	// panics on nil pointer receivers of concrete type. If wasPresent == false, value is nil
	GetData(parameterName string) (value any, wasPresent bool)
	// HasData tests if the parameter stored under the key is present.
	// As opposed to GetData, concretePtrType(nil).HasData(...) returns false and does not panic.
	HasData(parameterName string) bool
	// AddData records the value given by newParameter under the key given by parameter name.
	// Panics on nil receivers.
	AddData(parameterName string, newParameter any) (oldValue any, wasPresent bool)
	// DeleteData deletes the parameter keyed by parameterName. Does nothing if the key was not present to start with.
	// panics on nil receivers.
	DeleteData(parameterName string) (oldValue any, wasPresent bool)
}

// keys for parameterName. We use constants so the IDE/compiler yells at us if we typo.

const BYTES_READ_FLAG = "partialread"
const IO_ERROR_FLAG = "ioerror"

// Exported for cross-package testing. Will be removed/replaced by callback. Not part of the official interface
var GetDataPanicOnNonExistantKeys = false

// errorWithParams is a simple implementation of the BandersnatchError interface
// NOTE:
// functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned with as typed nil pointers, which is a serious footgun)
type errorWithParams struct {
	contained_error error          // wrapped underlying error
	message         string         // if not "", overrides the error string.
	params          map[string]any // map strings -> data. The string keys are all lowercased.
}

// Error is provided to satisfy the error interface
func (e *errorWithParams) Error() string {
	if e.message != "" {
		return e.message
	} else {
		return e.contained_error.Error()
	}
}

// Unwrap is provided to work with errors.Is
func (e *errorWithParams) Unwrap() error {
	// Note: This panics if e is nil (of type *errorWithParams)
	// While returnining untyped nil would give "meaningful" behaviour
	// (including for the recursive calls in HasData etc.),
	// we consider any nil pointer of concrete error type a bug.
	return e.contained_error
}

// GetData returns the value stored under parameterName and whether is was present.
func (e *errorWithParams) GetData(parameterName string) (any, bool) {
	parameterName = strings.ToLower(parameterName) // make parameterName case-insensitive. The map keys are all lower-case
	data, ok := e.params[parameterName]
	if !ok {
		// Testing hook only, probably better to install a callback
		if GetDataPanicOnNonExistantKeys {
			err := fmt.Errorf("bandersnatch / error handling: Requesting non-existing parameter %v (lowercased) from a BandersnatchError. The error from which the parameter was requested was %v", parameterName, e)
			panic(err)
		}
		return nil, false
	}
	return data, true
}

// HasData returns whether the key parameterName exists.
// This returns false on nil receivers of concrete type.
func (e *errorWithParams) HasData(parameterName string) bool {
	if e == nil {
		return false
	}
	parameterName = strings.ToLower(parameterName) // make parameterName case-insensitive. The map keys are all lower-case
	_, ok := e.params[parameterName]
	return ok
}

// AddData sets the value stored under parameterName to newParameter
func (e *errorWithParams) AddData(parameterName string, newParameter any) (oldValue any, wasPresent bool) {
	parameterName = strings.ToLower(parameterName) // make parameterName case-insensitive. The map keys are all lower-case
	oldValue, wasPresent = e.params[parameterName]
	e.params[parameterName] = newParameter
	return
}

// probably rarely used.

// DeteleData deletes the value stored under the given key.
func (e *errorWithParams) DeleteData(parameterName string) (oldValue any, wasPresent bool) {
	parameterName = strings.ToLower(parameterName)
	oldValue, wasPresent = e.params[parameterName]
	delete(e.params, parameterName)
	return
}

// HasData checks whether some error in err's error chain contains a parameter keyed by (lowercased) parameterName
// HasData(nil, ...) returns false
func HasData(err error, parameterName string) bool {
	if err == nil {
		return false
	}
	errWithParams, ok := err.(BandersnatchError)
	if ok {
		if errWithParams.HasData(parameterName) {
			return true
		}
	}
	return HasData(errors.Unwrap(err), parameterName)
}

// GetDataFromError returns the value stored under the key parameterName in the first error in err's error chain
// where some entry was found. If no entry was found in the error chain, returns nil, false.
func GetDataFromError(err error, parameterName string) (value any, wasPresent bool) {
	if err == nil {
		return nil, false
	}
	errWithParams, ok := err.(BandersnatchError)
	if ok {
		if errWithParams.HasData(parameterName) {
			return errWithParams.GetData(parameterName)
		}
	}
	return GetDataFromError(errors.Unwrap(err), parameterName)
}

// AddDataToError modifies *err, adding newParameter under the key parameterName.
// The error is modified to satisfy BandersnatchError (if non-nil). In particular, the type of *err might change.
// if err points to a nil error, *err is not modified.
func AddDataToError(err *error, parameterName string, newParameter any) (oldValue any, wasPresent bool) {
	if *err == nil {
		return nil, false
	}
	parameterName = strings.ToLower(parameterName)
	newError, ok := (*err).(BandersnatchError)
	if ok {
		return newError.AddData(parameterName, newParameter)
	}
	oldValue, wasPresent = GetDataFromError(*err, parameterName)
	newError = NewErrorWithParams(*err, "", parameterName, newParameter)
	*err = newError
	return oldValue, wasPresent
}

// TODO: We might not export NewErrorWithParams

// NOTE: We return an interface on purpose, because of potential nil's
// Also, even if we don't return nil, code such as
// The reason is that otherwise
// func() error{
// 	  e:=NewErrorWithParams(...)
//    [...]
//    e = nil
//    return e
// }
// becomes a footgun (because it creates a typed nil pointer in an error interface. For the caller of this err==nil is false!)

// NewErrorWithParams creates a new errorWithParams (wrapped in an interface) wrapping the given error, override message and (optional) parameters.
// For err == nil and overrideMessage == "", #params > 0, we panic
// For err == nil, overrideMessage == "", #params == 0, we return a nil interface
// For err == nil, overrideMessage != "", the returned error behaves like a new error that does not wrap an error.
func NewErrorWithParams(err error, overrideMessage string, params ...any) BandersnatchError {
	if len(params)%2 != 0 {
		panic("bandersnatch / error handling: called NewErrorWithParams(err, overrideMessage, args...) with an odd number of args. These are supposed to be parameter-name, parameter pairs")
	}
	extraParams := len(params) / 2
	if err == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic("bandersnatch / error handling: called NewErrorWithParams(nil,\"\",argName, arg1, ...)")
			}
			return nil
		}
		// We have contained_error == nil in this case. This is actually fine.
	}
	ret := errorWithParams{contained_error: err, message: overrideMessage, params: make(map[string]any)}
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf("bandersnatch / error handling: called NewErrorWithParams(err, overrideMessage, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		s = strings.ToLower(s)
		ret.params[s] = params[2*i+1]
	}
	return &ret
}
