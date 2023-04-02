package errorconsts

import (
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// WriteErrorData is a struct holding additional information about Serialization errors. This additional data can be accessed via the errorsWithData package.
type WriteErrorData struct {
	PartialWrite bool // If PartialWrite is true, this indicates that some write operations failed after partially writing something and the io.Writer is consequently in an invalid state.
	BytesWritten int  // BytesWritten indicates the number of bytes that were written by the operation that *caused* the error. NOTE: All Serialization functions accurately return the number of bytes written directly as a non-error parameter. This may differ, because the cause might be in a sub-call.
	IoError      bool // IoError indicates whether the error comes from math or from io.
}

// ReadErrorData is a struct holding additional information about Deserialization errors. This additional data can be accessed via the errorsWithData package.
type ReadErrorData struct {
	PartialRead  bool   // If PartialRead is true, this indicates that after the read error, the io.Reader is believed to be in an invalid state because what was read did not correspond to a complete blob of data that was expected.
	BytesRead    int    // BytesRead indicates the number of bytes that were read by the operation that *caused* the error. NOTE: All Deserialization functions accurately return the number of read bytes directly. The value reported here may differ, because it is the numbe of bytes read in the function that *caused* the error (which may be a sub-call).
	ActuallyRead []byte // this may contain information about data that was read when the error occured. It may be nil, is not guaranteed to be present (even if meaningful) and may be from a sub-call. The reason is that we do not buffer the raw input data, so we cannot provide it in a lot of cases. It serves purely as a debugging aid.
	IoError      bool
}

func NewIntermediateWriteErrorData(bytesWritten int, expectedToWrite int) *WriteErrorData {
	return &WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != expectedToWrite, BytesWritten: bytesWritten, IoError: true}
}

func NewIntermediateReadErrorData(bytesRead int, expectedToRead int, actuallyRead []byte) *ReadErrorData {
	return &ReadErrorData{PartialRead: bytesRead != 0 && bytesRead != expectedToRead, BytesRead: bytesRead, ActuallyRead: actuallyRead, IoError: true}
}

// NoReadAttempt is a constant of type ReadErrorData that can be used if no read attempt was ever made, e.g. because some error was detected even before trying to read.
var NoReadAttempt = ReadErrorData{
	PartialRead:  false,
	BytesRead:    0,
	ActuallyRead: nil,
	IoError:      false,
}

// NoWriteAttempt is a constant of type WriteErrorData that can be used if no write attept was ever made, e.g. because some error was detected before even trying.
var NoWriteAttempt = WriteErrorData{
	PartialWrite: false,
	BytesWritten: 0,
	IoError:      false,
}

// The errorsWithData package can access fields by name (using reflection internally). We export the field names as constants for IDE-friendliness and as a typo- and refactoring guard.

const FIELDNAME_PARTIAL_WRITE = "PartialWrite"
const FIELDNAME_PARTIAL_READ = "PartialRead"
const FIELDNAME_ACTUALLY_READ = "ActuallyRead"

const FIELDNAME_BYTES_READ = "BytesRead"
const FIELDNAME_BYTES_WRITTEN = "BytesWritten"

const FIELDNAME_IO_ERROR = "IoError"

// Refactoring guard. This panics if the strings above don't correspond to the names of the exported field.
func init() {
	errorsWithData.CheckParametersForStruct_all[WriteErrorData]([]string{FIELDNAME_BYTES_WRITTEN, FIELDNAME_PARTIAL_WRITE, FIELDNAME_IO_ERROR})
	errorsWithData.CheckParametersForStruct_all[ReadErrorData]([]string{FIELDNAME_BYTES_READ, FIELDNAME_PARTIAL_READ, FIELDNAME_ACTUALLY_READ, FIELDNAME_IO_ERROR})
}

type SerializationError = errorsWithData.ErrorWithData[WriteErrorData]
type DeserializationError = errorsWithData.ErrorWithData[ReadErrorData]
