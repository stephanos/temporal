package temporal_test

import (
	"bytes"
	"context"
	"errors"
	"hash/fnv"
	"testing"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	temporalmodel "umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

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
	response, err := temporalmodel.DispatchProbe(context.Background(), connection, &temporalv1.ProbeResponse{
		Method: workflowservice.WorkflowService_StartActivityExecution_FullMethodName,
		Recipe: &temporalv1.ProbeRecipe{DecodedLength: 1, Segments: []*temporalv1.ProbeSegment{{Literal: []byte{'x'}, Repeat: 1}}},
	})
	if err == nil || response != nil || connection.invoked {
		t.Fatal("corrupt recipe must fail before invoking the server")
	}
}

func TestProbeRecipeExpansionAndRefusals(t *testing.T) {
	large := bytes.Repeat([]byte{'x'}, 2<<20+1)
	hash := fnv.New64a()
	_, _ = hash.Write(large)
	recipe := &temporalv1.ProbeRecipe{
		Segments:      []*temporalv1.ProbeSegment{{Literal: []byte{'x'}, Repeat: uint32(len(large))}},
		DecodedLength: uint32(len(large)), Fnv64: hash.Sum64(),
	}
	got, err := temporalmodel.ExpandProbeRecipe(recipe)
	if err != nil || !bytes.Equal(got, large) {
		t.Fatal("actual oversized expansion failed")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*temporalv1.ProbeRecipe)
	}{
		{"corruption", func(r *temporalv1.ProbeRecipe) { r.Segments[0].Literal[0] = 'y' }},
		{"length", func(r *temporalv1.ProbeRecipe) { r.DecodedLength++ }},
		{"budget", func(r *temporalv1.ProbeRecipe) { r.DecodedLength = 16<<20 + 1 }},
		{"overflow", func(r *temporalv1.ProbeRecipe) { r.Segments[0].Repeat = ^uint32(0) }},
		{"zero_repeat", func(r *temporalv1.ProbeRecipe) { r.Segments[0].Repeat = 0 }},
		{"empty_literal", func(r *temporalv1.ProbeRecipe) { r.Segments[0].Literal = nil }},
		{"segment_budget", func(r *temporalv1.ProbeRecipe) { r.Segments = make([]*temporalv1.ProbeSegment, 65537) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := proto.Clone(recipe).(*temporalv1.ProbeRecipe)
			tc.mutate(bad)
			if _, err := temporalmodel.ExpandProbeRecipe(bad); err == nil {
				t.Fatal("invalid recipe admitted")
			}
		})
	}
}

func TestProbeObservationPatternAndDetails(t *testing.T) {
	testCase := &temporalv1.ProbeCase{Id: "case", Code: 3, MessagePattern: "too long|invalid"}
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
		observation, err := temporalmodel.ObserveProbe(testCase, status.Error(codes.InvalidArgument, tc.message))
		if err != nil || observation.MessageMatches != tc.matches || observation.Code != 3 || observation.Accepted {
			t.Fatal("observation contract mismatch")
		}
	}
	testCase.MessagePattern = "["
	if _, err := temporalmodel.ObserveProbe(testCase, status.Error(codes.InvalidArgument, "private-message")); err == nil || err.Error() != "invalid probe pattern for case" {
		t.Fatal("malformed pattern must refuse without raw pattern or message")
	}
	testCase.MessagePattern = "invalid"
	detailErr, err := status.New(codes.FailedPrecondition, "invalid").WithDetails(&temporalv1.Error{})
	if err != nil {
		t.Fatal(err)
	}
	observation, err := temporalmodel.ObserveProbe(testCase, detailErr.Err())
	if err != nil || observation.DetailsType != "umpire.temporal.v1.Error" || observation.Code != 9 {
		t.Fatal("status detail identity lost")
	}
	observation, err = temporalmodel.ObserveProbe(testCase, nil)
	if err != nil || !observation.Accepted {
		t.Fatal("actual acceptance lost")
	}
}
