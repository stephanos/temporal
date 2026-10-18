package measure

import (
	"context"
	"errors"
	"io"
	"os"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func closeResource(ctx context.Context, resource io.Closer) {
	err := resource.Close()
	if !errors.Is(err, os.ErrClosed) {
		recordFailure(ctx, err)
	}
}

func invalidMeasurement(dimension string) error {
	return &wasi.BoundaryError{Kind: "invalid", Dimension: dimension}
}

func terminalExitObservationFor(ctx context.Context, request wasi.Request, mismatch bool) wasi.TerminalExitObservation {
	observation, err := wasi.MeasureTerminalExit(ctx, request, mismatch)
	if err != nil && observation.Error == "" {
		observation.Error = err.Error()
	}
	return observation
}

func runtimeTerminalBytes(result wasi.Result) []byte {
	terminal := result.RuntimeTerminal()
	return terminal[:]
}

func setRuntimeTerminalBytes(result *wasi.Result, data []byte) int {
	terminal := result.RuntimeTerminal()
	count := copy(terminal[:], data)
	result.SetRuntimeTerminal(terminal)
	return count
}
