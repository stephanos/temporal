//go:build !canary_harness

package main

import "go.temporal.io/server/tools/canary/controller"

// seams are the untagged build's only ones: the embedded policy, the credential-requiring
// authority and no hook. No flag or environment variable replaces them.
func seams() controller.Seams { return controller.ProductionSeams() }
