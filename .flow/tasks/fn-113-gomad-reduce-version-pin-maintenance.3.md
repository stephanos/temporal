---
satisfies: [R4]
---
# fn-113-gomad-reduce-version-pin-maintenance.3 Refresh invalidated packs in one command and remove unselected variants

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Platform-aware pin/pack behavior, unavailable-platform refusal/unknown handling, measured steps, source reconciliation, Darwin gates and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

One command that runs `discover`, `review`, and `generate` for every request a bump invalidates, stopping at approval, and removal of pack variants nothing selects (R4).

**Size:** M
**Files:** `tools/gomad3/cmd/gomadtool/compatibility_pack.go`, `tools/gomad3/internal/compatibilitypack/authoring/*.go`, `tools/gomad3/internal/compatibilitypack/{packs,requests,reports}/`, `generation.json`, `tools/gomad3/Makefile`
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/internal/compatibilitypack/**, tools/gomad3/Makefile, tools/gomad3/qualification/corpus/**]

### Approach
- Add a `refresh` subcommand beside the existing five. It runs on a checkout where the bump is already applied, so the candidate versions are the ones the working tree resolves. It takes the invalidated request set from the task 1 report run against that checkout.
- Each request is discovered in its own target module. `discover` needs `--working-dir`, and requests name a module and package, not a directory. Move the request-to-directory mapping the Makefile qualification list holds today (repository root, qualification corpus, fixtures) into one checked-in table that both the Makefile target and `refresh` read. A request with no mapping is invalid input.
- Discover into scratch first and compute the review digest of the fresh evidence. Write the request only when its evidence changed. `authoring.Discover` clears approval unconditionally today, so refresh must not call it in place.
- Skip predicate: a request is done when its stored approval matches the review digest of the freshly discovered evidence. An approval of older evidence does not count. Approval stays per request through the existing `generate --approve-review`.
- Depends on task 2: both edit `cmd/gomadtool` and the pack tree, and task 2 reports the libc-bound packs this command repairs.
- A request for another platform is reported as not evaluable on this host and left untouched.
- Stale variants: `modernc-libc-xsys-v041` is selected only by `internal/compatibilitypack/testdata/v041/go.mod`. Before removing a variant, show that no qualified module, corpus module, or test fixture selects it, and retain that evidence. Delete the pack, request, and report together and update the Makefile qualification list.
- fn-105 task 26 rebound stale packs by hand; reuse its audit.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/cmd/gomadtool/compatibility_pack.go:20-34` — subcommand dispatch
- `tools/gomad3/internal/compatibilitypack/authoring/discover.go`, `review.go`, `generate.go` — steps to chain
- `tools/gomad3/Makefile:103-106` — pack qualification list
- `tools/gomad3/internal/compatibilitypack/testdata/v041/go.mod` — the only selector of the v041 variant

**Optional** (reference as needed):
- `tools/gomad3/qualification/corpus/go.mod:13-16` — selector of the v047 variant
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.26.md` — earlier manual rebind

### Key context
- fn-109 task 8 edits `compatibility_pack.go`; check its state first.
- Pack validation rejects any pack admitting `os/exec`, `os/signal`, `os/user`, `plugin`, or `runtime/cgo`; refresh changes none of that.

### Source-progress revival (2026-10-05): refresh status exhaustiveness

Revive only the existing `packPinImpact` status switch at committed candidate `765547c32d2f674993026743fadceb4319e641d8`: the retained integrated lint reports missing `StatusUnaffected` and `StatusNotSelected`. Add explicit no-op cases without changing evaluation, output, error precedence, approvals, pins, variant selection or retirement. Existing task Touches covers this correction; no new owner or requirement is introduced.

Retain focused baseline/final refresh tests and the actual pinned lint before/after, followed by one integrated lint gate on frozen final source and independent source-progress review. Keep task .1/.2 dependencies, every original acceptance criterion, current-source R4 reconciliation (including selected-variant preservation), native Darwin gates and formal review open. Linux execution remains nonblocking under fn-128.4/.7. MILESTONES permits reviewed source progress before predecessor acceptance and requires its own progress commit; this does not qualify or complete the task.

### Verified source checkpoint (2026-10-05)

The refresh switch now explicitly ignores the two unchanged statuses. [The checkpoint](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/refresh-exhaustive-20261005/progress.md) links baseline/final controls, exact source preservation, raw lint and independent source review. Scoped lint changes from 136 to 135 findings and integrated lint from 318 to 317; every remaining diagnostic block is unchanged. Both gates remain red. The inherited pin-impact host refusal remains a failure, original acceptance stays open, and no native qualification or formal SHIP is claimed.

stage: impl-review - skipped(policy: required lint/native/R4 gates remain open; independent source-progress review approved)
stage: plan-sync - skipped(config: disabled; no task completion)
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] `refresh` runs discover and review for every invalidated request in that request's mapped module and stops with one review digest per request
- [ ] A test with requests from two different modules shows discovery used each module's candidate versions; an unmapped request is invalid input
- [ ] Starting from two previously approved, now-invalidated requests: approving one refreshed request and rerunning leaves that one approved and reports only the other
- [ ] An approval that matches older evidence is never treated as current
- [ ] Other-platform requests are reported and untouched
- [ ] Each removed variant has retained evidence that nothing selects it; pack, request, report, and Makefile entry go together
- [ ] `make -C tools/gomad3 validate compatibility-pack-qualification` passes on darwin/arm64; linux status recorded
## Source progress - authoring import lint (2026-10-05)

Seven explicit `compatibility` import aliases repair all seven current authoring goimports findings. The imported package already declares that name. All other source bytes, import paths, bodies, assertions, comments, approvals, pins and generated outputs remain unchanged.

The source checkpoint is bound to base `8a8b57e8dc42202e8b5bb3dd974d6f306a913ea2` under `../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-3/import-alias-progress-2026-10-05/`. `checks.json` retains exact commands, exits, raw-log hashes and before/after source hashes. Both baseline and final ordinary authoring, selected CLI and architecture controls pass. The same pinned unfiltered scoped lint changes from seven goimports findings, exit 1, to zero issues, exit 0. Final `make validate` passes on the frozen source because authoring files are generator-validation inputs.

The conductor independently reran ordinary authoring, all five named CLI controls with verbose execution, architecture and actual pinned unfiltered scoped lint. It verified all 19 final source/log/config/patch bindings, seven Git-base hashes and seven byte-preservation comparisons. A fresh same-family Codex source reviewer found no introduced Critical, Important or Minor issue and approved this source-progress commit. See `source-review.md` and `conductor-verification.md`.

The checkpoint retains .1/.2 dependencies, complete current-source R4 reconciliation, original acceptance, native Darwin pack qualification and formal review. Historical Done/Evidence remains historical and byte-unchanged. No native qualification or formal SHIP is claimed. Transferred Linux execution remains under fn-128.4/.7.

stage: impl-review - skipped(policy: required native and broader product gates remain open; bounded source-progress review recorded separately)
stage: plan-sync - skipped(policy: planSync disabled; no accepted task completion)

## Done summary
# fn-113 task 3: pack refresh and stale variant retirement

Source bytes and the qualified Darwin toolchain key are bound in `source-binding.json`; pre-edit bytes and hashes are in `pre-edit-source.json`. No source commit, stage, or Flow lifecycle mutation was made by this worker.

`compatibility-pack refresh --root=... --impact-report=...` consumes invalidated or unknown pack-rule IDs from the task 1 JSON report. `targets.tsv` is the one request-to-module mapping read by refresh and by the Makefile qualification target. Unmapped IDs are invalid input. The command discovers and renders a fresh review before changing a request or report. It keeps an approval only when it equals the newly discovered review digest; changed evidence clears approval. Existing `generate --approve-review` remains the sole approval path. Other-platform requests are reported and left unchanged. Per-request failures return status 1 while other requests continue; invalid input returns 2.

The controlled real CLI fixture under `refresh-fixture/` copied two production requests with stale approvals, mapped `reflect2-go126` to the root module and `modernc-libc-xsys-v047` to the corpus module, and passed an invalidated request set. `real-refresh.log` records fresh digests for both. `real-refresh-state.json` records the root candidate `github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee` and the corpus candidate `golang.org/x/sys@v0.47.0`. `real-partial-approval.log` records approval of only the v047 request through the existing generate command. `real-rerun.log` reports only reflect2. `real-other-platform.log` reports the Linux request as not evaluable; its final bytes equal the copied source request. `real-unmapped.log` records CLI status 2 (wrapped by `go run` as shell exit 1). Production upstream evidence was not approved or changed.

Before retirement, `testdata/v041/go.mod` genuinely selected `golang.org/x/sys@v0.41.0`; the Makefile also qualified its pack, and `evidence_test.go` and `policy_test.go` loaded it. The active corpus at v0.47 already tests `NewTLS`, `CString`, `Xopen`, `Xwrite`, `Xlseek64`, `Xread`, and `Xclose`, plus `Xmkdir`. The exact-pack identity, evidence, capability, source-set, and near-miss assertions now target the v047 pack. The old pack, request, report, and fixture were removed together; the Makefile table contains no v041 entry. `selector-audit.json` scans 16 Go modules and finds no remaining v041 go.mod selector or task-source reference. Linux pack variants remain.

Verification, all on darwin/arm64 with stock Go 1.27.1 for host commands and patched toolchain key `2ecdbd330bb5928f360739fdd80cb3eb0f0764e513a5cc208cd83714fff0a457` for qualification:

- Pre-edit `go test -tags test_dep ./cmd/gomadtool ./internal/compatibilitypack/...`: exit 0, `baseline-focused.log`.
- Shared-table baseline `make validate compatibility-pack-qualification`: exit 0, `table-baseline-qualification.log`.
- Final `go test -tags test_dep -count=1 ./cmd/gomadtool ./internal/compatibilitypack/... ./upgrade`: exit 0, `focused-bound-final.log`.
- `go vet -tags test_dep ./cmd/gomadtool ./internal/compatibilitypack/... ./upgrade`: exit 0, `vet-final.log`. Gofmt and `git diff --check` were clean.
- `make validate compatibility-pack-qualification`: exit 0, `final-validate-qualification.log`; eight Darwin requests qualified. The old v041 request no longer runs.
- Scoped `golangci-lint run --build-tags test_dep --max-issues-per-linter=0 ./cmd/gomadtool ./internal/compatibilitypack/... ./upgrade`: exit 1, `scoped-lint-bound.log`, with 17 existing findings in surrounding packages. No finding points to the new refresh source or tests. Three findings in the edited legacy `compatibility_pack.go` are on output calls outside task 3's changed lines. The unchanged root-lint nested-module discovery failure is retained under task 2 and was not repeated.

Native linux/amd64 qualification was unavailable on this host; no cross-compile result is claimed as qualification. Task 4 owns full core/full-test and both-platform final gates.

Parent verified all16 current path bindings and the review patch digest. Independent same-family codex:gpt-6-sol:high review returned SHIP with R4 met and no findings (working-tree-review.json). The user owns commits; no files staged or committed.
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (sync active=false).

Blocked:
ORIGINAL_QUALIFICATION_OPEN. The refresh status exhaustive finding is repaired by one explicit no-op case. Focused refresh controls pass before and after, and the conductor recheck passes all 11 cases. The pin-impact control retains its exact pre-edit linux/arm64 preparation refusal (three pass, one fail); the build-pin proof is not satisfied here.

Actual full-config scoped lint remains red with 135 errcheck findings. Integrated lint remains red with 317 findings (252 errcheck, two exhaustive, 11 forbidigo, 52 staticcheck); Make exits 2 and never reaches integrated errortype. Literal preservation and complete raw diagnostic comparison confirm only the selected exhaustive finding was removed.

Task .1/.2 acceptance dependencies, full current-source R4 reconciliation including selected-v041 preservation, required native Darwin validate/pack and broader qualification, and formal review remain open. This checkpoint is verified source progress, not completed acceptance. Linux execution remains transferred and nonblocking under fn-128.4/.7.
## Evidence
- Commits:
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./internal/compatibilitypack/... ./upgrade (focused-bound-final.log; exit0), go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool ./internal/compatibilitypack/... ./upgrade (vet-final.log; exit0), make -C tools/gomad3 validate compatibility-pack-qualification (8 Darwin requests; exit0), real CLI refresh in root and corpus mappings; approve one via generate --approve-review; rerun reports only other; Linux request unchanged; unmapped CLI exits2, selector audit:16Go modules, no remaining v041 selectors; retired pack/request/report/fixture and migrated same coverage to v047, codex implementation review: SHIP,R4met,16path bindings including6deletions and patch digest verified, gofmt and git diff --check clean; scopedlint17pre-existing findings none newrefresh files; nativeLinux unavailable, finalgates task4
- PRs:

## Current acceptance blocker (2026-10-05)

Task 3's seven authoring import findings are repaired and its scoped lint, ordinary authoring, selected CLI, architecture and generator validation checks pass. Task-1/task-2 acceptance dependencies, complete R4 current-source reconciliation, required native Darwin validate/compatibility-pack qualification and formal review remain open. Developmental linux/arm64 evidence cannot satisfy those native gates. Missing transferred Linux qualification is not a blocker; fn-128.4/.7 own it.
