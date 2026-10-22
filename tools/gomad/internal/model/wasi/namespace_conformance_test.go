package wasi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
	"go.temporal.io/server/tools/gomad/internal/support/temporalinput"
)

func namespaceR3ConformanceScripts(t *testing.T) []namespaceConformanceScript {
	t.Helper()
	op := func(name, input string) namespaceConformanceOperation {
		return namespaceConformanceOperation{Op: name, Input: json.RawMessage(input)}
	}
	path := func(name string, fd uint32, value string) namespaceConformanceOperation {
		input, err := json.Marshal(struct {
			FD   uint32 `json:"fd"`
			Path []byte `json:"path_base64"`
		}{fd, []byte(value)})
		if err != nil {
			t.Fatal(err)
		}
		return op(name, string(input))
	}
	open := func(fd uint32, value string, flags uint16, base, inherit uint64) namespaceConformanceOperation {
		input, err := json.Marshal(openInput{fd, []byte(value), 0, flags, base, inherit, 0})
		if err != nil {
			t.Fatal(err)
		}
		return op("path_open", string(input))
	}
	base := testConfig()
	scripts := []namespaceConformanceScript{
		{"schema-order-before-later-null", "handwritten", base, []namespaceConformanceOperation{
			op("fd_read", `{"fd":"bad","length":null}`),
			op("path_open", `{"fd":3,"dirflags":"bad","path_base64":null,"oflags":0,"rights_base":0,"rights_inheriting":0,"fdflags":0}`),
		}},
		{"standard-descriptor-metadata-and-double-close", "handwritten", base, []namespaceConformanceOperation{
			op("fd_fdstat_get", `{"fd":0}`), op("fd_fdstat_set_flags", `{"fd":0,"flags":2}`), op("sock_accept", `{"fd":3,"flags":0}`),
			op("fd_filestat_set_size", `{"fd":3,"size":0}`),
			op("fd_close", `{"fd":3}`), op("fd_close", `{"fd":3}`),
		}},
		{"positional-stderr-refusal-charges-offered-bytes", "handwritten", base, []namespaceConformanceOperation{
			op("fd_pwrite", `{"fd":2,"offset":0,"data_base64":"eA=="}`), op("fd_write", `{"fd":1,"data_base64":"b2s="}`),
		}},
		{"rename-fd-preservation-and-refusal-checkpoints", "handwritten", base, []namespaceConformanceOperation{
			open(3, "source", 1, rightsAll, 0), op("path_rename", `{"fd":3,"path_base64":"c291cmNl","new_fd":3,"new_path_base64":"bWlzc2luZy8="}`),
			op("path_rename", `{"fd":3,"path_base64":"c291cmNl","new_fd":3,"new_path_base64":"dGFyZ2V0"}`), op("fd_filestat_get", `{"fd":5}`),
			path("path_create_directory", 3, "d"), path("path_create_directory", 3, "d/e"), op("path_rename", `{"fd":3,"path_base64":"ZA==","new_fd":3,"new_path_base64":"ZC9lL2Y="}`),
			path("path_remove_directory", 3, "d"), path("path_unlink_file", 3, "d"),
			path("path_create_directory", 3, "empty"), op("path_rename", `{"fd":3,"path_base64":"ZW1wdHk=","new_fd":3,"new_path_base64":"ZA=="}`),
		}},
		{"strict-null-overflow-and-length-precedence", "handwritten", base, []namespaceConformanceOperation{
			op("fd_close", `{"fd":null}`), op("fd_close", `{"fd":4294967296}`), op("fd_close", `{"fd":3,"length":4194305}`),
			op("fd_close", `{"fd":3,"fd":4}`), op("fd_close", `{"fd":3,"extra":0}`),
		}},
		{"open-descriptor-dirflags-fdflags-oflags-precedence", "handwritten", base, []namespaceConformanceOperation{
			op("path_open", `{"fd":999,"path_base64":"Li4veA==","dirflags":2,"oflags":3,"rights_base":536870911,"rights_inheriting":0,"fdflags":2}`),
			op("path_open", `{"fd":3,"path_base64":"Li4veA==","dirflags":2,"oflags":3,"rights_base":536870911,"rights_inheriting":0,"fdflags":2}`),
			op("path_open", `{"fd":3,"path_base64":"Li4veA==","dirflags":0,"oflags":3,"rights_base":536870911,"rights_inheriting":0,"fdflags":2}`),
			op("path_open", `{"fd":3,"path_base64":"Li4veA==","dirflags":0,"oflags":3,"rights_base":536870911,"rights_inheriting":0,"fdflags":0}`),
			open(3, ".", 2, rightPathOpen, 0), open(5, "x", 1, rightRead, 0), open(3, "missing/", 1, rightRead, 0),
		}},
		{"missing-component-before-parent-traversal", "handwritten", base, []namespaceConformanceOperation{
			open(3, "missing/../x", 0, rightRead, 0),
			open(3, "../x", 0, rightRead, 0), open(3, "/x", 0, rightRead, 0), open(3, "missing/../../x", 0, rightRead, 0), open(3, "x\x00y", 0, rightRead, 0),
			open(3, "file", 1, rightsAll, 0), open(3, "file/", 0, rightRead, 0), open(3, "file/../x", 0, rightRead, 0), open(3, "file/.", 0, rightRead, 0),
		}},
		{"raw-byte-readdir-order-and-partial-cookies", "handwritten", base, []namespaceConformanceOperation{
			open(3, string([]byte{255}), 1, rightRead, 0), open(3, "b", 1, rightRead, 0), open(3, "a", 1, rightRead, 0),
			op("fd_readdir", `{"fd":3,"length":100,"cookie":0}`), op("fd_readdir", `{"fd":3,"length":25,"cookie":0}`), op("fd_readdir", `{"fd":3,"length":100,"cookie":1}`),
			op("fd_readdir", `{"fd":3,"length":24,"cookie":0}`), op("fd_readdir", `{"fd":3,"length":1,"cookie":3}`), op("fd_readdir", `{"fd":3,"length":0,"cookie":4}`),
		}},
		{"remove-open-directory-with-unopened-descendant", "handwritten", base, []namespaceConformanceOperation{
			path("path_create_directory", 3, "d"), open(3, "d", 2, rightsAll, rightsAll), path("path_remove_directory", 3, "d"),
			path("path_create_directory", 5, "child"), path("path_create_directory", 5, "child/grandchild"),
		}},
		{"detached-file-data-in-final-digest", "handwritten", base, []namespaceConformanceOperation{
			path("path_create_directory", 3, "a"), path("path_create_directory", 3, "b"), open(3, "a", 2, rightsAll, rightsAll),
			op("path_rename", `{"fd":3,"path_base64":"Yg==","new_fd":3,"new_path_base64":"YQ=="}`), open(5, "child", 1, rightsAll, 0),
			op("fd_pwrite", `{"fd":6,"offset":0,"data_base64":"aW5pdGlhbA=="}`), op("fd_pwrite", `{"fd":6,"offset":0,"data_base64":"Y2hhbmdlZA=="}`),
		}},
	}
	for _, limit := range []uint64{27, 28, 47, 48} {
		config := testConfig()
		config.Limits.TranscriptBytes = limit
		scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("close-record-exact-%d", limit), "handwritten", config, []namespaceConformanceOperation{
			op("fd_close", `{"fd":999}`), op("fd_close", `{"fd":999}`), op("fd_close", `{"fd":999}`),
		}})
	}
	for _, limit := range []uint64{23, 45, 46} {
		config := testConfig()
		config.Limits.TranscriptBytes = limit
		scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("typed-pwrite-exact-reservation-%d", limit), "handwritten", config, []namespaceConformanceOperation{op("fd_pwrite", `{"fd":1,"data_base64":"YWI=","offset":9}`)}})
	}
	for _, limit := range []uint64{23, 24, 35, 36, 45, 46} {
		config := testConfig()
		config.Limits.TranscriptBytes = limit
		scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("binary-transcript-limit-%d", limit), "handwritten", config, []namespaceConformanceOperation{
			op("fd_read", `{"fd":0}`), op("fd_read", `{"fd":0,"length":0}`), op("fd_read", `{"fd":999,"length":10}`), op("fd_close", `{"fd":999}`),
		}})
	}
	config := testConfig()
	config.Clock.EpochNanos = 1000
	config.Clock.ReadStepNanos = 7
	scripts = append(scripts, namespaceConformanceScript{"data-offsets-append-shrink-and-exact-times", "handwritten", config, []namespaceConformanceOperation{
		op("clock_time_get", `{"clock_id":1,"precision":0}`), open(3, "file", 1, rightsAll, 0), op("fd_pwrite", `{"fd":5,"offset":2,"data_base64":"eHl6"}`),
		op("fd_read", `{"fd":5,"length":20}`), op("fd_read", `{"fd":5,"length":20}`), op("fd_seek", `{"fd":5,"offset":-2,"whence":2}`),
		op("fd_fdstat_set_flags", `{"fd":5,"flags":1}`), op("fd_pwrite", `{"fd":5,"offset":0,"data_base64":"IQ=="}`), op("fd_pread", `{"fd":5,"offset":0,"length":20}`),
		op("fd_seek", `{"fd":5,"offset":0,"whence":1}`), op("fd_filestat_get", `{"fd":5}`), op("fd_seek", `{"fd":5,"offset":-9223372036854775808,"whence":0}`),
		op("fd_pread", `{"fd":5,"offset":18446744073709551615,"length":1}`),
		op("fd_filestat_set_size", `{"fd":5,"size":2}`), op("fd_pread", `{"fd":5,"offset":0,"length":20}`), op("fd_sync", `{"fd":5}`),
	}})
	config = testConfig()
	config.Limits.Files = 3
	scripts = append(scripts, namespaceConformanceScript{"directory-create-files-capacity", "handwritten", config, []namespaceConformanceOperation{path("path_create_directory", 3, "new")}})
	config = testConfig()
	config.Limits.TranscriptBytes = 1
	scripts = append(scripts, namespaceConformanceScript{"read-buffer-before-transcript", "handwritten", config, []namespaceConformanceOperation{op("fd_read", `{"fd":99,"length":4194305}`)}})
	config = testConfig()
	config.Limits.TranscriptBytes = 52
	scripts = append(scripts, namespaceConformanceScript{"binary-clock-fold-invalid-and-different-precision", "handwritten", config, []namespaceConformanceOperation{
		op("clock_time_get", `{"clock_id":1,"precision":0}`), op("clock_time_get", `{"clock_id":1,"precision":0}`), op("clock_time_get", `{"clock_id":1,"precision":1}`),
		op("clock_time_get", `{"clock_id":1}`), op("clock_time_get", `{"clock_id":1,"precision":0}`), op("fd_close", `{"fd":999}`),
	}})
	scripts = append(scripts, namespaceConformanceScript{"binary-schema-order-over-write-buffer", "handwritten", testConfig(), []namespaceConformanceOperation{
		{Op: "fd_pwrite", Input: json.RawMessage(`{"fd":1,"data_base64":"","offset":null}`), RepeatData: bufferLimit + 1},
		op("poll_oneoff", `{"subscriptions":[{"userdata":0,"Type":1,"fd":0}]}`),
	}})
	config = testConfig()
	config.Limits.Calls = 1
	scripts = append(scripts, namespaceConformanceScript{"callback-capacity-before-invalid-null", "handwritten", config, []namespaceConformanceOperation{op("fd_close", `{"fd":999}`), op("fd_close", `{"fd":null}`)}})
	config = testConfig()
	config.Stdin = []byte("input")
	config.Limits.FilesystemBytes = 5
	scripts = append(scripts, namespaceConformanceScript{"filesystem-refusal-preserves-data-offset-and-output", "handwritten", config, []namespaceConformanceOperation{
		open(3, "file", 1, rightsAll, 0), op("fd_write", `{"fd":5,"data_base64":"AQ=="}`), op("fd_seek", `{"fd":5,"offset":0,"whence":1}`),
		op("fd_pread", `{"fd":5,"offset":0,"length":10}`), {Op: "fd_write", Input: json.RawMessage(`{"fd":1,"data_base64":""}`), RepeatData: bufferLimit + 1},
	}})
	config = testConfig()
	config.Limits.TranscriptBytes = 28
	scripts = append(scripts, namespaceConformanceScript{"invalid-reserves-zero-then-exact-close", "handwritten", config, []namespaceConformanceOperation{op("fd_read", `{"fd":0}`), op("fd_close", `{"fd":999}`)}})
	config = testConfig()
	config.Limits.TranscriptBytes = 1
	scripts = append(scripts, namespaceConformanceScript{"transcript-refuses-directory-before-mutation", "handwritten", config, []namespaceConformanceOperation{path("path_create_directory", 3, "a")}})
	config = testConfig()
	config.Stdin = []byte("stdin")
	scripts = append(scripts, namespaceConformanceScript{"stdin-ready-then-eof-and-write-badfd", "handwritten", config, []namespaceConformanceOperation{
		op("poll_oneoff", `{"subscriptions":[{"userdata":1,"type":1,"fd":0},{"userdata":2,"type":1,"fd":1},{"userdata":3,"type":1,"fd":3},{"userdata":4,"type":2,"fd":999}]}`),
		op("fd_read", `{"fd":0,"length":5}`),
		op("poll_oneoff", `{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":50,"precision":0,"flags":0},{"userdata":2,"type":1,"fd":0}]}`),
		op("clock_time_get", `{"clock_id":1,"precision":0}`),
	}})
	config = testConfig()
	config.Limits.Files = 4
	scripts = append(scripts, namespaceConformanceScript{"unlinked-open-file-retains-allocation-charge", "handwritten", config, []namespaceConformanceOperation{
		open(3, "file", 1, rightsAll, 0), op("fd_write", `{"fd":5,"data_base64":"eA=="}`), path("path_unlink_file", 3, "file"),
		op("fd_pread", `{"fd":5,"offset":0,"length":10}`), open(3, "new", 1, rightRead, 0), open(3, "file", 0, rightRead, 0),
	}})
	config = testConfig()
	long := "/" + strings.Repeat("x", 4096)
	config.WritableDirectories = []string{long}
	config.WorkingDirectory = long
	config.Environment[0] = "PWD=" + long
	scripts = append(scripts, namespaceConformanceScript{"configured-preopen-exceeds-runtime-path-limit", "handwritten", config, []namespaceConformanceOperation{
		op("fd_prestat_get", `{"fd":3}`), open(3, ".", 2, rightRead, 0), open(3, strings.Repeat("x", 4097), 0, rightRead, 0),
	}})
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/frozen"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	config = testConfig()
	config.CapturedInputs = inputs
	scripts = append(scripts, namespaceConformanceScript{"readonly-precedes-existing-and-size-rights-precede-readonly", "handwritten", config, []namespaceConformanceOperation{
		open(3, "file", 0, rightWrite, 0), path("path_create_directory", 3, "file"), open(3, "file", 0, rightRead|rightSeek|rightSetSize, 0),
		op("fd_filestat_set_size", `{"fd":6,"size":18446744073709551615}`), op("fd_write", `{"fd":6,"data_base64":"AQ=="}`),
	}})
	ancestorInputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/workspace/input"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	config = testConfig()
	config.WritableDirectories = []string{"/workspace"}
	config.CapturedInputs = ancestorInputs
	scripts = append(scripts, namespaceConformanceScript{"decoded-captured-file-refuses-unlink-through-writable-ancestor", "handwritten", config, []namespaceConformanceOperation{
		path("path_unlink_file", 3, "input/file"),
	}})
	for _, test := range []struct {
		name, input, supported string
		limit                  uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, `{"fd":3,"flags":1}`, 75},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, 96},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, "", 92},
	} {
		initialLimit := map[string]uint64{"fd_fdstat_set_flags": 30, "path_open": 61, "path_readlink": 41}[test.name]
		config = testConfig()
		config.Limits.TranscriptBytes = initialLimit
		scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("%s-admission-reservation-%d", test.name, initialLimit), "handwritten", config, []namespaceConformanceOperation{op(test.name, test.input)}})
		for _, input := range []struct{ name, value string }{{"unsupported", test.input}, {"supported", test.supported}} {
			if input.value == "" {
				continue
			}
			limits := []uint64{test.limit - 1, test.limit}
			if input.name == "unsupported" {
				limits = append(limits, 65536)
			}
			for _, limit := range limits {
				config = testConfig()
				config.Limits.TranscriptBytes = limit
				scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("%s-%s-reservation-%d", test.name, input.name, limit), "handwritten", config, []namespaceConformanceOperation{op(test.name, input.value)}})
			}
		}
	}
	for seed := uint64(0); seed < 100; seed++ {
		config, operations := namespaceConformanceGenerated(seed)
		scripts = append(scripts, namespaceConformanceScript{fmt.Sprintf("generated-seed-%03d", seed), "deterministic-prng", config, operations})
	}
	productionSource := t.TempDir()
	if err := os.MkdirAll(filepath.Join(productionSource, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"input": "frozen", "nested/deep": "deeper"} {
		if err := os.WriteFile(filepath.Join(productionSource, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{"/fixtures", "/workspace/frozen"} {
		inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: productionSource, Target: target}}, captured.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		config := testConfig()
		config.CapturedInputs = inputs
		scripts = append(scripts, namespaceConformanceScript{"production-image" + strings.ReplaceAll(target, "/", "-"), "handwritten", config, []namespaceConformanceOperation{open(3, "x", 1, rightWrite, 0)}})
	}
	repo, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	inputs, err = temporalinput.CaptureSchema(repo)
	if err != nil {
		t.Fatal(err)
	}
	activity, err := TemporalConfig(inputs, "/workspace/tests", TemporalPrototype, "-test.run=^TestActivityTestSuite$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	if err != nil {
		t.Fatal(err)
	}
	scripts = append(scripts, namespaceConformanceScript{"production-image-activity-selection", "handwritten", activity, []namespaceConformanceOperation{open(3, "x", 1, rightWrite, 0)}})
	return scripts
}
