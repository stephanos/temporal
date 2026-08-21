package observation

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

const (
	FormatVersion = "umpire3/observation-programs/v1"

	OperationExists                   = "exists"
	OperationAllExist                 = "all-exist"
	OperationAbsentWhenClosed         = "absent-when-closed"
	OperationAllExistAbsentWhenClosed = "all-exist-absent-when-closed"
	EpochRelationEqual                = "equal"
	EpochRelationNotEqual             = "not-equal"
	EpochRelationUnconstrained        = ""
)

type Truth string

const (
	True     Truth = "true"
	False    Truth = "false"
	Unknown  Truth = "unknown"
	Conflict Truth = "conflict"
)

type Selector struct {
	FactType              string `json:"factType"`
	Kind                  string `json:"kind"`
	OwnerEpochRelation    string `json:"ownerEpochRelation,omitempty"`
	CancellationCommitted *bool  `json:"cancellationCommitted,omitempty"`
	Outcome               string `json:"outcome,omitempty"`
	Closed                *bool  `json:"closed,omitempty"`
}

type Program struct {
	Identifier  string                 `json:"identifier"`
	Observation protocol.ObservationID `json:"observation"`
	Operation   string                 `json:"operation"`
	Matches     []Selector             `json:"matches,omitempty"`
	Violations  []Selector             `json:"violations,omitempty"`
	Closures    []Selector             `json:"closures,omitempty"`
}

type Evaluation struct {
	Value   Truth    `json:"value"`
	Support []string `json:"support,omitempty"`
}

type Fixture struct {
	Identifier  string                 `json:"identifier"`
	Observation protocol.ObservationID `json:"observation"`
	Facts       []Fact                 `json:"facts"`
	Expected    Evaluation             `json:"expected"`
}

type Catalog struct {
	FormatVersion string    `json:"formatVersion"`
	SemanticHash  string    `json:"semanticHash"`
	CatalogHash   string    `json:"catalogHash"`
	Programs      []Program `json:"programs"`
	Fixtures      []Fixture `json:"fixtures"`
}

//go:embed generated/programs.json
var defaultCatalogJSON []byte

func DecodeCatalog(encoded []byte) (Catalog, error) {
	if int64(len(encoded)) > protocol.DefaultDecodeLimit {
		return Catalog{}, errors.New("decode observation catalog: input exceeds size limit")
	}
	var catalog Catalog
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(encoded), protocol.DefaultDecodeLimit+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode observation catalog: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return Catalog{}, errors.New("decode observation catalog: trailing JSON value")
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func DefaultCatalog() (Catalog, error) {
	return DecodeCatalog(defaultCatalogJSON)
}

func (c Catalog) CanonicalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

func (c Catalog) Validate() error {
	if c.FormatVersion != FormatVersion || !validDigest(c.SemanticHash) || !validDigest(c.CatalogHash) ||
		len(c.Programs) == 0 || len(c.Fixtures) == 0 {
		return errors.New("complete versioned observation catalog is required")
	}
	semanticCatalog, err := protocol.DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := semanticCatalog.Digest()
	if err != nil {
		return err
	}
	if c.CatalogHash != catalogHash {
		return fmt.Errorf("observation catalog hash %q does not match semantic catalog %q", c.CatalogHash, catalogHash)
	}
	knownObservations := make(map[protocol.ObservationID]struct{}, len(semanticCatalog.Observations))
	for _, declaration := range semanticCatalog.Observations {
		knownObservations[protocol.ObservationID(declaration.Identifier)] = struct{}{}
	}
	programs := make(map[protocol.ObservationID]Program, len(c.Programs))
	identifiers := make(map[string]struct{}, len(c.Programs))
	for _, program := range c.Programs {
		if _, duplicate := identifiers[program.Identifier]; program.Identifier == "" || duplicate {
			return fmt.Errorf("observation program identifier %q is empty or duplicated", program.Identifier)
		}
		identifiers[program.Identifier] = struct{}{}
		if _, known := knownObservations[program.Observation]; !known {
			return fmt.Errorf("program %q references unknown observation %q", program.Identifier, program.Observation)
		}
		if _, duplicate := programs[program.Observation]; duplicate {
			return fmt.Errorf("duplicate program for observation %q", program.Observation)
		}
		if err := program.validate(); err != nil {
			return fmt.Errorf("program %q: %w", program.Identifier, err)
		}
		programs[program.Observation] = program
	}
	for _, declaration := range semanticCatalog.Observations {
		observation := protocol.ObservationID(declaration.Identifier)
		if _, exists := programs[observation]; !exists {
			return fmt.Errorf("semantic observation %q has no program", observation)
		}
	}
	fixtureIDs := make(map[string]struct{}, len(c.Fixtures))
	coveredPrograms := make(map[protocol.ObservationID]struct{}, len(c.Programs))
	coveredValues := make(map[Truth]struct{}, 4)
	for _, fixture := range c.Fixtures {
		if _, duplicate := fixtureIDs[fixture.Identifier]; fixture.Identifier == "" || duplicate {
			return fmt.Errorf("observation fixture identifier %q is empty or duplicated", fixture.Identifier)
		}
		fixtureIDs[fixture.Identifier] = struct{}{}
		program, ok := programs[fixture.Observation]
		if !ok {
			return fmt.Errorf("fixture %q references observation without a program", fixture.Identifier)
		}
		if actual := program.Evaluate(fixture.Facts); !evaluationEqual(actual, fixture.Expected) {
			return fmt.Errorf("fixture %q evaluated to %+v, expected %+v", fixture.Identifier, actual, fixture.Expected)
		}
		coveredPrograms[fixture.Observation] = struct{}{}
		coveredValues[fixture.Expected.Value] = struct{}{}
	}
	if len(coveredPrograms) != len(programs) {
		return errors.New("every observation program requires a checked fixture")
	}
	for _, value := range []Truth{True, False, Unknown, Conflict} {
		if _, covered := coveredValues[value]; !covered {
			return fmt.Errorf("observation fixtures do not cover %q", value)
		}
	}
	return nil
}

func (c Catalog) Program(observation protocol.ObservationID) (Program, bool) {
	for _, program := range c.Programs {
		if program.Observation == observation {
			return program, true
		}
	}
	return Program{}, false
}

func (p Program) validate() error {
	if p.Observation == "" {
		return errors.New("observation is required")
	}
	switch p.Operation {
	case OperationExists:
		if len(p.Matches) == 0 || len(p.Violations) != 0 || len(p.Closures) != 0 {
			return errors.New("exists requires only match selectors")
		}
	case OperationAllExist:
		if len(p.Matches) == 0 || len(p.Violations) != 0 || len(p.Closures) != 0 {
			return errors.New("all-exist requires only match selectors")
		}
	case OperationAbsentWhenClosed:
		if len(p.Matches) != 0 || len(p.Violations) == 0 || len(p.Closures) == 0 {
			return errors.New("absent-when-closed requires violation and closure selectors")
		}
	case OperationAllExistAbsentWhenClosed:
		if len(p.Matches) == 0 || len(p.Violations) == 0 || len(p.Closures) == 0 {
			return errors.New("all-exist-absent-when-closed requires match, violation, and closure selectors")
		}
	default:
		return fmt.Errorf("unknown operation %q", p.Operation)
	}
	for _, selectors := range [][]Selector{p.Matches, p.Violations, p.Closures} {
		for _, selector := range selectors {
			if err := selector.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s Selector) validate() error {
	if s.Kind == "" {
		return errors.New("selector kind is required")
	}
	switch s.FactType {
	case FactTypeHistoryEvent:
		if s.Outcome != "" || s.Closed != nil {
			return errors.New("history selector has unsupported predicates")
		}
		if s.OwnerEpochRelation != EpochRelationUnconstrained &&
			s.OwnerEpochRelation != EpochRelationEqual && s.OwnerEpochRelation != EpochRelationNotEqual {
			return fmt.Errorf("unknown owner epoch relation %q", s.OwnerEpochRelation)
		}
	case FactTypeMechanismReceipt:
		if s.OwnerEpochRelation != "" || s.CancellationCommitted != nil || s.Closed != nil {
			return errors.New("mechanism selector has unsupported predicates")
		}
	case FactTypeEvidenceWindow:
		if s.OwnerEpochRelation != "" || s.CancellationCommitted != nil || s.Outcome != "" || s.Closed == nil {
			return errors.New("evidence-window selector requires only closure state")
		}
	default:
		return fmt.Errorf("unknown fact type %q", s.FactType)
	}
	return nil
}

func (p Program) Evaluate(facts []Fact) Evaluation {
	facts, conflict := normalizeFacts(facts)
	if len(conflict) != 0 {
		return Evaluation{Value: Conflict, Support: conflict}
	}
	switch p.Operation {
	case OperationExists:
		if support := matchingSupport(p.Matches, facts); len(support) != 0 {
			return Evaluation{Value: True, Support: support}
		}
		return Evaluation{Value: Unknown}
	case OperationAllExist:
		if allSelectorsMatch(p.Matches, facts) {
			return Evaluation{Value: True, Support: matchingSupport(p.Matches, facts)}
		}
		return Evaluation{Value: Unknown}
	case OperationAbsentWhenClosed:
		if support := matchingSupport(p.Violations, facts); len(support) != 0 {
			return Evaluation{Value: False, Support: support}
		}
		if support := matchingSupport(p.Closures, facts); len(support) != 0 {
			return Evaluation{Value: True, Support: support}
		}
		return Evaluation{Value: Unknown}
	case OperationAllExistAbsentWhenClosed:
		if support := matchingSupport(p.Violations, facts); len(support) != 0 {
			return Evaluation{Value: False, Support: support}
		}
		if !allSelectorsMatch(p.Matches, facts) {
			return Evaluation{Value: Unknown}
		}
		closureSupport := matchingSupport(p.Closures, facts)
		if len(closureSupport) == 0 {
			return Evaluation{Value: Unknown}
		}
		return Evaluation{Value: True, Support: append(
			matchingSupport(p.Matches, facts), closureSupport...,
		)}
	default:
		return Evaluation{Value: Conflict}
	}
}

func allSelectorsMatch(selectors []Selector, facts []Fact) bool {
	for _, selector := range selectors {
		matched := false
		for _, fact := range facts {
			if selector.matches(fact) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func normalizeFacts(facts []Fact) ([]Fact, []string) {
	byID := make(map[string]Fact, len(facts))
	entity := ""
	var conflicts []string
	for _, fact := range facts {
		if err := fact.Validate(); err != nil {
			conflicts = append(conflicts, fact.Identifier)
			continue
		}
		if entity == "" {
			entity = fact.Source.EntityIdentity
		} else if fact.Source.EntityIdentity != entity {
			conflicts = append(conflicts, fact.Identifier)
			continue
		}
		if existing, duplicate := byID[fact.Identifier]; duplicate {
			left, _ := json.Marshal(existing)
			right, _ := json.Marshal(fact)
			if !bytes.Equal(left, right) {
				conflicts = append(conflicts, fact.Identifier)
			}
			continue
		}
		byID[fact.Identifier] = fact
	}
	result := make([]Fact, 0, len(byID))
	for _, fact := range byID {
		result = append(result, fact)
	}
	slices.SortFunc(result, func(left, right Fact) int { return strings.Compare(left.Identifier, right.Identifier) })
	slices.Sort(conflicts)
	return result, slices.Compact(conflicts)
}

func matchingSupport(selectors []Selector, facts []Fact) []string {
	var support []string
	for _, fact := range facts {
		for _, selector := range selectors {
			if selector.matches(fact) {
				support = append(support, fact.Identifier)
				break
			}
		}
	}
	slices.Sort(support)
	return slices.Compact(support)
}

func (s Selector) matches(fact Fact) bool {
	factType, kind := fact.factTypeAndKind()
	if s.FactType != factType || s.Kind != kind {
		return false
	}
	if fact.History != nil {
		if s.CancellationCommitted != nil &&
			(fact.History.CancellationCommitted == nil ||
				*fact.History.CancellationCommitted != *s.CancellationCommitted) {
			return false
		}
		if s.OwnerEpochRelation != "" {
			if fact.History.OwnerEpoch == nil || fact.History.CurrentOwnerEpoch == nil {
				return false
			}
			equal := *fact.History.OwnerEpoch == *fact.History.CurrentOwnerEpoch
			if (s.OwnerEpochRelation == EpochRelationEqual) != equal {
				return false
			}
		}
	}
	if fact.Mechanism != nil && s.Outcome != "" && fact.Mechanism.Outcome != s.Outcome {
		return false
	}
	return fact.Window == nil || s.Closed == nil || fact.Window.Closed == *s.Closed
}

func evaluationEqual(left, right Evaluation) bool {
	return left.Value == right.Value && slices.Equal(left.Support, right.Support)
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
