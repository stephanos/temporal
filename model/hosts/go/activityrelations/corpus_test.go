package activityrelations_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	realizationv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/realization/v1"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/temporal/activityrelations"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

const (
	service = "/temporal.api.workflowservice.v1.WorkflowService/"
	runA    = "22222222-2222-4222-8222-222222222222"
	runB    = "33333333-3333-4333-8333-333333333333"
)

func describeResponse(run string, attempt int32) *workflowservice.DescribeActivityExecutionResponse {
	return &workflowservice.DescribeActivityExecutionResponse{
		RunId: run,
		Info:  &activitypb.ActivityExecutionInfo{ActivityId: "activity", Status: 1, Attempt: attempt},
	}
}

type call struct {
	label    string
	method   string
	request  proto.Message
	response proto.Message
	err      error
}

// controlCalls is a probe-shaped call stream: two cases whose Start and Describe differ only in
// the echoed run identity, a crossed Describe, a Describe with no receipt, unbound and failed
// calls, cleanup and an unlabelled scope check.
func controlCalls() []call {
	start := &workflowservice.StartActivityExecutionRequest{Namespace: "ns", ActivityId: "activity"}
	describe := func(run string) *workflowservice.DescribeActivityExecutionRequest {
		return &workflowservice.DescribeActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: run}
	}
	return []call{
		{"a", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runA, Started: true}, nil},
		{"a", "DescribeActivityExecution", describe(runA), describeResponse(runA, 1), nil},
		{"a", "RequestCancelActivityExecution", &workflowservice.RequestCancelActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: runA}, &workflowservice.RequestCancelActivityExecutionResponse{}, nil},
		{"a", "TerminateActivityExecution", &workflowservice.TerminateActivityExecutionRequest{}, &workflowservice.TerminateActivityExecutionResponse{}, nil},
		{"b", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runB, Started: true}, nil},
		{"b", "DescribeActivityExecution", describe(runB), describeResponse(runB, 1), nil},
		{"b", "DescribeActivityExecution", describe(runB), describeResponse(runB, 2), nil},
		{"b", "DescribeActivityExecution", describe(runA), describeResponse(runA, 1), nil},
		{"b", "PollActivityTaskQueue", &workflowservice.PollActivityTaskQueueRequest{}, &workflowservice.PollActivityTaskQueueResponse{TaskToken: []byte("synthetic-token")}, nil},
		{"b", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{}, status.Error(codes.AlreadyExists, "exists")},
		{"c", "DescribeActivityExecution", describe(runA), describeResponse(runA, 1), nil},
		{"", "DescribeActivityExecution", describe(runA), describeResponse(runA, 1), nil},
	}
}

func capture(t *testing.T, collector *activityrelations.Collector, calls []call) {
	t.Helper()
	for _, c := range calls {
		ctx := context.Background()
		if c.label != "" {
			ctx = activityrelations.WithCase(ctx, "probe", c.label, 0)
		}
		reply := c.response.ProtoReflect().Type().New().Interface()
		invoke := func(_ context.Context, _ string, _, out any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			if c.err == nil {
				proto.Merge(out.(proto.Message), c.response)
			}
			return c.err
		}
		err := collector.Intercept(ctx, service+c.method, c.request, reply, nil, invoke)
		if status.Code(err) != status.Code(c.err) {
			t.Fatalf("%s: interceptor changed the result: %v", c.method, err)
		}
	}
}

func TestInterceptorCountsEveryCallUnderReceiptCustody(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	capture(t, collector, controlCalls())
	rows := collector.Rows()
	if len(rows) != len(controlCalls()) {
		t.Fatalf("captured %d of %d calls", len(rows), len(controlCalls()))
	}
	for _, row := range rows {
		if strings.Contains(row.Response, "73796e7468657469632d746f6b656e") {
			t.Fatal("an unbound task token reached the corpus")
		}
	}
	report := collector.Count(rows, "control")
	start := report.Methods["temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"]
	describe := report.Methods["temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"]
	cancel := report.Methods["temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"]
	poll := report.Methods["temporal.api.workflowservice.v1.WorkflowService/PollActivityTaskQueue"]
	terminate := report.Methods["temporal.api.workflowservice.v1.WorkflowService/TerminateActivityExecution"]
	for name, check := range map[string][2]any{
		"start success":            {start.Success, 2},
		"start rpc error":          {start.RPCError, 1},
		"start raw distinct":       {start.RawDistinct, 2},
		"start projected distinct": {start.ProjectedDistinct, 1},
		"describe attempted":       {describe.Attempted, 6},
		"describe success":         {describe.Success, 3},
		"describe refused":         {describe.Refused, 2},
		"describe excluded":        {describe.Excluded, 1},
		"describe raw distinct":    {describe.RawDistinct, 3},
		"describe projected":       {describe.ProjectedDistinct, 2},
		"cancel projected":         {cancel.ProjectedDistinct, 1},
		"poll not applicable":      {poll.NotApplicable, 1},
		"cleanup excluded":         {terminate.Excluded, 1},
		"total attempted":          {report.Total.Attempted, 12},
	} {
		if check[0] != check[1] {
			t.Errorf("%s = %v, want %v", name, check[0], check[1])
		}
	}
	if describe.Reasons["correlation:State/EchoMismatch"] != 1 || describe.Reasons["correlation:Unsupported/MissingReceipt"] != 1 {
		t.Fatalf("describe reasons %v", describe.Reasons)
	}
}

func observed(t *testing.T, method string, request, response proto.Message) *anypb.Any {
	t.Helper()
	requestAny, err := anypb.New(request)
	if err != nil {
		t.Fatal(err)
	}
	responseAny, err := anypb.New(response)
	if err != nil {
		t.Fatal(err)
	}
	packed, err := anypb.New(&realizationv1.Observation{Method: service + method, Request: requestAny,
		Result: &realizationv1.Observation_Response{Response: responseAny}})
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func TestObserveRefusesGuestHostDisagreementAndCountsKeys(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	start := &workflowservice.StartActivityExecutionRequest{Namespace: "ns", ActivityId: "activity"}
	describe := func(run string) *workflowservice.DescribeActivityExecutionRequest {
		return &workflowservice.DescribeActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: run}
	}
	judge := func(seed uint64, method string, request, response proto.Message) error {
		return collector.Observe(activityrelations.Judged{Harness: "explore", Case: "raw/Complete", Seed: seed,
			Observation: observed(t, method, request, response), Explained: true})
	}
	for seed, run := range map[uint64]string{1: runA, 2: runB} {
		if err := judge(seed, "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: run, Started: true}); err != nil {
			t.Fatal(err)
		}
		if err := judge(seed, "DescribeActivityExecution", describe(run), describeResponse(run, 1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := judge(2, "DescribeActivityExecution", describe(runB), describeResponse(runB, 2)); err != nil {
		t.Fatal(err)
	}
	if err := judge(1, "DescribeActivityExecution", describe(runB), describeResponse(runB, 1)); err == nil {
		t.Fatal("a guest-explained invocation the host refuses was accepted")
	}
	report := collector.Count(collector.Rows(), "control")
	counts := report.Methods["temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"]
	if counts.Success != 3 || counts.Refused != 1 || counts.RawDistinct != 3 || counts.ProjectedDistinct != 2 {
		t.Fatalf("describe counts %+v", counts)
	}
}

// The frozen control corpus recomputes to its frozen report with the same functions; UMPIRE_BLESS=1
// rewrites both from the control call stream.
func TestControlCorpusRecomputesItsFrozenReport(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	corpusPath := filepath.Join("testdata", "control-corpus.jsonl")
	reportPath := filepath.Join("testdata", "control-report.json")
	if os.Getenv("UMPIRE_BLESS") == "1" {
		capture(t, collector, controlCalls())
		var corpus bytes.Buffer
		sum, err := activityrelations.WriteCorpus(&corpus, collector.Rows())
		if err != nil {
			t.Fatal(err)
		}
		report, err := json.MarshalIndent(collector.Count(collector.Rows(), sum), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(corpusPath, corpus.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(reportPath, append(report, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	file, err := os.Open(corpusPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	rows, sum, err := activityrelations.ReadCorpus(file)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.MarshalIndent(collector.Count(rows, sum), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(append(actual, '\n')) != string(frozen) {
		t.Fatalf("recomputed report differs from the frozen report:\n%s", actual)
	}
}

func TestObserveKeepsReconciledInvocationsApart(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	start := &workflowservice.StartActivityExecutionRequest{Namespace: "ns", ActivityId: "activity"}
	judge := func(observation *anypb.Any) error {
		return collector.Observe(activityrelations.Judged{Harness: "explore", Case: "raw/CancelBeforeWorkerStart", Seed: 1, Observation: observation, Explained: true})
	}
	if err := judge(observed(t, "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runA, Started: true})); err != nil {
		t.Fatal(err)
	}
	cancel := &workflowservice.RequestCancelActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: runA}
	compound, err := anypb.New(&temporalv1.ReconciledObservation{
		Acknowledged: observed(t, "RequestCancelActivityExecution", cancel, &workflowservice.RequestCancelActivityExecutionResponse{}),
		Current: observed(t, "DescribeActivityExecution", &workflowservice.DescribeActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: runA},
			describeResponse(runA, 1)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := judge(compound); err != nil {
		t.Fatal(err)
	}
	report := collector.Count(collector.Rows(), "control")
	for _, method := range []string{"RequestCancelActivityExecution", "DescribeActivityExecution", "StartActivityExecution"} {
		if counts := report.Methods["temporal.api.workflowservice.v1.WorkflowService/"+method]; counts == nil || counts.Success != 1 {
			t.Fatalf("%s counts %+v", method, counts)
		}
	}
}

// secretDescribe carries synthetic tokens in every Secret-claimed field of a Describe response:
// payload data and metadata values in input and outcome, and a failure's stack trace.
func secretDescribe(run, token string) *workflowservice.DescribeActivityExecutionResponse {
	payload := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte(token + "-metadata")}, Data: []byte(token + "-data")}
	response := describeResponse(run, 1)
	response.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}
	response.Outcome = &activitypb.ActivityExecutionOutcome{Value: &activitypb.ActivityExecutionOutcome_Result{
		Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}}}
	response.Info.LastFailure = &failurepb.Failure{Message: "failed", StackTrace: token + "-stack"}
	return response
}

func TestStoredCorpusAndReportNeverHoldSecretValues(t *testing.T) {
	const alpha, beta = "synthetic-secret-alpha", "synthetic-secret-beta"
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	start := &workflowservice.StartActivityExecutionRequest{Namespace: "ns", ActivityId: "activity",
		Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte(alpha + "-input")}}}}
	describe := &workflowservice.DescribeActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: runA,
		LongPollToken: []byte(alpha + "-long-poll")}
	capture(t, collector, []call{
		{"s", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runA, Started: true}, nil},
		{"s", "DescribeActivityExecution", describe, secretDescribe(runA, alpha), nil},
		{"s", "DescribeActivityExecution", describe, secretDescribe(runA, beta), nil},
		{"s", "DescribeActivityExecution", describe, describeResponse(runA, 1), nil},
	})
	if err := collector.Observe(activityrelations.Judged{Harness: "explore", Case: "secret", Seed: 1, Explained: false,
		Observation: observed(t, "DescribeActivityExecution", describe, secretDescribe(runA, alpha))}); err != nil {
		t.Fatal(err)
	}
	var corpus bytes.Buffer
	sum, err := activityrelations.WriteCorpus(&corpus, collector.Rows())
	if err != nil {
		t.Fatal(err)
	}
	report, err := json.Marshal(collector.Count(collector.Rows(), sum))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{alpha, beta, "synthetic-secret"} {
		for _, encoded := range []string{token, hex.EncodeToString([]byte(token)), base64.StdEncoding.EncodeToString([]byte(token))} {
			if bytes.Contains(corpus.Bytes(), []byte(encoded)) || bytes.Contains(report, []byte(encoded)) {
				t.Fatalf("a Secret value reached the stored corpus or report")
			}
		}
	}
	counts := collector.Count(collector.Rows(), sum).Methods["temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"]
	// Placeholders keep presence, so tokens differing only in value share one key while the
	// response without them stays distinct; raw distinctness survives through the HMAC.
	if counts.Success != 3 || counts.RawDistinct != 3 || counts.ProjectedDistinct != 2 {
		t.Fatalf("describe counts %+v", counts)
	}
}

// The first successful Start of a scope holds the receipt; a later Start of another run in the
// same scope is checked against it, and an empty Cancel acknowledgment still gets the live check.
func TestFirstStartHoldsTheReceiptAndEmptyAcknowledgmentsAreChecked(t *testing.T) {
	collector, err := activityrelations.NewCollector()
	if err != nil {
		t.Fatal(err)
	}
	start := &workflowservice.StartActivityExecutionRequest{Namespace: "ns", ActivityId: "activity"}
	capture(t, collector, []call{
		{"a", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runA, Started: true}, nil},
		{"a", "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runB, Started: true}, nil},
	})
	counts := collector.Count(collector.Rows(), "control").Methods["temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"]
	if counts.Success != 1 || counts.Refused != 1 {
		t.Fatalf("start counts %+v", counts)
	}
	judge := func(observation *anypb.Any) error {
		return collector.Observe(activityrelations.Judged{Harness: "explore", Case: "cancel", Seed: 1, Observation: observation, Explained: true})
	}
	if err := judge(observed(t, "StartActivityExecution", start, &workflowservice.StartActivityExecutionResponse{RunId: runA, Started: true})); err != nil {
		t.Fatal(err)
	}
	crossed := &workflowservice.RequestCancelActivityExecutionRequest{Namespace: "ns", ActivityId: "activity", RunId: runB}
	if err := judge(observed(t, "RequestCancelActivityExecution", crossed, &workflowservice.RequestCancelActivityExecutionResponse{})); err == nil {
		t.Fatal("a guest-explained empty acknowledgment skipped the host check")
	}
}
