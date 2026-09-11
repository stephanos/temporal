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
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
)

const fixtureRoot = "../../../../tests/testcore/testpilot/testdata"

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
	}
}

// verdictSession is one bound Case whose Run is already decided, so the exit code and the report
// are the only things under test.
func verdictSession(status testpilotspb.VerdictStatus, rules ...*testpilotspb.RuleVerdict) opener {
	return func(context.Context, config, *testpilotspb.Case) (*session, error) {
		return &session{
			run: func(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				return &testpilotspb.Run{
						Status:  testpilotspb.RUN_STATUS_COMPLETED,
						Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED},
					},
					&testpilotspb.Verdict{Status: status, Rules: rules}, nil
			},
		}, nil
	}
}

func TestRunExitsZeroAndReportsEveryRuleVerdictWhenSatisfied(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run(requiredFlags(t, "async-nexus-case.json"), &stdout, &stderr,
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

			code := Run(requiredFlags(t, "async-nexus-case.json"), &stdout, &stderr,
				verdictSession(probe.status))

			require.Equal(t, probe.code, code)
			require.Contains(t, stdout.String(), "verdict "+probe.status.String())
			require.Empty(t, stderr.String())
		})
	}
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

	code := Run(requiredFlags(t, "async-nexus-case.json"), &stdout, &stderr, failing)

	require.Equal(t, exitFailed, code)
	require.Empty(t, stdout.String())
	require.Equal(t, []string{
		`run Case "temporal.case.async-nexus": frontend refused the connection`,
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
	encoded, err := os.ReadFile(fixturePath(t, "async-nexus-case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := testpilotdriver.DeriveProfile(source, catalog, testpilotdriver.Environment{
		Identity: "umpire-run.probe", Namespace: "probe", TaskQueue: "probe-queue",
		NexusEndpoint: "probe-endpoint",
	})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	driver := refusingDriver{err: errors.New("driver is unavailable")}
	var stdout, stderr bytes.Buffer

	code := Run(requiredFlags(t, "async-nexus-case.json"), &stdout, &stderr,
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

// A typed fixture carries a Profile `DeriveProfile` cannot derive, so it rejects before any server
// call, with the admission category that says why.
func TestRunRejectsATypedFixtureWithItsPreparationCategory(t *testing.T) {
	encoded, err := os.ReadFile(fixturePath(t, "typed-nexus-case.json"))
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
	code := Run(requiredFlags(t, "typed-nexus-case.json"), &stdout, &stderr,
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
		{"positional", append(requiredFlags(t, "async-nexus-case.json"), "extra"), "accepts no positional arguments"},
		{"non-positive timeout", append(requiredFlags(t, "async-nexus-case.json"), "--timeout", "0s"), "--timeout must be positive"},
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
	code := Run(append(requiredFlags(t, "async-nexus-case.json"), "--timeout", "50ms"),
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
	ctx, cancel := interruptible(context.Background(), time.Minute)
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

	code := Run(append(requiredFlags(t, "async-nexus-case.json"), "--create"),
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

	code := Run(requiredFlags(t, "async-nexus-case.json"), &stdout, &stderr, leaking)

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
