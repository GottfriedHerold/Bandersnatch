package bandersnatchErrors

import (
	"errors"
	"fmt"
	"strings"
)

// NewErrorWithParameters creates a new errorWithParams (wrapped in an interface) wrapping the given error, override message and (optional) parameters.
// Note: The difference between this and IncludeParametersInError is the message and nil handling:
// For err == nil and overrideMessage == "", #params > 0, we panic
// For err == nil, overrideMessage == "", #params == 0, we return a nil interface
// For err == nil, overrideMessage != "", the returned error behaves like a new error that does not wrap an error.
func NewErrorWithParameters(err error, overrideMessage string, params ...any) PlainErrorWithParameters {
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

// IncludeParametersInError creates a new error wrapping err with parameter under parameterName set to newParameter.
// As opposed to if err == nil, returns nil
func IncludeParametersInError(err error, parameterName string, newParameter any, furtherParameters ...any) PlainErrorWithParameters {
	if err == nil {
		return nil
	}
	if len(furtherParameters) > 0 {
		newError := NewErrorWithParameters(err, "", furtherParameters...)
		newError.(*errorWithParams).params[parameterName] = newParameter
		return newError
	}
	return NewErrorWithParameters(err, "", parameterName, newParameter)
}

// HasParameter checks whether some error in err's error chain contains a parameter keyed by parameterName
// HasParameter(nil, ...) returns false
func HasParameter(err error, parameterName string) bool {
	if err == nil {
		return false
	}
	errWithParams, ok := err.(PlainErrorWithParameters)
	if ok {
		if errWithParams.hasParameter(parameterName) {
			return true
		}
	}
	return HasParameter(errors.Unwrap(err), parameterName)
}

// GetParameterFromError returns the value stored under the key parameterName in the first error in err's error chain
// where some entry was found.
// If no entry was found in the error chain or err==nil, returns nil, false.
func GetParameterFromError(err error, parameterName string) (value any, wasPresent bool) {
	if err == nil {
		return nil, false
	}
	errWithParams, ok := err.(PlainErrorWithParameters)
	if ok {
		if errWithParams.hasParameter(parameterName) {
			return errWithParams.getParameter(parameterName)
		}
	}
	return GetParameterFromError(errors.Unwrap(err), parameterName)
}

// GetAllParametersFromError returns a map for all parameters stored in the error, including all of err's error chain.
// For err==nil, returns nil. If no error in err's error chain has any data, returns an empty map.
func GetAllParametersFromError(err error) map[string]any {
	if err == nil {
		return nil
	}
	// Find all errors it the error chain which satisfy the ErrorWithParameters interface
	relevantErrors := make([]*PlainErrorWithParameters, 0)
	for errChain := err; errChain != nil; errChain = errors.Unwrap(errChain) {
		errChainGood, ok := errChain.(PlainErrorWithParameters)
		if ok {
			if errChainGood == nil {
				panic("bandersnatch / error handling: typed nil error satisfying the ErrorWithParameters interface in error chain detected")
			}
			relevantErrors = append(relevantErrors, &errChainGood)
		}
	}
	// Build up the resulting map by going through the relevant ErrorWithParameters starting from the end of the error chain.
	ret := make(map[string]any)
	for i := len(relevantErrors) - 1; i >= 0; i-- {
		m := (*relevantErrors[i]).getAllParameters()
		if m == nil {
			panic("bandersnach / error handling: getParams returned nil for error satifying ErrorWithParameters in error chain")
		}
		for key, value := range m {
			ret[key] = value
		}
	}
	return ret
}
