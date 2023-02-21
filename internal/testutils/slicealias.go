package testutils

// NOTE: The "proper" way to do this would be (after checking 0-capacity/nil) to convert &x[0] and &y[0] to uintptr and do arithmetic.
// The issue here is that this is not guaranteed to work. In practical terms, this may fail if there is a gc run between the conversions that relocates the backing array (and modifies any pointers to it).
// Unfortunately, the spec for uintptr - arithmetic is vague.
// If one wants something that provably works according to spec, the relevant part of the spec is the set of allowed
// conversions unsafe.Pointer <-> uintptr. This gives a positive list of things that are allowed (special-cased patterns recognized by the compiler!)
// Alas, the listed cases do not really cover what we are doing here; the issue at hand is that this positive list only works because the compiler regonizes the pattern, so
// if the pattern is not present, usualy aguing about the language fails. The only thing I see that would work is
// p1, p2 := IDENTITY(uintptr(unsafe.Pointer(&x[0])), uintptr(unsafe.Pointer(&y[0]))),
// where IDENTITY(a uintptr, b uintptr) (uintptr, uintptr) is an identify function on pairs of uintprt that MUST BE IMPLEMENTED IN ASSEMBLY:
//
// """
// The compiler handles a Pointer converted to a uintptr in the argument list of a call to a function implemented in assembly by arranging that
// the referenced allocated object, if any, is retained and not moved until the call completes, even though from the types alone it would appear
// that the object is no longer needed during the call.
// """

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
