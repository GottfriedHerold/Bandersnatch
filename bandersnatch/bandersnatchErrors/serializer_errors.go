package bandersnatchErrors

import (
	"errors"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

type WriteErrorData struct {
	PartialWrite bool
	BytesWritten int
}

type ReadErrorData struct {
	PartialRead  bool
	BytesRead    int
	ActuallyRead []byte // this may contain information about data that was read when the error occured. It may be nil.
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
var ErrDidNotReadExpectedString = errors.New("bandersnatch / point deserialization: did not read expected string")

var ErrSizeDoesNotFitInt32 = errors.New("bandersnatch / point slice serialization: size of point slice does not fit into (signed) 32-bit integer")

var (
	ErrCannotSerializePointAtInfinity = errors.New("bandersnatch / point serialization: The selected serializer cannot serialize points at infinity")
	ErrCannotSerializeNaP             = errors.New("bandersnatch / point serialization: cannot serialize NaP")
	ErrCannotDeserializeNaP           = errors.New("bandersnatch / point deserialization: cannot deserialize coordinates corresponding to NaP")
	ErrCannotDeserializeXYAllZero     = errorsWithData.NewErrorWithGuaranteedParameters[struct{}](ErrCannotDeserializeNaP, "bandersnatch / point deserialization: trying to deserialize a point with coordinates x==y==0")
)
