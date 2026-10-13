package logstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	kindEntry  Kind = 1
	kindMarker Kind = 2
)

var testKinds = map[Kind]uint32{kindEntry: 256, kindMarker: 64}

type entry struct {
	Sequence uint64 `json:"sequence"`
	Text     string `json:"text"`
}

type marker struct {
	Marker string `json:"marker"`
}

func testOptions(create bool, hook Hook) Options {
	return Options{Kinds: testKinds, Create: create, Hook: hook}
}

// state is a strict consumer: it decodes and validates a whole record before
// applying any of it, so a rejected record leaves state unchanged.
type state struct {
	entries []entry
	markers []string
}

func (s *state) consume(record Record) error {
	switch record.Kind {
	case kindEntry:
		var decoded entry
		if err := DecodeJSON(record.Payload, &decoded); err != nil {
			return err
		}
		if decoded.Sequence != uint64(len(s.entries)) {
			return fmt.Errorf("entry sequence %d, want %d", decoded.Sequence, len(s.entries))
		}
		s.entries = append(s.entries, decoded)
	case kindMarker:
		var decoded marker
		if err := DecodeJSON(record.Payload, &decoded); err != nil {
			return err
		}
		s.markers = append(s.markers, decoded.Marker)
	default:
		return fmt.Errorf("unexpected kind %d", record.Kind)
	}
	return nil
}

func entryText(sequence int) string {
	return strings.Repeat(string(rune('a'+sequence%26)), 3+sequence*7)
}

func makeEntry(sequence int) entry {
	return entry{Sequence: uint64(sequence), Text: entryText(sequence)}
}

// newStore creates a store under a fresh parent with count entries and
// returns the store path.
func newStore(t *testing.T, count int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store")
	writer, _, err := OpenWriter(path, testOptions(true, nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	for sequence := range count {
		if err := writer.AppendJSON(kindEntry, makeEntry(sequence)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// rawStore creates a private store whose log holds exactly contents.
func rawStore(t *testing.T, contents []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{LockFile: nil, LogFile: contents} {
		if err := os.WriteFile(filepath.Join(path, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(path, name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func frameOf(t *testing.T, kind Kind, value any) []byte {
	t.Helper()
	payload, err := EncodeJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	return encodeFrame(kind, payload)
}

func entryFrames(t *testing.T, count int) ([]byte, []int64) {
	t.Helper()
	var log []byte
	boundaries := []int64{0}
	for sequence := range count {
		log = append(log, frameOf(t, kindEntry, makeEntry(sequence))...)
		boundaries = append(boundaries, int64(len(log)))
	}
	return log, boundaries
}

func readLog(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(path, LogFile))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// tree records every object below directory: type, mode, size and contents
// (or link target), for proving that an operation mutated nothing.
func tree(t *testing.T, directory string) map[string]string {
	t.Helper()
	objects := map[string]string{}
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		description := fmt.Sprintf("%v %d", info.Mode(), info.Size())
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			description += " -> " + target
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			description += " " + string(data)
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		objects[relative] = description
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return objects
}

func assertTree(t *testing.T, directory string, want map[string]string) {
	t.Helper()
	got := tree(t, directory)
	for name, description := range want {
		if got[name] != description {
			t.Errorf("%s = %q, want %q", name, got[name], description)
		}
	}
	for name := range got {
		if _, found := want[name]; !found {
			t.Errorf("unexpected object %s = %q", name, got[name])
		}
	}
}

func replayState(t *testing.T, path string) (state, Report) {
	t.Helper()
	var replayed state
	report, err := Replay(path, testOptions(false, nil), replayed.consume)
	if err != nil {
		t.Fatal(err)
	}
	return replayed, report
}

func requireEntries(t *testing.T, got []entry, count int) {
	t.Helper()
	if len(got) != count {
		t.Fatalf("replayed %d entries, want %d", len(got), count)
	}
	for sequence, value := range got {
		if value != makeEntry(sequence) {
			t.Fatalf("entry %d = %#v", sequence, value)
		}
	}
}

func closeWriter(t *testing.T, writer *Writer) {
	t.Helper()
	if err := writer.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
}
