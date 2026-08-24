package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/agentworkflow"
	"go.temporal.io/server/tools/agentworkflow/internal/backend/claude"
	"go.temporal.io/server/tools/agentworkflow/internal/backend/codex"
	projectconfig "go.temporal.io/server/tools/agentworkflow/internal/project"
)

const (
	exitOK          = 0
	exitNeedsChange = 2
	exitUnsupported = 3
	exitInterrupted = 4
	exitFailure     = 5
	exitUsage       = 64
)

func main() {
	os.Exit(runCLI(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	checkedStdout := &checkedWriter{target: stdout}
	checkedStderr := &checkedWriter{target: stderr}
	code := dispatch(ctx, arguments, checkedStdout, checkedStderr)
	if checkedStdout.err != nil || checkedStderr.err != nil {
		return exitFailure
	}
	return code
}

func dispatch(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		printUsage(stderr)
		return exitUsage
	}
	switch arguments[0] {
	case "init":
		return runInit(arguments[1:], stdout, stderr)
	case "doctor":
		return runDoctor(ctx, arguments[1:], stdout, stderr)
	case "run":
		return runWorkflow(ctx, arguments[1:], stdout, stderr)
	case "resume":
		return runResume(ctx, arguments[1:], stdout, stderr)
	case "inspect", "report":
		return runInspect(ctx, arguments[0], arguments[1:], stdout, stderr)
	case "diff":
		return runDiff(ctx, arguments[1:], stdout, stderr)
	case "apply":
		return runApply(ctx, arguments[1:], stdout, stderr)
	case "config":
		return runConfig(arguments[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitOK
	default:
		writef(stderr, "unknown command %q\n", arguments[0])
		printUsage(stderr)
		return exitUsage
	}
}

func runInit(arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("init", stderr)
	root := flags.String("project", ".", "project root")
	config := flags.String("config", "", "configuration path")
	if !parse(flags, arguments) {
		return exitUsage
	}
	path, err := projectconfig.WriteStarter(*root, *config)
	if err != nil {
		return printError(stderr, err)
	}
	writef(stdout, "wrote %s\n", path)
	writeln(stdout, "review the checks, workflow stages, and prompts, then run agentworkflow doctor")
	return exitOK
}

func runDoctor(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("doctor", stderr)
	common := addCommonFlags(flags)
	projectRoot := flags.String("project", ".", "project root")
	configPath := flags.String("config", "", "project configuration")
	target := flags.String("target", "", "monorepo target")
	if !parse(flags, arguments) {
		return exitUsage
	}
	resolved, err := projectconfig.Load(*configPath, *projectRoot, *target)
	if err != nil {
		return printError(stderr, err)
	}
	backend, err := makeBackend(common.backend, common.command, common.backendArgs, common.model, common.qualified)
	if err != nil {
		return printError(stderr, err)
	}
	info, err := backend.Describe(ctx)
	if err != nil {
		return printError(stderr, err)
	}
	writef(stdout, "backend: %s (%s)\n", info.Name, info.Version)
	writef(stdout, "capabilities: %s\n", joinCapabilities(info.Capabilities))
	writef(stdout, "project: %s\n", resolved.Root)
	if len(resolved.Checks) == 0 {
		writeln(stdout, "checks: none enabled (runs will be inconclusive)")
	} else {
		for _, check := range resolved.Checks {
			path, lookErr := exec.LookPath(check.Command[0])
			if lookErr != nil {
				writef(stderr, "check %s: missing executable %s\n", check.Name, check.Command[0])
				return exitUnsupported
			}
			writef(stdout, "check %s: %s\n", check.Name, path)
		}
	}
	return exitOK
}

func runWorkflow(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("run", stderr)
	common := addCommonFlags(flags)
	projectRoot := flags.String("project", ".", "project root")
	configPath := flags.String("config", "", "project configuration")
	target := flags.String("target", "", "monorepo target")
	taskText := flags.String("task", "", "task objective")
	taskFile := flags.String("task-file", "", "file containing the task objective")
	assurance := flags.String("assurance", "", "fast, standard, or high")
	applyResult := flags.Bool("apply", false, "apply a successful result after source-drift validation")
	jsonOutput := flags.Bool("json", false, "print JSON result")
	criteria := stringList{}
	flags.Var(&criteria, "criterion", "success criterion (repeatable)")
	if !parse(flags, arguments) {
		return exitUsage
	}
	if *taskText != "" && *taskFile != "" {
		writeln(stderr, "--task and --task-file are mutually exclusive")
		return exitUsage
	}
	objective := strings.TrimSpace(*taskText)
	if objective == "" && *taskFile != "" {
		data, err := readTask(*taskFile)
		if err != nil {
			return printError(stderr, err)
		}
		objective = strings.TrimSpace(string(data))
	}
	if objective == "" && flags.NArg() > 0 {
		objective = strings.TrimSpace(strings.Join(flags.Args(), " "))
	}
	if objective == "" {
		writeln(stderr, "a task is required via --task, --task-file, or a positional argument")
		return exitUsage
	}
	if len(criteria) == 0 {
		criteria = append(criteria, "The requested outcome is implemented and all declared required checks pass.")
	}
	resolved, err := projectconfig.Load(*configPath, *projectRoot, *target)
	if err != nil {
		return printError(stderr, err)
	}
	if *applyResult && !configStageEnabled(resolved, agentworkflow.StageApply) {
		return printError(stderr, fmt.Errorf("%w: apply stage is disabled by the admitted configuration", agentworkflow.ErrUnsupported))
	}
	request := requestFromConfig(resolved, objective, criteria)
	if *assurance != "" {
		request.Policy.Assurance = agentworkflow.Assurance(*assurance)
	}
	engine, err := openEngine(common, backendProgress(stderr, *jsonOutput))
	if err != nil {
		return printError(stderr, err)
	}
	result, err := engine.Run(ctx, request)
	if err != nil {
		if result.RunID != "" {
			printResult(stdout, result, *jsonOutput)
		}
		return printError(stderr, err)
	}
	printResult(stdout, result, *jsonOutput)
	if *applyResult {
		if result.Outcome != agentworkflow.OutcomeSucceeded {
			writeln(stderr, "result was not successful; candidate was not applied")
		} else if err := engine.Apply(ctx, result.RunID); err != nil {
			return printError(stderr, err)
		} else {
			writef(stderr, "applied run %s\n", result.RunID)
		}
	}
	return outcomeExit(result.Outcome)
}

func runResume(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("resume", stderr)
	common := addCommonFlags(flags)
	jsonOutput := flags.Bool("json", false, "print JSON result")
	if !parse(flags, arguments) || flags.NArg() != 1 {
		writeln(stderr, "resume requires one run ID")
		return exitUsage
	}
	engine, err := openEngine(common, backendProgress(stderr, *jsonOutput))
	if err != nil {
		return printError(stderr, err)
	}
	result, err := engine.Resume(ctx, agentworkflow.RunID(flags.Arg(0)))
	if result.RunID != "" {
		printResult(stdout, result, *jsonOutput)
	}
	if err != nil {
		return printError(stderr, err)
	}
	return outcomeExit(result.Outcome)
}

func runInspect(ctx context.Context, command string, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet(command, stderr)
	common := addCommonFlags(flags)
	jsonOutput := flags.Bool("json", command == "report", "print JSON status")
	if !parse(flags, arguments) || flags.NArg() != 1 {
		writef(stderr, "%s requires one run ID\n", command)
		return exitUsage
	}
	engine, err := openEngine(common, nil)
	if err != nil {
		return printError(stderr, err)
	}
	status, err := engine.Inspect(ctx, agentworkflow.RunID(flags.Arg(0)))
	if *jsonOutput {
		_ = writeJSON(stdout, status)
	} else {
		writef(stdout, "%s  %s  %s", status.RunID, status.State, status.Phase)
		if status.Outcome != "" {
			writef(stdout, "  %s", status.Outcome)
		}
		writeln(stdout)
	}
	if err != nil {
		return printError(stderr, err)
	}
	return outcomeExit(status.Outcome)
}

func runDiff(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("diff", stderr)
	common := addCommonFlags(flags)
	jsonOutput := flags.Bool("json", false, "print JSON changes")
	if !parse(flags, arguments) || flags.NArg() != 1 {
		writeln(stderr, "diff requires one run ID")
		return exitUsage
	}
	engine, err := openEngine(common, nil)
	if err != nil {
		return printError(stderr, err)
	}
	changes, err := engine.Diff(ctx, agentworkflow.RunID(flags.Arg(0)))
	if err != nil {
		return printError(stderr, err)
	}
	if *jsonOutput {
		_ = writeJSON(stdout, changes)
		return exitOK
	}
	for _, change := range changes {
		marker := map[string]string{"added": "A", "modified": "M", "deleted": "D"}[change.Kind]
		writef(stdout, "%s  %s\n", marker, change.Path)
	}
	return exitOK
}

func runApply(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("apply", stderr)
	common := addCommonFlags(flags)
	if !parse(flags, arguments) || flags.NArg() != 1 {
		writeln(stderr, "apply requires one run ID")
		return exitUsage
	}
	engine, err := openEngine(common, nil)
	if err != nil {
		return printError(stderr, err)
	}
	id := agentworkflow.RunID(flags.Arg(0))
	if err := engine.Apply(ctx, id); err != nil {
		return printError(stderr, err)
	}
	writef(stdout, "applied run %s\n", id)
	return exitOK
}

func runConfig(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] != "explain" {
		writeln(stderr, "config requires the explain subcommand")
		return exitUsage
	}
	flags := newFlagSet("config explain", stderr)
	projectRoot := flags.String("project", ".", "project root")
	configPath := flags.String("config", "", "project configuration")
	target := flags.String("target", "", "monorepo target")
	if !parse(flags, arguments[1:]) {
		return exitUsage
	}
	resolved, err := projectconfig.Load(*configPath, *projectRoot, *target)
	if err != nil {
		return printError(stderr, err)
	}
	data, err := projectconfig.Explain(resolved)
	if err != nil {
		return printError(stderr, err)
	}
	writeln(stdout, string(data))
	return exitOK
}

type commonFlags struct {
	store       *string
	backend     *string
	command     *string
	backendArgs *stringList
	model       *string
	qualified   *bool
}

func addCommonFlags(flags *flag.FlagSet) commonFlags {
	backendArguments := stringList{}
	flags.Var(&backendArguments, "backend-arg", "extra backend executable argument (repeatable)")
	return commonFlags{
		store:       flags.String("store", defaultStore(), "run artifact store"),
		backend:     flags.String("backend", "codex", "codex or claude"),
		command:     flags.String("backend-command", "", "backend executable override"),
		backendArgs: &backendArguments,
		model:       flags.String("model", "", "backend model override"),
		qualified:   flags.Bool("qualified", false, "ignore uncontrolled backend configuration"),
	}
}

func openEngine(common commonFlags, observe func(agentworkflow.Progress)) (*agentworkflow.Engine, error) {
	backend, err := makeBackend(common.backend, common.command, common.backendArgs, common.model, common.qualified)
	if err != nil {
		return nil, err
	}
	return agentworkflow.Open(agentworkflow.Config{Root: *common.store, Backend: backend, Limits: agentworkflow.DefaultLimits(), Observe: observe})
}

func makeBackend(name, command *string, backendArguments *stringList, model *string, qualified *bool) (agentworkflow.Backend, error) {
	if *qualified && (strings.TrimSpace(*command) != "" || len(*backendArguments) != 0) {
		return nil, fmt.Errorf("%w: --qualified cannot be combined with backend executable or argument overrides", agentworkflow.ErrUnsupported)
	}
	executable := strings.TrimSpace(*command)
	if executable == "" {
		executable = *name
	}
	commandLine := append([]string{executable}, []string(*backendArguments)...)
	switch *name {
	case "codex":
		return codex.New(codex.Config{Command: commandLine, Model: *model, Qualified: *qualified})
	case "claude":
		return claude.New(claude.Config{Command: commandLine, Model: *model, Qualified: *qualified})
	default:
		return nil, fmt.Errorf("unknown backend %q", *name)
	}
}

func requestFromConfig(config projectconfig.Resolved, objective string, criteria []string) agentworkflow.Request {
	checks := make([]agentworkflow.Check, len(config.Checks))
	for index, check := range config.Checks {
		checks[index] = agentworkflow.Check{
			Name: check.Name, Command: check.Command, Directory: check.Directory,
			Timeout: check.Timeout.Duration, Required: check.Required,
		}
	}
	return agentworkflow.Request{
		Task: agentworkflow.Task{Objective: objective, SuccessCriteria: append([]string(nil), criteria...)},
		Project: agentworkflow.Project{
			Root:         config.Root,
			Source:       agentworkflow.SourcePolicy{Mode: agentworkflow.SourceMode(config.Source.Mode), Exclude: config.Source.Exclude},
			Instructions: config.Instructions, Checks: checks,
			Environment:    agentworkflow.EnvironmentPolicy{Allow: config.Environment.Allow},
			ForbiddenPaths: config.ForbiddenPaths,
		},
		Policy: agentworkflow.Policy{
			Assurance: agentworkflow.Assurance(config.Policy.Assurance), MaxRepairs: config.Policy.MaxRepairs,
			Reviewers: config.Policy.Reviewers, BlockingSeverity: agentworkflow.Severity(config.Policy.BlockingSeverity),
		},
		Workflow: workflowFromConfig(config),
	}
}

func workflowFromConfig(config projectconfig.Resolved) agentworkflow.Workflow {
	workflow := agentworkflow.Workflow{Stages: make([]agentworkflow.WorkflowStage, len(config.Workflow.Stages))}
	for index, stage := range config.Workflow.Stages {
		var models agentworkflow.Models
		if stage.Models != nil {
			models = make(agentworkflow.Models, len(stage.Models))
			for provider, model := range stage.Models {
				models[provider] = model
			}
		}
		workflow.Stages[index] = agentworkflow.WorkflowStage{
			Kind: agentworkflow.StageKind(stage.Kind), Enabled: stage.Enabled, Models: models, Prompt: stage.Prompt,
			ReviewPrompt: stage.ReviewPrompt, RevisionPrompt: stage.RevisionPrompt, Mode: stage.Mode,
		}
	}
	return workflow
}

func configStageEnabled(config projectconfig.Resolved, kind agentworkflow.StageKind) bool {
	for _, stage := range config.Workflow.Stages {
		if string(stage.Kind) == string(kind) {
			return stage.Enabled
		}
	}
	return false
}

func backendProgress(stderr io.Writer, quiet bool) func(agentworkflow.Progress) {
	if quiet {
		return nil
	}
	var prior string
	return func(progress agentworkflow.Progress) {
		key := progress.State + "\x00" + progress.Phase + "\x00" + string(progress.Outcome)
		if key == prior {
			return
		}
		prior = key
		if progress.Outcome == "" {
			writef(stderr, "[%s] %s\n", progress.State, progress.Phase)
		} else {
			writef(stderr, "[%s] %s: %s\n", progress.State, progress.Phase, progress.Outcome)
		}
	}
}

func printResult(output io.Writer, result agentworkflow.Result, asJSON bool) {
	if asJSON {
		_ = writeJSON(output, result)
		return
	}
	writef(output, "run: %s\noutcome: %s\nphase: %s\n", result.RunID, result.Outcome, result.Phase)
	if result.Message != "" {
		writef(output, "message: %s\n", result.Message)
	}
	writef(output, "candidate digest: %s\nchanges: %d\nchecks: %d\nreviews: %d\n", result.CandidateDigest, len(result.Changes), len(result.Checks), len(result.Reviews))
}

func outcomeExit(outcome agentworkflow.Outcome) int {
	switch outcome {
	case "", agentworkflow.OutcomeSucceeded:
		return exitOK
	case agentworkflow.OutcomeNeedsChanges, agentworkflow.OutcomeProjectFailed, agentworkflow.OutcomeAgentFailed, agentworkflow.OutcomeInconclusive:
		return exitNeedsChange
	case agentworkflow.OutcomeUnsupported:
		return exitUnsupported
	case agentworkflow.OutcomeCancelled, agentworkflow.OutcomeTimedOut, agentworkflow.OutcomeCapacityExhausted, agentworkflow.OutcomeRecoverableInterruption:
		return exitInterrupted
	default:
		return exitFailure
	}
}

func printError(output io.Writer, err error) int {
	writef(output, "agentworkflow: %v\n", err)
	switch {
	case errors.Is(err, agentworkflow.ErrUnsupported):
		return exitUnsupported
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.Is(err, agentworkflow.ErrCapacity):
		return exitInterrupted
	default:
		return exitFailure
	}
}

func readTask(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if len(data) > 1<<20 {
		return nil, errors.New("task file exceeds 1 MiB")
	}
	return data, nil
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func joinCapabilities(values []agentworkflow.Capability) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = string(value)
	}
	return strings.Join(parts, ", ")
}

func defaultStore() string {
	if configured := strings.TrimSpace(os.Getenv("AGENTWORKFLOW_HOME")); configured != "" {
		return configured
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agentworkflow-runs")
	}
	return filepath.Join(cache, "agentworkflow", "runs")
}

func newFlagSet(name string, output io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(output)
	return flags
}

func parse(flags *flag.FlagSet, arguments []string) bool {
	return flags.Parse(arguments) == nil
}

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value cannot be empty")
	}
	*values = append(*values, value)
	return nil
}

func printUsage(output io.Writer) {
	writeln(output, "usage: agentworkflow <command> [options]")
	writeln(output, "commands: init, doctor, run, resume, inspect, report, diff, apply, config explain")
}

type checkedWriter struct {
	target io.Writer
	err    error
}

func (writer *checkedWriter) Write(data []byte) (int, error) {
	if writer.err != nil {
		return 0, writer.err
	}
	written, err := writer.target.Write(data)
	if err != nil {
		writer.err = err
	}
	return written, err
}

func writef(output io.Writer, format string, arguments ...any) {
	_, _ = fmt.Fprintf(output, format, arguments...)
}

func writeln(output io.Writer, arguments ...any) {
	_, _ = fmt.Fprintln(output, arguments...)
}
