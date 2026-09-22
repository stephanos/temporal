package campaign

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

// fakeBridge stands in for the bridge process: it answers each frame the way the Lean bridge
// does at the protocol level (sequence, set and profile echo, one candidate at a time, rejection
// of what is out of order) and hands out the candidates it was given. What a Run credits is the
// Lean campaign's to decide; the fake reads the Run's disposition, cleanup and Verdict the same
// way so the Go side's handling of each answer can be pinned.
type fakeBridge struct {
	t          testing.TB
	requests   chan request
	writer     io.Writer
	candidates []Candidate
	set        string
	profile    string
	seq        int
	handed     int
	outstand   *Candidate
	observed   []request
	rawReplies []string
}

func newFakeBridge(t testing.TB, candidates ...Candidate) (*Bridge, *fakeBridge) {
	t.Helper()
	toBridge, fromClient := io.Pipe()
	toClient, fromBridge := io.Pipe()
	fake := &fakeBridge{t: t, requests: make(chan request, 16), writer: fromBridge, candidates: candidates}
	go fake.serve(toBridge)
	t.Cleanup(func() {
		_ = fromClient.Close()
		_ = toClient.Close()
	})
	return New(fromClient, toClient, 1<<20), fake
}

func (f *fakeBridge) serve(input io.Reader) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		var frame request
		if err := json.Unmarshal(scanner.Bytes(), &frame); err != nil {
			f.write(`{"frame":"rejected","seq":0,"reason":"not a frame"}`)
			continue
		}
		f.requests <- frame
		if len(f.rawReplies) > 0 {
			raw := f.rawReplies[0]
			f.rawReplies = f.rawReplies[1:]
			f.write(raw)
			continue
		}
		f.write(f.answer(frame))
	}
}

func (f *fakeBridge) write(line string) {
	if _, err := io.WriteString(f.writer, line+"\n"); err != nil {
		f.t.Logf("fake bridge write: %v", err)
	}
}

func (f *fakeBridge) reject(seq int, reason string) string {
	encoded, _ := json.Marshal(map[string]any{"frame": "rejected", "seq": seq, "reason": reason})
	return string(encoded)
}

func (f *fakeBridge) answer(frame request) string {
	if frame.Seq != f.seq+1 {
		return f.reject(frame.Seq, "out-of-order frame")
	}
	header := map[string]any{"seq": frame.Seq, "set": f.set, "profile": f.profile}
	switch frame.Frame {
	case "initialize":
		f.set, f.profile = frame.Set, frame.Profile
		header["set"], header["profile"] = f.set, f.profile
		header["frame"] = "initialized"
		header["machine"] = "fake.machine"
		header["budget"] = "two"
		header["limits"] = map[string]int{"steps": 2, "actions": 2, "search": 64}
		targets := []string{}
		for _, candidate := range f.candidates {
			targets = append(targets, candidate.Covers...)
		}
		header["targets"] = targets
	case "next":
		if f.outstand != nil {
			return f.reject(frame.Seq, "a candidate is outstanding")
		}
		header["skipped"] = []Skipped{}
		if f.handed >= len(f.candidates) {
			header["frame"] = "exhausted"
			break
		}
		candidate := f.candidates[f.handed]
		f.handed++
		f.outstand = &candidate
		header["frame"] = "candidate"
		header["candidate"] = candidate.Identity
		header["target"] = candidate.Target
		header["covers"] = candidate.Covers
		header["caseId"] = candidate.CaseID
		header["fixture"] = candidate.Fixture
		header["case"] = json.RawMessage(candidate.Case)
	case "observe":
		if frame.Profile != f.profile {
			return f.reject(frame.Seq, "crossed profile")
		}
		if f.outstand == nil || frame.Candidate != f.outstand.Identity {
			return f.reject(frame.Seq, "crossed observe")
		}
		f.observed = append(f.observed, frame)
		observation, detail := "prepare-rejected", "preparation was rejected"
		if frame.PrepareRejected == nil {
			var run testpilotspb.Run
			if err := protojson.Unmarshal(frame.Run, &run); err != nil {
				return f.reject(frame.Seq, "run does not decode: "+err.Error())
			}
			observation, detail = readRun(f.outstand.CaseID, &run)
		}
		credited := []string{}
		status := "attempted"
		switch observation {
		case "satisfied":
			credited, status = f.outstand.Covers, "covered"
		case "violated":
			status = "violated"
		default:
		}
		statuses := []TargetStatus{}
		for _, key := range f.outstand.Covers {
			statuses = append(statuses, TargetStatus{Target: key, Status: status})
		}
		header["frame"] = "credited"
		header["candidate"] = f.outstand.Identity
		header["observation"] = observation
		header["detail"] = detail
		header["credited"] = credited
		header["statuses"] = statuses
		f.outstand = nil
	case "finish":
		header["frame"] = "finished"
		header["status"] = "stopped"
		if f.handed == len(f.candidates) && f.outstand == nil {
			header["status"] = "exhausted"
		}
		header["summary"] = Summary{Targets: 2, Selected: f.handed}
		header["counterexamples"] = []Counterexample{}
		header["ledger"] = []TargetStatus{}
	default:
		return f.reject(frame.Seq, "unknown frame")
	}
	f.seq = frame.Seq
	encoded, err := json.Marshal(header)
	if err != nil {
		f.t.Logf("fake bridge encode: %v", err)
	}
	return string(encoded)
}

// readRun mirrors the bridge's reading: satisfied on a completed Run with closed cleanup, violated
// on a closed Run the Monitor stopped or completed, inconclusive otherwise.
func readRun(caseID string, run *testpilotspb.Run) (observation, detail string) {
	if run.GetCaseId() != caseID {
		return "inconclusive", "run names another Case"
	}
	if run.GetCleanup().GetStatus() != testpilotspb.CLEANUP_STATUS_SUCCEEDED {
		return "inconclusive", "cleanup is not closed"
	}
	switch {
	case run.GetDisposition() == testpilotspb.RUN_DISPOSITION_COMPLETED && run.GetVerdict().GetStatus() == testpilotspb.VERDICT_STATUS_SATISFIED:
		return "satisfied", ""
	case run.GetVerdict().GetStatus() == testpilotspb.VERDICT_STATUS_VIOLATED &&
		(run.GetDisposition() == testpilotspb.RUN_DISPOSITION_COMPLETED || run.GetDisposition() == testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR):
		return "violated", ""
	}
	return "inconclusive", "not decisive"
}

func (f *fakeBridge) nextRequest(t testing.TB) request {
	t.Helper()
	select {
	case frame := <-f.requests:
		return frame
	case <-time.After(5 * time.Second):
		t.Fatal("the fake bridge received no frame")
		return request{}
	}
}

func (f *fakeBridge) requireNoRequest(t testing.TB) {
	t.Helper()
	select {
	case frame := <-f.requests:
		t.Fatalf("the bridge received a frame it should not have: %+v", frame)
	case <-time.After(50 * time.Millisecond):
	}
}

// A Case the fake hands out; the Go side decodes it and hands it to the binder.
func sampleCase(caseID string) json.RawMessage {
	source := &testpilotspb.Case{
		Version: &testpilotspb.FormatVersion{Major: 1},
		CaseId:  caseID,
		Program: &testpilotspb.Program{ProgramId: caseID + ".program"},
		Contract: &testpilotspb.Contract{
			ContractId: caseID + ".contract",
		},
	}
	encoded, err := protojson.Marshal(source)
	if err != nil {
		panic(err)
	}
	return encoded
}

const (
	firstIdentity  = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	secondIdentity = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

func sampleCandidates() []Candidate {
	return []Candidate{
		{Identity: firstIdentity, Target: "row:a", Covers: []string{"row:a", "result:x"}, CaseID: "temporal.case.set.1", Fixture: "set-1", Case: sampleCase("temporal.case.set.1")},
		{Identity: secondIdentity, Target: "row:b", Covers: []string{"row:b"}, CaseID: "temporal.case.set.2", Fixture: "set-2", Case: sampleCase("temporal.case.set.2")},
	}
}

// fakeBinder decides what binding one Case comes to.
type fakeBinder struct {
	bindErr  error
	run      *testpilotspb.Run
	verdict  *testpilotspb.Verdict
	runErr   error
	bound    int
	released int
	runs     int
	identity string
	// observedAtRelease is how many observe frames the fake bridge had received when Release
	// ran, so the order release-then-observe can be pinned.
	observedAtRelease int
	bridge            *fakeBridge
}

func (b *fakeBinder) Bind(_ context.Context, identity string, source *testpilotspb.Case) (Bound, error) {
	b.bound++
	b.identity = identity
	if b.bindErr != nil {
		return nil, b.bindErr
	}
	if b.run != nil && b.run.CaseId == "" {
		b.run.CaseId = source.GetCaseId()
	}
	return b, nil
}

func (b *fakeBinder) Run(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	b.runs++
	return b.run, b.verdict, b.runErr
}

func (b *fakeBinder) Release(context.Context) error {
	b.released++
	if b.bridge != nil {
		b.observedAtRelease = len(b.bridge.observed)
	}
	return nil
}

func closedRun(disposition testpilotspb.RunDisposition, cleanup testpilotspb.CleanupStatus, status testpilotspb.VerdictStatus) (*testpilotspb.Run, *testpilotspb.Verdict) {
	verdict := &testpilotspb.Verdict{Status: status}
	return &testpilotspb.Run{
		RunId: "run-1", Disposition: disposition,
		Cleanup: &testpilotspb.CleanupOutcome{Status: cleanup},
		Verdict: verdict,
	}, verdict
}

func initialized(t *testing.T, candidates ...Candidate) (*Bridge, *fakeBridge) {
	t.Helper()
	bridge, fake := newFakeBridge(t, candidates...)
	opened, err := bridge.Initialize(t.Context(), "set", "profile-a")
	require.NoError(t, err)
	require.Equal(t, "set", opened.Set)
	require.Equal(t, "profile-a", opened.Profile)
	require.Equal(t, Limits{Steps: 2, Actions: 2, Search: 64}, opened.Limits)
	fake.nextRequest(t)
	return bridge, fake
}

func TestNextRefusesWhileACandidateIsOutstandingBeforeWritingAnything(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	require.NotNil(t, next.Candidate)
	require.Equal(t, firstIdentity, next.Candidate.Identity)
	require.Equal(t, []string{"row:a", "result:x"}, next.Candidate.Covers)

	_, err = bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrCandidateOutstanding)
	fake.requireNoRequest(t)
	require.Equal(t, firstIdentity, bridge.Outstanding().Identity)
}

func TestObserveRefusesACrossedOrAbsentCandidateBeforeWritingAnything(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	detail := "no worker"
	_, err := bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	require.ErrorIs(t, err, ErrNoCandidate)
	fake.requireNoRequest(t)

	_, err = bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	_, err = bridge.Observe(t.Context(), secondIdentity, Result{PrepareRejected: &detail})
	require.ErrorIs(t, err, ErrCrossedCandidate)
	fake.requireNoRequest(t)
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{})
	require.Error(t, err)
	fake.requireNoRequest(t)
	require.NotNil(t, bridge.Outstanding())
}

func TestPreparationRejectionIsObservedWithoutARunAndReleasesNothing(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	binder := &fakeBinder{bindErr: &testpilot.PreparationError{Category: testpilot.PreparationUnsupported, Path: "program", Detail: "opcode outside the Profile"}}

	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	require.NoError(t, err)
	require.Equal(t, OutcomePrepareRejected, outcome.Kind)
	require.Equal(t, firstIdentity, outcome.Identity)
	require.Equal(t, "unsupported at program: opcode outside the Profile", outcome.Detail)
	require.Nil(t, outcome.Run)
	require.Zero(t, binder.runs)
	require.Zero(t, binder.released)
	require.Equal(t, "profile-a", binder.identity, "the Profile identity named at initialize is the one the Case is bound under")
	observed := fake.nextRequest(t)
	require.Equal(t, "observe", observed.Frame)
	require.NotNil(t, observed.PrepareRejected)
	require.Empty(t, observed.Run)
	require.Equal(t, "prepare-rejected", outcome.Credited.Observation)
	require.Empty(t, outcome.Credited.Credited)
	require.Nil(t, bridge.Outstanding())
}

func TestABindingFailureThatIsNotTheCasesLeavesTheCandidateOutstanding(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	binder := &fakeBinder{bindErr: errors.New("open SDK client: connection refused")}

	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	require.ErrorContains(t, err, "connection refused")
	require.Equal(t, OutcomeBindFailed, outcome.Kind)
	fake.requireNoRequest(t)
	require.Equal(t, firstIdentity, bridge.Outstanding().Identity)
}

// A closed Run that comes back beside an error -- a recorder or Monitor close failure after the
// Verdict was fixed -- is the authoritative record: it is observed, and the error travels beside
// the outcome.
func TestAClosedRunReturnedBesideAnErrorIsStillObserved(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	run, verdict := closedRun(testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_VIOLATED)
	binder := &fakeBinder{run: run, verdict: verdict, runErr: errors.New("recorder: closure capacity")}

	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, outcome.Kind)
	require.ErrorContains(t, outcome.RunError, "closure capacity")
	require.Equal(t, 1, binder.released)
	observed := fake.nextRequest(t)
	require.Equal(t, "observe", observed.Frame)
	var sent testpilotspb.Run
	require.NoError(t, protojson.Unmarshal(observed.Run, &sent))
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, sent.GetVerdict().GetStatus())
	require.Equal(t, "violated", outcome.Credited.Observation)
	require.Nil(t, bridge.Outstanding())
}

// A Run that comes back without an observed cleanup is not closed: it is reported as such, the
// bridge is not told, and the candidate stays outstanding.
func TestARunWithoutAnObservedCleanupIsNotObserved(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	run := &testpilotspb.Run{RunId: "run-1", Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Verdict: &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}}
	binder := &fakeBinder{run: run, verdict: run.Verdict}

	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	require.ErrorContains(t, err, "without an observed cleanup")
	require.Equal(t, OutcomeRunFailed, outcome.Kind)
	require.NotNil(t, outcome.Run, "the Run the facade returned travels with the outcome")
	require.Equal(t, 1, binder.released)
	fake.requireNoRequest(t)
	require.Equal(t, firstIdentity, bridge.Outstanding().Identity)
}

func TestARunThatCouldNotExecuteReleasesAndLeavesTheCandidateOutstanding(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	binder := &fakeBinder{runErr: errors.New("driver is unavailable")}

	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	require.ErrorContains(t, err, "driver is unavailable")
	require.Equal(t, OutcomeRunFailed, outcome.Kind)
	require.Equal(t, 1, binder.runs)
	require.Equal(t, 1, binder.released)
	fake.requireNoRequest(t)
	require.Equal(t, firstIdentity, bridge.Outstanding().Identity)
}

func TestADecisiveRunIsObservedAfterCleanupAndCreditedAlongItsPath(t *testing.T) {
	for _, probe := range []struct {
		name        string
		disposition testpilotspb.RunDisposition
		cleanup     testpilotspb.CleanupStatus
		verdict     testpilotspb.VerdictStatus
		observation string
		credited    []string
	}{
		{"satisfied", testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_SATISFIED, "satisfied", []string{"row:a", "result:x"}},
		{"violated", testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_VIOLATED, "violated", []string{}},
		{"cleanup failed", testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.CLEANUP_STATUS_FAILED, testpilotspb.VERDICT_STATUS_SATISFIED, "inconclusive", []string{}},
		{"inconclusive", testpilotspb.RUN_DISPOSITION_INCOMPLETE, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, "inconclusive", []string{}},
	} {
		t.Run(probe.name, func(t *testing.T) {
			bridge, fake := initialized(t, sampleCandidates()...)
			next, err := bridge.Next(t.Context())
			require.NoError(t, err)
			fake.nextRequest(t)
			run, verdict := closedRun(probe.disposition, probe.cleanup, probe.verdict)
			binder := &fakeBinder{run: run, verdict: verdict, bridge: fake}

			outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
			require.NoError(t, err)
			require.Equal(t, OutcomeCompleted, outcome.Kind)
			require.Equal(t, 1, binder.runs)
			require.Equal(t, 1, binder.released)
			require.Zero(t, binder.observedAtRelease, "the candidate is released before it is observed")
			observed := fake.nextRequest(t)
			require.Equal(t, "observe", observed.Frame)
			require.Equal(t, firstIdentity, observed.Candidate)
			require.Nil(t, observed.PrepareRejected)
			var sent testpilotspb.Run
			require.NoError(t, protojson.Unmarshal(observed.Run, &sent))
			require.Equal(t, "temporal.case.set.1", sent.GetCaseId())
			require.Equal(t, probe.cleanup, sent.GetCleanup().GetStatus(), "the observed cleanup travels with the Run")
			require.Equal(t, probe.verdict, sent.GetVerdict().GetStatus())
			require.Equal(t, probe.observation, outcome.Credited.Observation)
			require.Equal(t, probe.credited, outcome.Credited.Credited)
			require.Nil(t, bridge.Outstanding())
		})
	}
}

func TestExactlyOneCandidateFlowsThroughTheWholeCampaign(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	run, verdict := closedRun(testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_SATISFIED)
	binder := &fakeBinder{run: run, verdict: verdict}
	var identities []string
	for {
		next, err := bridge.Next(t.Context())
		require.NoError(t, err)
		fake.nextRequest(t)
		if next.Exhausted {
			break
		}
		require.NotNil(t, next.Candidate)
		identities = append(identities, next.Candidate.Identity)
		binder.run = &testpilotspb.Run{RunId: "run", Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}, Verdict: verdict}
		outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
		require.NoError(t, err)
		fake.nextRequest(t)
		require.Equal(t, OutcomeCompleted, outcome.Kind)
		require.Equal(t, "satisfied", outcome.Credited.Observation)
	}
	require.Equal(t, []string{firstIdentity, secondIdentity}, identities)
	require.Equal(t, 2, binder.runs)
	require.Equal(t, 2, binder.released)
	finished, err := bridge.Finish(t.Context(), "")
	require.NoError(t, err)
	fake.nextRequest(t)
	require.Equal(t, "exhausted", finished.Status)
	require.Equal(t, 2, finished.Summary.Selected)
	_, err = bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrFinished)
}

func TestFramesBeforeInitializeAreRefused(t *testing.T) {
	bridge, fake := newFakeBridge(t)
	_, err := bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrNotInitialized)
	_, err = bridge.Finish(t.Context(), "")
	require.ErrorIs(t, err, ErrNotInitialized)
	fake.requireNoRequest(t)
}

func TestARejectedFrameLeavesTheSequenceAndTheCandidateWhereTheyWere(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	fake.rawReplies = append(fake.rawReplies, `{"frame":"rejected","seq":3,"reason":"run names no runId"}`)
	garbage := json.RawMessage(`{}`)
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{Run: garbage})
	var rejected *RejectedError
	require.ErrorAs(t, err, &rejected)
	require.Equal(t, 3, rejected.Seq)
	require.Contains(t, rejected.Reason, "runId")
	fake.nextRequest(t)
	require.Equal(t, firstIdentity, bridge.Outstanding().Identity, "a rejected observe leaves the candidate outstanding")
	require.Equal(t, next.Candidate.Identity, bridge.Outstanding().Identity)
	// The next frame carries the same sequence number the rejected one did.
	detail := "no worker"
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	require.NoError(t, err)
	require.Equal(t, 3, fake.nextRequest(t).Seq)
}

func TestAReplyThatDoesNotMatchTheFrameIsAProtocolErrorAndBreaksTheBridge(t *testing.T) {
	for name, raw := range map[string]string{
		"sequence":  `{"frame":"exhausted","seq":9,"set":"set","profile":"profile-a","skipped":[]}`,
		"set":       `{"frame":"exhausted","seq":2,"set":"other","profile":"profile-a","skipped":[]}`,
		"profile":   `{"frame":"exhausted","seq":2,"set":"set","profile":"other","skipped":[]}`,
		"kind":      `{"frame":"finished","seq":2,"set":"set","profile":"profile-a"}`,
		"rejection": `{"frame":"rejected","seq":9,"reason":"stale"}`,
		"candidate": `{"frame":"candidate","seq":2,"set":"set","profile":"profile-a","candidate":"","caseId":"c","case":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			bridge, fake := initialized(t)
			fake.rawReplies = append(fake.rawReplies, raw)
			_, err := bridge.Next(t.Context())
			var protocol *ProtocolError
			require.ErrorAs(t, err, &protocol)
			fake.nextRequest(t)
			require.Nil(t, bridge.Outstanding())
			_, err = bridge.Next(t.Context())
			require.ErrorIs(t, err, ErrBroken)
			require.ErrorAs(t, err, &protocol, "the break carries the mismatch that caused it")
			fake.requireNoRequest(t)
		})
	}
}

func TestACreditedReplyForAnotherCandidateBreaksTheBridge(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	_, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	fake.rawReplies = append(fake.rawReplies, `{"frame":"credited","seq":3,"set":"set","profile":"profile-a","candidate":"`+secondIdentity+`","observation":"satisfied","detail":"","credited":[],"statuses":[]}`)
	detail := "no worker"
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	var protocol *ProtocolError
	require.ErrorAs(t, err, &protocol)
	require.ErrorIs(t, bridge.Broken(), ErrBroken)
	fake.nextRequest(t)
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	require.ErrorIs(t, err, ErrBroken)
	_, err = bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrBroken)
	fake.requireNoRequest(t)
}

func TestAnOversizedFrameInEitherDirectionIsRefused(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	huge := json.RawMessage(`{"runId":"` + strings.Repeat("x", 1<<20) + `"}`)
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{Run: huge})
	require.ErrorIs(t, err, ErrFrameTooLarge)
	fake.requireNoRequest(t)
	require.Equal(t, next.Candidate.Identity, bridge.Outstanding().Identity)

	fake.rawReplies = append(fake.rawReplies, `{"frame":"credited","seq":3,"set":"set","profile":"profile-a","candidate":"`+firstIdentity+`","detail":"`+strings.Repeat("y", 1<<20)+`"}`)
	detail := "no worker"
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	require.ErrorIs(t, err, ErrFrameTooLarge)
	fake.nextRequest(t)
	// The stream is out of step after an oversized reply: the bridge is broken, and every later
	// call says so without writing anything.
	require.ErrorIs(t, bridge.Broken(), ErrBroken)
	_, err = bridge.Observe(t.Context(), firstIdentity, Result{PrepareRejected: &detail})
	require.ErrorIs(t, err, ErrBroken)
	_, err = bridge.Next(t.Context())
	require.ErrorIs(t, err, ErrBroken)
	_, err = bridge.Finish(t.Context(), "")
	require.ErrorIs(t, err, ErrBroken)
	fake.requireNoRequest(t)
}

func TestACandidateWhoseCaseNamesAnotherCaseIsNotBound(t *testing.T) {
	bridge, fake := initialized(t, Candidate{
		Identity: firstIdentity, Target: "row:a", Covers: []string{"row:a"},
		CaseID: "temporal.case.set.1", Fixture: "set-1", Case: sampleCase("temporal.case.set.other"),
	})
	next, err := bridge.Next(t.Context())
	require.NoError(t, err)
	fake.nextRequest(t)
	binder := &fakeBinder{}
	outcome, err := RunCandidate(t.Context(), bridge, binder, next.Candidate)
	var protocol *ProtocolError
	require.ErrorAs(t, err, &protocol)
	require.Equal(t, OutcomeBindFailed, outcome.Kind)
	require.Zero(t, binder.bound)
	fake.requireNoRequest(t)
}

func TestRunCandidateRefusesACandidateThatIsNotOutstanding(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	candidates := sampleCandidates()
	_, err := RunCandidate(t.Context(), bridge, &fakeBinder{}, &candidates[0])
	require.ErrorIs(t, err, ErrCrossedCandidate)
	fake.requireNoRequest(t)
}

// A spawned bridge that neither reads nor answers is abandoned at the deadline and killed on
// Close, so a coordinator giving each call its own deadline is never stuck behind a search.
func TestCloseKillsABridgeThatNeverAnswers(t *testing.T) {
	bridge, err := Start(t.Context(), Options{Executable: "sh", Args: []string{"-c", "exec sleep 60"}})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err = bridge.Initialize(ctx, "set", "profile")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, bridge.Broken(), ErrBroken)
	closed := make(chan error, 1)
	go func() { closed <- bridge.Close() }()
	select {
	case err := <-closed:
		require.Error(t, err, "a killed bridge reports its exit status")
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return for a bridge that never answers")
	}
}

// A finished bridge is given its EOF and exits on its own; Close waits for it.
func TestCloseWaitsForAFinishedBridge(t *testing.T) {
	bridge, err := Start(t.Context(), Options{Executable: "sh", Args: []string{"-c", `read line; printf '%s\n' '{"frame":"initialized","seq":1,"set":"set","profile":"p","targets":[]}'; read line; printf '%s\n' '{"frame":"finished","seq":2,"set":"set","profile":"p","status":"stopped","summary":{},"counterexamples":[],"ledger":[]}'; cat >/dev/null`}})
	require.NoError(t, err)
	_, err = bridge.Initialize(t.Context(), "set", "p")
	require.NoError(t, err)
	finished, err := bridge.Finish(t.Context(), "stopped")
	require.NoError(t, err)
	require.Equal(t, "stopped", finished.Status)
	require.NoError(t, bridge.Close())
}

func TestAContextThatEndsWhileWaitingOnTheBridgeReturns(t *testing.T) {
	toBridge, fromClient := io.Pipe()
	toClient, _ := io.Pipe()
	t.Cleanup(func() { _ = fromClient.Close(); _ = toClient.Close() })
	go func() { _, _ = io.Copy(io.Discard, toBridge) }()
	bridge := New(fromClient, toClient, 0)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := bridge.Initialize(ctx, "set", "profile")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// A reader is still waiting on the stream, so the bridge is broken rather than read twice.
	_, err = bridge.Initialize(t.Context(), "set", "profile")
	require.ErrorIs(t, err, ErrBroken)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
