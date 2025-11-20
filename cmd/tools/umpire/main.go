package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/collector/pdata/ptrace"
	coltrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/tools/umpire"
	// Import model to register models via init()
	_ "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// stringSlice is a simple flag.Value to accept multiple --model flags.
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type traceReceiver struct {
	logger      log.Logger
	umpire      *umpire.Umpire
	unmarshaler ptrace.Unmarshaler
	coltrace.UnimplementedTraceServiceServer
}

func (t *traceReceiver) Export(ctx context.Context, req *coltrace.ExportTraceServiceRequest) (*coltrace.ExportTraceServiceResponse, error) {
	var resources, scopes, spans int
	// Count spans for logging.
	for _, rs := range req.GetResourceSpans() {
		resources++
		if sps := rs.GetScopeSpans(); len(sps) > 0 {
			for _, ss := range sps {
				scopes++
				spans += len(ss.GetSpans())
			}
		}
		// legacy InstrumentationLibrarySpans not supported in used proto; ignoring
	}
	t.logger.Info(
		"umpire: received trace export",
		tag.NewStringTag("summary", fmt.Sprintf("resources=%d scopes=%d spans=%d", resources, scopes, spans)),
	)

	// Convert the incoming OTLP proto request to pdata.Traces.
	// We marshal the ExportTraceServiceRequest and use the ProtoUnmarshaler to get pdata.Traces.
	var pdata ptrace.Traces
	if t.unmarshaler != nil {
		// NOTE: The ProtoUnmarshaler expects the wire format of an ExportTraceServiceRequest
		// (or Traces) and returns pdata.Traces accordingly.
		// We ensure the request contains a flat copy of ResourceSpans.
		bytes, err := proto.Marshal(normalizeExportTraceRequest(req))
		if err == nil {
			pdata, err = t.unmarshaler.UnmarshalTraces(bytes)
			if err != nil {
				t.logger.Warn("umpire: failed to unmarshal request to pdata, falling back to empty traces", tag.Error(err))
				pdata = ptrace.NewTraces()
			}
		} else {
			t.logger.Warn("umpire: failed to marshal request to bytes, falling back to empty traces", tag.Error(err))
			pdata = ptrace.NewTraces()
		}
	} else {
		pdata = ptrace.NewTraces()
	}

	// Forward to umpire for processing.
	_ = t.umpire.AddTraces(ctx, pdata)

	return &coltrace.ExportTraceServiceResponse{}, nil
}

// normalizeExportTraceRequest ensures the request contains a ResourceSpans slice in the
// modern ScopeSpans form when possible, so downstream unmarshaling behaves predictably.
func normalizeExportTraceRequest(in *coltrace.ExportTraceServiceRequest) *coltrace.ExportTraceServiceRequest {
	if in == nil {
		return &coltrace.ExportTraceServiceRequest{}
	}

	// If the incoming request uses legacy InstrumentationLibrarySpans, keep as-is;
	// the ProtoUnmarshaler supports both. This helper exists to make any future
	// normalization straightforward if needed.
	out := &coltrace.ExportTraceServiceRequest{
		ResourceSpans: make([]*tracepb.ResourceSpans, 0, len(in.GetResourceSpans())),
	}
	for _, rs := range in.GetResourceSpans() {
		out.ResourceSpans = append(out.ResourceSpans, rs)
	}
	return out
}

func main() {
	var (
		listenAddr    string
		checkInterval time.Duration
		modelFlags    stringSlice
	)

	flag.StringVar(&listenAddr, "listen-addr", "127.0.0.1:4317", "OTLP gRPC listen address (host:port)")
	flag.DurationVar(&checkInterval, "check-interval", 1*time.Second, "period for running model checks")
	flag.Var(&modelFlags, "model", "enable a verification model (repeat flag to enable multiple)")
	flag.Parse()

	logger := log.NewCLILogger()
	logger.Info("umpire starting",
		tag.NewStringTag("listen", listenAddr),
		tag.NewStringTag("checkInterval", checkInterval.String()),
		tag.NewStringTag("models", modelFlags.String()),
	)

	// Listen TCP for OTLP gRPC.
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("umpire: failed to listen", tag.NewStringTag("addr", listenAddr), tag.Error(err))
		return
	}

	// Create umpire instance.
	u, err := umpire.New(umpire.Config{
		Logger:        logger,
		CheckInterval: checkInterval,
		ModelNames:    []string(modelFlags),
	})
	if err != nil {
		logger.Fatal("umpire: failed to create umpire", tag.Error(err))
		return
	}

	// Start umpire.
	ctx, cancel := context.WithCancel(context.Background())
	if err := u.Start(ctx); err != nil {
		logger.Fatal("umpire: failed to start umpire", tag.Error(err))
		return
	}

	// Create gRPC server.
	grpcServer := grpc.NewServer()

	// Register OTLP trace receiver.
	coltrace.RegisterTraceServiceServer(grpcServer, &traceReceiver{
		logger:      logger,
		umpire:      u,
		unmarshaler: &ptrace.ProtoUnmarshaler{},
	})

	// Serve gRPC.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("umpire: OTLP gRPC server listening", tag.NewStringTag("addr", listenAddr))
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()

	// Handle graceful shutdown on SIGINT/SIGTERM or server error.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sigCh:
		logger.Info("umpire: received shutdown signal", tag.NewStringTag("signal", s.String()))
	case serveErr := <-errCh:
		logger.Error("umpire: gRPC server stopped with error", tag.Error(serveErr))
	}

	// Stop umpire.
	cancel()
	if err := u.Stop(); err != nil {
		logger.Warn("umpire: error during stop", tag.Error(err))
	}

	// GracefulStop with a small timeout; then force Stop if needed.
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		logger.Warn("umpire: force stopping gRPC server after timeout")
		grpcServer.Stop()
	}

	logger.Info("umpire stopped")
}
