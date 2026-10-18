---
satisfies: [R7, R8]
---
# fn-65-design-and-prototype-approachable.12 Preserve trigger-time exceptions across bounded temporal obligations

## Description
Implements R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/**`, `model/Umpire/Observation/Check.lean`, `model/Umpire/Observation/Verdict.lean`, `model/Umpire/Observation/Tests/**`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Case/Compiler.lean`, `model/Umpire/Case/CompilerTests.lean`, `model/Temporal/Feature/Nexus2/Tests.lean`
**Touches:** [model/Umpire/Property/**, model/Umpire/Observation/Check.lean, model/Umpire/Observation/Verdict.lean, model/Umpire/Observation/Tests/**, model/Umpire/Planning/Engine.lean, model/Umpire/Case/Compiler.lean, model/Umpire/Case/CompilerTests.lean, model/Temporal/Feature/Nexus2/Tests.lean]

### Approach
Extend PropertyDeclaration/ResolvedPropertyClause, checking, evaluation and agreement together. Introduce explicit parent/case/exception/clause identity and same-step Boolean expectations plus guarded legacy single-pattern bounded obligations. The Boolean kernel is already proved. Update exhaustive consumers to typed unsupported rejection in this task so no intermediate successful path drops a guard; the following compatibility task completes canonical/version inventory and rejection coverage.

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


Consume admitted same-step cases and identity/context machinery. This slice adds guarded existing single-pattern temporal responses with original inclusive semantics, plus temporal agreement and safe consumer/version handling. It does not redesign the same-step validator or add compound temporal response operators.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.
## Acceptance
- [x] Guarded existing single-pattern bounded obligations evaluate all applicability/exception conditions once at the original prior-state/selected-Action trigger. Capture the original coordinate, resolved Limit and unit; all applicable clauses remain conjoined with independent invariants.
- [x] Positive/negative tests show true/false exceptions at trigger, later exception/state change cannot withdraw a pending obligation, inclusive response at trigger or within the bound, missing response and preserved unit/Limit errors. Parent exclusion is not a replacement obligation and source order never resolves conflict.
- [x] Extend bounded-clause and whole-Property evaluator/denotation agreement with kernel proofs and unchanged legacy meanings/theorem contracts. Audit transitive axioms; no native proof fallback or weakened hypotheses.
- [x] Reject future/result-state guard reads, dynamic/until exceptions, unknown input through negation and compound temporal responses. New temporal forms receive canonical/version discrimination and typed unsupported handling at actual downstream lowering/evaluation boundaries from their first admission; checked-input tests prove no condition is dropped.
- [x] Preserve parent/case/exception/clause IDs and source diagnostics in temporal failures. Wire focused regressions immediately into Umpire.Property.Tests and affected consumer test roots.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: