package bandersnatch

/*
func convertToPoint_xtw(input CurvePointPtrInterfaceRead) Point_xtw {
	switch input := input.(type) {
	case *Point_efgh:
		return input.ToDecaf_xtw()
	case CurvePointPtrInterfaceReadConvertToDecaf:
		return input.ToDecaf_xtw()
	default:
		// TODO !
		panic("Cannot convert to xtw")
	}
}

func convertToPoint_axtw(input CurvePointPtrInterfaceRead) Point_axtw {
	switch input := input.(type) {
	case *Point_efgh:
		return input.ToDecaf_axtw()
	case CurvePointPtrInterfaceReadConvertToDecaf:
		return input.ToDecaf_axtw()
	default:
		// TODO !
		panic("Not implemented yet: Cannot convert to axtw")
	}
}

func getDecafXYZ(input CurvePointPtrInterfaceRead) (X, Y, Z FieldElement) {
	switch input := input.(type) {
	case CurvePointPtrInterfaceCooReadDecafProjective:
		X = input.X_decaf_projective()
		Y = input.Y_decaf_projective()
		Z = input.Z_decaf_projective()
	default:
		X, Y, Z = input.XYZ_projective()
	}
	return
}
*/
