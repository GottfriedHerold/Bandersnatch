package pointserializer

type ParameterAware interface {
	RecognizedParameters() []string
	HasParameter(paramName string) bool
	GetParameter(paramName string) any
	// WithParameter(paramName string, newParam any) AssignableToReceiver -- Note: The return type may be either an interface or the concrete receiver type. The lack of covariance is why this is not part of the actual interface, but enforced at runtime using reflection.
	// We ask that the *dynamic* type returned is the same as the receiver.
}

// TODO: Might not be exported?

func WithParameter[T ParameterAware](t T, paramName string, newParam any) T {
	panic(0)
	return t
}
