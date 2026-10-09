---
satisfies: [R6, R7, R11]
---
# fn-112-gomad-determinism-assurance-and-test.10 Add the scheduled determinism soak gate and update the docs to the delivered state

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.5](../tasks/fn-128-gomad-deferred-linux-qualification-and.5.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Actual Darwin soak report/ledger/measured bound, cohort/diagnostic controls and shared documentation. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

The soak gate (R6) and final documentation (R11). Combined because the docs report the bound the soak measures.

**Size:** M
**Files:** `Makefile` (root), `tools/gomad3integration/qualification/` (a soak manifest), `.github/workflows/gomad3.yml`, `tools/gomad3integration/README.md`, `tools/gomad3/README.md`, `SPEC.md`, `ARCHITECTURE.md`, `TUTORIAL.md`, `MILESTONES.md`
**Touches:** [Makefile, tools/gomad3integration/**, .github/workflows/gomad3.yml, tools/gomad3/*.md, MILESTONES.md, AGENTS.md, tools/gomad3/qualification/**, tools/gomad3/cmd/gomadtool/soak.go, tools/gomad3/cmd/gomadtool/soak_test.go]

### Approach
- Soak selection: the smoke suites plus one guarded-mode workload, seeds 11 and 17, choice tracing and diagnostics on, with unrelated CPU load from the existing load helper or an equivalent host-side loader.
- First measure per-run cost, then choose the largest N per seed that fits one scheduled job by looping `qualify` at its 32-repetition bound. Record N and the reasoning.
- Each `qualify` invocation compares only against its own first execution. The soak therefore compares the evidence baseline across batches within one cohort (workload, seed, platform, execution identity) and fails when two batches disagree. Cumulative counts across scheduled runs are aggregated per cohort only, with the cohort's baseline evidence retained, and a changed toolchain identity starts a new cohort.
- Test the aggregation with a batch sequence A, A then B, B: it must be a divergence.
- The gate accepts zero divergences. A trace overflow or infrastructure failure is reported separately and is not a pass.
- On a divergence, upload both diagnostic traces and the differ output.
- The report states repetitions, seeds, load, platform, toolchain identity, and a cumulative count across retained scheduled runs.
- Linux soak runs and reports belong to fn-128.5 and fn-128.7. linux/amd64 stays informational until fn-128.2 closes D12, and the report says so. Coordinate the deferred Linux workflow edit with fn-128.2, which owns removal of the D12 allowances; missing transferred Linux evidence does not block this task.
- Docs: take the flag and subcommand names, the gate name, the inventory location, the declared-differences table, and the drafted contract sentences from the done summaries of tasks 2, 4, 5, 6, and 7. Update the milestones quality-assessment section, README Contract and Development, SPEC, ARCHITECTURE choice-trace section, and TUTORIAL determinism section to the delivered state, including the measured bound. Rerun fn-111's manual link and command-inventory checks; they have no Make target. Keep the `MILESTONES.md` headings `Quality assessment (2026-10-01)`, `Maintenance cost`, and `Constraints`, which specs link to.

### Investigation targets
**Required** (read before coding):
- `Makefile:192-222` — root qualification targets
- `.github/workflows/gomad3.yml:23-24`, `:136-161`, `:245-282` — cron and schedule-gated steps
- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go:295` — CPU load helper
- `tools/gomad3/cmd/gomad/internal/cli/qualify.go:20-105` — repeat bound
- `tools/gomad3integration/qualification/smoke.json` — selection to copy

**Optional** (reference as needed):
- `tools/gomad3/TUTORIAL.md:588-623` — determinism and qualification text
- `MILESTONES.md` section `Quality assessment (2026-10-01)`

### Key context
- The bound is measured with diagnostics on; say so wherever it is quoted.
- Spec Open Question 1 (target bound and runners) is unanswered; deliver the cumulative report and state the count reached.

### Source acceptance admission (2026-10-09)

The owner prioritized near-complete specs. Resume task 10 at committed source base `15f56644664f3d3749bab2387aa97936a1cac6dd` under [the root admission](../artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/source-acceptance-20261009/admission.md). Existing acceptance, scope, dependency edges and historical evidence remain unchanged. Native obligations stay deferred under fn-149/fn-128. Root owns lifecycle, independent review and commits. Current source gates must pass before Done; documentation-only gaps do not license weakening tests or retrying unchanged failures.

### Diagnostic-write continuation admission (2026-10-09)

Root admits exactly the four task-owned soak diagnostic checks and additive adapter controls under [the bounded admission](../artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/diagnostic-writes-20261009/admission.md), at base `4de2ba7892570a865c27a178b681e31f28fca79b`. This corrects the adapter omission in Touches without changing acceptance, dependency edges or semantics; all other command adapters remain outside this repair. Existing source/native requirements and historical evidence stay in force.

### Remaining source-gate routing — 2026-10-09

The [continuation audit](../artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/continuation-20261009/handover.md) rebinds nine final gate receipts to committed source `eb82ea59a4`, with six direct predecessors Done. Remaining affected lint has 59 diagnostic writes now owned by [fn-109.52](fn-109-gomad-deepen-modules-and-tool-interfaces.52.md). The original-base integrated lint remains RED204 with errortype unreached, under fn-109 correction owners and task21's reconciliation. Cross-spec task edges are unsupported, so this source prerequisite is referenced here; existing dependency edges are unchanged. Formal task10 review follows green retained source gates. Native fn149/fn128 obligations remain deferred and unverified.

### Current source-gate routing (2026-10-09, builder correction)

Root keeps task10 as the near-complete spec close-out target. The earlier RED204 continuation audit retains its historical source scope. Task52 is Done; tasks53/54/55 retain reviewed source-progress commits. Task56 now measures original-base RED95 to RED80, exactly15 builder cleanup findings removed and zero added, in [its corrected packet](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-56/handover.md). Its configured toolchain lint retains the competing-build test sleep, and original-base integrated errortype remains unreached. This evidence does not complete task10 or transfer any source requirement.

Root admits disjoint correction tasks fn-109.57 (16 test cleanups), fn-109.58 (two production cleanups) and fn-109.59 (two explicit test-switch no-op cases) to reduce the same required source gate. Those repairs are pending; their expected reductions supply no measured result. Isolated worktrees permit concurrent implementation while root serializes shared gates and verifies the integrated target. Formal task10 source review and completion follow green retained source acceptance. Native fn149/fn128 remain deferred and unverified, with no workflow dispatch, PR, push or CI authority.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.5](../tasks/fn-128-gomad-deferred-linux-qualification-and.5.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] A scheduled darwin/arm64 job runs the soak with stated N, seeds, load, and identities, and fails on any divergence; the corresponding Linux job execution and report belong to fn-128.5 and fn-128.7
- [ ] Darwin-owned shared documentation identifies fn-128.5 and fn-128.7 as the Linux job/report owners and fn-128.2 as the D12 owner; the Linux informational result while D12 is open is verified by those deferred owners
- [ ] A divergence retains both traces and the differ output as workflow artifacts
- [ ] Overflow and infrastructure failures are reported separately from divergence
- [ ] Batches of one cohort are compared against each other; an A, A then B, B sequence is reported as a divergence by a test
- [ ] Cumulative counts are per cohort, and a new toolchain identity starts a new cohort
- [ ] One completed scheduled or dispatched darwin/arm64 run is retained with its report; the corresponding Linux run and report belong to fn-128.5 and fn-128.7 and do not block this task
- [ ] Milestones, README, SPEC, CLI.md, ARCHITECTURE, TUTORIAL, and `AGENTS.md` describe the delivered gate, localiser flag and differ, inventory, fixtures, closure-mode limit, declared differences, and measured bound; links and command inventories checked
- [ ] `make -C tools/gomad3 validate` passes
## Done summary
Blocked:
Blocked: the soak gate and contract docs are delivered and reviewed (SHIP, claude-fable-5-1, round 2). Two items remain, and both need native CI: "one completed scheduled or dispatched run per platform is retained with its report", and the native measured bound that run supplies.

Done (commits bae373d14, 204732c33, 8ecc7e584, 4efa98213 on gomad-fn112; base 331b75bb6):
- `gomadtool soak` (tools/gomad3/qualification/soak) runs `qualify --diagnostics` batches of 32 under 2 busy host threads.
  - Per-cohort comparison: workload, seed, platform, and execution identity (a new toolchain build key starts a new cohort). The ledger is carried between runs.
  - Overflow, target, and infrastructure failures are reported apart from divergence and never count as a pass.
  - A divergence retains both traces, both evidence records, and the differ output.
  - The bound counts clean-batch repetitions only.
  - Rounds: always at least 2, at most 16, gated by the measured round cost against a 300m budget.
  - An A,A then B,B test reports a divergence.
- tools/gomad3integration/qualification/soak.json: the 4 smoke suites plus the guarded frontend-system-info, seeds 11 and 17, with the sizing reasoning recorded. linux/amd64 is informational under fn-105 D12.
- Root `make gomad3-soak`. gomad3.yml gains `determinism-soak-darwin` (strict) and `determinism-soak-linux` (informational): schedule/dispatch only, a workload x seed matrix, the ledger restored via `gh run download`, and uploads that use the existing pinned SHAs.
- Docs updated: README, SPEC (new RUNTIME.CHOICES.DIAGNOSTICS, RUNTIME.STREAMS, RUNTIME.TRUST.EXCLUSIONS, QUALIFICATION.SOAK, and gomadtool DIAGNOSTIC.DIFF/SOAK rows), CLI, ARCHITECTURE, TUTORIAL, AGENTS.md, MILESTONES, and the integration README.

Local evidence (linux/arm64, developmental only):
- `go test -tags test_dep ./qualification/soak`: 16 pass. Mutation check: removing the cross-batch divergence assignment fails 3 tests.
- With stock Go 1.27.1, `. ./cmd/gomadtool ./qualification/soak ./qualification/set`: pass, including TestPackageArchitecture and the vocabulary test.
- `make -C tools/gomad3 validate`: exit 0 (35s). actionlint: clean. go vet and gofmt: clean.
- fn-111 guide audit: no new errors relative to the pre-edit baseline.
- The soak was exercised end to end against a stand-in `gomad qualify`, because the patched toolchain refuses linux/arm64:
  - 2 clean runs, exit 0, ledger accumulating to 24 repetitions per cohort.
  - 1 injected divergence, exit 1, with traces retained and the differ reporting runtime_cheap_rand_draws.
  - No Gomad bound was measured.
- Evidence: .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/handover.md.

Remaining native gates:
1. Dispatch or schedule gomad3.yml and retain one completed run of the determinism-soak-darwin and determinism-soak-linux jobs, with their reports and ledgers. Quote their per-cohort counts as the first measured bound (diagnostics on), and re-check the round sizing from execution_wall_nanos.
2. Run `make -C tools/gomad3 test-host` on darwin/arm64 and linux/amd64 covering the new packages.

Blocked:
Blocked: the soak gate and contract docs are delivered and reviewed (SHIP, claude-fable-5-1, round 2). Two items remain, and both need native CI: "one completed scheduled or dispatched run per platform is retained with its report", and the native measured bound that run supplies.

Done (commits bae373d14, 204732c33, 8ecc7e584, 4efa98213, 9bc745d1e on gomad-fn112; base 331b75bb6):
- `gomadtool soak` (tools/gomad3/qualification/soak) runs `qualify --diagnostics` batches of 32 under 2 busy host threads.
  - Per-cohort comparison: workload, seed, platform, and execution identity (a new toolchain build key starts a new cohort). The ledger is carried between runs.
  - Overflow, target, and infrastructure failures are reported apart from divergence and never count as a pass.
  - A divergence retains both traces, both evidence records, and the differ output.
  - The bound counts clean-batch repetitions only.
  - Rounds: always at least 2, at most 4 (N 64 to 128 per seed), gated by the measured round cost against a 120m per-job budget (cost cap, commit 9bc745d1e; delta review SHIP).
  - An A,A then B,B test reports a divergence.
- tools/gomad3integration/qualification/soak.json: the 4 smoke suites plus the guarded frontend-system-info, seeds 11 and 17, with the sizing reasoning recorded. linux/amd64 is informational under fn-105 D12.
- Root `make gomad3-soak`. gomad3.yml gains `determinism-soak-darwin` (strict) and `determinism-soak-linux` (informational): schedule/dispatch only, a workload x seed matrix, the ledger restored via `gh run download`, and uploads that use the existing pinned SHAs.
- Docs updated: README, SPEC (new RUNTIME.CHOICES.DIAGNOSTICS, RUNTIME.STREAMS, RUNTIME.TRUST.EXCLUSIONS, QUALIFICATION.SOAK, and gomadtool DIAGNOSTIC.DIFF/SOAK rows), CLI, ARCHITECTURE, TUTORIAL, AGENTS.md, MILESTONES, and the integration README.

Local evidence (linux/arm64, developmental only):
- `go test -tags test_dep ./qualification/soak`: 16 pass. Mutation check: removing the cross-batch divergence assignment fails 3 tests.
- With stock Go 1.27.1, `. ./cmd/gomadtool ./qualification/soak ./qualification/set`: pass, including TestPackageArchitecture and the vocabulary test.
- `make -C tools/gomad3 validate`: exit 0 (35s). actionlint: clean. go vet and gofmt: clean.
- fn-111 guide audit: no new errors relative to the pre-edit baseline.
- The soak was exercised end to end against a stand-in `gomad qualify`, because the patched toolchain refuses linux/arm64:
  - 2 clean runs, exit 0, ledger accumulating to 24 repetitions per cohort.
  - 1 injected divergence, exit 1, with traces retained and the differ reporting runtime_cheap_rand_draws.
  - No Gomad bound was measured.
- Evidence: .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/handover.md.

Remaining native gates:
1. Dispatch or schedule gomad3.yml and retain one completed run of the determinism-soak-darwin and determinism-soak-linux jobs, with their reports and ledgers. Quote their per-cohort counts as the first measured bound (diagnostics on), and re-check the round sizing from execution_wall_nanos.
2. Run `make -C tools/gomad3 test-host` on darwin/arm64 and linux/amd64 covering the new packages.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Actual Darwin soak report/ledger/measured bound, cohort/diagnostic controls and shared documentation. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
