---
satisfies: [R2, R3, R4]
---
# fn-65-design-and-prototype-approachable.8 Build the Nexus2 baseline and hide finite planner transport

## Description
Implements R2, R3, R4; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/CoreImportTests.lean`, `model/Umpire/Behavior/Language.lean`, `model/Umpire/Behavior/ImportTests.lean`, `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target/ImportTests.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Planning/Tests/Enumeration.lean`, `model/Umpire/Planning/VisibilityTests.lean`, `model/Temporal/Feature/Nexus2/Lifecycle.lean`, `model/Temporal/Feature/Nexus2/Cancellation.lean`, `model/Temporal/Feature/Nexus2/Tests.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/CoreImportTests.lean, model/Umpire/Behavior/Language.lean, model/Umpire/Behavior/ImportTests.lean, model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target/ImportTests.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Planning/Tests/Enumeration.lean, model/Umpire/Planning/VisibilityTests.lean, model/Temporal/Feature/Nexus2/Lifecycle.lean, model/Temporal/Feature/Nexus2/Cancellation.lean, model/Temporal/Feature/Nexus2/Tests.lean]

### Approach
Reuse Engine.lean:149 ofCheckedQuery? and FiniteKernelOrder rather than feature-specific equality transport. Add a narrow Target-owned typed-catalog-to-ModelValue admission adapter because QueryTarget is fixed to ModelValue/List RoleBinding; the adapter consumes typed setup-role bindings and namespace/field identities, uses validated stable catalog keys, and preserves the existing Target checker and capability-law obligations. Add Planning-owned finite order validation and typed missing-completeness/Target-mismatch failures while preserving the existing `ofCheckedQuery?` compatibility API. Author all baseline rows from DESIGN.md and start/cancel/success Property-Behavior-Query declarations through existing checked owners. Keep Nexus2 identities explicit and independent from Nexus.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Engine.lean` — finite adapter
- `model/Temporal/Feature/Nexus/Operations/Planning.lean` — transport to eliminate
- `model/Temporal/Feature/Nexus/Lifecycle/Semantics.lean` — exact baseline and capability law
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — setups and domain order
- `model/Temporal/Feature/Nexus/Operations/Cancellation.lean` — checked declaration journey
- `model/Temporal/Feature/Nexus2/DESIGN.md` — baseline identity mapping

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Post-edit focused gate (mandatory after creating the root): `(cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests)`.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

### Implementation evidence

- The typed Nexus2 table declares 4 states, 3 Actions, 2 setups, and 3 transition rows. `Temporal.Feature.Nexus2.Tests` normalizes all 12 state/Action pairs from both admitted Targets and proves the same exact three transitions, then compares all three exact bounded planner traces through the explicit identity mapping.
- The three independently authored Property/Behavior/Query journeys each report `found` with one-semantic-transition, one-selected-Action, eight-candidate Limits and explicit shortest/seed-17/Definition-ID policy.
- Direct dedicated-root elaboration, measured without deleting Lake caches, took 3.07s, 3.06s, and 3.06s over three runs on this checkout. Planning caches the invariant sorted Action catalog once per admitted kernel; setup and step sorting remains pull-local because generic finite completeness does not constrain arbitrary out-of-domain selectors, so transporting the established `ofFinite` forall ordering law would be unsound.
- The dedicated comparison fails closed over every admitted domain value, every setup-to-initial-state alternative, and every emitted result; mutation fixtures reject unknown result states, unknown facts, an extra admitted transition, and an extra admitted initial alternative. Both identity roots must reproduce the exact ordered setup-to-initial relation. `native_decide` is used only to execute closed test fixtures, and no such theorem is imported by or feeds Target, provider-law, declaration, or planner admission. Public adapter/checkBaseline dependencies remain the standard `propext`, `Classical.choice`, and `Quot.sound`; `lifecycleLawProof` has no axioms.
- Final `make lint-model` passed. `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2 with exactly 1,316 diagnostics before and after this Lean-only task: errcheck 220, exhaustive 6, forbidigo 209, goimports 1, govet 5, revive 735, staticcheck 139, testifylint 1 (+0/-0).
## Acceptance
- [ ] Nexus2 reproduces the four states, three transitions and two setups under an explicit state/Action/outcome/fact identity mapping. Compare complete admitted transitions and exact bounded traces against established Nexus; different Nexus2 IDs/source paths do not require equal artifact bytes. Established Nexus imports and registrations remain unchanged.
- [ ] Start, cancellation and successful-completion declarations compile, admit and execute their bounded Queries with explicit providers, Query forms, stage/unit Limits, strategy, seed and tie-breaking. Requirements are independently authored rather than inferred from rows.
- [ ] An already checked finite Target/Query obtains the planner kernel without feature equality transport, representation unfolding or cleanup proofs. Missing finite completeness and mismatched Target identity are explicit failures, with no inferred partial kernel.
- [ ] Test invalid references, missing capabilities/providers, unresolved competing providers, invalid/omitted units or Limits, contradictory Behavior constraints and unsatisfiable scenarios at their responsible checker/planner status. Raw construction alone is not success evidence.
- [ ] Add runnable Nexus2 Tests and preserve all existing baseline regressions and trust boundaries; ordinary author examples contain no encoded-value assembly, proof editing or support-code changes for a state/transition extension.

## Done summary
Implemented the independent Nexus2 baseline as typed finite catalogs with 4 states, 3 actions, 2 setups, and 3 transitions; explicit stable identity/setup mappings lower validated values through the Target owner. Start, cancel, and success each follow the successful Target → Property → Behavior → Query → generic finite planner journey with explicit forms, unit-bearing limits, shortest strategy, seed 17, and DefinitionId tie-breaking.

Target now owns validated typed-to-ModelValue admission, including setup membership checks, while Planning owns typed Target-mismatch, missing-completeness, and noncanonical-order failures. The compatibility `ofCheckedQuery?` API remains available, and the separate canonical finite adapter preserves total ordering laws without author equality transport. Invariant action sorting is hoisted; setup and step ordering remain pull-local because finite completeness does not constrain arbitrary out-of-domain selectors.

The dedicated `Temporal.Feature.Nexus2.Tests` gate compares the exact complete ordered catalogs, setup-to-initial-state relation, 12 state/action transition space, and three bounded traces against established Nexus through explicit fail-closed identity normalization. Mutation fixtures reject unknown result states, unknown facts, extra transitions, extra initial alternatives, and changed cancellation/success semantics. Checker-specific negatives cover malformed catalogs, undeclared setups, provider selection, Property references/capabilities, contradictory Behavior, Query limit values/units, omitted units/limits, unsatisfiable planner status, Target mismatch, missing completeness, and noncanonical action/result order.

The capability law states the three required source/action/result meanings independently through pair-specific row lookup; unrelated extensions do not require changing the law. `#print axioms` reports only `propext`, `Classical.choice`, and `Quot.sound` for public admission/planning/checkBaseline declarations, while `lifecycleLawProof` has no axioms. `native_decide` executes closed test fixtures only and no such theorem feeds production admission, provider laws, declarations, or checked values.

Dedicated-root direct elaboration measured 3.07s, 3.06s, and 3.06s without deleting caches. Final focused Nexus2 build and `make lint-model` passed. `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2 with exactly 1,316 diagnostics before and after this Lean-only task: errcheck 220, exhaustive 6, forbidigo 209, goimports 1, govet 5, revive 735, staticcheck 139, testifylint 1 (+0/-0).

Task 9 can consume `Temporal.Feature.Nexus2.Lifecycle.table`, `identity`, `targetResult`, and `Temporal.Feature.Nexus2.Cancellation.checkBaseline`; race behavior should extend typed rows/declarations and retain the same exact fail-closed comparison style. Existing Nexus imports and registrations were not migrated.

No commits, push, worktree, reset, revert, or cache deletion; the user retains commit ownership and prior staged work is preserved.

stage: impl-review - ran [NEEDS_WORK..SHIP] (model: codex:gpt-5.6-sol:medium; session: 01a072b8-dab1-7663-91d1-a495c0b415d4)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)

Review receipt: `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.8.json`
Review log: `/tmp/fn65-task8-review-resume.log`
Reviewed base/staged tree: `2d4ab82eaffa1f87d402b9a2b4f65230d7fe9c77..7de76f50a364f81bbe62c70b7d41ecdd86339343`
## Evidence
- Commits:
- Tests: baseline: (cd model && mise exec -- lake build Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests) (pass; 65 jobs), baseline: make lint-model (pass; 239 jobs), baseline: make lint-code GOLANGCI_LINT_FIX=false (inherited red: 1316 diagnostics), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Planning.Tests) (pass; 61 jobs), (cd model && mise exec -- lake build Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus2.Tests) (pass; 69 jobs), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests) (post-review-fix pass; 40 jobs), make lint-model (post-review-fix pass; 239 jobs), make lint-code GOLANGCI_LINT_FIX=false (inherited red: 1316 diagnostics; zero added; zero removed), direct lake env lean Temporal.Feature.Nexus2.Tests elaboration: 3.07s, 3.06s, 3.06s without cache deletion, #print axioms FiniteTable.checkModelTarget, IncrementalPlannerKernel.ofCheckedQuery, and Cancellation.checkBaseline: propext, Classical.choice, Quot.sound only; Lifecycle.lifecycleLawProof: no axioms, impl-review codex:gpt-5.6-sol:medium: SHIP
- PRs: