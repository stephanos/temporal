package testcore

import (
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/testing/taskpoller"
)

type TestEnv struct {
	*FunctionalTestBase
	Logger        log.Logger
	cluster       *TestCluster
	dynamicConfig map[dynamicconfig.Key]any
	nsName        namespace.Name
	taskPoller    *taskpoller.TaskPoller
}

func NewTestEnv(t *testing.T, dynamicConfigOverrides ...map[dynamicconfig.Key]any) *TestEnv {
	t.Parallel()

	var dcOverrides map[dynamicconfig.Key]any
	if len(dynamicConfigOverrides) > 0 {
		dcOverrides = dynamicConfigOverrides[0]
	}

	cluster := clusterPool.getOrCreateCluster(t, dcOverrides)
	tbase := clusterPool.getTestBase()

	// Create a separate namespace.
	ns := namespace.Name(RandomizeStr(t.Name()))
	if _, err := tbase.RegisterNamespace(
		ns,
		1, // 1 day retention
		enumspb.ARCHIVAL_STATE_DISABLED,
		"",
		"",
	); err != nil {
		t.Fatalf("Failed to register namespace: %v", err)
	}

	return &TestEnv{
		FunctionalTestBase: tbase,
		cluster:            cluster,
		dynamicConfig:      dcOverrides,
		nsName:             ns,
		Logger:             tbase.Logger,
		taskPoller:         taskpoller.New(t, cluster.FrontendClient(), ns.String()),
	}
}

func (s *TestEnv) GetTestCluster() *TestCluster {
	return s.cluster
}

func (s *TestEnv) FrontendClient() workflowservice.WorkflowServiceClient {
	return s.cluster.FrontendClient()
}

func (s *TestEnv) Namespace() namespace.Name {
	return s.nsName
}

func (s *TestEnv) TaskPoller() *taskpoller.TaskPoller {
	return s.taskPoller
}
