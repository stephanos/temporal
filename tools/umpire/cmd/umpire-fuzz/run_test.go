package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	firstIdentity  = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	secondIdentity = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

// scriptedBridge answers frames the way the Lean bridge does at the protocol level, over two
// candidates whose credit it decides from the Run's disposition, cleanup and Verdict, and reports
// the ledger the command must copy rather than infer.
type scriptedBridge struct {
	t          testing.TB
	writer     io.Writer
	candidates []campaign.Candidate
	handed     int
	outstand   *campaign.Candidate
	set        string
	profile    string
	seq        int
	statuses   map[string]string
	violated   []campaign.Counterexample
	toolingOn  int
	// rejectFinish answers finish with a rejection, so the summary is never read.
	rejectFinish bool
	// proposalPath names each counterexample's proposal; empty names `<set>-<digest>.lean`.
	proposalPath string
}

// proposalFor is the scripted proposal of one violated candidate: bytes naming the candidate, so
// two candidates never share a digest, and the digest the bridge would seal them with.
func proposalFor(identity, path string) campaign.Counterexample {
	source := "-- proposal for " + identity + "\n"
	digest := sha256.Sum256([]byte(source))
	encoded := hex.EncodeToString(digest[:])
	if path == "" {
		path = "nexusCallerExploration-" + strings.TrimPrefix(identity, "sha256:") + ".lean"
	}
	return campaign.Counterexample{PromotionSourceSHA256: &encoded, PromotionSourcePath: path, PromotionSource: source}
}

func sampleCase(caseID string) json.RawMessage {
	encoded, err := protojson.Marshal(&testpilotspb.Case{
		Version:  &testpilotspb.FormatVersion{Major: 1},
		CaseId:   caseID,
		Program:  &testpilotspb.Program{ProgramId: caseID + ".program"},
		Contract: &testpilotspb.Contract{ContractId: caseID + ".contract"},
	})
	if err != nil {
		panic(err)
	}
	// The scripted bridge hands the Case out through json.Marshal, which compacts a RawMessage;
	// protojson's spacing is not stable, so the sample is compacted here to count the same bytes.
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, encoded); err != nil {
		panic(err)
	}
	return compacted.Bytes()
}

func sampleCandidates() []campaign.Candidate {
	return []campaign.Candidate{
		{Identity: firstIdentity, Target: "row:a", Covers: []string{"row:a", "result:x"}, CaseID: "temporal.case.set.1", Fixture: "set-1", Case: sampleCase("temporal.case.set.1")},
		{Identity: secondIdentity, Target: "class:m:f:c", Covers: []string{"row:b", "class:m:f:c"}, CaseID: "temporal.case.set.2", Fixture: "set-2", Case: sampleCase("temporal.case.set.2")},
	}
}

func (s *scriptedBridge) serve(input io.Reader) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		var frame map[string]json.RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &frame); err != nil {
			s.write(map[string]any{"frame": "rejected", "seq": 0, "reason": "not a frame"})
			continue
		}
		s.write(s.answer(frame))
	}
}

func (s *scriptedBridge) write(reply map[string]any) {
	encoded, err := json.Marshal(reply)
	if err != nil {
		s.t.Logf("scripted bridge encode: %v", err)
		return
	}
	if _, err := s.writer.Write(append(encoded, '\n')); err != nil {
		s.t.Logf("scripted bridge write: %v", err)
	}
}

func text(frame map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(frame[key], &value)
	return value
}

func (s *scriptedBridge) answer(frame map[string]json.RawMessage) map[string]any {
	var seq int
	_ = json.Unmarshal(frame["seq"], &seq)
	if seq != s.seq+1 {
		return map[string]any{"frame": "rejected", "seq": seq, "reason": "out-of-order frame"}
	}
	kind := text(frame, "frame")
	reply := map[string]any{"seq": seq, "set": s.set, "profile": s.profile}
	switch kind {
	case "initialize":
		s.set, s.profile = text(frame, "set"), text(frame, "profile")
		reply["set"], reply["profile"] = s.set, s.profile
		reply["frame"] = "initialized"
		reply["machine"] = "temporal.nexus.caller.machine.nexusProtocol"
		reply["budget"] = "four"
		reply["limits"] = map[string]int{"steps": 4, "actions": 4, "search": 32768}
		reply["targets"] = s.targetKeys()
	case "next":
		if s.outstand != nil {
			return map[string]any{"frame": "rejected", "seq": seq, "reason": "a candidate is outstanding"}
		}
		reply["skipped"] = []campaign.Skipped{}
		if s.toolingOn > 0 && s.handed == s.toolingOn {
			reply["frame"] = "toolingFailure"
			reply["target"] = "row:z"
			reply["reason"] = "production: unbound"
			break
		}
		if s.handed >= len(s.candidates) {
			reply["frame"] = "exhausted"
			break
		}
		candidate := s.candidates[s.handed]
		s.handed++
		s.outstand = &candidate
		for _, key := range candidate.Covers {
			s.statuses[key] = "planned"
		}
		reply["frame"] = "candidate"
		reply["candidate"] = candidate.Identity
		reply["target"] = candidate.Target
		reply["covers"] = candidate.Covers
		reply["caseId"] = candidate.CaseID
		reply["fixture"] = candidate.Fixture
		reply["case"] = json.RawMessage(candidate.Case)
	case "observe":
		if s.outstand == nil || text(frame, "candidate") != s.outstand.Identity || text(frame, "profile") != s.profile {
			return map[string]any{"frame": "rejected", "seq": seq, "reason": "crossed observe"}
		}
		observation, detail := "prepare-rejected", "preparation was rejected"
		if raw, ok := frame["run"]; ok {
			var run testpilotspb.Run
			if err := protojson.Unmarshal(raw, &run); err != nil {
				return map[string]any{"frame": "rejected", "seq": seq, "reason": "run does not decode"}
			}
			observation, detail = readRun(s.outstand.CaseID, &run)
		}
		credited := []string{}
		status := "attempted"
		switch observation {
		case "satisfied":
			credited, status = s.outstand.Covers, "covered"
		case "violated":
			status = "violated"
			for _, key := range s.outstand.Covers {
				if strings.HasPrefix(key, "class:") {
					sample := proposalFor(s.outstand.Identity, s.proposalPath)
					sample.ClassName, sample.Target, sample.Candidate = "c", key, s.outstand.Identity
					s.violated = append(s.violated, sample)
				}
			}
		default:
		}
		statuses := []campaign.TargetStatus{}
		for _, key := range s.outstand.Covers {
			s.statuses[key] = status
			statuses = append(statuses, campaign.TargetStatus{Target: key, Status: status})
		}
		reply["frame"] = "credited"
		reply["candidate"] = s.outstand.Identity
		reply["observation"] = observation
		reply["detail"] = detail
		reply["credited"] = credited
		reply["statuses"] = statuses
		s.outstand = nil
	case "finish":
		if s.rejectFinish {
			return map[string]any{"frame": "rejected", "seq": seq, "reason": "the summary is refused"}
		}
		reply["frame"] = "finished"
		status := text(frame, "status")
		if status == "" {
			status = "stopped"
		}
		if s.handed == len(s.candidates) && s.outstand == nil {
			status = "exhausted"
		}
		reply["status"] = status
		summary := campaign.Summary{Targets: len(s.statuses), Selected: s.handed, Exhausted: status == "exhausted"}
		for _, value := range s.statuses {
			switch value {
			case "covered":
				summary.Covered++
			case "violated":
				summary.Violated++
			case "attempted":
				summary.Attempted++
			default:
				summary.Pending++
			}
		}
		reply["summary"] = summary
		counterexamples := []campaign.Counterexample{}
		counterexamples = append(counterexamples, s.violated...)
		reply["counterexamples"] = counterexamples
		ledger := []campaign.TargetStatus{}
		for _, key := range s.targetKeys() {
			ledger = append(ledger, campaign.TargetStatus{Target: key, Status: s.statuses[key]})
		}
		reply["ledger"] = ledger
	default:
		return map[string]any{"frame": "rejected", "seq": seq, "reason": "unknown frame"}
	}
	s.seq = seq
	return reply
}

func (s *scriptedBridge) targetKeys() []string {
	keys := []string{}
	for _, candidate := range s.candidates {
		keys = append(keys, candidate.Covers...)
	}
	return keys
}

// readRun mirrors the bridge's reading of a Run.
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
	case run.GetVerdict().GetStatus() == testpilotspb.VERDICT_STATUS_VIOLATED:
		return "violated", ""
	}
	return "inconclusive", "not decisive"
}

// scriptedBinder answers every candidate the same way.
type scriptedBinder struct {
	bindErr  error
	verdicts []testpilotspb.VerdictStatus
	cleanup  testpilotspb.CleanupStatus
	runs     int
	caseID   string
}

func (b *scriptedBinder) Bind(_ context.Context, _ string, source *testpilotspb.Case) (campaign.Bound, error) {
	if b.bindErr != nil {
		return nil, b.bindErr
	}
	b.caseID = source.GetCaseId()
	return b, nil
}

func (b *scriptedBinder) Run(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	status := testpilotspb.VERDICT_STATUS_SATISFIED
	if b.runs < len(b.verdicts) {
		status = b.verdicts[b.runs]
	}
	b.runs++
	cleanup := b.cleanup
	if cleanup == testpilotspb.CLEANUP_STATUS_UNSPECIFIED {
		cleanup = testpilotspb.CLEANUP_STATUS_SUCCEEDED
	}
	disposition := testpilotspb.RUN_DISPOSITION_COMPLETED
	if status == testpilotspb.VERDICT_STATUS_VIOLATED {
		disposition = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	}
	verdict := &testpilotspb.Verdict{Status: status}
	return &testpilotspb.Run{RunId: "run", CaseId: b.caseID, Disposition: disposition, Events: []*testpilotspb.RunEvent{{Sequence: 1}},
		Cleanup: &testpilotspb.CleanupOutcome{Status: cleanup}, Verdict: verdict}, verdict, nil
}

func (b *scriptedBinder) Release(context.Context) error { return nil }

func scriptedOpener(t testing.TB, binder campaign.Binder, candidates []campaign.Candidate, script func(*scriptedBridge)) opener {
	return func(ctx context.Context, configuration config, _ io.Writer) (*bound, error) {
		toBridge, fromClient := io.Pipe()
		toClient, fromBridge := io.Pipe()
		scripted := &scriptedBridge{t: t, writer: fromBridge, candidates: candidates, statuses: map[string]string{}}
		for _, candidate := range candidates {
			for _, key := range candidate.Covers {
				scripted.statuses[key] = "pending"
			}
		}
		if script != nil {
			script(scripted)
		}
		go scripted.serve(toBridge)
		bridge := campaign.New(fromClient, toClient, 1<<20)
		profile := "umpire-fuzz." + configuration.Deployment.Namespace
		opened, err := bridge.Initialize(ctx, configuration.Set, profile)
		if err != nil {
			return nil, err
		}
		return &bound{bridge: bridge, binder: binder, profile: profile, opened: opened, release: func(context.Context) error {
			return errors.Join(fromClient.Close(), toClient.Close())
		}}, nil
	}
}

func requiredFlags(extra ...string) []string {
	return append([]string{"run", "--set", "nexusCallerExploration", "--grpc", "127.0.0.1:7233", "--http", "127.0.0.1:7243",
		"--namespace", "fuzz", "--task-queue", "fuzz-queue"}, extra...)
}

func decodeSummary(t *testing.T, stdout string) summary {
	t.Helper()
	require.True(t, strings.HasSuffix(stdout, "\n"), "the summary ends with one LF")
	require.Equal(t, 1, strings.Count(stdout, "\n"), "the summary is one line")
	var decoded summary
	require.NoError(t, json.Unmarshal([]byte(stdout), &decoded))
	return decoded
}

func TestRunExhaustsTheSetAndReportsCoverageFromTheLedgerOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	candidates := sampleCandidates()
	binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE}}
	code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, binder, candidates, nil))
	require.Equal(t, exitExhausted, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "exhausted", decoded.Status)
	require.Equal(t, "nexusCallerExploration", decoded.Set)
	require.Equal(t, "umpire-fuzz.fuzz", decoded.Profile)
	require.Equal(t, "four", decoded.Budget)
	require.Equal(t, &campaign.Limits{Steps: 4, Actions: 4, Search: 32768}, decoded.Limits)
	require.Equal(t, campaign.Counters{Planned: 2, Prepared: 2, Started: 2, Decisive: 1, Inconclusive: 1, CaseBytes: int64(len(candidates[0].Case) + len(candidates[1].Case)), RunEvents: 2}, decoded.Counters)
	require.NotNil(t, decoded.Coverage)
	require.Equal(t, 2, decoded.Coverage.Covered, "only the satisfied Run's path is covered")
	require.Equal(t, 2, decoded.Coverage.Attempted, "the inconclusive Run's path is attempted, never covered")
	require.Empty(t, decoded.Counterexamples)
	require.Equal(t, []campaign.TargetStatus{{Target: "row:a", Status: "covered"}, {Target: "result:x", Status: "covered"}, {Target: "row:b", Status: "attempted"}, {Target: "class:m:f:c", Status: "attempted"}}, decoded.Targets)
	require.Len(t, decoded.Candidates, 2)
	require.Equal(t, candidate{Candidate: firstIdentity, Target: "row:a", Kind: "completed", Observation: "satisfied", Credited: []string{"row:a", "result:x"}}, decoded.Candidates[0])
	require.Equal(t, candidate{Candidate: secondIdentity, Target: "class:m:f:c", Kind: "completed", Observation: "inconclusive", Credited: []string{}}, decoded.Candidates[1])
	require.Contains(t, stderr.String(), "campaign exhausted")
}

func TestRunExitsOneOnACounterexample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_VIOLATED}}
	code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), nil))
	require.Equal(t, exitViolated, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "exhausted", decoded.Status)
	require.Len(t, decoded.Counterexamples, 1)
	require.Equal(t, "class:m:f:c", decoded.Counterexamples[0].Target)
	require.Equal(t, secondIdentity, decoded.Counterexamples[0].Candidate)
	require.Equal(t, 2, decoded.Coverage.Violated)
}

// The proposal is in the summary by digest and path, never by its bytes; with a promotion root it
// is written there, at the path the bridge named, and the summary says where. The same campaign
// twice writes the same summary bytes and the same file.
func TestRunWritesEachProposalUnderThePromotionRootOnly(t *testing.T) {
	violated := func() *scriptedBinder {
		return &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_VIOLATED}}
	}
	expected := proposalFor(secondIdentity, "")

	var stdout, stderr bytes.Buffer
	code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, violated(), sampleCandidates(), nil))
	require.Equal(t, exitViolated, code, stderr.String())
	withoutRoot := decodeSummary(t, stdout.String())
	require.Len(t, withoutRoot.Counterexamples, 1)
	require.Equal(t, counterexample{ClassName: "c", Target: "class:m:f:c", Candidate: secondIdentity,
		PromotionSourceSHA256: expected.PromotionSourceSHA256, PromotionSourcePath: expected.PromotionSourcePath}, withoutRoot.Counterexamples[0])
	require.NotContains(t, stdout.String(), "-- proposal", "the source bytes are never in the summary")

	root := filepath.Join(t.TempDir(), "proposals")
	var stdoutOnce, stdoutAgain bytes.Buffer
	code = Run(requiredFlags("--promotion-root", root), &stdoutOnce, &stderr, scriptedOpener(t, violated(), sampleCandidates(), nil))
	require.Equal(t, exitViolated, code, stderr.String())
	written := filepath.Join(root, expected.PromotionSourcePath)
	bytesOnce, err := os.ReadFile(written)
	require.NoError(t, err)
	require.Equal(t, expected.PromotionSource, string(bytesOnce))
	withRoot := decodeSummary(t, stdoutOnce.String())
	require.Len(t, withRoot.Counterexamples, 1)
	require.Equal(t, written, withRoot.Counterexamples[0].Written)
	require.Equal(t, expected.PromotionSourceSHA256, withRoot.Counterexamples[0].PromotionSourceSHA256)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1, "one file per counterexample, nothing else")

	code = Run(requiredFlags("--promotion-root", root), &stdoutAgain, &stderr, scriptedOpener(t, violated(), sampleCandidates(), nil))
	require.Equal(t, exitViolated, code, stderr.String())
	require.Equal(t, stdoutOnce.String(), stdoutAgain.String(), "the same campaign writes the same summary bytes")
	bytesAgain, err := os.ReadFile(written)
	require.NoError(t, err)
	require.Equal(t, bytesOnce, bytesAgain)
}

// A proposal path that would leave the promotion root is refused: the campaign's findings stand
// in the summary, the file is not written, and the command exits as a tooling failure.
func TestRunRefusesAProposalPathOutsideThePromotionRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proposals")
	binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_VIOLATED}}
	var stdout, stderr bytes.Buffer
	code := Run(requiredFlags("--promotion-root", root), &stdout, &stderr,
		scriptedOpener(t, binder, sampleCandidates(), func(s *scriptedBridge) { s.proposalPath = "../escaped.lean" }))
	require.Equal(t, exitToolingError, code, stderr.String())
	require.Contains(t, stderr.String(), "outside the promotion root")
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "tooling-failure", decoded.Status)
	require.Len(t, decoded.Counterexamples, 1)
	require.Empty(t, decoded.Counterexamples[0].Written)
	require.NoFileExists(t, filepath.Join(filepath.Dir(root), "escaped.lean"))
}

// Every proposal path is checked before any file is written: one path that would leave the root
// leaves the root empty, whatever came before it in the summary.
func TestWriteProposalsWritesNothingWhenAnyPathEscapes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proposals")
	first := proposalFor(firstIdentity, "")
	first.Candidate = firstIdentity
	escaping := proposalFor(secondIdentity, "../escaped.lean")
	escaping.Candidate = secondIdentity
	written, err := writeProposals(root, &campaign.Finished{Counterexamples: []campaign.Counterexample{first, escaping}})
	require.ErrorContains(t, err, "outside the promotion root")
	require.Empty(t, written)
	require.NoDirExists(t, root)
	written, err = writeProposals(root, &campaign.Finished{Counterexamples: []campaign.Counterexample{first, {Candidate: secondIdentity, PromotionError: "nonFoundResult"}}})
	require.NoError(t, err)
	require.Equal(t, map[string]string{firstIdentity: filepath.Join(root, first.PromotionSourcePath)}, written)
	written, err = writeProposals("", &campaign.Finished{Counterexamples: []campaign.Counterexample{first}})
	require.NoError(t, err)
	require.Empty(t, written, "no root, nothing written")
}

// The same scripted campaign twice writes the same summary bytes; a campaign whose bridge is cut
// after the first candidate writes that candidate as the full campaign did.
func TestRunWritesTheSameSummaryTwice(t *testing.T) {
	run := func(extra ...string) (summary, string) {
		var stdout, stderr bytes.Buffer
		binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE}}
		Run(requiredFlags(extra...), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), nil))
		return decodeSummary(t, stdout.String()), stdout.String()
	}
	full, once := run()
	_, again := run()
	require.Equal(t, once, again)
	capped, _ := run("--max-candidates", "1")
	require.Equal(t, "limit-reached", capped.Status)
	require.Equal(t, full.Candidates[:1], capped.Candidates)
}

func TestRunExitsTwoAtACapAndReportsWhichOne(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(requiredFlags("--max-candidates", "1"), &stdout, &stderr, scriptedOpener(t, &scriptedBinder{}, sampleCandidates(), nil))
	require.Equal(t, exitLimitOrStop, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "limit-reached", decoded.Status)
	require.Equal(t, "candidates", decoded.Limit)
	require.Equal(t, 1, decoded.Counters.Planned)
	require.False(t, decoded.Coverage.Exhausted)
	require.Contains(t, stderr.String(), "campaign limit-reached (candidates)")
}

// A counterexample outranks the cap that stopped the campaign: the finding is what it ran for.
func TestRunExitsOneOnACounterexampleEvenAtACap(t *testing.T) {
	var stdout, stderr bytes.Buffer
	candidates := sampleCandidates()
	candidates[0], candidates[1] = candidates[1], candidates[0]
	binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_VIOLATED}}
	code := Run(requiredFlags("--max-candidates", "1"), &stdout, &stderr, scriptedOpener(t, binder, candidates, nil))
	require.Equal(t, exitViolated, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "limit-reached", decoded.Status)
	require.Len(t, decoded.Counterexamples, 1)
}

func TestRunExitsTwoWhenStoppedByItsTimeoutAndNamesTheLostIteration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	binder := &blockingBinder{}
	code := Run(requiredFlags("--timeout", "300ms"), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), nil))
	require.Equal(t, exitLimitOrStop, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "stopped", decoded.Status)
	require.Equal(t, firstIdentity, decoded.Lost)
	require.Zero(t, decoded.Coverage.Covered, "a lost iteration is never coverage")
	require.Len(t, decoded.Candidates, 1)
	require.Equal(t, "lost", decoded.Candidates[0].Kind)
	require.Empty(t, decoded.Candidates[0].Observation)
	require.Contains(t, stderr.String(), "campaign stopped (lost "+firstIdentity+")")
}

// blockingBinder's Run waits for its context to end, as a Run interrupted by the campaign's
// timeout does.
type blockingBinder struct{}

func (blockingBinder) Bind(context.Context, string, *testpilotspb.Case) (campaign.Bound, error) {
	return blockingBinder{}, nil
}
func (blockingBinder) Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	<-ctx.Done()
	return nil, nil, ctx.Err()
}
func (blockingBinder) Release(context.Context) error { return nil }

