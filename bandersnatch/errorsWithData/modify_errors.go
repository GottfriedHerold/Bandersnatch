package errorsWithData

import (
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// NewErrorWithData_struct creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// overrideMessage is used to create the new error message, where an empty string is
// interpreted as a default error message (containing %w and %m).
// Parameters are added for each visible field of StructType.
//
// For baseError == nil, overrideMessage == "", #visibleFields of (*data) > 0, this function panics.
// For baseError == nil, overrideMessage == "", #visibleFields of (*data) ==0, returns nil
// For baseError == nil, overrideMessage != "", creates a new error that does not wrap an error.
func NewErrorWithData_struct[StructType any](baseError error, overrideMessage string, data *StructType) ErrorWithData[StructType] {
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	if baseError == nil {
		if overrideMessage == "" {
			if len(allStructFields) > 0 {
				panic(ErrorPrefix + "called NewErrorWithData(nil,\"\",data) with non-empty data")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}

	createdError := makeErrorWithParametersCommon(baseError, overrideMessage)

	fillMapFromStruct(data, &createdError.params)
	return &errorWithParameters_T[StructType]{errorWithParameters_common: createdError}
}

// NewErrorWithData_params creates a new ErrorWithParameters wrapping the given baseError,
// possibly overriding the error message message and adding parameters.
// If overrideMessage == "", DefaultOverrideMessage is used (except if baseError == nil).
// Note: The only difference between this and IncludeParametersInError is the message and nil handling:
//
// For baseError == nil and overrideMessage == "", #params > 0, we panic
// For baseError == nil, overrideMessage == "", #params == 0, we return a nil interface
func NewErrorWithData_params[StructType any](baseError error, overrideMessage string, params ...any) ErrorWithData[StructType] {
	// make some validity checks to give meaningful error messages.
	// Impressive: go - staticcheck actually recognizes this pattern and has my IDE complain at the call site about violations (calling with odd number of args)!
	if len(params)%2 != 0 {
		panic(ErrorPrefix + "called NewErrorWithParameters(err, overrideMessage, args...) with an odd number of args. These are supposed to be name-value pairs")
	}
	extraParams := len(params) / 2
	if baseError == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic(ErrorPrefix + "called NewErrorWithParameters(nil,\"\",argName, arg1, ...)")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}

	// create a wrapper, copying all parameters from baseError
	ret := makeErrorWithParametersCommon(baseError, overrideMessage)

	// add new parameters to it
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"called NewErrorWithParams(err, overrideMessage, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		ret.params[s] = params[2*i+1]
	}

	// Check whether the promise of being able to construct an instance of StructType is satisfied.
	validationError := canMakeStructFromParametersInError[StructType](&ret)
	if validationError != nil {
		panic(validationError)
	}

	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// NewErrorWithData_map has the same meaning as NewErrorWithGuaranteedParameters, but the parameters are passed as a map rather than string, any - pairs.
func NewErrorWithData_map[StructType any](baseError error, overrideMessage string, params map[string]any) ErrorWithData[StructType] {
	extraParams := len(params) // 0 for nil
	if baseError == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic(ErrorPrefix + "called NewErrorWithParametersMap(nil,\"\",actualParams_map)")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}

	ret := makeErrorWithParametersCommon(baseError, overrideMessage)

	for key, value := range params {
		ret.params[key] = value
	}
	validationError := canMakeStructFromParametersInError[StructType](&ret)
	if validationError != nil {
		panic(validationError)
	}
	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// NewErrorWithData_any_params is identical to NewErrorWithGuaranteedParameters except for the guarantee about containing data.
func NewErrorWithData_any_params(baseError error, overrideMessage string, parameters ...any) ErrorWithData_any {
	return NewErrorWithData_params[struct{}](baseError, overrideMessage, parameters...)
}

// NewErrorWithData_any_map is identical to NewErrorWithGuaranteedParametersFromMap except for the guarantee about containing data.
func NewErrorWithData_any_map(baseError error, overrideMessage string, parameters map[string]any) ErrorWithData_any {
	return NewErrorWithData_map[struct{}](baseError, overrideMessage, parameters)
}

// IncludeGuaranteedParametersInError creates a new error wrapping baseError with additional parameters set.
// This is identical to NewErrorWithGuaranteedParameters, except that it always used the default overrideMessage
// and for the err==nil case:
// If err == nil, returns nil
func IncludeGuaranteedParametersInError[StructType any](baseError error, parameters ...any) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, "", parameters...)
}

// IncludeGuaranteedParametersInErrorFromMap is identical to IncludeGuaranteedParametersInError, except it
// takes parameters as a map[string]any rather than variadic string, any - pairs.
func IncludeGuaranteedParametersInErrorFromMap[StructType any](err error, parameters map[string]any) ErrorWithData[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithData_map[StructType](err, "", parameters)
}

// IncludeParametersInError is identical to IncludeGuaranteedParametersInError except for the guarantee about containing data.
func IncludeParametersInError(baseError error, parameters ...any) ErrorWithData_any {
	return IncludeGuaranteedParametersInError[struct{}](baseError, parameters...)
}

// IncludeParametersInErrorsFromMap is identical to IncludeGuaranteedParametersInErrorFromMap except for the guaranteed about containing data.
func IncludeParametersInErrorsFromMap(baseError error, parameters map[string]any) ErrorWithData_any {
	return IncludeGuaranteedParametersInErrorFromMap[struct{}](baseError, parameters)
}

// IncludeDataInError returns a new error with the data provided.
// This is identical to NewErrorWithParametersFromData except for the baseError == nil case.
//
// On nil input for baseError, returns nil, ignoring the provided data.
func IncludeDataInError[StructType any](baseError error, data *StructType) ErrorWithData[StructType] {
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
	ret := makeErrorWithParametersCommon(err, "")
	delete(ret.params, parameterName)
	return &errorWithParameters_T[struct{}]{errorWithParameters_common: ret}
}

// WrapErrorWithParameters returns a new error based on baseError with error message given by message (which is subject to interpolation).
func WrapErrorWithParameter[StructType any](baseError ErrorWithData[StructType], message string) ErrorWithData[StructType] {
	return NewErrorWithData_params[StructType](baseError, message)
}

// AsErrorWithData[StructType](err) returns a copy of the error with a data type that guarantees that a struct of type StructType is contained in the data.
// This is intended to "downcast" StructTypes to a sub-struct. Returns nil on nil input.
//
// NOTE: We make no guarantees about whether the returned error wraps the input error;
func AsErrorWithData[StructType any](baseError error) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, "%w")
}
