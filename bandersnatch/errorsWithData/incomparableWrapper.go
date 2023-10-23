package errorsWithData

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

// TODO: Needs some cleanup (mostly removing parts of the API)

// This file defines features needed to make errors into incomparable types.
//
// Notably, our errorWithData package can be used to create errors that are only
// meaningful as base errors of an error chain.
//
// e.g. a package can export an error ErrFoo
//
//   var ErrFoo = NewErrorWithData_any_params(nil, "something went wrong. The value of Foo was $v{Foo}", ReplacePreviousData )
//
// This error contains no actual value for Foo
// The actually retuned errors would then be defined as
//
//   returnedError = NewErrorWithData_any_params(ErrFoo, "", ReplacePreviousData, "Foo", valueOfFoo)
//
// Users are expected to check the error type via errors.Is(returnedError, ErrFoo)
// Naively comparing err == ErrFoo is essentially always a bug.
//
// We try to disarm this potential footgun by making the exported ErrFoo incomparable (so err==ErrFoo outright does not compile).
// This is achieved by boxing ErrFoo in a incomparableError struct and only exporting the boxed variant.
//
//   BoxedErrFoo = MakeErrorIncomparable(ErrFoo)
//
// (In this scenario, we might actually opt to not export ErrFoo at all and probably use MakeErrorIncomparable_any)
// Note that this feature requires exporting the boxed variant as a non-interface type:
// go interfaces are always comparable; if the dynamic types turn out to be incomparable, we get a run-time panic.
// While this would be OK-ish (after all, it's a bug and panic is better than silently giving the wrong answer), we would essentially never trigger the panic:
// comparing interfaces first checks whether the dynamic types are equal and only if yes even checks whether the type is comparable.
// Consequently, exporting BoxedErrFoo via interfaces would cause (buggy!) returnedError == BoxedErrFoo to just always return false (and not panic),
// since returnedError is of a different type.
//
// To interplay nicely with the rest of the package and error wrapping, there are 2 options wrt error chaining:
//  - Base returnedError on BoxedErrFoo and have BoxedErrFoo wrap (via error chainging) ErrFoo
//  - Base returnedError on ErrFoo and have errors.Is not distinguish between BoxedErrFoo and ErrFoo
//
// We actually choose the second option; this is done to avoid having incomparable errors in error chains,
// as user code that follows the chain (e.g. a logger creating a map keyed by the errors it encountered) may wrongly assume errors are comparable and panic on it.
// To implement the second option, we use the hook that errors.Is provides:
// Notably, errors.Is(err, target) checks whether err's dynamic type defines an
// Is(target error) bool method and prefers that over plain comparison via ==.
// So we just need to define such an Is method that unboxes the error.
//
// Unfortunately, this Is method needs to be defined on err, not on target, which is the wrong way round for our needs.
// So it's actually the unboxed error that needs to opt-in to this mechanism
// This imposes some annoying limitations, complicates the implementation and, worst of all, couples it (somewhat) to the ErrorsWithData type.
// The alternative to box all returned errors would require returning errors as non-interface types.
// This is just a very bad idea due to potential confusion of nil interfaces with typed nils;
// also, making actually returned errors incomparable might trip up code that does not expect it (e.g. a logger creating a map keyed by errors will panic on this).
//
// To simplify some handling, we do the following:
//   - We only allow one level of boxing. All functions first unbox its inputs.
//   - As all our functions first unbox its inputs, using a boxed error as a base error for wrapping will in reality use the unboxed version.
//   - The boxed versions also satisfy the same relevant interface as its unboxed counterparts.
//
// Note that the last point actually requires having multiple MakeErrorIncomparable - variants and boxing types, depending on the preserved interface.
// (Making everything dependent on the preserved interface would require C++ - style *templates* and cannot be done with Go's *generics*)

// TODO: Unexport and rename.

// ComparableMaker is an internal interface satisfied by the incomparable errors.
//
// It allows to access the boxed comparable error and "tags" boxed types made incomparable by this package.
type ComparableMaker interface {
	AsComparable() IncomparableMaker
}

