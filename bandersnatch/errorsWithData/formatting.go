package errorsWithData

import (
	"fmt"
	"go/ast"
	"strings"
	"unicode/utf8"
)

// Providing this value as overrideMessage for creating an ErrorWithParameters will create an actual empty string
// (Giving a literal empty string will instead default to DefaultOverrideMessage)
const (
	OverrideByEmptyMessage = "%Empty"
	DefaultOverrideMessage = "%w%!M>0{ Included Parameters: %m}"
)
const nonEmptyMapFormatString = "!M>0" // without %

// FormatError will print/interpolate the given format string using the parameters map if output == true
// For output == false, it will only do some validity parsing checks.
// This is done in one function in order to de-duplicate code.
//
// The format is as follows: %FMT{arg} is printed like fmt.Printf(%FMT, parameters[arg])
// %% is used to escape literal %
// %m is used to print the parameters map itself
// %$nonEmptyMapFormatString{STR} will evaluate STR if len(parameters) > 0, nothing otherwise.
// %w will print baseError.Error()
func formatError(formatString string, parameters map[string]any, baseError error, output bool) (returned_string string, err error) {
	if !utf8.ValidString(formatString) {
		panic(ErrorPrefix + "formatString not a valid UTF-8 string")
	}

	// We build up the returned string piece-by-piece by writing to ret.
	// This avoids some allocations & copying
	var ret strings.Builder
	if output {
		defer func() {
			returned_string = ret.String()
			if err != nil {
				returned_string += fmt.Sprintf("<error when printing error: %v>", err)
			}
		}()
	}

	var suffix string = formatString // holds the remaining yet-unprocessed part of the input.

	for { // annoying to write as a "usual" for loop, because both init and iteration would need to consist of 2 lines (due to ret.WriteString(prefix)).

		// No := to define prefix and found, because that would create a local temporary suffix variable, shadowing the one from the outer scope.

		var prefix string
		var found bool
		// look for first % appearing and split according to that.
		prefix, suffix, found = strings.Cut(suffix, `%`)

		// everything before the first % can just go to output; for the rest (now in suffix), we need to actually do some parsing
		if output {
			ret.WriteString(prefix)
		}

		if !found {
			// We are guaranteed suffix == "" at this point and we are done.
			return
		}

		// Trailing % in format string, not part of %% - escape.
		if len(suffix) == 0 {
			err = fmt.Errorf("invalid terminating \"%%\"")
			return
		}

		// handle %c - cases where c is a single rune not followed by {param}

		if output {
			switch suffix[0] {
			case '%': // Handle %% - escape for literal "%"
				suffix = suffix[1:]
				ret.WriteRune('%')
				continue
			case 'm': // Handle %m - print map of all parameters
				suffix = suffix[1:]
				_, err = fmt.Fprint(&ret, parameters)
				if err != nil {
					return
				}
				continue
			case 'w': // handle %w - print wrapped error
				suffix = suffix[1:]
				ret.WriteString(baseError.Error())
				continue
			// case '!' handled later
			default:
				// Do nothing
			}
		} else { // In the output == false-case, we just consume %%, %m, %w
			switch suffix[0] {
			case '%', 'm', 'w':
				suffix = suffix[1:]
				continue
			}
		}

		// Everything else must be of the form %fmtString{parameterName}remainingString

		// Get fmtString
		var fmtString string
		fmtString, suffix, found = strings.Cut(suffix, "{")
		if !found {
			err = fmt.Errorf("remaining error override string %%%v, which is missing mandatory {...}-brackets", fmtString)
			return
		}

		// handle case where fmtString contains another %. This is not allowed (probably caused by some missing '{' ) and would cause strange errors later when we pass
		// %fmtString to a ftm.[*]Printf - variant.
		if strings.ContainsRune(fmtString, '%') {
			err = fmt.Errorf("invalid format string: Must be %%fmtString{parameterName}, parsed fmtString as %v, which contains another %%", fmtString)
			return
		}

		// Get parameterName
		var parameterName string
		parameterName, suffix, found = strings.Cut(suffix, "}")
		if !found {
			err = fmt.Errorf("error override message contained format string %v, which is missing terminating \"}\">", parameterName)
			return
		}

		// If we don't actually output anything, we have no way of checking validity.
		// (due to the possibility of custom format verbs)
		if !output {
			continue
		}

		// handle special case "!"
		if fmtString == nonEmptyMapFormatString {
			if len(parameters) > 0 {
				var recursiveParseResult string
				recursiveParseResult, err = formatError(parameterName, parameters, baseError, output)
				if output {
					ret.WriteString(recursiveParseResult)
				}
				if err != nil {
					return
				}
			}
			continue
		}

		// default to %v
		if fmtString == "" {
			fmtString = "v"
		}

		// actually print the parameter
		_, err = fmt.Fprintf(&ret, "%"+fmtString, parameters[parameterName])
		if err != nil {
			return
		}

		continue // redundant
	}
}

type ErrorInterpolater interface {
	error
	Error_interpolate(map[string]any) string
}

type interpolationMode int

const (
	full_eval    interpolationMode = 0
	partial_eval                   = iota
	parse_check                    = iota
	param_check                    = iota
)

var escaper *strings.Replacer = strings.NewReplacer("%", "%%", "$", "%$")

// TODO: Doc

// FormatError will print/interpolate the given format string using the parameters map if output == true
// For output == false, it will only do some validity parsing checks.
// This is done in one function in order to de-duplicate code.
//
// The format is as follows: %FMT{arg} is printed like fmt.Printf(%FMT, parameters[arg])
// %% is used to escape literal %
// %m is used to print the parameters map itself
// %$nonEmptyMapFormatString{STR} will evaluate STR if len(parameters) > 0, nothing otherwise.
// %w will print baseError.Error()
func formatError_new(formatString string, parameters_own map[string]any, parameters_passed map[string]any, baseError error, mode interpolationMode) (returned_string string, err error) {
	if !utf8.ValidString(formatString) {
		panic(ErrorPrefix + "error message subject to interpolation was not a valid UTF-8 string")
	}

	var output bool = (mode == full_eval || mode == partial_eval)

	// We build up the returned string piece-by-piece by writing to ret.
	// This avoids some allocations & copying
	var ret strings.Builder
	if output {
		defer func() {
			returned_string = ret.String()
			if err != nil {
				returned_string += fmt.Sprintf("<error when printing error: %v>", err)
			}
		}()
	}

	var remaining_string string = formatString // holds the remaining yet-unprocessed part of the input.

	for { // annoying to write as a "usual" for loop, because both init and iteration would need to consist of 2 lines (due to ret.WriteString(prefix)).

		var passmap bool // bool indicating whether we found a $ (passmap == true) or a % (passmap == false)
		// look for first % or $ appearing and split according to that.
		pos := strings.IndexAny(remaining_string, "%$")
		if pos == -1 { // No % or $ in the string, we just plain write
			if output {
				ret.WriteString(remaining_string)
			}
			return
		}

		switch remaining_string[pos] {
		case '%':
			passmap = false
		case '$':
			passmap = true
		default:
			panic(ErrorPrefix + "cannot happen")
		}
		// remaining_string has the form <sth>[$%]<sth>

		// Ensure that the first $ or % is not the last character:
		// For such a trailing % or $ in formatString, which is not part of %% or %$- escape, we print this character as-is and report an error.
		if len(remaining_string) == pos+1 {
			if output {
				ret.WriteString(remaining_string)
			}
			if passmap {
				err = fmt.Errorf("invalid terminating \"$\"")
			} else {
				err = fmt.Errorf("invalid terminating \"%%\"")
			}
			return
		}

		// everything before the first % or $ can just go to output; for the rest, we need to actually do some parsing
		if output {
			ret.WriteString(remaining_string[0:pos])
		}
		remaining_string = remaining_string[pos+1:]

		// Note: Remaining string now contains everything after the % or $, excluding the original % or $ itself.

		// handle cases of the form %c where we do not require the verb to be followed by {args}. In each of these cases, c is a single character (by design).
		if passmap { // $c
			switch remaining_string[0] {
			// $% and $$ are invalid

			case '%':
				err = fmt.Errorf("error interpolation string contained invalid escape $%%")
				return
			case '$':
				err = fmt.Errorf("error interpolation string contained invalid escape sequence $$")
			case 'w': // $w : print base error with current parameters
				remaining_string = remaining_string[1:]      // remove the "w"
				if baseError == nil && mode != parse_check { // $w refers to the wrapped error, but there is no wrapped error.
					err = fmt.Errorf("invalid format verb $w: error does not wrap anything")
					return
				}
				baseError_interpolate, baseErrorOK := baseError.(ErrorInterpolater)
				if !baseErrorOK {
					err = fmt.Errorf("invalid format verb $w: wrapped base error does not support this")
					return
				}
				switch mode {
				case partial_eval:
					ret.WriteString("$w")
				case full_eval:
					ret.WriteString(baseError_interpolate.Error_interpolate(parameters_passed))
				case param_check, parse_check:
					// do nothing
				default:
					panic(ErrorPrefix + "unhandled case")
				}
				continue
			case 'm': // print map (with inherited params)
				remaining_string = remaining_string[1:]
				switch mode {
				case partial_eval:
					ret.WriteString("$m")
				case full_eval:
					_, err = fmt.Fprint(&ret, parameters_passed) // print parameters map using fmt.Fprint
					if err != nil {
						return
					}
				case param_check, parse_check:
					// do nothing
				default:
					panic(ErrorPrefix + "unhandled case")
				}
				continue
			default:
				// do nothing. $c with c not among %,$, w, m is handled below
			}
		} else { // %c rather than $c
			switch escapedChar := remaining_string[0]; escapedChar {
			case '%', '$':
				switch mode {
				case partial_eval:
					ret.WriteByte('%')
					ret.WriteByte(escapedChar)
				case full_eval:
					ret.WriteByte(escapedChar)
				case param_check, parse_check:
					// do nothing
				default:
					panic(ErrorPrefix + "unhandled case")
				}
				remaining_string = remaining_string[1:]
				continue
			case 'w':
				remaining_string = remaining_string[1:]
				if baseError == nil && mode != parse_check {
					err = fmt.Errorf("invalid format verb %%w: no wrapped error")
					return
				}
				switch mode {
				case partial_eval:
					ret.WriteString(escaper.Replace(baseError.Error()))
				case full_eval:
					ret.WriteString(baseError.Error())
				case param_check, parse_check:
					// do nothing
				default:
					panic(ErrorPrefix + "unhandled case")
				}
				continue
			case 'm':
				remaining_string = remaining_string[1:]
				switch mode {
				case partial_eval:
					ret.WriteString(escaper.Replace(fmt.Sprint(parameters_own)))
				case full_eval:
					_, err = fmt.Fprint(&ret, parameters_own)
					if err != nil {
						return
					}
				case param_check, parse_check:
					// do nothing
				default:
					panic(ErrorPrefix + "unhandled case")
				}
				continue
			}
		}

		// If we get here, we had read something of the form $<foo> or %<foo> where foo does not start with %,$, m, w
		// remaining_string is everything after the $ or %
		// In this case, foo must be of the form fmtString{arg}tail

		// Get fmtString
		var fmtString string
		var found bool
		fmtString, remaining_string, found = strings.Cut(remaining_string, "{")
		if !found {
			err = fmt.Errorf("remaining error override string %%%v, which is missing mandatory {...}-brackets", fmtString)
			return
		}

		// handle case where fmtString contains another %. This is not allowed (probably caused by some missing '{' ) and would cause strange errors later when we pass
		// %fmtString to a ftm.[*]Printf - variant.
		if strings.ContainsRune(fmtString, '%') {
			err = fmt.Errorf("invalid format string: Must be %%fmtString{parameterName}, parsed %v for fmtString, which contains another literal \"%%\"", fmtString)
			return
		}

		// Get parameterName
		var parameterName string
		parameterName, remaining_string, found = strings.Cut(remaining_string, "}")
		if !found {
			err = fmt.Errorf("error override message contained format string %v, which is missing terminating \"}\">", parameterName)
			return
		}

		// handle special cases starting with "!"
		if fmtString == nonEmptyMapFormatString {
			switch mode {
			case param_check, parse_check:
				_, err = formatError_new(parameterName, parameters_own, parameters_passed, baseError, mode)
				if err != nil {
					return
				}
			case full_eval:
				if ((!passmap) && len(parameters_own) > 0) || (passmap && len(parameters_passed) > 0) {
					var recursiveParseResult string
					recursiveParseResult, err = formatError_new(parameterName, parameters_own, parameters_passed, baseError, mode)
					if err != nil {
						return
					}
					ret.WriteString(recursiveParseResult)
				}
			case partial_eval:
				if (!passmap) && len(parameters_own) > 0 {
					var recursiveParseResult string
					recursiveParseResult, err = formatError_new(parameterName, parameters_own, parameters_passed, baseError, mode)
					if err != nil {
						return
					}
					ret.WriteString(recursiveParseResult)
				} else if passmap { // $!M>0{...} and we don't know if we should eval or not.
					_, err = formatError_new(parameterName, parameters_own, parameters_passed, baseError, parse_check)
					if err != nil {
						return
					}
					ret.WriteByte('$')
					ret.WriteString(nonEmptyMapFormatString)
					ret.WriteByte('{')
					ret.WriteString(parameterName)
					ret.WriteByte('}')
				} else { // passmap == false, len(parameters_own) == 0. We just drop the argument. However, we still check that it parses.
					_, err = formatError_new(parameterName, parameters_own, parameters_passed, baseError, parse_check)
					if err != nil {
						return
					}
				}
			default:
				panic(ErrorPrefix + "unhandled case")
			}
			continue
		}

		// default to %v or $v. This also guarantees that len(fmtString) > 0
		if fmtString == "" {
			fmtString = "v"
		}

		if !ast.IsExported(parameterName) {
			err = fmt.Errorf("invalid argument: \"%v\" is not a valid exported field name", parameterName)
			return
		}

		// %fmtString{parameterName} or $fmtString{parameterName}
		// This means we intend to formate parameter[paramterName] using format string %fmtString via fmt.Fprintf.
		// NOTE: parameter is either parameters_passed or parameters_own, depending on passmap.
		switch mode {
		case full_eval: // We actually format parameterName using fmtString
			var paramVal any
			var paramFound bool
			if passmap {
				paramVal, paramFound = parameters_passed[parameterName]
			} else {
				paramVal, paramFound = parameters_own[parameterName]
			}
			if !paramFound {
				ret.WriteString("<MISSING ARGUMENT>")
				err = fmt.Errorf("missing argument: %v", parameterName)
				return
			}
			_, err = fmt.Fprintf(&ret, "%"+fmtString, paramVal)
			if err != nil {
				return
			}
		case partial_eval:
			if passmap {
				ret.WriteByte('$')
				ret.WriteString(fmtString)
				ret.WriteByte('{')
				ret.WriteString(parameterName)
				ret.WriteByte('}')
			} else {
				paramVal, paramFound := parameters_own[parameterName]
				if !paramFound {
					ret.WriteString("<MISSING ARGUMENT>")
					err = fmt.Errorf("missing argument: %v", parameterName)
					return
				}
				ret.WriteString(escaper.Replace(fmt.Sprintf("%"+fmtString, paramVal)))
			}
		case parse_check:
			// do nothing
		case param_check:
			// check if parameter exists (if we can tell)
			if !passmap {
				_, paramFound := parameters_own[parameterName]
				if !paramFound {
					err = fmt.Errorf("missing argument: %v", parameterName)
					return
				}
			}
		default:
			panic(ErrorPrefix + "unhandled case")
		}
		continue // redundant
	} // end of for loop. We terminate (at the latest) if remaining_string does not contain any relevant control characters
}
