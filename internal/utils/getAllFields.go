package utils

import "reflect"

// Embedded struct pointers can be used to create struct-embedding cycles,
// thereby allowing field selectors such as A.A.A.A.A.A.A.A.A.A.x of arbitrary length;
// a naive recursive algorithm might therefore run into an endless loop.
// It is unclear what the answer should even be, since, semanically, the returned fields slice should have infinite length.
//
// We solve this by simply disregarding embedded struct-pointers altogether, even if they don't lead to cycles.
// For our application (the errorsWithData package), this is fine.

// AllFields returns all fields (including fields from embedded structs) in t.
//
// The returned StructField.Index is an index sequence relative to the passed t; use this to access the corresponding entry.
//   - For embedded structs, fields contains both the embedded struct itself as well as its fields.
//   - For embedded struct pointers, fields contains ONLY the embedded struct poiner, NOT the fields.
//   - For embedded non-structs, non-pointer, fields contains the embedded type.
//
// If an embedded struct pointer was encountered, embeddedStructPointer is set to true.
//
// This function panics if t.Kind() is not reflect.Struct
func AllFields(t reflect.Type) (fields []reflect.StructField, embeddedStructPointer bool) {
	fields = make([]reflect.StructField, 0)
	embeddedStructPointer = allFields(t, []int{}, &fields)
	return
}

// allFields is the (recursive) implementation of [AllFields]
func allFields(t reflect.Type, indexPrefix []int, result *[]reflect.StructField) (embeddedStructPointer bool) {

	if t.Kind() != reflect.Struct {
		panic(ErrorPrefix + "called allFields on non-struct type")
	}
	prefixLen := len(indexPrefix)
	numFields := t.NumField()
	for i := 0; i < numFields; i++ {
		var newField reflect.StructField = t.Field(i)
		newField.Index = make([]int, prefixLen+1)
		copy(newField.Index, indexPrefix)
		newField.Index[prefixLen] = i
		*result = append(*result, newField)

		if newField.Anonymous {
			switch newField.Type.Kind() {
			case reflect.Struct:
				newPrefix := append([]int{}, newField.Index...)
				embeddedStructPointer = allFields(newField.Type, newPrefix, result) || embeddedStructPointer
			case reflect.Pointer:
				if newField.Type.Elem().Kind() == reflect.Struct {
					embeddedStructPointer = true
					// We do *NOT* descend via allFields in this case
				}
			default:
				// do nothing
			}
		}
	}
	return
}
