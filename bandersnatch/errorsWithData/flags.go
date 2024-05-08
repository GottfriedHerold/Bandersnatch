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
// Should we validate newly created errors (e.g. to ensure there is no syntax mistake in the interpolation string). If yes, how?
// (Validating base errors vs. final errors)
//
// How should we treat actual mistakes in this process? Should we just panic? This may be appropriate for init-level function that create static errors on startup,
// but I general, I am very hesitant to just let a library function panic.

// Implementation-wise, we distinguish between flags passed as arguments to our (public) methods and our (internal) configuration.
// Notably, flag arguments are of (interface) type flagArgument. These are parsed and each flag causes our config struct to change.
// flagArgument is intentionally not exported. We simply provide a set of (all) possible values as exported constants (well, variables, because Go has no const structs).
// These are simply some struct flag{value int} for type-safety.
// Note that different flags *do* actually have a different type. This allows to satisfy sub-interfaces of flagArgument to restrict a method to only take a subset of the
// possible flags, with a compile-time failure for meaningless flags.

// flagArgument is the interface type satisfied by all optional flags that are passed to exported function.
// Note that the API of these exported functions typically takes a more constrained interface than flagArgument to
// further filter the set of allowed flags on a per-function basis.
type flagArgument interface {
	isFlag()        // tag to mark flags
	getValue() int  // returns a enum-style constant flagArg_<foo> that determines the actual value/meaning of the flag.
	String() string // returns a string describing the meaning of the flag. Only used for debugging.
}

// Note: Even though our different flags are realized with different types,
// we have a single list of int values that determine the actual meaning of the flag (rather than a separate list for each type).
// This is done to decouple the type of the exported argument flags (which is just there to restrict methods to only meaningful flags) from their actual meaning.
// This allows changing the type without needed to refactor much.
// Actual flags return the corresponding value by calling flagArgument.getValue()

const (
	flagArg_Unset = iota // The zero value corresponds to an unset value. This should never appear and we panic if it does. Note that since we don't export any concrete type implementing flagArgument, it takes considerable effort to trigger this.

	// For overwriting data, there are really 3 (essential) config items:
	//  - preference for old/new
	//  - should we check for equality
	//  - how should we check for equality (we could use nil here as don't check at all, but nil could also special case plain ==)
	// Note that the old/new preference actually still matters even if we do an equality check, because a custom equality check might not do plain "==". In fact, the default one does not.
	// Still, we simplify the API insofar as that setting any old/new preference unsets the equality check and this is the only way to unset it.
	// Conversely, setting the equality check will honor the last value of the old/new - preference.
	// Also, setting any equality check function will actually request an equality check.
	flagArg_PreferOld                     // prefer old values when overwriting data.
	flagArg_PreferNew                     // prefer new values when overwriting data
	flagArg_AssertEqual                   // assume values are equal (using the default comparison function). Note that this is really two options: the last setting of PreferOld/PreferNew actually still determines preference for old/new. We just present it as a ternary toggle to the user for simplicity.
	flagArg_AssertEqual_fun               // assume values are equal (using a custom comparison function)). Note that a flagArg with this value may have a type that also contains a function pointer in addition to wrapping just this int.
	flagArg_RecoverPanicFromEqualityTests // recover from panic in comparison functions and turn them into mistakes.
	flagArg_PassPanicFromEqualiltyTests   // if a comparison function panics, just forward the panic

	// for missing data, there is just one config item:
	// either silently zero-initialize or zero-initialize and treat it as mistake.
	flagArg_FillWithZeros        // zero-initialize missing value for T's fields when creating an ErrorWithData[T]
	flagArg_MissingDataIsMistake // missing values for T's fields are an mistake when creating an ErrorWithData[T]

	// Since we use a single utility function for checking that data is there and zeroing out the data,
	// flagArg_IgnoreMissingData and flagArg_EnsureDataPresent are kind-of equivalent to
	// flagArg_FillWithZeros and flagArg_MissingDataIsMistake, but used (exclusively) for [HasData].
	// The reason is simply that [HasData] does not actually modify the error and zero out missing data, so the above flags would be a misnomer.
	// (Also, the defaults may be different)

	flagArg_IgnoreMissingData // Treat missing data as zero of appropriate type. Only check that data that is there has the right type. Only valid for [HasData]
	flagArg_EnsureDataPresent // Don't treat missing data as zero. Only valid for [HasData]

	// Is validation requested?
	flagArg_NoValidation   // Don't perform validation
	flagArg_ValidateSyntax // Check syntax of interpolation strings.
	flagArg_ValidateBase   // Validate as a base error (i.e. ignore missing values for $fmt{Var} - expression, as those might be filled in later). Note that we follow the error chain if applicable.
	flagArg_ValidateFinal  // Validate as a final error (this means we ensure all variables referred to by format strings actuall exist). Note that we may follow the error chain if applicable.

	// Mistake handling: Return mistake or panic
	// We might consider separating that according to the type of mistake
	flagArg_PanicOnMistakes // panic on (all) mistakes
	flagArg_ReturnMistakes  // return mistake as last return value

	// Handling empty interpolation strings
	flagArg_AllowEmptyString // an empty interpolation string is just that: an empty string. This allows .Error() to return an empty string.
	flagArg_DefaultToWrapped // an empty interpolation string is interpreted as "refer to base error" (and panic if no base error exists).
)

