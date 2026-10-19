package measure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
)

func TestHarnessRetentionPublishedRerunUsesRequestedRoot(t *testing.T) {
	t.Parallel()
	for _, producer := range []struct {
		name, root string
		shell      bool
	}{
		{"guest-profile", "GOMAD_WASM_GUEST_PROFILE_ROOT", true},
		{"guest-variants", "GOMAD_WASM_GUEST_PROFILE_ROOT", true},
		{"diagnostic-cluster-baseline", "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", true},
		{"matched-release-candidate", "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT", false},
	} {
		t.Run(producer.name, func(t *testing.T) {
			t.Parallel()
			base := filepath.Join(t.TempDir(), "output root")
			fixture := t.TempDir()
			recorder, arguments := filepath.Join(fixture, "record-arguments"), filepath.Join(fixture, "arguments")
			require.NoError(t, os.WriteFile(recorder, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$RERUN_ARGUMENTS\"\n"), 0700))
			current := base
			for range 3 {
				scope := &runScope{runResources: &runResources{}, name: producer.name, options: Options{Repo: fixture, Log: io.Discard, Inputs: map[string]string{producer.root: current}}}
				ctx := context.WithValue(t.Context(), runKey{}, scope)
				require.NoError(t, scope.execute(ctx, func(ctx context.Context) error {
					root := input(ctx, producer.root)
					record := rerunRecord(ctx, recorder, fixture, producer.name, map[string]string{producer.root: root}, producer.shell)
					return temporalJSON(ctx, filepath.Join(root, "report.json"), map[string]any{"rerun": record})
				}))
				path := retentionLatestFor(t, current, producer.name)
				data, err := os.ReadFile(filepath.Join(path, "report.json"))
				require.NoError(t, err)
				var report struct {
					Rerun struct{ Command json.RawMessage }
				}
				require.NoError(t, json.Unmarshal(data, &report))
				var command *exec.Cmd
				if producer.shell {
					var shell string
					require.NoError(t, json.Unmarshal(report.Rerun.Command, &shell))
					command = exec.CommandContext(t.Context(), "/bin/sh", "-c", shell)
				} else {
					var argv []string
					require.NoError(t, json.Unmarshal(report.Rerun.Command, &argv))
					command = exec.CommandContext(t.Context(), argv[0], argv[1:]...)
				}
				command.Env = append(os.Environ(), "RERUN_ARGUMENTS="+arguments)
				output, err := command.CombinedOutput()
				require.NoError(t, err, "%s", output)
				argv, err := os.ReadFile(arguments)
				require.NoError(t, err)
				args := strings.Split(strings.TrimSuffix(string(argv), "\n"), "\n")
				index := slices.Index(args, "--"+measurement.Flag(producer.root))
				require.GreaterOrEqual(t, index, 0)
				require.Less(t, index+1, len(args))
				current = args[index+1]
				t.Logf("published report %s: %s", path, data)
				require.Equal(t, base, current, "rerun nests retention below the immutable result")
				entries, err := filepath.Glob(filepath.Join(base, ".gomad-measure", producer.name, "run-*"))
				require.NoError(t, err)
				require.Len(t, entries, 1)
			}
		})
	}
}

func retentionRun(t *testing.T, base string, producer func(context.Context) error) error {
	t.Helper()
	scope := retentionScope(t, base)
	return scope.execute(context.WithValue(t.Context(), runKey{}, scope), producer)
}

func retentionScope(t *testing.T, base string) *runScope {
	t.Helper()
	return &runScope{runResources: &runResources{}, name: "runtime-cost-profile", options: Options{
		Repo: t.TempDir(), Log: io.Discard, Inputs: map[string]string{"GOMAD_WASM_PROFILE_ROOT": base},
	}}
}

func retentionLatest(t *testing.T, base string) string {
	t.Helper()
	return retentionLatestFor(t, base, "runtime-cost-profile")
}

func retentionLatestFor(t *testing.T, base, producer string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, ".gomad-measure", producer, "latest.json"))
	require.NoError(t, err)
	var run string
	require.NoError(t, json.Unmarshal(data, &run))
	require.Equal(t, filepath.Base(run), run)
	return filepath.Join(base, ".gomad-measure", producer, run)
}

func TestHarnessRetentionRealIdentityProducer(t *testing.T) {
	t.Parallel()
	base, repo := t.TempDir(), t.TempDir()
	for range 3 {
		require.NoError(t, Run(t.Context(), "compiled-identity-owners", Options{Repo: repo, Inputs: map[string]string{"GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT": base}, Log: io.Discard}))
		path := retentionLatestFor(t, base, "compiled-identity-owners")
		data, err := os.ReadFile(filepath.Join(path, "compiled-owner-identities.json"))
		require.NoError(t, err)
		require.True(t, json.Valid(data))
		entries, err := filepath.Glob(filepath.Join(base, ".gomad-measure", "compiled-identity-owners", "run-*"))
		require.NoError(t, err)
		require.Len(t, entries, 1)
	}
}

func TestHarnessRetentionReplacesCompleteRuns(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	unrelated := filepath.Join(base, "historical-evidence")
	require.NoError(t, os.WriteFile(unrelated, []byte("preserve"), 0600))
	var prior string
	for attempt := range 6 {
		want := fmt.Sprint(attempt)
		require.NoError(t, retentionRun(t, base, func(ctx context.Context) error {
			root := input(ctx, "GOMAD_WASM_PROFILE_ROOT")
			if prior != "" {
				require.Equal(t, prior, retentionLatest(t, base), "incomplete attempt became latest")
			}
			if err := os.WriteFile(filepath.Join(root, "result.json"), []byte(want), 0600); err != nil {
				return err
			}
			if err := os.Symlink(unrelated, filepath.Join(root, "external-link")); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(root, "module.wasm"), make([]byte, 4096), 0600)
		}))
		latest := retentionLatest(t, base)
		got, err := os.ReadFile(filepath.Join(latest, "result.json"))
		require.NoError(t, err)
		require.Equal(t, want, string(got))
		if prior != "" {
			_, err := os.Stat(prior)
			require.ErrorIs(t, err, os.ErrNotExist)
		}
		prior = latest
		entries, err := filepath.Glob(filepath.Join(base, ".gomad-measure", "runtime-cost-profile", "run-*"))
		require.NoError(t, err)
		require.Len(t, entries, 1, "superseded modules accumulate")
		var bytes int64
		require.NoError(t, filepath.WalkDir(base, func(_ string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err == nil && info.Mode().IsRegular() {
				bytes += info.Size()
			}
			return err
		}))
		require.LessOrEqual(t, bytes, int64(8192), "staging or superseded bytes accumulate")
	}
	got, err := os.ReadFile(unrelated)
	require.NoError(t, err)
	require.Equal(t, "preserve", string(got))
}

func TestHarnessRetentionFailurePreservesLatest(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.NoError(t, retentionRun(t, base, func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
	}))
	prior := retentionLatest(t, base)
	failure := errors.New("incomplete measurement")
	for range 4 {
		err := retentionRun(t, base, func(ctx context.Context) error {
			require.NoError(t, os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "diagnostic"), []byte("failed"), 0600))
			return failure
		})
		require.ErrorIs(t, err, failure)
		require.Equal(t, prior, retentionLatest(t, base))
		entries, err := filepath.Glob(filepath.Join(base, ".gomad-measure", "runtime-cost-profile", "run-*"))
		require.NoError(t, err)
		require.Len(t, entries, 2, "failed diagnostics accumulate")
	}
	got, err := os.ReadFile(filepath.Join(prior, "result"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(got))
}

func TestHarnessRetentionRetainedConsumerPinsInputsUntilCollected(t *testing.T) {
	t.Parallel()
	source, consumer := t.TempDir(), t.TempDir()
	produce := func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "module.wasm"), []byte("module"), 0600)
	}
	require.NoError(t, retentionRun(t, source, produce))
	original := retentionLatest(t, source)
	scope := retentionScope(t, consumer)
	scope.options.Inputs["GOMAD_WASM_TEST_HELPER"] = filepath.Join(original, "module.wasm")
	require.NoError(t, scope.execute(context.WithValue(t.Context(), runKey{}, scope), produce))
	for range 3 {
		require.NoError(t, retentionRun(t, source, produce))
		got, err := os.ReadFile(filepath.Join(original, "module.wasm"))
		require.NoError(t, err, "retained consumer lost its authenticated input")
		require.Equal(t, "module", string(got))
	}
	require.NoError(t, retentionRun(t, consumer, produce))
	_, err := os.Stat(original)
	require.ErrorIs(t, err, os.ErrNotExist, "collected consumer left its input pinned")
}

func TestHarnessRetentionTransitiveInputsAndUnrelatedRuns(t *testing.T) {
	t.Parallel()
	source, middle, consumer := t.TempDir(), t.TempDir(), t.TempDir()
	produce := func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "module.wasm"), []byte("module"), 0600)
	}
	pin := func(base, dependency string) {
		scope := retentionScope(t, base)
		scope.options.Inputs["GOMAD_WASM_TEST_HELPER"] = filepath.Join(dependency, "module.wasm")
		require.NoError(t, scope.execute(context.WithValue(t.Context(), runKey{}, scope), produce))
	}
	require.NoError(t, retentionRun(t, source, produce))
	original := retentionLatest(t, source)
	unrelated := filepath.Join(filepath.Dir(original), "run-historical")
	require.NoError(t, os.Mkdir(unrelated, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(unrelated, "evidence"), []byte("historical"), 0600))
	abandoned := filepath.Join(filepath.Dir(original), ".stage-run-interrupted")
	require.NoError(t, os.Mkdir(abandoned, 0700))
	require.NoError(t, os.Symlink(unrelated, filepath.Join(abandoned, "external-link")))
	partialPointer := filepath.Join(filepath.Dir(original), ".stage-interrupted-pointer")
	require.NoError(t, os.WriteFile(partialPointer, []byte("partial pointer"), 0600))
	pin(middle, original)
	intermediate := retentionLatest(t, middle)
	pin(consumer, intermediate)
	require.NoError(t, retentionRun(t, source, produce))
	_, err := os.Stat(abandoned)
	require.ErrorIs(t, err, os.ErrNotExist, "interrupted initialization was not reclaimed")
	_, err = os.Stat(partialPointer)
	require.ErrorIs(t, err, os.ErrNotExist, "interrupted pointer staging was not reclaimed")
	require.NoError(t, retentionRun(t, middle, produce))
	_, err = os.Stat(filepath.Join(original, "module.wasm"))
	require.NoError(t, err, "transitive input was collected")
	require.NoError(t, retentionRun(t, consumer, produce))
	for _, path := range []string{original, intermediate} {
		_, err := os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist, "transitive pin was not released")
	}
	got, err := os.ReadFile(filepath.Join(unrelated, "evidence"))
	require.NoError(t, err)
	require.Equal(t, "historical", string(got))
}

func TestHarnessRetentionOverlappingRunsAndExplicitKeep(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	produce := func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
	}
	keptScope := retentionScope(t, base)
	keptScope.options.Keep = true
	require.NoError(t, keptScope.execute(context.WithValue(t.Context(), runKey{}, keptScope), produce))
	kept := retentionLatest(t, base)
	active, finish, result := make(chan string, 1), make(chan struct{}), make(chan error, 1)
	scope := retentionScope(t, base)
	ctx := context.WithValue(t.Context(), runKey{}, scope)
	go func() {
		result <- scope.execute(ctx, func(ctx context.Context) error {
			active <- input(ctx, "GOMAD_WASM_PROFILE_ROOT")
			select {
			case <-finish:
				return produce(ctx)
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	var running string
	select {
	case running = <-active:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	require.NoError(t, retentionRun(t, base, produce))
	_, err := os.Stat(running)
	require.NoError(t, err, "overlapping publication collected the active run")
	close(finish)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	_, err = os.Stat(kept)
	require.NoError(t, err)
	require.NotEqual(t, kept, retentionLatest(t, base))
	require.NoError(t, os.Remove(filepath.Join(kept, ".keep")))
	require.NoError(t, retentionRun(t, base, produce))
	_, err = os.Stat(kept)
	require.ErrorIs(t, err, os.ErrNotExist, "released keep was not collected")
}

func TestHarnessRetentionAllowsCanonicalAncestorAliases(t *testing.T) {
	t.Parallel()
	parent, real := t.TempDir(), t.TempDir()
	alias := filepath.Join(parent, "var")
	require.NoError(t, os.Symlink(real, alias))
	base := filepath.Join(alias, "new-output")
	for range 3 {
		require.NoError(t, retentionRun(t, base, func(ctx context.Context) error {
			return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
		}))
	}
	latest := retentionLatest(t, base)
	got, err := os.ReadFile(filepath.Join(latest, "result"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(got))
}

func TestHarnessRetentionRejectsRedirectedInputLease(t *testing.T) {
	t.Parallel()
	source, consumer := t.TempDir(), t.TempDir()
	require.NoError(t, retentionRun(t, source, func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "module.wasm"), []byte("module"), 0600)
	}))
	original := retentionLatest(t, source)
	outside := filepath.Join(t.TempDir(), "evidence")
	require.NoError(t, os.WriteFile(outside, []byte("preserve"), 0600))
	require.NoError(t, os.Remove(filepath.Join(original, ".lease")))
	require.NoError(t, os.Symlink(outside, filepath.Join(original, ".lease")))
	scope := retentionScope(t, consumer)
	scope.options.Inputs["GOMAD_WASM_TEST_HELPER"] = filepath.Join(original, "module.wasm")
	err := scope.execute(context.WithValue(t.Context(), runKey{}, scope), func(context.Context) error {
		t.Error("producer ran without a valid input lease")
		return nil
	})
	require.ErrorIs(t, err, ErrRefused)
	got, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, "preserve", string(got))
}

func TestHarnessRetentionPublicationFailureLeavesPreviousResult(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	produce := func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
	}
	require.NoError(t, retentionRun(t, base, produce))
	prior := retentionLatest(t, base)
	pointer := filepath.Join(filepath.Dir(prior), "latest.json")
	linked := filepath.Join(t.TempDir(), "linked-pointer")
	require.NoError(t, os.Link(pointer, linked))
	require.Error(t, retentionRun(t, base, produce), "unsafe publication pointer must fail closed")
	require.Equal(t, prior, retentionLatest(t, base))
	got, err := os.ReadFile(filepath.Join(prior, "result"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(got))
	require.NoError(t, os.Remove(linked))
	require.NoError(t, retentionRun(t, base, produce), "next invocation must recover")
}

func TestHarnessRetentionCleanupAndCancellationPrecedePublication(t *testing.T) {
	t.Parallel()
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			require.NoError(t, retentionRun(t, base, func(context.Context) error { return nil }))
			prior := retentionLatest(t, base)
			scope := retentionScope(t, base)
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), runKey{}, scope))
			defer cancel()
			cleanupFailure := errors.New("final report failed")
			err := scope.execute(ctx, func(ctx context.Context) error {
				addCleanup(ctx, func(context.Context) error {
					require.Equal(t, prior, retentionLatest(t, base))
					if canceled {
						cancel()
						return nil
					}
					return cleanupFailure
				})
				return nil
			})
			if canceled {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, cleanupFailure)
			}
			require.Equal(t, prior, retentionLatest(t, base))
		})
	}
}

func TestHarnessRetentionInterruptedRunRecovery(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.NoError(t, retentionRun(t, base, func(context.Context) error { return nil }))
	prior := retentionLatest(t, base)
	for range 3 {
		command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestHarnessRetentionInterruptedChild$")
		command.Env = append(os.Environ(), "GOMAD_RETENTION_CRASH_ROOT="+base)
		output, err := command.CombinedOutput()
		exit, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok, "%s: %v", output, err)
		require.Equal(t, 23, exit.ExitCode(), "%s", output)
		require.Equal(t, prior, retentionLatest(t, base))
	}
	require.NoError(t, retentionRun(t, base, func(context.Context) error { return nil }))
	entries, err := filepath.Glob(filepath.Join(base, ".gomad-measure", "runtime-cost-profile", "run-*"))
	require.NoError(t, err)
	require.Len(t, entries, 2, "abandoned attempts must converge to one diagnostic")
}

func TestHarnessRetentionInterruptedChild(t *testing.T) {
	base := os.Getenv("GOMAD_RETENTION_CRASH_ROOT")
	if base == "" {
		t.Skip("child-process fixture")
	}
	scope := &runScope{runResources: &runResources{}, name: "runtime-cost-profile", options: Options{Repo: base, Log: io.Discard, Inputs: map[string]string{"GOMAD_WASM_PROFILE_ROOT": base}}}
	err := scope.execute(context.WithValue(t.Context(), runKey{}, scope), func(ctx context.Context) error {
		if err := os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "partial"), []byte("interrupted"), 0600); err != nil {
			return err
		}
		os.Exit(23)
		return nil
	})
	t.Fatalf("child did not interrupt: %v", err)
}

func TestHarnessRetentionRejectsSymlinkRoots(t *testing.T) {
	t.Parallel()
	for _, component := range []string{"output", ".gomad-measure", "runtime-cost-profile"} {
		t.Run(component, func(t *testing.T) {
			t.Parallel()
			base, outside := t.TempDir(), t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(outside, "evidence"), []byte("preserve"), 0600))
			switch component {
			case "output":
				base = filepath.Join(base, "redirect")
				require.NoError(t, os.Symlink(outside, base))
			case ".gomad-measure":
				require.NoError(t, os.Symlink(outside, filepath.Join(base, component)))
			default:
				require.NoError(t, os.Mkdir(filepath.Join(base, ".gomad-measure"), 0700))
				require.NoError(t, os.Symlink(outside, filepath.Join(base, ".gomad-measure", component)))
			}
			err := retentionRun(t, base, func(context.Context) error {
				t.Error("producer reached redirected output root")
				return nil
			})
			require.Error(t, err)
			entries, err := os.ReadDir(outside)
			require.NoError(t, err)
			require.Len(t, entries, 1)
		})
	}
}

func TestHarnessRetentionPublicationWaitsForStoreLock(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	produce := func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
	}
	require.NoError(t, retentionRun(t, base, produce))
	prior := retentionLatest(t, base)
	store, err := retentionDirectory(filepath.Dir(prior), false)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	running, locked, result := make(chan string, 1), make(chan struct{}), make(chan error, 1)
	go func() {
		result <- retentionRun(t, base, func(ctx context.Context) error {
			running <- input(ctx, "GOMAD_WASM_PROFILE_ROOT")
			<-locked
			return produce(ctx)
		})
	}()
	var run string
	select {
	case run = <-running:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	held, err := retentionStoreLock(t.Context(), store)
	close(locked)
	require.NoError(t, err)
	// Tree sync and the completion marker happen under the run's own lease, before the store lock.
	poll, deadline := time.NewTicker(time.Millisecond), time.After(10*time.Second)
	defer poll.Stop()
	for {
		if _, err := os.Stat(filepath.Join(run, ".complete")); err == nil {
			break
		}
		select {
		case <-poll.C:
		case <-deadline:
			t.Fatal("run was not completed before taking the store lock")
		}
	}
	select {
	case err := <-result:
		t.Fatalf("publication finished while the store lock was held: %v", err)
	default:
	}
	require.Equal(t, prior, retentionLatest(t, base))
	require.NoError(t, held.Release())
	select {
	case err := <-result:
		require.NoError(t, err, "contended publication lost a complete result")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	require.Equal(t, run, retentionLatest(t, base))
	_, err = os.Stat(prior)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestHarnessRetentionReclaimFailureKeepsPublishedResultAndRecovers(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	base := t.TempDir()
	locked := ""
	produce := func(ctx context.Context) error {
		root := input(ctx, "GOMAD_WASM_PROFILE_ROOT")
		if locked == "" {
			locked = filepath.Join(root, "cache")
			if err := os.Mkdir(locked, 0700); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(locked, "module.cwasm"), []byte("compiled"), 0600); err != nil {
				return err
			}
		}
		return os.WriteFile(filepath.Join(root, "result"), []byte("complete"), 0600)
	}
	require.NoError(t, retentionRun(t, base, produce))
	prior := retentionLatest(t, base)
	require.NoError(t, os.Chmod(locked, 0500))
	t.Cleanup(func() {
		trash, _ := filepath.Glob(filepath.Join(filepath.Dir(prior), ".trash-*", "cache"))
		for _, path := range append(trash, locked) {
			_ = os.Chmod(path, 0700)
		}
	})
	require.NoError(t, retentionRun(t, base, produce), "a published result must not fail on deferred reclaim")
	latest := retentionLatest(t, base)
	require.NotEqual(t, prior, latest)
	_, err := os.Stat(prior)
	require.ErrorIs(t, err, os.ErrNotExist, "partially reclaimed run kept its authenticated path")
	trash, err := filepath.Glob(filepath.Join(filepath.Dir(prior), ".trash-*"))
	require.NoError(t, err)
	require.Len(t, trash, 1)
	require.NoError(t, os.Chmod(filepath.Join(trash[0], "cache"), 0700))
	require.NoError(t, retentionRun(t, base, produce))
	trash, err = filepath.Glob(filepath.Join(filepath.Dir(prior), ".trash-*"))
	require.NoError(t, err)
	require.Empty(t, trash, "partially deleted run was not reclaimed")
}

func TestHarnessRetentionRefusesOutputRootInsideStore(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	require.NoError(t, retentionRun(t, base, func(ctx context.Context) error {
		return os.WriteFile(filepath.Join(input(ctx, "GOMAD_WASM_PROFILE_ROOT"), "result"), []byte("complete"), 0600)
	}))
	latest := retentionLatest(t, base)
	for _, nested := range []string{latest, filepath.Join(latest, "nested"), filepath.Dir(latest)} {
		err := retentionRun(t, nested, func(context.Context) error {
			t.Error("producer nested inside a retained run")
			return nil
		})
		require.ErrorIs(t, err, ErrRefused)
	}
	_, err := os.Stat(filepath.Join(latest, ".gomad-measure"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(latest, "nested"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestHarnessRetentionRerunRecordsUseRequestedRoot(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "run.go" {
			continue
		}
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		require.NotContains(t, string(data), "measurement.RerunRecord(", "%s publishes a rerun of the staged run path; use rerunRecord", file)
	}
}
