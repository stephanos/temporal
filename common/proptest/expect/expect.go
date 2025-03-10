// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
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

package expect

import (
	"reflect"
	"time"

	. "go.temporal.io/server/common/proptest/internal"
)

func require[T any](name string, args ...any) Rule {
	return &requireRule{
		name:     name,
		args:     args,
		varTypes: []VarType{reflect.TypeFor[T]()},
	}
}

func require2[T1, T2 any](name string, args ...any) Rule {
	return &requireRule{
		name:     name,
		args:     args,
		varTypes: []VarType{reflect.TypeFor[T1](), reflect.TypeFor[T2]()},
	}
}

// Contains asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	a.Contains("Hello World", "World")
//	a.Contains(["Hello", "World"], "World")
//	a.Contains({"Hello": "World"}, "Hello")
func Contains[T any](contains any, msgAndArgs ...any) Rule {
	return require[T]("Contains", varRef, contains, msgAndArgs)
}

// Containsf asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	a.Containsf("Hello World", "World", "error message %s", "formatted")
//	a.Containsf(["Hello", "World"], "World", "error message %s", "formatted")
//	a.Containsf({"Hello": "World"}, "Hello", "error message %s", "formatted")
func Containsf[T any](contains any, msg string, args ...any) Rule {
	return require[T]("Containsf", varRef, contains, msg, args)
}

// ElementsMatch asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// a.ElementsMatch([1, 3, 2, 3], [1, 3, 3, 2])
//func ElementsMatch[T any](listA any, listB any, msgAndArgs ...any) Rule {
//	return require[T]("ElementsMatch", listA, listB, msgAndArgs)
//}

// ElementsMatchf asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// a.ElementsMatchf([1, 3, 2, 3], [1, 3, 3, 2], "error message %s", "formatted")
//func ElementsMatchf[T any](listA any, listB any, msg string, args ...any) Rule {
//	return require[T]("ElementsMatchf", listA, listB, msg, args)
//}

// Empty asserts that the specified object is empty.  I.e. nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	a.Empty(obj)
func Empty[T any](msgAndArgs ...any) Rule {
	return require[T]("Empty", varRef, msgAndArgs)
}

// Emptyf asserts that the specified object is empty.  I.e. nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	a.Emptyf(obj, "error message %s", "formatted")
func Emptyf[T any](msg string, args ...any) Rule {
	return require[T]("Emptyf", varRef, msg, args)
}

// Equal asserts that two variables are equal.
//
//	a.Equal[Var1, Var2]()
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal[T1 any, T2 any](msgAndArgs ...any) Rule {
	return require2[T1, T2]("Equal", varRef, varRef, msgAndArgs)
}

// Equals asserts that two objects are equal.
//
//	a.Equals(123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equals[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("Equal", expected, varRef, msgAndArgs)
}

// EqualError asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	a.EqualError(err,  expectedErrorString)
func EqualError[T error](errString string, msgAndArgs ...any) Rule {
	return require[T]("EqualError", varRef, errString, msgAndArgs)
}

// EqualErrorf asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	a.EqualErrorf(err,  expectedErrorString, "error message %s", "formatted")
func EqualErrorf[T error](errString string, msg string, args ...any) Rule {
	return require[T]("EqualErrorf", varRef, errString, msg, args)
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
func EqualExportedValues[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("EqualExportedValues", expected, varRef, msgAndArgs)
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
func EqualExportedValuesf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("EqualExportedValuesf", expected, varRef, msg, args)
}

// EqualValues asserts that two objects are equal or convertible to the larger
// type and equal.
//
//	a.EqualValues(uint32(123), int32(123))
func EqualValues[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("EqualValues", expected, varRef, msgAndArgs)
}

// EqualValuesf asserts that two objects are equal or convertible to the larger
// type and equal.
//
//	a.EqualValuesf(uint32(123), int32(123), "error message %s", "formatted")
func EqualValuesf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("EqualValuesf", expected, varRef, msg, args)
}

// Equalf asserts that two objects are equal.
//
//	a.Equalf(123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("Equalf", expected, varRef, msg, args)
}

// Error asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.Error(err) Rule {
//		   assert.Equal(t, expectedError, err)
//	  }
func Error[T error](msgAndArgs ...any) Rule {
	return require[T]("Error", varRef, msgAndArgs)
}

// ErrorAs asserts that at least one of the errors in err's chain matches target, and if so, sets target to that error value.
// This is a wrapper for errors.As.
func ErrorAs[T error](target any, msgAndArgs ...any) Rule {
	return require[T]("ErrorAs", varRef, target, msgAndArgs)
}

// ErrorAsf asserts that at least one of the errors in err's chain matches target, and if so, sets target to that error value.
// This is a wrapper for errors.As.
func ErrorAsf[T error](target any, msg string, args ...any) Rule {
	return require[T]("ErrorAsf", varRef, target, msg, args)
}

// ErrorContains asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	a.ErrorContains(err,  expectedErrorSubString)
func ErrorContains[T error](contains string, msgAndArgs ...any) Rule {
	return require[T]("ErrorContains", varRef, contains, msgAndArgs)
}

// ErrorContainsf asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	a.ErrorContainsf(err,  expectedErrorSubString, "error message %s", "formatted")
func ErrorContainsf[T error](contains string, msg string, args ...any) Rule {
	return require[T]("ErrorContainsf", varRef, contains, msg, args)
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIs[T error](target error, msgAndArgs ...any) Rule {
	return require[T]("ErrorIs", varRef, target, msgAndArgs)
}

// ErrorIsf asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIsf[T error](target error, msg string, args ...any) Rule {
	return require[T]("ErrorIsf", varRef, target, msg, args)
}

// Errorf asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.Errorf(err, "error message %s", "formatted") Rule {
//		   assert.Equal(t, expectedErrorf, err)
//	  }
func Errorf[T error](msg string, args ...any) Rule {
	return require[T]("Errorf", varRef, msg, args)
}

// Exactly asserts that two objects are equal in value and type.
//
//	a.Exactly(int32(123), int64(123))
func Exactly[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("Exactly", expected, varRef, msgAndArgs)
}

// Exactlyf asserts that two objects are equal in value and type.
//
//	a.Exactlyf(int32(123), int64(123), "error message %s", "formatted")
func Exactlyf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("Exactlyf", expected, varRef, msg, args)
}

// False asserts that the specified value is false.
//
//	a.False(myBool)
func False[T ~bool](msgAndArgs ...any) Rule {
	return require[T]("False", varRef, msgAndArgs)
}

// Falsef asserts that the specified value is false.
//
//	a.Falsef(myBool, "error message %s", "formatted")
func Falsef[T ~bool](msg string, args ...any) Rule {
	return require[T]("Falsef", varRef, msg, args)
}

// Greater asserts that the first element is greater than the second
//
//	a.Greater(2, 1)
//	a.Greater(float64(2), float64(1))
//	a.Greater("b", "a")
func Greater[T any](e2 any, msgAndArgs ...any) Rule {
	return require[T]("Greater", varRef, e2, msgAndArgs)
}

// GreaterOrEqual asserts that the first element is greater than or equal to the second
//
//	a.GreaterOrEqual(2, 1)
//	a.GreaterOrEqual(2, 2)
//	a.GreaterOrEqual("b", "a")
//	a.GreaterOrEqual("b", "b")
func GreaterOrEqual[T any](e2 any, msgAndArgs ...any) Rule {
	return require[T]("GreaterOrEqual", varRef, e2, msgAndArgs)
}

// GreaterOrEqualf asserts that the first element is greater than or equal to the second
//
//	a.GreaterOrEqualf(2, 1, "error message %s", "formatted")
//	a.GreaterOrEqualf(2, 2, "error message %s", "formatted")
//	a.GreaterOrEqualf("b", "a", "error message %s", "formatted")
//	a.GreaterOrEqualf("b", "b", "error message %s", "formatted")
func GreaterOrEqualf[T any](e2 any, msg string, args ...any) Rule {
	return require[T]("GreaterOrEqualf", varRef, e2, msg, args)
}

// Greaterf asserts that the first element is greater than the second
//
//	a.Greaterf(2, 1, "error message %s", "formatted")
//	a.Greaterf(float64(2), float64(1), "error message %s", "formatted")
//	a.Greaterf("b", "a", "error message %s", "formatted")
func Greaterf[T any](e2 any, msg string, args ...any) Rule {
	return require[T]("Greaterf", varRef, e2, msg, args)
}

// Implements asserts that an object is implemented by the specified interface.
//
//	a.Implements((*MyInterface)(nil), new(MyObject))
func Implements[T any](interfaceObject any, msgAndArgs ...any) Rule {
	return require[T]("Implements", interfaceObject, varRef, msgAndArgs)
}

// Implementsf asserts that an object is implemented by the specified interface.
//
//	a.Implementsf((*MyInterface)(nil), new(MyObject), "error message %s", "formatted")
func Implementsf[T any](interfaceObject any, msg string, args ...any) Rule {
	return require[T]("Implementsf", interfaceObject, varRef, msg, args)
}

// InDelta asserts that the two numerals are within delta of each other.
//
//	a.InDelta(math.Pi, 22/7.0, 0.01)
func InDelta[T any](expected any, delta float64, msgAndArgs ...any) Rule {
	return require[T]("InDelta", expected, varRef, delta, msgAndArgs)
}

// InDeltaMapValues is the same as InDelta, but it compares all values between two maps. Both maps must have exactly the same keys.
func InDeltaMapValues[T any](expected any, delta float64, msgAndArgs ...any) Rule {
	return require[T]("InDeltaMapValues", expected, varRef, delta, msgAndArgs)
}

// InDeltaMapValuesf is the same as InDelta, but it compares all values between two maps. Both maps must have exactly the same keys.
func InDeltaMapValuesf[T any](expected any, delta float64, msg string, args ...any) Rule {
	return require[T]("InDeltaMapValuesf", expected, varRef, delta, msg, args)
}

// InDeltaSlice is the same as InDelta, except it compares two slices.
func InDeltaSlice[T any](expected any, delta float64, msgAndArgs ...any) Rule {
	return require[T]("InDeltaSlice", expected, varRef, delta, msgAndArgs)
}

// InDeltaSlicef is the same as InDelta, except it compares two slices.
func InDeltaSlicef[T any](expected any, delta float64, msg string, args ...any) Rule {
	return require[T]("InDeltaSlicef", expected, varRef, delta, msg, args)
}

// InDeltaf asserts that the two numerals are within delta of each other.
//
//	a.InDeltaf(math.Pi, 22/7.0, 0.01, "error message %s", "formatted")
func InDeltaf[T any](expected any, delta float64, msg string, args ...any) Rule {
	return require[T]("InDeltaf", expected, varRef, delta, msg, args)
}

// InEpsilon asserts that expected and actual have a relative error less than epsilon
func InEpsilon[T any](expected any, epsilon float64, msgAndArgs ...any) Rule {
	return require[T]("InEpsilon", expected, varRef, epsilon, msgAndArgs)
}

// InEpsilonSlice is the same as InEpsilon, except it compares each value from two slices.
func InEpsilonSlice[T any](expected any, epsilon float64, msgAndArgs ...any) Rule {
	return require[T]("InEpsilonSlice", expected, varRef, epsilon, msgAndArgs)
}

// InEpsilonSlicef is the same as InEpsilon, except it compares each value from two slices.
func InEpsilonSlicef[T any](expected any, epsilon float64, msg string, args ...any) Rule {
	return require[T]("InEpsilonSlicef", expected, varRef, epsilon, msg, args)
}

// InEpsilonf asserts that expected and actual have a relative error less than epsilon
func InEpsilonf[T any](expected any, epsilon float64, msg string, args ...any) Rule {
	return require[T]("InEpsilonf", expected, varRef, epsilon, msg, args)
}

// IsDecreasing asserts that the collection is decreasing
//
//	a.IsDecreasing([]int{2, 1, 0})
//	a.IsDecreasing([]float{2, 1})
//	a.IsDecreasing([]string{"b", "a"})
func IsDecreasing[T any](msgAndArgs ...any) Rule {
	return require[T]("IsDecreasing", varRef, msgAndArgs)
}

// IsDecreasingf asserts that the collection is decreasing
//
//	a.IsDecreasingf([]int{2, 1, 0}, "error message %s", "formatted")
//	a.IsDecreasingf([]float{2, 1}, "error message %s", "formatted")
//	a.IsDecreasingf([]string{"b", "a"}, "error message %s", "formatted")
func IsDecreasingf[T any](msg string, args ...any) Rule {
	return require[T]("IsDecreasingf", varRef, msg, args)
}

// IsIncreasing asserts that the collection is increasing
//
//	a.IsIncreasing([]int{1, 2, 3})
//	a.IsIncreasing([]float{1, 2})
//	a.IsIncreasing([]string{"a", "b"})
func IsIncreasing[T any](msgAndArgs ...any) Rule {
	return require[T]("IsIncreasing", varRef, msgAndArgs)
}

// IsIncreasingf asserts that the collection is increasing
//
//	a.IsIncreasingf([]int{1, 2, 3}, "error message %s", "formatted")
//	a.IsIncreasingf([]float{1, 2}, "error message %s", "formatted")
//	a.IsIncreasingf([]string{"a", "b"}, "error message %s", "formatted")
func IsIncreasingf[T any](msg string, args ...any) Rule {
	return require[T]("IsIncreasingf", varRef, msg, args)
}

// IsNonDecreasing asserts that the collection is not decreasing
//
//	a.IsNonDecreasing([]int{1, 1, 2})
//	a.IsNonDecreasing([]float{1, 2})
//	a.IsNonDecreasing([]string{"a", "b"})
func IsNonDecreasing[T any](msgAndArgs ...any) Rule {
	return require[T]("IsNonDecreasing", varRef, msgAndArgs)
}

// IsNonDecreasingf asserts that the collection is not decreasing
//
//	a.IsNonDecreasingf([]int{1, 1, 2}, "error message %s", "formatted")
//	a.IsNonDecreasingf([]float{1, 2}, "error message %s", "formatted")
//	a.IsNonDecreasingf([]string{"a", "b"}, "error message %s", "formatted")
func IsNonDecreasingf[T any](msg string, args ...any) Rule {
	return require[T]("IsNonDecreasingf", varRef, msg, args)
}

// IsNonIncreasing asserts that the collection is not increasing
//
//	a.IsNonIncreasing([]int{2, 1, 1})
//	a.IsNonIncreasing([]float{2, 1})
//	a.IsNonIncreasing([]string{"b", "a"})
func IsNonIncreasing[T any](msgAndArgs ...any) Rule {
	return require[T]("IsNonIncreasing", varRef, msgAndArgs)
}

// IsNonIncreasingf asserts that the collection is not increasing
//
//	a.IsNonIncreasingf([]int{2, 1, 1}, "error message %s", "formatted")
//	a.IsNonIncreasingf([]float{2, 1}, "error message %s", "formatted")
//	a.IsNonIncreasingf([]string{"b", "a"}, "error message %s", "formatted")
func IsNonIncreasingf[T any](msg string, args ...any) Rule {
	return require[T]("IsNonIncreasingf", varRef, msg, args)
}

// IsType asserts that the specified objects are of the same type.
func IsType[T any](expectedType any, msgAndArgs ...any) Rule {
	return require[T]("IsType", expectedType, varRef, msgAndArgs)
}

// IsTypef asserts that the specified objects are of the same type.
func IsTypef[T any](expectedType any, msg string, args ...any) Rule {
	return require[T]("IsTypef", expectedType, varRef, msg, args)
}

// Len asserts that the specified object has specific length.
// Len also fails if the object has a type that len() not accept.
//
//	a.Len(mySlice, 3)
func Len[T any](length int, msgAndArgs ...any) Rule {
	return require[T]("Len", varRef, length, msgAndArgs)
}

// Lenf asserts that the specified object has specific length.
// Lenf also fails if the object has a type that len() not accept.
//
//	a.Lenf(mySlice, 3, "error message %s", "formatted")
func Lenf[T any](length int, msg string, args ...any) Rule {
	return require[T]("Lenf", varRef, length, msg, args)
}

// Less asserts that the first element is less than the second
//
//	a.Less(1, 2)
//	a.Less(float64(1), float64(2))
//	a.Less("a", "b")
func Less[T any](e2 any, msgAndArgs ...any) Rule {
	return require[T]("Less", varRef, e2, msgAndArgs)
}

// LessOrEqual asserts that the first element is less than or equal to the second
//
//	a.LessOrEqual(1, 2)
//	a.LessOrEqual(2, 2)
//	a.LessOrEqual("a", "b")
//	a.LessOrEqual("b", "b")
func LessOrEqual[T any](e2 any, msgAndArgs ...any) Rule {
	return require[T]("LessOrEqual", varRef, e2, msgAndArgs)
}

// LessOrEqualf asserts that the first element is less than or equal to the second
//
//	a.LessOrEqualf(1, 2, "error message %s", "formatted")
//	a.LessOrEqualf(2, 2, "error message %s", "formatted")
//	a.LessOrEqualf("a", "b", "error message %s", "formatted")
//	a.LessOrEqualf("b", "b", "error message %s", "formatted")
func LessOrEqualf[T any](e2 any, msg string, args ...any) Rule {
	return require[T]("LessOrEqualf", varRef, e2, msg, args)
}

// Lessf asserts that the first element is less than the second
//
//	a.Lessf(1, 2, "error message %s", "formatted")
//	a.Lessf(float64(1), float64(2), "error message %s", "formatted")
//	a.Lessf("a", "b", "error message %s", "formatted")
func Lessf[T any](e2 any, msg string, args ...any) Rule {
	return require[T]("Lessf", varRef, e2, msg, args)
}

// Negative asserts that the specified element is negative
//
//	a.Negative(-1)
//	a.Negative(-1.23)
func Negative[T any](msgAndArgs ...any) Rule {
	return require[T]("Negative", varRef, msgAndArgs)
}

// Negativef asserts that the specified element is negative
//
//	a.Negativef(-1, "error message %s", "formatted")
//	a.Negativef(-1.23, "error message %s", "formatted")
func Negativef[T any](msg string, args ...any) Rule {
	return require[T]("Negativef", varRef, msg, args)
}

// Nil asserts that the specified object is nil.
//
//	a.Nil(err)
func Nil[T any](msgAndArgs ...any) Rule {
	return require[T]("Nil", varRef, msgAndArgs)
}

// Nilf asserts that the specified object is nil.
//
//	a.Nilf(err, "error message %s", "formatted")
func Nilf[T any](msg string, args ...any) Rule {
	return require[T]("Nilf", varRef, msg, args)
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.NoError(err) Rule {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoError[T error](msgAndArgs ...any) Rule {
	return require[T]("NoError", varRef, msgAndArgs)
}

// NoErrorf asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if a.NoErrorf(err, "error message %s", "formatted") Rule {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoErrorf[T error](msg string, args ...any) Rule {
	return require[T]("NoErrorf", varRef, msg, args)
}

// NotContains asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	a.NotContains("Hello World", "Earth")
//	a.NotContains(["Hello", "World"], "Earth")
//	a.NotContains({"Hello": "World"}, "Earth")
func NotContains[T any](contains any, msgAndArgs ...any) Rule {
	return require[T]("NotContains", varRef, contains, msgAndArgs)
}

// NotContainsf asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	a.NotContainsf("Hello World", "Earth", "error message %s", "formatted")
//	a.NotContainsf(["Hello", "World"], "Earth", "error message %s", "formatted")
//	a.NotContainsf({"Hello": "World"}, "Earth", "error message %s", "formatted")
func NotContainsf[T any](contains any, msg string, args ...any) Rule {
	return require[T]("NotContainsf", varRef, contains, msg, args)
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
//func NotElementsMatch(listA any, listB any, msgAndArgs ...any) Rule {
//	return require[T]("NotElementsMatch", listA, listB, msgAndArgs)
//}

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
//func NotElementsMatchf(listA any, listB any, msg string, args ...any) Rule {
//	return require[T]("NotElementsMatchf", listA, listB, msg, args)
//}

// NotEmpty asserts that the specified object is NOT empty.  I.e. not nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	if a.NotEmpty(obj) Rule {
//	  assert.Equal(t, "two", obj[1])
//	}
func NotEmpty[T any](msgAndArgs ...any) Rule {
	return require[T]("NotEmpty", varRef, msgAndArgs)
}

// NotEmptyf asserts that the specified object is NOT empty.  I.e. not nil, "", false, 0 or either
// a slice or a channel with len == 0.
//
//	if a.NotEmptyf(obj, "error message %s", "formatted") Rule {
//	  assert.Equal(t, "two", obj[1])
//	}
func NotEmptyf[T any](msg string, args ...any) Rule {
	return require[T]("NotEmptyf", varRef, msg, args)
}

// NotEqual asserts that the specified values are NOT equal.
//
//	a.NotEqual(obj1, obj2)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqual[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("NotEqual", expected, varRef, msgAndArgs)
}

// NotEqualValues asserts that two objects are not equal even when converted to the same type
//
//	a.NotEqualValues(obj1, obj2)
func NotEqualValues[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("NotEqualValues", expected, varRef, msgAndArgs)
}

// NotEqualValuesf asserts that two objects are not equal even when converted to the same type
//
//	a.NotEqualValuesf(obj1, obj2, "error message %s", "formatted")
func NotEqualValuesf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("NotEqualValuesf", expected, varRef, msg, args)
}

