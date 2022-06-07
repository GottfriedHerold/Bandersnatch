package bandersnatchErrors

import (
	"errors"
	"io"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/errorsWithData"
)

// UnexptectEOF turns an (error wrapping an) io.EOF error into an io.UnexpectedEOF or an error wrapping io.UnexpectedEOF.
// The latter is commonly used by the standard library to indicate an EOF when reading multiple bytes from a stream and there was an EOF in the middle of reading.
// By contrast, io.EOF is returned when there is an EOF at the beginning.
//
// Note: If the error wraps io.EOF then the additional errors in the error chain are lost.
// However, errorsWithData parameters are retained; if such parameters are present, the error will wrap io.UnexpectedEOF
// rather than being equal to it.
func UnexpectEOF(errPtr *error) {
	if errors.Is(*errPtr, io.EOF) {
		m := errorsWithData.GetAllParametersFromError(*errPtr)
		if len(m) > 0 {
			*errPtr = errorsWithData.NewErrorWithParametersUnconstrainedMap(io.ErrUnexpectedEOF, "", m)
		} else {
			*errPtr = io.ErrUnexpectedEOF
		}
	}
}
