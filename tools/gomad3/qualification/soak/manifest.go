// Package soak runs the scheduled determinism soak: N fresh same-seed
// repetitions per seed of a named workload selection, in gomad qualify
// batches of at most 32, under bounded unrelated CPU load, with choice tracing
// and the diagnostic trace on.
//
// A qualify batch compares its repetitions only with its own first execution,
// so the soak also compares each batch's evidence baseline with the baseline
// of its cohort (workload, seed, platform, execution identity), within one run
// and across the retained runs of a cumulative ledger. Any disagreement is a
// divergence; a trace overflow, a target failure, and an infrastructure
// failure are reported separately and none of them is a pass.
package soak

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/qualification/set"
)

const ManifestSchema = "gomad3.determinism-soak/v1"

const maximumManifestBytes = 1 << 20

// maximumBatchRepeat is gomad qualify's repetition bound.
const maximumBatchRepeat = 32

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,127}$`)

var platformPattern = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+$`)

// Manifest names a soak selection and its size. Selection manifests and
// working directories are relative to the soak manifest's directory.
type Manifest struct {
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Seeds       []uint64 `json:"seeds"`
	// BatchRepeat is the repetition count of one gomad qualify batch.
	BatchRepeat uint64 `json:"repeat"`
	// A run executes rounds of one batch per workload and seed. It always runs
	// MinimumBatches rounds and starts each further round, up to Batches, only
	// while the previous round's measured cost still fits the budget, so N is
	// the largest multiple of BatchRepeat that fits the scheduled job.
	MinimumBatches uint64 `json:"minimum_batches"`
	Batches        uint64 `json:"batches"`
	// BatchTimeout bounds one batch; Budget bounds the whole run. A batch
	// that cannot start inside the budget is reported, not skipped silently.
	BatchTimeout string `json:"qualify_timeout"`
	Budget       string `json:"budget"`
	// LoadWorkers is the number of busy host threads the soak runs beside
	// the workload for its whole duration.
	LoadWorkers uint64 `json:"load_workers"`
	// Sizing records how N was chosen, so the bound can be quoted with it.
	Sizing string `json:"sizing"`
	// InformationalPlatforms maps a platform to the open finding that keeps
	// its divergences from failing the gate. Every other platform is strict.
	InformationalPlatforms map[string]string `json:"informational_platforms,omitempty"`
	Selections             []Selection       `json:"selections"`
}

type Selection struct {
	Manifest   string   `json:"manifest"`
	WorkingDir string   `json:"working_dir"`
	Suites     []string `json:"suites"`
}

// workloadPlan is one selected suite resolved from its qualification-set
// manifest.
type workloadPlan struct {
	setManifest set.Manifest
	workload    set.Workload
	workingDir  string
}

func LoadManifest(path string) (Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read soak manifest: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumManifestBytes {
		return Manifest{}, fmt.Errorf("soak manifest must be between 1 and %d bytes", maximumManifestBytes)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read soak manifest: %w", err)
	}
	var manifest Manifest
	if err := canonicaljson.StrictDecode(contents, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode soak manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema || !namePattern.MatchString(manifest.Name) || strings.TrimSpace(manifest.Description) == "" || strings.TrimSpace(manifest.Sizing) == "" {
		return errors.New("soak manifest identity is invalid")
	}
	if len(manifest.Seeds) == 0 || len(manifest.Seeds) > 32 {
		return errors.New("soak manifest needs between 1 and 32 seeds")
	}
	for index, seed := range manifest.Seeds {
		if index > 0 && seed <= manifest.Seeds[index-1] {
			return errors.New("soak manifest seeds must be sorted and unique")
		}
	}
	if manifest.BatchRepeat < 2 || manifest.BatchRepeat > maximumBatchRepeat {
		return fmt.Errorf("soak repeat must be between 2 and %d", maximumBatchRepeat)
	}
	if manifest.MinimumBatches == 0 || manifest.Batches < manifest.MinimumBatches || manifest.Batches > 1024 {
		return errors.New("soak minimum_batches must be at least 1 and batches between it and 1024")
	}
	if manifest.LoadWorkers > 64 {
		return errors.New("soak load_workers must be at most 64")
	}
	if err := validateDurations(manifest); err != nil {
		return err
	}
	for platform, finding := range manifest.InformationalPlatforms {
		if !platformPattern.MatchString(platform) || strings.TrimSpace(finding) == "" {
			return fmt.Errorf("soak informational platform %q needs a GOOS/GOARCH key and a finding", platform)
		}
	}
	return validateSelections(manifest.Selections)
}

func validateDurations(manifest Manifest) error {
	batchTimeout, err := time.ParseDuration(manifest.BatchTimeout)
	if err != nil || batchTimeout <= 0 {
		return errors.Join(errors.New("soak qualify_timeout is invalid"), err)
	}
	budget, err := time.ParseDuration(manifest.Budget)
	if err != nil || budget < batchTimeout {
		return errors.Join(errors.New("soak budget is invalid or shorter than one batch"), err)
	}
	return nil
}

func validateSelections(selections []Selection) error {
	if len(selections) == 0 {
		return errors.New("soak manifest selects no workloads")
	}
	seen := map[string]bool{}
	for index, selection := range selections {
		if !relativePath(selection.Manifest) || !relativePath(selection.WorkingDir) || len(selection.Suites) == 0 {
			return fmt.Errorf("soak selection %d needs a relative manifest, working directory, and suites", index)
		}
		for _, suite := range selection.Suites {
			if !namePattern.MatchString(suite) || seen[suite] {
				return fmt.Errorf("soak selection %d suite %q is invalid or repeated", index, suite)
			}
			seen[suite] = true
		}
	}
	return nil
}

func relativePath(path string) bool {
	return path != "" && !filepath.IsAbs(path) && !strings.ContainsRune(path, 0)
}

// resolve loads every selection's qualification-set manifest and returns the
// selected workloads in manifest order, restricted to the named workloads when
// any are given.
func resolve(manifestPath string, manifest Manifest, only []string) ([]workloadPlan, error) {
	root := filepath.Dir(manifestPath)
	var plans []workloadPlan
	for _, selection := range manifest.Selections {
		setManifest, err := set.LoadManifest(filepath.Join(root, selection.Manifest))
		if err != nil {
			return nil, err
		}
		workingDir, err := filepath.Abs(filepath.Join(root, selection.WorkingDir))
		if err != nil {
			return nil, err
		}
		for _, id := range selection.Suites {
			index := slices.IndexFunc(setManifest.Suites, func(workload set.Workload) bool { return workload.ID == id })
			if index < 0 {
				return nil, fmt.Errorf("soak suite %s is not in %s", id, selection.Manifest)
			}
			if len(only) != 0 && !slices.Contains(only, id) {
				continue
			}
			plans = append(plans, workloadPlan{setManifest: setManifest, workload: setManifest.Suites[index], workingDir: workingDir})
		}
	}
	for _, id := range only {
		if !slices.ContainsFunc(plans, func(plan workloadPlan) bool { return plan.workload.ID == id }) {
			return nil, fmt.Errorf("soak manifest does not select workload %s", id)
		}
	}
	if len(plans) == 0 {
		return nil, errors.New("soak run selects no workloads")
	}
	return plans, nil
}
