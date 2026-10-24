package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"google.golang.org/grpc"
	"umpire.local/model/umpire/runner"
	"umpire.local/temporal"
)

// The invocation's statuses beyond preflight's refusals and the iterations' verdicts, each its own
// named outcome.
const (
	StatusPolicyUnavailable     = "policy-unavailable"
	StatusAuthorityUnavailable  = "authority-unavailable"
	StatusToolingFailure        = "tooling-failure"
	StatusLeaseUnreconciled     = "lease-unreconciled"
	StatusCleanupUncertain      = "cleanup-uncertain"
	StatusNoIteration           = "no-iteration"
	StatusPublicationConflict   = "publication-conflict"
	StatusPublicationUnreported = "publication-unreported"
	StatusPublicationFailed     = "publication-failed"
	StatusInterrupted           = "interrupted"
)

// publishTimeout bounds publication, which runs under its own context after the cleanup attempt,
// so an interrupt that ended the iterations still publishes every Run already recorded.
const publishTimeout = time.Minute

// The exit codes, by precedence: the highest an invocation reaches is its exit.
const (
	ExitAccepted  = 0
	ExitDecided   = 1
	ExitUncertain = 2
	ExitFailed    = 3
)

// maxDetailBytes bounds the summary's one free-text field.
const maxDetailBytes = 4096

// Seams are what a build supplies: the policy, the Model guest it runs, the authority that turns
// the environment into a transport, and a hook told of each phase. The untagged build's are
// ProductionSeams; only the harness build supplies others.
type Seams struct {
	Policy    func() (*policy.Policy, error)
	Guest     []byte
	Authority func(authority.Lookup) (*authority.Authority, error)
	Hook      func(phase string)
}

// ProductionSeams are the untagged build's: the embedded policy, the committed Temporal guest, the
// credential-requiring authority, and no hook.
func ProductionSeams() Seams {
	return Seams{Policy: policy.Embedded, Guest: temporal.Guest(), Authority: authority.Load}
}

// Invocation is one `run`: the build's seams, the job's environment, the existing output
// directory the artifact uploads, the recovery record's path, where progress goes, and when it
// started.
type Invocation struct {
	Seams    Seams
	Lookup   authority.Lookup
	Output   string
	Recovery string
	Progress io.Writer
	Started  time.Time
	// dial and service are the SDK client and the lease's view of the target; a unit test
	// replaces them with a fake server.
	dial    func(client.Options) (client.Client, error)
	service func(*authority.Authority, string, io.Writer) (*lazyService, error)
	// prepare lets a unit test script the iterations.
	prepare func(*invocation)
	// publish is runner.Publish; a unit test wraps it to fail a publication.
	publish func(context.Context, string, *runner.Run) (runner.Publication, error)
}

// Summary is the one JSON document `run` writes, in a fixed key order. It carries statuses,
// identities and IDs only; its detail passes through the Redactor.
type Summary struct {
	Status     string             `json:"status"`
	Detail     string             `json:"detail,omitempty"`
	Invocation string             `json:"invocation,omitempty"`
	Iterations []SummaryIteration `json:"iterations"`
	Cleanup    *SummaryCleanup    `json:"cleanup,omitempty"`
}

// SummaryIteration is one iteration: its fenced activity, its status, and its published Run's key
// under the output directory.
type SummaryIteration struct {
	ActivityID string `json:"activityId"`
	Status     string `json:"status"`
	Run        string `json:"run,omitempty"`
}

// The cleanup outcomes a summary names.
const (
	CleanupReleased  = "released"
	CleanupUncertain = "uncertain"
)

// SummaryCleanup is the invocation's cleanup: released or uncertain, and the fenced activities.
type SummaryCleanup struct {
	Outcome    string   `json:"outcome"`
	Fenced     []string `json:"fenced"`
	Unverified []string `json:"unverified"`
}

// outcome gathers an invocation's statuses and keeps the one with the highest exit.
type outcome struct {
	summary Summary
	code    int
}

func (o *outcome) raise(status string, code int, detail string) {
	if o.summary.Status == "" || code > o.code {
		o.summary.Status, o.code, o.summary.Detail = status, code, detail
	}
}

// Invoke is one `run`: preflight, which pins the Model guest; the lease; the serial iterations,
// each one Run the Runner records in memory; the cleanup attempt; then publication of every Run
// with runner.Publish, whatever the cleanup outcome. It returns the summary and the exit code by
// precedence: 3 for a refusal, a tooling failure, a failed iteration or a publication failure; 2
// for an unreconciled lease or an uncertain cleanup; 1 for an unexplained or refused Run; 0 when
// every Run is explained.
func Invoke(ctx context.Context, invocation Invocation) (Summary, int) {
	result := &outcome{summary: Summary{Iterations: []SummaryIteration{}}}
	redactor := authority.NewRedactor()
	finish := func() (Summary, int) {
		result.summary.Detail = truncate(redactor.Redact(result.summary.Detail))
		return result.summary, result.code
	}
	canary, err := invocation.Seams.Policy()
	if err != nil {
		result.raise(StatusPolicyUnavailable, ExitFailed, err.Error())
		return finish()
	}
	loaded, err := invocation.Seams.Authority(invocation.Lookup)
	if err != nil {
		result.raise(StatusAuthorityUnavailable, ExitFailed, err.Error())
		return finish()
	}
	redactor = loaded.Redactor
	// One capped, redacting writer for everything the invocation and its SDK clients write.
	progress := progressWriter(redactor, invocation.Progress, canary.Limits.ProgressBytes)
	defer func() { _ = progress.Close() }()

	dial := invocation.dial
	if dial == nil {
		dial = client.Dial
	}
	service, err := openService(invocation.service, dial, loaded, progress)
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}
	defer service.close()
	scope, err := preflight.Check(ctx, preflight.Input{
		Policy: canary, Lookup: invocation.Lookup, Coordinates: loaded.Coordinates, Redactor: redactor, Namespaces: service,
		Guest: invocation.Seams.Guest,
	})
	if refusal, refused := preflight.AsRefusal(err); refused {
		result.raise(refusal.Status, ExitFailed, refusal.Detail)
		return finish()
	}
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}
	result.summary.Invocation = scope.InvocationID
	store, err := recovery.Create(invocation.Recovery, scope.InvocationID)
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}

	config := Config{
		Policy: canary, Scope: scope, Namespace: loaded.Coordinates.Namespace, TaskQueue: loaded.Coordinates.TaskQueue,
		Transport: loaded.Transport, Redactor: redactor, Service: service, Identity: identityPrefix + scope.InvocationID,
		Recovery: store, Progress: capped{progress}, Started: invocation.Started, Hook: invocation.Seams.Hook,
	}
	if err := config.validate(); err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}
	ran, err := runWith(ctx, config, invocation.prepare)
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}
	if ran.Unreconciled {
		result.raise(StatusLeaseUnreconciled, ExitUncertain, "the canary lease's latest run is open or closed without a canary termination; reconcile it first")
		return finish()
	}
	switch ran.StoppedBy {
	case StopRelease, StopRecord:
		result.raise(StatusToolingFailure, ExitFailed, ran.Stopped)
	case StopInterrupted:
		result.raise(StatusInterrupted, ExitFailed, ran.Stopped)
	default:
		// A decision or the limit ended the loop; the iterations' own statuses say what it was.
	}
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), publishTimeout)
	defer cancel()
	invocation.record(publishCtx, ran, store, result)
	return finish()
}

// record raises the iterations' and cleanup's statuses, then publishes every recorded Run,
// whatever the cleanup outcome.
func (invocation Invocation) record(ctx context.Context, ran *Result, store *recovery.Store, result *outcome) {
	cleanupOf(ran, result)
	runs := invocation.decided(ran, result)
	if len(ran.Iterations) == 0 {
		result.raise(StatusNoIteration, ExitFailed, ran.Stopped)
	}
	invocation.publishAll(ctx, runs, store, result)
}

// cleanupOf raises an uncertain cleanup and summarizes it, naming the fenced activities: the
// fence's own list, or, when it could not be read back, the iterations' own activities.
func cleanupOf(ran *Result, result *outcome) {
	cleanup := ran.Cleanup
	fenced := cleanup.Fenced
	if len(fenced) == 0 {
		for _, iteration := range ran.Iterations {
			if iteration.ActivityID != "" {
				fenced = append(fenced, iteration.ActivityID)
			}
		}
	}
	outcome := CleanupReleased
	if !cleanup.Released {
		outcome = CleanupUncertain
		detail := "cleanup could not verify every fenced activity closed; the lease stays held for reconcile"
		if cleanup.Err != nil {
			detail = cleanup.Err.Error()
		}
		result.raise(StatusCleanupUncertain, ExitUncertain, detail)
	}
	result.summary.Cleanup = &SummaryCleanup{Outcome: outcome, Fenced: nonNil(fenced), Unverified: nonNil(cleanup.Unverified)}
}

// decided raises each iteration's status, summarizes it, and returns the Runs to publish: every
// iteration's but a failed one's, which recorded none.
func (invocation Invocation) decided(ran *Result, result *outcome) []published {
	var runs []published
	for index, iteration := range ran.Iterations {
		label := "iteration " + fmt.Sprint(index+1)
		switch iteration.Outcome.Status {
		case StatusExplained:
		case StatusUnexplained, StatusRefused:
			result.raise(iteration.Outcome.Status, ExitDecided, label+" was "+iteration.Outcome.Status)
		default:
			detail := label + " failed"
			if iteration.Outcome.Err != nil {
				detail += ": " + iteration.Outcome.Err.Error()
			}
			result.raise(StatusFailed, ExitFailed, detail)
			result.summary.Iterations = append(result.summary.Iterations, SummaryIteration{ActivityID: iteration.ActivityID, Status: StatusFailed})
			continue
		}
		result.summary.Iterations = append(result.summary.Iterations, SummaryIteration{ActivityID: iteration.ActivityID, Status: iteration.Outcome.Status})
		runs = append(runs, published{activityID: iteration.ActivityID, run: iteration.Outcome.Run, entry: len(result.summary.Iterations) - 1})
	}
	return runs
}

// published is one Run to publish, its activity, and its iteration's place in the summary.
type published struct {
	activityID string
	run        *runner.Run
	entry      int
}

// publishAll publishes each Run in order, records it as published, and names a publication
// failure; a Run already published is the same Run, never a conflict.
func (invocation Invocation) publishAll(ctx context.Context, runs []published, store *recovery.Store, result *outcome) {
	if err := store.Update(func(record *recovery.Record) { record.Phase = recovery.PhasePublishing }); err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return
	}
	publish := invocation.publish
	if publish == nil {
		publish = runner.Publish
	}
	for _, next := range runs {
		publication, err := publish(ctx, invocation.Output, next.run)
		var conflict *runner.ConflictError
		switch {
		case errors.As(err, &conflict):
			result.raise(StatusPublicationConflict, ExitFailed, err.Error())
			return
		case err != nil:
			result.raise(StatusPublicationFailed, ExitFailed, err.Error())
			return
		}
		result.summary.Iterations[next.entry].Run = publication.Key
		if err := store.Update(func(record *recovery.Record) {
			for index := range record.Iterations {
				if record.Iterations[index].ActivityID == next.activityID {
					record.Iterations[index].Published = true
				}
			}
		}); err != nil {
			result.raise(StatusPublicationUnreported, ExitFailed, fmt.Sprintf("%s was published, but the recovery record could not say so: %s", publication.Key, err))
			return
		}
	}
	if err := store.Update(func(record *recovery.Record) { record.Phase = recovery.PhaseFinished }); err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return
	}
	if result.summary.Status == "" {
		result.raise(StatusExplained, ExitAccepted, "")
	}
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func truncate(detail string) string {
	if len(detail) <= maxDetailBytes {
		return detail
	}
	return detail[:maxDetailBytes] + "..."
}

