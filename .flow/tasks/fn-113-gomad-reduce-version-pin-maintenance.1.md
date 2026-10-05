---
satisfies: [R1, R2]
---
# fn-113-gomad-reduce-version-pin-maintenance.1 Baseline the pins and add the pin impact report

## Description

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

## Acceptance

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

## Done summary
The retained rebased-source baseline measures1044 runtime patch lines in20 files,61 overlay files/18437 lines,15 adapters/135 SHA256 anchors,12 packs/54 rules/19 module-version pins,131 interceptions/132 fingerprints, and25 clock references. MILESTONES maintenance counts are corrected; baseline.json includes the historical fn-110 baseline, source hashes and per-bump commands/hand edits.

The new gomadtool pin-impact command reads candidate go.mod/go.sum and the same adapter registry and validated pack descriptors as the build. Canonical path-free JSON and human output list adapters, source-set-bound pack rules, both-platform interception fingerprints and clock references. Exact immutable module version/sum identities need no resolution or downloads; missing sums are unknown. Removed/replaced modules, changed sums, indirect bumps, unselected pack variants, immutable inputs, status0/1/2/3 and output failure are tested. A sentry/reflect2 fixture invalidates exactly the adapter and pack the fail-closed build rejects.

The final review fixed missing module-directive validation, retaining a CLI red status0 and green status2. Full cmd/gomadtool and upgrade tests and scoped vet pass after that three-line guard. The prior shared native45-package host gate and validate pass on unchanged runtime/generated inputs; see fn-114/task-13/integrated-source-hashes.json for the full-gate source snapshot plus supplemental fix hashes. Independent re-review returned SHIP at fn-114/task-13/integrated-review.json. Linux remains unverified; root lint cannot load nested-module paths. No implementation commits or pushes; the user owns commits. Adapter regeneration and pack refresh remain later tasks.
## Evidence
- Commits:
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/..., make -C tools/gomad3 validate, gomadtool pin-impact --format json on current rebased root, go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool ./upgrade/...
- PRs:
