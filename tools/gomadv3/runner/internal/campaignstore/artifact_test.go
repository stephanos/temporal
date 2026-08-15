package campaignstore

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/choice"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestPublishArtifactWritesAndValidatesChoiceTrace(t *testing.T) {
	input := artifactInput(t)
	trace := choiceTracePayload(t)
	tapeSHA256, decisions := choiceTapeMetadata(t, input.Manifest, trace)
	input.Manifest.ChoiceProfile = &evidence.ChoiceProfile{
		Name:                 choice.Profile,
		ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: evidence.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: evidence.HashBytes(trace), Bytes: evidence.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: tapeSHA256, Decisions: evidence.Uint64String(decisions),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right evidence.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	published, err := PublishArtifact(evidence.Store{Root: t.TempDir()}, input)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := evidence.OpenArtifact(published.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if opened.Manifest.ChoiceProfile == nil || opened.Manifest.ChoiceProfile.Trace.File != "choices.bin" || opened.Manifest.ChoiceProfile.Trace.TapeSHA256 != tapeSHA256 || opened.Manifest.ChoiceProfile.Trace.Decisions != 1 {
		t.Fatalf("choice profile = %#v", opened.Manifest.ChoiceProfile)
	}
	observed, err := evidence.ReadPayload(opened, "choices.bin", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if string(observed) != string(trace) {
		t.Fatalf("choice trace = %q", observed)
	}
}

func TestPublishArtifactRejectsMalformedChoiceTraceWithMatchingIdentity(t *testing.T) {
	input := artifactInput(t)
	payloadBytes, err := choice.TracePayloadBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	trace := make([]byte, payloadBytes)
	input.Manifest.ChoiceProfile = &evidence.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: evidence.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: evidence.HashBytes(trace), Bytes: evidence.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 0, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: evidence.HashBytes([]byte("invalid tape")),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right evidence.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	if _, err := PublishArtifact(evidence.Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "validate choice trace payload") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func TestPublishArtifactRejectsChangedChoiceTraceIdentity(t *testing.T) {
	input := artifactInput(t)
	input.Manifest.ChoiceProfile = &evidence.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: evidence.HashBytes([]byte("choice implementation")),
		Trace: evidence.ChoiceTrace{Schema: "gomadv3.choice-trace/v2", SHA256: evidence.HashBytes([]byte("expected")), Bytes: 8, Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: 1 << 20, TapeSHA256: evidence.HashBytes([]byte("tape")), Decisions: 1},
	}
	input.Manifest.Limits.ChoiceTraceBytes = 1 << 20
	input.Manifest.Environment = append(input.Manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right evidence.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = []byte("changed!")
	if _, err := PublishArtifact(evidence.Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "choice trace implementation identity") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func TestPublishArtifactRejectsChangedChoiceTapeIdentity(t *testing.T) {
	input := artifactInput(t)
	trace := choiceTracePayload(t)
	_, decisions := choiceTapeMetadata(t, input.Manifest, trace)
	input.Manifest.ChoiceProfile = &evidence.ChoiceProfile{
		Name: choice.Profile, ImplementationSHA256: choiceImplementationIdentity(t, input.Manifest.Toolchain.BuildKey),
		Trace: evidence.ChoiceTrace{
			Schema: "gomadv3.choice-trace/v2", SHA256: evidence.HashBytes(trace), Bytes: evidence.Uint64String(len(trace)),
			Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: choice.MinimumTraceBytes,
			TapeSHA256: evidence.HashBytes([]byte("different tape")), Decisions: evidence.Uint64String(decisions),
		},
	}
	input.Manifest.Limits.ChoiceTraceBytes = choice.MinimumTraceBytes
	input.Manifest.Environment = append(input.Manifest.Environment, evidence.Environment{Name: "GOMADV3_CHOICE_PROFILE", Value: choice.Profile})
	slices.SortFunc(input.Manifest.Environment, func(left, right evidence.Environment) int { return strings.Compare(left.Name, right.Name) })
	input.ChoiceTrace = trace

	if _, err := PublishArtifact(evidence.Store{Root: t.TempDir()}, input); err == nil || !strings.Contains(err.Error(), "choice tape") {
		t.Fatalf("PublishArtifact() error = %v", err)
	}
}

func artifactInput(t *testing.T) ArtifactInput {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	worldRecord, worldPayloads := evidence.NoneWorld()
	exitCode := evidence.Uint64String(2)
	stdout := []byte("stdout")
	stderr := []byte("stderr")
	profile := deterministicio.Default()
	return ArtifactInput{
		Manifest: evidence.ExecutionRecord{
			SchemaVersion: evidence.SchemaVersion, ArtifactKind: evidence.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "batch-1", SelectionOrdinal: 0, Seed: 7, ReplayMode: evidence.ReplayExact,
			Runner:    evidence.Runner{RecordContract: evidence.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: evidence.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: evidence.Target{
				Kind: "go-run", Source: ".", SHA256: evidence.HashBytes(targetBytes), Size: evidence.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
				Adapters: []evidence.TargetAdapter{}, Compatibility: []evidence.CompatibilityPack{}, BuildInfo: evidence.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure",
			},
			IOProfile:   evidence.IOProfile{Name: profile.Name(), ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: evidence.SHA256(profile.InventorySHA256())},
			Environment: []evidence.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
			Limits:      evidence.Limits{RunTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
			World:       worldRecord,
			Outcome:     evidence.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
			Streams: evidence.Streams{
				Stdout: evidence.Stream{FullSHA256: evidence.HashBytes(stdout), TotalBytes: evidence.Uint64String(len(stdout)), RetainedBytes: evidence.Uint64String(len(stdout))},
				Stderr: evidence.Stream{FullSHA256: evidence.HashBytes(stderr), TotalBytes: evidence.Uint64String(len(stderr)), RetainedBytes: evidence.Uint64String(len(stderr))},
			},
			Host: evidence.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: 1},
		},
		TargetPath: targetPath, Stdout: stdout, Stderr: stderr, World: worldPayloads,
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

func choiceTapeMetadata(t *testing.T, manifest evidence.ExecutionRecord, payload []byte) (evidence.SHA256, uint64) {
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
	return evidence.SHA256FromSum(tape.SHA256), uint64(len(tape.Decisions))
}

func choiceImplementationIdentity(t *testing.T, buildKey string) evidence.SHA256 {
	t.Helper()
	implementation, err := choice.ImplementationIdentity(buildKey)
	if err != nil {
		t.Fatal(err)
	}
	return evidence.SHA256FromSum(implementation)
}
