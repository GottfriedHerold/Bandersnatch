package curvePoints

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime/debug"
	"strconv"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

/*
	This file contains code for the testing framework used to test the elliptic curve operations (not the field operations).
	It does not contain any tests itself.

	Our simple testing framework works mostly as follows: A lot of tests call a
	checkfunction taking a TestSample and returning a (success bool, errormsg string) pair
	TestSample contains a k-tuple of points (We only support and need k=1, k=2, k=3) and the checkfunction is supposed to verify some property for this particular k-tuple.
	(e.g. a hypothetical checkfunction_addition_commutative could run on a 2-tuple P,Q and check whether P+Q == Q+P holds)
	In addition to the the k-tuple, TestSamples also contain some metadata for each point such as "This point is the neutral element".
	This is used to derive the actual expected behaviour in a test.
	(e.g. a checkfunction_IsNeutralElement could run on 1-tuples P and could check whether P.IsNeutralElement() matches the metadata)

	We note that the testing framework heavily makes use of interfaces and reflection to avoid writing separate tests for all Point Types.
	Also, note that checkfunction might need additional parameters. This is usually achieved using closures.
	(e.g. checkfunction_addition_commutative actually is a make_checkfun_addcommutes(receiverType) that returns a checkfunction
	with a given receiverType that determines which type P+Q and Q+P actually has (P, Q and P+Q do not need to have the same type))
*/

// checkfunction is the type of function that we run on test samples.
// Note that checkfunctions are supposed to be run on fresh (copies of) samples, so inadvertent modifications to the provided samples are not visible in other tests.
// (The only reason we have a pointer receiver is to use TestSample.Log)
// At any rate, the framework always (re)generates fresh samples.
type checkfunction func(*TestSample) (bool, string)

// PointFlags are used to mark TestSamples with meta-information about them. This is used to derive the expected behaviour
// we test against.
type PointFlags uint64

const (
	// TODO: Rename
	PointFlagNAP                 PointFlags = 1 << iota // Point is a NaP
	PointFlag_infinite                                  // Point is at infinity
	PointFlag_2torsion                                  // Point is 2-torsion
	PointFlag_outsideP253                               // Point is outside the prime-order subgroup. Note: Subgroup-points in Decaf-style representation must *NOT* have this flag.
	PointFlag_outsideGoodgroup                          // Point is outside the subgroup spanned by the prime-order subgroup and the affine two-torsion point.
	PointFlag_equalExact                                // For TestSamples with 2 points: Both points have this if they are equal. Points with this flag must have Case_equal_moduloA as well
	PointFlag_equalModuloA                              // For TestSamples with 2 points: Both points have this if they are equal modulo A.
	PointFlag_zeroModuloA                               // Point is either the neutral element or the affine 2-torsion point
	PointFlag_zeroExact                                 // Point is the neutral element
	PointFlag_random                                    // Points was sampled randomly
	PointFlag_differenceInfinite                        // For TestSamples with 2 points: The difference of the points is at infinity
	PointFlag_sumInfinite                               // For TestSamples with 2 points: The sum of the points is at infinity
	PointFlag_E1                                        // Point is the E1 point at infinity
	PointFlag_E2                                        // Point is the E2 point at infinity
	PointFlag_A                                         // Point is the affine 2-torsion point
)

// excludeNoPoints is used as an argument to functions taking a bitmask to exclude certain samples. Use this to exclude no samples.
const excludeNoPoints = PointFlags(0)

// CheckFlag returns true if any of the checked_flag is present in flags. checked_flag should be a bit-wise or of flags.
func (flags PointFlags) CheckFlag(checked_flags PointFlags) bool {
	return flags&checked_flags != 0
}

// maybeFlipDecaf will run flipDecaf if that is meaningful for the given point type; do nothing otherwise
func maybeFlipDecaf(p CurvePointPtrInterfaceTestSample) (ok bool) {
	var p_conv curvePointPtrInterfaceDecaf
	p_conv, ok = p.(curvePointPtrInterfaceDecaf)
	if ok {
		ok = p_conv.HasDecaf()
		if ok {
			p_conv.flipDecaf()
		}
	}
	return
}

var (
	_ CurvePointPtrInterfaceTestSample = &Point_efgh_subgroup{}
	_ CurvePointPtrInterfaceTestSample = &Point_efgh_full{}

	_ CurvePointPtrInterfaceTestSample = &Point_xtw_full{}
	_ CurvePointPtrInterfaceTestSample = &Point_xtw_subgroup{}

	_ CurvePointPtrInterfaceTestSample = &Point_axtw_subgroup{}
	_ CurvePointPtrInterfaceTestSample = &Point_axtw_full{}
)

var (
	_ sampleableNaP = &point_efgh_base{}
	_ sampleableNaP = &point_xtw_base{}
	_ sampleableNaP = &point_axtw_base{}
)

var (
	_ curvePointPtrInterfaceTestSampleA = &Point_xtw_full{}
	_ curvePointPtrInterfaceTestSampleA = &Point_axtw_full{}
	_ curvePointPtrInterfaceTestSampleA = &Point_efgh_full{}
)

var (
	_ curvePointPtrInterfaceTestSampleE = &Point_xtw_full{}
	_ curvePointPtrInterfaceTestSampleE = &Point_efgh_full{}
)

// PointType can hold the type of a curve Point.
// Note that this is based on reflect.Type and is intended to store the type of a pointer (i.e.
// something satisfying the CurvePointPtrInterface or even CurvePointPtrInterfaceTestSample)
//
// Note that using PointType for a type not satisfying this will likely lead to a panic.
type PointType reflect.Type

// Do we need this for the base types?

var (
	pointTypeXTWBase     = reflect.TypeOf((*point_xtw_base)(nil))
	pointTypeXTWFull     = reflect.TypeOf((*Point_xtw_full)(nil))
	pointTypeXTWSubgroup = reflect.TypeOf((*Point_xtw_subgroup)(nil))

	pointTypeAXTWBase     = reflect.TypeOf((*point_axtw_base)(nil))
	pointTypeAXTWFull     = reflect.TypeOf((*Point_axtw_full)(nil))
	pointTypeAXTWSubgroup = reflect.TypeOf((*Point_axtw_subgroup)(nil))

	pointTypeEFGHBase     = reflect.TypeOf((*point_efgh_base)(nil))
	pointTypeEFGHFull     = reflect.TypeOf((*Point_efgh_full)(nil))
	pointTypeEFGHSubgroup = reflect.TypeOf((*Point_efgh_subgroup)(nil))
)

// makeCurvePointPtrInterface creates a pointer to a new zero-initialized curve point of the given type.
// The return value is of type CurvePointPtrInterface and may need to be further type-asserted by the caller.
// Note that the return value likely is a NaP.
//
// It panics if pointType is invalid.
func makeCurvePointPtrInterface(pointType PointType) CurvePointPtrInterface {
	return reflect.New(pointType.Elem()).Interface().(CurvePointPtrInterface)
}

// makeCurvePointPtrInterfaceBase creates a pointer to a new zero-initialized base curve point of the given type.
// The return value is of type CurvePointPtrInterfaceBaseRead and may need to be further type-asserted by the caller.
// Note that the return value likely is a NaP.
//
// It panics if pointType is invalid.
func makeCurvePointPtrInterfaceBase(pointType PointType) CurvePointPtrInterfaceBaseRead {
	return reflect.New(pointType.Elem()).Interface().(CurvePointPtrInterfaceBaseRead)
}

// pointTypeToStringMap is just used to implement PointTypeToString as a look-up-table
var pointTypeToStringMap map[PointType]string = map[PointType]string{
	pointTypeXTWBase:      "xtw_base",
	pointTypeXTWFull:      "xtw_full",
	pointTypeXTWSubgroup:  "xtw_subgroup",
	pointTypeAXTWBase:     "axtw_base",
	pointTypeAXTWFull:     "axtw_full",
	pointTypeAXTWSubgroup: "axtw_subgroup",
	pointTypeEFGHBase:     "efgh_base",
	pointTypeEFGHFull:     "efgh_full",
	pointTypeEFGHSubgroup: "efgh_subgroup",
}

// pointTypeToString returns a string description of the given point type.
func pointTypeToString(c PointType) string {
	ret, ok := pointTypeToStringMap[c]
	if ok {
		return ret
	} else {
		return "unrecognized type [" + utils.GetReflectName(c) + "]"
	}
}

// pointTypeToTagMap is just used to implement PointTypeToTag as a look-up-table.
var pointTypeToTagMap map[PointType]string = map[PointType]string{
	pointTypeXTWBase:      "tb",
	pointTypeXTWFull:      "tf",
	pointTypeXTWSubgroup:  "ts",
	pointTypeAXTWBase:     "ab",
	pointTypeAXTWFull:     "af",
	pointTypeAXTWSubgroup: "as",
	pointTypeEFGHBase:     "sb",
	pointTypeEFGHFull:     "sf",
	pointTypeEFGHSubgroup: "ss",
}

// pointTypeToTag turns a pointType to a short tag string; this is useful e.g. in making benchmarking tables.
func pointTypeToTag(c PointType) string {
	ret, ok := pointTypeToTagMap[c]
	if ok {
		return ret
	} else {
		return "unrecognized tag [" + utils.GetReflectName(c) + "]"
	}
}

// typeCanRepresentInfinity is used to query whether a given point type can respresent and distinguish the two points at infinity.
func typeCanRepresentInfinity(pointType PointType) bool {
	return makeCurvePointPtrInterface(pointType).CanRepresentInfinity()
}

// typeCanOnlyRepresentSubgroup is used to query whether a given point type can only represent elements from the prime-order subgroup or arbitrary curve points.
func typeCanOnlyRepresentSubgroup(pointType PointType) bool {
	return makeCurvePointPtrInterface(pointType).CanOnlyRepresentSubgroup()
}

// getPointType returns the type (as a PointType) of a given concrete curve point.
func getPointType(p CurvePointPtrInterfaceTestSample) PointType {
	return reflect.TypeOf(p)
}

// TestSample is a struct that is used as input to most of our test functions, encapsulating a set of points
// together with metadata.
type TestSample struct {
	Points  []CurvePointPtrInterfaceTestSample // a slice of 1--3 points. The individual points can have different concrete type.
	Flags   []PointFlags                       // flags that give additional information about the points. This determines intended behaviour to test against.
	Comment string                             // A human-readable comment that describes the sample. Useful for diagnostics.
	Len     uint                               // Len == len(Points) == len(Flags). The given TestSample consists of this many points. Must be one of 1,2 or 3.
	info    []string                           // uninitialized by default. This can be used to record information via Log(...) that is output as diagnostic on errors.
}

// Log records a string representation (via fmt.Sprint) of the given args in the sample. These are output in s.String() and can provide useful information when there is an error.
func (s *TestSample) Log(args ...interface{}) {
	if s.info == nil {
		s.info = make([]string, 0)
	}
	var str string = fmt.Sprint(args...)
	s.info = append(s.info, str)
}

// AssertNumberOfPoints asserts that the given TestSample consists of the given number of points.
// This is usually used at the beginning of a testfunction to ensure we do not run tests for
// pairs of points on triples etc.
func (s *TestSample) AssertNumberOfPoints(expectedLen int) {
	if int(s.Len) != expectedLen {
		panic("Test samples with a different number of curve points per samples expected")
	}
}

// TODO: Signature inconsistent with CurvePointPtrInterface's Clone (returns value!)

// Clone returns an independent copy of the given TestSample. The contained points are copied and do not retain any pointer-links to the original.
// Note that this Clone() function returns a value, not a pointer.
func (s *TestSample) Clone() (ret TestSample) {
	ret.Len = s.Len
	ret.Comment = s.Comment
	ret.Flags = make([]PointFlags, ret.Len)
	ret.Points = make([]CurvePointPtrInterfaceTestSample, ret.Len)
	for i := 0; i < int(ret.Len); i++ {
		ret.Flags[i] = s.Flags[i]
		ret.Points[i] = s.Points[i].Clone().(CurvePointPtrInterfaceTestSample)
	}
	if s.info != nil {
		l := len(s.info)
		ret.info = make([]string, l)
		copy(ret.info, s.info)
	}
	return
}

// AnyFlags returns the OR of all the flags of the TestSample.
// This is usually used as s.AnyFlags().CheckFlag(some_flag)
func (s TestSample) AnyFlags() (ret PointFlags) {
	for _, v := range s.Flags {
		ret |= v
	}
	return
}

// TODO: Automatically add flags based on type of p?

// MakeSample1 turns point p into a 1-point sample with given flags and comment.
// This takes ownership of p in the sense that the pointer p given as argument must no longer be used by the caller.
func MakeSample1(p CurvePointPtrInterfaceTestSample, flags PointFlags, comment string) (ret TestSample) {
	ret.Points = []CurvePointPtrInterfaceTestSample{p}
	ret.Flags = []PointFlags{flags}
	ret.Len = 1
	ret.Comment = comment
	ret.info = nil
	return
}

// ZipSample takes 2 samples a, b (consisting of n_a, n_b points) and combines them into
// a sample with n_a + n_b points. extra_flags get OR-ed to each.
func ZipSample(a, b TestSample, extra_flags PointFlags) (ret TestSample) {
	ret.Flags = append([]PointFlags{}, a.Flags...)
	ret.Flags = append(ret.Flags, b.Flags...)
	ret.Points = make([]CurvePointPtrInterfaceTestSample, 0, a.Len+b.Len)
	for _, point := range a.Points {
		ret.Points = append(ret.Points, point.Clone().(CurvePointPtrInterfaceTestSample))
	}
	for _, point := range b.Points {
		ret.Points = append(ret.Points, point.Clone().(CurvePointPtrInterfaceTestSample))
	}
	ret.Comment = a.Comment + ", " + b.Comment
	ret.Len = a.Len + b.Len
	for i := range ret.Flags {
		ret.Flags[i] |= extra_flags
	}
	if a.info != nil || b.info != nil {
		ret.info = make([]string, 0)
		if a.info != nil {
			ret.info = append(ret.info, a.info...)
		}
		if b.info != nil {
			ret.info = append(ret.info, b.info...)
		}
	}
	return
}

