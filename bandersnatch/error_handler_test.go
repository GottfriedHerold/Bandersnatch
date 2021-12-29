package bandersnatch

import "testing"

func do_nothing() {

}

func call_error() {
	var p Point_xtw
	napEncountered("dummy error for error testing", false, &p)
}

var _ NaPErrorHandler = trivial_error_handler
var _ NaPErrorHandler = panic_error_handler

func TestErrorHandling(t *testing.T) {
	f := GetNaPErrorHandler()
	if f("", false) != false {
		t.Fatal("Predefined error handler does not return false")
	}
	var x int = 2
	new_handler := func(string, bool, ...CurvePointPtrInterfaceRead) bool {
		x += 3
		return false
	}
	SetNaPErrorHandler(new_handler)
	call_error()
	if x != 5 {
		t.Fatal("Installed error handler was not called")
	}
	g := GetNaPErrorHandler()
	g("", false)
	if x != 8 {
		t.Fatal("Did not get back installed error handler")
	}
	if wasInvalidPointEncountered(do_nothing) != false {
		t.Fatal("expectError should have returned false")
	}
	if wasInvalidPointEncountered(call_error) != true {
		t.Fatal("exptecError should have returned true")
	}
	if x != 11 {
		t.Fatal("Did not call error handler when expected to")
	}
	SetNaPErrorHandler(f)
}
