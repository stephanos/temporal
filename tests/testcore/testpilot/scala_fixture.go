package testpilot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ScalaCase struct {
	Source     *testpilotspb.Case
	Bytes      []byte
	Assessment testpilot.AssessmentFactory
	// Property is the Query's Property: the claim a Run of the Case is assessed for.
	Property string
	Expected *lower.ExpectedRun
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

func LoadScalaCase(path string, query umpiremodel.ClaimKey, set string) (*ScalaCase, error) {
	model, err := umpiremodel.Load(path)
	if err != nil {
		return nil, err
	}
	producer, err := lower.NewProducer(model)
	if err != nil {
		return nil, err
	}
	lowered, err := producer.Lower(query.Name, lower.IdentityFor("temporal.case", set, query.Name))
	if err != nil {
		return nil, err
	}
	if lowered.Standing != lower.Lowered {
		return nil, fmt.Errorf("%s: %s: %v", query.Name, lowered.Standing, lowered.Unsupported)
	}
	return prepareScalaCase(model, query, lowered.Case)
}

func LoadGeneratedScalaCase(directory string, entry lower.GeneratedCase) (*ScalaCase, error) {
	encoded, err := os.ReadFile(filepath.Join(directory, entry.File))
	if err != nil {
		return nil, err
	}
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	if err != nil {
		return nil, err
	}
	model, err := umpiremodel.Load(filepath.Join(directory, "..", "ir", entry.Model))
	if err != nil {
		return nil, err
	}
	fixture, err := prepareScalaCase(model, entry.Query, source)
	if err != nil {
		return nil, err
	}
	fixture.Bytes, fixture.Expected = encoded, entry.Expected
	return fixture, nil
}

func ScalaManifest(directory string) ([]lower.GeneratedCase, error) {
	encoded, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return nil, err
	}
	manifest, err := lower.DecodeManifest(encoded)
	if err != nil {
		return nil, err
	}
	return manifest.Queries, nil
}

func prepareScalaCase(model *umpirespb.Model, query umpiremodel.ClaimKey, source *testpilotspb.Case) (*ScalaCase, error) {
	assessment, err := conformance.Prepare(model, query, source, conformance.Limits{
		MaxEvents: 2048, MaxProperties: 16, MaxDuration: time.Minute, MaxCandidates: 1 << 16, MaxWork: 1 << 22, MaxReadings: 1 << 22,
	})
	if err != nil {
		return nil, err
	}
	encoded, err := protojson.Marshal(source)
	if err != nil {
		return nil, err
	}
	var canonical bytes.Buffer
	if err := json.Compact(&canonical, encoded); err != nil {
		return nil, err
	}
	var property string
	for _, q := range model.GetQueries() {
		if q.GetName() == query.Name {
			property = q.GetProperty().GetName()
		}
	}
	return &ScalaCase{Source: source, Bytes: canonical.Bytes(), Assessment: assessment, Property: property, Durable: durableEvidence(model, query.Owner, source)}, nil
}

func durableEvidence(model *umpirespb.Model, owner string, source *testpilotspb.Case) []string {
	local := map[string]string{}
	for _, name := range source.GetProvenance().GetLocalNames() {
		local[name.GetDefinitionId()] = name.GetLocalName()
	}
	carried := map[string]bool{}
	for _, declared := range source.GetProgram().GetEvidence() {
		carried[declared.GetEvidenceId()] = true
	}
	var durable []string
	for _, realization := range model.GetRealizations() {
		if realization.GetMachine() != owner {
			continue
		}
		for _, e := range realization.GetEvidence() {
			if name := local[e.GetId()]; e.GetCommitment() == umpirespb.Evidence_COMMITMENT_DURABLE && carried[name] {
				durable = append(durable, name)
			}
		}
	}
	return durable
}
