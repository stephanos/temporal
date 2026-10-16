package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func TestDecodeCoordinatorMessagesForwardsProgressAndFinalResponse(t *testing.T) {
	payload := []byte("{\"type\":\"progress\",\"progress\":{\"Phase\":\"preparing\",\"campaign_path\":\"/campaign\",\"Selected\":3,\"Attempted\":0,\"Running\":0,\"Succeeded\":0,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"Artifacts\":null}}\n{\"type\":\"result\",\"response\":{\"CampaignResult\":{\"campaign_path\":\"/campaign\",\"SelectionCount\":3,\"Attempted\":3,\"Succeeded\":3,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"StopReason\":\"seeds_exhausted\",\"Artifacts\":null},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n")
	var progress []CampaignEvent
	response, err := decodeCoordinatorMessages(bytes.NewReader(payload), func(update CampaignEvent) error {
		progress = append(progress, update)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 1 || progress[0].Phase != ProgressPreparing || response.CampaignResult.Attempted != 3 {
		t.Fatalf("progress = %#v, response = %#v", progress, response)
	}
}

func TestDecodeCoordinatorMessagesRejectsProgressAfterResult(t *testing.T) {
	payload := []byte("{\"type\":\"result\",\"response\":{\"CampaignResult\":{},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n{\"type\":\"progress\",\"progress\":{\"Phase\":\"running\"}}\n")
	if _, err := decodeCoordinatorMessages(bytes.NewReader(payload), nil); err == nil {
		t.Fatal("decodeCoordinatorMessages() succeeded")
	}
}

func TestDecodeCoordinatorMessagesPreservesDrainFailure(t *testing.T) {
	drainFailure := errors.New("drain failed")
	input := &sequencedReader{reads: []readerResult{{data: []byte("x")}, {err: drainFailure}}}
	_, err := decodeCoordinatorMessages(input, nil)
	if err == nil || !errors.Is(err, drainFailure) {
		t.Fatalf("decodeCoordinatorMessages() error = %v", err)
	}
}

func (reader *sequencedReader) Read(output []byte) (int, error) {
	if len(reader.reads) == 0 {
		return 0, io.EOF
	}
	result := reader.reads[0]
	reader.reads = reader.reads[1:]
	return copy(output, result.data), result.err
}

func TestCoordinatorTransportCoversEveryCampaignSpecField(t *testing.T) {
	specType := reflect.TypeFor[CampaignSpec]()
	wireType := reflect.TypeFor[coordinatorConfig]()
	for index := range specType.NumField() {
		field := specType.Field(index)
		wireField, transported := wireType.FieldByName(field.Name)
		if _, local := coordinatorLocalOnlyFields[field.Name]; local {
			if transported {
				t.Errorf("CampaignSpec.%s is listed as local-only but is transported", field.Name)
			}
			continue
		}
		if !transported {
			t.Errorf("CampaignSpec.%s is neither transported to the coordinator nor listed as local-only", field.Name)
			continue
		}
		if wireField.Type != field.Type {
			t.Errorf("CampaignSpec.%s has type %s but is transported as %s", field.Name, field.Type, wireField.Type)
		}
	}
	for _, field := range reflect.VisibleFields(wireType) {
		if field.PkgPath != "" {
			continue
		}
		if _, found := specType.FieldByName(field.Name); !found {
			t.Errorf("coordinator transport field %s has no CampaignSpec field", field.Name)
		}
	}
	for name := range coordinatorLocalOnlyFields {
		if _, found := specType.FieldByName(name); !found {
			t.Errorf("local-only field %s is not a CampaignSpec field", name)
		}
	}
}

func TestCoordinatorTransportRoundTripsEveryTransportedField(t *testing.T) {
	const childTimeout = 987654321 * time.Nanosecond
	distinct := distinctCampaignSpec(t)
	falseOverride := false
	explicitFalse := distinct
	explicitFalse.GuideRegressionOverride = &falseOverride
	for _, test := range []struct {
		name string
		spec CampaignSpec
	}{
		{name: "distinct nonzero values", spec: distinct},
		{name: "zero values", spec: CampaignSpec{}},
		{name: "explicit false regression override", spec: explicitFalse},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(campaignRequestFromSpec(test.spec).coordinatorConfig(childTimeout))
			if err != nil {
				t.Fatal(err)
			}
			decoder := json.NewDecoder(bytes.NewReader(encoded))
			decoder.DisallowUnknownFields()
			var wire coordinatorConfig
			if err := decoder.Decode(&wire); err != nil {
				t.Fatal(err)
			}
			got := wire.campaignRequest(test.spec.Backend, test.spec.RunnerBuild)
			want := campaignRequestFromSpec(test.spec)
			want.OverallTimeout = childTimeout
			want.Progress, want.Preparer, want.executor, want.Replayer = nil, nil, nil, nil
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("coordinator transport changed:\n got %#v\nwant %#v", got, want)
			}
		})
	}
}

func distinctCampaignSpec(t *testing.T) CampaignSpec {
	t.Helper()
	var spec CampaignSpec
	next := uint64(100)
	value := reflect.ValueOf(&spec).Elem()
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if _, local := coordinatorLocalOnlyFields[field.Name]; local {
			continue
		}
		fillDistinct(t, field.Name, value.Field(index), &next)
	}
	return spec
}

func fillDistinct(t *testing.T, path string, value reflect.Value, next *uint64) {
	t.Helper()
	*next++
	switch value.Kind() {
	case reflect.String:
		value.SetString(fmt.Sprintf("%s-%d", path, *next))
	case reflect.Int, reflect.Int64:
		value.SetInt(int64(*next))
	case reflect.Uint64, reflect.Uint8:
		value.SetUint(*next)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		fillDistinct(t, path+"*", value.Elem(), next)
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 2, 2))
		for index := range value.Len() {
			fillDistinct(t, fmt.Sprintf("%s[%d]", path, index), value.Index(index), next)
		}
	case reflect.Struct:
		for index := range value.NumField() {
			field := value.Type().Field(index)
			if !field.IsExported() {
				t.Fatalf("%s.%s is unexported and cannot cross the coordinator transport", path, field.Name)
			}
			fillDistinct(t, path+"."+field.Name, value.Field(index), next)
		}
	default:
		t.Fatalf("%s has kind %s, which the coordinator transport test cannot populate", path, value.Kind())
	}
}

func differingExportedFields(got, want CampaignSpec) []string {
	gotValue, wantValue := reflect.ValueOf(got), reflect.ValueOf(want)
	var names []string
	for index := range gotValue.NumField() {
		field := gotValue.Type().Field(index)
		if field.IsExported() && !reflect.DeepEqual(gotValue.Field(index).Interface(), wantValue.Field(index).Interface()) {
			names = append(names, field.Name)
		}
	}
	return names
}

func simulationBounds(summary SimulationExplorationSummary) SimulationExplorationSummary {
	return SimulationExplorationSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxForcedDecisions: summary.MaxForcedDecisions,
		MaxExplorationBytes: summary.MaxExplorationBytes, MaxResultBytes: summary.MaxResultBytes, FailureBudget: summary.FailureBudget, Limits: summary.Limits,
	}
}

func canonicalEvidence(t *testing.T, summary CampaignResult) []byte {
	t.Helper()
	if summary.ExecutionEvidence == nil {
		t.Fatalf("campaign = %#v, want execution evidence", summary)
	}
	encoded, err := json.Marshal(*summary.ExecutionEvidence)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func sortedEnvironment(environment []record.Environment) []record.Environment {
	sorted := slices.Clone(environment)
	slices.SortFunc(sorted, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	return sorted
}

func coordinatorRequest(t *testing.T, config CampaignSpec, omit ...string) []byte {
	t.Helper()
	encoded, err := json.Marshal(campaignRequestFromSpec(config).coordinatorConfig(config.OverallTimeout))
	if err != nil {
		t.Fatal(err)
	}
	if len(omit) == 0 {
		return encoded
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		t.Fatal(err)
	}
	for _, name := range omit {
		if _, found := members[name]; !found {
			t.Fatalf("coordinator request has no member %q", name)
		}
		delete(members, name)
	}
	encoded, err = json.Marshal(members)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

type sequencedReader struct {
	reads []readerResult
}

type readerResult struct {
	data []byte
	err  error
}

var coordinatorLocalOnlyFields = map[string]string{
	"Backend":              "each process receives its configured provider",
	"RunnerBuild":          "the provider supplies its local Runner identity",
	"Progress":             "replaced in the child by a callback that forwards events to the parent",
	"Preparer":             "injected preparation is rejected for isolated campaigns",
	"Replayer":             "injected replay is rejected for isolated campaigns",
	"resumePreflight":      "the child opens its own resume preflight",
	"guidancePlan":         "selected locally after target preparation; resume restores it from the recorded plan",
	"failureArtifactLimit": "the child derives it from the campaign plan",
	"failureBytesLimit":    "the child derives it from the campaign plan",
}
