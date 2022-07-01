package utils

import (
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// TypeOfType[T]() returns the reflect.Type of T.
// As opposed to reflect.TypeOf, this works with a type parameter rather than
// a value and also works for interface types T
func TypeOfType[T any]() reflect.Type {
	var t *T
	return reflect.TypeOf(t).Elem()
}

// NameOfType gives returns a string describing the type T.
// This is useful for diagnostics.
// Note that this works if T is an interface type.
func NameOfType[T any]() string {
	return testutils.GetReflectName(TypeOfType[T]())
}

// IsNilable returns whether values of type t can be set to nil
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
