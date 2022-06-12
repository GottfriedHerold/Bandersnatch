package bandersnatch

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"math/rand"
	"testing"
)

func TestSerializeFieldElements(t *testing.T) {
	const iterations = 100
	var drng *rand.Rand = rand.New(rand.NewSource(87))
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe bsFieldElement_64
		fe.SetRandomUnsafe(drng)
		// do little endian and big endian half the time
		var byteOrder binary.ByteOrder = binary.LittleEndian
		if i%2 == 0 {
			byteOrder = binary.BigEndian
		}
		bytes_written, err := fe.Serialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Serialization of field element failed with error ", err)
		}
		if bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element did not write exptected number of bytes")
		}
		var fe2 bsFieldElement_64
		bytes_read, err := fe2.Deserialize(&buf, byteOrder)
		if err != nil {
			t.Fatal("Deserialization of field element failed with error ", err)
		}
		if bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element did not read expceted number of bytes")
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Deserializing of field element did not reproduce what was serialized")

		}
	}
	for i := 0; i < iterations; i++ {
		var buf bytes.Buffer
		var fe, fe2 bsFieldElement_64
		fe.SetRandomUnsafe(drng)
		if fe.Sign() < 0 {
			fe.NegEq()
		}
		if fe.Sign() < 0 {
			t.Fatal("Sign does not work as expected")
		}
		if bits.LeadingZeros64(fe.undoMontgomery()[3]) < 2 {
			t.Fatal("Positive sign field elements do not start with 00")
		}
		var random_prefix prefixBits = (prefixBits(i) / 2) % 4
		var byteOrder binary.ByteOrder = binary.LittleEndian
		if i%2 == 0 {
			byteOrder = binary.BigEndian
		}

		bytes_written, err := fe.SerializeWithPrefix(&buf, PrefixBits(random_prefix), 2, byteOrder)
		if err != nil || bytes_written != BaseFieldByteLength {
			t.Fatal("Serialization of field element failed with long prefix: ", err)
		}
		bytes_read, err := fe2.DeserializeWithPrefix(&buf, PrefixBits(random_prefix), 2, byteOrder)
		if err != nil || bytes_read != BaseFieldByteLength {
			t.Fatal("Deserialization of field element failed with long prefix: ", err)
		}
		if !fe.IsEqual(&fe2) {
			t.Fatal("Roundtripping field elements failed with long prefix")
		}
		buf.Reset() // not really needed
		bytes_written, err = fe.SerializeWithPrefix(&buf, PrefixBits(1), 1, byteOrder)
		if bytes_written != BaseFieldByteLength || err != nil {
			t.Fatal("Serialization of field elements failed on resetted buffer")
		}
		_, err = fe2.DeserializeWithPrefix(&buf, PrefixBits(0), 1, byteOrder)
		if err != ErrPrefixMismatch {
			t.Fatal("Prefix mismatch was not detected in deserialization of field elements")
		}
		buf.Reset()
		fe.Serialize(&buf, binary.BigEndian)
		buf.Bytes()[0] |= 0x80
		bytes_read, err = fe2.Deserialize(&buf, binary.BigEndian)
		if bytes_read != BaseFieldByteLength || err != ErrNonNormalizedDeserialization {
			t.Fatal("Non-normalized field element not recognized as such during deserialization")
		}
	}
}
