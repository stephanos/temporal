---
satisfies: [R1, R3, R7]
---

# fn-26-local-qualification-receipts-and-staged.1 Define reusable Evaluation Profiles and the local policy

## Description
Add `Umpire.Evaluation`, Temporal-free: a checked `EvaluationProfile` (name, claim text, trust basis, the Known Gap kinds that block acceptance (as `Umpire.KnownGap.KnownGapKind`, not declared again), and an ordered reason table: each reason's name, one status-specific condition from the closed set `verdict-violated`, `verdict-inconclusive`, `disposition-stopped`, `disposition-incomplete`, `cleanup-unclosed`, `known-gap-blocking`, `unsupported-rule`, and the decision it forces, `rejected` or `incomplete`, in precedence order), checked when declared: an empty table, an empty or duplicate reason name, a condition named twice, `known-gap-blocking` named with no blocking kind, and blocking kinds with no `known-gap-blocking` reason reject by name. A subject for which no reason holds is accepted. It carries no catalog, endpoint, credential, path or execution authority. Add `Temporal.Evaluation.Local` declaring `local-ephemeral`, trust `local-ephemeral-cluster`, `capability` and `interpretation` gaps blocking, and the table, in precedence order: `verdict-violated` rejected, `disposition-stopped` rejected, `verdict-inconclusive` incomplete, `disposition-incomplete` incomplete, `cleanup-unclosed` incomplete, `known-gap-blocking` incomplete, `unsupported-rule` incomplete. Add the non-default `umpire-evaluation-profiles` executable rendering every declared Profile to canonical JSON under `tools/umpire/evaluation/profiles/<name>.json`, and `make umpire-gen-evaluation-profiles` / `umpire-check-evaluation-profiles`, the check added to `umpire-check-regression`'s prerequisites; the Profile identity is the SHA-256 of the rendered bytes, written `sha256:<hex>`, pinned in `Temporal/Evaluation/LocalTests.lean`; the same Profile renders the same bytes. The new modules are imported by their aggregators (`Umpire.lean`, `UmpireTests.lean`, `Temporal.lean`, `TemporalModelTests.lean`), so `make umpire-check-model-module-index` finds no uncovered source.

### Quick commands
`cd model && lake build Umpire.Evaluation.Tests umpire-evaluation-profiles && cd .. && make umpire-check-evaluation-profiles`

**Size:** M
**Files:** `model/Umpire/Evaluation.lean`, `model/Umpire/Evaluation/Tests.lean`, `model/Temporal/Evaluation/Local.lean`, `model/Temporal/Evaluation/LocalTests.lean`, `model/Temporal/Tool/EvaluationProfiles.lean`, `model/Umpire.lean`, `model/UmpireTests.lean`, `model/Temporal.lean`, `model/TemporalModelTests.lean`, `model/lakefile.lean`, `Makefile`, `tools/umpire/evaluation/profiles/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] An empty table, an empty or duplicate reason name, a condition named twice, and a contradictory Known Gap policy fail to declare, each by name.
- [x] No endpoint, credential, path, catalog, Limit, Driver, execution authority, or Temporal value enters reusable Umpire.
- [x] Same Profile bytes yield the same identity, pinned in Lean; a different Profile remains an independent assessment.
- [x] Every new module is imported by its aggregator and the module-index check passes.

## Done summary
`Umpire.Evaluation` declares a checked, Temporal-free Evaluation Profile (name, claim, trust basis, blocking Known Gap kinds as `KnownGapKind`, an ordered reason table over seven status-specific conditions forcing `rejected` or `incomplete`); `Profile.declare` rejects an invalid name, an empty claim, trust or table, an empty or duplicate reason, a repeated condition, a duplicate blocking kind and either half of a contradictory Known Gap policy, each by name. `Temporal.Evaluation.Local` declares `local-ephemeral` with the plan's table; `umpire-evaluation-profiles` renders every declared Profile to compact canonical JSON under `tools/umpire/evaluation/profiles/`, and `umpire-check-evaluation-profiles` (in `umpire-check-regression`) diffs a fresh render. The identity `sha256:2803afa2…174c` is pinned in `Temporal/Evaluation/LocalTests.lean`. Implementation review: SHIP in one round, its two P3 notes applied as polish.
## Evidence
- Commits: 9d012aff02f21f9bedb5680e07b3456d900071de, 4052acf58138672eef097019fe90d34af9792701
- Tests: cd model && lake build, make umpire-check-evaluation-profiles, make umpire-check-model-module-index umpire-check-inventory, make umpire-check-retired-vocabulary, LEAN_NUM_THREADS=1 make lint-model (baseline: 163 + 1 errors, none in the new modules)
- PRs: