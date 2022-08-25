package pointserializer

import (
	"errors"
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
	type Point = curvePoints.Point_axtw_subgroup
	const Cap = 31
	const size = 15
	var arr [Cap]Point
	var existingSlice []Point = arr[:]
	var sliceMaker DeserializeSliceMaker = UseExistingSlice(existingSlice[0:size])
	output, _, _ := sliceMaker(-1)
	if _, ok := output.(int); !ok {
		t.Fatalf("UseExistingSlice did not return int on caller error")
	}

	const targetLen = 9
	output, slice, err := sliceMaker(targetLen)
	outputReal := output.(int)
	if err != nil {
		t.Fatalf("UseExistingSlice returned unexpected error %v", err)
	}
	if outputReal != targetLen {
		t.Fatalf("UseExistingSlice returned unexpected value %v,", outputReal)
	}
	if slice == nil {
		t.Fatalf("UseExistingSlice returned nil slice")
	}
	if slice.Len() != targetLen {
		t.Fatalf("UseExistingSlice returned CurvePointSlice with wrong Len")
	}
	if &arr[4] != slice.GetByIndex(4) {
		t.Fatalf("UseExistingSlice returns wrong Slice")
	}

	const exceedingLen = 29
	output, _, err = sliceMaker(exceedingLen)
	_ = output.(int)
	if err == nil {
		t.Fatalf("UseExistingSlice did not return error on invalid slice length")
	}
	if !errors.Is(err, ErrInsufficientBufferForDeserialization) {
		t.Fatalf("UseExistingSlice returned wrong error")
	}
	// fmt.Printf("%v\n", err)

	output, _, err = sliceMaker(2 * exceedingLen)
	_ = output.(int)
	if err == nil {
		t.Fatalf("UseExistingSlice did not return error on invalid slice length")
	}
	if !errors.Is(err, ErrInsufficientBufferForDeserialization) {
		t.Fatalf("UseExistingSlice returned wrong error")
	}
	// fmt.Printf("%v\n", err)

}
