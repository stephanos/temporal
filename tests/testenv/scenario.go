package testenv

import (
	"context"
	"testing"

	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/tests/testcore"
)

type Scenario struct {
	*cluster
	suite       *Suite
	testCaseT   *testing.T
	standalone  bool
	nsName      namespace.Name
	nsID        namespace.ID
	dcOverrides []DynamicConfigOverride
}

func NewScenario(ts *Suite, t *testing.T, overrides ...DynamicConfigOverride) *Scenario {
	return newScenario(ts, t, false, overrides...)
}

func NewStandaloneScenario(ts *Suite, t *testing.T, overrides ...DynamicConfigOverride) *Scenario {
	return newScenario(ts, t, true, overrides...)
}

func newScenario(
	ts *Suite,
	t *testing.T,
	standalone bool,
	overrides ...DynamicConfigOverride,
) *Scenario {
	// Enforce parallel test execution. Non-negotiable.
	t.Parallel()

	res := &Scenario{
		suite:       ts,
		testCaseT:   t,
		standalone:  standalone,
		nsName:      namespace.Name(t.Name()),
		dcOverrides: overrides,
	}
	res.cluster, res.nsID = ts.getOrCreateCluster(res)

	// Teardown after test completed.
	t.Cleanup(func() {
		// TODO: port over TearDownTest()
	})

	return res
}

func (s *Scenario) NewContext() context.Context {
	return testcore.NewContext()
}

func (s *Scenario) T() *testing.T {
	return s.testCaseT
}

func (s *Scenario) Namespace() namespace.Name {
	return s.nsName
}

func (s *Scenario) NamespaceID() namespace.ID {
	return s.nsID
}
