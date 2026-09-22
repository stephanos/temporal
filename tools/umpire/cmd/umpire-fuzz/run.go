package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/umpire/binding"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// Exit codes. A counterexample or violated coverage outranks a stop or a cap, because the finding
// is what the campaign ran for; a tooling failure outranks everything, because nothing it reports
// can be trusted.
const (
	exitExhausted    = 0
	exitViolated     = 1
	exitLimitOrStop  = 2
	exitToolingError = 3
)

const (
	defaultTimeout    = 30 * time.Minute
	defaultRunTimeout = 5 * time.Minute
	defaultModelRoot  = "model"
	// The bridge the model package builds, relative to the model root.
	bridgeRelativePath = ".lake/build/bin/umpire-explore"
	// minimumReportBytes is the floor a report cap cannot go below: the terminal-only summary
	// (status, set, Profile, machine, budget, Limits, counters, coverage counts and the
	// counterexamples by identity) is always written in full, so a cap under it could only be met by truncation, which is never
	// done. The floor covers the fixed fields; a long failure message, a long set name or many
	// counterexamples can still carry that summary past the cap, which is then said on stderr
	// rather than hidden.
	minimumReportBytes = 1024
)

// config is what the caller names: the set, the deployment as `umpire-run` names it, and the
// campaign's caps. None of it names a target or widens a declared Limit.
type config struct {
	Set        string
	Deployment binding.Deployment
	ModelRoot  string
	// PromotionRoot is where compiled proposals are written, one file each; empty writes none.
	PromotionRoot string
	Bridge        string
	Timeout       time.Duration
	Caps          campaign.Caps
}

// bound is one opened campaign: its bridge, initialized over the set, the binder its candidates
// bind through, and the release of everything the opening created.
type bound struct {
	bridge  *campaign.Bridge
	binder  campaign.Binder
	profile string
	opened  campaign.Initialized
	release func(ctx context.Context) error
}

// opener opens one campaign. The real one binds the deployment and spawns the bridge; a test
// supplies its own.
type opener func(ctx context.Context, configuration config, stderr io.Writer) (*bound, error)

// summary is the command's canonical output: the coordinator's terminal, its counters, what the
// bridge's ledger says per target, the counterexamples, and one line per candidate. Coverage is
// the bridge's alone; nothing unexecuted, inconclusive or cleanup-uncertain appears as covered.
type summary struct {
	Status          string                  `json:"status"`
	Limit           string                  `json:"limit,omitempty"`
	Failure         string                  `json:"failure,omitempty"`
	Lost            string                  `json:"lost,omitempty"`
	Set             string                  `json:"set"`
	Profile         string                  `json:"profile"`
	Machine         string                  `json:"machine,omitempty"`
	Budget          string                  `json:"budget,omitempty"`
	Limits          *campaign.Limits        `json:"limits,omitempty"`
	Counters        campaign.Counters       `json:"counters"`
	Coverage        *campaign.Summary       `json:"coverage,omitempty"`
	Counterexamples []counterexample        `json:"counterexamples"`
	Targets         []campaign.TargetStatus `json:"targets"`
	Candidates      []candidate             `json:"candidates"`
}

// counterexample is one violated class member in the summary: the bridge's proposal by digest and
// path, or why none compiled, and the file the proposal was written to when `--promotion-root`
// named one. The source bytes are never in the summary: they are the file, and its digest is what
// two campaigns compare.
type counterexample struct {
	ClassName             string  `json:"className"`
	Target                string  `json:"target"`
	Candidate             string  `json:"candidate"`
	PromotionSourceSHA256 *string `json:"promotionSourceSha256"`
	PromotionSourcePath   string  `json:"promotionSourcePath,omitempty"`
	PromotionError        string  `json:"promotionError,omitempty"`
	Written               string  `json:"written,omitempty"`
}

type candidate struct {
	Candidate   string   `json:"candidate"`
	Target      string   `json:"target"`
	Kind        string   `json:"kind"`
	Observation string   `json:"observation,omitempty"`
	Detail      string   `json:"detail,omitempty"`
	Credited    []string `json:"credited"`
}

