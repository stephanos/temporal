---
satisfies: [R7, R8]
---
# fn-65-design-and-prototype-approachable.11 Admit same-step guarded cases with conjunctive semantics and agreement

## Description
Implements R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/**`, `model/Umpire/Observation/{Check,Verdict}.lean`, `model/Umpire/Observation/Tests/{Check,Fixtures,Verdict}.lean`, `model/Umpire/Query/Language.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Case/Compiler.lean`, `model/Temporal/Feature/Nexus/**`, `model/Temporal/Feature/Nexus2/Tests.lean`, `model/Temporal/{ImplementationLinkTests,System/Nexus}/**`
**Touches:** [model/Umpire/Property/**, model/Umpire/Observation/Check.lean, model/Umpire/Observation/Verdict.lean, model/Umpire/Observation/Tests/Check.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Observation/Tests/Verdict.lean, model/Umpire/Query/Language.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Case/Compiler.lean, model/Temporal/Feature/Nexus/Operations/AsyncStartTests.lean, model/Temporal/Feature/Nexus/Operations/CancellationTests.lean, model/Temporal/Feature/Nexus/Operations/SuccessfulCompletionTests.lean, model/Temporal/Feature/Nexus2/Tests.lean, model/Temporal/ImplementationLinkTests/Nexus.lean, model/Temporal/System/Nexus/ImplementationLink.lean]

### Approach
Extend PropertyDeclaration/ResolvedPropertyClause, checking, evaluation and agreement together. Introduce explicit parent/case/exception/clause identity and same-step Boolean expectations with guarded temporal forms rejected until the next task. The Boolean kernel is already proved. Update exhaustive consumers to typed unsupported rejection in this task so no intermediate successful path drops a guard; the following compatibility task completes canonical/version inventory and rejection coverage.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean` — current public representation
- `model/Umpire/Property/Check.lean` — resolved types/canonical meaning
- `model/Umpire/Property/Evaluation.lean` — denotation, evaluator and agreement
- `model/Umpire/Observation/Verdict.lean` — consumer destructuring
- `model/Umpire/Case/Compiler.lean` — unsupported lowering boundary
- `model/Temporal/Feature/Nexus2/DESIGN.md` — parent/case/exception semantics

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests Umpire.Query.Tests Umpire.CaseTests Umpire.Case.CompilerTests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

This slice admits same-step guarded cases/group obligations only. Guarded temporal clauses remain explicitly unsupported until the next task; existing unguarded temporal clauses stay unchanged. First same-step admission includes safe version discrimination/canonical identity and affected-consumer typed rejection atomically.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [x] Named cases evaluate every applicable obligation conjunctively, including independent invariants. Effective applicability is parent guard minus parent exception, case guard minus case exception. Parent exclusions exclude all children; a sibling exceptional case explicitly supplies replacement behavior. No order/priority winner is selected.
- [x] Complete/exclusive flags are explicit checked obligations over effective case guards under the parent guard. Same-step tests cover request/special-resolution cases, compatible overlaps, missing replacement and excluded contexts separately. Static admission is not reachability or exhaustive coverage.
- [x] Same-step applicability uses only the triggering prior-state/selected-Action context. Test true/false parent and case exceptions, missing replacement and an independent invariant violated during an exception. Guarded temporal forms reject explicitly until the next task; legacy temporal semantics are unchanged.
- [x] Checker errors cover duplicate/malformed parent/case/clause/exception IDs, wrong/missing references, unsupported context/operators and invalid units/structures. Full clause identity includes case identity. Preserve typed kind and related IDs/source provenance; no partially checked declaration escapes.
- [x] Extend whole-clause/property evaluator-denotation agreement with no weaker hypotheses or native trust. Legacy declarations retain exact meanings/encodings; new semantics get explicit version discrimination. All affected exhaustive consumers compile and reject unsupported guarded forms before any success, including Observation and Case lowering. Case reorder preserves stable IDs/semantic fingerprints; guard changes change fingerprint.
- [x] Locate actual checked-Property-to-ContractLowering producers and test new-form rejection from those checked inputs. Compiler.lean consumes lowered monitor/unsupported data and is not itself a Property recognizer. Where no generic lowering exists, document that non-consumer boundary; never count hand-authored unsupported fixtures or add an unrelated universal compiler.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: