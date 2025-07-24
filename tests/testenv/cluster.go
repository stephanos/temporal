package testenv

import (
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/tests/testcore"
)

type DynamicConfigOverride struct {
	Key  dynamicconfig.GenericSetting
	Vals []dynamicconfig.ConstrainedValue
}

type cluster struct {
	*testcore.FunctionalTestBase
}

func newCluster(s *Scenario) *cluster {
	var opts []testcore.TestClusterOption

	dc := make(map[dynamicconfig.Key]any, len(s.dcOverrides))
	for _, cfg := range s.dcOverrides {
		dc[cfg.Key.Key()] = cfg.Vals
	}
	opts = append(opts, testcore.WithDynamicConfigOverrides(dc))

	tbase := &testcore.FunctionalTestBase{}
	tbase.SetT(s.suite.testSuiteT)       // using suite's T for logging
	tbase.SetupSuiteWithCluster(opts...) // TODO: skip creating default namespaces

	return &cluster{FunctionalTestBase: tbase}
}

func (c *cluster) clone(s *Scenario) *cluster {
	return &cluster{FunctionalTestBase: c.FunctionalTestBase.Clone(s.testCaseT)}
}

func (c *cluster) createNamespace(name namespace.Name) (namespace.ID, error) {
	return c.RegisterNamespace(name, 1, enumspb.ARCHIVAL_STATE_DISABLED, "", "")
}

func (c *cluster) Stop() {
	c.FunctionalTestBase.TearDownCluster()
}
