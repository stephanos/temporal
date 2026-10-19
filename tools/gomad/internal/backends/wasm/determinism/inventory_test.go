package determinism

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const moduleRoot = "../../../.."

func loadInventory(t *testing.T) Inventory {
	t.Helper()
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestInventoryLoadsAndNamesExistingEvidence(t *testing.T) {
	inventory := loadInventory(t)
	if err := inventory.CheckEvidence(moduleRoot); err != nil {
		t.Fatal(err)
	}
	broken := inventory
	broken.Entries = slices.Clone(inventory.Entries)
	broken.Entries[0].Evidence = append(slices.Clone(broken.Entries[0].Evidence), "go:internal/backends/wasm/determinism:TestRemovedEvidence")
	if err := broken.CheckEvidence(moduleRoot); err == nil || !strings.Contains(err.Error(), "TestRemovedEvidence") {
		t.Fatalf("missing evidence test not named: %v", err)
	}
	broken.Entries[0].Evidence = []string{"rust:tests/helper_corpus.rs:removed_rust_test"}
	if err := broken.CheckEvidence(moduleRoot); err == nil || !strings.Contains(err.Error(), "removed_rust_test") {
		t.Fatalf("missing rust evidence test not named: %v", err)
	}
}

func TestInventoryLoaderRejectsMalformedRows(t *testing.T) {
	mutate := func(change func(map[string]any)) []byte {
		var document map[string]any
		if err := json.Unmarshal(inventoryJSON, &document); err != nil {
			t.Fatal(err)
		}
		change(document)
		data, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	entry := func(document map[string]any, index int) map[string]any {
		return document["entries"].([]any)[index].(map[string]any)
	}
	for _, test := range []struct {
		name, want string
		data       []byte
	}{
		{"unknown top-level field", `unknown field "extra"`, mutate(func(d map[string]any) { d["extra"] = true })},
		{"unknown entry field", `unknown field "note"`, mutate(func(d map[string]any) { entry(d, 0)["note"] = "x" })},
		{"unknown reviewed field", `unknown field "date"`, mutate(func(d map[string]any) { d["reviewed"].(map[string]any)["date"] = "x" })},
		{"reused id", "reused", mutate(func(d map[string]any) { entry(d, 1)["id"] = entry(d, 0)["id"] })},
		{"duplicate site", "has rows", mutate(func(d map[string]any) { entry(d, 1)["site"] = entry(d, 0)["site"] })},
		{"no evidence", "no evidence", mutate(func(d map[string]any) { entry(d, 0)["evidence"] = []any{} })},
		{"bad evidence", "unknown kind", mutate(func(d map[string]any) { entry(d, 0)["evidence"] = []any{"TestOnlyAName"} })},
		{"escape without justification", "justification", mutate(func(d map[string]any) { entry(d, 0)["disposition"] = "escape" })},
		{"justification outside escape", "only for escape", mutate(func(d map[string]any) { entry(d, 0)["justification"] = "x" })},
		{"unknown class", "unknown class", mutate(func(d map[string]any) { entry(d, 0)["class"] = "luck" })},
		{"wrong schema", "schema", mutate(func(d map[string]any) { d["schema"] = "gomad.wasm-determinism-inventory/v0" })},
		{"trailing data", "trailing", append(slices.Clone(inventoryJSON), []byte("{}")...)},
	} {
		if _, err := Parse(test.data); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: err = %v, want %q", test.name, err, test.want)
		}
	}
}

// diagnosticStreams derives the seeded streams from the code: every draw
// counter of the diagnostic record that choice.DiffDiagnostics reports when it
// alone differs.
func diagnosticStreams(t *testing.T) []string {
	t.Helper()
	recordType := reflect.TypeFor[choice.DiagnosticRecord]()
	var streams []string
	for index := range recordType.NumField() {
		field := recordType.Field(index)
		if !strings.HasSuffix(field.Name, "Draws") {
			continue
		}
		var changed choice.DiagnosticRecord
		reflect.ValueOf(&changed).Elem().Field(index).SetUint(1)
		expected, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{}}, 4096)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{changed}, 4096)
		if err != nil {
			t.Fatal(err)
		}
		difference, err := choice.DiffDiagnostics(expected.Bytes, actual.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if difference == nil || len(difference.Fields) != 1 {
			t.Fatalf("DiffDiagnostics does not compare seeded stream %s: %+v", field.Name, difference)
		}
		streams = append(streams, difference.Fields[0])
	}
	if len(streams) == 0 {
		t.Fatal("no seeded streams derived from the diagnostic record")
	}
	return streams
}

func TestInventoryCoversDiagnosticStreams(t *testing.T) {
	inventory := loadInventory(t)
	streams := diagnosticStreams(t)
	if err := inventory.Complete("stream", streams); err != nil {
		t.Fatal(err)
	}
	// Removing a row and adding a stream without one both fail by name.
	if err := inventory.Without("stream:"+streams[0]).Complete("stream", streams); err == nil || !strings.Contains(err.Error(), `"`+streams[0]+`" has no inventory row`) {
		t.Fatalf("removed stream row not reported: %v", err)
	}
	if err := inventory.Complete("stream", append(slices.Clone(streams), "gc_pacer_draws")); err == nil || !strings.Contains(err.Error(), `"gc_pacer_draws" has no inventory row`) {
		t.Fatalf("new stream without a row not reported: %v", err)
	}
	if err := inventory.Complete("stream", streams[1:]); err == nil || !strings.Contains(err.Error(), "no longer has") {
		t.Fatalf("row without a stream not reported: %v", err)
	}
}

func stockRoot(t *testing.T) string {
	t.Helper()
	command := os.Getenv("GOMAD_STOCK_GO")
	if command == "" {
		command = "go"
	}
	cmd := exec.Command(command, "env", "GOROOT")
	cmd.Env = append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local")
	data, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

// TestWasip1SeededDrawCheckDecision pins the decision recorded in row
// runtime.check-seeded-draw: the WASM runtime's host-timed-draw check is a
// no-op because wasip1 has no host-timed execution context. If the check
// gains a body, or the stock runtime gains sysmon or thread creation on wasm,
// the row has to be re-reviewed.
func TestWasip1SeededDrawCheckDecision(t *testing.T) {
	inventory := loadInventory(t)
	row, ok := inventory.Sites("runtime-decision")["gomadDiagnosticCheckSeededDraw"]
	if !ok || row.Disposition != "controlled" {
		t.Fatalf("seeded-draw decision row missing or changed: %+v", row)
	}
	glue, err := os.ReadFile("../toolchain/runtime_wasm.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(glue, []byte("//go:nosplit\nfunc gomadDiagnosticCheckSeededDraw(*m) {}\n")) {
		t.Fatal("WASM gomadDiagnosticCheckSeededDraw is no longer the reviewed no-op")
	}
	for _, marker := range []string{"gomadHostTimed", "gomadHostDraw"} {
		if bytes.Contains(glue, []byte(marker)) {
			t.Fatalf("WASM runtime glue marks host-timed paths (%s); re-review the seeded-draw decision", marker)
		}
	}
	if inventory.Reviewed.GoToolchain != "go1.27.1" {
		t.Fatalf("seeded-draw decision reviewed against go1.27.1, inventory says %s", inventory.Reviewed.GoToolchain)
	}
	root := stockRoot(t)
	version, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(version, []byte(inventory.Reviewed.GoToolchain+"\n")) {
		t.Fatalf("stock toolchain %q differs from reviewed %s", strings.SplitN(string(version), "\n", 2)[0], inventory.Reviewed.GoToolchain)
	}
	for file, premise := range map[string]string{
		"src/runtime/proc.go":    `const haveSysmon = GOARCH != "wasm"`,
		"src/runtime/os_wasm.go": "func newosproc(mp *m) {\n\tthrow(\"newosproc: not implemented\")\n}",
	} {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte(premise)) {
			t.Fatalf("%s no longer contains %q; re-review the seeded-draw decision", file, premise)
		}
	}
}
