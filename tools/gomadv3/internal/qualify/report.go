package qualify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/runner"
)

const (
	ReportSchema            = "gomadv3.qualification/v3"
	PreviousReportSchema    = "gomadv3.qualification/v2"
	LegacyReportSchema      = "gomadv3.qualification/v1"
	ChoiceReplayNone        = "none"
	ChoiceReplayExact       = "exact"
	ChoiceReplayDiverged    = "diverged"
	ChoiceReplayUnavailable = "unavailable"
)

const maximumReportBytes = 16 << 20

type Run struct {
	BatchPath    string
	ArtifactPath string
	Evidence     runner.RunEvidence
	Replay       *Replay
}

type Input struct {
	Command []string
	Runs    []Run
	Replay  *Replay
}

type Replay struct {
	ArtifactPath       string `json:"artifact_path"`
	Attempted          bool   `json:"attempted"`
	Match              bool   `json:"match"`
	Diagnostic         bool   `json:"diagnostic"`
	Divergence         string `json:"divergence,omitempty"`
	ChoiceReplayStatus string `json:"choice_replay_status,omitempty"`
}

type Failure struct {
	Classification string              `json:"classification"`
	Message        string              `json:"message"`
	Iteration      record.Uint64String `json:"iteration"`
	ImportPath     string              `json:"import_path,omitempty"`
	Capability     string              `json:"capability,omitempty"`
}

type RunReport struct {
	BatchPath      string        `json:"batch_path"`
	ArtifactPath   string        `json:"artifact_path,omitempty"`
	EvidenceDigest record.SHA256 `json:"evidence_digest"`
	Replay         *Replay       `json:"replay,omitempty"`
}

type Report struct {
	Schema          string              `json:"schema"`
	Qualified       bool                `json:"qualified"`
	Deterministic   bool                `json:"deterministic"`
	TargetSuccess   bool                `json:"target_success"`
	Seed            record.Uint64String `json:"seed"`
	Repeat          record.Uint64String `json:"repeat"`
	Command         []string            `json:"command"`
	EvidenceDigest  record.SHA256       `json:"evidence_digest,omitempty"`
	Evidence        *runner.RunEvidence `json:"evidence,omitempty"`
	Runs            []RunReport         `json:"runs"`
	FirstDivergence string              `json:"first_divergence,omitempty"`
	Failure         *Failure            `json:"failure,omitempty"`
}

type legacyRunReport struct {
	BatchPath      string        `json:"batch_path"`
	ArtifactPath   string        `json:"artifact_path,omitempty"`
	EvidenceDigest record.SHA256 `json:"evidence_digest"`
}

type legacyReport struct {
	Schema          string              `json:"schema"`
	Qualified       bool                `json:"qualified"`
	Deterministic   bool                `json:"deterministic"`
	TargetSuccess   bool                `json:"target_success"`
	Seed            record.Uint64String `json:"seed"`
	Repeat          record.Uint64String `json:"repeat"`
	Command         []string            `json:"command"`
	EvidenceDigest  record.SHA256       `json:"evidence_digest,omitempty"`
	Evidence        *runner.RunEvidence `json:"evidence,omitempty"`
	Runs            []legacyRunReport   `json:"runs"`
	FirstDivergence string              `json:"first_divergence,omitempty"`
	Replay          *Replay             `json:"replay,omitempty"`
	Failure         *Failure            `json:"failure,omitempty"`
}

func Build(input Input) (Report, error) {
	if len(input.Command) == 0 || input.Command[0] == "" {
		return Report{}, fmt.Errorf("qualification command is required")
	}
	if len(input.Runs) < 2 {
		return Report{}, fmt.Errorf("qualification requires at least two runs")
	}
	baseline, err := cloneEvidence(input.Runs[0].Evidence)
	if err != nil {
		return Report{}, fmt.Errorf("validate run evidence 0: %w", err)
	}
	if baseline.Schema != runner.RunEvidenceSchema {
		return Report{}, fmt.Errorf("run evidence 0 has unsupported schema %q", baseline.Schema)
	}
	report := Report{
		Schema: ReportSchema, Deterministic: true, TargetSuccess: true,
		Seed: baseline.Seed, Repeat: record.Uint64String(len(input.Runs)), Command: append([]string(nil), input.Command...), Evidence: &baseline,
		Runs: make([]RunReport, 0, len(input.Runs)),
	}
	legacyReplayAssigned := false
	replayOK := true
	for index, run := range input.Runs {
		if run.BatchPath == "" {
			return Report{}, fmt.Errorf("qualification run %d has no batch path", index)
		}
		if run.Evidence.Schema != runner.RunEvidenceSchema {
			return Report{}, fmt.Errorf("run evidence %d has unsupported schema %q", index, run.Evidence.Schema)
		}
		if run.Evidence.Seed != baseline.Seed {
			return Report{}, fmt.Errorf("qualification run %d has seed %d, want %d", index, run.Evidence.Seed, baseline.Seed)
		}
		digest, digestErr := evidenceDigest(run.Evidence)
		if digestErr != nil {
			return Report{}, fmt.Errorf("hash run evidence %d: %w", index, digestErr)
		}
		if index == 0 {
			report.EvidenceDigest = digest
		} else if digest != report.EvidenceDigest {
			report.Deterministic = false
			if report.FirstDivergence == "" {
				report.FirstDivergence = firstDivergence(baseline, run.Evidence)
			}
		}
		if run.Evidence.Outcome.Domain != "success" {
			report.TargetSuccess = false
		}
		replayEvidence := run.Replay
		if replayEvidence == nil && input.Replay != nil && !legacyReplayAssigned && run.Evidence.Outcome.Domain != "success" {
			replayEvidence = input.Replay
			legacyReplayAssigned = true
		}
		copiedReplay, replayErr := cloneReplay(replayEvidence, run.ArtifactPath)
		if replayErr != nil {
			return Report{}, fmt.Errorf("validate run replay %d: %w", index, replayErr)
		}
		if copiedReplay != nil && !copiedReplay.Match {
			replayOK = false
		}
		if copiedReplay != nil && run.Evidence.Choices != nil && copiedReplay.Match && copiedReplay.ChoiceReplayStatus != ChoiceReplayExact {
			return Report{}, fmt.Errorf("validate run replay %d: exact choice replay evidence is required", index)
		}
		report.Runs = append(report.Runs, RunReport{BatchPath: run.BatchPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, Replay: copiedReplay})
	}
	report.Qualified = report.Deterministic && report.TargetSuccess && replayOK
	return report, nil
}

func BuildFailure(command []string, seed uint64, repeat uint64, completed []Run, failure Failure) (Report, error) {
	if len(command) == 0 || command[0] == "" {
		return Report{}, fmt.Errorf("qualification command is required")
	}
	if repeat < 2 {
		return Report{}, fmt.Errorf("qualification requires at least two runs")
	}
	if failure.Classification == "" || failure.Message == "" || uint64(failure.Iteration) == 0 || uint64(failure.Iteration) > repeat {
		return Report{}, fmt.Errorf("qualification failure is incomplete")
	}
	report := Report{
		Schema: ReportSchema, Seed: record.Uint64String(seed), Repeat: record.Uint64String(repeat), Command: append([]string(nil), command...),
		Runs: make([]RunReport, 0, len(completed)), Failure: &failure,
	}
	for index, run := range completed {
		if run.BatchPath == "" || run.Evidence.Schema != runner.RunEvidenceSchema || uint64(run.Evidence.Seed) != seed {
			return Report{}, fmt.Errorf("completed qualification run %d is invalid", index)
		}
		digest, err := evidenceDigest(run.Evidence)
		if err != nil {
			return Report{}, fmt.Errorf("hash completed qualification run %d: %w", index, err)
		}
		if report.Evidence == nil {
			cloned, cloneErr := cloneEvidence(run.Evidence)
			if cloneErr != nil {
				return Report{}, cloneErr
			}
			report.Evidence = &cloned
			report.EvidenceDigest = digest
		} else if digest != report.EvidenceDigest && report.FirstDivergence == "" {
			report.FirstDivergence = firstDivergence(*report.Evidence, run.Evidence)
		}
		replayEvidence, replayErr := cloneReplay(run.Replay, run.ArtifactPath)
		if replayErr != nil {
			return Report{}, fmt.Errorf("validate completed qualification replay %d: %w", index, replayErr)
		}
		if replayEvidence != nil && run.Evidence.Choices != nil && replayEvidence.Match && replayEvidence.ChoiceReplayStatus != ChoiceReplayExact {
			return Report{}, fmt.Errorf("validate completed qualification replay %d: exact choice replay evidence is required", index)
		}
		report.Runs = append(report.Runs, RunReport{BatchPath: run.BatchPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, Replay: replayEvidence})
	}
	return report, nil
}

func Write(artifactRoot string, report Report) (string, error) {
	if artifactRoot == "" {
		return "", fmt.Errorf("artifact root is required")
	}
	if err := validateReport(report); err != nil {
		return "", err
	}
	encoded, err := record.CanonicalJSON(report)
	if err != nil {
		return "", fmt.Errorf("encode qualification report: %w", err)
	}
	encoded = append(encoded, '\n')
	root := filepath.Join(artifactRoot, "qualifications", "v3")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create qualification report directory: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", fmt.Errorf("make qualification report directory private: %w", err)
	}
	temporary, err := os.CreateTemp(root, ".qualification-*.partial")
	if err != nil {
		return "", fmt.Errorf("create qualification report staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return "", fmt.Errorf("make qualification report private: %w", err)
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write qualification report: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync qualification report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close qualification report: %w", err)
	}
	name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(temporaryPath), ".qualification-"), ".partial")
	path := filepath.Join(root, "qualification-"+name+".json")
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("publish qualification report: %w", err)
	}
	directory, err := os.Open(root)
	if err != nil {
		return path, fmt.Errorf("open qualification report directory: %w", err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return path, fmt.Errorf("sync qualification report directory: %w", syncErr)
	}
	if closeErr != nil {
		return path, fmt.Errorf("close qualification report directory: %w", closeErr)
	}
	return path, nil
}

func Open(path string) (Report, error) {
	file, err := os.Open(path)
	if err != nil {
		return Report{}, fmt.Errorf("open qualification report: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Report{}, fmt.Errorf("stat qualification report: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maximumReportBytes {
		return Report{}, fmt.Errorf("qualification report must be a regular file no larger than %d bytes", maximumReportBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximumReportBytes+1))
	if err != nil {
		return Report{}, fmt.Errorf("read qualification report: %w", err)
	}
	data = bytes.TrimSuffix(data, []byte{'\n'})
	return Decode(data)
}

func Decode(data []byte) (Report, error) {
	if len(data) == 0 || len(data) > maximumReportBytes {
		return Report{}, fmt.Errorf("qualification report must be between 1 and %d bytes", maximumReportBytes)
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return Report{}, fmt.Errorf("decode qualification report schema: %w", err)
	}
	var report Report
	switch header.Schema {
	case ReportSchema:
		if err := record.DecodeCanonicalJSON(data, &report); err != nil {
			return Report{}, fmt.Errorf("decode qualification report: %w", err)
		}
	case PreviousReportSchema:
		if err := record.DecodeCanonicalJSON(data, &report); err != nil {
			return Report{}, fmt.Errorf("decode previous qualification report: %w", err)
		}
		normalizePreviousReport(&report)
	case LegacyReportSchema:
		var legacy legacyReport
		if err := record.DecodeCanonicalJSON(data, &legacy); err != nil {
			return Report{}, fmt.Errorf("decode legacy qualification report: %w", err)
		}
		var normalizeErr error
		report, normalizeErr = normalizeLegacyReport(legacy)
		if normalizeErr != nil {
			return Report{}, normalizeErr
		}
	default:
		return Report{}, fmt.Errorf("unsupported qualification report schema %q", header.Schema)
	}
	if err := validateReport(report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func normalizePreviousReport(report *Report) {
	report.Schema = ReportSchema
	legacyChoice := report.Evidence != nil && report.Evidence.Choices != nil
	for index := range report.Runs {
		replayed := report.Runs[index].Replay
		if replayed == nil {
			continue
		}
		if legacyChoice {
			replayed.Match = false
			replayed.Divergence = "choice_profile.replay_unavailable"
			replayed.ChoiceReplayStatus = ChoiceReplayUnavailable
			report.Qualified = false
		} else {
			replayed.ChoiceReplayStatus = ChoiceReplayNone
		}
	}
}

func normalizeLegacyReport(legacy legacyReport) (Report, error) {
	report := Report{
		Schema: LegacyReportSchema, Qualified: legacy.Qualified, Deterministic: legacy.Deterministic,
		TargetSuccess: legacy.TargetSuccess, Seed: legacy.Seed, Repeat: legacy.Repeat,
		Command: append([]string(nil), legacy.Command...), EvidenceDigest: legacy.EvidenceDigest,
		Evidence: legacy.Evidence, FirstDivergence: legacy.FirstDivergence, Failure: legacy.Failure,
		Runs: make([]RunReport, len(legacy.Runs)),
	}
	for index, run := range legacy.Runs {
		report.Runs[index] = RunReport{BatchPath: run.BatchPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: run.EvidenceDigest}
	}
	if err := validateLegacyReport(report, legacy.Replay); err != nil {
		return Report{}, err
	}
	if legacy.Replay != nil {
		matched := false
		for index := range report.Runs {
			if report.Runs[index].ArtifactPath == legacy.Replay.ArtifactPath {
				copied := *legacy.Replay
				report.Runs[index].Replay = &copied
				matched = true
				break
			}
		}
		if !matched {
			return Report{}, errors.New("legacy qualification replay does not match a retained artifact")
		}
	}
	normalizePreviousReport(&report)
	return report, nil
}

func validateLegacyReport(report Report, replay *Replay) error {
	if report.Schema != LegacyReportSchema {
		return fmt.Errorf("unsupported qualification report schema %q", report.Schema)
	}
	report.Schema = PreviousReportSchema
	if replay != nil {
		if !replay.Attempted || replay.Match && replay.Divergence != "" || !replay.Match && replay.Divergence == "" {
			return errors.New("legacy qualification replay is invalid")
		}
		for index := range report.Runs {
			if report.Runs[index].ArtifactPath == replay.ArtifactPath {
				copied := *replay
				report.Runs[index].Replay = &copied
				break
			}
		}
	}
	normalizePreviousReport(&report)
	return validateReport(report)
}

func validateReport(report Report) error {
	if report.Schema != ReportSchema {
		return fmt.Errorf("unsupported qualification report schema %q", report.Schema)
	}
	if len(report.Command) == 0 || report.Command[0] == "" || uint64(report.Repeat) < 2 {
		return fmt.Errorf("qualification report command or repetition count is invalid")
	}
	if report.Failure != nil {
		if report.Qualified || report.Deterministic || report.TargetSuccess || report.Failure.Classification == "" || report.Failure.Message == "" || uint64(report.Failure.Iteration) == 0 || report.Failure.Iteration > report.Repeat || len(report.Runs) > int(report.Repeat) {
			return fmt.Errorf("qualification failure result is inconsistent")
		}
		if len(report.Runs) == 0 {
			if report.Evidence != nil || report.EvidenceDigest != "" {
				return fmt.Errorf("qualification failure has evidence without completed runs")
			}
			return nil
		}
	}
	if report.Failure == nil && (len(report.Runs) < 2 || uint64(report.Repeat) != uint64(len(report.Runs))) {
		return fmt.Errorf("qualification report repetition count is invalid")
	}
	if report.Evidence == nil || report.Evidence.Schema != runner.RunEvidenceSchema && report.Evidence.Schema != runner.LegacyRunEvidenceSchema || report.Evidence.Seed != report.Seed {
		return fmt.Errorf("qualification baseline evidence identity is invalid")
	}
	digest, err := evidenceDigest(*report.Evidence)
	if err != nil {
		return fmt.Errorf("hash qualification baseline evidence: %w", err)
	}
	if digest != report.EvidenceDigest {
		return fmt.Errorf("qualification baseline evidence digest is invalid")
	}
	deterministic := true
	for index, run := range report.Runs {
		if run.BatchPath == "" || run.EvidenceDigest == "" {
			return fmt.Errorf("qualification run %d identity is invalid", index)
		}
		if run.EvidenceDigest != report.EvidenceDigest {
			deterministic = false
		}
	}
	if report.Failure != nil {
		return nil
	}
	if report.Deterministic != deterministic || report.Deterministic != (report.FirstDivergence == "") {
		return fmt.Errorf("qualification determinism result is inconsistent")
	}
	if report.Deterministic && report.TargetSuccess != (report.Evidence.Outcome.Domain == "success") {
		return fmt.Errorf("qualification target result is inconsistent with deterministic evidence")
	}
	replayOK := true
	for index, run := range report.Runs {
		if _, err := cloneReplay(run.Replay, run.ArtifactPath); err != nil {
			return fmt.Errorf("qualification run %d replay is invalid: %w", index, err)
		}
		if run.Replay != nil && !run.Replay.Match {
			replayOK = false
		}
		if run.Replay != nil && report.Evidence.Choices != nil && run.Replay.Match && run.Replay.ChoiceReplayStatus != ChoiceReplayExact {
			return fmt.Errorf("qualification run %d lacks exact choice replay evidence", index)
		}
	}
	if report.Qualified != (report.Deterministic && report.TargetSuccess && replayOK) {
		return fmt.Errorf("qualification result is inconsistent")
	}
	return nil
}

func cloneReplay(replay *Replay, artifactPath string) (*Replay, error) {
	if replay == nil {
		return nil, nil
	}
	if artifactPath == "" || replay.ArtifactPath != artifactPath || !replay.Attempted || replay.Match && replay.Divergence != "" || !replay.Match && replay.Divergence == "" {
		return nil, errors.New("replay does not match its retained artifact and result")
	}
	switch replay.ChoiceReplayStatus {
	case "", ChoiceReplayNone:
	case ChoiceReplayExact:
		if !replay.Match {
			return nil, errors.New("exact choice replay status requires a match")
		}
	case ChoiceReplayDiverged:
		if replay.Match {
			return nil, errors.New("diverged choice replay status cannot match")
		}
	case ChoiceReplayUnavailable:
		if replay.Match || replay.Divergence != "choice_profile.replay_unavailable" {
			return nil, errors.New("unavailable choice replay status is inconsistent")
		}
	default:
		return nil, errors.New("choice replay status is invalid")
	}
	copied := *replay
	return &copied, nil
}

func evidenceDigest(evidence runner.RunEvidence) (record.SHA256, error) {
	encoded, err := record.CanonicalJSON(evidence)
	if err != nil {
		return "", err
	}
	domain := RunEvidenceDigestDomain
	if evidence.Schema == runner.LegacyRunEvidenceSchema {
		domain = LegacyRunEvidenceDigestDomain
	}
	return record.DomainHash(domain, encoded), nil
}

const (
	RunEvidenceDigestDomain       = "gomadv3.qualification-evidence/v2"
	LegacyRunEvidenceDigestDomain = "gomadv3.qualification-evidence/v1"
)

func cloneEvidence(evidence runner.RunEvidence) (runner.RunEvidence, error) {
	encoded, err := record.CanonicalJSON(evidence)
	if err != nil {
		return runner.RunEvidence{}, err
	}
	var cloned runner.RunEvidence
	if err := record.StrictDecode(encoded, &cloned); err != nil {
		return runner.RunEvidence{}, err
	}
	return cloned, nil
}

func firstDivergence(expected, actual runner.RunEvidence) string {
	fields := []struct {
		name     string
		expected any
		actual   any
	}{
		{"schema", expected.Schema, actual.Schema},
		{"seed", expected.Seed, actual.Seed},
		{"runner_build", expected.RunnerBuild, actual.RunnerBuild},
		{"toolchain", expected.Toolchain, actual.Toolchain},
		{"target", expected.Target, actual.Target},
		{"io_profile", expected.IOProfile, actual.IOProfile},
		{"environment", expected.Environment, actual.Environment},
		{"limits", expected.Limits, actual.Limits},
		{"outcome", expected.Outcome, actual.Outcome},
		{"group_gone", expected.GroupGone, actual.GroupGone},
		{"stdout.full_sha256", expected.Stdout.FullSHA256, actual.Stdout.FullSHA256},
		{"stdout", expected.Stdout, actual.Stdout},
		{"stderr.full_sha256", expected.Stderr.FullSHA256, actual.Stderr.FullSHA256},
		{"stderr", expected.Stderr, actual.Stderr},
		{"io_transcript.sha256", expected.IOTranscriptSHA256, actual.IOTranscriptSHA256},
		{"io_transcript.records", expected.IOTranscriptRecords, actual.IOTranscriptRecords},
		{"io_transcript.complete", expected.IOTranscriptComplete, actual.IOTranscriptComplete},
		{"choices", expected.Choices, actual.Choices},
		{"world", expected.World, actual.World},
		{"read_only_mounts_sha256", expected.ReadOnlyMountsSHA256, actual.ReadOnlyMountsSHA256},
		{"semantic_coverage", expected.SemanticCoverage, actual.SemanticCoverage},
	}
	for _, field := range fields {
		if !reflect.DeepEqual(field.expected, field.actual) {
			return field.name
		}
	}
	return "evidence"
}
