package qualificationset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/qualify"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const ManifestSchema = "gomadv3.qualification-set/v1"
const ReportSchema = "gomadv3.qualification-set-report/v1"

const maximumCommandOutputBytes = 64 << 20
const maximumManifestBytes = 1 << 20
const maximumReportBytes = 64 << 20

type Manifest struct {
	Schema               string  `json:"schema"`
	Name                 string  `json:"name"`
	Seed                 uint64  `json:"seed"`
	Repeat               uint64  `json:"repeat"`
	RunTimeout           string  `json:"run_timeout"`
	OverallTimeout       string  `json:"overall_timeout"`
	TerminateGrace       string  `json:"terminate_grace"`
	OutputBytes          uint64  `json:"output_bytes"`
	WorldTransitionBytes uint64  `json:"world_transition_bytes"`
	Suites               []Suite `json:"suites"`
}

type Suite struct {
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
	Schema         string        `json:"schema"`
	Name           string        `json:"name"`
	Qualified      bool          `json:"qualified"`
	Selected       uint64        `json:"selected"`
	Completed      uint64        `json:"completed"`
	ManifestSHA256 record.SHA256 `json:"manifest_sha256"`
	Manifest       Manifest      `json:"manifest"`
	Suites         []SuiteReport `json:"suites"`
}

type SuiteReport struct {
	Name           string         `json:"name"`
	Expected       Expectation    `json:"expected"`
	ExpectationMet bool           `json:"expectation_met"`
	Classification string         `json:"classification,omitempty"`
	ExitCode       int            `json:"exit_code"`
	Command        []string       `json:"command"`
	StdoutSHA256   record.SHA256  `json:"stdout_sha256,omitempty"`
	StderrSHA256   record.SHA256  `json:"stderr_sha256,omitempty"`
	ReportPath     string         `json:"report_path,omitempty"`
	ReportSHA256   record.SHA256  `json:"report_sha256,omitempty"`
	Report         qualify.Report `json:"report,omitempty"`
	Error          string         `json:"error,omitempty"`
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

type qualificationEvent struct {
	Schema         string          `json:"schema"`
	Type           string          `json:"type"`
	Classification string          `json:"classification,omitempty"`
	Message        string          `json:"message,omitempty"`
	ReportPath     string          `json:"report_path,omitempty"`
	Report         json.RawMessage `json:"report,omitempty"`
}

var setNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var testNamePattern = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)

func LoadManifest(path string) (Manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read qualification set manifest: %w", err)
	}
	if len(contents) > maximumManifestBytes {
		return Manifest{}, fmt.Errorf("qualification set manifest exceeds %d bytes", maximumManifestBytes)
	}
	var manifest Manifest
	if err := record.StrictDecode(contents, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode qualification set manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
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
	manifestBytes, err := record.CanonicalJSON(manifest)
	if err != nil {
		return SetReport{}, err
	}
	report := SetReport{
		Schema: ReportSchema, Name: manifest.Name, Selected: uint64(len(manifest.Suites)), ManifestSHA256: record.HashBytes(manifestBytes),
		Manifest: manifest, Suites: make([]SuiteReport, 0, len(manifest.Suites)),
	}
	failed := make([]string, 0)
	for _, suite := range manifest.Suites {
		if err := ctx.Err(); err != nil {
			failed = append(failed, suite.Name)
			break
		}
		command := suiteCommand(config, manifest, suite)
		result := config.Execute(ctx, command)
		suiteReport := SuiteReport{
			Name: suite.Name, Expected: suite.Expectation, ExitCode: result.ExitCode,
			Command: append([]string{command.Executable}, command.Args...), StdoutSHA256: record.HashBytes(result.Stdout), StderrSHA256: record.HashBytes(result.Stderr),
		}
		opened, classification, retainedPath, retainedDigest, evidenceErr := retainedQualification(config.ArtifactRoot, command, result)
		if evidenceErr != nil {
			suiteReport.Error = evidenceErr.Error()
			if result.Err != nil {
				suiteReport.Error = errors.Join(evidenceErr, result.Err).Error()
			}
		} else {
			suiteReport.Classification = classification
			suiteReport.ReportPath = retainedPath
			suiteReport.ReportSHA256 = retainedDigest
			suiteReport.Report = opened
			suiteReport.ExpectationMet = matchesExpectation(suite.Expectation, classification, opened, result.ExitCode)
			report.Completed++
		}
		if !suiteReport.ExpectationMet {
			failed = append(failed, suite.Name)
		}
		report.Suites = append(report.Suites, suiteReport)
	}
	report.Qualified = report.Completed == report.Selected && len(failed) == 0
	if err := writeReport(config.OutputPath, report); err != nil {
		return report, err
	}
	if !report.Qualified {
		return report, &ExpectationError{Suites: failed}
	}
	return report, nil
}

func OpenReport(path string) (SetReport, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() <= 0 || info.Size() > maximumReportBytes {
		return SetReport{}, errors.Join(errors.New("qualification set report must be a private bounded regular file"), err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return SetReport{}, fmt.Errorf("read qualification set report: %w", err)
	}
	if len(contents) > maximumReportBytes || len(contents) == 0 || contents[len(contents)-1] != '\n' {
		return SetReport{}, fmt.Errorf("qualification set report framing is invalid")
	}
	contents = contents[:len(contents)-1]
	var report SetReport
	if err := record.StrictDecode(contents, &report); err != nil {
		return SetReport{}, fmt.Errorf("decode qualification set report: %w", err)
	}
	canonical, err := record.CanonicalJSON(report)
	if err != nil || !bytes.Equal(canonical, contents) {
		return SetReport{}, errors.Join(errors.New("qualification set report is not canonical"), err)
	}
	if err := validateSetReport(report); err != nil {
		return SetReport{}, err
	}
	return report, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema || !setNamePattern.MatchString(manifest.Name) || manifest.Repeat < 2 || manifest.Repeat > 32 || len(manifest.Suites) == 0 || len(manifest.Suites) > 64 || manifest.OutputBytes == 0 || manifest.WorldTransitionBytes == 0 {
		return errors.New("qualification set manifest identity or bounds are invalid")
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
		if !setNamePattern.MatchString(suite.Name) || !strings.HasPrefix(suite.Package, "./") || packagePath == "" || filepath.ToSlash(filepath.Clean(filepath.FromSlash(packagePath))) != packagePath || strings.HasPrefix(packagePath, "../") || !testNamePattern.MatchString(suite.Test) {
			return fmt.Errorf("qualification suite %d identity is invalid", index)
		}
		if _, duplicate := seen[suite.Name]; duplicate {
			return fmt.Errorf("qualification suite name is duplicated: %s", suite.Name)
		}
		seen[suite.Name] = struct{}{}
		if !sortedUnique(suite.BuildTags) || !sortedUnique(suite.Environment) || !sortedUnique(suite.RequiredProbes) {
			return fmt.Errorf("qualification suite %s lists must be sorted and unique", suite.Name)
		}
		for _, mount := range suite.ReadOnlyMounts {
			if mount.Source == "" || mount.Target == "" || strings.ContainsAny(mount.Source+mount.Target, "\x00\n") {
				return fmt.Errorf("qualification suite %s has an invalid read-only mount", suite.Name)
			}
		}
		if err := validateExpectation(suite.Expectation); err != nil {
			return fmt.Errorf("qualification suite %s: %w", suite.Name, err)
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

func suiteCommand(config Config, manifest Manifest, suite Suite) Command {
	args := []string{
		"qualify", "--json", "--seed=" + strconv.FormatUint(manifest.Seed, 10), "--repeat=" + strconv.FormatUint(manifest.Repeat, 10),
		"--run-timeout=" + manifest.RunTimeout, "--overall-timeout=" + manifest.OverallTimeout, "--terminate-grace=" + manifest.TerminateGrace,
		"--output-limit=" + strconv.FormatUint(manifest.OutputBytes, 10), "--world-transition-limit=" + strconv.FormatUint(manifest.WorldTransitionBytes, 10),
		"--artifacts=" + config.ArtifactRoot,
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
	return Command{Executable: config.GomadPath, Args: args, Dir: config.WorkingDir, ArtifactRoot: config.ArtifactRoot}
}

func executeCommand(ctx context.Context, command Command) CommandResult {
	process := exec.CommandContext(ctx, command.Executable, command.Args...)
	process.Dir = command.Dir
	stdout := &boundedBuffer{maximum: maximumCommandOutputBytes}
	stderr := &boundedBuffer{maximum: maximumCommandOutputBytes}
	process.Stdout = stdout
	process.Stderr = stderr
	err := process.Run()
	result := CommandResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Err: err}
	if err == nil {
		return result
	}
	result.ExitCode = -1
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
		result.Err = nil
	}
	if stdout.truncated || stderr.truncated {
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
	event, err := resultEvent(result.Stdout)
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
	classification := qualificationClassification(opened)
	if event.Classification != classification || result.ExitCode != classificationStatus(classification) {
		return qualify.Report{}, "", "", "", fmt.Errorf("qualification result classification or status is inconsistent: %s/%d", event.Classification, result.ExitCode)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return qualify.Report{}, "", "", "", err
	}
	return opened, classification, path, record.HashBytes(contents), nil
}

func resultEvent(contents []byte) (qualificationEvent, error) {
	var result qualificationEvent
	for _, line := range bytes.Split(bytes.TrimSuffix(contents, []byte{'\n'}), []byte{'\n'}) {
		if len(line) == 0 {
			return qualificationEvent{}, errors.New("qualification event stream contains an empty record")
		}
		var event qualificationEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return qualificationEvent{}, fmt.Errorf("decode qualification event: %w", err)
		}
		if event.Schema != "gomadv3.qualify-event/v1" {
			return qualificationEvent{}, fmt.Errorf("unsupported qualification event schema %q", event.Schema)
		}
		switch event.Type {
		case "progress":
		case "result":
			if result.Type != "" || event.ReportPath == "" || event.Classification == "" {
				return qualificationEvent{}, errors.New("qualification result event is invalid or duplicated")
			}
			result = event
		case "error":
			return qualificationEvent{}, fmt.Errorf("unretained qualification error %s: %s", event.Classification, event.Message)
		default:
			return qualificationEvent{}, fmt.Errorf("unknown qualification event type %q", event.Type)
		}
	}
	if result.Type == "" {
		return qualificationEvent{}, errors.New("qualification event stream has no retained result")
	}
	return result, nil
}

func qualificationClassification(report qualify.Report) string {
	if report.Failure != nil {
		return report.Failure.Classification
	}
	if !report.Deterministic {
		return "nondeterministic"
	}
	if report.Replay != nil && !report.Replay.Match {
		return "replay_divergence"
	}
	if !report.TargetSuccess {
		return "target_failure"
	}
	return "qualified"
}

func classificationStatus(classification string) int {
	switch classification {
	case "qualified":
		return 0
	case "target_failure", "nondeterministic", "replay_divergence", "semantic_coverage_failure":
		return 1
	case "unsupported_target", "invalid_input":
		return 2
	default:
		return 3
	}
}

func matchesExpectation(expected Expectation, classification string, report qualify.Report, exitCode int) bool {
	if expected.Classification != classification || exitCode != classificationStatus(classification) {
		return false
	}
	switch classification {
	case "qualified":
		return report.Qualified && report.Deterministic && report.TargetSuccess && report.Failure == nil
	case "unsupported_target":
		return report.Failure != nil && report.Failure.ImportPath == expected.ImportPath && report.Failure.Capability == expected.Capability
	case "target_failure":
		return report.Deterministic && !report.TargetSuccess && report.Replay != nil && report.Replay.Attempted && report.Replay.Match
	case "nondeterministic":
		return !report.Deterministic
	case "replay_divergence":
		return report.Replay != nil && report.Replay.Attempted && !report.Replay.Match
	default:
		return false
	}
}

func writeReport(path string, report SetReport) (retErr error) {
	if err := validateSetReport(report); err != nil {
		return err
	}
	contents, err := record.CanonicalJSON(report)
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".qualification-set-*.partial")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		retErr = errors.Join(retErr, os.Remove(temporaryPath))
	}()
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func validateSetReport(report SetReport) error {
	if report.Schema != ReportSchema || report.Name == "" || report.Selected != uint64(len(report.Manifest.Suites)) || report.Completed > report.Selected || len(report.Suites) > int(report.Selected) {
		return errors.New("qualification set report identity or counts are invalid")
	}
	if err := validateManifest(report.Manifest); err != nil {
		return err
	}
	manifestBytes, err := record.CanonicalJSON(report.Manifest)
	if err != nil || record.HashBytes(manifestBytes) != report.ManifestSHA256 {
		return errors.Join(errors.New("qualification set manifest digest is invalid"), err)
	}
	completed := uint64(0)
	qualified := len(report.Suites) == int(report.Selected)
	for index, suite := range report.Suites {
		manifestSuite := report.Manifest.Suites[index]
		if suite.Name != manifestSuite.Name || suite.Expected != manifestSuite.Expectation || len(suite.Command) == 0 || suite.Command[0] == "" {
			return fmt.Errorf("qualification set suite report %d identity is invalid", index)
		}
		if suite.ReportPath != "" {
			completed++
			if suite.Report.Schema != qualify.ReportSchema || suite.Classification != qualificationClassification(suite.Report) {
				return fmt.Errorf("qualification set suite report %s evidence is invalid", suite.Name)
			}
			contents, err := record.CanonicalJSON(suite.Report)
			if err != nil {
				return err
			}
			contents = append(contents, '\n')
			if record.HashBytes(contents) != suite.ReportSHA256 {
				return fmt.Errorf("qualification set suite report %s digest is invalid", suite.Name)
			}
		}
		qualified = qualified && suite.ExpectationMet
	}
	if completed != report.Completed || report.Qualified != (qualified && completed == report.Selected) {
		return errors.New("qualification set result is inconsistent")
	}
	return nil
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

type boundedBuffer struct {
	bytes.Buffer
	maximum   int
	truncated bool
}

func (buffer *boundedBuffer) Write(contents []byte) (int, error) {
	remaining := buffer.maximum - buffer.Len()
	if remaining <= 0 {
		buffer.truncated = true
		return len(contents), nil
	}
	accepted := contents
	if len(accepted) > remaining {
		accepted = accepted[:remaining]
		buffer.truncated = true
	}
	_, err := buffer.Buffer.Write(accepted)
	return len(contents), err
}
