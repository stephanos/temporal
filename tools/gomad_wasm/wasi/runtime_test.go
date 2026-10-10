package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

func TestCooperativeFrozenClockPreservesStartupAndStockPolicy(t *testing.T) {
	config := testConfig()
	config.Clock.ReadStepNanos = 0
	if _, err := NewEnvironment(config); err == nil {
		t.Fatal("stock profile admitted frozen clock")
	}
	config.Profile = CooperativeProfile
	e := newTestEnvironment(t, config)
	for range 2 {
		var timestamp timestampOutput
		output(t, call(t, e, "clock_time_get", `{"clock_id":1,"precision":0}`), &timestamp)
		if timestamp.Timestamp != 1 || e.now != 1 || e.config.Clock.ReadStepNanos != 0 {
			t.Fatalf("frozen monotonic clock advanced: %+v now=%d policy=%+v", timestamp, e.now, e.config.Clock)
		}
	}
}

func TestCooperativeEnvironmentSeparatesRuntimeControlFromWASIObservations(t *testing.T) {
	config := testConfig()
	config.Profile = CooperativeProfile
	environment, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	environment.runtime, err = newRuntimeSession(&RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}})
	if err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(bytesOutput{Data: make([]byte, 16)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := environment.Handle(Call{Type: "call", ID: 1, Op: "runtime_config", Input: input}); err != nil {
		t.Fatal(err)
	}
	if len(environment.Transcript()) != 0 {
		t.Fatal("control payload leaked into replay observation")
	}
	deadline := make([]byte, 16)
	binary.BigEndian.PutUint32(deadline, 1)
	binary.BigEndian.PutUint64(deadline[8:], 1000000)
	input, err = json.Marshal(bytesOutput{Data: deadline})
	if err != nil {
		t.Fatal(err)
	}
	reply, err := environment.Handle(Call{Type: "call", ID: 2, Op: "runtime_idle", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	var data bytesOutput
	if err := json.Unmarshal(reply.Output, &data); err != nil || binary.BigEndian.Uint64(data.Data[8:]) != 1000000 || environment.now != 1000000 {
		t.Fatalf("idle deadline handshake = %#v, %v", data, err)
	}
	if _, err := environment.Handle(Call{Type: "call", ID: 3, Op: "random_get", Input: json.RawMessage(`{"length":8}`)}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(environment.Transcript(), []byte("runtime_")) || !bytes.Contains(environment.Transcript(), []byte("random_get")) {
		t.Fatal("runtime and application observation domains mixed")
	}
}

func runtimeChoiceInput(kind uint32, site uint64, identities [][32]byte, selected uint32) []byte {
	data := make([]byte, 32+32*len(identities))
	binary.BigEndian.PutUint32(data, 1)
	binary.BigEndian.PutUint32(data[4:], kind)
	binary.BigEndian.PutUint64(data[8:], site)
	binary.BigEndian.PutUint32(data[20:], uint32(len(identities)))
	binary.BigEndian.PutUint32(data[24:], selected)
	for i, id := range identities {
		copy(data[32+32*i:], id[:])
	}
	return data
}

func TestRuntimeTransportForcesLogicalAlternativesAndConsumesWholeTape(t *testing.T) {
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	control := &RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	observed, err := newRuntimeSession(control)
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(session *runtimeSession, name string, bytes []byte) ([]byte, error) {
		input, err := json.Marshal(bytesOutput{Data: bytes})
		if err != nil {
			t.Fatal(err)
		}
		value, err := session.handle(name, input)
		return value.Data, err
	}
	if _, err := invoke(observed, "runtime_config", make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(observed, "runtime_decision", runtimeChoiceInput(1, 10, [][32]byte{a, b}, 0)); err != nil {
		t.Fatal(err)
	}
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	if _, err := invoke(observed, "runtime_finish", finish); err != nil {
		t.Fatal(err)
	}
	trace, _, err := observed.collect()
	if err != nil || len(trace.Records) != 1 || trace.Records[0].SelectedIdentity != a {
		t.Fatalf("recorded logical choice: %#v %v", trace, err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"exact", "missing", "extra", "kind", "high-kind", "site", "enabled", "rank", "readiness", "duplicate", "unconfigured", "unfinished"} {
		t.Run(name, func(t *testing.T) {
			plan := tape
			mode := choice.ModeReplay
			if name == "missing" {
				plan, err = tape.Prefix(0)
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "rank" {
				plan.Decisions = append([]choice.Decision(nil), tape.Decisions...)
				plan.Decisions[0].Selected = 99
			}
			if name == "readiness" {
				plan.Readiness = []choice.SelectReadiness{{Known: true, Ready: 1}}
			}
			replay, err := newRuntimeSession(&RuntimeControl{Seed: 100, Choice: &runnerbackend.ChoiceRequest{Mode: mode, ExecutionIdentity: identity, Limit: 8192, Tape: &plan}})
			if name == "rank" || name == "readiness" {
				if err == nil {
					t.Fatal("altered decoded tape projection accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if name != "unconfigured" {
				if _, err := invoke(replay, "runtime_config", make([]byte, 16)); err != nil {
					t.Fatal(err)
				}
			}
			input := runtimeChoiceInput(1, 10, [][32]byte{b, a}, 0)
			switch name {
			case "kind":
				binary.BigEndian.PutUint32(input[4:], 2)
			case "high-kind":
				binary.BigEndian.PutUint32(input[4:], 257)
			case "site":
				binary.BigEndian.PutUint64(input[8:], 11)
			case "enabled":
				input = runtimeChoiceInput(1, 10, [][32]byte{a, sha256.Sum256([]byte("other"))}, 0)
			case "duplicate":
				input = runtimeChoiceInput(1, 10, [][32]byte{a, a}, 0)
			}
			if name != "extra" {
				output, callErr := invoke(replay, "runtime_decision", input)
				err = callErr
				if name == "exact" || name == "unfinished" {
					if err != nil || binary.BigEndian.Uint32(output[24:]) != 1 {
						t.Fatalf("logical rank failed across physical order: %x %v", output, err)
					}
				} else if err == nil {
					t.Fatal("inapplicable decision accepted")
				}
				if name == "duplicate" || name == "high-kind" {
					var boundary *BoundaryError
					if !errors.As(err, &boundary) || boundary.Kind != "invalid" {
						t.Fatalf("malformed guest decision was not invalid: %v", err)
					}
				}
			}
			if name == "extra" {
				if _, err := invoke(replay, "runtime_finish", finish); err == nil {
					t.Fatal("unconsumed tape accepted")
				}
			}
			if name == "exact" {
				if _, err := invoke(replay, "runtime_finish", finish); err != nil {
					t.Fatal(err)
				}
				forced, _, err := replay.collect()
				if err != nil || forced.SHA256 != trace.SHA256 {
					t.Fatalf("forced complete trace differs: %x %v", forced.SHA256, err)
				}
			}
			if name == "unfinished" {
				if _, _, err := replay.collect(); err == nil {
					t.Fatal("missing runtime finish accepted")
				}
			}
		})
	}
}
