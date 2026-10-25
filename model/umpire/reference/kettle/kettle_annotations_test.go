package kettle_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	kettlev1 "umpire.local/kettle/gen/kettle/v1"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	model "umpire.local/model/host"
)

func (r *kettleRun) dispatchProbe(testCase *annotationsv1.ProbeCase, prepared *annotationsv1.ProbeResponse) (*annotationsv1.ProbeObservation, error) {
	if testCase.GetApplicability() != annotationsv1.ProbeApplicability_APPLICABLE {
		return model.ObserveProbe(testCase, nil)
	}
	wire := prepared.GetMessage()
	if prepared.GetRecipe() != nil {
		var err error
		wire, err = model.ExpandProbeRecipe(prepared.Recipe)
		if err != nil {
			return nil, err
		}
	}
	prediction, err := model.Annotate(r.ctx, r.host, &annotationsv1.AnnotationRequest{Method: prepared.Method, Message: wire, Seed: 233})
	if err != nil {
		return nil, err
	}
	if prediction.GetOutOfScope() != "" {
		return nil, fmt.Errorf("probe %s unexpectedly out of scope: %s", testCase.Id, prediction.OutOfScope)
	}
	r.t.Logf("probe %s annotation predicts %v; sending original bytes", testCase.Id, prediction.Violations)
	// Deliberate probes must reach the server even when annotation predicts rejection.
	ctx := context.WithValue(r.ctx, kettleProbeContextKey{}, true)
	response, rpcErr := model.DispatchProbe(ctx, r.connection, prepared, "umpire.reference.kettle.v1.KettleService")
	if response == nil {
		return nil, fmt.Errorf("probe %s local dispatch failure: %w", testCase.Id, rpcErr)
	}
	observation, err := kettleProbeObservation(testCase, rpcErr)
	if err != nil {
		return nil, err
	}
	if rpcErr == nil {
		actual, err := proto.Marshal(response)
		if err != nil {
			return nil, err
		}
		annotated, err := model.Annotate(r.ctx, r.host, &annotationsv1.AnnotationRequest{Method: prepared.Method, Response: true, Message: actual})
		if err != nil {
			return nil, err
		}
		if annotated.OutOfScope != "" || len(annotated.Violations) != 0 || !bytes.Equal(annotated.Message, actual) {
			return nil, errors.New("probe response annotation has violations or changed bytes or scope")
		}
	}
	return observation, nil
}

func kettleProbeObservation(testCase *annotationsv1.ProbeCase, rpcErr error) (*annotationsv1.ProbeObservation, error) {
	if rpcErr != nil {
		serverStatus, ok := status.FromError(rpcErr)
		if !ok {
			return nil, fmt.Errorf("probe %s transport failure: %w", testCase.GetId(), rpcErr)
		}
		switch serverStatus.Code() {
		case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unauthenticated:
		default:
			return nil, fmt.Errorf("probe %s inconclusive server status: %w", testCase.GetId(), rpcErr)
		}
	}
	return model.ObserveProbe(testCase, rpcErr)
}

func TestKettleProbeTransportFailuresAreNotFindings(t *testing.T) {
	testCase := &annotationsv1.ProbeCase{Id: "transport-control", Code: int32(codes.InvalidArgument)}
	for _, rpcErr := range []error{errors.New("connection lost"), status.Error(codes.Unknown, "unknown"), status.Error(codes.Unavailable, "unavailable"), status.Error(codes.DeadlineExceeded, "deadline"), status.Error(codes.Canceled, "canceled"), status.Error(codes.Internal, "internal")} {
		observation, err := kettleProbeObservation(testCase, rpcErr)
		if err == nil || observation != nil {
			t.Fatalf("transport error became a finding: %v: %v %v", rpcErr, observation, err)
		}
	}
	for _, rpcErr := range []error{nil, status.Error(codes.InvalidArgument, "kettle_id")} {
		if observation, err := kettleProbeObservation(testCase, rpcErr); err != nil || observation == nil {
			t.Fatalf("application response lost: %v %v", observation, err)
		}
	}
}

func TestKettleAnnotationCoverageAndActualWireGaps(t *testing.T) {
	r := newKettleRun(t)
	coverage, err := model.Probe(r.ctx, r.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_COVERAGE})
	if err != nil {
		t.Fatal(err)
	}
	compareSnapshot(t, "coverage.txt", coverage.Report)
	if !strings.Contains(coverage.Report, "Ignored") {
		t.Fatal("coverage must retain an Ignored field with its reason")
	}
	inventory, err := model.Probe(r.ctx, r.host, &kettlev1.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Cases) == 0 {
		t.Fatal("empty Kettle probe inventory")
	}
	seen := make(map[string]bool)
	var observations []*annotationsv1.ProbeObservation
	for _, testCase := range inventory.Cases {
		if testCase.Id == "" || seen[testCase.Id] {
			t.Fatalf("empty or duplicate probe identity: %q", testCase.Id)
		}
		seen[testCase.Id] = true
		t.Run(testCase.Id, func(t *testing.T) {
			if testCase.Applicability != annotationsv1.ProbeApplicability_APPLICABLE {
				observation, err := model.ObserveProbe(testCase, nil)
				if err != nil {
					t.Fatal(err)
				}
				observations = append(observations, observation)
				return
			}
			for _, baseline := range []bool{true, false} {
				t.Run(fmt.Sprintf("baseline=%t", baseline), func(t *testing.T) {
					witness := newKettleRun(t)
					switch testCase.Prerequisite {
					case "empty":
					case "filled", "heating":
						witness.fill()
						if testCase.Prerequisite == "heating" {
							witness.heat(true)
						}
					default:
						t.Fatalf("unsupported probe prerequisite: %q", testCase.Prerequisite)
					}
					prepared, err := model.Probe(witness.ctx, witness.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_PREPARE, CaseId: testCase.Id, Baseline: baseline, KettleId: "room"})
					if err != nil {
						t.Fatal(err)
					}
					if prepared.Method != testCase.Method || len(prepared.Cases) != 1 || prepared.Cases[0].Id != testCase.Id {
						t.Fatalf("prepared probe changed identity: %v", prepared)
					}
					observation, err := witness.dispatchProbe(prepared.Cases[0], prepared)
					if err != nil {
						t.Fatal(err)
					}
					if baseline {
						if !observation.Accepted {
							t.Fatalf("valid baseline rejected: %v", observation)
						}
						return
					}
					observations = append(observations, observation)
				})
			}
		})
	}
	if len(observations) != len(inventory.Cases) {
		t.Fatal("every probe must produce one actual observation or explicit local refusal")
	}
	classifyCtx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	report, err := model.Probe(classifyCtx, r.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_CLASSIFY, Observations: observations})
	if err != nil {
		t.Fatal(err)
	}
	if report.Fails || !strings.Contains(report.Report, "Unverifiable") {
		t.Fatalf("healthy gap report must pass and retain Unverifiable: %s", report.Report)
	}
	compareSnapshot(t, "gaps.txt", report.Report)
}
