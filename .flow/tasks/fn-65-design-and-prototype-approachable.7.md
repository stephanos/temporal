---
satisfies: [R2, R5]
---
# fn-65-design-and-prototype-approachable.7 Derive FiniteMachine evidence and checked admission from validated tables

## Description
Implements R2, R5; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target/FiniteTable.lean` (new if needed), `model/Umpire/TargetTests.lean`
**Touches:** [model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target/FiniteTable.lean, model/Umpire/TargetTests.lean]

### Approach
Extend the existing FiniteMachine boundary (FiniteMachine.lean:12,43,92), with a total typed admission result over explicit catalogs and transition rows. Derive enumerators and mechanical evidence from the same rows; do not retain a second step function. Keep the adapter free of Query/Planning and syntax imports. Reuse AuthoredTarget/checkTarget; measure a generic kernel proof over admitted data before dependent feature work.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/FiniteMachine.lean` — authoritative adapter and proof fields
- `model/Umpire/Target/Language.lean` — typed admission and encodings
- `model/Umpire/TargetTests.lean` — invalid Target patterns
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — actual finite obligations
- `model/Temporal/Feature/Nexus2/DESIGN.md` — finite table and trust decisions

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.TargetTests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

The prior task owns typed catalog/table parsing and all malformed-table diagnostics. Consume that validated representation; this slice owns generic closure/executability proofs, existing FiniteMachine/AuthoredTarget/checkTarget integration and trust/extension measurements. Do not duplicate validator tests.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Reuse the validated table to derive its authoritative enumerators and all actual FiniteMachine membership/domain-closure/Action-executability witnesses with generic kernel-checked proofs. Package AuthoredTarget and call checkTarget; there is no second step function or alternate checked Target.
- [ ] Authors provide typed catalogs/keys, setups, complete alternatives and explicit provider selection, but no mechanical proofs or encoded-value assembly. Preserve actual capability-law obligations and existing failures for missing/conflicting providers; no dummy law or automatic success label.
- [ ] Failed validation/admission exposes no checked Target. The successful-branch interface makes raw construction and semantic admission observably distinct; preserve the expert TransitionKernel path.
- [ ] A focused extension fixture adds a state, catalog entry and row without support-code or feature-proof edits. Verify exact finite enumeration and ordering, including all nondeterministic alternatives, through the resulting checked Target.
- [ ] Audit all new transitive load-bearing proof dependencies and measure baseline/10x table admission cost. No sorry/admit/custom or compiler/native axioms; stop or retain a measured successful-branch route if efficient trusted admission is unavailable.

## Implementation evidence

The reproducible `linearTable`/`linearAdmission` fixture in
`model/Umpire/Target/Tests/FiniteMachine.lean` admits a three-state baseline and a ten-times-larger
table with 30 states, rows, and result alternatives. A forced interpreted run of 20 admissions via
`mise exec -- lake env lean ../.flow/tmp/fn65-task7-admission-measure.lean` measured 253 ms for the
3/3/3 table and 551 ms for the 30/30/30 table. This 2.18x observation supports retaining the total
successful-branch route for the prototype; it does not predict scaling for broader model shapes.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: