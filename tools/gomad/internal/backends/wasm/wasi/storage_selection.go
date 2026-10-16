package wasi

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

const storageCrashContextBytes = 1 << 20

// StorageCrashContext is the pending inventory observed at a prior crash boundary.
// PendingSHA256 binds the underlying persistence effects, not just operation IDs.
type StorageCrashContext struct {
	PendingSHA256 string
	Operations    []volume.Operation
}

// ValidateStorageCrashSelection admits detached crash intent before executing a guest.
func ValidateStorageCrashSelection(request Request) error {
	if request.Config.Storage == nil && (request.StorageCrashSelection != nil || request.StorageCrashLimits != nil || request.StorageCrashContext != nil) {
		return invalid("storage crash controls without storage model")
	}
	for id, selected := range request.StorageCrashSelection {
		if !slices.ContainsFunc(request.Config.Storage.Volumes, func(config volume.VolumeConfig) bool { return config.ID == id }) {
			return invalid("storage crash selection volume")
		}
		seen := make(map[uint64]bool, len(selected))
		for _, operation := range selected {
			if operation == 0 {
				return invalid("storage crash selection zero operation")
			}
			if seen[operation] {
				return invalid("storage crash selection duplicate operation")
			}
			seen[operation] = true
		}
		context, exists := request.StorageCrashContext[id]
		if len(selected) == 0 && !exists {
			continue
		}
		if !exists {
			return invalid("storage crash selection requires prior observation")
		}
		pending := make(map[uint64]volume.Operation, len(context.Operations))
		for _, operation := range context.Operations {
			pending[operation.ID] = operation
		}
		for _, id := range selected {
			operation, exists := pending[id]
			if !exists {
				return invalid("storage crash selection unknown operation")
			}
			for _, dependency := range operation.Dependencies {
				if _, exists := pending[dependency]; exists && !seen[dependency] {
					return invalid("storage crash selection is not dependency-closed")
				}
			}
		}
	}
	if request.StorageCrashContext == nil {
		return nil
	}
	encoded, err := json.Marshal(request.StorageCrashContext)
	if err != nil || len(encoded) > storageCrashContextBytes {
		return invalid("storage crash context capacity")
	}
	for id, context := range request.StorageCrashContext {
		if !slices.ContainsFunc(request.Config.Storage.Volumes, func(config volume.VolumeConfig) bool { return config.ID == id }) {
			return invalid("storage crash context volume")
		}
		digest, err := hex.DecodeString(strings.TrimPrefix(context.PendingSHA256, "sha256:"))
		if err != nil || len(digest) != 32 || "sha256:"+hex.EncodeToString(digest) != context.PendingSHA256 || uint64(len(context.Operations)) > request.Config.Storage.Limits.PendingOperations {
			return invalid("storage crash context inventory")
		}
		var previous uint64
		for _, operation := range context.Operations {
			if operation.ID <= previous {
				return invalid("storage crash context operation order")
			}
			previous = operation.ID
			for _, dependency := range operation.Dependencies {
				if dependency == 0 || dependency >= operation.ID {
					return invalid("storage crash context dependency")
				}
			}
		}
	}
	return nil
}

func (e *Environment) storageCrashContext() map[string]StorageCrashContext {
	result := make(map[string]StorageCrashContext)
	operations := e.StorageOperations()
	snapshots := e.StorageSnapshots()
	for _, state := range e.namespace.storage.fs.ConformanceState().Volumes {
		pending := make(map[uint64]bool, len(state.Pending))
		for _, operation := range state.Pending {
			pending[operation.ID] = true
		}
		context := StorageCrashContext{Operations: []volume.Operation{}}
		for _, snapshot := range snapshots {
			if snapshot.ID == state.ID {
				context.PendingSHA256 = snapshot.PendingSHA256
			}
		}
		for _, operation := range operations[state.ID] {
			if pending[operation.ID] {
				context.Operations = append(context.Operations, operation)
			}
		}
		result[state.ID] = context
	}
	return result
}

func storageCrashContextMatches(expected, actual map[string]StorageCrashContext) bool {
	for id, context := range expected {
		if !reflect.DeepEqual(context, actual[id]) {
			return false
		}
	}
	return true
}
