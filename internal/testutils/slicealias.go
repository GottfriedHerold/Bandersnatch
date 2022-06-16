package testutils

// CheckSliceAlias returns true iff both x and y are both non-nil, have positive capacity and share the same backing array.
// Note: The slices do not have to start/end at the same point, so reslicing does not affect this (except when resliced to zero capacitiy).
func CheckSliceAlias[T any](x []T, y []T) bool {
	if x == nil || y == nil {
		return false
	}
	cx := cap(x)
	cy := cap(y)
	// Whether or not the underlying storage array is the same for capacity-0 slices
	// is undetectable anyway without looking into the internals of the slice implementation.
	if cx == 0 || cy == 0 {
		return false
	}
	// While reslicing may change the start, the end (extended to capacity) is invariant.
	// So we check the address of the last entries (last when extended to capacity)
	return &(x[0:cx][cx-1]) == &(y[0:cy][cy-1])
}
