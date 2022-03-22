package bandersnatchErrors

// WrappedError is a simple error that only contains a string (similar to errors.New(s string)'s output), but in addition wraps another given error.
// This has the effect that errors.Is(error1, error2) will return true if error2 == error1.Inner
// WrappedError is similar to the output of fmt.Errorf("%w", Inner ), execpt that WrappedError's Error() method can avoid including the wrapped error's string.
// This really should be provided as part of the standard library (e.g. as a silent version of %w)
type WrappedError struct {
	Inner   error
	Message string
}

func NewWrappedError(inner error, s string) error {
	return &WrappedError{Inner: inner, Message: s}
}

func (we *WrappedError) Error() string {
	return we.Message
}

func (we *WrappedError) Unwrap() error {
	return we.Inner
}
