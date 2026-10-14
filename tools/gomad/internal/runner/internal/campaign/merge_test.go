package campaign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/logstore"
)

const frontierMergeParallel = 2

// frontierMergeShard is one shard of the merge test plan run through the
// admission frontier.
type frontierMergeShard struct {
	path       string
	failure    string
	success    string
	diagnostic string
}

// publishFrontierMergeShard runs one shard of the merge test plan through the
// admission frontier: it admits both of its ordinals at once, records their
// receipts out of order and commits their outcomes in order. Its lower
// ordinal fails with the failure every shard shares; its higher ordinal
// succeeds, is retained, and adopts the diagnostic sidecar its receipt
// published. With publish unset the shard stops before its publication.
func publishFrontierMergeShard(t *testing.T, artifacts string, index uint64, publish bool) frontierMergeShard {
	t.Helper()
	id := fmt.Sprintf("frontier-shard-%d", index)
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: artifacts, CampaignID: id, PlanSHA256: mergeTestPlan, Shard: &CampaignShard{Index: record.Uint64String(index), Count: mergeTestShards},
		Selection: mergeTestSelection, SelectionCount: mergeTestOrdinals, Parallel: frontierMergeParallel,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, step := range []func() error{journal.BeginPreparation, journal.CompletePreparation, journal.StartExecutions} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	failing, passing := index, index+mergeTestShards
	if err := journal.Admit([]SeedJob{{Ordinal: failing, Seed: 100 + failing}, {Ordinal: passing, Seed: 100 + passing}}); err != nil {
		t.Fatal(err)
	}

	failureInput := campaignArtifactInput(t)
	failureInput.Manifest.CampaignID, failureInput.Manifest.SelectionOrdinal, failureInput.Manifest.Seed = id, record.Uint64String(failing), record.Uint64String(100+failing)
	failureInput.Manifest.Environment = slices.Clone(failureInput.Manifest.Environment)
	failureInput.Manifest.Environment[1].Value = fmt.Sprint(100 + failing)
	publishedFailure, err := artifact.PublishArtifact(artifact.Store{Root: journal.FailuresPath()}, failureInput)
	if err != nil {
		t.Fatal(err)
	}
	failureReference := "failures/" + filepath.Base(publishedFailure.Path)
	signature := publishedFailure.Manifest.Outcome.FailureSignature
	failure := ExecutionRecord{
		SelectionOrdinal: record.Uint64String(failing), Seed: record.Uint64String(100 + failing), Domain: "target",
		Reason: publishedFailure.Manifest.Outcome.Reason, Termination: publishedFailure.Manifest.Outcome.Termination,
		FailureSignature: &signature, Artifact: &failureReference,
	}

	successInput := campaignArtifactInput(t)
	if err := os.WriteFile(successInput.TargetPath, mergeTestTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	exitCode := record.Uint64String(0)
	manifest := &successInput.Manifest
	manifest.Target.SHA256, manifest.Target.Size = record.HashBytes(mergeTestTarget), record.Uint64String(len(mergeTestTarget))
	manifest.ArtifactKind, manifest.CampaignID, manifest.SelectionOrdinal, manifest.Seed = record.ArtifactSuccess, id, record.Uint64String(passing), record.Uint64String(100+passing)
	manifest.Environment = slices.Clone(manifest.Environment)
	manifest.Environment[1].Value = fmt.Sprint(100 + passing)
	successInput.Stdout = []byte(fmt.Sprintf("stdout %d", 100+passing))
	manifest.Streams.Stdout = record.Stream{FullSHA256: record.HashBytes(successInput.Stdout), TotalBytes: record.Uint64String(len(successInput.Stdout)), RetainedBytes: record.Uint64String(len(successInput.Stdout))}
	manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
	publishedSuccess, err := artifact.PublishArtifact(artifact.Store{Root: journal.SuccessesPath(), Key: artifact.StoreKeyRecord, TargetPool: artifact.TargetPool(artifacts)}, successInput)
	if err != nil {
		t.Fatal(err)
	}
	successReference := "successes/" + filepath.Base(publishedSuccess.Path)
	diagnostic, err := artifact.PublishOwnedFile(context.Background(), filepath.Join(journal.Path(), "diagnostics"), fmt.Sprintf("%020d-trace.bin", passing), []byte("trace"))
	if err != nil {
		t.Fatal(err)
	}

	observation := successExecution(passing, 100+passing)
	if err := journal.RecordReceipt(Receipt{Observation: observation, Artifacts: []string{successReference}, Diagnostics: []artifact.FileReference{diagnostic}}); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordReceipt(Receipt{Observation: failure, Artifacts: []string{failureReference}}); err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitOutcome(failure, nil); err != nil {
		t.Fatal(err)
	}
	success := observation
	successBytes := record.Uint64String(publishedSuccess.StoredBytes)
	success.SuccessArtifact, success.SuccessArtifactBytes = &successReference, &successBytes
	if err := journal.CommitOutcome(success, nil); err != nil {
		t.Fatal(err)
	}
	if publish {
		if err := journal.Publish(CampaignSummary{
			Attempted: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1, FailureSignatures: []record.SHA256{signature},
			RetainedSuccesses: 1, RetainedSuccessBytes: publishedSuccess.StoredBytes, StopReason: "seeds_exhausted",
		}); err != nil {
			t.Fatal(err)
		}
	}
	return frontierMergeShard{
		path: journal.Path(), failure: filepath.Join(journal.Path(), failureReference), success: filepath.Join(journal.Path(), successReference),
		diagnostic: filepath.Join(journal.Path(), "diagnostics", diagnostic.Name),
	}
}

// frontierMergeShards publishes both shards of the merge test plan in one
// artifacts root, so their success evidence shares one target pool.
func frontierMergeShards(t *testing.T) (string, []frontierMergeShard, []string) {
	t.Helper()
	root := t.TempDir()
	shards := make([]frontierMergeShard, mergeTestShards)
	paths := make([]string, mergeTestShards)
	for index := range shards {
		shards[index] = publishFrontierMergeShard(t, root, uint64(index), true)
		paths[index] = shards[index].path
	}
	return root, shards, paths
}

func frontierMergeSpec(t *testing.T, paths []string, output string) MergeSpec {
	t.Helper()
	spec := mergeTestSpecWith(t, paths, 1<<30, false, frontierMergeParallel)
	if output != "" {
		spec.Output = output
	}
	return spec
}

// treeSnapshot is every entry under a root: its mode and, for a regular file,
// its content digest.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(contents)
			value += " " + hex.EncodeToString(sum[:])
		}
		snapshot[relative] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func assertTreeUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	if after := treeSnapshot(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("shard tree changed:\nbefore %v\nafter  %v", before, after)
	}
}

func TestMergeConcatenatesSourceScopesAndReplaysTheirCombinedTotals(t *testing.T) {
	root, _, paths := frontierMergeShards(t)
	before := treeSnapshot(t, root)
	merged, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, paths, ""))
	if err != nil {
		t.Fatal(err)
	}
	assertTreeUnchanged(t, root, before)

	// The aggregate is its envelope, each shard's own records in its scope,
	// then one evidence record per execution that retains evidence.
	aggregate := mergedLogRecords(t, merged.Path)
	want := []mergedLogRecord{aggregate[0]}
	var combined CampaignRecord
	for index, path := range paths {
		source := mergedLogRecords(t, path)
		kinds := make([]logstore.Kind, len(source))
		for position, entry := range source {
			kinds[position] = entry.kind
		}
		if !slices.Contains(kinds, kindAdmission) || !slices.Contains(kinds, kindReceipt) {
			t.Fatalf("shard %d records %v hold no admission and receipt", index, kinds)
		}
		scope, err := logstore.EncodeJSON(mergedSourceRecord{Index: record.Uint64String(index)})
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, mergedLogRecord{kind: kindMergedSource, payload: scope})
		want = append(want, source...)
		opened, err := OpenCampaign(path)
		if err != nil {
			t.Fatal(err)
		}
		combined.Attempted += opened.Record.Attempted
		combined.Succeeded += opened.Record.Succeeded
		combined.Failures += opened.Record.Failures
		combined.RetainedSuccesses += opened.Record.RetainedSuccesses
	}
	if len(aggregate) != len(want)+mergeTestOrdinals || !reflect.DeepEqual(aggregate[:len(want)], want) {
		t.Fatalf("aggregate records do not concatenate the source scopes")
	}
	for _, entry := range aggregate[len(want):] {
		if entry.kind != kindMergedEvidence {
			t.Fatalf("aggregate ends in record kind %d, want only evidence", entry.kind)
		}
	}

	opened, err := OpenMergedCampaign(merged.Path)
	if err != nil {
		t.Fatal(err)
	}
	result := opened.Record
	if result.Attempted != combined.Attempted || result.Succeeded != combined.Succeeded || result.Failures != combined.Failures || result.RetainedSuccesses != combined.RetainedSuccesses ||
		result.Attempted != mergeTestOrdinals || result.DistinctFailures != 1 || result.RetainedEvidence != 3 || result.Partial || len(result.Shards) != mergeTestShards {
		t.Fatalf("merged totals = %#v, shards combined %#v", result, combined)
	}
	if !reflect.DeepEqual(opened.Record, merged.Record) || !reflect.DeepEqual(opened.Executions, merged.Executions) {
		t.Fatal("reopened aggregate differs from the merge result")
	}
	for index, shard := range result.Shards {
		if shard.Path != paths[index] || shard.Records == 0 || !validRecordSHA256(shard.LogSHA256) {
			t.Fatalf("shard %d provenance = %#v", index, shard)
		}
	}
	if err := RefuseLegacyLayout(merged.Path); err != nil {
		t.Fatalf("RefuseLegacyLayout() = %v", err)
	}
	inspected, report, err := InspectMergedCampaign(merged.Path)
	if err != nil || !reflect.DeepEqual(inspected.Record, merged.Record) || report.Tail != logstore.TailNone || report.Records == 0 || report.ValidBytes != report.SnapshotBytes || report.RepairedBytes != 0 {
		t.Fatalf("InspectMergedCampaign() = %#v, %v", report, err)
	}
	assertTreeUnchanged(t, root, before)
}

func TestMergePreflightFailsBeforeCreatingAnyOutput(t *testing.T) {
	for _, test := range []struct {
		name string
		// prepare damages the shards or the spec before the merge.
		prepare func(t *testing.T, shards []frontierMergeShard, spec *MergeSpec)
		check   func(error) bool
		want    string
	}{
		{name: "another plan", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.PlanSHA256 = record.HashBytes([]byte("another plan"))
		}, want: "does not match the campaign plan"},
		{name: "another selection", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.Selection = "200-203"
		}, want: "does not match the campaign plan"},
		{name: "another seed at an ordinal", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.SeedAt = func(ordinal uint64) (uint64, bool) { return 200 + ordinal, ordinal < mergeTestOrdinals }
		}, want: "does not match the campaign plan"},
		{name: "other journal limits", prepare: func(t *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.Journal = mergeTestSpec(t, spec.ShardPaths, 1, false).Journal
		}, want: "does not match the campaign plan"},
		{name: "duplicated shard", prepare: func(_ *testing.T, shards []frontierMergeShard, spec *MergeSpec) {
			spec.ShardPaths = []string{shards[0].path, shards[0].path, shards[1].path}
		}, want: "campaign shard 0/2 is duplicated"},
		{name: "missing ordinals without partial", prepare: func(_ *testing.T, shards []frontierMergeShard, spec *MergeSpec) {
			spec.ShardPaths = []string{shards[1].path}
		}, want: "campaign merge is missing 2 ordinal ranges"},
		{name: "incomplete input", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.SeedAt = nil
		}, want: "campaign merge input is incomplete"},
		{name: "malformed source record", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			flipLogByte(t, shards[1].path, 40)
		}, check: IsIntegrityError, want: "replay shard campaign"},
		{name: "torn source tail", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			appendLogBytes(t, shards[0].path, []byte{1, 2, 3})
		}, check: IsIntegrityError, want: "torn"},
		{name: "exploration record in a seed shard", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			appendLogRecord(t, shards[1].path, kindExploration, []byte(`{}`))
		}, check: IsIntegrityError, want: "exploration record"},
		{name: "missing failure artifact", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			if err := os.RemoveAll(shards[1].failure); err != nil {
				t.Fatal(err)
			}
		}, check: IsIntegrityError},
		{name: "corrupt success artifact", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			if err := os.Rename(shards[0].success, shards[0].success+".moved"); err != nil {
				t.Fatal(err)
			}
		}, check: IsIntegrityError},
		{name: "corrupt diagnostic sidecar", prepare: func(t *testing.T, shards []frontierMergeShard, _ *MergeSpec) {
			if err := os.WriteFile(shards[1].diagnostic, []byte("changed"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, check: func(err error) bool { return IsIntegrityError(err) && errors.Is(err, artifact.ErrCorruptReference) }},
		{name: "success evidence over its bound", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.Artifacts.SuccessBytes = 1
			spec.Artifacts.TotalBytes = spec.Artifacts.FailureBytes + 1
		}, check: func(err error) bool {
			var capacityErr *ArtifactCapacityError
			return errors.As(err, &capacityErr) && capacityErr.Limit == artifactLimitSuccessBytes && capacityErr.Outcome == CapacityInfrastructureFailure
		}},
		{name: "failure evidence over its count", prepare: func(_ *testing.T, _ []frontierMergeShard, spec *MergeSpec) {
			spec.Artifacts.FailureArtifacts = 0
		}, check: func(err error) bool {
			var capacityErr *ArtifactCapacityError
			return errors.As(err, &capacityErr) && capacityErr.Limit == ArtifactLimitFailureCount
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, shards, paths := frontierMergeShards(t)
			parent := filepath.Join(t.TempDir(), "absent")
			spec := frontierMergeSpec(t, paths, filepath.Join(parent, "merged"))
			test.prepare(t, shards, &spec)
			before := treeSnapshot(t, root)
			_, err := MergeCampaigns(context.Background(), spec)
			if err == nil || test.check != nil && !test.check(err) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("MergeCampaigns() error = %v, want %q", err, test.want)
			}
			if _, statErr := os.Lstat(parent); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("failed preflight created output: %v", statErr)
			}
			assertTreeUnchanged(t, root, before)
		})
	}
	t.Run("unpublished shard", func(t *testing.T) {
		root := t.TempDir()
		published := publishFrontierMergeShard(t, root, 0, true)
		running := publishFrontierMergeShard(t, root, 1, false)
		parent := filepath.Join(t.TempDir(), "absent")
		_, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, []string{published.path, running.path}, filepath.Join(parent, "merged")))
		if err == nil || !IsIntegrityError(err) || !strings.Contains(err.Error(), "campaign is not published") {
			t.Fatalf("MergeCampaigns() error = %v", err)
		}
		if _, statErr := os.Lstat(parent); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("failed preflight created output: %v", statErr)
		}
	})
}

