package cli

import (
	"context"

	"go.temporal.io/server/tools/umpire/publish"
)

// The publisher is tools/umpire/publish; these are the names the commands use.
const (
	StatusPublished        = publish.StatusPublished
	StatusAlreadyPublished = publish.StatusAlreadyPublished
)

type (
	Publication   = publish.Publication
	ConflictError = publish.ConflictError
)

// Publish is publish.Publish.
func Publish(ctx context.Context, root, name string, contents []byte) (Publication, error) {
	return publish.Publish(ctx, root, name, contents)
}
