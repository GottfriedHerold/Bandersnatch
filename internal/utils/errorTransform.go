package utils

import (
	"errors"
	"io"
)

func UnexpectEOF(errPtr *error) {
	if errors.Is(*errPtr, io.EOF) {
		*errPtr = io.ErrUnexpectedEOF
	}
}
