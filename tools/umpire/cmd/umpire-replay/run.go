package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/binding"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/internal/cli"
	"go.temporal.io/server/tools/umpire/replay"
)

const (
	defaultTimeout   = 30 * time.Minute
	defaultModelRoot = "model"
	// The bridge the model package builds, relative to the model root.
	bridgeRelativePath = ".lake/build/bin/umpire-replay-bridge"
)

// config is what the caller names: the subject's files, the set and the Query or target the bridge
// recovers it by, the deployment as `umpire-run` names it, and where proposals go. The limits are
// fixed and named in the report; nothing here names a Driver, a checker, an edit or a
// compatibility option.
type config struct {
	Case          string
	Run           string
	Set           string
	Query         string
	Target        string
	Deployment    binding.Deployment
	ModelRoot     string
	PromotionRoot string
	Bridge        string
	Timeout       time.Duration
}

// environmentFor builds what a replay runs against from the configuration; the real one binds the
// deployment the flags name, and a test supplies its own.
type environmentFor func(configuration config, stderr io.Writer) replay.Environment

// Run is the whole command: refuse the command line before reading or opening anything, read the
// subject, execute the replay, write the report whole, and exit by what it says.
func Run(arguments []string, stdout, stderr io.Writer, environment environmentFor) int {
	configuration, err := parseConfig(arguments, stderr)
	if err != nil {
		return replay.ExitToolingFailure
	}
	caseBytes, err := os.ReadFile(configuration.Case)
	if err != nil {
		cli.WriteLine(stderr, "--case: %s", err)
		return replay.ExitToolingFailure
	}
	runBytes, err := os.ReadFile(configuration.Run)
	if err != nil {
		cli.WriteLine(stderr, "--run: %s", err)
		return replay.ExitToolingFailure
	}
	ctx, cancel := cli.Interruptible(context.Background(), configuration.Timeout)
	defer cancel()

	report := replay.Execute(ctx, replay.Request{
		Case: caseBytes, Run: runBytes, Set: configuration.Set,
		Named: replay.Named{Query: configuration.Query, Target: configuration.Target},
	}, environment(configuration, stderr))
	rendered, err := report.Render()
	if err != nil {
		cli.WriteLine(stderr, "render report: %s", err)
		return replay.ExitToolingFailure
	}
	code := report.ExitCodeWithin(len(rendered))
	if report.OverCap(len(rendered)) {
		cli.WriteLine(stderr, "the report is %d bytes, over the cap of %d, and is written whole", len(rendered), report.Limits.ReportBytes)
	}
	if _, err := stdout.Write(rendered); err != nil {
		cli.WriteLine(stderr, "write report: %s", err)
		return replay.ExitToolingFailure
	}
	cli.WriteLine(stderr, "replay %s", describe(report))
	return code
}

// describe is the one summary line on stderr.
func describe(report replay.Report) string {
	switch {
	case report.Failure != "":
		return "failed: " + report.Failure
	case report.Reduction != nil && report.Reduction.Failure != "":
		return "failed: " + report.Reduction.Failure
	case report.Reduction != nil && report.Reduction.Stopped && !report.Reduction.Attempted:
		return "stopped: " + report.Reduction.NotAttempted
	case report.Admission.Status != replay.StatusAdmitted:
		return fmt.Sprintf("rejected (%s): %s", report.Admission.Reason, report.Admission.Detail)
	case report.Reproduction == nil:
		return "not rerun"
	case report.Reduction == nil || !report.Reduction.Attempted:
		return fmt.Sprintf("%s, not reduced", report.Reproduction.Class)
	default:
		return fmt.Sprintf("%s, reduction %s, proposal %s", report.Reproduction.Class, report.Reduction.Status, report.Proposal.Status)
	}
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	if len(arguments) == 0 || arguments[0] != "run" {
		cli.WriteLine(stderr, "usage: umpire-replay run --case <case.json> --run <run.json> --set <set> (--query <query> | --target <key>) --grpc <address> --http <address> --namespace <namespace> --task-queue <queue> [flags]")
		return config{}, errors.New("the subcommand is run")
	}
	var configuration config
	flags := flag.NewFlagSet("umpire-replay run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.Case, "case", "", "the subject's Case, canonical or its persisted form")
	flags.StringVar(&configuration.Run, "run", "", "the Run recorded against the Case, with the Profile identity it was prepared under")
	flags.StringVar(&configuration.Set, "set", "", "the set whose Query produced the Case")
	flags.StringVar(&configuration.Query, "query", "", "the functional set's Query the Case realizes")
	flags.StringVar(&configuration.Target, "target", "", "the exploratory set's target key the Case's candidate was planned for")
	binding.RegisterFlags(flags, &configuration.Deployment, "the replay's Cases")
	flags.StringVar(&configuration.ModelRoot, "model-root", defaultModelRoot, "the model package the bridge runs in")
	flags.StringVar(&configuration.PromotionRoot, "promotion-root", "", "a directory outside the model to write the proposal under; none writes nothing")
	flags.StringVar(&configuration.Bridge, "bridge", "", "the replay bridge executable (default <model-root>/"+bridgeRelativePath+")")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "bound on the whole replay")
	if err := flags.Parse(arguments[1:]); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		cli.WriteLine(stderr, "umpire-replay run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	for _, required := range []struct{ flag, value string }{
		{"--case", configuration.Case}, {"--run", configuration.Run}, {"--set", configuration.Set},
	} {
		if required.value == "" {
			cli.WriteLine(stderr, "%s is required", required.flag)
			return config{}, fmt.Errorf("missing %s", required.flag)
		}
	}
	if (configuration.Query == "") == (configuration.Target == "") {
		cli.WriteLine(stderr, "exactly one of --query and --target is required")
		return config{}, errors.New("query or target")
	}
	if missing := binding.Missing(configuration.Deployment); len(missing) > 0 {
		cli.WriteLine(stderr, "%s is required", missing[0])
		return config{}, fmt.Errorf("missing %s", missing[0])
	}
	if configuration.Timeout <= 0 {
		cli.WriteLine(stderr, "--timeout must be positive")
		return config{}, errors.New("non-positive timeout")
	}
	modelRoot, err := filepath.Abs(configuration.ModelRoot)
	if err != nil {
		cli.WriteLine(stderr, "--model-root: %s", err)
		return config{}, err
	}
	configuration.ModelRoot = modelRoot
	if configuration.Bridge == "" {
		configuration.Bridge = filepath.Join(modelRoot, filepath.FromSlash(bridgeRelativePath))
	} else if configuration.Bridge, err = filepath.Abs(configuration.Bridge); err != nil {
		cli.WriteLine(stderr, "--bridge: %s", err)
		return config{}, err
	}
	if configuration.PromotionRoot != "" {
		// A proposal is for review, never an installed regression: the model never receives one,
		// whatever symlink the root is reached through.
		if configuration.PromotionRoot, err = cli.OutsideModel("--promotion-root", configuration.PromotionRoot, modelRoot); err != nil {
			cli.WriteLine(stderr, "%s", err)
			return config{}, err
		}
	}
	return configuration, nil
}

// deploymentEnvironment is the real environment: preparation under the deployment's names with no
// connection, the bridge spawned in the model package on a context that outlives the replay (its
// last answer is still read after a stop), and the deployment opened only when Execute asks.
func deploymentEnvironment(configuration config, stderr io.Writer) replay.Environment {
	deployment := configuration.Deployment
	handlerQueue := binding.HandlerQueue(deployment)
	return replay.Environment{
		Prepare: func(identity string, source *testpilotspb.Case) (*testpilot.PreparedCase, error) {
			prepared, err := binding.Prepare(deployment, handlerQueue, identity, source)
			if err != nil {
				return nil, err
			}
			return prepared.Case, nil
		},
		StartBridge: func(ctx context.Context) (*replay.Bridge, error) {
			return replay.StartBridge(context.WithoutCancel(ctx), campaign.Options{
				Executable: configuration.Bridge, Dir: configuration.ModelRoot, Stderr: stderr,
			})
		},
		OpenBinder: func(ctx context.Context) (campaign.Binder, func(context.Context) error, error) {
			opened, err := binding.Open(ctx, deployment, handlerQueue)
			if err != nil {
				return nil, nil, err
			}
			return campaign.CampaignBinder{Campaign: opened}, opened.Close, nil
		},
		PromotionRoot: configuration.PromotionRoot,
		Limits:        replay.DefaultLimits,
		Progress:      stderr,
	}
}
