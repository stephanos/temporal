---
satisfies: [R4, R7, R8]
---
# fn-16-authored-variation-spaces-and.6 Publish Space facades and authoring documentation

## Description
Expose the approved reusable package, document the small authored-space workflow and downstream contracts, and reconcile component status for R4/R7/R8. Actual fn-5 catalog consumption remains fn-5 implementation work.

**Size:** M
**Files:** `model/Umpire/Space.lean`, `model/Umpire.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_DSL.md`
**Touches:** [model/Umpire/Space.lean, model/Umpire.lean, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_DSL.md]

### Approach
- Publish a narrow facade over Language, Intent, Metadata, and Compiler modules and preserve vertical package ownership.
- Add a concise copyable authored-to-checked-to-batch walkthrough using the Temporal example, emphasizing pure properties, requested attempts, and target-owned outcomes.
- Document `CheckedSpaceMetadata` as fn-5 input and `lowerSpacePoint`/coverage goals as later C8 inputs, without claiming those consumers are implemented.
- Update C3/C4 and Milestone A implementation status only after all focused and aggregate tests pass; retain decoder/runtime/exploration gaps.
- Keep existing root Make commands; add no model-local Makefile or CI workflow.

### Investigation targets
**Required** (read before coding):
- `model/Umpire.lean:1-8` — reusable facade convention
- `model/Umpire/ARCHITECTURE.md:31-79` — package lifecycle and dependency diagram
- `model/README.md:3-66` — current author workflow
- `.plans/UMPIRE4_COMPONENTS.md:203-262` — C3/C4 responsibilities/status
- `.plans/UMPIRE4_COMPONENTS.md:568-611` — Milestone A exit evidence
- `.flow/specs/fn-5-umpire-discovery-promotion-and-artifact.md` — downstream catalog boundary

### Acceptance
- [ ] Public imports expose one reusable Space lifecycle without Temporal vocabulary or a second semantic API.
- [ ] Documentation gives a concise real example and distinguishes batch compilation from exploration/runtime/conformance.
- [ ] Roadmap states only tested implementation and preserves all remaining C8/decoder/runtime gaps.
- [ ] Fn-5's recorded dependency and task contract consume checked metadata later; fn-16 does not implement catalog UI or persistence.
- [ ] `UmpireTests`, `TemporalModelTests`, `make umpire-build-model`, and `make umpire-check-regression` pass.
- [ ] No CI, Go facade, model-local Makefile, or Umpire3 reference is added.

## Acceptance
- [ ] Facades, walkthrough, architecture, and roadmap match the implemented Space lifecycle.
- [ ] Downstream fn-5/C8 contracts are explicit without implementation overclaim.
- [ ] Aggregate model/regression gates pass with package purity preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
