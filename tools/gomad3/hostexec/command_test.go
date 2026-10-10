package hostexec_test

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/hostexec"
)

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
