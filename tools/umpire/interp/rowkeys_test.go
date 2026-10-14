package interp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func rowKeyCatalogs(states, classes []string) ([]Value, []Class) {
	values := make([]Value, len(states))
	for i, state := range states {
		values[i] = Value{Kind: TextValue, Text: state}
	}
	actions := make([]Class, len(classes))
	for i, class := range classes {
		actions[i] = Class{Key: class}
	}
	return values, actions
}

func rowKeyMachine() *umpirespb.Machine {
	return &umpirespb.Machine{Name: "m", Position: &umpirespb.Position{File: "rowkeys", Line: 7}}
}

func originalRowKeyError(states, classes []string) error {
	seen := make(map[string][2]string, len(states)*len(classes))
	for _, state := range states {
		for _, class := range classes {
			key := state + "-" + class
			if earlier, ok := seen[key]; ok {
				return fmt.Errorf("rowkeys:7: m: the state %s with the class %s, and the state %s with the class %s, share the row key %q",
					earlier[0], earlier[1], state, class, key)
			}
			seen[key] = [2]string{state, class}
		}
	}
	return nil
}

func TestRowKeysPreservesEverySmallCatalogDiagnostic(t *testing.T) {
	keys := []string{"", "a", "b", "-", "a-", "-a", "a-b", "é", "é-a"}
	catalogs := [][]string{nil}
	for _, first := range keys {
		catalogs = append(catalogs, []string{first})
		for _, second := range keys {
			catalogs = append(catalogs, []string{first, second})
		}
	}
	decl := rowKeyMachine()
	for _, states := range catalogs {
		for _, classes := range catalogs {
			values, actions := rowKeyCatalogs(states, classes)
			require.Equal(t, fmt.Sprint(originalRowKeyError(states, classes)), fmt.Sprint(RowKeys(decl, values, actions)),
				"states %q, classes %q", states, classes)
		}
	}
	t.Logf("compared all %d ordered catalog pairs", len(catalogs)*len(catalogs))
}

func TestRowKeysReportsTheFirstCollisionInStatesMajorOrder(t *testing.T) {
	for _, test := range []struct {
		name            string
		states, classes []string
		message         string
	}{
		{"ambiguous boundary", []string{"a", "a-b"}, []string{"b-c", "c"},
			`m: the state a with the class b-c, and the state a-b with the class c, share the row key "a-b-c"`},
		{"duplicate state", []string{"a", "a"}, []string{"go", "stop"},
			`m: the state a with the class go, and the state a with the class go, share the row key "a-go"`},
		{"duplicate class before an ambiguous boundary", []string{"a", "a-b"}, []string{"b-c", "c", "c"},
			`m: the state a with the class c, and the state a with the class c, share the row key "a-c"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			values, actions := rowKeyCatalogs(test.states, test.classes)
			var collision *Error
			require.ErrorAs(t, RowKeys(rowKeyMachine(), values, actions), &collision)
			require.Equal(t, &Error{Position: "rowkeys:7", Message: test.message}, collision)
		})
	}
}

func TestRowKeysAvoidsAllocatingTheCollisionFreeProduct(t *testing.T) {
	states, classes := make([]string, 128), make([]string, 96)
	for i := range states {
		states[i] = fmt.Sprintf("state-%d-ready", i)
	}
	for i := range classes {
		classes[i] = fmt.Sprintf("step-%d-", i)
	}
	values, actions := rowKeyCatalogs(states, classes)
	decl := rowKeyMachine()
	var err error
	allocations := testing.AllocsPerRun(1, func() { err = RowKeys(decl, values, actions) })
	require.NoError(t, err)
	t.Logf("allocations=%g for %d states × %d classes", allocations, len(states), len(classes))
	require.Less(t, allocations, float64(4*(len(states)+len(classes))), "collision-free row keys need no Cartesian allocation")
}