// IncomparableMaker is a interface that errors must satisfy in order to be usable by [MakeErrorIncompatible] and its variants.
//
// This is because the mechanism requires opt-in from a suitable Is - method, which is defined on the *unboxed* error.
// So we only allow boxing errors if the unboxed error has one.
type IncomparableMaker interface {
	error
	Is(target error) bool // needs to unbox target. A valid implementation of e.Is(target) is e==UnboxError(target)
	CanMakeIncomparable() // dummy function, this only serves to "mark" types as compatible with MakeErrorIncomparable. We never call this.
}

// Q: Should we actually export the concrete type (and rename the internal interface)?

// incomparableError_plain boxes an error, but is intentionally made to be incomparable.
// Note that a given incomparableError_plain does not wrap (in the sense of error wrapping) its contained error,
// but rather behaves like it wrt most purposes. Use errors.Is() for comparison.
//
// The aim of this is that exporting errors as this type will make plain comparison via the == operator fail at compile-time and force
// users to use errors.Is(). For "base errors", the latter is the only correct usage pattern (and actually what we expect users to always do).
//
// NOTE: This feature relies on exporting base errors (that users are expected to compare against) as a concrete type, NOT as an interface.
type incomparableError_plain struct {
	utils.MakeIncomparable
	IncomparableMaker // not error, because we want to promote Is and CanMakeIncomparable
}

// ewd_MakeIncomparable_any is the union of [ErrorWithData_any] and [IncomparableMaker].
// This is used for struct embedding, which is why it needs some (internal) name.
//
// Note: IncomparableMaker may actually a be subinterface of ErrorWithData_any,
// so this may be technically superfluous. (May change by refactoring)
// However, even if true, I don't like to rely on this here.
type ewd_MakeIncomparable_any = interface {
	ErrorWithData_any
	IncomparableMaker
}

// ewd_MakeIncomparable_struct is the union of [ErrorWithData] and [IncomparableMaker], similar to [ewd_MakeIncomparable_any]
type ewd_MakeIncomparable_struct[StructType any] interface {
	ErrorWithData[StructType]
	IncomparableMaker
}

// incomparableError_any is similar to [incomparableError_plain], but holds an [ErrorWithData_any]
//
// The point here is that this struct-embeds the ErrorWithData_any and consequently satisfies that interface itself.
type incomparableError_any struct {
	utils.MakeIncomparable
	ewd_MakeIncomparable_any // Note: This is NOT an incomparableError itself
}

// incomparableError is similar to [incomparableError_plain], but holds an ErrorWithData[StructType]
//
// The point here is that this struct-embeds the ErrorWithData[StructType] and consequently satisfies that interface itself.
type incomparableError[StructType any] struct {
	utils.MakeIncomparable
	ewd_MakeIncomparable_struct[StructType]
}

// UnboxError unboxes an error made incomparable by MakeIncomparable, returning the contained error.
// On non-boxed errors, just returns e itself.
func UnboxError(e error) error {
	if errUnboxable, ok := e.(ComparableMaker); ok {
		return errUnboxable.AsComparable()
	} else {
		return e
	}
}

// Q: Do we need this?

// Q: Should we use unboxable_any (or something like that) here?
// Currently, we type-assert after calling "plain" AsComparable.
// The reason is that this allows to later change the input type to plain error and
// have the function still work as intended for users that use plain MakeErrorIncomparable
// on ErrorWithData_any.

// UnboxError_any does the same as [UnboxError], but preserves the [ErrorWithData_any] interface.
//
// Note: [IncomparableMaker] is a subinterface of [ErrorWithData_any], so the restriction on e is partly redundant.
// This is just for being explicit.
func UnboxError_any(e interface {
	ErrorWithData_any
	IncomparableMaker
}) ErrorWithData_any {
	if errUnboxable, ok := e.(ComparableMaker); ok {
		return errUnboxable.AsComparable().(ErrorWithData_any)
	} else {
		return e
	}
}

// Q: Do we need this?

// UnboxError_struct does the same as [UnboxError], but preserves the [ErrorWithData] interface.
//
// Note: [IncomparableMaker] is a subinterface of [ErrorWithData], so the restriction on e is partly redundant.
// This is just for being explicit.
func UnboxError_struct[StructType any](e interface {
	ErrorWithData[StructType]
	IncomparableMaker
}) ErrorWithData[StructType] {
	if errUnboxable, ok := e.(ComparableMaker); ok {
		return errUnboxable.AsComparable().(ErrorWithData[StructType])
	} else {
		return e
	}
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError_plain) AsComparable() IncomparableMaker {
	return e.IncomparableMaker
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError_any) AsComparable() IncomparableMaker {
	return e.ewd_MakeIncomparable_any
}

