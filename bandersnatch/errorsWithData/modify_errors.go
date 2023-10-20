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

var defaultValidation = validationParams{doValidation: flagArg_ValidateSyntax}

// NewErrorWithData_struct creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil.
// Parameters are added for each visible field of StructType.
// mode is optional and determines how/whether pre-existing parameters with the same key are handled.
// The default is to overwrite old data.
//
// For baseError == nil, interpolationString == "", this function panics.
// For baseError == nil, interpolationString != "", creates a new error that does not wrap an error.
// The function also panics if StructType is unsuited (e.g. contains unexported fields)
func NewErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, flags ...flagArgument_NewErrorStruct) ErrorWithData[StructType] {

	var config = errorCreationParams{validationParams: defaultValidation}

	parseFlagArgs(&config, flags...)

	if baseError == nil && interpolationString == "" {
		panic(ErrorPrefix + "called NewErrorWithData_struct with nil base error and empty interpolation string")
	}

	// unbox base error if possible.
	baseError = UnboxError(baseError)

	err := StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType
	if err != nil {
		// TODO: Other handling? Print interpolation string and data?
		panic(err)
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

	// TODO: Validation?
	return newErrorWithData_struct(baseError, interpolationString, data, &config)
}

// Note: We could pass mode and/or missingData as optional arguments as part of params.
// The issue is that at least my code analysis tool recognizes
// the check to ensure than len(params) is even and giving meaningful errors on violation at the call site.
// This property is much more more user-friendly than not having to write mode explicitly.

// NewErrorWithData_params creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil
// Parameters are supposed to be passed as string - value pairs, e.g:
//
//	NewErrorWIthData_params[StrucType](nil, "Some error with $v{Param1} and $s{Param2}", mode, missingData, "Param1", 5, "Param2", `some_string`)
//
// If parameter names are repeated, the last value takes precendence.
//
// We only default to %w or $w (which refers to the base's error message) if there is a baseError to start with.
//
//   - For baseError == nil, interpolationString == "", this function panics.
//   - For baseError == nil, interpolationString != "", creates a new error that does not wrap an error.
//
// The function also panics if StructType is unsuited (e.g. contains unexported fields) or params is malformed (i.e. does not conist of (string,any) pairs)
//
// mode controls how the function treats parameters present in both baseError and being given.
//
// If the final set of all params does not allow to construct an instance of StructType, the behaviour depends on missingDataTreatment:
//   - If there is a type mismatch between what is required for the struct field and the given parameter, the function panics, except
//     that the nil interface is allowed if the struct field can be nil (possibly of some concrete type)
//   - For missingDataTreatment == [EnsureDataIsPresent], the function panics if there are missing parameters.
//   - For missingDataTreatment == [MissingDataAsZero], we add a zero-initialized parameter for each field of StructType that is missing.
//     Those zero-valued parameter are actually explicitly added when creating the error, not when retrieving the data.
//     It is not possible to distinguish automatically zero-initialized parameters from parameters explicitly set to zero.
//     In particular, [HasParameter] returns true for those parameters and when using an error with such zero-initialized parameters as a base for error wrapping, the wrapping
//     error will see those zero-initialized parameters.
func NewErrorWithData_params[StructType any](baseError error, interpolationString string, mode flagPreviousDataTreatment, missingDataTreatment flagMissingDataTreatment, params ...any) (ret ErrorWithData[StructType]) {
	baseError = UnboxError(baseError)

	// make some validity checks to give meaningful error messages.
	// Impressive: go - staticcheck actually recognizes this pattern and has my IDE complain at the call site about violations (calling with odd number of args)!
	if len(params)%2 != 0 {
		panic(ErrorPrefix + "called NewErrorWithData_params(err, interpolationString, mode, missingData, params...) with an odd number of params. These are supposed to be name-value pairs")
	}
	numberOfExtraParams := len(params) / 2
	if baseError == nil && interpolationString == "" {
		panic(ErrorPrefix + "called NewErrorWithData_params with nil base error and empty interpolation string")
	}
	if interpolationString == "" {
		if _, baseSupportsParams := baseError.(ErrorInterpolater); baseSupportsParams {
			interpolationString = "$w"
		} else {
			interpolationString = "%w"
		}
	}

	// We copy the given variadic params into a map[string]any.
	// This is an unneccessary copy, but simplifies the code by unifying it with the _map variant.
	//
	// NOTE: Due to copying, in the case where a parameter is repeated, the last one takes precedence.
	// This is independent of mode (which the API spec actually requires as stated), which is difficult to achieve otherwise in a simple way.

	params_map := make(ParamMap)

	// add new parameters to it
	for i := 0; i < numberOfExtraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf(ErrorPrefix+"called NewErrorWithData_params(err, interpolationString, mode, missingData, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		params_map[s] = params[2*i+1]
	}

	var err error
	ret, err = newErrorWithData_map[StructType](baseError, interpolationString, mode, missingDataTreatment, params_map)
	if err != nil {
		// This can only happen if the given parameters do not suffice to construct an instance of StructType.
		panic(err)
	}
	return
}

// NewErrorWithData_map has the same meaning as [NewErrorWithData_params], but the parameters are passed as a map rather than (string, any) - pairs.
func NewErrorWithData_map[StructType any](baseError error, interpolationString string, mode flagPreviousDataTreatment, missingDataTreatment flagMissingDataTreatment, newParams ParamMap) (ret ErrorWithData[StructType]) {
	baseError = UnboxError(baseError)

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

	var err error
	ret, err = newErrorWithData_map[StructType](baseError, interpolationString, mode, missingDataTreatment, newParams)

	if err != nil {
		// This can only happen if the given parameters do not suffice to construct an instance of StructType.
		panic(err)
	}
	return
}

// There is no missingDataTreatment argument for the _any - variants, as that makes no sense.

// NewErrorWithData_any_params is identical to [NewErrorWithData_params] except for the guarantee about containing data.
func NewErrorWithData_any_params(baseError error, interpolationString string, mode flagPreviousDataTreatment, parameters ...any) ErrorWithData_any {
	return forgetStructType(NewErrorWithData_params[struct{}](baseError, interpolationString, mode, MissingDataIsError, parameters...))
}

// NewErrorWithData_any_map is identical to [NewErrorWithData_map] except for the guarantee about containing data.
func NewErrorWithData_any_map(baseError error, interpolationString string, mode flagPreviousDataTreatment, parameters map[string]any) ErrorWithData_any {
	return forgetStructType(NewErrorWithData_map[struct{}](baseError, interpolationString, mode, MissingDataIsError, parameters))
}

// AddDataToError_params creates a new error wrapping baseError with additional parameters set.
// This is identical to [NewErrorWithData_params], except that it always uses the default interpolationString
// and for the err==nil case:
// If err == nil, this function returns nil
func AddDataToError_params[StructType any](baseError error, mode flagPreviousDataTreatment, missingDataTreatment flagMissingDataTreatment, parameters ...any) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, "", mode, missingDataTreatment, parameters...)
}

// AddDataToError_map is identical to [AddDataToError_params], except it
// takes parameters as a map[string]any rather than variadic (string, any) - pairs.
//
// Like [AddDataToError_params], it returns nil if the provided err==nil.
func AddDataToError_map[StructType any](err error, mode flagPreviousDataTreatment, missingDataTreatment flagMissingDataTreatment, parameters ParamMap) ErrorWithData[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithData_map[StructType](err, "", mode, missingDataTreatment, parameters)
}

// AddDataToError_any_params is identical to [AddDataToError_params] except for the guarantee about containing data.
func AddDataToError_any_params(baseError error, mode flagPreviousDataTreatment, parameters ...any) ErrorWithData_any {
	return forgetStructType(AddDataToError_params[struct{}](baseError, mode, MissingDataIsError, parameters...))
}

// AddDataToError_any_map is identical to [AddDataToError_map] except for the guaranteed about containing data.
func AddDataToError_any_map(baseError error, mode flagPreviousDataTreatment, parameters map[string]any) ErrorWithData_any {
	return forgetStructType(AddDataToError_map[struct{}](baseError, mode, MissingDataIsError, parameters))
}

// AddDataToError_struct returns a new error based on baseError with the data struct merged to the parameters.
// This is identical to NewErrorWithData_struct except for the baseError == nil case:
//
// On nil input for baseError, returns nil, ignoring the provided data.
func AddDataToError_struct[StructType any](baseError error, data *StructType, mode ...flagPreviousDataTreatment) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_struct(baseError, "", data, mode...)
}

// DeleteParameterFromError_any takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Has no effect (except for copying and wrapping) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If the input error is nil, returns nil
func DeleteParameterFromError_any(err error, parameterName string) ErrorWithData_any {
	if err == nil {
		return nil
	}
	err = UnboxError(err)

	if errInterpolatable, baseSupportsParams := err.(ErrorInterpolater); baseSupportsParams {
		return deleteParameterFromError_any(errInterpolatable, "$w", parameterName)
	} else {
		return deleteParameterFromError_any(err, "%w", parameterName)
	}
}

// DeleteParameterFromError takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Deletion has no effect (except for copying, wrapping and ensuring it safisfies ErrorWithData[StructType]) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If err == nil, this function returns nil.
//
// As opposed to [DeleteParameterFromError_any], this function returns an ErrorWithData[StructType] for some StructType.
// If StructType is unsuited, this function panics. If the resulting error, after deleting the parameter, does not have sufficient parameters
// to construct an instance of StructType, the behaviour depends on missingDataTreatment:
//   - for missingDataTreatment == [MissingDataAsZero], we fill up the data with appropriate zero values (possibly adding back the deleted value)
//   - for missingDataTreatment == [EnsureDataIsPresent], we panic on missing data.
//
// Note that data that is present, but has wrong type for StructType's fields, will always cause a panic.
func DeleteParameterFromError[StructType any](err error, parameterName string, missingDataTreatment flagMissingDataTreatment) (ret ErrorWithData[StructType]) {
	if err == nil {
		return nil
	}
	err = UnboxError(err)

	// indicates failure of deleteParameterFromError.
	// This can happen if parameters in the error are missing (depending on missingDataTreatment) or have to wrong type for
	// ErrorWithData[StructType]
	var bad_params_error error

	if errInterpolatable, baseSupportsParams := err.(ErrorInterpolater); baseSupportsParams {
		ret, bad_params_error = deleteParameterFromError[StructType](errInterpolatable, "$w", parameterName, missingDataTreatment)
	} else {
		ret, bad_params_error = deleteParameterFromError[StructType](err, "%w", parameterName, missingDataTreatment)
	}
	if bad_params_error != nil {
		panic(bad_params_error)
	}
	return
}

/*
// TODO: Rename? The purpose of this function is to change the displayed string.

// WrapAsErrorWithData returns a new ErrorWithData[StructType] based on baseError with error message given by the interpolation string.
// If baseError == nil, this function return nil
//
// WrapAsErrorWithData is deprecated
func WrapAsErrorWithData[StructType any](baseError ErrorWithData[StructType], interpolationString string) ErrorWithData[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithData_params[StructType](baseError, interpolationString, AssertDataIsNotReplaced)
}
*/

// AsErrorWithData[StructType](inputError) returns a copy of inputErrorerror with a data type that guarantees that a struct of type StructType is contained in the data.
// This is intended to "cast" ErrorWithData_any to ErrorWithData[StructType] or to change StructType. Returns (nil, true) on nil input.
//
// This function panics for invalid StructType (e.g. with non-exported fields).
// If conversion fails, because inputError does not contain the needed parameters or parameters have wrong type, ok is set to false. The value of convertedError is unspecified in that case.
//
// If missingDataTreatment is [MissingDataAsZero], we may add zero values to convertedError's parameters to ensure an instance of StructType can be constructed.
// This does not modify inputError.
//
// NOTE: We make no guarantees about whether the returned error wraps inputError; if it does not, it wraps whatever inputError wrapped.
func AsErrorWithData[StructType any](inputError error, missingDataTreatment flagMissingDataTreatment) (convertedError ErrorWithData[StructType], ok bool) {

	if inputError == nil {
		return nil, true
	}

	inputError = UnboxError(inputError)

	var conversionError error

	// Note: We actually add a layer of error wrapping here.
	// This is not really neccessary if inputError is our (only) implementation *errorWithParameters_T or *errorWithParameters_common
	// The issue is that we do not want to tie the functions on modify_errors.go to the particular implementation.
	// Furthermore, note that if missingDataTreatment is MissingDataAsZero, we actually modify the parameters (so making a copy is neccessary anyway)
	// At best, we could make an optimized implementation for this special case.

	if errInterpolatable, baseSupportsParams := inputError.(ErrorInterpolater); baseSupportsParams {
		convertedError, conversionError = newErrorWithData_map[StructType](errInterpolatable, "$w", PreferPreviousData, missingDataTreatment, ParamMap{})
	} else {
		// unlikely to get here. The function does not make too much sense if the inputError does not support ErrorInterpolater.
		convertedError, conversionError = newErrorWithData_map[StructType](inputError, "%w", PreferPreviousData, missingDataTreatment, ParamMap{})
	}

	if conversionError != nil {
		var zero StructType
		// we prefer to return an actual error as the returned converted error,
		// whose error message indicates that conversion failed (and maybe why).
		// This is much nicer to users that don't check the ok value (although it's their own fault).
		// This is the reason why the value of convertedError on convertion failure is unspecified.
		convertedError = NewErrorWithData_struct(nil, fmt.Sprintf(ErrorPrefix+"AsErrorWithData failed to convert error to an ErrorWithData[%v] due the following reason: %v", utils.GetReflectName(utils.TypeOfType[StructType]()), conversionError), &zero)
		ok = false
		return
	}

	ok = true
	return
}
