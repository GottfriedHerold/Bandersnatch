package errorsWithData

import (
	"fmt"
	"strconv"
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
//   - ast_parentPercentMulti: %w{. No children (uint stored directly, with -1 having special meaning)
//   - ast_parentDollarMulti:  $w{. No children (uint stored directly, with -1 having special meaning)
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
// NOTE: In retrospect, the code would probalby be easier with reduction steps (mostly due to mistake handling) -- however, I'm not gonna touch this again unless I must.

// [1]: The internal implementation of variable x of interfaces type uses a pair (type_info, STH), where STH is either a value of a pointer to it (depending on size of the type).
// If the type changes, the memory for STH is reused.  Acquiring a pointer to a value-stored STH, then changing the values of x to something of a different type would result in a pointer of
// of type *T pointing to something of type quite different from T, leading to disaster (even if not dereferenced, the garbage collector might misbehave badly if some stray pointers are kept around in non-garbage collected memory / STH contains pointers).
// Of course, things work fine if STH is itself a pointer -- which is kind-of enforced by making *T satisfy the interface: Then the interface directly stores a value of type *T.
// [2]: With usual production rules in (non-extended) BNF, a standard approach would not lead to ast_list storing n elements, but to a (binary) right/left-leaning tree.
// We take "production rule" to allow List -> SequenceElement* rules.
// This is in fact the only reason (apart from defaulting to 'v' for fmtString, which could be inserted by a more complex tokenizer) why the language is not LL(0);

// ast_I is an interface type that is satisfied by all nodes (and in particular by the root) of the abstract syntax tree that we
// parse interpolation strings into.
//
// Note that all types implementing ast_I are prefixed with ast_.
// These types may be aliases to pointer types and it's always the ast_foo type itself (and NOT *ast_foo) that satisfies ast_I.
// Creating any instance must be performed by new_ast_<foo> functions. Modifying instance must be performed through designated methods.
// Assignment of any mutable ast_foo - type is always shallow. (i.e. y=x; y.modify(...) modifies x as well).
type ast_I interface {
	IsNode()        // Only to "mark" relevant types
	String() string // Only used for debugging and testing (some test-cases compare against an expected output of String())

	// Interpolate returns a string (by appending to *s) by actually evaluating the subtree below that node.
	// parameters_direct, parameters_passed and baseError are used to evaluate special tokens.
	// parameters_passed == nil has the special meaning of not using this feature and behaves mostly like parameters_passed == parameters_direct.
	// This is very different from parameters_passed being an empty map. parameters_direct should not not be nil (use an empty map instead)
	//
	// Appending to *s rather than returning a string is purely done for efficiency reasons; this reduces memory allocations.
	Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder)

	// handleSyntaxConditions handles the following syntactic conditions on nodes:
	// - literal % in formatVerbs
	// - invalid variable names
	// - unrecognized conditions
	//
	// This checks for the presence of these mistakes in the subtree of the given node and returns the first mistake.
	// This methods also actually *modifies* the tree to handle the mistake. If called on the root, it memoizes the mistake.
	// The latter is done because the modifications would interfere with retrieving the mistake.
	// We assume that this method is called on the root node after [make_ast].
	//
	// NOTE: We could handle these mistake during [make_ast], but it feels cleaner to separate that (as [make_ast] is already too complicated) and it makes testing easier.
	//
	// NOTE: Since this method changes the ast, this function *must* be called before we return anything to the package user in order to prevent potential data races.
	// The fact that this method is automatically called on demand is a leftover from before the author realized this issue.
	// Refactoring to just unconditionally call it as part of [make_ast] would require refactoring tests.
	handleSyntaxConditions() (err Mistake)

	// VerifyParameters_direct report syntax or interpolation mistakes from the subtree below that node.
	// Note that we may cut corners here and only require this description to be accurate for the root (we assume that all calls to a non-root node must be a result from recursive calls).
	// parameters_direct and baseError are used for the interpolation. We assume parameters_direct to be non-nil.
	VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err Mistake)

	// VerifyParameters_passed report syntax or interpolation mistakes from the subtree below that node.
	// Again, note that we may cut corners here and only require this description to be accurate for the root (we assume that all calls to a non-root node must be a result from recursive calls).
	// parameters_direct, parameters_passed and baseError are used for the interpolation. We assume parameters_direct to be non-nil.
	// parameters_passed == nil has the special meaning of not using this feature (and behaves like parameters_passed == parameters_direct)
	// This is very different from parameters_passed being an empty map.
	VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err Mistake)
}

// interfaces satisfied by a subset of the AST types. This is used to consolidate both the parsing code and testing.

type (
	// ast_with_children is satisfied by AST types that have other ASTs as children, namely [ast_root] and both types satisfying [ast_cond]
	// Note that "children" (plural) is a bit of a misnomer, implementation-wise: each such AST actually only refers to a single other AST
	// which is often of type [ast_list].
	ast_with_children interface {
		ast_I                    // is an AST
		set_child_list(ast_list) // sets the child. We always take an [ast_list] here to simplify matters.
		get_children() ast_I     // gets the child ast. This is likely of type [ast_list].
		simplify()               // asserts the type of child has type [ast_list] and, if that list has exactly 1 element, replaces it by this single element.
	}
	// ast_fmt is satisfied by AST types [ast_fmtPercent] and [ast_fmtDollar], i.e. formatting strings.
	ast_fmt interface {
		ast_I                         // is an AST
		set_variableName(stringToken) // setter for variable name
		get_variableName() string     // getter for variable name
		set_formatString(stringToken) // setter for format string. We take a stringToken rather than a string here to simplify the code
		get_formatString() string     // getter for format string. We return a string as this is what's needed in the code.
		token() string                // outputs either `%` or `$`
	}
	// ast_cond is satisfied by AST types [ast_condPercent] and [ast_condDollar]. i.e. conditional evaluation.
	ast_cond interface {
		ast_I                      // is an AST
		set_condition(stringToken) // sets the condition string. Taking a stringToken rather than string is for convenience.
		get_condition() string     // gets the condition string. Returning a string rather than stringToken is for convenience.
		token() string             // outputs either `%!` or `$!`
		make_invalid(flags uint)   // flags the AST as invalid. This is called if we detect an mistake to improve diagnostic message.
		is_valid() bool            // checks whether make_invalid has been called on the AST.
		simplify()                 // see [ast_with_children] (Note: we could just embedd [ast_with_children], actually)
		set_child_list(ast_list)   // see [ast_with_children] (Note: we could just embedd [ast_with_children], actually)
		// Note: get_chilren is missing here, solely because we don't need it.
	}
	// ast_parentMulti is satisfied by AST types [ast_parentPercentMulti] and [ast_parentDollarMulti], i.e. %w{...} and $w{...} - related ASTs
	ast_parentMulti interface {
		ast_I                               // is an AST
		set_childIndex(stringToken) Mistake // setter for child index argument. No getter needed.
		get_childIndex() int                // getter for child index. Only used in testing.
		token() string                      // outputs either `%w{` or $w{`
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
		// parseMistake is non - nil if there was a parse mistake when this tree was constructed.
		// This is needed to make any Verify - function fail early.
		// It causes Interpolate to unconditionally output all the base error and all parameters
		// parseMistake takes precendence over argumentMistake
		parseMistake Mistake
		// argumentMistake is non-nil if there was a syntax mistake with the argument of a token.
		// It is set by calling [handleSyntaxConditions] on the root, which needs to be done after [make_ast]
		// Notably, it records if one of the following has occurred:
		// a fmtVerb contains a %
		// a condition string was not recognized
		// a variable name was invalid
		// Either of these causes Interpolate to unconditionally output all parameters.
		argumentMistake Mistake

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

// append_ast appends a new node to the list of children.
func (al ast_list) append_ast(a ast_I) {
	*al = append(*al, a)
}

// remove_last removes that last added child node from the list.
//
// This method asserts that the list has lenght >0. It is only used during rollback on certain parse mistakes.
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
	formatString  string
	variableName  string
	mistakeString Mistake // set by [handleSyntaxConditions] during post-processing if a mistake is detected. If non-nil, causes Interpolate to actually report an in-band diagnostic message.
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
// This is provided to satisfy [ast_fmt] and unify cases in mistake reporting.
func (a ast_fmtPercent) token() string {
	return `%`
}

// token returns a literal '$' for ast_fmtDollar.
//
// This is provided to satisfy [ast_fmt] and unify cases in mistake reporting.
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
 * ast_parentPercentMulti
 * ast_parentDollarMulti
 */

type (
	base_ast_parentMult struct{ whichChild int } // Note: whichChild is guaranteed to be !=0.
	// Not using uint because values that don't fit into an int wouldn't work anyway. Also allows to use -1 as special value for #
	v_ast_parentPercentMult struct{ base_ast_parentMult }
	v_ast_parentDollarMult  struct{ base_ast_parentMult }
	ast_parentPercentMulti  = *v_ast_parentPercentMult
	ast_parentDollarMulti   = *v_ast_parentDollarMult
)

func new_ast_parentPercentMult() ast_parentPercentMulti {
	return new(v_ast_parentPercentMult)
}

func new_ast_parentDollarMult() ast_parentDollarMulti {
	return new(v_ast_parentDollarMult)
}

// set_childIndex sets the actual child index from s.
// For this, s is parsed as either a literal "#" or a positive int using [strconv]'s [Parseint]
// Returns a non-nil err on failure; in this case, the child index is set to 0 (which is an invalid value in this context)
func (a *base_ast_parentMult) set_childIndex(s stringToken) (err Mistake) {
	sString := string(s)
	if sString == outputChildNumber { // outputChildNumber == "#"
		a.whichChild = -1
		return
	}

	// Note: sString == "" would cause ParseInt to fail rather than output result==0.
	// This is the correct behaviour for us (although this cannot happen due to how the AST parser works: stringTokens are never empty)

	var result int64                                       // needed beause ParseInt returns an int64 rather than an int.
	result, errParseInt := strconv.ParseInt(sString, 0, 0) // 0,0 means "May use sign and prefix (such as 0x)", "Fit into int"
	if errParseInt != nil {
		err = errParseInt
		a.whichChild = 0
	}
	if result <= 0 {
		if err == nil {
			err = fmt.Errorf("invalid index:%v", sString)
		}
		a.whichChild = 0
		return
	}
	if err == nil {
		a.whichChild = int(result)
	}
	return
}

// get_childIndex returns the actual child index. This index is 1-based.
// For uninitialized asts or after set_childIndex failed, this returns 0, which signals "invalid".
// The special value -1 is returned if we parsed a literal "#" (indicating we want to output the number of children).
//
// NOTE: We do not use this function outside of testing.
func (a *base_ast_parentMult) get_childIndex() int {
	return a.whichChild
}

// token returns a literal '%w{' for [ast_parentPercentMult].
//
// This is provided to satisfy [ast_parentMult] and unify cases in mistake reporting.
func (ast_parentPercentMulti) token() string { return `%w{` }

// token returns a literal '$w{' for [ast_parentDollarMult].
//
// This is provided to satisfy [ast_parentMult] and unify cases in mistake reporting.
func (ast_parentDollarMulti) token() string { return `$w{` }

/*
 * ast_condPercent
 * ast_condDollar
 */

// potential values for [base_ast_condition.invalidParse] these may be bitwise-OR-ed.
//
// These affect mistake reporting as follows:
const (
	astConditionValidity_VALID            = 0
	astConditionValidity_OUTPUT_CHILD     = 1 // Interpolate outputs children unconditionally, ignoring the condition.
	astConditionValidity_OUTPUT_CONDITION = 2 // Interpolate outputs the condition string (typically followed by the children, if the above was set as well)
)

// potential values for [base_ast_condition.conditionType]
const (
	conditionType_Invalid          int = iota // invalid condition
	conditionType_EmptyMap                    // m == 0 condition
	conditionType_NonEmptyMap                 // m != 0 condition
	conditionType_ParameterZero               // Variable == 0 conditon
	conditionType_ParameterNonZero            // Variable != 0 condition
	conditionType_ParameterPresent            // Variable present
	conditionType_ParameterMissing            // Variable not present
)

// base_ast_condition is a helper type for joint functionality of [ast_condPercent] and [ast_condDollar] (via struct embedding)
type base_ast_condition struct {
	condition string // condition string that controls under what condition child is interpolated.
	child     ast_I  // child node. During construction of the tree, we always initialize this with a [ast_list]-node that may later be replaced by a non-list node.
	// invalidParse is set if there was a mistake when creating this node and the mistake happened after condition was read.
	// Additionally, this flag is set if there was a parse mistake in the child subtree.
	// This flag then signals whether we should output the condition string and child unconditionally.
	// The purpose of this behaviour is to give better output diagnostics in case of mistakes; in particular,
	// we need to ensure that mistakes are not hidden by a condition that would not output the child.
	//
	// Note that not all mistakes need to set this flag.
	// values are defined by astConditionValidity_<FOO> constants.
	invalidParse uint

	// Note: we store invalidParse rather than validParse, because this way, the zero value makes newly generated instances valid.

	// These are set by post-processing via handleSyntaxCondition
	conditionType int    // type of condition (one of the conditionType_Foo constants)
	variableName  string // for some condition types that refer to a parameter name
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
// This is provided to satisfy the [invalidatable] interface. It is called when certain mistakes are caught during parsing (in particular mistakes in the child-subtree).
// These flags are read by [Interpolate] and cause special treatment of output; in particular, we may unconditionally interpolate the child subtree in order
// to ensure that the cause of parsing mistakes is actually displayed.
func (a *base_ast_condition) make_invalid(flags uint) {
	a.invalidParse |= flags
	a.conditionType = conditionType_Invalid
}

// is_valid returns whether the node of type [ast_condPercent] or [ast_condDollar] is valid
//
// This returns true unless [make_invalid] has been called on the node with a non-zero flag, which happens on certain parse mistakes.
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
// This is provided to satisfy [initialTokenGetter] and unify cases in mistake reporting.
func (a ast_condPercent) token() string {
	return `%!`
}

// token returns a literal '$!' for [ast_condDollar].
//
// This is provided to satisfy [initialTokenGetter] and unify cases in mistake reporting.
func (a ast_condDollar) token() string {
	return `$!`
}

// All ast_foo - types have an IsNode() method to signal they are intented to satisfy ast_I.
// This is mostly to ensure that other types (such as helper types like base_ast_condition) don't accidentially satisfy ast_I.
// This helps to prevent errors when writing code.

func (ast_root) IsNode()               {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_list) IsNode()               {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_string) IsNode()             {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_fmtPercent) IsNode()         {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_fmtDollar) IsNode()          {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_parentPercent) IsNode()      {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_parentDollar) IsNode()       {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_condPercent) IsNode()        {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_condDollar) IsNode()         {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_parentDollarMulti) IsNode()  {} // IsNode is a dummy method provided to satisfy [ast_I]
func (ast_parentPercentMulti) IsNode() {} // IsNode is a dummy method provided to satisfy [ast_I]

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
	if a.mistakeString != nil {
		return a.mistakeString.Error()
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
	if a.mistakeString != nil {
		return a.mistakeString.Error()
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
func (a ast_parentPercentMulti) String() string {
	if a.whichChild == -1 {
		return `%w{#}`
	}
	return fmt.Sprintf("%%w{%v}", a.whichChild)
}

// String is required for the [ast_I] interface.
//
// It is only used for debugging and testing.
func (a ast_parentDollarMulti) String() string {
	if a.whichChild == -1 {
		return `$w{#}`
	}
	return fmt.Sprintf("$w{%v}", a.whichChild)
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
	parseMode_Sequence        parseMode = iota // currently reading a sequence of list elements
	parseMode_FmtString                        // expecting a format string (after % or $) or { for default format string
	parseMode_Condition                        // expecting a condition (after %! or $!)
	parseMode_VariableName                     // expecting a variable name
	parseMode_ChildIndex                       // expecting a string encoding a number for the child index (or literal "#")
	parseMode_OpenSequence                     // expecting a { to start a sequence (after %!COND or $!COND)
	parseMode_OpenVariable                     // expecting a { to be followed by a variable name (after %fmtString or $fmtString)
	parseMode_CloseVariable                    // expecting a } terminating a variable name
	parseMode_CloseChildIndex                  // expecting a } terminating a child index
	parseMode_Mistake                          // set after the first mistake

	// NOTE: There is no parseMode_CloseSequence: The terminating '}' in %!COND{...} and $!COND{...} is handled by parseMode_Sequence
)

// embeddedParseMistake is used to create error strings for the purpose of embedding them into the tree.
// the string s may contain formatting verbs understood by [fmt] and args are passed to some fmt formatting function such as [fmt.Sprintf].
//
// The intended usage is to call this function and place it in the tree as a node (for this reason, we return an [ast_string] for convenience).
// By doing that, the string returned from embeddedParseMistake is displayed whenever the tree is interpolated (i.e. whenever we call Error on the errors returned by the package).
// This is done for diagnostics.
//
// The actual reason to use this function (over plain [fmt.Sprintf]) is that we may add some extra diagnostic string to designate parsing mistakes.
// Using this function unifies the extra diagnostic string.
func embeddedParseMistake(s string, args ...any) ast_string {
	return new_ast_string(stringToken(fmt.Sprintf(`<!PARSE-ERROR: `+s+`>`, args...))) // should be write PARSE-MISTAKE here?
}

// make_ast creates a syntax tree out of the list of tokens.
//
// IMPORTANT: Any ast_root (contained in a struct) that is returned by an exported function *must* have
// been post-processed by [handleSyntaxConditions]. While this post-processing is triggered by anything that requires it,
// it modifies the ast on its first call without any kind of locking; consequently, forgetting this yields a potential thread-safety issue.
//
// On failure, reports the first mistake. Note that we do NOT stop parsing on such mistakes;
// we rather process the input to the end and build a meaningful syntax tree.
// The returned syntax tree will contain a diagnostic message (as a valid node of string type).
// Any tokens read after the first mistake are turned into top-level inactive string tokens (in particular, there will be no more mistakes). Interpolate will output them unevaluated.
//
// If there is a parse mistake, the returned err is additionally stored in the (root node of the) returned ret.
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
// Failures of these assumptions may cause a panic or possibly weird misparses rather than report mistakes.
//
// Also note that make_ast only constructs the tree. It does not care whether the tokens "make sense".
// In particular, formatVerbs can contain extra "%", Variable names could be unexported and not even valid Go identifiers, conditions not recognized etc.
// These (optional) checks come later.
func make_ast(tokens tokenList) (ret ast_root, err Mistake) {

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
	// Mistake handling:
	// There is actually not terribly many mistake cases involved here and we handle them as follows:
	//  - We Pop the stack up until we are at the top-level list and enter a special parseMode_Mistake state
	//    In this state, everything further read will just be treated as an inactive string token to be appended to this top-level list
	//    Any ast_condPercent or ast_condDollar popped this way will be marked as tainted.
	//  - We append a string token that describes the mistake reason as an in-band report
	//  - We also report the mistake in the root node an function's return value.
	//  - If a mistake occurs while we are in the process of creating an ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar (i.e. while we step though parseMode states, but read an unexpected token):
	//    We actually already created the ast-node upon reading the introducing %, $, %! or $!. So in this case, we roll back and remove that node (replacing it by a string node for mistake reporting).
	//    This may serve as the in-band mistake report (Note that it might not in the top-level list)
	//  - In all other cases, the only thing that can go wrong at the [make_ast] state are tokens of unexpected type, handled as above or
	//    stack errors, meaning that we encounter } when there is no stack to pop or finish reading without popping the stack.
	//  - We use the [set_parseMistake] local function to handle the stack popping, tainting ast_cond nodes and returning the mistake in the root and returned value err.
	//    The in-band error string is processed by [embeddedParseMistake]. The actual error string is handled slightly differently by these two:
	//    [set_parseMistake] prefixes the error with [ErrorPrefix], whereas [embeddedParseMistake] adds some <!META-ERROR...> tag to make the error stand out.
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
		panic(ErrorPrefix + "internal error: empty or nil token list (missing start/end markers)") // unreachable for tokens output by tokenizeInterpolationString.
	}
	if tokens[0] != tokenStart {
		panic(ErrorPrefix + "internal error: invalid token list (missing start marker)") // unreachable for tokens outputs of tokenizeInterpolationString
	}
	tokens = tokens[1:]

	var mode parseMode = parseMode_Sequence // we expect a list of stringTokens, %w, $w etc.

	// set_parseMistake is a closure that is called when a parse mistake is encountered.
	// s is a format string and args are its arguments, used to create the returned diagnostic message
	//
	// Before or after calling this closure, the parser should embed a diagnostic message as a ast_string into the returned ast.
	// Usually, this diagnostic message resembles s.
	//
	// This closure assumes the stack is in a good state as in parseMode_Sequence.
	// Notably, it contains (starting from bottom): ROOT - LIST, followed by any number of pairs COND - LIST.
	// The conditions in any COND-node on the stack are non-empty strings.
	//
	// We set the mistake returned by make_ast (both in the ast_root and err), terminate all open ast_cond's
	// (so the resulting stack after the call to set_parseMistake is ROOT - LIST) and flag them as invalid.
	// We then set mode to parseMode_Mistake.
	//
	// The parser will then remain in parseMode_Mistake, where every input token just gets turned into a string (which can produce no more mistakes)
	// flagging the ast_cond - path as invalid will make Interpolate ignore the condition.
	// This causes the offending part that caused the parse mistake to be unconditionally displayed.
	set_parseMistake := func(s string, args ...any) {
		// record first found mistake both in value returned from function and in the returned root node.
		// The latter is done to make sure Validation function can reproduce the mistake.
		if err == nil {
			err = fmt.Errorf(ErrorPrefix+s, args...)
			ret.parseMistake = err
		} else {
			// err is only set by set_parseMistake.
			// we enter parseMode_Mistake at the end of set_parseMistake. In this parseMode, we can never encounter another mistake, because
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
			condNode.make_invalid(astConditionValidity_OUTPUT_CHILD)
			condNode.simplify()
		}
		mode = parseMode_Mistake
	}

	for _, token := range tokens {
		// goto redo can be used to "re-scan" the last token.
		// This is done after certain failure cases:
		// Re-scanning the token in parseMode_Mistake will just do the right thing,
		// thereby simplifying the mistake handling.
	redo:
		// NOTE: We will overwrite top by a type-asserted top:=top.(ast_*) after we branch, since we know more about the type
		// NOTE: To avoid confusion, our convention is to stop using the top variable after any operation that changes the
		// stack shape (set_parseMistake, stack.Push, stack.Pop) until we get here again.
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
					embeddedErrorNode := embeddedParseMistake(`Unexpected "{"`)
					top.append_ast(embeddedErrorNode)
					set_parseMistake(`Unexpected "{" in format string`)

				case tokenCloseBracket:
					if stack.Len() <= 3 {
						// We always have ROOT-LIST on the stack until we read tokenEnd.
						if stack.Len() != 2 {
							panic("Cannot happen")
						}
						embeddedErrorNode := embeddedParseMistake(`Unexpected "}"`)
						top.append_ast(embeddedErrorNode)
						set_parseMistake(`unexpected "}" in format string`)
						continue // with mode == parseMode_Mistake, set by set_parseMistake
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
						set_parseMistake(`Missing "}" in format string`)
						// stack length is 2 after calling set_parseMistake

						currentNode := (*stack.Top()).(ast_list)
						embeddedErrorNode := embeddedParseMistake(`Missing "}" in format string`)
						currentNode.append_ast(embeddedErrorNode)
						goto redo // reprocess tokenEnd in parseMode_Mistake; this is just to simplify the code.
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
				case tokenParentPercentMulti: // create and add node for %w{. Child index will be parsed next
					newNode := new_ast_parentPercentMult()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_ChildIndex // read child index next
				case tokenParentDollarMulti: // create and add node for $w{. Child index will be parsed next
					newNode := new_ast_parentDollarMult()
					top.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_ChildIndex // read child index next
				default:
					panic(fmt.Errorf(ErrorPrefix+"Unhandled token: %v", token)) // cannot happen for tokenLists output by tokenizeInterpolationString.
				}
			default: // token not of type stringToken or specialToken
				panic(ErrorPrefix + "Invalid entry in token list") // cannot happen
			}

		case parseMode_FmtString: // expect to read (optional) format string (which must be a string literal)
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST, followed by any number >=0 of COND,LIST pairs, followed by an AST_FMT
			top := top.(ast_fmt)
			if token == tokenOpenBracket { // %{ or ${ is interpreted as %v{ or $v{
				// We treat an empty format string as 'v'.
				// However, we don't want to just set formatString to 'v' at this point, because this would interact with
				// handling of parse mistakes: if there is a parse mistake (such as missing "}" ) in further processing the %{...} - clause
				// we "undo" the parse and just literally output parts of the %{...} - clause that were read so far (together with an diagnostic message)
				// If we set the formatString to 'v' here, parsing "%{foo" would result in a confusing "%v{foo" appearing in the diagnostic message.
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

					// The case distinctions is just for better diagnostic messages.
					if token == tokenEnd {
						embeddedErrorNode := embeddedParseMistake(`Interpolation string ends in "%s"`, percentOrDollar)
						currentNode.append_ast(embeddedErrorNode)
						set_parseMistake(`Interpolation string ends in unescaped "%s"`, percentOrDollar)
					} else {
						embeddedErrorNode := embeddedParseMistake(`Invalid token "%s" after "%s"`, token.String(), percentOrDollar)
						currentNode.append_ast(embeddedErrorNode)
						set_parseMistake(`Invalid token "%s" after "%s"`, token.String(), percentOrDollar) // sets mode to parseMode_Mistake
					}
					goto redo // re-read actual offending token in parseMode_Mistake. This also handles token==tokenEnd correctly.

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

				// The case distinctions is just for better diagnostic messages.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseMistake(`Interpolation string ends in "%s"`, percentOrDollarExlamMark)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Interpolation string ends in unescaped "%s"`, percentOrDollarExlamMark)
				} else {
					embeddedErrorNode := embeddedParseMistake(`Invalid token "%s" after "%s"`, token.String(), percentOrDollarExlamMark)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Invalid token "%s" after "%s"`, token.String(), percentOrDollarExlamMark) // sets mode to parseMode_Mistake
				}
				goto redo // re-read tokenEnd in parseMode_Mistake

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

				// add a diagnostic node and call set_parseMistake.
				// The case distinction is just to provide better diagnostic messages, since tokenEnd.String() would not return the right string.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseMistake(`Interpolation string ends where variable name was expected`)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Interpolation string ends where variable name was expected`)
				} else {
					embeddedErrorNode := embeddedParseMistake(`Got "%v" where variable name was expected`, token.String())
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Got "%v" where variable name was expected`, token.String())
				}
				goto redo // re-read offending token in parseMode_Mistake

			} else {
				// good case: token is string token. It is non-empty by assumpition on token_list
				top.set_variableName(token_string)
				// stack remains unchanged.
				mode = parseMode_CloseVariable // expect to read } next
			}
		case parseMode_ChildIndex: // expect to read an encoding of a uint or literal "#"
			// The stack is (from top to bottom) ast_parentMult - {ast_list - ast_cond -}* ast_list - ast_root
			top := top.(ast_parentMulti) // ast_parentDollarMulti or ast_parentPercentMulti
			token_string, ok := token.(stringToken)
			if !ok {
				// completely remove the currently read token and replace it by a literal %w{ or $w{
				percentOrDollarBracket := top.token()

				// remove %w{ or $w{ - node
				_ = stack.Pop()
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarBracket))) // add %w{ or $w{ as literal string

				// Add diagnostic node and call set_parseMistake
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseMistake(`Interpolation string ends where child index or "#" was expected`)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Interpolation string ends where child index or "#" was expected`)
				} else {
					embeddedErrorNode := embeddedParseMistake(`Got "%v" where child index or "#" was expected`, token.String())
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Got "%v" where child index or "#" was expected`, token.String())
				}
				goto redo // re-read offening token in parseMode_Mistake
			} else {
				// ok == true. token is string token.
				// We defer setting parsing it and setting top until we read the }.
				// The reason is that if we parse it now, we lose the actual string (e.g. we could not distinguish "0x10" from "16")
				// This would be bad for mistake reporting
				stack.Push(new_ast_string(token_string))
				mode = parseMode_CloseChildIndex
			}

		case parseMode_CloseChildIndex: // expect to read a literal }
			// the stack is (from top to bottom) ast_string - ast parentMult - {ast_list - ast_cond -}* ast_list - ast_root
			// the top of the stack is the stray (parse-delayed) child-index string
			childIndexString_ast := stack.Pop().(ast_string) // retrieve the string (as an AST) indicating what child index should be
			top := (*stack.Top()).(ast_parentMulti)          // get the actual $w{ or %w{ node
			token := token.(specialToken)                    // Since consecutive string tokens are merged, this cannot fail

			// handle potential mistakes first if we did not read the expected '}'
			if token != tokenCloseBracket {
				// We need to insert an diagnostic string and a %w{ or $w{ together with what we read as supposed childIndex.

				percentOrDollarBracket := top.token() // "%w{" or "$w{"

				// remove $w{ or $w{ node
				_ = stack.Pop()
				currentNode := (*stack.Top()).(ast_list)
				currentNode.remove_last()

				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarBracket))) // replay the %w{ or $w{
				currentNode.append_ast(childIndexString_ast)                                // replay the child index
				// case distinction to improve diagnostic messages.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseMistake(`unexpected end of format string`)
					currentNode.append_ast(embeddedErrorNode)
				} else {
					embeddedErrorNode := embeddedParseMistake(`child index not terminated by "}"`)
					currentNode.append_ast(embeddedErrorNode)
				}
				set_parseMistake(`Child index not terminated by "}"`)
				goto redo // to actually handle the tokenEnd token as ending the parse.

			} else { // token == tokenCloseBracket
				intParseError := top.set_childIndex(stringToken(childIndexString_ast))
				if intParseError != nil { // parsing as int or "#" has failed
					percentOrDollarBracket := top.token() // "%w{" or "$w{"

					// remove $w{ or $w{ node
					_ = stack.Pop()
					currentNode := (*stack.Top()).(ast_list)
					currentNode.remove_last()
					currentNode.append_ast(new_ast_string(stringToken(percentOrDollarBracket))) // replay the %w{ or $w{
					currentNode.append_ast(childIndexString_ast)                                // replay the child index
					embeddedErrorNode := embeddedParseMistake(`could not parse child index:%v`, intParseError)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`could not parse child index: %v`, intParseError)
					goto redo // to re-read the "}"
				}
				// If we get here, everything worked out OK:
				_ = stack.Pop() // remove the $w{ or %w{ node
				mode = parseMode_Sequence
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
				// For that reason, we place the embedded diagnostic message just after the %! or $! rather than at the place where we expect the {
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarExclam)))
				embeddedErrorNode := embeddedParseMistake(`"%v" has no matching "{"`, percentOrDollarExclam)
				currentNode.append_ast(embeddedErrorNode)
				currentNode.append_ast(new_ast_string(stringToken(condition)))
				set_parseMistake(`Missing "{" after %v%v`, percentOrDollarExclam, condition)
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

			// top := top.(ast_fmt) // commented out, because it's only needed in case of mistake

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
				// For that reason, we place the embedded diagnostic message just after the % or $ rather than at the place where we expect the {

				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar)))

				embeddedErrorNode := embeddedParseMistake(`unescaped "%v" has no matching "{"`, percentOrDollar)
				currentNode.append_ast(embeddedErrorNode)
				currentNode.append_ast(new_ast_string(stringToken(formatString)))
				set_parseMistake(`Missing "{" after %v%v`, percentOrDollar, formatString)
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
			// However, this needs to be done after handling parse mistakes: on mistake, we actually output the formatString and outputting a "v" would be confusing.

			// We now handle the parse mistake case if what we read actually was not a }
			// We previously read a string, so token is guaranteed to be a specialToken (no consecutive string tokens above).
			token := token.(specialToken)
			if token != tokenCloseBracket {
				// We need to insert an diagnostic string and a literal interpretation of %FmtString{VariableName

				percentOrDollar := top.token()         // "%" or "$"
				VariableName := top.get_variableName() // variableName
				_ = stack.Pop()

				currentNode := (*stack.Top()).(ast_list)

				currentNode.remove_last()                                                                                // remove the ast_fmtPercent or ast_fmtDollar
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar + formatString + "{" + VariableName))) // replay what was read so far as a plain string
				// case distinction to improve diagnostic messages.
				if token == tokenEnd {
					embeddedErrorNode := embeddedParseMistake(`unexpected end of format string after reading a variable name without closing "}"`)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Variable name not terminated by "}"`)
					goto redo // to actually handle the tokenEnd token as ending the parse.

				} else {
					embeddedErrorNode := embeddedParseMistake(`Variable name not terminated by "}"`)
					currentNode.append_ast(embeddedErrorNode)
					set_parseMistake(`Variable name not terminated by "}"`)
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

		case parseMode_Mistake:
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST.
			currentNode := top.(ast_list) // top is an ast_list node if we are in parseMode_Mistake
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
				} else { // token == tokenEnd
					// Alternatively, we could do ret.simplify() to avoid the type assertion
					// The two stack.Pop()'s are just to maintain the invariant that the stack is empty after we finish.
					_ = stack.Pop() // type popped is ast_list.
					root := stack.Pop().(ast_root)
					root.simplify()
				}
			}

		default:
			panic(ErrorPrefix + "Unhandled status in syntax tree parser") // cannot happen
		}
	}

	// Double-check that the algorithm above terminated in the expected state.

	// The only way to exit the above is reading tokenEnd in parseMode_Sequence or parseMode_Mistake.
	// (Reading a tokenEnd in other modes causes a switch to parseMode_Mistake and re-scanning it)
	if (mode != parseMode_Sequence) && (mode != parseMode_Mistake) {
		panic(ErrorPrefix + "Cannot happen")
	}

	// NOTE: We could remove this and remove the 2 stack.Pop() calls when handling tokenEnd
	// and use ret.simplify() rather then calling simplify on the returned value from Pop()
	if stack.Len() != 0 {
		panic(ErrorPrefix + "Cannot happen")
	}

	// Parse mistakes are reported both inside the returned ast as well as via the returned err.
	if err != ret.parseMistake {
		panic(ErrorPrefix + "Cannot happen")
	}

	// err is set iff we end up in parseMode_Mistake mode.
	if (mode == parseMode_Mistake) != (err != nil) {
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
