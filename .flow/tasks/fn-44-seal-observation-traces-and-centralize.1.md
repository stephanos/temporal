---
satisfies: [R1, R6]
---
# fn-44-seal-observation-traces-and-centralize.1 Centralize Model Trace coordinate semantics

## Description
Add the Core-owned coordinate vocabulary and semantic operations required by R1, with exact executable regressions. This is the early proof point and stays independent of Observation admission changes.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/CoreTests/Trace.lean`, `model/Umpire/CoreTests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/CoreTests/Trace.lean, model/Umpire/CoreTests.lean]

### Approach
- Move `ModelCoordinate` to the lowest shared semantic owner without changing its constructors, ordering, or derived instances.
- Add documented direct operations for canonical coordinate enumeration, strict one-based `ModelValue` lookup, and coordinate Definition kind.
- Encode zero/out-of-range rejection before subtraction and preserve source-order enumeration.
- Add equational and executable checks for empty, one-step, repeated-value, and multi-observation traces.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:102-124` — current Model Value/Trace ownership seam.
- `model/Umpire/Observation/Evaluation.lean:160-167` — current coordinate type to relocate.
- `model/Umpire/Observation/Evaluation.lean:1100-1107` — canonical enumeration baseline.
- `model/Umpire/ImplementationLink/Application.lean:275-309` — strict lookup/kind implementation baseline.
- `model/Umpire/CoreTests/Trace.lean:1-34` — focused Core trace-check style.
- `model/Umpire/Observation/Tests/EvidenceLink.lean:161-177` — repeated-value coordinate oracle.

### Key context
Keep the API direct on the existing semantic types; do not add a cached view, a second trace representation, or Property/Observation imports to Core. Preserve current comments when ownership moves.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.CoreTests
```

## Acceptance
- [ ] `ModelCoordinate` and its docstring live at the Core semantic boundary with unchanged constructors, order, and derived behavior.
- [ ] One Core API enumerates coordinates, performs strict one-based lookup, and classifies Definition kind.
- [ ] Every zero and out-of-range coordinate returns no value; empty and repeated-value traces retain exact canonical positions.
- [ ] Focused tests cover enumeration/lookup bijection, ordering, kinds, and boundary cases without new axiom or trust dependencies.
- [ ] Existing comments remain present and accurate, and the focused Core target builds without warnings.

## Done summary
Moved `ModelCoordinate` and its preserved documentation to Core, added canonical source-order enumeration, strict one-based `ModelValue` lookup, and Definition-kind classification, and covered empty, one-step, multi-observation, repeated-value, zero, and out-of-range behavior. `Umpire.CoreTests`, downstream Observation/Implementation Link builds, and `make lint-model` pass; `make lint-code` remains red on 1,373 inherited Go findings and applied autofixes were exactly undone because this task changes only Lean.

stage: impl-review - ran [2026-09-01T01:21:08Z..2026-09-01T01:24:35Z] (Codex SHIP; receipt /tmp/impl-review-receipt-fn-44-seal-observation-traces-and-centralize.1.json)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 824ddf38ad4f8d064f47f3e599efdd66032ada56
- Tests: baseline: green - cd model && mise exec -- lake build Umpire.CoreTests, RED_EXPECTED: cd model && mise exec -- lake build Umpire.CoreTests (exit 1: missing Umpire.ModelTrace.coordinates, Umpire.ModelTrace.valueAt?, and Umpire.ModelCoordinate.definitionKind), cd model && mise exec -- lake build Umpire.CoreTests, cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.ImplementationLink.Tests, make lint-model, INHERITED_RED: make lint-code (exit 2: 1373 pre-existing Go lint findings; task diff contains only model/Umpire/*.lean; lint autofixes exactly undone), git diff --check, impl-review Codex SHIP session 01a05a8e-4b55-7d11-bfd3-6083358faef2 receipt /tmp/impl-review-receipt-fn-44-seal-observation-traces-and-centralize.1.json
- PRs:
