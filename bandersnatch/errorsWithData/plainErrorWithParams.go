package errorsWithData

import (
	"strings"
	"unicode/utf8"
)

// errorWithParameters_common is a simple implementation of the [ErrorWithData_any] interface
// NOTE:
// exported functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned as typed nil pointers, which is a serious footgun)
//
// [MakeIncomparable] is kind-of an exception to this (it panics on nil, anyway).
type errorWithParameters_common struct {
	contained_error           error    // wrapped underlying error, can be nil
	parsedInterpolationString ast_I    // (parsed) error message
	params                    ParamMap // map strings -> data.
}

// extension of errorWithParameters_common that satisfies ErrorWithData[StructType]
type errorWithParameters_T[StructType any] struct {
	errorWithParameters_common
}

// forgetStructType "downcasts" ErrorWithData to ErrorWithData_any.
//
// Note: The same effect can be achieved by plain assignment (one is a sub-interface of the other, after all)
// What this function does is actually changing the implementing struct type and removing the extra functionality.
// This is only done for consistency:
// The point is that the actual implementing struct type, or at least the exported extra methods, are in principle visible (e.g. by type-asserting).
// In some cases, we create ErrorWithData_any variables from an errorWithParameters_T[struct{}] rather
// than a errorWithParameters_common. We can use forgetStructType to *consistently* use errorWithParameters_common as the implementing type.
func forgetStructType[StructType any](err ErrorWithData[StructType]) ErrorWithData_any {
	errImpl, ok := err.(*errorWithParameters_T[StructType])
	if !ok {
		return err
	} else {
		return &errImpl.errorWithParameters_common
	}
}

// Error_interpolate is provided to satisfy the ErrorWithData_any interface. This is an extension of Error() that
// will interpolate the error string (which may contain formatting instructions like %{Param1}, ${Param2} ).
// As opposed to plain Error(), formatting instructions with $ differs from those with % by taking the parameters from params_passed rather than using the
// parameters stored in the error. The presence of this method is what makes the "$w" formatting verb work.
func (e *errorWithParameters_common) Error_interpolate(params_passed map[string]any) string {
	if e == nil {
		panic(ErrorPrefix + "called Error_interpolate() on nil error of concrete type errorWithParameters_common. This is a bug, since nil errors of this type should never exist.")
	}
	var s strings.Builder
	e.parsedInterpolationString.Interpolate(e.params, params_passed, e.contained_error, &s)
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
	return e.contained_error
}

// GetData is provided to satisfy ErrorWithParameters[StructType].
//
// It constructs a value of type StructType from the provided parameters.
func (e *errorWithParameters_T[StructType]) GetData_struct() (ret StructType) {
	ret, err := makeStructFromMap[StructType](e.params, EnsureDataIsPresent)
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
func (e *errorWithParameters_common) GetData_map() (ret map[string]any) {
	ret = make(map[string]any, len(e.params))
	for key, value := range e.params {
		ret[key] = value
	}
	return
}

// makeErrorWithParameterCommon creates a new errorWithParameters_common from the given baseError and interpolation string.
// The parameter map is copied from baseError
//
// NOTE: This is an internal function, used by the (exported) functions in modify_errors.go
// These functions actually modify ret.params after creation. The purpose of this method is just to unify those functions.
func makeErrorWithParametersCommon(baseError error, interpolationString string) (ret errorWithParameters_common) {
	if !utf8.ValidString(interpolationString) {
		panic(ErrorPrefix + "message for error creation was not a valid UTF-8 string")
	}

	ret.contained_error = baseError
	tokens := tokenizeInterpolationString(interpolationString)

	// Parse the intepolation string.
	// We ignore potential returned errors: If there is a parsing error, this is additionally recorded in ret.parsedInterpolationString itself.
	ret.parsedInterpolationString, _ = make_ast(tokens)
	ret.params = GetData_map(baseError)
	return
}

// newErrorWithData_struct is the actual implementation of [NewErrorWithData_struct].
//
// It is tied to our particular implementation of ErrorWithData and does not return an interface.
// There are no checks on the inputs. In particular, interpolation string is assumed to have been replaced by "%w" or "$w" by the caller.
//
// The function may panic if called with invalid StructType, so StructType should be checked by the caller beforehand.
func newErrorWithData_struct[StructType any](baseError error, interpolationString string, data *StructType, mode PreviousDataTreatment) (ret *errorWithParameters_T[StructType]) {
	ret = new(errorWithParameters_T[StructType])
	ret.errorWithParameters_common = makeErrorWithParametersCommon(baseError, interpolationString)
	fillMapFromStruct(data, &ret.errorWithParameters_common.params, mode)
	return
}

// ValidateSyntax checks whether the created error has any syntax error in its interpolation string
func (e *errorWithParameters_common) ValidateSyntax() error {
	return e.parsedInterpolationString.VerifySyntax()
}

// ValidateError_Final checks whether the created error contains certain errors that would trigger on .Error()
func (e *errorWithParameters_common) ValidateError_Final() error {
	return e.parsedInterpolationString.VerifyParameters_passed(e.params, e.params, e.contained_error)
}

// ValidateError_Base checks whether the created errors contains certain errors, up to the fact that any $-statements are only syntax-checked
func (e *errorWithParameters_common) ValidateError_Base() error {
	return e.parsedInterpolationString.VerifyParameters_direct(e.params, e.contained_error)
}

// ValidateError_Params checks whether the created error contains errors (in particular, ${VarName}-statements are valid), given the passed parameter map.
// params_passed == nil is taken as using e's own parameters (this is distinct from passing an empty map)
func (e *errorWithParameters_common) ValidateError_Params(params_passed ParamMap) error {
	return e.parsedInterpolationString.VerifyParameters_passed(e.params, params_passed, e.contained_error)
}
