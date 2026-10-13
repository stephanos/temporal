package simulation

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
)

func TestRetainedOrdinarySpellingKeepsRawAuthenticationAndPresence(t *testing.T) {
	bundle := retainedExecution{Schema: RetainedSchema, ModuleSHA256: strings.Repeat("a", 64), Record: []byte{}, Boots: []retainedBoot{}, Oracles: &OraclePlan{}, OutputStops: nil}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	payload := retainedJSONPayload(data)
	decoded, err := decodeRetained(payload)
	if err != nil || !reflect.DeepEqual(decoded, bundle) {
		t.Fatalf("ordinary evidence lost nil/empty or pointer presence: %v", err)
	}
	for _, mutate := range []func(*target.BackendPayload){
		func(p *target.BackendPayload) { p.Reference.SHA256 = record.HashBytes([]byte("foreign bytes")) },
		func(p *target.BackendPayload) { p.Reference.Bytes++ },
		func(p *target.BackendPayload) { p.Reference.File = "backend/foreign.json" },
		func(p *target.BackendPayload) { p.Data = append(bytes.Clone(p.Data), '\n') },
	} {
		bad := payload
		mutate(&bad)
		if _, err := decodeRetained(bad); err == nil || !strings.Contains(err.Error(), "payload reference mismatch") {
			t.Fatalf("ordinary spelling bypassed raw authentication: %v", err)
		}
	}
}

func TestRetainedAdmissionConsumesPrivateInputsOnlyOnce(t *testing.T) {
	module, helper := []byte("retained-module"), []byte("retained-helper")
	moduleID, helperID := strings.TrimPrefix(string(record.HashBytes(module)), "sha256:"), strings.TrimPrefix(string(record.HashBytes(helper)), "sha256:")
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": {ModuleSHA256: moduleID, HelperSHA256: helperID, Engine: wasi.PinnedEngine()}}}
	data, err := json.Marshal(retainedExecution{Schema: RetainedSchema, Boots: []retainedBoot{{ID: "a"}}, ModuleSHA256: moduleID, Record: []byte("original-canonical-record")})
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
	source := retainedModuleBytes(t, module)
	a := &retainedAdmission{bundle: bundle, provider: p}
	if a.bundle.Record == nil || a.bundle.ModuleSHA256 == "" {
		t.Fatal("admission did not start with the decoded executable bundle")
	}
	directory := t.TempDir()
	_, reopened, err := a.reopen(source, directory, installed)
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
	if !reflect.DeepEqual(a.bundle, retainedExecution{}) || a.provider != nil {
		t.Fatal("consumed admission retains executable buffers or a reusable provider")
	}
	if _, _, err := a.reopen(source, directory, installed); err == nil {
		t.Fatal("consumed admission could reopen again")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("second consumption wrote executable inputs: %v %v", entries, err)
	}
	failed := &retainedAdmission{bundle: bundle, provider: p}
	if _, _, err := failed.reopen(source, filepath.Join(directory, "missing"), installed); err == nil {
		t.Fatal("invalid replay directory accepted")
	}
	if _, _, err := failed.reopen(source, directory, installed); err == nil {
		t.Fatal("failed single-use admission could be reused")
	}
}

func retainedModuleBytes(t *testing.T, module []byte) RetainedModule {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(path, module, 0700); err != nil {
		t.Fatal(err)
	}
	return retainedModuleFile(path, string(record.HashBytes(module)))
}

// retainedModuleFile stands in for the Artifact target, read without
// following symbolic links.
func retainedModuleFile(path, digest string) RetainedModule {
	return RetainedModule{SHA256: digest, Copy: func(destination string) (retErr error) {
		source, info, err := hostfs.OpenPath(path)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, source.Close()) }()
		if err := CheckRetainedModuleSize(uint64(info.Size())); err != nil {
			return err
		}
		return copyRetainedModule(destination, io.LimitReader(source, maximumRetainedModuleBytes+1))
	}}
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
			a := &retainedAdmission{bundle: retainedExecution{Boots: []retainedBoot{{ID: "a"}}, ModuleSHA256: moduleID}, provider: p}
			directory := t.TempDir()
			_, reopened, err := a.reopen(retainedModuleBytes(t, module), directory, test.installed)
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
		_, original, provider, err := ReplayBoundRetained(t.Context(), target.BackendPayload{}, RetainedModule{SHA256: primary}, directory, InstalledHelper{})
		if err == nil || original != "" || provider != nil {
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

// Schema v2 references the Artifact target, so replay must reject any target
// bytes that do not hash to the retained digest, before a helper can start.
func TestRetainedReplayRejectsTamperedOrMismatchedTargetModule(t *testing.T) {
	module, helper := []byte("retained-module"), []byte("installed-helper")
	moduleID := strings.TrimPrefix(string(record.HashBytes(module)), "sha256:")
	installed := installedHelperFile(t, helper)
	admit := func() *retainedAdmission {
		p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": {ModuleSHA256: moduleID, HelperSHA256: installed.SHA256, Engine: installed.Engine}}}
		return &retainedAdmission{bundle: retainedExecution{Boots: []retainedBoot{{ID: "a"}}, ModuleSHA256: moduleID}, provider: p}
	}
	tampered := retainedModuleBytes(t, []byte("tampered-module"))
	tampered.SHA256 = "sha256:" + moduleID
	foreign := retainedModuleBytes(t, []byte("foreign-module"))
	genuinePath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(genuinePath, module, 0700); err != nil {
		t.Fatal(err)
	}
	genuine := retainedModuleFile(genuinePath, "sha256:"+moduleID)
	// Both links resolve to the genuine module, so only link refusal rejects them.
	link := filepath.Join(t.TempDir(), "target")
	if err := os.Symlink(genuinePath, link); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		module RetainedModule
		want   error
	}{
		{"tampered-bytes", tampered, nil},
		{"foreign-digest", foreign, nil},
		{"missing-source", RetainedModule{SHA256: "sha256:" + moduleID}, nil},
		{"symbolic-link-source", retainedModuleFile(link, "sha256:"+moduleID), hostfs.ErrSymbolicLink},
		{"symbolic-link-copy", RetainedModule{SHA256: "sha256:" + moduleID, Copy: func(destination string) error {
			return os.Symlink(genuinePath, destination)
		}}, hostfs.ErrSymbolicLink},
		{"truncated", RetainedModule{SHA256: "sha256:" + moduleID, Copy: func(destination string) error {
			return copyRetainedModule(destination, bytes.NewReader(module[:len(module)-1]))
		}}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			if _, reopened, err := admit().reopen(test.module, directory, installed); err == nil || reopened != nil || test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("retained replay admitted target bytes that differ from the referenced digest: %v", err)
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 0 {
				t.Fatalf("rejected target module left executable inputs: %v %v", entries, err)
			}
		})
	}
	if _, reopened, err := admit().reopen(genuine, t.TempDir(), installed); err != nil || reopened == nil {
		t.Fatalf("genuine referenced target rejected: %v", err)
	}
}

func TestRetainedSchemaV1IsRefusedWithVersionedError(t *testing.T) {
	data := []byte(`{"schema":"gomad3.wasm-simulation-retained/v1","inputs":{}}`)
	payload := target.BackendPayload{Data: data, Reference: record.BackendPayload{Schema: "gomad3.wasm-simulation-retained/v1", File: "backend/simulation.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}}
	if !IsRetainedSchema(payload.Reference.Schema) || !IsRetainedSchema(RetainedSchema) || IsRetainedSchema("gomad3.other/v1") {
		t.Fatal("retained schema routing does not recognize historical cluster evidence")
	}
	directory := t.TempDir()
	module := RetainedModule{SHA256: "sha256:" + strings.Repeat("a", 64)}
	_, _, verifyErr := VerifyRetained(payload)
	_, _, _, replayErr := ReplayBoundRetained(t.Context(), payload, module, directory, InstalledHelper{})
	_, selectionErr := RetainedStorageCrashSelections(payload)
	for _, err := range []error{verifyErr, replayErr, selectionErr} {
		if !errors.Is(err, ErrRetainedSchemaUnsupported) || !strings.Contains(err.Error(), "retained/v1") || !strings.Contains(err.Error(), RetainedSchema) || !strings.Contains(err.Error(), "re-record") {
			t.Fatalf("v1 retained evidence lacks a clear versioned refusal: %v", err)
		}
	}
	if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
		t.Fatalf("refused v1 evidence wrote executable inputs: %v %v", entries, err)
	}
}
