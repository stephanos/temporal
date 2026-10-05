package main

import (
	"bytes"
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

const (
	fixtureRoot            = "../../../../tests/testcore/testpilot/testdata"
	asyncCompletionFixture = "generated/nexus-caller-asyncCompletion-case.json"
)

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(fixtureRoot, name)
	_, err := os.Stat(path)
	require.NoError(t, err)
	return path
}

func requiredFlags(t *testing.T, fixture string) []string {
	t.Helper()
	return []string{
		"--case", fixturePath(t, fixture),
		"--grpc", "127.0.0.1:7233",
		"--http", "127.0.0.1:7243",
		"--namespace", "umpire-run-namespace",
		"--task-queue", "umpire-run-queue",
		"--nexus-endpoint", "umpire-run-endpoint",
	}
}

// verdictSession is one bound Case whose Run is already decided, so the exit code and the report
// are the only things under test.
func verdictSession(status testpilotspb.VerdictStatus, rules ...*testpilotspb.RuleVerdict) opener {
	return func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return &session{
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				return &testpilotspb.Run{
						Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
						Cleanup:     &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
					},
					&testpilotspb.Verdict{Status: status, Rules: rules}, nil
			},
		}, nil
	}
}

func TestRunExitsZeroAndReportsEveryRuleVerdictWhenSatisfied(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr,
		verdictSession(testpilotspb.VERDICT_STATUS_SATISFIED,
			&testpilotspb.RuleVerdict{
				RuleId: "clause-one", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
				TerminalStateId: "answered",
			},
			&testpilotspb.RuleVerdict{
				RuleId: "clause-two", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED,
				TerminalStateId: "answered",
			}))

	require.Equal(t, exitSatisfied, code)
	require.Empty(t, stderr.String())
	require.Equal(t, []string{
		"run Completed",
		"cleanup Succeeded",
		"verdict Satisfied",
		"rule clause-one Satisfied answered",
		"rule clause-two Satisfied answered",
	}, strings.Split(strings.TrimSpace(stdout.String()), "\n"))
}

// --record writes the closed Run with the identity it was prepared under, after the report, and
// never replaces a file that exists.
func TestRunRecordsTheClosedRunWhenAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	identity := testpilot.DriverIdentity{Profile: "umpire-run.fuzz", Catalog: "catalog", Bindings: "bindings"}
	open := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return &session{
			identity: identity,
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				verdict := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}
				return &testpilotspb.Run{
					RunId: "run-1", CaseId: "temporal.case.scala.nexus-caller.asyncCompletion", Disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
					Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}, Verdict: verdict,
				}, verdict, nil
			},
		}, nil
	}
	var stdout, stderr bytes.Buffer
	code := Run(append(requiredFlags(t, asyncCompletionFixture), "--record", path), &stdout, &stderr, open)
	require.Equal(t, exitViolated, code, stderr.String())
	recorded, err := os.ReadFile(path)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(recorded)
	require.NoError(t, err)
	require.Equal(t, identity, decoded.Driver)
	fixture, err := os.ReadFile(fixturePath(t, asyncCompletionFixture))
	require.NoError(t, err)
	caseIdentity, err := recordedrun.CaseIdentity(fixture)
	require.NoError(t, err)
	require.Equal(t, caseIdentity, decoded.Case, "the record names the fixture's canonical Case")
	run := decoded.Run
	require.Equal(t, "run-1", run.GetRunId())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, run.GetVerdict().GetStatus())

	// An existing record, or a directory that does not exist, is refused before anything runs.
	opened := false
	refusing := func(context.Context, config, *testpilotspb.Case) (*session, error) { opened = true; return nil, nil }
	stdout.Reset()
	code = Run(append(requiredFlags(t, asyncCompletionFixture), "--record", path), &stdout, &stderr, refusing)
	require.Equal(t, exitFailed, code, "an existing record is never replaced")
	require.False(t, opened)
	require.Contains(t, stderr.String(), "exists and is never replaced")
	stderr.Reset()
	code = Run(append(requiredFlags(t, asyncCompletionFixture), "--record", filepath.Join(t.TempDir(), "missing", "run.json")), &stdout, &stderr, refusing)
	require.Equal(t, exitFailed, code)
	require.False(t, opened)
	require.Contains(t, stderr.String(), "directory does not exist")

	// A record that fails after the Run, a race on the path, keeps the Verdict's exit code and is
	// said on stderr.
	raced := filepath.Join(t.TempDir(), "raced.json")
	racing := func(ctx context.Context, configuration config, source *testpilotspb.Case) (*session, error) {
		require.NoError(t, os.WriteFile(raced, []byte("{}"), 0o644))
		return open(ctx, configuration, source)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(append(requiredFlags(t, asyncCompletionFixture), "--record", raced), &stdout, &stderr, racing)
	require.Equal(t, exitViolated, code)
	require.Contains(t, stdout.String(), "verdict Violated")
	require.Contains(t, stderr.String(), "exist")
	require.Equal(t, "{}", string(mustRead(t, raced)), "the raced file is never replaced")
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func TestRunSeparatesViolatedInconclusiveAndFailedExitCodes(t *testing.T) {
	for _, probe := range []struct {
		name   string
		status testpilotspb.VerdictStatus
		code   int
	}{
		{"violated", testpilotspb.VERDICT_STATUS_VIOLATED, exitViolated},
		{"inconclusive", testpilotspb.VERDICT_STATUS_INCONCLUSIVE, exitInconclusive},
	} {
		t.Run(probe.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr,
				verdictSession(probe.status))

			require.Equal(t, probe.code, code)
			require.Contains(t, stdout.String(), "verdict "+probe.status.String())
			require.Empty(t, stderr.String())
		})
	}
}

// Each Run diagnostic is one line after the rule lines, by its kind and code alone, so a caller in
// another process can tell why a Run was inconclusive without the Run itself.
func TestRunReportsEveryRunDiagnosticAfterTheRules(t *testing.T) {
	var stdout, stderr bytes.Buffer
	open := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return &session{
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				verdict := &testpilotspb.Verdict{
					Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE,
					Rules:  []*testpilotspb.RuleVerdict{{RuleId: "clause-one", Status: testpilotspb.RULE_VERDICT_STATUS_PENDING, TerminalStateId: "awaiting"}},
				}
				return &testpilotspb.Run{
					Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
					Cleanup:     &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
					Verdict:     verdict,
					Diagnostics: []*testpilotspb.RunDiagnostic{
						{DiagnosticId: "d-1", Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION, Code: "instruction-timeout", Detail: "finish-workflow in run-1"},
						{DiagnosticId: "d-2", Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_MONITOR, Code: "pending", Detail: "clause-one in run-1"},
					},
				}, verdict, nil
			},
		}, nil
	}

	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr, open)

	require.Equal(t, exitInconclusive, code)
	require.Empty(t, stderr.String())
	require.Equal(t, []string{
		"run Incomplete",
		"cleanup Succeeded",
		"verdict Inconclusive",
		"rule clause-one Pending awaiting",
		"diagnostic Execution instruction-timeout",
		"diagnostic Monitor pending",
	}, strings.Split(strings.TrimSpace(stdout.String()), "\n"))
}

// A Run that could not execute is exit 3, not the inconclusive Verdict it never reached. That
// separation is what lets CI tell an unreachable server from a real inconclusive answer.
func TestRunExitsThreeWithOneStderrLineWhenTheRunFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failing := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return &session{
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				return nil, nil, errors.New("frontend refused the connection")
			},
		}, nil
	}

	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr, failing)

	require.Equal(t, exitFailed, code)
	require.Empty(t, stdout.String())
	require.Equal(t, []string{
		`run Case "temporal.case.scala.nexus-caller.asyncCompletion": frontend refused the connection`,
	}, strings.Split(strings.TrimSpace(stderr.String()), "\n"))
}

// A real Driver that cannot answer for itself fails the Run through the real prepared-Case path.
type refusingDriver struct{ err error }

func (d refusingDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return testpilot.DriverIdentity{}, d.err
}

func (d refusingDriver) Validate(context.Context, testpilot.PreparedProgram) error { return d.err }

func (d refusingDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	return nil, d.err
}

func TestRunExitsThreeWhenTheDriverRefusesThePreparedCase(t *testing.T) {
	encoded, err := os.ReadFile(fixturePath(t, asyncCompletionFixture))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: "umpire-run.probe", Namespace: "probe", TaskQueue: "probe-queue",
		HandlerTaskQueue: "probe-queue-handler", NexusEndpoint: "probe-endpoint",
	})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	driver := refusingDriver{err: errors.New("driver is unavailable")}
	var stdout, stderr bytes.Buffer

	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr,
		func(context.Context, config, *testpilotspb.Case) (*session, error) {
			return &session{
				run: func(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
					return prepared.Run(ctx, driver)
				},
			}, nil
		})

	require.Equal(t, exitFailed, code)
	require.Contains(t, stderr.String(), "driver is unavailable")
}

// A Nexus fixture run against an environment that binds no Nexus endpoint carries a Profile
// `DeriveProfile` cannot derive, so it rejects before any server call, with the admission category
// that says why.
func TestRunRejectsAFixtureWithItsPreparationCategory(t *testing.T) {
	encoded, err := os.ReadFile(fixturePath(t, asyncCompletionFixture))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	_, deriveErr := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: "umpire-run.probe", Namespace: "probe", TaskQueue: "probe-queue",
	})
	require.Error(t, deriveErr)

	var stdout, stderr bytes.Buffer
	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr,
		func(_ context.Context, _ config, source *testpilotspb.Case) (*session, error) {
			_, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
				Identity: "umpire-run.probe", Namespace: "probe", TaskQueue: "probe-queue",
			})
			return nil, err
		})

	require.Equal(t, exitFailed, code)
	require.Empty(t, stdout.String())
	require.NotEmpty(t, stderr.String())
}

// A static admission rejection is reported by its category, not as an opaque error string.
func TestDescribeFailureNamesThePreparationCategory(t *testing.T) {
	rejection := &testpilot.PreparationError{
		Category: testpilot.PreparationUnsupported,
		Path:     "program.instructions[3]",
		Detail:   "opcode is outside the Profile",
	}

	require.Equal(t,
		"prepare Case: unsupported at program.instructions[3]: opcode is outside the Profile",
		describeFailure(rejection))
	require.Equal(t, "plain failure", describeFailure(errors.New("plain failure")))
}

func TestRunRejectsMissingFlagsAndPositionalArgumentsBeforeAnyServerCall(t *testing.T) {
	for _, probe := range []struct {
		name      string
		arguments []string
		message   string
	}{
		{"no case", []string{"--grpc", "a", "--http", "b", "--namespace", "c", "--task-queue", "d"}, "--case is required"},
		{"no namespace", []string{"--case", "x", "--grpc", "a", "--http", "b", "--task-queue", "d"}, "--namespace is required"},
		{"positional", append(requiredFlags(t, asyncCompletionFixture), "extra"), "accepts no positional arguments"},
		{"non-positive timeout", append(requiredFlags(t, asyncCompletionFixture), "--timeout", "0s"), "--timeout must be positive"},
	} {
		t.Run(probe.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			opened := false

			code := Run(probe.arguments, &stdout, &stderr,
				func(context.Context, config, *testpilotspb.Case) (*session, error) {
					opened = true
					return nil, nil
				})

			require.Equal(t, exitFailed, code)
			require.False(t, opened, "no binding is attempted on a rejected command line")
			require.Contains(t, stderr.String(), probe.message)
		})
	}
}

func TestRunReportsAnUnreadableOrUndecodableFixture(t *testing.T) {
	unreadable := filepath.Join(t.TempDir(), "missing-case.json")
	var stdout, stderr bytes.Buffer

	code := Run([]string{
		"--case", unreadable, "--grpc", "a", "--http", "b",
		"--namespace", "c", "--task-queue", "d",
	}, &stdout, &stderr, verdictSession(testpilotspb.VERDICT_STATUS_SATISFIED))

	require.Equal(t, exitFailed, code)
	require.Contains(t, stderr.String(), "read Case fixture")
}

// The timeout bounds the whole Run, including a binding that never completes.
func TestRunHonoursTheTimeoutWhileBinding(t *testing.T) {
	var stdout, stderr bytes.Buffer
	blocking := func(ctx context.Context, _ config, _ *testpilotspb.Case) (*session, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	started := time.Now()
	code := Run(append(requiredFlags(t, asyncCompletionFixture), "--timeout", "50ms"),
		&stdout, &stderr, blocking)

	require.Equal(t, exitFailed, code)
	require.Less(t, time.Since(started), 30*time.Second)
	require.Contains(t, stderr.String(), context.DeadlineExceeded.Error())
}

// SIGINT cancels the Run's context. Teardown runs afterwards on its own context, so an interrupted
// Run still releases what it created.
func TestInterruptibleContextCancelsOnSIGINT(t *testing.T) {
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	ctx, cancel := cli.Interruptible(context.Background(), time.Minute)
	defer cancel()

	if err := process.Signal(os.Interrupt); err != nil {
		t.Skipf("this platform cannot deliver os.Interrupt to itself: %v", err)
	}

	select {
	case <-ctx.Done():
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("SIGINT did not cancel the Run context")
	}
}

// A `--create` collision is an error, not a silent reuse: the caller asked to own the resources.
func TestRunReportsACreateCollision(t *testing.T) {
	var stdout, stderr bytes.Buffer
	colliding := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return nil, errors.New(`register namespace "umpire-run-namespace": namespace already exists`)
	}

	code := Run(append(requiredFlags(t, asyncCompletionFixture), "--create"),
		&stdout, &stderr, colliding)

	require.Equal(t, exitFailed, code)
	require.Contains(t, stderr.String(), "namespace already exists")
}

// Teardown reports one line per resource it could not remove, rather than one joined blob.
func TestRunReportsEveryLeakedResourceOnItsOwnLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	leaking := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		bound, _ := verdictSession(testpilotspb.VERDICT_STATUS_SATISFIED)(context.Background(), config{}, nil)
		bound.release = func(context.Context) error {
			return errors.Join(
				errors.New(`delete Nexus endpoint umpire-run-endpoint: still referenced`),
				errors.New(`delete namespace umpire-run-namespace: deletion in progress`))
		}
		return bound, nil
	}

	code := Run(requiredFlags(t, asyncCompletionFixture), &stdout, &stderr, leaking)

	require.Equal(t, exitSatisfied, code)
	require.Equal(t, []string{
		"delete Nexus endpoint umpire-run-endpoint: still referenced",
		"delete namespace umpire-run-namespace: deletion in progress",
	}, strings.Split(strings.TrimSpace(stderr.String()), "\n"))
}

// The CLI imports the Driver and the SDK, never the functional test cluster or a server service.
// The fn-70 deferral note recorded that exporting the live-test binding pulled the whole server in;
// the coupling was the provisioning's test environment, and this pins that it stays gone.
func TestUmpireRunImportsNeitherTheTestClusterNorAServerService(t *testing.T) {
	for name, declared := range packageImports(t) {
		for _, imported := range declared {
			require.NotContains(t, imported, "go.temporal.io/server/tests/testcore", "in %s", name)
			require.NotContains(t, imported, "go.temporal.io/server/service/", "in %s", name)
		}
	}
}

// packageImports returns the import paths each non-test file of this package declares.
func packageImports(t *testing.T) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	declared := map[string][]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imported := range parsed.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			require.NoError(t, err)
			declared[entry.Name()] = append(declared[entry.Name()], path)
		}
	}
	require.NotEmpty(t, declared)
	return declared
}

// The transitive closure carries no functional test cluster at all, and only the server services
// the Driver's own Nexus support already pulled in through `common/dynamicconfig`,
// `common/persistence` and `chasm`. A new one appearing here is a new coupling, which is exactly
// what the fn-70 deferral note was about.
var transitiveServicePackages = []string{
	"go.temporal.io/server/service/history/consts",
	"go.temporal.io/server/service/history/tasks",
	"go.temporal.io/server/service/matching/counter",
}

func TestUmpireRunLinksNoTestClusterAndNoNewServerService(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("the Go toolchain is not on PATH: %v", err)
	}
	listed, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err)

	var services []string
	for _, dependency := range strings.Split(strings.TrimSpace(string(listed)), "\n") {
		require.NotEqual(t, "go.temporal.io/server/tests/testcore", dependency)
		require.False(t, strings.HasPrefix(dependency, "go.temporal.io/server/tests/testcore/"),
			"the CLI must not link the functional test cluster: %s", dependency)
		if strings.HasPrefix(dependency, "go.temporal.io/server/service/") {
			services = append(services, dependency)
		}
	}

	slices.Sort(services)
	require.Equal(t, transitiveServicePackages, services)
}

var _ testpilot.Driver = refusingDriver{}

// A record names the Case by its canonical bytes' identity, so with --record a fixture in no
// canonical form is refused before anything is opened; without --record it still runs.
func TestRunRefusesARecordOfANoncanonicalFixtureBeforeRunning(t *testing.T) {
	fixture, err := os.ReadFile(fixturePath(t, asyncCompletionFixture))
	require.NoError(t, err)
	respaced := filepath.Join(t.TempDir(), "respaced-case.json")
	require.NoError(t, os.WriteFile(respaced, []byte(strings.Replace(string(fixture), "{", "{ ", 1)), 0o644))
	flags := requiredFlags(t, asyncCompletionFixture)
	flags[1] = respaced
	opened := false
	refusing := func(context.Context, config, *testpilotspb.Case) (*session, error) {
		opened = true
		return nil, errors.New("not opened")
	}
	var stdout, stderr bytes.Buffer
	code := Run(append(flags, "--record", filepath.Join(t.TempDir(), "run.json")), &stdout, &stderr, refusing)
	require.Equal(t, exitFailed, code)
	require.False(t, opened, "nothing runs for a record that could not name its Case")
	require.Contains(t, stderr.String(), "not in a canonical form")
	stderr.Reset()
	require.Equal(t, exitFailed, Run(flags, &stdout, &stderr, refusing))
	require.True(t, opened, "without --record the fixture's form does not matter")
}

func TestRunSkipsMissingDeliveryCapabilityBeforeOpening(t *testing.T) {
	arguments := []string{"--case", "../../../../model/cases/activity-race-heldAdmission.staleDelivery-case.json", "--grpc", "localhost:1", "--http", "localhost:2", "--namespace", "ns", "--task-queue", "q"}
	var stdout, stderr bytes.Buffer
	opened := false
	code := Run(arguments, &stdout, &stderr, func(context.Context, config, *testpilotspb.Case) (*session, error) {
		opened = true
		return nil, errors.New("opened before admission")
	})
	require.False(t, opened)
	require.Equal(t, exitFailed, code)
	require.Contains(t, stderr.String(), "skipped: prepare Case: unsupported at controller.hold-dispatch")
}

const (
	modelRoot              = "../../../../model"
	activityCompletionCase = "../../../../model/cases/activity-completion-case.json"
)

// modelFlags run the generated activity completion Case, which expects a satisfied Contract and a
// conformant Run with its `completes` property satisfied, with its Model's assessment.
func modelFlags() []string {
	return []string{
		"--case", activityCompletionCase, "--model", modelRoot,
		"--grpc", "127.0.0.1:7233", "--http", "127.0.0.1:7243",
		"--namespace", "umpire-run-namespace", "--task-queue", "umpire-run-queue",
	}
}

func satisfiedAssessment() *testpilot.Assessment {
	return &testpilot.Assessment{
		Model: "model", Query: "query",
		Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant},
		Properties:  []testpilot.PropertyAssessment{{ID: "completes", Status: testpilot.PropertySatisfied}},
	}
}

// assessedSession is one bound Case whose Run and Model assessment are already decided. It records
// the factory the command bound, which must be the Model's assessment of this very Case.
func assessedSession(t *testing.T, status testpilotspb.VerdictStatus, assessment *testpilot.Assessment) opener {
	return func(_ context.Context, _ config, source *testpilotspb.Case) (*session, error) {
		return &session{
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				t.Fatal("a Run with --model runs with its assessment")
				return nil, nil, nil
			},
			assessed: func(_ context.Context, factory testpilot.AssessmentFactory) (*testpilotspb.Run, *testpilotspb.Verdict, *testpilot.Assessment, error) {
				fingerprint, err := testpilot.CaseFingerprint(source)
				require.NoError(t, err)
				require.Equal(t, fingerprint, factory.Binding().Case)
				require.Contains(t, factory.Binding().Query, "temporal.activity.standalone/activityProtocol/completion#")
				disposition := testpilotspb.RUN_DISPOSITION_COMPLETED
				if status == testpilotspb.VERDICT_STATUS_VIOLATED {
					disposition = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
				}
				return &testpilotspb.Run{Disposition: disposition, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}},
					&testpilotspb.Verdict{Status: status}, assessment, nil
			},
		}, nil
	}
}

// With --model the report adds the conformance and every property with its reason id after the
// Run's lines, then whether the Run is the one the Query expects; the exit code is the worse of the
// Verdict and the assessment, one test per code.
func TestRunWithAModelReportsTheAssessmentAndExitsByTheWorse(t *testing.T) {
	for name, probe := range map[string]struct {
		verdict    testpilotspb.VerdictStatus
		assessment func(*testpilot.Assessment)
		code       int
		lines      []string
	}{
		"satisfied": {testpilotspb.VERDICT_STATUS_SATISFIED, nil, exitSatisfied, []string{
			"conformance conformant", "property completes satisfied", "expected match",
		}},
		"a violated property under a satisfied Verdict": {testpilotspb.VERDICT_STATUS_SATISFIED, func(a *testpilot.Assessment) {
			a.Properties[0].Status, a.Properties[0].Reason = testpilot.PropertyViolated, "every_explanation_violates"
		}, exitViolated, []string{
			"conformance conformant", "property completes violated every_explanation_violates",
			"expected differs: completes's status is violated, expected satisfied",
			"expected differs: completes's reason is every_explanation_violates, expected none",
		}},
		"nonconformant": {testpilotspb.VERDICT_STATUS_SATISFIED, func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, Reason: "unexplained"}
			a.Properties[0].Status, a.Properties[0].Reason = testpilot.PropertyInconclusive, "unexplained"
		}, exitViolated, []string{
			"conformance nonconformant unexplained", "property completes inconclusive unexplained",
			"expected differs: the conformance is nonconformant, expected conformant",
			"expected differs: completes's status is inconclusive, expected satisfied",
			"expected differs: completes's reason is unexplained, expected none",
		}},
		"inconclusive": {testpilotspb.VERDICT_STATUS_SATISFIED, func(a *testpilot.Assessment) {
			a.Properties[0].Status, a.Properties[0].Reason = testpilot.PropertyInconclusive, "never_evaluated"
		}, exitInconclusive, []string{
			"conformance conformant", "property completes inconclusive never_evaluated",
			"expected differs: completes's status is inconclusive, expected satisfied",
			"expected differs: completes's reason is never_evaluated, expected none",
		}},
		"an inconclusive Verdict": {testpilotspb.VERDICT_STATUS_INCONCLUSIVE, nil, exitInconclusive, []string{
			"conformance conformant", "property completes satisfied",
			"expected differs: the Contract's Verdict is inconclusive, expected satisfied",
		}},
		"a violated Verdict": {testpilotspb.VERDICT_STATUS_VIOLATED, nil, exitViolated, []string{
			"conformance conformant", "property completes satisfied",
			"expected differs: the disposition is stopped_by_monitor, expected completed",
			"expected differs: the Contract's Verdict is violated, expected satisfied",
		}},
		"an assessment failure": {testpilotspb.VERDICT_STATUS_SATISFIED, func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive}
			a.Properties = nil
			a.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentLimitExceeded, Detail: "work ceiling", EventSequence: 7}
		}, exitFailed, []string{
			"conformance inconclusive", "assessment failed limit_exceeded at event 7",
			"expected differs: the assessment failed: limit_exceeded work ceiling",
			"expected differs: the conformance is inconclusive, expected conformant",
			"expected differs: the assessment concludes 0 claims, expected 1",
			"expected differs: the assessment omits completes",
		}},
		"a violation established before the assessment failed": {testpilotspb.VERDICT_STATUS_SATISFIED, func(a *testpilot.Assessment) {
			a.Conformance = testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive}
			a.Properties[0].Status = testpilot.PropertyViolated
			a.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, EventSequence: 3}
		}, exitViolated, nil},
	} {
		t.Run(name, func(t *testing.T) {
			assessment := satisfiedAssessment()
			if probe.assessment != nil {
				probe.assessment(assessment)
			}
			var stdout, stderr bytes.Buffer
			code := Run(modelFlags(), &stdout, &stderr, assessedSession(t, probe.verdict, assessment))
			require.Equal(t, probe.code, code, stderr.String())
			require.Empty(t, stderr.String())
			lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			require.Equal(t, "verdict "+probe.verdict.String(), lines[2])
			if probe.lines != nil {
				require.Equal(t, probe.lines, lines[3:])
			}
		})
	}
}

// A Case the model directory does not lower, or a model directory that cannot be read, is refused
// with exit 3 before any binding is opened.
func TestRunWithAModelRefusesACaseItDoesNotAssessBeforeOpening(t *testing.T) {
	for name, probe := range map[string]struct {
		arguments []string
		message   string
	}{
		"a hand-written Case": {append(requiredFlags(t, "nexusPairTests-bothComplete-case.json"), "--model", modelRoot), "not a generated Case"},
		"no model directory":  {append(requiredFlags(t, asyncCompletionFixture), "--model", t.TempDir()), "manifest.json"},
	} {
		t.Run(name, func(t *testing.T) {
			opened := false
			var stdout, stderr bytes.Buffer
			code := Run(probe.arguments, &stdout, &stderr, func(context.Context, config, *testpilotspb.Case) (*session, error) {
				opened = true
				return nil, errors.New("not opened")
			})
			require.Equal(t, exitFailed, code)
			require.False(t, opened, "nothing is opened for a Case the Model does not assess")
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), "--model: ")
			require.Contains(t, stderr.String(), probe.message)
		})
	}
}

// A binding that cannot run an assessment fails the Run rather than running it unassessed.
func TestRunWithAModelNeedsABindingThatAssesses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(modelFlags(), &stdout, &stderr, verdictSession(testpilotspb.VERDICT_STATUS_SATISFIED))
	require.Equal(t, exitFailed, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "cannot run a Model assessment")
}
