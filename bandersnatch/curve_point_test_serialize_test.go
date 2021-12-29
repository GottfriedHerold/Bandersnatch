package bandersnatch

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
)

/*
	TODO: More tests on failing cases
*/

func checkfun_NaP_serialization(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	infinite := s.Flags[0].CheckFlag(Case_infinite)
	expect_error := singular || infinite
	var buf bytes.Buffer
	var bytes_written int
	var err error
	var gotErrNaP, gotErrInfinity bool

	encounted_NaP_error := wasInvalidPointEncountered(func() { bytes_written, err = s.Points[0].SerializeLong(&buf) })
	gotErrInfinity = (err == ErrCannotSerializePointAtInfinity)
	gotErrNaP = (err == ErrCannotSerializeNaP)

	if bytes_written != buf.Len() {
		return false, "Number of bytes written in deserialization was reported wrongly: bytes reported = " + strconv.Itoa(bytes_written) + " Actually written = " + strconv.Itoa(buf.Len())
	}

	if encounted_NaP_error && !singular {
		return false, "NaP handler was called when calling SerializeLong on a non-NaP point"
	}
	if !encounted_NaP_error && singular {
		return false, "NaP handler was not called when calling SerializeLong on a NaP"
	}
	if expect_error {
		if err == nil {
			return false, "SerializeLong did not give an error even though it should"
		}
		if singular && infinite {
			// might actually be OK, but we bail out for now.
			panic("Error in testing framework: sample was flagged as both infinite and singular")
		}
		if singular && !gotErrNaP {
			return false, "Did not get NaP error when calling SerializeLong on NaP"
		}
		if infinite && !gotErrInfinity {
			return false, "Did not get Infinite error when calling SerializeLong on Infinite point"
		}
	} else {
		if err != nil {
			// Note: s.Points[0] might NOT be in the subgroup.
			return false, "SerializeLong gave an error even though the point was neither infinite nor a NaP"
		}
		if bytes_written != 64 {
			return false, "unexpeced number of bytes written in SerializeLong. Number written was " + strconv.Itoa(bytes_written)
		}
	}

	buf.Reset()

	encounted_NaP_error = wasInvalidPointEncountered(func() { bytes_written, err = s.Points[0].SerializeShort(&buf) })
	gotErrInfinity = (err == ErrCannotSerializePointAtInfinity)
	gotErrNaP = (err == ErrCannotSerializeNaP)

	if bytes_written != buf.Len() {
		return false, "Number of bytes written in deserialization was reported wrongly: bytes reported = " + strconv.Itoa(bytes_written) + " Actually written = " + strconv.Itoa(buf.Len())
	}

	if encounted_NaP_error && !singular {
		return false, "NaP handler was called when calling SerializeShort on a non-NaP point"
	}
	if !encounted_NaP_error && singular {
		return false, "NaP handler was not called when calling SerializeShort on a NaP"
	}
	if expect_error {
		if err == nil {
			return false, "SerializeShort did not give an error even though it should"
		}
		if singular && infinite {
			// might actually be OK, but we bail out for now.
			panic("Error in testing framework: sample was flagged as both infinite and singular")
		}
		if singular && !gotErrNaP {
			return false, "Did not get NaP error when calling SerializeShort on NaP"
		}
		if infinite && !gotErrInfinity {
			return false, "Did not get Infinite error when calling SerializeShort on Infinite point"
		}
	} else {
		if err != nil {
			// Note: s.Points[0] might NOT be in the subgroup.
			return false, "SerializeShort gave an error even though the point was neither infinite nor a NaP"
		}
		if bytes_written != 32 {
			return false, "unexpeced number of bytes written in SerializeShort Number written was " + strconv.Itoa(bytes_written)
		}
	}
	return true, ""
}

func checkfun_serialization_type_consistency(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	infinite := s.Flags[0].CheckFlag(Case_infinite)
	if infinite || singular {
		return true, "" // converted by checkfun_NaP_serialization. No need to complicate things here
	}
	var buf1, buf2 bytes.Buffer

	point_axtw := s.Points[0].AffineExtended()
	_, err1 := s.Points[0].SerializeLong(&buf1)
	_, err2 := point_axtw.SerializeLong(&buf2)

	if err1 != nil || err2 != nil {
		return false, "Unexpected error in checkfun_type_consistency. Refer to output of checkfun_NaP_serialization"
	}

	if buf1.Len() != buf2.Len() {
		return false, "SerializeLong did not write same number of bytes depending on receiver type"
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		return false, "SerializeLong did not output the same bytes depending on receiver type"
	}

	buf1.Reset()
	buf2.Reset()
	_, err1 = s.Points[0].SerializeShort(&buf1)
	_, err2 = point_axtw.SerializeShort(&buf2)

	if err1 != nil || err2 != nil {
		return false, "Unexpected error in checkfun_type_consistency. Refer to output of checkfun_NaP_serialization"
	}

	if buf1.Len() != buf2.Len() {
		return false, "SerializeShort did not write same number of bytes depending on receiver type"
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		return false, "SerializeShort did not output the same bytes depending on receiver type"
	}

	return true, ""
}

// Checks roundtrip-capabilities on the happy path, i.e. if the points are in the correct subgroup.
func checkfun_serialization_roundtrip(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	infinite := s.Flags[0].CheckFlag(Case_infinite)
	not_in_goodgroup := s.Flags[0].CheckFlag(Case_outside_goodgroup)
	if infinite || singular {
		return true, "" // infinte and singular are convered by checkfun_NaP_serialization. No need to complicate things here
	}
	if not_in_goodgroup {
		return true, "" // Untrusted deserialization is intended to fail for those. We have a separate test for this.
	}

	var point_out CurvePointPtrInterfaceRead = s.Points[0].Clone()
	var point_in CurvePointPtrInterface = MakeCurvePointPtrInterfaceFromType(GetPointType(point_out))
	var buf bytes.Buffer
	var err error
	var bytes_read int

	buf.Reset()
	_, err = point_out.SerializeLong(&buf)
	if err != nil {
		return false, "error during SerializeLong: " + err.Error()
	}
	_, err = point_in.DeserializeLong(&buf, UntrustedInput)
	if err != nil {
		return false, "error during untrusted DeserializeLong " + err.Error()
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for untrusted (De)SerializeLong"
	}

	buf.Reset()
	_, err = point_out.SerializeLong(&buf)
	if err != nil {
		return false, "error during SerializeLong: " + err.Error()
	}
	_, err = point_in.DeserializeLong(&buf, TrustedInput)
	if err != nil {
		return false, "error during trusted DeserializeLong " + err.Error()
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for trusted (De)SerializeLong"
	}

	buf.Reset()
	_, err = point_out.SerializeLong(&buf)
	if err != nil {
		return false, "error during SerializeLong: " + err.Error()
	}
	bytes_read, err = point_in.DeserializeAuto(&buf, UntrustedInput)
	if err != nil {
		return false, "error during untrusted auto-DeserializeLong " + err.Error()
	}
	if bytes_read != 64 {
		return false, "Did not read correct number of bytes during untrusted auto-DeserializeLong"
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for untrusted AUto-(De)SerializeLong"
	}

	buf.Reset()
	_, err = point_out.SerializeLong(&buf)
	if err != nil {
		return false, "error during SerializeLong: " + err.Error()
	}
	bytes_read, err = point_in.DeserializeAuto(&buf, TrustedInput)
	if err != nil {
		return false, "error during trusted auto-DeserializeLong " + err.Error()
	}
	if bytes_read != 64 {
		return false, "Did not read correct number of bytes during trusted auto-DeserializeLong"
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for trusted AUto-(De)SerializeLong"
	}

	buf.Reset()
	_, err = point_out.SerializeShort(&buf)
	if err != nil {
		return false, "error during SerializeShort: " + err.Error()
	}
	_, err = point_in.DeserializeShort(&buf, UntrustedInput)
	if err != nil {
		return false, "error during untrusted DeserializeShort " + err.Error()
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for untrusted (De)SerializeShort"
	}

	buf.Reset()
	_, err = point_out.SerializeShort(&buf)
	if err != nil {
		return false, "error during SerializeShort: " + err.Error()
	}
	_, err = point_in.DeserializeShort(&buf, TrustedInput)
	if err != nil {
		return false, "error during trusted DeserializeShort " + err.Error()
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for trusted (De)SerializeShort"
	}

	buf.Reset()
	_, err = point_out.SerializeShort(&buf)
	if err != nil {
		return false, "error during SerializeShort: " + err.Error()
	}
	bytes_read, err = point_in.DeserializeAuto(&buf, UntrustedInput)
	if err != nil {
		return false, "error during untrusted Auto-DeserializeShort " + err.Error()
	}
	if bytes_read != 32 {
		return false, "Did not read correct number of bytes during untrusted auto-DeserializeShort"
	}
	if !point_in.IsEqual(point_out) {
		fmt.Println(point_in.String())
		return false, "Rountrip error for untrusted Auto-(De)SerializeShort"
	}

	buf.Reset()
	_, err = point_out.SerializeShort(&buf)
	if err != nil {
		return false, "error during SerializeShort: " + err.Error()
	}
	bytes_read, err = point_in.DeserializeAuto(&buf, TrustedInput)
	if err != nil {
		return false, "error during trusted Auto-DeserializeShort " + err.Error()
	}
	if bytes_read != 32 {
		return false, "Did not read correct number of bytes during untrusted auto-DeserializeShort"
	}
	if !point_in.IsEqual(point_out) {
		return false, "Rountrip error for trusted Auto-(De)SerializeShort"
	}

	return true, ""
}

