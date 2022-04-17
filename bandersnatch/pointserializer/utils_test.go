package pointserializer

import (
	"strings"
	"testing"
)


func TestParamsLowercase(t *testing.T) {
	for key := range serializerParams {
		if key != strings.ToLower(key) {
			t.Fatalf("serializerParams has non-lowercased key %v", key)
		}
	}
}
