/*
Package metatesting is a package for writing normal go tests that invoke gomad
tests. Such metatests can run gomad tests with different seeds, assert that some
scenarios happen often or never, etc. With metatesting, gomad tests can be
integrated in a normal go test run, so that 'go test ./...' for a module can
cover both simulated and non-simulated tests.

# Using build constraints to combine gomad tests and metatests

The metatesting API does not work inside gomad. To combine gomad tests and metatests
in a single package, use the '//go:build gomad' and '//go:build !gomad' contraints
at the top of test files.

# Working with the go test cache

Metatesting has some unfortunate interactions with go test cache. To run
metatests using 'go test', the run needs either be non-cached or the metatests
need to be precompiled.

To run metatests with a non-cached 'go test' invocation, run 'go test -count=1
./package/with/metatests'. Then metatesting will build the gomad tests when
needed. This is convenient, but this metatest will not be cached.

To cache tests, the gomad tests need to be built before running the metatest
with 'gomad build-tests'.  If metatesting is used incorrectly it tries to give a
helpful error message. A simple way to run tests is to use a script that invoke
'gomad build-tests' before running 'go test':

	# build gomad test binaries
	gomad build-tests pkgA pkgA/pkgB
	# then run tests
	go test ./...

Tests need to be precompiled when using the test cache because the test cache
should be invalidated when the gomad test changes. Normal go tests get
invalidated when their files change because the go tool knows which files are
compiled into the test. However, for a metatest that invokes 'gomad test' the go
tool does not know what files 'gomad test' accesses, and so does not invalidate
the cache if those files change. To sidestep those issues, run 'gomad
build-tests' before running 'go test'.

TODO: Support fuzzing.

TODO: Support parallelism.
*/
package metatesting
