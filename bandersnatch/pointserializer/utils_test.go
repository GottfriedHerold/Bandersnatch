package pointserializer

import (
	"fmt"
	"strings"
	"testing"
)

// keys in the global serializerParams act case-insensitve, which is implemented via normalization to lowercase. So the entries in the map must be lowercase.
func TestParamsLowercase(t *testing.T) {
	for key := range serializerParams {
		if key != strings.ToLower(key) {
			t.Fatalf("serializerParams has non-lowercased key %v", key)
		}
	}
}

func TestFoo(t *testing.T) {
	var x [4]byte = [4]byte{1, 2, 3, 4}
	var y []byte = make([]byte, 3)
	y = copyByteSlice(x[:])
	fmt.Println(y)
}
