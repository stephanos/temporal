package cli

import (
	"fmt"

	"go.temporal.io/server/common/testing/testpilot/publish"
)

// OutsideModel resolves a root a command writes under and refuses one under the model root, both
// resolved through their symlinks first: a proposal is for review and a record is a replay's
// input, and the model never receives either.
func OutsideModel(flag, root, modelRoot string) (string, error) {
	resolvedRoot, err := publish.Resolve(root)
	if err != nil {
		return "", fmt.Errorf("%s: %w", flag, err)
	}
	resolvedModel, err := publish.Resolve(modelRoot)
	if err != nil {
		return "", fmt.Errorf("model root: %w", err)
	}
	if publish.Within(resolvedModel, resolvedRoot) {
		return "", fmt.Errorf("%s must not be under the model root %s", flag, resolvedModel)
	}
	return resolvedRoot, nil
}
