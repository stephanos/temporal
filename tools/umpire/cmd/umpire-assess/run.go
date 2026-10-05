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
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/common/testing/testpilot/publish"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/internal/cli"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
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
	statusInternalError         = "internal-error"
	statusUnreadableInput       = "unreadable-input"
	statusCatalogUnavailable    = "catalog-unavailable"
	statusReceiptOversized      = "receipt-oversized"
	statusReceiptUnreadable     = "receipt-unreadable"
	statusPublicationConflict   = "publication-conflict"
	statusPublicationFailed     = "publication-failed"
	statusInterrupted           = "interrupted"
	statusPublicationUnreported = "publication-unreported"
	// With --model: the model directory does not assess the subject's Case, or cannot be read.
	statusModelUnassessable = "model-unassessable"
	// With --model: the Case does not prepare offline, or its recorded Run does not evaluate to the
	// recorded Verdict, so no assessment of it can be trusted to be of that Run.
	statusAssessmentUnreproducible = "assessment-unreproducible"
)

// config is what the caller names: the subject's files, the Profile's exact name and where the
// receipt goes. Nothing names a Driver, a deployment, a checker, a policy or a retry.
type config struct {
	Case        string
	Run         string
	Profile     *evaluation.Profile
	ReceiptRoot string
	// Model, when named, is the model directory (its `cases/` and `ir/`) whose Model assesses the
	// recorded Run beside its Verdict; empty assesses nothing.
	Model string
}

// environment is what the command reads beyond its arguments: the tree's catalog fingerprint, the
// context publication runs under, and the receipt's self-check and publisher. A test supplies its
// own, to reach failures no real subject or root produces; a nil field is the real one.
type environment struct {
	Catalog func() (string, error)
	Context func() (context.Context, context.CancelFunc)
	Decode  func([]byte) (*evaluation.Receipt, error)
	Publish func(ctx context.Context, root, name string, contents []byte) (publish.Publication, error)
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
	var unknown *profileError
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
			return failed(statusInternalError, "admission returned no rejection: %s", err)
		}
		return report(stdout, stderr, summary{Status: statusRejectedSubject, Rejection: rejection.Reason, Detail: rejection.Detail}, exitFailed)
	}
	var assessment *testpilot.Assessment
	if configuration.Model != "" {
		var status string
		if assessment, status, err = assessRecorded(configuration.Model, caseBytes, runBytes); err != nil {
			return failed(status, "--model: %s", err)
		}
	}
	decision := evaluation.Assess(subject, *configuration.Profile, assessment)
	rendered, err := evaluation.Render(subject, *configuration.Profile, decision)
	var oversized *evaluation.ReceiptOversizedError
	if errors.As(err, &oversized) {
		return failed(statusReceiptOversized, "%s", err)
	}
	if err != nil {
		return failed(statusInternalError, "render the receipt: %s", err)
	}
	if _, err := env.decode(rendered); err != nil {
		return failed(statusReceiptUnreadable, "the rendered receipt does not read back: %s", err)
	}

	identity := evaluation.ReceiptIdentity(rendered)
	ctx, cancel := env.context()
	defer cancel()
	publication, err := env.publish(ctx, configuration.ReceiptRoot, identity+".json", rendered)
	var conflict *publish.ConflictError
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
		reasons = append(reasons, string(reason))
	}
	return report(stdout, stderr, summary{
		Status: decision.Outcome, Reasons: reasons, Receipt: identity,
		Publication: publication.Status, Path: publication.Path,
	}, exitCode(decision.Outcome))
}

// assessRecorded is the Model's assessment of an admitted recorded Run: the Case is found among the
// model directory's lowered Cases by the identity an assessment is bound to, prepared offline, and
// its recorded events are driven through its Query's assessment under the shared ceilings, as the
// live test that ran it replays them. The offline reading must give the recorded Verdict. A failure
// says which status it is.
func assessRecorded(root string, caseBytes, runBytes []byte) (*testpilot.Assessment, string, error) {
	canonical, err := casefile.Canonical(caseBytes)
	if err != nil {
		return nil, statusInternalError, err
	}
	source, err := testpilot.DecodeCaseProtoJSON(canonical)
	if err != nil {
		return nil, statusInternalError, err
	}
	decoded, err := recordedrun.Decode(runBytes)
	if err != nil {
		return nil, statusInternalError, err
	}
	entry, err := lower.FindGeneratedCase(filepath.Join(root, "cases"), source)
	if err != nil {
		return nil, statusModelUnassessable, err
	}
	model, err := umpiremodel.Load(filepath.Join(root, "ir", entry.Model))
	if err != nil {
		return nil, statusModelUnassessable, err
	}
	factory, err := conformance.Prepare(model, entry.Query, source, conformance.DefaultLimits())
	if err != nil {
		return nil, statusModelUnassessable, err
	}
	prepared, err := prepareOffline(source, decoded.Driver.Profile)
	if err != nil {
		return nil, statusAssessmentUnreproducible, err
	}
	assessed, err := prepared.WithAssessment(factory)
	if err != nil {
		return nil, statusModelUnassessable, err
	}
	verdict, evaluated, err := assessed.Evaluate(context.Background(), decoded.Run, nil)
	if err != nil {
		return nil, statusAssessmentUnreproducible, err
	}
	if !proto.Equal(verdict, decoded.Run.GetVerdict()) {
		return nil, statusAssessmentUnreproducible, fmt.Errorf("the recorded Run evaluates to %s, its recorded Verdict is %s",
			verdict.GetStatus(), decoded.Run.GetVerdict().GetStatus())
	}
	return evaluated.Assessment, "", nil
}

// offlineNames are the deployment names a Case is prepared offline under. The Contract and the
// assessment read the recorded events, never a deployment name, so any names serve.
const offlineNames = "umpire-assess"

// prepareOffline prepares the Case to read a recorded Run of it, under the Profile name the Run was
// recorded with. The Run was recorded in an environment that supplied what its Case requires, or the
// Case would not have prepared there, so it is prepared here with exactly that: every setting the
// Program requires, and delivery control. Nothing is driven, so neither is exercised.
func prepareOffline(source *testpilotspb.Case, identity string) (*testpilot.PreparedCase, error) {
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return nil, err
	}
	settings := map[string]string{}
	for _, setting := range source.GetProgram().GetRequiredSettings() {
		settings[setting.GetKey()] = setting.GetValue()
	}
	handlerQueue := ""
	if testpilotdriver.HandlerTaskQueueBindingID(source.GetProgram()) != "" {
		handlerQueue = offlineNames + "-handler"
	}
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: identity, Namespace: offlineNames, TaskQueue: offlineNames, HandlerTaskQueue: handlerQueue,
		NexusEndpoint: offlineNames, DeliveryControl: true, DynamicConfig: settings,
	})
	if err != nil {
		return nil, err
	}
	return testpilot.Prepare(source, profile)
}

func (env environment) context() (context.Context, context.CancelFunc) {
	if env.Context != nil {
		return env.Context()
	}
	return cli.Interruptible(context.Background(), assessTimeout)
}

func (env environment) decode(rendered []byte) (*evaluation.Receipt, error) {
	if env.Decode != nil {
		return env.Decode(rendered)
	}
	return evaluation.DecodeReceipt(rendered)
}

func (env environment) publish(ctx context.Context, root, name string, contents []byte) (publish.Publication, error) {
	if env.Publish != nil {
		return env.Publish(ctx, root, name, contents)
	}
	return publish.Publish(ctx, root, name, contents)
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
			if encoded, err = json.Marshal(result); err != nil {
				cli.WriteLine(stderr, "umpire-assess: encode the summary: %s", err)
				return exitFailed
			}
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
	usage := "usage: umpire-assess run --case <case.json> --run <recorded-run.json> --profile <name> --receipt-root <dir> [--model <dir>] [--model-root <dir>]"
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
	flags.StringVar(&configuration.Model, "model", "", "assess the recorded Run against the Model of the generated Case in this model directory (its cases/ and ir/), beside its Verdict")
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
		return config{}, &profileError{err: err}
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

// profileError says --profile names no loadable Profile: refused before anything is read, and
// reported as an unknown name or as an embedded Profile that does not load.
type profileError struct{ err error }

func (e *profileError) Error() string { return "--profile: " + e.err.Error() }
