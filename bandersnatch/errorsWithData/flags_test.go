package errorsWithData

import (
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var (
	// list of all exported functions that takes flag and a full list of all flags taken for each.
	// Note that validFlagRestrictions needs an entry for each variable here to specify restrictions.
	validFlags_HasData                 []flagArgument = []flagArgument{EnsureDataIsPresent, IgnoreMissingData}
	validFlags_GetData_struct          []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors}
	validFlags_NewErrorWithData_struct []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, AssertDataIsNotReplaced, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping}
	validFlags_NewErrorWithData_params []flagArgument = []flagArgument{PreferPreviousData, ReplacePreviousData, AssertDataIsNotReplaced, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal, AllowEmptyString, DefaultToWrapping, MissingDataAsZero, MissingDataIsError}
	validFlags_NewErrorWithData_map                   = validFlags_NewErrorWithData_params
	validFlags_DeleteParam_any         []flagArgument = []flagArgument{ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal}
	validFlags_DeleteParam_T           []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors, NoValidation, ErrorUnlessValidSyntax, ErrorUnlessValidBase, ErrorUnlessValidFinal}
	validFlags_AsErrorWithData_T       []flagArgument = []flagArgument{MissingDataAsZero, MissingDataIsError, ReturnError, PanicOnAllErrors}
)

var (
	// interface restriction on the flags passed to the corresponding function. Note: We use utils.TypeOfType rather than reflect.TypeOf, because the former works for interfaces as intended.
	validFlagRestrictions map[*[]flagArgument]reflect.Type = map[*[]flagArgument]reflect.Type{
		&validFlags_HasData:                 utils.TypeOfType[flagArgument_HasData](),
		&validFlags_GetData_struct:          utils.TypeOfType[flagArgument_GetData](),
		&validFlags_NewErrorWithData_struct: utils.TypeOfType[flagArgument_NewErrorStruct](),
		&validFlags_NewErrorWithData_params: utils.TypeOfType[flagArgument_NewErrorParams](), // Note: This is not part of the functions API, but checked at runtime via type-assertion
		&validFlags_NewErrorWithData_map:    utils.TypeOfType[flagArgument_NewErrorParams](),
		&validFlags_DeleteParam_any:         utils.TypeOfType[flagArgument_DeleteAny](),
		&validFlags_DeleteParam_T:           utils.TypeOfType[flagArgument_Delete](),
		&validFlags_AsErrorWithData_T:       utils.TypeOfType[flagArgument_AsErrorWithData](),
	}
)
