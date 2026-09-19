package campaign

import (
	"context"
	"os"
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
