package errorsWithData

import (
	"strings"
)

// This file is part of the errorsWithData package and defines the (currently only) implementation of the [ErrorWithData]
// and [ErrorWithData_any] interface.
// Note that the user-facing functions in modify_errors.go that are used to actually create an [ErrorWithData] or [ErrorWithData_any]
// are somewhat coupled to the concrete implementation.
// We try to keep the specifics to the concrete implementation in this file here, so the functions in modify_errors.go
// are mostly wrappers to some function defined here.
//
// NOTE: The functions in modify_errors.go usually call a function from plainErrorWithParams.go and store the result in an [ErrorWithData]
// or [ErrorWithData_any] interface (and from that point on only uses the interface API). This is done to separate the interface from the implementation.
// Observe, however, that this means that the functions in modify_errors.go have no way to modify these result without adding a
// level of wrapping, because [ErrorWithData] only allows modifying copies. Extra wrapping breaks validation ([ValidateSyntax] does not follow the error chain).
// For that reason, the functions defined in plainErrorWithParams.go need to basically create the final result "in one go";
// different functions in modify_errors.go call completely different functions in plainErrorWithParams.go.
// If there is shared code, it will therefore be in plainErrorWithParams.go; this is particularly annoying for the
// distinction between *_any variants and the non-*_any variants:
// keeping the code "clean" by maintaining this separation neccessitates copy&pasting a lot of the actual code in modifyErrors.go.

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
// It is identical as a struct and some methods coincide, so we just stuct-embed [errorWithParameters_common]
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

// NOTE: The e == nil checks can never be satisfied if the concrete type of e is errorWithParameters_T due the way struct-embedding in Go works:
// In this case, Go will resolve the struct-embedding by calling (*e).errorWithParameters_common.METHOD(...), which dereferences nil and panics
// before the promoted method is even called.

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
	// errorWithParameters_T is designed to satisfy the invariant that there can never be a problem extracting a struct of type StructType.
	// The chosen config here should not matter. It is chosen to detect more (internal) bugs.
	config := config_ImplicitZero{implicitZero: false}
	ret, err := makeStructFromMap[StructType](e.params, config)
	if err != nil {
		panic(err) // This is not supposed to fail, due to the properties of ErrorWithData[StructType].
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

// precompute tokenizations. Note that we cannot precompute the creation of the ast as easily.
var (
	tokenListParentWithError = tokenizeInterpolationString("$w")
	tokenListParent          = tokenizeInterpolationString("%w")
)

// Note: Returning pointer (to newly allocated object) or value may change.

// makeErrorWithParameterCommon creates a new errorWithParameters_common from the given baseError and interpolation string.
// The parameter map is copied from baseError.
//
// NOTE: This is an internal function, used by
// [deleteParameterFromError], [deleteParameterFromError_any], [newErrorWithData_struct], [newErrorWithData_map] and possibly the (exported) functions in modify_errors.go
// These functions actually modify ret.params after creation. The purpose of this method is just to unify those functions.
//
// This function does not directly report errors. Errors are reported in-band inside ret.parsedInterpolationString.
// The [ValidateSyntax] method can be used to access these.
// If config.AllowEmptyString is NOT set, interpolationString == "" and baseError == nil, the function panics.
// This needs to be handled at the call site (mostly, because we want the error message to originate from the call sites rather than from here)
func makeErrorWithParametersCommon_any(baseError error, interpolationString string, config config_EmptyString) (ret errorWithParameters_common) {
	var tokens tokenList
	ret.wrapped_error = baseError
	if !config.AllowEmptyString() && interpolationString == "" {
		if baseError == nil {
			panic("This must be unreachable") // needs to be caught at call site
		}

		// Note: We cannot set ret.parsedInterpolationString to a precomputed AST-parsed "$w" resp. "%w" here.
		// If we did that, every such error would actually share the same ast. This would almost be fine, except
		// that validation actually reports and *stores* validation errors in the ast itself.
		// Validation follows the error chain, so validation of "$w" or "%w$ can actually fail.
		//
		// for the tokenizing step, there is no such issue, so we can use precomputed tokenLists rather than re-tokinzing
		// "$w" or "%w"
		if _, baseSupportParams := baseError.(ErrorInterpolater); baseSupportParams {
			tokens = tokenListParentWithError // "$w"
		} else {
			tokens = tokenListParent // "%w"
		}
	} else {
		tokens = tokenizeInterpolationString(interpolationString)
	}
	// Parse the intepolation string.
	// We ignore potential returned errors: If there is a parsing error, this is additionally recorded in ret.parsedInterpolationString itself.
	ret.parsedInterpolationString, _ = make_ast(tokens)
	_ = ret.parsedInterpolationString.handleSyntaxConditions() // ensure this is called at least once.
	ret.params = GetData_map(baseError)
	return
}

// makeErrorWithParameterCommon creates a new errorWithParamters_T[StructType] from the given base error and interpolation string.
//
// This works similar to (and indeed just calls) [makeErrorWithParameterCommon_any]. Again, note that this is an internal function
// that only serves to unify functions within this file.
// The additional parameter c_ImplicitZero controls whether missing data should be either
//   - *silently* zero-initialized
//   - zero-initialzed and trigger an error
//
// This function reports errors in two separate channels:
// The returned ret allows to check for syntax and parse errors by querying ret for those via [ValidateSyntax] etc.
// What is returned in err is *only* errors about a mismatch between StructType and what is in baseError's ParamMap.
//
// NOTE: This function is only really useful for AsErrorWithData, i.e. casting. The reason for that is this function does not take "extra" parameters outside from baseError.
// As a consequence, this function is only ever called with interpolationString == "" and defaulted c_emptyString.
// For those, there can be no syntax / parse errors.
//
// NOTE2: We could implement public API function to create instances of ErrorWithParam[T] by always using some *_any variant, then doing modification
// and finally converting with makeErrorWithParameterCommon. This is *NOT* a good idea:
// The reason is that makeErrorWithParamerCommon actually create a level of indirection by wrapping; as a consequence, ret.ValidateSyntax() will never report errors
// as this won't follow the chain.
//
// This indirection could only be avoided by special-casing everything if baseError has type *errorWithParameters_common.
// However, this would be hacky, because it would not work with a user-defined implementation of ErrorWithData_any.
// At any rate, the needed functionality (without wrapping) is literally provided by ensureCanMakeStructFromParameters;
// we just don't call ensureCanMakeStructFromParameters from any public API function in modify_errors_go, but rather call a small function in this file
// that in turn calls ensureCanMakeStructFromparameters, because the latter is not part of the ErrorWithData interface.
func makeErrorWithParameterCommon[StructType any](baseError error, interpolationString string, c_emptyString config_EmptyString, c_ImplicitZero config_ImplicitZero) (ret errorWithParameters_T[StructType], err error) {
	ret.errorWithParameters_common = makeErrorWithParametersCommon_any(baseError, interpolationString, c_emptyString)
	err = ensureCanMakeStructFromParameters[StructType](&ret.params, c_ImplicitZero, config_SetZeros{setErrorsToZero: true})
	return
}

// simplified replacement for makeErrorWithParamtersCommon. The only use-case was AsErrorWithData, which had some arguments hardwired. We don't need those and can simplify the API (and doc).

// asErrorWithData is the implementation for AsErrorWithData that is tied to our specific implementation of the [ErrorWithData] interface.
//
// This function just "converts" (by wrapping) an arbitrary base error into a *errorWithParameters_T[StructType].
// Note that we do not allow changing/setting the interpolation string.
//
// c_ImplicitZero determines whether zero-ing missing parameters is silent or not.
//
// This function must not be called with baseError == nil.
//
// NOTE: This function always wraps baseError, even if baseError satisfies [ErrorWithData_any], for consistency reasons.
// We could implement public API function to create instances of ErrorWithParam[T] by always using some *_any variant, then doing modification
// and finally converting with asErrorWithData. This is *NOT* a good idea:
// The reason is that asErrorWithData creates a level of indirection by wrapping; as a consequence, ret.ValidateSyntax() will never report errors
// as this won't follow the chain.
//
// This indirection could only be avoided by special-casing everything if baseError has type *errorWithParameters_common.
// However, this would be hacky, because then the exported [AsErrorWithData] would work differently with a user-defined implementation of ErrorWithData_any.
// At any rate, the needed functionality (without wrapping) is literally provided by ensureCanMakeStructFromParameters;
// we just don't call ensureCanMakeStructFromParameters from any public API function in modify_errors_go, but rather call a small function in this file
// that in turn calls ensureCanMakeStructFromparameters, because the latter is not part of the ErrorWithData interface.
func asErrorWithData[StructType any](baseError error, c_ImplicitZero config_ImplicitZero) (ret *errorWithParameters_T[StructType], err error) {
	if baseError == nil {
		panic("Must not happen")
	}
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon_any(baseError, "", config_EmptyString{})
	err = ensureCanMakeStructFromParameters[StructType](&ret.params, c_ImplicitZero, config_SetZeros{setErrorsToZero: true})
	return
}

// deleteParameterFromError_any is the implementation for DeleteParameterFromError_any that is tied to our specific implementation of the
// [ErrorWithData_any] interface.
//
// It creates a new errorWithParameters_common based on baseError and interpolationString, with the parameter given by parameterName removed.
// If parameterName is not present to start with, this deletion has no effect. We still create a new error in that case.
//
// c_EmptyString toggles how an empty interpolationString is interpreted.
// If c_EmptyString.AllowEmptyString() is true, no special handling is performed.
// If c_EmptyString.AllowEmptyString() is false (the default), we default to "%w" resp. to "$w" if interpolationString is empty.
// In this case, this function panics if baseError == nil. This case must be caught at the call site rather than letting this function panic, because the error message is not right.
func deleteParameterFromError_any(baseError error, interpolationString string, parameterName string, c_EmptyString config_EmptyString) (ret *errorWithParameters_common) {
	ret = new(errorWithParameters_common)
	*ret = makeErrorWithParametersCommon_any(baseError, interpolationString, c_EmptyString)
	delete(ret.params, parameterName)
	return
}

// deleteParameterFromError is the implementation for DeleteParameterFromError that is tied to our specific implementation of the
// [ErrorWithData] interface.
//
// The behaviour mostly matches that of [deleteParametersFromError_any], but we need to handle mismatched/missing data that violates the
// newly created error's promise regarding StructType.
//
// If c_ImplicitZero.IsMissingDataError() == true, missing data is an error, otherwise we silently fill up with zero values.
// (Note that we fill up with zero values even on error, just not silently)
//
// If data is present, but has the wrong type, we always fails.
//
// Any such failures are reported in err. If err != nil, the returned *ret will have failing entries zero-initialized; all non-failing entries are valid (i.e. we do not abort on first error).
//
// This function may panic if called with invalid StructType.
// The caller must make sure ensure that the combination interpolationString == "", c_EmptyString.AllowEmptyString() == false, baseError == nil never happens.
// (The function would panic with an uninformative error message)
func deleteParameterFromError[StructType any](baseError error, interpolationString string, parameterName string, c_ImplicitZero config_ImplicitZero, c_EmptyString config_EmptyString) (ret *errorWithParameters_T[StructType], err error) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon_any(baseError, interpolationString, c_EmptyString)
	delete(ret.params, parameterName)
	err = ensureCanMakeStructFromParameters[StructType](&ret.errorWithParameters_common.params, c_ImplicitZero, config_SetZeros{setErrorsToZero: true})
	return
}

// newErrorWithData_struct is the actual implementation of [NewErrorWithData_struct].
//
// It is tied to our particular implementation of ErrorWithData and so does not return an interface.
// It creates a new error with data with the given baseError, interpolationString and data.
//
// NOTE: We must not call this with interpolationString == "", baseError==nil, c_EmptyString set to default.
// (We need to handle this at the call site, possibly by panic)
//
// We return all errors as a slice. The returned slice is nil (rather than empty) iff there was no error.
func newErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, c_OldData config_OldData, c_EmptyString config_EmptyString) (ret *errorWithParameters_T[StructType], errs []error) {

	ret = new(errorWithParameters_T[StructType])

	// This would panic if interpolationString=="", baseError==nil, c_EmptyString defaulted. Ruled out by pre-condition.
	ret.errorWithParameters_common = makeErrorWithParametersCommon_any(baseError, interpolationString, c_EmptyString)
	errs = fillMapFromStruct(data, &ret.errorWithParameters_common.params, c_OldData)

	// NOTE: We are given *data, so we are guaranteed to have no missing keys in the resulting ParamMap.
	// However, pre-existing data in baseError might have the wrong type. So if c_OldData.preferOld is set, this can still fail,
	// config_ImplictZero{} is irrelevant (missing data cannot happen anyway),
	// config_SetZero{setErrorToZero:true} is here to replace pre-existing data in baseError of the wrong type by a zero value;
	// This is done to ensure that the invariants we expect from errorWithParameters_T[StructType] are satisfied even on error.
	// (The option to unconditionally take the new value from *data on error is not supported by our API)
	wrongTypeErrors := ensureCanMakeStructFromParameters[StructType](&ret.params, config_ImplicitZero{}, config_SetZeros{setErrorsToZero: true})
	if wrongTypeErrors != nil {
		errs = append(errs, wrongTypeErrors)
	}
	return
}

// newErrorWithData_map serves a similar purpose as [newErrorWithData_struct], but takes the new data via a ParamMap.
//
// Notably, it creates a new ErrorWithData based on baseError with the given interpolation string.
// c_OldData controls how pre-existing data should be handled.
// c_ImplicitZero controls how data that is missing (for constructing an instance of StructType) should be handled.
// c_EmptyString controls how an empty interpolationString is interpreted.
//
// This function may panic if called with an invalid Struct type.
//
// If the given params (together with any params from baseError) are unsuited to construct an instance of StructType
// (such as params with wrong type or missing params with c_ImplitZero set via [MissingDataIsError]), we return an error in err.
//
// Note that err contains information about all failing fields of StructType, not just the first failing one. The value returned in ret
// has zeroed entries for all failing fields and the requested values for all other fields.
func newErrorWithData_map[StructType any](baseError error, interpolationString string, params ParamMap, c_OldData config_OldData, c_ImplicitZero config_ImplicitZero, c_EmptyString config_EmptyString) (ret *errorWithParameters_T[StructType], err error) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon_any(baseError, interpolationString, c_EmptyString)
	mergeMaps(&ret.errorWithParameters_common.params, params, c_OldData)

	// We want to maintain the invariant for the returned value even on error, so we zero out bad values.
	err = ensureCanMakeStructFromParameters[StructType](&ret.errorWithParameters_common.params, c_ImplicitZero, config_SetZeros{setErrorsToZero: true})
	return
}
