package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
)

const ParityFormatVersion = "umpire3/parity-ledger/v2"

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
	Proof           string `json:"proof"`
	Executable      string `json:"executable"`
	Monitor         string `json:"monitor"`
	NegativeControl string `json:"negativeControl"`
}

type ParityEntry struct {
	Category           ParityCategory    `json:"category"`
	LegacyName         string            `json:"legacyName"`
	SemanticIdentifier string            `json:"semanticIdentifier"`
	Disposition        ParityDisposition `json:"disposition"`
	Fidelity           Fidelity          `json:"fidelity"`
	EvidenceLevel      EvidenceLevel     `json:"evidenceLevel"`
	ExplorationStatus  string            `json:"explorationStatus"`
	Owner              string            `json:"owner"`
	Evidence           ParityEvidence    `json:"evidence"`
}

type ParityLedger struct {
	FormatVersion string        `json:"formatVersion"`
	SemanticHash  string        `json:"semanticHash"`
	CatalogHash   string        `json:"catalogHash"`
	Entries       []ParityEntry `json:"entries"`
}

//go:embed generated/parity-ledger.json
var defaultParityLedgerJSON []byte

func DecodeParityLedger(encoded []byte) (ParityLedger, error) {
	var ledger ParityLedger
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "parity ledger", &ledger); err != nil {
		return ParityLedger{}, err
	}
	if err := ledger.Validate(); err != nil {
		return ParityLedger{}, err
	}
	return ledger, nil
}

func DefaultParityLedger() (ParityLedger, error) {
	return DecodeParityLedger(defaultParityLedgerJSON)
}

func (l ParityLedger) Validate() error {
	if l.FormatVersion != ParityFormatVersion || !validHash(l.SemanticHash) || len(l.Entries) == 0 {
		return errors.New("complete parity ledger provenance and entries are required")
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
	for _, entry := range l.Entries {
		if entry.LegacyName == "" || entry.SemanticIdentifier == "" || entry.Owner == "" {
			return errors.New("every parity entry requires legacy, semantic, and owner identity")
		}
		if entry.ExplorationStatus != "complete" && entry.ExplorationStatus != "incomplete" &&
			entry.ExplorationStatus != "resource-limited" {
			return fmt.Errorf("parity entry %q has unknown exploration status %q", entry.LegacyName, entry.ExplorationStatus)
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
			if entry.ExplorationStatus != "complete" {
				return fmt.Errorf("parity entry %q claims equivalence with incomplete exploration", entry.LegacyName)
			}
			if entry.Evidence.Proof == "" || entry.Evidence.Executable == "" ||
				entry.Evidence.Monitor == "" || entry.Evidence.NegativeControl == "" {
				return fmt.Errorf("parity entry %q requires complete evidence", entry.LegacyName)
			}
		case ParityIntentionallyUnsupported, ParityNotYetImplemented:
			if entry.Fidelity != FidelityPartial && entry.Fidelity != FidelityInventoryOnly {
				return fmt.Errorf("incomplete parity entry %q has fidelity %q", entry.LegacyName, entry.Fidelity)
			}
			if entry.ExplorationStatus == "complete" {
				return fmt.Errorf("incomplete parity entry %q claims complete exploration", entry.LegacyName)
			}
		default:
			return fmt.Errorf("unknown parity disposition %q", entry.Disposition)
		}
	}
	return nil
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
