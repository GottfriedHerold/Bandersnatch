package errorsWithData

import "fmt"

// This file is responsible for handling flags that can be passed to exported functions that are used to create new instances of ErrorWithData.
// These flags control the behaviour of such creation functions with regards to the following questions:
//
// When setting values for parameters, if there is already a parameter present from the wrapped error, which one should take precendence?
// In addition to either prefering the old or new one, there is also the option to require that the values actually match.
// For this matching, we may further need to specify a custom equality check (to accomodate for uncomparable types and IsEqual-methods)
//
// When creating an ErrorWithData[T] for some appropriate T, we need to make sure that there is some data for every field of T.
// We can either silently zero-initialize missing fields or treat missing fields as errors
//
// Should we validate newly created errors (e.g. to ensure there is no syntax error in the interpolation string). If yes, how?
// (Validating base errors vs. final errors)
//
// How should we treat actual errors in this process? Should we just panic? This may be appropriate for init-level function that create static errors on startup,
// but I general, I am very hesitant to just let a library function panic.

// Implementation-wise, we distinguish between flags passed as arguments to our (public) methods and our (internal configuration)
// Notably, flag arguments are of (interface) type flagArgument. These are parsed and each flag causes our config struct to change.
// flagArgument is intentionally not exported. We simply provide a set of (all) possible values as exported constants (well, variables, because Go has no const structs).
// These are simply some struct flag{value int} for type-safety.
// Note that different flags *may* actually have a different type. This allows to satisfy sub-interfaces of flagArgument to restrict a method to only take a subset of the
// possible flags, with a compile-time failure for meaningless flags.

// flagArgument is the interface type satisfied by all optional flags that are passed to exported function.
// Note that the API of these exported functions typically takes a more constrained interface than flagArgument to
// further filter the set of allowed flags on a per-function basis.
type flagArgument interface {
	isFlag()        // tag to mark flags
	getValue() int  // returns a enum-type constant flagArg_<foo> that determines the actual value/meaning of the flag.
	String() string // returns a string describing the meaning of the flag. Only used for debugging.
}

// Note: Even though our different flags are realized with different types,
// we have a single list of int values that determine the actual meaning of the flag (rather than a separate list for each type).
// This is done to decouple the type of the exported argument flags (which is just there to restrict methods to only meaningful flags) from their actual meaning.
// This allows changing the type without needed to refactor much.

