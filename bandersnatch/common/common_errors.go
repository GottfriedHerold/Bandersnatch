package common

import "github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"

// SerializationError is an interface type extending the error interface.
// It is used to return errors from serialization methods or functions.
// As opposed to plain errors, variables of this type embed embed additional diagnostic data via the [errorsWithData] framework.
// Concretely, it contains (at least) data as specified by the [WriteErrorData] type. Note that in many cases, we have even more data.
//
// Caveat: Our serialization functions/methods for more comlex objects often operate piece-by-piece and make sub-calls to serialize parts of the data.
// Serialization failures might thus originate from these sub-calls.
// Rather than directly forwarding the sub-call's errors, we account for that by error wrapping and modifying the diagnostic data before returning errors to the user.
// Please refer to [WriteErrorData] for the precise meaning of its field.
type SerializationError = errorsWithData.ErrorWithData[WriteErrorData]

// DeserializationError is an interface type extending the error interface.
// It is used to return errors from deserialization methods or functions.
// As opposed to plain errors, variables of this type embed embed additional diagnostic data via the [errorsWithData] framework.
// Concretely, it contains (at least) data as specified by the [ReadErrorData] type. Note that in many cases, we have even more data.
//
// Be aware of the caveats regarding the accuracy / precise meaning of these data entries as described in [ReadErrorData].
type DeserializationError = errorsWithData.ErrorWithData[ReadErrorData]

// WriteErrorData is a struct holding additional information about [SerializationError]s. This additional data can be accessed via the [errorsWithData] package.
type WriteErrorData struct {
	PartialWrite bool // If PartialWrite is true, this indicates that some write operations failed after partially writing something and the io.Writer is consequently likely in an unusable state.
	BytesWritten int  // BytesWritten indicates the number of bytes that were written by the operation that *caused* the error.
	// NOTE: All Serialization functions accurately return the number of bytes written directly as a non-error parameter.
	// The value BytesWritten returned as parameter of the error may differ, because the cause might be in a sub-call.
	IoError bool // IoError indicates whether the error comes from math or from I/O.
	// Notably, IoError is true if the error was due a problem with the io.Writer object being written to.
	// IoError is false if the error is due to a problem with the object(s) that the package was asked to serialize.
	// The latter can only happen for certain serialization methods, e.g. SerializeWithPrefix with non-fitting prefix.
}

// TODO: PartialRead is confusingly phrased and does not really correspond to what we are doing.
// After an unexpected EOF, the io.Reader is in a "good" state; the issue is more with the data read into.

// ReadErrorData is a struct holding additional information about [DeserializationError]s. This additional data can be accessed via the [errorsWithData] package.
type ReadErrorData struct {
	PartialRead bool // If PartialRead is true, this indicates that after the read error, the io.Reader is believed to be in an unusable state for further reads, because what was read did not correspond to a complete blob of data that was expected.
	// PartialRead is not
	BytesRead int // BytesRead indicates the number of bytes that were read by the operation that *caused* the error.
	// NOTE: All Deserialization functions accurately return the number of read bytes directly.
	// The value BytesRead reported in the error may differ, because it is the numbe of bytes read in the function that *caused* the error (which may be a sub-call).
	ActuallyRead []byte // this *may* contain information about data that was read when the error occured.
	// It may be nil even if we actually read data (so it would be meaningful) and may be from a sub-call.
	// The reason is that we do not buffer the raw input data, so we cannot provide it in a lot of cases.
	// It serves purely as a debugging aid.
	IoError bool // IoError indicates whether the error comes from math or from I/O.
}
