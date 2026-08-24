package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/toolchain"
)

const usage = `usage:
  gomad plan [flags] --output FILE (exec --provenance FILE -- BINARY [ARG ...] | go-run PACKAGE -- [ARG ...] | go-test PACKAGE -- [TEST_BINARY_ARG ...])
  gomad execute-shard [--json] [--artifacts DIR] [--toolchain-root DIR] --shard INDEX/COUNT CAMPAIGN_PLAN
  gomad merge [--json] [--partial] --output DIR CAMPAIGN_PLAN SHARD_BATCH...
  gomad explore [flags] exec --provenance FILE -- BINARY [ARG ...]
  gomad explore [flags] go-run PACKAGE -- [ARG ...]
  gomad explore [flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
  gomad qualify [flags] exec --provenance FILE -- BINARY [ARG ...]
  gomad qualify [flags] go-run PACKAGE -- [ARG ...]
  gomad qualify [flags] go-test PACKAGE -- [TEST_BINARY_ARG ...]
  gomad qualify-set --manifest FILE --working-dir DIR [--artifacts DIR] [--output FILE] [--format=text|json]
  gomad compare-support --baseline FILE --candidate FILE [--approve-boundary-diff SHA256] [--format=text|json]
  gomad analyze [--format=text|json] [--timeout DURATION] [--toolchain-root DIR] [--build-tag TAG ...] (go-run PACKAGE | go-test PACKAGE -- [TEST_BINARY_ARG ...])
  gomad resume [--json] INTERRUPTED_BATCH
  gomad recover [--json] INTERRUPTED_BATCH
  gomad replay [--verify-only] ARTIFACT_DIR
  gomad minimize [--json] [--attempt-budget N] [--artifacts DIR] ARTIFACT_DIR
  gomad doctor [--artifacts DIR] [--json]
  gomad inspect [--json] [--choices] ARTIFACT_OR_BATCH
`

type byteSize uint64

func (size *byteSize) String() string {
	return strconv.FormatUint(uint64(*size), 10)
}

func (size *byteSize) Set(input string) error {
	multiplier := uint64(1)
	number := input
	for suffix, value := range map[string]uint64{"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30} {
		if strings.HasSuffix(input, suffix) {
			multiplier = value
			number = strings.TrimSuffix(input, suffix)
			break
		}
	}
	if number == "" || len(number) > 1 && number[0] == '0' {
		return fmt.Errorf("invalid byte size %q", input)
	}
	value, err := strconv.ParseUint(number, 10, 64)
	if err != nil || value == 0 || value > ^uint64(0)/multiplier {
		return fmt.Errorf("invalid byte size %q", input)
	}
	if multiplier == 1 && input != number {
		return fmt.Errorf("invalid byte size %q", input)
	}
	*size = byteSize(value * multiplier)
	return nil
}

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type targetInput struct {
	kind       target.Kind
	source     string
	provenance string
	arguments  []string
}

func Run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch arguments[0] {
	case "__coordinator", "__target_bootstrap", "__supervisor":
		if err := runner.DispatchPrivateMode(arguments[0], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		return 0
	case "explore":
		return runExplore(arguments[1:], stdout, stderr)
	case "plan":
		return runPlan(arguments[1:], stdout, stderr)
	case "execute-shard":
		return runCampaignShard(arguments[1:], stdout, stderr)
	case "merge":
		return runMergeCampaigns(arguments[1:], stdout, stderr)
	case "qualify":
		return runQualify(arguments[1:], stdout, stderr)
	case "qualify-set":
		return runQualifySet(arguments[1:], stdout, stderr)
	case "compare-support":
		return runCompareSupport(arguments[1:], stdout, stderr)
	case "analyze":
		return runAnalyze(arguments[1:], stdout, stderr)
	case "resume":
		return runResume(arguments[1:], stdout, stderr)
	case "recover":
		return runRecover(arguments[1:], stdout, stderr)
	case "replay":
		return runReplay(arguments[1:], stdout, stderr)
	case "minimize":
		return runMinimize(arguments[1:], stdout, stderr)
	case "doctor":
		executable, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "resolve gomad executable: %v\n", err)
			return 3
		}
		return runDoctor(arguments[1:], stdout, stderr, executable)
	case "inspect":
		return runInspect(arguments[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown gomad command %q\n%s", arguments[0], usage)
		return 2
	}
}

func runInspect(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	choices := flags.Bool("choices", false, "project a validated runtime choice trace")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		if _, err := fmt.Fprint(stderr, usage); err != nil {
			return 3
		}
		return 2
	}
	report, err := runner.Inspect(flags.Arg(0), runner.InspectOptions{Choices: *choices})
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "inspect %s: %v\n", flags.Arg(0), err); writeErr != nil {
			return 3
		}
		return 2
	}
	if *jsonOutput {
		encoded, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			if _, writeErr := fmt.Fprintf(stderr, "encode inspection report: %v\n", marshalErr); writeErr != nil {
				return 3
			}
			return 3
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
		return 0
	}
	if err := printInspection(stdout, report); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "write inspection report: %v\n", err); writeErr != nil {
			return 3
		}
		return 3
	}
	return 0
}

type inspectionPrinter struct {
	output io.Writer
	err    error
}

func (printer *inspectionPrinter) printf(format string, arguments ...any) {
	if printer.err != nil {
		return
	}
	_, printer.err = fmt.Fprintf(printer.output, format, arguments...)
}

