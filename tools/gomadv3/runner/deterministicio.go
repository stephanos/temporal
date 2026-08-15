package runner

import (
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
)

func executionAdapters(adapters []deterministicio.BuildAdapter) []evidence.TargetAdapter {
	result := make([]evidence.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = evidence.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func deterministicAdapters(adapters []evidence.TargetAdapter) []deterministicio.Adapter {
	result := make([]deterministicio.Adapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = deterministicio.Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func recordedIOProfile(profile deterministicio.Spec) evidence.IOProfile {
	return evidence.IOProfile{
		Name:                 profile.Name(),
		ImplementationSHA256: evidence.SHA256(profile.ImplementationSHA256()),
		Inventory:            string(profile.Inventory()),
		InventorySHA256:      evidence.SHA256(profile.InventorySHA256()),
	}
}

func recordedCapturedInputs(manifest deterministicio.CapturedInputsManifest) evidence.ReadOnlyMounts {
	return evidence.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: evidence.SHA256(manifest.SHA256), Bytes: evidence.Uint64String(manifest.Bytes),
		Entries: evidence.Uint64String(manifest.Entries), NotExist: evidence.Uint64String(manifest.NotExist), TotalBytes: evidence.Uint64String(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: recordedCapturedInputLimits(manifest.Limits),
	}
}

func deterministicCapturedInputs(manifest evidence.ReadOnlyMounts) deterministicio.CapturedInputsManifest {
	return deterministicio.CapturedInputsManifest{
		Schema: manifest.Schema, File: manifest.File, SHA256: deterministicio.Digest(manifest.SHA256), Bytes: uint64(manifest.Bytes),
		Entries: uint64(manifest.Entries), NotExist: uint64(manifest.NotExist), TotalBytes: uint64(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: deterministicCapturedInputLimits(manifest.Limits),
	}
}

func recordedCapturedInputLimits(limits deterministicio.CapturedInputLimits) evidence.ReadOnlyMountLimits {
	return evidence.ReadOnlyMountLimits{
		PathBytes: evidence.Uint64String(limits.PathBytes), Requests: evidence.Uint64String(limits.Requests), Files: evidence.Uint64String(limits.Files),
		DirectoryEntries: evidence.Uint64String(limits.DirectoryEntries), SingleFileBytes: evidence.Uint64String(limits.SingleFileBytes), TotalBytes: evidence.Uint64String(limits.TotalBytes),
	}
}

func deterministicCapturedInputLimits(limits evidence.ReadOnlyMountLimits) deterministicio.CapturedInputLimits {
	return deterministicio.CapturedInputLimits{
		PathBytes: uint64(limits.PathBytes), Requests: uint64(limits.Requests), Files: uint64(limits.Files),
		DirectoryEntries: uint64(limits.DirectoryEntries), SingleFileBytes: uint64(limits.SingleFileBytes), TotalBytes: uint64(limits.TotalBytes),
	}
}
