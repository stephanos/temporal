// Command umpire-fuzz runs one bounded exploration campaign against any Temporal deployment.
//
// `umpire-fuzz run` names an exploratory set and the deployment `umpire-run` binds to, opens the
// Lean exploration bridge, and takes the campaign's candidates one at a time through preparation,
// one Run and cleanup, handing each closed Run back to the bridge. It names no target and widens
// no Limit: what it adds are the campaign's own caps. One canonical JSON summary goes to stdout at
// the end; stderr carries the bridge's progress lines per candidate (the candidate handed out and its
// observation) and the diagnostics.
package main

import (
	"os"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr, openCampaign))
}