// precomputedSampleSlice holds several TestSamples that are reused (after being copied) across multiple tests.
// This is because creating these TestSamples is too slow otherwise.
type precomputedSampleSlice struct {
	// Samples come in three flavours: fixed samples are always present and contain
	// "special" samples such as the neutral element that are likely candidates to exhibit failures of our algorithms.
	// random samples are randomly generated/appended. Their number can increase upon request
	// NaPSamples contain (random) invalid points. Their number can increase upon request.
	fixedSamples  []TestSample
	randomSamples []TestSample
	NaPSamples    []TestSample
	rnd           *rand.Rand  // we keep the random seed to create/append new samples and create the samples in a deterministic order. This way, everything is reproducible
	sampleLen     int         // Each sample in fixedSamples, randomSamples, NaPSamples needs to have the same number of points given by sampleLen
	initialized   bool        // bool to denote whether this was already initialized (done on first request)
	pointTypes    []PointType // all samples have these point Types. len(pointTypes) == sampleLen
}

// pointTypePair denotes a pair of PointType's. This type is needed because it's used a key of a map.
type pointTypePair struct {
	a, b PointType
}

// pointTypeTriple denotes a triple of PointType's. This type is just used a key of a map.
type pointTypeTriple struct {
	a, b, c PointType
}

// precomputedSamples<N> is a map of pointType(s) -> (pointer to)precomputedSampleSlice.
// The values are generated upon first access.
var (
	precomputedSamples1 map[PointType]*precomputedSampleSlice       = make(map[PointType]*precomputedSampleSlice)
	precomputedSamples2 map[pointTypePair]*precomputedSampleSlice   = make(map[pointTypePair]*precomputedSampleSlice)
	precomputedSamples3 map[pointTypeTriple]*precomputedSampleSlice = make(map[pointTypeTriple]*precomputedSampleSlice)
)

const (
	// initialRandom1 = 0
	// initialNaP1    = 0
	initialRandom2 = 12
	initialNaP2    = 12
)

// initialize is called to create a valid precomputedSampleSlice with actual TestSamples.
// for the given point types and (determinstic) random source.
func (s *precomputedSampleSlice) initialize(rnd *rand.Rand, types []PointType) {
	s.initialized = true
	len := len(types)
	if len == 0 {
		panic("Trying to create precomputedSampleSlice with 0 points per entry")
	}
	s.sampleLen = len
	s.pointTypes = make([]PointType, len)
	for i := 0; i < len; i++ {
		s.pointTypes[i] = types[i]
	}
	s.rnd = rnd
	s.NaPSamples = make([]TestSample, 0)
	s.randomSamples = make([]TestSample, 0)
	s.fixedSamples = make([]TestSample, 0)
	switch len {
	case 1:
		s.prepareFixedSamples1()
	case 2:
		s.prepareFixedSamples2()
	case 3:
		s.prepareFixedSamples3()
	}
}

// prepareFixedSamples1 is called at the end of initialize() for sampleLen==1. Its job is to create TestSamples with 1 curve Point.
func (s *precomputedSampleSlice) prepareFixedSamples1() {
	if s.sampleLen != 1 {
		panic("Cannot happen")
	}
	var rnd *rand.Rand = s.rnd
	pointType1 := s.pointTypes[0]
	var newSample1, newSample2, newSample3 TestSample
	var ok bool
	for _, f := range []func(PointType) (TestSample, bool){makeSample_N, makeSample_A, makeSample_E1, makeSample_E2, makeSample_Gen} {
		newSample1, ok = f(pointType1)
		if ok {
			s.fixedSamples = append(s.fixedSamples, newSample1.Clone())
			newSample2 = newSample1.Clone()
			newSample2.Points[0].rerandomizeRepresentation(rnd)
			newSample2.Comment += " Rerandomized"
			s.fixedSamples = append(s.fixedSamples, newSample2.Clone())
			newSample3 = newSample1.Clone()
			if maybeFlipDecaf(newSample3.Points[0]) {
				newSample3.Comment += " decaf flipped"
				s.fixedSamples = append(s.fixedSamples, newSample3)
				newSample3 = newSample2.Clone()
				maybeFlipDecaf(newSample3.Points[0])
				newSample3.Comment += " decaf flipped"
				s.fixedSamples = append(s.fixedSamples, newSample3)
			}
		}
	}
	newSample1, _ = makeSample_Uninit(pointType1)
	s.fixedSamples = append(s.fixedSamples, newSample1.Clone())
}

// prepareFixedSamples2 is called at the end of initialize() for sampleLen==2. Its job is to create TestSamples with pairs of curve points.
func (s *precomputedSampleSlice) prepareFixedSamples2() {
	testutils.Assert(s.sampleLen == 2)
	var rnd *rand.Rand = s.rnd
	var sampleType1 PointType = s.pointTypes[0]
	var sampleType2 PointType = s.pointTypes[1]
	samples1 := getSamples(0, excludeNoPoints, sampleType1)
	samples2 := getSamples(0, excludeNoPoints, sampleType2)
	for _, sample1 := range samples1 {
		flags1 := sample1.Flags[0]
		for _, sample2 := range samples2 {
			flags2 := sample2.Flags[0]
			var newFlags PointFlags
			newSample := ZipSample(sample1, sample2, PointFlags(0))
			newSample = newSample.Clone() // avoid any pointers pointing to the same things.
			if flags1.CheckFlag(PointFlag_zeroExact) && flags2.CheckFlag(PointFlag_zeroExact) {
				newFlags |= PointFlag_equalExact
			}
			if flags1.CheckFlag(PointFlag_A) && flags2.CheckFlag(PointFlag_A) {
				newFlags |= PointFlag_equalExact
			}
			if flags1.CheckFlag(PointFlag_E1) && flags2.CheckFlag(PointFlag_E1) {
				newFlags |= PointFlag_equalExact
			}
			if flags1.CheckFlag(PointFlag_E2) && flags2.CheckFlag(PointFlag_E2) {
				newFlags |= PointFlag_equalExact
			}
			// This is only true due to the specific samples that are in samples1/2:
			// We have the example generator in both cases.
			if !flags1.CheckFlag(PointFlag_2torsion|PointFlagNAP) && !flags2.CheckFlag(PointFlag_2torsion|PointFlagNAP) {
				newFlags |= PointFlag_equalExact
				newFlags |= PointFlag_equalModuloA
			}
			if flags1.CheckFlag(PointFlag_2torsion) && flags2.CheckFlag(PointFlag_2torsion) {
				if flags1.CheckFlag(PointFlag_infinite) == flags2.CheckFlag(PointFlag_infinite) {
					newFlags |= PointFlag_equalModuloA
				} else {
					newFlags |= PointFlag_differenceInfinite
					newFlags |= PointFlag_sumInfinite // sum and difference are the same for 2-torsion
				}
			}
			if flags1.CheckFlag(PointFlag_random) || flags2.CheckFlag(PointFlag_random) {
				panic("Unexpected random point in fixedSamples of length 1")
			}
			// they don't really apply individually; we give them to both.
			newSample.Flags[0] |= newFlags
			newSample.Flags[1] |= newFlags
			s.fixedSamples = append(s.fixedSamples, newSample)
		}
	}
	sample1, ok := makeSample_Gen(sampleType1)
	if ok {
		sample2, ok := makeSample_Gen(sampleType2)
		if ok {
			p, ok := sample2.Points[0].(CurvePointPtrInterfaceWrite)
			if ok {
				p.NegEq()
				sample2.Comment += " negated"
				newSample := ZipSample(sample1, sample2, PointFlags(0))
				s.fixedSamples = append(s.fixedSamples, newSample)
			}
		}
	}
	for i := 0; i < initialRandom2; i++ {
		newSampleRandom1, ok := makeSample_random(sampleType1, rnd)
		if ok {
			for _, sample2 := range samples2 {
				newSample := ZipSample(newSampleRandom1, sample2, PointFlags(0))
				s.randomSamples = append(s.randomSamples, newSample)
			}
		}
		newSampleRandom2, ok := makeSample_random(sampleType2, rnd)
		if ok {
			for _, sample1 := range samples1 {
				newSample := ZipSample(sample1, newSampleRandom2, PointFlags(0))
				s.randomSamples = append(s.randomSamples, newSample)
			}
		}
	}

	for i := 0; i < initialNaP2; i++ {
		newSampleNaP1, ok := makeSample_NaP(sampleType1, rnd, i)
		if ok {
			for _, sample2 := range samples2 {
				newSample := ZipSample(newSampleNaP1, sample2, PointFlags(0))
				s.NaPSamples = append(s.NaPSamples, newSample)
			}
		}
		newSampleNaP2, ok := makeSample_NaP(sampleType2, rnd, i)
		if ok {
			for _, sample1 := range samples1 {
				newSample := ZipSample(sample1, newSampleNaP2, PointFlags(0))
				s.NaPSamples = append(s.NaPSamples, newSample)
			}
		}
	}
}

func (s *precomputedSampleSlice) prepareFixedSamples3() {
	testutils.Assert(s.sampleLen == 3)
	// var rnd *rand.Rand = s.rnd
	var sampleType1 PointType = s.pointTypes[0]
	var sampleType2 PointType = s.pointTypes[1]
	var sampleType3 PointType = s.pointTypes[2]
	samples12 := getSamples(5, excludeNoPoints, sampleType1, sampleType2)
	samples3 := getSamples(5, excludeNoPoints, sampleType3)
	for _, sample12 := range samples12 {
		for _, sample3 := range samples3 {
			newSample := ZipSample(sample12, sample3, PointFlags(0))
			newSample = newSample.Clone() // Be sure to avoid any pointers pointing to the same things. Probably not needed as ZipSample should do it.
			s.fixedSamples = append(s.fixedSamples, newSample)
		}
	}
}

func (s *precomputedSampleSlice) elongate(newSize int) {
	if len(s.randomSamples) >= newSize && len(s.NaPSamples) >= newSize {
		return
	}
	var toAddRandom int = newSize - len(s.randomSamples)
	var toAddNaP int = newSize - len(s.NaPSamples)
	var toAdd int
	if toAddRandom > toAddNaP {
		toAdd = toAddRandom
	} else {
		toAdd = toAddNaP
	}
	if toAdd < 0 {
		panic("Cannot happen")
	}
	var i int
	switch s.sampleLen {
	case 1:
		for i = 0; i < toAdd; i++ {
			s.elongate1()
		}
	case 2:
		for i = 0; i < toAdd; i++ {
			s.elongate2()
		}
	case 3:
		for i = 0; i < toAdd; i++ {
			s.elongate3()
		}
	default:
		panic("Cannot happen")
	}
}

func (s *precomputedSampleSlice) elongate1() {
	testutils.Assert(s.sampleLen == 1)
	testutils.Assert(len(s.pointTypes) == 1)
	pointType1 := s.pointTypes[0]
	var rnd *rand.Rand = s.rnd
	randomSample, _ := makeSample_random(pointType1, rnd)
	s.randomSamples = append(s.randomSamples, randomSample)
	NaPSample, ok := makeSample_NaP(pointType1, rnd, len(s.NaPSamples))
	if ok {
		s.NaPSamples = append(s.NaPSamples, NaPSample)
	}
}

func (s *precomputedSampleSlice) elongate2() {
	testutils.Assert(s.sampleLen == 2)
	testutils.Assert(len(s.pointTypes) == 2)
	pointType1 := s.pointTypes[0]
	pointType2 := s.pointTypes[1]
	var rnd *rand.Rand = s.rnd
	var ok bool
	randomSample1, ok := makeSample_random(pointType1, rnd)
	if !ok {
		panic("Could not create random sample")
	}
	randomSample2, ok := makeSample_random(pointType2, rnd)
	if !ok {
		panic("Could not create random sample")
	}
	newSample := ZipSample(randomSample1, randomSample2, PointFlags(0))
	s.randomSamples = append(s.randomSamples, newSample)
	NaPSample1, ok := makeSample_NaP(pointType1, rnd, len(s.NaPSamples))
	if !ok {
		panic("Could not create NaP sample")
	}
	NaPSample2, ok := makeSample_NaP(pointType2, rnd, rnd.Intn(256))
	if !ok {
		panic("Could not create NaP sample")
	}
	newSample = ZipSample(NaPSample1, NaPSample2, PointFlags(0))
	s.NaPSamples = append(s.NaPSamples, newSample)
}

func (s *precomputedSampleSlice) elongate3() {
	testutils.Assert(s.sampleLen == 3)
	testutils.Assert(len(s.pointTypes) == 3)
	pointType1 := s.pointTypes[0]
	pointType2 := s.pointTypes[1]
	pointType3 := s.pointTypes[2]
	var rnd *rand.Rand = s.rnd
	var ok bool
	randomSample1, ok := makeSample_random(pointType1, rnd)
	if !ok {
		panic("Could not create random sample")
	}
	randomSample2, ok := makeSample_random(pointType2, rnd)
	if !ok {
		panic("Could not create random sample")
	}
	randomSample3, ok := makeSample_random(pointType3, rnd)
	if !ok {
		panic("Could not create random sample")
	}
	newSample12 := ZipSample(randomSample1, randomSample2, PointFlags(0))
	newSample := ZipSample(newSample12, randomSample3, PointFlags(0))
	s.randomSamples = append(s.randomSamples, newSample)
	NaPSample1, ok := makeSample_NaP(pointType1, rnd, len(s.NaPSamples))
	if !ok {
		panic("Could not create NaP sample")
	}
	NaPSample2, ok := makeSample_NaP(pointType2, rnd, rnd.Intn(256))
	if !ok {
		panic("Could not create NaP sample")
	}
	NaPSample3, ok := makeSample_NaP(pointType3, rnd, rnd.Intn(256))
	if !ok {
		panic("Could not create NaP sample")
	}
	newSample12 = ZipSample(NaPSample1, NaPSample2, PointFlags(0))
	newSample = ZipSample(newSample12, NaPSample3, PointFlags(0))
	s.NaPSamples = append(s.NaPSamples, newSample)
}

// makePointTypeTuple turns a single/pair/triple of pointTypes into a variable of type PointType / PointTypePair / PointTypeTriple.
func makePointTypeTuple(pointTypes ...PointType) interface{} {
	length := len(pointTypes)
	switch length {
	case 1:
		return pointTypes[0]
	case 2:
		return pointTypePair{a: pointTypes[0], b: pointTypes[1]}
	case 3:
		return pointTypeTriple{a: pointTypes[0], b: pointTypes[1], c: pointTypes[2]}
	default:
		panic("makePointTypeTuple only supports 1--3 arguments")
	}
}

