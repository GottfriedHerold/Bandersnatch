package errorsWithData

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Parsing and Interpolating format strings goes through multiple steps:
//
//   - Tokenize the interpolation string
//   - Parse the tokenized string into a syntax tree
//   - Make some syntactic validity checks on the strings for conditions, variable names and format verbs.
//   - [Optional] Perform some validity checks.
//     (2 subchecks, actually. Those would be checked when actually producing output anyway, but sometime we want those checks early)
//     Those validity checks are related to whether data to be output is actually present, so it's not only a function of the interpolation string.
//   - Actually produce the interpolated error string.

// This file contains the code for the last 3 steps.

// The syntactic validity checks are handled by [handleSyntaxConditions], making the following checks:
//   - format strings verbs cannot contain literal %
//   - VariableNames must be exported Go identifiers (or denote the parameter map)
//   - Conditions after %! or $! must be recognized by our language.
// Calling handleSyntaxConditions is mandatory for the later steps; calling it modifies the syntax tree on mistake, records the first mistake in the root and flags that it was called.
// The later procesing steps such as Interpolate actually call [handleSyntaxConditions] to ensure it was called (it's a no-op to call it twice; the flag that was set ensures that).
//
// For optional validity checks, we have
//
//  - VerifyParameters_direct(parameters_direct paramMap, baseError error) Mistake
//  - VerifyParameters_passed(parameters_direct paramMap, parameters_passed paramMap, baseError error) Mistake
//
// Each of these checks subsumes the checks above it and requires more "context".
// If there was a mistake in make_ast or [handleSyntaxConditions], that mistake is just reported again and no further checks are made.
//
//  - VerifyParameters_direct checks that:
//     - %w or $w is only present if there is actually a non-nil wrapped error and, for $w, supports this.
//     - variables referred to by %fmtString{VariableName} actually exist in parameters_direct
//     The condition in %!COND{...} is evaluated for this purpose and failures are ignored in a non-taken sub-tree.
//
//  - VerifyParameters_passed furthermore checks that:
//     - variables referred to by $fmtString{VariableName} actually exist in parameters_passed
//     The conditions in both %!COND{...} and $!COND{...} are evaluated for this purpose. Failures are ignored in a non-taken sub-tree.
//
// Note that even VerifyParameters_passed does not guarantee that Interpolation works, because e.g. the format verb might be invalid for the given type.
// Also, a custom String method or Format method might panic.
// For the latter, note that the [fmt] package actually recovers from such panics and reports it in-band in the output string.
// Generally, [fmt] does a good job here, but detecting such mistakes beforehand is out of scope of this package.

// valid entries for Condition strings
// var validConditions [2]string = [2]string{ConditionEmptyMap, ConditionNonEmptyMap}
var validMapSelectors [4]string = [4]string{"!m", "!map", "!parameters", "!params"}
var conditionMapSelectors [4]string = [4]string{"m", "map", "parameters", "params"}
var specialVariableNameIndicator byte = '!' // must be first byte of each validMapSelectors - entry. Note type is byte, not rune.

// $w{#} or %w{#} outputs the lenth of the list output of base_error.Unwrap(), where Unwrap returns []error.
const outputChildNumber string = "#"

// multiUnwrap is the interface satisfied by errors with an Unwrap() []error method.
type multiUnwrap interface {
	error
	Unwrap() []error
}

// NOTE on panics:
// None of handleSyntaxConditions, VerifyParameters_direct, VerifyParameters_passed or Interpolate is supposed to ever panic.
// All panics in those methods are (supposed to be) unreachable if called on the output of [make_ast] (with input satisfying its assertions), even for mis-parses.
// Those panics just double-check internal invariants.

/*
 * handleSyntaxConditions defined here.
 *
 * handleSyntaxConditions recursively goes through the tree and checks for the following conditions:
 *
 *  - Is there a literal % in a formatVerb?
 *  - Are conditions recognized
 *  - Are variable names valid
 *
 * If a mistake is found, returns the first found mistake as a non-nil return value.
 * The mistake is also recorded in a.(ast_root).argumentError for the root.
 * Note that we always process all nodes and do *not* abort on first mistake, because we actually modify the ast:
 * - ast_fmt - nodes with invalid format verbs or invalid variable names are flagged as erroneous and
 *   we record a diagnostic message inside them, to be displayed when calling Interpolate.
 * - Invalid condition strings are marked to trigger unconditional evaluation and special display behaviour.
 */

// TODO: mistake reporting ([]mistake vs mistake?), thread-safety issues with the design.

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
// This post-processing is mandatory; this is automatically triggered by the other relevant methods.
//
// It returns the first mistake encountered, but processes the whole tree.
// This method must still be called even if [make_ast] returned a mistake.
// The reason is that [make_ast]'s mistake handling works in a way that always produces an actual tree, whose guarantees are no different from the success case.
//
// For such trees, handleSyntaxConditions might return a different mistake (or no mistake at all);
// For user-facing methods, the mistake returned by [make_ast] needs to take precedence over the one by handleSyntaxConditions.
//
// Note that the code currently assumes that handleSyntaxConditions is always called through [ast_root], never directly on other nodes.
func (a ast_root) handleSyntaxConditions() (err Mistake) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child") // cannot happen
	}
	// a.syntaxHandled is a flag set to indicate that we already called [handleSyntaxConditions]
	// No need to process the tree twice.
	// NOTE: Some other node handlers currently assume that handleSyntaxConditions is never called twice on them, so this "optimization" is actually mandatory atm.
	if a.syntaxHandled {
		return a.argumentMistake // may be nil
	}
	// proceed with child (probably of type list).
	// Note that we store the mistake inside a.argumentMistake, to avoid processing everything multiple times.
	a.argumentMistake = a.ast.handleSyntaxConditions()
	a.syntaxHandled = true
	return a.argumentMistake
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_list, we just call it on each child and report the first mistake. Note that we do not abort on first mistake.
func (a ast_list) handleSyntaxConditions() (err Mistake) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list") // cannot happen for ast's created by make_ast
	}
	for _, ast := range *a {
		// We report the first mistake, but do process all nodes.
		if err == nil {
			err = ast.handleSyntaxConditions()
		} else {
			_ = ast.handleSyntaxConditions()
		}
	}
	return err
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_string nodes, there are no failure cases
func (a ast_string) handleSyntaxConditions() Mistake {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_parentPercent, there are no failure cases
func (a ast_parentPercent) handleSyntaxConditions() Mistake {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_parentDollar, there are no failure cases
func (a ast_parentDollar) handleSyntaxConditions() Mistake {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_parentDollarMulti or ast_parentPercentMulti, there are no failure cases
func (*base_ast_parentMult) handleSyntaxConditions() Mistake {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_fmt, we check the variable name and the format verb
// On mistake, we flag the ast_fmt node by setting abase.errorString. If non-nil, this gets displayed by Interpolate instead of using [fmt].
func (abase *base_ast_fmt) handleSyntaxConditions() Mistake {

	// abase.errorString is supposed to be only set by this method, and we never call it twice.
	// (because the root node detects that handleSyntaxConditions was already called)
	// If we change code such that this assumption no longer is guaranteed, this needs to be reviewed here.
	if abase.mistakeString != nil {
		panic("Cannot happen")
	}

	// Cannot happen: This is caught by the parser and replaced by `v`
	if abase.variableName == "" {
		panic(ErrorPrefix + "Uninitialized variable name")
	}

	if strings.ContainsRune(abase.formatString, '%') {
		abase.mistakeString = fmt.Errorf(`<!Format Verb %s for %s contains %%>`+abase.formatString, abase.variableName)
		return fmt.Errorf(ErrorPrefix+`Interpolation string contains a format string verb %s, which contains  '%%'. This will not work with the fmt package`, abase.formatString)
	}

	// Note: if we detect an invalid variable name, the actual format verb does not affect or appear in the output. This is considered OK.

	if abase.variableName[0] == specialVariableNameIndicator {
		if !utils.ElementInList(abase.variableName, validMapSelectors[:]) {
			abase.mistakeString = fmt.Errorf(`<!Variable name %s in interpolation string starting with %s not recognized by the language>`, abase.variableName, string(specialVariableNameIndicator))
			return fmt.Errorf(ErrorPrefix+"Variable name %s in interpolation string starting with %s is not recognized by the language", abase.variableName, string(specialVariableNameIndicator))
		}
	} else if !IsExportedIdentifier(abase.variableName) {
		abase.mistakeString = fmt.Errorf(`<!Variable name %s not allowed by the language`, abase.variableName)
		return fmt.Errorf(ErrorPrefix+"Variable name %s is not allowed by the language", abase.variableName)
	}

	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that mistakes are handled correctly later.
//
// For ast_cond, we just flag the conditional as invalid on failure.
// We also drop all whitespace if there is no error.
func (abase *base_ast_condition) handleSyntaxConditions() Mistake {
	abase.variableName, abase.conditionType = parseConditionString(abase.condition)

	if abase.conditionType == conditionType_Invalid {
		// This causes Interpolate to display children unconditionally && display a diagnostic message containing the condition string.
		abase.make_invalid(astConditionValidity_OUTPUT_CHILD | astConditionValidity_OUTPUT_CONDITION)
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", abase.condition)
	} else {
		return nil
	}
}

/**
 *
 * VerifyParameters_direct and VerifyParameters_passed defined here.
 * NOTE: VerifyParameters_direct corresponds to VerifyParameters_passed with an UNKNOWN value for params_passed.
 *
 * VerifyParameters_direct checks whether
 *  - %w is only used if there is a parent baseError
 *  - $w is only used if there is a parent baseError that supports this
 *  - %w{#} is only used if there is a parent baseError that supports this (i.e. has Unwrap() []error)
 *  - %w{n} is only used if there is a parent baseError that Unwraps at least n errors.
 *  - $w{#} is only used if there is a parent baseError that supports this (i.e. has Unwrap() []error)
 *  - $w{n} is only used if there is a parent baseError that Unwraps at least n errors and that n'th error supports $
 *  - variable names referred to by %verb{variable} are actually present in the ParamMap
 *
 * VerifyParameters_passed checks whether
 *  - variable names referred to by $verb{variable} are actually present in the (passed through) ParamMap
 *
 * If a condition is known not to be taken, the subtree is ignored.
 * For VerifyParameters_passed, this means we evaluate all conditions and check whether they are taken.
 * For VerifyParameters_direct, we only do so for %cond{}, but not $cond{}. For the latter, we assume the branch is taken.
 *
 * If a base error is referenced by %w, $w, %w{n} or $w{n} and that error supports ErrorInterpolater, we recursively check the base as appropriate.
 *
 * We only report the first mistake encountered.
 * Note that both parse mistakes and syntax mistakes uncovered by [handleSyntaxConditions] take priority.
 * If there was an parse or syntax mistake, we always report it instead.
 */

// NOTE: We assumes the ast was created by [make_ast] and we ensure [handleSyntaxConditions] was called for post-processing.
// Furthermore, we assume that all calls go through ast_root.
// In particular, we detect mistakes recorded by [make_ast] and [handleSyntaxConditions] at the root and never
// process the tree.

// For parameters_passed, note that is should never be nil.

// VerifyParameters_direct checks whether the AST contains any parse or syntax mistakes that were recorded when creating it.
// If not, it checks whether parameters in %fmtVerb{VariableName} expressions are actually present and
// %w and $w expressions refer to valid baseErrors.
// It also recursively checks the baseError, if referred to via %w or $w.
// Untaken conditional branches are not checked (but parse or syntax mistakes there will be reported)
// Only the first mistake is reported.
//
// The method uses parameters_direct its for variables and baseError as the baseError.
//
// VerifyParameters_direct for the root node just checks for mistakes (those were recorded in the root node by [make_ast], and [handleSyntaxConditions] and hands off to the child)
func (a ast_root) VerifyParameters_direct(parameters_direct ParamMap, baseError error) Mistake {

	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child") // cannot happen for ASTs created by make_ast
	}

	syntaxError := a.handleSyntaxConditions() // ensure this is called
	// If [make_ast] detected a parse mistake, this overrides any further tests.
	if a.parseMistake != nil {
		return a.parseMistake
	}

	if syntaxError != nil {
		return syntaxError
	}

	return a.ast.VerifyParameters_direct(parameters_direct, baseError)
}

// VerifyParameters_passed checks whether the AST contains any parse or syntax mistakes that were recorded when creating it.
// If not, it checks whether parameters in %fmtVerb{VariableName} or $fmtVerb{VariableName} expressions are actually present and
// %w and $w expressions refer to valid baseErrors.
// It also recursively checks the baseError, if referred to via %w or $w and the the base error supports this (via ValidateError_Params())
// Untaken conditional branches are not checked (but parse or syntax mistakes there will be reported)
// Only the first mistake is reported.
//
// The method uses parameters_direct resp. parameters_passed for its for variables and baseError as the baseError.
// NOTE: parameters_passed must not be nil (an empty map is fine), to avoid confusion. We panic in this case.
// The special-cased meaning of parameters_passed == nil in [ValidateError_Params] from the [ErrorWithData_any] or [ErrorInterpolater] interface
// needs to be handled by [ValidateError_Params] rather than here.
//
// VerifyParameters_passed for the root node just checks for mistakes (those were recorded in the root node by [make_ast], and [handleSyntaxConditions] and hands off to the child)
func (a ast_root) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) Mistake {

	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child") // cannot happen for ASTs created by make_ast
	}

	if parameters_passed == nil {
		panic(ErrorPrefix + "VerifyParameters_passed called with nil map for parameters_passed. This must be unreachable for the exported API") // bug in caller.
	}

	syntaxError := a.handleSyntaxConditions() // ensure this is called
	// If [make_ast] detected a parse mistake, this overrides any further tests.
	if a.parseMistake != nil {
		return a.parseMistake
	}

	if syntaxError != nil {
		return syntaxError
	}
	return a.ast.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)

}

// VerifyParameters_direct for list nodes just reports the first mistake in a child
func (a ast_list) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err Mistake) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list") // cannot happen for ASTs created by make_ast
	}
	for _, ast := range *a {
		err = ast.VerifyParameters_direct(parameters_direct, baseError)
		if err != nil {
			return
		}
	}
	return nil
}

// VerifyParameters_passed for list nodes just reports the first mistake in a child
func (a ast_list) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err Mistake) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list") // cannot happen for ASTs created by make_ast
	}
	for _, ast := range *a {
		err = ast.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		if err != nil {
			return
		}
	}
	return nil
}

// VerifyParameters_direct for string nodes never reports an mistake
func (a ast_string) VerifyParameters_direct(ParamMap, error) Mistake {
	return nil
}

// VerifyParameters_passed for string nodes never reports an mistake
func (a ast_string) VerifyParameters_passed(ParamMap, ParamMap, error) Mistake {
	return nil
}

// VerifyParameters_direct for %fmtVerb{variableName} checks whether the parameter is present.
func (a ast_fmtPercent) VerifyParameters_direct(parameters_direct ParamMap, _ error) (err Mistake) {

	_, ok := parameters_direct[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

// VerifyParameters_passed for %fmtVerb{variableName} checks whether the parameter is present.
func (a ast_fmtPercent) VerifyParameters_passed(parameters_direct ParamMap, _ ParamMap, _ error) (err Mistake) {

	// same as VerifyParameters_direct. We ignore the parameters_passed map
	_, ok := parameters_direct[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

// VerifyParameters_direct for $fmtVerb{variableName} never reports an mistake (this is so because the parameter might later be filled in)
func (a ast_fmtDollar) VerifyParameters_direct(_ ParamMap, _ error) Mistake {
	return nil
}

// VerifyParameters_passed for $fmtVerb{variableName} checks whether the variable is present in parameters_passed.
//
// NOTE: we assume that parameters_passed is not nil. This is checked/handled at the root node.
func (a ast_fmtDollar) VerifyParameters_passed(_ ParamMap, parameters_passed ParamMap, _ error) (err Mistake) {

	_, ok := parameters_passed[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

// VerifyParameters_direct for %w checks whether a base error is present.
// If possible, we also actually check for problems in the referred baseError
//
// NOTE: The latter is done by using ValidateError_Params, which will then call VerifyParameters_passed on the referred error.
// This change to the *_passed - variant is actually correct: %w prevents arguments from being passed to it,
// so we know what the passed parameters to the baseError are.
func (a ast_parentPercent) VerifyParameters_direct(_ ParamMap, baseError error) Mistake {
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w, but the error does not wrap a non-nil error")
	} else {
		if errValidatable, ok := baseError.(ErrorInterpolater); ok {
			// ValidateError_Params(nil) checks whether the base error is valid with its own parameters.
			// This is the correct question here.
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

// VerifyParameters_passed for %w checks whether the base error is present.
// If possible, we also actually check for problems in the referred baseError
//
// The same considerations as with VerifyParameters_direct apply here.
func (a ast_parentPercent) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) Mistake {
	// exactly the same as VerifyParamter_direct
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w, but the error does not wrap a non-nil error")
	} else {
		if errValidatable, ok := baseError.(ErrorInterpolater); ok {
			// ValidateError_Params(nil) checks whether the base error is valid solely with its *own* parameters.
			// This is the correct question here.
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

// VerifyParameters_direct for $w checks whether the base error is present and supports $w.
//
// We also check for problems in the referred baseError via [ValidateError_Base]
//
// NOTE: ValidateError_Base() will call VerifyParameters_direct on the referred error.
func (a ast_parentDollar) VerifyParameters_direct(_ ParamMap, baseError error) Mistake {
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

// VerifyParameters_passed for $w checks whether the base error is present and supports $w
//
// We also check for problems in the referred baseError via [ValidateError_Params]
//
// NOTE: ValidateError_Params(parameters_passed) will call VerifyParameters_passed on the referred error, with parameters_passed passed through.
func (a ast_parentDollar) VerifyParameters_passed(_ ParamMap, parameters_passed ParamMap, baseError error) Mistake {
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

func (a ast_parentPercentMulti) VerifyParameters_direct(_ ParamMap, baseError error) Mistake {
	// Check that the base error is non-nil has an Unwrap() []error - method
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w{...}-expression, but the error does not wrap a non-nil error")
	}
	errMulti, ok := baseError.(multiUnwrap)
	if !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w{...}-expression, but the base error has no Unwrap() []error - method")
	}

	// a.whichChild == -1 indicates we just want count the number of errors. This cannot fail.
	if a.whichChild == -1 {
		return nil
	}
	if a.whichChild <= 0 { // whichChild is 1-indexed into the result of Unwrap()
		panic("Cannot happen")
	}

	// ensure the relevant child exists; if yes and we can recurse, do so:
	childErrors := errMulti.Unwrap()
	if a.whichChild > len(childErrors) {
		return fmt.Errorf(ErrorPrefix+"Interpolating string contains %%w{%v}-expression, but the base error wraps only %v errors", a.whichChild, len(childErrors))
	}
	if relevantChild, CanRecurse := childErrors[a.whichChild-1].(ErrorInterpolater); CanRecurse {
		if relevantChild == nil { // should not happen unless the user uses a custom Join method.
			return nil
		}
		if errFromChild := relevantChild.ValidateError_Params(nil); errFromChild != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromChild)
		} else {
			return nil
		}
	}
	return nil
}

func (a ast_parentPercentMulti) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) Mistake {
	// exactly the same code as VerifyParameters_direct
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w{...}-expression, but the error does not wrap a non-nil error")
	}
	errMulti, ok := baseError.(multiUnwrap)
	if !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains %%w{...}-expression, but the base error has no Unwrap() []error - method")
	}
	if a.whichChild == -1 {
		return nil
	}
	if a.whichChild <= 0 {
		panic("Cannot happen")
	}
	childErrors := errMulti.Unwrap()
	if a.whichChild > len(childErrors) {
		return fmt.Errorf(ErrorPrefix+"Interpolating string contains %%w{%v}-expression, but the base error wraps only %v errors", a.whichChild, len(childErrors))
	}
	if relevantChild, CanRecurse := childErrors[a.whichChild-1].(ErrorInterpolater); CanRecurse {
		if relevantChild == nil { // should not happen unless the user uses a custom Join method.
			return nil
		}
		if errFromChild := relevantChild.ValidateError_Params(nil); errFromChild != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromChild)
		} else {
			return nil
		}
	}
	return nil
}

func (a ast_parentDollarMulti) VerifyParameters_direct(_ ParamMap, baseError error) Mistake {
	// similar to the above:

	// Check that the base error is non-nil has an Unwrap() []error - method
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w{...}-expression, but the error does not wrap a non-nil error")
	}
	errMulti, ok := baseError.(multiUnwrap)
	if !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w{...}-expression, but the base error has no Unwrap() []error - method")
	}

	// a.whichChild == -1 indicates we just want count the number of errors. This cannot fail.
	if a.whichChild == -1 {
		return nil
	}
	if a.whichChild <= 0 { // whichChild is 1-indexed into the result of Unwrap()
		panic("Cannot happen")
	}

	// ensure the relevant child exists; we actually require it to support recursion now.
	childErrors := errMulti.Unwrap()
	if a.whichChild > len(childErrors) {
		return fmt.Errorf(ErrorPrefix+"Interpolating string contains %%w{%v}-expression, but the base error wraps only %v errors", a.whichChild, len(childErrors))
	}
	if relevantChild, CanRecurse := childErrors[a.whichChild-1].(ErrorInterpolater); CanRecurse {
		if relevantChild == nil { // should not happen unless the user uses a custom Join method.
			return nil
		}
		if errFromChild := relevantChild.ValidateError_Base(); errFromChild != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", errFromChild)
		} else {
			return nil
		}
	} else { // CanRecurse is false
		return fmt.Errorf(ErrorPrefix+"Interpolation string contains $w{%v}-expression, but the referenced error does not support this", a.whichChild)
	}
}

func (a ast_parentDollarMulti) VerifyParameters_passed(_ ParamMap, params_passed ParamMap, baseError error) Mistake {
	// similar to the above, except that we call relevantChild.ValidateError_Params(params_passed) instead of relevantChild.ValidateError_Base()

	// Check that the base error is non-nil has an Unwrap() []error - method
	if baseError == nil {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w{...}-expression, but the error does not wrap a non-nil error")
	}
	errMulti, ok := baseError.(multiUnwrap)
	if !ok {
		return fmt.Errorf(ErrorPrefix + "Interpolation string contains $w{...}-expression, but the base error has no Unwrap() []error - method")
	}

	// a.whichChild == -1 indicates we just want count the number of errors. This cannot fail.
	if a.whichChild == -1 {
		return nil
	}
	if a.whichChild <= 0 { // whichChild is 1-indexed into the result of Unwrap()
		panic("Cannot happen")
	}

	// ensure the relevant child exists; we actually require it to support recursion now.
	childErrors := errMulti.Unwrap()
	if a.whichChild > len(childErrors) {
		return fmt.Errorf(ErrorPrefix+"Interpolating string contains %%w{%v}-expression, but the base error wraps only %v errors", a.whichChild, len(childErrors))
	}
	if relevantChild, CanRecurse := childErrors[a.whichChild-1].(ErrorInterpolater); CanRecurse {
		if relevantChild == nil { // should not happen unless the user uses a custom Join method.
			return nil
		}
		if mistakeFromChild := relevantChild.ValidateError_Params(params_passed); mistakeFromChild != nil {
			return fmt.Errorf(ErrorPrefix+"Problem in wrapped error: %w", mistakeFromChild)
		} else {
			return nil
		}
	} else { // CanRecurse is false
		return fmt.Errorf(ErrorPrefix+"Interpolation string contains $w{%v}-expression, but the referenced error does not support this", a.whichChild)
	}
}

// checkWhetherBranchIsTaken checks the condition determined by conditionType, params and variableName is taken.
//
// conditionType must be one of the enum-style constants conditionType_Foo other than conditionType_Invalid.
// params determines where parameters are looked up (for presence, zero-ness or just checking their number).
// variableName is the option variableName argument required for some types.
//
// If conditionType is conditionType_invalid, this function panics.
func checkWhetherBranchIsTaken(conditionType int, params ParamMap, variableName string) (takeBranch bool) {
	if conditionType == conditionType_Invalid {
		panic("Cannot happen")
	}

	switch conditionType {
	case conditionType_EmptyMap:
		takeBranch = len(params) == 0
	case conditionType_NonEmptyMap:
		takeBranch = len(params) != 0
	case conditionType_ParameterZero, conditionType_ParameterNonZero:
		value, found := params[variableName]
		if !found {
			break
		}
		var valueIsZero bool
		if value == nil {
			valueIsZero = true
		} else {
			valueIsZero = reflect.ValueOf(value).IsZero()
		}
		takeBranch = valueIsZero == (conditionType == conditionType_ParameterZero)
	case conditionType_ParameterPresent:
		_, takeBranch = params[variableName]
	case conditionType_ParameterMissing:
		_, takeBranch = params[variableName]
		takeBranch = !takeBranch

	default: // including conditionType_Invalid
		panic("Cannot happen")
	}
	return
}

// VerifyParameters_direct for %condition{Subtree} will conditionally check the subtree if the condition holds
func (a ast_condPercent) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err Mistake) {
	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an mistake in the root node.
	}

	// For condPercent, we have all information that we need, so we can actually evaluate the condtion.
	// We only check the subtree if the condition holds.
	var takeBranch bool = checkWhetherBranchIsTaken(a.conditionType, parameters_direct, a.variableName)

	if takeBranch {
		return a.child.VerifyParameters_direct(parameters_direct, baseError)
	} else {
		return nil
	}

}

// VerifyParamters_passed for %condition{Subtree} will conditionally check the subtree if the condition holds.
func (a ast_condPercent) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err Mistake) {
	// same as VerifyParameters_direct, except for calling the approprite VerifyParamters_passed on the subtree

	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an mistake in the root node.
	}

	// For condPercent, we have all information that we need, so we can actually evaluate the condtion.
	// We only check the subtree if the condition holds.
	var takeBranch bool = checkWhetherBranchIsTaken(a.conditionType, parameters_direct, a.variableName)

	if takeBranch {
		return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
	} else {
		return nil
	}
}

// VerifyParameters_direct for $condition{Subtree} will always check the subtree.
//
// This is because using the error as a base may actually cause the condition to be true;
// The mistakes potentially detected in the subtree are only those that would be mistakes for _any_ choice of passed parameters, so we want to
// detect those. Recall that VerifiyParameters_direct corresponds to an *unknown* value for parameters_passed.
func (a ast_condDollar) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err Mistake) {
	return a.child.VerifyParameters_direct(parameters_direct, baseError)
}

// VerifyParameters_passed for $condition{Subtree} will conditionally check the subtree if the condition holds.
func (a ast_condDollar) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err Mistake) {
	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an mistake in the root node.
	}

	takeBranch := checkWhetherBranchIsTaken(a.conditionType, parameters_passed, a.variableName)
	if takeBranch {
		return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
	} else {
		return nil
	}
}

// Interpolate actually produces the required output string.
// For reasons of efficiency, the Interpolate-functions here do not return a string, but rather take a strings.Builder argument and append to that.
//
// parameters_direct is the map used to evaluate % - expressions
// parameters_passed is the map used to evaluate $ - expressions and is forwarded to $w
// baseError is the baseError used to evaluate %w and $w

