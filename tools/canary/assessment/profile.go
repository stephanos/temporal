// Package assessment is the canary's use of fn-26 Claim Assessment: the Evaluation Profile the
// canary assesses under, and (in later parts) the admission of each iteration and the provenance
// bound to its receipt.
package assessment

import (
	"embed"
	"fmt"
	"path"
	"strings"

	"go.temporal.io/server/tools/umpire/evaluation"
)

// The Lean-rendered canary Profile, `make umpire-gen-evaluation-profiles` writes it; the harness's
// Profile lives with the harness build and never here.
//
//go:embed profiles/*.json
var embeddedProfiles embed.FS

// LoadProfile loads an embedded canary Profile by its exact name through fn-26's strict parse.
func LoadProfile(name string) (*evaluation.Profile, error) {
	// A Profile is a name, never a path.
	if name == "" || strings.ContainsAny(name, `/\.`) {
		return nil, fmt.Errorf("%w: %q", evaluation.ErrUnknownProfile, name)
	}
	encoded, err := embeddedProfiles.ReadFile(path.Join("profiles", name+".json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %q", evaluation.ErrUnknownProfile, name)
	}
	profile, err := evaluation.ParseProfile(encoded)
	if err != nil {
		return nil, fmt.Errorf("canary Evaluation Profile %q: %w", name, err)
	}
	if profile.Name != name {
		return nil, fmt.Errorf("the canary Evaluation Profile file %q names Profile %q", name, profile.Name)
	}
	return profile, nil
}
