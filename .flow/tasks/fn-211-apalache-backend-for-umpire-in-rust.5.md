---
satisfies: [R6]
---
# fn-211-apalache-backend-for-umpire-in-rust.5 After .3: symbolic World constants through ConstInit with native replay under the decoded World

## Description
Owner request 2026-10-21 ("well want that"): Apalache checks every World in a declared finite domain in one run per property instead of one selected World. Depends on fn-211.3 (typed ITF decoding and replay) and on fn-227 (declared Worlds, `Finite`/`#[inputs]` domains, fn-227.14's derived value handoff).

**Size:** M.

### Approach

1. Opt-in per fixture (e.g. a `symbolic` marker beside `expected/apalache.txt`; settle the spelling in planning without adding Model annotations). Require every World field to have a declared finite domain via the existing one-range spelling; otherwise refuse symbolic mode naming the field.
2. Emit World fields as `CONSTANT`s with a typed `ConstInit` predicate over their domains; retain concrete-World binding semantics for TLC; its representation may change with coordinated reader/golden updates.
3. Init over symbolic constants: decide between lifting the `#[starts]` body from source and a per-World start table built natively from derived values; prefer the source route if the lifter's expression subset covers the fixtures, since meaning must stay independent. Record the choice.
4. Decode the constant values from the ITF trace through the existing decoder (its serde-derive envelope is fn-248's change; do not rework envelope or value decoding here), construct the World natively (inverse of the derived value export), and replay through `checked_step`; out-of-domain or unconstructible Worlds are witness-rejected.
5. `apalache.txt` records the World domain and, for violations, the decoded World. NoViolation is confirmed natively on the fixture's selected Worlds, and on every World when the domain product is within the catalog limit; otherwise it is reported as bounded symbolic only.
6. Controls on `parameterized_world` and `action_inputs`: a property violated only in a non-selected World is found and replayed; a domain-less World field refuses; a forged out-of-domain trace is rejected.

### Verification

Quick: the Apalache crate and fixture tests under `UMPIRE_BACKENDS=require`. Full after review: `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Verify TLC, native and concrete-World Apalache semantic agreement and typed replay; review and regenerate affected goldens rather than pinning previous bytes.

## Acceptance
- [ ] Opted-in fixtures emit World constants with ConstInit over declared domains; one bounded run per property covers the domain.
- [ ] A violation only reachable in a non-selected World is found, decoded to a concrete World and replayed natively.
- [ ] Domain-less World fields refuse symbolic mode naming the field; out-of-domain or unconstructible counterexample Worlds are witness-rejected.
- [ ] NoViolation agreement is claimed natively only where native checked it (selected Worlds, or all Worlds within the catalog limit), otherwise reported as bounded symbolic.
- [ ] The Init-over-symbolic-Worlds choice is recorded; TLC and concrete-World semantics agree with native checking, with affected goldens reviewed and regenerated.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
