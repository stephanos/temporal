package qualificationset

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

	"go.temporal.io/server/tools/gomadv3/internal/capabilityanalysis"
	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/safefile"
)

const ManifestSchema = "gomadv3.qualification-set/v2"
const LegacyManifestSchema = "gomadv3.qualification-set/v1"
const ReportSchema = "gomadv3.qualification-set-report/v4"
const PreviousReportSchema = "gomadv3.qualification-set-report/v3"
const LegacyReportSchema = "gomadv3.qualification-set-report/v2"

const maximumCommandOutputBytes = 64 << 20
const maximumManifestBytes = 1 << 20
const maximumReportBytes = 64 << 20

type Manifest struct {
	Schema               string   `json:"schema"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Module               string   `json:"module"`
	Seeds                []uint64 `json:"seeds"`
	Repeat               uint64   `json:"repeat"`
	RunTimeout           string   `json:"run_timeout"`
	OverallTimeout       string   `json:"overall_timeout"`
	TerminateGrace       string   `json:"terminate_grace"`
	OutputBytes          uint64   `json:"output_bytes"`
	WorldTransitionBytes uint64   `json:"world_transition_bytes"`
	Suites               []Suite  `json:"suites"`
	Seed                 uint64   `json:"-"`
	legacy               bool
}

type Suite struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Tier                 uint64      `json:"tier"`
	Invariant            string      `json:"invariant"`
	Package              string      `json:"package"`
	Test                 string      `json:"test"`
	BuildTags            []string    `json:"build_tags,omitempty"`
	Environment          []string    `json:"environment,omitempty"`
	ReadOnlyMounts       []Mount     `json:"read_only_mounts,omitempty"`
	RequiredProbes       []string    `json:"required_probes,omitempty"`
	ChoiceBytes          uint64      `json:"choice_bytes"`
	ReplaySuccesses      bool        `json:"replay_successes"`
	SuccessArtifactLimit uint64      `json:"success_artifact_limit"`
	SuccessBytesLimit    uint64      `json:"success_bytes_limit"`
	RunTimeout           string      `json:"run_timeout,omitempty"`
	OverallTimeout       string      `json:"overall_timeout,omitempty"`
	Expectation          Expectation `json:"expectation"`
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
	Name           string      `json:"name"`
	Package        string      `json:"package"`
	Test           string      `json:"test"`
	BuildTags      []string    `json:"build_tags,omitempty"`
	Environment    []string    `json:"environment,omitempty"`
	ReadOnlyMounts []Mount     `json:"read_only_mounts,omitempty"`
	RequiredProbes []string    `json:"required_probes,omitempty"`
	Expectation    Expectation `json:"expectation"`
}

type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Expectation struct {
	Classification string `json:"classification"`
	ImportPath     string `json:"import_path,omitempty"`
	Capability     string `json:"capability,omitempty"`
}

type SetReport struct {
	Schema               string                       `json:"schema"`
	Name                 string                       `json:"name"`
	Description          string                       `json:"description"`
	ManifestSHA256       record.SHA256                `json:"manifest_sha256"`
	Seeds                []record.Uint64String        `json:"seeds"`
	Module               ModuleIdentity               `json:"module"`
	Platform             PlatformIdentity             `json:"platform"`
	Toolchain            capabilityanalysis.Toolchain `json:"toolchain"`
	IOProfile            ioprofile.Identity           `json:"io_profile"`
	Dimensions           EvidenceDimensions           `json:"dimensions"`
	ExpectationsMet      bool                         `json:"expectations_met"`
	Selected             uint64                       `json:"selected"`
	AnalysisCompleted    uint64                       `json:"analysis_completed"`
	Completed            uint64                       `json:"completed"`
	Supported            uint64                       `json:"supported"`
	Unsupported          uint64                       `json:"unsupported"`
	Failed               uint64                       `json:"failed"`
	InfrastructureErrors uint64                       `json:"infrastructure_errors"`
	Replayed             uint64                       `json:"replayed"`
	ReplayDiverged       uint64                       `json:"replay_diverged"`
	Cancelled            uint64                       `json:"cancelled"`
	TimedOut             uint64                       `json:"timed_out"`
	ElapsedNanos         record.Uint64String          `json:"elapsed_nanos"`
	ArtifactBytes        record.Uint64String          `json:"artifact_bytes"`
	TraceBytes           record.Uint64String          `json:"trace_bytes"`
	Suites               []SuiteReport                `json:"workloads"`
}

type SuiteReport struct {
	ID             string                       `json:"id"`
	Name           string                       `json:"name"`
	Tier           uint64                       `json:"tier"`
	Invariant      string                       `json:"invariant"`
	Expected       Expectation                  `json:"expected"`
	ExpectationMet bool                         `json:"expectation_met"`
	Classification string                       `json:"classification"`
	Analysis       *capabilityanalysis.Report   `json:"analysis,omitempty"`
	AnalysisError  string                       `json:"analysis_error,omitempty"`
	Seeds          []SeedReport                 `json:"seeds"`
	Blockers       []capabilityanalysis.Blocker `json:"blockers"`
	Choice         ChoiceCoverage               `json:"choice"`
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
	Seed              record.Uint64String `json:"seed"`
	Classification    string              `json:"classification"`
	EvidenceSHA256    record.SHA256       `json:"evidence_sha256,omitempty"`
	Replayed          bool                `json:"replayed"`
	ReplayMatch       bool                `json:"replay_match"`
	ReplayDivergence  string              `json:"replay_divergence,omitempty"`
	ChoiceReplayExact bool                `json:"choice_replay_exact,omitempty"`
	ElapsedNanos      record.Uint64String `json:"elapsed_nanos"`
	ArtifactBytes     record.Uint64String `json:"artifact_bytes"`
	TraceBytes        record.Uint64String `json:"trace_bytes"`
	Choice            ChoiceCoverage      `json:"choice"`
}

type ChoiceCoverage struct {
	Available              bool                 `json:"available"`
	Profile                string               `json:"profile,omitempty"`
	ImplementationSHA256   record.SHA256        `json:"implementation_sha256,omitempty"`
	Limit                  record.Uint64String  `json:"limit,omitempty"`
	TapeSHA256             record.SHA256        `json:"tape_sha256,omitempty"`
	Decisions              record.Uint64String  `json:"decisions,omitempty"`
	ExactReplayAvailable   bool                 `json:"exact_replay_available,omitempty"`
	Features               []choicewire.Feature `json:"features"`
	AdjacentPairsObserved  record.Uint64String  `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool                 `json:"adjacent_pairs_truncated"`
}

