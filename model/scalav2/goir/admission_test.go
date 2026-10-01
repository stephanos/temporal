package goir

// What a reader rejects before any check (model/scalav2/SEMANTICS.md, Admission), each case one
// mutation of a source-derived Model, reported at the position the lifter recorded for the
// declaration or node concerned.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/proto"
)

const (
	admLifts        = "model/scalav2/lifter/testdata/lifts/"
	admChannelsAt   = admLifts + "Channels.scala.fixture:"
	admDeclaredAt   = admLifts + "Declarations.scala.fixture:"
	admPresenceAt   = admLifts + "Presence.scala.fixture:"
	admAdmissionAt  = admLifts + "Admission.scala.fixture:"
	admChannelsPkg  = "fixture.channels.Channels$package$."
	admDeclaredPkg  = "fixture.declarations.Declarations$package$."
	admAdmissionPkg = "fixture.specimens.admission.Admission$package$."
)

type admissionCase struct {
	name    string
	fixture string
	mutate  func(m *modelirspb.Model)
	want    string
}

func runAdmissionCases(t *testing.T, cases []admissionCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := admFixture(t, c.fixture)
			c.mutate(m)
			require.ErrorContains(t, Validate(m), c.want)
		})
	}
}

func admFixture(t *testing.T, name string) *modelirspb.Model {
	t.Helper()
	if name == "nexus" {
		return proto.Clone(load(t)).(*modelirspb.Model)
	}
	return proto.Clone(lifted(t, name)).(*modelirspb.Model)
}

func admMachine(m *modelirspb.Model, name string) *modelirspb.Machine {
	for _, mm := range m.GetMachines() {
		if mm.GetName() == name {
			return mm
		}
	}
	return nil
}

func admAction(m *modelirspb.Model, id string) *modelirspb.Action {
	for _, a := range m.GetActions() {
		if a.GetId() == id {
			return a
		}
	}
	return nil
}

func admType(m *modelirspb.Model, name string) *modelirspb.Type {
	for _, t := range m.GetTypes() {
		if t.GetName() == name {
			return t
		}
	}
	return nil
}

func admChannel(m *modelirspb.Model, name string) *modelirspb.Channel {
	for _, c := range m.GetChannels() {
		if c.GetName() == name {
			return c
		}
	}
	return nil
}

func admMonitor(m *modelirspb.Model, name string) *modelirspb.Monitor {
	for _, mo := range m.GetMonitors() {
		if mo.GetName() == name {
			return mo
		}
	}
	return nil
}

func admComposition(m *modelirspb.Model, name string) *modelirspb.Composition {
	for _, c := range m.GetCompositions() {
		if c.GetName() == name {
			return c
		}
	}
	return nil
}

func admQuery(m *modelirspb.Model, name string) *modelirspb.Query {
	for _, q := range m.GetQueries() {
		if q.GetName() == name {
			return q
		}
	}
	return nil
}

func admProperty(m *modelirspb.Model, machine, name string) *modelirspb.Property {
	for _, p := range m.GetProperties() {
		if p.GetMachine() == machine && p.GetName() == name {
			return p
		}
	}
	return nil
}

func admScenario(m *modelirspb.Model, machine, name string) *modelirspb.Scenario {
	for _, s := range m.GetScenarios() {
		if s.GetMachine() == machine && s.GetName() == name {
			return s
		}
	}
	return nil
}

// admFirst is the first expression under x, in evaluation order, that accepts.
func admFirst(x *modelirspb.Expr, accept func(*modelirspb.Expr) bool) *modelirspb.Expr {
	if x == nil {
		return nil
	}
	if accept(x) {
		return x
	}
	var under []*modelirspb.Expr
	switch k := x.GetKind().(type) {
	case *modelirspb.Expr_Field:
		under = []*modelirspb.Expr{k.Field.GetBase()}
	case *modelirspb.Expr_Call:
		under = k.Call.GetArgs()
	case *modelirspb.Expr_Construct:
		under = k.Construct.GetArgs()
	case *modelirspb.Expr_Copy:
		under = []*modelirspb.Expr{k.Copy.GetBase()}
		for _, u := range k.Copy.GetUpdates() {
			under = append(under, u.GetValue())
		}
	case *modelirspb.Expr_Unary:
		under = []*modelirspb.Expr{k.Unary.GetOperand()}
	case *modelirspb.Expr_Binary:
		under = []*modelirspb.Expr{k.Binary.GetLeft(), k.Binary.GetRight()}
	case *modelirspb.Expr_If:
		under = []*modelirspb.Expr{k.If.GetCondition(), k.If.GetThen(), k.If.GetElse()}
	case *modelirspb.Expr_Match:
		under = []*modelirspb.Expr{k.Match.GetScrutinee()}
		for _, c := range k.Match.GetCases() {
			under = append(under, c.GetGuard(), c.GetBody())
		}
	case *modelirspb.Expr_Let:
		under = []*modelirspb.Expr{k.Let.GetValue(), k.Let.GetBody()}
	case *modelirspb.Expr_List:
		under = k.List.GetItems()
	case *modelirspb.Expr_Lambda:
		under = []*modelirspb.Expr{k.Lambda.GetBody()}
	case *modelirspb.Expr_Inbox:
		under = []*modelirspb.Expr{k.Inbox.GetContents(), k.Inbox.GetMessage()}
	default:
	}
	for _, u := range under {
		if found := admFirst(u, accept); found != nil {
			return found
		}
	}
	return nil
}

func admBody(m *modelirspb.Model, name string) *modelirspb.Expr {
	return function(m, name).GetBody()
}

func isBinary(x *modelirspb.Expr) bool { return x.GetBinary() != nil }
func isInbox(x *modelirspb.Expr) bool  { return x.GetInbox() != nil }
func isHole(x *modelirspb.Expr) bool   { return x.GetHole() != "" }

func TestAdmissionAdmitsTheSourceDerivedModels(t *testing.T) {
	for _, name := range []string{"admission", "channels", "closereset", "declarations", "presence", "nexus"} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, Validate(admFixture(t, name)))
		})
	}
}

func TestAdmissionRejectsUnknownVersionsAndConstructs(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"version", "channels", func(m *modelirspb.Model) { m.Version = 1 },
			"model/scalav2: fixture.channels.Channels$package$.relay, fixture.channels.Channels$package$.tallying: " +
				"version 1 is not a version this reader knows"},
		{"binary op unspecified", "declarations", func(m *modelirspb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().Op = modelirspb.Binary_OP_UNSPECIFIED
		}, admDeclaredAt + "70: no binary operator 0"},
		{"binary op outside the enum", "declarations", func(m *modelirspb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().Op = 99
		}, admDeclaredAt + "70: no binary operator 99"},
		{"unary op outside the enum", "declarations", func(m *modelirspb.Model) {
			cond := admFirst(admBody(m, "crashStep"), isBinary)
			cond.Kind = &modelirspb.Expr_Unary{Unary: &modelirspb.Unary{Op: 7, Operand: cond.GetBinary().GetLeft()}}
		}, admDeclaredAt + "70: no unary operator 7"},
		{"inbox op unspecified", "channels", func(m *modelirspb.Model) {
			admFirst(admBody(m, "talkStep"), isInbox).GetInbox().Op = modelirspb.Inbox_OP_UNSPECIFIED
		}, admChannelsAt + "36: no inbox operation 0"},
		{"pattern of no kind", "declarations", func(m *modelirspb.Model) {
			admBody(m, "putStep").GetMatch().GetCases()[0].Pattern = &modelirspb.Pattern{}
		}, admDeclaredAt + "59: a pattern of no known kind"},
		{"value of no kind", "declarations", func(m *modelirspb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().GetRight().GetLiteral().Kind = nil
		}, admDeclaredAt + "70: a value of no known kind"},
		{"channel order unspecified", "channels", func(m *modelirspb.Model) {
			admChannel(m, "wire").Order = modelirspb.Channel_ORDER_UNSPECIFIED
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire has no known order"},
		{"channel order outside the enum", "channels", func(m *modelirspb.Model) {
			admChannel(m, "wire").Order = 5
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire has no known order"},
		{"query form unspecified", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").Form = modelirspb.Query_FORM_UNSPECIFIED
		}, admDeclaredAt + "160: query durableStays has no known form"},
		{"query form outside the enum", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").Form = 3
		}, admDeclaredAt + "160: query durableStays has no known form"},
		{"monitor without an evaluation point", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "endsDurable").Evaluate = nil
		}, admDeclaredAt + "87: monitor endsDurable has no evaluation point"},
	})
}

