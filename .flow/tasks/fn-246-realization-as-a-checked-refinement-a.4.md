---
satisfies: [R5]
---
# fn-246-realization-as-a-checked-refinement-a.4 Generic native Refinement consumer and shared activity declaration

## Description

Implement R5's generic Refinement fixture consumer and connect the activity declaration to .3's shared semantic projection after the independent protocol and migrated binding land. The consumer remains reusable by non-Temporal fixtures.

**Size:** M
**Files:** testing consumer/renderer/discovery, generic refinement controls, Temporal refines syntax and activity declaration/expected report.
**Touches:** [model/umpire/dev/testing/src/**, model/umpire/dev/testing/tests/**, model/umpire/dev/fixtures/**, model/temporal/annotate/syntax/src/**, model/temporal/annotate/tests/**, model/temporal/feature/activity/{model.rs,realization.rs,expected/**}, model/temporal/testing/fixtures.rs]

## Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors, golden hashes and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Add the typed concrete-to-abstract relation alongside existing CHECKER/SUITE consumers at `runner.rs`, selecting it through expected/refinement.txt with deterministic discovery/rendering. Reuse existing Machine exploration, action catalogs, outcomes and typed replay; no Temporal special case or second step evaluator.
2. Require an explicit mapping classification of mapped action/outcome/facts, declared full-state stutter or unmapped gap. Check every concrete start/edge and exact typed emitted Facts, including ExpiredDeadline, in the admitted World. Mapped disabled/wrong-outcome/wrong-fact and changed-state/visible-emission stutters violate; unmapped edges are coverage-incomplete. Permit extra feature edges. Do not weaken facts to phase/outcome equality or add Clone bounds to arbitrary emissions.
3. Add Temporal-owned refines admission through existing syntax. Offline use .3's total full-state map, including all Remaining budgets; online use its validated observable target and candidate-state compatibility with the shared neutral view. Check view/compatibility commutation against the full-state map; a snapshot never constructs a full state. Retain exact every-field World/state accounting and independently authored timer laws. Add hidden-budget/countdown/reset/spanning/priority/expiry-fact/default-erasure controls even where public views match. Keep source types distinct, avoid duplicate semantic tables and reconcile mapping counts without changing feature semantics or live expiry-cause admission.
4. Extend generic fixtures with independently specified positive and mutant graphs. Preserve the retained activity compatibility checkpoint and original artifacts as a separate consumer. Print comparable World/depth/completeness, counts and exact offending starts/edges without private values.

## Investigation targets

**Required:**

- `model/umpire/dev/testing/src/runner.rs:7,28,91`, `discovery.rs` and `render.rs` for consumer ownership.
- `model/umpire/backends/native` and `model/umpire/protocol/runtime/src/runtime.rs:581-616` for exploration/catalog/checked replay patterns.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:36-93` and original manifest for retained independent evidence.
- `model/temporal/annotate/syntax/src/realize.rs` and generated dispatch owner for declaration admission.
- .3's actual `model/temporal/feature/activity/realization.rs` projection owner and protocol task's shared state types.
- `model/umpire/dev/fixtures` current positive/negative fixture conventions.

## Verification

From model/: `mise exec -- cargo test -p umpire-testing -p umpire-fixtures -p umpire-temporal-annotate-syntax -p umpire-temporal-tests`; run required protocol TLC own-invariant fixtures separately to avoid claiming TLC refinement. Scoped fmt/strict clippy, exact original checkpoint and shared-projection controls are required.

## Acceptance

- [ ] Generic fixture consumer reports all concrete starts/edges as mapped, stutter, gap or violation in deterministic comparable scopes and preserves extra feature-edge permissiveness.
- [ ] Changed protocol edge, removed feature edge, wrong state/action projection, wrong start/outcome, mapped-disabled edge, changed-state/visible-outcome stutter and incomplete graph controls fail. Explicit unmapped gap is coverage-incomplete and never a pass.
- [ ] Activity full-state refinement and online observable-target compatibility share and commute through the neutral visible contract. Every World/state field and dynamic budget remains accounted; erased/defaulted budgets, wrong timer/reset/spanning/priority/fact and visible-equal/full-state-changing stutters fail. Missing/duplicate/incompatible declarations fail without changing Models or fabricating snapshot state.
- [ ] Independent original checkpoint/manifest remains byte/hash exact; native relation report and protocol TLC own-invariant report stay distinct. Focused generic/Temporal fixture, syntax, fmt and strict clippy evidence passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
