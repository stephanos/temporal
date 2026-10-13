package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestHelperInputLargeBootstrapStartsBeforeReader(t *testing.T) {
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	var sentDone chan struct{}
	t.Cleanup(func() {
		if err := input.finish(); err != nil {
			t.Error(err)
		}
		if sentDone != nil {
			<-sentDone
		}
	})
	initial := bytes.Repeat([]byte("bootstrap"), 1<<18)
	initial = append(initial, '\n')
	started := make(chan struct{})
	go func() { input.start(initial); close(started) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		input.closeInput()
		<-started
		if err := input.finish(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("bootstrap blocked before the child could start reading")
	}
	reply := []byte("authorization\nreply\ninterrupt\n")
	sent := make(chan error, 1)
	sentDone = make(chan struct{})
	go func() {
		defer close(sentDone)
		n, err := input.Write(reply)
		if err == nil && n != len(reply) {
			err = io.ErrShortWrite
		}
		sent <- err
	}()
	want := append(bytes.Clone(initial), reply...)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(input.reader, got); err != nil {
		t.Fatal(err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("reply interleaved with bootstrap bytes")
	}
}

func TestHelperInputCloseUnblocksBootstrapAndReply(t *testing.T) {
	for _, stage := range []string{"bootstrap", "reply"} {
		t.Run(stage, func(t *testing.T) {
			input, err := newHelperInput()
			if err != nil {
				t.Fatal(err)
			}
			var writtenDone chan struct{}
			t.Cleanup(func() {
				if err := input.finish(); err != nil {
					t.Error(err)
				}
				if writtenDone != nil {
					<-writtenDone
				}
			})
			initial := []byte("execute\n")
			if stage == "bootstrap" {
				initial = bytes.Repeat([]byte("x"), 2<<20)
			}
			input.start(initial)
			if stage == "reply" {
				got := make([]byte, len(initial))
				if _, err := io.ReadFull(input.reader, got); err != nil {
					t.Fatal(err)
				}
			}
			written := make(chan error, 1)
			writtenDone = make(chan struct{})
			go func() {
				defer close(writtenDone)
				_, err := input.Write(bytes.Repeat([]byte("y"), 2<<20))
				written <- err
			}()
			input.closeInput()
			select {
			case err := <-written:
				if err != io.ErrClosedPipe {
					t.Fatalf("closed input changed the transport sentinel: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("closing input did not unblock writer")
			}
			if err := input.finish(); err != nil {
				t.Fatal(err)
			}
			if err := input.finish(); err != nil {
				t.Fatal("repeated cleanup: ", err)
			}
			if _, err := input.reader.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("read descriptor not closed: %v", err)
			}
		})
	}
}

func TestHelperInputBrokenPeerIsNotModeledCancellation(t *testing.T) {
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := input.finish(); !errors.Is(err, syscall.EPIPE) || !errors.Is(err, os.ErrClosed) {
			t.Errorf("broken-peer cleanup: %v", err)
		}
	})
	input.start(bytes.Repeat([]byte("x"), 2<<20))
	if err := input.reader.Close(); err != nil {
		t.Fatal(err)
	}
	// The read end was deliberately closed above; cleanup must retain that error.
	<-input.ready
	err = input.finish()
	if !errors.Is(err, syscall.EPIPE) || !errors.Is(err, os.ErrClosed) || modeledReapedCancellation(err, true, true, true, false) {
		t.Fatalf("cleanup concealed an independent bootstrap failure: %v", err)
	}
}

type characterizedBootstrap struct {
	*bytes.Reader
	writeErr error
}

func (input *characterizedBootstrap) WriteTo(writer io.Writer) (int64, error) {
	written, err := input.Reader.WriteTo(writer)
	input.writeErr = err
	return written, err
}

func TestTransportOldExecPumpBrokenPeerCharacterization(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", "exec 0<&-; exit 0")
	input := &characterizedBootstrap{Reader: bytes.NewReader(bytes.Repeat([]byte("x"), 2<<20))}
	command.Stdin = input
	if err := command.Run(); err != nil || !errors.Is(input.writeErr, syscall.EPIPE) {
		t.Fatalf("old os/exec pump precedence: exit=%v bootstrap=%v", err, input.writeErr)
	}
}

type transportWriteFunc func([]byte) (int, error)

func (write transportWriteFunc) Write(data []byte) (int, error) { return write(data) }

func TestTransportReplyWriteErrorPrecedence(t *testing.T) {
	failure := errors.New("writer failed")
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{"short-write", nil, io.ErrShortWrite},
		{"error-before-short-write", failure, failure},
	} {
		t.Run(test.name, func(t *testing.T) {
			sink := frameSink{input: transportWriteFunc(func(data []byte) (int, error) {
				if string(data) != "{\"type\":\"interrupt\",\"termination\":\"stop\",\"message\":\"modeled\"}\n" {
					t.Fatalf("interrupt bytes changed: %q", data)
				}
				return len(data) - 1, test.err
			})}
			if err := sink.interrupt("stop", "modeled"); err != test.want {
				t.Fatalf("write precedence = %v, want %v", err, test.want)
			}
		})
	}
}

func TestTransportDirectInputHandlesLargeBootstrapBeforeFirstRead(t *testing.T) {
	request := syntheticHelper(t, "read request\n")
	request.CompileCache = "/" + strings.Repeat("x", 2<<20)
	authorization := fmt.Sprintf(`{"type":"authorize","module_sha256":%q,"configuration":%q}`, request.ModuleSHA256, request.Engine.Configuration)
	script := "#!/bin/sh\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread request\n" +
		"test \"${#request}\" -gt 2097152 || exit 21\nread authorization\n" +
		"test \"$authorization\" = '" + authorization + "' || exit 22\n" +
		"printf '%s\\n' '{\"type\":\"result\",\"termination\":\"exit\",\"exit_code\":0,\"message\":\"\",\"fuel_remaining\":0,\"peak_memory_bytes\":65536,\"output_bytes\":0}'\n"
	if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	result, err := Run(t.Context(), request)
	if err != nil || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped {
		t.Fatalf("large bootstrap = %s %s reaped=%v err=%v", result.Termination, result.Message, result.Reaped, err)
	}
}

func TestTransportDirectInputReapsBlockedBootstrapAndFailedStart(t *testing.T) {
	for _, stage := range []string{"parent", "watchdog", "failed-start", "early-exit", "started-without-drain"} {
		t.Run(stage, func(t *testing.T) {
			request := syntheticHelper(t, "/bin/sleep 30\n")
			request.CompileCache = "/" + strings.Repeat("x", 2<<20)
			request.Timeout = time.Second
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch stage {
			case "parent":
				timer := time.AfterFunc(50*time.Millisecond, cancel)
				defer timer.Stop()
			case "watchdog":
				request.Timeout = 50 * time.Millisecond
			case "failed-start", "early-exit", "started-without-drain":
				script := "not an executable image\n"
				if stage == "early-exit" {
					script = "#!/bin/sh\nexit 0\n"
				}
				if stage == "started-without-drain" {
					script = "#!/bin/sh\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nexit 0\n"
					request.Timeout = 50 * time.Millisecond
				}
				if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				request.HelperSHA256 = fileHash(t, request.HelperPath)
			}
			result, err := Run(ctx, request)
			if err != nil || result.Termination != "infrastructure" || result.Message == "" || result.Reaped != (stage != "failed-start") || result.Cancelled != (stage == "parent") || result.WatchdogTimeout != (stage == "watchdog" || stage == "started-without-drain") || result.RuntimeInterrupted {
				t.Fatalf("%s = %s %s reaped=%v cancelled=%v watchdog=%v interrupted=%v err=%v", stage, result.Termination, result.Message, result.Reaped, result.Cancelled, result.WatchdogTimeout, result.RuntimeInterrupted, err)
			}
		})
	}
}

func TestModeledCancellationRequiresExactCauseAndReapedRevocation(t *testing.T) {
	for _, test := range []struct {
		name                                       string
		err                                        error
		revoked, reaped, cancelled, watchdog, want bool
	}{
		{"no-transport-error", nil, true, true, true, false, true},
		{"cancel-sentinel", context.Canceled, true, true, true, false, true},
		{"closed-pipe-sentinel", io.ErrClosedPipe, true, true, true, false, true},
		{"ordinary-closed-pipe", io.ErrClosedPipe, false, true, true, false, false},
		{"not-reaped", io.ErrClosedPipe, true, false, true, false, false},
		{"unexpected-pipe-before-cancel", io.ErrClosedPipe, true, true, false, false, false},
		{"watchdog", io.ErrClosedPipe, true, true, true, true, false},
		{"invalid-terminal", invalid("terminal classification"), true, true, true, false, false},
		{"arbitrary-error", errors.New("io: read/write on closed pipe"), true, true, true, false, false},
		{"wrapped-cause", fmt.Errorf("transport: %w", io.ErrClosedPipe), true, true, true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := modeledReapedCancellation(test.err, test.revoked, test.reaped, test.cancelled, test.watchdog); got != test.want {
				t.Fatalf("modeled interruption = %v, want %v", got, test.want)
			}
		})
	}
}

func helperPath(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("GOMAD_WASM_TEST_HELPER"); override != "" {
		if !filepath.IsAbs(override) {
			t.Fatal("GOMAD_WASM_TEST_HELPER must be absolute")
		}
		if _, err := os.Stat(override); err != nil {
			t.Fatal(err)
		}
		return override
	}
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

// A revocation can land after the broker handled a call and its reply was
// written, but before delivery completes. The helper may already have issued
// its next call, so that call receives the interrupt in place of its reply.
// Any later call, like any call after an interrupt that replaced a reply,
// remains a protocol violation.
func TestTransportRevocationAfterReplyInterruptsTheCrossingCall(t *testing.T) {
	call := func(id int) []byte {
		return fmt.Appendf(nil, `{"type":"call","id":%d,"op":"fd_write","input":{"fd":1,"data_base64":"eA=="}}`, id)
	}
	for _, kind := range []string{"crash", "stop"} {
		for _, afterReply := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/after-reply=%v", kind, afterReply), func(t *testing.T) {
				b, err := NewBroker(brokerTestConfig())
				if err != nil {
					t.Fatal(err)
				}
				revoke := func() {
					if kind == "crash" {
						err = b.Crash("a")
					} else {
						err = b.Stop("a")
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				var frames []string
				sink := frameSink{request: Request{Broker: b.Session("a")}, environment: newTestEnvironment(t, testConfig()), started: &Started{}, context: ctx, input: transportWriteFunc(func(data []byte) (int, error) {
					frames = append(frames, string(data))
					if afterReply && len(frames) == 1 {
						revoke()
					}
					return len(data), nil
				})}
				if !afterReply {
					revoke()
				}
				if err := sink.frame(call(1)); err != nil {
					t.Fatal(err)
				}
				if afterReply {
					if err := sink.frame(call(2)); err != nil {
						t.Fatalf("call crossing a revocation after its predecessor's reply: %v", err)
					}
				}
				interrupt := fmt.Sprintf(`{"type":"interrupt","termination":%q,"message":"WASI %s: broker incarnation revoked"}`+"\n", kind, kind)
				if len(frames) == 0 || frames[len(frames)-1] != interrupt || sink.interruption != kind {
					t.Fatalf("frames %q interruption %q", frames, sink.interruption)
				}
				if err := sink.frame(call(3)); err == nil || !strings.Contains(err.Error(), "callback after modeled interruption") {
					t.Fatalf("callback after a delivered interrupt accepted: %v", err)
				}
			})
		}
	}
}
