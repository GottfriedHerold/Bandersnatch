package bandersnatch

func legendreCheckA_affineX(x FieldElement) bool {
	// x is passed by value. We use it as a temporary.
	x.SquareEq()
	x.multiply_by_five()
	x.AddEq(&FieldElementOne) // 1 + 5x^2 = 1-ax^2
	return x.Jacobi() >= 0    // cannot be ==0, since a is a non-square
}

func legendreCheckA_projectiveXZ(x FieldElement, z FieldElement) bool {
	// x, z are passed by value. We use them as temporaries.
	x.SquareEq()
	x.multiply_by_five()
	z.SquareEq()
	x.AddEq(&z)
	return x.Jacobi() >= 0
}

func legendreCheckA_EG(e FieldElement, g FieldElement) bool {
	e.SquareEq()
	g.SquareEq()
	e.multiply_by_five()
	g.AddEq(&e)
	return g.Jacobi() >= 0
}

func legendreCheckE1_affineY(y FieldElement) bool {
	// TODO: formula not optimized

	var acc FieldElement
	// r := \sqrt(d/a), for a fixed particular choice of square root
	acc.Square(&y)                // y^2
	acc.MulEq(&squareRootDbyA_fe) // r * y^2
	var rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	// y passed by value
	y.MulEq(&rAndOne)
	acc.SubEq(&y)               // -(r+1)y + r*y^2
	acc.AddEq(&FieldElementOne) // ry^2 -(r+1)y + 1
	// Note: This is zero for the neutral element
	return acc.Jacobi() <= 0
}

func legendreCheckE1_projectiveYZ(y FieldElement, z FieldElement) bool {
	var acc, temp, rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	acc.Square(&z)
	temp.Mul(&y, &z)
	temp.MulEq(&rAndOne)
	acc.SubEq(&temp)
	temp.Square(&y)
	temp.MulEq(&squareRootDbyA_fe)
	acc.AddEq(&temp)
	// Note: acc is zero for neutral element and *both* points at infinity.
	return acc.Jacobi() <= 0
}

func legendreCheckE1_FH(f FieldElement, h FieldElement) bool {
	// identical to YZ, actually. But special cases resulting in 0 differ.
	var acc, temp, rAndOne FieldElement
	rAndOne.Add(&FieldElementOne, &squareRootDbyA_fe)
	acc.Square(&f)
	temp.Mul(&h, &f)
	temp.MulEq(&rAndOne)
	acc.SubEq(&temp)
	temp.Square(&h)
	temp.MulEq(&squareRootDbyA_fe)
	acc.AddEq(&temp)
	// Note: acc is zero for neutral element and E1.
	return acc.Jacobi() <= 0
}
