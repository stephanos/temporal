package testpilot

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/verification"
	"google.golang.org/protobuf/proto"
)

// Evaluation is what an offline replay says beyond the Verdict: for each violated rule, the Run
// Event whose evidence resolved it and that evidence, read off the Contract's own evaluation of
// the recorded events rather than searched for.
type Evaluation struct {
	Violations []RuleViolation
}

// RuleViolation names what violated one rule. A monitor rule carries the observation ids of the
// violating event, or none when its deadline violated it; a correlated rule carries the kind of
// the evidence whose release resolved the obligation and the event that carried that evidence,
// or neither (Sequence 0, no kind) when the violation was found at closure with the obligation
// still pending under a final ending.
type RuleViolation struct {
	RuleID         string
	Sequence       int64
	ObservationIDs []string
	CorrelatedKind string
}

// Evaluate replays a closed Run's events through the same prepared Contract the Monitor ran, with
// no Driver and no target, and returns the Verdict that reading gives with its Evaluation. A Run
// that is not closed, or that names another Program, errs.
func (p *PreparedCase) Evaluate(ctx context.Context, run *testpilotspb.Run) (*testpilotspb.Verdict, *Evaluation, error) {
	if p == nil || p.factory == nil || isNil(ctx) {
		return nil, nil, errors.New("prepared Case and context are required")
	}
	replayer, ok := p.factory.(interface {
		Evaluate(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, []verification.Violation, error)
	})
	if !ok {
		return nil, nil, errors.New("prepared Contract cannot be replayed")
	}
	verdict, violations, err := replayer.Evaluate(ctx, proto.CloneOf(run))
	if err != nil {
		return verdict, nil, err
	}
	evaluation := &Evaluation{}
	for _, violation := range violations {
		evaluation.Violations = append(evaluation.Violations, RuleViolation{
			RuleID: violation.RuleID, Sequence: violation.Sequence,
			ObservationIDs: append([]string(nil), violation.ObservationIDs...), CorrelatedKind: violation.Kind,
		})
	}
	return verdict, evaluation, nil
}

func (p *PreparedCase) Run(ctx context.Context, driver Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	executionDriver, monitor, err := p.preflight(ctx, driver)
	if err != nil {
		return nil, nil, err
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
