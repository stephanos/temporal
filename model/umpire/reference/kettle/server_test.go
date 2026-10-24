package kettle

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	pb "umpire.local/kettle/gen/kettle/v1"
)

func client(t *testing.T) (context.Context, pb.KettleServiceClient) {
	t.Helper()
	return clientWithOptions(t, Options{})
}

func clientWithOptions(t *testing.T, options Options) (context.Context, pb.KettleServiceClient) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := Serve(listener, options)
	t.Cleanup(stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx, pb.NewKettleServiceClient(conn)
}

func TestHeatingReceipts(t *testing.T) {
	ctx, c := client(t)
	filled, err := c.Fill(ctx, &pb.FillRequest{KettleId: "k"})
	if err != nil || filled.GetState().GetTemperature() != 20 {
		t.Fatalf("fill: %v %v", filled, err)
	}
	if _, err := c.Heat(ctx, &pb.HeatRequest{KettleId: "k", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for temperature := int32(20); temperature <= 100; temperature++ {
		poll, err := c.PollHeating(ctx, &pb.PollHeatingRequest{KettleId: "k"})
		if err != nil {
			t.Fatal(err)
		}
		described, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: "k"})
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(poll.GetState(), described.GetState()) || described.GetState().GetTemperature() != temperature || described.GetState().GetBeeping() {
			t.Fatalf("poll changed state: %v %v", poll, described)
		}
		want := pb.HeatingAction_HEATING_ACTION_HEAT
		if temperature == 100 {
			want = pb.HeatingAction_HEATING_ACTION_FAILOVER
		}
		if poll.GetAction() != want || poll.GetKettleId() != "k" {
			t.Fatalf("wrong work: %v", poll)
		}
		receipt, err := c.RespondHeating(ctx, &pb.RespondHeatingRequest{KettleId: "k", Action: poll.GetAction()})
		if err != nil || receipt.GetAction() != want {
			t.Fatalf("receipt: %v %v", receipt, err)
		}
	}
	final, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: "k"})
	if err != nil || !final.GetState().GetBeeping() || final.GetState().GetHeatingOn() || final.GetState().GetTemperature() != 100 {
		t.Fatalf("final: %v %v", final, err)
	}
	repeated, err := c.Fill(ctx, &pb.FillRequest{KettleId: "k"})
	if err != nil || !proto.Equal(repeated.GetState(), final.GetState()) {
		t.Fatalf("repeat fill reset state: %v %v", repeated, err)
	}
	poured, err := c.Pour(ctx, &pb.PourRequest{KettleId: "k"})
	if err != nil || !proto.Equal(poured.GetState(), final.GetState()) {
		t.Fatalf("pour changed state: %v %v", poured, err)
	}
	if _, err := c.Heat(ctx, &pb.HeatRequest{KettleId: "k", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	off, err := c.Heat(ctx, &pb.HeatRequest{KettleId: "k", Enabled: false})
	if err != nil || off.GetState().GetHeatingOn() || off.GetState().GetBeeping() {
		t.Fatalf("disable: %v %v", off, err)
	}
}

func TestRejectionsPreserveState(t *testing.T) {
	ctx, c := client(t)
	if _, err := c.Fill(ctx, &pb.FillRequest{KettleId: "compact"}); err != nil {
		t.Fatal(err)
	}
	valid := func() *pb.FillRequest_FillOptions {
		return &pb.FillRequest_FillOptions{WaterKind: pb.FillRequest_FillOptions_WATER_KIND_TAP, Source: &pb.FillRequest_FillOptions_Tap{Tap: "sink"}}
	}
	cases := []struct {
		name    string
		code    codes.Code
		message string
		send    func() error
	}{
		{"id", codes.InvalidArgument, "kettle_id", func() error { _, e := c.Fill(ctx, &pb.FillRequest{KettleId: strings.Repeat("x", 33)}); return e }},
		{"kind", codes.InvalidArgument, "water_kind", func() error {
			_, e := c.Fill(ctx, &pb.FillRequest{KettleId: "compact", Options: &pb.FillRequest_FillOptions{WaterKind: 99}})
			return e
		}},
		{"source", codes.InvalidArgument, "water source", func() error {
			o := valid()
			o.Source = &pb.FillRequest_FillOptions_Cartridge{Cartridge: []byte{1}}
			_, e := c.Fill(ctx, &pb.FillRequest{KettleId: "compact", Options: o})
			return e
		}},
		{"labels", codes.InvalidArgument, "labels", func() error {
			o := valid()
			o.Labels = map[string]string{"": "v"}
			_, e := c.Fill(ctx, &pb.FillRequest{KettleId: "compact", Options: o})
			return e
		}},
		{"scoped tags", codes.InvalidArgument, "scoped limit", func() error {
			o := valid()
			o.Tags = []string{"a", "b", "c"}
			_, e := c.Fill(ctx, &pb.FillRequest{KettleId: "compact", Options: o})
			return e
		}},
		{"duration", codes.InvalidArgument, "timeout", func() error {
			o := valid()
			o.Timeout = durationpb.New(-time.Second)
			_, e := c.Fill(ctx, &pb.FillRequest{KettleId: "compact", Options: o})
			return e
		}},
		{"missing", codes.NotFound, "not been filled", func() error { _, e := c.Heat(ctx, &pb.HeatRequest{KettleId: "missing", Enabled: true}); return e }},
		{"second kettle", codes.ResourceExhausted, "one kettle", func() error { _, e := c.Fill(ctx, &pb.FillRequest{KettleId: "second"}); return e }},
		{"disabled toggle", codes.FailedPrecondition, "toggle", func() error { _, e := c.Heat(ctx, &pb.HeatRequest{KettleId: "compact"}); return e }},
		{"disabled poll", codes.FailedPrecondition, "requires heating", func() error { _, e := c.PollHeating(ctx, &pb.PollHeatingRequest{KettleId: "compact"}); return e }},
		{"invalid action", codes.InvalidArgument, "action must", func() error {
			_, e := c.RespondHeating(ctx, &pb.RespondHeatingRequest{KettleId: "compact", Action: 99})
			return e
		}},
		{"disabled action", codes.FailedPrecondition, "action is disabled", func() error {
			_, e := c.RespondHeating(ctx, &pb.RespondHeatingRequest{KettleId: "compact", Action: pb.HeatingAction_HEATING_ACTION_HEAT})
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: "compact"})
			if err != nil {
				t.Fatal(err)
			}
			err = tc.send()
			if status.Code(err) != tc.code || !strings.Contains(status.Convert(err).Message(), tc.message) {
				t.Fatalf("got %v; want %v containing %q", err, tc.code, tc.message)
			}
			after, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: "compact"})
			if err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(before, after) {
				t.Fatalf("rejection changed state: %v -> %v", before, after)
			}
		})
	}
}

