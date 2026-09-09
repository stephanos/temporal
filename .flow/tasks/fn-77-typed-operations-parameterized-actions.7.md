---
satisfies: [R2, R6, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.7 Support exact portable field and capture evaluation

## Description
Support exact portable field and capture evaluation for the referenced parent requirements.

**Size:** M
**Files:** model/Testpilot/**; proto/internal/temporal/server/api/testpilot/v1/**; api/testpilot/v1/**; common/testing/testpilot/internal/ir/**; common/testing/testpilot/internal/verification/**; common/testing/testpilot/internal/execution/**; model/Umpire/Case/Tests/**
**Touches:** [model/Testpilot/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/execution/**, model/Umpire/Case/Tests/**]

### Approach
- Map the required task3/5/6 fragment onto existing portable values, Program/Contract paths, expressions and captures; inventory demonstrably missing forms before adding any versioned generic capability.
- Implement only necessary generic schema/value/capture support in existing codec/admission/evaluator owners, preserving disjoint Program/Contract contexts and declared Observation-only Contract access.
- Range-check Lean numbers and schema identities before encoding, reject unknown/stale capability fields and enforce typed projected-value/capture work. Preserve existing Case1.0 and default-empty meaning where unchanged.
- Add independently derived Lean/Go fixtures for exact bytes, optional/default, oneof, nested/repeated/maps, wrong schemas, malformed/incomplete evidence and bounded failures; demonstrate online/offline parity. Regenerate complete protocol surfaces if annotated/proto inputs change.

### Investigation targets
**Required:**
- model/Testpilot/Authoring.lean:35 — existing portable concrete value constructors.
- model/Testpilot/Protocol.lean — generated protocol authority.
- common/testing/testpilot/internal/ir/expression.go:78 — typed expression binding.
- common/testing/testpilot/internal/ir/runtime_value.go — codec validation.
- common/testing/testpilot/internal/verification/scoped.go — fn78 run-local scoped evaluation.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/internal/ir ./common/testing/testpilot/internal/verification ./common/testing/testpilot/internal/execution`
`cd model && mise exec -- lake build Testpilot.Tests Umpire.Case.CompilerTests`
`make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance`

`cd model && mise exec -- lake build Testpilot.Tests.Fields`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Testpilot.Tests.Fields into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Required field/capture forms execute through generic existing owners; any new capability has a closed versioned reason and unknown/stale rejection.
- [ ] Independent expected values pin exact codec/evaluator behavior and Program/Contract availability across positive, violated, incomplete and malformed cases.
- [ ] Numeric/schema/capture/work limits reject atomically, with no private Slot/raw payload access or model-specific runtime branches.
- [ ] Affected Lean/Go and owned-generation staleness checks pass; unchanged protocol/Case fixtures retain bytes.

## Done summary
The portable scoped capability now carries keyed field captures and a closed correlation, and a
capture's own coordinates establish their presence so optional and oneof-selected fields can be
captured.

The inventory this task started from: the generic Program/Contract IR already carries every value
form (bytes, all integer kinds, enum, list, map, Any), every `FieldPath` selector (field, repeated
wildcard, map key, presence, oneof), a closed `ContractExpression` with path/present/equals/compare,
and `ContractCaptureDefinition`/`CaptureRef` captures. The demonstrably missing forms were all in
the *scoped* fragment, whose clauses carried only a trigger and a response predicate over the
static transition row and dropped every evidence field value after validating it.

`ScopedClause` gained default-empty `captures` and a `correlation`; `ScopedLimits` gained
`max_captures` and `max_correlation_depth`, required only when a clause declares one of them. A
capability that declares neither keeps its exact existing encoding and meaning -- the conformance
fixtures and the Case artifacts are byte-identical. `ScopedCorrelation` is a closed
predicate/comparison/all/any vocabulary over declared evidence only: an exact literal, one declared
retained evidence field of the step being admitted, or one retained earlier occurrence of a declared
capture. No private Slot or raw payload is reachable from it.

Both interpreters implement the same semantics. `Testpilot.Scoped` decodes and evaluates it in Lean;
`verification/scoped.go` and `scoped_prepare.go` do so in Go. Groups evaluate left to right and stop
at the first decisive operand, so an operand an earlier one made irrelevant is never read. A
correlation decides which authorized steps are the operation's semantic steps at all -- it never
supplies a trigger or a response, and the bounded countdown stays exactly the one the trigger and
response patterns describe. This step's occurrences are retained only once its whole append was
admitted, so no correlation reads the occurrence its own step creates. Admission rejects unretained
capture fields, unbound capture references, ordinals beyond a declared lifetime, empty groups,
exhausted correlation depth, unset ceilings, capture identities repeated anywhere in the capability,
and comparisons whose two operands do not share one declared scalar kind.

On the model side, `PropertyFieldPath.retainedFacts` gives a capture's coordinates their own
presence facts: a checked cursor reaches an `establish` or `select` step only over a payload that
supplied the optional value or selected the oneof member, so a retained occurrence's presence was
decided at the step that admitted it rather than in the Boolean branch that later reads it. The
same coordinates read as this step's own operand still require a fact established in their branch.

Evidence is written on both sides independently: `Testpilot.Tests.Fields` authors the portable
contract directly, the way a producer other than the Lean lowering would, and the Go tests author it
again in Go. The Go suite additionally compares live admission against offline replay at every chunk
boundary.

Known Gaps for task 8. `Umpire/Case/Scoped.lean` still rejects a capture-bearing scoped clause, and
`Umpire/Observation/Evaluation/Scoped.lean` still rejects one at the evidence adapter. Both now fail
for one reason, and it is not a missing portable capability: the portable contract names a retained
value by the declared evidence field id the projector emits, and a model `PropertyFieldPath`'s
structural coordinates have no declared evidence identity yet. Supplying one is the modeled-fields
to declared-Observations coverage map that task 8 owns. The evidence adapter has a second, narrower
blocker recorded for the same owner: `PropertyFieldEvidence` is reachable only through a real
`Field.Cursor` over an admitted `Umpire.Value` payload, and the projector supplies bare
`Shared.SemanticData` scalars with no schema, no `Raw` tree and no denotation.

Follow-up not built here (YAGNI): the scoped evidence scalar domain is still text/natural/boolean.
Bytes, signed/unsigned integers and enums are exact in the generic Contract IR and in the checked
`Umpire.Value` layer; widening `Shared.SemanticData.Scalar` would ripple through the projector,
`EvidenceFieldDeclaration` and both runtimes, and no qualifying example needs it yet.

stage: impl-review - ran [round 1 NEEDS_WORK (copilot/gpt-5.4) .. round 2 SHIP (copilot/gpt-5.4)]
## Evidence
- Commits: c5c0165c9c75b3ff6657d194d5aa1be6f5f6eeb1, 62f1e2f45fe99f49179300f7bca5e31c9abc6b60
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/... (all green), cd model && mise exec -- lake build Testpilot.Tests Umpire.Case.CompilerTests, cd model && mise exec -- lake build Testpilot.Tests.Fields, cd model && mise exec -- lake build Umpire.Property.Tests.Scoped.Fields UmpireTests, make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance (conformance fixtures byte-identical), make umpire-build-model, make protoc; make lint-protos; make lint-api, make lint-model (inherited red: the same 2 findings as base 2f91193c - unusedArguments on Umpire.instReprPropertyFieldProjection from task .5, simpNF on Umpire.Operation.CheckedRpc.mk.injEq from task .1; no third finding added), make lint-code GOLANGCI_LINT_FIX=false (inherited red: exactly 1,284 diagnostics, identical count to the recorded baseline; the 4 findings this task first introduced were fixed)
- PRs: