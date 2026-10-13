package native

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/native/execution"
	"go.temporal.io/server/tools/gomad3/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
)

func TestProviderExecutePreservesNativeProcessObservation(t *testing.T) {
	config := Config{SupervisorCommand: []string{os.Args[0], "-test.run=^TestNativePrivateHelper$"}, BootstrapCommand: []string{os.Args[0], "-test.run=^TestNativePrivateHelper$"}}
	provider := New(config)
	config.SupervisorCommand[0] = "must-not-launch"
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	request := backends.ExecutionRequest{
		Target:  target.Prepared{Path: os.Args[0], Argv: []string{"gomad3-target", "-test.run=^TestNativeTargetHelper$"}, SHA256: string(record.HashBytes(binary)), Size: uint64(len(binary)), Compatibility: []record.CompatibilityPack{}},
		Command: os.Args[0], Args: []string{"-test.run=^TestNativeTargetHelper$"}, Argv0: "gomad3-target", Dir: t.TempDir(),
		Env: []string{"GOMAD3_NATIVE_TARGET_TEST=1"}, ExecutionTimeout: 5 * time.Second, TerminateGrace: time.Second, OutputLimit: 64,
		World: backends.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
	}
	observed, err := provider.Execute(t.Context(), request)
	if err != nil || observed.Termination != backends.TerminationExit || observed.ExitCode != 7 || !observed.GroupGone || observed.PID <= 0 || observed.PGID != observed.PID || string(observed.Stdout.Bytes) != "native stdout\n" || string(observed.Stderr.Bytes) != "native stderr\n" {
		t.Fatalf("native observation = %#v, %v", observed, err)
	}
	if record.SHA256FromSum(observed.Stdout.FullSHA256) != record.HashBytes([]byte("native stdout\n")) || observed.Stdout.TotalBytes != 14 {
		t.Fatalf("native output hashes/count = %#v", observed.Stdout)
	}
	for _, misuse := range []struct {
		name   string
		change func(*backends.ExecutionRequest)
		want   string
	}{
		{"command", func(request *backends.ExecutionRequest) { request.Command = "/must-not-launch" }, "execution command does not match the prepared target"},
		{"arguments", func(request *backends.ExecutionRequest) { request.Args = []string{"-test.run=^MustNotLaunch$"} }, "execution arguments do not match the prepared target"},
		{"argv0", func(request *backends.ExecutionRequest) { request.Argv0 = "foreign-target" }, "execution argv0 does not match the prepared target"},
		{"binary identity", func(request *backends.ExecutionRequest) {
			request.Target.SHA256 = string(record.HashBytes([]byte("foreign")))
		}, "prepared target changed after preparation"},
		{"compatibility", func(request *backends.ExecutionRequest) { request.Target.Compatibility = nil }, "compatibility pack identity is missing"},
	} {
		t.Run(misuse.name, func(t *testing.T) {
			altered := request
			misuse.change(&altered)
			if _, err := provider.Execute(t.Context(), altered); err == nil || !strings.Contains(err.Error(), misuse.want) {
				t.Fatalf("misuse error = %v, want %s", err, misuse.want)
			}
		})
	}
	withIO := request
	withIO.IO = &backends.IOCapability{Profile: recordedProfile(transcript.Default())}
	for _, missing := range []struct {
		config Config
		want   string
	}{
		{Config{}, "supervisor command is required"},
		{Config{SupervisorCommand: []string{"unused"}, BootstrapCommand: []string{""}}, "target bootstrap command is required"},
	} {
		if _, err := New(missing.config).Execute(t.Context(), withIO); err == nil || err.Error() != missing.want {
			t.Fatalf("missing native command error = %v, want %s", err, missing.want)
		}
	}
}

func TestNativePrivateHelper(t *testing.T) {
	var err error
	switch {
	case os.Getenv("GOMAD3_TARGET_BOOTSTRAP") == "1":
		err = execution.BootstrapMain()
	case os.Getenv("GOMAD3_PROCESS_SUPERVISOR") == "1":
		err = execution.SupervisorMain()
	default:
		t.Skip("native subprocess only")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	os.Exit(0)
}

func TestNativeTargetHelper(t *testing.T) {
	if os.Getenv("GOMAD3_NATIVE_TARGET_TEST") != "1" {
		t.Skip("target subprocess only")
	}
	if _, err := fmt.Fprintln(os.Stdout, "native stdout"); err != nil {
		os.Exit(4)
	}
	if _, err := fmt.Fprintln(os.Stderr, "native stderr"); err != nil {
		os.Exit(5)
	}
	os.Exit(7)
}

func TestProviderCoordinatorOwnsTransferredWriter(t *testing.T) {
	for _, mode := range []string{"complete", "start-failure", "cancelled", "mixed-target"} {
		t.Run(mode, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := reader.Close(); err != nil {
					t.Error(err)
				}
			}()
			provider := New(Config{CoordinatorCommand: []string{os.Args[0], "-test.run=^TestNativeCoordinatorHelper$"}, SupervisorCommand: []string{"native-supervisor"}, RunnerBuild: "native-build"})
			if mode == "start-failure" {
				provider = New(Config{CoordinatorCommand: []string{"/missing-native-coordinator"}})
			}
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			deadline := time.Now().Add(2 * time.Second)
			request := backends.ExecutionRequest{Coordinator: &backends.CoordinatorRequest{Input: []byte(`{"Target":{"ToolchainRoot":"native-root"},"Parallel":7}`), Stdout: writer, Deadline: deadline}}
			if mode == "mixed-target" {
				request.Command = "must-not-launch"
			}
			observed, runErr := provider.Execute(ctx, request)
			if mode == "complete" && (runErr != nil || !observed.GroupGone) {
				t.Fatalf("coordinator result = %#v, %v", observed, runErr)
			}
			if mode == "start-failure" && (runErr == nil || observed.CoordinatorErrorReason != "coordinator_start") {
				t.Fatalf("start failure = %#v, %v", observed, runErr)
			}
			if mode == "cancelled" && !errors.Is(runErr, context.Canceled) {
				t.Fatalf("cancelled coordinator = %v", runErr)
			}
			if mode == "mixed-target" && (runErr == nil || !strings.Contains(runErr.Error(), "coordinator request cannot include target execution")) {
				t.Fatalf("mixed coordinator error = %v", runErr)
			}
			if _, err := writer.Write([]byte("closed")); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("transferred writer error = %v", err)
			}
			bytes, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "complete" && (!strings.Contains(string(bytes), `"Parallel":7`) || !strings.Contains(string(bytes), `"SupervisorCommand":["native-supervisor"]`)) {
				t.Fatalf("coordinator wire = %s", bytes)
			}
		})
	}
}

func TestNativeCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		os.Exit(6)
	}
	os.Exit(0)
}
