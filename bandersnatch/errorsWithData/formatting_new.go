package errorsWithData

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/GottfriedHerold/Bandersnatch/internal/stack"
)

type paramMap = map[string]any

type parsed_token int

var re_tokenize = regexp.MustCompile(`(?s)(%%|%$|%!|\$!|\$\$|\{\{|%\}|%w|\$w|%\{|\$\{|%|\$|\{|\}|[^\$\{\}%]+)`)

const (
	invalidToken             parsed_token = iota // zero value intentially invalid
	percent                                      // % - token (not followed by ! or w)
	dollar                                       // $ - token (not followed by ! or w)
	percent_special                              // %!
	dollar_special                               // $!
	open_bracket                                 // {
	close_bracket                                // }
	parent_error_simple                          // %w
	parent_error_passthrough                     // $w
	EOF                                          // internally added to the end of the tokenized string by make_ast; this simplifies things a bit
)

func (token parsed_token) String() string {
	switch token {
	case invalidToken:
		return `INVALID TOKEN`
	case percent:
		return `%`
	case dollar:
		return `$`
	case percent_special:
		return `%!`
	case dollar_special:
		return `$!`
	case open_bracket:
		return `{`
	case close_bracket:
		return `}`
	case parent_error_simple:
		return `%w`
	case parent_error_passthrough:
		return `$w`
	case EOF:
		return `EOF`
	default:
		panic(ErrorPrefix + " Unknown token encountered")
	}
}

type token_list []any // each entry is either a string or a parsed_token

func (tokens token_list) String() string {
	var ret strings.Builder
	ret.WriteRune('[')
	for i, t := range tokens {
		if i > 0 {
			ret.WriteRune(' ')
		}
		switch t := t.(type) {
		case string:
			ret.WriteByte('"')
			ret.WriteString(t)
			ret.WriteByte('"')
		case parsed_token:
			ret.WriteString(t.String())
		default:
			panic(ErrorPrefix + "Invalid type in token_list")
		}
	}
	ret.WriteRune(']')
	return ret.String()
}

func tokenizeFormatString(s string) (ret token_list) {
	if !utf8.ValidString(s) {
		panic(ErrorPrefix + "formatString not a valid UTF-8 string")
	}
	decomposition := re_tokenize.FindAllString(s, -1)
	ret = make(token_list, 3*len(decomposition))
	i := 0
	for _, entry := range decomposition {
		switch entry {
		case `%%`:
			ret[i] = `%`
		case `%$`, `$$`:
			ret[i] = `$`
		case `{{`:
			ret[i] = `{`
		case `%}`:
			ret[i] = `}`
		case `%!`:
			ret[i] = percent_special
		case `$!`:
			ret[i] = dollar_special
		case `%`:
			ret[i] = percent
		case `$`:
			ret[i] = dollar
		case `{`:
			ret[i] = open_bracket
		case `}`:
			ret[i] = close_bracket
		case `%w`:
			ret[i] = parent_error_simple
		case `$w`:
			ret[i] = parent_error_passthrough
		case `%{`:
			ret[i] = percent
			i++
			ret[i] = `v`
			i++
			ret[i] = open_bracket
		case `${`:
			ret[i] = dollar
			i++
			ret[i] = `v`
			i++
			ret[i] = open_bracket
		default:
			ret[i] = entry
		}
		// merge consecutive entries of type string. This is required for escped %,$,{ or } that appear in identifiers such as format string verbs.

		if i > 0 {
			newlyadded, ok1 := ret[i].(string)
			addedbefore, ok2 := ret[i-1].(string)
			if ok1 && ok2 {
				ret[i-1] = addedbefore + newlyadded
				i--
			}

		}

		i++
	}
	ret = ret[0:i:i]
	return
}

type ast_I interface {
	// Interpolate(parameters_direct paramMap, parameters_passed paramMap, baseError error, s strings.Builder) (err error)
	// VerifyWeak(parameters_direct paramMap, baseError error) error
	// VerifyStrong(parameters_direct paramMap, parameters_passed paramMap, baseError error) error
}

type ast_root struct {
	ast *ast_I
}

func (a *ast_root) set_child(child *ast_I) {
	a.ast = child
}

func new_ast_root(child *ast_I) ast_I {
	return ast_root{ast: child}
}

type ast_list struct {
	l []*ast_I
}

func new_ast_list() ast_I {
	return &ast_list{l: make([]*ast_I, 0)}
}

type ast_parent_error_simple struct{}

func new_ast_parent_error_simple() ast_I {
	return &ast_parent_error_simple{}
}

type ast_parent_error_passthrough struct{}

func new_ast_parent_error_passthrough() ast_I {
	return &ast_parent_error_passthrough{}
}

type ast_fmt struct {
	formatString string
	variableName string
}

func (a *ast_fmt) set_formatString(formatString string) {
	a.formatString = formatString
}

func (a *ast_fmt) set_variableName(variableName string) {
	a.variableName = variableName
}

type ast_fmt_percent struct {
	ast_fmt
}

func new_ast_fmt_percent() ast_I {
	return &ast_fmt_percent{}
}

type ast_fmt_dollar struct {
	ast_fmt
}

func new_ast_fmt_dollar() ast_I {
	return &ast_fmt_dollar{}
}

type ast_string string

func new_ast_string(s string) ast_I {
	return ast_string(s)
}

type ast_special struct {
	condition string
	arg       *ast_I
}

func (a *ast_special) set_condition(cond string) {
	a.condition = cond
}

func (a *ast_special) set_child(child *ast_I) {
	a.arg = child
}

type ast_special_percent struct {
	ast_special
}

func new_ast_special_percent() ast_I {
	return &ast_special_percent{}
}

type ast_special_dollar struct {
	ast_special
}

func new_ast_special_dollar() ast_I {
	return &ast_special_dollar{}
}

type parseMode int

const (
	parseMode_Sequence parseMode = iota
	parseMode_FmtString
	parseMode_Condition
	parseMode_VariableName
	parseMode_OpenSequence
	parseMode_OpenVariable
	parseMode_CloseVariable
	// parseMode_Finish

	// parseMode_SubSequence
)

// NOTE: Asserts that token_list has no consecutive tokens of type string
// NOTE: Asserts that token_list has no percent-openbracket or dollar-openbracket sequences. The tokenizer inserts a (default) `v` fmtString.

