package choice

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"slices"
	"strings"
	"testing"
)

func readinessWord(t *testing.T, ready uint32, flags Readiness) Readiness {
	t.Helper()
	word, err := NewReadiness(ready, flags)
	if err != nil {
		t.Fatal(err)
	}
	return word
}

func readinessTestIdentity() ExecutionIdentity {
	return ExecutionIdentity{
		TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
		GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("implementation")),
	}
}

// selectPollDecision is the poll decision at step alternatives-1 of the select
// at site: the alternatives are the case identities polled so far.
func selectPollDecision(t *testing.T, ordinal uint64, site uint64, alternatives int) Decision {
	t.Helper()
	identities := make([][sha256.Size]byte, alternatives)
	for index := range identities {
		identities[index] = sha256.Sum256([]byte{byte(site), byte(index)})
	}
	decision, err := CanonicalDecision(ordinal, KindSelectPoll, site, false, identities, identities[alternatives-1], uint32(alternatives-1))
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func runnableDecision(t *testing.T, ordinal uint64) Decision {
	t.Helper()
	first, second := sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))
	decision, err := CanonicalDecision(ordinal, KindRunnable, 0, true, [][sha256.Size]byte{first, second}, first, 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func selectResult(t *testing.T, ordinal, origin, site uint64, cases, polled uint32, readiness SelectReadiness) Record {
	t.Helper()
	word, err := readiness.Word()
	if err != nil {
		t.Fatal(err)
	}
	alternatives := cases
	if readiness.Default {
		alternatives++
	}
	return Record{Ordinal: ordinal, Kind: KindSelectResult, Flags: FlagObservation, SiteOffset: site, Alternatives: alternatives, Selected: 0, Data: polled, Readiness: word, Origin: origin}
}

func TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions(t *testing.T) {
	oneReadyClosed := SelectReadiness{Known: true, Ready: 1, ClosedChannel: true, NilChannel: true}
	// A three-case select at site 40 with one nil case polls two cases, so it
	// records one poll decision; a four-case select at site 50 records three
	// and blocks between its polls and its result; a select at site 60 polls
	// but never resumes. Readiness attaches by origin, not adjacency.
	records := []Record{
		runnableDecision(t, 0).Record(),
		selectPollDecision(t, 1, 40, 2).Record(),
		selectResult(t, 2, 1, 40, 3, 2, oneReadyClosed),
		selectPollDecision(t, 3, 50, 2).Record(),
		selectPollDecision(t, 4, 50, 3).Record(),
		selectPollDecision(t, 5, 50, 4).Record(),
		runnableDecision(t, 6).Record(),
		selectPollDecision(t, 7, 60, 2).Record(),
		selectResult(t, 8, 3, 50, 4, 4, SelectReadiness{Known: true, Ready: 0, Default: true, TimerChannel: true}),
	}
	trace, err := BuildTrace(records, TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	identity := readinessTestIdentity()
	plan, err := ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	zeroReadyDefaultTimer := SelectReadiness{Known: true, Default: true, TimerChannel: true}
	want := []SelectReadiness{{}, oneReadyClosed, zeroReadyDefaultTimer, zeroReadyDefaultTimer, zeroReadyDefaultTimer, {}, {}}
	if len(plan.Decisions) != len(want) || !slices.Equal(plan.Readiness, want) {
		t.Fatalf("projected readiness = %+v, want %+v", plan.Readiness, want)
	}

	validated, err := ValidateReplayPlan(plan, identity)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(validated.Readiness, want) || !slices.Equal(validated.Decisions, plan.Decisions) {
		t.Fatalf("validated readiness = %+v", validated.Readiness)
	}
	target := plan.Decisions[3]
	prefix, err := BuildRankPrefix(plan, 3, (target.Selected+1)%target.Alternatives)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(prefix.Readiness, want[:4]) {
		t.Fatalf("prefix readiness = %+v", prefix.Readiness)
	}
	shortened, err := plan.Prefix(2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(shortened.Readiness, want[:2]) {
		t.Fatalf("shortened prefix readiness = %+v", shortened.Readiness)
	}
	// The readiness is in the tape bytes: a plan without it is a different tape.
	bare, err := encodeTape(identity, trace.SHA256, plan.Decisions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bare.SHA256 == plan.SHA256 || !slices.Equal(bare.Decisions, plan.Decisions) {
		t.Fatal("readiness did not change the tape bytes or changed its decisions")
	}
}

func TestProjectReplayPlanRejectsResultsThatMisnameTheirPollDecisions(t *testing.T) {
	ready := SelectReadiness{Known: true, Ready: 2}
	for _, test := range []struct {
		name    string
		records []Record
	}{
		{name: "origin at a runnable decision", records: []Record{
			runnableDecision(t, 0).Record(), selectPollDecision(t, 1, 40, 2).Record(), selectResult(t, 2, 0, 40, 2, 2, ready),
		}},
		{name: "origin at another site", records: []Record{
			selectPollDecision(t, 0, 41, 2).Record(), selectResult(t, 1, 0, 40, 2, 2, ready),
		}},
		{name: "poll steps out of order", records: []Record{
			selectPollDecision(t, 0, 40, 3).Record(), selectPollDecision(t, 1, 40, 2).Record(), selectResult(t, 2, 0, 40, 3, 3, ready),
		}},
		{name: "fewer decisions than polled cases", records: []Record{
			selectPollDecision(t, 0, 40, 2).Record(), selectResult(t, 1, 0, 40, 3, 3, ready),
		}},
		{name: "two results name one decision", records: []Record{
			selectPollDecision(t, 0, 40, 2).Record(), selectResult(t, 1, 0, 40, 2, 2, ready), selectResult(t, 2, 0, 40, 2, 2, ready),
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			trace, err := BuildTrace(test.records, TerminalComplete)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ProjectReplayPlan(trace, readinessTestIdentity()); !errors.Is(err, ErrInvalidReplayPlan) {
				t.Fatalf("ProjectReplayPlan() error = %v", err)
			}
		})
	}
}

func TestProjectReplayPlanLeavesSingleCaseSelectsUnmatched(t *testing.T) {
	// A select with one polled case records no decision; its origin is the
	// ordinal the next record took, which here is another select's decision.
	records := []Record{
		selectPollDecision(t, 0, 40, 2).Record(),
		selectResult(t, 1, 0, 50, 1, 1, SelectReadiness{Known: true, Ready: 1}),
	}
	trace, err := BuildTrace(records, TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ProjectReplayPlan(trace, readinessTestIdentity())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(plan.Readiness, []SelectReadiness{{}}) {
		t.Fatalf("readiness = %+v", plan.Readiness)
	}
}

func TestDecodeTraceRejectsReadinessOnADecision(t *testing.T) {
	record := selectPollDecision(t, 0, 40, 2).Record()
	record.Readiness, _ = NewReadiness(1, 0)
	if _, err := BuildTrace([]Record{record}, TerminalComplete); !errors.Is(err, ErrMalformed) {
		t.Fatalf("BuildTrace() error = %v", err)
	}
}

func TestChoiceReadersRejectOtherWireVersions(t *testing.T) {
	record := runnableDecision(t, 0).Record()
	payload := encodeRecords(t, []Record{record})
	digest := sha256.Sum256(payload)
	metadata := TerminalMetadata{State: TerminalComplete, Limit: traceHeaderBytes + traceRecordBytes, Records: 1, SHA256: digest}
	for _, profile := range []string{SupersededProfile, "gomad3-choice-trace/v4"} {
		if _, err := DecodeStoredTrace(profile, payload, metadata); !errors.Is(err, ErrMalformed) || !strings.Contains(err.Error(), profile) {
			t.Fatalf("DecodeStoredTrace(%q) error = %v", profile, err)
		}
	}
	complete := encodeTerminal(terminal{State: TerminalComplete, Records: 1, MappingBytes: traceHeaderBytes + traceRecordBytes, PayloadHash: digest})
	identity := readinessTestIdentity()
	plan, err := encodeTape(identity, digest, []Decision{runnableDecision(t, 0)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []uint32{Version2, Version3 + 1} {
		frame := append([]byte(nil), complete[:]...)
		binary.BigEndian.PutUint32(frame[8:12], version)
		if _, err := DecodeTrace(payload, frame, traceHeaderBytes+traceRecordBytes); !errors.Is(err, ErrMalformed) {
			t.Fatalf("DecodeTrace(version %d) error = %v", version, err)
		}
		tape := ReplayPlan{Bytes: append([]byte(nil), plan.Bytes...)}
		binary.BigEndian.PutUint32(tape.Bytes[8:12], version)
		tape.SHA256 = sha256.Sum256(tape.Bytes)
		if _, err := ValidateReplayPlan(tape, identity); !errors.Is(err, ErrInvalidReplayPlan) {
			t.Fatalf("ValidateReplayPlan(version %d) error = %v", version, err)
		}
	}
}
