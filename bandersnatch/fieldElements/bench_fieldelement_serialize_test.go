package fieldElements

import (
	"bytes"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/utils"
)

func BenchmarkFESerialize_AllTypes(b *testing.B) {
	b.Run("SerializeFE_LittleEndian-genericMethod", utils.Bind2(benchmarkSerializeFieldElement_genericMethod[bsFieldElement_MontgomeryNonUnique], LittleEndian))
	b.Run("SerializeFE_LittleEndian-freefunction", utils.Bind2(benchmarkSerializeFieldElement_freeFun[bsFieldElement_MontgomeryNonUnique], LittleEndian))
	b.Run("SerializeFE_LittleEndian-specificMethod", utils.Bind2(benchmarkSerializeFieldElement_specificType, LittleEndian))

	b.Run("MontgomeryNonUnique", benchmarkFESerialize_all[bsFieldElement_MontgomeryNonUnique])
	// b.Run("big.Int Wrapper", benchmarkFESerialize_all[bsFieldElement_BigInt])
}

func benchmarkFESerialize_all[FE any, FEPtr interface {
	*FE
	FieldElementSerializeMethods
	FieldElementInterface[FEPtr]
}](b *testing.B) {

}

func benchmarkSerializeFieldElement_genericMethod[FE any, FEPtr interface {
	*FE
	FieldElementSerializeMethods
	FieldElementInterface[FEPtr]
}](b *testing.B, endianness FieldElementEndianness) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var buf bytes.Buffer
	buf.Grow(32 * benchS)

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			buf.Reset()
		}
		_, err := FEPtr(&bench_x[nn]).Serialize(&buf, endianness)
		if err != nil {
			b.Fatalf("Unexpected error")
		}
	}
}

func benchmarkSerializeFieldElement_freeFun[FE any, FEPtr interface {
	*FE
	FieldElementInterface[FEPtr]
}](b *testing.B, endianness FieldElementEndianness) {
	var bench_x []FE = GetPrecomputedFieldElements[FE, FEPtr](10001, benchS)
	var buf bytes.Buffer
	buf.Grow(32 * benchS)

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			buf.Reset()
		}
		_, err := SerializeFieldElement(FEPtr(&bench_x[nn]), &buf, endianness)
		if err != nil {
			b.Fatalf("Unexpected error")
		}
	}
}

func benchmarkSerializeFieldElement_specificType(b *testing.B, endianness FieldElementEndianness) {
	var bench_x []bsFieldElement_MontgomeryNonUnique = GetPrecomputedFieldElements[bsFieldElement_MontgomeryNonUnique](10001, benchS)
	var buf bytes.Buffer
	buf.Grow(32 * benchS)

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			buf.Reset()
		}
		_, err := bench_x[nn].Serialize(&buf, endianness)
		if err != nil {
			b.Fatalf("Unexpected error")
		}
	}
}
