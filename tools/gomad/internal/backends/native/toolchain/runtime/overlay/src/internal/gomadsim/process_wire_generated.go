// Code generated from simulation/schema/processwire.json by gomadtool protocol-generate; DO NOT EDIT.

package gomadsim

import "bytes"

const maximumProcessFrameBytes = 134217728

func processRequestBlocksSimulationTime(request []byte) bool {
	return !bytes.HasPrefix(request, []byte("{\"profile\":\"gomad.simulation-process/v3\",\"kind\":\"wait\","))
}
