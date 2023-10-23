package errorsWithData

import "fmt"

// This file is responsible for handling flags that can be passed to exported functions that are used to create new instances of ErrorWithData.
// These flags control the behaviour of such creation functions with regards to the following questions:
//
// When setting values for parameters, if there is already a parameter present from the wrapped error, which one should take precendence?
// In addition to either prefering the old or new one, there is also the option to require that the values actually match.
// For this this matching, we may further need to specify a custom equality check (to accomodate for uncomparable types and IsEqual-methods)
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

type flagArgument interface {
	isFlag()
	getValue() int
}

type flagArgument_HasData interface {
	flagArgument
	isFlag_HasData()
}

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

type flagArgument_NewErrorAny interface {
	flagArgument
	isFlag_NewErrorAny()
}

type flagArgument_Delete interface {
	flagArgument
	isFlag_Delete()
}

type flagArgument_DeleteAny interface {
	isFlag_DeleteAny()
}

type flagArgument_NoEmptyString interface {
	isFlag_NoEmptyString()
}

// Note: We have a single list of int values that determine the actual meaning of the flag (rather than a separate list for each type).
// This is done to decouple the type of the exported argument flags (which is just there to restrict methods to only meaningful flags) from their actual meaning.
// This allows changing the type without needed to refactor much.

const (
	flagArg_Unset = iota // The zero value corresponds to an unset value. This should never appear and we panic if it does. Note that since we don't export any concrete type implementing flagArgument, it takes considerable effort to trigger this.

	// For overwriting data, there are really 3 (essential) config items:
	//  - preference for old/new
	//  - should be check for equality
	//  - how should we check for equality (we could use nil here as don't check at all, but nil could also special case plain ==)
	// Note that the old/new preference actually still matters even if we do an equality check, because a custom equality check might not do plain "=="". In fact, the default one does not.
  	// Still, we simplify the API insofar as that setting any old/new preference unsets the equality check and this is the only way to unset it.
	// Conversely, setting the equality check will honor the last value of the old/new - preference.
	// Also, setting any equality check function will actually request an equality check.
	flagArg_PreferOld       // prefer old values when overwriting data.
	flagArg_PreferNew       // prefer new values when overwriting data
	flagArg_AssertEqual     // assume values are equal (using the default comparison function). Note that this is really two options: the last setting of PreferOld/PreferNew actually still determines preference for old/new. We just present it as a ternary toggle to the user for simplicity.
	flagArg_AssertEqual_fun // assume values are equal (using a custom comparsing function)). Note that a flagArg with this value may have a type that also contains a function pointer in addition to wrapping just this int.

	// for missing data, there is just one config item:
	//  either silently zero-initialize or zero-initialize and treat it as error.

	flagArg_FillWithZeros      // zero-initialize missing value for T's fields when creating an ErrorWithData[T]
	flagArg_MissingDataIsError // missing values for T's fields are an error when creating an ErrorWithData[T]

	flagArg_IgnoreMissingData // Treat missing data as zero of appropriate type. Only check that data that is there has the right type. Only valid for [HasData]
	flagArg_EnsureDataPresent // Don't treat missing data as zero. Only valid for [HasData]

	// Is validation requested?
	flagArg_NoValidation   // Don't perform validation
	flagArg_ValidateSyntax // Check for syntax errors
	flagArg_ValidateBase   // Validate as a base error (i.e. ignore missing values for $fmt{Var} - expression, as those might be filled in later). Note that we follow the error chain if applicable.
	flagArg_ValidateFinal  // Validate as a final error (i.e. ensure all variables refered) actuall exist. Note that we may follow the error chain if applicable.

	// Error handling: Return error or panic
	// We might consider separating that according to
	flagArg_PanicOnErrors // panic on (all) errors
	flagArg_ReturnErrors  // return errors as last return value

	// Handling empty interpolation strings
	flagArg_AllowEmptyString
	flagArg_DefaultToWrapped
)

type fArg struct {
	val int
}

func (*fArg) isFlag()         {}
func (f *fArg) getValue() int { return f.val }
func (f fArg) String() string { return printFlagArg(&f) }

type fArg_HasData struct{ val int }
type fArg_MissingData struct{ val int }
type fArg_Validity struct{ val int }
type fArg_Panic struct{ val int }
type fArg_OldData struct{ val int }
type fArg_EmptyString struct{ val int }

func (*fArg_HasData) isFlag()         {}
func (f *fArg_HasData) getValue() int { return f.val }
func (f fArg_HasData) String() string { return printFlagArg(&f) }

