package ir

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

func fieldPath(names ...string) *testpilotspb.FieldPath {
	path := &testpilotspb.FieldPath{}
	for _, name := range names {
		path.Segments = append(path.Segments, &testpilotspb.FieldPathSegment{Field: name})
	}
	return path
}
func TestPathsPreserveTypePresenceAndCardinality(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	oneof := &testpilotspb.FieldPath{Segments: []*testpilotspb.FieldPathSegment{{Field: "result", Selector: &testpilotspb.FieldPathSegment_Oneof{Oneof: &testpilotspb.OneofSelector{SelectedField: "success"}}}}}
	presence := fieldPath("child", "optional_text")
	presence.Segments[1].Selector = &testpilotspb.FieldPathSegment_Presence{Presence: &testpilotspb.PresenceSelector{}}
	wildcard := fieldPath("items", "text")
	wildcard.Segments[0].Selector = &testpilotspb.FieldPathSegment_Repeated{Repeated: &testpilotspb.RepeatedWildcard{}}
	lookup := fieldPath("labels")
	lookup.Segments[0].Selector = &testpilotspb.FieldPathSegment_MapKey{MapKey: &testpilotspb.MapKeySelector{Key: text("key")}}
	for _, tt := range []struct {
		name           string
		path           *testpilotspb.FieldPath
		kind           testpilotspb.ScalarKind
		cardinality    Cardinality
		absent, fanout bool
	}{
		{"nested", fieldPath("child", "text"), testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"optional", fieldPath("optional_text"), testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"presence", presence, testpilotspb.SCALAR_KIND_BOOLEAN, Singular, false, false},
		{"oneof", oneof, testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"wildcard", wildcard, testpilotspb.SCALAR_KIND_TEXT, Repeated, false, true},
		{"map lookup", lookup, testpilotspb.SCALAR_KIND_INT64, Singular, true, false},
		{"map", fieldPath("labels"), testpilotspb.SCALAR_KIND_INT64, Map, false, false},
		{"whole list", fieldPath("items"), testpilotspb.SCALAR_KIND_UNSPECIFIED, Repeated, false, false},
		{"wkt", fieldPath("when", "seconds"), testpilotspb.SCALAR_KIND_INT64, Singular, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := c.BindPath(source, tt.path, DefaultLimits())
			require.NoError(t, err)
			require.Equal(t, tt.kind, p.Type().Scalar())
			require.Equal(t, tt.cardinality, p.Type().Cardinality())
			require.Equal(t, tt.absent, p.MayBeAbsent())
			require.Equal(t, tt.fanout, p.Fanout())
		})
	}
	original := proto.CloneOf(lookup)
	p, err := c.BindPath(source, lookup, DefaultLimits())
	require.NoError(t, err)
	lookup.Segments[0].Field = "unknown"
	steps := p.Steps()
	steps[0].Key.Value = &testpilotspb.Value_Text{Text: "mutated"}
	require.Equal(t, "labels", string(p.Steps()[0].Field.Name()))
	require.True(t, proto.Equal(original.Segments[0].GetMapKey().Key, p.Steps()[0].Key))
	total, err := p.CheckFanout(2, 3)
	require.NoError(t, err)
	require.EqualValues(t, 6, total)
	_, err = p.CheckFanout(math.MaxInt64, 2)
	require.Error(t, err)
}

func TestPathsRejectInvalidSelectorsAndTraversal(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	wildcard := func(field string) *testpilotspb.FieldPath {
		p := fieldPath(field, "text")
		p.Segments[0].Selector = &testpilotspb.FieldPathSegment_Repeated{Repeated: &testpilotspb.RepeatedWildcard{}}
		return p
	}
	for name, path := range map[string]*testpilotspb.FieldPath{
		"nil": nil, "nil segment": {Segments: []*testpilotspb.FieldPathSegment{nil}}, "missing": fieldPath("missing"), "scalar traversal": fieldPath("text", "x"), "any traversal": fieldPath("payload", "value"), "list traversal": fieldPath("items", "text"), "map traversal": fieldPath("labels", "value"), "wrong wildcard": wildcard("text"),
		"wrong oneof member": {Segments: []*testpilotspb.FieldPathSegment{{Field: "result", Selector: &testpilotspb.FieldPathSegment_Oneof{Oneof: &testpilotspb.OneofSelector{SelectedField: "text"}}}}},
		"no presence":        {Segments: []*testpilotspb.FieldPathSegment{{Field: "text", Selector: &testpilotspb.FieldPathSegment_Presence{Presence: &testpilotspb.PresenceSelector{}}}}},
		"wrong map key":      {Segments: []*testpilotspb.FieldPathSegment{{Field: "labels", Selector: &testpilotspb.FieldPathSegment_MapKey{MapKey: &testpilotspb.MapKeySelector{Key: unsigned("1")}}}}},
		"nested collection":  wildcard("items"),
	} {
		t.Run(name, func(t *testing.T) {
			if name == "nested collection" {
				path.Segments[1].Field = "items"
			}
			_, err := c.BindPath(source, path, DefaultLimits())
			require.Error(t, err)
		})
	}
	unknown := fieldPath("text")
	unknown.Segments[0].ProtoReflect().SetUnknown([]byte{0x78, 1})
	_, err := c.BindPath(source, unknown, DefaultLimits())
	require.Error(t, err)
	opaque := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}}}}
	_, err = c.BindPath(boundType(t, c, opaque), &testpilotspb.FieldPath{}, DefaultLimits())
	require.Error(t, err)
	p, err := c.BindPath(source, fieldPath("payload"), DefaultLimits())
	require.NoError(t, err)
	require.True(t, p.Type().Any())
	limits := DefaultLimits()
	limits.Fanout = math.MaxInt64
	_, err = c.BindPath(source, fieldPath("text"), limits)
	require.Error(t, err)
}
