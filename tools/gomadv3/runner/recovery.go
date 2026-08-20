package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/runner/internal/campaignstore"
)

const RecoverySchema = "gomadv3.recovery/v1"

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
	Schema  string                      `json:"schema"`
	Path    string                      `json:"path"`
	Action  string                      `json:"action,omitempty"`
	Changed bool                        `json:"changed"`
	Before  CampaignLifecycleInspection `json:"before"`
	After   CampaignLifecycleInspection `json:"after"`
}

func Recover(ctx context.Context, path string) (Recovery, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Recovery{}, fmt.Errorf("resolve recovery path: %w", err)
	}
	recovered, err := campaignstore.RecoverCampaign(ctx, absolute)
	if err != nil {
		if campaignstore.IsIntegrityError(err) {
			return Recovery{}, &invalidRecoveryError{err: err}
		}
		return Recovery{}, err
	}
	return Recovery{
		Schema: RecoverySchema, Path: absolute, Action: string(recovered.Action), Changed: recovered.Changed,
		Before: projectCampaignLifecycle(recovered.Before), After: projectCampaignLifecycle(recovered.After),
	}, nil
}
