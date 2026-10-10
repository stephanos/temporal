package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
)

func testConfig() Config {
	return Config{Profile: StockProfile, Args: []string{"guest", "literal $arg"}, Environment: []string{"PWD=/workspace", "TZ=UTC", "ONLY=captured"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
}

func openPath(t *testing.T, e *Environment, fd uint32, name string, oflags uint16, rights uint64) uint32 {
	t.Helper()
	input, _ := json.Marshal(struct {
		FD       uint32 `json:"fd"`
		Path     []byte `json:"path_base64"`
		Dirflags uint32 `json:"dirflags"`
		Oflags   uint16 `json:"oflags"`
		Base     uint64 `json:"rights_base"`
		Inherit  uint64 `json:"rights_inheriting"`
		Flags    uint16 `json:"fdflags"`
	}{fd, []byte(name), 0, oflags, rights, 0, 0})
	var result struct {
		FD uint32 `json:"fd"`
	}
	output(t, call(t, e, "path_open", string(input)), &result)
	return result.FD
}

func TestNamespaceRightsOffsetsDirectoryCookiesAndClosure(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	var pre struct {
		Length uint32 `json:"name_length"`
	}
	output(t, call(t, e, "fd_prestat_get", `{"fd":3}`), &pre)
	var name bytesOutput
	output(t, call(t, e, "fd_prestat_dir_name", fmt.Sprintf(`{"fd":3,"length":%d}`, pre.Length)), &name)
	if string(name.Data) != "/tmp" {
		t.Fatalf("first preopen = %q", name.Data)
	}
	if call(t, e, "fd_prestat_get", `{"fd":5}`).Errno != errnoBadf {
		t.Fatal("preopen sequence has no boundary")
	}
	fd := openPath(t, e, 3, "b", 1, rightsAll)
	var written writtenOutput
	output(t, call(t, e, "fd_pwrite", fmt.Sprintf(`{"fd":%d,"offset":2,"data_base64":"eHl6"}`, fd)), &written)
	var read bytesOutput
	output(t, call(t, e, "fd_read", fmt.Sprintf(`{"fd":%d,"length":20}`, fd)), &read)
	if !bytes.Equal(read.Data, []byte{0, 0, 'x', 'y', 'z'}) || written.Written != 3 {
		t.Fatalf("partial read/write = %q %#v", read.Data, written)
	}
	output(t, call(t, e, "fd_read", fmt.Sprintf(`{"fd":%d,"length":20}`, fd)), &read)
	if len(read.Data) != 0 {
		t.Fatal("EOF is not empty success")
	}
	var offset offsetOutput
	output(t, call(t, e, "fd_seek", fmt.Sprintf(`{"fd":%d,"offset":-2,"whence":2}`, fd)), &offset)
	if offset.Offset != 3 {
		t.Fatalf("seek = %d", offset.Offset)
	}
	if call(t, e, "fd_seek", fmt.Sprintf(`{"fd":%d,"offset":-6,"whence":2}`, fd)).Errno != errnoInval {
		t.Fatal("negative seek accepted")
	}
	readonly := openPath(t, e, 3, "b", 0, rightRead)
	if call(t, e, "fd_write", fmt.Sprintf(`{"fd":%d,"data_base64":"eA=="}`, readonly)).Errno != errnoNotcapable {
		t.Fatal("write bypasses descriptor rights")
	}
	openPath(t, e, 3, "a", 1, rightRead)
	var dirs bytesOutput
	output(t, call(t, e, "fd_readdir", `{"fd":3,"length":25,"cookie":0}`), &dirs)
	if len(dirs.Data) != 25 || string(dirs.Data[24:]) != "a" || binary.LittleEndian.Uint64(dirs.Data[:8]) != 1 {
		t.Fatalf("first dirent = %x", dirs.Data)
	}
	output(t, call(t, e, "fd_readdir", `{"fd":3,"length":100,"cookie":1}`), &dirs)
	if string(dirs.Data[24:]) != "b" || binary.LittleEndian.Uint64(dirs.Data[:8]) != 2 {
		t.Fatalf("cookie = %x", dirs.Data)
	}
	if call(t, e, "fd_readdir", `{"fd":3,"length":1,"cookie":9}`).Errno != errnoInval {
		t.Fatal("invalid cookie accepted")
	}
	for _, name := range []string{"../escape", "/absolute", "a/../../escape", "bad\x00name"} {
		input, _ := json.Marshal(map[string]any{"fd": uint32(3), "path_base64": []byte(name)})
		if call(t, e, "path_create_directory", string(input)).Errno != errnoNotcapable {
			t.Fatalf("path escape %q accepted", name)
		}
	}
	output(t, call(t, e, "fd_close", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	if call(t, e, "fd_close", fmt.Sprintf(`{"fd":%d}`, fd)).Errno != errnoBadf {
		t.Fatal("double close accepted")
	}
	if call(t, e, "sock_accept", `{"fd":3,"flags":0}`).Unsupported != true {
		t.Fatal("live socket operation accepted")
	}
	if call(t, e, "path_symlink", `{"old_path_base64":"eA==","fd":3,"path_base64":"eQ=="}`).Unsupported != true {
		t.Fatal("symlink operation accepted")
	}
}

func TestNamespaceRenameRejectsMissingDirectoryDestinationWithoutMutation(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	fd := openPath(t, e, 3, "source", 1, rightsAll)
	output(t, call(t, e, "fd_write", fmt.Sprintf(`{"fd":%d,"data_base64":"c3RhYmxl"}`, fd)), &writtenOutput{})
	source := e.namespace.lookup("/tmp/source")
	input := `{"fd":3,"path_base64":"c291cmNl","new_fd":3,"new_path_base64":"bWlzc2luZy8="}`
	if call(t, e, "path_rename", input).Errno != errnoNotdir || e.namespace.lookup("/tmp/source") != source || e.namespace.lookup("/tmp/missing") != nil || string(source.data) != "stable" {
		t.Fatal("rename to missing directory requirement mutated source/destination")
	}
	output(t, call(t, e, "path_rename", `{"fd":3,"path_base64":"c291cmNl","new_fd":3,"new_path_base64":"bWlzc2luZw=="}`), &struct{}{})
	if e.namespace.lookup("/tmp/source") != nil || e.namespace.lookup("/tmp/missing") != source || e.namespace.descriptors[fd].node != source {
		t.Fatal("ordinary rename did not preserve open descriptor")
	}
}

func TestCapturedNamespaceNeverReopensHostAndInfersAbsence(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "input"), []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: source, Target: "/fixtures"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig()
	config.CapturedInputs = captured
	e := newTestEnvironment(t, config)
	for _, payload := range captured.Payloads {
		clear(payload)
	}
	clear(captured.Descriptor)
	clear(captured.Manifest.Mappings)
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	fd := openPath(t, e, 3, "input", 0, rightRead)
	var data bytesOutput
	output(t, call(t, e, "fd_read", fmt.Sprintf(`{"fd":%d,"length":20}`, fd)), &data)
	if string(data.Data) != "frozen" {
		t.Fatalf("read = %q", data.Data)
	}
	if call(t, e, "path_filestat_get", `{"fd":3,"flags":0,"path_base64":"bWlzc2luZw=="}`).Errno != errnoNoent {
		t.Fatal("frozen inventory absence not modeled")
	}
	if call(t, e, "path_create_directory", `{"fd":3,"path_base64":"bmV3"}`).Errno != errnoRofs {
		t.Fatal("capture is writable")
	}
}

func TestCapturedNamespaceAllowsStockReadOnlyOpenWithoutMutation(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "input"), []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: source, Target: "/fixtures"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig()
	config.CapturedInputs = captured
	e := newTestEnvironment(t, config)
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	var opened fdOutput
	output(t, call(t, e, "path_open", `{"fd":3,"path_base64":"aW5wdXQ=","dirflags":1,"oflags":0,"rights_base":267910846,"rights_inheriting":268435455,"fdflags":0}`), &opened)
	var rights fdstat
	output(t, call(t, e, "fd_fdstat_get", fmt.Sprintf(`{"fd":%d}`, opened.FD)), &rights)
	if rights.Base != 267910846 || rights.Inheriting != 268435455 {
		t.Fatalf("stock open rights changed: %+v", rights)
	}
	var read bytesOutput
	output(t, call(t, e, "fd_read", fmt.Sprintf(`{"fd":%d,"length":20}`, opened.FD)), &read)
	if string(read.Data) != "frozen" {
		t.Fatalf("captured read = %q", read.Data)
	}
	var before filestat
	output(t, call(t, e, "fd_filestat_get", fmt.Sprintf(`{"fd":%d}`, opened.FD)), &before)
	if before.Size != 6 {
		t.Fatalf("captured stat = %+v", before)
	}
	files, usedBytes, nextFD := e.namespace.files, e.namespace.bytes, e.namespace.nextFD
	for _, test := range []struct {
		op, input string
		errno     uint16
	}{
		{"fd_filestat_set_size", fmt.Sprintf(`{"fd":%d,"size":0}`, opened.FD), errnoRofs},
		{"fd_filestat_set_size", fmt.Sprintf(`{"fd":%d,"size":20}`, opened.FD), errnoRofs},
		{"fd_write", fmt.Sprintf(`{"fd":%d,"data_base64":"eA=="}`, opened.FD), errnoNotcapable},
		{"fd_pwrite", fmt.Sprintf(`{"fd":%d,"offset":0,"data_base64":"eA=="}`, opened.FD), errnoNotcapable},
		{"path_open", `{"fd":3,"path_base64":"aW5wdXQ=","dirflags":1,"oflags":0,"rights_base":267910910,"rights_inheriting":268435455,"fdflags":0}`, errnoRofs},
		{"path_open", `{"fd":3,"path_base64":"bmV3","dirflags":1,"oflags":1,"rights_base":267910846,"rights_inheriting":268435455,"fdflags":0}`, errnoRofs},
		{"path_open", `{"fd":3,"path_base64":"aW5wdXQ=","dirflags":1,"oflags":8,"rights_base":267910846,"rights_inheriting":268435455,"fdflags":0}`, errnoRofs},
		{"path_create_directory", `{"fd":3,"path_base64":"bmV3"}`, errnoRofs},
		{"path_rename", `{"fd":3,"path_base64":"aW5wdXQ=","new_fd":3,"new_path_base64":"bmV3"}`, errnoRofs},
		{"path_unlink_file", `{"fd":3,"path_base64":"aW5wdXQ="}`, errnoRofs},
	} {
		if reply := call(t, e, test.op, test.input); reply.Errno != test.errno {
			t.Fatalf("%s %s: errno = %d, want %d", test.op, test.input, reply.Errno, test.errno)
		}
		n := e.namespace.lookup("/fixtures/input")
		if n == nil || !bytes.Equal(n.data, []byte("frozen")) || e.stat(n) != before || len(e.namespace.lookup("/fixtures").children) != 1 || e.namespace.files != files || e.namespace.bytes != usedBytes || e.namespace.nextFD != nextFD {
			t.Fatalf("%s changed the immutable captured namespace", test.op)
		}
	}
}

func TestCapturedNamespaceRejectsWritableDescendantAndAllowsAncestor(t *testing.T) {
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: t.TempDir(), Target: "/workspace/frozen"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	config := testConfig()
	config.CapturedInputs = captured
	e := newTestEnvironment(t, config)
	if !e.namespace.lookup("/workspace/frozen").readonly || e.namespace.lookup("/workspace").readonly {
		t.Fatal("legal writable ancestor changed captured mount")
	}
	for _, cwd := range []string{"/workspace", "/workspace/frozen/new"} {
		config.WorkingDirectory = cwd
		config.Environment[0] = "PWD=" + cwd
		config.WritableDirectories = []string{"/tmp", "/workspace", "/workspace/frozen/new"}
		if _, err := NewEnvironment(config); err == nil {
			t.Fatalf("writable descendant of captured inventory accepted: cwd=%q", cwd)
		}
	}
	if len(e.namespace.lookup("/workspace/frozen").children) != 0 {
		t.Fatal("rejected configuration changed earlier capture")
	}
}

func newTestEnvironment(t *testing.T, config Config) *Environment {
	t.Helper()
	environment, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	return environment
}

func call(t *testing.T, environment *Environment, op, input string) Reply {
	t.Helper()
	reply, err := environment.Handle(Call{Type: "call", ID: environment.nextCall, Op: op, Input: json.RawMessage(input)})
	if err != nil {
		t.Fatal(err)
	}
	return reply
}

func output(t *testing.T, reply Reply, target any) {
	t.Helper()
	if reply.Errno != 0 || reply.Unsupported {
		t.Fatalf("reply = %#v", reply)
	}
	if err := json.Unmarshal(reply.Output, target); err != nil {
		t.Fatal(err)
	}
}

func TestCapturedArgumentsEnvironmentAndEntropy(t *testing.T) {
	config := testConfig()
	environment := newTestEnvironment(t, config)
	config.Args[0] = "mutated"
	var args valuesOutput
	output(t, call(t, environment, "args_get", `{}`), &args)
	if len(args.Values) != 2 || string(args.Values[0]) != "guest" || string(args.Values[1]) != "literal $arg" {
		t.Fatalf("args = %q", args.Values)
	}
	var sizes sizesOutput
	output(t, call(t, environment, "args_sizes_get", `{}`), &sizes)
	if sizes.Count != 2 || sizes.Bytes != 19 {
		t.Fatalf("sizes = %#v", sizes)
	}
	var env valuesOutput
	output(t, call(t, environment, "environ_get", `{}`), &env)
	if len(env.Values) != 3 || string(env.Values[2]) != "ONLY=captured" {
		t.Fatalf("environment = %q", env.Values)
	}
	var first, second, whole bytesOutput
	output(t, call(t, environment, "random_get", `{"length":7}`), &first)
	output(t, call(t, environment, "random_get", `{"length":57}`), &second)
	fresh := newTestEnvironment(t, testConfig())
	output(t, call(t, fresh, "random_get", `{"length":64}`), &whole)
	if !bytes.Equal(append(first.Data, second.Data...), whole.Data) || bytes.Equal(first.Data, second.Data[:7]) {
		t.Fatalf("entropy is restarted or chunk-dependent: %x %x %x", first.Data, second.Data, whole.Data)
	}
	if environment.Identity() != fresh.Identity() {
		t.Fatal("caller mutation changed immutable input identity")
	}
}

func TestEnvironmentRejectsAmbiguousInputsAndMalformedCalls(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"pwd":       func(c *Config) { c.WorkingDirectory = "/other" },
		"duplicate": func(c *Config) { c.Environment = append(c.Environment, "ONLY=again") },
		"nul":       func(c *Config) { c.Args[0] = "bad\x00arg" },
		"limits":    func(c *Config) { c.Limits.Descriptors = 0 },
		"profile":   func(c *Config) { c.Profile = "strict" },
	} {
		t.Run(name, func(t *testing.T) {
			c := testConfig()
			mutate(&c)
			if _, err := NewEnvironment(c); err == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
	for _, input := range []string{`{}`, `{"length":-1}`, `{"length":1.0}`, `{"length":1,"extra":2}`, `{"length":1,"length":2}`, `{"length":1} {}`} {
		environment := newTestEnvironment(t, testConfig())
		if _, err := environment.Handle(Call{Type: "call", ID: 1, Op: "random_get", Input: json.RawMessage(input)}); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}

func TestEnvironmentRejectsInvalidUTF8IdentityInputs(t *testing.T) {
	for _, invalidByte := range []byte{0xff, 0xfe} {
		invalidText := string([]byte{invalidByte})
		for name, mutate := range map[string]func(*Config){
			"argument":    func(c *Config) { c.Args[1] = invalidText },
			"environment": func(c *Config) { c.Environment[2] = "ONLY=" + invalidText },
			"cwd": func(c *Config) {
				c.WorkingDirectory += "/" + invalidText
				c.Environment[0] = "PWD=" + c.WorkingDirectory
				c.WritableDirectories[0] = c.WorkingDirectory
			},
			"writable":         func(c *Config) { c.WritableDirectories[1] += "/" + invalidText },
			"manifest-schema":  func(c *Config) { c.CapturedInputs.Manifest.Schema = invalidText },
			"manifest-file":    func(c *Config) { c.CapturedInputs.Manifest.File = invalidText },
			"manifest-sha":     func(c *Config) { c.CapturedInputs.Manifest.SHA256 = record.SHA256("sha256:" + invalidText) },
			"manifest-mapping": func(c *Config) { c.CapturedInputs.Manifest.Mappings = []string{"/" + invalidText} },
		} {
			t.Run(fmt.Sprintf("%s/%x", name, invalidByte), func(t *testing.T) {
				config := testConfig()
				mutate(&config)
				if environment, err := NewEnvironment(config); err == nil {
					t.Fatalf("accepted invalid UTF-8 identity input: identity=%s", environment.Identity())
				}
			})
		}
	}
}

func TestEnvironmentPreservesMultibyteConfigAndRawRuntimeFilenames(t *testing.T) {
	config := testConfig()
	config.Args[1] = "雪"
	config.WorkingDirectory = "/workspace/雪"
	config.WritableDirectories[0] = config.WorkingDirectory
	config.Environment[0] = "PWD=" + config.WorkingDirectory
	config.Environment[2] = "ONLY=café"
	environment := newTestEnvironment(t, config)
	var args, env valuesOutput
	output(t, call(t, environment, "args_get", `{}`), &args)
	output(t, call(t, environment, "environ_get", `{}`), &env)
	if string(args.Values[1]) != "雪" || string(env.Values[2]) != "ONLY=café" {
		t.Fatalf("multibyte inputs changed: args=%q env=%q", args.Values, env.Values)
	}
	name := string([]byte{0xff, 0xfe})
	fd := openPath(t, environment, 3, name, 1, rightsAll)
	output(t, call(t, environment, "fd_write", fmt.Sprintf(`{"fd":%d,"data_base64":"cmF3"}`, fd)), &writtenOutput{})
	fd = openPath(t, environment, 3, name, 0, rightRead)
	var data bytesOutput
	output(t, call(t, environment, "fd_read", fmt.Sprintf(`{"fd":%d,"length":3}`, fd)), &data)
	if string(data.Data) != "raw" {
		t.Fatalf("raw filename read = %q", data.Data)
	}
	output(t, call(t, environment, "fd_readdir", `{"fd":3,"length":100,"cookie":0}`), &data)
	if !bytes.Equal(data.Data[24:], []byte(name)) {
		t.Fatalf("raw filename dirent = %x", data.Data)
	}
}

func TestExploratoryClockAndPollDoNotUseWallTimeOrYieldTicks(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	var wall, mono timestampOutput
	output(t, call(t, e, "clock_time_get", `{"clock_id":0,"precision":0}`), &wall)
	output(t, call(t, e, "clock_time_get", `{"clock_id":1,"precision":0}`), &mono)
	if wall.Timestamp != 946684800000000000 || mono.Timestamp != 1000 || e.now != 2000 {
		t.Fatalf("clock pair = %#v %#v now %d", wall, mono, e.now)
	}
	output(t, call(t, e, "sched_yield", `{}`), &struct{}{})
	var events eventsOutput
	output(t, call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":7,"type":0,"clock_id":1,"timeout":0,"precision":0,"flags":0},{"userdata":7,"type":2,"fd":1}]}`), &events)
	if e.now != 2000 || len(events.Events) != 2 || events.Events[0].Type != 0 || events.Events[1].Type != 2 {
		t.Fatalf("nonblocking duplicate-userdata poll = %#v now %d", events, e.now)
	}
	output(t, call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":8,"type":0,"clock_id":1,"timeout":3000,"precision":1,"flags":0},{"userdata":9,"type":0,"clock_id":0,"timeout":946684800000005000,"precision":0,"flags":1}]}`), &events)
	if e.now != 5000 || len(events.Events) != 2 || events.Events[0].Userdata != 8 || events.Events[1].Userdata != 9 {
		t.Fatalf("relative/absolute tie = %#v now %d", events, e.now)
	}
	output(t, call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":1,"type":1,"fd":999}]}`), &events)
	if e.now != 5000 || len(events.Events) != 1 || events.Events[0].Errno != errnoBadf {
		t.Fatalf("bad fd poll = %#v", events)
	}
	if call(t, e, "poll_oneoff", `{"subscriptions":[{"userdata":0,"type":0,"clock_id":1,"timeout":18446744073709551615,"precision":0,"flags":0}]}`).Errno != errnoInval || e.now != 5000 {
		t.Fatal("overflow poll mutates clock")
	}
	if !call(t, e, "clock_time_get", `{"clock_id":2,"precision":0}`).Unsupported {
		t.Fatal("host CPU clock accepted")
	}
}

func TestCapacityRejectsMutationAndBoundsAllGrowingState(t *testing.T) {
	for _, dimension := range []string{"filesystem", "descriptor", "file", "output", "transcript", "calls", "events"} {
		t.Run(dimension, func(t *testing.T) {
			config := testConfig()
			switch dimension {
			case "filesystem":
				config.Limits.FilesystemBytes = 2
			case "descriptor":
				config.Limits.Descriptors = 5
			case "file":
				config.Limits.Files = 3
			case "output":
				config.Limits.OutputBytes = 2
			case "transcript":
				config.Limits.TranscriptBytes = 100
			case "calls":
				config.Limits.Calls = 1
			case "events":
				config.Limits.PendingEvents = 1
			}
			e := newTestEnvironment(t, config)
			op, input := "path_open", `{"fd":3,"path_base64":"eA==","dirflags":0,"oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`
			switch dimension {
			case "filesystem":
				fd := openPath(t, e, 3, "file", 1, rightsAll)
				op = "fd_write"
				input = fmt.Sprintf(`{"fd":%d,"data_base64":"eHl6"}`, fd)
			case "output":
				op = "fd_write"
				input = `{"fd":1,"data_base64":"eHl6"}`
			case "transcript":
				op = "random_get"
				input = `{"length":1}`
			case "calls":
				call(t, e, "sched_yield", `{}`)
				op = "sched_yield"
				input = `{}`
			case "events":
				op = "poll_oneoff"
				input = `{"subscriptions":[{"userdata":0,"type":1,"fd":0},{"userdata":1,"type":2,"fd":1}]}`
			}
			before := e.Transcript()
			fdCount := len(e.namespace.descriptors)
			fileCount := e.namespace.files
			byteCount := e.namespace.bytes
			entropy := e.entropyOffset
			_, err := e.Handle(Call{Type: "call", ID: e.nextCall, Op: op, Input: json.RawMessage(input)})
			boundary, ok := err.(*BoundaryError)
			if !ok || boundary.Kind != "capacity" || !bytes.Equal(e.Transcript(), before) || fdCount != len(e.namespace.descriptors) || fileCount != e.namespace.files || byteCount != e.namespace.bytes || entropy != e.entropyOffset {
				t.Fatalf("capacity failure changed state: %v", err)
			}
		})
	}
}

func TestPathOpenRequiresCreateRightAndPreservesDirectoryRequirement(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	var opened fdOutput
	output(t, call(t, e, "path_open", `{"fd":3,"path_base64":"Lg==","dirflags":0,"oflags":2,"rights_base":8192,"rights_inheriting":536870911,"fdflags":0}`), &opened)
	dir := opened.FD
	input := fmt.Sprintf(`{"fd":%d,"path_base64":"eA==","dirflags":0,"oflags":1,"rights_base":2,"rights_inheriting":0,"fdflags":0}`, dir)
	if call(t, e, "path_open", input).Errno != errnoNotcapable || e.namespace.lookup("/tmp/x") != nil {
		t.Fatal("create bypassed directory create rights")
	}
	if call(t, e, "path_open", `{"fd":3,"path_base64":"eA==","dirflags":0,"oflags":3,"rights_base":2,"rights_inheriting":0,"fdflags":0}`).Errno != errnoInval || e.namespace.lookup("/tmp/x") != nil {
		t.Fatal("create-directory open created regular file")
	}
	openPath(t, e, 3, "file", 1, rightRead)
	for _, name := range []string{"file/", "file/../x", "file/."} {
		input, err := json.Marshal(map[string]any{"fd": 3, "flags": 0, "path_base64": []byte(name)})
		if err != nil {
			t.Fatal(err)
		}
		if call(t, e, "path_filestat_get", string(input)).Errno != errnoNotdir {
			t.Fatalf("non-directory traversal %q accepted", name)
		}
	}
}
func TestOfferedOutputChargesRejectedWritesAndEntropyBufferReservation(t *testing.T) {
	config := testConfig()
	config.Limits.OutputBytes = 2
	e := newTestEnvironment(t, config)
	output(t, call(t, e, "fd_close", `{"fd":1}`), &struct{}{})
	if call(t, e, "fd_write", `{"fd":1,"data_base64":"eHk="}`).Errno != errnoBadf {
		t.Fatal("closed stdout accepted")
	}
	_, err := e.Handle(Call{Type: "call", ID: e.nextCall, Op: "fd_write", Input: json.RawMessage(`{"fd":1,"data_base64":"eA=="}`)})
	if boundary, ok := err.(*BoundaryError); !ok || boundary.Kind != "capacity" {
		t.Fatalf("rejected output not charged: %v", err)
	}
	fresh := newTestEnvironment(t, testConfig())
	var entropy bytesOutput
	output(t, call(t, fresh, "random_get", `{"length":8192}`), &entropy)
	if len(entropy.Data) != 8192 {
		t.Fatal("large response reservation lost entropy")
	}
}

func TestEnvironmentRejectsWireFrameCapacityBeforeRecording(t *testing.T) {
	e := newTestEnvironment(t, testConfig())
	_, err := e.Handle(Call{Type: "call", ID: 1, Op: strings.Repeat("x", frameLimit), Input: json.RawMessage(`{}`)})
	if boundary, ok := err.(*BoundaryError); !ok || boundary.Kind != "capacity" || e.nextCall != 1 || len(e.Transcript()) != 0 {
		t.Fatalf("call frame capacity = %v next=%d", err, e.nextCall)
	}
	config := testConfig()
	config.Args = make([]string, frameLimit/3+1)
	config.Limits.TranscriptBytes = 64 << 20
	e = newTestEnvironment(t, config)
	_, err = e.Handle(Call{Type: "call", ID: 1, Op: "args_get", Input: json.RawMessage(`{}`)})
	if boundary, ok := err.(*BoundaryError); !ok || boundary.Kind != "capacity" || e.nextCall != 1 || len(e.Transcript()) != 0 {
		t.Fatalf("reply frame capacity = %v next=%d", err, e.nextCall)
	}
}

func TestCapacityIncludesCapturedStdinInFilesystemBudget(t *testing.T) {
	config := testConfig()
	config.Stdin = []byte("full")
	config.Limits.FilesystemBytes = 4
	e := newTestEnvironment(t, config)
	fd := openPath(t, e, 3, "scratch", 1, rightWrite)
	input := json.RawMessage(fmt.Sprintf(`{"fd":%d,"data_base64":"eA=="}`, fd))
	_, err := e.Handle(Call{Type: "call", ID: e.nextCall, Op: "fd_write", Input: input})
	if boundary, ok := err.(*BoundaryError); !ok || boundary.Kind != "capacity" || len(e.namespace.lookup("/tmp/scratch").data) != 0 {
		t.Fatalf("captured stdin budget = %v", err)
	}
}
