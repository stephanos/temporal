package process

import (
	"crypto/sha256"
	"errors"
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
		Choice: &ChoiceCapability{Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: choicewire.HeaderBytes + choicewire.RecordBytes},
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
			request.Choice = &ChoiceCapability{Profile: test.profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: test.limit}
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
	for _, choice := range []*ChoiceCapability{nil, {Profile: choicewire.Profile, ImplementationSHA256: testChoiceImplementationSHA256, Limit: MinimumChoiceTraceBytes}} {
		for _, name := range []string{choiceProfileEnvironmentName, choiceTraceFDEnvironmentName, choiceTerminalFDEnvironmentName, choiceTraceBytesEnvironmentName} {
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
	backing, err := newChoiceTraceBacking(limit)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})

	record, err := choicewire.EncodeRecord(choicewire.Record{Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, Alternatives: 2, Selected: 1})
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
	backing, err := newChoiceTraceBacking(limit)
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
	backing, err := newChoiceTraceBacking(limit)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backing.close(); err != nil {
			t.Error(err)
		}
	})
	record, err := choicewire.EncodeRecord(choicewire.Record{Kind: choicewire.KindRunnable, Flags: choicewire.FlagDecision, Alternatives: 2, Selected: 0})
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