type Config struct {
	ManifestPath string
	GomadPath    string
	WorkingDir   string
	ArtifactRoot string
	OutputPath   string
	Execute      ExecuteFunc
}

type Command struct {
	Executable   string
	Args         []string
	Dir          string
	ArtifactRoot string
	Timeout      time.Duration
	Grace        time.Duration
}

type CommandResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Err      error
}

type ExecuteFunc func(context.Context, Command) CommandResult

type ExpectationError struct {
	Suites []string
}

func (err *ExpectationError) Error() string {
	return "qualification set did not match expectations: " + strings.Join(err.Suites, ", ")
}

type InvalidReportError struct {
	Err error
}

func (err *InvalidReportError) Error() string {
	return err.Err.Error()
}

func (err *InvalidReportError) Unwrap() error {
	return err.Err
}

func IsInvalidReport(err error) bool {
	var invalid *InvalidReportError
	return errors.As(err, &invalid)
}

func invalidReport(err error) error {
	return &InvalidReportError{Err: err}
}

var setNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var testNamePattern = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)

func LoadManifest(path string) (Manifest, error) {
	file, info, err := safefile.OpenPath(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read qualification set manifest: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumManifestBytes {
		return Manifest{}, errors.Join(fmt.Errorf("qualification set manifest must be between 1 and %d bytes", maximumManifestBytes), file.Close())
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumManifestBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return Manifest{}, errors.Join(fmt.Errorf("read qualification set manifest: %w", readErr), closeErr)
	}
	if len(contents) > maximumManifestBytes {
		return Manifest{}, fmt.Errorf("qualification set manifest exceeds %d bytes", maximumManifestBytes)
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(contents, &header); err != nil {
		return Manifest{}, fmt.Errorf("decode qualification set manifest schema: %w", err)
	}
	var manifest Manifest
	switch header.Schema {
	case ManifestSchema:
		if err := record.StrictDecode(contents, &manifest); err != nil {
			return Manifest{}, fmt.Errorf("decode qualification set manifest: %w", err)
		}
	case LegacyManifestSchema:
		var legacy legacyManifest
		if err := record.StrictDecode(contents, &legacy); err != nil {
			return Manifest{}, fmt.Errorf("decode legacy qualification set manifest: %w", err)
		}
		manifest = normalizeLegacyManifest(legacy)
	default:
		return Manifest{}, fmt.Errorf("unsupported qualification set manifest schema %q", header.Schema)
	}
	if len(manifest.Seeds) != 0 {
		manifest.Seed = manifest.Seeds[0]
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func normalizeLegacyManifest(legacy legacyManifest) Manifest {
	manifest := Manifest{
		Schema: ManifestSchema, Name: legacy.Name, Seeds: []uint64{legacy.Seed}, Seed: legacy.Seed,
		Repeat: legacy.Repeat, RunTimeout: legacy.RunTimeout, OverallTimeout: legacy.OverallTimeout,
		TerminateGrace: legacy.TerminateGrace, OutputBytes: legacy.OutputBytes,
		WorldTransitionBytes: legacy.WorldTransitionBytes, Suites: make([]Suite, len(legacy.Suites)), legacy: true,
	}
	for index, suite := range legacy.Suites {
		manifest.Suites[index] = Suite{
			ID: suite.Name, Name: suite.Name, Tier: 1, Package: suite.Package, Test: suite.Test,
			BuildTags: append([]string(nil), suite.BuildTags...), Environment: append([]string(nil), suite.Environment...),
			ReadOnlyMounts: append([]Mount(nil), suite.ReadOnlyMounts...), RequiredProbes: append([]string(nil), suite.RequiredProbes...),
			Expectation: suite.Expectation,
		}
	}
	return manifest
}

func Run(ctx context.Context, config Config) (SetReport, error) {
	manifest, err := LoadManifest(config.ManifestPath)
	if err != nil {
		return SetReport{}, err
	}
	if config.GomadPath == "" || config.WorkingDir == "" || config.ArtifactRoot == "" || config.OutputPath == "" {
		return SetReport{}, errors.New("qualification set requires gomad, working, artifact, and output paths")
	}
	for _, field := range []*string{&config.GomadPath, &config.WorkingDir, &config.ArtifactRoot, &config.OutputPath} {
		absolute, err := filepath.Abs(*field)
		if err != nil {
			return SetReport{}, err
		}
		*field = absolute
	}
	if config.Execute == nil {
		info, err := os.Stat(config.GomadPath)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return SetReport{}, errors.Join(errors.New("gomad executable is invalid"), err)
		}
		config.Execute = executeCommand
	}
	moduleIdentity, err := identifyModule(config.WorkingDir, manifest.Module)
	if err != nil {
		return SetReport{}, err
	}
	manifestBytes, err := record.CanonicalJSON(manifest)
	if err != nil {
		return SetReport{}, err
	}
	report := SetReport{
		Schema: ReportSchema, Name: manifest.Name, Description: manifest.Description,
		ManifestSHA256: record.HashBytes(manifestBytes), Module: moduleIdentity,
		Dimensions: EvidenceDimensions{PortableV3: !manifest.legacy, Analysis: true, Replay: !manifest.legacy, Choice: !manifest.legacy},
		Selected:   uint64(len(manifest.Suites)), Seeds: make([]record.Uint64String, len(manifest.Seeds)),
		Suites: make([]SuiteReport, len(manifest.Suites)),
	}
	for index, seed := range manifest.Seeds {
		report.Seeds[index] = record.Uint64String(seed)
	}
	for index, suite := range manifest.Suites {
		report.Suites[index] = SuiteReport{
			ID: suite.ID, Name: suite.Name, Tier: suite.Tier, Invariant: suite.Invariant,
			Expected: suite.Expectation, Seeds: []SeedReport{}, Blockers: []capabilityanalysis.Blocker{},
			Choice: emptyChoiceCoverage(),
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
			suiteReport.Blockers = append([]capabilityanalysis.Blocker{}, analysis.Blockers...)
			report.AnalysisCompleted++
			if err := mergeAnalysisIdentity(&report, analysis); err != nil {
				suiteReport.AnalysisError = "runner_failure"
				suiteReport.Classification = "runner_failure"
				analysisFailed = true
				failed = append(failed, suite.ID)
			} else if analysis.Classification == capabilityanalysis.ClassificationUnsupported {
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
			if report.Suites[index].Analysis == nil || report.Suites[index].Analysis.Classification == capabilityanalysis.ClassificationUnsupported {
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
				seedReport := SeedReport{Seed: record.Uint64String(seed), Classification: classification, Choice: emptyChoiceCoverage()}
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
		return report, &ExpectationError{Suites: failed}
	}
	return report, nil
}

func OpenReport(path string) (SetReport, error) {
	file, info, err := safefile.OpenPath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return SetReport{}, fmt.Errorf("open qualification set report: %w", err)
		}
		return SetReport{}, invalidReport(fmt.Errorf("open qualification set report: %w", err))
	}
	if info.Mode().Perm() != 0o600 || info.Size() <= 0 || info.Size() > maximumReportBytes {
		return SetReport{}, invalidReport(errors.Join(errors.New("qualification set report must be a private bounded regular file"), file.Close()))
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumReportBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return SetReport{}, errors.Join(fmt.Errorf("read qualification set report: %w", readErr), closeErr)
	}
	if len(contents) > maximumReportBytes || len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return SetReport{}, invalidReport(errors.New("qualification set report framing is invalid"))
	}
	contents = contents[:len(contents)-1]
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(contents, &header); err != nil {
		return SetReport{}, invalidReport(fmt.Errorf("decode qualification set report schema: %w", err))
	}
	if header.Schema == LegacyReportSchema {
		report, err := decodeLegacySetReport(contents)
		if err != nil {
			return SetReport{}, invalidReport(err)
		}
		return report, nil
	}
	if header.Schema != ReportSchema && header.Schema != PreviousReportSchema {
		return SetReport{}, invalidReport(fmt.Errorf("unsupported qualification set report schema %q", header.Schema))
	}
	var report SetReport
	if err := record.DecodeCanonicalJSON(contents, &report); err != nil {
		return SetReport{}, invalidReport(fmt.Errorf("decode qualification set report: %w", err))
	}
	if report.Schema == PreviousReportSchema {
		report.Schema = ReportSchema
	}
	if err := validateSetReport(report); err != nil {
		return SetReport{}, invalidReport(err)
	}
	return report, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema || !setNamePattern.MatchString(manifest.Name) || manifest.Repeat < 2 || manifest.Repeat > 32 || len(manifest.Seeds) == 0 || len(manifest.Seeds) > 32 || len(manifest.Suites) == 0 || len(manifest.Suites) > 64 || manifest.OutputBytes == 0 || manifest.WorldTransitionBytes == 0 {
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

func validateExpectation(expectation Expectation) error {
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

func suiteCommand(config Config, manifest Manifest, suite Suite, seed uint64) Command {
	runTimeout, overallTimeout := suiteTimeouts(manifest, suite)
	terminateGrace := manifestGrace(manifest)
	args := []string{
		"qualify", "--json", "--seed=" + strconv.FormatUint(seed, 10), "--repeat=" + strconv.FormatUint(manifest.Repeat, 10),
		"--run-timeout=" + runTimeout.String(), "--overall-timeout=" + overallTimeout.String(), "--terminate-grace=" + manifest.TerminateGrace,
		"--output-limit=" + strconv.FormatUint(manifest.OutputBytes, 10), "--world-transition-limit=" + strconv.FormatUint(manifest.WorldTransitionBytes, 10),
		"--artifacts=" + config.ArtifactRoot,
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
	return Command{
		Executable: config.GomadPath, Args: args, Dir: config.WorkingDir, ArtifactRoot: config.ArtifactRoot,
		Timeout: overallTimeout + terminateGrace + 10*time.Second, Grace: terminateGrace,
	}
}

func executeCommand(ctx context.Context, command Command) CommandResult {
	executed, err := commandrun.Run(ctx, commandrun.Request{
		Command: append([]string{command.Executable}, command.Args...), Dir: command.Dir, Env: os.Environ(),
		Timeout: command.Timeout, TerminateGrace: command.Grace, OutputLimit: maximumCommandOutputBytes,
	})
	result := CommandResult{ExitCode: executed.ExitCode, Stdout: executed.Stdout.Bytes, Stderr: executed.Stderr.Bytes, Err: err}
	if executed.Termination == commandrun.TerminationSignal {
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

func retainedQualification(artifactRoot string, command Command, result CommandResult) (qualify.Report, string, string, record.SHA256, error) {
	if result.Err != nil {
		return qualify.Report{}, "", "", "", result.Err
	}
	if len(result.Stderr) != 0 {
		return qualify.Report{}, "", "", "", errors.New("JSON qualification command wrote to stderr")
	}
	event, err := qualify.DecodeResultEvent(result.Stdout)
	if err != nil {
		return qualify.Report{}, "", "", "", err
	}
	path, err := filepath.Abs(event.ReportPath)
	if err != nil {
		return qualify.Report{}, "", "", "", err
	}
	relative, err := filepath.Rel(artifactRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return qualify.Report{}, "", "", "", errors.Join(errors.New("qualification report is outside the artifact root"), err)
	}
	opened, err := qualify.Open(path)
	if err != nil {
		return qualify.Report{}, "", "", "", err
	}
	logicalCommand := append([]string{"gomad"}, command.Args...)
	if !slices.Equal(opened.Command, logicalCommand) {
		return qualify.Report{}, "", "", "", errors.New("retained qualification command does not match the executed command")
	}
	classification := qualify.Classify(opened)
	if event.Classification != classification || result.ExitCode != qualify.ExitStatus(classification) {
		return qualify.Report{}, "", "", "", fmt.Errorf("qualification result classification or status is inconsistent: %s/%d", event.Classification, result.ExitCode)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return qualify.Report{}, "", "", "", err
	}
	return opened, classification, path, record.HashBytes(contents), nil
}

func matchesExpectation(expected Expectation, classification string, report qualify.Report, exitCode int) bool {
	if expected.Classification != classification || exitCode != qualify.ExitStatus(classification) {
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

func firstReplay(report qualify.Report) *qualify.Replay {
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
	Schema string    `json:"schema"`
	Phase  string    `json:"phase"`
	Key    string    `json:"key"`
	Report SetReport `json:"report"`
}

func writeCheckpoint(ctx context.Context, path, phase, key string, report SetReport) error {
	contents, err := record.CanonicalJSON(checkpoint{Schema: "gomadv3.qualification-set-checkpoint/v1", Phase: phase, Key: key, Report: report})
	if err != nil {
		return err
	}
	if len(contents) > maximumReportBytes {
		return fmt.Errorf("qualification set checkpoint exceeds %d bytes", maximumReportBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return safefile.ReplaceContext(ctx, path, append(contents, '\n'), 0o600)
}

func writeReport(ctx context.Context, path string, report SetReport) error {
	if err := validateSetReport(report); err != nil {
		return err
	}
	contents, err := record.CanonicalJSON(report)
	if err != nil {
		return err
	}
	if len(contents) > maximumReportBytes {
		return fmt.Errorf("qualification set report exceeds %d bytes", maximumReportBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return safefile.ReplaceContext(ctx, path, append(contents, '\n'), 0o600)
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
	elapsedNanos, artifactBytes, traceBytes                                      record.Uint64String
}

func deriveSetReportCounters(report SetReport) setReportCounters {
	var result setReportCounters
	for _, suite := range report.Suites {
		if suite.Analysis != nil {
			result.analysisCompleted++
			if suite.Analysis.Classification == capabilityanalysis.ClassificationUnsupported || len(suite.Seeds) == len(report.Seeds) {
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

func finalizeSetReportCounters(report *SetReport) {
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

func validateSetReport(report SetReport) error {
	if report.Schema != ReportSchema || !setNamePattern.MatchString(report.Name) || report.ManifestSHA256 == "" || len(report.Seeds) == 0 || report.Selected != uint64(len(report.Suites)) || report.Completed > report.Selected || report.AnalysisCompleted > report.Selected {
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
		if !setNamePattern.MatchString(suite.ID) || suite.Name == "" || suite.Tier != 1 && suite.Tier != 2 || report.Dimensions.PortableV3 && suite.Invariant == "" || suite.Seeds == nil || suite.Blockers == nil || suite.Choice.Features == nil || !validSetClassification(suite.Classification) || index > 0 && suite.ID <= report.Suites[index-1].ID {
			return fmt.Errorf("qualification set workload %d identity is invalid", index)
		}
		if report.Dimensions.Analysis && (suite.Analysis == nil && suite.AnalysisError == "" || suite.Analysis != nil && suite.AnalysisError != "") {
			return fmt.Errorf("qualification set workload %s analysis state is invalid", suite.ID)
		}
		if !report.Dimensions.Analysis && (suite.Analysis != nil || suite.AnalysisError != "dimension_unavailable") {
			return fmt.Errorf("qualification set workload %s legacy analysis state is invalid", suite.ID)
		}
		if suite.Analysis != nil {
			encoded, err := record.CanonicalJSON(*suite.Analysis)
			if err != nil {
				return err
			}
			if _, err := capabilityanalysis.Decode(encoded); err != nil {
				return fmt.Errorf("qualification set workload %s analysis is invalid: %w", suite.ID, err)
			}
			if suite.Analysis.Classification == capabilityanalysis.ClassificationUnsupported && (suite.Classification != "unsupported_target" || len(suite.Seeds) != 0) {
				return fmt.Errorf("qualification set workload %s unsupported analysis is inconsistent", suite.ID)
			}
			if suite.Analysis.Classification == capabilityanalysis.ClassificationSupported && suite.Classification == "unsupported_target" {
				return fmt.Errorf("qualification set workload %s supported analysis is inconsistent", suite.ID)
			}
		}
		for seedIndex, seed := range suite.Seeds {
			if seed.Choice.Features == nil || seed.ChoiceReplayExact && (!seed.Replayed || !seed.ReplayMatch || !seed.Choice.ExactReplayAvailable || seed.Choice.TapeSHA256 == "") || !validSetClassification(seed.Classification) || seedIndex >= len(report.Seeds) || seed.Seed != report.Seeds[seedIndex] {
				return fmt.Errorf("qualification set workload %s seed evidence is invalid", suite.ID)
			}
		}
		if suite.Choice.ExactReplayAvailable && (!suite.Choice.Available || suite.Choice.Profile != choicewire.Profile) {
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