// openService is the target a mode acts on: a unit test's fake when it supplies one, or the SDK
// client dialed on first use.
func openService(fake func(*authority.Authority, string, io.Writer) (*lazyService, error), dial func(client.Options) (client.Client, error),
	loaded *authority.Authority, progress io.Writer) (*lazyService, error) {
	if fake != nil {
		return fake(loaded, loaded.Coordinates.Namespace, progress)
	}
	return connectLazily(dial, loaded, loaded.Coordinates.Namespace, progress), nil
}

// connectLazily is the real lease service: an SDK client for the canary namespace, dialed on its
// first use, so preflight's connection-free checks refuse before anything reaches the target.
func connectLazily(dial func(client.Options) (client.Client, error), loaded *authority.Authority, namespace string, progress io.Writer) *lazyService {
	return &lazyService{dial: func() (client.Client, error) {
		return dial(loaded.Transport.ClientOptions(namespace, loaded.Redactor.Logger(progress)))
	}}
}

// lazyService is the WorkflowService the lease, cleanup and preflight's read use, over one SDK
// client dialed on first use. A unit test supplies a Service directly.
type lazyService struct {
	once    sync.Once
	dial    func() (client.Client, error)
	client  client.Client
	direct  leaseService
	dialErr error
}

// leaseService is what the lease, cleanup and preflight's read call on the target.
type leaseService interface {
	Service
	preflight.Namespaces
}

func (s *lazyService) get() (leaseService, error) {
	if s.direct != nil {
		return s.direct, nil
	}
	s.once.Do(func() {
		s.client, s.dialErr = s.dial()
	})
	if s.dialErr != nil {
		return nil, fmt.Errorf("connect to the canary namespace: %w", s.dialErr)
	}
	return s.client.WorkflowService(), nil
}

func (s *lazyService) close() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *lazyService) DescribeNamespace(ctx context.Context, request *workflowservice.DescribeNamespaceRequest, options ...grpc.CallOption) (*workflowservice.DescribeNamespaceResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.DescribeNamespace(ctx, request, options...)
}

func (s *lazyService) DescribeActivityExecution(ctx context.Context, request *workflowservice.DescribeActivityExecutionRequest, options ...grpc.CallOption) (*workflowservice.DescribeActivityExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.DescribeActivityExecution(ctx, request, options...)
}

func (s *lazyService) TerminateActivityExecution(ctx context.Context, request *workflowservice.TerminateActivityExecutionRequest, options ...grpc.CallOption) (*workflowservice.TerminateActivityExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.TerminateActivityExecution(ctx, request, options...)
}

func (s *lazyService) StartWorkflowExecution(ctx context.Context, request *workflowservice.StartWorkflowExecutionRequest, options ...grpc.CallOption) (*workflowservice.StartWorkflowExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.StartWorkflowExecution(ctx, request, options...)
}

func (s *lazyService) SignalWorkflowExecution(ctx context.Context, request *workflowservice.SignalWorkflowExecutionRequest, options ...grpc.CallOption) (*workflowservice.SignalWorkflowExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.SignalWorkflowExecution(ctx, request, options...)
}

func (s *lazyService) TerminateWorkflowExecution(ctx context.Context, request *workflowservice.TerminateWorkflowExecutionRequest, options ...grpc.CallOption) (*workflowservice.TerminateWorkflowExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.TerminateWorkflowExecution(ctx, request, options...)
}

func (s *lazyService) DescribeWorkflowExecution(ctx context.Context, request *workflowservice.DescribeWorkflowExecutionRequest, options ...grpc.CallOption) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.DescribeWorkflowExecution(ctx, request, options...)
}

func (s *lazyService) GetWorkflowExecutionHistory(ctx context.Context, request *workflowservice.GetWorkflowExecutionHistoryRequest, options ...grpc.CallOption) (*workflowservice.GetWorkflowExecutionHistoryResponse, error) {
	service, err := s.get()
	if err != nil {
		return nil, err
	}
	return service.GetWorkflowExecutionHistory(ctx, request, options...)
}
