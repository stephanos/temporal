//go:build gomad3_toolchain

package gomad3sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type filesystemHandleCase struct {
	name      string
	operation func(string) error
}

func TestFilesystemHandleOperationParity(t *testing.T) {
	for index, test := range filesystemHandleCases() {
		t.Run("standalone/"+test.name, func(t *testing.T) { require.NoError(t, test.operation(t.TempDir())) })
		t.Run("in-process/"+test.name, func(t *testing.T) { runFilesystemHandleCase(t, BackendInProcess, index) })
	}
}

func TestProcessFilesystemHandlePartialIOAndLifetime(t *testing.T) {
	runProcessFilesystemHandleCase(t, 0)
}
func TestProcessFilesystemHandleDirectoryAndChdir(t *testing.T) { runProcessFilesystemHandleCase(t, 1) }
func TestProcessFilesystemHandleAccessAndBounds(t *testing.T)   { runProcessFilesystemHandleCase(t, 2) }

func TestProcessFilesystemHandleReplayRejectsWriteBeforeMutation(t *testing.T) {
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	boot := uniqueBootID("filesystem-handle-replay-rejection")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error {
		file, err := os.OpenFile("/data/file", os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte("payload")); err != nil {
			info, statErr := file.Stat()
			if statErr != nil || info.Size() != 0 {
				return fmt.Errorf("rejected write mutated file: %v,%v", info, statErr)
			}
			return errors.New("filesystem handle replay write rejected before mutation")
		}
		return file.Close()
	}))
	spec := oneNodeVolumeSpec(boot)
	spec.Backend, spec.Fidelity = BackendProcess, FidelityHardIsolation
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			handle, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			terminal, err := cluster.Wait(ctx, handle)
			if err != nil {
				return err
			}
			if spec.Replay == nil && terminal.State != NodeStateExited {
				return fmt.Errorf("recording terminal=%s: %s", terminal.State, terminal.Reason)
			}
			if spec.Replay != nil && (terminal.State != NodeStateFailed || terminal.Reason != "filesystem handle replay write rejected before mutation") {
				return fmt.Errorf("rejection terminal=%s: %s", terminal.State, terminal.Reason)
			}
			return nil
		})
		require.NoError(t, err)
		return result
	}
	first := run(spec)
	require.Equal(t, OutcomeCompleted, first.Outcome, first.Reason)
	plan, err := ReplayPlanFor(first.Record)
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
	for index := range plan.Transitions {
		if plan.Transitions[index].Action == LifecycleWait {
			plan.Transitions[index].To = NodeStateFailed
		}
	}
	for index := range plan.Nodes {
		plan.Nodes[index].State, plan.Nodes[index].Reason = NodeStateFailed, "filesystem handle replay write rejected before mutation"
	}
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, OutcomeReplayDiverged, second.Outcome, second.Reason)
	require.NotNil(t, second.Divergence)
	require.Equal(t, ReplayDimensionVolume, second.Divergence.Dimension)
	require.NotNil(t, second.Divergence.ActualVolume)
	require.Equal(t, VolumeOperationWrite, second.Divergence.ActualVolume.Kind)
}

func runProcessFilesystemHandleCase(t *testing.T, index int) {
	t.Helper()
	if !processBackendAvailable() {
		t.Skip("Runner simulation transport is unavailable")
	}
	runFilesystemHandleCase(t, BackendProcess, index)
}

func runFilesystemHandleCase(t *testing.T, backend Backend, index int) {
	t.Helper()
	test := filesystemHandleCases()[index]
	boot := uniqueBootID("filesystem-handles-" + string(backend) + "-" + test.name)
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error {
		base := "/tmp/filesystem-handles-" + test.name
		if err := os.Mkdir(base, 0700); err != nil {
			return err
		}
		return errors.Join(test.operation(base), os.RemoveAll(base))
	}))
	spec := oneNodeVolumeSpec(boot)
	spec.Backend = backend
	if backend == BackendProcess {
		spec.Fidelity = FidelityHardIsolation
	}
	run := func(spec Spec) Result {
		result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
			handle, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			terminal, err := cluster.Wait(ctx, handle)
			if err != nil {
				return err
			}
			if terminal.State != NodeStateExited {
				return fmt.Errorf("filesystem boot state=%s: %s", terminal.State, terminal.Reason)
			}
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
		return result
	}
	first := run(spec)
	plan, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	spec.Replay = &plan
	second := run(spec)
	require.Equal(t, first.Record.Identity, second.Record.Identity)
}

func filesystemHandleCases() []filesystemHandleCase {
	return []filesystemHandleCase{
		{"partial-io-lifetime", func(base string) error {
			path := filepath.Join(base, "file")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return err
			}
			defer file.Close()
			if n, err := file.Write([]byte("abcd")); n != 4 || err != nil {
				return fmt.Errorf("write=%d,%v", n, err)
			}
			if n, err := file.Seek(1, io.SeekStart); n != 1 || err != nil {
				return fmt.Errorf("seek=%d,%v", n, err)
			}
			buffer := make([]byte, 3)
			if n, err := file.ReadAt(buffer, 2); n != 2 || err != io.EOF || !bytes.Equal(buffer, []byte{'c', 'd', 0}) {
				return fmt.Errorf("partial read=%d,%v,%q", n, err, buffer)
			}
			if n, err := file.Read(buffer[:1]); n != 1 || err != nil || buffer[0] != 'b' {
				return fmt.Errorf("offset read=%d,%v,%q", n, err, buffer[:1])
			}
			if n, err := file.WriteAt([]byte("Z"), 0); n != 1 || err != nil {
				return fmt.Errorf("write-at=%d,%v", n, err)
			}
			if n, err := file.Seek(0, io.SeekCurrent); n != 2 || err != nil {
				return fmt.Errorf("write-at offset=%d,%v", n, err)
			}
			if err := file.Truncate(2); err != nil {
				return err
			}
			if err := file.Chmod(0640); err != nil {
				return err
			}
			stamp := time.Unix(123, 0)
			if err := os.Chtimes(path, stamp, stamp); err != nil {
				return err
			}
			info, err := file.Stat()
			if err != nil || info.Size() != 2 || info.Mode().Perm() != 0640 || !info.ModTime().Equal(stamp) {
				return fmt.Errorf("metadata=%v,%v", info, err)
			}
			appender, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				return err
			}
			if n, err := appender.WriteAt([]byte("bad"), 0); n != 0 || err == nil {
				return errors.Join(fmt.Errorf("append write-at=%d,%v", n, err), appender.Close())
			}
			if n, err := appender.Write([]byte("!")); n != 1 || err != nil {
				return errors.Join(fmt.Errorf("append write=%d,%v", n, err), appender.Close())
			}
			if err := appender.Close(); err != nil {
				return err
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			if n, err := file.ReadAt(buffer, 0); n != 3 || err != nil || string(buffer) != "Zb!" {
				return fmt.Errorf("unlinked read=%d,%v,%q", n, err, buffer)
			}
			if err := file.Sync(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
			for _, operation := range []func() error{file.Close, func() error { _, err := file.Read(buffer); return err }, func() error { _, err := file.Stat(); return err }} {
				if err := operation(); !errors.Is(err, os.ErrClosed) {
					return fmt.Errorf("closed identity=%v", err)
				}
			}
			return nil
		}},
		{"directory-chdir", func(base string) error {
			for _, name := range []string{"z", "a", "m"} {
				if err := os.WriteFile(filepath.Join(base, name), []byte(name), 0600); err != nil {
					return err
				}
			}
			directory, err := os.Open(base)
			if err != nil {
				return err
			}
			defer directory.Close()
			var names []string
			for range 3 {
				entries, err := directory.ReadDir(1)
				if err != nil || len(entries) != 1 {
					return fmt.Errorf("incremental directory=%v,%v", entries, err)
				}
				names = append(names, entries[0].Name())
			}
			if !slices.Equal(names, []string{"a", "m", "z"}) {
				return fmt.Errorf("directory order=%v", names)
			}
			if entries, err := directory.ReadDir(1); len(entries) != 0 || err != io.EOF {
				return fmt.Errorf("directory eof=%v,%v", entries, err)
			}
			previous, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := directory.Chdir(); err != nil {
				return err
			}
			contents, readErr := os.ReadFile("m")
			restoreErr := os.Chdir(previous)
			if readErr != nil || restoreErr != nil || string(contents) != "m" {
				return errors.Join(fmt.Errorf("relative read=%q,%v", contents, readErr), restoreErr)
			}
			return nil
		}},
		{"access-bounds", func(base string) error {
			path := filepath.Join(base, "file")
			if err := os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
				return err
			}
			reader, err := os.Open(path)
			if err != nil {
				return err
			}
			defer reader.Close()
			if n, err := reader.Write([]byte("bad")); n != 0 || !errors.Is(err, syscall.EBADF) || errors.Is(err, os.ErrClosed) {
				return fmt.Errorf("access error=%d,%v", n, err)
			}
			writer, err := os.OpenFile(path, os.O_RDWR, 0)
			if err != nil {
				return err
			}
			defer writer.Close()
			if err := writer.Truncate(-1); !errors.Is(err, syscall.EINVAL) {
				return fmt.Errorf("negative truncate=%v", err)
			}
			if n, err := writer.WriteAt([]byte("bad"), -1); n != 0 || err == nil {
				return fmt.Errorf("negative write-at=%d,%v", n, err)
			}
			contents, err := os.ReadFile(path)
			if err != nil || string(contents) != "unchanged" {
				return fmt.Errorf("validation mutated data=%q,%v", contents, err)
			}
			return nil
		}},
	}
}
