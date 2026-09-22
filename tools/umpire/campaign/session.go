package campaign

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Status is the terminal status of a campaign. Exactly one of the four ends it.
type Status string

const (
	// StatusExhausted: no target is pending; each is covered, unreachable, violated, attempted or
	// unrealizable.
	StatusExhausted Status = "exhausted"
	// StatusLimitReached: a campaign counter tripped before the action it bounds.
	StatusLimitReached Status = "limit-reached"
	// StatusStopped: the coordinator was stopped, by a signal or its own deadline.
	StatusStopped Status = "stopped"
	// StatusToolingFailure: a loader, bridge, binding, admission or invariant failure.
	StatusToolingFailure Status = "tooling-failure"
)

// State is where the coordinator stands. Every transition consumes the state it starts from, so
// two outstanding candidates cannot be represented: there is one State value, and it is in one
// place.
type State string

const (
	// StateIdle: no candidate is outstanding and the next may be planned.
	StateIdle State = "idle"
	// StatePlanning: the bridge was asked for the next candidate.
	StatePlanning State = "planning"
	// StatePreparing: a candidate's Case is being bound and prepared; no Run exists.
	StatePreparing State = "preparing"
	// StateRunning: one Run is in flight.
	StateRunning State = "running"
	// StateObserving: the closed Run is being handed back to the bridge.
	StateObserving State = "observing"
	// StateFinished: the campaign ended; only the summary remains.
	StateFinished State = "finished"
)

// Caps are the campaign's own counters, declared once and enforced before the action each one
// bounds. Zero leaves a cap unset. The budget's `search` limit bounds one Search inside the
// bridge and is never a campaign cap; per-Case static work is the Profile's admission limits.
type Caps struct {
	// Candidates bounds how many candidates are planned, checked before each `next`.
	Candidates int
	// CaseBytes bounds the aggregate bytes of the Cases handed out, checked before each `next`
	// against what was handed out so far and again on the candidate that arrives.
	CaseBytes int64
	// RunEvents bounds the aggregate Run Events observed, checked before each `next`.
	RunEvents int64
	// ReportBytes bounds the summary, checked on the rendered report.
	ReportBytes int64
	// RunTimeout bounds one Run, applied to its context before the Run opens. A Run that reaches
	// it closes as the facade closes an interrupted Run and is observed as that Run, which the
	// bridge reads as inconclusive; it is per-Run work, not a campaign counter, so it never ends
	// the campaign as limit-reached. A timeout that fires before the facade opened the Driver
	// returns no Run at all, which is run-failed and ends the campaign as tooling-failure, since
	// nothing honest can be observed for it.
	RunTimeout time.Duration
}

// LimitError is a cap that tripped on what was measured against it.
type LimitError struct {
	Limit    string
	Measured int64
	Cap      int64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("%s: %s of %d exceeds the cap of %d", StatusLimitReached, e.Limit, e.Measured, e.Cap)
}

// Counters are what the campaign has consumed. With the per-candidate summaries the report keeps
// (identity, kind, observation, never a Run), they are the coordinator's retained state, whatever
// the volume.
type Counters struct {
	Planned      int   `json:"planned"`
	Prepared     int   `json:"prepared"`
	Started      int   `json:"started"`
	Decisive     int   `json:"decisive"`
	Rejected     int   `json:"rejected"`
	Failed       int   `json:"failed"`
	Inconclusive int   `json:"inconclusive"`
	Skipped      int   `json:"skipped"`
	CaseBytes    int64 `json:"caseBytes"`
	RunEvents    int64 `json:"runEvents"`
}

// Terminal is how the campaign ended.
type Terminal struct {
	Status Status
	// Limit names the cap that tripped, for limit-reached.
	Limit string
	// Failure names the defect, for tooling-failure.
	Failure string
	// Lost is the identity of the candidate whose Run was in flight when the campaign stopped,
	// for stopped; empty when the stop fell between candidates.
	Lost string
}

// ErrConsumed is returned by a transition on a state value that already transitioned.
var ErrConsumed = errors.New("this coordinator state was already consumed")

// Session is the process-local coordinator state: one State, the caps, the counters, the
// outstanding candidate and, once finished, the terminal. It never recovers, resumes or persists.
type Session struct {
	state       State
	caps        Caps
	counters    Counters
	outstanding *Candidate
	// ran says a Run was opened for the outstanding candidate and not yet credited, so a stop
	// now loses it whether the Run is in flight or closed and unobserved.
	ran      bool
	terminal *Terminal
	consumed bool
}

// NewSession is a coordinator in StateIdle.
func NewSession(caps Caps) *Session {
	return &Session{state: StateIdle, caps: caps}
}

func (s *Session) State() State       { return s.state }
func (s *Session) Counters() Counters { return s.counters }
func (s *Session) Outstanding() *Candidate {
	return s.outstanding
}

// Terminal is how the campaign ended, or nil while it runs.
func (s *Session) Terminal() *Terminal { return s.terminal }

// transition consumes the current state for one of the states named. The consumed session is
// returned as the new one, so a caller that kept the old value finds it spent.
func (s *Session) transition(from []State, to State) (*Session, error) {
	if s.consumed {
		return nil, ErrConsumed
	}
	found := false
	for _, admitted := range from {
		if s.state == admitted {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("coordinator is %s, not %v", s.state, from)
	}
	next := *s
	s.consumed = true
	next.consumed = false
	next.state = to
	return &next, nil
}

func (s *Session) finish(terminal Terminal) (*Session, error) {
	next, err := s.transition([]State{StateIdle, StatePlanning, StatePreparing, StateRunning, StateObserving}, StateFinished)
	if err != nil {
		return nil, err
	}
	next.terminal = &terminal
	return next, nil
}

// Plan moves idle to planning, or to finished with limit-reached when a cap the next candidate
// would exceed has tripped. The caps are checked here, before the bridge is asked for anything,
// and only from idle: a tripped cap never ends a campaign with a candidate outstanding.
func (s *Session) Plan() (*Session, error) {
	if s.consumed {
		return nil, ErrConsumed
	}
	if s.state != StateIdle {
		return nil, fmt.Errorf("coordinator is %s, not idle", s.state)
	}
	if limit := s.tripped(); limit != "" {
		return s.finish(Terminal{Status: StatusLimitReached, Limit: limit})
	}
	return s.transition([]State{StateIdle}, StatePlanning)
}

// tripped names the cap the next candidate would exceed, or "".
func (s *Session) tripped() string {
	switch {
	case s.caps.Candidates > 0 && s.counters.Planned >= s.caps.Candidates:
		return "candidates"
	case s.caps.CaseBytes > 0 && s.counters.CaseBytes >= s.caps.CaseBytes:
		return "case-bytes"
	case s.caps.RunEvents > 0 && s.counters.RunEvents >= s.caps.RunEvents:
		return "run-events"
	}
	return ""
}

// Planned takes the bridge's answer to next: a candidate moves planning to preparing, exhaustion
// and a tooling failure finish the campaign. A candidate whose Case would push the aggregate over
// the cap finishes it as limit-reached; that Case is never bound.
func (s *Session) Planned(next Next) (*Session, error) {
	if s.consumed {
		return nil, ErrConsumed
	}
	if s.state != StatePlanning {
		return nil, fmt.Errorf("coordinator is %s, not planning", s.state)
	}
	skipped := len(next.Skipped)
	ended := func(terminal Terminal) (*Session, error) {
		after, err := s.finish(terminal)
		if err != nil {
			return nil, err
		}
		after.counters.Skipped += skipped
		return after, nil
	}
	switch {
	case next.Failure != nil:
		return ended(Terminal{Status: StatusToolingFailure, Failure: next.Failure.Target + ": " + next.Failure.Reason})
	case next.Exhausted:
		return ended(Terminal{Status: StatusExhausted})
	case next.Candidate == nil:
		return ended(Terminal{Status: StatusToolingFailure, Failure: "the bridge answered next with neither a candidate nor exhaustion"})
	}
	bytes := int64(len(next.Candidate.Case))
	if s.caps.CaseBytes > 0 && s.counters.CaseBytes+bytes > s.caps.CaseBytes {
		return ended(Terminal{Status: StatusLimitReached, Limit: "case-bytes"})
	}
	after, err := s.transition([]State{StatePlanning}, StatePreparing)
	if err != nil {
		return nil, err
	}
	after.counters.Skipped += skipped
	after.counters.Planned++
	after.counters.CaseBytes += bytes
	after.outstanding = next.Candidate
	after.ran = false
	return after, nil
}

// Prepared moves preparing to running once the candidate's Case is bound and prepared; one Run
// may now open, under RunTimeout, and a stop from here on loses this candidate.
func (s *Session) Prepared() (*Session, error) {
	after, err := s.transition([]State{StatePreparing}, StateRunning)
	if err != nil {
		return nil, err
	}
	after.counters.Prepared++
	after.ran = true
	return after, nil
}

// Rejected takes a preparation rejection: no Run exists, the bridge is told next, so the state
// moves to observing.
func (s *Session) Rejected() (*Session, error) {
	after, err := s.transition([]State{StatePreparing}, StateObserving)
	if err != nil {
		return nil, err
	}
	after.counters.Rejected++
	return after, nil
}

// Ran moves running to observing once the Run closed with its cleanup observed: a started Run,
// whose runEvents are counted against the cap before the next candidate. A prepared candidate
// whose Run never came back is prepared and not started.
func (s *Session) Ran(runEvents int) (*Session, error) {
	after, err := s.transition([]State{StateRunning}, StateObserving)
	if err != nil {
		return nil, err
	}
	after.counters.Started++
	after.counters.RunEvents += int64(runEvents)
	return after, nil
}

// Observed takes what the bridge credited and returns to idle. A decisive observation is counted
// as such; any other counts as inconclusive.
func (s *Session) Observed(credited Credited) (*Session, error) {
	after, err := s.transition([]State{StateObserving}, StateIdle)
	if err != nil {
		return nil, err
	}
	switch credited.Observation {
	case "satisfied", "violated":
		after.counters.Decisive++
	case "prepare-rejected":
	default:
		after.counters.Inconclusive++
	}
	after.outstanding = nil
	after.ran = false
	return after, nil
}

// Failed ends the campaign on a defect: a binding or Driver failure that is not the Case's own, a
// Run that could not execute or be observed, a bridge that broke. Nothing is synthesized for the
// outstanding candidate.
func (s *Session) Failed(failure string) (*Session, error) {
	after, err := s.finish(Terminal{Status: StatusToolingFailure, Failure: failure})
	if err != nil {
		return nil, err
	}
	after.counters.Failed++
	return after, nil
}

// Stopped ends the campaign on a stop. A candidate whose Run was opened and not credited -- in
// flight, or closed by the interruption and never observed -- is the lost iteration, named by its
// identity; a stop before any Run opened, or between candidates, loses none. No Verdict or
// coverage is made up for a lost iteration.
func (s *Session) Stopped() (*Session, error) {
	terminal := Terminal{Status: StatusStopped}
	if s.ran && s.outstanding != nil {
		terminal.Lost = s.outstanding.Identity
	}
	return s.finish(terminal)
}

// CheckReport enforces the report cap on the rendered summary: a report over the cap is
// limit-reached, as a *LimitError, never truncated.
func (s *Session) CheckReport(rendered int) error {
	if s.caps.ReportBytes > 0 && int64(rendered) > s.caps.ReportBytes {
		return &LimitError{Limit: "report-bytes", Measured: int64(rendered), Cap: s.caps.ReportBytes}
	}
	return nil
}

// RunContext bounds one Run by RunTimeout, applied before the Run opens.
func (s *Session) RunContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.caps.RunTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.caps.RunTimeout)
}

// Report is what one campaign came to: the terminal, the counters, and the bridge's summary
// when the bridge could still be asked for one.
type Report struct {
	Terminal Terminal
	Counters Counters
	Finished *Finished
	// Outcomes summarize the candidates in order; no Run is retained.
	Outcomes []OutcomeSummary
}

// OutcomeSummary is what the report keeps of one candidate: its identity and target, how far it
// got, and what the bridge read its result as.
type OutcomeSummary struct {
	Identity    string
	Target      string
	Kind        OutcomeKind
	Detail      string
	Observation string
	Credited    []string
}

func summarize(candidate *Candidate, outcome Outcome) OutcomeSummary {
	summary := OutcomeSummary{Identity: candidate.Identity, Target: candidate.Target, Kind: outcome.Kind, Detail: outcome.Detail}
	if outcome.Credited != nil {
		summary.Observation = outcome.Credited.Observation
		summary.Credited = outcome.Credited.Credited
	}
	return summary
}

// stepper moves the session as the serial path moves one candidate.
type stepper struct{ session *Session }

func (p *stepper) rejected() error { return p.move((*Session).Rejected) }
func (p *stepper) prepared() error { return p.move((*Session).Prepared) }
func (p *stepper) ran(runEvents int) error {
	return p.move(func(s *Session) (*Session, error) { return s.Ran(runEvents) })
}
func (p *stepper) observed(credited Credited) error {
	return p.move(func(s *Session) (*Session, error) { return s.Observed(credited) })
}
func (p *stepper) runContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return p.session.RunContext(ctx)
}
func (p *stepper) move(transition func(*Session) (*Session, error)) error {
	next, err := transition(p.session)
	if err != nil {
		return err
	}
	p.session = next
	return nil
}

// Drive runs one campaign to its terminal: plan, prepare, run, observe, one candidate at a time,
// under the caps, until the bridge is exhausted, a cap trips, the context ends, or something
// fails. Progress goes to `progress`, one line per candidate as it is selected and as it comes to
// its outcome. The bridge is initialized by the caller and finished here, when it can still be
// asked. The report's Terminal is the authoritative outcome, whatever the bridge answers; the
// error carries the cause only when something struck the coordinator itself -- a binding, Run,
// bridge or context failure -- and is nil when the campaign ended by what the bridge reported,
// exhaustion or a tooling failure of the bridge's own.
func Drive(ctx context.Context, bridge *Bridge, binder Binder, caps Caps, progress io.Writer) (Report, error) {
	current := &stepper{session: NewSession(caps)}
	var outcomes []OutcomeSummary
	var driveErr error
	for current.session.State() != StateFinished {
		if err := ctx.Err(); err != nil {
			driveErr = current.end(ctx, err)
			break
		}
		if err := current.move((*Session).Plan); err != nil {
			return Report{}, err
		}
		if current.session.State() == StateFinished {
			break
		}
		next, err := bridge.Next(ctx)
		if err != nil {
			driveErr = current.end(ctx, fmt.Errorf("next: %w", err))
			break
		}
		for _, skipped := range next.Skipped {
			writeProgress(progress, "skipped %s %s %s", skipped.Candidate, skipped.Target, skipped.Reason)
		}
		if err := current.move(func(s *Session) (*Session, error) { return s.Planned(next) }); err != nil {
			return Report{}, err
		}
		if current.session.State() == StateFinished {
			break
		}
		candidate := next.Candidate
		writeProgress(progress, "candidate %s %s", candidate.Identity, candidate.Target)
		outcome, err := runCandidate(ctx, bridge, binder, candidate, current)
		outcomes = append(outcomes, summarize(candidate, outcome))
		if err != nil {
			driveErr = current.end(ctx, err)
			writeProgress(progress, "candidate %s %s", candidate.Identity, current.session.Terminal().Status)
			break
		}
		writeProgress(progress, "candidate %s %s", candidate.Identity, outcomeSummary(outcome))
	}
	report := Report{Terminal: *current.session.Terminal(), Counters: current.session.Counters(), Outcomes: outcomes}
	if lost := report.Terminal.Lost; lost != "" {
		for index := range report.Outcomes {
			if report.Outcomes[index].Identity == lost {
				report.Outcomes[index].Kind = OutcomeLost
			}
		}
	}
	// The summary is asked for on a context of its own, bounded like Close: a stopped campaign's
	// context has ended, and its summary is still the bridge's to give.
	if bridge.Broken() == nil {
		status := ""
		if report.Terminal.Status == StatusStopped || report.Terminal.Status == StatusLimitReached {
			status = string(report.Terminal.Status)
		}
		finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), CloseTimeout)
		finished, err := bridge.Finish(finishCtx, status)
		cancel()
		if err != nil {
			return report, errors.Join(driveErr, fmt.Errorf("finish: %w", err))
		}
		report.Finished = &finished
	}
	return report, driveErr
}

// end finishes the session on the error that struck: a context that ended is a stop (a Run that
// was opened and not credited is the lost iteration), anything else a tooling failure. The error
// returned is the one that struck, so a caller can still tell a broken bridge from a rejected
// frame from a Run's deadline.
func (p *stepper) end(ctx context.Context, cause error) error {
	if ctx.Err() != nil {
		if err := p.move((*Session).Stopped); err != nil {
			return err
		}
		return cause
	}
	if err := p.move(func(s *Session) (*Session, error) { return s.Failed(cause.Error()) }); err != nil {
		return err
	}
	return cause
}

func outcomeSummary(outcome Outcome) string {
	if outcome.Credited != nil {
		return outcome.Credited.Observation
	}
	return string(outcome.Kind)
}

func writeProgress(progress io.Writer, format string, arguments ...any) {
	if progress == nil {
		return
	}
	_, _ = fmt.Fprintf(progress, format+"\n", arguments...)
}