func TestRunExitsThreeOnAToolingFailureAndSaysWhat(t *testing.T) {
	t.Run("the binding fails", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		binder := &scriptedBinder{bindErr: errors.New("open SDK client: connection refused")}
		code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), nil))
		require.Equal(t, exitToolingError, code)
		decoded := decodeSummary(t, stdout.String())
		require.Equal(t, "tooling-failure", decoded.Status)
		require.Contains(t, decoded.Failure, "connection refused")
		require.Equal(t, 1, decoded.Counters.Failed)
		require.Contains(t, stderr.String(), "campaign tooling-failure: ")
	})
	t.Run("the bridge fails", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, &scriptedBinder{}, sampleCandidates(), func(s *scriptedBridge) { s.toolingOn = 1 }))
		require.Equal(t, exitToolingError, code)
		decoded := decodeSummary(t, stdout.String())
		require.Equal(t, "tooling-failure", decoded.Status)
		require.Contains(t, decoded.Failure, "production: unbound")
		require.Equal(t, 1, decoded.Counters.Planned)
	})
	t.Run("the campaign cannot open", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(requiredFlags(), &stdout, &stderr, func(context.Context, config, io.Writer) (*bound, error) {
			return nil, errors.New(`dial "127.0.0.1:7233": refused`)
		})
		require.Equal(t, exitToolingError, code)
		require.Empty(t, stdout.String())
		require.Contains(t, stderr.String(), "refused")
	})
}

// A preparation rejection is reported as such, never as coverage, and the campaign goes on.
func TestRunReportsPreparationRejectionsAndGoesOn(t *testing.T) {
	var stdout, stderr bytes.Buffer
	binder := &scriptedBinder{bindErr: &testpilot.PreparationError{Category: testpilot.PreparationUnsupported, Path: "program", Detail: "opcode"}}
	code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), nil))
	require.Equal(t, exitExhausted, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, 2, decoded.Counters.Rejected)
	require.Zero(t, decoded.Counters.Started)
	require.Zero(t, decoded.Coverage.Covered)
	require.Equal(t, 4, decoded.Coverage.Attempted)
	require.Equal(t, "prepare-rejected", decoded.Candidates[0].Kind)
	require.Equal(t, "unsupported at program: opcode", decoded.Candidates[0].Detail)
}

// wideCandidates cover enough targets that the full summary is over a small cap while the
// terminal-only summary is under it.
func wideCandidates() []campaign.Candidate {
	candidates := sampleCandidates()
	for index := range 40 {
		candidates[0].Covers = append(candidates[0].Covers, "row:wide-"+strings.Repeat("x", 20)+string(rune('a'+index%26)))
	}
	return candidates
}

// The report cap is enforced on the rendered summary: over it, the terminal, the counters and the
// counterexamples are written as limit-reached, never a truncated report. The terminal-only
// summary is the floor a cap cannot go below, which the flag set refuses.
func TestRunReportsLimitReachedRatherThanTruncatingTheSummary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(requiredFlags("--max-report-bytes", "1024"), &stdout, &stderr, scriptedOpener(t, &scriptedBinder{}, wideCandidates(), nil))
	require.Equal(t, exitLimitOrStop, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "limit-reached", decoded.Status)
	require.Equal(t, "report-bytes", decoded.Limit)
	require.Nil(t, decoded.Coverage)
	require.Empty(t, decoded.Candidates)
	require.LessOrEqual(t, stdout.Len(), 1024, "the terminal-only summary fits the cap")
	require.Contains(t, stderr.String(), "report-bytes")
}

// A failure, a stop or a campaign cap keeps its terminal under the report cap: what it names
// outranks the cap.
func TestReportCapFallbackKeepsAFailureOrStopTerminal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	binder := &scriptedBinder{bindErr: errors.New("open SDK client: connection refused")}
	code := Run(requiredFlags("--max-report-bytes", "1024"), &stdout, &stderr, scriptedOpener(t, binder, wideCandidates(), nil))
	require.Equal(t, exitToolingError, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "tooling-failure", decoded.Status)
	require.Contains(t, decoded.Failure, "connection refused")
	require.Nil(t, decoded.Coverage)

	stdout.Reset()
	code = Run(requiredFlags("--max-report-bytes", "1024", "--max-candidates", "1"), &stdout, &stderr, scriptedOpener(t, &scriptedBinder{}, wideCandidates(), nil))
	require.Equal(t, exitLimitOrStop, code, stderr.String())
	decoded = decodeSummary(t, stdout.String())
	require.Equal(t, "limit-reached", decoded.Status)
	require.Equal(t, "candidates", decoded.Limit, "the cap that ended the campaign is kept")
	require.Nil(t, decoded.Coverage)
	require.Contains(t, stderr.String(), "report-bytes")
}

