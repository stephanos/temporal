package testpilot_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const facadeCorpusRoot = "testdata/case-runtime-conformance"

type facadeExpectedResult struct {
	Class       string                     `json:"class"`
	Preparation string                     `json:"preparation"`
	RunCount    int                        `json:"runCount"`
	Projection  *facadeStableRunProjection `json:"projection,omitempty"`
}

type facadeStableRunProjection struct {
	CaseID                   string                             `json:"caseId"`
	ProgramID                string                             `json:"programId"`
	Disposition              string                             `json:"disposition"`
	CleanupStatus            string                             `json:"cleanupStatus"`
	CleanupDiagnostics       []facadeStableDiagnosticProjection `json:"cleanupDiagnostics"`
	Events                   []facadeStableEventProjection      `json:"events"`
	Diagnostics              []facadeStableDiagnosticProjection `json:"diagnostics"`
	VerdictStatus            string                             `json:"verdictKind"`
	Rules                    []facadeStableRuleProjection       `json:"rules"`
	SupportingEventSequences []int64                            `json:"supportingEventSequences"`
}

type facadeStableEventProjection struct {
	Kind                string `json:"kind"`
	EntrypointID        string `json:"entrypointId,omitempty"`
	InstructionID       string `json:"instructionId,omitempty"`
	Attempt             int64  `json:"attempt,omitempty"`
	OutcomeStatus       string `json:"outcomeStatus,omitempty"`
	ExecutionIncomplete bool   `json:"executionIncomplete"`
}

type facadeStableRuleProjection struct {
	RuleID                   string  `json:"ruleId"`
	Kind                     string  `json:"kind"`
	TerminalStateID          string  `json:"terminalStateId"`
	SupportingEventSequences []int64 `json:"supportingEventSequences"`
}

type facadeStableDiagnosticProjection struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

func TestCaseRuntimePublicFacadeConformance(t *testing.T) {
	classes := []string{
		"satisfied",
		"violated",
		"inconclusive",
		"static-preparation-rejection",
		"cleanup-failure-after-proved-violation",
		"cross-run-isolation",
	}
	require.Len(t, classes, 6)
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			source := loadFacadeCase(t, class)
			expected := loadFacadeExpected(t, class)
			require.Equal(t, expected.Class, class)
			profile := facadeProfile(t, source)
			driver := &facadeDriver{failCleanup: class == "cleanup-failure-after-proved-violation"}
			prepared, err := testpilot.Prepare(source, profile)
			if expected.Preparation == "rejected" {
				require.Error(t, err)
				require.Nil(t, prepared)
				require.Empty(t, driver.openedRunIDs())
				return
			}
			require.Equal(t, "accepted", expected.Preparation)
			require.NoError(t, err)
			driver.identity = prepared.Identity()
			results := runFacadeCase(t, prepared, driver, expected.RunCount)
			runIDs := make(map[string]struct{}, len(results))
			for _, result := range results {
				require.NoError(t, result.err)
				require.NotNil(t, result.run)
				require.NotNil(t, result.verdict)
				require.Equal(t, *expected.Projection, projectFacadeRun(result.run))
				require.True(t, proto.Equal(result.verdict, result.run.GetVerdict()))
				validateFacadeDynamicFields(t, result.run)
				require.NotContains(t, runIDs, result.run.GetRunId())
				runIDs[result.run.GetRunId()] = struct{}{}
			}
			require.ElementsMatch(t, driver.openedRunIDs(), mapKeys(runIDs))
			require.Equal(t, expected.RunCount, driver.closedSessions())
		})
	}
}

type facadeRunResult struct {
	run     *testpilotpb.Run
	verdict *testpilotpb.Verdict
	err     error
}

func runFacadeCase(t *testing.T, prepared *testpilot.PreparedCase, driver testpilot.Driver, count int) []facadeRunResult {
	t.Helper()
	results := make(chan facadeRunResult, count)
	var wait sync.WaitGroup
	for range count {
		wait.Go(func() {
			run, verdict, err := prepared.Run(t.Context(), driver)
			results <- facadeRunResult{run: run, verdict: verdict, err: err}
		})
	}
	wait.Wait()
	close(results)
	collected := make([]facadeRunResult, 0, count)
	for result := range results {
		collected = append(collected, result)
	}
	return collected
}

func loadFacadeCase(t testing.TB, class string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(facadeCorpusRoot, class, "case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func loadFacadeExpected(t testing.TB, class string) facadeExpectedResult {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(facadeCorpusRoot, class, "expected.json"))
	require.NoError(t, err)
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.DisallowUnknownFields()
	var expected facadeExpectedResult
	require.NoError(t, decoder.Decode(&expected))
	return expected
}

func facadeProfile(t testing.TB, source *testpilotpb.Case) testpilot.ProfileSpec {
	t.Helper()
	catalog, err := testpilot.NewCatalog(facadeDescriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	return testpilot.ProfileSpec{
		Identity: "facade-conformance",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{{
			ID: "temporal.workflow-service", Kind: testpilotpb.ROLE_KIND_ENDPOINT,
			Methods: []string{"/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"},
		}},
		Capabilities:   []testpilot.Capability{testpilot.InvokeRPC},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}

func facadeDescriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := make(map[string]struct{})
	result := &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := seen[file.Path()]; exists {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			add(imports.Get(index))
		}
		result.File = append(result.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return result
}

type facadeDriver struct {
	identity    testpilot.DriverIdentity
	failCleanup bool
	mu          sync.Mutex
	runIDs      []string
	sessions    []*facadeSession
}

func (h *facadeDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return h.identity, nil
}
func (h *facadeDriver) Open(_ context.Context, runID string, _ testpilot.PreparedProgram) (testpilot.Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session := &facadeSession{driver: h}
	h.runIDs = append(h.runIDs, runID)
	h.sessions = append(h.sessions, session)
	return session, nil
}
func (h *facadeDriver) openedRunIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.runIDs...)
}
func (h *facadeDriver) closedSessions() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	closed := 0
	for _, session := range h.sessions {
		if session.closed {
			closed++
		}
	}
	return closed
}

type facadeSession struct {
	driver *facadeDriver
	closed bool
}

func (*facadeSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	return nil, errors.New("facade conformance Cases do not reserve activations")
}
func (s *facadeSession) InvokeRPC(_ context.Context, coordinate testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	if s.driver.failCleanup && coordinate.EntrypointID == "cleanup" {
		return nil, errors.New("fixture cleanup failure")
	}
	response := dynamicpb.NewMessage(method.Output())
	if field := response.Descriptor().Fields().ByName("server_version"); field != nil {
		response.Set(field, protoreflect.ValueOfString("facade-conformance"))
	}
	return facadeEffect{result: testpilot.EffectResult{
		Outcome:  &testpilotpb.InstructionOutcome{Status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: response,
	}}, nil
}
func (*facadeSession) CompleteNexusOperation(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, *testpilotpb.Value) (testpilot.EffectHandle, error) {
	return nil, errors.New("facade conformance Cases do not complete Nexus operations")
}
func (*facadeSession) Bridge(context.Context) (testpilot.CapabilityBridge, error) {
	return nil, errors.New("facade conformance Cases do not use capability bridges")
}
func (*facadeSession) Quarantine(context.Context, testpilot.EffectHandle) error {
	return errors.New("facade conformance effects complete synchronously")
}
func (s *facadeSession) Close(context.Context) error {
	s.driver.mu.Lock()
	defer s.driver.mu.Unlock()
	s.closed = true
	return nil
}
func (*facadeSession) Diagnose(context.Context, string, *testpilotpb.RunDiagnostic) error { return nil }

type facadeEffect struct{ result testpilot.EffectResult }

func (e facadeEffect) Wait(context.Context) (testpilot.EffectResult, error) { return e.result, nil }
func (facadeEffect) Cancel(context.Context) error                           { return nil }
func (facadeEffect) Drain(context.Context) error                            { return nil }

func projectFacadeRun(run *testpilotpb.Run) facadeStableRunProjection {
	events := make([]facadeStableEventProjection, 0, len(run.GetEvents()))
	for _, event := range run.GetEvents() {
		events = append(events, facadeStableEventProjection{
			Kind: eventKindName(event.GetKind()), EntrypointID: event.GetCoordinates().GetEntrypointId(),
			InstructionID: event.GetCoordinates().GetInstructionId(), Attempt: event.GetCoordinates().GetAttempt(),
			OutcomeStatus: outcomeStatusName(event.GetOutcome().GetStatus()), ExecutionIncomplete: event.GetExecutionIncomplete(),
		})
	}
	rules := make([]facadeStableRuleProjection, 0, len(run.GetVerdict().GetRules()))
	for _, rule := range run.GetVerdict().GetRules() {
		rules = append(rules, facadeStableRuleProjection{
			RuleID: rule.GetRuleId(), Kind: ruleVerdictName(rule.GetStatus()), TerminalStateID: rule.GetTerminalStateId(),
			SupportingEventSequences: append([]int64{}, rule.GetSupportingEventSequences()...),
		})
	}
	diagnostics := make([]facadeStableDiagnosticProjection, 0, len(run.GetDiagnostics()))
	diagnosticsByID := make(map[string]facadeStableDiagnosticProjection, len(run.GetDiagnostics()))
	for _, diagnostic := range run.GetDiagnostics() {
		projection := facadeStableDiagnosticProjection{
			Kind: diagnosticKindName(diagnostic.GetKind()), Code: diagnostic.GetCode(),
		}
		diagnostics = append(diagnostics, projection)
		diagnosticsByID[diagnostic.GetDiagnosticId()] = projection
	}
	cleanupDiagnostics := make([]facadeStableDiagnosticProjection, 0, len(run.GetCleanup().GetDiagnosticIds()))
	for _, diagnosticID := range run.GetCleanup().GetDiagnosticIds() {
		cleanupDiagnostics = append(cleanupDiagnostics, diagnosticsByID[diagnosticID])
	}
	return facadeStableRunProjection{
		CaseID: run.GetCaseId(), ProgramID: run.GetProgramId(), Disposition: dispositionName(run.GetStatus()),
		CleanupStatus: cleanupStatusName(run.GetCleanup().GetStatus()), CleanupDiagnostics: cleanupDiagnostics,
		Events: events, Diagnostics: diagnostics, VerdictStatus: verdictName(run.GetVerdict().GetStatus()), Rules: rules,
		SupportingEventSequences: append([]int64{}, run.GetVerdict().GetSupportingEventSequences()...),
	}
}

func validateFacadeDynamicFields(t testing.TB, run *testpilotpb.Run) {
	t.Helper()
	require.True(t, strings.HasPrefix(run.GetRunId(), "testpilot.run."))
	_, err := uuid.Parse(strings.TrimPrefix(run.GetRunId(), "testpilot.run."))
	require.NoError(t, err)
	sources := make(map[string]int64, len(run.GetEvents()))
	var elapsed int64
	for index, event := range run.GetEvents() {
		require.Equal(t, int64(index+1), event.GetSequence())
		require.GreaterOrEqual(t, event.GetElapsedMilliseconds(), elapsed)
		elapsed = event.GetElapsedMilliseconds()
		require.NotEmpty(t, event.GetSourceId())
		require.NotContains(t, sources, event.GetSourceId())
		for _, cause := range event.GetCausalSourceIds() {
			sequence, exists := sources[cause]
			require.True(t, exists)
			require.Less(t, sequence, event.GetSequence())
		}
		sources[event.GetSourceId()] = event.GetSequence()
		if event.GetCoordinates().GetEntrypointId() != "" {
			require.NotEmpty(t, event.GetCoordinates().GetActivationId())
		}
	}
	validateSupportingSequences(t, len(run.GetEvents()), run.GetVerdict().GetSupportingEventSequences())
	for _, rule := range run.GetVerdict().GetRules() {
		validateSupportingSequences(t, len(run.GetEvents()), rule.GetSupportingEventSequences())
	}
	diagnosticIDs := make(map[string]struct{}, len(run.GetDiagnostics()))
	for _, diagnostic := range run.GetDiagnostics() {
		require.NotEmpty(t, diagnostic.GetDiagnosticId())
		require.NotContains(t, diagnosticIDs, diagnostic.GetDiagnosticId())
		diagnosticIDs[diagnostic.GetDiagnosticId()] = struct{}{}
		require.NotEmpty(t, diagnostic.GetDetail())
		if sequence := diagnostic.GetSupportingEventSequence(); sequence != 0 {
			validateSupportingSequences(t, len(run.GetEvents()), []int64{sequence})
		}
	}
	cleanupDiagnosticIDs := make(map[string]struct{}, len(run.GetCleanup().GetDiagnosticIds()))
	for _, diagnosticID := range run.GetCleanup().GetDiagnosticIds() {
		require.NotEmpty(t, diagnosticID)
		require.NotContains(t, cleanupDiagnosticIDs, diagnosticID)
		cleanupDiagnosticIDs[diagnosticID] = struct{}{}
		_, exists := diagnosticIDs[diagnosticID]
		require.True(t, exists)
	}
}

func validateSupportingSequences(t testing.TB, eventCount int, sequences []int64) {
	t.Helper()
	for _, sequence := range sequences {
		require.GreaterOrEqual(t, sequence, int64(1))
		require.LessOrEqual(t, sequence, int64(eventCount))
	}
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func eventKindName(kind testpilotpb.RunEventKind) string {
	switch kind {
	case testpilotpb.RUN_EVENT_KIND_RUN_OPENED:
		return "RUN_OPENED"
	case testpilotpb.RUN_EVENT_KIND_ACTIVATION_OPENED:
		return "ACTIVATION_OPENED"
	case testpilotpb.RUN_EVENT_KIND_INSTRUCTION_STARTED:
		return "INSTRUCTION_STARTED"
	case testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED:
		return "INSTRUCTION_COMPLETED"
	case testpilotpb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT:
		return "INSTRUCTION_TIMED_OUT"
	case testpilotpb.RUN_EVENT_KIND_ACTIVATION_CLOSED:
		return "ACTIVATION_CLOSED"
	case testpilotpb.RUN_EVENT_KIND_CLEANUP_STARTED:
		return "CLEANUP_STARTED"
	case testpilotpb.RUN_EVENT_KIND_CLEANUP_COMPLETED:
		return "CLEANUP_COMPLETED"
	case testpilotpb.RUN_EVENT_KIND_RUN_CLOSED:
		return "RUN_CLOSED"
	case testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC:
		return "DIAGNOSTIC"
	default:
		return "UNSPECIFIED"
	}
}

func dispositionName(value testpilotpb.RunStatus) string {
	switch value {
	case testpilotpb.RUN_STATUS_COMPLETED:
		return "COMPLETED"
	case testpilotpb.RUN_STATUS_STOPPED_BY_MONITOR:
		return "STOPPED_BY_MONITOR"
	case testpilotpb.RUN_STATUS_INCOMPLETE:
		return "INCOMPLETE"
	default:
		return "UNSPECIFIED"
	}
}

func cleanupStatusName(value testpilotpb.CleanupStatus) string {
	switch value {
	case testpilotpb.CLEANUP_STATUS_SUCCEEDED:
		return "SUCCEEDED"
	case testpilotpb.CLEANUP_STATUS_FAILED:
		return "FAILED"
	case testpilotpb.CLEANUP_STATUS_TIMED_OUT:
		return "TIMED_OUT"
	default:
		return "UNSPECIFIED"
	}
}

func diagnosticKindName(value testpilotpb.RunDiagnosticKind) string {
	switch value {
	case testpilotpb.RUN_DIAGNOSTIC_KIND_EXECUTION:
		return "EXECUTION"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_MONITOR:
		return "MONITOR"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_RECORDER:
		return "RECORDER"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_INVARIANT:
		return "INVARIANT"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_LIMIT:
		return "LIMIT"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_DRIVER_CONTRACT:
		return "HOST_CONTRACT"
	case testpilotpb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT:
		return "POST_CLOSE_EVENT"
	default:
		return "UNSPECIFIED"
	}
}

func verdictName(value testpilotpb.VerdictStatus) string {
	switch value {
	case testpilotpb.VERDICT_STATUS_SATISFIED:
		return "SATISFIED"
	case testpilotpb.VERDICT_STATUS_VIOLATED:
		return "VIOLATED"
	case testpilotpb.VERDICT_STATUS_INCONCLUSIVE:
		return "INCONCLUSIVE"
	default:
		return "UNSPECIFIED"
	}
}

func ruleVerdictName(value testpilotpb.RuleVerdictStatus) string {
	switch value {
	case testpilotpb.RULE_VERDICT_STATUS_PENDING:
		return "PENDING"
	case testpilotpb.RULE_VERDICT_STATUS_SATISFIED:
		return "SATISFIED"
	case testpilotpb.RULE_VERDICT_STATUS_VIOLATED:
		return "VIOLATED"
	case testpilotpb.RULE_VERDICT_STATUS_INCONCLUSIVE:
		return "INCONCLUSIVE"
	default:
		return "UNSPECIFIED"
	}
}

func outcomeStatusName(value testpilotpb.InstructionOutcomeStatus) string {
	switch value {
	case testpilotpb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED:
		return "SUCCEEDED"
	case testpilotpb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS:
		return "PROTOCOL_NON_SUCCESS"
	case testpilotpb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE:
		return "SDK_FAILURE"
	case testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT:
		return "TIMED_OUT"
	case testpilotpb.INSTRUCTION_OUTCOME_STATUS_CANCELED:
		return "CANCELED"
	default:
		return ""
	}
}
