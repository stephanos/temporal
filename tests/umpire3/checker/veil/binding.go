package veil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

const BindingFormatVersion = "umpire3/veil-binding/v1"

type SMTTrustMode string

const (
	ReconstructedSMT SMTTrustMode = "reconstructed"
	TrustedSMT       SMTTrustMode = "trusted"
)

type NameBinding struct {
	Identifier        string `json:"identifier"`
	BackendIdentifier string `json:"backendIdentifier"`
}

type EnumBinding struct {
	Identifier        string        `json:"identifier"`
	BackendIdentifier string        `json:"backendIdentifier"`
	Values            []NameBinding `json:"values"`
}

type SemanticBinding struct {
	Declaration string                     `json:"declaration"`
	Axioms      []string                   `json:"axioms"`
	TrustBadge  protocolcatalog.TrustBadge `json:"trustBadge"`
}

type CompiledBinding struct {
	View               protocolchecker.FirstOrderView `json:"view"`
	SemanticBinding    SemanticBinding                `json:"semanticBinding"`
	ModuleName         string                         `json:"moduleName"`
	ConcreteModuleName string                         `json:"concreteModuleName"`
	TrustMode          SMTTrustMode                   `json:"trustMode"`
	ActionLabels       []protocolchecker.TraceSource  `json:"actionLabels"`
	FieldLabels        []NameBinding                  `json:"fieldLabels"`
	EnumLabels         []EnumBinding                  `json:"enumLabels"`
	PropertyLabel      string                         `json:"propertyLabel"`
}

type BindingArtifact struct {
	FormatVersion   string          `json:"formatVersion"`
	BackendRevision string          `json:"backendRevision"`
	SourceDigest    string          `json:"sourceDigest"`
	ArtifactDigest  string          `json:"artifactDigest"`
	Binding         CompiledBinding `json:"binding"`
}

func DecodeBindingArtifact(reader io.Reader, limit int64) (BindingArtifact, error) {
	var artifact BindingArtifact
	if err := decodeStrictJSON(reader, limit, "Veil binding artifact", &artifact); err != nil {
		return BindingArtifact{}, err
	}
	if artifact.ArtifactDigest == "derived" {
		if err := artifact.DeriveArtifactDigest(); err != nil {
			return BindingArtifact{}, err
		}
	}
	if err := artifact.Validate(); err != nil {
		return BindingArtifact{}, err
	}
	return artifact, nil
}

func (a BindingArtifact) CanonicalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(a)
}

func (a *BindingArtifact) DeriveArtifactDigest() error {
	if a.ArtifactDigest != "derived" {
		return errors.New("veil binding artifact digest must be derived")
	}
	if err := a.validateIdentity(); err != nil {
		return err
	}
	digest, err := a.computedArtifactDigest()
	if err != nil {
		return err
	}
	a.ArtifactDigest = digest
	return nil
}

func (a BindingArtifact) Validate() error {
	if err := a.validateIdentity(); err != nil {
		return err
	}
	if !validBindingDigest(a.ArtifactDigest) {
		return errors.New("veil binding requires a valid artifact digest")
	}
	expected, err := a.computedArtifactDigest()
	if err != nil {
		return err
	}
	if a.ArtifactDigest != expected {
		return errors.New("veil binding artifact digest does not match its contents")
	}
	return nil
}

func (a BindingArtifact) ValidateAgainst(view protocolchecker.FirstOrderView) error {
	if err := a.Validate(); err != nil {
		return err
	}
	expected, err := view.CanonicalJSON()
	if err != nil {
		return err
	}
	actual, err := a.Binding.View.CanonicalJSON()
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return errors.New("veil binding does not match the first-order view")
	}
	return nil
}

func (a BindingArtifact) validateIdentity() error {
	if a.FormatVersion != BindingFormatVersion {
		return fmt.Errorf("unsupported Veil binding format version %q", a.FormatVersion)
	}
	if a.BackendRevision != protocolchecker.VeilBackendRevision {
		return fmt.Errorf("unsupported Veil backend revision %q", a.BackendRevision)
	}
	if !validBindingDigest(a.SourceDigest) {
		return errors.New("veil binding requires a valid source digest")
	}
	return a.Binding.Validate()
}

func (a BindingArtifact) computedArtifactDigest() (string, error) {
	canonical := a
	canonical.ArtifactDigest = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode Veil binding digest input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (b CompiledBinding) Validate() error {
	if err := b.View.Validate(); err != nil {
		return fmt.Errorf("validate Veil binding first-order view: %w", err)
	}
	if err := b.SemanticBinding.Validate(); err != nil {
		return err
	}
	if b.ModuleName == "" || b.ConcreteModuleName == "" || b.PropertyLabel == "" {
		return errors.New("veil binding requires interactive, concrete, and property declarations")
	}
	if b.TrustMode != ReconstructedSMT && b.TrustMode != TrustedSMT {
		return fmt.Errorf("unsupported Veil binding trust mode %q", b.TrustMode)
	}
	if len(b.ActionLabels) != len(b.View.Actions) {
		return fmt.Errorf("veil binding has %d action labels; expected %d",
			len(b.ActionLabels), len(b.View.Actions))
	}
	backendActions := make([]string, 0, len(b.ActionLabels))
	for index, label := range b.ActionLabels {
		expected := protocolcatalog.ActionKind(b.View.Actions[index].Identifier)
		if label.Action != expected {
			return fmt.Errorf("veil action label %d maps %q; expected %q", index, label.Action, expected)
		}
		if label.BackendAction == "" || slices.Contains(backendActions, label.BackendAction) {
			return fmt.Errorf("veil action label %d has an empty or duplicate backend declaration", index)
		}
		backendActions = append(backendActions, label.BackendAction)
	}
	if len(b.FieldLabels) != len(b.View.StateFields) {
		return fmt.Errorf("veil binding has %d field labels; expected %d",
			len(b.FieldLabels), len(b.View.StateFields))
	}
	if err := validateNameBindings(b.FieldLabels, func(index int) string {
		return b.View.StateFields[index].Identifier
	}, "field"); err != nil {
		return err
	}
	enumSorts := make([]protocolchecker.FirstOrderSort, 0, len(b.View.Sorts))
	for _, sort := range b.View.Sorts {
		if sort.Kind == protocolchecker.FirstOrderSortEnum {
			enumSorts = append(enumSorts, sort)
		}
	}
	if len(b.EnumLabels) != len(enumSorts) {
		return fmt.Errorf("veil binding has %d enum labels; expected %d", len(b.EnumLabels), len(enumSorts))
	}
	backendEnums := make([]string, 0, len(b.EnumLabels))
	for index, enum := range b.EnumLabels {
		expected := enumSorts[index]
		if enum.Identifier != expected.Identifier {
			return fmt.Errorf("veil enum label %d maps %q; expected %q",
				index, enum.Identifier, expected.Identifier)
		}
		if enum.BackendIdentifier == "" || slices.Contains(backendEnums, enum.BackendIdentifier) {
			return fmt.Errorf("veil enum label %d has an empty or duplicate backend declaration", index)
		}
		backendEnums = append(backendEnums, enum.BackendIdentifier)
		if len(enum.Values) != len(expected.Values) {
			return fmt.Errorf("veil enum %q has %d values; expected %d",
				enum.Identifier, len(enum.Values), len(expected.Values))
		}
		if err := validateNameBindings(enum.Values, func(valueIndex int) string {
			return expected.Values[valueIndex]
		}, "enum value"); err != nil {
			return err
		}
	}
	return nil
}

func (b SemanticBinding) Validate() error {
	if b.Declaration == "" {
		return errors.New("veil semantic binding declaration is required")
	}
	if b.Axioms == nil {
		return errors.New("veil semantic binding axiom inventory is required")
	}
	seen := make(map[string]struct{}, len(b.Axioms))
	for _, axiom := range b.Axioms {
		if axiom == "" || axiom == "sorryAx" || axiom == "Lean.ofReduceBool" {
			return fmt.Errorf("veil semantic binding has invalid axiom %q", axiom)
		}
		if _, duplicate := seen[axiom]; duplicate {
			return fmt.Errorf("veil semantic binding has duplicate axiom %q", axiom)
		}
		seen[axiom] = struct{}{}
	}
	expectedTrust := protocolcatalog.TrustBadgeKernel
	if len(b.Axioms) != 0 {
		expectedTrust = protocolcatalog.TrustBadgeKernelWithDeclaredAxioms
	}
	if b.TrustBadge != expectedTrust {
		return errors.New("veil semantic binding trust badge does not match its axiom inventory")
	}
	return nil
}

func (b CompiledBinding) equal(other CompiledBinding) bool {
	left, leftErr := json.Marshal(b)
	right, rightErr := json.Marshal(other)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func validateNameBindings(bindings []NameBinding, expected func(int) string, kind string) error {
	backendIdentifiers := make([]string, 0, len(bindings))
	for index, binding := range bindings {
		identifier := expected(index)
		if binding.Identifier != identifier {
			return fmt.Errorf("veil %s label %d maps %q; expected %q",
				kind, index, binding.Identifier, identifier)
		}
		if binding.BackendIdentifier == "" || slices.Contains(backendIdentifiers, binding.BackendIdentifier) {
			return fmt.Errorf("veil %s label %d has an empty or duplicate backend declaration", kind, index)
		}
		backendIdentifiers = append(backendIdentifiers, binding.BackendIdentifier)
	}
	return nil
}

func validBindingDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