// The counterexamples survive the report cap: they are what exit 1 names, and they are bounded by
// the class targets, not by the campaign.
func TestReportCapKeepsTheCounterexamples(t *testing.T) {
	var stdout, stderr bytes.Buffer
	binder := &scriptedBinder{verdicts: []testpilotspb.VerdictStatus{testpilotspb.VERDICT_STATUS_SATISFIED, testpilotspb.VERDICT_STATUS_VIOLATED}}
	code := Run(requiredFlags("--max-report-bytes", "1024"), &stdout, &stderr, scriptedOpener(t, binder, wideCandidates(), nil))
	require.Equal(t, exitViolated, code, stderr.String())
	decoded := decodeSummary(t, stdout.String())
	require.Equal(t, "limit-reached", decoded.Status)
	require.Equal(t, "report-bytes", decoded.Limit)
	require.Len(t, decoded.Counterexamples, 1)
	require.Equal(t, secondIdentity, decoded.Counterexamples[0].Candidate)
	require.Equal(t, "class:m:f:c", decoded.Counterexamples[0].Target)
	require.Nil(t, decoded.Coverage)
	require.Empty(t, decoded.Candidates)
}

// A summary the bridge refuses after an exhausted campaign is a tooling failure, never an exit 0
// with no coverage; a violated Run observed before a stop still exits 1 without the summary.
func TestRunNeverTrustsAMissingSummary(t *testing.T) {
	t.Run("refused after exhaustion", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(requiredFlags(), &stdout, &stderr, scriptedOpener(t, &scriptedBinder{}, sampleCandidates(), func(s *scriptedBridge) { s.rejectFinish = true }))
		require.Equal(t, exitToolingError, code, stderr.String())
		decoded := decodeSummary(t, stdout.String())
		require.Equal(t, "tooling-failure", decoded.Status)
		require.Contains(t, decoded.Failure, "refused")
		require.Nil(t, decoded.Coverage)
	})
	t.Run("violated before a stop", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		binder := &violatedThenBlocking{}
		code := Run(requiredFlags("--timeout", "300ms"), &stdout, &stderr, scriptedOpener(t, binder, sampleCandidates(), func(s *scriptedBridge) { s.rejectFinish = true }))
		require.Equal(t, exitViolated, code, stderr.String())
		decoded := decodeSummary(t, stdout.String())
		require.Equal(t, "stopped", decoded.Status)
		require.Equal(t, secondIdentity, decoded.Lost)
		require.Equal(t, "violated", decoded.Candidates[0].Observation)
	})
	t.Run("an empty report", func(t *testing.T) {
		settled := settle(campaign.Report{}, errors.New("invariant"))
		require.Equal(t, campaign.StatusToolingFailure, settled.Terminal.Status)
		require.Equal(t, "invariant", settled.Terminal.Failure)
		require.Equal(t, exitToolingError, exitCode(settled))
	})
}

// violatedThenBlocking's first Run is violated; its second waits for the campaign to end.
type violatedThenBlocking struct {
	runs   int
	caseID string
}

func (b *violatedThenBlocking) Bind(_ context.Context, _ string, source *testpilotspb.Case) (campaign.Bound, error) {
	b.caseID = source.GetCaseId()
	return b, nil
}
func (b *violatedThenBlocking) Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	b.runs++
	if b.runs > 1 {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	verdict := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}
	return &testpilotspb.Run{RunId: "run", CaseId: b.caseID, Disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
		Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}, Verdict: verdict}, verdict, nil
}
func (b *violatedThenBlocking) Release(context.Context) error { return nil }

// The real opening starts the bridge on a context that outlives the campaign's, so a stopped
// campaign still reads its summary; the stand-in bridge here is a script that answers the frames.
func TestOpenCampaignStartsTheBridgeBeyondTheCampaignContext(t *testing.T) {
	script := filepath.Join(t.TempDir(), "bridge.sh")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
read line
printf '%s\n' '{"frame":"initialized","seq":1,"set":"s","profile":"umpire-fuzz.fuzz","machine":"m","budget":"b","limits":{"steps":1,"actions":1,"search":1},"targets":[]}'
read line
printf '%s\n' '{"frame":"finished","seq":2,"set":"s","profile":"umpire-fuzz.fuzz","status":"stopped","summary":{},"counterexamples":[],"ledger":[]}'
cat >/dev/null
`), 0o755))
	var stderr bytes.Buffer
	configuration, err := parseConfig(requiredFlags("--set", "s", "--bridge", script, "--model-root", t.TempDir(), "--grpc", "127.0.0.1:1"), &stderr)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	opened, err := openCampaign(ctx, configuration, &stderr)
	require.NoError(t, err)
	require.Equal(t, "m", opened.opened.Machine)
	cancel()
	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelFinish()
	finished, err := opened.bridge.Finish(finishCtx, "stopped")
	require.NoError(t, err)
	require.Equal(t, "stopped", finished.Status)
	require.NoError(t, opened.release(context.WithoutCancel(ctx)))
}

func TestRunRejectsTheCommandLineBeforeOpeningAnything(t *testing.T) {
	for _, probe := range []struct {
		name      string
		arguments []string
		message   string
	}{
		{"no subcommand", []string{}, "usage"},
		{"other subcommand", []string{"list"}, "usage"},
		{"no set", []string{"run", "--grpc", "a", "--http", "b", "--namespace", "c", "--task-queue", "d"}, "--set is required"},
		{"no namespace", []string{"run", "--set", "s", "--grpc", "a", "--http", "b", "--task-queue", "d"}, "--namespace is required"},
		{"a target", requiredFlags("--target", "row:x"), "flag provided but not defined: -target"},
		{"a wider limit", requiredFlags("--search", "99999"), "flag provided but not defined: -search"},
		{"positional", requiredFlags("extra"), "accepts no positional arguments"},
		{"non-positive timeout", requiredFlags("--timeout", "0s"), "must be positive"},
		{"negative cap", requiredFlags("--max-candidates", "-1"), "must not be negative"},
		{"report cap below the floor", requiredFlags("--max-report-bytes", "200"), "at least"},
		{"promotion root under the model", requiredFlags("--model-root", "m", "--promotion-root", "m/proposals"), "must not be under the model root"},
		{"promotion root is the model", requiredFlags("--model-root", "m", "--promotion-root", "m"), "must not be under the model root"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			opened := false
			code := Run(probe.arguments, &stdout, &stderr, func(context.Context, config, io.Writer) (*bound, error) {
				opened = true
				return nil, nil
			})
			require.Equal(t, exitToolingError, code)
			require.False(t, opened, "nothing is opened on a rejected command line")
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), probe.message)
		})
	}
}

func TestParseConfigDerivesTheBridgeFromTheModelRoot(t *testing.T) {
	var stderr bytes.Buffer
	configuration, err := parseConfig(requiredFlags("--model-root", "elsewhere"), &stderr)
	require.NoError(t, err)
	root, err := filepath.Abs("elsewhere")
	require.NoError(t, err)
	require.Equal(t, root, configuration.ModelRoot)
	require.Equal(t, filepath.Join(root, ".lake", "build", "bin", "umpire-explore"), configuration.Bridge)
	require.Equal(t, "fuzz-queue", configuration.Deployment.TaskQueue)
	require.Equal(t, defaultRunTimeout, configuration.Caps.RunTimeout)
	configuration, err = parseConfig(requiredFlags("--bridge", "explore"), &stderr)
	require.NoError(t, err)
	bridge, err := filepath.Abs("explore")
	require.NoError(t, err)
	require.Equal(t, bridge, configuration.Bridge)
}

// The default flags name the bridge relative to the model root, and the bridge runs in that
// root: the command must find `model/.lake/build/bin/umpire-explore` from the invoking directory,
// not from inside `model`.
func TestOpenCampaignFindsTheDefaultBridgeFromTheInvokingDirectory(t *testing.T) {
	invokedIn := t.TempDir()
	bridgeDir := filepath.Join(invokedIn, defaultModelRoot, filepath.FromSlash(filepath.Dir(bridgeRelativePath)))
	require.NoError(t, os.MkdirAll(bridgeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bridgeDir, filepath.Base(bridgeRelativePath)), []byte(`#!/bin/sh
read line
printf '%s\n' '{"frame":"initialized","seq":1,"set":"s","profile":"umpire-fuzz.fuzz","machine":"m","budget":"b","limits":{"steps":1,"actions":1,"search":1},"targets":[]}'
read line
printf '%s\n' '{"frame":"finished","seq":2,"set":"s","profile":"umpire-fuzz.fuzz","status":"exhausted","summary":{},"counterexamples":[],"ledger":[]}'
cat >/dev/null
`), 0o755))
	t.Chdir(invokedIn)
	var stderr bytes.Buffer
	configuration, err := parseConfig(requiredFlags("--set", "s", "--grpc", "127.0.0.1:1"), &stderr)
	require.NoError(t, err)
	opened, err := openCampaign(t.Context(), configuration, &stderr)
	require.NoError(t, err)
	require.Equal(t, "m", opened.opened.Machine)
	finished, err := opened.bridge.Finish(t.Context(), "exhausted")
	require.NoError(t, err)
	require.Equal(t, "exhausted", finished.Status)
	require.NoError(t, opened.release(t.Context()))
}
