package authoring

import (
	"fmt"

	"go.temporal.io/server/tools/gomadv3/target"
)

type DriftError struct {
	PackID string
}

func (err *DriftError) Error() string {
	return fmt.Sprintf("compatibility-pack request %s no longer matches the current target review", err.PackID)
}

func Qualify(request Request, review target.CapabilityReview) error {
	currentDigest, err := ApprovalSHA256(request)
	if err != nil {
		return err
	}
	if request.ApprovalSHA256 != "" && request.ApprovalSHA256 != currentDigest {
		return &DriftError{PackID: request.ID}
	}
	_, discoveredDigest, err := Discover(request, review)
	if err != nil {
		return err
	}
	if discoveredDigest != currentDigest {
		return &DriftError{PackID: request.ID}
	}
	return nil
}
