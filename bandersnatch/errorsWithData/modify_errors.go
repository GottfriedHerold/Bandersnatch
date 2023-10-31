package errorsWithData

import (
	"fmt"
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

func validateError(inputError ErrorWithData_any, config validationParams) (err error) {
	switch config.WhatValidationIsRequested() {
	case flagArg_NoValidation:
		// do nothing
	case flagArg_ValidateSyntax:
		err = inputError.ValidateSyntax()
	case flagArg_ValidateBase:
		err = inputError.ValidateError_Base()
	case flagArg_ValidateFinal:
		err = inputError.ValidateError_Final()
	default:
		panic("Cannot happen")
	}
	return
}

// NewErrorWithData_struct creates a new [ErrorWithData] wrapping the given baseError if non-nil.
// interpolationString is used to create the new error message, where by default an empty string is
// interpreted as a default interpolation string ("$w" or "%w") if baseError is non-nil.
// Parameters are added for each visible field of StructType.
// flags are optional and can be used to change the default behaviour.
//
// We support the following flags:
// - [PreferPreviousData], [ReplacePreviousData] (default), [AssertDataIsNotReplaced]: Controls how to handle data already present in baseError with the same key.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
// - [NoValidation], [ErrorUnlessValidSyntax] (default), [ErrorUnlessValidBase], [ErrorUnlessValidFinal]: Controls validation of created errors
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".
//
// Note that if baseError == nil, the newly created error does not wrap an error.
// If [DefaultToWrapping] is set and baseError == nil and interpolationString == "", this function panics.
//
// If StructType does not satisfy [StructSuitableForErrorsWithData], this function panics.
func NewErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, flags ...flagArgument_NewErrorStruct) (ret ErrorWithData[StructType], err error) {

	var config = errorCreationParams{validationParams: defaultValidation} // zero value is the correct default value for the other config entries

	parseFlagArgs(&config, flags...)

	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_struct with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	// unbox base error if possible.
	baseError = UnboxError(baseError)

	err = StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType
	if err != nil {
		// TODO: Other handling? Print interpolation string and data?
		panic(err)
	}

	ret, err = newErrorWithData_struct[StructType](baseError, interpolationString, data, config.mergeParams, config.handleEmptyInterpolationString)

	if err != nil {
		validateError(ret, config.validationParams)
	} else {
		err = validateError(ret, config.validationParams)
	}

	if err != nil && config.PanicOnError() {
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
// - [PreferPreviousData], [ReplacePreviousData] (default), [AssertDataIsNotReplaced]: Controls how to handle data already present in baseError with the same key.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
// - [NoValidation], [ErrorUnlessValidSyntax] (default), [ErrorUnlessValidBase], [ErrorUnlessValidFinal]: Controls validation of created errors
// - [AllowEmptyString], [DefaultToWrappring] (default): Controls whether an empty interpolation string is interpreted as "$w" resp. "%w".
// - [MissingDataAsZero], [MissingDataIsError] (default): Controls whether data required for StructType that is missing is zero-initialized
//
// The function panics under any of the following conditions:
//   - StructType is unsuited, i.e. does not satisfy [StructSuitableForErrorsWithData]
//   - paramsAndFlags is malformed
//   - interpolationsString == "", baseError == nil, [AllowEmptyString] is not set
//   - [PanicOnAllErrors] was set and there is an error
//
// Note that even on error, ret will be a valid ErrorWithData[StructType].
// For each field of StructType where the provided/inherited parameter is missing or has the wrong type, we add or replace it by a zero value of appropriate type.
// This implies that [MissingDataAsZero] or [MissingDataIsError] only affect whether adding zeros happens silently or triggers an error.
// These zero values are actually added when creating the error, not when retrieving data. In particular, [HasParameter] will see those entries
// and they are inherited if ret is subsequently used as baseError.
//
// A nil interface value among the params is converted to a nil of appropriate type if the corresponding field can be nil. So this case is not treated as "has the wrong type" above if the field can be nil.
// However, this conversion happens when *retrieving* data via the struct API. The parameter contained in the error is remains any(nil) and is not converted.
func NewErrorWithData_params[StructType any](baseError error, interpolationString string, paramsAndFlags ...any) (ret ErrorWithData[StructType], err error) {
	baseError = UnboxError(baseError)

	// zero value is the correct default value for the other config entries
	var config = errorCreationParams{validationParams: defaultValidation, zeroFillParams: zeroFillParams{missingDataIsError: true, addMissingData: true}}

	L := len(paramsAndFlags)
	params_map := make(ParamMap, L/2)
	flagArgs := make([]flagArgument_NewErrorParams, 0, L)

	// parse paramsAndFlags into flagArgs and params_map
	for i := 0; i < L; i++ {
		switch arg := paramsAndFlags[i].(type) {
		case string:
			if i == L-1 {
				panic(fmt.Errorf(ErrorPrefix+"invalid arguments to NewErrorWithData_params: trailing parameter %v has no value", arg))
			}
			i++
			params_map[arg] = paramsAndFlags[i]
		case flagArgument_NewErrorParams:
			flagArgs = append(flagArgs, arg)
		case flagArgument: // -- general case of a flag. This catches unsupported flags
			panic(fmt.Errorf(ErrorPrefix + "NewErrorWithData_params called with a flag that is not supported by this function"))
		default:
			panic(fmt.Errorf(ErrorPrefix + "NewErrorWithData_params called with invalid parameters. Parameters must come as string-any pairs and valid flags"))
		}
	}
	parseFlagArgs(&config, flagArgs...)

	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_params with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	errInvalidStruct := StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType
	if errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	ret, err = newErrorWithData_map[StructType](baseError, interpolationString, params_map, config.mergeParams, config.zeroFillParams, config.handleEmptyInterpolationString)

	if err != nil {
		validateError(ret, config.validationParams)
	} else {
		err = validateError(ret, config.validationParams)
	}

	if err != nil && config.PanicOnError() {
		panic(err)
	}

	return
}

// NewErrorWithData_map has the same meaning as [NewErrorWithData_params], but the parameters are passed as a map rather than (string, any) - pairs.
func NewErrorWithData_map[StructType any](baseError error, interpolationString string, newParams ParamMap, flags ...flagArgument_NewErrorParams) (ret ErrorWithData[StructType], err error) {
	baseError = UnboxError(baseError)

	// zero value is the correct default value for the other config entries
	var config = errorCreationParams{validationParams: defaultValidation, zeroFillParams: zeroFillParams{missingDataIsError: true, addMissingData: true}}
	parseFlagArgs(&config, flags...)

	if baseError == nil && interpolationString == "" && !config.AllowEmptyString() {
		panic(ErrorPrefix + "called NewErrorWithData_map with nil base error and empty interpolation string without [AllowEmptyString] flag")
	}

	errInvalidStruct := StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType
	if errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	ret, err = newErrorWithData_map[StructType](baseError, interpolationString, newParams, config.mergeParams, config.zeroFillParams, config.handleEmptyInterpolationString)

	if err != nil {
		validateError(ret, config.validationParams)
	} else {
		err = validateError(ret, config.validationParams)
	}

	if err != nil && config.PanicOnError() {
		panic(err)
	}

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

// DeleteParameterFromError_any takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Has no effect (except for copying, possibly unboxing [MakeIncomparable], and wrapping) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If the input error is nil, returns nil
func DeleteParameterFromError_any(inputError error, parameterName string, flags ...flagArgument_Delete) (ret ErrorWithData_any, err error) {
	if inputError == nil {
		return nil, nil
	}
	inputError = UnboxError(inputError)

	var config = errorCreationParams{validationParams: defaultValidation}
	parseFlagArgs(&config, flags...)

	ret = deleteParameterFromError_any(inputError, "", parameterName, handleEmptyInterpolationString{allowEmpty: false})
	err = validateError(ret, config.validationParams)

	if err != nil && config.PanicOnError() {
		panic(err)
	}

	return
}

// DeleteParameterFromError takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Deletion has no effect (except for copying, wrapping and ensuring it safisfies ErrorWithData[StructType]) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If err == nil, this function returns nil.
//
// This function accepts the following optional flags:
//
// - [MissingDataAsZero], [MissingDataIsError] (default),
// - [ReturnError] (default), [PanicOnAllErrors],
// - [NoValidation] (default), [ErrorUnlessValidSyntax], [ErrorUnlessValidBase], [ErrorUnlessValidFinal]
//
// As opposed to [DeleteParameterFromError_any], this function returns an ErrorWithData[StructType] for some StructType.
// If StructType does not satisfy the conditions explained in [StructSuitableForErrorsWithData], this function panics, irrespective of the [ReturnError] or [PanicOnAllErrors] flag.
//
// Error handling depends on the flags passed. The same considerations as laid out in [NewErrorWithData_params] apply.
// Note that validation actually follows the error chain, so the validation flags are meaningful.
func DeleteParameterFromError[StructType any](inputError error, parameterName string, flags ...flagArgument_Delete) (ret ErrorWithData[StructType], err error) {

	errInvalidStruct := StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType. This happens even for nil inputError.
	if errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	if inputError == nil {
		return nil, nil
	}
	inputError = UnboxError(inputError)

	var config = errorCreationParams{validationParams: defaultValidation, zeroFillParams: zeroFillParams{missingDataIsError: true, addMissingData: true}}
	parseFlagArgs(&config, flags...)

	// note that the last argument actually equals config.handleEmptyInterpolationString
	ret, err = deleteParameterFromError[StructType](inputError, "", parameterName, config.zeroFillParams, handleEmptyInterpolationString{allowEmpty: false})

	if err != nil {
		validateError(ret, config.validationParams)
	} else {
		err = validateError(ret, config.validationParams)
	}

	if err != nil && config.PanicOnError() {
		panic(err)
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

// AsErrorWithData[StructType](inputError) returns a copy of inputError with a data type that guarantees that a struct of type StructType is contained in the data.
// This is intended to "cast" ErrorWithData_any to ErrorWithData[StructType] or to change StructType. Returns (nil, nil) on nil input.
//
// This function panics for StructType not satisfying [StructSuitableForErrorsWithData].
// TODO: DOC. Note that we make little guarantees regarding wrapping.
func AsErrorWithData[StructType any](inputError error, flags ...flagArgument_AsErrorWithData) (convertedError ErrorWithData[StructType], err error) {

	errInvalidStruct := StructSuitableForErrorsWithData[StructType]() // trigger early panic for invalid StructType. This happens even for nil inputError.
	if errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	if inputError == nil {
		return nil, nil
	}

	inputError = UnboxError(inputError)

	var config = errorCreationParams{validationParams: defaultValidation, zeroFillParams: zeroFillParams{missingDataIsError: true, addMissingData: true}}
	parseFlagArgs(&config, flags...)

	// Consider refactoring makeErrorWithParametersCommon into newErrorWithParametersCommon to avoid this
	convertedErrorVal, err := makeErrorWithParameterCommon[StructType](inputError, "", config.handleEmptyInterpolationString, config.zeroFillParams)
	return &convertedErrorVal, err

	// NOTE: No validateError here; this would make little sense.
}
