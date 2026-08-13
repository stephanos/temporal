package choicewire

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
)

func TestImplementationIdentityAndProjectionAreCanonical(t *testing.T) {
	target := sha256.Sum256([]byte("target"))
	records := []Record{
		{Ordinal: 0, Kind: KindRunnable, Flags: FlagDecision, SiteOffset: 24, Alternatives: 3, Selected: 2},
		{Ordinal: 1, Kind: KindRunnable, Flags: FlagDecision, SiteOffset: 24, Alternatives: 5, Selected: 4},
		{Ordinal: 2, Kind: KindSelectResult, Flags: FlagObservation | FlagSiteMissing, Alternatives: 2, Selected: 1, Data: 1},
	}
	payload := encodeRecords(t, records)
	projection, err := ProjectComplete(payload, CompleteMetadata{Limit: HeaderBytes + uint64(len(payload)), Records: 3, SHA256: sha256.Sum256(payload)}, target)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Summary.Branching != 2 || len(projection.Sites) != 2 || projection.Sites[0].Count != 2 || projection.Sites[0].MaximumAlternatives != 5 || projection.Sites[0].Fingerprint == "" || projection.Sites[1].Fingerprint != MissingSiteFingerprint {
		t.Fatalf("projection = %+v", projection)
	}
	if len(projection.Sites[0].Fingerprint) != sha256.Size*2 {
		t.Fatalf("site fingerprint = %q", projection.Sites[0].Fingerprint)
	}
}

func TestImplementationIdentityBindsToolchainBuildKey(t *testing.T) {
	first, err := ImplementationIdentity(strings.Repeat("a", sha256.Size*2))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ImplementationIdentity(strings.Repeat("b", sha256.Size*2))
	if err != nil {
		t.Fatal(err)
	}
	if first == ([sha256.Size]byte{}) || first == second {
		t.Fatalf("implementation identities = %x, %x", first, second)
	}
	for _, malformed := range []string{"", strings.Repeat("A", sha256.Size*2), strings.Repeat("g", sha256.Size*2)} {
		if _, err := ImplementationIdentity(malformed); err == nil {
			t.Fatalf("ImplementationIdentity(%q) succeeded", malformed)
		}
	}
}

func TestDecodeTraceValidatesRecordsAndSummarizes(t *testing.T) {
	records := []Record{
		{Ordinal: 0, Kind: KindRunnable, Flags: FlagDecision, SiteOffset: 24, Alternatives: 3, Selected: 2},
		{Ordinal: 1, Kind: KindSelectPoll, Flags: FlagDecision | FlagSiteMissing, Alternatives: 2, Selected: 0},
		{Ordinal: 2, Kind: KindSelectResult, Flags: FlagObservation, SiteOffset: 40, Alternatives: 4, Selected: 3, Data: 2},
	}
	payload := encodeRecords(t, records)
	terminal := EncodeTerminal(Terminal{State: TerminalComplete, Records: 3, MappingBytes: HeaderBytes + uint64(len(payload)), PayloadHash: sha256.Sum256(payload)})
	trace, err := DecodeTrace(payload, terminal[:], HeaderBytes+uint64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if trace.Summary.Records != 3 || trace.Summary.Branching != 2 || trace.Summary.Runnable != 1 || trace.Summary.SelectPoll != 1 || trace.Summary.SelectResult != 1 || trace.Summary.Terminal != TerminalComplete {
		t.Fatalf("summary = %+v", trace.Summary)
	}
	if trace.SHA256 != sha256.Sum256(payload) {
		t.Fatalf("digest = %x", trace.SHA256)
	}
}

func TestDecodeTraceRejectsMalformedEvidence(t *testing.T) {
	valid := Record{Ordinal: 0, Kind: KindRunnable, Flags: FlagDecision, Alternatives: 2, Selected: 1}
	payload := encodeRecords(t, []Record{valid})
	digest := sha256.Sum256(payload)
	complete := EncodeTerminal(Terminal{State: TerminalComplete, Records: 1, MappingBytes: HeaderBytes + RecordBytes, PayloadHash: digest})

	tests := []struct {
		name     string
		payload  []byte
		terminal []byte
		want     error
	}{
		{name: "unterminated", payload: payload, want: ErrUnterminated},
		{name: "partial record", payload: append([]byte(nil), payload[:len(payload)-1]...), terminal: complete[:], want: ErrMalformed},
		{name: "digest", payload: append([]byte(nil), payload...), terminal: mutateTerminalDigest(complete), want: ErrMalformed},
		{name: "ordinal gap", payload: encodeRecords(t, []Record{{Ordinal: 1, Kind: KindRunnable, Flags: FlagDecision, Alternatives: 2, Selected: 0}}), terminal: complete[:], want: ErrMalformed},
		{name: "selected bound", payload: mutateSelected(payload, 2), terminal: complete[:], want: ErrMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeTrace(test.payload, test.terminal, HeaderBytes+RecordBytes)
			if !errors.Is(err, test.want) {
				t.Fatalf("DecodeTrace() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestDecodeTraceReportsOverflowSeparately(t *testing.T) {
	payload := encodeRecords(t, []Record{{Ordinal: 0, Kind: KindRunnable, Flags: FlagDecision, Alternatives: 2, Selected: 0}})
	terminal := EncodeTerminal(Terminal{State: TerminalOverflow, Records: 1, MappingBytes: HeaderBytes + RecordBytes, PayloadHash: sha256.Sum256(payload)})
	trace, err := DecodeTrace(payload, terminal[:], HeaderBytes+RecordBytes)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("DecodeTrace() error = %v", err)
	}
	if trace.Summary.Records != 1 || trace.Summary.Terminal != TerminalOverflow || trace.SHA256 != sha256.Sum256(payload) || string(trace.Bytes) != string(payload) {
		t.Fatalf("overflow trace = %+v", trace)
	}
}

func TestProjectAcceptsValidatedOverflowWhileProjectCompleteRemainsStrict(t *testing.T) {
	payload := encodeRecords(t, []Record{{Ordinal: 0, Kind: KindRunnable, Flags: FlagDecision, Alternatives: 2, Selected: 0}})
	digest := sha256.Sum256(payload)
	target := sha256.Sum256([]byte("target"))
	projection, err := Project(payload, TerminalMetadata{State: TerminalOverflow, Limit: HeaderBytes + RecordBytes, Records: 1, SHA256: digest}, target)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Summary.Terminal != TerminalOverflow || projection.Summary.Records != 1 || projection.Summary.Branching != 1 {
		t.Fatalf("overflow projection = %+v", projection)
	}
	if _, err := ProjectComplete(payload, CompleteMetadata{Limit: HeaderBytes + RecordBytes, Records: 2, SHA256: digest}, target); !errors.Is(err, ErrMalformed) {
		t.Fatalf("ProjectComplete() error = %v", err)
	}
}

func encodeRecords(t *testing.T, records []Record) []byte {
	t.Helper()
	payload := make([]byte, 0, len(records)*RecordBytes)
	for _, record := range records {
		encoded, err := EncodeRecord(record)
		if err != nil {
			t.Fatal(err)
		}
		payload = append(payload, encoded[:]...)
	}
	return payload
}

func mutateTerminalDigest(terminal [TerminalFrameBytes]byte) []byte {
	terminal[32]++
	return terminal[:]
}

func mutateSelected(payload []byte, selected byte) []byte {
	result := append([]byte(nil), payload...)
	result[19] = selected
	return result
}
