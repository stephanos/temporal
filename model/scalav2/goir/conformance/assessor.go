package conformance

import (
	"context"
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

// assessor is the assessment state of one Run or one replay.
type assessor struct {
	testpilot.SingleUse
	plan *plan

	// instances are the machine instances the Run recorded evidence of, in the order it first named
	// each.
	instances []*instance
	// byName finds an instance without reading the others, however many operations a Run has.
	byName map[string]*instance
	named  map[string]*observation
	// awaited is, by a causal parent's identity not yet recorded, an observation that names it.
	awaited map[string]*observation
	// scope is the one scope the Run's evidence is recorded under, set by the first piece.
	scope string
	// frozen says execution became incomplete: as for the Contract, no later event is read.
	frozen bool
	spent  int
	// nonconformance and violations are what Observe has reported as established.
	nonconformance *testpilot.ConformanceAssessment
	violations     []testpilot.PropertyAssessment
	// failure is the error that ended the reading of events.
	failure error
}

// instance is one instance of the machine: one operation of one Run scope.
type instance struct {
	name     string
	evidence []*observation
	// open is the latest reading of its evidence that concludes nothing from what is absent.
	open *survey
}

func newAssessor(p *plan) *assessor {
	return &assessor{plan: p, byName: map[string]*instance{}, named: map[string]*observation{}, awaited: map[string]*observation{}}
}

func (i *instance) support() []int64 {
	support := make([]int64, 0, len(i.evidence))
	for _, e := range i.evidence {
		support = append(support, e.sequence)
	}
	slices.Sort(support)
	return slices.Compact(support)
}

// Observe implements testpilot.Assessor. It reads the evidence the event carries and reports what
// the evidence so far establishes for good: read without concluding anything from what is absent,
// which more evidence can only narrow.
func (a *assessor) Observe(ctx context.Context, event *testpilotspb.RunEvent) (testpilot.Established, error) {
	if a.failure != nil {
		return testpilot.Established{}, a.failure
	}
	established, err := a.observe(ctx, event)
	if err != nil {
		a.failure = err
		return testpilot.Established{}, err
	}
	return established, nil
}

func (a *assessor) observe(ctx context.Context, event *testpilotspb.RunEvent) (testpilot.Established, error) {
	if err := ctx.Err(); err != nil {
		return testpilot.Established{}, err
	}
	if event == nil {
		return testpilot.Established{}, &EvidenceError{Message: "no Run Event"}
	}
	if a.frozen = a.frozen || event.GetExecutionIncomplete(); a.frozen {
		return testpilot.Established{}, nil
	}
	seen, err := a.plan.reader.read(event)
	if err != nil || seen == nil {
		return testpilot.Established{}, err
	}
	of, err := a.admit(seen)
	if err != nil || of == nil {
		return testpilot.Established{}, err
	}
	related, err := order(of.evidence, a.named)
	if err != nil {
		return testpilot.Established{}, err
	}
	if of.open, err = a.plan.explore(related, regime{event: seen.sequence}, &a.spent); err != nil {
		return testpilot.Established{}, err
	}
	return a.established(of), nil
}

// admit takes one observation into the instance it is of, or returns no instance for evidence the Run
// had recorded before. Evidence that contradicts what the Run recorded is refused: another scope, an
// identity recorded with other content, a causal parent of another operation.
func (a *assessor) admit(seen *observation) (*instance, error) {
	if a.scope == "" {
		a.scope = seen.scope
	}
	if seen.scope != a.scope {
		return nil, &EvidenceError{Event: seen.sequence, Message: fmt.Sprintf("evidence under scope %s, and the Run's evidence is under %s", seen.scope, a.scope)}
	}
	if earlier, republished := a.named[seen.identity]; republished {
		if proto.Equal(earlier.recorded, seen.recorded) {
			return nil, nil
		}
		return nil, &EvidenceError{Event: seen.sequence, Message: "evidence " + seen.identity + " is recorded twice, with different content"}
	}
	of, known := a.byName[seen.instance]
	if !known {
		of = &instance{name: seen.instance}
		a.byName[seen.instance] = of
		a.instances = append(a.instances, of)
	}
	if len(of.evidence) >= maxEvidence {
		return nil, &LimitError{Resource: "evidence of one operation", Ceiling: maxEvidence, Event: seen.sequence}
	}
	of.evidence = append(of.evidence, seen)
	a.named[seen.identity] = seen
	// A parent recorded for another operation is found where the later of the two is read, whichever
	// that is, without reading the other operations again.
	if child, awaited := a.awaited[seen.identity]; awaited && child.instance != seen.instance {
		return nil, &EvidenceError{Event: seen.sequence, Message: fmt.Sprintf("evidence %s names %s, evidence of another operation, as its causal parent",
			child.identity, seen.identity)}
	}
	for _, parent := range seen.after {
		if _, recorded := a.named[parent]; recorded {
			continue
		}
		if other, awaited := a.awaited[parent]; awaited && other.instance != seen.instance {
			return nil, &EvidenceError{Event: seen.sequence, Message: fmt.Sprintf("evidence %s and %s, of two operations, name one causal parent %s",
				other.identity, seen.identity, parent)}
		}
		a.awaited[parent] = seen
	}
	return of, nil
}

// established is what an instance's latest reading settles for good and had not been reported.
func (a *assessor) established(of *instance) testpilot.Established {
	var established testpilot.Established
	if conformance, why := conformanceConclusion(of.open.candidates, of.open.holes, false); conformance == testpilot.ConformanceNonconformant && a.nonconformance == nil {
		a.nonconformance = &testpilot.ConformanceAssessment{Status: conformance, SupportingEventSequences: of.support(), Detail: a.detail(of, why)}
		established.Nonconformance = a.nonconformance
	}
	for i, c := range a.plan.claims {
		conclusion, why := claimConclusion(of.open.tallies[i], false)
		if conclusion != testpilot.PropertyViolated || slices.ContainsFunc(a.violations, func(v testpilot.PropertyAssessment) bool { return v.ID == c.id }) {
			continue
		}
		violation := testpilot.PropertyAssessment{ID: c.id, Status: conclusion, SupportingEventSequences: of.support(),
			Detail: a.detail(of, why)}
		a.violations = append(a.violations, violation)
		established.Violations = append(established.Violations, violation)
	}
	return established
}

func (a *assessor) detail(of *instance, why string) string {
	if why == "" {
		return ""
	}
	return fmt.Sprintf("%s, %s: %s", a.plan.machine, of.name, why)
}

// Close implements testpilot.Assessor. A Run that closed complete is read once more as a whole, so
// that a monitor read at the end of a path is read. One that did not keeps what its events
// established, and concludes nothing else. Neither reading concludes anything from evidence that is
// absent.
func (a *assessor) Close(ctx context.Context, closure testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	positive := a.failure == nil && !a.frozen && closure.Disposition == testpilotspb.RUN_DISPOSITION_COMPLETED && closure.EvaluationFailureSequence == 0

	conformances, reasons := make([]testpilot.ConformanceStatus, 0, len(a.instances)), make([]string, 0, len(a.instances))
	claims, claimReasons := make([][]testpilot.PropertyStatus, len(a.plan.claims)), make([][]string, len(a.plan.claims))
	var support []int64
	for _, of := range a.instances {
		support = append(support, of.support()...)
		read := of.open
		if positive {
			related, err := order(of.evidence, a.named)
			if err != nil {
				return nil, err
			}
			event := of.evidence[len(of.evidence)-1].sequence
			if read, err = a.plan.explore(related, regime{ended: true, event: event}, &a.spent); err != nil {
				return nil, err
			}
		}
		if read == nil {
			// The reading of events ended before this instance was read once.
			read = &survey{candidates: 1, tallies: make([]tally, len(a.plan.claims))}
			for i := range read.tallies {
				read.tallies[i].count[unread] = 1
			}
		}
		conformance, why := conformanceConclusion(read.candidates, read.holes, positive)
		conformances, reasons = append(conformances, conformance), append(reasons, a.detail(of, why))
		for i := range a.plan.claims {
			conclusion, why := claimConclusion(read.tallies[i], positive)
			claims[i], claimReasons[i] = append(claims[i], conclusion), append(claimReasons[i], a.detail(of, why))
		}
	}
	slices.Sort(support)
	support = slices.Compact(support)

	outcome := &testpilot.AssessmentOutcome{}
	status, why := over(conformances, reasons, testpilot.ConformanceNonconformant, testpilot.ConformanceConformant, testpilot.ConformanceInconclusive)
	switch {
	case a.nonconformance != nil:
		outcome.Conformance = *a.nonconformance
	case status == testpilot.ConformanceInconclusive:
		outcome.Conformance = testpilot.ConformanceAssessment{Status: status, Detail: why}
	default:
		outcome.Conformance = testpilot.ConformanceAssessment{Status: status, SupportingEventSequences: support, Detail: why}
	}
	for i, c := range a.plan.claims {
		if at := slices.IndexFunc(a.violations, func(v testpilot.PropertyAssessment) bool { return v.ID == c.id }); at >= 0 {
			outcome.Properties = append(outcome.Properties, a.violations[at])
			continue
		}
		status, why := over(claims[i], claimReasons[i], testpilot.PropertyViolated, testpilot.PropertySatisfied, testpilot.PropertyInconclusive)
		assessed := testpilot.PropertyAssessment{ID: c.id, Status: status, Detail: why}
		if status != testpilot.PropertyInconclusive {
			assessed.SupportingEventSequences = support
		}
		outcome.Properties = append(outcome.Properties, assessed)
	}
	return outcome, nil
}
