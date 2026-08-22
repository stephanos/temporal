package runner

import (
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomadv3/record"
)

func executionAdapters(adapters []deterministicio.BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func deterministicAdapters(adapters []record.TargetAdapter) []deterministicio.Adapter {
	result := make([]deterministicio.Adapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = deterministicio.Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func recordedIOProfile(profile deterministicio.Spec) record.IOProfile {
	return record.IOProfile{
		Name:                 profile.Name(),
		ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()),
		Inventory:            string(profile.Inventory()),
		InventorySHA256:      record.SHA256(profile.InventorySHA256()),
	}
}

func recordedCapturedInputs(manifest readonlymount.CapturedInputsManifest) record.ReadOnlyMounts {
	return record.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: record.SHA256(manifest.SHA256), Bytes: record.Uint64String(manifest.Bytes),
		Entries: record.Uint64String(manifest.Entries), NotExist: record.Uint64String(manifest.NotExist), TotalBytes: record.Uint64String(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: recordedCapturedInputLimits(manifest.Limits),
	}
}

func deterministicCapturedInputs(manifest record.ReadOnlyMounts) readonlymount.CapturedInputsManifest {
	return readonlymount.CapturedInputsManifest{
		Schema: manifest.Schema, File: manifest.File, SHA256: manifest.SHA256, Bytes: uint64(manifest.Bytes),
		Entries: uint64(manifest.Entries), NotExist: uint64(manifest.NotExist), TotalBytes: uint64(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: deterministicCapturedInputLimits(manifest.Limits),
	}
}

func recordedCapturedInputLimits(limits readonlymount.CapturedInputLimits) record.ReadOnlyMountLimits {
	return record.ReadOnlyMountLimits{
		PathBytes: record.Uint64String(limits.PathBytes), Requests: record.Uint64String(limits.Requests), Files: record.Uint64String(limits.Files),
		DirectoryEntries: record.Uint64String(limits.DirectoryEntries), SingleFileBytes: record.Uint64String(limits.SingleFileBytes), TotalBytes: record.Uint64String(limits.TotalBytes),
	}
}

func deterministicCapturedInputLimits(limits record.ReadOnlyMountLimits) readonlymount.CapturedInputLimits {
	return readonlymount.CapturedInputLimits{
		PathBytes: uint64(limits.PathBytes), Requests: uint64(limits.Requests), Files: uint64(limits.Files),
		DirectoryEntries: uint64(limits.DirectoryEntries), SingleFileBytes: uint64(limits.SingleFileBytes), TotalBytes: uint64(limits.TotalBytes),
	}
}
