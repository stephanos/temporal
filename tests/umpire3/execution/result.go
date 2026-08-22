package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	checkertrace "go.temporal.io/server/tests/umpire3/checker/trace"
	evidencegraph "go.temporal.io/server/tests/umpire3/execution/evidence"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

func finalizeAssurance(result *Result) {
	result.DeriveAssurance()
}

func (r *Result) DeriveAssurance() {
	r.TrustBadge = protocolcatalog.TrustBadgeTestedInstance
	switch r.Claim.Kind {
	case ClaimConforming:
		r.ResultClass = protocolcatalog.ResultClassImplementationConforming
	case ClaimViolating:
		r.ResultClass = protocolcatalog.ResultClassTraceWitness
	default:
		r.ResultClass = protocolcatalog.ResultClassUnknown
	}
}

func (r Result) ValidateAssurance() error {
	expected := Result{Claim: r.Claim}
	finalizeAssurance(&expected)
	if r.ResultClass != expected.ResultClass || r.TrustBadge != expected.TrustBadge {
		return fmt.Errorf("runtime assurance %q/%q does not match final claim %q",
			r.ResultClass, r.TrustBadge, r.Claim.Kind)
	}
	if r.Claim.Kind == ClaimViolating {
		if r.Trace == nil {
			return errors.New("violating runtime result requires a canonical semantic trace")
		}
		if err := checkertrace.Validate(*r.Trace); err != nil {
			return fmt.Errorf("validate violating runtime semantic trace: %w", err)
		}
		if r.Trace.Kind != protocolchecker.SemanticTraceLive ||
			r.Trace.Producer != protocolchecker.SemanticTraceProducerLive ||
			r.Trace.ExperimentDigest != r.ExperimentDigest ||
			string(r.Trace.Property) != r.Claim.Property {
			return errors.New("violating runtime semantic trace does not match its result")
		}
	} else if r.Trace != nil {
		return errors.New("non-violating runtime result cannot carry a semantic trace")
	}
	return nil
}

func (r Result) ValidateEvidenceDigest() error {
	encoded, err := r.canonicalEvidence()
	if err != nil {
		return fmt.Errorf("encode runtime evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	expected := "sha256:" + hex.EncodeToString(digest[:])
	if r.EvidenceDigest != expected {
		return fmt.Errorf("runtime evidence digest %q does not match %q", r.EvidenceDigest, expected)
	}
	return nil
}

func (r *Result) BindEvidenceDigest() error {
	encoded, err := r.canonicalEvidence()
	if err != nil {
		return fmt.Errorf("encode runtime evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	r.EvidenceDigest = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func (r *Result) NormalizeEvidence(maxBytes int64) error {
	if maxBytes <= 0 {
		return errors.New("positive evidence byte limit is required")
	}
	claim := r.Claim.Kind
	finalizeEvidenceGraph(r, maxBytes)
	if r.Claim.Kind != claim {
		return errors.New(r.Claim.Reason)
	}
	if err := r.Evidence.Validate(); err != nil {
		return err
	}
	return r.ValidateEvidenceDigest()
}

func (r Result) canonicalEvidence() ([]byte, error) {
	if _, err := r.Evidence.CanonicalJSON(); err != nil {
		return nil, err
	}
	factIdentifiers := make(map[string]struct{}, len(r.Facts))
	for _, fact := range r.Facts {
		if err := fact.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := factIdentifiers[fact.Identifier]; duplicate {
			return nil, fmt.Errorf("duplicate runtime fact %q", fact.Identifier)
		}
		factIdentifiers[fact.Identifier] = struct{}{}
	}
	if len(r.Facts) != 0 {
		for _, interpreted := range r.Observations {
			if len(interpreted.SupportingFacts) == 0 {
				return nil, fmt.Errorf("observation %q has no supporting facts", interpreted.CheckpointID)
			}
			for _, identifier := range interpreted.SupportingFacts {
				if _, exists := factIdentifiers[identifier]; !exists {
					return nil, fmt.Errorf("observation %q references missing supporting fact %q",
						interpreted.CheckpointID, identifier)
				}
			}
		}
	}
	encoded, err := json.Marshal(struct {
		Facts        []observation.Fact             `json:"facts"`
		Actions      []ActionResult                 `json:"actions"`
		Observations []Observation                  `json:"observations"`
		Graph        evidencegraph.Graph            `json:"graph"`
		Trace        *protocolchecker.SemanticTrace `json:"trace,omitempty"`
	}{
		Facts: r.Facts, Actions: r.Actions, Observations: r.Observations,
		Graph: r.Evidence, Trace: r.Trace,
	})
	if err != nil {
		return nil, fmt.Errorf("encode runtime evidence: %w", err)
	}
	return encoded, nil
}
