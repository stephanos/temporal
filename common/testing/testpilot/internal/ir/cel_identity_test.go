package ir

import (
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func parsedCEL(t *testing.T, input string) *celpb.ParsedExpr {
	t.Helper()
	parsed := new(celpb.ParsedExpr)
	require.NoError(t, protojson.Unmarshal([]byte(input), parsed))
	return parsed
}

func TestCELIdentityCanonicalOrderAndDiagnostics(t *testing.T) {
	first := parsedCEL(t, `{"expr":{"id":"80","structExpr":{"entries":[{"id":"81","mapKey":{"id":"82","constExpr":{"stringValue":"z"}},"value":{"id":"83","constExpr":{"int64Value":"2"}}},{"id":"84","mapKey":{"id":"85","constExpr":{"stringValue":"a"}},"value":{"id":"86","constExpr":{"int64Value":"1"}}}]}},"sourceInfo":{"location":"author.scala","positions":{"80":0,"81":20,"82":21,"83":22,"84":10,"85":11,"86":12}}}`)
	snapshot := proto.CloneOf(first)
	canonical, err := CanonicalCEL(first)
	require.NoError(t, err)
	want := parsedCEL(t, `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":{"stringValue":"a"}},"value":{"id":"4","constExpr":{"int64Value":"1"}}},{"id":"5","mapKey":{"id":"6","constExpr":{"stringValue":"z"}},"value":{"id":"7","constExpr":{"int64Value":"2"}}}]}},"sourceInfo":{"location":"author.scala","positions":{"1":0,"2":10,"3":11,"4":12,"5":20,"6":21,"7":22}}}`)
	require.True(t, proto.Equal(want, canonical), canonical)
	require.True(t, proto.Equal(snapshot, first), "canonicalization must not mutate caller data")
	again, err := CanonicalCEL(canonical)
	require.NoError(t, err)
	require.True(t, proto.Equal(canonical, again))
	bytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	require.NoError(t, err)
	canonical.SourceInfo.Location = "moved.scala"
	moved, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	require.NoError(t, err)
	require.NotEqual(t, bytes, moved, "diagnostic metadata participates in exact artifact identity")
}

func TestCELIdentityPreservesEvaluationOrder(t *testing.T) {
	source := parsedCEL(t, `{"expr":{"id":"10","callExpr":{"function":"_?_:_","args":[{"id":"90","identExpr":{"name":"ready"}},{"id":"20","selectExpr":{"operand":{"id":"60","identExpr":{"name":"response"}},"field":"value"}},{"id":"50","listExpr":{"elements":[{"id":"70","constExpr":{"int64Value":"2"}},{"id":"80","constExpr":{"int64Value":"1"}}]}}]}}}`)
	canonical, err := CanonicalCEL(source)
	require.NoError(t, err)
	want := parsedCEL(t, `{"expr":{"id":"1","callExpr":{"function":"_?_:_","args":[{"id":"2","identExpr":{"name":"ready"}},{"id":"3","selectExpr":{"operand":{"id":"4","identExpr":{"name":"response"}},"field":"value"}},{"id":"5","listExpr":{"elements":[{"id":"6","constExpr":{"int64Value":"2"}},{"id":"7","constExpr":{"int64Value":"1"}}]}}]}}}`)
	require.True(t, proto.Equal(want, canonical), canonical)
}

func TestCELIdentityMapKeyOrder(t *testing.T) {
	for _, keys := range []struct {
		first, second string
	}{
		{`{"boolValue":true}`, `{"boolValue":false}`},
		{`{"int64Value":"10"}`, `{"int64Value":"2"}`},
		{`{"uint64Value":"18446744073709551615"}`, `{"uint64Value":"2"}`},
	} {
		t.Run(keys.first, func(t *testing.T) {
			source := parsedCEL(t, `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":`+keys.first+`},"value":{"id":"4","constExpr":{"boolValue":true}}},{"id":"5","mapKey":{"id":"6","constExpr":`+keys.second+`},"value":{"id":"7","constExpr":{"boolValue":false}}}]}}}`)
			canonical, err := CanonicalCEL(source)
			require.NoError(t, err)
			want := new(celpb.Constant)
			require.NoError(t, protojson.Unmarshal([]byte(keys.second), want))
			require.True(t, proto.Equal(want, canonical.Expr.GetStructExpr().Entries[0].GetMapKey().GetConstExpr()))
		})
	}
}

func TestCELIdentityRejectsUnsafeConstruction(t *testing.T) {
	for name, input := range map[string]string{
		"dynamic map value":     `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":{"stringValue":"a"}},"value":{"id":"4","identExpr":{"name":"missing"}}}]}}}`,
		"dynamic map key":       `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","identExpr":{"name":"key"}},"value":{"id":"4","constExpr":{"boolValue":true}}}]}}}`,
		"duplicate map key":     `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":{"stringValue":"a"}},"value":{"id":"4","constExpr":{"boolValue":true}}},{"id":"5","mapKey":{"id":"6","constExpr":{"stringValue":"a"}},"value":{"id":"7","constExpr":{"boolValue":false}}}]}}}`,
		"mixed map key types":   `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":{"int64Value":"1"}},"value":{"id":"4","constExpr":{"boolValue":true}}},{"id":"5","mapKey":{"id":"6","constExpr":{"uint64Value":"1"}},"value":{"id":"7","constExpr":{"boolValue":false}}}]}}}`,
		"optional map entry":    `{"expr":{"id":"1","structExpr":{"entries":[{"id":"2","mapKey":{"id":"3","constExpr":{"stringValue":"a"}},"value":{"id":"4","constExpr":{"boolValue":true}},"optionalEntry":true}]}}}`,
		"message constructor":   `{"expr":{"id":"1","structExpr":{"messageName":"example.Message","entries":[]}}}`,
		"duplicate IDs":         `{"expr":{"id":"1","listExpr":{"elements":[{"id":"1","constExpr":{"boolValue":true}}]}}}`,
		"unknown diagnostic ID": `{"expr":{"id":"1","constExpr":{"boolValue":true}},"sourceInfo":{"positions":{"9":10}}}`,
		"macro":                 `{"expr":{"id":"1","constExpr":{"boolValue":true}},"sourceInfo":{"macroCalls":{"1":{"id":"2","constExpr":{"boolValue":true}}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			canonical, err := CanonicalCEL(parsedCEL(t, input))
			require.Nil(t, canonical)
			require.Error(t, err)
		})
	}
}
