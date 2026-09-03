package portableevaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var parityFixtureOutput = flag.String(
	"parity-fixture-output",
	"",
	"write Lean-generated portable evaluation parity fixtures to this directory",
)

func TestGeneratePortableEvaluationParityFixtures(t *testing.T) {
	if *parityFixtureOutput == "" {
		t.Skip("fixture generation was not requested")
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	duplicateEvidenceBytes, err := os.ReadFile(filepath.Join(
		repositoryRoot,
		"tools", "umpire", "temporal", "nexus", "testdata",
		"caller-closure-duplicate-delivery-run-set", "artifacts", "raw-evidence.json",
	))
	require.NoError(t, err)
	duplicateEvidence, err := artifact.DecodeRawEvidenceV2(duplicateEvidenceBytes)
	require.NoError(t, err)
	var normalEvidence artifactv2.RawEvidence

	for _, name := range []string{"normal", "duplicate-delivery", "any-operator"} {
		t.Run(name, func(t *testing.T) {
			command := exec.Command(
				"mise", "exec", "--", "lake", "env", "lean", "--run",
				"Temporal/Tool/PortableEvaluationContractTests.lean", name,
			)
			command.Dir = filepath.Join(repositoryRoot, "model")
			protoJSON, err := command.Output()
			require.NoError(t, err)
			contractBytes, err := evaluationcontract.Pack(protoJSON)
			if err != nil {
				decoded := new(umpirespb.EvaluationContract)
				require.NoError(t, protojson.Unmarshal(protoJSON, decoded))
				canonical, canonicalErr := evaluationcontract.CanonicalProtoJSON(decoded)
				require.NoError(t, canonicalErr)
				index := firstDifferentByte(protoJSON, canonical)
				t.Logf("ProtoJSON differs at byte %d: Lean %q; Go %q",
					index, byteWindow(protoJSON, index), byteWindow(canonical, index))
			}
			require.NoError(t, err)
			contract, err := evaluationcontract.Admit(contractBytes)
			require.NoError(t, err)

			evidence := duplicateEvidence
			var oracle map[string][]byte
			switch name {
			case "normal":
				evidence = projectEvidenceToContract(t, contract, duplicateEvidence)
				evidence, oracle = leanRunEvaluationOracle(t, repositoryRoot, name, evidence)
				normalEvidence = evidence
			case "duplicate-delivery":
				evidence, oracle = leanRunEvaluationOracle(t, repositoryRoot, name, evidence)
			case "any-operator":
				evidence = normalEvidence
			default:
				require.Failf(t, "unsupported parity fixture", "name=%q", name)
			}
			evidenceBytes, err := artifact.EncodeRawEvidenceV2(evidence)
			require.NoError(t, err)

			root := filepath.Join(*parityFixtureOutput, name)
			require.NoError(t, os.MkdirAll(root, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(root, "contract.pb"), contractBytes, 0o644))
			require.NoError(t,
				os.WriteFile(filepath.Join(root, "raw-evidence.json"), evidenceBytes, 0o644))
			for path, encoded := range oracle {
				require.NoError(t, os.WriteFile(filepath.Join(root, path), encoded, 0o644))
			}
		})
	}
	generateLeanRunBranchOracles(t, repositoryRoot, normalEvidence)
	generateLeanPortablePlans(t, repositoryRoot)
	command := exec.Command(
		"mise", "exec", "--", "lake", "env", "lean", "--run",
		"Temporal/Tool/PortableEvaluationContractTests.lean", "operator-branches",
	)
	command.Dir = filepath.Join(repositoryRoot, "model")
	branches, err := command.Output()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(*parityFixtureOutput, "operator-branches.json"), branches, 0o644,
	))
}

func generateLeanPortablePlans(t testing.TB, repositoryRoot string) {
	t.Helper()
	for _, name := range []string{"normal", "duplicate-delivery", "required-obligation"} {
		command := exec.Command(
			"mise", "exec", "--", "lake", "env", "lean", "--run",
			"Temporal/Tool/PortableEvaluationContractTests.lean", "portable-test-plan", name,
		)
		command.Dir = filepath.Join(repositoryRoot, "model")
		protoJSON, err := command.Output()
		require.NoError(t, err)
		plan := new(umpirespb.PortableTestPlan)
		require.NoError(t, protojson.Unmarshal(protoJSON, plan))
		plan, err = testplan.Seal(plan)
		require.NoError(t, err)
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(plan)
		require.NoError(t, err)
		root := filepath.Join(*parityFixtureOutput, "portable-test-plan-v1", name)
		require.NoError(t, os.MkdirAll(root, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "plan.pb"), encoded, 0o644))
	}
}

func generateLeanRunBranchOracles(
	t *testing.T,
	repositoryRoot string,
	baseline artifactv2.RawEvidence,
) {
	t.Helper()
	tests := []struct {
		name   string
		mutate func(testing.TB, *artifactv2.RawEvidence)
	}{
		{
			name: "correlation-conflict",
			mutate: func(t testing.TB, evidence *artifactv2.RawEvidence) {
				setRawField(t, evidence, "umpire.evidence.kind.workflow-history-event",
					"umpire.evidence.field.operation-correlation-id",
					"runtime.correlation.operation.conflict")
			},
		},
	}
	for _, test := range tests {
		t.Run("run-branch-"+test.name, func(t *testing.T) {
			evidence := cloneParityEvidence(t, baseline)
			test.mutate(t, &evidence)
			evidence = resealRawEvidence(t, evidence)
			_, oracle := leanRunEvaluationOracle(t, repositoryRoot, "normal", evidence)
			root := filepath.Join(*parityFixtureOutput, "run-branches", test.name)
			require.NoError(t, os.MkdirAll(root, 0o755))
			for path, encoded := range oracle {
				require.NoError(t, os.WriteFile(filepath.Join(root, path), encoded, 0o644))
			}
		})
	}
}

func cloneParityEvidence(t testing.TB, evidence artifactv2.RawEvidence) artifactv2.RawEvidence {
	t.Helper()
	encoded, err := artifact.EncodeRawEvidenceV2(evidence)
	require.NoError(t, err)
	cloned, err := artifact.DecodeRawEvidenceV2(encoded)
	require.NoError(t, err)
	return cloned
}

type leanEvaluationSummary struct {
	Destination string `json:"destination"`
}

type leanBranchOracle struct {
	Name                     string `json:"name"`
	Source                   string `json:"source"`
	ToolingStatus            string `json:"toolingStatus"`
	OperationalStatus        string `json:"operationalStatus"`
	ObservationStatus        string `json:"observationStatus"`
	ImplementationLinkStatus string `json:"implementationLinkStatus"`
	SemanticStatus           string `json:"semanticStatus"`
	CleanupStatus            string `json:"cleanupStatus"`
	Decision                 string `json:"decision"`
	DiagnosticCode           string `json:"diagnosticCode"`
}

func loadBranchOracles(t testing.TB) map[string]leanBranchOracle {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", "operator-branches.json"))
	require.NoError(t, err)
	var values []leanBranchOracle
	require.NoError(t, json.Unmarshal(encoded, &values))
	result := make(map[string]leanBranchOracle, len(values))
	for _, value := range values {
		require.NotEmpty(t, value.Name)
		require.NotEmpty(t, value.Source)
		_, duplicate := result[value.Name]
		require.False(t, duplicate, value.Name)
		result[value.Name] = value
	}
	return result
}

func loadLeanRunBranchOracle(
	t testing.TB,
	name string,
) (artifactv2.Evidence, artifactv2.Result) {
	t.Helper()
	root := filepath.Join("testdata", "run-branches", name)
	evidenceBytes, err := os.ReadFile(filepath.Join(root, "lean-evidence.json"))
	require.NoError(t, err)
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	resultBytes, err := os.ReadFile(filepath.Join(root, "lean-result.json"))
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	return evidence, result
}

func requireLeanRunBranchStatusParity(
	t testing.TB,
	actual *umpirespb.EvaluationResult,
	evidence artifactv2.Evidence,
	result artifactv2.Result,
) {
	t.Helper()
	require.Equal(t, result.OperationalStatus, operationalStatusName(actual.GetOperationalStatus()))
	require.Equal(t, evidence.ObservationEvaluationStatus,
		observationStatusName(actual.GetObservation().GetStatus()))
	require.Equal(t, result.ImplementationLinkStatus,
		implementationLinkStatusName(actual.GetImplementationLink().GetStatus()))
	require.Equal(t, result.SemanticStatus, evaluationStatusName(actual.GetSemanticStatus()))
	require.Equal(t, result.CleanupStatus, cleanupStatusName(actual.GetCleanupStatus()))
}

func requireBranchOracle(
	t testing.TB,
	oracle leanBranchOracle,
	result *umpirespb.EvaluationResult,
) {
	t.Helper()
	require.NotEmpty(t, oracle.Name)
	tooling, ok := umpirespb.ToolingStatus_value[oracle.ToolingStatus]
	require.True(t, ok)
	operational, ok := umpirespb.OperationalStatus_value[oracle.OperationalStatus]
	require.True(t, ok)
	observation, ok := umpirespb.ObservationStatus_value[oracle.ObservationStatus]
	require.True(t, ok)
	link, ok := umpirespb.ImplementationLinkStatus_value[oracle.ImplementationLinkStatus]
	require.True(t, ok)
	semantic, ok := umpirespb.EvaluationStatus_value[oracle.SemanticStatus]
	require.True(t, ok)
	cleanup, ok := umpirespb.CleanupStatus_value[oracle.CleanupStatus]
	require.True(t, ok)
	decision, ok := umpirespb.CanaryDecision_value[oracle.Decision]
	require.True(t, ok)
	require.Equal(t, umpirespb.ToolingStatus(tooling), result.GetToolingStatus())
	require.Equal(t, umpirespb.OperationalStatus(operational), result.GetOperationalStatus())
	require.Equal(t, umpirespb.ObservationStatus(observation), result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.ImplementationLinkStatus(link),
		result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.EvaluationStatus(semantic), result.GetSemanticStatus())
	require.Equal(t, umpirespb.CleanupStatus(cleanup), result.GetCleanupStatus())
	require.Equal(t, umpirespb.CanaryDecision(decision), result.GetDecision())
	if oracle.DiagnosticCode != "" {
		diagnostic, ok := umpirespb.DiagnosticCode_value[oracle.DiagnosticCode]
		require.True(t, ok)
		require.Contains(t, allDiagnostics(result), umpirespb.DiagnosticCode(diagnostic))
	}
}

func leanRunEvaluationOracle(
	t testing.TB,
	repositoryRoot string,
	name string,
	evidence artifactv2.RawEvidence,
) (artifactv2.RawEvidence, map[string][]byte) {
	t.Helper()
	setRoot := filepath.Join(
		repositoryRoot,
		"tools", "umpire", "temporal", "nexus", "testdata",
		"caller-closure-duplicate-delivery-run-set",
	)
	if name == "normal" {
		inputRoot := filepath.Join(
			repositoryRoot,
			"tools", "umpire", "temporal", "nexus", "testdata", "caller-closure-input-set",
		)
		members := make([]artifact.SetMember, 0, 2)
		for _, path := range []string{"experiment.json", "runtime-configuration.json"} {
			encoded, err := os.ReadFile(filepath.Join(inputRoot, "artifacts", path))
			require.NoError(t, err)
			members = append(members, artifact.SetMember{Path: "artifacts/" + path, Encoded: encoded})
		}
		admitted, err := artifact.AdmitSet(members)
		require.NoError(t, err)
		executable, ok := admitted.Executable()
		require.True(t, ok)
		runBytes, err := os.ReadFile(filepath.Join(setRoot, "artifacts", "experiment-run.json"))
		require.NoError(t, err)
		run, err := artifact.DecodeExperimentRunV2(runBytes)
		require.NoError(t, err)
		run.Experiment, err = artifactv2.ExperimentArtifactBinding(executable.Experiment())
		require.NoError(t, err)
		run.RuntimeConfiguration =
			artifactv2.RuntimeConfigurationArtifactBinding(executable.RuntimeConfiguration())
		run.SourceClosures = make([]artifactv2.SourceClosure, len(evidence.Sources))
		for index, source := range evidence.Sources {
			run.SourceClosures[index] = artifactv2.SourceClosure{
				SourceDefinitionID: source.SourceDefinitionID,
				Status:             source.Status,
				RecordCount:        source.FactCount,
				ByteCount:          source.ByteCount,
			}
		}
		run, err = artifactv2.SealExperimentRun(run)
		require.NoError(t, err)
		evidence.Run = artifactv2.ExperimentRunArtifactBinding(run)
		evidence, err = artifactv2.SealRawEvidence(evidence)
		require.NoError(t, err)
		execution, err := executable.AdmitExecution(run, evidence)
		require.NoError(t, err)
		setRoot, err = artifact.PublishSet(parityTempDir(t, repositoryRoot), execution)
		require.NoError(t, err)
	}

	outputRoot := parityTempDir(t, repositoryRoot)
	command := exec.Command(
		"make", "-C", repositoryRoot, "--no-print-directory",
		"umpire-check-local-run-evaluation", "SET="+setRoot, "OUTPUT_ROOT="+outputRoot,
	)
	stdout, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		require.ErrorAs(t, err, &exitError)
		require.Equal(t, 2, exitError.ExitCode(), string(exitError.Stderr))
	}
	var summary leanEvaluationSummary
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(stdout), &summary))
	require.NotEmpty(t, summary.Destination)
	oracle := make(map[string][]byte, 2)
	for _, path := range []string{"evidence.json", "result.json"} {
		encoded, readErr := os.ReadFile(filepath.Join(summary.Destination, "artifacts", path))
		require.NoError(t, readErr)
		oracle["lean-"+path] = encoded
	}
	return evidence, oracle
}

func parityTempDir(t testing.TB, repositoryRoot string) string {
	t.Helper()
	temporaryRoot := filepath.Join(repositoryRoot, ".flow", "tmp")
	require.NoError(t, os.MkdirAll(temporaryRoot, 0o755))
	directory, err := os.MkdirTemp(temporaryRoot, "portable-evaluation-parity.")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(directory))
	})
	return directory
}

func firstDifferentByte(left, right []byte) int {
	limit := min(len(left), len(right))
	if bytes.Equal(left, right) {
		return limit
	}
	for index := range limit {
		if left[index] != right[index] {
			return index
		}
	}
	return limit
}

func byteWindow(encoded []byte, index int) string {
	start := max(0, index-80)
	end := min(len(encoded), index+160)
	return string(encoded[start:end])
}

func projectEvidenceToContract(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
	evidence artifactv2.RawEvidence,
) artifactv2.RawEvidence {
	t.Helper()
	allowedFields := make(map[string]map[string]struct{})
	for _, kind := range contract.GetObservation().GetProfile().GetKinds() {
		fields := make(map[string]struct{}, len(kind.GetFields()))
		for _, field := range kind.GetFields() {
			fields[field.GetFieldDefinitionId()] = struct{}{}
		}
		allowedFields[kind.GetKindDefinitionId()] = fields
	}
	projectedFacts := make([]artifactv2.RawEvidenceFact, 0, len(evidence.Facts))
	evidence.Sources = append([]artifactv2.RawEvidenceSource(nil), evidence.Sources...)
	for _, fact := range evidence.Facts {
		fields, ok := allowedFields[fact.KindDefinitionID]
		if !ok || fact.FactDefinitionID == "umpire.runtime.fact.participant.synthetic-duplicate.fixture" {
			continue
		}
		projected := fact
		projected.Fields = nil
		for _, field := range fact.Fields {
			if _, ok := fields[field.FieldDefinitionID]; ok {
				projected.Fields = append(projected.Fields, field)
			}
		}
		projectedFacts = append(projectedFacts, projected)
	}
	for index := range projectedFacts {
		if !rawEvidenceFactHasField(
			projectedFacts[index], "umpire.evidence.field.cancellation-callback-count",
		) {
			continue
		}
		for _, kind := range contract.GetObservation().GetProfile().GetKinds() {
			if kind.GetKindDefinitionId() != projectedFacts[index].KindDefinitionID {
				continue
			}
			for _, declaration := range kind.GetFields() {
				if declaration.GetDisposition() != umpirespb.FIELD_DISPOSITION_KIND_HASH ||
					rawEvidenceFactHasField(projectedFacts[index], declaration.GetFieldDefinitionId()) {
					continue
				}
				digest := sha256.Sum256([]byte(declaration.GetFieldDefinitionId()))
				projectedFacts[index].Fields = append(projectedFacts[index].Fields,
					artifactv2.RawEvidenceField{
						FieldDefinitionID: declaration.GetFieldDefinitionId(),
						Disposition:       "sha256",
						Value:             fmt.Sprintf("sha256:%x", digest),
					})
			}
		}
		slices.SortFunc(projectedFacts[index].Fields, func(left, right artifactv2.RawEvidenceField) int {
			return strings.Compare(left.FieldDefinitionID, right.FieldDefinitionID)
		})
	}
	evidence.Experiment = artifactBinding(contract.GetExperiment())
	evidence.RuntimeConfiguration = artifactBinding(contract.GetRuntimeConfig())
	evidence.Facts = projectedFacts
	for index := range evidence.Sources {
		count := uint64(0)
		for _, fact := range evidence.Facts {
			if fact.SourceDefinitionID == evidence.Sources[index].SourceDefinitionID {
				count++
			}
		}
		evidence.Sources[index].FactCount = artifactv2.NaturalFromUint64(count)
	}
	sealed, err := artifactv2.SealRawEvidence(evidence)
	require.NoError(t, err)
	require.NoError(t, artifactv2.ValidateRawEvidence(sealed))
	return sealed
}

func artifactBinding(binding *umpirespb.ArtifactBinding) artifactv2.ArtifactBinding {
	return artifactv2.ArtifactBinding{
		FormatVersion: binding.GetFormatVersion(), ArtifactChecksum: binding.GetArtifactChecksum(),
		BehaviorFingerprint: binding.GetBehaviorFingerprint(),
		ProvenanceChecksum:  binding.GetProvenanceChecksum(),
	}
}

func rawEvidenceFactHasField(fact artifactv2.RawEvidenceFact, definitionID string) bool {
	return slices.ContainsFunc(fact.Fields, func(field artifactv2.RawEvidenceField) bool {
		return field.FieldDefinitionID == definitionID
	})
}

func TestPortableEvaluatorMatchesLeanRunEvaluationFixtures(t *testing.T) {
	tests := []struct {
		name        string
		semantic    umpirespb.EvaluationStatus
		decision    umpirespb.CanaryDecision
		clauseState []umpirespb.SemanticStatus
	}{
		{
			name: "normal", semantic: umpirespb.EVALUATION_STATUS_SATISFIED,
			decision: umpirespb.CANARY_DECISION_PASS,
			clauseState: []umpirespb.SemanticStatus{
				umpirespb.SEMANTIC_STATUS_SATISFIED,
				umpirespb.SEMANTIC_STATUS_SATISFIED,
				umpirespb.SEMANTIC_STATUS_SATISFIED,
			},
		},
		{
			name: "duplicate-delivery", semantic: umpirespb.EVALUATION_STATUS_VIOLATED,
			decision: umpirespb.CANARY_DECISION_FAIL,
			clauseState: []umpirespb.SemanticStatus{
				umpirespb.SEMANTIC_STATUS_SATISFIED,
				umpirespb.SEMANTIC_STATUS_SATISFIED,
				umpirespb.SEMANTIC_STATUS_VIOLATED,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join("testdata", test.name)
			contractBytes, err := os.ReadFile(filepath.Join(root, "contract.pb"))
			require.NoError(t, err)
			contract, err := evaluationcontract.Admit(contractBytes)
			require.NoError(t, err)
			evidenceBytes, err := os.ReadFile(filepath.Join(root, "raw-evidence.json"))
			require.NoError(t, err)
			evidence, err := artifact.DecodeRawEvidenceV2(evidenceBytes)
			require.NoError(t, err)
			leanEvidenceBytes, err := os.ReadFile(filepath.Join(root, "lean-evidence.json"))
			require.NoError(t, err)
			leanEvidence, err := artifact.DecodeEvidenceV2(leanEvidenceBytes)
			require.NoError(t, err)
			leanResultBytes, err := os.ReadFile(filepath.Join(root, "lean-result.json"))
			require.NoError(t, err)
			leanResult, err := artifact.DecodeResultV2(leanResultBytes)
			require.NoError(t, err)

			result := Evaluate(context.Background(), requestFor(contract, evidence))

			require.Equal(t, umpirespb.OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
			require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED,
				result.GetObservation().GetStatus(), result.GetObservation().GetDiagnostics())
			require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
				result.GetImplementationLink().GetStatus(), result.GetImplementationLink().GetDiagnostics())
			require.Equal(t, test.semantic, result.GetSemanticStatus())
			require.Equal(t, test.decision, result.GetDecision())
			require.Len(t, result.GetProperties(), 1)
			require.Len(t, result.GetProperties()[0].GetClauses(), len(test.clauseState))
			for index, state := range test.clauseState {
				require.Equal(t, state, result.GetProperties()[0].GetClauses()[index].GetStatus())
			}
			requireLeanRunEvaluationParity(t, contract, result, leanEvidence, leanResult)
		})
	}
}

func TestLeanGeneratedPortablePlansUseSharedAdmissionAndRetainExactBindings(t *testing.T) {
	tests := []struct {
		name        string
		contract    string
		obligations int
	}{
		{name: "normal", contract: "normal"},
		{name: "duplicate-delivery", contract: "duplicate-delivery"},
		{name: "required-obligation", contract: "normal", obligations: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := os.ReadFile(filepath.Join(
				"testdata", "portable-test-plan-v1", test.name, "plan.pb",
			))
			require.NoError(t, err)
			plan := new(umpirespb.PortableTestPlan)
			require.NoError(t, proto.Unmarshal(encoded, plan))
			admitted, err := testplan.Admit(plan)
			require.NoError(t, err)
			contract, _ := loadParityFixture(t, test.contract)

			protorequire.ProtoEqual(t, contract.GetQuery(), plan.GetModelCompiled().GetQuery())
			protorequire.ProtoEqual(t, contract.GetObservation().GetProfile(), plan.GetVerification().GetEvidence())
			if test.obligations == 0 {
				protorequire.ProtoEqual(t, contract.GetObservation(), plan.GetVerification().GetObservation())
			}
			protorequire.ProtoEqual(t, contract.GetImplementationLink(), plan.GetVerification().GetRenameExactLink())
			require.Len(t, plan.GetVerification().GetProperties(), len(contract.GetProperties()))
			for index, property := range contract.GetProperties() {
				protorequire.ProtoEqual(t, property, plan.GetVerification().GetProperties()[index])
			}
			require.Len(t, plan.GetExecution().GetRequestedActions(), 1)
			require.Len(t, plan.GetExecution().GetModelOutcomes(), 1)
			require.Len(t, plan.GetExecution().GetResultingStates(), 1)
			require.Len(t, plan.GetExecution().GetOccurrences(), 1)
			require.Len(t, plan.GetExternalObligations(), test.obligations)

			verified := testplan.ModelProvenanceBinding{
				PlanChecksum: admitted.Checksum(), ModelCompiled: proto.CloneOf(plan.GetModelCompiled()),
			}
			authorized, err := testplan.Authorize(
				context.Background(), admitted,
				func(context.Context, testplan.ModelProvenanceBinding) (testplan.ModelProvenanceBinding, error) {
					return verified, nil
				},
			)
			require.NoError(t, err)
			result, err := authorized.ScopeResult(successfulPortablePlanResult())
			require.NoError(t, err)
			require.Equal(t, umpirespb.CLAIM_SCOPE_MODEL_BOUND, result.GetClaimScope())
			if test.obligations == 0 {
				require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
			} else {
				require.Equal(t, umpirespb.EXECUTION_DECISION_INCONCLUSIVE, result.GetDecision())
				require.Len(t, result.GetUnresolvedExternalObligations(), test.obligations)
				for _, obligation := range result.GetUnresolvedExternalObligations() {
					require.Equal(t, umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_REQUIRED,
						obligation.GetKind())
				}
			}
		})
	}

	t.Run("external plans remain plan local", func(t *testing.T) {
		plan := loadLeanPortablePlan(t, "normal")
		plan.Provenance = &umpirespb.PortableTestPlan_External{External: &umpirespb.ExternalPlanProvenance{
			Sources: []*umpirespb.SourceLocation{proto.CloneOf(plan.GetModelCompiled().GetSources()[0])},
		}}
		plan, err := testplan.Seal(plan)
		require.NoError(t, err)
		admitted, err := testplan.Admit(plan)
		require.NoError(t, err)
		authorized, err := testplan.Authorize(context.Background(), admitted, nil)
		require.NoError(t, err)
		result, err := authorized.ScopeResult(successfulPortablePlanResult())
		require.NoError(t, err)
		require.Equal(t, umpirespb.PROVENANCE_OUTCOME_EXTERNAL, result.GetProvenanceOutcome())
		require.Equal(t, umpirespb.CLAIM_SCOPE_PLAN_LOCAL, result.GetClaimScope())
		require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
	})
}

func TestLeanGeneratedPortablePlanRejectsChecksumBindingSourceAndLimitMutations(t *testing.T) {
	plan := loadLeanPortablePlan(t, "normal")
	original, err := testplan.Admit(plan)
	require.NoError(t, err)
	trusted := testplan.ModelProvenanceBinding{
		PlanChecksum: original.Checksum(), ModelCompiled: proto.CloneOf(plan.GetModelCompiled()),
	}

	t.Run("checksum", func(t *testing.T) {
		mutated := proto.CloneOf(plan)
		mutated.PlanChecksum[0] ^= 0xff
		_, err := testplan.Admit(mutated)
		requirePlanAdmissionCode(t, err, testplan.ErrorChecksum)
	})
	t.Run("crossed binding", func(t *testing.T) {
		mutated := proto.CloneOf(plan)
		mutated.Execution.Query.BehaviorFingerprint =
			"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		_, err := testplan.Seal(mutated)
		requirePlanAdmissionCode(t, err, testplan.ErrorBinding)
	})
	for _, test := range []struct {
		name   string
		mutate func(*umpirespb.PortableTestPlan)
	}{
		{
			name: "source",
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetModelCompiled().Sources[0].Path = "Mutation.lean"
			},
		},
		{
			name: "compiler binding",
			mutate: func(plan *umpirespb.PortableTestPlan) {
				plan.GetModelCompiled().CompilerContract.BehaviorFingerprint =
					"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
		},
		{
			name: "obligation",
			mutate: func(plan *umpirespb.PortableTestPlan) {
				required := loadLeanPortablePlan(t, "required-obligation").GetExternalObligations()
				require.Len(t, required, 3)
				plan.ExternalObligations = []*umpirespb.ExternalVerificationObligation{
					proto.CloneOf(required[0]),
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := proto.CloneOf(plan)
			test.mutate(mutated)
			mutated, err := testplan.Seal(mutated)
			require.NoError(t, err)
			admitted, err := testplan.Admit(mutated)
			require.NoError(t, err)
			_, err = testplan.Authorize(
				context.Background(), admitted,
				func(context.Context, testplan.ModelProvenanceBinding) (testplan.ModelProvenanceBinding, error) {
					return trusted, nil
				},
			)
			requirePlanAdmissionCode(t, err, testplan.ErrorProvenance)
		})
	}
	t.Run("action N plus one", func(t *testing.T) {
		mutated := proto.CloneOf(plan)
		mutated.Execution.RequestedActions = append(
			mutated.Execution.RequestedActions,
			proto.CloneOf(mutated.Execution.RequestedActions[0]),
		)
		_, err := testplan.Seal(mutated)
		requirePlanAdmissionCode(t, err, testplan.ErrorLimit)
	})
	t.Run("mandatory result N and N plus one", func(t *testing.T) {
		atLimit := proto.CloneOf(plan)
		atLimit.GetLimits().GetOutput().MaxDiagnosticBytes = 256
		atLimit, err = testplan.Seal(atLimit)
		require.NoError(t, err)
		admitted, err := testplan.Admit(atLimit)
		require.NoError(t, err)
		for atLimit.GetLimits().GetOutput().GetMaxResultBytes() != int64(admitted.MandatoryResultBytes()) {
			atLimit.GetLimits().GetOutput().MaxResultBytes = int64(admitted.MandatoryResultBytes())
			atLimit, err = testplan.Seal(atLimit)
			require.NoError(t, err)
			admitted, err = testplan.Admit(atLimit)
			require.NoError(t, err)
		}
		require.Equal(t, atLimit.GetLimits().GetOutput().GetMaxResultBytes(),
			int64(admitted.MandatoryResultBytes()))

		beyondLimit := proto.CloneOf(atLimit)
		beyondLimit.GetLimits().GetOutput().MaxResultBytes--
		_, err = testplan.Seal(beyondLimit)
		requirePlanAdmissionCode(t, err, testplan.ErrorLimit)
	})
}

func loadLeanPortablePlan(t testing.TB, name string) *umpirespb.PortableTestPlan {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"testdata", "portable-test-plan-v1", name, "plan.pb",
	))
	require.NoError(t, err)
	plan := new(umpirespb.PortableTestPlan)
	require.NoError(t, proto.Unmarshal(encoded, plan))
	return plan
}

func successfulPortablePlanResult() *umpirespb.ExecutionResult {
	return &umpirespb.ExecutionResult{
		ToolingStatus:     umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED,
		OperationalStatus: umpirespb.EXECUTION_OPERATIONAL_STATUS_SUCCEEDED,
		Observation:       &umpirespb.ObservationEvaluationResult{Status: umpirespb.OBSERVATION_STATUS_ACCEPTED},
		TraceProjection:   &umpirespb.TraceProjectionResult{Status: umpirespb.TRACE_PROJECTION_STATUS_APPLIED},
		SemanticStatus:    umpirespb.EXECUTION_EVALUATION_STATUS_SATISFIED,
		CleanupStatus:     umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE,
		Decision:          umpirespb.EXECUTION_DECISION_PASS,
		Work:              &umpirespb.EvaluationWork{},
	}
}

func requirePlanAdmissionCode(t testing.TB, err error, want testplan.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	got, ok := testplan.CodeOf(err)
	require.True(t, ok)
	require.Equal(t, want, got)
}

func TestLeanParityContractsCoverV1OperatorVocabulary(t *testing.T) {
	operators := make(map[string]bool)
	patterns := make(map[string]bool)
	for _, name := range []string{"normal", "duplicate-delivery", "any-operator"} {
		contract, _ := loadParityFixture(t, name)
		for _, emit := range contract.GetObservation().GetEmits() {
			collectObservationOperators(t, operators, emit.GetCondition())
			collectObservationOperators(t, operators, emit.GetValue())
		}
		for _, property := range contract.GetProperties() {
			for _, clause := range property.GetClauses() {
				patterns["per_step_implies"] = clause.GetPerStepImplies() != nil
				for _, pattern := range []*umpirespb.Pattern{
					clause.GetPerStepImplies().GetTrigger(), clause.GetPerStepImplies().GetRequired(),
				} {
					switch pattern.GetOperator().(type) {
					case *umpirespb.Pattern_EqualsText:
						patterns["equals_text"] = true
					case *umpirespb.Pattern_NaturalAtMost:
						patterns["natural_at_most"] = true
					default:
						require.Failf(t, "unsupported Property pattern operator", "operator=%T",
							pattern.GetOperator())
					}
				}
			}
		}
		require.NotEmpty(t, contract.GetImplementationLink().GetEntries())
		patterns["rename_exact"] = true
	}
	require.Equal(t, map[string]bool{
		"literal_text": true, "literal_natural": true, "field": true,
		"natural_render_v1": true, "present": true, "equals": true,
		"all": true, "any": true,
	}, operators)
	require.Equal(t, map[string]bool{
		"rename_exact": true, "per_step_implies": true,
		"equals_text": true, "natural_at_most": true,
	}, patterns)
}

func TestLeanParityContractsCoverFalseMissingAndTypeErrorBranches(t *testing.T) {
	oracles := loadBranchOracles(t)
	tests := []struct {
		name    string
		fixture string
		mutate  func(testing.TB, *umpirespb.EvaluationContract, *artifactv2.RawEvidence)
	}{
		{
			name: "any true", fixture: "any-operator",
			mutate: func(testing.TB, *umpirespb.EvaluationContract, *artifactv2.RawEvidence) {
			},
		},
		{
			name: "all false", fixture: "normal",
			mutate: func(t testing.TB, _ *umpirespb.EvaluationContract, evidence *artifactv2.RawEvidence) {
				setRawField(t, evidence, "umpire.evidence.kind.control-receipt",
					"umpire.evidence.field.action-definition-id", "workflow.action.other")
			},
		},
		{
			name: "any false", fixture: "any-operator",
			mutate: func(t testing.TB, _ *umpirespb.EvaluationContract, evidence *artifactv2.RawEvidence) {
				setRawField(t, evidence, "umpire.evidence.kind.control-receipt",
					"umpire.evidence.field.action-definition-id", "workflow.action.other")
			},
		},
		{
			name: "present false", fixture: "duplicate-delivery",
			mutate: func(_ testing.TB, _ *umpirespb.EvaluationContract, evidence *artifactv2.RawEvidence) {
				removeRawField(evidence,
					"umpire.evidence.kind.participant-command.synthetic-duplicate",
					"umpire.evidence.field.operation-correlation-id")
			},
		},
		{
			name: "field type error", fixture: "normal",
			mutate: func(t testing.TB, _ *umpirespb.EvaluationContract, evidence *artifactv2.RawEvidence) {
				setRawField(t, evidence, "umpire.evidence.kind.participant-command",
					"umpire.evidence.field.cancellation-callback-count", true)
			},
		},
		{
			name: "field missing", fixture: "normal",
			mutate: func(t testing.TB, contract *umpirespb.EvaluationContract, evidence *artifactv2.RawEvidence) {
				removeRawField(evidence, "umpire.evidence.kind.participant-command",
					"umpire.evidence.field.cancellation-callback-count")
				for _, emit := range contract.GetObservation().GetEmits() {
					if emit.GetValue().GetNaturalRenderV1() != nil {
						emit.Condition = equals(literalText("admitted"), literalText("admitted"))
						return
					}
				}
				require.Fail(t, "natural_render_v1 emit is absent")
			},
		},
		{
			name: "equals type error", fixture: "normal",
			mutate: func(_ testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				equals := contract.GetObservation().GetEmits()[0].GetCondition().GetAll().GetOperands()[0].GetEquals()
				equals.Right = &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_LiteralNatural{LiteralNatural: &umpirespb.LiteralNatural{Value: "1"}}}
			},
		},
		{
			name: "all type error", fixture: "normal",
			mutate: func(_ testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				contract.GetObservation().GetEmits()[0].GetCondition().GetAll().Operands[0] =
					literalText("not-a-boolean")
			},
		},
		{
			name: "any type error", fixture: "any-operator",
			mutate: func(_ testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				contract.GetObservation().GetEmits()[0].GetCondition().GetAny().Operands[0] =
					literalText("not-a-boolean")
			},
		},
		{
			name: "natural render type error", fixture: "normal",
			mutate: func(t testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				for _, emit := range contract.GetObservation().GetEmits() {
					if render := emit.GetValue().GetNaturalRenderV1(); render != nil {
						render.Operand = &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_LiteralText{LiteralText: &umpirespb.LiteralText{Value: "1"}}}
						return
					}
				}
				require.Fail(t, "natural_render_v1 emit is absent")
			},
		},
		{
			name: "equals text rejects natural", fixture: "normal",
			mutate: func(t testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				setDestinationValue(t, contract, "nexus.observation.cancellation-delivered",
					&umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "1"}})
			},
		},
		{
			name: "natural at most rejects text", fixture: "normal",
			mutate: func(t testing.TB, contract *umpirespb.EvaluationContract, _ *artifactv2.RawEvidence) {
				setDestinationValue(t, contract, "nexus.observation.pending-cancellation-count",
					&umpirespb.Value{Value: &umpirespb.Value_Text{Text: "1"}})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract, evidence := loadParityFixture(t, test.fixture)
			test.mutate(t, contract, &evidence)
			contract = repackParityContract(t, contract)
			evidence = resealRawEvidence(t, evidence)

			result := Evaluate(context.Background(), requestFor(contract, evidence))

			requireBranchOracle(t, oracles[test.name], result)
		})
	}
}

func TestCorrelationSlotsAllowOptionalReferencesAndRejectWhollyMissing(t *testing.T) {
	oracles := loadBranchOracles(t)
	contract, evidence := loadParityFixture(t, "normal")
	result := Evaluate(context.Background(), requestFor(contract, evidence))
	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())

	operationField := "umpire.evidence.field.operation-correlation-id"
	t.Run("production Run Evaluation conflict", func(t *testing.T) {
		conflicting := cloneParityEvidence(t, evidence)
		setRawField(t, &conflicting, "umpire.evidence.kind.workflow-history-event",
			operationField, "runtime.correlation.operation.conflict")
		conflicting = resealRawEvidence(t, conflicting)
		result := Evaluate(context.Background(), requestFor(contract, conflicting))
		leanEvidence, leanResult := loadLeanRunBranchOracle(t, "correlation-conflict")
		requireLeanRunBranchStatusParity(t, result, leanEvidence, leanResult)
	})
	t.Run("portable-only wholly missing optional slot", func(t *testing.T) {
		missing := cloneParityEvidence(t, evidence)
		for _, fact := range missing.Facts {
			removeRawField(&missing, fact.KindDefinitionID, operationField)
		}
		missing = resealRawEvidence(t, missing)
		result := Evaluate(context.Background(), requestFor(contract, missing))
		oracle := oracles["correlation missing"]
		require.Equal(t, "portable-v1-proof", oracle.Source)
		requireBranchOracle(t, oracle, result)
	})
}

func TestLeanParityContractsFailClosedOnCanonicalOrderAndCrossedPairs(t *testing.T) {
	for _, name := range []string{"normal", "duplicate-delivery"} {
		t.Run(name+" evidence order", func(t *testing.T) {
			contract, evidence := loadParityFixture(t, name)
			evidence.Facts[0], evidence.Facts[1] = evidence.Facts[1], evidence.Facts[0]
			var err error
			evidence, err = artifactv2.SealRawEvidence(evidence)
			require.NoError(t, err)
			result := Evaluate(context.Background(), requestFor(contract, evidence))
			require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_INPUT, result.GetToolingStatus())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
		})
		t.Run(name+" contract order", func(t *testing.T) {
			contract, _ := loadParityFixture(t, name)
			contract.ImplementationLink.Entries[0], contract.ImplementationLink.Entries[1] =
				contract.ImplementationLink.Entries[1], contract.ImplementationLink.Entries[0]
			contract.ArtifactChecksum = nil
			canonical, err := evaluationcontract.CanonicalProtoJSON(contract)
			require.NoError(t, err)
			_, err = evaluationcontract.Pack(canonical)
			require.Error(t, err)
		})
	}
	normalContract, normalEvidence := loadParityFixture(t, "normal")
	duplicateContract, duplicateEvidence := loadParityFixture(t, "duplicate-delivery")
	oracle := loadBranchOracles(t)["crossed pair"]
	for _, request := range []Request{
		requestFor(normalContract, duplicateEvidence), requestFor(duplicateContract, normalEvidence),
	} {
		result := Evaluate(context.Background(), request)
		requireBranchOracle(t, oracle, result)
	}
}

func TestLeanParityContractsEnforceExactWorkBoundary(t *testing.T) {
	oracle := loadBranchOracles(t)["work limit exceeded"]
	for _, name := range []string{"normal", "duplicate-delivery", "any-operator"} {
		t.Run(name, func(t *testing.T) {
			contract, evidence := loadParityFixture(t, name)
			baseline := Evaluate(context.Background(), requestFor(contract, evidence))
			require.Greater(t, baseline.GetWork().GetTotal(), int64(1))
			exactWork := baseline.GetWork().GetTotal()

			exact := proto.CloneOf(contract)
			exact.Limits.MaxEvaluationWork = exactWork
			exact = repackParityContract(t, exact)
			exactResult := Evaluate(context.Background(), requestFor(exact, evidence))
			require.Equal(t, baseline.GetDecision(), exactResult.GetDecision())
			require.Equal(t, exactWork, exactResult.GetWork().GetTotal())

			over := proto.CloneOf(contract)
			over.Limits.MaxEvaluationWork = exactWork - 1
			over = repackParityContract(t, over)
			overResult := Evaluate(context.Background(), requestFor(over, evidence))
			requireBranchOracle(t, oracle, overResult)
		})
	}
}

func TestLeanParityRawKnownGapIsRetainedAndInconclusive(t *testing.T) {
	oracle := loadBranchOracles(t)["raw known gap"]
	contract, evidence := loadParityFixture(t, "normal")
	subject := "temporal.run.gap"
	detail := "bounded Evidence omission"
	evidence.KnownGaps = []artifactv2.KnownGap{{
		Kind: "input", Code: "umpire.gap.parity", Subject: &subject, Detail: &detail,
	}}
	evidence = resealRawEvidence(t, evidence)

	result := Evaluate(context.Background(), requestFor(contract, evidence))

	requireBranchOracle(t, oracle, result)
	require.Equal(t, "umpire.gap.parity", result.GetKnownGaps()[0].GetCode())
	require.Len(t, result.GetKnownGaps(), len(contract.GetKnownGaps())+1)
}

func collectObservationOperators(
	t testing.TB,
	operators map[string]bool,
	expression *umpirespb.ObservationExpression,
) {
	t.Helper()
	switch operator := expression.GetOperator().(type) {
	case *umpirespb.ObservationExpression_LiteralText:
		operators["literal_text"] = true
	case *umpirespb.ObservationExpression_LiteralNatural:
		operators["literal_natural"] = true
	case *umpirespb.ObservationExpression_Field:
		operators["field"] = true
	case *umpirespb.ObservationExpression_NaturalRenderV1:
		operators["natural_render_v1"] = true
		collectObservationOperators(t, operators, operator.NaturalRenderV1.GetOperand())
	case *umpirespb.ObservationExpression_Present:
		operators["present"] = true
		collectObservationOperators(t, operators, operator.Present.GetOperand())
	case *umpirespb.ObservationExpression_Equals:
		operators["equals"] = true
		collectObservationOperators(t, operators, operator.Equals.GetLeft())
		collectObservationOperators(t, operators, operator.Equals.GetRight())
	case *umpirespb.ObservationExpression_All:
		operators["all"] = true
		for _, operand := range operator.All.GetOperands() {
			collectObservationOperators(t, operators, operand)
		}
	case *umpirespb.ObservationExpression_Any:
		operators["any"] = true
		for _, operand := range operator.Any.GetOperands() {
			collectObservationOperators(t, operators, operand)
		}
	default:
		require.Failf(t, "unsupported Observation operator", "operator=%T", expression.GetOperator())
	}
}

func loadParityFixture(
	t testing.TB,
	name string,
) (*umpirespb.EvaluationContract, artifactv2.RawEvidence) {
	t.Helper()
	root := filepath.Join("testdata", name)
	contractBytes, err := os.ReadFile(filepath.Join(root, "contract.pb"))
	require.NoError(t, err)
	contract, err := evaluationcontract.Admit(contractBytes)
	require.NoError(t, err)
	evidenceBytes, err := os.ReadFile(filepath.Join(root, "raw-evidence.json"))
	require.NoError(t, err)
	evidence, err := artifact.DecodeRawEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	return contract, evidence
}

func repackParityContract(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
) *umpirespb.EvaluationContract {
	t.Helper()
	contract.ArtifactChecksum = nil
	canonical, err := evaluationcontract.CanonicalProtoJSON(contract)
	require.NoError(t, err)
	encoded, err := evaluationcontract.Pack(canonical)
	require.NoError(t, err)
	admitted, err := evaluationcontract.Admit(encoded)
	require.NoError(t, err)
	return admitted
}

func setRawField(
	t testing.TB,
	evidence *artifactv2.RawEvidence,
	kindDefinitionID string,
	fieldDefinitionID string,
	value any,
) {
	t.Helper()
	for factIndex := range evidence.Facts {
		fact := &evidence.Facts[factIndex]
		if fact.KindDefinitionID != kindDefinitionID {
			continue
		}
		for fieldIndex := range fact.Fields {
			if fact.Fields[fieldIndex].FieldDefinitionID == fieldDefinitionID {
				fact.Fields[fieldIndex].Value = value
				return
			}
		}
	}
	require.Failf(t, "Raw Evidence field is absent", "kind=%q field=%q",
		kindDefinitionID, fieldDefinitionID)
}

func removeRawField(
	evidence *artifactv2.RawEvidence,
	kindDefinitionID string,
	fieldDefinitionID string,
) {
	for factIndex := range evidence.Facts {
		fact := &evidence.Facts[factIndex]
		if fact.KindDefinitionID != kindDefinitionID {
			continue
		}
		fact.Fields = slices.DeleteFunc(fact.Fields, func(field artifactv2.RawEvidenceField) bool {
			return field.FieldDefinitionID == fieldDefinitionID
		})
	}
}

func setDestinationValue(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
	definitionID string,
	value *umpirespb.Value,
) {
	t.Helper()
	for _, entry := range contract.GetImplementationLink().GetEntries() {
		if entry.GetDestination().GetDefinition().GetDefinitionId() == definitionID {
			entry.Destination.Value = value
			return
		}
	}
	require.Fail(t, "destination definition is absent", definitionID)
}

func requireLeanRunEvaluationParity(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
	result *umpirespb.EvaluationResult,
	leanEvidence artifactv2.Evidence,
	leanResult artifactv2.Result,
) {
	t.Helper()
	require.Equal(t, leanResult.OperationalStatus, operationalStatusName(result.GetOperationalStatus()))
	require.Equal(t, leanEvidence.ObservationEvaluationStatus,
		observationStatusName(result.GetObservation().GetStatus()))
	require.Equal(t, leanResult.ImplementationLinkStatus,
		implementationLinkStatusName(result.GetImplementationLink().GetStatus()))
	require.Equal(t, leanResult.SemanticStatus, evaluationStatusName(result.GetSemanticStatus()))
	require.Equal(t, leanResult.CleanupStatus, cleanupStatusName(result.GetCleanupStatus()))
	require.Equal(t, leanEvidence.ObservationProgram.DefinitionID,
		contract.GetObservation().GetDefinition().GetDefinitionId())
	require.Equal(t, leanEvidence.Mapping.DefinitionID,
		contract.GetObservation().GetMapping().GetDefinitionId())
	require.Equal(t, leanResult.ImplementationLink.DefinitionID,
		contract.GetImplementationLink().GetDefinition().GetDefinitionId())
	require.Equal(t, leanResult.ImplementationLink.BehaviorFingerprint,
		contract.GetImplementationLink().GetDefinition().GetBehaviorFingerprint())
	require.Equal(t, leanResult.ImplementationLink.SourceTarget.DefinitionID,
		contract.GetImplementationLink().GetSourceTarget().GetDefinitionId())
	require.Equal(t, leanResult.ImplementationLink.DestinationTarget.DefinitionID,
		contract.GetImplementationLink().GetDestinationTarget().GetDefinitionId())

	require.NotNil(t, leanEvidence.EvidenceBackedModelTrace)
	requireTraceParity(t, result.GetObservation().GetTrace(), leanEvidence.EvidenceBackedModelTrace.Trace)
	require.Len(t, result.GetObservation().GetEvidenceLinks(), len(leanEvidence.EvidenceLinks))
	for index, leanLink := range leanEvidence.EvidenceLinks {
		goLink := result.GetObservation().GetEvidenceLinks()[index]
		require.Equal(t, stableGoCoordinate(goLink.GetCoordinate()), stableLeanCoordinate(t, leanLink.Coordinate))
		require.Equal(t, leanLink.MappingDefinitionID, goLink.GetMapping().GetDefinitionId())
		require.Equal(t, leanLink.MappingBehaviorFingerprint,
			goLink.GetMapping().GetBehaviorFingerprint())
		require.Equal(t, leanLink.RuleDefinitionID, semanticRuleDefinitionID(goLink.GetRuleDefinitionId()))
		require.Equal(t, leanLink.EvidenceDefinitionIDs, goLink.GetEvidenceDefinitionIds())
		leanDispositions := make([]stableDisposition, len(leanLink.AppliedDispositions))
		for dispositionIndex, disposition := range leanLink.AppliedDispositions {
			leanDispositions[dispositionIndex] = stableDisposition{
				KindDefinitionID:         disposition.Field.KindDefinitionID,
				FieldDefinitionID:        disposition.Field.FieldDefinitionID,
				Disposition:              disposition.Kind,
				NormalizedValue:          optionalString(disposition.NormalizedValue),
				DigestPolicyDefinitionID: optionalString(disposition.DigestPolicyDefinitionID),
				DigestToken:              optionalString(disposition.DigestToken),
			}
		}
		goDispositions := make([]stableDisposition, len(goLink.GetAppliedDispositions()))
		for dispositionIndex, disposition := range goLink.GetAppliedDispositions() {
			goDispositions[dispositionIndex] = stableDisposition{
				KindDefinitionID:         disposition.GetField().GetKindDefinitionId(),
				FieldDefinitionID:        disposition.GetField().GetFieldDefinitionId(),
				Disposition:              stableDispositionKind(disposition.GetDisposition()),
				NormalizedValue:          stableValue(disposition.GetNormalizedValue()),
				DigestPolicyDefinitionID: disposition.GetDigestPolicyDefinitionId(),
				DigestToken:              disposition.GetDigestToken(),
			}
		}
		require.ElementsMatch(t, leanDispositions, goDispositions)
		goOrdering := make(map[string]*umpirespb.OrderingFact, len(goLink.GetOrderingSupport()))
		for _, fact := range goLink.GetOrderingSupport() {
			goOrdering[fact.GetEvidenceDefinitionId()] = fact
		}
		for _, leanFact := range leanLink.OrderingSupport {
			goFact := goOrdering[leanFact.FactDefinitionID]
			require.NotNil(t, goFact)
			require.Equal(t, string(leanFact.Ordinal), fmt.Sprint(goFact.GetOrdinal()))
			require.Equal(t, leanFact.CausalFactDefinitionIDs,
				append([]string{}, goFact.GetCausalEvidenceDefinitionIds()...))
		}
	}

	require.Len(t, result.GetProperties(), len(leanResult.PropertyVerdicts))
	for propertyIndex, leanProperty := range leanResult.PropertyVerdicts {
		goProperty := result.GetProperties()[propertyIndex]
		require.Equal(t, leanProperty.PropertyDefinitionID,
			goProperty.GetProperty().GetDefinitionId())
		require.Equal(t, leanProperty.Status, semanticStatusName(goProperty.GetStatus()))
		require.Len(t, goProperty.GetClauses(), len(leanProperty.Clauses))
		for clauseIndex, leanClause := range leanProperty.Clauses {
			goClause := goProperty.GetClauses()[clauseIndex]
			require.Equal(t, leanClause.ClauseDefinitionID, goClause.GetClauseDefinitionId())
			require.Equal(t, leanClause.Status, semanticStatusName(goClause.GetStatus()))
			require.Len(t, goClause.GetCoordinates(), len(leanClause.Coordinates))
			for coordinateIndex, leanCoordinate := range leanClause.Coordinates {
				require.Equal(t, stableLeanCoordinate(t, leanCoordinate),
					stableGoCoordinate(goClause.GetCoordinates()[coordinateIndex]))
			}
		}
	}
	expectedGaps := make([]*umpirespb.KnownGap, 0, len(leanResult.KnownGaps)+len(contract.GetKnownGaps()))
	for _, gap := range leanResult.KnownGaps {
		expectedGaps = append(expectedGaps, &umpirespb.KnownGap{
			Kind: parityKnownGapKind(gap.Kind), Code: gap.Code,
			Subject: optionalString(gap.Subject), Detail: optionalString(gap.Detail),
		})
	}
	for _, gap := range contract.GetKnownGaps() {
		expectedGaps = append(expectedGaps, proto.CloneOf(gap))
	}
	require.Len(t, result.GetKnownGaps(), len(expectedGaps))
	for index, gap := range expectedGaps {
		protorequire.ProtoEqual(t, gap, result.GetKnownGaps()[index])
	}
}

type stableDisposition struct {
	KindDefinitionID         string
	FieldDefinitionID        string
	Disposition              string
	NormalizedValue          string
	DigestPolicyDefinitionID string
	DigestToken              string
}

func stableDispositionKind(kind umpirespb.FieldDispositionKind) string {
	return map[umpirespb.FieldDispositionKind]string{
		umpirespb.FIELD_DISPOSITION_KIND_RETAIN: "retained",
		umpirespb.FIELD_DISPOSITION_KIND_REDACT: "redacted-contribution",
		umpirespb.FIELD_DISPOSITION_KIND_HASH:   "digest-token",
		umpirespb.FIELD_DISPOSITION_KIND_REJECT: "rejected-material",
	}[kind]
}

func stableValue(value *umpirespb.Value) string {
	if value == nil {
		return ""
	}
	switch typed := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return typed.Text
	case *umpirespb.Value_Natural:
		return typed.Natural
	case *umpirespb.Value_BoolValue:
		return fmt.Sprint(typed.BoolValue)
	default:
		return ""
	}
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parityKnownGapKind(kind string) umpirespb.KnownGapKind {
	return map[string]umpirespb.KnownGapKind{
		"capability-contract": umpirespb.KNOWN_GAP_KIND_CAPABILITY_CONTRACT,
		"input":               umpirespb.KNOWN_GAP_KIND_INPUT,
		"interpretation":      umpirespb.KNOWN_GAP_KIND_INTERPRETATION,
		"claim":               umpirespb.KNOWN_GAP_KIND_CLAIM,
	}[kind]
}

func semanticRuleDefinitionID(definitionID string) string {
	definitionID = strings.TrimSuffix(definitionID, ".initial")
	return strings.TrimSuffix(definitionID, ".resulting")
}

type stableCoordinate struct {
	Kind     string
	Step     int64
	Position int64
}

func stableGoCoordinate(coordinate *umpirespb.ModelCoordinate) stableCoordinate {
	kinds := map[umpirespb.TraceField]string{
		umpirespb.TRACE_FIELD_INITIAL_STATE:   "initial-state",
		umpirespb.TRACE_FIELD_PRIOR_STATE:     "prior-state",
		umpirespb.TRACE_FIELD_SELECTED_ACTION: "selected-action",
		umpirespb.TRACE_FIELD_MODEL_OUTCOME:   "model-outcome",
		umpirespb.TRACE_FIELD_RESULTING_STATE: "resulting-state",
		umpirespb.TRACE_FIELD_OBSERVATION:     "observation",
	}
	return stableCoordinate{
		Kind: kinds[coordinate.GetField()], Step: coordinate.GetStep(), Position: coordinate.GetPosition(),
	}
}

func stableLeanCoordinate(t testing.TB, coordinate artifactv2.ModelCoordinate) stableCoordinate {
	t.Helper()
	result := stableCoordinate{Kind: coordinate.Kind}
	if coordinate.Step != nil {
		result.Step = int64(naturalUint64(t, *coordinate.Step))
	}
	if coordinate.Position != nil {
		result.Position = int64(naturalUint64(t, *coordinate.Position))
	}
	return result
}

func naturalUint64(t testing.TB, value artifactv2.Natural) uint64 {
	t.Helper()
	result, err := strconv.ParseUint(string(value), 10, 64)
	require.NoError(t, err)
	return result
}

func requireTraceParity(t testing.TB, goTrace *umpirespb.ModelTrace, leanTrace artifactv2.ModelTrace) {
	t.Helper()
	requireModelValueParity(t, goTrace.GetInitialState(), leanTrace.InitialState)
	require.Len(t, goTrace.GetSteps(), len(leanTrace.Steps))
	for index, leanStep := range leanTrace.Steps {
		goStep := goTrace.GetSteps()[index]
		require.Equal(t, string(leanStep.Position), fmt.Sprint(goStep.GetPosition()))
		requireModelValueParity(t, goStep.GetSelectedAction(), leanStep.SelectedAction)
		requireModelValueParity(t, goStep.GetModelOutcome(), leanStep.ModelOutcome)
		requireModelValueParity(t, goStep.GetResultingState(), leanStep.ResultingState)
		require.Len(t, goStep.GetObservations(), len(leanStep.Observations))
		for observationIndex, leanObservation := range leanStep.Observations {
			requireModelValueParity(t, goStep.GetObservations()[observationIndex], leanObservation)
		}
	}
}

func requireModelValueParity(t testing.TB, goValue *umpirespb.ModelValue, leanValue artifactv2.ModelValue) {
	t.Helper()
	require.Equal(t, leanValue.DefinitionID, goValue.GetDefinition().GetDefinitionId())
	switch value := goValue.GetValue().GetValue().(type) {
	case *umpirespb.Value_Text:
		require.Equal(t, leanValue.Value, value.Text)
	case *umpirespb.Value_Natural:
		require.Equal(t, leanValue.Value, value.Natural)
	case *umpirespb.Value_BoolValue:
		require.Equal(t, leanValue.Value, fmt.Sprint(value.BoolValue))
	default:
		require.Fail(t, "portable value is absent")
	}
}

func operationalStatusName(status umpirespb.OperationalStatus) string {
	return map[umpirespb.OperationalStatus]string{
		umpirespb.OPERATIONAL_STATUS_SUCCEEDED:  "succeeded",
		umpirespb.OPERATIONAL_STATUS_INCOMPLETE: "incomplete",
		umpirespb.OPERATIONAL_STATUS_FAILED:     "failed",
	}[status]
}

func observationStatusName(status umpirespb.ObservationStatus) string {
	return map[umpirespb.ObservationStatus]string{
		umpirespb.OBSERVATION_STATUS_ACCEPTED:    "accepted",
		umpirespb.OBSERVATION_STATUS_UNKNOWN:     "unknown",
		umpirespb.OBSERVATION_STATUS_CONFLICT:    "conflict",
		umpirespb.OBSERVATION_STATUS_UNSUPPORTED: "unsupported",
	}[status]
}

func implementationLinkStatusName(status umpirespb.ImplementationLinkStatus) string {
	return map[umpirespb.ImplementationLinkStatus]string{
		umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED: "not-evaluated",
		umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED:       "applied",
		umpirespb.IMPLEMENTATION_LINK_STATUS_INVALID:       "invalid",
		umpirespb.IMPLEMENTATION_LINK_STATUS_UNKNOWN:       "unknown",
		umpirespb.IMPLEMENTATION_LINK_STATUS_CONFLICT:      "conflict",
		umpirespb.IMPLEMENTATION_LINK_STATUS_UNSUPPORTED:   "unsupported",
	}[status]
}

func semanticStatusName(status umpirespb.SemanticStatus) string {
	return map[umpirespb.SemanticStatus]string{
		umpirespb.SEMANTIC_STATUS_SATISFIED:   "satisfied",
		umpirespb.SEMANTIC_STATUS_VIOLATED:    "violated",
		umpirespb.SEMANTIC_STATUS_UNKNOWN:     "unknown",
		umpirespb.SEMANTIC_STATUS_CONFLICT:    "conflict",
		umpirespb.SEMANTIC_STATUS_UNSUPPORTED: "unsupported",
	}[status]
}

func evaluationStatusName(status umpirespb.EvaluationStatus) string {
	return map[umpirespb.EvaluationStatus]string{
		umpirespb.EVALUATION_STATUS_SATISFIED:  "satisfied",
		umpirespb.EVALUATION_STATUS_VIOLATED:   "violated",
		umpirespb.EVALUATION_STATUS_INCOMPLETE: "incomplete",
	}[status]
}

func cleanupStatusName(status umpirespb.CleanupStatus) string {
	return map[umpirespb.CleanupStatus]string{
		umpirespb.CLEANUP_STATUS_COMPLETE:   "complete",
		umpirespb.CLEANUP_STATUS_INCOMPLETE: "incomplete",
		umpirespb.CLEANUP_STATUS_FAILED:     "failed",
	}[status]
}
