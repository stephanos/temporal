package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func helperPath(t *testing.T) string {
	t.Helper()
	name, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); err != nil {
		t.Skip("build pinned release wasmhost to run guest gates: ", err)
	}
	return name
}
func fileHash(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
func fixtureRequest(t *testing.T, wat string, imports []Import) Request {
	t.Helper()
	name := filepath.Join(t.TempDir(), "guest.wat")
	if err := os.WriteFile(name, []byte(wat), 0600); err != nil {
		t.Fatal(err)
	}
	helper := helperPath(t)
	return Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: name, ModuleSHA256: fileHash(t, name), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: 1, MemoryBytes: 128 << 20, Fuel: 100000000, Config: testConfig(), Timeout: 30 * time.Second}
}
func TestGuestTransportAuthorizesIdentityBeforeStart(t *testing.T) {
	imports := []Import{{Module: "wasi_snapshot_preview1", Name: "fd_write", Params: []string{"i32", "i32", "i32", "i32"}, Results: []string{"i32"}}}
	request := fixtureRequest(t, `(module (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export "memory") 1) (data (i32.const 0) "hello") (data (i32.const 16) "\00\00\00\00\05\00\00\00") (func $init i32.const 1 i32.const 16 i32.const 1 i32.const 32 call $write drop) (start $init) (func (export "_start")))`, imports)
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || string(result.Stdout) != "hello" || !result.Reaped || len(result.Transcript) == 0 {
		t.Fatalf("result = %s: %s reaped=%v stdout=%q", result.Termination, result.Message, result.Reaped, result.Stdout)
	}
	request.Engine.Version = "different"
	result, err = Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != "invalid" || len(result.Stdout) != 0 || len(result.Transcript) != 0 || !result.Reaped {
		t.Fatalf("identity permitted guest start: %#v", result)
	}
	request.Engine = PinnedEngine()
	request.Imports = nil
	result, err = Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != "invalid" || len(result.Stdout) != 0 || len(result.Transcript) != 0 {
		t.Fatalf("import identity accepted: %#v", result)
	}
}
func TestGuestTransportClassifiesCapacityMalformedAndUnsupported(t *testing.T) {
	for _, fixture := range []struct {
		name, wat, want string
		imports         []Import
		output          uint64
	}{
		{"fuel", `(module (memory (export "memory") 1) (func (export "_start") (loop $again br $again)))`, "capacity", nil, 0},
		{"range", `(module (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32))) (memory (export "memory") 1) (func (export "_start") i32.const -1 i32.const 2 call $random drop))`, "trap", []Import{{"wasi_snapshot_preview1", "random_get", []string{"i32", "i32"}, []string{"i32"}}}, 0},
		{"socket", `(module (import "wasi_snapshot_preview1" "sock_accept" (func $accept (param i32 i32 i32) (result i32))) (memory (export "memory") 1) (func (export "_start") i32.const 3 i32.const 0 i32.const 0 call $accept drop))`, "unsupported", []Import{{"wasi_snapshot_preview1", "sock_accept", []string{"i32", "i32", "i32"}, []string{"i32"}}}, 0},
		{"output", `(module (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32))) (memory (export "memory") 1) (data (i32.const 0) "hello") (data (i32.const 16) "\00\00\00\00\05\00\00\00") (func (export "_start") i32.const 1 i32.const 16 i32.const 1 i32.const 32 call $write drop))`, "capacity", []Import{{"wasi_snapshot_preview1", "fd_write", []string{"i32", "i32", "i32", "i32"}, []string{"i32"}}}, 4},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			request := fixtureRequest(t, fixture.wat, fixture.imports)
			request.Fuel = 10000
			if fixture.output != 0 {
				request.Config.Limits.OutputBytes = fixture.output
			}
			result, err := Run(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Termination != fixture.want || !result.Reaped {
				t.Fatalf("result = %s: %s reaped=%v stdout=%q", result.Termination, result.Message, result.Reaped, result.Stdout)
			}
		})
	}
}

func TestTransportRejectsChangedFilesAndInvalidTerminalClaims(t *testing.T) {
	request := fixtureRequest(t, `(module (memory (export "memory") 1) (func (export "_start")))`, nil)
	request.ModuleSHA256 = strings.Repeat("0", 64)
	if _, err := Run(context.Background(), request); err == nil {
		t.Fatal("changed module bytes accepted")
	}
	request.ModuleSHA256 = fileHash(t, request.ModulePath)
	request.HelperSHA256 = strings.Repeat("0", 64)
	if _, err := Run(context.Background(), request); err == nil {
		t.Fatal("changed helper bytes accepted")
	}
	for _, fixture := range []struct{ name, terminal, want string }{
		{"invented-output", `{"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":0,"peak_memory_bytes":65536,"output_bytes":1}`, "invalid"},
		{"excess-fuel", `{"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":100000001,"peak_memory_bytes":65536,"output_bytes":0}`, "invalid"},
		{"missing-exit-code", `{"type":"result","termination":"exit","message":"","fuel_remaining":0,"peak_memory_bytes":65536,"output_bytes":0}`, "invalid"},
		{"trap-with-exit-code", `{"type":"result","termination":"trap","exit_code":0,"message":"","fuel_remaining":0,"peak_memory_bytes":65536,"output_bytes":0}`, "invalid"},
		{"missing-terminal", "", "infrastructure"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			request := syntheticHelper(t, "read request\n")
			request.Timeout = time.Second
			script := "#!/bin/sh\nread request\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\n"
			if fixture.terminal != "" {
				script += "printf '%s\\n' '" + fixture.terminal + "'\n"
			}
			if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			request.HelperSHA256 = fileHash(t, request.HelperPath)
			result, err := Run(context.Background(), request)
			if err != nil || result.Termination != fixture.want || !result.Reaped || fixture.name == "missing-terminal" && result.WatchdogTimeout {
				t.Fatalf("terminal = %s %s reaped=%v err=%v", result.Termination, result.Message, result.Reaped, err)
			}
		})
	}
}

func TestStockGoGuestRepeatsIn100FreshInstances(t *testing.T) {
	helper := helperPath(t)
	goCommand := os.Getenv("GOMAD3_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	compiler, err := exec.Command(goCommand, "version").Output()
	if err != nil || !strings.Contains(string(compiler), "go1.27.1 ") {
		t.Fatalf("stock compiler identity: %q, %v", compiler, err)
	}
	module := filepath.Join(t.TempDir(), "environment.wasm")
	command := exec.Command(goCommand, "build", "-trimpath", "-o", module, "../testdata/environment")
	command.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build real guest: %v\n%s", err, data)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Clock.ReadStepNanos = 1000000
	environment := newTestEnvironment(t, config)
	request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: 60 * time.Second}
	var stdout, transcript []byte
	for i := 0; i < 100; i++ {
		result, err := Run(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped || result.EnvironmentIdentity != environment.Identity() {
			t.Fatalf("guest %d: %s: %s reaped=%v\nstdout=%s stderr=%s transcript bytes=%d hash=%x", i, result.Termination, result.Message, result.Reaped, string(result.Stdout), string(result.Stderr), len(result.Transcript), sha256.Sum256(result.Transcript))
		}
		if i == 0 {
			stdout = result.Stdout
			transcript = result.Transcript
			for _, marker := range []string{"map ", "select ", "random ", "timer ", "fakeTCP echo", "cwd /workspace env captured arg literal $arg", "file scratch"} {
				if !strings.Contains(string(stdout), marker) {
					t.Fatalf("missing guest probe %q: %s", marker, stdout)
				}
			}
		} else if !bytes.Equal(stdout, result.Stdout) || !bytes.Equal(transcript, result.Transcript) {
			t.Fatalf("fresh guest %d diverged: stdout %q != %q; transcript %x != %x", i, result.Stdout, stdout, sha256.Sum256(result.Transcript), sha256.Sum256(transcript))
		}
	}
	t.Logf("100 fresh guests, 100 output and transcript equalities: compiler=%s experiment=nogreenteagc environment=%s module=%s helper=%s stdout=%x transcript=%x imports=%d memory-pages=%d", strings.TrimSpace(string(compiler)), environment.Identity(), request.ModuleSHA256, request.HelperSHA256, sha256.Sum256(stdout), sha256.Sum256(transcript), len(imports), pages)
}

func binaryInventory(t *testing.T, data []byte) ([]Import, uint64) {
	t.Helper()
	if len(data) < 8 || string(data[:4]) != "\x00asm" {
		t.Fatal("not binary WASM")
	}
	readUint := func(buffer *[]byte) uint64 {
		var value uint64
		for shift := uint(0); shift < 64; shift += 7 {
			if len(*buffer) == 0 {
				t.Fatal("truncated WASM LEB")
			}
			b := (*buffer)[0]
			*buffer = (*buffer)[1:]
			value |= uint64(b&127) << shift
			if b < 128 {
				return value
			}
		}
		t.Fatal("overflow WASM LEB")
		return 0
	}
	readName := func(buffer *[]byte) string {
		size := readUint(buffer)
		if size > uint64(len(*buffer)) {
			t.Fatal("truncated WASM name")
		}
		name := string((*buffer)[:size])
		*buffer = (*buffer)[size:]
		return name
	}
	types := []Import{}
	imports := []Import{}
	pages := uint64(0)
	for remaining := data[8:]; len(remaining) > 0; {
		kind := remaining[0]
		remaining = remaining[1:]
		size := readUint(&remaining)
		if size > uint64(len(remaining)) {
			t.Fatal("truncated WASM section")
		}
		section := remaining[:size]
		remaining = remaining[size:]
		switch kind {
		case 1:
			count := readUint(&section)
			for i := uint64(0); i < count; i++ {
				if len(section) == 0 || section[0] != 0x60 {
					t.Fatal("fixture WASM function type")
				}
				section = section[1:]
				signature := Import{Params: []string{}, Results: []string{}}
				for j := 0; j < 2; j++ {
					count := readUint(&section)
					for k := uint64(0); k < count; k++ {
						if len(section) == 0 {
							t.Fatal("truncated value type")
						}
						name := map[byte]string{0x7f: "i32", 0x7e: "i64", 0x7d: "f32", 0x7c: "f64"}[section[0]]
						section = section[1:]
						if name == "" {
							t.Fatal("fixture WASM value type")
						}
						if j == 0 {
							signature.Params = append(signature.Params, name)
						} else {
							signature.Results = append(signature.Results, name)
						}
					}
				}
				types = append(types, signature)
			}
		case 2:
			count := readUint(&section)
			for i := uint64(0); i < count; i++ {
				module, name := readName(&section), readName(&section)
				if len(section) == 0 || section[0] != 0 {
					t.Fatal("fixture import is not function")
				}
				section = section[1:]
				index := readUint(&section)
				if index >= uint64(len(types)) {
					t.Fatal("invalid fixture type index")
				}
				signature := types[index]
				signature.Module = module
				signature.Name = name
				imports = append(imports, signature)
			}
		case 5:
			if readUint(&section) != 1 {
				t.Fatal("fixture memory count")
			}
			flags := readUint(&section)
			pages = readUint(&section)
			if flags&1 != 0 {
				readUint(&section)
			}
		}
	}
	if pages == 0 {
		t.Fatal("fixture memory missing")
	}
	return imports, pages
}

func syntheticHelper(t *testing.T, script string) Request {
	t.Helper()
	request := fixtureRequest(t, `(module (memory (export "memory") 1) (func (export "_start")))`, nil)
	helper := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperPath = helper
	request.HelperSHA256 = fileHash(t, helper)
	return request
}
func startedJSON(t *testing.T, request Request) string {
	t.Helper()
	data, err := json.Marshal(Started{Schema: "gomad3.wasm-host/v1", Type: "started", Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, Imports: []Import{}, Exports: []Export{{"memory", "memory"}, {"_start", "func"}}, InitialMemoryPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func TestTransportClassifiesHelperDeathWatchdogCancellationAndMalformedFrames(t *testing.T) {
	for _, fixture := range []struct {
		name, script, want string
		cancel             bool
		timeout            time.Duration
	}{
		{"death", "read request\nkill -KILL $$\n", "infrastructure", false, time.Second},
		{"watchdog", "read request\n/bin/sleep 30\n", "infrastructure", false, 50 * time.Millisecond},
		{"cancel", "read request\n/bin/sleep 30\n", "infrastructure", true, time.Second},
		{"malformed", "read request\nprintf '{\"type\":\"call\",\"id\":1,\"id\":2,\"op\":\"sched_yield\",\"input\":{}}\\n'\n", "invalid", false, time.Second},
		{"frame-capacity", "read request\nprintf '%8388610s\\n' ''\n", "capacity", false, time.Second},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			request := syntheticHelper(t, fixture.script)
			request.Timeout = fixture.timeout
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fixture.cancel {
				timer := time.AfterFunc(50*time.Millisecond, cancel)
				defer timer.Stop()
			}
			result, err := Run(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Termination != fixture.want || !result.Reaped || fixture.cancel && !result.Cancelled || fixture.name == "watchdog" && !result.WatchdogTimeout || fixture.want != "infrastructure" && (result.Cancelled || result.WatchdogTimeout) {
				t.Fatalf("lifecycle = %s %s reaped=%v cancel=%v watchdog=%v", result.Termination, result.Message, result.Reaped, result.Cancelled, result.WatchdogTimeout)
			}
		})
	}
	request := syntheticHelper(t, "read request\n")
	started := startedJSON(t, request)
	script := "read request\nprintf '%s\\n' '" + started + "'\nread authorization\nprintf '%s\\n' '{\"type\":\"call\",\"id\":2,\"op\":\"sched_yield\",\"input\":{}}'\nread reply\n"
	if err := os.WriteFile(request.HelperPath, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	result, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != "invalid" || !result.Reaped || len(result.Transcript) != 0 {
		t.Fatalf("out-of-order callback = %s %s", result.Termination, result.Message)
	}
}

func TestTransportPreservesCancellationMetadataWhileReplyIsBlocked(t *testing.T) {
	for _, mode := range []string{"parent", "watchdog"} {
		t.Run(mode, func(t *testing.T) {
			request := syntheticHelper(t, "read request\n")
			request.Config.Args = []string{strings.Repeat("x", 1<<20)}
			script := "#!/bin/sh\nread request\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\nprintf '%s\\n' '{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}'\n/bin/sleep 30\n"
			if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			request.HelperSHA256 = fileHash(t, request.HelperPath)
			request.Timeout = 2 * time.Second
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "parent" {
				request.Timeout = 4 * time.Second
				timer := time.AfterFunc(2*time.Second, cancel)
				defer timer.Stop()
			}
			result, err := Run(ctx, request)
			if err != nil || result.Termination != "infrastructure" || !result.Reaped || result.Cancelled != (mode == "parent") || result.WatchdogTimeout != (mode == "watchdog") || len(result.Transcript) < 1<<20 {
				t.Fatalf("blocked reply %s: %s %s reaped=%v cancel=%v watchdog=%v transcript=%d err=%v", mode, result.Termination, result.Message, result.Reaped, result.Cancelled, result.WatchdogTimeout, len(result.Transcript), err)
			}
		})
	}
}
