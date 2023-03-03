// errorsWithData is a package to add arbitrary parameters to errors in a way that is compatible with error wrapping.
//
// Parameters can be added and retrieved to errors in two flavours: as a paramMap (a type alias to map[string]any) or as structs.
// We allow both interchangably, identifying a struct{A: x, B: y} with a map[string]any{"A":x, "B":y}.
// The map keys are the field names (this gives some minor restrictions on what struct types are allowed).
// The map/struct interfaces can be mixed-and-matched and when retrieving as a struct, the fields may be a strict subset of the parameters present.
//
// The free functions that are part of the public API operate on errors of plain type error and are compatible with error wrapping.
// We (need to) treat errors as (shallowly) immutable objects, so any (shallow) modification to the parameters of an error will create a new one,
// typically wrapping the old one.
//
// Errors are returned either as an interface of type [ErrorWithData_any] or through a generic interface ErrorWithData[StructType].
// the first option corresponds roughly to StructType = struct{}, but is special-cased.
//
// StructType is used to communicate via the type system that certain parameters are present with certain types.
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
// We further assume that all errors involved are immutable (and in particular, their associated data is).
// This is enforced by our own implementation at least for shallow modification.
// In particular, when we have some error e with data m (in map form) with m["Foo"] = "Bar", then the supposed way to "modify" m is by creating
// a new error e2 wrapping e with modified map m2 and then work with e2.
// The reason is that e2 "inherits" its map from e and if we change e after creating e2, it is unclear whether we should track the changes or not, leading to confusion.
//
// The library allows using a plain error as the base of an error chain; the wrapping error then has parameters and satisfies [ErrorWithData_any].
// The general semantics is that we associate to *every* error an immutable parameter map, where error wrapping defaults to copying the map.
// (This default behaviour for errors outside our package means that for errors not satisfying [ErrorWithData_any], we follow the error chain until we hit nil or find an error that satisfied [ErrorWithData_any])
//
// We generally ask that for any error e in the involved error chains, the output of e.Error() does not change over time.
// The reason is that for the base error, we make no guarantee at what point(s) in time the base error's Error() method are called or the map is retrieved.
// Similarly, the actual associated data that is contained in error should not be modified. While we provide no way to modify the data (we only return copies), this also means that care should be taken
// when using slices or pointers as associated data (as the contents of the backing array or the value pointed-to may change, potentially affecting the output of Error() ).
// We recommend deep-copying slices.
//
// A second reason for the recommendation to deep-copy slices is that the pattern
//
//	    var [some_constant_number]T s
//		   ...
//		   if something_bad_has_happened{
//				some_slice := make([]some_type, len(s))
//		    	copy(some_slice, s[:])
//		     	return NewErrorWithData_params(..., some_slice)
//		   }
//
// is often *faster* than
//
//	    var [some_constant_number]T s
//		   ...
//		   if something_bad_has_happened{
//				return NewErrorWithData_params(..., s[:])
//		   }
//
// The reason is that in the latter case, escape analyis will likely fail and s will become heap-allocated, causing significant overhead.
// This cost needs to be paid even if no error occurs. In the former case, we additionally make an extra copy and still allocate on the heap, but only if an actual error occurs.
// If errors are rare, this is much better.
// In general, the parameters passed to create errors with data should be created only in the failing branch;
// doing otherwise may carry a large performance penalty (since Go's interfaces essentially break Go's escape analysis) due to allocation and garbage collection.
//
// Restrictions on StructTypes: Adding/retrieving data as structs has the following restrictions on allowed structs:
//   - All non-embedded field names must be exported.
//   - Fields of interface type are allowed.
//   - Embedded structs are also allowed.
//
// Anything else causes a panic.
//
// The map API does not have such a restriction and works with arbitrary keys, but some functionality is limited, particularly interpolation strings.
// For that reason it is recommended to only use keys that satisfy [IsExportedIdentifier].
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
// When retrieving data as an instance s of Struct2, s.Struct1.Data2 may or may not be zero-inintialized.
// In particular, roundtrip fails for shadowed fields:
// Creating an error with associated data struct s and retrieving it as a struct s' does NOT guarantee s == s' if there are shadowed embedded fields.
// For the map[string]any API, note that we allow (and special-case) nil interface entries (i.e. m["Foo"]=nil ); when using the struct API, with struct{Foo *int}, this gets converted into a nil of appropriate type *int.
//
// To create errors, we provide functions [NewErrorWithData_params], [NewErrorWithData_map], [NewErrorWithData_struct], [NewErrorWithData_any_params], [NewErrorWithData_any_map].
// These functions only differ in whether they return an [ErrorWithData] or [ErrorWithData_any] and how the data is passed.
// Each of these takes a base error (possibly nil) that the new error should wrap, an interpolation string used to create an error message and newly added parameters.
// The newly created error wraps the base error, inherits its data and add some of its own.
//
// Interpolation strings:
//
// The main power of this package is in the ability to refer to the parameters' values in the error message, e.g.
//
//	err := NewErrorWithData_params(nil, "Something bad happended, the value of Foo is ${Foo}.", "Foo", 5)
//	fmt.Println(err)
//
// will print "Something bad happened, the value of Foo is 5." (without the quotation marks)
//
// The language for interpolation strings is as follows:
//   - literal `%`, `$`, `{`, `}` and `\` have to be escaped as \%, \$, \{, \} and \\. Alternatively, %% also works for `%`. The backslash itself has no meaning beyond escaping and we recommend using `raw string`-syntax to avoid having to double-escape.
//   - %w and $w insert the error message of the wrapped error (with special behaviour for $w).
//   - %FormatVerb{VariableName} and $FormatVerb{VariableName} read the value of the associated data under the key VariableName and formats it via the [fmt] package with fmt.Printf("%FormatVerb", value).
//     An empty FormatVerb defaults to v. FormatVerb must not start with w or !.
//   - VariableName must either be an exported identifier or one of the special strings 'm', 'map', 'parameters', 'params'. For these we print all parameters as a map[string]any.
//   - %!Condition{Sub-InterpolationString} and $!Condition{Sub-InterpolationString} conditionally evaluate Sub-InterpolationString according to our grammar. We currently support the conditions
//     "m=0" and "m>0" (without the quotation marks). These conditions mean that the parameter map is empty or non-empty, respectively.
//   - The difference between $ and % is the following: % always refers to the parameters stored in the error itself to look up values or evalate conditions. %w just calls a wrapped error's Error() method.
//     By contrast, $ allows passing parameters through an error chain: If errFinal wraps errBase and errFinal's interpolation string contains a "$w", then
//     this does not call errBase's Error() string, but rather errBase.Error_interpolate(passed_params) where passed_params are errFinal's parameters (or those of another wrapping error calling via $w).
//     Error_Interpolate() will evaluate all $ with passed_params rather than the error's own parameters. It still uses its own for %.
//     Of course, this requires extra support from errBase beyond the error interface (notably errBase must satisfy the [ErrorInterpolater] interface to pass the parameters).
//
// Note that the $-syntax allows to globally define errors such as
//
//	errBase := NewErrorWithData_any_params(nil, "The value of Foo was ${Foo}, which is out of range")
//
// without actually setting the value of "Foo". Calling errBase.Error() will complain about a missing value for Foo.
// However, one can "derive" errors from errBase such as
//
//	errFinal := NewErrorWithData_any_params(errBase, "", "Foo", 5)
//
// (the empty interpolation string defaults to "$w" or "%w" depending on what the wrapped error supports). Then errFinal.Error() will output
// "The value of Foo was 5, which is out of range". Due to the fact that errors and their parameters are immutable, this pattern is common.
//
// Of course, there is the possibility of making a mistake when writing interpolation strings.
//
// The package handles such mistakes by still creating an error with the desired error wrapping behaviour and contained parameters, but calling
// its Error() method will return an error message instead telling about the mistake instead of the intended errror message.
// (In case of parse errors, this error message is rather verbose and prints the whole parameter map)
// The potential mistakes are
//   - syntax errors
//   - missing parameters
//   - using $w or %w if there is no wrapped error or the wrapped error does not support $w
//
// We provide methods ValidateSyntax, ValidateError_Base, ValidateError_Final to check whether an error was constructed OK.
//   - ValidateSyntax only checks the syntax of the interpolation string.
//   - ValidateError_Final additionally (recursively) checks whether rerefences to parameters or wrapped errors work OK.
//   - ValidateError_Base (recursively) is similar to ValidateError_Final, but assumes that $FmtVerb{Var} is filled in later and does not report an error if Var is missing.
//
// ValidateError_Final and ValidateError_Base only syntax-check sub-interpolation strings in $!Cond{sub-interpolationstring}, unless
// they can prove that the branch is taken, so missing variables in untaken branches are OK.
//
// This package does not check or report errors from invalid format verbs that are not supported by the data type at hand.
// This is handled solely by the [fmt] package, which just returns an error report where the formatted output should go; while this is similar to what we do, our Validate-methods do not report this.
package errorsWithData

import (
	"errors"
	"go/token"
)

// ParamMap is an alias to map[string]any. It is used to store arbitrary collections of data associated to a given error.
type ParamMap = map[string]any

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
// Doing otherwise is a serious footgun, since the zero value is a nil pointer of concrete type and will be non-nil
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
	// GetData_map returns a shallow copy of the parameter map. Note that this must never return a nil map.
	GetData_map() map[string]any
	// typically also has Unwrap() error -- all errors created by this package do.
	ValidateError_Base() error
	ValidateError_Final() error
	ValidateSyntax() error
}

// Q: Make this internal?

// ErrorInterpolater is an extension of the error interface that allows the error output to depend on additional data.
// This interface is required to make the $fmtString{VariableName} - mechanism work.
type ErrorInterpolater interface {
	error
	Error_interpolate(ParamMap) string
	ValidateError_Params(params_passed ParamMap) error
	ValidateError_Base() error
}

// DummyValidator is an empty struct that dummy-implements ValidateError_Base, ValidateError_Final, ValidateSyntax and ValidateError_Params (with value receivers). These method all return true.
// The usage scenario is struct-embedding them in an implementation of the ErrorWithData to satisfy the interface if no validation is supported.
type DummyValidator struct{}

func (DummyValidator) ValidateError_Base() error           { return nil }
func (DummyValidator) ValidateError_Final() error          { return nil }
func (DummyValidator) ValidateSyntax() error               { return nil }
func (DummyValidator) ValidateError_Params(ParamMap) error { return nil }

// ErrorWithData[StructType] is a generic interface extending [ErrorWithData_any].
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

// ErrorPrefix is a prefix added to all *internal* error messages/panics (such as invalid interpolation strings) that originate from this package.
const ErrorPrefix = "bandersnatch / error handling: "

// IsExportedIdentifier returns whether the given string (assumed to be valid utf8) denotes a valid name of an exported Go identifier.
// (Meaning it starts with a capital letter, followed by letters digits and underscores -- note that letters and digits may be non-ASCII)
func IsExportedIdentifier(s string) bool {
	return token.IsIdentifier(s) && token.IsExported(s) // use the functions from the go/token standard library. It's surprisingly difficult to get this right otherwise due to potential non-ASCII letters and digits.
}

// NOTE: Some complication in the specs of these functions is the option to DELETE parameters from the parameter map.
// Assume errBase has parameter "V" with value 1. Deleting "V" from it means creating a new errParent wrapping errBase,
// where "V" is absent from errParent's parameter map (note the distinction between nil and absent). Of course, errBase still has "V" due to
// our immutability requirements.
//
// Calling GetData_map or HasData on errParent should act as if "V" was never present in errBase.
// This is why it's wrong to say that GetParameter(err, "V") obtains "V" from the first error in the error chain where such as parameter was found.
// To get the correct semantics, we have to (mentally) associate to every error (including plain errors) an immutable parameter map, where error wrapping defaults to copying/inheriting the map.

// GetData_map returns a map for all parameters stored in the error, where error wrapping defaults to keeping the parameters of the wrapped error.
// For err==nil or if no error in err's error chain has any data, returns an empty map.
//
// Note that the returned map is a (shallow) copy, so the caller may modify it without affecting the error.
// err itself does not need to have been created by this package and is of plain error type.
// We simply follow err's error chain until we find some error that we can work with.
// In practise, this means that this will return a map of all parameters that were ever added to the error (adding creates a new error wrapping the old one)
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
	/*
		if f := GetInvalidParameterNameHandler(); f != nil {
			if !IsExportedIdentifier(parameterName) {
				f(parameterName)
			}
		}
	*/
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

// GetParameter returns the value stored under the key parameterName, possibly following err's error chain (error wrapping defaults to inheriting the wrapped error's parameters).
//
// If no entry was found in the error chain or err==nil, returns (nil, false). Note that the err argument is of plain error type.
func GetParameter(err error, parameterName string) (value any, wasPresent bool) {
	/*
		if f := GetInvalidParameterNameHandler(); f != nil {
			if !IsExportedIdentifier(parameterName) {
				f(parameterName)
			}
		}
	*/

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

// Callback if some parameter name is not a valid exported identifier.
// Removed, because we might as well support this on the map API.

/*
type invalidParameterNameHandlerType = func(s string)

var defaultInvalidParamerNameHandler invalidParameterNameHandlerType = func(s string) {}

// Note: Stores address of variable of function type; this needless extra indirection is because sync/atomic has no variant for variables of function type (which are pointers under the hood).
// At any rate, the Go language is not powerful enough to write that without indirection or reflection.
var invalidParameterNameHandler atomic.Pointer[invalidParameterNameHandlerType] = func() (ret atomic.Pointer[invalidParameterNameHandlerType]) {
	ret.Store(&defaultInvalidParamerNameHandler)
	return
}()

func SetInvalidParameterNameHandler(handler invalidParameterNameHandlerType) (old_handler invalidParameterNameHandlerType) {
	// NOTE: handler is a local copy of the relevant function (i.e. a copy of the function pointer).
	// The pointer-to-function_pointer stored inside invalidParameterNameHandler always either points to (non-exported) defaultInvalidParameterNameHandler
	// or to a (heap-escaped) such handler from this method.
	var old_handler_ptr *invalidParameterNameHandlerType

	for {
		old_handler_ptr = invalidParameterNameHandler.Load()
		if invalidParameterNameHandler.CompareAndSwap(old_handler_ptr, &handler) {
			break
		}
	}
	old_handler = *old_handler_ptr
	return
}

func GetInvalidParameterNameHandler() invalidParameterNameHandlerType {
	return *(invalidParameterNameHandler.Load())
}

*/
