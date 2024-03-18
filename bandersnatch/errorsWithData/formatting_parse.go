package errorsWithData

import (
	"fmt"
	"strings"

	"github.com/GottfriedHerold/Bandersnatch/internal/stack"
)

// Parsing and Interpolating format strings goes through multiple steps:
//
//   - Tokenize the interpolation string
//   - Parse the tokenized string into a syntax tree
//   - [Optional] Perform some validity checks. (3 subchecks, actually. Those would be checked when actually producing output anyway, but sometimes we want those checks early)
//   - Actually prodcuce the interpolated error string.

// This file contains the code for parsing into a syntax tree.

// Our syntax trees are trees, where each node is of (interface) type ast_I
// Depending on the concrete type of the node, a node may reference other child nodes (either as *ast_I or ast_I directly. Our implementation stores ast_I directly).

// We have the following types of nodes:
//   - ast_root:		type of the root node. Has 1 child (likely of type ast_list). This serves just to be able to treat this case specially.
//   - ast_list:		a list of children nodes. When interpolating the final error strings, we just concatenate the child results.
//						This is the basic mode of operation for interpolation strings such as `foo%w bar baz etc`
//   - ast_string:  	string literal (to be output as string)
//   - ast_fmtPercent:	%fmtString{VariableName}. No children. fmtString and VariableName stored directly
//   - ast_fmtDollar:   $fmtString{VariableName}. No children. fmtString and VariableName stored directly
//   - ast_parentPercent: %w. No children
//   - ast_parentDollar:  $w. No children
//   - ast_condPercent:   %!Cond{SubInterpolationString}. 1 child (typically of type ast_list). Condition stored directly.
//   - ast_condDollar:    $!Cond{SubInterpolationString}. 1 child (typically of type ast_list). Condition stored directly.

// The algorithm and the data structures suffer from excessive indirection (mostly due to language restrictions and the lack of unique-ptrs / ownership semantics in the language)
// For clarity, the ast_I interface is satisfied by those ast_foo-types themselves, not by &ast_foo; the explicit type is only really needed for type assertions.
// Quite possibly, the ast_foo-types are (aliases for) pointer types, in particular if the specific ast_foo - type supports some modifying operation.
// The issue here is that nodes need to store their children as (possibly pointers to) ast_I;
// Due to the way interfaces work in Go, assigning a concrete type to an interface and type-asserting back each makes a copy.
// There is no way to modify a value or to obtain a pointer to a value stored inside an interface. [1]
// However, our parser builds up the nodes piece by piece.
// As opposed to a "standard LL(1)" parser, we have no reduce step that creates a node.
// We create a node as soon as we read the first symbol of the "production rule" [2].
// Consequently, we need to be able to modify (existing) nodes.
// The convention is that
//  - modification of any ast_I should be done via type-assertion and calling an appropriate modifying method.
//  - assignment is shallow (i.e. y = x; y.Change() should semantically change x)
// Consequently, creating nodes needs to be done by new_ast_foo functions (there needs to be some kind of indirection, so zero values will likely be invalid nodes, depending on node type)

// [1]: The internal implementation of variable x of interfaces type uses a pair (type_info, STH), where STH is either a value of a pointer to it (depending on size of the type).
// If the type changes, the memory for STH is reused.  Acquiring a pointer to a value-stored STH, then changing the values of x to something of a different type would result in a pointer of
// of type *T pointing to something of type quite different from T, leading to disaster (even if not dereferenced, the garbage collector might misbehave badly if some stray pointers are kept around in non-garbage collected memory / STH contains pointers).
// Of course, things work fine if STH is itself a pointer -- which is kind-of enforced by making *T satisfy the interface: Then the interface directly stores a value of type *T.
// [2]: With usual production rules in (non-extended) BNF, a standard approach would not lead to ast_list storing n elements, but to a (binary) right/left-leaning tree.
// We take "production rule" to allow List -> SequenceElement* rules.
// This is in fact the only reason (apart from defaulting to 'v' for fmtString, which could be inserted by the tokenizer) why the language is not LL(0);

// ast_I is an interface type that is satisfied by all nodes (and in particular by the root) of the abstract syntax tree that we
// parse interpolation strings into.
//
// Note that all implementing types are prefixed with ast_.
// These types may be aliases to pointer types and it's always the ast_foo type (not *ast_foo) that satisfies ast_I.
// Assignment of any such ast_foo - type is always shallow. (i.e. y=x; y.modify(...) modifies x as well)
type ast_I interface {
	IsNode()        // Only to "mark" relevant types
	String() string // Only used for debugging and testing (some test-cases compare against an expected output of String())

	// Interpolate returns a string (by appending to *s) by actually evaluating the subtree below that node.
	// parameters_direct, parameters_passed and baseError are used to evaluate special tokens.
	// parameters_passed == nil has the special meaning of not using this feature and behaves mostly like parameters_passed == parameters_direct.
	// This is very different from parameters_passed being an empty map. parameters_direct should not not be nil (use an empty map instead)
	//
	// Appending to *s rather than returning a string is purely for efficiency reasons.
	Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder)

	// handleSyntaxConditions handles the following syntactic conditions on nodes:
	// - literal % in formatVerbs
	// - invalid variable names
	// - unrecognized conditions
	//
	// This checks for the presence of these errors in the subtree of the given node and returns the first error.
	// This methods also actually *modifies* the tree to handle the error. In particular, if called on the root, it actually records the error.
	// We assume that this method is called on the root node after [make_ast].
	//
	// NOTE: We could handle these errors during [make_ast], but it feels cleaner to separate that (as [make_ast] is already too complicated)
	handleSyntaxConditions() (err error)

	// VerifyParameters_direct report syntax or interpolation errors from the subtree below that node.
	// Note that we may cut corners here and only require this to be accurate for the root
	// parameters_direct and baseError are used for the interpolation. We assume parameters_direct to be non-nil.
	VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error)

	// VerifyParameters_passed report syntax or interpolation errors from the subtree below that node
	// note that we may cut corners here and only require this to be accurate for the root.
	// parameters_direct, parameters_passed and baseError are used for the interpolation. We assume parameters_direct to be non-nil.
	// parameters_passed == nil has the special meaning of not using this feature (and behaves like parameters_passed == parameters_direct)
	// This is very different from parameters_passed being an empty map.
	VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error)
}

/*
Replaced by consolidated version below that is nicer for testing

// We add interfaces for extra functionality that is shared by multiple node types:
// This allows to cut down the state space that we need to (explicitly) track in our DFA.
type (
	childSetter        interface{ set_child_list(ast_list) }      // ast_root, ast_condPercent, ast_condDollar -- set child ast (only list)
	variableNameSetter interface{ set_variableName(stringToken) } // ast_fmtPercent, ast_fmtDollar -- set variable name
	fmtStringSetter    interface{ set_formatString(stringToken) } // ast_fmtPercent, ast_fmtDollar -- set format string
	conditionSetter    interface{ set_condition(stringToken) }    // ast_condPercent, ast_condDollar -- set condition string
	simplifier         interface{ simplify() }                    // ast_root, ast_condPercent, ast_condDollar -- simplify the tree (replace one-element child list by single ast)
	invalidatable      interface{ make_invalid() }                // ast_condPercent, ast_condDollar -- invalidates the condition (indicates that there was a parse error in its subtree)
	initialTokenGetter interface{ token() string }                // ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar [Only used for error reporting] -- returns '%', '$', '%!' or '$!' depending on node type.
	conditionGetter    interface{ get_condition() string }        // ast_condPercent, ast_condDollar -- getter for the condition string
	variableNameGetter interface{ get_variableName() string }     // ast_fmtPercent, ast_fmtDollar -- getter for variable name
	fmtStringGetter    interface{ get_formatString() string }     // ast_fmtPercent, ast_fmtDollar -- getter for format string
)
*/

// consolidated:

type (
	ast_with_children interface {
		ast_I
		set_child_list(ast_list)
		get_children() ast_I
		simplify()
	}
	ast_fmt interface {
		ast_I
		set_variableName(stringToken)
		get_variableName() string
		set_formatString(stringToken)
		get_formatString() string
		token() string
	}
	ast_cond interface {
		ast_I
		set_condition(stringToken)
		get_condition() string
		token() string
		make_invalid(flags int)
		is_valid() bool
		simplify()
		set_child_list(ast_list)
	}
)

// *****
// DEFINITIONS OF THE INDIVIDUAL IMPLEMENTATIONS OF NODE TYPES:
// *****

/*
 * ast_root
 */

// ast_root is the type for the root of our abstract syntax trees.
type (
	v_ast_root struct {
		// actual "child" ast.
		ast ast_I
		// parseError is non - nil if there was a parse error when this tree was constructed.
		// This is needed to make any Verify - function fail early.
		// It causes Interpolate to unconditionally output all the base error and all parameters
		// parseError takes precendence over argumentError
		parseError error
		// argumentError is non-nil if there was a syntax error with the argument of a token.
		// It is set by calling [handleSyntaxConditions] on the root, which needs to be done after [make_ast]
		// Notably, it records if one of the following has occurred:
		// a fmtVerb contains a %
		// a condition string was not recognized
		// a variable name was invalid
		// Either of these causes Interpolate to unconditionally output all parameters.
		argumentError error

		// Set to true if [handleSyntaxConditions] was called once.
		syntaxHandled bool
	}
	ast_root = *v_ast_root
)

// simplify replaces the child list by its single entry if the list has length 1.
// This is accessible via the simplifier interface.
//
// This currently assumes the child is of type [ast_list], so it must not be called twice on the same receiver.
func (a ast_root) simplify() {
	(*a).ast = (*a).ast.(ast_list).squash_list()
}

// new_ast_root creates a new node of type root. Its child node is nil, so you must call [set_child_list] afterwards.
//
// An ast_root with nil child is considered invalid. Add an empty list as child if needed.
func new_ast_root() ast_root {
	return &v_ast_root{}
}

// set_child_list modifies a, setting the child ast to the provided child.
//
// set_child_list is accessible via the [childSetter] interface
func (a ast_root) set_child_list(child ast_list) {
	(*a).ast = child
}

// get_children returns the child ast. The returned value is typically of type ast_list.
//
// This method is only used in testing
func (a ast_root) get_children() ast_I {
	return (*a).ast
}

/*
 * ast_list
 */

// ast_list is the node type for lists of ast's. Can only occur as child of [ast_root], [ast_condPercent] or [ast_condDollar].
type (
	v_ast_list []ast_I
	ast_list   = *v_ast_list
)

// new_ast_list creates a new node of list type. The newly created node is a valid (empty) list
func new_ast_list() ast_list {
	v := make(v_ast_list, 0)
	return &v
}

// append_ast appends a new node to the list of children. This is needed for error handling.
func (al ast_list) append_ast(a ast_I) {
	*al = append(*al, a)
}

// remove_last removes that last added child node from the list.
//
// This method asserts that the list has lenght >0. It is only used during rollback on certain parse errors.
func (al ast_list) remove_last() {
	*al = (*al)[0 : len(*al)-1]
}

// squash_list returns an equivalent ast:
// If len(al) != 1, returns itself. For single-element lists, returns the sole enty.
func (al ast_list) squash_list() ast_I {
	if len(*al) == 1 {
		return (*al)[0]
	} else {
		return al
	}
}

/*
 * ast_string
 */

// ast_string is the node type for string literals. Note that string literals means string literals to be output as such.
// Names of Variables / formatStrings etc. are NOT stored with this node type. Those are stored directly as entries of their ast_fmt nodes and not as tree nodes at all.
type ast_string string

// new_ast_string creates a leaf node with the given string literal.
func new_ast_string(s stringToken) ast_string {
	return ast_string(s)
}

/*
 * ast_fmtPercent
 * ast_fmtDollar
 */

// base_ast_fmt is a helper type for joint functionality of [ast_fmtPercent] and [ast_fmtDollar]
// These types both struct-embedd base_ast_fmt.
type base_ast_fmt struct {
	formatString string
	variableName string
	errorString  error
}

// ast_fmtPercent and ast_fmtDollar are nodes for %fmtString{VariableName} and $fmtString{VariableName} expressions.
// These are leaves. The fmtString and VariableName entries are stored directly, not in child-nodes.
type (
	v_ast_fmtPercent struct{ base_ast_fmt }
	v_ast_fmtDollar  struct{ base_ast_fmt }
	ast_fmtPercent   = *v_ast_fmtPercent
	ast_fmtDollar    = *v_ast_fmtDollar
)

// set_formatString sets the format string of the [ast_fmtPercent] or [ast_fmtDollar]
func (a *base_ast_fmt) set_formatString(formatString stringToken) {
	a.formatString = string(formatString)
}

// set_variableName sets the variableName of the [ast_fmtPercent] or [ast_fmtDollar].
// Note that there is no validation in this function regarding potential validity of the given string as an valid variableName (being exported, not containing "." etc).
func (a *base_ast_fmt) set_variableName(variableName stringToken) {
	a.variableName = string(variableName)
}

// get_variableName reports the name of the variable.
// It is provided to make [ast_fmtPercent] and [ast_fmtDollar] both satisfy the [variableNameGetter] interface.
func (a *base_ast_fmt) get_variableName() string {
	return a.variableName
}

// get_formatString report the format string.
// It is provided to make [ast_fmt_percent] and [ast_fmt_dollar] both satisfy the [fmtStringGetter] interface.
func (a *base_ast_fmt) get_formatString() string {
	return a.formatString
}

// new_ast_fmtPercent creates a new node of type [ast_fmtPercent]. Its formatString and variableName have yet to be set.
func new_ast_fmtPercent() ast_fmtPercent {
	return &v_ast_fmtPercent{}
}

// new_ast_fmtPercent creates a new node of type [ast_fmtDollar]. Its formatString and variableName have yet to be set.
func new_ast_fmtDollar() ast_fmtDollar {
	return &v_ast_fmtDollar{}
}

// token returns a literal '%' for [ast_fmtPercent].
//
// This is provided to satisfy [initialTokenGetter] and unify cases in error reporting.
func (a ast_fmtPercent) token() string {
	return `%`
}

// token returns a literal '$' for ast_fmtDollar.
//
// This is provided to satisfy [initialTokenGetter] and unify cases in error reporting.
func (a ast_fmtDollar) token() string {
	return `$`
}

/*
 * ast_parentPercent
 * ast_parentDollar
 */

// ast_parentPercent and ast_parentDollar are the (stateless) leaf nodes for %w and $w in interpolationg strings.
// These model references to the wrapped error.
type (
	ast_parentPercent struct{} // ast_parentPercent is the leaf node for %w - entries
	ast_parentDollar  struct{} // ast_parentDollar is the leaf node for $w - entries
)

// new_ast_parentPercent creates a new node of type [ast_parentPercent]. This is ready to use.
// Note that the parsing step does not know about the actual parent error, so there is no validity check (is there a non-nil parent error?) here.
func new_ast_parentPercent() ast_parentPercent {
	return ast_parentPercent{}
}

// new_ast_parentDollar creates a new node of type [ast_parentDollar]. This is ready to use.
// Note that the parsing step does not know about the actual parent error, so there is no validity check (is there a non-nil parent error that supports this?) here.
func new_ast_parentDollar() ast_parentDollar {
	return ast_parentDollar{}
}

/*
 * ast_condPercent
 * ast_condDollar
 */

// potential values for base_ast_condition.invalidParse
const (
	astConditionValidity_VALID            = 0
	astConditionValidity_OUTPUT_CHILD     = 1
	astConditionValidity_OUTPUT_CONDITION = 2
)

// base_ast_condition is a helper type for joint functionality of [ast_condPercent] and [ast_condDollar] (via struct embedding)
type base_ast_condition struct {
	condition string // condition string that controls under what condition child is interpolated.
	child     ast_I  // child node. During construction of the tree, we always initialize this with a [ast_list]-node that may later be replaced by a non-list node.
	// invalidParse is set if there was a error when creating this node and the error happened after condition was read.
	// Additionally, this flag is set if there was a parse error in the child subtree.
	// This flag then signals whether we should output the condition string and child unconditionally.
	// The purpose of this behaviour is to give better output diagnostics in case of errors; in particular,
	// we need to ensure that errors are not hidden by a condition that would not output the child.
	//
	// Note that not all errors need to set this flag.
	// values are:
	//
	// 0: OK
	// 1: Output child unconditionally
	// 3: Output child unconditionally and make an error output with the condition
	invalidParse int

	// Note: we store invalidParse rather than validParse, because this way, the zero value makes newly generated instances valid (it was originially a bool).
}

// set_condition sets the condition string for an [ast_condPercent] or [ast_condDollar].
// Note that this functions does not perform any validity checks on cond.
//
// This method is accessed via the [conditionSetter] interface
func (a *base_ast_condition) set_condition(cond stringToken) {
	a.condition = string(cond)
}

// get_condition is the getter for the condition string.
//
// It is provided to satisfy the [conditionGetter] interface.
func (a *base_ast_condition) get_condition() string {
	return a.condition
}

// make_invalid sets the node of type [ast_condPercent] or [ast_condDollar] to invalid.
//
// This is provided to satisfy the [invalidatable] interface. It is called when certain errors during parsing (in particular errors in the child-subtree).
// This is caught by [Interpolate] and causes special treatment of output; in particular, it causes unconditional interpolation of the child subtree in order
// to ensure that the cause of parsing errors is actually displayed.
func (a *base_ast_condition) make_invalid(flags int) {
	a.invalidParse |= flags
}

// is_valid returns whether the node of type [ast_condPercent] or [ast_condDollar] is valid
//
// This returns true unless [make_invalid] has been called on the node, which happens on certain parse errors.
// This method may potentially be unused outside of testing.
func (a *base_ast_condition) is_valid() bool {
	return (a.invalidParse == astConditionValidity_VALID)
}

// set_child_list sets the child node of the node of type [ast_condPercent] or [ast_condDollar].
//
// This is provided to satisfy the [childSetter] interface.
// During our algorithm to construct the tree, the child is always an [ast_list] at first. We may later call simplify() to replace it by a non-list node.
func (a *base_ast_condition) set_child_list(child ast_list) {
	a.child = child
}

// get_children returns the child ast. The returned value is typically of type ast_list (unless simplify was called).
//
// This method is only used in testing.
func (a *base_ast_condition) get_children() ast_I {
	return a.child
}

// simplify is provided for node types [ast_condPercent] and [ast_condDollar] to satisfy the [simplifier] interface.
//
// This type-asserts that the child node is of type [ast_list] and replaces it by it sole entry if the length of that list is 1.
// In particular, simplify must not be called twice on the same node. Only call it when "finalizing".
func (a *base_ast_condition) simplify() {
	a.child = a.child.(ast_list).squash_list()
}

// ast_condPercent and ast_condDollar are the nodes for %!Cond{SubInterpolationString} and $!Cond{SubInterpolationString}
type (
	v_ast_condPercent struct{ base_ast_condition }
	v_ast_condDollar  struct{ base_ast_condition }
	ast_condPercent   = *v_ast_condPercent
	ast_condDollar    = *v_ast_condDollar
)

