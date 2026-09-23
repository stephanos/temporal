package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// Limits bound one reduction. Each is checked before the work it bounds: the edit cap at
// admission, the Run budget before a candidate is asked for and before a retry, the wall time
// before a candidate is asked for, the aggregate Case bytes and Run Events before a candidate is
// asked for and the Case bytes again on the candidate that arrives, and the report bytes on the
// rendered report. Zero leaves a cap unset, except Runs and Edits, which the first slice fixes.
type Limits struct {
	Edits       int
	Runs        int
	WallTime    time.Duration
	CaseBytes   int64
	RunEvents   int64
	ReportBytes int64
}

// DefaultLimits are the first vertical slice's: eight edits, twelve fresh Runs in all (two for the
// subject, two per candidate and one per retry), 25 minutes, and byte and event caps generous for
// one Case family and small for a defect.
var DefaultLimits = Limits{
	Edits:       8,
	Runs:        12,
	WallTime:    25 * time.Minute,
	CaseBytes:   64 << 20,
	RunEvents:   1 << 20,
	ReportBytes: 1 << 20,
}

// The limits a reduction names when one ends it.
const (
	LimitEdits       = "edits"
	LimitRuns        = "runs"
	LimitWallTime    = "wall-time"
	LimitCaseBytes   = "case-bytes"
	LimitRunEvents   = "run-events"
	LimitReportBytes = "report-bytes"
)

// CandidateReport is what one candidate came to: its edit, its digest and Case checksum, each
// Run's class in the order the Runs closed (a retry appended), the pair's class after the retry,
// and the fate the bridge settled.
type CandidateReport struct {
	Edit     Edit    `json:"edit"`
	Digest   string  `json:"digest"`
	Identity string  `json:"identity"`
	Classes  []Class `json:"classes"`
	Class    Class   `json:"class"`
	Fate     string  `json:"fate"`
	Reason   string  `json:"reason"`
}

// Reduction is one reduction's report: whether it was attempted and, if not, why; the bridge's
// result and every edit's fate; the limit, the stop or the failure that ended it early; the Run
// lost to a stop; the digests of the subject and of what was retained; the Runs spent; and each
// candidate's classes. It holds no time, so the same inputs and classes render the same bytes.
type Reduction struct {
	Attempted    bool              `json:"attempted"`
	NotAttempted string            `json:"notAttempted,omitempty"`
	Status       string            `json:"status"`
	Reason       string            `json:"reason"`
	Limit        string            `json:"limit,omitempty"`
	Stopped      bool              `json:"stopped"`
	Failure      string            `json:"failure,omitempty"`
	Lost         string            `json:"lost,omitempty"`
	Subject      string            `json:"subject"`
	Retained     string            `json:"retained"`
	Runs         int               `json:"runs"`
	Edits        []Settled         `json:"edits"`
	Candidates   []CandidateReport `json:"candidates"`
}

// ReductionNotAttempted is the status of a reduction that did not start.
const ReductionNotAttempted = "not-attempted"

// Reducer is one reduction over an admitted bridge: the subject, the binder its reruns bind
// through, the preparer that prepares a candidate under the subject's Profile name, and the limits.
type Reducer struct {
	Bridge   *Bridge
	Admitted Admitted
	Binder   campaign.Binder
	Prepare  Preparer
	Subject  *Subject
	Limits   Limits
	// Now reads the clock the wall time is measured on; nil reads time.Now.
	Now func() time.Time
	// Progress receives one line per decision; nil discards them.
	Progress io.Writer
}

// reduceState is where one reduction stands; each step moves it forward and none moves it back.
type reduceState int

const (
	stateAsking reduceState = iota
	stateRunning
	stateDone
)

type minimizer struct {
	Reducer
	state     reduceState
	deadline  time.Time
	caseBytes int64
	runEvents int64
	report    Reduction
}

// Reduce runs one sweep. It starts only when the subject's reruns reproduced its key; otherwise
// the report says the reduction was not attempted, and the bridge is finished as stopped. For
// each candidate the bridge hands out, it prepares the Case under the subject's Profile name (a
// rejection is reported as `rejected`, never rerun), reruns it twice through Rerun, reruns an
// indeterminate Run alone once, and reports the pair's class. It ends when the bridge's sweep is
// exhausted, an edit is undecided, a limit is reached, the context is cancelled, or a rerun
// cannot bind or release; the bridge's result says which edits were retained. A returned error is
// one the bridge could not be told about; the report beside it holds what was decided.
func (r Reducer) Reduce(ctx context.Context, subjectReruns *Reruns) (Reduction, error) {
	if r.Bridge == nil || r.Binder == nil || r.Prepare == nil || r.Subject == nil || subjectReruns == nil {
		return Reduction{}, errors.New("a bridge, a binder, a preparer, a subject and its reruns are required")
	}
	now := r.Now
	if now == nil {
		now = time.Now
	}
	m := &minimizer{Reducer: r, deadline: now().Add(r.Limits.WallTime)}
	m.Now = now
	m.report = Reduction{
		Subject: r.Admitted.Subject, Retained: r.Admitted.Subject,
		Runs: len(subjectReruns.Attempts), Edits: []Settled{}, Candidates: []CandidateReport{},
	}
	for _, attempt := range subjectReruns.Attempts {
		m.runEvents += int64(len(attempt.Run.GetEvents()))
	}
	var err error
	switch {
	case subjectReruns.Class != ClassReproduced:
		m.report.NotAttempted = fmt.Sprintf("the subject's reruns were %s", subjectReruns.Class)
		m.progress("not attempted: %s", m.report.NotAttempted)
		err = m.finish(ctx, "stopped")
	case r.Limits.Edits > 0 && len(r.Admitted.Edits) > r.Limits.Edits:
		m.report.Attempted = true
		err = m.limit(ctx, LimitEdits)
	default:
		m.report.Attempted = true
		for m.state == stateAsking {
			var done bool
			if done, err = m.ask(ctx); done {
				break
			}
		}
	}
	return m.report, err
}

func (m *minimizer) progress(format string, arguments ...any) {
	if m.Progress != nil {
		cli.WriteLine(m.Progress, format, arguments...)
	}
}

// ask checks every limit that bounds a candidate, asks the bridge for one and takes it through
// its Runs. done says the reduction ended.
func (m *minimizer) ask(ctx context.Context) (bool, error) {
	if ctx.Err() != nil {
		m.report.Stopped = true
		return true, m.finish(ctx, "stopped")
	}
	switch {
	case m.Limits.WallTime > 0 && !m.Now().Before(m.deadline):
		return true, m.limit(ctx, LimitWallTime)
	case m.Limits.Runs > 0 && m.report.Runs+Attempts > m.Limits.Runs:
		return true, m.limit(ctx, LimitRuns)
	case m.Limits.CaseBytes > 0 && m.caseBytes >= m.Limits.CaseBytes:
		return true, m.limit(ctx, LimitCaseBytes)
	case m.Limits.RunEvents > 0 && m.runEvents >= m.Limits.RunEvents:
		return true, m.limit(ctx, LimitRunEvents)
	default:
	}
	next, err := m.Bridge.Next(ctx)
	if err != nil {
		m.state = stateDone
		return true, fmt.Errorf("next: %w", err)
	}
	for _, skipped := range next.Skipped {
		m.progress("skipped %s %s %s", skipped.Edit.Edit, skipped.Fate, skipped.Reason)
	}
	if next.Exhausted {
		return true, m.finish(ctx, "")
	}
	m.state = stateRunning
	if done, err := m.candidate(ctx, next.Candidate); done {
		return true, err
	}
	m.state = stateAsking
	return false, nil
}

// candidate takes one candidate through preparation, its Runs and the bridge.
func (m *minimizer) candidate(ctx context.Context, candidate *BridgeCandidate) (bool, error) {
	m.caseBytes += int64(len(candidate.Case))
	if m.Limits.CaseBytes > 0 && m.caseBytes > m.Limits.CaseBytes {
		return true, m.limit(ctx, LimitCaseBytes)
	}
	entry := CandidateReport{Edit: candidate.Edit, Digest: candidate.Digest, Identity: candidate.Identity, Classes: []Class{}}
	target, rejection := m.prepare(candidate)
	if rejection != "" {
		m.progress("candidate %s %s prepare-rejected %s", candidate.Digest, candidate.Edit.Edit, rejection)
		return m.settle(ctx, candidate, entry, Decided{PrepareRejected: &rejection})
	}
	m.progress("candidate %s %s", candidate.Digest, candidate.Edit.Edit)
	reruns, err := Rerun(ctx, m.Binder, target)
	if err != nil {
		return true, m.fail(ctx, candidate, fmt.Sprintf("rerun: %s", err))
	}
	m.report.Runs += len(reruns.Attempts)
	m.count(reruns.Attempts)
	if ctx.Err() != nil {
		return true, m.lose(ctx, candidate)
	}
	// Each Run's class in the order it closed; an indeterminate Run is rerun alone once and its
	// retry's class stands in the pair for it.
	final := make([]Class, 0, len(reruns.Attempts))
	for _, attempt := range reruns.Attempts {
		entry.Classes = append(entry.Classes, attempt.Class)
		final = append(final, attempt.Class)
	}
	for index, class := range final {
		if class != ClassIndeterminate {
			continue
		}
		if m.Limits.Runs > 0 && m.report.Runs+1 > m.Limits.Runs {
			m.report.Candidates = append(m.report.Candidates, entry)
			return true, m.limit(ctx, LimitRuns)
		}
		retry, err := RerunOnce(ctx, m.Binder, target)
		if err != nil {
			return true, m.fail(ctx, candidate, fmt.Sprintf("retry: %s", err))
		}
		m.report.Runs++
		m.count([]Attempt{retry})
		if ctx.Err() != nil {
			return true, m.lose(ctx, candidate)
		}
		m.progress("retried %s attempt %d %s", candidate.Digest, index+1, retry.Class)
		entry.Classes = append(entry.Classes, retry.Class)
		final[index] = retry.Class
	}
	entry.Class = ClassifyPair(final...)
	return m.settle(ctx, candidate, entry, Decided{Class: entry.Class})
}

// prepare decodes the candidate's Case and prepares it under the subject's Profile name; the
// returned string is the rejection when it does not.
func (m *minimizer) prepare(candidate *BridgeCandidate) (Target, string) {
	source, err := testpilot.DecodeCaseProtoJSON(candidate.Case)
	if err != nil {
		return Target{}, fmt.Sprintf("decode: %s", err)
	}
	if source.GetCaseId() != candidate.CaseID {
		return Target{}, fmt.Sprintf("the Case is %s, not the candidate's %s", source.GetCaseId(), candidate.CaseID)
	}
	prepared, err := m.Prepare(m.Subject.Driver.Profile, source)
	if err != nil {
		return Target{}, err.Error()
	}
	return Target{Case: source, Prepared: prepared, Driver: prepared.Identity(), Key: m.Subject.Key}, ""
}

func (m *minimizer) count(attempts []Attempt) {
	for _, attempt := range attempts {
		m.runEvents += int64(len(attempt.Run.GetEvents()))
	}
}

// settle tells the bridge what the candidate's Runs decided and records the fate it answers. An
// undecided edit ends the reduction, as the bridge does.
func (m *minimizer) settle(ctx context.Context, candidate *BridgeCandidate, entry CandidateReport, decided Decided) (bool, error) {
	reply, err := m.Bridge.Observe(ctx, candidate.Digest, decided)
	if err != nil {
		m.report.Candidates = append(m.report.Candidates, entry)
		m.state = stateDone
		return true, fmt.Errorf("observe %s: %w", candidate.Digest, err)
	}
	entry.Fate, entry.Reason = reply.Fate, reply.Reason
	m.report.Candidates = append(m.report.Candidates, entry)
	m.report.Retained = reply.Retained
	m.progress("settled %s %s %s", candidate.Digest, candidate.Edit.Edit, reply.Fate)
	if reply.Fate == "undecided" {
		return true, m.finish(ctx, "")
	}
	return false, nil
}

// limit ends the reduction at a limit, naming it.
func (m *minimizer) limit(ctx context.Context, limit string) error {
	m.report.Limit = limit
	m.progress("limit %s", limit)
	return m.finish(ctx, "limit-reached")
}

// lose ends the reduction on a stop that fell while a candidate's Runs were open: the candidate is
// named as lost, never settled with a class its Runs did not finish deciding.
func (m *minimizer) lose(ctx context.Context, candidate *BridgeCandidate) error {
	m.report.Stopped = true
	m.report.Lost = candidate.Digest
	m.progress("lost %s", candidate.Digest)
	return m.finish(ctx, "stopped")
}

// fail ends the reduction on a rerun that could not bind, run or release honestly.
func (m *minimizer) fail(ctx context.Context, candidate *BridgeCandidate, failure string) error {
	m.report.Failure = failure
	m.progress("failed %s %s", candidate.Digest, failure)
	return m.finish(ctx, "stopped")
}

// finish asks the bridge for the result. The bridge is asked on a context the caller's
// cancellation does not reach, since its answer is what a stopped reduction still reports.
func (m *minimizer) finish(ctx context.Context, status string) error {
	m.state = stateDone
	finished, err := m.Bridge.Finish(context.WithoutCancel(ctx), status)
	if err != nil {
		return fmt.Errorf("finish: %w", err)
	}
	m.report.Edits = finished.Edits
	m.report.Retained = finished.Retained
	m.report.Status, m.report.Reason = finished.Status, finished.Reason
	if !m.report.Attempted {
		m.report.Status, m.report.Reason = ReductionNotAttempted, m.report.NotAttempted
	}
	rendered, err := json.Marshal(m.report)
	if err != nil {
		return fmt.Errorf("render reduction: %w", err)
	}
	if m.Limits.ReportBytes > 0 && int64(len(rendered)) > m.Limits.ReportBytes {
		m.report.Limit = LimitReportBytes
		return &campaign.LimitError{Limit: LimitReportBytes, Measured: int64(len(rendered)), Cap: m.Limits.ReportBytes}
	}
	return nil
}

// Render is the report's canonical bytes.
func (r Reduction) Render() ([]byte, error) { return json.Marshal(r) }
