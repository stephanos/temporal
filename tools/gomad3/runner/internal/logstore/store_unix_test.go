//go:build unix

package logstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

// A umask that grants the owner nothing still yields a private store the
// writer pins: creation sets the directory's mode itself.
func TestWriterCreatesStoreUnderAFullUmask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store")
	previous := syscall.Umask(0o777)
	writer, _, err := OpenWriter(path, testOptions(true, nil), nil)
	syscall.Umask(previous)
	if err != nil {
		t.Fatal(err)
	}
	defer closeWriter(t, writer)
	for name, mode := range map[string]os.FileMode{".": storeMode, LogFile: memberMode, LockFile: memberMode} {
		info, err := os.Lstat(filepath.Join(path, name))
		if err != nil || info.Mode() != mode {
			t.Fatalf("%s mode = %v, %v; want %v", name, info.Mode(), err, mode)
		}
	}
	if err := writer.AppendJSON(kindEntry, makeEntry(0)); err != nil {
		t.Fatal(err)
	}
}

func TestWriterLifecycleCreatesPrivateStoreAndReplays(t *testing.T) {
	// A restrictive umask proves that creation sets private modes through the
	// new descriptors rather than relying on the process umask.
	path := filepath.Join(t.TempDir(), "store")
	previous := syscall.Umask(0o277)
	defer syscall.Umask(previous)
	if _, _, err := OpenWriter(path, testOptions(false, nil), nil); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenWriter without Create error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenWriter without Create created the store: %v", err)
	}
	writer, report, err := OpenWriter(path, testOptions(true, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report != (Report{}) {
		t.Fatalf("new store report = %#v", report)
	}
	for name, mode := range map[string]os.FileMode{".": storeMode, LogFile: memberMode, LockFile: memberMode} {
		info, err := os.Lstat(filepath.Join(path, name))
		if err != nil || info.Mode() != mode {
			t.Fatalf("%s mode = %v, %v; want %v", name, info.Mode(), err, mode)
		}
	}
	for sequence := range 3 {
		if err := writer.AppendJSON(kindEntry, makeEntry(sequence)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Append(kindMarker, []byte(`{"marker":"done"}`)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(7, []byte(`{}`)); err == nil {
		t.Fatal("unknown kind appended")
	}
	if err := writer.Append(kindMarker, make([]byte, 65)); err == nil {
		t.Fatal("oversized payload appended")
	}
	if err := writer.AppendJSON(kindMarker, func() {}); err == nil {
		t.Fatal("unencodable payload appended")
	}
	if err := writer.Append(kindMarker, []byte(`{"marker":"after rejected input"}`)); err != nil {
		t.Fatalf("caller input errors poisoned the writer: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second Close error = %v", err)
	}
	if err := writer.Append(kindMarker, []byte(`{}`)); !errors.Is(err, ErrClosed) {
		t.Fatalf("Append after Close error = %v", err)
	}
	replayed, report := replayState(t, path)
	requireEntries(t, replayed.entries, 3)
	if len(replayed.markers) != 2 || report.Records != 5 || report.Tail != TailNone || report.ObservedTailBytes != 0 {
		t.Fatalf("replayed %#v, report %#v", replayed, report)
	}
	var reopened state
	writer, report, err = OpenWriter(path, testOptions(true, nil), reopened.consume)
	if err != nil {
		t.Fatal(err)
	}
	defer closeWriter(t, writer)
	requireEntries(t, reopened.entries, 3)
	if writer.Records() != 5 || writer.Offset() != report.ValidBytes || report.RepairedBytes != 0 {
		t.Fatalf("reopened writer at %d records, %d bytes, report %#v", writer.Records(), writer.Offset(), report)
	}
}

func TestSecondWriterIsInUseThroughAnyAcceptedAlias(t *testing.T) {
	path := newStore(t, 1)
	parent := filepath.Dir(path)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Fatal(err)
	}
	writer, _, err := OpenWriter(path, testOptions(false, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := tree(t, path)
	for _, other := range []string{path, filepath.Join(alias, "store"), parent + "/./store", filepath.Join(parent, "..", filepath.Base(parent), "store")} {
		if second, _, err := OpenWriter(other, testOptions(true, nil), nil); !errors.Is(err, ErrInUse) {
			if second != nil {
				closeWriter(t, second)
			}
			t.Fatalf("second writer at %s error = %v", other, err)
		}
		replayed, _ := replayState(t, other)
		requireEntries(t, replayed.entries, 1)
	}
	assertTree(t, path, before)
	if err := writer.AppendJSON(kindEntry, makeEntry(1)); err != nil {
		t.Fatalf("contended opens disturbed the owner: %v", err)
	}
	closeWriter(t, writer)
	writer, _, err = OpenWriter(filepath.Join(alias, "store"), testOptions(false, nil), nil)
	if err != nil {
		t.Fatalf("alias writer after release: %v", err)
	}
	closeWriter(t, writer)
}

// TestReadOnlyReplayNeverMutates covers torn tails, corruption and missing or
// partial stores: a reader creates, repairs, chmods and locks nothing.
func TestReadOnlyReplayNeverMutates(t *testing.T) {
	log, boundaries := entryFrames(t, 2)
	torn := rawStore(t, log[:boundaries[2]-3])
	before := tree(t, filepath.Dir(torn))
	for range 2 {
		replayed, report := replayState(t, torn)
		requireEntries(t, replayed.entries, 1)
		if report.Tail != TailShortFrame || report.ObservedTailBytes != boundaries[2]-3-boundaries[1] || report.RepairedBytes != 0 {
			t.Fatalf("torn report = %#v", report)
		}
	}
	assertTree(t, filepath.Dir(torn), before)
	lock, err := os.OpenFile(filepath.Join(torn, LockFile), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("reader left a lock behind: %v", err)
	}
	if _, report := replayState(t, torn); report.Records != 1 {
		t.Fatalf("reader under a held writer lock = %#v", report)
	}

	parent := t.TempDir()
	for _, setup := range []struct {
		name  string
		build func(path string)
	}{
		{"missing-store", func(string) {}},
		{"missing-lock", func(path string) {
			mustMkdir(t, path)
			mustWrite(t, filepath.Join(path, LogFile), nil)
		}},
		{"missing-log", func(path string) {
			mustMkdir(t, path)
			mustWrite(t, filepath.Join(path, LockFile), nil)
		}},
	} {
		path := filepath.Join(parent, setup.name)
		setup.build(path)
		before := tree(t, parent)
		if _, err := Replay(path, testOptions(true, nil), nil); err == nil {
			t.Fatalf("%s replayed", setup.name)
		}
		assertTree(t, parent, before)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestInvalidMetadataFailsWithoutMutation rejects symbolic links, hard-link
// aliases, nonregular members and wrong modes for readers and for writers that
// may create, leaving the store and an external sentinel unchanged.
func TestInvalidMetadataFailsWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name   string
		damage func(t *testing.T, path, sentinel string) string
	}{
		{"root-symlink", func(t *testing.T, path, _ string) string {
			link := path + "-link"
			if err := os.Symlink(path, link); err != nil {
				t.Fatal(err)
			}
			return link
		}},
		{"root-not-directory", func(t *testing.T, path, sentinel string) string { return sentinel }},
		{"root-mode", func(t *testing.T, path, _ string) string { chmod(t, path, 0o750); return path }},
		{"root-setgid", func(t *testing.T, path, _ string) string { chmod(t, path, 0o700|os.ModeSetgid); return path }},
		{"log-symlink", func(t *testing.T, path, sentinel string) string {
			replaceWithSymlink(t, filepath.Join(path, LogFile), sentinel)
			return path
		}},
		{"lock-symlink", func(t *testing.T, path, sentinel string) string {
			replaceWithSymlink(t, filepath.Join(path, LockFile), sentinel)
			return path
		}},
		{"log-hard-link", func(t *testing.T, path, _ string) string {
			link(t, filepath.Join(path, LogFile), path+"-log-alias")
			return path
		}},
		{"lock-hard-link", func(t *testing.T, path, _ string) string {
			link(t, filepath.Join(path, LockFile), path+"-lock-alias")
			return path
		}},
		{"log-directory", func(t *testing.T, path, _ string) string {
			replaceWithDirectory(t, filepath.Join(path, LogFile))
			return path
		}},
		{"lock-directory", func(t *testing.T, path, _ string) string {
			replaceWithDirectory(t, filepath.Join(path, LockFile))
			return path
		}},
		{"log-fifo", func(t *testing.T, path, _ string) string {
			replaceWithFIFO(t, filepath.Join(path, LogFile))
			return path
		}},
		{"lock-fifo", func(t *testing.T, path, _ string) string {
			replaceWithFIFO(t, filepath.Join(path, LockFile))
			return path
		}},
		{"log-mode", func(t *testing.T, path, _ string) string { chmod(t, filepath.Join(path, LogFile), 0o644); return path }},
		{"lock-mode", func(t *testing.T, path, _ string) string { chmod(t, filepath.Join(path, LockFile), 0o640); return path }},
		{"log-setuid", func(t *testing.T, path, _ string) string {
			chmod(t, filepath.Join(path, LogFile), 0o600|os.ModeSetuid)
			return path
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := newStore(t, 2)
			sentinel := filepath.Join(filepath.Dir(path), "sentinel")
			mustWrite(t, sentinel, []byte("sentinel"))
			chmod(t, sentinel, 0o644)
			opened := test.damage(t, path, sentinel)
			before := tree(t, filepath.Dir(path))
			if _, err := Replay(opened, testOptions(true, nil), nil); !errors.Is(err, ErrInvalidStore) {
				t.Fatalf("Replay error = %v", err)
			}
			if writer, _, err := OpenWriter(opened, testOptions(true, nil), nil); !errors.Is(err, ErrInvalidStore) {
				if writer != nil {
					closeWriter(t, writer)
				}
				t.Fatalf("OpenWriter error = %v", err)
			}
			assertTree(t, filepath.Dir(path), before)
		})
	}
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&^os.ModeType != mode {
		t.Skipf("host did not apply mode %v to %s: %v", mode, path, err)
	}
}

func link(t *testing.T, path, alias string) {
	t.Helper()
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
}

func replaceWithSymlink(t *testing.T, path, target string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func replaceWithDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	mustMkdir(t, path)
}

func replaceWithFIFO(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
}

// replacement swaps one store object for an impostor, keeping the original
// under a moved name so the test can prove neither was mutated.
func replacement(t *testing.T, path, object string) {
	t.Helper()
	if object == "root" {
		if err := os.Rename(path, path+".moved"); err != nil {
			t.Fatal(err)
		}
		mustMkdir(t, path)
		mustWrite(t, filepath.Join(path, LogFile), []byte("impostor"))
		mustWrite(t, filepath.Join(path, LockFile), []byte("impostor"))
		return
	}
	if err := os.Rename(filepath.Join(path, object), filepath.Join(path, object+".moved")); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(path, object), []byte("impostor"))
}

// TestReplacementDuringOpenAppendAndRepairIsRejected deterministically swaps
// the root, log or lock at each boundary after that object is opened. The
// operation fails before any append or truncation, and the original store,
// the impostor and an external sentinel keep their contents and modes.
func TestReplacementDuringOpenAppendAndRepairIsRejected(t *testing.T) {
	type boundary struct {
		point   Point
		objects []string
	}
	all := []string{"root", LockFile, LogFile}
	writerBoundaries := []boundary{
		{Point{PointOpen, "root"}, []string{"root"}},
		{Point{PointOpen, LockFile}, []string{"root", LockFile}},
		{Point{PointOpen, LogFile}, all},
		{Point{PointTruncate, "repair"}, all},
		{Point{PointWrite, "append"}, all},
	}
	readerBoundaries := []boundary{
		{Point{PointOpen, "root"}, []string{"root"}},
		{Point{PointOpen, LockFile}, []string{"root", LockFile}},
		{Point{PointOpen, LogFile}, all},
	}
	run := func(t *testing.T, reader bool, at Point, object string) {
		log, boundaries := entryFrames(t, 2)
		path := rawStore(t, log[:boundaries[2]-1])
		if at.Kind == PointWrite {
			writer, _, err := OpenWriter(path, testOptions(false, nil), nil)
			if err != nil {
				t.Fatal(err)
			}
			closeWriter(t, writer)
		}
		parent := filepath.Dir(path)
		mustWrite(t, filepath.Join(parent, "sentinel"), []byte("sentinel"))
		var want map[string]string
		armed := true
		hook := func(point Point) error {
			if armed && point == at {
				armed = false
				replacement(t, path, object)
				want = tree(t, parent)
			}
			return nil
		}
		var err error
		switch {
		case reader:
			_, err = Replay(path, testOptions(false, hook), nil)
		case at.Kind == PointWrite:
			writer, _, openErr := OpenWriter(path, testOptions(false, hook), nil)
			if openErr != nil {
				t.Fatal(openErr)
			}
			err = writer.AppendJSON(kindEntry, makeEntry(1))
			if !errors.Is(err, ErrPoisoned) || !errors.Is(writer.AppendJSON(kindEntry, makeEntry(1)), ErrPoisoned) {
				t.Fatalf("replacement before append did not poison the writer: %v", err)
			}
			closeWriter(t, writer)
		default:
			writer, _, openErr := OpenWriter(path, testOptions(false, hook), nil)
			if writer != nil {
				closeWriter(t, writer)
			}
			err = openErr
		}
		if armed {
			t.Fatalf("boundary %#v was never observed", at)
		}
		if !errors.Is(err, ErrInvalidStore) {
			t.Fatalf("replacement of %s at %#v error = %v", object, at, err)
		}
		assertTree(t, parent, want)
	}
	for _, mode := range []struct {
		name       string
		reader     bool
		boundaries []boundary
	}{{"writer", false, writerBoundaries}, {"reader", true, readerBoundaries}} {
		for _, boundary := range mode.boundaries {
			for _, object := range boundary.objects {
				t.Run(fmt.Sprintf("%s-%s-%s-%s", mode.name, boundary.point.Kind, boundary.point.Operation, object), func(t *testing.T) {
					run(t, mode.reader, boundary.point, object)
				})
			}
		}
	}
}

// TestChangedReaderSnapshotFailsWithoutRepair shrinks or replaces the log
// while a read-only replay holds its snapshot.
func TestChangedReaderSnapshotFailsWithoutRepair(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(t *testing.T, path string)
	}{
		{"truncated", func(t *testing.T, path string) {
			if err := os.Truncate(filepath.Join(path, LogFile), 3); err != nil {
				t.Fatal(err)
			}
		}},
		{"replaced", func(t *testing.T, path string) { replacement(t, path, LogFile) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := newStore(t, 3)
			calls := 0
			_, err := Replay(path, testOptions(false, nil), func(Record) error {
				calls++
				if calls == 1 {
					test.change(t, path)
				}
				return nil
			})
			if !errors.Is(err, ErrSnapshotChanged) {
				t.Fatalf("Replay error = %v after %d records", err, calls)
			}
		})
	}
}

// TestWriterRejectsExternalLogChangesBeforeAppend detects a log that no longer
// ends at the acknowledged offset and poisons rather than appending after it.
func TestWriterRejectsExternalLogChangesBeforeAppend(t *testing.T) {
	path := newStore(t, 1)
	writer, _, err := OpenWriter(path, testOptions(false, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer closeWriter(t, writer)
	file, err := os.OpenFile(filepath.Join(path, LogFile), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("external"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	before := readLog(t, path)
	if err := writer.AppendJSON(kindEntry, makeEntry(1)); !errors.Is(err, ErrPoisoned) || !errors.Is(err, ErrInvalidStore) {
		t.Fatalf("append after external change error = %v", err)
	}
	if string(readLog(t, path)) != string(before) {
		t.Fatal("poisoned writer changed the log")
	}
}

// TestConcurrentAppendsAndLockFreeReaders runs appenders and read-only
// replays together; every reader sees a valid prefix and the final log holds
// every acknowledged record.
func TestConcurrentAppendsAndLockFreeReaders(t *testing.T) {
	path := newStore(t, 0)
	writer, _, err := OpenWriter(path, testOptions(false, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer closeWriter(t, writer)
	const appenders, perAppender, readers = 4, 8, 4
	var group sync.WaitGroup
	errs := make(chan error, appenders*perAppender+readers*perAppender)
	for appender := range appenders {
		group.Go(func() {
			for index := range perAppender {
				errs <- writer.AppendJSON(kindMarker, marker{fmt.Sprintf("%d-%d", appender, index)})
			}
		})
	}
	for range readers {
		group.Go(func() {
			for range perAppender {
				var replayed state
				report, err := Replay(path, testOptions(false, nil), replayed.consume)
				if err == nil && report.Records != uint64(len(replayed.markers)) {
					err = fmt.Errorf("reader report %#v for %d markers", report, len(replayed.markers))
				}
				errs <- err
			}
		})
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	replayed, report := replayState(t, path)
	if len(replayed.markers) != appenders*perAppender || report.Records != writer.Records() || report.ValidBytes != writer.Offset() {
		t.Fatalf("final report %#v for %d markers", report, len(replayed.markers))
	}
}
