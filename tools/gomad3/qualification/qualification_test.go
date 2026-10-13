package qualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestBuildReportQualifiesRepeatedSuccessfulEvidence(t *testing.T) {
	evidence := successfulEvidence()
	command := []string{"gomad", "qualify", "--seed", "7", "go-test", "./pkg"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: command,
		Executions: []QualificationExecution{
			{CampaignPath: "/artifacts/run-1", Evidence: evidence},
			{CampaignPath: "/artifacts/run-2", Evidence: evidence},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != QualificationReportSchema || !report.Qualified || !report.Deterministic || !report.TargetSuccess || report.Seed != 7 || report.Repeat != 2 || report.EvidenceDigest == "" || report.FirstDivergence != "" {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Executions) != 2 || report.Executions[0].CampaignPath != "/artifacts/run-1" || report.Executions[0].EvidenceDigest != report.EvidenceDigest {
		t.Fatalf("runs = %#v", report.Executions)
	}
	command[0] = "changed"
	if report.Command[0] != "gomad" {
		t.Fatal("report command was not copied")
	}
}

func TestBuildReportNamesFirstEvidenceDivergence(t *testing.T) {
	first := successfulEvidence()
	second := first
	second.Stdout.FullSHA256 = record.HashBytes([]byte("different"))
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: first},
		{CampaignPath: "/artifacts/run-2", Evidence: second},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || report.Deterministic || report.FirstDivergence != "stdout.full_sha256" || report.Executions[0].EvidenceDigest == report.Executions[1].EvidenceDigest {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsFailureReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Outcome = runner.OutcomeEvidence{Domain: "target", Reason: "nonzero_exit", Termination: "exit"}
	report, err := BuildQualificationReport(QualificationInput{
		Command: []string{"gomad", "qualify"},
		Executions: []QualificationExecution{
			{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/failure-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/failure-1", Attempted: true, Match: true}},
			{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/failure-2", Evidence: evidence},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || !report.Deterministic || report.TargetSuccess || report.Executions[0].Replay == nil || !report.Executions[0].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRecordsPerRunSuccessfulReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Schema != QualificationReportSchema || !report.Qualified || len(report.Executions) != 2 || report.Executions[0].Replay == nil || !report.Executions[0].Replay.Match || report.Executions[1].Replay == nil || !report.Executions[1].Replay.Match {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRequiresExactChoiceStatusForMatchedChoiceReplay(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Choices = &runner.ChoiceEvidence{Profile: "gomad3-choice-trace/v3"}
	runs := []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true}},
	}
	if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: runs}); err == nil {
		t.Fatal("BuildQualificationReport() accepted matched choice replay without exact status")
	}
	for index := range runs {
		runs[index].Replay.ChoiceReplayStatus = ChoiceReplayExact
	}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: runs})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Qualified {
		t.Fatalf("report = %#v", report)
	}
}

func TestBuildReportRejectsMismatchedPerRunReplay(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Match: true}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Divergence: "stdout.full_sha256"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || ClassifyQualification(report) != "replay_divergence" {
		t.Fatalf("report = %#v", report)
	}
}

// The runner reports an exact choice replay whenever the choice tape replays
// without divergence, even when other evidence (a stream digest) still
// diverges; that is a replay divergence outside the choice tape, not invalid
// evidence.
func TestBuildReportClassifiesExactChoiceReplayWithEvidenceDivergence(t *testing.T) {
	evidence := successfulEvidence()
	evidence.Choices = &runner.ChoiceEvidence{Profile: "gomad3-choice-trace/v3"}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify", "--replay-successes"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", ArtifactPath: "/artifacts/success-1", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-1", Attempted: true, Divergence: "stderr.full_sha256", ChoiceReplayStatus: ChoiceReplayExact}},
		{CampaignPath: "/artifacts/run-2", ArtifactPath: "/artifacts/success-2", Evidence: evidence, Replay: &QualificationReplay{ArtifactPath: "/artifacts/success-2", Attempted: true, Match: true, ChoiceReplayStatus: ChoiceReplayExact}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || ClassifyQualification(report) != "replay_divergence" {
		t.Fatalf("report = %#v", report)
	}
	if _, err := WriteQualificationReport(t.TempDir(), report); err != nil {
		t.Fatal(err)
	}
}

func TestBuildReportRejectsInvalidEvidenceSet(t *testing.T) {
	evidence := successfulEvidence()
	for _, input := range []QualificationInput{
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}}},
		{Command: nil, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "", Evidence: evidence}, {CampaignPath: "/two", Evidence: evidence}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: evidence}, {CampaignPath: "/two", Evidence: withSeed(evidence, 8)}}},
		{Command: []string{"gomad"}, Executions: []QualificationExecution{{CampaignPath: "/one", Evidence: withSchema(evidence, "bad")}, {CampaignPath: "/two", Evidence: evidence}}},
	} {
		if _, err := BuildQualificationReport(input); err == nil {
			t.Fatalf("BuildQualificationReport(%#v) succeeded", input)
		}
	}
}

func TestBuildFailureRetainsFirstUnsupportedBoundary(t *testing.T) {
	report, err := BuildQualificationFailure(
		[]string{"gomad", "qualify", "go-test", "./pkg"},
		7,
		2,
		nil,
		QualificationFailure{
			Classification: "unsupported_target", Message: "example.com/target imports os/exec", Iteration: 1,
			ImportPath: "example.com/target", Capability: "imports os/exec",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || report.Failure == nil || report.Failure.Capability != "imports os/exec" || report.Seed != 7 || report.Repeat != 2 || len(report.Executions) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestWriteReportRetainsCanonicalPrivateFile(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: evidence},
		{CampaignPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	path, err := WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(path)) != "v1" {
		t.Fatalf("report path = %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %o", info.Mode().Perm())
	}
	opened, err := OpenQualificationReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.EvidenceDigest != report.EvidenceDigest || !opened.Qualified {
		t.Fatalf("opened report = %#v", opened)
	}
}

func TestWriteRejectsInconsistentDeterministicOutcome(t *testing.T) {
	evidence := successfulEvidence()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: evidence},
		{CampaignPath: "/artifacts/run-2", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	report.TargetSuccess = false
	report.Qualified = false
	if _, err := WriteQualificationReport(t.TempDir(), report); err == nil {
		t.Fatal("WriteQualificationReport() accepted an outcome inconsistent with deterministic evidence")
	}
}

func TestQualificationReportStoragePreservation(t *testing.T) {
	report := qualificationStorageReport(t)
	root := t.TempDir()
	paths := make(map[string]bool)
	for range 2 {
		path, err := WriteQualificationReport(root, report)
		if err != nil {
			t.Fatal(err)
		}
		if paths[path] || filepath.Dir(path) != filepath.Join(root, "qualifications", "v1") || !strings.HasPrefix(filepath.Base(path), "qualification-") || !strings.HasSuffix(path, ".json") || filepath.Base(path) == "qualification-.json" {
			t.Fatalf("published path = %q", path)
		}
		paths[path] = true
		for _, entry := range []struct {
			path string
			mode os.FileMode
		}{
			{path, 0o600},
			{filepath.Join(root, "qualifications"), 0o700},
			{filepath.Dir(path), 0o700},
		} {
			info, err := os.Stat(entry.path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != entry.mode {
				t.Fatalf("mode of %s = %o, want %o", entry.path, info.Mode().Perm(), entry.mode)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(data, []byte{'\n'}) != 1 || data[len(data)-1] != '\n' {
			t.Fatalf("encoded report framing changed: bytes=%q", data)
		}
		opened, err := OpenQualificationReport(path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(opened, report) || opened.EvidenceDigest == "" || opened.Executions[0].EvidenceDigest != opened.EvidenceDigest || opened.Executions[1].EvidenceDigest != opened.EvidenceDigest || !opened.Qualified || !opened.Deterministic || !opened.TargetSuccess || opened.Executions[0].Replay != nil || opened.Executions[1].Replay != nil {
			t.Fatalf("decoded report = %#v", opened)
		}
		evidence, err := json.Marshal(opened.Evidence)
		if err != nil {
			t.Fatal(err)
		}
		if record.DomainHash(ExecutionEvidenceDigestDomain, evidence) != opened.EvidenceDigest {
			t.Fatalf("decoded evidence digest = %s", record.DomainHash(ExecutionEvidenceDigestDomain, evidence))
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "qualifications", "v1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("published entries = %#v", entries)
	}
	for _, entry := range entries {
		if !paths[filepath.Join(root, "qualifications", "v1", entry.Name())] || entry.IsDir() {
			t.Fatalf("unexpected publication or staging remnant = %s", entry.Name())
		}
	}
}

func TestQualificationReportWriteValidationPreservation(t *testing.T) {
	report := qualificationStorageReport(t)
	for _, test := range []struct {
		name   string
		root   string
		report QualificationReport
		want   string
	}{
		{"empty root", "", report, "artifact root is required"},
		{"root before schema", "", QualificationReport{}, "artifact root is required"},
		{"invalid schema", filepath.Join(t.TempDir(), "absent"), QualificationReport{}, "unsupported qualification report schema \"\""},
		{"invalid report", filepath.Join(t.TempDir(), "absent"), QualificationReport{Schema: QualificationReportSchema}, "qualification report command or repetition count is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path, err := WriteQualificationReport(test.root, test.report)
			if path != "" || err == nil || err.Error() != test.want || errors.Unwrap(err) != nil {
				t.Fatalf("write = %q, %v, want empty path and %q", path, err, test.want)
			}
			if test.root != "" {
				if _, err := os.Stat(test.root); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("validation changed filesystem: %v", err)
				}
			}
		})
	}
	t.Run("non-directory ancestor", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(root, []byte("unchanged"), 0o600); err != nil {
			t.Fatal(err)
		}
		path, err := WriteQualificationReport(root, report)
		var pathErr *os.PathError
		if path != "" || err == nil || !strings.HasPrefix(err.Error(), "create qualification report directory: ") || !errors.As(err, &pathErr) || errors.Unwrap(err) != pathErr || pathErr.Op != "mkdir" || pathErr.Path != root || !errors.Is(err, syscall.ENOTDIR) {
			t.Fatalf("write = %q, %#v", path, err)
		}
		data, err := os.ReadFile(root)
		if err != nil || string(data) != "unchanged" {
			t.Fatalf("ancestor = %q, %v", data, err)
		}
	})
}

func TestQualificationReportReadValidationPreservation(t *testing.T) {
	path, err := WriteQualificationReport(t.TempDir(), qualificationStorageReport(t))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		data []byte
		want string
	}{
		{"empty", nil, "qualification report must be between 1 and 16777216 bytes"},
		{"malformed", []byte("{"), "decode qualification report:"},
		{"invalid utf8", []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}, "decode qualification report:"},
		{"duplicate", bytes.Replace(data, []byte(`"schema":`), []byte(`"schema":"gomad3.qualification/v1","schema":`), 1), "decode qualification report:"},
		{"unknown", bytes.Replace(data, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1), "decode qualification report:"},
		{"trailing", append(append([]byte(nil), data...), []byte(`{}`)...), "decode qualification report:"},
		{"old schema", bytes.Replace(data, []byte("gomad3.qualification/v1"), []byte("gomad3.qualification/v0"), 1), "unsupported qualification report schema \"gomad3.qualification/v0\""},
		{"future schema", bytes.Replace(data, []byte("gomad3.qualification/v1"), []byte("gomad3.qualification/v2"), 1), "unsupported qualification report schema \"gomad3.qualification/v2\""},
		{"invalid evidence", bytes.Replace(data, []byte("gomad3.execution-evidence/v1"), []byte("gomad3.execution-evidence/v0"), 1), "qualification baseline evidence identity is invalid"},
		{"invalid evidence digest", bytes.Replace(data, []byte(qualificationStorageReport(t).EvidenceDigest), []byte("sha256:invalid"), 1), "qualification baseline evidence digest is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.json")
			if err := os.WriteFile(path, test.data, 0o600); err != nil {
				t.Fatal(err)
			}
			opened, err := OpenQualificationReport(path)
			if !reflect.DeepEqual(opened, QualificationReport{}) || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("read = %#v, %v, want zero report and %q", opened, err, test.want)
			}
		})
	}
	t.Run("missing path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "absent")
		opened, err := OpenQualificationReport(path)
		var pathErr *os.PathError
		if !reflect.DeepEqual(opened, QualificationReport{}) || err == nil || !strings.HasPrefix(err.Error(), "open qualification report: ") || !errors.As(err, &pathErr) || errors.Unwrap(err) != pathErr || pathErr.Op != "open" || pathErr.Path != path || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read = %#v, %#v", opened, err)
		}
	})
	for _, kind := range []string{"directory", "oversized sparse file"} {
		t.Run(kind, func(t *testing.T) {
			path := t.TempDir()
			if kind == "oversized sparse file" {
				path = filepath.Join(path, "large.json")
				if err := os.WriteFile(path, nil, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Truncate(path, 16777217); err != nil {
					t.Fatal(err)
				}
			}
			opened, err := OpenQualificationReport(path)
			if !reflect.DeepEqual(opened, QualificationReport{}) || err == nil || err.Error() != "qualification report must be a regular file no larger than 16777216 bytes" || errors.Unwrap(err) != nil {
				t.Fatalf("read = %#v, %v", opened, err)
			}
		})
	}
	t.Run("without newline", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "report.json")
		if err := os.WriteFile(path, bytes.TrimSuffix(data, []byte{'\n'}), 0o600); err != nil {
			t.Fatal(err)
		}
		opened, err := OpenQualificationReport(path)
		if err != nil || !reflect.DeepEqual(opened, qualificationStorageReport(t)) {
			t.Fatalf("read = %#v, %v", opened, err)
		}
	})
	t.Run("ordinary formatting", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "report.json")
		formatted := append([]byte(" \n\t"), bytes.TrimSuffix(data, []byte{'\n'})...)
		formatted = append(formatted, '\n', '\n')
		if err := os.WriteFile(path, formatted, 0o600); err != nil {
			t.Fatal(err)
		}
		opened, err := OpenQualificationReport(path)
		if err != nil || !reflect.DeepEqual(opened, qualificationStorageReport(t)) {
			t.Fatalf("read = %#v, %v", opened, err)
		}
	})
}

func TestBuildQualificationReportOwnsNestedEvidence(t *testing.T) {
	evidence := successfulEvidence()
	signal := "none"
	evidence.Outcome.Signal = &signal
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/one", Evidence: evidence},
		{CampaignPath: "/two", Evidence: evidence},
	}})
	if err != nil {
		t.Fatal(err)
	}
	evidence.Environment[0].Value = "changed"
	*evidence.Outcome.Signal = "changed"
	if report.Evidence.Environment[0].Value == "changed" || *report.Evidence.Outcome.Signal == "changed" {
		t.Fatalf("report aliases caller evidence: %#v", report.Evidence)
	}
}

func TestBuildQualificationReportRejectsInvalidOriginalStrings(t *testing.T) {
	for _, mutate := range []func(*QualificationInput){
		func(input *QualificationInput) { input.Command[0] = "gomad\xff" },
		func(input *QualificationInput) { input.Executions[0].CampaignPath = "/campaign\xff" },
		func(input *QualificationInput) { input.Executions[0].Evidence.RunnerBuild = "runner\xff" },
		func(input *QualificationInput) { input.Executions[0].Evidence.Target.Argv[0] = "target\xff" },
	} {
		evidence := successfulEvidence()
		input := QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
			{CampaignPath: "/one", Evidence: evidence},
			{CampaignPath: "/two", Evidence: evidence},
		}}
		mutate(&input)
		if _, err := BuildQualificationReport(input); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
			t.Fatalf("BuildQualificationReport() error = %v", err)
		}
	}
}

type qualificationStageStub struct {
	path        string
	published   bool
	publishErr  error
	cleanupErr  error
	destination string
}

func (stage *qualificationStageStub) Path() string { return stage.path }
func (stage *qualificationStageStub) PublishContext(_ context.Context, destination string) (bool, error) {
	stage.destination = destination
	return stage.published, stage.publishErr
}
func (stage *qualificationStageStub) Cleanup() error { return stage.cleanupErr }

func TestWriteQualificationReportPreservesPublicationDisposition(t *testing.T) {
	report := qualificationStorageReport(t)
	primary := errors.New("primary")
	cleanup := errors.New("cleanup")
	for _, test := range []struct {
		name          string
		stageErr      error
		published     bool
		publishErr    error
		cleanupErr    error
		wantPath      bool
		wantPrimary   error
		wantCleanup   bool
		cancelContext bool
	}{
		{name: "stage permission", stageErr: os.ErrPermission, wantPrimary: os.ErrPermission},
		{name: "before rename", publishErr: primary, wantPrimary: primary},
		{name: "before rename cleanup", publishErr: primary, cleanupErr: cleanup, wantPrimary: primary, wantCleanup: true},
		{name: "published not durable", published: true, publishErr: primary, cleanupErr: cleanup, wantPath: true, wantPrimary: primary, wantCleanup: true},
		{name: "published cleanup", cleanupErr: cleanup, wantPath: true, wantPrimary: cleanup, wantCleanup: true},
		{name: "cancelled", cancelContext: true, wantPrimary: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.cancelContext {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			root := t.TempDir()
			stub := &qualificationStageStub{published: test.published, publishErr: test.publishErr, cleanupErr: test.cleanupErr}
			stage := func(_ context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (qualificationStagedFile, error) {
				if test.stageErr != nil {
					return nil, test.stageErr
				}
				if pattern != ".qualification-*.partial" || mode != 0o600 {
					t.Fatalf("stage = %q, %o", pattern, mode)
				}
				stub.path = filepath.Join(directory, ".qualification-token.partial")
				var encoded bytes.Buffer
				if err := fill(ctx, &encoded); err != nil || encoded.Len() == 0 || encoded.Bytes()[encoded.Len()-1] != '\n' {
					t.Fatalf("fill = %q, %v", encoded.Bytes(), err)
				}
				return stub, nil
			}
			path, err := writeQualificationReportWith(ctx, root, report, stage)
			if (path != "") != test.wantPath || !errors.Is(err, test.wantPrimary) || test.wantCleanup != errors.Is(err, cleanup) {
				t.Fatalf("write = %q, %v", path, err)
			}
			if test.wantPath && filepath.Base(path) != "qualification-token.json" {
				t.Fatalf("published path = %q", path)
			}
		})
	}
}

func qualificationStorageReport(t *testing.T) QualificationReport {
	t.Helper()
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/artifacts/run-1", Evidence: successfulEvidence()},
		{CampaignPath: "/artifacts/run-2", Evidence: successfulEvidence()},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func successfulEvidence() runner.ExecutionEvidence {
	return runner.ExecutionEvidence{
		Schema: runner.ExecutionEvidenceSchema, Seed: 7, RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Outcome:     runner.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}

func withSeed(evidence runner.ExecutionEvidence, seed record.Uint64String) runner.ExecutionEvidence {
	evidence.Seed = seed
	return evidence
}

func withSchema(evidence runner.ExecutionEvidence, schema string) runner.ExecutionEvidence {
	evidence.Schema = schema
	return evidence
}
