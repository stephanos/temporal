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
	"slices"
	"strings"
	"time"
)

// DefaultMaxFrameBytes bounds one frame in either direction. A candidate frame carries a whole
// Case and an observe frame a whole Run; anything larger is a defect, not a bigger Case.
const DefaultMaxFrameBytes = 16 << 20

// CloseTimeout bounds the wait for a bridge process to exit on Close after a finished campaign; a
// process still running past it, like one that was never finished or is broken, is killed.
const CloseTimeout = 10 * time.Second

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
	// PromotionSourcePath is the file the bridge names the compiled proposal at, relative to
	// wherever the caller chooses to write it, and PromotionSource is its bytes; both are absent
	// when the proposal did not compile, and PromotionError then says why. Go writes the bytes
	// where it is told and never under the model.
	PromotionSourcePath string `json:"promotionSourcePath,omitempty"`
	PromotionSource     string `json:"promotionSource,omitempty"`
	PromotionError      string `json:"promotionError,omitempty"`
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

// Conn is the frame transport every client of a Lean bridge shares: one process (or a pair of
// streams), one request outstanding, every reply matched to its request by sequence number, set
// and profile, a byte cap in either direction, and a broken state after which nothing is written.
// What the frames mean is the client's.
type Conn struct {
	stdin         io.Writer
	stdout        *bufio.Reader
	closeStdin    func() error
	wait          func() error
	kill          func() error
	maxFrameBytes int

	set      string
	seq      int
	finished bool
	broken   error
}

// Envelope is what every reply carries, whatever its kind.
type Envelope struct {
	Frame   string `json:"frame"`
	Seq     int    `json:"seq"`
	Set     string `json:"set"`
	Profile string `json:"profile"`
	Reason  string `json:"reason"`
}

// Bridge is one campaign's client of the bridge process. One request is outstanding at a time,
// every reply is matched to its request by sequence number, set and profile, and a candidate's
// Case is with the caller until Observe returns.
type Bridge struct {
	conn *Conn

	profile     string
	initialized bool
	outstanding *Candidate
}

// Options configure a bridge process.
type Options struct {
	// Executable is the bridge binary, `umpire-explore`.
	Executable string
	// Args are the executable's arguments; the bridge takes none, a test's stand-in may.
	Args []string
	// Dir is the working directory the bridge runs in.
	Dir string
	// Stderr receives the bridge's progress lines and diagnostics.
	Stderr io.Writer
	// MaxFrameBytes bounds one frame in either direction; zero takes DefaultMaxFrameBytes.
	MaxFrameBytes int
}

// StartConn spawns a bridge process in a process group of its own. Close ends it. The context
// bounds the process's life: a caller whose work may be cancelled while the bridge's last answer
// is still wanted starts it on a context that outlives the work and lets Close end it.
func StartConn(ctx context.Context, options Options) (*Conn, error) {
	if options.Executable == "" {
		return nil, errors.New("bridge executable is required")
	}
	command := exec.CommandContext(ctx, options.Executable, options.Args...)
	detach(command)
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
	conn := NewConn(stdin, stdout, options.MaxFrameBytes)
	conn.closeStdin = stdin.Close
	conn.wait = command.Wait
	conn.kill = func() error {
		if command.Process == nil {
			return nil
		}
		return command.Process.Kill()
	}
	return conn, nil
}

// NewConn is a transport over already-open streams: a test's fake, or a process a caller manages.
func NewConn(stdin io.Writer, stdout io.Reader, maxFrameBytes int) *Conn {
	if maxFrameBytes <= 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	return &Conn{
		stdin:         stdin,
		stdout:        bufio.NewReaderSize(stdout, 64<<10),
		closeStdin:    func() error { return nil },
		wait:          func() error { return nil },
		kill:          func() error { return nil },
		maxFrameBytes: maxFrameBytes,
	}
}

// Start spawns the exploration bridge process. Close ends it.
func Start(ctx context.Context, options Options) (*Bridge, error) {
	conn, err := StartConn(ctx, options)
	if err != nil {
		return nil, err
	}
	return &Bridge{conn: conn}, nil
}

// New is an exploration bridge over already-open streams.
func New(stdin io.Writer, stdout io.Reader, maxFrameBytes int) *Bridge {
	return &Bridge{conn: NewConn(stdin, stdout, maxFrameBytes)}
}

// Outstanding is the candidate whose Case is with the caller, or nil.
func (b *Bridge) Outstanding() *Candidate { return b.outstanding }

// Close ends the bridge process.
func (b *Bridge) Close() error { return b.conn.Close() }

// Close ends the bridge process. A finished bridge is given its stdin's EOF and CloseTimeout to
// exit; a bridge that is broken or was never finished may be stuck in a search and not reading
// stdin, so it is killed and then waited for. A process that had to be killed reports its exit
// status.
func (c *Conn) Close() error {
	closeErr := c.closeStdin()
	if c.broken != nil || !c.finished {
		return errors.Join(closeErr, c.kill(), c.wait())
	}
	waited := make(chan error, 1)
	go func() { waited <- c.wait() }()
	select {
	case err := <-waited:
		return errors.Join(closeErr, err)
	case <-time.After(CloseTimeout):
		return errors.Join(closeErr, c.kill(), <-waited)
	}
}

// Scope names the set every later frame carries; empty clears it.
func (c *Conn) Scope(set string) { c.set = set }

// Set is the set the transport's frames carry.
func (c *Conn) Set() string { return c.set }

// Finish marks the protocol complete: the bridge exits on its own, and nothing more is sent.
func (c *Conn) Finish() { c.finished = true }

// Finished says the protocol is complete.
func (c *Conn) Finished() bool { return c.finished }

// Broken is the failure after which the transport can no longer be trusted, or nil.
func (c *Conn) Broken() error { return c.broken }

// Initialize opens the campaign over one set under one Profile identity.
func (b *Bridge) Initialize(ctx context.Context, set, profile string) (Initialized, error) {
	if err := b.conn.Broken(); err != nil {
		return Initialized{}, err
	}
	if b.initialized {
		return Initialized{}, fmt.Errorf("campaign over %s is already initialized", b.conn.Set())
	}
	if set == "" || profile == "" {
		return Initialized{}, errors.New("set and profile identity are required")
	}
	b.conn.Scope(set)
	answer, err := b.exchange(ctx, request{Frame: "initialize", Profile: profile}, profile, nil, "initialized")
	if err != nil {
		b.conn.Scope("")
		return Initialized{}, err
	}
	b.profile = profile
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
	whole := func(answer reply) error {
		if answer.Frame == "candidate" && (answer.Candidate == "" || len(answer.Case) == 0 || answer.CaseID == "") {
			return &ProtocolError{Expected: "a candidate with an identity, a Case ID and a Case", Actual: "an incomplete candidate frame"}
		}
		return nil
	}
	answer, err := b.exchange(ctx, request{Frame: "next"}, b.profile, whole, "candidate", "exhausted", "toolingFailure")
	if err != nil {
		return Next{}, err
	}
	switch answer.Frame {
	case "candidate":
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
	same := func(answer reply) error {
		if answer.Candidate != identity {
			return &ProtocolError{Expected: "candidate " + identity, Actual: "candidate " + answer.Candidate}
		}
		return nil
	}
	answer, err := b.exchange(ctx, request{
		Frame: "observe", Profile: b.profile, Candidate: identity,
		Run: result.Run, PrepareRejected: result.PrepareRejected,
	}, b.profile, same, "credited")
	if err != nil {
		return Credited{}, err
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
	answer, err := b.exchange(ctx, request{Frame: "finish", Status: status}, b.profile, nil, "finished")
	if err != nil {
		return Finished{}, err
	}
	b.conn.Finish()
	return Finished{
		Status: answer.Status, Summary: answer.Summary,
		Counterexamples: answer.Counterexamples, Ledger: answer.Ledger,
	}, nil
}

// Broken is the failure after which the bridge can no longer be trusted, or nil.
func (b *Bridge) Broken() error { return b.conn.Broken() }

func (b *Bridge) open() error {
	if err := b.conn.Broken(); err != nil {
		return err
	}
	if !b.initialized {
		return ErrNotInitialized
	}
	if b.conn.Finished() {
		return ErrFinished
	}
	return nil
}

// exchange sends one exploration frame over the transport and reads its reply into the
// exploration shape, passing the kind's own check.
func (b *Bridge) exchange(ctx context.Context, frame request, profile string, check func(reply) error, kinds ...string) (reply, error) {
	var answer reply
	_, err := b.conn.Exchange(ctx, frame.Frame, func(seq int, set string) any {
		frame.Seq, frame.Set = seq, set
		return frame
	}, profile, func(line []byte) error {
		if err := json.Unmarshal(line, &answer); err != nil {
			return fmt.Errorf("decode bridge reply to %s: %w", frame.Frame, err)
		}
		if check != nil {
			return check(answer)
		}
		return nil
	}, kinds...)
	return answer, err
}

// Exchange writes the frame build returns for the next sequence number and the scoped set, and
// reads its reply, matched by sequence number, set and profile, of one of the kinds named, and
// passing check. A `rejected` reply is a RejectedError and leaves the sequence where it was, as a
// bridge does. A frame that could not be written, a reply that could not be read or did not match,
// or a context that ended mid-exchange breaks the transport: its stream is out of step, so every
// later call returns the same failure without writing.
func (c *Conn) Exchange(ctx context.Context, kind string, build func(seq int, set string) any, profile string,
	check func(line []byte) error, kinds ...string) ([]byte, error) {
	if c.broken != nil {
		return nil, c.broken
	}
	seq := c.seq + 1
	encoded, err := json.Marshal(build(seq, c.set))
	if err != nil {
		return nil, fmt.Errorf("encode %s frame: %w", kind, err)
	}
	if len(encoded)+1 > c.maxFrameBytes {
		return nil, fmt.Errorf("%s frame of %d bytes: %w", kind, len(encoded), ErrFrameTooLarge)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	line, err := c.transact(ctx, kind, seq, profile, encoded, kinds, check)
	if err != nil {
		var rejected *RejectedError
		if !errors.As(err, &rejected) {
			c.broken = fmt.Errorf("%w: %w", ErrBroken, err)
		}
		return nil, err
	}
	c.seq = seq
	return line, nil
}

func (c *Conn) transact(ctx context.Context, kind string, seq int, profile string, encoded []byte, kinds []string,
	check func(line []byte) error) ([]byte, error) {
	line, err := c.roundTrip(ctx, kind, append(encoded, '\n'))
	if err != nil {
		return nil, err
	}
	var answer Envelope
	if err := json.Unmarshal(line, &answer); err != nil {
		return nil, fmt.Errorf("decode bridge reply to %s: %w", kind, err)
	}
	if answer.Seq != seq {
		return nil, &ProtocolError{Expected: fmt.Sprintf("seq %d", seq), Actual: fmt.Sprintf("seq %d", answer.Seq)}
	}
	if answer.Frame == "rejected" {
		return nil, &RejectedError{Seq: answer.Seq, Reason: answer.Reason}
	}
	if answer.Set != c.set {
		return nil, &ProtocolError{Expected: "set " + c.set, Actual: "set " + answer.Set}
	}
	if answer.Profile != profile {
		return nil, &ProtocolError{Expected: "profile " + profile, Actual: "profile " + answer.Profile}
	}
	if !slices.Contains(kinds, answer.Frame) {
		return nil, &ProtocolError{Expected: "one of " + strings.Join(kinds, ", "), Actual: answer.Frame}
	}
	if check != nil {
		if err := check(line); err != nil {
			return nil, err
		}
	}
	return line, nil
}

// roundTrip writes one frame and reads its reply within the byte cap. Both honour the context: a
// bridge that neither reads nor answers is abandoned at the deadline, which breaks the transport,
// and Close kills it.
func (c *Conn) roundTrip(ctx context.Context, kind string, encoded []byte) ([]byte, error) {
	type read struct {
		line []byte
		err  error
	}
	done := make(chan read, 1)
	go func() {
		if _, err := c.stdin.Write(encoded); err != nil {
			done <- read{nil, fmt.Errorf("write %s frame: %w", kind, err)}
			return
		}
		var line bytes.Buffer
		for {
			fragment, isPrefix, err := c.stdout.ReadLine()
			if err != nil {
				done <- read{nil, fmt.Errorf("read bridge frame: %w", err)}
				return
			}
			line.Write(fragment)
			if line.Len() > c.maxFrameBytes {
				done <- read{nil, fmt.Errorf("bridge frame over %d bytes: %w", c.maxFrameBytes, ErrFrameTooLarge)}
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
