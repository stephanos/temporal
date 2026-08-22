package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/artifact"
	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/record"
)

func campaignArtifactInput(t *testing.T) artifact.ArtifactInput {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	worldRecord, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	stdout := []byte("stdout")
	stderr := []byte("stderr")
	profile := deterministicio.Default()
	return artifact.ArtifactInput{
		Manifest: record.ExecutionRecord{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "batch-1", SelectionOrdinal: 0, Seed: 7, ReplayMode: record.ReplayExact,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: record.Target{
				Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomadv3-target"}, BuildTags: []string{},
				Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure",
			},
			IOProfile:   record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())},
			Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "GOMADV3_IO_PROFILE", Value: profile.Name()}, {Name: "TZ", Value: "UTC"}},
			Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
			World:       worldRecord,
			Outcome:     record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
			Streams: record.Streams{
				Stdout: record.Stream{FullSHA256: record.HashBytes(stdout), TotalBytes: record.Uint64String(len(stdout)), RetainedBytes: record.Uint64String(len(stdout))},
				Stderr: record.Stream{FullSHA256: record.HashBytes(stderr), TotalBytes: record.Uint64String(len(stderr)), RetainedBytes: record.Uint64String(len(stderr))},
			},
			Host: record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: 1},
		},
		TargetPath: targetPath, Stdout: stdout, Stderr: stderr, World: worldPayloads,
	}
}
