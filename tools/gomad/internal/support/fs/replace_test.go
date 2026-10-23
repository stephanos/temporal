package hostfs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestStagingAndLinkCountUseSharedHostOwner(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "result")
	staged, err := StageContext(context.Background(), directory, ".result-*", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "published")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := staged.Cleanup(); err != nil {
			t.Errorf("Cleanup() = %v", err)
		}
	}()
	published, err := staged.PublishNoReplaceContext(context.Background(), destination)
	if err != nil || !published {
		t.Fatalf("PublishNoReplaceContext() = %t, %v", published, err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if links, known := LinkCount(info); known && links != 1 {
		t.Fatalf("LinkCount() = %d, %t", links, known)
	}
}

type faultStagedFile struct {
	file       *os.File
	chmodErr   error
	writeErr   error
	syncErr    error
	closeErr   error
	closeCalls int
}

func (file *faultStagedFile) Name() string { return file.file.Name() }
func (file *faultStagedFile) Chmod(mode os.FileMode) error {
	if file.chmodErr != nil {
		return file.chmodErr
	}
	return file.file.Chmod(mode)
}
func (file *faultStagedFile) Write(contents []byte) (int, error) {
	if file.writeErr != nil {
		return 0, file.writeErr
	}
	return file.file.Write(contents)
}
func (file *faultStagedFile) Sync() error {
	if file.syncErr != nil {
		return file.syncErr
	}
	return file.file.Sync()
}
func (file *faultStagedFile) Close() error {
	file.closeCalls++
	err := file.file.Close()
	return errors.Join(file.closeErr, err)
}

func TestStagedFilePublishesValidatedBytesAndReportsDisposition(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "result.json")
	staged, err := StageContext(context.Background(), directory, ".result-*.partial", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "result\n")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(staged.Path()) != directory {
		t.Fatalf("staged path = %q, want same directory %q", staged.Path(), directory)
	}
	if contents, err := os.ReadFile(staged.Path()); err != nil || string(contents) != "result\n" {
		t.Fatalf("staged bytes = %q, %v", contents, err)
	}
	published, err := staged.PublishContext(context.Background(), destination)
	if err != nil || !published {
		t.Fatalf("PublishContext() = %t, %v", published, err)
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != "result\n" {
		t.Fatalf("published bytes = %q, %v", contents, err)
	}
	if err := staged.Cleanup(); err != nil {
		t.Fatalf("Cleanup() after publication = %v", err)
	}
}

func TestStageContextPreservesOperationAndCleanupErrors(t *testing.T) {
	sentinel := errors.New("injected publication failure")
	cleanup := errors.New("injected cleanup failure")
	for _, test := range []struct {
		name       string
		configure  func(*faultStagedFile, *publicationOperations)
		fill       func(context.Context, io.Writer) error
		wantSecond error
	}{
		{name: "create", configure: func(_ *faultStagedFile, operations *publicationOperations) {
			operations.createTemp = func(string, string) (stagedHandle, error) { return nil, sentinel }
		}},
		{name: "chmod", configure: func(file *faultStagedFile, _ *publicationOperations) { file.chmodErr = sentinel }},
		{name: "fill", fill: func(context.Context, io.Writer) error { return sentinel }},
		{name: "write", configure: func(file *faultStagedFile, _ *publicationOperations) { file.writeErr = sentinel }},
		{name: "sync", configure: func(file *faultStagedFile, _ *publicationOperations) { file.syncErr = sentinel }},
		{name: "close", configure: func(file *faultStagedFile, _ *publicationOperations) { file.closeErr = sentinel }},
		{name: "primary-close-join", configure: func(file *faultStagedFile, _ *publicationOperations) { file.closeErr = cleanup }, fill: func(context.Context, io.Writer) error { return sentinel }, wantSecond: cleanup},
		{name: "primary-cleanup-join", configure: func(_ *faultStagedFile, operations *publicationOperations) {
			operations.remove = func(string) error { return cleanup }
		}, fill: func(context.Context, io.Writer) error { return sentinel }, wantSecond: cleanup},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			operations := defaultPublicationOperations
			var created *faultStagedFile
			if test.name == "create" || test.name == "primary-cleanup-join" {
				test.configure(nil, &operations)
			} else {
				createTemp := operations.createTemp
				operations.createTemp = func(directory, pattern string) (stagedHandle, error) {
					file, err := createTemp(directory, pattern)
					if err != nil {
						return nil, err
					}
					created = &faultStagedFile{file: file.(*os.File)}
					if test.configure != nil {
						test.configure(created, &operations)
					}
					return created, nil
				}
			}
			fill := test.fill
			if fill == nil {
				fill = func(_ context.Context, writer io.Writer) error {
					_, err := io.WriteString(writer, "bytes")
					return err
				}
			}
			staged, err := stageContext(context.Background(), directory, ".fault-*", 0o600, fill, operations)
			if staged != nil || !errors.Is(err, sentinel) {
				t.Fatalf("stageContext() = %#v, %v", staged, err)
			}
			if test.wantSecond != nil && !errors.Is(err, test.wantSecond) {
				t.Fatalf("stageContext() error = %v, want joined %v", err, test.wantSecond)
			}
		})
	}
}

func TestStagedFileReportsPreAndPostRenameFailures(t *testing.T) {
	renameFailure := errors.New("rename failed")
	directoryFailure := errors.New("directory sync failed")
	for _, test := range []struct {
		name          string
		failure       error
		configure     func(*publicationOperations)
		wantPublished bool
		wantOld       bool
	}{
		{name: "rename", failure: renameFailure, wantOld: true, configure: func(operations *publicationOperations) {
			operations.rename = func(string, string) error { return renameFailure }
		}},
		{name: "directory", failure: directoryFailure, wantPublished: true, configure: func(operations *publicationOperations) {
			operations.syncDirectory = func(string) error { return directoryFailure }
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			destination := filepath.Join(directory, "result")
			if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			operations := defaultPublicationOperations
			test.configure(&operations)
			staged, err := stageContext(context.Background(), directory, ".stage-*", 0o600, func(_ context.Context, writer io.Writer) error {
				_, err := io.WriteString(writer, "new")
				return err
			}, operations)
			if err != nil {
				t.Fatal(err)
			}
			published, err := staged.PublishContext(context.Background(), destination)
			if published != test.wantPublished || !errors.Is(err, test.failure) {
				t.Fatalf("PublishContext() = %t, %v", published, err)
			}
			contents, readErr := os.ReadFile(destination)
			if readErr != nil {
				t.Fatal(readErr)
			}
			want := "new"
			if test.wantOld {
				want = "old"
			}
			if string(contents) != want {
				t.Fatalf("destination = %q, want %q", contents, want)
			}
			if err := staged.Cleanup(); err != nil {
				t.Fatal(err)
			}
			if err := staged.Cleanup(); err != nil {
				t.Fatalf("second Cleanup() = %v", err)
			}
		})
	}
}

func TestStagedFileCancellationPreservesDestinationAndStage(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "result")
	if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	staged, err := StageContext(context.Background(), directory, ".stage-*", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "new")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	stagedPath := staged.Path()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if published, err := staged.PublishContext(ctx, destination); published || !errors.Is(err, context.Canceled) {
		t.Fatalf("PublishContext(cancelled) = %t, %v", published, err)
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != "old" {
		t.Fatalf("destination = %q, %v", contents, err)
	}
	if _, err := os.Stat(stagedPath); err != nil {
		t.Fatalf("cancelled publication removed stage: %v", err)
	}
	if err := staged.Cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestStagedFileNoReplacePublicationPreservesImmutableDestination(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "result")
	if err := os.WriteFile(destination, []byte("winner"), 0o600); err != nil {
		t.Fatal(err)
	}
	staged, err := StageContext(context.Background(), directory, ".stage-*", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "candidate")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := staged.PublishNoReplaceContext(context.Background(), destination)
	if published || !errors.Is(err, os.ErrExist) {
		t.Fatalf("PublishNoReplaceContext() = %t, %v", published, err)
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != "winner" {
		t.Fatalf("destination = %q, %v", contents, err)
	}
	if err := staged.Cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestStagedFileCleanupAndSyncFlushesRemovalAndReportsSyncFailure(t *testing.T) {
	directory := t.TempDir()
	syncFailure := errors.New("directory sync failed")
	operations := defaultPublicationOperations
	removed := false
	remove := operations.remove
	operations.remove = func(path string) error {
		if err := remove(path); err != nil {
			return err
		}
		removed = true
		return nil
	}
	operations.syncDirectory = func(path string) error {
		if path != directory || !removed {
			t.Fatalf("syncDirectory(%q) before removal", path)
		}
		return syncFailure
	}
	staged, err := stageContext(context.Background(), directory, ".stage-*", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "candidate")
		return err
	}, operations)
	if err != nil {
		t.Fatal(err)
	}
	path := staged.Path()
	if err := staged.CleanupAndSync(); !errors.Is(err, syncFailure) {
		t.Fatalf("CleanupAndSync() error = %v", err)
	}
	if staged.Path() != "" {
		t.Fatalf("CleanupAndSync() retained path %q", staged.Path())
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CleanupAndSync() retained stage: %v", err)
	}
	if err := staged.CleanupAndSync(); err != nil {
		t.Fatalf("second CleanupAndSync() = %v", err)
	}
}

func TestReplacePublishesFileWithRequestedMode(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "generated")
	path := filepath.Join(directory, "artifact.go")
	if err := Replace(path, []byte("generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "generated\n" || info.Mode().Perm() != 0o644 || len(entries) != 1 {
		t.Fatalf("contents = %q, mode = %o, entries = %d", contents, info.Mode().Perm(), len(entries))
	}
}

func TestReplaceContextRejectsCancellationBeforePublication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ReplaceContext(ctx, path, []byte("replacement\n"), 0o600); !errors.Is(err, context.Canceled) {
		t.Fatalf("ReplaceContext() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published cancelled replacement: %v", err)
	}
}