// Run is the whole command. A rejected command line, a campaign that cannot open, and a report
// over its cap all exit with one line on stderr naming the cause.
func Run(arguments []string, stdout, stderr io.Writer, open opener) int {
	configuration, err := parseConfig(arguments, stderr)
	if err != nil {
		return exitToolingError
	}
	ctx, cancel := cli.Interruptible(context.Background(), configuration.Timeout)
	defer cancel()

	opened, err := open(ctx, configuration, stderr)
	if err != nil {
		cli.WriteLine(stderr, "%s", err)
		return exitToolingError
	}
	defer func() {
		if err := opened.release(context.WithoutCancel(ctx)); err != nil {
			for _, failure := range cli.Flatten(err) {
				cli.WriteLine(stderr, "%s", failure)
			}
		}
	}()

	// The bridge's own stderr carries the progress line per candidate; the coordinator's would say
	// the same thing, so it goes nowhere.
	report, driveErr := campaign.Drive(ctx, opened.bridge, opened.binder, configuration.Caps, nil)
	report = settle(report, driveErr)
	if driveErr != nil {
		cli.WriteLine(stderr, "campaign %s: %s", report.Terminal.Status, driveErr)
	}
	written, err := writeProposals(configuration.PromotionRoot, report.Finished)
	if err != nil {
		// The campaign's findings stand; the command did not do what it was told with them.
		cli.WriteLine(stderr, "%s", err)
		report.Terminal.Status, report.Terminal.Failure = campaign.StatusToolingFailure, err.Error()
	}
	rendered, err := render(configuration, opened, report, written, false)
	if err != nil {
		cli.WriteLine(stderr, "render summary: %s", err)
		return exitToolingError
	}
	if err := configuration.Caps.CheckReport(len(rendered)); err != nil {
		// Over the cap is limit-reached, never a truncated report: the terminal, the counters, the
		// coverage counts and the counterexamples are written, and the ledger and the candidates,
		// which grow with the campaign, are not. A failure, a stop or a campaign cap keeps its
		// terminal, because what it names outranks this cap; only a campaign that ended exhausted
		// becomes limit-reached.
		cli.WriteLine(stderr, "%s", err)
		if report.Terminal.Status == campaign.StatusExhausted {
			report.Terminal = campaign.Terminal{Status: campaign.StatusLimitReached, Limit: "report-bytes"}
		}
		rendered, err = render(configuration, opened, report, written, true)
		if err != nil {
			cli.WriteLine(stderr, "render summary: %s", err)
			return exitToolingError
		}
		if err := configuration.Caps.CheckReport(len(rendered)); err != nil {
			cli.WriteLine(stderr, "the terminal-only summary is over the cap too and is written whole: %s", err)
		}
	}
	if _, err := stdout.Write(rendered); err != nil {
		cli.WriteLine(stderr, "write summary: %s", err)
		return exitToolingError
	}
	cli.WriteLine(stderr, "campaign %s", describeTerminal(report.Terminal))
	return exitCode(report)
}

// settle makes the report answer for itself: a terminal that is not one of the four, or a campaign
// that ended without the bridge's summary when it should have had one, is a tooling failure, because
// what such a report says cannot be trusted.
func settle(report campaign.Report, driveErr error) campaign.Report {
	switch report.Terminal.Status {
	case campaign.StatusExhausted, campaign.StatusLimitReached, campaign.StatusStopped, campaign.StatusToolingFailure:
	default:
		failure := "the coordinator ended without a terminal"
		if driveErr != nil {
			failure = driveErr.Error()
		}
		report.Terminal = campaign.Terminal{Status: campaign.StatusToolingFailure, Failure: failure}
		return report
	}
	if report.Finished == nil && report.Terminal.Status != campaign.StatusToolingFailure && report.Terminal.Status != campaign.StatusStopped {
		failure := "the bridge's summary could not be read"
		if driveErr != nil {
			failure = driveErr.Error()
		}
		report.Terminal = campaign.Terminal{Status: campaign.StatusToolingFailure, Failure: failure}
	}
	return report
}

func describeTerminal(terminal campaign.Terminal) string {
	switch terminal.Status {
	case campaign.StatusLimitReached:
		return fmt.Sprintf("%s (%s)", terminal.Status, terminal.Limit)
	case campaign.StatusToolingFailure:
		return fmt.Sprintf("%s: %s", terminal.Status, terminal.Failure)
	case campaign.StatusStopped:
		if terminal.Lost != "" {
			return fmt.Sprintf("%s (lost %s)", terminal.Status, terminal.Lost)
		}
	default:
	}
	return string(terminal.Status)
}

// exitCode maps the terminal and the findings to the exit code. A violation is read from the
// bridge's summary and from the candidates observed, so a stop that lost the summary still exits 1
// when an earlier Run was violated.
func exitCode(report campaign.Report) int {
	if report.Terminal.Status == campaign.StatusToolingFailure {
		return exitToolingError
	}
	violated := report.Finished != nil && (len(report.Finished.Counterexamples) > 0 || report.Finished.Summary.Violated > 0)
	for _, outcome := range report.Outcomes {
		if outcome.Observation == "violated" {
			violated = true
		}
	}
	if violated {
		return exitViolated
	}
	if report.Terminal.Status == campaign.StatusLimitReached || report.Terminal.Status == campaign.StatusStopped {
		return exitLimitOrStop
	}
	return exitExhausted
}

