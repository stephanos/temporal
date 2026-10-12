package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/volume"
)

func storageSQLiteRequest(t *testing.T) Request {
	t.Helper()
	helper := helperPath(t)
	version, err := exec.Command(stockgo.Command(), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("storage guest requires stock Go 1.27.1: %q %v", version, err)
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "storage.wasm")
	fixture, err := filepath.Abs("../testdata/storage_sqlite/main.go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-tags", "sqlite3_dotlk", "-overlay", overlay, "-o", module, fixture)
	command.Dir = "../../../../.."
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build file-backed SQLite guest: %v\n%s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Profile = CooperativeProfile
	config.WritableDirectories = append(config.WritableDirectories, "/volume")
	config.Limits.Calls = 100000
	config.Limits.TranscriptBytes = 32 << 20
	config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}}
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	t.Logf("actual SQLite guest identities: module=%s helper=%s runtime=%s model=%s", fileHash(t, module), fileHash(t, helper), identity, ImplementationSHA256())
	cache := t.TempDir()
	if err := os.Chmod(cache, 0700); err != nil {
		t.Fatal(err)
	}
	return Request{CompileCache: cache, HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, Runtime: &RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}
}

func storageGuestOperation(request Request, operation, journal string, image *StorageImage) Request {
	request.Config.Args = []string{"storage", operation, journal}
	request.StorageImage = image
	control := *request.Runtime
	choices := *control.Choice
	control.Choice = &choices
	request.Runtime = &control
	return request
}

func runStorageGuest(t *testing.T, request Request) Result {
	t.Helper()
	result, err := Run(t.Context(), request)
	if err != nil || !result.Reaped || result.StorageImage == nil {
		t.Fatalf("storage guest: %s %s reaped=%v image=%v err=%v stdout=%s stderr=%s", result.Termination, result.Message, result.Reaped, result.StorageImage != nil, err, result.Stdout, result.Stderr)
	}
	return result
}

func requireStorageGuestExit(t *testing.T, result Result) {
	t.Helper()
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("storage guest failed: %s %s exit=%v stdout=%s stderr=%s", result.Termination, result.Message, result.ExitCode, result.Stdout, result.Stderr)
	}
}

func replayStorageGuest(t *testing.T, request Request, recorded Result) Result {
	t.Helper()
	if recorded.Termination == "exit" || recorded.Termination == "storage_crash" {
		var tape choice.ReplayPlan
		var err error
		mode := choice.ModeReplay
		if recorded.Termination == "storage_crash" {
			if !recorded.RuntimeInterrupted || len(recorded.ChoiceTrace.Bytes) != 0 {
				t.Fatal("storage crash was not retained as interrupted runtime evidence")
			}
			tape, err = choice.ProjectInterruptedReplayPlan(recorded.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
			mode = choice.ModePrefix
		} else {
			tape, err = choice.ProjectReplayPlan(recorded.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		}
		if err != nil {
			t.Fatal(err)
		}
		control := *request.Runtime
		choices := *control.Choice
		choices.Mode, choices.Tape = mode, &tape
		control.Choice = &choices
		request.Runtime = &control
	}
	replayed := runStorageGuest(t, request)
	sameExit := replayed.ExitCode == nil && recorded.ExitCode == nil || replayed.ExitCode != nil && recorded.ExitCode != nil && *replayed.ExitCode == *recorded.ExitCode
	if replayed.Termination != recorded.Termination || !sameExit || replayed.Message != recorded.Message || !bytes.Equal(replayed.Stdout, recorded.Stdout) || !bytes.Equal(replayed.Stderr, recorded.Stderr) || !bytes.Equal(replayed.Transcript, recorded.Transcript) || !bytes.Equal(replayed.StorageTranscript, recorded.StorageTranscript) || !bytes.Equal(replayed.ChoiceTrace.Bytes, recorded.ChoiceTrace.Bytes) || !bytes.Equal(replayed.InterruptedChoiceBytes, recorded.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, recorded.InterruptedDiagnosticBytes) || replayed.RuntimeInterrupted != recorded.RuntimeInterrupted || replayed.StorageImage.Identity() != recorded.StorageImage.Identity() || replayed.EnvironmentIdentity != recorded.EnvironmentIdentity {
		t.Fatalf("fresh storage replay differs: %s/%s message=%q/%q stdout=%s/%s storage=%x/%x choice=%x/%x image=%s/%s", recorded.Termination, replayed.Termination, recorded.Message, replayed.Message, recorded.Stdout, replayed.Stdout, sha256.Sum256(recorded.StorageTranscript), sha256.Sum256(replayed.StorageTranscript), recorded.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, recorded.StorageImage.Identity(), replayed.StorageImage.Identity())
	}
	return replayed
}

func storageGuestEvents(t *testing.T, result Result) []StorageEvent {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(result.StorageTranscript))
	var events []StorageEvent
	for decoder.More() {
		var event StorageEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func requireStorageHistory(t *testing.T, request Request, image *StorageImage, history string) Result {
	t.Helper()
	request.StorageImage = image
	result := runStorageGuest(t, request)
	requireStorageGuestExit(t, result)
	// Exact ordered rows cover both lost and duplicate effects; integrity_check
	// independently checks the recovered database's structural invariant.
	want := "INTEGRITY ok\nHISTORY " + history + "\n"
	if string(result.Stdout) != want {
		t.Fatalf("recovered history: %q, want %q", result.Stdout, want)
	}
	replayStorageGuest(t, request, result)
	// A second incarnation must reach the same state within two fresh opens.
	request.StorageImage = result.StorageImage
	converged := runStorageGuest(t, request)
	requireStorageGuestExit(t, converged)
	if !bytes.Equal(result.Stdout, converged.Stdout) {
		t.Fatalf("recovered history failed bounded convergence: %s/%s", result.Stdout, converged.Stdout)
	}
	return result
}

func TestSQLiteStorageGuestRecovery(t *testing.T) {
	base := storageSQLiteRequest(t)
	t.Run("raw-durable-lock", func(t *testing.T) {
		request := storageGuestOperation(base, "raw-lock", "WAL", nil)
		locked := runStorageGuest(t, request)
		requireStorageGuestExit(t, locked)
		if string(locked.Stdout) != "RAW LOCK durable\n" {
			t.Fatalf("raw lock not acknowledged: %s", locked.Stdout)
		}
		replayStorageGuest(t, request, locked)
		probe := storageGuestOperation(base, "probe-lock", "WAL", locked.StorageImage)
		blocked := runStorageGuest(t, probe)
		requireStorageGuestExit(t, blocked)
		var code, extended int
		if _, err := fmt.Sscanf(string(blocked.Stdout), "LOCKED code=%d extended=%d\n", &code, &extended); err != nil || code != 5 && code != 6 || extended == 0 {
			t.Fatalf("durable lock did not block fresh SQLite with typed BUSY/LOCKED: %s %v", blocked.Stdout, err)
		}
		replayStorageGuest(t, probe, blocked)
		if !slices.ContainsFunc(blocked.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db.lock" }) {
			t.Fatal("raw persisted lock was deleted during recovery")
		}
		t.Logf("raw durable lock blocks reopened SQLite: code=%d extended=%d", code, extended)
	})
	t.Run("acknowledged-WAL-crash", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		model.Faults = []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
		request.Config.Storage = &model
		setup := runStorageGuest(t, storageGuestOperation(request, "setup-live", "WAL", nil))
		requireStorageGuestExit(t, setup)
		crashRequest := storageGuestOperation(request, "append-crash", "WAL", setup.StorageImage)
		crashRequest.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
		crashed := runStorageGuest(t, crashRequest)
		if crashed.Termination != "storage_crash" || string(crashed.Stdout) != "ACK 2\n" {
			t.Fatalf("acknowledged WAL was not crashed with SQLite open: %s %s %s", crashed.Termination, crashed.Message, crashed.Stdout)
		}
		replayed := replayStorageGuest(t, crashRequest, crashed)
		recoverRequest := storageGuestOperation(request, "recover", "WAL", nil)
		requireStorageHistory(t, recoverRequest, crashed.StorageImage, "1:committed,2:committed")
		requireStorageHistory(t, recoverRequest, replayed.StorageImage, "1:committed,2:committed")
		enumeration := crashed.StorageCrashEnumeration["sqlite"]
		if !enumeration.Complete || enumeration.Capacity != "" || len(enumeration.States) <= 1 || crashed.StorageSnapshots[0].PendingOperations == 0 {
			t.Fatalf("live SQLite crash envelope incomplete or trivial: %+v", enumeration)
		}
		markerStates := map[string]bool{}
		for index, state := range enumeration.States {
			t.Run(fmt.Sprintf("selection-%d", index), func(t *testing.T) {
				selectedRequest := crashRequest
				selectedRequest.StorageCrashSelection = map[string][]uint64{"sqlite": state.SelectedOperations}
				selected := runStorageGuest(t, selectedRequest)
				if selected.Termination != "storage_crash" || string(selected.Stdout) != "ACK 2\n" {
					t.Fatalf("selected acknowledged crash: %s %s %s", selected.Termination, selected.Message, selected.Stdout)
				}
				replayStorageGuest(t, selectedRequest, selected)
				recovered := requireStorageHistory(t, recoverRequest, selected.StorageImage, "1:committed,2:committed")
				predicted := slices.IndexFunc(state.Entries, func(entry volume.CrashEntry) bool { return entry.Path == "/marker" })
				actual := slices.IndexFunc(recovered.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/marker" })
				if predicted < 0 {
					if actual >= 0 {
						t.Fatal("unselected marker creation survived the crash")
					}
					markerStates["absent"] = true
					t.Logf("selected operations=%v: marker absent", state.SelectedOperations)
					return
				}
				data := state.Entries[predicted].Data
				if actual < 0 || recovered.StorageSnapshots[0].Persisted[actual].Size != uint64(len(data)) || recovered.StorageSnapshots[0].Persisted[actual].DataSHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
					t.Fatal("actual recovered marker differs from the reviewed native crash state")
				}
				markerStates[string(data)] = true
				t.Logf("selected operations=%v: marker bytes=%d sha256=%s", state.SelectedOperations, len(data), recovered.StorageSnapshots[0].Persisted[actual].DataSHA256)
			})
		}
		if len(markerStates) < 3 {
			t.Fatalf("crash selections did not realize absent, empty and written marker states: %v", markerStates)
		}
		t.Logf("all %d bounded dependency-valid SQLite crash states preserved ACK2; marker states=%d pending operations=%d", len(enumeration.States), len(markerStates), crashed.StorageSnapshots[0].PendingOperations)
	})
	t.Run("live-WAL-commit", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		request.Config.Storage = &model
		setupRequest := storageGuestOperation(request, "setup-live", "WAL", nil)
		setup := runStorageGuest(t, setupRequest)
		requireStorageGuestExit(t, setup)
		replayStorageGuest(t, setupRequest, setup)
		appendRequest := storageGuestOperation(request, "append-live", "WAL", setup.StorageImage)
		appended := runStorageGuest(t, appendRequest)
		requireStorageGuestExit(t, appended)
		if string(appended.Stdout) != "ACK 2\n" {
			t.Fatalf("live WAL transaction not acknowledged: %s", appended.Stdout)
		}
		replayStorageGuest(t, appendRequest, appended)
		if !slices.ContainsFunc(storageGuestEvents(t, appended), func(event StorageEvent) bool { return event.Resource == "/volume/db-wal" && event.Operation == "sync" }) {
			t.Fatal("live WAL commit did not sync its log")
		}
		if !slices.ContainsFunc(appended.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db-wal" }) {
			t.Fatal("live WAL disappeared before crash image capture")
		}
		requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), appended.StorageImage, "1:committed,2:committed")
	})
	for _, journal := range []string{"WAL", "DELETE"} {
		t.Run(journal, func(t *testing.T) {
			setupRequest := storageGuestOperation(base, "setup", journal, nil)
			setup := runStorageGuest(t, setupRequest)
			requireStorageGuestExit(t, setup)
			if len(setup.ChoiceTrace.Records) == 0 {
				t.Fatal("SQLite guest produced no cooperative decisions")
			}
			replayStorageGuest(t, setupRequest, setup)
			appendRequest := storageGuestOperation(base, "append", journal, setup.StorageImage)
			appended := runStorageGuest(t, appendRequest)
			requireStorageGuestExit(t, appended)
			if string(appended.Stdout) != "ACK 2\n" {
				t.Fatalf("successful commit not acknowledged: %s", appended.Stdout)
			}
			replayStorageGuest(t, appendRequest, appended)
			recoverRequest := storageGuestOperation(base, "recover", journal, nil)
			requireStorageHistory(t, recoverRequest, appended.StorageImage, "1:committed,2:committed")
			events := storageGuestEvents(t, appended)
			for _, operation := range []string{"read", "write", "sync"} {
				if !slices.ContainsFunc(events, func(event StorageEvent) bool { return event.Operation == operation }) {
					t.Fatalf("SQLite bypassed storage %s: %+v", operation, events)
				}
			}
			for _, operation := range []string{"fd_pread", "fd_pwrite", "fd_sync"} {
				if !bytes.Contains(appended.Transcript, []byte(`"op":"`+operation+`"`)) {
					t.Fatalf("SQLite did not reach WASI %s", operation)
				}
			}
			resource := "/volume/db-journal"
			if journal == "WAL" {
				resource = "/volume/db-wal"
			}
			for _, action := range []string{"short-write", "io-error", "no-space", "sync-error", "torn-write", "crash-before"} {
				t.Run(action, func(t *testing.T) {
					operation := "write"
					if action == "sync-error" || action == "crash-before" {
						operation = "sync"
					}
					index := slices.IndexFunc(events, func(event StorageEvent) bool { return event.Resource == resource && event.Operation == operation })
					if index < 0 {
						t.Fatalf("missing SQLite %s %s", resource, operation)
					}
					fault := StorageFault{Name: "sqlite-" + action, Resource: resource, Operation: operation, Occurrence: events[index].Occurrence, Action: action}
					if action == "short-write" || action == "torn-write" {
						fault.Bytes = 1
					}
					request := base
					model := *base.Config.Storage
					model.Faults = []StorageFault{fault}
					request.Config.Storage = &model
					initialized := runStorageGuest(t, storageGuestOperation(request, "setup", journal, nil))
					requireStorageGuestExit(t, initialized)
					faultRequest := storageGuestOperation(request, "append", journal, initialized.StorageImage)
					failed := runStorageGuest(t, faultRequest)
					switch action {
					case "crash-before":
						if failed.Termination != "storage_crash" {
							t.Fatalf("fault did not kill helper: %s %s", failed.Termination, failed.Message)
						}
					case "short-write":
						requireStorageGuestExit(t, failed)
						if string(failed.Stdout) != "ACK 2\n" {
							t.Fatalf("Go WriteAt did not complete the short write: %s", failed.Stdout)
						}
					default:
						requireStorageGuestExit(t, failed)
						var code, extended int
						if _, err := fmt.Sscanf(string(failed.Stdout), "REJECTED 2 code=%d extended=%d\n", &code, &extended); err != nil || extended == 0 || code != 10 && (action != "no-space" || code != 13) {
							t.Fatalf("faulty commit did not produce SQLite IOERR/FULL: %s (%v)", failed.Stdout, err)
						}
						t.Logf("actual SQLite storage fault %s: code=%d extended=%d", action, code, extended)
					}
					if !slices.ContainsFunc(storageGuestEvents(t, failed), func(event StorageEvent) bool { return event.Fault == fault.Name && event.Action == action }) {
						t.Fatal("storage fault not realized by actual SQLite guest")
					}
					faultReplay := replayStorageGuest(t, faultRequest, failed)
					recoveredRequest := storageGuestOperation(request, "recover", journal, nil)
					history := "1:committed"
					if action == "short-write" {
						history += ",2:committed"
					}
					recovered := requireStorageHistory(t, recoveredRequest, failed.StorageImage, history)
					replayedRecovery := requireStorageHistory(t, recoveredRequest, faultReplay.StorageImage, history)
					if !bytes.Equal(recovered.Stdout, replayedRecovery.Stdout) || !bytes.Equal(recovered.StorageTranscript, replayedRecovery.StorageTranscript) {
						t.Fatal("replayed fault recovered a different history")
					}
				})
			}
		})
	}
}

func TestSQLiteStorageGuestUnsyncedAcknowledgementCanary(t *testing.T) {
	request := storageSQLiteRequest(t)
	setup := runStorageGuest(t, storageGuestOperation(request, "setup", "WAL", nil))
	requireStorageGuestExit(t, setup)
	unsyncedRequest := storageGuestOperation(request, "unsynced", "WAL", setup.StorageImage)
	unsynced := runStorageGuest(t, unsyncedRequest)
	requireStorageGuestExit(t, unsynced)
	if string(unsynced.Stdout) != "ACK 2\n" {
		t.Fatalf("negative canary did not acknowledge: %s", unsynced.Stdout)
	}
	replayStorageGuest(t, unsyncedRequest, unsynced)
	recovered := requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), unsynced.StorageImage, "1:committed")
	if bytes.Contains(recovered.Stdout, []byte("2:committed")) {
		t.Fatal("missing-sync durability violation escaped the history oracle")
	}
}