func make_ast(tokens token_list) (ret ast_I, err error) {

	// we internally prepend a { and append a EOF.
	// This has the advantage of somewhat unifying parsing the top-level sequence with subsquences Bar arising from %!Foo{Bar} or $!Foo{Bar}
	tokens_new := make(token_list, 1, len(tokens)+2)
	tokens_new[0] = open_bracket
	tokens_new = append(tokens_new, tokens...)
	tokens_new = append(tokens_new, EOF)
	tokens = tokens_new

	// the top of our tree (apart from wrapping with a special root node) is a list, starting empty.
	// Note that we squash 1-element lists after we are finished building it.
	initial_list := new_ast_list()
	top_ast := new_ast_root(&initial_list)

	// since %!Foo{Bar} and $!Foo{Bar} can be nested, we can actually get a tree of arbitrary depth
	// We maintain a stack that contains pointers to the ast_nodes on the current path to the leaf we are working with.
	stack := stack.MakeStack[*ast_I]()
	stack.Push(&top_ast)
	stack.Push(&initial_list)

	// we initially expect to read { as part of sequence
	mode := parseMode_OpenSequence

	for _, token := range tokens {
		var top *ast_I = *stack.Top() // NOTE: stack cannot be empty
		switch mode {
		case parseMode_Sequence: // expect to get a sequence of strings or tokens
			currentNode := (*top).(*ast_list) // *top is a *ast_list

			switch token := token.(type) {
			case string:
				newNode := new_ast_string(token)
				currentNode.l = append(currentNode.l, &newNode)
			case parsed_token:
				switch token {
				case percent:
					newNode := new_ast_fmt_percent()
					currentNode.l = append(currentNode.l, &newNode)
					stack.Push(&newNode)
					mode = parseMode_FmtString
				case dollar:
					newNode := new_ast_fmt_dollar()
					currentNode.l = append(currentNode.l, &newNode)
					stack.Push(&newNode)
					mode = parseMode_FmtString
				case percent_special:
					newNode := new_ast_special_percent()
					currentNode.l = append(currentNode.l, &newNode)
					newList := new_ast_list()
					newNode.(interface{ set_child(*ast_I) }).set_child(&newList)
					stack.Push(&newNode)
					stack.Push(&newList)
					mode = parseMode_Condition
				case dollar_special:
					newNode := new_ast_special_dollar()
					currentNode.l = append(currentNode.l, &newNode)
					newList := new_ast_list()
					newNode.(interface{ set_child(*ast_I) }).set_child(&newList)
					stack.Push(&newNode)
					stack.Push(&newList)
					mode = parseMode_Condition
				case open_bracket:
					err = fmt.Errorf(ErrorPrefix + "Unexpected { in format string")
					return
				case close_bracket, EOF:
					// } cannot appear at the top level.
					if (token == close_bracket) && (stack.Len() <= 3) { // We have TOP - LIST - special - LIST at the start of the stack if } is valid.
						err = fmt.Errorf(ErrorPrefix + "Unexpected } in format string")
						return
					} // EOF must only appear at the top level
					if (token == EOF) && (stack.Len() != 2) {
						err = fmt.Errorf(ErrorPrefix + "Missing } in format string")
					}
					// stack.Len() >= 2 is guaranteed

					// squash list if one element:
					if len(currentNode.l) == 1 {
						var singleElementPtr *ast_I = currentNode.l[0]
						_ = stack.Pop()          // gives a copy of top
						parentPtr := stack.Pop() // pointer to parent node (of type either root, special_percent or special_dollar)
						(*parentPtr).(interface{ set_child(*ast_I) }).set_child(singleElementPtr)
					} else {
						stack.Pop()
						stack.Pop()
					}

					// If token == EOF, mode no longer matters
					// If token == parseMode_Sequence, we continue with the parent sequence
					mode = parseMode_Sequence // no-op, but added for clarity

				case parent_error_simple:
					newNode := new_ast_parent_error_simple()
					currentNode.l = append(currentNode.l, &newNode)
				case parent_error_passthrough:
					newNode := new_ast_parent_error_passthrough()
					currentNode.l = append(currentNode.l, &newNode)
				default:
					panic(ErrorPrefix + "Unhandled token")
				}
			default: // token not of type string or parsed_token
				panic(ErrorPrefix + "Invalid entry in token list")
			}

		case parseMode_FmtString: // expect to read format string (which must be a string literal)
			token_string, ok := token.(string)
			if !ok {
				err = fmt.Errorf("%s", `Missing format verb. % or $ must be followed by an (optional) format verb ("v" if absent), then {VariableName}`)
				return
			}
			(*top).(interface{ set_formatString(string) }).set_formatString(token_string)
			mode = parseMode_OpenVariable // expect to read {, followed by variable name next
		case parseMode_Condition: // expect to read a condition string (which must be a string literal)
			token_string, ok := token.(string)
			if !ok {
				err = fmt.Errorf("%s", `Missing conditional. %! or $! must be followed by a (non-empty) condition, then {format string}`)
				return
			}
			(*top).(interface{ set_condition(string) }).set_condition(token_string)
			mode = parseMode_OpenSequence // expect to read {, followed by a sequence
		case parseMode_VariableName: // expect to read the name of a variable (which must be a string literal)
			token_string, ok := token.(string)
			if !ok {
				err = fmt.Errorf("%s", `unescaped control character or EOF occurred in format string where the name of a variable was expected`)
				return
			}
			(*top).(interface{ set_variableName(string) }).set_variableName(token_string)
			mode = parseMode_CloseVariable // expect to read } next

		case parseMode_OpenSequence: // expect to read a { either at the very beginning or after %!COND or $!COND
			// parseMode_OpenSequence only happens at the beginning or after reading a string token in mode parseMode_Condition.
			token := token.(parsed_token) // token of type string cannot happen, because consecutive string tokens are merged by the tokenizer.
			if token != open_bracket {
				err = fmt.Errorf("%s", `%!Condition or $!Condition must be followed by a {...}`)
				return
			}
			mode = parseMode_Sequence
		case parseMode_OpenVariable: // expect to read a { initiating a variable name
			// parseMode_OpenVariable only happens after reading a string token in mode parseMode_FmtString.
			// Since consecutive string tokens are merged by the tokenizer, panic on type-assertion failure is fine:
			// (NOTE: The tokenizer inserts a `v` into ${} or %{}, so we can ignore the case of empty format strings either)
			token := token.(parsed_token)
			if token != open_bracket {
				err = fmt.Errorf("%s", `%fmtString or $fmtString (with possibly empty fmtString) must be followed by a {VariableName}. Missing {`)
				return
			}
			mode = parseMode_VariableName
		case parseMode_CloseVariable: // expect to read a } after a variable name
			// We previously read a string.
			token := token.(parsed_token)
			if token == EOF {
				err = fmt.Errorf("%s", `unexpected end of format string after reading a variable name without closing }.`)
				return
			}
			if token != close_bracket {
				err = fmt.Errorf("%s", `parsing error: In %fmtString{VariableName}, VariableName contained an unescaped control character`)
				return
			}
			stack.Pop()
			mode = parseMode_Sequence

		default:
			panic(ErrorPrefix + "Unhandled status in syntax tree parser")
		}
	}
	ret = top_ast
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
