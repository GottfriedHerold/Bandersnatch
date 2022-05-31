package bandersnatchErrors

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

// PlainErrorWithParameters is an error (wrapper) that wraps an arbitrary error, but extends the error interface by
// allowing to embed arbitrary data.
// Data is embedded as a strings -> interface{} map.
//
// It is recommended to define string constant for the keys.
//
// Note that an API built around this that is more likely to fit users' needs is also available by free functions
// GetDataFromError, AddDataToError, HasParameter, GetAllDataFromError.
// These free functions work on plain errors (by type-asserting to PlainErrorWithParameters internally)
// and for wrapped errors actually traverse the error chain.
//
// The methods GetParameter, HasParameter, AddData, DeleteData, GetAllParams with PlainErrorWithParameters receivers do NOT traverse the error chain.
type PlainErrorWithParameters interface {
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
	withShowParametersOnError(bool) PlainErrorWithParameters // TODO: Return type?
}

type ErrorWithParameters[T any] interface {
	PlainErrorWithParameters
	Get() T
}
