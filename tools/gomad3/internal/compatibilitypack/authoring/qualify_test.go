package authoring

import (
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

func TestQualifyRejectsCurrentReviewDrift(t *testing.T) {
	draft := validRequest()
	draft.Activation[0].Evidence = compatibility.PackModule{}
	draft.Packages[0].Evidence = compatibility.PackRule{}
	review := discoveryReview("sha256:" + strings.Repeat("4", 64))
	request, approval, err := Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	request.ApprovalSHA256 = approval
	if err := Qualify(request, review); err != nil {
		t.Fatal(err)
	}
	review.Closure.Packages[0].Sources[0].SHA256 = "sha256:" + strings.Repeat("e", 64)
	if err := Qualify(request, review); err == nil {
		t.Fatal("Qualify() accepted changed source evidence")
	}
}
