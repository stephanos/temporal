package preflight

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"google.golang.org/grpc"
)

var testCoordinates = authority.Coordinates{
	GRPC: "canary-frontend.example.internal:7233", Namespace: "canary-namespace-7c1",
	TaskQueue: "canary-queue-2b9", HandlerQueue: "canary-handler-queue-5e3", NexusEndpoint: "canary-endpoint-8d4",
}

// configured is the committed policy with the test coordinates' digests filled in.
func configured(t *testing.T) *policy.Policy {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = policy.Coordinates{
		GRPC: policy.Digest(testCoordinates.GRPC), Namespace: policy.Digest(testCoordinates.Namespace),
		TaskQueue: policy.Digest(testCoordinates.TaskQueue), HandlerQueue: policy.Digest(testCoordinates.HandlerQueue),
		NexusEndpoint: policy.Digest(testCoordinates.NexusEndpoint),
	}
	return canary
}

func dispatch(canary *policy.Policy) map[string]string {
	return map[string]string{
		VariableEventName:   "workflow_dispatch",
		VariableRepository:  canary.Repository,
		VariableRef:         canary.TrustedRef,
		VariableWorkflowRef: canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef,
		VariableRunID:       "1234567",
		VariableRunAttempt:  "2",
	}
}

func lookupOf(values map[string]string) authority.Lookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

// namespaces answers DescribeNamespace from a fixed reply and counts the reads.
type namespaces struct {
	reads     int
	requested string
	info      *namespacepb.NamespaceInfo
	err       error
}

func (n *namespaces) DescribeNamespace(_ context.Context, request *workflowservice.DescribeNamespaceRequest, _ ...grpc.CallOption) (*workflowservice.DescribeNamespaceResponse, error) {
	n.reads++
	n.requested = request.GetNamespace()
	if n.err != nil {
		return nil, n.err
	}
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: n.info}, nil
}

func registered() *namespaces {
	return &namespaces{info: &namespacepb.NamespaceInfo{Name: testCoordinates.Namespace, State: enumspb.NAMESPACE_STATE_REGISTERED}}
}

func input(canary *policy.Policy, environment map[string]string, coordinates authority.Coordinates, reads Namespaces) Input {
	return Input{
		Policy: canary, Lookup: lookupOf(environment), Coordinates: coordinates, Namespaces: reads,
		Redactor: authority.NewRedactor(testCoordinates.GRPC, testCoordinates.Namespace, testCoordinates.TaskQueue,
			testCoordinates.HandlerQueue, testCoordinates.NexusEndpoint, coordinates.GRPC, coordinates.Namespace),
	}
}

// A dispatch of the trusted workflow against the policy's coordinates proves the scope: the
// digests, the invocation, and the pinned Case prepared for these coordinates, after one read.
func TestCheckProvesTheCanaryScope(t *testing.T) {
	canary := configured(t)
	reads := registered()
	scope, err := Check(t.Context(), input(canary, dispatch(canary), testCoordinates, reads))
	require.NoError(t, err)
	require.Equal(t, "1234567-2", scope.InvocationID)
	require.Equal(t, canary.Coordinates, scope.Coordinates)
	require.Equal(t, 1, reads.reads)
	require.Equal(t, testCoordinates.Namespace, reads.requested)

	bound, err := casebinding.Bind(canary, testCoordinates.Driver())
	require.NoError(t, err)
	require.Equal(t, bound.Prepared.Identity(), scope.Prepared.Identity(), "the Case prepared for these coordinates, under the policy's Profile")
	require.Equal(t, canary.CaseProfile, scope.Prepared.Identity().Profile)
}

// Each mismatch refuses by name, returns no Scope, and one decided without a connection makes no
// read at all. Namespaces has no mutating method, so no refusal can have mutated the target.
func TestCheckRefusesEachMismatchByName(t *testing.T) {
	type mismatch struct {
		status      string
		edit        func(canary *policy.Policy, environment map[string]string, coordinates *authority.Coordinates, reads *namespaces)
		readsBefore bool
	}
	cases := map[string]mismatch{
		"a push event": {StatusWorkflowContext, func(_ *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableEventName] = "push"
		}, false},
		"another repository": {StatusWorkflowContext, func(_ *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableRepository] = "someone/fork"
		}, false},
		"another ref": {StatusWorkflowContext, func(_ *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableRef] = "refs/heads/feature"
		}, false},
		"another workflow file": {StatusWorkflowContext, func(c *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableWorkflowRef] = c.Repository + "/.github/workflows/other.yml@" + c.TrustedRef
		}, false},
		"the workflow on another ref": {StatusWorkflowContext, func(c *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableWorkflowRef] = c.Repository + "/" + c.WorkflowPath + "@refs/heads/feature"
		}, false},
		"no run ID": {StatusWorkflowContext, func(_ *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			delete(e, VariableRunID)
		}, false},
		"a run attempt of zero": {StatusWorkflowContext, func(_ *policy.Policy, e map[string]string, _ *authority.Coordinates, _ *namespaces) {
			e[VariableRunAttempt] = "0"
		}, false},
		"an unconfigured policy": {StatusPolicyUnconfigured, func(c *policy.Policy, _ map[string]string, _ *authority.Coordinates, _ *namespaces) {
			c.Coordinates.HandlerQueue = policy.Unconfigured
		}, false},
		"another gRPC target": {StatusCoordinateMismatch, func(_ *policy.Policy, _ map[string]string, k *authority.Coordinates, _ *namespaces) {
			k.GRPC = "elsewhere.example.internal:7233"
		}, false},
		"another namespace": {StatusCoordinateMismatch, func(_ *policy.Policy, _ map[string]string, k *authority.Coordinates, _ *namespaces) {
			k.Namespace = "customer-namespace"
		}, false},
		"another task queue": {StatusCoordinateMismatch, func(_ *policy.Policy, _ map[string]string, k *authority.Coordinates, _ *namespaces) {
			k.TaskQueue = "customer-queue"
		}, false},
		"another handler queue": {StatusCoordinateMismatch, func(_ *policy.Policy, _ map[string]string, k *authority.Coordinates, _ *namespaces) {
			k.HandlerQueue = "customer-handler-queue"
		}, false},
		"another Nexus endpoint": {StatusCoordinateMismatch, func(_ *policy.Policy, _ map[string]string, k *authority.Coordinates, _ *namespaces) {
			k.NexusEndpoint = "customer-endpoint"
		}, false},
		"another Case": {StatusCaseMismatch, func(c *policy.Policy, _ map[string]string, _ *authority.Coordinates, _ *namespaces) {
			c.CaseIdentity = "0000000000000000000000000000000000000000000000000000000000000000"
		}, false},
		"a missing namespace": {StatusNamespaceMissing, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.err = serviceerror.NewNamespaceNotFound(testCoordinates.Namespace)
		}, true},
		"a namespace not found": {StatusNamespaceMissing, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.err = serviceerror.NewNotFound("namespace " + testCoordinates.Namespace + " not found")
		}, true},
		"a deleted namespace": {StatusNamespaceUnavailable, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.info.State = enumspb.NAMESPACE_STATE_DELETED
		}, true},
		"another namespace described": {StatusNamespaceUnavailable, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.info.Name = "customer-namespace"
		}, true},
		"an unreachable frontend": {StatusNamespaceUnavailable, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.err = serviceerror.NewUnavailable("dial " + testCoordinates.GRPC + ": connection refused")
		}, true},
		"a credential the namespace refuses": {StatusNamespaceUnavailable, func(_ *policy.Policy, _ map[string]string, _ *authority.Coordinates, n *namespaces) {
			n.err = serviceerror.NewPermissionDenied("not a writer on "+testCoordinates.Namespace, "")
		}, true},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			canary := configured(t)
			environment := dispatch(canary)
			coordinates := testCoordinates
			reads := registered()
			test.edit(canary, environment, &coordinates, reads)
			scope, err := Check(t.Context(), input(canary, environment, coordinates, reads))
			require.Nil(t, scope)
			refusal, ok := AsRefusal(err)
			require.True(t, ok, "not a refusal: %v", err)
			require.Equal(t, test.status, refusal.Status, refusal.Detail)
			if test.readsBefore {
				require.Equal(t, 1, reads.reads)
			} else {
				require.Zero(t, reads.reads, "a refusal decided without a connection makes no read")
			}
			for _, value := range []string{
				testCoordinates.GRPC, "canary-frontend.example.internal", testCoordinates.Namespace, testCoordinates.TaskQueue,
				testCoordinates.HandlerQueue, testCoordinates.NexusEndpoint, coordinates.GRPC, coordinates.Namespace,
			} {
				require.NotContains(t, refusal.Error(), value, "a refusal's detail never carries a coordinate")
			}
		})
	}
}

func TestCheckRequiresItsInputs(t *testing.T) {
	canary := configured(t)
	whole := input(canary, dispatch(canary), testCoordinates, registered())
	for name, edit := range map[string]func(*Input){
		"no policy":     func(i *Input) { i.Policy = nil },
		"no lookup":     func(i *Input) { i.Lookup = nil },
		"no Redactor":   func(i *Input) { i.Redactor = nil },
		"no namespaces": func(i *Input) { i.Namespaces = nil },
	} {
		t.Run(name, func(t *testing.T) {
			partial := whole
			edit(&partial)
			_, err := Check(t.Context(), partial)
			require.Error(t, err)
			_, refused := AsRefusal(err)
			require.False(t, refused, "a missing input is the caller's error, not a refusal")
		})
	}
}
