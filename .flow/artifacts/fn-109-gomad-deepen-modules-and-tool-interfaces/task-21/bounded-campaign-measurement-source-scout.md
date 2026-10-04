# R19 bounded campaign measurement preparation

The read-only bounded_campaign_measurement_scout grounded a future measurement
in the injected seed-campaign harness. Numerical policy-storage and full-payload
copy measurements remain missing. This is preparation, not task-21 admission,
implementation, an executed test/profile, or native qualification.

## Existing controls and selected harness

The actual current control is runner/retention_characterization_test.go:874,
TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs. pairedExecutor
enforces two concurrent executions without retaining request/result history.
Its discard/count/byte cases prove active width and capacity classification,
not policy-storage bytes or copy counts. No ReleasesCompletedPayloads-named
test was found in the inspected current/retained inputs. fakeExecutor retains
requests and must not be used for memory measurement.

Use the existing newFakePreparer, testConfig, processResult, semanticTranscript,
completeEmptyTranscript and exploreWith private dependency seam. A scratch-only
same-package verification harness can be retained beneath task-21 artifacts and
applied only to isolated baseline/current measurement copies. It needs no
production test hooks or shipped source change. Production gaps still return to
their owning tasks; task 21 remains a final verification/evidence owner.

Select StrategySeed, Seeds 1-10/1-100, Parallel 2, PolicyAll, CoverageSemantic,
Guide false, no resume/sharding/diagnostics/choice exploration, identical limits
and timeouts. An executor patterned on pairedExecutor produces distinct bounded
stdout/stderr/transcript backing without retaining history. Suggested payloads
are 1 MiB per stream plus approximately 1 MiB of valid transcript with repeated
identical semantic probes; use identical OutputLimit/IOTranscriptLimit and World
limits. completionWorldRecord exercises connected World assessment. Freeze the
final task-19 candidate before resolving exact constructors.

Run discard to completion at both sizes, then novel retention with identical
features, count bound 3 and byte bound 64 MiB so one success is retained while
all jobs finish. Keep existing all-success count/byte exhaustion controls.
Those stop after five attempts and cannot alone measure 10/100 completed jobs.

## Report policy storage, evidence and copies separately

SeedController is constant-sized; one seedRange backs either selection and its
Iterator. newShardedSeedController/pendingJobs.Next and completion channels have
parallelism-bound state; orderShardRunCompletions owns the pending map, and the
fresh-run resume completed map is empty. A direct companion measurement can
report unsafe.Sizeof controller/job-source/iterator/range backing/channel slots
and maximum controller activity. These are logical bytes, not allocator/map
metadata. Actual full-campaign allocation/live profiles are needed to inspect
state beyond those named structures.

Use runtime.MemProfileRate=1 and a gated executor to collect profiles after GC
at first pair, late pair and completion. Attribute bytes/objects by allocation
site: controller/selection construction, channels, reorder-map insertion, payload
producers, assessment and publication. Retain journal records/bytes/segments,
artifact paths, retained count/bytes and unique novelty entries separately.
CampaignJournal streams records; bounded index evidence is not controller state.
normalizeExecutionJournalLimits/DeriveArtifactCapacityPlan derive aggregate
capacity from selection count, so those capacities legitimately differ at 10
and 100. Keep per-execution/retention limits fixed and report derived bounds.

runCompletion, assessCompletion, executionArtifactInput, PublishArtifact and
artifactDataPayload pass payload slice headers; direct constructor alias checks
should cover every nonempty field. Alias checks alone cannot exclude transient
copies. Profiles supply allocation evidence, normalized by completed execution
and payload kind. Cumulative allocations should grow with more executions;
unchanged per-execution copy cost and bounded live policy/payload state are the
requirements. Do not infer integer copy counts from aggregate heap totals.

Existing materialization sites include hostexec Capture.Result's retained/raw
output buffers; deterministicio Session.collectFrame's transcript allocation;
choice session framing/DecodeTrace and DecodeDiagnosticTrace clones;
readonlymount captured-file copies; execution/worldrecord canonical encoding;
and artifact ReadPayload materialization. Artifact publication uses a fixed
64 KiB streaming buffer; filesystem target sharing/copy fallback is a separate
publication count, not a heap-payload clone.

## Reconstruct the actual baseline

The first-task baseline is 6782b55f49a0317b230e827ea2a63a37d116d502 plus dirty
fn-108.2–.6, as task1-evidence.txt records. Neither planning d4d800fb47 nor the
bare commit supplies that source. Under fn-108 artifacts retain:

- final-working-tree.diff: 24 tracked modifications.
- task5.diff: new completion.go and completion tests/characterizations.
- task6.diff: new retention.go and retention tests/characterizations.
- task4-working-tree.diff: new upgrade_unix_test.go.
- final-protected-paths.txt/final.md: seven untracked files and source identity.

Extract relevant final new-file blocks rather than blindly applying overlapping
task diffs after the tracked final diff. Verify reconstructed identities against
the retained inventory. Later fn-109 preimages include predecessor extractions
and are cross-checks, not silent baseline substitutes. task6-pre-edit-overlay
references obsolete /private/tmp paths and is pre-fn-108.6, a different baseline.
Retained canonical projections/reviews prove behavior, not measured copy counts.

## Future execution and evidence bounds

Run each baseline/current 10/100 subcase separately with pinned Go 1.27.1,
-count=1 and -tags test_dep, retaining source/test/overlay/profile identities.
Example future commands, once the harness exists:

```bash
go test -count=1 -tags test_dep -run '^TestR19BoundedSeedCampaignMeasurement/10_jobs$' -memprofile=/absolute/evidence/current-10.pprof -memprofilerate=1 ./runner
go test -count=1 -tags test_dep -run '^TestR19BoundedSeedCampaignMeasurement/100_jobs$' -memprofile=/absolute/evidence/current-100.pprof -memprofilerate=1 ./runner
go tool pprof -top -alloc_space /absolute/evidence/current-10.pprof
go tool pprof -top -alloc_objects /absolute/evidence/current-10.pprof
go tool pprof -top -inuse_space /absolute/evidence/current-100-late-pair.pprof
```

The late-pair snapshot requires explicit harness profiling; ordinary memprofile
is process-end only. On this linux/arm64 host deterministicio validation rejects
the platform. An isolated, identical baseline/current developmental platform
overlay can permit injected host-policy measurements without a runtime stand-in.
Such results exclude real preparation, supervisor transport, capture, patched
runtime scheduling, replay and native qualification. Required native gates on
darwin/arm64 and linux/amd64 remain open.

Requested routing: gpt-6.1-sol/high; actual execution metadata unobservable.
Tier: session (jev-unavailable(no_key)); selector input was corrected after an
initial schema-validation failure, then judged once successfully. The scout
performed reads/searches only, no edits/artifacts/tests/builds/generation/package
loading, Flow/Git mutations, bridges or further agents. Task21 remains todo;
dependencies 19/20 are unresolved. The harness and numbers are future work.
