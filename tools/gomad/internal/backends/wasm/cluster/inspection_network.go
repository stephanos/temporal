package simulation

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func validateInspectionNetworks(execution *cluster.WASMExecutionEvidence, p *Provider, spec cluster.Spec) error {
	if execution == nil || execution.Inspection == nil || !execution.Inspection.Complete {
		return nil
	}
	for _, incarnation := range execution.Inspection.Incarnations {
		for _, boundary := range execution.Inspection.Boundaries {
			if incarnation.TerminalOrder == 0 || incarnation.TerminalOrder >= boundary.Order {
				continue
			}
			for _, cut := range boundary.Cuts {
				if cut.Handle != incarnation.Handle {
					continue
				}
				digest, err := cluster.WASMInspectionStateCommitment("network", incarnation.Streams.Network)
				if err != nil {
					return err
				}
				if !bytes.Equal(digest[:], cut.NetworkState) {
					return fmt.Errorf("%w: terminal network state commitment", cluster.ErrWASMInspectionUnsupported)
				}
			}
		}
		if incarnation.Streams.NetworkEncoding == "" {
			continue
		}
		var boot cluster.BootID
		for _, node := range spec.Nodes {
			if node.ID == incarnation.Handle.Node {
				boot = node.Boot
			}
		}
		for _, binding := range p.IncarnationBoots {
			if binding.Handle == incarnation.Handle {
				boot = binding.Boot
			}
		}
		request, ok := p.Boots[boot]
		if !ok || request.Config.NetworkV2 == nil || p.Broker.Network == nil {
			return fmt.Errorf("%w: network encoding lacks bound configuration", cluster.ErrWASMInspectionUnsupported)
		}
		decoded, err := wasi.DecodeClusterNetworkEvidence(incarnation.Streams.Network, wasi.ClusterNetworkLimits{Network: p.Broker.Network.Limits, Instance: *request.Config.NetworkV2, TranscriptBytes: request.Config.Limits.TranscriptBytes})
		if err != nil {
			return fmt.Errorf("%w: raw network stream: %v", cluster.ErrWASMInspectionUnsupported, err)
		}
		found := false
		for _, retained := range execution.Incarnations {
			if retained.Handle == incarnation.Handle {
				found = true
				if fmt.Sprintf("sha256:%x", sha256.Sum256(decoded.Document)) != retained.NetworkSHA256 {
					return fmt.Errorf("%w: canonical network identity", cluster.ErrWASMInspectionUnsupported)
				}
			}
		}
		if !found {
			return fmt.Errorf("%w: network incarnation identity", cluster.ErrWASMInspectionUnsupported)
		}
	}
	return nil
}
