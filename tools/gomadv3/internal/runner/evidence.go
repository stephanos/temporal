package runner

import (
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	executionoutcome "go.temporal.io/server/tools/gomadv3/internal/outcome"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const RunEvidenceSchema = "gomadv3.run-evidence/v1"

type IOProfileEvidence = ioprofile.Identity

type OutcomeEvidence struct {
	Domain      string               `json:"domain"`
	Reason      string               `json:"reason"`
	Termination string               `json:"termination"`
	ExitCode    *record.Uint64String `json:"exit_code,omitempty"`
	Signal      *string              `json:"signal,omitempty"`
	Deadline    *string              `json:"deadline,omitempty"`
}

type RunLimitsEvidence struct {
	RunTimeoutNanos      record.Uint64String `json:"run_timeout_nanos"`
	TerminateGraceNanos  record.Uint64String `json:"terminate_grace_nanos"`
	OutputBytes          record.Uint64String `json:"output_bytes"`
	WorldTransitionBytes record.Uint64String `json:"world_transition_bytes"`
	IOTranscriptBytes    record.Uint64String `json:"io_transcript_bytes"`
}

type RunEvidence struct {
	Schema               string                     `json:"schema"`
	Seed                 record.Uint64String        `json:"seed"`
	RunnerBuild          string                     `json:"runner_build"`
	Toolchain            record.Toolchain           `json:"toolchain"`
	Target               record.Target              `json:"target"`
	IOProfile            IOProfileEvidence          `json:"io_profile"`
	Environment          []record.Environment       `json:"environment"`
	Limits               RunLimitsEvidence          `json:"limits"`
	Outcome              OutcomeEvidence            `json:"outcome"`
	GroupGone            bool                       `json:"group_gone"`
	Stdout               record.Stream              `json:"stdout"`
	Stderr               record.Stream              `json:"stderr"`
	IOTranscriptSHA256   record.SHA256              `json:"io_transcript_sha256"`
	IOTranscriptRecords  record.Uint64String        `json:"io_transcript_records"`
	IOTranscriptComplete bool                       `json:"io_transcript_complete"`
	World                record.World               `json:"world"`
	ReadOnlyMountsSHA256 *record.SHA256             `json:"read_only_mounts_sha256,omitempty"`
	SemanticCoverage     ioprofile.SemanticCoverage `json:"semantic_coverage"`
}

func runEvidence(
	config Config,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	completion runCompletion,
	outcome executionoutcome.Classification,
	worldRecord record.World,
	mountArtifact *romount.ArtifactRecord,
	coverage ioprofile.SemanticCoverage,
) RunEvidence {
	profile := ioprofile.Default()
	evidence := RunEvidence{
		Schema:      RunEvidenceSchema,
		Seed:        record.Uint64String(completion.job.seed),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Target:      prepared.RecordTarget(),
		IOProfile:   profile.Identity(),
		Environment: append([]record.Environment(nil), environmentForSeed(baseEnvironment, completion.job.seed)...),
		Limits: RunLimitsEvidence{
			RunTimeoutNanos: record.Uint64String(config.RunTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit), IOTranscriptBytes: 64 << 20,
		},
		Outcome: OutcomeEvidence{
			Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
			ExitCode: cloneUint64String(outcome.ExitCode), Signal: cloneString(outcome.Signal), Deadline: cloneString(outcome.Deadline),
		},
		GroupGone:            completion.result.GroupGone,
		Stdout:               streamRecord(completion.result.Stdout),
		Stderr:               streamRecord(completion.result.Stderr),
		IOTranscriptSHA256:   record.SHA256FromSum(completion.result.IOTranscript.SHA256),
		IOTranscriptRecords:  record.Uint64String(completion.result.IOTranscript.Records),
		IOTranscriptComplete: completion.result.IOTranscript.Complete,
		World:                cloneWorld(worldRecord),
		SemanticCoverage:     coverage,
	}
	if mountArtifact != nil {
		digest := mountArtifact.Manifest.SHA256
		evidence.ReadOnlyMountsSHA256 = &digest
	}
	return evidence
}

func cloneBuildInfo(buildInfo record.BuildInfo) record.BuildInfo {
	buildInfo.Settings = append([]record.BuildSetting(nil), buildInfo.Settings...)
	return buildInfo
}

func cloneCompatibility(packs []record.CompatibilityPack) []record.CompatibilityPack {
	return append([]record.CompatibilityPack{}, packs...)
}

func cloneAdapters(adapters []record.TargetAdapter) []record.TargetAdapter {
	return append([]record.TargetAdapter{}, adapters...)
}

func cloneWorld(worldRecord record.World) record.World {
	worldRecord.Adapters = append([]record.WorldAdapter(nil), worldRecord.Adapters...)
	return worldRecord
}

func cloneUint64String(value *record.Uint64String) *record.Uint64String {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
