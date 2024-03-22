package errorsWithData

import (
	"errors"
	"fmt"
)

// This file contains functions used to create / modify errors.
// Due to immutability of errors, modifications really means return modified copy.
//
// On the implementations side, there is a subtlety to note here:
// - We unbox any input base error via incomparabilityUndoer_any, if applicable.
//   This is because we do not want incomparable errors to appear in actual error chains,
//   because this could trigger bugs in 3rd party code, which might silently make assumptions here.
//   The sole purpose of incomparable errors is to export them for users to compare against with
//   errors.Is, documenting potential errors and to wrap them (usually via this package).

// noValidation is a constant of type config_Validation with meaning "No validation requested"
var noValidation = config_Validation{doValidation: validationRequest_NoValidation}

// validateError validates inputError according to config (which specifies which type of validation is requested).
// The result is returned as err, which err==nil meaning that validation succeeded.
//
// Note: This is a purely internal function used to unify code. We happen to only need it for inputError of type ErrorWithData_any.
func validateError(inputError ErrorWithData_any, config config_Validation) (err error) {
	switch config.WhatValidationIsRequested() {
	case validationRequest_NoValidation:
		// do nothing
	case validationRequest_Syntax:
		err = inputError.ValidateSyntax()
	case validationRequest_Base:
		err = inputError.ValidateError_Base()
	case validationRequest_Final:
		err = inputError.ValidateError_Final()
	default:
		panic("Cannot happen")
	}
	return
}

// NOTE: Make sure that the set of documented accepted flags and the specific type in each function here precisely matches the list of flags_test.go

// NewErrorWithData_struct creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where by default an empty string is
// interpreted as a default interpolation string ("$w" or "%w"), which asserts that baseError is non-nil.
// Parameters are added for each visible field of StructType.
// flags are optional and can be used to change the default behaviour.
//
// We support the following flags:
// - [PreferPreviousData], [ReplacePreviousData] (default), [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]: Controls how to handle data already present in baseError with the same key.
// - [RecoverFromComparisonFunctionPanic] (default), [LetComparisonFunctionPanic]: Only meaningful if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set. Controls how panics during comparisons are handled.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
// - [NoValidation], [ErrorUnlessValidSyntax] (default), [ErrorUnlessValidBase], [ErrorUnlessValidFinal]: Controls validation of created errors
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".
//
// Note that if baseError == nil, the newly created error does not wrap an error.
// If [DefaultToWrapping] is set and baseError == nil and interpolationString == "", this function panics.
//
// If StructType does not satisfy [StructSuitableForErrorsWithData], this function panics.
func NewErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, flags ...flagArgument_NewErrorStruct) (ret ErrorWithData[StructType], err error) {

	// parse received flags into config
	var config errorCreationConfig // zero value is the correct default value for the config entries
	parseFlagArgs(&config, flags...)

	// Needs to be caught here, because newErrorWithData_struct is not supposed to handle this.
	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_struct with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	// unbox base error if possible.
	baseError = UnboxError(baseError)

	// trigger early panic for invalid StructType. This would panic down the line anyway, but doing so here simplifies debugging.
	if unsuitableStructError := StructSuitableForErrorsWithData[StructType](); unsuitableStructError != nil {
		panic(unsuitableStructError)
	}

	// actually create the error with function specific to the implementation of the ErrorWithData[StructType] interface.
	// Note that we get a valid ret even on failure.
	ret, errsNewError := newErrorWithData_struct[StructType](baseError, interpolationString, data, config.config_OldData, config.config_EmptyString)

	// perform syntax validation, if requested. Note that we do this even the step above "failed" with errsNewError != nil.
	// The reason is that this detects a largely disjoint set of failure cases and we report all of them.
	errValidate := validateError(ret, config.config_Validation)

	// construct error message to be displayed in case something went wrong above.
	if errsNewError != nil {
		if errValidate != nil {

			// NOTE: Go 1.20+ (or 1.21+) supports multiple %w's in fmt.Errorf
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_struct failed to create error for the following %v reasons: %w."+
				"\nAdditionally, validation or the result failed with the following error: %w", len(errsNewError), errors.Join(errsNewError...), errValidate)
		} else {
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_struct failed to create error for the following %v reasons: %w",
				len(errsNewError), errors.Join(errsNewError...))
		}
	} else {
		// We only got a validation error. We just forward it without wrappign and taking note that it originated from NewErrorWithData_struct
		// The reason is that validation errors are tied to ret, not to the particular way we constructed it.
		err = errValidate // may be nil
	}

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}

	return
}

// NewErrorWithData_params creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where by default an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil
// Parameters are supposed to be passed as string - value pairs. Flags can be mixed into those pairs, e.g:
//
//	NewErrorWIthData_params[StrucType](nil, "Some error with $v{Param1} and $s{Param2}", "Param1", 5, "Param2", `some_string`, PanicOnAllErrors)
//
// If parameter names are repeated, the last value takes precendence.
//
// We support the following flags:
// - [PreferPreviousData], [ReplacePreviousData] (default), [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]: Controls how to handle data already present in baseError with the same key.
// - [RecoverFromComparisonFunctionPanic] (default), [LetComparisonFunctionPanic]: Only meaningful if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set. Controls how panics during comparisons are handled.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
// - [NoValidation], [ErrorUnlessValidSyntax] (default), [ErrorUnlessValidBase], [ErrorUnlessValidFinal]: Controls validation of created errors
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".
// - [MissingDataAsZero], [MissingDataIsError] (default): Controls whether data required for StructType that is missing is silently zero-initialized
//
// The function panics under any of the following conditions:
//   - StructType is unsuited, i.e. does not satisfy [StructSuitableForErrorsWithData]
//   - paramsAndFlags is malformed
//   - interpolationsString == "", baseError == nil, [AllowEmptyString] is not set
//   - [PanicOnAllErrors] was set and there is an error
//   - [LetComparisonFunctionPanic] was set and there was a panic in a comparison function (e.g. by comparing values of equal incomparable type)
//
// Note that even on error, ret will be a valid ErrorWithData[StructType].
// For each field of StructType where the provided/inherited parameter is missing or has the wrong type, we add or replace it by a zero value of appropriate type.
// This implies that [MissingDataAsZero] or [MissingDataIsError] only affect whether adding zeros happens silently or triggers an error.
// These zero values are actually added when creating the error, not when retrieving data. In particular, [HasParameter] will see those zero entries
// and they are inherited if ret is subsequently used as baseError.
//
// A nil interface value among the params is converted to a nil of appropriate type if the corresponding field can be nil. So this case is not treated as "has the wrong type" above if the field can be nil.
// However, this conversion happens when *retrieving* data via the struct API. The parameter stored in the error remains any(nil) and is not converted.
func NewErrorWithData_params[StructType any](baseError error, interpolationString string, paramsAndFlags ...any) (ret ErrorWithData[StructType], err error) {
	baseError = UnboxError(baseError)

	// We parse the config and parameters first

	L := len(paramsAndFlags)
	params_map := make(ParamMap, L/2)                     // L/2 is given as lower bound for initial capacity .
	flagArgs := make([]flagArgument_NewErrorParams, 0, L) // L given as capacity to avoid reallocation

	// parse paramsAndFlags into flagArgs and params_map
	for i := 0; i < L; i++ { // i modified in loop body
		switch arg := paramsAndFlags[i].(type) {
		case string:
			if i == L-1 {
				panic(fmt.Errorf(ErrorPrefix+"invalid arguments to NewErrorWithData_params: trailing parameter %v has no value", arg))
			}
			i++
			params_map[arg] = paramsAndFlags[i]
		case flagArgument_NewErrorParams: // must come before flagArgument below
			flagArgs = append(flagArgs, arg)
		case flagArgument: // -- general case of a flag. This catches flags not satisfying flagArgument_NewErrorParams for better error message.
			panic(fmt.Errorf(ErrorPrefix + "NewErrorWithData_params called with a flag that is not supported by this function"))
		default:
			panic(fmt.Errorf(ErrorPrefix + "NewErrorWithData_params called with invalid parameters. Parameters must come as string-any pairs and valid flags"))
		}
	}
	// parse the collected flags into config
	var config errorCreationConfig // The zero value is the correct default value for the config entries
	parseFlagArgs(&config, flagArgs...)

	// We need to settle this case here rather than in newErrorWithData_map
	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_params with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	// trigger early panic for invalid StructType
	if errInvalidStruct := StructSuitableForErrorsWithData[StructType](); errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	ret, errCreateError := newErrorWithData_map[StructType](baseError, interpolationString, params_map, config.config_OldData, config.config_ImplicitZero, config.config_EmptyString)
	errValidation := validateError(ret, config.config_Validation)

	// merge the two errors
	if errCreateError != nil {
		if errValidation != nil {
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_params failed with the following error: %w.\nAdditionally, validation of the error failed with the following error: %w", errCreateError, errValidation)
		} else {
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_params failed with the following error: %w", errCreateError)
		}
	} else {
		err = errValidation // possibly nil
	}

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}

	return
}

