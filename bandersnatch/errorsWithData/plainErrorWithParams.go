package errorsWithData

import (
	"strings"
)

// This file is part of the errorsWithData package and defines the (currently only) implementation of the [ErrorWithData]
// and [ErrorWithData_any] interface.
// Note that the user-facing functions in modify_errors.go that are used to actually create an [ErrorWithData]
// are neccessarily coupled to the concrete implementation.
// We try to keep most of the specifics to the concrete implementation in this file here, so the functions in modify_errors.go
// are mostly wrappers to some function defined here.

// errorWithParameters_common is a simple implementation of the [ErrorWithData_any] interface
// NOTE:
// exported functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned as typed nil pointers, which is a serious footgun)
//
// [MakeIncomparable] is kind-of an exception to this (it's an ugly hack around limitations of the language and it panics on nil, anyway).
type errorWithParameters_common struct {
	wrapped_error             error    // wrapped underlying error, can be nil
	parsedInterpolationString ast_root // (parsed) error message. Must be non-nil
	params                    ParamMap // map strings -> data. Must be non-nil (use empty map for "no data" rather than nil)
}

// errorWithParameters_T is the extension of errorWithParameters_common that satisfies ErrorWithData[StructType]
//
// It is identical as a struct and only differs in some methods, so we just stuct-embed [errorWithParameters_common]
type errorWithParameters_T[StructType any] struct {
	errorWithParameters_common
}

// forgetStructType "downcasts" ErrorWithData to ErrorWithData_any.
//
// Note: The same effect can be achieved by plain assignment (one is a sub-interface of the other, after all),
// What this function does is actually changing the implementing struct type and removing the extra functionality.
// This is only done for consistency:
// The point is that the actual implementing struct type, or at least the exported extra methods, are in principle visible (e.g. by type-asserting).
// In some cases, we create ErrorWithData_any variables from an errorWithParameters_T[struct{}] rather
// than an errorWithParameters_common. We can use forgetStructType to *consistently* use errorWithParameters_common as the implementing type.
//
// This method is purely an implementation detail and may be removed in the future.
func forgetStructType[StructType any](err ErrorWithData[StructType]) ErrorWithData_any {
	errImpl, ok := err.(*errorWithParameters_T[StructType])
	if !ok {
		// currently unreachable. This just guards against other implementations of ErrorWithData[StructType] and gives a "correct" result.
		// (the whole function could be replaced by the identity anyway)
		return err
	} else {
		return &errImpl.errorWithParameters_common
	}
}

// NOTE: The e == nil checks will never be called if the concrete type of e is errorWithParameters_T due the way struct-embedding in Go works:
// In this case, Go will resolve the struct-embedding by calling (*e).errorWithParameters_common.METHOD(...), which dereferences nil
// before the actual method is called.

// Error_interpolate is provided to satisfy the ErrorWithData_any interface. This is an extension of Error() that
// will interpolate the error string (which may contain formatting instructions like %{Param1}, ${Param2} ).
// As opposed to plain Error(), formatting instructions with $ differs from those with % by taking the parameters from params_passed rather than using the
// parameters stored in the error. The presence of this method is what makes the "$w" formatting verb work.
func (e *errorWithParameters_common) Error_interpolate(params_passed map[string]any) string {
	if e == nil {
		// ought to be unreachable (unless the user tries on purpose with unsafe+reflection) because we export nothing that creates a nil error of this type
		panic(ErrorPrefix + "called Error_interpolate() on nil error of concrete type errorWithParameters_common. This is a bug, since nil errors of this type should never exist.")
	}
	var s strings.Builder
	e.parsedInterpolationString.Interpolate(e.params, params_passed, e.wrapped_error, &s)
	return s.String()
}

// Error is provided to satisfy the error interface
func (e *errorWithParameters_common) Error() string {
	if e == nil {
		panic(ErrorPrefix + "called Error() on nil error of concrete type errorWithParameters_common. This is a bug, since nil errors of this type should never exist.")
	}
	return e.Error_interpolate(nil)
}

// Unwrap is provided to work with errors.Is
func (e *errorWithParameters_common) Unwrap() error {
	// Note: This panics if e is nil (of type *errorWithParams).
	// While returnining untyped nil would give "meaningful" behaviour
	// (including for the recursive calls in HasParameter etc.),
	// we consider any nil pointer of concrete error type a bug.
	return e.wrapped_error
}

// GetData_struct is provided to satisfy ErrorWithParameters[StructType].
//
// It constructs a value of type StructType from the provided parameters.
func (e *errorWithParameters_T[StructType]) GetData_struct() (ret StructType) {
	ret, err := makeStructFromMap[StructType](e.params, EnsureDataIsPresent)
	if err != nil {
		panic(err) // This is not supposed to fail, due to the properties of ErrorWithData[StructType]
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
// On keys that were not present, returns (nil, false).
func (e *errorWithParameters_common) GetParameter(parameterName string) (value any, present bool) {
	value, present = e.params[parameterName]
	return
}

// GetAllParameters returns a map of all parameters present in the error.
// The returned map is a (shallow) copy, so modification of values of the returned map does not affect the error.
func (e *errorWithParameters_common) GetData_map() (ret map[string]any) {
	ret = make(map[string]any, len(e.params))
	for key, value := range e.params {
		ret[key] = value
	}
	return
}

// Note: Returning pointer (to newly allocated object) or value may change.

// makeErrorWithParameterCommon creates a new errorWithParameters_common from the given baseError and interpolation string.
// The parameter map is copied from baseError.
//
// NOTE: This is an internal function, used by
// [deleteParameterFromError], [deleteParameterFromError_any], [newErrorWithData_struct], [newErrorWithData_map] and possibly the (exported) functions in modify_errors.go
// These functions actually modify ret.params after creation. The purpose of this method is just to unify those functions.
//
// This function is never supposed to panic. Errors are reported in-band inside ret.parsedInterpolationString.
// The [ValidateSyntax] method can be used to access these.
func makeErrorWithParametersCommon(baseError error, interpolationString string) (ret errorWithParameters_common) {

	ret.wrapped_error = baseError
	tokens := tokenizeInterpolationString(interpolationString)

	// Parse the intepolation string.
	// We ignore potential returned errors: If there is a parsing error, this is additionally recorded in ret.parsedInterpolationString itself.
	ret.parsedInterpolationString, _ = make_ast(tokens)
	ret.params = GetData_map(baseError)
	return
}

// deleteParameterFromError_any is the implementation for DeleteParameterFromError_any that is tied to our specific implementation of the
// [ErrorWithData_any] interface.
//
// It creates a new errorWithParameters_common based on baseError and interpolationString, with the parameter given by parameterName removed.
// If parameterName is not present to start with, this deletion has no effect. We still create a new error in that case.
//
// This function does not perform any input validation or replacement of empty interpolation string by "%w" resp. "$w". These are the caller's job.
func deleteParameterFromError_any(baseError error, interpolationString string, parameterName string) (ret *errorWithParameters_common) {
	ret = new(errorWithParameters_common)
	*ret = makeErrorWithParametersCommon(baseError, interpolationString)
	delete(ret.params, parameterName)
	return
}

// deleteParameterFromError is the implementation for DeleteParameterFromError that is tied to our specific implementation of the
// [ErrorWithData] interface.
//
// The behaviour mostly matches that of [deleteParametersFromError_any], but we need to handle mismatched/missing data that violates the
// newly created error's promise regarding StructType.
//
// If missingDataTreatment == [EnsureDataIsPresent], we fail if data is missing
// If missingDataTreatment == [MissingDataAsZero], we fill up missing data with zero values
//
// If data is present, but has the wrong type, we always fails.
//
// Any such failures are reported in err. If err != nil, the returned *ret should not be used.
//
// This function may panic if called with invalid StructType or invalid missingDataTreatment.
// The caller should ensure that.
//
// This function does not perform replacement of empty interpolation string by "%w" resp. "$w". These are the caller's job.
func deleteParameterFromError[StructType any](baseError error, interpolationString string, parameterName string, missingDataTreatment MissingDataTreatment) (ret *errorWithParameters_T[StructType], err error) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon(baseError, interpolationString)
	delete(ret.params, parameterName)
	err = ensureCanMakeStructFromParameters[StructType](&ret.errorWithParameters_common.params, missingDataTreatment)
	return
}

// NOTE: For newErrorWithData_struct and newErrorWithData_map, we expect the caller to have checked the validity of the input beforehand.
// (i.e. StructType is valid, interpolating string non-empty). As such, those function should not panic (unless mode says so).

// newErrorWithData_struct is the actual implementation of [NewErrorWithData_struct].
//
// It is tied to our particular implementation of ErrorWithData and so does not return an interface.
// It creates a new error with data with the given baseError, interpolationString and data.
// There are no checks on the inputs. In particular, interpolation string is assumed to have been replaced by "%w" or "$w" by the caller.
// mode controls how already-present data in baseError should be handled.
//
// The function may panic if called with invalid StructType, so StructType should be checked by the caller beforehand.
// By design, it also panics if mode==[AssertDataIsNotReplaced] and there is different data in base error.
func newErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, mode PreviousDataTreatment) (ret *errorWithParameters_T[StructType]) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon(baseError, interpolationString)
	fillMapFromStruct(data, &ret.errorWithParameters_common.params, mode)
	return
}

