package replay

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/campaign"
)

// scriptedBinder binds the Case through the preparer and runs it through the scripted Driver, or
// answers with what its script says for that attempt; it records the order of binds and releases.
type scriptedBinder struct {
	prepare  Preparer
	script   []func(prepared *testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error)
	identity func(testpilot.DriverIdentity) testpilot.DriverIdentity
	release  error
	binds    int
	events   []string
	open     bool
}

type scriptedBound struct {
	binder   *scriptedBinder
	prepared *testpilot.PreparedCase
	run      func(prepared *testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error)
}

func (b *scriptedBinder) Bind(_ context.Context, identity string, source *testpilotspb.Case) (campaign.Bound, error) {
	if b.open {
		return nil, errors.New("bound while the previous binding is held")
	}
	prepared, err := b.prepare(identity, source)
	if err != nil {
		return nil, err
	}
	b.open = true
	b.events = append(b.events, "bind")
	run := func(prepared *testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		return prepared.Run(context.Background(), &scriptedDriver{identity: prepared.Identity()})
	}
	if b.binds < len(b.script) && b.script[b.binds] != nil {
		run = b.script[b.binds]
	}
	b.binds++
	return &scriptedBound{binder: b, prepared: prepared, run: run}, nil
}

func (b *scriptedBound) Run(context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	b.binder.events = append(b.binder.events, "run")
	return b.run(b.prepared)
}

func (b *scriptedBound) Release(context.Context) error {
	b.binder.open = false
	b.binder.events = append(b.binder.events, "release")
	return b.binder.release
}

func (b *scriptedBound) Identity() testpilot.DriverIdentity {
	if b.binder.identity != nil {
		return b.binder.identity(b.prepared.Identity())
	}
	return b.prepared.Identity()
}

func admittedSubject(t *testing.T) (*Subject, Preparer) {
	t.Helper()
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	_, _, recorded := recordedRunOf(t, prepare, profileName, caseBytes)
	subject, err := Admit(t.Context(), caseBytes, recorded, prepare)
	require.NoError(t, err)
	return subject, prepare
}

// Two fresh reruns of the corpus's violated subject bind under its exact identity, each released
// before the next binds, and both reproduce its key: the pair is reproduced and every attempt
// carries the key its own Run derived.
func TestRerunReproducesTheSubjectFresh(t *testing.T) {
	subject, prepare := admittedSubject(t)
	binder := &scriptedBinder{prepare: prepare}
	reruns, err := Rerun(t.Context(), binder, subject.Target())
	require.NoError(t, err)
	require.Equal(t, ClassReproduced, reruns.Class)
	require.True(t, reruns.Key.Equal(subject.Key))
	require.Len(t, reruns.Attempts, Attempts)
	require.Equal(t, []string{"bind", "run", "release", "bind", "run", "release"}, binder.events)
	for _, attempt := range reruns.Attempts {
		require.Equal(t, ClassReproduced, attempt.Class)
		require.Empty(t, attempt.Detail)
		require.True(t, attempt.Key.Equal(subject.Key))
		require.Equal(t, subject.Driver, attempt.Identity)
		require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, attempt.Run.GetDisposition())
		require.NotEqual(t, subject.Run.GetRunId(), attempt.Run.GetRunId(), "a fresh Run")
	}
	require.NotEqual(t, reruns.Attempts[0].Run.GetRunId(), reruns.Attempts[1].Run.GetRunId())
}

// Each class, alone and in the pair: a satisfied completed Run and a violated Run with another key
// are not reproduced, which wins over an indeterminate attempt; an incomplete Run, an unclosed
// cleanup, an inconclusive Verdict, a Run that errs and a Run the offline replay does not
// reproduce are indeterminate, which wins over a reproduced one.
func TestRerunClassesEveryOutcomeWithoutAFourth(t *testing.T) {
	subject, prepare := admittedSubject(t)
	opened := []*testpilotspb.RunEvent{{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}}
	closed := &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}
	scripted := func(run *testpilotspb.Run, verdict *testpilotspb.Verdict, err error) func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		return func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
			return run, verdict, err
		}
	}
	satisfied := scripted(
		&testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: closed},
		&testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, nil)
	incomplete := scripted(
		&testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE, Cleanup: closed},
		&testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE}, nil)
	unclosed := scripted(
		&testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_FAILED}},
		&testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, nil)
	errored := scripted(nil, nil, errors.New("the deployment went away"))
	forged := func(prepared *testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		run, verdict, err := prepared.Run(context.Background(), &scriptedDriver{identity: prepared.Identity()})
		if err != nil {
			return nil, nil, err
		}
		run.Verdict.Rules[0].RuleId, verdict.Rules[0].RuleId = "elsewhere", "elsewhere"
		return run, verdict, nil
	}

	for name, probe := range map[string]struct {
		script  []func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error)
		classes []Class
		pair    Class
		detail  string
	}{
		"satisfied then reproduced":      {[]func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error){satisfied, nil}, []Class{ClassNotReproduced, ClassReproduced}, ClassNotReproduced, "satisfied"},
		"incomplete then satisfied":      {[]func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error){incomplete, satisfied}, []Class{ClassIndeterminate, ClassNotReproduced}, ClassNotReproduced, "incomplete"},
		"reproduced then unclosed":       {[]func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error){nil, unclosed}, []Class{ClassReproduced, ClassIndeterminate}, ClassIndeterminate, "not closed"},
		"errored then reproduced":        {[]func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error){errored, nil}, []Class{ClassIndeterminate, ClassReproduced}, ClassIndeterminate, "went away"},
		"forged verdict then reproduced": {[]func(*testpilot.PreparedCase) (*testpilotspb.Run, *testpilotspb.Verdict, error){forged, nil}, []Class{ClassIndeterminate, ClassReproduced}, ClassIndeterminate, "offline replay"},
	} {
		t.Run(name, func(t *testing.T) {
			binder := &scriptedBinder{prepare: prepare, script: probe.script}
			reruns, err := Rerun(t.Context(), binder, subject.Target())
			require.NoError(t, err)
			require.Equal(t, probe.pair, reruns.Class)
			require.Len(t, reruns.Attempts, len(probe.classes))
			for index, class := range probe.classes {
				require.Equal(t, class, reruns.Attempts[index].Class)
			}
			details := reruns.Attempts[0].Detail + " " + reruns.Attempts[1].Detail
			require.Contains(t, details, probe.detail)
			require.Equal(t, Attempts, binder.binds, "every attempt binds fresh")
			require.False(t, binder.open, "every binding is released")
		})
	}

	// A violated Run whose key is another's: the subject compared to is one whose key differs
	// from what the Contract derives, and a fresh Run reproduces the Contract's key, not it.
	other := *subject
	other.Key = ViolationKey{Rules: []RuleKey{{Rule: "elsewhere", Terminal: subject.Key.Rules[0].Terminal, Evidence: subject.Key.Rules[0].Evidence}}}
	binder := &scriptedBinder{prepare: prepare}
	reruns, err := Rerun(t.Context(), binder, other.Target())
	require.NoError(t, err)
	require.Equal(t, ClassNotReproduced, reruns.Class)
	require.Contains(t, reruns.Attempts[0].Detail, "otherwise")
	require.True(t, reruns.Attempts[0].Key.Equal(subject.Key), "the attempt carries the key its Run derived")
	require.True(t, reruns.Key.Equal(other.Key), "the pair names the key it compared")
}

// A binder that prepares the subject under another identity, fails to bind or fails to release is
// an error, never a class: preparation was decided at admission.
func TestRerunRefusesAnotherIdentityAndAFailedRelease(t *testing.T) {
	subject, prepare := admittedSubject(t)
	drifted := &scriptedBinder{prepare: prepare, identity: func(identity testpilot.DriverIdentity) testpilot.DriverIdentity {
		identity.Bindings = "0000"
		return identity
	}}
	_, err := Rerun(t.Context(), drifted, subject.Target())
	require.ErrorContains(t, err, "not the target's")
	require.Equal(t, []string{"bind", "release"}, drifted.events, "released, never run")

	unreleased := &scriptedBinder{prepare: prepare, release: errors.New("namespace still held")}
	_, err = Rerun(t.Context(), unreleased, subject.Target())
	require.ErrorContains(t, err, "namespace still held")
	require.Equal(t, 1, unreleased.binds, "the next attempt never binds after a failed release")

	rejecting := &scriptedBinder{prepare: func(string, *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		return nil, errors.New("connection refused")
	}}
	_, err = Rerun(t.Context(), rejecting, subject.Target())
	require.ErrorContains(t, err, "connection refused")

	_, err = Rerun(t.Context(), nil, subject.Target())
	require.Error(t, err)
}

// The classifier's value carries each Run's class, the pair's class and the key it compared, and
// no field about history replay, which has no type here.
func TestRerunValueCarriesNoHistoryReplay(t *testing.T) {
	fields := func(value any) []string {
		var names []string
		kind := reflect.TypeOf(value)
		for index := range kind.NumField() {
			names = append(names, kind.Field(index).Name)
		}
		return names
	}
	require.Equal(t, []string{"Key", "Attempts", "Class"}, fields(Reruns{}))
	require.Equal(t, []string{"Class", "Detail", "Key", "Run", "Verdict", "Identity"}, fields(Attempt{}))
}
