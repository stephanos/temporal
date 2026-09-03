---
satisfies: [R4, R5, R6]
---
# fn-57-partition-the-observation-authoring.3 Document and verify the Observation authoring partition

## Description
Update internal architecture navigation to name Declaration and Compiler while continuing to direct ordinary authors to the stable Observation facade. Audit imports, comments, warnings, trust, and downstream compatibility, then run aggregate gates.

**Size:** S
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md]

### Approach
- Document Declaration as inert vocabulary and Compiler as the owner of DefinitionGraph-backed checking and canonical plan construction.
- Keep public author guidance pointed at `Umpire.Observation`; do not rewrite unchanged lifecycle or semantic guidance.
- Update DefinitionGraph ownership wording from the monolithic language implementation to Compiler.
- Verify the repository README remains accurate and edit it only if an internal-ownership statement became false.
- Audit preserved comments, public docstrings, import direction, warnings, and axiom/trust dependencies.
- Run focused, direct-consumer, aggregate model, regression, and lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:8-42` — public versus internal imports
- `model/Umpire/ARCHITECTURE.md:249-309` — public Observation lifecycle
- `model/ARCHITECTURE.md:157-165` — DefinitionGraph import ownership
- `model/ARCHITECTURE.md:410-435` — internal facade implementations
- `model/README.md:319-359` — public authoring guidance to verify

### Key context
- Documentation records ownership, not implementation status. Preserve unchanged user-facing semantics.
- Fn-54 follows this task and may extend the internal Evaluation module list.

## Acceptance
- [ ] R4-R6 are satisfied by architecture navigation that names Declaration and Compiler and still directs ordinary callers to the stable Observation facade.
- [ ] DefinitionGraph, checked-plan, and canonicalization ownership descriptions match the final import graph; unchanged public lifecycle text remains intact.
- [ ] Existing comments and docstrings are preserved and accurate, with no warning, import-boundary, or trust regression.
- [ ] Public facade, checked examples, canonical plan/error rendering, connected-target reconciliation, and downstream Observation behavior remain unchanged.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests UmpireTests TemporalModelTests` passes.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, and `make lint-code` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
