// errorTransform is an internal (mini-)package that contains sub-routines to manipulate errors that are used for both curvePoints and fieldElements.
//
// Note: This is a separate package mostly because it depends on errorsWithData
package errorTransform

import (
	"errors"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// This file contains common code that is often needed to modify errors. This will be moved to internal utils at some point, I guess.

// UnexpectEOF turns an (error wrapping an) [io.EOF] error into an [io.UnexpectedEOF] or an error wrapping [io.UnexpectedEOF].
// [io.UnexpectedEOF] is commonly used by the standard library to indicate an EOF when reading multiple bytes from a stream and there was an EOF in the middle of reading.
// By contrast, [io.EOF] is returned when there is an EOF at the beginning (i.e. the reader was empty to start with).
//
// Note: If the error wraps [io.EOF] then the additional errors in the error chain are lost.
// However, extra data embedded via the [errorsWithData] package are retained; if such parameters are present, the error will wrap [io.UnexpectedEOF]
// rather than being equal to it.
func UnexpectEOF(errPtr *error) {
	if errors.Is(*errPtr, io.EOF) {
		m := errorsWithData.GetData_map(*errPtr)
		if len(m) > 0 {
			*errPtr, _ = errorsWithData.NewErrorWithData_any_map(io.ErrUnexpectedEOF, "", m, errorsWithData.PanicOnAllMistakes)
		} else {
			*errPtr = io.ErrUnexpectedEOF
		}
	}
}

// UnexpectEOF2 does the same as UnexpectEOF, except that it preserves the compile-time information about the StructType in [errorsWithData.ErrorWithData][StructType]
func UnexpectEOF2[StructType any](errPtr *errorsWithData.ErrorWithData[StructType]) {
	if errors.Is(*errPtr, io.EOF) {
		m := errorsWithData.GetData_map(*errPtr)
		*errPtr, _ = errorsWithData.NewErrorWithData_map[StructType](io.ErrUnexpectedEOF, "", m, errorsWithData.PanicOnAllMistakes)
	}
}
