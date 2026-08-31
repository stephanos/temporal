package runevaluation

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

const (
	liveDuplicateDeliveryRunIdentity = "umpire.local.caller-closure.duplicate-delivery.live-evaluation-1"
	duplicateDeliveryFaultID         = "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
	duplicateDeliveryFaultReceiptID  = "temporal.nexus.caller-closure.fault-receipt.duplicate-delivery-observation"
	duplicateDeliveryMarker          = "temporal.nexus.caller-closure.marker.injected-duplicate-delivery-observation"
)

var liveDuplicateDeliveryBinding = runner.InputBinding{
	ArtifactSetIdentity:                     "umpire.artifact-set.2a6c3ef5fbd3b7dfba1acbe2c9ffc5ec3072b19daf50d3d63bd16b122fc2bd68",
	ArtifactSetChecksum:                     "sha256:3ddabf041e499ee0b7e970cac3900b8d6306ec9009e92924ef7b9ea0f584a5f8",
	ManifestSHA256:                          "sha256:96cf1869d444e1db25f9999ea3d3928f5c07308b8c7f387b570027f5f69b5f4b",
	ExperimentArtifactChecksum:              duplicateDeliveryExperimentChecksum,
	ExperimentBehaviorFingerprint:           duplicateDeliveryExperimentFingerprint,
	RuntimeConfigurationArtifactChecksum:    duplicateDeliveryConfigurationChecksum,
	RuntimeConfigurationBehaviorFingerprint: duplicateDeliveryConfigurationFingerprint,
	AuthorityRequiredCapabilityDefinitionIDs: []string{
		"umpire.runtime.capability.complete-workflow-history-read",
		"umpire.runtime.capability.ephemeral-server-lifecycle",
		"umpire.runtime.capability.sdk-worker-lifecycle",
	},
}

type liveNexusControlOutcome struct {
	input       artifact.AdmittedSet
	execution   artifact.AdmittedSet
	evaluation  artifact.AdmittedSet
	run         artifactv2.ExperimentRun
	rawEvidence artifactv2.RawEvidence
	evidence    artifactv2.Evidence
	result      artifactv2.Result
	summary     liveEvaluationSummary
}

func TestBoundedLiveNexusNegativeControl(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	requireNegativeControlMutationAndStatusControls(t, repositoryRoot)
	requirePairedLiveNexusNegativeControl(t, repositoryRoot)
}

func requireNegativeControlMutationAndStatusControls(t *testing.T, repositoryRoot string) {
	t.Helper()
	modelRoot := filepath.Join(repositoryRoot, "model")
	runBoundedCommand(t, modelRoot, "mise", "exec", "--", "lake", "build",
		"Temporal.Tool.RunEvaluationMutationTests")
	const nexusControls = "^(TestCheckRequestRejectsAnUnsupportedSetBeforeExecution|" +
		"TestProjectTerminalHistoryRejectsEveryIncompleteOrCorruptClosure|" +
		"TestValidateExecutionClosureAdmitsOnlyTheClosedMechanicalFourMemberSet|" +
		"TestFaultedExecutionRejectsImpossibleReceiptAndObservationOrder|" +
		"TestValidateExecutionClosureAdmitsFaultedEvidenceForEvaluation|" +
		"TestParticipantRejectsWrongCorrelationAndDuplicateCommandsBeforeAdapterIO|" +
		"TestParticipantCancellationBeforeRealizationIssuesNoControlRequest|" +
		"TestRealizationContributesOneDuplicateObservationOnlyForTheFaultedProgram|" +
		"TestFaultedRealizationEmitsNoSyntheticObservationWithoutCompletedCancellation|" +
		"TestFaultedRealizationRejectsAnUnstartedCancellationWithoutSyntheticObservation|" +
		"TestSDKAndContextFailuresRemainOperationalReceipts|" +
		"TestCleanupFailureRetainsReleasedResourcesWithoutReacquiringThem)$"
	runBoundedCommand(t, filepath.Join(repositoryRoot, "tools", "umpire", "temporal", "nexus"),
		"go", "test", "-count=1", "-tags", "test_dep", "-run", nexusControls, ".")
	const evaluationControls = "^(TestRawArtifactMutationFailsAtAdmission|" +
		"TestCheckerRequestSeparatesRuntimeAndCheckedMappings|" +
		"TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary|" +
		"TestRealCheckerObservationMutationMatrix|" +
		"TestRealCheckerDuplicateDeliveryMutationMatrix|" +
		"TestRealCheckerDuplicateDeliveryIgnoresOrdinaryOperationalFacts|" +
		"TestRealCheckerRejectsCrossedDuplicateDeliverySemanticClosure|" +
		"TestDuplicateDeliveryResponseRejectsStrictNormalSemanticBindings|" +
		"TestRealCheckerMisboundParticipantCancellationEvidenceIsSemanticConflict|" +
		"TestRealCheckerPartialEvidencePublishesAnInMemoryResult|" +
		"TestCheckWithCheckerAdmitsTheCompleteIndependentStatusMatrix|" +
		"TestCheckWithCheckerAdmitsAcceptedNonAppliedImplementationLinkResults|" +
		"TestCheckWithCheckerRejectsEverySemanticOutputInvariantClass|" +
		"TestRealCheckerCancellationPublishesNoPartialSet)$"
	runBoundedCommand(t, filepath.Join(repositoryRoot, "tools", "umpire", "runevaluation"),
		"go", "test", "-count=1", "-tags", "test_dep", "-run", evaluationControls, ".")
}

func requirePairedLiveNexusNegativeControl(t *testing.T, repositoryRoot string) {
	t.Helper()
	normalInput := liveCallerClosureInput(t)
	faultedInput := liveDuplicateDeliveryInput(t)
	require.NotEqual(t, normalInput.Identity(), faultedInput.Identity())
	require.NotEqual(t, normalInput.Checksum(), faultedInput.Checksum())
	requireCrossedLiveBindingsFailBeforeExecution(t, normalInput, faultedInput)

	normal := runLiveNexusControl(
		t,
		repositoryRoot,
		normalInput,
		liveCallerClosureBinding,
		liveCallerClosureRunIdentity,
		"satisfied",
		0,
	)
	faulted := runLiveNexusControl(
		t,
		repositoryRoot,
		faultedInput,
		liveDuplicateDeliveryBinding,
		liveDuplicateDeliveryRunIdentity,
		"violated",
		2,
	)

	require.Equal(t, normalInput.ManifestBytes(), liveCallerClosureInput(t).ManifestBytes())
	require.NotEqual(t, normal.run.RunIdentity, faulted.run.RunIdentity)
	require.NotEqual(t, normal.run.ArtifactChecksum, faulted.run.ArtifactChecksum)
	require.NotEqual(t, normal.rawEvidence.ArtifactChecksum, faulted.rawEvidence.ArtifactChecksum)
	require.NotEqual(t, normal.execution.Identity(), faulted.execution.Identity())
	require.NotEqual(t, normal.evaluation.Identity(), faulted.evaluation.Identity())
	require.NotEqual(t, normal.evaluation.ManifestBytes(), faulted.evaluation.ManifestBytes())
	require.NotEqual(t, normal.summary.Destination, faulted.summary.Destination)
	require.NotEqual(t, normal.result.ArtifactChecksum, faulted.result.ArtifactChecksum)
	require.Equal(t, checkerBehaviorFingerprint, normal.result.BehaviorFingerprint)
	require.Equal(t, checkerBehaviorFingerprint, faulted.result.BehaviorFingerprint)
	require.Equal(t, normal.result.BehaviorFingerprint, faulted.result.BehaviorFingerprint)
	require.Equal(t,
		normal.result.PropertyVerdicts[0].PropertyDefinitionID,
		faulted.result.PropertyVerdicts[0].PropertyDefinitionID,
	)
	require.Equal(t,
		normal.result.PropertyVerdicts[0].PropertyBehaviorFingerprint,
		faulted.result.PropertyVerdicts[0].PropertyBehaviorFingerprint,
	)
	require.Equal(t,
		normal.result.ImplementationLink.DestinationTarget,
		faulted.result.ImplementationLink.DestinationTarget,
	)
}

func requireCrossedLiveBindingsFailBeforeExecution(
	t *testing.T,
	normalInput artifact.AdmittedSet,
	faultedInput artifact.AdmittedSet,
) {
	t.Helper()
	for _, test := range []struct {
		name    string
		input   artifact.AdmittedSet
		binding runner.InputBinding
	}{
		{name: "normal input with faulted binding", input: normalInput, binding: liveDuplicateDeliveryBinding},
		{name: "faulted input with normal binding", input: faultedInput, binding: liveCallerClosureBinding},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			output, err := runner.Run(
				ctx,
				test.input,
				test.binding,
				"umpire.local.caller-closure.crossed-binding",
				nexus.Binding{},
			)
			require.Error(t, err)
			require.Empty(t, output.AdmittedSet().Identity())
		})
	}
}

func runLiveNexusControl(
	t *testing.T,
	repositoryRoot string,
	input artifact.AdmittedSet,
	binding runner.InputBinding,
	runIdentity string,
	semanticStatus string,
	evaluationExitStatus int,
) liveNexusControlOutcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()
	operational, err := runner.Run(ctx, input, binding, runIdentity, nexus.Binding{})
	require.NoError(t, err)
	requireLiveOperationalClosure(t, operational.ExperimentRun(), operational.RawEvidence())

	executionRoot := t.TempDir()
	firstExecutionDestination, err := artifact.PublishSet(executionRoot, operational.AdmittedSet())
	require.NoError(t, err)
	secondExecutionDestination, err := artifact.PublishSet(executionRoot, operational.AdmittedSet())
	require.NoError(t, err)
	require.Equal(t, firstExecutionDestination, secondExecutionDestination)
	reopenedExecution, err := artifact.LoadSet(firstExecutionDestination)
	require.NoError(t, err)
	require.Equal(t, operational.AdmittedSet().Identity(), reopenedExecution.Identity())
	require.Equal(t, operational.AdmittedSet().ManifestBytes(), reopenedExecution.ManifestBytes())

	commandInputRoot := filepath.Join(t.TempDir(), "execution")
	writeExecutionSet(t, commandInputRoot, operational.AdmittedSet())
	outputRoot := t.TempDir()
	firstSummaryBytes := runLiveEvaluationCommandWithStatus(
		t, repositoryRoot, commandInputRoot, outputRoot, evaluationExitStatus,
	)
	secondSummaryBytes := runLiveEvaluationCommandWithStatus(
		t, repositoryRoot, commandInputRoot, outputRoot, evaluationExitStatus,
	)
	require.Equal(t, firstSummaryBytes, secondSummaryBytes)

	var summary liveEvaluationSummary
	require.NoError(t, json.Unmarshal(firstSummaryBytes, &summary))
	require.Equal(t, "umpire-local-run-evaluation-summary/v2", summary.FormatVersion)
	require.Equal(t, runIdentity, summary.RunIdentity)
	require.Equal(t, "succeeded", summary.OperationalStatus)
	require.Equal(t, "accepted", summary.ObservationEvaluationStatus)
	require.Equal(t, semanticStatus, summary.SemanticStatus)
	require.Equal(t, filepath.Join(
		outputRoot, "sets", strings.TrimPrefix(summary.ManifestSHA256, "sha256:"),
	), summary.Destination)

	reopenedEvaluation, err := artifact.LoadSet(summary.Destination)
	require.NoError(t, err)
	require.Equal(t, summary.ArtifactSetChecksum, reopenedEvaluation.Checksum())
	require.Equal(t, summary.ManifestSHA256, reopenedEvaluation.ManifestSHA256())
	requireLiveEvaluationMembers(t, commandInputRoot, summary.Destination)
	evidenceBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "evidence.json"))
	require.NoError(t, err)
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	resultBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "result.json"))
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.Equal(t, summary.EvidenceArtifactChecksum, evidence.ArtifactChecksum)
	require.Equal(t, summary.ResultArtifactChecksum, result.ArtifactChecksum)
	require.Equal(t, summary.EvaluationOutcomeChecksum, *result.EvaluationOutcomeChecksum)

	if semanticStatus == "satisfied" {
		requireLiveSemanticResult(
			t, operational.ExperimentRun(), operational.RawEvidence(), evidence, result,
		)
	} else {
		requireLiveNegativeControlResult(
			t, input, operational.ExperimentRun(), operational.RawEvidence(), evidence, result,
		)
	}
	return liveNexusControlOutcome{
		input:       input,
		execution:   operational.AdmittedSet(),
		evaluation:  reopenedEvaluation,
		run:         operational.ExperimentRun(),
		rawEvidence: operational.RawEvidence(),
		evidence:    evidence,
		result:      result,
		summary:     summary,
	}
}

func runLiveEvaluationCommandWithStatus(
	t *testing.T,
	repositoryRoot string,
	executionRoot string,
	outputRoot string,
	expectedStatus int,
) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx,
		"make", "-C", repositoryRoot, "--no-print-directory",
		"umpire-check-local-run-evaluation", "SET="+executionRoot, "OUTPUT_ROOT="+outputRoot,
	)
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if expectedStatus == 0 {
		require.NoError(t, err, stderr.String())
		require.Empty(t, stderr.String())
	} else {
		var exitError *exec.ExitError
		require.ErrorAs(t, err, &exitError, stdout.String(), stderr.String())
		require.Equal(t, expectedStatus, exitError.ExitCode(), stdout.String(), stderr.String())
		require.Contains(t, stderr.String(), "Error 2")
	}
	require.NoError(t, ctx.Err())
	require.Equal(t, 1, bytes.Count(stdout.Bytes(), []byte{'\n'}))
	return stdout.Bytes()
}

func liveDuplicateDeliveryInput(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	root := filepath.Join(
		"..", "temporal", "nexus", "testdata", "caller-closure-duplicate-delivery-input-set",
	)
	files := make(map[string][]byte, 3)
	for _, path := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		files[path] = encoded
	}
	input, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return input
}

func requireLiveNegativeControlResult(
	t *testing.T,
	input artifact.AdmittedSet,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
	result artifactv2.Result,
) {
	t.Helper()
	executable, ok := input.Executable()
	require.True(t, ok)
	require.Len(t, executable.Experiment().Plan.RequestedFaults, 1)
	require.Equal(t, duplicateDeliveryFaultID,
		executable.Experiment().Plan.RequestedFaults[0].DefinitionID)
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.NotNil(t, run.ControlAttempts[0].ReceiptFactDefinitionID)
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Equal(t, "accepted", evidence.ObservationEvaluationStatus)
	require.NotNil(t, evidence.EvidenceBackedModelTrace)
	require.True(t, evidence.EvidenceBackedModelTrace.SourceClosed)
	require.Empty(t, evidence.Diagnostics)
	require.Empty(t, evidence.KnownGaps)
	require.Equal(t, "succeeded", result.OperationalStatus)
	require.Equal(t, "accepted", result.ObservationEvaluationStatus)
	require.Equal(t, "applied", result.ImplementationLinkStatus)
	require.Equal(t, "violated", result.SemanticStatus)
	require.Equal(t, "complete", result.CleanupStatus)
	require.Empty(t, result.KnownGaps)
	require.Equal(t, duplicateDeliveryImplementationLinkID,
		result.ImplementationLink.DefinitionID)
	require.Equal(t, callerClosureTargetID,
		result.ImplementationLink.DestinationTarget.DefinitionID)
	require.Len(t, result.PropertyVerdicts, 1)
	verdict := result.PropertyVerdicts[0]
	require.Equal(t, callerClosurePropertyID, verdict.PropertyDefinitionID)
	require.Equal(t, callerClosurePropertyFingerprint, verdict.PropertyBehaviorFingerprint)
	require.Equal(t, "violated", verdict.Status)
	require.Equal(t, []string{
		"workflow-nexus.property.clause.delivery",
		"workflow-nexus.property.clause.ownership",
		"workflow-nexus.property.clause.uniqueness",
	}, clauseDefinitionIDs(verdict.Clauses))
	require.Equal(t, []string{"satisfied", "satisfied", "violated"}, []string{
		verdict.Clauses[0].Status,
		verdict.Clauses[1].Status,
		verdict.Clauses[2].Status,
	})
	require.Equal(t, "violated", result.QuerySummary.Status)

	requested := liveFactsWithStringField(
		rawEvidence, umpireruntime.EvidenceFieldEventType,
		"temporal.history.NexusOperationCancelRequested",
	)
	completed := liveFactsWithStringField(
		rawEvidence, umpireruntime.EvidenceFieldEventType,
		"temporal.history.NexusOperationCancelRequestCompleted",
	)
	require.Len(t, requested, 1)
	require.Len(t, completed, 1)
	require.Equal(t, []string{requested[0].FactDefinitionID},
		completed[0].CausalFactDefinitionIDs)
	callback := liveFactsWithField(rawEvidence, umpireruntime.EvidenceFieldCancellationCallbackCount)
	synthetic := liveFactsWithField(rawEvidence, umpireruntime.EvidenceFieldSyntheticContributionMarker)
	require.Len(t, callback, 1)
	require.Len(t, synthetic, 1)
	require.Equal(t, json.Number("1"), rawEvidenceNaturalField(
		callback[0], umpireruntime.EvidenceFieldCancellationCallbackCount,
	))
	require.Equal(t, json.Number("1"), rawEvidenceNaturalField(
		synthetic[0], umpireruntime.EvidenceFieldSyntheticContributionCount,
	))
	require.Equal(t, duplicateDeliveryMarker, rawEvidenceStringField(
		synthetic[0], umpireruntime.EvidenceFieldSyntheticContributionMarker,
	))
	require.Equal(t, duplicateDeliveryFaultID, rawEvidenceStringField(
		synthetic[0], umpireruntime.EvidenceFieldFaultDefinitionID,
	))
	require.Equal(t, duplicateDeliveryFaultReceiptID, rawEvidenceStringField(
		synthetic[0], umpireruntime.EvidenceFieldFaultReceiptDefinitionID,
	))
	require.Equal(t, []string{callback[0].FactDefinitionID}, synthetic[0].CausalFactDefinitionIDs)
	propertyEvidence := propertyEvidenceDefinitionIDs(verdict.Clauses)
	require.Contains(t, propertyEvidence, synthetic[0].FactDefinitionID)
	for _, link := range evidence.EvidenceLinks {
		orderingFacts := make([]string, len(link.OrderingSupport))
		for index, support := range link.OrderingSupport {
			orderingFacts[index] = support.FactDefinitionID
		}
		require.Contains(t, orderingFacts, callback[0].FactDefinitionID)
		require.Contains(t, orderingFacts, synthetic[0].FactDefinitionID)
	}
}

func rawEvidenceNaturalField(fact artifactv2.RawEvidenceFact, definitionID string) json.Number {
	for _, field := range fact.Fields {
		if field.FieldDefinitionID == definitionID {
			value, _ := field.Value.(json.Number)
			return value
		}
	}
	return ""
}

func liveFactsWithField(
	rawEvidence artifactv2.RawEvidence,
	definitionID string,
) []artifactv2.RawEvidenceFact {
	return slices.DeleteFunc(slices.Clone(rawEvidence.Facts), func(fact artifactv2.RawEvidenceFact) bool {
		return !rawEvidenceFactHasField(fact, definitionID)
	})
}

func liveFactsWithStringField(
	rawEvidence artifactv2.RawEvidence,
	definitionID string,
	value string,
) []artifactv2.RawEvidenceFact {
	return slices.DeleteFunc(slices.Clone(rawEvidence.Facts), func(fact artifactv2.RawEvidenceFact) bool {
		return rawEvidenceStringField(fact, definitionID) != value
	})
}
