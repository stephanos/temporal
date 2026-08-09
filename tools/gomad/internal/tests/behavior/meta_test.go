//go:build !gomad

package behavior_test

import (
	"testing"

	"github.com/temporalio/gomad/metatesting"
)

func TestMetaDeterministic(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	metatesting.CheckDeterministic(t, mt)
}

func TestMetaSeeds(t *testing.T) {
	mt := metatesting.ForCurrentPackage(t)
	metatesting.CheckSeeds(t, mt, 5)
}

func TestGomad(t *testing.T) {
	runner := metatesting.ForCurrentPackage(t)
	runner.RunAllTests(t)
}
