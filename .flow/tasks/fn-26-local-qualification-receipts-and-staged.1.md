---
satisfies: [R1, R3, R7]
---

# fn-26-local-qualification-receipts-and-staged.1 Define reusable Evaluation Profiles and the local policy

## Description
Add `Umpire.Evaluation`, Temporal-free: a checked `EvaluationProfile` (name, claim text, trust basis, the Known Gap kinds that block acceptance, and an ordered reason table: each reason's name, one status-specific condition from the closed set `verdict-violated`, `verdict-inconclusive`, `disposition-stopped`, `disposition-incomplete`, `cleanup-unclosed`, `known-gap-blocking`, `unsupported-rule`, and the decision it forces, `rejected` or `incomplete`, in precedence order), checked when declared: an empty table, an empty or duplicate reason name and a condition named twice reject by name. A subject for which no reason holds is accepted. It carries no catalog, endpoint, credential, path or execution authority. Add `Temporal.Evaluation.Local` declaring `local-ephemeral`, trust `local-ephemeral-cluster`, `capability` and `interpretation` gaps blocking, and the table, in precedence order: `verdict-violated` rejected, `disposition-stopped` rejected, `verdict-inconclusive` incomplete, `disposition-incomplete` incomplete, `cleanup-unclosed` incomplete, `known-gap-blocking` incomplete, `unsupported-rule` incomplete. Add the non-default `umpire-evaluation-profiles` executable rendering every declared Profile to canonical JSON under `tools/umpire/evaluation/testdata/profiles/<name>.json`, and `make umpire-gen-evaluation-profiles` / `umpire-check-evaluation-profiles`, the check added to `umpire-check-regression`'s prerequisites; the Profile identity is the SHA-256 of the rendered bytes, and the same Profile renders the same bytes.

### Quick commands
`cd model && lake build Umpire.Evaluation.Tests umpire-evaluation-profiles && cd .. && make umpire-check-evaluation-profiles`

**Size:** M
**Files:** `model/Umpire/Evaluation.lean`, `model/Umpire/Evaluation/Tests.lean`, `model/Temporal/Evaluation/Local.lean`, `model/Temporal/Tool/EvaluationProfiles.lean`, `model/lakefile.lean`, `Makefile`, `tools/umpire/evaluation/testdata/profiles/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] An empty table, an empty or duplicate reason name, and a condition named twice fail to declare, each by name.
- [ ] No endpoint, credential, path, catalog, Limit, Driver, execution authority, or Temporal value enters reusable Umpire.
- [ ] Same Profile bytes yield the same identity; a different Profile remains an independent assessment; the Profile's rendered bytes and identity agree between Lean and Go.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