func TestMergedJournalReferenceReportsTypedAggregateBounds(t *testing.T) {
	limits := executionJournalLimits{MaximumExecutions: 3, MaximumBytes: 100}
	var capacityErr *JournalCapacityError
	if _, err := mergedJournalReference(limits, nil, 4); !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitExecutions || capacityErr.Required != 4 || capacityErr.Maximum != 3 {
		t.Fatalf("executions over the bound error = %#v", err)
	}
	shards := []mergedShard{{JournalBytes: 60}, {JournalBytes: 41}}
	if _, err := mergedJournalReference(limits, shards, 2); !errors.As(err, &capacityErr) || capacityErr.Limit != journalLimitBytes || capacityErr.Required != 101 || capacityErr.Maximum != 100 || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("bytes over the bound error = %#v", err)
	}
	reference, err := mergedJournalReference(limits, shards[:1], 3)
	if err != nil || reference != (executionJournalReference{Schema: MergedCampaignLogFormat, Records: 3, Bytes: 60}) {
		t.Fatalf("reference = %#v, %v", reference, err)
	}
}

func TestMergeFaultsAtEveryWritePointLeaveNoOutputAndRetryPublishesTheSameLog(t *testing.T) {
	root, _, paths := frontierMergeShards(t)
	before := treeSnapshot(t, root)
	reference, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, paths, ""))
	if err != nil {
		t.Fatal(err)
	}
	referenceLog, err := os.ReadFile(filepath.Join(reference.Path, storeDirectory, logstore.LogFile))
	if err != nil {
		t.Fatal(err)
	}
	var points []string
	observed := WithStoreFaults(context.Background(), func(kind, operation string) error {
		points = append(points, kind+"/"+operation)
		return nil
	})
	if _, err := MergeCampaigns(observed, frontierMergeSpec(t, paths, "")); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(points, "log-write/append") || !slices.Contains(points, "log-sync/append") || !slices.Contains(points, "rename/merged-campaign") {
		t.Fatalf("merge points %v miss the log append, sync and publication", points)
	}
	for index, point := range points {
		for _, fault := range []string{"error", "partial", "cancel"} {
			if fault == "partial" && point != "log-write/append" {
				continue
			}
			t.Run(fmt.Sprintf("%03d-%s-%s", index, strings.ReplaceAll(point, "/", "-"), fault), func(t *testing.T) {
				parent := t.TempDir()
				output := filepath.Join(parent, "merged")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				seen := 0
				faulted := WithStoreFaults(ctx, func(string, string) error {
					defer func() { seen++ }()
					if seen != index {
						return nil
					}
					switch fault {
					case "partial":
						return &logstore.PartialWrite{Bytes: 3}
					case "cancel":
						cancel()
						return nil
					default:
						return errors.New("injected merge storage failure")
					}
				})
				_, err := MergeCampaigns(faulted, frontierMergeSpec(t, paths, output))
				if err == nil && fault != "cancel" {
					t.Fatalf("merge with a %s at %s succeeded", fault, point)
				}
				if err != nil && fault == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled merge error = %v", err)
				}
				// A failure after the publishing rename leaves the complete
				// aggregate; any earlier one leaves nothing at all.
				entries, readErr := os.ReadDir(parent)
				if readErr != nil {
					t.Fatal(readErr)
				}
				published := len(entries) == 1 && entries[0].Name() == "merged"
				if !published && len(entries) != 0 || err == nil && !published {
					t.Fatalf("merge with a %s at %s left %v (error %v)", fault, point, entries, err)
				}
				assertTreeUnchanged(t, root, before)
				if !published {
					if _, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, paths, output)); err != nil {
						t.Fatalf("retry after a %s at %s: %v", fault, point, err)
					}
				}
				retried, readErr := os.ReadFile(filepath.Join(output, storeDirectory, logstore.LogFile))
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(retried, referenceLog) {
					t.Fatalf("merge after a %s at %s published another log", fault, point)
				}
				if _, err := OpenMergedCampaign(output); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestOpenMergedCampaignRefusesDamagedScopes(t *testing.T) {
	_, _, paths := frontierMergeShards(t)
	merged, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, paths, ""))
	if err != nil {
		t.Fatal(err)
	}
	original := mergedLogRecords(t, merged.Path)
	var scopes []int
	for index, entry := range original {
		if entry.kind == kindMergedSource {
			scopes = append(scopes, index)
		}
	}
	firstEvidence := slices.IndexFunc(original, func(entry mergedLogRecord) bool { return entry.kind == kindMergedEvidence })
	if len(scopes) != mergeTestShards || firstEvidence < 0 {
		t.Fatalf("aggregate scopes %v evidence %d", scopes, firstEvidence)
	}
	clone := func() []mergedLogRecord { return slices.Clone(original) }
	for _, test := range []struct {
		name    string
		records func() []mergedLogRecord
		change  func(*mergedCampaignRecord)
		want    string
	}{
		{name: "exploration record in a source scope", records: func() []mergedLogRecord {
			return slices.Insert(clone(), scopes[0]+2, mergedLogRecord{kind: kindExploration, payload: []byte(`{}`)})
		}, want: "exploration record"},
		{name: "source record outside a scope", records: func() []mergedLogRecord {
			records := clone()
			return slices.Insert(records, 1, records[scopes[0]+1])
		}, want: "outside a merged source scope"},
		{name: "scopes out of shard order", records: func() []mergedLogRecord {
			records := clone()
			reordered := append([]mergedLogRecord{records[0]}, records[scopes[1]:firstEvidence]...)
			reordered = append(reordered, records[scopes[0]:scopes[1]]...)
			return append(reordered, records[firstEvidence:]...)
		}, want: "not the next shard"},
		{name: "source scope without its publication", records: func() []mergedLogRecord {
			return slices.Delete(clone(), scopes[1]-1, scopes[1])
		}, want: "campaign is not published"},
		{name: "source record after a scope's publication", records: func() []mergedLogRecord {
			records := clone()
			return slices.Insert(records, scopes[1], records[scopes[0]+2])
		}, want: "published campaign log has a later record"},
		{name: "missing scope", records: func() []mergedLogRecord {
			return clone()[:scopes[1]]
		}, want: "merged campaign log is incomplete"},
		{name: "missing evidence record", records: func() []mergedLogRecord {
			records := clone()
			return records[:len(records)-1]
		}, want: "has no evidence record"},
		{name: "evidence before a scope", records: func() []mergedLogRecord {
			records := clone()
			return slices.Insert(records, scopes[1], records[firstEvidence])
		}, want: "follows the merged evidence"},
		{name: "changed source provenance", change: func(envelope *mergedCampaignRecord) {
			envelope.Shards[1].LogSHA256 = record.HashBytes([]byte("another source"))
		}, want: "does not match its recorded provenance"},
		{name: "moved source path", change: func(envelope *mergedCampaignRecord) {
			envelope.Shards[0].Path = filepath.Join(t.TempDir(), "elsewhere")
		}, want: "merged source"},
		{name: "changed totals", change: func(envelope *mergedCampaignRecord) {
			envelope.Succeeded++
			envelope.Failures--
		}, want: "validate merged executions"},
		{name: "changed missing ranges", change: func(envelope *mergedCampaignRecord) {
			envelope.Partial, envelope.Missing = true, []ordinalRange{{Start: 3, End: 3}}
		}, want: "merged campaign summary is inconsistent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.records != nil {
				rewriteMergedLog(t, merged.Path, test.records()...)
			} else {
				rewriteMergedLog(t, merged.Path, original...)
				rewriteMergedEnvelope(t, merged.Path, test.change)
			}
			if _, err := OpenMergedCampaign(merged.Path); err == nil || !IsIntegrityError(err) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("OpenMergedCampaign() error = %v, want %q", err, test.want)
			}
		})
	}
	rewriteMergedLog(t, merged.Path, original...)
	appendLogBytes(t, merged.Path, []byte{1, 2, 3})
	if _, err := OpenMergedCampaign(merged.Path); err == nil || !IsIntegrityError(err) || !errors.Is(err, ErrIncompleteLog) {
		t.Fatalf("OpenMergedCampaign() with a torn tail error = %v", err)
	}
	if _, report, err := InspectMergedCampaign(merged.Path); !errors.Is(err, ErrIncompleteLog) || report.Tail != logstore.TailShortFrame || report.ObservedTailBytes != 3 || report.RepairedBytes != 0 {
		t.Fatalf("InspectMergedCampaign() with a torn tail = %#v, %v", report, err)
	}
}

// The aggregate borrows its sources' evidence: opening it resolves every
// borrowed reference, and neither opening nor recovery ever changes a source.
func TestMergedCampaignResolvesBorrowedEvidenceAndNeverSweepsItsSources(t *testing.T) {
	root, shards, paths := frontierMergeShards(t)
	before := treeSnapshot(t, root)
	merged, err := MergeCampaigns(context.Background(), frontierMergeSpec(t, paths, ""))
	if err != nil {
		t.Fatal(err)
	}
	mergedBefore := treeSnapshot(t, merged.Path)
	if _, err := RecoverCampaign(context.Background(), merged.Path); err == nil || !IsIntegrityError(err) {
		t.Fatalf("RecoverCampaign(aggregate) error = %v", err)
	}
	if _, _, err := ResumeCampaignJournal(context.Background(), merged.Path); err == nil || !IsIntegrityError(err) {
		t.Fatalf("ResumeCampaignJournal(aggregate) error = %v", err)
	}
	for _, path := range paths {
		recovered, err := RecoverCampaign(context.Background(), path)
		if err != nil || recovered.Changed {
			t.Fatalf("RecoverCampaign(shard) = %#v, %v", recovered, err)
		}
	}
	assertTreeUnchanged(t, root, before)
	assertTreeUnchanged(t, merged.Path, mergedBefore)
	if _, err := OpenMergedCampaign(merged.Path); err != nil {
		t.Fatal(err)
	}

	for name, damage := range map[string]struct {
		path    string
		corrupt bool
	}{
		"missing borrowed success artifact": {path: shards[0].success},
		"missing borrowed failure artifact": {path: shards[1].failure},
		"corrupt borrowed diagnostic":       {path: shards[1].diagnostic, corrupt: true},
	} {
		t.Run(name, func(t *testing.T) {
			if damage.corrupt {
				original, err := os.ReadFile(damage.path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(damage.path, []byte("changed"), 0o600); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.WriteFile(damage.path, original, 0o600); err != nil {
						t.Fatal(err)
					}
				}()
			} else {
				if err := os.Rename(damage.path, damage.path+".moved"); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Rename(damage.path+".moved", damage.path); err != nil {
						t.Fatal(err)
					}
				}()
			}
			if _, err := OpenMergedCampaign(merged.Path); err == nil || !IsIntegrityError(err) {
				t.Fatalf("OpenMergedCampaign() with %s error = %v", name, err)
			}
		})
	}
	if _, err := OpenMergedCampaign(merged.Path); err != nil {
		t.Fatal(err)
	}
	assertTreeUnchanged(t, root, before)
}

func TestPartialMergeReopensItsMissingRangesAndDistinctEvidence(t *testing.T) {
	_, shards, _ := frontierMergeShards(t)
	spec := frontierMergeSpec(t, []string{shards[1].path}, "")
	spec.Partial = true
	merged, err := MergeCampaigns(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenMergedCampaign(merged.Path)
	if err != nil {
		t.Fatal(err)
	}
	result := opened.Record
	if !result.Partial || !reflect.DeepEqual(result.Missing, []ordinalRange{{Start: 0, End: 0}, {Start: 2, End: 2}}) || result.Attempted != 2 || result.DistinctFailures != 1 || result.RetainedEvidence != 2 {
		t.Fatalf("partial aggregate = %#v", result)
	}
	if !reflect.DeepEqual(opened.Record, merged.Record) || !reflect.DeepEqual(opened.Executions, merged.Executions) {
		t.Fatal("reopened partial aggregate differs from the merge result")
	}
}

// flipLogByte damages one byte of a campaign log at offset.
func flipLogByte(t *testing.T, path string, offset int64) {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(path, storeDirectory, logstore.LogFile), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	value := make([]byte, 1)
	if _, err := file.ReadAt(value, offset); err != nil {
		t.Fatal(err)
	}
	value[0] ^= 0xff
	if _, err := file.WriteAt(value, offset); err != nil {
		t.Fatal(err)
	}
}

func appendLogBytes(t *testing.T, path string, data []byte) {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(path, storeDirectory, logstore.LogFile), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(errors.Join(err, file.Close()))
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// appendLogRecord appends one complete framed record to a campaign log
// without validating it against the campaign.
func appendLogRecord(t *testing.T, path string, kind logstore.Kind, payload []byte) {
	t.Helper()
	writer, _, err := logstore.OpenWriter(filepath.Join(path, storeDirectory), campaignLogOptions(nil, false), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Append(kind, payload); err != nil {
		t.Fatal(errors.Join(err, writer.Close()))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}
