package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func RunNative(arguments []string) error {
	flags := flag.NewFlagSet("umpire3-native", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	operation := flags.String("operation", "produce", "bind, produce, check, benchmark, or validate-benchmark")
	input := flags.String("input", "", "path to a FirstOrderView/v2 artifact")
	output := flags.String("output", "", "path for generated output")
	certificatePath := flags.String("certificate", "", "path to a native certificate")
	receiptPath := flags.String("receipt", "", "path to a checked native certificate receipt")
	benchmarkPath := flags.String("benchmark", "", "path to a native benchmark report")
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
	if *input == "" {
		return errors.New("input is required")
	}
	if *operation != "validate-benchmark" && *output == "" {
		return errors.New("output is required")
	}
	view, err := readNativeFirstOrderView(*input)
	if err != nil {
		return err
	}

	switch *operation {
	case "bind":
		source, err := finite.BindingSource(view)
		if err != nil {
			return err
		}
		return finite.WriteArtifact(*output, source)
	case "produce":
		options := finite.Options{
			Workers: *workers, Replicas: *replicas,
			Limits: finite.SearchLimits{
				MaxDepth: *maxDepth, MaxStates: *maxStates,
				MaxTransitions: *maxTransitions, MaxStateBytes: *maxStateBytes,
			},
		}
		if *checkpointPath != "" {
			options.Checkpoint = func(checkpoint finite.Checkpoint) error {
				return finite.SaveCheckpoint(*checkpointPath, checkpoint)
			}
		}
		var checkpoint *finite.Checkpoint
		if *resume {
			if *checkpointPath == "" {
				return errors.New("checkpoint is required for resume")
			}
			loaded, err := finite.LoadCheckpoint(*checkpointPath, protocolexperiment.DefaultDecodeLimit)
			if err != nil {
				return err
			}
			checkpoint = &loaded
		}
		certificate, err := finite.Produce(context.Background(), view, options, checkpoint)
		if err != nil {
			return err
		}
		encoded, err := certificate.CanonicalJSON(view)
		if err != nil {
			return err
		}
		return finite.WriteArtifact(*output, append(encoded, '\n'))
	case "check":
		if *certificatePath == "" || *checkerCommand == "" {
			return errors.New("certificate and checker-command are required for check")
		}
		certificate, err := readNativeCertificate(*certificatePath, view)
		if err != nil {
			return err
		}
		receipt, err := finite.CheckCertificate(context.Background(), []string{*checkerCommand}, view, certificate)
		if err != nil {
			return err
		}
		encoded, err := receipt.CanonicalJSON(certificate)
		if err != nil {
			return err
		}
		return finite.WriteArtifact(*output, append(encoded, '\n'))
	case "benchmark":
		if *certificatePath == "" || *receiptPath == "" || *checkerCommand == "" {
			return errors.New("certificate, receipt, and checker-command are required for benchmark")
		}
		if *replicas != 10 {
			return errors.New("the native scale benchmark requires exactly 10 replicas")
		}
		certificate, err := readNativeCertificate(*certificatePath, view)
		if err != nil {
			return err
		}
		receipt, err := readNativeReceipt(*receiptPath, certificate)
		if err != nil {
			return err
		}
		report, _, _, err := finite.Benchmark(context.Background(), view, finite.BenchmarkOptions{
			ParallelWorkers: *workers,
			Limits: finite.SearchLimits{
				MaxDepth: *maxDepth, MaxStates: *maxStates,
				MaxTransitions: *maxTransitions, MaxStateBytes: *maxStateBytes,
			},
			CheckerCommand: []string{*checkerCommand},
		})
		if err != nil {
			return err
		}
		encoded, err := report.CanonicalJSON(view, certificate, receipt)
		if err != nil {
			return err
		}
		return finite.WriteArtifact(*output, append(encoded, '\n'))
	case "validate-benchmark":
		if *certificatePath == "" || *receiptPath == "" || *benchmarkPath == "" {
			return errors.New("certificate, receipt, and benchmark are required for validation")
		}
		certificate, err := readNativeCertificate(*certificatePath, view)
		if err != nil {
			return err
		}
		receipt, err := readNativeReceipt(*receiptPath, certificate)
		if err != nil {
			return err
		}
		_, err = readNativeBenchmarkReport(*benchmarkPath, view, certificate, receipt)
		return err
	default:
		return fmt.Errorf("unknown operation %q", *operation)
	}
}

func readNativeFirstOrderView(path string) (protocolchecker.FirstOrderView, error) {
	input, err := os.Open(path)
	if err != nil {
		return protocolchecker.FirstOrderView{}, fmt.Errorf("open first-order view: %w", err)
	}
	view, decodeErr := protocolchecker.DecodeFirstOrderView(input, protocolexperiment.DefaultDecodeLimit)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return protocolchecker.FirstOrderView{}, errors.Join(decodeErr, closeErr)
	}
	return view, nil
}

func readNativeCertificate(path string, view protocolchecker.FirstOrderView) (finite.Certificate, error) {
	input, err := os.Open(path)
	if err != nil {
		return finite.Certificate{}, fmt.Errorf("open native certificate: %w", err)
	}
	certificate, decodeErr := finite.DecodeCertificate(input, protocolexperiment.DefaultDecodeLimit, view)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return finite.Certificate{}, errors.Join(decodeErr, closeErr)
	}
	return certificate, nil
}

func readNativeReceipt(path string, certificate finite.Certificate) (finite.Receipt, error) {
	input, err := os.Open(path)
	if err != nil {
		return finite.Receipt{}, fmt.Errorf("open native certificate receipt: %w", err)
	}
	receipt, decodeErr := finite.DecodeReceipt(input, protocolexperiment.DefaultDecodeLimit, certificate)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return finite.Receipt{}, errors.Join(decodeErr, closeErr)
	}
	return receipt, nil
}

func readNativeBenchmarkReport(
	path string,
	view protocolchecker.FirstOrderView,
	certificate finite.Certificate,
	receipt finite.Receipt,
) (finite.BenchmarkReport, error) {
	input, err := os.Open(path)
	if err != nil {
		return finite.BenchmarkReport{}, fmt.Errorf("open native benchmark report: %w", err)
	}
	report, decodeErr := finite.DecodeBenchmarkReport(
		input, protocolexperiment.DefaultDecodeLimit, view, certificate, receipt)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return finite.BenchmarkReport{}, errors.Join(decodeErr, closeErr)
	}
	return report, nil
}
