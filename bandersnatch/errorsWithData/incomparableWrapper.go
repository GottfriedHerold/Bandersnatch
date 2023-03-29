package errorsWithData

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

// incomparabilityUndoer_any is an internaly interface satisfied by the incomparable errors.
// It allows to access to underlying comparable error and "tags" types made incomparable by this package.
type incomparabilityUndoer_any interface {
	AsComparable_any() ErrorWithData_any
}

// incomparabilityUndoer is an internal interface satisfied by the incomparable errors
// It extends [incomparabilityUndoer_any] by preserving the StructType
type incomparabilityUndoer[StructType any] interface {
	incomparabilityUndoer_any
	AsComparable() ErrorWithData[StructType]
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

// Is is provided to satisfy the nameless interface that [errors.Is] expects. Note that
// the Is method does not follow the error chain; following the error chain is done internally by [errors.Is]
func (e incomparableError_any) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer_any); ok {
		return e.ErrorWithData_any == ce.AsComparable_any()
	}
	return target == e.ErrorWithData_any
}

// Is is provided to satisfy the nameless interface that [errors.Is] expects. Note that
// the Is method does not follow the error chain; following the error chain is done internally by [errors.Is]
func (e incomparableError[StructType]) Is(target error) bool {
	if ce, ok := target.(incomparabilityUndoer_any); ok {
		return e.ErrorWithData == ce.AsComparable_any()
	}
	return target == e.ErrorWithData
}

// AsComparable_any is provided to satisfy the incomparabilityUndoer interface.
// This serves both as a "tag" to mark incomparable errors and to unbox.
func (e incomparableError_any) AsComparable_any() ErrorWithData_any {
	return e.ErrorWithData_any
}

// AsComparable_any is provided to satisfy the [incomparabilityUndoer] interface.
// This serves both as a "tag" to mark incomparable errors and to unbox.
func (e incomparableError[StructType]) AsComparable_any() ErrorWithData_any {
	return e.ErrorWithData
}

// AsComparable_struct is provided to satisfy the [incomparabilityUndoer_struct] interface.
//
// This serves both as a "tag" to mark incomparable errors and to unbox. It differs from [AsComparable] by preserving StructType
func (e incomparableError[StructType]) AsComparable() ErrorWithData[StructType] {
	return e.ErrorWithData
}

// NOTE: Only allowing one level of boxing is just done to simplify the Is - method.

// MakeErrorIncomparable_any returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable_any(e ErrorWithData_any) incomparableError_any {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_any on nil error")
	}
	if ce, ok := e.(incomparabilityUndoer_any); ok {
		inner := ce.AsComparable_any()
		return MakeErrorIncomparable_any(inner)
	}
	return incomparableError_any{ErrorWithData_any: e}
}

// MakeErrorIncomparable_returns a boxed version of the given error which is not comparable.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
func MakeErrorIncomparable[StructType any](e ErrorWithData[StructType]) incomparableError[StructType] {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable on nil error")
	}
	if ce_struct, ok := e.(incomparabilityUndoer[StructType]); ok {
		return MakeErrorIncomparable[StructType](ce_struct.AsComparable())
	}
	if ce, ok := e.(incomparabilityUndoer_any); ok {
		inner := ce.AsComparable_any().(ErrorWithData[StructType])
		return MakeErrorIncomparable[StructType](inner)
	}
	return incomparableError[StructType]{ErrorWithData: e}
}
