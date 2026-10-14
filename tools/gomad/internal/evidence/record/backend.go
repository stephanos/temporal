package record

import (
	"errors"
	"path"
	"strings"
)

func BackendIOProfile(metadata BackendMetadata) IOProfile {
	// The inventory describes the separate backend evidence contract; it does
	// not advertise native transcript probes or adapters.
	const inventory = "{}"
	return IOProfile{Name: metadata.Name, ImplementationSHA256: metadata.Provenance.SHA256, Inventory: inventory, InventorySHA256: HashBytes([]byte(inventory))}
}

func CloneBackendMetadata(metadata *BackendMetadata) *BackendMetadata {
	if metadata == nil {
		return nil
	}
	cloned := *metadata
	if metadata.Evidence != nil {
		evidence := *metadata.Evidence
		cloned.Evidence = &evidence
	}
	return &cloned
}

func BackendReferences(metadata BackendMetadata) []BackendPayload {
	references := []BackendPayload{metadata.Provenance}
	if metadata.Evidence != nil {
		references = append(references, *metadata.Evidence)
	}
	return references
}

func ValidateBackendMetadata(metadata BackendMetadata) error {
	if metadata.Name == "" || metadata.Name == "native" || (metadata.ReplayMode != ReplayObserved && metadata.ReplayMode != ReplayExact) {
		return errors.New("invalid external backend identity or replay capability")
	}
	seen := make(map[string]bool)
	for _, reference := range BackendReferences(metadata) {
		if reference.Schema == "" || !strings.HasPrefix(reference.File, "backend/") || path.Clean(reference.File) != reference.File || strings.Contains(reference.File, "\\") || reference.Bytes == 0 || seen[reference.File] {
			return errors.New("invalid external backend payload reference")
		}
		if err := validateSHA256(reference.SHA256); err != nil {
			return err
		}
		seen[reference.File] = true
	}
	return nil
}
