// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated with github.com/stretchr/testify/_codegen; DO NOT EDIT.

package require

import (
	assert "github.com/stretchr/testify/assert"
	"go.temporal.io/server/common/proptest"

	http "net/http"
	url "net/url"
	time "time"
)

// Condition uses a Comparison to assert a complex condition.
func Condition(ctx proptest.RunContext, comp assert.Comparison, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Condition", comp, msgAndArgs)
}

// Conditionf uses a Comparison to assert a complex condition.
func Conditionf(ctx proptest.RunContext, comp assert.Comparison, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Conditionf", comp, msg, args)
}

// Contains asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	a.Contains("Hello World", "World")
//	a.Contains(["Hello", "World"], "World")
//	a.Contains({"Hello": "World"}, "Hello")
func Contains(ctx proptest.RunContext, s interface{}, contains interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Contains", s, contains, msgAndArgs)
}

// Containsf asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	a.Containsf("Hello World", "World", "error message %s", "formatted")
//	a.Containsf(["Hello", "World"], "World", "error message %s", "formatted")
//	a.Containsf({"Hello": "World"}, "Hello", "error message %s", "formatted")
func Containsf(ctx proptest.RunContext, s interface{}, contains interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Containsf", s, contains, msg, args)
}

// DirExists checks whether a directory exists in the given path. It also fails
// if the path is a file rather a directory or there is an error checking whether it exists.
func DirExists(ctx proptest.RunContext, path string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("DirExists", path, msgAndArgs)
}

// DirExistsf checks whether a directory exists in the given path. It also fails
// if the path is a file rather a directory or there is an error checking whether it exists.
func DirExistsf(ctx proptest.RunContext, path string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("DirExistsf", path, msg, args)
}

// ElementsMatch asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// a.ElementsMatch([1, 3, 2, 3], [1, 3, 3, 2])
func ElementsMatch(ctx proptest.RunContext, listA interface{}, listB interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ElementsMatch", listA, listB, msgAndArgs)
}

// ElementsMatchf asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// a.ElementsMatchf([1, 3, 2, 3], [1, 3, 3, 2], "error message %s", "formatted")
func ElementsMatchf(ctx proptest.RunContext, listA interface{}, listB interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ElementsMatchf", listA, listB, msg, args)
}

// Empty asserts that the specified object is empty.  I.e. nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	a.Empty(obj)
func Empty(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Empty", object, msgAndArgs)
}

// Emptyf asserts that the specified object is empty.  I.e. nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	a.Emptyf(obj, "error message %s", "formatted")
func Emptyf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Emptyf", object, msg, args)
}

// Equal asserts that two objects are equal.
//
//	a.Equal(123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Equal", expected, actual, msgAndArgs)
}

// EqualError asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	a.EqualError(err,  expectedErrorString)
func EqualError(ctx proptest.RunContext, theError error, errString string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualError", theError, errString, msgAndArgs)
}

// EqualErrorf asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	a.EqualErrorf(err,  expectedErrorString, "error message %s", "formatted")
func EqualErrorf(ctx proptest.RunContext, theError error, errString string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualErrorf", theError, errString, msg, args)
}

// EqualExportedValues asserts that the types of two objects are equal and their public
// fields are also equal. This is useful for comparing structs that have private fields
// that could potentially differ.
//
//	 type S struct {
//		Exported     	int
//		notExported   	int
//	 }
//	 a.EqualExportedValues(S{1, 2}, S{1, 3}) => true
//	 a.EqualExportedValues(S{1, 2}, S{2, 3}) => false
func EqualExportedValues(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualExportedValues", expected, actual, msgAndArgs)
}

// EqualExportedValuesf asserts that the types of two objects are equal and their public
// fields are also equal. This is useful for comparing structs that have private fields
// that could potentially differ.
//
//	 type S struct {
//		Exported     	int
//		notExported   	int
//	 }
//	 a.EqualExportedValuesf(S{1, 2}, S{1, 3}, "error message %s", "formatted") => true
//	 a.EqualExportedValuesf(S{1, 2}, S{2, 3}, "error message %s", "formatted") => false
func EqualExportedValuesf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualExportedValuesf", expected, actual, msg, args)
}

// EqualValues asserts that two objects are equal or convertible to the larger
// type and equal.
//
//	a.EqualValues(uint32(123), int32(123))
func EqualValues(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualValues", expected, actual, msgAndArgs)
}

// EqualValuesf asserts that two objects are equal or convertible to the larger
// type and equal.
//
//	a.EqualValuesf(uint32(123), int32(123), "error message %s", "formatted")
func EqualValuesf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EqualValuesf", expected, actual, msg, args)
}

// Equalf asserts that two objects are equal.
//
//	a.Equalf(123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Equalf", expected, actual, msg, args)
}

// Error asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.Error(err) {
//		   assert.Equal(t, expectedError, err)
//	  }
func Error(ctx proptest.RunContext, err error, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Error", err, msgAndArgs)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target, and if so, sets target to that error value.
// This is a wrapper for errors.As.
func ErrorAs(ctx proptest.RunContext, err error, target interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorAs", err, target, msgAndArgs)
}

// ErrorAsf asserts that at least one of the errors in err's chain matches target, and if so, sets target to that error value.
// This is a wrapper for errors.As.
func ErrorAsf(ctx proptest.RunContext, err error, target interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorAsf", err, target, msg, args)
}

// ErrorContains asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	a.ErrorContains(err,  expectedErrorSubString)
func ErrorContains(ctx proptest.RunContext, theError error, contains string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorContains", theError, contains, msgAndArgs)
}

// ErrorContainsf asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	a.ErrorContainsf(err,  expectedErrorSubString, "error message %s", "formatted")
func ErrorContainsf(ctx proptest.RunContext, theError error, contains string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorContainsf", theError, contains, msg, args)
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIs(ctx proptest.RunContext, err error, target error, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorIs", err, target, msgAndArgs)
}

// ErrorIsf asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIsf(ctx proptest.RunContext, err error, target error, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("ErrorIsf", err, target, msg, args)
}

// Errorf asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.Errorf(err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedErrorf, err)
//	  }
func Errorf(ctx proptest.RunContext, err error, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Errorf", err, msg, args)
}

// Eventually asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	a.Eventually(func() bool { return true; }, time.Second, 10*time.Millisecond)
func Eventually(ctx proptest.RunContext, condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Eventually", condition, waitFor, tick, msgAndArgs)
}

// EventuallyWithT asserts that given condition will be met in waitFor time,
// periodically checking target function each tick. In contrast to Eventually,
// it supplies a CollectT to the condition function, so that the condition
// function can use the CollectT to call other assertions.
// The condition is considered "met" if no errors are raised in a tick.
// The supplied CollectT collects all errors from one tick (if there are any).
// If the condition is not met before waitFor, the collected errors of
// the last tick are copied to t.
//
//	externalValue := false
//	go func() {
//		time.Sleep(8*time.Second)
//		externalValue = true
//	}()
//	a.EventuallyWithT(func(c *assert.CollectT) {
//		// add assertions as needed; any assertion failure will fail the current tick
//		assert.True(c, externalValue, "expected 'externalValue' to be true")
//	}, 10*time.Second, 1*time.Second, "external state has not changed to 'true'; still false")
func EventuallyWithT(ctx proptest.RunContext, condition func(collect *assert.CollectT), waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EventuallyWithT", condition, waitFor, tick, msgAndArgs)
}

// EventuallyWithTf asserts that given condition will be met in waitFor time,
// periodically checking target function each tick. In contrast to Eventually,
// it supplies a CollectT to the condition function, so that the condition
// function can use the CollectT to call other assertions.
// The condition is considered "met" if no errors are raised in a tick.
// The supplied CollectT collects all errors from one tick (if there are any).
// If the condition is not met before waitFor, the collected errors of
// the last tick are copied to t.
//
//	externalValue := false
//	go func() {
//		time.Sleep(8*time.Second)
//		externalValue = true
//	}()
//	a.EventuallyWithTf(func(c *assert.CollectT, "error message %s", "formatted") {
//		// add assertions as needed; any assertion failure will fail the current tick
//		assert.True(c, externalValue, "expected 'externalValue' to be true")
//	}, 10*time.Second, 1*time.Second, "external state has not changed to 'true'; still false")
func EventuallyWithTf(ctx proptest.RunContext, condition func(collect *assert.CollectT), waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("EventuallyWithTf", condition, waitFor, tick, msg, args)
}

// Eventuallyf asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	a.Eventuallyf(func() bool { return true; }, time.Second, 10*time.Millisecond, "error message %s", "formatted")
func Eventuallyf(ctx proptest.RunContext, condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Eventuallyf", condition, waitFor, tick, msg, args)
}

// Exactly asserts that two objects are equal in value and type.
//
//	a.Exactly(int32(123), int64(123))
func Exactly(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Exactly", expected, actual, msgAndArgs)
}

// Exactlyf asserts that two objects are equal in value and type.
//
//	a.Exactlyf(int32(123), int64(123), "error message %s", "formatted")
func Exactlyf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Exactlyf", expected, actual, msg, args)
}

// Fail reports a failure through
func Fail(ctx proptest.RunContext, failureMessage string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Fail", failureMessage, msgAndArgs)
}

// FailNow fails test
func FailNow(ctx proptest.RunContext, failureMessage string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("FailNow", failureMessage, msgAndArgs)
}

// FailNowf fails test
func FailNowf(ctx proptest.RunContext, failureMessage string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("FailNowf", failureMessage, msg, args)
}

// Failf reports a failure through
func Failf(ctx proptest.RunContext, failureMessage string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Failf", failureMessage, msg, args)
}

// False asserts that the specified value is false.
//
//	a.False(myBool)
func False(ctx proptest.RunContext, value bool, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("False", value, msgAndArgs)
}

// Falsef asserts that the specified value is false.
//
//	a.Falsef(myBool, "error message %s", "formatted")
func Falsef(ctx proptest.RunContext, value bool, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Falsef", value, msg, args)
}

// FileExists checks whether a file exists in the given path. It also fails if
// the path points to a directory or there is an error when trying to check the file.
func FileExists(ctx proptest.RunContext, path string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("FileExists", path, msgAndArgs)
}

// FileExistsf checks whether a file exists in the given path. It also fails if
// the path points to a directory or there is an error when trying to check the file.
func FileExistsf(ctx proptest.RunContext, path string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("FileExistsf", path, msg, args)
}

// Greater asserts that the first element is greater than the second
//
//	a.Greater(2, 1)
//	a.Greater(float64(2), float64(1))
//	a.Greater("b", "a")
func Greater(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Greater", e1, e2, msgAndArgs)
}

// GreaterOrEqual asserts that the first element is greater than or equal to the second
//
//	a.GreaterOrEqual(2, 1)
//	a.GreaterOrEqual(2, 2)
//	a.GreaterOrEqual("b", "a")
//	a.GreaterOrEqual("b", "b")
func GreaterOrEqual(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("GreaterOrEqual", e1, e2, msgAndArgs)
}

// GreaterOrEqualf asserts that the first element is greater than or equal to the second
//
//	a.GreaterOrEqualf(2, 1, "error message %s", "formatted")
//	a.GreaterOrEqualf(2, 2, "error message %s", "formatted")
//	a.GreaterOrEqualf("b", "a", "error message %s", "formatted")
//	a.GreaterOrEqualf("b", "b", "error message %s", "formatted")
func GreaterOrEqualf(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("GreaterOrEqualf", e1, e2, msg, args)
}

// Greaterf asserts that the first element is greater than the second
//
//	a.Greaterf(2, 1, "error message %s", "formatted")
//	a.Greaterf(float64(2), float64(1), "error message %s", "formatted")
//	a.Greaterf("b", "a", "error message %s", "formatted")
func Greaterf(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Greaterf", e1, e2, msg, args)
}

// HTTPBodyContains asserts that a specified handler returns a
// body that contains a string.
//
//	a.HTTPBodyContains(myHandler, "GET", "www.google.com", nil, "I'm Feeling Lucky")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPBodyContains(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPBodyContains", handler, method, url, values, str, msgAndArgs)
}

// HTTPBodyContainsf asserts that a specified handler returns a
// body that contains a string.
//
//	a.HTTPBodyContainsf(myHandler, "GET", "www.google.com", nil, "I'm Feeling Lucky", "error message %s", "formatted")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPBodyContainsf(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPBodyContainsf", handler, method, url, values, str, msg, args)
}

// HTTPBodyNotContains asserts that a specified handler returns a
// body that does not contain a string.
//
//	a.HTTPBodyNotContains(myHandler, "GET", "www.google.com", nil, "I'm Feeling Lucky")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPBodyNotContains(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPBodyNotContains", handler, method, url, values, str, msgAndArgs)
}

// HTTPBodyNotContainsf asserts that a specified handler returns a
// body that does not contain a string.
//
//	a.HTTPBodyNotContainsf(myHandler, "GET", "www.google.com", nil, "I'm Feeling Lucky", "error message %s", "formatted")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPBodyNotContainsf(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPBodyNotContainsf", handler, method, url, values, str, msg, args)
}

// HTTPError asserts that a specified handler returns an error status code.
//
//	a.HTTPError(myHandler, "POST", "/a/b/c", url.Values{"a": []string{"b", "c"}}
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPError(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPError", handler, method, url, values, msgAndArgs)
}

// HTTPErrorf asserts that a specified handler returns an error status code.
//
//	a.HTTPErrorf(myHandler, "POST", "/a/b/c", url.Values{"a": []string{"b", "c"}}
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPErrorf(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPErrorf", handler, method, url, values, msg, args)
}

// HTTPRedirect asserts that a specified handler returns a redirect status code.
//
//	a.HTTPRedirect(myHandler, "GET", "/a/b/c", url.Values{"a": []string{"b", "c"}}
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPRedirect(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPRedirect", handler, method, url, values, msgAndArgs)
}

// HTTPRedirectf asserts that a specified handler returns a redirect status code.
//
//	a.HTTPRedirectf(myHandler, "GET", "/a/b/c", url.Values{"a": []string{"b", "c"}}
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPRedirectf(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPRedirectf", handler, method, url, values, msg, args)
}

// HTTPStatusCode asserts that a specified handler returns a specified status code.
//
//	a.HTTPStatusCode(myHandler, "GET", "/notImplemented", nil, 501)
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPStatusCode(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, statuscode int, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPStatusCode", handler, method, url, values, statuscode, msgAndArgs)
}

// HTTPStatusCodef asserts that a specified handler returns a specified status code.
//
//	a.HTTPStatusCodef(myHandler, "GET", "/notImplemented", nil, 501, "error message %s", "formatted")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPStatusCodef(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, statuscode int, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPStatusCodef", handler, method, url, values, statuscode, msg, args)
}

// HTTPSuccess asserts that a specified handler returns a success status code.
//
//	a.HTTPSuccess(myHandler, "POST", "http://www.google.com", nil)
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPSuccess(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPSuccess", handler, method, url, values, msgAndArgs)
}

// HTTPSuccessf asserts that a specified handler returns a success status code.
//
//	a.HTTPSuccessf(myHandler, "POST", "http://www.google.com", nil, "error message %s", "formatted")
//
// Returns whether the assertion was successful (true) or not (false).
func HTTPSuccessf(ctx proptest.RunContext, handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("HTTPSuccessf", handler, method, url, values, msg, args)
}

// Implements asserts that an object is implemented by the specified interface.
//
//	a.Implements((*MyInterface)(nil), new(MyObject))
func Implements(ctx proptest.RunContext, interfaceObject interface{}, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Implements", interfaceObject, object, msgAndArgs)
}

// Implementsf asserts that an object is implemented by the specified interface.
//
//	a.Implementsf((*MyInterface)(nil), new(MyObject), "error message %s", "formatted")
func Implementsf(ctx proptest.RunContext, interfaceObject interface{}, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Implementsf", interfaceObject, object, msg, args)
}

// InDelta asserts that the two numerals are within delta of each other.
//
//	a.InDelta(math.Pi, 22/7.0, 0.01)
func InDelta(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDelta", expected, actual, delta, msgAndArgs)
}

// InDeltaMapValues is the same as InDelta, but it compares all values between two maps. Both maps must have exactly the same keys.
func InDeltaMapValues(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDeltaMapValues", expected, actual, delta, msgAndArgs)
}

// InDeltaMapValuesf is the same as InDelta, but it compares all values between two maps. Both maps must have exactly the same keys.
func InDeltaMapValuesf(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDeltaMapValuesf", expected, actual, delta, msg, args)
}

// InDeltaSlice is the same as InDelta, except it compares two slices.
func InDeltaSlice(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDeltaSlice", expected, actual, delta, msgAndArgs)
}

// InDeltaSlicef is the same as InDelta, except it compares two slices.
func InDeltaSlicef(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDeltaSlicef", expected, actual, delta, msg, args)
}

// InDeltaf asserts that the two numerals are within delta of each other.
//
//	a.InDeltaf(math.Pi, 22/7.0, 0.01, "error message %s", "formatted")
func InDeltaf(ctx proptest.RunContext, expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InDeltaf", expected, actual, delta, msg, args)
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon
func InEpsilon(ctx proptest.RunContext, expected interface{}, actual interface{}, epsilon float64, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InEpsilon", expected, actual, epsilon, msgAndArgs)
}

// InEpsilonSlice is the same as InEpsilon, except it compares each value from two slices.
func InEpsilonSlice(ctx proptest.RunContext, expected interface{}, actual interface{}, epsilon float64, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InEpsilonSlice", expected, actual, epsilon, msgAndArgs)
}

// InEpsilonSlicef is the same as InEpsilon, except it compares each value from two slices.
func InEpsilonSlicef(ctx proptest.RunContext, expected interface{}, actual interface{}, epsilon float64, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InEpsilonSlicef", expected, actual, epsilon, msg, args)
}

// InEpsilonf asserts that expected and actual have a relative error less than epsilon
func InEpsilonf(ctx proptest.RunContext, expected interface{}, actual interface{}, epsilon float64, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("InEpsilonf", expected, actual, epsilon, msg, args)
}

// IsDecreasing asserts that the collection is decreasing
//
//	a.IsDecreasing([]int{2, 1, 0})
//	a.IsDecreasing([]float{2, 1})
//	a.IsDecreasing([]string{"b", "a"})
func IsDecreasing(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsDecreasing", object, msgAndArgs)
}

// IsDecreasingf asserts that the collection is decreasing
//
//	a.IsDecreasingf([]int{2, 1, 0}, "error message %s", "formatted")
//	a.IsDecreasingf([]float{2, 1}, "error message %s", "formatted")
//	a.IsDecreasingf([]string{"b", "a"}, "error message %s", "formatted")
func IsDecreasingf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsDecreasingf", object, msg, args)
}

// IsIncreasing asserts that the collection is increasing
//
//	a.IsIncreasing([]int{1, 2, 3})
//	a.IsIncreasing([]float{1, 2})
//	a.IsIncreasing([]string{"a", "b"})
func IsIncreasing(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsIncreasing", object, msgAndArgs)
}

// IsIncreasingf asserts that the collection is increasing
//
//	a.IsIncreasingf([]int{1, 2, 3}, "error message %s", "formatted")
//	a.IsIncreasingf([]float{1, 2}, "error message %s", "formatted")
//	a.IsIncreasingf([]string{"a", "b"}, "error message %s", "formatted")
func IsIncreasingf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsIncreasingf", object, msg, args)
}

// IsNonDecreasing asserts that the collection is not decreasing
//
//	a.IsNonDecreasing([]int{1, 1, 2})
//	a.IsNonDecreasing([]float{1, 2})
//	a.IsNonDecreasing([]string{"a", "b"})
func IsNonDecreasing(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsNonDecreasing", object, msgAndArgs)
}

// IsNonDecreasingf asserts that the collection is not decreasing
//
//	a.IsNonDecreasingf([]int{1, 1, 2}, "error message %s", "formatted")
//	a.IsNonDecreasingf([]float{1, 2}, "error message %s", "formatted")
//	a.IsNonDecreasingf([]string{"a", "b"}, "error message %s", "formatted")
func IsNonDecreasingf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsNonDecreasingf", object, msg, args)
}

// IsNonIncreasing asserts that the collection is not increasing
//
//	a.IsNonIncreasing([]int{2, 1, 1})
//	a.IsNonIncreasing([]float{2, 1})
//	a.IsNonIncreasing([]string{"b", "a"})
func IsNonIncreasing(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsNonIncreasing", object, msgAndArgs)
}

// IsNonIncreasingf asserts that the collection is not increasing
//
//	a.IsNonIncreasingf([]int{2, 1, 1}, "error message %s", "formatted")
//	a.IsNonIncreasingf([]float{2, 1}, "error message %s", "formatted")
//	a.IsNonIncreasingf([]string{"b", "a"}, "error message %s", "formatted")
func IsNonIncreasingf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsNonIncreasingf", object, msg, args)
}

// IsType asserts that the specified objects are of the same type.
func IsType(ctx proptest.RunContext, expectedType interface{}, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsType", expectedType, object, msgAndArgs)
}

// IsTypef asserts that the specified objects are of the same type.
func IsTypef(ctx proptest.RunContext, expectedType interface{}, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("IsTypef", expectedType, object, msg, args)
}

// JSONEq asserts that two JSON strings are equivalent.
//
//	a.JSONEq(`{"hello": "world", "foo": "bar"}`, `{"foo": "bar", "hello": "world"}`)
func JSONEq(ctx proptest.RunContext, expected string, actual string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("JSONEq", expected, actual, msgAndArgs)
}

// JSONEqf asserts that two JSON strings are equivalent.
//
//	a.JSONEqf(`{"hello": "world", "foo": "bar"}`, `{"foo": "bar", "hello": "world"}`, "error message %s", "formatted")
func JSONEqf(ctx proptest.RunContext, expected string, actual string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("JSONEqf", expected, actual, msg, args)
}

// Len asserts that the specified object has specific length.
// Len also fails if the object has a type that len() not accept.
//
//	a.Len(mySlice, 3)
func Len(ctx proptest.RunContext, object interface{}, length int, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Len", object, length, msgAndArgs)
}

// Lenf asserts that the specified object has specific length.
// Lenf also fails if the object has a type that len() not accept.
//
//	a.Lenf(mySlice, 3, "error message %s", "formatted")
func Lenf(ctx proptest.RunContext, object interface{}, length int, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Lenf", object, length, msg, args)
}

// Less asserts that the first element is less than the second
//
//	a.Less(1, 2)
//	a.Less(float64(1), float64(2))
//	a.Less("a", "b")
func Less(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Less", e1, e2, msgAndArgs)
}

// LessOrEqual asserts that the first element is less than or equal to the second
//
//	a.LessOrEqual(1, 2)
//	a.LessOrEqual(2, 2)
//	a.LessOrEqual("a", "b")
//	a.LessOrEqual("b", "b")
func LessOrEqual(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("LessOrEqual", e1, e2, msgAndArgs)
}

// LessOrEqualf asserts that the first element is less than or equal to the second
//
//	a.LessOrEqualf(1, 2, "error message %s", "formatted")
//	a.LessOrEqualf(2, 2, "error message %s", "formatted")
//	a.LessOrEqualf("a", "b", "error message %s", "formatted")
//	a.LessOrEqualf("b", "b", "error message %s", "formatted")
func LessOrEqualf(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("LessOrEqualf", e1, e2, msg, args)
}

// Lessf asserts that the first element is less than the second
//
//	a.Lessf(1, 2, "error message %s", "formatted")
//	a.Lessf(float64(1), float64(2), "error message %s", "formatted")
//	a.Lessf("a", "b", "error message %s", "formatted")
func Lessf(ctx proptest.RunContext, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Lessf", e1, e2, msg, args)
}

// Negative asserts that the specified element is negative
//
//	a.Negative(-1)
//	a.Negative(-1.23)
func Negative(ctx proptest.RunContext, e interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Negative", e, msgAndArgs)
}

// Negativef asserts that the specified element is negative
//
//	a.Negativef(-1, "error message %s", "formatted")
//	a.Negativef(-1.23, "error message %s", "formatted")
func Negativef(ctx proptest.RunContext, e interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Negativef", e, msg, args)
}

// Never asserts that the given condition doesn't satisfy in waitFor time,
// periodically checking the target function each tick.
//
//	a.Never(func() bool { return false; }, time.Second, 10*time.Millisecond)
func Never(ctx proptest.RunContext, condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Never", condition, waitFor, tick, msgAndArgs)
}

// Neverf asserts that the given condition doesn't satisfy in waitFor time,
// periodically checking the target function each tick.
//
//	a.Neverf(func() bool { return false; }, time.Second, 10*time.Millisecond, "error message %s", "formatted")
func Neverf(ctx proptest.RunContext, condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Neverf", condition, waitFor, tick, msg, args)
}

// Nil asserts that the specified object is nil.
//
//	a.Nil(err)
func Nil(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Nil", object, msgAndArgs)
}

// Nilf asserts that the specified object is nil.
//
//	a.Nilf(err, "error message %s", "formatted")
func Nilf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Nilf", object, msg, args)
}

// NoDirExists checks whether a directory does not exist in the given path.
// It fails if the path points to an existing _directory_ only.
func NoDirExists(ctx proptest.RunContext, path string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoDirExists", path, msgAndArgs)
}

// NoDirExistsf checks whether a directory does not exist in the given path.
// It fails if the path points to an existing _directory_ only.
func NoDirExistsf(ctx proptest.RunContext, path string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoDirExistsf", path, msg, args)
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.NoError(err) {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoError(ctx proptest.RunContext, err error, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoError", err, msgAndArgs)
}

// NoErrorf asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.NoErrorf(err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoErrorf(ctx proptest.RunContext, err error, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoErrorf", err, msg, args)
}

// NoFileExists checks whether a file does not exist in a given path. It fails
// if the path points to an existing _file_ only.
func NoFileExists(ctx proptest.RunContext, path string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoFileExists", path, msgAndArgs)
}

// NoFileExistsf checks whether a file does not exist in a given path. It fails
// if the path points to an existing _file_ only.
func NoFileExistsf(ctx proptest.RunContext, path string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NoFileExistsf", path, msg, args)
}

// NotContains asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	a.NotContains("Hello World", "Earth")
//	a.NotContains(["Hello", "World"], "Earth")
//	a.NotContains({"Hello": "World"}, "Earth")
func NotContains(ctx proptest.RunContext, s interface{}, contains interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotContains", s, contains, msgAndArgs)
}

// NotContainsf asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	a.NotContainsf("Hello World", "Earth", "error message %s", "formatted")
//	a.NotContainsf(["Hello", "World"], "Earth", "error message %s", "formatted")
//	a.NotContainsf({"Hello": "World"}, "Earth", "error message %s", "formatted")
func NotContainsf(ctx proptest.RunContext, s interface{}, contains interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotContainsf", s, contains, msg, args)
}

// NotElementsMatch asserts that the specified listA(array, slice...) is NOT equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should not match.
// This is an inverse of ElementsMatch.
//
// a.NotElementsMatch([1, 1, 2, 3], [1, 1, 2, 3]) -> false
//
// a.NotElementsMatch([1, 1, 2, 3], [1, 2, 3]) -> true
//
// a.NotElementsMatch([1, 2, 3], [1, 2, 4]) -> true
func NotElementsMatch(ctx proptest.RunContext, listA interface{}, listB interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotElementsMatch", listA, listB, msgAndArgs)
}

// NotElementsMatchf asserts that the specified listA(array, slice...) is NOT equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should not match.
// This is an inverse of ElementsMatch.
//
// a.NotElementsMatchf([1, 1, 2, 3], [1, 1, 2, 3], "error message %s", "formatted") -> false
//
// a.NotElementsMatchf([1, 1, 2, 3], [1, 2, 3], "error message %s", "formatted") -> true
//
// a.NotElementsMatchf([1, 2, 3], [1, 2, 4], "error message %s", "formatted") -> true
func NotElementsMatchf(ctx proptest.RunContext, listA interface{}, listB interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotElementsMatchf", listA, listB, msg, args)
}

// NotEmpty asserts that the specified object is NOT empty.  I.e. not nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	if a.NotEmpty(obj) {
//	  assert.Equal(t, "two", obj[1])
//	}
func NotEmpty(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEmpty", object, msgAndArgs)
}

// NotEmptyf asserts that the specified object is NOT empty.  I.e. not nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	if a.NotEmptyf(obj, "error message %s", "formatted") {
//	  assert.Equal(t, "two", obj[1])
//	}
func NotEmptyf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEmptyf", object, msg, args)
}

// NotEqual asserts that the specified values are NOT equal.
//
//	a.NotEqual(obj1, obj2)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqual(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEqual", expected, actual, msgAndArgs)
}

// NotEqualValues asserts that two objects are not equal even when converted to the same type
//
//	a.NotEqualValues(obj1, obj2)
func NotEqualValues(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEqualValues", expected, actual, msgAndArgs)
}

// NotEqualValuesf asserts that two objects are not equal even when converted to the same type
//
//	a.NotEqualValuesf(obj1, obj2, "error message %s", "formatted")
func NotEqualValuesf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEqualValuesf", expected, actual, msg, args)
}

// NotEqualf asserts that the specified values are NOT equal.
//
//	a.NotEqualf(obj1, obj2, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqualf(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotEqualf", expected, actual, msg, args)
}

// NotErrorAs asserts that none of the errors in err's chain matches target,
// but if so, sets target to that error value.
func NotErrorAs(ctx proptest.RunContext, err error, target interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotErrorAs", err, target, msgAndArgs)
}

// NotErrorAsf asserts that none of the errors in err's chain matches target,
// but if so, sets target to that error value.
func NotErrorAsf(ctx proptest.RunContext, err error, target interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotErrorAsf", err, target, msg, args)
}

// NotErrorIs asserts that none of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func NotErrorIs(ctx proptest.RunContext, err error, target error, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotErrorIs", err, target, msgAndArgs)
}

// NotErrorIsf asserts that none of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func NotErrorIsf(ctx proptest.RunContext, err error, target error, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotErrorIsf", err, target, msg, args)
}

// NotImplements asserts that an object does not implement the specified interface.
//
//	a.NotImplements((*MyInterface)(nil), new(MyObject))
func NotImplements(ctx proptest.RunContext, interfaceObject interface{}, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotImplements", interfaceObject, object, msgAndArgs)
}

