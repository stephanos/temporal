package executorhttp

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func FuzzHandlerWireSurfaceFailsClosed(f *testing.F) {
	valid, err := deterministicMarshal.Marshal(testRequest())
	require.NoError(f, err)
	f.Add([]byte{})
	f.Add([]byte{0x80})
	f.Add(valid)

	f.Fuzz(func(t *testing.T, body []byte) {
		var calls atomic.Int32
		handler := newHandler(func(context.Context, *umpirespb.ExecuteRequest) (*umpirespb.ExecuteResponse, error) {
			calls.Add(1)
			return toolingResponse(umpirespb.TOOLING_STATUS_INVALID_CONTRACT), nil
		}, 1<<10, 1<<20, time.Second)

		recorder := serve(handler, http.MethodPost, ExecutePath, ProtobufContentType, body)
		if recorder.Code != http.StatusOK {
			require.Empty(t, recorder.Body.Bytes())
			require.Zero(t, calls.Load())
			return
		}

		response := decodeResponse(t, recorder)
		require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_CONTRACT, response.GetResult().GetToolingStatus())
		require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, response.GetResult().GetDecision())
		require.Equal(t, int32(1), calls.Load())
	})
}
