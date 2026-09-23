---
satisfies: [R1, R3, R7]
---

# fn-26-local-qualification-receipts-and-staged.1 Define reusable Evaluation Profiles and the local policy

## Description
Add `Umpire.Evaluation`, Temporal-free: a checked `EvaluationProfile` (name, claim text, the Run dispositions, Verdict statuses and cleanup outcomes it accepts, the Known Gap kinds that block acceptance, whether every rule the Verdict names at a terminal state must carry a supporting sequence, whether the recorded Profile name must equal a named value, the trust basis, and an ordered reason table: each reason's name, the condition it names from a closed set -- `disposition`, `verdict`, `cleanup`, `known-gap`, `unsupported-rule`, `profile-name` -- the decision it forces, `rejected` or `incomplete`, and its precedence), checked when declared: an empty or duplicate reason, a reason naming an unknown condition, an accepted set that is empty or contradicts itself, and a condition the Profile requires but the table never names reject by name. It carries no catalog, endpoint, credential, path or execution authority. Add `Temporal.Evaluation.Local` declaring `local-ephemeral`: a `COMPLETED` Run whose cleanup `SUCCEEDED` with a `SATISFIED` Verdict, every rule supported, every `capability` and `interpretation` Known Gap blocking, trust `local-ephemeral-cluster`. Add the non-default `umpire-evaluation-profiles` executable rendering every declared Profile to canonical JSON under `tools/umpire/evaluation/testdata/profiles/<name>.json`, and `make umpire-gen-evaluation-profiles` / `umpire-check-evaluation-profiles`; the Profile identity is the SHA-256 of the rendered bytes, and the same Profile renders the same bytes.

### Quick commands
`cd model && lake build Umpire.Evaluation.Tests umpire-evaluation-profiles && cd .. && make umpire-check-evaluation-profiles`

**Size:** M
**Files:** `model/Umpire/Evaluation.lean`, `model/Umpire/Evaluation/Tests.lean`, `model/Temporal/Evaluation/Local.lean`, `model/Temporal/Tool/EvaluationProfiles.lean`, `model/lakefile.lean`, `Makefile`, `tools/umpire/evaluation/testdata/profiles/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Empty, duplicate, contradictory, unknown, stale, and N+1 inputs fail deterministically.
- [ ] No endpoint, credential, path, Driver, execution authority, or Temporal value enters reusable Umpire.
- [ ] Same Profile bytes yield the same identity; a different Profile remains an independent assessment.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
