//go:build unix

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestRunInspectReportsWhetherTheTargetIsShared(t *testing.T) {
	owner := t.TempDir()
	for _, test := range []struct {
		name  string
		store artifact.Store
		want  string
	}{
		{name: "store with a target pool", store: artifact.Store{Root: filepath.Join(owner, "pooled"), TargetPool: artifact.TargetPool(owner)}, want: "shared"},
		{name: "store without one", store: artifact.Store{Root: filepath.Join(owner, "private")}, want: "private"},
	} {
		t.Run(test.name, func(t *testing.T) {
			published, err := artifact.PublishArtifact(test.store, inspectArtifactInput(t))
			if err != nil {
				t.Fatal(err)
			}
			for format, output := range map[string]struct {
				arguments []string
				want      string
			}{
				"text": {arguments: []string{published.Path}, want: ` tags=[] sharing=` + test.want + "\n"},
				"json": {arguments: []string{"--json", published.Path}, want: `"sharing":"` + test.want + `"`},
			} {
				var stdout, stderr bytes.Buffer
				if status := runInspect(output.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
					t.Fatalf("%s: status = %d, stdout = %q, stderr = %q", format, status, stdout.String(), stderr.String())
				}
				if !strings.Contains(stdout.String(), output.want) {
					t.Fatalf("%s inspect output = %q, missing %q", format, stdout.String(), output.want)
				}
			}
		})
	}
}

func inspectArtifactInput(t *testing.T) artifact.ArtifactInput {
	t.Helper()
	targetPath := filepath.Join(t.TempDir(), "target")
	targetBytes := []byte("target bytes")
	if err := os.WriteFile(targetPath, targetBytes, 0o700); err != nil {
		t.Fatal(err)
	}
	world, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(2)
	profile := deterministicio.Default()
	return artifact.ArtifactInput{
		Manifest: record.ExecutionRecord{
			SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-08-10T12:00:00Z", CampaignID: "batch-1", Seed: 7, ReplayMode: record.ReplayExact,
			Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "darwin", HostArch: "arm64"},
			Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("c", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
			Target: record.Target{
				Kind: "go-run", Source: ".", SHA256: record.HashBytes(targetBytes), Size: record.Uint64String(len(targetBytes)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
				Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, CapabilityMode: "closure",
			},
			IOProfile:   record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())},
			Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name()}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
			Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
			World:       world,
			Outcome:     record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
			Streams:     record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
			Host:        record.Host{StartedAt: "2026-08-10T12:00:00Z", FinishedAt: "2026-08-10T12:00:01Z", ElapsedNanos: 1},
		},
		TargetPath: targetPath, World: worldPayloads,
	}
}
