package process

import (
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
)

var testChoiceImplementationSHA256 = sha256.Sum256([]byte("choice implementation"))

func TestValidateRequestChoiceCapability(t *testing.T) {
	request := Request{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World:  WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
		Choice: &ChoiceCapability{Mode: choicewire.ModeRecord, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: choicewire.HeaderBytes + choicewire.RecordBytes},
	}
	if err := validateRequest(request); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		profile string
		limit   uint64
	}{
		{name: "missing profile", limit: choicewire.HeaderBytes + choicewire.RecordBytes},
		{name: "unknown profile", profile: "unknown", limit: choicewire.HeaderBytes + choicewire.RecordBytes},
		{name: "no record capacity", profile: choicewire.Profile, limit: choicewire.HeaderBytes},
		{name: "over maximum", profile: choicewire.Profile, limit: maximumChoiceTraceBytes + 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request.Choice = &ChoiceCapability{Mode: choicewire.ModeRecord, Profile: test.profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: test.limit}
			if err := validateRequest(request); err == nil {
				t.Fatal("validateRequest() succeeded")
			}
		})
	}
}

func TestValidateRequestRejectsChoiceEnvironmentInjection(t *testing.T) {
	base := Request{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	for _, choice := range []*ChoiceCapability{nil, {Mode: choicewire.ModeRecord, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes}} {
		for _, name := range []string{choiceProfileEnvironmentName, choiceModeEnvironmentName, choiceTraceFDEnvironmentName, choiceTerminalFDEnvironmentName, choiceTraceBytesEnvironmentName, choiceTapeFDEnvironmentName, choiceTapeBytesEnvironmentName} {
			request := base
			request.Choice = choice
			request.Env = []string{name + "=injected"}
			if err := validateRequest(request); err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("validateRequest() accepted %s with choice %#v: %v", name, choice, err)
			}
		}
	}
}

func TestChoiceTraceBackingValidatesCompleteTrace(t *testing.T) {
	limit := uint64(choicewire.HeaderBytes + 2*choicewire.RecordBytes)
	backing, err := newChoiceTraceBacking(limit, choicewire.ModeRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})

	record, err := choicewire.EncodeRecord(testDecisionRecord(0, 2, 1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = backing.file.WriteAt(record[:], choicewire.HeaderBytes)
	if err != nil {
		t.Fatal(err)
	}
	header := choicewire.EncodeHeader(limit)
	if err := choicewire.PublishHeader(header[:], choicewire.HeaderBytes+choicewire.RecordBytes, 1); err != nil {
		t.Fatal(err)
	}
	_, err = backing.file.WriteAt(header[:], 0)
	if err != nil {
		t.Fatal(err)
	}
	terminal := choicewire.EncodeTerminal(choicewire.Terminal{State: choicewire.TerminalComplete, Records: 1, MappingBytes: choicewire.HeaderBytes + choicewire.RecordBytes, PayloadHash: sha256.Sum256(record[:])})

	trace, err := backing.result(terminal[:])
	if err != nil {
		t.Fatal(err)
	}
	if trace.Summary.Records != 1 || trace.Summary.Branching != 1 || string(trace.Bytes) != string(record[:]) {
		t.Fatalf("trace = %+v", trace)
	}
}

func TestChoiceTraceBackingRejectsInvalidTerminalStates(t *testing.T) {
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	backing, err := newChoiceTraceBacking(limit, choicewire.ModeRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})
	header := choicewire.EncodeHeader(limit)
	_, err = backing.file.WriteAt(header[:], 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = backing.result(nil)
	if !errors.Is(err, ErrChoiceTraceUnterminated) {
		t.Fatalf("unterminated error = %v", err)
	}
	overflow := choicewire.EncodeTerminal(choicewire.Terminal{State: choicewire.TerminalOverflow, MappingBytes: choicewire.HeaderBytes, PayloadHash: sha256.Sum256(nil)})
	_, err = backing.result(overflow[:])
	if !errors.Is(err, ErrChoiceTraceOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	malformed := overflow
	malformed[len(malformed)-1]++
	_, err = backing.result(malformed[:])
	if !errors.Is(err, ErrChoiceTraceMalformed) {
		t.Fatalf("malformed error = %v", err)
	}

	_, err = backing.result([]byte{1})
	if !errors.Is(err, ErrChoiceTraceMalformed) {
		t.Fatalf("short terminal error = %v", err)
	}
}

func TestChoiceTraceBackingReturnsValidatedOverflowTrace(t *testing.T) {
	limit := uint64(choicewire.HeaderBytes + choicewire.RecordBytes)
	backing, err := newChoiceTraceBacking(limit, choicewire.ModeRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})
	record, err := choicewire.EncodeRecord(testDecisionRecord(0, 2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backing.file.WriteAt(record[:], choicewire.HeaderBytes); err != nil {
		t.Fatal(err)
	}
	header := choicewire.EncodeHeader(limit)
	if err := choicewire.PublishHeader(header[:], limit, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := backing.file.WriteAt(header[:], 0); err != nil {
		t.Fatal(err)
	}
	terminal := choicewire.EncodeTerminal(choicewire.Terminal{State: choicewire.TerminalOverflow, Records: 1, MappingBytes: limit, PayloadHash: sha256.Sum256(record[:])})

	trace, err := backing.result(terminal[:])
	if !errors.Is(err, ErrChoiceTraceOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	if trace.Summary.Records != 1 || trace.Summary.Terminal != choicewire.TerminalOverflow || string(trace.Bytes) != string(record[:]) {
		t.Fatalf("overflow trace = %+v", trace)
	}
}

func TestValidateRequestChoiceControllerModeMatrix(t *testing.T) {
	identity := choicewire.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: testChoiceImplementationSHA256,
	}
	decision, err := choicewire.CanonicalDecision(0, choicewire.KindRunnable, 1, false, [][sha256.Size]byte{sha256.Sum256([]byte("one"))}, sha256.Sum256([]byte("one")), 0)
	if err != nil {
		t.Fatal(err)
	}
	record, err := choicewire.EncodeRecord(decision.Record())
	if err != nil {
		t.Fatal(err)
	}
	payload := record[:]
	tape, err := choicewire.ProjectDecisionTape(choicewire.Trace{
		Version: choicewire.Version2, Bytes: payload, SHA256: sha256.Sum256(payload), Records: []choicewire.Record{decision.Record()},
		Summary: choicewire.Summary{Records: 1, Runnable: 1, Terminal: choicewire.TerminalComplete},
	}, identity)
	if err != nil {
		t.Fatal(err)
	}
	base := Request{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	for _, mode := range []choicewire.Mode{choicewire.ModeRecord, choicewire.ModeReplay, choicewire.ModePrefix} {
		request := base
		request.Choice = &ChoiceCapability{Mode: mode, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, ExecutionIdentity: identity, Limit: MinimumChoiceTraceBytes}
		if mode == choicewire.ModeReplay || mode == choicewire.ModePrefix {
			request.Choice.Tape = &tape
		}
		if err := validateRequest(request); err != nil {
			t.Fatalf("validateRequest(%d) error = %v", mode, err)
		}
	}
	for _, capability := range []*ChoiceCapability{
		{Mode: choicewire.ModeSeed, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes},
		{Mode: choicewire.ModeRecord, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes, Tape: &tape},
		{Mode: choicewire.ModeReplay, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes},
	} {
		request := base
		request.Choice = capability
		if err := validateRequest(request); err == nil {
			t.Fatalf("validateRequest() accepted %#v", capability)
		}
	}
	oversized := tape
	oversized.Bytes = make([]byte, MaximumChoiceTapeBytes+1)
	request := base
	request.Choice = &ChoiceCapability{
		Mode: choicewire.ModeReplay, Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256,
		ExecutionIdentity: identity, Limit: MinimumChoiceTraceBytes, Tape: &oversized,
	}
	if err := validateRequest(request); err == nil || !strings.Contains(err.Error(), "exceeds its bound") {
		t.Fatalf("validateRequest() oversized tape error = %v", err)
	}
}

func TestChoiceTapeBackingIsInheritedReadOnly(t *testing.T) {
	identity := choicewire.ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: testChoiceImplementationSHA256,
	}
	tape, err := choicewire.ProjectDecisionTape(choicewire.Trace{Version: choicewire.Version2, Bytes: []byte{}, SHA256: sha256.Sum256(nil), Records: []choicewire.Record{}, Summary: choicewire.Summary{Terminal: choicewire.TerminalComplete}}, identity)
	if err != nil {
		t.Fatal(err)
	}
	backing, err := newChoiceTraceBacking(MinimumChoiceTraceBytes, choicewire.ModeReplay, &tape)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})
	if backing.expected == nil {
		t.Fatal("choice tape backing is missing")
	}
	want := append([]byte(nil), tape.Bytes...)
	tape.Bytes[0] ^= 0xff
	actual := make([]byte, len(want))
	if _, err := backing.expected.ReadAt(actual, 0); err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(want) {
		t.Fatal("choice tape backing aliases caller-owned input")
	}
	if _, err := backing.expected.WriteAt([]byte{1}, 0); err == nil {
		t.Fatal("choice tape descriptor is writable")
	}
	if err := backing.expected.Truncate(int64(len(tape.Bytes) + 1)); err == nil || !errors.Is(err, os.ErrPermission) && !errors.Is(err, errors.ErrUnsupported) {
		if err == nil {
			t.Fatal("choice tape descriptor is resizable")
		}
	}
}

func testDecisionRecord(ordinal uint64, alternatives, selected uint32) choicewire.Record {
	return choicewire.Record{
		Ordinal: ordinal, Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision,
		Alternatives: alternatives, Selected: selected,
		SelectedIdentity: sha256.Sum256([]byte("selected")), AlternativeSetDigest: sha256.Sum256([]byte("set")),
	}
}