func printInspection(output io.Writer, report runner.Inspection) error {
	printer := &inspectionPrinter{output: output}
	printer.printf("gomad inspect: kind=%s path=%s\n", report.Kind, report.Path)
	if plan := report.Plan; plan != nil {
		printer.printf("campaign-plan: sha256=%s bundle=%s mapping=%s strategy=%s selection=%s selected=%d parallel=%d runner=%s toolchain=%s target=%s/%d mounts=%d mount-sha256=%s journal-max-executions=%d journal-max-bytes=%d artifact-max-bytes=%d\n", plan.SHA256, plan.BundlePath, plan.Mapping, plan.Strategy, plan.Selection, plan.SelectionCount, plan.Parallel, plan.RunnerBuild, plan.Toolchain.BuildKey, plan.Target.SHA256, plan.Target.Size, len(plan.ReadOnlyMounts), plan.MountSHA256, plan.Journal.MaximumExecutions, plan.Journal.MaximumBytes, plan.ArtifactCapacity.TotalBytes)
		return printer.err
	}
	if inspected := report.Artifact; inspected != nil {
		printArtifactInspection(printer, inspected, report.Path)
		return printer.err
	}
	if merged := report.Merged; merged != nil {
		printer.printf("merged-campaign: plan=%s partial=%t selection=%s selected=%d shards=%d attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d retained-evidence=%d evidence-bytes=%d journal-segments=%d journal-bytes=%d missing=%s\n", merged.PlanSHA256, merged.Partial, merged.Selection, merged.SelectionCount, merged.Shards, merged.Attempted, merged.Succeeded, merged.Failures, merged.Watchdogs, merged.Cancelled, merged.DistinctFailures, merged.RetainedEvidence, merged.EvidenceBytes, merged.JournalSegments, merged.JournalBytes, formatMissingOrdinals(merged.Missing))
		for _, source := range merged.SourceCampaignIDs {
			printer.printf("source-campaign: %s\n", source)
		}
		for _, identity := range merged.EvidenceIdentities {
			printer.printf("evidence: %s\n", identity)
		}
		return printer.err
	}
	if lifecycle := report.Lifecycle; lifecycle != nil {
		printer.printf("lifecycle: state=%s stable=%s published=%t resumable=%t repairable=%t action=%s reason=%s\n", lifecycle.State, lifecycle.LastStableState, lifecycle.Published, lifecycle.Resumable, lifecycle.Repairable, lifecycle.Action, lifecycle.Reason)
		if report.SimulationExploration != nil {
			printSimulationExplorationInspection(printer, report.SimulationExploration)
		}
		if report.Campaign == nil {
			return printer.err
		}
	}
	printCampaignInspection(printer, report.Campaign)
	return printer.err
}

func printSimulationExplorationInspection(printer *inspectionPrinter, inspected *runner.SimulationExplorationInspection) {
	exploration := inspected.Summary
	limits := exploration.Limits
	printer.printf("simulation-exploration: rounds=%d executions=%d pending=%d bytes=%d seen=%d outcomes=%d failures=%d depth=%d max-executions=%d max-forced=%d max-bytes=%d max-result-bytes=%d runtime=%d scenario=%d network=%d storage=%d fault=%d crash=%d omitted-executions=%d omitted-depth=%d omitted-dimension=%d omitted-bytes=%d complete=%t implementation=%s chain=%s\n", exploration.CommittedRounds, exploration.LogicalExecutions, exploration.Pending, exploration.PendingBytes, exploration.SeenCandidates, exploration.DeduplicatedOutcomes, exploration.DistinctFailures, exploration.DeepestOverride, exploration.MaxExecutions, exploration.MaxForcedDecisions, exploration.MaxExplorationBytes, exploration.MaxResultBytes, limits.Runtime, limits.Scenario, limits.Network, limits.Storage, limits.Fault, limits.Crash, exploration.OmittedByExecutionBound, exploration.OmittedByDepth, exploration.OmittedByDimension, exploration.OmittedByCapacity, exploration.BoundedComplete, inspected.ImplementationSHA256, inspected.ChainSHA256)
	for _, candidate := range inspected.Pending {
		printer.printf("pending-candidate: sha256=%s parent=%s depth=%d\n", candidate.SHA256, candidate.ParentSHA256, len(candidate.Overrides))
		for _, override := range candidate.Overrides {
			printer.printf("forced-decision: candidate=%s dimension=%s ordinal=%d selected=%d alternatives=%d site=%s alternatives-sha256=%s selected-sha256=%s identity=%s control-bytes=%d control-sha256=%s\n", candidate.SHA256, override.Dimension, override.Ordinal, override.Selected, override.Alternatives, override.SiteSHA256, override.AlternativeSetSHA256, override.SelectedSHA256, override.Identity, override.ControlBytes, override.ControlSHA256)
		}
	}
	if staged := inspected.StagedRound; staged != nil {
		printer.printf("staged-round: index=%d candidates=%d attempted=%d\n", staged.Index, staged.Candidates, staged.Attempted)
	}
}

func printArtifactInspection(printer *inspectionPrinter, inspected *runner.ArtifactInspection, path string) {
	printer.printf("identity: record=%s campaign=%s ordinal=%d seed=%d toolchain=%s runner=%s\n", inspected.RecordHash, inspected.CampaignID, inspected.SelectionOrdinal, inspected.Seed, inspected.Toolchain.BuildKey, inspected.Runner.RunnerBuild)
	printer.printf("target: kind=%s source=%s sha256=%s size=%d argv=%q tags=%q\n", inspected.Target.Kind, inspected.Target.Source, inspected.Target.SHA256, inspected.Target.Size, inspected.Target.Argv, inspected.Target.BuildTags)
	printer.printf("outcome: domain=%s reason=%s termination=%s signature=%s replay-match=%s\n", inspected.Outcome.Domain, inspected.Outcome.Reason, inspected.Outcome.Termination, inspected.Outcome.FailureSignature, optionalBool(inspected.Outcome.ReplayMatch))
	if inspected.FirstDivergence != "" {
		printer.printf("first-divergence: %s\n", inspected.FirstDivergence)
	}
	if transcript := inspected.Transcript; transcript != nil {
		printer.printf("transcript: records=%d bytes=%d sha256=%s\n", transcript.Records, transcript.Bytes, transcript.SHA256)
	} else {
		printer.printf("transcript: none\n")
	}
	if choices := inspected.Choices; choices != nil {
		printer.printf("choices: profile=%s records=%d decisions=%d branching=%d bytes=%d limit=%d sha256=%s tape-sha256=%s exact-replay=%t terminal=%s runnable=%d select-poll=%d select-result=%d\n", choices.Profile, choices.Records, choices.Decisions, choices.BranchingRecords, choices.PayloadBytes, choices.Limit, choices.SHA256, choices.TapeSHA256, choices.ExactReplayAvailable, choices.TerminalState, choices.Runnable, choices.SelectPoll, choices.SelectResult)
		for _, site := range choices.Sites {
			printer.printf("choice-site: kind=%s fingerprint=%s count=%d max-alternatives=%d\n", site.Kind, site.Fingerprint, site.Count, site.MaximumAlternatives)
		}
	}
	if simulation := inspected.Simulation; simulation != nil {
		printer.printf("simulation: profile=%s controller=%s execution=%s candidate=%s outcome=%s failure=%s plan-schema=%s plan-bytes=%d plan-sha256=%s record-schema=%s record-bytes=%d record-limit=%d record-sha256=%s\n", simulation.Profile, simulation.ControllerSHA256, simulation.ExecutionSHA256, simulation.CandidateSHA256, simulation.OutcomeSHA256, simulation.FailureSHA256, simulation.Plan.Schema, simulation.Plan.Bytes, simulation.Plan.SHA256, simulation.Record.Schema, simulation.Record.Bytes, simulation.Record.Limit, simulation.Record.SHA256)
	}
	if minimization := inspected.Minimization; minimization != nil {
		printer.printf("minimization: parent=%s failure=%s candidate=%s->%s forced=%d->%d attempts=%d/%d accepted=%d replay=%t choice=%s simulation=%s implementation=%s\n", minimization.ParentRecordHash, minimization.ParentFailureSignature, minimization.OriginalCandidateSHA256, minimization.FinalCandidateSHA256, minimization.OriginalForcedDecisions, minimization.FinalForcedDecisions, minimization.Attempts, minimization.AttemptBudget, len(minimization.Accepted), minimization.Predicate.ReplayMatch, minimization.Predicate.ChoiceReplay, minimization.Predicate.SimulationReplay, minimization.ImplementationSHA256)
		for _, reduction := range minimization.Accepted {
			printer.printf("minimization-reduction: kind=%s before=%s after=%s removed=%d\n", reduction.Kind, reduction.BeforeSHA256, reduction.AfterSHA256, len(reduction.Removed))
		}
	}
	if mounts := inspected.CapturedMounts; mounts != nil {
		printer.printf("captured-mounts: mappings=%q entries=%d missing=%d bytes=%d\n", mounts.Mappings, mounts.Entries, mounts.NotExist, mounts.TotalBytes)
	} else {
		printer.printf("captured-mounts: none\n")
	}
	printer.printf("stdout: bytes=%d retained=%d discarded=%d truncated=%t sha256=%s\n", inspected.Stdout.TotalBytes, inspected.Stdout.RetainedBytes, inspected.Stdout.DiscardedBytes, inspected.Stdout.Truncated, inspected.Stdout.FullSHA256)
	printer.printf("stderr: bytes=%d retained=%d discarded=%d truncated=%t sha256=%s\n", inspected.Stderr.TotalBytes, inspected.Stderr.RetainedBytes, inspected.Stderr.DiscardedBytes, inspected.Stderr.Truncated, inspected.Stderr.FullSHA256)
	printer.printf("replay: gomad replay %s\n", quoteArgument(path))
}

