package wasi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
)

const networkTemporalObservationPrefix = "GOMAD_NETWORK_TEMPORAL_OBSERVATION "

type networkTemporalValue struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	Result     string   `json:"result"`
	History    []byte   `json:"history"`
	EventIDs   []int64  `json:"event_ids"`
	Types      []string `json:"types"`
	Attempts   []struct {
		StartedNanos uint64 `json:"started_nanos"`
		EndedNanos   uint64 `json:"ended_nanos"`
		Code         string `json:"code"`
	} `json:"attempts"`
	Connections []struct {
		OpenedNanos uint64 `json:"opened_nanos"`
		LocalPort   int    `json:"local_port"`
		RemotePort  int    `json:"remote_port"`
	} `json:"connections"`
}

func networkTemporalValueFrom(t *testing.T, stdout []byte) networkTemporalValue {
	t.Helper()
	var value networkTemporalValue
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte(networkTemporalObservationPrefix)) {
			continue
		}
		if err := json.Unmarshal(line[len(networkTemporalObservationPrefix):], &value); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 || value.WorkflowID == "" || value.RunID == "" || len(value.History) == 0 || len(value.EventIDs) == 0 || len(value.EventIDs) != len(value.Types) || value.Result != "durable timer survived rpc-reset" {
		t.Fatalf("invalid independent Temporal history observation: count=%d value=%+v", count, value)
	}
	for index, id := range value.EventIDs {
		if id != int64(index+1) {
			t.Fatalf("duplicate or lost Temporal history: %v", value.EventIDs)
		}
	}
	return value
}

func networkTemporalResetWitness(t *testing.T, result Result, observation networkTemporalValue) {
	t.Helper()
	var transcript struct {
		Operations []struct {
			At      uint64           `json:"at"`
			Fault   *NetworkFaultV2  `json:"fault"`
			Request netmodel.Request `json:"request"`
			Result  netmodel.Result  `json:"result"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(result.NetworkTranscript, &transcript); err != nil {
		t.Fatal(err)
	}
	const resetAt = uint64(40 * time.Second)
	// Guest elapsed durations exclude the cooperative runtime's 1 ns clock origin.
	const clockOrigin = uint64(1)
	fault := false
	ports := map[int]bool{}
	for _, connection := range observation.Connections {
		if connection.OpenedNanos+clockOrigin < resetAt {
			ports[connection.LocalPort] = true
		}
	}
	if len(ports) == 0 {
		t.Fatal("no SDK connection existed before the named reset")
	}
	handles := map[uint64]bool{}
	pending := map[uint64]uint64{}
	resetError, reconnected := false, false
	for _, operation := range transcript.Operations {
		if operation.Fault != nil && operation.Fault.Name == "rpc-reset" && operation.Fault.Action == "reset" && operation.Fault.AtNanos == resetAt {
			fault = true
		}
		if operation.Request.Op == "dial" && operation.Result.State == "completed" && operation.Result.Error == "" && operation.Result.Handle != 0 {
			if operation.At < resetAt && ports[operation.Result.Local.Port] {
				handles[operation.Result.Handle] = true
			}
			for _, connection := range observation.Connections {
				if operation.At >= resetAt && connection.OpenedNanos+clockOrigin >= resetAt && operation.Result.Local.Port == connection.LocalPort {
					reconnected = true
				}
			}
		}
		if operation.Request.Op == "read" || operation.Request.Op == "write" {
			pending[operation.Request.RequestID] = operation.Request.Handle
		}
		handle := operation.Request.Handle
		if operation.Request.Op == "poll" {
			handle = pending[operation.Request.RequestID]
		}
		if operation.At >= resetAt && handles[handle] && (operation.Request.Op == "read" || operation.Request.Op == "write" || operation.Request.Op == "poll") && (operation.Result.Error == "use of closed network connection" || operation.Result.Error == "connection reset by peer") {
			resetError = true
		}
	}
	if !fault || !resetError || !reconnected {
		t.Fatalf("named reset lacks an established SDK stream error and fresh SDK connection: fault=%v stream_error=%v reconnected=%v", fault, resetError, reconnected)
	}
	failed := false
	retried := false
	for _, attempt := range observation.Attempts {
		if attempt.StartedNanos+clockOrigin < resetAt && attempt.EndedNanos+clockOrigin >= resetAt && attempt.Code == "Unavailable" {
			failed = true
		}
		if failed && attempt.StartedNanos+clockOrigin >= resetAt && attempt.Code == "OK" {
			retried = true
		}
	}
	if !retried {
		t.Fatalf("no in-flight history RPC failed across rpc-reset and then recovered: %+v", observation.Attempts)
	}
}

func TestTemporalNetworkNamedResetRetriesAndExactReplay(t *testing.T) {
	evidence := os.Getenv("GOMAD_WASM_NETWORK_TEMPORAL_ROOT")
	if evidence == "" {
		t.Skip("set GOMAD_WASM_NETWORK_TEMPORAL_ROOT to run the actual Temporal network retry gate")
	}
	if !filepath.IsAbs(evidence) {
		t.Fatal("GOMAD_WASM_NETWORK_TEMPORAL_ROOT must be absolute")
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixtureSource, err := os.ReadFile(filepath.Join(root, "tools", "gomad_wasm", "testdata", "network_streams", "temporal_retry_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "network-streams-retry-"+digest(fixtureSource)[:12])
	if err := temporalStageFixture(root, filepath.Join("tools", "gomad_wasm", "testdata", "network_streams"), root, staged); err != nil {
		t.Fatal(err)
	}
	module, _ := prototypeBuild(t, root, "test", "./"+filepath.ToSlash(staged), filepath.Join(prototypeRoot(t), "network-temporal-retry.wasm"), "test_dep", "disable_grpc_modules", "sqlite3_dotlk")
	config := prototypeTemporalConfig(t, root, "/workspace/network-temporal", "-test.run=^TestTemporalWorkflowRecoversNamedNetworkReset$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Deliveries: 65536, Bytes: 128 << 20, Transitions: 1 << 20},
		Faults: []NetworkFaultV2{{Name: "rpc-reset", Action: "reset", AtNanos: uint64(40 * time.Second)}},
	}
	request := prototypeRequest(t, module, config, 7, 15*time.Minute)
	request.DigestTranscript = false
	observed, err := Run(t.Context(), request)
	retain := func(label string, result Result) {
		for suffix, data := range map[string][]byte{"stdout": result.Stdout, "stderr": result.Stderr, "wasi.jsonl": result.Transcript, "network.json": result.NetworkTranscript, "choices": result.ChoiceTrace.Bytes} {
			temporalWrite(t, filepath.Join(evidence, "network-temporal-"+label+"."+suffix), data)
		}
		temporalJSON(t, filepath.Join(evidence, "network-temporal-"+label+".terminal.json"), map[string]any{
			"termination": result.Termination, "message": result.Message, "exit_code": result.ExitCode, "reaped": result.Reaped,
			"environment_sha256": result.EnvironmentIdentity, "model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
			"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
			"clock": request.Config.Clock, "network": request.Config.NetworkV2,
		})
	}
	retain("record", observed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset recording", observed, 0)
	value := networkTemporalValueFrom(t, observed.Stdout)
	networkTemporalResetWitness(t, observed, value)
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	replayed, err := Run(t.Context(), request)
	retain("replay", replayed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("Temporal named-reset replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || !reflect.DeepEqual(value, networkTemporalValueFrom(t, replayed.Stdout)) {
		t.Fatal("exact replay changed I/O or independent workflow history")
	}
	temporalJSON(t, filepath.Join(evidence, "network-temporal-history-oracle.json"), value)
	temporalJSON(t, filepath.Join(evidence, "network-temporal-retry-report.json"), map[string]any{
		"schema": "gomad3.wasm-temporal-network-retry/v1", "exact_replay": true,
		"model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
		"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
		"environment_sha256": observed.EnvironmentIdentity, "seed": request.Runtime.Seed, "clock": request.Config.Clock, "network": request.Config.NetworkV2,
		"wasi_sha256": digest(observed.Transcript), "choices_sha256": digest(observed.ChoiceTrace.Bytes), "network_sha256": digest(observed.NetworkTranscript),
		"history_sha256": digest(value.History), "history": value, "termination": observed.Termination, "exit_code": observed.ExitCode,
	})
	t.Logf("Temporal history RPC retried after named reset; events=%d attempts=%d connections=%d choices=%d", len(value.EventIDs), len(value.Attempts), len(value.Connections), len(observed.ChoiceTrace.Records))
}
