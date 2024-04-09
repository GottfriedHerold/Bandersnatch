package errorsWithData

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
)

type joinedErrors_any struct {
	baseErrors []error
	params     ParamMap
}

type joinedErrors[StructType any] struct {
	joinedErrors_any
}

// NOTE: The if e==nil check and panic if true is only meaningful for joinedErrors_any.
// It does not extend to joinedErrors[StructType], because struct embedding tranlates e.f(...) to (*e).joinedErrors_any.f(...)
// if e has type joinedErrors[StructType]. In this expression, (*e).joinedErrors_any already panics before f would even be called.
// So we also get a panic, although a "default" one with generic error message.
// Due to that, the error messages in the panic from the explicit checks refer to joinedErrors_any rather than joinedErrors.

// Error_interpolate is provided to satisfy the [ErrorWithData_any] interface.
//
// For joinedErrors_any and joinedErrors, this just calls Error_interpolate (or Error, if not Error_interpolate is not supported)
// on all children and concatenates the results, separated by \n.
//
// This is consistent with the behavior of errors.Join
func (e *joinedErrors_any) Error_interpolate(params_passed ParamMap) string {
	if e == nil {
		panic(ErrorPrefix + "called Error_interpolate on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var s strings.Builder
	for i, child := range e.baseErrors {
		if child, ok := child.(ErrorInterpolater); ok {
			s.WriteString(child.Error_interpolate(params_passed))
		} else {
			s.WriteString(child.Error())
		}
		if i != len(e.baseErrors) {
			s.WriteRune('\n')
		}
	}
	return s.String()
}

// Error_interpolate is provided to satisfy the [ErrorWithData_any] and also the [error] interface.
//
// For joinedErrors_any and joinedErrors, this just calls Error
// on all children and concatenates the results, separated by \n.
//
// This is consistent with the behavior of errors.Join
func (e *joinedErrors_any) Error() string {
	if e == nil {
		panic(ErrorPrefix + "called Error on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var s strings.Builder
	for i, child := range e.baseErrors {
		s.WriteString(child.Error())
		if i != len(e.baseErrors) {
			s.WriteRune('\n')
		}
	}
	return s.String()
}

// Unwrap is provided to allow error wrapping. Note that this returns []error rather than error.
func (e *joinedErrors_any) Unwrap() []error {
	return e.baseErrors
}

// GetData_struct is provided for joinedErros to satisfy the [ErrorWithData] interface
func (e *joinedErrors[StructType]) GetData_struct() (ret StructType) {
	// identical to (*errorWithParameters[T]).GetData_struct

	//joinedErrors[StructType] is designed to satisfy the invariant that there can never be a problem extracting a struct of type StructType.
	// The chosen config here should not matter. It is chosen to detect more (internal) bugs.
	config := config_ImplicitZero{implicitZero: false}
	ret, err := makeStructFromMap[StructType](e.params, config)
	if err != nil {
		panic(err) // This is not supposed to fail, due to the invariants of the type
	}
	return
}

// HasParameter is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface
func (e *joinedErrors_any) HasParameter(parameterName string) bool {
	_, ok := e.params[parameterName]
	return ok
}

// GetParameter is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface
func (e *joinedErrors_any) GetParameter(parameterName string) (value any, present bool) {
	value, present = e.params[parameterName]
	return
}

// GetData_map is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface
func (e *joinedErrors_any) GetData_map() ParamMap {
	return maps.Clone(e.params)
}

// ValidateSyntax is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface.
//
// It calls ValidateSyntax for all child error that support this.
func (e *joinedErrors_any) ValidateSyntax() error {
	if e == nil {
		panic(ErrorPrefix + "called ValidateSyntax on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var foundErrors []error
	for _, child := range e.baseErrors {
		if child, ok := child.(interface{ ValidateSyntax() error }); ok {
			if issue := child.ValidateSyntax(); issue != nil {
				foundErrors = append(foundErrors, issue)
			}
		}
	}
	if foundErrors == nil {
		return nil
	}
	return errors.Join(foundErrors...)
}

// ValidateError_Final is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface.
//
// It calls ValidateError_Final for all child error that support this.
func (e *joinedErrors_any) ValidateError_Final() error {
	if e == nil {
		panic(ErrorPrefix + "called ValidateError_Final on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var foundErrors []error
	for _, child := range e.baseErrors {
		if child, ok := child.(interface{ ValidateError_Final() error }); ok {
			if issue := child.ValidateError_Final(); issue != nil {
				foundErrors = append(foundErrors, issue)
			}
		}
	}
	if foundErrors == nil {
		return nil
	}
	return errors.Join(foundErrors...)
}

// ValidateError_Base is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface.
//
// It calls ValidateError_Base for all child error that support this.
func (e *joinedErrors_any) ValidateError_Base() error {
	if e == nil {
		panic(ErrorPrefix + "called ValidateError_Base on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var foundErrors []error
	for _, child := range e.baseErrors {
		if child, ok := child.(interface{ ValidateError_Base() error }); ok {
			if issue := child.ValidateError_Base(); issue != nil {
				foundErrors = append(foundErrors, issue)
			}
		}
	}
	if foundErrors == nil {
		return nil
	}
	return errors.Join(foundErrors...)
}

// ValidateError_Params is provided for joinedErrors_any and joinedErrors to satisfy the [ErrorWithData_any] interface.
//
// It calls ValidateErrorParams recursively for any child error that supports [ErrorInterpolater].
func (e *joinedErrors_any) ValidateError_Params(params_passed ParamMap) error {
	if e == nil {
		panic(ErrorPrefix + "called ValidateError_Params on nil error of concrete type joinedErrors_any. This is a library bug, as nil errors of this type should never appear")
	}
	var foundErrors []error
	for _, child := range e.baseErrors {
		if child, ok := child.(ErrorInterpolater); ok {
			if issue := child.ValidateError_Params(params_passed); issue != nil {
				foundErrors = append(foundErrors, issue)
			}
		}
	}
	if foundErrors == nil {
		return nil
	}
	return errors.Join(foundErrors...)
}

// extractNonNilError will check whether x is an error, a slice or array containing errors and add either x or all elements of x to target, skipping any nils
// If x has other type, returns a non-nil error. In this case, it is unspecified what happens to target. The error message does not include ErrorPrefix and is supposed to be modfied by the caller.
// This must not be called with x==nil (this case needs to be handled by the caller)
//
// Note that the dynamic type of x may be []T, where T is an interface; in this case we check whether the *dynamic* type of each entry satisfies error.
func extractNonNilErrors(target *[]error, x any) (err error) {
	if x == nil {
		panic("Cannot happen.") // needs to be handled by the caller anyway, so we consider it a bug if this would happen.
	}
	if x, xIsError := x.(error); xIsError {
		*target = append(*target, x)
		return
	}
	// x is not a nil interface, so everything is always a valid; typeOfX is a (non-interface) type.
	// if x is a nil of type []T, the code below actually works as L will be 0.
	valueOfX := reflect.ValueOf(x)
	typeOfX := valueOfX.Type()
	kindOfX := typeOfX.Kind()
	switch kindOfX {
	case reflect.Array, reflect.Slice:
		L := valueOfX.Len() // Note: x may be a nil slice.
		for i := 0; i < L; i++ {
			entry := valueOfX.Index(i).Interface()
			if entryError, entryIsError := entry.(error); entryIsError {
				*target = append(*target, entryError)
			} else {
				err = fmt.Errorf("the %v'th entry %v of the slice/array passed does not satisfy error", i, entry)
				return
			}
		}
	default:
		err = fmt.Errorf("the entry %v passed is neither an error nor a slice/array", x)
	}

	return
}

// Join_any creates a new error that wraps all non-nil errors passed to it and merges their paramters.
// This is intended to be used with the %w{Number} or $w{Number} syntax of interpolation string.
// The resulting ret will have an Unwrap() []error method to wrap multiple errors (for compatibility of *some functions* of the [errors] standard library -- please check that doc).
// The resulting ret's error message will be the concatenation of the individual errors' messages, separated by "\n"
//
// Each arguments passed to Join_any must either be a supported flag (which alters Join_any's behaviour), an error or slice/array of errors (see note on covariance below).
// Arguments of other types cause Join_any to panic.
// We accept the following flags:
//
// - [PreferPreviousData], [ReplacePreviousData] (default), [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]: Controls how to handle data present in multiple passed errors with the same key.
// - [RecoverFromComparisonFunctionPanic] (default), [LetComparisonFunctionPanic]: Only meaningful if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set. Controls how panics during comparisons are handled.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
//
// Note that all flags are parsed (in order of appearence) before any non-flag argument is processed, so flags coming after a non-flag affect previous non-flags.
//
// Join_any will then process all errors that were passed to it (non-recursively iterating over slices/arrays, if needed) in order, skipping any nil errors.
// The parameter map of the resulting ret is construced as the union of the individual errors. Duplicate parameter names are handled according to the past flags;
// For the latter, the input errors are processed in order of appearance, so inputs earlier in appearance are considered "older".
//
// NOTE: When passing slices or arrays, Join_any supports (dynamic) argument covariance (as opposed to the Go language itself):
// It supports passing arguments x to it which may have (dynamic) type []T or [n]T for some T.
// In this case, we require only that the dynamic(!) type of every x[i] must satisfy error; T itself might not satisfy it.
// In particular, we support to pass []any - slices, provided each entry satisfies error.
func Join_any(errorsOrFlags ...any) (ret ErrorWithData_any, err error) {
	returnedValue := new(joinedErrors_any)
	ret = returnedValue // because it's a pointer, modifications to returnedValue will affect ret. We don't work with ret directly, because ret is an interface.

	returnedValue.baseErrors = make([]error, 0, len(errorsOrFlags)) // pre-allocate
	returnedValue.params = make(map[string]any)
	panic(0)
	return
}

// Join creates a new error that wraps all non-nil errors passed to it and merges their paramters.
// This is intended to be used with the %w{Number} or $w{Number} syntax of interpolation string.
// The resulting ret will have an Unwrap() []error method to wrap multiple errors (for compatibility of *some functions* of the [errors] standard library -- please check that doc).
// The resulting ret's error message will be the concatenation of the individual errors' messages, separated by "\n"
//
// Each arguments passed to Join must either be a supported flag (which alters Join's behaviour), an error or slice/array of errors (see note on covariance below).
// Arguments of other types cause Join to panic.
// We accept the following flags:
//
// - [PreferPreviousData], [ReplacePreviousData] (default), [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]: Controls how to handle data present in multiple passed errors with the same key.
// - [RecoverFromComparisonFunctionPanic] (default), [LetComparisonFunctionPanic]: Only meaningful if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set. Controls how panics during comparisons are handled.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
// - [MissingDataAsZero], [MissingDataIsError] (default): Controls whether data required for StructType that is missing is silently zero-initialized
//
// Note that all flags are parsed (in order of appearence) before any non-flag argument is processed, so flags coming after a non-flag affect previous non-flags.
//
// Join will then process all errors that were passed to it (non-recursively iterating over slices/arrays, if needed) in order, skipping any nil errors.
// The parameter map of the resulting ret is construced as the union of the individual errors. Duplicate parameter names are handled according to the past flags;
// For the latter, the input errors are processed in order of appearance, so inputs earlier in appearance are considered "older".
//
// NOTE: When passing slices or arrays, Join supports (dynamic) argument covariance (as opposed to the Go language itself):
// It supports passing arguments x to it which may have (dynamic) type []T or [n]T for some T.
// In this case, we require only that the dynamic(!) type of every x[i] must satisfy error; T itself might not satisfy it.
// In particular, we support to pass []any - slices, provided each entry satisfies error.
func Join[StructType any](errorsOrFlags ...any) (ret ErrorWithData[StructType], err error) {
	panic(0)
	return
}
