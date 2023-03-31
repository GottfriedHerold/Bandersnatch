package fieldElements

import (
	"errors"
	"fmt"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorconsts"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file collects all errors that can be returned by functions in this package.
//
// IMPORTANT: We often return errors wrapping some error given here. Never compare errors for equality. Use [errors.Is]

// ErrorPrefix is the prefix used by all error message strings originating from this package.
const ErrorPrefix = "bandersnatch / field element: "

var (
	noWriteEOF              = errorsWithData.AddDataToError_struct(io.EOF, &errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0})
	noWriteUnexpectedEOF    = errorsWithData.AddDataToError_struct(io.ErrUnexpectedEOF, &errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0})
	emptySliceForByteSer    = errorsWithData.AddDataToError_struct(io.EOF, &errorconsts.NoWriteAttempt)
	tooSmallSliceForByteSer = errorsWithData.AddDataToError_struct(io.ErrUnexpectedEOF, &errorconsts.NoWriteAttempt)
)

func init() {
	errorsWithData.EnsureTestsValid_Final(errPrefixDoesNotFit, noWriteEOF, noWriteUnexpectedEOF)
}

// Base error when ToUint64 or ToInt64 fail. Note that we always return an error wrapping this; for that reason, the error message given here will never occur.
var ErrCannotRepresentFieldElement = errors.New(ErrorPrefix + "field element not representable by the given data type")

var ErrDivisionByZero = errors.New(ErrorPrefix + "division by zero")

// These are the errors that can occur during (de)serialization.
var (
	errPrefixDoesNotFit                   = errorsWithData.NewErrorWithData_struct(nil, ErrorPrefix+"while trying to serialize a field element with a prefix, the prefix did not fit, because the number was too large", &errorconsts.NoWriteAttempt)
	ErrPrefixDoesNotFit                   = errorsWithData.MakeErrorIncomparable(errPrefixDoesNotFit)
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
type MultiInversionError = errorsWithData.ErrorWithData[MultiInversionErrorData]

// Canary: This will panic if we refactor field names. The reason is that some functions below use %v{FieldName} - syntax, which depends on these particular names.
func init() {
	errorsWithData.CheckParameterForStruct[MultiInversionErrorData]("ZeroIndices")
	errorsWithData.CheckParameterForStruct[MultiInversionErrorData]("NumberOfZeroIndices")
}

// GenerateMultiDivisionByZeroError creates an error indicating which of the provided field elements were zero. This is used to create errors for the Multi-Inversion functions.
// prefixForErrors is prefixed to the error string created.
// If none of the fieldElements are zero, returns nil
//
// NOTE: This is an internal function that is exported for cross-package usage.
func GenerateMultiDivisionByZeroError(fieldElements []*bsFieldElement_MontgomeryNonUnique, prefixForError string) errorsWithData.ErrorWithData[MultiInversionErrorData] {
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

	return errorsWithData.NewErrorWithData_struct(ErrDivisionByZero, errorString, &errorData)
}
