package assessment

import (
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// ErrLost is an iteration with no closed Run: a process that ended before its Run closed. It has
// no record and no subject; reconcile reports it.
var ErrLost = errors.New("a lost iteration has no recorded Run")

// Admit records one iteration's closed Run under the pinned Case's identity and the Driver
// identity the Case was prepared under, and admits it with fn-26 against the tree's catalog. The
// record is held in memory for this call only: it carries whole history events, so it is never
// written. The Driver identity must carry the policy's Profile name, and the subject's must be the
// one the Run was recorded under. The recorded Verdict, disposition and cleanup are the subject's
// unchanged; authority, isolation and cleanup facts enter only the provenance.
func Admit(canary *policy.Policy, driver testpilot.DriverIdentity, run *testpilotspb.Run) (*evaluation.Subject, error) {
	if canary == nil {
		return nil, errors.New("a canary policy is required")
	}
	if run == nil {
		return nil, ErrLost
	}
	crossed := func(format string, arguments ...any) (*evaluation.Subject, error) {
		return nil, &evaluation.Rejection{Reason: evaluation.ReasonCrossed, Detail: fmt.Sprintf(format, arguments...)}
	}
	if driver.Profile != canary.CaseProfile {
		return crossed("the Run was prepared under Profile %q, the policy's is %q", driver.Profile, canary.CaseProfile)
	}
	caseIdentity, err := casebinding.Identity()
	if err != nil {
		return nil, err
	}
	if caseIdentity != canary.CaseIdentity {
		return crossed("the pinned Case is %s, the policy's is %s", caseIdentity, canary.CaseIdentity)
	}
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("build the method catalog: %w", err)
	}
	recorded, err := recordedrun.Encode(caseIdentity, driver, run)
	if err != nil {
		return nil, &evaluation.Rejection{Reason: evaluation.ReasonMalformed, Detail: err.Error()}
	}
	subject, err := evaluation.Admit(casebinding.Case(), recorded, catalog.Identity())
	if err != nil {
		return nil, err
	}
	if subject.Driver != driver {
		return crossed("the subject's Driver identity is not the one the Run was recorded under")
	}
	return subject, nil
}
