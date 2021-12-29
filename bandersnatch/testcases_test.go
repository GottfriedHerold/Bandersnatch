package bandersnatch

import (
	"math/rand"
	"reflect"
	"runtime/debug"
	"strconv"
	"testing"
)

type PointFlags uint64

const (
	Case_singular PointFlags = 1 << iota
	Case_infinite
	Case_2torsion
	Case_outside_p253
	Case_outside_goodgroup
	Case_equal_exact
	Case_equal
	Case_zero
	Case_zero_exact
	Case_random
	Case_differenceInfinite
)

func (flags PointFlags) CheckFlag(check PointFlags) bool {
	return flags&check != 0
}

type PointType reflect.Type

var (
	pointTypeXTW  = reflect.TypeOf((*Point_xtw)(nil))
	pointTypeAXTW = reflect.TypeOf((*Point_axtw)(nil))
	pointTypeEFGH = reflect.TypeOf((*Point_efgh)(nil))
)

func PointTypeToString(c PointType) string {
	switch c {
	case pointTypeXTW:
		return "xtw"
	case pointTypeAXTW:
		return "axtw"
	case pointTypeEFGH:
		return "efgh"
	default:
		return "unknown type"
	}
}

func canRepresentInfinity(pointType PointType) bool {
	switch pointType {
	case pointTypeXTW:
		return true
	case pointTypeAXTW:
		return false
	case pointTypeEFGH:
		return true
	default:
		panic("Unknown type")
	}
}

func GetPointType(p CurvePointPtrInterfaceRead) PointType {
	// Could do a return PointType(reflect.TypeOf(p)), but we need to check that it is from a given list anyway.
	switch p.(type) {
	case *Point_xtw:
		return pointTypeXTW
	case *Point_axtw:
		return pointTypeAXTW
	case *Point_efgh:
		return pointTypeEFGH
	default:
		panic("Unrecognized Point type")
	}
}

func MakeCurvePointPtrInterfaceFromType(pointType PointType) CurvePointPtrInterface_FullCurve {
	return reflect.New(pointType.Elem()).Interface().(CurvePointPtrInterface_FullCurve)
}

type TestSample struct {
	Points  []CurvePointPtrInterfaceRead_FullCurve // TODO: CurvePointPtrInterfaceDebug interface?
	Flags   []PointFlags
	Comment string
	Len     uint
}

func (s *TestSample) AssertNumberOfPoints(expectedLen int) {
	if int(s.Len) != expectedLen {
		panic("Test samples with a different number of curve points per samples expected")
	}
}

func (s *TestSample) Clone() (ret TestSample) {
	ret.Len = s.Len
	ret.Comment = s.Comment
	ret.Flags = make([]PointFlags, ret.Len)
	ret.Points = make([]CurvePointPtrInterfaceRead_FullCurve, ret.Len)
	for i := 0; i < int(ret.Len); i++ {
		ret.Flags[i] = s.Flags[i]
		ret.Points[i] = s.Points[i].Clone().(CurvePointPtrInterfaceRead_FullCurve)
	}
	return
}

func (s TestSample) AnyFlags() (ret PointFlags) {
	for _, v := range s.Flags {
		ret |= v
	}
	return
}

func MakeSample1(p CurvePointPtrInterfaceRead_FullCurve, flags PointFlags, comment string) (ret TestSample) {
	ret.Points = []CurvePointPtrInterfaceRead_FullCurve{p}
	ret.Flags = []PointFlags{flags}
	ret.Len = 1
	ret.Comment = comment
	return

}

func ZipSample(a, b TestSample, extra_flags PointFlags) (ret TestSample) {
	ret.Flags = append([]PointFlags{}, a.Flags...)
	ret.Flags = append(ret.Flags, b.Flags...)
	ret.Points = make([]CurvePointPtrInterfaceRead_FullCurve, 0, a.Len+b.Len)
	for _, point := range a.Points {
		ret.Points = append(ret.Points, point.Clone().(CurvePointPtrInterfaceRead_FullCurve))
	}
	for _, point := range b.Points {
		ret.Points = append(ret.Points, point.Clone().(CurvePointPtrInterfaceRead_FullCurve))
	}
	ret.Comment = a.Comment + ", " + b.Comment
	ret.Len = a.Len + b.Len
	for i := range ret.Flags {
		ret.Flags[i] |= extra_flags
	}
	return
}

func AppendTestSamples(sample_list *[]TestSample, exclude_mask PointFlags, point_types []PointType, added_samples ...TestSample) {
	if len(added_samples) == 0 {
		return
	}
	var individual_sizes uint = 0
	if len(*sample_list) > 0 {
		individual_sizes = (*sample_list)[0].Len
	} else {
		individual_sizes = added_samples[0].Len
	}
	for _, item := range added_samples {
		if item.Len != individual_sizes {
			panic("Creating test samples failed. Samples mix up number of points per sample")
		}
		if item.AnyFlags()&exclude_mask != 0 {
			continue
		}
		good := true
		for i := 0; i < int(individual_sizes); i++ {
			if (!canRepresentInfinity(point_types[i])) && item.Flags[i].CheckFlag(Case_infinite) {
				good = false
			}
		}
		if !good {
			continue
		}
		*sample_list = append(*sample_list, item.CopyXTWToType(point_types))
	}
}

func (in *TestSample) CopyXTWToType(new_type []PointType) (ret TestSample) {
	ret.Comment = in.Comment
	ret.Len = in.Len
	if len(new_type) != int(in.Len) {
		panic("Invalid argument to CopyXTWToType: length mismatch for new_type")
	}
	for i := 0; i < int(in.Len); i++ {
		if GetPointType(in.Points[i]) != pointTypeXTW {
			if GetPointType(in.Points[i]) != new_type[i] {
				panic("Cannot convert sample")
			}
			ret.Points = append(ret.Points, in.Points[i].Clone().(CurvePointPtrInterfaceRead_FullCurve))
			ret.Flags = append(ret.Flags, in.Flags[i])
			continue
		}
		switch new_type[i] {
		case pointTypeXTW:
			ret.Points = append(ret.Points, in.Points[i].Clone().(CurvePointPtrInterfaceRead_FullCurve))
			ret.Flags = append(ret.Flags, in.Flags[i])
		case pointTypeAXTW:
			if in.Flags[i]&Case_infinite != 0 || in.Flags[i]&Case_singular != 0 {
				panic("Cannot transform infinite or singular test point into axtw coordinates")
			}
			var point_copy Point_axtw = in.Points[i].AffineExtended()
			ret.Points = append(ret.Points, &point_copy)
			ret.Flags = append(ret.Flags, in.Flags[i])
		case pointTypeEFGH:
			var point_copy Point_efgh
			point_copy.SetFrom(in.Points[i])
			ret.Points = append(ret.Points, &point_copy)
			ret.Flags = append(ret.Flags, in.Flags[i])
		default:
			panic("Not supported yet")
		}
	}
	return
}

var test_sample_N = MakeSample1(
	&NeutralElement_xtw,
	Case_zero_exact|Case_2torsion|Case_zero,
	"Neutral Element")

var test_sample_E1 = MakeSample1(
	&exceptionalPoint_1_xtw,
	Case_infinite|Case_2torsion|Case_outside_goodgroup|Case_outside_p253,
	"Infinte 2-torsion point 1")

var test_sample_E2 = MakeSample1(
	&exceptionalPoint_2_xtw,
	Case_infinite|Case_2torsion|Case_outside_goodgroup|Case_outside_p253,
	"Infinte 2-torsion point 2")

var test_sample_A = MakeSample1(
	&orderTwoPoint_xtw,
	Case_2torsion|Case_outside_p253|Case_zero,
	"Affine 2-torsion point")

var test_sample_NN = ZipSample(test_sample_N, test_sample_N, Case_equal|Case_equal_exact)
var test_sample_NA = ZipSample(test_sample_N, test_sample_A, Case_equal)
var test_sample_NE1 = ZipSample(test_sample_N, test_sample_E1, Case_differenceInfinite)
var test_sample_NE2 = ZipSample(test_sample_N, test_sample_E2, Case_differenceInfinite)
var test_sample_NG = ZipSample(test_sample_N, test_sample_gen, 0)

var test_sample_AN = ZipSample(test_sample_A, test_sample_N, Case_equal)
var test_sample_AA = ZipSample(test_sample_A, test_sample_A, Case_equal|Case_equal_exact)
var test_sample_AE1 = ZipSample(test_sample_A, test_sample_E1, Case_differenceInfinite)
var test_sample_AE2 = ZipSample(test_sample_A, test_sample_E1, Case_differenceInfinite)
var test_sample_AG = ZipSample(test_sample_A, test_sample_gen, 0)

var test_sample_E1N = ZipSample(test_sample_E1, test_sample_N, Case_differenceInfinite)
var test_sample_E1A = ZipSample(test_sample_E1, test_sample_A, Case_differenceInfinite)
var test_sample_E1E1 = ZipSample(test_sample_E1, test_sample_E1, Case_equal|Case_equal_exact)
var test_sample_E1E2 = ZipSample(test_sample_E1, test_sample_E2, Case_equal)
var test_sample_E1G = ZipSample(test_sample_E1, test_sample_gen, 0)

var test_sample_E2N = ZipSample(test_sample_E2, test_sample_N, Case_differenceInfinite)
var test_sample_E2A = ZipSample(test_sample_E2, test_sample_A, Case_differenceInfinite)
var test_sample_E2E1 = ZipSample(test_sample_E2, test_sample_E1, Case_equal)
var test_sample_E2E2 = ZipSample(test_sample_E2, test_sample_E2, Case_equal|Case_equal_exact)
var test_sample_E2G = ZipSample(test_sample_E2, test_sample_gen, 0)

var test_sample_GN = ZipSample(test_sample_gen, test_sample_N, 0)
var test_sample_GA = ZipSample(test_sample_gen, test_sample_A, 0)
var test_sample_GE1 = ZipSample(test_sample_gen, test_sample_E1, 0)
var test_sample_GE2 = ZipSample(test_sample_gen, test_sample_E2, 0)
var test_sample_GG = ZipSample(test_sample_gen, test_sample_gen, Case_equal|Case_equal_exact)

var test_sample_gen = MakeSample1(
	&example_generator_xtw,
	0,
	"Example generator")

var test_sample_unintialized_xtw = MakeSample1(
	&Point_xtw{},
	PointFlags(Case_singular),
	"Uninitialized xtw")

var test_sample_uninitialized_axtw = MakeSample1(
	&Point_axtw{},
	PointFlags(Case_singular),
	"Uninitialized axtw")

var test_sample_uninitialized_efgh = MakeSample1(
	&Point_efgh{},
	PointFlags(Case_singular),
	"Uninitialized efgh")

func (s *TestSample) String() string {
	var ret string
	if s.Len == 0 {
		return "Empty test sample consisting of 0-tuple of points"
	}
	for i := 0; i < int(s.Len); i++ {
		ret += "Point "
		ret += strconv.Itoa(i + 1)
		ret += " of type "
		ret += PointTypeToString(GetPointType(s.Points[i]))
		ret += ", "

	}
	ret += "Comment stored in sample: "
	ret += s.Comment
	ret += "\n"
	for i := 0; i < int(s.Len); i++ {
		ret += "Representation of Point " + strconv.Itoa(i+1) + " (" + PointTypeToString(GetPointType(s.Points[i])) + ") is "
		ret += s.Points[i].String()
		if i+1 < int(s.Len) {
			ret += "\n"
		}
	}
	return ret
}

func make_random_test_sample_xtw(rnd *rand.Rand, subgroup bool) TestSample {
	r := makeRandomPointOnCurve_t(rnd)
	flags := PointFlags(Case_random)
	var comment string
	// s.Flags = PointFlags(Case_random)
	if subgroup {
		r.DoubleEq() // clear cofactor
		if rnd.Intn(2) == 0 {
			r.AddEq(&orderTwoPoint_xtw)
			flags |= PointFlags(Case_outside_p253)
			comment = "Random Point (good coset)"
		} else {
			comment = "Random Point (exact subgroup)"
		}
	} else {
		flags |= PointFlags(Case_outside_goodgroup)
		comment = "Random point (full curve)"
	}
	return MakeSample1(&r, flags, comment)
}

func make_random_test_sample(rnd *rand.Rand, subgroup bool, point_type PointType) TestSample {
	switch point_type {
	case pointTypeXTW:
		return make_random_test_sample_xtw(rnd, subgroup)
	default:
		r := make_random_test_sample_xtw(rnd, subgroup)
		return r.CopyXTWToType([]PointType{point_type})
	}
}

func make_singular_test_sample(point_type PointType) TestSample {
	switch point_type {
	case pointTypeXTW:
		return test_sample_unintialized_xtw
	case pointTypeAXTW:
		return test_sample_uninitialized_axtw
	case pointTypeEFGH:
		return test_sample_uninitialized_efgh
	default:
		panic("Unrecognized point type")
	}
}

func make_random_singular_sample_xtw(rnd *rand.Rand) TestSample {
	var p Point_xtw
	p.x.SetZero()
	p.y.SetZero()
	p.t.setRandomUnsafe(rnd)
	p.z.setRandomUnsafe(rnd)
	return MakeSample1(&p, Case_singular|Case_random, "Random singular xtw")
}

func make_random_singular_sample_axtw(rnd *rand.Rand) TestSample {
	var p Point_axtw
	p.x.SetZero()
	p.y.SetZero()
	p.t.setRandomUnsafe(rnd)
	return MakeSample1(&p, Case_singular|Case_random, "Random singular axtw")
}

func make_random_singular_sample_efgh(rnd *rand.Rand) TestSample {
	var p Point_efgh
	var s string
	switch rnd.Intn(3) {
	case 0:
		p.e.SetZero()
		p.f.setRandomUnsafe(rnd)
		p.g.setRandomUnsafe(rnd)
		p.h.SetZero()
		s = "Random singular efgh (e=h=0)"
	case 1:
		p.e.setRandomUnsafe(rnd)
		p.f.SetZero()
		p.g.setRandomUnsafe(rnd)
		p.h.SetZero()
		s = "Random singular efgh (f=h=0)"
	case 2:
		p.e.SetZero()
		p.f.setRandomUnsafe(rnd)
		p.g.SetZero()
		p.h.setRandomUnsafe(rnd)
		s = "Random singular efgh (e=g=0)"
	}
	return MakeSample1(&p, Case_singular|Case_random, s)
}

func make_random_singular(rnd *rand.Rand, point_type PointType) TestSample {
	switch point_type {
	case pointTypeXTW:
		return make_random_singular_sample_xtw(rnd)
	case pointTypeAXTW:
		return make_random_singular_sample_axtw(rnd)
	case pointTypeEFGH:
		return make_random_singular_sample_efgh(rnd)
	default:
		panic("Type not regognized")
	}
}

func MakeTestSamples1(random_size int, point_type1 PointType, exclude_flags PointFlags) (ret []TestSample) {
	var point_types []PointType = []PointType{point_type1}
	AppendTestSamples(&ret, exclude_flags, point_types, test_sample_N, test_sample_A, test_sample_E1, test_sample_E2, test_sample_gen)
	AppendTestSamples(&ret, exclude_flags, point_types, make_singular_test_sample(point_type1))

	var drng *rand.Rand = rand.New(rand.NewSource(100))
	for i := 0; i < random_size; i++ {
		AppendTestSamples(&ret, exclude_flags, point_types, make_random_test_sample(drng, false, point_type1))
		AppendTestSamples(&ret, exclude_flags, point_types, make_random_test_sample(drng, true, point_type1))
	}
	drng = rand.New(rand.NewSource(101))
	for i := 0; i < random_size; i++ {
		AppendTestSamples(&ret, exclude_flags, point_types, make_random_singular(drng, point_type1))
	}
	return
}

func MakeTestSamples3(random_size int, point_type1 PointType, point_type2 PointType, point_type3 PointType, exclude_flags PointFlags) (ret []TestSample) {
	var point_types []PointType = []PointType{point_type1, point_type2, point_type3}
	l2 := MakeTestSamples2(random_size, point_type1, point_type2, exclude_flags)
	var drng *rand.Rand = rand.New(rand.NewSource(301))
	for _, item := range l2 {
		p := make_random_test_sample(drng, true, point_type3)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(item, p, 0))
		p = make_random_test_sample(drng, false, point_type3)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(item, p, 0))
	}

	drng = rand.New(rand.NewSource(302))
	l2 = MakeTestSamples2(random_size, point_type2, point_type3, exclude_flags)
	for _, item := range l2 {
		p := make_random_test_sample(drng, true, point_type1)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(p, item, 0))
		p = make_random_test_sample(drng, false, point_type1)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(p, item, 0))
	}
	return
}

func MakeTestSamples2(random_size int, point_type1 PointType, point_type2 PointType, exclude_flags PointFlags) (ret []TestSample) {
	var point_types []PointType = []PointType{point_type1, point_type2}
	AppendTestSamples(&ret, exclude_flags, point_types,
		test_sample_NN, test_sample_NA, test_sample_NE1, test_sample_NE2, test_sample_NG,
		test_sample_AN, test_sample_AA, test_sample_AE1, test_sample_AE2, test_sample_AG,
		test_sample_E1N, test_sample_E1A, test_sample_E1E1, test_sample_E1E2, test_sample_E1G,
		test_sample_E2N, test_sample_E2A, test_sample_E2E1, test_sample_E2E2, test_sample_E2G,
		test_sample_GN, test_sample_GA, test_sample_GE1, test_sample_GE2, test_sample_GG)

	// Create sample of the form random, random + E1. This needs to be done in xtw coordinates and converted later.
	var drng *rand.Rand = rand.New(rand.NewSource(102))
	var s1, s2 TestSample
	s1 = make_random_test_sample(drng, false, pointTypeXTW)
	s1.Flags[0] |= Case_differenceInfinite
	s2.Len = 1
	s2.Comment = s1.Comment + ", differs by E1"
	s2.Flags = make([]PointFlags, 1)
	s2.Flags[0] = s1.Flags[0] | Case_outside_goodgroup
	s2.Points = make([]CurvePointPtrInterfaceRead_FullCurve, 1)

	var p2 Point_xtw
	p2.Add(s1.Points[0], &exceptionalPoint_1_xtw) // We might consider writing down the coos directly
	if p2.IsNaP() {
		panic("Error while creating sample points for tests")
	}
	s2.Points[0] = &p2

	AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, Case_differenceInfinite))
	AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s2, s1, Case_differenceInfinite)) // almost the same distribution as above except that we chose s1 in the good group.

	drng = rand.New(rand.NewSource(103))
	for i := 0; i < 5+random_size/4; i++ {
		s1 = make_random_test_sample(drng, true, point_type1)
		s2 = make_random_test_sample(drng, true, point_type2)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, false, point_type1)
		s2 = make_random_test_sample(drng, true, point_type2)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, true, point_type1)
		s2 = make_random_test_sample(drng, false, point_type2)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, false, point_type1)
		s2 = make_random_test_sample(drng, false, point_type2)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, true, pointTypeXTW)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s1, Case_equal|Case_equal_exact))

		s1 = make_random_test_sample(drng, false, pointTypeXTW)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s1, Case_equal|Case_equal_exact))

		s1 = make_random_test_sample(drng, true, pointTypeXTW)
		s2 = s1.Clone()
		p2.Neg(s1.Points[0])
		s2.Points[0] = &p2
		s2.Comment += "adding to 0"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, false, pointTypeXTW)
		s2 = s1.Clone()
		p2.Neg(s1.Points[0])
		s2.Points[0] = &p2
		s2.Comment += "adding to 0"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))

		s1 = make_random_test_sample(drng, true, pointTypeXTW)
		s2 = s1.Clone()
		p2.Add(s1.Points[0], &orderTwoPoint_xtw)
		s2.Points[0] = &p2
		s2.Comment += "P2 = P1+A"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, Case_outside_p253|Case_equal))

		s1 = make_random_test_sample(drng, false, pointTypeXTW)
		s2 = s1.Clone()
		p2.Add(s1.Points[0], &orderTwoPoint_xtw)
		s2.Points[0] = &p2
		s2.Comment += "P2 = P1+A"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, Case_outside_p253|Case_equal))

		s1 = make_random_test_sample(drng, true, pointTypeXTW)
		s2 = s1.Clone()
		p2.Sub(&orderTwoPoint_xtw, s1.Points[0])
		s2.Points[0] = &p2
		s2.Comment += "P2 + P1 = A"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, Case_outside_p253))

		s1 = make_random_test_sample(drng, false, pointTypeXTW)
		s2 = s1.Clone()
		p2.Sub(&orderTwoPoint_xtw, s1.Points[0])
		s2.Points[0] = &p2
		s2.Comment += "P2 + P1 = A"
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, Case_outside_p253))
	}

	drng = rand.New(rand.NewSource(201))
	ss2 := MakeTestSamples1(random_size, point_type2, exclude_flags)
	s1 = make_singular_test_sample(point_type1)
	for _, s2 = range ss2 {
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))
		for i := 0; i < 4; i++ {
			s1r1 := make_random_singular(drng, point_type1)
			AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1r1, s2, 0))
		}
	}

	drng = rand.New(rand.NewSource(202))
	ss1 := MakeTestSamples1(random_size, point_type1, exclude_flags)
	s2 = make_singular_test_sample(point_type2)
	for _, s1 = range ss1 {
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))
		for i := 0; i < 4; i++ {
			s2r1 := make_random_singular(drng, point_type2)
			AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2r1, 0))
		}
	}

	drng = rand.New(rand.NewSource(203))
	for i := 0; i < random_size; i++ {
		s1 = make_random_singular(drng, point_type1)
		s2 = make_random_singular(drng, point_type2)
		AppendTestSamples(&ret, exclude_flags, point_types, ZipSample(s1, s2, 0))
	}

	return
}

/*
func TestMakeSample(t *testing.T) {
	x := MakeTestSamples2(10, pointTypeXTW, pointTypeAXTW, Case_infinite)
	for _, item := range x {
		fmt.Println(item)
	}
}
*/

type checkfunction func(TestSample) (bool, string)

func run_tests_on_samples(f checkfunction, t *testing.T, samples []TestSample, err_string string) {
	var num_errors int = 0
	var failed bool = false
	for _, samp := range samples {
		pass, error_reason := f(samp)
		if failed && !pass {
			num_errors++
		}
		if !failed && !pass {
			failed = true
			t.Error(err_string + "\nAdditional info: " + error_reason + "\nFailed Sample: " + samp.String() + "\nPrinting Stack trace")
			debug.PrintStack()
		}
	}
	if failed {
		t.Fatal(" and " + strconv.Itoa(num_errors) + " further errors")
	}
}

func make_samples1_and_run_tests(t *testing.T, f checkfunction, err_string string, point_type1 PointType, random_size int, excluded_flags PointFlags) {
	Samples := MakeTestSamples1(random_size, point_type1, excluded_flags)
	run_tests_on_samples(f, t, Samples, err_string)
}

func make_samples2_and_run_tests(t *testing.T, f checkfunction, err_string string, point_type1 PointType, point_type2 PointType, random_size int, excluded_flags PointFlags) {
	Samples := MakeTestSamples2(random_size, point_type1, point_type2, excluded_flags)
	run_tests_on_samples(f, t, Samples, err_string)
}
