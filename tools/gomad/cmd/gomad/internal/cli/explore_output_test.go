package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

type exploreFailureWriter struct {
	stream string
	writes *[]string
	errors []error
}

func (writer *exploreFailureWriter) Write(data []byte) (int, error) {
	*writer.writes = append(*writer.writes, writer.stream+": "+string(data))
	if len(writer.errors) > 0 {
		err := writer.errors[0]
		writer.errors = writer.errors[1:]
		if err != nil {
			return 0, err
		}
	}
	return len(data), nil
}

func TestExploreErrorJoinsDiagnosticAndReporterWriterFailures(t *testing.T) {
	for _, finalWriteFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "final_write_succeeds", true: "final_write_fails"}[finalWriteFails], func(t *testing.T) {
			var writes []string
			var finalError error
			if finalWriteFails {
				finalError = errors.New("final write failed")
			}
			stdout := &exploreFailureWriter{stream: "stdout", writes: &writes, errors: []error{errors.New("reporter write failed")}}
			stderr := &exploreFailureWriter{stream: "stderr", writes: &writes, errors: []error{errors.New("diagnostic write failed"), finalError}}
			reporter := newExploreReporter(true, stdout, stderr)
			status := reportExploreFailure(runner.CampaignResult{ChoiceTrace: &record.ChoiceTraceSummary{}}, errors.New("bad request"), reporter, stderr)
			want := []string{
				"stderr: gomad: choices-seed=0 choices-profile= choices-records=0 choices-decisions=0 choices-branching=0 choices-runnable=0 choices-select-poll=0 choices-select-result=0 choices-sha256= choices-tape-sha256= choices-terminal=\n",
				"stdout: {\"schema\":\"gomad.explore-event/v3\",\"type\":\"error\",\"classification\":\"invalid_input\",\"message\":\"bad request\"}\n",
				"stderr: write explore event: reporter write failed\ndiagnostic write failed\n",
			}
			if status != 3 || !reflect.DeepEqual(writes, want) {
				t.Fatalf("status = %d, writes = %#v, want status 3 and writes %#v", status, writes, want)
			}
		})
	}
}

func TestSoftAssertionProgressReportsEscapedObservations(t *testing.T) {
	seed := uint64(0)
	progress := runner.CampaignEvent{Phase: runner.ProgressRunning, ExecutionSeed: &seed, SoftAssertions: &record.SoftAssertions{Messages: []string{"invalid\nstate"}, ReportOnly: true}}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Progress(progress); err != nil {
			t.Fatal(err)
		}
		if jsonOutput {
			var event exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.ExecutionSeed == nil || *event.ExecutionSeed != 0 || event.SoftAssertions == nil || event.SoftAssertions.Messages[0] != "invalid\nstate" || !event.SoftAssertions.ReportOnly || stderr.Len() != 0 {
				t.Fatalf("JSON oracle report: %+v stderr=%q", event, stderr.String())
			}
		} else if stdout.Len() != 0 || !strings.Contains(stderr.String(), `seed=0 report-only=true inconclusive=false messages=["invalid\nstate"]`) {
			t.Fatalf("human oracle report: stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	}
	progress.ExecutionSeed = nil
	if err := newExploreReporter(false, &bytes.Buffer{}, &bytes.Buffer{}).Progress(progress); err == nil {
		t.Fatal("oracle observation without a seed admitted")
	}
}
