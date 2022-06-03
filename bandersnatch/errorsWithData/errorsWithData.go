package errorsWithData

import (
	"errors"
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file defines functionality to add arbitrary paramters to errors in a way that is compatible with error wrapping.
//
// Parameters can be added and retrieved to errors in two flavours: as a map[string]interface{} or a structs.
// We allow both interchangably, identifying a struct{A: x, B: y} with a map {"A":x, "B":y}, i.e.
// the map keys are the field names (this gives some restrictions on what struct types are allowed).
// The map/struct interfaces can be mixed-and-matched and when retrieving a struct only a subset of the parameters might be actually used.
// Naming-wise, the map interface talks about "parameters" and the struct interface about "data"
//
// The publicly-facing interface operates on errors of plain error type and is compatible with error wrapping.
// We (need to) treat errors as immutable objects, so any modification to the parameters of an error will create a new one, typically wrapping the old.
//
// Errors are returned in a parameterized interface ErrorWithParameters[StructType], where StructType is a struct type.
// This interface extends error and for a struct type struct{A:type1, B:type2} non-nil errors of type ErrorWithParameters[StructType]
// are guaranteed to contain (at least) parameters under keys "A" and "B" of appropriate type.
// Generally speaking, this (and retrievability as structs) exists purely as a way to get some partial type-safety.
//
// We recommend adding / retrieving via the struct interface for at least some compile-time type-safety.
// When using the map interface, we recommend defining string constants for the map keys.
//
// We assert that any errors that are contained in error chains are either nil *interfaces* or non-nil.
// In particular, no nil error of concrete struct (pointer) type shall ever appear.
//
// Restrictions on StructTypes: Adding/retrieving data as structs has the following restrictions
// All field names must be exported. Fields of interface type are allowed. embedded struct are also allowed; anything else causes a panic.
// Embededded fields act in the following way:
// For structs
// type Struct1 struct{Data1 bool; Data2 int}
// type Struct2 struct{Struct1; Data2 string}  (Note: actually, neither Struct1 or Struct2 actually need to be exported, only their fields)
//
// after adding data from an instance of type Struct2, we can retrieve parameters (using the map interface) under the keys
// "Data1" (yielding a bool) and "Data2" (yielding a string). There are no keys "Struct1" or "Struct1.Data1", "Struct1.Data2".
// In particular, the shadowed int from Struct1.Data2 is completely ignored when adding data.
// When retrieving data as an instance s of Struct2, s.Struct1.Data2 is zero-inintialized.
// In particular, shadowed fields not not roundtrip.

// Since not even the standard library function specifies whether error wrapping works by reference or copy and we
// do not want users to accidentially modify exported errors that people compare stuff against, we are essentially forced
// to treat errors as immutable objects. This form of immutability is possibly shallow.

/////////////

// Implementation considerations:
// Any kind of AddParametersToError(existingError error, params...) or possibly AddParametersToError(*error, params...)
// that we create runs afoul of the issue that existing errors do not support this;
// So we either
// a) maintain a separate global registry ([pointer-to-???]error -> parameter map) as aside-lookup table to lookup parameters without touching the existing errors
// or
// b) we create new wrappers (of a new type) that wrap the existing errors and support the interface.
// The issue with a) is that we cannot know when and how errors are copied.
//
// After
// 		err2 := err1
// 		Add parameter to err2 (possibly overwriting the err2 variable)
//		err3 := err2
//
// the parameters should be in err2 and err3, but not in err1. Keying the map by pointers-to-errors will break at err3:=err2
// Keying the map by errors itself will only work if we overwrite err2 by something that is unequal to err1 upon adding parameters.
// Basically, we would need to create a wrapper around err2, replace err2 by the wrapper and key the global registry by the wrapper.
// However, this means we need to touch the existing errors and their type (due to replacement with wrapper), so b) is actually better.
//
// On b) we would just create an error wrapper that supports the functionality and create an error chain using Unwrap()
// The resulting errors have an extended interface to communicate the functionality via the type system.

// For the wrapper, we define an interface (with private methods, even though there is only 1 implementation)
// This is because our wrappers needs to be ALWAYS returned as interfaces by our API, never as a concrete type.
// Doing otherwise is a serious footgut, since the zero value is a nil pointers of concrete type and will be non-nil
// (in the sense that comparing to nil gives false) when assigned to an (e.g. standard error) interface.
// For that reason, we consider the existence of any nil error of concrete type defined here a bug.
// Our API does not enable to create nil pointers of concrete type.

// We use the type system to communicate that certain parameters are guaranteed to be present on non-nil errors.
// (This is done so we get a least some compile-time(!) checks on the side creating the error for this, as
// error handling is prone to bad testing coverage)

// errorWithParameters_commonInterface is the common interface satified by all types satisfying ErrorWithParamerers[StructType]
//
// it allows retrieving the associated data for an error *WITHOUT* following the error chain.
// We really only need it to query all errors encountered along an arbitrary error's error chain for whether
// they (potentially) contain any data.
//
// Note that since the above is the only usage and we only have 1 real implementation via errorsWithParameters_common (the StructType parameter is tacked-on via struct embedding),
// we really could have defined this as an interface{ accessErrorParams() *errorsWithParameters_common },
// However, this is less clear and relies on being able to decay the implementations of ErrorWithParameters[T] to a joint struct that does not depend on T
type errorWithParameters_commonInterface interface {
	error
	Unwrap() error // Note: May return nil if there is nothing to wrap.
	// getParameter obtains the parameter stored under the key parameterName. Returns the value and whether it was present.
	// If wasPresent == false, returned value is nil. DOES NOT FOLLOW THE ERROR CHAIN.
	getParameter(parameterName string) (value any, wasPresent bool)
	// hasParameter tests if the parameter stored under the key is present. DOES NOT FOLLOW THE ERROR CHAIN and treates markedAsDeleted as a perfectly normal value.
	hasParameter(parameterName string) bool
	// GetAllParams returns a map of all parameters. DOES NOT FOLLOW THE ERROR CHAIN.
	getAllParameters() map[string]any
	// Queries whether the map of included parameters should be shown by Error() if non-empty.
	// Note that when *showing* parameters, we actually follow the whole error chain to get all parameters.
	ShowParametersOnError() bool
	// withShowParametersOnError creates a copy of the error with ShowParametersOnError set as requested.
	// withShowParametersOnError(bool) errorWithParameters_commonInterface // TODO: Return type?

	// isNil checks whether the receiver is a nil pointer of concrete type. These should never occur.
	isNil() bool
}

// TODO: Should we include the non-exported fields?
// We only use this interface for return values;
// Of course, other packagess can always define a sub-interface ED with just error, Unwrap, GetData on their own and assign to it.
// Note that a custom struct implementing ED will not work well together with this package, as none of the free functions would
// currently see the data contained in ED.
// Consequently, the non-exported fields serve no purpose other than preventing users from shooting themselves in the foot.

// ErrorWithParameters[StructType] is an interface extending error.
// Any non-nil error returned in such an interface is guaranteed to contain some additional data sufficient to create an instance of StructType.
//
// Obtaining the additional data can be done via the more general free functions
// GetAllParametersFromError, GetParameterFromError, GetDataFromError,
// but for ErrorWithParameters[StructType], we can also call the GetData member function and
// we are guaranteed that the error actually contains appropriate parameters to create an instance of StructType.
//
// ShowParametersOnError / WithShowParametersOnError query / set whether parameter are shown by Error().
// When set, this particular error's Error() message includes all parameters of itself and its error chain.
// Note that when an ErrorWithParameters is wrapped, this flag does not affect what the wrapping error does (unless it calls Error() on its ancestor)
// and any new additional parameters are not shown (unless the wrapping error also sets this flag, in which case some parameters are shown twice with
// possibly different values if the new values override the old ones)
type ErrorWithParameters[StructType any] interface {
	error
	Unwrap() error       // Note: May return nil if there is nothing to wrap.
	GetData() StructType // Note: e.GetData() Is equivalent to calling GetDataFromError[StructType](e)

	// getParameter obtains the parameter stored under the key parameterName. Returns the value and whether it was present.
	// If wasPresent == false, returned value is nil. DOES NOT FOLLOW THE ERROR CHAIN.
	getParameter(parameterName string) (value any, wasPresent bool)
	// hasParameter tests if the parameter stored under the key is present. DOES NOT FOLLOW THE ERROR CHAIN.
	hasParameter(parameterName string) bool
	// GetAllParams returns a map of all parameters. DOES NOT FOLLOW THE ERROR CHAIN.
	getAllParameters() map[string]any
	// Queries whether the map of included parameters should be shown by Error() if non-empty.
	// Note that when *showing* parameters, we actually follow the whole error chain to get all parameters.
	ShowParametersOnError() bool
	// withShowParametersOnError creates a copy of the error with ShowParametersOnError set as requested.
	WithShowParametersOnError(bool) ErrorWithParameters[StructType] // TODO: Return type?
}

// UnconstrainedErrorWithParameters is the special case of ErrorWithParameters without any data guarantees.
type UnconstrainedErrorWithParameters = ErrorWithParameters[struct{}]

// To delete a parameter from an error, we need to actually place a "This value is deleted"-marker, which is an arbitrary singleton.
// Just removing the value would not work, because the actualy value might be from somewhere down the error chain.

type deletedType struct{}

var markedAsDeleted deletedType // we just want an unique unexported value that compares unequal to everything a library use can create.

// errorPrefix is a prefix added to all (internal) error messages/panics that originate from this package. Does not apply to wrapped errors.
const errorPrefix = "bandersnatch / error handling:"

// WARNING: Parts of the implementations of the structs realizing the interfaces
// internally make use of GetAllParametersFromError.
// Make sure not to create dependency cycles.
// however, getAllParameters does not.

// GetAllParametersFromError returns a map for all parameters stored in the error, including all of err's error chain.
// For err==nil, returns nil. If no error in err's error chain has any data, returns an empty map.
func GetAllParametersFromError(err error) map[string]any {
	if err == nil {
		return nil
	}
	// Find all errors it the error chain which satisfy the ErrorWithParameters_plain interface
	relevantErrors := make([]*errorWithParameters_commonInterface, 0)
	for errChain := err; errChain != nil; errChain = errors.Unwrap(errChain) {
		errChainGood, ok := errChain.(errorWithParameters_commonInterface)
		if ok {
			if errChainGood.isNil() {
				// Users of the package should never be able to trigger this.
				panic(errorPrefix + "internal bug: typed nil error satisfying the ErrorWithParameters interface in error chain detected")
			}
			relevantErrors = append(relevantErrors, &errChainGood)
		}
	}
	// Build up the resulting map by going through the relevant ErrorWithParameters starting from the end of the error chain.
	ret := make(map[string]any)
	for i := len(relevantErrors) - 1; i >= 0; i-- {
		m := (*relevantErrors[i]).getAllParameters()
		if m == nil {
			panic(errorPrefix + "getParams returned nil for error satifying ErrorWithParameters in error chain")
		}
		for key, value := range m {
			ret[key] = value
			if value == markedAsDeleted {
				delete(ret, key)
			}
		}
	}
	return ret
}

// NewErrorWithParameters creates a new ErrorWithParameters wrapping the given baseError, possibly overriding the error message message and adding parameters.
// If overrideMessage == "", the old error message is kept.
// Note: The difference between this and IncludeParametersInError is the message and nil handling:
//
// For baseError == nil and overrideMessage == "", #params > 0, we panic
// For baseError == nil, overrideMessage == "", #params == 0, we return a nil interface
// For baseError == nil, overrideMessage != "", the returned error does not wrap an error.
func NewErrorWithParameters[StructType any](baseError error, overrideMessage string, params ...any) ErrorWithParameters[StructType] {
	if len(params)%2 != 0 {
		panic(errorPrefix + "called NewErrorWithParameters(err, overrideMessage, args...) with an odd number of args. These are supposed to be name-value pairs")
	}
	extraParams := len(params) / 2
	if baseError == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic(errorPrefix + "called NewErrorWithParameters(nil,\"\",argName, arg1, ...)")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}
	ret := errorWithParameters_common{contained_error: baseError, message: overrideMessage, params: make(map[string]any)}
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf(errorPrefix+"called NewErrorWithParams(err, overrideMessage, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		ret.params[s] = params[2*i+1]
	}
	validationError := canMakeStructFromParametersInError[StructType](&ret)
	if validationError != nil {
		panic(validationError)
	}
	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// IncludeParametersInError creates a new error wrapping err with parameter under parameterName set to newParameter.
// As opposed to if err == nil, returns nil
func IncludeParametersInError[StructType any](err error, parameters ...any) ErrorWithParameters[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithParameters[StructType](err, "", parameters...)
}

// Special case for StructType == struct{}

var NewErrorWithParametersUnconstrained func(err error, messageOverride string, params ...any) UnconstrainedErrorWithParameters = NewErrorWithParameters[struct{}]
var IncludeParametersInErrorUnconstrained func(err error, parameters ...any) UnconstrainedErrorWithParameters = IncludeParametersInError[struct{}]

// TODO: global rename after old usage is refactored, intended name NewErrorWithData currently clashes.

// NewErrorWithParametersAsData creates a new ErrorWithParameters wrapping the given error if non-nil, overriding the error message if != "" and adding
// parameters for each visible field of StructType.
//
// For err == nil, overrideMessage == "", #visibleFields of (*data) > 0, this function panics.
// For err == nil, overrideMessage == "", #visibleFields of (*data) ==0, returns nil
// For err == nil, overrideMessage != "", creates a new error that does not wrap an error.
func NewErrorWithParametersAsData[StructType any](err error, overrideMessage string, data *StructType) ErrorWithParameters[StructType] {
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	if err == nil {
		if overrideMessage == "" {
			if len(allStructFields) > 0 {
				panic(errorPrefix + "called NewErrorWithData(nil,\"\",data) with non-empty data")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}
	createdError := errorWithParameters_common{contained_error: err, message: overrideMessage, params: make(map[string]any)}
	fillMapFromStruct(data, &createdError.params)
	return &errorWithParameters_T[StructType]{errorWithParameters_common: createdError}
}

// IncludeDataInError returns a new error with the data provided.
//
// On nil input, returns nil, ignoring the provided data. Use NewErrorWithData for a variant that behaves differently instead.
func IncludeDataInError[StructType any](err error, data *StructType) ErrorWithParameters[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithParametersAsData(err, "", data)
}

// HasParameter checks whether some error in err's error chain contains a parameter keyed by parameterName
// HasParameter(nil, ...) returns false
func HasParameter(err error, parameterName string) bool {
	if err == nil {
		return false
	}
	errWithParams, ok := err.(errorWithParameters_commonInterface)
	if ok {
		if arg, present := errWithParams.getParameter(parameterName); present {
			return arg != markedAsDeleted
		}
	}
	return HasParameter(errors.Unwrap(err), parameterName)
}

// HasData checks whether the error contains enough parameters of correct types to create an instance of StructType.
//
// Note: panics if StructType is malformed. If data is present, but of wrong type, returns false.
func HasData[StructType any](err error) bool {
	return canMakeStructFromParametersInError[StructType](err) == nil
}

// GetParameterFromError returns the value stored under the key parameterName in the first error in err's error chain
// where some entry was found.
// If no entry was found in the error chain or err==nil, returns nil, false.
func GetParameterFromError(err error, parameterName string) (value any, wasPresent bool) {
	if err == nil {
		return nil, false
	}
	errWithParams, ok := err.(errorWithParameters_commonInterface)
	if ok {
		if arg, present := errWithParams.getParameter(parameterName); present {
			if arg == markedAsDeleted {
				return nil, false
			} else {
				return arg, true
			}
		}
	}
	return GetParameterFromError(errors.Unwrap(err), parameterName)
}

// GetDataFromError obtains the parameters contained in err in the form of a struct of type StructType.
//
// If err does not contain enough parameters ( including the case err==nil, #visibleFields(StructType)>0 ), this function panics.
func GetDataFromError[StructType any](err error) (ret StructType) {
	allParams := GetAllParametersFromError(err)
	ret, wrongDataError := makeStructFromMap[StructType](allParams)
	if wrongDataError != nil {
		panic(wrongDataError)
	}
	return
}

// DeleteParameterFromError takes an error and returns a modified copy (wrapping the original) that has the given parameter removed.
// Has no effect (except for copying and wrapping) if the parameter was not present to start with.
// It works even if the input error's parameter is due to something deep in the error chain.
//
// If the input error is nil, returns nil
func DeleteParameterFromError(err error, parameterName string) UnconstrainedErrorWithParameters {
	if err == nil {
		return nil
	}
	return IncludeParametersInErrorUnconstrained(err, parameterName, markedAsDeleted)
}

// Exported for cross-package testing. Will be removed/replaced by callback. Not part of the official interface
var GetDataPanicOnNonExistantKeys = false
