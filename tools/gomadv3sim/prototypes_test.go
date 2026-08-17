package gomadv3sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrototypeTwoNodeRequestResponse(t *testing.T) {
	serverBoot := uniqueBootID("prototype-request-server")
	clientBoot := uniqueBootID("prototype-request-client")
	var boots []NodeContext
	var serverEndpoint string
	var serverHandler http.Handler
	var clientResponse string
	require.NoError(t, RegisterBoot(serverBoot, func(_ context.Context, node NodeContext) error {
		boots = append(boots, node)
		serverEndpoint = net.JoinHostPort(node.Address, "7233")
		response := append([]byte(nil), node.Config...)
		serverHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			payload, err := io.ReadAll(request.Body)
			if err != nil || request.Method != http.MethodPost || request.URL.Path != "/request" || !bytes.Equal(payload, []byte("request")) {
				http.Error(writer, "invalid request", http.StatusBadRequest)
				return
			}
			if _, err := writer.Write(response); err != nil {
				http.Error(writer, "write response", http.StatusInternalServerError)
			}
		})
		return nil
	}))
	require.NoError(t, RegisterBoot(clientBoot, func(ctx context.Context, node NodeContext) error {
		boots = append(boots, node)
		if serverHandler == nil {
			return errors.New("server handler is not initialized")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+serverEndpoint+"/request", bytes.NewReader(node.Config))
		if err != nil {
			return err
		}
		response := httptest.NewRecorder()
		serverHandler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			return fmt.Errorf("request status = %d, body = %q", response.Code, response.Body.String())
		}
		clientResponse = response.Body.String()
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
	cluster := newPrototypeCluster(t, spec, []string{"start:server:1", "start:client:1", "wait:client:1", "stop:server:1"})
	scenario := Scenario(func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
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
	require.NoError(t, scenario(context.Background(), cluster))
	cluster.requireComplete()
	require.Equal(t, []NodeID{"server", "client"}, []NodeID{boots[0].Node, boots[1].Node})
	require.Equal(t, []byte("response"), boots[0].Config)
	require.Equal(t, []byte("request"), boots[1].Config)
	require.Equal(t, "response", clientResponse)
}

func TestPrototypeRestart(t *testing.T) {
	serverBoot := uniqueBootID("prototype-restart-server")
	clientBoot := uniqueBootID("prototype-restart-client")
	require.NoError(t, RegisterBoot(serverBoot, func(context.Context, NodeContext) error { return nil }))
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
	cluster := newPrototypeCluster(t, spec, []string{
		"start:server:1",
		"start:client-before:1",
		"wait:client-before:1",
		"crash:server:1",
		"restart:server:2",
		"start:client-after:1",
		"wait:client-after:1",
		"stop:server:2",
	})
	scenario := Scenario(func(ctx context.Context, cluster Cluster) error {
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
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
		restarted, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		if restarted.Node != server.Node || restarted.Incarnation != server.Incarnation+1 {
			return fmt.Errorf("restart handle = %+v after %+v", restarted, server)
		}
		after, err := cluster.Start(ctx, "client-after")
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, after); err != nil {
			return err
		}
		return cluster.Stop(ctx, restarted)
	})
	require.NoError(t, scenario(context.Background(), cluster))
	cluster.requireComplete()
	require.Equal(t, "10.0.0.1", cluster.nodes["server"].Address)
	require.Equal(t, []VolumeMount{{Volume: "server-data", Path: "/var/lib/server"}}, cluster.nodes["server"].Volumes)
}

type prototypeCluster struct {
	t            *testing.T
	nodes        map[NodeID]NodeSpec
	incarnations map[NodeID]uint64
	states       map[NodeHandle]NodeState
	wantActions  []string
	actions      []string
}

func newPrototypeCluster(t *testing.T, spec Spec, wantActions []string) *prototypeCluster {
	t.Helper()
	nodes := make(map[NodeID]NodeSpec, len(spec.Nodes))
	for _, node := range spec.Nodes {
		nodes[node.ID] = node
	}
	return &prototypeCluster{
		t:            t,
		nodes:        nodes,
		incarnations: make(map[NodeID]uint64),
		states:       make(map[NodeHandle]NodeState),
		wantActions:  wantActions,
	}
}

func (cluster *prototypeCluster) Start(ctx context.Context, id NodeID) (NodeHandle, error) {
	return cluster.start(ctx, id, "start")
}

func (cluster *prototypeCluster) Wait(_ context.Context, handle NodeHandle) (NodeResult, error) {
	if cluster.states[handle] != NodeStateRunning {
		return NodeResult{}, fmt.Errorf("wait for non-running handle %+v", handle)
	}
	cluster.record("wait", handle)
	cluster.states[handle] = NodeStateExited
	return NodeResult{Handle: handle, State: NodeStateExited}, nil
}

func (cluster *prototypeCluster) Stop(_ context.Context, handle NodeHandle) error {
	if cluster.states[handle] != NodeStateRunning {
		return fmt.Errorf("stop non-running handle %+v", handle)
	}
	cluster.record("stop", handle)
	cluster.states[handle] = NodeStateStopped
	return nil
}

func (cluster *prototypeCluster) Crash(_ context.Context, handle NodeHandle) error {
	if cluster.states[handle] != NodeStateRunning {
		return fmt.Errorf("crash non-running handle %+v", handle)
	}
	cluster.record("crash", handle)
	cluster.states[handle] = NodeStateCrashed
	return nil
}

func (cluster *prototypeCluster) Restart(ctx context.Context, id NodeID) (NodeHandle, error) {
	return cluster.start(ctx, id, "restart")
}

func (cluster *prototypeCluster) start(ctx context.Context, id NodeID, action string) (NodeHandle, error) {
	node, ok := cluster.nodes[id]
	if !ok {
		return NodeHandle{}, fmt.Errorf("unknown node %q", id)
	}
	if action == "restart" {
		prior := NodeHandle{Node: id, Incarnation: cluster.incarnations[id]}
		if cluster.states[prior] != NodeStateCrashed && cluster.states[prior] != NodeStateStopped {
			return NodeHandle{}, fmt.Errorf("restart live node %q", id)
		}
	}
	cluster.incarnations[id]++
	handle := NodeHandle{Node: id, Incarnation: cluster.incarnations[id]}
	boot, ok := RegisteredBoot(node.Boot)
	if !ok {
		return NodeHandle{}, fmt.Errorf("unregistered boot %q", node.Boot)
	}
	config := append([]byte(nil), node.Config...)
	if err := boot(ctx, NodeContext{NodeHandle: handle, Address: node.Address, Config: config}); err != nil {
		return NodeHandle{}, err
	}
	cluster.states[handle] = NodeStateRunning
	cluster.record(action, handle)
	return handle, nil
}

func (cluster *prototypeCluster) record(action string, handle NodeHandle) {
	cluster.actions = append(cluster.actions, fmt.Sprintf("%s:%s:%d", action, handle.Node, handle.Incarnation))
}

func (cluster *prototypeCluster) requireComplete() {
	cluster.t.Helper()
	require.Equal(cluster.t, cluster.wantActions, cluster.actions)
}
