package errorsWithData

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file contains functions used to create / modify errors.
// Due to immutability of errors, modifications really means return modified copy.
//
// On the implementations side, there are 2 subtleties to note here:
// - We unbox any input base error via incomparabilityUndoer_any, if applicable.
//   This is because we do not want incomparable errors to appear in actual error chains,
//   because this could trigger bugs in 3rd party code, which might silently make assumptions here.
//   The sole purpose of incomparable errors is to export them for users to compare against with
//   errors.Is, documenting potential errors and to wrap them (usually via this package).
// - In spite of the usual immutability principle for errors, the code to create actual errors first
//   creates some new error and then modifies its parameters before returning the new error. This is
//   done purely to unify the code.

// NewErrorWithData_struct creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil.
// Parameters are added for each visible field of StructType.
//
// For baseError == nil, interpolationString == "", this function panics.
// For baseError == nil, interpolationString != "", creates a new error that does not wrap an error.
// The function also panics if StructType is unsuited (e.g. contains unexported fields)
func NewErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType) ErrorWithData[StructType] {
	if baseErrorUnboxed, ok := baseError.(incomparabilityUndoer_any); ok {
		return NewErrorWithData_struct[StructType](baseErrorUnboxed.AsComparable_any(), interpolationString, data)
	}
	reflectedStructType := utils.TypeOfType[StructType]()
	_ = getStructMapConversionLookup(reflectedStructType) // trigger early panic for invalid StructType
	if baseError == nil && interpolationString == "" {
		panic(ErrorPrefix + "called NewErrorWithData_struct with nil base error and empty interpolation string")
	}
	if interpolationString == "" {
		// Note: baseError != nil
		if _, baseSupportParams := baseError.(ErrorInterpolater); baseSupportParams {
			interpolationString = "$w"
		} else {
			interpolationString = "%w"
		}
	}
	// Note: baseError may be nil. This is actually fine.

	createdError := makeErrorWithParametersCommon(baseError, interpolationString)

	// TODO: Validation

	fillMapFromStruct(data, &createdError.params)
	return &errorWithParameters_T[StructType]{errorWithParameters_common: createdError}
}

// NewErrorWithData_params creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil
// Parameters are supposed to be passed as string - value pairs, e.g:
//
//	NewErrorWIthData_params[StrucType](nil, "Some error with $v{Param1} and $s{Param2}", "Param1", 5, "Param2", `some_string`)
//
// We only default to %w or $w (which refers to the base's error message) if there is a baseError to start with.
//
//   - For baseError == nil, interpolationString == "", this function panics.
//   - For baseError == nil, interpolationString != "", creates a new error that does not wrap an error.
//
// The function also panics if StructType is unsuited (e.g. contains unexported fields), params is malformed or the set of all params does not allow construct an instance of StructType.
func NewErrorWithData_params[StructType any](baseError error, interpolationString string, params ...any) ErrorWithData[StructType] {
	if baseErrorUnboxed, ok := baseError.(incomparabilityUndoer_any); ok {
		return NewErrorWithData_params[StructType](baseErrorUnboxed.AsComparable_any(), interpolationString, params...)
	}
	// make some validity checks to give meaningful error messages.
	// Impressive: go - staticcheck actually recognizes this pattern and has my IDE complain at the call site about violations (calling with odd number of args)!
	if len(params)%2 != 0 {
		panic(ErrorPrefix + "called NewErrorWithData_params(err, interpolationString, args...) with an odd number of args. These are supposed to be name-value pairs")
	}
	extraParams := len(params) / 2
	if baseError == nil && interpolationString == "" {
		panic(ErrorPrefix + "called NewErrorWithData_params with nil base error and empty interpolation string")
	}
	if interpolationString == "" {
		if _, baseSupportParams := baseError.(ErrorInterpolater); baseSupportParams {
			interpolationString = "$w"
		} else {
			interpolationString = "%w"
		}
	}

	// create new error, copying all parameters from baseError
	ret := makeErrorWithParametersCommon(baseError, interpolationString)

	// add new parameters to it
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"called NewErrorWithData_params(err, interpolationString, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		ret.params[s] = params[2*i+1]
	}

	// Check whether the promise of being able to construct an instance of StructType is satisfied.
	cannotMakeStructError := canMakeStructFromParametersInError[StructType](&ret)
	if cannotMakeStructError != nil {
		panic(cannotMakeStructError)
	}

	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// NewErrorWithData_map has the same meaning as [NewErrorWithData_params], but the parameters are passed as a map rather than (string, any) - pairs.
