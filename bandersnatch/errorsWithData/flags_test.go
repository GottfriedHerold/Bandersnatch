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
		if strings.HasPrefix(s, "Unrecognized") || strings.HasPrefix(s, "Zero value of flag argument") {
			t.Fatalf("printFlagArg does not handle exported flag %v of type %T correctly. Output is:\n\"%v\"", i, value, s)
		}
	}
}
