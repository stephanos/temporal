---
satisfies: [R13]
---
# fn-159-qualified-action-names-in-rules.6 Drop Observation's free-text read field

## Description

Close the current source/API/lift proof for `Observation(name, on)`. The removal is already
integrated in commit `50bc14c2fa9deec636528c0524dcd6cf0a851f90`; inspect and reuse it, rather
than deleting the field again. Its parent contains the actual retired three-argument signature.
The current owner Scope permits the framework/lifter, shared code and `features/activity_mini`
only. Real-feature uses, realizations, lift proofs and consumer closure remain at the sweep.

**Size:** S
**Files:** isolated Observation API/name/typed-read controls, task-named migration/source proof,
current framework/lifter comments and author docs where a demonstrated gap requires a change.
**Touches:** [model/framework/Action.scala, model/framework/Observation.test.scala, model/irgen/Constants.scala, model/irgen/test/Observations.test.scala, model/irgen/testdata/observations/**, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Start from local `umpire` after the committed central-Bounds source checkpoint. Re-anchor the
  current two-field API, named-by-val folding in `Constants.constString`, source comments and
  README. The mini has no author-maintained `Observation` declaration; do not add an artificial
  observation to the permanent mini merely to make this task's proof look exercised.
- Independently prove the native API has only name/on and rejects the retired named `read`
  argument. Prove inferred and explicit observation names, aliases and machine evidence references
  in isolated compiled/lifted fixtures, rather than merely searching for the removed parameter.
- A typed `Observed[Message]` names the protobuf message; the actual read's typed response selector
  names the response field. Do not claim an `Observed` token by itself proves that field.
  An isolated realization/read fixture must independently assert the observation identity/message,
  exact typed response path and matching hint comment. Include an independent wrong/invalid-path
  control at the existing compiler/lifter boundary, without adding a new runtime policy.
- Verify the retired text was not carried by the Scala lowering or shared IR observation transport.
  Existing `Constants` folds the declaration's first name argument; the schema's `Observed` carries
  id/position/message, and actual read paths are separate metadata. Bounded read-only inspection
  of existing Go lint/lowering may establish their use of observation ids/messages and read targets,
  but no Go tests/changes, live/Case verdict or whole-consumer credit is authorized. If a concrete
  consumer of the old free text is found, report it before changing behavior.
- Ship actual compiled `Fn159Task6...` Scalafix source under the established convention. Remove
  the provably literal retired label from supported Observation constructors, preserving name/on,
  typed read selectors, comments and unrelated APIs. Ambiguous/shadowed constructors, unsupported
  argument shapes and nonliteral expressions whose evaluation cannot safely be dropped refuse
  atomically with a concrete manual recipe. Prove transformation, idempotence and refusal controls.
- Reuse the existing `model/migrate` harness. A copied-mini source proof may carry an explicitly
  isolated scratch Observation probe alongside the copied feature, since the actual mini has no
  Observation call to migrate. The actual compiled rule must transform that probe and restore its
  exact accepted source. Compile/lift the restored probe and copied mini independently. Label
  scratch-probe evidence separately; do not claim a real mini observation was changed or exercised.
  If retired-API pre/post compilation is needed, use the actual old signature only in an isolated
  scratch framework, never the current checkout or real-feature tree. Do not normalize away
  metadata/positions or substitute a no-op run for transformation evidence.
- Keep framework comments/README accurate and all thirty source-cited mini guide excerpts exact.
  Existing integrated documentation may already satisfy this; no manufactured source edits.

### Verification and boundaries

Fresh selected mini/shared compile, focused native Observation API controls, isolated name/typed-read
lifter fixtures, focused migration/source-citation tests, exact copied-source restoration and complete
raw mini IR/waiver comparison are this source checkpoint's gates. Do not invoke the broad Fixtures
class to filter one case. Freeze source/jar inputs before proofs and do not overlap formatting/edits.
Use matching immutable jar-based classpaths rather than mutable Scala compiler output roots.
Independently check emitted fixture metadata and preserved mini declarations/positions.

Run direct `FEATURES=activity_mini` Scala `--check-equivalence ... --steps-only` against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified task map. It is the complete pre-change task8-jar selected capture,
sixteen tables/11,325 rows including shared Query roots. Both this and the retained original
ten-table baseline must pass their integrity scripts; neither capture nor the old batch-1d map
may be changed or recaptured. Run changed-file formatting/diff checks and retain actual failed
attempts. Checked-in generated IR/Cases and real-feature/legacy payloads stay untouched.

Real-feature observation removals/hint proofs, managed regeneration and canonical/consumer gates
remain held; existing Duration/protobuf prerequisites do not block this source-only closure.
No Go/proto/schema/Make/CI/runtime/Case/live/production loader changes are authorized.
Lint/manual acceptance is NOT RUN. Broad validation and independent review run at the ONE
Scala batch-end gate, not here. No SHIP, canonical-green, full-consumer or real-feature credit.

## Acceptance

- [ ] The native two-field API and retired-argument refusal are independently tested; isolated
  compiled/lifted inferred/explicit observation names, aliases and machine evidence stay correct.
- [ ] Typed observation identity/message and the actual response selector/path are independently
  asserted, with a meaningful invalid-path control and an accurate nearby response-field hint.
- [ ] Bounded source/schema/consumer inspection finds no reader of the retired free text; actual
  typed read metadata is distinguished from that inert label. Unrun consumer gates stay held.
- [ ] Task-named compiled Scalafix source has real literal-label transformation, idempotence and
  atomic ambiguity/nonliteral refusals; an explicitly isolated copied-mini probe proves source
  restoration and compilation/lift, without inventing an authored mini observation.
- [ ] Fresh focused source gates, raw complete mini IR/waiver preservation, direct sixteen-table
  native equivalence and both frozen baseline integrity checks pass; protected sources/artifacts,
  pins, positions and historical map are unchanged.
- [ ] Framework/lifter/README comments remain accurate and exact guide citations pass; failed
  attempts, owner-held real/consumer obligations and NOT RUN lint/broad-review gates are recorded
  honestly rather than reported as passing.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
