//go:build canary_harness

// Package testharness is the canary's harness build: a policy read from a test file, the
// canary-harness Evaluation Profile, a plaintext transport to a test cluster, and hooks a live test
// uses to crash or pause the process at a phase. It compiles only under the canary_harness tag, so
// the untagged binary the protected workflow runs contains none of it, and a harness receipt is
// never a production receipt: its Profile, trust and authority class say harness.
package testharness

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/evaluation"
	"google.golang.org/grpc/credentials/insecure"
)

// pausePoll is how often a paused harness looks for its file.
const pausePoll = 50 * time.Millisecond

// The Lean-rendered canary-harness Profile, which only this build embeds.
//
//go:embed profiles/*.json
var profiles embed.FS

// Seams are the harness build's: the test policy, a plaintext transport and the hooks, all from the
// environment lookup gives.
func Seams(lookup authority.Lookup) controller.Seams {
	return controller.Seams{
		Policy:    func() (*policy.Policy, *evaluation.Profile, error) { return LoadPolicy(lookup) },
		Authority: Authority,
		Hook:      hook(lookup),
	}
}

// LoadPolicy reads the test policy strictly and refuses one that names any Evaluation Profile but
// canary-harness or any authority class but harness, so no harness run can claim production.
func LoadPolicy(lookup authority.Lookup) (*policy.Policy, *evaluation.Profile, error) {
	path, ok := lookup(VariablePolicy)
	if !ok || path == "" {
		return nil, nil, fmt.Errorf("%s is not set", VariablePolicy)
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	canary, err := policy.Decode(encoded)
	if err != nil {
		return nil, nil, err
	}
	if canary.EvaluationProfile != ProfileName {
		return nil, nil, fmt.Errorf("a harness policy names the %s Evaluation Profile only, not %q", ProfileName, canary.EvaluationProfile)
	}
	if canary.AuthorityClass != policy.AuthorityHarness {
		return nil, nil, fmt.Errorf("a harness policy's authority class is %s only, not %q", policy.AuthorityHarness, canary.AuthorityClass)
	}
	profile, err := evaluation.LoadProfileIn(profiles, ProfileName)
	if err != nil {
		return nil, nil, err
	}
	return canary, profile, nil
}

// Authority is the harness's plaintext transport to the coordinates' target, which the untagged
// build never builds; it needs no credential, and its Redactor knows every coordinate.
func Authority(lookup authority.Lookup) (*authority.Authority, error) {
	coordinates, err := authority.LoadCoordinates(lookup)
	if err != nil {
		return nil, err
	}
	return &authority.Authority{
		Coordinates: coordinates,
		Transport:   authority.Transport{Target: coordinates.GRPC, Credentials: insecure.NewCredentials()},
		Redactor: authority.NewRedactor(coordinates.GRPC, coordinates.Namespace, coordinates.TaskQueue,
			coordinates.HandlerQueue, coordinates.NexusEndpoint),
	}, nil
}

// hook crashes or pauses the process at the phases the environment names; with neither set it
// does nothing.
func hook(lookup authority.Lookup) func(string) {
	crash, _ := lookup(VariableCrash)
	pause, _ := lookup(VariablePause)
	pausePhase, pauseFile, _ := strings.Cut(pause, ":")
	return func(phase string) {
		if crash != "" && phase == crash {
			os.Exit(CrashExit)
		}
		if pausePhase != "" && phase == pausePhase {
			ticker := time.NewTicker(pausePoll)
			defer ticker.Stop()
			for {
				if _, err := os.Stat(pauseFile); !errors.Is(err, os.ErrNotExist) {
					return
				}
				<-ticker.C
			}
		}
	}
}
