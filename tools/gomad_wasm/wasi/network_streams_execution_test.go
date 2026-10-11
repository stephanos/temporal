package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
)

func TestNetworkStreamsGuestRecordsAndReplays(t *testing.T) {
	root := prototypeRoot(t)
	module := prototypeFixture(t, "network_streams")
	config := prototypeFixtureConfig()
	config.Args = []string{"network_streams"}
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 16, Connections: 128, Deliveries: 4096, Bytes: 8 << 20, Transitions: 16384},
		Faults: []NetworkFaultV2{
			{Name: "stream-delay", Action: "delay", DelayNanos: uint64(5 * time.Millisecond)},
			{Name: "dial-partition", Action: "partition", AtNanos: uint64(50 * time.Millisecond)},
			{Name: "dial-heal", Action: "heal", AtNanos: uint64(90 * time.Millisecond)},
			{Name: "connection-reset", Action: "reset", AtNanos: uint64(120 * time.Millisecond), Port: 9000},
		}}
	request := prototypeRequest(t, module, config, 7, 2*time.Minute)
	request.DigestTranscript = false
	request.MemoryBytes = 256 << 20
	request.Fuel = 1 << 36
	observed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream recording", observed, 0)
	for _, want := range []string{"chunks 196745 sha256", "half-close drained", "read and accept deadlines", "partition dial deadline and cancellation", "named reset; new connection survived"} {
		if !bytes.Contains(observed.Stdout, []byte(want)) {
			t.Fatalf("missing %q in guest stdout: %s", want, observed.Stdout)
		}
	}
	for _, want := range []string{"stream-delay", "dial-partition", "dial-heal", "connection-reset", `"kind":"deliver"`, `"kind":"network"`, `"op":"cancel"`, `"op":"poll"`, `"op":"close_read"`, `"op":"close_write"`} {
		if !bytes.Contains(observed.NetworkTranscript, []byte(want)) {
			t.Fatalf("missing %q in network evidence", want)
		}
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("network stream replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || len(observed.NetworkTranscript) == 0 {
		t.Fatal("network stream replay transcript differs or network evidence is empty")
	}
	report, err := json.MarshalIndent(struct {
		Schema, ModuleSHA256, HelperSHA256, RuntimeSHA256 string
		NetworkSHA256, WASISHA256, ChoiceSHA256           string
		ModelImplementationSHA256, ProjectionSHA256       string
		Seed                                              uint64
		ChoiceRecords                                     int
		ExactReplay                                       bool
		Model                                             *NetworkModelV2
	}{"gomad.wasm-network-streams-execution/v1", request.ModuleSHA256, request.HelperSHA256,
		request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, digest(observed.NetworkTranscript),
		digest(observed.Transcript), digest(observed.ChoiceTrace.Bytes), ImplementationSHA256(), netmodel.ImplementationSHA256(), request.Runtime.Seed,
		len(observed.ChoiceTrace.Records), true, config.NetworkV2}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"network-streams.report.json":  report,
		"network-streams.stdout":       observed.Stdout,
		"network-streams.wasi.jsonl":   observed.Transcript,
		"network-streams.network.json": observed.NetworkTranscript,
		"network-streams.choices":      observed.ChoiceTrace.Bytes,
	} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("actual stream bytes, parked readiness and exact replay: network=%s wasi=%s choices=%d", digest(observed.NetworkTranscript), digest(observed.Transcript), len(observed.ChoiceTrace.Records))
}
