package bandersnatchErrors

import (
	"errors"
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// This file defines functionality to add arbitrary paramters to errors in a way that is compatible with error wrapping.
//
// NOTE: Since not even the standard library function specifies whether error wrapping works by reference or copy and we
// do not want users to accidentially modify exported errors that people compare stuff against, we are essentially forced
// to treat errors as immutable objects. This form of immutability is possibly shallow.

// By the above, any kind of AddParametersToError(existingError error, params...) or possibly AddParametersToError(*error, params...)
// that we create runs afoul of the issue that existing errors do not support this;
// So we either a) maintain a separate global registry ([pointer-to-???]error -> parameter map) as aside-lookup table to lookup parameters without touching the existing errors
// or b) we create new wrappers (of a new type) that wrap the existing errors and support the interface.
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
// This is because our wrappers needs to be ALWAYS returned as interfaces, never as a concrete type.
// Otherwise, there is a serious footgut, since the zero value is a nil pointers of concrete type:
// If such an error is assigned to an (e.g. error) interface, comparison with nil will give false.
// We consider the existence of any nil error of concrete type defined here a bug.

// We use the type system to communicate that certain parameters are guaranteed to be present on non-nil errors.
// (Preferably we also want to get compile-time(!) checks on the side creating the error for this, as
// error handling is prone to bad testing coverage)

// errorWithParameters_commonInterface is an error (wrapper) that wraps an arbitrary error, but extends the error interface by
// allowing to embed arbitrary data.
// Data is embedded as a strings -> interface{} map.
//
// It is recommended to define string constant for the keys.
//
// Note that an API built around this that is more likely to fit users' needs is also available by free functions
// GetDataFromError, AddDataToError, HasParameter, GetAllDataFromError.
// These free functions work on plain errors (by type-asserting to errorWithParameters_commonInterface internally)
// and for wrapped errors actually traverse the error chain.
//
// The methods GetParameter, HasParameter, AddData, DeleteData, GetAllParams with errorWithParameters_commonInterface receivers do NOT traverse the error chain.
type errorWithParameters_commonInterface interface {
	error
	Unwrap() error // Note: May return nil if there is nothing to wrap.
	// getParameter obtains the parameter stored under the key parameterName. Returns the value and whether it was present.
	// If wasPresent == false, returned value is nil. DOES NOT FOLLOW THE ERROR CHAIN.
	getParameter(parameterName string) (value any, wasPresent bool)
	// hasParameter tests if the parameter stored under the key is present. DOES NOT FOLLOW THE ERROR CHAIN.
	hasParameter(parameterName string) bool
	// GetAllParams returns a map of all parameters. DOES NOT FOLLOW THE ERROR CHAIN.
	getAllParameters() map[string]any
	// Queries whether the map of included parameters should be shown by Error() if non-empty.
	// Note that when *showing* parameters, we actually follow the whole error chain to get all parameters.
	showParametersOnError() bool
	// withShowParametersOnError creates a copy of the error with ShowParametersOnError set as requested.
	withShowParametersOnError(bool) errorWithParameters_commonInterface // TODO: Return type?
}

type ErrorWithParameters[T any] interface {
	error
	Unwrap() error // Note: May return nil if there is nothing to wrap.
	// getParameter obtains the parameter stored under the key parameterName. Returns the value and whether it was present.
	// If wasPresent == false, returned value is nil. DOES NOT FOLLOW THE ERROR CHAIN.
	getParameter(parameterName string) (value any, wasPresent bool)
	// hasParameter tests if the parameter stored under the key is present. DOES NOT FOLLOW THE ERROR CHAIN.
	hasParameter(parameterName string) bool
	// GetAllParams returns a map of all parameters. DOES NOT FOLLOW THE ERROR CHAIN.
	getAllParameters() map[string]any
	// Queries whether the map of included parameters should be shown by Error() if non-empty.
	// Note that when *showing* parameters, we actually follow the whole error chain to get all parameters.
	showParametersOnError() bool
	// withShowParametersOnError creates a copy of the error with ShowParametersOnError set as requested.
	withShowParametersOnError(bool) errorWithParameters_commonInterface // TODO: Return type?
	Get() T
}

type UnconstrainedErrorWithParameters = ErrorWithParameters[struct{}]

type deletedType struct{}

var markedAsDeleted deletedType // we just want an unique unexported value that compares unequal to everything a library use can create.

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
			if errChainGood == nil {
				panic("bandersnatch / error handling: typed nil error satisfying the ErrorWithParameters interface in error chain detected")
			}
			relevantErrors = append(relevantErrors, &errChainGood)
		}
	}
	// Build up the resulting map by going through the relevant ErrorWithParameters starting from the end of the error chain.
	ret := make(map[string]any)
	for i := len(relevantErrors) - 1; i >= 0; i-- {
		m := (*relevantErrors[i]).getAllParameters()
		if m == nil {
			panic("bandersnach / error handling: getParams returned nil for error satifying ErrorWithParameters in error chain")
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

// NewErrorWithParameters creates a new errorWithParams (wrapped in an interface) wrapping the given error, override message and (optional) parameters.
// Note: The difference between this and IncludeParametersInError is the message and nil handling:
//
// For err == nil and overrideMessage == "", #params > 0, we panic
// For err == nil, overrideMessage == "", #params == 0, we return a nil interface
// For err == nil, overrideMessage != "", the returned error behaves like a new error that does not wrap an error.
func NewErrorWithParameters[T any](err error, overrideMessage string, params ...any) ErrorWithParameters[T] {
	if len(params)%2 != 0 {
		panic("bandersnatch / error handling: called NewErrorWithParameters(err, overrideMessage, args...) with an odd number of args. These are supposed to be name-value pairs")
	}
	extraParams := len(params) / 2
	if err == nil {
		if overrideMessage == "" {
			if extraParams > 0 {
				panic("bandersnatch / error handling: called NewErrorWithParameters(nil,\"\",argName, arg1, ...)")
			}
			return nil
		}
		// If we get here, err==nil, overrideMessage != "".
		// If we just proceed, the returned error will have contained_error == nil in this case.
		// This is actually fine.
	}
	ret := errorWithParameters_common{contained_error: err, message: overrideMessage, params: make(map[string]any)}
	for i := 0; i < extraParams; i++ {
		s, ok := params[2*i].(string)
		if !ok {
			panic(fmt.Errorf("bandersnatch / error handling: called NewErrorWithParams(err, overrideMessage, args... with invalid parameters. args are supposed to come in (string-any) pairs, but got a non-string in position %v", 2*i))
		}
		ret.params[s] = params[2*i+1]
	}
	validationError := validateErrorContainsData[T](&ret)
	if validationError != nil {
		panic(validationError)
	}
	return &errorWithParameters_T[T]{errorWithParameters_common: ret}
}

// IncludeParametersInError creates a new error wrapping err with parameter under parameterName set to newParameter.
// As opposed to if err == nil, returns nil
func IncludeParametersInError[T any](err error, parameters ...any) ErrorWithParameters[T] {
	if err == nil {
		return nil
	}
	return NewErrorWithParameters[T](err, "", parameters...)
}

var NewErrorWithParametersUnconstrained func(err error, messageOverride string, params ...any) UnconstrainedErrorWithParameters = NewErrorWithParameters[struct{}]
var IncludeParametersInErrorUnconstrained func(err error, parameters ...any) UnconstrainedErrorWithParameters = IncludeParametersInError[struct{}]

// TODO: global Rename, name currently clashes.

func NewErrorWithParametersAsData[StructType any](err error, overrideMessage string, data *StructType) ErrorWithParameters[StructType] {
	reflectedStructType := utils.TypeOfType[StructType]()
	allStructFields := getStructMapConversionLookup(reflectedStructType)
	if err == nil {
		if overrideMessage == "" {
			if len(allStructFields) > 0 {
				panic("bandersnatch / error handling: called NewErrorWithData(nil,\"\",data) with non-empty data")
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

func GetDataFromError[StructType any](err error) (ret StructType) {
	allParams := GetAllParametersFromError(err)
	ret, wrongDataError := makeStructFromMap[StructType](allParams)
	if wrongDataError != nil {
		panic(wrongDataError)
	}
	return
}
