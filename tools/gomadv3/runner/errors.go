package runner

import (
	"errors"

	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
)

func IsCapacityError(err error) bool {
	var journal *campaignstore.JournalCapacityError
	var artifact *campaignstore.ArtifactCapacityError
	return errors.As(err, &journal) || errors.As(err, &artifact)
}