func appendSamplesAsCopy(destination *[]TestSample, source *[]TestSample, excludeFlags PointFlags, num int) {
	sourceLen := len(*source)
	if num > sourceLen {
		panic("Trying to copy more than there is")
	}
	var newSample TestSample
	for i := 0; i < num; i++ {
		newSample = (*source)[i].Clone()
		if !newSample.AnyFlags().CheckFlag(excludeFlags) {
			*destination = append(*destination, newSample)
		}
	}
}

func getSamples(random_size int, excludeFlags PointFlags, pointTypes ...PointType) (ret []TestSample) {
	var index interface{} = makePointTypeTuple(pointTypes...)
	numTypes := len(pointTypes)
	var precomp *precomputedSampleSlice
	var alreadyExists bool
	switch numTypes {
	case 1:
		precomp, alreadyExists = precomputedSamples1[index.(PointType)]
	case 2:
		precomp, alreadyExists = precomputedSamples2[index.(pointTypePair)]
	case 3:
		precomp, alreadyExists = precomputedSamples3[index.(pointTypeTriple)]
	default:
		panic("Cannot happen")
	}
	if !alreadyExists {
		var rnd *rand.Rand = rand.New(rand.NewSource(800))
		precomp = new(precomputedSampleSlice)
		precomp.initialize(rnd, pointTypes)
		switch numTypes {
		case 1:
			precomputedSamples1[index.(PointType)] = precomp
		case 2:
			precomputedSamples2[index.(pointTypePair)] = precomp
		case 3:
			precomputedSamples3[index.(pointTypeTriple)] = precomp
		default:
			panic("Cannot happen")
		}
	}
	if random_size >= 0 && (random_size > len(precomp.randomSamples) || random_size > len(precomp.NaPSamples)) {
		precomp.elongate(random_size)
	}
	if random_size >= 0 && (random_size > len(precomp.randomSamples) || random_size > len(precomp.NaPSamples)) {
		panic("Cannot happen")
	}
	var outputLength int = len(precomp.fixedSamples)
	var outputRandom, outputNaP int
	if random_size == -1 {
		outputRandom = len(precomp.randomSamples)
		outputNaP = len(precomp.NaPSamples)
	} else {
		outputRandom = random_size
		outputNaP = random_size
	}
	outputLength += outputRandom
	outputLength += outputNaP
	ret = make([]TestSample, 0, outputLength)
	appendSamplesAsCopy(&ret, &precomp.fixedSamples, excludeFlags, len(precomp.fixedSamples))
	appendSamplesAsCopy(&ret, &precomp.randomSamples, excludeFlags, outputRandom)
	appendSamplesAsCopy(&ret, &precomp.NaPSamples, excludeFlags, outputNaP)
	return
}

func makeSample_prep(pointTypes ...PointType) (ret TestSample) {
	ret.Points = make([]CurvePointPtrInterfaceTestSample, len(pointTypes))
	for i, pointType := range pointTypes {
		p := makeCurvePointPtrInterface(pointType).(CurvePointPtrInterfaceTestSample)
		ret.Points[i] = p
	}
	ret.Flags = make([]PointFlags, len(pointTypes))
	ret.Len = uint(len(pointTypes))
	ret.Comment = "uninitialized"
	return
}

func makeSample_N(pointType PointType) (ret TestSample, ok bool) {
	ok = true
	ret = makeSample_prep(pointType)
	ret.Points[0].SetNeutral()
	ret.Flags[0] = PointFlag_zeroExact | PointFlag_2torsion | PointFlag_zeroModuloA
	ret.Comment = "Neutral Elements"
	return
}

func makeSample_A(pointType PointType) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	p_conv, ok := ret.Points[0].(curvePointPtrInterfaceTestSampleA)
	if !ok {
		return
	}
	// p_conv is a pointer, so will affect ret.Points[0]
	p_conv.SetAffineTwoTorsion()
	ret.Flags[0] = PointFlag_zeroModuloA | PointFlag_2torsion | PointFlag_outsideP253 | PointFlag_A
	ret.Comment = "Affine two-torsion"
	return
}

func makeSample_E1(pointType PointType) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	p_conv, ok := ret.Points[0].(curvePointPtrInterfaceTestSampleE)
	if !ok {
		return
	}
	if !ret.Points[0].CanRepresentInfinity() {
		panic("Point type has SetE1() defined, but CanRepresentInfinity is false")
	}
	p_conv.SetE1()
	ret.Flags[0] = PointFlag_2torsion | PointFlag_infinite | PointFlag_outsideGoodgroup | PointFlag_outsideP253 | PointFlag_E1
	ret.Comment = "infinite point E1"
	return
}

