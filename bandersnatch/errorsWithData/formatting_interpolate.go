package errorsWithData

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Parsing and Interpolating format strings goes through multiple steps:
//
//   - Tokenize the interpolation string
//   - Parse the tokenized string into a syntax tree
//   - [Optional] Perform some validity checks. (3 subchecks, actually. Those would be checked when actually producing output anyway, but sometime we want those checks early)
//   - Actually produce the interpolated error string.

// This file contains the code for the last 2 steps.

// For the validity checks, we have
//
//  - VerifySyntax() error
//  - VerifyParameters_direct(parameters_direct paramMap, baseError error) error
//  - VerifyParameters_passed(parameters_direct paramMap, parameters_passed paramMap, baseError error) error
//
// Each of these checks subsumes the checks above it and requires more "context".
// If there was an error in make_ast, the error is just repeated
//
//  - VerifySyntax makes only some basic check if the (parsed) interpolation string is potentially meaningful and catches
//     - format strings verbs cannot contain literal %
//     - VariableNames must be exported Go identifiers (or denote the paramter map)
//     - Conditions after %! or $! must be recognized by our language.
//  - VerifyParameters_direct furthermore checks that:
//     - %w or $w is only present if there is actually a non-nil wrapped error and, for $w, supports this.
//     - variables referred to by %fmtString{VariableName} actually exist in parameters_direct
//     The condition in %!COND{...} is evaluated for this purpose and failures (apart from VerifySyntax-failures) are ignored in a non-taken sub-tree.
//     $!COND{...} is only Syntax-checked
//  - VerifyParameters_passed furthermore checks that:
//     - variables referred to by $fmtString{VariableName} actually exist in paramters_passed
//     The conditions in both %!COND{...} and $!COND{...} are evaluated for this purpose. Failures (apart from VerifySyntax-failures) are ignored in a non-taken sub-tree.

// Note that even VerifyParameters_passed does not guarantee that Interpolation works, because the format verb might be invalid for the given type or a custom String or Format method might even panic.
// For the latter, note that the fmt package actually recovers from such panics and reports it in the output string. This is beyond the scope of this package.

// Interpolate actually produces the required output string.
// For reasons of efficiency, the Interpolate-functions here do not return a string, but rather take a strings.Builder argument and append to that.

// valid entries for Condition strings
var validConditions [2]string = [2]string{ConditionEmptyMap, ConditionNonEmptyMap}
var validMapSelectors [4]string = [4]string{"m", "map", "parameters", "params"}

// Validation stages:
// 1.) syntactic
// 2.) % valid
// 3.) $ valid (fmt strings not checked)

// NOTE: Assumes a was created with make_ast
// Furthermore, we assume that all calls go through ast_root. In particular, parse errors are caught by ast_root and we never recurse the tree.

// For parameters_passed, note that we make a distinction between nil and empty map:
// parameters_passed == nil means that parameters_passed is the very same as parameters_direct and we do not use that mechanic.
// This means we interpret it (mostly) as parameters_passed == parameters_direct, but in some case produce more accurate error messages.

func (a ast_root) VerifySyntax() (err error) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}
	if a.parseError != nil {
		return a.parseError
	}
	return a.ast.VerifySyntax()
}

func (a ast_root) VerifyParameters_direct(parameters_direct ParamMap, baseError error) error {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}
	if a.parseError != nil {
		return a.parseError
	}

	return a.ast.VerifyParameters_direct(parameters_direct, baseError)
}

func (a ast_root) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) error {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}
	if a.parseError != nil {
		return a.parseError
	}
	return a.ast.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
}

func (a ast_root) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}
	if parameters_passed == nil {
		a.ast.Interpolate(parameters_direct, parameters_direct, baseError, s)
	} else {
		a.ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

	if a.parseError != nil {
		if baseError != nil {
			s.WriteString("\nBase error:\n")
			s.WriteString(baseError.Error()) // Note: We don't check for baseError.(ErrorInterpolater), because we output the parameters anyway.
		}
		if len(parameters_direct) != 0 {
			s.WriteString("\nParameters in error:\n")
			fmt.Fprintf(s, "%v", parameters_direct)
		}
		if parameters_passed != nil {
			s.WriteString("\nParameters from outer error:\n")
			fmt.Fprintf(s, "%v", parameters_passed)
		}
	}
}

func (a ast_list) VerifySyntax() (err error) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		err = ast.VerifySyntax()
		if err != nil {
			return
		}
	}
	return nil
}

func (a ast_list) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		err = ast.VerifyParameters_direct(parameters_direct, baseError)
		if err != nil {
			return
		}
	}
	return nil
}

func (a ast_list) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		err = ast.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		if err != nil {
			return
		}
	}
	return nil
}

func (a ast_list) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}
}

func (a ast_string) VerifySyntax() error {
	return nil
}

func (a ast_string) VerifyParameters_direct(ParamMap, error) error {
	return nil
}

func (a ast_string) VerifyParameters_passed(ParamMap, ParamMap, error) error {
	return nil
}

func (a ast_string) Interpolate(_ ParamMap, _ ParamMap, _ error, s *strings.Builder) {
	s.WriteString(string(a)) // NOTE: need string(a), not a.String() here; the latter would add literal "-marks.
}

func (abase *base_ast_fmt) VerifySyntax() error {
	if strings.ContainsRune(abase.formatString, '%') {
		return fmt.Errorf(ErrorPrefix+`Interpolation string contains a format string verb %s, which contains  '%%'. This will not work with the fmt package`, abase.formatString)
	}
	if abase.invalidParse { // should be detected at root
		panic(ErrorPrefix + "Invalidly parsed interpolation string not detected at root")
	}
	if abase.variableName == "" {
		panic(ErrorPrefix + "Uninitialized variable name") // ought to have been caught by the parser.
	}

	switch abase.variableName {
	case "m", "map", "parameters", "params":
		return nil
	default:
		if !token.IsIdentifier(abase.variableName) {
			return fmt.Errorf(ErrorPrefix+"Variable name in interpolation string is not a valid Go identifier. The offending variable name was: %s", abase.variableName)
		}
		if !token.IsExported(abase.variableName) {
			return fmt.Errorf(ErrorPrefix+"Variable name %s in interpolation string is unexported. This does not work", abase.variableName)
		}
		return nil
	}
}

// joint helper for ast_fmtPercent and ast_fmtDollar

func (a *base_ast_fmt) _Interpolate(parameters_relevant ParamMap, s *strings.Builder, PercentOrDollar rune) {
	var value any
	var ok bool = true
	switch a.variableName {
	case "m", "map", "parameters", "params":
		value = parameters_relevant
		if parameters_relevant == nil {
			value = make(ParamMap) // nil -> empty map. This should not happen, but better safe than sorry.
		}
	default:
		value, ok = parameters_relevant[a.variableName]
	}
	if a.formatString == "" {
		panic(ErrorPrefix + "Empty format string. This cannot happen")
	}

	if !ok {
		s.WriteString(`%` + a.formatString + `!<missing value>`)
	} else {
		fmt.Fprintf(s, "%"+a.formatString, value) // NOTE: a.formatString may contain/start with a literal '%'. This will just be reported by fmt.Fprintf accordingly, so we don't check this.
	}
}

func (a ast_fmtPercent) VerifyParameters_direct(parameters_direct ParamMap, _ error) (err error) {
	err = a.VerifySyntax()
	if err != nil {
		return
	}

	_, ok := parameters_direct[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

func (a ast_fmtPercent) VerifyParameters_passed(parameters_direct ParamMap, _ ParamMap, _ error) (err error) {
	return a.VerifyParameters_direct(parameters_direct, nil)
}

func (a ast_fmtPercent) Interpolate(parameters_direct ParamMap, _ ParamMap, _ error, s *strings.Builder) {
	a._Interpolate(parameters_direct, s, '%')
}

func (a ast_fmtDollar) VerifyParameters_direct(_ ParamMap, _ error) error {
	return a.VerifySyntax()
}

func (a ast_fmtDollar) VerifyParameters_passed(_ ParamMap, parameters_passed ParamMap, _ error) (err error) {
	if err = a.VerifySyntax(); err != nil {
		return
	}
	_, ok := parameters_passed[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

func (a ast_fmtDollar) Interpolate(_ ParamMap, parameters_passed ParamMap, _ error, s *strings.Builder) {
	a._Interpolate(parameters_passed, s, '$')
}

func (a ast_parentPercent) VerifySyntax() error {
	return nil
}

func (a ast_parentPercent) VerifyParameters_direct(_ ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w, but the error does not wrap a non-nil error")
	} else {
		if errValidatable, ok := baseError.(ErrorInterpolater); ok {
			errFromBase := errValidatable.ValidateError_Params(nil)
			if errFromBase != nil {
				return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromBase)
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
}

func (a ast_parentPercent) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) error {
	return a.VerifyParameters_direct(nil, baseError) // first argument is ignored anyway
}

func (a ast_parentPercent) Interpolate(_ ParamMap, _ ParamMap, baseError error, s *strings.Builder) {
	if baseError == nil {
		s.WriteString(`%!w(<nil>)`)
	} else {
		s.WriteString(baseError.Error())
	}
}

func (a ast_parentDollar) VerifySyntax() error {
	return nil
}

func (a ast_parentDollar) VerifyParameters_direct(_ ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the error does not wrap a non-nil error")
	}
	if errValidatable, ok := baseError.(ErrorInterpolater); !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the base error does not support this")
	} else {
		errFromBase := errValidatable.ValidateError_Base()
		if errFromBase != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromBase)
		}
	}

	return nil
}

func (a ast_parentDollar) VerifyParameters_passed(_ ParamMap, parameters_passed ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the error does not wrap a non-nil error")
	}
	if errValidatable, ok := baseError.(ErrorInterpolater); !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the base error does not support this")
	} else {
		errFromBase := errValidatable.ValidateError_Params(parameters_passed)
		if errFromBase != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromBase)
		}
	}
	return nil
}

func (a ast_parentDollar) Interpolate(_ ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if baseError == nil {
		s.WriteString(`$!w<nil>`)
	}
	if errInterpolatable, ok := baseError.(ErrorInterpolater); !ok {
		s.WriteString(`$!w($w not supported)`)
	} else {
		s.WriteString(errInterpolatable.Error_interpolate(parameters_passed))
	}
}

func (abase *base_ast_condition) VerifySyntax() error {
	if abase.invalidParse {
		panic(ErrorPrefix + "Invalid parse not handled") // should be caught be ast_root.
	}
	if !utils.ElementInList(abase.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", abase.condition)
	}
	return abase.child.VerifySyntax()
}

func (a ast_condPercent) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)
	}
	// We actually evalutate the condition here. If the condition is false, we weaken the child-check to syntax only
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifyParameters_direct(parameters_direct, baseError)
		} else {
			return a.child.VerifySyntax()
		}
	case ConditionNonEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifySyntax()
		} else {
			return a.child.VerifyParameters_direct(parameters_direct, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}

func (a ast_condPercent) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)
	}
	// We actually evalutate the condition here. If the condition is false, we weaken the child-check to syntax only
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		} else {
			return a.child.VerifySyntax()
		}
	case ConditionNonEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifySyntax()
		} else {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}

func (a ast_condPercent) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		s.WriteString(`%!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}
	if a.invalidParse {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_direct) == 0 {
			a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		}
	case ConditionNonEmptyMap:
		if len(parameters_direct) != 0 {
			a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}

func (a ast_condDollar) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)
	}
	return a.child.VerifySyntax()
}

func (a ast_condDollar) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)
	}
	// We actually evalutate the condition here. If the condition is false, we weaken the child-check to syntax only
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_passed) == 0 {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		} else {
			return a.child.VerifySyntax()
		}
	case ConditionNonEmptyMap:
		if len(parameters_passed) == 0 {
			return a.child.VerifySyntax()
		} else {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}

func (a ast_condDollar) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if !utils.ElementInList(a.condition, validConditions[:]) {
		s.WriteString(`$!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}
	if a.invalidParse {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}

	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_passed) == 0 {
			a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		}
	case ConditionNonEmptyMap:
		if len(parameters_passed) != 0 {
			a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}
