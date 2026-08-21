package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("umpire3-veil", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operation := flags.String("operation", "generate", "generate or check")
	input := flags.String("input", "", "path to a FirstOrderView/v1 artifact")
	output := flags.String("output", "", "path for generated output")
	mode := flags.String("mode", string(veil.Interactive), "Veil job mode")
	trust := flags.String("smt-trust", string(veil.ReconstructedSMT), "SMT trust mode")
	backendCommand := flags.String("backend-command", "", "path to the Veil concrete checker executable")
	replayCommand := flags.String("replay-command", "", "path to the canonical Lean replay executable")
	job := flags.String("job", "", "symbolic-trace or invariant Veil job")
	jobCommand := flags.String("job-command", "", "path to the checked Veil job receipt executable")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *input == "" || *output == "" {
		return errors.New("input and output are required")
	}
	view, err := readFirstOrderView(*input)
	if err != nil {
		return err
	}

	switch *operation {
	case "generate":
		generated, err := veil.GenerateWithTrust(view, veil.Mode(*mode), veil.SMTTrustMode(*trust))
		if err != nil {
			return err
		}
		return writeOutput(*output, generated.Source)
	case "check-concrete":
		if *backendCommand == "" {
			return errors.New("backend-command is required for a checked Veil concrete job")
		}
		generated, err := veil.Generate(view, veil.Concrete)
		if err != nil {
			return err
		}
		var replay []string
		if *replayCommand != "" {
			replay = []string{*replayCommand}
		}
		result, err := veil.CheckConcrete(context.Background(), []string{*backendCommand}, replay,
			view, generated)
		if err != nil {
			return err
		}
		return writeBackendResult(*output, result)
	case "check-job":
		if *jobCommand == "" {
			return errors.New("job-command is required for a checked Veil job")
		}
		generated, err := veil.GenerateWithTrust(view, veil.Interactive, veil.SMTTrustMode(*trust))
		if err != nil {
			return err
		}
		result, err := veil.RunJob(context.Background(), []string{*jobCommand}, view, generated,
			protocol.BackendJob(*job))
		if err != nil {
			return err
		}
		return writeBackendResult(*output, result)
	default:
		return fmt.Errorf("unknown operation %q", *operation)
	}
}

func writeBackendResult(path string, result protocol.BackendResult) error {
	encoded, err := result.CanonicalJSON()
	if err != nil {
		return err
	}
	return writeOutput(path, append(encoded, '\n'))
}

func readFirstOrderView(path string) (protocol.FirstOrderView, error) {
	input, err := os.Open(path)
	if err != nil {
		return protocol.FirstOrderView{}, fmt.Errorf("open first-order view: %w", err)
	}
	view, decodeErr := protocol.DecodeFirstOrderView(input, protocol.DefaultDecodeLimit)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return protocol.FirstOrderView{}, fmt.Errorf("decode first-order view: %w", errors.Join(decodeErr, closeErr))
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
