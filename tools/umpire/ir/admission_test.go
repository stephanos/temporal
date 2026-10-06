package ir

// What a reader rejects before any check (model/SEMANTICS.md, Admission), each case one
// mutation of a source-derived Model, reported at the position the lifter recorded for the
// declaration or node concerned.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

const (
	admLifts        = "model/irgen/testdata/lifts/"
	admChannelsAt   = admLifts + "Channels.scala:"
	admDeclaredAt   = admLifts + "Declarations.scala:"
	admPresenceAt   = admLifts + "Presence.scala:"
	admAdmissionAt  = admLifts + "Admission.scala:"
	admChannelsPkg  = "fixture.channels.Channels$package$."
	admChannelsID   = "fixture.channels."
	admDeclaredPkg  = "fixture.declarations.Declarations$package$."
	admDeclaredID   = "fixture.declarations."
	admAdmissionPkg = "fixture.specimens.admission.Admission$package$."
	admAdmissionID  = "fixture.specimens.admission."
)

type admissionCase struct {
	name    string
	fixture string
	mutate  func(m *umpirespb.Model)
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

func admFixture(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	if name == "nexus" {
		return proto.Clone(load(t)).(*umpirespb.Model)
	}
	return proto.Clone(lifted(t, name)).(*umpirespb.Model)
}

func admMachine(m *umpirespb.Model, name string) *umpirespb.Machine {
	for _, mm := range m.GetMachines() {
		if mm.GetName() == name {
			return mm
		}
	}
	return nil
}

func admAction(m *umpirespb.Model, id string) *umpirespb.Action {
	for _, a := range m.GetActions() {
		if a.GetId() == id {
			return a
		}
	}
	return nil
}

func admType(m *umpirespb.Model, name string) *umpirespb.Type {
	for _, t := range m.GetTypes() {
		if t.GetName() == name {
			return t
		}
	}
	return nil
}

func admChannel(m *umpirespb.Model, name string) *umpirespb.Channel {
	for _, c := range m.GetChannels() {
		if c.GetName() == name {
			return c
		}
	}
	return nil
}

func admMonitor(m *umpirespb.Model, name string) *umpirespb.Monitor {
	for _, mo := range m.GetMonitors() {
		if mo.GetName() == name {
			return mo
		}
	}
	return nil
}

func admComposition(m *umpirespb.Model, name string) *umpirespb.Composition {
	for _, c := range m.GetCompositions() {
		if c.GetName() == name {
			return c
		}
	}
	return nil
}

func admQuery(m *umpirespb.Model, name string) *umpirespb.Query {
	for _, q := range m.GetQueries() {
		if q.GetName() == name {
			return q
		}
	}
	return nil
}

func admProperty(m *umpirespb.Model, machine, name string) *umpirespb.Property {
	for _, p := range m.GetProperties() {
		if p.GetMachine() == machine && p.GetName() == name {
			return p
		}
	}
	return nil
}

func admScenario(m *umpirespb.Model, machine, name string) *umpirespb.Scenario {
	for _, s := range m.GetScenarios() {
		if s.GetMachine() == machine && s.GetName() == name {
			return s
		}
	}
	return nil
}

// admFirst is the first expression under x, in evaluation order, that accepts.
func admFirst(x *umpirespb.Expr, accept func(*umpirespb.Expr) bool) *umpirespb.Expr {
	if x == nil {
		return nil
	}
	if accept(x) {
		return x
	}
	var under []*umpirespb.Expr
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Field:
		under = []*umpirespb.Expr{k.Field.GetBase()}
	case *umpirespb.Expr_Call:
		under = k.Call.GetArgs()
	case *umpirespb.Expr_Construct:
		under = k.Construct.GetArgs()
	case *umpirespb.Expr_Copy:
		under = []*umpirespb.Expr{k.Copy.GetBase()}
		for _, u := range k.Copy.GetUpdates() {
			under = append(under, u.GetValue())
		}
	case *umpirespb.Expr_Unary:
		under = []*umpirespb.Expr{k.Unary.GetOperand()}
	case *umpirespb.Expr_Binary:
		under = []*umpirespb.Expr{k.Binary.GetLeft(), k.Binary.GetRight()}
	case *umpirespb.Expr_If:
		under = []*umpirespb.Expr{k.If.GetCondition(), k.If.GetThen(), k.If.GetElse()}
	case *umpirespb.Expr_Match:
		under = []*umpirespb.Expr{k.Match.GetScrutinee()}
		for _, c := range k.Match.GetCases() {
			under = append(under, c.GetGuard(), c.GetBody())
		}
	case *umpirespb.Expr_Let:
		under = []*umpirespb.Expr{k.Let.GetValue(), k.Let.GetBody()}
	case *umpirespb.Expr_List:
		under = k.List.GetItems()
	case *umpirespb.Expr_Lambda:
		under = []*umpirespb.Expr{k.Lambda.GetBody()}
	case *umpirespb.Expr_Inbox:
		under = []*umpirespb.Expr{k.Inbox.GetContents(), k.Inbox.GetMessage()}
	default:
	}
	for _, u := range under {
		if found := admFirst(u, accept); found != nil {
			return found
		}
	}
	return nil
}

func admBody(m *umpirespb.Model, name string) *umpirespb.Expr {
	return function(m, name).GetBody()
}

func isBinary(x *umpirespb.Expr) bool { return x.GetBinary() != nil }
func isInbox(x *umpirespb.Expr) bool  { return x.GetInbox() != nil }
func isHole(x *umpirespb.Expr) bool   { return x.GetHole() != "" }

func TestAdmissionAdmitsTheSourceDerivedModels(t *testing.T) {
	for _, name := range []string{"admission", "channels", "declarations", "presence", "nexus"} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, Validate(admFixture(t, name)))
		})
	}
}

func TestAdmissionRejectsUnknownVersionsAndConstructs(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"version", "channels", func(m *umpirespb.Model) { m.Version = 1 },
			"model: fixture.channels.Relay, fixture.channels.Tallying: " +
				"version 1 is not a version this reader knows"},
		{"binary op unspecified", "declarations", func(m *umpirespb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().Op = umpirespb.Binary_OP_UNSPECIFIED
		}, admDeclaredAt + "67: no binary operator 0"},
		{"binary op outside the enum", "declarations", func(m *umpirespb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().Op = 99
		}, admDeclaredAt + "67: no binary operator 99"},
		{"unary op outside the enum", "declarations", func(m *umpirespb.Model) {
			cond := admFirst(admBody(m, "crashStep"), isBinary)
			cond.Kind = &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: 7, Operand: cond.GetBinary().GetLeft()}}
		}, admDeclaredAt + "67: no unary operator 7"},
		{"inbox op unspecified", "channels", func(m *umpirespb.Model) {
			admFirst(admBody(m, "talkStep"), isInbox).GetInbox().Op = umpirespb.Inbox_OP_UNSPECIFIED
		}, admChannelsAt + "37: no inbox operation 0"},
		{"pattern of no kind", "declarations", func(m *umpirespb.Model) {
			admBody(m, "putStep").GetMatch().GetCases()[0].Pattern = &umpirespb.Pattern{}
		}, admDeclaredAt + "56: a pattern of no known kind"},
		{"value of no kind", "declarations", func(m *umpirespb.Model) {
			admFirst(admBody(m, "crashStep"), isBinary).GetBinary().GetRight().GetLiteral().Kind = nil
		}, admDeclaredAt + "67: a value of no known kind"},
		{"channel order unspecified", "channels", func(m *umpirespb.Model) {
			admChannel(m, "wire").Order = umpirespb.Channel_ORDER_UNSPECIFIED
		}, admChannelsAt + "14: channel " + admChannelsID + "wire has no known order"},
		{"channel order outside the enum", "channels", func(m *umpirespb.Model) {
			admChannel(m, "wire").Order = 5
		}, admChannelsAt + "14: channel " + admChannelsID + "wire has no known order"},
		{"query form unspecified", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").Form = umpirespb.Query_FORM_UNSPECIFIED
		}, admDeclaredAt + "163: query durableStays has no known form"},
		{"query form outside the enum", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").Form = 3
		}, admDeclaredAt + "163: query durableStays has no known form"},
		{"monitor without an evaluation point", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "endsDurable").Evaluate = nil
		}, admDeclaredAt + "82: monitor endsDurable has no evaluation point"},
	})
}

func TestAdmissionRejectsUndeclaredNamesAndArities(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"channel type", "channels", func(m *umpirespb.Model) {
			admType(m, "fixture.channels.RelayState").GetRecord().GetFields()[1].Type = &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: "nowhere"}}
		}, admChannelsAt + "23: no channel nowhere"},
		{"inbox channel", "channels", func(m *umpirespb.Model) {
			admFirst(admBody(m, "talkStep"), isInbox).GetInbox().Channel = "nowhere"
		}, admChannelsAt + "37: no channel nowhere"},
		{"delivered channel", "channels", func(m *umpirespb.Model) {
			admAction(m, admChannelsID+"wire.deliver").Delivers = "nowhere"
		}, admChannelsAt + "14: no channel nowhere"},
		{"lost channel", "channels", func(m *umpirespb.Model) {
			admAction(m, admChannelsID+"radio.lose").Loses = "nowhere"
		}, admChannelsAt + "16: no channel nowhere"},
		{"machine monitor", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").Monitors[0] = "nowhere"
		}, admDeclaredAt + "93: no monitor nowhere"},
		{"machine assumption", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "store").Assumes[0] = "nowhere"
		}, admDeclaredAt + "34: no assumption nowhere"},
		{"progress assumption", "declarations", func(m *umpirespb.Model) {
			m.GetProgress()[0].Assumptions[0] = "nowhere"
		}, admDeclaredAt + "113: no assumption nowhere"},
		{"fair action", "declarations", func(m *umpirespb.Model) {
			m.GetAssumptions()[0].Fair[0] = "nowhere"
		}, admDeclaredAt + "26: no action nowhere"},
		{"hole", "declarations", func(m *umpirespb.Model) {
			admFirst(admBody(m, "crashStep"), isHole).Kind = &umpirespb.Expr_Hole{Hole: "nowhere"}
		}, admDeclaredAt + "67: no hole nowhere"},
		{"refined machine", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").GetRefines().Product = "nowhere"
		}, admDeclaredAt + "93: no machine nowhere"},
		{"member machine", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "pair").GetMembers()[0].Machine = "nowhere"
		}, admDeclaredAt + "124: no machine nowhere"},
		{"replaced machine", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "detailedPair").GetMembers()[1].Replaces = "nowhere"
		}, admDeclaredAt + "129: no machine nowhere"},
		{"sync member", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "pair").GetSyncs()[0].GetFirst().Member = "middle"
		}, admDeclaredAt + "124: sync putBoth of pair has no member middle"},
		{"sync action", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "pair").GetSyncs()[0].GetSecond().Action = "flush"
		}, admDeclaredAt + "124: sync putBoth of pair: store binds no action flush"},
		{"property machine", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "store", "putStores").Machine = "nowhere"
		}, admDeclaredAt + "141: no machine or composition nowhere"},
		{"scenario machine", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "store", "putOnce").Machine = "nowhere"
		}, admDeclaredAt + "152: no machine or composition nowhere"},
		{"progress machine", "declarations", func(m *umpirespb.Model) {
			m.GetProgress()[0].Machine = "nowhere"
		}, admDeclaredAt + "113: no machine nowhere"},
		{"query property", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").GetProperty().Name = "nothing"
		}, admDeclaredAt + "163: query durableStays: no Property nothing of disk"},
		{"query scenario", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").GetScenario().Name = "nothing"
		}, admDeclaredAt + "163: query durableStays: no Scenario nothing of disk"},
		{"monitor next arity", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").Next = admDeclaredID + "storedOnce.violated"
		}, admDeclaredAt + "79: storedOnce: " + admDeclaredID + "storedOnce.violated is not a function of three arguments"},
		{"monitor violated arity", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").Violated = admDeclaredPkg + "countStored"
		}, admDeclaredAt + "79: storedOnce: " + admDeclaredPkg + "countStored is not a function of one argument"},
		{"monitor after", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "stagedBeforeDurable").Evaluate = &umpirespb.Monitor_After{After: "nowhere"}
		}, admDeclaredAt + "87: stagedBeforeDurable: nowhere is not a function of one argument"},
		{"same-step property arity", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "store", "putStores").Holds = "disk.property.durableStays"
		}, admDeclaredAt + "141: store.putStores: disk.property.durableStays is not a function of one argument"},
		{"transition property arity", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "disk", "durableStays").Holds = "disk.property.putAccepted"
		}, admDeclaredAt + "137: disk.durableStays: disk.property.putAccepted is not a function of two arguments"},
		{"progress from arity", "declarations", func(m *umpirespb.Model) {
			m.GetProgress()[0].From = "disk.property.durableStays"
		}, admDeclaredAt + "113: disk.durableEventually: disk.property.durableStays is not a function of one argument"},
		{"progress to", "declarations", func(m *umpirespb.Model) {
			m.GetProgress()[0].To = "nowhere"
		}, admDeclaredAt + "113: disk.durableEventually: nowhere is not a function of one argument"},
		{"visible arity", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").GetRefines().Visible = "disk.property.durableStays"
		}, admDeclaredAt + "93: disk: disk.property.durableStays is not a function of one argument"},
		{"visible outcomes", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").GetRefines().VisibleOutcomes = "nowhere"
		}, admDeclaredAt + "93: disk: nowhere is not a function of one argument"},
	})
}

func TestAdmissionRejectsCrossedTypes(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"step state", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").GetSteps()[0].Function = "store.put"
		}, admDeclaredAt + "110: store.put steps put, so its parameter s takes fixture.declarations.DiskState, not fixture.declarations.StoreState"},
		{"step input", "channels", func(m *umpirespb.Model) {
			admMachine(m, "relay").GetSteps()[0].Function = admChannelsPkg + "flashStep"
		}, admChannelsAt + "59: " + admChannelsPkg + "flashStep steps talk, so its parameter s takes fixture.channels.Note, not fixture.channels.Signal"},
		{"step input of a range", "channels", func(m *umpirespb.Model) {
			function(m, "counted").GetParams()[1].Type = &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &umpirespb.Empty{}}}
		}, admChannelsAt + "90: " + admChannelsPkg + "counted steps tallyDelivery, so its parameter n takes 0..2, not Boolean"},
		{"watched state", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "store").Monitors = []string{admDeclaredID + "storedOnce"}
		}, admDeclaredAt + "34: store names monitor storedOnce, whose next takes fixture.declarations.DiskState, not the state fixture.declarations.StoreState"},
		{"monitor next state", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").Next = admDeclaredID + "endsDurable.next"
		}, admDeclaredAt + "79: monitor storedOnce's next takes Boolean, not its state fixture.declarations.Seen"},
		{"monitor violated state", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").Violated = admDeclaredID + "endsDurable.violated"
		}, admDeclaredAt + "79: monitor storedOnce's violated takes Boolean, not its state fixture.declarations.Seen"},
		{"delivered message", "channels", func(m *umpirespb.Model) {
			admAction(m, admChannelsID+"wire.deliver").GetInputs()[0].Type = &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: "fixture.channels.Signal"}}
		}, admChannelsAt + "14: " + admChannelsID + "wire.deliver delivers " + admChannelsID + "wire, so it takes one input of fixture.channels.Note"},
		{"lost messages", "channels", func(m *umpirespb.Model) {
			a := admAction(m, admChannelsID+"radio.lose")
			a.Inputs = append(a.Inputs, a.GetInputs()[0])
		}, admChannelsAt + "16: " + admChannelsID + "radio.lose loses " + admChannelsID + "radio, so it takes one input of fixture.channels.Signal"},
		{"member field", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "pair").GetMembers()[1].Field = "middle"
		}, admDeclaredAt + "124: member middle of pair is no field of fixture.declarations.PairState"},
		{"member state", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "pair").GetMembers()[1].Machine = "disk"
		}, admDeclaredAt + "124: member back of pair holds fixture.declarations.StoreState, not the state fixture.declarations.DiskState of disk"},
	})
}

func TestAdmissionRejectsDuplicates(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"type", "declarations", func(m *umpirespb.Model) {
			m.Types = append(m.Types, proto.Clone(admType(m, "fixture.declarations.StoreState")).(*umpirespb.Type))
		}, admDeclaredAt + "13: two types named fixture.declarations.StoreState"},
		{"function", "declarations", func(m *umpirespb.Model) {
			m.Functions = append(m.Functions, proto.Clone(function(m, "store.put")).(*umpirespb.Function))
		}, admDeclaredAt + "44: two functions named store.put"},
		{"action", "declarations", func(m *umpirespb.Model) {
			m.Actions = append(m.Actions, proto.Clone(admAction(m, admDeclaredID+"put")).(*umpirespb.Action))
		}, admDeclaredAt + "21: two actions with id " + admDeclaredID + "put"},
		{"machine", "declarations", func(m *umpirespb.Model) {
			m.Machines = append(m.Machines, proto.Clone(admMachine(m, "store")).(*umpirespb.Machine))
		}, admDeclaredAt + "34: two machines named store"},
		{"machine and composition", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "store").Name = "pair"
		}, admDeclaredAt + "124: a machine and a composition are both named pair"},
		{"channel", "channels", func(m *umpirespb.Model) {
			m.Channels = append(m.Channels, proto.Clone(admChannel(m, "wire")).(*umpirespb.Channel))
		}, admChannelsAt + "14: two channels with id " + admChannelsID + "wire"},
		{"monitor id", "declarations", func(m *umpirespb.Model) {
			m.Monitors = append(m.Monitors, proto.Clone(admMonitor(m, "storedOnce")).(*umpirespb.Monitor))
		}, admDeclaredAt + "79: two monitors with id " + admDeclaredID + "storedOnce"},
		{"monitor name", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "endsDurable").Name = "storedOnce"
		}, admDeclaredAt + "79: two monitors named storedOnce"},
		{"assumption", "declarations", func(m *umpirespb.Model) {
			m.Assumptions = append(m.Assumptions, proto.Clone(m.GetAssumptions()[1]).(*umpirespb.Assumption))
		}, admDeclaredAt + "25: two assumptions with id " + admDeclaredID + "storeOpaque"},
		{"hole", "declarations", func(m *umpirespb.Model) {
			m.Holes = append(m.Holes, proto.Clone(m.GetHoles()[0]).(*umpirespb.Hole))
		}, admDeclaredAt + "27: two holes with id " + admDeclaredID + "crashUnmodeled"},
		{"composition", "declarations", func(m *umpirespb.Model) {
			m.Compositions = append(m.Compositions, proto.Clone(admComposition(m, "pair")).(*umpirespb.Composition))
		}, admDeclaredAt + "124: two compositions named pair"},
		{"property", "declarations", func(m *umpirespb.Model) {
			m.Properties = append(m.Properties, proto.Clone(admProperty(m, "store", "putStores")).(*umpirespb.Property))
		}, admDeclaredAt + "141: two Properties named putStores on store"},
		{"scenario", "declarations", func(m *umpirespb.Model) {
			m.Scenarios = append(m.Scenarios, proto.Clone(admScenario(m, "store", "putOnce")).(*umpirespb.Scenario))
		}, admDeclaredAt + "152: two Scenarios named putOnce on store"},
		{"query", "declarations", func(m *umpirespb.Model) {
			m.Queries = append(m.Queries, proto.Clone(admQuery(m, "putStores")).(*umpirespb.Query))
		}, admDeclaredAt + "165: two Queries named putStores"},
		{"progress", "declarations", func(m *umpirespb.Model) {
			m.Progress = append(m.Progress, proto.Clone(m.GetProgress()[0]).(*umpirespb.Progress))
		}, admDeclaredAt + "113: two progress claims named durableEventually on disk"},
		{"record field", "channels", func(m *umpirespb.Model) {
			r := admType(m, "fixture.channels.RelayState").GetRecord()
			r.Fields = append(r.Fields, proto.Clone(r.GetFields()[0]).(*umpirespb.Field))
		}, admChannelsAt + "23: fixture.channels.RelayState has two fields named heard"},
		{"enum case", "channels", func(m *umpirespb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, proto.Clone(e.GetCases()[0]).(*umpirespb.Case))
		}, admChannelsAt + "8: fixture.channels.Note has two cases named ping"},
		{"case field", "presence", func(m *umpirespb.Model) {
			sent := admType(m, "fixture.presence.Report").GetEnum().GetCases()[1]
			sent.Fields = append(sent.Fields, proto.Clone(sent.GetFields()[0]).(*umpirespb.Field))
		}, admPresenceAt + "12: case sent of fixture.presence.Report has two fields named result"},
		{"parameter", "declarations", func(m *umpirespb.Model) {
			f := function(m, "countStored")
			f.Params[1].Name = "seen"
		}, admDeclaredAt + "72: " + admDeclaredPkg + "countStored has two parameters named seen"},
	})
}

func admInt() *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Int{Int: &umpirespb.Empty{}}}
}

func admList(of string) *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_List{List: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: of}}}}
}

func TestAdmissionRejectsInfiniteCatalogsAndBounds(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"monitor state of Int", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").State = admInt()
		}, admDeclaredAt + "79: monitor storedOnce needs a state of a finite type, not Int"},
		{"monitor state of a list", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").State = admList("fixture.declarations.Seen")
		}, admDeclaredAt + "79: monitor storedOnce needs a state of a finite type, not List[fixture.declarations.Seen]"},
		{"monitor state of a channel", "declarations", func(m *umpirespb.Model) {
			admMonitor(m, "storedOnce").State = &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: "wire"}}
		}, admDeclaredAt + "79: monitor storedOnce needs a state of a finite type, not channel wire"},
		{"channel message of Int", "channels", func(m *umpirespb.Model) {
			admChannel(m, "tally").Message = admInt()
		}, admChannelsAt + "67: channel " + admChannelsID + "tally needs a message of a finite type, not Int"},
		{"channel message of a list", "channels", func(m *umpirespb.Model) {
			admChannel(m, "wire").Message = admList("fixture.channels.Note")
		}, admChannelsAt + "14: channel " + admChannelsID + "wire needs a message of a finite type, not List[fixture.channels.Note]"},
		{"channel capacity", "channels", func(m *umpirespb.Model) {
			admChannel(m, "wire").Capacity = 0
		}, admChannelsAt + "14: channel " + admChannelsID + "wire has capacity 0, below 1"},
		{"channel duplicates", "channels", func(m *umpirespb.Model) {
			admChannel(m, "wire").Duplicates = -1
		}, admChannelsAt + "14: channel " + admChannelsID + "wire has -1 duplicates, below 0"},
		{"limits steps", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").GetLimits().Steps = -1
		}, admDeclaredAt + "163: query durableStays limits steps to -1, below 0"},
		{"limits actions", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").GetLimits().Actions = -2
		}, admDeclaredAt + "163: query durableStays limits actions to -2, below 0"},
		{"limits search", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "durableStays").GetLimits().Search = -3
		}, admDeclaredAt + "163: query durableStays limits search to -3, below 0"},
		{"progress within", "declarations", func(m *umpirespb.Model) {
			m.GetProgress()[0].Within = 0
		}, admDeclaredAt + "113: progress claim durableEventually of disk is within 0 steps, fewer than one"},
	})
}

func TestAdmissionRejectsChannelMisuse(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"one channel in two fields", "channels", func(m *umpirespb.Model) {
			admType(m, "fixture.channels.RelayState").GetRecord().GetFields()[2].Type = &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: admChannelsID + "wire"}}
		}, admChannelsAt + "23: fixture.channels.RelayState holds channel " + admChannelsID + "wire in two fields"},
		{"delivery of a channel the state does not hold", "channels", func(m *umpirespb.Model) {
			admMachine(m, "tallying").GetSteps()[1].Action = admChannelsID + "wire.deliver"
		}, admChannelsAt + "90: tallying binds a delivery of channel " + admChannelsID + "wire, which its state fixture.channels.Tally does not hold"},
		{"loss of a channel the state does not hold", "channels", func(m *umpirespb.Model) {
			admMachine(m, "tallying").GetSteps()[1].Action = admChannelsID + "radio.lose"
		}, admChannelsAt + "90: tallying binds a loss of channel " + admChannelsID + "radio, which its state fixture.channels.Tally does not hold"},
		{"lossy channel with no loss", "channels", func(m *umpirespb.Model) {
			relay := admMachine(m, "relay")
			relay.Steps = relay.GetSteps()[:4]
		}, admChannelsAt + "53: relay holds lossy channel " + admChannelsID + "radio but binds no loss of it"},
		{"loss of a reliable channel", "channels", func(m *umpirespb.Model) {
			admChannel(m, "radio").Lossy = false
		}, admChannelsAt + "16: " + admChannelsID + "radio.lose loses channel " + admChannelsID + "radio, which is not lossy"},
	})
}

func TestAdmissionRejectsReadingsWithoutARefinement(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"visible facts of no refinement", "declarations", func(m *umpirespb.Model) {
			admMachine(m, "disk").GetRefines().Product = ""
		}, admDeclaredAt + "93: disk names what a refined machine sees but refines none"},
		{"replacement its member does not refine", "declarations", func(m *umpirespb.Model) {
			admComposition(m, "detailedPair").GetMembers()[1].Replaces = "disk"
		}, admDeclaredAt + "129: member back of detailedPair replaces disk, which disk does not refine"},
		{"property of another machine", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "putStores").Scenario = &umpirespb.ClaimRef{Machine: "disk", Name: "putThenFlush"}
		}, admDeclaredAt + "165: query putStores pairs a Property of store with a Scenario of disk"},
		{"through what the scenario machine does not refine", "declarations", func(m *umpirespb.Model) {
			admQuery(m, "putStoresThroughDisk").Scenario = &umpirespb.ClaimRef{Machine: "pair", Name: "any"}
		}, admDeclaredAt + "166: query putStoresThroughDisk pairs a Property of store with a Scenario of pair"},
		{"product property without through", "admission", func(m *umpirespb.Model) {
			admQuery(m, "activityRecord.product.pausedIsNotDispatched").Through = false
		}, admAdmissionAt + "373: query activityRecord.product.pausedIsNotDispatched pairs a Property of activityProduct with a Scenario of activityRecord"},
	})
}

// admCall is a call of name with x as its one argument, at the position of at.
func admCall(at *umpirespb.Expr, name string, x *umpirespb.Expr) *umpirespb.Expr {
	return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Call{Call: &umpirespb.Call{Function: name, Args: []*umpirespb.Expr{x}}}}
}

func admVar(at *umpirespb.Expr, name string) *umpirespb.Expr {
	return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Var{Var: name}}
}

func TestAdmissionRejectsRecursion(t *testing.T) {
	oneMore := admAdmissionPkg + "oneMore"
	runAdmissionCases(t, []admissionCase{
		{"direct", "admission", func(m *umpirespb.Model) {
			f := function(m, "oneMore")
			f.Body = admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))
		}, admAdmissionAt + "167: " + oneMore + " calls itself"},
		{"through others", "admission", func(m *umpirespb.Model) {
			f := function(m, "Admission$package$.admitted")
			f.Body = admCall(f.GetBody(), admAdmissionPkg+"admitCurrent", admVar(f.GetBody(), "s"))
		}, admAdmissionAt + "209: " + admAdmissionPkg + "admitCurrent calls itself through " + admAdmissionPkg + "admitted"},
		{"in a lambda", "admission", func(m *umpirespb.Model) {
			f := function(m, "oneMore")
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{
				Params: f.GetParams(), Body: admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))}}}
		}, admAdmissionAt + "167: " + oneMore + " calls itself"},
		{"in a precondition", "admission", func(m *umpirespb.Model) {
			f := function(m, "oneMore")
			f.Requires = admCall(f.GetBody(), oneMore, admVar(f.GetBody(), "a"))
		}, admAdmissionAt + "167: " + oneMore + " calls itself"},
	})
	m := admFixture(t, "admission")
	f := function(m, "Admission$package$.admitted")
	f.Body = admCall(f.GetBody(), admAdmissionPkg+"admitCurrent", admVar(f.GetBody(), "s"))
	require.ErrorContains(t, Validate(m), admAdmissionAt+"179: "+admAdmissionPkg+"admitted calls itself through "+admAdmissionPkg+"admitCurrent",
		"every function on the cycle is reported")
}

func TestAdmissionReportsEveryProblem(t *testing.T) {
	m := admFixture(t, "channels")
	m.Version = 2
	admChannel(m, "wire").Capacity = 0
	admChannel(m, "radio").Lossy = false
	admMachine(m, "tallying").GetSteps()[0].Function = "nowhere"
	err := Validate(m)
	require.ErrorContains(t, err, "model: fixture.channels.Relay, fixture.channels.Tallying: "+
		"version 2 is not a version this reader knows")
	require.ErrorContains(t, err, admChannelsAt+"14: channel "+admChannelsID+"wire has capacity 0, below 1")
	require.ErrorContains(t, err, admChannelsAt+"16: "+admChannelsID+"radio.lose loses channel "+admChannelsID+"radio, which is not lossy")
	require.ErrorContains(t, err, admChannelsAt+"90: no function nowhere")
	require.Len(t, strings.Split(err.Error(), "\n"), 4, "each problem once, and nothing else")
}

func admLiteral(at *umpirespb.Expr, v *umpirespb.Value) *umpirespb.Expr {
	return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Literal{Literal: v}}
}

func admEnum(typ, c string) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: c}}}
}

func admIntValue(n int64) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: n}}
}

func TestAdmissionRejectsAStepThatReturnsNoSteps(t *testing.T) {
	putStep := admDeclaredPkg + "putStep"
	runAdmissionCases(t, []admissionCase{
		{"a Boolean", "declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = admLiteral(f.GetBody(), &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: false}})
		}, admDeclaredAt + "56: " + putStep + " returns a Boolean, not a list of steps"},
		{"a record", "declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_Construct{Construct: &umpirespb.Construct{
				Type: "fixture.declarations.DiskState", Args: []*umpirespb.Expr{admLiteral(f.GetBody(), admEnum("fixture.declarations.Stage", "empty"))}}}}
		}, admDeclaredAt + "56: " + putStep + " returns a fixture.declarations.DiskState, not a list of steps"},
		{"a list of records", "declarations", func(m *umpirespb.Model) {
			f := function(m, "Declarations$package$.putStep")
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{Items: []*umpirespb.Expr{
				admLiteral(f.GetBody(), admEnum("fixture.declarations.Stage", "empty"))}}}}
		}, admDeclaredAt + "56: " + putStep + " returns a list holding a fixture.declarations.Stage, not a list of steps"},
	})
}

func TestAdmissionRejectsAnActionThatBothDeliversAndLoses(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"delivers and loses", "channels", func(m *umpirespb.Model) {
			admAction(m, admChannelsID+"radio.lose").Delivers = admChannelsID + "radio"
		}, admChannelsAt + "16: " + admChannelsID + "radio.lose both delivers and loses " + admChannelsID + "radio; an action does one"},
	})
}

func TestAdmissionRejectsACatalogThatContainsItself(t *testing.T) {
	runAdmissionCases(t, []admissionCase{
		{"a record of itself", "channels", func(m *umpirespb.Model) {
			admType(m, "fixture.channels.RelayState").GetRecord().GetFields()[0].Type = interp.Named("fixture.channels.RelayState")
		}, admChannelsAt + "23: type fixture.channels.RelayState has no finite catalog: it contains itself"},
		{"a message holding its channel", "channels", func(m *umpirespb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, &umpirespb.Case{Name: "echo", Fields: []*umpirespb.Field{{Name: "back",
				Type: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: admChannelsID + "wire"}}}}})
		}, admChannelsAt + "8: type fixture.channels.Note has no finite catalog: it contains itself through channel " + admChannelsID + "wire"},
		{"a channel holding its message", "channels", func(m *umpirespb.Model) {
			e := admType(m, "fixture.channels.Note").GetEnum()
			e.Cases = append(e.Cases, &umpirespb.Case{Name: "echo", Fields: []*umpirespb.Field{{Name: "back",
				Type: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Channel{Channel: admChannelsID + "wire"}}}}})
		}, admChannelsAt + "14: channel " + admChannelsID + "wire has no finite catalog: it contains itself through type fixture.channels.Note"},
	})
}

// A Scenario on tallying, declared beside the lifted ones, that delivers message n.
func admTallyScenario(m *umpirespb.Model, n int64) {
	at := &umpirespb.Position{File: "generic", Line: 1}
	m.Scenarios = append(m.Scenarios, &umpirespb.Scenario{Machine: "tallying", Name: "counts", Position: at,
		Start:   proto.Clone(admMachine(m, "tallying").GetStarts()[0]).(*umpirespb.Expr),
		Actions: []*umpirespb.ActionClass{{Action: admChannelsID + "tally.deliver", Inputs: []*umpirespb.Value{admIntValue(n)}}}})
}

func TestAdmissionRejectsMisaddressedSelectors(t *testing.T) {
	put, flush, control := admDeclaredID+"put", admDeclaredID+"flush", admAdmissionID+"control"
	runAdmissionCases(t, []admissionCase{
		{"when_class input the action does not take", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "store", "putStores").GetWhenClass().Inputs = []*umpirespb.Value{admIntValue(1)}
		}, admDeclaredAt + "141: store.putStores: " + put + " takes 0 inputs, not 1"},
		{"when_class of an action the machine does not bind", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "store", "putStores").GetWhenClass().Action = flush
		}, admDeclaredAt + "141: store.putStores: store binds no action " + flush},
		{"when_action the machine does not bind", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "disk", "putAccepted").When = &umpirespb.Property_WhenAction{WhenAction: "nothing"}
		}, admDeclaredAt + "143: disk.putAccepted: disk binds no action nothing"},
		// putBoth takes each member's put, so the composition has no class of an action put, and none
		// of front's put either.
		{"when_action of no class of a composition", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenAction{WhenAction: "put"}
		}, admDeclaredAt + "147: detailedPair.frontHeld: detailedPair has no class of the action put"},
		{"when_action of a synced member action of a composition", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenAction{WhenAction: "front_put"}
		}, admDeclaredAt + "147: detailedPair.frontHeld: detailedPair has no class of the action front_put"},
		{"when_action of an action no member binds on a composition", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenAction{WhenAction: "front_flush"}
		}, admDeclaredAt + "147: detailedPair.frontHeld: detailedPair has no class of the action front_flush"},
		{"when_class of no class of a composition", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: put}}
		}, admDeclaredAt + "147: detailedPair.frontHeld: detailedPair has no class put"},
		{"when_class of an undeclared action on a composition", "declarations", func(m *umpirespb.Model) {
			admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: "nowhere"}}
		}, admDeclaredAt + "147: no action nowhere"},
		{"scenario input the action does not take", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "store", "putOnce").GetActions()[0].Inputs = []*umpirespb.Value{admIntValue(1)}
		}, admDeclaredAt + "152: store.putOnce: " + put + " takes 0 inputs, not 1"},
		{"scenario input of a crossed type", "admission", func(m *umpirespb.Model) {
			admScenario(m, "activityRecord", "staleDeliveryAfterPause").GetActions()[1].Inputs[0] = admEnum(
				"fixture.specimens.admission.AdmissionPhase", "paused")
		}, admAdmissionAt + "352: activityRecord.staleDeliveryAfterPause: " + control + " takes a fixture.specimens.admission.Control for control, not paused"},
		{"scenario input outside its range", "channels", func(m *umpirespb.Model) {
			admTallyScenario(m, 3)
		}, "generic:1: tallying.counts: " + admChannelsID + "tally.deliver takes a 0..2 for message, not 3"},
		{"scenario action the machine does not bind", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "store", "putOnce").GetActions()[0].Action = flush
		}, admDeclaredAt + "152: store.putOnce: store binds no action " + flush},
		{"machine scenario start of another type", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "store", "putOnce").Start = proto.Clone(admScenario(m, "disk", "putThenFlush").GetStart()).(*umpirespb.Expr)
		}, admDeclaredAt + "152: store.putOnce starts at empty, which is no fixture.declarations.StoreState"},
		{"composition scenario start of another type", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "detailedPair", "bothPut").Start = proto.Clone(admScenario(m, "pair", "any").GetStart()).(*umpirespb.Expr)
		}, admDeclaredAt + "156: detailedPair.bothPut starts at nothing-nothing, which is no fixture.declarations.DetailedPairState"},
		{"composition scenario key of no class", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "detailedPair", "bothPut").Keys = []string{"putNone"}
		}, admDeclaredAt + "156: detailedPair.bothPut: detailedPair has no class putNone"},
		{"composition scenario key of an action its member does not bind", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "detailedPair", "bothPut").Keys = []string{"front_flush"}
		}, admDeclaredAt + "156: detailedPair.bothPut: detailedPair has no class front_flush"},
		// putBoth takes front's put, so front has no put class of its own.
		{"composition scenario of a synced member action", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "detailedPair", "bothPut").Keys = []string{"front_put"}
		}, admDeclaredAt + "156: detailedPair.bothPut: detailedPair has no class front_put"},
		{"composition scenario of actions", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "detailedPair", "bothPut").Actions = []*umpirespb.ActionClass{{Action: put}}
		}, admDeclaredAt + "156: detailedPair.bothPut: a Scenario of a composition schedules its class keys, not actions"},
		{"machine scenario of keys", "declarations", func(m *umpirespb.Model) {
			admScenario(m, "store", "putOnce").Keys = []string{"put"}
		}, admDeclaredAt + "152: store.putOnce: a Scenario of a machine schedules action classes, not keys"},
	})
}

func TestAdmissionAdmitsWellAddressedSelectors(t *testing.T) {
	m := admFixture(t, "declarations")
	admScenario(m, "detailedPair", "bothPut").Keys = []string{"putBoth", "back_flush", "back_crash"}
	require.NoError(t, Validate(recounted(t, m)))
	m = admFixture(t, "channels")
	admTallyScenario(m, 2)
	require.NoError(t, Validate(m))
}

// A Property of a composition may be about some of its steps: the classes of a sync, by the sync's
// name, the classes of a member's own action, by its composed name, and one class, by its key.
func TestAdmissionAdmitsAWhenOverComposedClasses(t *testing.T) {
	for _, action := range []string{"putBoth", "back_flush", "back_crash"} {
		m := admFixture(t, "declarations")
		admProperty(m, "detailedPair", "frontHeld").When = &umpirespb.Property_WhenAction{WhenAction: action}
		require.NoError(t, Validate(m), action)
	}
	// standaloneActivity's sync poll is named as the activity's action it takes, so that
	// action's one class is keyed as the sync's step is.
	m := proto.Clone(activityModel(t)).(*umpirespb.Model)
	var poll string
	for _, a := range m.GetActions() {
		if a.GetName() == "poll" {
			poll = a.GetId()
		}
	}
	require.NotEmpty(t, poll)
	started := admProperty(m, "standaloneActivity", "startedByPollingWorker")
	require.NotNil(t, started)
	started.When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: poll}}
	require.NoError(t, Validate(m))
	// The activity's own control has a class for each of its inputs, and the action names them all.
	started.When = &umpirespb.Property_WhenAction{WhenAction: "activity_control"}
	require.NoError(t, Validate(m))
	started.When = &umpirespb.Property_WhenAction{WhenAction: "activity_control-pause"}
	require.ErrorContains(t, Validate(m), "standaloneActivity.startedByPollingWorker: standaloneActivity has no class of the action activity_control-pause")
	started.When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: poll, Inputs: []*umpirespb.Value{admIntValue(1)}}}
	require.ErrorContains(t, Validate(m), "standaloneActivity.startedByPollingWorker: standaloneActivity has no class poll-1")
}
