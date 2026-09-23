package replay

import (
	"context"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
	"google.golang.org/protobuf/proto"
)

// Attempts is how many fresh reruns classify a subject or a candidate: two, so that one Run's
// answer is never the pair's alone.
const Attempts = 2

// Attempt is one fresh rerun classed alone: its class with the detail behind it, the key its own
// Run derived when the Run is in the admissible violated form, the Run and Verdict, and the
// identity it was bound under.
type Attempt struct {
	Class    Class
	Detail   string
	Key      ViolationKey
	Run      *testpilotspb.Run
	Verdict  *testpilotspb.Verdict
	Identity testpilot.DriverIdentity
}

// Reruns is what the fresh reruns say about one key: the key every attempt was compared to, each
// attempt in order, and the pair's class by precedence. It is a value with no listener, as the
// campaign's Outcome is, and carries nothing about history replay, which has no type here.
type Reruns struct {
	Key      ViolationKey
	Attempts []Attempt
	Class    Class
}

// Target is what fresh reruns bind and compare against: a Case, the Case prepared under the
// Profile identity every attempt must bind to again, and the key every attempt is compared to. A
// subject's target is its own; a reduction's candidate is its own Case and identity with the
// subject's key.
type Target struct {
	Case     *testpilotspb.Case
	Prepared *testpilot.PreparedCase
	Driver   testpilot.DriverIdentity
	Key      ViolationKey
}

// Target is the subject's own Case, identity and key.
func (s *Subject) Target() Target {
	return Target{Case: s.Case, Prepared: s.Prepared, Driver: s.Driver, Key: s.Key}
}

// Rerun binds the target's Case fresh under its exact Profile identity, once per attempt, runs it,
// releases the binding before the next attempt binds, and classifies the attempts against the
// target's key. The binder is one already open against a deployment; opening is the caller's,
// after admission, so nothing rejected or stale at admission reaches a Run. A binding that fails
// or arrives at another identity is an error, since preparation was decided at admission and a
// deployment that prepares otherwise is not the subject's; a Run that errs is classed
// indeterminate, as an incomplete Run is; a release that fails is an error, since the next attempt
// would not be isolated from it.
func Rerun(ctx context.Context, binder campaign.Binder, target Target) (*Reruns, error) {
	if binder == nil || target.Case == nil || target.Prepared == nil {
		return nil, errors.New("a binder and a prepared target are required")
	}
	reruns := &Reruns{Key: target.Key}
	for attempt := range Attempts {
		result, err := rerunOnce(ctx, binder, target)
		if err != nil {
			return nil, fmt.Errorf("attempt %d: %w", attempt+1, err)
		}
		reruns.Attempts = append(reruns.Attempts, result)
	}
	classes := make([]Class, 0, len(reruns.Attempts))
	for _, attempt := range reruns.Attempts {
		classes = append(classes, attempt.Class)
	}
	reruns.Class = ClassifyPair(classes...)
	return reruns, nil
}

// RerunOnce binds, runs, releases and classes one fresh attempt of the target: the retry a pair
// with an indeterminate Run spends one Run on.
func RerunOnce(ctx context.Context, binder campaign.Binder, target Target) (Attempt, error) {
	if binder == nil || target.Case == nil || target.Prepared == nil {
		return Attempt{}, errors.New("a binder and a prepared target are required")
	}
	return rerunOnce(ctx, binder, target)
}

func rerunOnce(ctx context.Context, binder campaign.Binder, target Target) (Attempt, error) {
	bound, err := binder.Bind(ctx, target.Driver.Profile, target.Case)
	if err != nil {
		return Attempt{}, fmt.Errorf("bind %s: %w", target.Driver.Profile, err)
	}
	// Release on a context the caller's cancellation does not reach, as the campaign does: a Run
	// stopped by cancellation still closes, and its binding must still be torn down.
	release := func() error { return bound.Release(context.WithoutCancel(ctx)) }
	identity := bound.Identity()
	if identity != target.Driver {
		releaseErr := release()
		return Attempt{}, errors.Join(fmt.Errorf("the deployment prepared %s as %s/%s/%s, not the target's %s/%s/%s",
			target.Driver.Profile, identity.Profile, identity.Catalog, identity.Bindings,
			target.Driver.Profile, target.Driver.Catalog, target.Driver.Bindings), releaseErr)
	}
	run, verdict, runErr := bound.Run(ctx)
	if err := release(); err != nil {
		return Attempt{}, fmt.Errorf("release: %w", err)
	}
	attempt := Attempt{Run: run, Verdict: verdict, Identity: identity}
	if runErr != nil {
		attempt.Class, attempt.Detail = ClassIndeterminate, "the Run did not close: "+runErr.Error()
		return attempt, nil
	}
	attempt.Class, attempt.Detail, attempt.Key = classify(ctx, target, run, verdict)
	return attempt, nil
}

// classify derives the Run's key through the target's prepared Case, the same Contract under the
// same identity, when the Run is in the violated form, and classes it; a Run the offline replay
// cannot reproduce has no key and is indeterminate.
func classify(ctx context.Context, target Target, run *testpilotspb.Run, verdict *testpilotspb.Verdict) (Class, string, ViolationKey) {
	if ok, _, _ := ViolatedForm(run, verdict); !ok {
		class, detail := Classify(target.Key, run, verdict, ViolationKey{})
		return class, detail, ViolationKey{}
	}
	replayed, evaluation, err := target.Prepared.Evaluate(ctx, run)
	if err != nil {
		return ClassIndeterminate, "the offline replay failed: " + err.Error(), ViolationKey{}
	}
	if !proto.Equal(replayed, verdict) {
		return ClassIndeterminate, "the offline replay does not give the Run's Verdict", ViolationKey{}
	}
	key := KeyOf(target.Case, verdict, evaluation)
	class, detail := Classify(target.Key, run, verdict, key)
	return class, detail, key
}
