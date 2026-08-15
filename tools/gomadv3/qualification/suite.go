package qualification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/target"
)

const SuiteManifestSchema = "gomadv3.qualification-set/v3"
const PriorSuiteManifestSchema = "gomadv3.qualification-set/v2"
const LegacySuiteManifestSchema = "gomadv3.qualification-set/v1"
const SuiteReportSchema = "gomadv3.qualification-set-report/v6"
const PreviousSuiteReportSchema = "gomadv3.qualification-set-report/v5"
const PreLinkedSuiteReportSchema = "gomadv3.qualification-set-report/v4"
const PreChoiceSuiteReportSchema = "gomadv3.qualification-set-report/v3"
const LegacySuiteReportSchema = "gomadv3.qualification-set-report/v2"

const maximumCommandOutputBytes = 64 << 20
const maximumManifestBytes = 1 << 20
const maximumSuiteReportBytes = 64 << 20

type SuiteManifest struct {
	Schema               string     `json:"schema"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	Module               string     `json:"module"`
	Seeds                []uint64   `json:"seeds"`
	Repeat               uint64     `json:"repeat"`
	RunTimeout           string     `json:"run_timeout"`
	OverallTimeout       string     `json:"overall_timeout"`
	TerminateGrace       string     `json:"terminate_grace"`
	OutputBytes          uint64     `json:"output_bytes"`
	WorldTransitionBytes uint64     `json:"world_transition_bytes"`
	Suites               []Workload `json:"suites"`
	Seed                 uint64     `json:"-"`
	legacy               bool
}

type Workload struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Tier                 uint64                `json:"tier"`
	Invariant            string                `json:"invariant"`
	Package              string                `json:"package"`
	Test                 string                `json:"test"`
	BuildTags            []string              `json:"build_tags,omitempty"`
	CapabilityMode       target.CapabilityMode `json:"capability_mode"`
	Environment          []string              `json:"environment,omitempty"`
	ReadOnlyMounts       []Mount               `json:"read_only_mounts,omitempty"`
	RequiredProbes       []string              `json:"required_probes,omitempty"`
	ChoiceBytes          uint64                `json:"choice_bytes"`
	ReplaySuccesses      bool                  `json:"replay_successes"`
	SuccessArtifactLimit uint64                `json:"success_artifact_limit"`
	SuccessBytesLimit    uint64                `json:"success_bytes_limit"`
	RunTimeout           string                `json:"run_timeout,omitempty"`
	OverallTimeout       string                `json:"overall_timeout,omitempty"`
	Expectation          WorkloadExpectation   `json:"expectation"`
}

type legacyManifest struct {
	Schema               string        `json:"schema"`
	Name                 string        `json:"name"`
	Seed                 uint64        `json:"seed"`
	Repeat               uint64        `json:"repeat"`
	RunTimeout           string        `json:"run_timeout"`
	OverallTimeout       string        `json:"overall_timeout"`
	TerminateGrace       string        `json:"terminate_grace"`
	OutputBytes          uint64        `json:"output_bytes"`
	WorldTransitionBytes uint64        `json:"world_transition_bytes"`
	Suites               []legacySuite `json:"suites"`
}

type legacySuite struct {
	Name           string              `json:"name"`
	Package        string              `json:"package"`
	Test           string              `json:"test"`
	BuildTags      []string            `json:"build_tags,omitempty"`
	Environment    []string            `json:"environment,omitempty"`
	ReadOnlyMounts []Mount             `json:"read_only_mounts,omitempty"`
	RequiredProbes []string            `json:"required_probes,omitempty"`
	Expectation    WorkloadExpectation `json:"expectation"`
}

type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type WorkloadExpectation struct {
	Classification string `json:"classification"`
	ImportPath     string `json:"import_path,omitempty"`
	Capability     string `json:"capability,omitempty"`
}

type SuiteReport struct {
	Schema               string                   `json:"schema"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	ManifestSHA256       evidence.SHA256          `json:"manifest_sha256"`
	Seeds                []evidence.Uint64String  `json:"seeds"`
	Module               ModuleIdentity           `json:"module"`
	Platform             PlatformIdentity         `json:"platform"`
	Toolchain            AnalysisToolchain        `json:"toolchain"`
	IOProfile            deterministicio.Contract `json:"io_profile"`
	Dimensions           EvidenceDimensions       `json:"dimensions"`
	ExpectationsMet      bool                     `json:"expectations_met"`
	Selected             uint64                   `json:"selected"`
	AnalysisCompleted    uint64                   `json:"analysis_completed"`
	Completed            uint64                   `json:"completed"`
	Supported            uint64                   `json:"supported"`
	Unsupported          uint64                   `json:"unsupported"`
	Failed               uint64                   `json:"failed"`
	InfrastructureErrors uint64                   `json:"infrastructure_errors"`
	Replayed             uint64                   `json:"replayed"`
	ReplayDiverged       uint64                   `json:"replay_diverged"`
	Cancelled            uint64                   `json:"cancelled"`
	TimedOut             uint64                   `json:"timed_out"`
	ElapsedNanos         evidence.Uint64String    `json:"elapsed_nanos"`
	ArtifactBytes        evidence.Uint64String    `json:"artifact_bytes"`
	TraceBytes           evidence.Uint64String    `json:"trace_bytes"`
	Suites               []WorkloadReport         `json:"workloads"`
}

type WorkloadReport struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Tier           uint64                `json:"tier"`
	Invariant      string                `json:"invariant"`
	Expected       WorkloadExpectation   `json:"expected"`
	ExpectationMet bool                  `json:"expectation_met"`
	Classification string                `json:"classification"`
	Analysis       *AnalysisReport       `json:"analysis,omitempty"`
	AnalysisError  string                `json:"analysis_error,omitempty"`
	Seeds          []SeedReport          `json:"seeds"`
	Blockers       []AnalysisBlocker     `json:"blockers"`
	Choice         ChoiceCoverage        `json:"choice"`
	CapabilityMode target.CapabilityMode `json:"capability_mode,omitempty"`
}

type PlatformIdentity struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

type EvidenceDimensions struct {
	PortableV3 bool `json:"portable_v3"`
	Analysis   bool `json:"analysis"`
	Replay     bool `json:"replay"`
	Choice     bool `json:"choice"`
}

type SeedReport struct {
	Seed              evidence.Uint64String `json:"seed"`
	Classification    string                `json:"classification"`
	EvidenceSHA256    evidence.SHA256       `json:"evidence_sha256,omitempty"`
	Replayed          bool                  `json:"replayed"`
	ReplayMatch       bool                  `json:"replay_match"`
	ReplayDivergence  string                `json:"replay_divergence,omitempty"`
	ChoiceReplayExact bool                  `json:"choice_replay_exact,omitempty"`
	ElapsedNanos      evidence.Uint64String `json:"elapsed_nanos"`
	ArtifactBytes     evidence.Uint64String `json:"artifact_bytes"`
	TraceBytes        evidence.Uint64String `json:"trace_bytes"`
	Choice            ChoiceCoverage        `json:"choice"`
}

type ChoiceCoverage struct {
	Available              bool                  `json:"available"`
	Profile                string                `json:"profile,omitempty"`
	ImplementationSHA256   evidence.SHA256       `json:"implementation_sha256,omitempty"`
	Limit                  evidence.Uint64String `json:"limit,omitempty"`
	TapeSHA256             evidence.SHA256       `json:"tape_sha256,omitempty"`
	Decisions              evidence.Uint64String `json:"decisions,omitempty"`
	ExactReplayAvailable   bool                  `json:"exact_replay_available,omitempty"`
	Features               []choice.Feature      `json:"features"`
	AdjacentPairsObserved  evidence.Uint64String `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool                  `json:"adjacent_pairs_truncated"`
}

type SuiteSpec struct {
	ManifestPath string
	GomadPath    string
	WorkingDir   string
	ArtifactRoot string
	OutputPath   string
	Execute      SuiteExecuteFunc
}

type SuiteCommand struct {
	Executable   string
	Args         []string
	Dir          string
	ArtifactRoot string
	Timeout      time.Duration
	Grace        time.Duration
}

type SuiteCommandResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Err      error
}

type SuiteExecuteFunc func(context.Context, SuiteCommand) SuiteCommandResult

type SuiteExpectationError struct {
	Suites []string
}

func (err *SuiteExpectationError) Error() string {
	return "qualification set did not match expectations: " + strings.Join(err.Suites, ", ")
}

type InvalidSuiteReportError struct {
	Err error
}

func (err *InvalidSuiteReportError) Error() string {
	return err.Err.Error()
}

func (err *InvalidSuiteReportError) Unwrap() error {
	return err.Err
}

func IsInvalidSuiteReport(err error) bool {
	var invalid *InvalidSuiteReportError
	return errors.As(err, &invalid)
}

func invalidReport(err error) error {
	return &InvalidSuiteReportError{Err: err}
}

var setNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var testNamePattern = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)

func LoadSuiteManifest(path string) (SuiteManifest, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return SuiteManifest{}, fmt.Errorf("read qualification set manifest: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumManifestBytes {
		return SuiteManifest{}, errors.Join(fmt.Errorf("qualification set manifest must be between 1 and %d bytes", maximumManifestBytes), file.Close())
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumManifestBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return SuiteManifest{}, errors.Join(fmt.Errorf("read qualification set manifest: %w", readErr), closeErr)
	}
	if len(contents) > maximumManifestBytes {
		return SuiteManifest{}, fmt.Errorf("qualification set manifest exceeds %d bytes", maximumManifestBytes)
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(contents, &header); err != nil {
		return SuiteManifest{}, fmt.Errorf("decode qualification set manifest schema: %w", err)
	}
	var manifest SuiteManifest
	switch header.Schema {
	case SuiteManifestSchema:
		if err := evidence.StrictDecode(contents, &manifest); err != nil {
			return SuiteManifest{}, fmt.Errorf("decode qualification set manifest: %w", err)
		}
	case PriorSuiteManifestSchema:
		if err := evidence.StrictDecode(contents, &manifest); err != nil {
			return SuiteManifest{}, fmt.Errorf("decode previous qualification set manifest: %w", err)
		}
		manifest.Schema = SuiteManifestSchema
		for index := range manifest.Suites {
			manifest.Suites[index].CapabilityMode = target.CapabilityModeClosure
		}
	case LegacySuiteManifestSchema:
		var legacy legacyManifest
		if err := evidence.StrictDecode(contents, &legacy); err != nil {
			return SuiteManifest{}, fmt.Errorf("decode legacy qualification set manifest: %w", err)
		}
		manifest = normalizeLegacyManifest(legacy)
	default:
		return SuiteManifest{}, fmt.Errorf("unsupported qualification set manifest schema %q", header.Schema)
	}
	if len(manifest.Seeds) != 0 {
		manifest.Seed = manifest.Seeds[0]
	}
	if err := validateManifest(manifest); err != nil {
		return SuiteManifest{}, err
	}
	return manifest, nil
}

func normalizeLegacyManifest(legacy legacyManifest) SuiteManifest {
	manifest := SuiteManifest{
		Schema: SuiteManifestSchema, Name: legacy.Name, Seeds: []uint64{legacy.Seed}, Seed: legacy.Seed,
		Repeat: legacy.Repeat, RunTimeout: legacy.RunTimeout, OverallTimeout: legacy.OverallTimeout,
		TerminateGrace: legacy.TerminateGrace, OutputBytes: legacy.OutputBytes,
		WorldTransitionBytes: legacy.WorldTransitionBytes, Suites: make([]Workload, len(legacy.Suites)), legacy: true,
	}
	for index, suite := range legacy.Suites {
		manifest.Suites[index] = Workload{
			ID: suite.Name, Name: suite.Name, Tier: 1, Package: suite.Package, Test: suite.Test,
			BuildTags: append([]string(nil), suite.BuildTags...), Environment: append([]string(nil), suite.Environment...),
			ReadOnlyMounts: append([]Mount(nil), suite.ReadOnlyMounts...), RequiredProbes: append([]string(nil), suite.RequiredProbes...),
			Expectation: suite.Expectation, CapabilityMode: target.CapabilityModeClosure,
		}
	}
	return manifest
}

func RunSuite(ctx context.Context, config SuiteSpec) (SuiteReport, error) {
	manifest, err := LoadSuiteManifest(config.ManifestPath)
	if err != nil {
		return SuiteReport{}, err
	}
	if config.GomadPath == "" || config.WorkingDir == "" || config.ArtifactRoot == "" || config.OutputPath == "" {
		return SuiteReport{}, errors.New("qualification set requires gomad, working, artifact, and output paths")
	}
	for _, field := range []*string{&config.GomadPath, &config.WorkingDir, &config.ArtifactRoot, &config.OutputPath} {
		absolute, err := filepath.Abs(*field)
		if err != nil {
			return SuiteReport{}, err
		}
		*field = absolute
	}
	if config.Execute == nil {
		info, err := os.Stat(config.GomadPath)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return SuiteReport{}, errors.Join(errors.New("gomad executable is invalid"), err)
		}
		config.Execute = executeCommand
	}
	moduleIdentity, err := identifyModule(config.WorkingDir, manifest.Module)
	if err != nil {
		return SuiteReport{}, err
	}
	manifestBytes, err := evidence.CanonicalJSON(manifest)
	if err != nil {
		return SuiteReport{}, err
	}
	report := SuiteReport{
		Schema: SuiteReportSchema, Name: manifest.Name, Description: manifest.Description,
		ManifestSHA256: evidence.HashBytes(manifestBytes), Module: moduleIdentity,
		Dimensions: EvidenceDimensions{PortableV3: !manifest.legacy, Analysis: true, Replay: !manifest.legacy, Choice: !manifest.legacy},
		Selected:   uint64(len(manifest.Suites)), Seeds: make([]evidence.Uint64String, len(manifest.Seeds)),
		Suites: make([]WorkloadReport, len(manifest.Suites)),
	}
	for index, seed := range manifest.Seeds {
		report.Seeds[index] = evidence.Uint64String(seed)
	}
	for index, suite := range manifest.Suites {
		report.Suites[index] = WorkloadReport{
			ID: suite.ID, Name: suite.Name, Tier: suite.Tier, Invariant: suite.Invariant,
			Expected: suite.Expectation, Seeds: []SeedReport{}, Blockers: []AnalysisBlocker{},
			Choice: emptyChoiceCoverage(), CapabilityMode: suite.CapabilityMode,
		}
	}
	failed := make([]string, 0, len(manifest.Suites))
	analysisFailed := false
	for index, suite := range manifest.Suites {
		suiteReport := report.Suites[index]
		if err := ctx.Err(); err != nil {
			suiteReport.AnalysisError = contextClassification(err)
			suiteReport.Classification = suiteReport.AnalysisError
			report.Suites[index] = suiteReport
			failed = append(failed, suite.ID)
			analysisFailed = true
			break
		}
		command := analysisCommand(config, manifest, suite)
		result := config.Execute(ctx, command)
		analysis, classification, analysisErr := retainedAnalysis(result)
		if analysisErr != nil {
			suiteReport.AnalysisError = classification
			suiteReport.Classification = classification
			analysisFailed = true
			failed = append(failed, suite.ID)
		} else {
			suiteReport.Analysis = &analysis
			suiteReport.Blockers = append([]AnalysisBlocker{}, analysis.Blockers...)
			report.AnalysisCompleted++
			if err := mergeAnalysisIdentity(&report, analysis); err != nil {
				suiteReport.AnalysisError = "runner_failure"
				suiteReport.Classification = "runner_failure"
				analysisFailed = true
				failed = append(failed, suite.ID)
			} else if analysis.Classification == ClassificationUnsupported {
				suiteReport.Classification = "unsupported_target"
				suiteReport.ExpectationMet = matchesUnsupportedAnalysis(suite.Expectation, analysis)
				report.Completed++
				report.Unsupported++
				if !suiteReport.ExpectationMet {
					failed = append(failed, suite.ID)
				}
			}
		}
		report.Suites[index] = suiteReport
		if checkpointErr := writeCheckpoint(context.WithoutCancel(ctx), config.OutputPath+".partial", "analysis", suite.ID, report); checkpointErr != nil {
			return report, checkpointErr
		}
	}
	if analysisFailed {
		classification := contextClassification(ctx.Err())
		for index := range report.Suites {
			if report.Suites[index].Classification != "" {
				continue
			}
			report.Suites[index].Classification = classification
			if report.Suites[index].Analysis == nil {
				report.Suites[index].AnalysisError = classification
			}
			failed = append(failed, report.Suites[index].ID)
		}
	}
	if !analysisFailed {
		for index, suite := range manifest.Suites {
			if report.Suites[index].Analysis == nil || report.Suites[index].Analysis.Classification == ClassificationUnsupported {
				continue
			}
			allQualified := true
			for _, seed := range manifest.Seeds {
				if err := ctx.Err(); err != nil {
					classification := contextClassification(err)
					report.Suites[index].Classification = classification
					failed = append(failed, suite.ID)
					allQualified = false
					break
				}
				command := suiteCommand(config, manifest, suite, seed)
				result := config.Execute(ctx, command)
				opened, classification, _, _, evidenceErr := retainedQualification(config.ArtifactRoot, command, result)
				seedReport := SeedReport{Seed: evidence.Uint64String(seed), Classification: classification, Choice: emptyChoiceCoverage()}
				if evidenceErr == nil {
					projected, projectErr := projectSeedReport(opened, classification, seed, suite, *report.Suites[index].Analysis)
					if projectErr == nil {
						seedReport = projected
					}
					evidenceErr = projectErr
				}
				if evidenceErr != nil {
					classification = retainedErrorClassification(result, evidenceErr)
					seedReport.Classification = classification
				}
				report.Suites[index].Seeds = append(report.Suites[index].Seeds, seedReport)
				addSeedTotals(&report, seedReport)
				if classification != "qualified" {
					allQualified = false
					if report.Suites[index].Classification == "" {
						report.Suites[index].Classification = classification
					}
				}
				if checkpointErr := writeCheckpoint(context.WithoutCancel(ctx), config.OutputPath+".partial", "qualification", suite.ID+"/"+strconv.FormatUint(seed, 10), report); checkpointErr != nil {
					return report, checkpointErr
				}
			}
			coverage, coverageErr := aggregateChoiceCoverage(report.Suites[index].Seeds)
			if coverageErr != nil {
				report.Suites[index].Classification = "runner_failure"
				allQualified = false
			}
			if allQualified && len(report.Suites[index].Seeds) == len(manifest.Seeds) {
				report.Suites[index].Classification = "qualified"
				report.Supported++
			} else {
				switch classificationBucket(report.Suites[index].Classification) {
				case qualificationFailed:
					report.Failed++
				default:
					report.InfrastructureErrors++
				}
			}
			report.Completed++
			report.Suites[index].Choice = coverage
			report.Suites[index].ExpectationMet = matchesSupportedExpectation(suite.Expectation, report.Suites[index])
			if !report.Suites[index].ExpectationMet {
				failed = append(failed, suite.ID)
			}
		}
	}
	finalizeSetReportCounters(&report)
	failed = slices.Compact(failed)
	report.ExpectationsMet = report.Completed == report.Selected && len(failed) == 0
	if err := writeReport(ctx, config.OutputPath, report); err != nil {
		return report, err
	}
	if err := removeCheckpoint(config.OutputPath + ".partial"); err != nil {
		return report, err
	}
	if !report.ExpectationsMet {
		return report, &SuiteExpectationError{Suites: failed}
	}
	return report, nil
}

func OpenSuiteReport(path string) (SuiteReport, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return SuiteReport{}, fmt.Errorf("open qualification set report: %w", err)
		}
		return SuiteReport{}, invalidReport(fmt.Errorf("open qualification set report: %w", err))
	}
	if info.Mode().Perm() != 0o600 || info.Size() <= 0 || info.Size() > maximumSuiteReportBytes {
		return SuiteReport{}, invalidReport(errors.Join(errors.New("qualification set report must be a private bounded regular file"), file.Close()))
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumSuiteReportBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return SuiteReport{}, errors.Join(fmt.Errorf("read qualification set report: %w", readErr), closeErr)
	}
	if len(contents) > maximumSuiteReportBytes || len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return SuiteReport{}, invalidReport(errors.New("qualification set report framing is invalid"))
	}
	contents = contents[:len(contents)-1]
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(contents, &header); err != nil {
		return SuiteReport{}, invalidReport(fmt.Errorf("decode qualification set report schema: %w", err))
	}
	if header.Schema == LegacySuiteReportSchema {
		report, err := decodeLegacySetReport(contents)
		if err != nil {
			return SuiteReport{}, invalidReport(err)
		}
		return report, nil
	}
	if header.Schema == PreviousSuiteReportSchema || header.Schema == PreLinkedSuiteReportSchema || header.Schema == PreChoiceSuiteReportSchema {
		report, err := decodePreviousSetReport(contents, header.Schema)
		if err != nil {
			return SuiteReport{}, invalidReport(err)
		}
		return report, nil
	}
	if header.Schema != SuiteReportSchema {
		return SuiteReport{}, invalidReport(fmt.Errorf("unsupported qualification set report schema %q", header.Schema))
	}
	var report SuiteReport
	if err := evidence.DecodeCanonicalJSON(contents, &report); err != nil {
		return SuiteReport{}, invalidReport(fmt.Errorf("decode qualification set report: %w", err))
	}
	if err := validateSetReport(report); err != nil {
		return SuiteReport{}, invalidReport(err)
	}
	return report, nil
}

func validateManifest(manifest SuiteManifest) error {
	if manifest.Schema != SuiteManifestSchema || !setNamePattern.MatchString(manifest.Name) || manifest.Repeat < 2 || manifest.Repeat > 32 || len(manifest.Seeds) == 0 || len(manifest.Seeds) > 32 || len(manifest.Suites) == 0 || len(manifest.Suites) > 64 || manifest.OutputBytes == 0 || manifest.WorldTransitionBytes == 0 {
		return errors.New("qualification set manifest identity or bounds are invalid")
	}
	if !manifest.legacy && (strings.TrimSpace(manifest.Description) == "" || strings.TrimSpace(manifest.Module) == "" || strings.ContainsAny(manifest.Module, "\x00\n\r\t ")) {
		return errors.New("qualification set portable identity is invalid")
	}
	for index, seed := range manifest.Seeds {
		if index > 0 && seed <= manifest.Seeds[index-1] {
			return errors.New("qualification set seeds must be sorted and unique")
		}
	}
	runTimeout, err := time.ParseDuration(manifest.RunTimeout)
	if err != nil || runTimeout <= 0 {
		return errors.Join(errors.New("qualification set run timeout is invalid"), err)
	}
	overallTimeout, err := time.ParseDuration(manifest.OverallTimeout)
	if err != nil || overallTimeout <= 0 {
		return errors.Join(errors.New("qualification set overall timeout is invalid"), err)
	}
	grace, err := time.ParseDuration(manifest.TerminateGrace)
	if err != nil || grace < 0 || grace > runTimeout || grace > overallTimeout {
		return errors.Join(errors.New("qualification set termination grace is invalid"), err)
	}
	seen := make(map[string]struct{}, len(manifest.Suites))
	for index, suite := range manifest.Suites {
		packagePath := strings.TrimPrefix(suite.Package, "./")
		if !setNamePattern.MatchString(suite.ID) || strings.TrimSpace(suite.Name) == "" || !strings.HasPrefix(suite.Package, "./") || packagePath == "" || filepath.ToSlash(filepath.Clean(filepath.FromSlash(packagePath))) != packagePath || strings.HasPrefix(packagePath, "../") || !testNamePattern.MatchString(suite.Test) {
			return fmt.Errorf("qualification suite %d identity is invalid", index)
		}
		if !manifest.legacy && (suite.Tier != 1 && suite.Tier != 2 || strings.TrimSpace(suite.Invariant) == "") {
			return fmt.Errorf("qualification suite %s tier or invariant is invalid", suite.ID)
		}
		if index > 0 && suite.ID <= manifest.Suites[index-1].ID {
			return errors.New("qualification suite identities must be sorted and unique")
		}
		if _, duplicate := seen[suite.ID]; duplicate {
			return fmt.Errorf("qualification suite identity is duplicated: %s", suite.ID)
		}
		seen[suite.ID] = struct{}{}
		if !sortedUnique(suite.BuildTags) || !sortedUnique(suite.Environment) || !sortedUnique(suite.RequiredProbes) {
			return fmt.Errorf("qualification suite %s lists must be sorted and unique", suite.ID)
		}
		if suite.CapabilityMode != target.CapabilityModeClosure && suite.CapabilityMode != target.CapabilityModeLinked {
			return fmt.Errorf("qualification suite %s capability mode is invalid", suite.ID)
		}
		for _, mount := range suite.ReadOnlyMounts {
			if mount.Source == "" || mount.Target == "" || strings.ContainsAny(mount.Source+mount.Target, "\x00\n") {
				return fmt.Errorf("qualification suite %s has an invalid read-only mount", suite.ID)
			}
		}
		if suite.ReplaySuccesses != (suite.SuccessArtifactLimit != 0 && suite.SuccessBytesLimit != 0) || suite.ReplaySuccesses && suite.ChoiceBytes == 0 || !suite.ReplaySuccesses && (suite.SuccessArtifactLimit != 0 || suite.SuccessBytesLimit != 0) {
			return fmt.Errorf("qualification suite %s replay bounds are invalid", suite.ID)
		}
		for _, override := range []string{suite.RunTimeout, suite.OverallTimeout} {
			if override != "" {
				parsed, parseErr := time.ParseDuration(override)
				if parseErr != nil || parsed <= 0 {
					return errors.Join(fmt.Errorf("qualification suite %s timeout override is invalid", suite.ID), parseErr)
				}
			}
		}
		if err := validateExpectation(suite.Expectation); err != nil {
			return fmt.Errorf("qualification suite %s: %w", suite.ID, err)
		}
	}
	return nil
}

func validateExpectation(expectation WorkloadExpectation) error {
	switch expectation.Classification {
	case "qualified":
		if expectation.ImportPath != "" || expectation.Capability != "" {
			return errors.New("qualified expectation cannot include an unsupported boundary")
		}
	case "unsupported_target":
		if expectation.ImportPath == "" || expectation.Capability == "" {
			return errors.New("unsupported expectation requires exact import and capability")
		}
	case "target_failure", "nondeterministic", "replay_divergence":
		if expectation.ImportPath != "" || expectation.Capability != "" {
			return errors.New("failure expectation cannot include an unsupported boundary")
		}
	default:
		return fmt.Errorf("unknown qualification expectation %q", expectation.Classification)
	}
	return nil
}

func suiteCommand(config SuiteSpec, manifest SuiteManifest, suite Workload, seed uint64) SuiteCommand {
	runTimeout, overallTimeout := suiteTimeouts(manifest, suite)
	terminateGrace := manifestGrace(manifest)
	args := []string{
		"qualify", "--json", "--seed=" + strconv.FormatUint(seed, 10), "--repeat=" + strconv.FormatUint(manifest.Repeat, 10),
		"--run-timeout=" + runTimeout.String(), "--overall-timeout=" + overallTimeout.String(), "--terminate-grace=" + manifest.TerminateGrace,
		"--output-limit=" + strconv.FormatUint(manifest.OutputBytes, 10), "--world-transition-limit=" + strconv.FormatUint(manifest.WorldTransitionBytes, 10),
		"--artifacts=" + config.ArtifactRoot,
		"--capability-mode=" + string(suite.CapabilityMode),
	}
	if suite.ChoiceBytes != 0 {
		args = append(args, "--choices", "--choice-bytes="+strconv.FormatUint(suite.ChoiceBytes, 10))
	}
	if suite.ReplaySuccesses {
		args = append(args, "--replay-successes", "--success-limit="+strconv.FormatUint(suite.SuccessArtifactLimit, 10), "--success-bytes="+strconv.FormatUint(suite.SuccessBytesLimit, 10))
	}
	for _, value := range suite.BuildTags {
		args = append(args, "--build-tag="+value)
	}
	for _, value := range suite.Environment {
		args = append(args, "--env="+value)
	}
	for _, mount := range suite.ReadOnlyMounts {
		args = append(args, "--io-ro-mount="+mount.Source+"="+mount.Target)
	}
	for _, value := range suite.RequiredProbes {
		args = append(args, "--require-probe="+value)
	}
	args = append(args, "go-test", suite.Package, "--", "-test.run=^"+regexp.QuoteMeta(suite.Test)+"$")
	return SuiteCommand{
		Executable: config.GomadPath, Args: args, Dir: config.WorkingDir, ArtifactRoot: config.ArtifactRoot,
		Timeout: overallTimeout + terminateGrace + 10*time.Second, Grace: terminateGrace,
	}
}

func executeCommand(ctx context.Context, command SuiteCommand) SuiteCommandResult {
	executed, err := hostexec.Run(ctx, hostexec.Request{
		Command: append([]string{command.Executable}, command.Args...), Dir: command.Dir, Env: os.Environ(),
		Timeout: command.Timeout, TerminateGrace: command.Grace, OutputLimit: maximumCommandOutputBytes,
	})
	result := SuiteCommandResult{ExitCode: executed.ExitCode, Stdout: executed.Stdout.Bytes, Stderr: executed.Stderr.Bytes, Err: err}
	if executed.Termination == hostexec.TerminationSignal {
		result.ExitCode = -1
	}
	if executed.WatchdogTimeout {
		result.Err = errors.Join(result.Err, context.DeadlineExceeded)
	}
	if executed.Cancelled {
		result.Err = errors.Join(result.Err, context.Canceled)
	}
	if executed.Stdout.Truncated || executed.Stderr.Truncated {
		result.Err = errors.Join(result.Err, errors.New("qualification command output exceeded its bound"))
	}
	return result
}

func retainedQualification(artifactRoot string, command SuiteCommand, result SuiteCommandResult) (QualificationReport, string, string, evidence.SHA256, error) {
	if result.Err != nil {
		return QualificationReport{}, "", "", "", result.Err
	}
	if len(result.Stderr) != 0 {
		return QualificationReport{}, "", "", "", errors.New("JSON qualification command wrote to stderr")
	}
	event, err := DecodeResultEvent(result.Stdout)
	if err != nil {
		return QualificationReport{}, "", "", "", err
	}
	path, err := filepath.Abs(event.ReportPath)
	if err != nil {
		return QualificationReport{}, "", "", "", err
	}
	relative, err := filepath.Rel(artifactRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return QualificationReport{}, "", "", "", errors.Join(errors.New("qualification report is outside the artifact root"), err)
	}
	opened, err := OpenQualificationReport(path)
	if err != nil {
		return QualificationReport{}, "", "", "", err
	}
	logicalCommand := append([]string{"gomad"}, command.Args...)
	if !slices.Equal(opened.Command, logicalCommand) {
		return QualificationReport{}, "", "", "", errors.New("retained qualification command does not match the executed command")
	}
	classification := ClassifyQualification(opened)
	if event.Classification != classification || result.ExitCode != ExitStatus(classification) {
		return QualificationReport{}, "", "", "", fmt.Errorf("qualification result classification or status is inconsistent: %s/%d", event.Classification, result.ExitCode)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return QualificationReport{}, "", "", "", err
	}
	return opened, classification, path, evidence.HashBytes(contents), nil
}

func matchesExpectation(expected WorkloadExpectation, classification string, report QualificationReport, exitCode int) bool {
	if expected.Classification != classification || exitCode != ExitStatus(classification) {
		return false
	}
	switch classification {
	case "qualified":
		return report.Qualified && report.Deterministic && report.TargetSuccess && report.Failure == nil
	case "unsupported_target":
		return report.Failure != nil && report.Failure.ImportPath == expected.ImportPath && report.Failure.Capability == expected.Capability
	case "target_failure":
		replay := firstReplay(report)
		return report.Deterministic && !report.TargetSuccess && replay != nil && replay.Attempted && replay.Match
	case "nondeterministic":
		return !report.Deterministic
	case "replay_divergence":
		replay := firstReplay(report)
		return replay != nil && replay.Attempted && !replay.Match
	default:
		return false
	}
}

func firstReplay(report QualificationReport) *QualificationReplay {
	for _, run := range report.Runs {
		if run.Replay != nil {
			return run.Replay
		}
	}
	return nil
}

type qualificationBucket uint8

const (
	qualificationSupported qualificationBucket = iota
	qualificationUnsupported
	qualificationFailed
	qualificationInfrastructure
)

func classificationBucket(classification string) qualificationBucket {
	switch classification {
	case "qualified":
		return qualificationSupported
	case "unsupported_target":
		return qualificationUnsupported
	case "target_failure", "nondeterministic", "replay_divergence", "semantic_coverage_failure":
		return qualificationFailed
	default:
		return qualificationInfrastructure
	}
}

type checkpoint struct {
	Schema string      `json:"schema"`
	Phase  string      `json:"phase"`
	Key    string      `json:"key"`
	Report SuiteReport `json:"report"`
}

func writeCheckpoint(ctx context.Context, path, phase, key string, report SuiteReport) error {
	contents, err := evidence.CanonicalJSON(checkpoint{Schema: "gomadv3.qualification-set-checkpoint/v1", Phase: phase, Key: key, Report: report})
	if err != nil {
		return err
	}
	if len(contents) > maximumSuiteReportBytes {
		return fmt.Errorf("qualification set checkpoint exceeds %d bytes", maximumSuiteReportBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return hostfs.ReplaceContext(ctx, path, append(contents, '\n'), 0o600)
}

func writeReport(ctx context.Context, path string, report SuiteReport) error {
	if err := validateSetReport(report); err != nil {
		return err
	}
	contents, err := evidence.CanonicalJSON(report)
	if err != nil {
		return err
	}
	if len(contents) > maximumSuiteReportBytes {
		return fmt.Errorf("qualification set report exceeds %d bytes", maximumSuiteReportBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return hostfs.ReplaceContext(ctx, path, append(contents, '\n'), 0o600)
}

func removeCheckpoint(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove qualification checkpoint: %w", err)
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

type setReportCounters struct {
	analysisCompleted, completed, supported, unsupported, failed, infrastructure uint64
	replayed, replayDiverged, cancelled, timedOut                                uint64
	elapsedNanos, artifactBytes, traceBytes                                      evidence.Uint64String
}

func deriveSetReportCounters(report SuiteReport) setReportCounters {
	var result setReportCounters
	for _, suite := range report.Suites {
		if suite.Analysis != nil {
			result.analysisCompleted++
			if suite.Analysis.Classification == ClassificationUnsupported || len(suite.Seeds) == len(report.Seeds) {
				result.completed++
			}
		}
		switch classificationBucket(suite.Classification) {
		case qualificationSupported:
			result.supported++
		case qualificationUnsupported:
			result.unsupported++
		case qualificationFailed:
			result.failed++
		default:
			result.infrastructure++
		}
		if suite.Classification == "cancelled" {
			result.cancelled++
		}
		if suite.Classification == "overall_timeout" {
			result.timedOut++
		}
		for _, seed := range suite.Seeds {
			result.elapsedNanos += seed.ElapsedNanos
			result.artifactBytes += seed.ArtifactBytes
			result.traceBytes += seed.TraceBytes
			if seed.Replayed {
				result.replayed++
			}
			if seed.Replayed && !seed.ReplayMatch {
				result.replayDiverged++
			}
		}
	}
	return result
}

func finalizeSetReportCounters(report *SuiteReport) {
	counters := deriveSetReportCounters(*report)
	report.AnalysisCompleted = counters.analysisCompleted
	report.Completed = counters.completed
	report.Supported = counters.supported
	report.Unsupported = counters.unsupported
	report.Failed = counters.failed
	report.InfrastructureErrors = counters.infrastructure
	report.Replayed = counters.replayed
	report.ReplayDiverged = counters.replayDiverged
	report.Cancelled = counters.cancelled
	report.TimedOut = counters.timedOut
	report.ElapsedNanos = counters.elapsedNanos
	report.ArtifactBytes = counters.artifactBytes
	report.TraceBytes = counters.traceBytes
}

func validateSetReport(report SuiteReport) error {
	if report.Schema != SuiteReportSchema || !setNamePattern.MatchString(report.Name) || report.ManifestSHA256 == "" || len(report.Seeds) == 0 || report.Selected != uint64(len(report.Suites)) || report.Completed > report.Selected || report.AnalysisCompleted > report.Selected {
		return errors.New("qualification set report identity or counts are invalid")
	}
	for index, seed := range report.Seeds {
		if index > 0 && seed <= report.Seeds[index-1] {
			return errors.New("qualification set report seeds must be sorted and unique")
		}
	}
	if _, err := report.ManifestSHA256.Bytes(); err != nil {
		return fmt.Errorf("qualification set manifest digest is invalid: %w", err)
	}
	if report.Dimensions.PortableV3 {
		if report.Module.Path == "" || report.Module.GoModSHA256 == "" {
			return errors.New("qualification set module identity is incomplete")
		}
		if _, err := report.Module.GoModSHA256.Bytes(); err != nil {
			return fmt.Errorf("qualification set module digest is invalid: %w", err)
		}
	}
	if report.Dimensions.PortableV3 && (report.Platform.GOOS == "" || report.Platform.GOARCH == "" || report.Toolchain.BuildKey == "" || report.IOProfile.Name == "") {
		return errors.New("qualification set implementation identity is incomplete")
	}
	if report.Supported+report.Unsupported+report.Failed+report.InfrastructureErrors != report.Selected {
		return errors.New("qualification set result counts are inconsistent")
	}
	expectationsMet := report.Completed == report.Selected
	for index, suite := range report.Suites {
		if !setNamePattern.MatchString(suite.ID) || suite.Name == "" || suite.Tier != 1 && suite.Tier != 2 || report.Dimensions.PortableV3 && suite.Invariant == "" || suite.Seeds == nil || suite.Blockers == nil || suite.Choice.Features == nil || suite.CapabilityMode != target.CapabilityModeClosure && suite.CapabilityMode != target.CapabilityModeLinked || !validSetClassification(suite.Classification) || index > 0 && suite.ID <= report.Suites[index-1].ID {
			return fmt.Errorf("qualification set workload %d identity is invalid", index)
		}
		if report.Dimensions.Analysis && (suite.Analysis == nil && suite.AnalysisError == "" || suite.Analysis != nil && suite.AnalysisError != "") {
			return fmt.Errorf("qualification set workload %s analysis state is invalid", suite.ID)
		}
		if !report.Dimensions.Analysis && (suite.Analysis != nil || suite.AnalysisError != "dimension_unavailable") {
			return fmt.Errorf("qualification set workload %s legacy analysis state is invalid", suite.ID)
		}
		if suite.Analysis != nil {
			encoded, err := evidence.CanonicalJSON(*suite.Analysis)
			if err != nil {
				return err
			}
			if _, err := DecodeAnalysisReport(encoded); err != nil {
				return fmt.Errorf("qualification set workload %s analysis is invalid: %w", suite.ID, err)
			}
			if suite.Analysis.Target.CapabilityMode != suite.CapabilityMode {
				return fmt.Errorf("qualification set workload %s capability mode does not match its analysis", suite.ID)
			}
			if suite.Analysis.Classification == ClassificationUnsupported && (suite.Classification != "unsupported_target" || len(suite.Seeds) != 0) {
				return fmt.Errorf("qualification set workload %s unsupported analysis is inconsistent", suite.ID)
			}
			if suite.Analysis.Classification == ClassificationSupported && suite.Classification == "unsupported_target" {
				return fmt.Errorf("qualification set workload %s supported analysis is inconsistent", suite.ID)
			}
		}
		for seedIndex, seed := range suite.Seeds {
			if seed.Choice.Features == nil || seed.ChoiceReplayExact && (!seed.Replayed || !seed.ReplayMatch || !seed.Choice.ExactReplayAvailable || seed.Choice.TapeSHA256 == "") || !validSetClassification(seed.Classification) || seedIndex >= len(report.Seeds) || seed.Seed != report.Seeds[seedIndex] {
				return fmt.Errorf("qualification set workload %s seed evidence is invalid", suite.ID)
			}
		}
		if suite.Choice.ExactReplayAvailable && (!suite.Choice.Available || suite.Choice.Profile != choice.Profile) {
			return fmt.Errorf("qualification set workload %s exact choice identity is invalid", suite.ID)
		}
		if suite.Classification == "qualified" && len(suite.Seeds) != len(report.Seeds) {
			return fmt.Errorf("qualification set workload %s is missing qualified seed evidence", suite.ID)
		}
		expectationsMet = expectationsMet && suite.ExpectationMet
	}
	counters := deriveSetReportCounters(report)
	if report.AnalysisCompleted != counters.analysisCompleted || report.Completed != counters.completed || report.Supported != counters.supported || report.Unsupported != counters.unsupported || report.Failed != counters.failed || report.InfrastructureErrors != counters.infrastructure || report.Replayed != counters.replayed || report.ReplayDiverged != counters.replayDiverged || report.Cancelled != counters.cancelled || report.TimedOut != counters.timedOut || report.ElapsedNanos != counters.elapsedNanos || report.ArtifactBytes != counters.artifactBytes || report.TraceBytes != counters.traceBytes {
		return errors.New("qualification set aggregate counters do not match workload evidence")
	}
	if report.ExpectationsMet != expectationsMet {
		return errors.New("qualification set expectation result is inconsistent")
	}
	return nil
}

func validSetClassification(classification string) bool {
	switch classification {
	case "qualified", "unsupported_target", "target_failure", "nondeterministic", "replay_divergence", "semantic_coverage_failure", "invalid_input", "runner_failure", "watchdog", "overall_timeout", "cancelled":
		return true
	default:
		return false
	}
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}
