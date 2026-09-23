package replay

import (
	"context"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/casefile"
	"go.temporal.io/server/tools/umpire/internal/recordedrun"
	"google.golang.org/protobuf/proto"
)

// Rejection is why a subject was not admitted, before any target effect: the reason names the
// class, the detail what was found.
type Rejection struct {
	Reason string
	Detail string
}

func (r *Rejection) Error() string { return r.Reason + ": " + r.Detail }

// The rejection reasons, each its own.
const (
	ReasonNoncanonical = "noncanonical"
	ReasonCrossed      = "crossed"
	ReasonIncompatible = "incompatible"
	ReasonStale        = "stale"
	ReasonIncomplete   = "incomplete"
	ReasonMalformed    = "malformed"
	ReasonNonViolated  = "non-violated"
	ReasonUnsupported  = "unsupported"
	ReasonDuplicate    = "duplicate"
	ReasonReplay       = "replay"
)

func reject(reason, format string, arguments ...any) error {
	return &Rejection{Reason: reason, Detail: fmt.Sprintf(format, arguments...)}
}

// IsRejection reports the rejection an error carries, when it is one.
func IsRejection(err error) (*Rejection, bool) {
	var rejection *Rejection
	if errors.As(err, &rejection) {
		return rejection, true
	}
	return nil, false
}

// Preparer prepares the subject's Case under a Profile name, touching no deployment; the binding's
// Prepare is the real one and a test supplies its own.
type Preparer func(identity string, source *testpilotspb.Case) (*testpilot.PreparedCase, error)

// Subject is one admitted violated Run of one Case: the Case in its canonical bytes and their
// identity, the recorded Run with its Verdict and the identity it was prepared under, the prepared
// Case the offline replay evaluated, that replay's evaluation, and the violation key.
type Subject struct {
	// Identity is the SHA-256 of the canonical Case bytes: the Case's identity, never part of the key.
	Identity  string
	Canonical []byte
	Case      *testpilotspb.Case
	Run       *testpilotspb.Run
	Verdict   *testpilotspb.Verdict
	Driver    testpilot.DriverIdentity
	Prepared  *testpilot.PreparedCase
	Replay    *testpilot.Evaluation
	Key       ViolationKey
}

// Admit reads one Case and one recorded Run and admits them as a subject or rejects them with a
// reason, before any target effect: the Case must be canonical, the Run must be the Case's (recorded
// from these canonical Case bytes, with the Case's IDs; a record naming no Case is incompatible) and in
// the admissible violated form with every supporting sequence naming one event, the Case must
// prepare under the recorded Profile name with the recorded catalog and bindings, and the recorded
// Run replayed offline must give the recorded Verdict. The key is derived from that replay.
func Admit(ctx context.Context, caseInput, recordedInput []byte, prepare Preparer) (*Subject, error) {
	if prepare == nil {
		return nil, errors.New("a preparer is required")
	}
	canonical, err := casefile.Canonical(caseInput)
	if err != nil {
		return nil, reject(ReasonNoncanonical, "%s", err)
	}
	source, err := testpilot.DecodeCaseProtoJSON(canonical)
	if err != nil {
		return nil, reject(ReasonNoncanonical, "the Case does not decode: %s", err)
	}
	decoded, err := DecodeRecordedRun(recordedInput)
	if errors.Is(err, ErrRecordNamesNoCase) {
		return nil, reject(ReasonIncompatible, "%s", err)
	}
	if err != nil {
		return nil, reject(ReasonMalformed, "%s", err)
	}
	driver, run := decoded.Driver, decoded.Run
	identity := recordedrun.Digest(canonical)
	// IDs are names, not hashes: a Case regenerated under the same IDs is another Case, and a Run
	// of the older one is not its Run.
	if decoded.Case != identity {
		return nil, reject(ReasonCrossed, "the Run was recorded from Case %s, the Case is %s", decoded.Case, identity)
	}
	if crossed := recordedrun.Crossed(source, run); crossed != "" {
		return nil, reject(ReasonCrossed, "%s", crossed)
	}
	verdict := run.GetVerdict()
	if ok, class, detail := ViolatedForm(run, verdict); !ok {
		return nil, reject(class, "%s", detail)
	}
	if problem := recordedrun.CheckSupport(run, verdict); problem != nil {
		if problem.Problem == recordedrun.SupportRepeated {
			return nil, reject(ReasonDuplicate, "%s", problem)
		}
		return nil, reject(ReasonUnsupported, "%s", problem)
	}
	prepared, err := prepare(driver.Profile, source)
	if err != nil {
		// The Case's own static rejection is the model having moved since the Run was recorded;
		// anything else, a deployment flag disagreeing with the Case included, is a tooling failure.
		if rejection := (*testpilot.PreparationError)(nil); errors.As(err, &rejection) {
			return nil, reject(ReasonStale, "the Case no longer prepares under Profile %s: %s", driver.Profile, err)
		}
		return nil, fmt.Errorf("prepare the subject's Case: %w", err)
	}
	if under := prepared.Identity(); under != driver {
		return nil, reject(ReasonStale, "the Case prepares under %s/%s/%s, the Run was recorded under %s/%s/%s",
			under.Profile, under.Catalog, under.Bindings, driver.Profile, driver.Catalog, driver.Bindings)
	}
	replayed, evaluation, err := prepared.Evaluate(ctx, run)
	if err != nil {
		return nil, reject(ReasonReplay, "the recorded Run does not replay: %s", err)
	}
	if !proto.Equal(replayed, verdict) {
		return nil, reject(ReasonReplay, "the recorded Run replays to %s, its recorded Verdict is %s", replayed.GetStatus(), verdict.GetStatus())
	}
	return &Subject{
		Identity: identity, Canonical: canonical, Case: source,
		Run: run, Verdict: verdict, Driver: driver, Prepared: prepared, Replay: evaluation,
		Key: KeyOf(source, verdict, evaluation),
	}, nil
}
