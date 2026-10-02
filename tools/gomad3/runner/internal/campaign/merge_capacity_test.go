package campaign

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestValidateMergedArtifactCapacityReportsSaturatedTotalOverflow(t *testing.T) {
	maximum := record.Uint64String(^uint64(0))
	err := validateMergedArtifactCapacity(ArtifactCapacityPlan{
		FailureArtifacts: 1,
		FailureBytes:     maximum,
		SuccessArtifacts: 1,
		SuccessBytes:     maximum,
		TotalBytes:       maximum,
	}, 1, ^uint64(0), 1, 1)
	var capacityErr *ArtifactCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != ArtifactLimitTotalBytes || capacityErr.Required != ^uint64(0) || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("validateMergedArtifactCapacity() error = %#v", err)
	}
}

func TestCheckedMergedEvidenceBytesReportsTypedOverflow(t *testing.T) {
	_, err := checkedMergedEvidenceBytes(^uint64(0), 1, ArtifactLimitFailureBytes, 100)
	var capacityErr *ArtifactCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != ArtifactLimitFailureBytes || capacityErr.Required != ^uint64(0) || capacityErr.Maximum != 100 {
		t.Fatalf("checkedMergedEvidenceBytes() error = %#v", err)
	}
}

func TestOpenMergedCampaignRejectsNoncanonicalAndInvalidRecords(t *testing.T) {
	canonical, err := canonicaljson.CanonicalJSON(MergedCampaignRecord{Schema: MergedCampaignSchema})
	if err != nil {
		t.Fatal(err)
	}
	invalidIdentity, err := canonicaljson.CanonicalJSON(MergedCampaignRecord{Schema: "gomad3.merged-campaign/v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		contents string
		want     string
	}{
		{name: "incomplete object", contents: `{"schema":"gomad3.merged-campaign/v1"}`, want: "JSON is not canonical"},
		{name: "trailing whitespace", contents: string(canonical) + "\n", want: "JSON is not canonical"},
		{name: "trailing data", contents: string(canonical) + "{}", want: "unexpected trailing JSON token {"},
		{name: "unknown field", contents: `{"extra":true}`, want: `decode JSON: json: unknown field "extra"`},
		{name: "malformed", contents: `{"schema":`, want: "decode JSON token: EOF"},
		{name: "invalid schema", contents: string(invalidIdentity), want: "merged campaign record is invalid"},
		{name: "invalid identity", contents: string(canonical), want: "merged campaign record is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "merged")
			if err := os.MkdirAll(filepath.Join(path, "executions"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "merge.json"), []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenMergedCampaign(path); err == nil || err.Error() != test.want {
				t.Fatalf("OpenMergedCampaign() error = %v, want %s", err, test.want)
			}
		})
	}
}

const (
	mergeTestSelection = "100-103"
	mergeTestOrdinals  = 4
	mergeTestShards    = 2
)

var mergeTestPlan = record.HashBytes([]byte("merge test plan"))

// mergeTestTarget is large against the rest of an artifact, as a real target is.
var mergeTestTarget = bytes.Repeat([]byte("target bytes "), 8<<10)

// publishMergeShard runs one shard of the test plan in the artifacts root and
// retains every execution as a success artifact of one target. It returns the
// campaign path and the stored bytes of each artifact.
func publishMergeShard(t *testing.T, artifacts string, index uint64) (string, []uint64) {
	t.Helper()
	id := fmt.Sprintf("shard-%d", index)
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: artifacts, CampaignID: id, PlanSHA256: mergeTestPlan, Shard: &CampaignShard{Index: record.Uint64String(index), Count: mergeTestShards},
		Selection: mergeTestSelection, SelectionCount: mergeTestOrdinals,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	for _, step := range []func() error{journal.BeginPreparation, journal.CompletePreparation, journal.StartExecutions} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	var stored []uint64
	var total uint64
	for ordinal := index; ordinal < mergeTestOrdinals; ordinal += mergeTestShards {
		seed := 100 + ordinal
		run, err := journal.BeginExecution(ordinal, seed)
		if err != nil {
			t.Fatal(err)
		}
		for _, state := range []ExecutionState{ExecutionStarting, ExecutionExited, ExecutionCaptured, ExecutionClassified} {
			if err := run.Transition(state); err != nil {
				t.Fatal(err)
			}
		}
		input := campaignArtifactInput(t)
		if err := os.WriteFile(input.TargetPath, mergeTestTarget, 0o700); err != nil {
			t.Fatal(err)
		}
		exitCode := record.Uint64String(0)
		manifest := &input.Manifest
		manifest.Target.SHA256, manifest.Target.Size = record.HashBytes(mergeTestTarget), record.Uint64String(len(mergeTestTarget))
		manifest.ArtifactKind, manifest.CampaignID, manifest.SelectionOrdinal, manifest.Seed = record.ArtifactSuccess, id, record.Uint64String(ordinal), record.Uint64String(seed)
		manifest.Environment[1].Value = fmt.Sprint(seed)
		// Evidence of one outcome signature merges into one artifact, so each
		// execution prints its own output.
		input.Stdout = []byte(fmt.Sprintf("stdout %d", seed))
		manifest.Streams.Stdout = record.Stream{FullSHA256: record.HashBytes(input.Stdout), TotalBytes: record.Uint64String(len(input.Stdout)), RetainedBytes: record.Uint64String(len(input.Stdout))}
		manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
		published, err := artifact.PublishArtifact(artifact.Store{Root: journal.SuccessesPath(), Key: artifact.StoreKeyRecord, TargetPool: artifact.TargetPool(artifacts)}, input)
		if err != nil {
			t.Fatal(err)
		}
		reference, err := filepath.Rel(journal.Path(), published.Path)
		if err != nil {
			t.Fatal(err)
		}
		reference = filepath.ToSlash(reference)
		bytes := record.Uint64String(published.StoredBytes)
		if err := journal.AppendExecution(ExecutionRecord{
			SelectionOrdinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed), Domain: "success", Reason: "success", Termination: "exit", ElapsedNanos: 1,
			SuccessArtifact: &reference, SuccessArtifactBytes: &bytes,
		}); err != nil {
			t.Fatal(err)
		}
		if err := run.Complete(); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, published.StoredBytes)
		total += published.StoredBytes
	}
	count := uint64(len(stored))
	if err := journal.Publish(CampaignSummary{Attempted: count, Succeeded: count, RetainedSuccesses: count, RetainedSuccessBytes: total, StopReason: "seeds_exhausted"}); err != nil {
		t.Fatal(err)
	}
	return journal.Path(), stored
}

func mergeTestSpec(t *testing.T, shards []string, successBytes uint64, partial bool) MergeSpec {
	t.Helper()
	limits, err := normalizeExecutionJournalLimits(CampaignConfig{SelectionCount: mergeTestOrdinals})
	if err != nil {
		t.Fatal(err)
	}
	const failureBytes = 1 << 30
	return MergeSpec{
		Output: filepath.Join(t.TempDir(), "merged"), PlanSHA256: mergeTestPlan, Selection: mergeTestSelection, SelectionCount: mergeTestOrdinals,
		Journal: recordExecutionJournalLimits(limits), Partial: partial, ShardPaths: shards,
		SeedAt: func(ordinal uint64) (uint64, bool) { return 100 + ordinal, ordinal < mergeTestOrdinals },
		Artifacts: ArtifactCapacityPlan{
			FailureArtifacts: mergeTestOrdinals, FailureBytes: failureBytes, SuccessArtifacts: mergeTestOrdinals, SuccessBytes: record.Uint64String(successBytes),
			TotalBytes: record.Uint64String(failureBytes + successBytes), TranscriptBytes: 1,
			FailureOutcome: CapacityInfrastructureFailure, SuccessOutcome: CapacityInfrastructureFailure,
		},
	}
}

func TestMergeCountsTheTargetOfItsSuccessEvidenceOnce(t *testing.T) {
	sameRoot := t.TempDir()
	for name, roots := range map[string][mergeTestShards]string{
		"shards of one artifacts root":  {sameRoot, sameRoot},
		"shards of two artifacts roots": {t.TempDir(), t.TempDir()},
	} {
		t.Run(name, func(t *testing.T) {
			var shards, targets []string
			var full uint64
			for index, root := range roots {
				path, stored := publishMergeShard(t, root, uint64(index))
				shards = append(shards, path)
				for _, bytes := range stored {
					full += bytes
				}
				shardTargets, err := filepath.Glob(filepath.Join(path, "successes", "*", "target"))
				if err != nil {
					t.Fatal(err)
				}
				targets = append(targets, shardTargets...)
			}
			once := full - (mergeTestOrdinals-1)*uint64(len(mergeTestTarget))
			if once >= full/2 {
				t.Fatalf("evidence with one target is %d of %d bytes: the limits below would not tell the two sums apart", once, full)
			}

			_, err := MergeCampaigns(context.Background(), mergeTestSpec(t, shards, once-1, false))
			var capacityErr *ArtifactCapacityError
			if !errors.As(err, &capacityErr) || capacityErr.Limit != ArtifactLimitSuccessBytes || capacityErr.Required != once {
				t.Fatalf("MergeCampaigns() under a limit one byte short = %#v, want %d success bytes required", err, once)
			}
			merged, err := MergeCampaigns(context.Background(), mergeTestSpec(t, shards, once, false))
			if err != nil {
				t.Fatal(err)
			}
			if merged.Record.RetainedEvidence != mergeTestOrdinals || uint64(merged.Record.EvidenceBytes) != once || uint64(merged.Record.RetainedSuccessBytes) != full {
				t.Fatalf("merged evidence = %d artifacts in %d bytes, executions report %d; want %d in %d, executions %d", merged.Record.RetainedEvidence, merged.Record.EvidenceBytes, merged.Record.RetainedSuccessBytes, mergeTestOrdinals, once, full)
			}
			opened, err := OpenMergedCampaign(merged.Path)
			if err != nil {
				t.Fatalf("OpenMergedCampaign() rejected the record MergeCampaigns() published: %v", err)
			}
			if uint64(opened.Record.EvidenceBytes) != once {
				t.Fatalf("reopened evidence bytes = %d, want %d", opened.Record.EvidenceBytes, once)
			}
			partial, err := MergeCampaigns(context.Background(), mergeTestSpec(t, shards[:1], once, true))
			if err != nil || !partial.Record.Partial {
				t.Fatalf("partial merge = %#v, %v", partial.Record, err)
			}

			// Merging reads the shard stores and changes nothing in them: each
			// artifacts root still holds the one copy its shards share.
			distinct := map[string]struct{}{}
			for _, root := range roots {
				distinct[root] = struct{}{}
			}
			if copies := distinctTargetFiles(t, targets); len(targets) != mergeTestOrdinals || copies != len(distinct) {
				t.Fatalf("%d success targets are %d files, want one per artifacts root (%d)", len(targets), copies, len(distinct))
			}
		})
	}
}

func distinctTargetFiles(t *testing.T, paths []string) int {
	t.Helper()
	var distinct []os.FileInfo
next:
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, seen := range distinct {
			if os.SameFile(seen, info) {
				continue next
			}
		}
		distinct = append(distinct, info)
	}
	return len(distinct)
}

func TestOpenMergedCampaignCountsEvidenceWithoutATargetInFull(t *testing.T) {
	digest := record.HashBytes([]byte("evidence"))
	target := record.HashBytes([]byte("target"))
	for name, test := range map[string]struct {
		evidence []MergedEvidence
		want     uint64
		wantErr  bool
	}{
		"record written before targets were shared": {evidence: []MergedEvidence{{StoredBytes: 110}, {StoredBytes: 120}}, want: 230},
		"one target":                       {evidence: []MergedEvidence{{StoredBytes: 110, TargetSHA256: target, TargetBytes: 100}, {StoredBytes: 120, TargetSHA256: target, TargetBytes: 100}}, want: 130},
		"target as large as its artifact":  {evidence: []MergedEvidence{{StoredBytes: 100, TargetSHA256: target, TargetBytes: 100}}, wantErr: true},
		"target bytes without an identity": {evidence: []MergedEvidence{{StoredBytes: 110, TargetBytes: 100}}, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			var successes mergedEvidenceBytes
			var err error
			for _, evidence := range test.evidence {
				evidence.SHA256 = digest
				if err = successes.add(evidence, ArtifactLimitSuccessBytes, 1<<20); err != nil {
					break
				}
			}
			if (err != nil) != test.wantErr || !test.wantErr && successes.retained.Total() != test.want {
				t.Fatalf("success evidence bytes = %d, %v, want %d (error %t)", successes.retained.Total(), err, test.want, test.wantErr)
			}
		})
	}
}

// testdata/pre-target-evidence-merged is a merged record the Runner published
// at b5b498004, before merged evidence named its target: publishMergeShard's
// four successes of one target, merged under a success-byte limit of exactly
// their stored bytes.
func TestOpenMergedCampaignOpensARecordWrittenBeforeEvidenceNamedItsTarget(t *testing.T) {
	opened, err := OpenMergedCampaign(copyRetainedRecords(t, "pre-target-evidence-merged"))
	if err != nil {
		t.Fatalf("OpenMergedCampaign() rejected the retained record: %v", err)
	}
	var stored uint64
	for _, run := range opened.Executions {
		if run.Evidence == nil || run.Evidence.sharedTarget() != (artifact.SharedTarget{}) {
			t.Fatalf("retained execution evidence = %#v, want evidence that names no target", run.Evidence)
		}
		stored += uint64(run.Evidence.StoredBytes)
	}
	// Every artifact counts in full, so the record sits exactly at its limit.
	got := [3]uint64{uint64(len(opened.Executions)), uint64(opened.Record.EvidenceBytes), uint64(opened.Record.Artifacts.SuccessBytes)}
	if want := [3]uint64{mergeTestOrdinals, stored, stored}; got != want {
		t.Fatalf("retained record executions, evidence bytes, and success-byte limit = %d, want %d", got, want)
	}
}

// The executions of a merged campaign may report more success bytes than the
// success-byte limit, because the limit bounds the retained evidence. Every
// other limit of the record is still checked with the value the record states.
func TestOpenMergedCampaignChecksItsOtherLimitsWhenExecutionsReportMoreSuccessBytes(t *testing.T) {
	root := t.TempDir()
	var shards []string
	var full uint64
	for index := uint64(0); index < mergeTestShards; index++ {
		path, stored := publishMergeShard(t, root, index)
		shards = append(shards, path)
		for _, bytes := range stored {
			full += bytes
		}
	}
	once := full - (mergeTestOrdinals-1)*uint64(len(mergeTestTarget))
	merged, err := MergeCampaigns(context.Background(), mergeTestSpec(t, shards, once, false))
	if err != nil {
		t.Fatal(err)
	}
	if uint64(merged.Record.RetainedSuccessBytes) <= uint64(merged.Record.Artifacts.SuccessBytes) {
		t.Fatalf("executions report %d success bytes under a limit of %d: the record would not need the bound skipped", merged.Record.RetainedSuccessBytes, merged.Record.Artifacts.SuccessBytes)
	}
	if _, err := OpenMergedCampaign(merged.Path); err != nil {
		t.Fatalf("OpenMergedCampaign() rejected the record before any limit was changed: %v", err)
	}
	for name, change := range map[string]func(*ArtifactCapacityPlan){
		"total bytes that are not the sum of the byte limits":  func(limits *ArtifactCapacityPlan) { limits.TotalBytes++ },
		"fewer success artifacts than the executions retained": func(limits *ArtifactCapacityPlan) { limits.SuccessArtifacts = mergeTestOrdinals - 1 },
		"no transcript bytes": func(limits *ArtifactCapacityPlan) { limits.TranscriptBytes = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := merged.Record
			change(&changed.Artifacts)
			manifest, err := canonicaljson.CanonicalJSON(changed)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(merged.Path, "merge.json"), manifest, 0o600); err != nil {
				t.Fatal(err)
			}
			const want = "validate merged executions: campaign artifact capacity is invalid"
			if _, err := OpenMergedCampaign(merged.Path); err == nil || err.Error() != want {
				t.Fatalf("OpenMergedCampaign() error = %v, want %s", err, want)
			}
		})
	}
}