func TestMutantOptions(t *testing.T) {
	for _, name := range append(Mutants(), "") {
		if err := (Options{Mutant: name}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if err := (Options{Mutant: "unknown"}).Validate(); err == nil {
		t.Fatal("unknown mutant accepted")
	}
}

func TestMutantsAreServerLocal(t *testing.T) {
	for _, name := range Mutants() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, mutant := range []string{name, ""} {
				t.Run("server="+mutant, func(t *testing.T) {
					t.Parallel()
					ctx, c := clientWithOptions(t, Options{Mutant: mutant})
					switch name {
					case "overlong-id":
						id := strings.Repeat("x", MaxKettleIDBytes+1)
						want := codes.InvalidArgument
						if mutant != "" {
							want = codes.OK
						}
						_, err := c.Fill(ctx, &pb.FillRequest{KettleId: id})
						if status.Code(err) != want {
							t.Fatalf("Fill: got %v, want %v", err, want)
						}
						described, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: id})
						if status.Code(err) != want || want == codes.OK && described.GetState().GetTemperature() != 20 {
							t.Fatalf("Describe: got %v %v, want %v", described, err, want)
						}
						if _, err := c.Fill(ctx, &pb.FillRequest{}); status.Code(err) != codes.InvalidArgument {
							t.Fatalf("empty id accepted: %v", err)
						}
					case "removed-validator":
						o := &pb.FillRequest_FillOptions{WaterKind: pb.FillRequest_FillOptions_WATER_KIND_TAP, Source: &pb.FillRequest_FillOptions_Tap{Tap: "sink"}, Labels: map[string]string{"": "value"}}
						_, err := c.Fill(ctx, &pb.FillRequest{KettleId: "k", Options: o})
						want := codes.InvalidArgument
						if mutant != "" {
							want = codes.OK
						}
						if status.Code(err) != want {
							t.Fatalf("labels: got %v, want %v", err, want)
						}
						o.WaterKind = 99
						if _, err := c.Fill(ctx, &pb.FillRequest{KettleId: "k", Options: o}); status.Code(err) != codes.InvalidArgument || !strings.Contains(status.Convert(err).Message(), "water_kind") {
							t.Fatalf("independent validator removed: %v", err)
						}
					case "wrong-status":
						want := codes.InvalidArgument
						if mutant != "" {
							want = codes.FailedPrecondition
						}
						for _, id := range []string{"", strings.Repeat("x", MaxKettleIDBytes+1)} {
							if _, err := c.Fill(ctx, &pb.FillRequest{KettleId: id}); status.Code(err) != want {
								t.Fatalf("Fill: got %v, want %v", err, want)
							}
							if _, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: id}); status.Code(err) != want {
								t.Fatalf("Describe: got %v, want %v", err, want)
							}
						}
					case "premature-boil":
						if _, err := c.Fill(ctx, &pb.FillRequest{KettleId: "k"}); err != nil {
							t.Fatal(err)
						}
						if _, err := c.Heat(ctx, &pb.HeatRequest{KettleId: "k", Enabled: true}); err != nil {
							t.Fatal(err)
						}
						poll, err := c.PollHeating(ctx, &pb.PollHeatingRequest{KettleId: "k"})
						if err != nil || poll.GetAction() != pb.HeatingAction_HEATING_ACTION_HEAT || poll.GetState().GetTemperature() != 20 {
							t.Fatalf("PollHeating: %v %v", poll, err)
						}
						described, err := c.Describe(ctx, &pb.DescribeRequest{KettleId: "k"})
						want := &pb.State{Temperature: 20, HeatingOn: true}
						if mutant != "" {
							want = &pb.State{Temperature: 100, Beeping: true}
						}
						if err != nil || !proto.Equal(described.GetState(), want) {
							t.Fatalf("Describe: %v %v, want %v", described, err, want)
						}
						receipt, err := c.RespondHeating(ctx, &pb.RespondHeatingRequest{KettleId: "k", Action: poll.GetAction()})
						if err != nil || receipt.GetState().GetTemperature() != 21 || !receipt.GetState().GetHeatingOn() || receipt.GetState().GetBeeping() {
							t.Fatalf("Describe changed stored state: %v %v", receipt, err)
						}
					}
				})
			}
		})
	}
}

func TestServeRejectsUnknownMutant(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Error(err)
		}
	})
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("Serve accepted unknown mutant")
		} else if err, ok := recovered.(error); !ok || !strings.Contains(err.Error(), "unknown kettle mutant") {
			t.Errorf("unexpected panic: %v", recovered)
		}
	}()
	stop := Serve(listener, Options{Mutant: "unknown"})
	stop()
}
