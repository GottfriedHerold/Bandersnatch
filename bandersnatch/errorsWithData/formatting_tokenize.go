package errorsWithData

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Parsing and Interpolating format strings goes through multiple steps:
//
//   - Tokenize the interpolation string
//   - Parse the tokenized string into a syntax tree
//   - [Optional] Perform some validity checks. (2 subchecks, actually. Those would be checked when actually producing output anyway, but sometime we want those checks early)
//   - Actually prodcuce the interpolated error string.

// This file contains the tokenizing code

// We tokenize the input string in the following way:
// We recognize %, $, {, }, %!, $!, %w and $w as special tokens.
// Note that the Go regexp a|b prefers a over b. We use this to (greedily) prefer $! and $w over $ and similarly for %! and %w over %
// Special-casing %w and %! (rather than viewing w and ! as part of the subsequent string) makes parsing easier.
// We add a start and end token at the beginning / end. This simplifies the parsing code.
// Consecutive string tokens get concatenated into a single string token. This includes string tokens that result from escape sequences for %,$,{,}

// Regular expression to greedily subdivide the input string into non-overlapping instances of
//   - all escape sequences %%, %$, $$, {{, %}
//   - all token sequences %!, $!, %, $, {, }, %w, $w
//   - strings without $, %, {, }
//
// NOTE: (?s) turns off special handling newlines in the string to be tokenized. Literal $ and { and } in the regexp string must be escaped as \$ \{ \}
// NOTE2: %%, %w, %} must come before % etc, because | is greedy.
var re_tokenize = regexp.MustCompile(`(?s)(%%|%$|\$\$|\{\{|%\}|%!|\$!|%w|\$w|%|\$|\{|\}|[^\$\{\}%]+)`)

type token_I interface {
	IsToken()       // only used to mark the types as valid for token_I
	String() string // only used for debugging
}

// specialToken is an enum type for special tokens that appear in interpolation strings such as %, $ etc.
type specialToken int

// IsToken just serves to satify token_I
func (specialToken) IsToken() {}

// stringToken is a type holding literal string tokens arising from tokenizing interpolation strings
type stringToken string

// IsToken just serves to satify token_I
func (stringToken) IsToken() {}

// String just converts the stringToken back to the string.
// NOTE regarding escaping: If the original interpolation string was `%%`, tokenizing results in the stringToken `%` and stringToken.String outputs `%`).
func (s stringToken) String() string { return string(s) }

type tokenList []token_I // each entry is either a stringToken or a specialToken. The first and last are the (only) tokens with values tokenStart and tokenEnd

const (
	tokenInvalid       specialToken = iota // zero value intentionally invalid
	tokenPercent                           // % - token (not followed by ! or w)
	tokenDollar                            // $ - token (not followed by ! or w)
	tokenPercentCond                       // %!
	tokenDollarCond                        // $!
	tokenOpenBracket                       // {
	tokenCloseBracket                      // }
	tokenParentPercent                     // %w
	tokenParentDollar                      // $w
	tokenStart                             // added to the start of the tokenized string; this simplifies things a bit
	tokenEnd                               // added to the end of the tokenized string; this simplifies things a bit
)

// String will output a string representation of the special token. It mostly matches the string that gets parsed into it
func (token specialToken) String() string {
	switch token {
	case tokenInvalid:
		return `INVALID TOKEN`
	case tokenPercent:
		return `%`
	case tokenDollar:
		return `$`
	case tokenPercentCond:
		return `%!`
	case tokenDollarCond:
		return `$!`
	case tokenOpenBracket:
		return `{`
	case tokenCloseBracket:
		return `}`
	case tokenParentPercent:
		return `%w`
	case tokenParentDollar:
		return `$w`
	case tokenEnd:
		return `]`
	case tokenStart:
		return `[`
	default:
		panic(ErrorPrefix + " Unknown token encountered")
	}
}

// String will output a string representation of the token list.
// For valid tokenList, this should have the form `[ TOKEN1 TOKEN2 TOKEN3 ]`
// where for each special token we output a string that would be parsed into the corresponding token and
// each string token get output with ""s added.
// (tokenStart and tokenEnd correspond to the [ ])
func (tokens tokenList) String() string {
	var ret strings.Builder
	for i, t := range tokens {
		if i > 0 {
			ret.WriteRune(' ')
		}
		switch t := t.(type) {
		case stringToken:
			ret.WriteByte('"')
			ret.WriteString(t.String())
			ret.WriteByte('"')
		case specialToken:
			ret.WriteString(t.String())
		default:
			panic(ErrorPrefix + "Invalid type in token_list")
		}
	}
	return ret.String()
}

func tokenizeFormatString(s string) (ret tokenList) {
	if !utf8.ValidString(s) {
		panic(ErrorPrefix + "formatString not a valid UTF-8 string")
	}
	decomposition := re_tokenize.FindAllString(s, -1)
	ret = make(tokenList, len(decomposition)+2) // +2 comes from tokenStart and tokenEnd.
	ret[0] = tokenStart
	i := 1 // index of the next token to be added. Because we merge consecutive strings (which modifies this), we don't use i, entry := range decomposition
	for _, entry := range decomposition {
		switch entry {
		case `%%`:
			ret[i] = stringToken(`%`)
		case `%$`, `$$`:
			ret[i] = stringToken(`$`)
		case `{{`:
			ret[i] = stringToken(`{`)
		case `%}`:
			ret[i] = stringToken(`}`)
		case `%!`:
			ret[i] = tokenPercentCond
		case `$!`:
			ret[i] = tokenDollarCond
		case `%`:
			ret[i] = tokenPercent
		case `$`:
			ret[i] = tokenDollar
		case `{`:
			ret[i] = tokenOpenBracket
		case `}`:
			ret[i] = tokenCloseBracket
		case `%w`:
			ret[i] = tokenParentPercent
		case `$w`:
			ret[i] = tokenParentDollar
		default:
			ret[i] = stringToken(entry)
		}

		// merge consecutive entries of type stringToken.
		// This is required for escaped %,$,{ or } that appear in identifiers such as format string verbs.
		// It also makes writing the parser easier if we know that no consecutive stringTokens appear.
		if i > 0 {
			newlyadded, ok1 := ret[i].(stringToken)
			addedbefore, ok2 := ret[i-1].(stringToken)
			if ok1 && ok2 {
				ret[i-1] = stringToken(addedbefore.String() + newlyadded.String())
				i--
			}

		}

		i++
	}
	ret[i] = tokenEnd
	ret = ret[0 : i+1 : i+1]
	return
}
