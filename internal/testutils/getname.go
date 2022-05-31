package testutils

import (
	"fmt"
	"reflect"
)

// GetReflectName tries to obtain a string representation of the given type using the reflection package.
// It covers more cases that plain c.Name() does (which only works for defined types and fails for e.g. pointers to defined types).
// We only use it for better diagnostic messages in testing.
func GetReflectName(c reflect.Type) (ret string) {
	// reflect.Type's  Name() only works for defined types, which
	// e.g. *Point_xtw_full is not. (Only Point_xtw_full is a defined type)
	ret = c.Name()
	if ret != "" {
		return
	}

	switch c.Kind() {
	case reflect.Pointer: // used to be called reflect.Ptr in Go1.17; reflect.Ptr is deprecated
		return "*" + GetReflectName(c.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%v]%v", c.Len(), GetReflectName(c.Elem()))
	case reflect.Slice:
		return fmt.Sprintf("[]%v", GetReflectName(c.Elem()))
	default:
		return "<<type with unknown name>>"
	}
}