const (
	flagArg_Unset = iota // The zero value corresponds to an unset value. This should never appear and we panic if it does. Note that since we don't export any concrete type implementing flagArgument, it takes considerable effort to trigger this.

	// For overwriting data, there are really 3 (essential) config items:
	//  - preference for old/new
	//  - should we check for equality
	//  - how should we check for equality (we could use nil here as don't check at all, but nil could also special case plain ==)
	// Note that the old/new preference actually still matters even if we do an equality check, because a custom equality check might not do plain "=="". In fact, the default one does not.
	// Still, we simplify the API insofar as that setting any old/new preference unsets the equality check and this is the only way to unset it.
	// Conversely, setting the equality check will honor the last value of the old/new - preference.
	// Also, setting any equality check function will actually request an equality check.
	flagArg_PreferOld       // prefer old values when overwriting data.
	flagArg_PreferNew       // prefer new values when overwriting data
	flagArg_AssertEqual     // assume values are equal (using the default comparison function). Note that this is really two options: the last setting of PreferOld/PreferNew actually still determines preference for old/new. We just present it as a ternary toggle to the user for simplicity.
	flagArg_AssertEqual_fun // assume values are equal (using a custom comparison function)). Note that a flagArg with this value may have a type that also contains a function pointer in addition to wrapping just this int.

	// for missing data, there is just one config item:
	// either silently zero-initialize or zero-initialize and treat it as error.
	flagArg_FillWithZeros      // zero-initialize missing value for T's fields when creating an ErrorWithData[T]
	flagArg_MissingDataIsError // missing values for T's fields are an error when creating an ErrorWithData[T]

	// Since we use a single utility function for checking that data is there and zeroing out the data,
	// these are kind-of equivalent to FillWithZeros and MissingDataIsError, but used for [HasData].
	// The reason is simply that [HasData] does not actually modify the error and zero out missing data, so the above flags would be a misnomer.
	flagArg_IgnoreMissingData // Treat missing data as zero of appropriate type. Only check that data that is there has the right type. Only valid for [HasData]
	flagArg_EnsureDataPresent // Don't treat missing data as zero. Only valid for [HasData]

	// Is validation requested?
	flagArg_NoValidation   // Don't perform validation
	flagArg_ValidateSyntax // Check for syntax errors
	flagArg_ValidateBase   // Validate as a base error (i.e. ignore missing values for $fmt{Var} - expression, as those might be filled in later). Note that we follow the error chain if applicable.
	flagArg_ValidateFinal  // Validate as a final error (this means we ensure all variables referred to by format strings actuall exist). Note that we may follow the error chain if applicable.

	// Error handling: Return error or panic
	// We might consider separating that according to the type of error
	flagArg_PanicOnErrors // panic on (all) errors
	flagArg_ReturnErrors  // return errors as last return value

	// Handling empty interpolation strings
	flagArg_AllowEmptyString // an empty interpolation string is just an empty string. This allows .Error() to return an empty string.
	flagArg_DefaultToWrapped // an empty interpolation string is interpreted as "refer to base error" (and panic if no base error exists).
)

// fArg is an implementation of flagArgument that just wraps an int (enum-style)
// Note that flagArgument is implemented by the value type fArg itself as opposed to *fArg (which then also implements it, by Go's rules).
//
// the fArg type is not really used directly; our concrete exported errors have various types fArg_Foo, which all struct-embed fArg.
type fArg struct {
	val int
}

func (fArg) isFlag()          {}
func (f fArg) getValue() int  { return f.val }
func (f fArg) String() string { return printFlagArg(&f) } // Both &f and f work here.

// the various fArg_Foo types are the concrete types that our exported flags have. Each of these types satisfies the flagArgument interface.
// The actual type depends on the flag. Each flag types implements certain extra functions, which makes it satisfy a more constrained interface, depening on the flag type.
// This is done to further restrict the set of allowed flags to certain functions at compile-time, if applicable.
type (
	fArg_HasData     struct{ fArg }
	fArg_MissingData struct{ fArg }
	fArg_Validity    struct{ fArg }
	fArg_Panic       struct{ fArg }
	fArg_OldData     struct{ fArg }
	fArg_EmptyString struct{ fArg }
)

// printFlagArg output a string representation of the flag. This is only used for debugging.
func printFlagArg(f flagArgument) string {
	switch v := f.getValue(); v {
	case flagArg_Unset:
		// Note: The type information is lost here due to fArg_Foo implementing String(), which calls printFlagArg, via struct-embedding.
		// There is no way to retain the type-information without proper inheritance, which Go does not provide.
		return fmt.Sprintf("Zero value of flag argument of some specific type, possibly %T", f)

	case flagArg_PreferOld:
		return "Prefer old values when overwriting data"
	case flagArg_PreferNew:
		return "Prefer new values when overwriting data"
	case flagArg_AssertEqual:
		return "Check that values are equal when overwriting already set data"
	case flagArg_AssertEqual_fun:
		return "Check that values are equal when overwriting already set data, using a custom comparison function"

	case flagArg_FillWithZeros:
		return "Zero-initialize missing values for T when creating an ErrorWithData[T]"
	case flagArg_MissingDataIsError:
		return "missing values for T's fields are an error when creating an ErrorWithData[T]"

	case flagArg_IgnoreMissingData:
		return "missing values for T's fields are ignored"
	case flagArg_EnsureDataPresent:
		return "require a value for each of T's fields"

	case flagArg_NoValidation:
		return "No validation of created error is requested"
	case flagArg_ValidateSyntax:
		return "Check validity of created error with ValidateSyntax"
	case flagArg_ValidateBase:
		return "Check validity of created error with ValidateError_Base"
	case flagArg_ValidateFinal:
		return "Check validity of created error with ValidateError_Final"

	case flagArg_PanicOnErrors:
		return "Panic if an error is encountered"
	case flagArg_ReturnErrors:
		return "Return an error rather than panicking if an error is encountered (some failure conditions still panic, but those are explicitly documented)"

	case flagArg_AllowEmptyString:
		return "Allow creating errors with empty error message"
	case flagArg_DefaultToWrapped:
		return "Empty interpolation string defaults to wrapping base error (or panic)"

	default:
		return fmt.Sprintf("Unrecognized flag argument with internal value set to %v", v)
	}
}

// config_OldData collects the (internal) flags that determine how old vs new data is handled.
// Note that we require that the zero value of this type corresponds to the default settings.
// Since this couples semantics of internals to the defaults (we would have use a preferNew bool to change the default)
// we require all read accesses to go through methods to get an API independent from the defaults.
type config_OldData struct {
	preferOld       bool
	doEqualityCheck bool
	checkFun        func(x, y any) (isEqual bool, inequalityReason error)
}

// PreferOld reads out the configuration to determine whether data from the base error or newly provided data should take preference.
func (p *config_OldData) PreferOld() bool {
	return p.preferOld
}

// PreferNew reads out the configuration to determine whether data from the base error or newly provided data should take preference.
// It is just the negation of [PreferOld]
func (p *config_OldData) PreferNew() bool {
	return !p.preferOld
}

// PerformEqualityCheck reads out the configuration to determine whether we perform an equality check (with a failure considered an error)
// between data from the base error and newly provided data (if both exist).
//
// Note the the equality check is not just done by ==, but rather by calling a custom function, which is obtained by [GetCheckFun]
func (p *config_OldData) PerformEqualityCheck() bool {
	return p.doEqualityCheck
}

// GetCheckFun reads out the configuration to provide the custom equality check function.
// This is only meaningful if [PerformEqualityCheck] returns true.
func (p *config_OldData) GetCheckFun() func(x, y any) (isEqual bool, inequalityReason error) {
	if p.checkFun == nil {
		return comparison_very_naive_old
	} else {
		return p.checkFun
	}

}

// config_ErrorHandling collects the internal flags that determine how errors during creating of errors are handled.
// Notably, we may just return an error or panic, the latter being appropriate for error constants that are created during program initialization.
//
// similar to [config_OldData], reads should go through methods.
type config_ErrorHandling struct {
	panicOnError bool
}

// PanicOnAllErrors returns true if the configuration is set to panic on all errors.
func (p *config_ErrorHandling) PanicOnAllErrors() bool {
	return p.panicOnError
}

// config_Validation collects the internal flags that determine whether validation of the newly created error is requested (and what kind of validation)
//
// Similar to [config_OldData], reads should go through methods.
type config_Validation struct {
	doValidation int // use constants validationRequest_* for meaning.
}

// Note: We do not want to (re-)use the flagArg_* values here, because we want the zero value to match our default setting.
const (
	validationRequest_Syntax = iota
	validationRequest_NoValidation
	validationRequest_Base
	validationRequest_Final
)

// WhatValidationIsRequested returns an int determining the type of requested validation.
//
// The meaning of the returned int-constants is defined by the validationRequest_* constants
func (p *config_Validation) WhatValidationIsRequested() int {
	return p.doValidation
}

// config_ImplicitZero collects the internal flags that determine what should be done if data for some error is missing.
// Note that "missing" only makes sense for ErrorWithData[T] rather than ErrorWithData_any and refers to the possibility that some
// data required for a field of T might not be present.
//
// This config determines whether missing data should be *silently* treated as zero or considered an error (and still be treated as zero, because there is nothing else we can do)
//
// Similar to [config_OldData], reads should go through methods.
type config_ImplicitZero struct {
	implicitZero bool
	// addMissingData     bool
}

/*
// ShouldAddMissingData determines whether we actually modify the error under consideration and add zero entries for missing data.
//
// This is the correct behaviour for methods creating errors of type (satisfying) ErrorWithData[T], because we want to guarantee internal invariants even if data is missing.
func (p *config_ZeroFill) ShouldAddMissingData() bool {
	return p.addMissingData
}
*/

// IsMissingDataError determines whether missing data is treated as an error or silently as a zero value.
func (p *config_ImplicitZero) IsMissingDataError() bool {
	return !p.implicitZero
}

// config_SetZeros is passed to [ensureCanMakeStructFromParameters] to determine whether the function should actually modify the error or just perform a check.
// Which of these options is appropriate is entirely determined by the call site, so we only ever use constants of this type.
type config_SetZeros struct {
	setErrorsToZero bool
}

// ModifyData is used to query a given [config_SetZeros] as to whether data should be modified or not.
func (p *config_SetZeros) ModifyData() bool {
	return p.setErrorsToZero
}

// config_EmtpyString collects the interal flags that determine how empty interpolation strings should be handled.
// We can either treat them as-is or have them default to "refer to the base error" (with a panic if there is no base error)
// Since we do not want to encourage empty error messages, we default to the latter.
//
// Similar to [config_OldData], reads should go through methods.
type config_EmptyString struct {
	allowEmpty bool
}

// AllowEmptyString reads out the config whether empty strings are just taken literally.
func (p *config_EmptyString) AllowEmptyString() bool {
	return p.allowEmpty
}

// errorCreatingConfig is a struct collecting all our configuration options (except [config_SetZero]).
// The intended usage is to initialize it with defaults and call [parseFlagArgs] on it to modify it according to user-provided flags.
//
// In order to avoid having a multitude of [parseFlagArgs] - variants, [parseFlagArgs] only works with this struct (which contains all config_* types), even if some parts of
// it are meaningless in a given context. Note that the internals of each config type are chosen such that the zero value of errorCreationConfig corresponds to the default value,
// at least for most cases.
type errorCreationConfig struct {
	config_OldData
	config_ErrorHandling
	config_Validation
	config_ImplicitZero
	config_EmptyString
}

// Actually exported flags go here:

