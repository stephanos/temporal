package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	protocolmonitor "go.temporal.io/server/tests/umpire3/protocol/monitor"
	protocolrelease "go.temporal.io/server/tests/umpire3/protocol/release"
)

func BindArtifactBindings(
	manifest protocolrelease.ReleaseManifest,
	experiments []protocolexperiment.Experiment,
) (protocolrelease.ReleaseManifest, error) {
	if err := manifest.Validate(); err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	if len(experiments) == 0 {
		return protocolrelease.ReleaseManifest{}, errors.New("release binding requires experiments")
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.CatalogHash, err = catalog.Digest()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.LeanVersion = catalog.LeanVersion
	protobuf, err := protocolcatalog.DefaultProtobufInventory()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.DescriptorHash = protobuf.DescriptorDigest
	monitors, err := protocolmonitor.DefaultMonitorCatalog()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.MonitorSemanticHash = monitors.SemanticHash
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.CompositionSemanticHash = composition.SemanticHash
	parity, err := protocolcatalog.DefaultParityLedger()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.ParitySemanticHash = parity.SemanticHash
	checkerCoverage, err := protocolchecker.DefaultCheckerCoverage()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	checkerCoverageJSON, err := checkerCoverage.CanonicalJSON()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.CheckerCoverageHash = releaseDigest(checkerCoverageJSON)

	manifest.Experiments = make(map[string]protocolrelease.ReleaseExperiment, len(experiments))
	for _, experiment := range experiments {
		if err := experiment.Validate(); err != nil {
			return protocolrelease.ReleaseManifest{}, fmt.Errorf("bind experiment %q: %w", experiment.ExperimentID, err)
		}
		if experiment.Model.LeanVersion != catalog.LeanVersion {
			return protocolrelease.ReleaseManifest{}, fmt.Errorf("bind experiment %q: Lean version %q does not match catalog %q",
				experiment.ExperimentID, experiment.Model.LeanVersion, catalog.LeanVersion)
		}
		if _, duplicate := manifest.Experiments[experiment.ExperimentID]; duplicate {
			return protocolrelease.ReleaseManifest{}, fmt.Errorf("bind duplicate experiment %q", experiment.ExperimentID)
		}
		digest, digestErr := experiment.Digest()
		if digestErr != nil {
			return protocolrelease.ReleaseManifest{}, fmt.Errorf("digest experiment %q: %w", experiment.ExperimentID, digestErr)
		}
		manifest.Experiments[experiment.ExperimentID] = protocolrelease.ReleaseExperiment{
			SemanticHash: experiment.Model.SemanticHash,
			Digest:       digest,
		}
	}

	proofManifests, err := protocolchecker.DefaultProofManifests()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	manifest.ProofManifests = make([]protocolrelease.ReleaseProofManifest, len(proofManifests))
	for index, proofManifest := range proofManifests {
		if proofManifest.LeanVersion != catalog.LeanVersion {
			return protocolrelease.ReleaseManifest{}, fmt.Errorf("proof manifest %q Lean version %q does not match catalog %q",
				proofManifest.Identifier, proofManifest.LeanVersion, catalog.LeanVersion)
		}
		digest, digestErr := proofManifest.Digest()
		if digestErr != nil {
			return protocolrelease.ReleaseManifest{}, digestErr
		}
		manifest.ProofManifests[index] = protocolrelease.ReleaseProofManifest{
			Identifier: proofManifest.Identifier,
			Digest:     digest,
		}
	}
	slices.SortFunc(manifest.ProofManifests, func(left, right protocolrelease.ReleaseProofManifest) int {
		return strings.Compare(left.Identifier, right.Identifier)
	})
	if err := manifest.Validate(); err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	return manifest, nil
}

func releaseDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}
