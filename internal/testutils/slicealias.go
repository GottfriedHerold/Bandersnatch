package testutils

// CheckSliceAlias returns true iff both x and y are both non-nil and share the same underlying array of positive capacity
func CheckSliceAlias[T any](x []T, y []T) bool {
	if x == nil || y == nil {
		return false
	}
	cx := cap(x)
	cy := cap(y)
	if cx == 0 || cy == 0 {
		return false
	}
	return &(x[0:cx][cx-1]) == &(y[0:cy][cy-1])
}
