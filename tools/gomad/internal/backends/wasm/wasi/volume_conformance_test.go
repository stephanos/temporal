//go:build test_dep

package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

var volumeConformanceOutput = flag.String("volume-conformance-output", "", "write Go reference corpus to an external directory for review")

const volumeConformanceModel = "internal/model/volume"
const volumeConformanceImageIdentity = volumeConformanceBinaryImageIdentity

type volumeConformanceSession struct {
	Model   string                    `json:"model"`
	Image   namespaceConformanceImage `json:"image"`
	Storage *StorageModel             `json:"storage"`
}

type volumeConformanceOperation struct {
	Op    string          `json:"op"`
	Input json.RawMessage `json:"input"`
	Now   uint64          `json:"now"`
}

type pathInput struct {
	FD   uint32 `json:"fd"`
	Path []byte `json:"path_base64"`
}

type pathStatInput struct {
	FD    uint32 `json:"fd"`
	Path  []byte `json:"path_base64"`
	Flags uint32 `json:"flags"`
}

type renameInput struct {
	FD      uint32 `json:"fd"`
	Path    []byte `json:"path_base64"`
	NewFD   uint32 `json:"new_fd"`
	NewPath []byte `json:"new_path_base64"`
}

type volumeConformanceOperations struct {
	Volume     string             `json:"volume"`
	Operations []volume.Operation `json:"operations"`
}

type volumeConformanceObservation struct {
	Reply           []byte                        `json:"reply_base64"`
	Events          []byte                        `json:"events_base64"`
	Snapshots       []volume.VolumeSnapshot       `json:"snapshots"`
	Operations      []volumeConformanceOperations `json:"operations"`
	Enumeration     *volume.CrashEnumeration      `json:"enumeration"`
	Selections      map[string][]uint64           `json:"selections"`
	ImageDigest     string                        `json:"image_digest"`
	ImageData       []byte                        `json:"image_base64"`
	ImageIdentity   string                        `json:"image_identity"`
	NamespaceDigest string                        `json:"namespace_digest"`
}

type volumeConformanceCorpus struct {
	Schema       string                         `json:"schema"`
	Name         string                         `json:"name"`
	Origin       string                         `json:"origin"`
	Seed         uint64                         `json:"seed"`
	Session      volumeConformanceSession       `json:"session"`
	InitialReply []byte                         `json:"initial_reply_base64"`
	Operations   []volumeConformanceOperation   `json:"operations"`
	Observations []volumeConformanceObservation `json:"observations"`
}

type volumeConformanceScript struct {
	name, origin string
	seed         uint64
	config       Config
	operations   []volumeConformanceOperation
}

func TestVolumeConformanceCorpus(t *testing.T) {
	for _, script := range volumeConformanceScripts(t) {
		t.Run(script.name, func(t *testing.T) {
			got := volumeConformanceRun(t, script)
			name := filepath.Join("testdata", "volume-conformance", script.name+".json")
			if *volumeConformanceOutput != "" {
				if !filepath.IsAbs(*volumeConformanceOutput) {
					t.Fatal("volume-conformance-output must be an absolute external directory")
				}
				data, err := json.MarshalIndent(got, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(*volumeConformanceOutput, script.name+".json"), append(data, '\n'), 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			data, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("Go volume corpus missing: %v", err)
			}
			var want volumeConformanceCorpus
			if err := json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			if difference := volumeConformanceDifference(want, got); difference != "" {
				t.Fatalf("Go volume drift: %s; inspect before updating %s", difference, name)
			}
		})
	}
}

func TestVolumeConformance(t *testing.T) {
	helper := namespaceConformanceHelper(t)
	for _, script := range volumeConformanceScripts(t) {
		t.Run(script.name, func(t *testing.T) {
			want := volumeConformanceRun(t, script)
			exchange, closeDriver := namespaceConformanceTransport(t, helper, script.name)
			defer closeDriver()
			initial := exchange(want.Session)
			if !bytes.Equal(initial, want.InitialReply) {
				t.Fatalf("seed=%d session divergence: Go=%s Rust=%s", script.seed, want.InitialReply, initial)
			}
			for step, operation := range script.operations {
				data := exchange(operation)
				var got volumeConformanceObservation
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatalf("seed=%d step=%d invalid reply: %v", script.seed, step, err)
				}
				if !reflect.DeepEqual(want.Observations[step], got) {
					t.Fatalf("seed=%d first divergence at step=%d op=%s now=%d\nGo=%s\nRust=%s", script.seed, step, operation.Op, operation.Now, volumeConformanceReply(t, want.Observations[step], nil), data)
				}
			}
		})
	}
}

func volumeConformanceDifference(want, got volumeConformanceCorpus) string {
	for step := range want.Operations {
		var compact bytes.Buffer
		if err := json.Compact(&compact, want.Operations[step].Input); err != nil {
			return fmt.Sprintf("seed=%d invalid operation input at step=%d: %v", got.Seed, step, err)
		}
		want.Operations[step].Input = compact.Bytes()
	}
	if len(want.Observations) != len(got.Observations) {
		return fmt.Sprintf("seed=%d observation count: want=%d got=%d", got.Seed, len(want.Observations), len(got.Observations))
	}
	for step := range got.Observations {
		if !reflect.DeepEqual(want.Observations[step], got.Observations[step]) {
			wantJSON, err := json.Marshal(want.Observations[step])
			if err != nil {
				return err.Error()
			}
			gotJSON, err := json.Marshal(got.Observations[step])
			if err != nil {
				return err.Error()
			}
			return fmt.Sprintf("seed=%d first divergence at step=%d op=%s now=%d\nwant=%s\ngot=%s", got.Seed, step, got.Operations[step].Op, got.Operations[step].Now, wantJSON, gotJSON)
		}
	}
	want.Observations, got.Observations = nil, nil
	if !reflect.DeepEqual(want, got) {
		return fmt.Sprintf("seed=%d session or operation records differ", got.Seed)
	}
	return ""
}

func volumeConformanceRun(t *testing.T, script volumeConformanceScript) volumeConformanceCorpus {
	t.Helper()
	corpus := volumeConformanceCorpus{Schema: "gomad3.volume-conformance/v1", Name: script.name, Origin: script.origin, Seed: script.seed, Operations: script.operations}
	initialConfig := script.config
	initialConfig.Storage = nil
	initial := newTestEnvironment(t, initialConfig)
	corpus.Session = volumeConformanceSession{volumeConformanceModel, namespaceConformanceInitialImage(initial, initialConfig), script.config.Storage}
	e, err := NewEnvironment(script.config)
	corpus.InitialReply = volumeConformanceReply(t, struct{}{}, err)
	if err != nil {
		return corpus
	}
	images := map[string]*StorageImage{}
	pages := map[string]volume.CrashEnumeration{}
	selections := map[string]map[string][]uint64{}
	imageDigest := ""
	var imageData []byte
	for step, operation := range script.operations {
		e.now = operation.Now
		before := len(e.StorageTranscript())
		observation := volumeConformanceObservation{ImageIdentity: volumeConformanceImageIdentity}
		var result any = struct{}{}
		var err error
		switch operation.Op {
		case "enumerate":
			var input struct {
				Volume string                        `json:"volume"`
				Limits volume.CrashEnumerationLimits `json:"limits"`
				Cursor string                        `json:"cursor"`
				Save   string                        `json:"save"`
			}
			volumeConformanceDecode(t, operation.Input, &input)
			var cursor *volume.CrashExploration
			if input.Cursor != "" {
				page, exists := pages[input.Cursor]
				if !exists || page.ChoiceExploration == nil {
					t.Fatalf("seed=%d step=%d missing cursor %s", script.seed, step, input.Cursor)
				}
				cursor = page.ChoiceExploration
			}
			page, enumerationErr := e.EnumerateStorageCrashStates(input.Volume, input.Limits, cursor)
			err = enumerationErr
			observation.Enumeration = &page
			if err == nil {
				pages[input.Save] = page
			}
		case "select":
			var input struct {
				Seed uint64 `json:"seed"`
				Save string `json:"save"`
			}
			volumeConformanceDecode(t, operation.Input, &input)
			observation.Selections = e.namespace.storage.fs.SelectCrashOperations(input.Seed)
			selections[input.Save] = observation.Selections
		case "crash":
			var input struct {
				Selections map[string][]uint64 `json:"selections"`
				Selection  string              `json:"selection"`
				Page       string              `json:"page"`
				State      int                 `json:"state"`
				Image      string              `json:"image"`
			}
			volumeConformanceDecode(t, operation.Input, &input)
			selected := input.Selections
			if input.Selection != "" {
				var exists bool
				selected, exists = selections[input.Selection]
				if !exists {
					t.Fatalf("step=%d missing selection %s", step, input.Selection)
				}
			}
			if input.Page != "" {
				page := pages[input.Page]
				if input.State < 0 || input.State >= len(page.States) {
					t.Fatalf("step=%d invalid page state", step)
				}
				state := page.States[input.State]
				selected = map[string][]uint64{state.Volume: state.SelectedOperations}
			}
			observation.Selections = selected
			var image *StorageImage
			image, err = e.CrashStorage(selected)
			if err == nil {
				images[input.Image] = image
				imageData, imageDigest, err = volumeConformanceEncodeImage(image, (64<<20)+script.config.Limits.FilesystemBytes)
				if err != nil {
					t.Fatal(err)
				}
			}
		case "recover":
			var input struct {
				Image string `json:"image"`
			}
			volumeConformanceDecode(t, operation.Input, &input)
			image, exists := images[input.Image]
			if !exists {
				t.Fatalf("step=%d missing image %s", step, input.Image)
			}
			var recovered *Environment
			recovered, err = NewRecoveredEnvironment(script.config, image)
			if err == nil {
				e = recovered
				e.now = operation.Now
				before = 0
				imageData, imageDigest, err = volumeConformanceEncodeImage(image, (64<<20)+script.config.Limits.FilesystemBytes)
				if err != nil {
					t.Fatal(err)
				}
			}
		default:
			var reply Reply
			reply, err = e.Handle(Call{Type: "call", ID: e.nextCall, Op: operation.Op, Input: operation.Input})
			result = struct {
				Errno       uint16          `json:"errno"`
				Output      json.RawMessage `json:"output"`
				Unsupported bool            `json:"unsupported,omitempty"`
				Message     string          `json:"message,omitempty"`
			}{reply.Errno, reply.Output, reply.Unsupported, reply.Message}
		}
		observation.Reply = volumeConformanceReply(t, result, err)
		observation.Events = slices.Clone(e.StorageTranscript()[before:])
		observation.Snapshots = e.StorageSnapshots()
		operations := e.StorageOperations()
		ids := make([]string, 0, len(operations))
		for id := range operations {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			observation.Operations = append(observation.Operations, volumeConformanceOperations{id, operations[id]})
		}
		observation.ImageDigest = imageDigest
		observation.ImageData = slices.Clone(imageData)
		observation.NamespaceDigest = namespaceConformanceDigest(e)
		corpus.Observations = append(corpus.Observations, observation)
	}
	return corpus
}

func volumeConformanceDecode(t *testing.T, input json.RawMessage, target any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func volumeConformanceReply(t *testing.T, result any, err error) []byte {
	t.Helper()
	if err != nil {
		switch value := err.(type) {
		case *BoundaryError:
			result = struct {
				Boundary string `json:"boundary"`
			}{value.Kind + ":" + value.Dimension}
		case *StorageCrash:
			result = struct {
				Crash string `json:"storage_crash"`
			}{value.Fault}
		default:
			result = struct {
				Error string `json:"error"`
			}{err.Error()}
		}
	}
	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	return data
}

func volumeConformanceOp(op string, input any) volumeConformanceOperation {
	data, err := json.Marshal(input)
	if err != nil {
		panic(err)
	}
	return volumeConformanceOperation{Op: op, Input: data}
}

func volumeConformanceOpen(name string, flags uint16, rights uint64) volumeConformanceOperation {
	return volumeConformanceOp("path_open", openInput{FD: 4, Path: []byte(name), Oflags: flags, Base: rights})
}

func volumeConformanceWrite(fd uint32, data string) volumeConformanceOperation {
	return volumeConformanceOp("fd_pwrite", struct {
		FD     uint32 `json:"fd"`
		Offset uint64 `json:"offset"`
		Data   []byte `json:"data_base64"`
	}{fd, 0, []byte(data)})
}

func volumeConformanceEnumerate(limits volume.CrashEnumerationLimits, cursor, save string) volumeConformanceOperation {
	return volumeConformanceOp("enumerate", struct {
		Volume string                        `json:"volume"`
		Limits volume.CrashEnumerationLimits `json:"limits"`
		Cursor string                        `json:"cursor"`
		Save   string                        `json:"save"`
	}{"data", limits, cursor, save})
}

func volumeConformanceScripts(t *testing.T) []volumeConformanceScript {
	t.Helper()
	limits := volume.CrashEnumerationLimits{States: 64, Operations: 64, Depth: 64, Bytes: 1 << 20, Work: 1 << 16}
	fd := func(op string, n uint32) volumeConformanceOperation { return volumeConformanceOp(op, fdInput{FD: n}) }
	read := func(n uint32) volumeConformanceOperation {
		return volumeConformanceOp("fd_pread", struct {
			FD     uint32 `json:"fd"`
			Offset uint64 `json:"offset"`
			Length uint32 `json:"length"`
		}{n, 0, 100})
	}
	crash := volumeConformanceOp("crash", map[string]any{"image": "durable", "selections": nil})
	recover := volumeConformanceOp("recover", map[string]string{"image": "durable"})
	base := []volumeConformanceOperation{volumeConformanceOpen("db", 1, rightsAll), volumeConformanceWrite(5, "committed"), fd("fd_sync", 5), fd("fd_sync", 4)}
	scripts := []volumeConformanceScript{{name: "sync-crash-reopen", origin: "wasi/volume_test.go:TestStorageFileAndDirectorySyncCrashReopen", config: storageConfig(), operations: []volumeConformanceOperation{
		base[0], base[1], base[2], crash, fd("fd_filestat_get", 5), recover,
		volumeConformanceOpen("db", 0, rightsAll), base[0], base[1], base[2], base[3], volumeConformanceWrite(5, "volatile!"),
		volumeConformanceEnumerate(limits, "", "all"), crash, recover, volumeConformanceOpen("db", 0, rightsAll), read(5), recover, volumeConformanceOpen("db", 0, rightsAll), read(5),
	}}}
	for _, action := range []string{"io-error", "sync-error", "no-space", "short-write", "torn-write", "crash-before"} {
		config := storageConfig()
		operation := "write"
		if action == "sync-error" || action == "crash-before" {
			operation = "sync"
		}
		fault := StorageFault{Name: "named-" + action, Resource: "/workspace/db", Operation: operation, Occurrence: 2, Action: action}
		if action == "short-write" || action == "torn-write" {
			fault.Bytes = 2
		}
		config.Storage.Faults = []StorageFault{fault}
		ops := []volumeConformanceOperation{volumeConformanceOpen("other", 1, rightsAll), volumeConformanceOpen("db", 1, rightsAll), volumeConformanceWrite(5, "unrelated"), volumeConformanceWrite(6, "initial")}
		if operation == "sync" {
			ops = append(ops, fd("fd_sync", 6), fd("fd_sync", 6))
		} else {
			ops = append(ops, volumeConformanceWrite(6, "changed"))
		}
		ops = append(ops, read(6), volumeConformanceEnumerate(limits, "", "all"), crash, recover)
		scripts = append(scripts, volumeConformanceScript{name: "fault-" + action, origin: "wasi/volume_test.go:TestStorageFaultResourceOccurrenceReplayAndTypedOutcomes and TestStorageCrashFaultAndModelLimits", config: config, operations: ops})
	}
	for _, action := range []string{"short-write", "torn-write"} {
		config := storageConfig()
		config.Storage.Faults = []StorageFault{{Name: "nonshortening", Resource: "/workspace/db", Operation: "write", Occurrence: 1, Action: action, Bytes: 2}}
		scripts = append(scripts, volumeConformanceScript{name: "invalid-" + action, origin: "wasi/volume.go:storageDecision runtime refusal", config: config, operations: []volumeConformanceOperation{base[0], volumeConformanceWrite(5, "xx"), read(5)}})
	}
	invalidConfig := storageConfig()
	invalidConfig.Storage.Faults = []StorageFault{{Name: "rejected", Resource: "/workspace/db", Operation: "write", Occurrence: 1, Action: "crash-after"}}
	scripts = append(scripts, volumeConformanceScript{name: "invalid-crash-after", origin: "wasi/volume.go:validateStorageModel rejected action", config: invalidConfig})
	tiny := limits
	tiny.Work = 1
	workOps := []volumeConformanceOperation{base[0], volumeConformanceWrite(5, "value"), volumeConformanceEnumerate(limits, "", "all"), volumeConformanceEnumerate(tiny, "", "first")}
	for i := 0; i < 15; i++ {
		cursor := "first"
		if i > 0 {
			cursor = fmt.Sprintf("page-%d", i-1)
		}
		workOps = append(workOps, volumeConformanceEnumerate(tiny, cursor, fmt.Sprintf("page-%d", i)))
	}
	workOps = append(workOps, volumeConformanceWrite(5, "changed"), volumeConformanceEnumerate(limits, "first", "stale"))
	scripts = append(scripts, volumeConformanceScript{name: "work-resume-stale", origin: "internal/model/volume/work_test.go:TestTinyWorkResumesWithExactlyTheUnrestrictedStatesAndOrder", config: storageConfig(), operations: workOps})
	for _, capacity := range []string{"states", "operations", "depth", "bytes", "work", "zero-work"} {
		bounded := limits
		switch capacity {
		case "states":
			bounded.States = 1
		case "operations":
			bounded.Operations = 1
		case "depth":
			bounded.Depth = 1
		case "bytes":
			bounded.Bytes = 1
		case "work":
			bounded.Work = 1
		case "zero-work":
			bounded.Work = 0
		}
		scripts = append(scripts, volumeConformanceScript{name: "enumeration-" + capacity, origin: "internal/model/volume/volume_test.go crash enumeration limits", config: storageConfig(), operations: []volumeConformanceOperation{base[0], base[1], volumeConformanceEnumerate(bounded, "", "bounded")}})
	}
	for _, adapted := range []bool{false, true} {
		config := storageConfig()
		if adapted {
			config.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
		}
		ops := []volumeConformanceOperation{volumeConformanceOpen("db.lock", 1|4, rightsAll), volumeConformanceOp("path_create_directory", pathInput{FD: 4, Path: []byte("db-shm")})}
		for i, name := range []string{"db", "db-wal", "db-journal", "other.lock"} {
			n := uint32(6 + i)
			ops = append(ops, volumeConformanceOpen(name, 1, rightsAll), volumeConformanceWrite(n, "durable"), fd("fd_sync", n))
		}
		ops = append(ops, fd("fd_sync", 4), crash, recover)
		for _, name := range []string{"db.lock", "db-shm", "db", "db-wal", "db-journal", "other.lock"} {
			ops = append(ops, volumeConformanceOp("path_filestat_get", pathStatInput{FD: 4, Path: []byte(name)}))
		}
		scripts = append(scripts, volumeConformanceScript{name: fmt.Sprintf("sqlite-lock-opt-in-%v", adapted), origin: "wasi/volume_test.go:TestStorageSQLiteLockAdapterIsExactOptIn", config: config, operations: ops})
	}
	ops := []volumeConformanceOperation{volumeConformanceOp("path_create_directory", pathInput{FD: 4, Path: []byte("dir")}), volumeConformanceOpen("dir/db", 1, rightsAll), volumeConformanceWrite(5, "abcdef"), volumeConformanceOp("fd_filestat_set_size", struct {
		FD   uint32 `json:"fd"`
		Size uint64 `json:"size"`
	}{5, 3}), read(5), fd("fd_close", 5), volumeConformanceOpen("dir/db", 0, rightRead), fd("fd_sync", 6), volumeConformanceOp("path_unlink_file", pathInput{FD: 4, Path: []byte("dir/db")}), volumeConformanceOp("path_remove_directory", pathInput{FD: 4, Path: []byte("dir")}), fd("fd_sync", 4), volumeConformanceEnumerate(limits, "", "all"), volumeConformanceOp("crash", map[string]any{"page": "all", "state": 0, "image": "durable"}), recover}
	scripts = append(scripts, volumeConformanceScript{name: "filesystem-operations-rights", origin: "wasi/volume_test.go:TestStorageEnumerationAndCapabilityRights and namespace filesystem operations", config: storageConfig(), operations: ops})
	for _, capacity := range []string{"volume-bytes", "pending-operations", "transitions", "filesystem-bytes", "storage-transcript"} {
		config := storageConfig()
		switch capacity {
		case "volume-bytes":
			config.Storage.Volumes[0].CapacityBytes = 2
		case "pending-operations":
			config.Storage.Limits.PendingOperations = 1
		case "transitions":
			config.Storage.Limits.Transitions = 1
		case "filesystem-bytes":
			config.Storage.Volumes[0].CapacityBytes = config.Limits.FilesystemBytes
			config.Limits.FilesystemBytes = 3
			config.Storage.Volumes[0].CapacityBytes = 3
		case "storage-transcript":
			config.Limits.TranscriptBytes = 160
		}
		ops := []volumeConformanceOperation{base[0], volumeConformanceWrite(5, "1234"), read(5), volumeConformanceEnumerate(limits, "", "all")}
		if capacity == "filesystem-bytes" {
			ops = append([]volumeConformanceOperation{volumeConformanceOp("path_open", openInput{FD: 3, Path: []byte("volatile"), Oflags: 1, Base: rightsAll}), volumeConformanceWrite(5, "12"), base[0], volumeConformanceWrite(6, "12")}, read(6))
		}
		scripts = append(scripts, volumeConformanceScript{name: "capacity-" + capacity, origin: "wasi/volume_test.go:TestStorageCrashFaultAndModelLimits and storageBoundary classification", config: config, operations: ops})
	}
	multi := storageConfig()
	multi.WritableDirectories = append(multi.WritableDirectories, "/secondary")
	multi.Storage.Volumes = append([]volume.VolumeConfig{{ID: "z-secondary", Path: "/secondary", CapacityBytes: 1 << 20}}, multi.Storage.Volumes...)
	scripts = append(scripts, volumeConformanceScript{name: "sorted-volumes-invalid-selection", origin: "internal/model/volume/volume_test.go: multi-volume selection and sorted observations", config: multi, operations: []volumeConformanceOperation{volumeConformanceOp("path_open", openInput{FD: 5, Path: []byte("db"), Oflags: 1, Base: rightsAll}), volumeConformanceWrite(6, "committed"), volumeConformanceOp("select", map[string]any{"seed": 7, "save": "seeded"}), volumeConformanceOp("crash", map[string]any{"selections": map[string][]uint64{"data": {999}}, "image": "bad"}), volumeConformanceEnumerate(limits, "", "all"), volumeConformanceOp("crash", map[string]any{"selection": "seeded", "image": "durable"}), recover}})
	config := storageConfig()
	config.Storage.Faults = []StorageFault{{Name: "after-recovery", Resource: "/workspace/db", Operation: "write", Occurrence: 2, Action: "io-error"}}
	ops = append(slices.Clone(base), crash, recover, volumeConformanceOpen("db", 0, rightsAll), volumeConformanceWrite(5, "changed"), read(5), crash, recover, volumeConformanceOpen("db", 0, rightsAll), volumeConformanceWrite(5, "continued"), read(5))
	scripts = append(scripts, volumeConformanceScript{name: "recover-occurrence-continuation", origin: "wasi/volume.go:NewRecoveredEnvironment preserved occurrence counters", config: config, operations: ops})
	for seed := uint64(0); seed < 16; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed^0x766f6c756d65))
		config := storageConfig()
		faultOperation := []string{"write", "sync", "rename", "remove"}[rng.IntN(4)]
		resource := "/workspace/db"
		occurrence := 1 + uint64(rng.IntN(2))
		if faultOperation == "rename" {
			occurrence = 1
		}
		if faultOperation == "remove" {
			resource = "/workspace/renamed"
			occurrence = 1
		}
		config.Storage.Faults = []StorageFault{{Name: "generated", Resource: resource, Operation: faultOperation, Occurrence: occurrence, Action: "io-error"}}
		ops := slices.Clone(base)
		ops = append(ops, volumeConformanceWrite(5, fmt.Sprintf("seed-%d", seed)), fd("fd_sync", 5), volumeConformanceOp("path_rename", renameInput{FD: 4, Path: []byte("db"), NewFD: 4, NewPath: []byte("renamed")}), volumeConformanceOp("path_unlink_file", pathInput{FD: 4, Path: []byte("renamed")}), volumeConformanceEnumerate(limits, "", "all"), volumeConformanceOp("select", map[string]any{"seed": seed, "save": "seeded"}), volumeConformanceOp("crash", map[string]any{"selection": "seeded", "image": "durable"}), recover, volumeConformanceOpen("continued", 1, rightsAll), volumeConformanceWrite(5, "continue"), fd("fd_sync", 5), fd("fd_sync", 4), volumeConformanceEnumerate(limits, "", "continued"))
		scripts = append(scripts, volumeConformanceScript{name: fmt.Sprintf("generated-%02d", seed), origin: "seeded PCG write/sync/rename/unlink, enumerate/select/crash/recover/continue", seed: seed, config: config, operations: ops})
	}
	for i := range scripts {
		for j := range scripts[i].operations {
			scripts[i].operations[j].Now = 100 + uint64(j)*17
		}
	}
	return append(scripts, volumeConformanceRecordedScript(t))
}

func volumeConformanceRecordedScript(t *testing.T) volumeConformanceScript {
	t.Helper()
	data, err := os.ReadFile("testdata/volume-conformance/recorded-storage-sqlite/sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var sample struct {
		Producer   []byte `json:"producer_base64"`
		Manifest   []byte `json:"manifest_base64"`
		Provenance []byte `json:"provenance_base64"`
		Simulation []byte `json:"simulation_base64"`
	}
	volumeConformanceDecode(t, data, &sample)
	for _, input := range []struct {
		data []byte
		hash string
	}{
		{sample.Producer, "017d5d9a84f3c37d3b6f93fd1570b388d9d63120c5f4a23d17f18f763f712b66"},
		{sample.Manifest, "bf4cefdefcf4166bded95962bcda09cb473c7259a58bee28f2bbbb35ccf3d344"},
		{sample.Provenance, "dd783f52ca63f5cd05ff02af0789cf8b682c369256b504596c15e6125e69aefb"},
		{sample.Simulation, "d02db41991a0640606bc53578f4958e71d95a46c001437920ee5f7e9d075997b"},
	} {
		if fmt.Sprintf("%x", sha256.Sum256(input.data)) != input.hash {
			t.Fatal("recorded storage_sqlite sample authentication mismatch")
		}
	}
	var provenance struct {
		Source       string
		ModuleSHA256 string
		Config       Config
	}
	if err := json.Unmarshal(sample.Provenance, &provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.Source != "./tools/gomad/internal/backends/wasm/testdata/storage_sqlite/main.go" || provenance.ModuleSHA256 != "sha256:5b6eade386726065e5e57fd2b67c399104a084563ea1b88c1fc1192902ca7bc3" {
		t.Fatal("recorded sample guest identity mismatch")
	}
	return volumeConformanceScript{name: "recorded-storage-sqlite-prefix", origin: "authenticated storage_sqlite recorded volume transitions 0..3, projected to two creates; flags/rights/now=1 are derived inputs (see recorded-storage-sqlite/README.md)", seed: 7, config: provenance.Config, operations: []volumeConformanceOperation{
		{Op: "path_open", Input: volumeConformanceOpen("db", 1, rightsAll).Input, Now: 1},
		{Op: "path_open", Input: volumeConformanceOpen("db-journal", 1, rightsAll).Input, Now: 1},
	}}
}

func TestVolumeConformanceRecordedStorageOperations(t *testing.T) {
	script := volumeConformanceRecordedScript(t)
	got := volumeConformanceRun(t, script)
	operations := got.Observations[1].Operations[0].Operations
	want := []volume.Operation{
		{ID: 1, Kind: "alloc", Inode: 4, EffectSHA256: "sha256:1b2057a9e9c5a90413199dea178939f401d5c6dfdb5e7ac9acb15600a8fa88a8"},
		{ID: 2, Kind: "namespace", Dependencies: []uint64{1}, Inode: 4, Path: "/db", EffectSHA256: "sha256:13e3c6c0e31f88669fec3c274465273a52986f8dd03697a7bd69166a4cced161"},
		{ID: 3, Kind: "alloc", Inode: 5, EffectSHA256: "sha256:1bc7e6757991d99abd99680dcdf3e145d6a8b732749f199560fcc2f469da8413"},
		{ID: 4, Kind: "namespace", Dependencies: []uint64{3}, Inode: 5, Path: "/db-journal", EffectSHA256: "sha256:1ef714983098e0abee6a1ac765141075b7fecaa28a4d46be8fbeda65dc4121ac"},
	}
	if !reflect.DeepEqual(want, operations) {
		t.Fatalf("recorded storage operation prefix differs:\nwant=%+v\ngot=%+v", want, operations)
	}
}

func TestVolumeConformanceReferenceContracts(t *testing.T) {
	for _, script := range volumeConformanceScripts(t) {
		t.Run(script.name, func(t *testing.T) {
			got := volumeConformanceRun(t, script)
			switch script.name {
			case "invalid-crash-after":
				if string(got.InitialReply) != `{"boundary":"invalid:storage fault action"}` {
					t.Fatalf("crash-after admission = %s", got.InitialReply)
				}
			case "invalid-short-write", "invalid-torn-write":
				step := got.Observations[1]
				if string(step.Reply) != `{"boundary":"invalid:storage write fault must shorten the matched write"}` || len(step.Events) != 0 {
					t.Fatalf("invalid write mutated event stream: %+v", step)
				}
			case "fault-short-write":
				if string(got.Observations[4].Reply) != `{"errno":0,"output":{"written":2}}` {
					t.Fatalf("short write reply=%s", got.Observations[4].Reply)
				}
			case "fault-io-error", "fault-torn-write":
				if string(got.Observations[4].Reply) != `{"errno":29,"output":{}}` {
					t.Fatalf("error reply=%s", got.Observations[4].Reply)
				}
			case "fault-no-space":
				if string(got.Observations[4].Reply) != `{"errno":51,"output":{}}` {
					t.Fatalf("no-space reply=%s", got.Observations[4].Reply)
				}
			case "fault-sync-error":
				if string(got.Observations[5].Reply) != `{"errno":29,"output":{}}` {
					t.Fatalf("sync error reply=%s", got.Observations[5].Reply)
				}
			case "fault-crash-before":
				if string(got.Observations[5].Reply) != `{"storage_crash":"named-crash-before"}` {
					t.Fatalf("crash reply=%s", got.Observations[5].Reply)
				}
			case "work-resume-stale":
				var resumed []volume.CrashState
				for _, observation := range got.Observations[3:19] {
					resumed = append(resumed, observation.Enumeration.States...)
				}
				if !got.Observations[18].Enumeration.Complete || !reflect.DeepEqual(resumed, got.Observations[2].Enumeration.States) {
					t.Fatal("tiny Work lost or reordered states")
				}
				if string(got.Observations[20].Reply) == "{}" {
					t.Fatal("stale cursor was accepted")
				}
			case "recover-occurrence-continuation":
				if !bytes.Contains(got.Observations[7].Events, []byte(`"occurrence":2,"fault":"after-recovery"`)) || !bytes.Contains(got.Observations[12].Events, []byte(`"occurrence":3,"action":"complete"`)) {
					t.Fatal("recovery reset fault occurrences")
				}
			case "capacity-filesystem-bytes":
				if string(got.Observations[3].Reply) != `{"boundary":"capacity:filesystem-bytes"}` {
					t.Fatalf("namespace byte capacity=%s", got.Observations[3].Reply)
				}
			case "sorted-volumes-invalid-selection":
				if string(got.InitialReply) != "{}" || len(got.Observations[0].Operations) != 2 || got.Observations[0].Operations[0].Volume != "data" || got.Observations[0].Operations[1].Volume != "z-secondary" {
					t.Fatal("multi-volume script did not exercise sorted operation observations")
				}
			}
			if strings.HasPrefix(script.name, "generated-") {
				var events []byte
				for _, observation := range got.Observations {
					events = append(events, observation.Events...)
				}
				if !bytes.Contains(events, []byte(`"fault":"generated"`)) {
					t.Fatalf("seed=%d never realized its scheduled fault", script.seed)
				}
			}
		})
	}
}
