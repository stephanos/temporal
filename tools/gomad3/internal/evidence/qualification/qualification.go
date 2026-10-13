package qualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/model/strictjson"
)

const (
	QualificationReportSchema = "gomad3.qualification/v1"
	ChoiceReplayNone          = "none"
	ChoiceReplayExact         = "exact"
	ChoiceReplayDiverged      = "diverged"
	ChoiceReplayUnavailable   = "unavailable"
	DiagnosticComplete        = "complete"
	DiagnosticUnavailable     = "unavailable"
)

const maximumQualificationReportBytes = 16 << 20

type qualificationStagedFile interface {
	Path() string
	PublishContext(context.Context, string) (bool, error)
	Cleanup() error
}

type qualificationStage func(context.Context, string, string, os.FileMode, func(context.Context, io.Writer) error) (qualificationStagedFile, error)

func stageQualification(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (qualificationStagedFile, error) {
	return hostfs.StageContext(ctx, directory, pattern, mode, fill)
}

type QualificationExecution struct {
	CampaignPath     string
	ArtifactPath     string
	WallElapsedNanos uint64
	Evidence         record.ExecutionEvidence
	Replay           *QualificationReplay
	Diagnostics      *record.DiagnosticTraceReference
}

type QualificationInput struct {
	Command    []string
	Executions []QualificationExecution
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
	Classification string              `json:"classification"`
	Message        string              `json:"message"`
	Iteration      record.Uint64String `json:"iteration"`
	ImportPath     string              `json:"import_path,omitempty"`
	Capability     string              `json:"capability,omitempty"`
}

type QualificationExecutionReport struct {
	Evidence         *record.ExecutionEvidence        `json:"evidence,omitempty"`
	DiagnosticStatus string                           `json:"diagnostic_status,omitempty"`
	CampaignPath     string                           `json:"campaign_path"`
	ArtifactPath     string                           `json:"artifact_path,omitempty"`
	EvidenceDigest   record.SHA256                    `json:"evidence_digest"`
	WallElapsedNanos record.Uint64String              `json:"wall_elapsed_nanos"`
	Diagnostics      *record.DiagnosticTraceReference `json:"diagnostics,omitempty"`
	Replay           *QualificationReplay             `json:"replay,omitempty"`
}

type QualificationDiagnosticDivergence struct {
	BaselineIteration record.Uint64String `json:"baseline_iteration"`
	Iteration         record.Uint64String `json:"iteration"`
	choice.DiagnosticDivergence
}

type QualificationReport struct {
	Schema               string                             `json:"schema"`
	Qualified            bool                               `json:"qualified"`
	Deterministic        bool                               `json:"deterministic"`
	TargetSuccess        bool                               `json:"target_success"`
	Seed                 record.Uint64String                `json:"seed"`
	Repeat               record.Uint64String                `json:"repeat"`
	Command              []string                           `json:"command"`
	EvidenceDigest       record.SHA256                      `json:"evidence_digest,omitempty"`
	Evidence             *record.ExecutionEvidence          `json:"evidence,omitempty"`
	Executions           []QualificationExecutionReport     `json:"executions"`
	DiagnosticDivergence *QualificationDiagnosticDivergence `json:"diagnostic_divergence,omitempty"`
	FirstDivergence      string                             `json:"first_divergence,omitempty"`
	Failure              *QualificationFailure              `json:"failure,omitempty"`
}

func BuildQualificationReport(input QualificationInput) (QualificationReport, error) {
	if err := validateQualificationInputStrings(input); err != nil {
		return QualificationReport{}, err
	}
	if len(input.Command) == 0 || input.Command[0] == "" {
		return QualificationReport{}, fmt.Errorf("qualification command is required")
	}
	if len(input.Executions) < 2 {
		return QualificationReport{}, fmt.Errorf("qualification requires at least two executions")
	}
	baseline, err := cloneEvidence(input.Executions[0].Evidence)
	if err != nil {
		return QualificationReport{}, fmt.Errorf("validate execution evidence 0: %w", err)
	}
	if baseline.Schema != record.ExecutionEvidenceSchema {
		return QualificationReport{}, fmt.Errorf("execution evidence 0 has unsupported schema %q", baseline.Schema)
	}
	report := QualificationReport{
		Schema: QualificationReportSchema, Deterministic: true, TargetSuccess: true,
		Seed: baseline.Seed, Repeat: record.Uint64String(len(input.Executions)), Command: append([]string(nil), input.Command...), Evidence: &baseline,
		Executions: make([]QualificationExecutionReport, 0, len(input.Executions)),
	}
	replayOK := true
	for index, run := range input.Executions {
		if run.CampaignPath == "" {
			return QualificationReport{}, fmt.Errorf("qualification execution %d has no campaign path", index)
		}
		if run.Evidence.Schema != record.ExecutionEvidenceSchema {
			return QualificationReport{}, fmt.Errorf("execution evidence %d has unsupported schema %q", index, run.Evidence.Schema)
		}
		if run.Evidence.Seed != baseline.Seed {
			return QualificationReport{}, fmt.Errorf("qualification execution %d has seed %d, want %d", index, run.Evidence.Seed, baseline.Seed)
		}
		if (run.Evidence.Diagnostics == nil) != (run.Diagnostics == nil) {
			return QualificationReport{}, fmt.Errorf("qualification execution %d diagnostic evidence is incomplete", index)
		}
		if run.Diagnostics != nil && run.Diagnostics.DiagnosticEvidence != *run.Evidence.Diagnostics {
			return QualificationReport{}, fmt.Errorf("qualification execution %d diagnostic evidence disagrees", index)
		}
		digest, digestErr := evidenceDigest(run.Evidence)
		if digestErr != nil {
			return QualificationReport{}, fmt.Errorf("hash execution evidence %d: %w", index, digestErr)
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
		copiedReplay, replayErr := cloneReplay(run.Replay, run.ArtifactPath)
		if replayErr != nil {
			return QualificationReport{}, fmt.Errorf("validate execution replay %d: %w", index, replayErr)
		}
		if copiedReplay != nil && !copiedReplay.Match {
			replayOK = false
		}
		if copiedReplay != nil && run.Evidence.Choices != nil && copiedReplay.Match && copiedReplay.ChoiceReplayStatus != ChoiceReplayExact {
			return QualificationReport{}, fmt.Errorf("validate execution replay %d: exact choice replay evidence is required", index)
		}
		report.Executions = append(report.Executions, QualificationExecutionReport{CampaignPath: run.CampaignPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, WallElapsedNanos: record.Uint64String(run.WallElapsedNanos), Replay: copiedReplay, Diagnostics: cloneDiagnosticReference(run.Diagnostics)})
	}
	if err := bindDiagnosticEvidence(&report, input.Executions); err != nil {
		return QualificationReport{}, err
	}
	if err := attachDiagnosticComparison(&report); err != nil {
		return QualificationReport{}, err
	}
	report.Qualified = report.Deterministic && report.TargetSuccess && replayOK
	return report, nil
}

func BuildQualificationFailure(command []string, seed uint64, repeat uint64, completed []QualificationExecution, failure QualificationFailure) (QualificationReport, error) {
	if err := validateUTF8Strings(append(append([]string{}, command...), failure.Classification, failure.Message, failure.ImportPath, failure.Capability)); err != nil {
		return QualificationReport{}, err
	}
	for _, run := range completed {
		if err := validateQualificationExecutionStrings(run); err != nil {
			return QualificationReport{}, err
		}
	}
	if len(command) == 0 || command[0] == "" {
		return QualificationReport{}, fmt.Errorf("qualification command is required")
	}
	if repeat < 2 {
		return QualificationReport{}, fmt.Errorf("qualification requires at least two executions")
	}
	if failure.Classification == "" || failure.Message == "" || uint64(failure.Iteration) == 0 || uint64(failure.Iteration) > repeat {
		return QualificationReport{}, fmt.Errorf("qualification failure is incomplete")
	}
	report := QualificationReport{
		Schema: QualificationReportSchema, Seed: record.Uint64String(seed), Repeat: record.Uint64String(repeat), Command: append([]string(nil), command...),
		Executions: make([]QualificationExecutionReport, 0, len(completed)), Failure: &failure,
	}
	for index, run := range completed {
		if run.CampaignPath == "" || run.Evidence.Schema != record.ExecutionEvidenceSchema || uint64(run.Evidence.Seed) != seed {
			return QualificationReport{}, fmt.Errorf("completed qualification execution %d is invalid", index)
		}
		if (run.Evidence.Diagnostics == nil) != (run.Diagnostics == nil) {
			return QualificationReport{}, fmt.Errorf("completed qualification execution %d diagnostic evidence is incomplete", index)
		}
		if run.Diagnostics != nil && run.Diagnostics.DiagnosticEvidence != *run.Evidence.Diagnostics {
			return QualificationReport{}, fmt.Errorf("completed qualification execution %d diagnostic evidence disagrees", index)
		}
		digest, err := evidenceDigest(run.Evidence)
		if err != nil {
			return QualificationReport{}, fmt.Errorf("hash completed qualification execution %d: %w", index, err)
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
		report.Executions = append(report.Executions, QualificationExecutionReport{CampaignPath: run.CampaignPath, ArtifactPath: run.ArtifactPath, EvidenceDigest: digest, WallElapsedNanos: record.Uint64String(run.WallElapsedNanos), Replay: replayEvidence, Diagnostics: cloneDiagnosticReference(run.Diagnostics)})
	}
	if err := bindDiagnosticEvidence(&report, completed); err != nil {
		return QualificationReport{}, err
	}
	if err := attachDiagnosticComparison(&report); err != nil {
		return QualificationReport{}, err
	}
	return report, nil
}

func WriteQualificationReport(artifactRoot string, report QualificationReport) (reportPath string, retErr error) {
	return writeQualificationReport(context.Background(), artifactRoot, report)
}

func writeQualificationReport(ctx context.Context, artifactRoot string, report QualificationReport) (reportPath string, retErr error) {
	return writeQualificationReportWith(ctx, artifactRoot, report, stageQualification)
}

func writeQualificationReportWith(ctx context.Context, artifactRoot string, report QualificationReport, stage qualificationStage) (reportPath string, retErr error) {
	if artifactRoot == "" {
		return "", fmt.Errorf("artifact root is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := validateQualificationReport(report); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("encode qualification report: %w", err)
	}
	encoded = append(encoded, '\n')
	root := filepath.Join(artifactRoot, "qualifications", "v1")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("create qualification report directory: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", fmt.Errorf("make qualification report directory private: %w", err)
	}
	staged, err := stage(ctx, root, ".qualification-*.partial", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := writer.Write(encoded)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("stage qualification report: %w", err)
	}
	defer func() {
		if cleanupErr := staged.Cleanup(); cleanupErr != nil {
			cleanupErr = fmt.Errorf("clean qualification report staging file: %w", cleanupErr)
			if retErr == nil {
				retErr = cleanupErr
			} else {
				retErr = errors.Join(retErr, cleanupErr)
			}
		}
	}()
	name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(staged.Path()), ".qualification-"), ".partial")
	path := filepath.Join(root, "qualification-"+name+".json")
	published, err := staged.PublishContext(ctx, path)
	if err != nil {
		if published {
			return path, fmt.Errorf("publish qualification report: %w", err)
		}
		return "", fmt.Errorf("publish qualification report: %w", err)
	}
	return path, nil
}

func OpenQualificationReport(path string) (report QualificationReport, retErr error) {
	file, err := os.Open(path)
	if err != nil {
		return QualificationReport{}, fmt.Errorf("open qualification report: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			report = QualificationReport{}
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
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
	var report QualificationReport
	if err := strictjson.Decode(data, &report); err != nil {
		return QualificationReport{}, fmt.Errorf("decode qualification report: %w", err)
	}
	if err := validateQualificationReport(report); err != nil {
		return QualificationReport{}, err
	}
	return report, nil
}

func validateQualificationReport(report QualificationReport) error {
	if err := validateQualificationReportStrings(report); err != nil {
		return err
	}
	if report.Schema != QualificationReportSchema {
		return fmt.Errorf("unsupported qualification report schema %q", report.Schema)
	}
	if len(report.Command) == 0 || report.Command[0] == "" || uint64(report.Repeat) < 2 {
		return fmt.Errorf("qualification report command or repetition count is invalid")
	}
	if report.Failure != nil {
		if report.Qualified || report.Deterministic || report.TargetSuccess || report.Failure.Classification == "" || report.Failure.Message == "" || uint64(report.Failure.Iteration) == 0 || report.Failure.Iteration > report.Repeat || len(report.Executions) > int(report.Repeat) {
			return fmt.Errorf("qualification failure result is inconsistent")
		}
		if len(report.Executions) == 0 {
			if report.Evidence != nil || report.EvidenceDigest != "" || report.DiagnosticDivergence != nil {
				return fmt.Errorf("qualification failure has evidence without completed executions")
			}
			return nil
		}
	}
	if report.Failure == nil && (len(report.Executions) < 2 || uint64(report.Repeat) != uint64(len(report.Executions))) {
		return fmt.Errorf("qualification report repetition count is invalid")
	}
	if report.Evidence == nil || report.Evidence.Schema != record.ExecutionEvidenceSchema || report.Evidence.Seed != report.Seed {
		return fmt.Errorf("qualification baseline evidence identity is invalid")
	}
	digest, err := evidenceDigest(*report.Evidence)
	if err != nil {
		return fmt.Errorf("hash qualification baseline evidence: %w", err)
	}
	if digest != report.EvidenceDigest {
		return fmt.Errorf("qualification baseline evidence digest is invalid")
	}
	if err := validateDiagnosticEvidence(report); err != nil {
		return err
	}
	deterministic := true
	for index, run := range report.Executions {
		if run.CampaignPath == "" || run.EvidenceDigest == "" {
			return fmt.Errorf("qualification execution %d identity is invalid", index)
		}
		if run.Diagnostics != nil && (run.Diagnostics.Path == "" || run.Diagnostics.Profile != choice.DiagnosticProfile || !validDiagnosticReference(*run.Diagnostics)) {
			return errors.New("qualification diagnostic trace reference is invalid")
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
	for index, run := range report.Executions {
		if _, err := cloneReplay(run.Replay, run.ArtifactPath); err != nil {
			return fmt.Errorf("qualification execution %d replay is invalid: %w", index, err)
		}
		if run.Replay != nil && !run.Replay.Match {
			replayOK = false
		}
		if run.Replay != nil && report.Evidence.Choices != nil && run.Replay.Match && run.Replay.ChoiceReplayStatus != ChoiceReplayExact {
			return fmt.Errorf("qualification execution %d lacks exact choice replay evidence", index)
		}
	}
	if report.Qualified != (report.Deterministic && report.TargetSuccess && replayOK) {
		return fmt.Errorf("qualification result is inconsistent")
	}
	return nil
}

func validateQualificationInputStrings(input QualificationInput) error {
	if err := validateUTF8Strings(input.Command); err != nil {
		return err
	}
	for _, run := range input.Executions {
		if err := validateQualificationExecutionStrings(run); err != nil {
			return err
		}
	}
	return nil
}

func validateQualificationExecutionStrings(run QualificationExecution) error {
	values := []string{run.CampaignPath, run.ArtifactPath}
	appendExecutionEvidenceStrings(&values, run.Evidence)
	if run.Replay != nil {
		values = append(values, run.Replay.ArtifactPath, run.Replay.Divergence, run.Replay.ChoiceReplayStatus)
	}
	if run.Diagnostics != nil {
		values = append(values, run.Diagnostics.Path, run.Diagnostics.Profile, string(run.Diagnostics.SHA256))
	}
	return validateUTF8Strings(values)
}

func validateQualificationReportStrings(report QualificationReport) error {
	values := append([]string{report.Schema, string(report.EvidenceDigest), report.FirstDivergence}, report.Command...)
	if report.Evidence != nil {
		appendExecutionEvidenceStrings(&values, *report.Evidence)
	}
	if report.Failure != nil {
		values = append(values, report.Failure.Classification, report.Failure.Message, report.Failure.ImportPath, report.Failure.Capability)
	}
	if report.DiagnosticDivergence != nil {
		values = append(values, report.DiagnosticDivergence.Fields...)
	}
	for _, run := range report.Executions {
		values = append(values, run.DiagnosticStatus, run.CampaignPath, run.ArtifactPath, string(run.EvidenceDigest))
		if run.Evidence != nil {
			appendExecutionEvidenceStrings(&values, *run.Evidence)
		}
		if run.Diagnostics != nil {
			values = append(values, run.Diagnostics.Path, run.Diagnostics.Profile, string(run.Diagnostics.SHA256))
		}
		if run.Replay != nil {
			values = append(values, run.Replay.ArtifactPath, run.Replay.Divergence, run.Replay.ChoiceReplayStatus)
		}
	}
	return validateUTF8Strings(values)
}

func appendExecutionEvidenceStrings(values *[]string, evidence record.ExecutionEvidence) {
	*values = append(*values,
		evidence.Schema, evidence.RunnerBuild,
		evidence.Toolchain.GoVersion, evidence.Toolchain.BuildKey, evidence.Toolchain.TargetGOOS, evidence.Toolchain.TargetGOARCH,
		evidence.IOProfile.Name, string(evidence.IOProfile.ImplementationSHA256), string(evidence.IOProfile.InventorySHA256),
		evidence.Outcome.Domain, evidence.Outcome.Reason, evidence.Outcome.Termination,
		evidence.Stdout.File, string(evidence.Stdout.RetainedSHA256), string(evidence.Stdout.FullSHA256),
		evidence.Stderr.File, string(evidence.Stderr.RetainedSHA256), string(evidence.Stderr.FullSHA256), string(evidence.IOTranscriptSHA256),
		string(evidence.BackendEvidenceSHA256), evidence.World.Initial.Schema, evidence.World.Initial.File,
		string(evidence.World.Initial.RawSHA256), string(evidence.World.Initial.SemanticDigest),
		evidence.World.Transitions.Schema, evidence.World.Transitions.File, string(evidence.World.Transitions.RawSHA256),
		string(evidence.World.Transitions.TranscriptDigest), evidence.World.Final.Schema, evidence.World.Final.File,
		string(evidence.World.Final.RawSHA256), string(evidence.World.Final.SemanticDigest),
		evidence.World.Terminal.Kind, evidence.World.Terminal.Detail,
		evidence.SemanticCoverage.Schema, string(evidence.SemanticCoverage.Digest),
	)
	appendTargetStrings(values, evidence.Target)
	for _, entry := range evidence.Environment {
		*values = append(*values, entry.Name, entry.Value)
	}
	if evidence.Outcome.Signal != nil {
		*values = append(*values, *evidence.Outcome.Signal)
	}
	if evidence.Outcome.Deadline != nil {
		*values = append(*values, *evidence.Outcome.Deadline)
	}
	for _, adapter := range evidence.World.Adapters {
		*values = append(*values, adapter.Schema, string(adapter.InitialDigest), string(adapter.FinalDigest))
	}
	*values = append(*values, evidence.SemanticCoverage.Probes...)
	if evidence.Diagnostics != nil {
		*values = append(*values, evidence.Diagnostics.Profile, string(evidence.Diagnostics.SHA256))
	}
	if evidence.Choices != nil {
		choices := evidence.Choices
		*values = append(*values, choices.Profile, string(choices.ImplementationSHA256), string(choices.SHA256), choices.TerminalState, string(choices.TapeSHA256))
		for _, feature := range choices.Features {
			*values = append(*values, string(feature.Kind), feature.Value)
		}
	}
	if evidence.ReadOnlyMountsSHA256 != nil {
		*values = append(*values, string(*evidence.ReadOnlyMountsSHA256))
	}
	if evidence.ChoiceExploration != nil {
		exploration := evidence.ChoiceExploration
		*values = append(*values, string(exploration.ImplementationSHA256), string(exploration.CandidateSHA256), string(exploration.ParentSHA256), string(exploration.PrefixSHA256), string(exploration.OutcomeSHA256))
	}
}

func appendTargetStrings(values *[]string, target record.Target) {
	*values = append(*values, target.Kind, target.Source, target.File, string(target.SHA256), target.CapabilityMode,
		target.BuildInfo.GoVersion, target.BuildInfo.Path, target.BuildInfo.MainModule)
	*values = append(*values, target.Argv...)
	*values = append(*values, target.BuildTags...)
	for _, adapter := range target.Adapters {
		*values = append(*values, adapter.Module, adapter.Version, adapter.Sum)
	}
	for _, pack := range target.Compatibility {
		*values = append(*values, pack.ID, string(pack.SHA256))
	}
	for _, setting := range target.BuildInfo.Settings {
		*values = append(*values, setting.Key, setting.Value)
	}
	if target.CapabilityManifest != nil {
		manifest := target.CapabilityManifest
		*values = append(*values, manifest.Schema, manifest.File, string(manifest.SHA256), string(manifest.ProducerImplementationSHA256), string(manifest.GuardImplementationSHA256), string(manifest.CapabilityUniverseSHA256))
	}
	if target.Backend != nil {
		backend := target.Backend
		*values = append(*values, backend.Name, backend.ReplayMode, backend.Provenance.Schema, backend.Provenance.File, string(backend.Provenance.SHA256))
		if backend.Evidence != nil {
			*values = append(*values, backend.Evidence.Schema, backend.Evidence.File, string(backend.Evidence.SHA256))
		}
	}
}

func validateUTF8Strings(values []string) error {
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("qualification string is not valid UTF-8")
		}
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
	case "", ChoiceReplayNone, ChoiceReplayExact:
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

func evidenceDigest(runRecord record.ExecutionEvidence) (record.SHA256, error) {
	encoded, err := json.Marshal(runRecord)
	if err != nil {
		return "", err
	}
	return record.DomainHash(ExecutionEvidenceDigestDomain, encoded), nil
}

const ExecutionEvidenceDigestDomain = "gomad3.qualification-evidence/v1"

func cloneEvidence(runRecord record.ExecutionEvidence) (record.ExecutionEvidence, error) {
	encoded, err := json.Marshal(runRecord)
	if err != nil {
		return record.ExecutionEvidence{}, err
	}
	var cloned record.ExecutionEvidence
	if err := strictjson.Decode(encoded, &cloned); err != nil {
		return record.ExecutionEvidence{}, err
	}
	return cloned, nil
}

func firstDivergence(expected, actual record.ExecutionEvidence) string {
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
		{"virtual_time_elapsed_nanos", expected.VirtualTimeElapsedNanos, actual.VirtualTimeElapsedNanos},
		{"choices", expected.Choices, actual.Choices},
		{"diagnostics", expected.Diagnostics, actual.Diagnostics},
		{"world", expected.World, actual.World},
		{"read_only_mounts_sha256", expected.ReadOnlyMountsSHA256, actual.ReadOnlyMountsSHA256},
		{"semantic_coverage", expected.SemanticCoverage, actual.SemanticCoverage},
		{"choice_exploration", expected.ChoiceExploration, actual.ChoiceExploration},
	}
	for _, field := range fields {
		if !reflect.DeepEqual(field.expected, field.actual) {
			return field.name
		}
	}
	return "evidence"
}

func cloneDiagnosticReference(reference *record.DiagnosticTraceReference) *record.DiagnosticTraceReference {
	if reference == nil {
		return nil
	}
	cloned := *reference
	return &cloned
}

func diagnosticStatus(evidence record.ExecutionEvidence) (string, error) {
	if evidence.Diagnostics != nil {
		return DiagnosticComplete, nil
	}
	enabled := false
	for _, entry := range evidence.Environment {
		if entry.Name == choice.DiagnosticProfileEnvironment {
			if entry.Value != choice.DiagnosticProfile {
				return "", errors.New("qualification diagnostic profile is invalid")
			}
			enabled = true
		}
	}
	if !enabled {
		return "", nil
	}
	if evidence.Outcome.Domain == "watchdog" && evidence.Outcome.Reason == "watchdog_timeout" || evidence.Outcome.Domain == "runner" && evidence.Outcome.Reason == "runner_cancelled" {
		return DiagnosticUnavailable, nil
	}
	return "", errors.New("qualification diagnostic trace is unavailable for an uninterrupted execution")
}

func bindDiagnosticEvidence(report *QualificationReport, runs []QualificationExecution) error {
	enabled := false
	for _, run := range runs {
		status, err := diagnosticStatus(run.Evidence)
		if err != nil {
			return err
		}
		enabled = enabled || status != ""
	}
	if !enabled {
		return nil
	}
	for index, run := range runs {
		status, err := diagnosticStatus(run.Evidence)
		if err != nil {
			return err
		}
		if status == "" {
			return fmt.Errorf("qualification execution %d omitted diagnostic evidence", index)
		}
		evidence, err := cloneEvidence(run.Evidence)
		if err != nil {
			return err
		}
		report.Executions[index].Evidence = &evidence
		report.Executions[index].DiagnosticStatus = status
	}
	return nil
}

func validateDiagnosticEvidence(report QualificationReport) error {
	status, err := diagnosticStatus(*report.Evidence)
	if err != nil {
		return err
	}
	enabled := status != ""
	for _, run := range report.Executions {
		enabled = enabled || run.Diagnostics != nil || run.Evidence != nil || run.DiagnosticStatus != ""
	}
	if !enabled {
		if report.DiagnosticDivergence != nil {
			return errors.New("qualification diagnostic divergence has no diagnostic evidence")
		}
		return nil
	}
	for index, run := range report.Executions {
		if run.Evidence == nil || run.Evidence.Schema != record.ExecutionEvidenceSchema || run.Evidence.Seed != report.Seed {
			return fmt.Errorf("qualification execution %d diagnostic evidence identity is invalid", index)
		}
		digest, err := evidenceDigest(*run.Evidence)
		if err != nil || digest != run.EvidenceDigest || index == 0 && digest != report.EvidenceDigest {
			return fmt.Errorf("qualification execution %d diagnostic evidence digest is invalid", index)
		}
		status, err := diagnosticStatus(*run.Evidence)
		if err != nil {
			return err
		}
		if status == "" || status != run.DiagnosticStatus {
			return fmt.Errorf("qualification execution %d diagnostic status is inconsistent", index)
		}
		if (run.Evidence.Diagnostics == nil) != (run.Diagnostics == nil) || run.Diagnostics != nil && run.Diagnostics.DiagnosticEvidence != *run.Evidence.Diagnostics {
			return fmt.Errorf("qualification execution %d diagnostic evidence disagrees", index)
		}
	}
	if difference := report.DiagnosticDivergence; difference != nil {
		if report.Deterministic || len(difference.Fields) == 0 || difference.BaselineIteration < 1 || difference.Iteration <= difference.BaselineIteration || uint64(difference.Iteration) > uint64(len(report.Executions)) {
			return errors.New("qualification diagnostic divergence is inconsistent")
		}
		baseline := report.Executions[int(difference.BaselineIteration)-1]
		actual := report.Executions[int(difference.Iteration)-1]
		if baseline.Diagnostics == nil || actual.Diagnostics == nil || baseline.Diagnostics.SHA256 == actual.Diagnostics.SHA256 {
			return errors.New("qualification diagnostic divergence pair is inconsistent")
		}
	}
	return nil
}

func attachDiagnosticComparison(report *QualificationReport) error {
	var baseline *choice.DiagnosticTrace
	baselineIndex := 0
	for index, run := range report.Executions {
		if run.Diagnostics == nil {
			continue
		}
		observed, err := choice.ReadDiagnosticTrace(run.Diagnostics.Path)
		if err != nil {
			return fmt.Errorf("qualification execution %d diagnostic trace: %w", index, err)
		}
		if run.Diagnostics.Profile != choice.DiagnosticProfile || run.Diagnostics.SHA256 != record.SHA256FromSum(observed.SHA256) || uint64(run.Diagnostics.Records) != uint64(len(observed.Records)) {
			return errors.New("qualification diagnostic trace identity changed")
		}
		if baseline == nil {
			baseline = &observed
			baselineIndex = index
			continue
		}
		if run.EvidenceDigest != report.Executions[baselineIndex].EvidenceDigest && report.DiagnosticDivergence == nil {
			difference, err := choice.DiffDiagnostics(baseline.Bytes, observed.Bytes)
			if err != nil {
				return err
			}
			if difference != nil {
				report.DiagnosticDivergence = &QualificationDiagnosticDivergence{BaselineIteration: record.Uint64String(baselineIndex + 1), Iteration: record.Uint64String(index + 1), DiagnosticDivergence: *difference}
			}
		}
	}
	return nil
}

func validDiagnosticReference(reference record.DiagnosticTraceReference) bool {
	_, err := record.ParseSHA256(string(reference.SHA256))
	return err == nil && uint64(reference.Records) <= (choice.MaximumDiagnosticBytes-64)/96
}
