package pointserializer

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/curvePoints"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

var _ DeserializeSliceMaker = UseExistingSlice([]curvePoints.Point_axtw_subgroup{})
var _ DeserializeSliceMaker = CreateNewSlice[curvePoints.Point_axtw_subgroup]

// This test checks the following:
//
//	a) SerializeCurvePoints and individual SerializeCurvePoint write the same data
//	b) Roundtrip of SerializeCurvePoints and DeserializeCurvePoints
func TestSerializeCurvePoints(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	drng := rand.New(rand.NewSource(1))

	for _, serializer := range allTestMultiSerializers {
		// create num many points
		const num = 200
		var point [num]curvePoints.Point_xtw_subgroup
		for i := 0; i < num-1; i++ {
			point[i] = curvePoints.MakeRandomPointUnsafe_xtw_subgroup(drng)
		}
		point[num-1].SetNeutral()

		buf1.Reset()
		buf2.Reset()
		bytesWritten1, err1 := serializer.SerializeCurvePoints(&buf1, curvePoints.AsCurvePointSlice(point[:]))
		testutils.FatalUnless(t, err1 == nil, "Unexpected error %v", err1)

		var bytesWritten2 int = 0

		for i := 0; i < num; i++ {
			bytesJustWritten, err2 := serializer.SerializeCurvePoint(&buf2, &point[i])
			bytesWritten2 += bytesJustWritten
			testutils.FatalUnless(t, err2 == nil, "Unexpected error %v", err1)
		}

		testutils.FatalUnless(t, bytesWritten1 == bytesWritten2, "Individual writes and multi-write differ in number of bytes written")
		testutils.FatalUnless(t, bytes.Equal(buf1.Bytes(), buf2.Bytes()), "Individual writes and multi-write differ in the bytes written")

		var readBack1, readBack2 [num]curvePoints.Point_efgh_subgroup
		bytesRead1, errRead1 := serializer.DeserializeCurvePoints(&buf1, UntrustedInput, curvePoints.AsCurvePointSlice(readBack1[:]))
		bytesRead2, errRead2 := serializer.AsDeserializer().DeserializeCurvePoints(&buf2, UntrustedInput, curvePoints.AsCurvePointSlice(readBack2[:]))
		testutils.FatalUnless(t, bytesRead1 == bytesWritten1, "Did not read back as much as was written")
		testutils.FatalUnless(t, bytesRead2 == bytesWritten1, "Did not read back as much as was written")
		testutils.FatalUnless(t, errRead1 == nil, "unexpected read error1 %v", errRead1)
		testutils.FatalUnless(t, errRead2 == nil, "unexpected read error2 %v", errRead2)
		for i := 0; i < num; i++ {
			testutils.FatalUnless(t, readBack1[i].IsEqual(&point[i]), "Did not read back point")
			testutils.FatalUnless(t, readBack2[i].IsEqual(&point[i]), "Did not read back point")
		}
	}
}

// func

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
