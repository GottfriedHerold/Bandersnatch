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
//   - [Optional] Perform some validity checks. (2 subchecks, actually. Those would be checked when actually producing output anyway, but sometime we want those checks early)
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
// of type *T pointing to something of type quite different from T, leading to disaster (even if not dereferenced, the garbage collector might misbehave badly).
// Of course, things work fine if STH is itself a pointer -- which is kind-of enforced by making *T satisfy the interface: Then the interface directly stores a value of type *T.
// [2]: With usual production rules in BNF, a standard approach would not lead to ast_list storing n elements, but to a (binary) right/left-leaning tree.
// We take "production rule" to allow List -> SequenceElement* rules.
// This is in fact the only reason (apart from defaulting to 'v' for fmtString, which could be inserted by the tokenizer) why the language is not LL(0);

type ast_I interface {
	IsNode()        // Only to "mark" relevant types
	String() string // only for debugging
	// Interpolate(parameters_direct paramMap, parameters_passed paramMap, baseError error, s strings.Builder) (err error)
	// VerifyWeak(parameters_direct paramMap, baseError error) error
	// VerifyStrong(parameters_direct paramMap, parameters_passed paramMap, baseError error) error
}

// We add interfaces for extra functionality that is shared by multiple node types:
// This allows to cut down the state space that we need to (explicitly) track in our DFA.
type (
	childSetter        interface{ set_child(ast_I) }              // ast_root, ast_condPercent, ast_condDollar
	variableNameSetter interface{ set_variableName(stringToken) } // ast_fmtPercent, ast_fmtDollar
	fmtStringSetter    interface{ set_formatString(stringToken) } // ast_fmtPercent, ast_fmtDollar
	conditionSetter    interface{ set_condition(stringToken) }    // ast_condPercent, ast_condDollar
	simplifier         interface{ simplify() }                    // ast_root, ast_condPercent, ast_condDollar
)

// *****
// DEFINITIONS OF THE INVIDUAL IMPLEMENTATIONS OF NODE TYPES:
// *****

// root node
type (
	v_ast_root struct{ ast ast_I }
	ast_root   = *v_ast_root
)

// simplify assumes the child is of type ast_list. If len(child) == 1, replace it by the child.
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

// new_ast_fmtPercent creates a new node of type ast_fmtPercent. Its formatString and variableName have yet to be set.
func new_ast_fmtPercent() ast_I {
	return &v_ast_fmtPercent{}
}

// new_ast_fmtPercent creates a new node of type ast_fmtDollar. Its formatString and variableName have yet to be set.
func new_ast_fmtDollar() ast_I {
	return &v_ast_fmtDollar{}
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
}

// set_condition sets the condition string for an ast_condPercent or ast_condDollar.
// Note that this functions does not perform any validity checks on cond.
func (a *base_ast_condition) set_condition(cond stringToken) {
	a.condition = string(cond)
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

// We define String methods on each AST node type. These are only used for debugging.

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
	parseMode_OpenVariable                   // expecting a { to be followed by a variable name (after %fmtString)
	parseMode_CloseVariable                  // expecting a } terminating a variable name (other uses of } are handled by parseMode_Sequence)
)

// NOTE: We asserts that token_list has no consecutive tokens of type string to simplify error reporting
// (as in: this allows us to skip treating those case, which means that violations results in possibly non-informative panics rather than non-nil error output. )

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

	var mode parseMode = parseMode_Sequence

	for _, token := range tokens {
		var top ast_I = *stack.Top() // Peek at top of stack. NOTE: stack cannot be empty
		switch mode {
		case parseMode_Sequence: // expect to get a sequence of strings or tokens
			currentNode := top.(ast_list) // top is a ast_list

			switch token := token.(type) {
			case stringToken:
				newNode := new_ast_string(token)
				currentNode.append_ast(newNode)
			case specialToken:
				switch token {
				case tokenPercent:
					newNode := new_ast_fmtPercent()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString
				case tokenDollar:
					newNode := new_ast_fmtDollar()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_FmtString
				case tokenPercentCond:
					newNode := new_ast_condPercent()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition
				case tokenDollarCond:
					newNode := new_ast_condDollar()
					currentNode.append_ast(newNode)
					stack.Push(newNode)
					mode = parseMode_Condition
				case tokenOpenBracket:
					err = fmt.Errorf(ErrorPrefix + "Unexpected { in format string")
					return
				case tokenCloseBracket, tokenEnd:
					// } cannot appear at the top level.
					if (token == tokenCloseBracket) && (stack.Len() <= 3) { // We have at least ROOT - LIST - COND - LIST on the stack if } is valid.
						err = fmt.Errorf(ErrorPrefix + "Unexpected } in format string")
						return
					} // EOF must only appear at the top level
					if (token == tokenEnd) && (stack.Len() != 2) {
						err = fmt.Errorf(ErrorPrefix + "Missing } in format string")
						return
					}
					// stack.Len() >= 2 is guaranteed

					_ = stack.Pop() // type is ast_list
					rootOrCond := stack.Pop()
					rootOrCond.(simplifier).simplify() // is the child list contains 1 element, replace list by single element.

					// If token == EOF, mode no longer matters
					// If token == parseMode_Sequence, we continue with the parent sequence
					mode = parseMode_Sequence // no-op, but added for clarity

				case tokenParentPercent:
					newNode := new_ast_parentPercent()
					currentNode.append_ast(newNode)
				case tokenParentDollar:
					newNode := new_ast_parentDollar()
					currentNode.append_ast(newNode)
				default:
					panic(fmt.Errorf(ErrorPrefix+"Unhandled token: %v", token))
				}
			default: // token not of type stringToken or specialToken
				panic(ErrorPrefix + "Invalid entry in token list")
			}

		case parseMode_FmtString: // expect to read format string (which must be a string literal)
			if token == tokenOpenBracket { // %{ or ${ is interpreted as %v{ or $v{
				top.(fmtStringSetter).set_formatString(stringToken('v'))
				mode = parseMode_VariableName
			} else {
				token_string, ok := token.(stringToken)
				if !ok {
					err = fmt.Errorf("%s", `Missing format verb. % or $ must be followed by an (optional) format verb ("v" if absent), then {VariableName}.Literal % must be (possibly doubly) escaped as \%`)
					return
				}
				top.(fmtStringSetter).set_formatString(token_string)
				mode = parseMode_OpenVariable // expect to read {, followed by variable name next
			}
		case parseMode_Condition: // expect to read a condition string (which must be a non-empty string literal)
			token_string, ok := token.(stringToken)
			if !ok {
				err = fmt.Errorf("%s", `Missing conditional. %! or $! must be followed by a (non-empty) condition, then {format string}`)
				return
			}
			top.(conditionSetter).set_condition(token_string)

			mode = parseMode_OpenSequence // expect to read {, followed by a sequence
		case parseMode_VariableName: // expect to read the name of a variable (which must be a string literal)
			token_string, ok := token.(stringToken)
			if !ok {
				err = fmt.Errorf("%s", `unescaped control character or EOF occurred in format string where the name of a variable was expected`)
				return
			}
			top.(variableNameSetter).set_variableName(token_string)
			mode = parseMode_CloseVariable // expect to read } next

		case parseMode_OpenSequence: // expect to read a after %!COND or $!COND
			// parseMode_OpenSequence only happens after reading a string token in mode parseMode_Condition.
			token := token.(specialToken) // token of type string cannot happen, because consecutive string tokens are merged by the tokenizer, so panic on type-assertion failure is OK.
			if token != tokenOpenBracket {
				err = fmt.Errorf("%s", `%!Condition or $!Condition must be followed by a {...}`)
				return
			}

			newList := new_ast_list()
			top.(childSetter).set_child(newList)
			stack.Push(newList)

			mode = parseMode_Sequence
		case parseMode_OpenVariable: // expect to read a { initiating a variable name
			// parseMode_OpenVariable only happens after reading a string token in mode parseMode_FmtString.
			// Since consecutive string tokens are merged by the tokenizer, panic on type-assertion failure is fine:
			// Empty format string jumps directly from parseMode_FmtString to parseMode_VariableName
			token := token.(specialToken)
			if token != tokenOpenBracket {
				err = fmt.Errorf("%s", `%fmtString or $fmtString (with possibly empty fmtString) must be followed by a {VariableName}. Missing {`)
				return
			}
			mode = parseMode_VariableName
		case parseMode_CloseVariable: // expect to read a } after a variable name
			// We previously read a string, to token is a specialToken (see above).
			token := token.(specialToken)
			// token must be tokenCloseBracket. The case distinction is just for the error message.
			if token == tokenEnd {
				err = fmt.Errorf("%s", `unexpected end of format string after reading a variable name without closing }.`)
				return
			}
			if token != tokenCloseBracket {
				err = fmt.Errorf("%s", `parsing error: In %fmtString{VariableName}, VariableName contained an unescaped control character`)
				return
			}
			stack.Pop()
			mode = parseMode_Sequence

		default:
			panic(ErrorPrefix + "Unhandled status in syntax tree parser")
		}
	}

	if stack.Len() != 0 {
		panic(ErrorPrefix + "Cannot happen")
	}
	if err != nil {
		panic(ErrorPrefix + "Cannot happen")
	}

	return
}

func (a ast_string) Interpolate(p1 paramMap, p2 paramMap, b error, s strings.Builder) error {
	s.WriteString(string(a))
	return nil
}

func (a ast_string) VerifyWeak(paramMap, error) error {
	return nil
}

func (a ast_string) VerifyStrong(paramMap, paramMap, error) error {
	return nil
}
