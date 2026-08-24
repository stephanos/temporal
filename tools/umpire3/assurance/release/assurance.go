package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"slices"

	umpire3assets "go.temporal.io/server/tools/umpire3"
	"go.temporal.io/server/tools/umpire3/assurance/audit/clockskew"
	"go.temporal.io/server/tools/umpire3/assurance/audit/developerexperience"
	"go.temporal.io/server/tools/umpire3/assurance/audit/documentationaudit"
	mutationaudit "go.temporal.io/server/tools/umpire3/assurance/audit/mutation"
	"go.temporal.io/server/tools/umpire3/assurance/audit/resilience"
	"go.temporal.io/server/tools/umpire3/assurance/migration"
	"go.temporal.io/server/tools/umpire3/mutation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	protocolrelease "go.temporal.io/server/tools/umpire3/protocol/release"
)

func Bind(
	manifest protocolrelease.ReleaseManifest,
	experiments []protocolexperiment.Experiment,
	ledger migration.Ledger,
	ledgerBytes []byte,
) (protocolrelease.ReleaseManifest, error) {
	bound, err := BindArtifactBindings(manifest, experiments)
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	bound.Migration, err = summarizeMigration(ledger, ledgerBytes)
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	bound.Assurance, err = deriveAssurance(bound, ledger)
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	if err := ValidateArtifactBindingsAgainstCurrent(bound); err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	return bound, nil
}

func RebindCurrentAssurance(manifest protocolrelease.ReleaseManifest) (protocolrelease.ReleaseManifest, error) {
	ledger, ledgerBytes, err := migration.DefaultLedger()
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	expectedMigration, err := summarizeMigration(ledger, ledgerBytes)
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	if !reflect.DeepEqual(manifest.Migration, expectedMigration) {
		return protocolrelease.ReleaseManifest{}, errors.New("release migration binding does not match the current ledger")
	}
	manifest.Assurance, err = deriveAssurance(manifest, ledger)
	if err != nil {
		return protocolrelease.ReleaseManifest{}, err
	}
	return manifest, nil
}

func ValidateAgainstCurrent(manifest protocolrelease.ReleaseManifest) error {
	if err := ValidateArtifactBindingsAgainstCurrent(manifest); err != nil {
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

func summarizeMigration(ledger migration.Ledger, encoded []byte) (protocolrelease.ReleaseMigration, error) {
	if len(encoded) == 0 {
		return protocolrelease.ReleaseMigration{}, errors.New("release binding requires encoded migration evidence")
	}
	decoded, err := migration.DecodeLedger(encoded)
	if err != nil {
		return protocolrelease.ReleaseMigration{}, err
	}
	if !reflect.DeepEqual(ledger, decoded) {
		return protocolrelease.ReleaseMigration{}, errors.New("release migration value does not match its encoded evidence")
	}
	summary := protocolrelease.ReleaseMigration{
		FormatVersion: ledger.FormatVersion,
		BehaviorCount: len(ledger.Entries),
	}
	digest := sha256.Sum256(encoded)
	summary.LedgerHash = "sha256:" + hex.EncodeToString(digest[:])
	for _, entry := range ledger.Entries {
		switch entry.Fidelity {
		case protocolcatalog.FidelityExact:
			summary.ExactCount++
		case protocolcatalog.FidelitySemanticEquivalent:
			summary.SemanticEquivalentCount++
		case protocolcatalog.FidelityPartial:
			summary.PartialCount++
		case protocolcatalog.FidelityInventoryOnly:
			summary.InventoryOnlyCount++
		default:
			return protocolrelease.ReleaseMigration{}, fmt.Errorf("unknown migration fidelity %q", entry.Fidelity)
		}
	}
	return summary, nil
}

func deriveAssurance(
	manifest protocolrelease.ReleaseManifest,
	ledger migration.Ledger,
) (protocolrelease.ReleaseAssurance, error) {
	if err := validateMigrationEvidence(ledger); err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	parity, err := protocolcatalog.DefaultParityLedger()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	proofManifests, err := protocolchecker.DefaultProofManifests()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	auditExperimentID := "nexus-cancellation-stale-completion-v1"
	auditBinding, exists := manifest.Experiments[auditExperimentID]
	if !exists {
		return protocolrelease.ReleaseAssurance{}, errors.New("release omits the approved mutation audit experiment")
	}
	mutationAudit, err := mutation.DefaultApprovedMutationAuditForBinding(
		auditExperimentID,
		string(protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess),
		auditBinding.Digest,
	)
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	if err := mutationAudit.ValidateCoverageGuidance(); err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("validate coverage-guided mutation audit: %w", err)
	}
	semanticMutationAudit, err := mutationaudit.Default()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("load semantic mutation audit: %w", err)
	}
	if semanticMutationAudit.CampaignSourceDigest != mutationAudit.SourceDigest {
		return protocolrelease.ReleaseAssurance{}, errors.New("semantic mutation audit does not match the release mutation campaign")
	}
	developerAudit, err := developerexperience.Run(mutationAudit.PromotionSource)
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("run developer UX audit: %w", err)
	}
	clockSkewAudit, err := clockskew.RunAudit()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("run clock-skew audit: %w", err)
	}
	documentationAudit, err := documentationaudit.Audit(umpire3assets.PublishedDocumentation)
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("audit documentation: %w", err)
	}
	resilienceAudit, err := resilience.DefaultAudit()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, fmt.Errorf("load resilience audit: %w", err)
	}
	checkerCoverage, err := protocolchecker.DefaultCheckerCoverage()
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}
	nativeBenchmark, err := nativeBenchmarkDigest(checkerCoverage)
	if err != nil {
		return protocolrelease.ReleaseAssurance{}, err
	}

	nodes := []protocolrelease.ReleaseEvidenceNode{
		node("approved-mutation-audit", mutationAudit.ResultClass,
			mutationAudit.TrustBadge, mutationAudit.ArtifactDigest),
		node("catalog", protocolcatalog.ResultClassMetadataValidated, catalogTrust(catalog), manifest.CatalogHash),
		node("checker-coverage", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, manifest.CheckerCoverageHash),
		node("clock-skew-audit", protocolcatalog.ResultClassImplementationConforming,
			protocolcatalog.TrustBadgeTestedInstance, clockSkewAudit.ArtifactDigest),
		node("composition", composition.ResultClass, composition.TrustBadge, manifest.CompositionSemanticHash),
		node("developer-ux", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, developerAudit.ArtifactDigest),
		node("documentation", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, documentationAudit.ArtifactDigest),
		node("migration", protocolcatalog.ResultClassImplementationConforming,
			protocolcatalog.TrustBadgeTestedInstance, manifest.Migration.LedgerHash),
		node("monitor-programs", protocolcatalog.ResultClassEvidenceResolved,
			protocolcatalog.TrustBadgeTestedInstance, manifest.MonitorSemanticHash),
		node("native-scale-benchmark", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, nativeBenchmark),
		node("parity", parity.ResultClass, parity.TrustBadge, manifest.ParitySemanticHash),
		node("protobuf-descriptor", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, manifest.DescriptorHash),
		node("resilience-audit", protocolcatalog.ResultClassImplementationConforming,
			protocolcatalog.TrustBadgeTestedInstance, resilienceAudit.ArtifactDigest),
		node("semantic-mutation-portfolio", protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, semanticMutationAudit.ArtifactDigest),
	}
	experimentNodes := make([]string, 0, len(manifest.Experiments))
	for identifier, binding := range manifest.Experiments {
		nodeIdentifier := "experiment/" + identifier
		nodes = append(nodes, node(nodeIdentifier, protocolcatalog.ResultClassMetadataValidated,
			protocolcatalog.TrustBadgeTestedInstance, binding.Digest))
		experimentNodes = append(experimentNodes, nodeIdentifier)
	}
	slices.Sort(experimentNodes)
	for _, proofManifest := range proofManifests {
		digest, digestErr := proofManifest.Digest()
		if digestErr != nil {
			return protocolrelease.ReleaseAssurance{}, digestErr
		}
		nodes = append(nodes, node("proof/"+proofManifest.Identifier,
			proofManifest.ResultClass, proofManifest.TrustBadge, digest))
	}
	profileNodes := make(map[string]string, len(manifest.Qualifications))
	for _, qualification := range manifest.Qualifications {
		identifier := "profile/" + qualification.Profile
		nodes = append(nodes, node(identifier, protocolcatalog.ResultClassImplementationConforming,
			protocolcatalog.TrustBadgeTestedInstance, qualification.ReceiptDigest))
		profileNodes[qualification.Profile] = identifier
	}

	goals := []protocolrelease.ReleaseEvidenceGoal{
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
	return protocolrelease.SealReleaseAssurance(protocolrelease.ReleaseAssurance{Nodes: nodes, Goals: goals})
}

func nativeBenchmarkDigest(coverage protocolchecker.CheckerCoverageManifest) (string, error) {
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

func catalogTrust(catalog protocolcatalog.Catalog) protocolcatalog.TrustBadge {
	for _, property := range catalog.Properties {
		if property.TrustBadge == protocolcatalog.TrustBadgeKernelWithDeclaredAxioms {
			return protocolcatalog.TrustBadgeKernelWithDeclaredAxioms
		}
	}
	return protocolcatalog.TrustBadgeKernel
}

func node(
	identifier string,
	resultClass protocolcatalog.ResultClass,
	trustBadge protocolcatalog.TrustBadge,
	digest string,
) protocolrelease.ReleaseEvidenceNode {
	return protocolrelease.ReleaseEvidenceNode{
		Identifier: identifier, ResultClass: resultClass, TrustBadge: trustBadge, Digest: digest,
	}
}

func goal(identifier string, requires []string, omissions ...string) protocolrelease.ReleaseEvidenceGoal {
	return protocolrelease.ReleaseEvidenceGoal{
		Identifier: identifier, Requires: append([]string(nil), requires...),
		Omissions: append([]string(nil), omissions...),
	}
}

func profileGoal(
	identifier string,
	profiles []string,
	requires []string,
	profileNodes map[string]string,
) protocolrelease.ReleaseEvidenceGoal {
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

func proofNodeIdentifiers(manifests []protocolchecker.ProofManifest) []string {
	identifiers := make([]string, len(manifests))
	for index, manifest := range manifests {
		identifiers[index] = "proof/" + manifest.Identifier
	}
	return identifiers
}
