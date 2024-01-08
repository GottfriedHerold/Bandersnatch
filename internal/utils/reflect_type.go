package utils

import (
	"fmt"
	"reflect"
)

// TypeOfType[T]() returns the [reflect.Type] of T.
//
// As opposed to [reflect.TypeOf], this works with a type parameter rather than a passed value and
// also works for interface types T
func TypeOfType[T any]() reflect.Type {
	var t *T
	return reflect.TypeOf(t).Elem()
}

// NameOfType[T] gives returns a string describing the type parameter T.
// This is useful for diagnostics.
//
// Note that due to passing the type as a type parameter, this works if T is an interface type.
func NameOfType[T any]() string {
	return GetReflectName(TypeOfType[T]())
}

// IsNilable returns whether values of type t can be set to nil
//
// The behaviour when calling this on reflect.TypeOf(nil) or the zero value of reflect.Type is unspecified.
//
// NOTE: We currently panic in the latter case. Recall that reflect.TypeOf(nil) returns the zero value of reflect.Type, which is considered an invalid object.
// The issue is that the behaviour of the standard reflect library is kind-of suboptimal (that's why this function is even needed in the first place)
// and there are serious considerations of changing it in future Go versions. Hence we give no promises that we panic, since we might want to update the behaviour if the
// standard library changes.
func IsNilable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface,
		reflect.Pointer,
		reflect.Chan,
		reflect.Func,
		reflect.Slice,
		reflect.Map:
		return true
	case reflect.Invalid:
		panic("IsNilable called on invalid type")
	default:
		return false
	}
}

// IsTypeNilable[T] returns whether values of type T can be set to nil. As opposed to [IsNilable], this takes the argument as a type parameter.
func IsTypeNilable[T any]() bool {
	return IsNilable(TypeOfType[T]())
}

// GetReflectName tries to obtain a string representation of the given type using the reflection package.
// It covers more cases that plain c.Name() does (which only works for defined types and fails for e.g. pointers to defined types).
// We only use it for better diagnostic messages in testing.
func GetReflectName(c reflect.Type) (ret string) {
	// reflect.Type's  Name() only works for defined types, which
	// e.g. *Point_xtw_full is not. (Only Point_xtw_full is a defined type)
	ret = c.Name()
	if ret != "" {
		return
	}

	switch c.Kind() {
	case reflect.Pointer: // used to be called reflect.Ptr in Go1.17; reflect.Ptr is deprecated
		return "*" + GetReflectName(c.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%v]%v", c.Len(), GetReflectName(c.Elem()))
	case reflect.Slice:
		return fmt.Sprintf("[]%v", GetReflectName(c.Elem()))
	default:
		return "<<type with unknown name>>"
	}
}
