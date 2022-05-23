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

// WILL BE REMOVED in favor of more flexible (but less type-safe) BandersnatchError

// ErrorWithData is an error wrapper similar to WrappedError that also contains a data payload of type DataType.
type ErrorWithData[DataType any] struct {
	Inner   error    // wrapped error
	Message string   // message that is displayed by Error, overriding Inner's error message. If Message == "", we take the one from Inner.
	Data    DataType // embedded Data
}

func (dce *ErrorWithData[DataType]) Error() string {
	if dce.Message != "" {
		return dce.Message
	} else {
		return dce.Inner.Error()
	}

}

func (dce *ErrorWithData[DataType]) Unwrap() error {
	return dce.Inner
}

func NewErrorWithData[DataType any](inner error, s string, data DataType) *ErrorWithData[DataType] {
	return &ErrorWithData[DataType]{Inner: inner, Message: s, Data: data}
}
