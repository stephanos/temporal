package measure

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

func TestTerminalExitFailureRetainsObservationStderrAndAggregate(t *testing.T) {
	t.Parallel()
	scope := &runScope{options: Options{Log: io.Discard}}
	ctx := context.WithValue(t.Context(), runKey{}, scope)
	evidence := t.TempDir()
	executionError := errors.New("helper transport failed")
	var calls, verified int
	execute := func(context.Context, wasi.Request, bool) (wasi.TerminalExitObservation, error) {
		calls++
		return wasi.TerminalExitObservation{Error: "helper transport failed", TerminalFrame: []byte("partial terminal"), Host: hostexec.Result{Stderr: hostexec.Output{RawBytes: []byte("helper diagnostics")}}}, executionError
	}
	report := func(observations []wasi.TerminalExitObservation) error {
		return temporalJSON(ctx, filepath.Join(evidence, "terminal-exit.json"), map[string]any{"observations": observations})
	}
	err := terminalExitCollect(ctx, evidence, wasi.Request{}, execute, func() error { verified++; return nil }, report)
	if !errors.Is(err, executionError) {
		t.Fatalf("execution error lost: %v", err)
	}
	for _, name := range []string{"success", "post-load-authorization-mismatch"} {
		data, err := os.ReadFile(filepath.Join(evidence, name+"-observation.json"))
		if err != nil {
			t.Fatal(err)
		}
		var observation wasi.TerminalExitObservation
		if err := json.Unmarshal(data, &observation); err != nil {
			t.Fatal(err)
		}
		if observation.Error != "helper transport failed" || string(observation.TerminalFrame) != "partial terminal" {
			t.Fatalf("observation lost: %+v", observation)
		}
		stderr, err := os.ReadFile(filepath.Join(evidence, name+"-helper-stderr.txt"))
		if err != nil || string(stderr) != "helper diagnostics" {
			t.Fatalf("stderr=%q error=%v", stderr, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(evidence, "terminal-exit.json"))
	if err != nil {
		t.Fatal(err)
	}
	var aggregate struct {
		Observations []wasi.TerminalExitObservation
	}
	if err := json.Unmarshal(data, &aggregate); err != nil {
		t.Fatal(err)
	}
	if len(aggregate.Observations) != 2 || calls != 2 || verified != 2 {
		t.Fatalf("observations=%d executions=%d identity checks=%d", len(aggregate.Observations), calls, verified)
	}
}

func TestWarmPairNotificationsLaunchAndJoinBothGuests(t *testing.T) {
	for _, warmed := range []bool{false, true} {
		name := "first-finished"
		if warmed {
			name = "warmed"
		}
		t.Run(name, func(t *testing.T) {
			scope := &runScope{options: Options{Log: io.Discard}}
			ctx := context.WithValue(t.Context(), runKey{}, scope)
			evidence := t.TempDir()
			secondStarted, secondFinished := make(chan struct{}), make(chan struct{})
			firstFinished := make(chan struct{})
			requests := []wasi.Request{
				{Broker: &wasi.BrokerSession{Node: wasi.BrokerNode{ID: "first"}}},
				{Broker: &wasi.BrokerSession{Node: wasi.BrokerNode{ID: "second"}}},
			}
			execute := func(ctx context.Context, request wasi.Request) (wasi.Result, error) {
				if request.Broker.Node.ID == "first" {
					defer close(firstFinished)
					if warmed {
						if _, err := request.StdoutMirror.Write([]byte("ready")); err != nil {
							return wasi.Result{}, err
						}
						select {
						case <-secondStarted:
						case <-ctx.Done():
							return wasi.Result{}, ctx.Err()
						}
					}
				} else {
					close(secondStarted)
					defer close(secondFinished)
				}
				code := uint32(0)
				return wasi.Result{Termination: "exit", ExitCode: &code, GoFilesystemCalls: map[string]uint64{}, TranscriptSHA256: [32]byte{1}}, nil
			}
			results, failures, err := temporalRunPairWith(ctx, evidence, "pair", requests, true, execute)
			if err != nil || len(results) != 2 || len(failures) != 2 {
				t.Fatalf("pair results=%d failures=%d error=%v", len(results), len(failures), err)
			}
			for _, finished := range []chan struct{}{firstFinished, secondFinished} {
				select {
				case <-finished:
				default:
					t.Fatal("returned before both launched guests completed")
				}
			}
			for index := range results {
				if failures[index] != nil || results[index].Termination != "exit" {
					t.Fatalf("guest %d: %s %v", index, results[index].Termination, failures[index])
				}
			}
			for _, name := range []string{"first-pair.stdout", "second-pair.stdout", "pair-host-budget.json"} {
				if _, err := os.Stat(filepath.Join(evidence, name)); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
