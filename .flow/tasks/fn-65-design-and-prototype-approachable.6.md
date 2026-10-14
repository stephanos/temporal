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
Implemented ordered typed catalogs, setup and transition rows, stable key encoding, and total typed structural validation through the Umpire.Target facade. Successful validation retains exact data plus kernel-checked catalog/closure/executability facts; it does not produce a FiniteMachine or checked Target, and leaves provider/capability-law validation with Target.

Catalogs define modeled domains, not every Lean carrier inhabitant (task acceptance clarified with conductor agreement). Missing references fail; isolated states and executable rows unreachable from setups are permitted. Encoding collisions are exactly duplicate stable keys. The 25-case negative fixture covers typed malformed/duplicate keys, duplicate values/pairs/setups, missing setup rows, empty alternatives, unexecutable Actions, and all domain references. Order/encoding/empty-domain/proof-projection fixtures are imported by Umpire.TargetTests.

Baseline: focused model build and make lint-model green; make lint-code GOLANGCI_LINT_FIX=false red pre-edit with 1316 inherited diagnostics. Logs: /tmp/fn65-task6-baseline-{build,lint-model,lint-code}.log. No Go file is owned or changed by this task.

Trust: #print axioms Umpire.FiniteTable.validate reports only propext, Classical.choice, Quot.sound; Umpire.FiniteCatalog.encode? is axiom-free. All new test proofs use kernel decide/direct terms; no native/custom axioms or placeholders. Full-carrier enumeration, semantic admission, planner adaptation and performance measurements remain task7+ work.

No commits, push, reset, reverts or worktrees; user commits externally. Prior staged changes preserved. Task base HEAD/tree are recorded in evidence and .flow/tmp/fn65-task6-review-snapshot.json.

stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)

Verification: Umpire.TargetTests and make lint-model passed; final Go lint remained red with exactly 1316 identical diagnostics (zero additions/removals; /tmp/fn65-task6-lint-comparison.json). Flow validation passed for 19 tasks. Final logs: /tmp/fn65-task6-final-{build,lint-model,lint-code}.log. Generic gate classification was FULL due inherited paths, and the actual task gates were run without skips.

stage: impl-review - ran (SHIP first pass; codex:gpt-6-astra:medium; started before user's routing update to gpt-5.6-sol medium)
Review receipt: /tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.6.json
Reviewed base/staged tree: c5feadd86e21ea22345b580858d502a4994c1a44..3759030ec7d92d409282cce2a7fdff6e30154f6f
## Evidence
- Commits:
- Tests: baseline: green model build and lint-model; red make lint-code GOLANGCI_LINT_FIX=false (1316 inherited diagnostics), (cd model && SDKROOT="$(xcrun --show-sdk-path)" mise exec -- lake build Umpire.TargetTests) — passed; includes new Umpire.Target.Tests.FiniteTable, make lint-model — passed, make lint-code GOLANGCI_LINT_FIX=false — inherited red: 1316 diagnostics, zero additions/removals versus pre-edit baseline, flowctl validate --spec fn-65-design-and-prototype-approachable --json — valid, 19 tasks, no warnings, git diff --cached --check — passed, #print axioms Umpire.FiniteTable.validate — standard propext/Classical.choice/Quot.sound only; encode? axiom-free
- PRs: