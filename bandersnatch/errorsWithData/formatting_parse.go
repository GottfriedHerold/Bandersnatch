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
//   - [Optional] Perform some validity checks. (3 subchecks, actually. Those would be checked when actually producing output anyway, but sometime we want those checks early)
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
// Consequently, creating nodes needs to be done by new_ast_foo functions (there needs to be some kind of indirection, so zero values will be invalid nodes)

// [1]: The internal implementation of variable x of interfaces type uses a pair (type_info, STH), where STH is either a value of a pointer to it (depending on size of the type).
// If the type changes, the memory for STH is reused.  Acquiring a pointer to a value-stored STH, then changing the values of x to something of a different type would result in a pointer of
// of type *T pointing to something of type quite different from T, leading to disaster (even if not dereferenced, the garbage collector might misbehave badly if some stray pointers are kept around in non-garbage collected memory / STH contains pointers).
// Of course, things work fine if STH is itself a pointer -- which is kind-of enforced by making *T satisfy the interface: Then the interface directly stores a value of type *T.
// [2]: With usual production rules in BNF, a standard approach would not lead to ast_list storing n elements, but to a (binary) right/left-leaning tree.
// We take "production rule" to allow List -> SequenceElement* rules.
// This is in fact the only reason (apart from defaulting to 'v' for fmtString, which could be inserted by the tokenizer) why the language is not LL(0);

type ast_I interface {
	IsNode()        // Only to "mark" relevant types
	String() string // only for debugging
	VerifySyntax() (err error)
	VerifyParameters_direct(parameters_direct ParamMap, baseError error) (err error)
	VerifyParameters_passed(parameters_direct ParamMap, parameters_passed ParamMap, baseError error) (err error)
	Interpolate(parameters_direct ParamMap, parameters_passed ParamMap, baseError error, s *strings.Builder)
}

// We add interfaces for extra functionality that is shared by multiple node types:
// This allows to cut down the state space that we need to (explicitly) track in our DFA.
type (
	childSetter        interface{ set_child(ast_I) }              // ast_root, ast_condPercent, ast_condDollar
	variableNameSetter interface{ set_variableName(stringToken) } // ast_fmtPercent, ast_fmtDollar
	fmtStringSetter    interface{ set_formatString(stringToken) } // ast_fmtPercent, ast_fmtDollar
	conditionSetter    interface{ set_condition(stringToken) }    // ast_condPercent, ast_condDollar
	simplifier         interface{ simplify() }                    // ast_root, ast_condPercent, ast_condDollar
	invalidatable      interface{ make_invalid() }                // ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar
	initialTokenGetter interface{ token() string }                // ast_fmtPercent, ast_fmtDollar, ast_condPercent, ast_condDollar [Only used for error reporting]
	conditionGetter    interface{ get_condition() string }
	variableNameGetter interface{ get_variableName() string }
	fmtStringGetter    interface{ get_formatString() string }
)

// *****
// DEFINITIONS OF THE INVIDUAL IMPLEMENTATIONS OF NODE TYPES:
// *****

// root node
type (
	v_ast_root struct {
		// actual "child" ast.
		ast ast_I
		// parseError is non - nil if there was an error when this tree was constructed.
		// This is needed to make any Verify - function fail
		parseError error
	}
	ast_root = *v_ast_root
)

// simplify assumes the child is of type ast_list. If len(child) == 1, replace the list by child[0].
func (a ast_root) simplify() {
	(*a).ast = (*a).ast.(ast_list).squash_list()
}

// new_ast_root creates a new node of type root. Its child node is nil, so you need to call [set_child] afterwards.
func new_ast_root() ast_I {
	return &v_ast_root{}
}

// set_child is provided to satisfy childSetter
func (a ast_root) set_child(child ast_I) {
	(*a).ast = child
}

// ast_list is node type for lists of ast's. Can only occur as child of ast_root, ast_condPercent or ast_condDollar.
type (
	v_ast_list []ast_I
	ast_list   = *v_ast_list
)

// new_ast_list creates a new node of list type. The newly created node is a valid (empty) list
func new_ast_list() ast_I {
	v := make(v_ast_list, 0)
	return &v
}

// append_ast appends a new child node to the list
func (al ast_list) append_ast(a ast_I) {
	*al = append(*al, a)
}

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

// ast_string is the node for string literals. Note that string literals means string literals to be output as such.
// Names of Variables / formatStrings etc. are NOT stored with this node type. Those are stored directly as entries of their ast_fmt nodes and not as tree nodes at all.
type ast_string string

// new_ast_string creates a leaf node with the given string literal.
func new_ast_string(s stringToken) ast_I {
	return ast_string(s)
}

// base_ast_fmt is a helper type for joint functionality of ast_fmtPercent and ast_fmtDollar
type base_ast_fmt struct {
	formatString string
	variableName string
	// invalidParse is set to true if there was a parse error when creating this node and the error happened after formatString was read.
	// This flag then signals that we should output the format string and the (likely empty) condition literally.
	// Note that not all parse errors set this flag.
	invalidParse bool
}

// ast_fmtPercent and ast_fmtDollar are nodes for %fmtString{VariableName} and $fmtString{VariableName} expressions.
// These are leaves. The fmtString and VariableName entries are stored directly, not in child-nodes.
type (
	v_ast_fmtPercent struct{ base_ast_fmt }
	v_ast_fmtDollar  struct{ base_ast_fmt }
	ast_fmtPercent   = *v_ast_fmtPercent
	ast_fmtDollar    = *v_ast_fmtDollar
)

// set_formatString sets the format string of the ast_fmtPercent or ast_fmtDollar
func (a *base_ast_fmt) set_formatString(formatString stringToken) {
	a.formatString = string(formatString)
}

// set_variableName sets the variableName of the ast_fmtPercent or ast_fmtDollar.
// Note that there is no validation in this function regarding potential validity of the given string as an valid variableName (being exported, not containing "." etc).
func (a *base_ast_fmt) set_variableName(variableName stringToken) {
	a.variableName = string(variableName)
}

func (a *base_ast_fmt) get_variableName() string {
	return a.variableName
}

func (a *base_ast_fmt) get_formatString() string {
	return a.formatString
}

func (a *base_ast_fmt) make_invalid() {
	a.invalidParse = true
}

// new_ast_fmtPercent creates a new node of type ast_fmtPercent. Its formatString and variableName have yet to be set.
func new_ast_fmtPercent() ast_I {
	return &v_ast_fmtPercent{}
}

// new_ast_fmtPercent creates a new node of type ast_fmtDollar. Its formatString and variableName have yet to be set.
func new_ast_fmtDollar() ast_I {
	return &v_ast_fmtDollar{}
}

func (a ast_fmtPercent) token() string {
	return `%`
}

func (a ast_fmtDollar) token() string {
	return `$`
}

type (
	ast_parentPercent struct{} // ast_parentPercent is the leaf node for %w - entries
	ast_parentDollar  struct{} // ast_parentDollar is the leaf node for $w - entries
)

// new_ast_parentPercent creates a new node of type ast_parentPercent. This is ready to use.
// Note that the parsing step does not know about the actual parent error, so there is no validity check (is there a non-nil parent error?) here.
func new_ast_parentPercent() ast_I {
	return ast_parentPercent{}
}

// new_ast_parentDollar creates a new node of type ast_parentDollar. This is ready to use.
// Note that the parsing step does not know about the actual parent error, so there is no validity check (is there a non-nil parent error that supports this?) here.
func new_ast_parentDollar() ast_I {
	return ast_parentDollar{}
}

// base_ast_condition is a helper type for joint functionality of ast_condPercent and ast_condDollar (via struct embedding)
type base_ast_condition struct {
	condition string
	child     ast_I
	// invalidParse is set to true if there was a parse error when creating this node and the error happened after formatString was read.
	// This flag then signals that we should output the condition string literally and child unconditionally.
	// Note that not all parse errors set this flag.
	invalidParse bool
}

// set_condition sets the condition string for an ast_condPercent or ast_condDollar.
// Note that this functions does not perform any validity checks on cond.
func (a *base_ast_condition) set_condition(cond stringToken) {
	a.condition = string(cond)
}

func (a *base_ast_condition) get_condition() string {
	return a.condition
}

// make_invalid sets the node to invalid
// This is caught by interpolate and causes special treatment of output.
func (a *base_ast_condition) make_invalid() {
	a.invalidParse = true
}

// set_child sets the child node (essentially always an ast_list that is later simplify()ed) for the ast_condPercent or ast_condDollar.
func (a *base_ast_condition) set_child(child ast_I) {
	a.child = child
}

// simplify is provided to satisfy the simplifier interface and is provided for ast_condPercent and ast_condDollar.
// This assumes the child node is a list l and replaces it by it sole entry if len(l) == 1.
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

// new_ast_condPercent creates a new node of type ast_condPercent. It condition is the empty string and the (parsed) SubInterpolationString has yet to be set by set_child
func new_ast_condPercent() ast_I {
	return &v_ast_condPercent{}
}

// new_ast_condPercent creates a new node of type ast_condPercent. It condition is the empty string and the (parsed) SubInterpolationString has yet to be set by set_child
func new_ast_condDollar() ast_I {
	return &v_ast_condDollar{}
}

func (a ast_condPercent) token() string {
	return `%!`
}

func (a ast_condDollar) token() string {
	return `$!`
}

// All ast_foo - types have an IsNode() method to signal they are intented to satisfy ast_I.
// This is mostly to ensure that other types (such as helper types like base_ast_condition) don't accidentially satisfy ast_I.

func (a ast_root) IsNode()          {}
func (a ast_list) IsNode()          {}
func (a ast_string) IsNode()        {}
func (a ast_fmtPercent) IsNode()    {}
func (a ast_fmtDollar) IsNode()     {}
func (a ast_parentPercent) IsNode() {}
func (a ast_parentDollar) IsNode()  {}
func (a ast_condPercent) IsNode()   {}
func (a ast_condDollar) IsNode()    {}

// We define String methods on each AST node type. These are exclusively used for debugging.

func (a ast_root) String() string {
	return "AST(" + (a.ast).String() + ")"
}

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

func (a ast_string) String() string {
	return `"` + string(a) + `"`
}

func (a ast_fmtPercent) String() string {
	var b strings.Builder
	b.WriteRune('%')
	b.WriteString((*a).formatString)
	b.WriteRune('{')
	b.WriteString((*a).variableName)
	b.WriteRune('}')
	return b.String()
}

func (a ast_fmtDollar) String() string {
	var b strings.Builder
	b.WriteRune('$')
	b.WriteString((*a).formatString)
	b.WriteRune('{')
	b.WriteString((*a).variableName)
	b.WriteRune('}')
	return b.String()
}

func (a ast_parentPercent) String() string {
	return "%w"
}

func (a ast_parentDollar) String() string {
	return "$w"
}

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

// parseMode is an enum type for the state of our AST parser. Our parser is essentially a DFA with access to a stack.
// Note that we require only very few states, allowing to write this by hand; this is mostly due to the fact that the language is so simple and because we use some extra Go interfaces.
// As mentioned above, we do not use a "traditional shift/reduce parser", but rather construct the node for the parse tree as soon as we see the first symbol
// involved in the related production rule.
// The language is simple enough to allow this:
// The only ambiguity is list length and optional presence of a fmtString, but these are NOT part of the AST tree structure, since we allow lists of arbittrary length.
// The stack is only needed for nested sub-interpolation strings of type %!COND{...} or $!COND{...}.
// Rather than reducing, we then modify the just-constructed node when reading the next symbols.
// Note that these modification go through an type assertion to an interface such as childSetter that is satisfied by several nodes types.
// This means (from a theory POV) that the dynamic type of the last-constructed node is actually part of the parser state and the interface dispatch implicitly part of the DFA's state transition function,
// but the algorithm does not need to make a (explicit) case distinction, which allows a small and managable state space.
type parseMode int

const (
	parseMode_Sequence      parseMode = iota // currently reading a sequence of list elements
	parseMode_FmtString                      // expecting a format string (after % or $) or { for default format string
	parseMode_Condition                      // expecting a condition (after %)
	parseMode_VariableName                   // expecting a variable name
	parseMode_OpenSequence                   // expecting a { to start a sequence (after %!COND or $!COND)
	parseMode_OpenVariable                   // expecting a { to be followed by a variable name (after %fmtString or $fmtString)
	parseMode_CloseVariable                  // expecting a } terminating a variable name
	// NOTE: There is no parseMode_CloseSequence: The terminating '}' in %!COND{...} and $!COND{...} is handled by parseMode_Sequence
	parseMode_Error // set after the first error
)

