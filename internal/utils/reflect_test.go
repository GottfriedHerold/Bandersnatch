package utils

import (
	"reflect"
	"testing"
)

func TestTypeOfType(t *testing.T) {
	var x int
	xType := reflect.TypeOf(x)
	xType2 := TypeOfType[int]()
	if xType != xType2 {
		t.Fatalf("TypeOfType does not work as expected")
	}
	iType := TypeOfType[error]()
	if iType.Kind() != reflect.Interface {
		t.Fatalf("TypeOfTyped does not work for interface types")
	}
}
