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
//   - Actually prodcuce the interpolated error string.

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

// valid entries for Condition strings
var validConditions [2]string = [2]string{ConditionEmptyMap, ConditionNonEmptyMap}
var validMapSelectors [4]string = [4]string{"m", "map", "parameters", "params"}

func IsExportedIdentifier(s string) bool {
	return token.IsIdentifier(s) && token.IsExported(s)
}

// Validation stages:
// 1.) syntactic
// 2.) % valid
// 3.) $ valid (fmt strings not checked)

// NOTE: Assumes a was created with make_ast

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
	a.ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
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
	s.WriteString(string(a)) // NOTE: need string(a), not a.String() here
}

func (abase *base_ast_fmt) VerifySyntax() error {
	if strings.ContainsRune(abase.formatString, '%') {
		return fmt.Errorf(ErrorPrefix+`Interpolation string contained a format string verb %s, which contained (escaped) %%. This will not work with the fmt package`, abase.formatString)
	}
	if abase.invalidParse { // should be detected at root
		panic(ErrorPrefix + "Invalidly parsed format string not detected at root")
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

func (a *base_ast_fmt) _Interpolate(parameters_relevant ParamMap, s *strings.Builder) {
	if a.invalidParse { // special case: If there was a parse error, we just plain output the format string
		s.WriteString(a.formatString)
		s.WriteString(a.variableName) // always "", actually
		return
	}
	var value any
	var ok bool = true
	switch a.variableName {
	case "m", "map", "parameters", "params":
		value = parameters_relevant
		if value == nil {
			value = make(ParamMap) // nil -> empty map. This should not happen, but better safe than sorry.
		}
	default:
		value, ok = parameters_relevant[a.variableName]
	}
	if a.formatString == "" {
		panic(ErrorPrefix + "Empty format string. This cannot happen")
	}

	if !ok {
		s.WriteString(`%!` + a.formatString + `<missing value>`)
	} else {
		fmt.Fprintf(s, "%"+a.formatString, value) // NOTE: a.formatString may contain/start with a literal '%'. This will just be reported by fmt.Fprintf accordingly, so we don't check this.
	}
	return
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
	a._Interpolate(parameters_direct, s)
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
	a._Interpolate(parameters_passed, s)
}

func (a ast_parentPercent) VerifySyntax() error {
	return nil
}

func (a ast_parentPercent) VerifyParameters_direct(_ ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w, but the error does not wrap a non-nil error")
	} else {
		return nil
	}
}

func (a ast_parentPercent) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w, but the error does not wrap a non-nil error")
	} else {
		return nil
	}
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
	if _, ok := baseError.(ErrorInterpolater); !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the base error does not support this")
	} else {
		return nil
	}
}

func (a ast_parentDollar) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) error {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the error does not wrap a non-nil error")
	}
	if _, ok := baseError.(ErrorInterpolater); !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w, but the base error does not support this")
	} else {
		return nil
	}
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
	if !utils.ElementInList[string](abase.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", abase.condition)
	}
	return abase.child.VerifySyntax()
}

func (a ast_condPercent) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
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
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
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
	if a.invalidParse {
		s.WriteString(a.condition)
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
		s.WriteString(`%!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
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
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)
	}
	return a.child.VerifySyntax()
}

func (a ast_condDollar) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
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
	if a.invalidParse {
		s.WriteString(a.condition)
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
		return
	}
	if !utils.ElementInList[string](a.condition, validConditions[:]) {
		s.WriteString(`$!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
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
