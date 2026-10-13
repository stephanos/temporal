package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/inspection"
)

const recoverySchema = "gomad3.recovery/v1"

type invalidRecoveryError struct {
	err error
}

func (err *invalidRecoveryError) Error() string {
	return err.err.Error()
}

func (err *invalidRecoveryError) Unwrap() error {
	return err.err
}

func IsInvalidRecoveryError(err error) bool {
	var invalidErr *invalidRecoveryError
	return errors.As(err, &invalidErr)
}

type Recovery struct {
	Schema  string               `json:"schema"`
	Path    string               `json:"path"`
	Action  string               `json:"action,omitempty"`
	Changed bool                 `json:"changed"`
	Before  inspection.Lifecycle `json:"before"`
	After   inspection.Lifecycle `json:"after"`
}

func Recover(ctx context.Context, path string) (Recovery, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Recovery{}, fmt.Errorf("resolve recovery path: %w", err)
	}
	recovered, err := campaign.RecoverCampaign(ctx, absolute)
	if err != nil {
		if campaign.IsIntegrityError(err) {
			return Recovery{}, &invalidRecoveryError{err: err}
		}
		return Recovery{}, err
	}
	return Recovery{
		Schema: recoverySchema, Path: absolute, Action: string(recovered.Action), Changed: recovered.Changed,
		Before: inspection.ProjectLifecycle(recovered.Before), After: inspection.ProjectLifecycle(recovered.After),
	}, nil
}
