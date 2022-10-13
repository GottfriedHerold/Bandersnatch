package exponents

import (
	"math/big"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

const CurveExponent = common.CurveExponent
const CurveOrder = common.CurveOrder
const GroupOrder = common.GroupOrder
const EndomorphismEigenvalue = common.EndomorphismEigenvalue

var CurveExponent_Int = common.CurveExponent_Int                   // Note: Copies pointer, but that's OK.
var GroupOrder_Int = common.GroupOrder_Int                         // Note: Copies pointer, but that's OK.
var EndomorphismEigenvalue_Int = common.EndomorphismEigenvalue_Int // Note: Copies pointer, but that's OK.
var CurveOrder_Int = common.CurveOrder_Int                         // Note: Copies pointer, but that's OK.

// EndomorphismEigenvalue is only defined modulo GroupOrder.
// We chose an odd representative. This info is needed to get some tests
// and assertions right.
const endomorphismEigenvalueIsOdd bool = (EndomorphismEigenvalue%2 == 0) // == true

const (
	curveExponent_0 = (CurveExponent >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	curveExponent_1
	curveExponent_2
	curveExponent_3
)

const (
	curveOrder_0 = (CurveOrder >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	curveOrder_1
	curveOrder_2
	curveOrder_3
)

const (
	groupOrder_0 = (GroupOrder >> (iota * 64)) & 0xFFFFFFFF_FFFFFFFF
	groupOrder_1
	groupOrder_2
	groupOrder_3
)

var twoTo128_Int *big.Int = utils.InitIntFromString("0x1_00000000_00000000_00000000_00000000")

// (p253-1)/2. We can represent Z/p253 by numbers from -halfGroupOrder, ... , + halfGroupOrder. This is used in the GLV decomposition algorithm.
const (
	halfGroupOrder        = (GroupOrder - 1) / 2
	halfGroupOrder_string = "6554484396890773809930967563523245729654577946720285125893201653364843836400"
)

// (p253-1)/2
var halfGroupOrder_Int = utils.InitIntFromString(halfGroupOrder_string)
