package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/campaign"
)

// fakeEdit is one edit the fake bridge's sweep holds: inapplicable ones produce no Case.
type fakeEdit struct {
	Edit
	inapplicable bool
}

// fakeReplayBridge stands in for `umpire-replay-bridge` at the protocol level: it echoes sequence,
// set and profile, hands out one candidate at a time, settles each edit as the Lean reduction
// does, and ends as it does. Its candidates all carry one Case, which the scripted binder runs.
type fakeReplayBridge struct {
	t        testing.TB
	writer   io.Writer
	identity string
	caseID   string
	caseJSON json.RawMessage
	sweep    []fakeEdit
	crossed  bool
	capped   bool
	// rejectAdmit, when set, rejects admit with this reason, as the bridge rejects a set, Query or
	// target it cannot recover.
	rejectAdmit string
	// proposalError, when set, is the finished frame's proposal error in place of a compiled one.
	proposalError string

	set, profile string
	seq          int
	pending      int
	outstanding  *fakeEdit
	settled      []Settled
	retained     string
	ended        string
	frames       []bridgeRequest
}

func newFakeReplayBridge(t testing.TB, caseJSON json.RawMessage, caseID string, sweep ...fakeEdit) (*Bridge, *fakeReplayBridge) {
	t.Helper()
	toBridge, fromClient := io.Pipe()
	toClient, fromBridge := io.Pipe()
	fake := &fakeReplayBridge{t: t, writer: fromBridge, caseJSON: caseJSON, caseID: caseID, sweep: sweep, identity: "subject-identity"}
	go fake.serve(toBridge)
	t.Cleanup(func() {
		_ = fromClient.Close()
		_ = toClient.Close()
	})
	return NewBridge(fromClient, toClient, 1<<20), fake
}

func (f *fakeReplayBridge) serve(input io.Reader) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		var frame bridgeRequest
		if err := json.Unmarshal(scanner.Bytes(), &frame); err != nil {
			f.write(map[string]any{"frame": "rejected", "seq": 0, "reason": "not a frame"})
			continue
		}
		f.frames = append(f.frames, frame)
		f.write(f.answer(frame))
	}
}

func (f *fakeReplayBridge) write(reply map[string]any) {
	encoded, err := json.Marshal(reply)
	if err != nil {
		f.t.Errorf("fake replay bridge encode: %v", err)
		return
	}
	if _, err := io.WriteString(f.writer, string(encoded)+"\n"); err != nil {
		f.t.Logf("fake replay bridge write: %v", err)
	}
}

func (f *fakeReplayBridge) digest(edit Edit) string { return fmt.Sprintf("candidate-%d", edit.Index) }

func (f *fakeReplayBridge) answer(frame bridgeRequest) map[string]any {
	if frame.Seq != f.seq+1 {
		return map[string]any{"frame": "rejected", "seq": frame.Seq, "reason": "out-of-order frame"}
	}
	if frame.Frame == "admit" && f.rejectAdmit != "" {
		return map[string]any{"frame": "rejected", "seq": frame.Seq, "reason": f.rejectAdmit}
	}
	if frame.Frame == "admit" {
		f.set, f.profile = frame.Set, frame.Profile
	}
	reply := map[string]any{"seq": frame.Seq, "set": f.set, "profile": f.profile}
	f.seq = frame.Seq
	switch frame.Frame {
	case "admit":
		if f.crossed || frame.Identity != f.identity {
			reply["frame"], reply["reason"] = "crossed", "the set produces another Case"
			return reply
		}
		edits := make([]Edit, 0, len(f.sweep))
		for _, edit := range f.sweep {
			edits = append(edits, edit.Edit)
		}
		f.retained = "subject-digest"
		reply["frame"], reply["subject"], reply["caseId"], reply["fixture"] = "admitted", f.retained, f.caseID, "fixture"
		reply["identity"], reply["edits"], reply["capped"] = frame.Identity, edits, f.capped
	case "next":
		if f.ended != "" {
			f.seq--
			return map[string]any{"frame": "rejected", "seq": frame.Seq, "reason": "the reduction ended; send `finish`"}
		}
		skipped := []Settled{}
		for f.pending < len(f.sweep) && f.sweep[f.pending].inapplicable {
			settled := Settled{Edit: f.sweep[f.pending].Edit, Fate: "inapplicable", Reason: "not-selected"}
			f.settled = append(f.settled, settled)
			skipped = append(skipped, settled)
			f.pending++
		}
		reply["skipped"] = skipped
		if f.pending >= len(f.sweep) {
			reply["frame"] = "exhausted"
			return reply
		}
		edit := f.sweep[f.pending]
		f.outstanding = &edit
		reply["frame"], reply["candidate"], reply["edit"], reply["index"], reply["action"] =
			"candidate", f.digest(edit.Edit), edit.Edit.Edit, edit.Index, edit.Action
		reply["caseId"], reply["fixture"], reply["identity"], reply["case"] = f.caseID, "fixture", "case-identity", f.caseJSON
	case "observe":
		edit := f.outstanding
		f.outstanding = nil
		f.pending++
		digest := f.digest(edit.Edit)
		fate := map[string]string{"reproduced": "retained", "not-reproduced": "not-reproduced", "indeterminate": "undecided"}[frame.Class]
		reason := ""
		if frame.PrepareRejected != nil {
			fate, reason = "rejected", "preparation: "+*frame.PrepareRejected
		}
		if fate == "retained" {
			f.retained = digest
		}
		if fate == "undecided" {
			f.ended = edit.Edit.Edit + " is undecided"
		}
		f.settled = append(f.settled, Settled{Edit: edit.Edit, Fate: fate, Reason: reason, Candidate: &digest})
		reply["frame"], reply["candidate"], reply["edit"], reply["index"], reply["action"] = "settled", digest, edit.Edit.Edit, edit.Index, edit.Action
		reply["fate"], reply["reason"], reply["retained"] = fate, reason, f.retained
	case "finish":
		status, reason := "irreducible", ""
		for _, settled := range f.settled {
			if settled.Fate == "retained" {
				status = "minimized"
			}
		}
		switch {
		case f.capped && f.ended == "" && f.outstanding == nil && frame.Status == "":
			status, reason = "incomplete", "the sweep was capped at 8 edits"
		case f.ended != "":
			status, reason = "incomplete", f.ended
		case f.outstanding != nil:
			status, reason = "incomplete", "candidate "+f.digest(f.outstanding.Edit)+" was never settled"
		case frame.Status != "":
			status, reason = "incomplete", frame.Status
		case f.pending < len(f.sweep):
			status, reason = "incomplete", "the sweep did not finish"
		default:
		}
		reply["frame"], reply["status"], reply["reason"], reply["subject"], reply["retained"], reply["edits"] =
			"finished", status, reason, "subject-digest", f.retained, f.settled
		reply["proposal"] = nil
		if (status == "minimized" || status == "irreducible") && f.proposalError != "" {
			reply["proposal"] = map[string]any{"digest": f.retained, "promotionSourceSha256": nil, "promotionError": f.proposalError}
		} else if status == "minimized" || status == "irreducible" {
			reply["proposal"] = map[string]any{
				"digest": f.retained, "promotionSourceSha256": "sha256:" + f.retained,
				"promotionSourcePath": f.set + "-" + f.retained + ".lean", "promotionSource": "-- proposal " + f.retained,
			}
		}
	default:
		return map[string]any{"frame": "rejected", "seq": frame.Seq, "reason": "unknown frame"}
	}
	return reply
}

func edit(index int, action string) fakeEdit {
	return fakeEdit{Edit: Edit{Edit: fmt.Sprintf("dropPrefixStep %d", index), Index: index, Action: action}}
}

// The client admits, hands candidates out one at a time and settles them; a subject the bridge
// does not produce is crossed and ends the protocol.
func TestBridgeClientAdmitsAndSettles(t *testing.T) {
	bridge, fake := newFakeReplayBridge(t, json.RawMessage(`{"caseId":"c"}`), "c", edit(1, "b"), edit(0, "a"))
	ctx := t.Context()
	_, err := bridge.Next(ctx)
	require.ErrorIs(t, err, ErrNotAdmitted)
	_, err = bridge.Admit(ctx, "set", "profile", Named{Query: "q", Target: "t"}, fake.identity)
	require.Error(t, err, "exactly one of a Query and a target")
	admitted, err := bridge.Admit(ctx, "set", "profile", Named{Query: "q"}, fake.identity)
	require.NoError(t, err)
	require.Equal(t, []Edit{edit(1, "b").Edit, edit(0, "a").Edit}, admitted.Edits)
	next, err := bridge.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, "candidate-1", next.Candidate.Digest)
	_, err = bridge.Next(ctx)
	require.ErrorIs(t, err, ErrCandidateOutstanding)
	_, err = bridge.Observe(ctx, "candidate-9", Decided{Class: ClassReproduced})
	require.ErrorIs(t, err, ErrCrossedCandidate)
	settled, err := bridge.Observe(ctx, "candidate-1", Decided{Class: ClassReproduced})
	require.NoError(t, err)
	require.Equal(t, "retained", settled.Fate)
	require.Equal(t, "candidate-1", settled.Retained)
	finished, err := bridge.Finish(ctx, "")
	require.NoError(t, err)
	require.Equal(t, "incomplete", finished.Status, "one edit was never tried")
	_, err = bridge.Next(ctx)
	require.ErrorIs(t, err, ErrReductionFinished)

	crossedBridge, crossedFake := newFakeReplayBridge(t, nil, "c")
	crossedFake.crossed = true
	_, err = crossedBridge.Admit(ctx, "set", "profile", Named{Query: "q"}, crossedFake.identity)
	var crossed *CrossedError
	require.ErrorAs(t, err, &crossed)
	_, err = crossedBridge.Next(ctx)
	require.ErrorIs(t, err, ErrReductionFinished)
}

// A reply out of step breaks the client: every later call returns the same failure.
func TestBridgeClientBreaksOnAMismatchedReply(t *testing.T) {
	toBridge, fromClient := io.Pipe()
	toClient, fromBridge := io.Pipe()
	t.Cleanup(func() { _ = fromClient.Close(); _ = toClient.Close() })
	go func() {
		scanner := bufio.NewScanner(toBridge)
		for scanner.Scan() {
			_, _ = io.WriteString(fromBridge, `{"frame":"admitted","seq":7,"set":"set","profile":"profile","edits":[]}`+"\n")
		}
	}()
	bridge := NewBridge(fromClient, toClient, 1<<20)
	_, err := bridge.Admit(t.Context(), "set", "profile", Named{Query: "q"}, "identity")
	var protocol *campaign.ProtocolError
	require.ErrorAs(t, err, &protocol)
	require.ErrorIs(t, bridge.Broken(), campaign.ErrBroken)
	_, err = bridge.Admit(t.Context(), "set", "profile", Named{Query: "q"}, "identity")
	require.ErrorIs(t, err, campaign.ErrBroken)
}
