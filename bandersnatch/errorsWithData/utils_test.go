package errorsWithData

// This file contains utils for tests

import (
	"errors"
	"maps"
	"testing"

	"github.com/GottfriedHerold/Bandersnatch/internal/testutils"
)

// helper function: check that the error e to be tested
//   - has ParamMap expectedParams (unless expectedParams["ignore"] == true). nil is treated as empty map.
//   - exports data *expectedData (unless s == nil)
//   - wraps (at least) all errors in wrapping
//   - e.Error() matches expectedString unless expectedString == "ignore"
func testError[StructType any](t *testing.T, e ErrorWithData[StructType], expectedString string, expectedData *StructType, expectedParams ParamMap, wrapping []error) {
	errorString := e.Error()
	data := e.GetData_struct() // need to contain in an interface
	if expectedData != nil {
		// Note: Wrapping data and *expectedData in any is because the generic StructType is contrained by any, not by comparable.
		// In fact, StructType might be not comparable; this is what setting expectedData to nil is for.
		testutils.FatalUnless(t, any(data) == any(*expectedData), "Unexpected data for %v, Expected:%v\nGot:%v", errorString, expectedData, data)
	}
	if expectedParams["ignore"] != true {
		m := e.GetData_map()
		testutils.FatalUnless(t, maps.Equal(m, expectedParams), "Unexpected param map for %v, Expected:%v\nGot:%v", errorString, expectedParams, m)
		for expectedParam, expectedVal := range expectedParams {
			testutils.FatalUnless(t, e.HasParameter(expectedParam), "%v does not have expected parameter %v", e, expectedParam)
			actualVal, present := e.GetParameter(expectedParam)
			testutils.FatalUnless(t, actualVal == expectedVal, "")
			testutils.FatalUnless(t, present == true, "")
		}
	}
	testutils.FatalUnless(t, e.HasParameter("not existing") == false, "%v has bogus parameter", e)
	actualVal, present := e.GetParameter("bogus")
	testutils.FatalUnless(t, actualVal == nil, "")
	testutils.FatalUnless(t, present == false, "")
	if expectedString != "ignore" {
		testutils.FatalUnless(t, expectedString == errorString, "Unexpected Error() output. Expected:%v\nGot:%v", expectedString, errorString)
	}
	for _, supposedWrapped := range wrapping {
		testutils.FatalUnless(t, errors.Is(e, supposedWrapped), "Error %v unexpectedly does not wrap %v", e, supposedWrapped)
	}
}

// helper function: check that the error e to be tested
//   - has ParamMap expectedParams (unless expectedParams["ignore"] == true). nil is treated as empty map.
//   - wraps (at least) all errors in wrapping
//   - e.Error() matches expectedString unless expectedString == "ignore"
func testError_any(t *testing.T, e ErrorWithData_any, expectedString string, expectedParams ParamMap, wrapping []error) {
	errorString := e.Error()
	if expectedParams["ignore"] != true {
		m := e.GetData_map()
		testutils.FatalUnless(t, maps.Equal(m, expectedParams), "Unexpected param map for %v, Expected:%v\nGot:%v", errorString, expectedParams, m)
	}
	if expectedString != "ignore" {
		testutils.FatalUnless(t, expectedString == errorString, "Unexpected Error() output. Expected:%v\nGot:%v", expectedString, errorString)
	}
	for _, supposedWrapped := range wrapping {
		testutils.FatalUnless(t, errors.Is(e, supposedWrapped), "Error %v unexpectedly does not wrap %v", e, supposedWrapped)
	}
}

// currently unused

// callOnErrorSubtree calls the passed function/closure f on each error in e's error tree (but not on e itself)
func callOnErrorSubtree(e error, f func(error)) {
	if e == nil {
		return
	}
	switch e2 := e.(type) {
	case interface {
		error
		Unwrap() error
	}:
		unwrapped := e2.Unwrap()
		callOnErrorTree(unwrapped, f)
	case interface {
		error
		Unwrap() []error
	}:
		unwrappedList := e2.Unwrap()
		for _, unwrapped := range unwrappedList {
			callOnErrorTree(unwrapped, f)
		}
	default:
		return
	}
}

// callOnErrorTree calls the passed function/closure f on each error in e's error tree (including e itself)
func callOnErrorTree(e error, f func(error)) {
	f(e)
	callOnErrorSubtree(e, f)
}
