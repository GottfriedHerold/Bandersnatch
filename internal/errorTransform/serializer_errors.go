package errorTransform

// TODO: Move some of the definitions here around.

import (
	"errors"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// This file contains common definitions of errors and error data that are not restricted to a single package.

// WriteErrorData is a struct holding additional information about Serialization errors. This additional data can be accessed via the errorsWithData package.
type WriteErrorData struct {
	PartialWrite bool // If PartialWrite is true, this indicates that some write operations failed after partially writing something and the io.Writer is consequently in an invalid state.
	BytesWritten int  // BytesWritten indicates the number of bytes that were written by the operation that *caused* the error. NOTE: All Serialization functions accurately return the number of bytes written directly as a non-error parameter. This may differ, because the cause might be in a sub-call.
}

// ReadErrorData is a struct holding additional information about Deserialization errors. This additional data can be accessed via the errorsWithData package.
type ReadErrorData struct {
	PartialRead  bool   // If PartialRead is true, this indicates that after the read error, the io.Reader is believed to be in an invalid state because what was read did not correspond to a complete blob of data that was expected.
	BytesRead    int    // BytesRead indicates the number of bytes that were read by the operation that *caused* the error. NOTE: All Deserialization functions accurately return the number of read bytes directly. The value reported here may differ, because it is the numbe of bytes read in the function that *caused* the error (which may be a sub-call).
	ActuallyRead []byte // this may contain information about data that was read when the error occured. It may be nil, is not guaranteed to be present (even if meaningful) and may be from a sub-call. The reason is that we do not buffer the raw input data, so we cannot provide it in a lot of cases. It serves purely as a debugging aid.
}

// NoReadAttempt is a constant of type ReadErrorData that can be used if no read attempt was ever made, e.g. because some error was detected even before trying to read.
var NoReadAttempt = ReadErrorData{
	PartialRead:  false,
	BytesRead:    0,
	ActuallyRead: nil,
}

// NoWriteAttempt is a constant of type WriteErrorData that can be used if no write attept was ever made, e.g. because some error was detected before even trying.
var NoWriteAttempt = WriteErrorData{
	PartialWrite: false,
	BytesWritten: 0,
}

// The errorsWithData package can access fields by name (using reflection internally). We export the field names as constants for IDE-friendliness and as a typo- and refactoring guard.

const FIELDNAME_PARTIAL_WRITE = "PartialWrite"
const FIELDNAME_PARTIAL_READ = "PartialRead"
const FIELDNAME_ACTUALLY_READ = "ActuallyRead"

const FIELDNAME_BYTES_READ = "BytesRead"
const FIELDNAME_BYTES_WRITTEN = "BytesWritten"

// Refactoring guard. This panics if the strings above don't correspond to the names of the exported field.
func init() {
	errorsWithData.CheckParametersForStruct[WriteErrorData]([]string{FIELDNAME_BYTES_WRITTEN, FIELDNAME_PARTIAL_WRITE})
	errorsWithData.CheckParametersForStruct[ReadErrorData]([]string{FIELDNAME_BYTES_READ, FIELDNAME_PARTIAL_READ, FIELDNAME_ACTUALLY_READ})
}

type SerializationError = errorsWithData.ErrorWithGuaranteedParameters[WriteErrorData]
type DeserializationError = errorsWithData.ErrorWithGuaranteedParameters[ReadErrorData]

// TODO: Move these definitions around?

var ErrWillNotSerializePointOutsideSubgroup = errors.New("bandersnatch / point serialization: trying to serialize point outside subgroup while serializer is subgroup-only")

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?

var (
	ErrXNotInSubgroup = errors.New("bandersnatch / point deserialization: received affine X coordinate does not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrXNotOnCurve    = errors.New("bandersnatch / point deserialization: received affine X coordinate does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrYNotOnCurve    = errors.New("bandersnatch / point deserialization: encountered affine Y coordinate that does not correspond to any (finite, rational) point of the Bandersnatch curve")
	ErrNotInSubgroup  = errors.New("bandersnatch / point deserialization: received affine X and Y coordinates do not correspond to a point in the p253 subgroup of the Bandersnatch curve")
	ErrNotOnCurve     = errors.New("bandersnatch / point deserialization: received affine X and Y coordinates do not correspond to a point on the Bandersnatch curve")
	ErrWrongSignY     = errors.New("bandersnatch / point deserialization: encountered affine Y coordinate with unexpected Sign bit")
	// ErrUnrecognizedFormat = errors.New("bandersnatch / point deserialization: could not automatically detect serialization format")
	ErrUnexpectedNegativeZero = errors.New("bandersnatch / point deserialization: encountered unexpected X=0 point with negative sign for X")
)

var ErrInvalidZeroSignX = errors.New("bandersnatch / point deserialization: When constructing curve point from Y and the sign of X, the sign of X was 0, but X==0 is not compatible with the given Y")
var ErrInvalidSign = errors.New("bandersnatch / point deserialization: impossible sign encountered")

// consumeExpectRead may return an error wrapping ErrDidNotReadExpectedString. Use errors.Is to compare.
var ErrDidNotReadExpectedString = errors.New("bandersnatch / point deserialization: did not read expected string") // Note: All users change the error message.

var ErrSizeDoesNotFitInt32 = errors.New("bandersnatch / point slice serialization: size of point slice does not fit into (signed) 32-bit integer")

var (
	ErrCannotSerializePointAtInfinity = errors.New("bandersnatch / point serialization: The selected serializer cannot serialize points at infinity")
	ErrCannotSerializeNaP             = errors.New("bandersnatch / point serialization: cannot serialize NaP")
	ErrCannotDeserializeNaP           = errors.New("bandersnatch / point deserialization: cannot deserialize coordinates corresponding to NaP")
	ErrCannotDeserializeXYAllZero     = errorsWithData.NewErrorWithGuaranteedParameters[struct{}](ErrCannotDeserializeNaP, "bandersnatch / point deserialization: trying to deserialize a point with coordinates x==y==0")
)
