package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/common/testing/testpilot/publish"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	controlCasePath = "../../../../common/testing/testpilot/replay/testdata/nexusCallerControl-forgedCompletion-case.json"
	controlRunPath  = "../../../../common/testing/testpilot/replay/testdata/nexusCallerControl-forgedCompletion-run.json"
)

// fixedCatalog is the catalog the control Run was recorded under, read from the record so that
// re-recording it is the only edit a catalog change needs.
func fixedCatalog() (string, error) {
	recorded, err := os.ReadFile(controlRunPath)
	if err != nil {
		return "", err
	}
	decoded, err := recordedrun.Decode(recorded)
	if err != nil {
		return "", err
	}
	return decoded.Driver.Catalog, nil
}

// resolvedTemp is a temporary directory named as it really is, so paths the command resolves
// compare equal.
func resolvedTemp(t *testing.T) string {
	t.Helper()
	resolved, err := publish.Resolve(t.TempDir())
	require.NoError(t, err)
	return resolved
}

// subjectFiles writes the pinned control pair unchanged when neither edit is supplied. Synthetic
// subjects omit its authored capability gap so each test isolates the decision it exercises.
func subjectFiles(t *testing.T, editCase func(*testpilotspb.Case), editRun func(*testpilotspb.Run)) (casePath string, runPath string) {
	t.Helper()
	caseBytes, err := os.ReadFile(controlCasePath)
	require.NoError(t, err)
	recorded, err := os.ReadFile(controlRunPath)
	require.NoError(t, err)
	if editCase != nil || editRun != nil {
		source := new(testpilotspb.Case)
		require.NoError(t, protojson.Unmarshal(caseBytes, source))
		source.Provenance.KnownGaps = nil
		if editCase != nil {
			editCase(source)
		}
		encoded, err := protojson.Marshal(source)
		require.NoError(t, err)
		var compact bytes.Buffer
		require.NoError(t, json.Compact(&compact, encoded))
		caseBytes = compact.Bytes()
	}
	if editCase != nil || editRun != nil {
		decoded, err := recordedrun.Decode(recorded)
		require.NoError(t, err)
		run := proto.CloneOf(decoded.Run)
		if editRun != nil {
			editRun(run)
		}
		identity, err := recordedrun.CaseIdentity(caseBytes)
		require.NoError(t, err)
		recorded, err = recordedrun.Encode(identity, decoded.Driver, run)
		require.NoError(t, err)
	}
	directory := t.TempDir()
	casePath, runPath = filepath.Join(directory, "case.json"), filepath.Join(directory, "run.json")
	require.NoError(t, os.WriteFile(casePath, caseBytes, 0o644))
	require.NoError(t, os.WriteFile(runPath, recorded, 0o644))
	return casePath, runPath
}

func satisfy(run *testpilotspb.Run) {
	run.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED
	run.Verdict.Status = testpilotspb.VERDICT_STATUS_SATISFIED
	for _, rule := range run.Verdict.Rules {
		rule.Status = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
	}
}

func flags(casePath, runPath, root string) []string {
	return []string{"run", "--case", casePath, "--run", runPath, "--profile", "local-ephemeral", "--receipt-root", root, "--model-root", "../../../../model"}
}

func run(t *testing.T, arguments []string, env environment) (int, summary, string) {
	t.Helper()
	if env.Catalog == nil {
		env.Catalog = fixedCatalog
	}
	var stdout, stderr bytes.Buffer
	code := Run(arguments, &stdout, &stderr, env)
	var result summary
	if stdout.Len() > 0 {
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &result), stdout.String())
		require.Equal(t, 1, strings.Count(stdout.String(), "\n"), "one summary, one line")
	}
	return code, result, stderr.String()
}

// Each decision exits by what it is, publishes one receipt named by its identity, and a second
// assessment of the same subject is already published.
func TestAssessDecidesAndPublishesOnce(t *testing.T) {
	for name, probe := range map[string]struct {
		editCase func(*testpilotspb.Case)
		editRun  func(*testpilotspb.Run)
		code     int
		reasons  []string
	}{
		"accepted": {nil, satisfy, exitAccepted, nil},
		"rejected": {nil, nil, exitRejected, []string{"verdict-violated", "known-gap-blocking"}},
		"incomplete": {func(source *testpilotspb.Case) {
			source.Provenance.KnownGaps = []*testpilotspb.KnownGap{{Kind: testpilotspb.KNOWN_GAP_KIND_CAPABILITY, Code: "umpire.gap.example"}}
		}, satisfy, exitIncomplete, []string{"known-gap-blocking"}},
	} {
		t.Run(name, func(t *testing.T) {
			casePath, runPath := subjectFiles(t, probe.editCase, probe.editRun)
			root := resolvedTemp(t)
			code, result, stderr := run(t, flags(casePath, runPath, root), environment{})
			require.Equal(t, probe.code, code, stderr)
			require.Equal(t, name, result.Status)
			require.Equal(t, probe.reasons, result.Reasons)
			require.Equal(t, publish.StatusPublished, result.Publication)
			require.Equal(t, filepath.Join(root, result.Receipt+".json"), result.Path)
			published, err := os.ReadFile(result.Path)
			require.NoError(t, err)
			require.Equal(t, result.Receipt, evaluation.ReceiptIdentity(published))
			receipt, err := evaluation.DecodeReceipt(published)
			require.NoError(t, err)
			require.Equal(t, name, receipt.Decision)
			require.Contains(t, stderr, "assess "+name)

			code, again, _ := run(t, flags(casePath, runPath, root), environment{})
			require.Equal(t, probe.code, code)
			require.Equal(t, publish.StatusAlreadyPublished, again.Publication)
			require.Equal(t, result.Receipt, again.Receipt)
			listed, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Len(t, listed, 1, "one receipt, and no temporary file left")
		})
	}
}

// The command line is refused before anything is read.
func TestAssessRefusesTheCommandLine(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, nil)
	root := resolvedTemp(t)
	model := resolvedTemp(t)
	for name, arguments := range map[string][]string{
		"no subcommand":          {},
		"another subcommand":     {"assess"},
		"no --case":              {"run", "--run", runPath, "--profile", "local-ephemeral", "--receipt-root", root},
		"no --run":               {"run", "--case", casePath, "--profile", "local-ephemeral", "--receipt-root", root},
		"no --profile":           {"run", "--case", casePath, "--run", runPath, "--receipt-root", root},
		"no --receipt-root":      {"run", "--case", casePath, "--run", runPath, "--profile", "local-ephemeral"},
		"a root under the model": {"run", "--case", casePath, "--run", runPath, "--profile", "local-ephemeral", "--receipt-root", filepath.Join(model, "receipts"), "--model-root", model},
		"a missing root":         {"run", "--case", casePath, "--run", runPath, "--profile", "local-ephemeral", "--receipt-root", filepath.Join(root, "missing")},
		"a positional argument":  append(flags(casePath, runPath, root), "extra"),
		"a Driver flag":          append(flags(casePath, runPath, root), "--grpc", "127.0.0.1:7233"),
		"a policy flag":          append(flags(casePath, runPath, root), "--policy", "x"),
	} {
		t.Run(name, func(t *testing.T) {
			catalogRead := false
			code, result, _ := run(t, arguments, environment{Catalog: func() (string, error) { catalogRead = true; return fixedCatalog() }})
			require.Equal(t, exitFailed, code)
			require.Empty(t, result.Status, "no summary for a refused command line")
			require.False(t, catalogRead, "nothing is read for a refused command line")
		})
	}
	// A Profile is a name from the embedded set, never a path; any other is refused before anything
	// is read, with its own status.
	for _, profile := range []string{"local", "profiles/local-ephemeral.json", "../evaluation/profiles/local-ephemeral"} {
		catalogRead := false
		arguments := []string{"run", "--case", casePath, "--run", runPath, "--profile", profile, "--receipt-root", root}
		code, result, _ := run(t, arguments, environment{Catalog: func() (string, error) { catalogRead = true; return fixedCatalog() }})
		require.Equal(t, exitFailed, code)
		require.Equal(t, statusUnknownProfile, result.Status, profile)
		require.Contains(t, result.Detail, "no such Evaluation Profile")
		require.False(t, catalogRead)
	}
	listed, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, listed)
}

