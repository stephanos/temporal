// Package preflight proves, before the canary mutates anything, that it runs in the protected
// workflow against exactly the canary's scope: the trusted ref and workflow file, the policy's
// coordinates, an existing namespace, and the pinned Case prepared under the tree's catalog. It
// claims nothing about the rest of production. The one call it makes is DescribeNamespace, a read;
// it never reads a Nexus endpoint, which needs cluster admin the canary's namespace-scoped
// credential does not have, so the endpoint's route is proved by the Run's own observation of the
// handler's reply.
package preflight

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The named refusals. Each is decided before any mutation, and creates no Run or receipt.
const (
	StatusWorkflowContext      = "workflow-context"
	StatusPolicyUnconfigured   = "policy-unconfigured"
	StatusCoordinateMismatch   = "coordinate-mismatch"
	StatusCaseMismatch         = "case-mismatch"
	StatusNamespaceMissing     = "namespace-missing"
	StatusNamespaceUnavailable = "namespace-unavailable"
)

// The workflow context GitHub Actions sets for every job.
const (
	VariableEventName   = "GITHUB_EVENT_NAME"
	VariableRepository  = "GITHUB_REPOSITORY"
	VariableRef         = "GITHUB_REF"
	VariableWorkflowRef = "GITHUB_WORKFLOW_REF"
	VariableRunID       = "GITHUB_RUN_ID"
	VariableRunAttempt  = "GITHUB_RUN_ATTEMPT"
)

// dispatchEvent is the only event the canary runs on: a manual dispatch.
const dispatchEvent = "workflow_dispatch"

// describeTimeout bounds the one read preflight makes.
const describeTimeout = 30 * time.Second

// Refusal is a preflight that did not pass: its named status and a redacted detail.
type Refusal struct {
	Status string
	Detail string
}

func (r *Refusal) Error() string { return "preflight " + r.Status + ": " + r.Detail }

// AsRefusal reports whether err is a preflight refusal.
func AsRefusal(err error) (*Refusal, bool) {
	var refusal *Refusal
	if errors.As(err, &refusal) {
		return refusal, true
	}
	return nil, false
}

// Namespaces is the one read preflight makes. It has no mutating method, so preflight cannot
// mutate the target whatever it is given.
type Namespaces interface {
	DescribeNamespace(ctx context.Context, request *workflowservice.DescribeNamespaceRequest, options ...grpc.CallOption) (*workflowservice.DescribeNamespaceResponse, error)
}

// Input is what preflight checks: the policy, the job's environment (the workflow context), the
// target's raw coordinates, the Redactor that removes them from every detail, and the namespace
// read.
type Input struct {
	Policy      *policy.Policy
	Lookup      authority.Lookup
	Coordinates authority.Coordinates
	Redactor    *authority.Redactor
	Namespaces  Namespaces
}

// Scope is what preflight proved: the invocation's ID, the coordinates as the policy's digests,
// and the pinned Case prepared for them under the canary's Driver Profile, which are the Case and
// Profile the controller runs. The ID and the digests are safe to write; the prepared Case and the
// Profile bind the raw coordinates, so nothing of them is written except through the Redactor.
type Scope struct {
	InvocationID string
	Coordinates  policy.Coordinates
	Prepared     *testpilot.PreparedCase
	Profile      testpilot.ProfileSpec
}

// Check runs every check in order, the ones that need no connection first, and returns the Scope
// or the first Refusal.
func Check(ctx context.Context, input Input) (*Scope, error) {
	if input.Policy == nil || input.Lookup == nil || input.Redactor == nil || input.Namespaces == nil {
		return nil, errors.New("preflight needs a policy, an environment, a Redactor and a namespace read")
	}
	refuse := func(named, format string, arguments ...any) (*Scope, error) {
		return nil, &Refusal{Status: named, Detail: input.Redactor.Redact(fmt.Sprintf(format, arguments...))}
	}
	canary := input.Policy
	invocationID, err := workflowContext(canary, input.Lookup)
	if err != nil {
		return refuse(StatusWorkflowContext, "%s", err)
	}
	if !canary.Configured() {
		return refuse(StatusPolicyUnconfigured, "the policy's coordinate digests are not committed yet")
	}
	digests := input.Coordinates.Digests()
	if name, differs := digests.Mismatch(canary.Coordinates); differs {
		return refuse(StatusCoordinateMismatch, "the %s coordinate's digest is not the policy's", name)
	}
	bound, err := casebinding.Bind(canary, input.Coordinates.Driver())
	if err != nil {
		return refuse(StatusCaseMismatch, "%s", err)
	}

	describeCtx, cancel := context.WithTimeout(ctx, describeTimeout)
	defer cancel()
	described, err := input.Namespaces.DescribeNamespace(describeCtx, &workflowservice.DescribeNamespaceRequest{Namespace: input.Coordinates.Namespace})
	var notFound *serviceerror.NamespaceNotFound
	var missing *serviceerror.NotFound
	switch {
	case errors.As(err, &notFound) || errors.As(err, &missing) || status.Code(err) == codes.NotFound:
		return refuse(StatusNamespaceMissing, "the canary namespace does not exist")
	case err != nil:
		return refuse(StatusNamespaceUnavailable, "describe the canary namespace: %s", err)
	case described.GetNamespaceInfo().GetName() != input.Coordinates.Namespace:
		return refuse(StatusNamespaceUnavailable, "the namespace described is not the canary namespace")
	case described.GetNamespaceInfo().GetState() != enumspb.NAMESPACE_STATE_REGISTERED:
		return refuse(StatusNamespaceUnavailable, "the canary namespace is %s, not registered", described.GetNamespaceInfo().GetState())
	}
	return &Scope{InvocationID: invocationID, Coordinates: digests, Prepared: bound.Prepared, Profile: bound.Profile}, nil
}

// workflowContext requires a manual dispatch of the policy's workflow file on its trusted ref in
// its repository, and returns the invocation ID, the Actions run and its attempt.
func workflowContext(canary *policy.Policy, lookup authority.Lookup) (string, error) {
	workflowRef := canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef
	for _, required := range []struct{ variable, want string }{
		{VariableEventName, dispatchEvent},
		{VariableRepository, canary.Repository},
		{VariableRef, canary.TrustedRef},
		{VariableWorkflowRef, workflowRef},
	} {
		if got, _ := lookup(required.variable); got != required.want {
			return "", fmt.Errorf("%s is %q, not %q", required.variable, got, required.want)
		}
	}
	var parts [2]string
	for index, variable := range []string{VariableRunID, VariableRunAttempt} {
		value, _ := lookup(variable)
		if number, err := strconv.ParseUint(value, 10, 64); err != nil || number == 0 {
			return "", fmt.Errorf("%s is %q, not a positive number", variable, value)
		}
		parts[index] = value
	}
	return parts[0] + "-" + parts[1], nil
}
