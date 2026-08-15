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

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
