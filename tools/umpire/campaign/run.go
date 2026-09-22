package campaign

import (
	"context"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/binding"
	"google.golang.org/protobuf/encoding/protojson"
)

// Binder binds one candidate's Case to the deployment: derive its Profile, prepare it, open its
// Driver. The campaign binding is the real one; a test supplies its own.
type Binder interface {
	Bind(ctx context.Context, identity string, source *testpilotspb.Case) (Bound, error)
}

// Bound is one prepared candidate with the Driver that runs it.
type Bound interface {
	Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error)
	Release(ctx context.Context) error
}

// CampaignBinder adapts the shared deployment binding to Binder.
type CampaignBinder struct{ Campaign *binding.Campaign }

func (b CampaignBinder) Bind(ctx context.Context, identity string, source *testpilotspb.Case) (Bound, error) {
	bound, err := b.Campaign.Bind(ctx, identity, source)
	if err != nil {
		return nil, err
	}
	return bound, nil
}

// OutcomeKind says how far one candidate got. They are distinguishable, and only Completed had a
// Run; only the bridge says what a Run credited.
type OutcomeKind string

const (
	// OutcomePrepareRejected: the Case's own static rejection, decided before any Driver opened.
	// No Run exists. The campaign advanced only when Credited is set; with an error beside the
	// outcome the bridge could not be told and the candidate is still outstanding.
	OutcomePrepareRejected OutcomeKind = "prepare-rejected"
	// OutcomeBindFailed: the deployment or the Driver could not be opened for a Case that
	// prepared. No Run exists and the candidate is still outstanding on the bridge.
	OutcomeBindFailed OutcomeKind = "bind-failed"
	// OutcomeRunFailed: the Run could not execute (the Driver refused, the context ended before a
	// Run opened), or what came back could not be observed (no cleanup, or a Run that does not
	// encode). Run and Verdict carry whatever the facade returned; the bridge was not told and the
	// candidate is still outstanding on it.
	OutcomeRunFailed OutcomeKind = "run-failed"
	// OutcomeCompleted: one Run exists with its cleanup observed. The campaign advanced only when
	// Credited is set; with an error beside the outcome the bridge could not be told (a rejected
	// frame, a broken bridge, a context that ended) and the candidate is still outstanding. The
	// facade may have returned an error beside the Run -- a recorder or Monitor close failure after
	// the Verdict was fixed -- which RunError carries; the Run is still the authoritative record.
	OutcomeCompleted OutcomeKind = "completed"
)

// Outcome is what one candidate came to. Run and Verdict are set whenever the facade returned a
// Run; Credited is set whenever the bridge was told and answered.
type Outcome struct {
	Kind     OutcomeKind
	Identity string
	// Detail names the rejection or the failure; empty for a completed Run.
	Detail   string
	Run      *testpilotspb.Run
	Verdict  *testpilotspb.Verdict
	Credited *Credited
	// RunError is the error the facade returned beside a closed Run, when it returned one; the
	// Run was observed regardless, because a proved Verdict is not erased by what followed it.
	RunError error
	// ReleaseError names what the candidate's teardown could not remove; it changes nothing about
	// the observation, whose cleanup status is the Run's own.
	ReleaseError error
}

// RunCandidate takes one outstanding candidate through the serial path: decode its Case, bind it
// (preparation first, before any Driver opens), run it once, observe its cleanup, and hand the
// closed Run back to the bridge. Exactly one Prepare or Run is in flight, and the bridge is told
// only what happened: a preparation rejection or the Run itself. A binding or execution failure
// leaves the candidate outstanding, because nothing honest can be observed for it.
func RunCandidate(ctx context.Context, bridge *Bridge, binder Binder, candidate *Candidate) (Outcome, error) {
	if bridge == nil || binder == nil || candidate == nil {
		return Outcome{}, errors.New("bridge, binder and candidate are required")
	}
	if outstanding := bridge.Outstanding(); outstanding == nil || outstanding.Identity != candidate.Identity {
		return Outcome{Identity: candidate.Identity}, ErrCrossedCandidate
	}
	source, err := testpilot.DecodeCaseProtoJSON(candidate.Case)
	if err != nil {
		return Outcome{Kind: OutcomeBindFailed, Identity: candidate.Identity, Detail: err.Error()},
			fmt.Errorf("decode candidate %s Case: %w", candidate.Identity, err)
	}
	if source.GetCaseId() != candidate.CaseID {
		return Outcome{Kind: OutcomeBindFailed, Identity: candidate.Identity, Detail: "Case ID differs"},
			&ProtocolError{Expected: "Case " + candidate.CaseID, Actual: "Case " + source.GetCaseId()}
	}
	bound, err := binder.Bind(ctx, bridge.profile, source)
	if err != nil {
		if rejection, ok := binding.IsPreparationRejection(err); ok {
			detail := fmt.Sprintf("%s at %s: %s", rejection.Category, rejection.Path, rejection.Detail)
			credited, err := bridge.Observe(ctx, candidate.Identity, Result{PrepareRejected: &detail})
			if err != nil {
				return Outcome{Kind: OutcomePrepareRejected, Identity: candidate.Identity, Detail: detail}, err
			}
			return Outcome{Kind: OutcomePrepareRejected, Identity: candidate.Identity, Detail: detail, Credited: &credited}, nil
		}
		return Outcome{Kind: OutcomeBindFailed, Identity: candidate.Identity, Detail: err.Error()}, err
	}
	run, verdict, runErr := bound.Run(ctx)
	// Teardown runs on its own context: an interrupted or timed-out Run still releases what it
	// opened, and what it could not remove is reported beside the outcome, never as the outcome.
	releaseErr := bound.Release(context.WithoutCancel(ctx))
	// The facade returns a closed Run beside an error when the failure came after the Verdict was
	// fixed; that Run is the authoritative record and is observed. Only no Run at all is a failure
	// to execute.
	if run == nil {
		if runErr == nil {
			runErr = errors.New("the Run returned nothing")
		}
		return Outcome{Kind: OutcomeRunFailed, Identity: candidate.Identity, Detail: runErr.Error(), ReleaseError: releaseErr},
			fmt.Errorf("run candidate %s: %w", candidate.Identity, runErr)
	}
	if run.GetCleanup() == nil {
		err := errors.New("the Run returned without an observed cleanup")
		return Outcome{Kind: OutcomeRunFailed, Identity: candidate.Identity, Detail: err.Error(), Run: run, Verdict: verdict, RunError: runErr, ReleaseError: releaseErr}, err
	}
	encoded, err := protojson.Marshal(run)
	if err != nil {
		return Outcome{Kind: OutcomeRunFailed, Identity: candidate.Identity, Detail: err.Error(), Run: run, Verdict: verdict, RunError: runErr, ReleaseError: releaseErr},
			fmt.Errorf("encode Run of candidate %s: %w", candidate.Identity, err)
	}
	credited, err := bridge.Observe(ctx, candidate.Identity, Result{Run: encoded})
	outcome := Outcome{Kind: OutcomeCompleted, Identity: candidate.Identity, Run: run, Verdict: verdict, RunError: runErr, ReleaseError: releaseErr}
	if err != nil {
		return outcome, err
	}
	outcome.Credited = &credited
	return outcome, nil
}
