// Command umpire-canary is the production canary's one binary. `run` preflights the protected
// workflow's scope, takes the lease, runs the pinned canary Case serially, cleans up and publishes
// each iteration's fn-26 receipt and provenance. `reconcile` closes exactly what its job's
// recovery record names and never dispatches. Neither takes a Case, target, Driver, checker, retry,
// executable, endpoint, credential or release option: the policy and the credential are the
// build's and the environment's.
package main

import (
	"os"
)

func main() {
	os.Exit(Main(os.Args[1:], os.Stdout, os.Stderr, os.LookupEnv, seams()))
}
