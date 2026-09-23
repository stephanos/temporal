package cli

import "go.temporal.io/server/tools/umpire/publish"

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
var Publish = publish.Publish
