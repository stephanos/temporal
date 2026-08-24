package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.temporal.io/server/tools/agentworkflow/internal/agentworkflow"
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

func Run(ctx context.Context, arguments []string, stdout, stderr io.Writer) int {
	checkedStdout := &checkedWriter{target: stdout}
	checkedStderr := &checkedWriter{target: stderr}
	application := application{stdout: checkedStdout, stderr: checkedStderr}
	root := application.rootCommand()
	root.SetArgs(normalizeLegacyFlags(root, arguments))
	code := exitOK
	executed, err := root.ExecuteContextC(ctx)
	if err != nil {
		if executed == nil {
			executed = root
		}
		code = application.handleExecutionError(executed, err)
	}
	if checkedStdout.err != nil || checkedStderr.err != nil {
		return exitFailure
	}
	return code
}

func normalizeLegacyFlags(root *cobra.Command, arguments []string) []string {
	valueFlags := map[string]bool{"help": false}
	var collectFlags func(*cobra.Command)
	collectFlags = func(command *cobra.Command) {
		visit := func(flag *pflag.Flag) {
			valueFlags[flag.Name] = valueFlags[flag.Name] || flag.NoOptDefVal == ""
		}
		command.LocalNonPersistentFlags().VisitAll(visit)
		command.PersistentFlags().VisitAll(visit)
		for _, child := range command.Commands() {
			collectFlags(child)
		}
	}
	collectFlags(root)

	normalized := append([]string(nil), arguments...)
	consumeValue := false
	for index, argument := range normalized {
		if consumeValue {
			consumeValue = false
			continue
		}
		if argument == "--" {
			break
		}
		name, hasValue, doubleDash := flagArgument(argument)
		takesValue, known := valueFlags[name]
		if !known {
			continue
		}
		if !doubleDash {
			normalized[index] = "--" + argument[1:]
		}
		consumeValue = takesValue && !hasValue
	}
	return normalized
}

func flagArgument(argument string) (name string, hasValue bool, doubleDash bool) {
	if len(argument) < 3 || argument[0] != '-' {
		return "", false, false
	}
	doubleDash = argument[1] == '-'
	start := 1
	if doubleDash {
		start = 2
	}
	if start == len(argument) {
		return "", false, doubleDash
	}
	name, _, hasValue = strings.Cut(argument[start:], "=")
	return name, hasValue, doubleDash
}

type application struct {
	stdout io.Writer
	stderr io.Writer
}

func (application application) rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "agentworkflow",
		Short:         "Run evidence-driven agent workflows",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				return &exitError{code: exitUsage, usage: true}
			}
			return fmt.Errorf("unknown command %q for %q", arguments[0], command.CommandPath())
		},
		RunE: func(*cobra.Command, []string) error { return nil },
	}
	root.SetOut(application.stdout)
	root.SetErr(application.stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.DisableSuggestions = true
	root.AddCommand(
		application.initCommand(),
		application.doctorCommand(),
		application.runCommand(),
		application.resumeCommand(),
		application.inspectCommand("inspect"),
		application.inspectCommand("report"),
		application.diffCommand(),
		application.applyCommand(),
		application.configCommand(),
	)
	return root
}

type exitError struct {
	code  int
	err   error
	usage bool
}

func (err *exitError) Error() string {
	if err.err == nil {
		return ""
	}
	return err.err.Error()
}

func (err *exitError) Unwrap() error {
	return err.err
}

func (application application) handleExecutionError(root *cobra.Command, err error) int {
	var status *exitError
	if errors.As(err, &status) {
		if status.usage {
			if status.err != nil {
				writeln(application.stderr, status.err)
			}
			root.SetOut(application.stderr)
			_ = root.Usage()
		}
		return status.code
	}
	writeln(application.stderr, err)
	root.SetOut(application.stderr)
	_ = root.Usage()
	return exitUsage
}

func (application application) operationError(err error) error {
	return &exitError{code: printError(application.stderr, err), err: err}
}

func usageError(err error) error {
	return &exitError{code: exitUsage, err: err, usage: true}
}

func outcomeError(outcome agentworkflow.Outcome) error {
	code := outcomeExit(outcome)
	if code == exitOK {
		return nil
	}
	return &exitError{code: code}
}

func (application application) initCommand() *cobra.Command {
	var projectRoot string
	var configPath string
	command := &cobra.Command{
		Use:   "init",
		Short: "Write a starter project configuration",
		Args:  cobra.ExactArgs(0),
		RunE: func(*cobra.Command, []string) error {
			path, err := projectconfig.WriteStarter(projectRoot, configPath)
			if err != nil {
				return application.operationError(err)
			}
			writef(application.stdout, "wrote %s\n", path)
			writeln(application.stdout, "review the checks, workflow stages, and prompts, then run agentworkflow doctor")
			return nil
		},
	}
	command.Flags().StringVar(&projectRoot, "project", ".", "project root")
	command.Flags().StringVar(&configPath, "config", "", "configuration path")
	return command
}

