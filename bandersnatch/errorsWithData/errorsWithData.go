// Package errorsWithData defines functionality to add arbitrary parameters to errors in a way that is compatible with error wrapping.
//
// Parameters can be added and retrieved to errors in two flavours: as a map[string]interface{} or as structs.
// We allow both interchangably, identifying a struct{A: x, B: y} with a map {"A":x, "B":y}, i.e.
// the map keys are the field names (this gives some minor restrictions on what struct types are allowed).
// The map/struct interfaces can be mixed-and-matched and when retrieving as a struct, the fields may be a strict subset of the parameters present.
//
// The free functions that are part of the public API operate on errors of plain type [error] and are compatible with error wrapping.
// We (need to) treat errors as (shallowly) immutable objects, so any (shallow) modification to the parameters of an error will create a new one,
// typically wrapping the old one.
//
// Errors are returned either as an interface of type [ErrorWithData_any] or through a generic interface ErrorWithData[StructType].
// the first option corresponds roughly to StructType = struct{}, but is special-cased.
//
// StructType is used to communicate through the type system that certain parameters are present with certain types.
// Notably, all these interfaces extend error and for a StructType= struct{A:type1, B:type2}, non-nil errors of type ErrorWithData[StructType]
// are guaranteed to contain (at least) parameters under keys "A" and "B" of appropriate type.
// For [ErrorWithData_any], we make no such guarantee about what parameters are present.
// Generally speaking, this (and retrievability as structs in general) exists purely as a way to get some partial type-safety.
//
// We recommend adding / retrieving via the struct rather than the map interface for at least some compile-time type-safety.
// When using the map interface, we recommend defining string constants for the map keys.
//
// We assert that any errors that are contained in error chains are either nil *interfaces* or non-nil.
// In particular, no nil error of concrete struct (pointer) type shall ever appear.
//
// We further assume that all errors involved are immutable (and in particular, their associated data is). This is enforced by our own implementation at least for shallow modification.
// In particular, when we have some error e with data m (in map form) with m["Foo"] = "Bar", then the supposed way to "modify" m is by creating
// a new error e2 wrapping e with modified map m2 and replace e by e2.
// The reason is that e2 "inherits" its map from e and if we change e after creating e2, it is unclear whether we should track the changes, leading to confusion.
// We generally ask that for any error e, the output of e.Error() does not change in time.
// The library allows using a plain error as the base of an error chain with parameters added to the wrapping errors.
// When violating immutability for the base error, we make no guarantee at what point(s) in time the base error's Error() method are called.
// Similarly, the actual associated data that is contained in error should not be modified. While we provide no way to modify the data, this also means that care should be taken
// when using slices or pointers as associated data (as the contents of the backing array or the value pointed-to may change). We recommend deep-copying slices.
//
// A second reason for the recommendation to deep-copy slices is that the pattern
//
//	   var [some_number]T s
//	   ...
//	   if <bad thing>{
//			some_slice := make([]some_type, len(s))
//	    	copy(some_slice, s[:])
//	     	return NewErrorWithData_params(..., some_slice)
//	   }
//
// avoids always heap-allocating s even if no error occurs. In general, the parameters passed to create errors with data should be created only in the failing branch;
// doing otherwise may carry a large perfomance penalty (since Go's interfaces essentially break Go's escape analysis).
//
// Restrictions on StructTypes: Adding/retrieving data as structs has the following restrictions on allowed structs:
//   - All non-embedded field names must be exported.
//   - Fields of interface type are allowed.
//   - Embedded struct are also allowed.
//   - Anything else causes a panic.
//
// Shadowed fields:
// When converting to a map, the promoted-field hierarchy get flattened. I.e. embededded fields act in the following way:
// For structs
//
//	type Struct1 struct{Data1 bool; Data2 int}
//	type Struct2 struct{Struct1; Data2 string}
//
// (Note: actually, neither Struct1 or Struct2 actually need to be exported, only their fields)
// after adding data from an instance of type Struct2, we can retrieve parameters (using the map interface) under the keys
// "Data1" (yielding a bool) and "Data2" (yielding a string). There are no keys "Struct1" or "Struct1.Data1", "Struct1.Data2".
// In particular, the shadowed int from Struct1.Data2 is completely ignored when adding data.
// When retrieving data as an instance s of Struct2, s.Struct1.Data2 may be zero-inintialized.
// In particular, roundtrip fails for shadowed fields:
// Creating an error with associated data struct s and retrieving it as a struct s' does NOT guarantee s == s' if there are embedded fields.
// For the map[string]any API, note that we allow (and special-case) nil interface entries (i.e. m["Foo"]=nil); when using the struct API, with struct{Foo *int}, this gets converted into a nil of appropriate type *int.
//
// TODO: REDO!
// Interfaces creating new errors take an overrideMessage string parameter.
// Supplying an empty string will make the function use the default value
// given by the DefaultOverrideMessage constant, which prints the baseError and the parameters map (if non-empty)
// If you really want to use an empty string, use the OverrideByEmptyMessage constant as argument.
//
// This overrideMessage will be used to format the error string of the newly created error with the following rules:
// % is used as a control character, a literal % must be escaped as %% (except when used as a parameter name)
// %m will print the map using fmt.FPrint
// %w will print the parent error
// %!M>0{string} will recursively print string iff the parameters map is non-empty.
// Other %!... are reserved.
// %FMTSTRING{PARAMETER} will look up PARAMETER in the error's map and format it using the fmt package with %FMTSTRING
// FMTSTRING must not contain (even escaped) '%' or '{' or '}'. PARAMETER may (but should not) contain unescaped %.
// The {} are mandatory, an empty FMTSTRING is interpreted as %v
//
// Note that most functions creating an error replace an empty string "" by
// a default error message (containing %w and %m).
package errorsWithData

import (
	"errors"
)

type paramMap = map[string]any // alias to avoid typing

const ConditionNonEmptyMap = "m>0"
const ConditionEmptyMap = "m=0"

/////////////

// Since not even the standard library function specifies whether error wrapping works by reference or copy and we
// do not want users to accidentially modify exported errors that people compare stuff against, we are essentially forced
// to treat errors as immutable objects. This form of immutability is possibly shallow.

// Implementation considerations:
// Any kind of AddParametersToError(existingError error, params...) or possibly AddParametersToError(*error, params...)
// that we create runs afoul of the issue that existing errors do not support this;
// So we either
// a) maintain a separate global registry ([pointer-to-?]error -> parameter map) as aside-lookup table to lookup parameters without touching the existing errors
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
// Doing otherwise is a serious footgut, since the zero value is a nil pointer of concrete type and will be non-nil
// (in the sense that comparing to nil gives false) when assigned to an (e.g. standard error) interface.
// For that reason, we consider the existence of any nil error of concrete type defined here a bug.
// Our API does not allow creating nil pointers of concrete type.

// We use the type system to communicate that certain parameters are guaranteed to be present on non-nil errors.
// (This is done so we get a least some compile-time(!) checks on the side creating the error for this, as
// error handling is prone to bad testing coverage)

// ErrorWithData_any is an interface extending error to also contain arbitrary parameters
// in the form of a map[string]any
// Obtaining the additional data can and should be done via the more general free functions
// [GetAllParametersFromError], [GetParameterFromError], [GetDataFromError], etc, as these work for arbitrary errors.
type ErrorWithData_any interface {
	error // i.e. provides an Error() string method
	// GetParameter obtains the value stored under the given parameterName and whether it was present. Returns (nil, false) if not.
	Error_interpolate(params map[string]any) string // extended version of Error() string that additionally takes a map of parameters. This is required to make any $foo (as opposed to %foo) formatting verbs work.
	GetParameter(parameterName string) (value any, wasPresent bool)
	// HasParameter returns whether parameterName is a key of the parameter map.
	HasParameter(parameterName string) bool
	// GetData_map returns a shallow copy of the parameter map.
	GetData_map() map[string]any
	// typically also has Unwrap() error -- all errors created by this package do.
}

// ErrorWithData[StructType] is an interface extending [ErrorWithData_any].
// Any non-nil error returned in such an interface is guaranteed to contain some additional data sufficient to create an instance of StructType.
//
// Obtaining the additional data can be done via the more general free functions
// GetAllParametersFromError, GetParameterFromError, GetDataFromError,
// but for ErrorWithData[StructType], we can also call the GetData member function and
// we are guaranteed that the error actually contains appropriate parameters to create an instance of StructType.
type ErrorWithData[StructType any] interface {
	ErrorWithData_any
	GetData_struct() StructType // Note: e.GetData() Is equivalent to calling GetDataFromError[StructType](e)
}

// unconstrainedErrorWithGuaranteedParameters is the special case of ErrorWithParameters without any data guarantees.
// It's functionally equivalent to [ErrorWithData_any]
type unconstrainedErrorWithGuaranteedParameters = ErrorWithData[struct{}]

// ErrorPrefix is a prefix added to all *internal* error messages/panics (such as invalid formatting strings) that originate from this package.
const ErrorPrefix = "bandersnatch / error handling: "

// GetData_map returns a map for all parameters stored in the error, including all of err's error chain.
// For err==nil or if no error in err's error chain has any data, returns an empty map.
//
// Note that the returned map is a (shallow) copy, so the caller may modify it without affecting the error.
func GetData_map(err error) map[string]any {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithData_any); ok {
			return errChainGood.GetData_map()
		}
	}
	return make(map[string]any)
}

// HasParameter checks whether some error in err's error chain contains a parameter keyed by parameterName
// HasParameter(nil, <anything>) returns false
func HasParameter(err error, parameterName string) bool {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithData_any); ok {
			return errChainGood.HasParameter(parameterName)
		}
	}
	return false
}

// HasData checks whether the error contains enough parameters of correct types to create an instance of StructType.
//
// Note: This function panics if StructType is malformed for this purpose (e.g containing non-exported fields).
// If data is present, but of wrong type, returns false.
func HasData[StructType any](err error) bool {
	return canMakeStructFromParametersInError[StructType](err) == nil
}

// GetParameter returns the value stored under the key parameterName in the first error in err's error chain
// where some entry was found.
// If no entry was found in the error chain or err==nil, returns (nil, false).
func GetParameter(err error, parameterName string) (value any, wasPresent bool) {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithData_any); ok {
			return errChainGood.GetParameter(parameterName)
		}
	}
	return nil, false
}

// GetData_struct obtains the parameters contained in err in the form of a struct of type StructType.
//
// If err does not contain enough parameters, this function panics.
// NOTE: If StructType is empty after flattening embedded fields, the function does not panic even if err == nil.
func GetData_struct[StructType any](err error) (ret StructType) {
	allParams := GetData_map(err)
	ret, wrongDataError := makeStructFromMap[StructType](allParams)
	if wrongDataError != nil {
		panic(wrongDataError)
	}
	return
}

// Unused, as far as I can see.
// Exported for cross-package testing. Will be removed/replaced by callback. Not part of the official interface
// var GetDataPanicOnNonExistentKeys = false
