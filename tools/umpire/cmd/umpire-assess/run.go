package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// Exit codes: the decision, or 3 for everything that is not one.
const (
	exitAccepted   = 0
	exitRejected   = 1
	exitIncomplete = 2
	exitFailed     = 3
)

// assessTimeout bounds publication, the one step an interrupt or a deadline can matter to: reading,
// admission, assessment and rendering are pure and quick, and run before the signal is captured, so
// an interrupt there simply ends the process with nothing published.
const assessTimeout = time.Minute

const defaultModelRoot = "model"

// The summary statuses beyond the three decisions, each its own named failure.
const (
	statusRejectedSubject       = "rejected-subject"
	statusUnknownProfile        = "unknown-profile"
	statusProfileUnreadable     = "profile-unreadable"
	statusAdmissionFailed       = "admission-failed"
	statusUnreadableInput       = "unreadable-input"
	statusCatalogUnavailable    = "catalog-unavailable"
	statusReceiptOversized      = "receipt-oversized"
	statusReceiptUnreadable     = "receipt-unreadable"
	statusPublicationConflict   = "publication-conflict"
	statusPublicationFailed     = "publication-failed"
	statusInterrupted           = "interrupted"
	statusPublicationUnreported = "publication-unreported"
)

// config is what the caller names: the subject's files, the Profile's exact name and where the
// receipt goes. Nothing names a Driver, a deployment, a checker, a policy or a retry.
type config struct {
	Case        string
	Run         string
	Profile     *evaluation.Profile
	ReceiptRoot string
}

// The self-check and the publisher; a test replaces them to reach the failures no real subject or
// root produces.
var (
	decodeReceipt = evaluation.DecodeReceipt
	publish       = cli.Publish
)

// environment is what the command reads beyond its arguments: the tree's catalog fingerprint, and
// the context an assessment runs under. A test supplies its own.
type environment struct {
	Catalog func() (string, error)
	Context func() (context.Context, context.CancelFunc)
}

// summary is the one JSON document on stdout, in a fixed key order.
type summary struct {
	// Status is the decision, or the named failure.
	Status      string   `json:"status"`
	Reasons     []string `json:"reasons,omitempty"`
	Rejection   string   `json:"rejection,omitempty"`
	Receipt     string   `json:"receipt,omitempty"`
	Publication string   `json:"publication,omitempty"`
	Path        string   `json:"path,omitempty"`
	Detail      string   `json:"detail,omitempty"`
}

// Run is the whole command: refuse the command line before reading anything, then admit, assess,
// render, check the rendering reads back, publish once and report.
func Run(arguments []string, stdout, stderr io.Writer, env environment) int {
	failed := func(status, format string, arguments ...any) int {
		return report(stdout, stderr, summary{Status: status, Detail: fmt.Sprintf(format, arguments...)}, exitFailed)
	}
	configuration, err := parseConfig(arguments, stderr)
	var unknown *unknownProfileError
	if errors.As(err, &unknown) {
		if errors.Is(unknown.err, evaluation.ErrUnknownProfile) {
			return failed(statusUnknownProfile, "%s", unknown.err)
		}
		return failed(statusProfileUnreadable, "%s", unknown.err)
	}
	if err != nil {
		return exitFailed
	}
	caseBytes, err := readCapped(configuration.Case, evaluation.MaxCaseBytes)
	if err != nil {
		return failed(statusUnreadableInput, "--case: %s", err)
	}
	runBytes, err := readCapped(configuration.Run, evaluation.MaxRunBytes)
	if err != nil {
		return failed(statusUnreadableInput, "--run: %s", err)
	}
	catalog, err := env.Catalog()
	if err != nil {
		return failed(statusCatalogUnavailable, "build the method catalog: %s", err)
	}
	subject, err := evaluation.Admit(caseBytes, runBytes, catalog)
	if err != nil {
		rejection, ok := evaluation.IsRejection(err)
		if !ok {
			return failed(statusAdmissionFailed, "%s", err)
		}
		return report(stdout, stderr, summary{Status: statusRejectedSubject, Rejection: rejection.Reason, Detail: rejection.Detail}, exitFailed)
	}
	decision := evaluation.Assess(subject, *configuration.Profile)
	rendered, err := evaluation.Render(subject, *configuration.Profile, decision)
	var oversized *evaluation.ReceiptOversizedError
	if errors.As(err, &oversized) {
		return failed(statusReceiptOversized, "%s", err)
	}
	if err != nil {
		return failed(statusReceiptUnreadable, "render the receipt: %s", err)
	}
	if _, err := decodeReceipt(rendered); err != nil {
		return failed(statusReceiptUnreadable, "the rendered receipt does not read back: %s", err)
	}

	identity := evaluation.ReceiptIdentity(rendered)
	ctx, cancel := env.context()
	defer cancel()
	publication, err := publish(ctx, configuration.ReceiptRoot, identity+".json", rendered)
	var conflict *cli.ConflictError
	switch {
	case errors.As(err, &conflict):
		return report(stdout, stderr, summary{Status: statusPublicationConflict, Receipt: identity, Path: conflict.Path, Detail: conflict.Detail}, exitFailed)
	case err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)):
		return failed(statusInterrupted, "%s", err)
	case err != nil:
		return failed(statusPublicationFailed, "%s", err)
	}

	var reasons []string
	for _, reason := range decision.Reasons {
		reasons = append(reasons, reason.Name)
	}
	return report(stdout, stderr, summary{
		Status: decision.Outcome, Reasons: reasons, Receipt: identity,
		Publication: publication.Status, Path: publication.Path,
	}, exitCode(decision.Outcome))
}

func (env environment) context() (context.Context, context.CancelFunc) {
	if env.Context != nil {
		return env.Context()
	}
	return cli.Interruptible(context.Background(), assessTimeout)
}

func exitCode(outcome string) int {
	switch outcome {
	case evaluation.DecisionAccepted:
		return exitAccepted
	case evaluation.DecisionRejected:
		return exitRejected
	case evaluation.DecisionIncomplete:
		return exitIncomplete
	default:
		return exitFailed
	}
}

// report writes the summary to stdout and one line to stderr. A summary stdout cannot take goes to
// stderr instead; after a publication that is the one ambiguity the command names, and it is never
// retried.
func report(stdout, stderr io.Writer, result summary, code int) int {
	encoded, err := json.Marshal(result)
	if err != nil {
		cli.WriteLine(stderr, "umpire-assess: encode the summary: %s", err)
		return exitFailed
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		if result.Publication != "" {
			result = summary{Status: statusPublicationUnreported, Receipt: result.Receipt, Publication: result.Publication, Path: result.Path,
				Detail: fmt.Sprintf("the receipt is %s but its summary could not be written: %s", result.Publication, err)}
			encoded, _ = json.Marshal(result)
		}
		cli.WriteLine(stderr, "%s", encoded)
		return exitFailed
	}
	switch {
	case result.Publication != "":
		cli.WriteLine(stderr, "assess %s, receipt %s %s", result.Status, result.Receipt, result.Publication)
	case result.Rejection != "":
		cli.WriteLine(stderr, "assess %s (%s): %s", result.Status, result.Rejection, result.Detail)
	default:
		cli.WriteLine(stderr, "assess %s: %s", result.Status, result.Detail)
	}
	return code
}

// readCapped reads at most one byte past limit, so an oversized input is refused by admission
// without ever being held whole.
func readCapped(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(io.LimitReader(file, int64(limit)+1))
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	usage := "usage: umpire-assess run --case <case.json> --run <recorded-run.json> --profile <name> --receipt-root <dir> [--model-root <dir>]"
	if len(arguments) == 0 || arguments[0] != "run" {
		cli.WriteLine(stderr, "%s", usage)
		return config{}, errors.New("the subcommand is run")
	}
	var configuration config
	var profile, receiptRoot, modelRoot string
	flags := flag.NewFlagSet("umpire-assess run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.Case, "case", "", "the subject's Case, canonical or its persisted form")
	flags.StringVar(&configuration.Run, "run", "", "the Run recorded against the Case")
	flags.StringVar(&profile, "profile", "", "the exact name of an Evaluation Profile the model declares")
	flags.StringVar(&receiptRoot, "receipt-root", "", "an existing directory outside the model to publish the receipt under")
	flags.StringVar(&modelRoot, "model-root", defaultModelRoot, "the model package, which never receives a receipt")
	if err := flags.Parse(arguments[1:]); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		cli.WriteLine(stderr, "umpire-assess run accepts no positional arguments")
		return config{}, errors.New("unexpected positional arguments")
	}
	for _, required := range []struct{ flag, value string }{
		{"--case", configuration.Case}, {"--run", configuration.Run}, {"--profile", profile}, {"--receipt-root", receiptRoot},
	} {
		if required.value == "" {
			cli.WriteLine(stderr, "%s is required", required.flag)
			return config{}, fmt.Errorf("missing %s", required.flag)
		}
	}
	loaded, err := evaluation.LoadProfile(profile)
	if err != nil {
		return config{}, &unknownProfileError{err: err}
	}
	configuration.Profile = loaded
	// A receipt is an assessment's output, never model input: the model never receives one,
	// whatever symlink the root is reached through.
	if configuration.ReceiptRoot, err = cli.OutsideModel("--receipt-root", receiptRoot, modelRoot); err != nil {
		cli.WriteLine(stderr, "%s", err)
		return config{}, err
	}
	if info, err := os.Stat(configuration.ReceiptRoot); err != nil || !info.IsDir() {
		cli.WriteLine(stderr, "--receipt-root %s: the directory does not exist", configuration.ReceiptRoot)
		return config{}, errors.New("receipt root missing")
	}
	return configuration, nil
}

// unknownProfileError says --profile names no loadable Profile: refused before anything is read,
// and reported as an unknown name or as an embedded Profile that does not load.
type unknownProfileError struct{ err error }

func (e *unknownProfileError) Error() string { return "--profile: " + e.err.Error() }
