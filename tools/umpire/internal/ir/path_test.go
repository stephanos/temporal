package ir

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func fieldPath(names ...string) *umpirespb.FieldPath {
	path := &umpirespb.FieldPath{}
	for _, name := range names {
		path.Segments = append(path.Segments, &umpirespb.FieldPathSegment{Field: name})
	}
	return path
}
func TestPathsPreserveTypePresenceAndCardinality(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	oneof := &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: "result", Selector: &umpirespb.FieldPathSegment_Oneof{Oneof: &umpirespb.OneofSelector{SelectedField: "success"}}}}}
	presence := fieldPath("child", "optional_text")
	presence.Segments[1].Selector = &umpirespb.FieldPathSegment_Presence{Presence: &umpirespb.PresenceSelector{}}
	wildcard := fieldPath("items", "text")
	wildcard.Segments[0].Selector = &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}
	lookup := fieldPath("labels")
	lookup.Segments[0].Selector = &umpirespb.FieldPathSegment_MapKey{MapKey: &umpirespb.MapKeySelector{Key: text("key")}}
	for _, tt := range []struct {
		name           string
		path           *umpirespb.FieldPath
		kind           umpirespb.ScalarKind
		cardinality    Cardinality
		absent, fanout bool
	}{
		{"nested", fieldPath("child", "text"), umpirespb.SCALAR_KIND_TEXT, Singular, true, false},
		{"optional", fieldPath("optional_text"), umpirespb.SCALAR_KIND_TEXT, Singular, true, false},
		{"presence", presence, umpirespb.SCALAR_KIND_BOOLEAN, Singular, false, false},
		{"oneof", oneof, umpirespb.SCALAR_KIND_TEXT, Singular, true, false},
		{"wildcard", wildcard, umpirespb.SCALAR_KIND_TEXT, Repeated, false, true},
		{"map lookup", lookup, umpirespb.SCALAR_KIND_INT64, Singular, true, false},
		{"map", fieldPath("labels"), umpirespb.SCALAR_KIND_INT64, Map, false, false},
		{"whole list", fieldPath("items"), umpirespb.SCALAR_KIND_UNSPECIFIED, Repeated, false, false},
		{"wkt", fieldPath("when", "seconds"), umpirespb.SCALAR_KIND_INT64, Singular, true, false},
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
	steps[0].Key.Value = &umpirespb.Value_Text{Text: "mutated"}
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
	wildcard := func(field string) *umpirespb.FieldPath {
		p := fieldPath(field, "text")
		p.Segments[0].Selector = &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}
		return p
	}
	for name, path := range map[string]*umpirespb.FieldPath{
		"nil": nil, "nil segment": {Segments: []*umpirespb.FieldPathSegment{nil}}, "missing": fieldPath("missing"), "scalar traversal": fieldPath("text", "x"), "any traversal": fieldPath("payload", "value"), "list traversal": fieldPath("items", "text"), "map traversal": fieldPath("labels", "value"), "wrong wildcard": wildcard("text"),
		"wrong oneof member": {Segments: []*umpirespb.FieldPathSegment{{Field: "result", Selector: &umpirespb.FieldPathSegment_Oneof{Oneof: &umpirespb.OneofSelector{SelectedField: "text"}}}}},
		"no presence":        {Segments: []*umpirespb.FieldPathSegment{{Field: "text", Selector: &umpirespb.FieldPathSegment_Presence{Presence: &umpirespb.PresenceSelector{}}}}},
		"wrong map key":      {Segments: []*umpirespb.FieldPathSegment{{Field: "labels", Selector: &umpirespb.FieldPathSegment_MapKey{MapKey: &umpirespb.MapKeySelector{Key: unsigned("1")}}}}},
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
	opaque := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}}
	_, err = c.BindPath(boundType(t, c, opaque), &umpirespb.FieldPath{}, DefaultLimits())
	require.Error(t, err)
	p, err := c.BindPath(source, fieldPath("payload"), DefaultLimits())
	require.NoError(t, err)
	require.True(t, p.Type().Any())
	limits := DefaultLimits()
	limits.Fanout = math.MaxInt64
	_, err = c.BindPath(source, fieldPath("text"), limits)
	require.Error(t, err)
}
