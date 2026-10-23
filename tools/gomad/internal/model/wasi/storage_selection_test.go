package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestStorageCrashValidationVolumeOrder(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"selection", "context"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			for iteration := range 64 {
				request := Request{Config: storageConfig()}
				request.Config.Storage.Volumes = []volume.VolumeConfig{{ID: "a"}, {ID: "b"}}
				ids := []string{"a", "b"}
				if iteration%2 != 0 {
					ids = []string{"b", "a"}
				}
				want := "WASI invalid: storage crash selection zero operation"
				if kind == "selection" {
					request.StorageCrashSelection = make(map[string][]uint64)
					for _, id := range ids {
						if id == "a" {
							request.StorageCrashSelection[id] = []uint64{0}
						} else {
							request.StorageCrashSelection[id] = []uint64{1, 1}
						}
					}
				} else {
					want = "WASI invalid: storage crash context inventory"
					request.StorageCrashContext = make(map[string]StorageCrashContext)
					for _, id := range ids {
						context := StorageCrashContext{PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: []volume.Operation{{ID: 0}}}
						if id == "a" {
							context.PendingSHA256 = "invalid"
						}
						request.StorageCrashContext[id] = context
					}
				}
				err := ValidateStorageCrashSelection(request)
				if err == nil || err.Error() != want {
					t.Fatalf("iteration %d validation = %v, want %q", iteration, err, want)
				}
			}
		})
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
