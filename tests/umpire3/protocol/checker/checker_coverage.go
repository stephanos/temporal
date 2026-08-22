package checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol/internal/generated"
)

const CheckerCoverageFormatVersion = "umpire3/checker-coverage/v1"

type CheckerKind string

const (
	CheckerExact  CheckerKind = "exact"
	CheckerNative CheckerKind = "native"
	CheckerVeil   CheckerKind = "veil"
)

type CheckerCoverageStatus string

const (
	CheckerCoverageChecked      CheckerCoverageStatus = "checked"
	CheckerCoverageNotSupported CheckerCoverageStatus = "not-supported"
)

type CheckerEvidence struct {
	Kind        string `json:"kind"`
	Digest      string `json:"digest"`
	Declaration string `json:"declaration,omitempty"`
}

type CheckerClaim struct {
	Job         string        `json:"job"`
	ResultClass ResultClass   `json:"resultClass"`
	TrustBadge  TrustBadge    `json:"trustBadge"`
	Exact       bool          `json:"exact"`
	Bounds      BackendBounds `json:"bounds"`
	Omissions   []string      `json:"omissions"`
}

type CheckerCoverageEntry struct {
	Target       TargetID              `json:"target"`
	Property     PropertyID            `json:"property"`
	Checker      CheckerKind           `json:"checker"`
	Status       CheckerCoverageStatus `json:"status"`
	World        string                `json:"world,omitempty"`
	Variant      string                `json:"variant,omitempty"`
	SemanticHash string                `json:"semanticHash,omitempty"`
	Claims       []CheckerClaim        `json:"claims"`
	Evidence     []CheckerEvidence     `json:"evidence"`
	Reason       string                `json:"reason,omitempty"`
}

type CheckerCoverageManifest struct {
	FormatVersion           string                 `json:"formatVersion"`
	CatalogHash             string                 `json:"catalogHash"`
	CompositionSemanticHash string                 `json:"compositionSemanticHash"`
	Entries                 []CheckerCoverageEntry `json:"entries"`
}

var defaultCheckerCoverageJSON = generated.Read(generated.CheckerCoverage)

func DecodeCheckerCoverage(encoded []byte) (CheckerCoverageManifest, error) {
	var manifest CheckerCoverageManifest
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit,
		"checker coverage manifest", &manifest); err != nil {
		return CheckerCoverageManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return CheckerCoverageManifest{}, err
	}
	return manifest, nil
}

func DefaultCheckerCoverage() (CheckerCoverageManifest, error) {
	return DecodeCheckerCoverage(defaultCheckerCoverageJSON)
}

func (m CheckerCoverageManifest) Validate() error {
	if m.FormatVersion != CheckerCoverageFormatVersion ||
		!validHash(m.CatalogHash) || !validHash(m.CompositionSemanticHash) ||
		len(m.Entries) == 0 {
		return errors.New("complete checker coverage identity, provenance, and entries are required")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	composition, err := DefaultComposition()
	if err != nil {
		return err
	}
	if m.CatalogHash != catalogHash || m.CompositionSemanticHash != composition.SemanticHash {
		return errors.New("checker coverage does not match the current catalog and composition")
	}

	type targetProperty struct {
		target   TargetID
		property PropertyID
	}
	expected := make(map[targetProperty]struct{})
	for _, target := range composition.Targets {
		for _, property := range target.Properties {
			expected[targetProperty{target: target.Identifier, property: property}] = struct{}{}
		}
	}
	if len(m.Entries) != len(expected)*3 {
		return fmt.Errorf("checker coverage has %d entries; expected %d",
			len(m.Entries), len(expected)*3)
	}
	if !slices.IsSortedFunc(m.Entries, compareCheckerCoverageEntry) {
		return errors.New("checker coverage entries must be sorted")
	}
	seen := make(map[string]struct{}, len(m.Entries))
	for index := range m.Entries {
		entry := &m.Entries[index]
		if _, known := expected[targetProperty{target: entry.Target, property: entry.Property}]; !known {
			return fmt.Errorf("checker coverage references unknown target/property %q/%q",
				entry.Target, entry.Property)
		}
		key := string(entry.Target) + "\x00" + string(entry.Property) + "\x00" + string(entry.Checker)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate checker coverage entry %q/%q/%q",
				entry.Target, entry.Property, entry.Checker)
		}
		seen[key] = struct{}{}
		if err := entry.validate(); err != nil {
			return fmt.Errorf("validate checker coverage entry %q/%q/%q: %w",
				entry.Target, entry.Property, entry.Checker, err)
		}
	}
	return nil
}

func (m CheckerCoverageManifest) CanonicalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (m CheckerCoverageManifest) Clone() CheckerCoverageManifest {
	result := m
	result.Entries = append([]CheckerCoverageEntry(nil), m.Entries...)
	for index := range result.Entries {
		entry := &result.Entries[index]
		entry.Claims = append([]CheckerClaim(nil), entry.Claims...)
		for claimIndex := range entry.Claims {
			entry.Claims[claimIndex].Omissions =
				append([]string(nil), entry.Claims[claimIndex].Omissions...)
		}
		entry.Evidence = append([]CheckerEvidence(nil), entry.Evidence...)
	}
	return result
}

func (e CheckerCoverageEntry) validate() error {
	switch e.Checker {
	case CheckerExact, CheckerNative, CheckerVeil:
	default:
		return fmt.Errorf("unknown checker %q", e.Checker)
	}
	switch e.Status {
	case CheckerCoverageNotSupported:
		if e.Checker == CheckerExact {
			return errors.New("exact checking is mandatory")
		}
		if e.Reason == "" || e.World != "" || e.Variant != "" || e.SemanticHash != "" ||
			len(e.Claims) != 0 || len(e.Evidence) != 0 {
			return errors.New("unsupported checker coverage requires only a reason")
		}
		return nil
	case CheckerCoverageChecked:
		if e.Reason != "" || e.World == "" || e.Variant == "" || !validHash(e.SemanticHash) ||
			len(e.Claims) == 0 || len(e.Evidence) == 0 {
			return errors.New("checked coverage requires scope, semantic provenance, claims, and evidence")
		}
	default:
		return fmt.Errorf("unknown checker coverage status %q", e.Status)
	}
	if !slices.IsSortedFunc(e.Claims, func(left, right CheckerClaim) int {
		return compareStrings(left.Job, right.Job)
	}) {
		return errors.New("checker claims must be sorted")
	}
	jobs := make(map[string]struct{}, len(e.Claims))
	for _, claim := range e.Claims {
		if claim.Job == "" || !claim.ResultClass.Valid() || !claim.TrustBadge.Valid() {
			return errors.New("checker claim requires a job, result class, and trust badge")
		}
		if claim.Omissions == nil {
			return fmt.Errorf("checker claim %q requires explicit omissions", claim.Job)
		}
		if _, duplicate := jobs[claim.Job]; duplicate {
			return fmt.Errorf("duplicate checker claim job %q", claim.Job)
		}
		jobs[claim.Job] = struct{}{}
		if !slices.IsSorted(claim.Omissions) ||
			len(slices.Compact(append([]string(nil), claim.Omissions...))) != len(claim.Omissions) {
			return fmt.Errorf("checker claim %q omissions must be sorted and unique", claim.Job)
		}
		for _, omission := range claim.Omissions {
			if omission == "" {
				return fmt.Errorf("checker claim %q has an empty omission", claim.Job)
			}
		}
	}
	if err := validateCheckerClaims(e.Checker, e.Claims); err != nil {
		return err
	}
	if !slices.IsSortedFunc(e.Evidence, compareCheckerEvidence) {
		return errors.New("checker evidence must be sorted")
	}
	evidence := make(map[string]struct{}, len(e.Evidence))
	for _, item := range e.Evidence {
		if item.Kind == "" || !validHash(item.Digest) {
			return errors.New("checker evidence requires a kind and digest")
		}
		key := item.Kind + "\x00" + item.Digest
		if _, duplicate := evidence[key]; duplicate {
			return fmt.Errorf("duplicate checker evidence %q", item.Kind)
		}
		evidence[key] = struct{}{}
	}
	return nil
}

func validateCheckerClaims(checker CheckerKind, claims []CheckerClaim) error {
	switch checker {
	case CheckerExact, CheckerNative:
		if len(claims) != 1 || claims[0].Job != "exhaustive" ||
			claims[0].ResultClass != ResultClassFiniteExhaustive ||
			claims[0].TrustBadge != TrustBadgeCheckedCertificate || !claims[0].Exact ||
			claims[0].Bounds != (BackendBounds{}) || len(claims[0].Omissions) != 0 {
			return fmt.Errorf("%s coverage requires one exact checked exhaustive claim", checker)
		}
	case CheckerVeil:
		if len(claims) != 3 {
			return errors.New("veil coverage requires concrete, invariant, and symbolic-trace claims")
		}
		byJob := make(map[string]CheckerClaim, len(claims))
		for _, claim := range claims {
			byJob[claim.Job] = claim
		}
		concrete, concreteFound := byJob[string(BackendJobConcrete)]
		invariant, invariantFound := byJob[string(BackendJobInvariant)]
		symbolic, symbolicFound := byJob[string(BackendJobSymbolicTrace)]
		if !concreteFound || !invariantFound || !symbolicFound {
			return errors.New("veil coverage is missing a required job")
		}
		if concrete.ResultClass != ResultClassExternalNoCounterexample ||
			concrete.TrustBadge != TrustBadgeTestedInstance || concrete.Exact ||
			concrete.Bounds.ConcreteStateLimit <= 0 ||
			!slices.Equal(concrete.Omissions, []string{VeilConcreteCollisionOmission}) {
			return errors.New("veil concrete coverage must retain its tested-instance collision qualification")
		}
		if symbolic.ResultClass != ResultClassBoundedSafe ||
			symbolic.TrustBadge != TrustBadgeTrustedSolver || !symbolic.Exact ||
			symbolic.Bounds.Depth <= 0 || len(symbolic.Omissions) != 0 {
			return errors.New("veil symbolic coverage must retain its bounded trusted-solver class")
		}
		if invariant.ResultClass != ResultClassInvariantProved || !invariant.Exact ||
			(invariant.TrustBadge != TrustBadgeReconstructedSolverProof &&
				invariant.TrustBadge != TrustBadgeTrustedSolver) ||
			invariant.Bounds != (BackendBounds{}) || len(invariant.Omissions) != 0 {
			return errors.New("veil invariant coverage must retain its solver proof trust")
		}
	default:
		return fmt.Errorf("unknown checker %q", checker)
	}
	return nil
}

func compareCheckerCoverageEntry(left, right CheckerCoverageEntry) int {
	if comparison := compareStrings(string(left.Target), string(right.Target)); comparison != 0 {
		return comparison
	}
	if comparison := compareStrings(string(left.Property), string(right.Property)); comparison != 0 {
		return comparison
	}
	return compareStrings(string(left.Checker), string(right.Checker))
}

func compareCheckerEvidence(left, right CheckerEvidence) int {
	if comparison := compareStrings(left.Kind, right.Kind); comparison != 0 {
		return comparison
	}
	if comparison := compareStrings(left.Digest, right.Digest); comparison != 0 {
		return comparison
	}
	return compareStrings(left.Declaration, right.Declaration)
}
