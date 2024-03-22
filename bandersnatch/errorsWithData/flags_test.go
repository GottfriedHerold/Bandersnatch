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
	validFlags_HasData                      []flagArgument = []flagArgument{EnsureDataIsPresent, IgnoreMissingData}
	validFlags_GetData_struct               []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors}
	validFlags_NewErrorWithData_struct      []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, EnsureDataIsNotReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_NewErrorWithData_params      []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, EnsureDataIsNotReplaced, customComparisonFlag, RecoverFromComparisonFunctionPanic, LetComparisonFunctionPanic, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping, MissingDataAsZero, MissingDataIsError}
	validFlags_NewErrorWithData_map                        = validFlags_NewErrorWithData_params
	validFlags_DeleteParameterFromError_any []flagArgument = []flagArgument{ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_DeleteParameterFromError     []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_AsErrorWithData              []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors}
	// validFlags_NewErrorWithData_params_any  []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, EnsureDataIsNotReplaced, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	// validFlags_NewErrorWithData_map_any     []flagArgument = validFlags_NewErrorWithData_params_any
)

var (
	// interface restriction on the flags passed to the corresponding function. Note: We use utils.TypeOfType rather than reflect.TypeOf, because the former works for interfaces as intended.
	validFlagRestrictions map[*[]flagArgument]reflect.Type = map[*[]flagArgument]reflect.Type{
		&validFlags_HasData:                      utils.TypeOfType[flagArgument_HasData](),
		&validFlags_GetData_struct:               utils.TypeOfType[flagArgument_GetData](),
		&validFlags_NewErrorWithData_struct:      utils.TypeOfType[flagArgument_NewErrorStruct](),
		&validFlags_NewErrorWithData_params:      utils.TypeOfType[flagArgument_NewErrorParams](), // Note: This is not part of the functions API, but checked at runtime via type-assertion
		&validFlags_NewErrorWithData_map:         utils.TypeOfType[flagArgument_NewErrorParams](),
		&validFlags_DeleteParameterFromError_any: utils.TypeOfType[flagArgument_DeleteAny](),
		&validFlags_DeleteParameterFromError:     utils.TypeOfType[flagArgument_Delete](),
		&validFlags_AsErrorWithData:              utils.TypeOfType[flagArgument_AsErrorWithData](),
		// &validFlags_NewErrorWithData_params_any:  utils.TypeOfType[flagArgument_NewErrorAny](),
		// &validFlags_NewErrorWithData_map_any:     utils.TypeOfType[flagArgument_NewErrorAny](),
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
	testutils.FatalUnless(t, configCreate.PreferOld() == false, "")
	testutils.FatalUnless(t, configCreate.PreferNew() == true, "")
	testutils.FatalUnless(t, configCreate.PerformEqualityCheck() == false, "")
	testutils.FatalUnless(t, configCreate.checkFun == nil, "") // NOTE: GetCheckFun returns a default function, which we cannot test.
	testutils.FatalUnless(t, configCreate.CatchPanic() == true, "")
	testutils.FatalUnless(t, configCreate.PanicOnAllErrors() == false, "")
	testutils.FatalUnless(t, configCreate.WhatValidationIsRequested() == validationRequest_Syntax, "")
	testutils.FatalUnless(t, configCreate.IsMissingDataError() == true, "")
	testutils.FatalUnless(t, configCreate.AllowEmptyString() == false, "")

	testutils.FatalUnless(t, configSetZero.ModifyData() == false, "") // ModifyData is not defined on errorCreationConfig
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
//   - "PanicOnAllErrors"
//   - "Validation"
//   - "IsMissingDataError"
//   - "AllowEmptyString"
//
// This is a helper function used in testing to ensure that the only thing
func ensureConfigUnchangedExcept(t *testing.T, c1, c2 *errorCreationConfig, changedArgs ...string) {

	// could do a loop over []struct{string, func}, but I don't like that complexity in the test.
	var xPreferOld, xPreferNew, xPerformEqualityCheck, xCatchPanic, xCheckFun, xPanicOnAllErrors, xValidation, xIsMissingDataError, xAllowEmptyString bool

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
		case "PanicOnAllErrors":
			xPanicOnAllErrors = true
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
		b1 := c1.PreferOld()
		b2 := c2.PreferOld()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PreferOld() = %v, c2.PreferOld() = %v ", b1, b2)
	}
	if !xPreferNew {
		b1 := c1.PreferNew()
		b2 := c2.PreferNew()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PreferNew() = %v, c2.PreferNew() = %v ", b1, b2)
	}
	if !xPerformEqualityCheck {
		b1 := c1.PerformEqualityCheck()
		b2 := c2.PerformEqualityCheck()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PerformEqualityCheck() = %v, c2.PerformEqualityCheck() = %v ", b1, b2)
	}
	if !xCatchPanic {
		b1 := c1.CatchPanic()
		b2 := c2.CatchPanic()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.CatchPanic() = %v, c2.CatchPanic() = %v ", b1, b2)
	}
	if !xCheckFun {
		b1 := (c1.checkFun == nil)
		b2 := (c2.checkFun == nil)
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.checkFun is nil: %v, c2.checkFun is nil: %v ", b1, b2)
	}
	if !xPanicOnAllErrors {
		b1 := c1.PanicOnAllErrors()
		b2 := c2.PanicOnAllErrors()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.PanicOnAllErrors() = %v, c2.PanicOnAllErrors() = %v ", b1, b2)
	}
	if !xValidation {
		b1 := c1.WhatValidationIsRequested() // type int
		b2 := c2.WhatValidationIsRequested() // type int
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.WhatValidationIsRequested() = %v, c2.WhatValidationIsRequested() = %v ", b1, b2)
	}
	if !xIsMissingDataError {
		b1 := c1.IsMissingDataError()
		b2 := c2.IsMissingDataError()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.IsMissingDataError() = %v, c2.IsMissingDataError() = %v ", b1, b2)
	}
	if !xAllowEmptyString {
		b1 := c1.AllowEmptyString()
		b2 := c2.AllowEmptyString()
		testutils.FatalUnless(t, b1 == b2, "Unexpected difference in configs: c1.AllowEmptyString() = %v, c2.AllowEmptyString() = %v ", b1, b2)
	}
}

func TestParseFlags(t *testing.T) {
	var c1, c2 errorCreationConfig
	ensureConfigUnchangedExcept(t, &c1, &c2) // sanity check
	c2.preferOld = true
	// ensureConfigUnchangedExcept(t, &c1, &c2) // does fail as expected
	ensureConfigUnchangedExcept(t, &c1, &c2, "PreferOld", "PreferNew")
	c2 = errorCreationConfig{}

	parseFlagArgs[flagArgument](&c1)
	ensureConfigUnchangedExcept(t, &c1, &c2)

	parseFlagArgs(&c1, PreferPreviousData)
	testutils.FatalUnless(t, c1.PreferOld() == true && c1.PreferNew() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PreferOld", "PreferNew")

	parseFlagArgs(&c1, ReplacePreviousData)
	testutils.FatalUnless(t, c1.PreferOld() == false && c1.PreferNew() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2)

	parseFlagArgs(&c1, EnsureDataIsNotReplaced)
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	parseFlagArgs(&c1, PreferPreviousData)
	parseFlagArgs(&c2, PreferPreviousData)
	testutils.FatalUnless(t, c1.PreferOld() == true && c1.PreferNew() == false, "")
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == false, "") // setting PreferPreviousData unsets perform equality check
	parseFlagArgs(&c1, EnsureDataIsNotReplaced)
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.PreferOld() == true && c1.PreferNew() == false, "") // keep last setting
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	parseFlagArgs(&c1, ReplacePreviousData)
	parseFlagArgs(&c2, ReplacePreviousData)
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == false, "") // setting ReplacePreviousData unsets perform equality check
	parseFlagArgs(&c1, EnsureDataIsNotReplaced)
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.PreferOld() == false && c1.PreferNew() == true, "") // keep last setting
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck")

	c1 = errorCreationConfig{}
	c2 = errorCreationConfig{}

	parseFlagArgs(&c1, EnsureDataIsNotReplaced_fun(Comparison_IsEqual))
	testutils.FatalUnless(t, c1.PerformEqualityCheck() == true, "")
	testutils.FatalUnless(t, c1.checkFun != nil, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PerformEqualityCheck", "checkFun")

	parseFlagArgs(&c1, ReplacePreviousData, LetComparisonFunctionPanic)
	testutils.FatalUnless(t, c1.CatchPanic() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "CatchPanic")

	parseFlagArgs(&c1, RecoverFromComparisonFunctionPanic)
	testutils.FatalUnless(t, c1.CatchPanic() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "CatchPanic")

	parseFlagArgs(&c1, MissingDataAsZero)
	testutils.FatalUnless(t, c1.IsMissingDataError() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "IsMissingDataError")

	parseFlagArgs(&c1, MissingDataIsError)
	testutils.FatalUnless(t, c1.IsMissingDataError() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "IsMissingDataError")

	parseFlagArgs(&c1, PanicOnAllErrors)
	testutils.FatalUnless(t, c1.PanicOnAllErrors() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PanicOnAllErrors")

	parseFlagArgs(&c1, ReturnError)
	testutils.FatalUnless(t, c1.PanicOnAllErrors() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "PanicOnAllErrors")

	parseFlagArgs(&c1, AllowEmptyString)
	testutils.FatalUnless(t, c1.AllowEmptyString() == true, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "AllowEmptyString")

	parseFlagArgs(&c1, DefaultToWrapping)
	testutils.FatalUnless(t, c1.AllowEmptyString() == false, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "AllowEmptyString")

	parseFlagArgs(&c1, NoValidation)
	testutils.FatalUnless(t, c1.WhatValidationIsRequested() == validationRequest_NoValidation, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidSyntax)
	testutils.FatalUnless(t, c1.WhatValidationIsRequested() == validationRequest_Syntax, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidBase)
	testutils.FatalUnless(t, c1.WhatValidationIsRequested() == validationRequest_Base, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")

	parseFlagArgs(&c1, ErrorUnlessValidFinal)
	testutils.FatalUnless(t, c1.WhatValidationIsRequested() == validationRequest_Final, "")
	ensureConfigUnchangedExcept(t, &c1, &c2, "Validation")
	c1 = errorCreationConfig{}

	// roundabout way to ensure we get (possibly something wrapping) the function back.
	// Due to incomparability of function types, no true equality check seems possible.
	var called bool = false
	var dummyCheckFun EqualityComparisonFunction = func(x, y any) bool { called = true; return true }
	parseFlagArgs(&c1, EnsureDataIsNotReplaced_fun(dummyCheckFun))
	get_fun := c1.GetCheckFun()
	testutils.FatalUnless(t, called == false, "")
	get_fun(0, 0)
	testutils.FatalUnless(t, called == true, "")

}
