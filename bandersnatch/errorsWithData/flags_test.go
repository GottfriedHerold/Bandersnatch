package errorsWithData

import (
	"reflect"
	"strings"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var (
	// list of all exported functions that take flags and a full list of all flags taken for each.
	// Note that validFlagRestrictions needs an entry for each variable here to specify restrictions.
	// NOTE: customComparisonFlag must be used instead of [MistakeIfDataIsReplaced_fun].
	validFlags_HasData                      []flagArgument = []flagArgument{EnsureDataIsPresent, IgnoreMissingData}
	validFlags_GetData_struct               []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsMistake, ReturnMistake, PanicOnAllMistakes}
	validFlags_NewErrorWithData_struct      []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_NewErrorWithData_params      []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping, MissingDataAsZero, MissingDataIsMistake}
	validFlags_NewErrorWithData_map                        = validFlags_NewErrorWithData_params
	validFlags_DeleteParameterFromError_any []flagArgument = []flagArgument{ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_DeleteParameterFromError     []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsMistake, ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_AsErrorWithData              []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsMistake, ReturnMistake, PanicOnAllMistakes}
	validFlags_NewErrorWithData_params_any  []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_NewErrorWithData_map_any     []flagArgument = validFlags_NewErrorWithData_params_any
	validFlags_JoinAny                      []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes}
	validFlags_Join                         []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes, MissingDataAsZero, MissingDataIsMistake}
	validFlags_AddErrorToData               []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, MistakeIfDataIsReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnMistake, PanicOnAllMistakes, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal}
)

var (
	// interface restriction on the flags passed to the corresponding function. Note: We use utils.TypeOfType rather than reflect.TypeOf, because the former works for interfaces as intended.
	validFlagRestrictions map[*[]flagArgument]reflect.Type = map[*[]flagArgument]reflect.Type{
		&validFlags_HasData:                      utils.TypeOfType[flagArgument_HasData](),
		&validFlags_GetData_struct:               utils.TypeOfType[flagArgument_GetData](),
		&validFlags_NewErrorWithData_struct:      utils.TypeOfType[flagArgument_NewErrorStruct](),
		&validFlags_NewErrorWithData_params:      utils.TypeOfType[flagArgument_NewErrorParams](), // Note: This is not part of the functions API, but checked at runtime via type-assertion
		&validFlags_NewErrorWithData_map:         utils.TypeOfType[flagArgument_NewErrorParams](),
		&validFlags_DeleteParameterFromError_any: utils.TypeOfType[flagArgument_DeleteAny](), // Checked at runtime
		&validFlags_DeleteParameterFromError:     utils.TypeOfType[flagArgument_Delete](),    // Checked at runtime
		&validFlags_AsErrorWithData:              utils.TypeOfType[flagArgument_AsErrorWithData](),
		&validFlags_NewErrorWithData_params_any:  utils.TypeOfType[flagArgument_NewErrorAny](), // Checked at runtime
		&validFlags_NewErrorWithData_map_any:     utils.TypeOfType[flagArgument_NewErrorAny](),
		&validFlags_JoinAny:                      utils.TypeOfType[flagArgument_JoinAny](),
		&validFlags_Join:                         utils.TypeOfType[flagArgument_Join](),
		&validFlags_AddErrorToData:               utils.TypeOfType[flagArgument_AddErrorToData](),
	}
)

func TestOnlyValidFlagsAccepted(t *testing.T) {
	var i = 0 // to simplify identifying which entry was failing.
	for key, typeRestriction := range validFlagRestrictions {

		testutils.FatalUnless(t, typeRestriction.Kind() == reflect.Interface, "validFlagRestrictions contains non-interface type")
		var flagList []flagArgument = *key
		for _, flag := range flagList {
			// -- does not work before Go1.20  because flagArgument is not comparable; of course, embedding comparable in flagArgument does not work either, because of Go's type system.
			testutils.FatalUnless(t, utils.ElementInList(flag, allFlagArgs), "flag %v not in allFlagArgs", flag)
		}
		for _, flag := range allFlagArgs {
			expectedYes := utils.ElementInList(flag, flagList)
			flagType := reflect.TypeOf(flag)
			assignable := flagType.AssignableTo(typeRestriction)
			testutils.FatalUnless(t, expectedYes == assignable, "Flag assignability not as expected for \"%v\" and %v. Iteration count = %v", flag, typeRestriction, i)
		}
		i++
	}
}

func TestPrintFlag(t *testing.T) {
	for i, value := range allFlagArgs {
		s := printFlagArg(value)
		s2 := value.String()
		testutils.FatalUnless(t, s == s2, "")
		if strings.HasPrefix(s, "Unrecognized") || strings.HasPrefix(s, "Zero value of flag argument") {
			t.Fatalf("printFlagArg does not handle exported flag %v of type %T correctly. Output is:\n\"%v\"", i, value, s)
		}
	}
	s := fArg{}.String() // meta-test for the checks above
	testutils.FatalUnless(t, strings.HasPrefix(s, "Zero value of flag argument"), "")
	s = fArg{val: 13513512}.String()
	testutils.FatalUnless(t, strings.HasPrefix(s, "Unrecognized"), "")
}

func TestConfigDefaults(t *testing.T) {
	var configCreate errorCreationConfig
	var configSetZero config_SetZeros
	testutils.FatalUnless(t, configCreate.preferOld() == false, "")
	testutils.FatalUnless(t, configCreate.preferNew() == true, "")
	testutils.FatalUnless(t, configCreate.performEqualityCheck() == false, "")
	testutils.FatalUnless(t, configCreate.checkFun == nil, "") // NOTE: GetCheckFun returns a default function, which we cannot test.
	testutils.FatalUnless(t, configCreate.catchPanic() == true, "")
	testutils.FatalUnless(t, configCreate.panicOnAllMistakes() == false, "")
	testutils.FatalUnless(t, configCreate.whatValidationIsRequested() == validationRequest_Syntax, "")
	testutils.FatalUnless(t, configCreate.isMissingDataMistake() == true, "")
	testutils.FatalUnless(t, configCreate.allowEmptyString() == false, "")

	testutils.FatalUnless(t, configSetZero.modifyData() == false, "") // ModifyData is not defined on errorCreationConfig
}

// helper function

// ensureConfigUnchangedExecpt ensures that the configs given by *c1 and *c2 do not differ in except for the queries specified changedArgs
//
// changedArgs must only contain the following strings
//   - "PreferOld"
//   - "PreferNew"
//   - "PerformEqualityCheck"
//   - "CatchPanic"
//   - "checkFun" -- if not set, only checks that *c1 and *c2 are both non-nil or both nil
//   - "PanicOnAllMistakes"
//   - "Validation"
//   - "IsMissingDataError"
//   - "AllowEmptyString"
//
// This is a helper function used in testing to ensure that the only thing
func ensureConfigUnchangedExcept(t *testing.T, c1, c2 *errorCreationConfig, changedArgs ...string) {

	// could do a loop over []struct{string, func}, but I don't like that complexity in the test.
	var xPreferOld, xPreferNew, xPerformEqualityCheck, xCatchPanic, xCheckFun, xPanicOnAllMistakes, xValidation, xIsMissingDataError, xAllowEmptyString bool

	for _, s := range changedArgs {
		switch s {
		case "PreferOld":
			xPreferOld = true
		case "PreferNew":
			xPreferNew = true
		case "PerformEqualityCheck":
			xPerformEqualityCheck = true
		case "CatchPanic":
			xCatchPanic = true
		case "checkFun":
			xCheckFun = true
		case "PanicOnAllMistakes":
			xPanicOnAllMistakes = true
		case "Validation":
			xValidation = true
		case "IsMissingDataError":
			xIsMissingDataError = true
		case "AllowEmptyString":
			xAllowEmptyString = true
		default:
			panic("Unrecognized string")
		}
	}
	if !xPreferOld {
		b1 := c1.preferOld()
		b2 := c2.preferOld()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PreferOld() = %v, c2.PreferOld() = %v ", b1, b2)
	}
	if !xPreferNew {
		b1 := c1.preferNew()
		b2 := c2.preferNew()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PreferNew() = %v, c2.PreferNew() = %v ", b1, b2)
	}
	if !xPerformEqualityCheck {
		b1 := c1.performEqualityCheck()
		b2 := c2.performEqualityCheck()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PerformEqualityCheck() = %v, c2.PerformEqualityCheck() = %v ", b1, b2)
	}
	if !xCatchPanic {
		b1 := c1.catchPanic()
		b2 := c2.catchPanic()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.CatchPanic() = %v, c2.CatchPanic() = %v ", b1, b2)
	}
	if !xCheckFun {
		b1 := (c1.checkFun == nil)
		b2 := (c2.checkFun == nil)
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.checkFun is nil: %v, c2.checkFun is nil: %v ", b1, b2)
	}
	if !xPanicOnAllMistakes {
		b1 := c1.panicOnAllMistakes()
		b2 := c2.panicOnAllMistakes()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PanicOnAllMistakes() = %v, c2.PanicOnAllMistakes() = %v ", b1, b2)
	}
	if !xValidation {
		b1 := c1.whatValidationIsRequested() // type int
		b2 := c2.whatValidationIsRequested() // type int
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.WhatValidationIsRequested() = %v, c2.WhatValidationIsRequested() = %v ", b1, b2)
	}
	if !xIsMissingDataError {
		b1 := c1.isMissingDataMistake()
		b2 := c2.isMissingDataMistake()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.IsMissingDataError() = %v, c2.IsMissingDataError() = %v ", b1, b2)
	}
	if !xAllowEmptyString {
		b1 := c1.allowEmptyString()
		b2 := c2.allowEmptyString()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.AllowEmptyString() = %v, c2.AllowEmptyString() = %v ", b1, b2)
	}
}

