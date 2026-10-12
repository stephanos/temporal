package simulation

import (
	"context"
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

type sqliteOracleCollector struct {
	gomad3sim.Cluster
	results []gomad3sim.OracleResult
}

func (c *sqliteOracleCollector) RecordOracle(_ context.Context, result gomad3sim.OracleResult) error {
	c.results = append(c.results, result)
	return nil
}

func TestSQLiteObservationsForwardToNativeOracleOwner(t *testing.T) {
	ack := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}
	recovered := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 3}
	plan := &OraclePlan{Schema: SQLiteObservationSchema, Prefix: "HISTORY ", MaximumBytes: 4096, SQLite: []SQLiteComparison{{Name: "sqlite.recovered", Acknowledged: ack, Recovered: recovered, ExpectedEvents: []string{"1", "2"}, ExpectedValue: "committed", ExpectedIntegrity: "ok"}}}
	e := &execution{results: map[gomad3sim.NodeHandle]wasi.Result{ack: {Reaped: true, Stdout: []byte("ACK 2\n")}, recovered: {Reaped: true, Stdout: []byte("INTEGRITY ok\nHISTORY 1:committed,2:committed\n")}}}
	collector := &sqliteOracleCollector{}
	if err := recordSQLiteOracles(t.Context(), collector, e, plan); err != nil {
		t.Fatal(err)
	}
	if len(collector.results) != 3 {
		t.Fatalf("native oracles missing: %+v", collector.results)
	}
	for _, result := range collector.results {
		if !result.Passed {
			t.Fatalf("committed SQLite observation failed: %+v", result)
		}
	}
}

func TestSQLiteOraclePlanRequiresSealedIntent(t *testing.T) {
	spec := gomad3sim.Spec{Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "sqlite"}}}
	good := &OraclePlan{Schema: SQLiteObservationSchema, Prefix: "HISTORY ", MaximumBytes: 4096, SQLite: []SQLiteComparison{{Name: "sqlite.recovered", Acknowledged: gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, Recovered: gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 3}, ExpectedEvents: []string{"1", "2"}, ExpectedValue: "committed", ExpectedIntegrity: "ok"}}}
	if err := validateOraclePlan(good, spec); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*OraclePlan){
		func(p *OraclePlan) { p.Schema = "unknown" },
		func(p *OraclePlan) { p.Prefix = "arbitrary " },
		func(p *OraclePlan) { p.Histories = []HistoryComparison{{Name: "mixed"}} },
		func(p *OraclePlan) { p.SQLite[0].Acknowledged.Node = "missing" },
		func(p *OraclePlan) { p.SQLite[0].Recovered = p.SQLite[0].Acknowledged },
		func(p *OraclePlan) { p.SQLite[0].ExpectedEvents = nil },
		func(p *OraclePlan) { p.SQLite[0].ExpectedEvents = []string{"01"} },
		func(p *OraclePlan) { p.SQLite[0].ExpectedValue = "comma,ambiguous" },
		func(p *OraclePlan) { p.SQLite[0].ExpectedIntegrity = "unchecked" },
	} {
		data, err := json.Marshal(good)
		if err != nil {
			t.Fatal(err)
		}
		var changed OraclePlan
		if err := json.Unmarshal(data, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		if err := validateOraclePlan(&changed, spec); err == nil {
			t.Fatal("unbound SQLite intent admitted")
		}
	}
}

func TestSQLiteObservationUsesFreshFixedLines(t *testing.T) {
	good := []byte("INTEGRITY ok\nHISTORY 1:committed,2:committed\n")
	value, err := decodeSQLiteObservation(good, 4096)
	if err != nil || value.Integrity != "ok" || len(value.IDs) != 2 || value.IDs[1] != "2" || value.Values[1] != "committed" {
		t.Fatalf("actual fixture observations not decoded: %+v %v", value, err)
	}
	for _, output := range []string{"", "INTEGRITY ok\n", "INTEGRITY ok\nHISTORY 1:committed\nHISTORY 2:committed\n", "INTEGRITY ok\nINTEGRITY ok\nHISTORY 1:committed\n", "INTEGRITY ok\nHISTORY x:committed\n", "INTEGRITY ok\nHISTORY 1:committed,\n"} {
		if _, err := decodeSQLiteObservation([]byte(output), 4096); err == nil {
			t.Fatalf("ambiguous SQLite output admitted: %q", output)
		}
	}
	if _, err := decodeSQLiteObservation(good, 1); err == nil {
		t.Fatal("oversized SQLite output admitted")
	}
	if id, err := decodeSQLiteAcknowledgement([]byte("ACK 2\n"), 4096); err != nil || id != "2" {
		t.Fatalf("ACK2 not observed: %s %v", id, err)
	}
	if _, err := decodeSQLiteAcknowledgement([]byte("ACK 2\nACK 2\n"), 4096); err == nil {
		t.Fatal("duplicate acknowledgements admitted")
	}
}

func TestProviderStorageCrashRequiresActualDurableBoundary(t *testing.T) {
	model := &wasi.StorageModel{Faults: []wasi.StorageFault{{Action: "crash-before"}}}
	handle := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}
	for _, termination := range []string{"exit", "storage_crash"} {
		broker, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "sqlite", Incarnation: 2}}, Seed: 7, Limits: world.Limits{MaxRequests: 100, MaxEvents: 100, MaxQueuedEvents: 100, MaxTransitions: 100, MaxPayloadBytes: 4096, MaxStringBytes: 1024}})
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		close(done)
		e := &execution{broker: broker, nodes: map[gomad3sim.NodeID]*node{"sqlite": {handle: handle, request: wasi.Request{Config: wasi.Config{Storage: model}}, done: done, cancel: func() {}, result: wasi.Result{Termination: termination, Reaped: true, RuntimeInterrupted: true}}}, provider: Provider{}}
		if _, err := e.Crash(context.Background(), handle); err == nil {
			t.Fatal("absent durable image was treated as a modeled storage crash")
		}
	}
}

func TestProviderOutputCrashKeepsItsExistingBoundary(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}
	done := make(chan struct{})
	close(done)
	e := &execution{nodes: map[gomad3sim.NodeID]*node{"sqlite": {handle: handle, request: wasi.Request{Config: wasi.Config{Storage: &wasi.StorageModel{Faults: []wasi.StorageFault{{Action: "crash-before"}}}}}, done: done, result: wasi.Result{Termination: "crash", Reaped: true}}}, provider: Provider{OutputCrashes: []OutputCrash{{Handle: handle, Marker: []byte("ACK 2")}}}}
	if _, err := e.Crash(t.Context(), handle); err != nil {
		t.Fatalf("unused storage fault replaced a configured output boundary: %v", err)
	}
}
