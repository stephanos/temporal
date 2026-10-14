// Package assessment is the canary's use of fn-26 Claim Assessment: the Evaluation Profile the
// canary assesses under, and (in later parts) the admission of each iteration and the provenance
// bound to its receipt.
package assessment

import (
	"embed"

	"go.temporal.io/server/common/testing/testpilot/evaluation"
)

// The canary Profile: checked-in configuration data that no live target renders; the harness's
// Profile lives with the harness build and never here.
//
//go:embed profiles/*.json
var embeddedProfiles embed.FS

// LoadProfile loads an embedded canary Profile by its exact name through fn-26's loader.
func LoadProfile(name string) (*evaluation.Profile, error) {
	return evaluation.LoadProfileIn(embeddedProfiles, name)
}
