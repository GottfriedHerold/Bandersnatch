package utils

import "fmt"

// Functionality scheduled for inclusion in Go's standard library,

// CompareSlices compares two slices for equality.
//
// Note: nil != empty slice, nil == nil here.
func CompareSlices[T comparable](x []T, y []T) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return false
	}
	if len(x) != len(y) {
		return false
	}
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// AssertSliceEquals(slice, element0...) asserts that
// slice == []T{element0, element1, ...} and panics otherwise.
// This is intended to be used in init() functions or global variables to create guards in the code.
//
// A typical use-case is having var allNodeTypes = []Node{Node1, Node2, Node3} (for an "enum" - type Node)
// Then some code with switch(nodeType){case Node1, Node2, Node3}
// can have some AssertSliceEquals(allNodeTypes, Node1, Node2, Node3) in a global init()-function close by.
func AssertSliceEquals[T comparable](x []T, y ...T) {
	if x == nil {
		panic("Called AssureSliceEquals on nil slice")
	}
	if len(x) != len(y) {
		panic("AssureSliceEquals called with #args != slice length")
	}
	for i, entry := range x {
		if entry != y[i] {
			panic(fmt.Errorf("AssureSliceEquals detected differing entries %v and %v", entry, y[i]))
		}
	}
}
