// Command umpire-replay replays one violated Run and reduces its Query.
//
// `umpire-replay run` names the subject -- a canonical Case and the Run recorded against it with
// the Profile identity it was prepared under -- the set and Query (or exploration target) the
// Lean replay bridge recovers it by, and the deployment `umpire-run` binds to. It admits the
// subject and replays it offline before anything is opened, reruns it twice against the
// deployment, reduces its Query through the bridge under fixed limits, and writes the retained
// candidate's review-only proposal under `--promotion-root`. One canonical JSON report goes to
// stdout; stderr carries bounded progress and the diagnostics.
package main

import (
	"os"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr, deploymentEnvironment))
}
