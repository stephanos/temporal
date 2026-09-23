package replay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/casefile"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const profileName = "umpire-replay.test"

// The corpus's violated Case, run once through the scripted Driver, is a subject: its Run is in the
// admissible violated form, replays offline to its own Verdict, and its key names the rule, its
// terminal state and the violating event's evidence in Definition IDs.
func TestAdmitAcceptsAViolatedRunAndDerivesItsKey(t *testing.T) {
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	driver, run, recorded := recordedRunOf(t, prepare, profileName, caseBytes)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())

	subject, err := Admit(t.Context(), caseBytes, recorded, prepare)
	require.NoError(t, err)
	compact, err := casefile.Canonical(caseBytes)
	require.NoError(t, err)
	require.Equal(t, compact, subject.Canonical)
	require.Len(t, subject.Identity, 64)
	require.Equal(t, driver, subject.Driver)
	require.Equal(t, "temporal.case.conformance.violated", subject.Case.GetCaseId())
	require.Equal(t, run.GetRunId(), subject.Run.GetRunId())
	require.NotNil(t, subject.Replay)
	require.Len(t, subject.Replay.Violations, 1)
	require.Equal(t, ViolationKey{Rules: []RuleKey{{Rule: "result", Terminal: "terminal", Evidence: []string{}}}}, subject.Key)
	require.Equal(t, "result@terminal[]", subject.Key.String())
	require.NotContains(t, subject.Key.String(), subject.Identity, "the Case identity is beside the key, never in it")

	// The compact form, with or without its newline, admits the same subject with the same identity.
	for _, form := range [][]byte{compact, append(append([]byte(nil), compact...), '\n')} {
		again, err := Admit(t.Context(), form, recorded, prepare)
		require.NoError(t, err)
		require.Equal(t, subject.Identity, again.Identity)
		require.True(t, subject.Key.Equal(again.Key))
	}
}

// Per-Run identities and times never enter the key; a different violated rule set, terminal state
// or violating evidence does; a Case-local renaming does not.
func TestKeyReadsDefinitionIDsAndNothingPerRun(t *testing.T) {
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	_, first, _ := recordedRunOf(t, prepare, profileName, caseBytes)
	_, second, _ := recordedRunOf(t, prepare, profileName, caseBytes)
	require.NotEqual(t, first.GetRunId(), second.GetRunId())
	source, err := testpilot.DecodeCaseProtoJSON(caseBytes)
	require.NoError(t, err)
	prepared, err := prepare(profileName, source)
	require.NoError(t, err)
	keyOf := func(run *testpilotspb.Run) ViolationKey {
		verdict, evaluation, err := prepared.Evaluate(t.Context(), run)
		require.NoError(t, err)
		return KeyOf(source, verdict, evaluation)
	}
	require.True(t, keyOf(first).Equal(keyOf(second)), "two Runs of one Case share the key")

	// Per-Run transport values: another run id, later times, other activation and instruction ids.
	shifted := proto.CloneOf(first)
	shifted.RunId = "another-run"
	for index, event := range shifted.GetEvents() {
		if index > 0 {
			event.ElapsedMilliseconds += 1000
		}
		if event.GetCoordinates() != nil {
			event.Coordinates.ActivationId = "another-activation"
			event.Coordinates.InstructionId = "another-instruction"
		}
	}
	require.True(t, keyOf(first).Equal(keyOf(shifted)))

	// A Case-local renaming: the rule and its states renamed consistently through provenance rows
	// that map the new local names back to the original Definition IDs.
	renamed := proto.CloneOf(source)
	renamed.Contract.Rules[0].RuleId = "r"
	for _, state := range renamed.Contract.Rules[0].States {
		state.StateId = "s." + state.StateId
	}
	for _, transition := range renamed.Contract.Rules[0].Transitions {
		transition.SourceStateId = "s." + transition.SourceStateId
		transition.TargetStateId = "s." + transition.TargetStateId
	}
	renamed.Contract.Rules[0].InitialStateId = "s.pending"
	if renamed.Provenance == nil {
		renamed.Provenance = &testpilotspb.CaseProvenance{}
	}
	renamed.Provenance.LocalNames = append(renamed.Provenance.LocalNames,
		&testpilotspb.LocalName{LocalName: "r", DefinitionId: "result"},
		&testpilotspb.LocalName{LocalName: "s.terminal", DefinitionId: "terminal"},
		&testpilotspb.LocalName{LocalName: "s.pending", DefinitionId: "pending"})
	renamedPrepared, err := prepare(profileName, renamed)
	require.NoError(t, err)
	renamedRun, renamedVerdict, err := renamedPrepared.Run(t.Context(), &scriptedDriver{identity: renamedPrepared.Identity()})
	require.NoError(t, err)
	require.Equal(t, "r", renamedVerdict.GetRules()[0].GetRuleId(), "the Verdict speaks local names")
	replayed, evaluation, err := renamedPrepared.Evaluate(t.Context(), renamedRun)
	require.NoError(t, err)
	require.True(t, keyOf(first).Equal(KeyOf(renamed, replayed, evaluation)), "a renamed Case keeps the key")

	// Another terminal state is another violation.
	other := proto.CloneOf(source)
	other.Contract.Rules[0].States[1].StateId = "elsewhere"
	other.Contract.Rules[0].Transitions[0].TargetStateId = "elsewhere"
	otherPrepared, err := prepare(profileName, other)
	require.NoError(t, err)
	otherRun, _, err := otherPrepared.Run(t.Context(), &scriptedDriver{identity: otherPrepared.Identity()})
	require.NoError(t, err)
	otherVerdict, otherEvaluation, err := otherPrepared.Evaluate(t.Context(), otherRun)
	require.NoError(t, err)
	require.False(t, keyOf(first).Equal(KeyOf(other, otherVerdict, otherEvaluation)))
	require.Equal(t, "result@elsewhere[]", KeyOf(other, otherVerdict, otherEvaluation).String())
}

// Every rejection happens before any target effect and names its reason.
func TestAdmitRejectsEachClassBeforeAnyTargetEffect(t *testing.T) {
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	driver, run, recorded := recordedRunOf(t, prepare, profileName, caseBytes)
	satisfiedBytes := loadCorpusCase(t, "satisfied")
	_, _, satisfiedRecorded := recordedRunOf(t, prepare, profileName, satisfiedBytes)
	caseIdentity, err := CaseIdentity(caseBytes)
	require.NoError(t, err)
	edited := func(edit func(run *testpilotspb.Run)) []byte {
		copied := proto.CloneOf(run)
		edit(copied)
		document, err := EncodeRecordedRun(caseIdentity, driver, copied)
		require.NoError(t, err)
		return document
	}
	withIdentity := func(identity testpilot.DriverIdentity) []byte {
		document, err := EncodeRecordedRun(caseIdentity, identity, run)
		require.NoError(t, err)
		return document
	}
	legacy, err := json.Marshal(map[string]any{"identity": RecordedIdentity{Profile: driver.Profile, Catalog: driver.Catalog, Bindings: driver.Bindings}, "run": json.RawMessage(mustProtoJSON(t, run))})
	require.NoError(t, err)
	otherCase, err := EncodeRecordedRun(strings.Repeat("0", 64), driver, run)
	require.NoError(t, err)
	persisted, err := casefile.Persisted(caseBytes)
	require.NoError(t, err)
	for name, probe := range map[string]struct {
		caseInput []byte
		recorded  []byte
		reason    string
		detail    string
	}{
		"noncanonical whitespace": {[]byte(strings.Replace(string(persisted), "  ", "    ", 1)), recorded, ReasonNoncanonical, "canonical"},
		"not a Case":              {[]byte(`{"nonsense":1}`), recorded, ReasonNoncanonical, "does not decode"},
		"crossed Case":            {caseBytes, edited(func(r *testpilotspb.Run) { r.CaseId = "temporal.case.other" }), ReasonCrossed, "names Case"},
		"another Case's record":   {caseBytes, otherCase, ReasonCrossed, "recorded from Case"},
		"a record naming no Case": {caseBytes, legacy, ReasonIncompatible, "names no Case"},
		"crossed Program":         {caseBytes, edited(func(r *testpilotspb.Run) { r.ProgramId = "other.program" }), ReasonCrossed, "names Program"},
		"stale catalog":           {caseBytes, withIdentity(testpilot.DriverIdentity{Profile: driver.Profile, Catalog: "other-catalog", Bindings: driver.Bindings}), ReasonStale, "recorded under"},
		"stale bindings":          {caseBytes, withIdentity(testpilot.DriverIdentity{Profile: driver.Profile, Catalog: driver.Catalog, Bindings: "other-bindings"}), ReasonStale, "recorded under"},
		"incomplete":              {caseBytes, edited(func(r *testpilotspb.Run) { r.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE }), ReasonIncomplete, "Incomplete"},
		"completed beside satisfied rules": {caseBytes, edited(func(r *testpilotspb.Run) {
			r.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED
			r.Verdict.Status = testpilotspb.VERDICT_STATUS_SATISFIED
		}), ReasonNonViolated, "not violated"},
		"unclosed cleanup":           {caseBytes, edited(func(r *testpilotspb.Run) { r.Cleanup.Status = testpilotspb.CLEANUP_STATUS_TIMED_OUT }), ReasonIncomplete, "cleanup"},
		"completed beside violation": {caseBytes, edited(func(r *testpilotspb.Run) { r.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED }), ReasonMalformed, "never produces"},
		"non-violated":               {satisfiedBytes, satisfiedRecorded, ReasonNonViolated, "not violated"},
		"unsupported":                {caseBytes, edited(func(r *testpilotspb.Run) { r.Verdict.Rules[0].SupportingEventSequences = []int64{99} }), ReasonUnsupported, "does not carry"},
		"duplicate":                  {caseBytes, edited(func(r *testpilotspb.Run) { r.Verdict.SupportingEventSequences = []int64{4, 4} }), ReasonDuplicate, "twice"},
		"replay disagrees":           {caseBytes, edited(func(r *testpilotspb.Run) { r.Verdict.Rules[0].TerminalStateId = "elsewhere" }), ReasonReplay, "replays to"},
		"no events":                  {caseBytes, edited(func(r *testpilotspb.Run) { r.Events = nil }), ReasonIncomplete, "0 events"},
	} {
		t.Run(name, func(t *testing.T) {
			subject, err := Admit(t.Context(), probe.caseInput, probe.recorded, prepare)
			require.Nil(t, subject)
			rejection, ok := IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, probe.reason, rejection.Reason)
			require.Contains(t, rejection.Detail, probe.detail)
		})
	}
	// A Case that no longer prepares statically is stale; any other preparation error is not a
	// rejection of the subject.
	staticRejection := func(string, *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		return nil, fmt.Errorf("derive Profile: %w", &testpilot.PreparationError{Category: testpilot.PreparationUnknown, Path: "program", Detail: "gone"})
	}
	_, err = Admit(t.Context(), caseBytes, recorded, staticRejection)
	rejection, ok := IsRejection(err)
	require.True(t, ok, "not a rejection: %v", err)
	require.Equal(t, ReasonStale, rejection.Reason)
	require.Contains(t, rejection.Detail, "no longer prepares")
	toolingFailure := func(string, *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		return nil, errors.New("the handler queue flag disagrees")
	}
	_, err = Admit(t.Context(), caseBytes, recorded, toolingFailure)
	require.Error(t, err)
	_, ok = IsRejection(err)
	require.False(t, ok)
	// A recorded Run that is not one is malformed; an unknown field is another protocol.
	_, err = Admit(t.Context(), caseBytes, []byte(`{"identity":{},"run":{},"extra":1}`), prepare)
	rejection, ok = IsRejection(err)
	require.True(t, ok)
	require.Equal(t, ReasonMalformed, rejection.Reason)
}

// The evidence part and the rule set of the key, as the evaluation names them: another observation
// id, another correlated kind or another violated rule is another violation, and each name resolves
// through the Case's local names.
func TestKeyReadsTheViolatingEvidenceAndTheRuleSet(t *testing.T) {
	source := &testpilotspb.Case{
		Contract: &testpilotspb.Contract{Correlated: &testpilotspb.CorrelatedContract{Rules: []*testpilotspb.CorrelatedRule{{RuleId: "c"}}}},
		Provenance: &testpilotspb.CaseProvenance{LocalNames: []*testpilotspb.LocalName{
			{LocalName: "r", DefinitionId: "umpire.rule.result"},
			{LocalName: "c", DefinitionId: "umpire.rule.correlated"},
			{LocalName: "failed", DefinitionId: "umpire.evidence.failed"},
			{LocalName: "obs.a", DefinitionId: "umpire.observation.a"},
		}},
	}
	violated := func(rules ...string) *testpilotspb.Verdict {
		verdict := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}
		for _, rule := range rules {
			verdict.Rules = append(verdict.Rules, &testpilotspb.RuleVerdict{RuleId: rule, Status: testpilotspb.RULE_VERDICT_STATUS_VIOLATED, TerminalStateId: "bad"})
		}
		return verdict
	}
	monitorA := KeyOf(source, violated("r"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "r", Sequence: 4, ObservationIDs: []string{"obs.a"}}}})
	require.Equal(t, "umpire.rule.result@bad[umpire.observation.a]", monitorA.String())
	monitorB := KeyOf(source, violated("r"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "r", Sequence: 9, ObservationIDs: []string{"obs.b"}}}})
	require.Equal(t, "umpire.rule.result@bad[obs.b]", monitorB.String(), "a name with no row is its own")
	require.False(t, monitorA.Equal(monitorB), "another observation is another violation")
	deadline := KeyOf(source, violated("r"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "r", Sequence: 4}}})
	require.Equal(t, "umpire.rule.result@bad[]", deadline.String())
	correlated := KeyOf(source, violated("c"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "c", Sequence: 6, CorrelatedKind: "failed"}}})
	require.Equal(t, "umpire.rule.correlated@correlated.violated[umpire.evidence.failed]", correlated.String())
	otherKind := KeyOf(source, violated("c"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "c", Sequence: 6, CorrelatedKind: "completed"}}})
	require.False(t, correlated.Equal(otherKind), "another kind is another violation")
	atClosure := KeyOf(source, violated("c"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "c"}}})
	require.Equal(t, "umpire.rule.correlated@correlated.violated[]", atClosure.String())
	both := KeyOf(source, violated("r", "c"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "c", CorrelatedKind: "failed"}, {RuleID: "r", ObservationIDs: []string{"obs.a"}}}})
	require.Equal(t, "umpire.rule.correlated@correlated.violated[umpire.evidence.failed];umpire.rule.result@bad[umpire.observation.a]", both.String())
	require.False(t, both.Equal(monitorA), "another rule set is another violation")
	require.True(t, both.Equal(KeyOf(source, violated("c", "r"), &testpilot.Evaluation{Violations: []testpilot.RuleViolation{{RuleID: "r", ObservationIDs: []string{"obs.a"}}, {RuleID: "c", CorrelatedKind: "failed"}}})), "order is not identity")
}

func TestRecordedRunRoundTripsAndNeverReplacesAFile(t *testing.T) {
	prepare := preparer(t)
	driver, run, recorded := recordedRunOf(t, prepare, profileName, loadCorpusCase(t, "violated"))
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorded, &document))
	require.Equal(t, []string{"case", "identity", "run"}, sortedKeys(document))
	caseBytes := loadCorpusCase(t, "violated")
	caseIdentity, err := CaseIdentity(caseBytes)
	require.NoError(t, err)
	decoded, err := DecodeRecordedRun(recorded)
	require.NoError(t, err)
	require.Equal(t, caseIdentity, decoded.Case)
	require.Equal(t, driver, decoded.Driver)
	require.True(t, proto.Equal(run, decoded.Run))
	expected, err := protojson.Marshal(run)
	require.NoError(t, err)
	var compactExpected, compactRecorded map[string]any
	require.NoError(t, json.Unmarshal(expected, &compactExpected))
	require.NoError(t, json.Unmarshal(document["run"], &compactRecorded))
	require.Equal(t, compactExpected, compactRecorded)

	path := t.TempDir() + "/run.json"
	require.NoError(t, WriteRecordedRun(path, caseBytes, driver, run))
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(recorded), string(written), "writing is encoding under the Case's identity")
	require.ErrorContains(t, WriteRecordedRun(path, caseBytes, driver, run), "exist")
	require.ErrorContains(t, WriteRecordedRun(t.TempDir()+"/run.json", []byte(" {}"), driver, run), "no identity")
	_, err = DecodeRecordedRun(nil)
	require.Error(t, err)
	_, err = DecodeRecordedRun(append(append([]byte(nil), recorded...), recorded...))
	require.ErrorContains(t, err, "after the document")
	_, err = DecodeRecordedRun(append(append([]byte(nil), recorded...), []byte("trailing")...))
	require.ErrorContains(t, err, "after the document")
	// Go's decoder matches keys without case and keeps the last of a repeated key; the record
	// refuses both, so two documents never decode to one record.
	_, err = DecodeRecordedRun([]byte(strings.Replace(string(recorded), `"case"`, `"Case"`, 1)))
	require.ErrorContains(t, err, `unknown field "Case"`)
	_, err = DecodeRecordedRun([]byte(strings.Replace(string(recorded), `"profile"`, `"PROFILE"`, 1)))
	require.ErrorContains(t, err, `unknown field "PROFILE"`)
	_, err = DecodeRecordedRun([]byte(strings.Replace(string(recorded), `{"case":`, `{"case":"x","case":`, 1)))
	require.ErrorContains(t, err, `"case" appears twice`)
	var withoutCase map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorded, &withoutCase))
	delete(withoutCase, "case")
	legacy, err := json.Marshal(withoutCase)
	require.NoError(t, err)
	_, err = DecodeRecordedRun(legacy)
	require.ErrorIs(t, err, ErrRecordNamesNoCase)
	_, err = DecodeRecordedRun([]byte(strings.Replace(string(recorded), caseIdentity, "not-a-digest", 1)))
	require.ErrorContains(t, err, "not a hex SHA-256")
}

func TestClassifyAndThePairRule(t *testing.T) {
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	_, run, recorded := recordedRunOf(t, prepare, profileName, caseBytes)
	subject, err := Admit(t.Context(), caseBytes, recorded, prepare)
	require.NoError(t, err)
	class, _ := Classify(subject.Key, run, run.GetVerdict(), subject.Key)
	require.Equal(t, ClassReproduced, class)
	other := ViolationKey{Rules: []RuleKey{{Rule: "result", Terminal: "elsewhere", Evidence: []string{}}}}
	class, detail := Classify(subject.Key, run, run.GetVerdict(), other)
	require.Equal(t, ClassNotReproduced, class)
	require.Contains(t, detail, "otherwise")
	opened := []*testpilotspb.RunEvent{{Sequence: 1, Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}}
	satisfied := &testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	class, _ = Classify(subject.Key, satisfied, &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, ViolationKey{})
	require.Equal(t, ClassNotReproduced, class)
	incomplete := &testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	class, _ = Classify(subject.Key, incomplete, &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE}, ViolationKey{})
	require.Equal(t, ClassIndeterminate, class)
	unclosed := &testpilotspb.Run{RunId: "r", Events: opened, Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_FAILED}}
	class, _ = Classify(subject.Key, unclosed, &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, ViolationKey{})
	require.Equal(t, ClassIndeterminate, class)

	require.Equal(t, ClassReproduced, ClassifyPair(ClassReproduced, ClassReproduced))
	require.Equal(t, ClassNotReproduced, ClassifyPair(ClassReproduced, ClassNotReproduced))
	require.Equal(t, ClassNotReproduced, ClassifyPair(ClassIndeterminate, ClassNotReproduced))
	require.Equal(t, ClassIndeterminate, ClassifyPair(ClassReproduced, ClassIndeterminate))
	require.Equal(t, ClassIndeterminate, ClassifyPair(ClassIndeterminate, ClassIndeterminate))
}

func sortedKeys(document map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func mustProtoJSON(t testing.TB, run *testpilotspb.Run) []byte {
	t.Helper()
	encoded, err := protojson.Marshal(run)
	require.NoError(t, err)
	return encoded
}