func (application application) doctorCommand() *cobra.Command {
	var projectRoot string
	var configPath string
	var target string
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Validate the project and backend",
		Args:  cobra.ExactArgs(0),
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, _ []string) error {
		resolved, err := projectconfig.Load(configPath, projectRoot, target)
		if err != nil {
			return application.operationError(err)
		}
		backend, err := makeBackend(common.backend, common.command, common.backendArgs, common.model, common.qualified)
		if err != nil {
			return application.operationError(err)
		}
		info, err := backend.Describe(command.Context())
		if err != nil {
			return application.operationError(err)
		}
		writef(application.stdout, "backend: %s (%s)\n", info.Name, info.Version)
		writef(application.stdout, "capabilities: %s\n", joinCapabilities(info.Capabilities))
		writef(application.stdout, "project: %s\n", resolved.Root)
		if len(resolved.Checks) == 0 {
			writeln(application.stdout, "checks: none enabled (runs will be inconclusive)")
		} else {
			for _, check := range resolved.Checks {
				path, lookErr := exec.LookPath(check.Command[0])
				if lookErr != nil {
					writef(application.stderr, "check %s: missing executable %s\n", check.Name, check.Command[0])
					return &exitError{code: exitUnsupported}
				}
				writef(application.stdout, "check %s: %s\n", check.Name, path)
			}
		}
		return nil
	}
	command.Flags().StringVar(&projectRoot, "project", ".", "project root")
	command.Flags().StringVar(&configPath, "config", "", "project configuration")
	command.Flags().StringVar(&target, "target", "", "monorepo target")
	return command
}

func (application application) runCommand() *cobra.Command {
	var projectRoot string
	var configPath string
	var target string
	var taskText string
	var taskFile string
	var assurance string
	var applyResult bool
	var jsonOutput bool
	criteria := stringList{}
	command := &cobra.Command{
		Use:   "run [objective]",
		Short: "Run a workflow",
		Args:  cobra.ArbitraryArgs,
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, arguments []string) error {
		if taskText != "" && taskFile != "" {
			return usageError(errors.New("--task and --task-file are mutually exclusive"))
		}
		objective := strings.TrimSpace(taskText)
		if objective == "" && taskFile != "" {
			data, err := readTask(taskFile)
			if err != nil {
				return application.operationError(err)
			}
			objective = strings.TrimSpace(string(data))
		}
		if objective == "" && len(arguments) > 0 {
			objective = strings.TrimSpace(strings.Join(arguments, " "))
		}
		if objective == "" {
			return usageError(errors.New("a task is required via --task, --task-file, or a positional argument"))
		}
		if len(criteria) == 0 {
			criteria = append(criteria, "The requested outcome is implemented and all declared required checks pass.")
		}
		resolved, err := projectconfig.Load(configPath, projectRoot, target)
		if err != nil {
			return application.operationError(err)
		}
		if applyResult && !configStageEnabled(resolved, agentworkflow.StageApply) {
			return application.operationError(fmt.Errorf("%w: apply stage is disabled by the admitted configuration", agentworkflow.ErrUnsupported))
		}
		request := requestFromConfig(resolved, objective, criteria)
		if assurance != "" {
			request.Policy.Assurance = agentworkflow.Assurance(assurance)
		}
		engine, err := openEngine(common, backendProgress(application.stderr, jsonOutput))
		if err != nil {
			return application.operationError(err)
		}
		result, err := engine.Run(command.Context(), request)
		if err != nil {
			if result.RunID != "" {
				printResult(application.stdout, result, jsonOutput)
			}
			return application.operationError(err)
		}
		printResult(application.stdout, result, jsonOutput)
		if applyResult {
			if result.Outcome != agentworkflow.OutcomeSucceeded {
				writeln(application.stderr, "result was not successful; candidate was not applied")
			} else if err := engine.Apply(command.Context(), result.RunID); err != nil {
				return application.operationError(err)
			} else {
				writef(application.stderr, "applied run %s\n", result.RunID)
			}
		}
		return outcomeError(result.Outcome)
	}
	command.Flags().StringVar(&projectRoot, "project", ".", "project root")
	command.Flags().StringVar(&configPath, "config", "", "project configuration")
	command.Flags().StringVar(&target, "target", "", "monorepo target")
	command.Flags().StringVar(&taskText, "task", "", "task objective")
	command.Flags().StringVar(&taskFile, "task-file", "", "file containing the task objective")
	command.Flags().StringVar(&assurance, "assurance", "", "fast, standard, or high")
	command.Flags().BoolVar(&applyResult, "apply", false, "apply a successful result after source-drift validation")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON result")
	command.Flags().Var(&criteria, "criterion", "success criterion (repeatable)")
	return command
}

func (application application) resumeCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "resume RUN_ID",
		Short: "Resume an interrupted workflow",
		Args:  cobra.ExactArgs(1),
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, arguments []string) error {
		engine, err := openEngine(common, backendProgress(application.stderr, jsonOutput))
		if err != nil {
			return application.operationError(err)
		}
		result, err := engine.Resume(command.Context(), agentworkflow.RunID(arguments[0]))
		if result.RunID != "" {
			printResult(application.stdout, result, jsonOutput)
		}
		if err != nil {
			return application.operationError(err)
		}
		return outcomeError(result.Outcome)
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON result")
	return command
}

func (application application) inspectCommand(name string) *cobra.Command {
	jsonOutput := name == "report"
	command := &cobra.Command{
		Use:   name + " RUN_ID",
		Short: "Inspect a workflow run",
		Args:  cobra.ExactArgs(1),
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, arguments []string) error {
		engine, err := openEngine(common, nil)
		if err != nil {
			return application.operationError(err)
		}
		status, err := engine.Inspect(command.Context(), agentworkflow.RunID(arguments[0]))
		if jsonOutput {
			_ = writeJSON(application.stdout, status)
		} else {
			writef(application.stdout, "%s  %s  %s", status.RunID, status.State, status.Phase)
			if status.Outcome != "" {
				writef(application.stdout, "  %s", status.Outcome)
			}
			writeln(application.stdout)
		}
		if err != nil {
			return application.operationError(err)
		}
		return outcomeError(status.Outcome)
	}
	command.Flags().BoolVar(&jsonOutput, "json", name == "report", "print JSON status")
	return command
}

func (application application) diffCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "diff RUN_ID",
		Short: "Show candidate changes",
		Args:  cobra.ExactArgs(1),
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, arguments []string) error {
		engine, err := openEngine(common, nil)
		if err != nil {
			return application.operationError(err)
		}
		changes, err := engine.Diff(command.Context(), agentworkflow.RunID(arguments[0]))
		if err != nil {
			return application.operationError(err)
		}
		if jsonOutput {
			_ = writeJSON(application.stdout, changes)
			return nil
		}
		for _, change := range changes {
			marker := map[string]string{"added": "A", "modified": "M", "deleted": "D"}[change.Kind]
			writef(application.stdout, "%s  %s\n", marker, change.Path)
		}
		return nil
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print JSON changes")
	return command
}

func (application application) applyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "apply RUN_ID",
		Short: "Apply a successful candidate",
		Args:  cobra.ExactArgs(1),
	}
	common := addCommonFlags(command)
	command.RunE = func(command *cobra.Command, arguments []string) error {
		engine, err := openEngine(common, nil)
		if err != nil {
			return application.operationError(err)
		}
		id := agentworkflow.RunID(arguments[0])
		if err := engine.Apply(command.Context(), id); err != nil {
			return application.operationError(err)
		}
		writef(application.stdout, "applied run %s\n", id)
		return nil
	}
	return command
}

func (application application) configCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Inspect resolved configuration",
		Args:  cobra.ExactArgs(0),
		RunE: func(*cobra.Command, []string) error {
			return usageError(errors.New("config requires the explain subcommand"))
		},
	}
	command.AddCommand(application.configExplainCommand())
	return command
}

func (application application) configExplainCommand() *cobra.Command {
	var projectRoot string
	var configPath string
	var target string
	command := &cobra.Command{
		Use:   "explain",
		Short: "Print the resolved project configuration",
		Args:  cobra.ExactArgs(0),
		RunE: func(*cobra.Command, []string) error {
			resolved, err := projectconfig.Load(configPath, projectRoot, target)
			if err != nil {
				return application.operationError(err)
			}
			data, err := projectconfig.Explain(resolved)
			if err != nil {
				return application.operationError(err)
			}
			writeln(application.stdout, string(data))
			return nil
		},
	}
	command.Flags().StringVar(&projectRoot, "project", ".", "project root")
	command.Flags().StringVar(&configPath, "config", "", "project configuration")
	command.Flags().StringVar(&target, "target", "", "monorepo target")
	return command
}

type commonFlags struct {
	store       *string
	backend     *string
	command     *string
	backendArgs *stringList
	model       *string
	qualified   *bool
}

func addCommonFlags(command *cobra.Command) commonFlags {
	flags := command.Flags()
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

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Type() string {
	return "value"
}

func (values *stringList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value cannot be empty")
	}
	*values = append(*values, value)
	return nil
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
