// Code generated from simulation/schema/processwire.json by gomadtool protocol-generate; DO NOT EDIT.

package gomadsim_test

import (
	"internal/gomadsim"
	"testing"
)

func TestGeneratedProcessWaitVector(t *testing.T) {
	if gomadsim.ProcessRequestBlocksSimulationTime([]byte("{\"profile\":\"gomad.simulation-process/v3\",\"kind\":\"wait\",\"request\":7}")) {
		t.Fatal("generated host wait frame blocks simulation time")
	}
}
