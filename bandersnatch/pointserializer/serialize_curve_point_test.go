package pointserializer

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

type testMultiSerializer = multiSerializer[pointSerializerXY, *pointSerializerXY]

var _ CurvePointSerializerModifyable = &multiSerializer[pointSerializerXY, *pointSerializerXY]{}
var _ CurvePointDeserializerModifyable = &multiDeserializer[pointSerializerXY, *pointSerializerXY]{}

var _ curvePoints.CurvePointSlice = reflectedPointSlice{}

func TestMakeCurvePointSlice(t *testing.T) {
	const size = 31
	pointTypeAXTW_Full := utils.TypeOfType[curvePoints.Point_axtw_full]()
	someSlice, sliceInInterface := makePointSlice(pointTypeAXTW_Full, size)
	real := sliceInInterface.([]curvePoints.Point_axtw_full)

	if len(real) != size {
		t.Fatalf("Size of interface is wrong")
	}
	if someSlice.Len() != size {
		t.Fatalf("Size of CurvePointSlice is wrong")
	}
	for i := 0; i < size; i++ {
		if &real[i] != someSlice.GetByIndex(i) {
			t.Fatalf("someSlice does not refer to the actual slice")
		}
	}
}
