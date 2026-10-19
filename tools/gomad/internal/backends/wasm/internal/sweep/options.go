// Package sweep runs every top-level test function of a Temporal checkout's
// tests packages under the cooperative WASM profile and classifies each
// outcome (fn-221). It drives a prebuilt gomad binary as a subprocess, one
// `explore --json` campaign per test function, so the explore path it observes
// is the operator's, unchanged. A sweep row is an observation on one candidate;
// it is never a support or qualification claim.
package sweep

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrRefused marks invalid or unbounded input. Commands map it to exit 2.
var ErrRefused = errors.New("refused")

// ConfigSource names where a package's installation came from.
type ConfigSource string

const (
	// ConfigThrowaway is a wasi.Config derived here from the measure-fixture
	// parameters and installed with `gomad install --config`. It exists only
	// for this sweep and is labelled on every row.
	ConfigThrowaway ConfigSource = "throwaway"
	// ConfigTemporalInstall uses fn-220's `gomad install --temporal-checkout
	// --temporal-package` once that command exists.
	ConfigTemporalInstall ConfigSource = "temporal-install"
)

// Limits on explicit budgets. A budget above them is treated as unbounded.
const (
	MaximumPerTestTimeout = 6 * time.Hour
	MaximumPrepareTimeout = 2 * time.Hour
	MaximumOverallBudget  = 14 * 24 * time.Hour
)

// Options are the sweep's explicit inputs. Nothing is read from the
// environment except what the gomad and go subprocesses themselves inherit,
// and Validate refuses any inherited GOMAD_* variable.
type Options struct {
	Checkout     string // absolute Temporal checkout (module go.temporal.io/server)
	Gomad        string // prebuilt gomad binary
	Compiler     string // pinned stock Go 1.27.1 go binary
	Helper       string // pinned gomad-wasmhost helper
	RuntimeRoot  string // tools/gomad/internal/model/runtimesource of the candidate
	ConfigSource ConfigSource
	Packages     []string // relative package patterns, e.g. ./tests/...
	MaxTests     int      // per-package cap in sorted order; 0 = every test
	Seed         uint64
	// PerTestTimeout is explore's --execution-timeout. PrepareTimeout is the
	// allowance added for the cold guest build and Cranelift compilation;
	// explore's --overall-timeout is their sum.
	PerTestTimeout, PrepareTimeout, OverallBudget time.Duration
	RetainBytes                                   uint64 // exemplar artifact budget
	Out                                           string // JSONL journal and sidecar
	Work                                          string // installations, caches, per-test artifact roots
	Artifacts                                     string // retained exemplar artifacts
	Environ                                       []string
}

// Validate refuses missing, relative, overlapping or unbounded inputs and any
// inherited GOMAD_* variable.
func (options Options) Validate() error {
	for _, entry := range options.Environ {
		if key, _, _ := strings.Cut(entry, "="); strings.HasPrefix(key, "GOMAD_") {
			return fmt.Errorf("%w: unset %s; pass inputs as flags", ErrRefused, key)
		}
	}
	for _, path := range []struct{ name, value string }{
		{"--checkout", options.Checkout}, {"--gomad", options.Gomad}, {"--compiler", options.Compiler},
		{"--helper", options.Helper}, {"--runtime-root", options.RuntimeRoot}, {"--out", options.Out},
		{"--work", options.Work}, {"--artifacts", options.Artifacts},
	} {
		if path.value == "" {
			return fmt.Errorf("%w: %s is required", ErrRefused, path.name)
		}
		if !filepath.IsAbs(path.value) || filepath.Clean(path.value) != path.value {
			return fmt.Errorf("%w: %s must be an absolute clean path", ErrRefused, path.name)
		}
	}
	for _, path := range []struct{ name, value string }{{"--work", options.Work}, {"--artifacts", options.Artifacts}, {"--out", options.Out}} {
		if within(options.Checkout, path.value) && path.name != "--out" {
			return fmt.Errorf("%w: %s must be outside --checkout", ErrRefused, path.name)
		}
	}
	if options.Out == options.Checkout {
		return fmt.Errorf("%w: --out must not be the checkout itself", ErrRefused)
	}
	switch options.ConfigSource {
	case ConfigThrowaway, ConfigTemporalInstall:
	default:
		return fmt.Errorf("%w: --config-source must be %q or %q", ErrRefused, ConfigThrowaway, ConfigTemporalInstall)
	}
	if len(options.Packages) == 0 {
		return fmt.Errorf("%w: --packages is required", ErrRefused)
	}
	for _, pattern := range options.Packages {
		if pattern != "./tests" && !strings.HasPrefix(pattern, "./tests/") {
			return fmt.Errorf("%w: package pattern %q is not under ./tests", ErrRefused, pattern)
		}
	}
	if options.MaxTests < 0 {
		return fmt.Errorf("%w: --max-tests must be nonnegative", ErrRefused)
	}
	for _, budget := range []struct {
		name           string
		value, maximum time.Duration
	}{{"--per-test-timeout", options.PerTestTimeout, MaximumPerTestTimeout}, {"--prepare-timeout", options.PrepareTimeout, MaximumPrepareTimeout}, {"--overall-budget", options.OverallBudget, MaximumOverallBudget}} {
		if budget.value <= 0 {
			return fmt.Errorf("%w: %s must be a positive bound; unbounded runs are refused", ErrRefused, budget.name)
		}
		if budget.value > budget.maximum {
			return fmt.Errorf("%w: %s %s exceeds %s", ErrRefused, budget.name, budget.value, budget.maximum)
		}
	}
	if options.OverallBudget < options.PerTestTimeout+options.PrepareTimeout {
		return fmt.Errorf("%w: --overall-budget cannot fit one test (per-test + prepare)", ErrRefused)
	}
	return nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// sanitizedEnviron returns the inherited environment for subprocesses.
// Validate has already refused GOMAD_*; nothing else is added or removed.
func sanitizedEnviron(environ []string) []string {
	return append([]string(nil), environ...)
}

func requireRegularFiles(options Options) error {
	for _, path := range []struct{ name, value string }{{"--gomad", options.Gomad}, {"--compiler", options.Compiler}, {"--helper", options.Helper}} {
		info, err := os.Stat(path.value)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			return fmt.Errorf("%w: %s %s is not an executable regular file", ErrRefused, path.name, path.value)
		}
	}
	for _, path := range []struct{ name, value string }{{"--checkout", options.Checkout}, {"--runtime-root", options.RuntimeRoot}} {
		info, err := os.Stat(path.value)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("%w: %s %s is not a directory", ErrRefused, path.name, path.value)
		}
	}
	if _, err := os.Stat(filepath.Join(options.Checkout, "tests", "testcore")); err != nil {
		return fmt.Errorf("%w: %s is not a Temporal checkout with tests/testcore", ErrRefused, options.Checkout)
	}
	return nil
}
