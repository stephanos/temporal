// Package intercept holds the interception targets and hooks that the
// conformance compiler rewrites; compiler-tests.json names each pair.
package intercept

import "strings"

// Value is the receiver for the method interception case.
type Value struct {
	Total int
}

var hookCalls = map[string]int{}

// Function is the plain function interception target.
func Function(value int) int {
	return value * 2
}

// Notify is the variadic interception target.
func Notify(values ...string) string {
	return strings.Join(values, ",")
}

// Add is the pointer-receiver method interception target.
func (v *Value) Add(delta int) int {
	v.Total += delta
	return v.Total
}

func gomadInterceptFunction(value int) (int, bool) {
	hookCalls["Function"]++
	return value + 1000, true
}

func gomadInterceptNotify(values ...string) (string, bool) {
	hookCalls["Notify"]++
	return "hooked:" + strings.Join(values, "|"), true
}

func gomadInterceptValueAdd(v *Value, delta int) (int, bool) {
	hookCalls["Add"]++
	if delta < 0 {
		return 0, false
	}
	v.Total += 2 * delta
	return v.Total, true
}
