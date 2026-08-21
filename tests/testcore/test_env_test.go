package testcore

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	persistencetests "go.temporal.io/server/common/persistence/persistence-tests"
	"go.temporal.io/server/common/testing/parallelsuite"
)

func TestWithInMemorySQLitePersistence(t *testing.T) {
	var options testOptions
	WithInMemorySQLitePersistence()(&options)

	params := ApplyTestClusterOptions(options.clusterOptions)
	require.True(t, options.dedicatedCluster)
	require.Equal(t, "in-memory SQLite persistence required", options.dedicatedReason)
	require.NotEmpty(t, params.Persistence.DBName)
	got := params.Persistence
	got.DBName = ""
	want := *persistencetests.GetSQLiteMemoryTestClusterOption()
	want.DBName = ""
	require.Equal(t, want, got)
}

type TestEnvSuite struct {
	parallelsuite.Suite[*TestEnvSuite]
}

func TestTestEnvSuite(t *testing.T) {
	parallelsuite.Run(t, &TestEnvSuite{})
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_NoErrorWithoutExplicitRequest() {
	guard := newDedicatedClusterGuard(false)

	s.NoError(guard.validate())
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_FailsWhenUnused() {
	guard := newDedicatedClusterGuard(true)

	s.EqualError(guard.validate(),
		`testcore.WithDedicatedCluster() was requested but no dedicated-cluster-only feature was used`)
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_NoErrorAfterUse() {
	guard := newDedicatedClusterGuard(true)
	guard.record("global hook")

	s.NoError(guard.validate())
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_ConcurrentRecord() {
	guard := newDedicatedClusterGuard(true)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			guard.record("reason")
		})
	}
	wg.Wait()
	s.NoError(guard.validate())
}
