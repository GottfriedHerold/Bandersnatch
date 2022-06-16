package errorsWithData

import (
	"fmt"
	"unicode/utf8"
)

// TODO: Global Renaming

// errorWithParameters_common is a simple implementation of the ErrorWithParameters interface
// NOTE:
// functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned as typed nil pointers, which is a serious footgun)
type errorWithParameters_common struct {
	contained_error error          // wrapped underlying error
	message         string         // if not "", overrides the error string.
	params          map[string]any // map strings -> data.
	// showparams      bool           // should we show embedded data on error
}

// extension of errorWithParameters_common that satisfies ErrorWithParameters[StructType]
type errorWithParameters_T[StructType any] struct {
	errorWithParameters_common
}

// Error is provided to satisfy the error interface
func (e *errorWithParameters_common) Error() string {
	if e == nil {
		panic(errorPrefix + "called Error() on nil error of concrete type errorWithParams. This is a bug, since nil errors of this type should never exist.")
	}
	s, formattingError := formatError(e.message, e.params, e.contained_error, true)
	if formattingError != nil {
		// TODO: Callback?
		panic(formattingError)
	}
	return s

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
func (e *errorWithParameters_T[StructType]) GetData() (ret StructType) {
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
func (e *errorWithParameters_common) GetAllParameters() (ret map[string]any) {
	ret = make(map[string]any)
	for key, value := range e.params {
		ret[key] = value
	}
	return
}

// TODO: Syntax-check the override message?

func makeErrorWithParametersCommon(baseError error, overrideMessage string) (ret errorWithParameters_common) {
	if !utf8.ValidString(overrideMessage) {
		panic(errorPrefix + "override message for error creation was not a valid UTF-8 string")
	}
	if overrideMessage == "" {
		overrideMessage = DefaultOverrideMessage
	} else if overrideMessage == OverrideByEmptyMessage {
		overrideMessage = ""
	}
	_, formattingError := formatError(overrideMessage, nil, nil, false)
	if formattingError != nil {
		panic(fmt.Errorf(errorPrefix+"creating of an error with parameters failed, because the error override message was malformed.\noverrideMessage = %v.\nreported error was: %v", overrideMessage, formattingError))
	}
	ret.contained_error = baseError
	ret.message = overrideMessage
	ret.params = GetAllParametersFromError(baseError)
	return
}
