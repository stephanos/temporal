package simulation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestMarkerBoundaryPreservesSelectedCapacityAndPriorErrors(t *testing.T) {
	handle := gomadsim.NodeHandle{Node: "a", Incarnation: 1}
	protocol := &wasi.BoundaryError{Kind: "invalid", Dimension: "malformed evidence"}
	divergence := &gomadsim.ReplayDivergenceError{}
	runtimeDivergence := &wasi.RuntimeReplayError{}
	for _, marker := range []string{"output-crash", "storage-crash", "output-stop"} {
		for _, test := range []struct {
			name       string
			result     wasi.Result
			prior      error
			classified bool
			unrelated  bool
		}{
			{name: "choice capacity", result: wasi.Result{Termination: "capacity", Message: "runtime-choice-bytes", Reaped: true}, classified: true},
			{name: "transcript capacity", result: wasi.Result{Termination: "capacity", Message: "transcript-bytes", Reaped: true}, classified: true},
			{name: "output capacity", result: wasi.Result{Termination: "capacity", Message: "output capacity", Reaped: true}, classified: true},
			{name: "unsupported CPU clock", result: wasi.Result{Termination: "unsupported", Message: "CPU clocks outside stock WASI profile", Reaped: true}, classified: true},
			{name: "selected divergence", result: wasi.Result{Termination: "divergence", Message: "choice replay diverged", Reaped: true}, classified: true},
			{name: "selected invalid", result: wasi.Result{Termination: "invalid", Message: "invalid local evidence", Reaped: true}, classified: true},
			{name: "selected infrastructure", result: wasi.Result{Termination: "infrastructure", Message: "helper transport", Reaped: true}, classified: true},
			{name: "protocol precedence", result: wasi.Result{Termination: "capacity", Reaped: true}, prior: protocol},
			{name: "divergence precedence", result: wasi.Result{Termination: "capacity", Reaped: true}, prior: divergence},
			{name: "unsupported protocol precedence", result: wasi.Result{Termination: "unsupported", Reaped: true}, prior: protocol},
			{name: "unsupported divergence precedence", result: wasi.Result{Termination: "unsupported", Reaped: true}, prior: divergence},
			{name: "runtime replay precedence", result: wasi.Result{Termination: "unsupported", Reaped: true}, prior: runtimeDivergence},
			{name: "controller precedence", result: wasi.Result{Termination: "unsupported", Reaped: true}, prior: gomadsim.ErrInvalidTransition},
			{name: "unreaped capacity", result: wasi.Result{Termination: "capacity"}},
			{name: "unreaped unsupported", result: wasi.Result{Termination: "unsupported"}},
			{name: "trap before marker", result: wasi.Result{Termination: "trap", Reaped: true}},
			{name: "unknown terminal", result: wasi.Result{Termination: "unknown", Reaped: true}},
			{name: "missing marker exit", result: wasi.Result{Termination: "exit", Reaped: true}},
			{name: "unrelated capacity", result: wasi.Result{Termination: "exit", Reaped: true}, unrelated: true},
		} {
			t.Run(marker+"/"+test.name, func(t *testing.T) {
				done := make(chan struct{})
				close(done)
				n := &node{handle: handle, done: done, result: test.result, err: test.prior}
				e := &execution{nodes: map[gomadsim.NodeID]*node{handle.Node: n}, results: map[gomadsim.NodeHandle]wasi.Result{handle: test.result}}
				if test.unrelated {
					e.results[gomadsim.NodeHandle{Node: "b", Incarnation: 1}] = wasi.Result{Termination: "capacity", Reaped: true}
				}
				switch marker {
				case "output-crash":
					e.provider.OutputCrashes = []OutputCrash{{Handle: handle, Marker: []byte("marker")}}
				case "storage-crash":
					n.request.Config.Storage = &wasi.StorageModel{Faults: []wasi.StorageFault{{Action: "crash-before"}}}
				case "output-stop":
					e.provider.OutputStops = []OutputCrash{{Handle: handle, Marker: []byte("marker")}}
				}
				var err error
				if marker == "output-stop" {
					_, err = e.Stop(t.Context(), handle)
				} else {
					_, err = e.Crash(t.Context(), handle)
				}
				if test.prior != nil {
					require.ErrorIs(t, err, test.prior)
					return
				}
				var boundary *wasi.BoundaryError
				if test.classified {
					require.ErrorAs(t, err, &boundary)
					require.Equal(t, test.result.Termination, boundary.Kind)
					require.Equal(t, test.result.Message, boundary.Dimension)
				} else {
					require.ErrorContains(t, err, "boundary was not reached")
					require.NotErrorAs(t, err, &boundary)
				}
			})
		}
	}
}
