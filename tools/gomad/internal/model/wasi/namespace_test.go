package wasi

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
)

var updateNamespaceConformance = flag.Bool("update-conformance", false, "update namespace conformance corpus from the Go reference")

type namespaceConformanceEntry struct {
	Path      []byte `json:"path_base64"`
	Directory bool   `json:"directory"`
	Readonly  bool   `json:"readonly"`
	Data      []byte `json:"data_base64"`
}

type namespaceConformanceLimits struct {
	Files           uint64 `json:"files"`
	Descriptors     uint64 `json:"descriptors"`
	FilesystemBytes uint64 `json:"filesystem_bytes"`
	OutputBytes     uint64 `json:"output_bytes"`
	Calls           uint64 `json:"calls"`
	TranscriptBytes uint64 `json:"transcript_bytes"`
	PendingEvents   uint64 `json:"pending_events"`
}

type namespaceConformanceImage struct {
	Entries             []namespaceConformanceEntry `json:"entries"`
	WritableDirectories [][]byte                    `json:"writable_directories"`
	Preopens            [][]byte                    `json:"preopens"`
	WorkingDirectory    []byte                      `json:"working_directory"`
	Stdin               []byte                      `json:"stdin_base64"`
	Limits              namespaceConformanceLimits  `json:"limits"`
	EpochNanos          uint64                      `json:"epoch_nanos"`
	ReadStepNanos       uint64                      `json:"read_step_nanos"`
}

type namespaceConformanceSession struct {
	Model string                    `json:"model"`
	Image namespaceConformanceImage `json:"image"`
	// Live sessions also carry Go's production image for the same namespace;
	// the driver must decode it to the same state and then runs on it.
	BinaryImage *namespaceConformanceBinaryImage `json:"binary_image,omitempty"`
}

type namespaceConformanceBinaryImage struct {
	Data  []byte `json:"data_base64"`
	Files uint64 `json:"files"`
}

type namespaceConformanceOperation struct {
	Op    string          `json:"op"`
	Input json.RawMessage `json:"input"`
	// Oversized writes stay compact on disk; the driver receives the expanded WASI input.
	RepeatData uint32 `json:"repeat_data,omitempty"`
}

type namespaceConformanceCorpus struct {
	Name       string                          `json:"name"`
	Origin     string                          `json:"origin"`
	Session    namespaceConformanceSession     `json:"session"`
	Operations []namespaceConformanceOperation `json:"operations"`
	Replies    []json.RawMessage               `json:"replies"`
	Digests    *[]string                       `json:"digests,omitempty"`
	Digest     string                          `json:"digest"`
}

func TestNamespaceConformanceCorpus(t *testing.T) {
	for _, script := range namespaceConformanceScripts(t) {
		t.Run(script.name, func(t *testing.T) {
			environment := newTestEnvironment(t, script.config)
			corpus := namespaceConformanceCorpus{Name: script.name, Origin: script.origin, Session: namespaceConformanceSession{"namespace", namespaceConformanceInitialImage(environment, script.config), nil}, Operations: script.operations}
			image, declaration, err := declareNamespaceImage(environment)
			if err != nil {
				t.Fatal(err)
			}
			var binaryImage bytes.Buffer
			if err := image.write(&binaryImage); err != nil {
				t.Fatal(err)
			}
			corpus.Session.BinaryImage = &namespaceConformanceBinaryImage{binaryImage.Bytes(), declaration.Files}
			digests := make([]string, 0, len(corpus.Operations))
			corpus.Digests = &digests
			for index, operation := range corpus.Operations {
				corpus.Replies = append(corpus.Replies, namespaceConformanceReply(t, environment, operation, index))
				digests = append(digests, namespaceConformanceDigest(environment))
			}
			corpus.Digest = namespaceConformanceDigest(environment)
			projections := []namespaceConformanceCorpus{corpus}
			if namespaceLegacyConformanceCase(script.name) {
				enhanced := corpus
				enhanced.Name += "-binary-image-and-step-digests"
				corpus.Session.BinaryImage, corpus.Digests = nil, nil
				projections = []namespaceConformanceCorpus{corpus, enhanced}
			}
			for _, projection := range projections {
				data, err := json.MarshalIndent(projection, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				data = append(data, '\n')
				name := filepath.Join("testdata", "namespace-conformance", projection.Name+".json")
				if *updateNamespaceConformance {
					if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(name, data, 0644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(name)
				if err != nil {
					t.Fatal("namespace corpus missing; regenerate with -update-conformance: ", err)
				}
				if !bytes.Equal(data, want) {
					t.Fatalf("Go namespace drift in %s; inspect the difference before regenerating with -update-conformance", name)
				}
			}
		})
	}
}

func namespaceLegacyConformanceCase(name string) bool {
	switch name {
	case "capacity-buffer-bytes", "capacity-callbacks", "capacity-descriptors", "capacity-files",
		"capacity-filesystem-bytes", "capacity-offered-output-bytes", "capacity-pending-events",
		"capacity-transcript-bytes", "capacity-write-buffer", "detached-directory",
		"directories-and-rename", "errno-and-containment", "invalid-fields-before-transcript",
		"metadata-and-data", "output-and-mixed-poll", "readonly", "recorded-stock-guest",
		"strict-and-unsupported":
		return true
	default:
		return false
	}
}

func namespaceConformanceHelper(t *testing.T) string {
	t.Helper()
	name := os.Getenv("GOMAD_WASM_CONFORMANCE_HELPER")
	explicit := name != ""
	if !explicit {
		name = "../../../rust/target/release/gomad-wasmhost"
	}
	if explicit && !filepath.IsAbs(name) {
		t.Fatal("GOMAD_WASM_CONFORMANCE_HELPER must be absolute")
	}
	name, err := filepath.Abs(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); err != nil {
		if explicit || os.Getenv("GOMAD_WASM_CONFORMANCE_REQUIRED") == "1" {
			t.Fatal(err)
		}
		t.Skip("build the test-only conformance helper with make test-conformance: ", err)
	}
	return name
}

func namespaceConformanceTransport(t *testing.T, helper, label string) (func(any) []byte, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, helper)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	closeDriver := func() {
		defer cancel()
		if err := input.Close(); err != nil && ctx.Err() == nil {
			t.Error(err)
		}
		if err := command.Wait(); err != nil {
			t.Errorf("%s conformance driver: %v; stderr=%s", label, err, stderr.String())
		}
		if err := helperProfileStderr(stderr.Bytes()); err != nil {
			t.Errorf("%s conformance driver: %v", label, err)
		}
	}
	reader := bufio.NewReaderSize(output, 64<<10)
	exchange := func(value any) []byte {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := input.Write(append(data, '\n')); err != nil {
			t.Fatal(err)
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatalf("%s driver reply: %v; stderr=%s", label, err, stderr.String())
		}
		if len(line) > frameLimit {
			t.Fatal("conformance reply exceeds frame bound")
		}
		return bytes.TrimSuffix(line, []byte{'\n'})
	}
	return exchange, closeDriver
}

func namespaceConformanceExpanded(t *testing.T, operation namespaceConformanceOperation) namespaceConformanceOperation {
	t.Helper()
	if operation.RepeatData == 0 {
		return operation
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(operation.Input, &input); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(bytes.Repeat([]byte{'x'}, int(operation.RepeatData)))
	if err != nil {
		t.Fatal(err)
	}
	input["data_base64"] = data
	operation.Input, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	operation.RepeatData = 0
	return operation
}

func namespaceConformanceReply(t *testing.T, environment *Environment, operation namespaceConformanceOperation, index int) json.RawMessage {
	t.Helper()
	operation = namespaceConformanceExpanded(t, operation)
	reply, err := environment.Handle(Call{Type: "call", ID: environment.nextCall, Op: operation.Op, Input: operation.Input})
	var value any
	if err != nil {
		boundary, ok := err.(*BoundaryError)
		if !ok {
			t.Fatalf("op %d: %v", index, err)
		}
		value = struct {
			Boundary string `json:"boundary"`
		}{boundary.Kind + ":" + boundary.Dimension}
	} else {
		value = struct {
			Errno       uint16          `json:"errno"`
			Output      json.RawMessage `json:"output"`
			Unsupported bool            `json:"unsupported,omitempty"`
			Message     string          `json:"message,omitempty"`
		}{reply.Errno, reply.Output, reply.Unsupported, reply.Message}
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func namespaceConformanceInitialImage(environment *Environment, config Config) namespaceConformanceImage {
	image := namespaceConformanceImage{Entries: []namespaceConformanceEntry{}, WritableDirectories: [][]byte{}, Preopens: [][]byte{}, WorkingDirectory: []byte(config.WorkingDirectory), Stdin: slices.Clone(environment.namespace.descriptors[0].node.data), Limits: namespaceConformanceLimits{config.Limits.Files, config.Limits.Descriptors, config.Limits.FilesystemBytes, config.Limits.OutputBytes, config.Limits.Calls, config.Limits.TranscriptBytes, config.Limits.PendingEvents}, EpochNanos: config.Clock.EpochNanos, ReadStepNanos: config.Clock.ReadStepNanos}
	paths := map[*node]string{}
	var walk func(*node, string)
	walk = func(n *node, name string) {
		paths[n] = name
		for key, child := range n.children {
			walk(child, strings.TrimSuffix(name, "/")+"/"+key)
		}
	}
	walk(environment.namespace.root, "/")
	nodes := make([]*node, 0, len(paths))
	for n := range paths {
		if n != environment.namespace.root {
			nodes = append(nodes, n)
		}
	}
	slices.SortFunc(nodes, func(a, b *node) int {
		if a.ino < b.ino {
			return -1
		}
		if a.ino > b.ino {
			return 1
		}
		return 0
	})
	for _, n := range nodes {
		image.Entries = append(image.Entries, namespaceConformanceEntry{[]byte(paths[n]), n.directory, n.readonly, slices.Clone(n.data)})
	}
	for _, name := range config.WritableDirectories {
		image.WritableDirectories = append(image.WritableDirectories, []byte(name))
	}
	for fd := uint32(3); fd < environment.namespace.nextFD; fd++ {
		image.Preopens = append(image.Preopens, []byte(environment.namespace.descriptors[fd].preopen))
	}
	return image
}

// Digest v1 is SHA-256 over a sorted tree and descriptor-only nodes, then sorted
// descriptors and namespace counters. Scalars are little-endian u64, booleans
// u8, and bytes have a u64 length prefix. The protocol note defines the fields.
func namespaceConformanceDigest(environment *Environment) string {
	h := sha256.New()
	h.Write([]byte("gomad.namespace-conformance/v1\x00"))
	number := func(value uint64) { var data [8]byte; binary.LittleEndian.PutUint64(data[:], value); h.Write(data[:]) }
	blob := func(data []byte) { number(uint64(len(data))); h.Write(data) }
	boolean := func(value bool) {
		if value {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	}
	nodeState := func(n *node) {
		number(n.ino)
		boolean(n.directory)
		boolean(n.readonly)
		number(nodeSize(n))
		blob(n.data)
		number(n.modified)
	}
	seen := map[*node]bool{}
	var walk func(*node, string)
	walk = func(n *node, name string) {
		seen[n] = true
		h.Write([]byte{'N'})
		blob([]byte(name))
		nodeState(n)
		names := make([]string, 0, len(n.children))
		for key := range n.children {
			names = append(names, key)
		}
		slices.Sort(names)
		for _, key := range names {
			walk(n.children[key], strings.TrimSuffix(name, "/")+"/"+key)
		}
	}
	walk(environment.namespace.root, "/")
	fds := make([]uint32, 0, len(environment.namespace.descriptors))
	detached := map[*node]bool{}
	var retain func(*node)
	retain = func(n *node) {
		if seen[n] || detached[n] {
			return
		}
		detached[n] = true
		for _, child := range n.children {
			retain(child)
		}
	}
	for fd, d := range environment.namespace.descriptors {
		fds = append(fds, fd)
		retain(d.node)
	}
	nodes := make([]*node, 0, len(detached))
	for n := range detached {
		nodes = append(nodes, n)
	}
	slices.SortFunc(nodes, func(a, b *node) int {
		if a.ino < b.ino {
			return -1
		}
		if a.ino > b.ino {
			return 1
		}
		return 0
	})
	for _, n := range nodes {
		h.Write([]byte{'O'})
		nodeState(n)
		names := make([]string, 0, len(n.children))
		for key := range n.children {
			names = append(names, key)
		}
		slices.Sort(names)
		number(uint64(len(names)))
		for _, name := range names {
			blob([]byte(name))
			number(n.children[name].ino)
		}
	}
	slices.Sort(fds)
	for _, fd := range fds {
		d := environment.namespace.descriptors[fd]
		h.Write([]byte{'D'})
		number(uint64(fd))
		number(d.node.ino)
		number(d.offset)
		number(d.rights)
		number(d.inheriting)
		number(uint64(d.flags))
		blob([]byte(d.preopen))
		number(uint64(d.std))
	}
	h.Write([]byte{'S'})
	number(uint64(environment.namespace.nextFD))
	number(environment.namespace.nextInode)
	number(environment.namespace.files)
	number(environment.namespace.bytes)
	number(environment.offeredOutput)
	blob(environment.stdout)
	blob(environment.stderr)
	return hex.EncodeToString(h.Sum(nil))
}

type namespaceConformanceScript struct {
	name, origin string
	config       Config
	operations   []namespaceConformanceOperation
}

func namespaceConformanceScripts(t *testing.T) []namespaceConformanceScript {
	t.Helper()
	op := func(name, input string) namespaceConformanceOperation {
		return namespaceConformanceOperation{Op: name, Input: json.RawMessage(input)}
	}
	pathOp := func(name string, fd uint32, path string) namespaceConformanceOperation {
		data, err := json.Marshal(struct {
			FD   uint32 `json:"fd"`
			Path []byte `json:"path_base64"`
		}{fd, []byte(path)})
		if err != nil {
			t.Fatal(err)
		}
		return op(name, string(data))
	}
	open := func(fd uint32, path string, flags uint16, base, inherit uint64, fdflags uint16) namespaceConformanceOperation {
		data, err := json.Marshal(openInput{fd, []byte(path), 0, flags, base, inherit, fdflags})
		if err != nil {
			t.Fatal(err)
		}
		return op("path_open", string(data))
	}
	rename := func(old, new string) namespaceConformanceOperation {
		data, err := json.Marshal(struct {
			FD      uint32 `json:"fd"`
			Path    []byte `json:"path_base64"`
			NewFD   uint32 `json:"new_fd"`
			NewPath []byte `json:"new_path_base64"`
		}{3, []byte(old), 3, []byte(new)})
		if err != nil {
			t.Fatal(err)
		}
		return op("path_rename", string(data))
	}
	base := testConfig()
	scripts := []namespaceConformanceScript{
		{"metadata-and-data", "handwritten", base, []namespaceConformanceOperation{
			op("fd_prestat_get", `{"fd":3}`), op("fd_prestat_dir_name", `{"fd":3,"length":4}`), op("fd_prestat_dir_name", `{"fd":3,"length":3}`), op("fd_prestat_get", `{"fd":0}`),
			op("path_filestat_get", `{"fd":3,"path_base64":"Lg==","flags":0}`), op("fd_seek", `{"fd":3,"offset":0,"whence":0}`),
			open(3, "b", 1, rightsAll, 0, 0), op("fd_fdstat_get", `{"fd":5}`), op("fd_filestat_get", `{"fd":5}`),
			op("fd_pwrite", `{"fd":5,"offset":2,"data_base64":"eHl6"}`), op("fd_read", `{"fd":5,"length":20}`), op("fd_read", `{"fd":5,"length":20}`),
			op("fd_seek", `{"fd":5,"offset":-2,"whence":2}`), op("fd_pread", `{"fd":5,"length":2,"offset":2}`), op("fd_read", `{"fd":5,"length":1}`),
			op("fd_filestat_set_size", `{"fd":5,"size":8}`), op("fd_fdstat_set_flags", `{"fd":5,"flags":1}`), op("fd_pwrite", `{"fd":5,"offset":0,"data_base64":"eA=="}`),
			op("fd_sync", `{"fd":5}`), op("fd_filestat_get", `{"fd":5}`), open(3, "a", 1, rightRead, 0, 0),
			op("fd_readdir", `{"fd":3,"length":25,"cookie":0}`), op("fd_readdir", `{"fd":3,"length":100,"cookie":1}`), op("fd_readdir", `{"fd":3,"length":1,"cookie":9}`), op("fd_readdir", `{"fd":3,"length":0,"cookie":2}`),
			pathOp("path_unlink_file", 3, "b"), op("fd_read", `{"fd":5,"length":20}`), op("fd_close", `{"fd":5}`), op("fd_close", `{"fd":5}`),
		}},
		{"errno-and-containment", "handwritten", base, []namespaceConformanceOperation{
			op("fd_read", `{"fd":999,"length":1}`), op("fd_read", `{"fd":1,"length":1}`), op("fd_read", `{"fd":3,"length":1}`), op("fd_write", `{"fd":3,"data_base64":"eA=="}`),
			open(3, "missing", 0, rightsAll, 0, 0), pathOp("path_create_directory", 3, "../escape"), pathOp("path_create_directory", 3, "/absolute"), pathOp("path_create_directory", 3, "a/../../escape"), pathOp("path_create_directory", 3, "bad\x00name"), pathOp("path_create_directory", 3, strings.Repeat("x", 4097)), pathOp("path_create_directory", 3, ""),
			op("path_filestat_get", `{"fd":3,"path_base64":"bWlzc2luZw==","flags":0}`), op("path_filestat_get", `{"fd":3,"path_base64":"Lg==","flags":2}`), op("fd_sync", `{"fd":0}`),
			open(3, "file", 3, rightRead, 0, 0), open(3, "file", 1, rightsAll, 0, 0), open(3, "file", 5, rightsAll, 0, 0), open(3, "file", 2, rightRead, 0, 0), open(3, ".", 8, rightsAll, 0, 0),
			open(3, "file/", 0, rightRead, 0, 0), pathOp("path_create_directory", 3, "file/../x"), pathOp("path_create_directory", 3, "file/."), pathOp("path_create_directory", 5, "x"),
			op("fd_pread", `{"fd":5,"length":1,"offset":9223372036854775808}`), op("fd_pwrite", `{"fd":5,"offset":9223372036854775808,"data_base64":"eA=="}`), op("fd_seek", `{"fd":5,"offset":-9223372036854775808,"whence":0}`), op("fd_seek", `{"fd":5,"offset":9223372036854775807,"whence":0}`), op("fd_seek", `{"fd":5,"offset":1,"whence":1}`), op("fd_seek", `{"fd":5,"offset":0,"whence":3}`),
			op("fd_readdir", `{"fd":5,"length":1,"cookie":0}`), pathOp("path_remove_directory", 3, "file"), pathOp("path_unlink_file", 3, "."), pathOp("path_remove_directory", 3, "."),
			open(3, ".", 2, rightPathOpen, rightsAll, 0), open(6, "new", 1, rightRead, 0, 0),
			open(3, "file", 0, rightRead, 0, 0), op("fd_write", `{"fd":7,"data_base64":"eA=="}`), op("fd_pread", `{"fd":7,"length":1,"offset":0}`), op("fd_filestat_get", `{"fd":7}`),
		}},
		{"directories-and-rename", "handwritten", base, []namespaceConformanceOperation{
			pathOp("path_create_directory", 3, "dir"), pathOp("path_create_directory", 3, "dir"), pathOp("path_create_directory", 3, "dir/child"), pathOp("path_remove_directory", 3, "dir"), rename("dir", "dir/child/cycle"),
			open(3, "source", 1, rightsAll, 0, 0), op("fd_write", `{"fd":5,"data_base64":"c3RhYmxl"}`), rename("source", "missing/"), rename("source", "source"), rename("source", "target"), rename("target", "dir"), rename("dir", "target"),
			pathOp("path_remove_directory", 3, "dir/child"), pathOp("path_remove_directory", 3, "dir"), pathOp("path_unlink_file", 3, "target"), op("fd_pread", `{"fd":5,"offset":0,"length":20}`),
		}},
		{"output-and-mixed-poll", "handwritten", base, []namespaceConformanceOperation{
			op("fd_write", `{"fd":1,"data_base64":"b3V0"}`), op("fd_write", `{"fd":2,"data_base64":"ZXJy"}`), op("fd_pwrite", `{"fd":1,"offset":0,"data_base64":"eA=="}`),
			open(3, "poll", 1, rightsAll, 0, 0), op("fd_write", `{"fd":5,"data_base64":"YWJj"}`), op("fd_seek", `{"fd":5,"offset":1,"whence":0}`),
			op("poll_oneoff", `{"subscriptions":[{"userdata":10,"type":0,"clock_id":1,"timeout":99,"precision":0,"flags":0},{"userdata":11,"type":1,"fd":5},{"userdata":12,"type":2,"fd":5},{"userdata":13,"type":1,"fd":3},{"userdata":14,"type":1,"fd":999},{"userdata":15,"type":1,"fd":1}]}`),
			op("poll_oneoff", `{"subscriptions":[{"userdata":20,"type":0,"clock_id":1,"timeout":99,"precision":0,"flags":0},{"userdata":21,"type":0,"clock_id":0,"timeout":946684800000000200,"precision":0,"flags":1}]}`), op("clock_time_get", `{"clock_id":1,"precision":0}`),
			op("fd_seek", `{"fd":5,"offset":0,"whence":2}`), op("poll_oneoff", `{"subscriptions":[{"userdata":22,"type":1,"fd":5}]}`), op("poll_oneoff", `{"subscriptions":[]}`),
		}},
		{"strict-and-unsupported", "handwritten", base, []namespaceConformanceOperation{
			op("fd_read", `{"fd":0}`), op("fd_read", `{"fd":0,"length":null}`), op("fd_read", `{"fd":0,"length":1,"extra":1}`), op("fd_read", `{"fd":0,"fd":1,"length":1}`), op("fd_read", `{"fd":"0","length":1}`),
			op("fd_close", `{"\ud800":0}`), op("fd_close", `{"\udfff":0}`), op("fd_close", `{"fd":0,"extra":{"\ud800":0}}`), op("fd_close", `{"\ud83d\ude00":0}`),
			op("fd_close", `{"fd":0,"extra":{"\ud800":0},"fd":1}`), op("fd_close", `{"z":"\ud800","a":{"x":0,"x":1}}`), op("fd_close", `{"z":`+strings.Repeat("[", 33)+`0`+strings.Repeat("]", 33)+`,"a":"\ud800"}`),
			op("fd_fdstat_set_flags", `{"fd":1,"flags":2}`), open(3, "x", 1, rightsAll, 0, 2), op("path_readlink", `{"fd":3,"path_base64":"eA==","length":1}`), op("path_symlink", `{"old_path_base64":"eA==","fd":3,"path_base64":"eQ=="}`), op("sock_accept", `{"fd":3,"flags":0}`),
		}},
	}
	for _, capacity := range []string{"files", "descriptors", "filesystem-bytes", "offered-output-bytes", "write-buffer", "buffer-bytes", "callbacks", "transcript-bytes", "pending-events"} {
		config := testConfig()
		operations := []namespaceConformanceOperation{open(3, "file", 1, rightsAll, 0, 0)}
		switch capacity {
		case "files":
			config.Limits.Files = 3
		case "descriptors":
			config.Limits.Descriptors = 5
		case "filesystem-bytes":
			config.Stdin = []byte("full")
			config.Limits.FilesystemBytes = 4
			operations = append(operations, op("fd_write", `{"fd":5,"data_base64":"eA=="}`))
		case "offered-output-bytes":
			config.Limits.OutputBytes = 2
			operations = []namespaceConformanceOperation{op("fd_close", `{"fd":1}`), op("fd_write", `{"fd":1,"data_base64":"eHk="}`), op("fd_write", `{"fd":1,"data_base64":"eA=="}`)}
		case "write-buffer":
			operations = []namespaceConformanceOperation{{Op: "fd_write", Input: json.RawMessage(`{"fd":999,"data_base64":""}`), RepeatData: bufferLimit + 1}}
		case "buffer-bytes":
			operations = []namespaceConformanceOperation{op("fd_read", `{"fd":999,"length":4194305}`)}
		case "callbacks":
			config.Limits.Calls = 1
			operations = []namespaceConformanceOperation{op("fd_fdstat_get", `{"fd":0}`), op("fd_close", `{"fd":0}`)}
		case "transcript-bytes":
			// The stream header plus one record header need 24 bytes.
			config.Limits.TranscriptBytes = 23
			operations = []namespaceConformanceOperation{op("fd_close", `{"fd":999}`)}
		case "pending-events":
			config.Limits.PendingEvents = 1
			operations = []namespaceConformanceOperation{op("poll_oneoff", `{"subscriptions":[{"userdata":0,"type":1,"fd":0},{"userdata":1,"type":2,"fd":1}]}`)}
		}
		scripts = append(scripts, namespaceConformanceScript{"capacity-" + capacity, "handwritten", config, operations})
	}
	fieldsFirst := testConfig()
	fieldsFirst.Limits.TranscriptBytes = 100
	scripts = append(scripts, namespaceConformanceScript{"invalid-fields-before-transcript", "handwritten", fieldsFirst, []namespaceConformanceOperation{op("fd_read", `{"fd":0}`)}})
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "input"), []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/fixtures"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	readonly := testConfig()
	readonly.CapturedInputs = inputs
	scripts = append(scripts, namespaceConformanceScript{"readonly", "handwritten", readonly, []namespaceConformanceOperation{
		open(3, "input", 0, rightsAll&^rightWrite, 0, 0), op("fd_read", `{"fd":6,"length":20}`), op("fd_filestat_set_size", `{"fd":6,"size":0}`), open(3, "input", 0, rightWrite, 0, 0), open(3, "new", 1, rightRead, 0, 0),
		pathOp("path_create_directory", 3, "new"), pathOp("path_unlink_file", 3, "input"), rename("input", "other"),
	}})
	scripts = append(scripts, namespaceConformanceScript{"detached-directory", "handwritten", base, []namespaceConformanceOperation{
		pathOp("path_create_directory", 3, "a"), pathOp("path_create_directory", 3, "b"), open(3, "a", 2, rightsAll, rightsAll, 0), rename("b", "a"), open(5, "child", 1, rightsAll, 0, 0), op("fd_write", `{"fd":6,"data_base64":"ZGV0YWNoZWQ="}`), op("fd_close", `{"fd":6}`),
	}})
	samplePath := "../../backends/wasm/backend/testdata/premove-stock-artifact/backend/evidence.bin"
	sample, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	var recorded []namespaceConformanceOperation
	scanner := bufio.NewScanner(bytes.NewReader(sample))
	scanner.Buffer(make([]byte, 4096), frameLimit)
	for scanner.Scan() {
		var record struct {
			Type  string          `json:"type"`
			Op    string          `json:"op"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		if record.Type == "call" && (strings.HasPrefix(record.Op, "fd_") || strings.HasPrefix(record.Op, "path_")) {
			recorded = append(recorded, namespaceConformanceOperation{Op: record.Op, Input: record.Input})
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(recorded) == 0 {
		t.Fatal("retained WASM artifact contains no filesystem calls")
	}
	sampleDigest := sha256.Sum256(sample)
	scripts = append(scripts, namespaceConformanceScript{"recorded-stock-guest", "filesystem calls extracted unchanged from backend/testdata/premove-stock-artifact/backend/evidence.bin sha256:" + hex.EncodeToString(sampleDigest[:]), base, recorded})
	return append(scripts, namespaceR3ConformanceScripts(t)...)
}

// namespaceConformanceGenerated returns one seeded session: its configuration
// and 128 operations over a small depth-4 path alphabet. Every fourth seed
// lowers one of the file, descriptor or filesystem-byte limits so generated
// runs reach those capacities; writes reach 64 KiB and fds span every
// descriptor slot.
func namespaceConformanceGenerated(seed uint64) (Config, []namespaceConformanceOperation) {
	random := rand.New(rand.NewPCG(seed, seed+1))
	config := testConfig()
	switch seed % 4 {
	case 1:
		config.Limits.Files = 8
	case 2:
		config.Limits.Descriptors = 8
	case 3:
		config.Limits.FilesystemBytes = 16 << 10
		config.Limits.OutputBytes = 8 << 10
	}
	components := []string{"a", "b", "c", "d"}
	// Parents come mostly from the seeded a/b/c chain, so paths up to depth
	// four resolve; the leaf is drawn from the same alphabet.
	path := func() string {
		prefix := []string{"", "a/", "a/b/", "a/b/c/"}[random.IntN(4)]
		if random.IntN(8) == 0 {
			prefix = components[random.IntN(len(components))] + "/" + prefix
		}
		return prefix + components[random.IntN(len(components))]
	}
	size := func() int {
		switch random.IntN(16) {
		case 0:
			return random.IntN(64<<10 + 1)
		case 1, 2:
			return random.IntN(4 << 10)
		default:
			return random.IntN(64)
		}
	}
	// Opened descriptors start at 5 after the two preopens; occasionally draw
	// standard streams or any slot up to just past the descriptor limit.
	fd := func() uint32 {
		switch random.IntN(8) {
		case 0:
			return uint32(random.IntN(3))
		case 1:
			return uint32(random.Uint64N(config.Limits.Descriptors + 2))
		default:
			return 5 + uint32(random.IntN(6))
		}
	}
	pathInput := func(fd uint32, name string) any {
		return struct {
			FD   uint32 `json:"fd"`
			Path []byte `json:"path_base64"`
		}{fd, []byte(name)}
	}
	operations := make([]namespaceConformanceOperation, 0, 128)
	// Seed a directory chain so deeper paths resolve.
	for _, name := range []string{"a", "a/b", "a/b/c"} {
		data, err := json.Marshal(pathInput(3, name))
		if err != nil {
			panic(err)
		}
		operations = append(operations, namespaceConformanceOperation{Op: "path_create_directory", Input: data})
	}
	for len(operations) < 128 {
		var operation namespaceConformanceOperation
		var input any
		switch random.IntN(22) {
		case 0, 1, 20, 21:
			operation.Op = "path_open"
			oflags := []uint16{0, 1, 1 | 4, 1 | 8, 2, 8}[random.IntN(6)]
			rights := []uint64{rightsAll, 2, 64, 2 | 64}[random.IntN(4)]
			input = openInput{3, []byte(path()), 0, oflags, rights, rightsAll, uint16(random.IntN(2))}
		case 2:
			operation.Op = "fd_read"
			input = fdLengthInput{fd(), uint32(size())}
		case 3:
			operation.Op = "fd_pread"
			input = struct {
				FD     uint32 `json:"fd"`
				Length uint32 `json:"length"`
				Offset uint64 `json:"offset"`
			}{fd(), uint32(size()), random.Uint64N(80 << 10)}
		case 4, 5:
			operation.Op = "fd_write"
			input = struct {
				FD   uint32 `json:"fd"`
				Data []byte `json:"data_base64"`
			}{fd(), bytes.Repeat([]byte{byte('a' + random.IntN(26))}, size())}
		case 6:
			operation.Op = "fd_pwrite"
			input = struct {
				FD     uint32 `json:"fd"`
				Data   []byte `json:"data_base64"`
				Offset uint64 `json:"offset"`
			}{fd(), bytes.Repeat([]byte{'p'}, size()), random.Uint64N(80 << 10)}
		case 7:
			operation.Op = "fd_seek"
			input = struct {
				FD     uint32 `json:"fd"`
				Offset int64  `json:"offset"`
				Whence uint8  `json:"whence"`
			}{fd(), int64(random.IntN(256)) - 128, uint8(random.IntN(4))}
		case 8:
			operation.Op = "fd_filestat_set_size"
			input = struct {
				FD   uint32 `json:"fd"`
				Size uint64 `json:"size"`
			}{fd(), uint64(size())}
		case 9:
			operation.Op = "path_unlink_file"
			input = pathInput(3, path())
		case 10:
			operation.Op = "fd_close"
			input = fdInput{fd()}
		case 11:
			operation.Op = []string{"fd_fdstat_get", "fd_filestat_get", "fd_prestat_get", "fd_sync"}[random.IntN(4)]
			input = fdInput{fd()}
		case 12:
			operation.Op = "fd_readdir"
			input = struct {
				FD     uint32 `json:"fd"`
				Length uint32 `json:"length"`
				Cookie uint64 `json:"cookie"`
			}{fd(), uint32(random.IntN(256)), random.Uint64N(10)}
		case 13:
			operation.Op = "path_create_directory"
			input = pathInput(3, path())
		case 14:
			operation.Op = "path_remove_directory"
			input = pathInput(3, path())
		case 15:
			operation.Op = "path_rename"
			input = struct {
				FD      uint32 `json:"fd"`
				Path    []byte `json:"path_base64"`
				NewFD   uint32 `json:"new_fd"`
				NewPath []byte `json:"new_path_base64"`
			}{3, []byte(path()), 3, []byte(path())}
		case 16:
			operation.Op = "path_filestat_get"
			input = struct {
				FD    uint32 `json:"fd"`
				Flags uint32 `json:"flags"`
				Path  []byte `json:"path_base64"`
			}{3, uint32(random.IntN(2)), []byte(path())}
		case 17:
			operation.Op = "fd_fdstat_set_flags"
			input = struct {
				FD    uint32 `json:"fd"`
				Flags uint16 `json:"flags"`
			}{fd(), []uint16{0, 1, 4, 5}[random.IntN(4)]}
		case 18:
			operation.Op = "poll_oneoff"
			type subscription struct {
				Userdata  uint64  `json:"userdata"`
				Type      uint8   `json:"type"`
				ClockID   *uint32 `json:"clock_id,omitempty"`
				Timeout   *uint64 `json:"timeout,omitempty"`
				Precision *uint64 `json:"precision,omitempty"`
				Flags     *uint16 `json:"flags,omitempty"`
				FD        *uint32 `json:"fd,omitempty"`
			}
			var subscriptions []subscription
			for index := range 1 + random.IntN(3) {
				kind := uint8(random.IntN(3))
				sub := subscription{Userdata: uint64(index), Type: kind}
				if kind == 0 {
					clock, timeout, precision, flags := uint32(random.IntN(2)), random.Uint64N(5000), uint64(0), uint16(0)
					sub.ClockID, sub.Timeout, sub.Precision, sub.Flags = &clock, &timeout, &precision, &flags
				} else {
					descriptor := fd()
					sub.FD = &descriptor
				}
				subscriptions = append(subscriptions, sub)
			}
			input = struct {
				Subscriptions []subscription `json:"subscriptions"`
			}{subscriptions}
		case 19:
			operation.Op = "clock_time_get"
			input = struct {
				ClockID   uint32 `json:"clock_id"`
				Precision uint64 `json:"precision"`
			}{uint32(random.IntN(2)), 0}
		}
		data, err := json.Marshal(input)
		if err != nil {
			panic(err)
		}
		operation.Input = data
		operations = append(operations, operation)
	}
	return config, operations
}

func TestNamespaceFilesystemCounterResultDoesNotAliasModel(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	call(t, e, "fd_read", `{"fd":99,"length":1}`)
	before := e.filesystemServed()["fd_read"]
	got := e.filesystemServed()
	got["fd_read"]++
	if e.filesystemServed()["fd_read"] != before {
		t.Fatal("result counter aliases model state")
	}
}

func TestNamespaceOwnedCallbackRefusedBeforeAdoptingRemoteState(t *testing.T) {
	for _, op := range []struct{ name, input string }{
		{"fd_close", `{"fd":3}`}, {"path_create_directory", `{"fd":3,"path_base64":"eA=="}`},
		{"poll_oneoff", `{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":1,"precision":0,"flags":0}]}`},
	} {
		t.Run(op.name, func(t *testing.T) {
			e := newTestEnvironment(t, testConfig())
			e.namespaceOwned = true
			before := namespaceConformanceDigest(e)
			owned := validatedCall{call: Call{Type: "call", ID: 3, Op: op.name, Input: json.RawMessage(op.input)}, input: &validatedInput{data: json.RawMessage(op.input)}}
			_, err := e.helperCallbackReply(owned, helperLocalState{Now: 2, Advances: 1, TranscriptBytes: 8})
			if err == nil || e.nextCall != 1 || e.now != 0 || e.advances != 0 || e.transcriptLength != 0 || before != namespaceConformanceDigest(e) || len(e.filesystemServed()) != 0 {
				t.Fatal("helper-owned filesystem callback committed state or was accepted", err)
			}
		})
	}
}

func TestNamespaceUnsupportedTranscriptAdmissionCounters(t *testing.T) {
	for _, test := range []struct {
		name, input string
		limit       uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, 30},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, 61},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, 41},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			config.Limits.TranscriptBytes = test.limit
			e := newTestEnvironment(t, config)
			_, err := e.Handle(Call{Type: "call", ID: 1, Op: test.name, Input: json.RawMessage(test.input)})
			if err == nil {
				t.Fatal("expected refusal before checking admission counters")
			}
			if e.nextCall != 1 || len(e.Transcript()) != 0 || len(e.filesystemServed()) != 0 {
				t.Fatalf("unsupported admission=%v next=%d transcript=%d limit=%d", err, e.nextCall, len(e.Transcript()), test.limit)
			}
		})
	}
}

func TestNamespaceUnsupportedTranscriptExactAdmissionCounters(t *testing.T) {
	for _, test := range []struct {
		op, input, supported string
		limit                uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, `{"fd":3,"flags":1}`, 75},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, 96},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, "", 92},
	} {
		t.Run(test.op, func(t *testing.T) {
			inputs := []string{test.input}
			if test.supported != "" {
				inputs = append(inputs, test.supported)
			}
			for _, input := range inputs {
				config := testConfig()
				config.Limits.TranscriptBytes = test.limit - 1
				environment := newTestEnvironment(t, config)
				_, err := environment.Handle(Call{Type: "call", ID: 1, Op: test.op, Input: json.RawMessage(input)})
				if err == nil {
					t.Fatal("expected refusal before checking admission counters")
				}
				if environment.nextCall != 1 || len(environment.Transcript()) != 0 || len(environment.filesystemServed()) != 0 {
					t.Fatal("refused exact-bound call changed admission counters")
				}
				config.Limits.TranscriptBytes = test.limit
				environment = newTestEnvironment(t, config)
				call(t, environment, test.op, input)
				if environment.nextCall != 2 || environment.filesystemServed()[test.op] != 1 {
					t.Fatal("admitted exact-bound call changed admission counters")
				}
			}
		})
	}
}

func TestNamespaceHelperOutputRefusesInconsistentOwnership(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	e.helperOwned, e.namespaceOwned = true, false
	sink := &frameSink{environment: e, started: &Started{}, request: Request{Config: testConfig()}}
	frame, err := newValidatedFrame([]byte(`{"type":"evidence","stream":"stdout","data_base64":"eg=="}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.receiveHelperEvidence(frame); err == nil {
		t.Fatal("remote namespace accepted helper output")
	}
}

func TestNamespacePartialEvidenceResultDoesNotAliasTransport(t *testing.T) {
	sink := &frameSink{helperEvidence: helperEvidence{transcript: []byte("GMWT\x01\x00\x00\x00\x08")}}
	result := Result{Termination: "infrastructure"}
	sink.retainPartialHelperEvidence(&result)
	result.PartialHelperTranscript[0] = 'x'
	if sink.helperEvidence.transcript[0] != 'G' {
		t.Fatal("partial result aliases transport evidence")
	}
}
