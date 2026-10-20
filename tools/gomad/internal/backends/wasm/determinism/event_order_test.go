package determinism

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func orderedLogFixture(t *testing.T, events []OrderedEvent, marker *EventBoundary) []byte {
	t.Helper()
	line := func(value any) []byte {
		t.Helper()
		data, err := json.Marshal(value)
		require.NoError(t, err)
		return []byte(EventOrderPrefix + string(data) + "\n")
	}
	data := line(map[string]any{"type": "header", "schema": EventOrderSchema})
	if marker != nil {
		data = append(data, line(struct {
			Type string `json:"type"`
			EventBoundary
		}{"boundary", *marker})...)
	}
	for _, event := range events {
		data = append(data, line(struct {
			Type string `json:"type"`
			OrderedEvent
		}{"event", event})...)
	}
	return append(data, line(map[string]any{"type": "complete", "events": len(events), "bytes": len(data)})...)
}

func TestEventOrderFramingRejectsPartialAndUncommittedDiagnostics(t *testing.T) {
	t.Parallel()
	events := []OrderedEvent{{Order: 1, Kind: KindClock, Instance: "guest", Incarnation: 1, TimeNanos: 42, Value: json.RawMessage(`{"timestamp":42}`)}}
	data := orderedLogFixture(t, events, nil)
	log, err := ParseEventOrder(append([]byte("other helper telemetry\n"), data...))
	require.NoError(t, err)
	require.True(t, log.Complete)
	require.Equal(t, events, log.Events)
	for _, test := range []struct {
		name string
		data string
	}{
		{"unknown field", strings.Replace(string(data), `"kind":"clock"`, `"kind":"clock","extra":true`, 1)},
		{"wrong ordinal", strings.Replace(string(data), `"order":1`, `"order":2`, 1)},
		{"missing owner", strings.Replace(string(data), `"instance":"guest"`, `"instance":""`, 1)},
		{"missing incarnation", strings.Replace(string(data), `"incarnation":1`, `"incarnation":0`, 1)},
		{"wrong count", strings.Replace(string(data), `"events":1`, `"events":2`, 1)},
		{"frame after terminal", string(data) + EventOrderPrefix + `{"type":"header","schema":"` + EventOrderSchema + `"}` + "\n"},
		{"truncated frame after terminal", string(data) + EventOrderPrefix + "{"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseEventOrder([]byte(test.data))
			require.Error(t, err)
		})
	}
	for _, cut := range []int{0, len(data) - 1, strings.LastIndex(string(data), EventOrderPrefix)} {
		partial, err := ParseEventOrder(data[:cut])
		require.NoError(t, err)
		require.False(t, partial.Complete)
		require.NotEmpty(t, partial.Reason)
	}
}

func TestEventOrderFindsEarlierOutputWithinOneInspectionInterval(t *testing.T) {
	t.Parallel()
	diagnostic, err := choice.BuildDiagnosticTrace(nil, 4096)
	require.NoError(t, err)
	marker := EventBoundary{Order: 0, Instance: "guest", Incarnation: 1, Trace: 16, Decision: 7, TimeNanos: 100, InspectionOrder: 44}
	events := []OrderedEvent{
		{Order: 1, Kind: KindOutput, Instance: "guest", Incarnation: 1, Trace: 17, Decision: 8, TimeNanos: 101, Value: json.RawMessage(`"early"`)},
		{Order: 2, Kind: KindClock, Instance: "guest", Incarnation: 1, Trace: 17, Decision: 8, TimeNanos: 102, Value: json.RawMessage(`1`)},
	}
	log, err := ParseEventOrder(orderedLogFixture(t, events, &marker))
	require.NoError(t, err)
	expected := Run{Workload: "ordered-fixture", Seed: 7, Identities: map[string]string{"module": "m"}, Command: []string{"rerun"}, ArtifactDir: "/actual-artifact", BoundaryScoped: true,
		Incarnations: []Incarnation{{Instance: "guest", Incarnation: 1, Diagnostic: diagnostic.Bytes, Stdout: []byte("early")}},
		Boundaries:   []ReplayBoundary{{Instance: "guest", Incarnation: 1, Trace: 16, Decision: 7, At: Coordinate{Shared: 44, ChoiceOrdinal: 16, VirtualTime: 100}}}, OrderLog: &log}
	changed := slices.Clone(events)
	changed[0].Value = json.RawMessage(`"changed"`)
	changed[1].Value = json.RawMessage(`2`)
	other, err := ParseEventOrder(orderedLogFixture(t, changed, &marker))
	require.NoError(t, err)
	actual := expected
	actual.OrderLog = &other
	actual.Incarnations = slices.Clone(expected.Incarnations)
	actual.Incarnations[0].Stdout = []byte("changed")
	report, err := Compare(expected, actual)
	require.NoError(t, err)
	require.Equal(t, "undiagnosable", report.Status)
	require.Equal(t, "helper-event", report.Ordering)
	require.Equal(t, KindOutput, report.First.Kind)
	require.Equal(t, uint64(1), report.First.At.Shared)
	require.Equal(t, int64(101), report.First.At.VirtualTime)
	require.Contains(t, report.Reproducer.PrefixUnavailable, "missing retained inspection evidence")
	actual.OrderLog = &EventOrder{Schema: other.Schema, Complete: true, Events: other.Events}
	report, err = Compare(expected, actual)
	require.NoError(t, err)
	require.Equal(t, "undiagnosable", report.Status)
	require.Contains(t, report.Reproducer.PrefixUnavailable, "both inspection boundaries")

	actual.OrderLog = expected.OrderLog
	report, err = Compare(expected, actual)
	require.NoError(t, err)
	require.Equal(t, "undiagnosable", report.Status)
	require.Contains(t, report.Reason, "canonical evidence differs")
	partial := *expected.OrderLog
	partial.Complete = false
	actual.OrderLog = &partial
	report, err = Compare(expected, actual)
	require.NoError(t, err)
	require.Equal(t, "undiagnosable", report.Status, fmt.Sprint(report))
}
