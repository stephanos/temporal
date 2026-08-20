package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tests/umpire3/canary"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type config struct {
	experimentPath string
	approvalPath   string
	outputPath     string
	recoveryRoot   string
	endpoint       string
	namespace      string
	taskQueue      string
	buildID        string
	workerCommand  string
	resumeCleanup  bool
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
	flags := flag.NewFlagSet("umpire3-canary", flag.ContinueOnError)
	flags.StringVar(&result.experimentPath, "experiment", "", "approved experiment JSON")
	flags.StringVar(&result.approvalPath, "approval", "", "sealed approval JSON")
	flags.StringVar(&result.outputPath, "output", "", "canary result JSON")
	flags.StringVar(&result.recoveryRoot, "recovery-dir", "", "durable recovery record directory")
	flags.StringVar(&result.endpoint, "endpoint", "", "Temporal HTTPS origin")
	flags.StringVar(&result.namespace, "namespace", "", "approved Temporal namespace")
	flags.StringVar(&result.taskQueue, "task-queue", "", "isolated canary task queue")
	flags.StringVar(&result.buildID, "build-id", "", "deployment build attestation")
	flags.StringVar(&result.workerCommand, "worker-command", "", "killable umpire3-canary-worker executable")
	flags.BoolVar(&result.resumeCleanup, "resume-cleanup", false, "resume cleanup from the durable recovery record")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	return result, nil
}

func run(ctx context.Context, configuration config) error {
	if err := validateConfig(configuration); err != nil {
		return err
	}
	experimentBytes, err := os.ReadFile(configuration.experimentPath)
	if err != nil {
		return fmt.Errorf("read canary experiment: %w", err)
	}
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	if err != nil {
		return err
	}
	approvalBytes, err := os.ReadFile(configuration.approvalPath)
	if err != nil {
		return fmt.Errorf("read canary approval: %w", err)
	}
	approval, err := decodeApproval(approvalBytes)
	if err != nil {
		return err
	}
	definition, err := profile.Define(profile.Canary(
		configuration.endpoint, "environment-api-key", configuration.buildID,
		configuration.namespace, configuration.taskQueue, []string{configuration.workerCommand},
	))
	if err != nil {
		return err
	}
	controller := canary.Controller{Store: canary.NewFileStore(configuration.recoveryRoot)}
	var result canary.Result
	if configuration.resumeCleanup {
		result, err = controller.ResumeCleanup(ctx, definition, approval, nil)
	} else {
		result, err = controller.Run(ctx, canary.Request{
			Experiment: experiment, Profile: definition, Approval: approval,
		})
	}
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode canary result: %w", err)
	}
	if err := os.WriteFile(configuration.outputPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write canary result: %w", err)
	}
	return nil
}

func validateConfig(configuration config) error {
	if configuration.experimentPath == "" || configuration.approvalPath == "" ||
		configuration.outputPath == "" || configuration.recoveryRoot == "" ||
		configuration.endpoint == "" || configuration.namespace == "" || configuration.taskQueue == "" ||
		configuration.buildID == "" || configuration.workerCommand == "" {
		return errors.New("experiment, approval, output, recovery directory, endpoint, namespace, task queue, build, and worker command are required")
	}
	return nil
}

func decodeApproval(encoded []byte) (canary.Approval, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var approval canary.Approval
	if err := decoder.Decode(&approval); err != nil {
		return canary.Approval{}, fmt.Errorf("decode canary approval: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return canary.Approval{}, errors.New("canary approval must contain one JSON document")
	}
	if approval.FormatVersion != canary.FormatVersion || approval.Identifier == "" || approval.ApprovalDigest == "" {
		return canary.Approval{}, errors.New("canary approval is incomplete or unsealed")
	}
	return approval, nil
}
