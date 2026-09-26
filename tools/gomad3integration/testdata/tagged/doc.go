// Package tagged is the wrapper fixture for the root gomad3-test target: its
// only test exists behind the test_dep build tag, so a run that drops the tag
// reports no test files instead of passing vacuously.
package tagged
