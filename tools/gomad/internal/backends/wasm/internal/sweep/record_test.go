package sweep

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validSidecar() Sidecar {
	candidate := Candidate{GomadSHA256: "sha256:g", HelperSHA256: "sha256:h", CompilerSHA256: "sha256:c", GoVersion: "go1.27.1", ModelSHA256: "sha256:m",
		RuntimeImplementationSHA256: "sha256:r", Engine: "wasmtime/1", Profile: "gomad.wasi-cooperative/v1", GitSHA: "abc", ConfigSource: ConfigThrowaway,
		BuildTags: GuestTags(), PerTestTimeout: "10m0s", PrepareTimeout: "30m0s"}
	return Sidecar{Schema: SidecarSchema, Statement: ObservationStatement, Candidate: candidate, CandidateKey: candidate.Key(), Hosts: []Host{{Hostname: "h"}},
		Packages: []Package{{Pattern: "./tests", Tests: []string{"TestA", "TestB", "TestC"}, ListCount: 3, ParsedCount: 3}},
		Rows: []TestEntry{{Package: "./tests", Name: "TestA", Class: ClassPass}, {Package: "./tests", Name: "TestB", Class: ClassPass, Skipped: true},
			{Package: "./tests", Name: "TestC", Class: ClassRefusedImport, Reason: "execute: sock_accept"}}}
}

func TestVerifyFindsEveryR2Violation(t *testing.T) {
	t.Parallel()
	if findings := Verify(validSidecar()); len(findings) != 0 {
		t.Fatalf("valid sidecar: %v", findings)
	}
	for _, mutation := range []struct {
		name   string
		change func(*Sidecar)
		want   string
	}{
		{"missing-row", func(s *Sidecar) { s.Rows = s.Rows[:2] }, "TestC: 0 rows"},
		{"duplicate-row", func(s *Sidecar) { s.Rows = append(s.Rows, s.Rows[0]) }, "TestA: 2 rows"},
		{"unknown-test", func(s *Sidecar) {
			s.Rows = append(s.Rows, TestEntry{Package: "./tests", Name: "TestZ", Class: ClassPass})
		}, "not enumerated"},
		{"no-class", func(s *Sidecar) { s.Rows[0].Class = "" }, "invalid class"},
		{"bad-class", func(s *Sidecar) { s.Rows[0].Class = "flaky" }, "invalid class"},
		{"no-reason", func(s *Sidecar) { s.Rows[2].Reason = " " }, "without a reason"},
		{"not-run-no-reason", func(s *Sidecar) { s.Rows[2] = TestEntry{Package: "./tests", Name: "TestC", Class: ClassNotRun} }, "without a reason"},
		{"skipped-failure", func(s *Sidecar) { s.Rows[2].Skipped = true }, "skipped is only valid on pass"},
		{"missing-helper", func(s *Sidecar) { s.Candidate.HelperSHA256 = ""; s.CandidateKey = s.Candidate.Key() }, "missing helper_sha256"},
		{"missing-git", func(s *Sidecar) { s.Candidate.GitSHA = ""; s.CandidateKey = s.Candidate.Key() }, "missing git_sha"},
		{"stale-key", func(s *Sidecar) { s.Candidate.Seed = 9 }, "candidate key"},
		{"unreconciled", func(s *Sidecar) { s.Packages[0].ListCount = 4 }, "not reconciled"},
		{"statement", func(s *Sidecar) { s.Statement = "supported" }, "observation statement"},
		{"no-host", func(s *Sidecar) { s.Hosts = nil }, "host identity"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			t.Parallel()
			sidecar := validSidecar()
			mutation.change(&sidecar)
			findings := Verify(sidecar)
			if !strings.Contains(strings.Join(findings, "\n"), mutation.want) {
				t.Fatalf("findings %v, want %q", findings, mutation.want)
			}
		})
	}
}

func TestVerifyFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sweep.json")
	encoded, _ := json.Marshal(validSidecar())
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if findings, err := VerifyFile(path); err != nil || len(findings) != 0 {
		t.Fatalf("%v %v", findings, err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":"gomad-wasm.sweep/v1","support":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(path); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestJournalIgnoresOnlyATornFinalLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sweep.jsonl")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Append(Row{Kind: "test", Candidate: "k", Test: &TestEntry{Package: "./tests", Name: "TestA", Class: ClassPass}}); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"schema":"gomad-wasm.sweep-journal/v1","kind":"te`); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadJournal(path)
	if err != nil || len(rows) != 1 {
		t.Fatalf("torn tail: %d rows, %v", len(rows), err)
	}
	if err := os.WriteFile(path, []byte("garbage\n"+`{"schema":"gomad-wasm.sweep-journal/v1","kind":"header"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadJournal(path); err == nil {
		t.Fatal("malformed interior line accepted")
	}
}
