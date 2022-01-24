package bandersnatch

import "testing"

var allTestPointTypes = []PointType{pointTypeXTWFull, pointTypeXTWSubgroup, pointTypeAXTWFull, pointTypeAXTWSubgroup, pointTypeEFGHFull, pointTypeEFGHSubgroup}
var allXTWTestPointTypes = []PointType{pointTypeXTWFull, pointTypeXTWSubgroup}
var allAXTWTestPointTypes = []PointType{pointTypeAXTWFull, pointTypeAXTWSubgroup}
var allEFGHTestPointTypes = []PointType{pointTypeEFGHFull, pointTypeEFGHSubgroup}
var allFullCurveTestPointTypes = []PointType{pointTypeXTWFull, pointTypeAXTWFull, pointTypeEFGHFull}
var allSubgroupCurveTestPointTypes = []PointType{pointTypeXTWSubgroup, pointTypeAXTWSubgroup, pointTypeEFGHSubgroup}
var allBasePointTypes = []PointType{pointTypeXTWBase, pointTypeAXTWBase, pointTypeEFGHBase}

func TestAllTestPointTypesSatisfyInterface(t *testing.T) {
	for _, pointType := range allBasePointTypes {
		_, ok := MakeCurvePointPtrInterfaceFromType(pointType).(CurvePointPtrInterfaceBaseRead)
		if !ok {
			t.Fatal("Base Point type not compatible with curvePointPtrInterfaceBaseRead")
		}
	}

	for _, pointType := range allTestPointTypes {
		pointInstance, ok := MakeCurvePointPtrInterfaceFromType(pointType).(curvePointPtrInterfaceTestSample)
		if !ok {
			t.Fatal("Point type not compatible with curvePointPtrInterfaceTestSample " + PointTypeToString(pointType))
		}
		// Note that pointInstance is nil (of the appropriate type).
		// So this also tests that certain functions can be called with nil receivers.

		// TODO: This might go away together with CanRepresentInfinity.
		if pointInstance.CanRepresentInfinity() {
			_, ok = pointInstance.(CurvePointPtrInterfaceReadCanDistinguishInfinity)
			if !ok {
				t.Fatal("Curve point type can represent infinity, but does not provide interface to distinguish")
			}
		}

		if pointInstance.HasDecaf() {
			_, ok = pointInstance.(CurvePointPtrInterfaceDecaf)
			if !ok {
				t.Fatal("Curve point type has HasDecaf() true, but type type does not satisfy interface")
			}
		}

	}

}
