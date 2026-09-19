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
Implemented trigger-frozen guarded bounded obligations directly and inside named cases, preserving parent/case/exception/clause provenance, original inclusive coordinates and resolved Limits. Extended checked admission, canonical forms, evaluator/denotation agreement, Planning support, and fail-closed Observation/Case boundaries without changing legacy meanings.

Review round 1 found that an unused default-empty temporal case field changed existing same-step canonical identities. The field is now omitted when empty, and an independently rebuilt task-start snapshot plus a checked-in golden regression freeze the exact legacy fingerprint.

Baseline was green at 87 jobs. Final exact Quick and lint-model gates are green; the repository Go lint remains the inherited 1316-diagnostic baseline with byte-identical normalized identities.
stage: impl-review - ran through 2026-09-05T21:06:22Z (codex:gpt-5.6-sol:medium; NEEDS_WORK round 1, SHIP round 2)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive).
## Evidence
- Commits:
- Tests: baseline: green ((cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests Umpire.Query.Tests Umpire.CaseTests Umpire.Case.CompilerTests), 87 jobs; /tmp/fn65-task12-baseline-quick-model.log), TDD red: guarded temporal admission/evaluation regressions failed before implementation as expected (/tmp/fn65-task12-tdd-red.log), (cd model && mise exec -- lake build Umpire.Property.Tests.GuardedCases Umpire.Property.Tests.GuardedTemporal) (green, 17 jobs; /tmp/fn65-task12-review-fix-focused.log), (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Observation.Tests Umpire.Query.Tests Umpire.CaseTests Umpire.Case.CompilerTests) (green, 88 jobs; /tmp/fn65-task12-review-fix-quick-model.log), make lint-model (green, 272 lint/import jobs plus 242 full-model jobs; /tmp/fn65-task12-review-fix-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited red: 1316 diagnostics; exact sorted individual identities match task .11, sha256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; /tmp/fn65-task12-final-lint-code.log), persisted task-start tree legacy fingerprint build (green; exact sha256:692aeccb2cfe68380842e8cddcf0e2f7b98c2827a42660e4ab8b1fcdb4eab1cf; /tmp/fn65-task12-prior-base-fingerprint.log), kernel trust audit (green; evaluatePropertyClause_agrees and evaluateProperty_agrees use only propext, Classical.choice, Quot.sound; /tmp/fn65-task12-trust-audit.log), git diff --cached --check (green), GATE_CLASSIFICATION:full (inherited unmatched .plans/UMPIRE4_ORDER.md), NO_RECEIPT:user-owned uncommitted staged tree
- PRs: