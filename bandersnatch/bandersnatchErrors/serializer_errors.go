package bandersnatchErrors

import (
	"errors"
	"fmt"
	"os"
)

// BatchSerializationError is the interface type of errors that are returned by (De-)Serialization routines for multiple points
// This extends the standard error interface by allowing to query for information in case of error about the state change in case in addition to the actual error.
// Notably, it allows to query how many points were actually read/written in case of potential partial reads.
//
// Note that in case of no error, we return nil, so this information (albeit it would meaningful) cannot be gotten via the returned error.
// However, it is available via other means such as len() of the returned slice, provided buffer etc.
type BatchSerializationError interface {
	error
	PointsWritten() int // How many points were written to the io.Writer in case of error
	PartialWrite() bool // returns true, if we partially wrote a point
	// PointsRead() int    // Number of points that were read
	Unwrap() error // requires for errors.Is and errors.As
}

type BatchDeserializationError interface {
	error
	PointsRead() int
	PartialRead() bool
	Unwrap() error
}

// batchSerializationError is an error wrapper returned when there is an error when trying to serializae a batch of curve points.
// Since serialization a batch of curve points is equivalent to serializing them individually, there is the meaningful case of writing into a too small buffer (which gives an EOF or UnexpectedEOF); in this case, only a subset will be written
// and the extra information contained can be used to query how many were written.
type batchSerializationError struct {
	e             error
	pointsWritten int
	partialWrite  bool
}

func (be *batchSerializationError) PointsWritten() int { return be.pointsWritten }
func (be *batchSerializationError) PartialWrite() bool { return be.partialWrite }
func (be *batchSerializationError) Unwrap() error      { return be.e }
func (be *batchSerializationError) Error() string {
	if be.partialWrite {
		return fmt.Sprintf("bandersnatch / point serialization: error during serialization of multiple points. Only %v points were written in addition to an incomplete write. The error encountered was %v", be.pointsWritten, be.e)
	} else {
		return fmt.Sprintf("bandersnatch / point serialization: error during serialization of multiple points. Only %v points were written. The error encountered was %v", be.pointsWritten, be.e)
	}
}

func NewBatchSerializationError(e error, pointsWritten int, partialWrite bool) BatchSerializationError {
	return &batchSerializationError{e: e, pointsWritten: pointsWritten, partialWrite: partialWrite}
}

// sliceSerializationError is an error wrapper returned when there is an error when trying to serialize a slice of curve points.
// Since serializating a slice of curve points should be considered atomic for the user (i.e. reading it back is only allowed/meaningful as a whole slice of the exact size that we wrote -- we write size information in-band into the stream),
// we do not care too much about the additional data and they only matter for diagnostics (in the sense that we do not expect the user to handle this gracefully)
type sliceSerializationError struct {
	e             error
	pointsWritten int
	partialWrite  bool // Note: This is basically always true except for the case where we wrote nothing at all.
}

func (be *sliceSerializationError) PointsWritten() int { return be.pointsWritten }
func (be *sliceSerializationError) PartialWrite() bool { return be.partialWrite }
func (be *sliceSerializationError) Unwrap() error      { return be.e }
func (be *sliceSerializationError) Error() string {
	if be.partialWrite {
		return fmt.Sprintf("bandersnatch / point serialization: error during serialization of a slice of points. The error occurred at some point after %v points were completely written. The error encountered was %v", be.pointsWritten, be.e)
	} else {
		if be.pointsWritten != 0 {
			// We must never get here.
			ret := fmt.Sprintf("bandersnatch / point serialization: INTERNAL ERROR IN ERROR HANDLING. Error was %v. Error claims to be no partial write, but wrote %v points, which makes no sense for slice writes", be.e, be.pointsWritten)
			os.Stderr.WriteString(ret)
			return ret
		} else {
			return fmt.Sprintf("bandersnatch / point serialization: error during serialization of slice of points. Could not write anything. Error was %v", be.e)
		}
	}
}

