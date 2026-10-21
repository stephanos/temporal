---
satisfies: [R5]
---
# fn-217-annotation-catalog-over-the-temporal.5 DESIGN.md: annotation catalog write-up

## Description
`model/DESIGN.md` section "Annotation catalog": the layering and admission rules, the four exhaustiveness levels and the `api.binpb` bump workflow, the borrowed vocabularies (protovalidate, JSON Schema, AIP) and why no engine is borrowed, the first `gaps.txt` and what it found, and which tags proved system-free (candidates to move into umpire). After .4. **Touches:** `model/DESIGN.md`, `model/README.md`, MILESTONES (root agent).

## Acceptance
- [ ] TBD

## Done summary
Documented annotation catalog layering, admission, the four exhaustiveness levels, descriptor linking and API updates, borrowed vocabularies, actual probe findings and system-free tag ownership. README links authors to the existing syntax guide; the complete mini activity slice write-up moved there with two cross-reference repairs, keeping DESIGN at 1,000 lines and README at 566.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-21T00:30:43Z..2026-10-21T00:32:18Z] - SHIP

Formal review used codex:gpt-6.1-sol:high, explicitly overriding the configured codex:gpt-5.6-sol:xhigh environment backend. All three lenses returned SHIP with zero findings. All three sidecars were read before empty-plan finalization at frozen source daa8228314732975f2afbfcff7308a4f8d3be6d0. Receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.5.json. Fanout RID: 31ead0763b8745fe88b9671291505c0b. Writer and CLI reviewers are from the same GPT/Codex family; no cross-family independence claim. Host metadata did not expose the executed writer model. No CLI implementation bridge or worker subagent ran, and no NEEDS_WORK transition occurred.

Baseline: green. The pre-edit mandated model gate exited 0 on original handle 86373; `.flow/tmp/fn2175-baseline-model.log` and `fn2175-baseline.md` retain evidence. An earlier wrong-directory nested-shell invocation exited 2 before executing the gate; explicit cwd fixed the invocation. The parent spec defines no Quick commands.

Post-edit `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check` exited 0 on original handle 79524, covering 280 Rust cases in 94 result groups and both Go hosts (1.062s generic, 46.403s Temporal). Full log: `.flow/tmp/fn2175-final-model.log`. The gate ran on the exact document bytes subsequently committed and reviewed; review made no changes. `git diff --check`, document/anchor/link-target checks and exact comparison of the relocated slice (apart from the two repaired cross-references) passed. Reviewers independently checked the documented source contracts, golden counts and local links; they did not rerun runtime suites.

Flow classified FULL because externally edited MILESTONES.md is unmatched. A green receipt was refused with `NO_RECEIPT: worktree dirty outside the ignore set (MILESTONES.md) - receipt not warrantable`; terminal gate output supplies the evidence. No receipt bypass or GATE_SKIPPED claim. Live slice/gap tests were not rerun for this documentation-only task. Documentation describes their existing entry points and verified predecessor evidence without claiming new live execution or Temporal TLC agreement.

Implementation changes are confined to model/DESIGN.md and model/README.md. The 62 Confirmed/52 expected rejections/10 intentional acceptances, 73 uncovered entries, three unresolved seams and two corrected Start policy expectations were checked against source and retained reports. The accepted-gap registry is empty and Visibility remains a Temporal-side test control. No planned derive rewrite or singular directory rename is presented as implemented.

Five externally owned planning paths remain untouched and excluded from both commits: MILESTONES.md; .flow/tasks/fn-217-annotation-catalog-over-the-temporal.6.md and .6.json; .flow/tasks/fn-217-annotation-catalog-over-the-temporal.7.md and .7.json. No clean-tree claim, external write, dependency, source/Model/server/golden/threshold change, feature-map edit, tracker work or plan-sync occurred. The conductor owns the deferred MILESTONES update.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: daa8228314732975f2afbfcff7308a4f8d3be6d0
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check, git diff --check, Documentation structure, link-target and relocated-slice preservation checks
- PRs: