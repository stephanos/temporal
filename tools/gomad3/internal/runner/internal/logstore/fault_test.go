//go:build unix

package logstore

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

const scriptAppends = 3

var errInjected = errors.New("injected storage failure")

// fault selects one observed boundary by index and how to fail it.
type fault struct {
	index   int
	partial *PartialWrite
}

// faultHook fails the selected boundary once and records every point seen.
func faultHook(selected fault, seen *[]Point) Hook {
	index := 0
	return func(point Point) error {
		*seen = append(*seen, point)
		current := index
		index++
		if current != selected.index {
			return nil
		}
		if selected.partial != nil {
			return selected.partial
		}
		return errInjected
	}
}

// runScript creates a store and appends scriptAppends entries, stopping at the
// first failure like a crashed process. It returns how many appends were
// acknowledged and proves that a failed append poisons the writer.
func runScript(t *testing.T, path string, hook Hook) (int, error) {
	t.Helper()
	writer, _, err := OpenWriter(path, testOptions(true, hook), (&state{}).consume)
	if err != nil {
		if writer != nil {
			t.Fatal("OpenWriter returned a writer with an error")
		}
		return 0, err
	}
	defer closeWriter(t, writer)
	for sequence := range scriptAppends {
		if err := writer.AppendJSON(kindEntry, makeEntry(sequence)); err != nil {
			if !errors.Is(err, ErrPoisoned) {
				t.Fatalf("failed append %d did not poison the writer: %v", sequence, err)
			}
			before := readLog(t, path)
			if retry := writer.AppendJSON(kindEntry, makeEntry(sequence)); !errors.Is(retry, ErrPoisoned) {
				t.Fatalf("poisoned writer accepted an append: %v", retry)
			}
			if after := readLog(t, path); string(after) != string(before) {
				t.Fatal("poisoned writer changed the log")
			}
			return sequence, err
		}
	}
	return scriptAppends, nil
}

// requireRecovery proves that every acknowledged entry survives a crash at
// the current state of path, that exactly the expected uncertain frame
// survives with it, and that the recovered writer appends again.
func requireRecovery(t *testing.T, path string, acknowledged, survivors int) {
	t.Helper()
	parent := filepath.Dir(path)
	before := tree(t, parent)
	var observed state
	report, readErr := Replay(path, testOptions(false, nil), observed.consume)
	assertTree(t, parent, before)
	if readErr == nil {
		requireEntries(t, observed.entries, survivors)
		if report.RepairedBytes != 0 {
			t.Fatalf("read-only replay repaired: %#v", report)
		}
	} else if acknowledged != 0 {
		t.Fatalf("read-only replay of acknowledged store: %v", readErr)
	}
	var recovered state
	writer, report, err := OpenWriter(path, testOptions(true, nil), recovered.consume)
	if err != nil {
		t.Fatalf("recover after %d acknowledged appends: %v", acknowledged, err)
	}
	defer closeWriter(t, writer)
	requireEntries(t, recovered.entries, survivors)
	if report.RepairedBytes != report.ObservedTailBytes || writer.Offset() != report.ValidBytes || int64(len(readLog(t, path))) != report.ValidBytes {
		t.Fatalf("recovery report %#v, writer offset %d", report, writer.Offset())
	}
	if err := writer.AppendJSON(kindEntry, makeEntry(survivors)); err != nil {
		t.Fatal(err)
	}
	closeWriter(t, writer)
	replayed, report := replayState(t, path)
	requireEntries(t, replayed.entries, survivors+1)
	if report.Tail != TailNone {
		t.Fatalf("recovered log report = %#v", report)
	}
}