// newErrorWithData_map serves a similar purpose as [newErrorWithData_struct], but takes the new data via a ParamMap.
//
// Notably, it creates a new ErrorWithData based on baseError with the given interpolation string.
// mode controls how pre-existing data should be handled.
// missingDataTreatment controls how data that is missing (for constructing an instance of StructType) should be handled.
//
// There are no checks on inputs; in particular, interpolating string is assumed to have been replaced by "%w" or "$w" by the caller
// and we assume that mode and missingDataTreatment are valid.
// This function may panic if called with an invalid Struct type.
//
// If the given params (together with any params from baseError) are unsuited to construct an instance of StructType
// (such as params with wrong type or missing params with missingDataTreatment set to [EnsureDataIsPresent]),
// we return an error in err. If err!=nil, ret should not be used by the caller.
// If mode == [AssertDataIsNotReplaced], the function will panic on conflicting data.
func newErrorWithData_map[StructType any](baseError error, interpolationString string, mode PreviousDataTreatment, missingDataTreatment MissingDataTreatment, params ParamMap) (ret *errorWithParameters_T[StructType], err error) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon(baseError, interpolationString)
	mergeMaps(&ret.errorWithParameters_common.params, params, mode)
	err = ensureCanMakeStructFromParameters[StructType](&ret.errorWithParameters_common.params, missingDataTreatment)
	return
}

// ValidateSyntax checks whether the created error has any syntax error in its interpolation string
func (e *errorWithParameters_common) ValidateSyntax() error {
	// Check for parse errors
	if e.parsedInterpolationString.parseError != nil {
		return e.parsedInterpolationString.parseError
	}
	// If no parse errors, check for syntax errors.
	return e.parsedInterpolationString.handleSyntaxConditions()
}

// ValidateError_Final checks whether the created error contains certain errors that would trigger on .Error()
// Note that this recurses through %w and $w
func (e *errorWithParameters_common) ValidateError_Final() error {
	return e.parsedInterpolationString.VerifyParameters_passed(e.params, e.params, e.wrapped_error)
}

// ValidateError_Base checks whether the created errors contains certain errors, up to the fact that any $-statements are only syntax-checked
// Note that this recurses through %w and $w
func (e *errorWithParameters_common) ValidateError_Base() error {
	return e.parsedInterpolationString.VerifyParameters_direct(e.params, e.wrapped_error)
}

// ValidateError_Params checks whether the created error contains errors (in particular, ${VarName}-statements are valid), given the passed parameter map.
// params_passed == nil is taken as using e's own parameters (this is distinct from passing an empty map)
//
// This method is required for [ValidateError_Base] or [ValidateError_Final] to recurse and part of the [ErrorInterpolater] interface.
func (e *errorWithParameters_common) ValidateError_Params(params_passed ParamMap) error {
	return e.parsedInterpolationString.VerifyParameters_passed(e.params, params_passed, e.wrapped_error)
}
