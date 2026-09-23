package evaluation

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/history/v1" // the control Run's events carry history events as Any values
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/internal/recordedrun"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	controlCasePath = "../../../tests/testcore/testpilot/testdata/nexusCallerControl-forgedCompletion-case.json"
	controlRunPath  = "../replay/testdata/nexusCallerControl-forgedCompletion-run.json"
	// controlCatalog is the catalog the control Run was recorded under; the tree's static catalog
	// is the same, which TestTheControlRecordIsCurrent pins.
	controlCatalog = "3e6992900900436a60362bb7837323e32a40ba2d420a90e0624b4ef30944a28c"
)

type control struct {
	caseBytes []byte
	source    *testpilotspb.Case
	decoded   recordedrun.Decoded
	recorded  []byte
}

func loadControl(t *testing.T) control {
	t.Helper()
	caseBytes, err := os.ReadFile(controlCasePath)
	require.NoError(t, err)
	recorded, err := os.ReadFile(controlRunPath)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(caseBytes)
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(recorded)
	require.NoError(t, err)
	return control{caseBytes: caseBytes, source: source, decoded: decoded, recorded: recorded}
}

// compactCase is a Case in compact ProtoJSON: canonical, with its own identity.
func compactCase(t *testing.T, source *testpilotspb.Case) []byte {
	t.Helper()
	encoded, err := protojson.Marshal(source)
	require.NoError(t, err)
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, encoded))
	return compact.Bytes()
}

// pair edits the control Case and its Run and re-records the Run from the edited Case's bytes, so
// the only thing wrong with the pair is the edit.
func (c control) pair(t *testing.T, editCase func(*testpilotspb.Case), editRun func(*testpilotspb.Run)) (caseBytes, recorded []byte) {
	t.Helper()
	caseBytes = c.caseBytes
	if editCase != nil {
		source := proto.CloneOf(c.source)
		editCase(source)
		caseBytes = compactCase(t, source)
	}
	identity, err := recordedrun.CaseIdentity(caseBytes)
	require.NoError(t, err)
	run := proto.CloneOf(c.decoded.Run)
	if editRun != nil {
		editRun(run)
	}
	recorded, err = recordedrun.Encode(identity, c.decoded.Driver, run)
	require.NoError(t, err)
	return caseBytes, recorded
}

func satisfy(run *testpilotspb.Run) {
	run.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED
	run.Verdict.Status = testpilotspb.VERDICT_STATUS_SATISFIED
	for _, rule := range run.Verdict.Rules {
		rule.Status = testpilotspb.RULE_VERDICT_STATUS_SATISFIED
	}
}

// The pinned control record admits as recorded: a correlated-only Case, whose rules are all in the
// correlated Contract, and the Run's identities bound as they are.
func TestAdmitTheControlRecord(t *testing.T) {
	c := loadControl(t)
	require.Empty(t, c.source.GetContract().GetRules(), "the control is correlated-only")
	subject, err := Admit(c.caseBytes, c.recorded, controlCatalog)
	require.NoError(t, err)
	caseIdentity, err := recordedrun.CaseIdentity(c.caseBytes)
	require.NoError(t, err)
	require.Equal(t, caseIdentity, subject.CaseIdentity)
	require.Equal(t, recordedrun.Digest(c.recorded), subject.RunIdentity)
	require.Equal(t, "temporal.case.nexusCallerControl.forgedCompletion", subject.CaseID)
	require.Equal(t, c.source.GetProgram().GetProgramId(), subject.ProgramID)
	require.Equal(t, c.source.GetContract().GetContractId(), subject.ContractID)
	require.Equal(t, c.decoded.Driver, subject.Driver)
	require.Equal(t, c.decoded.Run.GetRunId(), subject.RunID)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, subject.Disposition)
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, subject.Cleanup)
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, subject.Verdict.GetStatus())
	require.Equal(t, AdmissionCaps(), subject.Caps)
}

// Admission compares the recorded catalog with the one it is given; the command gives the tree's,
// which is the one the control was recorded under.
func TestTheControlRecordIsCurrent(t *testing.T) {
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	require.Equal(t, controlCatalog, catalog.Identity())
}

// Negative subjects that are well formed are admitted: a violated Run whose cleanup failed, a
// satisfied Run and an inconclusive one are all assessable.
func TestAdmitAdmitsEveryWellFormedClosedRun(t *testing.T) {
	c := loadControl(t)
	for name, edit := range map[string]func(*testpilotspb.Run){
		"violated, cleanup failed":    func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_FAILED },
		"violated, cleanup timed out": func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_TIMED_OUT },
		"satisfied":                   satisfy,
		"inconclusive, incomplete": func(run *testpilotspb.Run) {
			satisfy(run)
			run.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE
			run.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
		},
	} {
		t.Run(name, func(t *testing.T) {
			caseBytes, recorded := c.pair(t, nil, edit)
			_, err := Admit(caseBytes, recorded, controlCatalog)
			require.NoError(t, err)
		})
	}
}

// Every rejection class, before any assessment, in the order the checks run.
func TestAdmitRejectsEachClass(t *testing.T) {
	c := loadControl(t)
	respaced := func(document []byte) []byte {
		return []byte(strings.Replace(string(document), `":`, `": `, 1))
	}
	legacy := func() []byte {
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(c.recorded, &fields))
		delete(fields, "case")
		document, err := json.Marshal(fields)
		require.NoError(t, err)
		return append(document, '\n')
	}()
	manyEvents := func(run *testpilotspb.Run) {
		for len(run.Events) <= AdmissionCaps().RunEvents {
			run.Events = append(run.Events, &testpilotspb.RunEvent{Sequence: int64(len(run.Events) + 1)})
		}
	}
	manyCase, manyRecorded := c.pair(t, nil, manyEvents)
	regenerated, _ := c.pair(t, func(source *testpilotspb.Case) { source.Provenance.ProducerVersion += "-regenerated" }, nil)
	anotherCase, _ := c.pair(t, func(source *testpilotspb.Case) { source.CaseId = "temporal.case.other" }, nil)
	for name, probe := range map[string]struct {
		caseBytes []byte
		recorded  []byte
		reason    string
		detail    string
	}{
		"an oversized Case":           {bytes.Repeat([]byte(" "), AdmissionCaps().CaseBytes+1), c.recorded, ReasonOversized, "Case is"},
		"a re-spaced Case":            {respaced(compactCase(t, c.source)), c.recorded, ReasonNoncanonical, "canonical"},
		"a Case that is not JSON":     {[]byte("{"), c.recorded, ReasonNoncanonical, ""},
		"a Case that does not decode": {[]byte(`{"nonsense":1}`), c.recorded, ReasonMalformed, "does not decode"},
		"a Case of another version": {compactCase(t, func() *testpilotspb.Case {
			source := proto.CloneOf(c.source)
			source.Version.Minor = 1
			return source
		}()), c.recorded, ReasonIncompatible, "1.1"},
		"an oversized record":       {c.caseBytes, bytes.Repeat([]byte(" "), AdmissionCaps().RunBytes+1), ReasonOversized, "recorded Run is"},
		"a record that is not JSON": {c.caseBytes, []byte("{"), ReasonMalformed, ""},
		"a repeated record key":     {c.caseBytes, []byte(strings.Replace(string(c.recorded), `{"case":`, `{"case":"x","case":`, 1)), ReasonMalformed, "twice"},
		"a case-folded record key":  {c.caseBytes, []byte(strings.Replace(string(c.recorded), `"identity":`, `"Identity":`, 1)), ReasonMalformed, "unknown field"},
		"too many events":           {manyCase, manyRecorded, ReasonOversized, "events"},
		"a record naming no Case":   {c.caseBytes, legacy, ReasonIncompatible, "names no Case"},
		"a re-spaced record":        {c.caseBytes, respaced(c.recorded), ReasonNoncanonical, "form its writer produces"},
	} {
		t.Run(name, func(t *testing.T) {
			subject, err := Admit(probe.caseBytes, probe.recorded, controlCatalog)
			require.Nil(t, subject)
			rejection, ok := IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, probe.reason, rejection.Reason, rejection.Detail)
			require.Contains(t, rejection.Detail, probe.detail)
		})
	}

	pairs := map[string]struct {
		editCase func(*testpilotspb.Case)
		editRun  func(*testpilotspb.Run)
		reason   string
		detail   string
	}{
		"no Run ID":              {nil, func(run *testpilotspb.Run) { run.RunId = "" }, ReasonOpen, "no Run ID"},
		"no events":              {nil, func(run *testpilotspb.Run) { run.Events = nil; run.Verdict.SupportingEventSequences = nil }, ReasonOpen, "no events"},
		"no disposition":         {nil, func(run *testpilotspb.Run) { run.Disposition = testpilotspb.RUN_DISPOSITION_UNSPECIFIED }, ReasonOpen, "disposition"},
		"no cleanup":             {nil, func(run *testpilotspb.Run) { run.Cleanup = nil }, ReasonOpen, "cleanup"},
		"an unspecified cleanup": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = testpilotspb.CLEANUP_STATUS_UNSPECIFIED }, ReasonOpen, "cleanup"},
		"no Verdict":             {nil, func(run *testpilotspb.Run) { run.Verdict = nil }, ReasonOpen, "no Verdict"},
		"another Case ID":        {nil, func(run *testpilotspb.Run) { run.CaseId = "temporal.case.other" }, ReasonCrossed, "names Case"},
		"another Program ID":     {nil, func(run *testpilotspb.Run) { run.ProgramId = "other.program" }, ReasonCrossed, "names Program"},
		"a rule missing":         {nil, func(run *testpilotspb.Run) { run.Verdict.Rules = run.Verdict.Rules[:1] }, ReasonCrossed, "the Contract's are"},
		"a rule of another Contract": {nil, func(run *testpilotspb.Run) {
			run.Verdict.Rules = append(run.Verdict.Rules, &testpilotspb.RuleVerdict{RuleId: "elsewhere", Status: testpilotspb.RULE_VERDICT_STATUS_SATISFIED})
		}, ReasonCrossed, "the Contract's are"},
		"a rule named twice": {nil, func(run *testpilotspb.Run) {
			run.Verdict.Rules[1].RuleId = run.Verdict.Rules[0].RuleId
		}, ReasonCrossed, "twice"},
		"an undeclared disposition":    {nil, func(run *testpilotspb.Run) { run.Disposition = 99 }, ReasonMalformed, "disposition 99"},
		"an undeclared cleanup status": {nil, func(run *testpilotspb.Run) { run.Cleanup.Status = 99 }, ReasonMalformed, "cleanup status 99"},
		"an undeclared Verdict status": {nil, func(run *testpilotspb.Run) { run.Verdict.Status = 99 }, ReasonMalformed, "Verdict's status 99"},
		"an undeclared rule status":    {nil, func(run *testpilotspb.Run) { run.Verdict.Rules[0].Status = 99 }, ReasonMalformed, "status 99"},
		"an unknown supporting event":  {nil, func(run *testpilotspb.Run) { run.Verdict.Rules[0].SupportingEventSequences = []int64{99999} }, ReasonInconsistent, "does not carry"},
		"a supporting event twice":     {nil, func(run *testpilotspb.Run) { run.Verdict.SupportingEventSequences = []int64{9, 9} }, ReasonInconsistent, "twice"},
		"violated but completed":       {nil, func(run *testpilotspb.Run) { run.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED }, ReasonInconsistent, "disposition"},
		"stopped but satisfied": {nil, func(run *testpilotspb.Run) {
			satisfy(run)
			run.Disposition = testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR
		}, ReasonInconsistent, "disposition"},
		"satisfied beside a violation": {nil, func(run *testpilotspb.Run) { run.Verdict.Status = testpilotspb.VERDICT_STATUS_SATISFIED }, ReasonInconsistent, "violated: true"},
		"a pending rule":               {nil, func(run *testpilotspb.Run) { run.Verdict.Rules[0].Status = testpilotspb.RULE_VERDICT_STATUS_PENDING }, ReasonInconsistent, "Pending"},
		"an unspecified status":        {nil, func(run *testpilotspb.Run) { run.Verdict.Status = testpilotspb.VERDICT_STATUS_UNSPECIFIED }, ReasonInconsistent, "unspecified"},
		"an unspecified Known Gap kind": {func(source *testpilotspb.Case) {
			source.Provenance.KnownGaps = []*testpilotspb.KnownGap{{Code: "umpire.gap.x"}}
		}, nil, ReasonMalformed, "not a declared kind"},
		"an undeclared Known Gap kind": {func(source *testpilotspb.Case) {
			source.Provenance.KnownGaps = []*testpilotspb.KnownGap{{Kind: 99, Code: "umpire.gap.x"}}
		}, nil, ReasonMalformed, "kind 99"},
	}
	for name, probe := range pairs {
		t.Run(name, func(t *testing.T) {
			caseBytes, recorded := c.pair(t, probe.editCase, probe.editRun)
			_, err := Admit(caseBytes, recorded, controlCatalog)
			rejection, ok := IsRejection(err)
			require.True(t, ok, "not a rejection: %v", err)
			require.Equal(t, probe.reason, rejection.Reason, rejection.Detail)
			require.Contains(t, rejection.Detail, probe.detail)
		})
	}

	// IDs are names, not hashes: a Case regenerated under the same IDs does not inherit the older
	// Run, and neither does another Case.
	for name, caseBytes := range map[string][]byte{"regenerated": regenerated, "another": anotherCase} {
		_, err := Admit(caseBytes, c.recorded, controlCatalog)
		rejection, ok := IsRejection(err)
		require.True(t, ok, name)
		require.Equal(t, ReasonCrossed, rejection.Reason, name)
		require.Contains(t, rejection.Detail, "recorded from Case", name)
	}

	_, err := Admit(c.caseBytes, c.recorded, "another-catalog")
	rejection, ok := IsRejection(err)
	require.True(t, ok)
	require.Equal(t, ReasonStale, rejection.Reason)
}

// Qualification cannot execute: the package links neither the replay bridge and its rerun
// environment, nor a deployment binding, a Driver or a campaign. The admission and recorded-Run
// code it shares with replay live in the leaf package it does link.
func TestEvaluationLinksNothingThatExecutes(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("the Go toolchain is not on PATH: %v", err)
	}
	listed, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err)
	dependencies := strings.Split(strings.TrimSpace(string(listed)), "\n")
	require.Contains(t, dependencies, "go.temporal.io/server/tools/umpire/internal/recordedrun")
	for _, forbidden := range []string{
		"go.temporal.io/server/tools/umpire/replay",
		"go.temporal.io/server/tools/umpire/campaign",
		"go.temporal.io/server/tools/umpire/binding",
		"go.temporal.io/server/common/testing/testpilot/temporal",
	} {
		require.NotContains(t, dependencies, forbidden)
	}
}
