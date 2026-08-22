package checker

import (
	"bytes"
	_ "embed" // Enable the generated catalog embedding below.
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const FiniteReplayCatalogFormatVersion = "umpire3/finite-replay-catalog/v1"

type FiniteReplayBounds struct {
	MaxDepth       int `json:"maxDepth"`
	MaxStates      int `json:"maxStates"`
	MaxTransitions int `json:"maxTransitions"`
	MaxStateBytes  int `json:"maxStateBytes"`
	MaxWork        int `json:"maxWork"`
}

type FiniteReplayStatistics struct {
	States      int `json:"states"`
	Transitions int `json:"transitions"`
	StateBytes  int `json:"stateBytes"`
}

type FiniteReplayTransition struct {
	From   int    `json:"from"`
	Action string `json:"action"`
	To     int    `json:"to"`
}

type FiniteReplayAttempt struct {
	Action       ActionKind      `json:"action"`
	Outcomes     []ActionOutcome `json:"outcomes"`
	AppliedPaths [][]string      `json:"appliedPaths"`
}

type FiniteReplayTarget struct {
	Target         TargetID                 `json:"target"`
	Property       PropertyID               `json:"property"`
	World          string                   `json:"world"`
	Variant        string                   `json:"variant"`
	SemanticHash   string                   `json:"semanticHash"`
	CanonicalModel string                   `json:"canonicalModel"`
	Relation       ResolvedDeclaration      `json:"relation"`
	ResultClass    ResultClass              `json:"resultClass"`
	TrustBadge     TrustBadge               `json:"trustBadge"`
	Bounds         FiniteReplayBounds       `json:"bounds"`
	Statistics     FiniteReplayStatistics   `json:"statistics"`
	InitialStates  []int                    `json:"initialStates"`
	StateCount     int                      `json:"stateCount"`
	Transitions    []FiniteReplayTransition `json:"transitions"`
	Attempts       []FiniteReplayAttempt    `json:"attempts"`
}

type FiniteReplayCatalog struct {
	FormatVersion string               `json:"formatVersion"`
	SemanticHash  string               `json:"semanticHash"`
	CatalogHash   string               `json:"catalogHash"`
	Targets       []FiniteReplayTarget `json:"targets"`
}

func DecodeFiniteReplayCatalog(encoded []byte) (FiniteReplayCatalog, error) {
	var catalog FiniteReplayCatalog
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "finite replay catalog", &catalog); err != nil {
		return FiniteReplayCatalog{}, err
	}
	for index := range catalog.Targets {
		catalog.Targets[index].Relation.Derive()
	}
	if err := catalog.Validate(); err != nil {
		return FiniteReplayCatalog{}, err
	}
	return catalog, nil
}

func (c FiniteReplayCatalog) Validate() error {
	if c.FormatVersion != FiniteReplayCatalogFormatVersion || !validHash(c.SemanticHash) ||
		!validHash(c.CatalogHash) || len(c.Targets) == 0 {
		return errors.New("complete finite replay catalog provenance and targets are required")
	}
	semanticCatalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := semanticCatalog.Digest()
	if err != nil {
		return err
	}
	if c.CatalogHash != catalogHash {
		return fmt.Errorf("finite replay catalog hash %q does not match semantic catalog %q", c.CatalogHash, catalogHash)
	}
	composition, err := DefaultComposition()
	if err != nil {
		return err
	}
	type targetProperty struct {
		target   TargetID
		property PropertyID
	}
	expected := make(map[targetProperty]TargetProjection)
	for _, target := range composition.Targets {
		if target.Identifier == TargetIDNexusCancellation {
			continue
		}
		for _, property := range target.Properties {
			expected[targetProperty{target: target.Identifier, property: property}] = target
		}
	}
	seen := make(map[targetProperty]struct{}, len(c.Targets))
	for index := range c.Targets {
		target := &c.Targets[index]
		key := targetProperty{target: target.Target, property: target.Property}
		projection, known := expected[key]
		if !known {
			return fmt.Errorf("finite replay catalog has unknown target/property %q/%q", target.Target, target.Property)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("finite replay catalog has duplicate target/property %q/%q", target.Target, target.Property)
		}
		seen[key] = struct{}{}
		if target.SemanticHash != c.SemanticHash {
			return fmt.Errorf("finite replay target %q semantic hash does not match catalog", target.Target)
		}
		if err := target.validate(projection); err != nil {
			return fmt.Errorf("finite replay target %q property %q: %w", target.Target, target.Property, err)
		}
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("finite replay catalog covers %d target/property pairs; %d require finite replay", len(seen), len(expected))
	}
	return nil
}

func (t FiniteReplayTarget) validate(projection TargetProjection) error {
	if t.Target == "" || t.Property == "" || t.World == "" || t.Variant == "" ||
		!validHash(t.SemanticHash) || t.CanonicalModel == "" {
		return errors.New("complete finite replay identity and provenance are required")
	}
	if err := t.Relation.Validate(); err != nil {
		return fmt.Errorf("validate relation: %w", err)
	}
	if t.ResultClass != ResultClassFiniteExhaustive || t.TrustBadge != TrustBadgeCheckedCertificate {
		return errors.New("finite replay graph must be an exhaustive checked certificate")
	}
	if t.Bounds.MaxDepth <= 0 || t.Bounds.MaxStates <= 0 || t.Bounds.MaxTransitions <= 0 ||
		t.Bounds.MaxStateBytes <= 0 || t.Bounds.MaxWork <= 0 {
		return errors.New("positive finite replay bounds are required")
	}
	if t.StateCount <= 0 || t.StateCount > t.Bounds.MaxStates ||
		t.Statistics.States != t.StateCount || t.Statistics.Transitions < len(t.Transitions) ||
		t.Statistics.Transitions > t.Bounds.MaxTransitions || t.Statistics.StateBytes <= 0 ||
		t.Statistics.StateBytes > t.Bounds.MaxStateBytes {
		return errors.New("finite replay statistics exceed or contradict their enforced bounds")
	}
	if len(t.InitialStates) == 0 || len(t.Transitions) == 0 || len(t.Attempts) == 0 {
		return errors.New("finite replay graph, initial states, and attempts are required")
	}
	initials := make(map[int]struct{}, len(t.InitialStates))
	for _, state := range t.InitialStates {
		if state < 0 || state >= t.StateCount {
			return fmt.Errorf("initial state %d is outside the graph", state)
		}
		if _, duplicate := initials[state]; duplicate {
			return fmt.Errorf("duplicate initial state %d", state)
		}
		initials[state] = struct{}{}
	}
	type edgeKey struct {
		from   int
		action string
		to     int
	}
	edges := make(map[edgeKey]struct{}, len(t.Transitions))
	graphActions := make(map[string]struct{})
	for _, transition := range t.Transitions {
		if transition.From < 0 || transition.From >= t.StateCount || transition.To < 0 ||
			transition.To >= t.StateCount || transition.Action == "" {
			return errors.New("finite replay transition is incomplete or outside the graph")
		}
		key := edgeKey{from: transition.From, action: transition.Action, to: transition.To}
		if _, duplicate := edges[key]; duplicate {
			return fmt.Errorf("duplicate finite replay transition %d/%s/%d", transition.From, transition.Action, transition.To)
		}
		edges[key] = struct{}{}
		graphActions[transition.Action] = struct{}{}
	}
	reachable := t.reachableStates()
	if len(reachable) != t.StateCount {
		return fmt.Errorf("finite replay graph reaches %d states; certificate contains %d", len(reachable), t.StateCount)
	}
	expectedActions := make([]ActionKind, len(projection.RetainedActions))
	for index, action := range projection.RetainedActions {
		expectedActions[index] = ActionKind(action)
	}
	actualActions := make([]ActionKind, len(t.Attempts))
	mappedGraphActions := make(map[string]struct{})
	for index, attempt := range t.Attempts {
		actualActions[index] = attempt.Action
		if attempt.Action == "" || len(attempt.Outcomes) == 0 || len(attempt.AppliedPaths) == 0 {
			return errors.New("finite replay attempt is incomplete")
		}
		outcomes := make(map[ActionOutcome]struct{}, len(attempt.Outcomes))
		for _, outcome := range attempt.Outcomes {
			if !validActionOutcome(outcome) {
				return fmt.Errorf("attempt %q has unknown outcome %q", attempt.Action, outcome)
			}
			if _, duplicate := outcomes[outcome]; duplicate {
				return fmt.Errorf("attempt %q has duplicate outcome %q", attempt.Action, outcome)
			}
			outcomes[outcome] = struct{}{}
		}
		if _, applied := outcomes[ActionOutcomeApplied]; !applied {
			return fmt.Errorf("attempt %q has no applied outcome", attempt.Action)
		}
		for _, path := range attempt.AppliedPaths {
			for _, action := range path {
				if _, known := graphActions[action]; !known {
					return fmt.Errorf("attempt %q references unknown graph action %q", attempt.Action, action)
				}
				mappedGraphActions[action] = struct{}{}
			}
			if len(path) != 0 && !t.pathExecutableSomewhere(path) {
				return fmt.Errorf("attempt %q has no executable applied path %v", attempt.Action, path)
			}
		}
	}
	if !slices.Equal(actualActions, expectedActions) {
		return fmt.Errorf("attempt actions %v do not exactly cover retained target actions %v", actualActions, expectedActions)
	}
	if len(mappedGraphActions) != len(graphActions) {
		return fmt.Errorf("attempt mappings cover %d graph actions; exact graph contains %d", len(mappedGraphActions), len(graphActions))
	}
	return nil
}

func (c FiniteReplayCatalog) Target(target TargetID, property PropertyID) (FiniteReplayTarget, bool) {
	for _, candidate := range c.Targets {
		if candidate.Target == target && candidate.Property == property {
			return candidate, true
		}
	}
	return FiniteReplayTarget{}, false
}

func (c FiniteReplayCatalog) CanonicalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	canonical := c
	canonical.Targets = append([]FiniteReplayTarget(nil), c.Targets...)
	slices.SortFunc(canonical.Targets, func(left, right FiniteReplayTarget) int {
		if comparison := compareStrings(string(left.Target), string(right.Target)); comparison != 0 {
			return comparison
		}
		return compareStrings(string(left.Property), string(right.Property))
	})
	for index := range canonical.Targets {
		target := &canonical.Targets[index]
		target.InitialStates = append([]int(nil), target.InitialStates...)
		slices.Sort(target.InitialStates)
		target.Transitions = append([]FiniteReplayTransition(nil), target.Transitions...)
		slices.SortFunc(target.Transitions, func(left, right FiniteReplayTransition) int {
			if left.From != right.From {
				return left.From - right.From
			}
			if comparison := compareStrings(left.Action, right.Action); comparison != 0 {
				return comparison
			}
			return left.To - right.To
		})
	}
	return json.Marshal(canonical)
}

func (t FiniteReplayTarget) reachableStates() map[int]struct{} {
	reachable := make(map[int]struct{}, t.StateCount)
	queue := append([]int(nil), t.InitialStates...)
	for len(queue) != 0 {
		state := queue[0]
		queue = queue[1:]
		if _, visited := reachable[state]; visited {
			continue
		}
		reachable[state] = struct{}{}
		for _, transition := range t.Transitions {
			if transition.From == state {
				queue = append(queue, transition.To)
			}
		}
	}
	return reachable
}

func (t FiniteReplayTarget) pathExecutableSomewhere(path []string) bool {
	for state := range t.reachableStates() {
		if len(t.follow([]int{state}, path)) != 0 {
			return true
		}
	}
	return false
}

func (t FiniteReplayTarget) follow(states []int, path []string) []int {
	for _, action := range path {
		next := make(map[int]struct{})
		for _, state := range states {
			for _, transition := range t.Transitions {
				if transition.From == state && transition.Action == action {
					next[transition.To] = struct{}{}
				}
			}
		}
		states = states[:0]
		for state := range next {
			states = append(states, state)
		}
		if len(states) == 0 {
			return nil
		}
	}
	slices.Sort(states)
	return slices.Compact(states)
}
