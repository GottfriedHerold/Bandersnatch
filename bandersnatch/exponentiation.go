//go:build ignore

package bandersnatch

import "github.com/GottfriedHerold/Bandersnatch/internal/testutils"

// Note: all exponentiation implementations here assume that the points are in the subgroup.
// The individual Exponentiate - functions defined on point types need to take care of non-subgroup cases:
// Basically, we can just remove the lsb of the exponent and square the base to reduce to the subgroup case.

const simpleSlidingWindowSize = 3

func exponentiate_slidingWindow(arg CurvePointPtrInterfaceRead, exponent *Exponent) (ret Point_efgh_subgroup) {
	const k = simpleSlidingWindowSize
	testutils.Assert(arg.CanOnlyRepresentSubgroup())
	glv := GLV_representation(exponent)
	u_decomp := decomposeUnalignedSignedAdic(glv.U, simpleSlidingWindowSize)
	v_decomp := decomposeUnalignedSignedAdic(glv.V, simpleSlidingWindowSize)
	const precomputedTableSize = 1 << (k - 1)
	// precompute all odd k-bit powers of arg
	var table [precomputedTableSize]Point_xtw_subgroup
	var p2 Point_xtw_subgroup
	table[0].SetFrom(arg)
	p2.Double(&table[0])
	for i := 1; i < precomputedTableSize; i++ {
		table[i].Add(&table[i-1], &p2)
	}
	var doublingsRemaining int = -1
	var nextU, nextV int
	var nextUExponent, nextVExponent int
	nextU = len(u_decomp) - 1
	nextV = len(v_decomp) - 1
	var uLeft bool = (nextU >= 0)
	var vLeft bool = (nextV >= 0)
	// var accumulator Point_efgh_subgroup
	if uLeft {
		nextUExponent = int(u_decomp[nextU].position)
	} else {
		nextUExponent = -1
	}

	if vLeft {
		nextVExponent = int(v_decomp[nextV].position)
	} else {
		nextVExponent = -1
	}

	// We want to maintaing the following invariants:
	// 2^doublingsRemaing * ret + sum_i^last_u u_decomp[i]*arg + \sum_j^last_v v_decomp[j] * Endo(arg)
	// nextUExponent / nextVExponent is the largest exponent appearing in u_decomp resp. v_decomp
	// uLeft resp. vLeft are equivalent to lastU >= 0 resp. lastV >= 0.
	// If uLeft resp. vLeft are false, we set nextVExponent resp. nextUExponent to -1

	if nextUExponent > nextVExponent {
		doublingsRemaining = nextUExponent
		tableIndex := (u_decomp[nextU].coeff - 1) / 2
		ret.SetFrom(&table[tableIndex])
		if u_decomp[nextU].sign < 0 {
			ret.NegEq()
		}
		nextU--
		uLeft = (nextU >= 0)
		if uLeft {
			nextUExponent = int(u_decomp[nextU].position)
		} else {
			nextUExponent = -1
		}
	} else if nextUExponent < nextVExponent {
		doublingsRemaining = nextVExponent
		tableIndex := (v_decomp[nextV].coeff - 1) / 2
		ret.Endo(&table[tableIndex])
		if v_decomp[nextV].sign < 0 {
			ret.NegEq()
		}
		nextV--
		vLeft = (nextV >= 0)
		if vLeft {
			nextVExponent = int(v_decomp[nextV].position)
		} else {
			nextVExponent = -1
		}
	} else { // nextUExponent == nextVExponent
		if nextUExponent == -1 {
			ret.SetNeutral()
			return
		}
		doublingsRemaining = nextUExponent
		tableIndexV := (v_decomp[nextV].coeff - 1) / 2
		ret.Endo(&table[tableIndexV])
		if v_decomp[nextV].sign < 0 {
			ret.NegEq()
		}
		tableIndexU := (u_decomp[nextU].coeff - 1) / 2
		if u_decomp[nextU].sign < 0 {
			ret.SubEq(&table[tableIndexU])
		} else {
			ret.AddEq(&table[tableIndexU])
		}
		nextU--
		nextV--
		uLeft = (nextU >= 0)
		vLeft = (nextV >= 0)
		if uLeft {
			nextUExponent = int(u_decomp[nextU].position)
		} else {
			nextUExponent = -1
		}
		if vLeft {
			nextVExponent = int(v_decomp[nextV].position)
		} else {
			nextVExponent = -1
		}
	}
	testutils.Assert(doublingsRemaining >= 0)

	for doublingsRemaining > 0 {
		ret.DoubleEq()
		doublingsRemaining--
		if doublingsRemaining == nextUExponent {
			tableIndex := (u_decomp[nextU].coeff - 1) / 2
			if u_decomp[nextU].sign > 0 {
				ret.AddEq(&table[tableIndex])
			} else {
				ret.SubEq(&table[tableIndex])
			}
			nextU--
			uLeft = (nextU >= 0)
			if uLeft {
				nextUExponent = int(u_decomp[nextU].position)
			} else {
				nextUExponent = -1
			}
		}

		if doublingsRemaining == nextVExponent {
			tableIndex := (v_decomp[nextV].coeff - 1) / 2
			var temp Point_efgh_subgroup
			temp.Endo(&table[tableIndex])
			if v_decomp[nextV].sign > 0 {
				ret.AddEq(&temp)
			} else {
				ret.SubEq(&temp)
			}
			nextV--
			vLeft = (nextV >= 0)
			if vLeft {
				nextVExponent = int(v_decomp[nextV].position)
			} else {
				nextVExponent = -1
			}
		}
	}
	return
}
