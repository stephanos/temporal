package runner

import (
	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
)

const ExecutionEvidenceSchema = "gomadv3.execution-evidence/v1"

type OutcomeEvidence struct {
	Domain      string               `json:"domain"`
	Reason      string               `json:"reason"`
	Termination string               `json:"termination"`
	ExitCode    *record.Uint64String `json:"exit_code,omitempty"`
	Signal      *string              `json:"signal,omitempty"`
	Deadline    *string              `json:"deadline,omitempty"`
}

type ExecutionLimitsEvidence struct {
	ExecutionTimeoutNanos record.Uint64String `json:"execution_timeout_nanos"`
	TerminateGraceNanos   record.Uint64String `json:"terminate_grace_nanos"`
	OutputBytes           record.Uint64String `json:"output_bytes"`
	WorldTransitionBytes  record.Uint64String `json:"world_transition_bytes"`
	IOTranscriptBytes     record.Uint64String `json:"io_transcript_bytes"`
	ChoiceTraceBytes      record.Uint64String `json:"choice_trace_bytes,omitempty"`
}

type ChoiceEvidence struct {
	Profile                string              `json:"profile"`
	ImplementationSHA256   record.SHA256       `json:"implementation_sha256"`
	Limit                  record.Uint64String `json:"limit"`
	SHA256                 record.SHA256       `json:"sha256"`
	Records                record.Uint64String `json:"records"`
	BranchingRecords       record.Uint64String `json:"branching_records"`
	TerminalState          string              `json:"terminal_state"`
	TapeSHA256             record.SHA256       `json:"tape_sha256"`
	Decisions              record.Uint64String `json:"decisions"`
	Runnable               record.Uint64String `json:"runnable"`
	SelectPoll             record.Uint64String `json:"select_poll"`
	SelectResult           record.Uint64String `json:"select_result"`
	Features               []choice.Feature    `json:"features"`
	AdjacentPairsObserved  record.Uint64String `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool                `json:"adjacent_pairs_truncated"`
}

type ExecutionEvidence struct {
	Schema               string                           `json:"schema"`
	Seed                 record.Uint64String              `json:"seed"`
	RunnerBuild          string                           `json:"runner_build"`
	Toolchain            record.Toolchain                 `json:"toolchain"`
	Target               record.Target                    `json:"target"`
	IOProfile            deterministicio.Contract         `json:"io_profile"`
	Environment          []record.Environment             `json:"environment"`
	Limits               ExecutionLimitsEvidence          `json:"limits"`
	Outcome              OutcomeEvidence                  `json:"outcome"`
	GroupGone            bool                             `json:"group_gone"`
	Stdout               record.Stream                    `json:"stdout"`
	Stderr               record.Stream                    `json:"stderr"`
	IOTranscriptSHA256   record.SHA256                    `json:"io_transcript_sha256"`
	IOTranscriptRecords  record.Uint64String              `json:"io_transcript_records"`
	IOTranscriptComplete bool                             `json:"io_transcript_complete"`
	Choices              *ChoiceEvidence                  `json:"choices,omitempty"`
	World                record.World                     `json:"world"`
	ReadOnlyMountsSHA256 *record.SHA256                   `json:"read_only_mounts_sha256,omitempty"`
	SemanticCoverage     deterministicio.SemanticCoverage `json:"semantic_coverage"`
	ChoiceExploration    *ChoiceExplorationEvidence       `json:"choice_exploration,omitempty"`
}

type ChoiceExplorationEvidence struct {
	ImplementationSHA256 record.SHA256       `json:"implementation_sha256"`
	Round                record.Uint64String `json:"round"`
	CandidateSHA256      record.SHA256       `json:"candidate_sha256"`
	ParentSHA256         record.SHA256       `json:"parent_sha256,omitempty"`
	PrefixSHA256         record.SHA256       `json:"prefix_sha256,omitempty"`
	ForcedDepth          record.Uint64String `json:"forced_depth"`
	OutcomeSHA256        record.SHA256       `json:"outcome_sha256"`
}

func executionEvidence(
	config CampaignSpec,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	completion runCompletion,
	outcome execution.Classification,
	worldRecord record.World,
	mountArtifact *readonlymount.CapturedInputs,
	coverage deterministicio.SemanticCoverage,
	choiceFeatures *choice.FeatureProjection,
) ExecutionEvidence {
	profile := deterministicio.Default()
	runRecord := ExecutionEvidence{
		Schema:      ExecutionEvidenceSchema,
		Seed:        record.Uint64String(completion.job.seed),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Target:      prepared.RecordTarget(),
		IOProfile:   profile.Identity(),
		Environment: append([]record.Environment(nil), environmentForSeed(baseEnvironment, completion.job.seed)...),
		Limits: ExecutionLimitsEvidence{
			ExecutionTimeoutNanos: record.Uint64String(config.ExecutionTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit), IOTranscriptBytes: 64 << 20,
			ChoiceTraceBytes: record.Uint64String(config.ChoiceTraceLimit),
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
	if config.ChoiceTraceLimit != 0 && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalComplete {
		trace := completion.result.ChoiceTrace
		runRecord.Choices = &ChoiceEvidence{
			Profile: trace.Profile, ImplementationSHA256: record.SHA256FromSum(trace.ImplementationSHA256), Limit: record.Uint64String(trace.Limit),
			SHA256: record.SHA256FromSum(trace.Trace.SHA256), Records: record.Uint64String(trace.Trace.Summary.Records),
			BranchingRecords: record.Uint64String(trace.Trace.Summary.Branching), TerminalState: "complete",
			TapeSHA256: record.SHA256FromSum(trace.TapeSHA256), Decisions: record.Uint64String(trace.Decisions),
			Runnable: record.Uint64String(trace.Trace.Summary.Runnable), SelectPoll: record.Uint64String(trace.Trace.Summary.SelectPoll), SelectResult: record.Uint64String(trace.Trace.Summary.SelectResult),
			Features: []choice.Feature{},
		}
		if choiceFeatures != nil {
			runRecord.Choices.Features = append([]choice.Feature(nil), choiceFeatures.Values...)
			runRecord.Choices.AdjacentPairsObserved = record.Uint64String(choiceFeatures.AdjacentPairsObserved)
			runRecord.Choices.AdjacentPairsTruncated = choiceFeatures.AdjacentPairsTruncated
		}
	}
	if mountArtifact != nil {
		digest := record.SHA256(mountArtifact.Manifest.SHA256)
		runRecord.ReadOnlyMountsSHA256 = &digest
	}
	return runRecord
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
