package choicewire

import (
	"crypto/sha256"
	"slices"
	"strings"
	"testing"
)

func TestCanonicalDecisionIsIndependentOfPhysicalAlternativeOrder(t *testing.T) {
	first := sha256.Sum256([]byte("first"))
	second := sha256.Sum256([]byte("second"))
	third := sha256.Sum256([]byte("third"))

	left, err := CanonicalDecision(4, KindRunnable, 19, false, [][sha256.Size]byte{first, second, third}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	right, err := CanonicalDecision(4, KindRunnable, 19, false, [][sha256.Size]byte{third, first, second}, second, 0)
	if err != nil {
		t.Fatal(err)
	}
	if left != right || left.SelectedIdentity != second || left.Alternatives != 3 {
		t.Fatalf("canonical decisions differ: left=%#v right=%#v", left, right)
	}
}

func TestCanonicalDecisionRejectsInvalidAlternativeIdentities(t *testing.T) {
	identity := sha256.Sum256([]byte("identity"))
	missing := sha256.Sum256([]byte("missing"))
	zero := [sha256.Size]byte{}
	for _, test := range []struct {
		name         string
		alternatives [][sha256.Size]byte
		selected     [sha256.Size]byte
	}{
		{name: "empty", selected: identity},
		{name: "zero", alternatives: [][sha256.Size]byte{zero}, selected: zero},
		{name: "duplicate", alternatives: [][sha256.Size]byte{identity, identity}, selected: identity},
		{name: "missing selected", alternatives: [][sha256.Size]byte{identity}, selected: missing},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CanonicalDecision(0, KindRunnable, 0, true, test.alternatives, test.selected, 0); err == nil {
				t.Fatal("CanonicalDecision() accepted invalid identities")
			}
		})
	}
}

func TestProjectDecisionTapeCopiesAndBindsCompleteTrace(t *testing.T) {
	selected := sha256.Sum256([]byte("selected"))
	other := sha256.Sum256([]byte("other"))
	decision, err := CanonicalDecision(0, KindRunnable, 11, false, [][sha256.Size]byte{selected, other}, selected, 7)
	if err != nil {
		t.Fatal(err)
	}
	record := decision.Record()
	encoded, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	payload := append([]byte(nil), encoded[:]...)
	trace := Trace{
		Version: Version2,
		Bytes:   payload,
		SHA256:  sha256.Sum256(payload),
		Records: []Record{record},
		Summary: Summary{Records: 1, Branching: 1, Runnable: 1, Terminal: TerminalComplete},
	}
	identity := ExecutionIdentity{
		TargetSHA256:      sha256.Sum256([]byte("target")),
		ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS:              "darwin", GOARCH: "arm64",
		ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := ProjectDecisionTape(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	if tape.SourceTraceSHA256 != trace.SHA256 || len(tape.Decisions) != 1 || tape.SHA256 == ([sha256.Size]byte{}) || len(tape.Bytes) == 0 {
		t.Fatalf("tape = %#v", tape)
	}
	validated, err := ValidateDecisionTape(tape, identity)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(validated.Bytes, tape.Bytes) || validated.Decisions[0] != decision {
		t.Fatalf("validated tape = %#v", validated)
	}
	payload[0] ^= 0xff
	tape.Bytes[0] ^= 0xff
	if validated.Decisions[0] != decision {
		t.Fatal("validated tape aliases caller-owned input")
	}

	wrong := identity
	wrong.GOARCH = "amd64"
	if _, err := ValidateDecisionTape(validated, wrong); err == nil {
		t.Fatal("ValidateDecisionTape() accepted a different execution identity")
	}
}

func TestValidateDivergenceTerminalBindsExpectedTapeDecision(t *testing.T) {
	selected := sha256.Sum256([]byte("selected"))
	other := sha256.Sum256([]byte("other"))
	decision, err := CanonicalDecision(0, KindRunnable, 11, false, [][sha256.Size]byte{selected, other}, selected, 0)
	if err != nil {
		t.Fatal(err)
	}
	record, err := EncodeRecord(decision.Record())
	if err != nil {
		t.Fatal(err)
	}
	trace := Trace{
		Version: Version2, Bytes: record[:], SHA256: sha256.Sum256(record[:]), Records: []Record{decision.Record()},
		Summary: Summary{Records: 1, Branching: 1, Runnable: 1, Terminal: TerminalComplete},
	}
	identity := ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	tape, err := ProjectDecisionTape(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	observed := decision
	observed.SiteOffset++
	terminal := Terminal{
		State: TerminalDiverged, DivergenceReason: DivergenceSite, DivergentOrdinal: 0, TapeRecords: 1,
		ExpectedPresent: true, ObservedPresent: true, Expected: decision.Record(), Observed: observed.Record(),
	}
	if _, err := ValidateDivergenceTerminal(tape, ModeReplay, terminal); err != nil {
		t.Fatal(err)
	}
	terminal.TapeRecords++
	if _, err := ValidateDivergenceTerminal(tape, ModeReplay, terminal); err == nil {
		t.Fatal("ValidateDivergenceTerminal() accepted a different tape count")
	}
	terminal.TapeRecords--
	terminal.Expected.SiteOffset++
	if _, err := ValidateDivergenceTerminal(tape, ModeReplay, terminal); err == nil {
		t.Fatal("ValidateDivergenceTerminal() accepted expected metadata outside the tape")
	}
}

func TestValidateDivergenceTerminalAcceptsRankPrefix(t *testing.T) {
	identity := testExecutionIdentity()
	decision := testCanonicalDecision(t, 0, KindRunnable, 2, 0)
	tape, err := encodeTape(identity, sha256.Sum256([]byte("source trace")), []Decision{decision})
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := BuildRankPrefix(tape, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	expected := prefix.Decisions[0]
	observed := expected
	observed.RankOverride = false
	observed.SelectedIdentity = sha256.Sum256([]byte("resolved selected identity"))
	observed.SiteOffset++
	terminal := Terminal{
		State: TerminalDiverged, DivergenceReason: DivergenceSite, DivergentOrdinal: 0, TapeRecords: 1,
		ExpectedPresent: true, ObservedPresent: true, Expected: expected.Record(), Observed: observed.Record(),
	}
	if _, err := ValidateDivergenceTerminal(prefix, ModePrefix, terminal); err != nil {
		t.Fatal(err)
	}
}

func TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace(t *testing.T) {
	identity := ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	if _, err := ProjectDecisionTape(Trace{Version: Version1, Summary: Summary{Terminal: TerminalComplete}}, identity); !strings.Contains(errString(err), "unavailable") {
		t.Fatalf("legacy ProjectDecisionTape() error = %v", err)
	}
	if _, err := ProjectDecisionTape(Trace{Version: Version2, Summary: Summary{Terminal: TerminalOverflow}}, identity); err == nil {
		t.Fatal("ProjectDecisionTape() accepted overflow evidence")
	}
}

func TestBuildRankPrefixBindsCanonicalRankWithoutParentTraceIdentity(t *testing.T) {
	identity := testExecutionIdentity()
	first := testCanonicalDecision(t, 0, KindRunnable, 1, 0)
	branch := testCanonicalDecision(t, 1, KindSelectPoll, 3, 1)
	left, err := encodeTape(identity, sha256.Sum256([]byte("left parent trace")), []Decision{first, branch})
	if err != nil {
		t.Fatal(err)
	}
	right, err := encodeTape(identity, sha256.Sum256([]byte("right parent trace")), []Decision{first, branch})
	if err != nil {
		t.Fatal(err)
	}

	leftPrefix, err := BuildRankPrefix(left, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	rightPrefix, err := BuildRankPrefix(right, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if leftPrefix.SHA256 != rightPrefix.SHA256 || !slices.Equal(leftPrefix.Bytes, rightPrefix.Bytes) {
		t.Fatalf("rank prefixes retain parent trace identity: left=%x right=%x", leftPrefix.SHA256, rightPrefix.SHA256)
	}
	if len(leftPrefix.Decisions) != 2 || leftPrefix.Decisions[0] != first {
		t.Fatalf("rank prefix decisions = %#v", leftPrefix.Decisions)
	}
	override := leftPrefix.Decisions[1]
	if !override.RankOverride || override.Selected != 2 || override.SelectedIdentity != ([sha256.Size]byte{}) || override.AlternativeSetDigest != branch.AlternativeSetDigest {
		t.Fatalf("rank override = %#v", override)
	}
	validated, err := ValidatePrefixTape(leftPrefix, identity)
	if err != nil {
		t.Fatal(err)
	}
	if validated.SHA256 != leftPrefix.SHA256 {
		t.Fatalf("validated prefix = %#v", validated)
	}
	if _, err := ValidateDecisionTape(leftPrefix, identity); err == nil {
		t.Fatal("ValidateDecisionTape() accepted a rank override")
	}
}

func TestBuildRankPrefixRejectsInvalidOverrideTargets(t *testing.T) {
	identity := testExecutionIdentity()
	decision := testCanonicalDecision(t, 0, KindRunnable, 2, 0)
	tape, err := encodeTape(identity, sha256.Sum256([]byte("parent trace")), []Decision{decision})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name          string
		ordinal, rank uint64
	}{
		{name: "missing decision", ordinal: 1, rank: 1},
		{name: "selected rank", ordinal: 0, rank: 0},
		{name: "rank out of range", ordinal: 0, rank: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BuildRankPrefix(tape, test.ordinal, uint32(test.rank)); err == nil {
				t.Fatal("BuildRankPrefix() accepted an invalid override")
			}
		})
	}
}

func testExecutionIdentity() ExecutionIdentity {
	return ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
}

func testCanonicalDecision(t *testing.T, ordinal uint64, kind Kind, alternatives, selected uint32) Decision {
	t.Helper()
	identities := make([][sha256.Size]byte, alternatives)
	for index := range identities {
		identities[index] = sha256.Sum256([]byte{byte(ordinal), byte(index + 1)})
	}
	decision, err := CanonicalDecision(ordinal, kind, ordinal+1, false, identities, identities[selected], 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
