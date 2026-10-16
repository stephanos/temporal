package simulation

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// InspectionControl binds the helper's pre-dump admission to authenticated raw
// prefix material. Digest claims returned by an inspected guest are not inputs.
func InspectionControl(evidence *cluster.WASMInspectionEvidence, resolved cluster.WASMInspectionResolution, check string) ([]byte, error) {
	if err := cluster.ValidateWASMInspection(evidence, nil); err != nil {
		return nil, err
	}
	if !evidence.Complete {
		return nil, cluster.ErrWASMInspectionUnsupported
	}
	type command struct {
		Kind uint32 `json:"kind"`
		ID   uint32 `json:"id"`
	}
	request := struct {
		Target     cluster.NodeHandle               `json:"target"`
		Decision   uint64                           `json:"decision"`
		Expected   cluster.WASMInspectionBoundary   `json:"expected"`
		World      string                           `json:"world_sha256"`
		Lifecycle  string                           `json:"lifecycle_sha256"`
		Streams    []map[string]string              `json:"stream_sha256"`
		Probes     []uint32                         `json:"probes"`
		Commands   []command                        `json:"commands"`
		Registries []cluster.WASMInspectionRegistry `json:"registries"`
	}{Target: resolved.Target, Decision: resolved.Boundary.Decision, Expected: resolved.Boundary, Probes: append([]uint32{}, resolved.ProbeIDs...), Commands: []command{}}
	var authenticated bool
	for _, b := range evidence.Boundaries {
		if b.Handle == resolved.Target && b.Decision == resolved.Boundary.Decision {
			authenticated = true
			request.Expected = b
			break
		}
	}
	if !authenticated || request.Expected.Order != resolved.Boundary.Order || request.Expected.Trace != resolved.Boundary.Trace || request.Expected.TimeNanos != resolved.Boundary.TimeNanos {
		return nil, errors.New("inspection resolution differs from retained boundary")
	}
	digest := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	b := request.Expected
	request.World, request.Lifecycle = digest(evidence.World[:b.WorldBytes]), digest(evidence.Lifecycle[:b.LifecycleBytes])
	for _, cut := range b.Cuts {
		for _, incarnation := range evidence.Incarnations {
			if incarnation.Handle != cut.Handle {
				continue
			}
			s := incarnation.Streams
			request.Streams = append(request.Streams, map[string]string{"choice": digest(s.Choice[:cut.ChoiceBytes]), "wasi": digest(s.WASI[:cut.WASIBytes]), "network": digest(s.Network[:cut.NetworkBytes]), "storage": digest(s.Storage[:cut.StorageBytes]), "stdout": digest(s.Stdout[:cut.StdoutBytes]), "stderr": digest(s.Stderr[:cut.StderrBytes])})
			registry := cluster.WASMInspectionRegistry{Entries: []cluster.WASMInspectionRegistration{}}
			registry.Closed = incarnation.Registry.Closed && incarnation.Registry.ClosedOrder <= b.Order
			if registry.Closed {
				registry.ClosedOrder = incarnation.Registry.ClosedOrder
			}
			for _, entry := range incarnation.Registry.Entries {
				if entry.AvailableOrder <= b.Order {
					registry.Entries = append(registry.Entries, entry)
				}
			}
			request.Registries = append(request.Registries, registry)
			if cut.Handle != resolved.Target {
				continue
			}
			for _, name := range resolved.Requested.Dumps {
				switch name {
				case "stacks":
					request.Commands = append(request.Commands, command{Kind: 1})
				case "heap":
					request.Commands = append(request.Commands, command{Kind: 2})
				default:
					for _, entry := range registry.Entries {
						if entry.Kind == "dump" && entry.Name == name {
							request.Commands = append(request.Commands, command{Kind: 3, ID: entry.ID})
						}
					}
				}
			}
			if check != "" {
				found := false
				for _, entry := range registry.Entries {
					if entry.Kind == "check" && entry.Name == check {
						request.Commands = append(request.Commands, command{Kind: 4, ID: entry.ID})
						found = true
					}
				}
				if !found {
					return nil, errors.New("unknown or unavailable inspection check")
				}
			}
		}
	}
	return json.Marshal(request)
}
