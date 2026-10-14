package runner

import (
	backend "go.temporal.io/server/tools/gomad/internal/backends"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func recordedIOProfile(profile backend.Description) record.IOProfile {
	return record.IOProfile{
		Name:                 profile.Name(),
		ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()),
		Inventory:            string(profile.Inventory()),
		InventorySHA256:      record.SHA256(profile.InventorySHA256()),
	}
}

func recordedCapturedInputs(manifest captured.CapturedInputsManifest) record.ReadOnlyMounts {
	return record.ReadOnlyMounts{
		Schema: manifest.Schema, File: manifest.File, SHA256: record.SHA256(manifest.SHA256), Bytes: record.Uint64String(manifest.Bytes),
		Entries: record.Uint64String(manifest.Entries), NotExist: record.Uint64String(manifest.NotExist), TotalBytes: record.Uint64String(manifest.TotalBytes),
		Mappings: append([]string(nil), manifest.Mappings...), Limits: recordedCapturedInputLimits(manifest.Limits),
	}
}

func recordedCapturedInputLimits(limits captured.CapturedInputLimits) record.ReadOnlyMountLimits {
	return record.ReadOnlyMountLimits{
		PathBytes: record.Uint64String(limits.PathBytes), Requests: record.Uint64String(limits.Requests), Files: record.Uint64String(limits.Files),
		DirectoryEntries: record.Uint64String(limits.DirectoryEntries), SingleFileBytes: record.Uint64String(limits.SingleFileBytes), TotalBytes: record.Uint64String(limits.TotalBytes),
	}
}

func deterministicCapturedInputLimits(limits record.ReadOnlyMountLimits) captured.CapturedInputLimits {
	return captured.CapturedInputLimits{
		PathBytes: uint64(limits.PathBytes), Requests: uint64(limits.Requests), Files: uint64(limits.Files),
		DirectoryEntries: uint64(limits.DirectoryEntries), SingleFileBytes: uint64(limits.SingleFileBytes), TotalBytes: uint64(limits.TotalBytes),
	}
}
