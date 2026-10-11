package hostexec_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/hostexec"
)

func TestCurrentProcessResourcesMeasureSelfWithoutChildren(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("platform resource usage not admitted")
	}
	before := hostexec.CurrentProcessResourceUsage()
	started := time.Now()
	for time.Since(started) < 20*time.Millisecond {
	}
	after := hostexec.CurrentProcessResourceUsage()
	if before == nil || after == nil || after.UserNanos+after.SystemNanos <= before.UserNanos+before.SystemNanos || after.PeakRSSBytes == 0 {
		t.Fatalf("actual operator self usage omitted: before=%+v after=%+v", before, after)
	}
}

func TestPublicResourceUsageIsOptIn(t *testing.T) {
	if os.Getenv("GOMAD_HOSTEXEC_RESOURCE_CHILD") == "yes" {
		started := time.Now()
		for time.Since(started) < 30*time.Millisecond {
		}
		return
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("platform resource usage not admitted")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	request := hostexec.Request{Command: []string{executable, "-test.run=^TestPublicResourceUsageIsOptIn$"}, Dir: t.TempDir(), Env: append(os.Environ(), "GOMAD_HOSTEXEC_RESOURCE_CHILD=yes"), Timeout: time.Second, OutputLimit: 1024}
	plain, err := hostexec.Run(t.Context(), request)
	if err != nil || plain.ResourceUsage != nil {
		t.Fatalf("default resource projection changed: %+v %v", plain, err)
	}
	request.CollectResourceUsage = true
	measured, err := hostexec.Run(t.Context(), request)
	if err != nil || measured.ResourceUsage == nil || measured.ResourceUsage.UserNanos+measured.ResourceUsage.SystemNanos == 0 || measured.ResourceUsage.PeakRSSBytes == 0 {
		t.Fatalf("actual child resource usage omitted: %+v %v", measured, err)
	}
}

type replySink struct {
	bytes.Buffer
	input *io.PipeWriter
	sent  bool
}

func (sink *replySink) Write(data []byte) (int, error) {
	n, err := sink.Buffer.Write(data)
	if err != nil {
		return n, err
	}
	if !sink.sent && strings.Contains(sink.String(), "ready") {
		sink.sent = true
		if _, err := sink.input.Write([]byte("answer\n")); err != nil {
			return n, err
		}
		if err := sink.input.Close(); err != nil {
			return n, err
		}
	}
	return n, nil
}
func TestPublicSupervisionProjectsDuplexOutcomeAndBoundsDiagnostics(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix supervision")
	}
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	sink := &replySink{input: writer}
	request := hostexec.Request{Command: []string{"/bin/sh", "-c", `printf ready; read value; printf '%s' "$value"; printf diagnostic >&2; exit 7`}, Dir: "/", Env: []string{}, Stdin: reader, StdoutSink: sink, Timeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 8}
	result, err := hostexec.Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 7 || !result.GroupGone || result.Cancelled || result.WatchdogTimeout || sink.String() != "readyanswer" || len(result.Stdout) != 8 || len(result.Stderr) != 8 {
		t.Fatalf("public result = %#v sink=%q", result, sink.String())
	}
}

func TestPublicSupervisionPreservesEmptyAndInheritedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix supervision")
	}
	t.Setenv("GOMAD_HOSTEXEC_ENV_SENTINEL", "ambient")
	for _, fixture := range []struct {
		name string
		env  []string
		want string
	}{{"empty", []string{}, ""}, {"inherited", nil, "ambient"}} {
		t.Run(fixture.name, func(t *testing.T) {
			result, err := hostexec.Run(context.Background(), hostexec.Request{Command: []string{"/bin/sh", "-c", `printf '%s' "$GOMAD_HOSTEXEC_ENV_SENTINEL"`}, Dir: "/", Env: fixture.env, Timeout: time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 1024})
			if err != nil || result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || string(result.Stdout) != fixture.want || !result.GroupGone {
				t.Fatalf("environment stdout=%q want=%q reaped=%v err=%v", result.Stdout, fixture.want, result.GroupGone, err)
			}
		})
	}
}
