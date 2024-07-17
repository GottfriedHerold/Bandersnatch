package fieldElements

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"

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

// NOTE: $v{ValueType} may be a reflect.Type or a string -- we actually use both.

/*
var errNoWriteEOF, _ = errorsWithData.NewErrorWithData_struct(io.EOF, "",
	&common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
	errorsWithData.PanicOnAllMistakes)
*/

var (
	errNoWriteUnexpectedEOF, _ = errorsWithData.NewErrorWithData_struct(io.ErrUnexpectedEOF, "", &common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true}, errorsWithData.PanicOnAllMistakes)
	// emptySliceForByteSer, _    = errorsWithData.NewErrorWithData_struct(io.EOF, "", &errorconsts.NoWriteAttempt, errorsWithData.PanicOnAllMistakes)
	// tooSmallSliceForByteSer, _ = errorsWithData.NewErrorWithData_struct(io.ErrUnexpectedEOF, "", &errorconsts.NoWriteAttempt, errorsWithData.PanicOnAllMistakes)
)

// ErrTooSmallByteSlice and ErrEmptyByteSlice are the errors reported when trying to use variants of Serialize_*_Bytes or Deserialize_*_Bytes on too small/nil/empty byte slices.
var (
	errTooSmallByteSlice, _ = errorsWithData.NewErrorWithData_any_params(io.ErrUnexpectedEOF,
		ErrorPrefix+"Called (de)serializion method or function on too small slice", // NOTE: This should never be used directly. We use either the serialization or the deserialization version below.
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errTooSmallByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
		ErrorPrefix+"Trying to serialize a $T{Value} with value $v{Value} into a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
		&common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errTooSmallByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
		ErrorPrefix+"Trying to deserialize a $v{ValueType} from a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
		&common.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	// errTooSmallByteSlice, _ = errorsWithData.NewErrorWithData_any_params(io.ErrUnexpectedEOF,
	// 	ErrorPrefix+"Called (de)serializion method or function on too small slice", // NOTE: This should never be used directly. We use either the serialization or the deserialization version below.
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	// errTooSmallByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
	// 	ErrorPrefix+"Trying to serialize a $T{Value} with value $v{Value} into a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
	// 	&common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	// errTooSmallByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errTooSmallByteSlice,
	// 	ErrorPrefix+"Trying to deserialize a $v{ValueType} from a slice of insufficient size $v{SliceSize} instead of the required $v{RequiredSize}",
	// 	&common.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	ErrTooSmallByteSlice = errorsWithData.BoxErrorAsIncomparable(errTooSmallByteSlice)

	errEmptyBytesSlice, _ = errorsWithData.NewErrorWithData_any_params(io.EOF,
		"Called (de)serializion method or function on empty or nil slice", // NOTE: This is never used for ouput. We use either the serialization or the deserialization version below.
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errEmptyByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
		ErrorPrefix+"Trying to serialize a $T{Value} with value ${Value} into $! NilSlice != 0{a nil}$! NilSlice == 0{an empty} slice",
		&common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errEmptyByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
		ErrorPrefix+"Trying to deserialize a ${ValueType} from $! NilSlice != 0{a nil}$! NilSlice == 0 {an empty} slice",
		&common.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	// errEmptyBytesSlice, _ = errorsWithData.NewErrorWithData_any_params(io.EOF,
	// 	"Called (de)serializion method or function on empty or nil slice", // NOTE: This is never used for ouput. We use either the serialization or the deserialization version below.
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	// errEmptyByteSlice_Serialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
	// 	ErrorPrefix+"Trying to serialize a $T{Value} with value ${Value} into $! NilSlice != 0{a nil}$! NilSlice == 0{an empty} slice",
	// 	&common.WriteErrorData{PartialWrite: false, BytesWritten: 0, IoError: true},
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	// errEmptyByteSlice_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errEmptyBytesSlice,
	// 	ErrorPrefix+"Trying to deserialize a ${ValueType} from $! NilSlice != 0{a nil}$! NilSlice == 0 {an empty} slice",
	// 	&common.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
	// 	errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	ErrEmptyByteSlice = errorsWithData.BoxErrorAsIncomparable(errEmptyBytesSlice)
)

// handleTooSMallBuffer is the utility functions used to handle I/O errors when deserializing from a bytes.Buffer.
// The only possible error that can happen here is that input's length is too small.
// We mimick the general behaviour of our Deserialization function that take an io.Reader as input:
// We return an error either wrapping [io.EOF] or [io.ErrUnexpectedEOF] and we drain the buffer.
//
// params contains any additional paramters that we include in the returned error that may be call-site specific.
func handleTooSMallBuffer(input *bytes.Buffer, extra_params errorsWithData.ParamMap) (bytesRead int, err common.DeserializationError) {
	inputLen := input.Len()
	bytesRead = inputLen
	var errPlain error
	if inputLen == 0 {
		errPlain = io.EOF
	} else {
		errPlain = io.ErrUnexpectedEOF
	}
	buf := make([]byte, inputLen)
	copy(buf, input.Bytes())
	params := errorsWithData.ParamMap{
		"PartialRead":  bytesRead != 0,
		"BytesRead":    bytesRead,
		"ActuallyRead": buf,
		"IoError":      true,
	}
	if len(extra_params) != 0 {
		maps.Copy(params, extra_params)
	}

	err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](errPlain, "", params)
	input.Reset()
	return
}

func handleTooSmallByteSlice_deserialize(input []byte, extra_params errorsWithData.ParamMap) (bytesRead int, err common.DeserializationError) {
	params := errorsWithData.ParamMap{
		"NilSlice":  input == nil,
		"SliceSize": 0,
	}
	if len(extra_params) > 0 {
		maps.Copy(params, extra_params)
	}

	if len(input) == 0 {

		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](errEmptyByteSlice_Deserialize, "", params)
		return 0, err
	} else {
		err, _ = errorsWithData.NewErrorWithData_map[common.ReadErrorData](errTooSmallByteSlice_Deserialize, "", params)
		return 0, err
	}
}

func handleTooSmallByteSlice_serialize(output []byte, extra_params errorsWithData.ParamMap) (bytesRead int, err common.SerializationError) {
	params := errorsWithData.ParamMap{
		"NilSlice":  output == nil,
		"SliceSize": 0,
	}
	if len(extra_params) > 0 {
		maps.Copy(params, extra_params)
	}

	if len(output) == 0 {

		err, _ = errorsWithData.NewErrorWithData_map[common.WriteErrorData](errEmptyByteSlice_Serialize, "", params)
		return 0, err
	} else {
		err, _ = errorsWithData.NewErrorWithData_map[common.WriteErrorData](errTooSmallByteSlice_Serialize, "", params)
		return 0, err
	}

}

/*
var errTooSmallBufferForDeserialize, _ = errorsWithData.NewErrorWithData_struct[common.ReadErrorData](nil,
	"${ErrorPrefix}Called ${FunctionName} to deserialize a ${ValueType} with a too small bytes.Buffer of lenght ${Len} instead of ${ExpectedLen}",
	&common.ReadErrorData{PartialRead: false, BytesRead: 0, ActuallyRead: nil, IoError: true},
	errorsWithData.ErrorUnlessValidBase, errorsWithData.PanicOnAllMistakes,
)

// ErrTooSmallBufferForDeserialize is the error output by Deserialization methods
var ErrTooSmallBufferForDeserialize = errorsWithData.CreateIncomparableError[common.ReadErrorData](nil,
	"${ErrorPrefix}Called ${FunctionName} to deserialize a ${ValueType} with a too small bytes.Buffer of lenght ${Len} instead of ${ExpectedLen}",
	errorsWithData.ParamMap{
		"PartialRead":  false,
		"BytesRead":    0,
		"ActuallyRead": nil,
		"IoError":      true})

*/

// ErrPrefixDoesNotFit is the error returned by SerializeWithPrefix methods when the prefix does not actually fit.
var ErrPrefixDoesNotFit = errorsWithData.CreateIncomparableError[common.WriteErrorData](nil,
	ErrorPrefix+"while trying to serialize a $!ValueType{$v{ValueType}}$! !ValueType{$T{Value}} with value $v{Value} with a prefix, the prefix of length $v{PrefixLength} did not fit, because the number was too large, having only $v{LeadingZeroes} leading zeroes",
	errorsWithData.ParamMap{
		"ParialWrite":  false,
		"BytesWritten": 0,
		"IoError":      false})

// Base error when ToUint64 or ToInt64 fail. Note that we always return an error wrapping this; for that reason, the error message given here will never occur.
// var ErrCannotRepresentFieldElement = errors.New(ErrorPrefix + "field element not representable by the given data type")
var (
	errCannotRepresentFieldElement, _ = errorsWithData.NewErrorWithData_any_params(nil,
		ErrorPrefix+"field element ${Value} not representable by the given data Type ${DataType}",
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)

	// ErrCannotRepresentFieldElement = errorsWithData.BoxErrorAsIncomparable(errCannotRepresentFieldElement)
)

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
		ErrorPrefix+"during (de)serializitation involving a prefix, an invalid prefix length ${PrefixLength} > 8 was requested",
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	errPrefixLengthInvalid_Deserialize, _ = errorsWithData.NewErrorWithData_struct(errPrefixLengthInvalid,
		ErrorPrefix+"When deserializing a $v{ValueType}, an invalid prefix length ${PrefixLength} > 8 was requested",
		&errorconsts.NoReadAttempt,
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
	ErrPrefixLengthInvalid = errorsWithData.BoxErrorAsIncomparable(errPrefixLengthInvalid)

	ErrPrefixMismatch = errorsWithData.CreateIncomparableError_any(nil,
		ErrorPrefix+"during deserialization, the read prefix 0b$b{Prefix} did not match the expected 0b$b{ExpectedPrefix}", nil)

	// ErrNonNormalizedDeserialization error = errors.New(ErrorPrefix + "during FieldElement deserialization, the read number was not the minimal representative modulo BaseFieldSize")
	errNonNormalizedDeserialization, _ = errorsWithData.NewErrorWithData_any_params(nil,
		ErrorPrefix+"during field element deserialization, the read number ${ReadNumber} was not the minimal representative modulo BaseFieldSize. Reducing modulo BaseFieldSize gives ${ReducedNumber}",
		errorsWithData.PanicOnAllMistakes, errorsWithData.ErrorUnlessValidBase)
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
