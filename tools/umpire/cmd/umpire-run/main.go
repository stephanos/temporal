// Command umpire-run runs one checked-in Case against any Temporal deployment.
//
// It is the black-box consumer of the Case bytes: it reads a fixture, derives the Profile the Case
// implies, binds it to the namespace, task queue and Nexus endpoint the caller names, runs once,
// and reports the Verdict. It links the Driver and the SDK, never the test cluster.
package main

import (
	"os"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr, openSession))
}
