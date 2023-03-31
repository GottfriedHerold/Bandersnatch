package errorsWithData

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

// incomparabilityUndoer is an internaly interface satisfied by the incomparable errors.
// It allows to access to underlying comparable error and "tags" types made incomparable by this package.
type incomparabilityUndoer interface {
	AsComparable() error
}

/*
	NOTE: The point of having different versions here is because we want MakeIncomparable to preserve certain interfaces.
	(i.e. the boxing error should satisfy the same interface as the boxed one)
*/

// incomparableError_plain boxes an error, but is intentionally made to be incomparable.
// Note that a given incomparableError_plain does not wrap (in the sense of error wrapping) its contained error,
// but rather behaves like it wrt most purposes. Use errors.Is() for comparison.
//
// The aim of this is that exporting errors as this type will make plain == comparison fail at compile-time and force
// users to use errors.Is(). For "base errors", the latter is the correct usage pattern.
// NOTE: This relies on exporting base errors as a concrete type, NOT as an interface.
// since Go interfaces are always comparable -- if the contained type is not comparable, we get a run-time panic.
type incomparableError_plain struct {
	utils.MakeIncomparable
	error
}

// incomparableError_any is similar to [incomparableError_plain], but holds an [ErrorWithData_any]
//
// The point here is that this struct-embeds the ErrorWithData_any and consequently satisfies that interface itself.
type incomparableError_any struct {
	utils.MakeIncomparable
	ErrorWithData_any // Note: This is NOT an incomparableError itself
}

// incomparableError is similar to [incomparableError_plain], but holds an ErrorWithData[StructType]
//
// The point here is that this struct-embeds the ErrorWithData[StructType] and consequently satisfies that interface itself.
type incomparableError[StructType any] struct {
	utils.MakeIncomparable
	ErrorWithData[StructType] // Note: This is NOT an incomparableError itself
}

// Is is provided to satisfy the nameless interface that [errors.Is] expects. Note that
// the Is method does not follow the error chain; following the error chain is done internally by [errors.Is]
func (e incomparableError_any) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer); ok {
		return e.ErrorWithData_any == ce.AsComparable()
	}
	return target == e.ErrorWithData_any
}

// Is is provided to satisfy the nameless interface that [errors.Is] expects. Note that
// the Is method does not follow the error chain; following the error chain is done internally by [errors.Is]
func (e incomparableError[StructType]) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer); ok {
		return e.ErrorWithData == ce.AsComparable()
	}
	return target == e.ErrorWithData
}

func (e incomparableError_plain) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer); ok {
		return e.error == ce.AsComparable()
	}
	return target == e.error
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError_plain) AsComparable() error {
	return e.error
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError_any) AsComparable() error {
	return e.ErrorWithData_any
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError[StructType]) AsComparable() error {
	return e.ErrorWithData
}

// AsComparable_typed is similar to [AsComparable], but preserves the [ErrorWithData_any] extension of the error interface.
func (e incomparableError_any) AsComparable_typed() ErrorWithData_any {
	return e.ErrorWithData_any
}

// AsComparable_typed is similar to [AsComparable], but preserves the [ErrorWithData[StructType]] extension of the error interface.
func (e incomparableError[StructType]) AsComparable_typed() ErrorWithData[StructType] {
	return e.ErrorWithData
}

// NOTE: Only allowing one level of boxing is just done to simplify the Is - method.

// MakeErrorIncomparable_returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable(e error) incomparableError_plain {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_plain on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer); ok {
		inner := ce.AsComparable()
		return MakeErrorIncomparable(inner)
	}
	return incomparableError_plain{error: e}
}

// MakeErrorIncomparable_any returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable_any(e ErrorWithData_any) incomparableError_any {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_any on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer); ok {
		inner := ce.AsComparable().(ErrorWithData_any)
		return MakeErrorIncomparable_any(inner)
	}
	return incomparableError_any{ErrorWithData_any: e}
}

// MakeErrorIncomparable_returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable_struct[StructType any](e ErrorWithData[StructType]) incomparableError[StructType] {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer); ok {
		inner := ce.AsComparable().(ErrorWithData[StructType])
		return MakeErrorIncomparable_struct(inner)
	}
	return incomparableError[StructType]{ErrorWithData: e}
}
