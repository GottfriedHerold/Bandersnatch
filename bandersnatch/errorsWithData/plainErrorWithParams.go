package errorsWithData

import (
	"fmt"
	"unicode/utf8"
)

// errorWithParameters_common is a simple implementation of the ErrorWithData_any interface
// NOTE:
// exported functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned as typed nil pointers, which is a serious footgun)
type errorWithParameters_common struct {
	contained_error error          // wrapped underlying error, can be nil
	message         string         // error message, subject to interpolation
	params          map[string]any // map strings -> data.
}

// extension of errorWithParameters_common that satisfies ErrorWithData[StructType]
type errorWithParameters_T[StructType any] struct {
	errorWithParameters_common
}

// Error_interpolate is provided to satisfy the ErrorWithData_any interface. This is an extension of Error() that
// will interpolate the error string (which may contain formatting instructions like %{Param1}, ${Param2} ).
// As opposed to plain Error(), formatting instructions with $ differs from those with % by taking the parameters from params_passed rather than using the
// parameters stored in the error. The presence of this method is what makes the "$w" formatting verb work.
func (e *errorWithParameters_common) Error_interpolate(params_passed map[string]any) string {
	if e == nil {
		panic(ErrorPrefix + "called Error_interpolate() on nil error of concrete type errorWithParameters_common. This is a bug, since nil errors of this type should never exist.")
	}
	s, formattingError := formatError_new(e.message, e.params, params_passed, e.contained_error, full_eval)
	if formattingError != nil {
		// TODO: error handler
		panic(formattingError)
	}
	return s
}

// Error is provided to satisfy the error interface
func (e *errorWithParameters_common) Error() string {
	if e == nil {
		panic(ErrorPrefix + "called Error() on nil error of concrete type errorWithParameters_common. This is a bug, since nil errors of this type should never exist.")
	}
	return e.Error_interpolate(e.params)
}

// Unwrap is provided to work with errors.Is
func (e *errorWithParameters_common) Unwrap() error {
	// Note: This panics if e is nil (of type *errorWithParams).
	// While returnining untyped nil would give "meaningful" behaviour
	// (including for the recursive calls in HasParameter etc.),
	// we consider any nil pointer of concrete error type a bug.
	return e.contained_error
}

// GetData is provided to satisfy ErrorWithParameters[StructType].
//
// It constructs a value of type StructType from the provided parameters.
func (e *errorWithParameters_T[StructType]) GetData_struct() (ret StructType) {
	ret, err := makeStructFromMap[StructType](e.params)
	if err != nil {
		panic(err)
	}
	return
}

// HasParameter checks whether the parameter given by the name is present.
func (e *errorWithParameters_common) HasParameter(parameterName string) bool {
	_, ok := e.params[parameterName]
	return ok
}

// GetParameter retrieves the parameter stored under the key parameterName and whether it was present.
//
// On keys that were not present, returns nil, false.
func (e *errorWithParameters_common) GetParameter(parameterName string) (value any, present bool) {
	value, present = e.params[parameterName]
	return
}

// GetAllParameters returns a map of all parameters present in the error.
// The returned map is a (shallow) copy, so modification of values of the returned map does not affect the error.
func (e *errorWithParameters_common) GetData_map() (ret map[string]any) {
	ret = make(map[string]any)
	for key, value := range e.params {
		ret[key] = value
	}
	return
}

// makeErrorWithParameterCommon creates a new errorWithParameters_common from the given baseError and override message.
// NOTE: This is an internal function, used by the (exported) functions in modify_errors.go
// These functions actually modify ret.params after creation. The purpose of this method is just to unify those functions.
//
// NOTE: This functions performs a parse check, i.e. we check that override message is well-formed and panic if not.
func makeErrorWithParametersCommon(baseError error, overrideMessage string) (ret errorWithParameters_common) {
	if !utf8.ValidString(overrideMessage) {
		panic(ErrorPrefix + "override message for error creation was not a valid UTF-8 string")
	}
	if overrideMessage == "" {
		overrideMessage = DefaultOverrideMessage
	} else if overrideMessage == OverrideByEmptyMessage {
		overrideMessage = ""
	}
	ret.contained_error = baseError
	ret.message = overrideMessage
	ret.params = GetData_map(baseError)
	_, formattingError := formatError_new(overrideMessage, ret.params, ret.params, baseError, parse_check)
	if formattingError != nil {
		panic(fmt.Errorf(ErrorPrefix+"creating of an error with parameters failed, because the error message was malformed.\noverrideMessage = %v.\nreported error was: %v", overrideMessage, formattingError))
	}
	return
}