// fArg is an implementation of flagArgument that just wraps an int (enum-style)
// Note that flagArgument is implemented by the value type fArg itself as opposed to *fArg (which then also implements it, by Go's rules).
//
// the fArg type is not really used directly; our concrete exported flags have various types fArg_Foo, each of which struct-embeds fArg.
type fArg struct {
	val int
}

func (fArg) isFlag()          {} // acts solely as a tag to mark the type as a flag. This is never called.
func (f fArg) getValue() int  { return f.val }
func (f fArg) String() string { return printFlagArg(&f) } // Both &f and f work here.

// the various fArg_Foo types are the concrete types that our exported flags have. Each of these types satisfies the flagArgument interface.
// The actual type depends on the flag. Each flag types implements certain extra functions (which serve no purpose other than to act as additional tags),
// which makes it satisfy a more constrained interface, depening on the flag type.
// This is done to further restrict the set of allowed flags to certain functions at compile-time, if applicable.
type (
	fArg_HasData     struct{ fArg }
	fArg_MissingData struct{ fArg }
	fArg_Validity    struct{ fArg }
	fArg_Panic       struct{ fArg }
	fArg_OldData     struct {
		fArg
		f *EqualityComparisonFunction // using a pointer here means farg_OldData is a comparable type. This simplifies/unfies the testing code. It also simplifies treating the case where f is a closure with non-trivial state.
	}
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
	case flagArg_PassPanicFromEqualiltyTests:
		return "Do not recover panics in equality comparison functions"
	case flagArg_RecoverPanicFromEqualityTests:
		return "Recover from panics in equality comparison functions"

	case flagArg_FillWithZeros:
		return "Zero-initialize missing values for T when creating an ErrorWithData[T]"
	case flagArg_MissingDataIsMistake:
		return "missing values for T's fields are a mistake when creating an ErrorWithData[T]"

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

	case flagArg_PanicOnMistakes:
		return "Panic if a mistake is encountered"
	case flagArg_ReturnMistakes:
		return "Report a mistake rather than panicking if a mistake is encountered (some failure conditions still panic, but those are explicitly documented)"

	case flagArg_AllowEmptyString:
		return "Allow creating errors with empty error message"
	case flagArg_DefaultToWrapped:
		return "Empty interpolation string defaults to wrapping base error (or panic)"

	default:
		return fmt.Sprintf("Unrecognized flag argument with internal value set to %v", v)
	}
}

// config_OldData collects the (internal) flags that determine how old vs new data is handled.
// We require that the zero value of this type corresponds to the default settings.
// Since this couples semantics of internals to the defaults (we would use a preferNew bool to change the default)
// we require all read accesses to go through methods to get an API independent from the defaults.
type config_OldData struct {
	_preferOld      bool                       // prefer old vs. prefer new data. The underscore is due to disambiguate from the method.
	doEqualityCheck bool                       // should we check that data statys the same?
	checkFun        EqualityComparisonFunction // nil means "use default comparison function". This is handled by GetCheckFun(). Note that the defaul is not plain "==".
	passPanic       bool                       // do we recover panics in checkFun?
}