func NewErrorWithData_map[StructType any](baseError error, interpolationString string, params map[string]any) ErrorWithData[StructType] {
	if baseErrorUnboxed, ok := baseError.(incomparabilityUndoer_any); ok {
		return NewErrorWithData_map[StructType](baseErrorUnboxed.AsComparable_any(), interpolationString, params)
	}

	if baseError == nil && interpolationString == "" {
		panic(ErrorPrefix + "called NewErrorWithData_map with nil base error and empty interpolation string")
	}
	if interpolationString == "" {
		if _, baseSupportParams := baseError.(ErrorInterpolater); baseSupportParams {
			interpolationString = "$w"
		} else {
			interpolationString = "%w"
		}
	}

	ret := makeErrorWithParametersCommon(baseError, interpolationString)

	for key, value := range params {
		ret.params[key] = value
	}
	cannotMakeStructError := canMakeStructFromParametersInError[StructType](&ret)
	if cannotMakeStructError != nil {
		panic(cannotMakeStructError)
	}
	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// NewErrorWithData_any_params is identical to [NewErrorWithData_params] except for the guarantee about containing data.
func NewErrorWithData_any_params(baseError error, interpolationString string, parameters ...any) ErrorWithData_any {
	return NewErrorWithData_params[struct{}](baseError, interpolationString, parameters...)
}

// NewErrorWithData_any_map is identical to [NewErrorWithData_map] except for the guarantee about containing data.
func NewErrorWithData_any_map(baseError error, interpolationString string, parameters map[string]any) ErrorWithData_any {
	return NewErrorWithData_map[struct{}](baseError, interpolationString, parameters)
}

// AddDataToError_params creates a new error wrapping baseError with additional parameters set.
// This is identical to [NewErrorWithData_params], except that it always used the default interpolationString
// and for the err==nil case:
// If err == nil, this function returns nil
func AddDataToError_params[StructType any](baseError error, parameters ...any) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, "", parameters...)
}

// AddDataToError_map is identical to [AddDataToError_params], except it
// takes parameters as a map[string]any rather than variadic (string, any) - pairs.
//
// Like [AddDataToError_params], it returns nil if the provided err==nil.
func AddDataToError_map[StructType any](err error, parameters map[string]any) ErrorWithData[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithData_map[StructType](err, "", parameters)
}

// AddDataToError_any_params is identical to [AddDataToError_params] except for the guarantee about containing data.
func AddDataToError_any_params(baseError error, parameters ...any) ErrorWithData_any {
	return AddDataToError_params[struct{}](baseError, parameters...)
}

// AddDataToError_any_map is identical to [AddDataToError_map] except for the guaranteed about containing data.
func AddDataToError_any_map(baseError error, parameters map[string]any) ErrorWithData_any {
	return AddDataToError_map[struct{}](baseError, parameters)
}

// AddDataToError_struct returns a new error based on baseError with the data struct merged to the parameters.
// This is identical to NewErrorWithData_struct except for the baseError == nil case:
//
// On nil input for baseError, returns nil, ignoring the provided data.
func AddDataToError_struct[StructType any](baseError error, data *StructType) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_struct(baseError, "", data)
}

// DeleteParameterFromError takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Has no effect (except for copying and wrapping) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If the input error is nil, returns nil
func DeleteParameterFromError(err error, parameterName string) unconstrainedErrorWithGuaranteedParameters {
	if err == nil {
		return nil
	}
	if baseErrorUnboxed, ok := err.(incomparabilityUndoer_any); ok {
		return DeleteParameterFromError(baseErrorUnboxed.AsComparable_any(), parameterName)
	}
	var ret errorWithParameters_common
	if errInterpolatable, baseSupportsParams := err.(ErrorInterpolater); baseSupportsParams {
		ret = makeErrorWithParametersCommon(errInterpolatable, "$w")
	} else {
		ret = makeErrorWithParametersCommon(err, "%w")
	}

	delete(ret.params, parameterName)
	return &errorWithParameters_T[struct{}]{errorWithParameters_common: ret}
}

// TODO: Rename? The purpose of this function is to change the displayed string.

// WrapAsErrorWithData returns a new ErrorWithData[StructType] based on baseError with error message given by the interpolation string.
// If baseError == nil, this function return nil
//
// WrapAsErrorWithData is deprecated
func WrapAsErrorWithData[StructType any](baseError ErrorWithData[StructType], interpolationString string) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, interpolationString)
}

// AsErrorWithData[StructType](err) returns a copy of the error with a data type that guarantees that a struct of type StructType is contained in the data.
// This is intended to "downcast" StructTypes to a sub-struct. Returns nil on nil input.
//
// NOTE: We make no guarantees about whether the returned error wraps baseError; if it does not, it wraps whatever baseError wrapped.
func AsErrorWithData[StructType any](baseError error) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	if _, baseSupportsParams := baseError.(ErrorInterpolater); baseSupportsParams {
		return NewErrorWithData_params[StructType](baseError, "$w")
	} else {
		return NewErrorWithData_params[StructType](baseError, "%w")
	}

}
