package playground

/*
func getCofactors(P *Point_xtw) (tN, tA, t1, t2 FieldElement) {
	var tmp FieldElement
	tN.Sub(&P.z, &P.y)
	tA.Add(&P.z, &P.y)
	tmp.Mul(&squareRootDbyA_fe, &P.t)
	t1.Sub(&P.x, &tmp)
	t2.Add(&P.x, &tmp)
	return
}

func TestCofactorGroup(t *testing.T) {
	const iterations = 40
	drng := rand.New(rand.NewSource(103))

	for n := 0; n < iterations; n++ {

		var P Point_xtw = makeRandomPointInSubgroup_t(drng)
		P.DoubleEq()
		// P.AddEq(&exceptionalPoint_1_xtw)
		tN, tA, t1, t2 := getCofactors(&P)
		LtN := tN.Jacobi()
		LtA := tA.Jacobi()
		Lt1 := t1.Jacobi()
		Lt2 := t2.Jacobi()
		Lx := P.x.Jacobi()
		// Ly := P.y.Jacobi()
		// Lt := P.t.Jacobi()
		Lz := P.z.Jacobi()
		fmt.Println("Subgroup point ", LtN*Lz, " ", LtA*Lz, " ", Lt1*Lx, " ", Lt2*Lx, "---")
		// fmt.Println("Jacs ", Lx*Lz, " ", Ly*Lz, " ", Lt*Lz)
		// var tN, tA, t1, t2 tmp FieldElement
		// tN.Sub(&P.z, &P.y)
		// tA.Add(&P.z, &P.y)
		var tmp1, tmp2, yy FieldElement
		yy.Square(&P.y)
		tmp1.Square(&P.z)
		tmp2.Mul(&yy, &squareRootDbyA_fe)
		tmp1.AddEq(&tmp2)
		tmp2.Add(&FieldElementOne, &squareRootDbyA_fe)
		tmp2.MulEq(&P.y)
		tmp2.MulEq(&P.z)
		tmp1.SubEq(&tmp2)
		L := tmp1.Jacobi()
		fmt.Println("J: ", L)

	}
}

*/

/*
func TestStuff(t *testing.T) {
	f := func(x CurvePointSlice) bool {
		return true
	}
	var P [2]Point_axtw_subgroup
	f(CurvePointSlice_axtw_subgroup(P[:]))
}
*/
