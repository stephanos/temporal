package measure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestStressPublicRunValidatesInputsInsideOwnedStage(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "stress output")
	var log bytes.Buffer
	err := Run(t.Context(), "stress", Options{Repo: t.TempDir(), Log: &log, Inputs: map[string]string{
		"GOMAD_WASM_STRESS_ROOT": root, "GOMAD_WASM_STRESS_HOST_DESCRIPTION": "host-only admission regression",
		"GOMAD_WASM_TEST_HELPER": "/unused-helper", "GOMAD_STOCK_GO": "/unused-compiler",
		"GOMAD_WASM_STRESS_REPETITIONS": "invalid",
	}})
	require.ErrorIs(t, err, ErrRefused)
	require.ErrorContains(t, err, "--stress-repetitions must be 1..256")
	require.Contains(t, log.String(), "staged output=")
	runs, err := filepath.Glob(filepath.Join(root, ".gomad-measure", "stress", "run-*"))
	require.NoError(t, err)
	require.Len(t, runs, 1)
}

func TestStressPublishedRerunUsesRequestedRoot(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module stress-retention-fixture\n"), 0600))
	for _, args := range [][]string{{"init", "-q", repo}, {"-C", repo, "add", "go.mod"}} {
		output, err := exec.CommandContext(t.Context(), "git", args...).CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	root := filepath.Join(t.TempDir(), "stress output")
	err := Run(t.Context(), "stress", Options{Repo: repo, Inputs: map[string]string{
		"GOMAD_WASM_STRESS_ROOT": root, "GOMAD_WASM_STRESS_HOST_DESCRIPTION": "host-only admission regression",
		"GOMAD_WASM_TEST_HELPER": "/unused-helper", "GOMAD_STOCK_GO": "/unused-compiler",
		"GOMAD_WASM_STRESS_ONLY": "runtime", "GOMAD_WASM_STRESS_SEED": "1",
	}})
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(retentionLatestFor(t, root, "stress"), "report.json"))
	require.NoError(t, err)
	var report struct {
		Complete bool
		Rerun    []string
	}
	require.NoError(t, json.Unmarshal(data, &report))
	require.False(t, report.Complete, "an excluded seed must not claim a completed matrix")
	index := slices.Index(report.Rerun, "--stress-root")
	require.GreaterOrEqual(t, index, 0)
	require.Less(t, index+1, len(report.Rerun))
	require.Equal(t, root, report.Rerun[index+1])
}

func TestStressExecutionSharesParentStageLeaseAndDefersParentCleanup(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "stress")
	scope := &runScope{runResources: &runResources{}, name: "stress", options: Options{
		Repo: t.TempDir(), Log: io.Discard, Inputs: map[string]string{"GOMAD_WASM_STRESS_ROOT": base},
	}}
	cleanupErr := errors.New("parent cleanup result")
	cleanups := 0
	var stage string
	err := scope.execute(context.WithValue(t.Context(), runKey{}, scope), func(ctx context.Context) error {
		stage = input(ctx, "GOMAD_WASM_STRESS_ROOT")
		addCleanup(ctx, func(context.Context) error { cleanups++; return cleanupErr })
		request := wasi.Request{Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{Mode: choice.ModeReplay}}}
		_, err := stressExecute(ctx, stressCase{Workload: "runtime"}, request, "source", "reference", 0, stage, nil)
		require.ErrorContains(t, err, "stress requires fresh seeded execution")
		require.Zero(t, cleanups, "child ran the parent's cleanup stack")
		file, err := os.OpenFile(filepath.Join(stage, ".lease"), os.O_RDWR, 0)
		require.NoError(t, err)
		_, err = hostfs.TryDescriptor(file)
		require.ErrorIs(t, err, hostfs.ErrContended, "child released the parent's active lease")
		require.NoError(t, file.Close())
		return nil
	})
	require.ErrorIs(t, err, cleanupErr)
	require.Equal(t, 1, cleanups)
	file, err := os.OpenFile(filepath.Join(stage, ".lease"), os.O_RDWR, 0)
	require.NoError(t, err)
	lease, err := hostfs.TryDescriptor(file)
	require.NoError(t, err, "parent did not release its stage lease")
	require.NoError(t, lease.Release())
}

func TestStressExecutionRerunPreservesRequestedRoot(t *testing.T) {
	t.Parallel()
	repo, base := t.TempDir(), filepath.Join(t.TempDir(), "stress output")
	compiler := filepath.Join(repo, "unused-compiler")
	require.NoError(t, os.WriteFile(compiler, []byte("never executed"), 0600))
	scope := &runScope{runResources: &runResources{}, name: "stress", options: Options{
		Repo: repo, Log: io.Discard, Inputs: map[string]string{
			"GOMAD_WASM_STRESS_ROOT": base, "GOMAD_WASM_STRESS_HOST_DESCRIPTION": "host-only rerun regression",
			"GOMAD_WASM_TEST_HELPER": "/unused-helper", "GOMAD_STOCK_GO": compiler,
		},
	}}
	err := scope.execute(context.WithValue(t.Context(), runKey{}, scope), func(ctx context.Context) error {
		stage := input(ctx, "GOMAD_WASM_STRESS_ROOT")
		command := rerunRecord(ctx, "go", repo, "stress", scope.options.Inputs, false)["command"].([]string)
		request := wasi.Request{CompileCache: filepath.Join(stage, "compile-cache"), Engine: wasi.PinnedEngine(),
			Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{Mode: choice.ModeRecord}}}
		observation, err := stressExecute(ctx, stressCase{Workload: "two-cluster", Seed: 7}, request, "source", "reference", 0, filepath.Join(stage, "two-cluster-7"), command)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.ErrorContains(t, err, "temporal_storage/recovery_test.go")
		index := slices.Index(observation.Run.Command, "--stress-root")
		require.GreaterOrEqual(t, index, 0)
		require.Less(t, index+1, len(observation.Run.Command))
		require.Equal(t, base, observation.Run.Command[index+1])
		require.Contains(t, observation.Run.Command, "--stress-only")
		require.Contains(t, observation.Run.Command, "two-cluster")
		return nil
	})
	require.NoError(t, err)
}
