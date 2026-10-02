//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/protobuf/proto"
)

// umpireRepeatRunDirVariable names the directory umpire-repeat collects every closed Run of the
// affected live tests in; unset, nothing is written.
const umpireRepeatRunDirVariable = "UMPIRE_REPEAT_RUN_DIR"

var capturedRunCount atomic.Int64

// requireSigned runs check and, when check fails the test, logs the signature sign builds for
// assertion, so one failing run tells its cause without a rerun. A test that had already failed is
// not signed again: the first signature line is the one umpire-repeat counts.
func requireSigned(t testing.TB, assertion string, sign func(assertion string) recordedrun.Signature, check func()) {
	t.Helper()
	failedBefore := t.Failed()
	defer func() {
		if failedBefore || !t.Failed() {
			return
		}
		line, err := sign(assertion).Line()
		if err != nil {
			t.Errorf("render the failure signature of %q: %v", assertion, err)
			return
		}
		t.Log(line)
	}()
	check()
}

// liveRunCheck is one closed Run under assertion; its label names the Run among several in one
// test, and prefixes every assertion it signs.
type liveRunCheck struct {
	t       testing.TB
	label   string
	run     *testpilotpb.Run
	verdict *testpilotpb.Verdict
}

// require runs check as the assertion named, signing its failure with the Run's signature.
func (c liveRunCheck) require(assertion string, check func()) {
	c.t.Helper()
	if c.label != "" {
		assertion = c.label + ": " + assertion
	}
	requireSigned(c.t, assertion, func(assertion string) recordedrun.Signature {
		return recordedrun.RunSignature(c.t.Name(), assertion, c.run, c.verdict)
	}, check)
}

// requireRunSatisfied is what every affected live Run must show before its evidence is read: it
// completed and cleaned up, its Verdict is the Run's own and satisfied, and so is every rule. It
// returns the check its evidence assertions sign their own failures with.
func requireRunSatisfied(t testing.TB, label string, run *testpilotpb.Run, verdict *testpilotpb.Verdict) liveRunCheck {
	t.Helper()
	check := liveRunCheck{t: t, label: label, run: run, verdict: verdict}
	check.require("run disposition", func() {
		require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "%s diagnostics: %v", label, run.GetDiagnostics())
	})
	check.require("cleanup status", func() {
		require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus(), label)
	})
	check.require("verdict status", func() {
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus(), "%s diagnostics: %v", label, run.GetDiagnostics())
	})
	check.require("verdict is the Run's", func() {
		require.True(t, proto.Equal(verdict, run.GetVerdict()), label)
	})
	check.require("rule status", func() {
		for _, rule := range verdict.GetRules() {
			require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus(), "%s rule %s", label, rule.GetRuleId())
		}
	})
	return check
}

// captureRun writes a closed Run of fixture's Case, as the replay package records one, to the
// directory UMPIRE_REPEAT_RUN_DIR names, before anything is asserted about it, so passing and
// failing Runs alike are there to be timed.
func captureRun(t testing.TB, fixture string, live testpilotLiveCase, run *testpilotpb.Run) {
	t.Helper()
	dir := os.Getenv(umpireRepeatRunDirVariable)
	if dir == "" || run == nil {
		return
	}
	caseBytes, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", fixture+"-case.json"))
	require.NoError(t, err)
	path := capturePath(t, dir)
	require.NoError(t, recordedrun.Write(path, caseBytes, live.prepared.Identity(), run), "capture the Run to %s", path)
}

// capturePath is a file name under dir that no other Run of this process or of another one takes:
// the in-process mode repeats test names, so the test name alone is not enough.
func capturePath(t testing.TB, dir string) string {
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	return filepath.Join(dir, name+"-"+strconv.FormatInt(time.Now().UnixNano(), 10)+"-"+strconv.FormatInt(capturedRunCount.Add(1), 10)+".json")
}

// runCapturedCase runs fixture's Case once under the binding its own bytes derive and captures the
// closed Run.
func runCapturedCase(t *testing.T, env *testcore.TestEnv, fixture string) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	source := loadTestpilotCase(t, fixture)
	return runCapturedBoundCase(t, env, fixture, source, defaultBinding(fixture, source))
}

// runCapturedCaseWithBinding runs fixture's Case once under binding and captures the closed Run.
func runCapturedCaseWithBinding(t *testing.T, env *testcore.TestEnv, fixture string, binding CaseBinding) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	return runCapturedBoundCase(t, env, fixture, loadTestpilotCase(t, fixture), binding)
}

func runCapturedBoundCase(t *testing.T, env *testcore.TestEnv, fixture string, source *testpilotpb.Case, binding CaseBinding) (*testpilotpb.Run, *testpilotpb.Verdict) {
	t.Helper()
	binding.CreateEndpoint = binding.NexusEndpoint != ""
	live := bindCase(t, env, source, binding)
	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	captureRun(t, fixture, live, run)
	return run, verdict
}