// preferOld reads out the configuration to determine whether data from the base error or newly provided data should take preference.
func (p *config_OldData) preferOld() bool {
	return p._preferOld
}

// preferNew reads out the configuration to determine whether data from the base error or newly provided data should take preference.
// It is just the negation of [preferOld]
func (p *config_OldData) preferNew() bool {
	return !p._preferOld
}

// performEqualityCheck reads out the configuration to determine whether we perform an equality check (with a failure considered a mistake)
// between data from the base error and newly provided data (if both exist).
//
// Note the the equality check is not just done by ==, but rather by calling a custom function, which is obtained by [GetCheckFun]
func (p *config_OldData) performEqualityCheck() bool {
	return p.doEqualityCheck
}

// getCheckFun reads out the configuration to provide the (possibly custom) equality check function.
// This is only meaningful if [performEqualityCheck] returns true.
// If no equality check function was explicitly set, we return a default one.
func (p *config_OldData) getCheckFun() EqualityComparisonFunction {
	// NOTE: This (internal) function should never be called unless p.performEqualityCheck() == true.
	//
	// We do not check this (and panic on fail), because some testing functions actually DO call it with
	// p.PerformEqualityCheck() set to false. The latter choice was done to simplify writing the tests.
	if p.checkFun == nil {
		return Compare_CoerceNilInterface
	} else {
		return p.checkFun
	}
}

// catchPanic reads out the configuration to tell whether we should recover panic's in comparison functions.
// This is only meaningful if [performEqualityCheck] returns true.
func (p *config_OldData) catchPanic() bool {
	return !p.passPanic
}

// config_ErrorHandling collects the internal flags that determine how mistakes during creating of errors are handled.
// Notably, we may just return a mistake or panic, the latter being appropriate for error constants that are created during program initialization.
//
// similar to [config_OldData], reads should go through methods.
type config_ErrorHandling struct {
	panicOnError bool
}

// panicOnAllMistakes returns true if the configuration is set to panic on all errors.
func (p *config_ErrorHandling) panicOnAllMistakes() bool {
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
	validationRequest_Syntax = iota // usually the default
	validationRequest_NoValidation
	validationRequest_Base
	validationRequest_Final
)

// whatValidationIsRequested returns an enum-style int determining the type of requested validation.
//
// The meaning of the returned int-constants is defined by the validationRequest_* constants
func (p *config_Validation) whatValidationIsRequested() int {
	return p.doValidation
}

// config_ImplicitZero collects the internal flags that determine what should be done if data for some error is missing.
// Note that "missing" only makes sense for ErrorWithData[T] rather than ErrorWithData_any and refers to the possibility that some
// data required for a field of T might not be present.
//
// This config determines whether missing data should be *silently* treated as zero or considered a mistake (and still be treated as zero, because there is nothing else we can do)
//
// Similar to [config_OldData], reads should go through methods.
type config_ImplicitZero struct {
	implicitZero bool
	// addMissingData     bool
}

// isMissingDataMistake determines whether missing data is treated as an mistake or silently as a zero value.
func (p *config_ImplicitZero) isMissingDataMistake() bool {
	return !p.implicitZero
}

// config_SetZeros is passed to [ensureCanMakeStructFromParameters] to determine whether the function should actually modify the error or just perform a check.
// Which of these options is appropriate is entirely determined by the call site, so we only ever use constants of this type.
// (think of ensureCanMakeStructFromParamters as two different functions, depending on the setting -- making it a single function was just done to simplify things)
// User-provided flags never affect config_SetZeros.
type config_SetZeros struct {
	setErrorsToZero bool
}

// modifyData is used to query a given [config_SetZeros] as to whether data should be modified or not if data is missing/has the wrong type.
func (p *config_SetZeros) modifyData() bool {
	return p.setErrorsToZero
}

