package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.temporal.io/server/tests/umpire3/deployment"
	"go.temporal.io/server/tests/umpire3/deployment/canary"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type canaryConfig struct {
	experimentPath        string
	approvalPath          string
	approvalAuthority     string
	approvalPublicKeyPath string
	outputPath            string
	recoveryRoot          string
	endpoint              string
	namespace             string
	taskQueue             string
	buildID               string
	workerCommand         string
	resumeCleanup         bool
}

func RunCanary(ctx context.Context, arguments []string) error {
	configuration, err := parseCanaryFlags(arguments)
	if err != nil {
		return err
	}
	return runCanary(ctx, configuration)
}

func parseCanaryFlags(arguments []string) (canaryConfig, error) {
	var result canaryConfig
	flags := flag.NewFlagSet("umpire3-canary", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.experimentPath, "experiment", "", "approved experiment JSON")
	flags.StringVar(&result.approvalPath, "approval", "", "sealed approval JSON")
	flags.StringVar(&result.approvalAuthority, "approval-authority", "", "trusted approval authority identity")
	flags.StringVar(&result.approvalPublicKeyPath, "approval-public-key", "", "trusted Ed25519 approval public key")
	flags.StringVar(&result.outputPath, "output", "", "canary result JSON")
	flags.StringVar(&result.recoveryRoot, "recovery-dir", "", "durable recovery record directory")
	flags.StringVar(&result.endpoint, "endpoint", "", "Temporal HTTPS origin")
	flags.StringVar(&result.namespace, "namespace", "", "approved Temporal namespace")
	flags.StringVar(&result.taskQueue, "task-queue", "", "isolated canary task queue")
	flags.StringVar(&result.buildID, "build-id", "", "deployment build attestation")
	flags.StringVar(&result.workerCommand, "worker-command", "", "killable umpire3-canary-worker executable")
	flags.BoolVar(&result.resumeCleanup, "resume-cleanup", false, "resume cleanup from the durable recovery record")
	if err := flags.Parse(arguments); err != nil {
		return canaryConfig{}, err
	}
	if flags.NArg() != 0 {
		return canaryConfig{}, errors.New("unexpected positional arguments")
	}
	return result, nil
}

func runCanary(ctx context.Context, configuration canaryConfig) error {
	if err := validateCanaryConfig(configuration); err != nil {
		return err
	}
	experimentBytes, err := os.ReadFile(configuration.experimentPath)
	if err != nil {
		return fmt.Errorf("read canary experiment: %w", err)
	}
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(experimentBytes), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return err
	}
	approvalBytes, err := os.ReadFile(configuration.approvalPath)
	if err != nil {
		return fmt.Errorf("read canary approval: %w", err)
	}
	approval, err := decodeCanaryApproval(approvalBytes)
	if err != nil {
		return err
	}
	publicKeyBytes, err := os.ReadFile(configuration.approvalPublicKeyPath)
	if err != nil {
		return fmt.Errorf("read canary approval public key: %w", err)
	}
	if len(publicKeyBytes) > 16<<10 {
		return errors.New("canary approval public key exceeds 16384-byte limit")
	}
	publicKey, err := canary.ParseApprovalPublicKey(publicKeyBytes)
	if err != nil {
		return err
	}
	authority, err := canary.NewApprovalAuthority(configuration.approvalAuthority, publicKey)
	if err != nil {
		return err
	}
	definition, err := deployment.Define(deployment.Canary(
		configuration.endpoint, "environment-api-key", configuration.buildID,
		configuration.namespace, configuration.taskQueue, []string{configuration.workerCommand},
	))
	if err != nil {
		return err
	}
	controller := canary.Controller{
		Store: canary.NewFileStore(configuration.recoveryRoot), ApprovalAuthority: authority,
	}
	workerEnvironment := canaryWorkerEnvironment()
	var result canary.Result
	if configuration.resumeCleanup {
		result, err = controller.ResumeCleanup(ctx, definition, approval, workerEnvironment)
	} else {
		result, err = controller.Run(ctx, canary.Request{
			Experiment: experiment, Profile: definition, Approval: approval,
			WorkerEnvironment: workerEnvironment,
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

func canaryWorkerEnvironment() []string {
	credential, found := os.LookupEnv("UMPIRE3_TEMPORAL_API_KEY")
	if !found || credential == "" {
		return nil
	}
	return []string{"UMPIRE3_TEMPORAL_API_KEY=" + credential}
}

func validateCanaryConfig(configuration canaryConfig) error {
	if configuration.experimentPath == "" || configuration.approvalPath == "" ||
		configuration.approvalAuthority == "" || configuration.approvalPublicKeyPath == "" ||
		configuration.outputPath == "" || configuration.recoveryRoot == "" ||
		configuration.endpoint == "" || configuration.namespace == "" || configuration.taskQueue == "" ||
		configuration.buildID == "" || configuration.workerCommand == "" {
		return errors.New("experiment, approval, approval authority and public key, output, recovery directory, endpoint, namespace, task queue, build, and worker command are required")
	}
	return nil
}

func decodeCanaryApproval(encoded []byte) (canary.Approval, error) {
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
	if approval.FormatVersion != canary.FormatVersion || approval.Identifier == "" ||
		approval.ApprovalDigest == "" || approval.Signature == "" {
		return canary.Approval{}, errors.New("canary approval is incomplete or unsealed")
	}
	return approval, nil
}
