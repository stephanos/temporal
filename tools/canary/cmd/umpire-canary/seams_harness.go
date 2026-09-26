//go:build canary_harness

package main

import (
	"os"

	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/testharness"
)

// seams are the harness build's: a test policy, a plaintext transport and the crash and pause
// hooks, all from the environment. Only the live tests build this.
func seams() controller.Seams { return testharness.Seams(os.LookupEnv) }
