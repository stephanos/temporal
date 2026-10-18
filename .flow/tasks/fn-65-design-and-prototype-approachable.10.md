---
satisfies: [R7]
---
# fn-65-design-and-prototype-approachable.10 Define the typed Boolean Property predicate kernel and agreement

## Description
Implements R7; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/Language.lean`, `model/Umpire/Property/Check.lean`, `model/Umpire/Property/Evaluation.lean`, `model/Umpire/Property/Tests/**`
**Touches:** [model/Umpire/Property/Language.lean, model/Umpire/Property/Check.lean, model/Umpire/Property/Evaluation.lean, model/Umpire/Property/Tests/**]

### Approach
Add the smallest typed portable Boolean vocabulary behind Umpire.Property: atomic typed patterns, nonempty all/any and not. Bind guards to one prior-state/selected-Action context, and same-step expectations to one resulting-state/outcome/fact context. This foundational task does not yet admit public guarded clause forms; the next task installs cases atomically with consumer rejection. Extend the existing atomic agreement at Evaluation.lean:29,50.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean` — field/pattern/clause types
- `model/Umpire/Property/Check.lean` — reference and capability validation
- `model/Umpire/Property/Evaluation.lean` — Boolean/denotational agreement
- `model/Umpire/Property/Trace.lean` — complete trigger context
- `model/Umpire/Property/Tests/Validation.lean` — typed errors
- `model/Temporal/Feature/Nexus2/DESIGN.md` — exact operator and context limits

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Property.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Typed atoms and all/any/not are serializable pure data; literal one-of is same-field disjunction, not conjoined equalities. Guard and expectation contexts are explicit and cannot accidentally join different steps.
- [ ] Checker and context-validation negatives cover missing and wrong-kind references, missing capabilities, empty Boolean groups, type/payload mismatch, unsupported comparisons/cross-field equality, arbitrary callbacks/predicates, resulting/future-state guards and compound temporal responses.
- [ ] Missing/unsupported inputs produce a diagnostic before negation; unknown cannot become truth, false applicability or a successful exception. Test nested negation/disjunction against this boundary.
- [ ] Prove evaluator/denotation agreement for validated typed predicates with generic kernel-checked proofs. Preserve existing single-pattern evaluation and encodings; do not weaken existing theorem statements.
- [ ] Add focused positive/negative and same-context tests, public semantic docstrings and axiom audits; no new admitted external clause exists until the case-integration task.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: