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

// Literal `$`, `\`, `{` and `}` in a regexp string must be escaped as `\$`, `\\`, `\{` and `\}`.
// This strings.Replaces performs these escapes.
// This is just used to simplify readability of the definition of [re_tokenize].
//
// Note that re_escaper escapes all occurances, thereby precluding to use e.g. $ as its regexp-specific meaning (end of text/line)
// We use `...`-string syntax rather than "", because with the latter we would have to additionally escape all \'s in another layer...
var re_escaper = strings.NewReplacer(`\`, `\\`, `$`, `\$`, `{`, `\{`, `}`, `\}`)

// Regular expression to greedily subdivide the input string into non-overlapping instances of
//   - all escape sequences \%, \$, \{, \}, \\, %%
//   - all token sequences %!, $!, %, $, {, }, %w, $w
//   - strings without $, %, {, }, \
//   - plain unescaped \ (not followed by %, $, {, } or another \) -- taken as literal \
//
// NOTE: (?s) turns off special handling of newlines within the string to be tokenized.
// NOTE2: %%, %w, %!, must come before % etc, because | is greedy.
// NOTE3: We don't have a $$ - escape for $, because this makes $$$ ambiguous. For %%%, we parse as literal %, followed by token %, because the other
// order is always invalid (format verbs cannot start with %). For $$$, both orders are potentially valid ($ is a legit format string verb)
var re_tokenize = regexp.MustCompile(re_escaper.Replace(`(?s)(\%|\$|\{|\}|\\|%%|%!|$!|%w|$w|%|$|{|}|[^${}%\]+|\)`))

// token_I is the interface type holding a single token produced by the tokenizer.
// We provide two implementations:
//   - [stringToken] for string tokens
//   - [specialToken] for active tokens. This is an enum type; we don't need different types for different active tokens
type token_I interface {
	IsToken()       // only used to mark the types as valid for token_I
	String() string // returns a string representation of the token. For [specialToken], this gives the defining sequence. For [stringToken], gives the (unescaped) string.
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
// NOTE regarding escaping: If the original interpolation string was `%%`, tokenizing results in stringToken(`%`) and stringToken.String outputs `%`).
func (s stringToken) String() string { return string(s) }

// tokenList is a list of tokens, where each entry is either a stringToken or a specialToken.
// The first and last are the (only) tokens with values tokenStart and tokenEnd
// All string tokens are non-empty and no two consecutive string tokens appear.
type tokenList []token_I

// enum for all potential tokens of type [specialToken]
const (
	tokenInvalid       specialToken = iota // zero value intentionally invalid (to aid debugging -- this indicates that a variable has not been set), must never appear
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

// list of all tokens resp. all tokens that can be produced from strings. This is only used in testing.
var (
	allSpecialTokens                  = []specialToken{tokenPercent, tokenDollar, tokenPercentCond, tokenDollarCond, tokenOpenBracket, tokenCloseBracket, tokenParentPercent, tokenParentDollar, tokenStart, tokenEnd, tokenInvalid}
	allStringExpressibleSpecialTokens = []specialToken{tokenPercent, tokenDollar, tokenPercentCond, tokenDollarCond, tokenOpenBracket, tokenCloseBracket, tokenParentPercent, tokenParentDollar}
)

// tokenizeInterpolationString takes a string and tokenizes it.
//
// The resulting tokenList is a list of tokens with the properties that
//   - tokens are either stringTokens or special tokens
//   - no two consecutive tokens are string tokens
//   - The first and last tokens are tokenStart and tokenEnd and those only appear at the start and the end
//   - string tokens are non-empty
//
// Tokenizing cannot fail. Invalid utf-8 strings are handled by using the UTF-8 replacement character.
func tokenizeInterpolationString(s string) (ret tokenList) {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, string(utf8.RuneError)) // replace all invalid UTF-8 code points by replacement character
		// panic(ErrorPrefix + "formatString not a valid UTF-8 string")
	}
	decomposition := re_tokenize.FindAllString(s, -1)
	// We pre-allocate ret under the assumption that all elements of decomposition end up in its own token and set len(ret) accordingly.
	// Due to merging consecutive string tokens, the final result might be a shorter list. We fix this at the end.
	ret = make(tokenList, len(decomposition)+2) // +2 comes from tokenStart and tokenEnd.

	ret[0] = tokenStart
	i := 1 // index of the next token to be added. Because we merge consecutive strings (which modifies i), we don't use i, entry := range decomposition
	for _, entry := range decomposition {
		switch entry {
		case `\%`, `%%`:
			ret[i] = stringToken(`%`)
		case `\$`:
			ret[i] = stringToken(`$`)
		case `\{`:
			ret[i] = stringToken(`{`)
		case `\}`:
			ret[i] = stringToken(`}`)
		case `\\`, `\`:
			ret[i] = stringToken(`\`)
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
		// It also makes writing the parser significantly(!) easier if we know that no consecutive stringTokens appear.
		if i > 0 { // always true, actually.
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
	ret = ret[0 : i+1 : i+1] // to account for merging of consecutive stringTokens
	return
}

// String will output a string representation of the special token. It matches the string that gets parsed into it.
//
// For tokenStart resp. tokenEnd, we return a "[" resp. "]".
// This is not really important, but convenient for tokenList.String() and our tests rely on this (as it makes writing test-cases simpler).
func (token specialToken) String() string {
	switch token {
	case tokenInvalid:
		return `INVALID TOKEN` // must never happen. We don't panic, because this may be called during recover() by the testing framework for diagnostics.
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
		panic(ErrorPrefix + "internal error: Unknown token encountered") // cannot happen
	}
}

// String will output a string representation of the token list.
// For valid tokenList, this should have the form `[ TOKEN1 TOKEN2 TOKEN3 ]`.
// where for each special token we output a string that corresponds to the corresponding token and
// each string token get output with ""s added.
// (tokenStart and tokenEnd correspond to the [ ])
// We use a space as separator.
//
// Note that this function is only used internally, for testing the package itself. This output format is not part of the API.
// The user has no way of getting hold of an instance of tokenList, so this cannot be called.
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
			panic(ErrorPrefix + "internal error: Invalid type in token_list") // cannot happen
		}
	}
	return ret.String()
}
