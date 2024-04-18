package errorsWithData

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// nonNilUnion is a utility function. It takes any errors or []errors and outputs a single slice that is the concatenation of these, with nils removed.
// If the output lenght would be 0, it outputs nil rather than a 0-lenght slice.
func nonNilUnion(errsOrSlices ...any) (union []error) {
	for _, arg := range errsOrSlices {
		if arg == nil {
			continue
		}
		switch arg := arg.(type) {
		case error:
			union = append(union, arg) // non-nil
		case []error:
			// Note: arg may be []error(nil); this is fine.
			for _, argElement := range arg {
				if argElement != nil {
					union = append(union, argElement)
				}
			}
		default:
			panic(fmt.Errorf(ErrorPrefix+"internal error: argument %v of type %T to NonNilUnion has neither type error nor []error", arg, arg))
		}

	}
	return
}

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
		if childInterpolatable, ok := child.(ErrorInterpolater); ok {
			s.WriteString(childInterpolatable.Error_interpolate(params_passed))
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

// NOTE/minor rant: While convenient, the reason that we check the dynamic type of each array/slice entry rather than the static type is actually one of simplicity.
// The reason is that taking []error (let alone [n]error for unknown n) does not work very well due to Go's lack of covariance. After all, the
// user may have a slice of type []T, say []ErrorWithData_any; converting this to an []error requires knowing T / using generics or using reflection;
// generics would require T to be passed explicitly as generic argument to Join_any.
// Note that T cannot be inferred, because type inference is not dynamic and it does not work with the way we pass flags or mix argument types via variadic arguments (Go lacks sum types).
// So we have to use reflection here anyway, where looking at the dynamic type(s) is natural.

// extractNonNilError will check whether x is an error, a slice or array (or pointer-to-slice or pointer-to-array) containing ~errors and add either x or all elements of x to target, skipping any nils in the slice/array.
// (Note: Anything added is Unboxed first.) target must not be nil. (but *target may be nil, acting like append)
// If x has type other than error, slice or array, returns a non-nil error. In this case, it is unspecified what happens to target. The error message does not include ErrorPrefix and is supposed to be modfied by the caller.
// This must not be called with x==any(nil) (this case needs to be handled by the caller anyway for unrelated reasons)
//
// Note that the dynamic type of x may be []T, where T is an interface; in this case we check whether the *dynamic* type of each entry satisfies error.
//
// extractNonNilErrors does not recurse. We abort on first error. Note that a named type based on a slice that also satisfies the error interface will be treated as an error rather than a slice.
func extractNonNilErrors(target *[]error, x any) (err error) {
	// needs to be handled by the caller anyway when separating flags from the rest of the arguments, so we consider it a bug if this would happen.
	if x == nil {
		panic("Cannot happen.")
	}
	if x, xIsError := x.(error); xIsError {
		x = UnboxError(x)
		*target = append(*target, x)
		return
	}
	// x is not a nil interface, so no reflection gotchas from reflect.ValueOf(nil) or reflect.TypeOf(nil). typeOfX is a valid (non-interface) type.
	// if x is a nil of type []T, the code below actually works as L will be 0 and loop is never entered.
	valueOfX := reflect.ValueOf(x)
	typeOfX := valueOfX.Type()
	kindOfX := typeOfX.Kind() // cannot be invalid or interface
	switch kindOfX {
	case reflect.Array, reflect.Slice:
		L := valueOfX.Len() // Note: x may be a nil slice.
		for i := 0; i < L; i++ {
			entry := valueOfX.Index(i).Interface()
			if entry == nil {
				continue
			}
			if entryError, entryIsError := entry.(error); entryIsError {
				entryError = UnboxError(entryError)
				*target = append(*target, entryError)
			} else {
				err = fmt.Errorf("the %v'th entry %v of the slice/array passed does not satisfy error", i, entry)
				return
			}
		}
	case reflect.Pointer:
		typeOfElem := typeOfX.Elem()
		kindOfElem := typeOfElem.Kind()
		if kindOfElem != reflect.Array && kindOfElem != reflect.Slice {
			err = fmt.Errorf("the entry %v of type %T passed is neither an error nor a slice/array nor a pointer-to-array/slice", x, x)
			return
		}
		if valueOfX.IsNil() {
			return
		}
		valueOfElem := valueOfX.Elem()
		L := valueOfElem.Len() // *x may be a nil slice. This is fine; in this case L is 0.
		for i := 0; i < L; i++ {
			entry := valueOfElem.Index(i).Interface()
			if entry == nil {
				continue
			}
			if entryError, entryIsError := entry.(error); entryIsError {
				entryError = UnboxError(entryError)
				*target = append(*target, entryError)
			} else {
				err = fmt.Errorf("the %v'th entry %v of the pointer-to-array/slice passed does not satisfy error", i, entry)
				return
			}
		}

	default:
		err = fmt.Errorf("the entry %v passed is neither an error nor a slice/array nor pointer-to-array/slice", x)
	}

	return
}

// NOTE: List of accepted flags here must exactly match the list from validFlags_JoinAny resp. validFlags_Join in flag_test.go

// Join_any creates a new error that wraps all non-nil errors passed to it and merges their paramters.
// This is intended to be used with the %w{Number} or $w{Number} syntax of interpolation string.
// The resulting ret will have an Unwrap() []error method to wrap multiple errors (for compatibility of *some functions* of the [errors] standard library -- please check that doc).
// The resulting ret's error message will be the concatenation of the individual errors' messages, separated by "\n"
//
// Each arguments passed to Join_any must either be a supported flag (which alters Join_any's behaviour), an error or a slice/array of errors (see note on covariance below).
// Arguments of other types cause Join_any to panic (even if [ReturnError] is set).
// We accept the following flags:
//
// - [PreferPreviousData], [ReplacePreviousData] (default), [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]: Controls how to handle data present in multiple passed errors with the same key.
// - [RecoverFromComparisonFunctionPanic] (default), [LetComparisonFunctionPanic]: Only meaningful if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set. Controls how panics during comparisons are handled.
// - [ReturnError] (default), [PanicOnAllErrors]: Controls whether the function should panic on errors (useful when creating global errors on init)
//
// Note that all flags are parsed (in order of appearence) before any non-flag argument is processed, so flags coming after a non-flag affect previous non-flags.
//
// Join_any will then process all errors that were passed to it (non-recursively iterating over slices/arrays, if needed) in order, skipping any nil errors.
// The parameter map of the resulting ret is construced as the union of the individual errors. Duplicate parameter names are handled according to the passed flags;
// For the latter, the input errors are processed in order of appearance, so inputs earlier in appearance are considered "older".
//
// NOTE: This function can only fail in a non-panicking way if [EnsureDataIsNotReplaced] or [EnsureDataIsNotReplaced_fun] is set.
// As [PanicOnAllErrors] only affects these kinds of errors, it is only meaningful if one of those two flags is set as well.
//
// NOTE: When passing slices or arrays, Join_any supports a form of (dynamic) argument covariance (as opposed to the Go language itself):
// It supports passing arguments x to it which may have (dynamic) type []T or [n]T for some T.
// In this case, we require only that the dynamic(!) type of every x[i] must satisfy error; T itself might not satisfy it.
// In particular, we support to pass []any - slices, provided each entry satisfies error.
func Join_any(errorsOrFlags ...any) (ret ErrorWithData_any, err error) {
	returnedValue := new(joinedErrors_any)
	ret = returnedValue // because it's a pointer, modifications to returnedValue will affect ret. We don't work with ret directly, because ret is an interface.

	// baseErrors := make([]error, 0, len(errorsOrFlags)) // pre-allocate
	returnedValue.baseErrors = make([]error, 0, len(errorsOrFlags))
	returnedValue.params = make(map[string]any)
	flagArgs := make([]flagArgument_JoinAny, 0, len(errorsOrFlags))

	// parse the passed errorsOrFlags: flags go into flagArgs, errors into returnedValue.baseErrors
	for _, argument := range errorsOrFlags {
		if argument == nil {
			continue // skip nils
		}
		switch argument := argument.(type) {
		case flagArgument_JoinAny:
			flagArgs = append(flagArgs, argument)
		case flagArgument: // special case for better error messages
			panic(fmt.Errorf(ErrorPrefix+"Join_Any called with flag %v that is not supported by Join_Any", argument))
		default:
			errArguments := extractNonNilErrors(&returnedValue.baseErrors, argument)
			if errArguments != nil {
				panic(fmt.Errorf(ErrorPrefix+"invalid argument to Join_Any:%w", errArguments))
			}
		}
	}

	// Actually process the collected flags into config
	var config errorCreationConfig // Zero value is appropriate
	parseFlagArgs(&config, flagArgs...)

	var allDataErrors []error // collect all errors encountered from [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]

	// create the new error's paramMap from the individual base errors.
	for _, baseError := range returnedValue.baseErrors {
		paramsFromBase := GetData_map(baseError)
		newDataErrors := mergeMaps(&returnedValue.params, paramsFromBase, config.config_OldData)
		if newDataErrors != nil {
			allDataErrors = append(allDataErrors, newDataErrors...)
			// We do not abort on first error.
		}
	}

	if allDataErrors != nil {
		err = fmt.Errorf(ErrorPrefix+"Join_any encountered a data inconsistency in the given errors:\n%w", errors.Join(allDataErrors...))
	}
	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}
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
	// trigger early panic for invalid StructType
	if errInvalidStruct := StructSuitableForErrorsWithData[StructType](); errInvalidStruct != nil {
		panic(errInvalidStruct)
	}

	returnedValue := new(joinedErrors[StructType])
	ret = returnedValue // because it's a pointer, modifications to returnedValue will affect ret. We don't work with ret directly, because ret is an interface.

	// baseErrors := make([]error, 0, len(errorsOrFlags)) // pre-allocate
	returnedValue.baseErrors = make([]error, 0, len(errorsOrFlags))
	returnedValue.params = make(map[string]any)
	flagArgs := make([]flagArgument_Join, 0, len(errorsOrFlags))

	// parse the passed errorsOrFlags: flags go into flagArgs, errors into returnedValue.baseErrors
	for _, argument := range errorsOrFlags {
		if argument == nil {
			continue // skip nils
		}
		switch argument := argument.(type) {
		case flagArgument_Join:
			flagArgs = append(flagArgs, argument)
		case flagArgument:
			panic(fmt.Errorf(ErrorPrefix+"Join called with flag %v that is not supported by Join", argument))
		default:
			errJoining := extractNonNilErrors(&returnedValue.baseErrors, argument)
			if errJoining != nil {
				panic(fmt.Errorf(ErrorPrefix+"invalid argument to Join:%w", errJoining))
			}
		}
	}

	// Actually process the collected flags into config
	var config errorCreationConfig // Zero value is appropriate
	parseFlagArgs(&config, flagArgs...)

	var allDataErrors []error // collect all errors encountered from [EnsureDataIsNotReplaced], [EnsureDataIsNotReplaced_fun]

	// create the new error's paramMap from the individual base errors.
	for _, baseError := range returnedValue.baseErrors {
		paramsFromBase := GetData_map(baseError)
		newDataErrors := mergeMaps(&returnedValue.params, paramsFromBase, config.config_OldData)
		if newDataErrors != nil {
			allDataErrors = append(allDataErrors, newDataErrors...)
			// We do not abort on first error.
		}
	}

	errMissingData := ensureCanMakeStructFromParameters[StructType](&returnedValue.params, config.config_ImplicitZero, config_SetZeros{setErrorsToZero: true})

	if allDataErrors != nil && errMissingData != nil {
		allDataErrors = append(allDataErrors, errMissingData)
		err = fmt.Errorf(ErrorPrefix+"Join encountered a data inconsistency in the given errors and data was missing:\n%w", errors.Join(allDataErrors...))
	} else if allDataErrors != nil && errMissingData == nil {
		err = fmt.Errorf(ErrorPrefix+"Join encountered a data inconsistency in the given errors:\n%w", errors.Join(allDataErrors...))
	} else if allDataErrors == nil && errMissingData != nil {
		err = fmt.Errorf(ErrorPrefix+"Join called with errors whose parameters do not allow construct a %v:\n%w", utils.NameOfType[StructType](), errMissingData)
	}

	if err != nil && config.PanicOnAllErrors() {
		panic(err)
	}
	return
}
