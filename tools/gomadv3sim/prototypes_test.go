//go:build gomadv3_toolchain

package gomadv3sim

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrototypeTwoNodeRequestResponse(t *testing.T) {
	serverBoot := uniqueBootID("prototype-request-server")
	clientBoot := uniqueBootID("prototype-request-client")
	serverBooted := make(chan NodeContext, 1)
	clientBooted := make(chan NodeContext, 1)
	serverReady := make(chan struct{})
	clientResponse := make(chan string, 1)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		serverBooted <- node
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		defer listener.Close()
		close(serverReady)
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		defer connection.Close()
		request := make([]byte, len("request"))
		if _, err := io.ReadFull(connection, request); err != nil {
			return err
		}
		if string(request) != "request" {
			return fmt.Errorf("request = %q", request)
		}
		_, err = connection.Write(node.Config)
		return err
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, node NodeContext) error {
		clientBooted <- node
		<-serverReady
		connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", net.JoinHostPort("10.0.0.1", "7233"))
		if err != nil {
			return err
		}
		defer connection.Close()
		if _, err := connection.Write(node.Config); err != nil {
			return err
		}
		response, err := io.ReadAll(connection)
		if err != nil {
			return err
		}
		clientResponse <- string(response)
		return nil
	}))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     17,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client", Boot: clientBoot, Address: "10.0.0.2", Config: []byte("request")},
			{ID: "server", Boot: serverBoot, Address: "10.0.0.1", Config: []byte("response")},
		},
		Links: []LinkSpec{
			{From: "client", To: "server", Enabled: true},
			{From: "server", To: "client", Enabled: true},
		},
	}
	require.NoError(t, ValidateSpec(spec))
	scenario := Scenario(func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-serverReady
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		result, err := cluster.Wait(ctx, client)
		if err != nil {
			return err
		}
		if result.State != NodeStateExited {
			return fmt.Errorf("client state = %q", result.State)
		}
		return cluster.Stop(ctx, server)
	})
	result, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	serverNode := <-serverBooted
	clientNode := <-clientBooted
	require.Equal(t, NodeID("server"), serverNode.Node)
	require.Equal(t, NodeID("client"), clientNode.Node)
	require.Equal(t, []byte("response"), serverNode.Config)
	require.Equal(t, []byte("request"), clientNode.Config)
	require.Equal(t, "response", <-clientResponse)
}

func TestPrototypeRestart(t *testing.T) {
	serverBoot := uniqueBootID("prototype-restart-server")
	clientBoot := uniqueBootID("prototype-restart-client")
	firstRelease := make(chan struct{})
	serverBooted := make(chan NodeContext, 2)
	require.NoError(t, RegisterBoot(serverBoot, func(ctx context.Context, node NodeContext) error {
		serverBooted <- node
		if node.Incarnation == 1 {
			<-firstRelease
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(context.Context, NodeContext) error { return nil }))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     19,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client-after", Boot: clientBoot, Address: "10.0.0.3"},
			{ID: "client-before", Boot: clientBoot, Address: "10.0.0.2"},
			{ID: "server", Boot: serverBoot, Address: "10.0.0.1", Volumes: []VolumeMount{{Volume: "server-data", Path: "/var/lib/server"}}},
		},
		Links: []LinkSpec{
			{From: "client-after", To: "server", Enabled: true},
			{From: "client-before", To: "server", Enabled: true},
			{From: "server", To: "client-after", Enabled: true},
			{From: "server", To: "client-before", Enabled: true},
		},
		Volumes: []VolumeSpec{{ID: "server-data", CapacityBytes: 1 << 20}},
	}
	require.NoError(t, ValidateSpec(spec))
	var releaseOnce sync.Once
	scenario := Scenario(func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-serverBooted
		before, err := cluster.Start(ctx, "client-before")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, before); err != nil {
			return err
		}
		if err := cluster.Crash(ctx, server); err != nil {
			return err
		}
		releaseOnce.Do(func() { close(firstRelease) })
		restarted, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		if restarted.Node != server.Node || restarted.Incarnation != server.Incarnation+1 {
			return fmt.Errorf("restart handle = %+v after %+v", restarted, server)
		}
		<-serverBooted
		after, err := cluster.Start(ctx, "client-after")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, after); err != nil {
			return err
		}
		return cluster.Stop(ctx, restarted)
	})
	result, err := Run(context.Background(), spec, scenario)
	releaseOnce.Do(func() { close(firstRelease) })
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome)
	require.Equal(t, "10.0.0.1", spec.Nodes[2].Address)
	require.Equal(t, []VolumeMount{{Volume: "server-data", Path: "/var/lib/server"}}, spec.Nodes[2].Volumes)
}
