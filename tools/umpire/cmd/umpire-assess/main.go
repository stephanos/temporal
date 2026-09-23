// Command umpire-assess assesses one recorded Run of one Case under one Evaluation Profile.
//
// `umpire-assess run` names the subject -- a canonical Case and the Run recorded against it -- an
// Evaluation Profile by its exact name, and a receipt root outside the model. It admits the
// subject strictly, assesses the recorded Verdict under the Profile, renders the canonical
// receipt and publishes it under its identity, exclusively. It prepares, runs and replays
// nothing, and takes no Driver, deployment, endpoint, credential, checker, policy or retry flag.
// One JSON summary goes to stdout; stderr carries one line saying what happened.
package main

import (
	"os"

	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
)

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr, environment{Catalog: treeCatalog}))
}

// treeCatalog is the fingerprint of the tree's static method catalog, which a recorded Run must
// carry to be current; building it reads no Case and opens nothing.
func treeCatalog() (string, error) {
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return "", err
	}
	return catalog.Identity(), nil
}
