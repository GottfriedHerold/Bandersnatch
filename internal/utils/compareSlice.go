package utils

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
