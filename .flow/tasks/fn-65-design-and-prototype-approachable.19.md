---
satisfies: [R2, R3, R4, R5, R7, R8]
---
# fn-65-design-and-prototype-approachable.19 Consolidate prototype regressions, walkthrough and fn-62 evidence mapping

## Description
Implements R2, R3, R4, R5, R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus2/**`, `model/TemporalModelTests.lean`, `model/UmpireTests.lean`, `model/lakefile.toml` (only test-root wiring if needed), `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md` (targeted semantic summary only), `.flow/specs/fn-65-design-and-prototype-approachable.md` (evidence links via flowctl)
**Touches:** [model/Temporal/Feature/Nexus2/**, model/TemporalModelTests.lean, model/UmpireTests.lean, model/lakefile.toml, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, .flow/specs/fn-65-design-and-prototype-approachable.md]

### Approach
Fold final documentation, aggregate test wiring and compatibility evidence into one task. Keep explicit Nexus2 test imports and the established Nexus inspector registrations unchanged. Summarize real results with implementation/test pointers; publish a per-R-ID fn62 evidence inventory for later residual replanning, without rewriting its deferred graph or claiming automatic supersession.

### Investigation targets
**Required** (read before coding):
- `model/TemporalModelTests.lean` — aggregate feature tests
- `model/UmpireTests.lean` — aggregate generic tests
- `model/lakefile.toml` — actual roots
- `model/Temporal/Feature/Nexus2/DESIGN.md` — required evaluation and scope
- `.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md` — R1–R9 inventory
- `Makefile` — lint/build/regression gates

### Quick commands
```bash
(cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Temporal.Feature.Nexus2.AuthoringTests UmpireTests TemporalModelTests TemporalExperimentalTests)
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Dedicated runnable Nexus2 semantic and AuthoringTests roots are in the appropriate aggregate gates. All fn65 R2–R5/R7/R8 positives and enumerated failures map to concrete executable tests; no Markdown-only example is counted as passing evidence. Preserve existing Nexus public imports, registrations, IDs, fingerprints and bytes except explicitly reviewed nonsemantic changes.
- [ ] Publish a concise checked learning path and measured constructor/frontend comparison covering finite extension, baseline/race, explicit outcomes/providers/units, raw versus checked admission, conditional pass versus trigger coverage, exceptions without replacement, conjunctive conflict meaning and bounded assurance. Update owning public facades/docs with the new vocabulary, operators and unsupported cases; no Observation/live-runtime claims.
- [ ] Record transitive axiom audits, warnings and performance observations, plus focused builds, aggregate UmpireTests/TemporalModelTests/TemporalExperimentalTests, make umpire-build-model, make lint-model, make umpire-check-regression and make lint-code GOLANGCI_LINT_FIX=false. Every red gate is attributed to an observed baseline or fixed; no hidden waiver, cache deletion or unrelated repair. Go tests always carry test_dep.
- [ ] Write a table for every fn62 R1–R9 with implementation/test/gate evidence, covered/partial/uncovered assessment and exact residual or contract mismatch. Explicitly assess Observation (R1/R6), authored Known Gaps (R7), established Nexus migrations (R5/R8), public docs (R9), source/family identity (R4), planner transport (R3), and proof-responsibility/no-wrapper differences (R2/R5). No requirement is covered by this plan or DESIGN alone.
- [ ] Keep fn62 deferred and its old seven-task graph non-executable until separate evidence-based residual replan/review. Record prototype rule exceptions and broader adoption/human-usability limits without claiming line-by-line human grammar approval or globally changing Umpire rules.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: