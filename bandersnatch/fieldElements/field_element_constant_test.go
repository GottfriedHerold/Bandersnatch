package fieldElements

import (
	"math/big"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// We make copies of all our (exported and internal) "constant" arrays/structs etc..
// Since Go lacks const-arrays or structs, these may theoretically be modified (for constants of pointer type such *big.Int, we need to check the pointed-to value)
// 

var (
	BaseFieldSize_Int_COPY     = BaseFieldSize_Int
	BaseFieldSize_Int_DEEPCOPY = new(big.Int).Set(BaseFieldSize_Int)
)
var (
	baseFieldSize_Int_COPY     = baseFieldSize_Int
	baseFieldSize_Int_DEEPCOPY = new(big.Int).Set(baseFieldSize_Int)
)

var (
	BaseFieldSize_64_COPY = BaseFieldSize_64
	BaseFieldSize_32_COPY = BaseFieldSize_32
	BaseFieldSize_16_COPY = BaseFieldSize_16
	BaseFieldSize_8_COPY  = BaseFieldSize_8
)

func TestEnsureFieldElementConstantsWereNotChanged(t *testing.T) {
	ensureFieldElementConstantsWereNotChanged()
}

func ensureFieldElementConstantsWereNotChanged() {
	testutils.Assert(BaseFieldSize_Int_COPY == BaseFieldSize_Int)
	testutils.Assert(BaseFieldSize_Int_DEEPCOPY.Cmp(BaseFieldSize_Int) == 0)
	testutils.Assert(baseFieldSize_Int_COPY == baseFieldSize_Int)
	testutils.Assert(baseFieldSize_Int_DEEPCOPY.Cmp(baseFieldSize_Int) == 0)
	testutils.Assert(baseFieldSize_Int_DEEPCOPY.Cmp(baseFieldSize_Int_DEEPCOPY) == 0)

	testutils.Assert(BaseFieldSize_64 == BaseFieldSize_64_COPY)
	testutils.Assert(BaseFieldSize_32 == BaseFieldSize_32_COPY)
	testutils.Assert(BaseFieldSize_16 == BaseFieldSize_16_COPY)
	testutils.Assert(BaseFieldSize_8 == BaseFieldSize_8_COPY)

	testutils.Assert(&BaseFieldSize_64[0] != &BaseFieldSize_64_COPY[0])
}
