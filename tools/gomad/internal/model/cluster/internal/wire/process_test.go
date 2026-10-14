package wire

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessFrameRejectsMalformedValuesAndInvalidSemantics(t *testing.T) {
	for _, encoded := range []string{
		"", "{", `{"request":-1}`, `{"request":1} {}`, `{"request":1} trailing`, `{"request":1,"unknown":true}`,
	} {
		var frame ProcessFrame
		require.Error(t, DecodeProcessValue([]byte(encoded), &frame), "value %q", encoded)
	}
	for _, frame := range []ProcessFrame{
		{Profile: "unsupported", Kind: ProcessFrameStart, Request: 1},
		{Profile: ProcessProtocol, Kind: "unknown", Request: 1},
		{Profile: ProcessProtocol, Kind: ProcessFrameStart},
		{Profile: ProcessProtocol, Kind: ProcessFrameResponse, Request: 1, Arrivals: 1},
		{Profile: ProcessProtocol, Kind: ProcessFrameStart, Request: 1, Node: strings.Repeat("n", 257)},
		{Profile: ProcessProtocol, Kind: ProcessFrameTerminal, Request: 1, Error: strings.Repeat("e", 4096+1)},
	} {
		require.Error(t, ValidateProcessFrame(frame))
	}
}

func TestProcessFrameRoundTripAndStrictDecode(t *testing.T) {
	want := ProcessFrame{Profile: ProcessProtocol, Kind: ProcessFrameStart, Request: 7, Node: "history", Incarnation: 2, Payload: []byte("bootstrap")}
	encoded, err := EncodeProcessValue(want)
	require.NoError(t, err)
	var got ProcessFrame
	require.NoError(t, DecodeProcessValue(encoded, &got))
	require.Equal(t, want, got)
	require.NoError(t, ValidateProcessFrame(got))
	require.Error(t, DecodeProcessValue([]byte(`{"kind":"start","request":7,"unknown":true}`), &got))
}

func TestProcessActivationFramesAreValid(t *testing.T) {
	for _, kind := range []ProcessFrameKind{ProcessFrameActivate, ProcessFrameActivated} {
		require.NoError(t, ValidateProcessFrame(ProcessFrame{Profile: ProcessProtocol, Kind: kind, Request: 1, Node: "history", Incarnation: 2}))
	}
}

func TestProcessExplorationFramesAreValid(t *testing.T) {
	require.NoError(t, ValidateProcessFrame(ProcessFrame{Profile: ProcessProtocol, Kind: ProcessFrameExplorationPlan, Request: 1}))
	require.NoError(t, ValidateProcessFrame(ProcessFrame{Profile: ProcessProtocol, Kind: ProcessFrameExplorationRecord, Request: 2, Payload: []byte("record")}))
}
