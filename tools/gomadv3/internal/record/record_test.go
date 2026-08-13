package record

import (
	"bytes"
	"crypto/sha256"
	"sort"
	"strings"
	"testing"
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
	if got, want := DomainHash("gomadv3-run-record-v1", []byte("payload")), SHA256("sha256:087406f9758d2fbb56f25c4a24ef6fbc9986ba6108814b05105ae598447940a5"); got != want {
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

	decoded, err := DecodeManifest(firstBytes)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.RecordHash != first.RecordHash || decoded.Outcome.FailureSignature != first.Outcome.FailureSignature {
		t.Fatal("decoded manifest identity changed")
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
	if _, _, err := FinalizeManifest(invalid); err == nil || !strings.Contains(err.Error(), "inventory hash") {
		t.Fatalf("FinalizeManifest(stale inventory hash) error = %v", err)
	}
}

func TestFinalizeManifestRejectsProfilelessSchemaV2(t *testing.T) {
	manifest := manifestFixture()
	manifest.IOProfile = IOProfile{}
	manifest.Environment = []Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}}
	if _, _, err := FinalizeManifest(manifest); err == nil || !strings.Contains(err.Error(), "I/O profile identity is required") {
		t.Fatalf("FinalizeManifest(profileless schema v2) error = %v", err)
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
	if _, _, err := FinalizeManifest(invalid); err == nil || !strings.Contains(err.Error(), "compatibility") {
		t.Fatalf("FinalizeManifest(invalid compatibility pack) error = %v", err)
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
	if _, _, err := FinalizeManifest(invalid); err == nil || !strings.Contains(err.Error(), "adapter") {
		t.Fatalf("FinalizeManifest(null adapters) error = %v", err)
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
		left, err := CanonicalJSON(projections[0])
		if err != nil {
			t.Fatal(err)
		}
		right, err := CanonicalJSON(projections[1])
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
	if _, err := DecodeManifest(noncanonical.Bytes()); err == nil {
		t.Fatal("DecodeManifest accepted noncanonical whitespace")
	}

	manifest.RecordHash = HashBytes([]byte("changed"))
	changed, err := CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeManifest(changed); err == nil {
		t.Fatal("DecodeManifest accepted changed record hash")
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
	if world.Initial.Schema != "gomadv3.world.snapshot/none" || world.Transitions.Schema != "gomadv3.world.transitions/none" || world.Final.Schema != "gomadv3.world.snapshot/none" {
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
		if _, _, err := FinalizeManifest(manifest); err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Fatalf("FinalizeManifest() for %s error = %v", name, err)
		}
	}
}

func TestFinalizeManifestRejectsImpossibleUntruncatedStream(t *testing.T) {
	manifest := manifestFixture()
	manifest.Streams.Stdout.FullSHA256 = HashBytes([]byte("different complete stream"))
	if _, _, err := FinalizeManifest(manifest); err == nil || !strings.Contains(err.Error(), "untruncated") {
		t.Fatalf("FinalizeManifest() error = %v", err)
	}
}

func TestFinalizeManifestAcceptsReplayableSuccessfulRun(t *testing.T) {
	manifest := manifestFixture()
	manifest.ArtifactKind = ArtifactSuccess
	manifest.Outcome.Domain = "success"
	manifest.Outcome.Reason = "success"
	manifest.Outcome.ExitCode = uint64StringPointer(0)
	finalized, _, err := FinalizeManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.ReplayMode != ReplayExact || finalized.Outcome.FailureSignature == "" {
		t.Fatalf("successful manifest = %#v", finalized)
	}
}

func finalizedManifest(t *testing.T, input Manifest) (Manifest, []byte) {
	t.Helper()
	manifest, encoded, err := FinalizeManifest(input)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, encoded
}

func manifestFixture() Manifest {
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
	return Manifest{
		SchemaVersion:    SchemaVersion,
		ArtifactKind:     ArtifactTargetFailure,
		CreatedAt:        "2026-08-10T12:00:00Z",
		BatchID:          "batch-1",
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
			Kind:          "go-test",
			Source:        "./pkg",
			File:          "target",
			SHA256:        HashBytes(targetBytes),
			Size:          Uint64String(len(targetBytes)),
			Argv:          []string{"gomadv3-target", "-test.run=TestGate"},
			BuildTags:     []string{"gomad_fixture"},
			Adapters:      []TargetAdapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:adapter"}},
			Compatibility: []CompatibilityPack{{ID: "reflect2-go126", SHA256: HashBytes([]byte("compatibility pack"))}},
			BuildInfo:     BuildInfo{GoVersion: "go1.26.4", Path: "example.test/project/pkg.test"},
		},
		IOProfile: IOProfile{
			Name:                 "gomadv3-deterministic/v1",
			ImplementationSHA256: HashBytes([]byte("implementation")),
			Inventory:            `{"schema":"inventory/v1"}`,
			InventorySHA256:      HashBytes([]byte(`{"schema":"inventory/v1"}`)),
		},
		Environment: []Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: "gomadv3-deterministic/v1"}, {Name: "TZ", Value: "UTC"}},
		Limits: Limits{
			RunTimeoutNanos:      Uint64String(30_000_000_000),
			OverallTimeoutNanos:  Uint64String(600_000_000_000),
			TerminateGraceNanos:  Uint64String(2_000_000_000),
			OutputBytes:          Uint64String(8 << 20),
			WorldTransitionBytes: Uint64String(64 << 20),
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
