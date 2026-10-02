package testpilot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/scalav2/goir"
	"go.temporal.io/server/model/scalav2/goir/conformance"
	goirtestpilot "go.temporal.io/server/model/scalav2/goir/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ScalaCase struct {
	Source     *testpilotspb.Case
	Bytes      []byte
	Assessment testpilot.AssessmentFactory
	// Property is the Query's Property: the claim a Run of the Case is assessed for.
	Property string
	// Durable is the kinds of evidence the Case carries that its realization declares the record of a
	// durable commit, by their names in the Case.
	Durable []string
}

// WithoutDurableEvidence is a copy of a recorded Run with its durable-commit evidence taken out: the
// evidence of each durable kind, and the committed decision the Run Event that carried it recorded.
// What is left is what a Run without the commit observation would have recorded.
func (c *ScalaCase) WithoutDurableEvidence(run *testpilotspb.Run) (*testpilotspb.Run, error) {
	out := proto.CloneOf(run)
	for _, event := range out.GetEvents() {
		kept := event.GetObservations()[:0]
		for _, observation := range event.GetObservations() {
			evidence := &testpilotspb.CorrelatedEvidence{}
			if packed := observation.GetValue().GetMessageValue(); packed.MessageIs(evidence) {
				if err := packed.UnmarshalTo(evidence); err != nil {
					return nil, err
				}
				if slices.Contains(c.Durable, evidence.GetKind()) {
					continue
				}
			}
			kept = append(kept, observation)
		}
		if len(kept) == len(event.GetObservations()) {
			continue
		}
		event.Observations = kept
		if len(kept) == 0 {
			event.Observations = nil
		}
		if outcome := event.GetOutcome(); outcome != nil {
			outcome.DeliveryAdmission = nil
		}
	}
	return out, nil
}

func LoadScalaCase(path string, query goir.ClaimKey, set string) (*ScalaCase, error) {
	model, err := goir.Load(path)
	if err != nil {
		return nil, err
	}
	producer, err := goirtestpilot.NewProducer(model)
	if err != nil {
		return nil, err
	}
	lowered, err := producer.Lower(query.Name, cp.IdentityFor("temporal.case", set, query.Name))
	if err != nil {
		return nil, err
	}
	if lowered.Standing != goirtestpilot.Lowered {
		return nil, fmt.Errorf("%s: %s: %v", query.Name, lowered.Standing, lowered.Unsupported)
	}
	assessment, err := conformance.Prepare(model, query, lowered.Case, conformance.Limits{
		MaxEvents: 2048, MaxProperties: 16, MaxDuration: time.Minute, MaxCandidates: 1 << 16, MaxWork: 1 << 22, MaxReadings: 1 << 22,
	})
	if err != nil {
		return nil, err
	}
	encoded, err := protojson.Marshal(lowered.Case)
	if err != nil {
		return nil, err
	}
	var canonical bytes.Buffer
	if err := json.Compact(&canonical, encoded); err != nil {
		return nil, err
	}
	local := map[string]string{}
	for _, name := range lowered.Case.GetProvenance().GetLocalNames() {
		local[name.GetDefinitionId()] = name.GetLocalName()
	}
	carried := map[string]bool{}
	for _, declared := range lowered.Case.GetProgram().GetEvidence() {
		carried[declared.GetEvidenceId()] = true
	}
	var durable []string
	for _, realization := range model.GetRealizations() {
		for _, e := range realization.GetEvidence() {
			if name := local[e.GetId()]; e.GetCommitment() == modelirspb.Evidence_COMMITMENT_DURABLE && carried[name] {
				durable = append(durable, name)
			}
		}
	}
	var property string
	for _, q := range model.GetQueries() {
		if q.GetName() == query.Name {
			property = q.GetProperty().GetName()
		}
	}
	return &ScalaCase{Source: lowered.Case, Bytes: canonical.Bytes(), Assessment: assessment, Property: property, Durable: durable}, nil
}
