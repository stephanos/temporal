package qualification

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

func TestMandatoryCorpusCannotAcceptPartialOrProvisionalEvidence(t *testing.T) {
	if err := ValidateCorpus(nil); err == nil {
		t.Fatal("empty corpus passed")
	}
	if err := ValidateCorpus([]Result{{Manifest: testManifest()}}); err == nil {
		t.Fatal("provisional fixture substituted for four frozen workloads")
	}
}

func TestMandatoryCorpusRequiresPublicationReceipts(t *testing.T) {
	if err := ValidateCorpus([]Result{{}}); err == nil || !strings.Contains(err.Error(), "receipt") {
		t.Fatal("missing terminal receipt accepted")
	}
	if err := ValidateCorpus([]Result{{}}, PublicationReceipt{}, PublicationReceipt{}); err == nil {
		t.Fatal("extra terminal receipts accepted")
	}
}

func TestCorpusEntriesDistinguishResultsFromTinyReceipts(t *testing.T) {
	name := "result-" + strings.Repeat("a", 64) + ".json"
	for _, test := range []struct {
		name   string
		result bool
	}{{name, true}, {name + ".publication.json", false}} {
		path, result, err := corpusResultEntry(t.TempDir(), test.name)
		if err != nil || result != test.result || (result && filepath.Base(path) != name) {
			t.Fatalf("entry %q: result=%t path=%s error=%v", test.name, result, path, err)
		}
	}
	for _, name := range []string{"unknown.json", ".result-orphan.partial", "result-foreign.json", "result-foreign.json.publication.json", "../result-" + strings.Repeat("a", 64) + ".json"} {
		if _, _, err := corpusResultEntry(t.TempDir(), name); err == nil {
			t.Fatalf("unknown corpus entry accepted: %s", name)
		}
	}
}

func TestCorpusEntryPairsRequireExactlyOneReceipt(t *testing.T) {
	name := "result-" + strings.Repeat("a", 64) + ".json"
	other := "result-" + strings.Repeat("b", 64) + ".json"
	if err := validateCorpusEntryPairs([]string{name, name + ".publication.json"}); err != nil {
		t.Fatal(err)
	}
	for _, entries := range [][]string{{name}, {name + ".publication.json"}, {name, name + ".publication.json", name + ".publication.json"}, {name, other + ".publication.json"}, {name, name, name + ".publication.json"}, {name, name + ".publication.json", "foreign.json"}} {
		if err := validateCorpusEntryPairs(entries); err == nil {
			t.Fatalf("invalid corpus entries accepted: %v", entries)
		}
	}
}

func TestActualMandatoryWorkloadCorpus(t *testing.T) {
	checkerBefore := hostexec.CurrentProcessResourceUsage()
	qualificationIntent(t)
	if os.Getenv("GOMAD_WASM_QUALIFY_WORKLOADS") == "" {
		t.Skip("mandatory full frozen corpus gate requires explicit intent")
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("absolute fresh mandatory corpus evidence root required")
	}
	entries, err := os.ReadDir(filepath.Join(root, "wasm-qualifications", "v1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 128 {
		t.Fatal("mandatory corpus report count bound")
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	if err := validateCorpusEntryPairs(names); err != nil {
		t.Fatal(err)
	}
	results := []Result{}
	receipts := []PublicationReceipt{}
	var cpu uint64
	for _, entry := range entries {
		path, isResult, err := corpusResultEntry(filepath.Join(root, "wasm-qualifications", "v1"), entry.Name())
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
		results = append(results, result)
		receipt, err := OpenPublicationReceipt(path)
		if err != nil {
			t.Fatal(err)
		}
		tailCPU, err := ValidatePublicationReceipt(path, result, receipt)
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, receipt)
		if ^uint64(0)-cpu < tailCPU {
			t.Fatal("corpus terminal self CPU overflow")
		}
		cpu += tailCPU
		if result.Cost != nil {
			if ^uint64(0)-cpu < result.Cost.CPUNanos {
				t.Fatal("corpus CPU overflow")
			}
			cpu += result.Cost.CPUNanos
		}
	}
	if err := ValidateCorpus(results, receipts...); err != nil {
		t.Fatal(err)
	}
	checkerAfter := hostexec.CurrentProcessResourceUsage()
	if checkerBefore == nil || checkerAfter == nil || checkerAfter.PeakRSSBytes == 0 || checkerAfter.PeakRSSBytes > 2<<30 || checkerAfter.UserNanos < checkerBefore.UserNanos || checkerAfter.SystemNanos < checkerBefore.SystemNanos {
		t.Fatal("mandatory corpus checker self observations unavailable, nonmonotone, or over 2GiB")
	}
	user, system := checkerAfter.UserNanos-checkerBefore.UserNanos, checkerAfter.SystemNanos-checkerBefore.SystemNanos
	if ^uint64(0)-user < system || ^uint64(0)-cpu < user+system {
		t.Fatal("corpus checker self CPU overflow")
	}
	checkerCPU := user + system
	t.Logf("mandatory frozen actual workload cells=%d producers compiler+helper+operator including terminal publication CPU nanos=%d; checker self CPU nanos=%d lifetime RSS bytes=%d; total corpus CPU nanos=%d; each cold producer counted only in its own fresh cell", len(results), cpu, checkerCPU, checkerAfter.PeakRSSBytes, cpu+checkerCPU)
}
