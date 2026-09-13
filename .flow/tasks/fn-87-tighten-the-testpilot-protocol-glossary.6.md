---
satisfies: [R3, R5]
---
# fn-87-tighten-the-testpilot-protocol-glossary.6 Correlated conditions and evidence-lift guards on the one Expression

## Description
Finish R3 on the correlated side: `CorrelatedPredicate`, `CorrelatedComparison`, `CorrelatedOperand`, `CorrelatedCorrelation` and `CorrelatedCorrelationGroup` become `Expression` values over the evidence-field, correlated-capture and model-value references .5 declared; evidence-lift guards become `Expression`s over the projected value and `guard_equals_text` is removed. This also makes the correlated `present` constraint a presence marker instead of a `bool present` whose `false` means nothing (R5). Split from .5 because the correlated admission and evaluation are hand-rolled outside `ir` and shared with Lean semantics.

**Size:** M
**Files:** `proto/.../v1/correlated.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/internal/verification/{correlated_prepare.go,correlated.go,correlated_test.go,correlated_captures_test.go,nexus_correlation_test.go}`, `common/testing/testpilot/internal/execution/{dataflow.go,projection.go,program.go,evidence_lift_test.go}`, `common/testing/testpilot/internal/ir/expression.go` (context table), `model/Testpilot/{Correlated,Authoring}.lean`, `model/Umpire/Case/{Correlated,CorrelatedProofs}.lean`, `model/Umpire/Case/Tests/CorrelatedFixtures.lean`, `model/Temporal/Case/Evidence.lean`, typed Nexus Producer, `correlated.json` corpus and typed Nexus fixture, mapping, `internal/execution/README.md:148-154`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/internal/**, common/testing/testpilot/testdata/**, model/Testpilot/**, model/Umpire/Case/**, model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/**, tests/testcore/testpilot/**]

### Approach
- `Reference` needs two arms the spec's sketch lacks (record both as spec deviations in the done summary and in the spec's API Contracts note): a correlated predicate is an existential over the admitted step's action, outcome, state or fact list (`verification/correlated.go:293`), which a literal `model_value` cannot express, and a lift guard needs an operand for "the projected value". Recommended: `CorrelatedStepReference { StepField field; string definition_id; }` (`STEP_FIELD_ACTION|OUTCOME|STATE|FACT`) resolving to the step's model value for that definition, absent when the step carries none, admitted only in the correlated context; and an empty `ProjectedValueReference {}` marker admitted only in the evidence-lift context (marker messages that select a oneof arm are allowed by Edge Cases "Presence"). The spec's `model_value` arm stays for literal model values.
- Encoding: predicate `{field, definition_id, present: true}` → `present { reference.step {field, definition_id} }`; predicate `{..., equals_text: t}` → `compare { EQUAL, reference.step {...}, literal.text_value t }`; `CorrelatedOperand` literal/field/capture → `literal` / `reference.evidence_field_id` / `reference.correlated_capture {capture_id, ordinal}`; `CorrelatedCorrelation` all/any/predicate/comparison → `all`/`any`/the above. `CorrelatedRule.trigger`, `response`, `correlation` become `Expression`. Keep the admission rule that a trigger reads only the action (`correlated_prepare.go:32`): a trigger referencing another step field rejects with a located path.
- Evaluation must stay identical: `all`/`any` left to right, stopping at the first decisive operand so an operand made irrelevant is never read (`correlated.proto` comment); predicate no-match is false; a missing field or capture stays a Malformed error until .16. Port `verification/correlated.go:293-370` onto the context-aware evaluator, or keep a correlated evaluator that consumes `Expression` with the correlated context; choose whichever keeps `Shared.CorrelatedObligation` inputs (`Match.trigger/response`) bit-for-bit the same, and record it.
- Evidence lift: `CorrelatedEvidenceRule.guard` becomes `Expression` over `path` from the projected value (the evidence-lift context admits only paths over the projected value, per the spec table); `guard_equals_text` folds into `compare EQUAL`. The guard's operand is `reference.projected_value`; today's rule "fires where the path resolves" becomes `present(path(projected_value, p))`; a guard whose expression is not boolean rejects at preparation; with equality it is `all[present(path), compare(EQUAL, path, literal)]` (the presence conjunct is dropped by .16). Admission at `execution/dataflow.go:339-464`, runtime at `execution/projection.go:215-217`.
- Lean: `Testpilot.Correlated` (`decode` at `:211`, theorem `:492`) decodes the new shape; keep the theorem's statement and fix its proof; `Umpire/Case/Correlated.lean` lowers clauses to `Expression`. Axiom inventories pinned by `#print axioms` in tests must not change.
- Budgets: correlated work and depth limits (`max_obligation_work`, `max_correlation_depth`) differ from `ir`'s charge; charge correlated expressions exactly as today, and treat any moved LIMIT diagnostic or rejection as a finding to fix, not to accept.
- Context table in `ir`: correlated condition/trigger/response admit evidence field, correlated capture, model value; evidence-lift guard admits path over the projected value only. Add Go unit tests for a slot reference in a correlated trigger and an observation reference in a lift guard rejecting with located paths.
- Mapping step: rewrite correlated predicates, comparisons, operands, correlations and lift guards as above (declared per shape). Rename `CorrelatedCaptureRef` → `CorrelatedCaptureReference`. Retire `CorrelatedCaptureRef`, `CorrelatedPredicate`, `CorrelatedPredicateField`, `CORRELATED_PREDICATE_FIELD_`, `CorrelatedComparison`, `CorrelatedComparisonOperator`, `CORRELATED_COMPARISON_OPERATOR_`, `CorrelatedOperand`, `CorrelatedCorrelation`, `CorrelatedCorrelationGroup`, `guard_equals_text`, `guardEqualsText`, `GuardEqualsText`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/verification/correlated_prepare.go:28-300` — predicate/operand/correlation admission
- `common/testing/testpilot/internal/verification/correlated.go:290-370` — evaluation
- `common/testing/testpilot/internal/execution/dataflow.go:339-464`, `projection.go:200-280` — evidence lift
- `model/Testpilot/Correlated.lean:90-220,480-512`
- `model/Umpire/Case/Correlated.lean` — lowering of clauses

**Optional:**
- `model/Shared/CorrelatedObligation.lean` — semantics that must receive identical `Match` values
- `common/testing/testpilot/internal/execution/evidence_lift_test.go:90,220`

### Key context
- `correlated.json` (15 entries) and the typed Nexus fixture are the Verdict pins here: every entry's `expected` and `incomplete` unchanged.
- `Value.natural_value` is produced by the runtime for correlated evidence (`execution/projection.go:268-279`); do not touch it here (.9).

## Acceptance
- [ ] the five correlated condition messages and `guard_equals_text` are gone; correlated trigger, response, correlation and lift guards are `Expression`s; the correlated presence constraint is a `present` expression
- [ ] correlated and lift-guard context violations reject at preparation with located paths (Go unit tests); Lean `Testpilot.Correlated` decodes the new shape with unchanged theorem statements and axiom pins
- [ ] equivalence test passes with the declared correlated-rewrite steps; `correlated.json` expectations, `expected.json` files and live Verdicts unchanged
- [ ] retired tokens added; `make umpire-check-retired-vocabulary` green; docs at `internal/execution/README.md:148-154` updated
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
