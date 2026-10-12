package simulation

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3sim"
)

func TestRetainedAdmissionConsumesPrivateInputsOnlyOnce(t *testing.T) {
	module, helper := []byte("retained-module"), []byte("retained-helper")
	moduleID, helperID := strings.TrimPrefix(string(record.HashBytes(module)), "sha256:"), strings.TrimPrefix(string(record.HashBytes(helper)), "sha256:")
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": {ModuleSHA256: moduleID, HelperSHA256: helperID, Engine: wasi.PinnedEngine()}}}
	data, err := json.Marshal(retainedExecution{Schema: RetainedSchema, Boots: []retainedBoot{{ID: "a"}}, Inputs: map[string][]byte{moduleID: module, helperID: helper}, Record: []byte("original-canonical-record")})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := decodeRetained(target.BackendPayload{Data: data, Reference: record.BackendPayload{Schema: RetainedSchema, File: "backend/simulation.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}})
	if err != nil {
		t.Fatal(err)
	}
	clear(data)
	if string(bundle.Record) != "original-canonical-record" {
		t.Fatal("decoded original record aliases caller payload")
	}
	installed := installedHelperFile(t, helper)
	a := &retainedAdmission{bundle: bundle, provider: p}
	directory := t.TempDir()
	_, reopened, err := a.reopen(directory, installed)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(reopened.Boots["a"].ModulePath); err != nil || !bytes.Equal(data, module) {
		t.Fatalf("private module bytes changed: %v", err)
	}
	if reopened.Boots["a"].HelperPath != installed.Path {
		t.Fatal("retained replay does not execute the installed helper")
	}
	if entries, err := os.ReadDir(filepath.Dir(reopened.Boots["a"].ModulePath)); err != nil || len(entries) != 1 {
		t.Fatalf("retained replay wrote helper bytes: %v %v", entries, err)
	}
	if a.bundle.Inputs != nil || a.provider != nil {
		t.Fatal("consumed admission retains executable buffers or a reusable provider")
	}
	if _, _, err := a.reopen(directory, installed); err == nil {
		t.Fatal("consumed admission could reopen again")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("second consumption wrote executable inputs: %v %v", entries, err)
	}
	failed := &retainedAdmission{bundle: bundle, provider: p}
	if _, _, err := failed.reopen(filepath.Join(directory, "missing"), installed); err == nil {
		t.Fatal("invalid replay directory accepted")
	}
	if _, _, err := failed.reopen(directory, installed); err == nil {
		t.Fatal("failed single-use admission could be reused")
	}
}

func installedHelperFile(t *testing.T, helper []byte) InstalledHelper {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gomad3-wasmhost")
	if err := os.WriteFile(path, helper, 0700); err != nil {
		t.Fatal(err)
	}
	return InstalledHelper{Path: path, SHA256: strings.TrimPrefix(string(record.HashBytes(helper)), "sha256:"), Engine: wasi.PinnedEngine()}
}

// A resealed artifact is self-consistent, so only the installation can decide
// which native helper runs.
func TestRetainedReplayRequiresTheInstalledHelper(t *testing.T) {
	module, retained := []byte("retained-module"), []byte("resealed-helper")
	moduleID, retainedID := strings.TrimPrefix(string(record.HashBytes(module)), "sha256:"), strings.TrimPrefix(string(record.HashBytes(retained)), "sha256:")
	installed := installedHelperFile(t, []byte("installed-helper"))
	changedEngine := installed
	changedEngine.Engine.Version += "-changed"
	changedFile := installed
	changedFile.SHA256 = retainedID
	for _, test := range []struct {
		name      string
		boot      wasi.Request
		installed InstalledHelper
		names     []string
	}{
		{"retained-helper", wasi.Request{ModuleSHA256: moduleID, HelperSHA256: retainedID, Engine: installed.Engine}, installed, []string{retainedID, installed.SHA256}},
		{"retained-engine", wasi.Request{ModuleSHA256: moduleID, HelperSHA256: installed.SHA256, Engine: installed.Engine}, changedEngine, []string{changedEngine.Engine.Version}},
		{"installed-bytes", wasi.Request{ModuleSHA256: moduleID, HelperSHA256: retainedID, Engine: installed.Engine}, changedFile, nil},
		{"no-installation", wasi.Request{ModuleSHA256: moduleID, HelperSHA256: retainedID, Engine: installed.Engine}, InstalledHelper{}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": test.boot}}
			a := &retainedAdmission{bundle: retainedExecution{Boots: []retainedBoot{{ID: "a"}}, Inputs: map[string][]byte{moduleID: module, retainedID: retained}}, provider: p}
			directory := t.TempDir()
			_, reopened, err := a.reopen(directory, test.installed)
			if err == nil || reopened != nil {
				t.Fatal("retained replay admitted a helper other than the installation's")
			}
			for _, name := range test.names {
				if !strings.Contains(err.Error(), name) {
					t.Fatalf("rejection %q does not name %s", err, name)
				}
			}
			if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
				t.Fatalf("rejected helper identity wrote executable inputs: %v %v", entries, err)
			}
		})
	}
}

func TestBoundRetainedReplayRejectsBeforeWritingInputs(t *testing.T) {
	directory := t.TempDir()
	for _, primary := range []string{"", "sha256:" + strings.Repeat("0", 64)} {
		_, original, provider, err := ReplayBoundRetained(t.Context(), target.BackendPayload{}, primary, directory, InstalledHelper{})
		if err == nil || original != nil || provider != nil {
			t.Fatal("bound replay bypassed complete retained admission")
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid bound replay wrote inputs: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "compiled-cache")); !os.IsNotExist(err) {
		t.Fatal("invalid bound replay created a cache")
	}
}

func TestRetainedPrimaryBindingRejectsMissingAndForeignModules(t *testing.T) {
	primary := "sha256:" + strings.Repeat("a", 64)
	observed := gomad3sim.ClusterRecord{WASM: &gomad3sim.WASMExecutionEvidence{Incarnations: []gomad3sim.WASMIncarnationEvidence{{ModuleSHA256: strings.Repeat("a", 64)}}}}
	if err := bindRetainedPrimary(observed, primary); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{"", strings.Repeat("a", 64), "sha256:" + strings.Repeat("b", 64)} {
		if err := bindRetainedPrimary(observed, identity); err == nil {
			t.Fatalf("unbound primary accepted: %q", identity)
		}
	}
	if err := bindRetainedPrimary(gomad3sim.ClusterRecord{}, primary); err == nil {
		t.Fatal("missing incarnation evidence bound primary")
	}
}

func TestRetainedLifecycleRejectsHostCallbacksBeforeReplay(t *testing.T) {
	for _, plan := range []gomad3sim.ReplayPlan{
		{Faults: []gomad3sim.FaultRealization{{}}},
		{Scenarios: []gomad3sim.ScenarioDecision{{}}},
		{History: []gomad3sim.HistoryOperation{{}}},
		{Observations: []gomad3sim.Observation{{}}},
		{Oracles: []gomad3sim.OracleResult{{}}},
	} {
		if err := validateRetainedLifecycle(gomad3sim.Spec{Replay: &plan}, &Provider{}); err == nil {
			t.Fatal("host callback or unbound oracle accepted by retained lifecycle owner")
		}
	}
}
