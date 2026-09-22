// Package campaign is the Go side of one bounded exploration campaign: the client of the Lean
// exploration bridge (`umpire-explore`), and the serial path that takes one candidate's Case
// through preparation, one Run against a bound deployment, cleanup, and back to the bridge as an
// observation. Go never interprets a target, a Model coordinate or a Case family: it forwards one
// whole Case and returns one closed Run.
package campaign

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// DefaultMaxFrameBytes bounds one frame in either direction. A candidate frame carries a whole
// Case and an observe frame a whole Run; anything larger is a defect, not a bigger Case.
const DefaultMaxFrameBytes = 16 << 20

var (
	// ErrCandidateOutstanding is returned by Next while a candidate's Case is with the caller. It
	// is the client's own guard, raised before any frame is written.
	ErrCandidateOutstanding = errors.New("a candidate is outstanding; observe it before asking for the next")
	// ErrNoCandidate is returned by Observe when no candidate is outstanding.
	ErrNoCandidate = errors.New("no candidate is outstanding")
	// ErrCrossedCandidate is returned by Observe for an identity other than the outstanding one.
	ErrCrossedCandidate = errors.New("the observation names a candidate other than the outstanding one")
	// ErrFrameTooLarge is returned when a frame in either direction exceeds the byte cap.
	ErrFrameTooLarge = errors.New("frame exceeds the byte cap")
	// ErrNotInitialized is returned when a frame is sent before Initialize.
	ErrNotInitialized = errors.New("the campaign is not initialized")
	// ErrFinished is returned once Finish has been answered.
	ErrFinished = errors.New("the campaign is finished")
	// ErrBroken wraps the failure after which the bridge's stream can no longer be trusted: a
	// write or read failure, a reply that did not match its frame, a frame over the cap, or a
	// context that ended mid-exchange. Every later call returns it without writing a frame.
	ErrBroken = errors.New("the bridge is broken")
)

// RejectedError is the bridge's answer to a frame it refused: the campaign is untouched.
type RejectedError struct {
	Seq    int
	Reason string
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("bridge rejected frame %d: %s", e.Seq, e.Reason)
}

// ProtocolError says the bridge's reply did not match the frame sent: a different sequence
// number, set, profile, candidate or kind. The bridge is broken after one.
type ProtocolError struct {
	Expected string
	Actual   string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("bridge reply does not match the frame sent: expected %s, got %s", e.Expected, e.Actual)
}

// Limits are the budget's bounds the campaign searched under, written out by the bridge.
type Limits struct {
	Steps   int `json:"steps"`
	Actions int `json:"actions"`
	Search  int `json:"search"`
}

// Initialized is the bridge's answer to initialize.
type Initialized struct {
	Set     string
	Profile string
	Machine string
	Budget  string
	Limits  Limits
	Targets []string
}

// Skipped is a candidate the bridge passed over as unrealizable under the realization.
type Skipped struct {
	Candidate string `json:"candidate"`
	Target    string `json:"target"`
	Reason    string `json:"reason"`
}

// Candidate is one whole Case the bridge handed out, with the opaque identity the observation
// must echo and the target keys its planned path covers. Go reads the keys and interprets none.
type Candidate struct {
	Identity string
	Target   string
	Covers   []string
	CaseID   string
	Fixture  string
	Case     json.RawMessage
}

// ToolingFailure is the campaign's own defect, which ends it.
type ToolingFailure struct {
	Target string
	Reason string
}

// Next is the bridge's answer to next: exactly one of Candidate, Exhausted or Failure.
type Next struct {
	Candidate *Candidate
	Exhausted bool
	Failure   *ToolingFailure
	Skipped   []Skipped
}

// TargetStatus is one target's ledger status as the bridge renders it.
type TargetStatus struct {
	Target string `json:"target"`
	Status string `json:"status"`
}

// Credited is the bridge's answer to observe: what the observation was read as, and what it
// credited.
type Credited struct {
	Candidate   string
	Observation string
	Detail      string
	Credited    []string
	Statuses    []TargetStatus
}

// Summary is the campaign's counts as the bridge renders them.
type Summary struct {
	Targets      int  `json:"targets"`
	Selected     int  `json:"selected"`
	Covered      int  `json:"covered"`
	Unreachable  int  `json:"unreachable"`
	Violated     int  `json:"violated"`
	Attempted    int  `json:"attempted"`
	Unrealizable int  `json:"unrealizable"`
	Pending      int  `json:"pending"`
	Exhausted    bool `json:"exhausted"`
}

// Counterexample is a violated class-member target with the candidate that violated it.
type Counterexample struct {
	ClassName             string  `json:"className"`
	Target                string  `json:"target"`
	Candidate             string  `json:"candidate"`
	PromotionSourceSHA256 *string `json:"promotionSourceSha256"`
}

// Finished is the bridge's answer to finish.
type Finished struct {
	Status          string
	Summary         Summary
	Counterexamples []Counterexample
	Ledger          []TargetStatus
}

// Result is what an observe frame carries for the outstanding candidate: its closed Run as
// ProtoJSON, or the detail of its preparation rejection.
type Result struct {
	Run             json.RawMessage
	PrepareRejected *string
}

// The frames as the bridge reads and writes them. A request carries only the keys its kind
// admits; a reply is read into one shape and checked by kind.
type request struct {
	Frame           string          `json:"frame"`
	Seq             int             `json:"seq"`
	Set             string          `json:"set"`
	Profile         string          `json:"profile,omitempty"`
	Candidate       string          `json:"candidate,omitempty"`
	Run             json.RawMessage `json:"run,omitempty"`
	PrepareRejected *string         `json:"prepareRejected,omitempty"`
	Status          string          `json:"status,omitempty"`
}

type reply struct {
	Frame           string           `json:"frame"`
	Seq             int              `json:"seq"`
	Set             string           `json:"set"`
	Profile         string           `json:"profile"`
	Reason          string           `json:"reason"`
	Machine         string           `json:"machine"`
	Budget          string           `json:"budget"`
	Limits          Limits           `json:"limits"`
	Targets         []string         `json:"targets"`
	Candidate       string           `json:"candidate"`
	Target          string           `json:"target"`
	Covers          []string         `json:"covers"`
	CaseID          string           `json:"caseId"`
	Fixture         string           `json:"fixture"`
	Skipped         []Skipped        `json:"skipped"`
	Case            json.RawMessage  `json:"case"`
	Observation     string           `json:"observation"`
	Detail          string           `json:"detail"`
	Credited        []string         `json:"credited"`
	Statuses        []TargetStatus   `json:"statuses"`
	Status          string           `json:"status"`
	Summary         Summary          `json:"summary"`
	Counterexamples []Counterexample `json:"counterexamples"`
	Ledger          []TargetStatus   `json:"ledger"`
}

// Bridge is one campaign's client of the bridge process. One request is outstanding at a time,
// every reply is matched to its request by sequence number, set and profile, and a candidate's
// Case is with the caller until Observe returns.
type Bridge struct {
	stdin         io.Writer
	stdout        *bufio.Reader
	closeStdin    func() error
	wait          func() error
	maxFrameBytes int

	set         string
	profile     string
	seq         int
	initialized bool
	finished    bool
	outstanding *Candidate
	broken      error
}

// Options configure a bridge process.
type Options struct {
	// Executable is the bridge binary, `umpire-explore`.
	Executable string
	// Dir is the working directory the bridge runs in.
	Dir string
	// Stderr receives the bridge's progress lines and diagnostics.
	Stderr io.Writer
	// MaxFrameBytes bounds one frame in either direction; zero takes DefaultMaxFrameBytes.
	MaxFrameBytes int
}

// Start spawns the bridge process. Close ends it.
func Start(ctx context.Context, options Options) (*Bridge, error) {
	if options.Executable == "" {
		return nil, errors.New("bridge executable is required")
	}
	command := exec.CommandContext(ctx, options.Executable)
	command.Dir = options.Dir
	command.Stderr = options.Stderr
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("bridge stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("bridge stdout: %w", err)
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start bridge %q: %w", options.Executable, err)
	}
	bridge := New(stdin, stdout, options.MaxFrameBytes)
	bridge.closeStdin = stdin.Close
	bridge.wait = command.Wait
	return bridge, nil
}

// New is a bridge over already-open streams: a test's fake, or a process a caller manages.
func New(stdin io.Writer, stdout io.Reader, maxFrameBytes int) *Bridge {
	if maxFrameBytes <= 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	return &Bridge{
		stdin:         stdin,
		stdout:        bufio.NewReaderSize(stdout, 64<<10),
		closeStdin:    func() error { return nil },
		wait:          func() error { return nil },
		maxFrameBytes: maxFrameBytes,
	}
}

// Outstanding is the candidate whose Case is with the caller, or nil.
func (b *Bridge) Outstanding() *Candidate { return b.outstanding }

// Close ends the bridge: stdin closes, and a spawned process is waited for. A bridge that was
// not finished exits non-zero on the closed stdin; that status is returned.
func (b *Bridge) Close() error {
	return errors.Join(b.closeStdin(), b.wait())
}

// Initialize opens the campaign over one set under one Profile identity.
func (b *Bridge) Initialize(ctx context.Context, set, profile string) (Initialized, error) {
	if b.broken != nil {
		return Initialized{}, b.broken
	}
	if b.initialized {
		return Initialized{}, fmt.Errorf("campaign over %s is already initialized", b.set)
	}
	if set == "" || profile == "" {
		return Initialized{}, errors.New("set and profile identity are required")
	}
	b.set, b.profile = set, profile
	answer, err := b.exchange(ctx, request{Frame: "initialize", Profile: profile}, "initialized")
	if err != nil {
		b.set, b.profile = "", ""
		return Initialized{}, err
	}
	b.initialized = true
	return Initialized{
		Set: answer.Set, Profile: answer.Profile, Machine: answer.Machine, Budget: answer.Budget,
		Limits: answer.Limits, Targets: answer.Targets,
	}, nil
}

// Next asks for the next candidate. It refuses, before writing anything, while one is outstanding.
func (b *Bridge) Next(ctx context.Context) (Next, error) {
	if err := b.open(); err != nil {
		return Next{}, err
	}
	if b.outstanding != nil {
		return Next{}, ErrCandidateOutstanding
	}
	answer, err := b.exchange(ctx, request{Frame: "next"}, "candidate", "exhausted", "toolingFailure")
	if err != nil {
		return Next{}, err
	}
	switch answer.Frame {
	case "candidate":
		if answer.Candidate == "" || len(answer.Case) == 0 || answer.CaseID == "" {
			return Next{}, &ProtocolError{Expected: "a candidate with an identity, a Case ID and a Case", Actual: "an incomplete candidate frame"}
		}
		candidate := &Candidate{
			Identity: answer.Candidate, Target: answer.Target, Covers: answer.Covers,
			CaseID: answer.CaseID, Fixture: answer.Fixture, Case: answer.Case,
		}
		b.outstanding = candidate
		return Next{Candidate: candidate, Skipped: answer.Skipped}, nil
	case "exhausted":
		return Next{Exhausted: true, Skipped: answer.Skipped}, nil
	default:
		return Next{Failure: &ToolingFailure{Target: answer.Target, Reason: answer.Reason}, Skipped: answer.Skipped}, nil
	}
}

// Observe hands back the outstanding candidate's result. It refuses, before writing anything, an
// identity that is not the outstanding candidate's. On a credited answer the candidate is no
// longer outstanding; on a rejection it still is.
func (b *Bridge) Observe(ctx context.Context, identity string, result Result) (Credited, error) {
	if err := b.open(); err != nil {
		return Credited{}, err
	}
	if b.outstanding == nil {
		return Credited{}, ErrNoCandidate
	}
	if identity != b.outstanding.Identity {
		return Credited{}, ErrCrossedCandidate
	}
	if (len(result.Run) == 0) == (result.PrepareRejected == nil) {
		return Credited{}, errors.New("an observation carries exactly one of a Run and a preparation rejection")
	}
	answer, err := b.exchange(ctx, request{
		Frame: "observe", Profile: b.profile, Candidate: identity,
		Run: result.Run, PrepareRejected: result.PrepareRejected,
	}, "credited")
	if err != nil {
		return Credited{}, err
	}
	if answer.Candidate != identity {
		return Credited{}, &ProtocolError{Expected: "candidate " + identity, Actual: "candidate " + answer.Candidate}
	}
	b.outstanding = nil
	return Credited{
		Candidate: answer.Candidate, Observation: answer.Observation, Detail: answer.Detail,
		Credited: answer.Credited, Statuses: answer.Statuses,
	}, nil
}

// Finish asks for the summary. status names the terminal status when the coordinator ended the
// campaign with targets pending ("stopped" or "limit-reached"); empty leaves it to the bridge.
func (b *Bridge) Finish(ctx context.Context, status string) (Finished, error) {
	if err := b.open(); err != nil {
		return Finished{}, err
	}
	answer, err := b.exchange(ctx, request{Frame: "finish", Status: status}, "finished")
	if err != nil {
		return Finished{}, err
	}
	b.finished = true
	return Finished{
		Status: answer.Status, Summary: answer.Summary,
		Counterexamples: answer.Counterexamples, Ledger: answer.Ledger,
	}, nil
}

// Broken is the failure after which the bridge can no longer be trusted, or nil.
func (b *Bridge) Broken() error { return b.broken }

func (b *Bridge) open() error {
	if b.broken != nil {
		return b.broken
	}
	if !b.initialized {
		return ErrNotInitialized
	}
	if b.finished {
		return ErrFinished
	}
	return nil
}

// exchange writes one frame and reads its reply, matched by sequence number, set and profile. A
// `rejected` reply is a RejectedError and leaves the sequence where it was, as the bridge does.
// A frame that could not be written, a reply that could not be read or did not match, or a
// context that ended mid-exchange breaks the bridge: its stream is out of step, so every later
// call returns the same failure without writing.
func (b *Bridge) exchange(ctx context.Context, frame request, kinds ...string) (reply, error) {
	frame.Seq = b.seq + 1
	frame.Set = b.set
	encoded, err := json.Marshal(frame)
	if err != nil {
		return reply{}, fmt.Errorf("encode %s frame: %w", frame.Frame, err)
	}
	if len(encoded)+1 > b.maxFrameBytes {
		return reply{}, fmt.Errorf("%s frame of %d bytes: %w", frame.Frame, len(encoded), ErrFrameTooLarge)
	}
	if err := ctx.Err(); err != nil {
		return reply{}, err
	}
	answer, err := b.transact(ctx, frame, encoded, kinds)
	if err != nil {
		var rejected *RejectedError
		if !errors.As(err, &rejected) {
			b.broken = fmt.Errorf("%w: %w", ErrBroken, err)
		}
		return reply{}, err
	}
	b.seq = frame.Seq
	return answer, nil
}

func (b *Bridge) transact(ctx context.Context, frame request, encoded []byte, kinds []string) (reply, error) {
	if _, err := b.stdin.Write(append(encoded, '\n')); err != nil {
		return reply{}, fmt.Errorf("write %s frame: %w", frame.Frame, err)
	}
	line, err := b.readFrame(ctx)
	if err != nil {
		return reply{}, err
	}
	var answer reply
	if err := json.Unmarshal(line, &answer); err != nil {
		return reply{}, fmt.Errorf("decode bridge reply to %s: %w", frame.Frame, err)
	}
	if answer.Frame == "rejected" {
		return reply{}, &RejectedError{Seq: answer.Seq, Reason: answer.Reason}
	}
	if answer.Seq != frame.Seq {
		return reply{}, &ProtocolError{Expected: fmt.Sprintf("seq %d", frame.Seq), Actual: fmt.Sprintf("seq %d", answer.Seq)}
	}
	if answer.Set != b.set {
		return reply{}, &ProtocolError{Expected: "set " + b.set, Actual: "set " + answer.Set}
	}
	profile := b.profile
	if frame.Frame == "initialize" {
		profile = frame.Profile
	}
	if answer.Profile != profile {
		return reply{}, &ProtocolError{Expected: "profile " + profile, Actual: "profile " + answer.Profile}
	}
	if !contains(kinds, answer.Frame) {
		return reply{}, &ProtocolError{Expected: "one of " + strings.Join(kinds, ", "), Actual: answer.Frame}
	}
	return answer, nil
}

// readFrame reads one line within the byte cap. The read honours the context: a bridge that never
// answers is abandoned at the deadline, which breaks the bridge, and the caller closes it.
func (b *Bridge) readFrame(ctx context.Context) ([]byte, error) {
	type read struct {
		line []byte
		err  error
	}
	done := make(chan read, 1)
	go func() {
		var line bytes.Buffer
		for {
			fragment, isPrefix, err := b.stdout.ReadLine()
			if err != nil {
				done <- read{nil, fmt.Errorf("read bridge frame: %w", err)}
				return
			}
			line.Write(fragment)
			if line.Len() > b.maxFrameBytes {
				done <- read{nil, fmt.Errorf("bridge frame over %d bytes: %w", b.maxFrameBytes, ErrFrameTooLarge)}
				return
			}
			if !isPrefix {
				done <- read{line.Bytes(), nil}
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		return result.line, result.err
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
