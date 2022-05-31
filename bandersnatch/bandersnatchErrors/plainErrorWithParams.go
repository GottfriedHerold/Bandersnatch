package bandersnatchErrors

import (
	"fmt"
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
	showparams      bool           // should we show embedded data on error
}

type errorWithParameters_T[T any] struct {
	errorWithParameters_common
}

// Error is provided to satisfy the error interface
func (e *errorWithParameters_common) Error() string {
	if e == nil {
		panic("bandersnatch / error handling: called error on nil error of concrete type errorWithParams. This is likely a bug, since nil errors of concrete type should never exist.")
	}
	var mstring string = ""
	if e.showparams {
		m := GetAllParametersFromError(e) // this also get parameters from wrapped errors
		if len(m) > 0 {
			mstring = fmt.Sprintf("\n Data contained in error is:\n%v", m)
		}
	}
	if e.message != "" {
		return e.message + mstring
	} else {
		return e.contained_error.Error() + mstring
	}
}

// Unwrap is provided to work with errors.Is
func (e *errorWithParameters_common) Unwrap() error {
	// Note: This panics if e is nil (of type *errorWithParams).
	// While returnining untyped nil would give "meaningful" behaviour
	// (including for the recursive calls in HasParameter etc.),
	// we consider any nil pointer of concrete error type a bug.
	return e.contained_error
}

// getAllParameters returns the map of all parameters present, NOT following the error chain.
func (e *errorWithParameters_common) getAllParameters() map[string]any {
	return e.params
}

func (e *errorWithParameters_common) showParametersOnError() bool {
	return e.showparams
}

func (e *errorWithParameters_common) withShowParametersOnError(newVal bool) errorWithParameters_commonInterface {
	var ret errorWithParameters_common = *e // Note: This shallow-copies the map and any error chain, but since it's immutable, it's OK.
	ret.showparams = newVal
	return &ret
}

// getParameter returns the value stored under parameterName and whether is was present.
func (e *errorWithParameters_common) getParameter(parameterName string) (any, bool) {
	data, ok := e.params[parameterName]
	if !ok {
		// Testing hook only, probably better to install a callback
		if GetDataPanicOnNonExistantKeys {
			err := fmt.Errorf("bandersnatch / error handling: Requesting non-existing parameter %v (lowercased) from an errorWithParams. The error from which the parameter was requested was %v", parameterName, e)
			panic(err)
		}
		return nil, false
	}
	return data, true
}

// hasParameter returns whether the key parameterName exists.
func (e *errorWithParameters_common) hasParameter(parameterName string) bool {
	_, ok := e.params[parameterName]
	return ok
}

func (e *errorWithParameters_T[T]) Get() (ret T) {
	allParams := GetAllParametersFromError(e)
	ret, err := makeStructFromMap[T](allParams)
	if err != nil {
		panic(err)
	}
	return
}
