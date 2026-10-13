package runner

import (
	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	captured "go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func executionEvidence(
	config campaignRequest,
	prepared target.Prepared,
	baseEnvironment []record.Environment,
	completion runCompletion,
	outcome backend.Classification,
	worldRecord record.World,
	mountArtifact *captured.CapturedInputs,
	coverage record.SemanticCoverage,
	choiceFeatures *choice.FeatureProjection,
) record.ExecutionEvidence {
	profile := config.profile
	runRecord := record.ExecutionEvidence{
		Schema:      record.ExecutionEvidenceSchema,
		Seed:        record.Uint64String(completion.job.seed),
		RunnerBuild: config.RunnerBuild,
		Toolchain:   prepared.RecordToolchain(),
		Target:      prepared.RecordTarget(),
		IOProfile:   profile.Identity(),
		Environment: append([]record.Environment(nil), environmentForSeed(baseEnvironment, completion.job.seed)...),
		Limits: record.ExecutionLimitsEvidence{
			ExecutionTimeoutNanos: record.Uint64String(config.ExecutionTimeout), TerminateGraceNanos: record.Uint64String(config.TerminateGrace), OutputBytes: record.Uint64String(config.OutputLimit),
			WorldTransitionBytes: record.Uint64String(config.WorldTransitionLimit), IOTranscriptBytes: record.Uint64String(ioTranscriptLimit(config)),
			ChoiceTraceBytes: record.Uint64String(config.ChoiceTraceLimit),
		},
		Outcome: record.OutcomeEvidence{
			SoftAssertions: outcome.SoftAssertions,
			Domain:         outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination,
			ExitCode: cloneUint64String(outcome.ExitCode), Signal: cloneString(outcome.Signal), Deadline: cloneString(outcome.Deadline),
		},
		GroupGone:               completion.result.GroupGone,
		Stdout:                  streamRecord(completion.result.Stdout),
		Stderr:                  streamRecord(completion.result.Stderr),
		IOTranscriptSHA256:      record.SHA256FromSum(completion.result.IOTranscript.SHA256),
		IOTranscriptRecords:     record.Uint64String(completion.result.IOTranscript.Records),
		IOTranscriptComplete:    completion.result.IOTranscript.Complete,
		VirtualTimeElapsedNanos: record.Uint64String(completion.result.VirtualTimeElapsedNanos),
		World:                   cloneWorld(worldRecord),
		SemanticCoverage:        coverage,
	}
	if prepared.Backend != nil {
		backendProfile := record.BackendIOProfile(*prepared.Backend)
		runRecord.IOProfile = record.IOContract{
			Name: backendProfile.Name, ImplementationSHA256: record.SHA256(backendProfile.ImplementationSHA256), InventorySHA256: record.SHA256(backendProfile.InventorySHA256),
		}
		runRecord.Target.Backend = recordedBackend(prepared, completion.result)
		if len(completion.result.BackendEvidence) != 0 {
			runRecord.BackendEvidenceSHA256 = record.HashBytes(completion.result.BackendEvidence)
		}
	}
	if config.Diagnostics && len(completion.result.DiagnosticTrace.Bytes) != 0 {
		trace := completion.result.DiagnosticTrace
		runRecord.Diagnostics = &record.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: record.Uint64String(len(trace.Records))}
	}
	if config.ChoiceTraceLimit != 0 && completion.result.ChoiceTrace.Trace.Summary.Terminal == choice.TerminalComplete {
		trace := completion.result.ChoiceTrace
		runRecord.Choices = &record.ChoiceEvidence{
			Profile: trace.Profile, ImplementationSHA256: record.SHA256FromSum(trace.ImplementationSHA256), Limit: record.Uint64String(trace.Limit),
			SHA256: record.SHA256FromSum(trace.Trace.SHA256), Records: record.Uint64String(trace.Trace.Summary.Records),
			BranchingRecords: record.Uint64String(trace.Trace.Summary.Branching), TerminalState: "complete",
			TapeSHA256: record.SHA256FromSum(trace.TapeSHA256), Decisions: record.Uint64String(trace.Decisions),
			Runnable: record.Uint64String(trace.Trace.Summary.Runnable), SelectPoll: record.Uint64String(trace.Trace.Summary.SelectPoll), SelectResult: record.Uint64String(trace.Trace.Summary.SelectResult),
			PeakGoroutines: record.Uint64String(trace.Trace.Summary.PeakGoroutines),
			Features:       []choice.Feature{},
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
