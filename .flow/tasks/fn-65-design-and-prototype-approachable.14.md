---
satisfies: [R7, R8]
---
# fn-65-design-and-prototype-approachable.14 Analyze finite guarded-case coverage with reachable evidence

## Description
Implements R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Planning/Engine.lean`, `model/Umpire/Planning/CaseAnalysis.lean` (new), `model/Umpire/Planning.lean`, `model/Umpire/Planning/Tests.lean`, `model/Temporal/Feature/Nexus2/Tests.lean`
**Touches:** [model/Umpire/Planning/Engine.lean, model/Umpire/Planning/CaseAnalysis.lean, model/Umpire/Planning.lean, model/Umpire/Planning/Tests.lean, model/Temporal/Feature/Nexus2/Tests.lean]

### Approach
Put bounded coverage behind existing Query/Property ownership, consuming selected checked Properties, Target, Behavior and typed Limits. Reuse finite planning enumeration/completeness, not a parallel trace search or authoring language. Separate case truth, applicability/exclusions, trigger exercise and search completeness. No CLI/reporting subsystem is needed; source-linked typed results plus checked fixtures suffice.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean` — selected Properties and completeness
- `model/Umpire/Planning/Engine.lean` — finite enumeration and statuses
- `model/Umpire/Property/Evaluation.lean` — effective guards and group obligations
- `model/Umpire/Query/Tests/Completeness.lean` — finite absence claims
- `model/Temporal/Feature/Nexus2/DESIGN.md` — evidence classes

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus2.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Executable analysis lives behind Umpire.Planning with Query-owned input types; do not export a Planning-dependent analyzer from Umpire.Query (that creates an import cycle).

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

Extract a narrow shared bounded traversal seam from Planning.Engine: its existing candidate pull/state/backend are private and public plan stops at first selection. Both ordinary planning and analysis must reuse that traversal and its completion/budget accounting. Do not duplicate search, expose representation-specific cursor state to feature authors, or mistake the first selected trace for exhaustive traversal.
## Acceptance
- [ ] Analyze only the explicit Target, Behavior, selected Properties and stage-specific bounds; every result carries this exact scope and completeness/Limit Reached status. No unbounded or general compatibility claim is possible.
- [ ] Demonstrate normal/special cases, compatible overlaps, reachable uncovered complete group and overlapping exclusive group. Witnessed failures include parent/case/clause IDs, sources, trigger state/Action or prefix and effective guards.
- [ ] Parent/case exclusions and missing replacement behavior are separately visible. No reachable parent trigger or an unexercised guard is not exercised coverage; absence requires exhaustive bounded evidence or remains inconclusive.
- [ ] Test unreachable trigger, rejected malformed checked inputs/Target mismatch, and exhausted analysis budget. Budget exhaustion never becomes complete coverage or no overlap; case reordering preserves results and never selects a winner.
- [ ] Completeness/exclusivity failures remain obligations, distinct from ordinary per-Property results and from model transition availability. Reuse authoritative evaluator applicability; add focused executable Nexus2 and generic tests.
- [ ] Shared bounded traversal is consumed by both ordinary planning and analysis with identical candidate order, admitted-Behavior filtering, budget accounting and completion evidence; unchanged Query regressions prove the extraction preserves existing first-result behavior. Task .15 reuses this seam for exhaustive continuation analysis.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: