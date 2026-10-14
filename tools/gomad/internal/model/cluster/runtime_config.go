package cluster

import "go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"

func runtimeConfig(spec Spec) history.RuntimeConfig {
	config := history.RuntimeConfig{
		Seed: spec.Seed,
		Limits: history.RuntimeLimits{
			NetworkListeners:       spec.Limits.NetworkListeners,
			NetworkConnections:     spec.Limits.NetworkConnections,
			NetworkBytes:           spec.Limits.NetworkBytes,
			NetworkConnectionBytes: spec.Limits.NetworkConnectionBytes,
			NetworkStallNanos:      spec.Limits.NetworkStallNanos,
			NetworkTransitions:     spec.Limits.NetworkTransitions,
			VolumeOperations:       spec.Limits.VolumeOperations,
			VolumeTransitions:      spec.Limits.VolumeTransitions,
		},
	}
	for _, node := range spec.Nodes {
		value := history.RuntimeNode{ID: node.ID, Address: node.Address}
		for _, mount := range node.Volumes {
			value.Volumes = append(value.Volumes, history.RuntimeMount{Volume: mount.Volume, Path: mount.Path})
		}
		config.Nodes = append(config.Nodes, value)
	}
	for _, link := range spec.Links {
		config.Links = append(config.Links, history.RuntimeLink{From: link.From, To: link.To, Enabled: link.Enabled, DelayNanos: link.DelayNanos})
	}
	for _, volume := range spec.Volumes {
		config.Volumes = append(config.Volumes, history.RuntimeVolume{ID: volume.ID, CapacityBytes: volume.CapacityBytes})
	}
	if spec.Replay != nil {
		config.Replay = &history.RuntimeReplay{Network: spec.Replay.Network, Volumes: spec.Replay.Volumes}
	}
	return config
}
