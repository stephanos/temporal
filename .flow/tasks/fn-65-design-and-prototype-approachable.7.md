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
Derived `FiniteMachine` enumerators, catalog encoders, row-backed initial/step functions, closure witnesses, and Action-executability evidence entirely from `ValidatedFiniteTable`. Added `FiniteTargetDefinition`, the layered `FiniteTargetAdmissionError`, and total `FiniteTable.checkTarget`, which validates raw data, packages the existing `AuthoredTarget`, and invokes the existing semantic `Umpire.checkTarget`; it never calls the native-default `checkedTarget` or defines an alternate step function.

The intended consumer pattern is `FiniteTable.checkTarget table definition composition`, followed by matching its successful branch to use the sole `CheckedTarget`. `composition` remains explicit: provider selections and their real law witnesses are authored values, and the adapter preserves exact `missingProvider` and `conflictingProviders` Target diagnostics rather than manufacturing a capability label or proof.

The focused extension fixture adds only one state catalog entry and one transition row. Its resulting checked Target test covers every typed domain in authored order, initial states, enabled and disabled pairs, exact `[completeResult, retryResult]` alternative order, planning Action order, and the complete canonical encoded behavior view. Structural rejection and semantic rejection each expose no checked value.

Trust audit: `#print axioms` for `ValidatedFiniteTable.machine`, `.authoredTarget`, and `FiniteTable.checkTarget` reports only `propext`, `Classical.choice`, and `Quot.sound`. There is no `sorry`, `admit`, custom/compiler/native axiom, or native checked-value extraction in load-bearing declarations.

Admission measurement: the reproducible `linearTable`/`linearAdmission` fixture admits both sizes. A forced interpreted run of 20 admissions measured 253 ms for 3 states / 3 rows / 3 result alternatives and 551 ms for 30 / 30 / 30, about 2.18x for this 10x workload. This supports retaining the successful-branch prototype route; it does not predict broader-model scaling. Runner log: `/tmp/fn65-task7-admission-measure.log`.

Baseline and final focused `Umpire.TargetTests` builds passed, including the public `Umpire.Target.ImportTests`; final `make lint-model` passed. `make lint-code GOLANGCI_LINT_FIX=false` remained at its inherited red baseline of exactly 1316 diagnostics, with zero additions and zero removals and no Go file owned by this task. Comparison: `/tmp/fn65-task7-lint-comparison.json`; gate logs: `/tmp/fn65-task7-{baseline,final}-*.log` and `/tmp/fn65-task7-review-fix-*.log`.

No commits, push, worktree, reset, revert, or cache deletion; the user retains commit ownership and prior staged work is preserved.

stage: impl-review - ran [NEEDS_WORK..SHIP] (model: codex:gpt-5.6-sol:medium)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)

Review receipt: `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.7.json`
Reviewed base/staged tree: `7d9040ac4c44769b665ffed88829de8bf3c177d5..f588e6d691e449943a6e901511601e5ae340ca2b`
## Evidence
- Commits:
- Tests: baseline: (cd model && mise exec -- lake build Umpire.TargetTests) (pass), baseline: make lint-model (pass), baseline: make lint-code GOLANGCI_LINT_FIX=false (inherited red: 1316 diagnostics), (cd model && mise exec -- lake build Umpire.TargetTests) (pass), (cd model && mise exec -- lake build Umpire.Target.ImportTests) (pass), make lint-model (pass), make lint-code GOLANGCI_LINT_FIX=false (inherited red: 1316 diagnostics; zero added; zero removed), forced interpreted FiniteTable.checkTarget benchmark: 20x 3 states/rows/alternatives = 253 ms; 20x 30 states/rows/alternatives = 551 ms, #print axioms ValidatedFiniteTable.machine/authoredTarget and FiniteTable.checkTarget: propext, Classical.choice, Quot.sound only, impl-review codex:gpt-5.6-sol:medium: SHIP
- PRs: