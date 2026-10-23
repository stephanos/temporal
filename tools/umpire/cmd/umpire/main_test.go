package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"umpire.local/model/umpire/runner"
	"umpire.local/temporal"
)

func files(t *testing.T) (guest, world string) {
	t.Helper()
	dir := t.TempDir()
	guest = filepath.Join(dir, "guest.wasm")
	require.NoError(t, os.WriteFile(guest, temporal.Guest(), 0o600))
	start, err := anypb.New(&workflowservice.StartActivityExecutionRequest{
		Namespace: "namespace", ActivityId: "activity",
		ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"},
		StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1},
	})
	require.NoError(t, err)
	data, err := proto.Marshal(start)
	require.NoError(t, err)
	world = filepath.Join(dir, "world.binpb")
	require.NoError(t, os.WriteFile(world, data, 0o600))
	return guest, world
}

func TestExploreNeedsAConfiguredDriver(t *testing.T) {
	guest, world := files(t)
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"explore", "--model", guest, "--world", world, "--seed", "1", "--store", t.TempDir()},
		func(string) string { return "" }, &stdout, &stderr)
	require.Equal(t, runner.ExitError, code)
	require.Contains(t, stderr.String(), act.EnvAddress+" is required")
}

func TestAnUnreachableEndpointIsRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	env := map[string]string{act.EnvAddress: address, act.EnvNamespace: "namespace", act.EnvTaskQueue: "queue"}
	guest, world := files(t)
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"explore", "--model", guest, "--world", world, "--seed", "1", "--store", t.TempDir()},
		func(name string) string { return env[name] }, &stdout, &stderr)
	require.Equal(t, runner.ExitRefused, code, "stderr: %s", stderr.String())
	require.Contains(t, stdout.String(), "published ")
	require.Contains(t, stdout.String(), "refused at event 2: driver fault: the server did not answer")
}
