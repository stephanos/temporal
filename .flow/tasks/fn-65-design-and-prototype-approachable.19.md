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
Consolidated the Nexus2 prototype into the established aggregate test gates, published its checked learning path and executable fn-65 evidence inventory, and made the owner-local Query limit vocabulary consistent. Every fn-65 R2-R5/R7/R8 positive and enumerated failure maps to runnable Lean evidence; the established Nexus identities, registrations, fingerprints, bytes, and planner behavior remain unchanged.

The exact fn-62 R1-R9 comparison lives at `model/Temporal/Feature/Nexus2/EVIDENCE.md`: R3 is covered; R1/R2/R4/R5/R8/R9 are partial with explicit contract mismatches; R6/R7 are uncovered. In particular, the prototype adds no Observation authoring, authored Known Gaps, checked-Property-to-Observation compiler, established Nexus migration, or automatic kernel-checked frontend constants. fn-62 and its existing seven deferred tasks remain unchanged and non-executable pending the conductor's separate completion review and residual replan.

Final frozen verification: aggregate Lake build exit 0 (251 jobs), `make umpire-build-model` exit 0 (325 jobs), `make umpire-check-regression` exit 0, and `make lint-model` exit 0 (257 jobs). `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2: its 1,316 sorted diagnostic headers exactly equal task 18 with SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077` and an empty diff. The full logs and command evidence are recorded in the evidence inventory; the spec validator passed with 19 tasks and no errors or warnings.

Official review verdict: SHIP with zero findings. Receipt `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.19.json`, session `01a07471-2e01-7ec3-863c-efc727d37b38`, reviewed tree `686f5f36eab74106a555c0e96f13e2a24ddd9933`, receipt SHA-256 `84a9291e0842a2a9dc76f26c46816fcd6dfd1168199f4f1dee21d735ed39e5ff`. The reviewed tree exactly matched the full staged tree, and HEAD remained `7774fdc7ac751ac959816c9829516ce54af57194`.

No commit was created because the user retains commit ownership. Tracker sync is inactive.

stage: impl-review - ran [2026-09-06T01:59:27Z..2026-09-06T02:01:01Z] (model: gpt-5.6-sol at medium; verdict: SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Temporal.Feature.Nexus2.AuthoringTests UmpireTests TemporalModelTests TemporalExperimentalTests) (exit 0; 251 jobs; /tmp/fn65-task19-final3-aggregate.log), TMPDIR=/private/tmp/taskdir make umpire-build-model (exit 0; 325 jobs; /tmp/fn65-task19-final3-umpire-build-model.log), TMPDIR=/private/tmp/taskdir make umpire-check-regression (exit 0; /tmp/fn65-task19-final3-umpire-check-regression.log), TMPDIR=/private/tmp/taskdir make lint-model (exit 0; 257 jobs; /tmp/fn65-task19-final3-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; 1316 sorted diagnostic headers exactly match task 18; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; diff 0; /tmp/fn65-task19-final3-lint-code.log), flowctl validate --spec fn-65-design-and-prototype-approachable --json (exit 0; 19 tasks; no errors or warnings; /tmp/fn65-task19-final-flow-validate.json), git diff --check (exit 0 for task-owned unstaged diff before final staging), official impl-review (SHIP; zero findings; receipt /tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.19.json; session 01a07471-2e01-7ec3-863c-efc727d37b38; reviewed tree 686f5f36eab74106a555c0e96f13e2a24ddd9933)
- PRs: