package simulation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3sim"
)

func TestRetainedCanonicalDecodeAvoidsWholeBundleEncodingAllocation(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, Record: bytes.Repeat([]byte{0xa5}, 8<<20), ModuleSHA256: strings.Repeat("a", 64)}
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	payload := target.BackendPayload{Data: data, Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	decoded, err := decodeRetained(payload)
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(decoded)
	if err != nil {
		t.Fatal(err)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	const ceiling = 56 << 20
	t.Logf("actual retained decode allocation=%d ceiling=%d payload=%d record=%d (bounded8MiB allocation regression, not Temporal budget proof)", allocated, ceiling, len(data), 8<<20)
	if allocated > ceiling {
		t.Fatalf("whole-bundle canonical encoding exceeds allocation ceiling: actual=%d maximum=%d", allocated, ceiling)
	}
	t.Logf("current source-bound simulation implementation=%s", ImplementationSHA256())
}

func TestRetainedCanonicalWriterMatchesAllDeclaredFields(t *testing.T) {
	fields := reflect.VisibleFields(reflect.TypeFor[retainedExecution]())
	want := []string{"schema", "implementation_sha256", "spec", "record", "broker", "output_crashes", "output_stops", "incarnation_boots", "output_crash_groups", "oracles", "lifecycle", "boots", "module_sha256"}
	if len(fields) != len(want) {
		t.Fatal("closed retained writer schema drift: field count changed")
	}
	for i, field := range fields {
		if field.Tag.Get("json") != want[i] {
			t.Fatalf("closed retained writer schema drift: field %d is %s", i, field.Tag)
		}
	}
	populated := retainedExecution{Schema: "<&\u2028", Implementation: "\u2029&", Spec: gomad3sim.Spec{Schema: "<spec>", Seed: 7, Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "b", Config: []byte{1, 2}}}}, Record: []byte{0xff, 1, 2}, Broker: wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}}, Addresses: map[string]string{"<&\u2028": "\u2029"}}, OutputCrashes: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("crash")}}, OutputStops: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "b", Incarnation: 2}, Marker: []byte("stop")}}, IncarnationBoots: []IncarnationBoot{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "restart"}}, OutputCrashGroups: []OutputCrashGroup{{Handles: []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}}, Marker: []byte("ACK")}}, Oracles: &OraclePlan{Schema: "<&", Prefix: "HISTORY ", MaximumBytes: 19, Histories: []HistoryComparison{{Name: "history", ExpectedEvents: []string{"1"}}}}, Lifecycle: []gomad3sim.LifecycleTransition{{Action: gomad3sim.LifecycleStart, Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}}}, Boots: []retainedBoot{{ID: "b", Request: wasi.Request{Config: wasi.Config{Args: []string{"<&\u2028"}}}}}, ModuleSHA256: "<&\u2028" + string([]byte{0xff})}
	for _, bundle := range []retainedExecution{{}, {Record: []byte{}, Boots: []retainedBoot{}, Lifecycle: []gomad3sim.LifecycleTransition{}, OutputCrashes: []OutputCrash{}, OutputStops: []OutputCrash{}, IncarnationBoots: []IncarnationBoot{}, OutputCrashGroups: []OutputCrashGroup{}, Oracles: &OraclePlan{}}, populated} {
		assertRetainedCanonicalBytes(t, bundle)
	}
	for _, size := range []int{0, 1, 2, 3, 1022, 1023, 1024, 1025, 2047, 2048, 2049, 8191, 8192, 8193} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			data := bytes.Repeat([]byte{0xff}, size)
			assertRetainedCanonicalBytes(t, retainedExecution{Schema: RetainedSchema, Record: data, ModuleSHA256: strings.Repeat("a", 64)})
		})
	}
}

func assertRetainedCanonicalBytes(t *testing.T, bundle retainedExecution) {
	t.Helper()
	expected, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var actual bytes.Buffer
	if err := writeRetainedCanonical(&actual, bundle); err != nil || !bytes.Equal(actual.Bytes(), expected) {
		t.Fatalf("segmented writer changed stdlib canonical bytes: %v", err)
	}
}

func TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, Implementation: "<&\u2028", Record: []byte("abc"), ModuleSHA256: "m"}
	canonical, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	base := string(canonical)
	accepted := []string{
		base,
		strings.Replace(base, `"schema":`, `"Schema":`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":"\u0059WJj"`, 1),
		strings.Replace(base, `\u003c`, "<", 1),
		strings.Replace(base, `"schema":"gomad3.wasm-simulation-retained/v2","implementation_sha256":"\u003c\u0026\u2028"`, `"implementation_sha256":"\u003c\u0026\u2028","schema":"gomad3.wasm-simulation-retained/v2"`, 1),
		base + " ",
		strings.Replace(base, `"output_stops":null,`, "", 1),
	}
	for i, data := range accepted {
		t.Run("accepted-"+fmt.Sprint(i), func(t *testing.T) {
			actual, err := decodeRetained(retainedJSONPayload([]byte(data)))
			if err != nil || !reflect.DeepEqual(actual, bundle) {
				t.Fatalf("ordinary retained spelling rejected or changed values: %v", err)
			}
		})
	}
	mutations := []string{
		strings.Replace(base, `"schema":`, `"schema":"duplicate","schema":`, 1),
		base[:len(base)-1] + `,"unknown":true}`,
		strings.Replace(base, `"spec":{`, `"spec":{"unknown":true,`, 1),
		strings.Replace(base, `"spec":{"schema":"",`, `"spec":{"schema":"","schema":"",`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":1`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":"***"`, 1),
		strings.Replace(base, `"module_sha256":"m"`, `"module_sha256":"m","module_sha256":"n"`, 1),
		strings.Replace(base, `,"module_sha256":"m"`, ``, 1),
		strings.Replace(base, `"module_sha256":"m"`, `"module_sha256":"m","inputs":{}`, 1),
		strings.Replace(base, `\u003c`, string([]byte{0xff}), 1),
		base + "{}", base[:len(base)-1],
	}
	for i, data := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if _, err := decodeRetained(retainedJSONPayload([]byte(data))); err == nil {
				t.Fatal("malformed retained payload admitted")
			}
		})
	}
}

func retainedJSONPayload(data []byte) target.BackendPayload {
	return target.BackendPayload{Data: data, Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}}
}

type retainedFailureWriter struct {
	remaining int
	err       error
}

func (w *retainedFailureWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}
	w.remaining -= len(data)
	return len(data), nil
}

func TestRetainedCanonicalWriterChecksEveryWriteAndBase64Close(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, Record: bytes.Repeat([]byte{1}, 2050), ModuleSHA256: "m"}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("actual writer rejected bytes")
	closeBudget := strings.Index(string(encoded), `"record":"`) + len(`"record":"`) + 2733
	for _, budget := range []int{0, 1, 19, 100, 1024, closeBudget, len(encoded) - 1} {
		if err := writeRetainedCanonical(&retainedFailureWriter{remaining: budget, err: sentinel}, bundle); !errors.Is(err, sentinel) {
			t.Fatalf("write/close error lost at byte %d: %v", budget, err)
		}
		if err := writeRetainedCanonical(&retainedFailureWriter{remaining: budget}, bundle); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write accepted at byte %d: %v", budget, err)
		}
	}
}