func (*fArg_MissingData) isFlag()         {}
func (f *fArg_MissingData) getValue() int { return f.val }
func (f fArg_MissingData) String() string { return printFlagArg(&f) }

func (*fArg_Validity) isFlag()         {}
func (f *fArg_Validity) getValue() int { return f.val }
func (f fArg_Validity) String() string { return printFlagArg(&f) }

func (*fArg_Panic) isFlag()         {}
func (f *fArg_Panic) getValue() int { return f.val }
func (f fArg_Panic) String() string { return printFlagArg(&f) }

func (*fArg_OldData) isFlag()         {}
func (f *fArg_OldData) getValue() int { return f.val }
func (f fArg_OldData) String() string { return printFlagArg(&f) }

func (*fArg_EmptyString) isFlag()         {}
func (f *fArg_EmptyString) getValue() int { return f.val }
func (f fArg_EmptyString) String() string { return printFlagArg(&f) }

func (*fArg_HasData) isFlag_HasData() {}

func (*fArg_MissingData) isFlag_GetData() {}
func (*fArg_Panic) isFlag_GetData()       {}

func printFlagArg(f flagArgument) string {
	switch v := f.getValue(); v {
	case flagArg_Unset:
		return fmt.Sprintf("Zero value of flag argument of specific type %T", f)

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

type mergeParams struct {
	preferOld       bool
	doEqualityCheck bool
	checkFun        func(x, y any) (isEqual bool, inequalityReason error)
}

func (p *mergeParams) PreferOld() bool {
	return p.preferOld
}

func (p *mergeParams) PreferNew() bool {
	return !p.preferOld
}

func (p *mergeParams) PerformEqualityCheck() bool {
	return p.doEqualityCheck
}

func (p *mergeParams) GetCheckFun() func(x, y any) (bool, error) {
	if p.checkFun == nil {
		return comparison_very_naive
	} else {
		return p.checkFun
	}

}

type errorHandlingParams struct {
	panicOnError bool
}

func (p *errorHandlingParams) PanicOnError() bool {
	return p.panicOnError
}

type validationParams struct {
	doValidation int // reuse constants from flag_Arg_Validate*
}

func (p *validationParams) WhatValidationIsRequested() int {
	return p.doValidation
}

type zeroFillParams struct {
	missingDataIsError bool
	addMissingData     bool
}

func (p *zeroFillParams) ShouldAddMissingData() bool {
	return p.addMissingData
}

func (p *zeroFillParams) IsMissingDataError() bool {
	return p.missingDataIsError
}

func (p *zeroFillParams) ShouldZeroOnTypeError() bool {
	return p.addMissingData // TODO: Is this right?
}

type handleEmptyInterpolationString struct {
	allowEmpty bool
}

func (p *handleEmptyInterpolationString) AllowEmptyString() bool {
	return p.allowEmpty
}

type errorCreationParams struct {
	mergeParams
	errorHandlingParams
	validationParams
	zeroFillParams
	handleEmptyInterpolationString
}

var (
	// PreferPreviousData means that when replacing associated data in errors, we keep the old value if some value is already present for a given key.
	PreferPreviousData = fArg_OldData{val: flagArg_PreferOld}
	// ReplacePreviousData means that when replacing associated data in errors, we unconditionally override already-present values for a given key.
	ReplacePreviousData = fArg_OldData{val: flagArg_PreferNew}
	// TODO: Document equality notion here
	// AssertDataIsNotReplaced means that when replacing associated data in errors, we panic if a different value was already present for a given key.
	AssertDataIsNotReplaced = fArg_OldData{val: flagArg_AssertEqual}

	// MissingDataAsZero is passed to functions to indicate that missing data should be silently zero initialized
	MissingDataAsZero = fArg_MissingData{val: flagArg_FillWithZeros}
	// MissingDataIsError is passed to functions to indicate that the function should report an error (possibly panic) if data is missing
	MissingDataIsError = fArg_MissingData{val: flagArg_MissingDataIsError}

	// IgnoreMissingData is passed to [HasData] to cause it to ignore the case of merely missing data. We only type-check in this case
	IgnoreMissingData = fArg_HasData{val: flagArg_IgnoreMissingData}
	// EnsureMissingData is passed to [HasData] to cause it to not ignore merely missing data. As this is the default, this flag is never needed.
	EnsureDataIsPresent = fArg_HasData{val: flagArg_EnsureDataPresent}

	// ReturnError is passed to functions to indicate that it should return an error rather than panic. Note that certain (explicitly stated) conditions may still cause a panic.
	ReturnError = fArg_Panic{val: flagArg_ReturnErrors}
	// PanicOnAllErrors is passed to functions to indicate that they should panic on failure. Useful for init-level functions.
	PanicOnAllErrors = fArg_Panic{val: flagArg_PanicOnErrors}

	NoValidation           = fArg_Validity{val: flagArg_NoValidation}
	ErrorUnlessValidSyntax = fArg_Validity{val: flagArg_ValidateSyntax}
	ErrorUnlessValidBase   = fArg_Validity{val: flagArg_ValidateBase}
	ErrorUnlessValidFinal  = fArg_Validity{val: flagArg_ValidateFinal}

	// AllowEmptyString needs to be passed to functions creating errors to allow an empty error message.
	// Otherwise, an empty string defaults to %w resp. $w and we panic if there is no base error.
	AllowEmptyString = fArg_EmptyString{val: flagArg_AllowEmptyString}

	// DefaultToWrapping is passed to function creating errors to cause an empty interpolation string to default to
	// %w or $w, repeating the wrapped error. If there is no wrapped error, we panic.
	// Note that this is the default setting, so there should never be a need to pass this flag.
	DefaultToWrapping = fArg_EmptyString{val: flagArg_DefaultToWrapped}
)

const noFlagRestrictions = 0

func parseFlagArgs_HasData(flags ...flagArgument_HasData) (ret zeroFillParams) {
	ret = zeroFillParams{missingDataIsError: false, addMissingData: false}
	for _, flag := range flags {
		switch v := flag.getValue(); v {
		case flagArg_Unset:
			panic("cannot happen") // unless the user tries hard
		case flagArg_IgnoreMissingData:
			ret.missingDataIsError = false
		case flagArg_EnsureDataPresent:
			ret.missingDataIsError = true
		default:
			panic("Cannot happen")
		}
	}
	return
}

func parseFlagArgs_GetData(flags ...flagArgument_GetData) (retZeroFill zeroFillParams, retPanic errorHandlingParams) {
	retPanic = errorHandlingParams{panicOnError: false}
	retZeroFill = zeroFillParams{addMissingData: false, missingDataIsError: true}
	for _, flag := range flags {
		switch v := flag.getValue(); v {
		case flagArg_Unset:
			panic("Cannot happen")
		case flagArg_MissingDataIsError:
			retZeroFill.missingDataIsError = true
		case flagArg_FillWithZeros:
			retZeroFill.missingDataIsError = false
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
func parseFlagArgs[ArgType flagArgument](p *errorCreationParams, flags ...ArgType) {
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
			panic("Not implemented yet") // p.checkFun = ...
		case flagArg_AssertEqual_fun:
			panic("Not implemented yet")
		case flagArg_MissingDataIsError:
			p.missingDataIsError = true
		case flagArg_FillWithZeros:
			p.missingDataIsError = false
		case flagArg_ReturnErrors:
			p.panicOnError = false
		case flagArg_PanicOnErrors:
			p.panicOnError = true
		case flagArg_NoValidation, flagArg_ValidateBase, flagArg_ValidateSyntax, flagArg_ValidateFinal:
			p.doValidation = v
		case flagArg_AllowEmptyString:
			p.allowEmpty = true
		case flagArg_DefaultToWrapped:
			p.allowEmpty = false
		default:
			panic("Cannot happen")
		}
	}
}

/*

// flagMissingDataTreatment is a type used to pass to exported functions how the library should treat missing parameters
// when using the struct-based API.
//
// We provide [RENAMED] and [RENAMED] as possible values. The zero value of this type is invalid.
//
// Selecting [RENAMED] causes the package to report an error or panic if parameters are missing. (This is explicitly specified at a per-function basis).
// [MissingDataAsZero] causes the package to silently zero-initialize values if parameters are missing.
type flagMissingDataTreatment struct {
	val_MissingDataTreatment int
}

func (flagMissingDataTreatment) isFlag() {}

// potential values of RENAMED

const (
	handleMissingData_Unset              = iota // The zero value of [MissingDataTreatment] corresponds to an unset value. Using this causes a panic rather than some default behaviour.
	handleMissingData_FillWithZeros             // zero-initialize missing values
	handleMissingData_MissingDataIsError        // error if values are missing
)

var (
	// MissingDataAsZero is passed to functions to indicate that missing data should be silently zero initialized
	MissingDataAsZero flag = flagMissingDataTreatment{val_MissingDataTreatment: handleMissingData_FillWithZeros}
	// MissingDataIsError is passed to functions to indicate that the function should report an error (possibly panic) if data is missing
	MissingDataIsError flag = flagMissingDataTreatment{val_MissingDataTreatment: handleMissingData_MissingDataIsError}
)

// String is provided to make MissingDataTreatment satisfy fmt.Stringer. This is used for debugging only.
func (m flagMissingDataTreatment) String() string { // Note: value receiver
	switch m.val_MissingDataTreatment {
	case handleMissingData_Unset:
		return "Unset value for flagMissingDataTreatment"
	case handleMissingData_FillWithZeros:
		return "Fill missing data with zeros"
	case handleMissingData_MissingDataIsError:
		return "Error if data is missing"
	default:
		// cannot happen without using unsafe methods.
		return fmt.Sprintf("Unexpected internal value %v for flagMissingDataTreatment", m.val_MissingDataTreatment)
	}
}

type flagErrorHandling struct {
	val_ErrorHandling int
}

// tag to satisfy interfaces

func (flagErrorHandling) isFlag() {}

const (
	errorHandling_Unset = iota // unset value. Must never appear
	errorHandling_ReturnError
	errorHandling_PanicOnAllErrors
	errorHandling_PanicOnDataError
	errorHandling_DontPanicOnDataError
	errorHandling_PanicOnValidityError
	errorHandling_DontPanicOnValidityError
)

type errorHandlingBehaviour struct {
	panicOnDataError     bool
	panicOnValidityError bool
}

var (
	// ReturnError is passed to functions to indicate that it should return an error rather than panic. Note that certain (explicitly stated) conditions may still cause a panic.
	ReturnError flag = flagErrorHandling{val_ErrorHandling: errorHandling_ReturnError}
	// PanicOnAllErrors is passed to functions to indicate that they should panic on failure. Useful for init-level functions.
	PanicOnAllErrors flag = flagErrorHandling{val_ErrorHandling: errorHandling_PanicOnAllErrors}
	// PanicOnDataError is passed to functions to indicate that they should panic if the passed parameters are invalid (i.e. data is missing / has wrong type inconsistent with T).
	// This is only meaningful when creating an ErrorWithData[T] for some non-empty struct type T.
	PanicOnDataError flag = flagErrorHandling{val_ErrorHandling: errorHandling_PanicOnDataError}
	// PanicOnValidityError is passed to functions to indicate that they should panic if the created error fails validation.
	// This is only meaningful when validation is actually requested.
	PanicOnValidityError flag = flagErrorHandling{val_ErrorHandling: errorHandling_PanicOnValidityError}
	// DontPanicOnDataError is passed to functions to indicate that they should not panic (but return an error) if the passed parameters are invalid (i.e. data is missing / has wrong type inconsistent with T)
	// This is only meaningful when creating an ErrorWithData[T] for some non-empty struct type T.
	DontPanicOnDataError flag = flagErrorHandling{val_ErrorHandling: errorHandling_DontPanicOnDataError}
	// DontPanicOnValidityError is passed to functions to indicate that they should not panic (but return an error) if the created error fails validation.
	// This is only meaningful when validation is actually requested.
	DontPanicOnValidityError flag = flagErrorHandling{val_ErrorHandling: errorHandling_DontPanicOnValidityError}
)

func (m flagErrorHandling) String() string {
	switch m.val_ErrorHandling {
	case errorHandling_Unset:
		return "Unset value for flagErrorHandling"
	case errorHandling_PanicOnAllErrors:
		return "Panic on all errors"
	case errorHandling_ReturnError:
		return "Return error"
	case errorHandling_PanicOnDataError:
		return "Panic on data error"
	case errorHandling_DontPanicOnDataError:
		return "Don't panic on data error"
	case errorHandling_PanicOnValidityError:
		return "Panic on validity error"
	case errorHandling_DontPanicOnValidityError:
		return "Don't panic on validty error"
	default:
		// cannot happen without using unsafe methods
		return fmt.Sprintf("Unexpected internal value %v for flagErrorHandling", m.val_ErrorHandling)
	}
}

type flagValidationRequested struct {
	val_ValidationRequested int
}

// tag to satisfy interfaces

func (flagValidationRequested) isFlag() {}

const (
	validationRequested_Unset = iota
	validationRequested_ValidateSyntax
	validationRequested_ValidateBase
	validationRequested_ValidateFinal
)

// TODO: DOCSTRING

var (
	ErrorUnlessValidSyntax flag = flagValidationRequested{val_ValidationRequested: validationRequested_ValidateSyntax}
	ErrorUnlessValidBase   flag = flagValidationRequested{val_ValidationRequested: validationRequested_ValidateBase}
	ErrorUnlessValidFinal  flag = flagValidationRequested{val_ValidationRequested: validationRequested_ValidateFinal}
)

// Functions and methods that modify errors with data take as input a parameter that controls how already-present data should be handled.
// The options are to prefer the old, prefer the new, err on ambiguity or err is data was there.
// For type-safety, this choice is passed as a parameter of designated type PreviousDataTreatment.
// This file defined this type and its associated methods.

// flagPreviousDataTreatment is an encapsulated enum type passed to functions and methods that modify the data associated to errors.
//
// It controls how the library should treat setting values that are already present.
// We provide [PreferPreviousData], [ReplacePreviousData], [AssertDataIsNotReplaced] as potential values.
// The zero value of this type is not a valid flagPreviousDataTreatment. Using such a zero value will cause panics.
type flagPreviousDataTreatment struct {
	val_PreviousDataTreatment int
}

func (flagPreviousDataTreatment) isFlag() {}

// internal int-based enum for [PreviousDataTreatment]. We use a struct wrapping an int in our exported API.
// This is because we want stronger typing for methods that already take "any" or generic-parameter dependent values.
const (
	treatPreviousData_Unset                       = iota // zero value. This is not a valid value.
	treatPreviousData_Override                           // new value takes precedence
	treatPreviousData_PreferOld                          // old value takes precendence
	treatPreviousData_errorIfPresent                     // error if old value present
	treatPreviousData_errorOnCollision_naive             // error unless old value == new value (and both comparable)
	treatPreviousData_errorOnCollision_comparator        // error unless old value equal to new value (with IsEqual function preferred over ==)
)

var (
	// PreferPreviousData means that when replacing associated data in errors, we keep the old value if some value is already present for a given key.
	PreferPreviousData = flagPreviousDataTreatment{val_PreviousDataTreatment: treatPreviousData_PreferOld}
	// ReplacePreviousData means that when replacing associated data in errors, we unconditionally override already-present values for a given key.
	ReplacePreviousData = flagPreviousDataTreatment{val_PreviousDataTreatment: treatPreviousData_Override}
	// AssertDataIsNotReplaced means that when replacing associated data in errors, we panic if a different value was already present for a given key.
	AssertDataIsNotReplaced = flagPreviousDataTreatment{val_PreviousDataTreatment: treatPreviousData_errorOnCollision_naive}
)

// Only with diagnostics

// String is provided to make PreviousDataTreatment satisfy fmt.Stringer. It returns a string representing the meaning of the value.
func (s flagPreviousDataTreatment) String() string { // Note: Value receiver
	switch s.val_PreviousDataTreatment {
	case treatPreviousData_Unset:
		return "Unset value" // should we panic? I guess not, since this is just for diagnostics.
	case treatPreviousData_Override:
		return "Override old value"
	case treatPreviousData_PreferOld:
		return "Keep previous value"
	case treatPreviousData_errorOnCollision_naive:
		return "Error on ambiguity (naive equality)"
	default:
		// cannot really happen unless users use unsafe, because we don't export the type.
		panic(fmt.Errorf(ErrorPrefix+"invalid value of PreviousDataTreatment : %v", s.val_PreviousDataTreatment))
	}
}

const noFlagRestrictions = 0

type allFlags struct {
	missingData         flagMissingDataTreatment
	errorHandling       errorHandlingBehaviour
	oldData             flagPreviousDataTreatment
	validationRequested flagValidationRequested
}

func (f *allFlags) parseFlags(_ int, flags ...flag) {
	for _, individualFlag := range flags {
		switch individualFlag := individualFlag.(type) {
		case flagMissingDataTreatment:
			f.missingData = individualFlag
		case flagErrorHandling:
			switch individualFlag.val_ErrorHandling {
			case errorHandling_PanicOnAllErrors:
				f.errorHandling.panicOnDataError = true
				f.errorHandling.panicOnValidityError = true
			case errorHandling_ReturnError:
				f.errorHandling.panicOnDataError = false
				f.errorHandling.panicOnValidityError = false
			case errorHandling_PanicOnDataError:
				f.errorHandling.panicOnDataError = true
			case errorHandling_DontPanicOnDataError:
				f.errorHandling.panicOnDataError = false
			case errorHandling_PanicOnValidityError:
				f.errorHandling.panicOnValidityError = true
			case errorHandling_DontPanicOnValidityError:
				f.errorHandling.panicOnValidityError = false
			default:
				panic("cannot happen")
			}
		case flagPreviousDataTreatment:
			f.oldData = individualFlag
		case flagValidationRequested:
			f.validationRequested = individualFlag
		default:
			panic("Cannot happen") // may change
		}
	}
}

*/
