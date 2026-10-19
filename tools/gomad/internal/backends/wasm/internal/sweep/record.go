package sweep

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
)

const (
	JournalSchema = "gomad-wasm.sweep-journal/v1"
	SidecarSchema = "gomad-wasm.sweep/v1"
	// ObservationStatement is the first statement of every sidecar and
	// matrix: rows are observations on one candidate, never support claims.
	ObservationStatement = "Observation on one candidate only. A pass is an observed outcome of one seed under a bounded budget, not support or qualification."
)

// Candidate is the identity a row was observed on. Resume and verification
// compare its Key; changing any field starts a new sweep.
type Candidate struct {
	GomadSHA256                 string       `json:"gomad_sha256"`
	HelperSHA256                string       `json:"helper_sha256"`
	CompilerSHA256              string       `json:"compiler_sha256"`
	GoVersion                   string       `json:"go_version"`
	ModelSHA256                 string       `json:"model_sha256"`
	RuntimeImplementationSHA256 string       `json:"runtime_implementation_sha256"`
	Engine                      string       `json:"engine"`
	Profile                     string       `json:"profile"`
	GitSHA                      string       `json:"git_sha"`
	GitDirty                    bool         `json:"git_dirty"`
	ConfigSource                ConfigSource `json:"config_source"`
	BuildTags                   []string     `json:"build_tags"`
	Seed                        uint64       `json:"seed"`
	PerTestTimeout              string       `json:"per_test_timeout"`
	PrepareTimeout              string       `json:"prepare_timeout"`
}

// Key is the SHA-256 of the candidate's canonical JSON.
func (candidate Candidate) Key() string {
	encoded, _ := json.Marshal(candidate)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (candidate Candidate) missing() []string {
	var missing []string
	for name, value := range map[string]string{
		"gomad_sha256": candidate.GomadSHA256, "helper_sha256": candidate.HelperSHA256, "compiler_sha256": candidate.CompilerSHA256,
		"go_version": candidate.GoVersion, "model_sha256": candidate.ModelSHA256, "runtime_implementation_sha256": candidate.RuntimeImplementationSHA256,
		"engine": candidate.Engine, "profile": candidate.Profile, "git_sha": candidate.GitSHA, "config_source": string(candidate.ConfigSource),
		"per_test_timeout": candidate.PerTestTimeout,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(candidate.BuildTags) == 0 {
		missing = append(missing, "build_tags")
	}
	sort.Strings(missing)
	return missing
}

// Host identifies the machine a sweep invocation ran on.
type Host struct {
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	CPUs          int    `json:"cpus"`
	Kernel        string `json:"kernel,omitempty"`
	MemTotalBytes uint64 `json:"mem_total_bytes,omitempty"`
}

func currentHost() Host {
	host := Host{OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU()}
	host.Hostname, _ = os.Hostname()
	if release, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		host.Kernel = strings.TrimSpace(string(release))
	}
	if meminfo, err := os.ReadFile("/proc/meminfo"); err == nil {
		for line := range strings.SplitSeq(string(meminfo), "\n") {
			var kib uint64
			if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &kib); err == nil {
				host.MemTotalBytes = kib << 10
			}
		}
	}
	return host
}

// Row is one journal line. Kind "header" starts an invocation; kind "test"
// records one test function's outcome. A later test row for the same
// candidate, package and name supersedes an earlier not-run row.
type Row struct {
	Schema    string     `json:"schema"`
	Kind      string     `json:"kind"`
	Candidate string     `json:"candidate"`
	Time      string     `json:"time"`
	Header    *Header    `json:"header,omitempty"`
	Test      *TestEntry `json:"test,omitempty"`
}

// Header records one sweep invocation.
type Header struct {
	Statement     string    `json:"statement"`
	Candidate     Candidate `json:"candidate"`
	Host          Host      `json:"host"`
	Packages      []Package `json:"packages"`
	OverallBudget string    `json:"overall_budget"`
	MaxTests      int       `json:"max_tests"`
	RetainBytes   uint64    `json:"retain_bytes"`
	Command       []string  `json:"command"`
}

// TestEntry is the outcome of one top-level test function.
type TestEntry struct {
	Package            string    `json:"package"`
	Name               string    `json:"name"`
	Class              Class     `json:"class"`
	Reason             string    `json:"reason,omitempty"`
	Skipped            bool      `json:"skipped,omitempty"`
	Subtests           *Subtests `json:"subtests,omitempty"`
	Attempts           int       `json:"attempts"`
	FirstAttemptReason string    `json:"first_attempt_reason,omitempty"`
	GomadExit          int       `json:"gomad_exit"`
	WallSeconds        float64   `json:"wall_seconds"`
	PreparationBytes   uint64    `json:"preparation_bytes"`
	ArtifactBytes      uint64    `json:"artifact_bytes"`
	EngineCacheBytes   uint64    `json:"engine_cache_bytes"`
	EngineEntriesAdded int       `json:"engine_entries_added"`
	EngineCacheHit     bool      `json:"engine_cache_hit"`
	ArtifactRetained   bool      `json:"artifact_retained"`
	ArtifactPath       string    `json:"artifact_path,omitempty"`
	RetentionNote      string    `json:"retention_note,omitempty"`
}

// Journal is the append-only JSONL record of a sweep.
type Journal struct {
	path string
	file *os.File
}

func OpenJournal(path string) (*Journal, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	return &Journal{path: path, file: file}, nil
}

func (journal *Journal) Append(row Row) error {
	row.Schema = JournalSchema
	encoded, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if _, err := journal.file.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return journal.file.Sync()
}

func (journal *Journal) Close() error { return journal.file.Close() }

// ReadJournal decodes every row. A torn final line (an interrupted append)
// is ignored; any other malformed line is an error.
func ReadJournal(path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []Row
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	lines := bytes.Count(data, []byte("\n"))
	for index := 0; scanner.Scan(); index++ {
		var row Row
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil || row.Schema != JournalSchema {
			if index == lines && !bytes.HasSuffix(data, []byte("\n")) {
				break
			}
			return nil, fmt.Errorf("journal %s line %d is malformed", path, index+1)
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

// Latest returns the final test entry per package/name for one candidate.
func Latest(rows []Row, candidate string) map[string]TestEntry {
	latest := map[string]TestEntry{}
	for _, row := range rows {
		if row.Kind == "test" && row.Candidate == candidate && row.Test != nil {
			latest[row.Test.Package+"\x00"+row.Test.Name] = *row.Test
		}
	}
	return latest
}

// Sidecar is the machine-readable sweep result (task .2 retains it beside
// the support matrix).
type Sidecar struct {
	Schema       string                `json:"schema"`
	Statement    string                `json:"statement"`
	Candidate    Candidate             `json:"candidate"`
	CandidateKey string                `json:"candidate_key"`
	Hosts        []Host                `json:"hosts"`
	Budget       SidecarBudget         `json:"budget"`
	Packages     []Package             `json:"packages"`
	Rows         []TestEntry           `json:"rows"`
	ByClass      map[Class]int         `json:"by_class"`
	ByPackage    map[string]ClassTally `json:"by_package"`
	Reasons      []ReasonCount         `json:"reasons"`
}

type SidecarBudget struct {
	OverallBudget string `json:"overall_budget"`
	MaxTests      int    `json:"max_tests"`
	RetainBytes   uint64 `json:"retain_bytes"`
}

type ClassTally map[Class]int

type ReasonCount struct {
	Class  Class  `json:"class"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// BuildSidecar folds the journal's rows for the latest header's candidate
// into the sidecar. Every enumerated test appears exactly once, in package
// then name order; an enumerated test without a row is not-run.
func BuildSidecar(rows []Row) (Sidecar, error) {
	var header *Header
	var key string
	for _, row := range rows {
		if row.Kind == "header" && row.Header != nil {
			header, key = row.Header, row.Candidate
		}
	}
	if header == nil {
		return Sidecar{}, errors.New("journal has no header")
	}
	sidecar := Sidecar{Schema: SidecarSchema, Statement: ObservationStatement, Candidate: header.Candidate, CandidateKey: key, Packages: header.Packages,
		Budget:  SidecarBudget{OverallBudget: header.OverallBudget, MaxTests: header.MaxTests, RetainBytes: header.RetainBytes},
		ByClass: map[Class]int{}, ByPackage: map[string]ClassTally{}}
	for _, row := range rows {
		if row.Kind == "header" && row.Header != nil && row.Candidate == key && !slices.Contains(sidecar.Hosts, row.Header.Host) {
			sidecar.Hosts = append(sidecar.Hosts, row.Header.Host)
		}
	}
	latest := Latest(rows, key)
	reasons := map[[2]string]int{}
	for _, pkg := range header.Packages {
		for _, name := range pkg.Tests {
			entry, ok := latest[pkg.Pattern+"\x00"+name]
			if !ok {
				entry = TestEntry{Package: pkg.Pattern, Name: name, Class: ClassNotRun, Reason: "not reached by this sweep (budget or --max-tests)"}
			}
			sidecar.Rows = append(sidecar.Rows, entry)
			sidecar.ByClass[entry.Class]++
			if sidecar.ByPackage[pkg.Pattern] == nil {
				sidecar.ByPackage[pkg.Pattern] = ClassTally{}
			}
			sidecar.ByPackage[pkg.Pattern][entry.Class]++
			if entry.Class != ClassPass {
				reasons[[2]string{string(entry.Class), entry.Reason}]++
			}
		}
	}
	for reason, count := range reasons {
		sidecar.Reasons = append(sidecar.Reasons, ReasonCount{Class: Class(reason[0]), Reason: reason[1], Count: count})
	}
	sort.Slice(sidecar.Reasons, func(i, j int) bool {
		a, b := sidecar.Reasons[i], sidecar.Reasons[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Class != b.Class {
			return a.Class < b.Class
		}
		return a.Reason < b.Reason
	})
	return sidecar, nil
}

// Verify checks R2 on a sidecar: one valid class per enumerated test, a
// reason for every non-pass, reconciled enumeration counts, and a complete
// candidate identity that every row agrees with. It returns every finding.
func Verify(sidecar Sidecar) []string {
	var findings []string
	if sidecar.Schema != SidecarSchema {
		findings = append(findings, fmt.Sprintf("schema %q, want %q", sidecar.Schema, SidecarSchema))
	}
	if sidecar.Statement != ObservationStatement {
		findings = append(findings, "observation statement missing or altered")
	}
	if missing := sidecar.Candidate.missing(); len(missing) != 0 {
		findings = append(findings, "candidate identity missing "+strings.Join(missing, ", "))
	}
	if sidecar.CandidateKey != sidecar.Candidate.Key() {
		findings = append(findings, "candidate key does not match the candidate identity")
	}
	if len(sidecar.Hosts) == 0 {
		findings = append(findings, "no host identity recorded")
	}
	enumerated := map[string]bool{}
	for _, pkg := range sidecar.Packages {
		if pkg.ListCount != pkg.ParsedCount || pkg.ParsedCount != len(pkg.Tests) {
			findings = append(findings, fmt.Sprintf("%s: enumeration not reconciled (parsed %d, go test -list %d, tests %d)", pkg.Pattern, pkg.ParsedCount, pkg.ListCount, len(pkg.Tests)))
		}
		for _, name := range pkg.Tests {
			enumerated[pkg.Pattern+"\x00"+name] = true
		}
	}
	seen := map[string]int{}
	for _, row := range sidecar.Rows {
		id := row.Package + "\x00" + row.Name
		seen[id]++
		label := row.Package + " " + row.Name
		if !enumerated[id] {
			findings = append(findings, label+": row for a test that was not enumerated")
		}
		if !slices.Contains(Classes, row.Class) {
			findings = append(findings, fmt.Sprintf("%s: invalid class %q", label, row.Class))
		} else if row.Class != ClassPass && strings.TrimSpace(row.Reason) == "" {
			findings = append(findings, fmt.Sprintf("%s: class %s without a reason", label, row.Class))
		}
		if row.Skipped && row.Class != ClassPass {
			findings = append(findings, label+": skipped is only valid on pass")
		}
	}
	ids := make([]string, 0, len(enumerated))
	for id := range enumerated {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if count := seen[id]; count != 1 {
			findings = append(findings, fmt.Sprintf("%s: %d rows, want exactly one", strings.ReplaceAll(id, "\x00", " "), count))
		}
	}
	return findings
}

// VerifyFile decodes and verifies a sidecar file.
func VerifyFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var sidecar Sidecar
	if err := decoder.Decode(&sidecar); err != nil {
		return nil, fmt.Errorf("decode sidecar: %w", err)
	}
	return Verify(sidecar), nil
}

// WriteSidecar writes the sidecar atomically.
func WriteSidecar(path string, sidecar Sidecar) error {
	encoded, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(encoded, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
