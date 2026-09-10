package testpilot_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type correlatedFacadeFixture struct {
	Name       string            `json:"name"`
	Runnable   json.RawMessage   `json:"runnableCase"`
	Events     []json.RawMessage `json:"events"`
	Expected   int               `json:"expected"`
	Incomplete bool              `json:"incomplete"`
}

func correlatedFacadeFixtures(t testing.TB) []correlatedFacadeFixture {
	t.Helper()
	encoded, err := os.ReadFile(facadeCorpusRoot + "/correlated.json")
	require.NoError(t, err)
	var fixtures []correlatedFacadeFixture
	require.NoError(t, json.Unmarshal(encoded, &fixtures))
	return fixtures
}

func correlatedFacadeInputs(t testing.TB, fixture correlatedFacadeFixture) (*testpilotspb.Case, []*testpilotspb.CorrelatedEvidence) {
	t.Helper()
	source, err := testpilot.DecodeCaseProtoJSON(fixture.Runnable)
	require.NoError(t, err)
	events := make([]*testpilotspb.CorrelatedEvidence, len(fixture.Events))
	for i, encoded := range fixture.Events {
		events[i] = new(testpilotspb.CorrelatedEvidence)
		require.NoError(t, protojson.Unmarshal(encoded, events[i]))
	}
	return source, events
}

func correlatedFacadeProfile(t testing.TB, source *testpilotspb.Case) testpilot.ProfileSpec {
	t.Helper()
	descriptors := facadeDescriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	descriptors.File = append(descriptors.File, &descriptorpb.FileDescriptorProto{
		Name: proto.String("test/correlated/source.proto"), Package: proto.String("test.correlated"), Syntax: proto.String("proto3"),
		Dependency: []string{testpilotspb.File_temporal_server_api_testpilot_v1_run_proto.Path()},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Source"), Method: []*descriptorpb.MethodDescriptorProto{{
			Name: proto.String("Read"), InputType: proto.String(".temporal.server.api.testpilot.v1.CorrelatedEvidence"), OutputType: proto.String(".temporal.server.api.testpilot.v1.CorrelatedEvidence"),
		}}}},
	})
	catalog, err := testpilot.NewCatalog(descriptors)
	require.NoError(t, err)
	return testpilot.ProfileSpec{
		Identity: "correlated-facade", Catalog: catalog,
		Roles:         []testpilot.RolePolicy{{ID: "source", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/test.correlated.Source/Read"}}},
		Opcodes:       []testpilot.Opcode{testpilot.InvokeRPC},
		ProgramLimits: proto.CloneOf(source.GetProgram().GetLimits()), ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}

type correlatedFacadeDriver struct {
	identity testpilot.DriverIdentity
	events   []*testpilotspb.CorrelatedEvidence
	failAt   int
	closeErr error
	stop     context.CancelFunc
}

func (d *correlatedFacadeDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (*correlatedFacadeDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *correlatedFacadeDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	return &correlatedFacadeSession{facadeSession: &facadeSession{driver: &facadeDriver{}}, driver: d}, nil
}

type correlatedFacadeSession struct {
	*facadeSession
	driver *correlatedFacadeDriver
}

func (s *correlatedFacadeSession) InvokeRPC(_ context.Context, coordinate testpilot.Coordinate, endpoint string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	if endpoint != "source" || method.FullName() != "test.correlated.Source.Read" {
		return nil, errors.New("unexpected correlated fixture RPC")
	}
	index, err := strconv.Atoi(strings.TrimPrefix(coordinate.InstructionID, "read."))
	if err != nil || index < 0 || index >= len(s.driver.events) {
		return nil, errors.New("unexpected correlated fixture instruction")
	}
	if index == s.driver.failAt {
		if s.driver.stop != nil {
			s.driver.stop()
		}
		return nil, errors.New("evidence source lost")
	}
	return facadeEffect{result: testpilot.EffectResult{
		Outcome:  &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: proto.CloneOf(s.driver.events[index]),
	}}, nil
}
func (s *correlatedFacadeSession) Close(context.Context) error { return s.driver.closeErr }

func TestCorrelatedPublicFacade(t *testing.T) {
	for _, fixture := range correlatedFacadeFixtures(t) {
		if fixture.Incomplete {
			continue
		}
		t.Run(fixture.Name, func(t *testing.T) {
			source, events := correlatedFacadeInputs(t, fixture)
			prepared, err := testpilot.Prepare(source, correlatedFacadeProfile(t, source))
			require.NoError(t, err)
			driver := &correlatedFacadeDriver{identity: prepared.Identity(), events: events, failAt: -1}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			want := map[int]testpilotspb.VerdictStatus{0: testpilotspb.VERDICT_STATUS_INCONCLUSIVE, 2: testpilotspb.VERDICT_STATUS_SATISFIED, 3: testpilotspb.VERDICT_STATUS_VIOLATED}[fixture.Expected]
			require.Equal(t, want, verdict.GetStatus())
			disposition := testpilotspb.RUN_STATUS_COMPLETED
			if want == testpilotspb.VERDICT_STATUS_VIOLATED {
				disposition = testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR
			}
			require.Equal(t, disposition, run.GetStatus())
			require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
			require.True(t, proto.Equal(verdict, run.GetVerdict()))
			for _, sequence := range verdict.GetSupportingEventSequences() {
				require.Equal(t, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, run.GetEvents()[sequence-1].GetKind())
			}
		})
	}
}

func TestCorrelatedFacadeFailureAndPriorProof(t *testing.T) {
	fixtures := correlatedFacadeFixtures(t)
	for _, name := range []string{"wrong-correlation", "lost", "stopped", "cleanup-after-violation"} {
		t.Run(name, func(t *testing.T) {
			fixture := fixtures[3]
			if name == "cleanup-after-violation" {
				fixture = fixtures[1]
			}
			source, events := correlatedFacadeInputs(t, fixture)
			if name == "lost" || name == "stopped" {
				source.Contract.Correlated.Clauses[0].Ending = testpilotspb.TRACE_ENDING_FINAL
			}
			prepared, err := testpilot.Prepare(source, correlatedFacadeProfile(t, source))
			require.NoError(t, err)
			driver := &correlatedFacadeDriver{identity: prepared.Identity(), events: events, failAt: -1}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch name {
			case "wrong-correlation":
				events[1].Operation = "b"
			case "lost":
				driver.failAt = 1
			case "stopped":
				driver.failAt = 1
				driver.stop = cancel
			case "cleanup-after-violation":
				driver.closeErr = errors.New("cleanup unavailable")
			default:
				t.Fatalf("unknown failure fixture %q", name)
			}
			run, verdict, err := prepared.Run(ctx, driver)
			require.NoError(t, err)
			want, disposition := testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilotspb.RUN_STATUS_COMPLETED
			if name == "lost" || name == "stopped" {
				disposition = testpilotspb.RUN_STATUS_INCOMPLETE
			}
			if name == "cleanup-after-violation" {
				want, disposition = testpilotspb.VERDICT_STATUS_VIOLATED, testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR
				require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
				require.Equal(t, []int64{5}, verdict.GetSupportingEventSequences())
			}
			require.Equal(t, want, verdict.GetStatus())
			require.Equal(t, disposition, run.GetStatus())
			require.True(t, proto.Equal(verdict, run.GetVerdict()))
		})
	}
}

func TestCorrelatedFacadeRepeatedConcurrentIsolation(t *testing.T) {
	source, events := correlatedFacadeInputs(t, correlatedFacadeFixtures(t)[6])
	prepared, err := testpilot.Prepare(source, correlatedFacadeProfile(t, source))
	require.NoError(t, err)
	other := make([]*testpilotspb.CorrelatedEvidence, len(events))
	for i, event := range events {
		other[i] = proto.CloneOf(event)
	}
	other[len(other)-1].Operation = "b"
	drivers := []*correlatedFacadeDriver{
		{identity: prepared.Identity(), events: events, failAt: -1},
		{identity: prepared.Identity(), events: other, failAt: -1},
	}
	type expectedResult struct {
		facadeRunResult
		want testpilotspb.VerdictStatus
	}
	ids := map[string]bool{}
	for range 2 {
		results := make(chan expectedResult, 10)
		for index := range 10 {
			go func() {
				driver := drivers[index%len(drivers)]
				want := testpilotspb.VERDICT_STATUS_SATISFIED
				if index%len(drivers) == 1 {
					want = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
				}
				run, verdict, err := prepared.Run(t.Context(), driver)
				results <- expectedResult{facadeRunResult: facadeRunResult{run: run, verdict: verdict, err: err}, want: want}
			}()
		}
		for range 10 {
			result := <-results
			require.NoError(t, result.err)
			require.Equal(t, result.want, result.verdict.GetStatus())
			require.False(t, ids[result.run.GetRunId()])
			ids[result.run.GetRunId()] = true
			require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, result.run.GetStatus())
			wantRule := testpilotspb.RULE_VERDICT_STATUS_SATISFIED
			if result.want == testpilotspb.VERDICT_STATUS_INCONCLUSIVE {
				wantRule = testpilotspb.RULE_VERDICT_STATUS_INCONCLUSIVE
			}
			result.verdict.Rules[0].Status = testpilotspb.RULE_VERDICT_STATUS_VIOLATED
			require.Equal(t, wantRule, result.run.GetVerdict().GetRules()[0].GetStatus())
		}
	}
	require.Len(t, ids, 20)
}

func TestCorrelatedFacadeTenfoldLoad(t *testing.T) {
	for _, kind := range []string{"evidence", "obligations", "buffer", "work"} {
		for _, count := range []int{2, 20} {
			t.Run(fmt.Sprintf("%s/%d", kind, count), func(t *testing.T) {
				source, events := correlatedFacadeInputs(t, correlatedFacadeFixtures(t)[0])
				source.Contract.Correlated.Clauses[0].Bound = 100
				if kind == "obligations" {
					events[0].Kind = "test.request"
				}
				if kind == "buffer" {
					events[0].Identity.Ordinal = 1
				}
				node := source.Program.Entrypoints[0].Instructions[0]
				source.Program.Entrypoints[0].Instructions = nil
				first := events[0]
				events = nil
				for i := range count {
					event := proto.CloneOf(first)
					event.Identity.Ordinal += int64(i)
					events = append(events, event)
					instruction := proto.CloneOf(node)
					instruction.InstructionId = fmt.Sprintf("read.%d", i)
					if i > 0 {
						instruction.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: fmt.Sprintf("read.%d", i-1)}}
					}
					source.Program.Entrypoints[0].Instructions = append(source.Program.Entrypoints[0].Instructions, instruction)
				}
				if kind == "obligations" || kind == "work" {
					source.Contract.Correlated.Limits.MaxEvents = 32
				}
				if kind == "work" {
					source.Contract.Correlated.Limits.MaxObligationWork = 200
				}
				prepared, err := testpilot.Prepare(source, correlatedFacadeProfile(t, source))
				require.NoError(t, err)
				driver := &correlatedFacadeDriver{identity: prepared.Identity(), events: events, failAt: -1}
				run, verdict, err := prepared.Run(t.Context(), driver)
				require.NoError(t, err)
				want, disposition := testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilotspb.RUN_STATUS_COMPLETED
				if count == 2 && (kind == "evidence" || kind == "work") {
					want = testpilotspb.VERDICT_STATUS_SATISFIED
				}
				if count == 20 {
					disposition = testpilotspb.RUN_STATUS_INCOMPLETE
					require.NotEmpty(t, run.GetDiagnostics())
				}
				require.Equal(t, want, verdict.GetStatus())
				require.Equal(t, disposition, run.GetStatus())
				require.LessOrEqual(t, len(run.GetEvents()), 128)
			})
		}
	}
}

func TestCorrelatedFacadeCaptureAdmission(t *testing.T) {
	for _, bytes := range []bool{false, true} {
		t.Run(fmt.Sprint(bytes), func(t *testing.T) {
			source, _ := correlatedFacadeInputs(t, correlatedFacadeFixtures(t)[0])
			if bytes {
				source.Contract.Limits.MaxCaptureBytes = 1
			} else {
				source.Contract.Limits.MaxCaptures = 1
			}
			prepared, err := testpilot.Prepare(source, correlatedFacadeProfile(t, source))
			require.Error(t, err)
			require.Nil(t, prepared)
		})
	}
}
