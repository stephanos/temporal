package local

import (
	"context"

	"go.temporal.io/sdk/client"
)

type authorityStarter interface {
	Start(context.Context) (temporalAuthority, error)
}

type temporalAuthority interface {
	Connect(context.Context) error
	SDKClient() client.Client
	StartWorker(context.Context, string, string, WorkerRegistration) error
	Stop(context.Context) error
	OwnedResources() []ownedResource
	Namespace() string
	Endpoint() string
}

type ownedResourceKind string

const (
	ownedEnvironment ownedResourceKind = "environment"
	ownedWorker      ownedResourceKind = "worker"
)

type ownedResource struct {
	kind ownedResourceKind
}

func ownedKinds(resources []ownedResource) map[ownedResourceKind]struct{} {
	kinds := make(map[ownedResourceKind]struct{}, len(resources))
	for _, resource := range resources {
		kinds[resource.kind] = struct{}{}
	}
	return kinds
}
