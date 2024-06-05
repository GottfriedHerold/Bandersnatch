package errorsWithData

import "github.com/GottfriedHerold/Bandersnatch/internal/utils"

// NOTE: This file was refactored A LOT OF TIMES with many changes to function names.

// This file defines features needed to make errors into incomparable types.
//
// Notably, our errorWithData package can be used to create errors that are only
// meaningful as base errors of an error chain.
//
// e.g. with this package, we can create an exported ErrorWithData_any ErrFoo
//
//   var ErrFoo = NewErrorWithData_any_params(nil, "something went wrong. The value of Foo was $v{Foo}")
//
// This error contains no actual value for Foo.
// The actually retuned errors would then be defined, e.g. as
//
//   returnedError = NewErrorWithData_any_params(ErrFoo, "", ReplacePreviousData, "Foo", 5)
//
// This would output returnedError.Error() == "something went wrong. The value of Foo was 5".
//
// If users actually check the returned errors type, they are supposed to do so via errors.Is(returnedError, ErrFoo).
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
// Consequently, exporting BoxedErrFoo via interfaces would cause (the buggy) code snippet returnedError == BoxedErrFoo to just always return false (and not panic),
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
// *** Unfortunately, this Is method needs to be defined on err, not on target, which is the wrong way around for our needs.***
//
// It's actually the *unboxed* error that needs to opt-in to this mechanism.
// This imposes some annoying limitations, complicates the implementation and, worst of all, couples it (somewhat) to the ErrorsWithData type.
// The alternative to box all returned errors would require returning errors as non-interface types.
// This is just a very bad idea due to potential confusion of nil interfaces with typed nils;
// also, making actually returned errors incomparable might trip up code that does not expect it (e.g. a logger creating a map keyed by errors will panic on this).
//
// To simplify some handling, we do the following:
//   - We only allow one level of boxing. All exported functions first unbox its inputs.
//   - As all our functions first unbox its inputs, using a boxed error as a base error for wrapping will in reality use the unboxed version.
//   - The boxed versions also satisfy the same relevant interface as its unboxed counterparts.
//
// Note that the last point actually requires having multiple MakeErrorIncomparable - variants and boxing types, depending on the preserved interface.
// (Making everything dependent on the preserved interface would require C++ - style *templates* and cannot be done with Go's *generics*)

// NOTE: The *correct way* (that is, if the Go language had a non-trivial type system or templates instead of generics) to do this would be:
//
// type BoxableError_T[T error] interface{
//     T
//     Is(target error) bool
//     SupportsBoxingAsIncomparable
// }
//
// type UnboxableError_T[T UnboxableError] interface{
//     Unbox() T
// }
//
// type IncomparableError_T[T error] struct{
//		utils.MakeIncomparable
//		BoxableError_T[T]
// }
//
// Unfortunately, Go cannot do this.
// The issue with the actual code below is that
//   a) our type for boxed errors only promotes the methods of (and hence satisfies) the concrete (interface) type that we actually embed, not
//      the dynamic type that the embedded error actually satisfies. Go has no way to this except for defining a separate type for every possible option,
//      essentially copy&pasting the above for every value of T.
//   b) The UnboxableError interface has an Unbox() method that needs to return some type. Due to lack of contravariance of interface type,
//      implementations must match the signature exactly. This means that either the Unbox() methods throws away the information about what extended
//      interface the unboxed error satisfied or we have multiple incompatible interfaces. We opt for the former.

// BoxableError is a interface that errors must satisfy in order to be usable by [BoxErrorAsIncomparable].
//
// In order to actually work as intended, we require a suitable Is - method, which is defined on the *unboxed* error.
// The reason for this is that [errors.Is](err, target) has a hook in the form of such an Is - method, but this needs to be defined on err rather than target (which is not the way we would want).
// So we only allow boxing errors if the unboxed error has an appropriate Is(target error) method.
// We also ask for a specific tag method to explicitly opt-in.
// (accidential interface satisfaction is actually a real possibility here)
type BoxableError interface {
	error
	Is(target error) bool          // needs to unbox target. A valid implementation of e.Is(target) is `return e==UnboxError(target)`
	SupportsBoxingAsIncomparable() // dummy tag function for opt-in: this serves to "mark" types as compatible with MakeErrorIncomparable. We never call this.
}

// COMMENTED OUT: We only support the "plain version now":
// This means that boxing loses type information, but that is actually OK:
// After all, the only thing that users are supposed to do with boxed error err are
//    - comparing with errors.Is(sth, err)
//    - using it as the base of an error chain/tree.
// Both of the latter do not require any kind of extended interface.

/*
// boxableErrorWithData_any respectively boxableErrorWithData_any are
// the union of [ErrorWithData_any] respectively [ErrorWithData] and [boxableError].
// While we would prefer to just explicitly write interface{ErrorWithData_any;BoxableError},
// this is used for struct embedding, which is why it needs to have some (internal) name.
//
// NOTE: We haved refactored ErrorWithData_any to actually always include BoxableError,
// so this type alias becomes techically superfluous and it equals ErrorsWithData_any.
// We still write it explicitly here because this coupling is not neccessary at this point and
// BoxableError and ErrorWtihData_any are conceptually different.
// (We only did the refactoring to avoid some type assertion elsewhere)
type (
	boxableErrorWithData_any = interface {
		ErrorWithData_any
		BoxableError
	}

	boxableErrorWithData[StructType any] interface {
		ErrorWithData[StructType]
		BoxableError
	}
)

*/

// unboxableError is an internal interface satisfied by the incomparable errors.
//
// It allows to access the boxed comparable error and "tags" boxed types made incomparable by this package.
type unboxableError interface {
	Unbox() BoxableError
}

/*

// unboxableErrorWithData_any is an internal interface satisfied by incomparable errors.
//
// It allows to access the boxed comparable error and "tags" boxed types made incomparable by this package.
// As opposed to [unboxableError], this interface additionally requires a UnboxWithData_any() method that preserves the [ErrorWithData_any] interface.
type unboxableErrorWithData_any interface {
	unboxableError // subsume above
	UnboxWithData_any() boxableErrorWithData_any
}

// unboxableErrorWithData is an internal interface satisfied by incomparable errors.
//
// It allows to access the boxed comparable error and "tags" boxed types made incomparable by this package.
// As opposed to [unboxableError], this interface additionally requires UnboxWithData() and a UnboxWithData_any() methods
// that preserves the [ErrorWithData] and [ErrorWithData_any] interfaces.
type unboxableErrorWithData[StructType any] interface {
	unboxableErrorWithData_any // subsume above
	UnboxWithData() boxableErrorWithData[StructType]
}

*/

// incomparableError, incomparableErrorWithData_any respectively incomparableErrorWithData
// boxes an [error], [ErrorWithData_any] respectively [ErrorWithData], but is intentionally made to be incomparable.
// Note that a given incomparableError does not wrap (in the sense of error wrapping) its boxed error,
// but rather behaves like it wrt most purposes (in particular, it wraps whatever the boxed error wraps). Use errors.Is() for comparison.
//
// The aim of this is that exporting errors as one of these types will make plain comparison via the == operator fail at compile-time and force
// users to use errors.Is(). For "base errors", the latter is the only correct usage pattern (and actually what we expect users to always do).
//
// NOTE: This feature relies on exporting such base errors (that users are expected to compare against) as this concrete type, NOT as an interface.
type (
	// boxed BoxableError (i.e. plain error with some extra functionality to opt-in), but intentionally made incomparable
	incomparableError struct {
		// we just struct-embed Boxable error to promote Error, Is and SupportsBoxingAsIncomparable.
		// Indeed, promoting the Error and Is methods just does The Right Thing(tm) here.
		utils.MakeIncomparable // struct-embedded to make the type incomparable.
		BoxableError           // not error, because we want to promote Is(target error) bool and SupportsBoxingAsIncomparable().
	}

	/*
		// boxed [ErrorWithData_any], but intentionally made incomparable
		incomparableErrorWithData_any struct {
			// We struct-embedd the union of [ErrorWithData_any] and [BoxableError] to promote the methods of both interfaces:
			utils.MakeIncomparable   // struct-embedded to make the type incomparable
			boxableErrorWithData_any // union of [ErrorWithData_any] and [BoxableError] -- this interface type needs to have an (internal) name for struct-embedding to work.
		}

		// boxed [ErrorWithData_any], but intentionally made incomparable
		incomparableErrorWithData[StructType any] struct {
			// We struct-embedd the union of [ErrorWithData] and [BoxableError] to promote the methods of both interfaces.
			utils.MakeIncomparable           // struct-embedded to make the type incomparable
			boxableErrorWithData[StructType] // union of [ErrorWithData] and [BoxableError] -- this interface type needs to have an (internal) name for struct-embedding to work.
		}
	*/
)

func (incomp incomparableError) Unbox() BoxableError { return incomp.BoxableError }

// IsEqual is not really needed.
/*
func (incomp incomparableError) IsEqual(incomp2 incomparableError) bool {
	return incomp.BoxableError == incomp2.BoxableError
}
*/

/*

func (incomp incomparableErrorWithData_any) Unbox() BoxableError {
	return incomp.boxableErrorWithData_any
}
func (incomp incomparableErrorWithData[StructType]) Unbox() BoxableError {
	return incomp.boxableErrorWithData
}

func (incomp incomparableErrorWithData_any) UnboxWithData_any() boxableErrorWithData_any {
	return incomp.boxableErrorWithData_any
}
func (incomp incomparableErrorWithData[StructType]) UnboxWithData_any() boxableErrorWithData_any {
	return incomp.boxableErrorWithData
}

func (incomp incomparableErrorWithData[StructType]) UnboxWithData() boxableErrorWithData[StructType] {
	return incomp.boxableErrorWithData
}


func (incomp incomparableErrorWithData_any) IsEqual(incomp2 incomparableErrorWithData_any) bool {
	return incomp.boxableErrorWithData_any == incomp2.boxableErrorWithData_any
}

func (incomp incomparableErrorWithData[StructType]) IsEqual(incomp2 incomparableErrorWithData[StructType]) bool {
	return incomp.boxableErrorWithData == incomp2.boxableErrorWithData
}

*/

// UnboxError unboxes an error made incomparable by [BoxErrorAsIncomparable], returning the contained error.
//
// On non-unboxable errors, just returns e itself. In particular, this returns nil on nil input.
func UnboxError(e error) error {
	if errUnboxable, ok := e.(unboxableError); ok {
		return errUnboxable.Unbox()
	} else {
		return e
	}
}

/*
// UnboxErrorWithData_any does the same as [UnboxError], but preserves the [ErrorWithData_any] interface.
func UnboxErrorWithData_any(e ErrorWithData_any) ErrorWithData_any {
	if errUnboxable, ok := e.(unboxableErrorWithData_any); ok {
		return errUnboxable.UnboxWithData_any()
	} else {
		return e
	}
}

// UnboxErrorWithData does the same as [UnboxError], but preserves the [ErrorWithData] interface
func UnboxErrorWithData[StructType any](e interface{ ErrorWithData[StructType] }) ErrorWithData[StructType] {
	if errUnboxable, ok := e.(unboxableErrorWithData[StructType]); ok {
		return errUnboxable.UnboxWithData()
	} else {
		return e
	}
}
*/

/*
 NOTE: Design justification is outdated.
*/

// Note: We do not define a Box() SomeType *method* for boxable errors that should support this feature.
// This would be the obvious and the "feels right" way to do it, but it does not work well:
// The issue at hand is the return type and that this would need to be part of ErrorsWithData.
//
// We only expose errors that support this feature via the ErrorWithData or ErrorWithData_any interface, i.e.
// methods used to create ErrorsWithData actually return the newly created error in an interface.
// Consequently, such a Box() method would need to become a part of the interface itself
// (unless we want to force users to sprinkle type-assert all over the place).
//
// Now, we want the result of Box() to actually satisfy the same interface extending error as the original.
// Again, lack of interface co-/contravariance means that this would mean that we would need
// separate BoxError() incomparableError(), BoxError_any() incomparableErrorWithData_any etc.
// methods and we need to call the right one.
// Using a free function instead of a method somewhat alleviates this, as the "relevant" and correct method
// is just the one determined by the (static, interface) type used to store the original error.
//
// Also, this way lessens coupling the implementation(s) of the ErrorWithData or ErrorWithData_any interfaces
// to the boxing mechnism.

// NOTE: Only allowing one level of boxing is just done to simplify the Is - method.
// We currently panic on nil input due to the fact that the function returns a struct (which cannot be nil).
// this can be potentially confusing. We do not specify the behaviour in order to be able to change it later.
// (The alternative of returning a boxed nil may be useful in certain circumstances)

// NOTE: The BoxErrorAsIncomparable, BoxErrorWithDataAsIncomparable and BoxErrorWithDataAsIncomparable_any
// functions have been simplified by making use of the fact that ErrorWithData_any and ErrorWithData interfaces
// each contains the BoxableError interface.
// This is not neccessary conceptually, but simplifies things a bit:
//   - We would need to take the input e of type interface{ErrorWithData_any; BoxableError}
//     resp. interface{ErrorWithData[StructType]; BoxableError} otherwise.
//     Since the package user only has values stored in ErrorWithData_any, the user would need to type-assert to this type.
//   - We would need to type-assert the result of UnboxErrorWithData and UnboxErrorWithData_any in the function definitions.

// BoxErrorAsIncomparable returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
//
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
//
// Comparisons using errors.Is() will still work as intended. If e is already boxed, BoxErrorAsIncomparable(e) unboxes e beforehand to only get one layer.
// Calling this function on nil will panic.
func BoxErrorAsIncomparable(e BoxableError) incomparableError {
	if e == nil {
		panic(ErrorPrefix + "Called BoxErrorAsIncomparable on nil error")
	}
	// unbox the input, if possible
	e = UnboxError(e).(BoxableError)
	return incomparableError{BoxableError: e}
}

/*
// BoxErrorWithDataAsIncomparable_any returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
// This version preserves the ErrorsWithData_any interface.
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
// Calling this function on nil will panic.
func BoxErrorWithDataAsIncomparable_any(e ErrorWithData_any) incomparableErrorWithData_any {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_any on nil error")
	}
	e = UnboxErrorWithData_any(e)
	return incomparableErrorWithData_any{boxableErrorWithData_any: e}
}

// BoxErrorWithDataAsIncomparable returns a boxed version of the given error which is not comparable.
// Note that this function does not return an interface, but a struct containing an interface.
// This means that == will fail at compile-time (unless the returned value is stored in an interface -- then this causes a run-time panic or may give the wrong result)
// This version preserves the ErrorsWithData[StructType] interface.
//
// Comparison with errors.Is() will still work as intended. If e is already boxed, we unbox beforehand to only get one layer.
// Calling this function on nil will panic.
func BoxErrorWithDataAsIncomparable[StructType any](e ErrorWithData[StructType]) incomparableErrorWithData[StructType] {
	if e == nil {
		panic(ErrorPrefix + "Called MakeErrorIncomparable_struct on nil error")
	}
	e = UnboxErrorWithData(e)
	return incomparableErrorWithData[StructType]{boxableErrorWithData: e}
}

*/