func embeddedParseError(s string, args ...any) string {
	return fmt.Sprintf(`<!META-ERROR: `+s+`>`, args...)
}

// (as in: this allows us to skip treating those case, which means that violations results in possibly non-informative panics rather than non-nil error output. )

// make_ast creates a syntax tree out of the list of tokens.
//
// On failure, reports the first error. Note that we do NOT stop on errors; we rather process the input to the end
// and build a valid syntax tree, replacing offending tokens by what we heuristally believe to be useful error messages.
// The returned error is then also stored in the (root node of the) returned tree.
//
// To simplify the parser, we make the following assumption about the input list of tokens:
//   - tokens[0] == tokenStart
//   - token[len(token)-1] == tokenEnd
//   - no other tokenStart or tokenEnd appear
//   - no two consecutive tokens have type stringToken
//   - stringTokens are not the empty string (We have no token at all instead).
//
// These assumptions are satisfied by the output of our tokenizer.
// Note that failure of these assumptions will cause a panic (or possibly weird misparses) rather than report errors.
//
// Also note that make_ast only constructs the tree. It does not care whether the tokens "make sense".
// In particular, formatStrings can contain %, Variable names could be unexported and not even valid Go identifiers, conditions not recognized etc.
// These (optional) checks come later.
//
// The precise error behaviour is the following (subject to change):
//
// When expecting a sequence of strings or special instructions:
//   - unexpected } and EOF get turned into string ast's telling the error.
//   - unexpected { get turned into a fake condition (to enforce a matching }) outputting an error
//
// % or $ followed by EOF or by a non-string, non-"{" - token causes the % or $ to be replaced by an error string. (The token following % or $ has its meaning as if %/$ was not there)
// %! or $! followed by EOF or by a non-string-token causes the %! or $! to be acts similarly
// %fmtString/$fmtString not followed by { marks the ast_fmt - node as invalid (fmtString set, VariableName unset), adds error string
// %!COND/$!COND acts similarly
// %fmtString{ followed by Non-string: invalidate ast_fmt - node. Add fake condition to inject an error and enforce closing }
func make_ast(tokens tokenList) (ret ast_I, err error) {

	ret = new_ast_root() // Make root node. This is directly stored in ret, because we abort on first error and this allows looking at the partially constructed tree at the call site for debugging.

	// the top of our tree (apart from the root node) is a list, starting empty.
	initial_list := new_ast_list()
	ret.(childSetter).set_child(initial_list)

	// since %!Foo{Bar} and $!Foo{Bar} can be nested, we can actually get a tree of arbitrary depth
	// We maintain a stack that contains pointers to the ast_nodes on the current path to the leaf we are working with.
	// Note that we push copies of the nodes (rather than pointers to the node) to the stack.
	// This is fine, because all ast_I - nodes have shallow semantics anyway.
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

	// set_error is a closure that is called when a parse error is found.
	// s is a format string and args are its arguments, used to create the returned error
	//
	// This closure assumes the stack is a good state as in parseMode_Sequence
	// We set the returned error (both in the ast_root and err), terminate all open ast_cond's
	// (so the resulting stack is ast_root - ast_list) and flag them as invalid.
	// We then set mode to parseMode_Error.
	//
	// Note that the parser below will then remain in parseMode_Error, where every input token just gets turned into a string (which can produce no more errors)
	// flagging the ast_cond - path as invalid will make Interpolate ignore the condition. This causes the offending part that caused the parse error to be unconditionally displayed.
	set_error := func(s string, args ...any) {
		// record first found error both in value returned from function and in the returned root node.
		// The latter is done to make sure Validation function can reproduce the error.
		if err == nil {
			err = fmt.Errorf(ErrorPrefix+s, args...)
			ret.(ast_root).parseError = err
		} else {
			// we enter parseMode_Error at the end of set_error. In this parseMode, we can never encounter another error, because
			// we just turn every token that we read from this point on into a string.
			panic("Cannot happen")
		}

		// Not needed for set_error to be meaningful, but it is the case in our usage unless we screwed up:
		if stack.Len()%2 != 0 {
			panic("Cannot happen")
		}
		if stack.Len() < 2 {
			panic("Cannot happen")
		}

		// we proceed through the stack and mark any node.
		// node.make_invalid(alwaysPrintChildren) only really affects
		// nodes of types ast_condPercent and ast_condDollar and causes
		// the condition to be ignored, so the conditional interpolation is always evaluated.
		for i := 0; i < (stack.Len()/2)-1; i++ {
			_ = stack.Pop().(ast_list)
			condNode := stack.Pop()
			condition := condNode.(conditionGetter).get_condition()
			if condition == "" {
				panic("Cannot happen")
			}
			condNode.(invalidatable).make_invalid()
			condNode.(simplifier).simplify()
		}
		mode = parseMode_Error
	}

	for _, token := range tokens {
	redo:
		var top ast_I = *stack.Top() // Peek at top of stack. NOTE: stack cannot be empty
		switch mode {
		case parseMode_Sequence: // expect to get a sequence of strings or tokens.
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST followed by any number >=0 of COND,LIST pairs.
			currentNode := top.(ast_list) // top is a ast_list if we are in parseMode_Sequence

			switch token := token.(type) {
			case stringToken: // append token for the string literal in the list and continue in parseMode_sequence
				newNode := new_ast_string(token)
				currentNode.append_ast(newNode)
			case specialToken: // read token that is not a string literal
				switch token {
				case tokenPercent: // create and add node for %fmtString{Variable}. fmtString and VariableName are set later
					newNode := new_ast_fmtPercent()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString // read (optional) format string next
				case tokenDollar: // create and add node for $fmtString{Variable}. fmtString and VariableName are set later
					newNode := new_ast_fmtDollar()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString // read (optional) format string next
				case tokenPercentCond: // create and add node for %!Condition{Sub-AST}. Condition and Sub-AST are set later.
					newNode := new_ast_condPercent()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition // read Condition string next
				case tokenDollarCond: // create and add node for $!Condition{Sub-AST}. Condition and Sub-AST are set later.
					newNode := new_ast_condDollar()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition // read Condition string next
				case tokenOpenBracket: // { without prior %, $, %! or $!
					embeddedErrorMessage := embeddedParseError(`Unexpected '{'`)
					newNode := new_ast_string(stringToken(embeddedErrorMessage))
					currentNode.append_ast(newNode)
					set_error("Unexpected '{' in format string")

				case tokenCloseBracket, tokenEnd: // terminating a list. tokenCloseBracket is for %!Cond{List} and $!Cond{List}. tokenEnd is for the root.
					// ensure that } cannot appear at the top level of nested conditions:
					// recall that stack is (starting from bottom) ROOT, LIST, followed by COND, LIST - pairs
					// } is only valid if there is at least one such cond,list - pairs, which it terminates.
					if (token == tokenCloseBracket) && (stack.Len() <= 3) {
						// We always have ROOT-LIST on the stack until we read tokenEnd.
						if stack.Len() != 2 {
							panic("Cannot happen")
						}
						embeddedErrorMessage := embeddedParseError(`Unexpected '}'`)
						newNode := new_ast_string(stringToken(embeddedErrorMessage))
						currentNode.append_ast(newNode)
						set_error("Unexpected '}' in format string")
						continue // with mode == parseMode_Invalid, set by set_error
					} // tokenEnd must only appear at the top level.
					// If we read a tokenEnd while the stack size is != 2, we have an unterminated %!COND{... somewhere
					if (token == tokenEnd) && (stack.Len() != 2) {
						set_error(`Missing '}' in format string`)
						top = *stack.Top()
						currentNode = top.(ast_list)
						embeddedErrorMessage := embeddedParseError(`Missing '}' in format string`)
						newNode := new_ast_string(stringToken(embeddedErrorMessage))
						currentNode.append_ast(newNode)
						goto redo // reprocess tokenEnd in parseMode_Error; this is just to simplify the code.
					}
					// stack.Len() >= 2 is guaranteed

					_ = stack.Pop()                    // type popped is ast_list.
					rootOrCond := stack.Pop()          // type popped is either ast_condPercent, ast_condDollar or ast_root
					rootOrCond.(simplifier).simplify() // is the child list contains 1 element, replace list by single element.

					// If token == tokenEnd, mode no longer matters. In fact, since stack.Len() == 0 now, any further iteration will panic @ ast_I = *stack.Top()
					// If token == tokenCloseBracket, we end up in the parent sequence, so parseMode_Sequence is correct
					mode = parseMode_Sequence // no-op, but added for clarity

				case tokenParentPercent: // create and add node for %w and continue with the list.
					newNode := new_ast_parentPercent()
					currentNode.append_ast(newNode)
				case tokenParentDollar: // create and add node for $w and continue with the list.
					newNode := new_ast_parentDollar()
					currentNode.append_ast(newNode)
				default:
					panic(fmt.Errorf(ErrorPrefix+"Unhandled token: %v", token)) // cannot happen for tokenLists output by the tokenizer.
				}
			default: // token not of type stringToken or specialToken
				panic(ErrorPrefix + "Invalid entry in token list")
			}

		case parseMode_FmtString: // expect to read (optional) format string (which must be a string literal)
			if token == tokenOpenBracket { // %{ or ${ is interpreted as %v{ or $v{
				// Interpolate treats an empty format string as 'v'.
				// Note: We don't want to just set formatString to 'v' at this, because this would interact with
				// handling of parse errors: if there is a parse errors (such as missing } ) in further processing the %{...} - clause
				// we "undo" the parse and just literally output parts of the %{...} - clause that were read so far (together with an error message)
				// If we set the formatString to 'v' here, parsing "%{foo" would result in a confusing "%v{foo" appearing in the error message.

				mode = parseMode_VariableName // and proceed to the variable name
			} else {
				token_string, ok := token.(stringToken) // next token, if not {, must be a literal string
				if !ok {
					// remove the already-place ast_fmtPercent/ast_fmtDollar and replace it by a literal % or $
					undo := stack.Pop()
					top = *stack.Top() // ast_list
					currentNode := top.(ast_list)
					currentNode.remove_last()
					percentOrDollar := undo.(initialTokenGetter).token() // "%"  or "$"
					currentNode.append_ast(new_ast_string(stringToken(percentOrDollar)))

					if token == tokenEnd {
						embeddedErrorMessage := embeddedParseError(`Interpolation string ends in '%s'`, percentOrDollar)
						newNode := new_ast_string(stringToken(embeddedErrorMessage))
						currentNode.append_ast(newNode)
						set_error(`Interpolation string ends in unescaped '%s'`, percentOrDollar)
						goto redo // re-read tokenEnd in parseMode_Error
					} else {
						embeddedErrorMessage := embeddedParseError(`Invalid token %s after %s`, token.String(), percentOrDollar)
						newNode := new_ast_string(stringToken(embeddedErrorMessage))
						currentNode.append_ast(newNode)
						set_error(`Invalid token %s after %s`, token.String(), percentOrDollar) // sets mode to invalid
						goto redo                                                               // re-read and re-interpret the token following % or $
					}

				} // ok == true, token_string is an actual string
				top.(fmtStringSetter).set_formatString(token_string)
				mode = parseMode_OpenVariable // expect to read {, followed by variable name next
			}
		case parseMode_Condition: // expect to read a condition string (which must be a (non-empty) string literal)
			token_string, ok := token.(stringToken)
			if !ok {

				// remove the already-place ast_condPercent/ast_condDollar and replace it by a literal %! or $!
				undo := stack.Pop() // ast_condPercent or ast_condDollar
				top = *stack.Top()  // ast_list
				currentNode := top.(ast_list)
				currentNode.remove_last()
				percentOrDollarExlamMark := undo.(initialTokenGetter).token() // "%!"  or "$!"
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarExlamMark)))

				if token == tokenEnd {
					embeddedErrorMessage := embeddedParseError(`Interpolation string ends in "%s"`, percentOrDollarExlamMark)
					newNode := new_ast_string(stringToken(embeddedErrorMessage))
					currentNode.append_ast(newNode)
					set_error(`Interpolation string ends in unescaped "%s"`, percentOrDollarExlamMark)
					goto redo // re-read tokenEnd in parseMode_Error
				} else {
					embeddedErrorMessage := embeddedParseError(`Invalid token '%s' after "%s"`, token.String(), percentOrDollarExlamMark)
					newNode := new_ast_string(stringToken(embeddedErrorMessage))
					currentNode.append_ast(newNode)
					set_error(`Invalid token '%s' after "%s"`, token.String(), percentOrDollarExlamMark) // sets mode to invalid
					goto redo                                                                            // re-read and re-interpret the token following % or $
				}

			}
			// ok == true, the token we just read is a string. It cannot be empty due to how the tokenizer works.
			top.(conditionSetter).set_condition(token_string)
			mode = parseMode_OpenSequence // expect to read { next, followed by a sequence.
		case parseMode_VariableName: // expect to read the name of a variable (which must be a string literal) after having processed %fmtString{
			// The stack is (from top to bottom) ast_fmt - {ast_list - ast_cond -}* ast_list - ast_root
			// with the top already containing the format string
			token_string, ok := token.(stringToken)
			if !ok {
				// completely remove the ast_fmt and replace it by the literal string that was read so far.
				undo := stack.Pop() // ast_fmtPercent or ast_fmtDolalr
				top = *stack.Top()  // ast_list
				currentNode := top.(ast_list)
				currentNode.remove_last()
				fmtString := undo.(fmtStringGetter).get_formatString() // format string
				percentOrDollar := undo.(initialTokenGetter).token()   // "%" or "$"
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar + fmtString)))

				if token == tokenEnd {
					embeddedErrorMessage := embeddedParseError(`Interpolation string ends where variable name was expected`)
					newNode := new_ast_string(stringToken(embeddedErrorMessage))
					currentNode.append_ast(newNode)
					set_error(`Interpolation string ends where variable name was expected`)
					goto redo
				} else {
					embeddedErrorMessage := embeddedParseError(`Got "%v" where variable name was expected`, token.String())
					newNode := new_ast_string(stringToken(embeddedErrorMessage))
					currentNode.append_ast(newNode)
					set_error(`Got "%v" where variable name was expected`, token.String())
					goto redo // re-read current token
				}

			} // good case:
			top.(variableNameSetter).set_variableName(token_string)
			mode = parseMode_CloseVariable // expect to read } next

		case parseMode_OpenSequence: // expect to read a { after %!COND or $!COND
			// parseMode_OpenSequence only happens after reading a string token in mode parseMode_Condition.
			token := token.(specialToken) // token of type string cannot happen, because consecutive string tokens are merged by the tokenizer, so panic on type-assertion failure is OK.
			if token != tokenOpenBracket {

				// completely remove the ast_cond and replace it by the literal string that was read so far.
				undo := stack.Pop() // ast_condPercent or ast_condDolalr
				top = *stack.Top()  // ast_list
				currentNode := top.(ast_list)
				currentNode.remove_last()
				condition := undo.(conditionGetter).get_condition()        // condition
				percentOrDollarExclam := undo.(initialTokenGetter).token() // "%!" or "$!"

				// Note: The pattern %!Cond with missing { is likely because of a stray %! or $! that is not intended as a condition at all.
				// For that reason, we place the embedded error message just after the %! or $! rather than at the place where we expect the {

				currentNode.append_ast(new_ast_string(stringToken(percentOrDollarExclam)))

				embeddedErrorMessage := embeddedParseError(`"%v" has no matching '{'`, percentOrDollarExclam)
				currentNode.append_ast(new_ast_string(stringToken(embeddedErrorMessage)))
				currentNode.append_ast(new_ast_string(stringToken(condition)))
				set_error(`Missing '{' after %vCondition`, percentOrDollarExclam)
				goto redo // reread token. This may well be tokenEnd, which is fine.
			}
			// good case: We create a new sub-list
			newList := new_ast_list()
			top.(childSetter).set_child(newList)
			stack.Push(newList)
			mode = parseMode_Sequence
		case parseMode_OpenVariable: // expect to read a { initiating a variable name
			// parseMode_OpenVariable only happens after reading a string token in mode parseMode_FmtString.
			// Since consecutive string tokens are merged by the tokenizer, panic on type-assertion failure is fine:
			// Missing format string jumps directly from parseMode_FmtString to parseMode_VariableName.
			token := token.(specialToken)
			if token != tokenOpenBracket {

				// completely remove the ast_fmt and replace it by the literal string that was read so far.
				undo := stack.Pop() // ast_fmtPercent or ast_fmtDollar
				top = *stack.Top()  // ast_list
				currentNode := top.(ast_list)
				currentNode.remove_last()
				formatString := undo.(fmtStringGetter).get_formatString() // formatString
				percentOrDollar := undo.(initialTokenGetter).token()      // "%" or "$"

				// Note: The pattern %FmtString or $FmtString with missing { is likely because of a stray unescaped % or $ that is not intended as a formatting string at all.
				// For that reason, we place the embedded error message just after the % or $ rather than at the place where we expect the {

				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar)))

				embeddedErrorMessage := embeddedParseError(`unescaped '%v' has no matching '{'`, percentOrDollar)
				currentNode.append_ast(new_ast_string(stringToken(embeddedErrorMessage)))
				currentNode.append_ast(new_ast_string(stringToken(formatString)))
				set_error(`Missing '{' after %vFmtString`, percentOrDollar)
				goto redo // reread token. This may well be tokenEnd, which is fine.
			}
			// good case: { was present. Proceed to read variable name
			mode = parseMode_VariableName

		case parseMode_CloseVariable: // expect to read a } after a variable name

			// pretend we just read a }, no matter what.
			undo := stack.Pop()
			mode = parseMode_Sequence
			formatString := undo.(fmtStringGetter).get_formatString() // formatString

			// If what we read actually was not }, insert an error string and a literal interpretation of %FmtString{VariableName and re-read token

			// We previously read a string, so token is a specialToken (no consecutive string tokens above).
			token := token.(specialToken)
			// token must be tokenCloseBracket. The case distinction is just for the error message.

			if token != tokenCloseBracket {
				top = *stack.Top() // ast_list
				currentNode := top.(ast_list)
				currentNode.remove_last()

				percentOrDollar := undo.(initialTokenGetter).token()         // "%" or "$"
				VariableName := undo.(variableNameGetter).get_variableName() // variableName

				currentNode.remove_last()
				currentNode.append_ast(new_ast_string(stringToken(percentOrDollar + formatString + "{" + VariableName))) // replay what was read so far.
				if token == tokenEnd {
					embeddedErrorMessage := embeddedParseError(`unexpected end of format string after reading a variable name without closing '}'`)
					currentNode.append_ast(new_ast_string(stringToken(embeddedErrorMessage)))
					set_error(`Variable name not terminated by '}'`)
					goto redo

				} else {
					embeddedErrorMessage := embeddedParseError(`Variable name not terminated by '}'`)
					currentNode.append_ast(new_ast_string(stringToken(embeddedErrorMessage)))
					set_error(`Variable name not terminated by '}'`)
					goto redo
				}
			}
			// good case, token == tokenCloseBracket.
			if formatString == "" {
				undo.(fmtStringSetter).set_formatString(stringToken("v")) // no longer on stack, but still works.
			}
		case parseMode_Error:
			// Invariant: The stack looks exactly as follows (from the bottom:) ROOT, LIST followed by any number >=0 of COND,LIST pairs.
			currentNode := top.(ast_list) // top is a ast_list if we are in parseMode_Sequence
			if stack.Len() != 2 {
				panic("Cannot happen")
			}
			switch token := token.(type) {
			case stringToken:
				currentNode.append_ast(new_ast_string(token))
			case specialToken:
				if token != tokenEnd {
					currentNode.append_ast(new_ast_string(stringToken(token.String())))
				} else {
					_ = stack.Pop() // type popped is ast_list.
					root := stack.Pop().(ast_root)
					root.simplify()
				}
			}

		default:
			panic(ErrorPrefix + "Unhandled status in syntax tree parser")
		}
	}

	if (mode != parseMode_Sequence) && (mode != parseMode_Error) {
		panic(ErrorPrefix + "Cannot happen")
	}
	if stack.Len() != 0 {

		panic(ErrorPrefix + "Cannot happen")
	}
	if err != ret.(ast_root).parseError {
		panic(ErrorPrefix + "Cannot happen")
	}
	if (mode == parseMode_Error) != (err != nil) {
		panic(ErrorPrefix + "Cannot happen")
	}

	return
}
