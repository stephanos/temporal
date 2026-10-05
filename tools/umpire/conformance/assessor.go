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
	// closings is the last outcome the Run recorded of each closing read, and recorded the ordinals of
	// each source: what an exhaustive source is closed by.
	closings map[coordinate]closingOutcome
	recorded map[string]*ordinals
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
	// run is the activity run the Run Events that carried its evidence record, once one does.
	run role
}

func newAssessor(p *plan) *assessor {
	return &assessor{plan: p, byName: map[string]*instance{}, named: map[string]*observation{}, awaited: map[string]*observation{},
		closings: map[coordinate]closingOutcome{}, recorded: map[string]*ordinals{}}
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
	a.completion(event)
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

// completion records what an instruction's own completion says of a closing read: the Run Event that
// carries its outcome, which is no reservation's record and none of the events its response reads
// emit. A later completion of the same instruction replaces an earlier one.
func (a *assessor) completion(event *testpilotspb.RunEvent) {
	if kind := event.GetKind(); kind != testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED && kind != testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT {
		return
	}
	at := coordinate{entrypoint: event.GetCoordinates().GetEntrypointId(), instruction: event.GetCoordinates().GetInstructionId()}
	if !a.plan.closing[at] || event.GetOutcome() == nil {
		return
	}
	a.closings[at] = readFailed
	if event.GetOutcome().GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		a.closings[at] = readSucceeded
	}
}

// closed is the facts whose exhaustive source is closed on this Run, by the rule of sourceClosed.
func (a *assessor) closed(positive bool) map[string]bool {
	closed := map[string]bool{}
	for _, k := range a.plan.reader.kinds {
		if k.closing != nil && sourceClosed(positive, a.closings[*k.closing], a.recorded[k.source].unbroken()) {
			closed[k.records] = true
		}
	}
	return closed
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
	// Where the realization declares an activity's attempts of one run, one operation is one activity
	// run: evidence two Run Events record under two runs is crossed.
	if a.plan.reader.oneRun && seen.run.known && of.run.known && seen.run.id != of.run.id {
		return nil, &EvidenceError{Event: seen.sequence, Message: fmt.Sprintf("evidence of operation %s on a Run Event of activity run %q, and the operation's evidence is of activity run %q",
			of.name, seen.run.id, of.run.id)}
	}
	if seen.run.known {
		of.run = seen.run
	}
	of.evidence = append(of.evidence, seen)
	a.named[seen.identity] = seen
	if a.recorded[seen.source] == nil {
		a.recorded[seen.source] = &ordinals{}
	}
	a.recorded[seen.source].record(seen.ordinal)
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
		said := a.because(of, why)
		a.nonconformance = &testpilot.ConformanceAssessment{Status: conformance, SupportingEventSequences: of.support(), Reason: said.id(), Detail: said.detail}
		established.Nonconformance = a.nonconformance
	}
	for i, c := range a.plan.claims {
		conclusion, why := claimConclusion(of.open.tallies[i], false)
		if conclusion != testpilot.PropertyViolated || slices.ContainsFunc(a.violations, func(v testpilot.PropertyAssessment) bool { return v.ID == c.id }) {
			continue
		}
		said := a.because(of, why)
		violation := testpilot.PropertyAssessment{ID: c.id, Status: conclusion, SupportingEventSequences: of.support(),
			Reason: said.id(), Detail: said.detail}
		a.violations = append(a.violations, violation)
		established.Violations = append(established.Violations, violation)
	}
	return established
}

// because is why as it is said of the instance of: the reason, after the machine and the instance.
func (a *assessor) because(of *instance, why reason) because {
	if why == none {
		return because{}
	}
	return because{reason: why, detail: fmt.Sprintf("%s, %s: %s", a.plan.machine, of.name, wording[why])}
}

// Close implements testpilot.Assessor. A Run that closed complete is read once more as a whole, so
// that a monitor read at the end of a path is read, and so that an exhaustive source its closing read
// closed says what did not happen. One that did not close complete keeps what its events established,
// and concludes nothing else. No reading concludes anything from the absence of evidence whose source
// is not closed.
func (a *assessor) Close(ctx context.Context, closure testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	positive := a.failure == nil && !a.frozen && closure.Disposition == testpilotspb.RUN_DISPOSITION_COMPLETED && closure.EvaluationFailureSequence == 0

	conformances, reasons := make([]testpilot.ConformanceStatus, 0, len(a.instances)), make([]because, 0, len(a.instances))
	claims, claimReasons := make([][]testpilot.PropertyStatus, len(a.plan.claims)), make([][]because, len(a.plan.claims))
	var support []int64
	closed := a.closed(positive)
	for _, of := range a.instances {
		support = append(support, of.support()...)
		read := of.open
		if positive {
			related, err := order(of.evidence, a.named)
			if err != nil {
				return nil, err
			}
			event := of.evidence[len(of.evidence)-1].sequence
			if read, err = a.plan.explore(related, regime{ended: true, event: event, closed: closed}, &a.spent); err != nil {
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
		conformances, reasons = append(conformances, conformance), append(reasons, a.because(of, why))
		for i := range a.plan.claims {
			conclusion, why := claimConclusion(read.tallies[i], positive)
			claims[i], claimReasons[i] = append(claims[i], conclusion), append(claimReasons[i], a.because(of, why))
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
		outcome.Conformance = testpilot.ConformanceAssessment{Status: status, Reason: why.id(), Detail: why.detail}
	default:
		outcome.Conformance = testpilot.ConformanceAssessment{Status: status, SupportingEventSequences: support, Reason: why.id(), Detail: why.detail}
	}
	for i, c := range a.plan.claims {
		if at := slices.IndexFunc(a.violations, func(v testpilot.PropertyAssessment) bool { return v.ID == c.id }); at >= 0 {
			outcome.Properties = append(outcome.Properties, a.violations[at])
			continue
		}
		status, why := over(claims[i], claimReasons[i], testpilot.PropertyViolated, testpilot.PropertySatisfied, testpilot.PropertyInconclusive)
		assessed := testpilot.PropertyAssessment{ID: c.id, Status: status, Reason: why.id(), Detail: why.detail}
		if status != testpilot.PropertyInconclusive {
			assessed.SupportingEventSequences = support
		}
		outcome.Properties = append(outcome.Properties, assessed)
	}
	return outcome, nil
}
