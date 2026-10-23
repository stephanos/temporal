package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

type storageOpenError struct{}

func (storageOpenError) Error() string   { panic("storage error formatter dispatched arbitrary error") }
func (storageOpenError) Is(_ error) bool { panic("storage error mapper dispatched arbitrary Is") }
func (storageOpenError) As(any) bool     { panic("storage error mapper dispatched arbitrary As") }
func (storageOpenError) Unwrap() error   { panic("storage error mapper dispatched arbitrary Unwrap") }

func TestStorageErrorMappingIsClosed(t *testing.T) {
	if got := storageErrno(storageOpenError{}); got != 29 {
		t.Fatalf("closed error conversion got %d want 29", got)
	}
	if storageBoundary(storageOpenError{}) != nil {
		t.Fatal("unknown errors must not become modeled capacity")
	}
}

func storageConfig() Config {
	c := testConfig()
	c.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 1 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1000, Transitions: 1000}}
	return c
}

func writeStorage(t *testing.T, e *Environment, fd uint32, offset uint64, data string) Reply {
	t.Helper()
	return call(t, e, "fd_pwrite", fmt.Sprintf(`{"fd":%d,"offset":%d,"data_base64":%q}`, fd, offset, base64.StdEncoding.EncodeToString([]byte(data))))
}
func readStorage(t *testing.T, e *Environment, fd uint32) string {
	t.Helper()
	var read bytesOutput
	output(t, call(t, e, "fd_pread", fmt.Sprintf(`{"fd":%d,"offset":0,"length":100}`, fd)), &read)
	return string(read.Data)
}

var volumeConformanceOutput = flag.String("volume-conformance-output", "", "write Go reference corpus to an external directory for review")

const volumeConformanceModel = "internal/model/volume"

// volumeConformanceOverCapacityState selects the enumerated state that keeps
// b's write but not a's truncate.
const volumeConformanceOverCapacityState = 3
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
	ImportTranscript []byte                         `json:"import_transcript_base64,omitempty"`
	Schema           string                         `json:"schema"`
	Name             string                         `json:"name"`
	Origin           string                         `json:"origin"`
	Seed             uint64                         `json:"seed"`
	Session          volumeConformanceSession       `json:"session"`
	InitialReply     []byte                         `json:"initial_reply_base64"`
	Operations       []volumeConformanceOperation   `json:"operations"`
	Observations     []volumeConformanceObservation `json:"observations"`
}

type volumeConformanceScript struct {
	name, origin string
	seed         uint64
	config       Config
	operations   []volumeConformanceOperation
	observe      func(*Environment, int)
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
			if len(want.ImportTranscript) != 0 {
				replayed := volumeConformanceRun(t, script)
				if difference := volumeConformanceDifference(want, replayed); difference != "" {
					t.Fatalf("Go volume second-attempt drift: %s; inspect before updating %s", difference, name)
				}
			}
		})
	}
}

func TestVolumeConformanceInvalidRecoveryImages(t *testing.T) {
	config := storageConfig()
	config.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
	source := newTestEnvironment(t, storageConfig())
	fd := openPath(t, source, 4, "db.lock", 1, rightsAll)
	output(t, call(t, source, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	output(t, call(t, source, "fd_sync", `{"fd":4}`), &struct{}{})
	image, err := source.CrashStorage(nil)
	require.NoError(t, err)
	image.model, err = json.Marshal(config.Storage)
	require.NoError(t, err)
	_, err = NewRecoveredEnvironment(config, image)
	require.EqualError(t, err, "WASI invalid: durable image contains process-owned SQLite lock")
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
	corpus := volumeConformanceCorpus{Schema: "gomad.volume-conformance/v1", Name: script.name, Origin: script.origin, Seed: script.seed, Operations: script.operations}
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
				Volume      string                        `json:"volume"`
				Limits      volume.CrashEnumerationLimits `json:"limits"`
				Cursor      string                        `json:"cursor"`
				CursorInput *volume.CrashExploration      `json:"cursor_input,omitempty"`
				Save        string                        `json:"save"`
			}
			volumeConformanceDecode(t, operation.Input, &input)
			cursor := input.CursorInput
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
				imageData, imageDigest, err = volumeConformanceEncodeImage(image, storageImageLimit(script.config))
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
				imageData, imageDigest, err = volumeConformanceEncodeImage(image, storageImageLimit(script.config))
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
		if script.observe != nil {
			script.observe(e, step)
		}
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
	if strings.HasPrefix(script.name, "fault-import-transcript-") {
		corpus.ImportTranscript = slices.Clone(e.Transcript())
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
		readback := slices.Clone(ops)
		for i, name := range []string{"db", "db-wal", "db-journal", "other.lock"} {
			readback = append(readback, volumeConformanceOpen(name, 0, rightsAll), read(uint32(5+i)))
		}
		scripts = append(scripts, volumeConformanceScript{name: fmt.Sprintf("sqlite-durable-readback-opt-in-%v", adapted), origin: "wasi/volume_test.go:TestStorageSQLiteLockAdapterIsExactOptIn", config: config, operations: readback})
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
	// A dependency-closed partial selection can persist more bytes than the
	// volume capacity (only volatile growth is bounded); recovery admits it.
	over := storageConfig()
	over.Storage.Volumes[0].CapacityBytes = 1000
	setSize := func(n uint32, size uint64) volumeConformanceOperation {
		return volumeConformanceOp("fd_filestat_set_size", struct {
			FD   uint32 `json:"fd"`
			Size uint64 `json:"size"`
		}{n, size})
	}
	pathStat := func(name string) volumeConformanceOperation {
		return volumeConformanceOp("path_filestat_get", pathStatInput{FD: 4, Path: []byte(name)})
	}
	scripts = append(scripts, volumeConformanceScript{name: "recover-over-capacity-selection", origin: "fn-176 review: durable state beyond volume capacity after a partial selection", config: over, operations: []volumeConformanceOperation{
		volumeConformanceOpen("a", 1, rightsAll), volumeConformanceWrite(5, strings.Repeat("a", 600)), fd("fd_sync", 5), fd("fd_sync", 4),
		setSize(5, 0), volumeConformanceOpen("b", 1, rightsAll), volumeConformanceWrite(6, strings.Repeat("b", 600)),
		volumeConformanceEnumerate(volume.CrashEnumerationLimits{States: 64, Operations: 64, Depth: 64, Bytes: 1 << 20, Work: 1 << 16}, "", "all"),
		volumeConformanceOp("crash", map[string]any{"page": "all", "state": volumeConformanceOverCapacityState, "image": "over"}),
		volumeConformanceOp("recover", map[string]string{"image": "over"}), pathStat("a"), pathStat("b"),
	}})
	// History keeps every write payload, so rewriting one small file can
	// outgrow the recovery image bound (filesystem bytes + 1 MiB).
	large := storageConfig()
	large.Limits.FilesystemBytes = 64 << 10
	large.Storage.Volumes[0].CapacityBytes = 64 << 10
	rewrites := []volumeConformanceOperation{volumeConformanceOpen("db", 1, rightsAll)}
	for i := 0; i < 17; i++ {
		rewrites = append(rewrites, volumeConformanceWrite(5, strings.Repeat(string(rune('a'+i)), 64<<10)))
	}
	rewrites = append(rewrites, crash)
	scripts = append(scripts, volumeConformanceScript{name: "crash-image-capacity", origin: "fn-176 review: recovery image beyond filesystem bytes + 1 MiB is a capacity outcome", config: large, operations: rewrites})
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
	scripts = append(scripts, volumeConformanceMigrationScripts()...)
	scripts = append(scripts, volumeConformanceCursorScripts(t)...)
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
	if _, err := NewEnvironment(provenance.Config); err == nil || err.Error() != "WASI invalid: profile" {
		t.Fatalf("historical sample profile must be refused: %v", err)
	}
	provenance.Config.Profile = CooperativeProfile
	return volumeConformanceScript{name: "recorded-storage-sqlite-prefix", origin: "authenticated storage_sqlite recorded volume transitions 0..3, projected to two creates; flags/rights/now=1 are derived inputs (see recorded-storage-sqlite/README.md)", seed: 7, config: provenance.Config, operations: []volumeConformanceOperation{
		{Op: "path_open", Input: volumeConformanceOpen("db", 1, rightsAll).Input, Now: 1},
		{Op: "path_open", Input: volumeConformanceOpen("db-journal", 1, rightsAll).Input, Now: 1},
	}}
}

const volumeConformanceBinaryImageIdentity = StorageImageIdentity

func volumeConformanceEncodeImage(image *StorageImage, maximum uint64) ([]byte, string, error) {
	if image == nil || image.fs == nil {
		return nil, "", invalid("durable storage image")
	}
	return encodeStorageImage(image.fs, image.model, image.occurrences, maximum)
}

type volumeCoreVectors struct {
	Schema    string               `json:"schema"`
	Model     string               `json:"model"`
	Scenarios []volumeCoreScenario `json:"scenarios"`
}

type volumeCoreScenario struct {
	Name    string           `json:"name"`
	Storage StorageModel     `json:"storage"`
	Steps   []volumeCoreStep `json:"steps"`
}

type volumeCoreInput struct {
	Path        string              `json:"path,omitempty"`
	Destination string              `json:"destination,omitempty"`
	Handle      string              `json:"handle,omitempty"`
	Flags       *volume.OpenFlags   `json:"flags,omitempty"`
	Mode        uint32              `json:"mode,omitempty"`
	Offset      int64               `json:"offset,omitempty"`
	Whence      int                 `json:"whence,omitempty"`
	Size        int64               `json:"size,omitempty"`
	ModTime     int64               `json:"mod_time,omitempty"`
	Data        []byte              `json:"data_base64,omitempty"`
	Seed        *uint64             `json:"seed,omitempty"`
	Selections  map[string][]uint64 `json:"selections,omitempty"`
}

type volumeCoreStep struct {
	Op            string                        `json:"op"`
	Now           int64                         `json:"now"`
	Input         volumeCoreInput               `json:"input"`
	Snapshots     []volume.VolumeSnapshot       `json:"snapshots"`
	Operations    map[string][]volume.Operation `json:"operations"`
	State         volume.ConformanceState       `json:"state"`
	Selections    map[string][]uint64           `json:"selections"`
	Accepted      *bool                         `json:"accepted,omitempty"`
	Error         string                        `json:"error,omitempty"`
	Capacity      *volume.VolumeCapacityError   `json:"capacity,omitempty"`
	Value         int64                         `json:"value,omitempty"`
	Data          []byte                        `json:"data_base64,omitempty"`
	Names         []string                      `json:"names,omitempty"`
	Cwd           string                        `json:"cwd,omitempty"`
	HandlePath    string                        `json:"handle_path,omitempty"`
	UsedBytes     *uint64                       `json:"used_bytes,omitempty"`
	ImageDigest   string                        `json:"image_digest,omitempty"`
	ImageData     []byte                        `json:"image_base64,omitempty"`
	ImageIdentity string                        `json:"image_identity,omitempty"`
}

func TestVolumeCoreVectors(t *testing.T) {
	got := volumeCoreVectors{
		Schema: "gomad.volume-core-vectors/v1", Model: volumeConformanceModel,
		Scenarios: []volumeCoreScenario{volumeCoreScript(t), volumeCoreNestedOrphanScript(t), volumeCoreMetadataUnlinkScript(t)},
	}
	got.Scenarios = append(got.Scenarios, volumeCoreAPIScripts(t)...)
	got.Scenarios = append(got.Scenarios, volumeCoreGracefulLiteralScript(t))
	if *volumeConformanceOutput != "" {
		require.True(t, filepath.IsAbs(*volumeConformanceOutput), "volume-conformance-output must be an absolute external directory")
		directory := filepath.Join(*volumeConformanceOutput, "vectors")
		require.NoError(t, os.MkdirAll(directory, 0o755))
		data, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(directory, "core.json"), append(data, '\n'), 0o644))
		return
	}
	data, err := os.ReadFile(filepath.Join("testdata", "volume-conformance", "vectors", "core.json"))
	require.NoError(t, err, "Go core vectors missing; generate into an external directory for review")
	var want volumeCoreVectors
	require.NoError(t, json.Unmarshal(data, &want))
	require.Equal(t, want, got, "Go pure volume vector drift; inspect before updating vectors/core.json")
}

func volumeCoreStorage() StorageModel {
	return StorageModel{
		Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 65536}},
		Limits:  volume.VolumeLimits{PendingOperations: 128, Transitions: 256},
	}
}

func volumeCoreCapture(fs *volume.FS, clock *volume.Clock, step volumeCoreStep) volumeCoreStep {
	step.Now = clock.Value
	step.Snapshots = fs.VolumeSnapshots()
	step.Operations = fs.Operations()
	step.State = fs.ConformanceState()
	return step
}

func volumeCoreImage(t *testing.T, fs *volume.FS, storage StorageModel, step *volumeCoreStep) {
	t.Helper()
	model, err := json.Marshal(storage)
	require.NoError(t, err)
	image := &StorageImage{fs: fs, model: model, occurrences: map[string]uint64{}}
	step.ImageData, step.ImageDigest, err = volumeConformanceEncodeImage(image, 1<<20)
	require.NoError(t, err)
	step.ImageIdentity = volumeConformanceBinaryImageIdentity
}

func volumeCoreScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "metadata-rename-detached-lifecycle", Storage: volumeCoreStorage()}
	clock := &volume.Clock{Value: 100}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	file, err := fs.Open("/workspace/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/db", Handle: "db", Flags: &flags, Mode: 0o666})
	clock.Value = 117
	n, err := file.WriteAt([]byte("value"), 0)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")})
	clock.Value = 121
	require.NoError(t, fs.Chmod("/workspace/db", 0o600))
	capture("chmod", volumeCoreInput{Path: "/workspace/db", Mode: 0o600})
	require.NoError(t, fs.Chtimes("/workspace/db", -7))
	capture("chtimes", volumeCoreInput{Path: "/workspace/db", ModTime: -7})
	clock.Value = 133
	require.NoError(t, fs.MkdirAll("/workspace/a/b", 0o755))
	capture("mkdir-all", volumeCoreInput{Path: "/workspace/a/b", Mode: 0o755})
	clock.Value = 150
	require.NoError(t, fs.Rename("/workspace/db", "/workspace/a/b/moved"))
	capture("rename", volumeCoreInput{Path: "/workspace/db", Destination: "/workspace/a/b/moved"})
	for _, seed := range []uint64{0, 1, 7, 42} {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
			Op: "select", Input: volumeCoreInput{Seed: &seed}, Selections: fs.SelectCrashOperations(seed),
		}))
	}
	require.NoError(t, file.Sync())
	capture("file-sync", volumeCoreInput{Handle: "db"})
	for _, directory := range []string{"/workspace", "/workspace/a", "/workspace/a/b"} {
		clock.Value++
		handle, err := fs.Open(directory, volume.OpenFlags{Read: true}, 0)
		require.NoError(t, err)
		require.NoError(t, handle.Sync())
		require.NoError(t, handle.Close())
		capture("directory-sync", volumeCoreInput{Path: directory})
	}
	clock.Value = 171
	require.NoError(t, fs.Remove("/workspace/a/b/moved"))
	capture("remove", volumeCoreInput{Path: "/workspace/a/b/moved"})
	clock.Value = 173
	require.NoError(t, file.Truncate(3))
	capture("truncate", volumeCoreInput{Handle: "db", Size: 3})
	clock.Value = 177
	n, err = file.WriteAt([]byte("end"), 0)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("end")})
	clock.Value = 179
	require.NoError(t, file.Sync())
	capture("file-sync", volumeCoreInput{Handle: "db"})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "db"})
	clock.Value = 182
	fs = fs.CloneDurable()
	capture("clone-durable", volumeCoreInput{})
	clock.Value = 191
	file, err = fs.Open("/workspace/next", flags, 0o600)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/next", Handle: "next", Flags: &flags, Mode: 0o600})
	clock.Value = 193
	n, err = file.WriteAt([]byte("pending"), 0)
	require.NoError(t, err)
	require.Equal(t, 7, n)
	capture("write-at", volumeCoreInput{Handle: "next", Data: []byte("pending")})
	for _, seed := range []uint64{0, 1, 7, 42} {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
			Op: "select", Input: volumeCoreInput{Seed: &seed}, Selections: fs.SelectCrashOperations(seed),
		}))
	}
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "next"})
	clock.Value = 197
	selections := fs.SelectCrashOperations(7)
	fs, err = fs.AdvanceVolumeLifecycle(false, selections)
	require.NoError(t, err)
	capture("advance-crash", volumeCoreInput{Selections: selections})
	clock.Value = 201
	file, err = fs.Open("/workspace/after-crash", flags, 0o600)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/after-crash", Handle: "after-crash", Flags: &flags, Mode: 0o600})
	clock.Value = 203
	n, err = file.WriteAt([]byte("tail"), 0)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	capture("write-at", volumeCoreInput{Handle: "after-crash", Data: []byte("tail")})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "after-crash"})
	clock.Value = 211
	fs, err = fs.AdvanceVolumeLifecycle(true, nil)
	require.NoError(t, err)
	capture("advance-graceful", volumeCoreInput{})
	volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	return scenario
}

func volumeCoreNestedOrphanScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "nested-orphan-path-image", Storage: volumeCoreStorage()}
	clock := &volume.Clock{Value: 100}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	require.NoError(t, fs.Mkdir("/workspace/dir", 0o755))
	capture("mkdir", volumeCoreInput{Path: "/workspace/dir", Mode: 0o755})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	file, err := fs.Open("/workspace/dir/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/dir/db", Handle: "db", Flags: &flags, Mode: 0o666})
	clock.Value = 117
	n, err := file.WriteAt([]byte("value"), 0)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "db"})
	selections := map[string][]uint64{"data": {1, 3, 4, 5, 6}}
	fs, err = fs.AdvanceVolumeLifecycle(false, selections)
	require.NoError(t, err, "Go permits durable child path without its parent's namespace operation")
	accepted := true
	scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
		Op: "advance-crash", Input: volumeCoreInput{Selections: selections}, Accepted: &accepted,
	}))
	volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	return scenario
}

// storageStateEnvelope mirrors the helper's encode_state for decoder tests.
func storageStateEnvelope(version, count uint32, sections ...[]byte) []byte {
	data := []byte(storageStateMagic)
	data = binary.LittleEndian.AppendUint32(data, version)
	data = binary.LittleEndian.AppendUint32(data, count)
	for index, section := range sections {
		data = binary.LittleEndian.AppendUint32(data, uint32(index+1))
		data = binary.LittleEndian.AppendUint64(data, uint64(len(section)))
		data = append(data, section...)
	}
	return sealStorageState(data)
}

func sealStorageState(body []byte) []byte {
	digest := sha256.Sum256(body)
	data := binary.LittleEndian.AppendUint64(append([]byte(nil), body...), uint64(len(body)))
	return append(data, digest[:]...)
}

func validStorageSections() [][]byte {
	return [][]byte{
		[]byte(`[{"ID":"data","Mount":"/workspace","CapacityBytes":1024,"Persisted":[],"Volatile":[],"PendingOperations":0,"PendingSHA256":"","NextOperation":1,"Identity":""}]`),
		[]byte(`{"data":[{"ID":1,"Kind":"alloc","Dependencies":null,"Inode":4,"Offset":0,"Bytes":0,"PayloadSHA256":"","EffectSHA256":"sha256:x","Path":"","Destination":""}]}`),
		[]byte(`{"data":{"States":null,"ChoiceExploration":null,"Complete":true,"Capacity":""}}`),
		[]byte(strings.Repeat("a", 64)),
		storageLogBinding(nil),
	}
}

// contextStorageSections is a complete v3 envelope body: crash limits require
// the pending crash-context section, matching the snapshot's pending state.
func contextStorageSections() [][]byte {
	pending := "sha256:" + strings.Repeat("a", 64)
	sections := validStorageSections()
	sections[0] = []byte(`[{"ID":"data","Mount":"/workspace","CapacityBytes":1024,"Persisted":[],"Volatile":[],"PendingOperations":0,"PendingSHA256":"` + pending + `","NextOperation":1,"Identity":""}]`)
	return append(sections, []byte(`{"data":{"PendingSHA256":"`+pending+`","Operations":[]}}`))
}

func storageLogBinding(log []byte) []byte {
	digest := sha256.Sum256(log)
	return append(binary.LittleEndian.AppendUint64(nil, uint64(len(log))), digest[:]...)
}
