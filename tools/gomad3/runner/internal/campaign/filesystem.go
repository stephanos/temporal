package campaign

import (
	"context"
	"os"

	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

type mutationKind string

const (
	mutationCreate        mutationKind = "create"
	mutationFileSync      mutationKind = "file-sync"
	mutationDirectorySync mutationKind = "directory-sync"
	mutationRename        mutationKind = "rename"
	mutationDelete        mutationKind = "delete"
)

type mutationPoint struct {
	Kind      mutationKind
	Operation string
}

type mutationHook func(mutationPoint) error

type mutationHookContextKey struct{}

func withMutationHook(ctx context.Context, hook mutationHook) context.Context {
	return context.WithValue(ctx, mutationHookContextKey{}, hook)
}

func observeMutation(ctx context.Context, kind mutationKind, operation string) error {
	if ctx == nil {
		return nil
	}
	hook, ok := ctx.Value(mutationHookContextKey{}).(mutationHook)
	if !ok || hook == nil {
		return nil
	}
	return hook(mutationPoint{Kind: kind, Operation: operation})
}

func syncFileContext(ctx context.Context, file *os.File, operation string) error {
	if err := observeMutation(ctx, mutationFileSync, operation); err != nil {
		return err
	}
	return file.Sync()
}

func renameContext(ctx context.Context, oldPath, newPath, operation string) error {
	if err := observeMutation(ctx, mutationRename, operation); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

// logStoreHook bridges a context's mutation hook to the campaign log's
// append, sync, repair and creation boundaries. Each log point is observed as
// kind "log-<point>" with the log's operation name; a hook may return
// *logstore.PartialWrite at "log-write" to interrupt an append midway.
func logStoreHook(ctx context.Context) logstore.Hook {
	if ctx == nil {
		return nil
	}
	hook, ok := ctx.Value(mutationHookContextKey{}).(mutationHook)
	if !ok || hook == nil {
		return nil
	}
	return func(point logstore.Point) error {
		return hook(mutationPoint{Kind: mutationKind("log-" + string(point.Kind)), Operation: point.Operation})
	}
}
