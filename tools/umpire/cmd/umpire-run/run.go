package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/common/testing/testpilot/temporal/binding"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/internal/cli"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// Exit codes. 3 is deliberately separate from 2 so a caller can tell an unreachable server or a
// Case that could not be prepared from a Run that really was inconclusive. With a Model assessment
// the exit code is the worse of the Verdict and the assessment (assessedExitCode).
const (
	exitSatisfied    = 0
	exitViolated     = 1
	exitInconclusive = 2
	exitFailed       = 3
)

const defaultTimeout = 5 * time.Minute

// config is what the caller names. Nothing here ever enters the Case: addresses and credentials
// stay outside the bytes, and no flag widens a declared Limit.
type config struct {
	CasePath string
	// Deployment is what the Case binds to; its handler task queue, when empty, derives
	// `<task-queue>-handler` for a Case that binds one.
	Deployment binding.Deployment
	// RecordPath, when named, receives the closed Run with the identity it was prepared under,
	// the recorded Run a replay reads; empty records nothing.
	RecordPath string
	// ModelRoot, when named, is the model directory (its `cases/` and `ir/`) whose Model assesses
	// the Run beside its Contract; empty assesses nothing.
	ModelRoot string
	Timeout   time.Duration
}

// session is one bound Case: how to run it once, and how to release everything the binding opened.
type session struct {
	// run executes the prepared Case against the Driver the binding opened.
	run func(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error)
	// assessed is run with a Model assessment beside the Contract.
	assessed func(ctx context.Context, factory testpilot.AssessmentFactory) (*testpilotspb.Run, *testpilotspb.Verdict, *testpilot.Assessment, error)
	// identity is the Profile identity the Case was prepared under, recorded beside its Run.
	identity testpilot.DriverIdentity
	// release is best-effort teardown, run whatever the Run did. It names every resource it could
	// not remove.
	release func(ctx context.Context) error
}

// opener binds one Case. The real one dials the server; a test supplies its own.
type opener func(ctx context.Context, configuration config, source *testpilotspb.Case) (*session, error)

// Run is the whole command. It never panics on a caller mistake: a rejected flag, an unreadable
// fixture, a Case whose Profile cannot be derived, and an unreachable server all exit 3 with one
// line on stderr naming the cause.
func Run(arguments []string, stdout, stderr io.Writer, open opener) int {
	configuration, err := parseConfig(arguments, stderr)
	if err != nil {
		return exitFailed
	}

	encoded, source, err := readCase(configuration.CasePath)
	if err != nil {
		cli.WriteLine(stderr, "%s", err)
		return exitFailed
	}
	if configuration.RecordPath != "" {
		// The record names the Case by its canonical bytes' identity, which a fixture in no
		// canonical form does not have; refusing it now keeps a Run from happening that its record
		// would then lose.
		if _, err := recordedrun.CaseIdentity(encoded); err != nil {
			cli.WriteLine(stderr, "--record: Case fixture %q is not in a canonical form: %v", configuration.CasePath, err)
			return exitFailed
		}
	}

	prepared, err := binding.Prepare(configuration.Deployment, binding.HandlerQueueFor(configuration.Deployment, source.GetProgram()), "umpire-run."+configuration.Deployment.Namespace, source)
	if err != nil {
		var rejection *testpilot.PreparationError
		if errors.As(err, &rejection) && rejection.Category == testpilot.PreparationUnsupported {
			cli.WriteLine(stderr, "skipped: %s", describeFailure(err))
		} else {
			cli.WriteLine(stderr, "%s", describeFailure(err))
		}
		return exitFailed
	}
	var assessment *modelAssessment
	if configuration.ModelRoot != "" {
		// The Model, the Query and the assessment's binding to this Case are settled before anything
		// is opened, so a Case the Model does not assess creates no Run.
		if assessment, err = prepareAssessment(configuration.ModelRoot, source); err == nil {
			_, err = prepared.Case.WithAssessment(assessment.factory)
		}
		if err != nil {
			cli.WriteLine(stderr, "--model: %s", describeFailure(err))
			return exitFailed
		}
	}

	ctx, cancel := cli.Interruptible(context.Background(), configuration.Timeout)
	defer cancel()

	bound, err := open(ctx, configuration, source)
	if err != nil {
		cli.WriteLine(stderr, "%s", describeFailure(err))
		return exitFailed
	}
	defer releaseSession(bound, stderr)

	var (
		run      *testpilotspb.Run
		verdict  *testpilotspb.Verdict
		assessed *testpilot.Assessment
	)
	switch {
	case assessment == nil:
		run, verdict, err = bound.run(ctx)
	case bound.assessed == nil:
		err = errors.New("the binding cannot run a Model assessment")
	default:
		run, verdict, assessed, err = bound.assessed(ctx, assessment.factory)
	}
	if err != nil {
		cli.WriteLine(stderr, "run Case %q: %v", source.GetCaseId(), err)
		return exitFailed
	}

	report(stdout, run, verdict)
	if assessment != nil {
		reportAssessment(stdout, assessed, assessment.expected.Check(run, verdict, assessed))
	}
	if configuration.RecordPath != "" {
		// The Run is reported and its Verdict decides the exit code whatever happens to its
		// record; a record that could not be written after all is said on stderr, since the path
		// was checked before anything ran and only a race or the disk can fail it now.
		if err := recordedrun.Write(configuration.RecordPath, encoded, bound.identity, run); err != nil {
			cli.WriteLine(stderr, "%s", err)
		}
	}
	if assessment != nil {
		return assessedExitCode(verdict, assessed)
	}
	return exitCode(verdict)
}

// modelAssessment is the Model assessment of one generated Case: its factory, and the Run its Query
// expects.
type modelAssessment struct {
	factory  testpilot.AssessmentFactory
	expected *lower.ExpectedRun
}

// prepareAssessment finds the Case among the lowered Cases of the model directory, by the identity
// an assessment is bound to, and prepares its Query's assessment under the shared ceilings, as the
// live tests do.
func prepareAssessment(root string, source *testpilotspb.Case) (*modelAssessment, error) {
	entry, err := lower.FindGeneratedCase(filepath.Join(root, "cases"), source)
	if err != nil {
		return nil, err
	}
	model, err := umpiremodel.Load(filepath.Join(root, "ir", entry.Model))
	if err != nil {
		return nil, err
	}
	factory, err := conformance.Prepare(model, entry.Query, source, conformance.DefaultLimits())
	if err != nil {
		return nil, err
	}
	return &modelAssessment{factory: factory, expected: entry.Expected}, nil
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	var configuration config
	flags := flag.NewFlagSet("umpire-run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.CasePath, "case", "", "path to a checked-in Case fixture")
	binding.RegisterFlags(flags, &configuration.Deployment, "the Case")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "bound on the whole Run")
	flags.StringVar(&configuration.RecordPath, "record", "", "write the closed Run with the identity it was prepared under to this file, which must not exist yet in a directory that does")
	flags.StringVar(&configuration.ModelRoot, "model", "", "assess the Run against the Model of the generated Case in this model directory (its cases/ and ir/), beside the Contract")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		cli.WriteLine(stderr, "umpire-run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	if configuration.CasePath == "" {
		cli.WriteLine(stderr, "--case is required")
		return config{}, errors.New("missing --case")
	}
	if missing := binding.Missing(configuration.Deployment); len(missing) > 0 {
		cli.WriteLine(stderr, "%s is required", missing[0])
		return config{}, fmt.Errorf("missing %s", missing[0])
	}
	if configuration.Timeout <= 0 {
		cli.WriteLine(stderr, "--timeout must be positive")
		return config{}, errors.New("non-positive timeout")
	}
	if configuration.RecordPath != "" {
		// A record that could not be written would lose the Run it records, so the path is refused
		// before anything runs: it must not exist, and its directory must.
		path, err := filepath.Abs(configuration.RecordPath)
		if err != nil {
			cli.WriteLine(stderr, "--record: %s", err)
			return config{}, err
		}
		if _, err := os.Stat(path); err == nil {
			cli.WriteLine(stderr, "--record %s exists and is never replaced", path)
			return config{}, errors.New("record exists")
		}
		if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
			cli.WriteLine(stderr, "--record %s: the directory does not exist", path)
			return config{}, errors.New("record directory missing")
		}
		configuration.RecordPath = path
	}
	return configuration, nil
}

func readCase(path string) ([]byte, *testpilotspb.Case, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read Case fixture: %w", err)
	}
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	if err != nil {
		return nil, nil, fmt.Errorf("decode Case fixture %q: %w", path, err)
	}
	return encoded, source, nil
}

// describeFailure names a static admission rejection by its category, because "typed fixture" and
// "unreachable server" are different problems and the exit code is the same.
func describeFailure(err error) string {
	var rejection *testpilot.PreparationError
	if errors.As(err, &rejection) {
		return fmt.Sprintf("prepare Case: %s at %s: %s", rejection.Category, rejection.Path, rejection.Detail)
	}
	return err.Error()
}

func releaseSession(bound *session, stderr io.Writer) {
	if bound == nil || bound.release == nil {
		return
	}
	// Teardown runs on its own context: an interrupted or timed-out Run still releases what it
	// created.
	if err := bound.release(context.Background()); err != nil {
		for _, failure := range cli.Flatten(err) {
			cli.WriteLine(stderr, "%s", failure)
		}
	}
}

// flatten reports one line per leaked resource rather than one joined blob.
func report(stdout io.Writer, run *testpilotspb.Run, verdict *testpilotspb.Verdict) {
	cli.WriteLine(stdout, "run %s", run.GetDisposition())
	cli.WriteLine(stdout, "cleanup %s", run.GetCleanup().GetStatus())
	cli.WriteLine(stdout, "verdict %s", verdict.GetStatus())
	for _, rule := range verdict.GetRules() {
		cli.WriteLine(stdout, "rule %s %s %s", rule.GetRuleId(), rule.GetStatus(), rule.GetTerminalStateId())
	}
	// A diagnostic's kind and code, never its detail: the detail names ids a caller comparing two
	// Runs' reports would only have to strip again.
	for _, diagnostic := range run.GetDiagnostics() {
		cli.WriteLine(stdout, "diagnostic %s %s", diagnostic.GetKind(), diagnostic.GetCode())
	}
}

// reportAssessment prints the Model assessment after the Run's lines: the conformance and each
// property by status and reason id, never its prose, the failure when it failed, and whether the Run
// is the one its Query expects, as lower.ExpectedRun.Check words each difference on a line of its
// own.
func reportAssessment(stdout io.Writer, assessment *testpilot.Assessment, expected error) {
	cli.WriteLine(stdout, "%s", strings.TrimSpace("conformance "+string(assessment.Conformance.Status)+" "+assessment.Conformance.Reason))
	for _, property := range assessment.Properties {
		cli.WriteLine(stdout, "%s", strings.TrimSpace("property "+property.ID+" "+string(property.Status)+" "+property.Reason))
	}
	if failure := assessment.Failure; failure != nil {
		cli.WriteLine(stdout, "assessment failed %s at event %d", failure.Code, failure.EventSequence)
	}
	if expected == nil {
		cli.WriteLine(stdout, "expected match")
		return
	}
	for _, difference := range cli.Flatten(expected) {
		cli.WriteLine(stdout, "expected differs: %s", difference)
	}
}

// assessedExitCode is the worse of the Verdict and the Model assessment: 1 when either found a
// violation (a violated Verdict, a nonconformant Run or a violated property, one the assessment
// established before failing included), else 3 when the assessment failed, else 2 when anything is
// inconclusive, else 0.
func assessedExitCode(verdict *testpilotspb.Verdict, assessment *testpilot.Assessment) int {
	conformance := assessment.Conformance.Status
	violated := verdict.GetStatus() == testpilotspb.VERDICT_STATUS_VIOLATED || conformance == testpilot.ConformanceNonconformant
	open := verdict.GetStatus() != testpilotspb.VERDICT_STATUS_SATISFIED || conformance != testpilot.ConformanceConformant
	for _, property := range assessment.Properties {
		violated = violated || property.Status == testpilot.PropertyViolated
		open = open || property.Status != testpilot.PropertySatisfied
	}
	switch {
	case violated:
		return exitViolated
	case assessment.Failure != nil:
		return exitFailed
	case open:
		return exitInconclusive
	default:
		return exitSatisfied
	}
}

func exitCode(verdict *testpilotspb.Verdict) int {
	switch verdict.GetStatus() {
	case testpilotspb.VERDICT_STATUS_SATISFIED:
		return exitSatisfied
	case testpilotspb.VERDICT_STATUS_VIOLATED:
		return exitViolated
	default:
		return exitInconclusive
	}
}

// openSession is the real binding: the campaign-scoped part (dial the frontend, create the named
// resources when asked, build the catalog) opened for this one Case, then the candidate-scoped part
// (derive the Profile the Case implies, prepare its unchanged bytes, open one composite Driver with
// its own SDK worker). Both live in `common/testing/testpilot/temporal/binding`, which a campaign shares; the CLI's
// behavior and exit codes are unchanged.
func openSession(ctx context.Context, configuration config, source *testpilotspb.Case) (*session, error) {
	deployment := configuration.Deployment
	campaign, err := binding.Open(ctx, deployment, binding.HandlerQueueFor(deployment, source.GetProgram()))
	if err != nil {
		return nil, err
	}
	bound, err := campaign.Bind(ctx, "umpire-run."+deployment.Namespace, source)
	if err != nil {
		// A binding that failed rolls back on a context the failure cannot have cancelled.
		return nil, errors.Join(err, campaign.Close(context.WithoutCancel(ctx)))
	}
	return &session{
		run:      bound.Run,
		assessed: bound.RunAssessed,
		identity: bound.Identity(),
		release: func(ctx context.Context) error {
			return errors.Join(bound.Release(ctx), campaign.Close(ctx))
		},
	}, nil
}
