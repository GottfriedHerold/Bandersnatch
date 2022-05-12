package utils

import (
	"errors"
	"io"
)

// UnexptectEOF turns an (error wrapping an) io.EOF error into a io.UnexpectedEOF error.
// The latter is commonly used by the standard library to indicate an EOF when reading multiple bytes from a stream and there was an EOF in the middle of reading.
// By contrast, io.EOF is returned when there is an EOF at the beginning.
//
// Note: If the error wraps io.EOF and, any such additional information in the wrapper is lost.
func UnexpectEOF(errPtr *error) {
	if errors.Is(*errPtr, io.EOF) {
		*errPtr = io.ErrUnexpectedEOF
	}
}
