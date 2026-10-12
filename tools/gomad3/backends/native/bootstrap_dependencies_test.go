package native

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

const scriptedBootstrapMarker = "synthetic scripted Runner bootstrap; not a toolchain frame"

func TestNativeBootstrapOperationArgumentsAndErrorsRemainUnchanged(t *testing.T) {
	primary := errors.New("injected operation failed")
	profile := transcript.Default()
	prepared := target.Prepared{Path: "fixture result", SHA256: "fixture digest", Size: 17}
	marker := []byte(scriptedBootstrapMarker)
	provider := New(Config{})
	provider.bootstrap = func(received transcript.Spec, receivedPrepared target.Prepared, runner string, seed uint64) ([]byte, error) {
		if received != profile || !reflect.DeepEqual(receivedPrepared, prepared) || runner != "fixture runner" || seed != 42 {
			t.Fatalf("bootstrap arguments = %#v, %#v, %q, %d", received, receivedPrepared, runner, seed)
		}
		return marker, primary
	}
	if result, err := provider.bootstrapFrame(profile, prepared, "fixture runner", 42); !bytes.Equal(result, marker) || err != primary {
		t.Fatalf("bootstrap result = %q, %v", result, err)
	}
}

func TestNativeBootstrapFailureStopsBeforeSupervision(t *testing.T) {
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	prepared := target.Prepared{Path: os.Args[0], Argv: []string{"gomad3-target"}, SHA256: string(record.HashBytes(binary)), Size: uint64(len(binary)), Compatibility: []record.CompatibilityPack{}}
	provider := New(Config{SupervisorCommand: []string{"must-not-launch"}, BootstrapCommand: []string{"must-not-launch"}})
	primary := errors.New("injected bootstrap failed")
	var calls []uint64
	provider.bootstrap = func(profile transcript.Spec, received target.Prepared, runner string, seed uint64) ([]byte, error) {
		if profile != transcript.Default() || !reflect.DeepEqual(received, prepared) || runner != "fixture runner" {
			t.Fatalf("bootstrap inputs = %#v, %#v, %q", profile, received, runner)
		}
		calls = append(calls, seed)
		return nil, primary
	}
	for _, seed := range []uint64{7, 8, 9} {
		_, err := provider.Execute(context.Background(), backends.ExecutionRequest{Target: prepared, Seed: seed, RunnerBuild: "fixture runner", Dir: t.TempDir(), ExecutionTimeout: time.Second, OutputLimit: 64, IO: &backends.IOCapability{Profile: recordedProfile(transcript.Default())}})
		if err != primary {
			t.Fatalf("bootstrap failure = %v", err)
		}
	}
	if !reflect.DeepEqual(calls, []uint64{7, 8, 9}) {
		t.Fatalf("bootstrap seeds = %v", calls)
	}
}

func TestNativeBootstrapDefaultRetainsUnsupportedHostGuard(t *testing.T) {
	profile := transcript.Default()
	_, guard := profile.BootstrapFrame(target.Prepared{}, "invalid", 0)
	if guard == nil || !strings.HasPrefix(guard.Error(), "deterministic I/O requires one of ") {
		t.Skip("unsupported-host refusal control requires an unqualified host")
	}
	_, err := New(Config{}).bootstrapFrame(profile, target.Prepared{}, "invalid", 0)
	if err == nil || err.Error() != guard.Error() {
		t.Fatalf("real bootstrap guard = %v, want %v", err, guard)
	}
}
