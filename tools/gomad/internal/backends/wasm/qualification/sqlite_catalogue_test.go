package qualification

import (
	"os"
	"path/filepath"
	"testing"

	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// Each selected cell is an ordinary existing Result from a distinct producer.
// This separate reader proves exact catalogue coverage and retained bindings;
// it does not execute another simulator or import guessed preparation costs.
func TestActualSQLiteSelectionCatalogue(t *testing.T) {
	checkerBefore := hostexec.CurrentProcessResourceUsage()
	path := os.Getenv("GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT")
	if path == "" {
		t.Skip("explicit actual native SQLite source Result and supplementary evidence root required")
	}
	qualificationIntent(t)
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(path) || !filepath.IsAbs(root) {
		t.Fatal("absolute native source Result and supplementary root required")
	}
	source, err := OpenResult(path)
	if err != nil {
		t.Fatal(err)
	}
	sourceReceipt, err := OpenPublicationReceipt(path)
	if err != nil {
		t.Fatal(err)
	}
	nativeCatalogue, err := MatchedNativeCrashEnumeration(source, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "wasm-qualifications", "v1")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 64 {
		t.Fatal("bounded SQLite catalogue entry capacity")
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	if err := validateCorpusEntryPairs(names); err != nil {
		t.Fatal(err)
	}
	results, receipts := []Result{}, []PublicationReceipt{}
	var producersCPU uint64
	for _, entry := range entries {
		path, isResult, err := corpusResultEntry(directory, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !isResult {
			continue
		}
		result, err := OpenResult(path)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := OpenPublicationReceipt(path)
		if err != nil {
			t.Fatal(err)
		}
		tailCPU, err := ValidatePublicationReceipt(path, result, receipt)
		if err != nil {
			t.Fatal(err)
		}
		if result.Cost == nil || ^uint64(0)-producersCPU < result.Cost.CPUNanos || ^uint64(0)-(producersCPU+result.Cost.CPUNanos) < tailCPU {
			t.Fatal("selected cell CPU unavailable or overflowing")
		}
		producersCPU += result.Cost.CPUNanos + tailCPU
		results, receipts = append(results, result), append(receipts, receipt)
	}
	if err := ValidateSQLiteSelectionResults(source, sourceReceipt, results, receipts...); err != nil {
		t.Fatal(err)
	}
	checkerAfter := hostexec.CurrentProcessResourceUsage()
	if checkerBefore == nil || checkerAfter == nil || checkerAfter.PeakRSSBytes == 0 || checkerAfter.PeakRSSBytes > 2<<30 || checkerAfter.UserNanos < checkerBefore.UserNanos || checkerAfter.SystemNanos < checkerBefore.SystemNanos {
		t.Fatal("SQLite catalogue reader self metric unavailable/nonmonotone/over2GiB")
	}
	user, system := checkerAfter.UserNanos-checkerBefore.UserNanos, checkerAfter.SystemNanos-checkerBefore.SystemNanos
	if ^uint64(0)-user < system || ^uint64(0)-producersCPU < user+system {
		t.Fatal("SQLite catalogue reader CPU overflow")
	}
	t.Logf("native reviewed crash states=%d exact selected cells=%d producer CPU including terminal publication=%d reader self CPU=%d reader lifetime RSS=%d supplementary total CPU=%d; primary source cost belongs to its existing corpus cell and is not counted again", len(nativeCatalogue.States), len(results), producersCPU, user+system, checkerAfter.PeakRSSBytes, producersCPU+user+system)
}
