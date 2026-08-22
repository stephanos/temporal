package release

import (
	"errors"
	"fmt"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolmonitor "go.temporal.io/server/tests/umpire3/protocol/monitor"
	protocolrelease "go.temporal.io/server/tests/umpire3/protocol/release"
)

func ValidateArtifactBindingsAgainstCurrent(manifest protocolrelease.ReleaseManifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if manifest.LeanVersion != catalog.LeanVersion {
		return fmt.Errorf("release Lean version %q does not match current %q", manifest.LeanVersion, catalog.LeanVersion)
	}
	monitors, err := protocolmonitor.DefaultMonitorCatalog()
	if err != nil {
		return err
	}
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return err
	}
	parity, err := protocolcatalog.DefaultParityLedger()
	if err != nil {
		return err
	}
	checkerCoverage, err := protocolchecker.DefaultCheckerCoverage()
	if err != nil {
		return err
	}
	checkerCoverageJSON, err := checkerCoverage.CanonicalJSON()
	if err != nil {
		return err
	}
	protobuf, err := protocolcatalog.DefaultProtobufInventory()
	if err != nil {
		return err
	}
	proofManifests, err := protocolchecker.DefaultProofManifests()
	if err != nil {
		return err
	}
	for name, values := range map[string][2]string{
		"catalog": {manifest.CatalogHash, catalogHash}, "descriptor": {manifest.DescriptorHash, protobuf.DescriptorDigest},
		"monitor":          {manifest.MonitorSemanticHash, monitors.SemanticHash},
		"composition":      {manifest.CompositionSemanticHash, composition.SemanticHash},
		"parity":           {manifest.ParitySemanticHash, parity.SemanticHash},
		"checker coverage": {manifest.CheckerCoverageHash, releaseDigest(checkerCoverageJSON)},
	} {
		if values[0] != values[1] {
			return fmt.Errorf("release %s hash %q does not match current %q", name, values[0], values[1])
		}
	}
	currentProofManifests := make(map[string]string, len(proofManifests))
	for _, proofManifest := range proofManifests {
		digest, err := proofManifest.Digest()
		if err != nil {
			return err
		}
		currentProofManifests[proofManifest.Identifier] = digest
	}
	if len(manifest.ProofManifests) != len(currentProofManifests) {
		return errors.New("release proof manifests do not match current proof manifests")
	}
	for _, proofManifest := range manifest.ProofManifests {
		currentDigest, exists := currentProofManifests[proofManifest.Identifier]
		if !exists || proofManifest.Digest != currentDigest {
			return fmt.Errorf("release proof manifest %q does not match current artifact", proofManifest.Identifier)
		}
	}
	if manifest.Status == "qualified" {
		if err := validateQualifiedComposition(composition); err != nil {
			return err
		}
		if err := validateQualifiedParity(parity); err != nil {
			return err
		}
	}
	return nil
}

func validateQualifiedComposition(composition protocolcatalog.Composition) error {
	if composition.ResultClass != protocolcatalog.ResultClassCompositionProved {
		return errors.New("qualified release composition evidence is not proof-backed")
	}
	if len(composition.MissingMetadata()) != 0 {
		return errors.New("qualified release composition has missing metadata")
	}
	return nil
}

func validateQualifiedParity(parity protocolcatalog.ParityLedger) error {
	if parity.ResultClass != protocolcatalog.ResultClassEvidenceResolved {
		return errors.New("qualified release parity evidence is not declaration-resolved")
	}
	for _, entry := range parity.Entries {
		if entry.EvidenceStatus != protocolcatalog.MetadataPresent || entry.Disposition == protocolcatalog.ParityNotYetImplemented ||
			entry.Fidelity == protocolcatalog.FidelityPartial || entry.Fidelity == protocolcatalog.FidelityInventoryOnly ||
			(entry.EvidenceLevel != protocolcatalog.EvidenceLocalIntegration && entry.EvidenceLevel != protocolcatalog.EvidenceProfileQualified) {
			return fmt.Errorf("qualified release parity entry %q is incomplete", entry.LegacyName)
		}
	}
	return nil
}
