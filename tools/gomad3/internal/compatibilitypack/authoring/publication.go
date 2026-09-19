package authoring

import (
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

func PublishRequest(path string, request Request) error {
	if err := ValidateRequest(request); err != nil {
		return err
	}
	encoded, err := canonicaljson.CanonicalJSON(request)
	if err != nil {
		return fmt.Errorf("encode compatibility-pack request: %w", err)
	}
	if err := hostfs.Replace(path, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("publish compatibility-pack request: %w", err)
	}
	return nil
}

func PublishReview(path string, request Request) (string, error) {
	report, digest, err := RenderReview(request)
	if err != nil {
		return "", err
	}
	if err := hostfs.Replace(path, report, 0o644); err != nil {
		return "", fmt.Errorf("publish compatibility-pack review: %w", err)
	}
	return digest, nil
}
