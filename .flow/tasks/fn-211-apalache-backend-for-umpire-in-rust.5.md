---
satisfies: [R6]
---
# fn-211-apalache-backend-for-umpire-in-rust.5 After .4: symbolic World constants through ConstInit with native replay under the decoded World

## Description
Owner request 2026-10-21 ("well want that"): Apalache checks every World in a declared finite domain in one run per property instead of one selected World. Depends on the completed fn-211.4 fixture consumer (and transitively .3 typed ITF decoding/replay), fn-227's declared Worlds/value handoff, and fn-250's shared Literal/Schema/replay core. This task remains deferred with its parent: execute only after explicit reactivation and the recorded R6 ownership decision with fn-240; do not implement the same symbolic-World feature in both specs by default.

**Size:** M.

### Approach

1. Extend .4's existing logical `apalache.txt` fixture selection/ownership with an explicit symbolic-mode option; settle its spelling before implementation without adding Model annotations or an unowned sidecar. Require every World field to have an admitted finite domain from its type. The current parameterized_world/action_inputs Worlds have bare integer fields; add bounded fixture variants using the existing `Finite` wrapper/enum surface and preserve domain-less refusal controls. Direct `#[inputs]` on World fields is not admitted today and is outside this task. Compute domain cardinality with checked arithmetic before enumeration and use the current catalog limit for native all-World confirmation.
2. Emit World fields as `CONSTANT`s with a typed `ConstInit` predicate over their domains; retain concrete-World binding semantics for TLC; its representation may change with coordinated reader/golden updates.
3. Init over symbolic constants: choose and record source-lifted `#[starts]` bodies or a complete per-World start table from derived values before implementation. A table is allowed only when the full domain can be enumerated within the admitted limit; otherwise use an admitted symbolic source translation or refuse explicitly. Preserve correlated/empty starts for each World. Apply the same admission rule to World-dependent action domains: emitted guards and concrete action identities must describe that World's checked catalog, including varying payload domains/ordinals, rather than reusing the selected World's catalog. Source meaning stays independent of explored native edges.
4. Decode the constant values from the ITF trace through the existing decoder (its serde-derive envelope is fn-248's change; do not rework envelope or value decoding here), construct the World natively through a typed generated inverse of the derived value export, obtain its complete starts and checked catalog once, and reuse fn-250's shared replay core through the Apalache adapter. Validate exact fields/types/domains before construction; replay every concrete action/choice through `checked_step` under that World with its retained catalog. Out-of-domain or unconstructible Worlds, crossed starts/catalogs and actions admitted only under another World are witness-rejected; rendering never supplies identity.
5. `apalache.txt` records the exact World domain, transition bound, native-confirmation scope and, for violations, the decoded World. When the domain product is within the catalog limit, complete native exploration of every World supplies a common maximum native-depth-plus-one bound. Beyond it, use the selected Worlds' completed native-derived bound, compare only those Worlds and label the domain-wide answer bounded symbolic without native agreement. Refuse cut exploration, bound/cardinality overflow and catalog admission failure. Confirm NoViolation at the same reported bound; no selected-World result certifies unchecked Worlds.
6. Controls based on bounded variants of `parameterized_world` and `action_inputs`: a property violated only in a non-selected World is found and replayed with its correlated starts and exact payload catalog; domain-less fields refuse; forged out-of-domain/unconstructible Worlds and crossed-World starts/actions refuse. Exercise domain products at/beyond the limit, cardinality overflow, incomplete native exploration, and a larger-domain result that remains explicitly bounded symbolic. Retain .4's skip/ownership, update-failure and actual pinned-tool controls.

### Verification

Quick: the Apalache crate and fixture tests from `model/umpire` under `UMPIRE_APALACHE=require` (add `UMPIRE_BACKENDS=require` for TLC agreement controls). Full after review: one `UMPIRE_APALACHE=require make model-validate` from the repository root, retaining the existing stages and benchmark. Check default/TLC-only skips without launching Apalache. Verify TLC, native and concrete-World Apalache semantic agreement and typed replay; review and regenerate affected goldens rather than pinning previous bytes.

## Acceptance
- [ ] Opted-in fixtures emit World constants with ConstInit over admitted finite type domains; one bounded run per property covers the domain, with explicit bound/native-confirmation scope and checked domain/catalog admission.
- [ ] A violation only reachable in a non-selected World is found, decoded to a concrete World and replayed natively using its complete correlated starts and exact World-dependent action catalog; crossed-World witnesses refuse.
- [ ] Domain-less World fields refuse symbolic mode naming the field; out-of-domain or unconstructible counterexample Worlds are witness-rejected.
- [ ] NoViolation agreement is claimed natively only where native checked it (selected Worlds, or all Worlds within the catalog limit), otherwise reported as bounded symbolic.
- [ ] The Init/domain-encoding choice is recorded and bounded table enumeration cannot silently narrow the World domain; TLC and concrete-World semantics agree with native checking, with affected goldens reviewed and regenerated. Default/TLC-only skips and actual dedicated-opt-in tool controls pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
