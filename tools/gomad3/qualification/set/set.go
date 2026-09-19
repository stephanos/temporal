package set

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

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/qualification"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

const ManifestSchema = "gomad3.qualification-set/v1"
const ReportSchema = "gomad3.qualification-set-report/v1"

const maximumCommandOutputBytes = 64 << 20
const maximumManifestBytes = 1 << 20
const maximumSetReportBytes = 64 << 20

type Manifest struct {
	Schema               string     `json:"schema"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	Module               string     `json:"module"`
	Seeds                []uint64   `json:"seeds"`
	Repeat               uint64     `json:"repeat"`
	ExecutionTimeout     string     `json:"execution_timeout"`
	OverallTimeout       string     `json:"overall_timeout"`
	TerminateGrace       string     `json:"terminate_grace"`
	OutputBytes          uint64     `json:"output_bytes"`
	WorldTransitionBytes uint64     `json:"world_transition_bytes"`
	Workloads            []Workload `json:"workloads"`
	Seed                 uint64     `json:"-"`
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
	ExecutionTimeout     string                `json:"execution_timeout,omitempty"`
	OverallTimeout       string                `json:"overall_timeout,omitempty"`
	Expectation          WorkloadExpectation   `json:"expectation"`
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

type Report struct {
	Schema               string                       `json:"schema"`
	Name                 string                       `json:"name"`
	Description          string                       `json:"description"`
	ManifestSHA256       record.SHA256                `json:"manifest_sha256"`
	Seeds                []record.Uint64String        `json:"seeds"`
	Module               ModuleIdentity               `json:"module"`
	Platform             PlatformIdentity             `json:"platform"`
	Toolchain            capabilityanalysis.Toolchain `json:"toolchain"`
	IOProfile            deterministicio.Contract     `json:"io_profile"`
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
	Workloads            []WorkloadReport             `json:"workloads"`
}

type WorkloadReport struct {
	ID             string                       `json:"id"`
	Name           string                       `json:"name"`
	Tier           uint64                       `json:"tier"`
	Invariant      string                       `json:"invariant"`
	Expected       WorkloadExpectation          `json:"expected"`
	ExpectationMet bool                         `json:"expectation_met"`
	Classification string                       `json:"classification"`
	Analysis       *capabilityanalysis.Report   `json:"analysis,omitempty"`
	AnalysisError  string                       `json:"analysis_error,omitempty"`
	Seeds          []SeedReport                 `json:"seeds"`
	Blockers       []capabilityanalysis.Blocker `json:"blockers"`
	Choice         ChoiceCoverage               `json:"choice"`
	CapabilityMode target.CapabilityMode        `json:"capability_mode,omitempty"`
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
	Available              bool                `json:"available"`
	Profile                string              `json:"profile,omitempty"`
	ImplementationSHA256   record.SHA256       `json:"implementation_sha256,omitempty"`
	Limit                  record.Uint64String `json:"limit,omitempty"`
	TapeSHA256             record.SHA256       `json:"tape_sha256,omitempty"`
	Decisions              record.Uint64String `json:"decisions,omitempty"`
	ExactReplayAvailable   bool                `json:"exact_replay_available,omitempty"`
	Features               []choice.Feature    `json:"features"`
	AdjacentPairsObserved  record.Uint64String `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool                `json:"adjacent_pairs_truncated"`
}

type Spec struct {
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
	Workloads []string
}

func (err *ExpectationError) Error() string {
	return "qualification set did not match expectations: " + strings.Join(err.Workloads, ", ")
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
	file, info, err := hostfs.OpenPath(path)
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
	var manifest Manifest
	if err := canonicaljson.StrictDecode(contents, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode qualification set manifest: %w", err)
	}
	if len(manifest.Seeds) != 0 {
		manifest.Seed = manifest.Seeds[0]
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Run(ctx context.Context, config Spec) (Report, error) {
	manifest, err := LoadManifest(config.ManifestPath)
	if err != nil {
		return Report{}, err
	}
	if config.GomadPath == "" || config.WorkingDir == "" || config.ArtifactRoot == "" || config.OutputPath == "" {
		return Report{}, errors.New("qualification set requires gomad, working, artifact, and output paths")
	}
	for _, field := range []*string{&config.GomadPath, &config.WorkingDir, &config.ArtifactRoot, &config.OutputPath} {
		absolute, err := filepath.Abs(*field)
		if err != nil {
			return Report{}, err
		}
		*field = absolute
	}
	if config.Execute == nil {
		info, err := os.Stat(config.GomadPath)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return Report{}, errors.Join(errors.New("gomad executable is invalid"), err)
		}
		config.Execute = executeCommand
	}
	moduleIdentity, err := identifyModule(config.WorkingDir, manifest.Module)
	if err != nil {
		return Report{}, err
	}
	manifestBytes, err := canonicaljson.CanonicalJSON(manifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Schema: ReportSchema, Name: manifest.Name, Description: manifest.Description,
		ManifestSHA256: record.HashBytes(manifestBytes), Module: moduleIdentity,
		Dimensions: EvidenceDimensions{PortableV3: true, Analysis: true, Replay: true, Choice: true},
		Selected:   uint64(len(manifest.Workloads)), Seeds: make([]record.Uint64String, len(manifest.Seeds)),
		Workloads: make([]WorkloadReport, len(manifest.Workloads)),
	}
	for index, seed := range manifest.Seeds {
		report.Seeds[index] = record.Uint64String(seed)
	}
	for index, workload := range manifest.Workloads {
		report.Workloads[index] = WorkloadReport{
			ID: workload.ID, Name: workload.Name, Tier: workload.Tier, Invariant: workload.Invariant,
			Expected: workload.Expectation, Seeds: []SeedReport{}, Blockers: []capabilityanalysis.Blocker{},
			Choice: emptyChoiceCoverage(), CapabilityMode: workload.CapabilityMode,
		}
	}
	failed := make([]string, 0, len(manifest.Workloads))
	analysisFailed := false
	for index, workload := range manifest.Workloads {
		workloadReport := report.Workloads[index]
		if err := ctx.Err(); err != nil {
			workloadReport.AnalysisError = contextClassification(err)
			workloadReport.Classification = workloadReport.AnalysisError
			report.Workloads[index] = workloadReport
			failed = append(failed, workload.ID)
			analysisFailed = true
			break
		}
		command := analysisCommand(config, manifest, workload)
		result := config.Execute(ctx, command)
		analysis, classification, analysisErr := retainedAnalysis(result)
		if analysisErr != nil {
			workloadReport.AnalysisError = classification
			workloadReport.Classification = classification
			analysisFailed = true
			failed = append(failed, workload.ID)
		} else {
			workloadReport.Analysis = &analysis
			workloadReport.Blockers = append([]capabilityanalysis.Blocker{}, analysis.Blockers...)
			report.AnalysisCompleted++
			if err := mergeAnalysisIdentity(&report, analysis); err != nil {
				workloadReport.AnalysisError = "runner_failure"
				workloadReport.Classification = "runner_failure"
				analysisFailed = true
				failed = append(failed, workload.ID)
			} else if analysis.Classification == capabilityanalysis.ClassificationUnsupported {
				workloadReport.Classification = "unsupported_target"
				workloadReport.ExpectationMet = matchesUnsupportedAnalysis(workload.Expectation, analysis)
				report.Completed++
				report.Unsupported++
				if !workloadReport.ExpectationMet {
					failed = append(failed, workload.ID)
				}
			}
		}
		report.Workloads[index] = workloadReport
		if checkpointErr := writeCheckpoint(context.WithoutCancel(ctx), config.OutputPath+".partial", "analysis", workload.ID, report); checkpointErr != nil {
			return report, checkpointErr
		}
	}
	if analysisFailed {
		classification := contextClassification(ctx.Err())
		for index := range report.Workloads {
			if report.Workloads[index].Classification != "" {
				continue
			}
			report.Workloads[index].Classification = classification
			if report.Workloads[index].Analysis == nil {
				report.Workloads[index].AnalysisError = classification
			}
			failed = append(failed, report.Workloads[index].ID)
		}
	}
	if !analysisFailed {
		for index, workload := range manifest.Workloads {
			if report.Workloads[index].Analysis == nil || report.Workloads[index].Analysis.Classification == capabilityanalysis.ClassificationUnsupported {
				continue
			}
			allQualified := true
			for _, seed := range manifest.Seeds {
				if err := ctx.Err(); err != nil {
					classification := contextClassification(err)
					report.Workloads[index].Classification = classification
					failed = append(failed, workload.ID)
					allQualified = false
					break
				}
				command := workloadCommand(config, manifest, workload, seed)
				result := config.Execute(ctx, command)
				opened, classification, _, _, evidenceErr := retainedQualification(config.ArtifactRoot, command, result)
				seedReport := SeedReport{Seed: record.Uint64String(seed), Classification: classification, Choice: emptyChoiceCoverage()}
				if evidenceErr == nil {
					projected, projectErr := projectSeedReport(opened, classification, seed, workload, *report.Workloads[index].Analysis)
					if projectErr == nil {
						seedReport = projected
					}
					evidenceErr = projectErr
				}
				if evidenceErr != nil {
					classification = retainedErrorClassification(result, evidenceErr)
					seedReport.Classification = classification
				}
				report.Workloads[index].Seeds = append(report.Workloads[index].Seeds, seedReport)
				addSeedTotals(&report, seedReport)
				if classification != "qualified" {
					allQualified = false
					if report.Workloads[index].Classification == "" {
						report.Workloads[index].Classification = classification
					}
				}
				if checkpointErr := writeCheckpoint(context.WithoutCancel(ctx), config.OutputPath+".partial", "qualification", workload.ID+"/"+strconv.FormatUint(seed, 10), report); checkpointErr != nil {
					return report, checkpointErr
				}
			}
			coverage, coverageErr := aggregateChoiceCoverage(report.Workloads[index].Seeds)
			if coverageErr != nil {
				report.Workloads[index].Classification = "runner_failure"
				allQualified = false
			}
			if allQualified && len(report.Workloads[index].Seeds) == len(manifest.Seeds) {
				report.Workloads[index].Classification = "qualified"
				report.Supported++
			} else {
				switch classificationBucket(report.Workloads[index].Classification) {
				case qualificationFailed:
					report.Failed++
				default:
					report.InfrastructureErrors++
				}
			}
			report.Completed++
			report.Workloads[index].Choice = coverage
			report.Workloads[index].ExpectationMet = matchesSupportedExpectation(workload.Expectation, report.Workloads[index])
			if !report.Workloads[index].ExpectationMet {
				failed = append(failed, workload.ID)
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
		return report, &ExpectationError{Workloads: failed}
	}
	return report, nil
}

func OpenReport(path string) (Report, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return Report{}, fmt.Errorf("open qualification set report: %w", err)
		}
		return Report{}, invalidReport(fmt.Errorf("open qualification set report: %w", err))
	}
	if info.Mode().Perm() != 0o600 || info.Size() <= 0 || info.Size() > maximumSetReportBytes {
		return Report{}, invalidReport(errors.Join(errors.New("qualification set report must be a private bounded regular file"), file.Close()))
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumSetReportBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return Report{}, errors.Join(fmt.Errorf("read qualification set report: %w", readErr), closeErr)
	}
	if len(contents) > maximumSetReportBytes || len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return Report{}, invalidReport(errors.New("qualification set report framing is invalid"))
	}
	contents = contents[:len(contents)-1]
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(contents, &header); err != nil {
		return Report{}, invalidReport(fmt.Errorf("decode qualification set report schema: %w", err))
	}
	if header.Schema != ReportSchema {
		return Report{}, invalidReport(fmt.Errorf("unsupported qualification set report schema %q", header.Schema))
	}
	var report Report
	if err := canonicaljson.DecodeCanonicalJSON(contents, &report); err != nil {
		return Report{}, invalidReport(fmt.Errorf("decode qualification set report: %w", err))
	}
	if err := validateSetReport(report); err != nil {
		return Report{}, invalidReport(err)
	}
	return report, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema || !setNamePattern.MatchString(manifest.Name) || manifest.Repeat < 2 || manifest.Repeat > 32 || len(manifest.Seeds) == 0 || len(manifest.Seeds) > 32 || len(manifest.Workloads) == 0 || len(manifest.Workloads) > 64 || manifest.OutputBytes == 0 || manifest.WorldTransitionBytes == 0 {
		return errors.New("qualification set manifest identity or bounds are invalid")
	}
	if strings.TrimSpace(manifest.Description) == "" || strings.TrimSpace(manifest.Module) == "" || strings.ContainsAny(manifest.Module, "\x00\n\r\t ") {
		return errors.New("qualification set portable identity is invalid")
	}
	for index, seed := range manifest.Seeds {
		if index > 0 && seed <= manifest.Seeds[index-1] {
			return errors.New("qualification set seeds must be sorted and unique")
		}
	}
	runTimeout, err := time.ParseDuration(manifest.ExecutionTimeout)
	if err != nil || runTimeout <= 0 {
		return errors.Join(errors.New("qualification set execution timeout is invalid"), err)
	}
	overallTimeout, err := time.ParseDuration(manifest.OverallTimeout)
	if err != nil || overallTimeout <= 0 {
		return errors.Join(errors.New("qualification set overall timeout is invalid"), err)
	}
	grace, err := time.ParseDuration(manifest.TerminateGrace)
	if err != nil || grace < 0 || grace > runTimeout || grace > overallTimeout {
		return errors.Join(errors.New("qualification set termination grace is invalid"), err)
	}
	seen := make(map[string]struct{}, len(manifest.Workloads))
	for index, workload := range manifest.Workloads {
		packagePath := strings.TrimPrefix(workload.Package, "./")
		if !setNamePattern.MatchString(workload.ID) || strings.TrimSpace(workload.Name) == "" || !strings.HasPrefix(workload.Package, "./") || packagePath == "" || filepath.ToSlash(filepath.Clean(filepath.FromSlash(packagePath))) != packagePath || strings.HasPrefix(packagePath, "../") || !testNamePattern.MatchString(workload.Test) {
			return fmt.Errorf("qualification workload %d identity is invalid", index)
		}
		if workload.Tier != 1 && workload.Tier != 2 || strings.TrimSpace(workload.Invariant) == "" {
			return fmt.Errorf("qualification workload %s tier or invariant is invalid", workload.ID)
		}
		if index > 0 && workload.ID <= manifest.Workloads[index-1].ID {
			return errors.New("qualification workload identities must be sorted and unique")
		}
		if _, duplicate := seen[workload.ID]; duplicate {
			return fmt.Errorf("qualification workload identity is duplicated: %s", workload.ID)
		}
		seen[workload.ID] = struct{}{}
		if !sortedUnique(workload.BuildTags) || !sortedUnique(workload.Environment) || !sortedUnique(workload.RequiredProbes) {
			return fmt.Errorf("qualification workload %s lists must be sorted and unique", workload.ID)
		}
		if workload.CapabilityMode != target.CapabilityModeClosure && workload.CapabilityMode != target.CapabilityModeLinked && workload.CapabilityMode != target.CapabilityModeGuarded {
			return fmt.Errorf("qualification workload %s capability mode is invalid", workload.ID)
		}
		for _, mount := range workload.ReadOnlyMounts {
			if mount.Source == "" || mount.Target == "" || strings.ContainsAny(mount.Source+mount.Target, "\x00\n") {
				return fmt.Errorf("qualification workload %s has an invalid read-only mount", workload.ID)
			}
		}
		if workload.ReplaySuccesses != (workload.SuccessArtifactLimit != 0 && workload.SuccessBytesLimit != 0) || workload.ReplaySuccesses && workload.ChoiceBytes == 0 || !workload.ReplaySuccesses && (workload.SuccessArtifactLimit != 0 || workload.SuccessBytesLimit != 0) {
			return fmt.Errorf("qualification workload %s replay bounds are invalid", workload.ID)
		}
		for _, override := range []string{workload.ExecutionTimeout, workload.OverallTimeout} {
			if override != "" {
				parsed, parseErr := time.ParseDuration(override)
				if parseErr != nil || parsed <= 0 {
					return errors.Join(fmt.Errorf("qualification workload %s timeout override is invalid", workload.ID), parseErr)
				}
			}
		}
		if err := validateExpectation(workload.Expectation); err != nil {
			return fmt.Errorf("qualification workload %s: %w", workload.ID, err)
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

func workloadCommand(config Spec, manifest Manifest, workload Workload, seed uint64) Command {
	runTimeout, overallTimeout := workloadTimeouts(manifest, workload)
	terminateGrace := manifestGrace(manifest)
	args := []string{
		"qualify", "--json", "--seed=" + strconv.FormatUint(seed, 10), "--repeat=" + strconv.FormatUint(manifest.Repeat, 10),
		"--execution-timeout=" + runTimeout.String(), "--overall-timeout=" + overallTimeout.String(), "--terminate-grace=" + manifest.TerminateGrace,
		"--output-limit=" + strconv.FormatUint(manifest.OutputBytes, 10), "--world-transition-limit=" + strconv.FormatUint(manifest.WorldTransitionBytes, 10),
		"--artifacts=" + config.ArtifactRoot,
		"--capability-mode=" + string(workload.CapabilityMode),
	}
	if workload.ChoiceBytes != 0 {
		args = append(args, "--choices", "--choice-bytes="+strconv.FormatUint(workload.ChoiceBytes, 10))
	}
	if workload.ReplaySuccesses {
		args = append(args, "--replay-successes", "--success-limit="+strconv.FormatUint(workload.SuccessArtifactLimit, 10), "--success-bytes="+strconv.FormatUint(workload.SuccessBytesLimit, 10))
	}
	for _, value := range workload.BuildTags {
		args = append(args, "--build-tag="+value)
	}
	for _, value := range workload.Environment {
		args = append(args, "--env="+value)
	}
	for _, mount := range workload.ReadOnlyMounts {
		args = append(args, "--io-ro-mount="+mount.Source+"="+mount.Target)
	}
	for _, value := range workload.RequiredProbes {
		args = append(args, "--require-probe="+value)
	}
	args = append(args, "go-test", workload.Package, "--", "-test.run=^"+regexp.QuoteMeta(workload.Test)+"$")
	return Command{
		Executable: config.GomadPath, Args: args, Dir: config.WorkingDir, ArtifactRoot: config.ArtifactRoot,
		Timeout: overallTimeout + terminateGrace + 10*time.Second, Grace: terminateGrace,
	}
}

func executeCommand(ctx context.Context, command Command) CommandResult {
	executed, err := hostexec.Run(ctx, hostexec.Request{
		Command: append([]string{command.Executable}, command.Args...), Dir: command.Dir, Env: os.Environ(),
		Timeout: command.Timeout, TerminateGrace: command.Grace, OutputLimit: maximumCommandOutputBytes,
	})
	result := CommandResult{ExitCode: executed.ExitCode, Stdout: executed.Stdout.Bytes, Stderr: executed.Stderr.Bytes, Err: err}
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

func retainedQualification(artifactRoot string, command Command, result CommandResult) (qualification.QualificationReport, string, string, record.SHA256, error) {
	if result.Err != nil {
		return qualification.QualificationReport{}, "", "", "", result.Err
	}
	if len(result.Stderr) != 0 {
		return qualification.QualificationReport{}, "", "", "", errors.New("JSON qualification command wrote to stderr")
	}
	event, err := qualification.DecodeResultEvent(result.Stdout)
	if err != nil {
		return qualification.QualificationReport{}, "", "", "", err
	}
	path, err := filepath.Abs(event.ReportPath)
	if err != nil {
		return qualification.QualificationReport{}, "", "", "", err
	}
	relative, err := filepath.Rel(artifactRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return qualification.QualificationReport{}, "", "", "", errors.Join(errors.New("qualification report is outside the artifact root"), err)
	}
	opened, err := qualification.OpenQualificationReport(path)
	if err != nil {
		return qualification.QualificationReport{}, "", "", "", err
	}
	logicalCommand := append([]string{"gomad"}, command.Args...)
	if !slices.Equal(opened.Command, logicalCommand) {
		return qualification.QualificationReport{}, "", "", "", errors.New("retained qualification command does not match the executed command")
	}
	classification := qualification.ClassifyQualification(opened)
	if event.Classification != classification || result.ExitCode != qualification.ExitStatus(classification) {
		return qualification.QualificationReport{}, "", "", "", fmt.Errorf("qualification result classification or status is inconsistent: %s/%d", event.Classification, result.ExitCode)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return qualification.QualificationReport{}, "", "", "", err
	}
	return opened, classification, path, record.HashBytes(contents), nil
}

func matchesExpectation(expected WorkloadExpectation, classification string, report qualification.QualificationReport, exitCode int) bool {
	if expected.Classification != classification || exitCode != qualification.ExitStatus(classification) {
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

func firstReplay(report qualification.QualificationReport) *qualification.QualificationReplay {
	for _, run := range report.Executions {
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
	Schema string `json:"schema"`
	Phase  string `json:"phase"`
	Key    string `json:"key"`
	Report Report `json:"report"`
}

func writeCheckpoint(ctx context.Context, path, phase, key string, report Report) error {
	contents, err := canonicaljson.CanonicalJSON(checkpoint{Schema: "gomad3.qualification-set-checkpoint/v1", Phase: phase, Key: key, Report: report})
	if err != nil {
		return err
	}
	if len(contents) > maximumSetReportBytes {
		return fmt.Errorf("qualification set checkpoint exceeds %d bytes", maximumSetReportBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return hostfs.ReplaceContext(ctx, path, append(contents, '\n'), 0o600)
}

func writeReport(ctx context.Context, path string, report Report) error {
	if err := validateSetReport(report); err != nil {
		return err
	}
	contents, err := canonicaljson.CanonicalJSON(report)
	if err != nil {
		return err
	}
	if len(contents) > maximumSetReportBytes {
		return fmt.Errorf("qualification set report exceeds %d bytes", maximumSetReportBytes)
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
	elapsedNanos, artifactBytes, traceBytes                                      record.Uint64String
}

func deriveSetReportCounters(report Report) setReportCounters {
	var result setReportCounters
	for _, workload := range report.Workloads {
		if workload.Analysis != nil {
			result.analysisCompleted++
			if workload.Analysis.Classification == capabilityanalysis.ClassificationUnsupported || len(workload.Seeds) == len(report.Seeds) {
				result.completed++
			}
		}
		switch classificationBucket(workload.Classification) {
		case qualificationSupported:
			result.supported++
		case qualificationUnsupported:
			result.unsupported++
		case qualificationFailed:
			result.failed++
		default:
			result.infrastructure++
		}
		if workload.Classification == "cancelled" {
			result.cancelled++
		}
		if workload.Classification == "overall_timeout" {
			result.timedOut++
		}
		for _, seed := range workload.Seeds {
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

func finalizeSetReportCounters(report *Report) {
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

func validateSetReport(report Report) error {
	if report.Schema != ReportSchema || !setNamePattern.MatchString(report.Name) || report.ManifestSHA256 == "" || len(report.Seeds) == 0 || report.Selected != uint64(len(report.Workloads)) || report.Completed > report.Selected || report.AnalysisCompleted > report.Selected {
		return errors.New("qualification set report identity or counts are invalid")
	}
	if report.Dimensions != (EvidenceDimensions{PortableV3: true, Analysis: true, Replay: true, Choice: true}) {
		return errors.New("qualification set report omits a required evidence dimension")
	}
	for index, seed := range report.Seeds {
		if index > 0 && seed <= report.Seeds[index-1] {
			return errors.New("qualification set report seeds must be sorted and unique")
		}
	}
	if _, err := report.ManifestSHA256.Bytes(); err != nil {
		return fmt.Errorf("qualification set manifest digest is invalid: %w", err)
	}
	if report.Module.Path == "" || report.Module.GoModSHA256 == "" {
		return errors.New("qualification set module identity is incomplete")
	}
	if _, err := report.Module.GoModSHA256.Bytes(); err != nil {
		return fmt.Errorf("qualification set module digest is invalid: %w", err)
	}
	if report.Platform.GOOS == "" || report.Platform.GOARCH == "" || report.Toolchain.BuildKey == "" || report.IOProfile.Name == "" {
		return errors.New("qualification set implementation identity is incomplete")
	}
	if report.Supported+report.Unsupported+report.Failed+report.InfrastructureErrors != report.Selected {
		return errors.New("qualification set result counts are inconsistent")
	}
	expectationsMet := report.Completed == report.Selected
	for index, workload := range report.Workloads {
		if !setNamePattern.MatchString(workload.ID) || workload.Name == "" || workload.Tier != 1 && workload.Tier != 2 || workload.Invariant == "" || workload.Seeds == nil || workload.Blockers == nil || workload.Choice.Features == nil || workload.CapabilityMode != target.CapabilityModeClosure && workload.CapabilityMode != target.CapabilityModeLinked && workload.CapabilityMode != target.CapabilityModeGuarded || !validSetClassification(workload.Classification) || index > 0 && workload.ID <= report.Workloads[index-1].ID {
			return fmt.Errorf("qualification set workload %d identity is invalid", index)
		}
		if workload.Analysis == nil && workload.AnalysisError == "" || workload.Analysis != nil && workload.AnalysisError != "" {
			return fmt.Errorf("qualification set workload %s analysis state is invalid", workload.ID)
		}
		if workload.Analysis != nil {
			encoded, err := canonicaljson.CanonicalJSON(*workload.Analysis)
			if err != nil {
				return err
			}
			if _, err := capabilityanalysis.Decode(encoded); err != nil {
				return fmt.Errorf("qualification set workload %s analysis is invalid: %w", workload.ID, err)
			}
			if workload.Analysis.Target.CapabilityMode != workload.CapabilityMode {
				return fmt.Errorf("qualification set workload %s capability mode does not match its analysis", workload.ID)
			}
			if workload.Analysis.Classification == capabilityanalysis.ClassificationUnsupported && (workload.Classification != "unsupported_target" || len(workload.Seeds) != 0) {
				return fmt.Errorf("qualification set workload %s unsupported analysis is inconsistent", workload.ID)
			}
			if workload.Analysis.Classification == capabilityanalysis.ClassificationSupported && workload.Classification == "unsupported_target" {
				return fmt.Errorf("qualification set workload %s supported analysis is inconsistent", workload.ID)
			}
		}
		for seedIndex, seed := range workload.Seeds {
			if seed.Choice.Features == nil || seed.ChoiceReplayExact && (!seed.Replayed || !seed.ReplayMatch || !seed.Choice.ExactReplayAvailable || seed.Choice.TapeSHA256 == "") || !validSetClassification(seed.Classification) || seedIndex >= len(report.Seeds) || seed.Seed != report.Seeds[seedIndex] {
				return fmt.Errorf("qualification set workload %s seed evidence is invalid", workload.ID)
			}
		}
		if workload.Choice.ExactReplayAvailable && (!workload.Choice.Available || workload.Choice.Profile != choice.Profile) {
			return fmt.Errorf("qualification set workload %s exact choice identity is invalid", workload.ID)
		}
		if workload.Classification == "qualified" && len(workload.Seeds) != len(report.Seeds) {
			return fmt.Errorf("qualification set workload %s is missing qualified seed evidence", workload.ID)
		}
		expectationsMet = expectationsMet && workload.ExpectationMet
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
