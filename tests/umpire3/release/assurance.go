package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"slices"

	umpire3docs "go.temporal.io/server/tests/umpire3"
	"go.temporal.io/server/tests/umpire3/campaign"
	"go.temporal.io/server/tests/umpire3/clockskew"
	"go.temporal.io/server/tests/umpire3/developerux"
	"go.temporal.io/server/tests/umpire3/migration"
	"go.temporal.io/server/tests/umpire3/mutationaudit"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/resilience"
)

func Bind(
	manifest protocol.ReleaseManifest,
	experiments []protocol.Experiment,
	ledger migration.Ledger,
	ledgerBytes []byte,
) (protocol.ReleaseManifest, error) {
	bound, err := protocol.BindReleaseArtifactBindings(manifest, experiments)
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	bound.Migration, err = summarizeMigration(ledger, ledgerBytes)
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	bound.Assurance, err = deriveAssurance(bound, ledger)
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	if err := bound.ValidateArtifactBindingsAgainstCurrent(); err != nil {
		return protocol.ReleaseManifest{}, err
	}
	return bound, nil
}

func RebindCurrentAssurance(manifest protocol.ReleaseManifest) (protocol.ReleaseManifest, error) {
	ledger, ledgerBytes, err := migration.DefaultLedger()
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	expectedMigration, err := summarizeMigration(ledger, ledgerBytes)
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	if !reflect.DeepEqual(manifest.Migration, expectedMigration) {
		return protocol.ReleaseManifest{}, errors.New("release migration binding does not match the current ledger")
	}
	manifest.Assurance, err = deriveAssurance(manifest, ledger)
	if err != nil {
		return protocol.ReleaseManifest{}, err
	}
	return manifest, nil
}

func ValidateAgainstCurrent(manifest protocol.ReleaseManifest) error {
	if err := manifest.ValidateArtifactBindingsAgainstCurrent(); err != nil {
		return err
	}
	expected, err := RebindCurrentAssurance(manifest)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(manifest.Assurance, expected.Assurance) {
		return errors.New("release assurance graph does not match current typed evidence")
	}
	return nil
}

func summarizeMigration(ledger migration.Ledger, encoded []byte) (protocol.ReleaseMigration, error) {
	if len(encoded) == 0 {
		return protocol.ReleaseMigration{}, errors.New("release binding requires encoded migration evidence")
	}
	decoded, err := migration.DecodeLedger(encoded)
	if err != nil {
		return protocol.ReleaseMigration{}, err
	}
	if !reflect.DeepEqual(ledger, decoded) {
		return protocol.ReleaseMigration{}, errors.New("release migration value does not match its encoded evidence")
	}
	summary := protocol.ReleaseMigration{
		FormatVersion: ledger.FormatVersion,
		BehaviorCount: len(ledger.Entries),
	}
	digest := sha256.Sum256(encoded)
	summary.LedgerHash = "sha256:" + hex.EncodeToString(digest[:])
	for _, entry := range ledger.Entries {
		switch entry.Fidelity {
		case protocol.FidelityExact:
			summary.ExactCount++
		case protocol.FidelitySemanticEquivalent:
			summary.SemanticEquivalentCount++
		case protocol.FidelityPartial:
			summary.PartialCount++
		case protocol.FidelityInventoryOnly:
			summary.InventoryOnlyCount++
		}
	}
	return summary, nil
}

func deriveAssurance(
	manifest protocol.ReleaseManifest,
	ledger migration.Ledger,
) (protocol.ReleaseAssurance, error) {
	if err := validateMigrationEvidence(ledger); err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	composition, err := protocol.DefaultComposition()
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	parity, err := protocol.DefaultParityLedger()
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	proofManifests, err := protocol.DefaultProofManifests()
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	auditExperimentID := "nexus-cancellation-stale-completion-v1"
	auditBinding, exists := manifest.Experiments[auditExperimentID]
	if !exists {
		return protocol.ReleaseAssurance{}, errors.New("release omits the approved mutation audit experiment")
	}
	mutationAudit, err := campaign.DefaultApprovedMutationAuditForBinding(
		auditExperimentID,
		string(protocol.PropertyIDNexusCancellationWonExcludesSuccess),
		auditBinding.Digest,
	)
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	if err := mutationAudit.ValidateCoverageGuidance(); err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("validate coverage-guided mutation audit: %w", err)
	}
	semanticMutationAudit, err := mutationaudit.Default()
	if err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("load semantic mutation audit: %w", err)
	}
	if semanticMutationAudit.CampaignSourceDigest != mutationAudit.SourceDigest {
		return protocol.ReleaseAssurance{}, errors.New("semantic mutation audit does not match the release mutation campaign")
	}
	developerAudit, err := developerux.Run(mutationAudit.PromotionSource)
	if err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("run developer UX audit: %w", err)
	}
	clockSkewAudit, err := clockskew.RunAudit()
	if err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("run clock-skew audit: %w", err)
	}
	documentationAudit, err := umpire3docs.AuditDocumentation()
	if err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("audit documentation: %w", err)
	}
	resilienceAudit, err := resilience.DefaultAudit()
	if err != nil {
		return protocol.ReleaseAssurance{}, fmt.Errorf("load resilience audit: %w", err)
	}
	checkerCoverage, err := protocol.DefaultCheckerCoverage()
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}
	nativeBenchmark, err := nativeBenchmarkDigest(checkerCoverage)
	if err != nil {
		return protocol.ReleaseAssurance{}, err
	}

	nodes := []protocol.ReleaseEvidenceNode{
		node("approved-mutation-audit", mutationAudit.ResultClass,
			mutationAudit.TrustBadge, mutationAudit.ArtifactDigest),
		node("catalog", protocol.ResultClassMetadataValidated, catalogTrust(catalog), manifest.CatalogHash),
		node("checker-coverage", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, manifest.CheckerCoverageHash),
		node("clock-skew-audit", protocol.ResultClassImplementationConforming,
			protocol.TrustBadgeTestedInstance, clockSkewAudit.ArtifactDigest),
		node("composition", composition.ResultClass, composition.TrustBadge, manifest.CompositionSemanticHash),
		node("developer-ux", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, developerAudit.ArtifactDigest),
		node("documentation", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, documentationAudit.ArtifactDigest),
		node("migration", protocol.ResultClassImplementationConforming,
			protocol.TrustBadgeTestedInstance, manifest.Migration.LedgerHash),
		node("monitor-programs", protocol.ResultClassEvidenceResolved,
			protocol.TrustBadgeTestedInstance, manifest.MonitorSemanticHash),
		node("native-scale-benchmark", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, nativeBenchmark),
		node("parity", parity.ResultClass, parity.TrustBadge, manifest.ParitySemanticHash),
		node("protobuf-descriptor", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, manifest.DescriptorHash),
		node("resilience-audit", protocol.ResultClassImplementationConforming,
			protocol.TrustBadgeTestedInstance, resilienceAudit.ArtifactDigest),
		node("semantic-mutation-portfolio", protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, semanticMutationAudit.ArtifactDigest),
	}
	experimentNodes := make([]string, 0, len(manifest.Experiments))
	for identifier, binding := range manifest.Experiments {
		nodeIdentifier := "experiment/" + identifier
		nodes = append(nodes, node(nodeIdentifier, protocol.ResultClassMetadataValidated,
			protocol.TrustBadgeTestedInstance, binding.Digest))
		experimentNodes = append(experimentNodes, nodeIdentifier)
	}
	slices.Sort(experimentNodes)
	for _, proofManifest := range proofManifests {
		digest, digestErr := proofManifest.Digest()
		if digestErr != nil {
			return protocol.ReleaseAssurance{}, digestErr
		}
		nodes = append(nodes, node("proof/"+proofManifest.Identifier,
			proofManifest.ResultClass, proofManifest.TrustBadge, digest))
	}
	profileNodes := make(map[string]string, len(manifest.Qualifications))
	for _, qualification := range manifest.Qualifications {
		identifier := "profile/" + qualification.Profile
		nodes = append(nodes, node(identifier, protocol.ResultClassImplementationConforming,
			protocol.TrustBadgeTestedInstance, qualification.ReceiptDigest))
		profileNodes[qualification.Profile] = identifier
	}

	goals := []protocol.ReleaseEvidenceGoal{
		goal("clock-skew-safety", []string{"catalog", "clock-skew-audit", "monitor-programs"}),
		goal("coverage-guided-fuzzing", []string{
			"approved-mutation-audit", "checker-coverage", "migration", "native-scale-benchmark",
			"semantic-mutation-portfolio",
		}),
		goal("deterministic-plans", []string{"migration"}),
		goal("developer-authoring", []string{"developer-ux", "documentation", "migration", "protobuf-descriptor"}),
		goal("first-class-faults", []string{"catalog", "migration"}),
		goal("guided-exploration", []string{
			"checker-coverage", "migration", "native-scale-benchmark", "semantic-mutation-portfolio",
		}),
		goal("known-regression-verification", append([]string{"migration", "parity"}, experimentNodes...)),
		goal("non-linear-identity", []string{"migration", "monitor-programs"}),
		profileGoal("portable-profiles", []string{
			"local-in-process", "ci-test-cluster", "remote-deployment", "grpc-only-black-box", "production-canary",
		}, []string{"catalog", "documentation", "resilience-audit"}, profileNodes),
		goal("programmable-participants", []string{"migration"}),
		goal("single-semantic-model", append([]string{
			"catalog", "composition", "monitor-programs", "protobuf-descriptor",
			"semantic-mutation-portfolio",
		}, proofNodeIdentifiers(proofManifests)...)),
		goal("unknown-bug-exploration", []string{
			"approved-mutation-audit", "checker-coverage", "semantic-mutation-portfolio",
		}),
		profileGoal("white-box-black-box", []string{
			"local-in-process", "grpc-only-black-box",
		}, []string{"monitor-programs", "resilience-audit"}, profileNodes),
	}
	return protocol.SealReleaseAssurance(protocol.ReleaseAssurance{Nodes: nodes, Goals: goals})
}

func nativeBenchmarkDigest(coverage protocol.CheckerCoverageManifest) (string, error) {
	var digest string
	for _, entry := range coverage.Entries {
		for _, evidence := range entry.Evidence {
			if evidence.Kind != "native-scale-benchmark" {
				continue
			}
			if digest != "" && digest != evidence.Digest {
				return "", errors.New("checker coverage contains multiple native scale benchmarks")
			}
			digest = evidence.Digest
		}
	}
	if digest == "" {
		return "", errors.New("checker coverage omits the native scale benchmark")
	}
	return digest, nil
}

func validateMigrationEvidence(ledger migration.Ledger) error {
	if err := ledger.Validate(); err != nil {
		return err
	}
	hasFault := false
	hasExploration := false
	hasIdentity := false
	hasParticipant := false
	for _, entry := range ledger.Entries {
		if !entry.ArtifactReplay || entry.NegativeControl == "" {
			return fmt.Errorf("migration behavior %q lacks replay or negative-control evidence", entry.Behavior)
		}
		for _, executed := range entry.ExecutedContracts {
			if executed.ScenarioDigest == "" || executed.ScenarioDigest != executed.Explain.ScenarioDigest ||
				len(executed.ExperimentDigests) == 0 {
				return fmt.Errorf("migration behavior %q lacks deterministic compiled-plan evidence", entry.Behavior)
			}
		}
		hasFault = hasFault || len(entry.Faults) != 0
		hasExploration = hasExploration || slices.Contains(entry.Relations, "bounded-exploration")
		hasIdentity = hasIdentity ||
			(slices.Contains(entry.Evidence, "identity-lineage") &&
				(slices.Contains(entry.Relations, "continuation-lineage") ||
					slices.Contains(entry.Relations, "reset-lineage") ||
					slices.Contains(entry.Relations, "generated-continuation-lineage")))
		hasParticipant = hasParticipant || slices.Contains(entry.Relations, "participant-program")
	}
	if !hasFault || !hasExploration || !hasIdentity || !hasParticipant {
		return errors.New("migration ledger lacks fault, exploration, identity, or participant evidence")
	}
	return nil
}

func catalogTrust(catalog protocol.Catalog) protocol.TrustBadge {
	for _, property := range catalog.Properties {
		if property.TrustBadge == protocol.TrustBadgeKernelWithDeclaredAxioms {
			return protocol.TrustBadgeKernelWithDeclaredAxioms
		}
	}
	return protocol.TrustBadgeKernel
}

func node(
	identifier string,
	resultClass protocol.ResultClass,
	trustBadge protocol.TrustBadge,
	digest string,
) protocol.ReleaseEvidenceNode {
	return protocol.ReleaseEvidenceNode{
		Identifier: identifier, ResultClass: resultClass, TrustBadge: trustBadge, Digest: digest,
	}
}

func goal(identifier string, requires []string, omissions ...string) protocol.ReleaseEvidenceGoal {
	return protocol.ReleaseEvidenceGoal{
		Identifier: identifier, Requires: append([]string(nil), requires...),
		Omissions: append([]string(nil), omissions...),
	}
}

func profileGoal(
	identifier string,
	profiles []string,
	requires []string,
	profileNodes map[string]string,
) protocol.ReleaseEvidenceGoal {
	omissions := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		profileNode, exists := profileNodes[profile]
		if !exists {
			omissions = append(omissions, "profile-qualification-missing/"+profile)
			continue
		}
		requires = append(requires, profileNode)
	}
	return goal(identifier, requires, omissions...)
}

func proofNodeIdentifiers(manifests []protocol.ProofManifest) []string {
	identifiers := make([]string, len(manifests))
	for index, manifest := range manifests {
		identifiers[index] = "proof/" + manifest.Identifier
	}
	return identifiers
}
