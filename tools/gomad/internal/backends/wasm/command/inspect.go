package command

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
)

// clusterStreamTail bounds the stream tail an inspection shows per incarnation.
const clusterStreamTail = 256

// RetainedClusterInspection projects the retained cluster evidence of a
// runner-failure artifact (a cluster capacity abort, fn-235): the generic
// record omits each incarnation's partial streams, the abort virtual time and
// the exhausted limit. Other artifacts return nil. It authenticates the
// evidence like replay preflight and never resolves an installation.
func RetainedClusterInspection(path string) (*inspect.ClusterInspection, error) {
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		return nil, err
	}
	manifest := opened.Manifest()
	if manifest.ArtifactKind != record.ArtifactRunnerFailure || manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || !simulation.IsRetainedSchema(manifest.Target.Backend.Evidence.Schema) {
		return nil, opened.Close()
	}
	reference := *manifest.Target.Backend.Evidence
	data, err := opened.ReadPayload(reference.File, 512<<20)
	if closeErr := opened.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if err := simulation.CheckRetainedSchema(reference.Schema); err != nil {
		return nil, err
	}
	_, observed, err := simulation.VerifyRetained(target.BackendPayload{Reference: reference, Data: data})
	if err != nil {
		return nil, fmt.Errorf("retained WASM cluster evidence: %w", err)
	}
	return projectCluster(reference.Schema, observed), nil
}

func projectCluster(schema string, observed gomadsim.ClusterRecord) *inspect.ClusterInspection {
	inspected := &inspect.ClusterInspection{EvidenceSchema: schema, Outcome: string(observed.Outcome), Reason: observed.Reason, Instances: []inspect.ClusterInstanceInspection{}}
	if observed.WASM == nil {
		return inspected
	}
	inspected.VirtualElapsedNanos = observed.WASM.VirtualElapsedNanos
	for _, incarnation := range observed.WASM.Incarnations {
		instance := inspect.ClusterInstanceInspection{Node: string(incarnation.Handle.Node), Incarnation: incarnation.Handle.Incarnation, Interrupted: incarnation.Interrupted}
		for _, node := range observed.Nodes {
			if node.Handle == incarnation.Handle {
				instance.State, instance.Reason = string(node.State), node.Reason
			}
		}
		var stdout, stderr []byte
		for _, output := range observed.Outputs {
			if output.Handle != incarnation.Handle {
				continue
			}
			if output.Stream == gomadsim.OutputStdout {
				stdout = append(stdout, output.Bytes...)
				instance.Stdout.Bytes += output.TotalBytes
			} else {
				stderr = append(stderr, output.Bytes...)
				instance.Stderr.Bytes += output.TotalBytes
			}
		}
		instance.Stdout.Tail, instance.Stderr.Tail = streamTail(stdout), streamTail(stderr)
		if limit := exhaustedLimit(instance.Reason); limit != nil && inspected.Exhausted == nil {
			inspected.Exhausted = limit
		}
		inspected.Instances = append(inspected.Instances, instance)
	}
	slices.SortFunc(inspected.Instances, func(a, b inspect.ClusterInstanceInspection) int {
		if a.Node != b.Node {
			return strings.Compare(a.Node, b.Node)
		}
		return cmp.Compare(a.Incarnation, b.Incarnation)
	})
	return inspected
}

func streamTail(data []byte) string {
	if len(data) > clusterStreamTail {
		data = data[len(data)-clusterStreamTail:]
	}
	return strings.ToValidUTF8(string(data), "�")
}

// exhaustedLimit reads the helper's `WASI capacity: <dimension> limit=<n>
// used=<n> delta=<n>` terminal reason.
func exhaustedLimit(reason string) *inspect.ClusterLimitInspection {
	detail, found := strings.CutPrefix(reason, "WASI capacity: ")
	if !found {
		return nil
	}
	var limit inspect.ClusterLimitInspection
	if _, err := fmt.Sscanf(detail, "%s limit=%d used=%d delta=%d", &limit.Dimension, &limit.Limit, &limit.Used, &limit.Delta); err != nil {
		return nil
	}
	return &limit
}