func checkfun_rountrip_modulo2torsion(s TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	singular := s.Flags[0].CheckFlag(Case_singular)
	infinite := s.Flags[0].CheckFlag(Case_infinite)
	not_in_goodgroup := s.Flags[0].CheckFlag(Case_outside_goodgroup)
	torsion2 := s.Flags[0].CheckFlag(Case_2torsion)
	if infinite || singular || not_in_goodgroup {
		return true, "" // We only care about points in the p253 subgroup, modifying them as needed to get the cosets. The behaviour on the 2-torsion is
	}

	// point_outN, point_outA, point_outE are in different cosets of full curve / p253
	var point_outN CurvePointPtrInterfaceRead_FullCurve = s.Points[0].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	pointType := GetPointType(point_outN)
	point_outA := MakeCurvePointPtrInterfaceFromType(pointType)
	point_outE := MakeCurvePointPtrInterfaceFromType(pointType)

	point_inN := MakeCurvePointPtrInterfaceFromType(pointType)
	point_inA := MakeCurvePointPtrInterfaceFromType(pointType)
	point_inE := MakeCurvePointPtrInterfaceFromType(pointType)

	// serves as a sentinel value to check whether the value was overwritten (on Deserialization failure, it should not be)
	var sentinel_point Point_xtw
	sentinel_point.SetFrom(&example_generator_xtw)

	point_inN.SetFrom(&sentinel_point)
	point_inA.SetFrom(&sentinel_point)
	point_inE.SetFrom(&sentinel_point)

	// This is just an arbitrary non-2-torsion value outside the good subgroup.
	point_outE.SetFrom(&example_generator_xtw)
	point_outE.DoubleEq()
	point_outE.AddEq(&exceptionalPoint_1_xtw)

	point_outA.Add(point_outN, &orderTwoPoint_xtw)
	if !torsion2 {
		point_outE.Add(point_outN, &exceptionalPoint_1_xtw) // this would fail if s.Points[0] == point_out was 2-torsion, because a) we hit a potential singular case of addition and b) pointType might not be able to represent points at infinity.
	}
	if point_outA.IsNaP() || (!torsion2 && point_outE.IsNaP()) {
		panic("checkfun_roundtrip_modulo2torsion: Unexpected NaP")
	}

	// Note: point_outE is always outside the good subgroup, even if torsion2 == true. It is also NOT a point at infinity.

	var bufN, bufA, bufE bytes.Buffer
	var errN, errA, errE error

	_, errN = point_outN.SerializeLong(&bufN)
	_, errA = point_outA.SerializeLong(&bufA)
	_, errE = point_outE.SerializeLong(&bufE)

	if errN != nil || errA != nil {
		return false, "SerializeLong unexpectedly failed"
	}
	if errE != nil {
		return false, "SerializeLong failed when trying to SerializeLong outside good subgroup. Error was " + errE.Error() // This is not "really" a problem.
	}

	_, errN = point_inN.DeserializeLong(&bufN, UntrustedInput)
	_, errA = point_inA.DeserializeLong(&bufA, UntrustedInput)
	_, errE = point_inE.DeserializeLong(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "untrusted deserialization unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	if errE != ErrNotInSubgroup {
		return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeLong"
	}

	if !point_inE.IsEqual_FullCurve(&sentinel_point) {
		return false, "failed DeserializeLong overwrote point"
	}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "untrusted DeserializeLong had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "untrusted DeserializeLong did not result in loss of P vs. P+A information"
	}

	// repeat with trusted; this time without E
	bufN.Reset()
	bufA.Reset()
	// bufE.Reset()
	point_outN.SerializeLong(&bufN)
	point_outA.SerializeLong(&bufA)
	// point_outE.SerializeLong(&bufE)

	_, errN = point_inN.DeserializeLong(&bufN, TrustedInput)
	_, errA = point_inA.DeserializeLong(&bufA, TrustedInput)
	// _, errE = point_inE.DeserializeLong(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "trusted DeserializeLong unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	// if errE != ErrNotInSubgroup {
	// 	return false, "Did not get Not-In-Subgroup error upon trusted DeserializeLong"
	//}

	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "Trusted DeserializeLong had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "trusted DeserializeLong did not result in loss of P vs. P+A information"
	}

	// repeat with auto, untrusted.
	bufN.Reset()
	bufA.Reset()
	bufE.Reset()
	point_outN.SerializeLong(&bufN)
	point_outA.SerializeLong(&bufA)
	point_outE.SerializeLong(&bufE)

	_, errN = point_inN.DeserializeAuto(&bufN, UntrustedInput)
	_, errA = point_inA.DeserializeAuto(&bufA, UntrustedInput)
	_, errE = point_inE.DeserializeAuto(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "untrusted DeserializeAuto unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	if errE != ErrNotInSubgroup {
		return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeAuto"
	}

	if !point_inE.IsEqual_FullCurve(&sentinel_point) {
		return false, "failed DeserializeAuto overwrote point"
	}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "untrusted DeserializeAuto had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "untrusted DeserializeAUto did not result in loss of P vs. P+A information"
	}

	// repeat with auto, trusted.
	bufN.Reset()
	bufA.Reset()
	// bufE.Reset()
	point_outN.SerializeLong(&bufN)
	point_outA.SerializeLong(&bufA)
	// point_outE.SerializeLong(&bufE)

	_, errN = point_inN.DeserializeAuto(&bufN, UntrustedInput)
	_, errA = point_inA.DeserializeAuto(&bufA, UntrustedInput)
	// _, errE = point_inE.DeserializeAuto(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "trusted DeserializeAuto unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	// if errE != ErrNotInSubgroup {
	// 	return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeAuto"
	// }

	// if !point_inE.IsEqual_FullCurve(&example_generator_xtw) {
	//	return false, "failed DeserializeAuto overwrote point"
	// }
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "trusted DeserializeAuto had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "trusted DeserializeAUto did not result in loss of P vs. P+A information"
	}

	// repeat with short, untrusted.
	bufN.Reset()
	bufA.Reset()
	bufE.Reset()
	point_outN.SerializeShort(&bufN)
	point_outA.SerializeShort(&bufA)
	point_outE.SerializeShort(&bufE)

	_, errN = point_inN.DeserializeShort(&bufN, UntrustedInput)
	_, errA = point_inA.DeserializeShort(&bufA, UntrustedInput)
	_, errE = point_inE.DeserializeShort(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "untrusted DeserializeShort unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	if errE != ErrXNotInSubgroup {
		return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeShort"
	}

	if !point_inE.IsEqual_FullCurve(&sentinel_point) {
		return false, "failed DeserializeShort overwrote point"
	}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "untrusted DeserializeShort had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "untrusted DeserializeShort did not result in loss of P vs. P+A information"
	}

	// repeat with short, trusted.
	bufN.Reset()
	bufA.Reset()
	// bufE.Reset()
	point_outN.SerializeShort(&bufN)
	point_outA.SerializeShort(&bufA)
	// point_outE.SerializeShort(&bufE)

	_, errN = point_inN.DeserializeShort(&bufN, TrustedInput)
	_, errA = point_inA.DeserializeShort(&bufA, TrustedInput)
	// _, errE = point_inE.DeserializeShort(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "trusted DeserializeShort unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	// if errE != ErrNotInSubgroup {
	// 	return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeShort"
	// }

	// if !point_inE.IsEqual_FullCurve(&example_generator_xtw) {
	//  	return false, "failed DeserializeShort overwrote point"
	//}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "trusted DeserializeShort had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "trusted DeserializeShort did not result in loss of P vs. P+A information"
	}

	// repeat with auto, untrusted.
	bufN.Reset()
	bufA.Reset()
	bufE.Reset()
	point_outN.SerializeShort(&bufN)
	point_outA.SerializeShort(&bufA)
	point_outE.SerializeShort(&bufE)

	_, errN = point_inN.DeserializeAuto(&bufN, UntrustedInput)
	_, errA = point_inA.DeserializeAuto(&bufA, UntrustedInput)
	_, errE = point_inE.DeserializeAuto(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "untrusted DeserializeAuto unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	if errE != ErrXNotInSubgroup {
		return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeAuto"
	}

	if !point_inE.IsEqual_FullCurve(&sentinel_point) {
		return false, "failed DeserializeAuto overwrote point"
	}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "untrusted DeserializeAuto had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "untrusted DeserializeAuto did not result in loss of P vs. P+A information"
	}

	// repeat with short, trusted.
	bufN.Reset()
	bufA.Reset()
	// bufE.Reset()
	point_outN.SerializeShort(&bufN)
	point_outA.SerializeShort(&bufA)
	// point_outE.SerializeShort(&bufE)

	_, errN = point_inN.DeserializeAuto(&bufN, TrustedInput)
	_, errA = point_inA.DeserializeAuto(&bufA, TrustedInput)
	// _, errE = point_inE.DeserializeShort(&bufE, UntrustedInput)

	if (errN != nil) || (errA != nil) {
		return false, "trusted DeserializeAuto unexpectedly failed" // covered by checkfun_serialization_roundtrip
	}
	// if errE != ErrNotInSubgroup {
	// 	return false, "Did not get Not-In-Subgroup error upon untrusted DeserializeShort"
	// }

	// if !point_inE.IsEqual_FullCurve(&example_generator_xtw) {
	//  	return false, "failed DeserializeShort overwrote point"
	//}
	// Note that we do NOT have IsEqual_FullCurve here.
	if !point_inA.IsEqual(point_outA) {
		return false, "trusted DeserializeAuto had Roundtrip error (modulo A)"
	}
	if !point_inA.IsEqual_FullCurve(point_inN) {
		return false, "trusted DeserializeAuto did not result in loss of P vs. P+A information"
	}

	/*
		var point_out CurvePointPtrInterfaceRead = s.Points[0].Clone()
		var point_in CurvePointPtrInterface = MakeCurvePointPtrInterfaceFromType(GetPointType(point_out))
		var buf bytes.Buffer
		var err error
		var bytes_read int

	*/
	return true, ""
}

func test_serialization_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	// var type1, type2 PointType
	make_samples1_and_run_tests(t, checkfun_NaP_serialization, "Unexpected behaviour when serialializing wrt NaPs or infinite points "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_serialization_type_consistency, "Unexpected behaviour when comparing serialization depencency on receiver type "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_serialization_roundtrip, "Roundtripping points did not work "+point_string, receiverType, 10, excludedFlags)
	make_samples1_and_run_tests(t, checkfun_rountrip_modulo2torsion, "Serialization roundtripping dependency on addition of 2-torsion is not as expected "+point_string, receiverType, 10, excludedFlags)
}

func TestSerializationForXTW(t *testing.T) {
	test_serialization_properties(t, pointTypeXTW, 0)
}

func TestSerializationForAXTW(t *testing.T) {
	test_serialization_properties(t, pointTypeAXTW, 0)
}

func TestSerializationForEFGH(t *testing.T) {
	test_serialization_properties(t, pointTypeEFGH, 0)
}
