---
satisfies: [R3]
---
# fn-87-tighten-the-testpilot-protocol-glossary.5 One Expression language for Program and Contract with located context checks

## Description
Replace `ProgramExpression` and `ContractExpression` (fourteen messages) with the spec's `Expression` and `Reference` (R3, API Contracts), fold `equals` into `CompareExpression` with `EQUAL`/`NOT_EQUAL` in `ComparisonOperator`, and move the Program/Contract separation from types to a preparation check that rejects a reference outside its context with the static-preparation category and the expression's located path. The correlated predicate language moves onto the same `Expression` in .6; this task adds the correlated references to `Reference` (per the API Contracts sketch) but admits them in no context yet.

**Size:** M
**Files:** `proto/.../v1/{expression,program,instruction,contract}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/internal/ir/{expression.go,evaluate.go,expression_test.go,evaluate_test.go}`, `common/testing/testpilot/internal/execution/{dataflow.go,prepare.go}`, `common/testing/testpilot/internal/verification/{prepare.go,captures.go,evaluator.go}`, `model/Testpilot/Authoring.lean` (one `Expr` namespace), `model/Testpilot/Tests/{Protocol,Authoring,AuthoringFailures}.lean`, Lean Producers (`model/Temporal/Testpilot/CaseSupport.lean`, `model/Temporal/Case/**`, `model/Temporal/Feature/Nexus/Success/*.lean`, `model/Umpire/Case/Projection/Lowering.lean`, `model/Umpire/Variations/Lowering.lean`), conformance generator and renderer (`tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`, `model/Temporal/Testpilot/Conformance.lean`, `model/Temporal/Tool/Testpilot.lean`), `common/testing/testpilot/conformance_test.go`, fixtures, mapping
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, model/Testpilot/**, model/Temporal/**, model/Umpire/Case/**, model/Umpire/Variations/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/**]

### Approach
- Proto: `Expression { oneof expression { Value literal; Reference reference; PathExpression path; PresentExpression present; CompareExpression compare; NotExpression not; AllExpression all; AnyExpression any } }` and `Reference` exactly as the spec sketches (slot, outcome, run, environment binding, observation, run event, capture, evidence field, correlated capture, model value). `ComparisonOperator` gains `EQUAL`, `NOT_EQUAL`; `EqualsExpression` is gone. The Lean generator accepts `not`/`all`/`any` as arm names (verified by the Lean scout), so the spec names are used as written; a hand-written Lean `def not` inside `Testpilot.Authoring` would shadow `_root_.not`, so the Authoring constructor is `Expr.negate` (record the spelling).
- Go IR: `internal/ir/expression.go:78-293` already walks expressions by oneof field name; switch it to the new message and a `Reference` walk, and replace the `Equals` IR operator with `Compare(EQUAL|NOT_EQUAL)` keeping identical evaluation: `EQUAL` keeps today's equals typing (any comparable type), ordering operators stay ordered-types-only (NaN ordering stays false), and `NOT_EQUAL` is defined as `not(EQUAL)`. Admission checks operand types per operator and rejects an ordering operator on messages, lists, maps or bytes with a located error. Expression depth and work charges must equal today's for every existing Case. Keep "requires an explicit presence guard" behavior unchanged; .16 changes absent operands.
- Context admission: introduce an `ir.Context` (instruction input and guard; Contract transition predicate; correlated condition, trigger and response; evidence lift guard) carrying the admitted `Reference` kinds from the spec table. Binding a reference outside the context returns an `ir.Error` that surfaces as a `PreparationError` (the static-preparation rejection class). Use category `unknown` (`PreparationUnknown`), matching the existing scope miss "reference is not declared in this environment" (`ir/expression.go:212`), and record the choice. Located path: thread a path builder through binding so the error names e.g. `program.entrypoints[controller].instructions[start].guard.all[1].reference.observation_id`; `PreparationError.Path` keeps its existing right truncation at 256 bytes (`preparation_error.go:55-57`, pinned by `preparation_error_test.go:76-79`), so build compact paths (identifiers rather than repeated message names) that fit within 256 bytes for every checked-in Case and do not change the truncation rule; today paths are coarse (`"contract"`, `entrypoint.instruction`), so add the builder in `ir` and pass the prefix from `execution/dataflow.go:562-624` and `verification/prepare.go:192` and `captures.go:187`.
- Lean: one `Testpilot.Authoring.Expr` namespace replaces `ProgramExpr`/`ContractExpr`. `Testpilot/Tests/Protocol.lean:18-38` pins the old type mismatch; replace it with `#guard`s that a Program-context and a Contract-context expression are the same type, plus Lean-side context checks only if Authoring already validates contexts (if it does not, the Lean unit tests cover Authoring constructors and Go owns admission; the spec asks for "Lean and Go unit tests" of the rejection, so add a Lean test that renders a Case with a Contract reference in an instruction guard and a Go test that prepares it and asserts category and path).
- Conformance: the spec requires one new rejection case. EVD-18 fixes six classes, so keep six classes and allow the `static-preparation-rejection` class to carry two Cases: extend `productionManifest`/`validateManifest` (`generate.go:478-560`) with a per-class Case list, add Lean renderer argument `conformance-static-preparation-rejection-expression-context`, and make `conformance_test.go:86-90` assert category and path for the new Case (not only `require.Error`). Allowlist the new fixture path for `"bounds"` in `check.go` like its siblings if it carries that key.
- Mapping: a declared step rewrites `ProgramExpression`/`ContractExpression` trees into `Expression` (`slot|outcome|run|environment|observation|runEvent|capture` → `reference.{...}`; `equals{left,right}` → `compare{operator: COMPARISON_OPERATOR_EQUAL,...}`; `negation` → `not`). The surviving `*Ref` messages become `*Reference` (`RunRef` → `RunReference` marker, `InstructionOutcomeRef` → `InstructionOutcomeReference`, `RunEventFieldRef` → `RunEventReference`, `ObservationRef` in capture assignments → `observation_id` string or `ObservationReference`, record which). Retire tokens `ProgramExpression`, `ContractExpression`, `ProgramExpr`, `ContractExpr`, `EqualsExpression`, the `Program*`/`Contract*` operator message names, and the retired `*Ref` names (`SlotRef`, `RunRef`, `ObservationRef`, `CaptureRef`, `EnvironmentRef`, `InstructionOutcomeRef`, `RunEventFieldRef`).
- Carried from .3's review: rename `ResponseRead.kind` → `cardinality` (its type is `ReadCardinality`), with a declared mapping step.
- Docs: `model/Umpire/ARCHITECTURE.md:212-220` expression vocabulary sentence; `internal/execution/README.md` expression wording.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/ir/expression.go:17-47,78-293` — IR, reference kinds, walker
- `common/testing/testpilot/internal/ir/evaluate.go:100-160` — equals/compare evaluation
- `common/testing/testpilot/internal/execution/dataflow.go:483-624` — guard success facts and Program binding
- `common/testing/testpilot/internal/verification/prepare.go:180-245` — Contract binding scope
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:478-560` — manifest

**Optional:**
- `model/Testpilot/Tests/Protocol.lean:18-38`
- `common/testing/testpilot/preparation_error.go` — categories

### Key context
- `ir` has no Environment reference kind: environment references are rewritten to literals in `execution/dataflow.go:656-665`; keep that, and admit `environment_binding_id` only in the Program context.
- Evaluation order, short-circuiting and fail-closed behavior must not change (SEM-17, EVD-12); the equivalence test plus unchanged `expected.json` and live Verdicts are the pin.

## Acceptance
- [ ] `ProgramExpression`, `ContractExpression` and their fourteen operator messages are gone; `Expression`/`Reference` match the spec; `ComparisonOperator` has `EQUAL` and `NOT_EQUAL`
- [ ] a reference outside its context rejects at preparation with the chosen static-preparation category and a located path; Go unit tests cover each context's refused kinds; a Lean test renders such a Case
- [ ] conformance `static-preparation-rejection` carries the new expression-context Case and asserts category and path; six classes remain
- [ ] equivalence test passes with the declared expression-rewrite step; every `expected.json` and correlated `expected` unchanged; retired tokens added and gate green
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


