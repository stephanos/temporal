package runner

import (
	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
)

const (
	ExecutionEvidenceSchema       = "gomadv3.run-evidence/v4"
	PriorExecutionEvidenceSchema  = "gomadv3.run-evidence/v3"
	ChoiceExecutionEvidenceSchema = "gomadv3.run-evidence/v2"
	LegacyExecutionEvidenceSchema = "gomadv3.run-evidence/v1"
)

type IOProfileEvidence = deterministicio.Contract

type OutcomeEvidence struct {
	Domain      string                 `json:"domain"`
	Reason      string                 `json:"reason"`
	Termination string                 `json:"termination"`
	ExitCode    *evidence.Uint64String `json:"exit_code,omitempty"`
	Signal      *string                `json:"signal,omitempty"`
	Deadline    *string                `json:"deadline,omitempty"`
}

type ExecutionLimitsEvidence struct {
	RunTimeoutNanos      evidence.Uint64String `json:"run_timeout_nanos"`
	TerminateGraceNanos  evidence.Uint64String `json:"terminate_grace_nanos"`
	OutputBytes          evidence.Uint64String `json:"output_bytes"`
	WorldTransitionBytes evidence.Uint64String `json:"world_transition_bytes"`
	IOTranscriptBytes    evidence.Uint64String `json:"io_transcript_bytes"`
	ChoiceTraceBytes     evidence.Uint64String `json:"choice_trace_bytes,omitempty"`
}

type ChoiceEvidence struct {
	Profile                string                `json:"profile"`
	ImplementationSHA256   evidence.SHA256       `json:"implementation_sha256"`
	Limit                  evidence.Uint64String `json:"limit"`
	SHA256                 evidence.SHA256       `json:"sha256"`
	Records                evidence.Uint64String `json:"records"`
	BranchingRecords       evidence.Uint64String `json:"branching_records"`
	TerminalState          string                `json:"terminal_state"`
	TapeSHA256             evidence.SHA256       `json:"tape_sha256"`
	Decisions              evidence.Uint64String `json:"decisions"`
	Runnable               evidence.Uint64String `json:"runnable"`
	SelectPoll             evidence.Uint64String `json:"select_poll"`
	SelectResult           evidence.Uint64String `json:"select_result"`
	Features               []choice.Feature      `json:"features"`
	AdjacentPairsObserved  evidence.Uint64String `json:"adjacent_pairs_observed"`
	AdjacentPairsTruncated bool                  `json:"adjacent_pairs_truncated"`
}

type ExecutionEvidence struct {
	Schema               string                           `json:"schema"`
	Seed                 evidence.Uint64String            `json:"seed"`
	RunnerBuild          string                           `json:"runner_build"`
	Toolchain            evidence.Toolchain               `json:"toolchain"`
	Target               evidence.Target                  `json:"target"`
	IOProfile            IOProfileEvidence                `json:"io_profile"`
	Environment          []evidence.Environment           `json:"environment"`
	Limits               ExecutionLimitsEvidence          `json:"limits"`
	Outcome              OutcomeEvidence                  `json:"outcome"`
	GroupGone            bool                             `json:"group_gone"`
	Stdout               evidence.Stream                  `json:"stdout"`
	Stderr               evidence.Stream                  `json:"stderr"`
	IOTranscriptSHA256   evidence.SHA256                  `json:"io_transcript_sha256"`
	IOTranscriptRecords  evidence.Uint64String            `json:"io_transcript_records"`
	IOTranscriptComplete bool                             `json:"io_transcript_complete"`
	Choices              *ChoiceEvidence                  `json:"choices,omitempty"`
	World                evidence.World                   `json:"world"`
	ReadOnlyMountsSHA256 *evidence.SHA256                 `json:"read_only_mounts_sha256,omitempty"`
	SemanticCoverage     deterministicio.SemanticCoverage `json:"semantic_coverage"`
	Frontier             *FrontierRunEvidence             `json:"frontier,omitempty"`
}

type FrontierRunEvidence struct {
	ImplementationSHA256 evidence.SHA256       `json:"implementation_sha256"`
	Round                evidence.Uint64String `json:"round"`
	CandidateSHA256      evidence.SHA256       `json:"candidate_sha256"`
	ParentSHA256         evidence.SHA256       `json:"parent_sha256,omitempty"`
	PrefixSHA256         evidence.SHA256       `json:"prefix_sha256,omitempty"`
	ForcedDepth          evidence.Uint64String `json:"forced_depth"`
	OutcomeSHA256        evidence.SHA256       `json:"outcome_sha256"`
}

func runEvidence(
	config CampaignSpec,
	prepared target.Prepared,
	baseEnvironment []evidence.Environment,
	completion runCompletion,
	outcome execution.Classification,
	worldRecord evidence.World,
	mountArtifact *deterministicio.CapturedInputs,
	coverage deterministicio.SemanticCoverage,
	choiceFeatures *choice.FeatureProjection,
) ExecutionEvidence {
	profile := deterministicio.Default()
	runRecord := ExecutionEvidence{
		Schema:      ExecutionEvidenceSchema,
		Seed:        evidence.Uint64String(completion.job.seed),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Target:      prepared.RecordTarget(),
		IOProfile:   profile.Identity(),
		Environment: append([]evidence.Environment(nil), environmentForSeed(baseEnvironment, completion.job.seed)...),
		Limits: ExecutionLimitsEvidence{
			RunTimeoutNanos: evidence.Uint64String(config.RunTimeout), TerminateGraceNanos: evidence.Uint64String(config.TerminateGrace), OutputBytes: evidence.Uint64String(config.OutputLimit),
			WorldTransitionBytes: evidence.Uint64String(config.WorldTransitionLimit), IOTranscriptBytes: 64 << 20,
			ChoiceTraceBytes: evidence.Uint64String(config.ChoiceTraceLimit),
		},
		Outcome: OutcomeEvidence{
			Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
			ExitCode: cloneUint64String(outcome.ExitCode), Signal: cloneString(outcome.Signal), Deadline: cloneString(outcome.Deadline),
		},
		GroupGone:            completion.result.GroupGone,
		Stdout:               streamRecord(completion.result.Stdout),
		Stderr:               streamRecord(completion.result.Stderr),
		IOTranscriptSHA256:   evidence.SHA256FromSum(completion.result.IOTranscript.SHA256),
		IOTranscriptRecords:  evidence.Uint64String(completion.result.IOTranscript.Records),
		IOTranscriptComplete: completion.result.IOTranscript.Complete,
		World:                cloneWorld(worldRecord),
		SemanticCoverage:     coverage,
	}
	if config.ChoiceTraceLimit != 0 && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalComplete {
		trace := completion.result.ChoiceTrace
		runRecord.Choices = &ChoiceEvidence{
			Profile: trace.Profile, ImplementationSHA256: evidence.SHA256FromSum(trace.ImplementationSHA256), Limit: evidence.Uint64String(trace.Limit),
			SHA256: evidence.SHA256FromSum(trace.Trace.SHA256), Records: evidence.Uint64String(trace.Trace.Summary.Records),
			BranchingRecords: evidence.Uint64String(trace.Trace.Summary.Branching), TerminalState: "complete",
			TapeSHA256: evidence.SHA256FromSum(trace.TapeSHA256), Decisions: evidence.Uint64String(trace.Decisions),
			Runnable: evidence.Uint64String(trace.Trace.Summary.Runnable), SelectPoll: evidence.Uint64String(trace.Trace.Summary.SelectPoll), SelectResult: evidence.Uint64String(trace.Trace.Summary.SelectResult),
			Features: []choice.Feature{},
		}
		if choiceFeatures != nil {
			runRecord.Choices.Features = append([]choice.Feature(nil), choiceFeatures.Values...)
			runRecord.Choices.AdjacentPairsObserved = evidence.Uint64String(choiceFeatures.AdjacentPairsObserved)
			runRecord.Choices.AdjacentPairsTruncated = choiceFeatures.AdjacentPairsTruncated
		}
	}
	if mountArtifact != nil {
		digest := evidence.SHA256(mountArtifact.Manifest.SHA256)
		runRecord.ReadOnlyMountsSHA256 = &digest
	}
	return runRecord
}

func cloneBuildInfo(buildInfo evidence.BuildInfo) evidence.BuildInfo {
	buildInfo.Settings = append([]evidence.BuildSetting(nil), buildInfo.Settings...)
	return buildInfo
}

func cloneCompatibility(packs []evidence.CompatibilityPack) []evidence.CompatibilityPack {
	return append([]evidence.CompatibilityPack{}, packs...)
}

func cloneAdapters(adapters []evidence.TargetAdapter) []evidence.TargetAdapter {
	return append([]evidence.TargetAdapter{}, adapters...)
}

func cloneWorld(worldRecord evidence.World) evidence.World {
	worldRecord.Adapters = append([]evidence.WorldAdapter(nil), worldRecord.Adapters...)
	return worldRecord
}

func cloneUint64String(value *evidence.Uint64String) *evidence.Uint64String {
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
