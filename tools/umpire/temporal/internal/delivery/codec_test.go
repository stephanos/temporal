package delivery

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire"
)

func TestRouteCodecRoundTripIsExactAndCanonical(t *testing.T) {
	want := route{
		Version:   routeVersion,
		Kind:      workflowRoute,
		SessionID: "session-one",
		RunID:     "run-one",
		Origin: umpire.Coordinate{
			RunID:         "run-one",
			EntrypointID:  "controller",
			ActivationID:  "controller.0",
			InstructionID: "start-workflow",
			Attempt:       1,
		},
		Reservation: umpire.ReservationIdentity{
			Origin: umpire.Coordinate{
				RunID:         "run-one",
				EntrypointID:  "controller",
				ActivationID:  "controller.0",
				InstructionID: "start-workflow",
				Attempt:       1,
			},
			EntrypointID: "workflow",
			Ordinal:      0,
			ID:           "workflow-reservation",
		},
		Binding: binding{
			Namespace:    "namespace",
			WorkflowID:   "workflow-id",
			WorkflowType: "workflow-type",
			TaskQueue:    "task-queue",
		},
	}
	codec := routeCodec{maximumBytes: 2048}

	encoded, err := codec.encode(want)
	require.NoError(t, err)
	got, err := codec.decode(encoded, workflowRoute)
	require.NoError(t, err)
	require.Equal(t, want, got)
	reencoded, err := codec.encode(got)
	require.NoError(t, err)
	require.True(t, bytes.Equal(encoded, reencoded))
}

func TestRouteCodecRejectsInvalidInput(t *testing.T) {
	codec := routeCodec{maximumBytes: 2048}
	valid := route{
		Version:   routeVersion,
		Kind:      nexusRoute,
		SessionID: "session",
		RunID:     "run",
		Origin:    umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
		Reservation: umpire.ReservationIdentity{
			Origin:       umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
			EntrypointID: "handler",
			ID:           "handler-reservation",
		},
		Binding:             binding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"},
		WorkflowReservation: "workflow-reservation",
		WorkflowEntrypoint:  "workflow",
		WorkflowOrdinal:     2,
		WorkflowRunID:       "temporal-run",
		SourceInstructionID: "start-nexus",
	}
	encoded, err := codec.encode(valid)
	require.NoError(t, err)

	tests := map[string]struct {
		data []byte
		kind routeKind
		err  error
	}{
		"missing":          {kind: nexusRoute, err: ErrRouteMissing},
		"malformed":        {data: []byte("{"), kind: nexusRoute, err: ErrRouteMalformed},
		"unknown version":  {data: bytes.Replace(encoded, []byte(`"version":1`), []byte(`"version":2`), 1), kind: nexusRoute, err: ErrRouteVersion},
		"unknown field":    {data: append(append([]byte(nil), encoded[:len(encoded)-1]...), []byte(`,"extra":true}`)...), kind: nexusRoute, err: ErrRouteMalformed},
		"noncanonical":     {data: append([]byte(" "), encoded...), kind: nexusRoute, err: ErrRouteMalformed},
		"crossed kind":     {data: encoded, kind: workflowRoute, err: ErrRouteCrossed},
		"missing ordinal":  {data: bytes.Replace(encoded, []byte(`"workflow_ordinal":2,`), nil, 1), kind: nexusRoute, err: ErrRouteMalformed},
		"missing identity": {data: bytes.Replace(encoded, []byte("handler-reservation"), nil, 1), kind: nexusRoute, err: ErrRouteMalformed},
		"oversized":        {data: bytes.Repeat([]byte("x"), 2049), kind: nexusRoute, err: ErrRouteOversized},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := codec.decode(test.data, test.kind)
			require.ErrorIs(t, err, test.err)
		})
	}
}