func printCampaignInspection(printer *inspectionPrinter, campaign *runner.CampaignInspection) {
	printer.printf("campaign: id=%s strategy=%s selection=%s selected=%d attempted=%d succeeded=%d failures=%d watchdogs=%d cancelled=%d distinct=%d retained-successes=%d retained-success-bytes=%d stop=%s\n", campaign.CampaignID, campaign.Strategy, campaign.Selection, campaign.SelectionCount, campaign.Attempted, campaign.Succeeded, campaign.Failures, campaign.Watchdogs, campaign.Cancelled, campaign.DistinctFailures, campaign.RetainedSuccesses, campaign.RetainedSuccessBytes, campaign.StopReason)
	if campaign.Shard != nil {
		printer.printf("shard: plan=%s index=%d count=%d\n", campaign.PlanSHA256, campaign.Shard.Index, campaign.Shard.Count)
	}
	if journal := campaign.Journal; journal != nil {
		limits := journal.Limits
		printer.printf("journal: schema=%s index=%s segments=%d records=%d bytes=%d max-executions=%d max-bytes=%d segment-bytes=%d segment-records=%d max-segments=%d max-partials=%d capacity=%s\n", journal.Schema, journal.IndexSHA256, journal.Segments, journal.Records, journal.Bytes, limits.MaximumExecutions, limits.MaximumBytes, limits.SegmentBytes, limits.SegmentRecords, limits.MaximumSegments, limits.MaximumPartialExecutions, limits.CapacityOutcome)
	}
	if capacity := campaign.ArtifactCapacity; capacity != nil {
		printer.printf("artifact-capacity: failures=%d failure-bytes=%d successes=%d success-bytes=%d total-bytes=%d transcript-bytes=%d failure-outcome=%s success-outcome=%s\n", capacity.FailureArtifacts, capacity.FailureBytes, capacity.SuccessArtifacts, capacity.SuccessBytes, capacity.TotalBytes, capacity.TranscriptBytes, capacity.FailureOutcome, capacity.SuccessOutcome)
	}
	if exploration := campaign.ChoiceExploration; exploration != nil {
		printer.printf("exploration: rounds=%d pending=%d bytes=%d seen=%d outcomes=%d depth=%d max-executions=%d max-depth=%d max-bytes=%d omitted-executions=%d omitted-depth=%d omitted-bytes=%d complete=%t recovery-executions=%d implementation=%s chain=%s\n", exploration.CommittedRounds, exploration.Pending, exploration.PendingBytes, exploration.SeenPrefixes, exploration.DeduplicatedOutcomes, exploration.DeepestPrefix, exploration.MaxExecutions, exploration.MaxChoiceDepth, exploration.MaxExplorationBytes, exploration.OmittedByExecutionBound, exploration.OmittedByDepth, exploration.OmittedByCapacity, exploration.BoundedComplete, campaign.RecoveryExecutions, campaign.ChoiceExplorationImplementationSHA256, campaign.ChoiceExplorationChainSHA256)
	}
	if exploration := campaign.SimulationExploration; exploration != nil {
		limits := exploration.Limits
		printer.printf("simulation-exploration: rounds=%d executions=%d pending=%d bytes=%d seen=%d outcomes=%d failures=%d depth=%d max-executions=%d max-forced=%d max-bytes=%d max-result-bytes=%d runtime=%d scenario=%d network=%d storage=%d fault=%d crash=%d omitted-executions=%d omitted-depth=%d omitted-dimension=%d omitted-bytes=%d complete=%t recovery-executions=%d implementation=%s chain=%s\n", exploration.CommittedRounds, exploration.LogicalExecutions, exploration.Pending, exploration.PendingBytes, exploration.SeenCandidates, exploration.DeduplicatedOutcomes, exploration.DistinctFailures, exploration.DeepestOverride, exploration.MaxExecutions, exploration.MaxForcedDecisions, exploration.MaxExplorationBytes, exploration.MaxResultBytes, limits.Runtime, limits.Scenario, limits.Network, limits.Storage, limits.Fault, limits.Crash, exploration.OmittedByExecutionBound, exploration.OmittedByDepth, exploration.OmittedByDimension, exploration.OmittedByCapacity, exploration.BoundedComplete, campaign.RecoveryExecutions, campaign.SimulationExplorationImplementationSHA256, campaign.SimulationExplorationChainSHA256)
	}
	for _, run := range campaign.Executions {
		transcript := "none"
		if run.TranscriptSHA256 != nil && run.TranscriptRecords != nil {
			transcript = fmt.Sprintf("%s/%d", *run.TranscriptSHA256, *run.TranscriptRecords)
		}
		choices := "none"
		if run.ChoiceTraceSHA256 != nil && run.ChoiceTraceRecords != nil && run.ChoiceTraceBranchingRecords != nil && run.ChoiceTraceTerminalState != nil {
			choices = fmt.Sprintf("%s/%d/%d/%s", *run.ChoiceTraceSHA256, *run.ChoiceTraceRecords, *run.ChoiceTraceBranchingRecords, *run.ChoiceTraceTerminalState)
		}
		exploration := ""
		if run.Strategy == string(runner.StrategyChoiceExploration) {
			exploration = fmt.Sprintf(" round=%d candidate=%s parent=%s prefix=%s depth=%d outcome=%s", optionalUint64(run.Round), run.CandidateSHA256, run.ParentCandidateSHA256, run.PrefixSHA256, optionalUint64(run.ForcedDepth), run.OutcomeSHA256)
		} else if run.Strategy == string(runner.StrategySimulationExploration) {
			exploration = fmt.Sprintf(" round=%d candidate=%s parent=%s depth=%d outcome=%s", optionalUint64(run.Round), run.CandidateSHA256, run.ParentCandidateSHA256, optionalUint64(run.ForcedDepth), run.OutcomeSHA256)
		}
		printer.printf("execution: ordinal=%d seed=%d domain=%s reason=%s termination=%s elapsed=%dns transcript=%s choices=%s%s\n", run.SelectionOrdinal, run.Seed, run.Domain, run.Reason, run.Termination, run.ElapsedNanos, transcript, choices, exploration)
	}
	for _, failure := range campaign.FailureArtifacts {
		printer.printf("failure: signature=%s path=%s\nreplay: gomad replay %s\n", failure.Signature, failure.Path, quoteArgument(failure.Path))
	}
	for _, success := range campaign.SuccessArtifacts {
		printer.printf("success: bytes=%d novel=%q path=%s\nreplay: gomad replay %s\n", success.StoredBytes, success.NovelProbes, success.Path, quoteArgument(success.Path))
	}
}

func optionalUint64(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

func optionalBool(value *bool) string {
	if value == nil {
		return "not-recorded"
	}
	return strconv.FormatBool(*value)
}

func runDoctor(arguments []string, stdout, stderr io.Writer, executable string) int {
	flags := flag.NewFlagSet("gomad doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root to verify")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		fmt.Fprintf(stderr, "resolve gomad executable path: %v\n", err)
		return 3
	}
	artifactRoot, err := filepath.Abs(*artifacts)
	if err != nil {
		fmt.Fprintf(stderr, "resolve artifact directory: %v\n", err)
		return 2
	}
	resolved, err := toolchain.ResolveInstallation(toolchain.InstallationSpec{
		Executable: executable, ExplicitToolchainRoot: *toolchainRoot, EnvironmentToolchainRoot: os.Getenv("GOMAD3_TOOLCHAIN_DIR"),
	})
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "resolve Gomad installation: %v\n", err); writeErr != nil {
			return 3
		}
		return 2
	}
	report := Check(Config{
		ToolchainRoot: resolved.ToolchainRoot, InstallationSource: resolved.Source, RepairInstruction: resolved.RepairInstruction,
		RunnerPath: executable, ArtifactRoot: artifactRoot, HostOS: runtime.GOOS, HostArch: runtime.GOARCH,
	})
	if *jsonOutput {
		encoded, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			fmt.Fprintf(stderr, "encode doctor report: %v\n", marshalErr)
			return 3
		}
		fmt.Fprintf(stdout, "%s\n", encoded)
	} else {
		fmt.Fprintf(stdout, "gomad doctor: available=%t host=%s go=%s toolchain=%s runner=%s boundary=%s\n", report.Available, report.Host, report.GoVersion, report.ToolchainBuild, report.RunnerBuild, report.BoundaryManifestVersion)
		for _, check := range report.Checks {
			fmt.Fprintf(stdout, "%-10s %-5s %s\n", check.Name, check.Status, check.Detail)
		}
		if _, err := fmt.Fprintf(stdout, "installation: source=%s toolchain=%s\nrepair: %s\n", report.InstallationSource, report.ToolchainRoot, report.RepairInstruction); err != nil {
			return 3
		}
	}
	if !report.Available {
		return 1
	}
	return 0
}

