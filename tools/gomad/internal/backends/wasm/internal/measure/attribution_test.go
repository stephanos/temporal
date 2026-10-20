package measure

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// diagnosticV2Fixture is a schema v2 report: one clock import of 30 ns whose
// clock frame (20 ns exclusive) contains a 10 ns encoding span, reconciled
// against its execute wall; mutate edits the document before encoding.
func diagnosticV2Fixture(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(bytes.TrimPrefix(diagnosticFixture(t), []byte("gomad-wasm-profile-v1 ")), []byte("\n")), &document); err != nil {
		t.Fatal(err)
	}
	series := func(count, wall, exclusive, sample uint64) map[string]any {
		return map[string]any{"count": count, "wall_nanos": wall, "exclusive_wall_nanos": exclusive, "instantiate_wall_nanos": 0, "execute_wall_nanos": wall, "execute_exclusive_wall_nanos": exclusive, "min_nanos": sample, "median_nanos": sample, "p99_nanos": sample, "max_nanos": sample}
	}
	op := series(1, 30, 20, 30)
	op["served_by"] = map[string]any{"clock": 1}
	document["schema"] = "gomad.wasm-profile/v2"
	document["abi_excluded"] = map[string]uint64{"count": 0, "wall_nanos": 0, "instantiate_wall_nanos": 0, "execute_wall_nanos": 0}
	document["abi_ops"] = map[string]any{"wasi_snapshot_preview1.clock_time_get": op}
	document["owners"] = map[string]any{"clock": series(1, 30, 20, 20), "encoding": series(1, 10, 10, 10), "instrumentation": series(1, 2, 2, 2)}
	document["reconciliation"] = map[string]uint64{"execute_wall_nanos": 100, "attributed_host_wall_nanos": 32, "guest_residual_wall_nanos": 68, "outer_host_wall_nanos": 32, "difference_nanos": 0, "tolerance_nanos": 50_000, "distribution_relative_error_ppm": 62_500}
	document["threads"], document["threads_omitted"] = []any{}, 0
	document["unattributed_imports"], document["unframed_imports"], document["span_errors"] = 0, 0, 0
	// Round-trip so every nested object is a map[string]any for mutate.
	data, err := json.Marshal(document)
	if err == nil {
		document = nil
		err = json.Unmarshal(data, &document)
	}
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(document)
	}
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte("gomad-wasm-profile-v1 "), data...), '\n')
}

func TestDiagnosticAttributionRequiresEveryImportAndReconciliation(t *testing.T) {
	report, err := decodeDiagnosticReport(diagnosticV2Fixture(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := requireAttributedProfile(report); err != nil {
		t.Fatalf("valid attributed profile: %v", err)
	}
	if op := report.ABIOps["wasi_snapshot_preview1.clock_time_get"]; op.ServedBy["clock"] != 1 || report.Owners["encoding"].ExecuteExclusiveWallNanos != 10 {
		t.Fatalf("decoded attribution: %+v", report)
	}
	v1, err := decodeDiagnosticReport(diagnosticFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if requireAttributedProfile(v1) == nil {
		t.Fatal("v1 profile accepted as attributed")
	}
	at := func(document map[string]any, path ...string) map[string]any {
		value := document
		for _, key := range path {
			value = value[key].(map[string]any)
		}
		return value
	}
	clockOp := []string{"abi_ops", "wasi_snapshot_preview1.clock_time_get"}
	// The decoder refuses a malformed or inconsistent v2 inventory.
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"unknown-op", func(d map[string]any) { at(d, "abi_ops")["wasi_snapshot_preview1.fd_invented"] = at(d, clockOp...) }},
		{"unknown-owner", func(d map[string]any) { at(d, "owners")["kernel"] = at(d, "owners", "clock") }},
		{"unknown-served-owner", func(d map[string]any) { at(d, clockOp...)["served_by"] = map[string]any{"kernel": 1} }},
		{"missing-op-field", func(d map[string]any) { delete(at(d, clockOp...), "p99_nanos") }},
		{"missing-reconciliation-field", func(d map[string]any) { delete(at(d, "reconciliation"), "tolerance_nanos") }},
		{"missing-counter", func(d map[string]any) { delete(d, "span_errors") }},
		{"missing-thread-field", func(d map[string]any) {
			d["threads"] = []any{map[string]any{"slot": 1, "incarnation": 2, "execute_wall_nanos": 1}}
		}},
		{"served-exceeds-calls", func(d map[string]any) { at(d, clockOp...)["served_by"] = map[string]any{"clock": 1, "remote": 1} }},
		{"ops-differ-from-buckets", func(d map[string]any) { at(d, clockOp...)["count"] = 2 }},
		{"exclusive-exceeds-wall", func(d map[string]any) { at(d, "owners", "clock")["exclusive_wall_nanos"] = 31 }},
		{"unordered-distribution", func(d map[string]any) { at(d, clockOp...)["median_nanos"] = 31 }},
		{"execute-row-mismatch", func(d map[string]any) { at(d, "reconciliation")["execute_wall_nanos"] = 99 }},
	} {
		t.Run("decode-"+test.name, func(t *testing.T) {
			if _, err := decodeDiagnosticReport(diagnosticV2Fixture(t, test.mutate)); err == nil {
				t.Fatalf("accepted %s", test.name)
			}
		})
	}
	// The collector gate refuses any unattributed import or reconciliation miss.
	for _, test := range []struct {
		name, want string
		mutate     func(map[string]any)
	}{
		{"unattributed-import", "attribution errors", func(d map[string]any) { d["unattributed_imports"] = 1 }},
		{"unframed-import", "attribution errors", func(d map[string]any) { d["unframed_imports"] = 1 }},
		{"span-error", "attribution errors", func(d map[string]any) { d["span_errors"] = 1 }},
		{"op-without-owner", "have an owner", func(d map[string]any) { at(d, clockOp...)["served_by"] = map[string]any{} }},
		{"missing-serving-owner", "serving owner clock is missing", func(d map[string]any) {
			delete(at(d, "owners"), "clock")
			for _, field := range []string{"wall_nanos", "exclusive_wall_nanos", "execute_wall_nanos", "execute_exclusive_wall_nanos", "min_nanos", "median_nanos", "p99_nanos", "max_nanos"} {
				at(d, "owners", "encoding")[field] = 30
			}
		}},
		{"owner-wall-short", "owner wall", func(d map[string]any) { at(d, "owners", "encoding")["execute_exclusive_wall_nanos"] = 9 }},
		{"rust-row-disagrees", "reconciliation row", func(d map[string]any) { at(d, "reconciliation")["difference_nanos"] = 3 }},
		{"threads-omitted", "attribution errors", func(d map[string]any) { d["threads_omitted"] = 1 }},
		{"thread-miss", "thread slot=1 incarnation=2 reconciliation miss", func(d map[string]any) {
			d["threads"] = []any{map[string]any{"slot": 1, "incarnation": 2, "execute_wall_nanos": 1_000_000, "attributed_host_wall_nanos": 100_000, "guest_residual_wall_nanos": 800_000, "outer_host_wall_nanos": 200_000, "difference_nanos": 100_000, "tolerance_nanos": 50_000}}
		}},
		{"guest-gaps-do-not-telescope", "guest residual", func(d map[string]any) { at(d, "reconciliation")["outer_host_wall_nanos"] = 33 }},
		{"reconciliation-miss", "reconciliation miss", func(d map[string]any) {
			const execute = 20_000_000
			at(d, "phases", "execute")["wall_nanos"] = execute
			at(d, "residual")["execute_wall_nanos"] = execute - 30
			row := at(d, "reconciliation")
			row["execute_wall_nanos"], row["guest_residual_wall_nanos"], row["outer_host_wall_nanos"] = execute, execute-200_000, 200_000
			row["difference_nanos"], row["tolerance_nanos"] = 200_000-32, 100_000
		}},
	} {
		t.Run("gate-"+test.name, func(t *testing.T) {
			report, err := decodeDiagnosticReport(diagnosticV2Fixture(t, test.mutate))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if err := requireAttributedProfile(report); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("gate error %v, want %q", err, test.want)
			}
		})
	}
}

func TestDiagnosticThreadRowsBindAggregateAndFullIdentities(t *testing.T) {
	t.Parallel()
	report, err := decodeDiagnosticReport(diagnosticV2Fixture(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	thread := diagnosticThread{Slot: 1, Incarnation: 1 << 40, ExecuteWallNanos: 100, AttributedHostWallNanos: 32, GuestResidualWallNanos: 68, OuterHostWallNanos: 32, ToleranceNanos: 50_000}
	report.Threads = []diagnosticThread{thread}
	if err := requireAttributedProfile(report); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		rows []diagnosticThread
	}{
		{"duplicate", []diagnosticThread{thread, thread}},
		{"missing-aggregate-share", []diagnosticThread{{Slot: 1, ExecuteWallNanos: 50, GuestResidualWallNanos: 50, ToleranceNanos: 50_000}}},
		{"too-many", make([]diagnosticThread, 65)},
	} {
		t.Run(test.name, func(t *testing.T) {
			report.Threads = test.rows
			if err := requireAttributedProfile(report); err == nil {
				t.Fatal("accepted incomplete thread attribution")
			}
		})
	}
}
