package testpilot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/ir"
	irpaths "go.temporal.io/server/tools/umpire/ir/paths"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ModelCase struct {
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
func (c *ModelCase) WithoutDurableEvidence(run *testpilotspb.Run) (*testpilotspb.Run, error) {
	out := proto.CloneOf(run)
	for _, event := range out.GetEvents() {
		kept := event.GetObservations()[:0]
		for _, observation := range event.GetObservations() {
			evidence := &testpilotspb.CorrelatedEvidence{}
			if packed := observation.GetValue().GetObjectValue(); packed.MessageIs(evidence) {
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

func LoadModelCase(path string, query check.ClaimKey, set string) (*ModelCase, error) {
	model, err := ir.Load(path)
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
	return prepareModelCase(model, query, lowered.Case)
}

func LoadGeneratedCase(directory string, entry lower.GeneratedCase) (*ModelCase, error) {
	encoded, err := os.ReadFile(irpaths.CasePath(directory, entry.Model, entry.Query.Name))
	if err != nil {
		return nil, err
	}
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	if err != nil {
		return nil, err
	}
	model, err := ir.Load(irpaths.ModelPath(directory, entry.Model))
	if err != nil {
		return nil, err
	}
	fixture, err := prepareModelCase(model, entry.Query, source)
	if err != nil {
		return nil, err
	}
	fixture.Bytes, fixture.Expected = encoded, entry.Expected
	return fixture, nil
}

// GeneratedCaseName is the name a lowered Case runs under in the functional tests:
// `<model>-<query>`. It is the depth-2 test name the functional job shards and the salt optimizer
// times, so a renamed Query renames a shard unit.
func GeneratedCaseName(entry lower.GeneratedCase) string {
	return strings.TrimSuffix(entry.Model, ".json") + "-" + entry.Query.Name
}

func GeneratedCases(directory string) ([]lower.GeneratedCase, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	var entries []lower.GeneratedCase
	found := false
	err = filepath.WalkDir(irpaths.FeatureRoot(directory), func(path string, item os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() || item.Name() != "manifest.json" {
			return nil
		}
		found = true
		encoded, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest, err := lower.DecodeManifest(encoded)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		prefix, err := filepath.Rel(directory, filepath.Dir(path))
		if err != nil {
			return err
		}
		for _, entry := range manifest.Queries {
			if entry.File != "" {
				entry.File = filepath.Join(prefix, entry.File)
			}
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no generated Case manifest.json in %s: %w", directory, os.ErrNotExist)
	}
	slices.SortFunc(entries, func(a, b lower.GeneratedCase) int {
		if order := strings.Compare(a.Model, b.Model); order != 0 {
			return order
		}
		return strings.Compare(a.Query.Name, b.Query.Name)
	})
	return entries, nil
}

func prepareModelCase(model *umpirespb.Model, query check.ClaimKey, source *testpilotspb.Case) (*ModelCase, error) {
	assessment, err := conformance.Prepare(model, query, source, conformance.DefaultLimits())
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
	return &ModelCase{Source: source, Bytes: canonical.Bytes(), Assessment: assessment, Property: property, Durable: durableEvidence(model, query.Owner, source)}, nil
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
