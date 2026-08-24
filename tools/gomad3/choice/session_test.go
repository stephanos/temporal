package choice

import (
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSessionCollectsCompleteTrace(t *testing.T) {
	limit, err := TraceBytes(2)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(SessionSpec{Limit: limit, Mode: ModeRecord})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})

	record, err := encodeRecord(testSessionDecisionRecord(0, 2, 1))
	if err != nil {
		t.Fatal(err)
	}
	files := session.Files()
	if _, err := files.Trace.WriteAt(record[:], traceHeaderBytes); err != nil {
		t.Fatal(err)
	}
	header := encodeTraceHeader(limit)
	if err := publishTraceHeader(header[:], traceHeaderBytes+traceRecordBytes, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Trace.WriteAt(header[:], 0); err != nil {
		t.Fatal(err)
	}
	completed := encodeTerminal(terminal{State: TerminalComplete, Records: 1, MappingBytes: traceHeaderBytes + traceRecordBytes, PayloadHash: sha256.Sum256(record[:])})
	if _, err := files.Terminal.Write(completed[:]); err != nil {
		t.Fatal(err)
	}
	if err := files.Terminal.Close(); err != nil {
		t.Fatal(err)
	}

	trace, err := session.Collect()
	if err != nil {
		t.Fatal(err)
	}
	if trace.Summary.Records != 1 || trace.Summary.Branching != 1 || string(trace.Bytes) != string(record[:]) {
		t.Fatalf("trace = %+v", trace)
	}
}

func TestSessionRejectsInvalidTerminalStates(t *testing.T) {
	limit, err := TraceBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(SessionSpec{Limit: limit, Mode: ModeRecord})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	header := encodeTraceHeader(limit)
	if _, err := session.Files().Trace.WriteAt(header[:], 0); err != nil {
		t.Fatal(err)
	}

	if _, err := session.collectFrame(nil); !errors.Is(err, ErrUnterminated) {
		t.Fatalf("unterminated error = %v", err)
	}
	overflow := encodeTerminal(terminal{State: TerminalOverflow, MappingBytes: traceHeaderBytes, PayloadHash: sha256.Sum256(nil)})
	if _, err := session.collectFrame(overflow[:]); !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	malformed := overflow
	malformed[len(malformed)-1]++
	if _, err := session.collectFrame(malformed[:]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("malformed error = %v", err)
	}
	if _, err := session.collectFrame([]byte{1}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("short terminal error = %v", err)
	}
}

func TestSessionReturnsValidatedOverflowTrace(t *testing.T) {
	limit, err := TraceBytes(1)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(SessionSpec{Limit: limit, Mode: ModeRecord})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	record, err := encodeRecord(testSessionDecisionRecord(0, 2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Files().Trace.WriteAt(record[:], traceHeaderBytes); err != nil {
		t.Fatal(err)
	}
	header := encodeTraceHeader(limit)
	if err := publishTraceHeader(header[:], limit, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Files().Trace.WriteAt(header[:], 0); err != nil {
		t.Fatal(err)
	}
	completed := encodeTerminal(terminal{State: TerminalOverflow, Records: 1, MappingBytes: limit, PayloadHash: sha256.Sum256(record[:])})

	trace, err := session.collectFrame(completed[:])
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	if trace.Summary.Records != 1 || trace.Summary.Terminal != TerminalOverflow || string(trace.Bytes) != string(record[:]) {
		t.Fatalf("overflow trace = %+v", trace)
	}
}

func TestSessionReplayPlanIsInheritedReadOnly(t *testing.T) {
	identity := ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
	plan, err := ProjectReplayPlan(Trace{Version: Version2, Bytes: []byte{}, SHA256: sha256.Sum256(nil), Records: []Record{}, Summary: Summary{Terminal: TerminalComplete}}, identity)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(SessionSpec{Limit: MinimumTraceBytes, Mode: ModeReplay, ReplayPlan: &plan})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	file := session.Files().ReplayPlan
	if file == nil {
		t.Fatal("choice tape backing is missing")
	}
	want := append([]byte(nil), plan.Bytes...)
	plan.Bytes[0] ^= 0xff
	actual := make([]byte, len(want))
	if _, err := file.ReadAt(actual, 0); err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(want) {
		t.Fatal("choice tape backing aliases caller-owned input")
	}
	if _, err := file.WriteAt([]byte{1}, 0); err == nil {
		t.Fatal("choice tape descriptor is writable")
	}
	if err := file.Truncate(int64(len(plan.Bytes) + 1)); err == nil || !errors.Is(err, os.ErrPermission) && !errors.Is(err, errors.ErrUnsupported) {
		if err == nil {
			t.Fatal("choice tape descriptor is resizable")
		}
	}
}

func testSessionDecisionRecord(ordinal uint64, alternatives, selected uint32) Record {
	return Record{
		Ordinal: ordinal, Kind: KindRunnable, Flags: FlagDecision, Alternatives: alternatives, Selected: selected,
		SelectedIdentity: sha256.Sum256([]byte("selected")), AlternativeSetDigest: sha256.Sum256([]byte("set")),
	}
}
