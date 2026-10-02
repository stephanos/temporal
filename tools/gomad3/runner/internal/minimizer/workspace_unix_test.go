//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package minimizer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

const workspaceLockHolderEnvironment = "GOMAD3_MINIMIZER_WORKSPACE_LOCK_HOLDER"

func workspaceBinding() Binding {
	return Binding{
		ParentRecordHash: record.HashBytes([]byte("parent record")), ImplementationSHA256: ImplementationSHA256(), ToolchainBuildKey: "build-key",
	}
}

func otherWorkspaceBinding() Binding {
	binding := workspaceBinding()
	binding.ParentRecordHash = record.HashBytes([]byte("other parent record"))
	return binding
}

// parentStateDirectory is where root keeps the state of binding's parent.
func parentStateDirectory(root string, binding Binding) string {
	return filepath.Join(root, workspaceStateRoot, parentDirectory(binding.ParentRecordHash))
}

func workspaceInitialState(t *testing.T, attemptBudget uint64) State {
	t.Helper()
	config := testConfig()
	state, err := New(config, testCandidate(t, config,
		testForced(t, simulationengine.DimensionRuntime, 0),
		testForced(t, simulationengine.DimensionFault, 0),
	), attemptBudget)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func openTestWorkspace(t *testing.T, root string, resume bool) *Workspace {
	t.Helper()
	workspace, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), resume)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := workspace.Close(); err != nil {
			t.Error(err)
		}
	})
	return workspace
}

// commitAttempt commits the next attempt and, when it is accepted, retains a
// stand-in artifact directory for it first.
func commitAttempt(t *testing.T, workspace *Workspace, accepted bool) {
	t.Helper()
	state := workspace.Checkpoint().State
	attempt, ok, err := Next(state)
	if err != nil || !ok {
		t.Fatalf("Next() = %#v, %t, %v", attempt, ok, err)
	}
	next, err := Commit(state, attempt, accepted)
	if err != nil {
		t.Fatal(err)
	}
	var reference *AcceptedArtifact
	if accepted {
		reference = &AcceptedArtifact{
			Directory: "sha256-" + strings.TrimPrefix(string(attempt.Candidate.SHA256), "sha256:")[:32], RecordHash: attempt.Candidate.SHA256, ChoiceReplayStatus: "exact",
		}
		if err := os.Mkdir(filepath.Join(workspace.AcceptedRoot(), reference.Directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspace.Commit(next, reference); err != nil {
		t.Fatal(err)
	}
}

// interruptedWorkspace returns a root holding one rejected and one accepted attempt.
func interruptedWorkspace(t *testing.T) (string, Checkpoint) {
	t.Helper()
	root := t.TempDir()
	workspace := openTestWorkspace(t, root, false)
	commitAttempt(t, workspace, false)
	commitAttempt(t, workspace, true)
	checkpoint := workspace.Checkpoint()
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	return root, checkpoint
}

func TestWorkspaceResumeRestoresThePersistedCheckpoint(t *testing.T) {
	root, persisted := interruptedWorkspace(t)
	if persisted.State.Attempts != 2 || len(persisted.State.Accepted) != 1 || persisted.Accepted == nil {
		t.Fatalf("persisted checkpoint = %#v", persisted)
	}
	resumed := openTestWorkspace(t, root, true)
	if restored := resumed.Checkpoint(); !reflect.DeepEqual(restored, persisted) {
		t.Fatalf("restored checkpoint = %#v, want %#v", restored, persisted)
	}
	commitAttempt(t, resumed, true)
	accepted, err := os.ReadDir(resumed.AcceptedRoot())
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 1 || accepted[0].Name() != resumed.Checkpoint().Accepted.Directory {
		t.Fatalf("retained accepted artifacts = %v, want only %s", accepted, resumed.Checkpoint().Accepted.Directory)
	}
}

func TestWorkspaceResumeRejectsStateOfAnotherRun(t *testing.T) {
	root, _ := interruptedWorkspace(t)
	for _, test := range []struct {
		name   string
		change func(t *testing.T, binding *Binding, initial *State)
		want   string
	}{
		{name: "parent artifact", change: func(_ *testing.T, binding *Binding, _ *State) {
			*binding = otherWorkspaceBinding()
		}, want: ErrNoCheckpoint.Error()},
		{name: "minimizer implementation", change: func(_ *testing.T, binding *Binding, _ *State) {
			binding.ImplementationSHA256 = record.HashBytes([]byte("other implementation"))
		}, want: "different minimizer implementation"},
		{name: "toolchain build key", change: func(_ *testing.T, binding *Binding, _ *State) {
			binding.ToolchainBuildKey = "other-build-key"
		}, want: "different toolchain build key"},
		{name: "attempt budget", change: func(t *testing.T, _ *Binding, initial *State) {
			*initial = workspaceInitialState(t, 9)
		}, want: "attempt budget 8, not 9"},
		{name: "starting candidate", change: func(t *testing.T, _ *Binding, initial *State) {
			config := testConfig()
			other, err := New(config, testCandidate(t, config, testForced(t, simulationengine.DimensionRuntime, 0)), 8)
			if err != nil {
				t.Fatal(err)
			}
			*initial = other
		}, want: "does not start from the parent artifact's candidate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			binding, initial := workspaceBinding(), workspaceInitialState(t, 8)
			test.change(t, &binding, &initial)
			_, err := OpenWorkspace(context.Background(), root, binding, initial, true)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("OpenWorkspace() error = %v, want %q", err, test.want)
			}
		})
	}
	openTestWorkspace(t, root, true)
}

func TestWorkspaceResumeFailsClosedOnDamagedState(t *testing.T) {
	for _, test := range []struct {
		name   string
		damage func(t *testing.T, root string, persisted Checkpoint)
	}{
		{name: "missing accepted artifact", damage: func(t *testing.T, root string, persisted Checkpoint) {
			if err := os.Remove(filepath.Join(parentStateDirectory(root, workspaceBinding()), workspaceAcceptedName, persisted.Accepted.Directory)); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "edited state", damage: func(t *testing.T, root string, _ Checkpoint) {
			rewriteCheckpoint(t, root, func(encoded string) string {
				return strings.Replace(encoded, `"attempts":2`, `"attempts":1`, 1)
			})
		}},
		{name: "truncated state", damage: func(t *testing.T, root string, _ Checkpoint) {
			rewriteCheckpoint(t, root, func(encoded string) string { return encoded[:len(encoded)/2] })
		}},
		{name: "non-canonical state", damage: func(t *testing.T, root string, _ Checkpoint) {
			rewriteCheckpoint(t, root, func(encoded string) string { return encoded + "\n" })
		}},
		{name: "symbolic link state", damage: func(t *testing.T, root string, _ Checkpoint) {
			path := filepath.Join(parentStateDirectory(root, workspaceBinding()), workspaceCheckpointName)
			moved := path + ".moved"
			if err := os.Rename(path, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(moved, path); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, persisted := interruptedWorkspace(t)
			test.damage(t, root, persisted)
			if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), true); err == nil || errors.Is(err, ErrNoCheckpoint) {
				t.Fatalf("OpenWorkspace() error = %v", err)
			}
		})
	}
}

func TestWorkspaceKeepsStatePerParentArtifact(t *testing.T) {
	root, persisted := interruptedWorkspace(t)
	held := openTestWorkspace(t, root, true)
	initial := workspaceInitialState(t, 8)

	other, err := OpenWorkspace(context.Background(), root, otherWorkspaceBinding(), initial, false)
	if err != nil {
		t.Fatalf("OpenWorkspace() of another parent beside held state: %v", err)
	}
	if err := errors.Join(other.Complete(), other.Close()); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), initial, true); !errors.Is(err, hostfs.ErrContended) {
		t.Fatalf("second OpenWorkspace() of the held parent error = %v", err)
	}
	if err := held.Close(); err != nil {
		t.Fatal(err)
	}
	if restored := openTestWorkspace(t, root, true).Checkpoint(); !reflect.DeepEqual(restored, persisted) {
		t.Fatalf("checkpoint after another parent completed = %#v, want %#v", restored, persisted)
	}
}

func TestWorkspaceResumeRejectsStateMovedFromAnotherParent(t *testing.T) {
	root, _ := interruptedWorkspace(t)
	if err := os.Rename(parentStateDirectory(root, workspaceBinding()), parentStateDirectory(root, otherWorkspaceBinding())); err != nil {
		t.Fatal(err)
	}
	_, err := OpenWorkspace(context.Background(), root, otherWorkspaceBinding(), workspaceInitialState(t, 8), true)
	if want := "different parent artifact"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("OpenWorkspace() error = %v, want %q", err, want)
	}
}

func TestWorkspaceRejectsSymbolicLinkStateRoot(t *testing.T) {
	root, elsewhere := t.TempDir(), t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(root, workspaceStateRoot)); err != nil {
		t.Fatal(err)
	}
	for _, resume := range []bool{false, true} {
		if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), resume); err == nil || errors.Is(err, ErrNoCheckpoint) {
			t.Fatalf("OpenWorkspace(resume=%t) through a symbolic link error = %v", resume, err)
		}
	}
	if entries, err := os.ReadDir(elsewhere); err != nil || len(entries) != 0 {
		t.Fatalf("entries written through the symbolic link = %v, %v", entries, err)
	}
}

func rewriteCheckpoint(t *testing.T, root string, rewrite func(string) string) {
	t.Helper()
	path := filepath.Join(parentStateDirectory(root, workspaceBinding()), workspaceCheckpointName)
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := rewrite(string(encoded))
	if rewritten == string(encoded) {
		t.Fatal("checkpoint rewrite changed nothing")
	}
	if err := os.WriteFile(path, []byte(rewritten), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceCommitRequiresTheAcceptedArtifactOnDisk(t *testing.T) {
	root := t.TempDir()
	workspace := openTestWorkspace(t, root, false)
	before := workspace.Checkpoint()
	attempt, _, err := Next(before.State)
	if err != nil {
		t.Fatal(err)
	}
	next, err := Commit(before.State, attempt, true)
	if err != nil {
		t.Fatal(err)
	}
	reference := &AcceptedArtifact{Directory: "sha256-absent", RecordHash: attempt.Candidate.SHA256, ChoiceReplayStatus: "exact"}
	for name, accepted := range map[string]*AcceptedArtifact{"absent artifact": reference, "no artifact": nil} {
		if err := workspace.Commit(next, accepted); err == nil {
			t.Fatalf("Commit() with %s succeeded", name)
		}
	}
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	if persisted := openTestWorkspace(t, root, true).Checkpoint(); !reflect.DeepEqual(persisted, before) {
		t.Fatalf("persisted checkpoint = %#v, want %#v", persisted, before)
	}
}

func TestWorkspaceStateGatesInitialRunsAndResumes(t *testing.T) {
	root := t.TempDir()
	if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), true); !errors.Is(err, ErrNoCheckpoint) {
		t.Fatalf("resume without state error = %v", err)
	}
	workspace := openTestWorkspace(t, root, false)
	commitAttempt(t, workspace, true)
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), false); !errors.Is(err, ErrCheckpointExists) {
		t.Fatalf("initial run over existing state error = %v", err)
	}

	resumed := openTestWorkspace(t, root, true)
	if err := resumed.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(parentStateDirectory(root, workspaceBinding())); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state directory after completion: %v", err)
	}
	// A run that died between removing the checkpoint and its directory
	// leaves a directory the next initial run discards.
	if err := os.MkdirAll(filepath.Join(parentStateDirectory(root, workspaceBinding()), workspaceAcceptedName, "sha256-stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	restarted := openTestWorkspace(t, root, false)
	if checkpoint := restarted.Checkpoint(); checkpoint.State.Attempts != 0 || checkpoint.Accepted != nil {
		t.Fatalf("restarted checkpoint = %#v", checkpoint)
	}
	if stale, err := os.ReadDir(restarted.AcceptedRoot()); err != nil || len(stale) != 0 {
		t.Fatalf("accepted artifacts after restart = %v, %v", stale, err)
	}
}

func TestWorkspaceRecordsPublicationOnceTheStateStopped(t *testing.T) {
	root := t.TempDir()
	workspace := openTestWorkspace(t, root, false)
	published := PublishedArtifact{Directory: "sha256-published", RecordHash: record.HashBytes([]byte("published record"))}
	if err := os.Mkdir(filepath.Join(root, published.Directory), 0o700); err != nil {
		t.Fatal(err)
	}
	commitAttempt(t, workspace, true)
	if err := workspace.RecordPublication(published); err == nil {
		t.Fatal("RecordPublication() accepted a state that has not stopped")
	}
	for workspace.Checkpoint().State.StopReason == "" {
		commitAttempt(t, workspace, false)
	}
	if err := workspace.RecordPublication(published); err != nil {
		t.Fatal(err)
	}
	want := workspace.Checkpoint()
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	if restored := openTestWorkspace(t, root, true).Checkpoint(); restored.Published == nil || !reflect.DeepEqual(restored, want) {
		t.Fatalf("restored checkpoint = %#v, want %#v", restored, want)
	}
}

func TestWorkspaceLockOfKilledProcessDoesNotBlockResume(t *testing.T) {
	if root := os.Getenv(workspaceLockHolderEnvironment); root != "" {
		holdWorkspaceLockUntilKilled(t, root)
		return
	}
	root := t.TempDir()
	holder := exec.Command(os.Args[0], "-test.run=^TestWorkspaceLockOfKilledProcessDoesNotBlockResume$")
	holder.Env = append(os.Environ(), workspaceLockHolderEnvironment+"="+root)
	stdin, err := holder.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := holder.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	killed := false
	defer func() {
		if !killed {
			_ = holder.Process.Kill()
			_ = holder.Wait()
		}
	}()
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("lock holder reported %q, %v", line, err)
	}
	for _, resume := range []bool{false, true} {
		if _, err := OpenWorkspace(context.Background(), root, workspaceBinding(), workspaceInitialState(t, 8), resume); !errors.Is(err, hostfs.ErrContended) {
			t.Fatalf("OpenWorkspace(resume=%t) beside a live holder error = %v", resume, err)
		}
	}
	if err := holder.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	killed = true
	if err := holder.Wait(); err == nil {
		t.Fatal("lock holder exited cleanly instead of being killed")
	}
	if _, err := os.Lstat(parentStateDirectory(root, workspaceBinding()) + workspaceLockSuffix); err != nil {
		t.Fatalf("lock file of the killed holder: %v", err)
	}
	resumed := openTestWorkspace(t, root, true)
	if checkpoint := resumed.Checkpoint(); checkpoint.State.Attempts != 1 || checkpoint.Accepted == nil {
		t.Fatalf("checkpoint after the killed holder = %#v", checkpoint)
	}
}

func holdWorkspaceLockUntilKilled(t *testing.T, root string) {
	workspace := openTestWorkspace(t, root, false)
	commitAttempt(t, workspace, true)
	if _, err := os.Stdout.WriteString("locked\n"); err != nil {
		t.Fatal(err)
	}
	// The parent keeps stdin open until it has killed this process.
	_, _ = io.Copy(io.Discard, os.Stdin)
	t.Fatal("lock holder outlived its parent's kill")
}