// NotImplementsf asserts that an object does not implement the specified interface.
//
//	a.NotImplementsf((*MyInterface)(nil), new(MyObject), "error message %s", "formatted")
func NotImplementsf(ctx proptest.RunContext, interfaceObject interface{}, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotImplementsf", interfaceObject, object, msg, args)
}

// NotNil asserts that the specified object is not nil.
//
//	a.NotNil(err)
func NotNil(ctx proptest.RunContext, object interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotNil", object, msgAndArgs)
}

// NotNilf asserts that the specified object is not nil.
//
//	a.NotNilf(err, "error message %s", "formatted")
func NotNilf(ctx proptest.RunContext, object interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotNilf", object, msg, args)
}

// NotPanics asserts that the code inside the specified PanicTestFunc does NOT panic.
//
//	a.NotPanics(func(){ RemainCalm() })
func NotPanics(ctx proptest.RunContext, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotPanics", f, msgAndArgs)
}

// NotPanicsf asserts that the code inside the specified PanicTestFunc does NOT panic.
//
//	a.NotPanicsf(func(){ RemainCalm() }, "error message %s", "formatted")
func NotPanicsf(ctx proptest.RunContext, f assert.PanicTestFunc, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotPanicsf", f, msg, args)
}

// NotRegexp asserts that a specified regexp does not match a string.
//
//	a.NotRegexp(regexp.MustCompile("starts"), "it's starting")
//	a.NotRegexp("^start", "it's not starting")
func NotRegexp(ctx proptest.RunContext, rx interface{}, str interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotRegexp", rx, str, msgAndArgs)
}

// NotRegexpf asserts that a specified regexp does not match a string.
//
//	a.NotRegexpf(regexp.MustCompile("starts"), "it's starting", "error message %s", "formatted")
//	a.NotRegexpf("^start", "it's not starting", "error message %s", "formatted")
func NotRegexpf(ctx proptest.RunContext, rx interface{}, str interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotRegexpf", rx, str, msg, args)
}

// NotSame asserts that two pointers do not reference the same object.
//
//	a.NotSame(ptr1, ptr2)
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func NotSame(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotSame", expected, actual, msgAndArgs)
}

// NotSamef asserts that two pointers do not reference the same object.
//
//	a.NotSamef(ptr1, ptr2, "error message %s", "formatted")
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func NotSamef(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotSamef", expected, actual, msg, args)
}

// NotSubset asserts that the specified list(array, slice...) or map does NOT
// contain all elements given in the specified subset list(array, slice...) or
// map.
//
//	a.NotSubset([1, 3, 4], [1, 2])
//	a.NotSubset({"x": 1, "y": 2}, {"z": 3})
func NotSubset(ctx proptest.RunContext, list interface{}, subset interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotSubset", list, subset, msgAndArgs)
}

// NotSubsetf asserts that the specified list(array, slice...) or map does NOT
// contain all elements given in the specified subset list(array, slice...) or
// map.
//
//	a.NotSubsetf([1, 3, 4], [1, 2], "error message %s", "formatted")
//	a.NotSubsetf({"x": 1, "y": 2}, {"z": 3}, "error message %s", "formatted")
func NotSubsetf(ctx proptest.RunContext, list interface{}, subset interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotSubsetf", list, subset, msg, args)
}

// NotZero asserts that i is not the zero value for its type.
func NotZero(ctx proptest.RunContext, i interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotZero", i, msgAndArgs)
}

// NotZerof asserts that i is not the zero value for its type.
func NotZerof(ctx proptest.RunContext, i interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("NotZerof", i, msg, args)
}

// Panics asserts that the code inside the specified PanicTestFunc panics.
//
//	a.Panics(func(){ GoCrazy() })
func Panics(ctx proptest.RunContext, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Panics", f, msgAndArgs)
}

// PanicsWithError asserts that the code inside the specified PanicTestFunc
// panics, and that the recovered panic value is an error that satisfies the
// EqualError comparison.
//
//	a.PanicsWithError("crazy error", func(){ GoCrazy() })
func PanicsWithError(ctx proptest.RunContext, errString string, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("PanicsWithError", errString, f, msgAndArgs)
}

// PanicsWithErrorf asserts that the code inside the specified PanicTestFunc
// panics, and that the recovered panic value is an error that satisfies the
// EqualError comparison.
//
//	a.PanicsWithErrorf("crazy error", func(){ GoCrazy() }, "error message %s", "formatted")
func PanicsWithErrorf(ctx proptest.RunContext, errString string, f assert.PanicTestFunc, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("PanicsWithErrorf", errString, f, msg, args)
}

// PanicsWithValue asserts that the code inside the specified PanicTestFunc panics, and that
// the recovered panic value equals the expected panic value.
//
//	a.PanicsWithValue("crazy error", func(){ GoCrazy() })
func PanicsWithValue(ctx proptest.RunContext, expected interface{}, f assert.PanicTestFunc, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("PanicsWithValue", expected, f, msgAndArgs)
}

// PanicsWithValuef asserts that the code inside the specified PanicTestFunc panics, and that
// the recovered panic value equals the expected panic value.
//
//	a.PanicsWithValuef("crazy error", func(){ GoCrazy() }, "error message %s", "formatted")
func PanicsWithValuef(ctx proptest.RunContext, expected interface{}, f assert.PanicTestFunc, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("PanicsWithValuef", expected, f, msg, args)
}

// Panicsf asserts that the code inside the specified PanicTestFunc panics.
//
//	a.Panicsf(func(){ GoCrazy() }, "error message %s", "formatted")
func Panicsf(ctx proptest.RunContext, f assert.PanicTestFunc, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Panicsf", f, msg, args)
}

// Positive asserts that the specified element is positive
//
//	a.Positive(1)
//	a.Positive(1.23)
func Positive(ctx proptest.RunContext, e interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Positive", e, msgAndArgs)
}

// Positivef asserts that the specified element is positive
//
//	a.Positivef(1, "error message %s", "formatted")
//	a.Positivef(1.23, "error message %s", "formatted")
func Positivef(ctx proptest.RunContext, e interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Positivef", e, msg, args)
}

// Regexp asserts that a specified regexp matches a string.
//
//	a.Regexp(regexp.MustCompile("start"), "it's starting")
//	a.Regexp("start...$", "it's not starting")
func Regexp(ctx proptest.RunContext, rx interface{}, str interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Regexp", rx, str, msgAndArgs)
}

// Regexpf asserts that a specified regexp matches a string.
//
//	a.Regexpf(regexp.MustCompile("start"), "it's starting", "error message %s", "formatted")
//	a.Regexpf("start...$", "it's not starting", "error message %s", "formatted")
func Regexpf(ctx proptest.RunContext, rx interface{}, str interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Regexpf", rx, str, msg, args)
}

// Same asserts that two pointers reference the same object.
//
//	a.Same(ptr1, ptr2)
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func Same(ctx proptest.RunContext, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Same", expected, actual, msgAndArgs)
}

// Samef asserts that two pointers reference the same object.
//
//	a.Samef(ptr1, ptr2, "error message %s", "formatted")
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func Samef(ctx proptest.RunContext, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Samef", expected, actual, msg, args)
}

// Subset asserts that the specified list(array, slice...) or map contains all
// elements given in the specified subset list(array, slice...) or map.
//
//	a.Subset([1, 2, 3], [1, 2])
//	a.Subset({"x": 1, "y": 2}, {"x": 1})
func Subset(ctx proptest.RunContext, list interface{}, subset interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Subset", list, subset, msgAndArgs)
}

// Subsetf asserts that the specified list(array, slice...) or map contains all
// elements given in the specified subset list(array, slice...) or map.
//
//	a.Subsetf([1, 2, 3], [1, 2], "error message %s", "formatted")
//	a.Subsetf({"x": 1, "y": 2}, {"x": 1}, "error message %s", "formatted")
func Subsetf(ctx proptest.RunContext, list interface{}, subset interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Subsetf", list, subset, msg, args)
}

// True asserts that the specified value is true.
//
//	a.True(myBool)
func True(ctx proptest.RunContext, value bool, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("True", value, msgAndArgs)
}

// Truef asserts that the specified value is true.
//
//	a.Truef(myBool, "error message %s", "formatted")
func Truef(ctx proptest.RunContext, value bool, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Truef", value, msg, args)
}

// WithinDuration asserts that the two times are within duration delta of each other.
//
//	a.WithinDuration(time.Now(), time.Now(), 10*time.Second)
func WithinDuration(ctx proptest.RunContext, expected time.Time, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("WithinDuration", expected, actual, delta, msgAndArgs)
}

// WithinDurationf asserts that the two times are within duration delta of each other.
//
//	a.WithinDurationf(time.Now(), time.Now(), 10*time.Second, "error message %s", "formatted")
func WithinDurationf(ctx proptest.RunContext, expected time.Time, actual time.Time, delta time.Duration, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("WithinDurationf", expected, actual, delta, msg, args)
}

// WithinRange asserts that a time is within a time range (inclusive).
//
//	a.WithinRange(time.Now(), time.Now().Add(-time.Second), time.Now().Add(time.Second))
func WithinRange(ctx proptest.RunContext, actual time.Time, start time.Time, end time.Time, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("WithinRange", actual, start, end, msgAndArgs)
}

// WithinRangef asserts that a time is within a time range (inclusive).
//
//	a.WithinRangef(time.Now(), time.Now().Add(-time.Second), time.Now().Add(time.Second), "error message %s", "formatted")
func WithinRangef(ctx proptest.RunContext, actual time.Time, start time.Time, end time.Time, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("WithinRangef", actual, start, end, msg, args)
}

// YAMLEq asserts that two YAML strings are equivalent.
func YAMLEq(ctx proptest.RunContext, expected string, actual string, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("YAMLEq", expected, actual, msgAndArgs)
}

// YAMLEqf asserts that two YAML strings are equivalent.
func YAMLEqf(ctx proptest.RunContext, expected string, actual string, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("YAMLEqf", expected, actual, msg, args)
}

// Zero asserts that i is the zero value for its type.
func Zero(ctx proptest.RunContext, i interface{}, msgAndArgs ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Zero", i, msgAndArgs)
}

// Zerof asserts that i is the zero value for its type.
func Zerof(ctx proptest.RunContext, i interface{}, msg string, args ...interface{}) {
	r := ctx.Run()
	r.T().Helper()
	r.Assert("Zerof", i, msg, args)
}
