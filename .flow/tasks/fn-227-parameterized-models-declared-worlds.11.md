---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.11 Independent phase, rejection and closure-law source admission

## Description
**From fn-238.4's review:** `front/src/body/progress.rs:132` treats any macro path ending in `path` as Umpire's `path!`; use `call.mac.path.is_ident("path")` with a test.

**Scheduling (owner, 2026-10-21):** run after fn-238's shared front end migration, and build this construct once in the shared front end rather than separately in the macros and the lifter.

Deliver the framework source-admission frontier separated from .3, leaving the original .3 ID for the retry/refinement checkpoint. This is phase/rejection checkpoint P, allocated as fn-227-parameterized-models-declared-worlds.11. Admit the actual phase/reject/closure-law source forms required by the current activity Model and a generic control, through actual TLC replay. Change no activity lifecycle behavior or native golden. Owner amendments at `bbf0835fbf` preserve all laws; source omission of a law is failure.

**Size:** M candidate. One bounded source-admission frontier reuses existing typed branches, outcomes, helper SSA and replay. Phase access, rejecting branch outcomes and closure laws must be admitted together to certify the existing Model's complete declared meaning. The combined original .3 is L because this frontier plus retry behavior, realization adaptation and a new refinement harness are separate substantial changes.
**Dependencies:** After fn-227-parameterized-models-declared-worlds.8 (I), which consumes native .2 and source .7 and owns independent input lowering/exact-payload fairness. A .7-only dependency would suffice only for a reduced unit-action control; that would leave the actual retry consumer's input/rejected-edge identity unproved. Existing .3 consumes this checkpoint and retains its direct .2 edge.
**Files:** `model/umpire/lift/src/{shape,tree,expr,emit,names,build}.rs` as required by actual admitted shapes; `model/umpire/lift/src/body/{mod,expr,effects,claim,progress}.rs`; `model/umpire/lift/tests/source.rs` and source refusal controls; new generic shared fixture `model/umpire/tests/fixtures/phased_rejection/{model.rs,tests.rs,expected/**}`; replay controls in `model/umpire/testing/src/backend.rs`; only existing activity source-admission expected artifacts if the task installs the first actual TLC artifacts for its unchanged source; `model/README.md`.
**Touches:** [model/umpire/lift/**, model/umpire/tests/fixtures/phased_rejection/**, model/umpire/testing/src/backend.rs, model/temporal/feature/activity/expected/**, model/README.md]

### Approach

1. Baseline the actual unchanged activity source refusals and its native report/rows/states/coverage. Inventory its complete declaration/claim order, including imported closure helpers. Admit canonical Phased implementation and phase reads against source-owned enum/state identity; reject unrelated trait bodies or unsupported forms at location.
2. Lower complete reject! invocations and supported rejecting control flow into existing typed Outcome metadata and branches. Match identifier/bang/delimited invocation, not textual prefixes that confuse `!=`. Retain rejected self-loops, accepted stay edges, unchanged disabled actions, first rejection and reject-after-mutation semantics. Never convert rejection into a dropped branch.
3. Admit the concrete closure-law/helper source forms the activity uses, retaining their authored semantics and typed arguments in source lowering. Reuse existing helper/claim lowering and source library declarations as the source of law meaning. Do not hard-code a passing predicate by helper name, delete a law or use native verdicts/rows to construct it. Arbitrary Rust/generic evaluation remains refused.
4. Add a generic phased/rejecting fixture with payload actions and deliberately broken terminal transitions. Independently emit and execute TLC, then replay exact action payloads and outcomes through the native Model. The unchanged activity must stop producing an admission refusal and retain both registered closure laws in emitted/checkable metadata.

Consume I's exact-action fairness handoff when payloads reject. Existing native progress uses concrete action equality and existing TLC emission adds WF per action operator. Verify that fairness for a current-attempt Poll cannot be discharged by a different stale Poll's rejecting self-loop. I owns the finite per-value source/TLC repair and its standalone control; P adds rejecting-edge consumption coverage and claims no independent implementation of that repair or new fairness framework.

### Investigation targets

Required, reanchor existing files before coding:

- `model/temporal/feature/activity/model.rs` for the exact existing Phased/reject/helper/law declaration forms, read-only in this task.
- `model/umpire/lift/src/shape.rs` for trait/helper/source admission.
- `model/umpire/lift/src/body/effects.rs` for branch priority, SSA and supported helpers.
- `model/umpire/lift/src/body/claim.rs` and `model/umpire/lift/src/emit.rs` for source law semantics and generated checks.
- `model/umpire/testing/src/backend.rs` for action/outcome identity and rejected-edge replay.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-lift -p umpire-testing -p umpire-tests -p umpire-temporal-tests`; repeat admitted fixtures with `UMPIRE_BACKENDS=require`; scoped fmt and strict clippy. Retain the initial located refusal, complete law inventory and mutation-negative results.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Refresh changed guest bytes through the normal builder and rerun embedded slice/gaps if required. Native activity goldens and all unrelated expected bytes/metadata remain exact. Any previously retained located refusal replaced by genuine admission must be specifically identified, with its original retained as evidence and new TLC artifacts reviewed; this is not permission for global snapshot updates.
## Acceptance

- [ ] The exact existing activity source and generic control admit canonical phase/rejection forms; unsupported traits/helpers/control flow refuse by location.
- [ ] Typed accepted/rejected outcome identity, concrete action inputs, rejected self-loops, disabled arms and stay semantics survive independent emission and replay. Source-only outcome/action mutations fail comparison.
- [ ] Both terminal/closure laws remain in source order and checkable metadata. Reopening, accepted closed-step and wrong-rejection controls refute their actual predicates; no law-name shortcut or native-graph-derived meaning exists.
- [ ] Required TLC executes both the generic control and the unchanged activity; false-law counterexamples replay, including rejected edges and payload identity.
- [ ] Native activity behavior and unrelated golden/expansion bytes remain exact. Specifically reviewed admission-artifact changes, full gates, counts and timings are retained.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
