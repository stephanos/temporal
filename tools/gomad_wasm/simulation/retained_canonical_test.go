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

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3sim"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestRetainedCanonicalDecodeAvoidsWholeBundleEncodingAllocation(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, Record: []byte("{}"), Inputs: map[string][]byte{"module": bytes.Repeat([]byte{0xa5}, 8<<20)}}
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
	t.Logf("actual retained decode allocation=%d ceiling=%d payload=%d input=%d (bounded8MiB allocation regression, not Temporal budget proof)", allocated, ceiling, len(data), 8<<20)
	if allocated > ceiling {
		t.Fatalf("whole-bundle canonical encoding exceeds allocation ceiling: actual=%d maximum=%d", allocated, ceiling)
	}
	t.Logf("current source-bound simulation implementation=%s", ImplementationSHA256())
}

func TestRetainedCanonicalWriterMatchesAllDeclaredFields(t *testing.T) {
	fields := reflect.VisibleFields(reflect.TypeFor[retainedExecution]())
	want := []string{"schema", "implementation_sha256", "spec", "record", "broker", "output_crashes", "output_stops", "incarnation_boots", "output_crash_groups", "oracles", "lifecycle", "boots", "inputs"}
	if len(fields) != len(want) {
		t.Fatal("closed retained writer schema drift: field count changed")
	}
	for i, field := range fields {
		if field.Tag.Get("json") != want[i] {
			t.Fatalf("closed retained writer schema drift: field %d is %s", i, field.Tag)
		}
	}
	populated := retainedExecution{Schema: "<&\u2028", Implementation: "\u2029&", Spec: gomad3sim.Spec{Schema: "<spec>", Seed: 7, Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "b", Config: []byte{1, 2}}}}, Record: []byte{0xff, 1, 2}, Broker: wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}}, Addresses: map[string]string{"<&\u2028": "\u2029"}}, OutputCrashes: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("crash")}}, OutputStops: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "b", Incarnation: 2}, Marker: []byte("stop")}}, IncarnationBoots: []IncarnationBoot{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "restart"}}, OutputCrashGroups: []OutputCrashGroup{{Handles: []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}}, Marker: []byte("ACK")}}, Oracles: &OraclePlan{Schema: "<&", Prefix: "HISTORY ", MaximumBytes: 19, Histories: []HistoryComparison{{Name: "history", ExpectedEvents: []string{"1"}}}}, Lifecycle: []gomad3sim.LifecycleTransition{{Action: gomad3sim.LifecycleStart, Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}}}, Boots: []retainedBoot{{ID: "b", Request: wasi.Request{Config: wasi.Config{Args: []string{"<&\u2028"}}}}}, Inputs: map[string][]byte{"z": {1}, "a<&\u2028": {2, 3}, "nil": nil, "empty": {}, string([]byte{0xff}): {4}}}
	for _, bundle := range []retainedExecution{{}, {Record: []byte{}, Inputs: map[string][]byte{}, Boots: []retainedBoot{}, Lifecycle: []gomad3sim.LifecycleTransition{}, OutputCrashes: []OutputCrash{}, OutputStops: []OutputCrash{}, IncarnationBoots: []IncarnationBoot{}, OutputCrashGroups: []OutputCrashGroup{}, Oracles: &OraclePlan{}}, populated} {
		assertRetainedCanonicalBytes(t, bundle)
	}
	for _, size := range []int{0, 1, 2, 3, 1022, 1023, 1024, 1025, 2047, 2048, 2049, 8191, 8192, 8193} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			data := bytes.Repeat([]byte{0xff}, size)
			assertRetainedCanonicalBytes(t, retainedExecution{Schema: RetainedSchema, Record: data, Inputs: map[string][]byte{"z": data, "a": data, "empty": {}, "nil": nil}})
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
	if !retainedCanonicalEqual(bundle, expected) || retainedCanonicalEqual(bundle, append(bytes.Clone(expected), ' ')) || retainedCanonicalEqual(bundle, expected[:len(expected)-1]) {
		t.Fatal("canonical comparator failed exact byte/EOF admission")
	}
}

func TestRetainedCanonicalDecodePreservesStrictErrorOracle(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, Implementation: "<&\u2028", Record: []byte("abc"), Inputs: map[string][]byte{"a": {1}, "z": {2}}}
	canonical, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	base := string(canonical)
	mutations := []string{
		base,
		strings.Replace(base, `"schema":`, `"Schema":`, 1),
		strings.Replace(base, `"schema":`, `"schema":"duplicate","schema":`, 1),
		base[:len(base)-1] + `,"unknown":true}`,
		strings.Replace(base, `"output_stops":null,`, "", 1),
		strings.Replace(base, `"spec":{`, `"spec":{"unknown":true,`, 1),
		strings.Replace(base, `"spec":{"schema":"",`, `"spec":{"schema":"","schema":"",`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":1`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":"***"`, 1),
		strings.Replace(base, `"record":"YWJj"`, `"record":"\u0059WJj"`, 1),
		strings.Replace(base, `"inputs":{"a":"AQ==","z":"Ag=="}`, `"inputs":{"z":"Ag==","a":"AQ=="}`, 1),
		strings.Replace(base, `"a":"AQ=="`, `"a":"AQ==","a":"Ag=="`, 1),
		strings.Replace(base, `\u003c`, "<", 1),
		strings.Replace(base, `\u003c`, string([]byte{0xff}), 1),
		strings.Replace(base, `"schema":"gomad3.wasm-simulation-retained/v1","implementation_sha256":"\u003c\u0026\u2028"`, `"implementation_sha256":"\u003c\u0026\u2028","schema":"gomad3.wasm-simulation-retained/v1"`, 1),
		base + " ", base + "{}", base[:len(base)-1],
	}
	for i, data := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			payload := target.BackendPayload{Data: []byte(data), Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.HashBytes([]byte(data)), Bytes: record.Uint64String(len(data))}}
			actual, actualErr := decodeRetained(payload)
			expected, expectedErr := oldRetainedDecodeOracle(payload)
			if fmt.Sprint(actualErr) != fmt.Sprint(expectedErr) || actualErr == nil && !reflect.DeepEqual(actual, expected) {
				t.Fatalf("strict decoder/canonical admission changed: actual=%v old=%v", actualErr, expectedErr)
			}
			if i != 0 && actualErr == nil {
				t.Fatal("malformed/noncanonical retained payload admitted")
			}
		})
	}
}

func oldRetainedDecodeOracle(payload target.BackendPayload) (retainedExecution, error) {
	if payload.Reference.Schema != RetainedSchema || payload.Reference.File != "backend/simulation.json" || uint64(payload.Reference.Bytes) != uint64(len(payload.Data)) || record.HashBytes(payload.Data) != payload.Reference.SHA256 || len(payload.Data) > 512<<20 {
		return retainedExecution{}, errors.New("WASM retained payload reference mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload.Data))
	decoder.DisallowUnknownFields()
	var bundle retainedExecution
	if err := decoder.Decode(&bundle); err != nil {
		return retainedExecution{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return retainedExecution{}, errors.New("trailing retained WASM payload")
	}
	canonical, err := json.Marshal(bundle)
	if err != nil || !bytes.Equal(canonical, payload.Data) || bundle.Schema != RetainedSchema {
		return retainedExecution{}, errors.New("noncanonical retained WASM payload")
	}
	return bundle, nil
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
	bundle := retainedExecution{Schema: RetainedSchema, Record: bytes.Repeat([]byte{1}, 2049), Inputs: map[string][]byte{"a": {1, 2}}}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("actual writer rejected bytes")
	closeBudget := strings.Index(string(encoded), `"a":"AQI="`) + len(`"a":"`) + 2
	for _, budget := range []int{0, 1, 19, 100, 1024, closeBudget, len(encoded) - 1} {
		if err := writeRetainedCanonical(&retainedFailureWriter{remaining: budget, err: sentinel}, bundle); !errors.Is(err, sentinel) {
			t.Fatalf("write/close error lost at byte %d: %v", budget, err)
		}
		if err := writeRetainedCanonical(&retainedFailureWriter{remaining: budget}, bundle); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write accepted at byte %d: %v", budget, err)
		}
	}
}