// writeProposals writes each compiled proposal under the promotion root, at the path the bridge
// named, and returns the written path by candidate. No root, no writing: the digests alone are
// reported. Every path is checked before any file is written, so a path that would leave the
// root writes nothing at all; a write that then fails returns what was written before it.
func writeProposals(root string, finished *campaign.Finished) (map[string]string, error) {
	if root == "" || finished == nil {
		return nil, nil
	}
	paths := map[string]string{}
	for _, sample := range finished.Counterexamples {
		if sample.PromotionSourceSHA256 == nil || sample.PromotionSourcePath == "" {
			continue
		}
		relative := filepath.Clean(filepath.FromSlash(sample.PromotionSourcePath))
		if filepath.IsAbs(relative) || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("proposal for %s names a path outside the promotion root: %q", sample.Candidate, sample.PromotionSourcePath)
		}
		paths[sample.Candidate] = filepath.Join(root, relative)
	}
	written := map[string]string{}
	for _, sample := range finished.Counterexamples {
		path, ok := paths[sample.Candidate]
		if !ok {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return written, fmt.Errorf("write proposal for %s: %w", sample.Candidate, err)
		}
		if err := os.WriteFile(path, []byte(sample.PromotionSource), 0o644); err != nil {
			return written, fmt.Errorf("write proposal for %s: %w", sample.Candidate, err)
		}
		written[sample.Candidate] = path
	}
	return written, nil
}

// within reports whether path is root or under it; both are absolute and clean.
func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

// render is the canonical summary: one JSON document, keys in declaration order, one LF. The
// terminal-only form leaves out what grows with the campaign (the ledger and the candidates) and
// keeps the coverage counts, which are fixed in size, and the counterexamples, which are bounded
// by the class targets; together they are what exit 1 names.
func render(configuration config, opened *bound, report campaign.Report, written map[string]string, terminalOnly bool) ([]byte, error) {
	rendered := summary{
		Status:          string(report.Terminal.Status),
		Limit:           report.Terminal.Limit,
		Failure:         report.Terminal.Failure,
		Lost:            report.Terminal.Lost,
		Set:             configuration.Set,
		Profile:         opened.profile,
		Machine:         opened.opened.Machine,
		Budget:          opened.opened.Budget,
		Counterexamples: []counterexample{},
		Targets:         []campaign.TargetStatus{},
		Candidates:      []candidate{},
		Counters:        report.Counters,
	}
	if opened.opened.Budget != "" {
		limits := opened.opened.Limits
		rendered.Limits = &limits
	}
	if report.Finished != nil {
		coverage := report.Finished.Summary
		rendered.Coverage = &coverage
		for _, sample := range report.Finished.Counterexamples {
			rendered.Counterexamples = append(rendered.Counterexamples, counterexample{
				ClassName: sample.ClassName, Target: sample.Target, Candidate: sample.Candidate,
				PromotionSourceSHA256: sample.PromotionSourceSHA256, PromotionSourcePath: sample.PromotionSourcePath,
				PromotionError: sample.PromotionError, Written: written[sample.Candidate],
			})
		}
		if !terminalOnly && report.Finished.Ledger != nil {
			rendered.Targets = report.Finished.Ledger
		}
	}
	if terminalOnly {
		report.Outcomes = nil
	}
	for _, outcome := range report.Outcomes {
		credited := outcome.Credited
		if credited == nil {
			credited = []string{}
		}
		rendered.Candidates = append(rendered.Candidates, candidate{
			Candidate: outcome.Identity, Target: outcome.Target, Kind: string(outcome.Kind),
			Observation: outcome.Observation, Detail: outcome.Detail, Credited: credited,
		})
	}
	encoded, err := json.Marshal(rendered)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	if len(arguments) == 0 || arguments[0] != "run" {
		cli.WriteLine(stderr, "usage: umpire-fuzz run --set <set> --grpc <address> --http <address> --namespace <namespace> --task-queue <queue> [flags]")
		return config{}, errors.New("the subcommand is run")
	}
	var configuration config
	flags := flag.NewFlagSet("umpire-fuzz run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.Set, "set", "", "the exploratory set to explore")
	binding.RegisterFlags(flags, &configuration.Deployment, "the Cases")
	flags.StringVar(&configuration.ModelRoot, "model-root", defaultModelRoot, "the model package the bridge runs in")
	flags.StringVar(&configuration.PromotionRoot, "promotion-root", "", "a directory outside the model to write each counterexample's promotion source under; none writes nothing")
	flags.StringVar(&configuration.Bridge, "bridge", "", "the exploration bridge executable (default <model-root>/"+bridgeRelativePath+")")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "bound on the whole campaign")
	flags.DurationVar(&configuration.Caps.RunTimeout, "run-timeout", defaultRunTimeout, "bound on one Run")
	flags.IntVar(&configuration.Caps.Candidates, "max-candidates", 0, "cap on candidates planned (0: no cap)")
	flags.Int64Var(&configuration.Caps.CaseBytes, "max-case-bytes", 0, "cap on the aggregate Case bytes (0: no cap)")
	flags.Int64Var(&configuration.Caps.RunEvents, "max-run-events", 0, "cap on the aggregate Run Events (0: no cap)")
	flags.Int64Var(&configuration.Caps.ReportBytes, "max-report-bytes", 0, "cap on the summary's bytes (0: no cap)")
	if err := flags.Parse(arguments[1:]); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		cli.WriteLine(stderr, "umpire-fuzz run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	if configuration.Set == "" {
		cli.WriteLine(stderr, "--set is required")
		return config{}, errors.New("missing --set")
	}
	if missing := binding.Missing(configuration.Deployment); len(missing) > 0 {
		cli.WriteLine(stderr, "%s is required", missing[0])
		return config{}, fmt.Errorf("missing %s", missing[0])
	}
	if configuration.Timeout <= 0 || configuration.Caps.RunTimeout <= 0 {
		cli.WriteLine(stderr, "--timeout and --run-timeout must be positive")
		return config{}, errors.New("non-positive timeout")
	}
	for _, cap := range []struct {
		name  string
		value int64
	}{
		{"--max-candidates", int64(configuration.Caps.Candidates)},
		{"--max-case-bytes", configuration.Caps.CaseBytes},
		{"--max-run-events", configuration.Caps.RunEvents},
		{"--max-report-bytes", configuration.Caps.ReportBytes},
	} {
		if cap.value < 0 {
			cli.WriteLine(stderr, "%s must not be negative", cap.name)
			return config{}, errors.New("negative cap")
		}
	}
	if configuration.Caps.ReportBytes > 0 && configuration.Caps.ReportBytes < minimumReportBytes {
		cli.WriteLine(stderr, "--max-report-bytes must be 0 or at least %d, the terminal-only summary's floor", minimumReportBytes)
		return config{}, errors.New("report cap below the floor")
	}
	// The bridge runs in the model root, and a relative executable would be looked up there rather
	// than where the command was invoked, so both paths are made absolute first.
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
		if configuration.PromotionRoot, err = filepath.Abs(configuration.PromotionRoot); err != nil {
			cli.WriteLine(stderr, "--promotion-root: %s", err)
			return config{}, err
		}
		// A proposal is for review, never an installed regression: the model never receives one.
		if within(modelRoot, configuration.PromotionRoot) {
			cli.WriteLine(stderr, "--promotion-root must not be under the model root %s", modelRoot)
			return config{}, errors.New("promotion root under the model")
		}
	}
	return configuration, nil
}