func TestAdmissionRejectsUndeclaredNamesAndArities(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"channel type", "channels", func(m *modelirspb.Model) {
			admType(m, "fixture.channels.Relay").GetRecord().GetFields()[1].Type = &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: "nowhere"}}
		}, admChannelsAt + "22: no channel nowhere"},
		{"inbox channel", "channels", func(m *modelirspb.Model) {
			admFirst(admBody(m, "talkStep"), isInbox).GetInbox().Channel = "nowhere"
		}, admChannelsAt + "36: no channel nowhere"},
		{"delivered channel", "channels", func(m *modelirspb.Model) {
			admAction(m, admChannelsPkg+"wire.deliver").Delivers = "nowhere"
		}, admChannelsAt + "14: no channel nowhere"},
		{"lost channel", "channels", func(m *modelirspb.Model) {
			admAction(m, admChannelsPkg+"radio.lose").Loses = "nowhere"
		}, admChannelsAt + "16: no channel nowhere"},
		{"machine monitor", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").Monitors[0] = "nowhere"
		}, admDeclaredAt + "99: no monitor nowhere"},
		{"machine assumption", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "store").Assumes[0] = "nowhere"
		}, admDeclaredAt + "37: no assumption nowhere"},
		{"progress assumption", "declarations", func(m *modelirspb.Model) {
			m.GetProgress()[0].Assumptions[0] = "nowhere"
		}, admDeclaredAt + "111: no assumption nowhere"},
		{"fair action", "declarations", func(m *modelirspb.Model) {
			m.GetAssumptions()[0].Fair[0] = "nowhere"
		}, admDeclaredAt + "28: no action nowhere"},
		{"hole", "declarations", func(m *modelirspb.Model) {
			admFirst(admBody(m, "crashStep"), isHole).Kind = &modelirspb.Expr_Hole{Hole: "nowhere"}
		}, admDeclaredAt + "70: no hole nowhere"},
		{"refined machine", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").GetRefines().Product = "nowhere"
		}, admDeclaredAt + "99: no machine nowhere"},
		{"member machine", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "pair").GetMembers()[0].Machine = "nowhere"
		}, admDeclaredAt + "123: no machine nowhere"},
		{"replaced machine", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "detailedPair").GetMembers()[1].Replaces = "nowhere"
		}, admDeclaredAt + "128: no machine nowhere"},
		{"sync member", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "pair").GetSyncs()[0].GetFirst().Member = "middle"
		}, admDeclaredAt + "123: sync putBoth of pair has no member middle"},
		{"sync action", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "pair").GetSyncs()[0].GetSecond().Action = "flush"
		}, admDeclaredAt + "123: sync putBoth of pair: store binds no action flush"},
		{"property machine", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "store", "putStores").Machine = "nowhere"
		}, admDeclaredAt + "137: no machine or composition nowhere"},
		{"scenario machine", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "store", "putOnce").Machine = "nowhere"
		}, admDeclaredAt + "148: no machine or composition nowhere"},
		{"progress machine", "declarations", func(m *modelirspb.Model) {
			m.GetProgress()[0].Machine = "nowhere"
		}, admDeclaredAt + "111: no machine nowhere"},
		{"query property", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").GetProperty().Name = "nothing"
		}, admDeclaredAt + "160: query durableStays: no Property nothing of disk"},
		{"query scenario", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").GetScenario().Name = "nothing"
		}, admDeclaredAt + "160: query durableStays: no Scenario nothing of disk"},
		{"monitor next arity", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").Next = admDeclaredPkg + "storedOnce.violated"
		}, admDeclaredAt + "82: storedOnce: " + admDeclaredPkg + "storedOnce.violated is not a function of three arguments"},
		{"monitor violated arity", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").Violated = admDeclaredPkg + "countStored"
		}, admDeclaredAt + "82: storedOnce: " + admDeclaredPkg + "countStored is not a function of one argument"},
		{"monitor after", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "stagedBeforeDurable").Evaluate = &modelirspb.Monitor_After{After: "nowhere"}
		}, admDeclaredAt + "92: stagedBeforeDurable: nowhere is not a function of one argument"},
		{"same-step property arity", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "store", "putStores").Holds = "disk.property.durableStays"
		}, admDeclaredAt + "137: store.putStores: disk.property.durableStays is not a function of one argument"},
		{"transition property arity", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "disk", "durableStays").Holds = "disk.property.putAccepted"
		}, admDeclaredAt + "133: disk.durableStays: disk.property.putAccepted is not a function of two arguments"},
		{"progress from arity", "declarations", func(m *modelirspb.Model) {
			m.GetProgress()[0].From = "disk.property.durableStays"
		}, admDeclaredAt + "111: disk.durableEventually: disk.property.durableStays is not a function of one argument"},
		{"progress to", "declarations", func(m *modelirspb.Model) {
			m.GetProgress()[0].To = "nowhere"
		}, admDeclaredAt + "111: disk.durableEventually: nowhere is not a function of one argument"},
		{"visible arity", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").GetRefines().Visible = "disk.property.durableStays"
		}, admDeclaredAt + "99: disk: disk.property.durableStays is not a function of one argument"},
		{"visible outcomes", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").GetRefines().VisibleOutcomes = "nowhere"
		}, admDeclaredAt + "99: disk: nowhere is not a function of one argument"},
	})
}

