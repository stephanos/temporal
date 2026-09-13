---
satisfies: [R12, R7]
---
# fn-85-model-side-effects-as-typed-actions-and.12 Canary and exploratory sets admitted with coverage targets

## Description
Admit the two remaining set purposes over the Nexus Model (R12, the rest of R7): a canary set over Queries 1 and 2 with `handler: observed` admits, a canary set naming a Query whose Case carries a white-box Known Gap rejects, and an exploratory set over the protocol machine enumerates its coverage targets (rows, result values, members of claimed classes) deterministically under its budget, pinned by a golden. Running them stays with fn-70, fn-29 and fn-33.

**Size:** S
**Files:** `model/Umpire/Command/Authoring.lean` (canary admission: white-box gap check; exploratory: coverage-target enumeration), `model/Umpire/Command/Records.lean` (`CoverageTarget`), `model/Temporal/Feature/Nexus/Caller/Model.lean` (the two sets), `model/Temporal/Feature/Nexus/Caller/Tests.lean` (`#guard_msgs` for the rejections), `model/Temporal/Feature/Nexus/Caller/Fixtures/CallerExploratoryCoverage.json` (new golden, written by `Temporal.Tool.Goldens`), `model/Temporal/Tool/Goldens.lean`, `Makefile` (`UMPIRE_GOLDEN_DIRECTORIES` if a new directory)
**Touches:** [model/Umpire/Command/**, model/Temporal/Feature/Nexus/Caller/**, model/Temporal/Tool/Goldens.lean, Makefile]

### Approach
- White-box Known Gaps are the `capability` or `interpretation` kinds a Case records from an `unobservable` row or an unbindable setup parameter; the canary admission walks the would-be Case's gaps and rejects naming the Query and the gap.
- Coverage targets: enumerate rows by position, result values by enum, and class members by the product of constructor fields; order is the declaration order (AUT-09) so the golden is stable; the budget is a `limits` name and bounds the enumeration.
- Goldens change only through `make umpire-gen-goldens`; add the new file to the checked directories.

### Investigation targets
**Required:**
- `model/Umpire/KnownGap.lean:8-68` — kinds and canonical set
- `model/Temporal/Tool/Goldens.lean:41-74` — the golden writer
- `Makefile:85-89,577-595` — golden directories and gates

**Optional:**
- `.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md:28-40` — what fn-33 will read from the targets

### Key context
- Single-value classes carry no claim and produce no class-member target.

## Acceptance
- [ ] the canary set over Queries 1 and 2 with `handler: observed` admits; a canary set naming a Query with a white-box Known Gap rejects in place naming both, pinned by `#guard_msgs`
- [ ] the exploratory set enumerates its coverage targets deterministically; the golden is checked by `make umpire-check-goldens` and rendering twice is byte-identical
- [ ] `lake build TemporalModelTests` green; `make umpire-check-goldens` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