// AsComparable is provided to satisfy the [incomparabilityUndoer] interface. It just unboxes the error.
func (e incomparableError[StructType]) AsComparable() IncomparableMaker {
	return e.ewd_MakeIncomparable_struct
}

// Do we even need those 2?

// AsComparable_typed is similar to [AsComparable], but preserves the [ErrorWithData_any] extension of the error interface.
func (e incomparableError_any) AsComparable_typed() ErrorWithData_any {
	return e.ewd_MakeIncomparable_any
}

// AsComparable_typed is similar to [AsComparable], but preserves the [ErrorWithData[StructType]] extension of the error interface.
func (e incomparableError[StructType]) AsComparable_typed() ErrorWithData[StructType] {
	return e.ewd_MakeIncomparable_struct
}

// Note: Due to deficiencies of Go's type system, we do not define MakeErrorIncomparable as a method of types that
// support this feature (which would be the obvious and "right" way to do it)
// The issue at hand is the return type and that this would need to be part of ErrorsWithData.
//
// Since we only expose errors that support this feature via the ErrorWithData or ErrorWithData_any interface,
// any such MakeErrorIncomparable methods need to become a part of the interface (unless we want to force users to sprinkle type-assert all over the place)
//
// Notably, MakeErrorIncomparable needs to return a non-interface type to do its job.
// However, we want to have versions that preserve any kind of extended interface.
// Making this a generic dependency on the extended interface does not work:
// (generics are *not* templates -- incomparable[T error] struct{T; utils.MakeIncomparable} does not and cannot work).
// So we really need incomparableError_plain / incomparableError_any / incomparableError as separate structs.
// Those struct types are completely independent as far as the type system is concerned.
// Consequently we would need to have multiple methods (or just a "plain" method and the extended versions as free functions)
//
// Multiple methods would couple this boxing feature even closer to ErrorsWithData (which is not intended), having just a plain method
// and extended free functions is not much different from we have now (and just adds more confusion to the API)
//
// Furthermore, there is not really much of a way to customize this operation for a given user-defined error type;
// the free functions that preserve extended interface would need to know too much about the implementation.

// NOTE: Only allowing one level of boxing is just done to simplify the Is - method.
// We currently panic on nil input due to the fact that the function returns a struct (which cannot be nil).
// this can be potentially confusing. We do not specify the behaviour in order to be able to change it later.
// (The alternative of returning a boxed nil may be useful in certain circumstances)

// NOTE: The input types to MakeErrorIncomparable_any and MakeErrorIncomparable_struct are anonoymous interfaces,
// even though we may have (unexported) definitions matching those. This is intentional.

// MakeErrorIncomparable returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
//
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparisons using errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
// The behaviour when calling this function on nil is unspecified and the function may panic. The precise behaviour may be subject to change.
func MakeErrorIncomparable(e IncomparableMaker) incomparableError_plain {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_plain on nil error")
	}
	// unbox the input, if possible
	e = UnboxError(e).(IncomparableMaker)
	return incomparableError_plain{IncomparableMaker: e}
}

// MakeErrorIncomparable_any returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
// Use this _any version to preserve the ErrorsWithData_any interface.
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
// The behaviour when calling this function on nil is unspecified and the function may panic. The precise behaviour may be subject to change.
func MakeErrorIncomparable_any(e interface {
	ErrorWithData_any
	IncomparableMaker
}) incomparableError_any {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_any on nil error")
	}
	e = UnboxError(e).(ewd_MakeIncomparable_any) // UnboxError_any???
	return incomparableError_any{ewd_MakeIncomparable_any: e}
}

// MakeErrorIncomparable_struct returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
// Use the _struct version to preserve the ErrorsWithData[StructType] interface.
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
// The behaviour when calling this function on nil is unspecified and the function may panic. The precise behaviour may be subject to change.
func MakeErrorIncomparable_struct[StructType any](e interface {
	ErrorWithData[StructType]
	IncomparableMaker
}) incomparableError[StructType] {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_struct on nil error")
	}
	e = UnboxError_struct[StructType](e).(ewd_MakeIncomparable_struct[StructType])
	return incomparableError[StructType]{ewd_MakeIncomparable_struct: e}
}