// config_EmtpyString collects the interal flags that determine how empty interpolation strings should be handled.
// We can either treat them as-is or have them default to "refer to the base error" (usually with a panic if there is no base error)
// Since we do not want to encourage empty error messages, we default to "refer to the base error".
//
// Similar to [config_OldData], reads should go through methods.
type config_EmptyString struct {
	allowEmpty bool
}

// allowEmptyString reads out the config whether empty strings are just taken literally.
func (p *config_EmptyString) allowEmptyString() bool {
	return p.allowEmpty
}

// errorCreatingConfig is a struct collecting all our configuration options (except [config_SetZero]).
// The intended usage is to initialize it with defaults and call [parseFlagArgs] on it to modify it according to user-provided flags.
//
// In order to avoid having a multitude of [parseFlagArgs] - variants, [parseFlagArgs] only works with this struct (which contains all config_* types), even if some parts of
// it are meaningless in a given context. Note that the internals of each config type are chosen such that the zero value of errorCreationConfig corresponds to the default value,
// at least when possible (the meaningful default may depend on more than the type in some cases).
type errorCreationConfig struct {
	config_OldData
	config_ErrorHandling
	config_Validation
	config_ImplicitZero
	config_EmptyString
}

// Actually exported flags go here:

// This is the list of optional flags that some functions of our API accept.
// Each such function states the exact list of allowed flags (A subset of this list) in its documentation.
// Flags are passed as variadic arguments, if possible as f(...flags_for_f) where flags_for_f is an interface that only allows the accepted flags.
//
// NOTE: We always parse and process all flags from left-to-right before processing any other arguments. This implies that later flags override earlier ones and
// that flags affect the handling of earlier non-flag arguments, if applicable.
var (
	// PreferPreviousData means that when setting associated data in errors, we keep the old value if some value is already present for a given key. This unsets [MistakeIfDataIsReplaced].
	PreferPreviousData = fArg_OldData{fArg{val: flagArg_PreferOld}, nil}
	// ReplacePreviousData means that when setting associated data in errors, we unconditionally override already-present values for a given key. This is the default. This unsets [MistakeIfDataIsReplaced].
	ReplacePreviousData = fArg_OldData{fArg{val: flagArg_PreferNew}, nil}
	// MistakeIfDataIsReplaced means that when setting associated data in errors, we treat it as a mistake if a value was already present for a given key, unless the values are equal.
	// [MistakeIfDataIsReplaced_fun] may be used to customize this with a custom equality-comparison function.
	// NOTE1: We honor the last setting of [PreferPreviousData] vs. [ReplacePreviousData] (the default). Due the latter two flags also un-setting the equality-checking, the order of flags matters.
	// NOTE2: by default, we recover from a panic in the comparison function (such as using == on values of the same incomparable type), treating it a "unequal" with a custom mistake message determined by the panic value.
	// Use [LetComparisonFunctionPanic] to change that behaviour.
	// NOTE3: The default comparison function selected by this flag is *not* plain comparison with ==. Rather, we special case nil interfaces and treat them as equal to nils of non-interface type.
	// This is done because our ParamMap API to Struct API performs the same conversion. See [Compare_CoerceNilInterface]
	MistakeIfDataIsReplaced = fArg_OldData{fArg{val: flagArg_AssertEqual}, nil}

	// LetComparisonFunctionPanic is only useful if [MistakeIfDataIsReplaced] or [MistakeIfDataIsReplaced_fun] is set.
	// If LetComparsionFunctionPanic is set, a panic in the comparison function is not recovered from and escapes whatever function the user called.
	// In particular, setting of data is aborted at the point of panic (This differs from the usual behaviour of the package, which is to not abort on first mistake)
	// The [ReturnError] or [PanicOnAllErrors] setting does not affect this.
	//
	// This is useful if the external caller needs to handle the actual panic value.
	// Note that just instead setting [PanicOnAllErrors] would first recover(), turn the panic into an mistake, collect all mistakes and raise a new panic.
	LetComparisonFunctionPanic = fArg_OldData{fArg{val: flagArg_PassPanicFromEqualiltyTests}, nil}

	// RecoverFromComparisonFunctionPanic is only meaningful if [MistakeIfDataIsReplaced] or [MistakeIfDataIsReplaced_fun] is set.
	// It causes any panic(X) in the comparsion function (such as using == to compare values of the same incomparable type) to be recover()ed from.
	// We treat this then as a simple "unequal", with the panic value X entering the message contained in the returned mistake err (if X satisfies error, err will wrap X).
	// NOTE: if [PanicOnAllErrors] is set, we will then ultimately call panic(err). However, observe that err's message just prints X using [fmt]; the actual value of X may be lost.
	//
	// As this is the default behaviour, the [RecoverFromComparisonFunctionPanic] flag is never needed.
	RecoverFromComparisonFunctionPanic = fArg_OldData{fArg{val: flagArg_RecoverPanicFromEqualityTests}, nil}

	// MissingDataAsZero is passed to functions to indicate that missing data should be *silently* zero initialized.
	MissingDataAsZero = fArg_MissingData{fArg{val: flagArg_FillWithZeros}}
	// MissingDataIsMistake is passed to functions to indicate that the function should consider it a mistake if data is missing. This is the default.
	// Note: we zero-initalize missing values and do not abort on first mistake; so the only difference to [MissingDataAsZero] is whether the zero-initializing happens silently or not.
	MissingDataIsMistake = fArg_MissingData{fArg{val: flagArg_MissingDataIsMistake}}

	// Due to the way our code is structured, EnsureDataIsPresent and IgnoreMissingData are
	// functionally essentially equivalent to MissingDataAsZero and MissingDataIsMistake.
	// The only difference is really that EnsureDataIsPresent and IgnoreMissingData are the names specifically for [HasData].
	// This distinction was made to have a name for the flags that matches what it does on the exported API. Also, the defaults are different.

	// EnsureDataIsPresent is passed to [HasData] to cause it to not ignore merely missing data. As this is the default, this flag is never needed.
	EnsureDataIsPresent = fArg_HasData{fArg{val: flagArg_EnsureDataPresent}}
	// IgnoreMissingData is (only) passed to [HasData] to cause it to ignore the case of merely missing data. We only type-check existing data in this case.
	IgnoreMissingData = fArg_HasData{fArg{val: flagArg_IgnoreMissingData}}

	// ReturnMistake is passed to functions to indicate that it should report a mistake rather than panic. This is the default.
	// Note that certain (explicitly stated) conditions may still cause a panic.
	ReturnMistake = fArg_Panic{fArg{val: flagArg_ReturnMistakes}}
	// PanicOnAllMistakes is passed to functions to indicate that they should panic on failure. Useful for function calls that initialize global constants on startup.
	PanicOnAllMistakes = fArg_Panic{fArg{val: flagArg_PanicOnMistakes}}

	// Note: The default depends on the function for these 4 flags:

	// NoValidation is passed to functions creating errors to indicate that no validation (pertaining to recursive checking of interpolation strings) is requested.
	NoValidation = fArg_Validity{fArg{val: flagArg_NoValidation}}
	// ErrorUnlessValidSyntax is passed to functions creating errors to indicate that syntax-validation of the interpolation string is requested
	ErrorUnlessValidSyntax = fArg_Validity{fArg{val: flagArg_ValidateSyntax}}
	// ErrorUnlessValidBase is passed to functions creating errors to indicate that recursive validation of interpolation strings is requested.
	// This includes that data referred to actually exists, except for possible $fmtVerb{Param} expressions, which might be filled by a wrapping error.
	ErrorUnlessValidBase = fArg_Validity{fArg{val: flagArg_ValidateBase}}
	// ErrorUnlessValidFinal is passed to functions creating errors to indicate that recursive validation of interpolation strings is requested.
	// This includes checking that all data referred to by %fmtVerb{Param} or $fmtVerb{Param}-expressions actually exists.
	ErrorUnlessValidFinal = fArg_Validity{fArg{val: flagArg_ValidateFinal}}

	// AllowEmptyString needs to be passed to functions creating errors to allow an empty error message.
	// Otherwise, an empty string defaults to %w resp. $w and we consider it an error if there is no base error.
	AllowEmptyString = fArg_EmptyString{fArg{val: flagArg_AllowEmptyString}}

	// DefaultToWrapping is passed to function creating errors to cause an empty interpolation string to default to
	// %w or $w, repeating the wrapped error. If there is no error to wrap, we typically panic.
	// This is the default setting, so there should never be a need to pass this flag explicitly.
	DefaultToWrapping = fArg_EmptyString{fArg{val: flagArg_DefaultToWrapped}}
)

