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
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/artifact"
	"go.temporal.io/server/tests/umpire3/campaign"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/qualification"
	"go.temporal.io/server/tests/umpire3/replay"
	umpire3runner "go.temporal.io/server/tests/umpire3/runner"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

const diagnosticFormatVersion = "umpire3/diagnostic/v1"

type diagnostic struct {
	FormatVersion string `json:"formatVersion"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Data          any    `json:"data,omitempty"`
	Error         string `json:"error,omitempty"`
}

type connectionFlags struct {
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

type explanation struct {
	ExperimentDigest     string                     `json:"experimentDigest"`
	ExperimentID         string                     `json:"experimentID"`
	Property             string                     `json:"property"`
	ModelModules         []string                   `json:"modelModules"`
	RequiredCapabilities []string                   `json:"requiredCapabilities"`
	Resources            []protocol.Resource        `json:"resources"`
	Actions              []protocol.Action          `json:"actions"`
	Faults               []protocol.Fault           `json:"faults"`
	Order                []protocol.OrderConstraint `json:"order"`
	Checkpoints          []protocol.Checkpoint      `json:"checkpoints"`
	Seed                 int64                      `json:"seed"`
	Bounds               protocol.Bounds            `json:"bounds"`
}

type campaignExecution struct {
	Kind   campaign.MutationKind `json:"kind"`
	Path   string                `json:"path"`
	Digest string                `json:"digest"`
	Result umpire3runtime.Result `json:"result"`
}

type campaignOutput struct {
	Mutation campaign.MutationReport `json:"mutation"`
	Runs     []campaignExecution     `json:"runs"`
}

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	data, err := execute(context.Background(), os.Args[1:])
	status := "ok"
	message := ""
	if err != nil {
		status = "error"
		message = err.Error()
	}
	encoded, encodeErr := json.MarshalIndent(diagnostic{
		FormatVersion: diagnosticFormatVersion, Command: command, Status: status, Data: data, Error: message,
	}, "", "  ")
	if encodeErr != nil {
		_, _ = fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	output := os.Stdout
	if err != nil {
		output = os.Stderr
	}
	_, _ = output.Write(append(encoded, '\n'))
	if err != nil {
		os.Exit(1)
	}
}

func execute(ctx context.Context, arguments []string) (any, error) {
	if len(arguments) == 0 {
		return nil, errors.New("command is required: explain, run, replay, campaign, or qualify")
	}
	switch arguments[0] {
	case "explain":
		return executeExplain(arguments[1:])
	case "run":
		return executeRun(ctx, arguments[1:])
	case "replay":
		return executeReplay(ctx, arguments[1:])
	case "campaign":
		return executeCampaign(ctx, arguments[1:])
	case "qualify":
		return executeQualify(arguments[1:])
	default:
		return nil, fmt.Errorf("unknown command %q: expected explain, run, replay, campaign, or qualify", arguments[0])
	}
}

func executeExplain(arguments []string) (any, error) {
	flags := flag.NewFlagSet("explain", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	experimentPath := flags.String("experiment", "", "released experiment JSON")
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	experiment, err := readExperiment(*experimentPath)
	if err != nil {
		return nil, err
	}
	digest, err := experiment.Digest()
	if err != nil {
		return nil, err
	}
	capabilities := make(map[string]struct{})
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			capabilities[capability] = struct{}{}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			capabilities[capability] = struct{}{}
		}
	}
	required := make([]string, 0, len(capabilities))
	for capability := range capabilities {
		required = append(required, capability)
	}
	slices.Sort(required)
	return explanation{
		ExperimentDigest: digest, ExperimentID: experiment.ExperimentID,
		Property: experiment.Property.Identifier, ModelModules: experiment.Model.Modules,
		RequiredCapabilities: required, Resources: experiment.Resources, Actions: experiment.Actions,
		Faults: experiment.Faults, Order: experiment.Order, Checkpoints: experiment.Checkpoints,
		Seed: experiment.Scope.Seed, Bounds: experiment.Scope.Bounds,
	}, nil
}

func executeRun(ctx context.Context, arguments []string) (any, error) {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	experimentPath := flags.String("experiment", "", "released experiment JSON")
	outputPath := flags.String("output", "", "optional raw runtime result output")
	bundlePath := flags.String("bundle-output", "", "optional redacted replay bundle output")
	connection := addConnectionFlags(flags)
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	experiment, err := readExperiment(*experimentPath)
	if err != nil {
		return nil, err
	}
	result, err := umpire3runner.Execute(ctx, experiment, connection.options())
	if err != nil {
		return nil, err
	}
	if *outputPath != "" {
		if err := writeJSONFile(*outputPath, result); err != nil {
			return nil, err
		}
	}
	if *bundlePath != "" {
		encoded, err := artifact.Encode(experiment, result, experiment.Retention.MaxArtifactBytes)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(*bundlePath, append(encoded, '\n'), 0o600); err != nil {
			return nil, fmt.Errorf("write replay bundle: %w", err)
		}
	}
	return result, nil
}

func executeReplay(ctx context.Context, arguments []string) (any, error) {
	flags := flag.NewFlagSet("replay", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	bundlePath := flags.String("bundle", "", "redacted replay bundle")
	connection := addConnectionFlags(flags)
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	encoded, err := readRequiredFile("replay bundle", *bundlePath, protocol.DefaultDecodeLimit)
	if err != nil {
		return nil, err
	}
	bundle, err := artifact.Decode(encoded, protocol.DefaultDecodeLimit)
	if err != nil {
		return nil, err
	}
	if connection.profile == "" {
		connection.profile = bundle.Replay.Profile
	}
	return replay.Run(ctx, bundle, func(ctx context.Context, experiment protocol.Experiment) (umpire3runtime.Result, error) {
		return umpire3runner.Execute(ctx, experiment, connection.options())
	})
}

func executeCampaign(ctx context.Context, arguments []string) (any, error) {
	flags := flag.NewFlagSet("campaign", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	experimentPath := flags.String("experiment", "", "released seed experiment JSON")
	seed := flags.Int64("seed", 1, "deterministic campaign seed")
	maxCandidates := flags.Int("max-candidates", 16, "mutation and execution budget")
	connection := addConnectionFlags(flags)
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	experiment, err := readExperiment(*experimentPath)
	if err != nil {
		return nil, err
	}
	report, err := campaign.Mutate(campaign.MutationRequest{
		Experiment: experiment, Seed: *seed, MaxCandidates: *maxCandidates,
		Values:        mutationValues(experiment),
		TopologyKinds: []protocol.EntityKind{protocol.EntityKindActivity, protocol.EntityKindCallback},
	})
	if err != nil {
		return nil, err
	}
	result := campaignOutput{Mutation: report}
	for _, mutation := range report.Selected {
		runResult, runErr := umpire3runner.Execute(ctx, mutation.Experiment, connection.options())
		if runErr != nil {
			return nil, fmt.Errorf("execute mutation %s at %s: %w", mutation.Kind, mutation.Path, runErr)
		}
		result.Runs = append(result.Runs, campaignExecution{
			Kind: mutation.Kind, Path: mutation.Path, Digest: mutation.Digest, Result: runResult,
		})
	}
	return result, nil
}

func executeQualify(arguments []string) (any, error) {
	flags := flag.NewFlagSet("qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	releasePath := flags.String("release", "", "candidate release manifest")
	experimentPath := flags.String("experiment", "", "executed semantic experiment")
	resultPath := flags.String("result", "", "runtime or canary result")
	profile := flags.String("profile", "", "external profile being qualified")
	if err := flags.Parse(arguments); err != nil {
		return nil, err
	}
	releaseBytes, err := readRequiredFile("release", *releasePath, protocol.DefaultDecodeLimit)
	if err != nil {
		return nil, err
	}
	experimentBytes, err := readRequiredFile("experiment", *experimentPath, protocol.DefaultDecodeLimit)
	if err != nil {
		return nil, err
	}
	resultBytes, err := readRequiredFile("result", *resultPath, protocol.DefaultDecodeLimit)
	if err != nil {
		return nil, err
	}
	return qualification.Qualify(qualification.Request{
		ReleaseBytes: releaseBytes, ExperimentBytes: experimentBytes, ResultBytes: resultBytes, Profile: *profile,
	})
}

func addConnectionFlags(flags *flag.FlagSet) *connectionFlags {
	result := &connectionFlags{}
	flags.StringVar(&result.address, "address", "", "Temporal gRPC address or HTTPS origin")
	flags.StringVar(&result.namespace, "namespace", "", "Temporal namespace")
	flags.StringVar(&result.taskQueue, "task-queue", "", "isolated participant task queue")
	flags.StringVar(&result.buildID, "build-id", "", "deployment build attestation")
	flags.StringVar(&result.profile, "profile", "", "execution profile")
	flags.StringVar(&result.nexusEndpoint, "nexus-endpoint", "", "optional Nexus endpoint")
	flags.StringVar(&result.nexusService, "nexus-service", "", "optional Nexus service")
	flags.StringVar(&result.nexusOperation, "nexus-operation", "", "optional Nexus operation")
	flags.DurationVar(&result.timeout, "timeout", 5*time.Minute, "execution and cleanup budget")
	return result
}

func (flags connectionFlags) options() umpire3runner.Options {
	return umpire3runner.Options{
		Address: flags.address, Namespace: flags.namespace, TaskQueue: flags.taskQueue,
		BuildID: flags.buildID, Profile: flags.profile, NexusEndpoint: flags.nexusEndpoint,
		NexusService: flags.nexusService, NexusOperation: flags.nexusOperation,
		APIKey: os.Getenv("UMPIRE3_TEMPORAL_API_KEY"), Timeout: flags.timeout,
	}
}

func readExperiment(path string) (protocol.Experiment, error) {
	encoded, err := readRequiredFile("experiment", path, protocol.DefaultDecodeLimit)
	if err != nil {
		return protocol.Experiment{}, err
	}
	return protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
}

func readRequiredFile(kind, path string, limit int64) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("%s path is required", kind)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", kind, err)
	}
	encoded, err := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return nil, fmt.Errorf("read %s: %w", kind, errors.Join(err, closeErr))
	}
	if int64(len(encoded)) > limit {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", kind, limit)
	}
	return encoded, nil
}

func writeJSONFile(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func mutationValues(experiment protocol.Experiment) []protocol.Value {
	seen := make(map[protocol.ValueType]struct{})
	var result []protocol.Value
	for _, action := range experiment.Actions {
		for _, argument := range action.Arguments {
			if _, exists := seen[argument.Value.Type]; exists {
				continue
			}
			seen[argument.Value.Type] = struct{}{}
			switch argument.Value.Type {
			case protocol.ValueString:
				value := "umpire3-mutated"
				result = append(result, protocol.Value{Type: protocol.ValueString, Text: &value})
			case protocol.ValueInteger, protocol.ValueDuration:
				value := int64(1)
				if argument.Value.Integer != nil {
					value = *argument.Value.Integer + 1
				}
				result = append(result, protocol.Value{Type: argument.Value.Type, Integer: &value})
			case protocol.ValueBoolean:
				value := argument.Value.Boolean == nil || !*argument.Value.Boolean
				result = append(result, protocol.Value{Type: protocol.ValueBoolean, Boolean: &value})
			default:
			}
		}
	}
	return result
}
