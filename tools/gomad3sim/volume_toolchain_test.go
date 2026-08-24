//go:build gomad3_toolchain

package gomad3sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeCrashEnumerationRestartAndExactReplay(t *testing.T) {
	bootID := uniqueBootID("volume-crash-restart")
	ready := make(chan NodeContext, 4)
	recovered := make(chan []byte, 4)
	release := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			if err := os.MkdirAll("/data", 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile("/data/value", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			if _, err := file.Write([]byte("stable")); err != nil {
				return errors.Join(err, file.Close())
			}
			if err := file.Sync(); err != nil {
				return errors.Join(err, file.Close())
			}
			directory, err := os.Open("/data")
			if err != nil {
				return errors.Join(err, file.Close())
			}
			if err := directory.Sync(); err != nil {
				return errors.Join(err, directory.Close(), file.Close())
			}
			if err := directory.Close(); err != nil {
				return errors.Join(err, file.Close())
			}
			if _, err := file.Write([]byte("-volatile")); err != nil {
				return errors.Join(err, file.Close())
			}
			if err := file.Close(); err != nil {
				return err
			}
			ready <- node
			<-release
			return nil
		}
		file, err := os.Open("/data/value")
		if err != nil {
			return err
		}
		contents, err := io.ReadAll(file)
		if err != nil {
			return errors.Join(err, file.Close())
		}
		if err := file.Close(); err != nil {
			return err
		}
		recovered <- contents
		ready <- node
		<-ctx.Done()
		return ctx.Err()
	}))
	spec := oneNodeVolumeSpec(bootID)
	scenario := func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		limits := VolumeCrashEnumerationLimits{States: 1, Operations: 32, Depth: 32, Bytes: 1 << 20, WallNanos: 1_000_000_000}
		var frontier *VolumeCrashFrontier
		var states []VolumeCrashState
		for {
			page, err := cluster.EnumerateCrashStates(ctx, handle, "data", limits, frontier)
			if err != nil {
				return err
			}
			states = append(states, page.States...)
			if page.Complete {
				break
			}
			if page.Capacity != VolumeCrashCapacityStates || page.Frontier == nil {
				return errors.New("volume crash enumeration did not return a resumable state frontier")
			}
			frontier = page.Frontier
		}
		if len(states) < 2 {
			return errors.New("volume crash enumeration omitted partial persistence states")
		}
		foundStable := false
		foundVolatile := false
		for _, state := range states {
			contents := volumeCrashFile(state, "/value")
			if !bytes.HasPrefix(contents, []byte("stable")) {
				return errors.New("volume crash state lost acknowledged durable bytes")
			}
			foundStable = foundStable || bytes.Equal(contents, []byte("stable"))
			foundVolatile = foundVolatile || bytes.Equal(contents, []byte("stable-volatile"))
		}
		if !foundStable || !foundVolatile {
			return errors.New("volume crash enumeration omitted endpoint states")
		}
		if err := cluster.Crash(ctx, handle); err != nil {
			return err
		}
		restarted, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		contents := <-recovered
		if !bytes.HasPrefix(contents, []byte("stable")) {
			return errors.New("restart lost acknowledged durable bytes")
		}
		return cluster.Stop(ctx, restarted)
	}

	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
	require.NotEmpty(t, recorded.Volumes.Transitions)
	require.NotEmpty(t, recorded.Volumes.Snapshot.Identity)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &plan
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome, replayed.Reason)
	require.Equal(t, recorded.Volumes, replayed.Volumes)
	close(release)
}

func TestProcessBackendPreservesHostVolumeAcrossRestart(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	bootID := uniqueBootID("volume-process-restart")
	require.NoError(t, RegisterBoot(bootID, func(_ context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			if err := os.MkdirAll("/data", 0o755); err != nil {
				return err
			}
			return os.WriteFile("/data/value", []byte("persisted"), 0o600)
		}
		contents, err := os.ReadFile("/data/value")
		if err != nil {
			return err
		}
		if !bytes.Equal(contents, []byte("persisted")) {
			return fmt.Errorf("restarted volume contents = %q", contents)
		}
		return nil
	}))
	spec := oneNodeVolumeSpec(bootID)
	spec.Backend = BackendProcess
	spec.Fidelity = FidelityHardIsolation
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		terminal, err := cluster.Wait(ctx, first)
		if err != nil {
			return err
		}
		if terminal.State != NodeStateExited {
			return fmt.Errorf("first incarnation state = %s: %s", terminal.State, terminal.Reason)
		}
		second, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		terminal, err = cluster.Wait(ctx, second)
		if err != nil {
			return err
		}
		if terminal.State != NodeStateExited {
			return fmt.Errorf("second incarnation state = %s: %s", terminal.State, terminal.Reason)
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.NotEmpty(t, result.Volumes.Transitions)
}

func TestVolumeReplayRejectsChangedWriteBeforeFileMutation(t *testing.T) {
	bootID := uniqueBootID("volume-replay-before-mutation")
	rejectedSize := make(chan int64, 1)
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error {
		file, err := os.OpenFile("/data/value", os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte("payload")); err != nil {
			info, statErr := file.Stat()
			if statErr == nil {
				rejectedSize <- info.Size()
			}
			return errors.Join(err, statErr, file.Close())
		}
		return file.Close()
	}))
	spec := oneNodeVolumeSpec(bootID)
	scenario := func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		_, err = cluster.Wait(ctx, handle)
		return err
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome, recorded.Reason)
	plan, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	changed := false
	for index := range plan.Volumes.Transitions {
		if plan.Volumes.Transitions[index].Kind == VolumeOperationWrite {
			plan.Volumes.Transitions[index].Bytes++
			changed = true
			break
		}
	}
	require.True(t, changed)
	plan.Volumes.Snapshot.TransitionSHA256 = volumeTransitionsIdentity(plan.Volumes.Transitions)
	plan.Volumes.Snapshot.Identity = volumeRunSnapshotIdentity(plan.Volumes.Snapshot)
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	replayed, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, replayed.Outcome)
	require.NotNil(t, replayed.Divergence)
	require.Equal(t, ReplayDimensionVolume, replayed.Divergence.Dimension)
	require.Equal(t, int64(0), <-rejectedSize)
}

func TestVolumeGracefulStopFlushesUnsyncedWrites(t *testing.T) {
	bootID := uniqueBootID("volume-graceful-flush")
	ready := make(chan struct{}, 2)
	recovered := make(chan []byte, 1)
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			if err := os.WriteFile("/data/value", []byte("unsynced"), 0o600); err != nil {
				return err
			}
			ready <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		}
		contents, err := os.ReadFile("/data/value")
		if err != nil {
			return err
		}
		recovered <- contents
		ready <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}))
	result, err := Run(context.Background(), oneNodeVolumeSpec(bootID), func(ctx context.Context, cluster Cluster) error {
		first, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		if err := cluster.Stop(ctx, first); err != nil {
			return err
		}
		second, err := cluster.Restart(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		if !bytes.Equal(<-recovered, []byte("unsynced")) {
			return errors.New("graceful stop did not flush unsynced volume bytes")
		}
		return cluster.Stop(ctx, second)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
}

func TestVolumeFileAndDirectorySyncParity(t *testing.T) {
	bootID := uniqueBootID("volume-file-directory-sync")
	ready := make(chan struct{}, 2)
	continueBoot := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		file, err := os.OpenFile("/data/value", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte("value")); err != nil {
			return errors.Join(err, file.Close())
		}
		if err := file.Sync(); err != nil {
			return errors.Join(err, file.Close())
		}
		ready <- struct{}{}
		<-continueBoot
		directory, err := os.Open("/data")
		if err != nil {
			return errors.Join(err, file.Close())
		}
		if err := directory.Sync(); err != nil {
			return errors.Join(err, directory.Close(), file.Close())
		}
		if err := directory.Close(); err != nil {
			return errors.Join(err, file.Close())
		}
		if err := file.Close(); err != nil {
			return err
		}
		ready <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}))
	result, err := Run(context.Background(), oneNodeVolumeSpec(bootID), func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		states, err := enumerateVolumeCrashStates(ctx, cluster, handle)
		if err != nil {
			return err
		}
		if err := requireVolumeCrashProjections(states, []string{"", "/value=value\x00"}); err != nil {
			return err
		}
		close(continueBoot)
		<-ready
		states, err = enumerateVolumeCrashStates(ctx, cluster, handle)
		if err != nil {
			return err
		}
		if err := requireVolumeCrashProjections(states, []string{"/value=value\x00"}); err != nil {
			return err
		}
		return cluster.Stop(ctx, handle)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
}

func TestVolumeRenameAndTruncateCrashDependenciesParity(t *testing.T) {
	bootID := uniqueBootID("volume-rename-truncate")
	ready := make(chan struct{}, 2)
	continueBoot := make(chan struct{})
	require.NoError(t, RegisterBoot(bootID, func(ctx context.Context, _ NodeContext) error {
		file, err := os.OpenFile("/data/a", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte("abcd")); err != nil {
			return errors.Join(err, file.Close())
		}
		if err := file.Sync(); err != nil {
			return errors.Join(err, file.Close())
		}
		directory, err := os.Open("/data")
		if err != nil {
			return errors.Join(err, file.Close())
		}
		if err := directory.Sync(); err != nil {
			return errors.Join(err, directory.Close(), file.Close())
		}
		if err := os.Rename("/data/a", "/data/b"); err != nil {
			return errors.Join(err, directory.Close(), file.Close())
		}
		ready <- struct{}{}
		<-continueBoot
		if err := directory.Sync(); err != nil {
			return errors.Join(err, directory.Close(), file.Close())
		}
		if err := file.Truncate(2); err != nil {
			return errors.Join(err, directory.Close(), file.Close())
		}
		ready <- struct{}{}
		<-ctx.Done()
		return errors.Join(ctx.Err(), directory.Close(), file.Close())
	}))
	result, err := Run(context.Background(), oneNodeVolumeSpec(bootID), func(ctx context.Context, cluster Cluster) error {
		handle, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		states, err := enumerateVolumeCrashStates(ctx, cluster, handle)
		if err != nil {
			return err
		}
		if err := requireVolumeCrashProjections(states, []string{"/a=abcd\x00", "/b=abcd\x00"}); err != nil {
			return err
		}
		close(continueBoot)
		<-ready
		states, err = enumerateVolumeCrashStates(ctx, cluster, handle)
		if err != nil {
			return err
		}
		if err := requireVolumeCrashProjections(states, []string{"/b=ab\x00", "/b=abcd\x00"}); err != nil {
			return err
		}
		return cluster.Stop(ctx, handle)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
}

func oneNodeVolumeSpec(boot BootID) Spec {
	return Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 41, Limits: DefaultLimits(),
		Nodes:   []NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1", Volumes: []VolumeMount{{Volume: "data", Path: "/data"}}}},
		Volumes: []VolumeSpec{{ID: "data", CapacityBytes: 1 << 20}},
	}
}

func volumeCrashFile(state VolumeCrashState, path string) []byte {
	for _, entry := range state.Entries {
		if entry.Path == path && entry.Kind == "file" {
			return entry.Data
		}
	}
	return nil
}

func enumerateVolumeCrashStates(ctx context.Context, cluster Cluster, handle NodeHandle) ([]VolumeCrashState, error) {
	page, err := cluster.EnumerateCrashStates(ctx, handle, "data", VolumeCrashEnumerationLimits{
		States: 16, Operations: 64, Depth: 64, Bytes: 1 << 20, WallNanos: 1_000_000_000,
	}, nil)
	if err != nil {
		return nil, err
	}
	if !page.Complete || page.Frontier != nil || page.Capacity != "" {
		return nil, errors.New("volume crash enumeration unexpectedly stopped before completion")
	}
	return page.States, nil
}

func requireVolumeCrashProjections(states []VolumeCrashState, expected []string) error {
	actual := make([]string, 0, len(states))
	for _, state := range states {
		projection := ""
		for _, entry := range state.Entries {
			if entry.Kind == "file" {
				projection += entry.Path + "=" + string(entry.Data) + "\x00"
			}
		}
		actual = append(actual, projection)
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("volume crash projections = %q, want %q", actual, expected)
	}
	return nil
}
