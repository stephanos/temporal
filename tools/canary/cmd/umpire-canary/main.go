// Command umpire-canary is the production canary's one binary. `run` preflights the protected
// workflow's scope, takes the lease, runs the committed activity Model through the Runner serially,
// each Run on a fresh fenced activity, cleans up and publishes each Run. `reconcile` closes exactly
// what its job's recovery record names and never dispatches. Neither takes a Model, target,
// Driver, seed, retry, executable, endpoint, credential or release option: the policy and the
// credential are the build's and the environment's.
package main

import (
	"os"
)

func main() {
	os.Exit(Main(os.Args[1:], os.Stdout, os.Stderr, os.LookupEnv, seams()))
}
