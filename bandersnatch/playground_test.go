package bandersnatch

import (
	"fmt"
	"testing"
)

func TestPlayground(t *testing.T) {
	D := CurveParameterD_fe
	var A FieldElement
	A.SetUInt64(5)
	A.NegEq()
	var diff FieldElement
	diff.Sub(&A, &D)
	fmt.Println(diff.Jacobi())

}