// MistakeIfDataIsReplaced_fun returns a flag that it used to indicate that when potentially replacing associated data in errors,
// we treat it as a mistake if a value was already present for a given key, unless the old value equals the new.
// As opposed to plain [MistakeIfDataIsReplaced], the provided function f is used to test for equality.
//
// See [Compare_CoerceNilInterface] and [Comparison_IsEqual] for useful custom comparison methods.
//
// Calling this with f a nil function will cause a panic.
func MistakeIfDataIsReplaced_fun(f EqualityComparisonFunction) fArg_OldData {
	if f == nil {
		panic(ErrorPrefix + "called MistakeIfDataIsReplaced_fun with nil function")
	}

	// Note: This stores a pointer to the local variable f (which is a copy of what the caller provided) of function type.
	// This extra indirection (from having a pointer-to-function) inside fArg_OldData has basically one single purpose:
	// It avoids making fArg_OldData (and types containing it) incomparable.
	return fArg_OldData{fArg{val: flagArg_AssertEqual_fun}, &f}
}

// specific instance for allFlagArgs below. This is only used in testing, the issue being that
//
//	a) MistakeIfDataIsReplaced_fun(Comparison_IsEqual) == MistakeIfDataIsReplaced_fun(Comparison_IsEqual) may fail and
//	b) our testing framework has a list of all flags, which does not work for a flag factory.
//
// This would require some amount of special-casing for our tests, which is annoying. So we just use this fixed flag instead.
var customComparisonFlag = MistakeIfDataIsReplaced_fun(Comparison_IsEqual)

