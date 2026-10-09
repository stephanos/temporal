package producer_test

// What a realization declares of its evidence beyond a kind and a source reaches the Case: the fields
// a kind keeps, the kinds an exhaustive source reports whatever the path, the Run's own record and a
// read of one message as sources, and the steps of a path one kind confirms. The Model here is a job
// a worker takes, fails once and finishes, written out as a table so every expectation below is read
// off these few rows.

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
)

const (
	jobFamily   = "fixture.job"
	jobEvidence = jobFamily + ".evidence."
	jobSource   = jobFamily + ".source."
)

// jobTable is the job's machine. A submitted job is listed. A worker takes it, which records that it
// was taken and its attempt count; a failed attempt lists it again with the count; a finished one
// closes it. A job may also be dropped while it is queued, which records that it was dropped. A
// running job may be checked, which records nothing, and closed, which finishes it or drops it.
func jobTable(t *testing.T) *interp.Table {
	t.Helper()
	model := jobModel("once", jobOnce...)
	require.NoError(t, ir.Validate(model))
	built, err := interp.Build(model)
	require.NoError(t, err)
	require.Len(t, built, 1)
	return built["job"].Table
}

// jobQuery finds the job finished by the last step of a path of these classes.
func jobQuery(t *testing.T, name string, actions ...string) *check.Query {
	t.Helper()
	model := jobModel(name, actions...)
	table := jobTable(t)
	realizer, err := check.NewRealizer(model, check.DefaultScope)
	require.NoError(t, err)
	require.Equal(t, table.IDs(), realizer.Machine("job").Table.IDs())
	query, err := realizer.Find(check.ClaimKey{Family: jobFamily, Owner: "job", Name: name})
	require.NoError(t, err)
	return query
}

var (
	jobOnce    = []string{"submit", "take", "finish"}
	jobRetried = []string{"submit", "take", "fail", "settle", "take", "finish"}
)

func projected(path string) *testpilotspb.Expression { return cp.Path(cp.ProjectedValue(), path) }

// attemptIs is the guard that takes the record of one numbered attempt.
func attemptIs(number int64) *testpilotspb.Expression {
	return cp.Equal(projected("activity_attempt.sdk_attempt"), cp.Literal(cp.SignedInteger(number)))
}

func attemptRecord(guard *testpilotspb.Expression) cp.Recorded {
	return cp.Recorded{RunEvent: &cp.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, EntrypointID: "controller",
		InstructionID: "submit-job", RunKeyed: true, Guard: guard}}
}

var attemptFields = []cp.EvidenceField{
	{ID: "attempt", Path: "activity_attempt.sdk_attempt", Type: testpilotspb.SCALAR_KIND_UINT64, Role: "attempt"},
	{ID: "delivery", Path: "activity_attempt.delivery_id", Type: testpilotspb.SCALAR_KIND_TEXT, Role: "delivery"},
}

func jobKind(records string, recorded cp.Recorded, key string) *cp.EvidenceSource {
	return &cp.EvidenceSource{EventKind: records, Recorded: recorded, OperationKeyPath: key, KindID: jobEvidence + records,
		SourceID: jobSource + records}
}

// jobSources is the job's evidence. The listing and the closing status are read back. That a worker
// took the job is the Run's record of its first attempt; the record of its second attempt is what
// shows the failed attempt listed the job again and a worker took it again, so that one kind confirms
// both of those steps, and each of the two takes has a kind of its own.
func jobSources() []*cp.EvidenceSource {
	taken := jobKind("taken", attemptRecord(attemptIs(1)), "")
	taken.Fields, taken.Confirms = attemptFields, []cp.Taking{{Key: "take", Occurrence: 1}}
	again := jobKind("count", attemptRecord(attemptIs(2)), "")
	again.Fields, again.Confirms = attemptFields, []cp.Taking{{Key: "fail", Occurrence: 1}, {Key: "take", Occurrence: 2}}
	return []*cp.EvidenceSource{
		jobKind("listed", cp.Recorded{Method: "/fixture.Jobs/List", Path: "jobs"}, "job_id"),
		taken, again,
		jobKind("finished", cp.Recorded{Method: "/fixture.Jobs/Describe", Path: "job", Single: true}, "job_id"),
		jobKind("wasDropped", cp.Recorded{HistoryAttributes: "job_dropped_event_attributes"}, "attributes<job_dropped_event_attributes>.job_id"),
	}
}

func jobRealization(t *testing.T, sources []*cp.EvidenceSource) *cp.Realization {
	t.Helper()
	submit := jobTable(t).ActionAtom("submit").ID
	return &cp.Realization{ProducerID: "fixture.job.testpilot", ProducerVersion: "1", ProjectionID: jobFamily + ".projection",
		ScopeField: jobFamily + ".scope.run", OperationKey: jobFamily + ".scope.job", CorrelatedObservation: "correlated-evidence",
		Sources: sources, ProjectionLimits: cp.ProjectionLimits{Events: 8, Buffered: 8, Keys: 2, Support: 16, Work: 1000, EventSize: 512},
		Actions: []cp.ActionBinding{{Action: submit, Key: "submit", InstructionID: "submit-job",
			Node: func(_ cp.Placement, id string) *testpilotspb.InstructionNode {
				return cp.Node(id, cp.InvokeRPC("jobs", "/fixture.Jobs/Submit", nil, nil))
			}}},
		Plan: cp.ProgramPlan{
			Observations: []*testpilotspb.Observation{cp.MessageObservation("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")},
			Entrypoints: []cp.EntrypointPlan{{
				Items: []cp.Item{cp.Actions{Classes: []string{submit}}, cp.Fixed{Node: func(_ cp.Placement, rules []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return cp.Node("history", cp.InvokeRPC("jobs", "/fixture.Jobs/History", nil, []*testpilotspb.ResponseRead{
						cp.ResponseRead("events[*]", testpilotspb.READ_CARDINALITY_EMIT_EACH, cp.EvidenceTarget("correlated-evidence", rules))}))
				}}},
				Activate: func(_ cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
					return &testpilotspb.Entrypoint{EntrypointId: "controller", Instructions: nodes,
						Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}
				}}}}}
}

func jobIdentity(name string) cp.Identity { return cp.IdentityFor("fixture.case", "jobs", name) }

func produced(t *testing.T, name string, sources []*cp.EvidenceSource, actions ...string) *testpilotspb.Case {
	t.Helper()
	c, err := cp.Produce(jobQuery(t, name, actions...), jobIdentity(name), jobRealization(t, sources), cp.Source{Path: "job.go", Provenance: "test"})
	require.NoError(t, err)
	return c
}

// local is the Case's own name for a Definition ID.
func local(c *testpilotspb.Case, id string) string {
	for _, n := range c.GetProvenance().GetLocalNames() {
		if n.GetDefinitionId() == id {
			return n.GetLocalName()
		}
	}
	return id
}

// confirmed is, by the kind's recorded name, the steps each projection rule of a Case confirms, each
// as its action and the state it reaches.
func confirmed(c *testpilotspb.Case) map[string][][2]string {
	out := map[string][][2]string{}
	for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
		steps := [][2]string{}
		for _, output := range rule.GetOutputs() {
			steps = append(steps, [2]string{output.GetAction().GetValue(), output.GetState().GetValue()})
		}
		out[rule.GetKind()] = steps
	}
	return out
}

// A path that takes a class twice is confirmed occurrence by occurrence. The retried job takes `take`
// twice: the first is the first attempt's record, and the second attempt's record confirms the failed
// attempt, the settling no evidence names, and the second take, in that order, as the outputs of its
// one projection rule. Every kind is claimed by one rule, each output is a row of the table from the
// state the output before it reaches, and the step that records nothing is still a Known Gap.
func TestAClassTakenTwiceIsConfirmedOccurrenceByOccurrence(t *testing.T) {
	c := produced(t, "retried", jobSources(), jobRetried...)
	kind := func(records string) string { return local(c, jobEvidence+records) }
	require.Equal(t, map[string][][2]string{
		kind("listed"):   {{"submit", "queued"}},
		kind("taken"):    {{"take", "running"}},
		kind("count"):    {{"fail", "waiting"}, {"settle", "queued"}, {"take", "running"}},
		kind("finished"): {{"finish", "done"}},
	}, confirmed(c))

	var gaps []string
	for _, gap := range c.GetProvenance().GetKnownGaps() {
		gaps = append(gaps, gap.GetSubject())
	}
	require.Equal(t, []string{jobTable(t).ActionAtom("settle").ID}, gaps)

	// The path that takes the class once is confirmed by the first attempt's kind alone, and the second
	// attempt's kind, which no step of it is named by, is not carried.
	once := produced(t, "once", jobSources(), jobOnce...)
	require.Equal(t, map[string][][2]string{
		local(once, jobEvidence+"listed"):   {{"submit", "queued"}},
		local(once, jobEvidence+"taken"):    {{"take", "running"}},
		local(once, jobEvidence+"finished"): {{"finish", "done"}},
	}, confirmed(once))
}

// What a realization declares of a kind's recorded data is what the Case's evidence declaration says:
// the Run's own record under its guard, at the instruction that records it and keyed by the Run; a
// read of one message; and the fields a kind keeps, each at its path in the Program and with its type,
// retained, in the Contract.
func TestACaseDeclaresTheSourceAndTheFieldsOfEachKind(t *testing.T) {
	c := produced(t, "retried", jobSources(), jobRetried...)
	scope := []*testpilotspb.NamedValue{{FieldId: local(c, jobFamily+".scope.run"), Value: cp.Text("jobs-retried")}}
	declared := func(records string, guard *testpilotspb.Expression) *testpilotspb.EvidenceDeclaration {
		return &testpilotspb.EvidenceDeclaration{EvidenceId: local(c, jobEvidence+records), EvidenceSource: local(c, jobSource+records), Scope: scope,
			Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
				Guard: guard, Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "submit-job"}, RunKeyed: true}},
			Fields: []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "attempt", Path: "activity_attempt.sdk_attempt"},
				{FieldId: "delivery", Path: "activity_attempt.delivery_id"}}}
	}
	protorequire.ProtoSliceEqual(t, []*testpilotspb.EvidenceDeclaration{
		{EvidenceId: local(c, jobEvidence+"listed"), EvidenceSource: local(c, jobSource+"listed"), Scope: scope, Operation: "job_id",
			Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: "/fixture.Jobs/List", Path: "jobs"}}},
		declared("taken", attemptIs(1)),
		declared("count", attemptIs(2)),
		{EvidenceId: local(c, jobEvidence+"finished"), EvidenceSource: local(c, jobSource+"finished"), Scope: scope, Operation: "job_id",
			Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: "/fixture.Jobs/Describe", Path: "job", Single: true}}},
	}, c.GetProgram().GetEvidence())

	policies := []*testpilotspb.CorrelatedFieldPolicy{
		{FieldId: "attempt", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_UINT64}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN},
		{FieldId: "delivery", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN},
	}
	for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
		if rule.GetKind() == local(c, jobEvidence+"taken") || rule.GetKind() == local(c, jobEvidence+"count") {
			protorequire.ProtoSliceEqual(t, policies, rule.GetFields())
		} else {
			require.Empty(t, rule.GetFields(), rule.GetKind())
		}
	}
}

// An exhaustive kind is carried by every Case, whatever its path records: declared with its source,
// lifted by the read that lifts its source's other kinds, and given no meaning in the Contract, which
// reads the path's evidence as before. A kind that is not exhaustive and off the path is not carried.
func TestAnExhaustiveKindIsCarriedOffThePath(t *testing.T) {
	plain := produced(t, "once", jobSources(), jobOnce...)
	sources := jobSources()
	sources[4].Exhaustive = true
	c := produced(t, "once", sources, jobOnce...)
	dropped := local(c, jobEvidence+"wasDropped")

	declarations := c.GetProgram().GetEvidence()
	require.Len(t, declarations, len(plain.GetProgram().GetEvidence())+1)
	protorequire.ProtoEqual(t, &testpilotspb.EvidenceDeclaration{EvidenceId: dropped, EvidenceSource: local(c, jobSource+"wasDropped"),
		Scope:     []*testpilotspb.NamedValue{{FieldId: local(c, jobFamily+".scope.run"), Value: cp.Text("jobs-once")}},
		Operation: "attributes<job_dropped_event_attributes>.job_id",
		Source: &testpilotspb.EvidenceDeclaration_HistoryEvent{HistoryEvent: &testpilotspb.HistoryEventSource{
			AttributesField: "job_dropped_event_attributes"}}}, declarations[len(declarations)-1])

	lift := func(c *testpilotspb.Case) []*testpilotspb.CorrelatedEvidenceRule {
		history := c.GetProgram().GetEntrypoints()[0].GetInstructions()[1]
		return history.GetInstruction().GetInvokeRpc().GetResponseReads()[0].GetTargets()[0].GetCorrelatedEvidence().GetRules()
	}
	require.Empty(t, lift(plain), "the path records no history kind")
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidenceRule{{EvidenceId: dropped}}, lift(c))

	contract := c.GetContract().GetCorrelated()
	require.Contains(t, contract.GetSources(), local(c, jobSource+"wasDropped"))
	var irrelevant []string
	for _, rule := range contract.GetProjectionRules() {
		if rule.GetMeaning() == testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT {
			require.Empty(t, rule.GetOutputs())
			irrelevant = append(irrelevant, rule.GetKind())
		}
	}
	require.Equal(t, []string{dropped}, irrelevant)
	// The Contract reads the path as it did: the same rows, and the same meaning for every other kind.
	protorequire.ProtoSliceEqual(t, plain.GetContract().GetCorrelated().GetTransitions(), contract.GetTransitions())
	require.Equal(t, confirmed(plain), func() map[string][][2]string {
		with := confirmed(c)
		delete(with, dropped)
		return with
	}())
}

// The projection's fingerprint covers what a kind declares beyond its name: each field's id, path,
// type and role, and each kind carried off the path. A realization that declares none of them has the
// fingerprint it had.
func TestTheProjectionFingerprintCoversFieldsAndOffPathKinds(t *testing.T) {
	fingerprint := func(change func([]*cp.EvidenceSource)) string {
		sources := jobSources()
		change(sources)
		return produced(t, "once", sources, jobOnce...).GetContract().GetCorrelated().GetProjectionFingerprint()
	}
	declared := fingerprint(func([]*cp.EvidenceSource) {})
	for name, change := range map[string]func([]*cp.EvidenceSource){
		"a field's id": func(s []*cp.EvidenceSource) {
			s[1].Fields = retyped(s[1].Fields, func(f *cp.EvidenceField) { f.ID = "number" })
		},
		"a field's path": func(s []*cp.EvidenceSource) {
			s[1].Fields = retyped(s[1].Fields, func(f *cp.EvidenceField) { f.Path = "attempt" })
		},
		"a field's type": func(s []*cp.EvidenceSource) {
			s[1].Fields = retyped(s[1].Fields, func(f *cp.EvidenceField) { f.Type = testpilotspb.SCALAR_KIND_TEXT })
		},
		"a field's role": func(s []*cp.EvidenceSource) {
			s[1].Fields = retyped(s[1].Fields, func(f *cp.EvidenceField) { f.Role = "" })
		},
		"a field fewer":          func(s []*cp.EvidenceSource) { s[1].Fields = s[1].Fields[:1] },
		"a kind off the path":    func(s []*cp.EvidenceSource) { s[4].Exhaustive = true },
		"no field on the path":   func(s []*cp.EvidenceSource) { s[1].Fields = nil },
		"the fields in an order": func(s []*cp.EvidenceSource) { s[1].Fields = []cp.EvidenceField{s[1].Fields[1], s[1].Fields[0]} },
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, declared, fingerprint(change))
		})
	}
	// A kind the Case does not carry is in no fingerprint: the second attempt's fields are off this path.
	require.Equal(t, declared, fingerprint(func(s []*cp.EvidenceSource) { s[2].Fields = nil }))
	require.Equal(t, declared, fingerprint(func([]*cp.EvidenceSource) {}), "identical inputs give one fingerprint")
}

// The fingerprint is of the projection's canonical form, written out here for the job taken once
// with its dropped event carried off the path: the projection, the machine it reads and where it
// starts, the scope and the operation key, the sources, each rule in the order of its kind, and the
// window. A rule is its kind, the fields the kind retains, each with its path, its type, that it is
// retained and the identity it names, and what its evidence means: the steps it confirms, each with
// its action, state, outcome and facts, or nothing.
func TestTheProjectionFingerprintIsOfItsCanonicalForm(t *testing.T) {
	sources := jobSources()
	sources[4].Exhaustive = true
	table := jobTable(t)
	q := func(text string) string { return check.Quote(text) }
	canonical := "[" + strings.Join([]string{
		q("checked-projection/v2"), q(jobFamily + ".projection"), q(table.TargetFingerprint()), q(table.SetupKey()), q("idle"),
		"[" + q(jobFamily+".scope.run") + "]", q(jobFamily + ".scope.job"),
		"[" + strings.Join([]string{q(jobSource + "finished"), q(jobSource + "listed"), q(jobSource + "taken"), q(jobSource + "wasDropped")}, ",") + "]",
		"[" + strings.Join([]string{
			"[" + q(jobEvidence+"finished") + `,[],["confirmed",null,[["finish","done","accepted",["finished"]]]]]`,
			"[" + q(jobEvidence+"listed") + `,[],["confirmed",null,[["submit","queued","accepted",["listed"]]]]]`,
			"[" + q(jobEvidence+"taken") + `,[["attempt","activity_attempt.sdk_attempt","SCALAR_KIND_UINT64","retain","attempt"],` +
				`["delivery","activity_attempt.delivery_id","SCALAR_KIND_TEXT","retain","delivery"]],` +
				`["confirmed",null,[["take","running","accepted",["taken","count"]]]]]`,
			"[" + q(jobEvidence+"wasDropped") + `,[],["irrelevant"]]`,
		}, ",") + "]",
		"[8,8,2,16,1000,512]",
	}, ",") + "]"
	require.Equal(t, check.Fingerprint(canonical), produced(t, "once", sources, jobOnce...).GetContract().GetCorrelated().GetProjectionFingerprint())
}

// retyped is the fields with the first one changed, as a copy.
func retyped(fields []cp.EvidenceField, change func(*cp.EvidenceField)) []cp.EvidenceField {
	out := slices.Clone(fields)
	change(&out[0])
	return out
}

// Which kind confirms a step is one decision, made in one order: the kind that names the step, then
// the kind that records the first fact of the step's class that some kind naming no step records.
// Everything else is refused, and nothing is confirmed by a guess.
func TestTheKindThatConfirmsAStepIsRefusedWhereItIsNotOne(t *testing.T) {
	plain := func(records string) *cp.EvidenceSource {
		return jobKind(records, cp.Recorded{Method: "/fixture.Jobs/List", Path: "jobs"}, "job_id")
	}
	named := func(records string, takings ...cp.Taking) *cp.EvidenceSource {
		s := plain(records)
		s.KindID += ".named"
		s.Confirms = takings
		return s
	}
	take := func(n int) cp.Taking { return cp.Taking{Key: "take", Occurrence: n} }
	action := func(key string) string { return jobTable(t).ActionAtom(key).ID }
	for name, test := range map[string]struct {
		sources []*cp.EvidenceSource
		path    []string
		want    *cp.Error
	}{
		"a class taken again that no kind tells apart": {
			[]*cp.EvidenceSource{plain("listed"), plain("taken"), plain("count"), plain("finished")}, jobRetried,
			&cp.Error{Definition: action("fail"), Construct: "evidence.kind-repeated"}},
		"a class taken again whose one kind names no step": {
			[]*cp.EvidenceSource{plain("listed"), plain("taken"), named("count", cp.Taking{Key: "fail", Occurrence: 1}), plain("finished")}, jobRetried,
			&cp.Error{Definition: action("take"), Construct: "evidence.action-repeated"}},
		"a step two kinds name": {
			[]*cp.EvidenceSource{plain("listed"), named("taken", take(1)), named("count", take(1)), plain("finished")}, jobOnce,
			&cp.Error{Definition: jobEvidence + "count.named", Construct: "evidence.taking-ambiguous"}},
		"a step named by a kind that records none of its facts": {
			[]*cp.EvidenceSource{plain("listed"), plain("taken"), plain("count"), named("finished", take(1))}, jobOnce,
			&cp.Error{Definition: jobEvidence + "finished.named", Construct: "evidence.taking-crossed"}},
		"a step whose facts only kinds that name other steps record": {
			[]*cp.EvidenceSource{plain("listed"), named("taken", take(2)), named("count", take(2)), plain("finished")}, jobOnce,
			&cp.Error{Definition: action("take"), Construct: "evidence.taking-unrecorded"}},
		"a kind that names a step the path does not take": {
			[]*cp.EvidenceSource{plain("listed"), named("taken", take(1), take(2)), plain("count"), plain("finished")}, jobOnce,
			&cp.Error{Definition: jobEvidence + "taken.named", Construct: "evidence.confirmation-partial"}},
		"a kind whose steps another kind's evidence lies between": {
			[]*cp.EvidenceSource{plain("listed"), named("taken", take(1), take(2)), named("count", cp.Taking{Key: "fail", Occurrence: 1}), plain("finished")}, jobRetried,
			&cp.Error{Definition: jobEvidence + "taken.named", Construct: "evidence.kind-interrupted"}},
		"a fact no kind records": {
			[]*cp.EvidenceSource{plain("listed"), named("taken", take(1)), plain("finished")}, jobOnce,
			&cp.Error{Definition: "count", Construct: "evidence.kind-unknown"}},
	} {
		t.Run(name, func(t *testing.T) {
			q := jobQuery(t, "refused", test.path...)
			c, err := cp.Produce(q, jobIdentity("refused"), jobRealization(t, test.sources), cp.Source{})
			require.Nil(t, c)
			var refused *cp.Error
			require.ErrorAs(t, err, &refused)
			require.Equal(t, test.want, refused)
			require.Equal(t, err, cp.Preflight(q, jobIdentity("refused"), jobRealization(t, test.sources)), "Preflight refuses what Produce refuses")
		})
	}
}

// Another result of a row the path takes is confirmed with the steps before it that its own result is
// confirmed with. The job is taken, checked, which records nothing, and closed, which finished it: the
// finished kind confirms the check and the close. Closing may also drop the job, which records that it
// was dropped, so that kind confirms the check and the close that drops it, and a Run that took it
// reads as the other result and not as nothing.
func TestAnotherResultOfATakenRowIsConfirmedWithTheStepsBeforeIt(t *testing.T) {
	c := produced(t, "closed", jobSources(), "submit", "take", "check", "close")
	require.Equal(t, map[string][][2]string{
		local(c, jobEvidence+"listed"):     {{"submit", "queued"}},
		local(c, jobEvidence+"taken"):      {{"take", "running"}},
		local(c, jobEvidence+"finished"):   {{"check", "running"}, {"close", "done"}},
		local(c, jobEvidence+"wasDropped"): {{"check", "running"}, {"close", "dropped"}},
	}, confirmed(c))
}

// A kind that names a step confirms it before any kind that records the step's first fact does: the
// take records that the job was taken and its count, a kind that names no step records the first of
// the two, and a kind that names the take records the second. The take is the naming kind's, and the
// other kind, which then confirms no step of the path, is not carried.
func TestAKindThatNamesAStepConfirmsItBeforeAKindThatRecordsItsFact(t *testing.T) {
	sources := jobSources()
	sources[1].Confirms = nil
	sources[2].Confirms = []cp.Taking{{Key: "take", Occurrence: 1}}
	c := produced(t, "once", sources, jobOnce...)
	require.Equal(t, map[string][][2]string{
		local(c, jobEvidence+"listed"):   {{"submit", "queued"}},
		local(c, jobEvidence+"count"):    {{"take", "running"}},
		local(c, jobEvidence+"finished"): {{"finish", "done"}},
	}, confirmed(c))
}

// Identical inputs give identical bytes, with everything a realization can declare in play.
func TestAProducedCaseIsDeterministic(t *testing.T) {
	sources := func() []*cp.EvidenceSource {
		s := jobSources()
		s[4].Exhaustive = true
		return s
	}
	encode := func() []byte {
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(produced(t, "retried", sources(), jobRetried...))
		require.NoError(t, err)
		return encoded
	}
	first, second := encode(), encode()
	require.NotEmpty(t, first)
	require.Equal(t, first, second)
}

// What Produce decides of a path's evidence can be asked without a Case: which kind confirms which
// steps, by their place on the path, in path order, the steps that record nothing with the kind that
// confirms them. A reader that must know when a Run records each piece asks this.
func TestTheStepsEachKindConfirmsAreToldByTheirPlaceOnThePath(t *testing.T) {
	type confirmed struct {
		kind  string
		steps []int
	}
	got, err := cp.Confirmations(jobQuery(t, "retried", jobRetried...), jobIdentity("retried"), jobRealization(t, jobSources()))
	require.NoError(t, err)
	var told []confirmed
	for _, c := range got {
		told = append(told, confirmed{strings.TrimPrefix(c.Source.KindID, jobEvidence), c.Steps})
	}
	require.Equal(t, []confirmed{{"listed", []int{0}}, {"taken", []int{1}}, {"count", []int{2, 3, 4}}, {"finished", []int{5}}}, told)

	unsourced := jobSources()[:4]
	unsourced[2] = unsourced[3]
	q := jobQuery(t, "retried", jobRetried...)
	_, err = cp.Confirmations(q, jobIdentity("retried"), jobRealization(t, unsourced[:3]))
	require.Equal(t, cp.Preflight(q, jobIdentity("retried"), jobRealization(t, unsourced[:3])), err, "it refuses what Preflight refuses")
	require.Error(t, err)
}

// jobModel is the job as a lifted Model: its machine, a Property that the job finishes by the last of
// actions, and a Scenario and find Query of that path, named name.
func jobModel(name string, actions ...string) *umpirespb.Model {
	const stateType, outcomeType, factType, stepType = "JobState", "JobOutcome", "JobFact", "framework.Step"
	at := &umpirespb.Position{File: "job.go", Line: 1}
	named := func(name string) *umpirespb.TypeRef {
		return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: name}}
	}
	literal := func(value *umpirespb.Value) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: value}}
	}
	text := func(value string) *umpirespb.Expr {
		return literal(&umpirespb.Value{Kind: &umpirespb.Value_Text{Text: value}})
	}
	enumValue := func(typ, value string) *umpirespb.Value {
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: value}}}
	}
	enum := func(typ, value string) *umpirespb.Expr { return literal(enumValue(typ, value)) }
	variable := func(name string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Var{Var: name}}
	}
	field := func(name string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Field{Field: &umpirespb.FieldAccess{Base: variable("after"), Field: name}}}
	}
	list := func(values ...*umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{Items: values}}}
	}
	binary := func(op umpirespb.Binary_Op, left, right *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: op, Left: left, Right: right}}}
	}
	enumeration := func(name string, values ...string) *umpirespb.Type {
		cases := make([]*umpirespb.Case, 0, len(values))
		for _, value := range values {
			cases = append(cases, &umpirespb.Case{Name: value})
		}
		return &umpirespb.Type{Name: name, Position: at, Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: cases}}}
	}
	step := func(state string, facts ...string) *umpirespb.Expr {
		values := make([]*umpirespb.Expr, 0, len(facts))
		for _, fact := range facts {
			values = append(values, enum(factType, fact))
		}
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Construct{Construct: &umpirespb.Construct{Type: stepType,
			Args: []*umpirespb.Expr{enum(outcomeType, "accepted"), enum(stateType, state), list(values...), text("")}}}}
	}
	facts := []string{"listed", "taken", "count", "finished", "wasDropped"}
	machine := &umpirespb.Machine{Family: "fixture.job", Name: "job", Position: at, StateType: stateType,
		OutcomeType: outcomeType, FactType: factType, Starts: []*umpirespb.Expr{enum(stateType, "idle")}, Evidence: "job.evidence",
		Ends: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{
			Params: []*umpirespb.Param{{Name: "state", Type: named(stateType)}}, Body: binary(umpirespb.Binary_OP_OR,
				binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, "done")),
				binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, "dropped")))}}}}
	model := &umpirespb.Model{Source: "job.go", Types: []*umpirespb.Type{
		enumeration(stateType, "idle", "queued", "running", "waiting", "done", "dropped"), enumeration(outcomeType, "accepted"), enumeration(factType, facts...)},
		Machines: []*umpirespb.Machine{machine}}
	for _, row := range []struct {
		action, state string
		results       []*umpirespb.Expr
	}{
		{"submit", "idle", []*umpirespb.Expr{step("queued", "listed")}},
		{"take", "queued", []*umpirespb.Expr{step("running", "taken", "count")}},
		{"fail", "running", []*umpirespb.Expr{step("waiting", "listed", "count")}},
		{"settle", "waiting", []*umpirespb.Expr{step("queued")}},
		{"finish", "running", []*umpirespb.Expr{step("done", "finished")}},
		{"drop", "queued", []*umpirespb.Expr{step("dropped", "wasDropped")}},
		{"check", "running", []*umpirespb.Expr{step("running")}},
		{"close", "running", []*umpirespb.Expr{step("done", "finished"), step("dropped", "wasDropped")}},
	} {
		id := "job." + row.action
		model.Actions = append(model.Actions, &umpirespb.Action{Id: id, Name: row.action, Position: at, Actor: "fixture"})
		model.Functions = append(model.Functions, &umpirespb.Function{Name: id + ".step", Position: at,
			Params: []*umpirespb.Param{{Name: "state", Type: named(stateType)}}, Body: &umpirespb.Expr{Position: at,
				Kind: &umpirespb.Expr_If{If: &umpirespb.If{Condition: binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, row.state)), Then: list(row.results...), Else: list()}}}})
		machine.Steps = append(machine.Steps, &umpirespb.StepBinding{Action: id, Function: id + ".step", Position: at})
	}
	evidence := &umpirespb.Match{Scrutinee: variable("fact")}
	for _, fact := range facts {
		evidence.Cases = append(evidence.Cases, &umpirespb.MatchCase{Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Literal{Literal: enumValue(factType, fact)}}, Body: text(fact)})
	}
	model.Functions = append(model.Functions,
		&umpirespb.Function{Name: "job.evidence", Position: at, Params: []*umpirespb.Param{{Name: "fact", Type: named(factType)}}, Body: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Match{Match: evidence}}},
		&umpirespb.Function{Name: "job.finishes", Position: at, Params: []*umpirespb.Param{{Name: "after", Type: named(stepType)}}, Body: binary(umpirespb.Binary_OP_AND,
			binary(umpirespb.Binary_OP_EQ, field("state"), enum(stateType, "done")), binary(umpirespb.Binary_OP_CONTAINS, enum(factType, "finished"), field("facts")))})
	last := actions[len(actions)-1]
	model.Properties = []*umpirespb.Property{{Machine: "job", Name: "finishes", Position: at, Holds: "job.finishes",
		When: &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: "job." + last}}}}
	scenario := &umpirespb.Scenario{Machine: "job", Name: name, Position: at, Start: enum(stateType, "idle")}
	for _, action := range actions {
		scenario.Actions = append(scenario.Actions, &umpirespb.ActionClass{Action: "job." + action})
	}
	model.Scenarios = []*umpirespb.Scenario{scenario}
	model.Queries = []*umpirespb.Query{{Name: name, Position: at, Form: umpirespb.Query_FORM_FIND,
		Property: &umpirespb.ClaimRef{Machine: "job", Name: "finishes"}, Scenario: &umpirespb.ClaimRef{Machine: "job", Name: name},
		Limits: &umpirespb.Limits{Name: "eight", Steps: 8, Actions: 8, Search: 4096}}}
	return model
}
