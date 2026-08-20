package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runner "go.temporal.io/server/tests/umpire3/runner"
)

type config struct {
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

func main() {
	configuration, err := parseFlags(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := run(context.Background(), configuration); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFlags(arguments []string) (config, error) {
	var result config
	flags := flag.NewFlagSet("umpire3-run", flag.ContinueOnError)
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
		return config{}, err
	}
	return result, nil
}

func run(parent context.Context, configuration config) (retErr error) {
	options := runnerOptions(configuration)
	if _, err := umpire3runner.Validate(options); err != nil {
		return err
	}
	input, err := os.Open(configuration.experimentPath)
	if err != nil {
		return fmt.Errorf("open experiment: %w", err)
	}
	experiment, decodeErr := protocol.DecodeExperiment(input, protocol.DefaultDecodeLimit)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return errors.Join(decodeErr, closeErr)
	}
	result, err := umpire3runner.Execute(parent, experiment, options)
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

func runnerOptions(configuration config) umpire3runner.Options {
	return umpire3runner.Options{
		Address: configuration.address, Namespace: configuration.namespace,
		TaskQueue: configuration.taskQueue, BuildID: configuration.buildID,
		Profile: configuration.profile, NexusEndpoint: configuration.nexusEndpoint,
		NexusService: configuration.nexusService, NexusOperation: configuration.nexusOperation,
		APIKey: os.Getenv("UMPIRE3_TEMPORAL_API_KEY"), Timeout: configuration.timeout,
	}
}
