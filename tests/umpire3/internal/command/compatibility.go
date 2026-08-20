package command

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/qualification"
	umpire3runner "go.temporal.io/server/tests/umpire3/temporal"
)

type runCompatibilityOptions struct {
	experimentPath string
	outputPath     string
	address        string
	namespace      string
	taskQueue      string
	buildID        string
	profile        string
	nexusEndpoint  string
	nexusService   string
	nexusOperation string
	timeout        time.Duration
}

func RunCompatibility(ctx context.Context, arguments []string) error {
	return RunCompatibilityWithBackend(ctx, arguments, defaultBackend{})
}

func RunCompatibilityWithBackend(ctx context.Context, arguments []string, backend Backend) error {
	if backend == nil {
		return errors.New("command backend is required")
	}
	configuration, err := parseRunCompatibilityFlags(arguments)
	if err != nil {
		return err
	}
	options := runCompatibilityRunnerOptions(configuration)
	if _, err := umpire3runner.Validate(options); err != nil {
		return err
	}
	experiment, err := readExperiment(configuration.experimentPath)
	if err != nil {
		return err
	}
	result, err := backend.Execute(ctx, experiment, options)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode runtime result: %w", err)
	}
	if err := os.WriteFile(configuration.outputPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write runtime result: %w", err)
	}
	return nil
}

func parseRunCompatibilityFlags(arguments []string) (runCompatibilityOptions, error) {
	var result runCompatibilityOptions
	flags := flag.NewFlagSet("umpire3-run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.experimentPath, "experiment", "", "released experiment JSON")
	flags.StringVar(&result.outputPath, "output", "", "runtime result JSON")
	flags.StringVar(&result.address, "address", "", "Temporal gRPC address or HTTPS origin")
	flags.StringVar(&result.namespace, "namespace", "", "Temporal namespace")
	flags.StringVar(&result.taskQueue, "task-queue", "", "isolated participant task queue")
	flags.StringVar(&result.buildID, "build-id", "", "deployment build attestation")
	flags.StringVar(&result.profile, "profile", "", "local-in-process, ci-test-cluster, remote-deployment, or grpc-only-black-box")
	flags.StringVar(&result.nexusEndpoint, "nexus-endpoint", "", "optional Nexus endpoint")
	flags.StringVar(&result.nexusService, "nexus-service", "", "optional Nexus service")
	flags.StringVar(&result.nexusOperation, "nexus-operation", "", "optional Nexus operation")
	flags.DurationVar(&result.timeout, "timeout", 5*time.Minute, "execution and cleanup budget")
	if err := flags.Parse(arguments); err != nil {
		return runCompatibilityOptions{}, err
	}
	return result, nil
}

func runCompatibilityRunnerOptions(configuration runCompatibilityOptions) umpire3runner.Options {
	return umpire3runner.Options{
		Address: configuration.address, Namespace: configuration.namespace,
		TaskQueue: configuration.taskQueue, BuildID: configuration.buildID,
		Profile: configuration.profile, NexusEndpoint: configuration.nexusEndpoint,
		NexusService: configuration.nexusService, NexusOperation: configuration.nexusOperation,
		APIKey: os.Getenv("UMPIRE3_TEMPORAL_API_KEY"), Timeout: configuration.timeout,
	}
}

type qualificationCompatibilityOptions struct {
	releasePath    string
	experimentPath string
	resultPath     string
	profile        string
	outputPath     string
}

type receipt = qualification.Receipt

func QualifyCompatibility(arguments []string, stdout io.Writer) error {
	return QualifyCompatibilityWithBackend(arguments, stdout, defaultBackend{})
}

func QualifyCompatibilityWithBackend(arguments []string, stdout io.Writer, backend Backend) error {
	if backend == nil {
		return errors.New("command backend is required")
	}
	configuration, err := parseQualificationCompatibilityFlags(arguments)
	if err != nil {
		return err
	}
	return qualifyCompatibility(configuration, stdout, backend)
}

func qualifyCompatibility(configuration qualificationCompatibilityOptions, stdout io.Writer, backend Backend) error {
	if configuration.releasePath == "" || configuration.experimentPath == "" ||
		configuration.resultPath == "" || configuration.profile == "" {
		return errors.New("release, experiment, result, and profile are required")
	}
	releaseBytes, err := readRequiredFile("release", configuration.releasePath, protocol.DefaultDecodeLimit)
	if err != nil {
		return err
	}
	experimentBytes, err := readRequiredFile("experiment", configuration.experimentPath, protocol.DefaultDecodeLimit)
	if err != nil {
		return err
	}
	resultBytes, err := readRequiredFile("result", configuration.resultPath, protocol.DefaultDecodeLimit)
	if err != nil {
		return err
	}
	value, err := backend.Qualify(qualification.Request{
		ReleaseBytes: releaseBytes, ExperimentBytes: experimentBytes,
		ResultBytes: resultBytes, Profile: configuration.profile,
	})
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode qualification receipt: %w", err)
	}
	encoded = append(encoded, '\n')
	if configuration.outputPath != "" {
		if err := os.WriteFile(configuration.outputPath, encoded, 0o600); err != nil {
			return fmt.Errorf("write qualification receipt: %w", err)
		}
		return nil
	}
	_, err = stdout.Write(encoded)
	return err
}

func decodeResult(encoded []byte) (umpire3runtime.Result, error) {
	return qualification.DecodeResult(encoded)
}

func parseQualificationCompatibilityFlags(arguments []string) (qualificationCompatibilityOptions, error) {
	var result qualificationCompatibilityOptions
	flags := flag.NewFlagSet("umpire3-qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.releasePath, "release", "", "candidate release manifest")
	flags.StringVar(&result.experimentPath, "experiment", "", "executed semantic experiment")
	flags.StringVar(&result.resultPath, "result", "", "runtime or canary result artifact")
	flags.StringVar(&result.profile, "profile", "", "external profile being qualified")
	flags.StringVar(&result.outputPath, "output", "", "qualification receipt output")
	if err := flags.Parse(arguments); err != nil {
		return qualificationCompatibilityOptions{}, err
	}
	return result, nil
}
