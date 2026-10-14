package delivery

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
)

func TestRouteCodecRoundTripIsExactAndCanonical(t *testing.T) {
	want := route{
		Version:   routeVersion,
		Kind:      workflowRoute,
		SessionID: "session-one",
		RunID:     "run-one",
		Origin: testpilot.Coordinate{
			RunID:         "run-one",
			EntrypointID:  "controller",
			ActivationID:  "controller.0",
			InstructionID: "start-workflow",
			Attempt:       1,
		},
		Reservation: testpilot.ReservationIdentity{
			Origin: testpilot.Coordinate{
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

// The route rides in a reservation carrier between a Session and the worker that decodes it, so
// its wire bytes are pinned for both kinds: a change to the binding or route shape shows here.
func TestRouteCodecWireBytesGolden(t *testing.T) {
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}
	workflow := route{
		Version:     routeVersion,
		Kind:        workflowRoute,
		SessionID:   "session",
		RunID:       "run",
		Origin:      origin,
		Reservation: testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "workflow-reservation"},
		Binding:     binding{Namespace: "namespace", WorkflowID: "workflow-id", WorkflowType: "workflow-type", TaskQueue: "task-queue"},
	}
	nexus := workflow
	nexus.Kind = nexusRoute
	nexus.Reservation = testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", Ordinal: 1, ID: "handler-reservation"}
	nexus.WorkflowReservation = "workflow-reservation"
	nexus.WorkflowEntrypoint = "workflow"
	nexus.WorkflowOrdinal = 2
	nexus.WorkflowRunID = "temporal-run"
	nexus.SourceInstructionID = "start-nexus"
	codec := routeCodec{maximumBytes: 2048}

	for name, test := range map[string]struct {
		value route
		want  string
	}{
		"workflow": {value: workflow, want: `{"version":1,"kind":"workflow","session_id":"session","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"EntrypointID":"workflow","Ordinal":0,"ID":"workflow-reservation"},"binding":{"namespace":"namespace","workflow_id":"workflow-id","workflow_type":"workflow-type","task_queue":"task-queue"},"workflow_ordinal":0}`},
		"nexus":    {value: nexus, want: `{"version":1,"kind":"nexus","session_id":"session","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller.0","InstructionID":"start","Attempt":1},"EntrypointID":"handler","Ordinal":1,"ID":"handler-reservation"},"binding":{"namespace":"namespace","workflow_id":"workflow-id","workflow_type":"workflow-type","task_queue":"task-queue"},"workflow_reservation":"workflow-reservation","workflow_entrypoint":"workflow","workflow_ordinal":2,"workflow_run_id":"temporal-run","source_instruction_id":"start-nexus"}`},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := codec.encode(test.value)
			require.NoError(t, err)
			require.Equal(t, test.want, string(encoded))
			decoded, err := codec.decode([]byte(test.want), test.value.Kind)
			require.NoError(t, err)
			require.Equal(t, test.value, decoded)
		})
	}
}

func TestRouteCodecRejectsInvalidInput(t *testing.T) {
	codec := routeCodec{maximumBytes: 2048}
	valid := route{
		Version:   routeVersion,
		Kind:      nexusRoute,
		SessionID: "session",
		RunID:     "run",
		Origin:    testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
		Reservation: testpilot.ReservationIdentity{
			Origin:       testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
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
