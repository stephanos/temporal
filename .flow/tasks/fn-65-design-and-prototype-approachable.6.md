---
satisfies: [R2, R5]
---
# fn-65-design-and-prototype-approachable.6 Validate typed finite catalogs and transition tables

## Description
Implements R2, R5; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target/FiniteTable.lean` (new if needed), `model/Umpire/TargetTests.lean`
**Touches:** [model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target/FiniteTable.lean, model/Umpire/TargetTests.lean]

### Approach
Extend the existing FiniteMachine boundary (FiniteMachine.lean:12,43,92), with a total typed admission result over explicit catalogs and transition rows. Validate one authoritative finite table; the following task derives enumerators and proof evidence from it. Keep the adapter free of Query/Planning and syntax imports. Define the validated input seam for the following AuthoredTarget/checkTarget proof adapter.

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

This first slice owns only typed finite data, encoding and validation. It publishes validated inert table data, not a FiniteMachine or checked Target. The following task supplies generic kernel evidence and checked admission.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Define explicit typed catalogs, stable keys, setups, source/Action rows and complete state/outcome/fact alternatives under Umpire.Target. Validation returns typed success/error and retains explicit catalog order; no derived domain from rows and no second step function.
- [ ] Table-driven negatives reject referenced values missing from declared catalogs, malformed/duplicate keys, colliding encodings, duplicate source/action rows, empty alternatives, declared Actions with no row and out-of-domain setups/result states/outcomes/facts. Isolated states are allowed; row executability is not a reachability claim.
- [ ] Validated table data exposes the closure/executability facts needed by the next proof adapter without author-written ModelValue assembly. Row order cannot choose an alternative/provider or resolve a duplicate; provider/capability-law validation remains the existing Target responsibility.
- [ ] Add focused validation/order tests imported by Umpire.TargetTests and public data/error documentation. This slice does not claim semantic checked admission merely because data elaborates, and introduces no new axiom/native proof trust.

Catalogs define the modeled domains, not every inhabitant of their Lean carrier types. Unused carrier values need not be modeled; no second universe or inferred domain is required.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: