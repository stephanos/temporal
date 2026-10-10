//go:build unix

package execution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

func TestRunIOTerminalAfterTermination(t *testing.T) {
	var terminal bytes.Buffer
	if err := deterministicio.WriteCompletion(&terminal, deterministicio.Transcript{Complete: true, SHA256: sha256.Sum256(nil)}); err != nil {
		t.Fatal(err)
	}
	complete := terminal.Bytes()
	checksum := append([]byte(nil), complete...)
	checksum[len(checksum)-1] ^= 1
	for _, test := range []struct {
		name, mode string
		frame      []byte
		message    string
	}{
		{name: "watchdog_absent", mode: "watchdog"},
		{name: "cancelled_absent", mode: "cancelled"},
		{name: "success_absent", mode: "0", message: "I/O transcript terminal is absent"},
		{name: "nonzero_absent", mode: "7", message: "I/O transcript terminal is absent"},
		{name: "unverified_signal_absent", mode: "signal", message: "I/O transcript terminal is absent"},
		{name: "watchdog_truncated", mode: "watchdog", frame: complete[:len(complete)-1], message: "invalid I/O terminal frame"},
		{name: "cancelled_truncated", mode: "cancelled", frame: complete[:len(complete)-1], message: "invalid I/O terminal frame"},
		{name: "watchdog_checksum", mode: "watchdog", frame: checksum, message: "I/O terminal frame checksum mismatch"},
		{name: "cancelled_checksum", mode: "cancelled", frame: checksum, message: "I/O terminal frame checksum mismatch"},
		{name: "success_complete", mode: "0", frame: complete},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			request := Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
				BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
				Command:           os.Args[0], Args: []string{"-test.run=^TestIOTerminalTargetHelper$", "--", test.mode, hex.EncodeToString(test.frame)}, Argv0: "gomad3-target", Dir: t.TempDir(),
				ExecutionTimeout: 2 * time.Second, TerminateGrace: 100 * time.Millisecond, OutputLimit: 64,
				World: WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20},
				IO:    &IOCapability{Config: []byte("profile-frame"), Transcript: &IOTranscriptCapability{Limit: 1 << 20}},
			}
			if test.mode == "cancelled" {
				request.StdoutHead = cancelOnOutput{cancel}
			}
			result, err := Run(ctx, request)
			if !result.GroupGone {
				t.Fatalf("target group survived: %#v, error %v", result, err)
			}
			if test.mode == "watchdog" && !result.WatchdogTimeout || test.mode == "cancelled" && !result.Cancelled {
				t.Fatalf("termination not verified: %#v, error %v", result, err)
			}
			if test.mode == "signal" && (result.WatchdogTimeout || result.Cancelled) {
				t.Fatalf("unverified signal relabeled: %#v", result)
			}
			if test.message != "" {
				if err == nil || !strings.Contains(err.Error(), test.message) {
					t.Fatalf("Run error = %v, want %s", err, test.message)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if test.name == "success_complete" {
				if !result.IOTranscript.Complete {
					t.Fatalf("complete transcript lost: %#v", result.IOTranscript)
				}
			} else if !reflect.DeepEqual(result.IOTranscript, deterministicio.Transcript{}) {
				t.Fatalf("incomplete transcript retained: %#v", result.IOTranscript)
			}
		})
	}
}

type cancelOnOutput struct{ cancel context.CancelFunc }

func (writer cancelOnOutput) Write(data []byte) (int, error) { writer.cancel(); return len(data), nil }

func TestIOTerminalTargetHelper(t *testing.T) {
	if len(os.Args) < 4 || os.Args[len(os.Args)-3] != "--" {
		t.Skip("I/O terminal subprocess only")
	}
	mode := os.Args[len(os.Args)-2]
	frame, err := hex.DecodeString(os.Args[len(os.Args)-1])
	if err != nil {
		t.Fatal(err)
	}
	signal.Ignore(syscall.SIGTERM)
	terminal := os.NewFile(uintptr(descriptorFor(targetStage, launchCapabilities{ioTranscript: true}, ioTerminalResource)), "io-terminal")
	if terminal == nil {
		t.Fatal("I/O terminal unavailable")
	}
	if _, err := terminal.Write(frame); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Close(); err != nil {
		t.Fatal(err)
	}
	if mode == "watchdog" || mode == "cancelled" {
		if _, err := fmt.Fprintln(os.Stdout, "ready"); err != nil {
			os.Exit(3)
		}
		for {
			<-time.After(time.Hour)
		}
	}
	if mode == "signal" {
		if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		return
	}
	status, err := strconv.Atoi(mode)
	if err != nil {
		t.Fatal(err)
	}
	os.Exit(status)
}
