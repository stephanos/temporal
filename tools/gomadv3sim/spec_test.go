package gomadv3sim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSpec(t *testing.T) {
	spec := validSpec()
	original := cloneSpec(spec)
	require.NoError(t, ValidateSpec(spec))
	require.Equal(t, original, spec)
}

func TestValidateSpecRejectsInvalidStructure(t *testing.T) {
	tests := map[string]func(*Spec){
		"schema": func(spec *Spec) {
			spec.Schema = "gomadv3.simulation-spec/v2"
		},
		"backend": func(spec *Spec) {
			spec.Backend = "host"
		},
		"fidelity": func(spec *Spec) {
			spec.Fidelity = "best_effort"
		},
		"hard isolation in process": func(spec *Spec) {
			spec.Fidelity = FidelityHardIsolation
		},
		"zero seed": func(spec *Spec) {
			spec.Seed = 0
		},
		"zero configured limit": func(spec *Spec) {
			spec.Limits.Nodes = 0
		},
		"configured limit above maximum": func(spec *Spec) {
			spec.Limits.DirectionalLinks = MaximumDirectionalLinks + 1
		},
		"unsorted nodes": func(spec *Spec) {
			spec.Nodes[0], spec.Nodes[1] = spec.Nodes[1], spec.Nodes[0]
		},
		"duplicate node": func(spec *Spec) {
			spec.Nodes[1].ID = spec.Nodes[0].ID
		},
		"invalid node ID": func(spec *Spec) {
			spec.Nodes[0].ID = "Server One"
		},
		"invalid boot ID": func(spec *Spec) {
			spec.Nodes[0].Boot = ""
		},
		"invalid address": func(spec *Spec) {
			spec.Nodes[0].Address = "localhost"
		},
		"duplicate address": func(spec *Spec) {
			spec.Nodes[1].Address = spec.Nodes[0].Address
		},
		"unsorted links": func(spec *Spec) {
			spec.Links[0], spec.Links[1] = spec.Links[1], spec.Links[0]
		},
		"self link": func(spec *Spec) {
			spec.Links[0].To = spec.Links[0].From
		},
		"unknown link node": func(spec *Spec) {
			spec.Links[0].To = "missing"
		},
		"unsorted volumes": func(spec *Spec) {
			spec.Volumes = append(spec.Volumes, VolumeSpec{ID: "archive", CapacityBytes: 1})
		},
		"zero volume capacity": func(spec *Spec) {
			spec.Volumes[0].CapacityBytes = 0
		},
		"unknown mounted volume": func(spec *Spec) {
			spec.Nodes[0].Volumes[0].Volume = "missing"
		},
		"relative mount": func(spec *Spec) {
			spec.Nodes[0].Volumes[0].Path = "var/lib/server"
		},
		"unclean mount": func(spec *Spec) {
			spec.Nodes[0].Volumes[0].Path = "/var/lib/../server"
		},
		"duplicate mount path": func(spec *Spec) {
			spec.Nodes[0].Volumes = append(spec.Nodes[0].Volumes, VolumeMount{Volume: "server-data", Path: "/var/lib/server"})
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validSpec()
			mutate(&spec)
			before := cloneSpec(spec)
			require.Error(t, ValidateSpec(spec))
			require.Equal(t, before, spec)
		})
	}
}

func TestValidateSpecCapacityErrors(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		mutate   func(*Spec)
	}{
		{
			name:     "nodes",
			resource: "nodes",
			mutate: func(spec *Spec) {
				spec.Limits.Nodes = 1
			},
		},
		{
			name:     "links",
			resource: "directional_links",
			mutate: func(spec *Spec) {
				spec.Limits.DirectionalLinks = 1
			},
		},
		{
			name:     "volumes",
			resource: "volumes",
			mutate: func(spec *Spec) {
				spec.Volumes = []VolumeSpec{{ID: "a", CapacityBytes: 1}, {ID: "server-data", CapacityBytes: 1024}}
				spec.Limits.Volumes = 1
			},
		},
		{
			name:     "boot config",
			resource: "boot_config_bytes",
			mutate: func(spec *Spec) {
				spec.Limits.BootConfigBytes = 1
			},
		},
		{
			name:     "volume capacity",
			resource: "volume_capacity_bytes",
			mutate: func(spec *Spec) {
				spec.Limits.VolumeCapacityBytes = 1023
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := validSpec()
			test.mutate(&spec)
			before := cloneSpec(spec)
			err := ValidateSpec(spec)
			var capacityErr *CapacityError
			require.ErrorAs(t, err, &capacityErr)
			require.Equal(t, test.resource, capacityErr.Resource)
			require.Greater(t, capacityErr.Required, capacityErr.Maximum)
			require.Equal(t, before, spec)
		})
	}
}

func TestCapacityErrorSupportsErrorsIs(t *testing.T) {
	err := &CapacityError{Resource: "nodes", Required: 2, Maximum: 1}
	require.True(t, errors.Is(err, ErrCapacity))
}

func validSpec() Spec {
	return Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     17,
		Limits:   DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client", Boot: "request-client", Address: "10.0.0.2", Config: []byte("c")},
			{ID: "server", Boot: "request-server", Address: "10.0.0.1", Config: []byte("s"), Volumes: []VolumeMount{{Volume: "server-data", Path: "/var/lib/server"}}},
		},
		Links: []LinkSpec{
			{From: "client", To: "server", Enabled: true, DelayNanos: 10},
			{From: "server", To: "client", Enabled: true, DelayNanos: 10},
		},
		Volumes: []VolumeSpec{{ID: "server-data", CapacityBytes: 1024}},
	}
}

func cloneSpec(spec Spec) Spec {
	cloned := spec
	cloned.Nodes = make([]NodeSpec, len(spec.Nodes))
	for index, node := range spec.Nodes {
		cloned.Nodes[index] = node
		cloned.Nodes[index].Config = append([]byte(nil), node.Config...)
		cloned.Nodes[index].Volumes = append([]VolumeMount(nil), node.Volumes...)
	}
	cloned.Links = append([]LinkSpec(nil), spec.Links...)
	cloned.Volumes = append([]VolumeSpec(nil), spec.Volumes...)
	return cloned
}

var _ = reflect.DeepEqual
