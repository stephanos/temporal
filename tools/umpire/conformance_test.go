package umpire_test

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
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/caseartifact"
	umpiretemporal "go.temporal.io/server/tools/umpire/temporal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	VerdictKind              string                             `json:"verdictKind"`
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
			host := &facadeHost{failCleanup: class == "cleanup-failure-after-proved-violation"}
			prepared, err := umpire.PrepareCase(source, profile)
			if expected.Preparation == "rejected" {
				require.Error(t, err)
				require.Nil(t, prepared)
				require.Empty(t, host.openedRunIDs())
				return
			}
			require.Equal(t, "accepted", expected.Preparation)
			require.NoError(t, err)
			host.identity = prepared.Identity()
			results := runFacadeCase(t, prepared, host, expected.RunCount)
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
			require.ElementsMatch(t, host.openedRunIDs(), mapKeys(runIDs))
			require.Equal(t, expected.RunCount, host.closedSessions())
		})
	}
}

type facadeRunResult struct {
	run     *umpirespb.Run
	verdict *umpirespb.Verdict
	err     error
}

func runFacadeCase(t *testing.T, prepared *umpire.PreparedCase, host umpire.Host, count int) []facadeRunResult {
	t.Helper()
	results := make(chan facadeRunResult, count)
	var wait sync.WaitGroup
	for range count {
		wait.Go(func() {
			run, verdict, err := prepared.Run(t.Context(), host)
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

func loadFacadeCase(t testing.TB, class string) *umpirespb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(facadeCorpusRoot, class, "case.json"))
	require.NoError(t, err)
	decoded, err := caseartifact.DecodeProtoJSON(encoded)
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

func facadeProfile(t testing.TB, source *umpirespb.Case) umpire.ProfileSpec {
	t.Helper()
	catalog, err := umpiretemporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	return umpire.ProfileSpec{
		Identity: "facade-conformance",
		Catalog:  catalog,
		Roles: []umpire.RolePolicy{{
			ID: "temporal.workflow-service", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT,
			Methods: []string{"/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"},
		}},
		Capabilities:   []umpire.Capability{umpire.InvokeRPC},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}

type facadeHost struct {
	identity    umpire.HostIdentity
	failCleanup bool
	mu          sync.Mutex
	runIDs      []string
	sessions    []*facadeSession
}

func (h *facadeHost) Identity(context.Context) (umpire.HostIdentity, error) { return h.identity, nil }
func (h *facadeHost) Open(_ context.Context, runID string, _ umpire.PreparedProgram) (umpire.Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session := &facadeSession{host: h}
	h.runIDs = append(h.runIDs, runID)
	h.sessions = append(h.sessions, session)
	return session, nil
}
func (h *facadeHost) openedRunIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.runIDs...)
}
func (h *facadeHost) closedSessions() int {
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
	host   *facadeHost
	closed bool
}

func (*facadeSession) Reserve(context.Context, umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	return nil, errors.New("facade conformance Cases do not reserve activations")
}
func (s *facadeSession) InvokeRPC(_ context.Context, coordinate umpire.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (umpire.EffectHandle, error) {
	if s.host.failCleanup && coordinate.EntrypointID == "cleanup" {
		return nil, errors.New("fixture cleanup failure")
	}
	response := dynamicpb.NewMessage(method.Output())
	if field := response.Descriptor().Fields().ByName("server_version"); field != nil {
		response.Set(field, protoreflect.ValueOfString("facade-conformance"))
	}
	return facadeEffect{result: umpire.EffectResult{
		Outcome:  &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: response,
	}}, nil
}
func (*facadeSession) CompleteNexusOperation(context.Context, umpire.Coordinate, umpire.OpaqueCapability, *umpirespb.Value) (umpire.EffectHandle, error) {
	return nil, errors.New("facade conformance Cases do not complete Nexus operations")
}
func (*facadeSession) Bridge(context.Context) (umpire.CapabilityBridge, error) {
	return nil, errors.New("facade conformance Cases do not use capability bridges")
}
func (*facadeSession) Quarantine(context.Context, umpire.EffectHandle) error {
	return errors.New("facade conformance effects complete synchronously")
}
func (s *facadeSession) Close(context.Context) error {
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	s.closed = true
	return nil
}
func (*facadeSession) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error { return nil }

type facadeEffect struct{ result umpire.EffectResult }

func (e facadeEffect) Wait(context.Context) (umpire.EffectResult, error) { return e.result, nil }
func (facadeEffect) Cancel(context.Context) error                        { return nil }
func (facadeEffect) Drain(context.Context) error                         { return nil }

func projectFacadeRun(run *umpirespb.Run) facadeStableRunProjection {
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
			RuleID: rule.GetRuleId(), Kind: ruleVerdictName(rule.GetKind()), TerminalStateID: rule.GetTerminalStateId(),
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
		CaseID: run.GetCaseId(), ProgramID: run.GetProgramId(), Disposition: dispositionName(run.GetDisposition()),
		CleanupStatus: cleanupStatusName(run.GetCleanup().GetStatus()), CleanupDiagnostics: cleanupDiagnostics,
		Events: events, Diagnostics: diagnostics, VerdictKind: verdictName(run.GetVerdict().GetKind()), Rules: rules,
		SupportingEventSequences: append([]int64{}, run.GetVerdict().GetSupportingEventSequences()...),
	}
}

func validateFacadeDynamicFields(t testing.TB, run *umpirespb.Run) {
	t.Helper()
	require.True(t, strings.HasPrefix(run.GetRunId(), "umpire.run."))
	_, err := uuid.Parse(strings.TrimPrefix(run.GetRunId(), "umpire.run."))
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
		if sequence := diagnostic.GetSupportingEventSequence().GetValue(); sequence != 0 {
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

func eventKindName(kind umpirespb.RunEventKind) string {
	switch kind {
	case umpirespb.RUN_EVENT_KIND_RUN_OPENED:
		return "RUN_OPENED"
	case umpirespb.RUN_EVENT_KIND_ACTIVATION_OPENED:
		return "ACTIVATION_OPENED"
	case umpirespb.RUN_EVENT_KIND_INSTRUCTION_STARTED:
		return "INSTRUCTION_STARTED"
	case umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED:
		return "INSTRUCTION_COMPLETED"
	case umpirespb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT:
		return "INSTRUCTION_TIMED_OUT"
	case umpirespb.RUN_EVENT_KIND_ACTIVATION_CLOSED:
		return "ACTIVATION_CLOSED"
	case umpirespb.RUN_EVENT_KIND_CLEANUP_STARTED:
		return "CLEANUP_STARTED"
	case umpirespb.RUN_EVENT_KIND_CLEANUP_COMPLETED:
		return "CLEANUP_COMPLETED"
	case umpirespb.RUN_EVENT_KIND_RUN_CLOSED:
		return "RUN_CLOSED"
	case umpirespb.RUN_EVENT_KIND_DIAGNOSTIC:
		return "DIAGNOSTIC"
	default:
		return "UNSPECIFIED"
	}
}

func dispositionName(value umpirespb.RunDisposition) string {
	switch value {
	case umpirespb.RUN_DISPOSITION_COMPLETED:
		return "COMPLETED"
	case umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR:
		return "STOPPED_BY_MONITOR"
	case umpirespb.RUN_DISPOSITION_INCOMPLETE:
		return "INCOMPLETE"
	default:
		return "UNSPECIFIED"
	}
}

func cleanupStatusName(value umpirespb.RunCleanupStatus) string {
	switch value {
	case umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED:
		return "SUCCEEDED"
	case umpirespb.RUN_CLEANUP_STATUS_FAILED:
		return "FAILED"
	case umpirespb.RUN_CLEANUP_STATUS_TIMED_OUT:
		return "TIMED_OUT"
	default:
		return "UNSPECIFIED"
	}
}

func diagnosticKindName(value umpirespb.RunDiagnosticKind) string {
	switch value {
	case umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION:
		return "EXECUTION"
	case umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR:
		return "MONITOR"
	case umpirespb.RUN_DIAGNOSTIC_KIND_RECORDER:
		return "RECORDER"
	case umpirespb.RUN_DIAGNOSTIC_KIND_INVARIANT:
		return "INVARIANT"
	case umpirespb.RUN_DIAGNOSTIC_KIND_LIMIT:
		return "LIMIT"
	case umpirespb.RUN_DIAGNOSTIC_KIND_HOST_CONTRACT:
		return "HOST_CONTRACT"
	case umpirespb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT:
		return "POST_CLOSE_EVENT"
	default:
		return "UNSPECIFIED"
	}
}

func verdictName(value umpirespb.VerdictKind) string {
	switch value {
	case umpirespb.VERDICT_KIND_SATISFIED:
		return "SATISFIED"
	case umpirespb.VERDICT_KIND_VIOLATED:
		return "VIOLATED"
	case umpirespb.VERDICT_KIND_INCONCLUSIVE:
		return "INCONCLUSIVE"
	default:
		return "UNSPECIFIED"
	}
}

func ruleVerdictName(value umpirespb.RuleVerdictKind) string {
	switch value {
	case umpirespb.RULE_VERDICT_KIND_PENDING:
		return "PENDING"
	case umpirespb.RULE_VERDICT_KIND_SATISFIED:
		return "SATISFIED"
	case umpirespb.RULE_VERDICT_KIND_VIOLATED:
		return "VIOLATED"
	case umpirespb.RULE_VERDICT_KIND_INCONCLUSIVE:
		return "INCONCLUSIVE"
	default:
		return "UNSPECIFIED"
	}
}

func outcomeStatusName(value umpirespb.InstructionOutcomeStatus) string {
	switch value {
	case umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED:
		return "SUCCEEDED"
	case umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS:
		return "PROTOCOL_NON_SUCCESS"
	case umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE:
		return "SDK_FAILURE"
	case umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT:
		return "TIMED_OUT"
	case umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED:
		return "CANCELED"
	default:
		return ""
	}
}
