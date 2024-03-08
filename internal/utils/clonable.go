package utils

import "reflect"

// Clonable is the generic interface for types with a type-preserving Clone method.
// Clone methods are supposed to return a (new pointer to a) copy of the receiver.
//
// NOTE: We use Clone for pointer receivers, so K is a pointer type. If we want a non-pointer self-copy functions, we usually call it makeCopy or sth.
type Clonable[K any] interface {
	Clone() K // returns a copy of itself
}

// AddressOfCopy makes a copy of the (non-pointer) argument and returns a pointer to it.
func AddressOfCopy[K any](in K) *K {
	return &in
}

// PointerToCopy creates a copy of (the value inside) val and returns a (reflected) pointer to the copy.
func PointerToCopy(val reflect.Value) (ret reflect.Value) {
	ret = reflect.New(val.Type())
	retDeref := ret.Elem()
	retDeref.Set(val)
	return
}
