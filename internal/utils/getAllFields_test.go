package utils

import (
	"reflect"
	"testing"
)

func TestAllFields(t *testing.T) {
	type A struct {
		x int
		y bool
	}
	type Empty struct {
	}
	type AEmbedded struct {
		A
		x uint
	}
	type Chain struct {
		AEmbedded
		*A
		Empty
	}
	type foo int
	type EmbeddedInt struct {
		foo
	}
	type EmbeddedIntPtr struct {
		*foo
	}

	intType := TypeOfType[int]()
	boolType := TypeOfType[bool]()
	uintType := TypeOfType[uint]()

	var (
		AType              reflect.Type = TypeOfType[A]()
		EmptyType          reflect.Type = TypeOfType[Empty]()
		AEmbeddedType      reflect.Type = TypeOfType[AEmbedded]()
		ChainType          reflect.Type = TypeOfType[Chain]()
		fooType            reflect.Type = TypeOfType[foo]()
		EmbeddedIntType    reflect.Type = TypeOfType[EmbeddedInt]()
		EmbeddedIntPtrType reflect.Type = TypeOfType[EmbeddedIntPtr]()
	)

	checkFun := func(typ reflect.Type, embeddedStructPointer bool, expected []reflect.StructField) {
		f, emb := AllFields(typ)
		if emb != embeddedStructPointer {
			t.Fatalf("Unexpected answer for embeddedStructPointer for %v\n Expected %v, got %v", typ.Name(), embeddedStructPointer, emb)
		}
		if len(f) != len(expected) {
			t.Fatalf("Unexpected lengths of returned field slice for %v\n Expected %v, got %v. Returned slice is %v", typ.Name(), len(expected), len(f), f)
		}
		for i, field := range f {
			if expected[i].Name != field.Name {
				t.Fatalf("Unexpected Name in %v'th returned field for %v. Expected %v, got %v", i, typ.Name(), expected[i].Name, field.Name)
			}
			if expected[i].Anonymous != field.Anonymous {
				t.Fatalf("Unexpected Embeddedness in %v'th returned field for %v. Expected %v, got %v", i, typ.Name(), expected[i].Anonymous, field.Anonymous)
			}
			if expected[i].Type != field.Type {
				t.Fatalf("Unexpected Type in %v'th returned field for %v. Expected %v, got %v", i, typ.Name(), expected[i].Type, field.Type)
			}
			if CompareSlices(expected[i].Index, field.Index) == false {
				t.Fatalf("Unexpected Index sequence in %v'th returned field for %v. Expected %v, got %v", i, typ.Name(), expected[i].Index, field.Index)
			}
		}
	}
	checkFun(AType, false, []reflect.StructField{
		{Name: "x", Index: []int{0}, Type: intType},
		{Name: "y", Index: []int{1}, Type: boolType},
	})
	checkFun(EmptyType, false, []reflect.StructField{})
	checkFun(AEmbeddedType, false, []reflect.StructField{
		{Name: "A", Index: []int{0}, Type: AType, Anonymous: true},
		{Name: "x", Index: []int{0, 0}, Type: intType},
		{Name: "y", Index: []int{0, 1}, Type: boolType},
		{Name: "x", Index: []int{1}, Type: uintType},
	})
	checkFun(ChainType, true, []reflect.StructField{
		{Name: "AEmbedded", Index: []int{0}, Type: AEmbeddedType, Anonymous: true},
		{Name: "A", Index: []int{0, 0}, Type: AType, Anonymous: true},
		{Name: "x", Index: []int{0, 0, 0}, Type: intType},
		{Name: "y", Index: []int{0, 0, 1}, Type: boolType},
		{Name: "x", Index: []int{0, 1}, Type: uintType},
		{Name: "A", Index: []int{1}, Type: reflect.PointerTo(AType), Anonymous: true},
		{Name: "Empty", Index: []int{2}, Type: EmptyType, Anonymous: true},
	})
	checkFun(EmbeddedIntType, false, []reflect.StructField{
		{Name: "foo", Index: []int{0}, Type: fooType, Anonymous: true},
	})
	checkFun(EmbeddedIntPtrType, false, []reflect.StructField{
		{Name: "foo", Index: []int{0}, Type: reflect.PointerTo(fooType), Anonymous: true},
	})

}