// TestInjectedCreateAppendAndSyncFaultsPreserveAcknowledgedCommits fails every
// create, directory-sync, open, write, sync and post-sync boundary of a store
// creation plus appends, including a partial write of every frame prefix. For
// each failure it then models a crash that loses any unsynced suffix at every
// byte. Acknowledged appends always survive; an uncertain complete frame
// survives only when it was fully written.
func TestInjectedCreateAppendAndSyncFaultsPreserveAcknowledgedCommits(t *testing.T) {
	var points []Point
	if acknowledged, err := runScript(t, filepath.Join(t.TempDir(), "store"), faultHook(fault{index: -1}, &points)); err != nil || acknowledged != scriptAppends {
		t.Fatalf("unfaulted script = %d, %v", acknowledged, err)
	}
	for _, required := range []PointKind{PointCreate, PointDirectorySync, PointOpen, PointWrite, PointSync, PointSynced} {
		if !slices.ContainsFunc(points, func(point Point) bool { return point.Kind == required }) {
			t.Fatalf("script observed no %s boundary: %v", required, points)
		}
	}
	writes := 0
	for index, point := range points {
		faults := []fault{{index: index}}
		if point.Kind == PointWrite {
			frame := len(frameOf(t, kindEntry, makeEntry(writes)))
			for size := 0; size <= frame; size++ {
				faults = append(faults, fault{index: index, partial: &PartialWrite{Bytes: size}})
			}
			writes++
		}
		for _, selected := range faults {
			name := fmt.Sprintf("%02d-%s-%s", index, point.Kind, point.Operation)
			if selected.partial != nil {
				name += fmt.Sprintf("-partial-%d", selected.partial.Bytes)
			}
			t.Run(name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "store")
				var seen []Point
				acknowledged, err := runScript(t, path, faultHook(selected, &seen))
				if !errors.Is(err, errInjected) && !errors.As(err, new(*PartialWrite)) {
					t.Fatalf("script error = %v", err)
				}
				if seen[index] != point {
					t.Fatalf("boundary %d = %#v, want %#v", index, seen[index], point)
				}
				complete := point.Kind == PointSync || point.Kind == PointSynced ||
					selected.partial != nil && selected.partial.Bytes == len(frameOf(t, kindEntry, makeEntry(acknowledged)))
				survivors := acknowledged
				if complete {
					survivors++
				}
				if point.Kind != PointWrite && point.Kind != PointSync && point.Kind != PointSynced {
					requireRecovery(t, path, acknowledged, survivors)
					return
				}
				log := readLog(t, path)
				acknowledgedEnd, _ := entryFrames(t, acknowledged)
				for cut := len(acknowledgedEnd); cut <= len(log); cut++ {
					if selected.partial != nil && cut != len(log) {
						continue
					}
					lost := survivors
					if cut != len(log) {
						lost = acknowledged
					}
					requireRecovery(t, rawStore(t, log[:cut]), acknowledged, lost)
				}
			})
		}
	}
}

// TestInjectedRepairFaultsNeverLoseCommittedRecords fails every boundary of a
// writer-open that must truncate a torn tail. A failed repair leaves either
// the original torn log or the truncated prefix, and a later writer-open
// completes the repair.
func TestInjectedRepairFaultsNeverLoseCommittedRecords(t *testing.T) {
	log, boundaries := entryFrames(t, 2)
	torn := log[:boundaries[2]-5]
	var points []Point
	writer, report, err := OpenWriter(rawStore(t, torn), testOptions(false, faultHook(fault{index: -1}, &points)), nil)
	if err != nil {
		t.Fatal(err)
	}
	closeWriter(t, writer)
	if report.RepairedBytes == 0 || !slices.Contains(points, Point{PointTruncate, "repair"}) || !slices.Contains(points, Point{PointSync, "repair"}) {
		t.Fatalf("repair observed %v with report %#v", points, report)
	}
	for index, point := range points {
		t.Run(fmt.Sprintf("%02d-%s-%s", index, point.Kind, point.Operation), func(t *testing.T) {
			path := rawStore(t, torn)
			var seen []Point
			writer, _, err := OpenWriter(path, testOptions(false, faultHook(fault{index: index}, &seen)), nil)
			if writer != nil || !errors.Is(err, errInjected) {
				t.Fatalf("faulted repair = %v, %v", writer, err)
			}
			got := readLog(t, path)
			truncate := slices.Index(seen, Point{PointTruncate, "repair"})
			want := torn
			if truncate >= 0 && truncate < index {
				want = log[:boundaries[1]]
			}
			if string(got) != string(want) {
				t.Fatalf("failed repair left %d log bytes", len(got))
			}
			requireRecovery(t, path, 1, 1)
		})
	}
}
