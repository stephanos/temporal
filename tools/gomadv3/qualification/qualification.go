package qualification

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

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner"
	"go.temporal.io/server/tools/gomadv3/target"
)

const (
	QualificationReportSchema         = "gomadv3.qualification/v4"
	PriorQualificationReportSchema    = "gomadv3.qualification/v3"
	PreviousQualificationReportSchema = "gomadv3.qualification/v2"
	LegacyQualificationReportSchema   = "gomadv3.qualification/v1"
	ChoiceReplayNone                  = "none"
	ChoiceReplayExact                 = "exact"
	ChoiceReplayDiverged              = "diverged"
	ChoiceReplayUnavailable           = "unavailable"
)

const maximumQualificationReportBytes = 16 << 20

type QualificationExecution struct {
	CampaignPath string
	ArtifactPath string
	Evidence     runner.ExecutionEvidence
	Replay       *QualificationReplay
}

type QualificationInput struct {
	Command []string
	Runs    []QualificationExecution
	Replay  *QualificationReplay
}

type QualificationReplay struct {
	ArtifactPath       string `json:"artifact_path"`
	Attempted          bool   `json:"attempted"`
	Match              bool   `json:"match"`
	Diagnostic         bool   `json:"diagnostic"`
	Divergence         string `json:"divergence,omitempty"`
	ChoiceReplayStatus string `json:"choice_replay_status,omitempty"`
}

type QualificationFailure struct {
	Classification string                `json:"classification"`
	Message        string                `json:"message"`
	Iteration      evidence.Uint64String `json:"iteration"`
	ImportPath     string                `json:"import_path,omitempty"`
	Capability     string                `json:"capability,omitempty"`
}

type QualificationExecutionReport struct {
	CampaignPath   string               `json:"batch_path"`
	ArtifactPath   string               `json:"artifact_path,omitempty"`
	EvidenceDigest evidence.SHA256      `json:"evidence_digest"`
	Replay         *QualificationReplay `json:"replay,omitempty"`
}

type QualificationReport struct {
	Schema          string                         `json:"schema"`
	Qualified       bool                           `json:"qualified"`
	Deterministic   bool                           `json:"deterministic"`
	TargetSuccess   bool                           `json:"target_success"`
	Seed            evidence.Uint64String          `json:"seed"`
	Repeat          evidence.Uint64String          `json:"repeat"`
	Command         []string                       `json:"command"`
	EvidenceDigest  evidence.SHA256                `json:"evidence_digest,omitempty"`
	Evidence        *runner.ExecutionEvidence      `json:"evidence,omitempty"`
	Runs            []QualificationExecutionReport `json:"runs"`
	FirstDivergence string                         `json:"first_divergence,omitempty"`
	Failure         *QualificationFailure          `json:"failure,omitempty"`
}

type legacyRunReport struct {
	CampaignPath   string          `json:"batch_path"`
	ArtifactPath   string          `json:"artifact_path,omitempty"`
	EvidenceDigest evidence.SHA256 `json:"evidence_digest"`
}

type legacyReport struct {
	Schema          string                    `json:"schema"`
	Qualified       bool                      `json:"qualified"`
	Deterministic   bool                      `json:"deterministic"`
	TargetSuccess   bool                      `json:"target_success"`
	Seed            evidence.Uint64String     `json:"seed"`
	Repeat          evidence.Uint64String     `json:"repeat"`
	Command         []string                  `json:"command"`
	EvidenceDigest  evidence.SHA256           `json:"evidence_digest,omitempty"`
	Evidence        *runner.ExecutionEvidence `json:"evidence,omitempty"`
	Runs            []legacyRunReport         `json:"runs"`
	FirstDivergence string                    `json:"first_divergence,omitempty"`
	Replay          *QualificationReplay      `json:"replay,omitempty"`
	Failure         *QualificationFailure     `json:"failure,omitempty"`
}

func BuildQualificationReport(input QualificationInput) (QualificationReport, error) {
	if len(input.Command) == 0 || input.Command[0] == "" {
		return QualificationReport{}, fmt.Errorf("qualification command is required")
	}
	if len(input.Runs) < 2 {
		return QualificationReport{}, fmt.Errorf("qualification requires at least two runs")
	}
	baseline, err := cloneEvidence(input.Runs[0].Evidence)
	if err != nil {
		return QualificationReport{}, fmt.Errorf("validate run evidence 0: %w", err)
	}
	if baseline.Schema != runner.ExecutionEvidenceSchema {
		return QualificationReport{}, fmt.Errorf("run evidence 0 has unsupported schema %q", baseline.Schema)
	}
	report := QualificationReport{
		Schema: QualificationReportSchema, Deterministic: true, TargetSuccess: true,
		Seed: baseline.Seed, Repeat: evidence.Uint64String(len(input.Runs)), Command: append([]string(nil), input.Command...), Evidence: &baseline,
		Runs: make([]QualificationExecutionReport, 0, len(input.Runs)),
	}
	legacyReplayAssigned := false
	replayOK := true
	for index, run := range input.Runs {
		if run.CampaignPath == "" {
			return QualificationReport{}, fmt.Errorf("qualification run %d has no batch path", index)
		}
		if run.Evidence.Schema != runner.ExecutionEvidenceSchema {
			return QualificationReport{}, fmt.Errorf("run evidence %d has unsupported schema %q", index, run.Evidence.Schema)
		}
		if run.Evidence.Seed != baseline.Seed {
			return QualificationReport{}, fmt.Errorf("qualification run %d has seed %d, want %d", index, run.Evidence.Seed, baseline.Seed)
		}
		digest, digestErr := evidenceDigest(run.Evidence)
		if digestErr != nil {
			return QualificationReport{}, fmt.Errorf("hash run evidence %d: %w", index, digestErr)
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
			return QualificationReport{}, fmt.Errorf("validate run replay %d: %w", index, replayErr)
		}
		if copiedReplay != nil && !copiedReplay.Match {
			replayOK = false
		}
		if copiedReplay != nil && run.Evidence.Choices != nil && copiedReplay.Match && copiedReplay.ChoiceReplayStatus != ChoiceReplayExact {
			return QualificationReport{}, fmt.Errorf("validate run replay %d: exact choice replay evidence is required", index)
		}
		report.Runs = append(report.Runs, QualificationExecutionReport{CampaignPath: run.CampaignPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, Replay: copiedReplay})
	}
	report.Qualified = report.Deterministic && report.TargetSuccess && replayOK
	return report, nil
}

func BuildQualificationFailure(command []string, seed uint64, repeat uint64, completed []QualificationExecution, failure QualificationFailure) (QualificationReport, error) {
	if len(command) == 0 || command[0] == "" {
		return QualificationReport{}, fmt.Errorf("qualification command is required")
	}
	if repeat < 2 {
		return QualificationReport{}, fmt.Errorf("qualification requires at least two runs")
	}
	if failure.Classification == "" || failure.Message == "" || uint64(failure.Iteration) == 0 || uint64(failure.Iteration) > repeat {
		return QualificationReport{}, fmt.Errorf("qualification failure is incomplete")
	}
	report := QualificationReport{
		Schema: QualificationReportSchema, Seed: evidence.Uint64String(seed), Repeat: evidence.Uint64String(repeat), Command: append([]string(nil), command...),
		Runs: make([]QualificationExecutionReport, 0, len(completed)), Failure: &failure,
	}
	for index, run := range completed {
		if run.CampaignPath == "" || run.Evidence.Schema != runner.ExecutionEvidenceSchema || uint64(run.Evidence.Seed) != seed {
			return QualificationReport{}, fmt.Errorf("completed qualification run %d is invalid", index)
		}
		digest, err := evidenceDigest(run.Evidence)
		if err != nil {
			return QualificationReport{}, fmt.Errorf("hash completed qualification run %d: %w", index, err)
		}
		if report.Evidence == nil {
			cloned, cloneErr := cloneEvidence(run.Evidence)
			if cloneErr != nil {
				return QualificationReport{}, cloneErr
			}
			report.Evidence = &cloned
			report.EvidenceDigest = digest
		} else if digest != report.EvidenceDigest && report.FirstDivergence == "" {
			report.FirstDivergence = firstDivergence(*report.Evidence, run.Evidence)
		}
		replayEvidence, replayErr := cloneReplay(run.Replay, run.ArtifactPath)
		if replayErr != nil {
			return QualificationReport{}, fmt.Errorf("validate completed qualification replay %d: %w", index, replayErr)
		}
		if replayEvidence != nil && run.Evidence.Choices != nil && replayEvidence.Match && replayEvidence.ChoiceReplayStatus != ChoiceReplayExact {
			return QualificationReport{}, fmt.Errorf("validate completed qualification replay %d: exact choice replay evidence is required", index)
		}
		report.Runs = append(report.Runs, QualificationExecutionReport{CampaignPath: run.CampaignPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, Replay: replayEvidence})
	}
	return report, nil
}

func WriteQualificationReport(artifactRoot string, report QualificationReport) (string, error) {
	if artifactRoot == "" {
		return "", fmt.Errorf("artifact root is required")
	}
	if err := validateQualificationReport(report); err != nil {
		return "", err
	}
	encoded, err := evidence.CanonicalJSON(report)
	if err != nil {
		return "", fmt.Errorf("encode qualification report: %w", err)
	}
	encoded = append(encoded, '\n')
	root := filepath.Join(artifactRoot, "qualifications", "v4")
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

func OpenQualificationReport(path string) (QualificationReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return QualificationReport{}, fmt.Errorf("open qualification report: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return QualificationReport{}, fmt.Errorf("stat qualification report: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maximumQualificationReportBytes {
		return QualificationReport{}, fmt.Errorf("qualification report must be a regular file no larger than %d bytes", maximumQualificationReportBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximumQualificationReportBytes+1))
	if err != nil {
		return QualificationReport{}, fmt.Errorf("read qualification report: %w", err)
	}
	data = bytes.TrimSuffix(data, []byte{'\n'})
	return DecodeQualificationReport(data)
}

func DecodeQualificationReport(data []byte) (QualificationReport, error) {
	if len(data) == 0 || len(data) > maximumQualificationReportBytes {
		return QualificationReport{}, fmt.Errorf("qualification report must be between 1 and %d bytes", maximumQualificationReportBytes)
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return QualificationReport{}, fmt.Errorf("decode qualification report schema: %w", err)
	}
	var report QualificationReport
	historicalEvidence := false
	switch header.Schema {
	case QualificationReportSchema:
		if err := evidence.DecodeCanonicalJSON(data, &report); err != nil {
			return QualificationReport{}, fmt.Errorf("decode qualification report: %w", err)
		}
	case PriorQualificationReportSchema:
		if err := evidence.DecodeCanonicalJSON(data, &report); err != nil {
			return QualificationReport{}, fmt.Errorf("decode prior qualification report: %w", err)
		}
		report.Schema = QualificationReportSchema
		historicalEvidence = true
	case PreviousQualificationReportSchema:
		if err := evidence.DecodeCanonicalJSON(data, &report); err != nil {
			return QualificationReport{}, fmt.Errorf("decode previous qualification report: %w", err)
		}
		normalizePreviousReport(&report)
		historicalEvidence = true
	case LegacyQualificationReportSchema:
		var legacy legacyReport
		if err := evidence.DecodeCanonicalJSON(data, &legacy); err != nil {
			return QualificationReport{}, fmt.Errorf("decode legacy qualification report: %w", err)
		}
		var normalizeErr error
		report, normalizeErr = normalizeLegacyReport(legacy)
		if normalizeErr != nil {
			return QualificationReport{}, normalizeErr
		}
		historicalEvidence = true
	default:
		return QualificationReport{}, fmt.Errorf("unsupported qualification report schema %q", header.Schema)
	}
	if err := validateQualificationReport(report); err != nil {
		return QualificationReport{}, err
	}
	if historicalEvidence && report.Evidence != nil {
		report.Evidence.Target.CapabilityMode = string(target.CapabilityModeClosure)
	}
	return report, nil
}

func normalizePreviousReport(report *QualificationReport) {
	report.Schema = QualificationReportSchema
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

func normalizeLegacyReport(legacy legacyReport) (QualificationReport, error) {
	report := QualificationReport{
		Schema: LegacyQualificationReportSchema, Qualified: legacy.Qualified, Deterministic: legacy.Deterministic,
		TargetSuccess: legacy.TargetSuccess, Seed: legacy.Seed, Repeat: legacy.Repeat,
		Command: append([]string(nil), legacy.Command...), EvidenceDigest: legacy.EvidenceDigest,
		Evidence: legacy.Evidence, FirstDivergence: legacy.FirstDivergence, Failure: legacy.Failure,
		Runs: make([]QualificationExecutionReport, len(legacy.Runs)),
	}
	for index, run := range legacy.Runs {
		report.Runs[index] = QualificationExecutionReport{CampaignPath: run.CampaignPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: run.EvidenceDigest}
	}
	if err := validateLegacyReport(report, legacy.Replay); err != nil {
		return QualificationReport{}, err
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
			return QualificationReport{}, errors.New("legacy qualification replay does not match a retained artifact")
		}
	}
	normalizePreviousReport(&report)
	return report, nil
}

func validateLegacyReport(report QualificationReport, replay *QualificationReplay) error {
	if report.Schema != LegacyQualificationReportSchema {
		return fmt.Errorf("unsupported qualification report schema %q", report.Schema)
	}
	report.Schema = PreviousQualificationReportSchema
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
	return validateQualificationReport(report)
}

func validateQualificationReport(report QualificationReport) error {
	if report.Schema != QualificationReportSchema {
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
	if report.Evidence == nil || report.Evidence.Schema != runner.ExecutionEvidenceSchema && report.Evidence.Schema != runner.PriorExecutionEvidenceSchema && report.Evidence.Schema != runner.ChoiceExecutionEvidenceSchema && report.Evidence.Schema != runner.LegacyExecutionEvidenceSchema || report.Evidence.Seed != report.Seed {
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
		if run.CampaignPath == "" || run.EvidenceDigest == "" {
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

func cloneReplay(replay *QualificationReplay, artifactPath string) (*QualificationReplay, error) {
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

func evidenceDigest(runRecord runner.ExecutionEvidence) (evidence.SHA256, error) {
	encoded, err := evidence.CanonicalJSON(runRecord)
	if err != nil {
		return "", err
	}
	domain := ExecutionEvidenceDigestDomain
	if runRecord.Schema == runner.PriorExecutionEvidenceSchema {
		domain = PriorExecutionEvidenceDigestDomain
	} else if runRecord.Schema == runner.ChoiceExecutionEvidenceSchema {
		domain = ChoiceExecutionEvidenceDigestDomain
	} else if runRecord.Schema == runner.LegacyExecutionEvidenceSchema {
		domain = LegacyExecutionEvidenceDigestDomain
	}
	return evidence.DomainHash(domain, encoded), nil
}

const (
	ExecutionEvidenceDigestDomain       = "gomadv3.qualification-evidence/v4"
	PriorExecutionEvidenceDigestDomain  = "gomadv3.qualification-evidence/v3"
	ChoiceExecutionEvidenceDigestDomain = "gomadv3.qualification-evidence/v2"
	LegacyExecutionEvidenceDigestDomain = "gomadv3.qualification-evidence/v1"
)

func cloneEvidence(runRecord runner.ExecutionEvidence) (runner.ExecutionEvidence, error) {
	encoded, err := evidence.CanonicalJSON(runRecord)
	if err != nil {
		return runner.ExecutionEvidence{}, err
	}
	var cloned runner.ExecutionEvidence
	if err := evidence.StrictDecode(encoded, &cloned); err != nil {
		return runner.ExecutionEvidence{}, err
	}
	return cloned, nil
}

func firstDivergence(expected, actual runner.ExecutionEvidence) string {
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
		{"frontier", expected.Frontier, actual.Frontier},
	}
	for _, field := range fields {
		if !reflect.DeepEqual(field.expected, field.actual) {
			return field.name
		}
	}
	return "evidence"
}
