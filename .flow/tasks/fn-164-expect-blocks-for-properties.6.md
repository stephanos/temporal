---
satisfies: [R4, R5, R6, R7]
---
# fn-164-expect-blocks-for-properties.6 Remove lambda finishers and seal expectation fixtures and author docs

## Description
**Note 2026-10-18 (one-language series):** IR parity proofs use the in-process internal IR (fn-201); no checked-in IR.

**Size:** M
**Files:** public Property builder finishers, remaining framework/test fixtures and author documentation.
**Touches:** [model/framework/**, model/irgen/**, model/check/**, model/README.md, model/framework/SEMANTICS.md, model/testdata/equivalence/**]

Finish R6/R7, with a final R4/R5 integration seal, after production migration. This task releases the existing colon-style task; the latter supplies the final pre-fault seal.

### Approach
- Remove public holds/holdsAcross builder finishers together, with no compatibility layer; retain internal PropertyDecl predicate fields and core construction storage. Convert remaining framework regression callers, layout template, positive/refusal fixtures and sugar Core form examples. Direct core fixtures continue proving equivalent IR without retaining removed authoring methods.
- Distinguish intentionally invalid old-form fixtures from overlooked production callers. Update located diagnostics/spans using source mappings without weakening refusal expectations. SyntaxRule/core separation and fn-141's generic expansion handoff remain explicit.
- Update README guidance and SEMANTICS' field-reading/level/claim rules for typed after/before, Unit requirements, branch grouping, short-circuit/holes and temporal forms outside blocks. Integrate around any concurrent owner document edits; do not overwrite unrelated content.
- Freeze source/IR inventories and run all Scala framework/Model/lifter/gate tests plus scratch comparisons and step-table equivalence. Include ordinary witness-generated claims, generic capability laws and all outer predicate bridges. Comparisons pin full declaration/function inventories and allowed source spans; test green alone does not establish parity.
- Record the committed source seal and handoff to colon .1, then fn-123 Scala work and fn-141. Shared production regeneration/Go/full gates/review/live stay pending; no whole-spec closure substitution.

### Investigation targets
**Required:** `model/framework/Claims.scala:19-48`; `model/framework/Syntax.scala` Core form examples; `model/irgen/test/Fixtures.test.scala`; `model/README.md` claim/block/shared-predicate/core-sugar guidance; `model/framework/SEMANTICS.md` current claims/levels guidance; fn-164 predecessor inventories/source maps.
**Optional:** `model/check/SyntaxRule.scala`, layout template fixtures and independent equivalence baseline.

### Quick commands
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen
mise exec -- scala-cli test model/check
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Run serially; checked-in IR/Cases remain stale under owner mode. Full gate/review/live remain deferred, not passed.
## Acceptance
- [ ] Public lambda finishers are gone with no compatibility surface; complete framework/production/template/fixture inventory accounts for every old spelling and preserves internal core predicate storage (R6).
- [ ] Author docs and all sugar Core forms describe expectation contexts, requirements/branches, field reads and outer temporal forms consistently (R7).
- [ ] Complete scratch lifts have exact declaration/function parity except mapped positions, and all Scala suites/compile/syntax/step-table checks pass (R4/R5).
- [ ] The committed migration seal releases colon .1; deferred regeneration/Go/full gates/review/live and fn-141 generic lowering obligations are explicitly retained.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: