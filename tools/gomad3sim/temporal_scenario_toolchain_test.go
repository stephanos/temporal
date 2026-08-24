//go:build gomad3_toolchain

package gomad3sim

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/collection"
)

func TestTemporalMatchingDuplicateDeliveryFailureExactlyReplays(t *testing.T) {
	type clientCommand struct {
		result chan error
	}
	type delivery struct {
		task      string
		duplicate bool
	}
	serverBoot := uniqueBootID("temporal-matching-server")
	clientBoot := uniqueBootID("temporal-matching-client")
	serverReady := make(chan struct{}, 2)
	clientReady := make(chan struct{}, 2)
	deliveries := make(chan delivery, 4)
	acknowledgements := make(chan struct{}, 4)
	clientCommands := make(chan clientCommand)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		seen := collection.NewSyncMap[string, struct{}]()
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		listenerClosed := make(chan error, 1)
		go func() {
			<-ctx.Done()
			listenerClosed <- listener.Close()
		}()
		serverReady <- struct{}{}
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				if ctx.Err() != nil {
					return errors.Join(ctx.Err(), <-listenerClosed)
				}
				return errors.Join(acceptErr, listener.Close())
			}
			payload := make([]byte, len("task-7"))
			if _, readErr := io.ReadFull(connection, payload); readErr != nil {
				return errors.Join(readErr, connection.Close(), listener.Close())
			}
			task := string(payload)
			_, duplicate := seen.GetOrSet(task, struct{}{})
			deliveries <- delivery{task: task, duplicate: duplicate}
			select {
			case <-acknowledgements:
			case <-ctx.Done():
				return errors.Join(ctx.Err(), connection.Close(), listener.Close())
			}
			_, writeErr := connection.Write([]byte{'a'})
			if closeErr := connection.Close(); writeErr != nil || closeErr != nil {
				if ctx.Err() != nil {
					return errors.Join(ctx.Err(), writeErr, closeErr, listener.Close())
				}
			}
		}
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, _ NodeContext) error {
		clientReady <- struct{}{}
		for {
			select {
			case command := <-clientCommands:
				connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", "10.0.0.1:7233")
				if err == nil {
					err = connection.SetDeadline(time.Now().Add(time.Millisecond))
				}
				if err == nil {
					_, err = connection.Write([]byte("task-7"))
				}
				if err == nil {
					ack := make([]byte, 1)
					_, err = io.ReadFull(connection, ack)
				}
				if connection != nil {
					err = errors.Join(err, connection.Close())
				}
				command.result <- err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}))
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "drop-ack", Kind: FaultDisconnect, From: "server", To: "client"},
		{ID: "restore-ack", Kind: FaultReconnect, From: "server", To: "client"},
	})
	require.NoError(t, err)
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Seed = 53
	spec.Faults = &plan
	scenario := func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		<-serverReady
		<-clientReady
		firstResult := make(chan error, 1)
		clientCommands <- clientCommand{result: firstResult}
		first := <-deliveries
		if first.task != "task-7" || first.duplicate {
			return errors.New("first Temporal task delivery was not unique")
		}
		if err := cluster.RecordOperation(ctx, HistoryOperation{ID: "delivery-1", Actor: "matching", Kind: "deliver", Invocation: 1, Completion: 2, Input: []byte(first.task), Output: []byte("accepted")}); err != nil {
			return err
		}
		if _, err := cluster.ApplyFault(ctx, plan.Actions[0]); err != nil {
			return err
		}
		acknowledgements <- struct{}{}
		if err := <-firstResult; err == nil {
			return errors.New("disconnected acknowledgement unexpectedly succeeded")
		}
		if _, err := cluster.ApplyFault(ctx, plan.Actions[1]); err != nil {
			return err
		}
		secondResult := make(chan error, 1)
		clientCommands <- clientCommand{result: secondResult}
		second := <-deliveries
		if second.task != "task-7" || !second.duplicate {
			return errors.New("Temporal retry did not expose duplicate delivery")
		}
		if err := cluster.RecordOperation(ctx, HistoryOperation{ID: "delivery-2", Actor: "matching", Kind: "deliver", Invocation: 3, Completion: 4, Input: []byte(second.task), Output: []byte("duplicate")}); err != nil {
			return err
		}
		acknowledgements <- struct{}{}
		if err := <-secondResult; err != nil {
			return err
		}
		oracle, err := NoDuplicateOrLost("temporal.matching.delivery", []string{"task-7"}, []string{first.task, second.task}, spec.Limits.ScenarioEvidenceBytes)
		if err != nil {
			return err
		}
		if err := cluster.RecordOracle(ctx, oracle); err != nil {
			return err
		}
		if err := cluster.Stop(ctx, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeOracleFailed, recorded.Outcome)
	require.NotEmpty(t, recorded.FailureIdentity)
	require.Len(t, recorded.History, 2)
	require.Len(t, recorded.Oracles, 1)
	require.False(t, recorded.Oracles[0].Passed)
	encoded, err := EncodeClusterRecord(recorded.Record)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded), MaximumClusterRecordBytes)

	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeOracleFailed, replayed.Outcome)
	require.Equal(t, recorded.FailureIdentity, replayed.FailureIdentity)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
}
