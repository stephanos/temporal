package ir

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

const activitySystemIR = "../../../model/ir/activity-standalone-record.json"

// realizationNamed is the realization a Model declares under a name: a Model declares several.
func realizationNamed(t testing.TB, m *umpirespb.Model, name string) *umpirespb.Realization {
	t.Helper()
	for _, r := range m.GetRealizations() {
		if r.GetName() == name {
			return r
		}
	}
	require.FailNow(t, "no realization "+name)
	return nil
}

// recounted is m with each Query that asserts a total asserting m's count (WithTotals).
func recounted(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	out, err := WithTotals(m)
	require.NoError(t, err)
	return out
}

// functionNamed is the function of exactly this name, where function takes the first of a suffix.
func functionNamed(m *umpirespb.Model, name string) *umpirespb.Function {
	for _, f := range m.GetFunctions() {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func boolValue(b bool) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: b}}
}

func built(t *testing.T, m *umpirespb.Model) map[string]*interp.Machine {
	t.Helper()
	out, err := interp.Build(m)
	require.NoError(t, err)
	return out
}

const nexusCloseIR = "../../../model/ir/nexus-workflow-close.json"

const irPath = "../../../model/ir/nexus-workflow.json"

// BuildWithin is Build within explicit ceilings.
func BuildWithin(m *umpirespb.Model, c interp.Ceilings) (map[string]*interp.Machine, error) {
	in := interp.NewInterpreterWithin(m, c)
	return in.Build(m)
}

const activityIR = "../../../model/ir/activity-standalone.json"

var activityBaseline = sync.OnceValues(func() (*umpirespb.Model, error) { return Load(activityIR) })

func activityModel(t *testing.T) *umpirespb.Model {
	t.Helper()
	m, err := activityBaseline()
	require.NoError(t, err)
	return m
}
