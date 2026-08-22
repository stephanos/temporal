package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const FormatVersion = "umpire3/evidence-graph/v2"

type Fact struct {
	Identifier                string   `json:"identifier"`
	Kind                      string   `json:"kind"`
	Value                     bool     `json:"value"`
	SourceIdentity            string   `json:"sourceIdentity"`
	ClockDomain               string   `json:"clockDomain"`
	SourceSequence            int64    `json:"sourceSequence"`
	AuthoritativeTimeUnixNano int64    `json:"authoritativeTimeUnixNano,omitempty"`
	ObservedAtUnixNano        int64    `json:"observedAtUnixNano"`
	Reference                 string   `json:"reference"`
	CausalReferences          []string `json:"causalReferences"`
	EntityIdentity            string   `json:"entityIdentity"`
	Lineage                   []string `json:"lineage"`
	PayloadDigest             string   `json:"payloadDigest,omitempty"`
}

type Action struct {
	Identifier     string   `json:"identifier"`
	Kind           string   `json:"kind"`
	Outcome        string   `json:"outcome"`
	SourceIdentity string   `json:"sourceIdentity"`
	Reference      string   `json:"reference"`
	EntityIdentity string   `json:"entityIdentity,omitempty"`
	Lineage        []string `json:"lineage"`
	PayloadDigest  string   `json:"payloadDigest,omitempty"`
}

type Relation struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type Claim struct {
	Property string `json:"property"`
	Verdict  string `json:"verdict"`
	Reason   string `json:"reason,omitempty"`
}

type Graph struct {
	FormatVersion string     `json:"formatVersion"`
	Facts         []Fact     `json:"facts"`
	Actions       []Action   `json:"actions"`
	Relations     []Relation `json:"relations"`
	Omissions     []string   `json:"omissions"`
	Claims        []Claim    `json:"claims"`
}

type Limits struct {
	MaxFacts int
	MaxBytes int64
}

type Builder struct {
	limits Limits
	graph  Graph
	bytes  int64
}

type ContradictionError struct {
	Kind  string
	Facts []string
}

func (e *ContradictionError) Error() string {
	return fmt.Sprintf("contradictory evidence for %q in facts %v", e.Kind, e.Facts)
}

func NewBuilder(limits Limits) *Builder {
	return &Builder{limits: limits, graph: Graph{FormatVersion: FormatVersion}}
}

func (b *Builder) AddFact(fact Fact) error {
	if b.limits.MaxFacts <= 0 || b.limits.MaxBytes <= 0 {
		return errors.New("positive evidence fact and byte limits are required")
	}
	if err := fact.validate(); err != nil {
		return err
	}
	if len(b.graph.Facts) == b.limits.MaxFacts {
		return fmt.Errorf("evidence fact limit %d exceeded", b.limits.MaxFacts)
	}
	encoded, err := json.Marshal(fact)
	if err != nil {
		return fmt.Errorf("measure evidence fact: %w", err)
	}
	if b.bytes+int64(len(encoded)) > b.limits.MaxBytes {
		return fmt.Errorf("evidence byte limit %d exceeded", b.limits.MaxBytes)
	}
	b.bytes += int64(len(encoded))
	b.graph.Facts = append(b.graph.Facts, fact)
	return nil
}

func (b *Builder) AddAction(action Action) error {
	if err := action.validate(); err != nil {
		return err
	}
	b.graph.Actions = append(b.graph.Actions, action)
	return nil
}

func (b *Builder) AddRelation(relation Relation) error {
	if relation.Kind == "" || relation.Source == "" || relation.Target == "" {
		return errors.New("complete evidence relation is required")
	}
	b.graph.Relations = append(b.graph.Relations, relation)
	return nil
}

func (b *Builder) AddOmission(omission string) {
	if omission != "" {
		b.graph.Omissions = append(b.graph.Omissions, omission)
	}
}

func (b *Builder) AddClaim(claim Claim) error {
	if claim.Property == "" || claim.Verdict == "" {
		return errors.New("complete evidence claim is required")
	}
	b.graph.Claims = append(b.graph.Claims, claim)
	return nil
}

func (b *Builder) Build() (Graph, error) {
	graph := b.graph
	if err := graph.Validate(); err != nil {
		return graph, err
	}
	return graph, nil
}

func (g Graph) Validate() error {
	if g.FormatVersion != "" && g.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported evidence graph format %q", g.FormatVersion)
	}
	identifiers := make(map[string]struct{}, len(g.Facts))
	values := make(map[string]Fact, len(g.Facts))
	for _, fact := range g.Facts {
		if err := fact.validate(); err != nil {
			return fmt.Errorf("fact %q: %w", fact.Identifier, err)
		}
		if _, duplicate := identifiers[fact.Identifier]; duplicate {
			return fmt.Errorf("duplicate evidence fact %q", fact.Identifier)
		}
		identifiers[fact.Identifier] = struct{}{}
		key := contradictionKey(fact)
		if previous, exists := values[key]; exists && previous.Value != fact.Value {
			return &ContradictionError{Kind: fact.Kind, Facts: []string{previous.Identifier, fact.Identifier}}
		}
		values[key] = fact
	}
	actionIdentifiers := make(map[string]struct{}, len(g.Actions))
	for _, action := range g.Actions {
		if err := action.validate(); err != nil {
			return fmt.Errorf("action %q: %w", action.Identifier, err)
		}
		if _, duplicate := actionIdentifiers[action.Identifier]; duplicate {
			return fmt.Errorf("duplicate evidence action %q", action.Identifier)
		}
		actionIdentifiers[action.Identifier] = struct{}{}
	}
	return nil
}

func (g Graph) Before(beforeID, afterID string) (bool, error) {
	if err := g.Validate(); err != nil {
		return false, err
	}
	byID := make(map[string]Fact, len(g.Facts))
	byReference := make(map[string]Fact, len(g.Facts))
	for _, fact := range g.Facts {
		byID[fact.Identifier] = fact
		byReference[fact.Reference] = fact
	}
	before, beforeExists := byID[beforeID]
	after, afterExists := byID[afterID]
	if !beforeExists || !afterExists {
		return false, fmt.Errorf("unknown evidence ordering facts %q and %q", beforeID, afterID)
	}
	if before.SourceIdentity == after.SourceIdentity && before.ClockDomain == after.ClockDomain &&
		before.SourceSequence < after.SourceSequence {
		return true, nil
	}
	visited := make(map[string]struct{})
	var causedBy func(Fact, string) bool
	causedBy = func(current Fact, targetReference string) bool {
		if _, seen := visited[current.Identifier]; seen {
			return false
		}
		visited[current.Identifier] = struct{}{}
		if slices.Contains(current.CausalReferences, targetReference) {
			return true
		}
		for _, reference := range current.CausalReferences {
			if predecessor, exists := byReference[reference]; exists && causedBy(predecessor, targetReference) {
				return true
			}
		}
		return false
	}
	return causedBy(after, before.Reference), nil
}

func (g Graph) CanonicalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("encode evidence graph: %w", err)
	}
	return encoded, nil
}

func (f Fact) validate() error {
	if f.Identifier == "" || f.Kind == "" || f.Reference == "" {
		return errors.New("fact identifier, kind, and reference are required")
	}
	if f.SourceIdentity == "" {
		return errors.New("source identity is required")
	}
	if f.ClockDomain == "" || f.SourceSequence <= 0 {
		return errors.New("clock domain and positive source sequence are required")
	}
	if f.ObservedAtUnixNano <= 0 {
		return errors.New("observation time is required")
	}
	if f.EntityIdentity == "" || len(f.Lineage) == 0 {
		return errors.New("entity identity and lineage are required")
	}
	for _, identity := range f.Lineage {
		if identity == "" {
			return errors.New("lineage contains an empty identity")
		}
	}
	if f.PayloadDigest != "" && !validDigest(f.PayloadDigest) {
		return errors.New("payload digest must be sha256")
	}
	return nil
}

func (a Action) validate() error {
	if a.Identifier == "" || a.Kind == "" || a.Outcome == "" ||
		a.SourceIdentity == "" || a.Reference == "" {
		return errors.New("complete action evidence identity and outcome are required")
	}
	return nil
}

func contradictionKey(fact Fact) string {
	return fact.Kind + "\x00" + fact.EntityIdentity + "\x00" + strings.Join(fact.Lineage, "\x00")
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
