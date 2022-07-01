package errorsWithData

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file defines functionality to add arbitrary parameters to errors in a way that is compatible with error wrapping.
//
// Parameters can be added and retrieved to errors in two flavours: as a map[string]interface{} or a structs.
// We allow both interchangably, identifying a struct{A: x, B: y} with a map {"A":x, "B":y}, i.e.
// the map keys are the field names (this gives some minor restrictions on what struct types are allowed).
// The map/struct interfaces can be mixed-and-matched and when retrieving a struct only a subset of the parameters might be actually used.
// Naming-wise, the API (arbitrarily) uses the term "parameters" for the map interface and the term "data" for the struct interface.
//
// The publicly-facing API operates on errors of plain error type and is compatible with error wrapping.
// We (need to) treat errors as immutable objects, so any modification to the parameters of an error will create a new one,
// typically wrapping the old one.
//
// Errors are returned in a parameterized interface ErrorWithGuaranteedParameters[StructType],
// where StructType is a struct type or as an interface ErrorWithParameters.
// These interfaces extends error and for a struct type struct{A:type1, B:type2} non-nil errors of type ErrorWithGuaranteedParameters[StructType]
// are guaranteed to contain (at least) parameters under keys "A" and "B" of appropriate type.
// For ErrorWithParameters, we make no such guarantee.
// Generally speaking, this (and retrievability as structs) exists purely as a way to get some partial type-safety.
//
// We recommend adding / retrieving via the struct rather than the map interface for at least some compile-time type-safety.
// When using the map interface, we recommend defining string constants for the map keys.
//
// We assert that any errors that are contained in error chains are either nil *interfaces* or non-nil.
// In particular, no nil error of concrete struct (pointer) type shall ever appear.
//
// We further assume that all errors involved are immutable. This is satisfied by our own implementation,
// but if *T is a custom implementation of ErrorWithParameters and t has type T, then after
// t2 := NewErrorWithParameters(&t, "", ...), we make no guarantees what happens with t2's parameters
// if t's parameters are changed afterwards.
//
// Restrictions on StructTypes: Adding/retrieving data as structs has the following restrictions on allowed structs:
// - All field names must be exported.
// - Fields of interface type are allowed.
// - Embedded struct are also allowed.
// - Anything else causes a panic.
// Embededded fields act in the following way:
// For structs
// type Struct1 struct{Data1 bool; Data2 int}
// type Struct2 struct{Struct1; Data2 string}  (Note: actually, neither Struct1 or Struct2 actually need to be exported, only their fields)
//
// after adding data from an instance of type Struct2, we can retrieve parameters (using the map interface) under the keys
// "Data1" (yielding a bool) and "Data2" (yielding a string). There are no keys "Struct1" or "Struct1.Data1", "Struct1.Data2".
// In particular, the shadowed int from Struct1.Data2 is completely ignored when adding data.
// When retrieving data as an instance s of Struct2, s.Struct1.Data2 is zero-inintialized.
// In particular, roundtrip fails for shadowed fields.
//
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

/////////////

// Since not even the standard library function specifies whether error wrapping works by reference or copy and we
// do not want users to accidentially modify exported errors that people compare stuff against, we are essentially forced
// to treat errors as immutable objects. This form of immutability is possibly shallow.

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

// ErrorWithParameters is an interface extending error to also contain arbitrary parameters
// in the form of a map[string]any
// Obtaining the additional data can and should be done via the more general free functions
// GetAllParametersFromError, GetParameterFromError, GetDataFromError, etc.
type ErrorWithParameters interface {
	error
	// GetParameter obtains the value stored under the given parameterName and whether it was present. Returns nil, false if not.
	GetParameter(parameterName string) (value any, wasPresent bool)
	// HasParameter returns whether parameterName is a key of the parameter map.
	HasParameter(parameterName string) bool
	// GetAllParameters returns *A SHALLOW COPY OF* the parameter map.
	GetAllParameters() map[string]any
	// typically also has Unwrap() error -- all errors created by this package do.
}

// ErrorWithGuaranteedParameters[StructType] is an interface extending ErrorWithParameters.
// Any non-nil error returned in such an interface is guaranteed to contain some additional data sufficient to create an instance of StructType.
//
// Obtaining the additional data can be done via the more general free functions
// GetAllParametersFromError, GetParameterFromError, GetDataFromError,
// but for ErrorWithGuaranteedParameters[StructType], we can also call the GetData member function and
// we are guaranteed that the error actually contains appropriate parameters to create an instance of StructType.
type ErrorWithGuaranteedParameters[StructType any] interface {
	ErrorWithParameters
	GetData() StructType // Note: e.GetData() Is equivalent to calling GetDataFromError[StructType](e)
}

