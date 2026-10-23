package model

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"umpire.local/model/reference/kettle"
	kettlev1 "umpire.local/model/reference/kettle/v1"
	annotationsv1 "umpire.local/model/umpire/annotations/v1"
	modelv1 "umpire.local/model/umpire/model/v1"
	realizationv1 "umpire.local/model/umpire/realization/v1"
)

type kettleMutantWitness struct {
	name, driver, caseID, finding string
	code                          codes.Code
}

func kettleMutantWitnesses() []kettleMutantWitness {
	return []kettleMutantWitness{
		{"overlong-id", "Probe", "generic::Fill::umpire.reference.kettle.v1.FillRequest.kettle_id:mutation.limit_plus_one(32)", "umpire.reference.kettle.v1.FillRequest.kettle_id: string.max_bytes: Accepted", codes.OK},
		{"premature-boil", "Judge", "", "Unexplained", codes.OK},
		{"removed-validator", "Probe", "generic::Fill::umpire.reference.kettle.v1.FillRequest.options.labels.key:mutation.empty", "umpire.reference.kettle.v1.FillRequest.FillOptions.labels: bytes.min_len: Accepted", codes.OK},
		{"wrong-status", "Probe", "generic::Fill::umpire.reference.kettle.v1.FillRequest.kettle_id:mutation.limit_plus_one(32)", "umpire.reference.kettle.v1.FillRequest.kettle_id: string.max_bytes: Mismatch", codes.FailedPrecondition},
	}
}

func kettleWitnessNames(names []string, witnesses []kettleMutantWitness) error {
	expected := make([]string, len(witnesses))
	for i, witness := range witnesses {
		expected[i] = witness.name
	}
	names = slices.Clone(names)
	slices.Sort(names)
	slices.Sort(expected)
	if !slices.Equal(names, expected) {
		return fmt.Errorf("mutant witness names differ: mutants=%v witnesses=%v", names, expected)
	}
	return nil
}

func kettleMutantCorpus(t *testing.T, r *kettleRun) []*annotationsv1.ProbeObservation {
	t.Helper()
	inventory, err := Probe(r.ctx, r.host, &kettlev1.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var observations []*annotationsv1.ProbeObservation
	for _, testCase := range inventory.Cases {
		r.renew(t)
		witness := r
		if testCase.Applicability == annotationsv1.ProbeApplicability_APPLICABLE {
			witness = newKettleRun(t)
			switch testCase.Prerequisite {
			case "filled":
				witness.fill()
			case "heating":
				witness.fill()
				witness.heat(true)
			case "empty":
			default:
				t.Fatalf("unknown prerequisite: %s", testCase.Prerequisite)
			}
		}
		prepared, err := Probe(witness.ctx, witness.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_PREPARE, CaseId: testCase.Id, KettleId: "room"})
		if err != nil {
			t.Fatal(err)
		}
		observation, err := witness.dispatchProbe(prepared.Cases[0], prepared)
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, observation)
	}
	return observations
}

// renew bounds the shared corpus run's next probe or classification on its own, so the whole
// mutant sweep is not held to one run's deadline.
func (r *kettleRun) renew(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	r.ctx = ctx
}

func kettleClassifyWitness(t *testing.T, r *kettleRun, corpus []*annotationsv1.ProbeObservation, actual *annotationsv1.ProbeObservation) *annotationsv1.ProbeResponse {
	t.Helper()
	observations := slices.Clone(corpus)
	found := false
	for i, observation := range observations {
		if observation.CaseId == actual.CaseId {
			observations[i], found = actual, true
		}
	}
	if !found {
		t.Fatalf("witness absent from corpus: %s", actual.CaseId)
	}
	r.renew(t)
	report, err := Probe(r.ctx, r.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_CLASSIFY, Observations: observations})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func kettleDetected(w kettleMutantWitness, healthy, mutant *annotationsv1.ProbeObservation, report *annotationsv1.ProbeResponse) error {
	if proto.Equal(healthy, mutant) || !report.Fails || !strings.Contains(report.Report, w.finding) {
		return fmt.Errorf("undetected mutant %s by %s: healthy=%v mutant=%v fails=%t", w.name, w.driver, healthy, mutant, report.Fails)
	}
	return nil
}

func kettleJudgeDetected(w kettleMutantWitness, healthy, mutant *kettlev1.DescribeResponse, explained, unexplained *modelv1.JudgeResponse) error {
	if proto.Equal(healthy, mutant) || explained.GetExplained() == nil || unexplained.GetUnexplained() == nil {
		return fmt.Errorf("undetected mutant %s by %s: actual=%v judgment=%v", w.name, w.driver, mutant, unexplained)
	}
	return nil
}

func kettlePrematureWitness(t *testing.T, mutant string) (*kettlev1.DescribeResponse, *modelv1.JudgeResponse) {
	t.Helper()
	r := newKettleRunWithOptions(t, kettle.Options{Mutant: mutant})
	r.fill()
	r.heat(true)
	r.poll(20, kettlev1.HeatingAction_HEATING_ACTION_HEAT)
	request := &kettlev1.DescribeRequest{KettleId: "room"}
	response, err := r.client.Describe(r.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	observation := &realizationv1.Observation{Method: "/umpire.reference.kettle.v1.KettleService/Describe", Request: kettlePack(t, request), Result: &realizationv1.Observation_Response{Response: kettlePack(t, response)}}
	judgment, err := r.host.Judge(r.ctx, &modelv1.JudgeRequest{Observation: kettlePack(t, observation)})
	if err != nil {
		t.Fatal(err)
	}
	return response, judgment
}

func TestKettleMutants(t *testing.T) {
	witnesses := kettleMutantWitnesses()
	if err := kettleWitnessNames(kettle.Mutants(), witnesses); err != nil {
		t.Fatal(err)
	}
	r := newKettleRun(t)
	corpus := kettleMutantCorpus(t, r)
	var golden strings.Builder
	for _, w := range witnesses {
		t.Run(w.name, func(t *testing.T) {
			if w.driver == "Judge" {
				healthy, explained := kettlePrematureWitness(t, "")
				mutant, unexplained := kettlePrematureWitness(t, w.name)
				if !proto.Equal(healthy.State, &kettlev1.State{Temperature: 20, HeatingOn: true}) || explained.GetExplained() == nil || len(explained.GetExplained().Steps) != 0 {
					t.Fatalf("healthy current-only judgment: %v %v", healthy, explained)
				}
				if err := kettleJudgeDetected(w, healthy, mutant, explained, unexplained); err != nil {
					t.Fatal(err)
				}
				if !proto.Equal(mutant.State, &kettlev1.State{Temperature: 100, Beeping: true}) {
					t.Fatalf("wrong actual mutant projection: %v", mutant)
				}
				noop, noopJudgment := kettlePrematureWitness(t, "")
				if err := kettleJudgeDetected(w, healthy, noop, explained, noopJudgment); err == nil || !strings.Contains(err.Error(), w.name) || !strings.Contains(err.Error(), w.driver) {
					t.Fatalf("healthy-equivalent no-op counted as detection: %v", err)
				}
				fmt.Fprintf(&golden, "%s -> %s -> %s\n", w.name, w.driver, w.finding)
				return
			}
			healthyRun := newKettleRun(t)
			mutantRun := newKettleRunWithOptions(t, kettle.Options{Mutant: w.name})
			prepared, err := Probe(healthyRun.ctx, healthyRun.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_PREPARE, CaseId: w.caseID, KettleId: "room"})
			if err != nil {
				t.Fatal(err)
			}
			testCase := prepared.Cases[0]
			if len(testCase.Predictions) != 1 || testCase.Predictions[0].Code != int32(codes.InvalidArgument) {
				t.Fatalf("wrong declared rejection: %v", testCase)
			}
			request := new(kettlev1.FillRequest)
			if err := proto.Unmarshal(prepared.Message, request); err != nil || request.IdempotencyKey == "" {
				t.Fatalf("probe requires filled idempotency ID: %v %v", request, err)
			}
			original := bytes.Clone(prepared.Message)
			healthy, err := healthyRun.dispatchProbe(testCase, prepared)
			if err != nil {
				t.Fatal(err)
			}
			mutant, err := mutantRun.dispatchProbe(testCase, prepared)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original, prepared.Message) {
				t.Fatal("differential witness bytes changed")
			}
			healthyReport := kettleClassifyWitness(t, r, corpus, healthy)
			if healthy.Accepted || healthy.Code != int32(codes.InvalidArgument) || healthy.PredictionCount != 1 || healthyReport.Fails {
				t.Fatalf("healthy control: %v %v", healthy, healthyReport)
			}
			noopRun := newKettleRun(t)
			noop, err := noopRun.dispatchProbe(testCase, prepared)
			if err != nil {
				t.Fatal(err)
			}
			noopReport := kettleClassifyWitness(t, r, corpus, noop)
			if err := kettleDetected(w, healthy, noop, noopReport); err == nil || !strings.Contains(err.Error(), w.name) || !strings.Contains(err.Error(), w.driver) {
				t.Fatalf("healthy-equivalent no-op counted as detection: %v", err)
			}
			report := kettleClassifyWitness(t, r, corpus, mutant)
			if err := kettleDetected(w, healthy, mutant, report); err != nil {
				t.Fatal(err)
			}
			if mutant.Accepted != (w.code == codes.OK) || mutant.Code != int32(w.code) || mutant.PredictionCount != 1 || len(mutant.Predictions) != 1 {
				t.Fatalf("wrong actual response: %v", mutant)
			}
			start := strings.Index(report.Report, "Generic case: "+w.caseID+"\n")
			if start < 0 {
				t.Fatal("framework omitted witness identity")
			}
			line := strings.SplitN(report.Report[start:], "\n", 3)[1]
			if !strings.HasPrefix(line, w.finding) {
				t.Fatalf("wrong framework finding: %s", line)
			}
			fmt.Fprintf(&golden, "%s -> %s %s -> %s\n", w.name, w.driver, w.caseID, line)
		})
	}
	if t.Failed() {
		return
	}
	kettleAnnotationGolden(t, "mutants.snap.txt", golden.String())
}

func TestKettleMutantMissingWitness(t *testing.T) {
	if err := kettleWitnessNames(append(kettle.Mutants(), "new-unwitnessed-mutant"), kettleMutantWitnesses()); err == nil || !strings.Contains(err.Error(), "new-unwitnessed-mutant") {
		t.Fatalf("missing witness accepted: %v", err)
	}
}

func TestKettleMutantBrokenTransport(t *testing.T) {
	r := newKettleRun(t)
	prepared, err := Probe(r.ctx, r.host, &kettlev1.ProbeRequest{Operation: kettlev1.ProbeRequest_PREPARE, CaseId: kettleMutantWitnesses()[0].caseID, KettleId: "room"})
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(r.ctx)
	cancel()
	response, rpcErr := DispatchProbe(canceled, r.connection, prepared, "umpire.reference.kettle.v1.KettleService")
	if response == nil || rpcErr == nil {
		t.Fatalf("transport control did not attempt RPC: %v %v", response, rpcErr)
	}
	observation, err := kettleProbeObservation(prepared.Cases[0], rpcErr)
	if err == nil || observation != nil {
		t.Fatalf("broken actual transport became finding: %v %v", observation, err)
	}
}
