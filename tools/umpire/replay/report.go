package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.temporal.io/server/tools/umpire/binding"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// Exit codes of one replay. A tooling failure outranks everything, because nothing it reports can
// be trusted; then what the reruns said of the subject; then whether the reduction completed.
const (
	ExitReproduced     = 0
	ExitNotReproduced  = 1
	ExitIndeterminate  = 2
	ExitToolingFailure = 3
)

// Request is one replay: the subject's Case and recorded Run, and the set and Query (or
// exploration target) the bridge recovers its Query by.
type Request struct {
	Case  []byte
	Run   []byte
	Set   string
	Named Named
}

// Environment is what a replay runs against. Prepare touches no deployment and serves admission;
// StartBridge spawns the replay bridge; OpenBinder opens the deployment, and is called only after
// the subject is admitted, by the recorded Run and by the bridge.
type Environment struct {
	Prepare       Preparer
	StartBridge   func(ctx context.Context) (*Bridge, error)
	OpenBinder    func(ctx context.Context) (campaign.Binder, func(context.Context) error, error)
	PromotionRoot string
	Limits        Limits
	Now           func() time.Time
	Progress      io.Writer
}

// AdmissionReport says whether the subject was admitted and, when not, the rejection's reason and
// what was found. A subject the set does not produce is `crossed`.
type AdmissionReport struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// SemanticReplayReport is the offline replay of the recorded Run through the same prepared
// Contract: `reproduced` when it gave the recorded Verdict, `rejected` when it erred or disagreed,
// `not-run` when admission rejected the subject before it.
type SemanticReplayReport struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// RerunReport is one fresh Run of the subject: its class and why, and the Run's identity and how
// it closed.
type RerunReport struct {
	Class       Class  `json:"class"`
	Detail      string `json:"detail,omitempty"`
	RunID       string `json:"runId,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Cleanup     string `json:"cleanup,omitempty"`
	Verdict     string `json:"verdict,omitempty"`
	Key         string `json:"key,omitempty"`
	// Diagnostics are the Run's own, by kind, code and detail: why a Run ended as it did.
	Diagnostics []string `json:"diagnostics,omitempty"`
}

// ReproductionReport is the subject's two fresh reruns and the pair's class.
type ReproductionReport struct {
	Class  Class         `json:"class"`
	Reruns []RerunReport `json:"reruns"`
}

// LimitsReport names the fixed limits the replay ran under, by value.
type LimitsReport struct {
	Edits       int    `json:"edits"`
	Runs        int    `json:"runs"`
	WallTime    string `json:"wallTime"`
	CaseBytes   int64  `json:"caseBytes"`
	RunEvents   int64  `json:"runEvents"`
	ReportBytes int64  `json:"reportBytes"`
}

// CleanupReport says whether what the replay opened was released.
type CleanupReport struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Report is one replay's canonical report. Each answer has its own field: admission, the offline
// semantic replay, the Contract-relative key and the Case identity beside it (never in it), the
// reruns, the reduction, the limits, cleanup, the proposal and a tooling failure. History replay
// has no field: it proves nothing here.
type Report struct {
	Admission      AdmissionReport      `json:"admission"`
	SemanticReplay SemanticReplayReport `json:"semanticReplay"`
	Key            string               `json:"key"`
	Identity       string               `json:"identity"`
	Reproduction   *ReproductionReport  `json:"reproduction"`
	Reduction      *Reduction           `json:"reduction"`
	Limits         LimitsReport         `json:"limits"`
	Cleanup        CleanupReport        `json:"cleanup"`
	Proposal       ProposalReport       `json:"proposal"`
	Failure        string               `json:"failure,omitempty"`
}

// ReasonUnrecovered is a subject the bridge could not recover a Query for: an unknown set, Query
// or target. Unlike crossed, the bridge never produced a Case to compare.
const ReasonUnrecovered = "unrecovered"

// The statuses a report's fields take beside the classes and the proposal's.
const (
	StatusUndecided  = "undecided"
	StatusAdmitted   = "admitted"
	StatusRejected   = "rejected"
	StatusReproduced = "reproduced"
	StatusNotRun     = "not-run"
	StatusReleased   = "released"
	StatusFailed     = "failed"
	StatusNothing    = "nothing-opened"
)

func limitsReport(limits Limits) LimitsReport {
	return LimitsReport{
		Edits: limits.Edits, Runs: limits.Runs, WallTime: limits.WallTime.String(),
		CaseBytes: limits.CaseBytes, RunEvents: limits.RunEvents, ReportBytes: limits.ReportBytes,
	}
}

// Execute runs one replay: admission and the offline semantic replay, before anything is opened;
// the bridge's recovery of the subject's Query, which a subject the set does not produce fails as
// crossed; then, and only then, the deployment is opened, the subject is rerun twice, and the
// reduction and its proposal follow. Everything opened is released before it returns.
func Execute(ctx context.Context, request Request, environment Environment) (report Report) {
	report = Report{
		Admission:      AdmissionReport{Status: StatusUndecided},
		SemanticReplay: SemanticReplayReport{Status: StatusNotRun},
		Limits:         limitsReport(environment.Limits),
		Cleanup:        CleanupReport{Status: StatusNothing},
		Proposal:       ProposalReport{Status: ProposalNone},
	}
	subject, err := Admit(ctx, request.Case, request.Run, environment.Prepare)
	if err != nil && ctx.Err() != nil {
		// A stop during the offline replay decides nothing about the subject.
		return report.stopped("stopped during admission", "", nil)
	}
	if err != nil {
		rejection, ok := IsRejection(err)
		if !ok {
			report.Failure = err.Error()
			return report
		}
		report.Admission = AdmissionReport{Status: StatusRejected, Reason: rejection.Reason, Detail: rejection.Detail}
		if rejection.Reason == ReasonReplay {
			report.SemanticReplay = SemanticReplayReport{Status: StatusRejected, Detail: rejection.Detail}
		}
		return report
	}
	report.SemanticReplay.Status = StatusReproduced
	report.Key, report.Identity = subject.Key.String(), subject.Identity

	bridge, err := environment.StartBridge(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return report.stopped("stopped before the bridge started", "", nil)
		}
		report.Failure = fmt.Sprintf("start the replay bridge: %s", err)
		return report
	}
	releases := []func(context.Context) error{func(context.Context) error { return bridge.Close() }}
	defer func() {
		// Teardown runs on a context the caller's stop does not reach, each release bounded.
		if err := binding.ReleaseAll(context.WithoutCancel(ctx), releases); err != nil {
			report.Cleanup = CleanupReport{Status: StatusFailed, Detail: err.Error()}
		} else {
			report.Cleanup.Status = StatusReleased
		}
	}()
	admitted, err := bridge.Admit(ctx, request.Set, subject.Driver.Profile, request.Named, subject.Identity)
	if err != nil && ctx.Err() != nil {
		return report.stopped("stopped during the bridge's admission", "", nil)
	}
	if err != nil {
		var crossed *CrossedError
		if errors.As(err, &crossed) {
			report.Admission = AdmissionReport{Status: StatusRejected, Reason: ReasonCrossed, Detail: crossed.Reason}
			return report
		}
		var rejected *campaign.RejectedError
		if errors.As(err, &rejected) {
			report.Admission = AdmissionReport{Status: StatusRejected, Reason: ReasonUnrecovered, Detail: rejected.Reason}
			return report
		}
		report.Failure = fmt.Sprintf("admit the subject on the bridge: %s", err)
		return report
	}
	report.Admission.Status = StatusAdmitted

	binder, release, err := environment.OpenBinder(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return report.stopped("stopped while opening the deployment", admitted.Subject, bridge)
		}
		report.Failure = fmt.Sprintf("open the deployment: %s", err)
		return report
	}
	releases = append(releases, release)
	reruns, err := Rerun(ctx, binder, subject.Target())
	if err != nil {
		if ctx.Err() == nil {
			report.Failure = fmt.Sprintf("rerun the subject: %s", err)
			return report
		}
		// A stop during the subject's reruns decides nothing: the subject is indeterminate, the
		// Runs that closed are reported, the reduction is not attempted, and the bridge is told.
		report.Reproduction = reproductionReport(reruns)
		return report.stopped("stopped during the subject's reruns", admitted.Subject, bridge)
	}
	report.Reproduction = reproductionReport(reruns)
	if environment.Progress != nil {
		cli.WriteLine(environment.Progress, "subject %s %s", subject.Identity, reruns.Class)
	}

	reduction, err := Reducer{
		Bridge: bridge, Admitted: admitted, Binder: binder, Prepare: environment.Prepare,
		Subject: subject, Limits: environment.Limits, Now: environment.Now, Progress: environment.Progress,
	}.Reduce(ctx, reruns)
	report.Reduction = &reduction
	var limit *campaign.LimitError
	if err != nil && !errors.As(err, &limit) {
		report.Failure = fmt.Sprintf("reduce: %s", err)
		return report
	}
	report.Proposal = WriteProposal(environment.PromotionRoot, reduction.Proposal)
	return report
}

// stopped records a stop that fell before the reduction could start: the reduction is reported
// not attempted and stopped, with the subject's Runs that closed, and a bridge that admitted the
// subject is told, on a context the stop does not reach.
func (r *Report) stopped(reason, subject string, bridge *Bridge) Report {
	runs := 0
	if r.Reproduction != nil {
		runs = len(r.Reproduction.Reruns)
	}
	r.Reduction = &Reduction{
		NotAttempted: reason, Status: ReductionNotAttempted, Reason: reason, Stopped: true,
		Subject: subject, Retained: subject, Runs: runs, Edits: []Settled{}, Candidates: []CandidateReport{},
	}
	if bridge != nil {
		if _, err := bridge.Finish(context.WithoutCancel(context.Background()), string(campaign.StatusStopped)); err != nil {
			r.Failure = fmt.Sprintf("finish the bridge after a stop: %s", err)
		}
	}
	return *r
}

func reproductionReport(reruns *Reruns) *ReproductionReport {
	report := &ReproductionReport{Class: reruns.Class, Reruns: []RerunReport{}}
	if reruns == nil {
		report.Class = ClassIndeterminate
		return report
	}
	for _, attempt := range reruns.Attempts {
		entry := RerunReport{Class: attempt.Class, Detail: attempt.Detail}
		if run := attempt.Run; run != nil {
			entry.RunID, entry.Disposition = run.GetRunId(), run.GetDisposition().String()
			entry.Cleanup = run.GetCleanup().GetStatus().String()
			for _, diagnostic := range run.GetDiagnostics() {
				entry.Diagnostics = append(entry.Diagnostics,
					fmt.Sprintf("%s %s: %s", diagnostic.GetKind(), diagnostic.GetCode(), diagnostic.GetDetail()))
			}
		}
		if attempt.Verdict != nil {
			entry.Verdict = attempt.Verdict.GetStatus().String()
		}
		if len(attempt.Key.Rules) > 0 {
			entry.Key = attempt.Key.String()
		}
		report.Reruns = append(report.Reruns, entry)
	}
	return report
}

// ExitCode maps the report to the command's exit code: 3 for a tooling failure (a candidate's
// rerun that could not bind or release included); 2 for a stop before the reduction started; 3
// for a rejected subject or a proposal that did not compile or could not be written; then 1 or 2
// by the reruns' class when the subject was not reproduced; then 2 for a reduction that did not
// complete; else 0.
func (r Report) ExitCode() int {
	if r.Failure != "" {
		return ExitToolingFailure
	}
	if r.Reduction != nil && r.Reduction.Stopped && !r.Reduction.Attempted {
		return ExitIndeterminate
	}
	if r.Admission.Status != StatusAdmitted {
		return ExitToolingFailure
	}
	if r.Reduction != nil && r.Reduction.Failure != "" {
		return ExitToolingFailure
	}
	if r.Proposal.Status == ProposalNotCompiled || r.Proposal.Status == ProposalWriteFailed {
		return ExitToolingFailure
	}
	if r.Reproduction == nil {
		return ExitToolingFailure
	}
	switch r.Reproduction.Class {
	case ClassNotReproduced:
		return ExitNotReproduced
	case ClassIndeterminate:
		return ExitIndeterminate
	default:
	}
	if r.Reduction == nil || r.Reduction.Stopped || r.Reduction.Limit != "" ||
		(r.Reduction.Status != "minimized" && r.Reduction.Status != "irreducible") {
		return ExitIndeterminate
	}
	return ExitReproduced
}

// ExitCodeWithin is ExitCode for a report rendered to `rendered` bytes under a cap: a report over
// the cap is written whole, never truncated, and a replay that would otherwise exit 0 did not
// complete within its limits, so it exits 2.
func (r Report) ExitCodeWithin(rendered int, limit int64) int {
	code := r.ExitCode()
	if limit > 0 && int64(rendered) > limit && code == ExitReproduced {
		return ExitIndeterminate
	}
	return code
}

// Render is the report's canonical bytes: one JSON document and one LF.
func (r Report) Render() ([]byte, error) {
	rendered, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return append(rendered, '\n'), nil
}
