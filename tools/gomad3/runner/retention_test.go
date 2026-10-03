package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestDecideSuccessRetentionJudgesNoveltyTranscriptAndBounds(t *testing.T) {
	const (
		incomplete = "success_artifact_publication: retained success requires a complete I/O transcript for exact replay"
		exhausted  = "success_retention_capacity: successful-execution retention capacity is exhausted"
	)
	for _, test := range []struct {
		name          string
		policy        KeepSuccesses
		probes        []string
		choices       []string
		incomplete    bool
		retained      uint64
		retainedBytes uint64
		want          successRetention
		failure       string
	}{
		{name: "discard", policy: KeepSuccessesNone, probes: []string{"new"}, choices: []string{"new"}},
		{name: "unset policy", probes: []string{"new"}, choices: []string{"new"}},
		{name: "discard an unreplayable success beyond the bounds", policy: KeepSuccessesNone, incomplete: true, retained: 9, retainedBytes: 900},
		{name: "all", policy: KeepSuccessesAll, probes: []string{"seen"}, retained: 2, retainedBytes: 40, want: successRetention{retain: true, maximumBytes: 60}},
		{name: "all needs a complete transcript", policy: KeepSuccessesAll, incomplete: true, want: successRetention{retain: true}, failure: incomplete},
		{name: "novel probe", policy: KeepSuccessesNovel, probes: []string{"seen", "new"}, choices: []string{"seen"}, want: successRetention{retain: true, novelProbes: []string{"new"}, novelChoices: []string{}, maximumBytes: 100}},
		{name: "novel choice", policy: KeepSuccessesNovel, probes: []string{"seen"}, choices: []string{"seen", "new"}, want: successRetention{retain: true, novelProbes: []string{}, novelChoices: []string{"new"}, maximumBytes: 100}},
		{
			name: "probes and choices are judged against their own committed sets", policy: KeepSuccessesNovel, probes: []string{"seen choice"}, choices: []string{"seen probe"},
			want: successRetention{retain: true, novelProbes: []string{"seen choice"}, novelChoices: []string{"seen probe"}, maximumBytes: 100},
		},
		{name: "novel values keep their observed order", policy: KeepSuccessesNovel, probes: []string{"z", "seen", "a"}, want: successRetention{retain: true, novelProbes: []string{"z", "a"}, novelChoices: []string{}, maximumBytes: 100}},
		{
			name: "nothing novel is not kept, whatever the transcript and the bounds", policy: KeepSuccessesNovel, probes: []string{"seen"}, choices: []string{"seen"},
			incomplete: true, retained: 9, retainedBytes: 900, want: successRetention{novelProbes: []string{}, novelChoices: []string{}},
		},
		{name: "novel needs a complete transcript", policy: KeepSuccessesNovel, probes: []string{"new"}, incomplete: true, want: successRetention{retain: true, novelProbes: []string{"new"}, novelChoices: []string{}}, failure: incomplete},
		{name: "count used up", policy: KeepSuccessesAll, retained: 3, want: successRetention{retain: true}, failure: exhausted},
		{name: "count exceeded", policy: KeepSuccessesAll, retained: 4, want: successRetention{retain: true}, failure: exhausted},
		{name: "bytes used up", policy: KeepSuccessesAll, retainedBytes: 100, want: successRetention{retain: true}, failure: exhausted},
		{name: "bytes exceeded", policy: KeepSuccessesAll, retainedBytes: 101, want: successRetention{retain: true}, failure: exhausted},
		{name: "last count and byte", policy: KeepSuccessesAll, retained: 2, retainedBytes: 99, want: successRetention{retain: true, maximumBytes: 1}},
		{name: "transcript before bounds", policy: KeepSuccessesAll, incomplete: true, retained: 3, retainedBytes: 100, want: successRetention{retain: true}, failure: incomplete},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := CampaignSpec{KeepSuccesses: test.policy, SuccessArtifactLimit: 3, SuccessBytesLimit: 100}
			assessed := completedExecution{coverage: deterministicio.SemanticCoverage{Probes: test.probes}, choiceFeatures: test.choices}
			seenProbes, seenChoices := map[string]struct{}{"seen": {}, "seen probe": {}}, map[string]struct{}{"seen": {}, "seen choice": {}}
			decision, hostError := decideSuccessRetention(campaignRequestFromSpec(config), assessed, !test.incomplete, seenProbes, seenChoices, test.retained, test.retainedBytes)
			failure := ""
			if hostError != nil {
				failure = hostError.Reason + ": " + hostError.Err.Error()
			}
			if !reflect.DeepEqual(decision, test.want) || failure != test.failure {
				t.Fatalf("decideSuccessRetention() = %#v, %q, want %#v, %q", decision, failure, test.want, test.failure)
			}
			if len(seenProbes) != 2 || len(seenChoices) != 2 {
				t.Fatalf("decideSuccessRetention() advanced the committed novelty to %v, %v", seenProbes, seenChoices)
			}
		})
	}
}

// A campaign's success-byte limit counts every kept success in full, where a
// corpus and a merged campaign count a shared target once
// (artifact.RetainedBytes says why): two successes that link to one pool entry
// do not fit a limit that is half a target short of their stored bytes.
func TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit(t *testing.T) {
	// The target is large against the bytes a manifest varies by between runs.
	targetBytes := bytes.Repeat([]byte("target bytes "), 8<<10)
	run := func(limit uint64) (CampaignResult, error) {
		preparer := newFakePreparer(t)
		prepared := &preparer.prepared
		if err := os.Chmod(prepared.Path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(prepared.Path, targetBytes, 0o700); err != nil {
			t.Fatal(err)
		}
		prepared.SHA256, prepared.Size = fmt.Sprintf("sha256:%x", sha256.Sum256(targetBytes)), uint64(len(targetBytes))
		config, configDependencies := testConfig(t, preparer, &fakeExecutor{result: func(seed uint64) execution.Result {
			result := processResult(0, fmt.Sprint(seed), "")
			result.IOTranscript = completeEmptyTranscript()
			return result
		}}, "1-2", PolicyAll, 1)
		config.KeepSuccesses = KeepSuccessesAll
		config.SuccessArtifactLimit = 2
		config.SuccessBytesLimit = limit
		return exploreWith(context.Background(), config, configDependencies)
	}
	measured, err := run(64 << 20)
	if err != nil || len(measured.SuccessArtifacts) != 2 || measured.SuccessArtifacts[0] == measured.SuccessArtifacts[1] {
		t.Fatalf("summary = %#v, error = %v, want two success artifacts", measured, err)
	}
	var targets [2]os.FileInfo
	for index, path := range measured.SuccessArtifacts {
		if targets[index], err = os.Lstat(filepath.Join(path, "target")); err != nil {
			t.Fatal(err)
		}
	}
	if !os.SameFile(targets[0], targets[1]) {
		t.Fatal("the two kept successes do not share one target file")
	}
	summary, err := run(measured.RetainedSuccessBytes - uint64(len(targetBytes))/2)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "success_retention_capacity" || summary.RetainedSuccesses != 1 {
		t.Fatalf("summary = %#v, error = %v, want a capacity failure at the second success", summary, err)
	}
}

func TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts(t *testing.T) {
	config, configDependencies := testConfig(t, newFakePreparer(t), &fakeExecutor{result: func(uint64) execution.Result {
		result := processResult(0, "same output", "")
		result.IOTranscript = completeEmptyTranscript()
		return result
	}}, "1-2", PolicyAll, 1)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(summary.CampaignPath, "successes"))
	if err != nil {
		t.Fatal(err)
	}
	if summary.RetainedSuccesses != 2 || len(summary.SuccessArtifacts) != 2 || len(entries) != 2 || len(opened.Executions) != 2 || opened.Record.RetainedSuccesses != 2 {
		t.Fatalf("summary=%#v, executions=%#v, stored=%d", summary, opened.Executions, len(entries))
	}
	var signature record.SHA256
	var storedBytes uint64
	for index, path := range summary.SuccessArtifacts {
		if opened.Executions[index].SuccessArtifact == nil || filepath.Join(summary.CampaignPath, *opened.Executions[index].SuccessArtifact) != path {
			t.Fatalf("execution %d references a different success artifact: %#v", index, opened.Executions[index])
		}
		retained, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest := retained.Manifest()
		storedBytes += retained.StoredBytes()
		if err := retained.Close(); err != nil {
			t.Fatal(err)
		}
		if manifest.Seed != record.Uint64String(index+1) || manifest.Seed != opened.Executions[index].Seed || manifest.Streams.Stdout.FullSHA256 != record.HashBytes([]byte("same output")) {
			t.Fatalf("retained success %d = %#v", index, manifest)
		}
		if index == 0 {
			signature = manifest.Outcome.FailureSignature
		} else if manifest.Outcome.FailureSignature != signature || summary.SuccessArtifacts[0] == path {
			t.Fatalf("successes with one signature collapsed: %v", summary.SuccessArtifacts)
		}
	}
	if storedBytes != summary.RetainedSuccessBytes || record.Uint64String(storedBytes) != opened.Record.RetainedSuccessBytes {
		t.Fatalf("stored bytes=%d, summary=%d, record=%d", storedBytes, summary.RetainedSuccessBytes, opened.Record.RetainedSuccessBytes)
	}
}

