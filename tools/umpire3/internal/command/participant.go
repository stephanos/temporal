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

	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	umpire3temporal "go.temporal.io/server/tools/umpire3/adapter/temporal"
	"go.temporal.io/server/tools/umpire3/execution/participant"
)

const participantReportFormatVersion = "umpire3/participant-report/v1"

type participantConfig struct {
	programPath    string
	outputPath     string
	address        string
	namespace      string
	taskQueue      string
	workflowID     string
	nexusEndpoint  string
	nexusService   string
	nexusOperation string
	timeout        time.Duration
}

type participantReport struct {
	FormatVersion   string               `json:"formatVersion"`
	ProgramID       string               `json:"programID"`
	Results         []participant.Result `json:"results"`
	CleanupComplete bool                 `json:"cleanupComplete"`
}

func RunParticipant(parent context.Context, arguments []string) error {
	configuration, err := parseParticipantFlags(arguments)
	if err != nil {
		return err
	}
	return runParticipant(parent, configuration)
}

func parseParticipantFlags(arguments []string) (participantConfig, error) {
	var result participantConfig
	flags := flag.NewFlagSet("umpire3-participant", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.programPath, "program", "", "participant program JSON")
	flags.StringVar(&result.outputPath, "output", "", "redacted execution report JSON")
	flags.StringVar(&result.address, "address", "", "Temporal gRPC address or HTTPS origin")
	flags.StringVar(&result.namespace, "namespace", "", "Temporal namespace")
	flags.StringVar(&result.taskQueue, "task-queue", "", "isolated participant task queue")
	flags.StringVar(&result.workflowID, "workflow-id", "", "participant workflow identity")
	flags.StringVar(&result.nexusEndpoint, "nexus-endpoint", "", "optional Nexus endpoint")
	flags.StringVar(&result.nexusService, "nexus-service", "", "optional Nexus service")
	flags.StringVar(&result.nexusOperation, "nexus-operation", "", "optional Nexus operation")
	flags.DurationVar(&result.timeout, "timeout", 5*time.Minute, "hard command and cleanup budget")
	if err := flags.Parse(arguments); err != nil {
		return participantConfig{}, err
	}
	if flags.NArg() != 0 {
		return participantConfig{}, errors.New("unexpected positional arguments")
	}
	return result, nil
}

func runParticipant(parent context.Context, configuration participantConfig) error {
	if err := validateParticipantConfig(configuration); err != nil {
		return err
	}
	program, err := loadParticipantProgram(configuration.programPath)
	if err != nil {
		return err
	}
	clientOptions, err := umpire3temporal.ClientOptions(configuration.address, configuration.namespace,
		"umpire3-participant/"+configuration.workflowID, os.Getenv("UMPIRE3_TEMPORAL_API_KEY"))
	if err != nil {
		return err
	}
	client, err := sdkclient.Dial(clientOptions)
	if err != nil {
		return fmt.Errorf("dial Temporal: %w", err)
	}
	defer client.Close()
	sdkWorker := worker.New(client, configuration.taskQueue, worker.Options{})
	runner, err := umpire3temporal.NewSDKParticipantAdapter(umpire3temporal.SDKParticipantOptions{
		Client: client, Registry: sdkWorker, Namespace: configuration.namespace, TaskQueue: configuration.taskQueue,
		WorkflowID: configuration.workflowID, CleanupTimeout: configuration.timeout,
		NexusEndpoint: configuration.nexusEndpoint, NexusService: configuration.nexusService,
		NexusOperation: configuration.nexusOperation,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, configuration.timeout)
	defer cancel()
	session, err := participant.Start(ctx, program, runner)
	if err != nil {
		return err
	}
	if err := sdkWorker.Start(); err != nil {
		cleanupErr := session.Cleanup(ctx)
		return errors.Join(fmt.Errorf("start SDK worker: %w", err), cleanupErr)
	}
	defer sdkWorker.Stop()
	result := participantReport{FormatVersion: participantReportFormatVersion, ProgramID: program.Identifier}
	for _, command := range program.Commands {
		receipt, executeErr := session.Execute(ctx, command.Identifier)
		if executeErr != nil {
			cleanupErr := session.Cleanup(ctx)
			return errors.Join(executeErr, cleanupErr)
		}
		result.Results = append(result.Results, receipt)
	}
	if err := session.Cleanup(ctx); err != nil {
		return err
	}
	result.CleanupComplete = true
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode participant report: %w", err)
	}
	if err := os.WriteFile(configuration.outputPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write participant report: %w", err)
	}
	return nil
}

func validateParticipantConfig(configuration participantConfig) error {
	if configuration.programPath == "" || configuration.outputPath == "" || configuration.address == "" ||
		configuration.namespace == "" || configuration.taskQueue == "" || configuration.workflowID == "" {
		return errors.New("program, output, address, namespace, task queue, and workflow ID are required")
	}
	if configuration.timeout <= 0 {
		return errors.New("participant timeout must be positive")
	}
	nexusValues := 0
	for _, value := range []string{configuration.nexusEndpoint, configuration.nexusService, configuration.nexusOperation} {
		if value != "" {
			nexusValues++
		}
	}
	if nexusValues != 0 && nexusValues != 3 {
		return errors.New("nexus endpoint, service, and operation must be supplied together")
	}
	return nil
}

func loadParticipantProgram(path string) (participant.Program, error) {
	input, err := os.Open(path)
	if err != nil {
		return participant.Program{}, fmt.Errorf("open participant program: %w", err)
	}
	defer func() { _ = input.Close() }()
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var program participant.Program
	if err := decoder.Decode(&program); err != nil {
		return participant.Program{}, fmt.Errorf("decode participant program: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return participant.Program{}, errors.New("participant program must contain one JSON document")
	}
	if _, err := participant.Compile(program); err != nil {
		return participant.Program{}, fmt.Errorf("validate participant program: %w", err)
	}
	return program, nil
}
