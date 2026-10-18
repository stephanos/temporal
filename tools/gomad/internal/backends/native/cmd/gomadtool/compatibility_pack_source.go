package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

func publishCompatibilitySourceInventory(ctx context.Context, path, platform string, review target.CapabilityReview) (retErr error) {
	staged, err := hostfs.StageContext(ctx, filepath.Dir(path), ".source-inventory-*", 0o600, func(_ context.Context, writer io.Writer) error {
		return json.NewEncoder(writer).Encode(struct {
			Schema                   string                  `json:"schema"`
			Platform                 string                  `json:"platform"`
			NativeExecutionQualified bool                    `json:"native_execution_qualified"`
			Review                   target.CapabilityReview `json:"review"`
		}{Schema: "gomad.compatibility-pack-source-review/v1", Platform: platform, Review: review})
	})
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, staged.Cleanup()) }()
	_, err = staged.PublishContext(ctx, path)
	return err
}
