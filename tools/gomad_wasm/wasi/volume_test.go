package wasi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio/volume"
)

type storageOpenError struct{}

func (storageOpenError) Error() string   { panic("storage error formatter dispatched arbitrary error") }
func (storageOpenError) Is(_ error) bool { panic("storage error mapper dispatched arbitrary Is") }
func (storageOpenError) As(any) bool     { panic("storage error mapper dispatched arbitrary As") }
func (storageOpenError) Unwrap() error   { panic("storage error mapper dispatched arbitrary Unwrap") }

func TestStorageErrorMappingIsClosed(t *testing.T) {
	for _, test := range []struct {
		err  error
		want uint16
	}{{nil, 0}, {io.EOF, 0}, {volume.ErrClosed, 8}, {syscall.ENOENT, 44}, {syscall.ENOSPC, 51}, {syscall.EIO, 29}, {&volume.VolumeCapacityError{Resource: "volume_bytes"}, 51}, {storageOpenError{}, 29}} {
		if got := storageErrno(test.err); got != test.want {
			t.Fatalf("closed error conversion got %d want %d", got, test.want)
		}
	}
	if storageBoundary(storageOpenError{}) != nil {
		t.Fatal("unknown errors must not become modeled capacity")
	}
	if storageBoundary(&volume.VolumeCapacityError{Resource: "pending_operations"}) == nil {
		t.Fatal("known model capacity must retain typed classification")
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

func TestStorageFileAndDirectorySyncCrashReopen(t *testing.T) {
	c := storageConfig()
	e := newTestEnvironment(t, c)
	fd := openPath(t, e, 4, "db", 1, rightsAll)
	output(t, writeStorage(t, e, fd, 0, "committed"), &writtenOutput{})
	output(t, call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	// File sync persists data but cannot make an unsynced directory entry durable.
	image, err := e.CrashStorage(nil)
	if err != nil {
		t.Fatal(err)
	}
	if call(t, e, "fd_filestat_get", fmt.Sprintf(`{"fd":%d}`, fd)).Errno != errnoBadf {
		t.Fatal("crash retained guest handles")
	}
	recovered, err := NewRecoveredEnvironment(c, image)
	if err != nil {
		t.Fatal(err)
	}
	if reply := call(t, recovered, "path_filestat_get", `{"fd":4,"path_base64":"ZGI=","flags":0}`); reply.Errno != errnoNoent {
		t.Fatalf("unsynced entry survived: %+v", reply)
	}
	fd = openPath(t, recovered, 4, "db", 1, rightsAll)
	output(t, writeStorage(t, recovered, fd, 0, "committed"), &writtenOutput{})
	output(t, call(t, recovered, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	output(t, call(t, recovered, "fd_sync", `{"fd":4}`), &struct{}{})
	output(t, writeStorage(t, recovered, fd, 0, "volatile!"), &writtenOutput{})
	image, err = recovered.CrashStorage(nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		fresh, err := NewRecoveredEnvironment(c, image)
		if err != nil {
			t.Fatal(err)
		}
		fd = openPath(t, fresh, 4, "db", 0, rightsAll)
		if got := readStorage(t, fresh, fd); got != "committed" {
			t.Fatalf("durable bytes after fresh reopen %d: %q", i, got)
		}
	}
}

func TestStorageFaultResourceOccurrenceReplayAndTypedOutcomes(t *testing.T) {
	for _, action := range []string{"short-write", "torn-write", "io-error", "no-space", "sync-error"} {
		t.Run(action, func(t *testing.T) {
			c := storageConfig()
			operation := "write"
			if action == "sync-error" {
				operation = "sync"
			}
			fault := StorageFault{Name: "named-" + action, Resource: "/workspace/db", Operation: operation, Occurrence: 2, Action: action}
			if action == "short-write" || action == "torn-write" {
				fault.Bytes = 2
			}
			c.Storage.Faults = []StorageFault{fault}
			var transcript, model []byte
			for attempt := 0; attempt < 2; attempt++ {
				e := newTestEnvironment(t, c)
				other := openPath(t, e, 4, "other", 1, rightsAll)
				fd := openPath(t, e, 4, "db", 1, rightsAll)
				output(t, writeStorage(t, e, other, 0, "unrelated"), &writtenOutput{})
				output(t, writeStorage(t, e, fd, 0, "initial"), &writtenOutput{})
				if action == "sync-error" {
					output(t, call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
					if reply := call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)); reply.Errno != 29 {
						t.Fatalf("sync error=%+v", reply)
					}
				} else {
					reply := writeStorage(t, e, fd, 0, "changed")
					switch action {
					case "short-write":
						var written writtenOutput
						output(t, reply, &written)
						if written.Written != 2 || readStorage(t, e, fd) != "chitial" {
							t.Fatalf("short result %+v", written)
						}
					case "torn-write":
						if reply.Errno != 29 || readStorage(t, e, fd) != "chitial" {
							t.Fatalf("reviewed prefix tear=%+v", reply)
						}
					case "io-error":
						if reply.Errno != 29 || readStorage(t, e, fd) != "initial" {
							t.Fatalf("io result %+v", reply)
						}
					case "no-space":
						if reply.Errno != 51 || readStorage(t, e, fd) != "initial" {
							t.Fatalf("space result %+v", reply)
						}
					}
				}
				if attempt == 0 {
					transcript, model = e.Transcript(), e.StorageTranscript()
				} else if !bytes.Equal(transcript, e.Transcript()) || !bytes.Equal(model, e.StorageTranscript()) {
					t.Fatal("fault and import transcript replay diverged")
				}
				if !bytes.Contains(e.StorageTranscript(), []byte(fault.Name)) {
					t.Fatal("stable fault evidence absent")
				}
			}
		})
	}
}

func TestStorageCrashFaultAndModelLimits(t *testing.T) {
	c := storageConfig()
	c.Storage.Faults = []StorageFault{{Name: "before-sync", Resource: "/workspace/db", Operation: "sync", Occurrence: 1, Action: "crash-before"}}
	e := newTestEnvironment(t, c)
	fd := openPath(t, e, 4, "db", 1, rightsAll)
	input, _ := json.Marshal(fdInput{FD: fd})
	_, err := e.Handle(Call{Type: "call", ID: e.nextCall, Op: "fd_sync", Input: input})
	var crash *StorageCrash
	if !errors.As(err, &crash) || crash.Fault != "before-sync" {
		t.Fatalf("crash classification: %v", err)
	}
	c = storageConfig()
	c.Storage.Volumes[0].CapacityBytes = 2
	e = newTestEnvironment(t, c)
	fd = openPath(t, e, 4, "db", 1, rightsAll)
	if reply := writeStorage(t, e, fd, 0, "123"); reply.Errno != 51 {
		t.Fatalf("volume space exhaustion must be ENOSPC: %+v", reply)
	}
	if readStorage(t, e, fd) != "" {
		t.Fatal("ENOSPC mutated visible data")
	}
	c = storageConfig()
	c.Storage.Limits.PendingOperations = 1
	e = newTestEnvironment(t, c)
	input, _ = json.Marshal(openInput{FD: 4, Path: []byte("db"), Oflags: 1, Base: rightsAll})
	_, err = e.Handle(Call{Type: "call", ID: e.nextCall, Op: "path_open", Input: input})
	var boundary *BoundaryError
	if !errors.As(err, &boundary) || boundary.Kind != "capacity" || e.namespace.lookup("/workspace/db") != nil {
		t.Fatalf("pending operation bound: err=%v", err)
	}
}

func TestStorageEnumerationAndCapabilityRights(t *testing.T) {
	c := storageConfig()
	e := newTestEnvironment(t, c)
	fd := openPath(t, e, 4, "db", 1, rightsAll)
	output(t, writeStorage(t, e, fd, 0, "ack"), &writtenOutput{})
	output(t, call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	output(t, call(t, e, "fd_sync", `{"fd":4}`), &struct{}{})
	output(t, writeStorage(t, e, fd, 0, "new"), &writtenOutput{})
	states, err := e.EnumerateStorageCrashStates("data", volume.CrashEnumerationLimits{States: 16, Operations: 32, Depth: 32, Bytes: 1 << 20, Work: 1 << 16}, nil)
	if err != nil || !states.Complete || len(states.States) < 2 {
		t.Fatalf("reviewed crash states: %+v %v", states, err)
	}
	for _, state := range states.States {
		for _, entry := range state.Entries {
			if entry.Path == "/db" && string(entry.Data) != "ack" && string(entry.Data) != "new" {
				t.Fatalf("unreviewed torn bytes %q", entry.Data)
			}
		}
	}
	readonly := openPath(t, e, 4, "db", 0, rightRead)
	before := e.StorageTranscript()
	if reply := call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, readonly)); reply.Errno != errnoNotcapable {
		t.Fatal("sync bypassed WASI rights")
	}
	if !bytes.Equal(before, e.StorageTranscript()) {
		t.Fatal("denied operation consumed storage occurrence")
	}
	if reply := call(t, e, "path_open", `{"fd":4,"path_base64":"Li4vZGI=","dirflags":0,"oflags":1,"rights_base":0,"rights_inheriting":0,"fdflags":0}`); reply.Errno != errnoNotcapable {
		t.Fatal("malformed escaping path accepted")
	}
}

func TestStorageSQLiteLockAdapterIsExactOptIn(t *testing.T) {
	for _, adapted := range []bool{false, true} {
		t.Run(fmt.Sprintf("adapted=%v", adapted), func(t *testing.T) {
			c := storageConfig()
			if adapted {
				c.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
			}
			e := newTestEnvironment(t, c)
			openPath(t, e, 4, "db.lock", 1|4, rightsAll)
			output(t, call(t, e, "path_create_directory", `{"fd":4,"path_base64":"ZGItc2ht"}`), &struct{}{})
			for _, name := range []string{"db", "db-wal", "db-journal", "other.lock"} {
				fd := openPath(t, e, 4, name, 1, rightsAll)
				output(t, writeStorage(t, e, fd, 0, "durable"), &writtenOutput{})
				output(t, call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
			}
			output(t, call(t, e, "fd_sync", `{"fd":4}`), &struct{}{})
			image, err := e.CrashStorage(nil)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := NewRecoveredEnvironment(c, image)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"db", "db-wal", "db-journal", "other.lock"} {
				fd := openPath(t, fresh, 4, name, 0, rightsAll)
				if readStorage(t, fresh, fd) != "durable" {
					t.Fatalf("lock adapter changed %s", name)
				}
			}
			for _, name := range []string{"db.lock", "db-shm"} {
				input, _ := json.Marshal(struct {
					FD    uint32 `json:"fd"`
					Path  []byte `json:"path_base64"`
					Flags uint32 `json:"flags"`
				}{4, []byte(name), 0})
				reply := call(t, fresh, "path_filestat_get", string(input))
				if adapted && reply.Errno != errnoNoent || !adapted && reply.Errno != 0 {
					t.Fatalf("process-lock persistence %s %+v", name, reply)
				}
			}
		})
	}
	for _, databases := range [][]string{{"/workspace/db", "/workspace/db"}, {"/workspace/db-shm", "/workspace/db"}, {"/outside/db"}, {"/workspace/db", "/workspace/db.lock"}} {
		c := storageConfig()
		c.Storage.SQLiteLockDatabases = databases
		if _, err := NewEnvironment(c); err == nil {
			t.Fatalf("invalid lock policy accepted: %v", databases)
		}
	}
	c := storageConfig()
	c.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
	c.Storage.Faults = []StorageFault{{Name: "invalid-lock-fault", Resource: "/workspace/db.lock", Operation: "open", Occurrence: 1, Action: "io-error"}}
	if _, err := NewEnvironment(c); err == nil {
		t.Fatal("unmodeled process-owned lock fault accepted")
	}
}
