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
	operation := flags.String("operation", "", "check-concrete or check-job")
	input := flags.String("input", "", "path to a FirstOrderView/v2 artifact")
	bindingPath := flags.String("binding", "", "path to a checked Veil binding artifact")
	output := flags.String("output", "", "path for generated output")
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
	if *operation != "check-concrete" && *operation != "check-job" {
		return fmt.Errorf("unknown operation %q", *operation)
	}
	if *input == "" || *output == "" {
		return errors.New("input and output are required")
	}
	view, err := readFirstOrderView(*input)
	if err != nil {
		return err
	}

	switch *operation {
	case "check-concrete":
		if *backendCommand == "" {
			return errors.New("backend-command is required for a checked Veil concrete job")
		}
		binding, err := readBinding(*bindingPath)
		if err != nil {
			return err
		}
		var replay []string
		if *replayCommand != "" {
			replay = []string{*replayCommand}
		}
		result, err := veil.CheckConcrete(context.Background(), []string{*backendCommand}, replay,
			view, binding)
		if err != nil {
			return err
		}
		return writeBackendResult(*output, result)
	case "check-job":
		if *jobCommand == "" {
			return errors.New("job-command is required for a checked Veil job")
		}
		binding, err := readBinding(*bindingPath)
		if err != nil {
			return err
		}
		result, err := veil.RunJob(context.Background(), []string{*jobCommand}, view, binding,
			protocol.BackendJob(*job))
		if err != nil {
			return err
		}
		return writeBackendResult(*output, result)
	}
	return nil
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

func readBinding(path string) (veil.BindingArtifact, error) {
	if path == "" {
		return veil.BindingArtifact{}, errors.New("binding is required for a checked Veil job")
	}
	input, err := os.Open(path)
	if err != nil {
		return veil.BindingArtifact{}, fmt.Errorf("open Veil binding: %w", err)
	}
	binding, decodeErr := veil.DecodeBindingArtifact(input, protocol.DefaultDecodeLimit)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return veil.BindingArtifact{}, fmt.Errorf("decode Veil binding: %w", errors.Join(decodeErr, closeErr))
	}
	return binding, nil
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
