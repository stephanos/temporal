package ir

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

func fieldPath(names ...string) string {
	return strings.Join(names, ".")
}

func TestPathsPreserveTypePresenceAndCardinality(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	lookup := `labels["key"]`
	for _, tt := range []struct {
		name           string
		path           string
		kind           testpilotspb.ScalarKind
		cardinality    Cardinality
		absent, fanout bool
	}{
		{"nested", "child.text", testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"optional", "optional_text", testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"presence", "child.optional_text?", testpilotspb.SCALAR_KIND_BOOLEAN, Singular, false, false},
		{"oneof", "result<success>", testpilotspb.SCALAR_KIND_TEXT, Singular, true, false},
		{"wildcard", "items[*].text", testpilotspb.SCALAR_KIND_TEXT, Repeated, false, true},
		{"map lookup", lookup, testpilotspb.SCALAR_KIND_INT64, Singular, true, false},
		{"map", "labels", testpilotspb.SCALAR_KIND_INT64, Map, false, false},
		{"whole list", "items", testpilotspb.SCALAR_KIND_UNSPECIFIED, Repeated, false, false},
		{"wkt", "when.seconds", testpilotspb.SCALAR_KIND_INT64, Singular, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := c.BindPath(source, "path", tt.path, DefaultLimits())
			require.NoError(t, err)
			require.Equal(t, tt.kind, p.Type().Scalar())
			require.Equal(t, tt.cardinality, p.Type().Cardinality())
			require.Equal(t, tt.absent, p.MayBeAbsent())
			require.Equal(t, tt.fanout, p.Fanout())
		})
	}
	p, err := c.BindPath(source, "path", lookup, DefaultLimits())
	require.NoError(t, err)
	steps := p.Steps()
	steps[0].Key.Value = &testpilotspb.Value_TextValue{TextValue: "mutated"}
	require.Equal(t, "labels", string(p.Steps()[0].Field.Name()))
	require.True(t, proto.Equal(text("key"), p.Steps()[0].Key))
	total, err := p.CheckFanout(2, 3)
	require.NoError(t, err)
	require.EqualValues(t, 6, total)
	_, err = p.CheckFanout(math.MaxInt64, 2)
	require.Error(t, err)
}

func TestPathsRejectInvalidSelectorsAndTraversal(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	for name, path := range map[string]string{
		"missing": "missing", "scalar traversal": "text.x", "any traversal": "payload.value", "list traversal": "items.text", "map traversal": "labels.value", "wrong wildcard": "text[*].text",
		"wrong oneof member": "result<text>", "no presence": "text?", "nested collection": "items[*].items",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := c.BindPath(source, "path", path, DefaultLimits())
			require.Error(t, err)
		})
	}
	_, err := c.BindPath(c.OpaqueHandleType(), "path", "", DefaultLimits())
	require.Error(t, err)
	p, err := c.BindPath(source, "path", "payload", DefaultLimits())
	require.NoError(t, err)
	require.True(t, p.Type().Any())
	limits := DefaultLimits()
	limits.Fanout = math.MaxInt64
	_, err = c.BindPath(source, "path", "text", limits)
	require.Error(t, err)
}

// Every segment kind reads back from its canonical spelling, and a text key's escapes normalize to the
// spelling Testpilot.Authoring.Path writes.
func TestPathGrammarRoundTripsEverySegmentKind(t *testing.T) {
	for _, tt := range []struct {
		spelling string
		want     []pathSegment
	}{
		{"", nil},
		{"text", []pathSegment{{field: "text"}}},
		{"history.events[*]", []pathSegment{{field: "history"}, {field: "events", selector: Wildcard}}},
		{`labels["a \"b\"\\\n\r\u0001é"]`, []pathSegment{{field: "labels", selector: MapKey, key: pathKey{kind: textKey, text: "a \"b\"\\\n\r\x01é"}}}},
		{"counts[-42]", []pathSegment{{field: "counts", selector: MapKey, key: pathKey{kind: integerKey, text: "-42"}}}},
		{"flags[true]", []pathSegment{{field: "flags", selector: MapKey, key: pathKey{kind: booleanKey, text: "true"}}}},
		{"child.optional_text?", []pathSegment{{field: "child"}, {field: "optional_text", selector: Presence}}},
		{"attributes<nexus_operation_completed_event_attributes>.scheduled_event_id", []pathSegment{
			{field: "attributes", selector: Oneof, member: "nexus_operation_completed_event_attributes"}, {field: "scheduled_event_id"},
		}},
	} {
		t.Run(tt.spelling, func(t *testing.T) {
			parsed, err := parsePath(tt.spelling)
			require.NoError(t, err)
			require.Equal(t, tt.want, parsed)
			require.Equal(t, tt.spelling, formatPath(parsed))
		})
	}
	parsed, err := parsePath(`labels["\u0041\/"]`)
	require.NoError(t, err)
	require.Equal(t, `labels["A/"]`, formatPath(parsed))
}

// A path outside the grammar, or one whose map key does not match its map, rejects at its location
// quoting its text.
func TestPathsOutsideTheGrammarRejectWithTheirText(t *testing.T) {
	c := fixtureCatalog(t)
	source := boundType(t, c, named("fixture.Payload", false))
	for _, tt := range []struct {
		name, path string
		category   ErrorCategory
		detail     string
	}{
		{"unterminated key", `labels["key]`, Malformed, "unterminated map key"},
		{"unterminated selector", "items[*", Malformed, "unterminated selector"},
		{"unknown selector", "items[first]", Malformed, "unknown selector [first]"},
		{"unknown selector character", "text!", Malformed, `unknown selector "!"`},
		{"map key of the wrong kind", "labels[42]", TypeMismatch, "does not match the map's SCALAR_KIND_TEXT keys"},
		{"empty segment", "child..text", Malformed, "empty segment"},
		{"trailing separator", "child.", Malformed, "empty segment"},
		{"unterminated oneof member", "result<success", Malformed, "unterminated oneof member"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.BindPath(source, "program.read.path", tt.path, DefaultLimits())
			var rejected *Error
			require.ErrorAs(t, err, &rejected)
			require.Equal(t, tt.category, rejected.Category)
			require.Equal(t, "program.read.path", rejected.Path)
			require.Contains(t, rejected.Detail, `path "`+strings.ReplaceAll(tt.path, `"`, `\"`)+`"`)
			require.Contains(t, rejected.Detail, tt.detail)
		})
	}
}

// Integer and boolean keys take the kind of the map they select from.
func TestPathKeysTakeTheirMapKeyKind(t *testing.T) {
	for _, tt := range []struct {
		key  pathKey
		kind testpilotspb.ScalarKind
		want *testpilotspb.Value
	}{
		{pathKey{kind: textKey, text: "x"}, testpilotspb.SCALAR_KIND_TEXT, text("x")},
		{pathKey{kind: booleanKey, text: "true"}, testpilotspb.SCALAR_KIND_BOOLEAN, boolean(true)},
		{pathKey{kind: integerKey, text: "-1"}, testpilotspb.SCALAR_KIND_SINT32, signed("-1")},
		{pathKey{kind: integerKey, text: "7"}, testpilotspb.SCALAR_KIND_FIXED64, unsigned("7")},
		{pathKey{kind: textKey, text: "7"}, testpilotspb.SCALAR_KIND_INT64, nil},
		{pathKey{kind: integerKey, text: "1"}, testpilotspb.SCALAR_KIND_BOOLEAN, nil},
	} {
		value, err := tt.key.value(tt.kind)
		if tt.want == nil {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.True(t, proto.Equal(tt.want, value))
	}
}
