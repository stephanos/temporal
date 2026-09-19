---
satisfies: [R8]
---
# fn-65-design-and-prototype-approachable.15 Diagnose bounded joint Property conflicts without choosing a winner

## Description
Implements R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/Evaluation.lean`, `model/Umpire/Property/ImportTests.lean`, `model/Umpire/Planning/CaseAnalysis.lean`, `model/Umpire/Planning/VisibilityTests.lean`, `model/Umpire/Query/Tests.lean`, `model/Umpire/Query/Tests/JointConflicts.lean`, `model/Temporal/Feature/Nexus2/Tests.lean`
**Touches:** [model/Umpire/Property/Evaluation.lean, model/Umpire/Property/ImportTests.lean, model/Umpire/Planning/CaseAnalysis.lean, model/Umpire/Planning/VisibilityTests.lean, model/Umpire/Query/Tests.lean, model/Umpire/Query/Tests/JointConflicts.lean, model/Temporal/Feature/Nexus2/Tests.lean]

### Approach
Extend the coverage result seam with finite same-step expectation compatibility and modeled continuation analysis, reusing the checked predicate denotation and finite planner. Keep logical contradiction distinct from a violating trace and from bounded Target-relative incompatibility. Explicit selected obligations always remain conjoined; never narrow Behavior to conceal a conflict.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Evaluation.lean` — joint obligations
- `model/Umpire/Planning/Engine.lean` — exact admitted continuations and completeness
- `model/Umpire/Planning/Tests/Outcomes.lean` — outcome/Limit Reached conventions
- `model/Umpire/Query/Tests/Completeness.lean` — exhaustive evidence
- `model/Temporal/Feature/Nexus2/DESIGN.md` — conflict evidence requirements

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus2.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

Use the shared bounded traversal extracted by task .14, including its candidate order, budget accounting and completion evidence; public plan alone cannot enumerate all continuations.
## Acceptance
- [x] Reachable same-trigger mutually exclusive same-step expectations yield contradiction evidence naming the conflicting clauses, typed values/guards and common trigger. Compatible overlapping requirements are retained and pass together. Unsupported formula classes return explicit unsupported/inconclusive, never a broad logical verdict.
- [x] Joint bounded incompatibility requires a reachable common trigger with at least one admitted continuation and exhaustive evidence that none meets all selected obligations under exact Limits. Include the modeled prefix, continuation scope and responsible source-linked expectations.
- [x] Distinct tests demonstrate one Property violation without contradiction, logical contradiction, model-relative incompatibility without logical contradiction, no-continuation dead end, no-trigger/unexercised requirement and exhausted budget. A dead end is not conflict; a violating witness alone proves no mutual inconsistency.
- [x] Temporal obligations retain trigger coordinates/bounds during joint checking; later exceptions cannot remove them. Limits on search and trace horizon remain explicit; exhausted work is Limit Reached rather than exhaustive incompatibility.
- [x] Deterministic evidence and outcomes are invariant under case/property source reordering. No priority resolution, inferred transitions, live execution, general solver or unbounded claim is introduced.
## Done summary
Implemented finite joint Property analysis with source-linked logical contradiction evidence, exact realized temporal trigger scopes, and exhaustive bounded Target-relative incompatibility evidence. Outcomes keep violations, dead ends, static unsatisfiability, unexercised triggers, exhausted limits, and unsupported formula classes separate; generic and Nexus2 regressions cover multiway constraints, set-valued facts, ordering, frozen exceptions, multiple same-transition occurrences, and absent-trigger continuations.

The Codex review found one trigger-grouping defect; the resumed review confirmed the occurrence-identity fix and returned SHIP with R8 met. Tracker sync is inactive and plan sync is disabled for this spec.

stage: impl-review - ran [2026-09-05T22:44:42Z..2026-09-05T23:00:46Z] (model: gpt-5.6-sol at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: (cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus2.Tests) (green, 71 jobs; /tmp/fn65-task15-baseline-quick.log), finding-focused: (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Query.Tests.JointConflicts Umpire.Planning.VisibilityTests Umpire.Property.ImportTests) (green, 48 jobs; /tmp/fn65-task15-review-fix-focused2.log), (cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus2.Tests) (green, 72 jobs; /tmp/fn65-task15-review-fix-final-quick2.log), make lint-model (green, 245 jobs; /tmp/fn65-task15-review-fix-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited nonzero only: 1316 diagnostics, symmetric difference 0, normalized SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; /tmp/fn65-task15-review-fix-final-lint-code.log), impl-review codex:gpt-5.6-sol:medium SHIP round 2 (session 01a073be-deb4-7880-b2cc-97bc3491b145; /tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.15.json)
- PRs: