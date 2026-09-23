package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/casefile"
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
// reason, before any target effect: the Case must be canonical, the Run must be the Case's and in
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
	driver, run, err := DecodeRecordedRun(recordedInput)
	if err != nil {
		return nil, reject(ReasonMalformed, "%s", err)
	}
	if run.GetCaseId() != source.GetCaseId() {
		return nil, reject(ReasonCrossed, "the Run names Case %q, the Case is %q", run.GetCaseId(), source.GetCaseId())
	}
	if run.GetProgramId() != source.GetProgram().GetProgramId() {
		return nil, reject(ReasonCrossed, "the Run names Program %q, the Case's is %q", run.GetProgramId(), source.GetProgram().GetProgramId())
	}
	verdict := run.GetVerdict()
	if ok, class, detail := ViolatedForm(run, verdict); !ok {
		return nil, reject(class, "%s", detail)
	}
	if err := checkSupport(run, verdict); err != nil {
		return nil, err
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
	if identity := prepared.Identity(); identity != driver {
		return nil, reject(ReasonStale, "the Case prepares under %s/%s/%s, the Run was recorded under %s/%s/%s",
			identity.Profile, identity.Catalog, identity.Bindings, driver.Profile, driver.Catalog, driver.Bindings)
	}
	replayed, evaluation, err := prepared.Evaluate(ctx, run)
	if err != nil {
		return nil, reject(ReasonReplay, "the recorded Run does not replay: %s", err)
	}
	if !proto.Equal(replayed, verdict) {
		return nil, reject(ReasonReplay, "the recorded Run replays to %s, its recorded Verdict is %s", replayed.GetStatus(), verdict.GetStatus())
	}
	digest := sha256.Sum256(canonical)
	return &Subject{
		Identity: hex.EncodeToString(digest[:]), Canonical: canonical, Case: source,
		Run: run, Verdict: verdict, Driver: driver, Prepared: prepared, Replay: evaluation,
		Key: KeyOf(source, verdict, evaluation),
	}, nil
}

// checkSupport requires every supporting sequence, of the Verdict and of each rule, to name one
// event of the Run, once.
func checkSupport(run *testpilotspb.Run, verdict *testpilotspb.Verdict) error {
	check := func(owner string, sequences []int64) error {
		seen := map[int64]bool{}
		for _, sequence := range sequences {
			if sequence <= 0 || sequence > int64(len(run.GetEvents())) || run.GetEvents()[sequence-1].GetSequence() != sequence {
				return reject(ReasonUnsupported, "%s names supporting event %d, which the Run does not carry", owner, sequence)
			}
			if seen[sequence] {
				return reject(ReasonDuplicate, "%s names supporting event %d twice", owner, sequence)
			}
			seen[sequence] = true
		}
		return nil
	}
	if err := check("the Verdict", verdict.GetSupportingEventSequences()); err != nil {
		return err
	}
	for _, rule := range verdict.GetRules() {
		if err := check("rule "+rule.GetRuleId(), rule.GetSupportingEventSequences()); err != nil {
			return err
		}
	}
	return nil
}
