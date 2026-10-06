package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	admLifts       = "model/irgen/testdata/lifts/"
	admDeclaredAt  = admLifts + "Declarations.scala:"
	admDeclaredPkg = "fixture.declarations.Declarations$package$."
	admDeclaredID  = "fixture.declarations."
	// The disk's refinement map, its refinement section's toProduct.
	declaredRefinementMap = "fixture.declarations.Disk$.refinement$.toProduct"
)

func admMachine(m *umpirespb.Model, name string) *umpirespb.Machine {
	for _, mm := range m.GetMachines() {
		if mm.GetName() == name {
			return mm
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

func admLiteral(at *umpirespb.Expr, v *umpirespb.Value) *umpirespb.Expr {
	return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Literal{Literal: v}}
}

func admEnum(typ, c string) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: c}}}
}

func admIntValue(n int64) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: n}}
}

func upTo(high int64) *umpirespb.TypeRef {
	return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_IntRange{IntRange: &umpirespb.IntRange{High: high}}}
}

const admissionPackage = "fixture.specimens.admission.Admission$package$."

// stepAt is the step record construct of an item of a list expression.
func stepAt(t *testing.T, list *umpirespb.Expr, item int) *umpirespb.Construct {
	t.Helper()
	c := list.GetList().GetItems()[item].GetConstruct()
	require.Equal(t, interp.StepType, c.GetType())
	return c
}

// namedAdmission is the admission specimen with its committed admission's alternatives named: a
// redelivered message is consumed, a first delivery consumed or retained. A dispatch, one result of
// its own, is named too; the stale-message rejection of admitCurrent is left unnamed. plain is the
// specimen with the names its source gives with `choose` cleared.
func namedAdmission(t *testing.T) (plain, named *umpirespb.Model) {
	t.Helper()
	plain = lifted(t, "admission")
	chosen := functionNamed(plain, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, chosen.GetElse(), 0).Choice = ""
	stepAt(t, chosen.GetElse(), 1).Choice = ""
	named = proto.CloneOf(plain)
	admitted := functionNamed(named, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, admitted.GetThen(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 1).Choice = "retained"
	stepAt(t, functionNamed(named, admissionPackage+"dispatchStep").GetBody().GetIf().GetElse(), 0).Choice = "enqueued"
	return plain, named
}

func choicesOf(row interp.Row) []string {
	out := []string{}
	for _, res := range row.Results {
		out = append(out, res.Choice)
	}
	return out
}

func load(t *testing.T) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(irPath)
	require.NoError(t, err)
	return m
}

func function(m *umpirespb.Model, suffix string) *umpirespb.Function {
	for _, f := range m.GetFunctions() {
		if strings.HasSuffix(f.GetName(), suffix) {
			return f
		}
	}
	return nil
}

func lifted(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", name+".json"))
	require.NoError(t, err)
	return m
}

func at(line int32) *umpirespb.Position { return &umpirespb.Position{File: "generic", Line: line} }

func enumType(name string, line int32, cases ...string) *umpirespb.Type {
	e := &umpirespb.Enum{}
	for _, c := range cases {
		e.Cases = append(e.Cases, &umpirespb.Case{Name: c})
	}
	return &umpirespb.Type{Name: name, Position: at(line), Shape: &umpirespb.Type_Enum{Enum: e}}
}

func caseOf(typ, c string) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: c}}}
}

func built(t *testing.T, m *umpirespb.Model) map[string]*interp.Machine {
	t.Helper()
	out, err := interp.Build(m)
	require.NoError(t, err)
	return out
}

func row(t *testing.T, mm *interp.Machine, key string) interp.Row {
	t.Helper()
	for _, r := range mm.Table.Rows {
		if r.Key == key {
			return r
		}
	}
	require.Failf(t, "no row", "%s has no row %s", mm.Decl.GetName(), key)
	return interp.Row{}
}

func hasRow(mm *interp.Machine, key string) bool {
	for _, r := range mm.Table.Rows {
		if r.Key == key {
			return true
		}
	}
	return false
}

func slicesDelete[T any](xs []T, drop func(T) bool) []T {
	var out []T
	for _, x := range xs {
		if !drop(x) {
			out = append(out, x)
		}
	}
	return out
}

const (
	repoRoot = "../../.."
)

// positions returns every source position in m.
func positions(m proto.Message) []*umpirespb.Position {
	var out []*umpirespb.Position
	var walk func(protoreflect.Message)
	walk = func(r protoreflect.Message) {
		if p, ok := r.Interface().(*umpirespb.Position); ok {
			out = append(out, p)
			return
		}
		r.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			switch {
			case fd.IsList() && fd.Message() != nil:
				for i := range v.List().Len() {
					walk(v.List().Get(i).Message())
				}
			case fd.IsMap() && fd.MapValue().Message() != nil:
				v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
					walk(mv.Message())
					return true
				})
			case !fd.IsList() && !fd.IsMap() && fd.Message() != nil:
				walk(v.Message())
			default:
				// A scalar holds no position.
			}
			return true
		})
	}
	walk(m.ProtoReflect())
	return out
}

const irPath = "../../../model/ir/nexus-workflow.json"

func machines(t *testing.T) map[string]*interp.Machine {
	t.Helper()
	built, err := interp.Build(readIR(t, irPath))
	require.NoError(t, err)
	return built
}

// readIR decodes ProtoJSON without admission, which depends on the interpreter.
func readIR(t *testing.T, path string) *umpirespb.Model {
	t.Helper()
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	m := &umpirespb.Model{}
	require.NoError(t, protojson.UnmarshalOptions{}.Unmarshal(encoded, m))
	return m
}

const admRealizationAt = "model/temporal/features/nexus/workflow/Realization.scala:"
