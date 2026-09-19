package record

import (
	"bytes"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

type recordProjection struct {
	SchemaVersion     uint32                       `json:"schema_version"`
	Runner            Runner                       `json:"runner"`
	Toolchain         Toolchain                    `json:"toolchain"`
	Target            targetProjection             `json:"target"`
	IOProfile         ioProfileProjection          `json:"io_profile"`
	ChoiceProfile     *choiceProfileProjection     `json:"choice_profile,omitempty"`
	SimulationProfile *simulationProfileProjection `json:"simulation_profile,omitempty"`
	Minimization      *Minimization                `json:"minimization,omitempty"`
	Environment       []Environment                `json:"environment"`
	Limits            Limits                       `json:"limits"`
	Seed              Uint64String                 `json:"seed"`
	World             worldProjection              `json:"world"`
	Outcome           outcomeProjection            `json:"outcome"`
	Streams           streamsProjection            `json:"streams"`
}

type failureProjection struct {
	SchemaVersion uint32                   `json:"schema_version"`
	Toolchain     Toolchain                `json:"toolchain"`
	Target        targetProjection         `json:"target"`
	IOProfile     ioProfileProjection      `json:"io_profile"`
	ChoiceProfile *choiceProfileProjection `json:"choice_profile,omitempty"`
	Environment   []Environment            `json:"environment"`
	World         worldProjection          `json:"world"`
	Outcome       outcomeProjection        `json:"outcome"`
	StdoutSHA256  SHA256                   `json:"stdout_sha256"`
	StderrSHA256  SHA256                   `json:"stderr_sha256"`
}

type simulationFailureProjection struct {
	SchemaVersion    uint32            `json:"schema_version"`
	Toolchain        Toolchain         `json:"toolchain"`
	Target           targetProjection  `json:"target"`
	ControllerSHA256 SHA256            `json:"controller_sha256"`
	ExecutionSHA256  SHA256            `json:"execution_sha256"`
	FailureSHA256    SHA256            `json:"failure_sha256"`
	Outcome          outcomeProjection `json:"outcome"`
}

type targetProjection struct {
	Kind               string                              `json:"kind"`
	SHA256             SHA256                              `json:"sha256"`
	Size               Uint64String                        `json:"size"`
	Argv               []string                            `json:"argv"`
	BuildTags          []string                            `json:"build_tags"`
	Adapters           []TargetAdapter                     `json:"adapters"`
	Compatibility      []CompatibilityPack                 `json:"compatibility"`
	BuildInfo          BuildInfo                           `json:"build_info"`
	CapabilityMode     string                              `json:"capability_mode,omitempty"`
	CapabilityManifest *targetCapabilityManifestProjection `json:"capability_manifest,omitempty"`
}

type targetCapabilityManifestProjection struct {
	Schema                       string       `json:"schema"`
	SHA256                       SHA256       `json:"sha256"`
	Bytes                        Uint64String `json:"bytes"`
	Facts                        Uint64String `json:"facts"`
	ProducerImplementationSHA256 SHA256       `json:"producer_implementation_sha256"`
	GuardImplementationSHA256    SHA256       `json:"guard_implementation_sha256,omitempty"`
	CapabilityUniverseSHA256     SHA256       `json:"capability_universe_sha256"`
}

type ioProfileProjection struct {
	Name                 string                   `json:"name"`
	ImplementationSHA256 SHA256                   `json:"implementation_sha256"`
	Inventory            string                   `json:"inventory"`
	InventorySHA256      SHA256                   `json:"inventory_sha256"`
	Transcript           *ioTranscriptProjection  `json:"transcript,omitempty"`
	ReadOnlyMounts       *readOnlyMountProjection `json:"read_only_mounts,omitempty"`
}

type ioTranscriptProjection struct {
	Schema  string       `json:"schema"`
	SHA256  SHA256       `json:"sha256"`
	Bytes   Uint64String `json:"bytes"`
	Records Uint64String `json:"records"`
}

type choiceProfileProjection struct {
	Name                 string                `json:"name"`
	ImplementationSHA256 SHA256                `json:"implementation_sha256"`
	Trace                choiceTraceProjection `json:"trace"`
}

type choiceTraceProjection struct {
	Schema           string       `json:"schema"`
	SHA256           SHA256       `json:"sha256"`
	Bytes            Uint64String `json:"bytes"`
	Records          Uint64String `json:"records"`
	BranchingRecords Uint64String `json:"branching_records"`
	TerminalState    string       `json:"terminal_state"`
	Limit            Uint64String `json:"limit"`
	TapeSHA256       SHA256       `json:"tape_sha256,omitempty"`
	Decisions        Uint64String `json:"decisions,omitempty"`
}

type simulationProfileProjection struct {
	Name             string                      `json:"name"`
	ControllerSHA256 SHA256                      `json:"controller_sha256"`
	ExecutionSHA256  SHA256                      `json:"execution_sha256"`
	CandidateSHA256  SHA256                      `json:"candidate_sha256"`
	OutcomeSHA256    SHA256                      `json:"outcome_sha256"`
	FailureSHA256    SHA256                      `json:"failure_sha256,omitempty"`
	Plan             simulationPayloadProjection `json:"plan"`
	Record           simulationRecordProjection  `json:"record"`
}

type simulationPayloadProjection struct {
	Schema string       `json:"schema"`
	SHA256 SHA256       `json:"sha256"`
	Bytes  Uint64String `json:"bytes"`
}

type simulationRecordProjection struct {
	Schema string       `json:"schema"`
	SHA256 SHA256       `json:"sha256"`
	Bytes  Uint64String `json:"bytes"`
	Limit  Uint64String `json:"limit"`
}

type readOnlyMountProjection struct {
	Schema     string              `json:"schema"`
	SHA256     SHA256              `json:"sha256"`
	Bytes      Uint64String        `json:"bytes"`
	Entries    Uint64String        `json:"entries"`
	NotExist   Uint64String        `json:"not_exist,omitempty"`
	TotalBytes Uint64String        `json:"total_bytes"`
	Mappings   []string            `json:"mappings"`
	Limits     ReadOnlyMountLimits `json:"limits"`
}

type outcomeProjection struct {
	Domain      string        `json:"domain"`
	Reason      string        `json:"reason"`
	Termination string        `json:"termination"`
	ExitCode    *Uint64String `json:"exit_code"`
	Signal      *string       `json:"signal"`
	Deadline    *string       `json:"deadline"`
}

type worldProjection struct {
	Initial     worldPayloadProjection     `json:"initial"`
	Transitions worldTransitionsProjection `json:"transitions"`
	Final       worldPayloadProjection     `json:"final"`
	Adapters    []WorldAdapter             `json:"adapters"`
	Terminal    WorldTerminal              `json:"terminal"`
}

type worldPayloadProjection struct {
	Schema         string `json:"schema"`
	RawSHA256      SHA256 `json:"raw_sha256"`
	SemanticDigest SHA256 `json:"semantic_digest"`
}

type worldTransitionsProjection struct {
	Schema           string       `json:"schema"`
	RawSHA256        SHA256       `json:"raw_sha256"`
	Count            Uint64String `json:"count"`
	TranscriptDigest SHA256       `json:"transcript_digest"`
}

type streamsProjection struct {
	Stdout streamProjection `json:"stdout"`
	Stderr streamProjection `json:"stderr"`
}

type streamProjection struct {
	RetainedSHA256 SHA256       `json:"retained_sha256"`
	FullSHA256     SHA256       `json:"full_sha256"`
	TotalBytes     Uint64String `json:"total_bytes"`
	RetainedBytes  Uint64String `json:"retained_bytes"`
	DiscardedBytes Uint64String `json:"discarded_bytes"`
	Truncated      bool         `json:"truncated"`
}

func recordProjectionOf(manifest ExecutionRecord) recordProjection {
	return recordProjection{
		SchemaVersion:     manifest.SchemaVersion,
		Runner:            manifest.Runner,
		Toolchain:         manifest.Toolchain,
		Target:            projectTarget(manifest.Target),
		IOProfile:         projectIOProfile(manifest.IOProfile),
		ChoiceProfile:     projectChoiceProfile(manifest.ChoiceProfile),
		SimulationProfile: projectSimulationProfile(manifest.SimulationProfile),
		Minimization:      cloneMinimization(manifest.Minimization),
		Environment:       manifest.Environment,
		Limits:            manifest.Limits,
		Seed:              manifest.Seed,
		World:             projectWorld(manifest.World),
		Outcome:           projectOutcome(manifest.Outcome),
		Streams:           projectStreams(manifest.Streams),
	}
}

func cloneMinimization(minimization *Minimization) *Minimization {
	if minimization == nil {
		return nil
	}
	cloned := *minimization
	cloned.Accepted = make([]MinimizationReduction, len(minimization.Accepted))
	for index, reduction := range minimization.Accepted {
		cloned.Accepted[index] = reduction
		cloned.Accepted[index].Removed = append([]MinimizationDecision(nil), reduction.Removed...)
	}
	return &cloned
}

func failureProjectionOf(manifest ExecutionRecord) any {
	if profile := manifest.SimulationProfile; profile != nil {
		failure := profile.FailureSHA256
		if failure == "" {
			failure = profile.OutcomeSHA256
		}
		return simulationFailureProjection{
			SchemaVersion: manifest.SchemaVersion, Toolchain: manifest.Toolchain, Target: projectTarget(manifest.Target),
			ControllerSHA256: profile.ControllerSHA256, ExecutionSHA256: profile.ExecutionSHA256,
			FailureSHA256: failure, Outcome: projectOutcome(manifest.Outcome),
		}
	}
	environment := make([]Environment, 0, len(manifest.Environment))
	for _, entry := range manifest.Environment {
		if entry.Name != "GOMADSEED" {
			environment = append(environment, entry)
		}
	}
	return failureProjection{
		SchemaVersion: manifest.SchemaVersion,
		Toolchain:     manifest.Toolchain,
		Target:        projectTarget(manifest.Target),
		IOProfile:     projectIOProfile(manifest.IOProfile),
		ChoiceProfile: projectChoiceProfile(manifest.ChoiceProfile),
		Environment:   environment,
		World:         projectWorld(manifest.World),
		Outcome:       projectOutcome(manifest.Outcome),
		StdoutSHA256:  manifest.Streams.Stdout.FullSHA256,
		StderrSHA256:  manifest.Streams.Stderr.FullSHA256,
	}
}

func projectSimulationProfile(profile *SimulationProfile) *simulationProfileProjection {
	if profile == nil {
		return nil
	}
	return &simulationProfileProjection{
		Name: profile.Name, ControllerSHA256: profile.ControllerSHA256, ExecutionSHA256: profile.ExecutionSHA256,
		CandidateSHA256: profile.CandidateSHA256, OutcomeSHA256: profile.OutcomeSHA256, FailureSHA256: profile.FailureSHA256,
		Plan:   simulationPayloadProjection{Schema: profile.Plan.Schema, SHA256: profile.Plan.SHA256, Bytes: profile.Plan.Bytes},
		Record: simulationRecordProjection{Schema: profile.Record.Schema, SHA256: profile.Record.SHA256, Bytes: profile.Record.Bytes, Limit: profile.Record.Limit},
	}
}

func projectChoiceProfile(profile *ChoiceProfile) *choiceProfileProjection {
	if profile == nil {
		return nil
	}
	return &choiceProfileProjection{
		Name: profile.Name, ImplementationSHA256: profile.ImplementationSHA256,
		Trace: choiceTraceProjection{
			Schema: profile.Trace.Schema, SHA256: profile.Trace.SHA256, Bytes: profile.Trace.Bytes,
			Records: profile.Trace.Records, BranchingRecords: profile.Trace.BranchingRecords,
			TerminalState: profile.Trace.TerminalState, Limit: profile.Trace.Limit,
			TapeSHA256: profile.Trace.TapeSHA256, Decisions: profile.Trace.Decisions,
		},
	}
}

func projectIOProfile(profile IOProfile) ioProfileProjection {
	projected := ioProfileProjection{
		Name: profile.Name, ImplementationSHA256: profile.ImplementationSHA256, Inventory: profile.Inventory, InventorySHA256: profile.InventorySHA256,
	}
	if profile.Transcript != nil {
		projected.Transcript = &ioTranscriptProjection{
			Schema: profile.Transcript.Schema, SHA256: profile.Transcript.SHA256, Bytes: profile.Transcript.Bytes, Records: profile.Transcript.Records,
		}
	}
	if profile.ReadOnlyMounts != nil {
		mounts := profile.ReadOnlyMounts
		projected.ReadOnlyMounts = &readOnlyMountProjection{
			Schema: mounts.Schema, SHA256: mounts.SHA256, Bytes: mounts.Bytes, Entries: mounts.Entries,
			NotExist: mounts.NotExist, TotalBytes: mounts.TotalBytes, Mappings: append([]string(nil), mounts.Mappings...), Limits: mounts.Limits,
		}
	}
	return projected
}

func projectTarget(target Target) targetProjection {
	projected := targetProjection{
		Kind: target.Kind, SHA256: target.SHA256, Size: target.Size, Argv: target.Argv, BuildTags: target.BuildTags,
		Adapters: target.Adapters, Compatibility: target.Compatibility, BuildInfo: target.BuildInfo, CapabilityMode: target.CapabilityMode,
	}
	if manifest := target.CapabilityManifest; manifest != nil {
		projected.CapabilityManifest = &targetCapabilityManifestProjection{
			Schema: manifest.Schema, SHA256: manifest.SHA256, Bytes: manifest.Bytes, Facts: manifest.Facts,
			ProducerImplementationSHA256: manifest.ProducerImplementationSHA256,
			GuardImplementationSHA256:    manifest.GuardImplementationSHA256,
			CapabilityUniverseSHA256:     manifest.CapabilityUniverseSHA256,
		}
	}
	return projected
}

func cloneTargetCapabilityManifest(manifest *TargetCapabilityManifest) *TargetCapabilityManifest {
	if manifest == nil {
		return nil
	}
	cloned := *manifest
	return &cloned
}

func SameTargetIdentity(left, right Target) (bool, error) {
	leftBytes, err := canonicaljson.CanonicalJSON(projectTarget(left))
	if err != nil {
		return false, err
	}
	rightBytes, err := canonicaljson.CanonicalJSON(projectTarget(right))
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftBytes, rightBytes), nil
}

func projectOutcome(outcome Outcome) outcomeProjection {
	return outcomeProjection{Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, ExitCode: outcome.ExitCode, Signal: outcome.Signal, Deadline: outcome.Deadline}
}

func projectWorld(value World) worldProjection {
	return worldProjection{
		Initial: worldPayloadProjection{Schema: value.Initial.Schema, RawSHA256: value.Initial.RawSHA256, SemanticDigest: value.Initial.SemanticDigest},
		Transitions: worldTransitionsProjection{
			Schema: value.Transitions.Schema, RawSHA256: value.Transitions.RawSHA256, Count: value.Transitions.Count,
			TranscriptDigest: value.Transitions.TranscriptDigest,
		},
		Final:    worldPayloadProjection{Schema: value.Final.Schema, RawSHA256: value.Final.RawSHA256, SemanticDigest: value.Final.SemanticDigest},
		Adapters: append([]WorldAdapter(nil), value.Adapters...),
		Terminal: value.Terminal,
	}
}

func projectStreams(value Streams) streamsProjection {
	return streamsProjection{Stdout: projectStream(value.Stdout), Stderr: projectStream(value.Stderr)}
}

func projectStream(value Stream) streamProjection {
	return streamProjection{
		RetainedSHA256: value.RetainedSHA256, FullSHA256: value.FullSHA256, TotalBytes: value.TotalBytes,
		RetainedBytes: value.RetainedBytes, DiscardedBytes: value.DiscardedBytes, Truncated: value.Truncated,
	}
}
