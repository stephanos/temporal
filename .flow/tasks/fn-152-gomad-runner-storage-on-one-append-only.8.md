---
satisfies: [R1, R2, R8, R9]
---
# fn-152-gomad-runner-storage-on-one-append-only.8 Expose log and corpus state through existing Inspect dispatch and migrate consumers

## Description
Implement R9 and migrate the surveyed active storage consumers.

**Size:** M
**Files:** Core files (4-5): runner/inspect.go, inspect_capacity.go, cmd/gomad/internal/cli/cli.go and focused Runner/CLI inspection tests. Wider touches: characterization/projection/exploration-divergence fixtures; qualification/set/prune.go and tests only if campaign path assumptions changed; e2e readiness fixtures move to committed log evidence; qualification diagnostic and artifact fixture constructors retain their source contracts.
**Touches:** [tools/gomad3/runner/inspect*.go, tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/cmd/gomad/e2e_test.go, tools/gomad3/qualification/set/**, tools/gomad3/qualification/qualification.go, tools/gomad3/qualification/soak/**, tools/gomad3/cmd/gomadtool/diagnostic.go, tools/gomad3/internal/gomadtool/conformance/testdata/runner_external/consumer.go]

### Approach

Extend existing inspect PATH dispatch by the current log envelope kind, including direct corpus-directory paths. This is a root design inference from the existing campaign/artifact/plan dispatcher, not a quoted original user command. Preserve --json and --choices grammar. Project log kind/lineage, snapshot EOF, validated record count/offset, observed/repaired tail distinction, corruption location/reason, final committed state and applicable count/byte limits. A recognized corrupted store returns a structured report plus typed error so CLI writes its human/JSON diagnosis on stderr and returns status 2; JSON errors have no additional plaintext duplicate; output delivery failure remains status 3. Healthy/torn observation uses status 0 and never mutates storage. Preserve artifact and portable-plan inspection, existing outcomes/replay commands and diagnostics visibility. Update the complete active consumer inventory rather than replacing unrelated qualification manifests also named corpus.json.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/inspect.go:361`
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:145`
- `tools/gomad3/cmd/gomad/internal/cli/cli_test.go:785`
- `tools/gomad3/qualification/set/prune.go:36`
- `tools/gomad3/qualification/qualification.go:614`
- `tools/gomad3/cmd/gomad/e2e_test.go:379`
- `.flow/memory/bug/integration/engine-summary-fields-need-the-runner-2026-10-03.md`

### Verification

Investigate tools/gomad3/runner/inspect.go:361,437,455; cmd/gomad/internal/cli/cli.go:145,200; cli/cli_test.go:785; qualification/set/prune.go:36; qualification/qualification.go:614; cmd/gomad/e2e_test.go:379; .flow/memory/bug/integration/engine-summary-fields-need-the-runner-2026-10-03.md. Focused command uses go -C tools/gomad3 test -tags test_dep -count=1 with existing portable Inspect/CLI selectors, retained exactly.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R9: campaign, merged campaign and direct corpus-directory inspection in human and JSON form derives state from read-only log replay and shows log count/bytes/offset/kind/version and snapshot/tail status.
- [ ] R9: recognized corruption emits a human or JSON diagnosis on stderr and returns status 2 without successful-state stdout or a plaintext JSON duplicate; partial replay is explicitly an invalid store, missing/invalid paths retain prior diagnostics precedence, and report delivery failures return status 3.
- [ ] R1/R9: inspect reports an observed torn final tail without repair or writer-lock acquisition, remains usable while a writer appends, and distinguishes corruption/unsupported lineage from the validated prefix.
- [ ] R2/R8/R9: active qualification/pruning/diagnostic consumers and CLI/manual fixtures use current APIs or payload paths; old journal readers are gone, unrelated qualification corpus manifests are preserved, and artifact/plan/--choices inspection still works.
- [ ] R9: public projections retain classified outcomes, replay evidence and existing byte/count/novelty/lifecycle fields whose semantics survive the storage rewrite.

## Done summary
Implemented read-only append-only inspection for campaign, merged-campaign, and direct corpus stores, migrated active consumers to log-backed projections, and added stable human/JSON invalid-store diagnostics under `gomad3.inspect/v8`. Inspection takes one replay snapshot, never acquires a writer lock or repairs state, reports torn tails without mutation, and separates corruption encoded in the log from failures in external borrowed evidence.

### Design report

- Campaign, merged-campaign, and corpus inspection project committed state from append-only replay reports, including lineage, validated bytes/records, snapshot bytes, tail state, lifecycle/frontier state, source provenance, and applicable count/byte limits.
- Recognized malformed or semantically invalid current-format stores return `InvalidStoreError` with a structured report; the CLI renders that report alone on stderr and returns status 2. Unsupported lineage remains distinct, while healthy and observed torn-tail snapshots return status 0 without repair.
- Merged-log envelope and embedded-source semantic defects carry a dedicated corruption marker. Pure embedded-state validation runs before external artifact resolution, so missing/corrupt borrowed files retain ordinary external-error behavior.
- The public inspection schema is `gomad3.inspect/v8`. The corresponding README reference remains assigned to documentation task `.9` and was deliberately excluded from this task's implementation range.

### Acceptance-to-test mapping

- R9 campaign/merged/corpus human and JSON projection: `TestInspectReportsAGuidedCorpusFromItsLog`, `TestMergedCampaignInspectionReportsTheShardsCombinedTotals`, `TestRunInspectReportsAGuidedCorpusDirectory`, and `TestPrintInspectionReportsCommittedStateRoundsAndMergedSources`.
- R9 structured corruption, stderr/status handling, delivery failure, and validated prefix: `TestInspectDiagnosesADamagedLogAsAnInvalidStore`, `TestInspectClassifiesInvalidCurrentCorpusHeadersAsCorruption`, `TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly`, plus the merged relative-path and missing-publication subtests in `TestMergedCampaignInspectionReportsTheShardsCombinedTotals`.
- R1/R9 torn-tail and concurrent read-only behavior: `TestInspectObservesATornCampaignTailWithoutRepairingIt`, `TestInspectReportsCampaignLogStateWhileAWriterAppends`, and `TestInspectCampaignStoreReadsALiveLogWithoutRepairingIt`.
- R1/R8 metadata and lineage guards: `TestInspectRejectsANonPrivateCorpusRootWithoutMutation`, `TestReadLogRefusesARootThatIsNotADirectory`, and the current-format corpus identity/admission corruption cases.
- R2/R8/R9 consumer and public-projection compatibility: `TestCreateCampaignPlanPublishesCanonicalPreparedTargetBundle`, the CLI projection tests above, the architecture/purity checks, and the external runner-consumer conformance build exercised by the WASM/validation gates.

### Verification

- Baseline and final Quick: `go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus` — green.
- Stock Go 1.27.1 WASM gate: `make -C tools/gomad3 test-wasm GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go` — green; WASI completed in 290.090s.
- All task-focused inspection tests, including merged semantic corruption, repeated with `-count=5` — green.
- `go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/...` — green.
- `make -C tools/gomad3 validate` and `GOMAD3_ARCH_ALL_PLATFORMS=1 go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck ./internal/puritycheck` — green.
- Task-scoped Gomad lint with stock Go 1.27.1 and repository linters — green. A whole-repository fast-lint probe exposed only the pre-existing `cmd/tools/lintcode/main.go` cyclomatic finding outside this task and is not counted as task evidence.
- The broad portable `./runner` package is not reported as a gate: on this linux/arm64 host it includes known native-host/toolchain cases that are outside the parent Quick command. The required portable-focused, race, validation, architecture, and supported WASM surfaces are green.

stage: impl-review - ran [2026-10-13T06:08Z..2026-10-13T06:45Z] (SHIP, bounded three-round fix loop)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d344601348cd0a2a0e12ef951fef5bb0fd135c9e, b8857c437807b6f828f81b2f3a31b0f86e81ac18, 992bba73fd25b75aea3f375e95f5f43aea2ce782
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus, make -C tools/gomad3 test-wasm GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go, go -C tools/gomad3 test -tags test_dep -run '^(TestInspectReportsCampaignLogStateWhileAWriterAppends|TestInspectObservesATornCampaignTailWithoutRepairingIt|TestInspectDiagnosesADamagedLogAsAnInvalidStore|TestInspectClassifiesInvalidCurrentCorpusHeadersAsCorruption|TestInspectRejectsANonPrivateCorpusRootWithoutMutation|TestAppendOnlyStoreInspectionUsesVersionedSchema|TestInspectReportsAGuidedCorpusFromItsLog|TestMergedCampaignInspectionReportsTheShardsCombinedTotals)$' -count=5 ./runner, go -C tools/gomad3 test -tags test_dep -run '^TestInspectCampaignStoreReadsALiveLogWithoutRepairingIt$' -count=5 ./runner/internal/campaign, go -C tools/gomad3 test -tags test_dep -run '^TestReadLogRefusesARootThatIsNotADirectory$' -count=5 ./runner/internal/corpus, go -C tools/gomad3 test -tags test_dep -run '^(TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly|TestRunInspectReportsATornTailWithoutRepairingIt|TestRunInspectReportsAGuidedCorpusDirectory|TestPrintInspectionReportsCommittedStateRoundsAndMergedSources)$' -count=5 ./cmd/gomad/internal/cli, go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/..., make -C tools/gomad3 validate, GOMAD3_ARCH_ALL_PLATFORMS=1 go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck ./internal/puritycheck, PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make GOLANGCI_LINT_BASE_REV=cbee7a6f8fd7a28d2c1f23acd7a3f9b9faf61692 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=.flow/tmp/fn152-8-linux-tools/golangci-lint ERRORTYPE=.flow/tmp/fn152-8-linux-tools/errortype lint-code-gomad3
- PRs:
## Progress (2026-10-13, aborted before acceptance)

Implementation committed, unmerged: `08bff5e6a1` on branch `fn152-inspect` (worktree `/tmp/fn156-final/fn152g-wt`, base `31c34a520d`): "inspect campaign, merged and corpus logs through one read-only replay", 29 files, +1486/−235.

Verification done before the owner aborted the run:
- Portable failing-test names and failing packages identical to baseline (native-toolchain failures only).
- `-race ./runner/internal/...`, architecture (`GOMAD3_ARCH_ALL_PLATFORMS=1`), `make validate` and `lint-code-fast` pass.
- Identity guard empty.

Not done:
- `make test-wasm` was terminated partway (0 failures up to the abort).
- The new tests have not been repeated (`-count=5`).
- The design report and the acceptance-criterion → test mapping are not written yet.

To resume:
1. Rerun `make -C tools/gomad3 test-wasm` and `-count=5` on `08bff5e6a1`.
2. Check the commit against the acceptance criteria.
3. Fast-forward merge, then complete.

Baseline for the next worker: `/tmp/fn156-final/fn152-8/after-fails.txt` (the failing-test set at `08bff5e6a1`).
