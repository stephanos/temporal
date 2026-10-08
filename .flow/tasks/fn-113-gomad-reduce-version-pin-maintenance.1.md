---
satisfies: [R1, R2]
---
# fn-113-gomad-reduce-version-pin-maintenance.1 Baseline the pins and add the pin impact report

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Platform-aware pin/pack behavior, unavailable-platform refusal/unknown handling, measured steps, source reconciliation, Darwin gates and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Re-measure the pin baseline (R1) and add a `gomadtool` subcommand that reports every pin a candidate `go.mod` invalidates (R2). This is the spec's early proof point.

**Size:** M
**Files:** new subcommand file under `tools/gomad3/cmd/gomadtool/`, `tools/gomad3/cmd/gomadtool/main.go`, a new report package or an addition to `tools/gomad3/upgrade/`, `tools/gomad3/deterministicio/adapter_registry.go` (read access only), fixtures, `MILESTONES.md`
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/upgrade/**, tools/gomad3/deterministicio/**, tools/gomad3/internal/compatibilitypack/**, tools/gomad3/architecture_test.go, MILESTONES.md, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/**]

### Approach
- Baseline: count each pin class and list the commands and hand edits one bump of each needs today. Correct the counts in the milestones Maintenance cost section if they differ. fn-110 task 1 already baselined patch and overlay counts; reuse them.
- Report inputs: one candidate `go.mod` with its `go.sum`, default the repository root module.
- Read adapter identities and anchors through the adapter registry and pack rules through the pack loader, so the report uses the values the build checks.
- Output path-free canonical JSON through the existing canonical encoder plus a human rendering. Exit 0 for no invalidated pin, 1 for at least one, 2 for invalid input, 3 for infrastructure failure. A pin that cannot be evaluated is reported unknown and counts as invalidated.
- Resolve modules outside the target module with a private module cache; `go mod download` inside a target module rewrites its `go.sum`.
- Cover: version bump, same version with a changed sum, module removed or replaced, and an indirect-only bump.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/deterministicio/adapter_registry.go:75` — registry construction from version data
- `tools/gomad3/deterministicio/sentry_adapter.go:7-18`, `:59-65` — anchor constants and definition shape
- `tools/gomad3/toolchain/version/version.json` — adapter module identities
- `tools/gomad3/cmd/gomadtool/main.go:22` — subcommand list
- `tools/gomad3/upgrade/upgrade.go:189` — dossier, the nearest existing report

**Optional** (reference as needed):
- `tools/gomad3/internal/compatibilitypack/generation.json` — pack output digests
- `.flow/memory/bug/integration/go-mod-download-inside-a-target-module-2026-09-28.md`
- `tools/gomad3/internal/canonicaljson` — canonical encoder

### Key context
- fn-105 task 8 adds downstream adapters and changes the baseline count; record the commit the baseline was taken at.
- A new package must satisfy the import allowlist in `architecture_test.go`.

## Source correction: explicit pack-root diagnostics

fn-113 R2 requires path-free canonical pin-impact reports. The explicit PacksDirectory loader can return an os.ReadDir error containing the absolute selected directory, while evaluatePacks redacts only the environment-selected root. Reproduce with an existing regular file supplied where a pack directory is expected, at two otherwise equivalent temporary locations. Missing directories intentionally hold no packs; preserve that behavior.

**Touches:** [tools/gomad3/upgrade/pinimpact/pinimpact.go, tools/gomad3/upgrade/pinimpact/packs_directory_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/explicit-pack-path-progress/**]

Use the actual selected root during error rendering and replace an explicit root with a stable logical label. Preserve environment-selected report bytes, explicit-root precedence, unknown pin classification and invalidated disposition, baseline and candidate validation ordering, pack loading/selection behavior, CLI grammar/statuses, schemas, policies, module files, generated outputs and toolchain inputs. No dependencies or generic path sanitizer are needed.

Use TDD: retain a failing public Evaluate/Encode/Render regression before production edits. Two same-content regular files under different roots must produce equal canonical/human reports containing neither host root, with one unknown compatibility-pack pin and Invalidated=true. Check unchanged missing-root empty selection and existing environment precedence, malformed-pack and module-validation controls. Run package tests, scoped lint/errortype/vet, source-bound check-only validate, relevant architecture checks and make lint-code-fast with pinned tools/fixes disabled. Keep the three original unsupported-host preparation comparisons and required native/full/formal gates explicitly open.

Retain compact RED/GREEN and final command evidence, a fresh independent source-progress review, and a separate verified-progress commit. This fixes source behavior but does not complete task1's original native acceptance. Linux remains deferred and unverified under fn128. Use the existing task1 rather than adding a duplicate owner.

## Source correction: missing pack activation sums

fn-113 R2 and the README require missing module identities to report unknown and count as invalidated. Investigate whether a missing zip checksum for an otherwise exact pack activation module produces not_selected when the baseline shares that omission, and whether a selected baseline yields an empty unknown reason.

**Touches:** [tools/gomad3/upgrade/pinimpact/pinimpact.go, tools/gomad3/upgrade/pinimpact/*_test.go, tools/gomad3/cmd/gomadtool/pin_impact_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-sum-unknown-progress/**]

Retain a public Evaluate/Encode/Render failing regression before production edits. Use validated checked-in packs and real immutable module files with only an activation module's zip sum omitted, retaining its /go.mod sum. Candidate uncertainty counts as unknown only when every activation identity matches or remains unknown. Preserve fully resolved dispositions and reason bytes. Mixed missing-sum and known-exclusion inputs currently hide known exclusions or emit empty unknown reasons depending on activation order; restore the existing resolved-exclusion disposition and reason for those mixed inputs. Baseline uncertainty alone must preserve behavior for a fully known repaired candidate. Cover both missing baselines/candidates, candidate-only missing sums, exact controls, repaired candidates and packs excluded by another known activation identity.

If the premise reproduces, repair only this selection and diagnostic propagation. Preserve exact resolved report bytes, pack validation, replacement policy, schemas, CLI grammar/status mappings, module immutability, pins, generated files, toolchain inputs, native guards and existing comments. Do not add dependencies or qualification claims. Keep the three original native preparation comparisons, required full/Darwin acceptance and formal review open. Linux remains deferred under fn128.

Validated packs may bind a rule module separately from their activation modules. An uncertain activation must not override a definitely absent or mismatched rule identity. Cover these cases through a validated external pack. Preserve stale-rule and baseline-unselected rule dispositions; mixed uncertain activation and known rule mismatches follow the resolved rule-exclusion disposition and reason.

Run focused and portable package tests with test_dep, unfiltered scoped lint, vet/errortype, check-only validate, relevant architecture checks and mandatory make lint-code-fast with fixes disabled. Retain RED/GREEN, final source-bound commands and a fresh same-family source-progress review, then commit verified progress separately. Preserve every original Acceptance, Done summary and Evidence entry.

## Source correction: missing rule-only pack sums

fn-113 R2 requires unresolved pin identities to report unknown and count as invalidated. The shipped golang-x-sys-v047-darwin-arm64 pack activates on x/sys but has an x/term rule. An otherwise exact activation with the rule's zip sum missing in both baseline and candidate currently reaches the baseline filter and hides that unknown as not-selected.

**Touches:** [tools/gomad3/upgrade/pinimpact/pinimpact.go, tools/gomad3/upgrade/pinimpact/*_test.go, tools/gomad3/cmd/gomadtool/pin_impact_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-rule-sum-progress/**]

Retain actual RED through public Evaluate/Encode/Render and the real CLI offline resolver before production edits. Use the unchanged checked-in x/sys pack and real immutable module files. Omit only x/term's zip sum, retaining its /go.mod checksum and the exact x/sys activation identity. Candidate rule uncertainty with known matching activation must be actionable independent of baseline rule selection. Keep known activation exclusions ahead of that uncertainty, existing activation-unknown and graph-reason precedence, baseline validation, exact resolved report bytes, rule absence/version/sum/replacement exclusions, repaired candidates, input snapshots and status mappings.

Repair only evaluation ordering and existing reason propagation if the premise reproduces. Add no dependency, platform rewrite, schema, policy, pin, generated-input or native-guard change. Run focused and portable package controls with test_dep, scoped lint/errortype/vet, check-only validation, relevant architecture checks and make lint-code-fast with fixes disabled. Retain compact source-bound receipts and a fresh same-family source-progress review; commit verified progress separately. This does not complete original native comparator, Darwin/full or formal acceptance. Linux remains deferred under fn128. Preserve original Acceptance, historical Done summary and Evidence.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Baseline of pin classes, counts, and per-bump manual steps retained under the spec's artifacts directory; milestone counts corrected if different
- [ ] The report lists invalidated adapters, pack rules, interception fingerprints, and clock-inventory references for a candidate `go.mod`
- [ ] A fixture bump of one adapted and one packed module yields exactly the expected entries, and the build's fail-closed check rejects the same pins
- [ ] Changed sum at the same version, removed module, replaced module, and indirect-only bump each have a test
- [ ] The target module's `go.mod` and `go.sum` are unchanged after a run
- [ ] Unknown pins are reported unknown, never unaffected; exit statuses follow 0/1/2/3
- [ ] `go -C tools/gomad3 test -tags test_dep ./cmd/gomadtool ./upgrade/...` and `make -C tools/gomad3 validate` pass

## Source progress — portable fixtures (2026-10-05)

This checkpoint repairs the private upgrade-dossier unit fixture and strengthens the pin-impact build acceptance assertion; it does not complete this task. The synthetic descriptor declares the executing host only inside its temporary fixture. A separate excluded-host fixture proves publication with the actual host identity, Supported=false and Qualified=false despite approved boundary, checked corpus and passed gate prerequisites. Production supported platforms, pins, profile identities and build refusal remain unchanged.

The adapter positive control now requires the actual pinned-module download error and the wrapped fork/exec ENOENT for its own bin/go. Its regression rejects unsupported-host refusal and unrelated missing-file, operation and toolchain errors rather than treating them as adapter acceptance. Retained RED/GREEN evidence covers the four original dossier failures and the former predicate's false positives.

Fresh stock-Go development checks pass for the entire upgrade package, the assertion regression, generated validation, scoped vet, architecture and diff checks. The required task Quick still fails three original pin/build comparator tests before their identity checks because production refuses linux/arm64. Unfiltered scoped lint retains six pre-existing findings on unchanged lines. These failures remain failures; no native runtime or Darwin qualification is claimed. R6's native/full qualification remains with task4, and deferred Linux execution with fn-128.

See [source-bound observations](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/portable-fixture-progress/observations.json) and [fresh source-progress review](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/portable-fixture-progress/source-review.md). The reviewer found no blocking issue in this test-only checkpoint; writer and reviewer are both Codex family. Root commits this verified progress before another implementation task. Earlier Done/Evidence below are historical and preserved.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer/reviewer routes retained.
stage: impl-review - skipped(policy: required task Quick and scoped lint red; source-progress approval is not formal SHIP)
stage: plan-sync - skipped(empty: no completed task; planSync.enabled=false)

## Source progress — pin matching and lint repair (2026-10-05)

The report now rejects identical duplicate zip-sum records for adapters and their dependent pack rules, matching the actual private build checksum checker. A shared literal fixture independently exercises that checker and the public report, including exact, separate /go.mod, identical duplicate, changed duplicate and missing-sum controls. Pack-only duplicate selection is preserved. Production adapter_registry.go is byte-identical to the preceding checkpoint; the three original preparation-endpoint comparisons remain intact.

The six scoped lint findings are repaired. Rendering failures are returned separately from gate evidence after publication, retaining dossier bytes, qualification status, later gate execution and primary-error precedence. Real gzip cleanup failures clear collision evidence; nil cleanup preserves the original result/error. Valid raw legacy NUL tar headers preserve regular-file collision, noncollision and trailing-slash directory behavior. The initial fixture setup failure is disclosed and earns no behavioral regression claim; valid pre-removal characterization is retained separately.

Focused regressions, the entire upgrade package, validation, scoped vet, architecture and source/document diff checks pass. Unfiltered pinned scoped lint reports zero issues. The required task Quick still fails only the three original unsupported-host comparator tests. This checkpoint does not complete acceptance, establish a native determinism bound, or qualify Darwin or Linux. The preceding fixture checkpoint's six lint findings are historical and superseded by this repair.

See [source-bound observations](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/lint-and-pin-progress/observations.json), [fresh conductor verification](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/lint-and-pin-progress/conductor-verification.json) and [source-progress review](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/lint-and-pin-progress/source-review.md). Independent fresh-context Sol/high review found no source issues and reran the focused regressions. Writer and reviewer are both Codex family. Historical Done/Evidence below remain unchanged.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer/reviewer routes retained.
stage: worker - ran (model: gpt-6.1-sol at high)
stage: impl-review - skipped(policy: required task Quick red; source-progress approval is not formal SHIP)
stage: plan-sync - skipped(empty: no completed task; planSync.enabled=false)

## Current acceptance blocker (2026-10-05)

Required task Quick still exits 1 on TestFixtureBumpMatchesBuildRejections, TestSameVersionWithChangedSum and TestReplacedModules: production refuses the developmental linux/arm64 host before the real pin/build endpoint comparisons. All six scoped lint findings are now repaired, and source-bound duplicate-sum, rendering, cleanup and legacy archive regressions pass, but their portable proof does not replace those original comparisons. Run the unchanged explicit task command on supported Darwin and retain passing real endpoint evidence, then obtain formal implementation review before completing task1. See .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/lint-and-pin-progress/ for current-source receipts and fresh source-progress approval. Production platforms, pins and the registry implementation remain unchanged. Native runtime/full qualification under fn-113 R6 remains with task4; transferred Linux execution belongs to fn-128 and is not this blocker.

## Source progress - missing pack sums (2026-10-07)

Pin-impact now reports missing activation zip sums as unknown with a module-specific explanation, including shared baseline/candidate omissions and an excluded baseline with a potentially selectable candidate. The real CLI regression changes status 0 to status 1 while preserving both input module snapshots. Candidate uncertainty no longer hides known activation or independent rule exclusions. Mixed-input classification and empty-reason corrections are intentional; nineteen candidate-resolved canonical/human control pairs retain their baseline bytes.

The worker's portable selection passes 29 top-level tests and 82 test/subtest records, with zero skips. The conductor and fresh same-family Codex reviewer independently pass six focused tests and 40 records each. Scoped pinimpact lint stays clean. Final expanded CLI/pinimpact lint remains red at 129 unchanged diagnostic blocks; mandatory fast lint passes changed lines while filtering 302 configured findings. Vet, errortype, check-only validation, four architecture checks, formatting and diff checks pass. Setup failures remain disclosed separately.

See [handover](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-sum-unknown-progress/handover.md), [conductor proof](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-sum-unknown-progress/conductor-proof.json) and [source review](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-sum-unknown-progress/source-review.md). Earlier explicit-root diagnostics and other checkpoint receipts remain historical source progress. Original native comparator, Darwin/full and formal acceptance remains open. Linux remains deferred and unverified under fn128. Original Acceptance, Done summary and Evidence below retain their meaning.

stage: worker - ran (model: gpt-6.1-sol at high)
stage: impl-review - skipped(policy: original task Quick/native/full acceptance remains red or unavailable; source-progress review grants no formal SHIP)
stage: plan-sync - skipped(empty: no completed task; planSync.enabled=false)

## Source progress - missing rule-only pack sums (2026-10-07)

Pin-impact now keeps an unresolved x/term rule visible when the shipped x/sys pack's activation matches, even if the baseline rule sum is missing, absent or bumped. The five-line ordering guard returns the existing module_sum_missing reason and makes the real CLI return status 1 in text and JSON. Actual pre-edit and exact-base-overlay RED reproduce eight failing subcases; the same frozen tests pass after the guard. Forty-three canonical/human control pairs retain their base bytes, including known exclusions and repaired candidates.

Portable pinimpact and all RunPinImpact CLI tests pass 32 top-level tests and 131 test/subtest records with zero failures or skips. The conductor and fresh same-family Codex reviewer each independently pass nine focused tests, 89 records and 70 leaves. Expanded scoped lint remains red at 129 identical baseline/final diagnostic blocks; pinimpact has zero scoped findings. Mandatory fast lint passes changed lines while filtering 302 configured findings. Vet, errortype, check-only validation, four architecture checks, formatting, diff and Flow validation pass. The reviewer's timer setup failure and the conductor's corrected task-admission/read-only lookup errors remain disclosed and earn no behavioral proof.

See [handover](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-rule-sum-progress/handover.md), [conductor proof](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-rule-sum-progress/conductor-proof.json) and [source review](../artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-rule-sum-progress/review.md). Earlier receipts remain historical source progress. Original native comparator, Darwin/full and formal acceptance remains open. Linux remains deferred and unverified under fn128. Original Acceptance, historical Done summary and Evidence retain their meaning.

stage: worker - ran (model: gpt-6.1-sol at high)
stage: impl-review - skipped(policy: original task Quick/native/full acceptance remains red or unavailable; source-progress review grants no formal SHIP)
stage: plan-sync - skipped(empty: no completed task; planSync.enabled=false)
Tracker sync: n/a (bridge inactive)

## Done summary
The retained rebased-source baseline measures1044 runtime patch lines in20 files,61 overlay files/18437 lines,15 adapters/135 SHA256 anchors,12 packs/54 rules/19 module-version pins,131 interceptions/132 fingerprints, and25 clock references. MILESTONES maintenance counts are corrected; baseline.json includes the historical fn-110 baseline, source hashes and per-bump commands/hand edits.

The new gomadtool pin-impact command reads candidate go.mod/go.sum and the same adapter registry and validated pack descriptors as the build. Canonical path-free JSON and human output list adapters, source-set-bound pack rules, both-platform interception fingerprints and clock references. Exact immutable module version/sum identities need no resolution or downloads; missing sums are unknown. Removed/replaced modules, changed sums, indirect bumps, unselected pack variants, immutable inputs, status0/1/2/3 and output failure are tested. A sentry/reflect2 fixture invalidates exactly the adapter and pack the fail-closed build rejects.

The final review fixed missing module-directive validation, retaining a CLI red status0 and green status2. Full cmd/gomadtool and upgrade tests and scoped vet pass after that three-line guard. The prior shared native45-package host gate and validate pass on unchanged runtime/generated inputs; see fn-114/task-13/integrated-source-hashes.json for the full-gate source snapshot plus supplemental fix hashes. Independent re-review returned SHIP at fn-114/task-13/integrated-review.json. Linux remains unverified; root lint cannot load nested-module paths. No implementation commits or pushes; the user owns commits. Adapter regeneration and pack refresh remain later tasks.

Blocked:
The explicit pack-root diagnostic correction has verified source progress. The selected directory now renders as $PacksDirectory in canonical and human reports. The public regression reproduced host-path leakage before the production edit and passes afterward. Missing-root empty selection, explicit/environment precedence, malformed-pack reporting, validation order and exact environment-selected report bytes are preserved.

Portable checks pass across four packages, with 85 top-level tests and 157 passing test/subtest records. The command explicitly excludes TestFixtureBumpMatchesBuildRejections, TestSameVersionWithChangedSum and TestReplacedModules because their original comparisons require the unavailable patched driver. Their acceptance remains open. Scoped unfiltered lint, vet, errortype, check-only generator validation and four architecture checks pass. Mandatory fast lint passes for changed lines while filtering 310 residual configured findings. Full lint remains red.

Evidence is retained under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/explicit-pack-path-progress/. The conductor independently reran the four report regressions, matched environment report bytes and checked the two-file diff and final source hashes. A fresh same-family Codex source-progress review supplies no formal SHIP or task-completion verdict.

Original task1 build-pin comparisons and required full/native-Darwin/formal gates remain open. This developmental Linux ARM host and absent patched toolchain cannot provide them. Do not repeat unchanged unsupported-host or missing-driver checks. Linux qualification remains deferred and unverified under fn128 and does not block this source correction. Resume original acceptance with source-bound native Darwin/full/comparator evidence. Historical qualification receipts remain historical evidence, not current-candidate proof.

### Historical implementation summary retained from the pre-correction task

The new gomadtool pin-impact command reads candidate go.mod/go.sum and the same adapter registry and validated pack descriptors as the build. Canonical path-free JSON and human output list adapters, source-set-bound pack rules, both-platform interception fingerprints and clock references. Exact immutable module version/sum identities need no resolution or downloads; missing sums are unknown. Removed/replaced modules, changed sums, indirect bumps, unselected pack variants, immutable inputs, status0/1/2/3 and output failure are tested. A sentry/reflect2 fixture invalidates exactly the adapter and pack the fail-closed build rejects.

The final review fixed missing module-directive validation, retaining a CLI red status0 and green status2. Full cmd/gomadtool and upgrade tests and scoped vet pass after that three-line guard. The prior shared native45-package host gate and validate pass on unchanged runtime/generated inputs; see fn-114/task-13/integrated-source-hashes.json for the full-gate source snapshot plus supplemental fix hashes. Independent re-review returned SHIP at fn-114/task-13/integrated-review.json. Linux remains unverified; root lint cannot load nested-module paths. No implementation commits or pushes; the user owns commits. Adapter regeneration and pack refresh remain later tasks.

Blocked:
The missing-pack-sum correction has verified source progress. Public reports now expose shared missing zip checksums as unknown rather than hiding actionable pack rules, and supply module-specific explanations. The real CLI regression changes status 0 to status 1 without mutating either input module. Known activation or independent rule exclusions retain their resolved-exclusion semantics; old mixed-input empty unknown or hidden not-selected results intentionally change. Nineteen candidate-resolved canonical/human control pairs match the base production overlay.

Portable selection passes 29 top-level tests and 82 test/subtest records, with zero failures or skips. The conductor and fresh same-family Codex reviewer independently pass six focused tests and 40 records each. Scoped pinimpact lint remains clean; final expanded CLI/pinimpact lint remains red at 129 identical diagnostic blocks. Mandatory fast lint passes changed lines while filtering 302 configured findings. Full lint remains red. Vet, errortype, check-only validation, four architecture checks, formatting and diff checks pass. Setup failures are retained and earn no behavioral proof.

Evidence is retained under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-sum-unknown-progress/. This checkpoint preserves the earlier explicit-root path correction and its evidence. The fresh source-progress review is neither formal SHIP nor task completion.

Original TestFixtureBumpMatchesBuildRejections, TestSameVersionWithChangedSum and TestReplacedModules remain unchanged and unproved, along with required native Darwin/full/formal gates. The developmental linux/arm64 host and absent patched driver cannot supply them. Do not repeat unchanged unsupported-host failures. Linux remains deferred and unverified under fn128 and does not block this source correction. Historical acceptance receipts remain historical evidence, not current-candidate native proof.

Blocked:
The rule-only checksum correction has verified source progress. Candidate rule uncertainty with a known matching activation is now unknown and actionable independently of baseline selection. The shipped x/sys pack and real offline CLI prove the missing x/term sum changes status 0 to status 1 without changing either input module. Frozen exact-base RED reproduces eight failing subcases; final GREEN and 43 canonical/human control pairs preserve known exclusions and repaired-candidate bytes.

Portable selection passes 32 top-level tests and 131 records with zero failures or skips. Conductor and independent fresh same-family Codex review each pass nine focused tests, 89 records and 70 leaves. Scoped pinimpact lint stays clean; expanded CLI/pinimpact lint remains red at 129 byte-identical diagnostic blocks, and full lint remains red with 302 configured findings. Mandatory fast lint passes changed lines. Vet, errortype, check-only validation, architecture, formatting, diff and Flow checks pass. Setup mistakes are disclosed separately.

Evidence is under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/pack-rule-sum-progress/. Source-progress review is neither formal SHIP nor task completion. Original TestFixtureBumpMatchesBuildRejections, TestSameVersionWithChangedSum and TestReplacedModules remain unchanged and unproved. Required native Darwin/full/formal gates remain open. Do not repeat unchanged unsupported-host or missing-driver checks. Linux remains deferred and unverified under fn128 and does not block this source correction. Historical receipts remain historical, not current-candidate native proof.

## Evidence
- Commits:
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/..., make -C tools/gomad3 validate, gomadtool pin-impact --format json on current rebased root, go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool ./upgrade/...
- PRs:
