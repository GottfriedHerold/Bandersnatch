package pointserializer

import (
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

type testMultiSerializer = multiSerializer[pointSerializerXY, *pointSerializerXY]

var _ CurvePointSerializerModifyable = &multiSerializer[pointSerializerXY, *pointSerializerXY]{}
var _ CurvePointDeserializerModifyable = &multiDeserializer[pointSerializerXY, *pointSerializerXY]{}

var _ DeserializeSliceMaker = UseExistingSlice([]curvePoints.Point_axtw_subgroup{})
var _ DeserializeSliceMaker = CreateNewSlice[curvePoints.Point_axtw_subgroup]

func TestCreateNewSlice(t *testing.T) {
	type Point = curvePoints.Point_axtw_subgroup
	const length = 31
	var sliceMaker DeserializeSliceMaker = CreateNewSlice[Point]

	output, _, _ := sliceMaker(-1)
	outputReal := output.([]Point)

	// Failure here would be ok actually. This is just to test the (internally) expected behaviour.
	if outputReal != nil {
		t.Fatalf("CreateNewSlice did not return nil slice on caller error")
	}

	output, slice, err := sliceMaker(length)
	outputReal = output.([]Point)
	if err != nil {
		t.Fatalf("CreateNewSlice returned unexpected error: %v", err)
	}
	if outputReal == nil {
		t.Fatalf("CreateNewSlice returned nil")
	}
	if L := len(outputReal); L != length {
		t.Fatalf("Slice created by CreateNewSlice had unexpected length: expected %v, got %v", length, L)
	}
	if slice == nil {
		t.Fatalf("CurvePointSlice creted by CreateNewSlice was nil")
	}
	if L := slice.Len(); L != length {
		t.Fatalf("CurvePointSlice created by CreateNewSlice had unexpected length: expected %v, got %v", length, L)
	}
	testutils.Assert(length >= 5)
	P1 := &outputReal[4]
	P2 := slice.GetByIndex(4)
	if P1 != P2 {
		t.Fatalf("CurvePointSlice and plain slice created by CreateNewSlice are incompatible")
	}
}

func TestUseExistingSlice(t *testing.T) {
	
}