func TestSuccessRetentionAnnotatesTheJournalRecord(t *testing.T) {
	relative, bytes := "successes/sha256-00", record.Uint64String(7)
	for _, test := range []struct {
		name     string
		decision successRetention
		want     campaign.ExecutionRecord
	}{
		{name: "all", decision: successRetention{retain: true}, want: campaign.ExecutionRecord{Domain: "success", SuccessArtifact: &relative, SuccessArtifactBytes: &bytes}},
		{
			name: "novel", decision: successRetention{retain: true, novelProbes: []string{"probe"}, novelChoices: []string{}},
			want: campaign.ExecutionRecord{Domain: "success", SuccessArtifact: &relative, SuccessArtifactBytes: &bytes, NovelSemanticProbes: []string{"probe"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			run := campaign.ExecutionRecord{Domain: "success"}
			test.decision.annotate(&run, relative, 7)
			if len(test.decision.novelProbes) != 0 {
				test.decision.novelProbes[0] = "changed after the record was annotated"
			}
			if !reflect.DeepEqual(run, test.want) {
				t.Fatalf("annotated record = %#v, want %#v", run, test.want)
			}
		})
	}
}

func TestSuccessPublicationFailureSeparatesCapacityFromPublication(t *testing.T) {
	plain := errors.New("store is unavailable")
	capacity := fmt.Errorf("publish: %w", &artifact.CapacityError{Required: 9, Maximum: 8})
	for _, test := range []struct {
		err  error
		want HostError
	}{
		{err: plain, want: HostError{Reason: "success_artifact_publication", Err: plain}},
		{err: capacity, want: HostError{Reason: "success_retention_capacity", Err: capacity}},
	} {
		if failure := successPublicationFailure(test.err); *failure != test.want {
			t.Fatalf("successPublicationFailure(%v) = %#v, want %#v", test.err, *failure, test.want)
		}
	}
}

func TestExecutionArtifactInputCarriesTheCapturedEvidence(t *testing.T) {
	manifest := record.ExecutionRecord{CampaignID: "campaign", Seed: 7}
	mounts := &readonlymount.CapturedInputs{}
	result := processResult(0, "stdout", "stderr")
	result.IOTranscript = deterministicio.Transcript{Bytes: []byte("transcript")}
	result.ChoiceTrace.Trace.Bytes = []byte("choices")
	result.SimulationRecords = [][]byte{[]byte("simulation record")}
	worldBundle := noneWorldBundle()
	want := artifact.ArtifactInput{
		Manifest: manifest, TargetPath: "/prepared/target", Stdout: []byte("stdout"), Stderr: []byte("stderr"),
		IOTranscript: []byte("transcript"), ChoiceTrace: []byte("choices"), ReadOnlyMounts: mounts, World: worldBundle.Payloads,
	}
	input := executionArtifactInput(manifest, target.Prepared{Path: "/prepared/target"}, result, mounts, worldBundle)
	if !reflect.DeepEqual(input, want) || input.ReadOnlyMounts != mounts {
		t.Fatalf("executionArtifactInput() = %#v, want %#v", input, want)
	}
}
