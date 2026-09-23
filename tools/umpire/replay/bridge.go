package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.temporal.io/server/tools/umpire/campaign"
)

// The replay bridge's frames. Go names a set, a Query or a target key and the subject's Case
// identity, receives whole Cases, and returns classes; it never names an edit or changes a Case.

var (
	// ErrNotAdmitted is returned when a frame is sent before Admit succeeded.
	ErrNotAdmitted = errors.New("no subject is admitted")
	// ErrReductionFinished is returned once Finish has been answered or admission was crossed.
	ErrReductionFinished = errors.New("the reduction is finished")
	// ErrCandidateOutstanding is returned by Next while a candidate's Case is with the caller.
	ErrCandidateOutstanding = errors.New("a candidate is outstanding; observe it before asking for the next")
	// ErrNoCandidate is returned by Observe when no candidate is outstanding.
	ErrNoCandidate = errors.New("no candidate is outstanding")
	// ErrCrossedCandidate is returned by Observe for a candidate other than the outstanding one.
	ErrCrossedCandidate = errors.New("the observation names a candidate other than the outstanding one")
)

// CrossedError is the bridge's answer that no set of the Model produces the subject's Case: the
// Case the named Query produces is not the subject's bytes. The protocol has ended.
type CrossedError struct{ Reason string }

func (e *CrossedError) Error() string { return "crossed: " + e.Reason }

// Named is what admission names the subject's Query by: a functional set's Query, or an
// exploratory set's target key. Exactly one is set.
type Named struct {
	Query  string
	Target string
}

// Edit is one `dropPrefixStep` edit as the bridge renders it: Go reads it and interprets nothing.
type Edit struct {
	Edit   string `json:"edit"`
	Index  int    `json:"index"`
	Action string `json:"action"`
}

// Settled is one edit with what it came to, as the bridge renders it.
type Settled struct {
	Edit
	Fate      string  `json:"fate"`
	Reason    string  `json:"reason"`
	Candidate *string `json:"candidate"`
}

// Admitted is the bridge's answer to admit: the subject's digest and Case identity, and the
// sweep's edits in the order they are tried.
type Admitted struct {
	Subject  string
	CaseID   string
	Fixture  string
	Identity string
	Edits    []Edit
	Capped   bool
}

// BridgeCandidate is one whole candidate Case with its digest (the edited Query's Plan checksum)
// and its Case checksum beside it.
type BridgeCandidate struct {
	Digest   string
	Edit     Edit
	CaseID   string
	Fixture  string
	Identity string
	Case     json.RawMessage
}

// BridgeNext is the bridge's answer to next: a candidate or exhaustion, with the edits passed
// over on the way.
type BridgeNext struct {
	Candidate *BridgeCandidate
	Exhausted bool
	Skipped   []Settled
}

// Decided is what Go says about the outstanding candidate's Runs: its pair class after the one
// retry, or the fact that its preparation was rejected.
type Decided struct {
	Class           Class
	PrepareRejected *string
}

// SettledReply is the bridge's answer to observe: the edit's fate and the digest later edits
// apply to.
type SettledReply struct {
	Settled
	Retained string
}

// BridgeFinished is the bridge's answer to finish: the result, the digest retained and every
// edit's fate.
type BridgeFinished struct {
	Status   string
	Reason   string
	Subject  string
	Retained string
	Edits    []Settled
}

type bridgeRequest struct {
	Frame           string  `json:"frame"`
	Seq             int     `json:"seq"`
	Set             string  `json:"set"`
	Profile         string  `json:"profile,omitempty"`
	Query           string  `json:"query,omitempty"`
	Target          string  `json:"target,omitempty"`
	Identity        string  `json:"identity,omitempty"`
	Candidate       string  `json:"candidate,omitempty"`
	Class           string  `json:"class,omitempty"`
	PrepareRejected *string `json:"prepareRejected,omitempty"`
	Status          string  `json:"status,omitempty"`
}

type bridgeReply struct {
	Frame     string          `json:"frame"`
	Reason    string          `json:"reason"`
	Subject   string          `json:"subject"`
	CaseID    string          `json:"caseId"`
	Fixture   string          `json:"fixture"`
	Identity  string          `json:"identity"`
	Edits     json.RawMessage `json:"edits"`
	Capped    bool            `json:"capped"`
	Candidate *string         `json:"candidate"`
	Edit      string          `json:"edit"`
	Index     int             `json:"index"`
	Action    string          `json:"action"`
	Skipped   []Settled       `json:"skipped"`
	Case      json.RawMessage `json:"case"`
	Fate      string          `json:"fate"`
	Retained  string          `json:"retained"`
	Status    string          `json:"status"`
}

// Bridge is the client of one replay bridge process: one request outstanding, every reply
// matched to its request, and a candidate's Case with the caller until Observe returns.
type Bridge struct {
	conn        *campaign.Conn
	profile     string
	admitted    bool
	outstanding *BridgeCandidate
}

// StartBridge spawns `umpire-replay-bridge`. Close ends it.
func StartBridge(ctx context.Context, options campaign.Options) (*Bridge, error) {
	conn, err := campaign.StartConn(ctx, options)
	if err != nil {
		return nil, err
	}
	return &Bridge{conn: conn}, nil
}

// NewBridge is a replay bridge client over already-open streams.
func NewBridge(stdin io.Writer, stdout io.Reader, maxFrameBytes int) *Bridge {
	return &Bridge{conn: campaign.NewConn(stdin, stdout, maxFrameBytes)}
}

// Close ends the bridge process.
func (b *Bridge) Close() error { return b.conn.Close() }

// Broken is the failure after which the bridge can no longer be trusted, or nil.
func (b *Bridge) Broken() error { return b.conn.Broken() }

// Outstanding is the candidate whose Case is with the caller, or nil.
func (b *Bridge) Outstanding() *BridgeCandidate { return b.outstanding }

// Admit names the subject's set and Query (or target key) and its Case identity. A subject the
// set does not produce is a *CrossedError, after which the protocol has ended.
func (b *Bridge) Admit(ctx context.Context, set, profile string, named Named, identity string) (Admitted, error) {
	if err := b.conn.Broken(); err != nil {
		return Admitted{}, err
	}
	if b.conn.Finished() {
		return Admitted{}, ErrReductionFinished
	}
	if b.admitted {
		return Admitted{}, fmt.Errorf("a subject of %s is already admitted", b.conn.Set())
	}
	if set == "" || profile == "" || identity == "" || (named.Query == "") == (named.Target == "") {
		return Admitted{}, errors.New("a set, a profile, an identity and exactly one of a Query and a target are required")
	}
	b.conn.Scope(set)
	answer, err := b.exchange(ctx, bridgeRequest{
		Frame: "admit", Profile: profile, Query: named.Query, Target: named.Target, Identity: identity,
	}, profile, nil, "admitted", "crossed")
	if err != nil {
		b.conn.Scope("")
		return Admitted{}, err
	}
	if answer.Frame == "crossed" {
		b.conn.Finish()
		return Admitted{}, &CrossedError{Reason: answer.Reason}
	}
	var edits []Edit
	if err := json.Unmarshal(answer.Edits, &edits); err != nil {
		return Admitted{}, fmt.Errorf("decode admitted edits: %w", err)
	}
	b.profile = profile
	b.admitted = true
	return Admitted{
		Subject: answer.Subject, CaseID: answer.CaseID, Fixture: answer.Fixture,
		Identity: answer.Identity, Edits: edits, Capped: answer.Capped,
	}, nil
}

// Next asks for the next candidate. It refuses, before writing anything, while one is outstanding.
func (b *Bridge) Next(ctx context.Context) (BridgeNext, error) {
	if err := b.open(); err != nil {
		return BridgeNext{}, err
	}
	if b.outstanding != nil {
		return BridgeNext{}, ErrCandidateOutstanding
	}
	whole := func(answer bridgeReply) error {
		if answer.Frame == "candidate" && (answer.Candidate == nil || *answer.Candidate == "" ||
			len(answer.Case) == 0 || answer.CaseID == "" || answer.Identity == "") {
			return &campaign.ProtocolError{Expected: "a candidate with a digest, a Case ID, an identity and a Case", Actual: "an incomplete candidate frame"}
		}
		return nil
	}
	answer, err := b.exchange(ctx, bridgeRequest{Frame: "next"}, b.profile, whole, "candidate", "exhausted")
	if err != nil {
		return BridgeNext{}, err
	}
	if answer.Frame == "exhausted" {
		return BridgeNext{Exhausted: true, Skipped: answer.Skipped}, nil
	}
	candidate := &BridgeCandidate{
		Digest:   *answer.Candidate,
		Edit:     Edit{Edit: answer.Edit, Index: answer.Index, Action: answer.Action},
		CaseID:   answer.CaseID,
		Fixture:  answer.Fixture,
		Identity: answer.Identity,
		Case:     answer.Case,
	}
	b.outstanding = candidate
	return BridgeNext{Candidate: candidate, Skipped: answer.Skipped}, nil
}

// Observe hands back what the outstanding candidate's Runs decided. On a settled answer the
// candidate is no longer outstanding; on a rejection it still is.
func (b *Bridge) Observe(ctx context.Context, digest string, decided Decided) (SettledReply, error) {
	if err := b.open(); err != nil {
		return SettledReply{}, err
	}
	if b.outstanding == nil {
		return SettledReply{}, ErrNoCandidate
	}
	if digest != b.outstanding.Digest {
		return SettledReply{}, ErrCrossedCandidate
	}
	if (decided.Class == "") == (decided.PrepareRejected == nil) {
		return SettledReply{}, errors.New("an observation carries exactly one of a class and a preparation rejection")
	}
	same := func(answer bridgeReply) error {
		if answer.Candidate == nil || *answer.Candidate != digest {
			return &campaign.ProtocolError{Expected: "candidate " + digest, Actual: "another candidate"}
		}
		return nil
	}
	answer, err := b.exchange(ctx, bridgeRequest{
		Frame: "observe", Profile: b.profile, Candidate: digest,
		Class: string(decided.Class), PrepareRejected: decided.PrepareRejected,
	}, b.profile, same, "settled")
	if err != nil {
		return SettledReply{}, err
	}
	b.outstanding = nil
	return SettledReply{
		Settled: Settled{
			Edit: Edit{Edit: answer.Edit, Index: answer.Index, Action: answer.Action},
			Fate: answer.Fate, Reason: answer.Reason, Candidate: answer.Candidate,
		},
		Retained: answer.Retained,
	}, nil
}

// Finish asks for the result. status names why the coordinator ended the reduction early
// ("stopped" or "limit-reached"); empty leaves it to the bridge.
func (b *Bridge) Finish(ctx context.Context, status string) (BridgeFinished, error) {
	if err := b.open(); err != nil {
		return BridgeFinished{}, err
	}
	answer, err := b.exchange(ctx, bridgeRequest{Frame: "finish", Status: status}, b.profile, nil, "finished")
	if err != nil {
		return BridgeFinished{}, err
	}
	b.conn.Finish()
	var edits []Settled
	if err := json.Unmarshal(answer.Edits, &edits); err != nil {
		return BridgeFinished{}, fmt.Errorf("decode finished edits: %w", err)
	}
	return BridgeFinished{
		Status: answer.Status, Reason: answer.Reason, Subject: answer.Subject,
		Retained: answer.Retained, Edits: edits,
	}, nil
}

func (b *Bridge) open() error {
	if err := b.conn.Broken(); err != nil {
		return err
	}
	if b.conn.Finished() {
		return ErrReductionFinished
	}
	if !b.admitted {
		return ErrNotAdmitted
	}
	return nil
}

func (b *Bridge) exchange(ctx context.Context, frame bridgeRequest, profile string, check func(bridgeReply) error, kinds ...string) (bridgeReply, error) {
	var answer bridgeReply
	_, err := b.conn.Exchange(ctx, frame.Frame, func(seq int, set string) any {
		frame.Seq, frame.Set = seq, set
		return frame
	}, profile, func(line []byte) error {
		if err := json.Unmarshal(line, &answer); err != nil {
			return fmt.Errorf("decode replay bridge reply to %s: %w", frame.Frame, err)
		}
		if check != nil {
			return check(answer)
		}
		return nil
	}, kinds...)
	return answer, err
}
