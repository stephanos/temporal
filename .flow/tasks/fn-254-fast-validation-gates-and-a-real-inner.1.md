---
satisfies: [R1]
---
# fn-254-fast-validation-gates-and-a-real-inner.1 Consolidate historical validation timings and unknowns


## Description
Consolidate existing evidence for R1. The owner explicitly waived fresh baseline measurements on 2026-10-25: "can we ignore baseline measurement? we already have historic data". Proceed to optimization after this lightweight report; do not wait for a quiet-host or build-cache lease.

**Size:** S
**Files:** `model/design/gate-speed.md` (new); retained optional probe scripts and historical source copies under `.flow/tmp/fn254-baseline/`.
**Touches:** [model/design/gate-speed.md, .flow/tmp/fn254-baseline/**]

### Approach
- Read fn-247's committed validation record for its 2,093s outer required/Apalache-opt-in run and stage breakdown; retain its exact source/mode rather than treating it as routine default cost.
- Read fn-237's retained default/required runs and existing per-binary/UI/build logs, including fn2511's annotate UI evidence. State workload/profile/host differences and any absent attribution.
- Report unknown per-test/linker/TLC counts explicitly. Do not turn a code-derived launch prediction into a measurement or infer per-binary compile time from an aggregate build.
- Name the five largest supported costs and rank expected optimization savings as estimates. Preserve source receipts/logs needed for review, without new production builds/tests/timing runs.
- Keep the prepared synthetic-tested observer/probe optional for R6, not a completion gate here. Correctness inventories remain owned by .3/.4/.5/.7 before their migrations.

### Investigation targets
**Required:**
- `model/scripts/time-gate.sh` — existing stage timer
- `Makefile:1270-1315` — `model-check`, `model-time`, `model-validate`
- `model/umpire/backends/tlc/src/runner.rs:129-250` — TLC launch path
- `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json` — previous stage timings

## Acceptance
- [ ] Historical timings and available counts are cited with their actual source/mode/coverage; unavailable per-test, compile/link or launch data is unknown with a reason.
- [ ] Supported largest costs and estimated optimization priorities are clear; incomparable runs and failed/partial gates are labelled honestly.
- [ ] No fresh baseline production measurement is required or claimed. Optional prepared probes and their synthetic evidence are retained separately for possible final measurements.

## Done summary
Consolidated historical validation costs and optimization priorities in `model/design/gate-speed.md`, with exact source/backend distinctions, failed/pending gate disclosures and explicit unknowns. Fresh baseline production measurements were waived by the owner and official amendment ded1ffe904; R6 final measurements and R4/R5 migration inventories remain required.

Tier: AGENTS implementer gpt-6.1-sol/medium; judge unavailable(no_key), explicit project routing retained.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

baseline: not run (owner explicitly waived fresh baseline production measurements; no new baseline pass claimed). Task .1 has no remaining production Quick gate after amendment.

Verification: historical extraction; report stage-value/link/hash checks; scratch Python parsing; Bash syntax; synthetic partial-byte/status/Java-observer controls all exit 0. Initial synthetic controls and the report transcription check failed, were corrected before handover, and remain recorded under `.flow/tmp/fn254-baseline/`; neither executed project code. Optional real-tool integration remains unverified. No attributable command remains running.

Original-range `gate classify` reported `FULL: unmatched: MILESTONES.md` for the inherited conductor amendment. Task-only range ded1ffe904..ef14f4e654 also returned `FULL: unmatched: model/design/gate-speed.md` (exit 1): the classifier does not admit this design path as tier-B. Neither observation is a classifier pass. No production suite was run, as explicitly directed by the owner waiver; the implementation commit is Markdown-only by git diff, independent of the conservative classifier. There are no invented GATE_SKIPPED lines for absent Quick gates.

Original persisted worker base is 72272d6f5e06258ee5d31179231c7b3a00fd3545. Conductor-owned ded1ffe904 was fast-forwarded as instructed; this worker's sole implementation commit is ef14f4e6549d131d7739d7126c18c57fe1996f7d. Integrate/review that report commit, not the inherited waiver as worker-authored work. The isolated branch is clean; keep its worktree for conductor review fixes.

Retained proof: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-baseline/README.md`, `historical/summary.json`, preserved historical logs and hashes, `report-verification-attempt{1,2}.log`, `synthetic-attempts.json`, `synthetic-verification.log`, `task-only-classification.log`. No .flow feature route changed.

Conductor verification: integrated report ef14f4e654 via f90bb325e5, rechecked transcription/link/source hashes and ran affected retired-name hygiene with the existing native helper (exit0). Review deterministically triaged SHIP as docs-only against ded1ffe904. R1 uses historical evidence under the explicit owner waiver; no current-source production pass is claimed by this report.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ef14f4e6549d131d7739d7126c18c57fe1996f7d, f90bb325e5
- Tests: bash .flow/tmp/fn254-baseline/run.sh historical (exit 0; existing evidence only), python3 .flow/tmp/fn254-baseline/probe/verify_report.py model/design/gate-speed.md (exit 0; timing/reference/hash verification), python3 .flow/tmp/fn254-baseline/probe/controls.py (exit 0; synthetic executables only), bash -n .flow/tmp/fn254-baseline/run.sh (exit 0), git diff --check (exit 0), flowctl gate classify --base 72272d6f5e06258ee5d31179231c7b3a00fd3545 (FULL: inherited conductor MILESTONES.md), flowctl gate classify --base ded1ffe904d7df20d2dc281daf70778ed929c39d (exit 1; FULL: unmatched: model/design/gate-speed.md; no classifier pass claimed), Production gates not run: owner waiver committed ded1ffe904d7df20d2dc281daf70778ed929c39d; no fresh baseline measurement or pass claimed, UMPIRE_XTASK=<root-target>/xtask-bootstrap/debug/xtask bash model/scripts/retired-names.sh (exit0; no compilation)
- PRs: