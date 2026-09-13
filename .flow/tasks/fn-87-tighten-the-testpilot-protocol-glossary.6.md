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
The five correlated condition messages and `guard_equals_text` are gone. A correlated rule's trigger, response and correlation, and an evidence-lift guard, are now each an `Expression`, and the correlated presence constraint is a `present` expression rather than a `bool present` (R5). No `expected`/`incomplete` value in `correlated.json`, no `expected.json` and no live Verdict moved. The oracle passes with the declared correlated steps, and the regression gate passed with 9 live identities.

**Protocol**
- `Reference` gains two arms, recorded as spec deviations in API Contracts:
  - `CorrelatedStepReference correlated_step = 11` (a `CorrelatedStepField` of ACTION/OUTCOME/STATE/FACT plus `definition_id`).
  - An empty `ProjectedValueReference projected_value = 12`.
- Naming decision: `CorrelatedStepReference`/`CorrelatedStepField` instead of the recommended `StepField`, to stay parallel with `correlated_capture` and the other `Correlated*` names.
- Deviation: `model_value` is admitted in no context. A step reference plus a text literal carries every model value a condition tests. Removing the arm is a follow-up for R6.
- A step condition is `present(correlated_step)` or `compare(EQUAL, correlated_step, text)`. A correlation is a step condition, an EQUAL/NOT_EQUAL comparison of literal, evidence-field or correlated-capture operands, or `all`/`any` of those.

**Go**
- `ir` gains `CorrelatedContext` and `EvidenceLiftContext` in its context table, the `ProjectedValueReference` kind, and `ir.AdmitReferences`. That function walks an expression and rejects an out-of-context reference at the exact path `BindExpression` reports. A test pins the paths through every nesting operator.
- Decision: the correlated capability keeps its own admission and evaluator over `Expression` (`verification/correlated_prepare.go`, `correlated.go`) rather than binding through `ir`.
  - This keeps the `Shared.CorrelatedObligation` inputs bit-for-bit the same.
  - Depth and work ceilings still count conditions exactly as before.
  - `all`/`any` still stop at the first decisive operand.
  - A trigger or response that reads the wrong step part rejects `unknown` at `...correlated_step.field`.
- Lift guards bind and evaluate through `ir` (`BindExpression`/`EvaluateExecution`) with the projected value as their only reference. Consequences:
  - Rejected now: a non-boolean guard, an unguarded absent read, and an out-of-context reference (located).
  - Admitted now: the former "presence-selector or empty guard path" rejections are gone, because such a path is now a valid boolean read. Their tests were replaced by nonboolean/unguarded/missing-guard cases.
  - Guard evaluation charges a few more runtime work units per rule than the bare path read did. No checked-in Case is near a ceiling.
- Tests:
  - `TestCorrelatedPrepareLocatesConditionsOutsideTheCorrelatedContext`: every foreign arm in trigger, response and correlation, plus the wrong-field trigger and response.
  - `TestEvidenceLiftGuardRejectsReferencesOutsideItsContext`.
  - `TestAdmitReferencesLocatesLikeBinding`.
  - Both context tests were confirmed red with the checks loosened.

**Lean**
- `Testpilot.Correlated.decode` reads the new shape and rejects foreign reference arms. Theorem statements are unchanged: `admitMany_append`'s proof was untouched.
- `Testpilot.Authoring` adds `Expr.evidenceField`, `correlatedCapture`, `correlatedStep` and `projectedValue`, and `correlatedEvidenceRule` takes an `Expression` guard.
- `Umpire.Case.Correlated` lowers clauses to `Expression`.
- The Temporal evidence lift and the typed Nexus Producer emit `present(path(projected_value, p))`, or `all[present, compare EQUAL]` where they need a text match.
- `Tests/Fields.lean` gains context-rejection `#guard`s.

**Fixtures, oracle and vocabulary**
- `correlated.json`, `typed-nexus-case.json` and `async-nexus-case.json` were regenerated through their generators.
- The oracle declares three things:
  - predicate → step-condition rewrite;
  - operand/comparison/correlation → Expression rewrite;
  - lift guard → Expression rewrite.
- Those steps are tested, including their refusal cases. A mutation check (wrong guard operator) made the oracle fail.
- Retired names: the seven message and enum names, `guard_equals_text`/`GuardEqualsText`, and the `CORRELATED_PREDICATE_FIELD_*` / `CORRELATED_COMPARISON_OPERATOR_*` families. They are also in `protocol_test`. The token list lives in `tools/umpire/internal/retiredvocabulary/check.go`, which is outside Touches but is where the acceptance criteria require them.
- Docs: `internal/execution/README.md` (the lift guard), `internal/verification/README.md` and `model/Umpire/ARCHITECTURE.md`. The spec's Planning decisions gained "Correlated conditions and lift guards (decided in .6)".

**Gates**
- Baseline was green via the receipts at 13294110.
- Regression gate:
  - Run 1 (before the lint fix): exit 0, 9 live identities.
  - Runs 2 and 3 on HEAD: `TestTestpilotTypedNexusOperationsCase` failed with the known evidence-ordering flake (history interleaving; the test passed 5/5 when run alone).
  - Run 4: exit 0, 9 live identities.
- `lint-code` shows 161 issues after `go clean -cache`, which is the baseline. Two new findings were fixed before that run: gci formatting in `evidence_lift_test.go` and a missing switch default in `ir`. `lint-model` shows 163, the baseline.

**Follow-ups (reviewer P3s, not applied after SHIP)**
- `predicate` re-parses the step condition that `correlationHolds` already read. Parse once at preparation instead.
- The located-path 256-byte truncation in `correlated_prepare.go` duplicates `ir`'s. Export a constructor from `ir` instead.

stage: impl-review - ran (claude backend, SHIP on first round, two P3)
## Evidence
- Commits: a1c1ec7d7c09de340250bf02c288bcc616bced5e
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/internal/retiredvocabulary/, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (run 1 pre-lint-fix: exit 0, 9 live identities; on HEAD run 2 and 3: TestTestpilotTypedNexusOperationsCase evidence-ordering known flake; run 4: exit 0, 9 live identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline), make lint-model (163 errors, baseline)
- PRs: