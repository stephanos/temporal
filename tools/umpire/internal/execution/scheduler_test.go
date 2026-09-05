package execution

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type schedulerHost struct {
	Session
	bridge   SlotBridge
	complete func(context.Context, Coordinate, OpaqueCapability, *umpirespb.Value) (EffectHandle, error)
	invoke   func(context.Context, Coordinate, proto.Message) (EffectHandle, error)
	reserve  func(context.Context, ReservationRequest) ([]ReservationHandle, error)
}

func (h *schedulerHost) InvokeRPC(ctx context.Context, c Coordinate, _ string, _ protoreflect.MethodDescriptor, m proto.Message) (EffectHandle, error) {
	return h.invoke(ctx, c, m)
}
func (h *schedulerHost) Reserve(ctx context.Context, r ReservationRequest) ([]ReservationHandle, error) {
	return h.reserve(ctx, r)
}
func (*schedulerHost) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error { return nil }

type schedulerEffect struct {
	EffectHandle
	wait func(context.Context) (EffectResult, error)
}

func (h *schedulerEffect) Wait(ctx context.Context) (EffectResult, error) { return h.wait(ctx) }

type schedulerMonitor struct {
	Monitor
	observe func(*umpirespb.RunEvent) Decision
}

func (m schedulerMonitor) Observe(_ context.Context, e *umpirespb.RunEvent) (Decision, error) {
	if m.observe != nil {
		return m.observe(e), nil
	}
	return Continue, nil
}
func TestSchedulerProjectsActualValues(t *testing.T) {
	p, _, _ := dataFixture(t)
	h := &schedulerHost{invoke: func(_ context.Context, c Coordinate, _ proto.Message) (EffectHandle, error) {
		require.Equal(t, int64(1), c.Attempt)
		return &schedulerEffect{wait: func(context.Context) (EffectResult, error) { return effectResponse(p, "kept"), nil }}, nil
	}}
	s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Equal(t, "kept", s.values.slots["text"].GetText())
	require.Len(t, s.outstanding(), 1)
	require.Equal(t, umpirespb.RUN_EVENT_KIND_ACTIVATION_CLOSED, s.recorder.run.Events[len(s.recorder.run.Events)-1].Kind)
	require.Error(t, s.execute(context.Background()))
}

func TestSchedulerDependencyConcurrencyGuardsAndIsolation(t *testing.T) {
	c, catalog, policy := fixture(t)
	first := c.Program.Entrypoints[0].Nodes[0]
	first.Bounds.MaxAttempts = 3
	second := rpcNode("second")
	skipped := rpcNode("skipped")
	skipped.Guard = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: false}}}}
	consumer := rpcNode("consumer")
	consumer.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}, {EntrypointId: "controller", InstructionId: "skipped"}}
	consumer.Guard = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Negation{Negation: &umpirespb.NotExpression{Operand: present(&umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: "controller", InstructionId: "skipped"}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})}}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, second, skipped, consumer)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	for _, run := range []string{"one", "two"} {
		t.Run(run, func(t *testing.T) {
			calls := make(chan Coordinate, 3)
			release := make(chan struct{})
			host := &schedulerHost{invoke: func(_ context.Context, c Coordinate, _ proto.Message) (EffectHandle, error) {
				calls <- c
				return &schedulerEffect{wait: func(ctx context.Context) (EffectResult, error) {
					select {
					case <-release:
						return effectResponse(p, "ok"), nil
					case <-ctx.Done():
						return EffectResult{}, ctx.Err()
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

type schedulerReservation struct {
	schedulerEffect
	identity   ReservationIdentity
	activation Coordinate
}

func (h *schedulerReservation) Identity() ReservationIdentity               { return h.identity }
func (h *schedulerReservation) Consume(context.Context) (Coordinate, error) { return h.activation, nil }
func TestSchedulerReservationsRetainEveryHandle(t *testing.T) {
	for _, mode := range []string{"exact", "partial", "error", "nil", "duplicate-id", "duplicate-ordinal", "crossed", "effect-error"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			addWorker(c)
			c.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 2}}
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			accepted := []EffectHandle{}
			calls := 0
			host := &schedulerHost{}
			host.reserve = func(_ context.Context, r ReservationRequest) ([]ReservationHandle, error) {
				result := []ReservationHandle{}
				for i := int64(0); i < 2; i++ {
					h := &schedulerReservation{identity: ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, Ordinal: i, ID: fmt.Sprintf("reservation.%d", i)}, activation: Coordinate{RunID: r.Origin.RunID, EntrypointID: r.EntrypointID, ActivationID: fmt.Sprintf("actual-worker.%d", i)}, schedulerEffect: schedulerEffect{wait: func(context.Context) (EffectResult, error) {
						return EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
					}}}
					if i == 1 {
						switch mode {
						case "partial":
							return result, nil
						case "error":
							return result, errors.New("reserve failed")
						case "nil":
							result = append(result, (*schedulerReservation)(nil))
							return result, nil
						case "duplicate-id":
							h.identity.ID = "reservation.0"
						case "duplicate-ordinal":
							h.identity.Ordinal = 0
						case "crossed":
							h.identity.Origin.RunID = "other"
						default:
						}
					}
					result = append(result, h)
					accepted = append(accepted, h)
				}
				return result, nil
			}
			host.invoke = func(context.Context, Coordinate, proto.Message) (EffectHandle, error) {
				calls++
				h := &schedulerEffect{wait: func(context.Context) (EffectResult, error) { return effectResponse(p, "ok"), nil }}
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
					if event.Kind == umpirespb.RUN_EVENT_KIND_DIAGNOSTIC {
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
	for _, status := range []umpirespb.InstructionOutcomeStatus{umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS} {
		t.Run(status.String(), func(t *testing.T) {
			c, catalog, policy := fixture(t)
			c.Program.Entrypoints[0].Nodes[0].Bounds.MaxAttempts = 3
			branch := rpcNode("branch")
			branch.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
			branch.Guard = succeeded("controller", "call")
			branch.Guard.GetEquals().Right.GetLiteral().GetEnumValue().Number = int32(status)
			c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, branch)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			calls := 0
			host := &schedulerHost{invoke: func(_ context.Context, c Coordinate, _ proto.Message) (EffectHandle, error) {
				calls++
				return &schedulerEffect{wait: func(context.Context) (EffectResult, error) {
					if c.InstructionID == "branch" {
						return effectResponse(p, "ok"), nil
					}
					if status == umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT {
						return EffectResult{}, context.DeadlineExceeded
					}
					return EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: status, ProtocolCode: "unavailable"}}, nil
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
	handle := &schedulerEffect{wait: func(ctx context.Context) (EffectResult, error) {
		close(waiting)
		<-ctx.Done()
		return EffectResult{}, ctx.Err()
	}}
	host := &schedulerHost{invoke: func(context.Context, Coordinate, proto.Message) (EffectHandle, error) {
		close(accepted)
		<-release
		return handle, nil
	}}
	s, err := newScheduler(p, "run", "case", host, schedulerMonitor{observe: func(e *umpirespb.RunEvent) Decision {
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
		_, err := s.recorder.publish(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: "external.stop"}}, nil)
		if err != nil {
			panic(err)
		}
		close(stopped)
	}()
	close(release)
	<-stopped
	require.Equal(t, []EffectHandle{handle}, s.outstanding())
	<-waiting
	require.Error(t, s.recorder.admit(ctx, func(context.Context) ([]EffectHandle, error) { panic("post Stop admission") }, s.retain))
	require.NoError(t, <-done)
	cancel()
	s.waits.Wait()
}

func (m schedulerMonitor) Close(context.Context, *umpirespb.Run) (*umpirespb.Verdict, error) {
	return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_SATISFIED}, nil
}
func TestSchedulerRequestsSlotsFanoutAndClosure(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxPathFanout = 3
	c.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "text", Kind: umpirespb.SLOT_KIND_VALUE, Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	c.Program.Observations = []*umpirespb.ObservationSchema{{ObservationId: "item", Type: scalar(umpirespb.SCALAR_KIND_TEXT)}}
	rpc := c.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc()
	rpc.RequestAssignments = []*umpirespb.RequestAssignment{{Target: field("text"), Value: textLiteral("constructed")}}
	rpc.ResponseProjections = []*umpirespb.ResponseProjection{{Source: field("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "text"}}}}, {Source: &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: "items", Selector: &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}}}}, Cardinality: umpirespb.PROJECTION_CARDINALITY_EMIT_EACH, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_ObservationId{ObservationId: "item"}}}}}
	wait := rpcNode("wait")
	wait.Instruction = &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitSlot{AwaitSlot: &umpirespb.AwaitSlot{SlotId: "text"}}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, wait)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	h := &schedulerHost{invoke: func(_ context.Context, _ Coordinate, m proto.Message) (EffectHandle, error) {
		require.Equal(t, "constructed", m.ProtoReflect().Get(m.ProtoReflect().Descriptor().Fields().ByName("text")).String())
		return &schedulerEffect{wait: func(context.Context) (EffectResult, error) {
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
			require.GreaterOrEqual(t, event.ElapsedMilliseconds, s.recorder.run.Events[i-1].ElapsedMilliseconds)
		}
		if len(event.Observations) > 0 {
			indexes = append(indexes, event.Coordinates.EmittedIndex)
			observations = append(observations, event.Observations[0].Value.GetText())
			require.Equal(t, []string{"scheduler.g0.n0.a1.completed"}, event.CausalSourceIds)
		}
	}
	require.Equal(t, []int64{0, 1, 2}, indexes)
	require.Equal(t, []string{"a", "b", "c"}, observations)
	run, _, err := s.recorder.close(context.Background(), umpirespb.RUN_DISPOSITION_COMPLETED, &umpirespb.CleanupOutcome{Status: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED})
	require.NoError(t, err)
	snapshot := proto.CloneOf(run)
	_, err = s.recorder.publish(context.Background(), []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: "late"}}, nil)
	require.Error(t, err)
	require.True(t, proto.Equal(snapshot, run))
	require.True(t, s.values.sealed)
	require.Equal(t, "slot", s.values.slots["text"].GetText())
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

func (m *boundedPublishMonitor) Observe(ctx context.Context, event *umpirespb.RunEvent) (Decision, error) {
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
			s, err := newScheduler(prepared, "run", c.CaseId, &schedulerHost{}, monitor, time.Now)
			require.NoError(t, err)
			s.lateTimeout = 10 * time.Millisecond
			_, err = s.recorder.publish(t.Context(), []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open"}}, nil)
			require.NoError(t, err)
			values, err := s.values.activate("controller", "controller.0")
			require.NoError(t, err)
			activation := &scheduledActivation{values: values}
			task := scheduledNode{activation: activation}
			s.pending = 1
			s.completions <- schedulerCompletion{node: &task, result: effectResponse(prepared, "result")}

			if mode == "drain expiry" {
				require.Error(t, s.settleBufferedCompletions(false))
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
				c.Program.Limits.MaxAttempts = 1
				c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, rpcNode("extra"))
			}
			if mode == "worker-error" {
				addWorker(c)
				c.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}}
			}
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			h := &schedulerHost{invoke: func(context.Context, Coordinate, proto.Message) (EffectHandle, error) {
				return &schedulerEffect{wait: func(context.Context) (EffectResult, error) {
					if mode == "malformed" {
						return EffectResult{}, nil
					}
					return effectResponse(p, "ok"), nil
				}}, nil
			}, reserve: func(_ context.Context, r ReservationRequest) ([]ReservationHandle, error) {
				return []ReservationHandle{&schedulerReservation{identity: ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: "reservation"}, schedulerEffect: schedulerEffect{wait: func(context.Context) (EffectResult, error) { return EffectResult{}, errors.New("shared worker failed") }}}}, nil
			}}
			s, err := newScheduler(p, "run", "case", h, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			if mode == "events" {
				s.recorder.maxEvents = 3
			}
			if mode == "conflict" {
				_, err = s.recorder.publish(context.Background(), []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open", ExecutionIncomplete: true}}, nil)
				require.NoError(t, err)
			}
			require.Error(t, s.execute(context.Background()))
			require.True(t, s.recorder.incomplete)
			require.NotEmpty(t, s.recorder.run.Diagnostics)
			s.waits.Wait()
		})
	}
}

func (h *schedulerHost) Bridge(context.Context) (SlotBridge, error) { return h.bridge, nil }
func (h *schedulerHost) CompleteNexusOperation(ctx context.Context, c Coordinate, capability OpaqueCapability, input *umpirespb.Value) (EffectHandle, error) {
	return h.complete(ctx, c, capability, input)
}

type schedulerBridge struct {
	SlotBridge
	ready      chan struct{}
	capability OpaqueCapability
}

func (b *schedulerBridge) Await(ctx context.Context, _ string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.ready:
		return nil
	}
}
func (b *schedulerBridge) Consume(context.Context, string) (OpaqueCapability, error) {
	return b.capability, nil
}
func TestSchedulerOpaqueReadinessAndCompletion(t *testing.T) {
	for _, mode := range []string{"success", "nil-bridge", "nil-capability"} {
		t.Run(mode, func(t *testing.T) {
			c, catalog, policy := capabilityFixture(t)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ready := make(chan struct{})
			capability := &struct{}{}
			bridge := &schedulerBridge{ready: ready, capability: capability}
			h := &schedulerHost{bridge: bridge}
			if mode == "nil-bridge" {
				h.bridge = (*schedulerBridge)(nil)
			}
			if mode == "nil-capability" {
				bridge.capability = (*struct{})(nil)
			}
			h.reserve = func(_ context.Context, r ReservationRequest) ([]ReservationHandle, error) {
				return []ReservationHandle{&schedulerReservation{identity: ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: r.EntrypointID}, schedulerEffect: schedulerEffect{wait: func(context.Context) (EffectResult, error) {
					return EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}}, nil
			}
			h.invoke = func(context.Context, Coordinate, proto.Message) (EffectHandle, error) {
				close(ready)
				return &schedulerEffect{wait: func(context.Context) (EffectResult, error) { return effectResponse(p, "ok"), nil }}, nil
			}
			completed := false
			h.complete = func(_ context.Context, c Coordinate, got OpaqueCapability, input *umpirespb.Value) (EffectHandle, error) {
				require.Equal(t, capability, got)
				require.Equal(t, "done", input.GetText())
				require.Equal(t, "complete", c.InstructionID)
				completed = true
				return &schedulerEffect{wait: func(context.Context) (EffectResult, error) {
					return EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
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
func TestSchedulerStopPreventsTriggerAndReservations(t *testing.T) {
	c, catalog, policy := fixture(t)
	addWorker(c)
	c.Program.Entrypoints[0].Nodes[0].ActivationReservations = []*umpirespb.ActivationReservation{{EntrypointId: "workflow", Count: 1}}
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	h := &schedulerHost{invoke: func(context.Context, Coordinate, proto.Message) (EffectHandle, error) { panic("effect crossed Stop") }, reserve: func(context.Context, ReservationRequest) ([]ReservationHandle, error) {
		panic("reservation crossed Stop")
	}}
	s, err := newScheduler(p, "run", "case", h, schedulerMonitor{observe: func(e *umpirespb.RunEvent) Decision {
		if e.Kind == umpirespb.RUN_EVENT_KIND_INSTRUCTION_STARTED {
			return Stop
		}
		return Continue
	}}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Empty(t, s.outstanding())
	require.Empty(t, s.reservations)
}