func NewSliceSerializationError(e error, pointsWritten int, partialWrite bool) BatchSerializationError {
	if !partialWrite && pointsWritten > 0 {
		panic("bandersnatch / point serialization: Creating an sliceSerializationError with partialWrite=false, pointsWritte>0.")
	}
	return &sliceSerializationError{e: e, pointsWritten: pointsWritten, partialWrite: partialWrite}
}

type batchDeserializationError struct {
	e           error
	pointsRead  int
	partialRead bool
}

func (be *batchDeserializationError) PointsRead() int   { return be.pointsRead }
func (be *batchDeserializationError) PartialRead() bool { return be.partialRead }
func (be *batchDeserializationError) Unwrap() error     { return be.e }
func (be *batchDeserializationError) Error() string {
	if be.partialRead {
		return fmt.Sprintf("bandersnatch / point deserialization: error during batch deserialization: Only %v points were read in in addition to an incomplete read. The error encountered was %v", be.pointsRead, be.e)
	} else {
		return fmt.Sprintf("bandersnatch / point deserialization: error during batch deserialization: Only %v points were read in. After that, the error encountered was %v", be.pointsRead, be.e)
	}
}

func NewBatchDeserializationError(e error, pointsRead int, partialRead bool) BatchDeserializationError {
	return &batchDeserializationError{e: e, pointsRead: pointsRead, partialRead: partialRead}
}

type sliceDeserializationError struct {
	e           error
	pointsRead  int
	partialRead bool // basically only true if we read nothing at all.
}

func (be *sliceDeserializationError) PointsRead() int   { return be.pointsRead }
func (be *sliceDeserializationError) PartialRead() bool { return be.partialRead }
func (be *sliceDeserializationError) Unwrap() error     { return be.e }
func (be *sliceDeserializationError) Error() string {
	if be.partialRead {
		return fmt.Sprintf("bandersnatch / point deserialization: error during deserialization of curve points at some point after reading %v points.\nError was %v", be.pointsRead, be.e)
	} else {
		if be.pointsRead != 0 {
			// We must never get here.
			ret := fmt.Sprintf("bandersnatch / point deserialization: INTERNAL ERROR IN ERROR HANDLING. Error was %v. Error claims to be no partial read of slice of points, but also claims to have read %v points", be.e, be.partialRead)
			os.Stderr.WriteString(ret)
			return ret
		} else {
			return fmt.Sprintf("bandersnatch / point deserialization: error during deserialization of curve points. Could not read anything. Error was %v", be.e)
		}
	}
}

var (
	ErrCannotSerializePointAtInfinity = errors.New("bandersnatch / point serialization: The selected serializer cannot serialize points at infinity")
	ErrCannotSerializeNaP             = errors.New("bandersnatch / point serialization: cannot serialize NaP")
	ErrCannotDeserializeXYAllZero     = NewWrappedError(ErrCannotSerializeNaP, "bandersnatch / point deserialization: trying to deserialize a point with coordinates x==y==0") // special case of ErrCannotSerializeNaP
)

func NewSliceDeserializationError(e error, pointsRead int, partialRead bool) BatchDeserializationError {
	if !partialRead && pointsRead > 0 {
		panic("bandersnatch / point deserialization: Creating an sliceDeserializationError with partialRead=false, pointsRead>0.")
	}

	return &sliceDeserializationError{e: e, pointsRead: pointsRead, partialRead: partialRead}
}

var ErrWillNotSerializePointOutsideSubgroup = errors.New("bandersnatch / point serialization: trying to serialize point outside subgroup while serializer is subgroup-only")

// Note: If X/Z is not on the curve, we might get either a "not on curve" or "not in subgroup" error. Should we clarify the wording to reflect that?

var (
	ErrXNotInSubgroup = errors.New("bandersnatch / point deserialization: received affine X coordinate does not correspond to any point in the p253 subgroup of the Bandersnatch curve")
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
