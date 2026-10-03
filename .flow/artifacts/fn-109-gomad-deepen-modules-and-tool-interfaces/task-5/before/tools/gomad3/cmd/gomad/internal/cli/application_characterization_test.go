package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/qualification"
	qualificationworkload "go.temporal.io/server/tools/gomad3/qualification/workload"
)

func TestPublicCommandGrammarRejectsMissingOperandsBeforeInstallation(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "relative-invalid-root")
	for _, command := range []string{
		"plan", "execute-shard", "merge", "explore", "qualify", "qualify-set",
		"merge-set", "compare-support", "analyze", "resume", "recover", "replay",
		"minimize", "doctor", "inspect",
	} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			arguments := []string{command}
			if command == "doctor" {
				arguments = append(arguments, "unexpected")
			}
			if status := Run(arguments, &stdout, &stderr); status != 2 {
				t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
			}
			if strings.Contains(stdout.String()+stderr.String(), "relative-invalid-root") {
				t.Fatalf("Run(%q) resolved installation before rejecting syntax: stdout=%q stderr=%q", arguments, stdout.String(), stderr.String())
			}
		})
	}
}

func TestPublicCommandExplicitZeroAndIrrelevantFlagClassification(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{name: "explore count zero", arguments: []string{"explore", "--count=0", "go-run", "./pkg"}, want: "--count must be greater than zero"},
		{name: "explore irrelevant choice bytes", arguments: []string{"explore", "--choice-bytes=8MiB", "go-run", "./pkg"}, want: "--choice-bytes requires --choices"},
		{name: "plan missing output", arguments: []string{"plan", "go-run", "./pkg"}, want: "gomad plan requires --output"},
		{name: "qualify repeat zero", arguments: []string{"qualify", "--repeat=0", "go-run", "./pkg"}, want: "--repeat must be between"},
		{name: "analyze negative timeout", arguments: []string{"analyze", "--timeout=-1s", "go-run", "./pkg"}, want: "analysis timeout must be non-negative"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := Run(test.arguments, &stdout, &stderr); status != 2 || !strings.Contains(stdout.String()+stderr.String(), test.want) {
				t.Fatalf("Run(%q) status=%d stdout=%q stderr=%q; want %q", test.arguments, status, stdout.String(), stderr.String(), test.want)
			}
		})
	}
}

func TestPublicCommandWriterFailuresPreserveStatus(t *testing.T) {
	for _, arguments := range [][]string{
		{"explore", "--json", "--count=0", "go-run", "./pkg"},
		{"analyze", "--format=invalid", "go-run", "./pkg"},
		{"inspect", "/missing/artifact"},
	} {
		var stdout, stderr bytes.Buffer
		output := failingWriter{}
		if arguments[0] == "inspect" || arguments[0] == "analyze" {
			if status := Run(arguments, &stdout, output); status != 3 {
				t.Fatalf("Run(%q) status=%d stdout=%q", arguments, status, stdout.String())
			}
		} else if status := Run(arguments, output, &stderr); status != 3 {
			t.Fatalf("Run(%q) status=%d stderr=%q", arguments, status, stderr.String())
		}
	}
}

func TestQualifyGrammarPreservesEnvironmentTagsAndTargetArguments(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var observed qualificationworkload.Spec
		dependencies := qualifyDependencies{
			identity:         func(string) (string, string, string, error) { return "/toolchain", "/bin/gomad", "sha256:runner", nil },
			workingDirectory: func() (string, error) { return "/workspace", nil },
			workload: func(_ context.Context, spec qualificationworkload.Spec) (qualificationworkload.Result, error) {
				observed = spec
				return qualificationworkload.Result{
					Report:     qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true},
					ReportPath: "/report.json",
				}, nil
			},
		}
		arguments := []string{"--env=FOO=bar", "--build-tag=gomad_fixture", "go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"}
		if jsonOutput {
			arguments = append([]string{"--json"}, arguments...)
		}
		var stdout, stderr bytes.Buffer
		status := runQualifyWith(arguments, &stdout, &stderr, dependencies)
		if status != 0 || stderr.Len() != 0 {
			t.Fatalf("json=%t status=%d stdout=%q stderr=%q", jsonOutput, status, stdout.String(), stderr.String())
		}
		got := observed.Campaign
		if len(got.Environment) != 1 || got.Environment[0] != "FOO=bar" || len(got.Target.BuildTags) != 1 || got.Target.BuildTags[0] != "gomad_fixture" || len(got.Target.Args) != 2 || got.Target.Args[0] != "-test.run=Test Name" || got.Target.Args[1] != "literal;$value" {
			t.Fatalf("json=%t campaign=%#v", jsonOutput, got)
		}
		want := "gomad: qualification qualified=true"
		if jsonOutput {
			want = `"classification":"qualified"`
		}
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("json=%t output=%q, want %q", jsonOutput, stdout.String(), want)
		}
	}
}
