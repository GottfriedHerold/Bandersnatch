package testutils

// CheckSliceAlias returns true iff both x and y are both non-nil, have positive capacity and share the same backing array (with same end).
// Note: The slices do not have to start/end at the same point, so usual reslicing does not affect this (except when resliced to zero capacitiy or using full slice notation).
//
// This function may fail to detect aliasing if the rarely used full slice notation x[start:end:max] is used to change the capacity of the slice to be smaller than
// the backing array.
func CheckSliceAlias[T any](x []T, y []T) bool {
	if x == nil || y == nil {
		return false
	}
	capX := cap(x)
	capY := cap(y)
	// Whether or not the underlying storage array is the same for capacity-0 slices
	// is undetectable anyway without looking into the internals of the slice implementation.
	if capX == 0 || capY == 0 {
		return false
	}
	// While reslicing may change the start, the end (extended to capacity) is invariant.
	// So we check the address of the last entries (last when extended to capacity)
	return &(x[0:capX][capX-1]) == &(y[0:capY][capY-1])
}
