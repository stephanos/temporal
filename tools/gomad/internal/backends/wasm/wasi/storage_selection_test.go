package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestStorageCrashContextAdmission(t *testing.T) {
	context := StorageCrashContext{PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: []volume.Operation{{ID: 1}, {ID: 2, Dependencies: []uint64{1}}}}
	for _, test := range []struct {
		name     string
		selected []uint64
		edit     func(*Request)
		want     string
	}{
		{"future-operations", []uint64{1, 2}, nil, ""},
		{"missing-context", []uint64{1}, func(r *Request) { r.StorageCrashContext = nil }, "prior observation"},
		{"unknown-operation", []uint64{3}, nil, "unknown operation"},
		{"dependency-closure", []uint64{2}, nil, "dependency-closed"},
		{"forged-hash", nil, func(r *Request) {
			c := r.StorageCrashContext["data"]
			c.PendingSHA256 = "a"
			r.StorageCrashContext["data"] = c
		}, "inventory"},
		{"repeated-inventory", nil, func(r *Request) {
			c := r.StorageCrashContext["data"]
			c.Operations = []volume.Operation{{ID: 1}, {ID: 1}}
			r.StorageCrashContext["data"] = c
		}, "operation order"},
		{"invented-dependency", []uint64{1}, func(r *Request) {
			c := r.StorageCrashContext["data"]
			c.Operations = []volume.Operation{{ID: 1, Dependencies: []uint64{2}}}
			r.StorageCrashContext["data"] = c
		}, "dependency"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := Request{Config: storageConfig(), StorageCrashSelection: map[string][]uint64{"data": test.selected}, StorageCrashContext: map[string]StorageCrashContext{"data": context}}
			if test.edit != nil {
				test.edit(&request)
			}
			err := ValidateStorageCrashSelection(request)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("admission = %v, want %q", err, test.want)
			}
		})
	}
}

func TestClusterBootAdmitsStorageCrashSelectionBeforeSealing(t *testing.T) {
	request := Request{Config: storageConfig(), StorageCrashSelection: map[string][]uint64{"data": {1}}}
	if _, err := prepareClusterBoot("a", request); err == nil || !strings.Contains(err.Error(), "prior observation") {
		t.Fatalf("cluster boot admission = %v", err)
	}
}

func TestStorageCrashContextLargeChunkTransport(t *testing.T) {
	config := storageConfig()
	config.Storage.Limits.PendingOperations = 5000
	context := StorageCrashContext{PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: make([]volume.Operation, 5000)}
	for index := range context.Operations {
		context.Operations[index] = volume.Operation{ID: uint64(index + 1), Kind: "write", Path: strings.Repeat("x", 256)}
	}
	request := Request{Config: config, StorageCrashContext: map[string]StorageCrashContext{"data": context}, StorageCrashSelection: map[string][]uint64{"data": {1}}}
	if err := ValidateStorageCrashSelection(request); err != nil {
		t.Fatal(err)
	}
	environment := newTestEnvironment(t, config)
	descriptor, _, input, err := helperStorageConfig(environment, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(input) <= 1<<20 || descriptor.CrashContext == nil || descriptor.CrashContext.Chunks < 17 {
		t.Fatalf("large context not chunked: bytes=%d descriptor=%+v", len(input), descriptor)
	}
	encoded, err := json.Marshal(descriptor)
	if err != nil || len(encoded) > 8192 {
		t.Fatalf("context leaked into execute frame: %d %v", len(encoded), err)
	}
	var output bytes.Buffer
	sink := frameSink{input: &output}
	if err := sink.sendStorageInput("storage_context", input); err != nil {
		t.Fatal(err)
	}
	var recovered []byte
	decoder := json.NewDecoder(&output)
	for sequence := uint64(0); sequence < descriptor.CrashContext.Chunks; sequence++ {
		var frame struct {
			Type     string `json:"type"`
			Sequence uint64 `json:"sequence"`
			Data     []byte `json:"data_base64"`
		}
		if err := decoder.Decode(&frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type != "storage_context_input" || frame.Sequence != sequence || len(frame.Data) > namespaceChunkBytes {
			t.Fatalf("invalid chunk: %+v", frame)
		}
		recovered = append(recovered, frame.Data...)
	}
	var end struct {
		Type string `json:"type"`
	}
	if err := decoder.Decode(&end); err != nil || end.Type != "storage_context_inputs_end" {
		t.Fatalf("end: %+v %v", end, err)
	}
	digest := sha256.Sum256(recovered)
	if !bytes.Equal(input, recovered) || hex.EncodeToString(digest[:]) != descriptor.CrashContext.SHA256 {
		t.Fatal("chunked context changed")
	}
	var decoded map[string]StorageCrashContext
	if err := json.Unmarshal(recovered, &decoded); err != nil {
		t.Fatal(err)
	}
	request.StorageCrashContext = decoded
	if err := ValidateStorageCrashSelection(request); err != nil {
		t.Fatal(err)
	}
}

func TestStorageCrashContextEvidenceRefusal(t *testing.T) {
	context := StorageCrashContext{PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: []volume.Operation{{ID: 1}}}
	for _, test := range []struct {
		name      string
		edit      func(*StorageCrashContext)
		wantError bool
	}{
		{"valid", nil, false},
		{"snapshot-hash", func(c *StorageCrashContext) { c.PendingSHA256 = "sha256:" + strings.Repeat("b", 64) }, true},
		{"operation-history", func(c *StorageCrashContext) { c.Operations = []volume.Operation{{ID: 1, Kind: "forged"}} }, true},
		{"missing-inventory", func(c *StorageCrashContext) { c.Operations = nil }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := context
			if test.edit != nil {
				test.edit(&value)
			}
			sections := validStorageSections()
			var err error
			sections[0], err = json.Marshal([]volume.VolumeSnapshot{{ID: "data", PendingSHA256: context.PendingSHA256, PendingOperations: 1}})
			if err != nil {
				t.Fatal(err)
			}
			sections[1], err = json.Marshal(map[string][]volume.Operation{"data": context.Operations})
			if err != nil {
				t.Fatal(err)
			}
			sections[2] = []byte(`{"data":{"Complete":true}}`)
			encoded, err := json.Marshal(map[string]StorageCrashContext{"data": value})
			if err != nil {
				t.Fatal(err)
			}
			sink := frameSink{request: Request{Config: storageConfig(), StorageCrashLimits: &volume.CrashEnumerationLimits{}}, terminal: &resultFrame{Termination: "exit"}}
			sink.helperEvidence.storageState = storageStateEnvelope(3, 6, append(sections, encoded)...)
			var result Result
			err = sink.collectHelperStorage(&result)
			if (err != nil) != test.wantError {
				t.Fatalf("context evidence=%v wantError=%v", err, test.wantError)
			}
		})
	}
	// Complete evidence carries the context section exactly when crash limits were requested.
	pending := "sha256:" + strings.Repeat("a", 64)
	snapshots := []byte(`[{"ID":"data","Mount":"/workspace","CapacityBytes":1024,"Persisted":[],"Volatile":[],"PendingOperations":0,"PendingSHA256":"` + pending + `","NextOperation":1,"Identity":""}]`)
	contextSection := []byte(`{"data":{"PendingSHA256":"` + pending + `","Operations":[]}}`)
	sections := validStorageSections()
	for _, test := range []struct {
		name     string
		limits   *volume.CrashEnumerationLimits
		envelope []byte
	}{
		{"v1-with-limits", &volume.CrashEnumerationLimits{}, storageStateEnvelope(1, 5, snapshots, sections[1], []byte(`{"data":{"Complete":true}}`), sections[3], sections[4])},
		{"v3-without-limits", nil, storageStateEnvelope(3, 6, snapshots, sections[1], []byte(`null`), sections[3], sections[4], contextSection)},
	} {
		t.Run(test.name, func(t *testing.T) {
			sink := frameSink{request: Request{Config: storageConfig(), StorageCrashLimits: test.limits}, terminal: &resultFrame{Termination: "exit"}}
			sink.helperEvidence.storageState = test.envelope
			var result Result
			if err := sink.collectHelperStorage(&result); err == nil || err.Error() != "WASI invalid: storage evidence: crash context" {
				t.Fatalf("context presence = %v, want crash context refusal", err)
			}
		})
	}
}

func TestStorageCrashContextSelectedOwnersAndDivergence(t *testing.T) {
	request := storageEvidenceRequest(t, `i32.const 4 i32.const 0 i32.const 2 call $mkdir if unreachable end`, `(data (i32.const 0) "db")`, 1)
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 16, Operations: 8, Depth: 8, Bytes: 1 << 20, Work: 256}
	var observed [2]Result
	for owner := range observed {
		request.remoteModel = owner == 0
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "exit" || len(result.StorageCrashContext["data"].Operations) == 0 {
			t.Fatalf("owner %d observation: %s %s %v", owner, result.Termination, result.Message, err)
		}
		observed[owner] = result
	}
	if difference := sameStorageEvidence(observed[0], observed[1]); difference != "" {
		t.Fatalf("owner observation differs in %s", difference)
	}
	states := observed[1].StorageCrashEnumeration["data"]
	if !states.Complete || len(states.States) < 2 {
		t.Fatalf("nontrivial enumeration required: %+v", states)
	}
	for index, state := range states.States {
		var selected [2]Result
		for owner := range selected {
			request.remoteModel = owner == 0
			request.StorageCrashSelection = map[string][]uint64{"data": state.SelectedOperations}
			request.StorageCrashContext = observed[owner].StorageCrashContext
			result, err := Run(t.Context(), request)
			if err != nil || result.Termination != "exit" {
				t.Fatalf("state %d owner %d: %s %s %v", index, owner, result.Termination, result.Message, err)
			}
			selected[owner] = result
		}
		if difference := sameStorageEvidence(selected[0], selected[1]); difference != "" {
			t.Fatalf("selected state %d differs in %s", index, difference)
		}
	}
	for owner := range observed {
		request.remoteModel = owner == 0
		context := observed[owner].StorageCrashContext["data"]
		context.PendingSHA256 = "sha256:" + strings.Repeat("0", 64)
		request.StorageCrashContext = map[string]StorageCrashContext{"data": context}
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "invalid" || result.Message != "storage crash context diverged from prior observation" || result.PeakMemoryBytes == 0 || result.StorageImage != nil || !reflect.DeepEqual(result.StorageCrashEnumeration, observed[owner].StorageCrashEnumeration) {
			t.Fatalf("owner %d divergence: %s %s image=%v peak=%d err=%v", owner, result.Termination, result.Message, result.StorageImage, result.PeakMemoryBytes, err)
		}
	}
}

func TestStorageCrashContextLargeObservedSelection(t *testing.T) {
	var data strings.Builder
	for index := range 50 {
		fmt.Fprintf(&data, `(data (i32.const %d) "%s%02d")`, 4096+4096*index, strings.Repeat(`\01`, 3998), index)
	}
	body := `loop $files
 i32.const 4 i32.const 0 i32.const 4096 local.get $i i32.const 4096 i32.mul i32.add i32.const 4000 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const 50 i32.lt_u br_if $files
 end`
	request := storageEvidenceRequest(t, body, data.String(), 4)
	request.Config.Limits.TranscriptBytes = 8 << 20
	request.Config.Limits.Descriptors = 128
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 1, Operations: 1, Depth: 1, Bytes: 1 << 20, Work: 1}
	var observed, selected [2]Result
	for owner := range observed {
		request.remoteModel = owner == 0
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "exit" {
			t.Fatalf("owner %d observation: %s %s %v", owner, result.Termination, result.Message, err)
		}
		encoded, err := json.Marshal(result.StorageCrashContext)
		if err != nil || len(encoded) <= 1<<20 {
			t.Fatalf("owner %d context bytes=%d err=%v", owner, len(encoded), err)
		}
		observed[owner] = result
	}
	if difference := sameStorageEvidence(observed[0], observed[1]); difference != "" {
		t.Fatalf("large observation differs in %s", difference)
	}
	for owner := range selected {
		request.remoteModel = owner == 0
		request.StorageCrashContext = observed[owner].StorageCrashContext
		var ids []uint64
		for _, operation := range request.StorageCrashContext["data"].Operations {
			ids = append(ids, operation.ID)
		}
		request.StorageCrashSelection = map[string][]uint64{"data": ids}
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "exit" || result.StorageImage == nil {
			t.Fatalf("owner %d large selection: %s %s %v", owner, result.Termination, result.Message, err)
		}
		selected[owner] = result
	}
	if difference := sameStorageEvidence(selected[0], selected[1]); difference != "" {
		t.Fatalf("large selected state differs in %s", difference)
	}
}

func TestStorageCrashContextEqualIsExact(t *testing.T) {
	// storageOperationEqual lists every field explicitly; a new field must be added there.
	if fields := reflect.TypeFor[volume.Operation]().NumField(); fields != 10 {
		t.Fatalf("volume.Operation has %d fields; update storageOperationEqual", fields)
	}
	base := StorageCrashContext{PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: []volume.Operation{{ID: 2, Kind: "write", Dependencies: []uint64{1}}}}
	if !base.Equal(base.Clone()) {
		t.Fatal("clone is not equal")
	}
	for name, edit := range map[string]func(*StorageCrashContext){
		"hash":             func(c *StorageCrashContext) { c.PendingSHA256 = "sha256:" + strings.Repeat("b", 64) },
		"nil-operations":   func(c *StorageCrashContext) { c.Operations = nil },
		"empty-operations": func(c *StorageCrashContext) { c.Operations = []volume.Operation{} },
		"field":            func(c *StorageCrashContext) { c.Operations[0].EffectSHA256 = "sha256:x" },
		"nil-dependencies": func(c *StorageCrashContext) { c.Operations[0].Dependencies = nil },
		"empty-dependencies": func(c *StorageCrashContext) {
			c.Operations[0].Dependencies = []uint64{}
		},
		"dependency": func(c *StorageCrashContext) { c.Operations[0].Dependencies = []uint64{3} },
	} {
		changed := base.Clone()
		edit(&changed)
		if base.Equal(changed) || changed.Equal(base) {
			t.Errorf("%s: unequal contexts compared equal", name)
		}
	}
	empty := StorageCrashContext{Operations: []volume.Operation{{ID: 1, Dependencies: []uint64{}}}}
	if empty.Equal(StorageCrashContext{Operations: []volume.Operation{{ID: 1}}}) {
		t.Error("empty and nil dependencies compared equal")
	}
	if !storageCrashContextMatches(map[string]StorageCrashContext{"data": base}, map[string]StorageCrashContext{"data": base.Clone()}) ||
		storageCrashContextMatches(map[string]StorageCrashContext{"data": base}, map[string]StorageCrashContext{}) {
		t.Error("context map matching")
	}
}