func TestAdmissionRejectsCrossedTypes(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"step state", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").GetSteps()[0].Function = "store.put"
		}, admDeclaredAt + "107: store.put steps put, so its parameter s takes fixture.declarations.Disk, not fixture.declarations.Store"},
		{"step input", "channels", func(m *modelirspb.Model) {
			admMachine(m, "relay").GetSteps()[0].Function = admChannelsPkg + "flashStep"
		}, admChannelsAt + "55: " + admChannelsPkg + "flashStep steps talk, so its parameter s takes fixture.channels.Note, not fixture.channels.Signal"},
		{"step input of a range", "channels", func(m *modelirspb.Model) {
			function(m, "counted").GetParams()[1].Type = &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Bool{Bool: &modelirspb.Empty{}}}
		}, admChannelsAt + "85: " + admChannelsPkg + "counted steps tallyDelivery, so its parameter n takes 0..2, not Boolean"},
		{"watched state", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "store").Monitors = []string{admDeclaredPkg + "storedOnce"}
		}, admDeclaredAt + "37: store names monitor storedOnce, whose next takes fixture.declarations.Disk, not the state fixture.declarations.Store"},
		{"monitor next state", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").Next = admDeclaredPkg + "endsDurable.next"
		}, admDeclaredAt + "82: monitor storedOnce's next takes Boolean, not its state fixture.declarations.Seen"},
		{"monitor violated state", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").Violated = admDeclaredPkg + "endsDurable.violated"
		}, admDeclaredAt + "82: monitor storedOnce's violated takes Boolean, not its state fixture.declarations.Seen"},
		{"delivered message", "channels", func(m *modelirspb.Model) {
			admAction(m, admChannelsPkg+"wire.deliver").GetInputs()[0].Type = &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Named{Named: "fixture.channels.Signal"}}
		}, admChannelsAt + "14: " + admChannelsPkg + "wire.deliver delivers " + admChannelsPkg + "wire, so it takes one input of fixture.channels.Note"},
		{"lost messages", "channels", func(m *modelirspb.Model) {
			a := admAction(m, admChannelsPkg+"radio.lose")
			a.Inputs = append(a.Inputs, a.GetInputs()[0])
		}, admChannelsAt + "16: " + admChannelsPkg + "radio.lose loses " + admChannelsPkg + "radio, so it takes one input of fixture.channels.Signal"},
		{"member field", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "pair").GetMembers()[1].Field = "middle"
		}, admDeclaredAt + "123: member middle of pair is no field of fixture.declarations.Pair"},
		{"member state", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "pair").GetMembers()[1].Machine = "disk"
		}, admDeclaredAt + "123: member back of pair holds fixture.declarations.Store, not the state fixture.declarations.Disk of disk"},
	})
}

func TestAdmissionRejectsDuplicates(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"type", "declarations", func(m *modelirspb.Model) {
			m.Types = append(m.Types, proto.Clone(admType(m, "fixture.declarations.Store")).(*modelirspb.Type))
		}, admDeclaredAt + "15: two types named fixture.declarations.Store"},
		{"function", "declarations", func(m *modelirspb.Model) {
			m.Functions = append(m.Functions, proto.Clone(function(m, "store.put")).(*modelirspb.Function))
		}, admDeclaredAt + "43: two functions named store.put"},
		{"action", "declarations", func(m *modelirspb.Model) {
			m.Actions = append(m.Actions, proto.Clone(admAction(m, admDeclaredPkg+"put")).(*modelirspb.Action))
		}, admDeclaredAt + "23: two actions with id " + admDeclaredPkg + "put"},
		{"machine", "declarations", func(m *modelirspb.Model) {
			m.Machines = append(m.Machines, proto.Clone(admMachine(m, "store")).(*modelirspb.Machine))
		}, admDeclaredAt + "37: two machines named store"},
		{"machine and composition", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "store").Name = "pair"
		}, admDeclaredAt + "123: a machine and a composition are both named pair"},
		{"channel", "channels", func(m *modelirspb.Model) {
			m.Channels = append(m.Channels, proto.Clone(admChannel(m, "wire")).(*modelirspb.Channel))
		}, admChannelsAt + "14: two channels with id " + admChannelsPkg + "wire"},
		{"monitor id", "declarations", func(m *modelirspb.Model) {
			m.Monitors = append(m.Monitors, proto.Clone(admMonitor(m, "storedOnce")).(*modelirspb.Monitor))
		}, admDeclaredAt + "82: two monitors with id " + admDeclaredPkg + "storedOnce"},
		{"monitor name", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "endsDurable").Name = "storedOnce"
		}, admDeclaredAt + "82: two monitors named storedOnce"},
		{"assumption", "declarations", func(m *modelirspb.Model) {
			m.Assumptions = append(m.Assumptions, proto.Clone(m.GetAssumptions()[1]).(*modelirspb.Assumption))
		}, admDeclaredAt + "27: two assumptions with id " + admDeclaredPkg + "storeOpaque"},
		{"hole", "declarations", func(m *modelirspb.Model) {
			m.Holes = append(m.Holes, proto.Clone(m.GetHoles()[0]).(*modelirspb.Hole))
		}, admDeclaredAt + "29: two holes with id " + admDeclaredPkg + "crashUnmodeled"},
		{"composition", "declarations", func(m *modelirspb.Model) {
			m.Compositions = append(m.Compositions, proto.Clone(admComposition(m, "pair")).(*modelirspb.Composition))
		}, admDeclaredAt + "123: two compositions named pair"},
		{"property", "declarations", func(m *modelirspb.Model) {
			m.Properties = append(m.Properties, proto.Clone(admProperty(m, "store", "putStores")).(*modelirspb.Property))
		}, admDeclaredAt + "137: two Properties named putStores on store"},
		{"scenario", "declarations", func(m *modelirspb.Model) {
			m.Scenarios = append(m.Scenarios, proto.Clone(admScenario(m, "store", "putOnce")).(*modelirspb.Scenario))
		}, admDeclaredAt + "148: two Scenarios named putOnce on store"},
		{"query", "declarations", func(m *modelirspb.Model) {
			m.Queries = append(m.Queries, proto.Clone(admQuery(m, "putStores")).(*modelirspb.Query))
		}, admDeclaredAt + "162: two Queries named putStores"},
		{"progress", "declarations", func(m *modelirspb.Model) {
			m.Progress = append(m.Progress, proto.Clone(m.GetProgress()[0]).(*modelirspb.Progress))
		}, admDeclaredAt + "111: two progress claims named durableEventually on disk"},
		{"record field", "channels", func(m *modelirspb.Model) {
			r := admType(m, "fixture.channels.Relay").GetRecord()
			r.Fields = append(r.Fields, proto.Clone(r.GetFields()[0]).(*modelirspb.Field))
		}, admChannelsAt + "22: fixture.channels.Relay has two fields named heard"},
		{"enum case", "channels", func(m *modelirspb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, proto.Clone(e.GetCases()[0]).(*modelirspb.Case))
		}, admChannelsAt + "8: fixture.channels.Note has two cases named ping"},
		{"case field", "presence", func(m *modelirspb.Model) {
			sent := admType(m, "fixture.presence.Report").GetEnum().GetCases()[1]
			sent.Fields = append(sent.Fields, proto.Clone(sent.GetFields()[0]).(*modelirspb.Field))
		}, admPresenceAt + "12: case sent of fixture.presence.Report has two fields named result"},
		{"parameter", "declarations", func(m *modelirspb.Model) {
			f := function(m, "countStored")
			f.Params[1].Name = "seen"
		}, admDeclaredAt + "75: " + admDeclaredPkg + "countStored has two parameters named seen"},
	})
}

func admInt() *modelirspb.TypeRef {
	return &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Int{Int: &modelirspb.Empty{}}}
}

func admList(of string) *modelirspb.TypeRef {
	return &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_List{List: &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Named{Named: of}}}}
}

func TestAdmissionRejectsInfiniteCatalogsAndBounds(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"monitor state of Int", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").State = admInt()
		}, admDeclaredAt + "82: monitor storedOnce needs a state of a finite type, not Int"},
		{"monitor state of a list", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").State = admList("fixture.declarations.Seen")
		}, admDeclaredAt + "82: monitor storedOnce needs a state of a finite type, not List[fixture.declarations.Seen]"},
		{"monitor state of a channel", "declarations", func(m *modelirspb.Model) {
			admMonitor(m, "storedOnce").State = &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: "wire"}}
		}, admDeclaredAt + "82: monitor storedOnce needs a state of a finite type, not channel wire"},
		{"channel message of Int", "channels", func(m *modelirspb.Model) {
			admChannel(m, "tally").Message = admInt()
		}, admChannelsAt + "64: channel " + admChannelsPkg + "tally needs a message of a finite type, not Int"},
		{"channel message of a list", "channels", func(m *modelirspb.Model) {
			admChannel(m, "wire").Message = admList("fixture.channels.Note")
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire needs a message of a finite type, not List[fixture.channels.Note]"},
		{"channel capacity", "channels", func(m *modelirspb.Model) {
			admChannel(m, "wire").Capacity = 0
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire has capacity 0, below 1"},
		{"channel duplicates", "channels", func(m *modelirspb.Model) {
			admChannel(m, "wire").Duplicates = -1
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire has -1 duplicates, below 0"},
		{"limits steps", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").GetLimits().Steps = -1
		}, admDeclaredAt + "160: query durableStays limits steps to -1, below 0"},
		{"limits actions", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").GetLimits().Actions = -2
		}, admDeclaredAt + "160: query durableStays limits actions to -2, below 0"},
		{"limits search", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "durableStays").GetLimits().Search = -3
		}, admDeclaredAt + "160: query durableStays limits search to -3, below 0"},
		{"progress within", "declarations", func(m *modelirspb.Model) {
			m.GetProgress()[0].Within = 0
		}, admDeclaredAt + "111: progress claim durableEventually of disk is within 0 steps, fewer than one"},
	})
}

func TestAdmissionRejectsChannelMisuse(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"one channel in two fields", "channels", func(m *modelirspb.Model) {
			admType(m, "fixture.channels.Relay").GetRecord().GetFields()[2].Type = &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: admChannelsPkg + "wire"}}
		}, admChannelsAt + "22: fixture.channels.Relay holds channel " + admChannelsPkg + "wire in two fields"},
		{"delivery of a channel the state does not hold", "channels", func(m *modelirspb.Model) {
			admMachine(m, "tallying").GetSteps()[1].Action = admChannelsPkg + "wire.deliver"
		}, admChannelsAt + "85: tallying binds a delivery of channel " + admChannelsPkg + "wire, which its state fixture.channels.Tally does not hold"},
		{"loss of a channel the state does not hold", "channels", func(m *modelirspb.Model) {
			admMachine(m, "tallying").GetSteps()[1].Action = admChannelsPkg + "radio.lose"
		}, admChannelsAt + "85: tallying binds a loss of channel " + admChannelsPkg + "radio, which its state fixture.channels.Tally does not hold"},
		{"lossy channel with no loss", "channels", func(m *modelirspb.Model) {
			relay := admMachine(m, "relay")
			relay.Steps = relay.GetSteps()[:4]
		}, admChannelsAt + "51: relay holds lossy channel " + admChannelsPkg + "radio but binds no loss of it"},
		{"loss of a reliable channel", "channels", func(m *modelirspb.Model) {
			admChannel(m, "radio").Lossy = false
		}, admChannelsAt + "16: " + admChannelsPkg + "radio.lose loses channel " + admChannelsPkg + "radio, which is not lossy"},
	})
}

func TestAdmissionRejectsReadingsWithoutARefinement(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"visible facts of no refinement", "declarations", func(m *modelirspb.Model) {
			admMachine(m, "disk").GetRefines().Product = ""
		}, admDeclaredAt + "99: disk names what a refined machine sees but refines none"},
		{"replacement its member does not refine", "declarations", func(m *modelirspb.Model) {
			admComposition(m, "detailedPair").GetMembers()[1].Replaces = "disk"
		}, admDeclaredAt + "128: member back of detailedPair replaces disk, which disk does not refine"},
		{"property of another machine", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "putStores").Scenario = &modelirspb.ClaimRef{Machine: "disk", Name: "putThenFlush"}
		}, admDeclaredAt + "162: query putStores pairs a Property of store with a Scenario of disk"},
		{"through what the scenario machine does not refine", "declarations", func(m *modelirspb.Model) {
			admQuery(m, "putStoresThroughDisk").Scenario = &modelirspb.ClaimRef{Machine: "pair", Name: "any"}
		}, admDeclaredAt + "163: query putStoresThroughDisk pairs a Property of store with a Scenario of pair"},
		{"product property without through", "admission", func(m *modelirspb.Model) {
			admQuery(m, "currentAdmission.product.pausedIsNotDispatched").Through = false
		}, admAdmissionAt + "202: query currentAdmission.product.pausedIsNotDispatched pairs a Property of activityProduct with a Scenario of currentAdmission"},
	})
}

// admCall is a call of name with x as its one argument, at the position of at.
func admCall(at *modelirspb.Expr, name string, x *modelirspb.Expr) *modelirspb.Expr {
	return &modelirspb.Expr{Position: at.GetPosition(), Kind: &modelirspb.Expr_Call{Call: &modelirspb.Call{Function: name, Args: []*modelirspb.Expr{x}}}}
}

func admVar(at *modelirspb.Expr, name string) *modelirspb.Expr {
	return &modelirspb.Expr{Position: at.GetPosition(), Kind: &modelirspb.Expr_Var{Var: name}}
}

func TestAdmissionRejectsRecursion(t *testing.T) {
	oneMore := admAdmissionPkg + "oneMore"
	runAdmissionCases(t, []admissionCase{
		{"direct", "admission", func(m *modelirspb.Model) {
			f := function(m, "oneMore")
			f.Body = admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))
		}, admAdmissionAt + "63: " + oneMore + " calls itself"},
		{"through others", "admission", func(m *modelirspb.Model) {
			f := function(m, "Admission$package$.admitted")
			f.Body = admCall(f.GetBody(), admAdmissionPkg+"admitCurrent", admVar(f.GetBody(), "s"))
		}, admAdmissionAt + "84: " + admAdmissionPkg + "admitCurrent calls itself through " + admAdmissionPkg + "admitted"},
		{"in a lambda", "admission", func(m *modelirspb.Model) {
			f := function(m, "oneMore")
			f.Body = &modelirspb.Expr{Position: f.GetBody().GetPosition(), Kind: &modelirspb.Expr_Lambda{Lambda: &modelirspb.Lambda{
				Params: f.GetParams(), Body: admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))}}}
		}, admAdmissionAt + "63: " + oneMore + " calls itself"},
		{"in a precondition", "admission", func(m *modelirspb.Model) {
			f := function(m, "oneMore")
			f.Requires = admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))
		}, admAdmissionAt + "63: " + oneMore + " calls itself"},
	})
	m := admFixture(t, "admission")
	f := function(m, "Admission$package$.admitted")
	f.Body = admCall(f.GetBody(), admAdmissionPkg+"admitCurrent", admVar(f.GetBody(), "s"))
	require.ErrorContains(t, Validate(m), admAdmissionAt+"73: "+admAdmissionPkg+"admitted calls itself through "+admAdmissionPkg+"admitCurrent",
		"every function on the cycle is reported")
}

func TestAdmissionReportsEveryProblem(t *testing.T) {
	m := admFixture(t, "channels")
	m.Version = 2
	admChannel(m, "wire").Capacity = 0
	admChannel(m, "radio").Lossy = false
	admMachine(m, "tallying").GetSteps()[0].Function = "nowhere"
	err := Validate(m)
	require.ErrorContains(t, err, "model/scalav2: fixture.channels.Channels$package$.relay, fixture.channels.Channels$package$.tallying: "+
		"version 2 is not a version this reader knows")
	require.ErrorContains(t, err, admChannelsAt+"14: channel "+admChannelsPkg+"wire has capacity 0, below 1")
	require.ErrorContains(t, err, admChannelsAt+"16: "+admChannelsPkg+"radio.lose loses channel "+admChannelsPkg+"radio, which is not lossy")
	require.ErrorContains(t, err, admChannelsAt+"85: no function nowhere")
	require.Len(t, strings.Split(err.Error(), "\n"), 4, "each problem once, and nothing else")
}

func admLiteral(at *modelirspb.Expr, v *modelirspb.Value) *modelirspb.Expr {
	return &modelirspb.Expr{Position: at.GetPosition(), Kind: &modelirspb.Expr_Literal{Literal: v}}
}

func admEnum(typ, c string) *modelirspb.Value {
	return &modelirspb.Value{Kind: &modelirspb.Value_Enum{Enum: &modelirspb.EnumValue{Type: typ, Case: c}}}
}

func admIntValue(n int64) *modelirspb.Value {
	return &modelirspb.Value{Kind: &modelirspb.Value_Int{Int: n}}
}

func TestAdmissionRejectsAStepThatReturnsNoSteps(t *testing.T) {
	putStep := admDeclaredPkg + "putStep"
	runAdmissionCases(t, []admissionCase{
		{"a Boolean", "declarations", func(m *modelirspb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = admLiteral(f.GetBody(), &modelirspb.Value{Kind: &modelirspb.Value_Bool{Bool: false}})
		}, admDeclaredAt + "59: " + putStep + " returns a Boolean, not a list of steps"},
		{"a record", "declarations", func(m *modelirspb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = &modelirspb.Expr{Position: f.GetBody().GetPosition(), Kind: &modelirspb.Expr_Construct{Construct: &modelirspb.Construct{
				Type: "fixture.declarations.Disk", Args: []*modelirspb.Expr{admLiteral(f.GetBody(), admEnum("fixture.declarations.Stage", "empty"))}}}}
		}, admDeclaredAt + "59: " + putStep + " returns a fixture.declarations.Disk, not a list of steps"},
		{"a list of records", "declarations", func(m *modelirspb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = &modelirspb.Expr{Position: f.GetBody().GetPosition(), Kind: &modelirspb.Expr_List{List: &modelirspb.ListOf{Items: []*modelirspb.Expr{
				admLiteral(f.GetBody(), admEnum("fixture.declarations.Stage", "empty"))}}}}
		}, admDeclaredAt + "59: " + putStep + " returns a list holding a fixture.declarations.Stage, not a list of steps"},
	})
}

func TestAdmissionRejectsAnActionThatBothDeliversAndLoses(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"delivers and loses", "channels", func(m *modelirspb.Model) {
			admAction(m, admChannelsPkg+"radio.lose").Delivers = admChannelsPkg + "radio"
		}, admChannelsAt + "16: " + admChannelsPkg + "radio.lose both delivers and loses " + admChannelsPkg + "radio; an action does one"},
	})
}

func TestAdmissionRejectsACatalogThatContainsItself(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"a record of itself", "channels", func(m *modelirspb.Model) {
			admType(m, "fixture.channels.Relay").GetRecord().GetFields()[0].Type = named("fixture.channels.Relay")
		}, admChannelsAt + "22: type fixture.channels.Relay has no finite catalog: it contains itself"},
		{"a message holding its channel", "channels", func(m *modelirspb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, &modelirspb.Case{Name: "echo", Fields: []*modelirspb.Field{{Name: "back",
				Type: &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: admChannelsPkg + "wire"}}}}})
		}, admChannelsAt + "8: type fixture.channels.Note has no finite catalog: it contains itself through channel " + admChannelsPkg + "wire"},
		{"a channel holding its message", "channels", func(m *modelirspb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, &modelirspb.Case{Name: "echo", Fields: []*modelirspb.Field{{Name: "back",
				Type: &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Channel{Channel: admChannelsPkg + "wire"}}}}})
		}, admChannelsAt + "14: channel " + admChannelsPkg + "wire has no finite catalog: it contains itself through type fixture.channels.Note"},
	})
}

// A Scenario on tallying, declared beside the lifted ones, that delivers message n.
func admTallyScenario(m *modelirspb.Model, n int64) {
	at := &modelirspb.Position{File: "generic", Line: 1}
	m.Scenarios = append(m.Scenarios, &modelirspb.Scenario{Machine: "tallying", Name: "counts", Position: at,
		Start:   proto.Clone(admMachine(m, "tallying").GetStarts()[0]).(*modelirspb.Expr),
		Actions: []*modelirspb.ActionClass{{Action: admChannelsPkg + "tally.deliver", Inputs: []*modelirspb.Value{admIntValue(n)}}}})
}

func TestAdmissionRejectsMisaddressedSelectors(t *testing.T) {
	put, flush, control := admDeclaredPkg+"put", admDeclaredPkg+"flush", "temporal.standaloneactivity.Model$package$.control"
	runAdmissionCases(t, []admissionCase{
		{"when_class input the action does not take", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "store", "putStores").GetWhenClass().Inputs = []*modelirspb.Value{admIntValue(1)}
		}, admDeclaredAt + "137: store.putStores: " + put + " takes 0 inputs, not 1"},
		{"when_class of an action the machine does not bind", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "store", "putStores").GetWhenClass().Action = flush
		}, admDeclaredAt + "137: store.putStores: store binds no action " + flush},
		{"when_action the machine does not bind", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "disk", "putAccepted").When = &modelirspb.Property_WhenAction{WhenAction: "nothing"}
		}, admDeclaredAt + "139: disk.putAccepted: disk binds no action nothing"},
		{"when_action on a composition", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &modelirspb.Property_WhenAction{WhenAction: "put"}
		}, admDeclaredAt + "143: detailedPair.frontHeld: a Property of a composition is about every step"},
		{"when_class on a composition", "declarations", func(m *modelirspb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &modelirspb.Property_WhenClass{WhenClass: &modelirspb.ActionClass{Action: put}}
		}, admDeclaredAt + "143: detailedPair.frontHeld: a Property of a composition is about every step"},
		{"scenario input the action does not take", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "store", "putOnce").GetActions()[0].Inputs = []*modelirspb.Value{admIntValue(1)}
		}, admDeclaredAt + "148: store.putOnce: " + put + " takes 0 inputs, not 1"},
		{"scenario input of a crossed type", "admission", func(m *modelirspb.Model) {
			admScenario(m, "currentAdmission", "staleDeliveryAfterPause").GetActions()[1].Inputs[0] = admEnum(
				"fixture.specimens.admission.AdmissionPhase", "paused")
		}, admAdmissionAt + "188: currentAdmission.staleDeliveryAfterPause: " + control + " takes a temporal.standaloneactivity.Control for control, not paused"},
		{"scenario input outside its range", "channels", func(m *modelirspb.Model) {
			admTallyScenario(m, 3)
		}, "generic:1: tallying.counts: " + admChannelsPkg + "tally.deliver takes a 0..2 for message, not 3"},
		{"scenario action the machine does not bind", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "store", "putOnce").GetActions()[0].Action = flush
		}, admDeclaredAt + "148: store.putOnce: store binds no action " + flush},
		{"machine scenario start of another type", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "store", "putOnce").Start = proto.Clone(admScenario(m, "disk", "putThenFlush").GetStart()).(*modelirspb.Expr)
		}, admDeclaredAt + "148: store.putOnce starts at empty, which is no fixture.declarations.Store"},
		{"composition scenario start of another type", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "detailedPair", "bothPut").Start = proto.Clone(admScenario(m, "pair", "any").GetStart()).(*modelirspb.Expr)
		}, admDeclaredAt + "152: detailedPair.bothPut starts at nothing-nothing, which is no fixture.declarations.DetailedPair"},
		{"composition scenario key of no class", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "detailedPair", "bothPut").Keys = []string{"putNone"}
		}, admDeclaredAt + "152: detailedPair.bothPut: detailedPair has no class putNone"},
		{"composition scenario key of an action its member does not bind", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "detailedPair", "bothPut").Keys = []string{"front_flush"}
		}, admDeclaredAt + "152: detailedPair.bothPut: detailedPair has no class front_flush"},
		{"composition scenario of actions", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "detailedPair", "bothPut").Actions = []*modelirspb.ActionClass{{Action: put}}
		}, admDeclaredAt + "152: detailedPair.bothPut: a Scenario of a composition schedules its class keys, not actions"},
		{"machine scenario of keys", "declarations", func(m *modelirspb.Model) {
			admScenario(m, "store", "putOnce").Keys = []string{"put"}
		}, admDeclaredAt + "148: store.putOnce: a Scenario of a machine schedules action classes, not keys"},
	})
}

func TestAdmissionAdmitsWellAddressedSelectors(t *testing.T) {
	m := admFixture(t, "declarations")
	admScenario(m, "detailedPair", "bothPut").Keys = []string{"putBoth", "front_put", "back_flush", "back_crash"}
	require.NoError(t, Validate(m))
	m = admFixture(t, "channels")
	admTallyScenario(m, 2)
	require.NoError(t, Validate(m))
}
