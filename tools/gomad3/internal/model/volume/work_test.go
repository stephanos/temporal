package volume_test

import (
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/model/volume"
)

func TestTinyWorkResumesWithExactlyTheUnrestrictedStatesAndOrder(t *testing.T) {
	fs := newFilesystem(t)
	file := openFile(t, fs, "/data/value")
	writeAt(t, file, "value", 0)
	expected := enumerate(t, fs, 64)
	before := fs.VolumeSnapshots()
	for _, work := range []uint64{1, 2, 3} {
		limits := volume.CrashEnumerationLimits{States: 64, Operations: 128, Depth: 128, Bytes: 1 << 20, Work: work}
		var cursor *volume.CrashExploration
		var actual []volume.CrashState
		seen := make(map[string]bool)
		complete := false
		emptyPages := 0
		for pageNumber := 0; pageNumber < 64; pageNumber++ {
			page, err := fs.EnumerateCrashStates("data", limits, cursor)
			if err != nil {
				t.Fatal(err)
			}
			for _, state := range page.States {
				if seen[state.Identity] {
					t.Fatalf("work=%d returned duplicate state %s", work, state.Identity)
				}
				seen[state.Identity] = true
			}
			actual = append(actual, page.States...)
			if page.Complete {
				complete = true
				break
			}
			if page.Capacity != volume.CrashCapacityWork || page.ChoiceExploration == nil {
				t.Fatalf("work=%d unexpected capacity %#v", work, page)
			}
			if len(page.States) == 0 {
				emptyPages++
			}
			if cursor != nil && reflect.DeepEqual(cursor.Cursor, page.ChoiceExploration.Cursor) {
				t.Fatalf("work=%d made no cursor progress", work)
			}
			cursor = page.ChoiceExploration
		}
		if !complete {
			t.Fatalf("work=%d failed to complete", work)
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("work=%d pagination lost or reordered states:\n%#v\nwant %#v", work, actual, expected)
		}
		if work == 1 && emptyPages == 0 {
			t.Fatal("tiny work fixture did not exercise invalid/duplicate selections")
		}
		if !reflect.DeepEqual(fs.VolumeSnapshots(), before) {
			t.Fatal("enumeration changed model state")
		}
	}
}

func TestCrashEnumerationRejectsZeroWork(t *testing.T) {
	fs := newFilesystem(t)
	limits := volume.CrashEnumerationLimits{States: 1, Operations: 1, Depth: 1, Bytes: 1024}
	if _, err := fs.EnumerateCrashStates("data", limits, nil); err == nil {
		t.Fatal("zero work limit accepted")
	}
}
