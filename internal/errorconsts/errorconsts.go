package errorconsts

import (
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// NewIntermediateWriteErrorData returns a (newly allocated) *WriteErrorData that is appropriate for an IO error
// where bytesWritten out of expectedToWrite many bytes were actually written in the failing operation.
//
// The returned [*WriteErrorData] has its PartialWriteFlag set iff 0 < bytesWritten < expectedToWrite. The IOError flag is always set to true.
// We assert (but do not test) that bytesWritten and expectedToWrite are non-negative.
func NewIntermediateWriteErrorData(bytesWritten int, expectedToWrite int) *common.WriteErrorData {
	return &common.WriteErrorData{PartialWrite: bytesWritten != 0 && bytesWritten != expectedToWrite, BytesWritten: bytesWritten, IoError: true}
}

// NewIntermediateReadErrorData returns a (newly allocated) *ReadErrorData that is appropriate for an IO error
// where bytesRead out of expectedToRead many bytes were read. actuallyRead is supposed to hold the actually read bytes or nil (see [ReadErrorData] for limitations)
//
// The returned [*ReadErrorData] has its ParialRead flag set iff bytesRead is neither 0 nor expectedToRead. The IOError flag is always true.
// The ActuallyRead field is directly set to the passed actuallyRead (without deep-copying). The limitation of this data field still apply.
func NewIntermediateReadErrorData(bytesRead int, expectedToRead int, actuallyRead []byte) *common.ReadErrorData {
	return &common.ReadErrorData{PartialRead: bytesRead != 0 && bytesRead != expectedToRead, BytesRead: bytesRead, ActuallyRead: actuallyRead, IoError: true}
}

// NoReadAttempt is a constant of type ReadErrorData that can be used if no read attempt was ever made, e.g. because some error was detected even before trying to read.
var NoReadAttempt = common.ReadErrorData{
	PartialRead:  false,
	BytesRead:    0,
	ActuallyRead: nil,
	IoError:      false,
}

// NoWriteAttempt is a constant of type WriteErrorData that can be used if no write attept was ever made, e.g. because some error was detected before even trying.
var NoWriteAttempt = common.WriteErrorData{
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
	errorsWithData.CheckParametersForStruct_all[common.WriteErrorData]([]string{FIELDNAME_BYTES_WRITTEN, FIELDNAME_PARTIAL_WRITE, FIELDNAME_IO_ERROR})
	errorsWithData.CheckParametersForStruct_all[common.ReadErrorData]([]string{FIELDNAME_BYTES_READ, FIELDNAME_PARTIAL_READ, FIELDNAME_ACTUALLY_READ, FIELDNAME_IO_ERROR})
}