// openCampaign is the real opening: bind the deployment once, spawn the bridge in the model
// package, and initialize it over the set under the Profile identity every candidate is bound
// under. A campaign that fails to open releases what it opened.
func openCampaign(ctx context.Context, configuration config, stderr io.Writer) (*bound, error) {
	deployment := configuration.Deployment
	handlerQueue := binding.HandlerQueue(deployment)
	campaignBinding, err := binding.Open(ctx, deployment, handlerQueue)
	if err != nil {
		return nil, err
	}
	releases := []func(context.Context) error{campaignBinding.Close}
	fail := func(err error) (*bound, error) {
		return nil, errors.Join(err, binding.ReleaseAll(context.WithoutCancel(ctx), releases))
	}
	// The bridge outlives the campaign context: a stop or the timeout ends the campaign, and the
	// bridge's summary is then asked for and read before release closes the bridge.
	bridge, err := campaign.Start(context.WithoutCancel(ctx), campaign.Options{
		Executable: configuration.Bridge, Dir: configuration.ModelRoot, Stderr: stderr,
	})
	if err != nil {
		return fail(err)
	}
	releases = append(releases, func(context.Context) error { return bridge.Close() })
	profile := "umpire-fuzz." + deployment.Namespace
	opened, err := bridge.Initialize(ctx, configuration.Set, profile)
	if err != nil {
		return fail(fmt.Errorf("initialize the campaign over %s: %w", configuration.Set, err))
	}
	return &bound{
		bridge:  bridge,
		binder:  campaign.CampaignBinder{Campaign: campaignBinding},
		profile: profile,
		opened:  opened,
		release: func(ctx context.Context) error { return binding.ReleaseAll(ctx, releases) },
	}, nil
}
