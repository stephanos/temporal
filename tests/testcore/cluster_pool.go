package testcore

import (
	"sync"
	"testing"

	"go.temporal.io/server/common/dynamicconfig"
)

var (
	clusterPool      = &sharedClusterPool{}
	clusterStartLock sync.Mutex
)

type sharedClusterPool struct {
	once      sync.Once
	tbase     *FunctionalTestBase
	userCount int
	mu        sync.Mutex
}

func (cp *sharedClusterPool) getOrCreateCluster(
	tb testing.TB,
	dynamicConfig map[dynamicconfig.Key]any,
) *TestCluster {
	cp.mu.Lock()
	cp.userCount++
	cp.mu.Unlock()

	cp.once.Do(func() {
		clusterStartLock.Lock()
		defer clusterStartLock.Unlock()

		tbase := &FunctionalTestBase{}
		// SetT requires *testing.T, so we need to assert the type
		if t, ok := tb.(*testing.T); ok {
			tbase.SetT(t)
		} else {
			tb.Fatal("TestEnv requires *testing.T, not testing.TB")
		}

		// Build options for cluster setup
		var opts []TestClusterOption
		if dynamicConfig != nil {
			opts = append(opts, WithDynamicConfigOverrides(dynamicConfig))
		}

		tbase.SetupSuiteWithCluster(opts...)
		cp.tbase = tbase
	})

	return cp.tbase.GetTestCluster()
}

func (cp *sharedClusterPool) getTestBase() *FunctionalTestBase {
	return cp.tbase
}