// NotEqualf asserts that the specified values are NOT equal.
//
//	a.NotEqualf(obj1, obj2, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqualf[T any](expected any, msg string, args ...any) Rule {
	return require[T]("NotEqualf", expected, varRef, msg, args)
}

// NotErrorAs asserts that none of the errors in err's chain matches target,
// but if so, sets target to that error value.
func NotErrorAs[T error](target any, msgAndArgs ...any) Rule {
	return require[T]("NotErrorAs", varRef, target, msgAndArgs)
}

// NotErrorAsf asserts that none of the errors in err's chain matches target,
// but if so, sets target to that error value.
func NotErrorAsf[T error](target any, msg string, args ...any) Rule {
	return require[T]("NotErrorAsf", varRef, target, msg, args)
}

// NotErrorIs asserts that none of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func NotErrorIs[T error](target error, msgAndArgs ...any) Rule {
	return require[T]("NotErrorIs", varRef, target, msgAndArgs)
}

// NotErrorIsf asserts that none of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func NotErrorIsf[T error](target error, msg string, args ...any) Rule {
	return require[T]("NotErrorIsf", varRef, target, msg, args)
}

// NotImplements asserts that an object does not implement the specified interface.
//
//	a.NotImplements((*MyInterface)(nil), new(MyObject))
func NotImplements[T any](interfaceObject any, msgAndArgs ...any) Rule {
	return require[T]("NotImplements", interfaceObject, varRef, msgAndArgs)
}

// NotImplementsf asserts that an object does not implement the specified interface.
//
//	a.NotImplementsf((*MyInterface)(nil), new(MyObject), "error message %s", "formatted")
func NotImplementsf[T any](interfaceObject any, msg string, args ...any) Rule {
	return require[T]("NotImplementsf", interfaceObject, varRef, msg, args)
}

// NotNil asserts that the specified object is not nil.
//
//	a.NotNil(err)
func NotNil[T any](msgAndArgs ...any) Rule {
	return require[T]("NotNil", varRef, msgAndArgs)
}

// NotNilf asserts that the specified object is not nil.
//
//	a.NotNilf(err, "error message %s", "formatted")
func NotNilf[T any](msg string, args ...any) Rule {
	return require[T]("NotNilf", varRef, msg, args)
}

// NotSame asserts that two pointers do not reference the same object.
//
//	a.NotSame(ptr1, ptr2)
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func NotSame[T any](expected any, msgAndArgs ...any) Rule {
	return require[T]("NotSame", expected, varRef, msgAndArgs)
}

// NotSamef asserts that two pointers do not reference the same object.
//
//	a.NotSamef(ptr1, ptr2, "error message %s", "formatted")
//
// Both arguments must be pointer variables. Pointer variable sameness is
// determined based on the equality of both type and value.
func NotSamef[T any](expected any, msg string, args ...any) Rule {
	return require[T]("NotSamef", expected, varRef, msg, args)
}

// NotSubset asserts that the specified list(array, slice...) or map does NOT
// contain all elements given in the specified subset list(array, slice...) or
// map.
//
//	a.NotSubset([1, 3, 4], [1, 2])
//	a.NotSubset({"x": 1, "y": 2}, {"z": 3})
func NotSubset[T any](subset any, msgAndArgs ...any) Rule {
	return require[T]("NotSubset", varRef, subset, msgAndArgs)
}

// NotSubsetf asserts that the specified list(array, slice...) or map does NOT
// contain all elements given in the specified subset list(array, slice...) or
// map.
//
//	a.NotSubsetf([1, 3, 4], [1, 2], "error message %s", "formatted")
//	a.NotSubsetf({"x": 1, "y": 2}, {"z": 3}, "error message %s", "formatted")
func NotSubsetf[T any](subset any, msg string, args ...any) Rule {
	return require[T]("NotSubsetf", varRef, subset, msg, args)
}

// NotZero asserts that i is not the zero value for its type.
func NotZero[T any](msgAndArgs ...any) Rule {
	return require[T]("NotZero", varRef, msgAndArgs)
}

// NotZerof asserts that i is not the zero value for its type.
func NotZerof[T any](msg string, args ...any) Rule {
	return require[T]("NotZerof", varRef, msg, args)
}

// Positive asserts that the specified element is positive
//
//	a.Positive(1)
//	a.Positive(1.23)
func Positive[T any](msgAndArgs ...any) Rule {
	return require[T]("Positive", varRef, msgAndArgs)
}

// Positivef asserts that the specified element is positive
//
//	a.Positivef(1, "error message %s", "formatted")
//	a.Positivef(1.23, "error message %s", "formatted")
func Positivef[T any](msg string, args ...any) Rule {
	return require[T]("Positivef", varRef, msg, args)
}

// Subset asserts that the specified list(array, slice...) or map contains all
// elements given in the specified subset list(array, slice...) or map.
//
//	a.Subset([1, 2, 3], [1, 2])
//	a.Subset({"x": 1, "y": 2}, {"x": 1})
func Subset[T any](subset any, msgAndArgs ...any) Rule {
	return require[T]("Subset", varRef, subset, msgAndArgs)
}

// Subsetf asserts that the specified list(array, slice...) or map contains all
// elements given in the specified subset list(array, slice...) or map.
//
//	a.Subsetf([1, 2, 3], [1, 2], "error message %s", "formatted")
//	a.Subsetf({"x": 1, "y": 2}, {"x": 1}, "error message %s", "formatted")
func Subsetf[T any](subset any, msg string, args ...any) Rule {
	return require[T]("Subsetf", varRef, subset, msg, args)
}

// True asserts that the specified value is true.
//
//	a.True(myBool)
func True[T ~bool](msgAndArgs ...any) Rule {
	return require[T]("True", varRef, msgAndArgs)
}

// Truef asserts that the specified value is true.
//
//	a.Truef(myBool, "error message %s", "formatted")
func Truef[T ~bool](msg string, args ...any) Rule {
	return require[T]("Truef", varRef, msg, args)
}

// WithinDuration asserts that the two times are within duration delta of each other.
//
//	a.WithinDuration(time.Now(), time.Now(), 10*time.Second)
func WithinDuration[T any](expected time.Time, actual time.Time, delta time.Duration, msgAndArgs ...any) Rule {
	return require[T]("WithinDuration", expected, varRef, delta, msgAndArgs)
}

// WithinDurationf asserts that the two times are within duration delta of each other.
//
//	a.WithinDurationf(time.Now(), time.Now(), 10*time.Second, "error message %s", "formatted")
func WithinDurationf[T any](expected time.Time, actual time.Time, delta time.Duration, msg string, args ...any) Rule {
	return require[T]("WithinDurationf", expected, varRef, delta, msg, args)
}

// WithinRange asserts that a time is within a time range (inclusive).
//
//	a.WithinRange(time.Now(), time.Now().Add(-time.Second), time.Now().Add(time.Second))
func WithinRange[T any](actual time.Time, start time.Time, end time.Time, msgAndArgs ...any) Rule {
	return require[T]("WithinRange", varRef, start, end, msgAndArgs)
}

// WithinRangef asserts that a time is within a time range (inclusive).
//
//	a.WithinRangef(time.Now(), time.Now().Add(-time.Second), time.Now().Add(time.Second), "error message %s", "formatted")
func WithinRangef[T any](actual time.Time, start time.Time, end time.Time, msg string, args ...any) Rule {
	return require[T]("WithinRangef", varRef, start, end, msg, args)
}

// Zero asserts that i is the zero value for its type.
func Zero[T any](msgAndArgs ...any) Rule {
	return require[T]("Zero", varRef, msgAndArgs)
}

// Zerof asserts that i is the zero value for its type.
func Zerof[T any](msg string, args ...any) Rule {
	return require[T]("Zerof", varRef, msg, args)
}