// new_ast_condPercent creates a new node of type [ast_condPercent].
//
// Its condition is the empty string and the (parsed) SubInterpolationString has yet to be set by set_child_list.
func new_ast_condPercent() ast_condPercent {
	return &v_ast_condPercent{}
}

// new_ast_condPercent creates a new node of type [ast_condPercent].
//
// Its condition is the empty string and the (parsed) SubInterpolationString has yet to be set by set_child_list
func new_ast_condDollar() ast_condDollar {
	return &v_ast_condDollar{}
}

// token returns a literal '%!' for [ast_condPercent].
//
// This is provided to satisfy [initialTokenGetter] and unify cases in error reporting.
func (a ast_condPercent) token() string {
	return `%!`
}

// token returns a literal '$!' for [ast_condDollar].
//
// This is provided to satisfy [initialTokenGetter] and unify cases in error reporting.
func (a ast_condDollar) token() string {
	return `$!`
}

// All ast_foo - types have an IsNode() method to signal they are intented to satisfy ast_I.
// This is mostly to ensure that other types (such as helper types like base_ast_condition) don't accidentially satisfy ast_I.
// This helps to prevent errors when writing code.

func (a ast_root) IsNode()          {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_list) IsNode()          {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_string) IsNode()        {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_fmtPercent) IsNode()    {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_fmtDollar) IsNode()     {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_parentPercent) IsNode() {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_parentDollar) IsNode()  {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_condPercent) IsNode()   {} // IsNode is a dummy method provided to satisfy [ast_I]
func (a ast_condDollar) IsNode()    {} // IsNode is a dummy method provided to satisfy [ast_I]

// We define String methods on each AST node type. These are exclusively used for debugging and testing.
// (Notably, to write down test-cases compactly.)

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_root) String() string {
	return "AST(" + (a.ast).String() + ")"
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_list) String() string {
	var b strings.Builder
	b.WriteRune('[')
	for i, c := range *a {
		if i > 0 {
			b.WriteRune(',')
		}
		b.WriteString(c.String())
	}
	b.WriteRune(']')
	return b.String()
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_string) String() string {
	return `"` + string(a) + `"`
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_fmtPercent) String() string {
	if a.errorString != nil {
		return a.errorString.Error()
	}
	var b strings.Builder
	b.WriteRune('%')
	b.WriteString((*a).formatString)
	b.WriteRune('{')
	b.WriteString((*a).variableName)
	b.WriteRune('}')
	return b.String()
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_fmtDollar) String() string {
	if a.errorString != nil {
		return a.errorString.Error()
	}
	var b strings.Builder
	b.WriteRune('$')
	b.WriteString((*a).formatString)
	b.WriteRune('{')
	b.WriteString((*a).variableName)
	b.WriteRune('}')
	return b.String()
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_parentPercent) String() string {
	return "%w"
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_parentDollar) String() string {
	return "$w"
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_condPercent) String() string {
	var b strings.Builder
	b.WriteRune('%')
	b.WriteRune('!')
	b.WriteString((*a).condition)
	b.WriteRune('{')
	b.WriteString(((*a).child).String())
	b.WriteRune('}')
	return b.String()
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_condDollar) String() string {
	var b strings.Builder
	b.WriteRune('$')
	b.WriteRune('!')
	b.WriteString((*a).condition)
	b.WriteRune('{')
	b.WriteString(((*a).child).String())
	b.WriteRune('}')
	return b.String()
}

// Our parser is essentially a DFA with access to a stack.
// Note that we require only very few states, allowing to write this by hand; this is mostly due to the fact that the language is so simple and because we use some extra Go interfaces.
// As mentioned above, we do not use a "traditional shift/reduce parser", but rather construct the node for the parse tree as soon as we see the first symbol
// involved in the related production rule.
// The language is simple enough to allow this:
// The only ambiguity is list length and optional presence of a fmtString, but these are NOT part of the AST tree structure, since we allow lists of arbitrary length.
// The stack is only needed for nested sub-interpolation strings of type %!COND{...} or $!COND{...}.
// Rather than reducing, we then modify the just-constructed node when reading the next symbols.
// Note that these modification go through an type assertion to an interface such as childSetter that is satisfied by several nodes types.
// This means (from a theory POV) that the dynamic type of the last-constructed node is actually part of the parser state and the interface dispatch implicitly part of the DFA's state transition function,
// but the algorithm does not need to make a (explicit) case distinction, which allows a small and managable state space.

// parseMode is an enum type for the state of our AST parser.
type parseMode int

// possible states for the DFA

const (
	parseMode_Sequence      parseMode = iota // currently reading a sequence of list elements
	parseMode_FmtString                      // expecting a format string (after % or $) or { for default format string
	parseMode_Condition                      // expecting a condition (after %! or $!)
	parseMode_VariableName                   // expecting a variable name
	parseMode_OpenSequence                   // expecting a { to start a sequence (after %!COND or $!COND)
	parseMode_OpenVariable                   // expecting a { to be followed by a variable name (after %fmtString or $fmtString)
	parseMode_CloseVariable                  // expecting a } terminating a variable name
	parseMode_Error                          // set after the first error

	// NOTE: There is no parseMode_CloseSequence: The terminating '}' in %!COND{...} and $!COND{...} is handled by parseMode_Sequence
)

// embeddedParseError is used to create error strings for the purpose of embedding them into the tree.
// the string s may contain formatting verbs understood by [fmt] and args are passed to some fmt formatting function such as [fmt.Sprintf].
//
// The intended usage is to call this function and place it in the tree as a node (for this reason, we return an [ast_string] for convenience).
// By doing that, the string returned from embeddedParseError is displayed whenever the tree is interpolated (i.e. whenever we call Error on the errors returned by the package).
// This is done for diagnostics.
//
// Note that the main reason to use this function (over plain [fmt.Sprintf]) is that we may add some extra error string to designate parsing errors.
// Using this function unifies this.
func embeddedParseError(s string, args ...any) ast_string {
	return new_ast_string(stringToken(fmt.Sprintf(`<!PARSE-ERROR: `+s+`>`, args...)))
}

