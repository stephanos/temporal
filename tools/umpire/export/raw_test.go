package export

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// The adapters exercise the private raw reader; the literal old traversal remains independent.
func characterizedRawMachine(r reader, input []byte) (*machineView, bool, error) {
	return r.rawMachine(input)
}

func characterizedRawRows[T any](input []byte, source, class func(any) (string, error), step func(any) (T, error)) (map[string]map[string][]T, error) {
	return rawBySource(input, source, class, step)
}

// Literal original traversal, independent of the production bySource helper.
func originalRawRows[T any](raw any, source, class func(any) (string, error), step func(any) (T, error)) (map[string]map[string][]T, error) {
	rows, err := list(raw)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string][]T{}
	for _, raw := range rows {
		row, err := record(raw, "a row")
		if err != nil {
			return nil, err
		}
		from, err := source(row["src"])
		if err != nil {
			return nil, err
		}
		pairs, err := list(row["by"])
		if err != nil {
			return nil, err
		}
		out[from] = map[string][]T{}
		for _, raw := range pairs {
			pair, err := record(raw, "a pair of a class and its steps")
			if err != nil {
				return nil, err
			}
			key, err := class(pair["cls"])
			if err != nil {
				return nil, err
			}
			if out[from][key], err = each(pair["steps"], step); err != nil {
				return nil, fmt.Errorf("%s by %s: %w", from, key, err)
			}
		}
	}
	return out, nil
}

func TestRawRowsPreserveDuplicateAndFirstErrorSemantics(t *testing.T) {
	const row = `{"src":"s","by":[{"cls":"a","steps":["first","second"]},{"cls":"b","steps":[]}]}`
	for _, specimen := range []struct {
		name, input, refusal string
		want                 map[string]map[string][]string
	}{
		{"list", `[` + row + `]`, "", map[string]map[string][]string{"s": {"a": {"first", "second"}, "b": {}}}},
		{"set", `{"#set":[` + row + `]}`, "", map[string]map[string][]string{"s": {"a": {"first", "second"}, "b": {}}}},
		{"last set field", `{"#set":false,"#set":[` + row + `]}`, "", map[string]map[string][]string{"s": {"a": {"first", "second"}, "b": {}}}},
		{"nested set is not another list", `{"#set":{"#set":[]}}`, "map[#set:[]] is no list and no set", nil},
		{"map-valued set", `{"#set":{"other":1}}`, "map[other:1] is no list and no set", nil},
		{"map-valued set ignores wrapper extras", `{"ignored":{"valid":[true,1]},"#set":{"other":1}}`, "map[other:1] is no list and no set", nil},
		{"empty", `[]`, "", map[string]map[string][]string{}},
		{"source reset", `[` + row + `,{"src":"s","by":[]}]`, "", map[string]map[string][]string{"s": {}}},
		{"class last", `[{"src":"s","by":[{"cls":"a","steps":["old"]},{"cls":"a","steps":["new"]}]}]`, "", map[string]map[string][]string{"s": {"a": {"new"}}}},
		{"duplicate source field", `[{"src":false,"src":"s","by":[]}]`, "", map[string]map[string][]string{"s": {}}},
		{"null list", `null`, "<nil> is no list and no set", nil},
		{"missing set", `{}`, "<nil> is no list and no set", nil},
		{"wrong list", `false`, "false is no list and no set", nil},
		{"null row", `[null,` + row + `]`, "a row is no record", nil},
		{"source before by", `[{"src":null,"by":false},` + row + `]`, "source is no string", nil},
		{"missing by", `[{"src":"s"},` + row + `]`, "<nil> is no list and no set", nil},
		{"null pair", `[{"src":"s","by":[null]}]`, "a pair of a class and its steps is no record", nil},
		{"class before steps", `[{"src":"s","by":[{"cls":false,"steps":false}]}]`, "class is no string", nil},
		{"missing steps", `[{"src":"s","by":[{"cls":"a"}]}]`, "s by a: <nil> is no list and no set", nil},
		{"first step before class duplicate", `[{"src":"s","by":[{"cls":"a","steps":[null]},{"cls":"a","steps":["new"]}]}]`, "s by a: step is no string", nil},
		{"first row before source duplicate", `[{"src":"s","by":[{"cls":"a","steps":[null]}]},` + row + `]`, "s by a: step is no string", nil},
	} {
		t.Run(specimen.name, func(t *testing.T) {
			input := []byte(specimen.input)
			pristine := slices.Clone(input)
			var raw any
			require.NoError(t, json.Unmarshal(input, &raw))
			callbacks := func() (func(any) (string, error), func(any) (string, error), func(any) (string, error), *[]string) {
				calls := []string{}
				read := func(role string) func(any) (string, error) {
					return func(raw any) (string, error) {
						calls = append(calls, fmt.Sprintf("%s:%v", role, raw))
						value, ok := raw.(string)
						if !ok {
							return "", fmt.Errorf("%s is no string", role)
						}
						return value, nil
					}
				}
				return read("source"), read("class"), read("step"), &calls
			}
			source, class, step, oldCalls := callbacks()
			want, oldErr := originalRawRows(raw, source, class, step)
			source, class, step, calls := callbacks()
			got, err := characterizedRawRows(input, source, class, step)
			require.Equal(t, oldErr, err)
			require.Equal(t, want, got)
			require.Equal(t, *oldCalls, *calls, "every callback, including the first failed row, in original order")
			if specimen.refusal == "" {
				require.NoError(t, err)
				require.Equal(t, specimen.want, got)
			} else {
				require.EqualError(t, err, specimen.refusal)
			}
			require.Equal(t, pristine, input)
		})
	}
}

func TestRawOwnersPreserveCompleteNativeViews(t *testing.T) {
	for _, fixture := range []struct {
		name string
		open func(*testing.T) *Slice
	}{
		{"monitored record", func(t *testing.T) *Slice { return openNamed(t, "activity-standalone-record") }},
		{"parameterized deadline owners", deadlineSlice},
		{"Nexus composition", func(t *testing.T) *Slice { return openNamed(t, "nexus-workflow") }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			model, pristine, modelBytes, name, machines, compositions, inputs, want := func() (*umpirespb.Model, *umpirespb.Model, []byte, string, []string, []string, [][]byte, []*machineView) {
				old := fixture.open(t)
				pristine := proto.Clone(old.Model).(*umpirespb.Model)
				modelBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(pristine)
				require.NoError(t, err)
				x := exported(t, old)
				switch fixture.name {
				case "monitored record":
					require.NotEmpty(t, old.machines["trustingActivityRecord"].Monitors)
				case "parameterized deadline owners":
					require.Equal(t, []string{"deadlineTimers", "simpleDeadlines"}, x.Machines)
				case "Nexus composition":
					require.NotEmpty(t, x.Compositions)
				default:
				}
				parts := dumpPartsOf(t, old, x)
				want := make([]*machineView, len(parts.encoded))
				for i, input := range parts.encoded {
					var part map[string]any
					require.NoError(t, json.Unmarshal(input, &part))
					if i < len(x.Machines) {
						want[i], err = x.legacyReader(i).machine(part)
					} else {
						r := x.composedReader(i - len(x.Machines))
						want[i], err = legacyITFReader(r).machine(part)
					}
					require.NoError(t, err)
					pristineOpenModel(t, pristine, modelBytes, old)
				}
				return old.Model, pristine, modelBytes, old.Name, slices.Clone(x.Machines), slices.Clone(x.Compositions), parts.encoded, want
			}()
			current := openSlice(t, proto.Clone(pristine).(*umpirespb.Model))
			current.Name = name
			x := exported(t, current)
			require.Equal(t, machines, x.Machines)
			require.Equal(t, compositions, x.Compositions)
			for i, input := range inputs {
				r := reader{}
				if i < len(x.Machines) {
					r = x.reader(i)
				} else {
					r = x.composedReader(i - len(x.Machines))
				}
				before := slices.Clone(input)
				got, present, err := characterizedRawMachine(r, input)
				require.NoError(t, err)
				require.True(t, present)
				require.Equal(t, want[i], got, "complete states/classes/results/claims/product, not sampled rows")
				if fixture.name == "monitored record" && i < len(x.Machines) && x.Machines[i] == "trustingActivityRecord" {
					require.NotNil(t, got.Product)
					require.NotEmpty(t, got.Product.Steps)
				}
				require.Equal(t, before, input)
				pristineOpenModel(t, pristine, modelBytes, current)
			}
			require.True(t, proto.Equal(pristine, model))
			originalBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(model)
			require.NoError(t, err)
			require.Equal(t, modelBytes, originalBytes)
		})
	}
}

func TestRawOwnerFieldsPreserveNativeRefusalOrder(t *testing.T) {
	s := deadlineSlice(t)
	x := exported(t, s)
	pristine := proto.Clone(s.Model).(*umpirespb.Model)
	modelBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(pristine)
	require.NoError(t, err)
	d, err := s.dumpOf(x, 0, s.machines[x.Machines[0]])
	require.NoError(t, err)
	faithful, err := json.Marshal(d)
	require.NoError(t, err)
	for _, specimen := range []struct{ name, input string }{
		{"faithful", string(faithful)},
		{"last starts valid", `{"starts":false,` + string(faithful[1:])},
		{"last starts invalid", string(faithful[:len(faithful)-1]) + `,"starts":false}`},
		{"null owner", `null`}, {"nonobject owner", `[]`}, {"missing fields", `{}`},
		{"first native field", `{"starts":false,"rows":false,"product":false}`},
		{"null starts", string(faithful[:len(faithful)-1]) + `,"starts":null}`},
		{"null rows", string(faithful[:len(faithful)-1]) + `,"rows":null}`},
		{"missing row source", string(faithful[:len(faithful)-1]) + `,"rows":[{"by":[]}]}`},
		{"class before step", string(faithful[:len(faithful)-1]) + `,"rows":[{"src":null,"by":[{"cls":false,"steps":false}]}]}`},
		{"null claims", string(faithful[:len(faithful)-1]) + `,"claims":null}`},
		{"bad claim reading", string(faithful[:len(faithful)-1]) + `,"claims":[null]}`},
		{"null product", string(faithful[:len(faithful)-1]) + `,"product":null}`},
		{"product starts before closed", string(faithful[:len(faithful)-1]) + `,"product":{"starts":null,"closed":false,"edges":null}}`},
		{"product closed before edges", string(faithful[:len(faithful)-1]) + `,"product":{"starts":[],"closed":null,"edges":null}}`},
		{"product edges", string(faithful[:len(faithful)-1]) + `,"product":{"starts":[],"closed":true,"edges":[null]}}`},
	} {
		t.Run(specimen.name, func(t *testing.T) {
			input := []byte(specimen.input)
			before := slices.Clone(input)
			var raw any
			require.NoError(t, json.Unmarshal(input, &raw))
			part, present := raw.(map[string]any)
			var want *machineView
			var oldErr error
			if present {
				want, oldErr = x.legacyReader(0).machine(part)
			}
			got, found, err := characterizedRawMachine(x.reader(0), input)
			require.Equal(t, present, found)
			require.Equal(t, oldErr, err)
			require.Equal(t, want, got)
			if specimen.name == "faithful" || specimen.name == "last starts valid" {
				require.NoError(t, err)
			} else if present {
				require.Error(t, err)
			}
			require.Equal(t, before, input)
			pristineOpenModel(t, pristine, modelBytes, s)
		})
	}
}
