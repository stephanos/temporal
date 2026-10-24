package temporal_test

import (
	"bytes"
	"context"
	"errors"
	"hash/fnv"
	"testing"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	model "umpire.local/model/host"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	temporalmodel "umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestEmbeddedProbeContextsSurviveResetAndIndependentHosts(t *testing.T) {
	hosts := []*model.Host{guest(t), guest(t)}
	coldPages := hosts[1].Pages()
	seen := make(map[[2]int]*annotationsv1.ProbeResponse)
	const genericCase = "generic::DescribeActivityExecution::temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.activity_id:mutation.empty"
	for hostIndex, host := range hosts {
		var warmPages uint32
		for round := range 3 {
			contextIndex := (hostIndex + round) % 2
			contexts := []struct{ namespace, activity, queue, run string }{
				{"synthetic-a", "activity-a", "queue-a", "00000000-0000-4000-8000-000000000001"},
				{"synthetic-b", "activity-b", "queue-b", "00000000-0000-4000-8000-000000000002"},
			}
			current := contexts[contextIndex]
			for mode, testCase := range []struct {
				id       string
				baseline bool
			}{{"", true}, {genericCase, true}, {genericCase, false}, {"RespondActivityTaskCompleted", true}} {
				request := &temporalv1.ProbeRequest{
					Operation: temporalv1.ProbeRequest_PREPARE, CaseId: testCase.id, Baseline: testCase.baseline,
					Namespace: current.namespace, ActivityId: current.activity, TaskQueue: current.queue,
					RunId: current.run, TaskToken: []byte{byte(contextIndex + 1)},
				}
				start := time.Now()
				prepared, err := temporalmodel.Probe(t.Context(), host, request)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("host=%d context=%d mode=%d round=%d duration=%s pages=%d", hostIndex, contextIndex, mode, round, time.Since(start), host.Pages())
				if prepared.Recipe != nil || len(prepared.Message) == 0 || proto.Size(prepared) > 1<<20 {
					t.Fatal("small contextual probe must return bounded request bytes")
				}
				switch mode {
				case 0:
					actual := new(workflowservice.StartActivityExecutionRequest)
					if err := proto.Unmarshal(prepared.Message, actual); err != nil {
						t.Fatal(err)
					}
					if actual.Namespace != current.namespace || actual.ActivityId != current.activity || actual.TaskQueue.GetName() != current.queue {
						t.Fatal("start request lost current context")
					}
				case 1, 2:
					actual := new(workflowservice.DescribeActivityExecutionRequest)
					if err := proto.Unmarshal(prepared.Message, actual); err != nil {
						t.Fatal(err)
					}
					activity := current.activity
					if mode == 2 {
						activity = ""
					}
					if actual.Namespace != current.namespace || actual.RunId != current.run || actual.ActivityId != activity {
						t.Fatal("generic request lost current context or mutation")
					}
				case 3:
					actual := new(workflowservice.RespondActivityTaskCompletedRequest)
					if err := proto.Unmarshal(prepared.Message, actual); err != nil {
						t.Fatal(err)
					}
					if actual.Namespace != current.namespace || !bytes.Equal(actual.TaskToken, request.TaskToken) {
						t.Fatal("completion request lost current namespace or task token")
					}
				default:
					t.Fatalf("unhandled probe test mode %d", mode)
				}
				key := [2]int{contextIndex, mode}
				if previous := seen[key]; previous != nil && !proto.Equal(previous, prepared) {
					t.Fatal("same-context response changed across calls, Reset or Hosts")
				}
				seen[key] = prepared
			}
			if _, err := host.Reset(t.Context(), &modelv1.ResetRequest{}); err != nil {
				t.Fatal(err)
			}
			pages := host.Pages()
			if pages > 1024 || (round > 0 && pages != warmPages) {
				t.Fatalf("retained memory must stay bounded after warm-up: before=%d after=%d", warmPages, pages)
			}
			warmPages = pages
			if hostIndex == 0 && hosts[1].Pages() != coldPages {
				t.Fatal("using one Host warmed the independent Host")
			}
		}
	}
}

type probeConnection struct{ invoked bool }

func (connection *probeConnection) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	connection.invoked = true
	return nil
}

func (*probeConnection) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("unexpected stream")
}

func TestProbeLocalRefusalNeverBecomesServerObservation(t *testing.T) {
	connection := new(probeConnection)
	response, err := temporalmodel.DispatchProbe(context.Background(), connection, &annotationsv1.ProbeResponse{
		Method: workflowservice.WorkflowService_StartActivityExecution_FullMethodName,
		Recipe: &annotationsv1.ProbeRecipe{DecodedLength: 1, Segments: []*annotationsv1.ProbeSegment{{Literal: []byte{'x'}, Repeat: 1}}},
	})
	if err == nil || response != nil || connection.invoked {
		t.Fatal("corrupt recipe must fail before invoking the server")
	}
}

func TestProbeRecipeExpansionAndRefusals(t *testing.T) {
	large := bytes.Repeat([]byte{'x'}, 2<<20+1)
	hash := fnv.New64a()
	_, _ = hash.Write(large)
	recipe := &annotationsv1.ProbeRecipe{
		Segments:      []*annotationsv1.ProbeSegment{{Literal: []byte{'x'}, Repeat: uint32(len(large))}},
		DecodedLength: uint32(len(large)), Fnv64: hash.Sum64(),
	}
	got, err := model.ExpandProbeRecipe(recipe)
	if err != nil || !bytes.Equal(got, large) {
		t.Fatal("actual oversized expansion failed")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*annotationsv1.ProbeRecipe)
	}{
		{"corruption", func(r *annotationsv1.ProbeRecipe) { r.Segments[0].Literal[0] = 'y' }},
		{"length", func(r *annotationsv1.ProbeRecipe) { r.DecodedLength++ }},
		{"budget", func(r *annotationsv1.ProbeRecipe) { r.DecodedLength = 16<<20 + 1 }},
		{"overflow", func(r *annotationsv1.ProbeRecipe) { r.Segments[0].Repeat = ^uint32(0) }},
		{"zero_repeat", func(r *annotationsv1.ProbeRecipe) { r.Segments[0].Repeat = 0 }},
		{"empty_literal", func(r *annotationsv1.ProbeRecipe) { r.Segments[0].Literal = nil }},
		{"segment_budget", func(r *annotationsv1.ProbeRecipe) { r.Segments = make([]*annotationsv1.ProbeSegment, 65537) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := proto.Clone(recipe).(*annotationsv1.ProbeRecipe)
			tc.mutate(bad)
			if _, err := model.ExpandProbeRecipe(bad); err == nil {
				t.Fatal("invalid recipe admitted")
			}
		})
	}
}

func TestProbeObservationPatternAndDetails(t *testing.T) {
	testCase := &annotationsv1.ProbeCase{Id: "case", Code: 3, MessagePattern: "too long|invalid"}
	for _, tc := range []struct {
		pattern string
		message string
		matches bool
	}{
		{"too long|invalid", "the input is too long (limit)", true},
		{"too long|invalid", "invalid duration\nseconds", true},
		{"too long|invalid", "different failure", false},
		{"^invalid$", "invalid", true},
		{"^invalid$", "prefix invalid", false},
		{"^invalid$", "invalid suffix", false},
	} {
		testCase.MessagePattern = tc.pattern
		observation, err := model.ObserveProbe(testCase, status.Error(codes.InvalidArgument, tc.message))
		if err != nil || observation.MessageMatches != tc.matches || observation.Code != 3 || observation.Accepted {
			t.Fatal("observation contract mismatch")
		}
	}
	testCase.MessagePattern = "["
	if _, err := model.ObserveProbe(testCase, status.Error(codes.InvalidArgument, "private-message")); err == nil || err.Error() != "invalid probe pattern for case" {
		t.Fatal("malformed pattern must refuse without raw pattern or message")
	}
	testCase.MessagePattern = "invalid"
	detailErr, err := status.New(codes.FailedPrecondition, "invalid").WithDetails(&realizationv1.Error{})
	if err != nil {
		t.Fatal(err)
	}
	observation, err := model.ObserveProbe(testCase, detailErr.Err())
	if err != nil || observation.DetailsType != "umpire.realization.v1.Error" || observation.Code != 9 {
		t.Fatal("status detail identity lost")
	}
	observation, err = model.ObserveProbe(testCase, nil)
	if err != nil || !observation.Accepted {
		t.Fatal("actual acceptance lost")
	}
}

