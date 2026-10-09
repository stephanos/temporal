package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"google.golang.org/protobuf/proto"
)

func originalQuintOut(itf []byte) (map[string]any, error) {
	var trace struct {
		States []map[string]any `json:"states"`
	}
	if err := json.Unmarshal(itf, &trace); err != nil {
		return nil, fmt.Errorf("the Quint dump is no ITF trace: %w", err)
	}
	if len(trace.States) == 0 {
		return nil, errors.New("the Quint dump holds no state")
	}
	out, ok := trace.States[0]["out"].(map[string]any)
	if !ok {
		return nil, errors.New("the Quint dump's first state has no variable out")
	}
	return out, nil
}

func TestQuintDumpIndexPreservesTheOriginalDecoder(t *testing.T) {
	valid := `{"states":[{"out":{"m0":{"tag":"first","n":1.25},"m1":null,"c0":[true,3]}}]}`
	for _, fixture := range []struct{ name, json string }{
		{"valid", valid},
		{"whitespace", " \n\t" + valid + "\r\n "},
		{"pretty", "{\n\"states\": [ { \"out\": { \"m0\": {\"x\": true} } } ]\n}"},
		{"null top", `null`}, {"empty top", `{}`}, {"array top", `[]`},
		{"number top", `1`}, {"string top", `"x"`}, {"bool top", `false`},
		{"null states", `{"states":null}`}, {"empty states", `{"states":[]}`},
		{"object states", `{"states":{}}`}, {"array-shaped states", `{"states":[[]]}`},
		{"number states", `{"states":1}`}, {"overflow-shaped states", `{"states":1e1000}`},
		{"string states", `{"states":"x"}`}, {"bool states", `{"states":true}`},
		{"null first", `{"states":[null]}`}, {"omitted out", `{"states":[{}]}`},
		{"null out", `{"states":[{"out":null}]}`}, {"number out", `{"states":[{"out":1}]}`},
		{"array out", `{"states":[{"out":[]}]}`}, {"empty out", `{"states":[{"out":{}}]}`},
		{"wrong part shapes", `{"states":[{"out":{"m0":false,"m1":[],"c0":"x"}}]}`},
		{"null later", `{"states":[{"out":{}},null]}`},
		{"number later", `{"states":[{"out":{}},2]}`},
		{"string later", `{"states":[{"out":{}},"bad"]}`},
		{"array later", `{"states":[{"out":{}},[]]}`},
		{"unknown top overflow", `{"ignored":1e1000,"states":[{"out":{}}]}`},
		{"unknown top nested overflow", `{"ignored":{"a":[1e1000]},"states":[{"out":{}}]}`},
		{"unknown state overflow", `{"states":[{"out":{},"ignored":1e1000}]}`},
		{"folded overflow path", `{"StAtEs":[{"a~/b":{"": [1e1000]}}]}`},
		{"whitespace overflow offset", " \n {\"states\": [{\"a\": [true, -1e1000]}]} \n"},
		{"Unicode folded later shape path", `{"ſtateſ":[{"out":{}},[]]}`},
		{"escaped nested error path", " \n {\"StAtEs\": [{\"out\": {\"m0\": {\"a.b~/\\n\": [true, 1e1000]}}}]} \n"},
		{"unknown part overflow", `{"states":[{"out":{"ignored":{"n":1e1000}}}]}`},
		{"later state overflow", `{"states":[{"out":{}},{"ignored":1e1000}]}`},
		{"underflow", `{"states":[{"out":{"m0":1e-1000}}]}`},
		{"finite boundary", `{"states":[{"out":{"m0":1.7976931348623157e308}}]}`},
		{"negative overflow", `{"states":[{"out":{"m0":-1e1000}}]}`},
		{"bigint string", `{"states":[{"out":{"m0":{"#bigint":"9007199254740993"}}}]}`},
		{"escaped keys and values", `{"states":[{"out":{"m\u0030":{"x":"<\"&\n>"}}}]}`},
		{"surrogate repair", `{"states":[{"out":{"m0":"\ud800"}}]}`},
		{"invalid UTF8 repair", "{\"states\":[{\"out\":{\"m0\":\"\xff\"}}]}"},
		{"upper States", `{"STATES":[{"out":{"m0":1}}]}`},
		{"mixed States", `{"StAtEs":[{"out":{"m0":1}}]}`},
		{"Unicode folded States", `{"ſtateſ":[{"out":{"m0":1}}]}`},
		{"reused first map retains out", `{"states":[{"out":{"m0":1}}],"states":[{"other":true}]}`},
		{"empty resets map", `{"states":[{"out":{"m0":1}}],"states":[],"states":[{}]}`},
		{"null resets map", `{"states":[{"out":{"m0":1}}],"states":null,"states":[{}]}`},
		{"null first clears map", `{"states":[{"out":{"m0":1}}],"states":[null],"states":[{}]}`},
		{"explicit out replaces", `{"states":[{"out":{"m0":1}}],"states":[{"out":{"c0":2}}]}`},
		{"duplicate out replaces", `{"states":[{"out":{"m0":1},"out":{"c0":2}}]}`},
		{"duplicate part last wins", `{"states":[{"out":{"m0":1,"m0":2}}]}`},
		{"overwritten states shape remembered", `{"states":1,"states":[{"out":{}}]}`},
		{"overwritten part overflow remembered", `{"states":[{"out":{"m0":1e1000,"m0":2}}]}`},
		{"overwritten out overflow remembered", `{"states":[{"out":{"x":1e1000},"out":{}}]}`},
		{"shape precedes overflow", `{"states":[2,{"x":1e1000}]}`},
		{"overflow precedes shape", `{"states":[{"x":1e1000},2]}`},
		{"wrong states skips overflowing subtree", `{"states":{"x":1e1000}}`},
		{"wrong state skips overflowing subtree", `{"states":[[1e1000]]}`},
		{"malformed suffix", valid + "x"}, {"second JSON value", valid + `{}`},
		{"syntax before shape", `{"states":1,"x":}`},
		{"syntax before overflow", `{"states":[{"x":1e1000}],"x":}`},
		{"depth error", `{"states":[{"out":{"m0":` + strings.Repeat("[", 10001) + "0" + strings.Repeat("]", 10001) + `}}]}`},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			input := []byte(fixture.json)
			want, oldErr := originalQuintOut(input)
			for _, wanted := range [][]string{{"m0", "c0", "m1"}, {"c0", "m1", "m0"}, nil} {
				got, err := indexQuintDump(input, wanted)
				require.Equal(t, reflect.TypeOf(oldErr), reflect.TypeOf(err))
				if oldErr != nil {
					require.EqualError(t, err, oldErr.Error())
					var oldSyntax, syntax *json.SyntaxError
					if errors.As(oldErr, &oldSyntax) {
						require.ErrorAs(t, err, &syntax)
						require.Equal(t, oldSyntax.Offset, syntax.Offset)
					}
					var oldType, actualType *json.UnmarshalTypeError
					if errors.As(oldErr, &oldType) {
						require.ErrorAs(t, err, &actualType)
						require.Equal(t, *oldType, *actualType)
					}
					continue
				}
				require.NoError(t, err)
				for _, key := range wanted {
					part, err := got.part(input, key)
					require.NoError(t, err)
					require.Equal(t, want[key], part, key)
				}
			}
		})
	}
}

func TestQuintDumpAdmissionDoesNotBuildEveryOwnerGraph(t *testing.T) {
	out := map[string]any{}
	wanted := []string{}
	for owner := range 64 {
		rows := []any{}
		for row := range 64 {
			rows = append(rows, map[string]any{"node": map[string]any{"leaf": map[string]any{}, "name": fmt.Sprint(row)}})
		}
		key := fmt.Sprintf("m%d", owner)
		wanted = append(wanted, key)
		out[key] = map[string]any{"rows": rows}
	}
	input, err := json.Marshal(map[string]any{"states": []any{map[string]any{"out": out}}})
	require.NoError(t, err)
	old, err := originalQuintOut(input)
	require.NoError(t, err)
	indexed, err := indexQuintDump(input, wanted)
	require.NoError(t, err)
	for _, key := range wanted {
		value, err := indexed.part(input, key)
		require.NoError(t, err)
		require.Equal(t, old[key], value)
	}
	measure := func(decode func() any) int64 {
		return testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				runtime.KeepAlive(decode())
			}
		}).AllocedBytesPerOp()
	}
	legacy := measure(func() any {
		value, err := originalQuintOut(input)
		if err != nil {
			panic(err)
		}
		return value
	})
	current := measure(func() any {
		value, err := indexQuintDump(input, wanted)
		if err != nil {
			panic(err)
		}
		return value
	})
	t.Logf("admission allocated bytes: original=%d current=%d input=%d", legacy, current, len(input))
	require.Less(t, current, legacy*9/10, "admission must avoid material owner-graph allocation, not benchmark noise")
}

func originalQuintAgreement(s *Slice, x *QuintExport, itf []byte) ([]Receipt, error) {
	out, err := originalQuintOut(itf)
	if err != nil {
		return nil, err
	}
	var receipts []Receipt
	var fresh *Slice
	replay := func(machine, monitor string, trace *check.Trace) error {
		if fresh == nil {
			var err error
			if fresh, err = Open(s.Model); err != nil {
				return err
			}
		}
		return fresh.replay(machine, monitor, trace)
	}
	for i := range x.Machines {
		compared, err := s.machineReceipts(x, i, out, replay)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	for j := range x.Compositions {
		compared, err := s.compositionReceipts(x, j, out)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	receipts = append(receipts, x.Unsupported...)
	for i := range receipts {
		receipts[i].Model = s.Name
	}
	return receipts, s.confirm(receipts, true)
}

func TestQuintDumpIndexPreservesTheCompleteAgreement(t *testing.T) {
	type comparison struct {
		name string
		dump []byte
		want []Receipt
		err  error
	}
	for _, fixture := range []struct {
		name string
		open func(*testing.T) *Slice
	}{
		{"record with ordered named alternatives", func(t *testing.T) *Slice { return namedChoices(t, "accepts", "rejects") }},
		{"Nexus composition", func(t *testing.T) *Slice { return openNamed(t, "nexus-workflow") }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			originalModel, pristine, name, machines, compositions, inputs := func() (*umpirespb.Model, *umpirespb.Model, string, []string, []string, []comparison) {
				original := fixture.open(t)
				pristine := proto.Clone(original.Model).(*umpirespb.Model)
				oldExport := exported(t, original)
				parts := dumpPartsOf(t, original, oldExport)
				var inputs []comparison
				for _, input := range []struct {
					name   string
					tamper func(string, map[string]any)
				}{
					{"faithful", nil},
					{"open frontier", func(_ string, part map[string]any) { part["closed"] = false }},
					{"typed refusal", func(_ string, part map[string]any) { part["starts"] = false }},
				} {
					dump := parts.encode(t, input.tamper)
					want, err := originalQuintAgreement(original, oldExport, dump)
					require.True(t, proto.Equal(pristine, original.Model))
					inputs = append(inputs, comparison{name: input.name, dump: dump, want: want, err: err})
				}
				require.True(t, proto.Equal(pristine, original.Model))
				return original.Model, pristine, original.Name, slices.Clone(oldExport.Machines), slices.Clone(oldExport.Compositions), inputs
			}()
			indexed := openSlice(t, proto.Clone(pristine).(*umpirespb.Model))
			indexed.Name = name
			currentExport := exported(t, indexed)
			require.Equal(t, machines, currentExport.Machines)
			require.Equal(t, compositions, currentExport.Compositions)
			for _, input := range inputs {
				t.Run(input.name, func(t *testing.T) {
					got, err := indexed.QuintAgreement(currentExport, input.dump)
					if input.err != nil {
						require.EqualError(t, err, input.err.Error())
					} else {
						require.NoError(t, err)
						require.NotEmpty(t, input.want)
					}
					require.Equal(t, input.want, got)
					require.True(t, proto.Equal(pristine, originalModel))
					require.True(t, proto.Equal(pristine, indexed.Model))
				})
			}
		})
	}
}
