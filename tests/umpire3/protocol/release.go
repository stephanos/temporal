package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

const ReleaseFormatVersion = "umpire3/release/v2"

type ReleaseEvidence struct {
	Goal    string   `json:"goal"`
	Status  string   `json:"status"`
	Anchors []string `json:"anchors"`
}

type ReleaseMigration struct {
	FormatVersion           string `json:"formatVersion"`
	LedgerHash              string `json:"ledgerHash"`
	BehaviorCount           int    `json:"behaviorCount"`
	ExactCount              int    `json:"exactCount"`
	SemanticEquivalentCount int    `json:"semanticEquivalentCount"`
	PartialCount            int    `json:"partialCount"`
	InventoryOnlyCount      int    `json:"inventoryOnlyCount"`
}

type ExternalQualification struct {
	Profile string `json:"profile"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

type ReleaseProofManifest struct {
	Identifier string `json:"identifier"`
	Digest     string `json:"digest"`
}

type ReleaseManifest struct {
	Release                 string                  `json:"release"`
	Status                  string                  `json:"status"`
	FormatVersion           string                  `json:"formatVersion"`
	ExperimentFormatVersion string                  `json:"experimentFormatVersion"`
	LeanVersion             string                  `json:"leanVersion"`
	CatalogHash             string                  `json:"catalogHash"`
	DescriptorHash          string                  `json:"descriptorHash"`
	MonitorSemanticHash     string                  `json:"monitorSemanticHash"`
	CompositionSemanticHash string                  `json:"compositionSemanticHash"`
	ParitySemanticHash      string                  `json:"paritySemanticHash"`
	Experiments             map[string]string       `json:"experiments"`
	ProofManifests          []ReleaseProofManifest  `json:"proofManifests"`
	Profiles                []string                `json:"profiles"`
	Migration               ReleaseMigration        `json:"migration"`
	Evidence                []ReleaseEvidence       `json:"evidence"`
	Documents               map[string]string       `json:"documents"`
	ExternalQualifications  []ExternalQualification `json:"externalQualifications"`
}

func DecodeReleaseManifest(encoded []byte) (ReleaseManifest, error) {
	var manifest ReleaseManifest
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "release manifest", &manifest); err != nil {
		return ReleaseManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return ReleaseManifest{}, err
	}
	return manifest, nil
}

func (m ReleaseManifest) Validate() error {
	if m.Release == "" || m.FormatVersion != ReleaseFormatVersion ||
		m.ExperimentFormatVersion != FormatVersion || m.LeanVersion == "" {
		return errors.New("complete release identity and format versions are required")
	}
	if m.Status != "candidate" && m.Status != "qualified" {
		return fmt.Errorf("unknown release status %q", m.Status)
	}
	for name, hash := range map[string]string{
		"catalog": m.CatalogHash, "descriptor": m.DescriptorHash, "monitor": m.MonitorSemanticHash,
		"composition": m.CompositionSemanticHash, "parity": m.ParitySemanticHash,
	} {
		if !validHash(hash) {
			return fmt.Errorf("release %s hash is invalid", name)
		}
	}
	if len(m.Experiments) == 0 || len(m.ProofManifests) == 0 {
		return errors.New("release experiments and proof manifests are required")
	}
	for identifier, hash := range m.Experiments {
		if identifier == "" || !validHash(hash) {
			return errors.New("release experiment identity and semantic hash are required")
		}
	}
	proofManifestIdentifiers := make(map[string]struct{}, len(m.ProofManifests))
	for _, manifest := range m.ProofManifests {
		if manifest.Identifier == "" {
			return errors.New("release proof manifest identity is required")
		}
		if !validHash(manifest.Digest) {
			return fmt.Errorf("release proof manifest digest for %q is invalid", manifest.Identifier)
		}
		if _, duplicate := proofManifestIdentifiers[manifest.Identifier]; duplicate {
			return fmt.Errorf("release contains duplicate proof manifest %q", manifest.Identifier)
		}
		proofManifestIdentifiers[manifest.Identifier] = struct{}{}
	}
	requiredProfiles := []string{
		"local-in-process", "ci-test-cluster", "remote-deployment", "grpc-only-black-box", "production-canary",
	}
	profiles := append([]string(nil), m.Profiles...)
	slices.Sort(profiles)
	expectedProfiles := append([]string(nil), requiredProfiles...)
	slices.Sort(expectedProfiles)
	if !slices.Equal(profiles, expectedProfiles) {
		return fmt.Errorf("release profiles %v do not cover %v", profiles, expectedProfiles)
	}
	if m.Migration.FormatVersion != "umpire3/migration-ledger/v3" ||
		!validHash(m.Migration.LedgerHash) || m.Migration.BehaviorCount <= 0 {
		return errors.New("complete root-test migration evidence is required")
	}
	if m.Migration.ExactCount < 0 || m.Migration.SemanticEquivalentCount < 0 ||
		m.Migration.PartialCount < 0 || m.Migration.InventoryOnlyCount < 0 ||
		m.Migration.ExactCount+m.Migration.SemanticEquivalentCount+
			m.Migration.PartialCount+m.Migration.InventoryOnlyCount != m.Migration.BehaviorCount {
		return errors.New("release migration fidelity counts do not match the behavior count")
	}
	requiredGoals := map[string]struct{}{
		"single-semantic-model": {}, "known-regression-verification": {}, "deterministic-plans": {},
		"portable-profiles": {}, "white-box-black-box": {}, "developer-authoring": {},
		"unknown-bug-exploration": {}, "first-class-faults": {}, "non-linear-identity": {},
		"programmable-participants": {}, "clock-skew-safety": {}, "guided-exploration": {},
		"coverage-guided-fuzzing": {},
	}
	seenGoals := make(map[string]struct{}, len(m.Evidence))
	for _, evidence := range m.Evidence {
		if _, required := requiredGoals[evidence.Goal]; !required {
			return fmt.Errorf("release contains unknown vision goal %q", evidence.Goal)
		}
		if _, duplicate := seenGoals[evidence.Goal]; duplicate {
			return fmt.Errorf("release contains duplicate vision goal %q", evidence.Goal)
		}
		seenGoals[evidence.Goal] = struct{}{}
		if evidence.Status != "passed" && evidence.Status != "partial" {
			return fmt.Errorf("release vision goal %q has unknown evidence status %q", evidence.Goal, evidence.Status)
		}
		if len(evidence.Anchors) == 0 {
			return fmt.Errorf("release vision goal %q lacks evidence anchors", evidence.Goal)
		}
		if m.Status == "qualified" && evidence.Status != "passed" {
			return fmt.Errorf("qualified release vision goal %q lacks passing evidence", evidence.Goal)
		}
		for _, anchor := range evidence.Anchors {
			if anchor == "" {
				return fmt.Errorf("release vision goal %q has an empty evidence anchor", evidence.Goal)
			}
		}
	}
	if len(seenGoals) != len(requiredGoals) {
		return errors.New("release does not disposition every Umpire vision goal")
	}
	for _, name := range []string{"support", "authoring", "modeling", "operations", "security", "incident-recovery"} {
		if m.Documents[name] == "" {
			return fmt.Errorf("release document %q is required", name)
		}
	}
	if m.Status == "qualified" && len(m.ExternalQualifications) != 0 {
		return errors.New("qualified release cannot retain external qualification gates")
	}
	for _, qualification := range m.ExternalQualifications {
		if qualification.Profile == "" || qualification.Command == "" || qualification.Status != "required" {
			return errors.New("external qualifications require a profile, command, and required status")
		}
	}
	if m.Status == "qualified" && (m.Migration.PartialCount != 0 || m.Migration.InventoryOnlyCount != 0) {
		return errors.New("qualified release migration behaviors remain partial")
	}
	return nil
}

func (m ReleaseManifest) ValidateAgainstCurrent() error {
	if err := m.Validate(); err != nil {
		return err
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	monitors, err := DefaultMonitorCatalog()
	if err != nil {
		return err
	}
	composition, err := DefaultComposition()
	if err != nil {
		return err
	}
	parity, err := DefaultParityLedger()
	if err != nil {
		return err
	}
	protobuf, err := DefaultProtobufInventory()
	if err != nil {
		return err
	}
	proofManifests, err := DefaultProofManifests()
	if err != nil {
		return err
	}
	for name, values := range map[string][2]string{
		"catalog": {m.CatalogHash, catalogHash}, "descriptor": {m.DescriptorHash, protobuf.DescriptorDigest},
		"monitor":     {m.MonitorSemanticHash, monitors.SemanticHash},
		"composition": {m.CompositionSemanticHash, composition.SemanticHash},
		"parity":      {m.ParitySemanticHash, parity.SemanticHash},
	} {
		if values[0] != values[1] {
			return fmt.Errorf("release %s hash %q does not match current %q", name, values[0], values[1])
		}
	}
	currentProofManifests := make(map[string]string, len(proofManifests))
	for _, manifest := range proofManifests {
		digest, err := manifest.Digest()
		if err != nil {
			return err
		}
		currentProofManifests[manifest.Identifier] = digest
	}
	if len(m.ProofManifests) != len(currentProofManifests) {
		return errors.New("release proof manifests do not match current proof manifests")
	}
	for _, manifest := range m.ProofManifests {
		currentDigest, exists := currentProofManifests[manifest.Identifier]
		if !exists || manifest.Digest != currentDigest {
			return fmt.Errorf("release proof manifest %q does not match current artifact", manifest.Identifier)
		}
	}
	if m.Status == "qualified" {
		if err := validateQualifiedComposition(composition); err != nil {
			return err
		}
		if err := validateQualifiedParity(parity); err != nil {
			return err
		}
	}
	return nil
}

func validateQualifiedComposition(composition Composition) error {
	if composition.ResultClass != ResultClassCompositionProved {
		return errors.New("qualified release composition evidence is not proof-backed")
	}
	if len(composition.MissingMetadata()) != 0 {
		return errors.New("qualified release composition has missing metadata")
	}
	return nil
}

func validateQualifiedParity(parity ParityLedger) error {
	if parity.ResultClass != ResultClassEvidenceResolved {
		return errors.New("qualified release parity evidence is not declaration-resolved")
	}
	for _, entry := range parity.Entries {
		if entry.EvidenceStatus != MetadataPresent || entry.Disposition == ParityNotYetImplemented ||
			entry.Fidelity == FidelityPartial || entry.Fidelity == FidelityInventoryOnly ||
			entry.EvidenceLevel != EvidenceProfileQualified {
			return fmt.Errorf("qualified release parity entry %q is incomplete", entry.LegacyName)
		}
	}
	return nil
}
