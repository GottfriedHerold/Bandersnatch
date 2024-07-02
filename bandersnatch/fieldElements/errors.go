package fieldElements

import (
	"errors"
	"fmt"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/errorconsts"
)

// This file is part of the fieldElements package. See the documentation of field_element.go for general remarks.

// This file collects all errors that can be returned by functions in this package.
//
// IMPORTANT: We often return errors wrapping some error given here. Never compare errors for equality. Use [errors.Is]

// ErrorPrefix is the prefix used by all error message strings originating from this package.
const ErrorPrefix = "bandersnatch / field element: "

/*
var errNoWriteEOF, _ = errorsWithData.NewErrorWithData_struct(io.EOF, "",
	&errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
	errorsWithData.PanicOnAllMistakes)
*/

var (
	errNoWriteUnexpectedEOF, _ = errorsWithData.NewErrorWithData_struct(io.ErrUnexpectedEOF, "", &errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true}, errorsWithData.PanicOnAllMistakes)
	// emptySliceForByteSer, _    = errorsWithData.NewErrorWithData_struct(io.EOF, "", &errorconsts.NoWriteAttempt, errorsWithData.PanicOnAllMistakes)
	// tooSmallSliceForByteSer, _ = errorsWithData.NewErrorWithData_struct(io.ErrUnexpectedEOF, "", &errorconsts.NoWriteAttempt, errorsWithData.PanicOnAllMistakes)
)

// TODO: Doc

// NOTE: $v{ValueType} may be a reflect.Type or a string -- we actually use both.

var (
	errEmptyBytesSlice, _ = errorsWithData.NewErrorWithData_any_params(io.EOF,
		"Called (de)serializion method or function on empty or nil slice", // NOTE: This is never used for ouput. We use either the serialization or the deserialization version below.
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	errEmptyByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
		ErrorPrefix+"Trying to serialize a $T{Value} with value $v{Value} into $! NilSlice != 0{a nil}$! NilSlice == 0{an empty} slice",
		&errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errEmptyByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
		ErrorPrefix+"Trying to deserialize a $v{ValueType} from $! NilSlice != 0{a nil}$! NilSlice == 0 {an empty} slice}",
		&errorconsts.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	errTooSmallByteSlice, _ = errorsWithData.NewErrorWithData_any_params(io.EOF,
		ErrorPrefix+"Called (de)serializion method or function on too small slice", // NOTE: This is never used for ouput. We use either the serialization or the deserialization version below.
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	errTooSmallByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
		ErrorPrefix+"Trying to serialize a $T{Value} with value $v{Value} into a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
		&errorconsts.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	errTooSmallByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
		ErrorPrefix+"Tryting to deserialize a $v{ValueType} from a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
		&errorconsts.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	errPrefixDoesNotFit, _ = errorsWithData.NewErrorWithData_struct(nil,
		ErrorPrefix+"while trying to serialize a $!ValueType{$v{ValueType}}$! !ValueType{$T{Value}} with value $v{Value} with a prefix, the prefix of length $v{PrefixLenght} did not fit, because the number was too large, having only $v{LeadingZeroes} leading zeros",
		&errorconsts.NoWriteAttempt, errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	ErrEmptyByteSlice    = errorsWithData.BoxErrorAsIncomparable(errEmptyBytesSlice)
	ErrTooSmallByteSlice = errorsWithData.BoxErrorAsIncomparable(errTooSmallByteSlice)
	ErrPrefixDoesNotFit  = errorsWithData.BoxErrorAsIncomparable(errPrefixDoesNotFit)
)

/*
func init() {
	errorsWithData.EnsureErrorsValid_Final(errPrefixDoesNotFit, errNoWriteEOF, errNoWriteUnexpectedEOF)
}
*/

// Base error when ToUint64 or ToInt64 fail. Note that we always return an error wrapping this; for that reason, the error message given here will never occur.
var ErrCannotRepresentFieldElement = errors.New(ErrorPrefix + "field element not representable by the given data type")

var ErrDivisionByZero = errors.New(ErrorPrefix + "division by zero")

// The error strings below assert common.MaxLengthPrefixBits == 8. This is a refactoring guard.
var _ = func() int {
	if common.MaxLengthPrefixBits != 8 {
		panic("Need to change errors below")
	}
	return 0
}()

// These are the errors that can occur during (de)serialization.
var (
	errPrefixLengthInvalid, _ = errorsWithData.NewErrorWithData_any_params(nil,
		ErrorPrefix+"in FieldElement (de)serializitation, an invalid prefix length ${PrefixLength} > 8 was requested",
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errPrefixLengthInvalid_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errPrefixLengthInvalid,
		ErrorPrefix+"When deserializing a $v{ValueType}, an invalid prefix length ${PrefixLength} > 8 was requested",
		&errorconsts.NoReadAttempt,
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	ErrPrefixLengthInvalid = errorsWithData.BoxErrorAsIncomparable(errPrefixLengthInvalid)

	errPrefixMismatch, _ error = errorsWithData.NewErrorWithData_any_params(nil,
		ErrorPrefix+"during deserialization, the read prefix 0b$b{Prefix} did not match the expected one 0b$b{ExpectedPrefix}",
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

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

	ret, _ := errorsWithData.NewErrorWithData_struct(ErrDivisionByZero, errorString, &errorData)

	return ret
}