func TestGenericProbeObservationMatchesEachPredictionWithoutValues(t *testing.T) {
	const private = "private-request-token-error"
	testCase := &annotationsv1.ProbeCase{
		Id: "generic-case", Generic: true,
		Predictions: []*annotationsv1.ProbeExpectation{
			{Id: "generic-case::prediction:first", MessagePattern: "missing identity"},
			{Id: "generic-case::prediction:second", MessagePattern: "different failure"},
		},
	}
	observation, err := model.ObserveProbe(testCase, status.Error(codes.InvalidArgument, "missing identity "+private))
	if err != nil || observation.PredictionCount != 2 || len(observation.Predictions) != 2 {
		t.Fatal("each prepared prediction requires an observation")
	}
	if !observation.Predictions[0].MessageMatches || observation.Predictions[1].MessageMatches || observation.Predictions[1].Index != 1 {
		t.Fatal("prediction patterns must be evaluated independently")
	}
	encoded, err := proto.Marshal(observation)
	if err != nil || bytes.Contains(encoded, []byte(private)) {
		t.Fatal("generic observations must remain value-free")
	}
	testCase.Predictions[1].MessagePattern = "["
	if _, err := model.ObserveProbe(testCase, status.Error(codes.InvalidArgument, private)); err == nil || err.Error() != "invalid probe pattern for generic-case" {
		t.Fatal("a malformed prediction pattern must fail without its contents")
	}
}

func TestProbeMetadataAdmissionAndRetiredWireFields(t *testing.T) {
	valid := &annotationsv1.ProbeCase{Id: "generic-case", Generic: true, Predictions: []*annotationsv1.ProbeExpectation{{Id: "generic-case::prediction:claim"}}}
	for _, change := range []func(*annotationsv1.ProbeCase){
		func(c *annotationsv1.ProbeCase) { c.Applicability = 99 },
		func(c *annotationsv1.ProbeCase) { c.Generic = false },
		func(c *annotationsv1.ProbeCase) { c.MessagePattern = "strict/generic mix" },
		func(c *annotationsv1.ProbeCase) {
			c.Predictions = append(c.Predictions, proto.Clone(c.Predictions[0]).(*annotationsv1.ProbeExpectation))
		},
		func(c *annotationsv1.ProbeCase) { c.Predictions[0].Id = "foreign-case::prediction:claim" },
		func(c *annotationsv1.ProbeCase) {
			c.Applicability = annotationsv1.ProbeApplicability_REFUSED
			c.Refusal = "local"
		},
	} {
		invalid := proto.Clone(valid).(*annotationsv1.ProbeCase)
		change(invalid)
		if _, err := model.ObserveProbe(invalid, nil); err == nil {
			t.Fatal("invalid probe metadata admitted")
		}
	}
	request := (&temporalv1.ProbeRequest{}).ProtoReflect().Descriptor()
	for _, number := range []protoreflect.FieldNumber{10, 11} {
		if !request.ReservedRanges().Has(number) {
			t.Fatal("retired request number reused")
		}
	}
	for _, name := range []protoreflect.Name{"uncovered_validators", "unresolved_validators"} {
		if !request.ReservedNames().Has(name) {
			t.Fatal("retired request name reused")
		}
	}
	caseDescriptor := (&annotationsv1.ProbeCase{}).ProtoReflect().Descriptor()
	if !caseDescriptor.ReservedRanges().Has(8) || !caseDescriptor.ReservedNames().Has("covers") {
		t.Fatal("retired mapping wire fields must stay reserved")
	}
}
