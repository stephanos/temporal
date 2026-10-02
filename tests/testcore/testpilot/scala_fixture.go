package testpilot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/scalav2/goir"
	"go.temporal.io/server/model/scalav2/goir/conformance"
	goirtestpilot "go.temporal.io/server/model/scalav2/goir/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

type ScalaCase struct {
	Source     *testpilotspb.Case
	Bytes      []byte
	Assessment testpilot.AssessmentFactory
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
	return &ScalaCase{Source: lowered.Case, Bytes: canonical.Bytes(), Assessment: assessment}, nil
}
