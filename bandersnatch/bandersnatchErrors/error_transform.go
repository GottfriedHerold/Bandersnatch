package bandersnatchErrors

// UnexptectEOF turns an (error wrapping an) io.EOF error into an io.UnexpectedEOF or an error wrapping io.UnexpectedEOF.
// The latter is commonly used by the standard library to indicate an EOF when reading multiple bytes from a stream and there was an EOF in the middle of reading.
// By contrast, io.EOF is returned when there is an EOF at the beginning.
//
// Note: If the error wraps io.EOF then the additional errors in the error chain are lost.
// However, BandersnatchError parameters are retained; if such parameters are present, the error will wrap io.UnexpectedEOF
// rather than being equal to it.

// TODO: Redo.

/*
func UnexpectEOF(errPtr *error) {
	if errors.Is(*errPtr, io.EOF) {
		m := GetAllParametersFromError(*errPtr)
		*errPtr = io.ErrUnexpectedEOF
		for key, value := range m {
			IncludeParametersInError(errPtr, key, value)
		}
	}
}
*/
