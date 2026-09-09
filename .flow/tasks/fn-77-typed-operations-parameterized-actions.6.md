---
satisfies: [R3, R5, R6, R9]
---
# fn-77-typed-operations-parameterized-actions.6 Compose keyed field captures with scoped temporal obligations

## Description
Compose keyed field captures with scoped temporal obligations for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Property/Scoped/**; model/Umpire/Property/Language.lean; model/Umpire/Property/Check.lean; model/Umpire/Property/Evaluation.lean; model/Umpire/Observation/Evaluation/Scoped.lean; model/Shared/ScopedObligation.lean
**Touches:** [model/Umpire/Property/Scoped/**, model/Umpire/Property/Language.lean, model/Umpire/Property/Check.lean, model/Umpire/Property/Evaluation.lean, model/Umpire/Observation/Evaluation/Scoped.lean, model/Shared/ScopedObligation.lean]

### Approach
- Add named prior-occurrence capture selection with explicit operation key, checked type and lifetime. Require deterministic exact occurrence identity/ordinal; reject ambiguity rather than implicit latest-match replacement.
- Retain immutable per-trigger captured values for repeated obligations and consume only admitted operation-local semantic steps. Reuse fn78 inclusive clock/endpoint and evidence admission, extending its correspondence for task5 operands.
- Charge capture values/keys/work under declared bounds; failed capture/admission publishes no partial state and cannot repair proved violations.
- Test two interleaved operations, repeated triggers, missing/future/unbound/wrong-key captures, partial evidence/chunking and close modes; prove incremental and complete execution correspondence.

### Investigation targets
**Required:**
- model/Umpire/Property/Scoped/Kernel.lean — checked scoped runtime.
- model/Umpire/Property/Scoped/Reference.lean — existing Property correspondence.
- model/Umpire/Observation/Evaluation/Scoped.lean — admitted evidence adapter.
- model/Shared/ScopedObligation.lean — inclusive independent obligation semantics.

### Quick commands
`cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.Tests.Scoped.Evidence Umpire.Case.CompilerTests`

`cd model && mise exec -- lake build Umpire.Property.Tests.Scoped.Fields`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Property.Tests.Scoped.Fields into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Keyed captures bind only declared earlier occurrences and reject future/unbound/ambiguous/wrong-operation access.
- [ ] Repeated triggers retain independent immutable captured values; matching responses use original fn78 deadlines and closure semantics.
- [ ] Capture/resource failures are atomic and bounded; cross-operation and chunk-boundary fixtures preserve support and inconclusive/violation distinctions.
- [ ] Extended checked correspondence composes with existing scoped proofs; old scoped fixtures and trust remain unchanged.

## Done summary
Keyed field captures now compose with the existing scoped bounded-response obligations without
touching the countdown. `PropertyFieldPath` gained an optional `PropertyFieldCaptureKey`, so a
capture operand names an exact earlier occurrence and resolves through the same operand path
task 5 built; `PropertyScopedClause` gained default-empty `captures` declarations and an optional
guard-context `correlation`, both canonicalized only when declared so existing scoped Properties
keep their exact metadata and behavior fingerprint. `Umpire.Property.Scoped.Captures` is the
append-only operation-local store, and its `record_extends` theorem proves a recorded ordinal is
never rewritten -- a latest-match implementation does not compile. `Run.consume` gained same-step
evidence: it gates admission on the correlation over that evidence plus the operation's retained
captures, retains this step's occurrences only after the whole append was admitted, and charges
retained values against a new `Limits.captures` budget. Future, unbound, ambiguous, wrong-key,
wrong-coordinate and foreign-operation access all reject; a rejected append publishes no state and
cannot repair a proved violation. `Run.consumeEvidence`/`consumeEvidence_append` carry chunk
correspondence, and the evidence adapter and portable Case lowering reject keyed captures rather
than silently dropping a correlation they cannot carry yet.

Follow-up for task 7: portable/offline capture evaluation is the reason
`Umpire/Observation/Evaluation/Scoped.lean` and `Umpire/Case/Scoped.lean` currently reject
capture-bearing clauses. A capture's retained coordinates are also validated with no established
presence facts, so an optional or oneof-selected field cannot yet be captured.

stage: impl-review - ran [round 1 NEEDS_WORK (copilot/gpt-5.4) .. round 2 SHIP (copilot/gpt-5.4)]
## Evidence
- Commits: d5c3328267a1937daabe7f4209c152a852c69a51, fc9f6a27f3c1d29d8976d6e5c0b4fbbf5983700c
- Tests: cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Property.Tests.Scoped.Evidence Umpire.Case.CompilerTests, cd model && mise exec -- lake build Umpire.Property.Tests.Scoped.Fields, cd model && mise exec -- lake build Umpire.Property.ImportTests Umpire.Query.Tests, mise exec -- make umpire-build-model, make lint-model (inherited red: 1 pre-existing unusedArguments diagnostic on Umpire.instReprPropertyFieldProjection, identical count verified at base 28189ca5), make lint-code not run: no Go changed; tracked red fn-2-agentworkflow-configuration-and-cli.6
- PRs: