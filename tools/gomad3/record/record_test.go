package record

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

func TestSHA256FromSumUsesCanonicalIdentity(t *testing.T) {
	sum := sha256.Sum256([]byte("payload"))
	if got, want := SHA256FromSum(sum), SHA256("sha256:239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"); got != want {
		t.Fatalf("SHA256FromSum() = %q, want %q", got, want)
	}
}

func TestParseSHA256RoundTripsCanonicalIdentity(t *testing.T) {
	want := sha256.Sum256([]byte("payload"))
	identity, err := ParseSHA256(string(SHA256FromSum(want)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := identity.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("SHA256.Bytes() = %x, want %x", got, want)
	}
}

func TestParseSHA256RejectsNonCanonicalIdentity(t *testing.T) {
	for _, value := range []string{
		"",
		strings.Repeat("0", sha256.Size*2),
		"sha256:" + strings.Repeat("0", sha256.Size*2-1),
		"sha256:" + strings.Repeat("A", sha256.Size*2),
		"sha256:" + strings.Repeat("z", sha256.Size*2),
	} {
		if _, err := ParseSHA256(value); err == nil {
			t.Fatalf("ParseSHA256(%q) succeeded", value)
		}
	}
}

func TestDomainHashUsesNamedNULTerminatedDomain(t *testing.T) {
	if got, want := DomainHash("gomad3-run-record-v1", []byte("payload")), SHA256("sha256:087406f9758d2fbb56f25c4a24ef6fbc9986ba6108814b05105ae598447940a5"); got != want {
		t.Fatalf("DomainHash() = %q, want %q", got, want)
	}
}

func TestFinalizeManifestSeparatesRunAndFailureIdentity(t *testing.T) {
	first, firstBytes := finalizedManifest(t, manifestFixture())
	secondInput := manifestFixture()
	secondInput.Seed = 99
	secondInput.Environment[0].Value = "99"
	second, _ := finalizedManifest(t, secondInput)
	if first.RecordHash == second.RecordHash {
		t.Fatal("record hash did not include seed")
	}
	if first.Outcome.FailureSignature != second.Outcome.FailureSignature {
		t.Fatal("failure signature included seed")
	}
	changedLimits := manifestFixture()
	changedLimits.Limits.OutputBytes++
	withChangedLimits, _ := finalizedManifest(t, changedLimits)
	if first.RecordHash == withChangedLimits.RecordHash {
		t.Fatal("record hash omitted execution limits")
	}
	if first.Outcome.FailureSignature != withChangedLimits.Outcome.FailureSignature {
		t.Fatal("failure signature included execution limits")
	}

	changedOutput := manifestFixture()
	changedOutput.Streams.Stdout.FullSHA256 = HashBytes([]byte("different complete stdout"))
	changedOutput.Streams.Stdout.TotalBytes++
	changedOutput.Streams.Stdout.DiscardedBytes = 1
	changedOutput.Streams.Stdout.Truncated = true
	third, _ := finalizedManifest(t, changedOutput)
	if first.RecordHash == third.RecordHash {
		t.Fatal("record hash omitted complete stdout hash")
	}
	if first.Outcome.FailureSignature == third.Outcome.FailureSignature {
		t.Fatal("failure signature omitted complete stdout hash")
	}

	decoded, err := DecodeExecutionRecord(firstBytes)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.RecordHash != first.RecordHash || decoded.Outcome.FailureSignature != first.Outcome.FailureSignature {
		t.Fatal("decoded manifest identity changed")
	}
}

func TestFinalizeManifestSeparatesExactSimulationReplayFromNormalizedFailure(t *testing.T) {
	failure := HashBytes([]byte("normalized simulation failure"))
	firstInput := manifestWithSimulationProfile(manifestFixture(), []byte("plan one"), []byte("record one"), failure)
	first, _ := finalizedManifest(t, firstInput)

	changedReplayInput := manifestWithSimulationProfile(manifestFixture(), []byte("plan two"), []byte("record two"), failure)
	changedReplay, _ := finalizedManifest(t, changedReplayInput)
	if first.RecordHash == changedReplay.RecordHash {
		t.Fatal("record hash omitted the exact simulation replay payloads")
	}
	if first.Outcome.FailureSignature != changedReplay.Outcome.FailureSignature {
		t.Fatal("normalized failure signature included the exact simulation replay payloads")
	}

	changedFailureInput := manifestWithSimulationProfile(manifestFixture(), []byte("plan two"), []byte("record two"), HashBytes([]byte("different simulation failure")))
	changedFailure, _ := finalizedManifest(t, changedFailureInput)
	if first.Outcome.FailureSignature == changedFailure.Outcome.FailureSignature {
		t.Fatal("normalized failure signature omitted the simulation failure identity")
	}
}

func TestFinalizeManifestBindsMinimizationLineageWithoutChangingFailureIdentity(t *testing.T) {
	failure := HashBytes([]byte("normalized simulation failure"))
	parentInput := manifestWithSimulationProfile(manifestFixture(), []byte("parent plan"), []byte("parent record"), failure)
	parent, _ := finalizedManifest(t, parentInput)
	minimizedInput := manifestWithSimulationProfile(manifestFixture(), []byte("minimized plan"), []byte("minimized record"), failure)
	originalCandidate := HashBytes([]byte("before"))
	finalCandidate := minimizedInput.SimulationProfile.CandidateSHA256
	minimizedInput.Minimization = &Minimization{
		Schema: "gomad3.minimization/v1", ImplementationSHA256: HashBytes([]byte("minimizer implementation")),
		ParentRecordHash: parent.RecordHash, ParentFailureSignature: parent.Outcome.FailureSignature,
		OriginalCandidateSHA256: originalCandidate, FinalCandidateSHA256: finalCandidate,
		AttemptBudget: 16, Attempts: 4, OriginalForcedDecisions: 5, FinalForcedDecisions: 2,
		Accepted: []MinimizationReduction{{
			Kind: "schedule_suffix", BeforeSHA256: originalCandidate, AfterSHA256: finalCandidate,
			Removed: []MinimizationDecision{{Dimension: "runtime", Ordinal: 4, Identity: HashBytes([]byte("forced decision"))}},
		}},
		Predicate: MinimizationPredicate{
			FailureSignature: parent.Outcome.FailureSignature, Domain: "target", Reason: "nonzero_exit", Termination: "exit",
			ReplayMatch: true, ChoiceReplay: "not_present", SimulationReplay: "exact",
		},
	}

	minimized, encoded := finalizedManifest(t, minimizedInput)
	if minimized.RecordHash == parent.RecordHash {
		t.Fatal("record hash omitted minimization evidence")
	}
	if minimized.Outcome.FailureSignature != parent.Outcome.FailureSignature {
		t.Fatal("minimization evidence changed the normalized failure signature")
	}
	decoded, err := DecodeExecutionRecord(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Minimization == nil || decoded.Minimization.ParentRecordHash != parent.RecordHash || len(decoded.Minimization.Accepted) != 1 {
		t.Fatalf("decoded minimization = %#v", decoded.Minimization)
	}

	minimizedInput.Minimization.Attempts = 17
	if _, _, err := FinalizeExecutionRecord(minimizedInput); err == nil {
		t.Fatal("FinalizeExecutionRecord() accepted minimization attempts beyond its budget")
	}
}

func TestFinalizeManifestBindsIOProfileInventory(t *testing.T) {
	firstInput := manifestFixture()
	first, _ := finalizedManifest(t, firstInput)

	changedInput := firstInput
	changedInput.IOProfile.Inventory = `{"schema":"inventory/v2"}`
	changedInput.IOProfile.InventorySHA256 = HashBytes([]byte(changedInput.IOProfile.Inventory))
	changed, _ := finalizedManifest(t, changedInput)
	if first.RecordHash == changed.RecordHash {
		t.Fatal("record hash omitted I/O profile inventory")
	}

	invalid := firstInput
	invalid.IOProfile.InventorySHA256 = HashBytes([]byte("stale"))
	if _, _, err := FinalizeExecutionRecord(invalid); err == nil || !strings.Contains(err.Error(), "inventory hash") {
		t.Fatalf("FinalizeExecutionRecord(stale inventory hash) error = %v", err)
	}
}

func TestFinalizeManifestRejectsMissingIOProfile(t *testing.T) {
	manifest := manifestFixture()
	manifest.IOProfile = IOProfile{}
	manifest.Environment = []Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}}
	if _, _, err := FinalizeExecutionRecord(manifest); err == nil || !strings.Contains(err.Error(), "I/O profile identity is required") {
		t.Fatalf("FinalizeExecutionRecord(profileless manifest) error = %v", err)
	}
}

func TestFinalizeManifestBindsCompatibilityPacks(t *testing.T) {
	firstInput := manifestFixture()
	first, _ := finalizedManifest(t, firstInput)

	changedInput := manifestFixture()
	changedInput.Target.Compatibility[0].SHA256 = HashBytes([]byte("changed compatibility pack"))
	changed, _ := finalizedManifest(t, changedInput)
	if first.RecordHash == changed.RecordHash || first.Outcome.FailureSignature == changed.Outcome.FailureSignature {
		t.Fatal("record identities omitted the compatibility pack")
	}

	invalid := manifestFixture()
	invalid.Target.Compatibility[0].SHA256 = "sha256:invalid"
	if _, _, err := FinalizeExecutionRecord(invalid); err == nil || !strings.Contains(err.Error(), "compatibility") {
		t.Fatalf("FinalizeExecutionRecord(invalid compatibility pack) error = %v", err)
	}
}

func TestFinalizeManifestBindsSelectedAdapters(t *testing.T) {
	firstInput := manifestFixture()
	first, _ := finalizedManifest(t, firstInput)

	changedInput := manifestFixture()
	changedInput.Target.Adapters[0].Version = "v1.72.4"
	changed, _ := finalizedManifest(t, changedInput)
	if first.RecordHash == changed.RecordHash || first.Outcome.FailureSignature == changed.Outcome.FailureSignature {
		t.Fatal("record identities omitted the selected adapter")
	}

	invalid := manifestFixture()
	invalid.Target.Adapters = nil
	if _, _, err := FinalizeExecutionRecord(invalid); err == nil || !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("FinalizeExecutionRecord(null adapters) error = %v", err)
	}
}

func TestFinalizeManifestBindsCapabilityModeAndManifestSemantics(t *testing.T) {
	closure, _ := finalizedManifest(t, manifestFixture())
	linkedInput := manifestFixture()
	payload := []byte(`{"facts":[]}`)
	linkedInput.Target.CapabilityMode = "linked"
	linkedInput.Target.CapabilityManifest = &TargetCapabilityManifest{
		Schema: "gomad3.live-capability-manifest/v1", File: "target-capabilities.json", SHA256: HashBytes(payload),
		Bytes: Uint64String(len(payload)), Facts: 0, ProducerImplementationSHA256: HashBytes([]byte("producer")), CapabilityUniverseSHA256: HashBytes([]byte("universe")),
	}
	linkedInput.Files = append(linkedInput.Files, File{Path: "target-capabilities.json", Mode: "0600", Size: Uint64String(len(payload)), SHA256: HashBytes(payload)})
	sort.Slice(linkedInput.Files, func(i, j int) bool { return linkedInput.Files[i].Path < linkedInput.Files[j].Path })
	linked, _ := finalizedManifest(t, linkedInput)
	if closure.RecordHash == linked.RecordHash || closure.Outcome.FailureSignature == linked.Outcome.FailureSignature {
		t.Fatal("record identities omitted linked capability semantics")
	}

	relocated := linkedInput
	relocated.Target.CapabilityManifest = cloneTargetCapabilityManifest(linkedInput.Target.CapabilityManifest)
	relocated.Target.CapabilityManifest.File = "relocated-capabilities.json"
	if left, right := projectTarget(linkedInput.Target), projectTarget(relocated.Target); !reflect.DeepEqual(left, right) {
		t.Fatal("target identity projection included the manifest payload path")
	}
}

func TestValidateCurrentTargetCapabilityRequiresGuardedManifest(t *testing.T) {
	guarded := Target{CapabilityMode: "guarded"}
	if err := ValidateCurrentTargetCapability(guarded); err == nil || !strings.Contains(err.Error(), "guarded target capability manifest identity is incomplete") {
		t.Fatalf("ValidateCurrentTargetCapability(missing guarded manifest) error = %v", err)
	}
	guarded.CapabilityManifest = &TargetCapabilityManifest{
		Schema: "gomad3.live-capability-manifest/v1", File: "target-capabilities.json", SHA256: HashBytes([]byte("payload")),
		Bytes: 7, Facts: 1, ProducerImplementationSHA256: HashBytes([]byte("producer")), CapabilityUniverseSHA256: HashBytes([]byte("universe")),
	}
	if err := ValidateCurrentTargetCapability(guarded); err == nil || !strings.Contains(err.Error(), "guarded target capability manifest identity is incomplete") {
		t.Fatalf("ValidateCurrentTargetCapability(v1 guarded manifest) error = %v", err)
	}
	guarded.CapabilityManifest.Schema = "gomad3.live-capability-manifest/v2"
	if err := ValidateCurrentTargetCapability(guarded); err == nil || !strings.Contains(err.Error(), "guard implementation") {
		t.Fatalf("ValidateCurrentTargetCapability(missing guard identity) error = %v", err)
	}
	guarded.CapabilityManifest.GuardImplementationSHA256 = HashBytes([]byte("guard"))
	if err := ValidateCurrentTargetCapability(guarded); err != nil {
		t.Fatalf("ValidateCurrentTargetCapability(guarded manifest): %v", err)
	}
}

func TestIdentityProjectionsExcludeArtifactPaths(t *testing.T) {
	first := manifestFixture()
	second := manifestFixture()
	second.Target.File = "relocated-target"
	second.Streams.Stdout.File = "relocated-stdout"
	second.Streams.Stderr.File = "relocated-stderr"
	second.World.Initial.File = "relocated/initial"
	second.World.Transitions.File = "relocated/transitions"
	second.World.Final.File = "relocated/final"
	for index := range second.Files {
		second.Files[index].Path = "relocated/" + second.Files[index].Path
	}
	for name, projections := range map[string][2]any{
		"record":  {recordProjectionOf(first), recordProjectionOf(second)},
		"failure": {failureProjectionOf(first), failureProjectionOf(second)},
	} {
		left, err := canonicaljson.CanonicalJSON(projections[0])
		if err != nil {
			t.Fatal(err)
		}
		right, err := canonicaljson.CanonicalJSON(projections[1])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatalf("%s identity projection included artifact paths", name)
		}
	}
}

func TestDecodeManifestRejectsNoncanonicalAndChangedIdentity(t *testing.T) {
	manifest, encoded := finalizedManifest(t, manifestFixture())
	var noncanonical bytes.Buffer
	noncanonical.WriteByte(' ')
	noncanonical.Write(encoded)
	if _, err := DecodeExecutionRecord(noncanonical.Bytes()); err == nil {
		t.Fatal("DecodeExecutionRecord accepted noncanonical whitespace")
	}

	manifest.RecordHash = HashBytes([]byte("changed"))
	changed, err := canonicaljson.CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeExecutionRecord(changed); err == nil {
		t.Fatal("DecodeExecutionRecord accepted changed record hash")
	}
}

func TestNoneWorldUsesHashableExplicitPayloads(t *testing.T) {
	world, payloads := NoneWorld()
	if got, want := string(payloads.Initial), "null"; got != want {
		t.Fatalf("initial payload = %q, want %q", got, want)
	}
	if got, want := len(payloads.Transitions), 0; got != want {
		t.Fatalf("transition payload length = %d, want %d", got, want)
	}
	if got, want := string(payloads.Final), "null"; got != want {
		t.Fatalf("final payload = %q, want %q", got, want)
	}
	if world.Initial.Schema != "gomad3.world.snapshot/none" || world.Transitions.Schema != "gomad3.world.transitions/none" || world.Final.Schema != "gomad3.world.snapshot/none" {
		t.Fatalf("none World schemas = %#v", world)
	}
	if world.Initial.RawSHA256 != HashBytes(payloads.Initial) || world.Transitions.RawSHA256 != HashBytes(payloads.Transitions) || world.Final.RawSHA256 != HashBytes(payloads.Final) {
		t.Fatal("none World raw hashes do not match payload bytes")
	}
}

func TestFinalizeManifestRejectsReservedReplayEnvironment(t *testing.T) {
	for _, name := range []string{"GODEBUG", "LIBPATH", "SHLIB_PATH"} {
		manifest := manifestFixture()
		manifest.Environment = []Environment{{Name: "GOMADSEED", Value: "7"}, {Name: name, Value: "x"}, {Name: "TZ", Value: "UTC"}}
		sort.Slice(manifest.Environment, func(i, j int) bool { return manifest.Environment[i].Name < manifest.Environment[j].Name })
		if _, _, err := FinalizeExecutionRecord(manifest); err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Fatalf("FinalizeExecutionRecord() for %s error = %v", name, err)
		}
	}
}

func TestFinalizeManifestRejectsImpossibleUntruncatedStream(t *testing.T) {
	manifest := manifestFixture()
	manifest.Streams.Stdout.FullSHA256 = HashBytes([]byte("different complete stream"))
	if _, _, err := FinalizeExecutionRecord(manifest); err == nil || !strings.Contains(err.Error(), "untruncated") {
		t.Fatalf("FinalizeExecutionRecord() error = %v", err)
	}
}

func TestFinalizeManifestAcceptsReplayableSuccessfulRun(t *testing.T) {
	manifest := manifestFixture()
	manifest.ArtifactKind = ArtifactSuccess
	manifest.Outcome.Domain = "success"
	manifest.Outcome.Reason = "success"
	manifest.Outcome.ExitCode = uint64StringPointer(0)
	finalized, _, err := FinalizeExecutionRecord(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.ReplayMode != ReplayExact || finalized.Outcome.FailureSignature == "" {
		t.Fatalf("successful manifest = %#v", finalized)
	}
}

func finalizedManifest(t *testing.T, input ExecutionRecord) (ExecutionRecord, []byte) {
	t.Helper()
	manifest, encoded, err := FinalizeExecutionRecord(input)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, encoded
}

func manifestFixture() ExecutionRecord {
	world, payloads := NoneWorld()
	targetBytes := []byte("target bytes")
	stdoutBytes := []byte("stdout")
	stderrBytes := []byte("stderr")
	files := []File{
		{Path: "stderr", Mode: "0600", Size: Uint64String(len(stderrBytes)), SHA256: HashBytes(stderrBytes)},
		{Path: "stdout", Mode: "0600", Size: Uint64String(len(stdoutBytes)), SHA256: HashBytes(stdoutBytes)},
		{Path: "target", Mode: "0700", Size: Uint64String(len(targetBytes)), SHA256: HashBytes(targetBytes)},
		{Path: world.Final.File, Mode: "0600", Size: Uint64String(len(payloads.Final)), SHA256: HashBytes(payloads.Final)},
		{Path: world.Initial.File, Mode: "0600", Size: Uint64String(len(payloads.Initial)), SHA256: HashBytes(payloads.Initial)},
		{Path: world.Transitions.File, Mode: "0600", Size: Uint64String(len(payloads.Transitions)), SHA256: HashBytes(payloads.Transitions)},
	}
	return ExecutionRecord{
		SchemaVersion:    SchemaVersion,
		ArtifactKind:     ArtifactTargetFailure,
		CreatedAt:        "2026-08-10T12:00:00Z",
		CampaignID:       "batch-1",
		SelectionOrdinal: 3,
		Seed:             7,
		ReplayMode:       ReplayExact,
		Runner: Runner{
			RecordContract: RecordContract,
			RunnerBuild:    "runner-build",
			HostOS:         "darwin",
			HostArch:       "arm64",
		},
		Toolchain: Toolchain{
			GoVersion:    "go1.26.4",
			BuildKey:     "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e",
			TargetGOOS:   "darwin",
			TargetGOARCH: "arm64",
		},
		Target: Target{
			Kind:           "go-test",
			Source:         "./pkg",
			File:           "target",
			SHA256:         HashBytes(targetBytes),
			Size:           Uint64String(len(targetBytes)),
			Argv:           []string{"gomad3-target", "-test.run=TestGate"},
			BuildTags:      []string{"gomad_fixture"},
			Adapters:       []TargetAdapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:adapter"}},
			Compatibility:  []CompatibilityPack{{ID: "reflect2-go126", SHA256: HashBytes([]byte("compatibility pack"))}},
			BuildInfo:      BuildInfo{GoVersion: "go1.26.4", Path: "example.test/project/pkg.test"},
			CapabilityMode: "closure",
		},
		IOProfile: IOProfile{
			Name:                 "gomad3-deterministic/v1",
			ImplementationSHA256: HashBytes([]byte("implementation")),
			Inventory:            `{"schema":"inventory/v1"}`,
			InventorySHA256:      HashBytes([]byte(`{"schema":"inventory/v1"}`)),
		},
		Environment: []Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMAD3_IO_PROFILE", Value: "gomad3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}},
		Limits: Limits{
			ExecutionTimeoutNanos: Uint64String(30_000_000_000),
			OverallTimeoutNanos:   Uint64String(600_000_000_000),
			TerminateGraceNanos:   Uint64String(2_000_000_000),
			OutputBytes:           Uint64String(8 << 20),
			WorldTransitionBytes:  Uint64String(64 << 20),
		},
		World: world,
		Outcome: Outcome{
			Domain:      "target",
			Reason:      "nonzero_exit",
			Termination: "exit",
			ExitCode:    uint64StringPointer(1),
			Signal:      nil,
			Deadline:    nil,
			ReplayMatch: nil,
		},
		Streams: Streams{
			Stdout: Stream{File: "stdout", RetainedSHA256: HashBytes(stdoutBytes), FullSHA256: HashBytes(stdoutBytes), TotalBytes: Uint64String(len(stdoutBytes)), RetainedBytes: Uint64String(len(stdoutBytes)), DiscardedBytes: 0, Truncated: false},
			Stderr: Stream{File: "stderr", RetainedSHA256: HashBytes(stderrBytes), FullSHA256: HashBytes(stderrBytes), TotalBytes: Uint64String(len(stderrBytes)), RetainedBytes: Uint64String(len(stderrBytes)), DiscardedBytes: 0, Truncated: false},
		},
		Files: files,
		Host: Host{
			StartedAt:    "2026-08-10T12:00:00Z",
			FinishedAt:   "2026-08-10T12:00:01Z",
			ElapsedNanos: Uint64String(1_000_000_000),
		},
	}
}

func uint64StringPointer(value uint64) *Uint64String {
	converted := Uint64String(value)
	return &converted
}

func manifestWithSimulationProfile(manifest ExecutionRecord, plan, record []byte, failure SHA256) ExecutionRecord {
	manifest.SimulationProfile = &SimulationProfile{
		Name:             "gomad3-simulation-exploration/v1",
		ControllerSHA256: HashBytes([]byte("simulation controller")),
		ExecutionSHA256:  HashBytes([]byte("simulation execution")),
		CandidateSHA256:  HashBytes(plan),
		OutcomeSHA256:    HashBytes([]byte("simulation outcome")),
		FailureSHA256:    failure,
		Plan: SimulationPlan{
			Schema: "gomad3.simulation-exploration-plan/v1", File: "simulation/plan.json",
			SHA256: HashBytes(plan), Bytes: Uint64String(len(plan)),
		},
		Record: SimulationRecord{
			Schema: "gomad3.cluster-record/v7", File: "simulation/record.json",
			SHA256: HashBytes(record), Bytes: Uint64String(len(record)), Limit: 128 << 20,
		},
	}
	manifest.Files = append(manifest.Files,
		File{Path: manifest.SimulationProfile.Plan.File, Mode: "0600", Size: manifest.SimulationProfile.Plan.Bytes, SHA256: manifest.SimulationProfile.Plan.SHA256},
		File{Path: manifest.SimulationProfile.Record.File, Mode: "0600", Size: manifest.SimulationProfile.Record.Bytes, SHA256: manifest.SimulationProfile.Record.SHA256},
	)
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	return manifest
}

func TestCurrentRecordContractUsesExecutionVocabulary(t *testing.T) {
	if SchemaVersion != 1 || RecordContract != "gomad3.execution-record/v1" {
		t.Fatalf("record contract = schema %d %q", SchemaVersion, RecordContract)
	}
}

func TestFinalizeManifestRequiresTapeIdentityForCurrentCompleteChoiceTrace(t *testing.T) {
	manifest := manifestFixture()
	manifest.ChoiceProfile = &ChoiceProfile{
		Name: "gomad3-choice-trace/v2", ImplementationSHA256: HashBytes([]byte("choice implementation")),
		Trace: ChoiceTrace{
			Schema: "gomad3.choice-trace/v2", File: "choices.bin", SHA256: HashBytes(make([]byte, 96)),
			Bytes: 96, Records: 1, BranchingRecords: 1, TerminalState: "complete", Limit: 160,
		},
	}
	manifest.Limits.ChoiceTraceBytes = 160
	manifest.Environment = append(manifest.Environment, Environment{Name: "GOMAD3_CHOICE_PROFILE", Value: manifest.ChoiceProfile.Name})
	sort.Slice(manifest.Environment, func(i, j int) bool { return manifest.Environment[i].Name < manifest.Environment[j].Name })
	manifest.Files = append(manifest.Files, File{Path: "choices.bin", Mode: "0600", Size: 96, SHA256: manifest.ChoiceProfile.Trace.SHA256})
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	if _, _, err := FinalizeExecutionRecord(manifest); err == nil || !strings.Contains(err.Error(), "tape") {
		t.Fatalf("FinalizeExecutionRecord() error = %v", err)
	}
}

func TestFinalizeManifestRejectsLegacyRecordSchema(t *testing.T) {
	manifest := manifestFixture()
	manifest.SchemaVersion = 2
	manifest.Runner.RecordContract = "gomad3.run-record/v2"
	if _, _, err := FinalizeExecutionRecord(manifest); err == nil || !strings.Contains(err.Error(), "unsupported manifest schema") {
		t.Fatalf("FinalizeExecutionRecord(legacy schema) error = %v", err)
	}
}