// NewErrorWithData_map has the same meaning as [NewErrorWithData_params], but the parameters are passed as a map rather than (string, any) - pairs.
func NewErrorWithData_map[StructType any](baseError error, interpolationString string, newParams ParamMap, flags ...flagArgument_NewErrorParams) (ret ErrorWithData[StructType], err error) {
	baseError = UnboxError(baseError)

	// zero value is the correct default value for the config
	var config errorCreationConfig
	parseFlagArgs(&config, flags...)

	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_map with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	// trigger early panic for invalid StructType
	if errInvalidStruct := StructSuitableForErrorsWithData[StructType](); errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	ret, errCreateError := newErrorWithData_map[StructType](baseError, interpolationString, newParams, config.config_OldData, config.config_ImplicitZero, config.config_EmptyString)
	errValidation := validateError(ret, config.config_Validation)

	// merge the two errors
	if errCreateError != nil {
		if errValidation != nil {
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_map failed with the following error: %w.\nAdditionally, validation of the error failed with the following error: %w", errCreateError, errValidation)
		} else {
			err = fmt.Errorf(ErrorPrefix+"NewErrorWithData_map failed with the following error: %w", errCreateError)
		}
	} else {
		err = errValidation // possibly nil
	}

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}

	return
}

// DeleteParameterFromError_any takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Has no effect (except for copying, possibly unboxing [MakeIncomparable], and wrapping) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// interpolationString is used to change the error message. If interpolationString == "" and [DefaultToWrapping], we refer to inputError.
//
// if the input error is nil, interpolationString == nil and [DefaultToWrapping] is set, returns (nil,nil)
// if the input error is nil, interpolationString == nil and [AllowEmptyString] is set, this function panics
//
// This function accepts the following optional flags:
//
// - [ReturnError] (default), [PanicOnAllErrors],
// - [NoValidation] (default), [ErrorUnlessValidSyntax], [ErrorUnlessValidBase], [ErrorUnlessValidFinal]
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".]
//
// Note that validation actually follows the error chain, so the validation flags are meaningful. The returned err can only be non-nil if a validation flag is set.
func DeleteParameterFromError_any(inputError error, interpolationString string, parameterName string, flags ...flagArgument_DeleteAny) (ret ErrorWithData_any, err error) {
	inputError = UnboxError(inputError)

	var config = errorCreationConfig{config_Validation: noValidation}
	parseFlagArgs(&config, flags...)

	if inputError == nil && interpolationString == "" {
		if config.AllowEmptyString() {
			panic(fmt.Errorf("DeleteParameterFromError_any called with explict empty interpolation string and nil inputError"))
		} else {
			return nil, nil
		}
	}

	ret = deleteParameterFromError_any(inputError, interpolationString, parameterName, config.config_EmptyString)
	err = validateError(ret, config.config_Validation)

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}

	return
}

// DeleteParameterFromError takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Deletion has no effect (except for copying, wrapping and ensuring it safisfies ErrorWithData[StructType]) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// interpolationString is used to change the error message. If interpolationString == "" and [DefaultToWrapping], we refer to inputError.
// if the input error is nil, interpolationString == nil and [DefaultToWrapping] is set, returns (nil,nil)
// if the input error is nil, interpolationString == nil and [AllowEmptyString] is set, this function panics
//
// This function accepts the following optional flags:
//
// - [MissingDataAsZero], [MissingDataIsError] (default),
// - [ReturnError] (default), [PanicOnAllErrors],
// - [NoValidation] (default), [ErrorUnlessValidSyntax], [ErrorUnlessValidBase], [ErrorUnlessValidFinal]
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".]
//
// As opposed to [DeleteParameterFromError_any], this function returns an ErrorWithData[StructType] for some StructType.
// If StructType does not satisfy the conditions explained in [StructSuitableForErrorsWithData], this function panics, irrespective of the [ReturnError] or [PanicOnAllErrors] flag.
//
// Error handling depends on the flags passed. The same considerations as laid out in [NewErrorWithData_params] apply.
// Note that validation actually follows the error chain, so the validation flags are meaningful.
func DeleteParameterFromError[StructType any](inputError error, interpolationString, parameterName string, flags ...flagArgument_Delete) (ret ErrorWithData[StructType], err error) {

	// trigger early panic for invalid StructType. This happens even for nil inputError.
	if errInvalidStruct := StructSuitableForErrorsWithData[StructType](); errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	inputError = UnboxError(inputError)

	var config = errorCreationConfig{config_Validation: noValidation}
	parseFlagArgs(&config, flags...)

	if inputError == nil && interpolationString == "" {
		if config.AllowEmptyString() {
			panic(fmt.Errorf("DeleteParameterFromError called with explict empty interpolation string and nil inputError"))
		} else {
			return nil, nil
		}
	}

	ret, errCreateError := deleteParameterFromError[StructType](inputError, interpolationString, parameterName, config.config_ImplicitZero, config.config_EmptyString)
	errValidation := validateError(ret, config.config_Validation)

	// merge the two errors
	if errCreateError != nil {
		if errValidation != nil {
			err = fmt.Errorf(ErrorPrefix+"DeleteParameterFromError failed with the following error: %w.\nAdditionally, validation of the error failed with the following error: %w", errCreateError, errValidation)
		} else {
			err = fmt.Errorf(ErrorPrefix+"DeleteParameterFromError failed with the following error: %w", errCreateError)
		}
	} else {
		// Note we do not wrap here, because the information that it was DeleteParameterFromError that failed is not useful.
		// Validation errors are tied to ret rather the way we constructed it.
		err = errValidation // possibly nil.
	}

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}
	return
}

// AsErrorWithData[StructType](inputError) returns a copy of inputError with a data type that guarantees that a struct of type StructType is contained in the data.
// This is intended to "cast" ErrorWithData_any to ErrorWithData[StructType] or to change StructType.
//
// This function unconditionally panics for StructType not satisfying [StructSuitableForErrorsWithData].
//
// If inputError == nil, this function returns (nil, nil).
//
// This function accepts the following optional flags:
//
// - [MissingDataAsZero], [MissingDataIsError] (default),
// - [ReturnError] (default), [PanicOnAllErrors]
//
// If inputError does not contain entries of appropriate type for each field of StructType, this function add zero-initialized fields to the parameters of convertedError.
// For entries that are merely missing this is considered an error depending on whether [MissingDataIsError] or [MissingDataIsZero] is set.
// In any case, if [ReturnError] is set (the default), these type-mismatch and missing-param errors are reported in err.
// If [PanicOnAllErrors] is set, we panic(err) instead.
func AsErrorWithData[StructType any](inputError error, flags ...flagArgument_AsErrorWithData) (ret ErrorWithData[StructType], err error) {

	// trigger early panic for invalid StructType. This happens even for nil inputError.
	if errInvalidStruct := StructSuitableForErrorsWithData[StructType](); errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	var config errorCreationConfig // zero value is correct default
	parseFlagArgs(&config, flags...)

	if inputError == nil {
		return nil, nil
	}

	inputError = UnboxError(inputError)

	ret, err = asErrorWithData[StructType](inputError, config.config_ImplicitZero)
	if config.PanicOnAllErrors() && err != nil {
		panic(err)
	}

	// NOTE: No validateError here; this would make little sense.
	return

}

// TEMPORARILY COMMENTED OUT

/*

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

*/
