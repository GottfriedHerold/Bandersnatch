package fieldElements

import (
	"errors"
	"fmt"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// ErrorPrefix is the prefix used by all error message strings originating from this package.
const ErrorPrefix = "bandersnatch / field element: "

var ErrCannotRepresentAsUInt64 = errors.New(ErrorPrefix + "cannot represent field element as a uint64")
var ErrDivisionByZero = errors.New(ErrorPrefix + "division by zero")

// These are the errors that can occur during (de)serialization.
var (
	ErrPrefixDoesNotFit             error = errors.New(ErrorPrefix + "while trying to serialize a field element with a prefix, the prefix did not fit, because the number was too large")
	ErrPrefixLengthInvalid          error = errors.New(ErrorPrefix + "in FieldElement deserializitation, an invalid prefix length > 8 was requested")
	ErrPrefixMismatch               error = errors.New(ErrorPrefix + "during deserialization, the read prefix did not match the expected one")
	ErrNonNormalizedDeserialization error = errors.New(ErrorPrefix + "during FieldElement deserialization, the read number was not the minimal representative modulo BaseFieldSize")
)

// MultiInversionErrorData is the struct type that holds the additional information
// if a Multi-Inversion of field elements goes wrong due to division by zero.
//
// In this case, the returned error satisfies the errorsWithData.ErrorWithGuaranteedParameters[MultiInversionErrorData] interface.
// in particular, the returned error has a method with signature GetData() MultiInversionErrorData.
type MultiInversionErrorData struct {
	ZeroIndices         []int
	NumberOfZeroIndices int
}

// MultiInversionError is an interface extending error.
// It is used to indicate errors in multiinversion algorithms.
type MultiInversionError = errorsWithData.ErrorWithGuaranteedParameters[MultiInversionErrorData]

func generateMultiDivisionByZeroError(fieldElements []*bsFieldElement_64, prefixForError string) errorsWithData.ErrorWithGuaranteedParameters[MultiInversionErrorData] {
	var errorData MultiInversionErrorData
	errorData.ZeroIndices = make([]int, 0)
	for i, fe := range fieldElements {
		if fe.IsZero() {
			errorData.NumberOfZeroIndices++
			errorData.ZeroIndices = append(errorData.ZeroIndices, i)
		}
	}
	if errorData.NumberOfZeroIndices == 0 {
		return nil
	}

	if len(errorData.ZeroIndices) != errorData.NumberOfZeroIndices {
		panic(ErrorPrefix + " internal error: number of zero indices and lenght of corresponding slice differ. This is not supposed to be possible")
	}

	var errorString string

	// Format error message depending on the number of zeros encountered.
	if errorData.NumberOfZeroIndices == 1 {
		errorString = fmt.Sprintf("%v\nThe %v'th argument (counting from 0) was the only one that was zero.", prefixForError, errorData.ZeroIndices[0])
	} else if errorData.NumberOfZeroIndices < 10 {
		errorString = prefixForError + "\nThere were %v{NumberOfZeroIndices} many arguments that were zero: Those were given at indices (starting from 0) %v{ZeroIndices}."
	} else {
		// Note: %%v becomes %v, which is handled by errorsWithData's processing.
		errorString = fmt.Sprintf("%v\nThere were %%v{NumberOfZeroIndices} many arguments that were zero. The first ten were at indices (starting from 0) %v", prefixForError, errorData.ZeroIndices[0:10])
	}

	return errorsWithData.NewErrorWithParametersFromData(ErrDivisionByZero, errorString, &errorData)
}
