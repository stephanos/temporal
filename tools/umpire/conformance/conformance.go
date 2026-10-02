// Package conformance assesses a Testpilot Run against an admitted IR Model: whether a modeled
// execution explains the evidence the Run recorded, and what each claim of one Query reads on the
// executions that do. It is a caller's assessment beside a Case's Contract (testpilot.AssessedCase):
// it runs nothing, stops nothing and changes no Verdict.
//
// Evidence is partial. A Run records some of the facts a machine's steps record, each as correlated
// evidence of one declared kind, keyed to its operation and ordered only by its causal parents and by
// its ordinal within its source. The assessment keeps every execution of the machine, from the
// Query's Scenario's start, that those observations and that order admit:
//
//   - One observation is one fact of one step. Two observations no order relates may have happened
//     either way round, and both orders are kept. Observations that name different attempts or
//     deliveries are not facts of one step: the attempt and the delivery are what a field the
//     realization gives that role names, and what the Run Event that carried the evidence records of
//     the activity attempt it came from.
//   - A step that records a fact no observation reports may still have happened. Evidence proves
//     what it reports and nothing about what it does not: what a caller was told can go untold, a
//     durable commit can go unobserved, and a fact of a kind the Case does not carry cannot be seen at
//     all. The one exception is declared: a kind of evidence the realization declares exhaustive,
//     which the Case carries, on a Run that closed complete, once the read that closes its source has
//     succeeded and the source's ordinals are unbroken. A step that records a fact of such a kind then
//     has its observation, and an execution that takes one unobserved is ruled out. Short of all of
//     that, no execution is ever ruled out by evidence that is absent.
//
// Conformance fails when no execution is left and no hole of the Model was in reach of the ones
// tried; a hole in reach leaves it inconclusive. A claim is violated when every execution left
// violates it, satisfied when every one reads it and none violates it, and inconclusive otherwise:
// when they disagree, when one never reaches the claim's evaluation point, when a hole was in reach
// of an execution that had not violated it, and whenever the Run did not close complete. The claims
// are the Query's Property and the monitors of its machine.
//
// The executions are explored within explicit ceilings. One that is reached ends the assessment with
// an error that names it, which Testpilot reports as the Assessment's failure: what the events had
// already established stands, and nothing else is concluded from a part.
package conformance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Limits are the ceilings one assessment runs within. Each must be set: there is no default a result
// could silently depend on.
type Limits struct {
	// MaxEvents, MaxProperties and MaxDuration are Testpilot's own ceilings on an assessment.
	MaxEvents     int64
	MaxProperties int64
	MaxDuration   time.Duration
	// MaxCandidates is the most partial executions one reading of one operation's evidence keeps.
	MaxCandidates int
	// MaxWork is the most steps an assessment takes over all its readings.
	MaxWork int
	// MaxReadings is the most readings of a claim's function on a step or a monitor state that
	// preparing the factory makes: every claim is read on every step once, before any Run.
	MaxReadings int
}

const (
	maxCandidates = 1 << 20
	maxWork       = 1 << 26
	// maxEvidence is how many observations of one operation a reading orders.
	maxEvidence = 64
)

// LimitError is a ceiling an assessment reached, with the Run Event it was reading. Nothing is
// concluded from the part explored before it.
type LimitError struct {
	Resource string
	Ceiling  int
	Event    int64
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("run event %d: conformance %s ceiling of %d reached", e.Event, e.Resource, e.Ceiling)
}

// EvidenceError is evidence a Run recorded that cannot be read as the Case and the realization
// declare it, at the Run Event that carries it.
type EvidenceError struct {
	Event   int64
	Message string
}

func (e *EvidenceError) Error() string { return fmt.Sprintf("run event %d: %s", e.Event, e.Message) }

// Factory is one Query of one admitted Model bound to the Case that records its evidence. It is
// immutable, and every Assessor it gives is the state of one Run or one replay only.
type Factory struct {
	binding testpilot.AssessmentBinding
	plan    *plan
}

// Prepare binds a Query of a Model to a Case. The Model is admitted and snapshotted, the Query's
// machine and claims are read whole, and the Case must carry the evidence observation of the
// machine's one realization. A Query this reader does not assess, a claim that cannot be read and a
// ceiling out of range are errors here, before any Run.
func Prepare(m *modelirspb.Model, query umpiremodel.ClaimKey, source *testpilotspb.Case, limits Limits) (*Factory, error) {
	if m == nil || source == nil {
		return nil, &umpiremodel.Error{Message: "a Model and a Case are required"}
	}
	for _, ceiling := range []struct {
		name       string
		value, max int
	}{{"candidates", limits.MaxCandidates, maxCandidates}, {"work", limits.MaxWork, maxWork}, {"readings", limits.MaxReadings, maxWork}} {
		if ceiling.value < 1 || ceiling.value > ceiling.max {
			return nil, &umpiremodel.Error{Message: fmt.Sprintf("conformance %s ceiling must be between 1 and %d", ceiling.name, ceiling.max)}
		}
	}
	snapshot := proto.CloneOf(m)
	compiled, err := compile(snapshot, query, proto.CloneOf(source), limits)
	if err != nil {
		return nil, err
	}
	fingerprint, err := testpilot.CaseFingerprint(source)
	if err != nil {
		return nil, err
	}
	model, err := modelIdentity(snapshot)
	if err != nil {
		return nil, err
	}
	// The ceilings are part of what was asked: a Run assessed under others is not replayed under these.
	asked := sha256.Sum256(fmt.Appendf(nil, "goir.conformance/v1\x00%s\x00%s\x00%s\x00%s\x00%d\x00%d", query.Family, query.Owner, query.Name,
		compiled.realization, limits.MaxCandidates, limits.MaxWork))
	identity := fmt.Sprintf("%s/%s/%s#%s", query.Family, query.Owner, query.Name, hex.EncodeToString(asked[:8]))
	if len(identity) > 256 {
		return nil, &umpiremodel.Error{Message: "query identity " + identity + " is longer than 256 bytes"}
	}
	return &Factory{plan: compiled, binding: testpilot.AssessmentBinding{Case: fingerprint, Model: model, Query: identity,
		Limits: testpilot.AssessmentLimits{MaxEvents: limits.MaxEvents, MaxProperties: limits.MaxProperties, MaxDuration: limits.MaxDuration}}}, nil
}

// Binding implements testpilot.AssessmentFactory.
func (f *Factory) Binding() testpilot.AssessmentBinding { return f.binding }

// New implements testpilot.AssessmentFactory.
func (f *Factory) New(context.Context) (testpilot.Assessor, error) { return newAssessor(f.plan), nil }

// modelIdentity is a Model's content with every source position taken out, so that moving a
// declaration in its file moves no identity.
func modelIdentity(m *modelirspb.Model) (string, error) {
	bare := proto.CloneOf(m)
	bare.Source = ""
	clearPositions(bare.ProtoReflect())
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(bare)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(append([]byte("goir.model/v1"), encoded...))
	return "goir.model/v1:sha256:" + hex.EncodeToString(digest[:]), nil
}

var positionName = (&modelirspb.Position{}).ProtoReflect().Descriptor().FullName()

func clearPositions(m protoreflect.Message) {
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.Message() == nil || field.IsMap() {
			return true
		}
		switch {
		case field.Message().FullName() == positionName:
			m.Clear(field)
		case field.IsList():
			for i := range value.List().Len() {
				clearPositions(value.List().Get(i).Message())
			}
		default:
			clearPositions(value.Message())
		}
		return true
	})
}