// allFlagArgs is a list of all possible flag argument values above (and possible outputs of functions generating flagArguments)
// This is only used for testing, but defined here to simplify refactoring, as it's tied to the above list of definitions.
var allFlagArgs []flagArgument = []flagArgument{
	PreferPreviousData,
	ReplacePreviousData,
	MistakeIfDataIsReplaced,
	MissingDataAsZero,
	MissingDataIsMistake,
	IgnoreMissingData,
	EnsureDataIsPresent,
	ReturnMistake,
	PanicOnAllMistakes,
	NoValidation,
	ErrorUnlessValidSyntax,
	ErrorUnlessValidBase,
	ErrorUnlessValidFinal,
	AllowEmptyString,
	DefaultToWrapping,
	RecoverFromComparisonFunctionPanic,
	LetComparisonFunctionPanic,
	customComparisonFlag,
}

// parseFlagArgs_HasData parses the provided flags and constructs a config_ImplicitZero from this.
//
// The flags are parsed in order and overwrite default or previous settings.
// Consequently, when conflicting flags are provided, the last ones take precedence.
//
// Currently accepted flags are [IgnoreMissingData] and [EnsureDataIsPresent] (default)
// NOTE: For the query-only HasData, the seemingly useless [IgnoreMissingData] setting has the purpose of type-checking the existing data.
func parseFlagArgs_HasData(flags ...flagArgument_HasData) (ret config_ImplicitZero) {
	ret = config_ImplicitZero{implicitZero: false}
	for _, flag := range flags {
		if flag == nil {
			panic(fmt.Errorf(ErrorPrefix + "nil passed as a flag to HasData"))
		}
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

// parseFlagArgs_GetData parses the provided flags and constructs a config_ImplicitZero and config_ErrorHandling from this.
//
// The flags are parsed in order and overwrite default or previous settings.
// Consequently, when conflicting flags are provided, the last ones take precedence.
//
// Currently accepted flags are
// - [MissingDataAsZero] or [MissingDataIsMistake] (default)
// - [ReturnMistake] (default) or [PanicOnAllMistakes]
func parseFlagArgs_GetData(flags ...flagArgument_GetData) (retZeroFill config_ImplicitZero, retPanic config_ErrorHandling) {

	// no-ops, but written here to make the defaults explicit.
	retPanic = config_ErrorHandling{panicOnError: false}
	retZeroFill = config_ImplicitZero{implicitZero: false}

	for _, flag := range flags {
		if flag == nil {
			panic(fmt.Errorf(ErrorPrefix + "nil passed as flag to GetData"))
		}
		switch v := flag.getValue(); v {
		case flagArg_Unset:
			panic("Cannot happen") // unless the user tries hard
		case flagArg_MissingDataIsMistake:
			retZeroFill.implicitZero = false
		case flagArg_FillWithZeros:
			retZeroFill.implicitZero = true
		case flagArg_PanicOnMistakes:
			retPanic.panicOnError = true
		case flagArg_ReturnMistakes:
			retPanic.panicOnError = false
		default:
			panic("Cannot happen")
		}
	}
	return
}

// Needs to be generic function due to Go's lack of covariance.
// Note that the ArgType that is used when we call this is actually an *interface* type (unusal for generics) refining flagArgument.

// parseFlagArgs parses the passed flags in order and modifies *p according to each flag.
// Note that ArgType will typically be an interface type that refines flagArgument, which may restrict the set of allowed flags.
func parseFlagArgs[ArgType flagArgument](p *errorCreationConfig, flags ...ArgType) {
	for _, individualFlag := range flags {
		// My code analysis tool (which uses go vet with some non-default build flags via gopls) thinks this condition is impossible to trigger.
		// The tool is actually wrong here (because ArgType is an interface type itself).
		// (Bug reported and confirmed with the go tools)
		if any(individualFlag) == nil {
			panic(fmt.Errorf(ErrorPrefix + "Passed nil as a flag to a function taking a variadic number of flag arguments"))
		}
		switch v := individualFlag.getValue(); v {
		case flagArg_Unset:
			panic("Cannot happen") // unless the user tries hard
		case flagArg_PreferOld:
			p._preferOld = true
			p.doEqualityCheck = false
			p.checkFun = nil
		case flagArg_PreferNew:
			p._preferOld = false
			p.doEqualityCheck = false
			p.checkFun = nil
		case flagArg_AssertEqual:
			p.doEqualityCheck = true
			p.checkFun = nil
		case flagArg_PassPanicFromEqualiltyTests:
			p.passPanic = true
		case flagArg_RecoverPanicFromEqualityTests:
			p.passPanic = false
		case flagArg_AssertEqual_fun:
			p.doEqualityCheck = true
			p.checkFun = *(flagArgument(individualFlag).(fArg_OldData).f)
		case flagArg_MissingDataIsMistake:
			p.implicitZero = false
		case flagArg_FillWithZeros:
			p.implicitZero = true
		case flagArg_ReturnMistakes:
			p.panicOnError = false
		case flagArg_PanicOnMistakes:
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

// Differentiate the various fArg-types according to their *usage* by defining an interface for each usage scenario (i.e. exported API function).

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

type flagArgument_JoinAny interface {
	flagArgument
	isFlag_JoinAny()
}

type flagArgument_Join interface {
	flagArgument
	isFlag_Join()
}

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

func (fArg_OldData) isFlag_JoinAny() {}
func (fArg_Panic) isFlag_JoinAny()   {}

func (fArg_OldData) isFlag_Join()     {}
func (fArg_Panic) isFlag_Join()       {}
func (fArg_MissingData) isFlag_Join() {}

func (fArg_Panic) isFlag_DeleteAny()       {}
func (fArg_Validity) isFlag_DeleteAny()    {}
func (fArg_EmptyString) isFlag_DeleteAny() {}

func (fArg_MissingData) isFlag_Delete() {}
func (fArg_Panic) isFlag_Delete()       {}
func (fArg_Validity) isFlag_Delete()    {}
func (fArg_EmptyString) isFlag_Delete() {}

func (fArg_MissingData) isFlag_AsErrorWithData() {}
func (fArg_Panic) isFlag_AsErrorWithData()       {}

// flagArgument_NewErrorAny handled by type alias
