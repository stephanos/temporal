---
satisfies: [R2, R5, R7]
---
# fn-21-nexus-duplicate-observation-control.7 Compile the fault-specific observation mapping

## Description
Compile the exact fault-specific Temporal evidence profile/program/mapping before RuntimeConfiguration composition for R2/R5/R7. Reuse the baseline caller-closure declarations and unchanged reusable Observation checker/evaluator.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Observation.lean`, `model/Temporal/Feature/Nexus/ObservationFaultTests.lean`, `model/Temporal/Feature/Nexus/ObservationTests.lean`, `model/Temporal/Feature/Nexus.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationFaultTests.lean, model/Temporal/Feature/Nexus/ObservationTests.lean, model/Temporal/Feature/Nexus.lean, model/TemporalModelTests.lean]

### Approach
- Compose a second checked Temporal mapping/profile identity from fn-20's baseline caller-closure declarations and Task `.1`'s exact fault identity; do not copy the generic mapper or Property evaluator.
- Freeze the fault-specific source schema before runtime binding: one requested/completed cancellation lifecycle, mechanical callback count one, synthetic-contribution count one, one injected marker, exact receipt/correlation/order/closure fields, and their dispositions.
- Derive semantic cancellation count two only from the exact callback-one plus contribution-one relation; retain delivery/ownership rules unchanged and preserve complete coordinate derivations.
- Expose the checked program/mapping semantic references and digests consumed by Task `.2`; every reorder-equivalent declaration remains identical while each meaning-bearing mutation changes identity or fails checking.
- Prove missing/duplicate/wrong-kind fields, stale fault/capability identities, impossible count relation, cleartext-from-redacted copying, and unauthorized semantic declarations fail mapping compilation.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-20-local-execution-semantic-conformance.2.md:13-35` — baseline checked Nexus mapping/checker seam
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.4.md` — Temporal-owned profile/program ownership
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:99-103` — derivation/disposition wire contract
- `model/Temporal/Feature/Nexus/CallerClosure.lean:441-462` — unchanged semantic outputs and Property clauses
- `.flow/specs/fn-21-nexus-duplicate-observation-control.md` — exact fault evidence and mutation table

### Acceptance
- [ ] One deterministic checked profile/program/mapping variant exists before Task `.2` and exposes exact semantic references/digests.
- [ ] Mechanical callback count remains one; only the checked mapping derives semantic count two from one labeled synthetic contribution.
- [ ] Baseline mapping and pure Property/evaluator definitions remain unchanged.
- [ ] Every invalid schema/reference/disposition/count-relation mutation fails compilation at its owning checked boundary.
- [ ] Focused and aggregate Lean tests pass with no Temporal vocabulary entering reusable Umpire modules.

## Acceptance
- [ ] R2 receives final checked mapping/profile references before RuntimeConfiguration binding.
- [ ] R5 exact semantic derivation contract is compiled independently of live execution.
- [ ] R7 single-authority, package-purity, and no-second-mapper boundaries hold.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
