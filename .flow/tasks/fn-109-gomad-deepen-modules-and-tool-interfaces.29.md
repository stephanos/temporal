---
satisfies: [R13, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.29 Preserve private artifact payload cleanup and error identity

## Description
Bounded R13/R18/R19 source repair after campaign checkpoint e98c7a3e2ca845b92edcdef185f5c9ae67be4a3a. Preserve original task 12/predecessors and task 21 acceptance; do not force-start or complete them.

**Size:** M
**Touches:** [tools/gomad3/artifact/store.go, tools/gomad3/artifact/store_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/**]
Root separately owns parent/task Flow, MILESTONES, independent review and Git. One source writer in this checkout; read-only scouts may overlap on disjoint source.

Read AGENTS.md, Gomad README and MILESTONES, original fn109 spec/task12, the artifact interface inventory and .flow/tmp/artifact-cleanup-source-mapping.md. Reuse the existing sixteen-site mapping. This owner covers ONLY seven errcheck sites in private copyPayload/writePayload: store.go baseline lines 333,346,352,356,402,406,410. Public Opened.CopyPayload, syncDirectory and verifySharedPayload remain separate owners; keep their bytes unchanged.

Baseline whole ordinary artifact tests and actual unfiltered pinned lint/errortype before production edits. Follow test-driven-development, writing-good-tests, code-style and verification-before-completion. The actual pinned analyzer is the existing source-policy RED for these seven unchecked cleanup returns. Add meaningful real-file primary-error/lifetime controls before implementation, deriving expectations from current behavior and literals; do not invent runtime first-Close fault proof.

Observe nonnil cleanup errors as secondary after the original operation error. When cleanup returns nil, retain the original error object, wording, single/multiple unwrap shape, classification and zero error result metadata. Preserve output-before-input cleanup and file acquisition/release boundaries, context checks, Stat acceptance and copy/hash/count. Success still performs copy/write, context-aware Sync, checked output Close, checked input Close for source copies, then metadata. Retire ownership BEFORE an explicit Close attempt, including one returning an error: copyPayload already closes input on success, so naively joining its deferred second Close would turn publication into os.ErrClosed failure. No double close, retry, caller-moved cleanup, path deletion, new public seam, fake file/framework, broad helper abstraction or injected syscall race.

Keep source/data routing, modes, exclusive destination create, canonical manifest and record identities, fixed-input bytes, no-replace rename, manifest-last ordering, validated reuse, independent target-pool transaction and capacity accounting unchanged. Existing comments and golden expectations retain their original strength. Add narrow same-package real-file controls as needed: already-cancelled standard contexts after file acquisition must preserve exact copy/write context wrappers and single unwrap to context.Canceled, zero record.File and retained partial destinations; successful source/inline operations must retain literal metadata/bytes/modes and not surface a second input close. Existing destination collision retains os.ErrExist/PathError and sentinel bytes; nonregular source returns its literal error. Existing Store tests still own staging removal and fixed-record publication. Do not broaden to new compatibility/availability or resource-cost work.

Use cached stock Go1.27.1 linux/arm64 for developmental verification, GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS='' cleared GOMADSEED/GOMAD3_CHILD_SEED, -count=1 -tags test_dep. Final whole ./artifact package, focused private payload/Store/publication/target-pool/mode/retained-cost controls, and relevant root artifact ownership/architecture/public alias/external Runner boundaries. Inspect generator inputs before edits and disposition make validate against protected hashes; no generated/runtime/protocol/pin/config/dependency changes, tools or downloads. Do not rerun unchanged broad rootfast/full/native failures now.

Pinned local Go: /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go. Analyzers: /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 and errortype. Use unchanged gitroot .github/.golangci.yml, --build-tags test_dep --fix=false ./artifact and errortype -tags test_dep ./artifact. Retain actual before/after raw findings, seven-site mapped delta, every residual/introduced finding and exact terminal commands/env/cwd/start/end/elapsed/exit, tools/config and stable artifact source hashes. No filtering, suppression, rule waiver or borrowed campaign counts. Historical broader 419 is not a fresh whole-scope count.

Reuse the task28 gate-capture pattern under task29 with artifact source selection; retain only meaningful RED/characterization and final receipts, one lean handover/evidence and compact protected aggregates (no repeated bulk inventories). Inspect active pinned os.File.Close implementation; disclose missing genuine first-Close fault coverage rather than adding fake seams. Original R13/R18/R19, task12/predecessors, task21, first-baseline fixed identities, complete/full/formal and patched-runtime native darwin/arm64 and linux/amd64 requirements remain open wherever unproved.

Worker owns only source/test and task29 artifacts; root alone stages/commits, fresh independent reviews and Flow lifecycle. No worktrees/stash/bridge/push/history rewrite or unrelated source writes. Do not call flowctl done or claim formal SHIP. Return only when owned commands and delegates are terminal, with task-unique handover.md/evidence.json, genuine source progress and remaining qualification. Root verifies, reviews and commits before the next writer.

## Acceptance
- [ ] Seven private payload cleanup findings are resolved while source/data routing, original primary error identity/shape, zero metadata on errors and output-before-input release remain correct.
- [ ] Explicit/deferred resource ownership permits exactly one Close attempt per acquired handle and preserves successful Sync/Close/metadata order without publication or pool transaction changes.
- [ ] Baseline/final whole artifact package and meaningful real-file payload/Store/publication/mode/capacity controls retain literal expectations, stable source bindings and terminal command receipts; unavailable first-Close fault proof is explicit.
- [ ] Actual unfiltered pinned lint/errortype establishes the seven-site mapped delta and all residual/introduced findings without rules/config/golden changes; generator and relevant public-boundary obligations are verified or kept open.
- [ ] Fresh independent source review reports no actionable introduced defect, and root commits verified task progress while original R13/R18/R19/task12/predecessors/task21/full/native/formal acceptance stays open.


## Done summary
SOURCE_PROGRESS_ONLY; authoritative task status remains blocked, not done.

Seven private copyPayload/writePayload cleanup returns are checked. Successful cleanup preserves the original primary error object and unwrap shape. Deferred output-before-input release preserves error traversal order; ownership is retired before explicit Close attempts so failed Close is not retried and the old second input Close cannot turn successful publication into os.ErrClosed. Source/data routing, literal metadata, modes, exclusive create, partial destinations, Store staging cleanup, publication and independent pool transactions retain their existing contracts.

The five new real-file characterization tests passed against baseline production before the implementation. The actual source-policy RED was pinned baseline errcheck, not a claimed failing first-Close runtime regression. Worker and independent reviewer ordinary package, focused controls, architecture/public-alias/signature/external Runner boundaries, pinned errortype and static checks pass on stock Go 1.27.1 linux/arm64. The protected generator inputs/outputs and successful make validate receipt bind this source; review reuses that receipt. Actual unfiltered artifact lint remains red at 21 after exactly seven mapped repairs from 28, with zero introduced findings. No new whole-Gomad count is inferred from the historical 419.

Fresh independent source review found zero introduced Critical, Important or Minor findings and permits SOURCE_PROGRESS_COMMIT_ONLY. Root revalidated all 18 worker/review receipt bindings and 1,042 protected files. This is not formal SHIP. The historical TestRecordAndArtifactHaveSeparateOwners name is absent; the five actual executed boundary tests are retained in the review checks. Genuine first-Close and simultaneous primary/output/input cleanup-failure runtime proof is absent. Original R13/R18/R19, task 12/predecessors, task 21, matched first-baseline fixed identities, complete/full/formal and both patched-runtime native qualifications remain open.

stage: impl-review - skipped(policy: conductor-deferred; fresh independent source review passed, but full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

Root commits this verified source progress with its evidence before admitting the next source writer. Worker/reviewer commands and delegates are terminal; no push, task-done event, qualification closure or compatibility expansion.

## Evidence
- Commits: 8ac436447572c55d494ca4c2088f6ba441db16db (reviewed source progress only; no task completion).
- Tests: [worker handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/handover.md), [worker evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/evidence.json), [independent source review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/independent-source-review.md), [review checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/independent-source-review-checks.json), [root verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/source-checkpoint-verification.json).
- Open qualification: [acceptance-open.md](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-29/acceptance-open.md).
- PRs: none; no push authorized.
