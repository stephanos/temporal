package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/rpc"
	"go.temporal.io/server/common/testing/testcontext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type progressHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	handled chan struct{}
}

func (s *progressHealthServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	fmt.Println("health handler entered")
	s.handled <- struct{}{}
	fmt.Println("health handler returning SERVING")
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func TestGRPCUnaryProgress(t *testing.T) {
	fmt.Println("diagnostic only: numeric fake TCP; common/rpc.Dial client; health unary handler; keepalive Time=1m Timeout=10s; test context=90s; no Temporal services, persistence, server interceptors or background workers")
	require.Equal(t, 90*time.Second, testcontext.DefaultTimeout())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	handler := &progressHealthServer{handled: make(chan struct{}, 1)}
	server := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{Time: time.Minute, Timeout: 10 * time.Second}))
	grpc_health_v1.RegisterHealthServer(server, handler)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	t.Cleanup(func() {
		fmt.Println("explicit grpc.Server.Stop entered")
		server.Stop()
		fmt.Println("explicit grpc.Server.Stop returned")
		serveErr := <-served
		fmt.Printf("grpc.Server.Serve returned error_type=%T error=%v\n", serveErr, serveErr)
		if serveErr != nil {
			t.Error(serveErr)
		}
	})
	conn, err := rpc.Dial(listener.Addr().String(), nil, log.NewNoopLogger(), metrics.NoopMetricsHandler)
	require.NoError(t, err)
	t.Cleanup(func() {
		fmt.Println("explicit grpc.ClientConn.Close entered")
		closeErr := conn.Close()
		fmt.Printf("explicit grpc.ClientConn.Close returned error_type=%T error=%v\n", closeErr, closeErr)
		if closeErr != nil {
			t.Error(closeErr)
		}
	})
	ctx := testcontext.For(t)
	fmt.Println("first unary health Check entered")
	response, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	fmt.Printf("first unary health Check returned response=%v error_type=%T error=%v typed_code=%s context_error=%v\n", response, err, err, status.Code(err), ctx.Err())
	require.NoError(t, err)
	require.NoError(t, ctx.Err())
	require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, response.GetStatus())
	select {
	case <-handler.handled:
	default:
		t.Fatal("successful unary response without handler entry")
	}
}
