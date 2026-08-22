package checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol/internal/generated"
)

const TemporalViewFormatVersion = "umpire3/temporal-view/v1"

const (
	TemporalLassoReplayInputFormatVersion   = "umpire3/temporal-lasso-replay-input/v1"
	TemporalLassoReplayReceiptFormatVersion = "umpire3/temporal-lasso-replay-receipt/v1"
)

type TemporalTransition struct {
	Action    ActionKind `json:"action"`
	FromState string     `json:"fromState"`
	ToState   string     `json:"toState"`
}

type TemporalFairness struct {
	Identifier    string     `json:"identifier"`
	Kind          string     `json:"kind"`
	Action        ActionKind `json:"action"`
	EnabledStates []string   `json:"enabledStates"`
}

type TemporalProgress struct {
	Identifier    PropertyID `json:"identifier"`
	TriggerStates []string   `json:"triggerStates"`
	GoalStates    []string   `json:"goalStates"`
}

type TemporalBounds struct {
	MaxTraceLength int `json:"maxTraceLength"`
}

type TemporalProof struct {
	Theorem             string      `json:"theorem"`
	Statement           string      `json:"statement"`
	ResultClass         ResultClass `json:"resultClass"`
	TrustBadge          TrustBadge  `json:"trustBadge"`
	Axioms              []string    `json:"axioms"`
	FairnessAssumptions []string    `json:"fairnessAssumptions"`
}

type TemporalView struct {
	FormatVersion   string               `json:"formatVersion"`
	Target          TargetID             `json:"target"`
	Property        PropertyID           `json:"property"`
	World           string               `json:"world"`
	Variant         string               `json:"variant"`
	ClaimScope      string               `json:"claimScope"`
	SemanticHash    string               `json:"semanticHash"`
	CanonicalModel  string               `json:"canonicalModel"`
	Resources       []FirstOrderResource `json:"resources"`
	LiveOnlyActions []ActionKind         `json:"liveOnlyActions"`
	States          []string             `json:"states"`
	Initial         string               `json:"initial"`
	Actions         []ActionKind         `json:"actions"`
	Transitions     []TemporalTransition `json:"transitions"`
	Fairness        []TemporalFairness   `json:"fairness"`
	Progress        TemporalProgress     `json:"progress"`
	Bounds          TemporalBounds       `json:"bounds"`
	Relation        FirstOrderRelation   `json:"relation"`
	Proof           *TemporalProof       `json:"proof"`
}

type TemporalLasso struct {
	States    []string     `json:"states"`
	Actions   []ActionKind `json:"actions"`
	LoopStart int          `json:"loopStart"`
}

type TemporalLassoReplayInput struct {
	FormatVersion string        `json:"formatVersion"`
	Target        TargetID      `json:"target"`
	Property      PropertyID    `json:"property"`
	World         string        `json:"world"`
	Variant       string        `json:"variant"`
	SemanticHash  string        `json:"semanticHash"`
	Lasso         TemporalLasso `json:"lasso"`
}

type TemporalLassoReplayReceipt struct {
	FormatVersion string            `json:"formatVersion"`
	LassoDigest   string            `json:"lassoDigest"`
	Target        TargetID          `json:"target"`
	Property      PropertyID        `json:"property"`
	World         string            `json:"world"`
	Variant       string            `json:"variant"`
	SemanticHash  string            `json:"semanticHash"`
	Lasso         TemporalLasso     `json:"lasso"`
	Status        TraceReplayStatus `json:"status"`
	TrustBadge    TrustBadge        `json:"trustBadge"`
	Axioms        []string          `json:"axioms"`
}

var defaultTaskDeliveryTemporalJSON = generated.Read(generated.TaskDeliveryTemporal)
var defaultTaskDeliveryMutatedTemporalJSON = generated.Read(generated.TaskDeliveryMutatedTemporal)

func DecodeTemporalView(encoded []byte) (TemporalView, error) {
	var view TemporalView
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "temporal view", &view); err != nil {
		return TemporalView{}, err
	}
	if err := view.Validate(); err != nil {
		return TemporalView{}, err
	}
	return view, nil
}

func DefaultTemporalView(variant string) (TemporalView, bool, error) {
	var encoded []byte
	switch variant {
	case "sound":
		encoded = defaultTaskDeliveryTemporalJSON
	case "delivery-fairness-removed":
		encoded = defaultTaskDeliveryMutatedTemporalJSON
	default:
		return TemporalView{}, false, nil
	}
	view, err := DecodeTemporalView(encoded)
	return view, true, err
}

func DecodeTemporalLassoReplayReceipt(encoded []byte) (TemporalLassoReplayReceipt, error) {
	var receipt TemporalLassoReplayReceipt
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit,
		"temporal lasso replay receipt", &receipt); err != nil {
		return TemporalLassoReplayReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return TemporalLassoReplayReceipt{}, err
	}
	return receipt, nil
}

func (v TemporalView) Validate() error {
	if v.FormatVersion != TemporalViewFormatVersion || v.Target == "" || v.Property == "" ||
		v.World == "" || v.Variant == "" || v.ClaimScope != "canonical-model-only" ||
		!validHash(v.SemanticHash) || v.CanonicalModel == "" ||
		len(v.Resources) == 0 || v.LiveOnlyActions == nil || len(v.States) == 0 || v.Initial == "" || len(v.Actions) == 0 || len(v.Transitions) == 0 ||
		v.Fairness == nil || v.Progress.Identifier != v.Property || len(v.Progress.TriggerStates) == 0 ||
		len(v.Progress.GoalStates) == 0 || v.Bounds.MaxTraceLength <= 0 {
		return errors.New("complete temporal view identity, semantics, progress, fairness, and bounds are required")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	if !catalogTargetProperty(catalog, v.Target, v.Property) {
		return fmt.Errorf("unknown temporal target/property %q/%q", v.Target, v.Property)
	}
	entityKinds := make(map[EntityKind]struct{}, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		entityKinds[EntityKind(entity.Identifier)] = struct{}{}
	}
	resourceIDs := make(map[string]struct{}, len(v.Resources))
	for _, resource := range v.Resources {
		if resource.Identifier == "" {
			return errors.New("temporal resource identifier is required")
		}
		if _, known := entityKinds[resource.Kind]; !known {
			return fmt.Errorf("temporal resource %q has unknown entity kind %q", resource.Identifier, resource.Kind)
		}
		if _, duplicate := resourceIDs[resource.Identifier]; duplicate {
			return fmt.Errorf("duplicate temporal resource %q", resource.Identifier)
		}
		resourceIDs[resource.Identifier] = struct{}{}
	}
	states, err := uniqueStrings("temporal state", v.States)
	if err != nil {
		return err
	}
	if _, exists := states[v.Initial]; !exists {
		return errors.New("temporal initial state is outside the declared state domain")
	}
	actions := make(map[ActionKind]struct{}, len(v.Actions))
	for _, action := range v.Actions {
		if _, known := catalog.Action(string(action)); !known {
			return fmt.Errorf("unknown temporal action %q", action)
		}
		if _, duplicate := actions[action]; duplicate {
			return fmt.Errorf("duplicate temporal action %q", action)
		}
		actions[action] = struct{}{}
	}
	liveOnlyActions := make(map[ActionKind]struct{}, len(v.LiveOnlyActions))
	for _, action := range v.LiveOnlyActions {
		if _, known := catalog.Action(string(action)); !known {
			return fmt.Errorf("unknown temporal live-only action %q", action)
		}
		if _, modeled := actions[action]; modeled {
			return fmt.Errorf("temporal action %q cannot be both modeled and live-only", action)
		}
		if _, duplicate := liveOnlyActions[action]; duplicate {
			return fmt.Errorf("duplicate temporal live-only action %q", action)
		}
		liveOnlyActions[action] = struct{}{}
	}
	transitions := make(map[TemporalTransition]struct{}, len(v.Transitions))
	for _, transition := range v.Transitions {
		if _, exists := actions[transition.Action]; !exists {
			return fmt.Errorf("temporal transition references undeclared action %q", transition.Action)
		}
		if _, exists := states[transition.FromState]; !exists {
			return fmt.Errorf("temporal transition references unknown source state %q", transition.FromState)
		}
		if _, exists := states[transition.ToState]; !exists {
			return fmt.Errorf("temporal transition references unknown target state %q", transition.ToState)
		}
		if _, duplicate := transitions[transition]; duplicate {
			return fmt.Errorf("duplicate temporal transition %+v", transition)
		}
		transitions[transition] = struct{}{}
	}
	fairnessIDs := make([]string, len(v.Fairness))
	fairnessSeen := make(map[string]struct{}, len(v.Fairness))
	for index, fairness := range v.Fairness {
		if fairness.Identifier == "" || fairness.Kind != "responsive" || len(fairness.EnabledStates) == 0 {
			return errors.New("temporal fairness requires an identifier, responsive kind, and enabled states")
		}
		if _, exists := actions[fairness.Action]; !exists {
			return fmt.Errorf("temporal fairness references undeclared action %q", fairness.Action)
		}
		if _, duplicate := fairnessSeen[fairness.Identifier]; duplicate {
			return fmt.Errorf("duplicate temporal fairness assumption %q", fairness.Identifier)
		}
		fairnessSeen[fairness.Identifier] = struct{}{}
		fairnessIDs[index] = fairness.Identifier
		for _, state := range fairness.EnabledStates {
			if _, exists := states[state]; !exists {
				return fmt.Errorf("temporal fairness references unknown enabled state %q", state)
			}
		}
	}
	for _, state := range append(append([]string(nil), v.Progress.TriggerStates...), v.Progress.GoalStates...) {
		if _, exists := states[state]; !exists {
			return fmt.Errorf("temporal progress references unknown state %q", state)
		}
	}
	if err := v.Relation.validate(); err != nil {
		return err
	}
	if v.Proof != nil {
		if v.Proof.Theorem == "" || v.Proof.Statement == "" ||
			v.Proof.ResultClass != ResultClassTemporalProved || v.Proof.Axioms == nil ||
			!slices.Equal(v.Proof.FairnessAssumptions, fairnessIDs) {
			return errors.New("temporal proof requires a resolved theorem and the exact fairness inventory")
		}
		expectedTrust := TrustBadgeKernel
		if len(v.Proof.Axioms) != 0 {
			expectedTrust = TrustBadgeKernelWithDeclaredAxioms
		}
		if v.Proof.TrustBadge != expectedTrust {
			return errors.New("temporal proof trust does not match its axiom inventory")
		}
	}
	return nil
}

func (v TemporalView) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func (l TemporalLasso) Validate(view TemporalView) error {
	if err := view.Validate(); err != nil {
		return err
	}
	if len(l.States) == 0 || len(l.States) > view.Bounds.MaxTraceLength ||
		len(l.Actions) != len(l.States) || l.LoopStart < 0 || l.LoopStart >= len(l.States) ||
		l.States[0] != view.Initial {
		return errors.New("bounded temporal lasso requires an initial state, one edge per state, and a valid loop")
	}
	states := make(map[string]struct{}, len(view.States))
	for _, state := range view.States {
		states[state] = struct{}{}
	}
	for index, state := range l.States {
		if _, exists := states[state]; !exists {
			return fmt.Errorf("lasso state %d is outside the temporal domain", index)
		}
		next := l.States[l.LoopStart]
		if index+1 < len(l.States) {
			next = l.States[index+1]
		}
		if l.Actions[index] == "" {
			if state != next {
				return fmt.Errorf("lasso edge %d is an invalid stutter", index)
			}
			continue
		}
		if !slices.Contains(view.Transitions, TemporalTransition{
			Action: l.Actions[index], FromState: state, ToState: next,
		}) {
			return fmt.Errorf("lasso edge %d is not a canonical temporal transition", index)
		}
	}
	for _, fairness := range view.Fairness {
		for index, state := range l.States {
			if slices.Contains(fairness.EnabledStates, state) && !lassoEventuallyActs(l, index, fairness.Action) {
				return fmt.Errorf("lasso violates fairness assumption %q", fairness.Identifier)
			}
		}
	}
	for index, state := range l.States {
		if slices.Contains(view.Progress.TriggerStates, state) && !lassoEventuallyReaches(l, index, view.Progress.GoalStates) {
			return nil
		}
	}
	return errors.New("lasso does not witness a progress violation")
}

func (i TemporalLassoReplayInput) Validate() error {
	if i.FormatVersion != TemporalLassoReplayInputFormatVersion || i.Target == "" || i.Property == "" ||
		i.World == "" || i.Variant == "" || !validHash(i.SemanticHash) {
		return errors.New("complete temporal lasso replay identity and provenance are required")
	}
	view, found, err := DefaultTemporalView(i.Variant)
	if err != nil {
		return err
	}
	if !found || view.Target != i.Target || view.Property != i.Property || view.World != i.World ||
		view.SemanticHash != i.SemanticHash {
		return errors.New("temporal lasso replay input does not match a generated executable view")
	}
	return i.Lasso.Validate(view)
}

func (i TemporalLassoReplayInput) CanonicalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(i)
}

func (i TemporalLassoReplayInput) Digest() (string, error) {
	encoded, err := i.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digestBytes(encoded), nil
}

func (r TemporalLassoReplayReceipt) Validate() error {
	if r.FormatVersion != TemporalLassoReplayReceiptFormatVersion || !validHash(r.LassoDigest) ||
		r.Status != TraceReplayAccepted || r.TrustBadge != TrustBadgeCheckedCertificate || r.Axioms == nil {
		return errors.New("temporal lasso replay receipt requires an accepted checked-certificate digest")
	}
	input := TemporalLassoReplayInput{
		FormatVersion: TemporalLassoReplayInputFormatVersion,
		Target:        r.Target,
		Property:      r.Property,
		World:         r.World,
		Variant:       r.Variant,
		SemanticHash:  r.SemanticHash,
		Lasso:         r.Lasso,
	}
	digest, err := input.Digest()
	if err != nil || digest != r.LassoDigest {
		return errors.New("temporal lasso replay receipt digest does not match its checked lasso")
	}
	return validateOrderedStrings("temporal lasso replay axiom", r.Axioms)
}

func lassoEventuallyActs(lasso TemporalLasso, start int, action ActionKind) bool {
	for index := start; index < len(lasso.Actions); index++ {
		if lasso.Actions[index] == action {
			return true
		}
	}
	for index := lasso.LoopStart; index < min(start, len(lasso.Actions)); index++ {
		if lasso.Actions[index] == action {
			return true
		}
	}
	return false
}

func lassoEventuallyReaches(lasso TemporalLasso, start int, goals []string) bool {
	for index := start; index < len(lasso.States); index++ {
		if slices.Contains(goals, lasso.States[index]) {
			return true
		}
	}
	for index := lasso.LoopStart; index < min(start, len(lasso.States)); index++ {
		if slices.Contains(goals, lasso.States[index]) {
			return true
		}
	}
	return false
}

func catalogTargetProperty(catalog Catalog, target TargetID, property PropertyID) bool {
	for _, declaration := range catalog.Targets {
		if declaration.Identifier == string(target) && slices.Contains(declaration.Properties, string(property)) {
			return true
		}
	}
	return false
}

func uniqueStrings(kind string, values []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return nil, fmt.Errorf("%s cannot be empty", kind)
		}
		if _, duplicate := result[value]; duplicate {
			return nil, fmt.Errorf("duplicate %s %q", kind, value)
		}
		result[value] = struct{}{}
	}
	return result, nil
}
