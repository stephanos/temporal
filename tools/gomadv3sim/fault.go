package gomadv3sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

const FaultPlanSchema = "gomadv3.fault-plan/v1"
const MaximumFaultActions uint64 = 4096
const MaximumFaultPlanBytes = 16 << 20

type FaultID string
type FaultKind string
type FaultPersistence string

const (
	FaultGracefulStop FaultKind = "graceful_stop"
	FaultHarshCrash   FaultKind = "harsh_crash"
	FaultRestart      FaultKind = "restart"
	FaultDisconnect   FaultKind = "disconnect"
	FaultReconnect    FaultKind = "reconnect"
	FaultPartition    FaultKind = "partition"
	FaultHeal         FaultKind = "heal"
	FaultDelay        FaultKind = "delay"
)

const (
	FaultPersistencePersisted FaultPersistence = "persisted_only"
	FaultPersistencePartial   FaultPersistence = "selected_partial"
)

type FaultMatch struct {
	Node        NodeID `json:"node,omitempty"`
	Incarnation uint64 `json:"incarnation,omitempty"`
	NodeClass   string `json:"node_class,omitempty"`
	Model       string `json:"model,omitempty"`
	Resource    string `json:"resource,omitempty"`
	Operation   string `json:"operation,omitempty"`
	Occurrence  uint64 `json:"occurrence,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Equivalence string `json:"equivalence,omitempty"`
}

type FaultAction struct {
	ID          FaultID          `json:"id"`
	Kind        FaultKind        `json:"kind"`
	Match       FaultMatch       `json:"match,omitempty"`
	Node        NodeID           `json:"node,omitempty"`
	Candidates  []NodeID         `json:"candidates,omitempty"`
	TargetFrom  FaultID          `json:"target_from,omitempty"`
	From        NodeID           `json:"from,omitempty"`
	To          NodeID           `json:"to,omitempty"`
	Left        []NodeID         `json:"left,omitempty"`
	Right       []NodeID         `json:"right,omitempty"`
	DelayNanos  uint64           `json:"delay_nanos,omitempty"`
	Persistence FaultPersistence `json:"persistence,omitempty"`
}

type FaultPlan struct {
	Schema   string        `json:"schema"`
	Actions  []FaultAction `json:"actions"`
	Identity string        `json:"identity"`
}

func NewFaultPlan(actions []FaultAction) (FaultPlan, error) {
	if err := checkCapacity("fault_actions", uint64(len(actions)), MaximumFaultActions); err != nil {
		return FaultPlan{}, err
	}
	plan := FaultPlan{Schema: FaultPlanSchema, Actions: cloneFaultActions(actions)}
	if err := validateFaultActions(plan.Actions); err != nil {
		return FaultPlan{}, err
	}
	identity, err := faultPlanIdentity(plan)
	if err != nil {
		return FaultPlan{}, err
	}
	plan.Identity = identity
	return plan, nil
}

func EncodeFaultPlan(plan FaultPlan) ([]byte, error) {
	if err := validateFaultPlan(plan); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("encode fault plan: %w", err)
	}
	if len(encoded) > MaximumFaultPlanBytes {
		return nil, &CapacityError{Resource: "fault_plan_bytes", Required: uint64(len(encoded)), Maximum: MaximumFaultPlanBytes}
	}
	return encoded, nil
}

func DecodeFaultPlan(data []byte) (FaultPlan, error) {
	if len(data) == 0 || len(data) > MaximumFaultPlanBytes {
		return FaultPlan{}, fmt.Errorf("fault plan must be between 1 and %d bytes", MaximumFaultPlanBytes)
	}
	if !utf8.Valid(data) {
		return FaultPlan{}, errors.New("fault plan is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan FaultPlan
	if err := decoder.Decode(&plan); err != nil {
		return FaultPlan{}, fmt.Errorf("decode fault plan: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return FaultPlan{}, errors.New("fault plan contains trailing JSON")
	}
	if err := validateFaultPlan(plan); err != nil {
		return FaultPlan{}, err
	}
	canonical, err := json.Marshal(plan)
	if err != nil {
		return FaultPlan{}, fmt.Errorf("canonicalize fault plan: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return FaultPlan{}, errors.New("fault plan is not canonical JSON")
	}
	return plan, nil
}

func validateFaultPlan(plan FaultPlan) error {
	if plan.Schema != FaultPlanSchema {
		return fmt.Errorf("fault plan schema = %q, want %q", plan.Schema, FaultPlanSchema)
	}
	if err := checkCapacity("fault_actions", uint64(len(plan.Actions)), MaximumFaultActions); err != nil {
		return err
	}
	if err := validateFaultActions(plan.Actions); err != nil {
		return err
	}
	if !validSHA256(plan.Identity) {
		return errors.New("fault plan identity is invalid")
	}
	want, err := faultPlanIdentity(plan)
	if err != nil {
		return err
	}
	if plan.Identity != want {
		return errors.New("fault plan identity does not match its contents")
	}
	return nil
}

func validateFaultActions(actions []FaultAction) error {
	seen := make(map[FaultID]struct{}, len(actions))
	for _, action := range actions {
		if err := validateFaultAction(action); err != nil {
			return err
		}
		if _, ok := seen[action.ID]; ok {
			return fmt.Errorf("fault action ID %q is duplicated", action.ID)
		}
		seen[action.ID] = struct{}{}
	}
	return nil
}

func validateFaultAction(action FaultAction) error {
	if err := validateID("fault action ID", string(action.ID)); err != nil {
		return err
	}
	if err := validateFaultMatch(action.Match); err != nil {
		return fmt.Errorf("fault %q: %w", action.ID, err)
	}
	if err := validateSortedNodeIDs("fault candidates", action.Candidates); err != nil {
		return fmt.Errorf("fault %q: %w", action.ID, err)
	}
	if err := validateSortedNodeIDs("fault left group", action.Left); err != nil {
		return fmt.Errorf("fault %q: %w", action.ID, err)
	}
	if err := validateSortedNodeIDs("fault right group", action.Right); err != nil {
		return fmt.Errorf("fault %q: %w", action.ID, err)
	}
	targetModes := 0
	if action.Node != "" {
		targetModes++
		if err := validateID("fault target node", string(action.Node)); err != nil {
			return err
		}
	}
	if len(action.Candidates) != 0 {
		targetModes++
	}
	if action.TargetFrom != "" {
		targetModes++
		if err := validateID("fault target reference", string(action.TargetFrom)); err != nil {
			return err
		}
	}
	if targetModes > 1 {
		return fmt.Errorf("fault %q mixes node target modes", action.ID)
	}
	switch action.Kind {
	case FaultGracefulStop, FaultRestart:
		if targetModes != 1 || action.From != "" || action.To != "" || len(action.Left) != 0 || len(action.Right) != 0 || action.DelayNanos != 0 || action.Persistence != "" {
			return fmt.Errorf("fault %q has an invalid lifecycle action shape", action.ID)
		}
	case FaultHarshCrash:
		if targetModes != 1 || action.TargetFrom != "" || action.From != "" || action.To != "" || len(action.Left) != 0 || len(action.Right) != 0 || action.DelayNanos != 0 || action.Persistence != FaultPersistencePersisted && action.Persistence != FaultPersistencePartial {
			return fmt.Errorf("fault %q has an invalid crash action shape", action.ID)
		}
	case FaultDisconnect, FaultReconnect:
		if targetModes != 0 || !validDirectionalPair(action.From, action.To) || len(action.Left) != 0 || len(action.Right) != 0 || action.DelayNanos != 0 || action.Persistence != "" {
			return fmt.Errorf("fault %q has an invalid directional action shape", action.ID)
		}
	case FaultDelay:
		if targetModes != 0 || !validDirectionalPair(action.From, action.To) || len(action.Left) != 0 || len(action.Right) != 0 || action.DelayNanos == 0 || action.Persistence != "" {
			return fmt.Errorf("fault %q has an invalid delay action shape", action.ID)
		}
	case FaultPartition, FaultHeal:
		if targetModes != 0 || action.From != "" || action.To != "" || len(action.Left) == 0 || len(action.Right) == 0 || action.DelayNanos != 0 || action.Persistence != "" || groupsOverlap(action.Left, action.Right) {
			return fmt.Errorf("fault %q has an invalid partition action shape", action.ID)
		}
	default:
		return fmt.Errorf("fault %q has invalid kind %q", action.ID, action.Kind)
	}
	return nil
}

func validateFaultMatch(match FaultMatch) error {
	if match.Incarnation != 0 && match.Node == "" {
		return errors.New("fault match incarnation has no node")
	}
	if match.Node != "" {
		if err := validateID("fault match node", string(match.Node)); err != nil {
			return err
		}
	}
	fields := []struct{ name, value string }{
		{"node class", match.NodeClass}, {"model", match.Model}, {"resource", match.Resource},
		{"operation", match.Operation}, {"phase", match.Phase}, {"equivalence", match.Equivalence},
	}
	for _, field := range fields {
		if field.value != "" {
			if err := validateID("fault match "+field.name, field.value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateFaultEvent(event FaultMatch) error {
	if event.Occurrence != 0 {
		return errors.New("fault event occurrence is controller-owned")
	}
	return validateFaultMatch(event)
}

func faultMatchEmpty(match FaultMatch) bool {
	return match == (FaultMatch{})
}

func faultMatches(pattern, event FaultMatch) bool {
	return (pattern.Node == "" || pattern.Node == event.Node) &&
		(pattern.Incarnation == 0 || pattern.Incarnation == event.Incarnation) &&
		(pattern.NodeClass == "" || pattern.NodeClass == event.NodeClass) &&
		(pattern.Model == "" || pattern.Model == event.Model) &&
		(pattern.Resource == "" || pattern.Resource == event.Resource) &&
		(pattern.Operation == "" || pattern.Operation == event.Operation) &&
		(pattern.Occurrence == 0 || pattern.Occurrence == event.Occurrence) &&
		(pattern.Phase == "" || pattern.Phase == event.Phase) &&
		(pattern.Equivalence == "" || pattern.Equivalence == event.Equivalence)
}

func validateSortedNodeIDs(name string, nodes []NodeID) error {
	for index, node := range nodes {
		if err := validateID(name, string(node)); err != nil {
			return err
		}
		if index != 0 && nodes[index-1] >= node {
			return fmt.Errorf("%s must be strictly sorted", name)
		}
	}
	return nil
}

func validDirectionalPair(from, to NodeID) bool {
	return from != "" && to != "" && from != to && validateID("source node", string(from)) == nil && validateID("destination node", string(to)) == nil
}

func groupsOverlap(left, right []NodeID) bool {
	for _, node := range left {
		if _, ok := slices.BinarySearch(right, node); ok {
			return true
		}
	}
	return false
}

func faultPlanIdentity(plan FaultPlan) (string, error) {
	plan.Identity = ""
	return hashCanonical("gomadv3-fault-plan/v1", plan)
}

func cloneFaultActions(actions []FaultAction) []FaultAction {
	cloned := make([]FaultAction, len(actions))
	for index, action := range actions {
		cloned[index] = cloneFaultAction(action)
	}
	return cloned
}

func cloneFaultAction(action FaultAction) FaultAction {
	action.Candidates = append([]NodeID(nil), action.Candidates...)
	action.Left = append([]NodeID(nil), action.Left...)
	action.Right = append([]NodeID(nil), action.Right...)
	return action
}
