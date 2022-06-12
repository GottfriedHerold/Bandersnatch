package errorsWithData

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TODO: Global Renaming

// errorWithParameters_common is a simple implementation of the ErrorWithParameters interface
// NOTE:
// functions must ALWAYS return an error as an interface, never as a concrete type.
// (since otherwise, nil errors are returned as typed nil pointers, which is a serious footgun)
type errorWithParameters_common struct {
	contained_error error          // wrapped underlying error
	message         string         // if not "", overrides the error string.
	params          map[string]any // map strings -> data.
	// showparams      bool           // should we show embedded data on error
}

// extension of errorWithParameters_common that satisfies ErrorWithParameters[StructType]
type errorWithParameters_T[StructType any] struct {
	errorWithParameters_common
}

// Error is provided to satisfy the error interface
func (e *errorWithParameters_common) Error() string {
	if e == nil {
		panic(errorPrefix + "called Error() on nil error of concrete type errorWithParams. This is a bug, since nil errors of this type should never exist.")
	}
	s, formattingError := formatError(e.message, e.params, e.contained_error, true)
	if formattingError != nil {
		// TODO: Callback?
		panic(formattingError)
	}
	return s

}

// Unwrap is provided to work with errors.Is
func (e *errorWithParameters_common) Unwrap() error {
	// Note: This panics if e is nil (of type *errorWithParams).
	// While returnining untyped nil would give "meaningful" behaviour
	// (including for the recursive calls in HasParameter etc.),
	// we consider any nil pointer of concrete error type a bug.
	return e.contained_error
}

// GetData is provided to satisfy ErrorWithParameters[StructType].
//
// It constructs a value of type StructType from the provided parameters.
func (e *errorWithParameters_T[StructType]) GetData() (ret StructType) {
	ret, err := makeStructFromMap[StructType](e.params)
	if err != nil {
		panic(err)
	}
	return
}

// HasParameter checks whether the parameter given by the name is present.
func (e *errorWithParameters_common) HasParameter(parameterName string) bool {
	_, ok := e.params[parameterName]
	return ok
}

// GetParameter retrieves the parameter stored under the key parameterName and whether it was present.
//
// On keys that were not present, returns nil, false.
func (e *errorWithParameters_common) GetParameter(parameterName string) (value any, present bool) {
	value, present = e.params[parameterName]
	return
}

// GetAllParameters returns a map of all parameters present in the error.
// The returned map is a (shallow) copy, so modification of values of the returned map does not affect the error.
func (e *errorWithParameters_common) GetAllParameters() (ret map[string]any) {
	ret = make(map[string]any)
	for key, value := range e.params {
		ret[key] = value
	}
	return
}

// TODO: Syntax-check the override message?

func makeErrorWithParametersCommon(baseError error, overrideMessage string) (ret errorWithParameters_common) {
	if !utf8.ValidString(overrideMessage) {
		panic(errorPrefix + "override message for error creation was not a valid UTF-8 string")
	}
	if overrideMessage == "" {
		overrideMessage = DefaultOverrideMessage
	} else if overrideMessage == OverrideByEmptyMessage {
		overrideMessage = ""
	}
	_, formattingError := formatError(overrideMessage, nil, nil, false)
	if formattingError != nil {
		panic(fmt.Errorf(errorPrefix+"creating of an error with parameters failed, because the error override message was malformed.\noverrideMessage = %v.\nreported error was: %v", overrideMessage, formattingError))
	}
	ret.contained_error = baseError
	ret.message = overrideMessage
	ret.params = GetAllParametersFromError(baseError)
	return
}

// Providing this value as overrideMessage for creating an ErrorWithParameters will create an actual empty string
// (Giving a literal empty string will instead default to the string of the wrapped error.)
const OverrideByEmptyMessage = "%Empty"
const DefaultOverrideMessage = "%w"

// FormatError will print/interpolate the given format string using the parameters map if output == true
// For output == false, it will only do some validity parsing checks.
// This is done in one function in order to de-duplicate code.
//
// The format is as follows: %FMT{arg} is printed like fmt.Printf(%FMT, parameters[arg])
// %% is used to escape literal %
// %m is used to print the parameters map itself
// %!{STR} will print STR if len(parameters) > 0, nothing otherwise.
// %w will print baseError.Error()
func formatError(formatString string, parameters map[string]any, baseError error, output bool) (returned_string string, err error) {
	if !utf8.ValidString(formatString) {
		panic(errorPrefix + "formatString not a valid UTF-8 string")
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
		var prefix string
		var found bool
		// No :=, because that would create a local temporary suffix variable, shadowing the one from the outer scope.

		// look for first % appearing and split according to that.
		prefix, suffix, found = strings.Cut(suffix, `%`)

		// everything before the first % can just go to output; for the rest (now in suffix, we need to actually do some parsing)
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
		} else {
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
			err = fmt.Errorf("invalid format string: Must be %%fmtString{parameterName}, parsed %v for fmtString, which contains another %%", fmtString)
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
		if fmtString == "!" {
			if len(parameters) > 0 {
				ret.WriteString(parameterName)
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
