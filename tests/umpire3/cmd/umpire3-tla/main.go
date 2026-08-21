//go:build umpire3_tla_experiment

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/server/tests/umpire3/model-checkers/tla"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("umpire3-tla", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operation := flags.String("operation", "generate", "generate or check")
	input := flags.String("input", "", "path to a TemporalView/v1 artifact")
	output := flags.String("output", "", "path for generated output")
	configOutput := flags.String("config-output", "", "path for generated TLC configuration")
	backend := flags.String("backend", "", "tlc or apalache")
	javaPath := flags.String("java", os.Getenv("UMPIRE_JAVA_TOOL"), "path to pinned Java executable")
	tlaJar := flags.String("tla-jar", os.Getenv("UMPIRE_TLA_JAR"), "path to pinned tla2tools.jar")
	apalachePath := flags.String("apalache", os.Getenv("UMPIRE_APALACHE_TOOL"), "path to pinned apalache-mc")
	replayCommand := flags.String("replay-command", "", "path to canonical Lean temporal replay executable")
	timeout := flags.Duration("timeout", 30*time.Second, "external checker wall-clock limit")
	maxOutputBytes := flags.Int64("max-output-bytes", 4<<20, "external checker output limit")
	cpuSeconds := flags.Int("cpu-seconds", 30, "external checker CPU limit")
	memoryBytes := flags.Int64("memory-bytes", 2<<30, "external checker memory limit")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *input == "" || *output == "" {
		return errors.New("input and output are required")
	}
	view, err := readTemporalView(*input)
	if err != nil {
		return err
	}

	switch *operation {
	case "generate":
		if *configOutput == "" {
			return errors.New("config-output is required for TLA+ generation")
		}
		generated, err := tla.Generate(view)
		if err != nil {
			return err
		}
		if err := writeOutput(*output, generated.TLA); err != nil {
			return err
		}
		return writeOutput(*configOutput, generated.Config)
	case "check":
		limits := tla.ToolLimits{
			Timeout: *timeout, MaxOutputBytes: *maxOutputBytes,
			CPUSeconds: *cpuSeconds, MemoryBytes: *memoryBytes,
		}
		return check(context.Background(), view, tla.Backend(*backend), *javaPath, *tlaJar,
			*apalachePath, *replayCommand, limits, *output)
	default:
		return fmt.Errorf("unknown operation %q", *operation)
	}
}

func check(
	ctx context.Context,
	view protocol.TemporalView,
	backend tla.Backend,
	javaPath string,
	tlaJar string,
	apalachePath string,
	replayCommand string,
	limits tla.ToolLimits,
	output string,
) error {
	var result tla.Result
	switch backend {
	case tla.BackendTLC:
		raw, err := tla.CheckTLC(ctx, view, javaPath, tlaJar, limits)
		if err != nil {
			return err
		}
		result, err = tla.NormalizeTLC(view, raw)
		if err != nil {
			return err
		}
		if result.Lasso != nil {
			if replayCommand == "" {
				return errors.New("replay-command is required for a TLC temporal counterexample")
			}
			input := protocol.TemporalLassoReplayInput{
				FormatVersion: protocol.TemporalLassoReplayInputFormatVersion,
				Target:        view.Target, Property: view.Property, World: view.World,
				Variant: view.Variant, SemanticHash: view.SemanticHash, Lasso: *result.Lasso,
			}
			receipt, err := tla.ReplayLasso(ctx, []string{replayCommand}, input)
			if err != nil {
				return err
			}
			result, err = tla.AttachReplay(result, receipt)
			if err != nil {
				return err
			}
		}
	case tla.BackendApalache:
		raw, err := tla.CheckApalache(ctx, view, apalachePath, limits)
		if err != nil {
			return err
		}
		result, err = tla.NormalizeApalache(view, raw)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown temporal backend %q", backend)
	}
	encoded, err := result.CanonicalJSON(view)
	if err != nil {
		return err
	}
	return writeOutput(output, append(encoded, '\n'))
}

func readTemporalView(path string) (protocol.TemporalView, error) {
	input, err := os.Open(path)
	if err != nil {
		return protocol.TemporalView{}, fmt.Errorf("open temporal view: %w", err)
	}
	encoded, readErr := io.ReadAll(io.LimitReader(input, protocol.DefaultDecodeLimit+1))
	closeErr := input.Close()
	if readErr != nil || closeErr != nil {
		return protocol.TemporalView{}, fmt.Errorf("read temporal view: %w", errors.Join(readErr, closeErr))
	}
	if int64(len(encoded)) > protocol.DefaultDecodeLimit {
		return protocol.TemporalView{}, fmt.Errorf("temporal view exceeds %d-byte limit", protocol.DefaultDecodeLimit)
	}
	view, err := protocol.DecodeTemporalView(encoded)
	if err != nil {
		return protocol.TemporalView{}, fmt.Errorf("decode temporal view: %w", err)
	}
	return view, nil
}

func writeOutput(path string, value []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(path, value, 0o600); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
