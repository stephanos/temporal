---
satisfies: [R9]
---
# fn-159-qualified-action-names-in-rules.3 Qualify every phase reference; refuse phase-member imports

## Description

Close the current Scala source/lift proof for qualified phases and phase-member import refusal.
Enforcement and the historical source migration are already integrated; re-anchor rather than
repeating them. The owner Scope limits this task to the mini feature, framework/lifter and shared
code. Standalone/Nexus/legacy sources and generated artifacts remain untouched for the sweep.

**Size:** M
**Files:** typed phase-import/rule-heading checks, isolated qualification fixtures, source-rule
proof and author docs; mini/shared source only where an unmet contract requires a change.
**Touches:** [model/irgen/Conventions.scala, model/irgen/Declarations.scala, model/irgen/test/QualifiedPhases.test.scala, model/irgen/testdata/qualifiedPhases/**, model/irgen/testdata/conventions/PhaseImports.scala, model/irgen/testdata/lifts/RuleRejects.scala, model/framework/Syntax.scala, model/temporal/features/activity_mini/**/*.scala, model/temporal/foundations/taskqueue/**/*.scala, model/migrate/**, model/README.md, model/SEMANTICS.md]

### Approach

- Begin after the committed .1 qualification closure and preserve its source-spelling/refusal
  work, focused fixture infrastructure and identities. Do not re-run broad suite baselines.
- The relevant phase type is the `P` of `Phased[S, P]`, including inherited projections reached
  by derivations/compositions, not an enum's name. Preserve unrelated enum/member imports.
  Independently test a differently named finite phase enum and accepted non-phase imports.
- Inspect current `Conventions.phaseImport` and `Declarations.qualifiedPhase`. Pin real authored
  wildcard/selective/renamed-member imports at file, machine, nested-section and rules-block levels,
  including an import written after another token on the same line. The current text check uses
  `trim.startsWith("import")`; do not assume the existing three whole-line fixtures prove all
  authored import positions. Keep compiler-generated enum companion imports distinct, not an
  excuse to suppress actual imports or malformed/source-unavailable authored positions.
- The lifter must refuse each applicable import at its actual source position and name the type
  and qualified remedy. It also refuses bare arguments of phase-form `when`, including a bare
  eta-expanded named set. Accepted `when(Phase.x)`, qualified set and role forms remain valid;
  Property-builder `when` is not this rule-case surface.
- There is no requested framework runtime phase check: varargs `when` is not inline. Preserve
  that boundary and all qualified mini effects/init/states/properties/choice alternatives.
  The native contract .1 establishes for action names is a different surface.
- Ship an actual task-named Scalafix source rule under the shared convention. Focused copied-mini
  legacy-to-qualified transformation, ambiguity/refusal, idempotence and compiled identity/meaning
  controls prove source correctness. Do not apply rules to real features or claim the held loader
  works. Reuse the existing rule-source/test harness; no competing generic rewrite engine.
- Documentation uses qualified phases and the complete mini's actual cited source. Keep the
  source-citation guard passing by fixing inaccurate excerpts/line references, not the oracle.

### Source and consumer boundary

R9's current source check is on selected mini/shared input and isolated fixtures. Real-feature
source/lift equivalence, managed IR/Case regeneration, Go/lint and canonical Make gates stay at
the sweep. No protected baseline, pin, expected IR or accepted lint debt is rewritten to obtain
source closure. Preserve the earlier task/spec history and actual failed attempts.

### Verification

Fresh mini/shared compile, focused native tests where changed and a new isolated QualifiedPhases
lifter suite are the per-step gates. The broad Fixtures class packages all fixture roots in
beforeAll, so do not invoke it merely to filter a phase test. Freeze each proof's source/jar inputs
while it runs. Scratch pre/post mini lifts must preserve selected declaration identities and
semantics, and state explicitly any justified rule-heading/source-span difference rather than
normalizing it away. Run direct `FEATURES=activity_mini` Scala steps-only equivalence against
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/`
with a separate justified mini map, never modifying either baseline or the batch-1d map. The complete
selected baseline includes shared query roots: 16 tables/11,325 rows from the pre-change task8 jar;
the first 10-table mini-reachable capture is retained unchanged.
Run focused migration-rule/source-citation checks and changed-file formatting/diff checks.
Lint/manual acceptance is NOT RUN. Broad validation/independent review is deferred to the ONE
mini batch-end gate; canonical Make/Go/regeneration/live remains held, never reported as green.

## Acceptance

- [ ] Actual phase-member imports are refused for wildcard/selective/renamed forms at all authored
  levels, including inline same-line locations, with independently asserted positions/type/remedy;
  compiler-generated companion imports and unrelated enum imports are not falsely rejected.
- [ ] Bare phase-form `when` and bare named sets are refused; qualified phase/set and role forms
  are accepted, with Property `when` unaffected.
- [ ] Selected mini/shared source contains only qualified phase references and no phase-member
  imports; its scratch lift and native tables preserve the contract, while real-feature sources,
  artifacts, pins and old baseline/map remain unchanged.
- [ ] The task's actual migration-rule source has focused transformation/refusal/idempotence and
  compiled copied-mini proof; real-feature application and production loading are explicitly held.
- [ ] README/framework/guide examples accurately show qualified phases and retain checked source
  citations. Focused source gates and direct mini native equivalence pass; any exact lift differences
  are explained and independently checked without laundering source positions/identities.
- [ ] Failed attempts and deferred canonical/consumer/lint/broad-review gates are recorded honestly;
  no full-set, live, broad-suite or SHIP credit is claimed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
