# fn-237 World-selected fairness planning findings

The activity consumer needs an independently bound exact fair-action list for each selected World. Add bounded source support inside fn-237 after O integrates and before activity .3. Preserve closed I and every cancellation-policy requirement. This report recommends two serial M tasks under the seven-task ceiling, with a typed admitted-plan handoff. One cohesive task remains a possible execution choice only if fresh review explicitly accepts its larger surface.

## Evidence and existing ownership

The starting canonical snapshot is `.flow/tmp/fn237-fairness-plan-{input.md,show.json}`. It has five tasks. F (.1) is done, O (.2) is in progress, activity .3 consumes F/O, live .4 depends natively on .3, and final documentation .5 depends natively on .4 while blocked for committed fn-243.5. The final section of `.flow/tmp/fn2373-execution-prep.md` supplies the authoritative source gap and proposed restricted grammar.

`flowctl show/cat fn-227.8` confirms I is done. Its receipt names implementation commits `3fd1d2f551` and `0469334f91`, exact typed catalog ordinals and payload replay, and one weak-fairness obligation for every concrete fair action. Its actual native/TLC controls include stale payloads and fair-specialization mutations. Its bounded-progress fairness control is pinned structurally and is expressly recorded as behaviorally nondiscriminating. Preserve that limitation. The new work concerns World-dependent selection of existing exact lists. It neither repairs nor reopens I.

The current `backends/tla/src/bind.rs:20` validates World values before catalog binding, but `bind_catalog` at line 63 invokes `scoped_catalog` at line 88 before an unresolved conditional plan could become a selected flat assumption. The new binding owner must resolve the independently typed World predicate before those existing membership checks. Native catalog admission retains its current exact membership rule.

## Sizing decision

| Choice | Resulting count | Assessment |
| --- | --- | --- |
| One cohesive prerequisite | Six tasks | Strong end-to-end cohesion, but front grammar, typed conditional IR, embedding, binding order, consumer handling, actual TLC and mutation proof exceed the plan skill's normal M indicators of 3-5 files and 3-5 acceptance criteria. Call this a larger task unless fresh review grounds an M exception in reuse. |
| Two serial prerequisites | Seven tasks | Recommended. Close source admission and typed transport first, then selected-World backend resolution and cross-checker proof. Freeze the typed plan interface at the first closure. Both tasks adapt the existing parser, expressions, builder leaves and fair emission. |

A parser task and a separate test task would leave an arbitrary boundary. The proposed first task delivers source admission, stable typed identity, macro transport and located refusals as its complete capability. The second consumes that capability to deliver usable selected-World backend semantics and actual native/TLC agreement. The first may refuse backend execution for unresolved plans until the second lands. Such refusal must be explicit and tested; no temporary union or default branch is permitted. If the implementation cannot expose that fail-closed handoff without broad incidental machinery, prefer one explicitly reviewed cohesive task instead of manufacturing a split.

### Proposed source-admission task, M

Own the exhaustive World-only assumption tree, registration/helper identity and opt-in embedded representation. Primary files are `model/umpire/front/src/body/progress.rs`, the existing typed expression/World environment where required, `model/umpire/ir/src/tree.rs`, and the DSL macro lifted transport. Keep native World-argument plumbing unchanged unless mechanical transport requires adjustment. Include focused front/embedding tests and any minimal exhaustive-match/refusal wiring needed to preserve compilation.

Acceptance clusters are:

1. Every branch independently admits the existing literal `fair`/`only`/`without` leaves and existing builder chains. Tail if/else and nested else-if require a final else. An inactive branch's well-typed Poll(3) is valid before a one-attempt World is selected.
2. Guards admit primitive World bools or primitive scalar comparisons with typed literals/source constants, parentheses and the existing boolean connectives. State, before/after, outcome and action-input identities, arbitrary calls/arithmetic, loops, match and computed action payloads produce located refusals. Helper recursion and identity/shadowing controls remain enforced.
3. Registered assumptions, named helpers and progress scopes carry the same typed source-derived plan, locations and names through macro embedding and independent re-admission. No compiled native callback selects a source branch.
4. Existing flat-source behavior and required artifact bytes remain exact. Newly admitted unresolved plans refuse backend execution until the resolver exists. Review Print placement before adding always-printed fields that could change every expansion.

### Proposed selected-binding and proof task, M

Consume the committed typed-plan interface. Own the narrow World predicate resolver in TLA binding, its ordering before selected catalog validation, resolved-plan consumer checks, and one generic fixture using the existing testing/discovery route. Audit the interpreted-native bridge for mechanical transport only. Reuse exact WF emission and metadata.

Acceptance clusters are:

1. Evaluate only the admitted typed guard subset against validated selected `binding.world`, select exactly one leaf, then apply unchanged selected-catalog checks. Resolve every registered/progress assumption consistently. A selected unavailable Poll refuses, and unselected leaves never become catalog filters or a union.
2. Bind maximum 1, 2 and 3 of one fixture type to exact lists containing respectively 2, 3 and 4 values, comprising Poll(1)..Poll(m) and close. Preserve all ordinary allowed actions, exact typed payloads/ordinals, one WF per selected concrete Poll and close, and binding consistency across all five CFG families.
3. Actual native/TLC runs and replay distinguish stale-payload starvation, missing-close heartbeat loops, omission of the last selected Poll, fair-payload substitution and union WF. Compare selected flat assumptions and WF metadata directly. Mutate source-only predicates/leaves while native code stays fixed, legal selected World bindings and out-of-catalog leaves. Each mutation must produce the appropriate independent disagreement or typed refusal.
4. Preserve existing flat fixtures/artifacts and native scope-stability checks. Retain actual counts, TLC/replay evidence, required gates and timing. Bounded claims keep the established no-fairness structural check without claiming a new behavioral discriminator.

## Sequencing and task admission

Use provisional labels A and B until the CLI allocates IDs. Do not renumber existing .3/.4/.5. With two new tasks the native dependency shape is `.2 -> A -> B -> .3 -> .4 -> .5`; retain .3's existing F/O dependencies. A consumes O because the overlapping IR/front/macros/binding source must start from O's committed closure and integrated tip. B consumes A. .3 consumes B and all pre-existing prerequisites. A and B are strictly serial, including source, generated code and fixture ownership.

With one prerequisite the shape is `.2 -> support -> .3 -> .4 -> .5`. The same committed-O integration and fresh review gates apply. No framework work starts inside active O, and .3 gains no new source construct of its own.

Retain the manual cross-spec chain `fn-237.4 -> fn-243.5 -> fn-237.5`. The conductor admits fn-243.5 only after fn-237.4 is done and both its implementation and Flow closure commits are ancestors of `umpire`. Native fn-243.4 remains its existing prerequisite. The conductor clears fn-237.5's block only after fn-243.5 is done with committed implementation/closure in `umpire`; native fn-237.4 remains independently required. Avoid whole-spec dependency edges between fn-237 and fn-243, which would deadlock the producer of the cooperative contract.

Read-only show calls confirm fn-228 and fn-243 remain open. fn-228 R7 and fn-243 R2/R5 retain their final unheld-cancellation/R40 obligations through fn-243.5. Done architecture checkpoints do not close those parents. Queue order and historical support receipt references add no new whole-spec edges.

## Coverage and canonical amendment

Both proposed support tasks satisfy existing R2. Retain R1-R5 identities and meanings. The support work makes R2's already required selected-World exact-action fairness expressible; it adds no scheduling policy and changes no R2 cancellation cell. R1 stays with F/O, R3 with activity .3, R4 with .4, and R5 with .5. Add the allocated support task(s) to R2's coverage while preserving the existing F/O/.3 entries.

When O has cleanly closed and integrated, the root can use CLI-only task creation, dependency updates and spec/task text updates. The amendment should explain the bounded World-only source form in R2's implementation contract, identify the closed-I/new-support ownership boundary in Resolved via Codebase, add the new prerequisite(s) to .3's dependencies, size and support-consumer description, and replace .3's stale instruction to return finite per-value repairs to I's owner with consumption of the new World-selection seam. Preserve the instruction to report a missing exact-WF regression through its owning support boundary. Update .5's support receipt inventory after allocation.

Generate final coverage with `flowctl validate --spec <canonical-id> --coverage --json` and persist the generated table through the normal CLI text update. Fresh plan review must pass after canonical changes and before any new support implementation. The previous SHIP applies to the starting artifact and cannot authorize this amended dependency/source surface.

## Scope of this scout

This report is the sole write. All canonical Flow operations were read-only show/cat calls using `python3 /home/agent/.codex/scripts/flowctl.py` after initial help, with an explicit `cd` to the Temporal repository. No task creation, state mutation, source edit, test, build, or additional agent ran.

## Sizing reconciliation after the repo scout

The authoritative repo scout established a smaller implementation surface than this scout initially assumed. Four production files require changes: `front/src/body/progress.rs`, `ir/src/tree.rs`, `backends/tla/src/bind.rs` and `backends/native/src/interpret.rs`. `backends/tla/src/emit.rs` may need explicit unresolved-plan refusal hardening. Existing Print embedding and native World plumbing carry the representation. No macro, DSL, Lifted/public API extension is required. This addendum accepts those findings without another source investigation.

That evidence supersedes the two-M recommendation above. One cohesive M prerequisite now matches the plan sizing rule's four or five production files and four acceptance clusters, with six total fn-237 tasks. Its independent outcome is an authored conditional plan that binds to the exact selected list before catalog validation and agrees with native execution and TLC. Splitting parser and resolver would add an intermediate unresolved-plan contract and serial handoff without an independently needed consumer capability.

Use four acceptance clusters while preserving every previously listed positive and negative case: (1) exhaustive World-only grammar, helper/identity handling and all located refusals; (2) typed selected-World branch resolution before unchanged membership checks, shared registration/progress identity and all five CFG families; (3) actual native/TLC agreement, exact list/ordinal/WF metadata, replay and every source/payload/World/union/starvation mutation; (4) unchanged flat artifacts, embedding parity, documentation, required gates, counts and timings. Grouping these cases changes their presentation only; it removes no control or refusal boundary.

The selected native chain becomes `.2 -> support -> .3 -> .4 -> .5`, retaining .3's existing F/O prerequisites and the manual `.4 -> fn-243.5 -> .5` cross-spec gates. Support starts from O's committed integrated closure, satisfies existing R2, and receives fresh plan review before implementation. The gap analyst should assess this concrete cohesive seam. The initial broader ownership inventory and its resulting two-task preference are historical reasoning, superseded by the repo scout's actual minimal surface. No Model policy, canonical task or source file changed in this reconciliation.
