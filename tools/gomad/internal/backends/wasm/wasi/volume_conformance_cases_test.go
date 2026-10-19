package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestVolumeConformanceMigrationContracts(t *testing.T) {
	for _, script := range append(volumeConformanceMigrationScripts(), volumeConformanceCursorScripts(t)...) {
		t.Run(script.name, func(t *testing.T) {
			for i := range script.operations {
				script.operations[i].Now = 100 + uint64(i)*17
			}
			got := volumeConformanceRun(t, script)
			if strings.HasPrefix(script.name, "invalid-") {
				require.Contains(t, string(got.InitialReply), `"boundary":"invalid:`)
				require.Empty(t, got.Observations)
				return
			}
			require.Equal(t, "{}", string(got.InitialReply))
			readData := func(index int) []byte {
				var reply struct {
					Errno  uint16      `json:"errno"`
					Output bytesOutput `json:"output"`
				}
				require.NoError(t, json.Unmarshal(got.Observations[index].Reply, &reply))
				require.Zero(t, reply.Errno)
				return reply.Output.Data
			}
			switch script.name {
			case "enumeration-ack-or-new-rights-refusal":
				page := got.Observations[5].Enumeration
				require.True(t, page.Complete)
				require.GreaterOrEqual(t, len(page.States), 2)
				for _, state := range page.States {
					for _, entry := range state.Entries {
						if entry.Path == "/db" {
							require.Contains(t, []string{"ack", "new"}, string(entry.Data))
						}
					}
				}
				require.JSONEq(t, `{"errno":76,"output":{}}`, string(got.Observations[7].Reply))
				require.Empty(t, got.Observations[7].Events)
				require.JSONEq(t, `{"errno":76,"output":{}}`, string(got.Observations[8].Reply))
			case "hash-literal-file-payloads":
				for i, digest := range []string{"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", "sha256:6509423fd9da5c225d2f8619ffae394b40f9f7686fee55a38c54b1424ac65f46", "sha256:6836cf13bac400e9105071cd6af47084dfacad4e5e302c94bfed24e013afb73e"} {
					require.Equal(t, digest, got.Observations[i].Snapshots[0].Volatile[1].DataSHA256)
				}
			case "sparse-write-sync":
				require.Equal(t, []byte{0, 0, 'x', 'y'}, readData(2))
				require.Len(t, got.Observations[3].Snapshots[0].Persisted, 1)
				require.Len(t, got.Observations[4].Snapshots[0].Persisted, 2)
			case "zero-length-write-past-eof":
				require.Equal(t, []byte{0, 0, 0}, readData(2))
				require.Equal(t, uint64(4), got.Observations[1].Snapshots[0].PendingOperations)
				ops := got.Observations[1].Operations[0].Operations
				require.Empty(t, ops[len(ops)-1].PayloadSHA256)
			case "fault-import-transcript-short-write", "fault-import-transcript-torn-write":
				require.Equal(t, []byte("chitial"), readData(5))
			case "fault-import-transcript-io-error", "fault-import-transcript-no-space":
				require.Equal(t, []byte("initial"), readData(5))
			case "subtree-rename-sorted-effects":
				require.Equal(t, "/z/sub/file", got.Observations[3].Operations[0].Operations[len(got.Observations[3].Operations[0].Operations)-1].Destination)
			}
			if strings.HasPrefix(script.name, "fault-import-transcript-") {
				require.NotEmpty(t, got.ImportTranscript)
				require.Empty(t, volumeConformanceDifference(got, volumeConformanceRun(t, script)))
				changed := got
				changed.ImportTranscript = append(slices.Clone(got.ImportTranscript), '\n')
				require.NotEmpty(t, volumeConformanceDifference(got, changed), "the comparison must conserve full WASI import bytes")
				var events []byte
				for _, observation := range got.Observations {
					events = append(events, observation.Events...)
				}
				require.Contains(t, string(events), `"fault":"named-`+strings.TrimPrefix(script.name, "fault-import-transcript-")+`"`)
			}
			if strings.HasPrefix(script.name, "cross-directory-rename-") {
				page := got.Observations[len(got.Observations)-3].Enumeration
				require.True(t, page.Complete)
				count := 1
				if strings.HasSuffix(script.name, "sync-0") {
					count = 2
				}
				require.Len(t, page.States, count)
				for _, observation := range got.Observations {
					require.NotContains(t, string(observation.Reply), `"error"`)
					require.NotContains(t, string(observation.Reply), `"boundary"`)
					var reply struct {
						Errno uint16 `json:"errno"`
					}
					require.NoError(t, json.Unmarshal(observation.Reply, &reply))
					require.Zero(t, reply.Errno)
				}
				require.Equal(t, "{}", string(got.Observations[len(got.Observations)-1].Reply))
			}
			if strings.HasPrefix(script.name, "enumeration-") && script.operations[len(script.operations)-1].Op == "enumerate" {
				last := got.Observations[len(got.Observations)-1]
				require.NotEqual(t, "{}", string(last.Reply), "forged cursor must be refused")
				require.Equal(t, got.Observations[2].Snapshots, last.Snapshots)
				require.Equal(t, got.Observations[2].Operations, last.Operations)
				require.Empty(t, last.Events)
			}
			if script.name == "fault-json-html-and-unicode-escaping" {
				require.True(t, bytes.Contains(got.Observations[1].Events, []byte(`\u003c\u003e\u0026`)))
			}
		})
	}
}

func volumeConformanceCursorScripts(t *testing.T) []volumeConformanceScript {
	t.Helper()
	limits := volume.CrashEnumerationLimits{States: 64, Operations: 64, Depth: 64, Bytes: 1 << 20, Work: 1}
	ops := []volumeConformanceOperation{volumeConformanceOpen("db", 1, rightsAll), volumeConformanceWrite(5, "value"), volumeConformanceEnumerate(limits, "", "first")}
	for i := range ops {
		ops[i].Now = 100 + uint64(i)*17
	}
	reference := volumeConformanceRun(t, volumeConformanceScript{config: storageConfig(), operations: ops})
	var scripts []volumeConformanceScript
	for _, mutation := range []string{"forged-identity", "invalid-cursor", "duplicate-seen", "invalid-seen"} {
		data, err := json.Marshal(reference.Observations[2].Enumeration.ChoiceExploration)
		require.NoError(t, err)
		var cursor volume.CrashExploration
		require.NoError(t, json.Unmarshal(data, &cursor))
		switch mutation {
		case "forged-identity":
			cursor.Identity += "0"
		case "invalid-cursor":
			cursor.Cursor[0] = 2
		case "duplicate-seen":
			cursor.Seen = append(cursor.Seen, cursor.Seen...)
		case "invalid-seen":
			cursor.Seen[0] = "invalid"
		}
		if mutation != "forged-identity" {
			hasher := sha256.New()
			frame := func(data []byte) {
				var length [8]byte
				binary.LittleEndian.PutUint64(length[:], uint64(len(data)))
				hasher.Write(length[:])
				hasher.Write(data)
			}
			frame([]byte("gomad-volume-crash-exploration/v1"))
			frame([]byte(cursor.Volume))
			frame([]byte(cursor.PendingSHA256))
			frame(cursor.Cursor)
			for _, identity := range cursor.Seen {
				frame([]byte(identity))
			}
			cursor.Identity = fmt.Sprintf("sha256:%x", hasher.Sum(nil))
		}
		steps := append([]volumeConformanceOperation(nil), ops...)
		steps = append(steps, volumeConformanceOp("enumerate", map[string]any{"volume": "data", "limits": limits, "cursor_input": cursor}))
		scripts = append(scripts, volumeConformanceScript{name: "enumeration-" + mutation, origin: "volume::crash::stale_invalid_duplicate_and_forged_cursors_are_refused", config: storageConfig(), operations: steps})
	}
	return scripts
}

func volumeCoreMetadataUnlinkScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "directory-metadata-unlink-orphan-history", Storage: volumeCoreStorage()}
	clock := &volume.Clock{Value: 100}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
		volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	file, err := fs.Open("/workspace/file", flags, 0o644)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/file", Handle: "file", Flags: &flags, Mode: 0o644})
	require.NoError(t, fs.Chtimes("/workspace/file", -7))
	capture("chtimes", volumeCoreInput{Path: "/workspace/file", ModTime: -7})
	syncDirectory := func() {
		directory, err := fs.Open("/workspace", volume.OpenFlags{Read: true}, 0)
		require.NoError(t, err)
		require.NoError(t, directory.Sync())
		require.NoError(t, directory.Close())
		capture("directory-sync", volumeCoreInput{Path: "/workspace"})
	}
	syncDirectory()
	require.Len(t, scenario.Steps[len(scenario.Steps)-1].State.Volumes[0].Pending, 1)
	require.NoError(t, fs.Remove("/workspace/file"))
	capture("remove", volumeCoreInput{Path: "/workspace/file"})
	syncDirectory()
	persisted := scenario.Steps[len(scenario.Steps)-1].State.Volumes[0].Persisted
	require.Len(t, persisted.Paths, 1)
	require.Len(t, persisted.Objects, 2)
	require.Equal(t, int64(-7), persisted.Objects[1].ModTime)
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "file"})
	return scenario
}

func volumeConformanceMigrationScripts() []volumeConformanceScript {
	fd := func(op string, n uint32) volumeConformanceOperation { return volumeConformanceOp(op, fdInput{FD: n}) }
	path := func(op, name string) volumeConformanceOperation {
		return volumeConformanceOp(op, pathInput{FD: 4, Path: []byte(name)})
	}
	write := func(n uint32, offset uint64, data string) volumeConformanceOperation {
		return volumeConformanceOp("fd_pwrite", struct {
			FD     uint32 `json:"fd"`
			Offset uint64 `json:"offset"`
			Data   []byte `json:"data_base64"`
		}{n, offset, []byte(data)})
	}
	read := func(n uint32) volumeConformanceOperation {
		return volumeConformanceOp("fd_pread", struct {
			FD     uint32 `json:"fd"`
			Offset uint64 `json:"offset"`
			Length uint32 `json:"length"`
		}{n, 0, 100})
	}
	resize := func(n uint32, size uint64) volumeConformanceOperation {
		return volumeConformanceOp("fd_filestat_set_size", struct {
			FD   uint32 `json:"fd"`
			Size uint64 `json:"size"`
		}{n, size})
	}
	rename := func(old, next string) volumeConformanceOperation {
		return volumeConformanceOp("path_rename", renameInput{FD: 4, Path: []byte(old), NewFD: 4, NewPath: []byte(next)})
	}
	var scripts []volumeConformanceScript
	add := func(name, origin string, config Config, ops ...volumeConformanceOperation) {
		scripts = append(scripts, volumeConformanceScript{name: name, origin: origin, config: config, operations: ops})
	}
	add("hash-literal-file-payloads", "volume::hash::raw_digest_matches_literal_sha256_vectors", storageConfig(), volumeConformanceOpen("hash", 1, rightsAll), write(5, 0, "abc"), write(5, 0, string([]byte{0, 255, 128, 1})), write(5, 0, strings.Repeat("a", 128)))
	for _, action := range []string{"short-write", "torn-write", "io-error", "no-space", "sync-error"} {
		config := storageConfig()
		operation := "write"
		if action == "sync-error" {
			operation = "sync"
		}
		fault := StorageFault{Name: "named-" + action, Resource: "/workspace/db", Operation: operation, Occurrence: 2, Action: action}
		if action == "short-write" || action == "torn-write" {
			fault.Bytes = 2
		}
		config.Storage.Faults = []StorageFault{fault}
		ops := []volumeConformanceOperation{volumeConformanceOpen("other", 1, rightsAll), volumeConformanceOpen("db", 1, rightsAll), write(5, 0, "unrelated"), write(6, 0, "initial")}
		if operation == "sync" {
			ops = append(ops, fd("fd_sync", 6), fd("fd_sync", 6))
		} else {
			ops = append(ops, write(6, 0, "changed"), read(6))
		}
		add("fault-import-transcript-"+action, "TestStorageFaultResourceOccurrenceReplayAndTypedOutcomes", config, ops...)
	}
	add("sparse-write-sync", "volume::state::allocation_write_and_sync_keep_link_separate_from_data", storageConfig(),
		volumeConformanceOpen("file", 1, rightsAll), write(5, 2, "xy"), read(5), fd("fd_sync", 5), fd("fd_sync", 4))
	add("zero-length-write-past-eof", "volume::state::zero_length_write_beyond_eof_still_records_resize_and_write", storageConfig(),
		volumeConformanceOpen("file", 1, rightsAll), write(5, 3, ""), read(5))
	bounded := storageConfig()
	bounded.Storage.Volumes[0].CapacityBytes = 3
	bounded.Storage.Limits.PendingOperations = 4
	add("refused-growth-exact-capacity-pending-resize", "volume::state::capacity_rejection_precedes_volatile_and_history_mutation", bounded,
		volumeConformanceOpen("file", 1, rightsAll), write(5, 0, "four"), write(5, 0, "abc"), resize(5, 1), read(5))
	bounded = storageConfig()
	bounded.Storage.Volumes[0].CapacityBytes = 5
	add("unlinked-handles-charge-and-last-close", "volume::state::detached_open_file_preserves_data_until_the_last_close", bounded,
		volumeConformanceOpen("file", 1, rightsAll), write(5, 0, "abc"), volumeConformanceOpen("file", 0, rightsAll), path("path_unlink_file", "file"), read(5), write(5, 3, "xy"), fd("fd_sync", 5), fd("fd_close", 5), read(6),
		volumeConformanceOpen("next", 1, rightsAll), write(7, 0, "x"), fd("fd_close", 6), write(7, 0, "12345"), read(7))
	add("rename-open-target-last-close", "volume::state::replacing_open_target_retains_its_inode_until_last_close", storageConfig(),
		volumeConformanceOpen("source", 1, rightsAll), write(5, 0, "new"), volumeConformanceOpen("target", 1, rightsAll), write(6, 0, "old"), rename("source", "target"), read(6), read(5), fd("fd_close", 6), fd("fd_close", 5))
	add("subtree-rename-sorted-effects", "volume::state::moving_directory_records_sorted_unlinks_then_sorted_links", storageConfig(),
		path("path_create_directory", "a"), path("path_create_directory", "a/sub"), volumeConformanceOpen("a/sub/file", 1, rightsAll), rename("a", "z"), fd("fd_filestat_get", 5))
	add("volume-preopen-escape", "TestStorageEnumerationAndCapabilityRights", storageConfig(), volumeConformanceOpen("../db", 1, rightsAll))
	config := storageConfig()
	config.Storage.Faults = []StorageFault{{Name: "a<b>&\b\f\x01\u2028\u2029\"\\雪", Resource: "/workspace/é<>&", Operation: "write", Occurrence: 1, Action: "io-error"}}
	add("fault-json-html-and-unicode-escaping", "volume::config::go_json_identity_matches_go_marshal", config,
		volumeConformanceOpen("é<>&", 1, rightsAll), write(5, 0, "x"), write(5, 0, "雪\x00<>&"), read(5), fd("fd_sync", 5), fd("fd_sync", 4))
	for _, test := range []struct {
		name, operation, action string
		bytes                   uint64
	}{
		{"unknown-action", "write", "unknown", 0},
		{"unknown-operation", "chmod", "io-error", 0},
		{"error-payload", "write", "io-error", 1},
		{"crash-payload", "write", "crash-before", 1},
		{"sync-operation", "write", "sync-error", 0},
		{"sync-payload", "sync", "sync-error", 1},
		{"space-operation", "read", "no-space", 0},
		{"space-payload", "write", "no-space", 1},
		{"short-operation", "sync", "short-write", 2},
		{"torn-zero", "write", "torn-write", 0},
		{"torn-overflow", "write", "torn-write", (4 << 20) + 1},
	} {
		config := storageConfig()
		config.Storage.Faults = []StorageFault{{Name: "fault", Resource: "/workspace/db", Operation: test.operation, Occurrence: 1, Action: test.action, Bytes: test.bytes}}
		add("invalid-fault-"+test.name, "volume::config::validation_mirrors_every_go_rule_including_crash_after", config)
	}
	for _, test := range []struct{ name, operation, action string }{
		{"valid-fault-open-no-space", "open", "no-space"},
		{"valid-fault-rename-crash-before", "rename", "crash-before"},
	} {
		config := storageConfig()
		config.Storage.Faults = []StorageFault{{Name: "f", Resource: "/workspace/db", Operation: test.operation, Occurrence: 1, Action: test.action}}
		add(test.name, "volume::config::validation_mirrors_every_go_rule_including_crash_after", config)
	}
	for i, resource := range []string{"/other/db", "/workspace/../db", "relative", "/workspace/db/"} {
		config := storageConfig()
		config.Storage.Faults = []StorageFault{{Name: "fault", Resource: resource, Operation: "write", Occurrence: 1, Action: "io-error"}}
		add(fmt.Sprintf("invalid-fault-resource-%d", i), "volume::config::validation_mirrors_every_go_rule_including_crash_after", config)
	}
	for i, databases := range [][]string{{"/other/db"}, {"/workspace/b", "/workspace/a"}, {"/workspace/db", "/workspace/db"}, {"/workspace/../db"}, {"/workspace/db", "/workspace/db.lock"}, {"/workspace/db", "/workspace/db/inner"}} {
		config := storageConfig()
		config.Storage.SQLiteLockDatabases = databases
		add(fmt.Sprintf("invalid-sqlite-paths-%d", i), "volume::config::validation_mirrors_every_go_rule_including_crash_after", config)
	}
	for _, name := range []string{"ambiguous", "duplicate", "zero-occurrence", "lock-target", "missing-preopen", "volume-capacity", "no-volumes", "many-volumes", "many-faults", "total-capacity", "empty-fault-name", "long-fault-name", "nul-fault-name", "nul-resource", "long-resource", "long-database", "many-databases", "lock-shm-collision", "lock-shm-child-collision"} {
		config := storageConfig()
		fault := StorageFault{Name: "a", Resource: "/workspace/db", Operation: "write", Occurrence: 1, Action: "io-error"}
		switch name {
		case "no-volumes":
			config.Storage.Volumes = nil
		case "many-volumes":
			for i := 0; i < 64; i++ {
				config.Storage.Volumes = append(config.Storage.Volumes, volume.VolumeConfig{ID: fmt.Sprintf("v%d", i), Path: "/workspace", CapacityBytes: 1})
			}
		case "many-faults":
			for i := 0; i < 4097; i++ {
				next := fault
				next.Name = fmt.Sprintf("f%d", i)
				next.Occurrence = uint64(i + 1)
				config.Storage.Faults = append(config.Storage.Faults, next)
			}
		case "total-capacity":
			config.Storage.Volumes[0].CapacityBytes = config.Limits.FilesystemBytes
			config.Storage.Volumes = append(config.Storage.Volumes, volume.VolumeConfig{ID: "temp", Path: "/tmp", CapacityBytes: 1})
		case "empty-fault-name":
			fault.Name = ""
			config.Storage.Faults = []StorageFault{fault}
		case "long-fault-name":
			fault.Name = strings.Repeat("n", 129)
			config.Storage.Faults = []StorageFault{fault}
		case "nul-fault-name":
			fault.Name = "nul\x00name"
			config.Storage.Faults = []StorageFault{fault}
		case "nul-resource":
			fault.Resource = "/workspace/nul\x00path"
			config.Storage.Faults = []StorageFault{fault}
		case "long-resource":
			fault.Resource = "/workspace/" + strings.Repeat("r", 4096)
			config.Storage.Faults = []StorageFault{fault}
		case "long-database":
			config.Storage.SQLiteLockDatabases = []string{"/workspace/" + strings.Repeat("d", 4096-15)}
		case "many-databases":
			for i := 0; i < 65; i++ {
				config.Storage.SQLiteLockDatabases = append(config.Storage.SQLiteLockDatabases, fmt.Sprintf("/workspace/d%03d", i))
			}
		case "lock-shm-collision":
			config.Storage.SQLiteLockDatabases = []string{"/workspace/db", "/workspace/db-shm"}
		case "lock-shm-child-collision":
			config.Storage.SQLiteLockDatabases = []string{"/workspace/db", "/workspace/db-shm/x"}
		case "ambiguous":
			next := fault
			next.Name = "b"
			next.Action = "no-space"
			config.Storage.Faults = []StorageFault{fault, next}
		case "duplicate":
			next := fault
			next.Occurrence = 2
			config.Storage.Faults = []StorageFault{fault, next}
		case "zero-occurrence":
			fault.Occurrence = 0
			config.Storage.Faults = []StorageFault{fault}
		case "lock-target":
			fault.Resource += ".lock"
			config.Storage.Faults = []StorageFault{fault}
			config.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
		case "missing-preopen":
			config.Storage.Volumes[0].Path = "/other"
		case "volume-capacity":
			config.Storage.Volumes[0].CapacityBytes = config.Limits.FilesystemBytes + 1
		}
		add("invalid-storage-"+name, "volume::config::validation_mirrors_every_go_rule_including_crash_after", config)
	}
	limits := volume.CrashEnumerationLimits{States: 64, Operations: 64, Depth: 64, Bytes: 1 << 20, Work: 1 << 16}
	add("enumeration-ack-or-new-rights-refusal", "TestStorageEnumerationAndCapabilityRights", storageConfig(), volumeConformanceOpen("db", 1, rightsAll), write(5, 0, "ack"), fd("fd_sync", 5), fd("fd_sync", 4), write(5, 0, "new"), volumeConformanceEnumerate(volume.CrashEnumerationLimits{States: 16, Operations: 32, Depth: 32, Bytes: 1 << 20, Work: 1 << 16}, "", "all"), volumeConformanceOpen("db", 0, rightRead), fd("fd_sync", 6), volumeConformanceOpen("../db", 1, 0))
	for _, directory := range []bool{false, true} {
		for _, sync := range []uint32{0, 6, 7} {
			ops := []volumeConformanceOperation{path("path_create_directory", "left"), path("path_create_directory", "right"), volumeConformanceOpen("left/db", 1, rightsAll), write(5, 0, "committed"), fd("fd_sync", 5), volumeConformanceOpen("left", 2, rightsAll&^rightWrite), volumeConformanceOpen("right", 2, rightsAll&^rightWrite), fd("fd_sync", 6), fd("fd_sync", 4)}
			old, next := "left/db", "right/db"
			if directory {
				old, next = "left", "right/left"
			}
			ops = append(ops, rename(old, next))
			if sync != 0 {
				n := sync
				if directory && sync == 6 {
					n = 4
				}
				ops = append(ops, fd("fd_sync", n))
			}
			ops = append(ops, volumeConformanceEnumerate(limits, "", "all"))
			if sync == 0 {
				ops = append(ops, volumeConformanceOp("crash", map[string]any{"page": "all", "state": 1, "image": "moved"}))
			} else {
				ops = append(ops, volumeConformanceOp("crash", map[string]any{"image": "moved"}))
			}
			ops = append(ops, volumeConformanceOp("recover", map[string]string{"image": "moved"}))
			add(fmt.Sprintf("cross-directory-rename-directory-%t-sync-%d", directory, sync), "TestVolumeConformanceCrossDirectoryRenameRecovery", storageConfig(), ops...)
		}
	}
	return scripts
}
