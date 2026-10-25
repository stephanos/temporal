---
satisfies: [R4, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.4 Native fixtures reuse Tiny; one SpaceBinding impl

## Description
Implement **Native fixtures (A7)** and **SpaceBinding (A10)** from the spec.

## Focused source-lane contract

Owner-authorized early parallel source lane (2026-10-25), ahead of whole fn248 completion. Parent fn252 still depends on whole fn248; this lane neither satisfies nor removes that dependency. Only admitted tasks .3/.4 may run early; other work remains unstarted.

**Touches:** [model/umpire/backends/native/tests/**, model/umpire/author/suite/src/space.rs, model/umpire/author/suite/tests/**]

Preserve distinct fixture transitions/outcomes/facts, action filters, shared starts and vacuity failures. The single binding implementation retains selected Rc allocation identity and registered World construction. Required focused checks are `cd model && mise exec -- cargo test -p umpire-checker -p umpire-suite --locked` and `cd model && mise exec -- cargo clippy -p umpire-checker -p umpire-suite --all-targets --locked -- -D warnings`, plus formatting. Existing independent controls cover ownership, construction and negative fixtures; review any changed snapshots.

The bounded wrap-up explicitly defers the complete Model pipeline, guest publication and whole-spec R9 acceptance. The R9 obligation below remains open at parent integration; focused source closure must not claim it passed. No snapshots changed in this lane.

## Acceptance
- [ ] The four fixtures reuse `tests/support`; one generic `SpaceBinding` impl.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
