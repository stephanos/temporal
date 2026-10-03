package soak

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

const LedgerSchema = "gomad3.determinism-soak-ledger/v1"

const ledgerFile = "ledger.json"

const maximumLedgerBytes = 16 << 20

// Batch outcomes. Only a clean batch is a pass; overflow, target failure, and
// infrastructure failure are kept apart from divergence because none of them
// can support or refute a determinism claim.
const (
	OutcomeClean          = "clean"
	OutcomeDivergence     = "divergence"
	OutcomeOverflow       = "overflow"
	OutcomeTargetFailure  = "target_failure"
	OutcomeInfrastructure = "infrastructure"
)

// Cross-batch comparisons of a clean batch with its cohort baseline.
const (
	ComparisonEstablished = "baseline_established"
	ComparisonMatches     = "matches_baseline"
	ComparisonDiffers     = "differs_from_baseline"
)

// CohortKey is the unit whose batches must agree. A changed execution identity,
// including a changed toolchain build key, is a different cohort.
type CohortKey struct {
	Workload          string              `json:"workload"`
	Seed              record.Uint64String `json:"seed"`
	Platform          string              `json:"platform"`
	ExecutionIdentity record.SHA256       `json:"execution_identity"`
}

type Counts struct {
	Batches                uint64 `json:"batches"`
	Repetitions            uint64 `json:"repetitions"`
	Divergences            uint64 `json:"divergences"`
	Overflows              uint64 `json:"overflows"`
	TargetFailures         uint64 `json:"target_failures"`
	InfrastructureFailures uint64 `json:"infrastructure_failures"`
}

func (counts *Counts) add(outcome string, repetitions uint64) {
	counts.Batches++
	counts.Repetitions += repetitions
	switch outcome {
	case OutcomeDivergence:
		counts.Divergences++
	case OutcomeOverflow:
		counts.Overflows++
	case OutcomeTargetFailure:
		counts.TargetFailures++
	case OutcomeInfrastructure:
		counts.InfrastructureFailures++
	default:
	}
}

// Baseline is the evidence every later batch of the cohort is compared with.
// Its files are relative to the ledger directory and travel with the ledger.
type Baseline struct {
	EvidenceDigest record.SHA256 `json:"evidence_digest"`
	Evidence       string        `json:"evidence"`
	Trace          string        `json:"trace,omitempty"`
	Run            string        `json:"run"`
	Batch          uint64        `json:"batch"`
}

type Cohort struct {
	ID        string           `json:"id"`
	Key       CohortKey        `json:"key"`
	Toolchain record.Toolchain `json:"toolchain"`
	Baseline  *Baseline        `json:"baseline,omitempty"`
	// Counts accumulate across every retained run that executed this cohort.
	Counts Counts   `json:"counts"`
	Runs   []string `json:"runs"`
}

type LedgerRun struct {
	ID             string        `json:"id"`
	Platform       string        `json:"platform"`
	ManifestSHA256 record.SHA256 `json:"manifest_sha256"`
	Started        string        `json:"started"`
}

// Ledger is the cumulative record a scheduled run restores from the previous
// run and retains for the next.
type Ledger struct {
	Schema  string      `json:"schema"`
	Runs    []LedgerRun `json:"runs"`
	Cohorts []*Cohort   `json:"cohorts"`
}

func newLedger() *Ledger {
	return &Ledger{Schema: LedgerSchema, Runs: []LedgerRun{}, Cohorts: []*Cohort{}}
}

// LoadLedger reads the ledger in directory, or returns an empty ledger when
// the directory holds none.
func LoadLedger(directory string) (*Ledger, error) {
	path := filepath.Join(directory, ledgerFile)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return newLedger(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read soak ledger: %w", err)
	}
	if info.Size() > maximumLedgerBytes {
		return nil, fmt.Errorf("soak ledger exceeds %d bytes", maximumLedgerBytes)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read soak ledger: %w", err)
	}
	ledger := newLedger()
	if err := canonicaljson.StrictDecode(contents, ledger); err != nil {
		return nil, fmt.Errorf("decode soak ledger: %w", err)
	}
	if ledger.Schema != LedgerSchema {
		return nil, fmt.Errorf("soak ledger has unsupported schema %q", ledger.Schema)
	}
	for _, cohort := range ledger.Cohorts {
		if cohort == nil || cohort.ID != cohortID(cohort.Key) {
			return nil, errors.New("soak ledger cohort identity is invalid")
		}
		if cohort.Baseline != nil && (!localPath(cohort.Baseline.Evidence) || cohort.Baseline.Trace != "" && !localPath(cohort.Baseline.Trace)) {
			return nil, fmt.Errorf("soak ledger cohort %s baseline path is invalid", cohort.ID)
		}
	}
	return ledger, nil
}

func (ledger *Ledger) save(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(directory, ledgerFile), ledger)
}

func localPath(path string) bool {
	return path != "" && filepath.IsLocal(path)
}

func cohortID(key CohortKey) string {
	encoded, err := canonicaljson.CanonicalJSON(key)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:8])
}

func (ledger *Ledger) cohort(key CohortKey, toolchain record.Toolchain) *Cohort {
	id := cohortID(key)
	for _, cohort := range ledger.Cohorts {
		if cohort.ID == id {
			return cohort
		}
	}
	cohort := &Cohort{ID: id, Key: key, Toolchain: toolchain, Runs: []string{}}
	ledger.Cohorts = append(ledger.Cohorts, cohort)
	return cohort
}

// observation is one batch's contribution to its cohort.
type observation struct {
	run         string
	batch       uint64
	key         CohortKey
	toolchain   record.Toolchain
	outcome     string
	digest      record.SHA256
	repetitions uint64
}

// observe records a batch in its cohort. A clean batch is compared with the
// cohort baseline: the first clean batch establishes it, and a later clean
// batch with a different evidence digest is a divergence. It returns the
// batch's final outcome, the comparison of a clean batch, and the cohort.
func (ledger *Ledger) observe(batch observation) (outcome, comparison string, cohort *Cohort) {
	cohort = ledger.cohort(batch.key, batch.toolchain)
	if !slices.Contains(cohort.Runs, batch.run) {
		cohort.Runs = append(cohort.Runs, batch.run)
	}
	outcome = batch.outcome
	if outcome == OutcomeClean {
		switch {
		case cohort.Baseline == nil:
			comparison = ComparisonEstablished
			cohort.Baseline = &Baseline{EvidenceDigest: batch.digest, Run: batch.run, Batch: batch.batch}
		case cohort.Baseline.EvidenceDigest == batch.digest:
			comparison = ComparisonMatches
		default:
			comparison = ComparisonDiffers
			outcome = OutcomeDivergence
		}
	}
	cohort.Counts.add(outcome, batch.repetitions)
	return outcome, comparison, cohort
}

// executionIdentity hashes every evidence field that is an input of the
// execution rather than a result of it: the Runner and toolchain builds, the
// target binary, the I/O profile, environment, limits, mounts, and the choice
// and diagnostic profiles.
func executionIdentity(evidence runner.ExecutionEvidence) (record.SHA256, error) {
	type choiceIdentity struct {
		Profile              string              `json:"profile"`
		ImplementationSHA256 record.SHA256       `json:"implementation_sha256"`
		Limit                record.Uint64String `json:"limit"`
	}
	projection := struct {
		RunnerBuild          string                         `json:"runner_build"`
		Toolchain            record.Toolchain               `json:"toolchain"`
		Target               record.Target                  `json:"target"`
		IOProfile            any                            `json:"io_profile"`
		Environment          []record.Environment           `json:"environment"`
		Limits               runner.ExecutionLimitsEvidence `json:"limits"`
		ReadOnlyMountsSHA256 *record.SHA256                 `json:"read_only_mounts_sha256,omitempty"`
		Choices              *choiceIdentity                `json:"choices,omitempty"`
		DiagnosticProfile    string                         `json:"diagnostic_profile,omitempty"`
	}{
		RunnerBuild: evidence.RunnerBuild, Toolchain: evidence.Toolchain, Target: evidence.Target, IOProfile: evidence.IOProfile,
		Environment: evidence.Environment, Limits: evidence.Limits, ReadOnlyMountsSHA256: evidence.ReadOnlyMountsSHA256,
	}
	if evidence.Choices != nil {
		projection.Choices = &choiceIdentity{Profile: evidence.Choices.Profile, ImplementationSHA256: evidence.Choices.ImplementationSHA256, Limit: evidence.Choices.Limit}
	}
	if evidence.Diagnostics != nil {
		projection.DiagnosticProfile = evidence.Diagnostics.Profile
	}
	encoded, err := canonicaljson.CanonicalJSON(projection)
	if err != nil {
		return "", err
	}
	return record.HashBytes(encoded), nil
}
