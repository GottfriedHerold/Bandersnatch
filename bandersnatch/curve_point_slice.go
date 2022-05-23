package bandersnatch

import "reflect"

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

type CurvePointSliceWrapper[PointType any, PointTypePtr interface {
	*PointType
	CurvePointPtrInterface
}] struct {
	v []PointType
}

func (w CurvePointSliceWrapper[PointType, PointTypePtr]) Len() int {
	return len(w.v)
}

func (w CurvePointSliceWrapper[PointType, PointTypePtr]) GetByIndex(n int) CurvePointPtrInterface {
	return PointTypePtr(&w.v[n])
}

func (w CurvePointSliceWrapper[PointType, PointTypePtr]) GetByIndexTyped(n int) PointTypePtr {
	return PointTypePtr(&w.v[n])
}

func AsCurvePointSlice[PointType any, PointTypePtr interface {
	*PointType
	CurvePointPtrInterface
}](v []PointType) (ret CurvePointSliceWrapper[PointType, PointTypePtr]) {
	ret.v = v
	return
}

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
