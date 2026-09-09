---
satisfies: [R8]
---
# fn-80-close-the-model-to-case-seam-and-harden.6 Add the query all verify form

## Description
Implements R8 (spec §R2 and R8). Replaces the always-erroring `query ... all ...` macro with elaboration through `QueryForm.verify`, and makes the Producer reject a verify Query as witness-absent.

**Size:** S
**Files:** `model/Temporal/Feature/Nexus3/Syntax.lean`, `model/Temporal/Feature/Nexus3/Authoring.lean`, `model/Temporal/Feature/Nexus3/Tests.lean`, `model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean`, `model/Temporal/Feature/Nexus3/Testpilot.lean` (witness-absent path)
**Touches:** [model/Temporal/Feature/Nexus3/**]

### Approach
- The stub at `Syntax.lean:127-128` becomes a real macro expanding to `Authoring.check` with `QueryForm.verify` (`Umpire/Query/Language.lean:119-126`: `.verify` yields `.universal` / `.verifiedWithinLimits`).
- `Authoring.CheckedModel` must carry an optional witness so `produce` can reject a verify Query with `witness.absent` (the rejection retained from task .4).
- Add a verify Query over the R2 race lifecycle and one over an unsatisfiable Behavior asserting `unsatisfiable`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:100-140` — `QueryForm` and quantifiers
- `model/Temporal/Feature/Nexus3/Authoring.lean:411-459` — `CheckedModel` and `check`

**Optional** (reference as needed):
- `model/Umpire/Query/Authoring.lean:243-300` — `query%` elaborator

## Acceptance
- [ ] `query <name> on <model> all <property> in <behavior> limits <limits>` elaborates and checks on the race lifecycle
- [ ] A verify Query over an unsatisfiable Behavior reports `unsatisfiable` (PLN-05), asserted with `#guard`
- [ ] `produce` rejects a verify-form `CheckedModel` with a `witness.absent` `LoweringError`, asserted with `#guard`
- [ ] `lake build Temporal TemporalModelTests` and `make lint-model` pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
