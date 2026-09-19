---
satisfies: [R1]
---
# fn-83-author-a-live-case-from-a-model-file.1 Extract the generic Producer from the Nexus success Producer

## Description
Move the Producer-neutral half of the Nexus success Producer into `Umpire.Case.Producer` (R1) with the `Producer.Input`, `Identity`, `Realization`, `EvidenceSource`, `EvidenceMapping`, `FaultLine`, and `Hook` types from the spec's API Contracts, and rewire the success Producer to call it with a hand-built `Realization` and today's identities. This is the early proof point: the async-Nexus fixture must regenerate byte-identical through the extracted code before any syntax exists. Split from .2 so the Lean-value seam is proved before the template and the command are built on it.

Paths below are the tree before fn-82's moves; after fn-82 the success Producer is `model/Temporal/Feature/Nexus/Success/Producer.lean` and `Umpire.Space` is `Umpire.Variations`.

**Size:** M
**Files:** `model/Umpire/Case/Producer.lean` (new), `model/Umpire/Case/Tests/Producer.lean` (new), `model/Umpire/Case.lean` (facade export), `model/Temporal/Feature/Nexus/Success/Producer.lean` (reduced to a `Realization`, an `Input` conversion, and one `produce` call), `model/Temporal/Feature/Nexus/Success/Tests.lean` (the part-substitution helper), `model/ModelLint` classifier entry if `Umpire.Case.Producer` needs a class, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Case/**, model/Umpire/Case.lean, model/Temporal/Feature/Nexus/Success/Producer.lean, model/Temporal/Feature/Nexus/Success/Tests.lean, model/UmpireTests.lean, model/ModelLint/**]

### Approach
- The Producer cannot take the Temporal authoring bundle: `ModelLint` forbids any `.umpire` module importing Temporal, and the only bundle that carries target, vocabulary, property, behavior, query, and witness is the Temporal one. Define `Umpire.Case.Producer.Input` per the spec (checked Model, vocabulary as `ModelValue` lists, checked Property, checked Scenario, witness trace, Query ID/source/Known Gaps) and write the conversion from the Temporal bundle beside the success Producer. The success tests' part-substitution helper now substitutes fields of the Temporal bundle before conversion; keep it compiling.
- Generic code to move unchanged: `loweringError`, `predicateField`, `predicateOf`, `patternHolds`, `scopedClauseOf` (keep the `property.clause-early-response` vacuity rule and the `-shape`/`-occurrence`/`-form` rejections), and the body of `produce` (witness-absent, `behavior.sequence.absent`, scoped compile, `Umpire.Case.Scoped.lower`, the coverage `eraseDups`).
- Evidence rules: `Realization.program` is a function of `Identity` and the resolved `EvidenceRule` list, so the history node's projection targets are built from the mapping. In this task the success Producer's `Realization` carries the two Nexus `EvidenceSource`s (attributes field, `scheduled_event_id` operation-key path, kind ID, source ID) and the hand-built mapping `awaitStart ↦ started`, `awaitSuccess ↦ completed`; `projectionDeclaration`'s hand-indexed `vocabulary.actionAt 0/1` rules are replaced by a derivation from the mapping plus the witness trace (the Step the witness takes on each mapped Action supplies state, outcome, facts).
- `Identity` is passed explicitly with today's values (`temporal.case.async-nexus-success`, program ID `temporal.case.async-nexus.program`, contract ID, scope literal `async-nexus`) so bytes do not move in this task; the derivation rule arrives with the `case` block in .3.
- The Producer must not import `Temporal.Testpilot.CaseSupport`; role IDs, binding IDs, and RPC method strings stay in the success Producer for now (.2 moves them into the template).
- Behavior pin: `make umpire-gen-case-runtime-conformance` then `git diff --exit-code tests/testcore/testpilot/testdata/async-nexus-case.json`. No diff is acceptable in this task.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus3/Testpilot.lean:188-364` — the generic half to extract and the `produce` body
- `model/Temporal/Feature/Nexus3/Testpilot.lean:59-113, 274-292` — evidence rules, the history node's projection target, and the hand-indexed projection declaration
- `model/Temporal/Feature/Nexus3/Authoring.lean` — the Temporal bundle (`CheckedModel`, `family`, `source`) the conversion reads
- `model/ModelLint/ImportGraph.lean:280-295` — the `.umpire` → Temporal import prohibition
- `model/Umpire/Case/Compiler.lean:60-97` — `Input` and `compile`, the Producer's output

**Optional:**
- `model/Temporal/Feature/Nexus3/Tests.lean:16-40` — `#guard match` pins and the part-substitution helper

### Key context
- SCP-02 and MOD-01 are enforced by `make lint-model`; run it before claiming done.
- Memory: diagnostics must retain source coordinates at nested boundaries; empty extensions must not change fingerprints.
## Acceptance
- [ ] `Umpire.Case.Producer` exists with `Input`, `Identity`, `Realization`, `EvidenceSource`, `EvidenceRule`, `EvidenceMapping`, `FaultLine`, `Hook`, and `produce`; `lake build` and `make lint-model` pass
- [ ] The success Producer is a `Realization` value, an `Input` conversion, and one `produce` call; no lowering logic remains in it
- [ ] `async-nexus-case.json` regenerates byte-identical
- [ ] Existing `#guard` and `#guard_msgs` blocks in the success tests pass, with the part-substitution helper adapted
- [ ] `make umpire-check-case-runtime-conformance` passes
## Done summary
Extracted the Producer-neutral half of the Nexus success Producer into `Umpire.Case.Producer`.

- `model/Umpire/Case/Producer.lean` (new): `Vocabulary`, `Input`, `Identity`, `HookPlacement`,
  `Hook`, `EvidenceSource`, `EvidenceRule`, `EvidenceMapping`, `FaultKind`, `FaultLine`,
  `Realization`, and `produce`. The correlated-clause derivation (`predicateField`, `predicateOf`,
  `patternHolds`, `scopedClauseOf` incl. the `property.clause-early-response` vacuity rule),
  the projection declaration, provenance, and coverage moved unchanged. It names no protocol,
  history attribute, or role, so it holds under SCP-02/MOD-01.
- `model/Temporal/Feature/Nexus/Success/Producer.lean`: reduced to a `Realization` value, an
  `Input` conversion (`producerInput`), the evidence mapping, and one `Case.Producer.produce` call.
  No lowering logic remains.
- `model/Umpire/Case/Tests/Producer.lean` (new) + `model/UmpireTests.lean`: pins for the identity
  derivation and vocabulary spelling resolution.
- `model/Umpire/Case.lean`: facade export.

Deliberate deviations from the spec's API Contracts block, both to keep this task byte-neutral:
- `Input.witness` is `Option Trace`, not `Trace` — the "no witness" rejection is a production-time
  diagnostic per Edge Cases, so the Producer must see the absence.
- `Input` additionally carries `operationRole`, `queryFingerprint`, and `source`; `Realization`
  carries `projectionId`, `producerId`, `correlatedObservation`, and the three limit records
  instead of the spec's `roles`/`environment` (which `Realization.program` already supplies).
- `Identity` carries `programId`/`contractId`/`runScope` with the fixture-derived values as field
  defaults, so the success Case can state the one identity (`temporal.case.async-nexus.program`)
  that predates the convention while `.3` gets the derivation for free.

Early proof point holds: `async-nexus-case.json` regenerates byte-identical
(`git diff --exit-code tests/testcore/testpilot/testdata/` clean after
`make umpire-gen-case-runtime-conformance`). No Nexus-specific branch was needed inside
`Umpire.Case.Producer`.

Review: SHIP. The pinned reviewer `claude:claude-fable-5-1:high` returned an account limit
("You've reached your Fable limit"), so the review ran on `claude:claude-sonnet-4-5:high`.
This is a same-family fallback, not an equivalent cross-family review.

stage: impl-review - ran (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bcfe3f5c40
- Tests: cd model && mise exec -- lake build, make umpire-gen-case-runtime-conformance && git diff --exit-code tests/testcore/testpilot/testdata/, make umpire-check-case-runtime-conformance, make lint-model (0 findings outside generated Temporal/API/Proto.lean)
- PRs: