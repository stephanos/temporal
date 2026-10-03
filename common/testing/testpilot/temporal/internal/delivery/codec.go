package delivery

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"go.temporal.io/server/common/testing/testpilot"
)

const routeVersion = 1

var (
	ErrRouteMissing   = errors.New("reservation route is missing")
	ErrRouteMalformed = errors.New("reservation route is malformed")
	ErrRouteOversized = errors.New("reservation route exceeds its byte limit")
	ErrRouteVersion   = errors.New("reservation route version is unsupported")
	ErrRouteCrossed   = errors.New("reservation route belongs to another delivery context")
)

type routeKind string

const (
	workflowRoute routeKind = "workflow"
	nexusRoute    routeKind = "nexus"
	activityRoute routeKind = "activity"
	// scheduledActivityRoute is the route of an activity a workflow scheduled: its workflow
	// binding and its schedule command name it, since the SDK may name the activity itself.
	scheduledActivityRoute routeKind = "scheduled_activity"
)

// WorkflowBinding is the physical Temporal workflow a reservation carrier starts. It is the route
// key, and its JSON tags are the route's wire format.
type WorkflowBinding struct {
	Namespace    string `json:"namespace"`
	WorkflowID   string `json:"workflow_id"`
	WorkflowType string `json:"workflow_type"`
	TaskQueue    string `json:"task_queue"`
}

// ActivityBinding is the physical standalone activity a reservation carrier starts. It is the route
// key of an activity route, and its JSON tags are the route's wire format.
type ActivityBinding struct {
	Namespace    string `json:"namespace"`
	ActivityID   string `json:"activity_id"`
	ActivityType string `json:"activity_type"`
	TaskQueue    string `json:"task_queue"`
}

// A route names exactly one started entity: a workflow, Nexus or scheduled-activity route its
// workflow binding, an activity route its activity binding. The other binding is absent from the
// wire.
type route struct {
	Version             int                           `json:"version"`
	Kind                routeKind                     `json:"kind"`
	SessionID           string                        `json:"session_id"`
	RunID               string                        `json:"run_id"`
	Origin              testpilot.Coordinate          `json:"origin"`
	Reservation         testpilot.ReservationIdentity `json:"reservation"`
	Binding             WorkflowBinding               `json:"binding,omitzero"`
	WorkflowReservation string                        `json:"workflow_reservation,omitempty"`
	WorkflowEntrypoint  string                        `json:"workflow_entrypoint,omitempty"`
	WorkflowOrdinal     int64                         `json:"workflow_ordinal"`
	WorkflowRunID       string                        `json:"workflow_run_id,omitempty"`
	SourceInstructionID string                        `json:"source_instruction_id,omitempty"`
	Activity            ActivityBinding               `json:"activity,omitzero"`
}

type routeCodec struct{ maximumBytes int }

func (c routeCodec) encode(value route) ([]byte, error) {
	if value.Version != routeVersion {
		return nil, ErrRouteVersion
	}
	if !validRoute(value) {
		return nil, ErrRouteMalformed
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, ErrRouteMalformed
	}
	if c.maximumBytes <= 0 || len(encoded) > c.maximumBytes {
		return nil, ErrRouteOversized
	}
	return encoded, nil
}

func (c routeCodec) decode(encoded []byte, kind routeKind) (route, error) {
	if len(encoded) == 0 {
		return route{}, ErrRouteMissing
	}
	if c.maximumBytes <= 0 || len(encoded) > c.maximumBytes {
		return route{}, ErrRouteOversized
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var value route
	if err := decoder.Decode(&value); err != nil {
		return route{}, ErrRouteMalformed
	}
	if err := requireEOF(decoder); err != nil {
		return route{}, ErrRouteMalformed
	}
	if value.Version != routeVersion {
		return route{}, ErrRouteVersion
	}
	if value.Kind != kind {
		return route{}, ErrRouteCrossed
	}
	if !validRoute(value) {
		return route{}, ErrRouteMalformed
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return route{}, ErrRouteMalformed
	}
	return value, nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrRouteMalformed
	}
	return nil
}

func validRoute(value route) bool {
	if !validRouteText(value.SessionID) || !validRouteText(value.RunID) || value.Origin.RunID != value.RunID || !validCoordinate(value.Origin) || value.Reservation.Origin != value.Origin || !validReservation(value.Reservation) {
		return false
	}
	unsourced := value.WorkflowReservation == "" && value.WorkflowEntrypoint == "" && value.WorkflowOrdinal == 0 && value.WorkflowRunID == "" && value.SourceInstructionID == ""
	switch value.Kind {
	case workflowRoute:
		return validBinding(value.Binding) && value.Activity == ActivityBinding{} && unsourced
	case nexusRoute, scheduledActivityRoute:
		return validBinding(value.Binding) && value.Activity == ActivityBinding{} && validRouteText(value.WorkflowReservation) && validRouteText(value.WorkflowEntrypoint) && value.WorkflowOrdinal >= 0 && validRouteText(value.WorkflowRunID) && validRouteText(value.SourceInstructionID)
	case activityRoute:
		return value.Binding == WorkflowBinding{} && validActivityBinding(value.Activity) && unsourced
	default:
		return false
	}
}

func validCoordinate(value testpilot.Coordinate) bool {
	return validRouteText(value.RunID) && validRouteText(value.EntrypointID) && validRouteText(value.ActivationID) && validRouteText(value.InstructionID) && value.Attempt > 0
}

func validReservation(value testpilot.ReservationIdentity) bool {
	return validRouteText(value.EntrypointID) && validRouteText(value.ID) && value.Ordinal >= 0
}

func validBinding(value WorkflowBinding) bool {
	return validRouteText(value.Namespace) && validRouteText(value.WorkflowID) && validRouteText(value.WorkflowType) && validRouteText(value.TaskQueue)
}

func validActivityBinding(value ActivityBinding) bool {
	return validRouteText(value.Namespace) && validRouteText(value.ActivityID) && validRouteText(value.ActivityType) && validRouteText(value.TaskQueue)
}

func validRouteText(value string) bool { return len(value) > 0 && len(value) <= 256 }
