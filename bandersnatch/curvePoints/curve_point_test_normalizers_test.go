package curvePoints

import (
	"reflect"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

var NormalizerForYType reflect.Type = utils.TypeOfType[NormalizerForY]()
var NormalizerForZType reflect.Type = utils.TypeOfType[NormalizerForZ]()

var _ NormalizerForZ = &Point_xtw_full{}
var _ NormalizerForZ = &Point_xtw_subgroup{}

var _ NormalizerForZ = &Point_axtw_full{}
var _ NormalizerForZ = &Point_axtw_subgroup{}

var _ NormalizerForY = &Point_xtw_full{}
var _ NormalizerForY = &Point_xtw_subgroup{}

func TestNormalizeForZ(t *testing.T) {
	for _, pointType := range allTestPointTypes {
		point_string := pointTypeToString(pointType)
		if !pointType.Implements(NormalizerForZType) {
			continue
		}
		make_samples1_and_run_tests(t, checkfun_normalizeForZ, "normalizeForZ failed for "+point_string, pointType, 50, excludeNoPoints)
	}
}

func TestNormalizeForY(t *testing.T) {
	for _, pointType := range allTestPointTypes {
		point_string := pointTypeToString(pointType)
		if !pointType.Implements(NormalizerForYType) {
			continue
		}
		make_samples1_and_run_tests(t, checkfun_normalizeForY, "normalizeForY failed for "+point_string, pointType, 50, excludeNoPoints)
	}
}

func checkfun_normalizeForZ(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	infinite := s.AnyFlags().CheckFlag(PointFlag_infinite)
	singular := s.AnyFlags().CheckFlag(PointFlagNAP)
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone()
	var ok bool = false

	do_normalize := func() {
		ok = clone1.(NormalizerForZ).NormalizeForZ()
	}
	// For NaPs, we expect ok == false and the nap handler be called
	if singular {
		napDetected := wasInvalidPointEncountered(do_normalize)
		if !napDetected {
			return false, "NormalizeForZ did not recognize NaP"
		}
		if ok {
			return false, "NormalizeForZ worked OK on NaP"
		}
		return true, ""
	}
	// For points at inifnity, we expect ok == false
	if infinite {
		do_normalize()
		if ok {
			return false, "NormalizeZ worked OK on point at infinity"
		}
		return true, ""
	}
	// non-singular, no-infinite point
	do_normalize()
	if !ok {
		return false, "NormalizeZ failed for good point"
	}
	if !clone1.IsEqual(clone2) {
		return false, "NormalizeZ changed point"
	}
	Zcoo := clone1.Z_decaf_projective()
	if !Zcoo.IsOne() {
		return false, "NormalizeZ did not result in Z-coo 1"
	}
	return true, ""

}

func checkfun_normalizeForY(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	infinite := s.AnyFlags().CheckFlag(PointFlag_infinite)
	singular := s.AnyFlags().CheckFlag(PointFlagNAP)
	clone1 := s.Points[0].Clone()
	clone2 := s.Points[0].Clone()
	var ok bool = false

	do_normalize := func() {
		ok = clone1.(NormalizerForY).NormalizeForY()
	}
	// For NaPs, we expect ok == false and the nap handler be called
	if singular {
		napDetected := wasInvalidPointEncountered(do_normalize)
		if !napDetected {
			return false, "NormalizeForY did not recognize NaP"
		}
		if ok {
			return false, "NormalizeForY worked OK on NaP"
		}
		return true, ""
	}
	// For points at inifnity, we expect ok == false
	if infinite {
		do_normalize()
		if ok {
			return false, "NormalizeY worked OK on point at infinity"
		}
		return true, ""
	}
	// non-singular, no-infinite point
	do_normalize()
	if !ok {
		return false, "NormalizeY failed for good point"
	}
	if !clone1.IsEqual(clone2) {
		return false, "NormalizeY changed point"
	}
	Ycoo := clone1.Y_decaf_projective()
	if !Ycoo.IsOne() {
		return false, "NormalizeY did not result in Y-coo 1"
	}
	return true, ""

}
