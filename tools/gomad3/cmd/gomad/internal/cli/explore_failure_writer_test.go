package cli

import (
	"errors"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner"
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
			status := reportExploreFailure(runner.CampaignResult{ChoiceTrace: &runner.ChoiceTraceSummary{}}, errors.New("bad request"), reporter, stderr)
			want := []string{
				"stderr: gomad: choices-seed=0 choices-profile= choices-records=0 choices-decisions=0 choices-branching=0 choices-runnable=0 choices-select-poll=0 choices-select-result=0 choices-sha256= choices-tape-sha256= choices-terminal=\n",
				"stdout: {\"schema\":\"gomad3.explore-event/v3\",\"type\":\"error\",\"classification\":\"invalid_input\",\"message\":\"bad request\"}\n",
				"stderr: write explore event: reporter write failed\ndiagnostic write failed\n",
			}
			if status != 3 || !reflect.DeepEqual(writes, want) {
				t.Fatalf("status = %d, writes = %#v, want status 3 and writes %#v", status, writes, want)
			}
		})
	}
}
