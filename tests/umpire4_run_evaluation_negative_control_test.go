//go:build test_dep && integration

package tests

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
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

const (
	liveDuplicateDeliveryRunIdentity = "umpire.local.caller-closure.duplicate-delivery.live-evaluation-1"
	duplicateDeliveryFaultID         = "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
	duplicateDeliveryFaultReceiptID  = "temporal.nexus.caller-closure.fault-receipt.duplicate-delivery-observation"
	duplicateDeliveryMarker          = "temporal.nexus.caller-closure.marker.injected-duplicate-delivery-observation"
	liveCheckerBehaviorFingerprint   = "sha256:e649a5e059ef42806eb661deb1c1ccba08ec5202425d7a824f7e25026f8134da"
	liveDuplicateDeliveryLinkID      = "temporal.system.nexus.caller-closure.duplicate-delivery.implementation-link"
)

type liveNexusControlOutcome struct {
	input                       artifact.AdmittedSet
	execution                   artifact.AdmittedSet
	evaluation                  artifact.AdmittedSet
	runIdentity                 string
	runArtifactChecksum         string
	rawEvidenceArtifactChecksum string
	semantic                    liveSemanticSnapshot
	summary                     liveEvaluationSummary
}

type runEvaluationFactoryAccess struct {
	prepareCalls int
}

func (f *runEvaluationFactoryAccess) Prepare(
	context.Context,
	umpireruntime.CheckedRunRequest,
	umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	f.prepareCalls++
	return nil, umpireruntime.Receipt{}
}

func TestUmpireDuplicateDeliveryRunEvaluation(t *testing.T) {
	repositoryRoot, err := filepath.Abs("..")
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
	env, factory := newUmpireTestEnvironment(t)
	adapter := newUmpireNexusBinding(t, factory)

	normal := runLiveNexusControl(
		env.Context(),
		t,
		repositoryRoot,
		normalInput,
		callerClosureInputBinding,
		liveCallerClosureRunIdentity,
		"satisfied",
		0,
		adapter,
	)
	faulted := runLiveNexusControl(
		env.Context(),
		t,
		repositoryRoot,
		faultedInput,
		callerClosureDuplicateDeliveryBinding(),
		liveDuplicateDeliveryRunIdentity,
		"violated",
		2,
		adapter,
	)
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())

	require.Equal(t, normalInput.ManifestBytes(), liveCallerClosureInput(t).ManifestBytes())
	require.NotEqual(t, normal.runIdentity, faulted.runIdentity)
	require.NotEqual(t, normal.runArtifactChecksum, faulted.runArtifactChecksum)
	require.NotEqual(t, normal.rawEvidenceArtifactChecksum, faulted.rawEvidenceArtifactChecksum)
	require.NotEqual(t, normal.execution.Identity(), faulted.execution.Identity())
	require.NotEqual(t, normal.evaluation.Identity(), faulted.evaluation.Identity())
	require.NotEqual(t, normal.evaluation.ManifestBytes(), faulted.evaluation.ManifestBytes())
	require.NotEqual(t, normal.summary.Destination, faulted.summary.Destination)
	require.NotEqual(t, normal.semantic.resultArtifactChecksum, faulted.semantic.resultArtifactChecksum)
	require.Equal(t, liveCheckerBehaviorFingerprint, normal.semantic.resultBehaviorFingerprint)
	require.Equal(t, liveCheckerBehaviorFingerprint, faulted.semantic.resultBehaviorFingerprint)
	require.Equal(t, normal.semantic.resultBehaviorFingerprint, faulted.semantic.resultBehaviorFingerprint)
	require.Equal(t,
		normal.semantic.propertyDefinitionID,
		faulted.semantic.propertyDefinitionID,
	)
	require.Equal(t,
		normal.semantic.propertyBehaviorFingerprint,
		faulted.semantic.propertyBehaviorFingerprint,
	)
	require.Equal(t,
		normal.semantic.implementationDestination,
		faulted.semantic.implementationDestination,
	)
}

func requireCrossedLiveBindingsFailBeforeExecution(
	t *testing.T,
	normalInput artifact.AdmittedSet,
	faultedInput artifact.AdmittedSet,
) {
	t.Helper()
	factory := &runEvaluationFactoryAccess{}
	adapter := newUmpireNexusBinding(t, factory)
	for _, test := range []struct {
		name    string
		input   artifact.AdmittedSet
		binding runner.InputBinding
	}{
		{name: "normal input with faulted binding", input: normalInput, binding: callerClosureDuplicateDeliveryBinding()},
		{name: "faulted input with normal binding", input: faultedInput, binding: callerClosureInputBinding},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			output, err := runner.Run(
				ctx,
				test.input,
				test.binding,
				"umpire.local.caller-closure.crossed-binding",
				adapter,
			)
			require.Error(t, err)
			require.Empty(t, output.AdmittedSet().Identity())
			require.Zero(t, factory.prepareCalls)
		})
	}
}

func runLiveNexusControl(
	baseContext context.Context,
	t *testing.T,
	repositoryRoot string,
	input artifact.AdmittedSet,
	binding runner.InputBinding,
	runIdentity string,
	semanticStatus string,
	evaluationExitStatus int,
	adapter nexus.Binding,
) liveNexusControlOutcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(baseContext, 135*time.Second)
	defer cancel()
	operational, err := runner.Run(ctx, input, binding, runIdentity, adapter)
	require.NoError(t, err)
	requireLiveOperationalClosure(t, operational)

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
	writeRunEvaluationExecutionSet(t, commandInputRoot, operational.AdmittedSet())
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
	resultBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "result.json"))
	require.NoError(t, err)

	var semantic liveSemanticSnapshot
	if semanticStatus == "satisfied" {
		semantic = requireLiveSemanticResult(
			t, operational, evidenceBytes, resultBytes, summary,
		)
	} else {
		semantic = requireLiveNegativeControlResult(
			t, input, operational, evidenceBytes, resultBytes, summary,
		)
	}
	run := operational.ExperimentRun()
	rawEvidence := operational.RawEvidence()
	return liveNexusControlOutcome{
		input:                       input,
		execution:                   operational.AdmittedSet(),
		evaluation:                  reopenedEvaluation,
		runIdentity:                 run.RunIdentity,
		runArtifactChecksum:         run.ArtifactChecksum,
		rawEvidenceArtifactChecksum: rawEvidence.ArtifactChecksum,
		semantic:                    semantic,
		summary:                     summary,
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
	return loadUmpireCallerClosureInputSet(t, "caller-closure-duplicate-delivery-input-set")
}

func requireLiveNegativeControlResult(
	t *testing.T,
	input artifact.AdmittedSet,
	operational umpireruntime.Output,
	evidenceBytes []byte,
	resultBytes []byte,
	summary liveEvaluationSummary,
) liveSemanticSnapshot {
	t.Helper()
	run := operational.ExperimentRun()
	rawEvidence := operational.RawEvidence()
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.Equal(t, summary.EvidenceArtifactChecksum, evidence.ArtifactChecksum)
	require.Equal(t, summary.ResultArtifactChecksum, result.ArtifactChecksum)
	require.Equal(t, summary.EvaluationOutcomeChecksum, *result.EvaluationOutcomeChecksum)
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
	require.Equal(t, liveDuplicateDeliveryLinkID,
		result.ImplementationLink.DefinitionID)
	require.Equal(t, callerClosureSubject.ImplementationLinkDestinationTarget.DefinitionID,
		result.ImplementationLink.DestinationTarget.DefinitionID)
	require.Len(t, result.PropertyVerdicts, 1)
	verdict := result.PropertyVerdicts[0]
	require.Equal(t, callerClosureSubject.Properties[0].DefinitionID, verdict.PropertyDefinitionID)
	require.Equal(t, callerClosureSubject.Properties[0].BehaviorFingerprint, verdict.PropertyBehaviorFingerprint)
	require.Equal(t, "violated", verdict.Status)
	clauseDefinitionIDs := make([]string, len(verdict.Clauses))
	for index, clause := range verdict.Clauses {
		clauseDefinitionIDs[index] = clause.ClauseDefinitionID
	}
	require.Equal(t, []string{
		"workflow-nexus.property.clause.delivery",
		"workflow-nexus.property.clause.ownership",
		"workflow-nexus.property.clause.uniqueness",
	}, clauseDefinitionIDs)
	require.Equal(t, []string{"satisfied", "satisfied", "violated"}, []string{
		verdict.Clauses[0].Status,
		verdict.Clauses[1].Status,
		verdict.Clauses[2].Status,
	})
	require.Equal(t, "violated", result.QuerySummary.Status)

	stringFields := make([]map[string]string, len(rawEvidence.Facts))
	naturalFields := make([]map[string]json.Number, len(rawEvidence.Facts))
	var requestedIndexes []int
	var completedIndexes []int
	var callbackIndexes []int
	var syntheticIndexes []int
	for index, fact := range rawEvidence.Facts {
		stringFields[index] = make(map[string]string)
		naturalFields[index] = make(map[string]json.Number)
		for _, field := range fact.Fields {
			if value, ok := field.Value.(string); ok {
				stringFields[index][field.FieldDefinitionID] = value
			}
			if value, ok := field.Value.(json.Number); ok {
				naturalFields[index][field.FieldDefinitionID] = value
			}
		}
		switch stringFields[index][umpireruntime.EvidenceFieldEventType] {
		case "temporal.history.NexusOperationCancelRequested":
			requestedIndexes = append(requestedIndexes, index)
		case "temporal.history.NexusOperationCancelRequestCompleted":
			completedIndexes = append(completedIndexes, index)
		default:
		}
		if _, ok := naturalFields[index][umpireruntime.EvidenceFieldCancellationCallbackCount]; ok && fact.KindDefinitionID == umpireruntime.EvidenceKindParticipantCommand {
			callbackIndexes = append(callbackIndexes, index)
		}
		if _, ok := stringFields[index][umpireruntime.EvidenceFieldSyntheticContributionMarker]; ok {
			syntheticIndexes = append(syntheticIndexes, index)
		}
	}
	require.Len(t, requestedIndexes, 1)
	require.Len(t, completedIndexes, 1)
	require.Len(t, callbackIndexes, 1)
	require.Len(t, syntheticIndexes, 1)
	requested := rawEvidence.Facts[requestedIndexes[0]]
	completed := rawEvidence.Facts[completedIndexes[0]]
	callbackIndex := callbackIndexes[0]
	syntheticIndex := syntheticIndexes[0]
	callback := rawEvidence.Facts[callbackIndex]
	synthetic := rawEvidence.Facts[syntheticIndex]
	require.Equal(t, []string{requested.FactDefinitionID}, completed.CausalFactDefinitionIDs)
	require.Equal(t, json.Number("1"),
		naturalFields[callbackIndex][umpireruntime.EvidenceFieldCancellationCallbackCount])
	require.Equal(t, json.Number("1"),
		naturalFields[syntheticIndex][umpireruntime.EvidenceFieldSyntheticContributionCount])
	require.Equal(t, json.Number("1"),
		naturalFields[syntheticIndex][umpireruntime.EvidenceFieldCancellationCallbackCount])
	require.Equal(t, duplicateDeliveryMarker,
		stringFields[syntheticIndex][umpireruntime.EvidenceFieldSyntheticContributionMarker])
	require.Equal(t, duplicateDeliveryFaultID,
		stringFields[syntheticIndex][umpireruntime.EvidenceFieldFaultDefinitionID])
	require.Equal(t, duplicateDeliveryFaultReceiptID,
		stringFields[syntheticIndex][umpireruntime.EvidenceFieldFaultReceiptDefinitionID])
	require.Equal(t, []string{callback.FactDefinitionID}, synthetic.CausalFactDefinitionIDs)
	var propertyEvidence []string
	for _, clause := range verdict.Clauses {
		for _, link := range clause.EvidenceLinks {
			propertyEvidence = append(propertyEvidence, link.EvidenceDefinitionIDs...)
		}
	}
	slices.Sort(propertyEvidence)
	propertyEvidence = slices.Compact(propertyEvidence)
	require.Contains(t, propertyEvidence, synthetic.FactDefinitionID)
	for _, link := range evidence.EvidenceLinks {
		orderingFacts := make([]string, len(link.OrderingSupport))
		for index, support := range link.OrderingSupport {
			orderingFacts[index] = support.FactDefinitionID
		}
		require.Contains(t, orderingFacts, callback.FactDefinitionID)
		require.Contains(t, orderingFacts, synthetic.FactDefinitionID)
	}
	return liveSemanticSnapshot{
		resultArtifactChecksum:      result.ArtifactChecksum,
		resultBehaviorFingerprint:   result.BehaviorFingerprint,
		propertyDefinitionID:        verdict.PropertyDefinitionID,
		propertyBehaviorFingerprint: verdict.PropertyBehaviorFingerprint,
		implementationDestination: liveImplementationTarget{
			definitionID:        result.ImplementationLink.DestinationTarget.DefinitionID,
			kind:                result.ImplementationLink.DestinationTarget.Kind,
			behaviorFingerprint: result.ImplementationLink.DestinationTarget.BehaviorFingerprint,
		},
	}
}
