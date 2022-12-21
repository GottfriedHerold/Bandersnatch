package fieldElements

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/bandersnatch/common"
	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

func BenchmarkUint256Serialization(b *testing.B) {
	b.Run("SerializeUint256_LittleEndian", func(b *testing.B) { benchmarkUint256_serialize_SerializeToBuffer(b, LittleEndian) })
	b.Run("SerializeUint256_BigEndian", func(b *testing.B) { benchmarkUint256_serialize_SerializeToBuffer(b, BigEndian) })
	b.Run("SerializeUint256Prefix_LittleEndian", func(b *testing.B) { benchmarkUint256_serialize_SerializeWithPrefix(b, LittleEndian) })
	b.Run("SerializeUint256Prefix_BigEndian", func(b *testing.B) { benchmarkUint256_serialize_SerializeWithPrefix(b, BigEndian) })
	b.Run("DeserializeUint256_LittleEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeFromBuffer(b, LittleEndian) })
	b.Run("DeserializeUint256_BigEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeFromBuffer(b, BigEndian) })
	b.Run("DeserializeUint256GetPrefix_LittleEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeAndGetPrefix(b, LittleEndian) })
	b.Run("DeserializeUint256ExpectedPrefix_BigEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeWithExpectedPrefix(b, BigEndian) })
	b.Run("DeserializeUint256ExpectedPrefix_LittleEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeWithExpectedPrefix(b, LittleEndian) })
	b.Run("DeserializeUint256ExpectedPrefix_BigEndian", func(b *testing.B) { benchmarkUint256_serialize_DeserializeWithExpectedPrefix(b, BigEndian) })
}

func benchmarkUint256_serialize_SerializeToBuffer(b *testing.B, endianness FieldElementEndianness) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var buf bytes.Buffer
	buf.Grow(32 * 256)

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			buf.Reset()
		}
		_, err := bench_x[nn].Serialize(&buf, endianness)
		if err != nil {
			b.Fatalf("unexpected error")
		}
	}
}

func benchmarkUint256_serialize_SerializeWithPrefix(b *testing.B, endianness FieldElementEndianness) {
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	var buf bytes.Buffer
	buf.Grow(32 * 256)
	prefix := common.MakeBitHeader(common.PrefixBits(0b10), 2)
	for i := 0; i < benchS; i++ {
		bench_x[i][3] &= 0x3FFFFFFF_FFFFFFFF
	}

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			buf.Reset()
		}
		_, err := bench_x[nn].SerializeWithPrefix(&buf, prefix, endianness)
		if err != nil {
			b.Fatalf("unexpected error")
		}
	}
}

func benchmarkUint256_serialize_DeserializeFromBuffer(b *testing.B, endianness FieldElementEndianness) {
	var data []byte = make([]byte, 32*benchS)
	var dataCopy []byte = make([]byte, 32*benchS)
	rng := rand.New(rand.NewSource(10001))

	_, errRng := rng.Read(data)
	testutils.Assert(errRng == nil)
	var buf *bytes.Buffer

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			copy(dataCopy, data)
			buf = bytes.NewBuffer(dataCopy)
		}
		_, err := DumpUint256[nn].Deserialize(buf, endianness)
		if err != nil {
			b.Fatalf("unexpected error")
		}
	}
}

func benchmarkUint256_serialize_DeserializeAndGetPrefix(b *testing.B, endianness FieldElementEndianness) {
	var data []byte = make([]byte, 32*benchS)
	var dataCopy []byte = make([]byte, 32*benchS)
	rng := rand.New(rand.NewSource(10001))

	_, errRng := rng.Read(data)
	testutils.Assert(errRng == nil)
	var buf *bytes.Buffer

	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			copy(dataCopy, data)
			buf = bytes.NewBuffer(dataCopy)
		}
		_, prefix, err := DumpUint256[nn].DeserializeAndGetPrefix(buf, 2, endianness)
		_ = prefix
		if err != nil {
			b.Fatalf("unexpected error")
		}
	}
}

func benchmarkUint256_serialize_DeserializeWithExpectedPrefix(b *testing.B, endianness FieldElementEndianness) {
	prefix := common.MakeBitHeader(common.PrefixBits(0b10), 2)
	var bench_x []Uint256 = CachedUint256.GetElements(pc_uint256_a, benchS)
	for i := 0; i < benchS; i++ {
		bench_x[i][3] &= 0x3FFFFFFF_FFFFFFFF
	}
	var initBuf bytes.Buffer
	for _, x := range bench_x {
		_, errInit := x.SerializeWithPrefix(&initBuf, prefix, endianness)
		testutils.Assert(errInit == nil)
	}

	var data []byte = initBuf.Bytes()
	testutils.Assert(len(data) == 32*benchS)
	var dataCopy []byte = make([]byte, 32*benchS)

	var buf *bytes.Buffer
	prepareBenchmarkFieldElements(b)
	for n := 0; n < b.N; n++ {
		nn := n % benchS
		if nn == 0 {
			copy(dataCopy, data)
			buf = bytes.NewBuffer(dataCopy)
		}
		_, err := DumpUint256[nn].DeserializeWithExpectedPrefix(buf, prefix, endianness)
		if err != nil {
			b.Fatalf("unexpected error")
		}
	}
}