// Interpolate for the root node is the entry point for Interpolate-calls.
//
// parameters_passed == nil means parameters_passed are the same as parameters_direct.
// We handle this special-case here, so other node types don't have to handle it.
// (Note: We could also let the caller or each node do that, but doing it here is more convenient -- it allows us to make some diagnostics prettier)
// parameters_direct should not be nil (use an empty map instead)
//
// Error handling: Note that [make_ast] always outputs a valid tree that contains a in-band diagnostic message.
// We also ensure that [handleSyntaxConditions] has been called.
// In either case, we just process it normally.
// Note that on parse mistakes, after the first mistake, [make_ast] has turned all special tokens inactive, so %w and $w
// and formatted parameter output might be suppressed.
// [handleSyntaxConditions] has marked ast_fmt nodes and ast_cond nodes. These will be processed by Interpolate.
// Since special tokens may have been rendered inactive and output not what the user expected,
// we always append a diagnostic *after* processing the tree normally and always explicitly print
// the parent error and the full parameter map. This is to ensure that, if this ends up in some log file, the relevant information is there.
func (a ast_root) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}

	// Ensure handleSyntaxConditions is called at least once and check whether there is any parse or syntax mistake in the ast.
	var hasError bool = a.handleSyntaxConditions() != nil
	hasError = hasError || (a.parseMistake != nil)

	// NOTE: Even if hasError is true, we still process everything.

	// parameters_passed == nil is (mostly) treated as if parameters_passed was equal to parameters_direct.
	// Note that there is a minor difference in mistake handling below in that we don't output the parameters twice in that case.
	if parameters_passed == nil {
		a.ast.Interpolate(parameters_direct, parameters_direct, baseError, s)
	} else {
		a.ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

	// Extended diagnostic in case of parse or syntax mistake.
	if hasError {
		if baseError != nil {
			s.WriteString("\nBase error:\n")
			s.WriteString(baseError.Error()) // Note: We don't check for baseError.(ErrorInterpolater), because we output the parameters anyway (if present).
		}

		// unconditionally output params, if present:
		// (parameters_direct == nil is handled as an empty map. This is not supposed to happen, but we handle it gracefully)
		if len(parameters_direct) != 0 {
			s.WriteString("\nParameters in error:\n")
			fmt.Fprintf(s, "%v", parameters_direct)
		}

		// If parameters_passed == nil, we don't need to output it.
		// Note that if parameters_passed is an empty map, we actually output it. This is intentional and correct.
		if parameters_passed != nil {
			s.WriteString("\nParameters from parent error:\n")
			fmt.Fprintf(s, "%v", parameters_passed)
		}
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For list node, just iterate over the list.
func (a ast_list) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For string nodes, just output the string
func (a ast_string) Interpolate(_ ParamMap, _ ParamMap, _ error, s *strings.Builder) {
	s.WriteString(string(a)) // NOTE: need string(a), not a.String() here; the latter would add literal "-marks. Also String() is a testing-only method.
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For %fmtVerb{Variable} nodes, hand off to [interpolate_helper] to unify with $fmtVerb{Variable} case
func (a ast_fmtPercent) Interpolate(parameters_direct ParamMap, _ ParamMap, _ error, s *strings.Builder) {
	a.interpolate_helper(parameters_direct, s, '%')
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For $fmtVerb{Variable} nodes, hand off to [interpolate_helper] to unify with %fmtVerb{Variable} case
func (a ast_fmtDollar) Interpolate(_ ParamMap, parameters_passed ParamMap, _ error, s *strings.Builder) {
	a.interpolate_helper(parameters_passed, s, '$')
}

// interpolate_helper is the actual implementation of Interpolate for both ast_fmtPercent and ast_fmtDollar.
//
// parameters_relevant is either parameters_direct (for %) or parameters_passed (for $).
// PercentOrDollar is a literal '$' or '%', required for diagnostic messages.
func (a *base_ast_fmt) interpolate_helper(parameters_relevant ParamMap, s *strings.Builder, PercentOrDollar rune) {
	// Check whether [handleSyntaxConditions] has detected an mistake. If so, output a replacement diagnostic message instead.
	if a.mistakeString != nil {
		s.WriteString(a.mistakeString.Error())
		return
	}

	var value any
	var ok bool
	if utils.ElementInList(a.variableName, validMapSelectors[:]) {
		if parameters_relevant == nil {
			value = make(ParamMap) // nil -> empty map. This should not happen, but better safe than sorry.
		} else {
			value = parameters_relevant
		}
		ok = true

	} else {
		// NOTE: [handleSyntaxConditions] has checked whether the variable name is a valid name for our language.
		// This means that an invalid name can never be looked up in the parameters_relevant map.
		value, ok = parameters_relevant[a.variableName]
	}

	if !ok {
		s.WriteRune(PercentOrDollar)
		s.WriteString(a.formatString)
		s.WriteString(`!<missing value>`)
	} else {
		// NOTE: a.formatString is guaranteed not to contain further %'s at this point.
		// At any rate, fmt.Fprintf would handle it just fine (by printing an diagnostic message).
		fmt.Fprintf(s, "%"+a.formatString, value)
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For %condition{subtree}, check the condition and (possibly) evaluate the subtree.
//
// NOTE: condition nodes flagged as tainted due to mistakes and handled specially:
// We have a flag for "always evaluate the subtree" and a flag for "display condition string"
func (a ast_condPercent) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {

	// Check whether we should output the condition string.
	if (a.invalidParse & astConditionValidity_OUTPUT_CONDITION) != 0 {
		s.WriteString(`%!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
	}

	// Note: Currently, if a.invalidParse & astConditionValidity_OUTPUT_CONDITION is set,
	// then _OUTPUT_CHILD is actually always set as well.

	// Check whether we should output the children unconditionally.
	if (a.invalidParse & astConditionValidity_OUTPUT_CHILD) != 0 {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

	if !a.is_valid() { // in this case, either of the above was triggered.
		return
	}

	takeBranch := checkWhetherBranchIsTaken(a.conditionType, parameters_direct, a.variableName)
	if takeBranch {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For $condition{subtree}, check the condition and (possibly) evaluate the subtree.
//
// NOTE: condition nodes flagged as tainted due to mistakes and handled specially:
// We have a flag for "always evaluate the subtree" and a flag for "display condition string"
func (a ast_condDollar) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {

	// Check whether we should output the condition string.
	if (a.invalidParse & astConditionValidity_OUTPUT_CONDITION) != 0 {
		s.WriteString(`$!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
	}

	// Check whether we should output the children unconditionally.
	if (a.invalidParse & astConditionValidity_OUTPUT_CHILD) != 0 {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

	if !a.is_valid() { // in this case, either of the above was triggered.
		return
	}

	takeBranch := checkWhetherBranchIsTaken(a.conditionType, parameters_passed, a.variableName)
	if takeBranch {
		a.child.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For %w, we just call Error() on the baseError
func (a ast_parentPercent) Interpolate(_ ParamMap, _ ParamMap, baseError error, s *strings.Builder) {
	if baseError == nil {
		s.WriteString(`%w(<nil>)`)
	} else {
		s.WriteString(baseError.Error())
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For $w, we call Error_interpolate(parameters_passed) on the baseError to pass parameters through.
// (If the baseError does not support this, we output a replacement message)
func (a ast_parentDollar) Interpolate(_ ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if baseError == nil {
		s.WriteString(`$w(<nil>)`)
	} else if errInterpolatable, ok := baseError.(ErrorInterpolater); !ok {
		s.WriteString(`<$w is not supported by base error!>`)
		s.WriteString(baseError.Error()) // we still output the base error
	} else {
		s.WriteString(errInterpolatable.Error_interpolate(parameters_passed))
	}
}

func (a ast_parentPercentMulti) Interpolate(_ ParamMap, _ ParamMap, baseError error, s *strings.Builder) {
	if baseError == nil {
		if a.whichChild == -1 {
			s.WriteString(`%w{#}(<nil>)`)
		} else {
			fmt.Fprintf(s, `%%w{%v}(<nil>)`, a.whichChild)
		}
		return
	}
	baseUnwrappable, okUnwrap := baseError.(multiUnwrap)
	if !okUnwrap {
		if a.whichChild == -1 {
			s.WriteString(`%w{#}(<Error without Unwrap() []error>)`)
		} else {
			fmt.Fprintf(s, `%%w{%v}(<Error without Unwrap() []error>)`, a.whichChild)
		}
		return
	}
	childErrors := baseUnwrappable.Unwrap()
	if a.whichChild == -1 {
		fmt.Fprintf(s, "%v", len(childErrors))
		return
	}
	if a.whichChild <= 0 {
		panic("Cannot happen")
	}
	if a.whichChild > len(childErrors) {
		fmt.Fprintf(s, `%%w{%v}(<Index out of bounds>)`, a.whichChild)
		return
	}
	relevantChild := childErrors[a.whichChild-1]
	if relevantChild == nil { // cannot happen for errors created either by errors.Join or our own Join. We still handle it.
		s.WriteString(`<nil>`)
		return
	}
	s.WriteString(relevantChild.Error())
}

func (a ast_parentDollarMulti) Interpolate(_ ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	// almost the same as above
	if baseError == nil {
		if a.whichChild == -1 {
			s.WriteString(`$w{#}(<nil>)`)
		} else {
			fmt.Fprintf(s, `$w{%v}(<nil>)`, a.whichChild)
		}
		return
	}
	baseUnwrappable, okUnwrap := baseError.(multiUnwrap)
	if !okUnwrap {
		if a.whichChild == -1 {
			s.WriteString(`$w{#}(<Error without Unwrap() []error>)`)
		} else {
			fmt.Fprintf(s, `$w{%v}(<Error without Unwrap() []error>)`, a.whichChild)
		}
		return
	}
	childErrors := baseUnwrappable.Unwrap()
	if a.whichChild == -1 {
		fmt.Fprintf(s, "%v", len(childErrors))
		return
	}
	if a.whichChild <= 0 {
		panic("Cannot happen")
	}
	if a.whichChild > len(childErrors) {
		fmt.Fprintf(s, `$w{%v}(<Index out of bounds>)`, a.whichChild)
		return
	}
	relevantChild := childErrors[a.whichChild-1]
	if relevantChild == nil { // cannot happen for errors created either by errors.Join or our own Join. We still handle it.
		s.WriteString(`<nil>`)
		return
	}
	if errInterpolatable, okInterpolate := relevantChild.(ErrorInterpolater); !okInterpolate {
		fmt.Fprintf(s, `<$w{%v} not supported by referenced error>`, a.whichChild)
		s.WriteString(relevantChild.Error()) // still output the relevant error
	} else {
		s.WriteString(errInterpolatable.Error_interpolate(parameters_passed))
	}
}