func makeSample_E2(pointType PointType) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	p_conv, ok := ret.Points[0].(curvePointPtrInterfaceTestSampleE)
	if !ok {
		return
	}
	if !ret.Points[0].CanRepresentInfinity() {
		panic("Point type has SetE2() defined, but CanRepresentInfinity is false")
	}
	p_conv.SetE2()
	ret.Flags[0] = PointFlag_2torsion | PointFlag_infinite | PointFlag_outsideGoodgroup | PointFlag_outsideP253 | PointFlag_E2
	ret.Comment = "infinite point E2"
	return
}

func makeSample_Gen(pointType PointType) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	ok = true
	var p Point_xtw_subgroup
	p.point_xtw_base = example_generator_xtw
	ret.Points[0].SetFrom(&p)
	ret.Comment = "example generator"
	return
}

func makeSample_Uninit(pointType PointType) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	ok = true
	ret.Flags[0] = PointFlagNAP
	ret.Comment = "Uninitialized point"
	return
}

func makeSample_random(pointType PointType, rnd *rand.Rand) (ret TestSample, ok bool) {
	ok = true
	ret = makeSample_prep(pointType)
	ret.Points[0].sampleRandomUnsafe(rnd)
	ret.Flags[0] = PointFlag_random
	if !ret.Points[0].CanOnlyRepresentSubgroup() {
		ret.Flags[0] |= PointFlag_outsideGoodgroup | PointFlag_outsideP253
	}
	ret.Comment = "Random sample"
	return
}

func makeSample_NaP(pointType PointType, rnd *rand.Rand, index int) (ret TestSample, ok bool) {
	ret = makeSample_prep(pointType)
	p_conv, ok := ret.Points[0].(sampleableNaP)
	if !ok {
		return
	}
	p_conv.sampleNaP(rnd, index)
	ret.Flags[0] = PointFlag_random | PointFlagNAP
	ret.Comment = "random NaP"
	return
}

func (s *TestSample) String() string {
	var ret string
	if s.Len == 0 {
		return "Empty test sample consisting of 0-tuple of points"
	}
	for i := 0; i < int(s.Len); i++ {
		ret += "Point "
		ret += strconv.Itoa(i + 1)
		ret += " of type "
		ret += pointTypeToString(getPointType(s.Points[i]))
		ret += ", "
	}
	ret += "Comment stored in sample: "
	ret += s.Comment
	ret += "\n"
	for i := 0; i < int(s.Len); i++ {
		ret += "Representation of Point " + strconv.Itoa(i+1) + " (" + pointTypeToString(getPointType(s.Points[i])) + ") is "
		ret += s.Points[i].String()
		if i+1 < int(s.Len) {
			ret += "\n"
		}
	}
	if s.info != nil {
		ret += "\nAdditional info:\n"
		for i, str := range s.info {
			ret += str
			if i+1 < len(s.info) {
				ret += "\n"
			}
		}
	}
	return ret
}

func run_tests_on_samples(f checkfunction, t *testing.T, samples []TestSample, err_string string) {
	var numErrors int = 0
	var failed bool = false
	var samp TestSample

	// This function is used to detect whether an error has occured.
	// We do NOT call recover(), because Go's internal testing framework actually recovers and we do not want interfere with it.
	panicked := true // set to false before return
	defer func() {
		// panicked can only be true if execution of the for-loop below was terminated prematurely be a panic.
		if panicked {
			t.Errorf("Panic detected. Context info: %v\n", err_string)
			t.Errorf("Failed Sample: %v", samp.String())
		}
	}()
	for _, samp = range samples {
		pass, error_reason := f(&samp)
		if failed && !pass {
			numErrors++
		}
		if !failed && !pass {
			failed = true
			t.Errorf("%v\nAdditional info: %v\nFailed Sample: %v\nPrinting Stack trace", err_string, error_reason, samp.String())
			debug.PrintStack()
		}
	}
	panicked = false
	if failed {
		t.Fatal(" and " + strconv.Itoa(numErrors) + " further errors")
	}
}

func make_samples1_and_run_tests(t *testing.T, f checkfunction, err_string string, point_type1 PointType, random_size int, excluded_flags PointFlags) {
	samples := getSamples(random_size, excluded_flags, point_type1)
	// Samples := MakeTestSamples1(random_size, point_type1, excluded_flags)
	run_tests_on_samples(f, t, samples, err_string)
}

func make_samples2_and_run_tests(t *testing.T, f checkfunction, err_string string, point_type1 PointType, point_type2 PointType, random_size int, excluded_flags PointFlags) {
	samples := getSamples(random_size, excluded_flags, point_type1, point_type2)
	run_tests_on_samples(f, t, samples, err_string)
}

func make_samples3_and_run_tests(t *testing.T, f checkfunction, err_string string, point_type1 PointType, point_type2 PointType, point_type3 PointType, random_size int, excluded_flags PointFlags) {
	samples := getSamples(random_size, excluded_flags, point_type1, point_type2, point_type3)
	run_tests_on_samples(f, t, samples, err_string)
}
