package wire

import (
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"
)

func NetworkEndpointKey(endpoint history.NetworkEndpoint) string {
	return fmt.Sprintf("%s\x00%020d\x00%s\x00%05d", endpoint.Node, endpoint.Incarnation, endpoint.Address, endpoint.Port)
}