func TestParseFlags(t *testing.T) {
	var c1, c2 errorCreationConfig
	ensureConfigUnchangedExcept(t, &c1, &c2) // sanity check
	c2._preferOld = true
	// ensureConfigUnchangedExcept(t, &c1, &c2) // does fail as expected
	ensureConfigUnchangedExcept(t, &c1, &c2, "PreferOld", "PreferNew")
	c2 = errorCreationConfig{}

	parseFlagArgs[flagArgument](&c1)
	ensureConfigUnchangedExcept(t, &c1, &c2)

	parseFlagArgs(&c1, PreferPreviousData)
	testutils.FatalUnless(t, c1.preferOld() == true && c1.preferNew() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PreferOld", "PreferNew")

	parseFlagArgs(&c1, ReplacePreviousData)
	testutils.FatalUnless(t, c1.preferOld() == false && c1.preferNew() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2)

	parseFlagArgs(&c1, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, c1.performEqualityCheck() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	parseFlagArgs(&c1, PreferPreviousData)
	parseFlagArgs(&c2, PreferPreviousData)
	testutils.FatalUnless(t, c1.preferOld() == true && c1.preferNew() == false, "")
	testutils.FatalUnless(t, c1.performEqualityCheck() == false, "") // setting PreferPreviousData unsets perform equality check
	parseFlagArgs(&c1, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, c1.performEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.preferOld() == true && c1.preferNew() == false, "") // keep last setting
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	parseFlagArgs(&c1, ReplacePreviousData)
	parseFlagArgs(&c2, ReplacePreviousData)
	testutils.FatalUnless(t, c1.performEqualityCheck() == false, "") // setting ReplacePreviousData unsets perform equality check
	parseFlagArgs(&c1, MistakeIfDataIsReplaced)
	testutils.FatalUnless(t, c1.performEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.preferOld() == false && c1.preferNew() == true, "") // keep last setting
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	c1 = errorCreationConfig{}
	c2 = errorCreationConfig{}

	parseFlagArgs(&c1, MistakeIfDataIsReplaced_fun(Comparison_IsEqual))
	testutils.FatalUnless(t, c1.performEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.checkFun != nil, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck", "checkFun")

	parseFlagArgs(&c1, ReplacePreviousData, LetComparisonFunctionPanic)
	testutils.FatalUnless(t, c1.catchPanic() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "CatchPanic")

	parseFlagArgs(&c1, RecoverFromComparisonFunctionPanic)
	testutils.FatalUnless(t, c1.catchPanic() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "CatchPanic")

	parseFlagArgs(&c1, MissingDataAsZero)
	testutils.FatalUnless(t, c1.isMissingDataMistake() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "IsMissingDataError")

	parseFlagArgs(&c1, MissingDataIsMistake)
	testutils.FatalUnless(t, c1.isMissingDataMistake() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "IsMissingDataError")

	parseFlagArgs(&c1, PanicOnAllMistakes)
	testutils.FatalUnless(t, c1.panicOnAllMistakes() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PanicOnAllMistakes")

	parseFlagArgs(&c1, ReturnMistake)
	testutils.FatalUnless(t, c1.panicOnAllMistakes() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PanicOnAllMistakes")

	parseFlagArgs(&c1, AllowEmptyString)
	testutils.FatalUnless(t, c1.allowEmptyString() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "AllowEmptyString")

	parseFlagArgs(&c1, DefaultToWrapping)
	testutils.FatalUnless(t, c1.allowEmptyString() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "AllowEmptyString")

	parseFlagArgs(&c1, NoValidation)
	testutils.FatalUnless(t, c1.whatValidationIsRequested() == validationRequest_NoValidation, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidSyntax)
	testutils.FatalUnless(t, c1.whatValidationIsRequested() == validationRequest_Syntax, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidBase)
	testutils.FatalUnless(t, c1.whatValidationIsRequested() == validationRequest_Base, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidFinal)
	testutils.FatalUnless(t, c1.whatValidationIsRequested() == validationRequest_Final, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")
	c1 = errorCreationConfig{}

	// roundabout way to ensure we get (possibly something wrapping) the function back.
	// Due to incomparability of function types, no true equality check seems possible.
	var called bool = false
	var dummyCheckFun EqualityComparisonFunction = func(x, y any) bool { called = true; return true }
	parseFlagArgs(&c1, MistakeIfDataIsReplaced_fun(dummyCheckFun))
	get_fun := c1.getCheckFun()
	testutils.FatalUnless(t, called == false, "")
	get_fun(0, 0)
	testutils.FatalUnless(t, called == true, "")

}

func TestParseFlagArgs_HasData(t *testing.T) {
	// make sure parseFlagArgs_HasData actually handles all possible flags.
	for _, flagGen := range validFlags_HasData {
		flag := flagGen.(flagArgument_HasData) // if this panics, then TestOnlyValidFlagsAccepted should also fail.
		_ = parseFlagArgs_HasData(flag)        // The test is that this does not panic (i.e. the switch-statement in the function is exhaustive)
	}
	c1 := parseFlagArgs_HasData(IgnoreMissingData)
	testutils.FatalUnless(t, c1.isMissingDataMistake() == false, "")
	c2 := parseFlagArgs_HasData(EnsureDataIsPresent)
	testutils.FatalUnless(t, c2.isMissingDataMistake() == true, "")

	// test default
	c3 := parseFlagArgs_HasData()
	testutils.FatalUnless(t, c3.isMissingDataMistake() == true, "")

	// test multiple arguments

	c4 := parseFlagArgs_HasData(IgnoreMissingData, EnsureDataIsPresent, IgnoreMissingData)
	testutils.FatalUnless(t, c4.isMissingDataMistake() == false, "")

	didPanic := testutils.CheckPanic(func() { parseFlagArgs_HasData(nil) })
	testutils.FatalUnless(t, didPanic == true, "")
}

func TestParseFlagArgs_GetData(t *testing.T) {
	for _, flagGeneral := range validFlags_GetData_struct {
		flag := flagGeneral.(flagArgument_GetData) // if this panics, then TestOnlyValidFlagsAccepted should fail.
		_, _ = parseFlagArgs_GetData(flag)         // The relevant test is that this does not panic (i.e. the switch-statement in the function is exhaustive)
	}
	zf1, p1 := parseFlagArgs_GetData(MissingDataAsZero)
	testutils.FatalUnless(t, zf1.isMissingDataMistake() == false, "")
	testutils.FatalUnless(t, p1.panicOnAllMistakes() == false, "")
	zf2, p2 := parseFlagArgs_GetData(MissingDataIsMistake)
	testutils.FatalUnless(t, zf2.isMissingDataMistake() == true, "")
	testutils.FatalUnless(t, p2.panicOnAllMistakes() == false, "")

	zf3, p3 := parseFlagArgs_GetData(PanicOnAllMistakes)
	testutils.FatalUnless(t, zf3.isMissingDataMistake() == true, "")
	testutils.FatalUnless(t, p3.panicOnAllMistakes() == true, "")

	zf4, p4 := parseFlagArgs_GetData(ReturnMistake)
	testutils.FatalUnless(t, zf4.isMissingDataMistake() == true, "")
	testutils.FatalUnless(t, p4.panicOnAllMistakes() == false, "")

	zfDefault, pDefault := parseFlagArgs_GetData()
	testutils.FatalUnless(t, zfDefault.isMissingDataMistake() == true, "")
	testutils.FatalUnless(t, pDefault.panicOnAllMistakes() == false, "")

	zf5, p5 := parseFlagArgs_GetData(PanicOnAllMistakes, MissingDataAsZero, MissingDataIsMistake, ReturnMistake, ReturnMistake, PanicOnAllMistakes, MissingDataIsMistake, MissingDataAsZero)
	testutils.FatalUnless(t, zf5.isMissingDataMistake() == false, "")
	testutils.FatalUnless(t, p5.panicOnAllMistakes() == true, "")

	didPanic := testutils.CheckPanic(func() { parseFlagArgs_GetData(nil) })
	testutils.FatalUnless(t, didPanic == true, "")
}

func TestMistakeIfDataIsReplaced_fun(t *testing.T) {
	testutils.FatalUnless(t, testutils.CheckPanic(MistakeIfDataIsReplaced_fun, nil) == true, "EnsureDataIsNotReplace_fun(nil) does not panic")

	var state int
	f := func(x, y any) bool {
		state += 1
		return true
	}
	flag := MistakeIfDataIsReplaced_fun(f)
	(*flag.f)(nil, nil) // ensure that flag actually contains f
	testutils.FatalUnless(t, state == 1, "")

}
