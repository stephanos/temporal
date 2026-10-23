package wasi

import (
	"encoding/hex"
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func storageCrashContextLimit(config Config) uint64 {
	return storageStateBudgetAdd(0, storageImageLimit(config), 18)
}

// StorageCrashContext is the pending inventory observed at a prior crash boundary.
// PendingSHA256 binds the underlying persistence effects, not just operation IDs.
type StorageCrashContext struct {
	PendingSHA256 string
	Operations    []volume.Operation
}

// Clone returns a deep copy that shares no operation or dependency storage.
func (c StorageCrashContext) Clone() StorageCrashContext {
	c.Operations = slices.Clone(c.Operations)
	for index := range c.Operations {
		c.Operations[index].Dependencies = slices.Clone(c.Operations[index].Dependencies)
	}
	return c
}

// ValidateStorageCrashSelection admits detached crash intent before executing a guest.
func ValidateStorageCrashSelection(request Request) error {
	if request.Config.Storage == nil && (request.StorageCrashSelection != nil || request.StorageCrashLimits != nil || request.StorageCrashContext != nil) {
		return invalid("storage crash controls without storage model")
	}
	var selectedVolumes []string
	for id := range request.StorageCrashSelection {
		selectedVolumes = append(selectedVolumes, id)
	}
	sort.Strings(selectedVolumes)
	for _, id := range selectedVolumes {
		selected := request.StorageCrashSelection[id]
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
	if err != nil || uint64(len(encoded)) > storageCrashContextLimit(request.Config) {
		return invalid("storage crash context capacity")
	}
	var contextVolumes []string
	for id := range request.StorageCrashContext {
		contextVolumes = append(contextVolumes, id)
	}
	sort.Strings(contextVolumes)
	for _, id := range contextVolumes {
		context := request.StorageCrashContext[id]
		if !slices.ContainsFunc(request.Config.Storage.Volumes, func(config volume.VolumeConfig) bool { return config.ID == id }) {
			return invalid("storage crash context volume")
		}
		digest, err := hex.DecodeString(strings.TrimPrefix(context.PendingSHA256, "sha256:"))
		if err != nil || len(digest) != 32 || "sha256:"+hex.EncodeToString(digest) != context.PendingSHA256 || context.Operations == nil || uint64(len(context.Operations)) > request.Config.Storage.Limits.PendingOperations {
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
		observed, exists := actual[id]
		if !exists || !context.Equal(observed) {
			return false
		}
	}
	return true
}

// Equal reports exact equality, including nil versus empty slices, which
// encode differently (null versus []) in the transported context.
func (c StorageCrashContext) Equal(other StorageCrashContext) bool {
	return c.PendingSHA256 == other.PendingSHA256 && (c.Operations == nil) == (other.Operations == nil) &&
		slices.EqualFunc(c.Operations, other.Operations, storageOperationEqual)
}

// storageOperationEqual is exact field equality for observed operations.
func storageOperationEqual(left, right volume.Operation) bool {
	if (left.Dependencies == nil) != (right.Dependencies == nil) || !slices.Equal(left.Dependencies, right.Dependencies) {
		return false
	}
	return left.ID == right.ID && left.Kind == right.Kind && left.Inode == right.Inode && left.Offset == right.Offset &&
		left.Bytes == right.Bytes && left.PayloadSHA256 == right.PayloadSHA256 && left.EffectSHA256 == right.EffectSHA256 &&
		left.Path == right.Path && left.Destination == right.Destination
}
