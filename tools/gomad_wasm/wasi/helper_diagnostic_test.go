package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func diagnosticFixture(t *testing.T) []byte {
	t.Helper()
	phases := map[string]any{}
	for _, name := range strings.Fields("module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute") {
		phases[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "user_nanos": 0, "system_nanos": 0, "cpu_samples": 0, "peak_rss_bytes_high_water": 0}
	}
	phases["execute"] = map[string]uint64{"count": 1, "wall_nanos": 100, "user_nanos": 70, "system_nanos": 10, "cpu_samples": 1, "peak_rss_bytes_high_water": 1024}
	abi := map[string]any{}
	for _, name := range strings.Fields("clock scheduler select observation idle other_local remote_callback") {
		abi[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "instantiate_wall_nanos": 0, "execute_wall_nanos": 0}
	}
	abi["clock"] = map[string]uint64{"count": 1, "wall_nanos": 30, "instantiate_wall_nanos": 0, "execute_wall_nanos": 30}
	nested := map[string]any{}
	for _, name := range strings.Fields("alternative_hash encoding transcript_hash flush remote_write remote_read_wait") {
		nested[name] = map[string]uint64{"count": 0, "wall_nanos": 0}
	}
	data, err := json.Marshal(map[string]any{"schema": "gomad3.wasm-profile/v1", "diagnostic_only": true, "backend": "cranelift-jit", "host_os": "linux", "host_arch": "aarch64", "phases": phases, "abi": abi, "nested": nested, "residual": map[string]uint64{"instantiate_wall_nanos": 0, "execute_wall_nanos": 70}, "abi_overlap_errors": 0})
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte("gomad-wasm-profile-v1 "), data...), '\n')
}

func TestDiagnosticReportRejectsMalformedOrUnattributedMeasurements(t *testing.T) {
	valid := diagnosticFixture(t)
	report, err := decodeDiagnosticReport(valid)
	if err != nil || report.Phases["execute"].UserNanos != 70 || report.ABI["clock"].ExecuteWallNanos != 30 || report.Residual.ExecuteWallNanos != 70 {
		t.Fatalf("literal phase profile: report=%+v err=%v", report, err)
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"missing", nil},
		{"oversized", bytes.Repeat([]byte{'x'}, 16<<10+1)},
		{"duplicate-report", append(append([]byte(nil), valid...), valid...)},
		{"trailing", append(append([]byte(nil), valid...), []byte("unexpected\n")...)},
		{"null", []byte("gomad-wasm-profile-v1 null\n")},
		{"wrong-schema", bytes.Replace(valid, []byte("gomad3.wasm-profile/v1"), []byte("gomad3.wasm-profile/v2"), 1)},
		{"unknown-top", bytes.Replace(valid, []byte(`"schema":`), []byte(`"extra":0,"schema":`), 1)},
		{"nested-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":1,"user_nanos":70`), 1)},
		{"escaped-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"\u0075ser_nanos":1,"user_nanos":70`), 1)},
		{"missing-phase", bytes.Replace(valid, []byte(`"module_read":`), []byte(`"absent":`), 1)},
		{"missing-measurement", bytes.Replace(valid, []byte(`"user_nanos":70,`), nil, 1)},
		{"unknown-measurement", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":70,"invented":0`), 1)},
		{"not-diagnostic", bytes.Replace(valid, []byte(`"diagnostic_only":true`), []byte(`"diagnostic_only":false`), 1)},
		{"not-jit", bytes.Replace(valid, []byte("cranelift-jit"), []byte("interpreter"), 1)},
		{"negative", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":-1`), 1)},
		{"too-many-cpu-samples", bytes.Replace(valid, []byte(`"cpu_samples":1`), []byte(`"cpu_samples":2`), 1)},
		{"overlapping-abi", bytes.Replace(valid, []byte(`"abi_overlap_errors":0`), []byte(`"abi_overlap_errors":1`), 1)},
		{"wrong-residual", bytes.Replace(valid, []byte(`"execute_wall_nanos":70`), []byte(`"execute_wall_nanos":71`), 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeDiagnosticReport(test.data); err == nil {
				t.Fatalf("accepted %s diagnostic", test.name)
			}
		})
	}
}

type diagnosticPhase struct {
	Count                 uint64 `json:"count"`
	WallNanos             uint64 `json:"wall_nanos"`
	UserNanos             uint64 `json:"user_nanos"`
	SystemNanos           uint64 `json:"system_nanos"`
	CPUSamples            uint64 `json:"cpu_samples"`
	PeakRSSBytesHighWater uint64 `json:"peak_rss_bytes_high_water"`
}
type diagnosticABI struct {
	Count                uint64 `json:"count"`
	WallNanos            uint64 `json:"wall_nanos"`
	InstantiateWallNanos uint64 `json:"instantiate_wall_nanos"`
	ExecuteWallNanos     uint64 `json:"execute_wall_nanos"`
}
type diagnosticNested struct {
	Count     uint64 `json:"count"`
	WallNanos uint64 `json:"wall_nanos"`
}
type diagnosticResidual struct {
	InstantiateWallNanos uint64 `json:"instantiate_wall_nanos"`
	ExecuteWallNanos     uint64 `json:"execute_wall_nanos"`
}
type diagnosticReport struct {
	Schema           string                      `json:"schema"`
	DiagnosticOnly   bool                        `json:"diagnostic_only"`
	Backend          string                      `json:"backend"`
	HostOS           string                      `json:"host_os"`
	HostArch         string                      `json:"host_arch"`
	Phases           map[string]diagnosticPhase  `json:"phases"`
	ABI              map[string]diagnosticABI    `json:"abi"`
	Nested           map[string]diagnosticNested `json:"nested"`
	Residual         diagnosticResidual          `json:"residual"`
	ABIOverlapErrors uint64                      `json:"abi_overlap_errors"`
}

func decodeDiagnosticReport(data []byte) (diagnosticReport, error) {
	var report diagnosticReport
	prefix := []byte("gomad-wasm-profile-v1 ")
	if len(data) > 16<<10 || !bytes.HasPrefix(data, prefix) || !bytes.HasSuffix(data, []byte{'\n'}) || bytes.Count(data, []byte{'\n'}) != 1 {
		return report, fmt.Errorf("missing, oversized or multiple diagnostic report")
	}
	data = data[len(prefix) : len(data)-1]
	if err := validateJSON(data); err != nil {
		return report, err
	}
	fields, err := diagnosticFields(data, "schema diagnostic_only backend host_os host_arch phases abi nested residual abi_overlap_errors")
	if err != nil {
		return report, err
	}
	for _, group := range []struct{ name, keys, measurements string }{
		{"phases", "module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute", "count wall_nanos user_nanos system_nanos cpu_samples peak_rss_bytes_high_water"},
		{"abi", "clock scheduler select observation idle other_local remote_callback", "count wall_nanos instantiate_wall_nanos execute_wall_nanos"},
		{"nested", "alternative_hash encoding transcript_hash flush remote_write remote_read_wait", "count wall_nanos"},
	} {
		entries, err := diagnosticFields(fields[group.name], group.keys)
		if err != nil {
			return report, err
		}
		for _, entry := range entries {
			if _, err := diagnosticFields(entry, group.measurements); err != nil {
				return report, err
			}
		}
	}
	if _, err := diagnosticFields(fields["residual"], "instantiate_wall_nanos execute_wall_nanos"); err != nil {
		return report, err
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	if report.Schema != "gomad3.wasm-profile/v1" || !report.DiagnosticOnly || report.Backend != "cranelift-jit" || report.HostOS != "linux" && report.HostOS != "macos" || report.HostArch != "aarch64" && report.HostArch != "x86_64" || report.ABIOverlapErrors != 0 {
		return report, fmt.Errorf("diagnostic identity or non-overlap claim")
	}
	for _, phase := range report.Phases {
		if phase.CPUSamples > phase.Count || phase.CPUSamples == 0 && (phase.UserNanos != 0 || phase.SystemNanos != 0) {
			return report, fmt.Errorf("diagnostic CPU sample claim")
		}
	}
	instantiate, execute := report.Residual.InstantiateWallNanos, report.Residual.ExecuteWallNanos
	for _, bucket := range report.ABI {
		if bucket.InstantiateWallNanos > math.MaxUint64-bucket.ExecuteWallNanos || bucket.InstantiateWallNanos+bucket.ExecuteWallNanos > bucket.WallNanos || bucket.InstantiateWallNanos > math.MaxUint64-instantiate || bucket.ExecuteWallNanos > math.MaxUint64-execute {
			return report, fmt.Errorf("diagnostic ABI wall overflow or attribution")
		}
		instantiate += bucket.InstantiateWallNanos
		execute += bucket.ExecuteWallNanos
	}
	if instantiate != report.Phases["instantiate"].WallNanos || execute != report.Phases["execute"].WallNanos {
		return report, fmt.Errorf("diagnostic residual attribution")
	}
	return report, nil
}

func diagnosticFields(data []byte, keys string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	names := strings.Fields(keys)
	if len(fields) != len(names) {
		return nil, fmt.Errorf("diagnostic field inventory")
	}
	for _, name := range names {
		if len(fields[name]) == 0 || bytes.Equal(fields[name], []byte("null")) {
			return nil, fmt.Errorf("missing diagnostic field %s", name)
		}
	}
	return fields, nil
}
