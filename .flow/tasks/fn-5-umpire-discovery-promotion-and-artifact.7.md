---
satisfies: [R1, R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.7 Wire focused Nexus discovery and promotion checks

## Description
Integrate the two retained fn-5 capabilities through root commands, aggregate tests, and concise
documentation without expanding either surface.

**Size:** S
**Files:** `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `model/TemporalExperimentalTests.lean`
**Touches:** [Makefile, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/TemporalExperimentalTests.lean]

### Approach

- Extend the existing root inspector target to document and exercise `list` and exact `explain` while
  preserving positional scenario inspection.
- Add one fixed review-only promotion check target that invokes the closed candidate identity and
  exposes no source or destination override; its check renders the inert proposal in memory, verifies
  both base and fault-bearing lineages, and clean-elaborates the embedded source.
- Keep both focused suites in the existing experimental aggregate and `umpire-check-regression` path.
- Document the four discovery entries, exact command examples, expected-count-one proposal meaning,
  and the fn22-to-fn5 caller gate without claiming runtime reproduction or installation.

### Non-goals

- No generic semantic graph, generated glossary, machine index, broad stable regression set, general artifact evolution, CI workflow, or model-local Makefile.

### Investigation targets

**Required:**
- `Makefile` — current `umpire-inspect` and `umpire-check-regression` ownership.
- `model/README.md` — current Nexus learning path and inspector instructions.
- `model/ARCHITECTURE.md` — current tool/import boundaries.
- `model/Umpire/ARCHITECTURE.md` — reusable package and generated-view boundaries.
- `model/TemporalExperimentalTests.lean` — current experimental aggregate.
- `.plans/UMPIRE4_SPEC.md` — CLI-03 and EXP-05 wording.

### Quick commands

`cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromoteTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote`

`make umpire-check-regression`

## Acceptance
- [ ] Root commands expose deterministic Nexus list/explain and one non-mutating fixed-candidate promotion check; documentation says the output is inert and fn-22 alone establishes runtime eligibility.
- [ ] Focused discovery/promotion tests and the existing experimental aggregate are wired into the root non-mutating regression check.
- [ ] Existing positional inspector behavior and checked-in regression outputs remain unchanged.
- [ ] Documentation names the exact four entries, the expected-count-one proposal, and fn-22 eligibility gate without claiming replay, minimization, automatic installation, or broader coverage.
- [ ] No CI workflow or model-local Makefile is added.
- [ ] Focused tests, aggregate tests, root regression checks, and `git diff --check` pass with comments preserved.

## Done summary
Added root Nexus list and exact-explain commands, a fixed non-mutating promotion check, deterministic regression wiring, and the missing experimental promotion-suite imports. Documented the exact four-entry discovery surface, expected-count-one inert proposal, and fn-22-only runtime eligibility boundary without changing positional inspection or checked-in outputs.

The canonical focused Quick builds, both root discovery commands, fixed promotion check, positional fixture comparisons, `make lint-model`, and `git diff --check` pass. `make umpire-check-regression` remains red only at the verified pre-edit `model/Umpire/SemanticInventory/KnownGaps.lean:296` Temporal-vocabulary finding; before reaching it, generated-view, portable-evaluation, hermetic Nexus, and the newly wired fixed-promotion checks pass.

stage: impl-review - ran (Codex SHIP; zero findings and no unaddressed requirements)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker inactive)
## Evidence
- Commits: 46edf0e2d943b47ccb9b310140597472cf8dd94c
- Tests: baseline: green (cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromotionBindingTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote; 191 jobs), baseline: red (make umpire-check-regression failed pre-edit only at model/Umpire/SemanticInventory/KnownGaps.lean:296 Temporal-owned vocabulary), TDD_RED: make umpire-list-nexus; make umpire-explain-nexus QUERY=workflow-nexus.query.exact-action-caller-closure; make umpire-check-promotion (targets absent), cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromoteTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote (192 jobs), cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromotionBindingTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote (192 jobs), make umpire-list-nexus (status 0; exact four entries; deterministic repeated bytes), make umpire-explain-nexus QUERY=workflow-nexus.query.exact-action-caller-closure (status 0; exact selector; deterministic repeated bytes), make umpire-check-promotion, make umpire-inspect SCENARIO=workflow-nexus.query.exact-action-caller-closure and switch.query.exact-action (checked-in fixtures byte-identical), make lint-model (216 jobs; status 0), make umpire-check-regression (post-change: generated views, portable evaluation, fixed promotion, and hermetic Nexus pass; inherited KnownGaps.lean:296 vocabulary failure remains), git diff --check
- PRs: