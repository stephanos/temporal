package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tests/umpire3/protocol/internal/generated"
)

const ParityFormatVersion = "umpire3/parity-ledger/v4"

type ParityCategory string

const (
	ParityProperty ParityCategory = "property"
	ParityTarget   ParityCategory = "target"
)

type ParityDisposition string

const (
	ParityEquivalent               ParityDisposition = "equivalent"
	ParityReplaced                 ParityDisposition = "replaced"
	ParityIntentionallyUnsupported ParityDisposition = "intentionally-unsupported"
	ParityNotYetImplemented        ParityDisposition = "not-yet-implemented"
)

type Fidelity string

const (
	FidelityExact              Fidelity = "exact"
	FidelitySemanticEquivalent Fidelity = "semantic-equivalent"
	FidelityPartial            Fidelity = "partial"
	FidelityInventoryOnly      Fidelity = "inventory-only"
)

type EvidenceLevel string

const (
	EvidenceInventory        EvidenceLevel = "inventory"
	EvidenceModelProof       EvidenceLevel = "model-proof"
	EvidenceLocalIntegration EvidenceLevel = "local-integration"
	EvidenceProfileQualified EvidenceLevel = "profile-qualified"
)

type ParityEvidence struct {
	Proof           ResolvedDeclaration `json:"proof"`
	Executable      ResolvedDeclaration `json:"executable"`
	Monitor         ResolvedDeclaration `json:"monitor"`
	NegativeControl ResolvedDeclaration `json:"negativeControl"`
}

type ParityEntry struct {
	Category           ParityCategory    `json:"category"`
	LegacyName         string            `json:"legacyName"`
	SemanticIdentifier string            `json:"semanticIdentifier"`
	Disposition        ParityDisposition `json:"disposition"`
	Fidelity           Fidelity          `json:"fidelity"`
	EvidenceLevel      EvidenceLevel     `json:"evidenceLevel"`
	EvidenceStatus     MetadataStatus    `json:"evidenceStatus"`
	Owner              string            `json:"owner"`
	Evidence           ParityEvidence    `json:"evidence"`
}

type ParityLedger struct {
	FormatVersion    string        `json:"formatVersion"`
	ResultClass      ResultClass   `json:"resultClass"`
	TrustBadge       TrustBadge    `json:"trustBadge"`
	SemanticHash     string        `json:"semanticHash"`
	SourceDigest     string        `json:"sourceDigest"`
	DependencyDigest string        `json:"dependencyDigest"`
	ArtifactDigest   string        `json:"artifactDigest"`
	CatalogHash      string        `json:"catalogHash"`
	Entries          []ParityEntry `json:"entries"`
}

var defaultParityLedgerJSON = generated.Read(generated.ParityLedger)

func DecodeParityLedger(encoded []byte) (ParityLedger, error) {
	var ledger ParityLedger
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "parity ledger", &ledger); err != nil {
		return ParityLedger{}, err
	}
	ledger.derive()
	if err := ledger.Validate(); err != nil {
		return ParityLedger{}, err
	}
	return ledger, nil
}

func DefaultParityLedger() (ParityLedger, error) {
	return DecodeParityLedger(defaultParityLedgerJSON)
}

func (l ParityLedger) Validate() error {
	if l.FormatVersion != ParityFormatVersion || l.ResultClass != ResultClassEvidenceResolved ||
		!validHash(l.SemanticHash) || l.SourceDigest != l.SemanticHash || !validHash(l.DependencyDigest) ||
		len(l.Entries) == 0 {
		return errors.New("resolved parity ledger provenance and entries are required")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if l.CatalogHash != catalogHash {
		return fmt.Errorf("parity catalog hash %q does not match semantic catalog %q", l.CatalogHash, catalogHash)
	}
	properties := make(map[string]struct{}, len(catalog.Properties))
	for _, property := range catalog.Properties {
		properties[property.Identifier] = struct{}{}
	}
	targets := make(map[string]struct{}, len(catalog.Targets))
	for _, target := range catalog.Targets {
		targets[target.Identifier] = struct{}{}
	}
	identities := make(map[string]struct{}, len(l.Entries))
	var declarations []ResolvedDeclaration
	for _, entry := range l.Entries {
		if entry.LegacyName == "" || entry.SemanticIdentifier == "" || entry.Owner == "" {
			return errors.New("every parity entry requires legacy, semantic, and owner identity")
		}
		if entry.EvidenceStatus != MetadataPresent && entry.EvidenceStatus != MetadataMissing {
			return fmt.Errorf("parity entry %q has unknown evidence metadata status %q", entry.LegacyName, entry.EvidenceStatus)
		}
		identity := string(entry.Category) + ":" + entry.LegacyName
		if _, duplicate := identities[identity]; duplicate {
			return fmt.Errorf("duplicate parity entry %q", identity)
		}
		identities[identity] = struct{}{}
		switch entry.Category {
		case ParityProperty:
			if _, known := properties[entry.SemanticIdentifier]; !known {
				return fmt.Errorf("parity entry %q references unknown property %q", entry.LegacyName, entry.SemanticIdentifier)
			}
		case ParityTarget:
			if _, known := targets[entry.SemanticIdentifier]; !known {
				return fmt.Errorf("parity entry %q references unknown target %q", entry.LegacyName, entry.SemanticIdentifier)
			}
		default:
			return fmt.Errorf("unknown parity category %q", entry.Category)
		}
		switch entry.Fidelity {
		case FidelityExact, FidelitySemanticEquivalent, FidelityPartial, FidelityInventoryOnly:
		default:
			return fmt.Errorf("unknown parity fidelity %q", entry.Fidelity)
		}
		switch entry.EvidenceLevel {
		case EvidenceInventory, EvidenceModelProof, EvidenceLocalIntegration, EvidenceProfileQualified:
		default:
			return fmt.Errorf("unknown parity evidence level %q", entry.EvidenceLevel)
		}
		switch entry.Disposition {
		case ParityEquivalent, ParityReplaced:
			if entry.Fidelity != FidelityExact && entry.Fidelity != FidelitySemanticEquivalent {
				return fmt.Errorf("parity entry %q claims equivalence with fidelity %q", entry.LegacyName, entry.Fidelity)
			}
			if entry.EvidenceLevel == EvidenceInventory {
				return fmt.Errorf("parity entry %q claims equivalence with inventory-only evidence", entry.LegacyName)
			}
			if entry.EvidenceStatus != MetadataPresent {
				return fmt.Errorf("parity entry %q claims equivalence with missing evidence metadata", entry.LegacyName)
			}
			entryDeclarations := []ResolvedDeclaration{
				entry.Evidence.Proof, entry.Evidence.Executable,
				entry.Evidence.Monitor, entry.Evidence.NegativeControl,
			}
			for _, declaration := range entryDeclarations {
				if err := declaration.Validate(); err != nil {
					return fmt.Errorf("parity entry %q requires complete evidence: %w", entry.LegacyName, err)
				}
			}
			declarations = append(declarations, entryDeclarations...)
		case ParityIntentionallyUnsupported, ParityNotYetImplemented:
			if entry.Fidelity != FidelityPartial && entry.Fidelity != FidelityInventoryOnly {
				return fmt.Errorf("incomplete parity entry %q has fidelity %q", entry.LegacyName, entry.Fidelity)
			}
			if entry.EvidenceStatus == MetadataPresent {
				return fmt.Errorf("incomplete parity entry %q claims present evidence metadata", entry.LegacyName)
			}
		default:
			return fmt.Errorf("unknown parity disposition %q", entry.Disposition)
		}
	}
	if l.TrustBadge != aggregateTrustBadge(declarations...) {
		return errors.New("parity trust badge does not match its resolved axiom inventories")
	}
	expectedArtifactDigest, err := l.computedArtifactDigest()
	if err != nil {
		return err
	}
	if l.ArtifactDigest != expectedArtifactDigest {
		return errors.New("parity artifact digest does not match its canonical contents")
	}
	return nil
}

func (l *ParityLedger) derive() {
	for entryIndex := range l.Entries {
		evidence := &l.Entries[entryIndex].Evidence
		for _, declaration := range []*ResolvedDeclaration{
			&evidence.Proof, &evidence.Executable, &evidence.Monitor, &evidence.NegativeControl,
		} {
			declaration.derive()
		}
	}
	if l.ArtifactDigest == "derived" {
		digest, err := l.computedArtifactDigest()
		if err == nil {
			l.ArtifactDigest = digest
		}
	}
}

func (l ParityLedger) computedArtifactDigest() (string, error) {
	canonical := l
	canonical.ArtifactDigest = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode parity digest payload: %w", err)
	}
	return digestBytes(encoded), nil
}

func (l ParityLedger) CanonicalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("encode parity ledger: %w", err)
	}
	return encoded, nil
}
