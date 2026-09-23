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
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/publication"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/publish"
	"google.golang.org/grpc"
)

// The invocation's statuses beyond preflight's refusals and fn-26's decisions, each its own
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
// so an interrupt that ended the iterations still publishes every receipt already decided.
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

// Seams are what a build supplies: the policy and the Evaluation Profile it names, the authority
// that turns the environment into a transport, and a hook told of each phase. The untagged build's
// are ProductionSeams; only the harness build supplies others.
type Seams struct {
	Policy    func() (*policy.Policy, *evaluation.Profile, error)
	Authority func(authority.Lookup) (*authority.Authority, error)
	Hook      func(phase string)
}

// ProductionSeams are the untagged build's: the embedded policy, its embedded Evaluation Profile,
// the credential-requiring authority, and no hook.
func ProductionSeams() Seams {
	return Seams{
		Policy: func() (*policy.Policy, *evaluation.Profile, error) {
			canary, err := policy.Embedded()
			if err != nil {
				return nil, nil, err
			}
			profile, err := assessment.LoadProfile(canary.EvaluationProfile)
			if err != nil {
				return nil, nil, err
			}
			return canary, profile, nil
		},
		Authority: authority.Load,
	}
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

// SummaryIteration is one iteration: its Run, status, and published documents' identities.
type SummaryIteration struct {
	RunID      string `json:"runId"`
	Status     string `json:"status"`
	Receipt    string `json:"receipt,omitempty"`
	Provenance string `json:"provenance,omitempty"`
}

// SummaryCleanup is the invocation's cleanup: released or uncertain, and the fenced workflows.
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

// Invoke is one `run`: preflight, which prepares the Case once; the lease; the serial iterations,
// each admitted, assessed and rendered in memory; the cleanup attempt; then publication of every
// decided iteration's receipt and provenance, whatever the cleanup outcome. It returns the summary
// and the exit code by precedence: 3 for a refusal, a tooling failure, an unconstructible
// iteration or a publication failure; 2 for an unreconciled lease or an uncertain cleanup; 1 for
// a rejected or incomplete iteration; 0 when every iteration is accepted.
func Invoke(ctx context.Context, invocation Invocation) (Summary, int) {
	result := &outcome{summary: Summary{Iterations: []SummaryIteration{}}}
	redactor := authority.NewRedactor()
	finish := func() (Summary, int) {
		result.summary.Detail = truncate(redactor.Redact(result.summary.Detail))
		return result.summary, result.code
	}
	canary, profile, err := invocation.Seams.Policy()
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
	progress := redactor.Writer(&boundedWriter{out: invocation.Progress, remaining: canary.Limits.ProgressBytes})
	defer func() { _ = progress.Close() }()

	connect := invocation.service
	if connect == nil {
		connect = invocation.connect
	}
	service, err := connect(loaded, loaded.Coordinates.Namespace, progress)
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return finish()
	}
	defer service.close()
	scope, err := preflight.Check(ctx, preflight.Input{
		Policy: canary, Lookup: invocation.Lookup, Coordinates: loaded.Coordinates, Redactor: redactor, Namespaces: service,
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

	dial := invocation.dial
	if dial == nil {
		dial = client.Dial
	}
	config := Config{
		Policy: canary, Scope: scope, Namespace: loaded.Coordinates.Namespace, Transport: loaded.Transport,
		Redactor: redactor, Service: service, Identity: "umpire-canary " + scope.InvocationID, Dial: dial,
		Decide: func(run *testpilotspb.Run, _ *testpilotspb.Verdict) Outcome {
			return decide(canary, profile, scope, run)
		},
		Recovery: store, Progress: progress, Started: invocation.Started, Hook: invocation.Seams.Hook,
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
	invocation.record(publishCtx, canary, scope, ran, store, result)
	return finish()
}

// decide admits the closed Run with fn-26 in memory, assesses it under the Profile and renders its
// receipt. A Run fn-26 does not admit is unconstructible and has no receipt.
func decide(canary *policy.Policy, profile *evaluation.Profile, scope *preflight.Scope, run *testpilotspb.Run) Outcome {
	subject, err := assessment.Admit(canary, scope.Prepared.Identity(), run)
	if err != nil {
		return Outcome{Status: StatusUnconstructible, Err: err}
	}
	decision := evaluation.Assess(subject, *profile)
	receipt, err := evaluation.Render(subject, *profile, decision)
	if err != nil {
		return Outcome{Status: StatusUnconstructible, Err: err}
	}
	return Outcome{Status: decision.Outcome, Receipt: receipt}
}

// record raises the iterations' and cleanup's statuses, then publishes every decided iteration's
// receipt and provenance, whatever the cleanup outcome.
func (invocation Invocation) record(ctx context.Context, canary *policy.Policy, scope *preflight.Scope, ran *Result, store *recovery.Store, result *outcome) {
	fenced, cleanup := cleanupOf(ran, result)
	items, err := invocation.decided(canary, scope, ran, fenced, cleanup, result)
	if err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return
	}
	if len(ran.Iterations) == 0 {
		result.raise(StatusNoIteration, ExitFailed, ran.Stopped)
	}
	invocation.publishAll(ctx, items, store, result)
}

// cleanupOf raises an uncertain cleanup and summarizes it, and names the fenced workflows: the
// fence's own list, or, when it could not be read back, the iterations' own Runs.
func cleanupOf(ran *Result, result *outcome) ([]string, string) {
	cleanup := ran.Cleanup
	fenced := cleanup.Fenced
	if len(fenced) == 0 {
		for _, iteration := range ran.Iterations {
			if iteration.RunID != "" {
				fenced = append(fenced, iteration.RunID)
			}
		}
	}
	outcome := assessment.CleanupReleased
	if !cleanup.Released {
		outcome = assessment.CleanupUncertain
		detail := "cleanup could not verify every fenced workflow closed; the lease stays held for reconcile"
		if cleanup.Err != nil {
			detail = cleanup.Err.Error()
		}
		result.raise(StatusCleanupUncertain, ExitUncertain, detail)
	}
	result.summary.Cleanup = &SummaryCleanup{Outcome: outcome, Fenced: nonNil(fenced), Unverified: nonNil(cleanup.Unverified)}
	return fenced, outcome
}

// decided raises each iteration's status, summarizes it, and makes its publication item: a decided
// iteration's receipt and provenance, or an unconstructible iteration's nothing.
func (invocation Invocation) decided(canary *policy.Policy, scope *preflight.Scope, ran *Result, fenced []string, cleanup string, result *outcome) ([]publication.Item, error) {
	var items []publication.Item
	for index, iteration := range ran.Iterations {
		entry := SummaryIteration{RunID: iteration.RunID, Status: iteration.Outcome.Status}
		label := "iteration " + fmt.Sprint(index+1)
		switch iteration.Outcome.Status {
		case StatusAccepted:
		case StatusRejected, StatusIncomplete:
			result.raise(iteration.Outcome.Status, ExitDecided, label+" was "+iteration.Outcome.Status)
		default:
			detail := label + " is unconstructible"
			if iteration.Outcome.Err != nil {
				detail += ": " + iteration.Outcome.Err.Error()
			}
			result.raise(StatusUnconstructible, ExitFailed, detail)
			result.summary.Iterations = append(result.summary.Iterations, entry)
			items = append(items, publication.Item{RunID: iteration.RunID, Status: publication.StatusUnconstructible})
			continue
		}
		provenance, err := invocation.provenance(canary, scope, ran, index, fenced, cleanup)
		if err != nil {
			return nil, err
		}
		entry.Receipt = evaluation.ReceiptIdentity(iteration.Outcome.Receipt)
		entry.Provenance = assessment.ProvenanceIdentity(provenance)
		result.summary.Iterations = append(result.summary.Iterations, entry)
		items = append(items, publication.Item{RunID: iteration.RunID, Status: iteration.Outcome.Status, Receipt: iteration.Outcome.Receipt, Provenance: provenance})
	}
	return items, nil
}

// publishAll publishes the items, records each as published, and names a publication failure.
func (invocation Invocation) publishAll(ctx context.Context, items []publication.Item, store *recovery.Store, result *outcome) {
	if err := store.Update(func(record *recovery.Record) { record.Phase = recovery.PhasePublishing }); err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return
	}
	_, err := publication.Publish(ctx, invocation.Output, items, func(runID string) error {
		return store.Update(func(record *recovery.Record) {
			for index := range record.Iterations {
				if record.Iterations[index].RunID == runID {
					record.Iterations[index].Published = true
				}
			}
		})
	})
	var conflict *publish.ConflictError
	var unreported *publication.UnreportedError
	switch {
	case errors.As(err, &conflict):
		result.raise(StatusPublicationConflict, ExitFailed, err.Error())
		return
	case errors.As(err, &unreported):
		result.raise(StatusPublicationUnreported, ExitFailed, err.Error())
		return
	case err != nil:
		result.raise(StatusPublicationFailed, ExitFailed, err.Error())
		return
	}
	if err := store.Update(func(record *recovery.Record) { record.Phase = recovery.PhaseFinished }); err != nil {
		result.raise(StatusToolingFailure, ExitFailed, err.Error())
		return
	}
	if result.summary.Status == "" {
		result.raise(StatusAccepted, ExitAccepted, "")
	}
}

// provenance renders one decided iteration's provenance, after the cleanup attempt.
func (invocation Invocation) provenance(canary *policy.Policy, scope *preflight.Scope, ran *Result, index int, fenced []string, cleanup string) ([]byte, error) {
	iteration := ran.Iterations[index]
	receipt, err := evaluation.DecodeReceipt(iteration.Outcome.Receipt)
	if err != nil {
		return nil, err
	}
	workflowRef, _ := invocation.Lookup(preflight.VariableWorkflowRef)
	workflowRun, _ := invocation.Lookup(preflight.VariableRunID)
	record := ran.Lease
	return assessment.RenderProvenance(&assessment.Provenance{
		Version:           assessment.ProvenanceFormatVersion,
		Receipt:           evaluation.ReceiptIdentity(iteration.Outcome.Receipt),
		EvaluationProfile: receipt.Profile.Identity,
		AuthorityClass:    canary.AuthorityClass,
		Workflow:          assessment.ProvenanceWorkflow{Ref: workflowRef, RunID: workflowRun},
		Coordinates:       scope.Coordinates,
		Lease:             assessment.ProvenanceLease{WorkflowIDDigest: policy.Digest(record.WorkflowID), Fence: record.RunID},
		Invocation:        assessment.ProvenanceInvocation{ID: scope.InvocationID, Iteration: index + 1, RunID: iteration.RunID},
		Limits:            canary.Limits,
		Cleanup:           assessment.ProvenanceCleanup{Iteration: receipt.Run.Cleanup, Invocation: cleanup},
		Isolation:         assessment.Isolation,
		Fenced:            fenced,
	})
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

// connect is the real lease service: an SDK client for the canary namespace, dialed on its first
// use, so preflight's connection-free checks refuse before anything reaches the target.
func (invocation Invocation) connect(loaded *authority.Authority, namespace string, progress io.Writer) (*lazyService, error) {
	dial := invocation.dial
	if dial == nil {
		dial = client.Dial
	}
	return &lazyService{dial: func() (client.Client, error) {
		return dial(loaded.Transport.ClientOptions(namespace, loaded.Redactor.Logger(progress)))
	}}, nil
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
