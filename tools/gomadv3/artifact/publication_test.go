package artifact

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/record"
)

func TestPublishArtifactWritesAndValidatesChoiceTrace(t *testing.T) {
	input := executionArtifactInput(t)
	trace := choiceTracePayload(t)
	tapeSHA256, decisions := choiceTapeMetadata(t, input.Manifest, trace)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name:                 choice.Profile,
		ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: record.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: record.HashBytes(trace), Bytes: record.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: tapeSHA256, Decisions: record.Uint64String(decisions),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	published, err := PublishArtifact(Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.File != "choices.bin" || opened.Manifest.ChoiceProfile.Trace.TapeSHA256 != tapeSHA256 || opened.Manifest.ChoiceProfile.Trace.Decisions != 1 {
		t.Fatalf("choice profile = %#v", opened.Manifest.ChoiceProfile)
	}
	observed, err := ReadPayload(opened, "choices.bin", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if string(observed) != string(trace) {
		t.Fatalf("choice trace = %q", observed)
	}
}

func TestPublishArtifactRejectsMalformedChoiceTraceWithMatchingIdentity(t *testing.T) {
	input := executionArtifactInput(t)
	payloadBytes, err := choice.TracePayloadBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	trace := make([]byte, payloadBytes)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: record.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: record.HashBytes(trace), Bytes: record.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 0, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: record.HashBytes([]byte("invalid tape")),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	if _, err := PublishArtifact(Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "validate choice trace payload") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func TestPublishArtifactRejectsChangedChoiceTraceIdentity(t *testing.T) {
	input := executionArtifactInput(t)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: record.HashBytes([]byte("choice implementation")),
		Trace: record.ChoiceTrace{Schema: "gomadv3.choice-trace/v2", SHA256: record.HashBytes([]byte("expected")), Bytes: 8, Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: 1 << 20, TapeSHA256: record.HashBytes([]byte("tape")), Decisions: 1},
	}
	input.Manifest.Limits.ChoiceTraceBytes = 1 << 20
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = []byte("changed!")
	if _, err := PublishArtifact(Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "choice trace implementation identity") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func TestPublishArtifactRejectsChangedChoiceTapeIdentity(t *testing.T) {
	input := executionArtifactInput(t)
	trace := choiceTracePayload(t)
	_, decisions := choiceTapeMetadata(t, input.Manifest, trace)
	input.Manifest.ChoiceProfile = &record.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: record.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: record.HashBytes(trace), Bytes: record.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: record.HashBytes([]byte("different tape")), Decisions: record.Uint64String(decisions),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, record.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	if _, err := PublishArtifact(Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "choice tape") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func TestPublishArtifactWritesSimulationExplorationEvidence(t *testing.T) {
	input := executionArtifactInput(t)
	plan := []byte(`{"schema":"gomadv3.simulation-exploration-plan/v1"}`)
	record := []byte(`{"schema":"gomadv3.cluster-record/v7"}`)
	input.Manifest.SimulationProfile = simulationProfile(plan, record)
	input.Simulation = &SimulationPayloads{Plan: plan, Record: record}

	published, err := PublishArtifact(Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	for name, want := range map[string][]byte{
		"simulation/plan.json":   plan,
		"simulation/record.json": record,
	} {
		got, err := ReadPayload(opened, name, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("payload %s = %q, want %q", name, got, want)
		}
	}
}

func TestPublishArtifactRejectsChangedSimulationExplorationEvidence(t *testing.T) {
	input := executionArtifactInput(t)
	plan := []byte(`{"schema":"gomadv3.simulation-exploration-plan/v1"}`)
	record := []byte(`{"schema":"gomadv3.cluster-record/v7"}`)
	input.Manifest.SimulationProfile = simulationProfile(plan, record)
	input.Simulation = &SimulationPayloads{Plan: []byte("changed"), Record: record}

	if _, err := PublishArtifact(Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "simulation plan identity changed") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func executionArtifactInput(t *testing.T) ArtifactInput {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	worldRecord, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	stdout := []byte("stdout")
	stderr := []byte("stderr")
	profile := deterministicio.Default()
	return ArtifactInput{
		Manifest: record.ExecutionRecord{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "batch-1", SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: record.Target{
				Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
				Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure",
			},
			IOProfile:   record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())},
			Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
			Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
			World:       worldRecord,
			Outcome:     record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
			Streams: record.Streams{
				Stdout: record.Stream{FullSHA256: record.HashBytes(stdout), TotalBytes: record.Uint64String(len(stdout)), RetainedBytes: record.Uint64String(len(stdout))},
				Stderr: record.Stream{FullSHA256: record.HashBytes(stderr), TotalBytes: record.Uint64String(len(stderr)), RetainedBytes: record.Uint64String(len(stderr))},
			},
			Host: record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: 1},
		},
		TargetPath: targetPath, Stdout: stdout, Stderr: stderr, World: worldPayloads,
	}
}

func simulationProfile(plan, recordBytes []byte) *record.SimulationProfile {
	return &record.SimulationProfile{
		Name: "gomadv3-simulation-exploration/v1", ControllerSHA256: record.HashBytes([]byte("controller")),
		ExecutionSHA256: record.HashBytes([]byte("execution")), CandidateSHA256: record.HashBytes([]byte("candidate")),
		OutcomeSHA256: record.HashBytes([]byte("outcome")), FailureSHA256: record.HashBytes([]byte("failure")),
		Plan: record.SimulationPlan{
			Schema: "gomadv3.simulation-exploration-plan/v1", File: "simulation/plan.json",
			SHA256: record.HashBytes(plan), Bytes: record.Uint64String(len(plan)),
		},
		Record: record.SimulationRecord{
			Schema: "gomadv3.cluster-record/v7", File: "simulation/record.json",
			SHA256: record.HashBytes(recordBytes), Bytes: record.Uint64String(len(recordBytes)), Limit: 128 << 20,
		},
	}
}

func choiceTracePayload(t *testing.T) []byte {
	t.Helper()
	first := sha256.Sum256([]byte("first alternative"))
	second := sha256.Sum256([]byte("second alternative"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 7, false, [][sha256.Size]byte{first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	return trace.Bytes
}

func choiceTapeMetadata(t *testing.T, manifest record.ExecutionRecord, payload []byte) (record.SHA256, uint64) {
	t.Helper()
	digest := sha256.Sum256(payload)
	records, err := choice.TraceRecordCount(payload)
	if err != nil {
		t.Fatal(err)
	}
	limit, err := choice.TraceBytes(records)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.DecodeStoredTrace(choice.Profile, payload, choice.TerminalMetadata{
		State: choice.TerminalComplete, Limit: limit, Records: records, SHA256: digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	targetSHA256, err := manifest.Target.SHA256.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := choice.ImplementationIdentity(manifest.Toolchain.BuildKey)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, choice.ExecutionIdentity{
		TargetSHA256: targetSHA256, ToolchainBuildKey: manifest.Toolchain.BuildKey,
		GOOS: manifest.Toolchain.TargetGOOS, GOARCH: manifest.Toolchain.TargetGOARCH, ImplementationSHA256: implementation,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record.SHA256FromSum(tape.SHA256), uint64(len(tape.Decisions))
}

func choiceImplementationIdentity(t *testing.T, buildKey string) record.SHA256 {
	t.Helper()
	implementation, err := choice.ImplementationIdentity(buildKey)
	if err != nil {
		t.Fatal(err)
	}
	return record.SHA256FromSum(implementation)
}