func runExplore(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad explore", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	strategy := flags.String("strategy", string(runner.StrategySeed), "seed, choice-exploration, or simulation-exploration")
	seeds := flags.String("seeds", "1", "seed set or inclusive ranges")
	count := flags.Uint64("count", 0, "explore seeds 0 through N-1")
	maxRuns := flags.Uint64("max-executions", 0, "maximum exploration candidates")
	maxChoiceDepth := flags.Uint64("max-choice-depth", 0, "maximum forced choice decisions")
	maxForcedDecisions := flags.Uint64("max-forced-decisions", 0, "maximum combined forced decisions")
	maxRuntimeDecisions := flags.Uint64("max-runtime-decisions", 0, "maximum runtime decision ordinal")
	maxScenarioDecisions := flags.Uint64("max-scenario-decisions", 0, "maximum scenario decision ordinal")
	maxNetworkDecisions := flags.Uint64("max-network-decisions", 0, "maximum network decision ordinal")
	maxStorageDecisions := flags.Uint64("max-storage-decisions", 0, "maximum storage decision ordinal")
	maxFaultDecisions := flags.Uint64("max-fault-decisions", 0, "maximum fault decision ordinal")
	maxCrashDecisions := flags.Uint64("max-crash-decisions", 0, "maximum crash-state decision ordinal")
	parallel := flags.Int("parallel", min(runtime.NumCPU(), 8), "maximum active targets")
	runTimeout := flags.Duration("execution-timeout", 30*time.Second, "per-execution host deadline")
	overallTimeout := flags.Duration("overall-timeout", 10*time.Minute, "complete exploration host deadline")
	terminateGrace := flags.Duration("terminate-grace", 2*time.Second, "termination grace inside deadlines")
	onFailure := flags.String("on-failure", string(runner.PolicyFirst), "first, budget, or all")
	failureBudget := flags.Uint64("failure-budget", 1, "distinct failure signature threshold")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	capabilityMode := flags.String("capability-mode", string(target.CapabilityModeClosure), "closure, linked, or guarded capability assessment")
	jsonOutput := flags.Bool("json", false, "emit stable JSON events")
	planOnly := flags.Bool("__plan", false, "create a campaign plan")
	planOutput := flags.String("output", "", "campaign plan output")
	choices := flags.Bool("choices", false, "record bounded runtime choices")
	coverage := flags.String("coverage", string(runner.CoverageNone), "none, semantic, choice, or semantic+choice")
	guide := flags.Bool("guide", false, "guide selection from a bounded coverage corpus")
	corpus := flags.String("corpus", "", "guided coverage corpus directory")
	keepSuccesses := flags.String("keep-successes", string(runner.KeepSuccessesNone), "none, novel, or all")
	successLimit := flags.Uint64("success-limit", 0, "maximum retained successful executions")
	outputLimit := byteSize(8 << 20)
	worldLimit := byteSize(64 << 20)
	successBytes := byteSize(0)
	choiceLimit := byteSize(8 << 20)
	explorationLimit := byteSize(0)
	explorationResultLimit := byteSize(0)
	flags.Var(&outputLimit, "output-limit", "retained bytes per output stream")
	flags.Var(&worldLimit, "world-transition-limit", "World transition capacity")
	flags.Var(&successBytes, "success-bytes", "total retained successful-execution bytes")
	flags.Var(&choiceLimit, "choice-bytes", "runtime choice trace capacity")
	flags.Var(&explorationLimit, "max-exploration-bytes", "maximum live choice-exploration bytes")
	flags.Var(&explorationResultLimit, "max-exploration-result-bytes", "maximum combined result bytes per candidate")
	var environment stringList
	var buildTags stringList
	var ioROMounts stringList
	var requiredSemanticProbes stringList
	flags.Var(&environment, "env", "target NAME=VALUE")
	flags.Var(&buildTags, "build-tag", "validated Go build tag")
	flags.Var(&ioROMounts, "io-ro-mount", "read-only HOST_DIRECTORY=TARGET_DIRECTORY mapping")
	flags.Var(&requiredSemanticProbes, "require-probe", "required semantic probe (requires --coverage=semantic)")
	if err := flags.Parse(arguments); err != nil {
		reporter := newExploreReporter(*jsonOutput, stdout, stderr)
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		if !*jsonOutput {
			flags.SetOutput(stderr)
			flags.Usage()
		}
		return 2
	}
	reporter := newExploreReporter(*jsonOutput, stdout, stderr)
	if !*planOnly && *planOutput != "" {
		if writeErr := reporter.Error("invalid_input", errors.New("--output is only valid with gomad plan")); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	resolvedCapabilityMode, err := parseCapabilityMode(*capabilityMode)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	var seedsSet, countSet, coverageSet, choiceLimitSet, maxRunsSet, maxChoiceDepthSet, maxForcedDecisionsSet, maxExplorationBytesSet, maxExplorationResultBytesSet bool
	var runtimeLimitSet, scenarioLimitSet, networkLimitSet, storageLimitSet, faultLimitSet, crashLimitSet bool
	flags.Visit(func(visited *flag.Flag) {
		switch visited.Name {
		case "seeds":
			seedsSet = true
		case "count":
			countSet = true
		case "coverage":
			coverageSet = true
		case "choice-bytes":
			choiceLimitSet = true
		case "max-executions":
			maxRunsSet = true
		case "max-choice-depth":
			maxChoiceDepthSet = true
		case "max-forced-decisions":
			maxForcedDecisionsSet = true
		case "max-exploration-bytes":
			maxExplorationBytesSet = true
		case "max-exploration-result-bytes":
			maxExplorationResultBytesSet = true
		case "max-runtime-decisions":
			runtimeLimitSet = true
		case "max-scenario-decisions":
			scenarioLimitSet = true
		case "max-network-decisions":
			networkLimitSet = true
		case "max-storage-decisions":
			storageLimitSet = true
		case "max-fault-decisions":
			faultLimitSet = true
		case "max-crash-decisions":
			crashLimitSet = true
		}
	})
	resolvedStrategy, resolvedChoices, err := resolveExploreStrategy(exploreStrategyOptions{
		Value: *strategy, Seeds: *seeds, CountSet: countSet, Guide: *guide, Choices: *choices,
		MaxExecutions: *maxRuns, MaxChoiceDepth: *maxChoiceDepth, MaxForcedDecisions: *maxForcedDecisions,
		MaxExplorationBytes: explorationLimit, MaxExplorationResultBytes: explorationResultLimit,
		SimulationDimensionLimits: runner.SimulationDimensionLimits{
			Runtime: *maxRuntimeDecisions, Scenario: *maxScenarioDecisions, Network: *maxNetworkDecisions,
			Storage: *maxStorageDecisions, Fault: *maxFaultDecisions, Crash: *maxCrashDecisions,
		},
		MaxExecutionsSet: maxRunsSet, MaxChoiceDepthSet: maxChoiceDepthSet, MaxForcedDecisionsSet: maxForcedDecisionsSet,
		MaxExplorationBytesSet: maxExplorationBytesSet, MaxExplorationResultBytesSet: maxExplorationResultBytesSet,
		RuntimeLimitSet: runtimeLimitSet, ScenarioLimitSet: scenarioLimitSet, NetworkLimitSet: networkLimitSet,
		StorageLimitSet: storageLimitSet, FaultLimitSet: faultLimitSet, CrashLimitSet: crashLimitSet,
	})
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	resolvedSeeds, err := resolveExploreSeeds(*seeds, *count, seedsSet, countSet)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	resolvedCoverage, err := resolveExploreGuidance(*guide, *corpus, *coverage, coverageSet)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	coverageMode, err := resolveExploreCoverage(resolvedCoverage, requiredSemanticProbes)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	resolvedChoiceLimit, err := resolveChoiceTrace(resolvedChoices, choiceLimit, choiceLimitSet)
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	if (coverageMode == runner.CoverageChoice || coverageMode == runner.CoverageSemanticChoice) && resolvedChoiceLimit == 0 {
		if writeErr := reporter.Error("invalid_input", fmt.Errorf("--coverage=%s requires --choices", coverageMode)); writeErr != nil {
			if _, printErr := fmt.Fprintln(stderr, writeErr); printErr != nil {
				return 3
			}
			return 3
		}
		return 2
	}
	parsedTarget, err := parseTarget(flags.Args())
	if err != nil {
		if writeErr := reporter.Error("invalid_input", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return 2
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		if writeErr := reporter.Error("runner_failure", fmt.Errorf("resolve working directory: %w", err)); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
		}
		return 3
	}
	toolchain, executable, runnerBuild, err := localIdentity(*toolchainRoot)
	if err != nil {
		if writeErr := reporter.Error("runner_failure", err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
		}
		return 3
	}
	config := runner.CampaignSpec{
		Strategy: resolvedStrategy, Seeds: resolvedSeeds, Parallel: *parallel, ExecutionTimeout: *runTimeout, OverallTimeout: *overallTimeout, TerminateGrace: *terminateGrace,
		OnFailure: runner.FailurePolicy(*onFailure), FailureBudget: *failureBudget, OutputLimit: uint64(outputLimit), WorldTransitionLimit: uint64(worldLimit),
		ChoiceTraceLimit: resolvedChoiceLimit, MaxExecutions: *maxRuns, MaxChoiceDepth: *maxChoiceDepth, MaxForcedDecisions: *maxForcedDecisions,
		MaxExplorationBytes: uint64(explorationLimit), MaxExplorationResultBytes: uint64(explorationResultLimit),
		SimulationDimensionLimits: runner.SimulationDimensionLimits{
			Runtime: *maxRuntimeDecisions, Scenario: *maxScenarioDecisions, Network: *maxNetworkDecisions,
			Storage: *maxStorageDecisions, Fault: *maxFaultDecisions, Crash: *maxCrashDecisions,
		},
		Artifacts: *artifacts, Environment: environment, IOROMounts: ioROMounts, SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"}, RunnerBuild: runnerBuild,
		Coverage: coverageMode, RequiredSemanticProbes: requiredSemanticProbes,
		KeepSuccesses: runner.KeepSuccesses(*keepSuccesses), SuccessArtifactLimit: *successLimit, SuccessBytesLimit: uint64(successBytes),
		Guide: *guide, Corpus: *corpus,
		Progress: reporter.Progress, ProgressInterval: 5 * time.Second,
		Target: target.Spec{
			Kind: parsedTarget.kind, Source: parsedTarget.source, Provenance: parsedTarget.provenance, Args: parsedTarget.arguments,
			BuildTags: buildTags, WorkingDir: workingDirectory, ToolchainRoot: toolchain, CapabilityMode: resolvedCapabilityMode,
		},
	}
	if *planOnly {
		if *planOutput == "" {
			if writeErr := reporter.Error("invalid_input", errors.New("gomad plan requires --output FILE")); writeErr != nil {
				fmt.Fprintln(stderr, writeErr)
				return 3
			}
			return 2
		}
		planned, err := runner.CreateCampaignPlan(context.Background(), runner.CampaignPlanSpec{Campaign: config, Output: *planOutput})
		if err != nil {
			classification := classifyExploreError(err)
			if writeErr := reporter.Error(classification, err); writeErr != nil {
				fmt.Fprintln(stderr, writeErr)
				return 3
			}
			return exploreErrorStatus(classification)
		}
		if *jsonOutput {
			encoded, err := json.Marshal(planned)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 3
			}
			if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
				return 3
			}
		} else if _, err := fmt.Fprintf(stdout, "gomad plan: path=%s bundle=%s sha256=%s selected=%d target=%s\n", planned.Path, planned.BundlePath, planned.SHA256, planned.SelectionCount, planned.TargetSHA256); err != nil {
			return 3
		}
		return 0
	}
	summary, err := runner.Explore(context.Background(), config)
	if err != nil {
		if summary.ChoiceTrace != nil {
			fmt.Fprintf(stderr, "gomad:%s\n", formatChoiceTrace(summary.ChoiceTrace))
		}
		classification := classifyExploreError(err)
		if writeErr := reporter.Error(classification, err); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 3
		}
		return exploreErrorStatus(classification)
	}
	if err := reporter.Result(summary); err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	if summary.Failures != 0 {
		return 1
	}
	return 0
}

func runPlan(arguments []string, stdout, stderr io.Writer) int {
	return runExplore(append([]string{"--__plan", "--on-failure=all"}, arguments...), stdout, stderr)
}

type exploreStrategyOptions struct {
	Value                        string
	Seeds                        string
	CountSet                     bool
	Guide                        bool
	Choices                      bool
	MaxExecutions                uint64
	MaxChoiceDepth               uint64
	MaxForcedDecisions           uint64
	MaxExplorationBytes          byteSize
	MaxExplorationResultBytes    byteSize
	SimulationDimensionLimits    runner.SimulationDimensionLimits
	MaxExecutionsSet             bool
	MaxChoiceDepthSet            bool
	MaxForcedDecisionsSet        bool
	MaxExplorationBytesSet       bool
	MaxExplorationResultBytesSet bool
	RuntimeLimitSet              bool
	ScenarioLimitSet             bool
	NetworkLimitSet              bool
	StorageLimitSet              bool
	FaultLimitSet                bool
	CrashLimitSet                bool
}

func resolveExploreStrategy(options exploreStrategyOptions) (runner.Strategy, bool, error) {
	strategy := runner.Strategy(options.Value)
	if strategy == "" {
		strategy = runner.StrategySeed
	}
	switch strategy {
	case runner.StrategySeed:
		if options.hasCombinedBounds() {
			return "", false, errors.New("simulation exploration bounds require --strategy=simulation-exploration")
		}
		if options.MaxExecutionsSet || options.MaxChoiceDepthSet || options.MaxExplorationBytesSet {
			return "", false, errors.New("exploration bounds require --strategy=choice-exploration")
		}
		return strategy, options.Choices, nil
	case runner.StrategyChoiceExploration:
		if options.hasCombinedBounds() {
			return "", false, errors.New("simulation exploration bounds require --strategy=simulation-exploration")
		}
		if options.CountSet {
			return "", false, errors.New("--strategy=choice-exploration does not accept --count")
		}
		selection, err := runner.ParseSeeds(options.Seeds)
		if err != nil {
			return "", false, err
		}
		if selection.Count() != 1 {
			return "", false, errors.New("--strategy=choice-exploration requires exactly one base seed")
		}
		if options.Guide {
			return "", false, errors.New("--strategy=choice-exploration does not support --guide")
		}
		if !options.MaxExecutionsSet || options.MaxExecutions == 0 {
			return "", false, errors.New("--strategy=choice-exploration requires an explicit positive --max-executions")
		}
		if !options.MaxChoiceDepthSet || options.MaxChoiceDepth == 0 {
			return "", false, errors.New("--strategy=choice-exploration requires an explicit positive --max-choice-depth")
		}
		if !options.MaxExplorationBytesSet || options.MaxExplorationBytes == 0 {
			return "", false, errors.New("--strategy=choice-exploration requires an explicit positive --max-exploration-bytes")
		}
		return strategy, true, nil
	case runner.StrategySimulationExploration:
		if options.CountSet {
			return "", false, errors.New("--strategy=simulation-exploration does not accept --count")
		}
		selection, err := runner.ParseSeeds(options.Seeds)
		if err != nil {
			return "", false, err
		}
		if selection.Count() != 1 {
			return "", false, errors.New("--strategy=simulation-exploration requires exactly one base seed")
		}
		if options.Guide {
			return "", false, errors.New("--strategy=simulation-exploration does not support --guide")
		}
		if options.MaxChoiceDepthSet {
			return "", false, errors.New("--strategy=simulation-exploration does not accept --max-choice-depth")
		}
		for _, bound := range []struct {
			name  string
			value uint64
			set   bool
		}{
			{name: "--max-executions", value: options.MaxExecutions, set: options.MaxExecutionsSet},
			{name: "--max-forced-decisions", value: options.MaxForcedDecisions, set: options.MaxForcedDecisionsSet},
			{name: "--max-exploration-bytes", value: uint64(options.MaxExplorationBytes), set: options.MaxExplorationBytesSet},
			{name: "--max-exploration-result-bytes", value: uint64(options.MaxExplorationResultBytes), set: options.MaxExplorationResultBytesSet},
			{name: "--max-runtime-decisions", value: options.SimulationDimensionLimits.Runtime, set: options.RuntimeLimitSet},
			{name: "--max-scenario-decisions", value: options.SimulationDimensionLimits.Scenario, set: options.ScenarioLimitSet},
			{name: "--max-network-decisions", value: options.SimulationDimensionLimits.Network, set: options.NetworkLimitSet},
			{name: "--max-storage-decisions", value: options.SimulationDimensionLimits.Storage, set: options.StorageLimitSet},
			{name: "--max-fault-decisions", value: options.SimulationDimensionLimits.Fault, set: options.FaultLimitSet},
			{name: "--max-crash-decisions", value: options.SimulationDimensionLimits.Crash, set: options.CrashLimitSet},
		} {
			if !bound.set || bound.value == 0 {
				return "", false, fmt.Errorf("--strategy=simulation-exploration requires an explicit positive %s", bound.name)
			}
		}
		return strategy, true, nil
	default:
		return "", false, fmt.Errorf("unknown exploration strategy %q", options.Value)
	}
}

