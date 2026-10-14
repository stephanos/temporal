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
`ProgramExpression`, `ContractExpression` and their fourteen operator messages are replaced by one `Expression` over one `Reference`, matching the spec sketch. `equals` is now `CompareExpression` with `COMPARISON_OPERATOR_EQUAL`, and `NOT_EQUAL` is defined as its negation. The Program/Contract separation moved from types to binding: a reference outside its context rejects at preparation with category `unknown` at a located path, for example `program.entrypoints[controller].instructions[execute].guard.reference.observation_id`. Every `expected.json` and correlated `expected` entry is unchanged, the oracle passes with the declared steps, and the regression gate passed with 9 live identities.

**Protocol**
- `ComparisonOperator` numbering is `EQUAL=1, NOT_EQUAL=2`, then the orderings at 3–6.
- `Reference` has all ten arms. The correlated ones (`evidence_field_id`, `correlated_capture`, `model_value`) are admitted in no context yet.
- Moved ahead of .6 so its `correlated.proto` can import `expression.proto` without an import cycle:
  - `ModelValue` moves to `value.proto`.
  - `CorrelatedCaptureRef` becomes `CorrelatedCaptureReference`, now in `expression.proto`.
- `ContractCaptureAssignment.observation` becomes `observation_id` (a string), and `ResponseRead.kind` becomes `cardinality`.
- Decision: the api-linter rejects the field name `not` (`core::0140::reserved-words`). The field keeps the spec's name under a field-level suppression, and the Lean constructor is `Expr.negate`.

**Go `ir`**
- New types: `ir.Context` (`ProgramContext`, `ContractContext`), `admittedReferences`, `ir.Site{Context, Path}`, and `Condition.Path`.
- The binders now take a `Site`. Paths are built from identities (entrypoint, instruction, rule, transition ids) plus operator segments: `all[i]`, `present`, `not`, `compare.left`, `path.operand`, `reference.<arm>`.
- Two errors are located:
  - the context rejection;
  - an ordering operator on a message, list, map or bytes operand (`type_mismatch` at `.compare`).
- Decision: all other expression errors keep their coarse paths. `preparation_error_test` pins `expression` for the presence-guard error.
- Decision: `ir` defines only the two contexts that have callers. .6 adds the correlated and evidence-lift contexts together with the references they admit.
- Tests:
  - `TestExpressionContextsRejectReferencesOutsideThem`: every arm in both contexts; confirmed red with the check disabled.
  - `TestComparisonOperatorsAdmitTheirOperandTypes`.
  - NOT_EQUAL result and work equal to EQUAL's, in `evaluate_test`.
  - `TestPrepareLocatesAReferenceOutsideTheProgramContext`: guard, request assignment, cleanup.
  - `TestPrepareLocatesAReferenceOutsideTheContractContext`.

**Work charges, measured on every checked-in Case before and after**
- Unchanged: bind count, logical depth, IR node counts, runtime work limits, and Contract per-event work bounds.
- Evaluation is the same IR.
- Deviation, recorded as a decision: surface and admission work grow, because the spec's shape adds one message per outcome, Run and Run Event reference and one enum per equality.
  - A success guard's binding work goes from 32 to 35.
  - typed-nexus Program admission goes from 2572 to 2698 against a ceiling of 100,000.

**Lean**
- One `Testpilot.Authoring.Expr` namespace. `Expr.equal` is `compare EQUAL`.
- Producers and tests were rewritten mechanically, and `lake build` is green.
- `Testpilot/Tests/AuthoringFailures.lean` is deleted, because the type mismatches it pinned no longer exist. `Tests/Protocol.lean` now `#guard`s that a guard reading an Observation and a predicate reading a Slot are the same type.
- `Tests/ProtoJSON.lean` renders a Case with an Observation reference in an instruction guard.

**Conformance**
- `static-preparation-rejection/expression-context/` is a second Case of that class, rendered by `conformance-static-preparation-rejection-expression-context`.
- The manifest has `Variant`. `validateManifest` requires six classes, each with a root Case; it is tested.
- `expected.json` gains an optional `preparationError{category,path}`, which `conformance_test.go` asserts. The existing `expected.json` bytes are unchanged, and the new fixture has no `"bounds"` key.

**Equivalence oracle**
- New declared steps: the comparison renumbering, and the expression rewrite into `reference` / `compare` / `not`. That step fails on two arms or on an extra key, and a mutation check made the oracle fail.
- Also declared: `PathExpression.source` → `operand`, the capture assignment `observation_id`, `CorrelatedCaptureReference`, and `ResponseRead.cardinality`.
- New `Added` list, tested both ways, declares the added fixture.

**Retired vocabulary**
- Retired: the two expression messages, both Lean `*Expr` namespaces, `EqualsExpression`, all 14 per-context operator messages, and the eight `*Ref` names.
- `protocol_test` lists them as retired descriptor names.
- Docs: `model/Umpire/ARCHITECTURE.md`, `internal/execution/README.md`, and the protocolmigration README.
- The spec's Planning decisions gained "One Expression (decided in .5)".

**Gates**
- Baseline was green: the quick commands ran, and the regression receipt at cc686825 was honored.
- After the change these pass: the oracle, the testpilot-protocol, testpilot-authoring, case-runtime-conformance and retired-vocabulary checks, and all Go tests in common/testing/testpilot, tests/testcore/testpilot and tools/umpire.
- `make umpire-check-regression`:
  - Runs 1 and 2 failed on one live identity each. Run 1 was `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone` and run 2 was `TestTestpilotAsyncNexusCase`; in both, the async-nexus Run's Verdict was INCONCLUSIVE instead of SATISFIED. This is not in the known-flake list.
  - I reproduced both at base 1bbb40a0a3 from a `git archive` copy with the same assertion: 1 failure each in 10 runs. So it is pre-existing.
  - Run 3 exited 0 with 9 passing live identities.
- `lint-code` shows 161 issues after `go clean -cache` and `lint-model` 163, both the baselines.

**Follow-ups**
- Reviewer P3s, not applied after SHIP:
  - Locate the environment-reference-in-wrong-position error, whose detail still says "unknown expression variant".
  - Make the path grammar uniform (`present`/`not` omit `.operand`, while `path.operand` keeps it).
  - The manifest test indexes `productionManifest()` by position.
- The async-nexus concurrent-Run INCONCLUSIVE flake should be investigated or added to the known-flake list.

stage: impl-review - ran (claude backend, SHIP on first round, two P3)
## Evidence
- Commits: 13294110c9296b3c54294187c91d12f4d65c812c
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/..., make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (run 1 and 2 failed on async-nexus inconclusive flake reproduced at base 1bbb40a0a3; run 3 exit 0, 9 passing live identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline 161), make lint-model (163 errors, baseline 163)
- PRs: