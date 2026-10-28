package kettle_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/kettle"
	kettlev1 "umpire.local/kettle/gen/kettle/v1"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	model "umpire.local/model/host"
)

type kettleRun struct {
	t          *testing.T
	ctx        context.Context
	host       *model.Host
	client     kettlev1.KettleServiceClient
	connection *grpc.ClientConn
	annotated  map[string][2]int
	transcript strings.Builder
	witness    []string
}

func newKettleRun(t *testing.T) *kettleRun {
	t.Helper()
	return newKettleRunWithOptions(t, kettle.Options{})
}

func newKettleRunWithOptions(t *testing.T, options kettle.Options) *kettleRun {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := kettle.Serve(listener, options)
	t.Cleanup(stop)
	r := &kettleRun{t: t, ctx: ctx, annotated: make(map[string][2]int)}
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(r.annotateRPC))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	path := os.Getenv("UMPIRE_KETTLE_WASM")
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1", "guest", "umpire_kettle_guest.wasm")
	}
	wasm, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Kettle guest required; build from the framework root with mise exec -- cargo xtask guest umpire-kettle-guest <path>: %v", err)
	}
	host, err := model.New(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := host.Close(closeCtx); err != nil {
			t.Error(err)
		}
	})
	reset, err := host.Reset(ctx, &modelv1.ResetRequest{})
	if err != nil || reset.GetProofContract() != modelv1.ProofContract_PROOF_CONTRACT_WHOLE_PREFIX_V1 {
		t.Fatalf("Reset proof contract: %v %v", reset, err)
	}
	r.host, r.connection, r.client = host, conn, kettlev1.NewKettleServiceClient(conn)
	return r
}

type kettleProbeContextKey struct{}

func (r *kettleRun) annotateRPC(ctx context.Context, method string, request, response any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
	if probe, _ := ctx.Value(kettleProbeContextKey{}).(bool); probe {
		return invoke(ctx, method, request, response, connection, options...)
	}
	if err := r.annotateMessage(ctx, method, request.(proto.Message), false); err != nil {
		return err
	}
	if err := invoke(ctx, method, request, response, connection, options...); err != nil {
		return err
	}
	return r.annotateMessage(ctx, method, response.(proto.Message), true)
}

func (r *kettleRun) annotateMessage(ctx context.Context, method string, message proto.Message, response bool) error {
	wire, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	annotated, err := model.Annotate(ctx, r.host, &annotationsv1.AnnotationRequest{Method: method, Response: response, Message: wire, Seed: 233})
	if err != nil {
		return err
	}
	if annotated.GetOutOfScope() != "" || len(annotated.GetViolations()) != 0 {
		return fmt.Errorf("%s response=%t annotation: %v", method, response, annotated)
	}
	counts := r.annotated[method]
	if response {
		if !bytes.Equal(annotated.Message, wire) {
			return fmt.Errorf("%s annotation changed actual response bytes", method)
		}
		counts[1]++
	} else {
		if err := proto.Unmarshal(annotated.Message, message); err != nil {
			return err
		}
		counts[0]++
	}
	r.annotated[method] = counts
	return nil
}

func kettlePack(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	packed, err := anypb.New(message)
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func (r *kettleRun) judge(method string, request, response proto.Message, state *kettlev1.State, steps ...string) {
	r.t.Helper()
	observation := &realizationv1.Observation{
		Method:  "/umpire.reference.kettle.v1.KettleService/" + method,
		Request: kettlePack(r.t, request),
		Result:  &realizationv1.Observation_Response{Response: kettlePack(r.t, response)},
	}
	judgment, err := r.host.Judge(r.ctx, &modelv1.JudgeRequest{Observation: kettlePack(r.t, observation)})
	steps = append(slices.Clone(r.witness), steps...)
	if err != nil || judgment.GetExplained() == nil || !slices.Equal(judgment.GetExplained().Steps, steps) {
		r.t.Fatalf("%s Judge: %v %v, want steps %v", method, judgment, err, steps)
	}
	outcomes := make([]string, len(steps))
	for i := range outcomes {
		outcomes[i] = "()"
	}
	if !slices.Equal(judgment.GetExplained().Outcomes, outcomes) {
		r.t.Fatalf("%s outcomes: %v", method, judgment.GetExplained().Outcomes)
	}
	r.witness = steps
	fmt.Fprintf(&r.transcript, "%s: temperature=%d heating_on=%t beeping=%t Explained steps=%v outcomes=%v\n", method, state.GetTemperature(), state.GetHeatingOn(), state.GetBeeping(), steps, outcomes)
}

func (r *kettleRun) fill() {
	r.t.Helper()
	request := &kettlev1.FillRequest{KettleId: "room"}
	response, err := r.client.Fill(r.ctx, request)
	if err != nil {
		r.t.Fatal(err)
	}
	if request.GetIdempotencyKey() == "" {
		r.t.Fatal("Annotate did not fill the omitted idempotency key")
	}
	if !proto.Equal(response.GetState(), &kettlev1.State{Temperature: 20}) {
		r.t.Fatalf("Fill: %v", response)
	}
	r.judge("Fill", request, response, response.GetState())
}

func (r *kettleRun) heat(enabled bool) {
	r.t.Helper()
	request := &kettlev1.HeatRequest{KettleId: "room", Enabled: enabled}
	response, err := r.client.Heat(r.ctx, request)
	if err != nil {
		r.t.Fatal(err)
	}
	step := "User::PressButton"
	if !enabled {
		step = "User::DepressButton"
	}
	r.judge("Heat", request, response, response.GetState(), step)
}

func (r *kettleRun) describe(want *kettlev1.State) {
	r.t.Helper()
	request := &kettlev1.DescribeRequest{KettleId: "room"}
	response, err := r.client.Describe(r.ctx, request)
	if err != nil {
		r.t.Fatal(err)
	}
	if !proto.Equal(response.GetState(), want) {
		r.t.Fatalf("Describe=%v, want %v", response.GetState(), want)
	}
	r.judge("Describe", request, response, response.GetState())
}

func (r *kettleRun) poll(temperature int32, action kettlev1.HeatingAction) *kettlev1.RespondHeatingRequest {
	r.t.Helper()
	task, err := r.client.PollHeating(r.ctx, &kettlev1.PollHeatingRequest{KettleId: "room"})
	if err != nil {
		r.t.Fatal(err)
	}
	if !proto.Equal(task.GetState(), &kettlev1.State{Temperature: temperature, HeatingOn: true}) || task.GetAction() != action {
		r.t.Fatalf("PollHeating=%v", task)
	}
	fmt.Fprintf(&r.transcript, "PollHeating: temperature=%d heating_on=true beeping=false action=%s\n", temperature, action)
	answer, err := r.host.Answer(r.ctx, &modelv1.AnswerRequest{Callback: kettlePack(r.t, &realizationv1.Callback{Task: kettlePack(r.t, task)})})
	if err != nil {
		r.t.Fatal(err)
	}
	reply := new(realizationv1.Reply)
	if err := answer.GetReply().UnmarshalTo(reply); err != nil {
		r.t.Fatal(err)
	}
	request := new(kettlev1.RespondHeatingRequest)
	if err := reply.GetRequest().UnmarshalTo(request); err != nil {
		r.t.Fatal(err)
	}
	if request.GetKettleId() != "room" || request.GetAction() != action {
		r.t.Fatalf("Answer=%v", request)
	}
	fmt.Fprintf(&r.transcript, "Answer: kettle_id=%s action=%s (not committed)\n", request.GetKettleId(), request.GetAction())
	return request
}

func TestKettleJudgeAnswerTranscript(t *testing.T) {
	r := newKettleRun(t)
	r.fill()
	r.heat(true)
	r.heat(false)
	r.describe(&kettlev1.State{Temperature: 20})
	r.heat(true)
	pourRequest := &kettlev1.PourRequest{KettleId: "room"}
	pour, err := r.client.Pour(r.ctx, pourRequest)
	if err != nil {
		t.Fatal(err)
	}
	r.judge("Pour", pourRequest, pour, pour.GetState())
	for temperature := int32(20); temperature <= 100; temperature++ {
		action, step := kettlev1.HeatingAction_HEATING_ACTION_HEAT, "Heater::Heat"
		if temperature == 100 {
			action, step = kettlev1.HeatingAction_HEATING_ACTION_FAILOVER, "Heater::Failover"
		}
		request := r.poll(temperature, action)
		if temperature == 20 || temperature == 100 {
			r.describe(&kettlev1.State{Temperature: temperature, HeatingOn: true})
		}
		response, err := r.client.RespondHeating(r.ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if response.GetAction() != action {
			t.Fatalf("wrong receipt: %v", response)
		}
		want := &kettlev1.State{Temperature: temperature + 1, HeatingOn: true}
		if temperature == 100 {
			want = &kettlev1.State{Temperature: 100, Beeping: true}
		}
		if !proto.Equal(response.GetState(), want) {
			t.Fatalf("RespondHeating=%v want %v", response, want)
		}
		r.judge("RespondHeating", request, response, response.GetState(), step)
	}
	r.describe(&kettlev1.State{Temperature: 100, Beeping: true})
	for method, want := range map[string]int{"Fill": 1, "Heat": 3, "Pour": 1, "Describe": 4, "PollHeating": 81, "RespondHeating": 81} {
		if got := r.annotated["/umpire.reference.kettle.v1.KettleService/"+method]; got != [2]int{want, want} {
			t.Fatalf("%s request/response annotation counts=%v, want %d each", method, got, want)
		}
	}
	exploration, err := r.host.Explore(r.ctx, &modelv1.ExploreRequest{Budget: 1024})
	if err != nil {
		t.Fatal(err)
	}
	expectedReport := readSnapshot(t, "expected/report.txt.snap")
	if exploration.GetStarts() != 81 || exploration.GetReport() != expectedReport {
		t.Fatalf("canonical exploration changed: %v", exploration)
	}
	fmt.Fprintf(&r.transcript, "Explore: starts=%d reached=%d outcome=Verified\n", exploration.GetStarts(), exploration.GetReached())
	compareSnapshot(t, "transcript.txt", r.transcript.String())
}
