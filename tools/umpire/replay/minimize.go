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
// admission (a sweep the bridge capped is recorded as ending at it); the wall time, the Run budget
// and the aggregate Run Events once a candidate is handed out, before it is prepared, and again
// before each Run is dispatched; the aggregate Case bytes on each candidate that arrives; and the
// report bytes on the rendered report. Zero leaves a cap
// unset.
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
	// Proposal is the bridge's proposal for a minimized or irreducible result, source bytes
	// included, carried for WriteProposal; the reduction's JSON leaves it out.
	Proposal *BridgeProposal `json:"-"`
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

type minimizer struct {
	Reducer
	deadline  time.Time
	caseBytes int64
	runEvents int64
	report    Reduction
}

// Reduce runs one sweep. It starts only when the subject's reruns reproduced its key; otherwise
// the report says the reduction was not attempted, and the bridge is finished as stopped. For
// each candidate the bridge hands out, it prepares the Case under the subject's Profile name (a
// rejection is reported as `rejected`, never rerun), dispatches two fresh Runs one at a time
// through RerunOnce, reruns each indeterminate Run of an indeterminate pair alone once, and
// reports the pair's class. It ends when the bridge's sweep is
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
		err = m.finish(ctx, string(campaign.StatusStopped))
	case r.Limits.Edits > 0 && len(r.Admitted.Edits) > r.Limits.Edits:
		m.report.Attempted = true
		err = m.limit(ctx, LimitEdits)
	default:
		m.report.Attempted = true
		for done := false; !done; {
			done, err = m.ask(ctx)
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
		return true, m.finish(ctx, string(campaign.StatusStopped))
	}
	next, err := m.Bridge.Next(ctx)
	if err != nil {
		return true, fmt.Errorf("next: %w", err)
	}
	for _, skipped := range next.Skipped {
		m.progress("skipped %s %s %s", skipped.Edit.Edit, skipped.Fate, skipped.Reason)
	}
	if next.Exhausted {
		if m.Admitted.Capped {
			// The bridge enumerated the cap's worth of edits and no more: a sweep that ran them
			// all ends incomplete at the edit cap.
			m.report.Limit = LimitEdits
		}
		return true, m.finish(ctx, "")
	}
	return m.candidate(ctx, next.Candidate)
}

// beforeDispatch names the limit that `runs` more Runs would cross, or nothing: the wall time,
// the Run budget and the aggregate Run Events, each checked before a Run is dispatched.
func (m *minimizer) beforeDispatch(runs int) string {
	switch {
	case m.Limits.WallTime > 0 && !m.Now().Before(m.deadline):
		return LimitWallTime
	case m.Limits.Runs > 0 && m.report.Runs+runs > m.Limits.Runs:
		return LimitRuns
	case m.Limits.RunEvents > 0 && m.runEvents >= m.Limits.RunEvents:
		return LimitRunEvents
	default:
		return ""
	}
}

// dispatch runs one fresh attempt of the target, after the stop and every limit that bounds a Run
// are checked; the closed Run is counted whatever it says. stop says the reduction ended here.
func (m *minimizer) dispatch(ctx context.Context, candidate *BridgeCandidate, target Target, dispatched bool) (Attempt, bool, error) {
	if ctx.Err() != nil {
		return Attempt{}, true, m.stop(ctx, candidate, dispatched)
	}
	if limit := m.beforeDispatch(1); limit != "" {
		return Attempt{}, true, m.limit(ctx, limit)
	}
	attempt, err := RerunOnce(ctx, m.Binder, target)
	if attempt.Run != nil {
		m.report.Runs++
		m.runEvents += int64(len(attempt.Run.GetEvents()))
	}
	if err != nil {
		if ctx.Err() != nil {
			return Attempt{}, true, m.stop(ctx, candidate, true)
		}
		return Attempt{}, true, m.fail(ctx, candidate, fmt.Sprintf("rerun: %s", err))
	}
	if ctx.Err() != nil {
		// The Run closed and is returned with the stop, so its class is still listed.
		return attempt, true, m.stop(ctx, candidate, true)
	}
	return attempt, false, nil
}

// candidate takes one candidate through preparation, its Runs and the bridge.
func (m *minimizer) candidate(ctx context.Context, candidate *BridgeCandidate) (bool, error) {
	// The limits that bound a candidate's Runs are checked once one is handed out and before it is
	// prepared: a sweep with nothing left ends exhausted, never at a limit it would not reach.
	if limit := m.beforeDispatch(Attempts); limit != "" {
		return true, m.limit(ctx, limit)
	}
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
	// Each Run is dispatched on its own, so a stop or a limit between the two falls before the
	// second Run rather than after it.
	// A reduction that ends while this candidate's Runs are open still reports the classes of the
	// Runs that closed, with no fate: the Runs spent and the classes listed agree.
	ended := func(err error) (bool, error) {
		if len(entry.Classes) > 0 {
			m.report.Candidates = append(m.report.Candidates, entry)
		}
		return true, err
	}
	final := make([]Class, 0, Attempts)
	for range Attempts {
		attempt, done, err := m.dispatch(ctx, candidate, target, len(final) > 0)
		if attempt.Class != "" {
			entry.Classes = append(entry.Classes, attempt.Class)
		}
		if done {
			return ended(err)
		}
		final = append(final, attempt.Class)
	}
	// Only a pair that is itself indeterminate is retried, and only while it stays so: a
	// not-reproduced Run decides the pair, whether it came first or on a retry. Each indeterminate
	// Run is rerun alone once, one Run apiece, and its retry's class stands in the pair for it.
	for index, class := range final {
		if ClassifyPair(final...) != ClassIndeterminate {
			break
		}
		if class != ClassIndeterminate {
			continue
		}
		retry, done, err := m.dispatch(ctx, candidate, target, true)
		if retry.Class != "" {
			entry.Classes = append(entry.Classes, retry.Class)
		}
		if done {
			return ended(err)
		}
		m.progress("retried %s attempt %d %s", candidate.Digest, index+1, retry.Class)
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

// settle tells the bridge what the candidate's Runs decided and records the fate it answers. An
// undecided edit ends the reduction, as the bridge does.
func (m *minimizer) settle(ctx context.Context, candidate *BridgeCandidate, entry CandidateReport, decided Decided) (bool, error) {
	reply, err := m.Bridge.Observe(ctx, candidate.Digest, decided)
	if err != nil {
		m.report.Candidates = append(m.report.Candidates, entry)
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
	return m.finish(ctx, string(campaign.StatusLimitReached))
}

// stop ends the reduction on a stop. When a Run of the candidate was dispatched, the candidate is
// named as lost, never settled with a class its Runs did not finish deciding.
func (m *minimizer) stop(ctx context.Context, candidate *BridgeCandidate, dispatched bool) error {
	m.report.Stopped = true
	if dispatched {
		m.report.Lost = candidate.Digest
		m.progress("lost %s", candidate.Digest)
	}
	return m.finish(ctx, string(campaign.StatusStopped))
}

// fail ends the reduction on a rerun that could not bind, run or release honestly.
func (m *minimizer) fail(ctx context.Context, candidate *BridgeCandidate, failure string) error {
	m.report.Failure = failure
	m.progress("failed %s %s", candidate.Digest, failure)
	return m.finish(ctx, string(campaign.StatusStopped))
}

// finish asks the bridge for the result. The bridge is asked on a context the caller's
// cancellation does not reach, since its answer is what a stopped reduction still reports.
func (m *minimizer) finish(ctx context.Context, status string) error {
	finished, err := m.Bridge.Finish(context.WithoutCancel(ctx), status)
	if err != nil {
		return fmt.Errorf("finish: %w", err)
	}
	m.report.Edits = finished.Edits
	m.report.Retained = finished.Retained
	m.report.Status, m.report.Reason = finished.Status, finished.Reason
	m.report.Proposal = finished.Proposal
	if !m.report.Attempted {
		m.report.Proposal = nil
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