// unconstrainedErrorWithGuaranteedParameters is the special case of ErrorWithParameters without any data guarantees.
// It's functionally equivalent to an ErrorWithParameters
type unconstrainedErrorWithGuaranteedParameters = ErrorWithGuaranteedParameters[struct{}]

// errorPrefix is a prefix added to all (internal) error messages/panics that originate from this package. Does not apply to wrapped errors.
const errorPrefix = "bandersnatch / error handling:"

// GetAllParametersFromError returns a map for all parameters stored in the error, including all of err's error chain.
// For err==nil or if no error in err's error chain has any data, returns an empty map.
func GetAllParametersFromError(err error) map[string]any {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithParameters); ok {
			return errChainGood.GetAllParameters()
		}
	}
	return make(map[string]any)
}

// NewErrorWithGuaranteedParameters creates a new ErrorWithParameters wrapping the given baseError,
// possibly overriding the error message message and adding parameters.
// If overrideMessage == "", DefaultOverrideMessage is used (except if baseError == nil).
// Note: The only difference between this and IncludeParametersInError is the message and nil handling:
//
// For baseError == nil and overrideMessage == "", #params > 0, we panic
// For baseError == nil, overrideMessage == "", #params == 0, we return a nil interface
func NewErrorWithGuaranteedParameters[StructType any](baseError error, overrideMessage string, params ...any) ErrorWithGuaranteedParameters[StructType] {
	// make some validity checks to give meaningful error messages.
	// Impressive: go - staticcheck actually recognizes this patterns and has my IDE complain about violations!
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

	// create a wrapper, copying all parameters from baseError
	ret := makeErrorWithParametersCommon(baseError, overrideMessage)

	// add new parameters to it
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf(errorPrefix+"called NewErrorWithParams(err, overrideMessage, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		ret.params[s] = params[2*i+1]
	}

	// Check whether the promise of being able to construct an instance of StructType is satisfied.
	validationError := canMakeStructFromParametersInError[StructType](&ret)
	if validationError != nil {
		panic(validationError)
	}

	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// NewErrorWithGuaranteedParametersFromMap has the same meaning as NewErrorWithGuaranteedParameters, but the parameters are passed as a map rather than string, any - pairs.
func NewErrorWithGuaranteedParametersFromMap[StructType any](baseError error, overrideMessage string, params map[string]any) ErrorWithGuaranteedParameters[StructType] {
	extraParams := len(params) // 0 for nil
	if baseError == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic(errorPrefix + "called NewErrorWithParametersMap(nil,\"\",actualParams_map)")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}

	ret := makeErrorWithParametersCommon(baseError, overrideMessage)

	for key, value := range params {
		ret.params[key] = value
	}
	validationError := canMakeStructFromParametersInError[StructType](&ret)
	if validationError != nil {
		panic(validationError)
	}
	return &errorWithParameters_T[StructType]{errorWithParameters_common: ret}
}

// IncludeGuaranteedParametersInError creates a new error wrapping baseError with additional parameters set.
// This is identical to NewErrorWithGuaranteedParameters, except that it always used the default overrideMessage
// and for the err==nil case:
// If err == nil, returns nil
func IncludeGuaranteedParametersInError[StructType any](baseError error, parameters ...any) ErrorWithGuaranteedParameters[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithGuaranteedParameters[StructType](baseError, "", parameters...)
}

// IncludeGuaranteedParametersInErrorFromMap is identical to IncludeGuaranteedParametersInError, except it
// takes parameters as a map[string]any rather than variadic string, any - pairs.
func IncludeGuaranteedParametersInErrorFromMap[StructType any](err error, parameters map[string]any) ErrorWithGuaranteedParameters[StructType] {
	if err == nil {
		return nil
	}
	return NewErrorWithGuaranteedParametersFromMap[StructType](err, "", parameters)
}


// NewErrorWithParameters is identical to NewErrorWithGuaranteedParameters except for the guarantee about containing data.
func NewErrorWithParameters(baseError error, overrideMessage string, parameters ...any) ErrorWithParameters {
	return NewErrorWithGuaranteedParameters[struct{}](baseError, overrideMessage, parameters...)
}

// IncludeParametersInError is identical to IncludeGuaranteedParametersInError except for the guarantee about containing data.
func IncludeParametersInError(baseError error, parameters ...any) ErrorWithParameters {
	return IncludeGuaranteedParametersInError[struct{}](baseError, parameters...)
}

// NewErrorWithParametersFromMap is identical to NewErrorWithGuaranteedParametersFromMap except for the guarantee about containing data.
func NewErrorWithParametersFromMap(baseError error, overrideMessage string, parameters map[string]any) ErrorWithParameters {
	return NewErrorWithGuaranteedParametersFromMap[struct{}](baseError, overrideMessage, parameters)
}

// IncludeParametersInErrorsFromMap is identical to IncludeGuaranteedParametersInErrorFromMap except for the guaranteed about containing data.
func IncludeParametersInErrorsFromMap(baseError error, parameters map[string]any) ErrorWithParameters {
	return IncludeGuaranteedParametersInErrorFromMap[struct{}](baseError, parameters)
}

// TODO: global rename after old usage is refactored, intended name NewErrorWithData currently clashes.

// NewErrorWithParametersFromData creates a new ErrorWithGuaranteedParameters wrapping the given baseError if non-nil.
// overrideMessage is used to create the new error message, where an empty string is
// interpreted as a default error message (containing %w and %m).
// Parameters are added for each visible field of StructType.
//
// For baseError == nil, overrideMessage == "", #visibleFields of (*data) > 0, this function panics.
// For baseError == nil, overrideMessage == "", #visibleFields of (*data) ==0, returns nil
// For baseError == nil, overrideMessage != "", creates a new error that does not wrap an error.
func NewErrorWithParametersFromData[StructType any](baseError error, overrideMessage string, data *StructType) ErrorWithGuaranteedParameters[StructType] {
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	if baseError == nil {
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

	createdError := makeErrorWithParametersCommon(baseError, overrideMessage)

	fillMapFromStruct(data, &createdError.params)
	return &errorWithParameters_T[StructType]{errorWithParameters_common: createdError}
}

// IncludeDataInError returns a new error with the data provided.
// This is identical to NewErrorWithParametersFromData except for the baseError == nil case.
//
// On nil input for baseError, returns nil, ignoring the provided data.
func IncludeDataInError[StructType any](baseError error, data *StructType) ErrorWithGuaranteedParameters[StructType] {
	if baseError == nil {
		return nil
	}
	return NewErrorWithParametersFromData(baseError, "", data)
}

// HasParameter checks whether some error in err's error chain contains a parameter keyed by parameterName
// HasParameter(nil, <anything>) returns false
func HasParameter(err error, parameterName string) bool {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithParameters); ok {
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

// GetParameterFromError returns the value stored under the key parameterName in the first error in err's error chain
// where some entry was found.
// If no entry was found in the error chain or err==nil, returns nil, false.
func GetParameterFromError(err error, parameterName string) (value any, wasPresent bool) {
	for errorChain := err; errorChain != nil; errorChain = errors.Unwrap(errorChain) {
		if errChainGood, ok := errorChain.(ErrorWithParameters); ok {
			return errChainGood.GetParameter(parameterName)
		}
	}
	return nil, false
}

// GetDataFromError obtains the parameters contained in err in the form of a struct of type StructType.
//
// If err does not contain enough parameters, this function panics.
// NOTE: If StructType is empty with 0 visible fields, the function does not panic, even if err == nil.
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
func DeleteParameterFromError(err error, parameterName string) unconstrainedErrorWithGuaranteedParameters {
	if err == nil {
		return nil
	}
	ret := makeErrorWithParametersCommon(err, "")
	delete(ret.params, parameterName)
	return &errorWithParameters_T[struct{}]{errorWithParameters_common: ret}
}

// Exported for cross-package testing. Will be removed/replaced by callback. Not part of the official interface
var GetDataPanicOnNonExistentKeys = false

// Providing this value as overrideMessage for creating an ErrorWithParameters will create an actual empty string
// (Giving a literal empty string will instead default to DefaultOverrideMessage)
const (
	OverrideByEmptyMessage = "%Empty"
	DefaultOverrideMessage = "%w%!M>0{ Included Parameters: %m}"
)
const nonEmptyMapFormatString = "!M>0" // without %

// FormatError will print/interpolate the given format string using the parameters map if output == true
// For output == false, it will only do some validity parsing checks.
// This is done in one function in order to de-duplicate code.
//
// The format is as follows: %FMT{arg} is printed like fmt.Printf(%FMT, parameters[arg])
// %% is used to escape literal %
// %m is used to print the parameters map itself
// %$nonEmptyMapFormatString{STR} will evaluate STR if len(parameters) > 0, nothing otherwise.
// %w will print baseError.Error()
func formatError(formatString string, parameters map[string]any, baseError error, output bool) (returned_string string, err error) {
	if !utf8.ValidString(formatString) {
		panic(errorPrefix + "formatString not a valid UTF-8 string")
	}

	// We build up the returned string piece-by-piece by writing to ret.
	// This avoids some allocations & copying
	var ret strings.Builder
	if output {
		defer func() {
			returned_string = ret.String()
			if err != nil {
				returned_string += fmt.Sprintf("<error when printing error: %v>", err)
			}
		}()
	}

	var suffix string = formatString // holds the remaining yet-unprocessed part of the input.

	for { // annoying to write as a "usual" for loop, because both init and iteration would need to consist of 2 lines (due to ret.WriteString(prefix)).
		var prefix string
		var found bool
		// No :=, because that would create a local temporary suffix variable, shadowing the one from the outer scope.

		// look for first % appearing and split according to that.
		prefix, suffix, found = strings.Cut(suffix, `%`)

		// everything before the first % can just go to output; for the rest (now in suffix, we need to actually do some parsing)
		if output {
			ret.WriteString(prefix)
		}

		if !found {
			// We are guaranteed suffix == "" at this point and we are done.
			return
		}

		// Trailing % in format string, not part of %% - escape.
		if len(suffix) == 0 {
			err = fmt.Errorf("invalid terminating \"%%\"")
			return
		}

		// handle %c - cases where c is a single rune not followed by {param}

		if output {
			switch suffix[0] {
			case '%': // Handle %% - escape for literal "%"
				suffix = suffix[1:]
				ret.WriteRune('%')
				continue
			case 'm': // Handle %m - print map of all parameters
				suffix = suffix[1:]
				_, err = fmt.Fprint(&ret, parameters)
				if err != nil {
					return
				}
				continue
			case 'w': // handle %w - print wrapped error
				suffix = suffix[1:]
				ret.WriteString(baseError.Error())
				continue
			// case '!' handled later
			default:
				// Do nothing
			}
		} else {
			switch suffix[0] {
			case '%', 'm', 'w':
				suffix = suffix[1:]
				continue
			}
		}

		// Everything else must be of the form %fmtString{parameterName}remainingString

		// Get fmtString
		var fmtString string
		fmtString, suffix, found = strings.Cut(suffix, "{")
		if !found {
			err = fmt.Errorf("remaining error override string %%%v, which is missing mandatory {...}-brackets", fmtString)
			return
		}

		// handle case where fmtString contains another %. This is not allowed (probably caused by some missing '{' ) and would cause strange errors later when we pass
		// %fmtString to a ftm.[*]Printf - variant.
		if strings.ContainsRune(fmtString, '%') {
			err = fmt.Errorf("invalid format string: Must be %%fmtString{parameterName}, parsed %v for fmtString, which contains another %%", fmtString)
			return
		}

		// Get parameterName
		var parameterName string
		parameterName, suffix, found = strings.Cut(suffix, "}")
		if !found {
			err = fmt.Errorf("error override message contained format string %v, which is missing terminating \"}\">", parameterName)
			return
		}

		// If we don't actually output anything, we have no way of checking validity.
		// (due to the possibility of custom format verbs)
		if !output {
			continue
		}

		// handle special case "!"
		if fmtString == nonEmptyMapFormatString {
			if len(parameters) > 0 {
				var recursiveParseResult string
				recursiveParseResult, err = formatError(parameterName, parameters, baseError, output)
				if output {
					ret.WriteString(recursiveParseResult)
				}
				if err != nil {
					return
				}
			}
			continue
		}

		// default to %v
		if fmtString == "" {
			fmtString = "v"
		}

		// actually print the parameter
		_, err = fmt.Fprintf(&ret, "%"+fmtString, parameters[parameterName])
		if err != nil {
			return
		}

		continue // redundant
	}

}
