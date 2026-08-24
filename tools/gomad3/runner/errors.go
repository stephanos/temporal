package runner

import (
	"errors"

	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

func IsCapacityError(err error) bool {
	var journal *campaign.JournalCapacityError
	var artifact *campaign.ArtifactCapacityError
	return errors.As(err, &journal) || errors.As(err, &artifact)
}
