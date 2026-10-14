package toolchain

import (
	"context"
	"io"
	"os"

	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

type stagedPublication interface {
	Path() string
	PublishContext(context.Context, string) (bool, error)
	Cleanup() error
}

type stagePublicationFunc func(
	context.Context,
	string,
	string,
	os.FileMode,
	func(context.Context, io.Writer) error,
) (stagedPublication, error)

func stagePublicationContext(
	ctx context.Context,
	directory string,
	pattern string,
	mode os.FileMode,
	fill func(context.Context, io.Writer) error,
) (stagedPublication, error) {
	return hostfs.StageContext(ctx, directory, pattern, mode, fill)
}
