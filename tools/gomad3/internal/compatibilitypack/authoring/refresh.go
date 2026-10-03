package authoring

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
)

// CapabilityReview is the target review discovery projects evidence from.
type CapabilityReview = target.CapabilityReview

// Reviewer reviews a request's target in its working directory, the module
// whose go.mod and go.sum select the candidate versions.
type Reviewer func(ctx context.Context, request Request, workingDirectory string) (CapabilityReview, error)

// RefreshSpec selects the requests a bump invalidated and how to review them.
type RefreshSpec struct {
	// Root is the pack authoring root holding requests/, reports/, packs/,
	// generation.json, and the working-directory table.
	Root string
	// Platform is the host platform, os/arch. A request that does not name
	// it cannot be reviewed here and is reported, untouched.
	Platform string
	// Invalidated maps each request ID the caller found invalidated to why.
	// Requests without an approval are always refreshed as well.
	Invalidated map[string]string
	// Profile, when set, is the host's current deterministic I/O profile.
	// A host-platform request with an adapter binding of another profile is
	// stale even when no module moved, and is refreshed too.
	Profile *ProfileIdentity
	Review  Reviewer
}

// ProfileIdentity names a deterministic I/O profile implementation.
type ProfileIdentity struct {
	Name                 string
	ImplementationSHA256 string
}

type RefreshStatus string

const (
	// RefreshCurrent requests carry an approval of the freshly discovered
	// evidence; nothing is left to do.
	RefreshCurrent RefreshStatus = "current"
	// RefreshAwaitingApproval requests carry fresh evidence and a review
	// report but no approval of it; generate --approve-review approves one.
	RefreshAwaitingApproval RefreshStatus = "awaiting-approval"
	// RefreshNotEvaluable requests govern another platform and are left
	// untouched for a host of that platform.
	RefreshNotEvaluable RefreshStatus = "not-evaluable"
	// RefreshFailed requests could not be reviewed or discovered, such as a
	// target that no longer selects an activation module.
	RefreshFailed RefreshStatus = "failed"
)

type RefreshResult struct {
	ID     string
	Status RefreshStatus
	// Reason is why the request was selected, or why it failed or cannot
	// be evaluated.
	Reason string
	// ReviewSHA256 is the review digest of the fresh evidence to approve.
	ReviewSHA256 string
	// Rewritten is true when the request's evidence or approval changed.
	Rewritten bool
	// WorkingDirectory is where the request was discovered.
	WorkingDirectory string
}

// Refresh discovers every selected request in its working directory and stops
// at approval. A request is current only when its stored approval equals the
// review digest of the freshly discovered evidence, so an approval of older
// evidence never counts. A request is rewritten, with its approval cleared,
// only when the fresh evidence differs from what it stores; the review
// reports and generated packs are then regenerated, which drops the pack of
// every request no longer approved. Approval stays per request through
// generate --approve-review.
func Refresh(ctx context.Context, spec RefreshSpec) ([]RefreshResult, error) {
	if spec.Root == "" || spec.Platform == "" || spec.Review == nil {
		return nil, errors.New("compatibility-pack refresh needs a root, platform, and reviewer")
	}
	requests, err := loadRequests(spec.Root, false)
	if err != nil {
		return nil, err
	}
	directories, err := workingDirectoriesFor(spec.Root, requests)
	if err != nil {
		return nil, err
	}
	selected := map[string]string{}
	for id, reason := range spec.Invalidated {
		if _, found := requests[id]; !found {
			return nil, &InputError{Err: fmt.Errorf("invalidated compatibility pack %s has no request under the pack root", id)}
		}
		selected[id] = reason
	}
	for id, request := range requests {
		if _, already := selected[id]; already {
			continue
		}
		if request.ApprovalSHA256 == "" {
			selected[id] = "request has no approval"
		} else if binding := staleProfileBinding(request, spec.Platform, spec.Profile); binding != "" {
			selected[id] = binding
		}
	}
	ids := make([]string, 0, len(selected))
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	results := make([]RefreshResult, 0, len(ids))
	rewritten := false
	for _, id := range ids {
		request := requests[id]
		result := RefreshResult{ID: id, Reason: selected[id], WorkingDirectory: directories[id]}
		if !slices.Contains(request.Platforms, spec.Platform) {
			result.Status = RefreshNotEvaluable
			result.Reason = fmt.Sprintf("governs %v; this host is %s; selected because %s", request.Platforms, spec.Platform, result.Reason)
			results = append(results, result)
			continue
		}
		fresh, digest, err := discoverFresh(ctx, spec.Review, request, directories[id])
		if err != nil {
			result.Status, result.Reason = RefreshFailed, err.Error()
			results = append(results, result)
			continue
		}
		result.ReviewSHA256 = digest
		if request.ApprovalSHA256 == digest {
			result.Status = RefreshCurrent
			results = append(results, result)
			continue
		}
		result.Status = RefreshAwaitingApproval
		same, err := sameRequest(request, fresh)
		if err != nil {
			return nil, err
		}
		if !same {
			requests[id] = fresh
			result.Rewritten, rewritten = true, true
		}
		results = append(results, result)
	}
	if !rewritten {
		return results, nil
	}
	rendered, err := renderGeneration(requests)
	if err != nil {
		return nil, err
	}
	if err := publishGeneration(spec.Root, rendered); err != nil {
		return nil, err
	}
	return results, nil
}

// discoverFresh reviews the request's target and discovers its evidence in
// memory; the stored request is the draft, so its selectors, facts, and
// dispositions carry over and newly observed facts arrive denied.
func discoverFresh(ctx context.Context, review Reviewer, request Request, directory string) (Request, string, error) {
	reviewed, err := review(ctx, request, directory)
	if err != nil {
		return Request{}, "", fmt.Errorf("review %s target: %w", request.ID, err)
	}
	return Discover(request, reviewed)
}

func sameRequest(left, right Request) (bool, error) {
	leftBytes, err := canonicaljson.CanonicalJSON(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := canonicaljson.CanonicalJSON(right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftBytes, rightBytes), nil
}

// staleProfileBinding describes the first adapter binding of a request for
// platform that names another profile than the current one, or returns "".
func staleProfileBinding(request Request, platform string, profile *ProfileIdentity) string {
	if profile == nil || !slices.Contains(request.Platforms, platform) {
		return ""
	}
	modules := make([]compatibility.PackModule, 0, len(request.Activation)+len(request.Packages))
	for _, activation := range request.Activation {
		modules = append(modules, activation.Evidence)
	}
	for _, pkg := range request.Packages {
		modules = append(modules, pkg.Evidence.Module)
	}
	for _, module := range modules {
		adapter := module.Replacement.Adapter
		if adapter != nil && (adapter.ProfileName != profile.Name || adapter.ProfileImplementationSHA256 != profile.ImplementationSHA256) {
			return fmt.Sprintf("binds %s to profile %s %s; the current profile is %s %s", module.Path, adapter.ProfileName, adapter.ProfileImplementationSHA256, profile.Name, profile.ImplementationSHA256)
		}
	}
	return ""
}
