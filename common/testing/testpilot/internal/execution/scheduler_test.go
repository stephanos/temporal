package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type schedulerMonitor struct {
	Monitor
	observe func(*testpilotspb.RunEvent) Decision
}

func (m schedulerMonitor) Observe(_ context.Context, e *testpilotspb.RunEvent) (Decision, error) {
	if m.observe != nil {
		return m.observe(e), nil
	}
	return Continue, nil
}
func TestSchedulerProjectsActualValues(t *testing.T) {
	p, _, _ := dataFixture(t)
	h := &testsupport.Session{OnInvokeRPC: func(_ context.Context, c contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message) (contract.EffectHandle, error) {
		require.Equal(t, int64(1), c.Attempt)
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "kept"), nil }}, nil
	}}
	s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Equal(t, "kept", s.values.slots["text"].GetStringValue())
	require.Len(t, s.outstanding(), 1)
	require.Equal(t, testpilotspb.RUN_EVENT_KIND_ACTIVATION_CLOSED, s.recorder.run.Events[len(s.recorder.run.Events)-1].Kind)
	require.Error(t, s.execute(context.Background()))
}

func TestSchedulerDependencyConcurrencyGuardsAndIsolation(t *testing.T) {
	c, catalog, policy := fixture(t)
	first := c.Program.Entrypoints[0].Instructions[0]
	first.Limits.MaxAttempts = proto.Int64(3)
	second := rpcNode("second")
	second.After = runsAfter("controller")
	skipped := rpcNode("skipped")
	skipped.After = runsAfter("controller")
	skipped.Guard = cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: false}})
	consumer := rpcNode("consumer")
	consumer.After = runsAfter("controller", "call", "skipped")
	consumer.Guard = cel.Not(present(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "skipped"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})))
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, second, skipped, consumer)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	for _, run := range []string{"one", "two"} {
		t.Run(run, func(t *testing.T) {
			calls := make(chan contract.Coordinate, 3)
			release := make(chan struct{})
			host := &testsupport.Session{OnInvokeRPC: func(_ context.Context, c contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message) (contract.EffectHandle, error) {
				calls <- c
				return &testsupport.Effect{OnWait: func(ctx context.Context) (contract.EffectResult, error) {
					select {
					case <-release:
						return effectResponse(p, "ok"), nil
					case <-ctx.Done():
						return contract.EffectResult{}, ctx.Err()
					}
				}}, nil
			}}
			s, err := newScheduler(p, run, "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() { done <- s.execute(context.Background()) }()
			first, second := <-calls, <-calls
			require.Equal(t, []string{"call", "second"}, []string{first.InstructionID, second.InstructionID})
			require.Equal(t, run, first.RunID)
			close(release)
			require.NoError(t, <-done)
			require.Equal(t, "consumer", (<-calls).InstructionID)
			require.Empty(t, calls)
			require.Len(t, s.outstanding(), 3)
			require.Len(t, s.values.activations, 1)
		})
	}
}

// A reserved entrypoint that carries no instruction may go undelivered: its reservation, released
// canceled when the parent finished, is recorded and the Run goes on. The same release of an
// entrypoint that does perform something fails the Run.
func TestSchedulerAdmitsAnUnusedReservationOfAnEmptyEntrypoint(t *testing.T) {
	for _, mode := range []string{"empty", "performing"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			addWorker(c, &policy)
			second := proto.CloneOf(c.Program.Entrypoints[1])
			second.EntrypointId = "workflow_second"
			if mode == "performing" {
				second.Instructions = []*testpilotspb.InstructionNode{{InstructionId: "finish", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: cel.Literal(textValue("done"))}}}, Limits: rpcNode("finish").Limits}}
			}
			c.Program.Entrypoints = append(c.Program.Entrypoints, second)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			host := &testsupport.Session{}
			var origin contract.Coordinate
			host.OnReserve = func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				origin = r.Origin
				h := &testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, Ordinal: 0, ID: "reservation." + r.EntrypointID}, Activation: contract.Coordinate{RunID: r.Origin.RunID, EntrypointID: r.EntrypointID, ActivationID: "actual-" + r.EntrypointID}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}
				return []contract.ReservationHandle{h}, nil
			}
			host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "ok"), nil }}, nil
			}
			s, err := newScheduler(p, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			require.NoError(t, s.execute(context.Background()))
			// The second entrypoint's reservation, released canceled by its parent's completion.
			released := schedulerCompletion{
				reservation: &scheduledReservation{identity: contract.ReservationIdentity{Origin: origin, EntrypointID: "workflow_second", ID: "reservation.workflow_second"}, source: "scheduler.g0.n0.a1.r1", cause: "scheduler.g0.n0.a1.started"},
				result:      contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}},
			}
			decision, err := s.publishCompletion(context.Background(), released)
			if mode == "performing" {
				require.Error(t, err)
				require.Equal(t, Stop, decision)
				require.True(t, s.recorder.incomplete)
				return
			}
			require.NoError(t, err)
			require.Equal(t, Continue, decision)
			require.False(t, s.recorder.incomplete)
			recorded := false
			for _, event := range s.recorder.run.Events {
				if event.Kind == testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC && event.GetOutcome().GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED {
					recorded = true
				}
			}
			require.True(t, recorded, "the released reservation is recorded as canceled")
		})
	}
}

func TestSchedulerReservationsRetainEveryHandle(t *testing.T) {
	for _, mode := range []string{"exact", "partial", "error", "nil", "duplicate-id", "ordinal-range", "crossed", "effect-error"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			addWorker(c, &policy)
			// The carrier reserves one activation of each workflow; the host misbehaves on the second.
			second := proto.CloneOf(c.Program.Entrypoints[1])
			second.EntrypointId = "workflow_second"
			c.Program.Entrypoints = append(c.Program.Entrypoints, second)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			accepted := []contract.EffectHandle{}
			calls := 0
			host := &testsupport.Session{}
			host.OnReserve = func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				h := &testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, Ordinal: 0, ID: "reservation." + r.EntrypointID}, Activation: contract.Coordinate{RunID: r.Origin.RunID, EntrypointID: r.EntrypointID, ActivationID: "actual-" + r.EntrypointID}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}
				if r.EntrypointID == "workflow_second" {
					switch mode {
					case "partial":
						return nil, nil
					case "error":
						return nil, errors.New("reserve failed")
					case "nil":
						return []contract.ReservationHandle{(*testsupport.Reservation)(nil)}, nil
					case "duplicate-id":
						h.ID.ID = "reservation.workflow"
					case "ordinal-range":
						h.ID.Ordinal = 1
					case "crossed":
						h.ID.Origin.RunID = "other"
					default:
					}
				}
				accepted = append(accepted, h)
				return []contract.ReservationHandle{h}, nil
			}
			host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				calls++
				h := &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "ok"), nil }}
				accepted = append(accepted, h)
				if mode == "effect-error" {
					return h, errors.New("partial effect")
				}
				return h, nil
			}
			s, err := newScheduler(p, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			err = s.execute(context.Background())
			if mode == "exact" {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
				for _, event := range s.recorder.run.Events {
					if event.Kind == testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC {
						require.Equal(t, "controller.0", event.Coordinates.ActivationId)
						require.Equal(t, []string{"scheduler.g0.n0.a1.started"}, event.CausalSourceIds)
					}
				}
			} else {
				require.Error(t, err)
				want := 0
				if mode == "effect-error" {
					want = 1
				}
				require.Equal(t, want, calls)
				require.True(t, s.recorder.incomplete)
			}
			require.Equal(t, accepted, s.outstanding())
		})
	}
}

func TestSchedulerTimeoutAndProtocolBranches(t *testing.T) {
	for _, status := range []testpilotspb.InstructionOutcomeStatus{testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE} {
		t.Run(status.String(), func(t *testing.T) {
			c, catalog, policy := fixture(t)
			c.Program.Entrypoints[0].Instructions[0].Limits.MaxAttempts = proto.Int64(3)
			branch := rpcNode("branch")
			branch.Guard = succeeded("controller", "call")
			branch.Guard = cel.Compare("_==_", cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "call"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}}), cel.Literal(cel.Enum(status)))
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, branch)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			calls := 0
			host := &testsupport.Session{OnInvokeRPC: func(_ context.Context, c contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message) (contract.EffectHandle, error) {
				calls++
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					if c.InstructionID == "branch" {
						return effectResponse(p, "ok"), nil
					}
					if status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT {
						return contract.EffectResult{}, context.DeadlineExceeded
					}
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: status, ProtocolCode: "unavailable"}}, nil
				}}, nil
			}}
			s, err := newScheduler(p, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			require.NoError(t, s.execute(context.Background()))
			require.Equal(t, 2, calls)
			require.False(t, s.recorder.incomplete)
		})
	}
}
func TestSchedulerStopDuringAcceptanceRetainsBeforePublication(t *testing.T) {
	p, _, _ := dataFixture(t)
	accepted := make(chan struct{})
	release := make(chan struct{})
	waiting := make(chan struct{})
	handle := &testsupport.Effect{OnWait: func(ctx context.Context) (contract.EffectResult, error) {
		close(waiting)
		<-ctx.Done()
		return contract.EffectResult{}, ctx.Err()
	}}
	host := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		close(accepted)
		<-release
		return handle, nil
	}}
	s, err := newScheduler(p, "run", "case", host, schedulerMonitor{observe: func(e *testpilotspb.RunEvent) Decision {
		if e.SourceId == "external.stop" {
			return Stop
		}
		return Continue
	}}, time.Now)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.execute(ctx) }()
	<-accepted
	stopped := make(chan struct{})
	go func() {
		_, err := s.recorder.publish(ctx, []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: "external.stop"}}, nil)
		if err != nil {
			panic(err)
		}
		close(stopped)
	}()
	close(release)
	<-stopped
	require.Equal(t, []contract.EffectHandle{handle}, s.outstanding())
	<-waiting
	require.Error(t, s.recorder.admit(ctx, func(context.Context) ([]contract.EffectHandle, error) { panic("post Stop admission") }, s.retain))
	require.NoError(t, <-done)
	cancel()
	s.waits.Wait()
}

func (m schedulerMonitor) Close(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, error) {
	return &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, nil
}
func TestSchedulerRequestsSlotsFanoutAndClosure(t *testing.T) {
	c, catalog, policy := fixture(t)
	policy.Limits.MaxPathFanout = 3
	c.Program.Slots = []*testpilotspb.Slot{valueSlot("text", scalar(testpilotspb.SCALAR_KIND_TEXT))}
	c.Program.Observations = []*testpilotspb.Observation{{ObservationId: "item", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}}
	rpc := c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc()
	rpc.RequestAssignments = []*testpilotspb.RequestAssignment{{Target: "text", Value: textLiteral("constructed")}}
	rpc.ResponseReads = []*testpilotspb.ResponseRead{{Path: "text", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_SlotId{SlotId: "text"}}}}, {Path: "items[*]", Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "item"}}}}}
	wait := rpcNode("wait")
	wait.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: "text"}}}
	wait.After = runsAfter("controller")
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, wait)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	h := &testsupport.Session{OnInvokeRPC: func(_ context.Context, _ contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, m proto.Message) (contract.EffectHandle, error) {
		require.Equal(t, "constructed", m.ProtoReflect().Get(m.ProtoReflect().Descriptor().Fields().ByName("text")).String())
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			r := effectResponse(p, "slot")
			message := r.Response.ProtoReflect()
			items := message.Mutable(message.Descriptor().Fields().ByName("items")).List()
			for _, v := range []string{"a", "b", "c"} {
				items.Append(protoreflect.ValueOfString(v))
			}
			return r, nil
		}}, nil
	}}
	s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	var indexes []int64
	var observations []string
	for i, event := range s.recorder.run.Events {
		require.Equal(t, int64(i+1), event.Sequence)
		if i > 0 {
			require.GreaterOrEqual(t, event.Elapsed.AsDuration().Milliseconds(), s.recorder.run.Events[i-1].Elapsed.AsDuration().Milliseconds())
		}
		if len(event.Observations) > 0 {
			indexes = append(indexes, event.Coordinates.EmittedIndex)
			observations = append(observations, event.Observations[0].Value.GetStringValue())
			require.Equal(t, []string{"scheduler.g0.n0.a1.completed"}, event.CausalSourceIds)
		}
	}
	require.Equal(t, []int64{0, 1, 2}, indexes)
	require.Equal(t, []string{"a", "b", "c"}, observations)
	run, _, err := s.recorder.close(context.Background(), testpilotspb.RUN_DISPOSITION_COMPLETED, &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED})
	require.NoError(t, err)
	snapshot := proto.CloneOf(run)
	_, err = s.recorder.publish(context.Background(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: "late"}}, nil)
	require.Error(t, err)
	require.True(t, proto.Equal(snapshot, run))
	require.True(t, s.values.sealed)
	require.Equal(t, "slot", s.values.slots["text"].GetStringValue())
}

func TestSchedulerLateOrdinaryCancellationDoesNotFailCleanupSettlement(t *testing.T) {
	s := &scheduler{completions: make(chan schedulerCompletion), pending: 1}
	s.markCanceled(false)
	go func() {
		s.completions <- schedulerCompletion{err: context.Canceled}
	}()

	require.NoError(t, s.settleCompletions(t.Context(), true))
}

type boundedPublishMonitor struct {
	Monitor
	canceled chan struct{}
}

func (m *boundedPublishMonitor) Observe(ctx context.Context, event *testpilotspb.RunEvent) (Decision, error) {
	if event.GetSourceId() == "scheduler.g0.n0.a1.completed" {
		<-ctx.Done()
		close(m.canceled)
		return Continue, ctx.Err()
	}
	return Continue, nil
}

func TestSchedulerBoundsBufferedCompletionPublication(t *testing.T) {
	for _, mode := range []string{"drain expiry", "begin close"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			monitor := &boundedPublishMonitor{canceled: make(chan struct{})}
			s, err := newScheduler(prepared, "run", c.CaseId, &testsupport.Session{}, monitor, time.Now)
			require.NoError(t, err)
			s.lateTimeout = 10 * time.Millisecond
			_, err = s.recorder.publish(t.Context(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open"}}, nil)
			require.NoError(t, err)
			values, err := s.values.activate("controller", "controller.0")
			require.NoError(t, err)
			activation := &scheduledActivation{values: values}
			task := scheduledNode{activation: activation}
			s.pending = 1
			s.completions <- schedulerCompletion{node: &task, result: effectResponse(prepared, "result")}

			if mode == "drain expiry" {
				require.Error(t, s.drainCompletions(false))
			} else {
				s.beginClose()
			}
			<-monitor.canceled
		})
	}
}

func TestSchedulerMalformedAndLimitFailures(t *testing.T) {
	for _, mode := range []string{"malformed", "attempts", "events", "worker-error", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			if mode == "attempts" {
				policy.Limits.MaxAttempts = 1
				extra := rpcNode("extra")
				extra.After = runsAfter("controller")
				c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, extra)
			}
			if mode == "worker-error" {
				addWorker(c, &policy)
			}
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			h := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					if mode == "malformed" {
						return contract.EffectResult{}, nil
					}
					return effectResponse(p, "ok"), nil
				}}, nil
			}, OnReserve: func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				return []contract.ReservationHandle{&testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: "reservation"}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{}, errors.New("shared worker failed")
				}}}}, nil
			}}
			s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			if mode == "events" {
				s.recorder.maxEvents = 3
			}
			if mode == "conflict" {
				_, err = s.recorder.publish(context.Background(), []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open", ExecutionIncomplete: true}}, nil)
				require.NoError(t, err)
			}
			require.Error(t, s.execute(context.Background()))
			require.True(t, s.recorder.incomplete)
			require.NotEmpty(t, s.recorder.run.Diagnostics)
			s.waits.Wait()
		})
	}
}

type schedulerBridge struct {
	contract.HandleBridge
	ready  chan struct{}
	handle contract.OpaqueHandle
}

func (b *schedulerBridge) Await(ctx context.Context, _ string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.ready:
		return nil
	}
}
func (b *schedulerBridge) Consume(context.Context, string) (contract.OpaqueHandle, error) {
	return b.handle, nil
}
func TestSchedulerOpaqueReadinessAndCompletion(t *testing.T) {
	for _, mode := range []string{"success", "nil-bridge", "nil-handle"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := handleFixture(t)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ready := make(chan struct{})
			handle := &struct{}{}
			bridge := &schedulerBridge{ready: ready, handle: handle}
			h := &testsupport.Session{HandleBridge: bridge}
			if mode == "nil-bridge" {
				h.HandleBridge = (*schedulerBridge)(nil)
			}
			if mode == "nil-handle" {
				bridge.handle = (*struct{})(nil)
			}
			h.OnReserve = func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				return []contract.ReservationHandle{&testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: r.EntrypointID}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}}, nil
			}
			h.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				close(ready)
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "ok"), nil }}, nil
			}
			completed := false
			h.OnInvokeHandle = func(_ context.Context, c contract.Coordinate, got contract.OpaqueHandle, input proto.Message) (contract.EffectHandle, error) {
				require.Equal(t, handle, got)
				require.Equal(t, []byte(`"done"`), input.(*commonpb.Payload).GetData())
				require.Equal(t, "complete", c.InstructionID)
				completed = true
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}, nil
			}
			s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			err = s.execute(context.Background())
			s.waits.Wait()
			if mode == "success" {
				require.NoError(t, err)
				require.True(t, completed)
			} else {
				require.Error(t, err)
				require.False(t, completed)
				require.True(t, s.recorder.incomplete)
			}
			require.Empty(t, s.values.slots)
		})
	}
}

// A typed completion delivers the payload or failure it carries to the opaque handle, not an
// evaluated interpreter value.
func TestSchedulerDeliversTheCarriedCompletion(t *testing.T) {
	for _, mode := range []string{"payload", "failure"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := handleFixture(t)
			if mode == "failure" {
				c.Program.Entrypoints[0].Instructions[2].Instruction.GetNexusOperationCompletion().Result = &testpilotspb.NexusOperationCompletion_Failure{Failure: &failurepb.Failure{Message: "failed"}}
			}
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ready := make(chan struct{})
			handle := &struct{}{}
			bridge := &schedulerBridge{ready: ready, handle: handle}
			h := &testsupport.Session{HandleBridge: bridge}
			h.OnReserve = func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				return []contract.ReservationHandle{&testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: r.EntrypointID}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}}, nil
			}
			h.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				close(ready)
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "ok"), nil }}, nil
			}
			var delivered proto.Message
			h.OnInvokeHandle = func(_ context.Context, c contract.Coordinate, got contract.OpaqueHandle, input proto.Message) (contract.EffectHandle, error) {
				require.Equal(t, handle, got)
				require.Equal(t, "complete", c.InstructionID)
				delivered = input
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}}, nil
				}}, nil
			}
			s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			require.NoError(t, s.execute(context.Background()))
			s.waits.Wait()
			if mode == "failure" {
				require.True(t, proto.Equal(&failurepb.Failure{Message: "failed"}, delivered))
			} else {
				require.True(t, proto.Equal(payloadCompletion("handle").GetPayload(), delivered))
			}
		})
	}
}

func TestSchedulerStopPreventsTriggerAndReservations(t *testing.T) {
	c, catalog, policy := fixture(t)
	addWorker(c, &policy)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	h := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		panic("effect crossed Stop")
	}, OnReserve: func(context.Context, contract.ReservationRequest) ([]contract.ReservationHandle, error) {
		panic("reservation crossed Stop")
	}}
	s, err := newScheduler(p, "run", "case", h, schedulerMonitor{observe: func(e *testpilotspb.RunEvent) Decision {
		if e.Kind == testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED {
			return Stop
		}
		return Continue
	}}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Empty(t, s.outstanding())
	require.Empty(t, s.reservations)
}