// Every failure that is not a decision exits 3 with its own named status and publishes nothing new.
func TestAssessNamesEveryFailure(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, nil)
	cancelled := func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, cancel
	}
	expired := func() (context.Context, context.CancelFunc) {
		return context.WithDeadline(context.Background(), time.Unix(0, 0))
	}
	for name, probe := range map[string]struct {
		casePath, runPath string
		env               environment
		status            string
		detail            string
	}{
		"an unreadable Case": {filepath.Join(t.TempDir(), "missing.json"), runPath, environment{}, statusUnreadableInput, "--case"},
		"an unreadable Run":  {casePath, filepath.Join(t.TempDir(), "missing.json"), environment{}, statusUnreadableInput, "--run"},
		"no catalog":         {casePath, runPath, environment{Catalog: func() (string, error) { return "", errors.New("no descriptors") }}, statusCatalogUnavailable, "no descriptors"},
		"a stale Run":        {casePath, runPath, environment{Catalog: func() (string, error) { return "another-catalog", nil }}, statusRejectedSubject, "catalog"},
		"interrupted":        {casePath, runPath, environment{Context: cancelled}, statusInterrupted, "interrupted before publishing"},
		"past its deadline":  {casePath, runPath, environment{Context: expired}, statusInterrupted, "deadline exceeded"},
		"a Run of another Case": {casePath, func() string {
			_, other := subjectFiles(t, func(source *testpilotspb.Case) { source.CaseId = "temporal.case.other" }, nil)
			return other
		}(), environment{}, statusRejectedSubject, "recorded from Case"},
	} {
		t.Run(name, func(t *testing.T) {
			root := resolvedTemp(t)
			code, result, stderr := run(t, flags(probe.casePath, probe.runPath, root), probe.env)
			require.Equal(t, exitFailed, code)
			require.Equal(t, probe.status, result.Status)
			require.Contains(t, result.Detail, probe.detail)
			require.Contains(t, stderr, "assess "+probe.status)
			listed, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Empty(t, listed, "nothing is published and no temporary file is left")
		})
	}
}

// A subject's rejection names its class; an oversized input is refused having read one byte past
// its cap.
func TestAssessReportsTheRejectionClass(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, nil)
	oversized := filepath.Join(t.TempDir(), "case.json")
	require.NoError(t, os.WriteFile(oversized, bytes.Repeat([]byte(" "), evaluation.MaxCaseBytes+1024), 0o644))
	root := resolvedTemp(t)
	code, result, _ := run(t, flags(oversized, runPath, root), environment{})
	require.Equal(t, exitFailed, code)
	require.Equal(t, statusRejectedSubject, result.Status)
	require.Equal(t, evaluation.ReasonOversized, result.Rejection)
	require.Contains(t, result.Detail, "is 4194305 bytes", "only the cap and one byte are read")

	respaced := filepath.Join(t.TempDir(), "run.json")
	recorded, err := os.ReadFile(runPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(respaced, []byte(strings.Replace(string(recorded), `":`, `": `, 1)), 0o644))
	code, result, _ = run(t, flags(casePath, respaced, root), environment{})
	require.Equal(t, exitFailed, code)
	require.Equal(t, evaluation.ReasonNoncanonical, result.Rejection)
}

// A receipt whose name holds other bytes is a conflict the command reports and never overwrites.
func TestAssessReportsAConflict(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, satisfy)
	root := resolvedTemp(t)
	code, result, _ := run(t, flags(casePath, runPath, root), environment{})
	require.Equal(t, exitAccepted, code)
	require.NoError(t, os.WriteFile(result.Path, []byte("tampered\n"), 0o644))
	code, conflict, _ := run(t, flags(casePath, runPath, root), environment{})
	require.Equal(t, exitFailed, code)
	require.Equal(t, statusPublicationConflict, conflict.Status)
	require.Equal(t, result.Receipt, conflict.Receipt)
	stored, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	require.Equal(t, "tampered\n", string(stored))
}

// A receipt over its cap is the named tooling failure and is never published. Three fully
// supported rules over the event cap's worth of events are more than the cap holds.
func TestAssessRefusesAnOversizedReceipt(t *testing.T) {
	casePath, runPath := subjectFiles(t, func(source *testpilotspb.Case) {
		for _, name := range []string{"extra-1", "extra-2"} {
			rule := proto.CloneOf(source.Contract.Correlated.Rules[0])
			rule.RuleId = name
			source.Contract.Correlated.Rules = append(source.Contract.Correlated.Rules, rule)
		}
	}, func(run *testpilotspb.Run) {
		satisfy(run)
		for len(run.Events) < evaluation.MaxRunEvents {
			run.Events = append(run.Events, &testpilotspb.RunEvent{Sequence: int64(len(run.Events) + 1)})
		}
		sequences := make([]int64, evaluation.MaxRunEvents)
		for index := range sequences {
			sequences[index] = int64(index + 1)
		}
		for _, name := range []string{"extra-1", "extra-2"} {
			run.Verdict.Rules = append(run.Verdict.Rules, &testpilotspb.RuleVerdict{RuleId: name, Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED})
		}
		for _, rule := range run.Verdict.Rules {
			rule.TerminalStateId = "done"
			rule.SupportingEventSequences = sequences
		}
	})
	root := resolvedTemp(t)
	code, result, _ := run(t, flags(casePath, runPath, root), environment{})
	require.Equal(t, exitFailed, code)
	require.Equal(t, statusReceiptOversized, result.Status, result.Detail)
	listed, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, listed)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout is closed") }

// A published receipt whose summary cannot be written is the one ambiguity the command names: it
// says so on stderr, exits 3, and the receipt stands.
func TestAssessNamesAPublicationItCouldNotReport(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, satisfy)
	root := resolvedTemp(t)
	var stderr bytes.Buffer
	code := Run(flags(casePath, runPath, root), failingWriter{}, &stderr, environment{Catalog: fixedCatalog})
	require.Equal(t, exitFailed, code)
	var result summary
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &result), stderr.String())
	require.Equal(t, statusPublicationUnreported, result.Status)
	require.Equal(t, publish.StatusPublished, result.Publication)
	_, err := os.Stat(result.Path)
	require.NoError(t, err, "the receipt stands")
}

// The failures no real subject or root produces are reached through the self-check and the
// publisher: a rendered receipt that does not read back, and a publication that fails, each exit 3
// with its own status and publish nothing.
func TestAssessNamesTheSelfCheckAndPublicationFailures(t *testing.T) {
	casePath, runPath := subjectFiles(t, nil, satisfy)
	t.Run("a receipt that does not read back", func(t *testing.T) {
		unreadable := func([]byte) (*evaluation.Receipt, error) { return nil, errors.New("not canonical") }
		root := resolvedTemp(t)
		code, result, _ := run(t, flags(casePath, runPath, root), environment{Decode: unreadable})
		require.Equal(t, exitFailed, code)
		require.Equal(t, statusReceiptUnreadable, result.Status)
		require.Contains(t, result.Detail, "does not read back")
		listed, err := os.ReadDir(root)
		require.NoError(t, err)
		require.Empty(t, listed)
	})
	t.Run("a publication that fails", func(t *testing.T) {
		failing := func(context.Context, string, string, []byte) (publish.Publication, error) {
			return publish.Publication{}, errors.New("disk full")
		}
		code, result, _ := run(t, flags(casePath, runPath, resolvedTemp(t)), environment{Publish: failing})
		require.Equal(t, exitFailed, code)
		require.Equal(t, statusPublicationFailed, result.Status)
		require.Contains(t, result.Detail, "disk full")
	})
}

const modelRoot = "../../../../model"

const currentControlCasePath = modelRoot + "/cases/nexus-workflow-control-forgedCompletion-case.json"
const currentControlRunPath = "../../../../common/testing/testpilot/replay/testdata/nexus-workflow-control-forgedCompletion-run.json"

// generatedControl is the generated Case of the control's Query: the pinned control Run is a Run of
// it, which the live test assessed as its Query expects.
func generatedControl(t *testing.T) (*testpilotspb.Case, *lower.GeneratedCase) {
	t.Helper()
	encoded, err := os.ReadFile(currentControlCasePath)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	entry, err := lower.FindGeneratedCase(filepath.Join(modelRoot, "cases"), source)
	require.NoError(t, err)
	require.Equal(t, "nexus-workflow-control-forgedCompletion-case.json", entry.File)
	return source, entry
}

// With --model, the recorded generated Run is assessed by its Model offline, as the live test that
// ran it replays it: the Assessment is the one its Query expects (lower.ExpectedRun.Check, the live
// test's own comparison), the receipt records it beside the Verdict, and it decides beside the
// Verdict.
func TestAssessWithAModelReproducesTheLiveAssessment(t *testing.T) {
	_, entry := generatedControl(t)
	caseBytes, err := os.ReadFile(currentControlCasePath)
	require.NoError(t, err)
	runBytes, err := os.ReadFile(currentControlRunPath)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(runBytes)
	require.NoError(t, err)
	identity, err := recordedrun.CaseIdentity(caseBytes)
	require.NoError(t, err)
	require.Equal(t, identity, decoded.Case, "crossed: the recorded Run is not of the current generated control Case")
	catalog, err := treeCatalog()
	require.NoError(t, err)
	require.Equal(t, catalog, decoded.Driver.Catalog, "stale: the recorded Run is not of the current Driver catalog")
	assessed, status, err := assessRecorded(modelRoot, caseBytes, runBytes)
	require.NoError(t, err, status)
	require.NoError(t, entry.Expected.Check(decoded.Run, decoded.Run.GetVerdict(), assessed))
	again, _, err := assessRecorded(modelRoot, caseBytes, runBytes)
	require.NoError(t, err)
	require.Equal(t, assessed, again, "the offline assessment is a function of the recorded Run")

	root := resolvedTemp(t)
	code, result, stderr := run(t, append(flags(currentControlCasePath, currentControlRunPath, root), "--model", modelRoot), environment{Catalog: treeCatalog})
	require.Equal(t, exitRejected, code, stderr)
	require.Equal(t, []string{"verdict-violated", "known-gap-blocking", "property-violated", "assessment-inconclusive"}, result.Reasons)
	published, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	receipt, err := evaluation.DecodeReceipt(published)
	require.NoError(t, err)
	require.NotNil(t, receipt.Assessment)
	require.Equal(t, assessed.Model, receipt.Assessment.Model)
	require.Equal(t, assessed.Query, receipt.Assessment.Query)
	require.Equal(t, entry.Expected.Conformance, receipt.Assessment.Conformance.Status)
	require.Equal(t, assessed.Conformance.Reason, receipt.Assessment.Conformance.Reason)
	require.Len(t, receipt.Assessment.Properties, len(entry.Expected.Properties))
	for i, expected := range entry.Expected.Properties {
		require.Equal(t, expected.ID, receipt.Assessment.Properties[i].ID)
		require.Equal(t, expected.Status, receipt.Assessment.Properties[i].Status)
		require.Equal(t, expected.Reason, receipt.Assessment.Properties[i].Reason)
	}
	require.Nil(t, receipt.Assessment.Failure)

	// Without --model the same subject is decided on its Verdict alone, and its receipt is another.
	code, plain, _ := run(t, flags(currentControlCasePath, currentControlRunPath, root), environment{Catalog: treeCatalog})
	require.Equal(t, exitRejected, code)
	require.Equal(t, []string{"verdict-violated", "known-gap-blocking"}, plain.Reasons)
	require.NotEqual(t, result.Receipt, plain.Receipt)
}

// A Case the model directory does not lower is not assessed: the subject is refused with its own
// status and nothing is published. A recorded Run that does not read back to its recorded Verdict is
// refused too, since its assessment would not be of the Run that was recorded.
func TestAssessWithAModelRefusesWhatItCannotAssess(t *testing.T) {
	notGenerated, notGeneratedRun := subjectFiles(t, nil, satisfy)
	unreproducible := func() (string, string) {
		caseBytes, err := os.ReadFile(currentControlCasePath)
		require.NoError(t, err)
		recorded, err := os.ReadFile(currentControlRunPath)
		require.NoError(t, err)
		decoded, err := recordedrun.Decode(recorded)
		require.NoError(t, err)
		lying := proto.CloneOf(decoded.Run)
		satisfy(lying)
		identity, err := recordedrun.CaseIdentity(caseBytes)
		require.NoError(t, err)
		encoded, err := recordedrun.Encode(identity, decoded.Driver, lying)
		require.NoError(t, err)
		runPath := filepath.Join(t.TempDir(), "run.json")
		require.NoError(t, os.WriteFile(runPath, encoded, 0o644))
		return currentControlCasePath, runPath
	}
	lyingCase, lyingRun := unreproducible()
	for name, probe := range map[string]struct {
		arguments []string
		catalog   func() (string, error)
		status    string
		detail    string
	}{
		"a Case the Model does not lower": {append(flags(notGenerated, notGeneratedRun, ""), "--model", modelRoot), fixedCatalog, statusModelUnassessable, "not a generated Case"},
		"no model directory":              {append(flags(controlCasePath, controlRunPath, ""), "--model", resolvedTemp(t)), fixedCatalog, statusModelUnassessable, "manifest.json"},
		"a Run that does not read back":   {append(flags(lyingCase, lyingRun, ""), "--model", modelRoot), treeCatalog, statusAssessmentUnreproducible, "completed disposition conflicts"},
	} {
		t.Run(name, func(t *testing.T) {
			root := resolvedTemp(t)
			arguments := slices.Clone(probe.arguments)
			arguments[slices.Index(arguments, "--receipt-root")+1] = root
			code, result, _ := run(t, arguments, environment{Catalog: probe.catalog})
			require.Equal(t, exitFailed, code)
			require.Equal(t, probe.status, result.Status)
			require.Contains(t, result.Detail, probe.detail)
			listed, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Empty(t, listed, "nothing is published")
		})
	}
}

// A receipt root inside the model directory --model reads is refused before anything is read, as
// one inside --model-root is, whatever --model-root says.
func TestAssessWithAModelRefusesAReceiptRootInsideIt(t *testing.T) {
	model := resolvedTemp(t)
	inside := filepath.Join(model, "receipts")
	require.NoError(t, os.Mkdir(inside, 0o755))
	arguments := []string{"run", "--case", controlCasePath, "--run", controlRunPath, "--profile", "local-ephemeral",
		"--receipt-root", inside, "--model-root", resolvedTemp(t), "--model", model}
	catalogRead := false
	code, result, stderr := run(t, arguments, environment{Catalog: func() (string, error) { catalogRead = true; return fixedCatalog() }})
	require.Equal(t, exitFailed, code)
	require.Empty(t, result.Status, "no summary for a refused command line")
	require.False(t, catalogRead, "nothing is read for a refused command line")
	require.Contains(t, stderr, "--receipt-root must not be under the model root")
	listed, err := os.ReadDir(inside)
	require.NoError(t, err)
	require.Empty(t, listed)
}
