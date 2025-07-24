package testenv

import (
	"sync"
	"testing"

	"go.temporal.io/server/common/namespace"
)

type Suite struct {
	testSuiteT        *testing.T
	sharedClusterLock sync.Mutex
	sharedCluster     *cluster // shared across tests that have no custom config
}

func NewSuite(t *testing.T) *Suite {
	return &Suite{testSuiteT: t}
}

func (s *Suite) getOrCreateCluster(scenario *Scenario) (*cluster, namespace.ID) {
	var pc *cluster
	if !scenario.standalone && len(scenario.dcOverrides) == 0 {
		pc = s.getOrCreateSharedCluster(scenario, pc)
	} else {
		pc = s.createStandaloneCluster(scenario, pc)
	}

	// Namespace just for this test.
	nsName := namespace.Name(scenario.testCaseT.Name())
	nsID, err := pc.createNamespace(nsName)
	if err != nil {
		scenario.testCaseT.Fatalf("failed to create namespace %q: %v", nsName, err)
	}

	pc.SetupTest()
	scenario.testCaseT.Cleanup(func() {
		pc.TearDownTest()
	})

	return pc, nsID
}

func (s *Suite) getOrCreateSharedCluster(scenario *Scenario, pc *cluster) *cluster {
	s.sharedClusterLock.Lock()
	defer s.sharedClusterLock.Unlock()

	// Create a new shared cluster if it doesn't already exist.
	if s.sharedCluster == nil {
		s.sharedCluster = newCluster(scenario)

		// Once the entire test suite is done, tear down the shared cluster.
		s.testSuiteT.Cleanup(func() {
			s.sharedClusterLock.Lock()
			defer s.sharedClusterLock.Unlock()
			if s.sharedCluster != nil {
				s.sharedCluster.Stop()
			}
		})
	}

	// TODO: refactor so `clone` is not needed
	pc = s.sharedCluster.clone(scenario)
	return pc
}

func (s *Suite) createStandaloneCluster(scenario *Scenario, pc *cluster) *cluster {
	// When there are custom configs, always start a fresh cluster just for that test.
	pc = newCluster(scenario)

	// Once the test case is done, tear down the cluster.
	s.testSuiteT.Cleanup(func() {
		pc.Stop()
	})
	return pc
}
