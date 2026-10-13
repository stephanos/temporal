package gomadfs_test

import (
	"bytes"
	"errors"
	"io"
	"syscall"
	"testing"

	. "internal/gomadfs"
)

func TestFilesystemHandleOperations(t *testing.T) {
	for name, create := range map[string]func() *FS{"standalone": New, "in-process": NewSimulation} {
		t.Run(name, func(t *testing.T) {
			fs := create()
			file, err := fs.Open("/file", OpenFlags{Read: true, Write: true, Create: true}, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if n, err := file.Write([]byte("abcd")); n != 4 || err != nil {
				t.Fatalf("Write=%d,%v", n, err)
			}
			if n, err := file.Seek(1, io.SeekStart); n != 1 || err != nil {
				t.Fatalf("Seek=%d,%v", n, err)
			}
			data := make([]byte, 3)
			if n, err := file.ReadAt(data, 2); n != 2 || err != io.EOF || !bytes.Equal(data, []byte{'c', 'd', 0}) {
				t.Fatalf("ReadAt=%d,%v,%q", n, err, data)
			}
			if n, err := file.Read(data[:1]); n != 1 || err != nil || data[0] != 'b' {
				t.Fatalf("offset Read=%d,%v,%q", n, err, data[:1])
			}
			if n, err := file.WriteAt([]byte("Z"), 0); n != 1 || err != nil {
				t.Fatalf("WriteAt=%d,%v", n, err)
			}
			if n, err := file.Seek(0, io.SeekCurrent); n != 2 || err != nil {
				t.Fatalf("offset=%d,%v", n, err)
			}
			if err := file.Truncate(2); err != nil {
				t.Fatal(err)
			}
			if err := file.Chmod(0640); err != nil {
				t.Fatal(err)
			}
			if err := file.Chtimes(123); err != nil {
				t.Fatal(err)
			}
			entry, err := file.Stat()
			if err != nil || entry.Mode != 0640 || entry.ModTime != 123 || string(entry.Data) != "Zb" || file.Path() != "/file" {
				t.Fatalf("Stat=%+v,%v Path=%q", entry, err, file.Path())
			}
			appender, err := fs.Open("/file", OpenFlags{Write: true, Append: true}, 0)
			if err != nil {
				t.Fatal(err)
			}
			if n, err := appender.WriteAt([]byte("bad"), 0); n != 0 || err != syscall.EINVAL {
				t.Fatalf("append WriteAt=%d,%v", n, err)
			}
			if n, err := appender.Write([]byte("!")); n != 1 || err != nil {
				t.Fatalf("append Write=%d,%v", n, err)
			}
			if err := appender.Close(); err != nil {
				t.Fatal(err)
			}
			if err := fs.Remove("/file"); err != nil {
				t.Fatal(err)
			}
			if n, err := file.ReadAt(data, 0); n != 3 || err != nil || string(data) != "Zb!" {
				t.Fatalf("unlinked ReadAt=%d,%v,%q", n, err, data)
			}
			if err := file.Sync(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != ErrClosed {
				t.Fatalf("closed identity=%v", err)
			}
			if _, err := file.Map(-1, 0, true); err != ErrClosed {
				t.Fatalf("local closed map=%v", err)
			}
			if stats := fs.Statistics(); stats != (Statistics{}) {
				t.Fatalf("released=%+v", stats)
			}
		})
	}
}

func TestFilesystemMappingAccountingTransfersAndSurvivesFileClose(t *testing.T) {
	fs := New()
	file, err := fs.Open("/file", OpenFlags{Read: true, Write: true, Create: true}, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("abcd")); err != nil {
		t.Fatal(err)
	}
	first, err := file.Map(0, 64<<20, true)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := file.Map(0, 64<<20, true)
	if err != nil {
		t.Fatal(err)
	}
	if stats := fs.Statistics(); stats.MappedBytes != 64<<20 {
		t.Fatalf("alias charge=%d", stats.MappedBytes)
	}
	if _, err := file.Map(1, 64<<20, true); err != syscall.EINVAL {
		t.Fatalf("overlap before capacity=%v", err)
	}
	if _, err := file.Map(64<<20, 1, false); err != syscall.ENOMEM {
		t.Fatalf("capacity=%v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if stats := fs.Statistics(); stats.MappedBytes != 64<<20 {
		t.Fatalf("transferred charge=%d", stats.MappedBytes)
	}
	if _, err := first.Bytes(); err != syscall.EINVAL {
		t.Fatalf("closed bytes=%v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := alias.Bytes()
	if err != nil || string(contents[:4]) != "abcd" {
		t.Fatalf("surviving map=%q,%v", contents[:4], err)
	}
	copy(contents, "live")
	reader, err := fs.Open("/file", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 4)
	if _, err := reader.Read(buffer); err != nil || string(buffer) != "live" {
		t.Fatalf("mapped stores=%q,%v", buffer, err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := alias.Close(); err != nil {
		t.Fatal(err)
	}
	if stats := fs.Statistics(); stats.MappedBytes != 0 {
		t.Fatalf("final charge=%d", stats.MappedBytes)
	}
}

func TestFilesystemMountedMappingRemainsReadOnly(t *testing.T) {
	fs := New()
	fs.SetLoader(func(string) (LoadEntry, MountStatus, error) {
		return LoadEntry{Kind: KindFile, Mode: 0600, Data: []byte("mount")}, MountOK, nil
	})
	file, err := fs.Open("/mount", OpenFlags{Read: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Map(0, 5, true); err != syscall.EBADF {
		t.Fatalf("writable map=%v", err)
	}
	if err := file.Chmod(0777); err != syscall.EROFS {
		t.Fatalf("chmod=%v", err)
	}
	if err := file.Chtimes(999); err != syscall.EROFS {
		t.Fatalf("chtimes=%v", err)
	}
	mapping, err := file.Map(0, 5, false)
	if err != nil {
		t.Fatal(err)
	}
	defer mapping.Close()
	contents, err := mapping.Bytes()
	if err != nil || string(contents) != "mount" {
		t.Fatalf("map=%q,%v", contents, err)
	}
	contents[0] = 'X'
	entry, err := file.Stat()
	if err != nil || string(entry.Data) != "mount" || entry.Mode != 0600 || entry.ModTime != 0 {
		t.Fatalf("immutable mount=%+v,%v", entry, err)
	}
}

func TestFilesystemProcessWritableMapPrecedesClosedAndBounds(t *testing.T) {
	file := ClosedProcessHandleForTest()
	if _, err := file.Map(-1, 0, true); !errors.Is(err, syscall.ENOTSUP) {
		t.Fatalf("process writable map=%v", err)
	}
	if _, err := file.Map(-1, 0, false); err != ErrClosed {
		t.Fatalf("process readonly map=%v", err)
	}
}

func TestFilesystemHandleObserverRejectsWriteBeforeMutation(t *testing.T) {
	fs := newTestVolumeFilesystem(t)
	file, err := fs.Open("/data/file", OpenFlags{Read: true, Write: true, Create: true}, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	before := len(fs.Operations()["data"])
	fs.SetVolumeObserver(rejectingVolumeObserver{err: syscall.EIO})
	if n, err := file.Write([]byte("payload")); n != 0 || err != syscall.EIO {
		t.Fatalf("rejected write=%d,%v", n, err)
	}
	entry, err := file.Stat()
	if err != nil || len(entry.Data) != 0 || len(fs.Operations()["data"]) != before || fs.Statistics().UsedBytes != 0 {
		t.Fatalf("rejected write mutated volume: %+v,%v stats=%+v", entry, err, fs.Statistics())
	}
}

func TestFilesystemHandleCapacityAndStaleBeforeClosed(t *testing.T) {
	fs := newTestVolumeFilesystem(t)
	var handles []*Handle
	for range 100_000 {
		file, err := fs.Open("/file", OpenFlags{Read: true, Create: true}, 0600)
		if err != nil {
			t.Fatal(err)
		}
		handles = append(handles, file)
	}
	if _, err := fs.Open("/rejected", OpenFlags{Read: true, Create: true}, 0600); err != syscall.EMFILE {
		t.Fatalf("handle capacity=%v", err)
	}
	if _, err := fs.Stat("/rejected"); err != syscall.ENOENT {
		t.Fatalf("capacity mutated namespace=%v", err)
	}
	for _, file := range handles {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	file := handles[0]
	if err := file.Sync(); err != ErrClosed {
		t.Fatalf("closed sync=%v", err)
	}
	if err := fs.CrashVolumes(nil); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []func() error{file.Close, file.Sync, func() error { _, err := file.Read(make([]byte, 1)); return err }, func() error { _, err := file.Map(0, 1, false); return err }} {
		if err := operation(); err != syscall.ESTALE {
			t.Fatalf("generation precedes closed=%v", err)
		}
	}
}