func (options exploreStrategyOptions) hasCombinedBounds() bool {
	return options.MaxForcedDecisionsSet || options.MaxExplorationResultBytesSet || options.RuntimeLimitSet || options.ScenarioLimitSet || options.NetworkLimitSet || options.StorageLimitSet || options.FaultLimitSet || options.CrashLimitSet
}

func resolveExploreGuidance(enabled bool, corpus, coverage string, coverageSet bool) (string, error) {
	if !enabled {
		if corpus != "" {
			return "", fmt.Errorf("--corpus requires --guide")
		}
		return coverage, nil
	}
	if corpus == "" {
		return "", fmt.Errorf("--guide requires --corpus DIR")
	}
	if coverageSet && coverage != string(runner.CoverageSemantic) && coverage != string(runner.CoverageChoice) && coverage != string(runner.CoverageSemanticChoice) {
		return "", errors.New("--guide requires semantic or choice coverage")
	}
	if !coverageSet {
		return string(runner.CoverageSemantic), nil
	}
	return coverage, nil
}

func resolveExploreSeeds(seeds string, count uint64, seedsSet, countSet bool) (string, error) {
	if seedsSet && countSet {
		return "", fmt.Errorf("--count and --seeds are mutually exclusive")
	}
	if !countSet {
		return seeds, nil
	}
	if count == 0 {
		return "", fmt.Errorf("--count must be greater than zero")
	}
	if count == 1 {
		return "0", nil
	}
	return "0-" + strconv.FormatUint(count-1, 10), nil
}

func resolveExploreCoverage(value string, required []string) (runner.CoverageMode, error) {
	mode := runner.CoverageMode(value)
	switch mode {
	case runner.CoverageNone, runner.CoverageChoice:
		if len(required) != 0 {
			return "", fmt.Errorf("--require-probe requires --coverage=semantic")
		}
	case runner.CoverageSemantic, runner.CoverageSemanticChoice:
		if _, err := deterministicio.MissingRequiredSemanticProbes(deterministicio.SemanticCoverage{}, required); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown coverage mode %q", value)
	}
	return mode, nil
}

func resolveChoiceTrace(enabled bool, limit byteSize, limitSet bool) (uint64, error) {
	if !enabled {
		if limitSet {
			return 0, fmt.Errorf("--choice-bytes requires --choices")
		}
		return 0, nil
	}
	if limit < runner.MinimumChoiceTraceBytes || limit > runner.MaximumChoiceTraceBytes {
		return 0, fmt.Errorf("--choice-bytes must be between %d bytes and 64MiB", runner.MinimumChoiceTraceBytes)
	}
	return uint64(limit), nil
}

