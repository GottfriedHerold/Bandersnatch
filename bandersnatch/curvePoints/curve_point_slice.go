package curvePoints

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// CurvePointSlice is a joint interface for slices of CurvePoints or pointers to CurvePoints.
//
// This interface is needed (due to inadequacies of Go's type system) to make certain functions work with slices of concrete point types.
// Notably, we need methods taking a []CurvePoint, which does not work with generics (because as of Go1.19, methods cannot be generic)
// The alternative is for those methods to take (functions returning the i-th point) as arguments, but then we might just as well make it an interface.
//
// Furthermore, some instantiations of CurvePointSlice satisfy additional interfaces that enable vastly more efficient batch-operations.
type CurvePointSlice interface {
	GetByIndex(int) CurvePointPtrInterface
	Len() int
}

type BatchNormalizerForZ interface {
	BatchNormalizeForZ() (zeroIndices []int)
	CurvePointSlice
}

type BatchNormalizerForY interface {
	BatchNormalizeForY() (zeroIndices []int)
	CurvePointSlice
}

// GenericPointSlice is the most simple implementation of the CurvePointSlice interface.
// It is just a slice of interfaces.
type GenericPointSlice []CurvePointPtrInterface

// GetByIndex is just a wrapper for v[n]
func (v GenericPointSlice) GetByIndex(n int) CurvePointPtrInterface {
	return v[n]
}

// Len returns the length of the slice. It is just a wrapper to len.
func (v GenericPointSlice) Len() int {
	return len(v)
}

// curvePointSliceWrapper is an implementation of the CurvePointSlice interface using generics.
// Note that these as always created with the AsCurvePointSlice function.
// This means that v is essentially always a view of some user-provided slice.
type curvePointSliceWrapper[PointType any, PointTypePtr interface {
	*PointType
	CurvePointPtrInterface
}] struct {
	v []PointType
}

// Len returns the length of the slice
func (w curvePointSliceWrapper[PointType, PointTypePtr]) Len() int {
	return len(w.v)
}

// GetByIndex returns the n'th element of the slice. Note that it returns a pointer as an interface value.
func (w curvePointSliceWrapper[PointType, PointTypePtr]) GetByIndex(n int) CurvePointPtrInterface {
	return PointTypePtr(&w.v[n])
}

// GetByIndexTyped retains the tye information, but is unfortunately hard to use because it involves the generic type parameter.
// This is mostly provided for comparison benchmarks.

// GetByIndexTyped returns the n'th element of the slice. Note that it returns a (typed) pointer.
func (w curvePointSliceWrapper[PointType, PointTypePtr]) GetByIndexTyped(n int) PointTypePtr {
	return PointTypePtr(&w.v[n])
}

// curvePointPtrSliceWrapper is an implementation of the CurvePointSlice interface using generics.
// It is similar to curvePointSliceWrapper, but stores a slice of pointers (instead of an slice of values) internally.
// Note that PointTypePtr might be the CurvePointPtrInterface interface type itself.
//
// We generally prefer the curvePointSliceWrapper type over it, because that one guarantees that the individual v[i]'s don't alias.
type curvePointPtrSliceWrapper[PointTypePtr CurvePointPtrInterface] struct {
	v []PointTypePtr
}

// Len returns the length of the slice
func (w curvePointPtrSliceWrapper[PointTypePtr]) Len() int {
	return len(w.v)
}

// GetByIndex returns the n'th element of the slice. Note that it returns a pointer as an interface value.
func (w curvePointPtrSliceWrapper[PointTypePtr]) GetByIndex(n int) CurvePointPtrInterface {
	return w.v[n]
}

// GetByIndexTyped retains the tye information, but is unfortunately hard to use because it involves the generic type parameter.
// This is mostly provided for comparison benchmarks.

// GetByIndexTyped returns the n'th element of the slice. Note that it returns a (typed) pointer.
func (w curvePointPtrSliceWrapper[PointTypePtr]) GetByIndexTyped(n int) PointTypePtr {
	return w.v[n]
}

// AsCurvePointSlice takes as input a slice v of type []PointType and returns another slice of (interface) type CurvePointSlice.
// The returned value can be used to access (the backing array of) v via the CurvePointSlice interface.
// This is intended to wrap existing arrays/slices in order to use them in certain Batch-operations which expect a CurvePointSlice.
func AsCurvePointSlice[PointType any, PointTypePtr interface {
	*PointType
	CurvePointPtrInterface
}](v []PointType) CurvePointSlice {

	// We need to special-case certain choices of types to optimize for bulk-operations.
	// We have two options here: Either dispatch to separate types or
	// special case operations in curvePointSliceWrapper.
	// Both are annyoning.

	// switching on a generic parameter is not (yet?) possible in Go1.18
	// So we workaround by tons of completely stupid type-assertions.
	switch any(v).(type) {
	case []Point_axtw_full:
		return CurvePointSlice_axtw_full(any(v).([]Point_axtw_full))
	case []Point_axtw_subgroup:
		return CurvePointSlice_axtw_subgroup(any(v).([]Point_axtw_subgroup))
	case []Point_xtw_full:
		return CurvePointSlice_xtw_full(any(v).([]Point_xtw_full))
	case []Point_xtw_subgroup:
		return CurvePointSlice_xtw_subgroup(any(v).([]Point_xtw_subgroup))
	case []Point_efgh_full:
		return CurvePointSlice_efgh_full(any(v).([]Point_efgh_full))
	case []Point_efgh_subgroup:
		return CurvePointSlice_efgh_subgroup(any(v).([]Point_efgh_subgroup))
	default:
		return curvePointSliceWrapper[PointType, PointTypePtr]{v: v}
	}

}

// AsCurvePointPtrSlice returns a (concrete, non-exported) generic type rather than CurvePointPtrInterface. This is slightly more efficient.

// AsCurvePointSlice takes as input a slice v of type []PointTypePtr and returns another view on v that satisfies the CurvePointSlice interface.
// The returned value can be used to access (the backing array of) v via the CurvePointSlice interface.
// This is intended to wrap existing arrays/slices in order to use them in certain Batch-operations which expect a CurvePointSlice.
//
// NOTE: The return type is unspecified and might change. The only guarantee is that is satisfies CurvePointSlice.
//
// NOTE: This function takes a slice of pointers as input. In general, it is preferred to work with slices of values and use AsCurvePointSlice instead.
// One issue is that most functions that take arguments of type CurvePointSlice require that the indididual slice elements do not alias.
// This is automatically guaranteed for []PointType, but not for []PointTypePtr, were some v[i]'s might equal each other.
func AsCurvePointPtrSlice[PointTypePtr CurvePointPtrInterface](v []PointTypePtr) (ret curvePointPtrSliceWrapper[PointTypePtr]) {
	// We might special-case here as well.
	ret.v = v
	return
}

// ***************************************************************************************
// Generic version that takes a reflect.Type as input. Moved here from old version of DeserializeSlice
// This is currently unused outside of testing.

var curvePointPtrInterfaceType reflect.Type = utils.TypeOfType[CurvePointPtrInterface]() // reflect.Type of CurvePointPtrInterface

// reflectedPointSlice is an implementation of CurvePointSlice useing reflection.
// The main point is that we can create a CurvePointSlice from a reflect.Type and a size.
// This is useful for (reflection-using) tests (most of which were written before Go even had generics).
type reflectedPointSlice struct {
	Slice reflect.Value
	L     int
}

// GetByIndex returns the n'th element of the slice as an interface
func (rps reflectedPointSlice) GetByIndex(n int) CurvePointPtrInterface {
	return rps.Slice.Index(n).Addr().Interface().(CurvePointPtrInterface)
}

// Len returns the length of the slice
func (rps reflectedPointSlice) Len() int {
	return rps.L
}

// makePointSlice creates a new slice of given type and size as reflect.Value and an CurvePointSlice that can be used to access it.
// The passed pointType must be the reflect.Type of a concrete (non-pointer, non-interface) CurvePointType.
func makePointSlice(pointType reflect.Type, size int) (asCurvePointSlice CurvePointSlice, asInterface reflect.Value) {

	// TODO: Special case common reflect.Types: The following is the generic "default", which is horribly inefficient, thanks to Go.

	if pointType.Kind() == reflect.Interface {
		panic(fmt.Errorf(ErrorPrefix+"Called makePointSlice with a reflect.Type for the type %v, which is an interface type. The provided type must be a concrete type", utils.GetReflectName(pointType)))
	}

	var PtrType reflect.Type = reflect.PointerTo(pointType)
	if !PtrType.Implements(curvePointPtrInterfaceType) {
		panic(fmt.Errorf(ErrorPrefix+"Called makePointSlice with a type %v, where %v does not satisfy the CurvePointPtrInterface interface", pointType, PtrType))
	}

	var sliceValue reflect.Value = reflect.MakeSlice(reflect.SliceOf(pointType), size, size)
	return reflectedPointSlice{Slice: sliceValue, L: size}, sliceValue
}

// **************************************************************************************
// Special-case implementations of the CurvePointSlice interface for all our point types.
// **************************************************************************************

type CurvePointSlice_axtw_full []Point_axtw_full
type CurvePointSlice_axtw_subgroup []Point_axtw_subgroup

func (v CurvePointSlice_axtw_subgroup) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_axtw_full) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_axtw_subgroup) GetByIndexTyped(n int) *Point_axtw_subgroup {
	return &v[n]
}

func (v CurvePointSlice_axtw_full) GetByIndexTyped(n int) *Point_axtw_full {
	return &v[n]
}

func (v CurvePointSlice_axtw_subgroup) Len() int {
	return len(v)
}

func (v CurvePointSlice_axtw_full) Len() int {
	return len(v)
}

// **************************************************************************************

type CurvePointSlice_xtw_subgroup []Point_xtw_subgroup
type CurvePointSlice_xtw_full []Point_xtw_full

func (v CurvePointSlice_xtw_subgroup) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_xtw_full) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_xtw_subgroup) Len() int {
	return len(v)
}

func (v CurvePointSlice_xtw_full) Len() int {
	return len(v)
}

func (v CurvePointSlice_xtw_subgroup) GetByIndexTyped(n int) *Point_xtw_subgroup {
	return &v[n]
}

func (v CurvePointSlice_xtw_full) GetByIndexTyped(n int) *Point_xtw_full {
	return &v[n]
}

// **************************************************************************************

type CurvePointSlice_efgh_subgroup []Point_efgh_subgroup
type CurvePointSlice_efgh_full []Point_efgh_full

func (v CurvePointSlice_efgh_subgroup) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_efgh_full) GetByIndex(n int) CurvePointPtrInterface {
	return &v[n]
}

func (v CurvePointSlice_efgh_subgroup) GetByIndexTyped(n int) *Point_efgh_subgroup {
	return &v[n]
}

func (v CurvePointSlice_efgh_full) GetByIndexTyped(n int) *Point_efgh_full {
	return &v[n]
}

func (v CurvePointSlice_efgh_subgroup) Len() int {
	return len(v)
}

func (v CurvePointSlice_efgh_full) Len() int {
	return len(v)
}
