package boundary

import (
	"fmt"
	"path/filepath"
)

type CapabilityBoundary struct {
	Package     string
	Target      string
	Hook        string
	Operation   string
	Probe       string
	ProbeID     uint64
	Disposition string
}

func CapabilityProjection(root string) (string, []CapabilityBoundary, error) {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return "", nil, err
	}
	digest, err := manifestIdentity(definition)
	if err != nil {
		return "", nil, err
	}
	byProbe := make(map[string]intercept, len(definition.Intercepts))
	for _, entry := range definition.Intercepts {
		byProbe[entry.Probe] = entry
	}
	result := make([]CapabilityBoundary, 0, len(definition.Intercepts))
	for _, entry := range definition.Intercepts {
		resolved := entry
		for resolved.Disposition == "delegate" {
			resolved = byProbe[resolved.DelegatedBoundary]
		}
		var disposition string
		switch resolved.Disposition {
		case "model":
			disposition = "modeled"
		case "deny":
			disposition = "denied"
		default:
			return "", nil, fmt.Errorf("unsupported live capability boundary disposition %q", resolved.Disposition)
		}
		result = append(result, CapabilityBoundary{
			Package: entry.Package, Target: targetName(entry.Receiver, entry.Symbol), Hook: entry.Hook,
			Operation: entry.Operation, Probe: entry.Probe, ProbeID: boundaryProbeID(entry.Probe), Disposition: disposition,
		})
	}
	return digest, result, nil
}
