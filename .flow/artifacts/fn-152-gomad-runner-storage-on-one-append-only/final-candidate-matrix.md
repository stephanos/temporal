# fn-152 final-candidate requirement matrix

Candidate base: `35fecd5fc974b4d294456b98635dce480747addc`  
Host: Linux/arm64, Go 1.27.1 stock toolchain  
Scope: the integrated fn-152 implementation through task `.9`

This is the retained final-candidate audit. A named test is a control only when
the verification ledger below records the command that executed its package.
Source review is used only for structural facts that runtime tests cannot prove.

## R1-R10 traceability

| Requirement | Implementation/control | Behavioral tests and source review | Final-candidate outcome |
| --- | --- | --- | --- |
| R1 — one append-only engine; acknowledged durability; torn-tail/corruption, writer exclusion, snapshot and filesystem guards | `runner/internal/logstore` owns `GMLG` v1 frames, per-kind bounds, strict stdlib JSON, CRC32C header/payload, exclusive writer replay/repair, lock-free replay, poisoning after uncertain append, and pinned private filesystem objects. Campaign, merge and corpus supply typed owners only. | `TestFrameRoundTripProtectsHeaderAndPayload`, `TestDecodeHeaderRejectsEveryUntrustedField`, `TestDecodeJSONIsStrict`, `TestOptionsRequireBoundedKnownKinds`, `TestEveryTruncationIsATornTailNeverCorruption`, `TestEveryByteFlipIsClassified`, `TestCorruptLengthIsNotALargeTornTail`, `TestValidFramesWithInvalidContentsAreCorruption`, `TestInjectedCreateAppendAndSyncFaultsPreserveAcknowledgedCommits`, `TestInjectedRepairFaultsNeverLoseCommittedRecords`, `TestSecondWriterIsInUseThroughAnyAcceptedAlias`, `TestReadOnlyReplayNeverMutates`, `TestInvalidMetadataFailsWithoutMutation`, `TestReplacementDuringOpenAppendAndRepairIsRejected`, `TestChangedReaderSnapshotFailsWithoutRepair`, `TestWriterRejectsExternalLogChangesBeforeAppend`, `TestConcurrentAppendsAndLockFreeReaders`. | Satisfied. The matrix below covers every mutation boundary and owner. |
| R2 — campaign lifecycle/outcomes/progress/resume only in the log; retired journal/segment/staging/archive state gone | `campaign_log.go` defines open, lifecycle, plan, admission, receipt, outcome, stop, exploration-round, publish, merged-source and counted-evidence records. `.prepared` and `.partial` are referenced payload/work directories, not competing state authorities. | `TestCampaignLogPublishesLifecycleAndOutcomesAsItsOnlyState`, `TestCampaignLogRecordsTheFrozenResumePlanAndRefusesChangedTargets`, `TestCampaignLogRetiresSegmentGroupingLimits`, `TestCampaignStorageRefusesLegacyAndUnknownFormatsBeforeMutation`; final inventory below. The stale `pre-target-evidence-merged` golden and its storage-only test were deleted; the dynamic behavioral refusal test remains. | Satisfied. Marker-only refusal detects an old layout but never decodes it. |
| R3 — interruption at every commit boundary resumes to uninterrupted semantics; no committed outcome repeats; durable admissions/receipts/atomic stop drain | Seed execution records admission before launch, terminal receipt before ordered outcome, and stop plus drain outcomes in one record. The built WASM CLI gate waits on committed log state, stops the CLI process group, fixes the durable prefix with a lock-free inspect while the writer remains stopped, discovers the now-stable owned process tree, sends SIGKILL, proves the same prefix post-kill, resumes, and compares 12 outcomes to an uninterrupted campaign. | `TestFrontierScriptsCommitTheirOrderedOutcomes`, `TestFrontierFaultsAtEveryPointResumeToTheUninterruptedOutcomes`, `TestFrontierAdmitsWithinParallelAndRefillsIncrementally`, `TestFrontierReceiptsRejectConflictsAndRetentionDecisions`, `TestFrontierStopGroupKeepsActualDrainOutcomesAndFreezesAdmission` (buffered successes/failures and committed cancellations), `TestFrontierPendingPayloadsStayLiveUntilTheirOutcomeAdoptsOrRetiresThem`, `TestSeedCampaignResumesEveryAppendFaultToTheUninterruptedOutcome`, `TestSeedCampaignRepeatedResumeConverges`, `TestSeedCampaignResumeRefusesAMissingReceiptPayload`, and Linux `TestInstalledOperatorKillResume`. | Satisfied on the supported Linux/arm64 WASM host. The old native `cmd/gomad.TestCLIKillResume` was source-reviewed only and is not claimed as executed. Scripted portable outcomes supplement but do not substitute for the WASM gate. |
| R4 — choice and simulation use one round journal | `campaign.Rounds`/`RoundEngine` own plan, begin and commit records for both strategies; no strategy-owned storage journal remains. | `TestExplorationRoundsCommitAndReplayWholeRounds`, `TestExplorationRoundsDiscardAnUncommittedRoundOnResume`, `TestExplorationRoundFaultsResumeToTheUninterruptedState`, `TestExplorationRoundsRejectInvalidTransactionsWithoutAppending`, `TestExplorationRoundsBoundTheirTransactionBeforeWriting`, `TestExplorationPublicationMatchesItsRounds`, `TestExplorationResumesEveryRoundAppendFaultToTheUninterruptedRounds`, and WASM interrupted-round tests in the full gate. | Satisfied. Source inventory finds semantic “segment” types but no strategy storage journal. |
| R5 — merge produces one log and exact combined totals; all invalid input fails before output | Merge preflights all source logs/evidence, then appends source-scoped records and counted evidence; aggregate borrows rather than owns shard payloads. | `TestMergeConcatenatesSourceScopesAndReplaysTheirCombinedTotals`, `TestMergePreflightFailsBeforeCreatingAnyOutput`, `TestMergeFaultsAtEveryWritePointLeaveNoOutputAndRetryPublishesTheSameLog`, `TestOpenMergedCampaignRefusesDamagedScopes`, `TestMergedCampaignInspectionReportsTheShardsCombinedTotals`, and portable merge completeness/order/dedup tests. | Satisfied. |
| R6 — corpus log plus rebuilt live index; duplicate no-op and evicted hash readmission; live bounds | `gomad3.guide-corpus-log/v1` appends a header then complete admission snapshots; open replays current live state into hash indexes. Live capacity is 1,024 entries/1 GiB; historical log bytes grow separately. | `TestCorpusLogReplaysEvictionAndReadmission`, `TestCorpusLogFaultsPreservePriorSnapshot`, `TestCorpusReclaimsUncommittedPublicationOnReopen`, `TestCorpusReplayRejectsInvalidAdmissionsBeforeApplying`, `TestCorpusTornTailIsReportedReadOnlyAndRepairedByOpen`, `TestCorpusPublishesCanonicalSnapshotOnlyAfterMatchingReplay`, `TestCorpusByteCapCountsTheSharedTargetOnce`, `TestCorpusCanonicalSnapshotBaseline`. | Satisfied. No source-bound corpus input is available on this host, so no parked size/time measurement is claimed or estimated. |
| R7 — publish payload before reference; reclaim only owned unreachable files; pending receipts stay live | Artifact publication syncs payloads/envelope/manifest/staging rename/root before returning. Campaign/corpus owners validate live references before sweeping. Merge payloads and shared pools remain borrowed. | `TestPublicationIsDurableBeforeItReturnsAReference`, `TestEveryPublicationCrashWindowLeavesOnlyReclaimableState`, `TestSweepKeepsLiveReferencesAndReclaimsWhatCommittedStateRetired`, `TestSweepRemovesNothingWhenALiveReferenceIsMissingOrCorrupt`, `TestSweepNeverTakesSharedPoolsBorrowedArtifactsOrForeignEntries`, `TestPublishOwnedFileIsDurableAndContentAddressedByName`, `TestCampaignRecoveryValidatesLiveEvidenceBeforeReclaimingAnything`, `TestCorpusMissingLiveCaseIsCorruptionAndPreventsCleanup`, `TestFrontierPendingPayloadsStayLiveUntilTheirOutcomeAdoptsOrRetiresThem`. | Satisfied. |
| R8 — old/unknown/missing lineage named and refused; no migration/decoder | Artifact storage envelope, campaign/merged marker detector, corpus marker detector and portable-plan storage field reject before mutation. Existing corpus stores receive a read-only lineage/state preflight before root chmod, tail repair or append; only an absent store may bootstrap its header. | `TestOpenRefusesEveryOtherStorageLineageBeforeDecodingTheManifest`, `TestOpenPreMoveMinimizedArtifactIsRefusedAsUnsupportedStorageFormat`, `TestCampaignStorageRefusesLegacyAndUnknownFormatsBeforeMutation`, `TestCorpusRejectsLegacyAndUnknownLineageBeforeMutation` (including a pre-existing empty log plus malformed/future-lineage storage-tree preservation), `TestCampaignPlanOfAnotherLineageIsRefusedBeforeExecutionMergeAndInspection`; final inventory below. | Satisfied. The completion review found and the final candidate fixed empty corpus-log bootstrap; the former old merged directory golden is absent, and refusal remains behavioral and dynamically constructed. |
| R9 — human/JSON inspection replays logs and preserves tail/corruption/status distinctions | Inspect v8 projects `LogInspection`, committed lifecycle/frontier, corpus live bounds, merged provenance, and invalid-store diagnostics from one read-only replay. Docs describe lock-free no-repair behavior and writer-only torn-tail repair. | `TestInspectReportsCampaignLogStateWhileAWriterAppends`, `TestInspectObservesATornCampaignTailWithoutRepairingIt`, `TestInspectDiagnosesADamagedLogAsAnInvalidStore`, `TestInspectClassifiesInvalidCurrentCorpusHeadersAsCorruption`, `TestAppendOnlyStoreInspectionUsesVersionedSchema`, `TestInspectReportsAGuidedCorpusFromItsLog`, `TestMergedCampaignInspectionReportsTheShardsCombinedTotals`, CLI `TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly`, `TestRunInspectReportsATornTailWithoutRepairingIt`, `TestRunInspectReportsAGuidedCorpusDirectory`, `TestPrintInspectionReportsCommittedStateRoundsAndMergedSources`. | Satisfied; README, ARCHITECTURE, SPEC and CLI now describe v8 and the implemented contracts. |
| R10 — no third-party storage dependency | `logstore` imports the standard library plus `internal/hostfs`; owners stay under Runner. `go.mod` and `go.sum` are unchanged in the candidate. | `internal/archcheck.TestPackageArchitecture`, `TestBackendImportDirection`, `TestHostPackageVet`, `internal/puritycheck`; final `git diff` dependency/generated checks. Strict payload decoding stays on stdlib `encoding/json`, explicitly outside fn-153. | Satisfied. |

## Append, sync, repair and owner control matrix

| Boundary/owner | Injected or adversarial control | Required durable observation |
| --- | --- | --- |
| Store/root/lock/log creation and parent/store directory sync | `TestInjectedCreateAppendAndSyncFaultsPreserveAcknowledgedCommits`; full-umask and invalid-metadata tests | Failure never fabricates a commit or mutates invalid existing metadata; a successful reopen sees the acknowledged prefix. |
| Frame write: zero bytes, every partial cut, full-frame error | `TestEveryTruncationIsATornTailNeverCorruption`; `TestInjectedCreateAppendAndSyncFaultsPreserveAcknowledgedCommits` | Unacknowledged frame is absent or a recognized final tail; prior acknowledged frames survive. |
| Append sync and post-sync/return uncertainty | same injected fault matrix | Writer is poisoned; reopen decides whether the full frame survived, and it is never half-applied. |
| Header/payload bit damage, unknown kind/version/flags, oversized length, strict-JSON or consumer-invalid payload | frame classification tests and owner invalid-record tables | Complete invalid data is corruption, never a repairable tail and never applied. |
| Torn-tail truncate and repair sync | `TestInjectedRepairFaultsNeverLoseCommittedRecords`, `TestReplacementDuringOpenAppendAndRepairIsRejected` | Failed repair never loses the valid prefix; replacement/identity changes prevent mutation. |
| Read-only snapshot replacement/shrink/concurrent growth | reader snapshot and concurrent-reader tests; campaign live inspect tests | No lock or mutation; stable captured prefix or explicit snapshot-changed failure. |
| Campaign create/open/lifecycle/plan/outcome/publish | `TestCampaignLogFaultsAtEveryAppendAndSyncPointLeaveReplayableState`, creation/resume/published-recovery fault tests | Replay yields absent, resumable or published state; retry converges. |
| Seed admission/receipt/outcome/stop group | 163-case frontier matrix plus seed append-fault and repeated-resume tests | Committed outcomes do not rerun; receipt-less admissions may rerun; buffered actual outcomes and cancellations drain atomically; stopped replay admits nothing. |
| Choice/simulation plan/begin/commit round | round fault matrices and Runner uninterrupted/resumed comparisons | An uncommitted round is discarded once; committed round order/state matches uninterrupted execution. |
| Merge envelope/source/count records | merge preflight and every-write-point fault test | No invalid output is published; retry produces the same log and totals. |
| Corpus header/admission sync | corpus log fault, eviction/readmission, invalid-admission and tail tests | Prior snapshot remains live; duplicate is a no-op; committed eviction enables readmission; only recognized tail is repaired by writer open. |
| Artifact/pending payload publication and sweep | artifact crash-window/sweep tests, campaign/corpus live-reference tests | Bytes are durable before reference; missing/corrupt live data prevents deletion; borrowed/shared data is never swept. |

## Final source inventory

Searches cover `runner`, `cmd/gomad`, `artifact`, and `backends/wasm` for
`campaign.json`, `corpus.json`, `corpus.lock`, `executions/index.json`, old
execution-journal schemas, segment/staging/archive journal types, and legacy
format names. Results are limited to:

- marker-only detectors in `campaign_log.go` and corpus log code that return
  `UnsupportedFormatError` before creating a current store;
- dynamic refusal tests and portable-plan negative controls;
- retired zero-valued compatibility report/plan fields (`index_sha256`,
  `segments`, `segment_records`, `maximum_segments`), which drive no storage;
- exploration-domain segment values unrelated to persistence.

There is no active old Campaign segment writer/reader, lifecycle file,
resume-archive, corpus index writer, strategy journal, migration, or decoder.
The only retained old-format artifact golden is the fn-164 Artifact refusal
control required by R8; the storage-only merged directory golden was removed.

## Verification and preservation ledger

All commands below ran on the Linux/arm64 final candidate with the stock Go
1.27.1 toolchain where stated. Exit status, rather than a scraped output line,
is the pass observation.

| Surface | Exact command/scope | Outcome |
| --- | --- | --- |
| Parent quick gate | `go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus` | PASS on the final uncommitted candidate: logstore and corpus. It is rerun at the committed review-fix head before the green receipt is minted. |
| Owner packages | `env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/campaign ./runner/internal/corpus ./artifact` | PASS: logstore, campaign, corpus and artifact. |
| Runner/CLI/backend preservation | `env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH go -C tools/gomad3 test -tags test_dep -count=1 -run '^(TestSeedCampaignResumesEveryAppendFaultToTheUninterruptedOutcome|TestSeedCampaignRepeatedResumeConverges|TestSeedCampaignResumeRefusesAMissingReceiptPayload|TestExplorationResumesEveryRoundAppendFaultToTheUninterruptedRounds|TestInspectReportsCampaignLogStateWhileAWriterAppends|TestInspectObservesATornCampaignTailWithoutRepairingIt|TestInspectDiagnosesADamagedLogAsAnInvalidStore|TestInspectClassifiesInvalidCurrentCorpusHeadersAsCorruption|TestInspectRejectsANonPrivateCorpusRootWithoutMutation|TestAppendOnlyStoreInspectionUsesVersionedSchema|TestInspectReportsAGuidedCorpusFromItsLog|TestMergedCampaignInspectionReportsTheShardsCombinedTotals|TestCampaignPlanOfAnotherLineageIsRefusedBeforeExecutionMergeAndInspection|TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly|TestRunInspectReportsATornTailWithoutRepairingIt|TestRunInspectReportsAGuidedCorpusDirectory|TestPrintInspectionReportsCommittedStateRoundsAndMergedSources|TestValidatePreparedAcceptsCanonicalEmptySequences)$' ./runner ./cmd/gomad/internal/cli ./backends/wasm/backend` | PASS. The selector covers every storage behavior this task names plus the nil/empty prepared-identity control. |
| Runner internals race gate | `env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/...` | PASS: campaign, corpus, exploration, logstore and minimizer. |
| Completion-review R8 fix | `go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus` and `go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/corpus` in the same stock-Go environment | PASS. The regression was demonstrated red before implementation: an existing empty log was accepted, and malformed/future-lineage refusals changed root mode. The final candidate refuses all three without changing the captured storage tree. |
| WASM built-CLI review-fix focus | `env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -v -tags test_dep -count=1 -timeout=10m -run '^(TestValidatePreparedAcceptsCanonicalEmptySequences|TestInstalledOperatorKillResume)$' ./backends/wasm/backend ./backends/wasm/command` (from `tools/gomad3`) | PASS. |
| WASM built-CLI stability | same stock-Go environment, `go test -v -tags test_dep -count=5 -timeout=10m -run '^TestInstalledOperatorKillResume$' ./backends/wasm/command` (from `tools/gomad3`) | PASS five of five; every run froze and killed after one committed outcome, post-kill inspect saw exactly one, and resume matched the 12-outcome baseline. |
| Full WASM gate | `make -C tools/gomad3 test-wasm GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go` | PASS in 287 seconds. `TestInstalledOperatorKillResume` executed (not skipped) and passed. Prototype tests that require `GOMAD_WASM_PROTOTYPE_ROOT` retained their pre-existing explicit skips; they are not storage acceptance. |
| Generated/toolchain validation | `env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make -C tools/gomad3 validate` | PASS: version/protocol/boundary/compiler checks, patch/script validation, compatibility packs and qualification manifest. No declaration changed, so there is no additional affected generator. |
| Both-source-set static checks | `env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_ARCH_ALL_PLATFORMS=1 PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck ./internal/puritycheck` | PASS. |
| Fast changed-package lint | `PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make GOLANGCI_LINT_BASE_REV=35fecd5fc974b4d294456b98635dce480747addc GOLANGCI_LINT_FIX=false GOLANGCI_LINT=.flow/tmp/fn152-8-linux-tools/golangci-lint ERRORTYPE=.flow/tmp/fn152-8-linux-tools/errortype lint-code-fast` | PASS, 0 issues. |
| Nested Gomad3 lint | `PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make GOLANGCI_LINT_BASE_REV=35fecd5fc974b4d294456b98635dce480747addc GOLANGCI_LINT_FIX=false GOLANGCI_LINT=.flow/tmp/fn152-8-linux-tools/golangci-lint ERRORTYPE=.flow/tmp/fn152-8-linux-tools/errortype lint-code-gomad3` | PASS, 0 issues. |
| Broad default-main lint diagnostic | same Linux-native lint tools, `make lint-code-fast` with the repository's default `main` base | RED on pre-existing untouched `cmd/tools/lintcode/main.go:412`: `ownership.classify` cyclomatic complexity 28 exceeds 25. The initial attempt also selected a wrong-host cached binary and was inconclusive; its one `--fix` blank-line side effect was removed. No unrelated lint debt was changed. |

The source-inventory script found an empty identity-path diff for
`runner/simulation_broker.go`, `choice/interrupted.go`, the embedded WASM
simulation/WASI source sets, `world`, `volume`, `simulation/**`, and
`internal/canonicaljson`. `backends/wasm/backend/provider.go` is outside those
embedded identities; its change only treats nil and empty ordered argument/tag
slices as the same semantic prepared target. `go.mod`, `go.sum`, and generated
paths have empty diffs. Production old-path hits are only marker refusal code
for `campaign.json`, `merge.json`, `executions`, `corpus.json`, and
`corpus.lock`, plus current append-only code whose public type is still named
`Journal`; none decodes the old bytes. `git ls-files` finds no surviving file
under the removed merged-fixture path.

README, ARCHITECTURE, SPEC and CLI match the current source. No source-bound
corpus input exists on this host, so no corpus size/time result is claimed or
estimated.

## Obligations deliberately remaining red

- fn-155 keeps its recorded priority; this task does not expand native support.
- fn-128/fn-149 still own native qualification and the older native
  `TestCLIKillResume`. It was source-reviewed, not executed on Linux/arm64.
- Full requalification/rotation obligations recorded by those specs remain;
  portable scripted tests do not satisfy them.
- A source-bound corpus size/time measurement remains parked until a real
  qualifying input is supplied.

`MILESTONES.md` is intentionally unchanged, so those priorities and deferrals
remain verbatim.