var (
	// PreferPreviousData means that when replacing associated data in errors, we keep the old value if some value is already present for a given key.
	PreferPreviousData = fArg_OldData{fArg{val: flagArg_PreferOld}}
	// ReplacePreviousData means that when replacing associated data in errors, we unconditionally override already-present values for a given key.
	ReplacePreviousData = fArg_OldData{fArg{val: flagArg_PreferNew}}
	// TODO: Document equality notion here
	// AssertDataIsNotReplaced means that when replacing associated data in errors, we panic if a different value was already present for a given key.
	AssertDataIsNotReplaced = fArg_OldData{fArg{val: flagArg_AssertEqual}}

	// MissingDataAsZero is passed to functions to indicate that missing data should be silently zero initialized
	MissingDataAsZero = fArg_MissingData{fArg{val: flagArg_FillWithZeros}}
	// MissingDataIsError is passed to functions to indicate that the function should report an error (possibly panic) if data is missing
	MissingDataIsError = fArg_MissingData{fArg{val: flagArg_MissingDataIsError}}

	// IgnoreMissingData is passed to [HasData] to cause it to ignore the case of merely missing data. We only type-check in this case
	IgnoreMissingData = fArg_HasData{fArg{val: flagArg_IgnoreMissingData}}
	// EnsureMissingData is passed to [HasData] to cause it to not ignore merely missing data. As this is the default, this flag is never needed.
	EnsureDataIsPresent = fArg_HasData{fArg{val: flagArg_EnsureDataPresent}}

	// ReturnError is passed to functions to indicate that it should return an error rather than panic. Note that certain (explicitly stated) conditions may still cause a panic.
	ReturnError = fArg_Panic{fArg{val: flagArg_ReturnErrors}}
	// PanicOnAllErrors is passed to functions to indicate that they should panic on failure. Useful for init-level functions.
	PanicOnAllErrors = fArg_Panic{fArg{val: flagArg_PanicOnErrors}}

	// NoValidation is passed to functions creating errors to indicate that no validation (pertaining to recursive checking of interpolation strings) is requested.
	NoValidation = fArg_Validity{fArg{val: flagArg_NoValidation}}
	// ErrorUnlessValidSyntax is passed to functions creating errors to indicate that syntax-validation of the interpolation string is requested
	ErrorUnlessValidSyntax = fArg_Validity{fArg{val: flagArg_ValidateSyntax}}
	// ErrorUnlessValidBase is passed to functions creating errors to indicate that recursive validation of interpolation strings is requested. This includes that data referred to actually exists, except for possible $fmtVerb{Param} expressions, which might be filled by a wrapping error.
	ErrorUnlessValidBase = fArg_Validity{fArg{val: flagArg_ValidateBase}}
	// ErrorUnlessValidFinal is passed to functions creating errors to indicate that recursive validation of interpolation strings is requested. This includes checking that data referred to by %fmtVerb{Param} or $fmtVerb{Param}-expressions actually exists.
	ErrorUnlessValidFinal = fArg_Validity{fArg{val: flagArg_ValidateFinal}}

	// AllowEmptyString needs to be passed to functions creating errors to allow an empty error message.
	// Otherwise, an empty string defaults to %w resp. $w and we panic if there is no base error.
	AllowEmptyString = fArg_EmptyString{fArg{val: flagArg_AllowEmptyString}}

	// DefaultToWrapping is passed to function creating errors to cause an empty interpolation string to default to
	// %w or $w, repeating the wrapped error. If there is no wrapped error, we panic.
	// Note that this is the default setting, so there should never be a need to pass this flag explicitly.
	DefaultToWrapping = fArg_EmptyString{fArg{val: flagArg_DefaultToWrapped}}
)

// allFlagArgs is a list of all possible flag argument values above (and possible outputs of functions generating flagArguments)
// This is only used for testing, but defined here to simplify refactoring, as it's tied to the above list of definitions.
var allFlagArgs []flagArgument = []flagArgument{
	PreferPreviousData, ReplacePreviousData, AssertDataIsNotReplaced, MissingDataAsZero, MissingDataIsError, IgnoreMissingData, EnsureDataIsPresent, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping,
}

func parseFlagArgs_HasData(flags ...flagArgument_HasData) (ret config_ImplicitZero) {
	ret = config_ImplicitZero{implicitZero: true}
	for _, flag := range flags {
		switch v := flag.getValue(); v {
		case flagArg_Unset:
			panic("cannot happen") // unless the user tries hard
		case flagArg_IgnoreMissingData:
			ret.implicitZero = true
		case flagArg_EnsureDataPresent:
			ret.implicitZero = false
		default:
			panic("Cannot happen")
		}
	}
	return
}

func parseFlagArgs_GetData(flags ...flagArgument_GetData) (retZeroFill config_ImplicitZero, retPanic config_ErrorHandling) {
	// retPanic = config_ErrorHandling{panicOnError: false}
	retZeroFill = config_ImplicitZero{implicitZero: false}
	for _, flag := range flags {
		switch v := flag.getValue(); v {
		case flagArg_Unset:
			panic("Cannot happen")
		case flagArg_MissingDataIsError:
			retZeroFill.implicitZero = false
		case flagArg_FillWithZeros:
			retZeroFill.implicitZero = true
		case flagArg_PanicOnErrors:
			retPanic.panicOnError = true
		case flagArg_ReturnErrors:
			retPanic.panicOnError = false
		default:
			panic("Cannot happen")
		}
	}
	return
}

// Needs to be generic function due to Go's lack of covariance.

// func (p *errorCreationParams) parseFlagArgs(flags ...flagArgument) {
func parseFlagArgs[ArgType flagArgument](p *errorCreationConfig, flags ...ArgType) {
	for _, individualFlag := range flags {
		switch v := individualFlag.getValue(); v {
		case flagArg_Unset:
			panic("Cannot happen")
		case flagArg_PreferOld:
			p.preferOld = true
			p.doEqualityCheck = false
		case flagArg_PreferNew:
			p.preferOld = false
			p.doEqualityCheck = false
		case flagArg_AssertEqual:
			p.doEqualityCheck = true
			p.checkFun = nil
		case flagArg_AssertEqual_fun:
			panic("Not implemented yet")
		case flagArg_MissingDataIsError:
			p.implicitZero = false
		case flagArg_FillWithZeros:
			p.implicitZero = true
		case flagArg_ReturnErrors:
			p.panicOnError = false
		case flagArg_PanicOnErrors:
			p.panicOnError = true
		case flagArg_NoValidation:
			p.doValidation = validationRequest_NoValidation
		case flagArg_ValidateSyntax:
			p.doValidation = validationRequest_Syntax
		case flagArg_ValidateBase:
			p.doValidation = validationRequest_Base
		case flagArg_ValidateFinal:
			p.doValidation = validationRequest_Final
		case flagArg_AllowEmptyString:
			p.allowEmpty = true
		case flagArg_DefaultToWrapped:
			p.allowEmpty = false
		default:
			panic("Cannot happen")
		}
	}
}

type flagArgument_HasData interface {
	flagArgument
	isFlag_HasData()
}

func (fArg_HasData) isFlag_HasData() {}

type flagArgument_GetData interface {
	flagArgument
	isFlag_GetData()
}

type flagArgument_NewErrorStruct interface {
	flagArgument
	isFlag_NewErrorStruct()
}

type flagArgument_NewErrorParams interface {
	flagArgument
	isFlag_NewErrorParams()
}

type flagArgument_NewErrorAny = flagArgument_NewErrorStruct // same constraints.

type flagArgument_Delete interface {
	flagArgument
	isFlag_Delete()
}

type flagArgument_DeleteAny interface {
	flagArgument
	isFlag_DeleteAny()
}

type flagArgument_AsErrorWithData interface {
	flagArgument
	isFlag_AsErrorWithData()
}

func (fArg_MissingData) isFlag_GetData() {}
func (fArg_Panic) isFlag_GetData()       {}

func (fArg_OldData) isFlag_NewErrorStruct()     {}
func (fArg_Panic) isFlag_NewErrorStruct()       {}
func (fArg_Validity) isFlag_NewErrorStruct()    {}
func (fArg_EmptyString) isFlag_NewErrorStruct() {}

func (fArg_OldData) isFlag_NewErrorParams()     {}
func (fArg_Panic) isFlag_NewErrorParams()       {}
func (fArg_Validity) isFlag_NewErrorParams()    {}
func (fArg_EmptyString) isFlag_NewErrorParams() {}
func (fArg_MissingData) isFlag_NewErrorParams() {}

func (fArg_Panic) isFlag_DeleteAny()    {}
func (fArg_Validity) isFlag_DeleteAny() {}

func (fArg_MissingData) isFlag_Delete() {}
func (fArg_Panic) isFlag_Delete()       {}
func (fArg_Validity) isFlag_Delete()    {}

func (fArg_MissingData) isFlag_AsErrorWithData() {}
func (fArg_Panic) isFlag_AsErrorWithData()       {}

// flagArgument_NewErrorAny handled by type alias
