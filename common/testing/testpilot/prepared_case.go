package testpilot

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/verification"
	"google.golang.org/protobuf/proto"
)

// Evaluation is what an offline replay says beyond the Verdict: for each violated rule, the Run
// Event whose evidence resolved it and that evidence, read off the Contract's own evaluation of
// the recorded events rather than searched for.
type Evaluation struct {
	Violations []RuleViolation
	// Assessment is what a fresh Assessor concludes from the recorded Run, when an AssessedCase
	// evaluated it; it is not part of the Verdict.
	Assessment *Assessment
}

// RuleViolation names what violated one rule. A monitor rule carries the observation ids of the
// violating event, or none when its deadline violated it; a correlated rule carries the kind of
// the evidence whose release resolved the obligation and the event that carried that evidence,
// or neither (Sequence 0, no kind) when the violation was found at closure with the obligation
// still pending under a final ending.
type RuleViolation = verification.Violation

// ConcludeVerdict is the one aggregation of a closed Run's rule statuses into its Verdict status
// and the disposition that Verdict leaves the Run in, the one the Monitor and the offline replay
// conclude through. Readers of recorded Runs check a Verdict against it rather than restating it.
var ConcludeVerdict = execution.Conclude

// Evaluate replays a closed Run's events through the same prepared Contract the Monitor ran, with
// no Driver and no target, and returns the Verdict that reading gives with its Evaluation. A Run
// that is not closed, or that names another Program, errs.
func (p *PreparedCase) Evaluate(ctx context.Context, run *testpilotspb.Run) (*testpilotspb.Verdict, *Evaluation, error) {
	if p == nil || p.contract == nil || ir.IsNil(ctx) {
		return nil, nil, errors.New("prepared Case and context are required")
	}
	verdict, violations, err := p.contract.Evaluate(ctx, proto.CloneOf(run))
	if err != nil {
		return verdict, nil, err
	}
	return verdict, &Evaluation{Violations: violations}, nil
}

func (p *PreparedCase) Run(ctx context.Context, driver Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	return p.run(ctx, driver, nil)
}

// run is Run with observer, when there is one, beside the Contract's Monitor. open creates it once
// the Run is admitted and before the Driver is opened.
func (p *PreparedCase) run(ctx context.Context, driver Driver, open func() (execution.EventObserver, error)) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	executionDriver, monitor, err := p.preflight(ctx, driver)
	if err != nil {
		return nil, nil, err
	}
	if open != nil {
		observer, err := open()
		if err != nil {
			return nil, nil, err
		}
		if monitor, err = execution.Observed(monitor, observer); err != nil {
			return nil, nil, err
		}
	}
	return execution.Run(ctx, p.program, executionDriver, monitor, RunIDPrefix+uuid.NewString(), p.source.GetCaseId())
}

// RunIDPrefix begins every Run ID Run chooses; the rest is a canonical UUID.
const RunIDPrefix = "testpilot.run."

// IsRunID reports whether id is a Run ID in the form Run chooses: RunIDPrefix and a lower-case,
// dashed UUID, and nothing else.
func IsRunID(id string) bool {
	suffix, ok := strings.CutPrefix(id, RunIDPrefix)
	if !ok {
		return false
	}
	parsed, err := uuid.Parse(suffix)
	return err == nil && parsed.String() == suffix
}
