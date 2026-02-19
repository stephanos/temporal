package mixedbrain

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/freeport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v3"
)

type portSet struct {
	FrontendGRPC       int
	FrontendMembership int
	FrontendHTTP       int
	HistoryGRPC        int
	HistoryMembership  int
	MatchingGRPC       int
	MatchingMembership int
	WorkerGRPC         int
	WorkerMembership   int
	PProf              int
	Metrics            int
}

func allocatePortSet() portSet {
	return portSet{
		FrontendGRPC:       freeport.MustGetFreePort(),
		FrontendMembership: freeport.MustGetFreePort(),
		FrontendHTTP:       freeport.MustGetFreePort(),
		HistoryGRPC:        freeport.MustGetFreePort(),
		HistoryMembership:  freeport.MustGetFreePort(),
		MatchingGRPC:       freeport.MustGetFreePort(),
		MatchingMembership: freeport.MustGetFreePort(),
		WorkerGRPC:         freeport.MustGetFreePort(),
		WorkerMembership:   freeport.MustGetFreePort(),
		PProf:              freeport.MustGetFreePort(),
		Metrics:            freeport.MustGetFreePort(),
	}
}

func (p portSet) frontendAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", p.FrontendGRPC)
}

func generateConfig(t *testing.T, configDir string, ports portSet, frontendPorts portSet, dbDir string, dynConfigPath string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(configDir, 0755))

	defaultDBPath := filepath.Join(dbDir, "temporal_default.db")
	visibilityDBPath := filepath.Join(dbDir, "temporal_visibility.db")

	config := map[string]any{
		"log": map[string]any{
			"stdout": true,
			"level":  "info",
		},
		"persistence": map[string]any{
			"defaultStore":     "sqlite-default",
			"visibilityStore":  "sqlite-visibility",
			"numHistoryShards": 1,
			"datastores": map[string]any{
				"sqlite-default": map[string]any{
					"sql": map[string]any{
						"pluginName":      "sqlite",
						"databaseName":    defaultDBPath,
						"connectAddr":     "localhost",
						"connectProtocol": "tcp",
						"connectAttributes": map[string]string{
							"cache":        "private",
							"setup":        "true",
							"journal_mode": "wal",
							"synchronous":  "2",
						},
						"maxConns":     1,
						"maxIdleConns": 1,
					},
				},
				"sqlite-visibility": map[string]any{
					"sql": map[string]any{
						"pluginName":      "sqlite",
						"databaseName":    visibilityDBPath,
						"connectAddr":     "localhost",
						"connectProtocol": "tcp",
						"connectAttributes": map[string]string{
							"cache":        "private",
							"setup":        "true",
							"journal_mode": "wal",
							"synchronous":  "2",
						},
						"maxConns":     1,
						"maxIdleConns": 1,
					},
				},
			},
		},
		"global": map[string]any{
			"membership": map[string]any{
				"maxJoinDuration":  "30s",
				"broadcastAddress": "127.0.0.1",
			},
			"pprof": map[string]any{
				"port": ports.PProf,
			},
			"metrics": map[string]any{
				"prometheus": map[string]any{
					"framework":     "tally",
					"timerType":     "histogram",
					"listenAddress": fmt.Sprintf("127.0.0.1:%d", ports.Metrics),
				},
			},
		},
		"services": map[string]any{
			"frontend": map[string]any{
				"rpc": map[string]any{
					"grpcPort":        ports.FrontendGRPC,
					"membershipPort":  ports.FrontendMembership,
					"bindOnLocalHost": true,
					"httpPort":        ports.FrontendHTTP,
				},
			},
			"matching": map[string]any{
				"rpc": map[string]any{
					"grpcPort":        ports.MatchingGRPC,
					"membershipPort":  ports.MatchingMembership,
					"bindOnLocalHost": true,
				},
			},
			"history": map[string]any{
				"rpc": map[string]any{
					"grpcPort":        ports.HistoryGRPC,
					"membershipPort":  ports.HistoryMembership,
					"bindOnLocalHost": true,
				},
			},
			"worker": map[string]any{
				"rpc": map[string]any{
					"grpcPort":        ports.WorkerGRPC,
					"membershipPort":  ports.WorkerMembership,
					"bindOnLocalHost": true,
				},
			},
		},
		"clusterMetadata": map[string]any{
			"enableGlobalNamespace":    false,
			"failoverVersionIncrement": 10,
			"masterClusterName":        "active",
			"currentClusterName":       "active",
			"clusterInformation": map[string]any{
				"active": map[string]any{
					"enabled":                true,
					"initialFailoverVersion": 1,
					"rpcName":                "frontend",
					"rpcAddress":             fmt.Sprintf("localhost:%d", frontendPorts.FrontendGRPC),
					"httpAddress":            fmt.Sprintf("localhost:%d", frontendPorts.FrontendHTTP),
				},
			},
		},
		"dcRedirectionPolicy": map[string]any{
			"policy": "noop",
		},
		"dynamicConfigClient": map[string]any{
			"filepath":     dynConfigPath,
			"pollInterval": "10s",
		},
	}

	data, err := yaml.Marshal(config)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(configDir, "development.yaml"), data, 0644)
	require.NoError(t, err)
}

type serverProcess struct {
	name    string
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	logFile *os.File
	logPath string
	done    chan error
}

func startServerProcess(t *testing.T, name, binary, configDir, logPath string) *serverProcess {
	t.Helper()
	t.Logf("Starting %s: %s", name, binary)

	logFile, err := os.Create(logPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, binary,
		"--root", "/",
		"--config", configDir,
		"--allow-no-auth",
		"start",
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = sourceRoot()
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second

	require.NoError(t, cmd.Start())

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	return &serverProcess{
		name:    name,
		cmd:     cmd,
		cancel:  cancel,
		logFile: logFile,
		logPath: logPath,
		done:    done,
	}
}

func (p *serverProcess) stop() {
	p.cancel()
	<-p.done
	_ = p.logFile.Close()
}

func (p *serverProcess) requireAlive(t *testing.T) {
	t.Helper()
	select {
	case err := <-p.done:
		t.Fatalf("%s exited unexpectedly: %v (logs: %s)", p.name, err, p.logPath)
	default:
	}
}

func waitForServerHealth(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, timeout, 500*time.Millisecond, "server at %s did not become healthy within %v", addr, timeout)
}

func registerDefaultNamespace(t *testing.T, grpcAddr string) {
	t.Helper()

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := workflowservice.NewWorkflowServiceClient(conn)

	require.Eventually(t, func() bool {
		_, err = client.RegisterNamespace(t.Context(), &workflowservice.RegisterNamespaceRequest{
			Namespace:                        "default",
			WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour),
		})
		if err == nil {
			return true
		}
		st, ok := status.FromError(err)
		return ok && st.Code() == codes.AlreadyExists
	}, 30*time.Second, time.Second, "failed to register default namespace at %s", grpcAddr)
}

func createNexusEndpoint(t *testing.T, grpcAddr, endpointName, namespace, taskQueue string) {
	t.Helper()

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := operatorservice.NewOperatorServiceClient(conn)

	require.Eventually(t, func() bool {
		_, err = client.CreateNexusEndpoint(t.Context(), &operatorservice.CreateNexusEndpointRequest{
			Spec: &nexuspb.EndpointSpec{
				Name: endpointName,
				Target: &nexuspb.EndpointTarget{
					Variant: &nexuspb.EndpointTarget_Worker_{
						Worker: &nexuspb.EndpointTarget_Worker{
							Namespace: namespace,
							TaskQueue: taskQueue,
						},
					},
				},
			},
		})
		if err == nil {
			return true
		}
		st, ok := status.FromError(err)
		return ok && st.Code() == codes.AlreadyExists
	}, 30*time.Second, time.Second, "failed to create nexus endpoint %s at %s", endpointName, grpcAddr)
}
