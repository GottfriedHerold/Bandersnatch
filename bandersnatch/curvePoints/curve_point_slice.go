package curvePoints

import (
	"fmt"
	"reflect"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

// CurvePointSlice is a joint interface for slices of CurvePoints or pointers to CurvePoints.
//
// This interface is needed (due to inadequacies of Go's type system) to make certain functions work with slices of concrete point types.
type CurvePointSlice interface {
	GetByIndex(int) CurvePointPtrInterface
	Len() int
}

type GenericPointSlice []CurvePointPtrInterface

func (v GenericPointSlice) GetByIndex(n int) CurvePointPtrInterface {
	return v[n]
}

func (v GenericPointSlice) Len() int {
	return len(v)
}

type curvePointSliceWrapper[PointType any, PointTypePtr interface {
	*PointType
	CurvePointPtrInterface
}] struct {
	v []PointType
}

func (w curvePointSliceWrapper[PointType, PointTypePtr]) Len() int {
	return len(w.v)
}

func (w curvePointSliceWrapper[PointType, PointTypePtr]) GetByIndex(n int) CurvePointPtrInterface {
	return PointTypePtr(&w.v[n])
}

func (w curvePointSliceWrapper[PointType, PointTypePtr]) GetByIndexTyped(n int) PointTypePtr {
	return PointTypePtr(&w.v[n])
}

type curvePointPtrSliceWrapper[PointTypePtr CurvePointPtrInterface] struct {
	v []PointTypePtr
}

func (w curvePointPtrSliceWrapper[PointTypePtr]) Len() int {
	return len(w.v)
}

func (w curvePointPtrSliceWrapper[PointTypePtr]) GetByIndex(n int) CurvePointPtrInterface {
	return w.v[n]
}

func (w curvePointPtrSliceWrapper[PointTypePtr]) GetByIndexTyped(n int) PointTypePtr {
	return w.v[n]
}

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

func AsCurvePointPtrSlice[PointTypePtr CurvePointPtrInterface](v []PointTypePtr) (ret curvePointPtrSliceWrapper[PointTypePtr]) {
	// We might special-case here as well.
	ret.v = v
	return
}

// ***************************************************************************************
// Generic version that takes a reflect.Type as input. Moved here from old version of DeserializeSlice

var curvePointPtrInterfaceType reflect.Type = utils.TypeOfType[CurvePointPtrInterface]() // reflect.Type of CurvePointPtrInterface

type reflectedPointSlice struct {
	Slice reflect.Value
	L     int
}

func (rps reflectedPointSlice) GetByIndex(n int) CurvePointPtrInterface {
	return rps.Slice.Index(n).Addr().Interface().(CurvePointPtrInterface)
}

func (rps reflectedPointSlice) Len() int {
	return rps.L
}

func makePointSlice(pointType reflect.Type, size int) (asCurvePointSlice CurvePointSlice, asInterface any) {

	// TODO: Special case common reflect.Types: The following is the generic "default", which is horribly inefficient, thanks to Go.

	if pointType.Kind() == reflect.Interface {
		panic(fmt.Errorf(ErrorPrefix+"Called makePointSlice with a reflect.Type for the type %v, which is an interface type. The provided type must be a concrete type", testutils.GetReflectName(pointType)))
	}

	var PtrType reflect.Type = reflect.PointerTo(pointType)
	if !PtrType.Implements(curvePointPtrInterfaceType) {
		panic(fmt.Errorf(ErrorPrefix+"Called makePointSlice with a type %v, where %v does not satisfy the CurvePointPtrInterface interface", pointType, PtrType))
	}

	var sliceValue reflect.Value = reflect.MakeSlice(reflect.SliceOf(pointType), size, size)
	return reflectedPointSlice{Slice: sliceValue, L: size}, sliceValue.Interface()
}

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

////////////// OLD CODE:

/*

// This function takes a slice v of curve points and returns a pointer to v[n] as CurvePointPtrInterface. It also is a testament to why Go's type system really needs generics.

// getElementFromCurvePointSlice returns a pointer to the n'th element of v as a CurvePointPtrInterface
// v can be any of:
// slice of concrete Point type []Point_xtw_full
// Pointer-to slice of concrete Point type *[]Point_xtw_full
// slice of pointer to concrete Point type []*Point_xtw_full
// Pointer-to slice of pointer to concrete Point type *[]*Point_xtw_full
// array of pointer to concrete Point type [2]*Point_xtw_full
// Pointer-to array of concrete Point types *[2]Point_xtw_full
// Pointer-to array of pointers to concrete point types *[2]*Point_xtw_full
// slice of interfaces holding curve point ptrs []CurvePointPtrInterface
// Ptr-To slice of interfaces holding curve point ptrs *[]CurvePointPtrInterface
// array of interfaces holding curve point ptrs [2]CurvePointPtrInterface
// Ptr-To array of interfaces holding curve point ptrs *[2]CurvePointPtrInterface
//
// (The only thing missing is [2]concrete type -- this would copy and does indeed not work)
func getElementFromCurvePointSlice(v interface{}, n int) CurvePointPtrInterface {
	switch v := v.(type) {

	case []Point_xtw_subgroup:
		return &v[n]
	case []Point_xtw_full:
		return &v[n]
	case []Point_axtw_full:
		return &v[n]
	case []Point_axtw_subgroup:
		return &v[n]
	case []Point_efgh_subgroup:
		return &v[n]
	case []Point_efgh_full:
		return &v[n]
	case []CurvePointPtrInterface:
		return v[n]
	case []CurvePointPtrInterfaceRead:
		return v[n].(CurvePointPtrInterface)
	case []CurvePointPtrInterfaceWrite:
		return v[n].(CurvePointPtrInterface)

	default:
		return getElementFromCurvePointSliceGeneral(v, n)
	}
}

func getElementFromCurvePointSliceGeneral(v interface{}, n int) CurvePointPtrInterface {
	v_type := reflect.TypeOf(v)
	v_ref := reflect.ValueOf(v)
	if v_ref.Kind() == reflect.Ptr {
		v_ref = v_ref.Elem()
		v_type = v_type.Elem()
	}

	var elemType reflect.Type = v_type.Elem()
	elem := v_ref.Index(n)
	switch elemType.Kind() {
	case reflect.Struct:
		if !elem.CanAddr() {
			panic("bandersnatch / curve point array/slice: cannot take address of element. Did you pass an array to getElementFromCurvePointSlice? If so, use a slice or take the adress of the array.")
		}
		return elem.Addr().Interface().(CurvePointPtrInterface)
	case reflect.Ptr:
		return elem.Interface().(CurvePointPtrInterface)
	case reflect.Interface:
		return elem.Interface().(CurvePointPtrInterface)
	default:
		panic("elements of Slice/array passed to getElemFromCurvePointSlice is not struct, pointer or interface")
	}
}
*/
