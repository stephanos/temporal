package namespacediagnostic

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/common"
	"go.temporal.io/server/common/persistence"
	persistencetests "go.temporal.io/server/common/persistence/persistence-tests"
	"go.temporal.io/server/common/primitives/timestamp"
	"go.temporal.io/server/common/testing/testcontext"
	"go.temporal.io/server/common/testing/testlogger"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc/status"
)

func TestNamespaceRPCCause(t *testing.T) {
	runNamespaceRPCCause(t, nil)
}

func runNamespaceRPCCause(t *testing.T, observePending func(context.Context) func()) {
	params := testcore.ApplyTestClusterOptions(nil)
	params.EnableWorkerService = false
	params.Persistence = *persistencetests.GetSQLiteMemoryTestClusterOption()
	config := &testcore.TestClusterConfig{
		FaultInjection:            params.FaultInjectionConfig,
		Persistence:               params.Persistence,
		HistoryConfig:             testcore.HistoryConfig{NumHistoryShards: cmp.Or(params.NumHistoryShards, 4)},
		DCRedirectionPolicy:       params.DCRedirectionPolicy,
		DynamicConfigOverrides:    params.DynamicConfigOverrides,
		EnableMetricsCapture:      true,
		EnableMTLS:                params.EnableMTLS,
		EnableHistoryTaskRecorder: params.EnableHistoryTaskRecorder,
		EnableReplicationRecorder: params.EnableReplicationRecorder,
		EnableArchival:            params.EnableArchival,
		AdditionalServerOptions:   params.AdditionalServerOptions,
		WorkerConfig:              testcore.WorkerConfig{DisableWorker: !params.EnableWorkerService},
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("diagnostic only: public standalone factory; same memory persistence/4 shards/metrics/worker-off/default pprof; bypass pool/namespace waiter; direct testing.T logger; literal original namespace name; config=%s\n", configJSON)
	logger := testlogger.NewTestLogger(t, testlogger.FailOnExpectedErrorOnly)
	cluster, err := testcore.NewTestClusterFactory().NewCluster(t, config, logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		logger.Close()
		fmt.Println("explicit TearDownCluster entered")
		err := cluster.TearDownCluster()
		fmt.Printf("explicit TearDownCluster returned error_type=%T error=%v\n", err, err)
		if err != nil {
			t.Error(err)
		}
	})
	ctx := testcontext.For(t)
	name := "namespace-de52f5bd-2ac9-4daa-987a-095906dcf71b"
	id := uuid.NewString()
	clusterName := cluster.TestBase().ClusterMetadata.GetCurrentClusterName()
	request := &persistence.CreateNamespaceRequest{
		Namespace: &persistencespb.NamespaceDetail{
			Info: &persistencespb.NamespaceInfo{Id: id, Name: name, State: enumspb.NAMESPACE_STATE_REGISTERED, Description: "namespace for functional tests"},
			Config: &persistencespb.NamespaceConfig{
				Retention:               timestamp.DurationFromDays(1),
				HistoryArchivalState:    enumspb.ARCHIVAL_STATE_DISABLED,
				VisibilityArchivalState: enumspb.ARCHIVAL_STATE_DISABLED,
				BadBinaries:             &namespacepb.BadBinaries{Binaries: map[string]*namespacepb.BadBinaryInfo{}},
			},
			ReplicationConfig: &persistencespb.NamespaceReplicationConfig{ActiveClusterName: clusterName, Clusters: []string{clusterName}},
			FailoverVersion:   common.EmptyVersion,
		},
		IsGlobalNamespace: false,
	}
	fmt.Println("CreateNamespace entered")
	created, err := cluster.TestBase().MetadataManager.CreateNamespace(ctx, request)
	fmt.Printf("CreateNamespace returned response=%+v error_type=%T error=%v\n", created, err, err)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("first strong DescribeNamespace entered")
	finishSnapshot := func() {}
	if observePending != nil {
		finishSnapshot = observePending(ctx)
	}
	response, describeErr := cluster.FrontendClient().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: name})
	finishSnapshot()
	fmt.Printf("first strong DescribeNamespace returned response=%v error_type=%T error=%v typed_code=%s context_error=%v\n", response, describeErr, describeErr, status.Code(describeErr), ctx.Err())
	fmt.Println("direct MetadataManager.GetNamespace entered after first RPC")
	direct, directErr := cluster.TestBase().MetadataManager.GetNamespace(ctx, &persistence.GetNamespaceRequest{Name: name})
	fmt.Printf("direct MetadataManager.GetNamespace returned response=%v error_type=%T error=%v context_error=%v\n", direct, directErr, directErr, ctx.Err())
	if describeErr != nil {
		t.Error(describeErr)
	}
	if directErr != nil {
		t.Error(directErr)
	}
}
