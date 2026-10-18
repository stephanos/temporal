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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: