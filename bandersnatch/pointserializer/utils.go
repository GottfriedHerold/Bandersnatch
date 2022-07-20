package pointserializer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/bandersnatchErrors"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

const PARTIAL_READ = bandersnatchErrors.PARTIAL_READ
const PARTIAL_WRITE = bandersnatchErrors.PARTIAL_WRITE

const errordata_ACTUALLYREAD = "ActuallyRead"

// additional data contained in errors returned by consumeExpectRead
type headerRead struct {
	PartialRead    bool
	ActuallyRead   []byte
	ExpectedToRead []byte
	BytesRead      int
}

// PARTIAL_READ and errordata_ACTUALLYREAD must coincide with field names.
// This is just to guard against refactorings renaming
func init() {
	fields := reflect.VisibleFields(utils.TypeOfType[headerRead]())
	if fields[0].Name != PARTIAL_READ {
		panic(0)
	}
	if fields[1].Name != errordata_ACTUALLYREAD {
		panic(1)
	}
}

const ErrorPrefix = "bandersnatch / serialization: "

var ErrDidNotReadExpectedString = bandersnatchErrors.ErrDidNotReadExpectedString

// consumeExpectRead reads and consumes len(expectToRead) bytes from input and reports an error if the read bytes differ from expectToRead.
// This is intended to read headers. Remember to use errors.Is to check the returned errors rather than == due to error wrapping.
//
// NOTES:
// Returns an error wrapping io.ErrUnexpectedEOF or io.EOF on end-of-file (io.EOF if the io.Reader was in EOF state to start with, io.ErrUnexpectedEOF if we encounter EOF after reading >0 bytes)
// On mismatch of expectToRead vs. actually read values, returns an error wrapping ErrDidNotReadExpectedString
//
// Panics if expectToRead has length >MaxInt32. The function always (tries to) consume len(expectToRead) bytes, even if a mismatch is already early in the stream.
// Panics if expectToRead is nil or input is nil (unless len(expectToRead)==0)
//
// The returned error type satisfies the error interface and, if non-nil, contains an instance of headerRead,
// ActuallyRead contains the actually read bytes (type []byte)
// PartialRead (type bool) is true iff 0 < bytes_read < len(expectToRead).
// Note here that if bytesRead == len(expectToRead), io errors are dropped and the only possible error is ErrDidNotReadExpectedString.
//
// Possible errors (modulo wrapping):
// io errors, io.EOF, io.ErrUnexpectedEOF, ErrDidNotReadExpectedString
func consumeExpectRead(input io.Reader, expectToRead []byte) (bytes_read int, returnedError errorsWithData.ErrorWithGuaranteedParameters[headerRead]) {
	// We do not treat nil as an empyt byte slice here. This is an internal function and we expect ourselves to behave properly: nil indicates a bug.
	if expectToRead == nil {
		panic(ErrorPrefix + "consumeExpectRead called with nil input for expectToRead")
	}
	l := len(expectToRead)
	if l > math.MaxInt32 {
		// should we return an error instead of panicking?
		panic(fmt.Errorf(ErrorPrefix+"trying to read from io.Reader, expecting to read %v bytes, which is more than MaxInt32", l))
	}
	if l == 0 {
		return 0, nil
	}
	if input == nil {
		panic(ErrorPrefix + "consumeExpectRead was called on nil reader")
	}
	var err error
	var buf []byte = make([]byte, l)
	bytes_read, err = io.ReadFull(input, buf)
	if err != nil {
		buf = buf[0:bytes_read:bytes_read] // We reduce the cap, so maybe some future version of Go actually frees the trailing memory. (We *could* copy it to a new buffer, but that's probably worse in most cases)

		// Note: We deep-copy the contents of expectToRead. The reason is that the caller might later modify the backing array otherwise.
		var returnedErrorData headerRead = headerRead{ActuallyRead: buf, ExpectedToRead: copyByteSlice(expectToRead), BytesRead: bytes_read} // extra data returned in error

		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Note: Sprintf is only used for the length of the expected input. The other %%v{arg} are done via errorsWithData, hence escaping the %.
			message := fmt.Sprintf(ErrorPrefix+"Unexpected EOF after reading %%v{BytesRead} out of %v bytes when reading header.\nReported error was %%w.\nBytes expected were 0x%%x{ExpectedToRead}, got 0x%%x{ActuallyRead}", len(expectToRead))
			returnedErrorData.PartialRead = true
			returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
			return
		} else if errors.Is(err, io.EOF) {
			message := ErrorPrefix + "EOF when trying to read buffer.\nExpected to read 0x%x{ExpectedToRead}, got EOF instead"
			if bytes_read != 0 {
				panic("Cannot happen")
			}
			returnedErrorData.ActuallyRead = make([]byte, 0) // no need to extend the lifetime of buf's underlying array.
			returnedErrorData.PartialRead = false

			returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
			return
		} else { // io error
			returnedErrorData.PartialRead = (bytes_read > 0) // NOTE: io.ReadFull guarantees bytes_read < l
			returnedError = errorsWithData.NewErrorWithParametersFromData(err, "", &returnedErrorData)
			return
		}

	}
	if !bytes.Equal(expectToRead, buf) {
		// Note: We deep-copy the contents of expectToRead. The reason is that the caller might later modify the backing array otherwise.
		var returnedErrorData headerRead = headerRead{ActuallyRead: buf, ExpectedToRead: copyByteSlice(expectToRead), BytesRead: bytes_read, PartialRead: false} // extra data returned in error

		err = bandersnatchErrors.ErrDidNotReadExpectedString
		message := ErrorPrefix + "Unexpected Header encountered upon deserialization. Expected 0x%x{ExpectedToRead}, got 0x%x{ActuallyRead}"
		returnedError = errorsWithData.NewErrorWithParametersFromData(err, message, &returnedErrorData)
		return
	}
	returnedError = nil // this is true anyway at this point, we like being explicit.
	return
}

// Note: This returns a copy (by design). For v==nil, we return a fresh, empty non-nil slice.

// copyByteSlice returns a copy of the given byte slice (with newly allocated underlying array).
// For nil inputs, returns an empty byte slice.
func copyByteSlice(v []byte) (ret []byte) {
	if v == nil {
		ret = make([]byte, 0)
		return
	}
	ret = make([]byte, len(v))
	L := copy(ret, v)
	testutils.Assert(L == len(v))
	return
}

// writeFull(output, data) wraps around output.Write(data) by adding error data.
func writeFull(output io.Writer, data []byte) (bytesWritten int, err bandersnatchErrors.SerializationError) {
	bytesWritten, errPlain := output.Write(data)
	if errPlain != nil {
		errPlain = errorsWithData.NewErrorWithGuaranteedParameters[bandersnatchErrors.WriteErrorData](errPlain, "Error %w occured when trying to write %v{Data} to io.Writer. We only wrote %v{BytesWritten} data.",
			"Data", copyByteSlice(data),
			"BytesWritten", bytesWritten,
			"PartialWrite", bytesWritten != 0 && bytesWritten < len(data),
		)
	}
	return
}
