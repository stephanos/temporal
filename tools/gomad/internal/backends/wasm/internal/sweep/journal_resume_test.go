package sweep

import (
	"os"
	"testing"
)

func TestSweepResumesAfterInterruptedJournalAppend(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, tail string
		retained   int
	}{
		{"partial", "{\"schema\":\"gomad-wasm.sweep-journal/v1\",\"kind\":", 0},
		{"complete-without-newline", "{\"schema\":\"gomad-wasm.sweep-journal/v1\",\"kind\":\"header\"}", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			if findings, err := f.sweeper().Run(t.Context()); err != nil || len(findings) != 0 {
				t.Fatalf("first sweep: %v %v", findings, err)
			}
			before := len(f.explores)
			path := JournalPath(f.options.Out)
			rowsBefore, err := ReadJournal(path)
			if err != nil {
				t.Fatal(err)
			}
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.WriteString(test.tail); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if findings, err := f.sweeper().Run(t.Context()); err != nil || len(findings) != 0 {
				t.Fatalf("resume after interrupted append: %v %v", findings, err)
			}
			if len(f.explores) != before {
				t.Fatalf("resume repeated completed functions: before %d after %d", before, len(f.explores))
			}
			rows, err := ReadJournal(path)
			if err != nil {
				t.Fatalf("journal remains unreadable after resume: %v", err)
			}
			if len(rows) != len(rowsBefore)+1+test.retained {
				t.Fatalf("valid journal rows not preserved: before %d after %d", len(rowsBefore), len(rows))
			}
		})
	}
}
