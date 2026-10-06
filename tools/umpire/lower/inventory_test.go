package lower

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// declared is what a realization declares, read off its message tree with no list kept here: every
// populated field of the realization, every populated field of its correlation, each element of a
// repeated declaration by its id, and each command of a script, once per class that performs it. A
// position says where a declaration was written and declares nothing.
func declared(t *testing.T, m *umpirespb.Model, r *umpirespb.Realization) [][2]string {
	t.Helper()
	var out [][2]string
	realizer, err := umpiremodel.NewRealizer(m, umpiremodel.DefaultScope)
	require.NoError(t, err)
	classKey := realizer.ClassKey
	message := r.ProtoReflect()
	fields := message.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		if !message.Has(f) || f.Name() == "position" {
			continue
		}
		name := string(f.Name())
		switch {
		case f.IsList():
			list := message.Get(f).List()
			for j := range list.Len() {
				element := list.Get(j).Message()
				if step, ok := element.Interface().(*umpirespb.ServerStep); ok {
					// A server step is named by its class.
					out = append(out, [2]string{name, classKey(step.GetStep())})
					continue
				}
				id := element.Descriptor().Fields().ByName("id")
				if id == nil {
					// A required setting is named by its key.
					id = element.Descriptor().Fields().ByName("key")
				}
				out = append(out, [2]string{name, element.Get(id).String()})
			}
		case f.Name() == "behavior":
			// The API behavior is its hints, each by its id.
			for _, v := range r.GetBehavior().GetVisibility() {
				out = append(out, [2]string{name, v.GetId()})
			}
			for _, c := range r.GetBehavior().GetCauses() {
				out = append(out, [2]string{name, c.GetId()})
			}
			// Its other declarations each by their own name.
			if r.GetBehavior().GetAttemptNumbering() != nil {
				out = append(out, [2]string{name, "attemptNumbering"})
			}
			if r.GetBehavior().GetInstructionDefaults() != nil {
				out = append(out, [2]string{name, "instructionDefaults"})
			}
			if r.GetBehavior().GetRunOrderIsCausal() {
				out = append(out, [2]string{name, "runOrderIsCausal"})
			}
		case f.Kind() == protoreflect.MessageKind:
			inner := message.Get(f).Message()
			inner.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
				if fd.Name() != "position" {
					out = append(out, [2]string{name, string(fd.Name())})
				}
				return true
			})
		default:
			out = append(out, [2]string{"realization", name})
		}
	}
	for _, s := range r.GetScripts() {
		for _, class := range s.GetActivity().GetStarts() {
			out = append(out, [2]string{"activation", s.GetId() + " [" + classKey(class) + "]"})
		}
		for _, item := range s.GetItems() {
			if c := item.GetCommand(); c != nil {
				out = append(out, [2]string{"command", s.GetId() + "/" + c.GetId()})
			}
			for _, p := range item.GetPerforms() {
				out = append(out, [2]string{"command", s.GetId() + "/" + p.GetCommand().GetId() + " [" + classKey(p.GetStep()) + "]"})
			}
		}
	}
	slices.SortFunc(out, func(a, b [2]string) int { return slices.Compare(a[:], b[:]) })
	return out
}

// The inventory of a Case has one entry for everything the realization declares, as its message tree
// lists it: its envelope, its correlation field by field, its cleanup, and every repeated declaration.
func TestTheInventoryIsEveryDeclarationOfTheRealization(t *testing.T) {
	m := loaded(t, "nexus-caller")
	want := declared(t, m, m.GetRealizations()[0])
	require.Contains(t, want, [2]string{"realization", "producer"})
	require.Contains(t, want, [2]string{"correlation", "event_size"})
	require.Contains(t, want, [2]string{"realization", "cleanup"})
	p, err := NewProducer(m)
	require.NoError(t, err)
	for _, query := range functionalQueries {
		t.Run(query, func(t *testing.T) {
			l, err := p.Lower(query, nexusIdentity(query))
			require.NoError(t, err)
			var got [][2]string
			for _, e := range l.Inventory {
				got = append(got, [2]string{e.Kind, e.ID})
			}
			slices.SortFunc(got, func(a, b [2]string) int { return slices.Compare(a[:], b[:]) })
			require.Equal(t, want, got)
		})
	}
}

// Every field the IR gives a Realization, a Correlation and a kind of evidence has a place in the
// inventory, so a field the schema gains fails here until it is accounted for.
func TestEveryFieldOfARealizationHasAPlaceInTheInventory(t *testing.T) {
	fields := (&umpirespb.Realization{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		require.Contains(t, realizationFields, fields.Get(i).Name())
	}
	require.Len(t, realizationFields, fields.Len())
	fields = (&umpirespb.Correlation{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		require.Contains(t, correlationFields, fields.Get(i).Name())
	}
	require.Len(t, correlationFields, fields.Len())
	fields = (&umpirespb.Evidence{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		require.Contains(t, evidenceFields, fields.Get(i).Name())
	}
	require.Len(t, evidenceFields, fields.Len())
}

// What the inventory says of each declaration is what the Case holds: a command in the Case is one of
// its instructions and every instruction is some command's, a kind of evidence in the Case is one it
// declares, and everything else the realization declares is in every Case or names the realization.
func TestTheInventoryAgreesWithTheCase(t *testing.T) {
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	for _, query := range functionalQueries {
		t.Run(query, func(t *testing.T) {
			l, err := p.Lower(query, nexusIdentity(query))
			require.NoError(t, err)
			var carried, evidence, waits []string
			for _, e := range l.Inventory {
				requireDeclaredIn(t, e.Position, realizationAt)
				require.Equal(t, e.Disposition == InCase, len(e.As) > 0, e.ID)
				switch e.Kind {
				case "command":
					carried = append(carried, e.As...)
				case "evidence":
					for _, part := range e.As {
						if strings.HasPrefix(part, "program.evidence[") {
							evidence = append(evidence, part)
						}
					}
				case "realization":
					require.Equal(t, slices.Contains([]string{"id", "name"}, e.ID), e.Disposition == Names, e.ID)
				case "behavior", "server_steps":
					require.Contains(t, []Disposition{InCase, Unread}, e.Disposition, e.ID)
					if slices.Contains([]string{"attemptNumbering", "instructionDefaults", "runOrderIsCausal"}, e.ID) {
						// The rest of the behavior is in the Program's part that carries it.
						for _, part := range e.As {
							require.True(t, strings.HasPrefix(part, "program.instruction_defaults.") || part == "program.run_order_is_causal" ||
								strings.HasPrefix(part, "program.entrypoints["), part)
						}
						continue
					}
					// A hint or server step a wait reads is in the instructions whose waits it shapes.
					waits = append(waits, e.As...)
				default:
					require.Equal(t, InCase, e.Disposition, e.ID)
				}
			}
			var instructions, declared []string
			for _, e := range l.Case.GetProgram().GetEntrypoints() {
				for _, n := range e.GetInstructions() {
					instructions = append(instructions, "program.entrypoints["+e.GetEntrypointId()+"].instructions["+n.GetInstructionId()+"]")
				}
			}
			for _, d := range l.Case.GetProgram().GetEvidence() {
				declared = append(declared, "program.evidence["+d.GetEvidenceId()+"]")
			}
			require.ElementsMatch(t, instructions, carried)
			require.ElementsMatch(t, declared, evidence)
			require.Subset(t, instructions, waits)
			require.NotEmpty(t, waits, "every Nexus Case waits for its scheduled event")
		})
	}

	// The deadline path stops the handler's worker and never answers: read off Claims.scala's
	// scheduleToStartExpires and the scripts of Realization.scala.
	l, err := p.Lower("scheduleToStartTimeout", nexusIdentity("scheduleToStartTimeout"))
	require.NoError(t, err)
	dispositions, as := map[string]Disposition{}, map[string][]string{}
	for _, e := range l.Inventory {
		dispositions[e.Kind+" "+e.ID], as[e.Kind+" "+e.ID] = e.Disposition, e.As
	}
	for id, want := range map[string]Disposition{
		"command controller/stop-handler-worker [stop]":                         InCase,
		"command controller/pending-attempts":                                   OffPath,
		"command controller/await-completion-authority":                         OffPath,
		"command controller/complete-nexus-operation [complete-succeeded]":      OffPath,
		"command handler/respond-sync [reply-syncSuccess]":                      OffPath,
		"command handler/respond-async [reply-async]":                           OffPath,
		"command workflow/start-nexus-operation [schedule-unset-expires-unset]": InCase,
		"command workflow/start-nexus-operation [schedule-unset-unset-unset]":   OffPath,
		"command workflow/await-nexus-operation":                                InCase,
		"evidence temporal.features.nexuscaller.evidence.scheduled":             InCase,
		"evidence temporal.features.nexuscaller.evidence.timedOut":              InCase,
		// The completed event is exhaustive, so the Case carries it though the path records none.
		"evidence temporal.features.nexuscaller.evidence.completed":       InCase,
		"evidence temporal.features.nexuscaller.evidence.pendingAttempts": OffPath,
		"realization name": Names,
	} {
		require.Equal(t, want, dispositions[id], id)
	}
	// Where each field of the envelope and the correlation went, read off the Case protocol.
	for id, want := range map[string][]string{
		"realization producer":         {"provenance.producer_id"},
		"realization producer_version": {"provenance.producer_version"},
		"realization machine":          {"provenance.definitions"},
		"realization cleanup":          {"program.cleanup"},
		"correlation projection":       {"contract.correlated.projection_id"},
		"correlation run":              {"contract.correlated.scope_fields"},
		"correlation operation":        {"contract.correlated.operation_field"},
		"correlation observation":      {"contract.correlated.evidence_observation_id"},
		"correlation events":           {"contract.correlated.projection_fingerprint"},
		"correlation event_size":       {"contract.correlated.projection_fingerprint"},
	} {
		require.Equal(t, want, as[id], id)
	}
}

// ready is a Query checked and its Case produced, for the inventory to be taken of a Case changed by
// hand.
func ready(t *testing.T, m *umpirespb.Model, query string) (*lowering, *testpilotspb.Case) {
	t.Helper()
	p, err := NewProducer(m)
	require.NoError(t, err)
	a, _, err := p.ask(query)
	require.NoError(t, err)
	l, problems := p.check(a, nexusIdentity(query))
	require.Empty(t, problems)
	produced, err := cp.Produce(l.query, nexusIdentity(query), l.realization, p.source(a.q))
	require.NoError(t, err)
	_, err = l.inventory(produced)
	require.NoError(t, err)
	return l, produced
}

// The inventory is taken both ways. A part of a Case that neither a declaration nor the Query accounts
// for is an error, in the Program, the Contract and the provenance alike; and so is a declaration the
// Case does not carry as it was declared.
func TestAnInventoryThatDoesNotCloseIsAnError(t *testing.T) {
	l, produced := ready(t, loaded(t, "nexus-caller"), "syncCompletion")
	for _, c := range []struct {
		name   string
		change func(c *testpilotspb.Case)
		want   string
	}{
		{"an instruction no command accounts for", func(c *testpilotspb.Case) {
			e := c.GetProgram().GetEntrypoints()[0]
			e.Instructions = append(e.Instructions, &testpilotspb.InstructionNode{InstructionId: "stray"})
		}, "carries program.entrypoints[controller].instructions[stray], which no declaration of realization asyncNexus accounts for"},
		{"a role no declaration accounts for", func(c *testpilotspb.Case) {
			c.GetProgram().Roles = append(c.GetProgram().Roles, &testpilotspb.Role{RoleId: "stray"})
		}, "carries program.roles[stray], which no declaration"},
		{"a Contract rule nothing accounts for", func(c *testpilotspb.Case) {
			c.GetContract().Rules = []*testpilotspb.ContractRule{{RuleId: "stray"}}
		}, "carries contract.rules, which no declaration"},
		{"a provenance row nothing accounts for", func(c *testpilotspb.Case) {
			c.GetProvenance().ModelValueFingerprints = []*testpilotspb.ModelValueFingerprint{{LocalName: "stray"}}
		}, "carries provenance.model_value_fingerprints, which no declaration"},
		{"a step of the path with no instruction", func(c *testpilotspb.Case) { c.GetProgram().GetEntrypoints()[2].Instructions = nil },
			"the path performs reply-syncSuccess, and the Case carries no handler/respond-sync for it"},
		{"a learned value in no slot", func(c *testpilotspb.Case) { c.GetProgram().Slots = nil },
			"learned completion-authority of realization asyncNexus is in no part of the Case"},
		{"another producer", func(c *testpilotspb.Case) { c.GetProvenance().ProducerId = "someone.else" },
			`realization producer of realization asyncNexus is "temporal.features.nexuscaller.testpilot", and the Case's provenance.producer_id carries "someone.else"`},
		{"another producer version", func(c *testpilotspb.Case) { c.GetProvenance().ProducerVersion = "2" },
			`realization producer_version of realization asyncNexus is "1", and the Case's provenance.producer_version carries "2"`},
		{"another target", func(c *testpilotspb.Case) { c.GetProvenance().GetDefinitions()[0].DefinitionId = "elsewhere" },
			`realization machine of realization asyncNexus is "temporal.features.nexuscaller.system.target.nexusSystem", and the Case's provenance.definitions carries "elsewhere"`},
		{"no target", func(c *testpilotspb.Case) { c.GetProvenance().Definitions = c.GetProvenance().GetDefinitions()[1:] },
			`realization machine of realization asyncNexus is "temporal.features.nexuscaller.system.target.nexusSystem", and the Case's provenance.definitions carries ""`},
		{"no cleanup", func(c *testpilotspb.Case) { c.GetProgram().Cleanup = nil },
			`realization cleanup of realization asyncNexus is "cleanup", and the Case's program.cleanup carries ""`},
		{"another projection", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().ProjectionId = "elsewhere" },
			`correlation projection of realization asyncNexus is "projection", and the Case's contract.correlated.projection_id carries "elsewhere"`},
		{"another projection in the provenance", func(c *testpilotspb.Case) { c.GetProvenance().GetCorrelatedRules()[0].ProjectionId = "elsewhere" },
			`the Case's provenance.correlated_rules carries "elsewhere"`},
		{"another scope", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().ScopeFields = []string{"run", "more"} },
			`correlation run of realization asyncNexus is "run", and the Case's contract.correlated.scope_fields carries "run,more"`},
		{"another scope on a kind of evidence", func(c *testpilotspb.Case) { c.GetProgram().GetEvidence()[0].GetScope()[0].FieldId = "elsewhere" },
			"correlation run of realization asyncNexus is \"run\", and the Case's program.evidence[evidence.scheduled] carries"},
		{"another operation key", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().OperationField = "elsewhere" },
			`correlation operation of realization asyncNexus is "operation", and the Case's contract.correlated.operation_field carries "elsewhere"`},
		{"another evidence observation", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().EvidenceObservationId = "history-event" },
			`correlation observation of realization asyncNexus is "correlated-evidence", and the Case's contract.correlated.evidence_observation_id carries "history-event"`},
		{"no projection fingerprint", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().ProjectionFingerprint = "" },
			"the Case's contract.correlated.projection_fingerprint carries"},
		{"a source the evidence does not name", func(c *testpilotspb.Case) { c.GetContract().GetCorrelated().Sources = []string{"elsewhere"} },
			"the Case's contract.correlated.sources carries \"elsewhere\""},
		{"an exhaustive kind with no closing read", func(c *testpilotspb.Case) {
			controller := c.GetProgram().GetEntrypoints()[0]
			controller.Instructions = controller.GetInstructions()[:len(controller.GetInstructions())-1]
		}, "evidence temporal.features.nexuscaller.evidence.started is exhaustive, and the Case carries no controller/history to close it"},
	} {
		t.Run(c.name, func(t *testing.T) {
			changed := proto.CloneOf(produced)
			c.change(changed)
			_, err := l.inventory(changed)
			require.ErrorContains(t, err, c.want)
		})
	}
}

// A realization's required settings are the Program's, in the order declared and with the values
// declared, each one entry of the inventory; a Case that carries one otherwise, or one more, does not
// close.
func TestTheRequiredSettingsAreTheProgramsAsDeclared(t *testing.T) {
	m := loaded(t, "nexus-caller")
	settings := []*umpirespb.RequiredSetting{{Key: "b.second", Value: "true"}, {Key: "a.first", Value: "1"}}
	m.GetRealizations()[0].RequiredSettings = settings
	p, err := NewProducer(m)
	require.NoError(t, err)
	lowered, err := p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.NoError(t, err)
	carried := lowered.Case.GetProgram().GetRequiredSettings()
	require.Len(t, carried, len(settings))
	for i, s := range settings {
		require.Equal(t, [2]string{s.GetKey(), s.GetValue()}, [2]string{carried[i].GetKey(), carried[i].GetValue()})
	}
	as := map[string][]string{}
	for _, e := range lowered.Inventory {
		if e.Kind == "required_settings" {
			require.Equal(t, InCase, e.Disposition)
			as[e.ID] = e.As
		}
	}
	require.Equal(t, map[string][]string{"b.second": {"program.required_settings[b.second]"}, "a.first": {"program.required_settings[a.first]"}}, as)
	got := declared(t, m, m.GetRealizations()[0])
	require.Contains(t, got, [2]string{"required_settings", "b.second"})

	l, produced := ready(t, m, "syncCompletion")
	for _, c := range []struct {
		name   string
		change func(c *testpilotspb.Case)
		want   string
	}{
		{"another value", func(c *testpilotspb.Case) { c.GetProgram().GetRequiredSettings()[0].Value = "false" },
			`required_settings b.second of realization asyncNexus is "b.second=true", and the Case's program.required_settings[b.second] carries "b.second=false"`},
		{"another order", func(c *testpilotspb.Case) { slices.Reverse(c.GetProgram().GetRequiredSettings()) },
			`required_settings b.second of realization asyncNexus is "b.second=true", and the Case's program.required_settings[b.second] carries "a.first=1"`},
		{"one missing", func(c *testpilotspb.Case) { c.GetProgram().RequiredSettings = c.GetProgram().GetRequiredSettings()[:1] },
			`required_settings a.first of realization asyncNexus is "a.first=1", and the Case's program.required_settings[a.first] carries ""`},
		{"one more", func(c *testpilotspb.Case) {
			c.GetProgram().RequiredSettings = append(c.GetProgram().RequiredSettings, &testpilotspb.RequiredSetting{Key: "stray", Value: "1"})
		}, "carries program.required_settings[stray], which no declaration of realization asyncNexus accounts for"},
	} {
		t.Run(c.name, func(t *testing.T) {
			changed := proto.CloneOf(produced)
			c.change(changed)
			_, err := l.inventory(changed)
			require.ErrorContains(t, err, c.want)
		})
	}
}

// A kind of evidence the Case carries is carried as the realization declares it or is an error: where
// it is recorded, and the fields it keeps. The Case here is produced first and the declaration changed
// after, so each change is one the Case does not carry.
func TestAKindOfEvidenceTheCaseDoesNotCarryAsDeclaredDoesNotClose(t *testing.T) {
	const completed = "evidence temporal.features.nexuscaller.evidence.completed"
	for name, test := range map[string]struct {
		change func(e *umpirespb.Evidence)
		want   string
	}{
		"a field": {func(e *umpirespb.Evidence) {
			e.Fields = []*umpirespb.EvidenceField{{Id: "request", Path: "request_id"}}
		}, completed + " keeps field request at request_id, and the Case's program.evidence[completed] does not"},
		"a read from one message": {func(e *umpirespb.Evidence) {
			e.From = &umpirespb.Evidence_Single{Single: &umpirespb.ReadSource{Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution", Path: "workflow_execution_info"}}
		}, completed + ` of realization asyncNexus is "single read /temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution workflow_execution_info", ` +
			`and the Case's program.evidence[completed] carries "history event nexus_operation_completed_event_attributes"`},
		"the Run's own record": {func(e *umpirespb.Evidence) {
			e.From = &umpirespb.Evidence_RunEvent{RunEvent: &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED,
				Script: "controller", Command: "start-workflow"}}
		}, completed + ` of realization asyncNexus is "run event KIND_INSTRUCTION_COMPLETED of controller/start-workflow keyed by nothing", ` +
			`and the Case's program.evidence[completed] carries "history event nexus_operation_completed_event_attributes"`},
		"another history event": {func(e *umpirespb.Evidence) {
			e.From = &umpirespb.Evidence_History{History: "nexus_operation_failed_event_attributes"}
		}, completed + ` of realization asyncNexus is "history event nexus_operation_failed_event_attributes", ` +
			`and the Case's program.evidence[completed] carries "history event nexus_operation_completed_event_attributes"`},
	} {
		t.Run(name, func(t *testing.T) {
			l, produced := ready(t, loaded(t, "nexus-caller"), "syncCompletion")
			for _, e := range l.a.r.GetEvidence() {
				if e.GetId() == "temporal.features.nexuscaller.evidence.completed" {
					test.change(e)
				}
			}
			_, err := l.inventory(produced)
			require.ErrorContains(t, err, test.want)
			require.ErrorContains(t, err, realizationAt)
		})
	}
}

// The window a check keeps reaches the Case only in the fingerprint of its projection, and every bound
// of it does: a Case lowered under another bound has another fingerprint.
func TestEveryBoundOfTheWindowIsInTheProjectionFingerprint(t *testing.T) {
	fingerprint := func(t *testing.T, change func(*umpirespb.Correlation)) string {
		m := loaded(t, "nexus-caller")
		change(m.GetRealizations()[0].GetCorrelation())
		_, produced := ready(t, m, "syncCompletion")
		return produced.GetContract().GetCorrelated().GetProjectionFingerprint()
	}
	declared := fingerprint(t, func(*umpirespb.Correlation) {})
	require.NotEmpty(t, declared)
	for name, change := range map[string]func(*umpirespb.Correlation){
		"events":     func(c *umpirespb.Correlation) { c.Events++ },
		"buffered":   func(c *umpirespb.Correlation) { c.Buffered++ },
		"keys":       func(c *umpirespb.Correlation) { c.Keys++ },
		"support":    func(c *umpirespb.Correlation) { c.Support++ },
		"work":       func(c *umpirespb.Correlation) { c.Work++ },
		"event_size": func(c *umpirespb.Correlation) { c.EventSize++ },
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, declared, fingerprint(t, change))
		})
	}
}
