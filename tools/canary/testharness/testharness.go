//go:build canary_harness

// Package testharness is the canary's harness build: a policy read from a test file, a plaintext
// transport to a test cluster, and hooks a live test uses to crash or pause the process at a phase.
// It compiles only under the canary_harness tag, so the untagged binary the protected workflow runs
// contains none of it, and a harness policy's authority class says harness.
package testharness

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc/credentials/insecure"
	"umpire.local/temporal"
)

// pausePoll is how often a paused harness looks for its file.
const pausePoll = 50 * time.Millisecond

// Seams are the harness build's: the test policy, a plaintext transport and the hooks, all from the
// environment lookup gives.
func Seams(lookup authority.Lookup) controller.Seams {
	return controller.Seams{
		Policy:    func() (*policy.Policy, error) { return LoadPolicy(lookup) },
		Guest:     temporal.Guest(),
		Authority: Authority,
		Hook:      hook(lookup),
	}
}

// LoadPolicy reads the test policy strictly and refuses one that names any authority class but
// harness, so no harness run can claim production.
func LoadPolicy(lookup authority.Lookup) (*policy.Policy, error) {
	path, ok := lookup(VariablePolicy)
	if !ok || path == "" {
		return nil, fmt.Errorf("%s is not set", VariablePolicy)
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	canary, err := policy.Decode(encoded)
	if err != nil {
		return nil, err
	}
	if canary.AuthorityClass != policy.AuthorityHarness {
		return nil, fmt.Errorf("a harness policy's authority class is %s only, not %q", policy.AuthorityHarness, canary.AuthorityClass)
	}
	return canary, nil
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
		Redactor:    coordinates.Redactor(),
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
