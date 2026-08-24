package finite

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"slices"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

const (
	CertificateFormatVersion = "umpire3/native-certificate/v1"
	ReceiptFormatVersion     = "umpire3/native-certificate-receipt/v1"
)

const (
	closureRecomputedSuccessors = "recomputed-successors"
	symmetryReplicatedWorlds    = "replicated-disjoint-worlds"
)

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type CompactNode struct {
	State  protocolchecker.FirstOrderState `json:"state"`
	Parent int                             `json:"parent"`
	Action protocolcatalog.ActionKind      `json:"action,omitempty"`
	Depth  int                             `json:"depth"`
}

type ClosureCertificate struct {
	Kind                  string `json:"kind"`
	ClosedRepresentatives int    `json:"closedRepresentatives"`
	RecomputedTransitions int    `json:"recomputedTransitions"`
}

type SymmetryCertificate struct {
	Kind            string `json:"kind"`
	Replicas        int    `json:"replicas"`
	Representatives int    `json:"representatives"`
	ExpandedStates  int    `json:"expandedStates"`
}

type Statistics struct {
	ExpandedStates       int `json:"expandedStates"`
	RepresentativeStates int `json:"representativeStates"`
	Transitions          int `json:"transitions"`
	StateBytes           int `json:"stateBytes"`
	MaxDepth             int `json:"maxDepth"`
}

type Certificate struct {
	FormatVersion string                     `json:"formatVersion"`
	ViewVersion   string                     `json:"viewVersion"`
	ViewDigest    string                     `json:"viewDigest"`
	Target        protocolcatalog.TargetID   `json:"target"`
	Property      protocolcatalog.PropertyID `json:"property"`
	World         string                     `json:"world"`
	Variant       string                     `json:"variant"`
	SemanticHash  string                     `json:"semanticHash"`
	Termination   string                     `json:"termination"`
	Nodes         []CompactNode              `json:"nodes"`
	Closure       ClosureCertificate         `json:"closure"`
	Symmetry      SymmetryCertificate        `json:"symmetry"`
	Statistics    Statistics                 `json:"statistics"`
	Digest        string                     `json:"digest"`
}

type Receipt struct {
	FormatVersion        string                      `json:"formatVersion"`
	CertificateDigest    string                      `json:"certificateDigest"`
	ViewDigest           string                      `json:"viewDigest"`
	Target               protocolcatalog.TargetID    `json:"target"`
	Property             protocolcatalog.PropertyID  `json:"property"`
	World                string                      `json:"world"`
	Variant              string                      `json:"variant"`
	SemanticHash         string                      `json:"semanticHash"`
	ResultClass          protocolcatalog.ResultClass `json:"resultClass"`
	TrustBadge           protocolcatalog.TrustBadge  `json:"trustBadge"`
	ExpandedStates       int                         `json:"expandedStates"`
	RepresentativeStates int                         `json:"representativeStates"`
	Replicas             int                         `json:"replicas"`
	Nodes                []CompactNode               `json:"nodes"`
	Axioms               []string                    `json:"axioms"`
}

func DecodeCertificate(input io.Reader, limit int64, view protocolchecker.FirstOrderView) (Certificate, error) {
	var certificate Certificate
	if err := decodeStrict(input, limit, &certificate); err != nil {
		return Certificate{}, fmt.Errorf("decode native certificate: %w", err)
	}
	if err := certificate.Validate(view); err != nil {
		return Certificate{}, err
	}
	return certificate, nil
}

func DecodeReceipt(input io.Reader, limit int64, certificate Certificate) (Receipt, error) {
	var receipt Receipt
	if err := decodeStrict(input, limit, &receipt); err != nil {
		return Receipt{}, fmt.Errorf("decode native certificate receipt: %w", err)
	}
	if err := receipt.Validate(certificate); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func (c Certificate) CanonicalJSON(view protocolchecker.FirstOrderView) ([]byte, error) {
	if err := c.Validate(view); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

func (r Receipt) CanonicalJSON(certificate Certificate) ([]byte, error) {
	if err := r.Validate(certificate); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

func (c *Certificate) seal() error {
	c.Digest = ""
	encoded, err := json.Marshal(c)
	if err != nil {
		return err
	}
	c.Digest = digest(encoded)
	return nil
}

func (c Certificate) Validate(view protocolchecker.FirstOrderView) error {
	if err := view.Validate(); err != nil {
		return err
	}
	viewDigest, err := firstOrderViewDigest(view)
	if err != nil {
		return err
	}
	if c.FormatVersion != CertificateFormatVersion || c.ViewVersion != view.FormatVersion ||
		c.ViewDigest != viewDigest || c.Target != view.Target || c.Property != view.Property ||
		c.World != view.World || c.Variant != view.Variant || c.SemanticHash != view.SemanticHash ||
		c.Termination != "exhausted" || len(c.Nodes) == 0 || !digestPattern.MatchString(c.Digest) {
		return errors.New("complete exhausted native certificate identity and provenance are required")
	}
	expected := c
	if err := expected.seal(); err != nil || expected.Digest != c.Digest {
		return errors.New("native certificate digest does not match its contents")
	}
	if c.Symmetry.Kind != symmetryReplicatedWorlds || c.Symmetry.Replicas <= 0 ||
		c.Symmetry.Replicas > 10 || c.Symmetry.Representatives != len(c.Nodes) ||
		c.Symmetry.ExpandedStates != len(c.Nodes)*c.Symmetry.Replicas {
		return errors.New("native certificate requires a complete replicated-world symmetry witness")
	}
	if c.Closure.Kind != closureRecomputedSuccessors ||
		c.Closure.ClosedRepresentatives != len(c.Nodes) {
		return errors.New("native certificate requires recomputed closure for every representative")
	}
	if c.Statistics.ExpandedStates != c.Symmetry.ExpandedStates ||
		c.Statistics.RepresentativeStates != len(c.Nodes) || c.Statistics.Transitions < 0 ||
		c.Statistics.StateBytes <= 0 || c.Statistics.MaxDepth < 0 {
		return errors.New("native certificate statistics do not match its checked scope")
	}
	machine, err := NewMachine(view)
	if err != nil {
		return err
	}
	initials, err := machine.InitialStates()
	if err != nil {
		return err
	}
	initialKeys := make(map[string]struct{}, len(initials))
	for _, state := range initials {
		key, keyErr := machine.StateKey(state)
		if keyErr != nil {
			return keyErr
		}
		initialKeys[key] = struct{}{}
	}
	seen := make(map[string]int, len(c.Nodes))
	rootKeys := make(map[string]struct{}, len(initials))
	maxDepth := 0
	for index, node := range c.Nodes {
		key, keyErr := machine.StateKey(node.State)
		if keyErr != nil {
			return fmt.Errorf("validate native node %d: %w", index, keyErr)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate native representative state %d", index)
		}
		seen[key] = index
		safe, invariantErr := machine.Invariant(node.State)
		if invariantErr != nil {
			return invariantErr
		}
		if !safe {
			return fmt.Errorf("native representative state %d violates the property", index)
		}
		if node.Parent == -1 {
			if node.Action != "" || node.Depth != 0 {
				return fmt.Errorf("native root node %d has parent evidence", index)
			}
			if _, initial := initialKeys[key]; !initial {
				return fmt.Errorf("native root node %d is not initial", index)
			}
			rootKeys[key] = struct{}{}
		} else {
			if node.Parent < 0 || node.Parent >= index || node.Action == "" ||
				node.Depth != c.Nodes[node.Parent].Depth+1 {
				return fmt.Errorf("native node %d has an invalid predecessor", index)
			}
			steps, successorErr := machine.Successors(c.Nodes[node.Parent].State)
			if successorErr != nil {
				return successorErr
			}
			if !slices.ContainsFunc(steps, func(step Step) bool {
				stepKey, stepErr := machine.StateKey(step.State)
				return stepErr == nil && step.Action == node.Action && stepKey == key
			}) {
				return fmt.Errorf("native node %d predecessor edge is not canonical", index)
			}
		}
		maxDepth = max(maxDepth, node.Depth)
	}
	if len(rootKeys) != len(initialKeys) {
		return errors.New("native certificate does not contain every initial state")
	}
	recomputedTransitions := 0
	for index, node := range c.Nodes {
		steps, successorErr := machine.Successors(node.State)
		if successorErr != nil {
			return successorErr
		}
		recomputedTransitions += len(steps)
		for _, step := range steps {
			key, keyErr := machine.StateKey(step.State)
			if keyErr != nil {
				return keyErr
			}
			if _, closed := seen[key]; !closed {
				return fmt.Errorf("native representative state %d has an uncovered successor", index)
			}
		}
	}
	if c.Closure.RecomputedTransitions != recomputedTransitions ||
		c.Statistics.Transitions != recomputedTransitions*c.Symmetry.Replicas ||
		c.Statistics.MaxDepth != maxDepth {
		return errors.New("native certificate closure statistics do not match canonical successors")
	}
	return nil
}

func (r Receipt) Validate(certificate Certificate) error {
	if r.FormatVersion != ReceiptFormatVersion || r.CertificateDigest != certificate.Digest ||
		r.ViewDigest != certificate.ViewDigest ||
		r.Target != certificate.Target || r.Property != certificate.Property || r.World != certificate.World ||
		r.Variant != certificate.Variant || r.SemanticHash != certificate.SemanticHash ||
		r.ResultClass != protocolcatalog.ResultClassFiniteExhaustive ||
		r.TrustBadge != protocolcatalog.TrustBadgeCheckedCertificate ||
		r.ExpandedStates != certificate.Statistics.ExpandedStates ||
		r.RepresentativeStates != certificate.Statistics.RepresentativeStates ||
		r.Replicas != certificate.Symmetry.Replicas || r.Axioms == nil {
		return errors.New("native certificate receipt does not match its checked certificate")
	}
	if !reflect.DeepEqual(r.Nodes, certificate.Nodes) {
		return errors.New("native certificate receipt does not retain the exact checked nodes")
	}
	if !slices.IsSorted(r.Axioms) || len(slices.Compact(append([]string(nil), r.Axioms...))) != len(r.Axioms) {
		return errors.New("native certificate receipt axioms must be sorted and unique")
	}
	return nil
}

func firstOrderViewDigest(view protocolchecker.FirstOrderView) (string, error) {
	encoded, err := view.CanonicalJSON()
	if err != nil {
		return "", err
	}
	return digest(encoded), nil
}

func digest(value []byte) string {
	hash := sha256.Sum256(value)
	return fmt.Sprintf("sha256:%x", hash)
}

func decodeStrict(input io.Reader, limit int64, value any) error {
	if limit <= 0 {
		return errors.New("positive decode limit is required")
	}
	encoded, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil {
		return err
	}
	if int64(len(encoded)) > limit {
		return fmt.Errorf("input exceeds %d-byte limit", limit)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("input must contain exactly one JSON value")
	}
	return nil
}
