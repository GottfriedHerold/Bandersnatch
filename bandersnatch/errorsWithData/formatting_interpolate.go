package errorsWithData

import (
	"fmt"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// Parsing and Interpolating format strings goes through multiple steps:
//
//   - Tokenize the interpolation string
//   - Parse the tokenized string into a syntax tree
//   - Make some syntactic validity checks on the strings for conditions, variable names and format verbs.
//   - [Optional] Perform some validity checks. (2 subchecks, actually.
//     Those would be checked when actually producing output anyway, but sometime we want those checks early)
//     Those validity checks are related to whether data to be output is actually present, so it's not only a function of the interpolation string.
//   - Actually produce the interpolated error string.

// This file contains the code for the last 3 steps.

// The syntactic validity checks are handled by [handleSyntaxConditions], making the following checks:
//   - format strings verbs cannot contain literal %
//   - VariableNames must be exported Go identifiers (or denote the parameter map)
//   - Conditions after %! or $! must be recognized by our language.
// Calling handleSyntaxConditions is mandatory for the later steps; calling it modifies the syntax tree on error, records the first error in the root and flags that it was called.
// The later procesing steps such as Interpolate actually call [handleSyntaxConditions] to ensure it was called (it's a no-op to call it twice; the flag that was set ensures that).
//
// For optional validity checks, we have
//
//  - VerifyParameters_direct(parameters_direct paramMap, baseError error) error
//  - VerifyParameters_passed(parameters_direct paramMap, parameters_passed paramMap, baseError error) error
//
// Each of these checks subsumes the checks above it and requires more "context".
// If there was an error in make_ast or [handleSyntaxConditions], the error is just repeated
//
//  - VerifyParameters_direct checks that:
//     - %w or $w is only present if there is actually a non-nil wrapped error and, for $w, supports this.
//     - variables referred to by %fmtString{VariableName} actually exist in parameters_direct
//     The condition in %!COND{...} is evaluated for this purpose and failures are ignored in a non-taken sub-tree.
//
//  - VerifyParameters_passed furthermore checks that:
//     - variables referred to by $fmtString{VariableName} actually exist in paramters_passed
//     The conditions in both %!COND{...} and $!COND{...} are evaluated for this purpose. Failures are ignored in a non-taken sub-tree.
//
// Note that even VerifyParameters_passed does not guarantee that Interpolation works, because the format verb might be invalid for the given type.
// Also, a custom String method or Format method might panic.
// For the latter, note that the [fmt] package actually recovers from such panics and reports it in-band in the output string.
// Generally, [fmt] does a good job with error handling, but detecting such errors beforehand is out of scope of this package.

// valid entries for Condition strings
var validConditions [2]string = [2]string{ConditionEmptyMap, ConditionNonEmptyMap}
var validMapSelectors [4]string = [4]string{"!m", "!map", "!parameters", "!params"}
var specialVariableNameIndicator byte = '!' // must be first byte of each validMapSelectors - entry. Note type is byte, not rune.

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
 * If an error is found, returns the first found error as a non-nil return value.
 * The error is also recorded in a.(ast_root).argumentError for the root.
 * Note that we always process all nodes and do *not* abort on first error, because we actually modify the ast:
 * - ast_fmt - nodes with invalid format verbs or invalid variable names are flagged as erroneous and
 *   we record an error message inside them, to be displayed when calling Interpolate.
 * - Invalid condition strings are marked to trigger unconditional evaluation and special display behaviour.
 */

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
// This post-processing is mandatory; this is automatically triggered by the other relevant methods.
//
// It returns the first error encountered, but processes the whole tree.
// This method must still be called even if [make_ast] returned an error.
// The reason is that [make_ast]'s error handling works in a way that always produces an actual tree, whose guarantees are no different from the success case.
//
// For such trees, handleSyntaxConditions might return a different error (or no error at all);
// For user-facing methods, the error returned by [make_ast] needs to take precedence over the one by handleSyntaxConditions.
//
// Note that the code currently assumes that handleSyntaxConditions is always called through [ast_root], never directly on other nodes.
func (a ast_root) handleSyntaxConditions() (err error) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child") // cannot happen
	}
	// a.syntaxHandled is a flag set to indicate that we already called [handleSyntaxConditions]
	// No need to process the tree twice.
	// NOTE: Some other node handlers currently assume that handleSyntaxConditions is never called twice on them, so this "optimization" is actually mandatory atm.
	if a.syntaxHandled {
		return a.argumentError
	}
	// proceed with child (probably of type list).
	// Note that we store the error inside a.argument error, to avoid processing everything multiple times.
	a.argumentError = a.ast.handleSyntaxConditions()
	a.syntaxHandled = true
	return a.argumentError
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
//
// For ast_list, we just call it on each child and report the first error. Note that we do not abort on first error.
func (a ast_list) handleSyntaxConditions() (err error) {
	if *a == nil { // Note: *a has type (based on) []ast_I
		panic(ErrorPrefix + "invalid syntax tree: unitialized list")
	}
	for _, ast := range *a {
		// We report the first error, but do process all nodes.
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
// and ensures that errors are handled correctly later.
//
// For ast_string nodes, there are no failure cases
func (a ast_string) handleSyntaxConditions() error {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
//
// For ast_parentPercent, there are no failure cases
func (a ast_parentPercent) handleSyntaxConditions() error {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
//
// For ast_parentDollar, there are no failure cases
func (a ast_parentDollar) handleSyntaxConditions() error {
	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
//
// For ast_fmt, we check the variable name and the format verb
// On error, we flag the ast_fmt node by setting abase.errorString. If non-nil, this gets displayed by Interpolate instead of using [fmt].
func (abase *base_ast_fmt) handleSyntaxConditions() error {

	// abase.errorString is supposed to be only set by this method, and we never call it twice.
	// (because the root node detects that handleSyntaxConditions was already called)
	// If we change code such that this assumption no longer is guaranteed, this needs to be reviewed here.
	if abase.errorString != nil {
		panic("Cannot happen")
	}

	if abase.variableName == "" {
		panic(ErrorPrefix + "Uninitialized variable name") // ought to have been caught by the parser and replaced by `v`
	}

	if strings.ContainsRune(abase.formatString, '%') {
		abase.errorString = fmt.Errorf(`<!Format Verb %s for %s contains %%>`+abase.formatString, abase.variableName)
		return fmt.Errorf(ErrorPrefix+`Interpolation string contains a format string verb %s, which contains  '%%'. This will not work with the fmt package`, abase.formatString)
	}

	// Note: if we detect an invalid variable name, the actual format verb does not affect or appear in the output. This is considered OK.

	if abase.variableName[0] == specialVariableNameIndicator {
		if !utils.ElementInList(abase.variableName, validMapSelectors[:]) {
			abase.errorString = fmt.Errorf(`<!Variable name %s in interpolation string starting with %s not recognized by the language>`, abase.variableName, string(specialVariableNameIndicator))
			return fmt.Errorf(ErrorPrefix+"Variable name %s in interpolation string starting with %s is not recognized by the language", abase.variableName, string(specialVariableNameIndicator))
		}
	} else if !ValidInterpolationName(abase.variableName) {
		abase.errorString = fmt.Errorf(`<!Variable name %s not allowed by the language`, abase.variableName)
		return fmt.Errorf(ErrorPrefix+"Variable name %s is not allowed by the language", abase.variableName)
	}

	return nil
}

// handleSyntaxConditions is used to post-process the ast after calling [make_ast]
//
// It checks that the strings given as format verbs, conditions, variable names satisfy specific constraints
// and ensures that errors are handled correctly later.
//
// For ast_cond, we just flag the conditional as invalid on failure.
func (abase *base_ast_condition) handleSyntaxConditions() error {
	if !validConditionString(abase.condition) {
		// make_invalid(3) causes Interpolate to display children unconditionally && display an error message containing the condition.
		abase.make_invalid(3)
		return fmt.Errorf(ErrorPrefix+"invalid condition string: %s", abase.condition)
	} else {
		return nil
	}
}

/**
 *
 * VerifyParameters_direct and VerifyParameters_passed defined here.
 *
 * VerifyParameters_direct checks whether
 *  - %w is only used if there is a parent baseError
 *  - $w is only used if there is a parent baseError that supports this
 *  - variable names referred to by %verb{variable} are actually present in the ParamMap
 *
 * VerifyParameters_passed checks whether
 *  - variable names referred to by $verb{variable} are actually present in the (passed through) ParamMap
 *
 * If a condition is known not be taken, the subtree is ignored.
 * For VerifyParameters_passed, this means we evaluate all conditions and check whether they are taken.
 * For VerifyParameters_direct, we only do so for %cond{}, but not $cond{}. For the latter, we assume the branch is taken.
 *
 * We only report the first error encountered.
 * Note that both parse errors and syntax errors uncovered by [handleSyntaxConditions] take priority.
 * If there was an parse or syntax error, we always report it instead.
 */

// NOTE: We assumes the ast was created by [make_ast] and we ensure [handleSyntaxConditions] was called for post-processing.
// Furthermore, we assume that all calls go through ast_root.
// In particular, we detect errors recorded by [make_ast] and [handleSyntaxConditions] at the root and never
// process the tree.

// For parameters_passed, note that is should never be nil.

// VerifyParameters_direct checks whether the AST contains any parse or syntax errors that were recorded when creating it.
// If not, it checks whether parameters in %fmtVerb{VariableName} expressions are actually present and
// %w and $w expressions refer to valid baseErrors.
// It also recursively checks the baseError, if referred to via %w or $w.
// Untaken conditional branches are not checked (but parse or syntax errors there will be reported)
// Only the first error is reported.
//
// The method uses parameters_direct its for variables and baseError as the baseError.
//
// VerifyParameters_direct for the root node just checks for errors (those were recorded in the root node by [make_ast], and [handleSyntaxConditions] and hands off to the child)
func (a ast_root) VerifyParameters_direct(parameters_direct ParamMap, baseError error) error {

	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child") // cannot happen
	}

	syntaxError := a.handleSyntaxConditions() // ensure this is called
	// If [make_ast] detected a parse error, this overrides any further tests.
	if a.parseError != nil {
		return a.parseError
	}

	if syntaxError != nil {
		return syntaxError
	}

	return a.ast.VerifyParameters_direct(parameters_direct, baseError)
}

// VerifyParameters_passed checks whether the AST contains any parse or syntax errors that were recorded when creating it.
// If not, it checks whether parameters in %fmtVerb{VariableName} or $fmtVerb{VariableName} expressions are actually present and
// %w and $w expressions refer to valid baseErrors.
// It also recursively checks the baseError, if referred to via %w or $w, the the base error supports this (via ValidateError_Params())
// Untaken conditional branches are not checked (but parse or syntax errors there will be reported)
// Only the first error is reported.
//
// The method uses parameters_direct resp. parameters_passed for its for variables and baseError as the baseError.
// NOTE: parameters_passed must not be nil (an empty map is fine), to avoid confusion.
// The special-cased meaning of parameters_passed == nil in [ValidateError_Params] from the [ErrorWithData_any] or [ErrorInterpolater] interface
// needs to be handled by [ValidateError_Params]
//
// VerifyParameters_passed for the root node just checks for errors (those were recorded in the root node by [make_ast], and [handleSyntaxConditions] and hands off to the child)
func (a ast_root) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) error {

	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}

	if parameters_passed == nil {
		panic(ErrorPrefix + "VerifyParameters_passed called with nil map for parameters_passed")
	}

	syntaxError := a.handleSyntaxConditions() // ensure this is called
	// If [make_ast] detected a parse error, this overrides any further tests.
	if a.parseError != nil {
		return a.parseError
	}

	if syntaxError != nil {
		return syntaxError
	}
	return a.ast.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)

}

// VerifyParameters_direct for list nodes just reports the first error in a child
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

// VerifyParameters_passed for list nodes just reports the first error in a child
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

// VerifyParameters_direct for string nodes never reports an error
func (a ast_string) VerifyParameters_direct(ParamMap, error) error {
	return nil
}

// VerifyParameters_passed for string nodes never reports an error
func (a ast_string) VerifyParameters_passed(ParamMap, ParamMap, error) error {
	return nil
}

// VerifyParameters_direct for %fmtVerb{variableName} checks whether the paramter is present.
func (a ast_fmtPercent) VerifyParameters_direct(parameters_direct ParamMap, _ error) (err error) {

	_, ok := parameters_direct[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

// VerifyParameters_passed for %fmtVerb{variableName} checks whether the paramter is present.
func (a ast_fmtPercent) VerifyParameters_passed(parameters_direct ParamMap, _ ParamMap, _ error) (err error) {

	// same as VerifyParameters_direct. We ignore the parameters_passed map
	_, ok := parameters_direct[a.variableName]
	if !ok {
		return fmt.Errorf(ErrorPrefix+"Interpolations string contains variable name %s, which is not present in the error", a.variableName)
	}
	return nil
}

// VerifyParameters_direct for $fmtVerb{variableName} never reports an error (this is because the parameter might later be filled in)
func (a ast_fmtDollar) VerifyParameters_direct(_ ParamMap, _ error) error {
	return nil
}

// VerifyParameters_passed for $fmtVerb{variableName} checks whether the variable is present in parameters_passed.
//
// NOTE: we assume that parameters_passed is either nil
func (a ast_fmtDollar) VerifyParameters_passed(_ ParamMap, parameters_passed ParamMap, _ error) (err error) {

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
func (a ast_parentPercent) VerifyParameters_direct(_ ParamMap, baseError error) error {
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
func (a ast_parentPercent) VerifyParameters_passed(_ ParamMap, _ ParamMap, baseError error) error {
	// exactly the same as VerifyParamter_direct
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

// VerifyParameters_direct for $w checks whether the base error is present and supports $w.
//
// We also check for problems in the referred baseError via [ValidateError_Base]
//
// NOTE: ValidateError_Base() will call VerifyParameters_direct on the referred error.
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

// VerifyParameters_passed for $w checks whether the base error is present and supports $w
//
// We also check for problems in the referred baseError via [ValidateError_Params]
//
// NOTE: ValidateError_Params(paramters_passed) will call VerifyParameters_passed on the referred error, with parameters_passed passed through.
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

// VerifyParameters_direct for %condition{Subtree} will conditionally check the subtree if the condition holds
func (a ast_condPercent) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an error in the root node.
	}

	if !utils.ElementInList(a.condition, validConditions[:]) {
		panic(fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)) // cannot happen, because it was caught by handleSyntaxConditions
	}

	// We actually evalutate the condition here and only check the subtree if the condition holds.
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifyParameters_direct(parameters_direct, baseError)
		} else {
			return nil
		}

	case ConditionNonEmptyMap:
		if len(parameters_direct) == 0 {
			return nil
		} else {
			return a.child.VerifyParameters_direct(parameters_direct, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition") // cannot happen
	}
}

// VerifyParamters_passed for %condition{Subtree} will conditionally check the subtree if the condition holds.
func (a ast_condPercent) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	// same as VerifyParameters_direct, except for calling the approprite VerifyParamters_passed on the subtree
	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an error in the root node.
	}

	if !utils.ElementInList(a.condition, validConditions[:]) {
		panic(fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)) // cannot happen, because it was caught by handleSyntaxConditions
	}

	// We actually evalutate the condition here and only check the subtree if the condition holds.
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_direct) == 0 {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		} else {
			return nil
		}
	case ConditionNonEmptyMap:
		if len(parameters_direct) == 0 {
			return nil
		} else {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
	}
}

// VerifyParameters_passed for $condition{Subtree} will always check the subtree.
//
// This is because using the error as a base may actually cause the condition to be true;
// The errors potentially detected in the subtree are only those that would be errors for _any_ choice of passed parameters, so we want to
// detect those.
func (a ast_condDollar) VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error) {
	return a.child.VerifyParameters_direct(parameters_direct, baseError)
}

// VerifyParamters_passed for $condition{Subtree} will conditionally check the subtree if the condition holds.
func (a ast_condDollar) VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error) {
	if !a.is_valid() {
		panic("Cannot happen") // caught by root node. Anything that would set this also sets an error in the root node.
	}

	if !utils.ElementInList(a.condition, validConditions[:]) {
		panic(fmt.Errorf(ErrorPrefix+"invalid condition string: %s", a.condition)) // cannot happen, because it would have been caught by handleSyntaxConditions
	}

	// We actually evalutate the condition here. If the condition is false, we weaken the child-check to syntax only
	switch a.condition {
	case ConditionEmptyMap:
		if len(parameters_passed) == 0 {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		} else {
			return nil
		}
	case ConditionNonEmptyMap:
		if len(parameters_passed) == 0 {
			return nil
		} else {
			return a.child.VerifyParameters_passed(parameters_direct, parameters_passed, baseError)
		}
	default:
		panic(ErrorPrefix + "Unsupported condition")
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
// (Note: We could also let the caller or each node do that, but doing it here is more convenient)
// paramters_direct should not be nil (use an empty map instead)
//
// Error handling: Note that [make_ast] always outputs a valid tree that contains a in-band error message.
// We also ensure that [handleSyntaxConditions] has been called.
// In either case, we just process it normally.
// Note that on parse error, after the first error, [make_ast] has turned all special tokens inactive, so %w and $w
// and formatted parameter output might be suppressed.
// [handleSyntaxConditions] has marked ast_fmt nodes and ast_cond nodes. These will be processed by Interpolate.
// Since special tokens may have been rendered inactive and output not what the user expected,
// we always append a diagnostic *after* processing the tree normally and always explicitly print
// the parent error and the full parameter map. This is to ensure that, if this ends up in some log file, the relevant information is there.
func (a ast_root) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {
	if a.ast == nil {
		panic(ErrorPrefix + "invalid syntax tree: root has no child")
	}

	// Ensure handleSyntaxConditions is called at least once and check whether there is any parse or syntax error in the ast.
	var hasError bool = a.handleSyntaxConditions() != nil
	hasError = hasError || (a.parseError != nil)

	// NOTE: Even if hasError is true, we still process everything.

	// parameters_passed == nil is (mostly) treated as if parameters_passed was equal to parameters_direct.
	// Note that there is a minor difference in error handling below in that we don't output the parameters twice in that case.
	if parameters_passed == nil {
		a.ast.Interpolate(parameters_direct, parameters_direct, baseError, s)
	} else {
		a.ast.Interpolate(parameters_direct, parameters_passed, baseError, s)
	}

	// Extended diagnostic in case of parse or syntax error.
	if hasError {
		if baseError != nil {
			s.WriteString("\nBase error:\n")
			s.WriteString(baseError.Error()) // Note: We don't check for baseError.(ErrorInterpolater), because we output the parameters anyway.
		}
		if len(parameters_direct) != 0 {
			s.WriteString("\nParameters in error:\n")
			fmt.Fprintf(s, "%v", parameters_direct)
		}
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
	s.WriteString(string(a)) // NOTE: need string(a), not a.String() here; the latter would add literal "-marks.
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
// PercentOrDollar is a literal '$' or '%', required for error handling
func (a *base_ast_fmt) interpolate_helper(parameters_relevant ParamMap, s *strings.Builder, PercentOrDollar rune) {
	// Check whether [handleSyntaxConditions] has detected an error. If so, output a replacement error message instead.
	if a.errorString != nil {
		s.WriteString(a.errorString.Error())
		return
	}

	var value any
	var ok bool
	if utils.ElementInList(a.variableName, validMapSelectors[:]) {
		value = parameters_relevant
		ok = true
		if value == nil {
			value = make(ParamMap) // nil -> empty map. This should not happen, but better safe than sorry.
		}
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
		fmt.Fprintf(s, "%"+a.formatString, value) // NOTE: a.formatString is guaranteed not to contain further %'s. At any rate, fmt.Fprintf would handle it just fine (by printing an error).
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For %condition{subtree}, check the condition and (possibly) evaluate the subtree.
//
// NOTE: condition nodes flagged with errors and handled specially:
// We have a flag for "always evaluate the subtree" and a flag for "display condition string"
func (a ast_condPercent) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {

	// invalidparse & 2 marks whether we should output the condition string.
	if (a.invalidParse & 2) != 0 {
		s.WriteString(`%!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
	}

	// invalidParse & 1 marks whether we should output the children unconditionally.
	if (a.invalidParse & 1) != 0 {
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
		// cannot happen, because handleSyntaxConditions would have set a.invalidParse
		panic(ErrorPrefix + "Unsupported condition")
	}
}

// Interpolate is used to produce the actual output string by appending to *s.
//
// For $condition{subtree}, check the condition and (possibly) evaluate the subtree.
//
// NOTE: condition nodes flagged with errors and handled specially:
// We have a flag for "always evaluate the subtree" and a flag for "display condition string"
func (a ast_condDollar) Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder) {

	// invalidparse & 2 marks whether we should output the condition string.
	if (a.invalidParse & 2) != 0 {
		s.WriteString(`$!<INVALID CONDITION:`)
		s.WriteString(a.condition)
		s.WriteRune('>')
	}

	// invalidParse & 1 marks whether we should output the children unconditionally.
	if (a.invalidParse & 1) != 0 {
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
		// cannot happen, because handleSyntaxConditions would have set a.invalidParse
		panic(ErrorPrefix + "Unsupported condition")
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
