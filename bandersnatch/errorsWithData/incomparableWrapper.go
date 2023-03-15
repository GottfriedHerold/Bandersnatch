package errorsWithData

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

type incomparabilityUndoer interface {
	AsComparable() ErrorWithData_any
}

// incomparableError_any boxes an ErrorWithData_any, but is intentionally made to be incomparable.
// Note that a given incomparableError_any does not wrap (in the sense of error wrapping) its contained error,
// but rather behaves like it wrt most purposes. Use errors.Is() for comparison.
//
// The aim of this is that exporting errors as this type will make plain == comparison fail at compile-time and force
// users to use errors.Is(). For "base errors", the latter is the correct usage pattern.
// NOTE: This relies on exporting base errors as a concrete type, NOT as an interface.
// since Go interfaces are always comparable -- if the contained type is not comparable, we get a run-time panic.
type incomparableError_any struct {
	utils.MakeIncomparable
	ErrorWithData_any // Note: This is NOT an incomparableError itself
}

// incomparableError boxes an ErrorWithData, but is intentionally made to be incomparable.
// Note that a given incomparableError_any does not wrap (in the sense of error wrapping) its contained error,
// but rather behaves like it wrt most purposes. Use errors.Is() for comparison.
//
// The aim of this is that exporting errors as this type will make plain == comparison fail at compile-time and force
// users to use errors.Is(). For "base errors", the latter is the correct usage pattern.
// NOTE: This relies on exporting base errors as a concrete type, NOT as an interface.
// since Go interfaces are always comparable -- if the contained type is not comparable, we get a run-time panic.
type incomparableError[StructType any] struct {
	utils.MakeIncomparable // Note: This is NOT an incomparableError itself
	ErrorWithData[StructType]
}

// Is is provided to satisfy the nameless interface that errors.Is expects. Note that
// the Is method does not follow the error chain; this is done by errors.Is
func (e incomparableError_any) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer); ok {
		return e.ErrorWithData_any == ce.AsComparable()
	}
	return target == e.ErrorWithData_any
}

// Is is provided to satisfy the nameless interface that errors.Is expects. Note that
// the Is method does not follow the error chain; this is done by errors.Is
func (e incomparableError[StructType]) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer); ok {
		return e.ErrorWithData == ce.AsComparable()
	}
	return target == e.ErrorWithData
}

// AsComparable is provided to satisfy the incomparabilityUndoer interface.
// This serves both as a "tag" to mark incomparable errors and to unbox.
func (e incomparableError_any) AsComparable() ErrorWithData_any {
	return e.ErrorWithData_any
}

// AsComparable is provided to satisfy the incomparabilityUndoer interface.
// This serves both as a "tag" to mark incomparable errors and to unbox.
func (e incomparableError[StructType]) AsComparable() ErrorWithData_any {
	return e.ErrorWithData
}

// NOTE: Only allowing one level of boxing is just done to simplify the Is - method.

// MakeErrorIncomparable_any returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic)
// Note that comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable_any(e ErrorWithData_any) incomparableError_any {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_any on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer); ok {
		return incomparableError_any{ErrorWithData_any: ce.AsComparable()}
	}
	return incomparableError_any{ErrorWithData_any: e}
}

// MakeErrorIncomparable_returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic)
// Note that comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable[StructType any](e ErrorWithData[StructType]) incomparableError[StructType] {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer); ok {
		return incomparableError[StructType]{ErrorWithData: ce.AsComparable().(ErrorWithData[StructType])}
	}
	return incomparableError[StructType]{ErrorWithData: e}
}
