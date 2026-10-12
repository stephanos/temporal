package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3sim"
)

func TestHistoryObservationRejectsAmbiguousInputs(t *testing.T) {
	plan := &OraclePlan{Schema: HistoryObservationSchema, Prefix: "OBS ", MaximumBytes: 4096}
	good := `{"cluster_id":"cluster","history":"YQ==","event_ids":[1],"types":["started"]}`
	for _, stdout := range []string{"", "OBS " + good + "\nOBS " + good, "OBS " + good[:len(good)-1] + `,"history":"Yg=="}`, "OBS " + good[:len(good)-1] + `,"unknown":true}`, "OBS {}", "OBS " + good + " {}"} {
		if _, err := decodeHistoryObservation([]byte(stdout), plan); err == nil {
			t.Fatal("ambiguous or missing observation admitted", stdout)
		}
	}
	if _, err := decodeHistoryObservation([]byte("OBS "+good), plan); err != nil {
		t.Fatal(err)
	}
}

func TestProviderDetachedOracleCanarySourceLessReplay(t *testing.T) {
	base := guestRequest(t)
	base.Config.Args = []string{"service", "observation"}
	var payloads []struct {
		data     []byte
		observed gomad3sim.ClusterRecord
		canary   bool
	}
	for _, canary := range []bool{false, true} {
		peer := base
		if canary {
			peer.Config.Args = []string{"service", "observation", "canary"}
		}
		a, err := BootConfiguration(base)
		if err != nil {
			t.Fatal(err)
		}
		b, err := BootConfiguration(peer)
		if err != nil {
			t.Fatal(err)
		}
		spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "a", Address: "10.0.0.1", Config: a}, {ID: "b", Boot: "b", Address: "10.0.0.2", Config: b}}}
		handles := []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "b", Incarnation: 1}}
		p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": base, "b": peer}, Oracles: &OraclePlan{Schema: HistoryObservationSchema, Prefix: "GOMAD_HISTORY_OBSERVATION ", MaximumBytes: 4096, Histories: []HistoryComparison{{Name: "fixture.recovered", Expected: handles[0], Actual: handles[1], ExpectedEvents: []string{"1", "2"}}}, Convergence: []HistoryConvergence{{Name: "fixture.converged", Handles: handles}}}}
		result, err := gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomad3sim.Cluster) error {
			for _, h := range handles {
				if _, err := c.Start(ctx, h.Node); err != nil {
					return err
				}
			}
			for _, h := range handles {
				if _, err := c.Wait(ctx, h); err != nil {
					return err
				}
			}
			return p.RecordOracles(ctx, c)
		}, p)
		if err != nil {
			t.Fatal(err)
		}
		expected := gomad3sim.OutcomeCompleted
		if canary {
			expected = gomad3sim.OutcomeOracleFailed
		}
		if result.Outcome != expected || len(result.Oracles) != 4 {
			t.Fatalf("fresh detached oracle result: %s reason=%s oracles=%+v", result.Outcome, result.Reason, result.Oracles)
		}
		payload, err := p.RetentionPayload(spec, result.Record)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		payloads = append(payloads, struct {
			data     []byte
			observed gomad3sim.ClusterRecord
			canary   bool
		}{encoded, result.Record, canary})
	}
	module := retainedTargetModule(t, base)
	if err := os.Rename(base.ModulePath, base.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD3_STOCK_GO", "/original/compiler/unavailable")
	for _, retained := range payloads {
		// Decode an existing Artifact payload envelope, not original scenario code.
		var backendPayload target.BackendPayload
		if err := json.Unmarshal(retained.data, &backendPayload); err != nil {
			t.Fatal(err)
		}
		result, err := ReplayRetained(t.Context(), backendPayload, module, t.TempDir(), installedHelper(base))
		if err != nil {
			t.Fatal(err)
		}
		left, err := gomad3sim.EncodeClusterRecord(retained.observed)
		if err != nil {
			t.Fatal(err)
		}
		right, err := gomad3sim.EncodeClusterRecord(result.Record)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatal("fresh typed oracle replay changed record/canary")
		}
	}
}
