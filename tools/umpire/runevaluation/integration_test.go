package runevaluation

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

var (
	realCheckerBuildOnce sync.Once
	realCheckerBuiltPath string
	realCheckerBuildErr  error
	realCheckerBuildLog  []byte
)

func TestRealCheckerSiblingIsDeterministic(t *testing.T) {
	process := realCheckerProcess(t)
	request := exactCallerClosureMutationRequest(t)

	firstBytes := runRealCheckerBytes(t, process, request)
	secondBytes := runRealCheckerBytes(t, process, request)
	require.Equal(t, firstBytes, secondBytes)
	require.LessOrEqual(t, len(firstBytes), maximumCheckerProtocolBytes)
	require.True(t, bytes.HasSuffix(firstBytes, []byte{'\n'}))
	decoded, err := checkerResponseDecoder.Decode(firstBytes)
	require.NoError(t, err)
	evidence := projectCheckerEvidence(decoded, request)
	require.NoError(t, artifactv2.ValidateEvidence(evidence))
	require.Contains(t, evidence.EvidenceBackedModelTrace.EvidenceDefinitionIDs,
		"umpire.runtime.fact.cleanup.fixture")
	require.False(t, slices.ContainsFunc(evidence.EvidenceLinks, func(link artifactv2.EvidenceLink) bool {
		return slices.Contains(link.EvidenceDefinitionIDs, "umpire.runtime.fact.cleanup.fixture")
	}))
	foreignEvidence := evidence
	foreignEvidence.EvidenceLinks = append([]artifactv2.EvidenceLink(nil), evidence.EvidenceLinks...)
	foreignEvidence.EvidenceLinks[0].EvidenceDefinitionIDs = []string{"umpire.runtime.fact.foreign"}
	require.ErrorContains(t, artifactv2.ValidateEvidence(foreignEvidence),
		"absent from its Evidence-backed Model Trace")
	crossPlanEvidence := evidence
	crossPlanEvidence.Mapping = artifactDefinitionReference(request.ObservationProgram)
	require.ErrorContains(t, artifactv2.ValidateEvidence(crossPlanEvidence),
		"does not match the Evidence mapping")
	mutatedEvidence := evidence
	mutatedEvidence.EvidenceLinks = append([]artifactv2.EvidenceLink(nil), evidence.EvidenceLinks...)
	mutatedEvidence.EvidenceLinks[1].OrderingSupport = append(
		[]artifactv2.EvidenceOrderingFact(nil), evidence.EvidenceLinks[1].OrderingSupport[1:]...,
	)
	require.ErrorContains(t, artifactv2.ValidateEvidence(mutatedEvidence),
		"Evidence-backed Model Trace")
	first, err := decodeCheckerResponse(firstBytes, request)
	require.NoError(t, err)
	second, err := process.run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, "accepted", first.ObservationEvaluationStatus)
	require.Equal(t, "satisfied", first.SemanticStatus)
	require.Equal(t, "satisfied", first.QuerySummary.Status)
	require.Len(t, first.PropertyVerdicts, 1)
	require.Equal(t, "workflow-nexus.property.caller-closure", first.PropertyVerdicts[0].PropertyDefinitionID)
	require.Equal(t, "satisfied", first.PropertyVerdicts[0].Status)
	require.NotNil(t, first.EvidenceBackedModelTrace)
	require.Equal(t, "temporal.system.nexus.caller-closure.state",
		first.EvidenceBackedModelTrace.Trace.InitialState.DefinitionID)
	require.Equal(t, "temporal.history.WorkflowExecutionStarted",
		first.EvidenceBackedModelTrace.Trace.InitialState.Value)
	require.Len(t, first.EvidenceBackedModelTrace.Trace.Steps, 1)
	require.Len(t, first.EvidenceBackedModelTrace.Trace.Steps[0].Observations, 3)
	require.Len(t, first.EvidenceLinks, 7)

}

func TestRealCheckerSiblingAdmitsDuplicateDeliveryViolation(t *testing.T) {
	process := realCheckerProcess(t)
	root := filepath.Join(
		"..", "temporal", "nexus", "testdata", "caller-closure-duplicate-delivery-run-set",
	)
	members := make([]artifact.SetMember, 0, 4)
	for _, name := range []string{
		"experiment.json", "runtime-configuration.json", "experiment-run.json", "raw-evidence.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, "artifacts", name))
		require.NoError(t, err)
		members = append(members, artifact.SetMember{Path: "artifacts/" + name, Encoded: encoded})
	}
	input, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	execution, ok := input.Execution()
	require.True(t, ok)

	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	configuration := execution.RuntimeConfiguration()
	request.Mapping = definitionReference{
		DefinitionID: configuration.Observation.MappingDefinitionID,
		BehaviorFingerprint: configuration.Observation.MappingBehaviorFingerprint,
	}
	require.Equal(t, "temporal.system.nexus.caller-closure.duplicate-delivery.observation-program",
		request.ObservationProgram.DefinitionID)
	require.Equal(t, "temporal.system.nexus.caller-closure.duplicate-delivery.mapping",
		request.Mapping.DefinitionID)

	stdout, stderr, runErr := runRealCheckerOutput(t, process, request)
	require.NoError(t, runErr, string(stderr))
	require.Empty(t, stderr)
	response, err := decodeCheckerResponse(stdout, request)
	require.NoError(t, err)
	require.Equal(t, "accepted", response.ObservationEvaluationStatus)
	require.Equal(t, "applied", response.ImplementationLinkStatus)
	require.Equal(t, "violated", response.SemanticStatus)
	require.Equal(t, "violated", response.QuerySummary.Status)
	require.Len(t, response.PropertyVerdicts, 1)
	verdict := response.PropertyVerdicts[0]
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
	require.NotNil(t, response.EvidenceBackedModelTrace)
	require.Equal(t, "temporal.system.nexus.caller-closure.duplicate-delivery.profile",
		response.EvidenceBackedModelTrace.ProfileDefinitionID)
	propertyEvidence := propertyEvidenceDefinitionIDs(verdict.Clauses)
	require.Contains(t, propertyEvidence, "umpire.runtime.fact.participant.fixture")
	require.Contains(t, propertyEvidence,
		"umpire.runtime.fact.participant.synthetic-duplicate.fixture")

	evidence, result, err := constructEvaluation(execution, request, response)
	require.NoError(t, err)
	require.Equal(t, "succeeded", result.OperationalStatus)
	require.Equal(t, "accepted", result.ObservationEvaluationStatus)
	require.Equal(t, "violated", result.SemanticStatus)
	first, err := execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)
	second, err := checkWithChecker(context.Background(), input, process.run)
	require.NoError(t, err)
	require.Equal(t, first.Identity(), second.Identity())
	require.Equal(t, first.ManifestBytes(), second.ManifestBytes())
}

func TestRealCheckerSiblingAdmitsExactAcceptedSet(t *testing.T) {
	process := realCheckerProcess(t)
	input := realAcceptedCallerClosureExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	response, err := process.run(context.Background(), request)
	require.NoError(t, err)
	evidence, result, err := constructEvaluation(execution, request, response)
	require.NoError(t, err)
	require.Equal(t, "applied", result.ImplementationLinkStatus)
	require.Equal(t, "temporal.system.nexus.caller-closure.implementation-link",
		result.ImplementationLink.DefinitionID)
	require.Equal(t, "sha256:96b55d0e5a782099f66479c6ced603c08c8046b565f89435b5b2a54848aed777",
		result.ImplementationLink.BehaviorFingerprint)
	require.Equal(t, "temporal.system.nexus.caller-closure.target",
		result.ImplementationLink.SourceTarget.DefinitionID)
	require.Equal(t, "workflow-nexus.target.caller-closure",
		result.ImplementationLink.DestinationTarget.DefinitionID)
	require.Len(t, evidence.EvidenceLinks, 7)
	require.Equal(t, "temporal.system.nexus.caller-closure.profile",
		evidence.EvidenceBackedModelTrace.ProfileDefinitionID)
	staleEvidence := evidence
	staleEvidence.EvidenceLinks = append([]artifactv2.EvidenceLink(nil), evidence.EvidenceLinks...)
	staleEvidence.EvidenceLinks[0].OrderingSupport = append(
		[]artifactv2.EvidenceOrderingFact(nil), evidence.EvidenceLinks[0].OrderingSupport...,
	)
	staleEvidence.EvidenceLinks[0].OrderingSupport[0].Ordinal = artifactv2.NaturalFromUint64(1)
	require.ErrorContains(t, artifactv2.ValidateEvidenceClosure(
		staleEvidence,
		execution.Experiment(),
		execution.RuntimeConfiguration(),
		execution.ExperimentRun(),
		execution.RawEvidence(),
	), "ordering support")

	first, err := execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)
	second, err := checkWithChecker(context.Background(), input, process.run)
	require.NoError(t, err)
	require.Equal(t, first.Identity(), second.Identity())
	require.Equal(t, first.ManifestBytes(), second.ManifestBytes())
	requireRealCheckerNotRunning(t, process)
}

func realAcceptedCallerClosureExecutionFixture(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	partial := callerClosureExecutionFixture(t)
	execution, ok := partial.Execution()
	require.True(t, ok)
	run := execution.ExperimentRun()
	rawEvidence := execution.RawEvidence()
	one := artifactv2.NaturalFromUint64(1)
	receiptID := "umpire.runtime.fact.control.fixture"
	run.OperationalStatus = "succeeded"
	run.ControlAttempts = []artifactv2.ControlAttempt{{
		OccurrenceDefinitionID:  "workflow-nexus.occurrence.force-close",
		ActionDefinitionID:      "workflow.action.force-close",
		Attempt:                 one,
		ReceiptFactDefinitionID: &receiptID,
		Status:                  "accepted",
	}}
	run.SourceClosures = mutationSourceClosures("closed")
	sealedRun, err := artifactv2.SealExperimentRun(run)
	require.NoError(t, err)
	rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(sealedRun)
	rawEvidence.CaptureStatus = "closed"
	rawEvidence.Sources = mutationSources("closed")
	rawEvidence.Facts = exactCallerClosureFacts(rawEvidence.RunIdentity)
	sealedRawEvidence, err := artifactv2.SealRawEvidence(rawEvidence)
	require.NoError(t, err)
	executableInput := callerClosureExecutableFixture(t)
	executable, ok := executableInput.Executable()
	require.True(t, ok)
	accepted, err := executable.AdmitExecution(sealedRun, sealedRawEvidence)
	require.NoError(t, err)
	return accepted
}

func TestRealCheckerCancellationPublishesNoPartialSet(t *testing.T) {
	process := realCheckerProcess(t)
	input := callerClosureExecutionFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, err := checkWithChecker(ctx, input, process.run)
	require.Error(t, err)
	require.ErrorIs(t, err, &checkerFailure{code: checkerFailureCanceled})
	require.Empty(t, output.Identity())
	requireRealCheckerNotRunning(t, process)
}

func requireRealCheckerNotRunning(t *testing.T, process checkerProcess) {
	t.Helper()
	checker, err := resolveCheckerSibling(process.controllerExecutable)
	require.NoError(t, err)
	processes, err := exec.Command("ps", "-ax", "-o", "command=").Output()
	require.NoError(t, err)
	for line := range strings.SplitSeq(string(processes), "\n") {
		require.NotEqual(t, checker, strings.TrimSpace(line))
	}
}

func realCheckerProcess(t *testing.T) checkerProcess {
	t.Helper()
	realCheckerBuildOnce.Do(func() {
		modelDirectory, err := filepath.Abs(filepath.Join("..", "..", "..", "model"))
		if err != nil {
			realCheckerBuildErr = err
			return
		}
		command := exec.Command("mise", "exec", "--", "lake", "build", checkerExecutableName)
		command.Dir = modelDirectory
		realCheckerBuildLog, realCheckerBuildErr = command.CombinedOutput()
		realCheckerBuiltPath = filepath.Join(
			modelDirectory, ".lake", "build", "bin", checkerExecutableName,
		)
	})
	require.NoError(t, realCheckerBuildErr, string(realCheckerBuildLog))

	controller := testController(t, "real-checker")
	checker := filepath.Join(filepath.Dir(controller), checkerExecutableName)
	executable, err := os.ReadFile(realCheckerBuiltPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checker, executable, 0o700))
	return checkerProcess{controllerExecutable: controller, timeout: checkerTimeout}
}

func runRealCheckerBytes(
	t *testing.T,
	process checkerProcess,
	request checkerRequest,
) []byte {
	t.Helper()
	stdout, stderr, err := runRealCheckerOutput(t, process, request)
	require.NoError(t, err, string(stderr))
	require.Empty(t, stderr)
	return stdout
}

func runRealCheckerOutput(
	t *testing.T,
	process checkerProcess,
	request checkerRequest,
) (standardOutput []byte, standardError []byte, runErr error) {
	t.Helper()
	encoded, err := encodeCheckerRequest(request)
	if err != nil {
		return nil, nil, err
	}
	checker, err := resolveCheckerSibling(process.controllerExecutable)
	if err != nil {
		return nil, nil, err
	}
	command := exec.Command(checker)
	command.Dir = filepath.Dir(checker)
	command.Env = []string{}
	command.Stdin = bytes.NewReader(encoded)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
