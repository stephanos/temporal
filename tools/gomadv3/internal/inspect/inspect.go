package inspect

import (
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/commandline"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const reportSchema = "gomadv3.inspect/v1"

type Report struct {
	Schema   string          `json:"schema"`
	Kind     string          `json:"kind"`
	Path     string          `json:"path"`
	Artifact *ArtifactReport `json:"artifact,omitempty"`
	Batch    *BatchReport    `json:"batch,omitempty"`
}

type ArtifactReport struct {
	ArtifactKind     string           `json:"artifact_kind"`
	RecordHash       record.SHA256    `json:"record_hash"`
	BatchID          string           `json:"batch_id"`
	SelectionOrdinal uint64           `json:"selection_ordinal"`
	Seed             uint64           `json:"seed"`
	ReplayMode       string           `json:"replay_mode"`
	ReplayCommand    string           `json:"replay_command"`
	Runner           record.Runner    `json:"runner"`
	Toolchain        record.Toolchain `json:"toolchain"`
	Target           TargetReport     `json:"target"`
	Outcome          OutcomeReport    `json:"outcome"`
	FirstDivergence  string           `json:"first_divergence,omitempty"`
	Transcript       *Transcript      `json:"transcript,omitempty"`
	CapturedMounts   *CapturedMounts  `json:"captured_mounts,omitempty"`
	Stdout           StreamReport     `json:"stdout"`
	Stderr           StreamReport     `json:"stderr"`
}

type TargetReport struct {
	Kind          string                     `json:"kind"`
	Source        string                     `json:"source"`
	SHA256        record.SHA256              `json:"sha256"`
	Size          uint64                     `json:"size"`
	Argv          []string                   `json:"argv"`
	BuildTags     []string                   `json:"build_tags"`
	Adapters      []record.TargetAdapter     `json:"adapters"`
	Compatibility []record.CompatibilityPack `json:"compatibility"`
	BuildInfo     record.BuildInfo           `json:"build_info"`
}

type OutcomeReport struct {
	Domain           string        `json:"domain"`
	Reason           string        `json:"reason"`
	Termination      string        `json:"termination"`
	ExitCode         *uint64       `json:"exit_code,omitempty"`
	Signal           *string       `json:"signal,omitempty"`
	Deadline         *string       `json:"deadline,omitempty"`
	FailureSignature record.SHA256 `json:"failure_signature"`
	ReplayMatch      *bool         `json:"replay_match,omitempty"`
}

type Transcript struct {
	Schema  string        `json:"schema"`
	SHA256  record.SHA256 `json:"sha256"`
	Bytes   uint64        `json:"bytes"`
	Records uint64        `json:"records"`
}

type CapturedMounts struct {
	Mappings   []string `json:"mappings"`
	Entries    uint64   `json:"entries"`
	NotExist   uint64   `json:"not_exist"`
	TotalBytes uint64   `json:"total_bytes"`
}

type StreamReport struct {
	FullSHA256     record.SHA256 `json:"full_sha256"`
	TotalBytes     uint64        `json:"total_bytes"`
	RetainedBytes  uint64        `json:"retained_bytes"`
	DiscardedBytes uint64        `json:"discarded_bytes"`
	Truncated      bool          `json:"truncated"`
}

type BatchReport struct {
	RunID                string            `json:"run_id"`
	Selection            string            `json:"selection"`
	SelectionCount       uint64            `json:"selection_count"`
	Attempted            uint64            `json:"attempted"`
	Succeeded            uint64            `json:"succeeded"`
	Failures             uint64            `json:"failures"`
	Watchdogs            uint64            `json:"watchdogs"`
	Cancelled            uint64            `json:"cancelled"`
	DistinctFailures     uint64            `json:"distinct_failures"`
	RetainedSuccesses    uint64            `json:"retained_successes"`
	RetainedSuccessBytes uint64            `json:"retained_success_bytes"`
	StopReason           string            `json:"stop_reason"`
	RunsSHA256           record.SHA256     `json:"runs_sha256"`
	Runs                 []BatchRun        `json:"runs"`
	FailureArtifacts     []FailureArtifact `json:"failure_artifacts"`
	SuccessArtifacts     []SuccessArtifact `json:"success_artifacts"`
}

type BatchRun struct {
	SelectionOrdinal     uint64         `json:"selection_ordinal"`
	Seed                 uint64         `json:"seed"`
	Domain               string         `json:"domain"`
	Reason               string         `json:"reason"`
	Termination          string         `json:"termination"`
	ElapsedNanos         uint64         `json:"elapsed_nanos"`
	FailureSignature     *record.SHA256 `json:"failure_signature,omitempty"`
	Artifact             *string        `json:"artifact,omitempty"`
	SuccessArtifact      *string        `json:"success_artifact,omitempty"`
	SuccessArtifactBytes *uint64        `json:"success_artifact_bytes,omitempty"`
	SemanticProbes       []string       `json:"semantic_probes,omitempty"`
	NovelSemanticProbes  []string       `json:"novel_semantic_probes,omitempty"`
	TranscriptSHA256     *record.SHA256 `json:"transcript_sha256,omitempty"`
	TranscriptRecords    *uint64        `json:"transcript_records,omitempty"`
}

type FailureArtifact struct {
	Signature     record.SHA256 `json:"signature"`
	Path          string        `json:"path"`
	ReplayCommand string        `json:"replay_command"`
}

type SuccessArtifact struct {
	Path          string   `json:"path"`
	StoredBytes   uint64   `json:"stored_bytes"`
	NovelProbes   []string `json:"novel_probes,omitempty"`
	ReplayCommand string   `json:"replay_command"`
}

func Open(path string) (Report, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Report{}, fmt.Errorf("resolve inspection path: %w", err)
	}
	hasManifest, err := regularChild(absolute, "manifest.json")
	if err != nil {
		return Report{}, err
	}
	hasBatch, err := regularChild(absolute, "batch.json")
	if err != nil {
		return Report{}, err
	}
	if hasManifest == hasBatch {
		return Report{}, fmt.Errorf("inspection path must contain exactly one of manifest.json or batch.json")
	}
	if hasManifest {
		opened, err := artifact.Open(absolute)
		if err != nil {
			return Report{}, err
		}
		defer opened.Close()
		projected := projectArtifact(opened.Manifest, absolute)
		return Report{Schema: reportSchema, Kind: "artifact", Path: absolute, Artifact: &projected}, nil
	}
	opened, err := artifact.OpenBatch(absolute)
	if err != nil {
		return Report{}, err
	}
	projected, err := projectBatch(opened)
	if err != nil {
		return Report{}, err
	}
	return Report{Schema: reportSchema, Kind: "batch", Path: absolute, Batch: &projected}, nil
}

func regularChild(root, name string) (bool, error) {
	info, err := os.Lstat(filepath.Join(root, name))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", name, err)
	}
	return info.Mode().IsRegular(), nil
}

func projectArtifact(manifest record.Manifest, path string) ArtifactReport {
	result := ArtifactReport{
		ArtifactKind: manifest.ArtifactKind, RecordHash: manifest.RecordHash, BatchID: manifest.BatchID,
		SelectionOrdinal: uint64(manifest.SelectionOrdinal), Seed: uint64(manifest.Seed), ReplayMode: manifest.ReplayMode,
		ReplayCommand: "gomad replay " + commandline.QuoteArgument(path), Runner: manifest.Runner, Toolchain: manifest.Toolchain,
		Target: TargetReport{
			Kind: manifest.Target.Kind, Source: manifest.Target.Source, SHA256: manifest.Target.SHA256, Size: uint64(manifest.Target.Size),
			Argv: append([]string(nil), manifest.Target.Argv...), BuildTags: append([]string(nil), manifest.Target.BuildTags...), Adapters: append([]record.TargetAdapter(nil), manifest.Target.Adapters...), Compatibility: append([]record.CompatibilityPack(nil), manifest.Target.Compatibility...), BuildInfo: manifest.Target.BuildInfo,
		},
		Outcome: projectOutcome(manifest.Outcome), FirstDivergence: firstDivergence(manifest),
		Stdout: projectStream(manifest.Streams.Stdout), Stderr: projectStream(manifest.Streams.Stderr),
	}
	if transcript := manifest.IOProfile.Transcript; transcript != nil {
		result.Transcript = &Transcript{Schema: transcript.Schema, SHA256: transcript.SHA256, Bytes: uint64(transcript.Bytes), Records: uint64(transcript.Records)}
	}
	if mounts := manifest.IOProfile.ReadOnlyMounts; mounts != nil {
		result.CapturedMounts = &CapturedMounts{
			Mappings: append([]string(nil), mounts.Mappings...), Entries: uint64(mounts.Entries), NotExist: uint64(mounts.NotExist), TotalBytes: uint64(mounts.TotalBytes),
		}
	}
	return result
}

func projectOutcome(outcome record.Outcome) OutcomeReport {
	result := OutcomeReport{
		Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, Signal: outcome.Signal,
		Deadline: outcome.Deadline, FailureSignature: outcome.FailureSignature, ReplayMatch: outcome.ReplayMatch,
	}
	if outcome.ExitCode != nil {
		value := uint64(*outcome.ExitCode)
		result.ExitCode = &value
	}
	return result
}

func firstDivergence(manifest record.Manifest) string {
	if manifest.World.Terminal.Kind == "replay-divergence" || manifest.World.Terminal.Kind == "replay_divergence" {
		if manifest.World.Terminal.Detail != "" {
			return manifest.World.Terminal.Detail
		}
		return "World replay diverged"
	}
	if manifest.Outcome.ReplayMatch != nil && !*manifest.Outcome.ReplayMatch {
		return "replay did not match the recorded outcome"
	}
	return ""
}

func projectStream(stream record.Stream) StreamReport {
	return StreamReport{
		FullSHA256: stream.FullSHA256, TotalBytes: uint64(stream.TotalBytes), RetainedBytes: uint64(stream.RetainedBytes),
		DiscardedBytes: uint64(stream.DiscardedBytes), Truncated: stream.Truncated,
	}
}

func projectBatch(opened artifact.Batch) (BatchReport, error) {
	batch := opened.Record
	result := BatchReport{
		RunID: batch.RunID, Selection: batch.Selection, SelectionCount: uint64(batch.SelectionCount), Attempted: uint64(batch.Attempted),
		Succeeded: uint64(batch.Succeeded), Failures: uint64(batch.Failures), Watchdogs: uint64(batch.Watchdogs), Cancelled: uint64(batch.Cancelled),
		DistinctFailures: uint64(batch.DistinctFailures), StopReason: batch.StopReason, RunsSHA256: batch.RunsSHA256,
		RetainedSuccesses: uint64(batch.RetainedSuccesses), RetainedSuccessBytes: uint64(batch.RetainedSuccessBytes),
		Runs: make([]BatchRun, 0, len(opened.Runs)), FailureArtifacts: []FailureArtifact{}, SuccessArtifacts: []SuccessArtifact{},
	}
	seenArtifacts := make(map[string]struct{})
	for _, run := range opened.Runs {
		projected := BatchRun{
			SelectionOrdinal: uint64(run.SelectionOrdinal), Seed: uint64(run.Seed), Domain: run.Domain, Reason: run.Reason,
			Termination: run.Termination, ElapsedNanos: uint64(run.ElapsedNanos), FailureSignature: run.FailureSignature, Artifact: run.Artifact,
			TranscriptSHA256: run.IOTranscriptSHA256,
			SuccessArtifact:  run.SuccessArtifact, SemanticProbes: append([]string(nil), run.SemanticProbes...), NovelSemanticProbes: append([]string(nil), run.NovelSemanticProbes...),
		}
		if run.SuccessArtifactBytes != nil {
			value := uint64(*run.SuccessArtifactBytes)
			projected.SuccessArtifactBytes = &value
		}
		if run.IOTranscriptRecords != nil {
			value := uint64(*run.IOTranscriptRecords)
			projected.TranscriptRecords = &value
		}
		result.Runs = append(result.Runs, projected)
		if run.SuccessArtifact != nil {
			retained, err := artifact.ResolveRetainedEvidence(opened.Path, batch.RunID, run)
			if err != nil {
				return BatchReport{}, fmt.Errorf("open retained success %s: %w", *run.SuccessArtifact, err)
			}
			result.SuccessArtifacts = append(result.SuccessArtifacts, SuccessArtifact{
				Path: retained.Path, StoredBytes: retained.StoredBytes, NovelProbes: append([]string(nil), run.NovelSemanticProbes...), ReplayCommand: "gomad replay " + commandline.QuoteArgument(retained.Path),
			})
		}
		if run.Artifact == nil {
			continue
		}
		path := filepath.Join(opened.Path, filepath.FromSlash(*run.Artifact))
		if _, found := seenArtifacts[path]; found {
			continue
		}
		failure, err := artifact.ResolveRetainedEvidence(opened.Path, batch.RunID, run)
		if err != nil {
			return BatchReport{}, fmt.Errorf("open retained failure %s: %w", *run.Artifact, err)
		}
		seenArtifacts[failure.Path] = struct{}{}
		result.FailureArtifacts = append(result.FailureArtifacts, FailureArtifact{
			Signature: *run.FailureSignature, Path: failure.Path, ReplayCommand: "gomad replay " + commandline.QuoteArgument(failure.Path),
		})
	}
	return result, nil
}
