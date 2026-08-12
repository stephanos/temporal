package process

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/world"
	worldchild "go.temporal.io/server/tools/gomadv3/world/child"
)

func TestRunCapturesTargetExitAndBothStreams(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand:    []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:     []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:              os.Args[0],
		Args:                 []string{"-test.run=TestTargetHelper"},
		Argv0:                "gomadv3-target",
		Dir:                  t.TempDir(),
		Env:                  []string{"GOMADV3_PROCESS_HELPER=output"},
		RunTimeout:           5 * time.Second,
		TerminateGrace:       time.Second,
		OutputLimit:          64,
		WorldRecordLimit:     1 << 20,
		WorldTransitionLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 7 || result.Signal != "" || result.WatchdogTimeout || result.Cancelled {
		t.Fatalf("result status = %#v", result)
	}
	if got, want := string(result.Stdout.Bytes), "target stdout\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := string(result.Stderr.Bytes), "target stderr\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	if result.PID <= 0 || result.PGID != result.PID || !result.GroupGone {
		t.Fatalf("process identity = pid %d pgid %d gone %v", result.PID, result.PGID, result.GroupGone)
	}
}

func TestRunInstallsBoundedIOConfigurationDescriptor(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=io-config"}, IOConfig: []byte("profile-frame"),
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "profile-frame" {
		t.Fatalf("result = %#v, stdout = %q", result, result.Stdout.Bytes)
	}
}

func TestRunInstallsReadOnlyMountBrokerDescriptors(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "file"), []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=io-ro-mount"}, IOConfig: []byte("profile-frame"), IOTranscriptLimit: 1 << 20,
		IOROMounts: []romount.Mapping{{Source: source, Target: "/mounted"}}, IOROMountLimits: romount.DefaultLimits(),
		RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || string(result.Stdout.Bytes) != "contents" || len(result.IOROMounts.Entries) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestValidateRequestRejectsExpectedIOTranscriptOutsideReplay(t *testing.T) {
	request := Request{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		RunTimeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024,
		WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		IOConfig: []byte("profile-frame"), IOTranscriptLimit: 1 << 20,
	}
	request.ExpectedIOTranscript = make([]byte, ioTranscriptRecordBytes)
	if err := validateRequest(request); err == nil || !strings.Contains(err.Error(), "requires replay mode") {
		t.Fatalf("validateRequest() error = %v", err)
	}
	request.IOReplay = true
	request.ExpectedIOTranscript = make([]byte, ioTranscriptRecordBytes-1)
	if err := validateRequest(request); err == nil || !strings.Contains(err.Error(), "invalid expected I/O transcript length") {
		t.Fatalf("validateRequest() error = %v", err)
	}
}

func TestRunTimesOutAndRemovesTermIgnoringProcessGroup(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand:    []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:     []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:              "/bin/sh",
		Args:                 []string{"-c", `trap '' TERM; (trap '' TERM; while :; do :; done) & while :; do :; done`},
		Argv0:                "gomadv3-target",
		Dir:                  t.TempDir(),
		Env:                  []string{"TZ=UTC"},
		RunTimeout:           750 * time.Millisecond,
		TerminateGrace:       250 * time.Millisecond,
		OutputLimit:          64,
		WorldRecordLimit:     1 << 20,
		WorldTransitionLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.WatchdogTimeout || result.Termination != TerminationSignal || result.Signal != "killed" || !result.GroupGone {
		t.Fatalf("timeout result = %#v", result)
	}
}

func TestRunBoundsFloodedOutputWithoutBlocking(t *testing.T) {
	stdoutHead, err := os.CreateTemp(t.TempDir(), "stdout-head")
	if err != nil {
		t.Fatal(err)
	}
	defer stdoutHead.Close()
	stderrHead, err := os.CreateTemp(t.TempDir(), "stderr-head")
	if err != nil {
		t.Fatal(err)
	}
	defer stderrHead.Close()
	result, err := Run(context.Background(), Request{
		SupervisorCommand:    []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:     []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:              os.Args[0],
		Args:                 []string{"-test.run=TestTargetHelper"},
		Argv0:                "gomadv3-target",
		Dir:                  t.TempDir(),
		Env:                  []string{"GOMADV3_PROCESS_HELPER=flood"},
		RunTimeout:           5 * time.Second,
		TerminateGrace:       time.Second,
		OutputLimit:          128,
		WorldRecordLimit:     1 << 20,
		WorldTransitionLimit: 1 << 20,
		StdoutHead:           stdoutHead,
		StderrHead:           stderrHead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 {
		t.Fatalf("result status = %#v", result)
	}
	for name, output := range map[string]Output{"stdout": result.Stdout, "stderr": result.Stderr} {
		if !output.Truncated || output.TotalBytes != 1<<20 || output.RetainedBytes != 128 || output.DiscardedBytes != 1<<20-128 {
			t.Fatalf("%s accounting = %#v", name, output)
		}
		if !strings.Contains(string(output.Bytes), "gomadv3 output truncated") {
			t.Fatalf("%s retained bytes omitted marker", name)
		}
	}
	for name, file := range map[string]*os.File{"stdout": stdoutHead, "stderr": stderrHead} {
		info, statErr := file.Stat()
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Size() != 96 {
			t.Fatalf("%s partial head size = %d, want 96", name, info.Size())
		}
	}
}

func TestRunBoundsUnresponsiveSupervisor(t *testing.T) {
	started := time.Now()
	_, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestUnresponsiveSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=output"}, RunTimeout: 300 * time.Millisecond, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
		WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
	})
	if err == nil || !strings.Contains(err.Error(), "supervisor") {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("unresponsive supervisor took %v", elapsed)
	}
}

func TestRunGivesDescendantsGraceAfterLeaderExit(t *testing.T) {
	directory := t.TempDir()
	readyPath := filepath.Join(directory, "ready")
	gracefulPath := filepath.Join(directory, "graceful")
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: directory,
		Env:        []string{"GOMADV3_PROCESS_HELPER=spawn-child", "GOMADV3_HELPER_EXE=" + os.Args[0], "GOMADV3_READY_PATH=" + readyPath, "GOMADV3_GRACEFUL_PATH=" + gracefulPath},
		RunTimeout: 3 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(gracefulPath); err != nil {
		t.Fatalf("descendant did not exit during TERM grace: %v", err)
	}
}

func TestRunCapturesWorldRecordFromExecutingChild(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: 1 << 20,
		WorldSeed: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err != nil {
		t.Fatal(err)
	}
	if recording.Initial.Config.Seed != 9 || len(recording.Final.Transitions)-len(recording.Initial.Transitions) != 2 || recording.Terminal.Kind != world.TerminalDeadlock {
		t.Fatalf("child World recording = %#v", recording)
	}
}

func TestRunPreservesPrematureWorldProducerMarker(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-open-only"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: 1 << 20,
		WorldSeed: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.WorldRecord) == 0 {
		t.Fatal("premature World producer was indistinguishable from an unconnected target")
	}
	if _, err := world.DecodeRecording(result.WorldRecord); err == nil {
		t.Fatal("premature World producer emitted a complete record")
	}
}

func TestRunPreservesWorldSeedMismatchMarker(t *testing.T) {
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=8", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: 1 << 20,
		WorldSeed: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 15 || len(result.WorldRecord) == 0 {
		t.Fatalf("seed-mismatch result = %#v", result)
	}
	if _, err := world.DecodeRecording(result.WorldRecord); err == nil {
		t.Fatal("World seed mismatch emitted a complete record")
	}
}

func TestRunInstallsWorldInitialReplayInputBeforeModeledWork(t *testing.T) {
	limits := world.Limits{MaxRequests: 11, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}
	core, err := world.New(world.Config{Seed: 9, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	expectedInitial, err := world.EncodeSnapshot(core.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADSEED=9", "GOMADV3_PROCESS_HELPER=world-record"}, RunTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		WorldRecordLimit: world.MaximumRecordingBytes, WorldTransitionLimit: 1 << 20, WorldSeed: 9, ExpectedWorldInitial: expectedInitial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("trusted-initial result = %#v", result)
	}
	recording, err := world.DecodeRecording(result.WorldRecord)
	if err != nil {
		t.Fatal(err)
	}
	if recording.Initial.Config.Limits.MaxRequests != 11 {
		t.Fatalf("initial max requests = %d, want 11", recording.Initial.Config.Limits.MaxRequests)
	}
}

func TestSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	if err := SupervisorMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestTargetBootstrapHelper(t *testing.T) {
	if os.Getenv("GOMADV3_TARGET_BOOTSTRAP") != "1" {
		t.Skip("target bootstrap subprocess only")
	}
	if err := BootstrapMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestTargetHelper(t *testing.T) {
	switch os.Getenv("GOMADV3_PROCESS_HELPER") {
	case "output":
		fmt.Fprintln(os.Stdout, "target stdout")
		fmt.Fprintln(os.Stderr, "target stderr")
		os.Exit(7)
	case "flood":
		if _, err := os.Stdout.Write(make([]byte, 1<<20)); err != nil {
			os.Exit(8)
		}
		if _, err := os.Stderr.Write(make([]byte, 1<<20)); err != nil {
			os.Exit(9)
		}
		os.Exit(0)
	case "io-config":
		configuration := os.NewFile(5, "gomadv3-io-config")
		if configuration == nil {
			os.Exit(21)
		}
		if _, err := io.Copy(os.Stdout, configuration); err != nil {
			os.Exit(22)
		}
		if err := configuration.Close(); err != nil {
			os.Exit(23)
		}
		os.Exit(0)
	case "io-ro-mount":
		request := os.NewFile(9, "gomadv3-io-ro-mount-request")
		response := os.NewFile(10, "gomadv3-io-ro-mount-response")
		if request == nil || response == nil || romount.WriteLookupRequest(request, 0, "/mounted/file") != nil {
			os.Exit(24)
		}
		entry, err := romount.ReadResponse(response, romount.DefaultLimits())
		if err != nil || entry.Status != romount.StatusOK {
			os.Exit(25)
		}
		if _, err := os.Stdout.Write(entry.Entry.Data); err != nil {
			os.Exit(26)
		}
		if err := writeEmptyIOTranscriptTerminal(); err != nil {
			os.Exit(27)
		}
		os.Exit(0)
	case "spawn-child":
		command := exec.Command(os.Getenv("GOMADV3_HELPER_EXE"), "-test.run=TestTargetHelper")
		command.Env = []string{
			"GOMADV3_PROCESS_HELPER=graceful-child",
			"GOMADV3_READY_PATH=" + os.Getenv("GOMADV3_READY_PATH"),
			"GOMADV3_GRACEFUL_PATH=" + os.Getenv("GOMADV3_GRACEFUL_PATH"),
		}
		if err := command.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(10)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := os.Stat(os.Getenv("GOMADV3_READY_PATH")); err == nil {
				os.Exit(0)
			}
			if time.Now().After(deadline) {
				os.Exit(11)
			}
			runtime.Gosched()
		}
	case "graceful-child":
		signal.Ignore(syscall.SIGTERM)
		if err := os.WriteFile(os.Getenv("GOMADV3_READY_PATH"), []byte("ready"), 0o600); err != nil {
			os.Exit(12)
		}
		<-time.After(250 * time.Millisecond)
		if err := os.WriteFile(os.Getenv("GOMADV3_GRACEFUL_PATH"), []byte("graceful"), 0o600); err != nil {
			os.Exit(13)
		}
		os.Exit(0)
	case "world-record":
		core, err := world.New(world.Config{Seed: 9, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(14)
		}
		session, err := worldchild.Open(core)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(15)
		}
		core = session.World()
		if _, err := core.Register(world.Request{Kind: "wait", Resource: world.ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(16)
		}
		if _, err := core.Quiesce(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(17)
		}
		if err := session.Finish(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(20)
		}
		os.Exit(0)
	case "world-open-only":
		core, err := world.New(world.Config{Seed: 9, Limits: world.Limits{MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 10, MaxPayloadBytes: 1024, MaxStringBytes: 64}})
		if err != nil {
			os.Exit(18)
		}
		if _, err := worldchild.Open(core); err != nil {
			os.Exit(19)
		}
		os.Exit(0)
	default:
		t.Skip("target subprocess only")
	}
}

func writeEmptyIOTranscriptTerminal() error {
	terminal := make([]byte, ioTerminalBytes)
	copy(terminal[:8], ioTerminalMagic[:])
	binary.BigEndian.PutUint32(terminal[8:12], 1)
	terminal[12] = 1
	binary.BigEndian.PutUint64(terminal[24:32], ioTranscriptHeaderBytes)
	digest := sha256.Sum256(nil)
	copy(terminal[32:64], digest[:])
	checksum := sha256.Sum256(terminal[:72])
	copy(terminal[72:], checksum[:])
	file := os.NewFile(7, "gomadv3-io-terminal")
	if file == nil {
		return errors.New("terminal descriptor unavailable")
	}
	if _, err := file.Write(terminal); err != nil {
		return err
	}
	return file.Close()
}

func TestUnresponsiveSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMADV3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	for {
	}
}