func runReplay(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gomad replay", flag.ContinueOnError)
	flags.SetOutput(stderr)
	verifyOnly := flags.Bool("verify-only", false, "validate without executing the target")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	toolchain, executable, _, err := localIdentity(*toolchainRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	result, err := runner.Replay(context.Background(), runner.ReplaySpec{
		ArtifactPath: flags.Arg(0), VerifyOnly: *verifyOnly, ToolchainRoot: toolchain, SupervisorCommand: []string{executable, "__supervisor"},
	})
	if err != nil {
		var preflightError *runner.ReplayPreflightError
		if errors.As(err, &preflightError) {
			fmt.Fprintln(stderr, err)
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 3
	}
	if *verifyOnly {
		fmt.Fprintf(stdout, "gomad: verified %s\n", result.Artifact.Path)
		return 0
	}
	status, err := reportReplayResult(stdout, result)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	return status
}

type minimizeDependencies struct {
	identity func(string) (string, string, string, error)
	minimize func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error)
}

func runMinimize(arguments []string, stdout, stderr io.Writer) int {
	return runMinimizeWith(arguments, stdout, stderr, minimizeDependencies{identity: localIdentity, minimize: runner.Minimize})
}

func runMinimizeWith(arguments []string, stdout, stderr io.Writer, dependencies minimizeDependencies) int {
	flags := flag.NewFlagSet("gomad minimize", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	artifacts := flags.String("artifacts", ".gomad/artifacts", "artifact root")
	toolchainRoot := flags.String("toolchain-root", "", "absolute pinned toolchain root")
	attemptBudget := flags.Uint64("attempt-budget", 64, "maximum fresh-process minimization candidates")
	maximumBytes := byteSize(0)
	flags.Var(&maximumBytes, "max-bytes", "maximum minimized artifact bytes")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 || *attemptBudget == 0 {
		return writeCommandError(stderr, 2, "%s", usage)
	}
	artifactRoot, err := filepath.Abs(*artifacts)
	if err != nil {
		return writeCommandError(stderr, 2, "resolve artifact directory: %v\n", err)
	}
	resolvedToolchain, executable, _, err := dependencies.identity(*toolchainRoot)
	if err != nil {
		return writeCommandError(stderr, 3, "%v\n", err)
	}
	result, err := dependencies.minimize(context.Background(), runner.MinimizeSpec{
		ArtifactPath: flags.Arg(0), OutputRoot: filepath.Join(artifactRoot, "minimized"),
		AttemptBudget: *attemptBudget, MaximumBytes: uint64(maximumBytes), ToolchainRoot: resolvedToolchain,
		SupervisorCommand: []string{executable, "__supervisor"},
	})
	if err != nil {
		var preflight *runner.ReplayPreflightError
		if errors.As(err, &preflight) {
			return writeCommandError(stderr, 2, "%v\n", err)
		}
		return writeCommandError(stderr, 3, "%v\n", err)
	}
	if *jsonOutput {
		encoded, err := json.Marshal(struct {
			ArtifactPath  string                         `json:"artifact_path"`
			RecordHash    record.SHA256                  `json:"record_hash"`
			Changed       bool                           `json:"changed"`
			Attempts      uint64                         `json:"attempts"`
			AttemptBudget uint64                         `json:"attempt_budget"`
			Accepted      []record.MinimizationReduction `json:"accepted"`
			StopReason    string                         `json:"stop_reason"`
		}{
			ArtifactPath: result.Artifact.Path, RecordHash: result.Artifact.Manifest.RecordHash, Changed: result.Changed,
			Attempts: result.Attempts, AttemptBudget: result.AttemptBudget, Accepted: result.Accepted, StopReason: result.StopReason,
		})
		if err != nil {
			return writeCommandError(stderr, 3, "encode minimization result: %v\n", err)
		}
		if _, err := fmt.Fprintf(stdout, "%s\n", encoded); err != nil {
			return 3
		}
		return 0
	}
	if _, err := fmt.Fprintf(
		stdout, "gomad minimize: changed=%t attempts=%d/%d accepted=%d stop=%s artifact=%s\n",
		result.Changed, result.Attempts, result.AttemptBudget, len(result.Accepted), result.StopReason, result.Artifact.Path,
	); err != nil {
		return 3
	}
	return 0
}

func reportReplayResult(output io.Writer, result runner.ReplayResult) (int, error) {
	if !result.Match {
		_, err := fmt.Fprintf(output, "gomad: reproduced=false divergence=%s choice-replay=%s\n", result.Divergence, result.ChoiceReplayStatus)
		return 1, err
	}
	if result.Diagnostic {
		_, err := fmt.Fprintf(output, "gomad: reproduced=true diagnostic=true result=watchdog_observation choice-replay=%s\n", result.ChoiceReplayStatus)
		return 1, err
	}
	if result.Artifact.Manifest.Outcome.Domain == "success" {
		_, err := fmt.Fprintf(output, "gomad: reproduced=true diagnostic=false result=success choice-replay=%s\n", result.ChoiceReplayStatus)
		return 0, err
	}
	_, err := fmt.Fprintf(output, "gomad: reproduced=true diagnostic=false result=target_failure choice-replay=%s\n", result.ChoiceReplayStatus)
	return 1, err
}

func parseTarget(arguments []string) (targetInput, error) {
	if len(arguments) == 0 {
		return targetInput{}, fmt.Errorf("target kind is required")
	}
	switch arguments[0] {
	case string(target.KindExec):
		if len(arguments) < 5 || arguments[1] != "--provenance" || arguments[2] == "" || arguments[3] != "--" || arguments[4] == "" {
			return targetInput{}, fmt.Errorf("exec requires --provenance FILE -- BINARY [ARG ...]")
		}
		return targetInput{kind: target.KindExec, source: arguments[4], provenance: arguments[2], arguments: append([]string(nil), arguments[5:]...)}, nil
	case string(target.KindGoRun), string(target.KindGoTest):
		if len(arguments) < 2 || arguments[1] == "" {
			return targetInput{}, fmt.Errorf("%s requires one package", arguments[0])
		}
		remaining := arguments[2:]
		if len(remaining) > 0 {
			if remaining[0] != "--" {
				return targetInput{}, fmt.Errorf("%s target arguments require -- separator", arguments[0])
			}
			remaining = remaining[1:]
		}
		return targetInput{kind: target.Kind(arguments[0]), source: arguments[1], arguments: append([]string(nil), remaining...)}, nil
	default:
		return targetInput{}, fmt.Errorf("unknown target kind %q", arguments[0])
	}
}

func parseCapabilityMode(value string) (target.CapabilityMode, error) {
	mode := target.CapabilityMode(value)
	switch mode {
	case target.CapabilityModeClosure, target.CapabilityModeLinked, target.CapabilityModeGuarded:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown capability mode %q", value)
	}
}

func localIdentity(explicitToolchainRoot string) (toolchainRoot, executable, runnerBuild string, err error) {
	executable, err = os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("resolve gomad executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve gomad executable path: %w", err)
	}
	resolved, err := toolchain.ResolveInstallation(toolchain.InstallationSpec{
		Executable: executable, ExplicitToolchainRoot: explicitToolchainRoot, EnvironmentToolchainRoot: os.Getenv("GOMAD3_TOOLCHAIN_DIR"),
	})
	if err != nil {
		return "", "", "", fmt.Errorf("resolve Gomad installation: %w", err)
	}
	toolchainRoot = resolved.ToolchainRoot
	bytes, err := os.ReadFile(executable)
	if err != nil {
		return "", "", "", fmt.Errorf("hash gomad executable: %w", err)
	}
	digest := sha256.Sum256(bytes)
	return toolchainRoot, executable, fmt.Sprintf("sha256:%x", digest), nil
}