// make_ast creates a syntax tree out of the list of tokens.
//
// IMPORTANT: Any ast_root (contained in a struct) that is returned by an exported function *must* have
// been post-processed by [handleSyntaxConditions]. While this post-processing is triggered by anything that requires it,
// it modifies the ast on its first call; consequently, forgetting this yields a potential thread-safety issue.
//
// On failure, reports the first error. Note that we do NOT stop on errors;
// we rather process the input to the end and build a meaningful syntax tree.
// The returned syntax tree will contain a diagnostic message (as a valid node of string type).
// Any tokens read after the first error are turned into top-level inactive string tokens (in particular, there will be no more errors). Interpolate will output them unevaluated.
//
// If there is a parse error, the returned err is additionally stored in the (root node of the) returned ret.
// This is needed for diagnostics.
//
// To simplify the parser, this function makes the following assumption about the input list of tokens:
//   - tokens[0] == tokenStart
//   - token[len(token)-1] == tokenEnd
//   - no other tokenStart or tokenEnd appear
//   - no two consecutive tokens have type stringToken
//   - stringTokens are not the empty string (We have no token at all instead).
//
// These assumptions are satisfied by the output of our tokenizer. (note the [validateTokenList] function defined in formatting_test.go that checks this)
// We make no guarantees what happens if these assumptions are not satisfied and do not check this.
// Failures of these assumptions may cause a panic or possibly weird misparses rather than report errors.
//
// Also note that make_ast only constructs the tree. It does not care whether the tokens "make sense".
// In particular, formatVerbs can contain extra "%", Variable names could be unexported and not even valid Go identifiers, conditions not recognized etc.
// These (optional) checks come later.
func make_ast(tokens tokenList) (ret ast_root, err error) {

	// Our parser internally works as follows:
	//
	// We process the input tokenList one-by-one
	// Absent %!cond{...} or $!cond{...} entries, there is actually no nesting involved, so the tree would just be a list
	// (Note that the ast-nodes for %fmtString{Variable} entries actually store fmtString and Variable inside the node, not as children)
	//
	// So our algorithm will read tokens off the input list and append to a list.
	// Any string or %w or $w token read can be processed immediately
	// Reading any %, $, %! or $! token determine what kind of tokens must follow.
	// For % or $, we expect to read fmtString(optional), {, a string, and }. After this, we are done creating a ast_fmtPercent or ast_fmtDollar-node
	// For simplicity, we already create the appropriate node (with as-of-yet empty fmtString and VariableName) of type ast_fmtPercent or ast_fmtDollar when reading the introducing % or $.
	// This temporary node is already appended to the list and additionally the top of the stack.
	// We then set parserState to an appropriate value to interpret the next tokens and actually set fmtString and VariableName.
	// When we finish reading the }, we pop it from the stack.
	// For %! or $!, we use the exact same strategy; however, observe that creating the ast_condPercent or ast_condDollar - node is actually finished
	// after reading %!cond{ or $!cond{, i.e. after the opening `{`
	// To account for the sub-tree, we just push a node of list-type on the stack and then proceed to append to that.
	// Reading the closing `}` just means we pop the stack. Note that the meaning of `}` is unambigous here, because the closing `}` of fmt-nodes is read in a parse modes where we MUST read a `}`.
	//
	// Our default parsing mode therefore is parseMode_Sequence, where we just expect to read the next node to be added to the list.
	// We either read a token
	//  - directly produces a node from a single-token (%w, $w, strings) or
	//  - initiates a ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar - node (%, $, %!, $!)
	//    In this case, we step through appropriate parseMode states to read what follows until the node is finished constructing and end up in parseMode_Sequence again.
	//  - affects the stack: `}` means we pop the stack. `{` is actually invalid (because we only may read as we step through parseMode states following %, $, %!, $!).
	//
	// Note that as a consequence, the stack only contains (starting at the bottom) ast_root - ast_list, followed by any number >=0 of (ast_cond - ast_list) pairs.
	// We note that we actually replace one-element lists by their single element, but this is done after the list is fully processed and does not affect parsing.
	//
	// Error handling:
	// There is actually not terribly many error cases involved here and we handle them as follows:
	//  - We Pop the stack up until we are at the top-level list and enter a special parseMode_Error state
	//    In this state, everything further read will just be treated as an inactive string token to be appended to this top-level list
	//    Any ast_condPercent or ast_condDollar popped this way will be marked as tainted.
	//  - We append a string token that describes the error as an in-band error report
	//  - We also report the error in the root node an function's return value.
	//  - If an error occurs while we are in the process of creating an ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar (i.e. while we step though parseMode states, but read an unexpected token):
	//    Note that we actually already created the ast-node upon reading the introducing %, $, %! or $!. In this case, we roll back and remove that node (replacing it by a string node for error reporting).
	//    This may serve as the in-band error report (Note that it might not in the top-level list)
	//  - The only errors that can occur are tokens of unexpected type, handled as above or
	//    stack errors, meaning that we encounter } when there is no stack to pop or finish reading without popping the stack.
	//  - We use the [set_error] local function to handle the stack popping, tainting ast_cond nodes and returning the error in the root and returned value err.
	//    The in-band error string is processed by [embeddedParseError]. The actual error string is handled slightly differently by these two:
	//    [set_error] prefixes the error with [ErrorPrefix], whereas [embeddedParseError] adds some <!META-ERROR...> tag to make the error stand out.
	//    Both are to follow appropriate conventions: [ErrorPrefix] is used to designate the origin package of the object of type error.
	//    <!META-ERROR...> is there to be consistent with [fmt]'s error reporting.

	// NOTE: All panic(...) calls in this functions are assertions to double-check on (internal) invariants.
	// It is (supposed to be) impossible to trigger those with any input tokenList that is the output of tokenize

	ret = new_ast_root() // Make root node. This is directly stored in ret for simplicity.

	// the top of our tree (apart from the root node) is a list, starting empty.
	initial_list := new_ast_list()
	ret.set_child_list(initial_list)

	// since %!Foo{Bar} and $!Foo{Bar} can be nested, we can actually get a tree of arbitrary depth
	// We maintain a stack that contains the ast_nodes on the current path to the leaf we are working with.
	// Note that we push copies of the nodes (rather than pointers) to the stack.
	// This is fine, because all ast_I - nodes have shallow semantics.
	stack := stack.MakeStack[ast_I]()
	stack.Push(ret)
	stack.Push(initial_list)

	// we skip over the tokenStart (to avoid needing a parseMode_start) and expect a sequence
	if len(tokens) == 0 {
		panic(ErrorPrefix + "empty or nil token list (missing start/end markers)")
	}
	if tokens[0] != tokenStart {
		panic(ErrorPrefix + "invalid token list (missing start marker)")
	}
	tokens = tokens[1:]

	var mode parseMode = parseMode_Sequence // we expect a list of stringTokens, %w, $w etc.

	// set_parse_error is a closure that is called when a parse error is encountered.
	// s is a format string and args are its arguments, used to create the returned error
	//
	// Before or after calling this closure, the parser should embed a diagnostic message as a ast_string into the returned ast.
	// Usually, this diagnostic message resembles s.
	//
	// This closure assumes the stack is in a good state as in parseMode_Sequence.
	// Notably, it contains (starting from bottom): ROOT - LIST, followed by any number of pairs COND - LIST.
	// The conditions in any COND-node on the stack are non-empty strings.
	//
	// We set the error returned by make_ast (both in the ast_root and err), terminate all open ast_cond's
	// (so the resulting stack after the call to set_parse_error is ROOT - LIST) and flag them as invalid.
	// We then set mode to parseMode_Error.
	//
	// The parser will then remain in parseMode_Error, where every input token just gets turned into a string (which can produce no more errors)
	// flagging the ast_cond - path as invalid will make Interpolate ignore the condition.
	// This causes the offending part that caused the parse error to be unconditionally displayed.
	set_parse_error := func(s string, args ...any) {
		// record first found error both in value returned from function and in the returned root node.
		// The latter is done to make sure Validation function can reproduce the error.
		if err == nil {
			err = fmt.Errorf(ErrorPrefix+s, args...)
			ret.parseError = err
		} else {
			// err is only set by set_error.
			// we enter parseMode_Error at the end of set_error. In this parseMode, we can never encounter another error, because
			// we just turn every token that we read from this point on into a string.
			panic("Cannot happen")
		}

		// Assert preconditions on stack shape. (the types of the node are checked via type assertions below)
		// We could work with other preconditions, I guess.
		if stack.Len()%2 != 0 {
			panic("Cannot happen")
		}
		if stack.Len() < 2 {
			panic("Cannot happen")
		}

		// we proceed through the stack and mark any node as invalid if needed.
		// node.make_invalid() only really affects
		// nodes of types ast_condPercent and ast_condDollar and causes
		// the condition to be ignored, so the conditional interpolation is always evaluated.
		// This is mostly to ensure that diagnostic is not hidden.
		for i := 0; i < (stack.Len()/2)-1; i++ {
			_ = stack.Pop().(ast_list)
			condNode := stack.Pop().(ast_cond)
			condition := condNode.get_condition()
			if condition == "" {
				panic("Cannot happen")
			}
			condNode.make_invalid(1)
			condNode.simplify()
		}
		mode = parseMode_Error
	}

	for _, token := range tokens {
		// goto redo can be used to "re-scan" the last token.
		// This is done after certain errors:
		// Re-scanning the token in parseMode_Error will just do the right thing,
		// thereby simplifying the error handling.
	redo:
		// NOTE: We will overwrite top by a type-asserted top:=top.(ast_*) after we branch, since we know more about the type
		// NOTE: To avoid confusion, our convention is to stop using the top variable after any operation that changes the
		// stack shape (set_parse_error, stack.Push, stack.Pop) until we get here again.
		var top ast_I = *stack.Top() // Peek at top of stack. NOTE: stack cannot be empty
		switch mode {
		case parseMode_Sequence: // expect to get a sequence of strings or tokens.
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST followed by any number >=0 of COND,LIST pairs.
			top := top.(ast_list) // top is a ast_list if we are in parseMode_Sequence

			switch token := token.(type) {
			case stringToken: // append token for the string literal in the list and continue in parseMode_sequence
				newNode := new_ast_string(token)
				top.append_ast(newNode)
			case specialToken: // read token that is not a string literal
				switch token {
				case tokenPercent: // create and add node for %fmtString{Variable}. fmtString and VariableName are set later
					newNode := new_ast_fmtPercent()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString // read (optional) format string next
				case tokenDollar: // create and add node for $fmtString{Variable}. fmtString and VariableName are set later
					newNode := new_ast_fmtDollar()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString // read (optional) format string next
				case tokenPercentCond: // create and add node for %!Condition{Sub-AST}. Condition and Sub-AST are set later.
					newNode := new_ast_condPercent()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition // read Condition string next
				case tokenDollarCond: // create and add node for $!Condition{Sub-AST}. Condition and Sub-AST are set later.
					newNode := new_ast_condDollar()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition // read Condition string next
				case tokenOpenBracket: // { without prior %, $, %! or $!
					embeddedErrorNode := embeddedParseError(`Unexpected "{"`)
					top.append_ast(embeddedErrorNode)
					set_parse_error(`Unexpected "{" in format string`)

				case tokenCloseBracket:
					if stack.Len() <= 3 {
						// We always have ROOT-LIST on the stack until we read tokenEnd.
						if stack.Len() != 2 {
							panic("Cannot happen")
						}
						embeddedErrorNode := embeddedParseError(`Unexpected "}"`)
						top.append_ast(embeddedErrorNode)
						set_parse_error(`unexpected "}" in format string`)
						continue // with mode == parseMode_Error, set by set_error
					}
					_ = stack.Pop()                    // type popped is ast_list.
					condNode := stack.Pop().(ast_cond) // type popped is either ast_condPercent or ast_condDollar
					condNode.simplify()                // if the child list contains 1 element, replace list by single element.

					// parseMode stays at parseMode_Sequence

				case tokenEnd: // terminating a list. tokenCloseBracket is for %!Cond{List} and $!Cond{List}. tokenEnd is for the root.
					// recall that stack is (starting from bottom) ROOT, LIST, followed by any number >=0 of (COND, LIST) - pairs
					// tokenEnd must only appear at the top level, if there are no (COND,LIST)-pairse
					// If we read a tokenEnd while the stack size is != 2, we therefore have an unterminated %!COND{... somewhere
					if stack.Len() != 2 {
						set_parse_error(`Missing "}" in format string`)
						// stack length is 2 after calling set_error

						currentNode := (*stack.Top()).(ast_list)
						embeddedErrorNode := embeddedParseError(`Missing "}" in format string`)
						currentNode.append_ast(embeddedErrorNode)
						goto redo // reprocess tokenEnd in parseMode_Error; this is just to simplify the code.
					} else {
						// stack.Len() == 2 is guaranteed
						_ = stack.Pop() // type popped is ast_list.
						// NOTE: We could just do _ = stack.Pop(); ret.simplify(), but this is clearer.
						root := stack.Pop().(ast_root) // type popped is ast_root
						root.simplify()                // if the child list contains 1 element, replace list by single element.

						// The _, token := range tokens - loop terminates now, as tokenEnd was the last token.
						// parseMode stays at parseMode_Sequence
					}

				case tokenParentPercent: // create and add node for %w and continue with the list.
					newNode := new_ast_parentPercent()
					top.append_ast(newNode)
				case tokenParentDollar: // create and add node for $w and continue with the list.
					newNode := new_ast_parentDollar()
					top.append_ast(newNode)
				default:
					panic(fmt.Errorf(ErrorPrefix+"Unhandled token: %v", token)) // cannot happen for tokenLists output by the tokenizer.
				}
			default: // token not of type stringToken or specialToken
				panic(ErrorPrefix + "Invalid entry in token list")
			}

		case parseMode_FmtString: // expect to read (optional) format string (which must be a string literal)
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_FMT
			top := top.(ast_fmt)
			if token == tokenOpenBracket { // %{ or ${ is interpreted as %v{ or $v{
				// We treat an empty format string as 'v'.
				// However, we don't want to just set formatString to 'v' at this point, because this would interact with
				// handling of parse errors: if there is a parse error (such as missing "}" ) in further processing the %{...} - clause
				// we "undo" the parse and just literally output parts of the %{...} - clause that were read so far (together with an error message)
				// If we set the formatString to 'v' here, parsing "%{foo" would result in a confusing "%v{foo" appearing in the error message.
				// So we perform that replacement later.

				mode = parseMode_VariableName // proceed to the variable name
			} else {
				token_string, ok := token.(stringToken) // next token, if not {, must be a literal string
				if !ok {
					// remove the already-place ast_fmtPercent/ast_fmtDollar and replace it by a literal % or $
					percentOrDollar := top.token() // "%"  or "$"
					_ = stack.Pop()
					currentNode := (*stack.Top()).(ast_list) // new top, cannot overwrite top variable (because of type).
					currentNode.remove_last()
					currentNode.append_ast(new_ast_string(stringToken(percentOrDollar)))

					// The case distinctions is just for better error messages.
					if token == tokenEnd {
						embeddedErrorNode := embeddedParseError(`Interpolation string ends in "%s"`, percentOrDollar)
						currentNode.append_ast(embeddedErrorNode)
						set_parse_error(`Interpolation string ends in unescaped "%s"`, percentOrDollar)
					} else {
						embeddedErrorNode := embeddedParseError(`Invalid token "%s" after "%s"`, token.String(), percentOrDollar)
						currentNode.append_ast(embeddedErrorNode)
						set_parse_error(`Invalid token "%s" after "%s"`, token.String(), percentOrDollar) // sets mode to parseMode_Error
					}
					goto redo // re-read actual offending token in parseMode_Error. This also handles token==tokenEnd correctly.

				} else { // ok == true, token_string is an actual string
					top.set_formatString(token_string)
					mode = parseMode_OpenVariable // expect to read {, followed by variable name next
				}
			}
		case parseMode_Condition: // expect to read a condition string (which must be a (non-empty) string literal)
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_COND
			top := top.(ast_cond)
			token_string, ok := token.(stringToken)
			if !ok {

				// remove the already-place ast_condPercent/ast_condDollar and replace it by a literal %! or $!
				percentOrDollarExlamMark := top.token() // "%!"  or "$!"
				_ = stack.Pop()                         // ast_condPercent or ast_condDollar
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarExlamMark)))

				// The case distinctions is just for better error messages.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseError(`Interpolation string ends in "%s"`, percentOrDollarExlamMark)
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Interpolation string ends in unescaped "%s"`, percentOrDollarExlamMark)
				} else {
					embeddedErrorNode := embeddedParseError(`Invalid token "%s" after "%s"`, token.String(), percentOrDollarExlamMark)
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Invalid token "%s" after "%s"`, token.String(), percentOrDollarExlamMark) // sets mode to parseMode_Error
				}
				goto redo // re-read tokenEnd in parseMode_Error

			} else {
				// ok == true, the token we just read is a string. It cannot be empty due to how the tokenizer works.
				top.set_condition(token_string)
				mode = parseMode_OpenSequence // expect to read { next, followed by a sequence.
			}
		case parseMode_VariableName: // expect to read the name of a variable (which must be a string literal) after having processed %fmtString{
			// The stack is (from top to bottom) ast_fmt - {ast_list - ast_cond -}* ast_list - ast_root
			// with the top already containing the format string
			top := top.(ast_fmt)
			token_string, ok := token.(stringToken)
			if !ok {
				// completely remove the ast_fmt and replace it by the literal string that was read so far.
				percentOrDollar := top.token()      // "%" or "$"
				fmtString := top.get_formatString() // format string
				_ = stack.Pop()                     // remove top
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar + fmtString)))

				// add a diagnostic node and call set_error.
				// The case distinction is just to provide better error messages, since tokenEnd.String() would not return the right string.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseError(`Interpolation string ends where variable name was expected`)
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Interpolation string ends where variable name was expected`)
				} else {
					embeddedErrorNode := embeddedParseError(`Got "%v" where variable name was expected`, token.String())
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Got "%v" where variable name was expected`, token.String())
				}
				goto redo // re-read offending token in parseMode_Error

			} else {
				// good case: token is string token. It is non-empty by assumpition on token_list
				top.set_variableName(token_string)
				// stack remains unchanged.
				mode = parseMode_CloseVariable // expect to read } next
			}

		case parseMode_OpenSequence: // expect to read a { after %!COND or $!COND
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_COND
			// The top AST_COND has a non-empty string set as condition
			top := top.(ast_cond)

			// parseMode_OpenSequence only happens after reading a string token in mode parseMode_Condition.
			// Because tokenList does not contain consecutive string tokens, the current token cannot be a string token; panic on type-assertion failure is OK, as it cannot happen.
			token := token.(specialToken)
			if token != tokenOpenBracket {

				// completely remove the ast_cond and replace it by the literal string that was read so far.
				percentOrDollarExclam := top.token() // "%!" or "$!"
				condition := top.get_condition()     // condition
				_ = stack.Pop()
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()

				// Note: The pattern %!Cond with missing { is likely because of a stray %! or $! that is not intended as a condition at all.
				// For that reason, we place the embedded error message just after the %! or $! rather than at the place where we expect the {
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarExclam)))
				embeddedErrorNode := embeddedParseError(`"%v" has no matching "{"`, percentOrDollarExclam)
				currentNode.append_ast(embeddedErrorNode)
				currentNode.append_ast(new_ast_string(stringToken(condition)))
				set_parse_error(`Missing "{" after %v%v`, percentOrDollarExclam, condition)
				goto redo // reread token. This may well be tokenEnd, which is fine.
			} else {
				// good case: We actually read {.
				// At this point, processing the condition node in the tree has finished.
				newList := new_ast_list()
				top.set_child_list(newList)
				stack.Push(newList)

				mode = parseMode_Sequence
			}
		case parseMode_OpenVariable: // expect to read a { initiating a variable name
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_FMT

			// top := top.(ast_fmt) // commented out, because it's only needed in error case

			// parseMode_OpenVariable only happens after reading a string token in mode parseMode_FmtString.
			// Since consecutive string tokens are merged by the tokenizer, panic on type-assertion failure is fine:
			// Missing format string jumps directly from parseMode_FmtString to parseMode_VariableName.
			token := token.(specialToken)
			if token != tokenOpenBracket {

				top := top.(ast_fmt)
				formatString := top.get_formatString() // formatString
				percentOrDollar := top.token()         // "%" or "$"
				// completely remove the ast_fmt and replace it by the literal string that was read so far.
				_ = stack.Pop()
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()

				// Note: The pattern %FmtString or $FmtString with missing { is likely because of a stray unescaped % or $ that is not intended as a formatting string at all.
				// For that reason, we place the embedded error message just after the % or $ rather than at the place where we expect the {

				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar)))

				embeddedErrorNode := embeddedParseError(`unescaped "%v" has no matching "{"`, percentOrDollar)
				currentNode.append_ast(embeddedErrorNode)
				currentNode.append_ast(new_ast_string(stringToken(formatString)))
				set_parse_error(`Missing "{" after %v%v`, percentOrDollar, formatString)
				goto redo // reread token. This may well be tokenEnd, which is fine.
			} else {
				// good case: { was present. Proceed to read variable name
				mode = parseMode_VariableName
			}

		case parseMode_CloseVariable: // expect to read a } after a variable name
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_FMT
			top := top.(ast_fmt)

			formatString := top.get_formatString()
			// Note: If the formatString of the % or $ expression is empty, we need to replace it by v.
			// However, this needs to be done after handling parse errors: on error, we actually output the formatString and outputting a "v" would be confusing.

			// We now handle the parse error case if what we read actually was not a }
			// We previously read a string, so token is guaranteed to be a specialToken (no consecutive string tokens above).
			token := token.(specialToken)
			if token != tokenCloseBracket {
				// We need to insert an error string and a literal interpretation of %FmtString{VariableName

				percentOrDollar := top.token()         // "%" or "$"
				VariableName := top.get_variableName() // variableName
				_ = stack.Pop()

				currentNode := (*stack.Top()).(ast_list)

				currentNode.remove_last()                                                                                // remove the ast_fmtPercent or ast_fmtDollar
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar + formatString + "{" + VariableName))) // replay what was read so far as a plain string
				// case distinction to improve error messages.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseError(`unexpected end of format string after reading a variable name without closing "}"`)
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Variable name not terminated by "}"`)
					goto redo // to actually handle the tokenEnd token as ending the parse.

				} else {
					embeddedErrorNode := embeddedParseError(`Variable name not terminated by "}"`)
					currentNode.append_ast(embeddedErrorNode)
					set_parse_error(`Variable name not terminated by "}"`)
					goto redo // to actually display the current token.
				}
			} else {
				// good case, token == tokenCloseBracket, formatString contains no literal `%`

				if formatString == "" {
					top.set_formatString(stringToken("v"))
				}
				_ = stack.Pop()

				mode = parseMode_Sequence
			}

		case parseMode_Error:
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST.
			currentNode := top.(ast_list) // top is an ast_list node if we are in parseMode_Error
			if stack.Len() != 2 {
				panic("Cannot happen")
			}

			switch token := token.(type) {
			case stringToken:
				currentNode.append_ast(new_ast_string(token))
			case specialToken:
				// do not interpret tokens (except for tokenEnd) but rather output a string representation of it as a plain ast_string.
				if token != tokenEnd {
					currentNode.append_ast(new_ast_string(stringToken(token.String())))
				} else {
					_ = stack.Pop() // type popped is ast_list.
					// Alternatively, we could do _ = stack.Pop(); ret.simplify() to avoid the type assertion
					// The stack.Pop()'s are just to maintain the invariant that the stack is empty after we finish.
					root := stack.Pop().(ast_root)
					root.simplify()
				}
			}

		default:
			panic(ErrorPrefix + "Unhandled status in syntax tree parser") // cannot happen
		}
	}

	// Double-check that the algorithm above terminated in the expected state.

	// The only way to exit the above is reading tokenEnd in parseMode_Sequence or parseMode_Error.
	// (Reading a tokenEnd in other modes causes a switch to parseMode_Error and re-scanning it)
	if (mode != parseMode_Sequence) && (mode != parseMode_Error) {
		panic(ErrorPrefix + "Cannot happen")
	}

	// NOTE: We could remove this and remove the 2 stack.Pop() calls when handling tokenEnd
	// and use ret.simplify() rather then calling simplify on the returned value from Pop()
	if stack.Len() != 0 {
		panic(ErrorPrefix + "Cannot happen")
	}

	// Parse errors are reported both inside the returned ast as well as via the returned err.
	if err != ret.parseError {
		panic(ErrorPrefix + "Cannot happen")
	}

	// err is set iff we end up in parseMode_Error mode.
	if (mode == parseMode_Error) != (err != nil) {
		panic(ErrorPrefix + "Cannot happen")
	}

	return
}

// We might actually move this to testing. However, it fits better here.

// make_ast_successfully is a variant of [make_ast] that panics on failure
//
// This is only used for creating test instances during testing of concrete implementation of [ErrorsWithData].
// exported panic-upon-failure functions should not use this.
func make_ast_successfully(s string) (ret ast_root) {
	t := tokenizeInterpolationString(s)
	ret, err := make_ast(t)
	if err != nil {
		panic(err)
	}
	err = ret.handleSyntaxConditions()
	if err != nil {
		panic(err)
	}
	return
}
