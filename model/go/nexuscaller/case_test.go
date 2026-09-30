package nexuscaller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/model/go/caseproducer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// persisted is the fixture byte form: the canonical compact JSON re-indented with two spaces and
// one trailing newline (tools/umpire/internal/casefile.Persisted, which this package may not
// import).
func persisted(t *testing.T, encoded []byte) []byte {
	t.Helper()
	var compact, indented bytes.Buffer
	require.NoError(t, json.Compact(&compact, encoded))
	require.NoError(t, json.Indent(&indented, compact.Bytes(), "", "  "))
	return append(indented.Bytes(), '\n')
}

// firstDifference walks two decoded JSON documents and names the first path where they differ.
func firstDifference(path string, want, got any) string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: lean %T, go %T", path, want, got)
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		slices.Sort(sorted)
		for _, k := range sorted {
			if d := firstDifference(path+"."+k, w[k], g[k]); d != "" {
				return d
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			return fmt.Sprintf("%s: lean %T, go %T", path, want, got)
		}
		for i := range min(len(w), len(g)) {
			if d := firstDifference(fmt.Sprintf("%s[%d]", path, i), w[i], g[i]); d != "" {
				return d
			}
		}
		if len(w) != len(g) {
			return fmt.Sprintf("%s: lean has %d elements, go %d", path, len(w), len(g))
		}
	default:
		if !reflect.DeepEqual(want, got) {
			return fmt.Sprintf("%s: lean %v, go %v", path, want, got)
		}
	}
	return ""
}

func TestCasesAreByteIdenticalToTheFixtures(t *testing.T) {
	realization := AsyncNexus("umpire.case.service", "complete")
	for _, q := range FunctionalQueries {
		t.Run(q.Name, func(t *testing.T) {
			fixture := "nexusCallerTests-" + q.Name
			produced, err := cp.Produce(q, cp.IdentityFor("temporal.case", "nexusCallerTests", q.Name), realization, ModelSource)
			require.NoError(t, err)
			encoded, err := protojson.Marshal(produced)
			require.NoError(t, err)
			got := persisted(t, encoded)
			want, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "testcore", "testpilot", "testdata", fixture+"-case.json"))
			require.NoError(t, err)
			if bytes.Equal(want, got) {
				return
			}
			var w, g any
			require.NoError(t, json.Unmarshal(want, &w))
			require.NoError(t, json.Unmarshal(got, &g))
			if d := firstDifference("$", w, g); d != "" {
				t.Fatalf("first difference: %s", d)
			}
			t.Fatalf("same JSON value, different bytes (key order or escaping); first byte %d", firstByte(want, got))
		})
	}
}

func firstByte(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

// Every produced Case decodes strictly and prepares, unchanged, under the Profile derived from it:
// what a black-box consumer does with a checked-in fixture. The environment names the handler's own
// task queue, which the realization binds apart from the caller workflow's.
func TestProducedCasesPrepareUnderTheirDerivedProfile(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	realization := AsyncNexus("umpire.case.service", "complete")
	for _, q := range FunctionalQueries {
		t.Run(q.Name, func(t *testing.T) {
			produced, err := cp.Produce(q, cp.IdentityFor("temporal.case", "nexusCallerTests", q.Name), realization, ModelSource)
			require.NoError(t, err)
			encoded, err := protojson.Marshal(produced)
			require.NoError(t, err)
			source, err := testpilot.DecodeCaseProtoJSON(encoded)
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{
				Identity: q.Name + "-profile", Namespace: "namespace", TaskQueue: "task-queue",
				HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint"})
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(source, profile)
			require.NoError(t, err)
			require.True(t, proto.Equal(source, prepared.Snapshot()), "preparation carries the Case unchanged")
		})
	}
}
