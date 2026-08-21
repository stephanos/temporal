package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tests/umpire3/model-checkers/native"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("umpire3-native", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operation := flags.String("operation", "produce", "bind, produce, or check")
	input := flags.String("input", "", "path to a FirstOrderView/v2 artifact")
	output := flags.String("output", "", "path for generated output")
	certificatePath := flags.String("certificate", "", "path to a native certificate")
	checkerCommand := flags.String("checker-command", "", "path to canonical Lean certificate checker")
	checkpointPath := flags.String("checkpoint", "", "optional transactional checkpoint path")
	resume := flags.Bool("resume", false, "resume from the checkpoint path")
	workers := flags.Int("workers", 8, "parallel producer workers")
	replicas := flags.Int("replicas", 10, "disjoint scale worlds")
	maxDepth := flags.Int("max-depth", 32, "search depth limit")
	maxStates := flags.Int("max-states", 1024, "expanded state limit")
	maxTransitions := flags.Int("max-transitions", 16384, "expanded transition limit")
	maxStateBytes := flags.Int("max-state-bytes", 1<<20, "expanded state storage limit")
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
	case "bind":
		source, err := native.BindingSource(view)
		if err != nil {
			return err
		}
		return native.WriteArtifact(*output, source)
	case "produce":
		options := native.Options{
			Workers: *workers, Replicas: *replicas,
			Limits: native.SearchLimits{
				MaxDepth: *maxDepth, MaxStates: *maxStates,
				MaxTransitions: *maxTransitions, MaxStateBytes: *maxStateBytes,
			},
		}
		if *checkpointPath != "" {
			options.Checkpoint = func(checkpoint native.Checkpoint) error {
				return native.SaveCheckpoint(*checkpointPath, checkpoint)
			}
		}
		var checkpoint *native.Checkpoint
		if *resume {
			if *checkpointPath == "" {
				return errors.New("checkpoint is required for resume")
			}
			loaded, err := native.LoadCheckpoint(*checkpointPath, protocol.DefaultDecodeLimit)
			if err != nil {
				return err
			}
			checkpoint = &loaded
		}
		certificate, err := native.Produce(context.Background(), view, options, checkpoint)
		if err != nil {
			return err
		}
		encoded, err := certificate.CanonicalJSON(view)
		if err != nil {
			return err
		}
		return native.WriteArtifact(*output, append(encoded, '\n'))
	case "check":
		if *certificatePath == "" || *checkerCommand == "" {
			return errors.New("certificate and checker-command are required for check")
		}
		certificate, err := readCertificate(*certificatePath, view)
		if err != nil {
			return err
		}
		receipt, err := native.CheckCertificate(context.Background(), []string{*checkerCommand}, view, certificate)
		if err != nil {
			return err
		}
		encoded, err := receipt.CanonicalJSON(certificate)
		if err != nil {
			return err
		}
		return native.WriteArtifact(*output, append(encoded, '\n'))
	default:
		return fmt.Errorf("unknown operation %q", *operation)
	}
}

func readFirstOrderView(path string) (protocol.FirstOrderView, error) {
	input, err := os.Open(path)
	if err != nil {
		return protocol.FirstOrderView{}, fmt.Errorf("open first-order view: %w", err)
	}
	view, decodeErr := protocol.DecodeFirstOrderView(input, protocol.DefaultDecodeLimit)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return protocol.FirstOrderView{}, errors.Join(decodeErr, closeErr)
	}
	return view, nil
}

func readCertificate(path string, view protocol.FirstOrderView) (native.Certificate, error) {
	input, err := os.Open(path)
	if err != nil {
		return native.Certificate{}, fmt.Errorf("open native certificate: %w", err)
	}
	certificate, decodeErr := native.DecodeCertificate(input, protocol.DefaultDecodeLimit, view)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return native.Certificate{}, errors.Join(decodeErr, closeErr)
	}
	return certificate, nil
}
